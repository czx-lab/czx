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
	Supervisor interface {
		Service
		StopableWorker
		SpawnChild(w ...Worker)
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
	supervisor struct {
		conf SupervisorConf
		ctx  context.Context
		// child holds the child actors managed by this supervisor.
		child    *cmap.CMap[string, Actor]
		restarts *cmap.CMap[string, []time.Time] // map of child ID to their restart timestamps
	}
)

func NewSupervisor(ctx context.Context, conf SupervisorConf) Supervisor {
	if conf.Recycler == nil {
		conf.Recycler = NewRecycler()
	}
	return &supervisor{
		conf:     conf,
		ctx:      ctx,
		child:    cmap.New[string, Actor]().WithRecycler(conf.Recycler),
		restarts: cmap.New[string, []time.Time]().WithRecycler(conf.Recycler),
	}
}

// SpawnChild implements Supervisor.
func (s *supervisor) SpawnChild(wks ...Worker) {
	for _, w := range wks {
		child := New(s.ctx, w)
		child.WithParent(s)
		s.child.Set(w.GetId(), child)
	}
}

// Start implements Supervisor.
func (s *supervisor) Start() {
	s.child.Iterator(func(_ string, a Actor) bool {
		a.Start()
		return true
	})
}

// Stop implements Supervisor.
func (s *supervisor) Stop() {
	s.child.Iterator(func(_ string, a Actor) bool {
		a.Stop()
		return true
	})
}

// StopChild implements Supervisor.
func (s *supervisor) StopChild(pid string) {
	if child, ok := s.child.Get(pid); ok {
		child.Stop()
		s.child.Delete(pid)
	}
}

// OnStop implements Supervisor.
func (s *supervisor) OnStop(pid string) {
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
	child.Stop()
	child.Start()
}

var _ Supervisor = (*supervisor)(nil)
