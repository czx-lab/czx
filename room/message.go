package room

import "time"

// Message represents a message to be processed in the room.
type Message struct {
	// PlayerID is the ID of the player sending the message.
	PlayerID uint64
	// The actual message content.
	Msg []byte
	// Timestamp is the time when the message was sent.
	Timestamp time.Time
}
