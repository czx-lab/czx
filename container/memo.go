// Package container provides a simple in-memory cache with expiration.
package container

import (
	"time"

	"github.com/czx-lab/czx/container/cmap"
	"github.com/czx-lab/czx/container/recycler"
)

// Memo is a simple in-memory cache with expiration.
type (
	// CacheItem is a struct that holds a value and an expiration time.
	CacheItem struct {
		Value    any
		ExpireAt time.Time
	}
	MemoOption struct {
		Interval time.Duration
		cmap.Option[string]
	}
	Memo struct {
		data     *cmap.Shareded[string, *CacheItem]
		interval time.Duration
		done     chan struct{}
	}
)

// NewMemo creates a new Memo instance.
func NewMemo(opt MemoOption, r recycler.Recycler) *Memo {
	mc := &Memo{
		data:     cmap.NewSharded[string, *CacheItem](opt.Option, r),
		done:     make(chan struct{}),
		interval: opt.Interval,
	}
	// start the cleanup goroutine.
	go mc.cleanup()
	return mc
}

// cleanup cleans up expired items in the cache.
func (mc *Memo) cleanup() {
	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var keys []string
			mc.data.Iterator(func(key string, item *CacheItem) bool {
				if item.ExpireAt.IsZero() {
					return true // No expiration set, keep the item
				}
				if item.ExpireAt.Before(time.Now()) {
					// Iterative internal deletion may result in deadlock.
					keys = append(keys, key)
				}
				return true
			})
			// Delete expired items
			for _, key := range keys {
				mc.data.Delete(key)
			}
		case <-mc.done:
			return
		}
	}
}

// Clear clears the cache.
func (mc *Memo) Clear() {
	mc.data.Clear()
}

// Destroy stops the cleanup goroutine and clears the cache.
func (mc *Memo) Destroy() {
	select {
	case <-mc.done:
	default:
		close(mc.done)
	}
	mc.data.Clear()
}

// Set sets a value in the cache with a specified time-to-live (TTL).
func (mc *Memo) Set(key string, value any, ttl time.Duration) error {
	item := &CacheItem{Value: value}
	if ttl > 0 {
		item.ExpireAt = time.Now().Add(ttl)
	}
	mc.data.Set(key, item)
	return nil
}

// Get retrieves a value from the cache. If the value is expired, nil is returned.
func (mc *Memo) Get(key string) (any, error) {
	item, ok := mc.data.Get(key)
	if !ok {
		return nil, nil
	}

	if !item.ExpireAt.IsZero() && item.ExpireAt.Before(time.Now()) {
		mc.data.Delete(key)
		return nil, nil
	}
	return item.Value, nil
}

// Delete a value from the cache.
func (mc *Memo) Delete(key string) {
	mc.data.Delete(key)
}
