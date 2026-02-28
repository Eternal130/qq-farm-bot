package bot

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"qq-farm-bot/proto/corepb"
	"qq-farm-bot/proto/itempb"
)

type WarehouseWorker struct {
	net    *Network
	logger *Logger
	cfg    *BotConfig
	gc     *GameConfig
}

func NewWarehouseWorker(net *Network, logger *Logger, cfg *BotConfig) *WarehouseWorker {
	return &WarehouseWorker{net: net, logger: logger, cfg: cfg, gc: GetGameConfig()}
}

func (ww *WarehouseWorker) RunLoop() {
	if !ww.cfg.EnableSell {
		return
	}

	select {
	case <-time.After(10 * time.Second):
	case <-ww.net.ctx.Done():
		return
	}

	ww.sellAllFruits()

	for {
		select {
		case <-time.After(60 * time.Second):
			ww.sellAllFruits()
		case <-ww.net.ctx.Done():
			return
		}
	}
}

func (ww *WarehouseWorker) sellAllFruits() {
	req := &itempb.BagRequest{}
	body, _ := proto.Marshal(req)
	replyBody, err := ww.net.SendRequest("gamepb.itempb.ItemService", "Bag", body)
	if err != nil {
		return
	}
	reply := &itempb.BagReply{}
	proto.Unmarshal(replyBody, reply)

	if reply.ItemBag == nil || len(reply.ItemBag.Items) == 0 {
		return
	}

	sellFilter := ParseCropIDs(ww.cfg.SellCropIDs)
	hasSellFilter := len(sellFilter) > 0

	var toSell []*corepb.Item
	var names []string

	for _, item := range reply.ItemBag.Items {
		id := int(item.Id)
		count := item.Count
		if ww.gc.IsFruitID(id) && count > 0 && item.Uid > 0 {
			if hasSellFilter {
				plantID := ww.gc.GetFruitPlantID(id)
				if plantID == 0 || !sellFilter[plantID] {
					continue
				}
			}
			toSell = append(toSell, item)
			names = append(names, fmt.Sprintf("%sx%d", ww.gc.GetFruitName(id), count))
		}
	}

	if len(toSell) == 0 {
		return
	}

	sellReq := &itempb.SellRequest{Items: toSell}
	sellBody, _ := proto.Marshal(sellReq)
	sellReplyBody, err := ww.net.SendRequest("gamepb.itempb.ItemService", "Sell", sellBody)
	if err != nil {
		ww.logger.Warnf("仓库", "出售失败: %v", err)
		return
	}

	sellReply := &itempb.SellReply{}
	proto.Unmarshal(sellReplyBody, sellReply)

	var totalGold int64
	for _, item := range sellReply.GetItems {
		if item.Id == 1001 || item.Id == 1 {
			totalGold = item.Count
		}
	}

	ww.logger.Infof("仓库", "出售 %s，获得 %d 金币", strings.Join(names, ", "), totalGold)
}
