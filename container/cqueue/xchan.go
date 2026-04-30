package cqueue

import (
	"context"

	"github.com/czx-lab/czx/container/ringbuffer"
)

type (
	XchanConf struct {
		Bufsize int // Size of the buffer
		Insize  int // Size of the input channel
		Outsize int // Size of the output channel
	}
	// Xchan is a concurrent channel that allows writing to an input channel and reading from an output channel.
	// It uses a ring buffer to manage the flow of data between the input and output channels.
	// It supports burst writes and ensures that the output channel is not blocked by full buffers.
	// The buffer size is configurable, and it can handle concurrent writes and reads efficiently.
	Xchan[T any] struct {
		conf   XchanConf
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

func (x *Xchan[T]) worker(ctx context.Context, in, out chan T) {
	defer close(out)

	drain := func() {
		for !x.buffer.IsEmpty() {
			val, ok := x.buffer.Peek()
			if !ok {
				return
			}
			select {
			case out <- val:
				x.buffer.Pop()
			case <-ctx.Done():
				return
			}
		}

		x.buffer.Reset()
	}

	for {
		if x.buffer.IsEmpty() {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					return
				}

				select {
				case out <- v:
				default:
					x.buffer.Write(v)
				}
			}

			continue
		}

		val, ok := x.buffer.Peek()
		if !ok {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case v, ok := <-in:
			if !ok {
				drain()
				return
			}

			x.buffer.Write(v)
		case out <- val:
			x.buffer.Pop()
			if x.buffer.IsEmpty() && x.buffer.Cap() > x.conf.Bufsize {
				x.buffer.Reset()
			}
		}
	}
}
