package room

import (
	"errors"
	"sync"
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
	mu sync.Mutex
	// loop is used to run the room loop
	// and send messages to Kafka
	// and receive messages from Kafka
	loop *Loop
	// running is used to check if the room is running
	// and to prevent multiple calls to Run()
	running bool

	// players is used to keep track of the players in the room
	// and to prevent multiple calls to Join()
	players map[uint64]struct{}

	// rpcClient is used to send messages to the room
	// and receive messages from the room
	processor RoomProcessor
}

func NewRoom(processor RoomProcessor, msgProcessor MessageProcessor, opt *RoomConf) *Room {
	room := &Room{
		opt:       opt,
		loop:      NewLoop(msgProcessor, opt),
		processor: processor,
		players:   make(map[uint64]struct{}),
	}

	// Set the stop callback
	room.loop.stopCallback(room.stop)
	return room
}

func (r *Room) ID() uint64 {
	return r.opt.roomID
}

// Message is used to send messages to the room
// and receive messages from the room
func (r *Room) WriteMessage(msg Message) error {
	if !r.running {
		// if the room is not running, drop the message
		// and return an error
		return ErrNotRunning
	}

	return r.loop.Push(msg)
}

// Join is used to add a player to the room
// and to prevent multiple calls to Join()
func (r *Room) Join(playerID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.players[playerID]; ok {
		return ErrMaxPlayer
	}
	if len(r.players) > r.opt.maxPlayer {
		return ErrRoomFull
	}

	r.players[playerID] = struct{}{}

	return r.processor.Join(playerID)
}

// Leave is used to remove a player from the room
// and to prevent multiple calls to Leave()
func (r *Room) Leave(playerID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.players, playerID)

	return r.processor.Leave(playerID)
}

// Start the room loop and process messages
// in a separate goroutine. It will run until the loop is stopped or an error occurs.
func (r *Room) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return ErrRunning
	}

	r.running = true

	return r.loop.Start()
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
	r.stop()

	// remove the stop callback
	r.loop.stopCallback(nil)
	r.loop.Stop()
}
