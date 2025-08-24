package actor

import (
	"context"
)

type (
	// Mailbox is an interface that combines the Actor, MailboxWriter, and MailboxReceiver interfaces.
	Mailbox[T any] interface {
		Actor
		MailboxWriter[T]
		MailboxReceiver[T]
	}

	MailboxWriter[T any] interface {
		Write(context.Context, T) error
	}

	MailboxReceiver[T any] interface {
		Receive() <-chan T
	}
)
