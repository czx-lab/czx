package network

import "net"

type Conn interface {
	Read() ([]byte, error)
	Write(args ...[]byte) error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Close()
	Destroy()
}
