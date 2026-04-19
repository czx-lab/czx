package frame

import (
	"context"
	"time"
)

const (
	// game logic frame processing frequency
	frequency uint = 30
	queueCap  int  = 100 // Capacity of the normal message queue
)

type (
	// LoopFace defines the interface for a game loop.
	LoopFace interface {
		// Start starts the loop in a separate goroutine.
		Start(context.Context) error
		// Stop stops the loop and waits for it to finish processing.
		Stop()
		// Write writes a message to the loop's input queue.
		Write(Message) error
		// Frequency sets the processing frequency of the loop.
		Frequency(uint) error
	}

	// FrameFace defines the interface for a game frame, which includes a loop.
	FrameFace interface {
		LoopFace
		// FrameId returns the current frame ID.
		FrameId() uint64
		// ResetFrameId resets the frame ID to the specified value.
		ResetFrameId(uint64)
		// PlayerIds returns a copy of the current player IDs and their last processed frame IDs.
		PlayerIds() map[string]uint
		// RegisterPlayer registers a new player to the frame loop and resends the last processed frame if necessary.
		RegisterPlayer(string)
		// DeletePlayer unregisters a player from the frame loop and removes their input queue.
		DeletePlayer(string)
	}

	// NormalFace defines the interface for a normal loop, which processes normal messages.
	NormalFace interface {
		LoopFace
		// WriteTimeout writes a message to the loop's input queue with a timeout.
		WriteTimeout(Message, time.Duration) error
	}
)
