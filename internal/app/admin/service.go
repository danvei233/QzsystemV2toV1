package admin

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"xiaoheiproxy/internal/app/ports"
	"xiaoheiproxy/internal/domain"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

type Service struct {
	username string
	password string
	ttl      time.Duration
	logs     ports.RequestLogRepository
	mu       sync.RWMutex
	tokens   map[string]time.Time
}

func NewService(username, password string, ttl time.Duration, logs ports.RequestLogRepository) *Service {
	return &Service{
		username: username,
		password: password,
		ttl:      ttl,
		logs:     logs,
		tokens:   map[string]time.Time{},
	}
}

func (s *Service) Login(username, password string) (string, time.Time, error) {
	s.mu.RLock()
	currentUsername := s.username
	currentPassword := s.password
	s.mu.RUnlock()
	if subtle.ConstantTimeCompare([]byte(username), []byte(currentUsername)) != 1 ||
		subtle.ConstantTimeCompare([]byte(password), []byte(currentPassword)) != 1 {
		return "", time.Time{}, ErrInvalidCredentials
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", time.Time{}, err
	}
	token := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().Add(s.ttl)
	s.mu.Lock()
	s.tokens[token] = expiresAt
	s.mu.Unlock()
	return token, expiresAt, nil
}

func (s *Service) UpdatePassword(password string) {
	s.mu.Lock()
	s.password = password
	s.tokens = map[string]time.Time{}
	s.mu.Unlock()
}

func (s *Service) ChangePassword(currentPassword, newPassword string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if subtle.ConstantTimeCompare([]byte(currentPassword), []byte(s.password)) != 1 {
		return ErrInvalidCredentials
	}
	if len(newPassword) < 8 {
		return errors.New("new password must be at least 8 characters")
	}
	s.password = newPassword
	return nil
}

func (s *Service) SetCredentials(username, password string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.username = username
	s.password = password
}

func (s *Service) Validate(token string) bool {
	if token == "" {
		return false
	}
	s.mu.RLock()
	expiresAt, ok := s.tokens[token]
	s.mu.RUnlock()
	if !ok || time.Now().After(expiresAt) {
		if ok {
			s.mu.Lock()
			delete(s.tokens, token)
			s.mu.Unlock()
		}
		return false
	}
	return true
}

func (s *Service) Logout(token string) {
	s.mu.Lock()
	delete(s.tokens, token)
	s.mu.Unlock()
}

func (s *Service) ListLogs(ctx context.Context, filter domain.RequestLogFilter) ([]domain.RequestLog, int64, error) {
	return s.logs.List(ctx, filter)
}

func (s *Service) GetLog(ctx context.Context, id uint) (*domain.RequestLog, error) {
	return s.logs.Get(ctx, id)
}

func (s *Service) UpdateLogRetention(policy domain.LogRetentionPolicy) error {
	store, ok := s.logs.(ports.LogRetentionStore)
	if !ok {
		return nil
	}
	return store.SetRetention(policy)
}
