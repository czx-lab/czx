package tcp

import (
	"errors"
	"net"
	"sync"

	"github.com/czx-lab/czx/network"
	xtcp "github.com/czx-lab/czx/network/tcp"

	"github.com/panjf2000/gnet/v2"
)

var (
	defaultPendingWrite = 100
)

type (
	// Conns is a map of network connections
	Conns map[network.Conn]struct{}

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
		gbuffer    *GBuffer
		clientAddr network.ClientAddrMessage
		metrics    network.ServerMetrics
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
		gbuffer:    NewGBuffer(),
	}
	gnetconn.init()

	return gnetconn
}

// WithMetrics sets the server metrics for the GnetConn instance
// This allows the user to specify metrics tracking for the connection
// It returns the GnetConn instance for method chaining
func (g *GnetConn) WithMetrics(m network.ServerMetrics) *GnetConn {
	g.metrics = m
	return g
}

// WithConn sets the underlying gnet connection for the GnetConn instance
// This allows the user to specify a custom gnet connection if needed
func (g *GnetConn) init() {
	go func() {
		for b := range g.writeQueue {
			if b == nil {
				break
			}

			n, err := g.gnetconn.Write(b)
			if err != nil {
				g.metrics.IncWriteErrors()
				break
			}
			g.metrics.AddSentBytes(n)
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

// WriteBuffer writes data to the GBuffer.
func (g *GnetConn) WriteBuffer(data []byte) (n int, err error) {
	g.metrics.AddReceivedBytes(len(data))
	return g.gbuffer.Write(data)
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
func (g *GnetConn) ReadMessage() (b []byte, err error) {
	g.Lock()
	defer g.Unlock()

	if g.done {
		return nil, errors.New("connection is closed")
	}
	return g.parse.Read(g.gbuffer)
}

// RemoteAddr implements network.Conn.
func (g *GnetConn) RemoteAddr() net.Addr {
	return g.gnetconn.RemoteAddr()
}

// ClientAddr implements network.Conn.
func (g *GnetConn) ClientAddr() network.ClientAddrMessage {
	return g.clientAddr
}

// withClientAddr sets the client address message for the GnetConn instance
// This allows the user to specify the client address information associated with the connection
func (g *GnetConn) withClientAddr(msg network.ClientAddrMessage) {
	g.clientAddr = msg
}

// WriteMessage implements network.Conn.
func (g *GnetConn) WriteMessage(args ...[]byte) error {
	return g.parse.Write(g, args...)
}

// Write implements io.Writer.
func (g *GnetConn) Write(p []byte) (n int, err error) {
	g.Lock()
	defer g.Unlock()

	if g.done || p == nil {
		return 0, errors.New("dead connection or nil data")
	}

	g.doWrite(p)
	return len(p), nil
}

var _ network.Conn = (*GnetConn)(nil)
