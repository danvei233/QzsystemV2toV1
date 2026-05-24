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
}

type LogRetentionStore interface {
	SetRetention(policy domain.LogRetentionPolicy) error
}

type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration)
	DeletePrefix(prefix string)
}
