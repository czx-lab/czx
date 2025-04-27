package frame

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/czx-lab/czx/xlog"
	"github.com/panjf2000/ants/v2"
)

const (
	// game logic frame processing frequency
	frequency          uint = 30
	maxQueueSize       uint = 100 // Maximum input queue size per player
	heartbeatFrequency uint = 5   // Heartbeat frequency in seconds
	poolSize           int  = 20  // Size of the worker pool

	// LoopTypeNormal indicates a normal game loop
	LoopTypeNormal = "normal"
	// LoopTypeSync indicates a synchronous game loop
	LoopTypeSync = "sync"
)

type (
	LoopConf struct {
		// frequency of game logic frame processing
		Frequency          uint
		HeartbeatFrequency uint
		MaxQueueSize       uint
		LoopType           string // Type of loop, e.g., "normal" or "sync"
		PoolSize           int    // Size of the worker pool
	}
	Loop struct {
		mu   sync.Mutex
		quit chan struct{}
		// Channel for adjusting the frequency dynamically
		adjust chan struct{} // Channel for adjusting the frequency dynamically
		conf   LoopConf
		// Processor interface for processing game logic
		frameProc  FrameProcessor
		normalProc NormalProcessor
		// Current frame being processed
		current Frame
		// Input queue for each player
		inFrameQueue map[string][]Message
		// Channel for normal processing
		inNormalQueue chan Message
		// Handler for empty processing
		emptyHandler func()
		workpool     *ants.Pool
	}
)

func NewLoop(conf LoopConf) (*Loop, error) {
	defaultConf(&conf)

	opts := []ants.Option{
		ants.WithNonblocking(true),
		ants.WithPreAlloc(true),
		ants.WithDisablePurge(true),
	}
	workerpool, err := ants.NewPool(conf.PoolSize, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker pool: %w", err)
	}

	return &Loop{
		conf:          conf,
		quit:          make(chan struct{}),
		adjust:        make(chan struct{}, 1), // Add buffer to avoid blocking
		inFrameQueue:  make(map[string][]Message),
		inNormalQueue: make(chan Message, conf.MaxQueueSize),
		workpool:      workerpool,
	}, nil
}

func (l *Loop) WithEmptyHandler(handler func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.emptyHandler = handler
}

// WithNormalProc sets the normal processor for the loop.
// It can only be set for normal loop type.
func (l *Loop) WithNormalProc(normalProc NormalProcessor) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.conf.LoopType != LoopTypeNormal {
		return errors.New("normal processor can only be set for normal loop type")
	}

	l.normalProc = normalProc
	return nil
}

// WithFrameProc sets the frame processor for the loop.
// It can only be set for sync loop type.
func (l *Loop) WithFrameProc(frameProc FrameProcessor) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.conf.LoopType != LoopTypeSync {
		return errors.New("frame processor can only be set for sync loop type")
	}

	l.frameProc = frameProc

	return nil
}

// Start starts the game loop, processing frames at the specified frequency.
// It uses a ticker to trigger the processing at regular intervals.
func (l *Loop) Start() {
	if l.workpool.IsClosed() {
		l.workpool.Reboot()
	}

	frequency := time.Second / time.Duration(l.conf.Frequency)
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	go func() {
		lastHeartbeat := time.Now() // Track the last heartbeat time

		for {
			select {
			case <-l.quit:
				return
			case <-ticker.C:
				l.mu.Lock()

				switch l.conf.LoopType {
				case LoopTypeNormal:
					// Process normal messages
					l.processNormal()
				case LoopTypeSync:
					// Process the current frame and its inputs
					l.processFrame()
				}

				l.mu.Unlock()

				// Check for heartbeat
				lastHeartbeat = l.checkHeartbeat(lastHeartbeat)
			}
		}
	}()

	// Monitor and adjust the frequency dynamically
	l.monitorFrequency(ticker)

	// Close the frame processor when the loop stops
	switch l.conf.LoopType {
	case LoopTypeNormal:
		l.normalProc.Close()
	case LoopTypeSync:
		l.frameProc.Close()
	}
}

// monitorFrequency monitors and adjusts the ticker frequency dynamically.
func (l *Loop) monitorFrequency(ticker *time.Ticker) {
	for {
		select {
		case <-l.quit:
			return
		case <-l.adjust:
			l.mu.Lock()
			frequency := time.Second / time.Duration(l.conf.Frequency)
			ticker.Reset(frequency)

			xlog.Write().Sugar().Debugf("Frequency adjusted to %d Hz", l.conf.Frequency)

			l.mu.Unlock()
		}
	}
}

// Process normal messages from the input queue.
// It handles the processing of messages in the normal loop type.
func (l *Loop) processNormal() {
	select {
	case message := <-l.inNormalQueue:
		l.workpool.Submit(func() {
			defer func() {
				if r := recover(); r != nil {
					xlog.Write().Sugar().Errorf("Recovered from panic in processor: %v", r)
				}
			}()

			l.normalProc.Process(message)
		})
	default:
		if l.emptyHandler != nil {
			l.emptyHandler()
		}
	}
}

// process processes the current frame and its inputs, and prepares the next frame.
// It handles the input queue for each player and updates the current frame accordingly.
func (l *Loop) processFrame() {
	if len(l.inFrameQueue) == 0 {
		if l.emptyHandler != nil {
			// Call the empty handler if there are no messages to process
			l.emptyHandler()
		}
		return
	}

	l.workpool.Submit(func() {
		defer func() {
			if r := recover(); r != nil {
				xlog.Write().Sugar().Errorf("Recovered from panic in processor: %v", r)
			}
		}()

		// Process the current frame and its inputs
		for playerID, messages := range l.inFrameQueue {
			if len(messages) > 0 {
				// Safely process the first message and update the queue
				if l.current.Inputs == nil {
					l.current.Inputs = make(map[string]Message)
				}

				l.current.Inputs[playerID] = messages[0]
				l.inFrameQueue[playerID] = messages[1:]
			} else {
				// Ensure the player's input map is cleared if no messages are left
				delete(l.current.Inputs, playerID)
			}

			// If the queue is empty, remove the player from the inFrameQueue
			delete(l.inFrameQueue, playerID)
		}

		// Process the current frame using the processor
		l.frameProc.Process(l.current)

		// Prepare the next frame
		l.current = Frame{
			FrameID: l.current.FrameID + 1,
			Inputs:  make(map[string]Message),
		}
	})
}

// Handle heartbeat if the interval has passed
func (l *Loop) checkHeartbeat(lastHeartbeat time.Time) time.Time {
	frequency := time.Second * time.Duration(l.conf.HeartbeatFrequency)
	if time.Since(lastHeartbeat) >= frequency {
		var handler func()
		switch l.conf.LoopType {
		case LoopTypeNormal:
			handler = l.normalProc.HandleIdle
		case LoopTypeSync:
			handler = l.frameProc.HandleIdle
		}

		handler()

		lastHeartbeat = time.Now() // Update the last heartbeat time
	}

	if time.Since(lastHeartbeat) >= 2*frequency {
		xlog.Write().Sugar().Warnf("Heartbeat timeout detected! Last heartbeat was %v ago", time.Since(lastHeartbeat))
	}

	return lastHeartbeat
}

// Stop stops the game loop and releases any resources.
// It closes the quit channel to signal the loop to stop.
func (l *Loop) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	select {
	case <-l.quit:
		// Already stopped
	default:
		close(l.quit) // Signal all goroutines to stop
	}

	// Wait for all tasks to finish
	l.workpool.Waiting()

	// Release the worker pool
	l.workpool.Release()

	// Note: No need to close the `adjust` channel.
	// It is only used for signaling and will be garbage collected when `Loop` is no longer referenced.
}

// Receive receives a message and adds it to the input queue for processing.
// It enforces a maximum queue size for each player in the sync loop type.
func (l *Loop) Receive(in Message) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.conf.LoopType == LoopTypeNormal {
		if len(l.inNormalQueue) >= cap(l.inNormalQueue) {
			return errors.New("input queue is full")
		}

		l.inNormalQueue <- in
		return nil
	}

	// Enforce maximum queue size
	if len(l.inFrameQueue[in.PlayerID]) >= int(l.conf.MaxQueueSize) {
		return fmt.Errorf("input queue for player %s is full, dropping message", in.PlayerID)
	}

	l.inFrameQueue[in.PlayerID] = append(l.inFrameQueue[in.PlayerID], in)

	return nil
}

// This method allows you to change the frequency of the game loop at runtime
func (l *Loop) Frequency(frequency uint) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if frequency == 0 {
		return errors.New("frequency must be greater than 0")
	}

	xlog.Write().Sugar().Debugf("Updating frequency from %d to %d", l.conf.Frequency, frequency)

	l.conf.Frequency = frequency

	// Non-blocking send to adjust channel
	select {
	case l.adjust <- struct{}{}:
	default:
		// If the channel is full, skip sending to avoid blocking
		xlog.Write().Sugar().Warn("Frequency adjustment signal already pending")
	}

	return nil
}

func defaultConf(conf *LoopConf) {
	if conf.Frequency == 0 {
		conf.Frequency = frequency
	}
	if conf.MaxQueueSize == 0 {
		conf.MaxQueueSize = maxQueueSize
	}
	if len(conf.LoopType) == 0 {
		conf.LoopType = LoopTypeNormal
	}
	if conf.HeartbeatFrequency == 0 {
		conf.HeartbeatFrequency = heartbeatFrequency
	}
	if conf.PoolSize == 0 {
		conf.PoolSize = poolSize
	}
}
