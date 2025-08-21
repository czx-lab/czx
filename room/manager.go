package room

import (
	"errors"
	"sync"

	"github.com/czx-lab/czx/container/cmap"
	"github.com/czx-lab/czx/xlog"

	"go.uber.org/zap"
)

var (
	ErrRoomExists   = errors.New("room already exists")
	ErrRoomNotFound = errors.New("room not found")
)

type RoomManager struct {
	wg    sync.WaitGroup
	mu    sync.RWMutex
	rooms *cmap.CMap[string, *Room]
}

// NewRoomManager creates a new RoomManager instance.
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: cmap.New[string, *Room](),
	}
}

// Add adds a new room to the manager.
func (rm *RoomManager) Add(room *Room) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.rooms.Has(room.ID()) {
		return ErrRoomExists
	}

	rm.rooms.Set(room.ID(), room)

	rm.wg.Add(1)
	go func() {
		defer rm.wg.Done()

		if err := room.Start(); err != nil {
			xlog.Write().Error("Error starting room:", zap.Error(err))
		}
	}()

	return nil
}

// Remove removes a room from the manager by its ID.
// It stops the room and waits for it to finish processing before removing it.
func (rm *RoomManager) Remove(roomID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, exists := rm.rooms.Get(roomID)
	if !exists {
		return ErrRoomNotFound
	}

	room.Stop()

	rm.rooms.Delete(roomID)

	return nil
}

// Get retrieves a room by its ID.
func (rm *RoomManager) Get(roomID string) (*Room, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, exists := rm.rooms.Get(roomID)
	if !exists {
		return nil, ErrRoomNotFound
	}

	return room, nil
}

// Check if the room exists in the manager.
func (rm *RoomManager) Has(roomID string) bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.rooms.Has(roomID)
}

// Num returns the number of rooms managed by the RoomManager.
func (rm *RoomManager) Num() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.rooms.Len()
}

// Stop stops all rooms managed by the RoomManager.
// It waits for all rooms to finish processing before returning.
func (rm *RoomManager) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Stop all rooms synchronously.
	rm.rooms.Iterator(func(_ string, room *Room) bool {
		room.Stop()
		return true
	})

	// Clear the rooms map before waiting for all rooms to stop.
	rm.rooms.Clear()

	rm.wg.Wait()
}

// Iterate over all rooms and apply the function.
// This function is not thread-safe, so it should be called with the room manager locked.
// It is recommended to use this function for read-only operations on rooms.
func (rm *RoomManager) Range(fn func(*Room)) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rm.rooms.Iterator(func(_ string, room *Room) bool {
		fn(room)
		return true
	})
}

// Number of players in the room
// Returns a map where the key is the room ID and the value is the number of players in that room.
// This function is not thread-safe, so it should be called with the room manager locked.
func (rm *RoomManager) RoomsPlayerNum() map[string]int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	nums := make(map[string]int)
	rm.rooms.Iterator(func(roomID string, room *Room) bool {
		nums[room.ID()] = room.players.Len()
		return true
	})

	return nums
}

// Returns a slice of all rooms managed by the RoomManager.
func (rm *RoomManager) Rooms() []*Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rooms := make([]*Room, 0, rm.rooms.Len())
	rm.rooms.Iterator(func(_ string, room *Room) bool {
		rooms = append(rooms, room)
		return true
	})

	return rooms
}

// Get the players in the room
func (rm *RoomManager) Players(roomId string) ([]string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, ok := rm.rooms.Get(roomId)
	if !ok {
		return nil, ErrRoomNotFound
	}

	return room.Players(), nil
}
