package frame

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
		// CatchUp brings a reconnecting player's client back to the current
		// frame. fromFrame is the last frame that client had applied; the
		// processor replays retained broadcasts after it, or sends a state
		// snapshot first when the gap is too large to replay.
		CatchUp(playerId string, fromFrame uint64)
	}
	// NormalProcessor is an interface for processing normal messages.
	// It is responsible for processing the input message.
	NormalProcessor interface {
		Processor
		// Process processes the input message.
		Process(message Message)
	}
)
