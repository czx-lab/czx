package cqueue

import (
	"container/heap"
	"slices"
	"sync"
	"time"

	"github.com/czx-lab/czx/container/recycler"
	"github.com/czx-lab/czx/utils/xslices"
)

type (
	// PriorityItem is an item with a priority.
	// It is used to store items in the priority queue.
	// Each item has a value of type T and an integer priority.
	PriorityItem[T any] struct {
		Value    T
		Priority int
	}
	// qitems is a slice of pointers to Item
	qitems[T any] []*item[T]
	item[T any]   struct {
		value T
		// The priority of the item in the queue.
		priority  int
		index     int // The index of the item in the heap.
		timestamp int64
	}
	// A PriorityQueue implements heap.Interface and holds Items.
	// The zero value for PriorityQueue is an empty queue ready to use.
	// PriorityQueue is a thread-safe priority queue that can hold elements of any type.
	// It uses a mutex to ensure that only one goroutine can access the queue at a time.
	PriorityQueue[T any] struct {
		items    qitems[T]
		maxCap   int
		cond     *sync.Cond
		recycler recycler.Recycler
		mu       sync.Mutex
	}
)

// Len implements heap.Interface.
func (q qitems[T]) Len() int { return len(q) }

// Less implements heap.Interface.
func (q qitems[T]) Less(i int, j int) bool {
	// If priorities are equal, compare timestamps
	if q[i].priority == q[j].priority {
		return q[i].timestamp < q[j].timestamp
	}
	return q[i].priority < q[j].priority
}

// Pop implements heap.Interface.
func (q *qitems[T]) Pop() any {
	oqitem := *q
	n := len(oqitem)
	item := oqitem[n-1]
	item.index = -1 // for safety
	*q = oqitem[0 : n-1]
	return item
}

// Push implements heap.Interface.
func (q *qitems[T]) Push(x any) {
	n := len(*q)
	item := x.(*item[T])
	item.index = n
	*q = append(*q, item)
}

// Swap implements heap.Interface.
func (q qitems[T]) Swap(i int, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func NewPriorityQueue[T any](MaxCapacity int) *PriorityQueue[T] {
	pq := &PriorityQueue[T]{
		maxCap: MaxCapacity,
	}
	pq.cond = sync.NewCond(&pq.mu)
	return pq
}

func (pq *PriorityQueue[T]) WithRecycler(r recycler.Recycler) *PriorityQueue[T] {
	pq.recycler = r
	return pq
}

func (pq *PriorityQueue[T]) shrink() {
	if pq.recycler == nil {
		return
	}
	if !pq.recycler.Shrink(len(pq.items), cap(pq.items)) {
		return
	}
	pq.items = slices.Clip(pq.items) // Shrink the slice to fit its length
}

// Len returns the number of elements in the priority queue.
func (pq *PriorityQueue[T]) Len() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	return len(pq.items)
}

// Push adds an element to the priority queue with the given priority.
// Returns false if the queue is at max capacity.
func (pq *PriorityQueue[T]) Push(value PriorityItem[T]) bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.maxCap > 0 && len(pq.items) >= pq.maxCap {
		return false
	}
	var available bool
	if len(pq.items) == 0 {
		available = true
	}
	item := &item[T]{value: value.Value, priority: value.Priority, timestamp: time.Now().UnixMilli()}
	heap.Push(&pq.items, item)
	if available {
		pq.cond.Signal()
	}
	return true
}

// WaitPop blocks until an element is available and returns it.
func (pq *PriorityQueue[T]) WaitPop() T {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	for len(pq.items) == 0 {
		pq.cond.Wait()
	}
	val := heap.Pop(&pq.items).(*item[T])
	pq.shrink()
	return val.value
}

// Pop removes and returns the highest priority element from the priority queue.
// If the queue is empty, it returns the zero value of T and false.
func (pq *PriorityQueue[T]) Pop() (value T, ok bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.items) == 0 {
		pq.shrink()
		var zero T
		return zero, false
	}
	val := heap.Pop(&pq.items).(*item[T])
	pq.shrink()
	return val.value, true
}

// Peek returns the highest priority element without removing it from the queue.
// If the queue is empty, it returns the zero value of T and false.
func (pq *PriorityQueue[T]) Peek() (value T, ok bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if len(pq.items) == 0 {
		pq.shrink()
		var zero T
		return zero, false
	}
	return pq.items[0].value, true
}

// SearchFunc searches for an element in the priority queue that satisfies the given function.
// It returns the element and true if found, otherwise it returns the zero value of T and false.
func (pq *PriorityQueue[T]) SearchFunc(fn func(T) bool) (T, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.items) == 0 {
		var zero T
		return zero, false
	}
	index, ok := xslices.Search(pq.items, func(item *item[T]) bool {
		return fn(item.value)
	})
	if ok {
		return pq.items[index].value, true
	}
	var zero T
	return zero, false
}

// Clear removes all elements from the priority queue.
func (pq *PriorityQueue[T]) Clear() {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	pq.items = pq.items[:0]
	pq.shrink()
}

// Shrink reduces the capacity of the priority queue's underlying slice to fit its length.
func (pq *PriorityQueue[T]) Shrink() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if len(pq.items) == cap(pq.items) {
		return
	}

	pq.items = slices.Clip(pq.items) // Shrink the slice to fit its length
}
