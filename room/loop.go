package room

import (
	"fmt"
	"sync"
	"time"
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
			// Batch process messages to improve efficiency
		BatchLoop:
			for range l.opt.batchSize {
				select {
				case msg := <-l.msgs:
					if err := l.processor.Process(msg); err != nil {
						fmt.Println("Error processing message:", err)
						// Log the error but continue the loop
					}

					// Update the last activity time
					// This is to ensure that the heartbeat logic works correctly
					lastActivity = time.Now()
				default:
					// No more messages to process, exit batch loop
					break BatchLoop
				}
			}
		case <-heartbeatTicker.C:
			if err := l.processor.HandleIdle(); err != nil {
				fmt.Println("Error during idle handling:", err)
			}

			// Unified idle handling and timeout logic
			if l.opt.timeout == 0 {
				continue // No timeout set, skip this check
			}

			if time.Since(lastActivity) > l.opt.timeout {
				l.Stop()
				break LOOP
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
		if l.onStop != nil {
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
