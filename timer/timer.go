package timer

import (
	"sync"
	"time"
)

type (
	// Timer is a struct that holds a time.Timer and a callback function.
	// The Timer struct is used to manage timers in the dispatcher.
	Timer struct {
		t  *time.Timer
		cb func()
	}

	// Dispatcher is a struct that holds a channel for timers.
	// The Dispatcher struct is used to manage the timers and their callbacks.
	// It is responsible for dispatching the timers and executing their callbacks.
	Dispatcher struct {
		chanTimer chan *Timer
		wg        sync.WaitGroup
		done      chan struct{}
		once      sync.Once
	}

	// Cron is a struct that holds a channel for timers and a callback function.
	// The Cron struct is used to manage cron jobs in the dispatcher.
	// It is responsible for dispatching the cron jobs and executing their callbacks.
	Cron struct {
		t *Timer
	}
)

// NewDispatcher creates a new Dispatcher with a channel for timers.
// The channel size is specified by the parameter l.
// The Dispatcher is used to manage the timers and their callbacks.
func NewDispatcher(l int) *Dispatcher {
	disp := new(Dispatcher)
	disp.chanTimer = make(chan *Timer, l)
	disp.done = make(chan struct{})

	return disp
}

// The Stop method is used to stop the timer and execute the callback function.
// It is called when the timer is no longer needed.
func (t *Timer) Stop() {
	if t.t != nil {
		t.t.Stop()
	}

	t.cb = nil
}

// exec executes the callback function of the timer.
func (t *Timer) exec() {
	if t.cb != nil {
		go t.cb()
	}
}

// AfterFunc creates a new Timer that will execute the callback function after the specified duration.
// The method takes a duration and a callback function as parameters.
// It returns a pointer to the Timer struct that was created.
func (disp *Dispatcher) AfterFunc(d time.Duration, cb func()) *Timer {
	t := new(Timer)
	t.cb = cb
	t.t = time.AfterFunc(d, func() {
		select {
		case disp.chanTimer <- t:
		case <-disp.done:
			t.exec()
			return
		}
	})

	return t
}

// Start the dispatcher and listen for timers
// The Start method is used to start the dispatcher and listen for timers.
// It is called when the dispatcher is started and runs in a separate goroutine.
// The method uses a select statement to listen for timers on the chanTimer channel and for a signal to stop the dispatcher on the done channel.
func (disp *Dispatcher) Start() {
	disp.wg.Add(1)
	defer disp.wg.Done()

	for {
		select {
		case t := <-disp.chanTimer:
			t.exec()
		case <-disp.done:
			for {
				select {
				case t := <-disp.chanTimer:
					t.exec()
				default:
					return
				}
			}
		}
	}
}

// Stop the dispatcher and clean up resources
// The Stop method is used to stop the dispatcher and clean up resources.
// It is called when the dispatcher is no longer needed.
func (disp *Dispatcher) Stop() {
	disp.once.Do(func() {
		close(disp.done)
		disp.wg.Wait()

		close(disp.chanTimer)
	})
}

// Stop the cron job and clean up resources
// The Stop method is used to stop the cron job and clean up resources.
// It is called when the cron job is no longer needed.
func (c *Cron) Stop() {
	if c.t == nil {
		return
	}

	c.t.Stop()
}

// cronExpr is a pointer to a CronExpr struct
// that represents a cron expression. The CronExpr struct is used to parse and evaluate cron expressions.
// The CronExpr struct contains fields for seconds, minutes, hours, day of month, month, and day of week.
func (disp *Dispatcher) CronFunc(cronExpr *CronExpr, _cb func()) *Cron {
	c := new(Cron)

	now := time.Now()
	nextTime := cronExpr.Next(now)
	if nextTime.IsZero() {
		return c
	}

	// callback
	var cb func()
	cb = func() {
		_cb()

		now := time.Now()
		nextTime := cronExpr.Next(now)
		if nextTime.IsZero() {
			return
		}

		c.t = disp.AfterFunc(nextTime.Sub(now), cb)
	}

	c.t = disp.AfterFunc(nextTime.Sub(now), cb)
	return c
}
