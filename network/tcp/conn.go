package tcp

import (
	"czx/network"
	"net"
	"sync"
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

// Init initializes the TcpConn instance
func (c *TcpConn) init() {
	go func() {
		for b := range c.writeQueue {
			if b == nil {
				break
			}

			if _, err := c.conn.Write(b); err != nil {
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
	panic("unimplemented")
}

// ReadMessage implements network.Conn.
func (c *TcpConn) ReadMessage() ([]byte, error) {
	panic("unimplemented")
}

// RemoteAddr implements network.Conn.
func (c *TcpConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// WriteMessage implements network.Conn.
func (c *TcpConn) WriteMessage(args ...[]byte) error {
	panic("unimplemented")
}
