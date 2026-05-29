package repository

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
	"xiaoheiproxy/internal/app/ports"
	"xiaoheiproxy/internal/domain"
)

func OpenRequestLogRepository(dsn string, retention domain.LogRetentionPolicy) (ports.RequestLogRepository, error) {
	source := strings.TrimSpace(dsn)
	normalized := normalizeDSN(source)
	db, err := OpenSQLite(normalized)
	if err != nil {
		return nil, err
	}
	if err := migrateJSONLToSQLite(db, source, retention); err != nil {
		return nil, err
	}
	return NewRequestLogRepository(db, normalized, retention), nil
}

func OpenGatewayRepositories(dsn string, retention domain.LogRetentionPolicy) (ports.RequestLogRepository, ports.HostV2MetadataRepository, error) {
	source := strings.TrimSpace(dsn)
	normalized := normalizeDSN(source)
	db, err := OpenSQLite(normalized)
	if err != nil {
		return nil, nil, err
	}
	if err := migrateJSONLToSQLite(db, source, retention); err != nil {
		return nil, nil, err
	}
	return NewRequestLogRepository(db, normalized, retention), NewHostV2MetadataRepository(db), nil
}

func normalizeDSN(dsn string) string {
	value := strings.TrimSpace(dsn)
	if value == "" {
		return "data/request_logs.db"
	}
	lower := strings.ToLower(value)
	if strings.HasSuffix(lower, ".jsonl") {
		base := strings.TrimSuffix(value, filepath.Ext(value))
		return base + ".db"
	}
	return value
}

func migrateJSONLToSQLite(db *gorm.DB, source string, retention domain.LogRetentionPolicy) error {
	source = strings.TrimSpace(source)
	if !strings.HasSuffix(strings.ToLower(source), ".jsonl") {
		return nil
	}
	if _, err := os.Stat(source); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var count int64
	if err := db.Model(&RequestLogRow{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	reader := &FileRequestLogRepository{path: source, retention: normalizeRetention(retention)}
	items, err := reader.readAll()
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	rows := make([]RequestLogRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, toRow(item))
	}
	return db.Transaction(func(tx *gorm.DB) error {
		return tx.CreateInBatches(rows, 200).Error
	})
}
