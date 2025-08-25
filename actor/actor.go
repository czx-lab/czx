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
	Supervised interface {
		WithParent(Supervisor)
	}
	// Actor defines the interface for an actor.
	Actor interface {
		Service
		Supervised
	}

	// actor is the concrete implementation of the Actor interface.
	// It manages the lifecycle of a Worker.
	actor struct {
		worker      Worker
		supervision Supervisor // parent actor, nil if root
		ctx         context.Context
		done        chan struct{} // channel to signal when the actor has stopped
		running     bool          // indicates if the actor is running
		mu          sync.Mutex
	}
)

func New(ctx context.Context, w Worker) Actor {
	return &actor{
		worker: w,
		ctx:    ctx,
		done:   make(chan struct{}),
	}
}

// WithParent implements Actor.
func (a *actor) WithParent(supervision Supervisor) {
	a.supervision = supervision
}

// Start implements Actor.
func (a *actor) Start() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return
	}
	a.running = true

	go a.run()
}

// Stop implements Actor.
func (a *actor) Stop() {
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
func (a *actor) run() {
	defer func() {
		if r := recover(); r != nil {
			select {
			case <-a.done:
			default:
				close(a.done)
				a.mu.Lock()
				a.running = false
				a.mu.Unlock()
			}

			// handle panic, possibly notify supervisor
			if a.supervision != nil {
				a.supervision.OnStop(a.worker.GetId()) // assuming we have a way to identify the child
			}
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
}

// start calls OnStart if the worker implements StartableWorker.
func (a *actor) start() {
	w, ok := a.worker.(StartableWorker)
	if !ok {
		return
	}
	w.OnStart(a.ctx)
}

// stop calls OnStop if the worker implements StopableWorker.
func (a *actor) stop() {
	w, ok := a.worker.(StopableWorker)
	if !ok {
		return
	}
	w.OnStop(a.worker.GetId())
}

var _ Actor = (*actor)(nil)
