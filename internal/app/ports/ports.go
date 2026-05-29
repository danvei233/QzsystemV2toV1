package ports

import (
	"context"
	"time"

	"xiaoheiproxy/internal/domain"
)

type RequestLogRepository interface {
	Create(ctx context.Context, log *domain.RequestLog) error
	List(ctx context.Context, filter domain.RequestLogFilter) ([]domain.RequestLog, int64, error)
	Get(ctx context.Context, id uint) (*domain.RequestLog, error)
	EndpointErrorStats(ctx context.Context, filter domain.RequestLogFilter, limit int) ([]domain.EndpointErrorStat, error)
}

type HostV2MetadataRepository interface {
	Upsert(ctx context.Context, metadata *domain.HostV2Metadata) error
	Get(ctx context.Context, hostID uint) (*domain.HostV2Metadata, error)
	Delete(ctx context.Context, hostID uint) error
}

type LogRetentionStore interface {
	SetRetention(policy domain.LogRetentionPolicy) error
}

type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration)
	DeletePrefix(prefix string)
}
