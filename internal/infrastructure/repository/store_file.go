//go:build !sqlite

package repository

import (
	"xiaoheiproxy/internal/app/ports"
	"xiaoheiproxy/internal/domain"
)

func OpenRequestLogRepository(dsn string, retention domain.LogRetentionPolicy) (ports.RequestLogRepository, error) {
	return NewFileRequestLogRepository(dsn, retention)
}
