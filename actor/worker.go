package actor

import (
	"context"

	"github.com/czx-lab/czx/container/cqueue"
)

type (
	// state returned by Worker.Exec
	WorkerState int8
	// Worker is the interface that wraps the Exec method.
	// Exec is called repeatedly by the actor until it returns WorkerStopped.
	Worker interface {
		Exec(context.Context) WorkerState
	}
	// StartableWorker is the interface that wraps the OnStart method.
	// OnStart is called when the actor starts.
	StartableWorker interface {
		OnStart(context.Context)
	}
	// StopableWorker is the interface that wraps the OnStop method.
	// OnStop is called when the actor stops.
	StopableWorker interface {
		OnStop(pid string)
	}
	// WorkerFunc is a function that implements the Worker interface.
	WorkerFunc func(context.Context) WorkerState
	// mailboxWorker is a worker that processes messages from a channel and sends them to another channel.
	// It uses a queue to buffer messages when the receiver channel is full.
	mailboxWorker[T any] struct {
		ctx      context.Context
		writer   chan *cqueue.PriorityItem[T]
		receiver chan T
		queue    *cqueue.PriorityQueue[T]
	}
	// worker is a simple implementation of the Worker interface that wraps a WorkerFunc.
	worker struct {
		id string
		fn WorkerFunc
	}
)

const (
	WorkerIdle WorkerState = iota
	WorkerRunning
	WorkerStopped
)

func newMailboxWorker[T any](opts mboxOpts, writer chan *cqueue.PriorityItem[T], receiver chan T) *mailboxWorker[T] {
	return &mailboxWorker[T]{
		writer:   writer,
		receiver: receiver,
		queue:    cqueue.NewPriorityQueue[T](0).WithRecycler(opts.Recycler),
	}
}

func (m *mailboxWorker[T]) withContext(ctx context.Context) {
	m.ctx = ctx
}

// Exec implements Worker.
func (m *mailboxWorker[T]) Exec(context.Context) WorkerState {
	msgFn := func(msg *cqueue.PriorityItem[T]) {
		select {
		case m.receiver <- msg.Value: // direct send to receiver
		default:
			// if receiver is full, push to queue
			m.queue.Push(cqueue.PriorityItem[T]{
				Value:    msg.Value,
				Priority: msg.Priority,
			})
		}
	}
	// if empty queue, try read from writer
	if m.queue.Len() == 0 {
		select {
		case <-m.ctx.Done():
			return WorkerStopped
		case msg := <-m.writer:
			msgFn(msg)
		}
	}

	// drain queue
	// if not empty queue, try write to receiver
	select {
	case <-m.ctx.Done():
		return WorkerStopped
	case msg := <-m.writer:
		msgFn(msg)
		return WorkerRunning
	default:
		if item, ok := m.queue.Pop(); ok {
			m.receiver <- item
			return WorkerRunning
		}
		return WorkerIdle
	}
}

// OnStop implements StopableWorker.
func (m *mailboxWorker[T]) OnStop(string) {
	defer close(m.receiver)

	for m.queue.Len() > 0 {
		if item, ok := m.queue.Pop(); ok {
			m.receiver <- item
		}
	}

ReadAll:
	for {
		select {
		case dta := <-m.writer:
			m.receiver <- dta.Value
		default:
			break ReadAll
		}
	}
}

func NewWorker(fn WorkerFunc) Worker {
	return &worker{fn: fn}
}

// GetId implements Worker.
func (w *worker) GetId() string {
	return w.id
}

// Exec implements Worker.
func (w *worker) Exec(ctx context.Context) WorkerState {
	return w.fn(ctx)
}

var (
	_ Worker         = (*mailboxWorker[any])(nil)
	_ StopableWorker = (*mailboxWorker[any])(nil)
	_ Worker         = (*worker)(nil)
)
