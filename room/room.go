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

	// msgs is used to send messages to the room
	// and receive messages from the room
	msgs chan []byte

	restartCount int // Counter for loop restarts
}

func NewRoom(processor RoomProcessor, msgProcessor MessageProcessor, opt *RoomConf) *Room {
	room := &Room{
		opt:       opt,
		msgs:      make(chan []byte, opt.maxBufferSize),
		loop:      NewLoop(msgProcessor, opt),
		processor: processor,
	}

	// Set the stop callback
	room.loop.stopCallback(room.stop, room.loopPanic)
	return room
}

func (r *Room) ID() uint64 {
	return r.opt.roomID
}

// Message is used to send messages to the room
// and receive messages from the room
func (r *Room) WriteMessage(msg []byte) error {
	if r.msgs == nil {
		r.msgs = make(chan []byte, r.opt.maxBufferSize)
	}
	if len(r.msgs) >= cap(r.msgs) {
		// if the buffer is full, drop the message
		// and return an error
		return ErrBufferFull
	}
	if !r.running {
		// if the room is not running, drop the message
		// and return an error
		return ErrNotRunning
	}

	r.msgs <- msg
	return nil
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
func (r *Room) Run() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return ErrRunning
	}

	r.running = true

	// Directly start the loop in a goroutine
	go r.loop.Start()

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

	// Reset restart counter
	r.restartCount = 0
}

func (r *Room) loopPanic() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return
	}

	// Increment the restart counter
	r.restartCount++

	if r.restartCount <= r.opt.counter {
		// Attempt to restart the loop after a panic
		go r.loop.Start()
	} else {
		// Stop the room if restart limit is reached
		r.running = false
	}
}

// Stop the room loop and release resources
// Stop the loop and release resources
func (r *Room) Stop() {
	r.stop()

	// remove the stop callback
	r.loop.stopCallback(nil, nil)
	r.loop.Stop()
}
