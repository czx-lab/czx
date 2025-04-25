package match

import (
	"container/list"
	"errors"
	"sync"
)

const defaultQueueSize = 1000

var (
	ErrMaxQueuePlayer = errors.New("max queue player")
	ErrAlreadyMatch   = errors.New("macth is already waiting")
	// Global queue shared by all Match instances.
	localQueue *list.List
	queueMutex sync.RWMutex
)

// Initialize the global queue with a default size.
func init() {
	localQueue = list.New()
}

type (
	// MatchStatus represents the status of a match.
	MatchStatus uint
	Match       struct {
		// Mutex to ensure thread-safe operations.
		mu sync.RWMutex
		// StatusMap maps matchID to its current status.
		statusMap map[string]struct{}
		// Processor is an interface for processing match logic.
		processor MatchProcessor
	}
)

func NewMatch(processor MatchProcessor) *Match {
	return &Match{
		processor: processor,
		statusMap: make(map[string]struct{}),
	}
}

// StartMatching starts a new match with the given matchID and number of players.
func (mc *Match) StartMatching(matchID string, num uint, fn MatchHandler) ([]uint64, error) {
	mc.mu.Lock()

	if _, exists := mc.statusMap[matchID]; exists {
		mc.mu.Unlock()
		return nil, ErrAlreadyMatch
	}

	mc.statusMap[matchID] = struct{}{}
	mc.mu.Unlock()

	players := make([]uint64, 0, num)

	for {
		if len(players) == int(num) {
			break
		}

		playerId, ok := mc.dequeue(fn)
		if !ok {
			continue
		}

		players = append(players, playerId)
	}

	mc.mu.Lock()
	delete(mc.statusMap, matchID)
	mc.mu.Unlock()

	return players, nil
}

// Enqueue adds a player to the queue of the specified matchID.
func (mc *Match) Enqueue(playerID uint64) error {
	return mc.push(playerID, false) // Add to the back of the queue.
}

// EnqueueFront adds a player to the front of the queue.
func (mc *Match) EnqueueFront(playerID uint64) error {
	return mc.push(playerID, true)
}

// push adds a player to the queue, either at the front or the back.
func (mc *Match) push(playerID uint64, toFront bool) error {
	queueMutex.Lock()
	defer queueMutex.Unlock()

	// Check if the queue is full
	if localQueue.Len() >= defaultQueueSize {
		if mc.processor == nil {
			return ErrMaxQueuePlayer
		}

		if err := mc.processor.Enqueue(playerID); err != nil {
			return err
		}
		return nil
	}

	// Add to the front or back of the queue
	if toFront {
		localQueue.PushFront(playerID)
	} else {
		localQueue.PushBack(playerID)
	}

	return nil
}

// Dequeue removes a player from the queue and returns their ID.
// It also ensures that the provided MatchHandler function is called and its result is respected.
func (mc *Match) dequeue(fn MatchHandler) (uint64, bool) {
	queueMutex.Lock()
	defer queueMutex.Unlock()

	var playerID uint64

	// Check if the queue is empty
	if localQueue.Len() == 0 {
		if mc.processor == nil {
			return 0, false
		}

		var err error
		playerID, err = mc.processor.Dequeue()
		if err != nil {
			return 0, false
		}
	} else {
		// Remove from the front of the queue
		element := localQueue.Front()
		playerID = element.Value.(uint64)
		localQueue.Remove(element)
	}

	// Ensure fn is called and its result is respected
	if fn != nil && !fn(playerID) {
		// Attempt to re-enqueue the player if fn returns false
		if err := mc.push(playerID, false); err != nil {
			return 0, false
		}
		return 0, false
	}

	return playerID, true
}

// QueueSize returns the current size of the queue.
func (mc *Match) QueueSize() int {
	queueMutex.RLock()
	defer queueMutex.RUnlock()

	return localQueue.Len()
}
