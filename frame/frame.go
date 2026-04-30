package frame

import (
	"context"
	"errors"
	"maps"
	"sync"
	"sync/atomic"
	"time"
)

type (
	FrameConf struct {
		Frequency uint // Frequency of game logic frame processing (in Hz)
	}
	FrameLoop struct {
		conf   FrameConf
		mu     sync.RWMutex
		proc   FrameProcessor
		adjust chan struct{} // Channel for adjusting the frequency dynamically

		frameId uint64 // Current frame ID

		// Input queue for each player
		queue map[string][]Message
		ids   map[string]uint // Last processed frame ID for each player
		done  chan struct{}
		flag  atomic.Uint32
		once  sync.Once
		wg    sync.WaitGroup
	}
)

func NewFrameLoop(conf FrameConf) *FrameLoop {
	defaultFrameConf(&conf)

	return &FrameLoop{
		conf:   conf,
		adjust: make(chan struct{}, 1), // Add buffer to avoid blocking
		queue:  make(map[string][]Message),
		ids:    make(map[string]uint),
		done:   make(chan struct{}),
	}
}

// Frequency implements [LoopFace].
func (f *FrameLoop) Frequency(frequency uint) error {
	if frequency == 0 {
		return errors.New("frequency must be greater than 0")
	}

	f.mu.Lock()
	f.conf.Frequency = frequency
	f.mu.Unlock()

	select {
	case f.adjust <- struct{}{}:
	default:
	}

	return nil
}

// WithProc sets the frame processor for the frame loop.
func (f *FrameLoop) WithProc(proc FrameProcessor) *FrameLoop {
	f.mu.Lock()
	f.proc = proc
	f.mu.Unlock()

	return f
}

// Start implements [LoopFace].
func (f *FrameLoop) Start(ctx context.Context) error {
	// Ensure that the loop can only be started once
	if !f.flag.CompareAndSwap(0, flagStarted) {
		return errors.New("loop already started")
	}

	defer f.flag.Store(0)

	f.wg.Add(1)
	defer f.wg.Done()

	frequency := time.Second / time.Duration(f.conf.Frequency)
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	f.mu.RLock()
	if f.proc == nil {
		f.mu.RUnlock()
		return errors.New("processor is not set")
	}
	f.mu.RUnlock()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-f.done:
			return nil
		case <-ticker.C:
			// Only execute the frame processing if the loop is not paused
			if f.flag.Load() == flagPaused {
				continue
			}

			f.exec()
		case <-f.adjust:
			f.mu.RLock()
			frequency = time.Second / time.Duration(f.conf.Frequency)
			f.mu.RUnlock()
			ticker.Reset(frequency)
		}
	}
}

// FrameId returns the current frame ID.
func (f *FrameLoop) FrameId() uint64 {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.frameId
}

// Reset resets the frame ID to a specific value, typically used for synchronization.
func (f *FrameLoop) Reset(id uint64) {
	f.mu.Lock()
	f.frameId = id

	for playerId := range f.ids {
		f.ids[playerId] = uint(id)
	}

	f.queue = make(map[string][]Message)

	f.mu.Unlock()
}

// exec processes the current frame using the frame processor.
func (f *FrameLoop) exec() {
	f.mu.Lock()
	f.frameId++

	// Create a new frame with the current frame ID
	frame := Frame{
		FrameID: f.frameId,
		Inputs:  make(map[string][]Message),
	}

	// Process inputs for all registered players
	for playerId := range f.ids {
		if input, ok := f.queue[playerId]; ok {
			// Player has input for this frame
			frame.Inputs[playerId] = input
			// Update the last processed frame ID for the player
			f.ids[playerId] = uint(input[len(input)-1].FrameID)
			continue
		}
		// If no input from the player, create an empty message
		emptyMessage := []Message{
			{
				PlayerID:  playerId,
				FrameID:   f.frameId,
				Timestamp: time.Now(),
			},
		}
		frame.Inputs[playerId] = emptyMessage
	}

	// Clear the frame queue for the next frame
	f.queue = make(map[string][]Message)

	proc := f.proc
	f.mu.Unlock()

	if proc != nil {
		proc.Process(frame)
	}
}

// Stop implements [LoopFace].
func (f *FrameLoop) Stop() {
	f.once.Do(func() {
		close(f.done)

		// Wait for the loop to finish processing before stopping
		f.wg.Wait()

		f.stop()
	})
}

// stop processes any remaining frames and calls the OnClose method of the processor.
func (f *FrameLoop) stop() {
	f.mu.RLock()
	hasData := len(f.queue) > 0
	proc := f.proc
	f.mu.RUnlock()

	if hasData {
		f.exec()
	}

	if proc == nil {
		return
	}

	proc.OnClose()
}

// Pause implements [FrameFace].
func (f *FrameLoop) Pause() bool {
	return f.flag.CompareAndSwap(flagStarted, flagPaused)
}

// Resume implements [FrameFace].
func (f *FrameLoop) Resume() bool {
	return f.flag.CompareAndSwap(flagPaused, flagStarted)
}

// PlayerIds returns a copy of the current player IDs and their last processed frame IDs.
func (f *FrameLoop) PlayerIds() map[string]uint {
	f.mu.RLock()
	defer f.mu.RUnlock()

	idsCopy := make(map[string]uint, len(f.ids))
	maps.Copy(idsCopy, f.ids)

	return idsCopy
}

// RegisterPlayer registers a new player to the frame loop and resends the last processed frame if necessary.
func (f *FrameLoop) RegisterPlayer(playerId string) {
	f.mu.Lock()

	lastFrameId, ok := f.ids[playerId]
	if !ok {
		// Initialize the last processed frame ID for the player
		f.ids[playerId] = uint(f.frameId)
		f.mu.Unlock()
		return
	}

	proc := f.proc
	f.mu.Unlock()

	if proc == nil {
		return
	}

	// Resend the last processed frame to the player
	proc.Resend(playerId, int(lastFrameId))
}

// DeletePlayer unregisters a player from the frame loop and removes their input queue.
func (f *FrameLoop) DeletePlayer(playerId string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.ids, playerId)
	delete(f.queue, playerId)
}

// Write implements [LoopFace].
func (f *FrameLoop) Write(in Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	select {
	case <-f.done:
		return errors.New("loop is closed")
	default:
	}

	// Check if player is registered
	if _, exists := f.ids[in.PlayerID]; !exists {
		return errors.New("player not registered")
	}

	// Check for stale messages
	if existing, ok := f.queue[in.PlayerID]; ok && len(existing) > 0 {
		if existing[len(existing)-1].FrameID >= in.FrameID {
			return errors.New("stale or duplicate message")
		}
	}

	// Only accept messages for current or future frames
	if in.FrameID <= f.frameId {
		return errors.New("message for past frame")
	}

	f.queue[in.PlayerID] = append(f.queue[in.PlayerID], in)

	return nil
}

func defaultFrameConf(conf *FrameConf) {
	if conf.Frequency == 0 {
		conf.Frequency = frequency
	}
}

var _ FrameFace = (*FrameLoop)(nil)
