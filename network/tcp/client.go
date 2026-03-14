package tcp

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"

	"github.com/czx-lab/czx/network"
	"github.com/czx-lab/czx/xlog"
)

type (
	TcpClientConf struct {
		Addr    string
		Pending int

		// message parser
		ParserConf MessageParserConf
	}
	TcpClient struct {
		mu        sync.Mutex
		conf      TcpClientConf
		agent     func(*TcpConn) network.Agent
		closeFlag atomic.Bool
		parser    *MessageParser
		conns     Conns

		wg sync.WaitGroup
	}
)

// NewTcpClient .
func NewTcpClient(conf TcpClientConf) *TcpClient {
	return &TcpClient{
		conf:   conf,
		conns:  make(Conns),
		parser: NewParse(&conf.ParserConf),
	}
}

func (c *TcpClient) WithAgent(agent func(*TcpConn) network.Agent) *TcpClient {
	c.agent = agent
	return c
}

func (c *TcpClient) Connect() (network.Agent, error) {
	if c.closeFlag.Load() {
		return nil, errors.New("client stopped")
	}
	if c.agent == nil {
		return nil, errors.New("agent is nil")
	}

	tconn, conn, err := c.dial()
	if err != nil {
		return nil, err
	}

	agent := c.agent(tconn)

	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		if err := c.connect(tconn, conn, agent); err != nil {
			xlog.Write().Sugar().Errorf("tcp client connect error: %v", err)
		}
	}()

	return agent, nil
}

func (c *TcpClient) connect(tconn *TcpConn, conn net.Conn, agent network.Agent) error {
	agent.Run()

	tconn.Close()
	agent.OnClose()

	c.mu.Lock()
	delete(c.conns, conn)
	c.mu.Unlock()

	return nil
}

func (c *TcpClient) dial() (*TcpConn, net.Conn, error) {
	conn, err := net.Dial("tcp", c.conf.Addr)
	if err != nil {
		return nil, nil, err
	}

	tcpconn := NewTcpConn(conn, &TcpConnConf{
		PendingWrite: c.conf.Pending,
	})

	c.mu.Lock()
	c.conns[conn] = struct{}{}
	c.mu.Unlock()

	return tcpconn, conn, nil
}

func (c *TcpClient) Close() {
	if c.closeFlag.Load() {
		return
	}
	c.closeFlag.Store(true)

	c.mu.Lock()
	for conn := range c.conns {
		conn.Close()
		delete(c.conns, conn)
	}
	c.mu.Unlock()

	c.wg.Wait()
}
