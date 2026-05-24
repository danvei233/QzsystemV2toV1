//go:build sqlite

package repository

import (
	"xiaoheiproxy/internal/app/ports"
	"xiaoheiproxy/internal/domain"
)

func OpenRequestLogRepository(dsn string, retention domain.LogRetentionPolicy) (ports.RequestLogRepository, error) {
	db, err := OpenSQLite(dsn)
	if err != nil {
		return nil, err
	}
	return NewRequestLogRepository(db, dsn, retention), nil
}
