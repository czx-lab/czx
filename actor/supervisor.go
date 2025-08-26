package actor

import (
	"context"
	"time"

	"github.com/czx-lab/czx/container/cmap"
	"github.com/czx-lab/czx/container/recycler"
)

type (
	// Supervisor is an interface that extends Service and StopableWorker.
	// It manages child actors and can restart them if they stop unexpectedly.
	Supervisor[T any] interface {
		Service
		StopableWorker
		SpawnChild(w ...Actor[T])
		StopChild(pid string)
	}

	// SupervisorConf holds configuration for a Supervisor.
	SupervisorConf struct {
		RestartOnPanic bool // whether to restart a child actor if it panics
		// maximum number of restarts within the time window
		// if 0, no limit
		MaxRestarts int
		TimeWindow  time.Duration     // time window for MaxRestarts
		Recycler    recycler.Recycler // recycler for internal maps
	}
	// supervisor is the concrete implementation of the Supervisor interface.
	supervisor[T any] struct {
		conf SupervisorConf
		ctx  context.Context
		// child holds the child actors managed by this supervisor.
		child    *cmap.CMap[string, Actor[T]]
		restarts *cmap.CMap[string, []time.Time] // map of child ID to their restart timestamps
	}
)

func NewSupervisor[T any](ctx context.Context, conf SupervisorConf) Supervisor[T] {
	if conf.Recycler == nil {
		conf.Recycler = NewRecycler()
	}
	return &supervisor[T]{
		conf:     conf,
		ctx:      ctx,
		child:    cmap.New[string, Actor[T]]().WithRecycler(conf.Recycler),
		restarts: cmap.New[string, []time.Time]().WithRecycler(conf.Recycler),
	}
}

// SpawnChild implements Supervisor.
func (s *supervisor[T]) SpawnChild(actors ...Actor[T]) {
	for _, actor := range actors {
		actor.WithParent(s)
		s.child.Set(actor.PID().ID, actor)
	}
}

// Start implements Supervisor.
func (s *supervisor[T]) Start() {
	s.child.Iterator(func(_ string, a Actor[T]) bool {
		a.Start()
		return true
	})
}

// Stop implements Supervisor.
func (s *supervisor[T]) Stop() {
	s.child.Iterator(func(_ string, a Actor[T]) bool {
		a.Stop()
		return true
	})
}

// StopChild implements Supervisor.
func (s *supervisor[T]) StopChild(pid string) {
	if child, ok := s.child.Get(pid); ok {
		child.Stop()
		s.child.Delete(pid)
	}
}

// OnStop implements Supervisor.
func (s *supervisor[T]) OnStop(pid string) {
	if !s.conf.RestartOnPanic {
		return
	}
	if s.conf.MaxRestarts > 0 {
		now := time.Now()
		times, _ := s.restarts.Get(pid)
		// filter out timestamps outside the time window
		var validTimes []time.Time
		for _, t := range times {
			if now.Sub(t) <= s.conf.TimeWindow {
				validTimes = append(validTimes, t)
			}
		}
		if len(validTimes) >= s.conf.MaxRestarts {
			// exceeded max restarts, do not restart
			return
		}
		validTimes = append(validTimes, now)
		s.restarts.Set(pid, validTimes)
	}
	child, ok := s.child.Get(pid)
	if !ok {
		return
	}
	child.Start()
}

var _ Supervisor[any] = (*supervisor[any])(nil)
