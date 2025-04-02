package room

import (
	"time"
)

type Loop struct {
	opt       *RoomConf
	quit      chan struct{}
	processor MessageProcessor
	// buffered channel to hold messages
	msgs chan []byte
	// callback to be called when the loop stops
	onStop func()
}

func NewLoop(processor MessageProcessor, opt *RoomConf) *Loop {
	return &Loop{
		opt:       opt,
		processor: processor,
		quit:      make(chan struct{}),
		msgs:      make(chan []byte, opt.maxBufferSize),
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

	timeoutTimer := time.NewTimer(l.opt.timeout)

LOOP:
	for {
		select {
		case <-timeoutTimer.C:
			l.Stop()
		case <-ticker.C:
			// Process one message per tick if available
			select {
			case msg := <-l.msgs:
				if err := l.processor.Process(msg); err != nil {
					l.Stop()
				}
			default:
				// No message to process, perform idle handling
				if err := l.processor.HandleIdle(); err != nil {
					l.Stop()
				}
			}
		case <-l.quit:
			break LOOP
		}
	}

	if l.onStop != nil {
		l.onStop()
	}

	return l.processor.Close()
}

// Stop the loop and close the quit channel
func (l *Loop) Stop() {
	close(l.quit)
	if l.onStop != nil {
		l.onStop() // Notify via callback
	}
}

// Push is used to send messages to the room
// and receive messages from the room
func (l *Loop) Push(msg []byte) error {
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
