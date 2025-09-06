package cqueue

import (
	"fmt"
	"slices"
	"sync"

	"github.com/czx-lab/czx/container/recycler"
	"github.com/czx-lab/czx/utils/xslices"
)

// Generic queue type
// Queue is a thread-safe queue that can hold elements of any type.
// It uses a mutex to ensure that only one goroutine can access the queue at a time.
type Queue[T any] struct {
	mu          sync.Mutex
	cond        *sync.Cond
	queue       []T
	maxCapacity int
	recycler    recycler.Recycler // Optional recycler for memory management
}

// NewQueue creates a new instance of Queue for the specified type T.
func NewQueue[T any](maxcap int) *Queue[T] {
	q := &Queue[T]{
		maxCapacity: maxcap,
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// WithRecycler sets a recycler for the queue.
func (q *Queue[T]) WithRecycler(r recycler.Recycler) *Queue[T] {
	q.recycler = r
	return q
}

// Delete removes the first occurrence of an element from the queue.
func (q *Queue[T]) DeleteFunc(fn func(T) bool) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == 0 {
		q.shrink()
		return false
	}

	q.queue = slices.DeleteFunc(q.queue, func(item T) bool {
		return fn(item)
	})

	q.shrink()
	return true
}

func (q *Queue[T]) shrink() {
	if q.recycler == nil {
		return
	}

	if q.recycler.Shrink(len(q.queue), cap(q.queue)) {
		q.queue = slices.Clip(q.queue) // Shrink the slice to fit its length
	}
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

	var available bool
	if len(q.queue) == 0 {
		available = true
	}

	q.queue = append(q.queue, data...)
	if available {
		q.cond.Signal() // Notify one waiting goroutine, if any
	}
	return nil
}

// Pop removes and returns the first element from the queue.
// It locks the queue to ensure thread safety while removing the element.
// If the queue is empty, it returns a zero value of type T and false.
func (q *Queue[T]) Pop() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == 0 {
		q.queue = nil
		var zero T
		return zero, false
	}

	data := q.queue[0]
	q.queue = q.queue[1:]

	if len(q.queue) == 0 {
		q.queue = nil // Clear the queue if it becomes empty
		return data, true
	}

	q.shrink()
	return data, true
}

// WaitPop removes and returns the first element from the queue.
func (q *Queue[T]) WaitPop() T {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.queue) == 0 {
		q.cond.Wait()
	}

	data := q.queue[0]
	q.queue = q.queue[1:]

	if len(q.queue) == 0 {
		q.queue = nil // Clear the queue if it becomes empty
		return data
	}

	q.shrink()
	return data
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
		return nil, false
	}

	if n > len(q.queue) {
		n = len(q.queue)
	}

	data := q.queue[:n]
	q.queue = q.queue[n:]

	if len(q.queue) == 0 {
		q.queue = nil // Clear the queue if it becomes empty
		return data, true
	}

	q.shrink()
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

	q.queue = slices.Clip(q.queue)
}
