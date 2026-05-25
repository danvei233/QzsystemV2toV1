package domain

import "time"

type RequestLog struct {
	ID                  uint      `json:"id"`
	TraceID             string    `json:"trace_id"`
	ClientIP            string    `json:"client_ip"`
	DownstreamMethod    string    `json:"downstream_method"`
	DownstreamPath      string    `json:"downstream_path"`
	DownstreamHeaders   string    `json:"downstream_headers"`
	DownstreamBody      string    `json:"downstream_body"`
	UpstreamMethod      string    `json:"upstream_method"`
	UpstreamURL         string    `json:"upstream_url"`
	UpstreamHeaders     string    `json:"upstream_headers"`
	UpstreamBody        string    `json:"upstream_body"`
	UpstreamRespStatus  int       `json:"upstream_response_status"`
	UpstreamRespHeaders string    `json:"upstream_response_headers"`
	UpstreamRespBody    string    `json:"upstream_response_body"`
	ResponseStatus      int       `json:"response_status"`
	ResponseHeaders     string    `json:"response_headers"`
	ResponseBody        string    `json:"response_body"`
	Success             bool      `json:"success"`
	CacheHit            bool      `json:"cache_hit"`
	Message             string    `json:"message"`
	DurationMillis      int64     `json:"duration_ms"`
	CreatedAt           time.Time `json:"created_at"`
}

type RequestLogFilter struct {
	Keyword string
	Path    string
	Success *bool
	StartAt *time.Time
	EndAt   *time.Time
	Limit   int
	Offset  int
}

type EndpointErrorStat struct {
	Path              string    `json:"path"`
	Total             int64     `json:"total"`
	Success           int64     `json:"success"`
	Failed            int64     `json:"failed"`
	ErrorRate         float64   `json:"error_rate"`
	AvgDurationMillis int64     `json:"avg_duration_ms"`
	LastMessage       string    `json:"last_message"`
	LastSeenAt        time.Time `json:"last_seen_at"`
}

type LogRetentionPolicy struct {
	RetentionDays int
	MaxSizeBytes  int64
}
