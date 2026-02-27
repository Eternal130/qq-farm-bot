package bot

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	"qq-farm-bot/proto/gatepb"
	"qq-farm-bot/proto/itempb"
	"qq-farm-bot/proto/plantpb"
	"qq-farm-bot/proto/userpb"
)

const (
	// writeWait is the deadline for write operations.
	writeWait = 10 * time.Second
	// pongWait is how long to wait for a pong before considering the connection dead.
	pongWait = 60 * time.Second
	// pingPeriod is the interval for sending WebSocket ping frames. Must be < pongWait.
	pingPeriod = 25 * time.Second
	// maxHeartbeatFailures is consecutive heartbeat failures before forced disconnect.
	maxHeartbeatFailures = 3
)

type pendingCall struct {
	ch    chan *callResult
	timer *time.Timer
}

type callResult struct {
	body []byte
	meta *gatepb.Meta
	err  error
}

// Network manages the WebSocket connection to the game server.
type Network struct {
	conn      *websocket.Conn
	writeMu   sync.Mutex // protects concurrent writes to conn
	clientSeq int64
	serverSeq int64

	pending   map[int64]*pendingCall
	pendingMu sync.Mutex

	state    *UserState
	logger   *Logger
	onNotify func(msgType string, body []byte)

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

type UserState struct {
	mu    sync.RWMutex
	GID   int64
	Name  string
	Level int64
	Exp   int64
	Gold  int64
}

func (s *UserState) Get() (gid, level, exp, gold int64, name string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.GID, s.Level, s.Exp, s.Gold, s.Name
}

func (s *UserState) SetFromLogin(gid, level, exp, gold int64, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GID = gid
	s.Name = name
	s.Level = level
	s.Exp = exp
	s.Gold = gold
}

func NewNetwork(logger *Logger) *Network {
	ctx, cancel := context.WithCancel(context.Background())
	return &Network{
		pending: make(map[int64]*pendingCall),
		state:   &UserState{},
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}
}

// writeMessage sends a WebSocket message with write mutex and deadline.
func (n *Network) writeMessage(messageType int, data []byte) error {
	n.writeMu.Lock()
	defer n.writeMu.Unlock()
	n.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return n.conn.WriteMessage(messageType, data)
}

// Connect establishes WebSocket connection.
func (n *Network) Connect(serverURL, platform, clientVersion, code string) error {
	url := fmt.Sprintf("%s?platform=%s&os=iOS&ver=%s&code=%s&openID=", serverURL, platform, clientVersion, code)
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	headers := map[string][]string{
		"User-Agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows WindowsWechat/WMPF WindowsWechat(0x63090a13)"},
		"Origin":     {"https://gate-obt.nqf.qq.com"},
	}
	conn, _, err := dialer.Dial(url, headers)
	if err != nil {
		return fmt.Errorf("ws dial: %w", err)
	}
	n.conn = conn

	// Set up WebSocket-level keepalive: ReadDeadline + PongHandler
	n.conn.SetReadDeadline(time.Now().Add(pongWait))
	n.conn.SetPongHandler(func(string) error {
		n.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Start read loop and ping loop
	go n.readLoop()
	go n.pingLoop()

	return nil
}

// pingLoop sends WebSocket ping frames periodically to detect dead connections.
func (n *Network) pingLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			if err := n.writeMessage(websocket.PingMessage, nil); err != nil {
				if n.ctx.Err() == nil {
					n.logger.Warnf("WS", "Ping 失败: %v", err)
				}
				n.cancel()
				return
			}
		}
	}
}

func (n *Network) Close() {
	n.cancel()
	if n.conn != nil {
		// Send close frame gracefully before closing
		n.writeMu.Lock()
		n.conn.SetWriteDeadline(time.Now().Add(writeWait))
		n.conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		n.writeMu.Unlock()
		n.conn.Close()
	}
	// Cancel all pending calls
	n.pendingMu.Lock()
	for seq, p := range n.pending {
		p.timer.Stop()
		p.ch <- &callResult{err: fmt.Errorf("connection closed")}
		delete(n.pending, seq)
	}
	n.pendingMu.Unlock()
}

func (n *Network) Done() <-chan struct{} { return n.ctx.Done() }
func (n *Network) State() *UserState     { return n.state }

// SendRequest sends a protobuf request and waits for response.
func (n *Network) SendRequest(service, method string, body []byte) ([]byte, error) {
	seq := atomic.AddInt64(&n.clientSeq, 1)
	msg := &gatepb.Message{
		Meta: &gatepb.Meta{
			ServiceName: service,
			MethodName:  method,
			MessageType: 1, // Request
			ClientSeq:   seq,
			ServerSeq:   atomic.LoadInt64(&n.serverSeq),
		},
		Body: body,
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	ch := make(chan *callResult, 1)
	timer := time.AfterFunc(10*time.Second, func() {
		n.pendingMu.Lock()
		if p, ok := n.pending[seq]; ok {
			delete(n.pending, seq)
			p.ch <- &callResult{err: fmt.Errorf("timeout: %s.%s", service, method)}
		}
		n.pendingMu.Unlock()
	})

	n.pendingMu.Lock()
	n.pending[seq] = &pendingCall{ch: ch, timer: timer}
	n.pendingMu.Unlock()

	if err := n.writeMessage(websocket.BinaryMessage, data); err != nil {
		n.pendingMu.Lock()
		delete(n.pending, seq)
		n.pendingMu.Unlock()
		timer.Stop()
		return nil, fmt.Errorf("write: %w", err)
	}

	result := <-ch
	if result.err != nil {
		return nil, result.err
	}
	if result.meta != nil && result.meta.ErrorCode != 0 {
		return nil, fmt.Errorf("%s.%s error: code=%d %s", service, method, result.meta.ErrorCode, result.meta.ErrorMessage)
	}
	return result.body, nil
}

func (n *Network) readLoop() {
	defer n.cancel()
	for {
		_, data, err := n.conn.ReadMessage()
		if err != nil {
			if n.ctx.Err() == nil {
				n.logger.Warnf("WS", "读取失败: %v", err)
			}
			return
		}
		n.handleMessage(data)
	}
}

func (n *Network) handleMessage(data []byte) {
	msg := &gatepb.Message{}
	if err := proto.Unmarshal(data, msg); err != nil {
		return
	}
	meta := msg.Meta
	if meta == nil {
		return
	}

	if meta.ServerSeq > 0 {
		for {
			old := atomic.LoadInt64(&n.serverSeq)
			if meta.ServerSeq <= old || atomic.CompareAndSwapInt64(&n.serverSeq, old, meta.ServerSeq) {
				break
			}
		}
	}

	switch meta.MessageType {
	case 2: // Response
		n.pendingMu.Lock()
		p, ok := n.pending[meta.ClientSeq]
		if ok {
			delete(n.pending, meta.ClientSeq)
			p.timer.Stop()
			p.ch <- &callResult{body: msg.Body, meta: meta}
		}
		n.pendingMu.Unlock()

	case 3: // Notify
		n.handleNotify(msg)
	}
}

func (n *Network) handleNotify(msg *gatepb.Message) {
	if len(msg.Body) == 0 {
		return
	}
	event := &gatepb.EventMessage{}
	if err := proto.Unmarshal(msg.Body, event); err != nil {
		return
	}
	msgType := event.MessageType

	// Handle known notify types inline
	if strings.Contains(msgType, "Kickout") {
		kick := &gatepb.KickoutNotify{}
		if err := proto.Unmarshal(event.Body, kick); err == nil {
			n.logger.Warnf("推送", "被踢下线: %s", kick.ReasonMessage)
		}
		n.cancel()
		return
	}

	if strings.Contains(msgType, "BasicNotify") {
		notify := &userpb.BasicNotify{}
		if err := proto.Unmarshal(event.Body, notify); err == nil && notify.Basic != nil {
			n.state.mu.Lock()
			oldLevel := n.state.Level
			if notify.Basic.Level > 0 {
				n.state.Level = notify.Basic.Level
			}
			if notify.Basic.Gold > 0 {
				n.state.Gold = notify.Basic.Gold
			}
			if notify.Basic.Exp > 0 {
				n.state.Exp = notify.Basic.Exp
			}
			n.state.mu.Unlock()
			if n.state.Level != oldLevel {
				n.logger.Infof("系统", "升级! Lv%d → Lv%d", oldLevel, n.state.Level)
			}
		}
		return
	}

	if strings.Contains(msgType, "ItemNotify") {
		notify := &itempb.ItemNotify{}
		if err := proto.Unmarshal(event.Body, notify); err == nil {
			for _, chg := range notify.Items {
				if chg.Item == nil {
					continue
				}
				id := chg.Item.Id
				count := chg.Item.Count
				if id == 1101 || id == 2 {
					n.state.mu.Lock()
					n.state.Exp = count
					n.state.mu.Unlock()
				} else if id == 1 || id == 1001 {
					n.state.mu.Lock()
					n.state.Gold = count
					n.state.mu.Unlock()
				}
			}
		}
		return
	}

	// Forward other notifies to bot
	if n.onNotify != nil {
		n.onNotify(msgType, event.Body)
	}
}

// Login sends login request to the game server.
func (n *Network) Login(clientVersion string) error {
	req := &userpb.LoginRequest{
		SharerId:     0,
		SharerOpenId: "",
		DeviceInfo: &userpb.DeviceInfo{
			ClientVersion: clientVersion,
			SysSoftware:   "iOS 26.2.1",
			Network:       "wifi",
			Memory:        7672,
			DeviceId:      "iPhone X<iPhone18,3>",
		},
		ShareCfgId: 0,
		SceneId:    "1256",
		ReportData: &userpb.ReportData{
			MinigameChannel: "other",
			MinigamePlatid:  2,
		},
	}
	body, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	replyBody, err := n.SendRequest("gamepb.userpb.UserService", "Login", body)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}
	reply := &userpb.LoginReply{}
	if err := proto.Unmarshal(replyBody, reply); err != nil {
		return fmt.Errorf("decode login reply: %w", err)
	}
	if reply.Basic == nil {
		return fmt.Errorf("login reply missing basic info")
	}
	b := reply.Basic
	n.state.SetFromLogin(b.Gid, b.Level, b.Exp, b.Gold, b.Name)

	n.logger.Infof("登录", "成功 GID=%d 昵称=%s Lv%d 金币=%d", b.Gid, b.Name, b.Level, b.Gold)
	return nil
}

// StartHeartbeat runs heartbeat loop until context is cancelled.
// Tracks consecutive failures and disconnects after maxHeartbeatFailures.
func (n *Network) StartHeartbeat(clientVersion string, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		consecutiveFailures := 0
		for {
			select {
			case <-n.ctx.Done():
				return
			case <-ticker.C:
				gid, _, _, _, _ := n.state.Get()
				if gid == 0 {
					continue
				}
				req := &userpb.HeartbeatRequest{Gid: gid, ClientVersion: clientVersion}
				body, _ := proto.Marshal(req)
				if _, err := n.SendRequest("gamepb.userpb.UserService", "Heartbeat", body); err != nil {
					consecutiveFailures++
					n.logger.Warnf("心跳", "失败 (%d/%d): %v", consecutiveFailures, maxHeartbeatFailures, err)
					if consecutiveFailures >= maxHeartbeatFailures {
						n.logger.Warnf("心跳", "连续失败 %d 次，断开连接", maxHeartbeatFailures)
						n.cancel()
						return
					}
				} else {
					consecutiveFailures = 0
				}
			}
		}
	}()
}

// AllLands fetches all farm lands.
func (n *Network) AllLands() (*plantpb.AllLandsReply, error) {
	req := &plantpb.AllLandsRequest{}
	body, _ := proto.Marshal(req)
	replyBody, err := n.SendRequest("gamepb.plantpb.PlantService", "AllLands", body)
	if err != nil {
		return nil, err
	}
	reply := &plantpb.AllLandsReply{}
	return reply, proto.Unmarshal(replyBody, reply)
}
