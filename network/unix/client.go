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
func (c *Client) Connect() (network.Agent, error) {
	if c.flag {
		return nil, errors.New("client stopped")
	}
	if c.agent == nil {
		return nil, errors.New("agent is nil")
	}

	conn, err := c.dial()
	if err != nil {
		return nil, err
	}

	agent := c.agent(conn)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		if err := c.connect(conn, agent); err != nil {
			xlog.Write().Sugar().Errorf("unix client connect error: %v", err)
		}
	}()

	return agent, nil
}

// connect establishes a Unix domain socket connection to the server and starts the agent.
func (c *Client) connect(conn net.Conn, agent network.Agent) error {
	c.mu.Lock()
	c.conns[conn] = struct{}{}
	c.mu.Unlock()

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
		xlog.Write().Sugar().Errorf("unix conn error: %v", err)
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
