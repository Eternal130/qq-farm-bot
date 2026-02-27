package bot

import (
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"qq-farm-bot/proto/corepb"
	"qq-farm-bot/proto/itempb"
	"qq-farm-bot/proto/mallpb"
)

// Item IDs for fertilizer system
const (
	couponItemID          = 1002   // 点券
	fertilizerPackID1     = 100003 // 化肥礼包1
	fertilizerPackID2     = 100004 // 化肥礼包2
	normalFertilizer1h    = 80001
	normalFertilizer4h    = 80002
	normalFertilizer8h    = 80003
	normalFertilizer12h   = 80004
	organicFertilizer1h   = 80011
	organicFertilizer4h   = 80012
	organicFertilizer8h   = 80013
	organicFertilizer12h  = 80014
	normalContainerID     = 1011
	organicContainerID    = 1012
	containerLimitHours   = 990
	mallFertilizerGoodsID = 1003 // Mall goods_id for normal fertilizer pack

	fertilizerLoopInterval = 1 * time.Hour
	fertilizerInitialDelay = 15 * time.Second
	throttleDelay          = 300 * time.Millisecond
	buyCooldown            = 10 * time.Minute
)

// FertilizerWorker handles automatic fertilizer pack buying, opening, and usage.
type FertilizerWorker struct {
	net    *Network
	logger *Logger
	cfg    *BotConfig

	mu             sync.Mutex
	dailyBuyCount  int
	dailyOpenCount int
	dailyDate      string
	lastBuyTime    time.Time
}

func NewFertilizerWorker(net *Network, logger *Logger, cfg *BotConfig) *FertilizerWorker {
	return &FertilizerWorker{net: net, logger: logger, cfg: cfg}
}

func (fw *FertilizerWorker) RunLoop() {
	// Neither feature enabled — nothing to do
	if !fw.cfg.AutoUseFertilizer && !fw.cfg.AutoBuyFertilizer {
		return
	}

	select {
	case <-time.After(fertilizerInitialDelay):
	case <-fw.net.ctx.Done():
		return
	}

	fw.runFertilizerTask()

	for {
		select {
		case <-time.After(fertilizerLoopInterval):
			fw.runFertilizerTask()
		case <-fw.net.ctx.Done():
			return
		}
	}
}

// runFertilizerTask orchestrates: buy → open → use surplus.
func (fw *FertilizerWorker) runFertilizerTask() {
	fw.resetDailyCounters()

	items, err := fw.getBagItems()
	if err != nil {
		fw.logger.Warnf("化肥", "获取背包失败: %v", err)
		return
	}

	// Step 1: Buy fertilizer packs if enabled
	if fw.cfg.AutoBuyFertilizer {
		fw.buyFertilizerPacks(items)
		time.Sleep(throttleDelay)
		// Re-fetch bag after buying
		items, err = fw.getBagItems()
		if err != nil {
			fw.logger.Warnf("化肥", "获取背包失败: %v", err)
			return
		}
	}

	// Step 2: Open fertilizer packs if enabled
	if fw.cfg.AutoUseFertilizer {
		fw.openFertilizerPacks(items)
		time.Sleep(throttleDelay)
		// Re-fetch bag after opening
		items, err = fw.getBagItems()
		if err != nil {
			fw.logger.Warnf("化肥", "获取背包失败: %v", err)
			return
		}
	}

	// Step 3: Use surplus fertilizer items
	if fw.cfg.AutoUseFertilizer {
		fw.useSurplusFertilizer(items)
	}
}

// getBagItems fetches the current bag contents.
func (fw *FertilizerWorker) getBagItems() ([]*corepb.Item, error) {
	req := &itempb.BagRequest{}
	body, _ := proto.Marshal(req)
	replyBody, err := fw.net.SendRequest("gamepb.itempb.ItemService", "Bag", body)
	if err != nil {
		return nil, err
	}
	reply := &itempb.BagReply{}
	proto.Unmarshal(replyBody, reply)
	if reply.ItemBag == nil {
		return nil, nil
	}
	return reply.ItemBag.Items, nil
}

// findItem returns the item with the given ID from the bag, or nil.
func findItem(items []*corepb.Item, id int64) *corepb.Item {
	for _, item := range items {
		if item.Id == id {
			return item
		}
	}
	return nil
}

// getItemCount returns the count of an item by ID, or 0.
func getItemCount(items []*corepb.Item, id int64) int64 {
	item := findItem(items, id)
	if item != nil {
		return item.Count
	}
	return 0
}

// containerHours returns the current hours stored in a container.
// The game stores the count in seconds; we convert to hours.
func containerHours(items []*corepb.Item, containerID int64) int64 {
	count := getItemCount(items, containerID)
	return count / 3600
}

// totalFertilizerItemCount returns the total count of all fertilizer items (normal + organic).
func totalFertilizerItemCount(items []*corepb.Item) int64 {
	ids := []int64{
		normalFertilizer1h, normalFertilizer4h, normalFertilizer8h, normalFertilizer12h,
		organicFertilizer1h, organicFertilizer4h, organicFertilizer8h, organicFertilizer12h,
	}
	var total int64
	for _, id := range ids {
		total += getItemCount(items, id)
	}
	return total
}

// totalPackCount returns the total number of fertilizer packs in bag.
func totalPackCount(items []*corepb.Item) int64 {
	return getItemCount(items, fertilizerPackID1) + getItemCount(items, fertilizerPackID2)
}

// resetDailyCounters resets counters if the date has changed.
func (fw *FertilizerWorker) resetDailyCounters() {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	today := time.Now().Format("2006-01-02")
	if fw.dailyDate != today {
		fw.dailyDate = today
		fw.dailyBuyCount = 0
		fw.dailyOpenCount = 0
	}
}

// buyFertilizerPacks purchases fertilizer packs from MallService using coupons.
func (fw *FertilizerWorker) buyFertilizerPacks(items []*corepb.Item) {
	fw.mu.Lock()
	dailyLimit := fw.cfg.FertilizerBuyDailyLimit
	alreadyBought := fw.dailyBuyCount
	lastBuy := fw.lastBuyTime
	fw.mu.Unlock()

	// Check daily limit
	if dailyLimit > 0 && alreadyBought >= dailyLimit {
		return
	}

	// Check buy cooldown
	if time.Since(lastBuy) < buyCooldown {
		return
	}

	// Check container limit (don't buy if containers are near full)
	normalHours := containerHours(items, normalContainerID)
	if normalHours >= containerLimitHours {
		fw.logger.Infof("化肥", "普通化肥容器已满 (%d小时), 跳过购买", normalHours)
		return
	}

	// Check coupon balance
	couponBalance := getItemCount(items, couponItemID)
	if couponBalance <= 0 {
		return
	}

	// Get mall info to check price
	price, err := fw.getMallFertilizerPrice()
	if err != nil || price <= 0 {
		return
	}

	if couponBalance < int64(price) {
		fw.logger.Infof("化肥", "点券不足 (余额:%d, 价格:%d)", couponBalance, price)
		return
	}

	// Calculate how many to buy
	maxByBalance := int(couponBalance / int64(price))
	toBuy := maxByBalance
	if dailyLimit > 0 {
		remaining := dailyLimit - alreadyBought
		if toBuy > remaining {
			toBuy = remaining
		}
	}
	if toBuy <= 0 {
		return
	}

	// Purchase one at a time with throttle
	bought := 0
	for i := 0; i < toBuy; i++ {
		purchaseReq := &mallpb.PurchaseRequest{
			GoodsId: mallFertilizerGoodsID,
			Count:   1,
		}
		purchaseBody, _ := proto.Marshal(purchaseReq)
		_, err := fw.net.SendRequest("gamepb.mallpb.MallService", "Purchase", purchaseBody)
		if err != nil {
			fw.logger.Warnf("化肥", "购买失败: %v", err)
			break
		}
		bought++
		time.Sleep(throttleDelay)
	}

	fw.mu.Lock()
	fw.dailyBuyCount += bought
	fw.lastBuyTime = time.Now()
	fw.mu.Unlock()

	if bought > 0 {
		fw.logger.Infof("化肥", "购买化肥礼包 x%d (今日累计:%d)", bought, fw.dailyBuyCount)
	}
}

// getMallFertilizerPrice queries the mall for the fertilizer pack price in coupons.
func (fw *FertilizerWorker) getMallFertilizerPrice() (int32, error) {
	req := &mallpb.GetMallListBySlotTypeRequest{SlotType: 1}
	body, _ := proto.Marshal(req)
	replyBody, err := fw.net.SendRequest("gamepb.mallpb.MallService", "GetMallListBySlotType", body)
	if err != nil {
		return 0, err
	}
	reply := &mallpb.GetMallListBySlotTypeResponse{}
	proto.Unmarshal(replyBody, reply)

	for _, goodsBytes := range reply.GoodsList {
		goods := &mallpb.MallGoods{}
		if err := proto.Unmarshal(goodsBytes, goods); err != nil {
			continue
		}
		if goods.GoodsId == mallFertilizerGoodsID {
			price := parseMallPriceValue(goods.Price)
			return price, nil
		}
	}
	return 0, nil
}

// parseMallPriceValue extracts the coupon price from the serialized price bytes.
// The price field is a protobuf message where field_number=2 is the coupon price (varint).
func parseMallPriceValue(data []byte) int32 {
	if len(data) == 0 {
		return 0
	}
	i := 0
	for i < len(data) {
		if i >= len(data) {
			break
		}
		// Read tag: (field_number << 3) | wire_type
		tag := int(data[i])
		i++
		fieldNumber := tag >> 3
		wireType := tag & 0x07

		switch wireType {
		case 0: // varint
			val, n := decodeVarint(data[i:])
			i += n
			if fieldNumber == 2 {
				return int32(val)
			}
		case 2: // length-delimited
			length, n := decodeVarint(data[i:])
			i += n
			i += int(length) // skip the bytes
		default:
			// Unknown wire type, bail
			return 0
		}
	}
	return 0
}

// decodeVarint decodes a protobuf varint from data, returning the value and bytes consumed.
func decodeVarint(data []byte) (uint64, int) {
	var val uint64
	var shift uint
	for i, b := range data {
		val |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			return val, i + 1
		}
		shift += 7
		if shift >= 64 {
			return val, i + 1
		}
	}
	return val, len(data)
}

// openFertilizerPacks opens fertilizer packs using BatchUse.
func (fw *FertilizerWorker) openFertilizerPacks(items []*corepb.Item) {
	var toOpen []*itempb.BatchUseItem

	pack1Count := getItemCount(items, fertilizerPackID1)
	if pack1Count > 0 {
		toOpen = append(toOpen, &itempb.BatchUseItem{ItemId: fertilizerPackID1, Count: pack1Count})
	}
	pack2Count := getItemCount(items, fertilizerPackID2)
	if pack2Count > 0 {
		toOpen = append(toOpen, &itempb.BatchUseItem{ItemId: fertilizerPackID2, Count: pack2Count})
	}

	if len(toOpen) == 0 {
		return
	}

	req := &itempb.BatchUseRequest{Items: toOpen}
	body, _ := proto.Marshal(req)
	_, err := fw.net.SendRequest("gamepb.itempb.ItemService", "BatchUse", body)
	if err != nil {
		fw.logger.Warnf("化肥", "开启礼包失败: %v", err)
		return
	}

	totalPacks := pack1Count + pack2Count
	fw.mu.Lock()
	fw.dailyOpenCount += int(totalPacks)
	fw.mu.Unlock()

	fw.logger.Infof("化肥", "开启化肥礼包 x%d", totalPacks)
}

// useSurplusFertilizer uses excess fertilizer items to fill containers when above target threshold.
func (fw *FertilizerWorker) useSurplusFertilizer(items []*corepb.Item) {
	targetCount := int64(fw.cfg.FertilizerTargetCount)
	totalItems := totalFertilizerItemCount(items)

	// Only use surplus when we have more than the target
	if targetCount > 0 && totalItems <= targetCount {
		return
	}

	normalHours := containerHours(items, normalContainerID)
	organicHours := containerHours(items, organicContainerID)

	var toUse []*itempb.BatchUseItem
	var usedDesc []string

	// Use normal fertilizer items to fill normal container
	if normalHours < containerLimitHours {
		normalIDs := []int64{normalFertilizer12h, normalFertilizer8h, normalFertilizer4h, normalFertilizer1h}
		normalHoursMap := map[int64]int64{
			normalFertilizer12h: 12,
			normalFertilizer8h:  8,
			normalFertilizer4h:  4,
			normalFertilizer1h:  1,
		}
		for _, id := range normalIDs {
			count := getItemCount(items, id)
			if count <= 0 {
				continue
			}
			hoursPerItem := normalHoursMap[id]
			// Calculate how many we can use without exceeding container limit
			spaceHours := containerLimitHours - normalHours
			maxBySpace := spaceHours / hoursPerItem
			if maxBySpace <= 0 {
				continue
			}
			useCount := count
			if useCount > maxBySpace {
				useCount = maxBySpace
			}
			// Keep target amount
			if targetCount > 0 {
				surplus := totalItems - targetCount
				if surplus <= 0 {
					break
				}
				if useCount > surplus {
					useCount = surplus
				}
			}
			if useCount > 0 {
				toUse = append(toUse, &itempb.BatchUseItem{ItemId: id, Count: useCount})
				normalHours += useCount * hoursPerItem
				totalItems -= useCount
				usedDesc = append(usedDesc, itemName(id, useCount))
			}
		}
	}

	// Use organic fertilizer items to fill organic container
	if organicHours < containerLimitHours {
		organicIDs := []int64{organicFertilizer12h, organicFertilizer8h, organicFertilizer4h, organicFertilizer1h}
		organicHoursMap := map[int64]int64{
			organicFertilizer12h: 12,
			organicFertilizer8h:  8,
			organicFertilizer4h:  4,
			organicFertilizer1h:  1,
		}
		for _, id := range organicIDs {
			count := getItemCount(items, id)
			if count <= 0 {
				continue
			}
			hoursPerItem := organicHoursMap[id]
			spaceHours := containerLimitHours - organicHours
			maxBySpace := spaceHours / hoursPerItem
			if maxBySpace <= 0 {
				continue
			}
			useCount := count
			if useCount > maxBySpace {
				useCount = maxBySpace
			}
			if targetCount > 0 {
				surplus := totalItems - targetCount
				if surplus <= 0 {
					break
				}
				if useCount > surplus {
					useCount = surplus
				}
			}
			if useCount > 0 {
				toUse = append(toUse, &itempb.BatchUseItem{ItemId: id, Count: useCount})
				organicHours += useCount * hoursPerItem
				totalItems -= useCount
				usedDesc = append(usedDesc, itemName(id, useCount))
			}
		}
	}

	if len(toUse) == 0 {
		return
	}

	req := &itempb.BatchUseRequest{Items: toUse}
	body, _ := proto.Marshal(req)
	_, err := fw.net.SendRequest("gamepb.itempb.ItemService", "BatchUse", body)
	if err != nil {
		fw.logger.Warnf("化肥", "使用化肥失败: %v", err)
		return
	}

	fw.logger.Infof("化肥", "使用化肥: 普通容器%d小时 有机容器%d小时", normalHours, organicHours)
}

// itemName returns a display string for a fertilizer item.
func itemName(id, count int64) string {
	names := map[int64]string{
		normalFertilizer1h:   "普通1h",
		normalFertilizer4h:   "普通4h",
		normalFertilizer8h:   "普通8h",
		normalFertilizer12h:  "普通12h",
		organicFertilizer1h:  "有机1h",
		organicFertilizer4h:  "有机4h",
		organicFertilizer8h:  "有机8h",
		organicFertilizer12h: "有机12h",
	}
	name := names[id]
	if name == "" {
		name = "未知"
	}
	return name
}
