package container

import (
	"slices"
	"sync"

	"github.com/czx-lab/czx/container/recycler"
)

const defaultWinSize = 16

type Xslices[T comparable] struct {
	mu       sync.RWMutex
	data     []T
	recycler recycler.Recycler
}

func New[T comparable]() *Xslices[T] {
	return &Xslices[T]{
		data: make([]T, 0),
	}
}

func (xs *Xslices[T]) WithRecycler(r recycler.Recycler) *Xslices[T] {
	xs.recycler = r
	return xs
}

func (xs *Xslices[T]) shrink() {
	if xs.recycler == nil {
		return
	}

	if xs.recycler.Shrink(len(xs.data), cap(xs.data)) {
		xs.data = slices.Clip(xs.data) // Shrink the slice to fit its length
	}
}

// Append adds new items to the Xslices data slice.
// It locks the mutex to ensure thread safety while appending items.
// After appending, it records the new length in the wins slice.
// If the data slice is empty, it initializes it to nil.
func (xs *Xslices[T]) Append(items ...T) {
	xs.mu.Lock()
	defer xs.mu.Unlock()

	xs.data = append(xs.data, items...)
}

// Get retrieves an item from the Xslices data slice by index.
func (xs *Xslices[T]) Get(index int) (T, bool) {
	xs.mu.RLock()
	defer xs.mu.RUnlock()

	if index < 0 || index >= len(xs.data) {
		var zero T
		return zero, false
	}
	return xs.data[index], true
}

// Remove deletes an item from the Xslices data slice by index.
// It locks the mutex to ensure thread safety while removing the item.
// If the index is out of bounds, it returns false.
func (xs *Xslices[T]) Remove(index int) bool {
	xs.mu.Lock()
	defer xs.mu.Unlock()

	if index < 0 || index >= len(xs.data) {
		return false
	}
	xs.data = slices.Delete(xs.data, index, index+1)

	xs.shrink()
	return true
}

// Len returns the current length of the Xslices data slice.
func (xs *Xslices[T]) Len() int {
	xs.mu.RLock()
	defer xs.mu.RUnlock()

	return len(xs.data)
}

// Clone creates a shallow copy of the Xslices data slice.
func (xs *Xslices[T]) Clone() []T {
	xs.mu.RLock()
	defer xs.mu.RUnlock()

	return slices.Clone(xs.data)
}

// Clear removes all items from the Xslices data slice.
func (xs *Xslices[T]) Clear() {
	xs.mu.Lock()
	defer xs.mu.Unlock()

	xs.data = nil
}

// SearchFunc searches for an item in the Xslices data slice using a provided function.
func (xs *Xslices[T]) SearchFunc(fn func(v T) bool) (v T, ok bool) {
	xs.mu.RLock()
	defer xs.mu.RUnlock()

	for _, item := range xs.data {
		if fn(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}

// DeleteIf removes items from the Xslices data slice that match a provided condition.
func (xs *Xslices[T]) DeleteIf(fn func(item T) bool) {
	xs.mu.Lock()
	defer xs.mu.Unlock()

	for i := len(xs.data) - 1; i >= 0; i-- {
		if fn(xs.data[i]) {
			xs.data = slices.Delete(xs.data, i, i+1)
		}
	}

	xs.shrink()
}

// Iterator iterates over the Xslices data slice,
// calling the provided function for each item.
func (xs *Xslices[T]) Iterator(fn func(item T)) {
	xs.mu.RLock()
	defer xs.mu.RUnlock()

	for _, item := range xs.data {
		fn(item)
	}
}

// Shrink reduces the capacity of the Xslices data slice
// to fit its current length.
func (xs *Xslices[T]) Shrink() {
	xs.mu.Lock()
	defer xs.mu.Unlock()

	if len(xs.data) == 0 {
		xs.data = nil
		return
	}

	xs.data = slices.Clip(xs.data)
}
