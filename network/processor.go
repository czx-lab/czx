package network

type (
	ProcessorConf struct {
		// LittleEndian indicates if the processor should use little-endian byte order.
		LittleEndian bool
	}

	// Processor defines the interface for processing messages.
	// It includes methods for processing, unmarshalling, and marshalling messages.
	Handler func([]any)

	Processor interface {
		// Process handles the incoming data and returns a response.
		Process(data any) error
		// Unmarshal converts the byte slice to a message.
		// It returns the message and an error if any.
		Unmarshal(data []byte) (any, error)
		// Marshal converts the message to a byte slice.
		// It returns a slice of byte slices to support fragmentation.
		Marshal(msgs any) ([][]byte, error)
		// MarshalWithCode converts the message to a byte slice with a specific code.
		MarshalWithCode(code uint16, msg any) ([][]byte, error)
		// Register registers a message type with an ID.
		Register(id uint16, msg any) error
		// RegisterHandler registers a handler for a message type.
		RegisterHandler(msg any, handler Handler) error
	}

	// JsonProcessor defines the interface for JSON processing.
	JsonProcessor interface {
		Processor
		RegisterExceptID(msg any) error // Register a message type without an ID.
	}
)
