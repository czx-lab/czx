package match

import (
	"container/list"
	"errors"
	"sync"
	"time"

	"github.com/czx-lab/czx/utils/xslices"
)

const defaultQueueSize = 1000

var (
	ErrMaxQueuePlayer = errors.New("max queue player")
	ErrAlreadyMatch   = errors.New("macth is already waiting")
	ErrMatchTimeout   = errors.New("match timeout")
	ErrPlayerInQueue  = errors.New("player in queue")
	ErrGtMatchNum     = errors.New("greater than the number of matches")
)

type (
	Matching struct {
		MatchID string
		Num     uint
		Fn      MatchHandler
		Timeout time.Duration
	}
	// MatchStatus represents the status of a match.
	MatchStatus uint
	Match       struct {
		// Mutex to ensure thread-safe operations.
		mu sync.RWMutex
		// StatusMap maps matchID to its current status.
		statusMap map[string]struct{}
		// Processor is an interface for processing match logic.
		processor MatchProcessor
		queue     *list.List
		index     map[string]*list.Element
		done      chan string
	}
)

func NewMatch(processor MatchProcessor) *Match {
	return &Match{
		processor: processor,
		statusMap: make(map[string]struct{}),
		queue:     list.New(),
		index:     make(map[string]*list.Element),
		done:      make(chan string),
	}
}

// StartMatching starts a new match with the given matchID and number of players.
func (mc *Match) StartMatching(args Matching, ownerIDs ...string) ([]string, error) {
	mc.mu.Lock()

	if _, exists := mc.statusMap[args.MatchID]; exists {
		mc.mu.Unlock()
		return nil, ErrAlreadyMatch
	}

	if len(ownerIDs) > int(args.Num) {
		mc.mu.Unlock()
		return nil, ErrGtMatchNum
	}

	mc.statusMap[args.MatchID] = struct{}{}
	mc.mu.Unlock()

	deadline := time.Now().Add(args.Timeout)
	players := make([]string, 0, args.Num)

	// If there are ownerIDs, we need to remove them from the queue.
	if len(ownerIDs) > 0 {
		players = append(players, ownerIDs...)
	}

	for {
		select {
		case matchId := <-mc.done:
			if matchId == args.MatchID {
				for _, player := range players {
					if _, ok := xslices.Search(ownerIDs, func(id string) bool {
						return id == player
					}); ok {
						continue
					}
					mc.push(player, true) // Re-enqueue the player to the front of the queue.
				}
				return players, nil
			}
		default:
			// Continue without blocking if no match is done.
		}
		if len(players) == int(args.Num) {
			break
		}
		if time.Now().After(deadline) {
			return nil, ErrMatchTimeout
		}
		playerId, ok := mc.dequeue(args.Fn)
		if !ok {
			continue
		}

		players = append(players, *playerId)
	}

	mc.mu.Lock()
	delete(mc.statusMap, args.MatchID)
	mc.mu.Unlock()

	return players, nil
}

// CancelMatch cancels a match with the given matchID.
// It removes the matchID from the statusMap and signals that the match is done.
func (mc *Match) CancelMatch(matchID string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	delete(mc.statusMap, matchID)
	mc.done <- matchID
}

// Enqueue adds a player to the queue of the specified matchID.
func (mc *Match) Enqueue(playerID string) error {
	return mc.push(playerID, false) // Add to the back of the queue.
}

// EnqueueFront adds a player to the front of the queue.
func (mc *Match) EnqueueFront(playerID string) error {
	return mc.push(playerID, true)
}

// Remove removes a player from the queue.
// It locks the queue to ensure thread safety while removing the player.
func (mc *Match) Remove(playerID string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if e, ok := mc.index[playerID]; ok {
		mc.queue.Remove(e)
		delete(mc.index, playerID)
	}
}

// push adds a player to the queue, either at the front or the back.
func (mc *Match) push(playerID string, toFront bool) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.index[playerID]; exists {
		return ErrPlayerInQueue
	}

	// Check if the queue is full
	if mc.queue.Len() >= defaultQueueSize {
		if mc.processor == nil {
			return ErrMaxQueuePlayer
		}

		if err := mc.processor.Enqueue(playerID); err != nil {
			return err
		}
		return nil
	}

	// Add to the front or back of the queue
	var e *list.Element
	if toFront {
		e = mc.queue.PushFront(playerID)
	} else {
		e = mc.queue.PushBack(playerID)
	}

	mc.index[playerID] = e
	return nil
}

// Dequeue removes a player from the queue and returns their ID.
// It also ensures that the provided MatchHandler function is called and its result is respected.
func (mc *Match) dequeue(fn MatchHandler) (*string, bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	var playerID string

	// Check if the queue is empty
	if mc.queue.Len() == 0 {
		if mc.processor == nil {
			return nil, false
		}

		var err error
		playerID, err = mc.processor.Dequeue()
		if err != nil {
			return nil, false
		}
	} else {
		// Remove from the front of the queue
		element := mc.queue.Front()
		playerID = element.Value.(string)
		mc.queue.Remove(element)
		delete(mc.index, playerID)
	}

	// Ensure fn is called and its result is respected
	if fn != nil && !fn(playerID) {
		// Attempt to re-enqueue the player if fn returns false
		if err := mc.push(playerID, false); err != nil {
			return nil, false
		}
		return nil, false
	}

	return &playerID, true
}

// QueueSize returns the current size of the queue.
func (mc *Match) QueueSize() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return mc.queue.Len()
}
