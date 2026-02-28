package bot

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"qq-farm-bot/proto/friendpb"
	"qq-farm-bot/proto/plantpb"
	"qq-farm-bot/proto/visitpb"
)

type FriendWorker struct {
	net    *Network
	logger *Logger
	cfg    *BotConfig
	gc     *GameConfig
	stats  *BotStats
}

type BotStats struct {
	TotalSteal   int64
	TotalHelp    int64
	FriendsCount int
}

func NewFriendWorker(net *Network, logger *Logger, cfg *BotConfig, stats *BotStats) *FriendWorker {
	return &FriendWorker{net: net, logger: logger, cfg: cfg, gc: GetGameConfig(), stats: stats}
}

func (fw *FriendWorker) RunLoop() {
	select {
	case <-time.After(5 * time.Second):
	case <-fw.net.ctx.Done():
		return
	}

	fw.checkAndAcceptApplications()

	for {
		fw.checkFriends()
		select {
		case <-time.After(time.Duration(fw.cfg.FriendInterval) * time.Second):
		case <-fw.net.ctx.Done():
			return
		}
	}
}

func (fw *FriendWorker) checkFriends() {
	gid, _, _, _, _ := fw.net.state.Get()
	if gid == 0 {
		return
	}

	req := &friendpb.GetAllRequest{}
	body, _ := proto.Marshal(req)
	replyBody, err := fw.net.SendRequest("gamepb.friendpb.FriendService", "GetAll", body)
	if err != nil {
		fw.logger.Warnf("好友", "获取好友失败: %v", err)
		return
	}
	reply := &friendpb.GetAllReply{}
	proto.Unmarshal(replyBody, reply)

	friends := reply.GameFriends
	if len(friends) == 0 {
		return
	}
	fw.stats.FriendsCount = len(friends)

	type friendTarget struct {
		gid  int64
		name string
	}
	var targets []friendTarget

	for _, f := range friends {
		if f.Gid == gid {
			continue
		}
		name := f.Remark
		if name == "" {
			name = f.Name
		}
		if name == "" {
			name = fmt.Sprintf("GID:%d", f.Gid)
		}

		hasSteal := f.Plant != nil && f.Plant.StealPlantNum > 0
		hasHelp := f.Plant != nil && (f.Plant.DryNum > 0 || f.Plant.WeedNum > 0 || f.Plant.InsectNum > 0)

		canSteal := hasSteal && fw.cfg.EnableSteal
		canHelp := hasHelp && fw.cfg.EnableHelpFriend

		if canSteal || canHelp {
			targets = append(targets, friendTarget{gid: f.Gid, name: name})
		}
	}

	if len(targets) == 0 {
		return
	}

	totalActions := struct {
		steal, water, weed, bug int
	}{}

	for _, t := range targets {
		actions := fw.visitFriend(t.gid, t.name, gid)
		totalActions.steal += actions.steal
		totalActions.water += actions.water
		totalActions.weed += actions.weed
		totalActions.bug += actions.bug
		time.Sleep(500 * time.Millisecond)
	}

	var summary []string
	if totalActions.steal > 0 {
		summary = append(summary, fmt.Sprintf("偷%d", totalActions.steal))
		fw.stats.TotalSteal += int64(totalActions.steal)
	}
	if totalActions.weed > 0 {
		summary = append(summary, fmt.Sprintf("除草%d", totalActions.weed))
	}
	if totalActions.bug > 0 {
		summary = append(summary, fmt.Sprintf("除虫%d", totalActions.bug))
	}
	if totalActions.water > 0 {
		summary = append(summary, fmt.Sprintf("浇水%d", totalActions.water))
	}
	if totalActions.weed+totalActions.bug+totalActions.water > 0 {
		fw.stats.TotalHelp += int64(totalActions.weed + totalActions.bug + totalActions.water)
	}
	if len(summary) > 0 {
		fw.logger.Infof("好友", "巡查 %d 人 → %s", len(targets), strings.Join(summary, "/"))
	}
}

type friendActions struct {
	steal, water, weed, bug int
}

func (fw *FriendWorker) visitFriend(friendGid int64, name string, myGid int64) friendActions {
	var actions friendActions

	enterReq := &visitpb.EnterRequest{HostGid: friendGid, Reason: 2}
	enterBody, _ := proto.Marshal(enterReq)
	enterReplyBody, err := fw.net.SendRequest("gamepb.visitpb.VisitService", "Enter", enterBody)
	if err != nil {
		return actions
	}
	enterReply := &visitpb.EnterReply{}
	proto.Unmarshal(enterReplyBody, enterReply)

	defer func() {
		leaveReq := &visitpb.LeaveRequest{HostGid: friendGid}
		leaveBody, _ := proto.Marshal(leaveReq)
		fw.net.SendRequest("gamepb.visitpb.VisitService", "Leave", leaveBody)
	}()

	lands := enterReply.Lands
	if len(lands) == 0 {
		return actions
	}

	status := fw.analyzeFriendLands(lands, myGid)
	var parts []string

	// Help operations (respect config toggle)
	if fw.cfg.EnableHelpFriend {
		if len(status.needWeed) > 0 {
			for _, landID := range status.needWeed {
				req := &plantpb.WeedOutRequest{LandIds: []int64{landID}, HostGid: friendGid}
				body, _ := proto.Marshal(req)
				if _, err := fw.net.SendRequest("gamepb.plantpb.PlantService", "WeedOut", body); err == nil {
					actions.weed++
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
		if len(status.needBug) > 0 {
			for _, landID := range status.needBug {
				req := &plantpb.InsecticideRequest{LandIds: []int64{landID}, HostGid: friendGid}
				body, _ := proto.Marshal(req)
				if _, err := fw.net.SendRequest("gamepb.plantpb.PlantService", "Insecticide", body); err == nil {
					actions.bug++
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
		if len(status.needWater) > 0 {
			for _, landID := range status.needWater {
				req := &plantpb.WaterLandRequest{LandIds: []int64{landID}, HostGid: friendGid}
				body, _ := proto.Marshal(req)
				if _, err := fw.net.SendRequest("gamepb.plantpb.PlantService", "WaterLand", body); err == nil {
					actions.water++
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	// Steal (respect config + crop filter)
	if fw.cfg.EnableSteal && len(status.stealable) > 0 {
		stealFilter := ParseCropIDs(fw.cfg.StealCropIDs)
		hasStealFilter := len(stealFilter) > 0

		for _, sl := range status.stealable {
			if hasStealFilter && !stealFilter[int(sl.cropID)] {
				continue
			}
			req := &plantpb.HarvestRequest{LandIds: []int64{sl.landID}, HostGid: friendGid, IsAll: true}
			body, _ := proto.Marshal(req)
			if _, err := fw.net.SendRequest("gamepb.plantpb.PlantService", "Harvest", body); err == nil {
				actions.steal++
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	if actions.weed > 0 {
		parts = append(parts, fmt.Sprintf("草%d", actions.weed))
	}
	if actions.bug > 0 {
		parts = append(parts, fmt.Sprintf("虫%d", actions.bug))
	}
	if actions.water > 0 {
		parts = append(parts, fmt.Sprintf("水%d", actions.water))
	}
	if actions.steal > 0 {
		parts = append(parts, fmt.Sprintf("偷%d", actions.steal))
	}
	if len(parts) > 0 {
		fw.logger.Infof("好友", "%s: %s", name, strings.Join(parts, "/"))
	}

	return actions
}

type stealableLand struct {
	landID int64
	cropID int64
}

type friendLandStatus struct {
	stealable []stealableLand
	needWater []int64
	needWeed  []int64
	needBug   []int64
}

func (fw *FriendWorker) analyzeFriendLands(lands []*plantpb.LandInfo, myGid int64) *friendLandStatus {
	s := &friendLandStatus{}
	nowSec := time.Now().Unix()

	for _, land := range lands {
		plant := land.Plant
		if plant == nil || len(plant.Phases) == 0 {
			continue
		}
		phase := getCurrentPhase(plant.Phases, nowSec)
		if phase == nil {
			continue
		}

		switch plantpb.PlantPhase(phase.Phase) {
		case plantpb.PlantPhase_MATURE:
			if plant.Stealable {
				s.stealable = append(s.stealable, stealableLand{landID: land.Id, cropID: plant.Id})
			}
		case plantpb.PlantPhase_DEAD:
			continue
		default:
			if plant.DryNum > 0 {
				s.needWater = append(s.needWater, land.Id)
			}
			if len(plant.WeedOwners) > 0 {
				s.needWeed = append(s.needWeed, land.Id)
			}
			if len(plant.InsectOwners) > 0 {
				s.needBug = append(s.needBug, land.Id)
			}
		}
	}
	return s
}

func (fw *FriendWorker) checkAndAcceptApplications() {
	req := &friendpb.GetApplicationsRequest{}
	body, _ := proto.Marshal(req)
	replyBody, err := fw.net.SendRequest("gamepb.friendpb.FriendService", "GetApplications", body)
	if err != nil {
		return
	}
	reply := &friendpb.GetApplicationsReply{}
	proto.Unmarshal(replyBody, reply)

	if len(reply.Applications) == 0 {
		return
	}

	gids := make([]int64, len(reply.Applications))
	names := make([]string, len(reply.Applications))
	for i, a := range reply.Applications {
		gids[i] = a.Gid
		names[i] = a.Name
	}

	acceptReq := &friendpb.AcceptFriendsRequest{FriendGids: gids}
	acceptBody, _ := proto.Marshal(acceptReq)
	if _, err := fw.net.SendRequest("gamepb.friendpb.FriendService", "AcceptFriends", acceptBody); err == nil {
		fw.logger.Infof("申请", "已同意 %d 人: %s", len(gids), strings.Join(names, ", "))
	}
}
