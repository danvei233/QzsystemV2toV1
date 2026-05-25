package repository

import (
	"sort"
	"strings"

	"xiaoheiproxy/internal/domain"
)

type endpointStatAccumulator struct {
	stat        domain.EndpointErrorStat
	durationSum int64
}

func endpointErrorStats(items []domain.RequestLog, filter domain.RequestLogFilter, limit int) []domain.EndpointErrorStat {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	groups := map[string]*endpointStatAccumulator{}
	for _, item := range items {
		if !matchesStatsFilter(item, filter) {
			continue
		}
		path := strings.TrimSpace(item.DownstreamPath)
		if path == "" {
			path = "(unknown)"
		}
		acc := groups[path]
		if acc == nil {
			acc = &endpointStatAccumulator{stat: domain.EndpointErrorStat{Path: path}}
			groups[path] = acc
		}
		acc.stat.Total++
		if item.Success {
			acc.stat.Success++
		} else {
			acc.stat.Failed++
		}
		acc.durationSum += item.DurationMillis
		if item.CreatedAt.After(acc.stat.LastSeenAt) {
			acc.stat.LastSeenAt = item.CreatedAt
			acc.stat.LastMessage = item.Message
		}
	}
	out := make([]domain.EndpointErrorStat, 0, len(groups))
	for _, acc := range groups {
		if acc.stat.Total > 0 {
			acc.stat.ErrorRate = float64(acc.stat.Failed) / float64(acc.stat.Total) * 100
			acc.stat.AvgDurationMillis = acc.durationSum / acc.stat.Total
		}
		out = append(out, acc.stat)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ErrorRate != out[j].ErrorRate {
			return out[i].ErrorRate > out[j].ErrorRate
		}
		if out[i].Failed != out[j].Failed {
			return out[i].Failed > out[j].Failed
		}
		if out[i].Total != out[j].Total {
			return out[i].Total > out[j].Total
		}
		return out[i].Path < out[j].Path
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func matchesStatsFilter(item domain.RequestLog, filter domain.RequestLogFilter) bool {
	path := strings.TrimSpace(filter.Path)
	if path != "" && item.DownstreamPath != path {
		return false
	}
	if filter.StartAt != nil && item.CreatedAt.Before(*filter.StartAt) {
		return false
	}
	if filter.EndAt != nil && item.CreatedAt.After(*filter.EndAt) {
		return false
	}
	kw := strings.ToLower(strings.TrimSpace(filter.Keyword))
	if kw == "" {
		return true
	}
	return strings.Contains(strings.ToLower(item.TraceID+" "+item.DownstreamPath+" "+item.UpstreamURL+" "+item.Message), kw)
}
