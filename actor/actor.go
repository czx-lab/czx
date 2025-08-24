package actor

import "context"

type (
	// Actor defines the interface for an actor.
	Actor interface {
		Start()
		Stop()
	}

	// Worker defines the interface for a worker.
	Worker interface {
		Exec(context.Context)
	}
)
