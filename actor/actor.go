package actor

import (
	"context"
	"sync"
)

type (
	// Service defines the basic lifecycle methods for an actor.
	Service interface {
		// Start starts the actor in a new goroutine.
		Start()
		// Stop stops the actor and waits for it to finish.
		Stop()
	}
	// Supervised defines the interface for an actor that can have a parent supervisor.
	Supervised[T any] interface {
		WithParent(Supervisor[T])
	}
	// Actor defines the interface for an actor.
	Actor[T any] interface {
		Service
		Supervised[T]
		ActorRef[T]
	}
	ActorRef[T any] interface {
		PID() *PID            // get the actor's PID
		Tell(message T) error // send a message to the actor
		// Set the actor's PID
		// WithPID returns the actor itself for chaining.
		WithPID(pid *PID) Actor[T]
		// Set the actor's mailbox
		// WithMailbox returns the actor itself for chaining.
		WithMailbox(mbox Mailbox[T]) Actor[T]
	}

	// actor is the concrete implementation of the Actor interface.
	// It manages the lifecycle of a Worker.
	actor[T any] struct {
		pid         *PID
		worker      Worker
		mailbox     Mailbox[T]
		supervision Supervisor[T] // parent actor, nil if root
		ctx         context.Context
		done        chan struct{} // channel to signal when the actor has stopped
		running     bool          // indicates if the actor is running
		mu          sync.Mutex
	}
)

func New[T any](ctx context.Context, w Worker) Actor[T] {
	return &actor[T]{
		worker: w,
		ctx:    ctx,
		done:   make(chan struct{}),
	}
}

func (a *actor[T]) WithPID(pid *PID) Actor[T] {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.pid = pid
	return a
}

func (a *actor[T]) WithMailbox(mbox Mailbox[T]) Actor[T] {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.mailbox = mbox
	return a
}

// PID implements Actor.
func (a *actor[T]) PID() *PID {
	return a.pid
}

// Tell implements Actor.
func (a *actor[T]) Tell(message T) error {
	return a.mailbox.Write(a.ctx, message)
}

// WithParent implements Actor.
func (a *actor[T]) WithParent(supervision Supervisor[T]) {
	a.supervision = supervision
}

// Start implements Actor.
func (a *actor[T]) Start() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return
	}
	a.running = true

	go a.run()
}

// Stop implements Actor.
func (a *actor[T]) Stop() {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return
	}
	a.running = false
	a.mu.Unlock()

	<-a.done // wait for actor to stop
}

// run is the main loop of the actor.
func (a *actor[T]) run() {
	// recover from panic to notify supervisor
	// if the worker panics, we catch it here and notify the supervisor
	defer func() {
		if r := recover(); r != nil {
			// handle panic, possibly notify supervisor
			if a.supervision != nil {
				a.supervision.OnStop(a.PID().ID) // assuming we have a way to identify the child
			}
		}
	}()

	// ensure we handle panics and call OnStop if applicable
	defer func() {
		a.stop() // call OnStop if applicable
		// signal that the actor has stopped
		{
			a.mu.Lock()
			a.running = false
			select {
			case <-a.done:
			default:
				close(a.done)
			}
			a.mu.Unlock()
		}
	}()

	a.start() // call OnStart if applicable

Exit:
	for {
		select {
		case <-a.ctx.Done():
			break Exit
		default:
		}
		state := a.worker.Exec(a.ctx)
		if state == WorkerStopped {
			break
		}
	}
}

// start calls OnStart if the worker implements StartableWorker.
func (a *actor[T]) start() {
	w, ok := a.worker.(StartableWorker)
	if !ok {
		return
	}
	w.OnStart(a.ctx)
}

// stop calls OnStop if the worker implements StopableWorker.
func (a *actor[T]) stop() {
	w, ok := a.worker.(StopableWorker)
	if !ok {
		return
	}
	w.OnStop(a.PID().ID)
}

var _ Actor[any] = (*actor[any])(nil)
