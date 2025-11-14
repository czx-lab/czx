package cmap

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"

	"github.com/czx-lab/czx/container/recycler"
)

type (
	Option[K comparable] struct {
		Count int         // Number of shards
		Hash  func(K) int // Optional custom hash function
	}
	// Shareded is a sharded concurrent map implementation.
	// It divides the key space into multiple shards to reduce lock contention.
	Shareded[K comparable, V any] struct {
		shards []*CMap[K, V] // Array of shards
		opt    Option[K]
	}
)

func NewSharded[K comparable, V any](opt Option[K], r recycler.Recycler) *Shareded[K, V] {
	shards := make([]*CMap[K, V], opt.Count)
	for i := 0; i < opt.Count; i++ {
		shards[i] = New[K, V]().WithRecycler(r)
	}
	return &Shareded[K, V]{
		shards: shards,
		opt:    opt,
	}
}

// shard returns the shard corresponding to the given key.
func (s *Shareded[K, V]) shard(key K) *CMap[K, V] {
	var hash int
	if s.opt.Hash != nil {
		hash = s.opt.Hash(key)
	} else {
		// Default hash function using FNV
		h := fnv.New32a()
		switch k := any(key).(type) {
		case string:
			h.Write([]byte(k))
		case int:
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, uint64(k))
			h.Write(b)
		default:
			fmt.Fprintf(h, "%v", key)
		}
		hash = int(h.Sum32())
	}
	return s.shards[hash%len(s.shards)]
}

// Has checks if the key exists in the map.
func (s *Shareded[K, V]) Has(key K) bool {
	shard := s.shard(key)
	return shard.Has(key)
}

// Get retrieves the value for the given key.
func (s *Shareded[K, V]) Get(key K) (V, bool) {
	shard := s.shard(key)
	return shard.Get(key)
}

// Delete removes the key from the map.
func (s *Shareded[K, V]) Delete(key K) {
	shard := s.shard(key)
	shard.Delete(key)
}

// Shrink reduces the memory usage of all shards.
func (s *Shareded[K, V]) Shrink() {
	for _, shard := range s.shards {
		shard.Shrink()
	}
}

// Set sets the value for the given key.
func (s *Shareded[K, V]) Set(key K, value V) {
	shard := s.shard(key)
	shard.Set(key, value)
}

// Iterator iterates over all key-value pairs in the map.
func (s *Shareded[K, V]) Iterator(fn func(K, V) bool) {
	for _, shard := range s.shards {
		cont := true
		shard.Iterator(func(k K, v V) bool {
			cont = fn(k, v)
			return cont
		})
		if !cont {
			break
		}
	}
}

// Keys returns a slice of all keys in the map.
func (s *Shareded[K, V]) Keys() []K {
	var keys []K
	for _, shard := range s.shards {
		shardKeys := shard.Keys()
		keys = append(keys, shardKeys...)
	}
	return keys
}

// Len returns the total number of key-value pairs in the map.
func (s *Shareded[K, V]) Len() int {
	total := 0
	for _, shard := range s.shards {
		total += shard.Len()
	}
	return total
}

// Clear removes all key-value pairs from the map.
func (s *Shareded[K, V]) Clear() {
	for _, shard := range s.shards {
		shard.Clear()
	}
}
