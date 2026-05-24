package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"xiaoheiproxy/internal/domain"
)

type FileRequestLogRepository struct {
	mu        sync.Mutex
	path      string
	nextID    uint
	retention domain.LogRetentionPolicy
}

func NewFileRequestLogRepository(path string, retention domain.LogRetentionPolicy) (*FileRequestLogRepository, error) {
	if strings.TrimSpace(path) == "" {
		path = "data/request_logs.jsonl"
	}
	retention = normalizeRetention(retention)
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	repo := &FileRequestLogRepository{path: path, nextID: 1, retention: retention}
	items, err := repo.readAll()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	for _, item := range items {
		if item.ID >= repo.nextID {
			repo.nextID = item.ID + 1
		}
	}
	if err := repo.compactLocked(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return repo, nil
}

func (r *FileRequestLogRepository) Create(_ context.Context, log *domain.RequestLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	log.ID = r.nextID
	r.nextID++
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	b, err := json.Marshal(log)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return r.compactLocked()
}

func (r *FileRequestLogRepository) SetRetention(policy domain.LogRetentionPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retention = normalizeRetention(policy)
	return r.compactLocked()
}

func (r *FileRequestLogRepository) compactLocked() error {
	items, err := r.readAll()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return os.WriteFile(r.path, nil, 0o644)
		}
		return err
	}
	cutoff := time.Now().AddDate(0, 0, -r.retention.RetentionDays)
	kept := make([]domain.RequestLog, 0, len(items))
	for _, item := range items {
		if item.CreatedAt.IsZero() || item.CreatedAt.Before(cutoff) {
			continue
		}
		kept = append(kept, item)
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].ID < kept[j].ID })
	lines := make([][]byte, 0, len(kept))
	var total int64
	for i := len(kept) - 1; i >= 0; i-- {
		b, err := json.Marshal(kept[i])
		if err != nil {
			continue
		}
		lineSize := int64(len(b) + 1)
		if total+lineSize > r.retention.MaxSizeBytes {
			break
		}
		total += lineSize
		lines = append(lines, append(b, '\n'))
	}
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	tmp := r.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	for _, line := range lines {
		if _, err := f.Write(line); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, r.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (r *FileRequestLogRepository) List(_ context.Context, filter domain.RequestLogFilter) ([]domain.RequestLog, int64, error) {
	r.mu.Lock()
	items, err := r.readAll()
	r.mu.Unlock()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, 0, err
	}
	filtered := make([]domain.RequestLog, 0, len(items))
	kw := strings.ToLower(strings.TrimSpace(filter.Keyword))
	path := strings.TrimSpace(filter.Path)
	for _, item := range items {
		if path != "" && item.DownstreamPath != path {
			continue
		}
		if filter.Success != nil && item.Success != *filter.Success {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(item.TraceID+" "+item.DownstreamPath+" "+item.UpstreamURL+" "+item.Message), kw) {
			continue
		}
		filtered = append(filtered, item)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].ID > filtered[j].ID })
	total := int64(len(filtered))
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(filtered) {
		return []domain.RequestLog{}, total, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}

func (r *FileRequestLogRepository) Get(_ context.Context, id uint) (*domain.RequestLog, error) {
	r.mu.Lock()
	items, err := r.readAll()
	r.mu.Unlock()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.ID == id {
			copy := item
			return &copy, nil
		}
	}
	return nil, os.ErrNotExist
}

func (r *FileRequestLogRepository) readAll() ([]domain.RequestLog, error) {
	f, err := os.Open(r.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	var out []domain.RequestLog
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item domain.RequestLog
		if err := json.Unmarshal([]byte(line), &item); err == nil {
			out = append(out, item)
		}
	}
	return out, scanner.Err()
}
