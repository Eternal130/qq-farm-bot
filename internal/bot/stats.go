package bot

import (
	"qq-farm-bot/internal/model"
	"qq-farm-bot/internal/store"
)

// StatsCollector records operation statistics to the database.
// It is safe for concurrent use — each call writes directly to SQLite
// (which handles concurrency via WAL mode and busy timeout).
type StatsCollector struct {
	accountID int64
	store     *store.Store
}

// NewStatsCollector creates a new stats collector for the given account.
func NewStatsCollector(accountID int64, s *store.Store) *StatsCollector {
	return &StatsCollector{accountID: accountID, store: s}
}

// Record writes a single operation record to the database.
// count: number of items/lands involved in this operation.
// goldDelta: gold change (positive=earned, negative=spent).
// expDelta: experience earned.
func (sc *StatsCollector) Record(opType string, count int64, goldDelta int64, expDelta int64) {
	if sc == nil || sc.store == nil || count == 0 {
		return
	}
	_ = sc.store.AddOpStat(&model.OpRecord{
		AccountID: sc.accountID,
		OpType:    opType,
		Count:     count,
		GoldDelta: goldDelta,
		ExpDelta:  expDelta,
	})
}

// RecordSimple writes a simple count-only operation record.
func (sc *StatsCollector) RecordSimple(opType string, count int64) {
	sc.Record(opType, count, 0, 0)
}

// RecordWithDetail writes an operation record with a detail string (e.g., crop name, friend name).
func (sc *StatsCollector) RecordWithDetail(opType string, count int64, goldDelta int64, expDelta int64, detail string) {
	if sc == nil || sc.store == nil || count == 0 {
		return
	}
	_ = sc.store.AddOpStat(&model.OpRecord{
		AccountID: sc.accountID,
		OpType:    opType,
		Count:     count,
		GoldDelta: goldDelta,
		ExpDelta:  expDelta,
		Detail:    detail,
	})
}
