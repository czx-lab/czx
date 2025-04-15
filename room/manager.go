package room

import (
	"czx/xlog"
	"errors"
	"sync"

	"go.uber.org/zap"
)

var (
	ErrRoomExists   = errors.New("room already exists")
	ErrRoomNotFound = errors.New("room not found")
)

type RoomManager struct {
	wg    sync.WaitGroup
	mu    sync.RWMutex
	rooms map[uint64]*Room
}

// NewRoomManager creates a new RoomManager instance.
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[uint64]*Room),
	}
}

// Add adds a new room to the manager.
func (rm *RoomManager) Add(room *Room) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.rooms[room.ID()]; exists {
		return ErrRoomExists
	}

	rm.rooms[room.ID()] = room

	rm.wg.Add(1)
	go func() {
		defer rm.wg.Done()

		if err := room.Start(); err != nil {
			xlog.Write().Error("Error starting room:", zap.Error(err))
		}
	}()

	return nil
}

// Get retrieves a room by its ID.
func (rm *RoomManager) Get(roomID uint64) (*Room, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		return nil, ErrRoomNotFound
	}

	return room, nil
}

// Num returns the number of rooms managed by the RoomManager.
func (rm *RoomManager) Num() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return len(rm.rooms)
}

// Stop stops all rooms managed by the RoomManager.
// It waits for all rooms to finish processing before returning.
func (rm *RoomManager) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Stop all rooms synchronously.
	for _, room := range rm.rooms {
		room.Stop()
	}

	// Clear the rooms map before waiting for all rooms to stop.
	rm.rooms = make(map[uint64]*Room)

	rm.wg.Wait()
}
