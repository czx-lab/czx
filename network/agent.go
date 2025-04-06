package network

import "net"

type Agent interface {
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
}
