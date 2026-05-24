//go:build sqlite

package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xiaoheiproxy/internal/domain"

	_ "github.com/ncruces/go-sqlite3/driver"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type RequestLogRow struct {
	ID                uint `gorm:"primaryKey"`
	TraceID           string
	ClientIP          string
	DownstreamMethod  string
	DownstreamPath    string `gorm:"index"`
	DownstreamHeaders string `gorm:"type:text"`
	DownstreamBody    string `gorm:"type:text"`
	UpstreamMethod    string
	UpstreamURL       string `gorm:"type:text"`
	UpstreamHeaders   string `gorm:"type:text"`
	UpstreamBody      string `gorm:"type:text"`
	ResponseStatus    int    `gorm:"index"`
	ResponseHeaders   string `gorm:"type:text"`
	ResponseBody      string `gorm:"type:text"`
	Success           bool   `gorm:"index"`
	CacheHit          bool
	Message           string `gorm:"type:text"`
	DurationMillis    int64
	CreatedAt         time.Time `gorm:"index"`
}

func (RequestLogRow) TableName() string { return "request_logs" }

type SQLiteRequestLogRepository struct {
	db        *gorm.DB
	dsn       string
	retention domain.LogRetentionPolicy
}

func OpenSQLite(dsn string) (*gorm.DB, error) {
	if dir := filepath.Dir(dsn); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite3", DSN: dsn}, &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&RequestLogRow{}); err != nil {
		return nil, err
	}
	return db, nil
}

func NewRequestLogRepository(db *gorm.DB, dsn string, retention domain.LogRetentionPolicy) *SQLiteRequestLogRepository {
	return &SQLiteRequestLogRepository{db: db, dsn: dsn, retention: normalizeRetention(retention)}
}

func (r *SQLiteRequestLogRepository) Create(ctx context.Context, log *domain.RequestLog) error {
	row := toRow(*log)
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now()
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	log.ID = row.ID
	log.CreatedAt = row.CreatedAt
	if err := r.compact(ctx); err != nil {
		return err
	}
	return nil
}

func (r *SQLiteRequestLogRepository) List(ctx context.Context, filter domain.RequestLogFilter) ([]domain.RequestLog, int64, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	q := r.db.WithContext(ctx).Model(&RequestLogRow{})
	if strings.TrimSpace(filter.Path) != "" {
		q = q.Where("downstream_path = ?", strings.TrimSpace(filter.Path))
	}
	if filter.Success != nil {
		q = q.Where("success = ?", *filter.Success)
	}
	if strings.TrimSpace(filter.Keyword) != "" {
		kw := "%" + strings.TrimSpace(filter.Keyword) + "%"
		q = q.Where("trace_id LIKE ? OR downstream_path LIKE ? OR upstream_url LIKE ? OR message LIKE ?", kw, kw, kw, kw)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []RequestLogRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.RequestLog, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRow(row))
	}
	return out, total, nil
}

func (r *SQLiteRequestLogRepository) Get(ctx context.Context, id uint) (*domain.RequestLog, error) {
	var row RequestLogRow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	log := fromRow(row)
	return &log, nil
}

func (r *SQLiteRequestLogRepository) SetRetention(policy domain.LogRetentionPolicy) error {
	r.retention = normalizeRetention(policy)
	return r.compact(context.Background())
}

func (r *SQLiteRequestLogRepository) compact(ctx context.Context) error {
	cutoff := time.Now().AddDate(0, 0, -r.retention.RetentionDays)
	if err := r.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&RequestLogRow{}).Error; err != nil {
		return err
	}
	if strings.HasPrefix(strings.TrimSpace(r.dsn), "file:") {
		return nil
	}
	for {
		info, err := os.Stat(r.dsn)
		if err != nil || info.Size() <= r.retention.MaxSizeBytes {
			return nil
		}
		var row RequestLogRow
		if err := r.db.WithContext(ctx).Order("id ASC").First(&row).Error; err != nil {
			return nil
		}
		if err := r.db.WithContext(ctx).Delete(&RequestLogRow{}, row.ID).Error; err != nil {
			return err
		}
	}
}

func toRow(log domain.RequestLog) RequestLogRow {
	return RequestLogRow{
		ID:                log.ID,
		TraceID:           log.TraceID,
		ClientIP:          log.ClientIP,
		DownstreamMethod:  log.DownstreamMethod,
		DownstreamPath:    log.DownstreamPath,
		DownstreamHeaders: log.DownstreamHeaders,
		DownstreamBody:    log.DownstreamBody,
		UpstreamMethod:    log.UpstreamMethod,
		UpstreamURL:       log.UpstreamURL,
		UpstreamHeaders:   log.UpstreamHeaders,
		UpstreamBody:      log.UpstreamBody,
		ResponseStatus:    log.ResponseStatus,
		ResponseHeaders:   log.ResponseHeaders,
		ResponseBody:      log.ResponseBody,
		Success:           log.Success,
		CacheHit:          log.CacheHit,
		Message:           log.Message,
		DurationMillis:    log.DurationMillis,
		CreatedAt:         log.CreatedAt,
	}
}

func fromRow(row RequestLogRow) domain.RequestLog {
	return domain.RequestLog{
		ID:                row.ID,
		TraceID:           row.TraceID,
		ClientIP:          row.ClientIP,
		DownstreamMethod:  row.DownstreamMethod,
		DownstreamPath:    row.DownstreamPath,
		DownstreamHeaders: row.DownstreamHeaders,
		DownstreamBody:    row.DownstreamBody,
		UpstreamMethod:    row.UpstreamMethod,
		UpstreamURL:       row.UpstreamURL,
		UpstreamHeaders:   row.UpstreamHeaders,
		UpstreamBody:      row.UpstreamBody,
		ResponseStatus:    row.ResponseStatus,
		ResponseHeaders:   row.ResponseHeaders,
		ResponseBody:      row.ResponseBody,
		Success:           row.Success,
		CacheHit:          row.CacheHit,
		Message:           row.Message,
		DurationMillis:    row.DurationMillis,
		CreatedAt:         row.CreatedAt,
	}
}
