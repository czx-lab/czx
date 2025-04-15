package room

import (
	"czx/xlog"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Loop struct {
	opt       *RoomConf
	quit      chan struct{}
	processor MessageProcessor
	// buffered channel to hold messages
	msgs chan Message
	// callback to be called when the loop stops
	onStop   func()
	stopOnce sync.Once // Ensure Stop is called only once
}

func NewLoop(processor MessageProcessor, opt *RoomConf) *Loop {
	return &Loop{
		opt:       opt,
		processor: processor,
		quit:      make(chan struct{}),
		msgs:      make(chan Message, opt.maxBufferSize),
	}
}

func (l *Loop) Start() error {
	return l.loop()
}

// Start starts the loop and processes messages
// in a separate goroutine. It will run until the loop is stopped or an error occurs.
func (l *Loop) loop() error {
	ticker := time.NewTicker(l.opt.frequency)
	defer ticker.Stop()

	heartbeatTicker := time.NewTicker(l.opt.heartbeatFrequency) // Separate ticker for heartbeat
	defer heartbeatTicker.Stop()

	lastActivity := time.Now() // Track the last activity time

LOOP:
	for {
		select {
		case <-ticker.C:
			select {
			case msg := <-l.msgs:
				if err := l.processor.Process(msg); err != nil {
					xlog.Write().Error("Error processing message:", zap.Error(err))

					l.Stop()
					break
				}

				// Update the last activity time
				lastActivity = time.Now()
			default:
				// No message in the channel, skip processing
			}
		case <-heartbeatTicker.C:
			if err := l.processor.HandleIdle(); err != nil {
				xlog.Write().Error("Error handling idle:", zap.Error(err))

				l.Stop()
				break
			}

			// Unified idle handling and timeout logic
			if l.opt.timeout == 0 {
				continue // No timeout set, skip this check
			}

			if time.Since(lastActivity) > l.opt.timeout {
				l.Stop()
			}
		case <-l.quit:
			break LOOP
		}
	}

	return l.processor.Close()
}

// Stop the loop and close the quit channel
func (l *Loop) Stop() {
	l.stopOnce.Do(func() {
		close(l.quit)

		if l.onStop == nil {
			return
		}
		l.onStop() // Notify via callback
	})
}

// Push is used to send messages to the room
// and receive messages from the room
func (l *Loop) Push(msg Message) error {
	if len(l.msgs) >= cap(l.msgs) {
		return ErrBufferFull
	}

	l.msgs <- msg
	return nil
}

// StopCallback is used to set a callback function
// that will be called when the loop stops.
func (l *Loop) stopCallback(callback func()) {
	l.onStop = callback
}
