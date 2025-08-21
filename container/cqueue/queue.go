package cqueue

import (
	"fmt"
	"slices"
	"sync"

	"github.com/czx-lab/czx/utils/xslices"
)

const defaultWinSize = 16

// Generic queue type
// Queue is a thread-safe queue that can hold elements of any type.
// It uses a mutex to ensure that only one goroutine can access the queue at a time.
type Queue[T any] struct {
	mu          sync.Mutex
	queue       []T
	maxCapacity int
	// wins is a circular buffer that records the length of the queue over time.
	// It helps in tracking the average length of the queue.
	wins []int
	// Index for the current window in the wins array
	winIdx int
	// Length of the wins array, used for circular indexing
	winLen int
}

// NewQueue creates a new instance of Queue for the specified type T.
func NewQueue[T any](maxcap int) *Queue[T] {
	return &Queue[T]{
		maxCapacity: maxcap,
		winLen:      defaultWinSize,
		wins:        make([]int, defaultWinSize),
	}
}

// WithWinLen sets the size of the window for recording the length of the queue.
// It initializes the `wins` slice to the specified size.
// This allows the queue to track its length over time, which can be useful for monitoring and
// performance analysis.
// The `winLen` parameter specifies how many past lengths to keep track of.
// If the size is less than or equal to zero, it defaults to 16.
// This method returns the queue instance itself to allow for method chaining.
func (q *Queue[T]) WithWinLen(size int) *Queue[T] {
	q.winLen = size
	q.wins = make([]int, size)
	return q
}

// Delete removes the first occurrence of an element from the queue.
func (q *Queue[T]) DeleteFunc(fn func(T) bool) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == 0 {
		return false
	}

	q.queue = slices.DeleteFunc(q.queue, func(item T) bool {
		return fn(item)
	})

	return true
}

// Search searches for an element in the queue using a custom function.
func (q *Queue[T]) SearchFunc(fn func(T) bool) (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == 0 {
		var zero T
		return zero, false
	}

	index, ok := xslices.Search(q.queue, func(item T) bool {
		return fn(item)
	})
	if ok {
		return q.queue[index], true
	}

	var zero T
	return zero, false
}

// Push adds one or more elements to the end of the queue.
// It locks the queue to ensure thread safety while adding elements.
// If the queue has a maximum capacity and is full, it returns an error.
// If the queue is not full, it appends the elements to the end of the queue.
// It returns nil if the operation is successful.
func (q *Queue[T]) Push(data ...T) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.maxCapacity > 0 && len(q.queue) >= q.maxCapacity {
		return fmt.Errorf("queue is full, max capacity: %d", q.maxCapacity)
	}

	q.queue = append(q.queue, data...)

	q.recordLenWindow()
	return nil
}

// RecordLenWindow records the current length of the queue in a circular buffer.
// It updates the `wins` array with the current length of the queue at the current index.
// The `winIdx` is incremented to point to the next index for the next recording.
func (q *Queue[T]) recordLenWindow() {
	q.wins[q.winIdx%q.winLen] = len(q.queue)
	q.winIdx++
}

// AvgLenWindow calculates the average length of the queue over the last 16 recordings.
// It sums up the lengths recorded in the `wins` array and divides by the number of recordings.
// This provides a smoothed view of the queue's length over time, helping to understand its usage patterns.
// The average length can be useful for monitoring and performance analysis.
func (q *Queue[T]) avgLenWindow() int {
	sum := 0
	for _, v := range q.wins {
		sum += v
	}
	return sum / q.winLen
}

// Pop removes and returns the first element from the queue.
// It locks the queue to ensure thread safety while removing the element.
// If the queue is empty, it returns a zero value of type T and false.
func (q *Queue[T]) Pop() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == 0 {
		q.queue = nil

		q.recordLenWindow()
		var zero T
		return zero, false
	}

	data := q.queue[0]
	q.queue = q.queue[1:]

	q.recordLenWindow()

	if len(q.queue) == 0 {
		q.queue = nil // Clear the queue if it becomes empty
	}

	avg := q.avgLenWindow()
	if avg > 0 && cap(q.queue) > avg*2 && len(q.queue) > 0 {
		newQueue := make([]T, len(q.queue))
		copy(newQueue, q.queue)
		q.queue = newQueue
	}

	return data, true
}

// Len returns the current length of the queue.
// It locks the queue to ensure thread safety while accessing the length.
func (q *Queue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.queue)
}

// IsEmpty checks if the queue is empty.
// It locks the queue to ensure thread safety while checking.
func (q *Queue[T]) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.queue) == 0
}

// Peek returns the first element of the queue without removing it.
// It locks the queue to ensure thread safety while accessing the element.
// If the queue is empty, it returns a zero value of type T and false.
func (q *Queue[T]) Peek() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == 0 {
		var zero T
		return zero, false
	}

	return q.queue[0], true
}

// PopBatch removes and returns up to `n` elements from the queue.
// It locks the queue to ensure thread safety while removing the elements.
// If the queue has fewer than `n` elements, it returns all available elements.
func (q *Queue[T]) PopBatch(n int) ([]T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == 0 {
		q.queue = nil
		q.recordLenWindow()
		return nil, false
	}

	if n > len(q.queue) {
		n = len(q.queue)
	}

	data := q.queue[:n]
	q.queue = q.queue[n:]

	q.recordLenWindow()

	if len(q.queue) == 0 {
		q.queue = nil // Clear the queue if it becomes empty
	}

	avg := q.avgLenWindow()
	if avg > 0 && cap(q.queue) > avg*2 && len(q.queue) > 0 {
		newQueue := make([]T, len(q.queue))
		copy(newQueue, q.queue)
		q.queue = newQueue
	}

	return data, true
}

// Clear removes all elements from the queue.
// It locks the queue to ensure thread safety while clearing.
func (q *Queue[T]) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.queue = nil
}

// Shrink reduces the capacity of the queue to fit its current length.
// It locks the queue to ensure thread safety while shrinking.
func (q *Queue[T]) Shrink() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == 0 {
		q.queue = nil
		return
	}

	if cap(q.queue) == len(q.queue) {
		return // No need to shrink if capacity is already equal to length
	}

	newQueue := make([]T, len(q.queue))
	copy(newQueue, q.queue)
	q.queue = newQueue
}
