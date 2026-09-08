// Package cache provides a caching layer for GoKS with in-memory and Redis support.
package cache

import (
	"sync"
	"time"
)

// Cache is the GoKS caching interface.
type Cache interface {
	// Get retrieves a cached value. Returns (value, true) on hit, ("", false) on miss.
	Get(key string) (any, bool)
	// Set stores a value with a TTL. TTL=0 means no expiry.
	Set(key string, value any, ttl time.Duration)
	// Delete removes a key.
	Delete(key string)
	// Flush clears all cached entries.
	Flush()
	// Remember retrieves a cached value, or calls fn to compute it and cache it.
	Remember(key string, ttl time.Duration, fn func() any) any
}

// -----------------------------------------------------------------------
// In-memory cache implementation
// -----------------------------------------------------------------------

type entry struct {
	value     any
	expiresAt time.Time
	noExpiry  bool
}

// MemoryCache is a simple in-process cache backed by a sync.Map.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*entry
}

// NewMemory creates a new in-memory cache.
// It automatically runs a background goroutine to expire stale entries.
func NewMemory() *MemoryCache {
	c := &MemoryCache{entries: make(map[string]*entry)}
	go c.gcLoop()
	return c
}

func (c *MemoryCache) Get(key string) (any, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !e.noExpiry && time.Now().After(e.expiresAt) {
		c.Delete(key)
		return nil, false
	}
	return e.value, true
}

func (c *MemoryCache) Set(key string, value any, ttl time.Duration) {
	e := &entry{value: value}
	if ttl == 0 {
		e.noExpiry = true
	} else {
		e.expiresAt = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.entries[key] = e
	c.mu.Unlock()
}

func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

func (c *MemoryCache) Flush() {
	c.mu.Lock()
	c.entries = make(map[string]*entry)
	c.mu.Unlock()
}

func (c *MemoryCache) Remember(key string, ttl time.Duration, fn func() any) any {
	if v, ok := c.Get(key); ok {
		return v
	}
	v := fn()
	c.Set(key, v, ttl)
	return v
}

// gcLoop removes expired entries every 30 seconds.
func (c *MemoryCache) gcLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, e := range c.entries {
			if !e.noExpiry && now.After(e.expiresAt) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}

// -----------------------------------------------------------------------
// Global default cache
// -----------------------------------------------------------------------

// Default is the global GoKS cache instance (in-memory by default).
var Default Cache = NewMemory()

// Get retrieves a value from the default cache.
func Get(key string) (any, bool) { return Default.Get(key) }

// Set stores a value in the default cache.
func Set(key string, value any, ttl time.Duration) { Default.Set(key, value, ttl) }

// Delete removes a key from the default cache.
func Delete(key string) { Default.Delete(key) }

// Remember retrieves or computes and caches a value.
func Remember(key string, ttl time.Duration, fn func() any) any {
	return Default.Remember(key, ttl, fn)
}
