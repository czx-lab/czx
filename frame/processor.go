package frame

import "time"

type (
	Processor interface {
		// HandleIdle is called when the loop is idle
		// It is responsible for handling idle timeouts and other idle tasks
		HandleIdle()
		// Close closes the processor and releases any resources.
		// It should be called when the loop is stopped.
		Close()
	}
	// FrameProcessor is an interface for processing game frames.
	// It is responsible for processing the current frame and its inputs.
	FrameProcessor interface {
		Processor
		// Process processes the input frame.
		Process(frame Frame)
		// Resend re-sends the input message to the player.
		// It should be called when the input message is not received by the player.
		Resend(playerId string, sequenceID int)
	}
	// EmptyProcessor is a processor that does nothing.
	EmptyProcessor struct {
		Handler   func()
		Frequency time.Duration
	}
	// NormalProcessor is an interface for processing normal messages.
	// It is responsible for processing the input message.
	NormalProcessor interface {
		Processor
		// Process processes the input message.
		Process(message Message)
	}
)
