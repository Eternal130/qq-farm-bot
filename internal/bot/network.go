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

// ServerError represents a business error returned by the game server.
type ServerError struct {
	Service string
	Method  string
	Code    int64
	Message string
}

func (e *ServerError) Error() string { return e.Message }

// ---------------------------------------------------------------------------
// Disconnect reason taxonomy
// ---------------------------------------------------------------------------

// DisconnectReason classifies why a connection was lost, enabling
// differentiated reconnection strategies in the watchdog.
type DisconnectReason int

const (
	// DisconnectUnknown is the zero value — no reason recorded yet.
	DisconnectUnknown DisconnectReason = iota
	// DisconnectPingFailed — WebSocket ping could not be sent.
	DisconnectPingFailed
	// DisconnectReadError — WebSocket read loop encountered an error.
	DisconnectReadError
	// DisconnectKickout — server explicitly kicked us offline.
	DisconnectKickout
	// DisconnectHeartbeatTimeout — consecutive heartbeat failures exceeded threshold.
	DisconnectHeartbeatTimeout
	// DisconnectLoginFailed — login request returned a business error.
	DisconnectLoginFailed
	// DisconnectLoginTimeout — login request timed out (30 s).
	DisconnectLoginTimeout
	// DisconnectClosed — Close() was called explicitly (user-initiated stop).
	DisconnectClosed
)

func (r DisconnectReason) String() string {
	switch r {
	case DisconnectPingFailed:
		return "ping_failed"
	case DisconnectReadError:
		return "read_error"
	case DisconnectKickout:
		return "kickout"
	case DisconnectHeartbeatTimeout:
		return "heartbeat_timeout"
	case DisconnectLoginFailed:
		return "login_failed"
	case DisconnectLoginTimeout:
		return "login_timeout"
	case DisconnectClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// Retryable reports whether the watchdog should attempt reconnection.
func (r DisconnectReason) Retryable() bool {
	switch r {
	case DisconnectKickout:
		return false // server kicked us; retrying is futile
	case DisconnectClosed:
		return false // intentional stop
	default:
		return true
	}
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	// writeWait is the deadline for write operations.
	writeWait = 10 * time.Second
	// pongWait is how long to wait for a pong before considering the connection dead.
	pongWait = 60 * time.Second
	// pingPeriod is the interval for sending WebSocket ping frames. Must be < pongWait.
	pingPeriod = 25 * time.Second
	// defaultRequestTimeout is the deadline for normal RPC requests.
	defaultRequestTimeout = 10 * time.Second
	// loginTimeout is the deadline for the Login RPC (longer than default to
	// tolerate slow initial handshakes).
	loginTimeout = 30 * time.Second
	// maxHeartbeatFailures is consecutive heartbeat failures before forced disconnect.
	maxHeartbeatFailures = 3
	// heartbeatResponseDeadline is elapsed time since last successful heartbeat
	// response before the connection is considered unhealthy.
	heartbeatResponseDeadline = 60 * time.Second
)

// ---------------------------------------------------------------------------
// Internal types
// ---------------------------------------------------------------------------

type pendingCall struct {
	ch    chan *callResult
	timer *time.Timer
}

type callResult struct {
	body []byte
	meta *gatepb.Meta
	err  error
}

// ---------------------------------------------------------------------------
// Network
// ---------------------------------------------------------------------------

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

	// Disconnect reason — written at most once via disconnectOnce.
	disconnectOnce   sync.Once
	disconnectReason DisconnectReason

	// Heartbeat health tracking: unix-millis of last successful heartbeat response.
	lastHeartbeatAt atomic.Int64

	// Server time delta (milliseconds): serverTime - localTime.
	// Approximate server now = time.Now().UnixMilli() + ServerTimeDelta().
	serverTimeDelta atomic.Int64

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
	n := &Network{
		pending: make(map[int64]*pendingCall),
		state:   &UserState{},
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}
	n.lastHeartbeatAt.Store(time.Now().UnixMilli())
	return n
}

// disconnectWithReason records the disconnect reason (first-writer-wins)
// and cancels the context to signal all goroutines.
func (n *Network) disconnectWithReason(reason DisconnectReason) {
	n.disconnectOnce.Do(func() {
		n.disconnectReason = reason
	})
	n.cancel()
}

// ---------------------------------------------------------------------------
// Low-level I/O
// ---------------------------------------------------------------------------

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
				n.disconnectWithReason(DisconnectPingFailed)
				return
			}
		}
	}
}

func (n *Network) Close() {
	n.disconnectWithReason(DisconnectClosed)
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

func (n *Network) Done() <-chan struct{}                 { return n.ctx.Done() }
func (n *Network) State() *UserState                     { return n.state }
func (n *Network) GetDisconnectReason() DisconnectReason { return n.disconnectReason }

// ServerTimeDelta returns the offset (in milliseconds) between server time and
// local time.  Approximate server now ≈ time.Now().UnixMilli() + delta.
func (n *Network) ServerTimeDelta() int64 { return n.serverTimeDelta.Load() }

// ---------------------------------------------------------------------------
// RPC layer
// ---------------------------------------------------------------------------

// sendRequestWithTimeout sends a protobuf request and waits for the response
// with a caller-specified timeout.
func (n *Network) sendRequestWithTimeout(service, method string, body []byte, timeout time.Duration) ([]byte, error) {
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
	timer := time.AfterFunc(timeout, func() {
		n.pendingMu.Lock()
		if p, ok := n.pending[seq]; ok {
			delete(n.pending, seq)
			p.ch <- &callResult{err: fmt.Errorf("timeout: %s.%s (after %v)", service, method, timeout)}
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
		return nil, &ServerError{Service: service, Method: method, Code: result.meta.ErrorCode, Message: result.meta.ErrorMessage}
	}
	return result.body, nil
}

// SendRequest sends a protobuf request with the default 10 s timeout.
func (n *Network) SendRequest(service, method string, body []byte) ([]byte, error) {
	return n.sendRequestWithTimeout(service, method, body, defaultRequestTimeout)
}

// ---------------------------------------------------------------------------
// Read loop & message dispatch
// ---------------------------------------------------------------------------

func (n *Network) readLoop() {
	defer func() {
		// First-writer-wins: if Kickout / heartbeat already set a reason,
		// this will be a no-op thanks to disconnectOnce.
		n.disconnectWithReason(DisconnectReadError)
	}()
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
		n.disconnectWithReason(DisconnectKickout)
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

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

// Login sends login request to the game server with an extended 30 s timeout.
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
	replyBody, err := n.sendRequestWithTimeout("gamepb.userpb.UserService", "Login", body, loginTimeout)
	if err != nil {
		// Tag disconnect reason so the watchdog can differentiate.
		if strings.Contains(err.Error(), "timeout") {
			n.disconnectWithReason(DisconnectLoginTimeout)
		} else {
			n.disconnectWithReason(DisconnectLoginFailed)
		}
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

// ---------------------------------------------------------------------------
// Heartbeat
// ---------------------------------------------------------------------------

// StartHeartbeat runs the heartbeat loop until context is cancelled.
//
// Improvements over the basic version:
//   - Tracks time since last successful response for richer diagnostics
//   - Proactively clears stale pending calls when health degrades
//   - Syncs server time delta from HeartbeatReply
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

				// Diagnostic: check elapsed time since last successful heartbeat
				lastMs := n.lastHeartbeatAt.Load()
				elapsed := time.Since(time.UnixMilli(lastMs))
				if elapsed > heartbeatResponseDeadline {
					n.logger.Warnf("心跳", "连接可能已断开 (%ds 无响应, pending=%d)",
						int(elapsed.Seconds()), n.pendingCount())
				}

				req := &userpb.HeartbeatRequest{Gid: gid, ClientVersion: clientVersion}
				body, _ := proto.Marshal(req)
				replyBody, err := n.SendRequest("gamepb.userpb.UserService", "Heartbeat", body)
				if err != nil {
					consecutiveFailures++
					n.logger.Warnf("心跳", "失败 (%d/%d): %v", consecutiveFailures, maxHeartbeatFailures, err)

					// Health degradation: clear stale pending calls to prevent
					// memory leaks and goroutine pile-up.
					if consecutiveFailures >= 2 {
						n.clearPendingCalls("心跳连续失败，清理残留请求")
					}

					if consecutiveFailures >= maxHeartbeatFailures {
						n.logger.Warnf("心跳", "连续失败 %d 次，断开连接", maxHeartbeatFailures)
						n.disconnectWithReason(DisconnectHeartbeatTimeout)
						return
					}
				} else {
					consecutiveFailures = 0
					n.lastHeartbeatAt.Store(time.Now().UnixMilli())
					// Sync server time from heartbeat reply
					n.syncServerTime(replyBody)
				}
			}
		}
	}()
}

// pendingCount returns the number of in-flight pending requests.
func (n *Network) pendingCount() int {
	n.pendingMu.Lock()
	defer n.pendingMu.Unlock()
	return len(n.pending)
}

// clearPendingCalls fails and removes all pending request callbacks.
func (n *Network) clearPendingCalls(reason string) {
	n.pendingMu.Lock()
	count := len(n.pending)
	for seq, p := range n.pending {
		p.timer.Stop()
		p.ch <- &callResult{err: fmt.Errorf(reason)}
		delete(n.pending, seq)
	}
	n.pendingMu.Unlock()
	if count > 0 {
		n.logger.Warnf("心跳", "已清理 %d 个残留请求", count)
	}
}

// syncServerTime extracts server_time from HeartbeatReply and updates the
// server-local time delta.
func (n *Network) syncServerTime(replyBody []byte) {
	if len(replyBody) == 0 {
		return
	}
	reply := &userpb.HeartbeatReply{}
	if err := proto.Unmarshal(replyBody, reply); err != nil {
		return
	}
	if reply.ServerTime > 0 {
		localNow := time.Now().UnixMilli()
		n.serverTimeDelta.Store(reply.ServerTime - localNow)
	}
}

// ---------------------------------------------------------------------------
// Farm RPCs (unchanged)
// ---------------------------------------------------------------------------

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

// UnlockLand sends a land unlock request.
func (n *Network) UnlockLand(landID int64) (*plantpb.UnlockLandReply, error) {
	req := &plantpb.UnlockLandRequest{LandId: landID}
	body, _ := proto.Marshal(req)
	replyBody, err := n.SendRequest("gamepb.plantpb.PlantService", "UnlockLand", body)
	if err != nil {
		return nil, err
	}
	reply := &plantpb.UnlockLandReply{}
	return reply, proto.Unmarshal(replyBody, reply)
}

// UpgradeLand sends a land upgrade request.
func (n *Network) UpgradeLand(landID int64) (*plantpb.UpgradeLandReply, error) {
	req := &plantpb.UpgradeLandRequest{LandId: landID}
	body, _ := proto.Marshal(req)
	replyBody, err := n.SendRequest("gamepb.plantpb.PlantService", "UpgradeLand", body)
	if err != nil {
		return nil, err
	}
	reply := &plantpb.UpgradeLandReply{}
	return reply, proto.Unmarshal(replyBody, reply)
}
