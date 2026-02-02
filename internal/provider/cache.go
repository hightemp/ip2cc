// Package provider handles provider/ASN resolution.
package provider

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// CacheEntry represents a cached ASN holder entry.
type CacheEntry struct {
	Holder    string    `json:"holder"`
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Cache is a persistent cache for ASN holder information.
type Cache struct {
	mu      sync.RWMutex
	entries map[int]*CacheEntry
	path    string
	ttl     time.Duration
	dirty   bool
}

// NewCache creates a new provider cache.
func NewCache(path string, ttlDays int) *Cache {
	return &Cache{
		entries: make(map[int]*CacheEntry),
		path:    path,
		ttl:     time.Duration(ttlDays) * 24 * time.Hour,
	}
}

// Load loads the cache from disk.
func (c *Cache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var entries map[int]*CacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	c.entries = entries
	return nil
}

// Save saves the cache to disk.
func (c *Cache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.dirty {
		return nil
	}

	data, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0644)
}

// Get retrieves a cached holder for an ASN.
func (c *Cache) Get(asn int) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[asn]
	if !ok {
		return "", false
	}

	if time.Now().After(entry.ExpiresAt) {
		return "", false
	}

	return entry.Holder, true
}

// Set stores a holder for an ASN.
func (c *Cache) Set(asn int, holder string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	c.entries[asn] = &CacheEntry{
		Holder:    holder,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}
	c.dirty = true
}

// Clear removes all cache entries.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[int]*CacheEntry)
	c.dirty = true
}

// Cleanup removes expired entries.
func (c *Cache) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0
	for asn, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, asn)
			removed++
		}
	}
	if removed > 0 {
		c.dirty = true
	}
	return removed
}

// Size returns the number of cached entries.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
