package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

const globalModelReleaseAPI = "https://api.github.com/repos/arashrasoulzadeh/eye/releases?per_page=100"

type GlobalModelRelease struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	URL       string    `json:"browser_download_url"`
	Digest    string    `json:"digest"`
}

type GlobalModelStatus struct {
	Installed string              `json:"installed"`
	Latest    *GlobalModelRelease `json:"latest"`
}

type GlobalModelUpdates struct {
	wake    chan struct{}
	client  *http.Client
	mu      sync.Mutex
	latest  *GlobalModelRelease
	checked time.Time
}

func NewGlobalModelUpdates() *GlobalModelUpdates {
	return &GlobalModelUpdates{wake: make(chan struct{}, 1), client: &http.Client{Timeout: 30 * time.Second}}
}

func (u *GlobalModelUpdates) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model release request: HTTP %d", resp.StatusCode)
	}
	const limit = 8 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if len(data) > limit {
		return nil, fmt.Errorf("model release response exceeds size limit")
	}
	return data, err
}

// latestRelease runs under mu so concurrent checks share one GitHub request.
func (u *GlobalModelUpdates) latestRelease(ctx context.Context) (*GlobalModelRelease, error) {
	if time.Since(u.checked) < time.Minute {
		return u.latest, nil
	}
	data, err := u.get(ctx, globalModelReleaseAPI)
	if err != nil {
		return nil, err
	}
	var releases []struct {
		Tag        string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name string `json:"name"`
			GlobalModelRelease
		} `json:"assets"`
	}
	if err = json.Unmarshal(data, &releases); err != nil {
		return nil, err
	}
	var latest *GlobalModelRelease
	for _, release := range releases {
		if release.Draft || release.Prerelease {
			continue
		}
		for _, asset := range release.Assets {
			if asset.Name != "eye-global-model.json" {
				continue
			}
			if latest != nil && !asset.CreatedAt.After(latest.CreatedAt) {
				continue
			}
			item := asset.GlobalModelRelease
			item.Version = release.Tag
			latest = &item
		}
	}
	u.latest, u.checked = latest, time.Now()
	return latest, nil
}

func (u *GlobalModelUpdates) Status(ctx context.Context, svc *UserIntelligenceService) (GlobalModelStatus, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	active, err := cachedGlobalBundle(ctx, svc)
	if err != nil {
		return GlobalModelStatus{}, err
	}
	release, err := u.latestRelease(ctx)
	return GlobalModelStatus{Installed: active.Version, Latest: release}, err
}

func cachedGlobalBundle(ctx context.Context, svc *UserIntelligenceService) (GlobalIntelligenceBundle, error) {
	var active GlobalIntelligenceBundle
	var stored model.GlobalIntelligenceRelease
	err := svc.DB.WithContext(ctx).Find(&stored, 1).Error
	if err == nil && stored.Bundle != "" {
		err = json.Unmarshal([]byte(stored.Bundle), &active)
	}
	return active, err
}

func (u *GlobalModelUpdates) Install(ctx context.Context, svc *UserIntelligenceService) (GlobalModelStatus, error) {
	status, err := u.download(ctx, svc)
	if err == nil && status.Installed != "" {
		select {
		case u.wake <- struct{}{}:
		default:
		}
	}
	return status, err
}

func (u *GlobalModelUpdates) download(ctx context.Context, svc *UserIntelligenceService) (GlobalModelStatus, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	active, err := cachedGlobalBundle(ctx, svc)
	if err != nil {
		return GlobalModelStatus{}, err
	}
	release, err := u.latestRelease(ctx)
	status := GlobalModelStatus{Installed: active.Version, Latest: release}
	if err != nil || release == nil || release.Version == active.Version {
		return status, err
	}
	data, err := u.get(ctx, release.URL)
	if err != nil {
		return status, err
	}
	if release.Digest != fmt.Sprintf("sha256:%x", sha256.Sum256(data)) {
		return status, fmt.Errorf("model release digest mismatch")
	}
	var bundle GlobalIntelligenceBundle
	if err = json.Unmarshal(data, &bundle); err != nil {
		return status, err
	}
	if bundle.Schema != 1 || bundle.Version != release.Version || len(bundle.Next) == 0 {
		return status, fmt.Errorf("unsupported global model release %q", release.Version)
	}
	if err = svc.DB.WithContext(ctx).Save(&model.GlobalIntelligenceRelease{ID: 1, Bundle: string(data)}).Error; err != nil {
		return status, err
	}
	status.Installed = bundle.Version
	return status, nil
}

func RunGlobalModelJob(ctx context.Context, svc *UserIntelligenceService, updates *GlobalModelUpdates, logger *slog.Logger) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		if _, err := updates.download(ctx, svc); err != nil {
			logger.Warn("global model download failed", "err", err)
		}
		active, err := cachedGlobalBundle(ctx, svc)
		if err != nil {
			logger.Error("cached global model unavailable", "err", err)
		} else if active.Version != "" {
			if err := svc.ScoreGlobalModel(ctx, active, time.Now().UTC()); err != nil {
				logger.Error("global model inference failed", "err", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-updates.wake:
		}
	}
}
