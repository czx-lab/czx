package cmap

import (
	"sync"
)

type CMap[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

func New[K comparable, V any]() *CMap[K, V] {
	return &CMap[K, V]{
		data: make(map[K]V),
	}
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
