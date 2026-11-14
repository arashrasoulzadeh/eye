package handler

import (
	"context"
	"log/slog"
	"slices"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

func attachTags(ctx context.Context, svc *service.UserIntelligenceService, kind string, refs []service.IntelligenceTagRef, apply func(int, []string)) {
	if svc == nil {
		return
	}
	offset := 0
	for batch := range slices.Chunk(refs, 400) {
		tags, err := svc.IntelligenceTags(ctx, kind, batch)
		if err != nil {
			slog.Warn("intelligence tags unavailable", "kind", kind, "err", err)
			return
		}
		for i, values := range tags {
			apply(offset+i, values)
		}
		offset += len(batch)
	}
}

func attachSessionTags(ctx context.Context, svc *service.UserIntelligenceService, rows []model.Session, includeUsers bool) {
	if svc == nil {
		return
	}
	refs := make([]service.IntelligenceTagRef, len(rows))
	for i, row := range rows {
		refs[i] = service.IntelligenceTagRef{TenantID: row.TenantID, ID: row.ID}
	}
	attachTags(ctx, svc, "session", refs, func(i int, tags []string) { rows[i].IntelligenceTags = tags })
	if includeUsers {
		attachTags(ctx, svc, "user", refs, func(i int, tags []string) { rows[i].UserIntelligenceTags = tags })
	}
}

func attachEventTags(ctx context.Context, svc *service.UserIntelligenceService, rows []model.Event) []model.Event {
	if svc == nil {
		return rows
	}
	// QueryService can return a cached slice shared by concurrent callers.
	rows = slices.Clone(rows)
	refs := make([]service.IntelligenceTagRef, len(rows))
	for i, row := range rows {
		refs[i] = service.IntelligenceTagRef{TenantID: row.TenantID, ID: row.EventID}
	}
	attachTags(ctx, svc, "event", refs, func(i int, tags []string) { rows[i].IntelligenceTags = tags })
	return rows
}
