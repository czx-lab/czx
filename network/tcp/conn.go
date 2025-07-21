package tcp

import (
	"net"
	"sync"

	"github.com/czx-lab/czx/network"
	"github.com/czx-lab/czx/xlog"

	"go.uber.org/zap"
)

var (
	defaultPendingWrite = 100
)

type (
	// Conns is a map of network connections
	Conns map[net.Conn]struct{}

	TcpConnConf struct {
		// Number of pending writes
		PendingWrite int
	}

	TcpConn struct {
		sync.Mutex
		conf *TcpConnConf
		// The underlying network connection
		conn net.Conn
		// Queue for outgoing data
		writeQueue chan []byte
		done       bool
		parse      *MessageParser
		clientAddr network.ClientAddrMessage
	}
)

var _ network.Conn = (*TcpConn)(nil)

func NewTcpConn(conn net.Conn, conf *TcpConnConf) *TcpConn {
	if conf.PendingWrite <= 0 {
		conf.PendingWrite = defaultPendingWrite
	}

	// Create a new TcpConn instance with the provided connection and configuration
	// Initialize the write queue with the specified size
	tcpconn := &TcpConn{
		conn:       conn,
		writeQueue: make(chan []byte, conf.PendingWrite),
		conf:       conf,
	}

	tcpconn.init()

	return tcpconn
}

// WithParse sets the message parser for the TcpConn instance
// This allows the user to specify how messages should be parsed from the connection
// It returns the TcpConn instance for method chaining
func (c *TcpConn) WithParse(parse *MessageParser) *TcpConn {
	c.parse = parse
	return c
}

// Init initializes the TcpConn instance
func (c *TcpConn) init() {
	go func() {
		for b := range c.writeQueue {
			if b == nil {
				break
			}

			if _, err := c.conn.Write(b); err != nil {
				xlog.Write().Error("tcp conn write error", zap.Error(err))
				break
			}
		}

		c.conn.Close()
		c.Lock()
		c.done = true
		c.Unlock()
	}()
}

// Close implements network.Conn.
func (c *TcpConn) Close() {
	c.Lock()
	defer c.Unlock()

	if c.done {
		return
	}

	c.doWrite(nil)
	c.done = true
}

func (c *TcpConn) doWrite(b []byte) {
	if len(c.writeQueue) == cap(c.writeQueue) {
		c.doDestroy()
		return
	}

	c.writeQueue <- b
}

// Destroy implements network.Conn.
func (c *TcpConn) Destroy() {
	c.Lock()
	defer c.Unlock()

	c.doDestroy()
}

func (c *TcpConn) doDestroy() {
	c.conn.(*net.TCPConn).SetLinger(0)
	c.conn.Close()

	if !c.done {
		close(c.writeQueue)
		c.done = true
	}
}

// LocalAddr implements network.Conn.
func (c *TcpConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// Read implements network.Conn.
func (c *TcpConn) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

// ReadMessage implements network.Conn.
func (c *TcpConn) ReadMessage() ([]byte, error) {
	return c.parse.Read(c)
}

// RemoteAddr implements network.Conn.
func (c *TcpConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// ClientAddr implements network.Conn.
func (c *TcpConn) ClientAddr() network.ClientAddrMessage {
	return c.clientAddr
}

// WithClientAddr sets the client address message for the GnetConn instance
// This allows the user to specify the client address information associated with the connection
func (c *TcpConn) WithClientAddr(clientAddr network.ClientAddrMessage) {
	c.clientAddr = clientAddr
}

// WriteMessage implements network.Conn.
func (c *TcpConn) WriteMessage(args ...[]byte) error {
	return c.parse.Write(c, args...)
}
