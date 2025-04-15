package gnetx

import (
	"czx/network"
	"net"

	"github.com/panjf2000/gnet/v2"
)

type GnetConn struct {
	gnetconn gnet.Conn
}

func NewGnetConn(c gnet.Conn) *GnetConn {
	return &GnetConn{gnetconn: c}
}

// Close implements network.Conn.
func (g *GnetConn) Close() {
	panic("unimplemented")
}

// Destroy implements network.Conn.
func (g *GnetConn) Destroy() {
	panic("unimplemented")
}

// LocalAddr implements network.Conn.
func (g *GnetConn) LocalAddr() net.Addr {
	panic("unimplemented")
}

// ReadMessage implements network.Conn.
func (g *GnetConn) ReadMessage() ([]byte, error) {
	panic("unimplemented")
}

// RemoteAddr implements network.Conn.
func (g *GnetConn) RemoteAddr() net.Addr {
	panic("unimplemented")
}

// WriteMessage implements network.Conn.
func (g *GnetConn) WriteMessage(args ...[]byte) error {
	panic("unimplemented")
}

var _ network.Conn = (*GnetConn)(nil)
