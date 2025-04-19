package network

import (
	"net"
	"net/http"
)

type (
	// Conn is an interface for handling network connections and messages.
	// It provides methods for reading and writing messages, managing connection state,
	Conn interface {
		// Read reads data from the connection and returns it as a byte slice.
		ReadMessage() ([]byte, error)
		// Write sends a message to the connection.
		WriteMessage(args ...[]byte) error
		// LocalAddr returns the local address of the connection.
		LocalAddr() net.Addr
		// RemoteAddr returns the remote address of the connection.
		RemoteAddr() net.Addr
		// Returns the client address of the connection.
		ClientAddr() ClientAddrMessage
		// Close closes the connection.
		// NOTE: Close guarantees that messages will not be lost
		// but does not guarantee that the connection will be closed.
		Close()
		// Destroy closes the connection and releases resources.
		// NOTE: Destroy does not guarantee that messages will not be lost
		// it guarantees that the connection will be closed.
		Destroy()
	}

	// Reader is an interface for reading data from a connection.
	// It provides a method to read data into a byte slice.
	Reader interface {
		// Read reads data from the connection and returns it as an int and an error.
		Read([]byte) (int, error)
	}

	// ClientAddrMessage is a struct that contains information about the client address and the request.
	// It includes the IP address, port, and the HTTP request associated with the connection.
	ClientAddrMessage struct {
		IP   string
		Port string
		Req  *http.Request
	}

	// PreConnHandler is a function type that handles incoming connections and messages. It takes an Agent and a PreHandlerMessage as arguments and returns an error.
	PreConnHandler func(Agent, ClientAddrMessage)
)
