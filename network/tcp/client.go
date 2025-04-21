package tcp

import (
	"net"
	"sync"
	"time"

	"github.com/czx-lab/czx/network"
)

var (
	//	default values for the client configuration
	defaultConnNum         = 1
	defaultPendingWriteNum = 100
	defaultConnectInterval = 3 * time.Second
)

type (
	TcpClientConf struct {
		// Server address
		Addr string
		// Number of connections to the server
		ConnNum int
		// Number of pending writes
		PendingWriteNum int
		// Interval between connection attempts
		ConnectInterval time.Duration
		// Automatically reconnect on close
		AutoReconnect bool
	}
	TcpClient struct {
		sync.Mutex
		wg   sync.WaitGroup
		conf *TcpClientConf
		// Agent interface for handling messages
		// The agent is responsible for handling messages and events
		agent func(*TcpConn) network.Agent
		// Message parser for the client
		parse *MessageParser
		// Connection pool
		conns Conns
		// Flag to indicate if the client is closed
		done bool
	}
)

func NewClient(conf *TcpClientConf) *TcpClient {
	defaultClientConf(conf)

	return &TcpClient{
		conf:  conf,
		conns: make(Conns),
	}
}

func (t *TcpClient) dial() net.Conn {
	for {
		conn, err := net.Dial("tcp", t.conf.Addr)
		if err == nil || t.done {
			return conn
		}

		time.Sleep(t.conf.ConnectInterval)
		continue
	}
}

// WithParse sets the message parser for the TcpClient instance
// This allows the user to specify how messages should be parsed from the connection
func (t *TcpClient) WithParse(parse *MessageParser) *TcpClient {

	t.Lock()
	defer t.Unlock()

	t.parse = parse
	return t
}

// WithAgent sets the agent for the TcpClient instance
// The agent is responsible for handling messages and events
func (t *TcpClient) WithAgent(agent func(*TcpConn) network.Agent) {
	t.Lock()
	defer t.Unlock()

	t.agent = agent
}

// Start the client and establish connections to the server
func (c *TcpClient) Start() {
	// Set the maximum number of connections to the server
	for range c.conf.ConnNum {
		c.wg.Add(1)
		go c.connect()
	}
}

func (t *TcpClient) connect() {
	defer t.wg.Done()

Reconnect:
	conn := t.dial()
	if conn == nil {
		return
	}

	t.Lock()
	if t.done {
		t.Unlock()
		conn.Close()
		return
	}
	t.conns[conn] = struct{}{}
	t.Unlock()

	// Set the connection to read/write mode
	tcpconn := NewTcpConn(conn, &TcpConnConf{
		PendingWrite: t.conf.PendingWriteNum,
	}).WithParse(t.parse)

	agent := t.agent(tcpconn)
	agent.Run()

	tcpconn.Close()
	t.Lock()
	delete(t.conns, conn)
	t.Unlock()
	agent.OnClose()

	// If the connection is closed, reconnect if AutoReconnect is enabled
	if t.conf.AutoReconnect {
		time.Sleep(t.conf.ConnectInterval)
		goto Reconnect
	}
}

// Close the client and all connections
func (t *TcpClient) Close() {
	t.Lock()
	t.done = true
	for conn := range t.conns {
		conn.Close()
	}
	t.conns = nil
	t.Unlock()

	t.wg.Wait()
}

func defaultClientConf(conf *TcpClientConf) {
	if conf.ConnNum <= 0 {
		conf.ConnNum = defaultConnNum
	}
	if conf.ConnectInterval <= 0 {
		conf.ConnectInterval = defaultConnectInterval
	}
	if conf.PendingWriteNum <= 0 {
		conf.PendingWriteNum = defaultPendingWriteNum
	}
}
