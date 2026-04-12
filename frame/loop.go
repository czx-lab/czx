package frame

import "context"

const (
	// game logic frame processing frequency
	frequency uint = 30
	queueCap  int  = 100 // Capacity of the normal message queue
)

// LoopFace defines the interface for a game loop.
type LoopFace interface {
	// Start starts the loop in a separate goroutine.
	Start(context.Context) error
	// Stop stops the loop and waits for it to finish processing.
	Stop()
	// Write writes a message to the loop's input queue.
	Write(Message) error
	// Frequency sets the processing frequency of the loop.
	Frequency(uint) error
}
