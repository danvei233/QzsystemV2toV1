package domain

import "time"

type RequestLog struct {
	ID                uint      `json:"id"`
	TraceID           string    `json:"trace_id"`
	ClientIP          string    `json:"client_ip"`
	DownstreamMethod  string    `json:"downstream_method"`
	DownstreamPath    string    `json:"downstream_path"`
	DownstreamHeaders string    `json:"downstream_headers"`
	DownstreamBody    string    `json:"downstream_body"`
	UpstreamMethod    string    `json:"upstream_method"`
	UpstreamURL       string    `json:"upstream_url"`
	UpstreamHeaders   string    `json:"upstream_headers"`
	UpstreamBody      string    `json:"upstream_body"`
	ResponseStatus    int       `json:"response_status"`
	ResponseHeaders   string    `json:"response_headers"`
	ResponseBody      string    `json:"response_body"`
	Success           bool      `json:"success"`
	CacheHit          bool      `json:"cache_hit"`
	Message           string    `json:"message"`
	DurationMillis    int64     `json:"duration_ms"`
	CreatedAt         time.Time `json:"created_at"`
}

type RequestLogFilter struct {
	Keyword string
	Path    string
	Success *bool
	Limit   int
	Offset  int
}

type LogRetentionPolicy struct {
	RetentionDays int
	MaxSizeBytes  int64
}
