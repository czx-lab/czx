package ringbuffer

const (
	// defaultCapacity is the default capacity of the ring buffer
	defaultCapacity = 1024
)

type RingBuffer[T any] struct {
	// Buffer to hold the elements
	buf []T
	// Size of the buffer
	init, size int
	r          int // Read index
	w          int // Write index
}

func NewRingBuffer[T any](cap int) *RingBuffer[T] {
	if cap <= 0 {
		cap = defaultCapacity
	}

	return &RingBuffer[T]{
		buf:  make([]T, cap),
		init: cap,
		size: cap,
	}
}

// Read reads an element from the ring buffer.
// It returns the element and a boolean indicating whether the read was successful.
func (rb *RingBuffer[T]) Read() (T, bool) {
	var zero T
	if rb.r == rb.w {
		return zero, false // Buffer is empty
	}

	item := rb.buf[rb.r]
	rb.r = (rb.r + 1) % rb.size // Move read index forward

	return item, true
}

// Pop removes and returns the first element from the ring buffer.
// It returns the element and a boolean indicating whether the pop was successful.
func (rb *RingBuffer[T]) Pop() (T, bool) {
	v, ok := rb.Read()
	if !ok {
		return v, false // Buffer is empty
	}
	return v, true
}

// Peek returns the first element from the ring buffer without removing it.
// It returns the element and a boolean indicating whether the peek was successful.
func (rb *RingBuffer[T]) Peek() (T, bool) {
	if rb.r == rb.w {
		var zero T
		return zero, false // Buffer is empty
	}

	return rb.buf[rb.r], true // Return the item at the read index without removing it
}

// Write adds an element to the end of the ring buffer.
// If the buffer is full, it grows the buffer to accommodate more elements.
func (rb *RingBuffer[T]) Write(data T) {
	nextW := (rb.w + 1) % rb.size
	if nextW == rb.r {
		rb.grow() // Buffer is full, grow it
		nextW = (rb.w + 1) % rb.size
	}

	rb.buf[rb.w] = data
	rb.w = nextW
}

// grow increases the size of the ring buffer.
// It reallocates the buffer and copies the existing elements to the new buffer.
func (rb *RingBuffer[T]) grow() {
	var size int
	if rb.size < 1024 {
		size = rb.size * 2
	} else {
		size = rb.size + rb.size/4
	}
	buf := make([]T, size)
	n := rb.Len()
	for i := range n {
		buf[i] = rb.buf[(rb.r+i)%rb.size]
	}
	rb.r = 0
	rb.w = n
	rb.size = size
	rb.buf = buf
}

// IsEmpty checks if the ring buffer is empty.
func (rb *RingBuffer[T]) IsEmpty() bool {
	return rb.r == rb.w
}

// Cap returns the capacity of the ring buffer.
// It returns the size of the buffer.
func (rb *RingBuffer[T]) Cap() int {
	return rb.size
}

// Len returns the number of elements in the ring buffer.
// It calculates the length based on the read and write indices.
func (rb *RingBuffer[T]) Len() int {
	if rb.w >= rb.r {
		return rb.w - rb.r
	}

	return rb.size - rb.r + rb.w
}

// Reset resets the ring buffer to its initial state.
// It clears the read and write indices and reinitializes the buffer.
func (rb *RingBuffer[T]) Reset() {
	rb.r = 0
	rb.w = 0
	rb.size = rb.init
	rb.buf = make([]T, rb.init)
}
