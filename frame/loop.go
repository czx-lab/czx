package frame

import (
	"errors"
	"sync"
	"time"

	"github.com/czx-lab/czx/xlog"
)

const (
	// game logic frame processing frequency
	frequency    uint = 30
	maxQueueSize uint = 100 // Maximum input queue size per player

	// LoopTypeNormal indicates a normal game loop
	LoopTypeNormal = "normal"
	// LoopTypeFream indicates a synchronous game loop
	LoopTypeFream = "frame"
)

type (
	LoopConf struct {
		// frequency of game logic frame processing
		Frequency    uint
		LoopType     string // Type of loop, e.g., "normal" or "frame"
		MaxQueueSize uint   // Maximum input queue size per player
	}

	Loop struct {
		mu   sync.RWMutex
		quit chan struct{}
		// Channel for adjusting the frequency dynamically
		adjust chan struct{} // Channel for adjusting the frequency dynamically
		conf   LoopConf
		// Processor interface for processing game logic
		frameProc  FrameProcessor
		normalProc NormalProcessor
		frameId    uint64 // Current frame ID
		// Input queue for each player
		frameQueue map[string]Message
		playerIds  map[string]uint
		// Channel for normal processing
		normalQueue chan Message
	}
)

func NewLoop(conf LoopConf) (*Loop, error) {
	defaultConf(&conf)

	loop := &Loop{
		conf:   conf,
		quit:   make(chan struct{}),
		adjust: make(chan struct{}, 1), // Add buffer to avoid blocking
	}
	if conf.LoopType == LoopTypeNormal {
		loop.normalQueue = make(chan Message, conf.MaxQueueSize)
	}
	if conf.LoopType == LoopTypeFream {
		loop.frameQueue = make(map[string]Message)
		loop.playerIds = make(map[string]uint)
	}

	return loop, nil
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

	if l.conf.LoopType != LoopTypeFream {
		return errors.New("frame processor can only be set for sync loop type")
	}

	l.frameProc = frameProc

	return nil
}

// Start starts the game loop, processing frames at the specified frequency.
// It uses a ticker to trigger the processing at regular intervals.
func (l *Loop) Start() {
	frequency := time.Second / time.Duration(l.conf.Frequency)
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-l.quit:
				return
			case <-ticker.C:
				switch l.conf.LoopType {
				case LoopTypeNormal:
					// Process normal messages
					l.processN()
				case LoopTypeFream:
					// Process the current frame and its inputs
					l.processF()
				}
			}
		}
	}()

	// Monitor and adjust the frequency dynamically
	l.monitorFrequency(ticker)

	// Close the frame processor when the loop stops
	switch l.conf.LoopType {
	case LoopTypeNormal:
		if l.normalProc != nil {
			l.normalProc.OnClose()
		}
	case LoopTypeFream:
		if l.frameProc != nil {
			l.frameProc.OnClose()
		}
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
func (l *Loop) processN() {
	// Process all available messages without blocking
	for {
		select {
		case message := <-l.normalQueue:
			l.normalProc.Process(message)
		default:
			// No more messages available, return
			return
		}
	}
}

// process processes the current frame and its inputs, and prepares the next frame.
// It handles the input queue for each player and updates the current frame accordingly.
func (l *Loop) processF() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.frameId++

	// Create a new frame with the current frame ID
	frame := Frame{
		FrameID: l.frameId,
		Inputs:  make(map[string]Message),
	}

	// Process inputs for all registered players
	for playerId := range l.playerIds {
		if input, ok := l.frameQueue[playerId]; ok {
			// Player has input for this frame
			frame.Inputs[playerId] = input
			// Update the last processed frame ID for the player
			l.playerIds[playerId] = uint(input.FrameID)
			continue
		}
		// If no input from the player, create an empty message
		emptyMessage := Message{
			PlayerID:  playerId,
			FrameID:   int(l.frameId),
			Timestamp: time.Now(),
		}
		frame.Inputs[playerId] = emptyMessage
	}

	// Clear the frame queue for the next frame
	l.frameQueue = make(map[string]Message)

	// Process the frame
	l.frameProc.Process(frame)
}

// Stop stops the game loop and releases any resources.
// It closes the quit channel to signal the loop to stop.
func (l *Loop) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	select {
	case <-l.quit:
		// Already stopped
		return
	default:
		close(l.quit) // Signal all goroutines to stop
	}

	// Close the normal queue if it exists
	if l.normalQueue != nil {
		close(l.normalQueue)
	}

	// Note: No need to close the `adjust` channel.
	// It is only used for signaling and will be garbage collected when `Loop` is no longer referenced.
}

// Write writes an input message to the loop's input queue.
// It enforces the maximum queue size and handles stale messages.
func (l *Loop) Write(in Message) error {
	if l.conf.LoopType == LoopTypeNormal {
		select {
		case l.normalQueue <- in:
			return nil
		default:
			return errors.New("input queue is full")
		}
	}

	// For frame-based processing
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if player is registered
	if _, exists := l.playerIds[in.PlayerID]; !exists {
		return errors.New("player not registered")
	}

	// Check for stale messages
	if existing, ok := l.frameQueue[in.PlayerID]; ok && existing.FrameID >= in.FrameID {
		return errors.New("stale or duplicate message")
	}

	// Only accept messages for current or future frames
	if in.FrameID <= int(l.frameId) {
		return errors.New("message for past frame")
	}

	l.frameQueue[in.PlayerID] = in
	return nil
}

// RegisterPlayer registers a new player in the loop.
// It initializes the player's last processed frame ID.
func (l *Loop) RegisterPlayer(playerId string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	lastFrameId, ok := l.playerIds[playerId]
	if !ok {
		// Initialize the last processed frame ID for the player
		l.playerIds[playerId] = uint(l.frameId)
		return
	}

	// Resend the last processed frame to the player
	l.frameProc.Resend(playerId, int(lastFrameId))
}

// UnregisterPlayer removes a player from the loop.
// It deletes the player's input queue and last processed frame ID.
func (l *Loop) UnregisterPlayer(playerId string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.playerIds, playerId)
	delete(l.frameQueue, playerId)
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
	if len(conf.LoopType) == 0 {
		conf.LoopType = LoopTypeNormal
	}
	if conf.MaxQueueSize == 0 {
		conf.MaxQueueSize = maxQueueSize
	}
}
