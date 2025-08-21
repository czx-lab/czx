package cmap

import (
	"maps"
	"sync"
)

// Default window size for tracking the length of the map over time.
const defaultWinSize = 16

type CMap[K comparable, V any] struct {
	mu      sync.RWMutex
	data    map[K]V
	winLens []int
	winIdx  int
	winSize int
}

func New[K comparable, V any]() *CMap[K, V] {
	return &CMap[K, V]{
		data:    make(map[K]V),
		winLens: make([]int, defaultWinSize),
		winSize: defaultWinSize,
	}
}

func (c *CMap[K, V]) WithWinSize(size int) *CMap[K, V] {
	if size <= 0 {
		size = defaultWinSize
	}
	c.winSize = size
	c.winLens = make([]int, size)
	return c
}

func (c *CMap[K, V]) recordWinLen() {
	c.winLens[c.winIdx%c.winSize] = len(c.data)
	c.winIdx++
}

func (c *CMap[K, V]) avgWinLen() int {
	sum := 0
	for _, v := range c.winLens {
		sum += v
	}
	return sum / c.winSize
}

// Has checks if the map contains the given key.
// It returns true if the key exists, false otherwise.
func (c *CMap[K, V]) Has(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if the map has any elements
	if c.data == nil {
		return false
	}
	// If the map is not nil, check if it has any keys
	if len(c.data) == 0 {
		return false
	}
	// If the map has keys, return true
	_, ok := c.data[key]
	return ok
}

// Get retrieves the value for the given key.
func (c *CMap[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.data[key]
	return value, exists
}

// Delete removes the key-value pair for the given key.
func (c *CMap[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)

	c.recordWinLen()

	avg := c.avgWinLen()
	if avg > 0 && len(c.data) > 0 && len(c.data) > avg*2 {
		c.shrinkUnlocked()
	}
}

// shrinkUnlocked shrinks the internal map to reduce memory usage.
// It is called when the average length of the map is significantly smaller than the current size.
func (c *CMap[K, V]) shrinkUnlocked() {
	if len(c.data) == 0 {
		c.data = make(map[K]V)
		return
	}
	newData := make(map[K]V, len(c.data))
	maps.Copy(newData, c.data)
	c.data = newData
}

// Shrink reduces the size of the internal map to optimize memory usage.
// It is called to clean up unused memory after many deletions.
func (c *CMap[K, V]) Shrink() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.shrinkUnlocked()
}

// DeleteIf removes key-value pairs that match the provided condition.
// The function f should return true for pairs that should be deleted.
func (c *CMap[K, V]) DeleteIf(f func(K, V) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Iterate over the map and delete keys that match the condition
	for k, v := range c.data {
		if f(k, v) {
			delete(c.data, k)

			c.recordWinLen()

			avg := c.avgWinLen()
			if avg > 0 && len(c.data) > 0 && len(c.data) > avg*2 {
				c.shrinkUnlocked()
			}
		}
	}
}

// Iterator iterates over all key-value pairs in the map,
// calling the provided function for each pair.
func (c *CMap[K, V]) Iterator(f func(K, V) bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for k, v := range c.data {
		if !f(k, v) {
			break
		}
	}
}

// Keys returns a slice of all keys in the map.
// It returns an empty slice if the map is nil or empty.
func (c *CMap[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Get the keys from the map
	keys := make([]K, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}

	return keys
}

// OrderedIterator iterates over the map in the order of the provided keys,
// calling the provided function for each key-value pair.
func (c *CMap[K, V]) OrderedIterator(keys []K, f func(K, V) bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Iterate over the map in the order of keys
	for _, k := range keys {
		if v, exists := c.data[k]; exists {
			if !f(k, v) {
				break
			}
		}
	}
}

// Len returns the number of key-value pairs in the map.
// It returns 0 if the map is nil or empty.
func (c *CMap[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return the length of the map
	return len(c.data)
}

// Set adds or updates the value for the given key.
// If the key already exists, it updates the value.
func (c *CMap[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// Clear removes all key-value pairs from the map.
// It resets the map to an empty state.
func (c *CMap[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear the map
	c.data = make(map[K]V)
}
