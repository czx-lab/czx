package frame

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
	}
	// NormalProcessor is an interface for processing normal messages.
	// It is responsible for processing the input message.
	NormalProcessor interface {
		Processor
		// Process processes the input message.
		Process(message Message)
	}
)
