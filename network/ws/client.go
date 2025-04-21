package ws

import (
	"sync"
	"time"

	"github.com/czx-lab/czx/network"

	"github.com/gorilla/websocket"
)

var (
	//	default values for the client configuration
	defaultConnNum                = 1
	defaultPendingWriteNum        = 100
	defaultMaxMsgSize       int64 = 4096
	defaultHandshakeTimeout       = 10 * time.Second
	defaultConnectInterval        = 3 * time.Second
)

type (
	WsClientConf struct {
		// Server address
		Addr string
		// Number of connections to the server
		ConnNum int
		// Number of pending writes
		PendingWriteNum int
		// Maximum message size
		MaxMsgSize int64
		// Interval between connection attempts
		ConnectInterval time.Duration
		// Handshake timeout duration
		HandshakeTimeout time.Duration
		// Automatically reconnect on close
		AutoReconnect bool
	}
	WsClient struct {
		sync.Mutex
		wg sync.WaitGroup
		// Configuration for the client
		conf   *WsClientConf
		dialer websocket.Dialer
		// Agent interface for handling messages
		// The agent is responsible for handling messages and events
		agent func(*WsConn) network.Agent
		// Connection pool
		conns WsConns
		// Flag to indicate if the client is closed
		done bool
	}
)

// NewClient creates a new WebSocket client with the given configuration
// The client is responsible for establishing connections to the server and handling messages
func NewClient(conf *WsClientConf) *WsClient {
	defaultConf(conf)

	return &WsClient{
		conf:  conf,
		conns: make(WsConns),
		dialer: websocket.Dialer{
			HandshakeTimeout: conf.HandshakeTimeout,
		},
	}
}

// WithAgent sets the agent for the client
// The agent is responsible for handling messages and events
func (c *WsClient) WithAgent(agent func(*WsConn) network.Agent) {
	c.Lock()
	defer c.Unlock()

	c.agent = agent
}

// Start the client and establish connections to the server
func (c *WsClient) Start() {
	// Set the maximum number of connections to the server
	for range c.conf.ConnNum {
		c.wg.Add(1)
		go c.connect()
	}
}

func (c *WsClient) dial() *websocket.Conn {
	for {
		conn, _, err := c.dialer.Dial(c.conf.Addr, nil)
		if err == nil || c.done {
			return conn
		}

		time.Sleep(c.conf.ConnectInterval)
		continue
	}
}

// Connect to the server and handle messages
// This function is responsible for establishing a connection to the server
func (c *WsClient) connect() {
	defer c.wg.Done()

Reconnect:
	conn := c.dial()
	if conn == nil {
		return
	}

	//	Set the read limit for the connection
	// This is important to prevent large messages from being read into memory
	conn.SetReadLimit(c.conf.MaxMsgSize)
	c.Lock()

	if c.done {
		c.Unlock()
		conn.Close()
		return
	}

	c.conns[conn] = struct{}{}
	c.Unlock()

	// Set the connection to read/write mode
	wsconn := NewConn(conn, &WsConnConf{
		pendingWriteNum: c.conf.PendingWriteNum,
		MaxMsgSize:      uint32(c.conf.MaxMsgSize),
	})

	agent := c.agent(wsconn)
	agent.Run()

	wsconn.Close()

	c.Lock()
	delete(c.conns, conn)
	c.Unlock()
	agent.OnClose()

	if c.conf.AutoReconnect {
		time.Sleep(c.conf.ConnectInterval)
		goto Reconnect
	}
}

// Close the client and all connections
// This function is responsible for closing all connections and cleaning up resources
func (c *WsClient) Close() {
	c.Lock()
	c.done = true
	for conn := range c.conns {
		conn.Close()
	}
	c.conns = nil
	c.Unlock()

	c.wg.Wait()
}

func defaultConf(conf *WsClientConf) {
	if conf.ConnNum <= 0 {
		conf.ConnNum = defaultConnNum
	}
	if conf.ConnectInterval <= 0 {
		conf.ConnectInterval = defaultConnectInterval
	}
	if conf.PendingWriteNum <= 0 {
		conf.PendingWriteNum = defaultPendingWriteNum
	}
	if conf.MaxMsgSize <= 0 {
		conf.MaxMsgSize = defaultMaxMsgSize
	}
	if conf.HandshakeTimeout <= 0 {
		conf.HandshakeTimeout = defaultHandshakeTimeout
	}
}
