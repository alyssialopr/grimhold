// Package cache fournit un cache mémoire avec durée de vie (TTL).
package cache

import (
	"sync"
	"time"
)

type entry struct {
	val     []byte
	expires time.Time
}

// Cache est une map protégée par un mutex. Une goroutine (le "reaper")
// supprime régulièrement les entrées expirées.
type Cache struct {
	mu       sync.Mutex
	entries  map[string]entry
	ttl      time.Duration
	now      func() time.Time
	stop     chan struct{}
	stopOnce sync.Once
}

// New crée un cache et démarre le reaper qui passe toutes les `interval`.
func New(ttl, interval time.Duration) *Cache {
	c := NewWithClock(ttl, time.Now)
	go c.reapLoop(interval)
	return c
}

// NewWithClock crée un cache sans reaper avec une horloge injectée
// (pratique pour les tests).
func NewWithClock(ttl time.Duration, now func() time.Time) *Cache {
	return &Cache{
		entries: make(map[string]entry),
		ttl:     ttl,
		now:     now,
		stop:    make(chan struct{}),
	}
}

// Add stocke une valeur pour la durée du TTL.
func (c *Cache) Add(key string, val []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = entry{val: val, expires: c.now().Add(c.ttl)}
}

// Get renvoie la valeur si elle existe et n'a pas expiré.
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || !c.now().Before(e.expires) {
		return nil, false
	}
	return e.val, true
}

// Len renvoie le nombre d'entrées (expirées ou non) encore en mémoire.
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// Reap supprime les entrées expirées et renvoie combien ont été supprimées.
func (c *Cache) Reap() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	n := 0
	for k, e := range c.entries {
		if !now.Before(e.expires) {
			delete(c.entries, k)
			n++
		}
	}
	return n
}

// Stop arrête le reaper.
func (c *Cache) Stop() {
	c.stopOnce.Do(func() { close(c.stop) })
}

func (c *Cache) reapLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.Reap()
		case <-c.stop:
			return
		}
	}
}
