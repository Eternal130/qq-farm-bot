package bot

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"qq-farm-bot/proto/corepb"
	"qq-farm-bot/proto/taskpb"
)

type TaskWorker struct {
	net    *Network
	logger *Logger
	cfg    *BotConfig
}

func NewTaskWorker(net *Network, logger *Logger, cfg *BotConfig) *TaskWorker {
	return &TaskWorker{net: net, logger: logger, cfg: cfg}
}

func (tw *TaskWorker) RunLoop() {
	if !tw.cfg.EnableClaimTask {
		return
	}

	select {
	case <-time.After(4 * time.Second):
	case <-tw.net.ctx.Done():
		return
	}

	tw.checkAndClaim()

	for {
		select {
		case <-time.After(5 * time.Minute):
			tw.checkAndClaim()
		case <-tw.net.ctx.Done():
			return
		}
	}
}

func (tw *TaskWorker) checkAndClaim() {
	req := &taskpb.TaskInfoRequest{}
	body, _ := proto.Marshal(req)
	replyBody, err := tw.net.SendRequest("gamepb.taskpb.TaskService", "TaskInfo", body)
	if err != nil {
		return
	}
	reply := &taskpb.TaskInfoReply{}
	proto.Unmarshal(replyBody, reply)

	if reply.TaskInfo == nil {
		return
	}

	var allTasks []*taskpb.Task
	allTasks = append(allTasks, reply.TaskInfo.GrowthTasks...)
	allTasks = append(allTasks, reply.TaskInfo.DailyTasks...)
	allTasks = append(allTasks, reply.TaskInfo.Tasks...)

	var claimable []*taskpb.Task
	for _, task := range allTasks {
		if task.IsUnlocked && !task.IsClaimed && task.Progress >= task.TotalProgress && task.TotalProgress > 0 {
			claimable = append(claimable, task)
		}
	}

	if len(claimable) == 0 {
		return
	}

	tw.logger.Infof("任务", "发现 %d 个可领取任务", len(claimable))

	for _, task := range claimable {
		useShare := task.ShareMultiple > 1
		claimReq := &taskpb.ClaimTaskRewardRequest{Id: task.Id, DoShared: useShare}
		claimBody, _ := proto.Marshal(claimReq)
		claimReplyBody, err := tw.net.SendRequest("gamepb.taskpb.TaskService", "ClaimTaskReward", claimBody)
		if err != nil {
			tw.logger.Warnf("任务", "领取失败 #%d: %v", task.Id, err)
			continue
		}

		claimReply := &taskpb.ClaimTaskRewardReply{}
		proto.Unmarshal(claimReplyBody, claimReply)

		rewardStr := formatRewards(claimReply.Items)
		multiStr := ""
		if useShare {
			multiStr = fmt.Sprintf(" (%d倍)", task.ShareMultiple)
		}
		tw.logger.Infof("任务", "领取: %s%s → %s", task.Desc, multiStr, rewardStr)
		time.Sleep(300 * time.Millisecond)
	}
}

func formatRewards(items []*corepb.Item) string {
	if len(items) == 0 {
		return "无"
	}
	var parts []string
	for _, item := range items {
		switch item.Id {
		case 1:
			parts = append(parts, fmt.Sprintf("金币%d", item.Count))
		case 2:
			parts = append(parts, fmt.Sprintf("经验%d", item.Count))
		default:
			parts = append(parts, fmt.Sprintf("物品(%d)x%d", item.Id, item.Count))
		}
	}
	return strings.Join(parts, "/")
}
