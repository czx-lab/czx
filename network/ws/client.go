package ws

import (
	"sync"
	"time"

	"github.com/czx-lab/czx/network"
	"github.com/czx-lab/czx/xlog"
	"github.com/gorilla/websocket"
)

type (
	ClientConf struct {
		WsConnConf
		Addr             string
		ConnInterval     uint
		Num              uint
		HandshakeTimeout uint
		AutoReconnect    bool
	}
	Client struct {
		sync.Mutex
		conf   ClientConf
		dialer websocket.Dialer
		conns  WsConns
		agent  func(*WsConn) network.Agent
		wg     sync.WaitGroup
		done   bool
	}
)

func NewClient(conf ClientConf, agent func(*WsConn) network.Agent) *Client {
	defaultClientConf(&conf)

	return &Client{
		conf:  conf,
		agent: agent,
		conns: make(WsConns),
	}
}

// Start the client and establish connections to the server.
// It will create the specified number of connections and start the agent for each connection.
func (c *Client) Start() {
	c.Lock()
	defer c.Unlock()

	c.dialer = websocket.Dialer{
		HandshakeTimeout: time.Duration(c.conf.HandshakeTimeout) * time.Second,
	}

	for range c.conf.Num {
		c.wg.Add(1)

		go c.connect()
	}
}

// connect establishes a WebSocket connection to the server and starts the agent.
// It will automatically reconnect if the connection is closed and AutoReconnect is enabled.
func (c *Client) connect() {
	defer c.wg.Done()

Reconnect:
	conn := c.dial()
	if conn == nil {
		return
	}
	conn.SetReadLimit(int64(c.conf.MaxMsgSize))
	c.Lock()
	if c.done {
		c.Unlock()
		conn.Close()
		return
	}
	c.conns[conn] = struct{}{}
	c.Unlock()
	wsconn := NewConn(conn, &c.conf.WsConnConf)
	agent := c.agent(wsconn)
	agent.Run()

	// Close the connection and remove it from the map
	wsconn.Close()
	c.Lock()
	delete(c.conns, conn)
	c.Unlock()
	agent.OnClose()

	// If AutoReconnect is enabled, wait for the specified interval before reconnecting
	// Otherwise, exit the loop
	if c.conf.AutoReconnect {
		time.Sleep(time.Duration(c.conf.ConnInterval) * time.Second)
		goto Reconnect
	}
}

func (c *Client) dial() *websocket.Conn {
	for {
		conn, _, err := c.dialer.Dial(c.conf.Addr, nil)
		if err == nil || c.done {
			return conn
		}

		xlog.Write().Sugar().Infof("connect to %v error: %v", c.conf.Addr, err)
		time.Sleep(time.Duration(c.conf.ConnInterval) * time.Second)
		continue
	}
}

// Stop stops the client and closes all connections.
// It will also wait for all connections to be closed before returning.
func (c *Client) Stop() {
	c.Lock()

	if c.done {
		return
	}
	c.done = true

	for conn := range c.conns {
		conn.Close()
	}

	c.conns = nil
	c.Unlock()

	c.wg.Wait()
}

func defaultClientConf(conf *ClientConf) {
	if conf.Num == 0 {
		conf.Num = 1
	}
	if conf.ConnInterval == 0 {
		conf.ConnInterval = 3
	}
	if conf.MaxMsgSize == 0 {
		conf.MaxMsgSize = 1024 * 1024 // 1MB
	}
	if conf.PendingWriteNum == 0 {
		conf.PendingWriteNum = 4096
	}
	if conf.HandshakeTimeout == 0 {
		conf.HandshakeTimeout = 10
	}
}
