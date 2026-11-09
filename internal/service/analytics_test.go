package service

import (
	"testing"
	"time"
)

func TestAnalyticsHelpers(t *testing.T) {
	tests := []struct {
		name     string
		tsMillis int64
		wantDay  int64
		wantWeek int64
		wantDayStr string
		wantWeekStr string
	}{
		{
			name:     "epoch",
			tsMillis: 0,
			wantDay:  0,
			wantWeek: 0,
			wantDayStr:  "1970-01-01",
			wantWeekStr: "1970-W01",
		},
		{
			name:     "known date",
			tsMillis: 1705315200000, // 2024-01-15 12:00:00 UTC
			wantDay:  19737,
			wantWeek: 2819,
			wantDayStr:  "2024-01-15",
			wantWeekStr: "2024-W03",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotDay := dayIndex(tc.tsMillis)
			if gotDay != tc.wantDay {
				t.Errorf("dayIndex(%d) = %d, want %d", tc.tsMillis, gotDay, tc.wantDay)
			}

			gotWeek := weekIndex(tc.tsMillis)
			if gotWeek != tc.wantWeek {
				t.Errorf("weekIndex(%d) = %d, want %d", tc.tsMillis, gotWeek, tc.wantWeek)
			}

			gotDayStr := dayPeriod(tc.tsMillis)
			if gotDayStr != tc.wantDayStr {
				t.Errorf("dayPeriod(%d) = %s, want %s", tc.tsMillis, gotDayStr, tc.wantDayStr)
			}

			gotWeekStr := weekPeriod(tc.tsMillis)
			if gotWeekStr != tc.wantWeekStr {
				t.Errorf("weekPeriod(%d) = %s, want %s", tc.tsMillis, gotWeekStr, tc.wantWeekStr)
			}
		})
	}
}

func TestTimeFromMillis(t *testing.T) {
	ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	tm := timeFromMillis(ts)
	if tm.Year() != 2024 || tm.Month() != 1 || tm.Day() != 15 {
		t.Errorf("timeFromMillis = %v, want 2024-01-15", tm)
	}
}