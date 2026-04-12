package network

import (
	"encoding/binary"
	"errors"

	fb "github.com/google/flatbuffers/go"
)

const (
	IDCodeLenType8  IDCodeLenType = iota + 1 // 1 byte
	IDCodeLenType16                          // 2 bytes
	IDCodeLenType32 IDCodeLenType = iota + 2 // 4 bytes
)

type (
	// IDCodeLenType defines the length (in bytes) of the message ID field or status code.
	IDCodeLenType uint8

	// ProcessorConf holds the configuration for the message processor,
	// including endianness and message ID length.
	ProcessorConf struct {
		// LittleEndian indicates if the processor should use little-endian byte order.
		LittleEndian bool
		IDLength     IDCodeLenType // 1, 2, or 4 bytes for the message ID
		CodeLength   IDCodeLenType // 1, 2, or 4 bytes for the status code (optional)
	}

	// Processor defines the interface for processing messages.
	// It includes methods for processing, unmarshalling, and marshalling messages.
	Handler func([]any)

	// Message represents a message type with an ID and data.
	Message struct {
		// ID of the message.
		// if it's a json message processor, the ID won't do anything
		ID uint
		// Data of the message.
		Data any
		// Serializer function for the message.
		Fn any
	}

	// FlatbuffersSerializerFn defines the function signature for serializing Flatbuffers messages.
	FlatbuffersSerializerFn func(*fb.Builder, any) fb.UOffsetT

	// message format
	// protobuf: stateless code
	// --------------------------------------------------
	// |   1/2/4     |     1/2/4     |       data       |
	// --------------------------------------------------
	// |   length    |   	msgid    | protobuf message |
	// --------------------------------------------------
	//
	// protobuf: write status code
	// ---------------------------------------------------------------
	// |   1/2/4     |     1/2/4     |    1/2/4   |       data       |
	// ---------------------------------------------------------------
	// |   length    |   	code     |    msgid   | protobuf message |
	// ---------------------------------------------------------------
	//
	// jsonx: stateless code
	// ------------------------------
	// |   1/2/4     |     data   	|
	// ------------------------------
	// |   length    | json message |
	// ------------------------------
	//
	// jsonx: write status code
	// ------------------------------------------
	// |   1/2/4     |  1/2/4   |      data   	|
	// ------------------------------------------
	// |   length    | 	 code   |  json message |
	// ------------------------------------------
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
		MarshalWithCode(code uint, msg any) ([][]byte, error)
		// Register registers a message type with an ID.
		Register(msg Message) error
		// RegisterHandler registers a handler for a message type.
		RegisterHandler(msg any, handler Handler) error
	}
)

// HandlerArgs extracts the typed message and Agent from the handler arguments.
func HandlerArgs[T any](args []any) (T, Agent) {
	return args[0].(T), args[1].(Agent)
}

// PutCode writes the status code into the provided buffer according to the configured code length and endianness.
func PutCode(buffer []byte, code uint, conf ProcessorConf) {
	switch conf.CodeLength {
	case IDCodeLenType8:
		buffer[0] = byte(code)
	case IDCodeLenType16:
		if conf.LittleEndian {
			binary.LittleEndian.PutUint16(buffer, uint16(code))
		} else {
			binary.BigEndian.PutUint16(buffer, uint16(code))
		}
	case IDCodeLenType32:
		if conf.LittleEndian {
			binary.LittleEndian.PutUint32(buffer, uint32(code))
		} else {
			binary.BigEndian.PutUint32(buffer, uint32(code))
		}
	}
}

// PutID writes the message ID into the provided buffer according to the configured ID length and endianness.
func PutID(buffer []byte, id uint, conf ProcessorConf) {
	switch conf.IDLength {
	case IDCodeLenType8:
		buffer[0] = byte(id)
	case IDCodeLenType16:
		if conf.LittleEndian {
			binary.LittleEndian.PutUint16(buffer, uint16(id))
		} else {
			binary.BigEndian.PutUint16(buffer, uint16(id))
		}
	case IDCodeLenType32:
		if conf.LittleEndian {
			binary.LittleEndian.PutUint32(buffer, uint32(id))
		} else {
			binary.BigEndian.PutUint32(buffer, uint32(id))
		}
	}
}

// GetID reads the message ID from the provided data according to the configured ID length and endianness.
func GetID(data []byte, conf ProcessorConf) (uint, error) {
	if len(data) < int(conf.IDLength) {
		return 0, errors.New("data too short for ID")
	}

	var id uint
	switch conf.IDLength {
	case IDCodeLenType8:
		id = uint(data[0])
	case IDCodeLenType16:
		if conf.LittleEndian {
			id = uint(binary.LittleEndian.Uint16(data))
		} else {
			id = uint(binary.BigEndian.Uint16(data))
		}
	case IDCodeLenType32:
		if conf.LittleEndian {
			id = uint(binary.LittleEndian.Uint32(data))
		} else {
			id = uint(binary.BigEndian.Uint32(data))
		}
	}

	return id, nil
}
