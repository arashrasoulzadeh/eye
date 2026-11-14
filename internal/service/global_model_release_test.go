package service

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type modelReleaseTransport func(*http.Request) (*http.Response, error)

func (f modelReleaseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGlobalModelStatusAndInstall(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.GlobalIntelligenceRelease{}); err != nil {
		t.Fatal(err)
	}
	svc := &UserIntelligenceService{DB: db}
	updates := NewGlobalModelUpdates()
	const bundle = `{"schema":1,"version":"model-v1","next":{"navigation":"engagement"}}`
	downloads := 0
	updates.client.Transport = modelReleaseTransport(func(r *http.Request) (*http.Response, error) {
		body := bundle
		if r.URL.String() == globalModelReleaseAPI {
			body = fmt.Sprintf(`[{"tag_name":"model-v1","assets":[{"name":"eye-global-model.json","created_at":"2026-11-14T12:00:00Z","browser_download_url":"https://example.test/model","digest":"sha256:%x"}]}]`, sha256.Sum256([]byte(bundle)))
		} else {
			downloads++
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	status, err := updates.Status(t.Context(), svc)
	if err != nil || status.Installed != "" || status.Latest == nil || status.Latest.CreatedAt.IsZero() || downloads != 0 {
		t.Fatalf("status=%+v downloads=%d err=%v", status, downloads, err)
	}
	status, err = updates.Install(t.Context(), svc)
	if err != nil || status.Installed != "model-v1" || downloads != 1 || len(updates.wake) != 1 {
		t.Fatalf("status=%+v downloads=%d err=%v", status, downloads, err)
	}
}
