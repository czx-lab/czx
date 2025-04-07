package match

import (
	"errors"
	"sync"
)

const defaultQueueSize = 1000

var (
	ErrMaxQueuePlayer = errors.New("max queue player")
	ErrAlreadyMatch   = errors.New("macth is already waiting")
	// Global queue shared by all Match instances.
	localQueue chan uint64
)

// Initialize the global queue with a default size.
func init() {
	localQueue = make(chan uint64, defaultQueueSize)
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
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Check if the global queue is full before adding a player.
	if len(localQueue) >= cap(localQueue) {
		if mc.processor == nil {
			return ErrMaxQueuePlayer // Return error if the queue is full and no processor is available.
		}

		if err := mc.processor.Enqueue(playerID); err != nil {
			return err
		}
		return nil
	}

	localQueue <- playerID
	return nil
}

// Dequeue removes a player from the queue and returns their ID.
// It also ensures that the provided MatchHandler function is called and its result is respected.
func (mc *Match) dequeue(fn MatchHandler) (uint64, bool) {
	var playerID uint64
	var err error

	select {
	case playerID = <-localQueue:
	default:
		if mc.processor == nil {
			return 0, false // Return false if the channel is empty.
		}

		playerID, err = mc.processor.Dequeue()
		if err != nil {
			return 0, false
		}
	}

	// Ensure fn is called and its result is respected.
	if fn != nil && !fn(playerID) {
		// Attempt to re-enqueue the player if fn returns false.
		if err := mc.Enqueue(playerID); err != nil {
			return 0, false
		}
		return 0, false
	}

	return playerID, true
}

// QueueSize returns the current size of the queue.
func (mc *Match) QueueSize() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return len(localQueue)
}
