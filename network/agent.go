package network

import "net"

type Agent interface {
	// Run starts the agent to handle incoming messages.
	Run()
	// Write sends a message to the connection.
	Write(msg any) error
	// WriteWithError sends a message with an error code to the connection.
	WriteWithError(code int, msg any) error
	// LocalAddr returns the local address of the connection.
	LocalAddr() net.Addr
	// RemoteAddr returns the remote address of the connection.
	RemoteAddr() net.Addr
	Close()
	Destroy()
}
