package room

import (
	"errors"
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
		defer func() {
			rm.mu.Lock()
			delete(rm.rooms, room.ID())
			rm.mu.Unlock()

			rm.wg.Done()
		}()

		if err := room.Start(); err != nil {
			// Handle error (e.g., log it)
			// For now, just print it
			println("Error starting room:", err.Error())
		}
	}()

	return nil
}

// RemoveRoom removes a room from the manager by its ID.
func (rm *RoomManager) RemoveRoom(roomID uint64) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.rooms[roomID]; !exists {
		return ErrRoomNotFound
	}

	delete(rm.rooms, roomID)
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

// ListRooms returns a list of all room IDs.
func (rm *RoomManager) ListRooms() []uint64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	ids := make([]uint64, 0, len(rm.rooms))
	for id := range rm.rooms {
		ids = append(ids, id)
	}
	return ids
}

// Stop stops all rooms managed by the RoomManager.
// It waits for all rooms to finish processing before returning.
func (rm *RoomManager) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for _, room := range rm.rooms {
		room.Stop()
	}
	rm.wg.Wait()
	rm.rooms = make(map[uint64]*Room)
}
