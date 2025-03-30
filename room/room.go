package room

import (
	"errors"
	"sync"
)

var (
	ErrRoomFull  = errors.New("room is full")
	ErrMaxPlayer = errors.New("max player count exceeded")
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

	players map[uint64]struct{}
}

func NewRoom(opt *RoomConf) *Room {
	return &Room{
		opt:  opt,
		loop: NewLoop(opt),
	}
}

func (r *Room) ID() uint64 {
	return r.opt.roomID
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

	return nil
}

func (r *Room) Leave(playerID uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.players, playerID)

	// todo:: 通知房间内玩家
}

func (r *Room) Run() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return
	}

	r.running = true
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		r.loop.Start()
	}()
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
