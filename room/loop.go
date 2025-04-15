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

	lastActivity := time.Now()  // Track the last activity time
	lastHeartbeat := time.Now() // Track the last heartbeat time

LOOP:
	for {
		select {
		case <-ticker.C:
			// Process messages
			select {
			case msg := <-l.msgs:
				if err := l.processor.Process(msg); err != nil {
					xlog.Write().Error("Error processing message:", zap.Error(err))
					l.Stop()
					break
				}
				lastActivity = time.Now() // Update the last activity time
			default:
				// No message in the channel, skip processing
			}

			// Handle heartbeat if the interval has passed
			if time.Since(lastHeartbeat) >= l.opt.heartbeatFrequency {
				if err := l.processor.HandleIdle(); err != nil {
					xlog.Write().Error("Error handling idle:", zap.Error(err))
					l.Stop()
					break
				}
				lastHeartbeat = time.Now() // Update the last heartbeat time
			}

			// Check for timeout
			if l.opt.timeout > 0 && time.Since(lastActivity) > l.opt.timeout {
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
