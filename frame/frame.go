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
		Frequency       uint // Frequency of game logic frame processing (in Hz)
		InputDelay      uint // frames to delay input execution
		MaxFutureFrames uint // max frames a message may target in the future
		DelayChangeLead uint // frames of lead time before a coordinated input-delay change takes effect
	}
	FrameLoop struct {
		conf   FrameConf
		mu     sync.RWMutex
		proc   FrameProcessor
		adjust chan struct{} // Channel for adjusting the frequency dynamically

		frameId uint64 // Current frame ID

		// pendingDelay/pendingFrame schedule a coordinated input-delay change:
		// the new delay is applied when frameId reaches pendingFrame, so every
		// client switches at the same frame. pendingFrame == 0 means none pending.
		pendingDelay uint
		pendingFrame uint64

		// Queue of messages for each frame, indexed by frame ID and player ID
		queue map[uint64]map[string]Message
		// synced stores the last frame each player has real input for.
		// Empty inputs filled in for missing messages do not advance it;
		// it marks the resend start point when a player reconnects.
		synced map[string]uint64
		done   chan struct{}
		flag   atomic.Uint32
		once   sync.Once
		wg     sync.WaitGroup
	}
)

func NewFrameLoop(conf FrameConf) *FrameLoop {
	defaultFrameConf(&conf)

	return &FrameLoop{
		conf:   conf,
		adjust: make(chan struct{}, 1), // Add buffer to avoid blocking
		queue:  make(map[uint64]map[string]Message),
		synced: make(map[string]uint64),
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

// AdjustInputDelay schedules a coordinated change of the uniform input delay
// from a measured round-trip time. Frame sync runs every client on the same
// frame, so the new delay takes effect DelayChangeLead frames from now (the
// server applies it when frameId reaches that frame) and the processor is
// notified so it can broadcast the switch to all clients.
func (f *FrameLoop) AdjustInputDelay(rtt time.Duration) {
	f.mu.Lock()

	frameInterval := time.Second / time.Duration(f.conf.Frequency)

	// ceil(RTT/2 / frameInterval) + 1 buffer frame for jitter.
	delay := uint((rtt/2+frameInterval-1)/frameInterval) + 1

	// Back to the currently applied value: cancel any pending switch.
	if delay == f.conf.InputDelay {
		f.pendingFrame = 0
		f.pendingDelay = 0
		f.mu.Unlock()
		return
	}

	// A switch to this delay is already scheduled: keep the original frame so
	// repeated samples do not keep pushing the effective frame out.
	if f.pendingFrame != 0 && f.pendingDelay == delay {
		f.mu.Unlock()
		return
	}

	f.pendingDelay = delay
	f.pendingFrame = f.frameId + uint64(f.conf.DelayChangeLead)

	proc := f.proc
	effective := f.pendingFrame
	f.mu.Unlock()

	if proc != nil {
		proc.OnDelayChange(delay, effective)
	}
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
	f.pendingFrame = 0
	f.pendingDelay = 0

	for playerId := range f.synced {
		f.synced[playerId] = id
	}

	f.queue = make(map[uint64]map[string]Message)

	f.mu.Unlock()
}

// exec processes the current frame using the frame processor.
func (f *FrameLoop) exec() {
	f.mu.Lock()
	f.frameId++

	// Apply a coordinated delay change once its effective frame arrives.
	if f.pendingFrame != 0 && f.frameId >= f.pendingFrame {
		f.conf.InputDelay = f.pendingDelay
		f.pendingFrame = 0
		f.pendingDelay = 0
	}

	// Nothing to execute until the delay window has elapsed.
	if f.frameId <= uint64(f.conf.InputDelay) {
		f.mu.Unlock()
		return
	}

	// Frame sync advances uniformly: every tick executes exactly one frame,
	// InputDelay frames behind the current input frame.
	targetFrame := f.frameId - uint64(f.conf.InputDelay)

	inputs := f.queue[targetFrame]

	frame := Frame{
		FrameID: targetFrame,
		Inputs:  make(map[string]Message),
	}

	// Process inputs for all registered players.
	for playerId := range f.synced {
		if input, ok := inputs[playerId]; ok {
			// Player has input for this frame.
			frame.Inputs[playerId] = input
			// Advance the player's sync point to the executed frame.
			f.synced[playerId] = targetFrame
			continue
		}

		// If no input from the player, create an empty message.
		frame.Inputs[playerId] = Message{
			PlayerID: playerId, FrameID: targetFrame, Timestamp: time.Now(),
		}
	}

	// Remove the processed frame from the queue.
	delete(f.queue, targetFrame)

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
	proc := f.proc
	f.mu.RUnlock()

	for len(f.queue) > 0 {
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

// PlayerIds returns a copy of the current player IDs and the last frame each
// player has real input for. Empty inputs filled in for missing messages do
// not advance this value; it marks the resend start point when a player
// reconnects.
func (f *FrameLoop) PlayerIds() map[string]uint64 {
	f.mu.RLock()
	defer f.mu.RUnlock()

	syncedCopy := make(map[string]uint64, len(f.synced))
	maps.Copy(syncedCopy, f.synced)

	return syncedCopy
}

// RegisterPlayer registers a new player to the frame loop, or resends frames
// missed by a reconnecting player starting from their sync point.
func (f *FrameLoop) RegisterPlayer(playerId string) {
	f.mu.Lock()

	lastFrameId, ok := f.synced[playerId]
	if !ok {
		// New player: initialize the sync point to the current frame
		f.synced[playerId] = f.frameId
		f.mu.Unlock()
		return
	}

	proc := f.proc
	f.mu.Unlock()

	if proc == nil {
		return
	}

	// Resend frames from the player's sync point to catch them up
	proc.Resend(playerId, int(lastFrameId))
}

// DeletePlayer unregisters a player from the frame loop and removes their input queue.
// NOTE: use only for players permanently leaving the room. Disconnected players
// must stay registered so that reconnecting via RegisterPlayer can resend
// missed frames from their sync point; deleting the entry disables that.
func (f *FrameLoop) DeletePlayer(playerId string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.synced, playerId)

	// Remove the player's input from all frames in the queue
	for frameId := range f.queue {
		delete(f.queue[frameId], playerId)
	}
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
	if _, exists := f.synced[in.PlayerID]; !exists {
		return errors.New("player not registered")
	}

	// Check for duplicate messages for the same frame and player
	if frameInputs, ok := f.queue[in.FrameID]; ok {
		if _, ok := frameInputs[in.PlayerID]; ok {
			return errors.New("duplicate message for frame")
		}
	}

	// Reject messages for frames that have already been executed. Frame sync
	// executes frameId-InputDelay each tick, so that is the past-frame boundary.
	executed := uint64(0)
	if f.frameId > uint64(f.conf.InputDelay) {
		executed = f.frameId - uint64(f.conf.InputDelay)
	}
	if in.FrameID <= executed {
		return errors.New("message for past frame")
	}

	// Reject messages too far ahead to prevent unbounded queue growth
	if in.FrameID > f.frameId+uint64(f.conf.MaxFutureFrames) {
		return errors.New("message too far in the future")
	}

	if f.queue[in.FrameID] == nil {
		f.queue[in.FrameID] = make(map[string]Message)
	}

	f.queue[in.FrameID][in.PlayerID] = in

	return nil
}

func defaultFrameConf(conf *FrameConf) {
	if conf.Frequency == 0 {
		conf.Frequency = frequency
	}

	if conf.InputDelay == 0 {
		conf.InputDelay = 2
	}

	if conf.MaxFutureFrames == 0 {
		conf.MaxFutureFrames = 60 // e.g. 2 seconds at 30Hz
	}

	if conf.DelayChangeLead == 0 {
		conf.DelayChangeLead = 30 // e.g. 1 second at 30Hz
	}
}

var _ FrameFace = (*FrameLoop)(nil)
