package network

type (
	ProcessorConf struct {
		// LittleEndian indicates if the processor should use little-endian byte order.
		LittleEndian bool
	}

	// Processor defines the interface for processing messages.
	// It includes methods for processing, unmarshalling, and marshalling messages.
	Handler func([]any)

	// Message represents a message type with an ID and data.
	Message struct {
		// ID of the message.
		// if it's a json message processor, the ID won't do anything
		ID uint16
		// Data of the message.
		Data any
	}

	// message format
	// protobuf: stateless code
	// ----------------------------------------------
	// |   1/2/4     |     2     |       data       |
	// ----------------------------------------------
	// |   length    |   msgid   | protobuf message |
	// ----------------------------------------------
	//
	// protobuf: write status code
	// ------------------------------------------------------
	// |   1/2/4     |     2    |    2   |       data       |
	// ------------------------------------------------------
	// |   length    |   code   |  msgid | protobuf message |
	// ------------------------------------------------------
	//
	// jsonx: stateless code
	// ------------------------------
	// |   1/2/4     |     data   	|
	// ------------------------------
	// |   length    | json message |
	// ------------------------------
	//
	// jsonx: write status code
	// --------------------------------------
	// |   1/2/4     |   2  |      data   	|
	// --------------------------------------
	// |   length    | code |  json message |
	// --------------------------------------
	Processor interface {
		// Process handles the incoming data and returns a response.
		Process(data any, agent Agent) error
		// Unmarshal converts the byte slice to a message.
		// It returns the message and an error if any.
		Unmarshal(data []byte) (any, error)
		// Marshal converts the message to a byte slice.
		// It returns a slice of byte slices to support fragmentation.
		Marshal(msgs any) ([][]byte, error)
		// MarshalWithCode converts the message to a byte slice with a specific code.
		MarshalWithCode(code uint16, msg any) ([][]byte, error)
		// Register registers a message type with an ID.
		Register(msg Message) error
		// RegisterHandler registers a handler for a message type.
		RegisterHandler(msg any, handler Handler) error
		// RegisterRawHandler registers a raw handler for a message type.
		// This handler is called before the message is unmarshalled.
		RegisterRawHandler(msg any, handler Handler) error
	}
)
