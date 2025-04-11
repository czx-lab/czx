package network

import "net"

type Conn interface {
	// Write sends a message to the connection.
	Write(args ...[]byte) error
	// LocalAddr returns the local address of the connection.
	LocalAddr() net.Addr
	// RemoteAddr returns the remote address of the connection.
	RemoteAddr() net.Addr
	// Close closes the connection.
	Close()
	// Destroy closes the connection and releases resources.
	Destroy()
}

type WsReader interface {
	// Read reads data from the connection and returns it as a byte slice.
	Read() ([]byte, error)
}

type TcpReader interface {
	// Read reads data from the connection and returns it as an int and an error.
	Read([]byte) (int, error)
}
