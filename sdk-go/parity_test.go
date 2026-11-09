package eyesdk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fixtureCase mirrors tests/fixtures/sdk_ring_buffer_parity.json, the shared
// fixture also driven through the TS web-sdk in tests/web/parity.test.ts, so
// both EyeClient ports are asserted to evict/retain events identically for
// the same inputs.
type fixtureCase struct {
	Name              string   `json:"name"`
	MaxBufferSize     int      `json:"maxBufferSize"`
	BatchSize         int      `json:"batchSize"`
	TrackNames        []string `json:"trackNames"`
	ExpectedRemaining []string `json:"expectedRemaining"`
}

type fixtureFile struct {
	Cases []fixtureCase `json:"cases"`
}

func loadParityFixture(t *testing.T) fixtureFile {
	t.Helper()
	path := filepath.Join("..", "tests", "fixtures", "sdk_ring_buffer_parity.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var f fixtureFile
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return f
}

func TestRingBufferParityWithWebSDKFixture(t *testing.T) {
	fixture := loadParityFixture(t)
	if len(fixture.Cases) == 0 {
		t.Fatal("fixture has no cases")
	}

	for _, tc := range fixture.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			c := NewClient(Options{
				Endpoint:      "http://unused.invalid",
				TenantID:      "t1",
				MaxBufferSize: tc.MaxBufferSize,
				BatchSize:     tc.BatchSize,
				FlushInterval: time.Hour, // disable background timer flush for determinism
			})
			defer c.Close()
			c.SetSessionID("sess-1")

			for _, name := range tc.TrackNames {
				if err := c.Track(name, nil, nil); err != nil {
					t.Fatalf("track(%q): %v", name, err)
				}
			}

			buf := c.PeekBuffer()
			got := make([]string, len(buf))
			for i, ev := range buf {
				got[i] = ev.EventName
			}

			if len(got) != len(tc.ExpectedRemaining) {
				t.Fatalf("remaining = %v, want %v", got, tc.ExpectedRemaining)
			}
			for i := range got {
				if got[i] != tc.ExpectedRemaining[i] {
					t.Fatalf("remaining = %v, want %v", got, tc.ExpectedRemaining)
				}
			}
		})
	}
}
