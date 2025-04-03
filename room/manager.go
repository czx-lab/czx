package room

import (
	"errors"
	"fmt"
	"sync"
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

// AddRoom adds a new room to the manager.
func (rm *RoomManager) AddRoom(room *Room) error {
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
			println("Error starting room:", err.Error())
		}
	}()

	return nil
}

// GetRoom retrieves a room by its ID.
func (rm *RoomManager) GetRoom(roomID uint64) (*Room, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		return nil, ErrRoomNotFound
	}

	return room, nil
}

func (rm *RoomManager) RoomNum() int {
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

	fmt.Println("waiting for all rooms to stop")
	rm.wg.Wait()

	// Log after all rooms have stopped.
	fmt.Println("stopped all rooms")
}
