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
	// wg is used to wait for the loop to finish
	wg sync.WaitGroup
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
	return &Room{
		opt:       opt,
		loop:      NewLoop(msgProcessor, opt),
		processor: processor,
	}
}

func (r *Room) ID() uint64 {
	return r.opt.roomID
}

// Message is used to send messages to the room
// and receive messages from the room
func (r *Room) WriteMessage(msg []byte) error {
	if !r.running {
		return ErrNotRunning
	}

	// Push the message to the loop
	return r.loop.Push(msg)
}

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

func (r *Room) Leave(playerID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.players, playerID)

	return r.processor.Leave(playerID)
}

func (r *Room) Run() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return ErrRunning
	}

	r.running = true
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		if err := r.loop.Start(); err != nil {
			r.mu.Lock()
			defer r.mu.Unlock()

			// If the loop fails, we need to stop the room
			// and set the running state to false
			r.running = false
		}
	}()

	return nil
}

func (r *Room) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return
	}
	r.running = false

	r.loop.Stop()
	r.wg.Wait()
}
