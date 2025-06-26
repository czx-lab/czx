package cqueue

import (
	"context"
	"sync/atomic"

	"github.com/czx-lab/czx/container/ringbuffer"
)

type (
	XchanConf struct {
		Bufsize int // Size of the buffer
		Insize  int // Size of the input channel
		Outsize int // Size of the output channel
	}
	Xchan[T any] struct {
		conf   XchanConf
		size   int64
		in     chan<- T // channel for write
		out    <-chan T // channel for read
		buffer *ringbuffer.RingBuffer[T]
	}
)

func NewXchan[T any](ctx context.Context, conf XchanConf) *Xchan[T] {
	in := make(chan T, conf.Insize)
	out := make(chan T, conf.Outsize)
	xch := &Xchan[T]{
		conf:   conf,
		in:     in,
		out:    out,
		buffer: ringbuffer.NewRingBuffer[T](conf.Bufsize),
	}

	go xch.worker(ctx, in, out)

	return xch
}

// In returns the input channel for writing elements.
func (x *Xchan[T]) In() chan<- T {
	return x.in
}

// Out returns the output channel for reading elements.
func (x *Xchan[T]) Out() <-chan T {
	return x.out
}

// Len returns the total number of elements in the input channel, buffer, and output channel.
// It calculates the length of the input channel, the size of the buffer, and the length
func (x *Xchan[T]) Len() int {
	return len(x.in) + x.BufferLen() + len(x.out)
}

// BufferLen returns the number of elements in the buffer.
// It uses atomic operations to ensure thread safety when accessing the size.
func (x *Xchan[T]) BufferLen() int {
	return int(atomic.LoadInt64(&x.size))
}

// worker is the goroutine that processes the input channel and writes to the output channel.
// It reads from the input channel, writes to the buffer, and drains the buffer to the output channel.
// It also handles the case where the output channel is full by writing to the buffer instead.
func (x *Xchan[T]) worker(ctx context.Context, in, out chan T) {
	defer close(out)

	drain := func() {
		for !x.buffer.IsEmpty() {
			val, ok := x.buffer.Pop()
			if !ok {
				return // Buffer is empty
			}
			select {
			case out <- val:
				atomic.AddInt64(&x.size, -1)
			case <-ctx.Done():
				return
			}
		}

		x.buffer.Reset()
		atomic.StoreInt64(&x.size, 0)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case val, ok := <-in:
			if !ok { // in is closed
				drain()
				return
			}
			if atomic.LoadInt64(&x.size) > 0 {
				x.buffer.Write(val)
				atomic.AddInt64(&x.size, 1)
			} else {
				// out is not full
				select {
				case out <- val:
					continue
				default:
				}

				// out is full
				x.buffer.Write(val)
				atomic.AddInt64(&x.size, 1)
			}
			for !x.buffer.IsEmpty() {
				select {
				case <-ctx.Done():
					return
				case val, ok := <-in:
					// in is closed
					if !ok {
						drain()
						return
					}
					x.buffer.Write(val)
					atomic.AddInt64(&x.size, 1)
				default:
					val, ok := x.buffer.Peek()
					if !ok {
						continue
					}
					out <- val
					x.buffer.Pop()
					atomic.AddInt64(&x.size, -1)
					if x.buffer.IsEmpty() && x.buffer.Cap() > x.conf.Bufsize { // after burst
						x.buffer.Reset()
						atomic.StoreInt64(&x.size, 0)
					}
				}
			}
		}
	}
}
