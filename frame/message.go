package frame

import "time"

type (
	// Message represents a single input message from a player.
	Message struct {
		PlayerID  string
		Data      []byte
		Timestamp time.Time
	}

	// Frame represents a single frame of game state
	// and the inputs from all players.
	Frame struct {
		FrameID uint64
		Inputs  map[string]Message
	}
)
