package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

const globalModelReleaseAPI = "https://api.github.com/repos/arashrasoulzadeh/eye/releases?per_page=100"

func loadGlobalBundle(ctx context.Context, client *http.Client) (GlobalIntelligenceBundle, error) {
	var bundle GlobalIntelligenceBundle
	get := func(url string, limit int64) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("model release request: HTTP %d", resp.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
		if int64(len(data)) > limit {
			return nil, fmt.Errorf("model release response exceeds size limit")
		}
		return data, err
	}
	data, err := get(globalModelReleaseAPI, 8<<20)
	if err != nil {
		return bundle, err
	}
	var releases []struct {
		Tag        string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name   string `json:"name"`
			URL    string `json:"browser_download_url"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(data, &releases); err != nil {
		return bundle, err
	}
	for _, release := range releases {
		if release.Draft || release.Prerelease {
			continue
		}
		for _, asset := range release.Assets {
			if asset.Name != "eye-global-model.json" {
				continue
			}
			data, err := get(asset.URL, 8<<20)
			if err != nil {
				return bundle, err
			}
			if asset.Digest != fmt.Sprintf("sha256:%x", sha256.Sum256(data)) {
				return bundle, fmt.Errorf("model release digest mismatch")
			}
			if err := json.Unmarshal(data, &bundle); err != nil {
				return bundle, err
			}
			if bundle.Schema != 1 || bundle.Version != release.Tag || len(bundle.Next) == 0 {
				return bundle, fmt.Errorf("unsupported global model release %q", release.Tag)
			}
			return bundle, nil
		}
	}
	return bundle, nil
}

func RunGlobalModelJob(ctx context.Context, svc *UserIntelligenceService, logger *slog.Logger) {
	client := &http.Client{Timeout: 30 * time.Second}
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	var active GlobalIntelligenceBundle
	var stored model.GlobalIntelligenceRelease
	if err := svc.DB.WithContext(ctx).Find(&stored, 1).Error; err == nil && stored.Bundle != "" {
		if err := json.Unmarshal([]byte(stored.Bundle), &active); err != nil {
			active = GlobalIntelligenceBundle{}
			logger.Warn("cached global model is invalid", "err", err)
		}
	}
	for {
		candidate, err := loadGlobalBundle(ctx, client)
		if err != nil {
			logger.Warn("global model download failed", "err", err)
		} else if candidate.Version != "" && candidate.Version != active.Version {
			if err := svc.DB.WithContext(ctx).Save(&model.GlobalIntelligenceRelease{ID: 1, Bundle: inferenceJSON(candidate)}).Error; err != nil {
				logger.Error("global model cache update failed", "err", err)
			} else {
				active = candidate
			}
		}
		if active.Version != "" {
			if err := svc.ScoreGlobalModel(ctx, active, time.Now().UTC()); err != nil {
				logger.Error("global model inference failed", "err", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
