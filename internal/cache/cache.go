package cache

import (
	"sync"
	"time"
)

type entry struct {
	val      interface{}
	expireAt int64
}

// Cache is a simple TTL map with FIFO eviction at capacity (mirrors CacheUtils).
type Cache struct {
	mu   sync.RWMutex
	m    map[string]*entry
	max  int
	keys []string
}

// New creates a cache with the given capacity (default 8192).
func New(max int) *Cache {
	if max <= 0 {
		max = 8192
	}
	return &Cache{m: make(map[string]*entry), max: max}
}

// Get returns the value for key if present and not expired.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	e, ok := c.m[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if e.expireAt > 0 && time.Now().UnixMilli() > e.expireAt {
		c.Remove(key)
		return nil, false
	}
	return e.val, true
}

// Put stores a value with an optional TTL in milliseconds (0 = never expires).
func (c *Cache) Put(key string, val interface{}, ttlMillis int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.m[key]; !ok {
		c.keys = append(c.keys, key)
		if len(c.keys) > c.max {
			old := c.keys[0]
			c.keys = c.keys[1:]
			delete(c.m, old)
		}
	}
	var exp int64
	if ttlMillis > 0 {
		exp = time.Now().UnixMilli() + ttlMillis
	}
	c.m[key] = &entry{val: val, expireAt: exp}
}

// Contains reports whether the key exists and is unexpired.
func (c *Cache) Contains(key string) bool {
	_, ok := c.Get(key)
	return ok
}

// PutDuration stores a value with a TTL expressed as a time.Duration.
func (c *Cache) PutDuration(key string, val interface{}, d time.Duration) {
	c.Put(key, val, int64(d/time.Millisecond))
}

// Remove deletes a key.
func (c *Cache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.m[key]; ok {
		delete(c.m, key)
		for i, k := range c.keys {
			if k == key {
				c.keys = append(c.keys[:i], c.keys[i+1:]...)
				break
			}
		}
	}
}

// Size returns the number of cached entries.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.m)
}

// Clear empties the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = make(map[string]*entry)
	c.keys = nil
}

// Default is the process-wide TTL cache.
var Default = New(8192)