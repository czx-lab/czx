package network

import "net"

type (
	// Agent is an interface for handling network connections and messages.
	// It provides methods for sending and receiving messages, managing connection state,
	Agent interface {
		// Run starts the agent to handle incoming messages.
		Run()
		// Write sends a message to the connection.
		Write(msg any) error
		// WriteWithCode sends a message with a specific error code to the connection.
		// This is useful for sending error messages or status codes.
		WriteWithCode(code uint16, msg any) error
		// LocalAddr returns the local address of the connection.
		LocalAddr() net.Addr
		// RemoteAddr returns the remote address of the connection.
		RemoteAddr() net.Addr
		// ClientAddr returns the client address of the connection.
		// This includes the IP address, port, and the HTTP request associated with the connection.
		ClientAddr() ClientAddrMessage
		// Close closes the connection.
		Close()
		// Destroy cleans up the agent and releases resources.
		Destroy()
		// OnClose is called when the connection is closed.
		OnClose()
		// SetUserData sets user data for the connection.
		SetUserData(data any)
		// GetUserData retrieves user data for the connection.
		GetUserData() any
		// PreConnHandler is a function that handles incoming connections and messages.
		// It takes an Agent and a PreHandlerMessage as arguments and returns an error.
		OnPreConn(ClientAddrMessage)
	}

	// GnetAgent is an interface for handling incoming data in a gnet connection.
	// It is used to process messages received from the network.
	GnetAgent interface {
		// Process incoming data
		React([]byte)
	}
)
