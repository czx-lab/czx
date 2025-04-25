package room

import (
	"errors"
	"sync"

	"github.com/czx-lab/czx/frame"
)

var (
	ErrRoomFull   = errors.New("room is full")
	ErrMaxPlayer  = errors.New("max player count exceeded")
	ErrBufferFull = errors.New("buffer is full")
	ErrNotRunning = errors.New("room is not running")
	ErrRunning    = errors.New("room is already running")
)

type Room struct {
	opt *RoomConf

	// mu is used to protect the room state and the loop
	mu sync.RWMutex
	// loop is used to run the room loop
	// and send messages to Kafka
	// and receive messages from Kafka
	loop *frame.Loop
	// running is used to check if the room is running
	// and to prevent multiple calls to Run()
	running bool

	// players is used to keep track of the players in the room
	// and to prevent multiple calls to Join()
	players map[string]struct{}

	// rpcClient is used to send messages to the room
	// and receive messages from the room
	processor RoomProcessor

	// data is used to store the room data
	data any
}

func NewRoom(processor RoomProcessor, opt *RoomConf) *Room {
	loopConf := frame.LoopConf{
		Frequency:          uint(opt.frequency),
		HeartbeatFrequency: uint(opt.heartbeatFrequency),
		LoopType:           opt.loopType,
		MaxQueueSize:       uint(opt.maxBufferSize),
	}
	room := &Room{
		opt:       opt,
		loop:      frame.NewLoop(loopConf),
		processor: processor,
		players:   make(map[string]struct{}),
	}

	return room
}

func (r *Room) ID() string {
	return r.opt.roomID
}

// WithData is used to set the room data
func (r *Room) WithData(data any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data = data
}

// GetData is used to get the room data
func (r *Room) Data() any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.data
}

// Message is used to send messages to the room
// and receive messages from the room
func (r *Room) WriteMessage(msg frame.Message) error {
	if !r.running {
		// if the room is not running, drop the message
		// and return an error
		return ErrNotRunning
	}

	return r.loop.Receive(msg)
}

// Join is used to add a player to the room
// and to prevent multiple calls to Join()
func (r *Room) Join(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.players[playerID]; ok {
		return ErrMaxPlayer
	}
	if len(r.players) >= r.opt.maxPlayer {
		return ErrRoomFull
	}

	r.players[playerID] = struct{}{}

	if err := r.processor.Join(playerID); err != nil {
		delete(r.players, playerID)
		// If the player is already in the room, remove it
		return err
	}

	return nil
}

// Leave is used to remove a player from the room
// and to prevent multiple calls to Leave()
func (r *Room) Leave(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.players, playerID)

	return r.processor.Leave(playerID)
}

// Check if the room is running
// Returns true if the room is running, false otherwise
func (r *Room) Status() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.running
}

// Start the room loop and process messages
// in a separate goroutine. It will run until the loop is stopped or an error occurs.
func (r *Room) Start() error {
	// Remove the lock here to avoid potential deadlocks
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return ErrRunning
	}

	r.running = true
	r.mu.Unlock()

	// Start the loop without holding the lock
	r.loop.Start()

	return nil
}

// stop the room loop and release resources
func (r *Room) stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return
	}
	r.running = false
}

// Stop the room loop and release resources
// Stop the loop and release resources
func (r *Room) Stop() {
	// Check if the room is running
	// and stop the loop if it is
	// This is to prevent multiple calls to Stop()
	status := r.Status()
	r.stop()

	if !status {
		return
	}

	// Stop the loop synchronously to ensure proper cleanup.
	r.loop.Stop()
}

// Number of players in the room
// Returns the number of players in the room
func (r *Room) Num() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.players)
}

// Check if the player is in the room
// Returns true if the player is in the room, false otherwise
func (r *Room) Has(playerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.players[playerID]
	return ok
}

// Returns a slice of player IDs in the room
func (r *Room) Players() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.players) == 0 {
		return nil
	}

	var players []string
	for player := range r.players {
		players = append(players, player)
	}

	return players
}
