package unix

import (
	"errors"
	"net"
	"sync"

	"github.com/czx-lab/czx/network"
	"github.com/czx-lab/czx/xlog"
)

type (
	ClientConf struct {
		Path string
	}
	Client struct {
		wg    sync.WaitGroup
		mu    sync.Mutex
		conf  ClientConf
		conns map[net.Conn]struct{}
		agent func(net.Conn) network.Agent
		flag  bool
	}
)

func NewClient(conf ClientConf) (*Client, error) {
	client := &Client{
		conf:  conf,
		conns: make(map[net.Conn]struct{}),
	}

	return client, nil
}

// WithAgent sets the agent function for the client.
func (c *Client) WithAgent(agent func(net.Conn) network.Agent) *Client {
	c.agent = agent
	return c
}

// Connect establishes a Unix domain socket connection to the server and starts the agent.
func (c *Client) Connect() error {
	if c.flag {
		return errors.New("client stopped")
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		if err := c.connect(); err != nil {
			xlog.Write().Sugar().Errorf("unix client connect error: %w", err)
		}
	}()

	return nil
}

// connect establishes a Unix domain socket connection to the server and starts the agent.
func (c *Client) connect() error {
	conn, err := c.dial()
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conns[conn] = struct{}{}
	c.mu.Unlock()
	if c.agent == nil {
		return errors.New("agent not set")
	}
	agent := c.agent(conn)
	agent.Run()

	conn.Close()
	agent.OnClose()
	c.mu.Lock()
	delete(c.conns, conn)
	c.mu.Unlock()
	return nil
}

// dial establishes a Unix domain socket connection to the server.
func (c *Client) dial() (net.Conn, error) {
	conn, err := net.Dial("unix", c.conf.Path)
	if err != nil {
		xlog.Write().Sugar().Errorf("unix conn error: %w", err)
		return nil, err
	}
	return conn, nil
}

// Stop stops the client and closes all active connections.
func (c *Client) Stop() {
	c.mu.Lock()

	if c.flag {
		c.mu.Unlock()
		return
	}
	c.flag = true

	for conn := range c.conns {
		conn.Close()
		delete(c.conns, conn)
	}
	c.mu.Unlock()
	c.wg.Wait()
}
