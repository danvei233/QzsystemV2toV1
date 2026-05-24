package cache

import (
	"strings"
	"sync"
	"time"
)

type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]entry
}

type entry struct {
	value     []byte
	expiresAt time.Time
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{items: map[string]entry{}}
}

func (c *MemoryCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(item.expiresAt) {
		if ok {
			c.mu.Lock()
			delete(c.items, key)
			c.mu.Unlock()
		}
		return nil, false
	}
	out := make([]byte, len(item.value))
	copy(out, item.value)
	return out, true
}

func (c *MemoryCache) Set(key string, value []byte, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	out := make([]byte, len(value))
	copy(out, value)
	c.mu.Lock()
	c.items[key] = entry{value: out, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
}

func (c *MemoryCache) DeletePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.items {
		if strings.HasPrefix(key, prefix) {
			delete(c.items, key)
		}
	}
}
