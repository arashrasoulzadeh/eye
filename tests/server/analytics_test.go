package test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func newAnalyticsTestDB(t *testing.T) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&model.Event{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func mustInsertEvent(t *testing.T, db *gorm.DB, id, session, tenant, name string, ts int64) {
	t.Helper()
	e := model.Event{EventID: id, SessionID: session, TenantID: tenant, EventName: name, ClientTS: ts}
	if err := db.Create(&e).Error; err != nil {
		t.Fatalf("insert event: %v", err)
	}
}

func TestFunnelInOrderCompletion(t *testing.T) {
	db := newAnalyticsTestDB(t)
	svc := &service.AnalyticsService{DB: db}

	// session A completes all 3 steps in order
	mustInsertEvent(t, db, "a1", "sessA", "t1", "page_view", 100)
	mustInsertEvent(t, db, "a2", "sessA", "t1", "signup", 200)
	mustInsertEvent(t, db, "a3", "sessA", "t1", "purchase", 300)

	// session B only does page_view then signup (no purchase)
	mustInsertEvent(t, db, "b1", "sessB", "t1", "page_view", 100)
	mustInsertEvent(t, db, "b2", "sessB", "t1", "signup", 200)

	// session C does signup before page_view -- out of order, shouldn't count signup
	mustInsertEvent(t, db, "c1", "sessC", "t1", "signup", 50)
	mustInsertEvent(t, db, "c2", "sessC", "t1", "page_view", 100)

	results, err := svc.Funnel("t1", []string{"page_view", "signup", "purchase"}, nil, nil)
	if err != nil {
		t.Fatalf("funnel: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(results))
	}

	// page_view: sessA, sessB, sessC all reach it = 3
	if results[0].Sessions != 3 {
		t.Fatalf("step 0 (page_view): expected 3 sessions, got %d", results[0].Sessions)
	}
	// signup: sessA and sessB reach it in order; sessC's signup was before
	// its page_view so it doesn't count (out-of-order) = 2
	if results[1].Sessions != 2 {
		t.Fatalf("step 1 (signup): expected 2 sessions, got %d", results[1].Sessions)
	}
	// purchase: only sessA = 1
	if results[2].Sessions != 1 {
		t.Fatalf("step 2 (purchase): expected 1 session, got %d", results[2].Sessions)
	}

	if results[0].ConversionRate != 1.0 {
		t.Fatalf("step 0 conversion rate should be 1.0, got %v", results[0].ConversionRate)
	}
	wantRate1 := 2.0 / 3.0
	if results[1].ConversionRate != wantRate1 {
		t.Fatalf("step 1 conversion rate: expected %v, got %v", wantRate1, results[1].ConversionRate)
	}
	wantRate2 := 1.0 / 2.0
	if results[2].ConversionRate != wantRate2 {
		t.Fatalf("step 2 conversion rate: expected %v, got %v", wantRate2, results[2].ConversionRate)
	}
}

func TestFunnelSkippedStepDoesNotCount(t *testing.T) {
	db := newAnalyticsTestDB(t)
	svc := &service.AnalyticsService{DB: db}

	// session skips "signup" entirely, going straight from page_view to purchase
	mustInsertEvent(t, db, "a1", "sessA", "t1", "page_view", 100)
	mustInsertEvent(t, db, "a2", "sessA", "t1", "purchase", 300)

	results, err := svc.Funnel("t1", []string{"page_view", "signup", "purchase"}, nil, nil)
	if err != nil {
		t.Fatalf("funnel: %v", err)
	}
	if results[0].Sessions != 1 {
		t.Fatalf("page_view: expected 1, got %d", results[0].Sessions)
	}
	if results[1].Sessions != 0 {
		t.Fatalf("signup: expected 0 (skipped), got %d", results[1].Sessions)
	}
	if results[2].Sessions != 0 {
		t.Fatalf("purchase: expected 0 (never reached because signup step wasn't satisfied), got %d", results[2].Sessions)
	}
}

func TestFunnelScopedToTenant(t *testing.T) {
	db := newAnalyticsTestDB(t)
	svc := &service.AnalyticsService{DB: db}

	mustInsertEvent(t, db, "a1", "sessA", "t1", "page_view", 100)
	mustInsertEvent(t, db, "x1", "sessX", "t2", "page_view", 100)

	results, err := svc.Funnel("t1", []string{"page_view"}, nil, nil)
	if err != nil {
		t.Fatalf("funnel: %v", err)
	}
	if results[0].Sessions != 1 {
		t.Fatalf("expected only t1's session counted, got %d", results[0].Sessions)
	}
}

func TestRetentionDailyCohorts(t *testing.T) {
	db := newAnalyticsTestDB(t)
	svc := &service.AnalyticsService{DB: db}

	day0 := int64(1000) * daySecondsForTest()
	day1 := day0 + daySecondsForTest()
	day2 := day0 + 2*daySecondsForTest()

	// cohort day0: sessions s1, s2, s3 first seen on day0
	mustInsertEvent(t, db, "e1", "s1", "t1", "page_view", day0)
	mustInsertEvent(t, db, "e2", "s2", "t1", "page_view", day0)
	mustInsertEvent(t, db, "e3", "s3", "t1", "page_view", day0)

	// day1: s1 and s2 come back (retained), s3 doesn't
	mustInsertEvent(t, db, "e4", "s1", "t1", "page_view", day1)
	mustInsertEvent(t, db, "e5", "s2", "t1", "page_view", day1)

	// day2: only s1 comes back
	mustInsertEvent(t, db, "e6", "s1", "t1", "page_view", day2)

	rows, err := svc.Retention("t1", "daily")
	if err != nil {
		t.Fatalf("retention: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 cohort (day0), got %d: %+v", len(rows), rows)
	}
	row := rows[0]
	if row.CohortSize != 3 {
		t.Fatalf("expected cohort size 3, got %d", row.CohortSize)
	}
	if len(row.ActiveByN) != 3 {
		t.Fatalf("expected 3 periods (N=0,1,2), got %d: %+v", len(row.ActiveByN), row.ActiveByN)
	}
	if row.ActiveByN[0] != 3 {
		t.Fatalf("N=0 (cohort day itself): expected 3 active, got %d", row.ActiveByN[0])
	}
	if row.ActiveByN[1] != 2 {
		t.Fatalf("N=1: expected 2 active (s1,s2), got %d", row.ActiveByN[1])
	}
	if row.ActiveByN[2] != 1 {
		t.Fatalf("N=2: expected 1 active (s1), got %d", row.ActiveByN[2])
	}
	wantPct1 := 2.0 / 3.0
	if row.RetainedPct[1] != wantPct1 {
		t.Fatalf("N=1 retained pct: expected %v, got %v", wantPct1, row.RetainedPct[1])
	}
}

func TestEventsPerSessionIsTenantScopedAndOrdered(t *testing.T) {
	db := newAnalyticsTestDB(t)
	mustInsertEvent(t, db, "e1", "s1", "t1", "click", 1)
	mustInsertEvent(t, db, "e2", "s1", "t1", "purchase", 2)
	mustInsertEvent(t, db, "e3", "s2", "t1", "click", 3)
	mustInsertEvent(t, db, "e4", "other", "t2", "click", 4)

	rows, err := (&service.AnalyticsService{DB: db}).EventsPerSession("t1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].SessionID != "s1" || rows[0].Count != 2 || rows[1].Count != 1 {
		t.Fatalf("unexpected session aggregation: %+v", rows)
	}
}

func TestDeviceBreakdownIsTenantScoped(t *testing.T) {
	db := newAnalyticsTestDB(t)
	db.AutoMigrate(&model.Session{})
	db.Create(&model.Session{ID: "s1", TenantID: "t1", DeviceInfo: `{"platform":"web"}`})
	db.Create(&model.Session{ID: "s2", TenantID: "t1", DeviceInfo: `{"platform":"ios"}`})
	db.Create(&model.Session{ID: "s3", TenantID: "t2", DeviceInfo: `{"platform":"web"}`})

	rows, err := (&service.AnalyticsService{DB: db}).DeviceBreakdown("t1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Platform != "ios" || rows[0].Sessions != 1 || rows[1].Platform != "web" {
		t.Fatalf("unexpected device breakdown: %+v", rows)
	}
}

func TestTopNUsesMaterializedCountsAndTenantScope(t *testing.T) {
	db := newAnalyticsTestDB(t)
	db.AutoMigrate(&model.AggregateCount{})
	db.Create(&model.AggregateCount{TenantID: "t1", EventName: "click", DimsKey: "web", Count: 9})
	db.Create(&model.AggregateCount{TenantID: "t1", EventName: "click", DimsKey: "ios", Count: 4})
	db.Create(&model.AggregateCount{TenantID: "t2", EventName: "click", DimsKey: "secret", Count: 100})

	rows, err := (&service.AnalyticsService{DB: db}).TopN("t1", "click", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].DimsKey != "web" || rows[0].Count != 9 {
		t.Fatalf("unexpected top-n rows: %+v", rows)
	}
}

func TestSlidingWindowCountsGroupsEventsByWindow(t *testing.T) {
	db := newAnalyticsTestDB(t)
	mustInsertEvent(t, db, "w1", "s1", "t1", "click", 1000)
	mustInsertEvent(t, db, "w2", "s1", "t1", "click", 59000)
	mustInsertEvent(t, db, "w3", "s1", "t1", "click", 61000)
	mustInsertEvent(t, db, "w4", "s1", "t2", "click", 1000)

	rows, err := (&service.AnalyticsService{DB: db}).SlidingWindowCounts("t1", "click", 60, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Count != 2 || rows[1].Count != 1 {
		t.Fatalf("unexpected sliding-window rows: %+v", rows)
	}
}

// daySecondsForTest returns the millisecond value of one day, matching the
// unit AnalyticsService expects ClientTS to be in (milliseconds), since the
// service's dayIndex divides by 1000 before dividing by day-in-seconds.
func daySecondsForTest() int64 { return 86400 * 1000 }
