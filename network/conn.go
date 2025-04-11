package network

import "net"

type Conn interface {
	// Read reads data from the connection and returns it as a byte slice.
	ReadMessage() ([]byte, error)
	// Write sends a message to the connection.
	WriteMessage(args ...[]byte) error
	// LocalAddr returns the local address of the connection.
	LocalAddr() net.Addr
	// RemoteAddr returns the remote address of the connection.
	RemoteAddr() net.Addr
	// Close closes the connection.
	Close()
	// Destroy closes the connection and releases resources.
	Destroy()
}

type Reader interface {
	// Read reads data from the connection and returns it as an int and an error.
	Read([]byte) (int, error)
}
