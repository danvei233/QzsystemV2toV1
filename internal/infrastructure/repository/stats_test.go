package repository

import (
	"testing"
	"time"

	"xiaoheiproxy/internal/domain"
)

func TestEndpointErrorStatsSortsByErrorRate(t *testing.T) {
	now := time.Now()
	items := []domain.RequestLog{
		{DownstreamPath: "/api/v1/a", Success: false, DurationMillis: 100, Message: "a failed", CreatedAt: now.Add(-4 * time.Minute)},
		{DownstreamPath: "/api/v1/a", Success: true, DurationMillis: 200, Message: "a ok", CreatedAt: now.Add(-3 * time.Minute)},
		{DownstreamPath: "/api/v1/b", Success: false, DurationMillis: 300, Message: "b failed", CreatedAt: now.Add(-2 * time.Minute)},
		{DownstreamPath: "/api/v1/b", Success: false, DurationMillis: 500, Message: "b failed again", CreatedAt: now.Add(-1 * time.Minute)},
	}

	stats := endpointErrorStats(items, domain.RequestLogFilter{}, 10)
	if len(stats) != 2 {
		t.Fatalf("expected 2 stats, got %d", len(stats))
	}
	if stats[0].Path != "/api/v1/b" {
		t.Fatalf("expected /api/v1/b first, got %s", stats[0].Path)
	}
	if stats[0].ErrorRate != 100 {
		t.Fatalf("expected 100 error rate, got %.2f", stats[0].ErrorRate)
	}
	if stats[0].AvgDurationMillis != 400 {
		t.Fatalf("expected avg duration 400, got %d", stats[0].AvgDurationMillis)
	}
	if stats[0].LastMessage != "b failed again" {
		t.Fatalf("expected latest message, got %q", stats[0].LastMessage)
	}
}

func TestEndpointErrorStatsAppliesTimeAndPathFilters(t *testing.T) {
	now := time.Now()
	start := now.Add(-2 * time.Minute)
	items := []domain.RequestLog{
		{DownstreamPath: "/api/v1/a", Success: false, CreatedAt: now.Add(-5 * time.Minute)},
		{DownstreamPath: "/api/v1/a", Success: true, CreatedAt: now.Add(-1 * time.Minute)},
		{DownstreamPath: "/api/v1/b", Success: false, CreatedAt: now.Add(-1 * time.Minute)},
	}

	stats := endpointErrorStats(items, domain.RequestLogFilter{
		Path:    "/api/v1/a",
		StartAt: &start,
	}, 10)
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(stats))
	}
	if stats[0].Path != "/api/v1/a" || stats[0].Total != 1 || stats[0].Failed != 0 {
		t.Fatalf("unexpected stat: %+v", stats[0])
	}
}
