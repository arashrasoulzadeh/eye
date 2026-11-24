package service

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"sync"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

type QueryService struct {
	DB *gorm.DB
	// SetTenantContext is configured for PostgreSQL/Timescale deployments so
	// every tenant-filtered read also carries transaction-local RLS context.
	// It remains nil for SQLite/dev deployments.
	SetTenantContext func(*gorm.DB, string) error
	flights singleflight.Group
	gateOnce sync.Once
	gate chan struct{}
	queue chan struct{}
	tenantMu sync.Mutex
	tenants map[string]int
	cacheMu sync.Mutex
	cache map[string]queryCacheEntry
}

type queryCacheEntry struct {
	expires, accessed time.Time
	rows []model.Event
}

func cloneQueryRows(rows []model.Event) []model.Event {
	if rows == nil { return nil }
	out := append(make([]model.Event, 0, len(rows)), rows...)
	for i := range out {
		out[i].IntelligenceTags = append([]string(nil), rows[i].IntelligenceTags...)
	}
	return out
}

func (q *QueryService) cached(key string) ([]model.Event, bool) {
	q.cacheMu.Lock()
	defer q.cacheMu.Unlock()
	entry, ok := q.cache[key]
	if !ok || !time.Now().Before(entry.expires) { delete(q.cache, key); return nil, false }
	entry.accessed = time.Now()
	q.cache[key] = entry
	return cloneQueryRows(entry.rows), true
}

func (q *QueryService) store(key string, rows []model.Event) {
	if len(rows) > 500 { return }
	size := 0
	for _, row := range rows {
		size += len(row.Payload) + len(row.Dims) + len(row.EventName) + len(row.EventID) + len(row.SessionID) + len(row.TenantID) + len(row.UserID) + len(row.ContentHash) + len(row.Source) + len(row.AggregationID) + len(row.ExperimentID) + len(row.Variant) + 256
		if size > 128*1024 { return }
	}
	q.cacheMu.Lock()
	defer q.cacheMu.Unlock()
	if q.cache == nil { q.cache = make(map[string]queryCacheEntry) }
	now := time.Now()
	for k, e := range q.cache { if !now.Before(e.expires) { delete(q.cache, k) } }
	if len(q.cache) >= 128 {
		var oldest string
		var at time.Time
		for k, e := range q.cache { if at.IsZero() || e.accessed.Before(at) { oldest, at = k, e.accessed } }
		delete(q.cache, oldest)
	}
	q.cache[key] = queryCacheEntry{now.Add(5*time.Second), now, cloneQueryRows(rows)}
}

func (q *QueryService) EventsByName(name string, limit int) ([]model.Event, error) {
	var list []model.Event
	if err := q.DB.Where("event_name = ?", name).Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// EventFilter is a conjunction of optional filters for Filter. Zero-value
// fields are treated as "unset" and don't constrain the query.
type EventFilter struct {
	TenantID  string
	SessionID string
	EventName string
	// FromTS/ToTS bound ClientTS (inclusive) when set. nil means unbounded.
	FromTS *int64
	ToTS   *int64
	// DimsContains does a substring match against the raw Dims JSON column.
	// This is a pragmatic stand-in for real JSON querying: it works on
	// SQLite (dev) and Postgres/Timescale (prod) alike without a JSON
	// operator dependency, at the cost of being a text match rather than a
	// structural one.
	DimsContains string
	// SortDesc orders results by ClientTS descending (newest first) instead
	// of the default ascending order. Ascending stays the default because
	// gRPC streaming pagination (query_grpc.go) and existing callers rely on
	// a stable, append-only chronological order; only the admin Events page
	// opts into newest-first via the REST handler's `sort=desc` param.
	SortDesc      bool
	AfterClientTS *int64
	AfterEventID  string
}

// Filter returns events matching all set fields of f, ordered by ClientTS
// ascending, with limit/offset pagination.
func (q *QueryService) Filter(f EventFilter, limit, offset int) ([]model.Event, error) {
	return q.FilterContext(context.Background(), f, limit, offset)
}

// FilterContext preserves the existing Filter contract while allowing callers to cancel.
func (q *QueryService) FilterContext(ctx context.Context, f EventFilter, limit, offset int) ([]model.Event, error) {
	keyBytes, err := json.Marshal(struct { Filter EventFilter; Limit, Offset int }{f, limit, offset})
	if err != nil { return nil, err }
	key := string(keyBytes)
	if err := ctx.Err(); err != nil { return nil, err }
	// Bound waiting callers as well as actual database work.
	q.gateOnce.Do(func() { q.gate = make(chan struct{}, 8); q.queue = make(chan struct{}, 128); q.tenants = make(map[string]int) })
	if err := ctx.Err(); err != nil { return nil, err }
	ch := q.flights.DoChan(key, func() (any, error) {
		select { case q.queue <- struct{}{}: defer func(){ <-q.queue }(); default: return nil, fmt.Errorf("query queue full") }
		work, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		q.tenantMu.Lock()
		if q.tenants[f.TenantID] >= 4 { q.tenantMu.Unlock(); return nil, fmt.Errorf("tenant query limit reached") }
		q.tenants[f.TenantID]++
		q.tenantMu.Unlock()
		defer func(){ q.tenantMu.Lock(); q.tenants[f.TenantID]--; if q.tenants[f.TenantID] == 0 { delete(q.tenants, f.TenantID) }; q.tenantMu.Unlock() }()
		select { case q.gate <- struct{}{}: defer func(){ <-q.gate }(); case <-work.Done(): return nil, work.Err() }
		if rows, ok := q.cached(key); ok { return rows, nil }
		rows, err := q.filterDB(work, f, limit, offset)
		if err == nil { q.store(key, rows) }
		return rows, err
	})
	select {
	case <-ctx.Done(): return nil, ctx.Err()
	case result := <-ch:
		if result.Err != nil { return nil, result.Err }
		rows := result.Val.([]model.Event)
		return cloneQueryRows(rows), nil
	}
}

func (q *QueryService) filterDB(ctx context.Context, f EventFilter, limit, offset int) ([]model.Event, error) {
	queryDB := q.DB.WithContext(ctx)
	var tx *gorm.DB
	if q.SetTenantContext != nil && f.TenantID != "" {
		tx = queryDB.Begin()
		if tx.Error != nil {
			return nil, tx.Error
		}
		if err := q.SetTenantContext(tx, f.TenantID); err != nil {
			tx.Rollback()
			return nil, err
		}
		queryDB = tx
	}
	query := queryDB.Model(&model.Event{})
	if f.TenantID != "" {
		query = query.Where("tenant_id = ?", f.TenantID)
	}
	if f.SessionID != "" {
		query = query.Where("session_id = ?", f.SessionID)
	}
	if f.EventName != "" {
		query = query.Where("event_name = ?", f.EventName)
	}
	if f.FromTS != nil {
		query = query.Where("client_ts >= ?", *f.FromTS)
	}
	if f.ToTS != nil {
		query = query.Where("client_ts <= ?", *f.ToTS)
	}
	if f.DimsContains != "" {
		query = query.Where("dims LIKE ?", "%"+f.DimsContains+"%")
	}

	order := "client_ts asc"
	if f.SortDesc {
		order = "client_ts desc"
	}
	if f.AfterClientTS != nil && f.AfterEventID != "" {
		if f.SortDesc {
			query = query.Where("(client_ts < ?) OR (client_ts = ? AND event_id < ?)", *f.AfterClientTS, *f.AfterClientTS, f.AfterEventID)
		} else {
			query = query.Where("(client_ts > ?) OR (client_ts = ? AND event_id > ?)", *f.AfterClientTS, *f.AfterClientTS, f.AfterEventID)
		}
	}
	var list []model.Event
	if err := query.Order(order + ", event_id " + map[bool]string{true: "desc", false: "asc"}[f.SortDesc]).Limit(limit).Offset(offset).Find(&list).Error; err != nil {
		if tx != nil {
			tx.Rollback()
		}
		return nil, err
	}
	if tx != nil {
		if err := tx.Commit().Error; err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (q *QueryService) Count(f EventFilter) (int64, error) {
	return q.CountContext(context.Background(), f)
}

func (q *QueryService) CountContext(ctx context.Context, f EventFilter) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	q.gateOnce.Do(func() { q.gate = make(chan struct{}, 8); q.queue = make(chan struct{}, 128); q.tenants = make(map[string]int) })
	select { case q.queue <- struct{}{}: defer func(){ <-q.queue }(); default: return 0, fmt.Errorf("query queue full") }
	q.tenantMu.Lock()
	if q.tenants[f.TenantID] >= 4 { q.tenantMu.Unlock(); return 0, fmt.Errorf("tenant query limit reached") }
	q.tenants[f.TenantID]++
	q.tenantMu.Unlock()
	defer func(){ q.tenantMu.Lock(); q.tenants[f.TenantID]--; if q.tenants[f.TenantID] == 0 { delete(q.tenants, f.TenantID) }; q.tenantMu.Unlock() }()
	select { case q.gate <- struct{}{}: defer func(){ <-q.gate }(); case <-ctx.Done(): return 0, ctx.Err() }
	db := q.DB.WithContext(ctx)
	if q.SetTenantContext != nil && f.TenantID != "" {
		var total int64
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := q.SetTenantContext(tx, f.TenantID); err != nil { return err }
			var err error
			total, err = countEvents(tx, f)
			return err
		})
		return total, err
	}
	return countEvents(db, f)
}

func countEvents(db *gorm.DB, f EventFilter) (int64, error) {
	db = db.Model(&model.Event{})
	if f.TenantID != "" {
		db = db.Where("tenant_id = ?", f.TenantID)
	}
	if f.SessionID != "" {
		db = db.Where("session_id = ?", f.SessionID)
	}
	if f.EventName != "" {
		db = db.Where("event_name = ?", f.EventName)
	}
	if f.DimsContains != "" {
		db = db.Where("dims LIKE ?", "%"+f.DimsContains+"%")
	}
	if f.FromTS != nil {
		db = db.Where("client_ts >= ?", *f.FromTS)
	}
	if f.ToTS != nil {
		db = db.Where("client_ts <= ?", *f.ToTS)
	}
	var total int64
	err := db.Count(&total).Error
	return total, err
}
