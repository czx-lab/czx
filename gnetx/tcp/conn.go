package tcp

import (
	"czx/network"
	xtcp "czx/network/tcp"
	"net"
	"sync"

	"github.com/panjf2000/gnet/v2"
)

var (
	defaultPendingWrite = 100
)

type (
	GnetTcpConnConf struct {
		// Number of pending writes
		PendingWrite int
	}

	GnetConn struct {
		sync.Mutex
		conf     *GnetTcpConnConf
		gnetconn gnet.Conn
		done     bool

		// Queue for outgoing data
		writeQueue chan []byte
		parse      *xtcp.MessageParser
	}
)

func NewGnetConn(c gnet.Conn, conf *GnetTcpConnConf) *GnetConn {
	if conf.PendingWrite <= 0 {
		conf.PendingWrite = defaultPendingWrite
	}

	gnetconn := &GnetConn{
		gnetconn:   c,
		conf:       conf,
		writeQueue: make(chan []byte, conf.PendingWrite),
	}
	gnetconn.init()

	return gnetconn
}

// WithConn sets the underlying gnet connection for the GnetConn instance
// This allows the user to specify a custom gnet connection if needed
func (g *GnetConn) init() {
	go func() {
		for b := range g.writeQueue {
			if b == nil {
				break
			}

			if _, err := g.gnetconn.Write(b); err != nil {
				break
			}
		}

		g.gnetconn.Close()
		g.Lock()
		g.done = true
		g.Unlock()
	}()
}

// WithParse sets the message parser for the TcpConn instance
// This allows the user to specify how messages should be parsed from the connection
// It returns the TcpConn instance for method chaining
func (g *GnetConn) WithParse(parse *xtcp.MessageParser) *GnetConn {
	g.parse = parse
	return g
}

// Close implements network.Conn.
func (g *GnetConn) Close() {
	g.Lock()
	defer g.Unlock()

	if g.done {
		return
	}

	g.doWrite(nil)
	g.done = true
}

func (g *GnetConn) doWrite(b []byte) {
	if len(g.writeQueue) == cap(g.writeQueue) {
		g.doDestroy()
		return
	}

	g.writeQueue <- b
}

// Destroy implements network.Conn.
func (g *GnetConn) Destroy() {
	g.Lock()
	defer g.Unlock()

	g.doDestroy()
}

func (c *GnetConn) doDestroy() {
	c.gnetconn.SetLinger(0)
	c.gnetconn.Close()

	if c.done {
		return
	}

	close(c.writeQueue)
	c.done = true
}

// LocalAddr implements network.Conn.
func (g *GnetConn) LocalAddr() net.Addr {
	return g.gnetconn.LocalAddr()
}

// ReadMessage implements network.Conn.
//
// NOTE: GnetConn does not support reading messages directly. Instead, it uses the gnet framework's event loop to handle incoming messages asynchronously.
// This method is a placeholder and should not be used directly.
func (g *GnetConn) ReadMessage() (b []byte, err error) {
	return
}

// RemoteAddr implements network.Conn.
func (g *GnetConn) RemoteAddr() net.Addr {
	return g.gnetconn.RemoteAddr()
}

// WriteMessage implements network.Conn.
func (g *GnetConn) WriteMessage(args ...[]byte) error {
	return g.parse.Write(g, args...)
}

var _ network.Conn = (*GnetConn)(nil)
