package match

type (
	// Package match provides functionality for managing player matches in a game.
	// It includes a Match struct that handles player queues and matching logic.
	MatchProcessor interface {
		// Dequeue removes a player from the queue and returns their ID.
		Dequeue() (playerID string, err error)
		// Enqueue adds a player to the queue.
		Enqueue(playerID string) error
		// OnMatched is called when a group of players are successfully matched.
		OnMatched(players []string)
		// OnDeadline is called when a matching process times out.
		// It receives the MatchID to identify which match has timed out.
		OnDeadline(MatchID string)
	}
	// MatchHandler is a function type that takes a player ID and returns a boolean indicating success or failure.
	MatchHandler func(playerId string) bool
)
