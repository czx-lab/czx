package frame

import "time"

type (
	Processor interface {
		// OnClose closes the processor and releases any resources.
		// It should be called when the loop is stopped.
		OnClose()
	}
	// FrameProcessor is an interface for processing game frames.
	// It is responsible for processing the current frame and its inputs.
	FrameProcessor interface {
		Processor
		// Process processes the input frame.
		Process(frame Frame)
		// Resend re-sends the input message to the player.
		// It should be called when the input message is not received by the player.
		Resend(playerId string, frameId int)
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
