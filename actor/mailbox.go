package actor

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

var (
	ErrMailboxNotRunning = errors.New("mailbox is not running")
	ErrMailboxStopped    = errors.New("mailbox is stopped")
)

type (
	// mbxState represents the state of the mailbox.
	mbxState int32
	// Mailbox is an interface that combines the Actor, MailboxWriter, and MailboxReceiver interfaces.
	Mailbox[T any] interface {
		Service
		MailboxWriter[T]
		MailboxReceiver[T]
	}
	// MailboxWriter is an interface for writing messages to the mailbox.
	MailboxWriter[T any] interface {
		Write(context.Context, T) error
	}

	// MailboxReceiver is an interface for receiving messages from the mailbox.
	MailboxReceiver[T any] interface {
		Receive() <-chan T
	}

	// mailbox is a concrete implementation of the Mailbox interface.
	mailbox[T any] struct {
		actor    Actor[T]
		writer   chan T
		receiver <-chan T
		done     chan struct{} // closed when the mailbox is stopped
		state    *atomic.Int32 // mbxState
	}
)

const (
	mbxIdle mbxState = iota
	mbxRunning
	mbxStopped
)

func (s mbxState) Int32() int32 {
	return int32(s)
}

func NewMailbox[T any](ctx context.Context, opts ...MboxOpt) Mailbox[T] {
	o := newOptions(opts...).Mailbox
	return mailboxIns[T](ctx, o)
}

func mailboxIns[T any](ctx context.Context, opts mboxOpts) *mailbox[T] {
	var (
		writer   = make(chan T, opts.Capacity)
		receiver = make(chan T, opts.Capacity)
	)
	mboxWorker := newMailboxWorker(opts, writer, receiver)
	mboxWorker.withContext(ctx)
	return &mailbox[T]{
		state:    new(atomic.Int32),
		writer:   writer,
		receiver: receiver,
		done:     make(chan struct{}),
		actor:    New[T](ctx, mboxWorker).WithPID(DefaultPID()),
	}
}

// Start implements Mailbox.
func (m *mailbox[T]) Start() {
	ok := m.state.CompareAndSwap(mbxIdle.Int32(), mbxRunning.Int32())
	if !ok {
		return
	}
	m.actor.Start()
}

// Stop implements Mailbox.
func (m *mailbox[T]) Stop() {
	ok := m.state.CompareAndSwap(mbxRunning.Int32(), mbxStopped.Int32())
	if !ok {
		return
	}

	m.actor.Stop()
	select {
	case <-m.done:
	default:
		close(m.done)
	}
}

// Write implements Mailbox.
func (m *mailbox[T]) Write(ctx context.Context, data T) error {
	state := m.state.Load()
	if state != mbxRunning.Int32() {
		return ErrMailboxNotRunning
	}
	select {
	case <-m.done:
		return ErrMailboxStopped
	case <-ctx.Done():
		return fmt.Errorf("write mailbox data timeout: %w", ctx.Err())
	case m.writer <- data:
		return nil
	}
}

// Receive implements Mailbox.
func (m *mailbox[T]) Receive() <-chan T {
	return m.receiver
}

var _ Mailbox[any] = (*mailbox[any])(nil)
