package frame

import (
	"encoding/json"
	"time"
)

type (
	// Message represents a single input message from a player.
	Message struct {
		PlayerID  string
		Data      []byte
		Timestamp time.Time
		FrameID   int // Sequence ID for the message
	}

	// Frame represents a single frame of game state
	// and the inputs from all players.
	Frame struct {
		FrameID uint64
		Inputs  map[string][]Message
	}
)

// Serialize serializes the Frame to JSON.
// It uses the json package to convert the Frame struct to a JSON byte slice.
// The function returns the serialized byte slice and any error encountered during serialization.
func (r *Frame) Serialize() ([]byte, error) {
	return json.Marshal(r)
}

// Deserialize deserializes the JSON byte slice to a Frame struct.
// It uses the json package to convert the JSON byte slice to a Frame struct.
func (r *Frame) Deserialize(data []byte) error {
	return json.Unmarshal(data, r)
}

// Serialize serializes the Message to JSON.
// It uses the json package to convert the Message struct to a JSON byte slice.
func (m *Message) Serialize() ([]byte, error) {
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}

	return json.Marshal(m)
}

// Deserialize deserializes the JSON byte slice to a Message struct.
// It uses the json package to convert the JSON byte slice to a Message struct.
func (m *Message) Deserialize(data []byte) error {
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}

	return json.Unmarshal(data, m)
}
