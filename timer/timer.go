package timer

import (
	"czx/xlog"
	"runtime"
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
		ChanTimer chan *Timer
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
	disp.ChanTimer = make(chan *Timer, l)
	return disp
}

// The Stop method is used to stop the timer and execute the callback function.
// It is called when the timer is no longer needed.
func (t *Timer) Stop() {
	t.t.Stop()
	t.cb = nil
}

// Cb is a method that executes the callback function of the timer.
// It is called when the timer expires and the callback function is set.
// The method uses a deferred function to recover from any panic that may occur during the execution of the callback function.
func (t *Timer) Cb() {
	defer func() {
		t.cb = nil
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			l := runtime.Stack(buf, false)
			xlog.Write().Sugar().Errorf("timer panic error %v: %s", r, buf[:l])
		}
	}()

	if t.cb != nil {
		t.cb()
	}
}

// AfterFunc creates a new Timer that will execute the callback function after the specified duration.
// The method takes a duration and a callback function as parameters.
// It returns a pointer to the Timer struct that was created.
func (disp *Dispatcher) AfterFunc(d time.Duration, cb func()) *Timer {
	t := new(Timer)
	t.cb = cb
	t.t = time.AfterFunc(d, func() {
		disp.ChanTimer <- t
	})
	return t
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
		defer _cb()

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
