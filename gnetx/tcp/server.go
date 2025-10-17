package tcp

import (
	"context"
	"strings"
	"time"

	"github.com/czx-lab/czx/network"
	xtcp "github.com/czx-lab/czx/network/tcp"
	"github.com/czx-lab/czx/xlog"

	"github.com/panjf2000/gnet/v2"
	"go.uber.org/zap"
)

var (
	defaultKeepAlive  uint64 = 2 * 60 // 2 minutes
	defaultMsgMinSize uint32 = 1
)

type (
	GnetTcpServerConf struct {
		GnetTcpConnConf
		xtcp.MessageParserConf
		Addr      string
		KeepAlive uint64
		NoDelay   gnet.TCPSocketOpt
		Multicore bool
		Ticker    bool
		// Maximum number of connections
		MaxConn int
		// If ImmediateRelease is true, the server will release resources immediately after stopping.
		// This may lead to abrupt disconnections for active connections.
		// If false, the server will wait for all active connections to close gracefully before releasing resources.
		// Default is false.
		ImmediateRelease bool
	}
	GnetTcpServer struct {
		conf   *GnetTcpServerConf
		eng    gnet.Engine
		agent  func(network.Conn) network.Agent
		delay  time.Duration
		tickFn func()
		parse  *xtcp.MessageParser
	}
)

var _ gnet.EventHandler = (*GnetTcpServer)(nil)

func NewGNetTcpServer(conf *GnetTcpServerConf, agent func(network.Conn) network.Agent) *GnetTcpServer {
	defaultConf(conf)

	return &GnetTcpServer{
		conf:  conf,
		agent: agent,
		parse: xtcp.NewParse(&conf.MessageParserConf),
	}
}

// WithTick sets the tick function and delay for the server
// The tick function is called every delay duration
func (g *GnetTcpServer) WithTick(delay time.Duration, fn func()) {
	g.delay = delay
	g.tickFn = fn
}

func (g *GnetTcpServer) Start() {
	addrs := strings.Split(g.conf.Addr, ":")
	opts := []gnet.Option{
		gnet.WithMulticore(g.conf.Multicore),
		gnet.WithTCPKeepAlive(time.Duration(g.conf.KeepAlive) * time.Second),
		gnet.WithTicker(g.conf.Ticker),
		gnet.WithTCPNoDelay(g.conf.NoDelay),
	}

	go gnet.Run(g, strings.TrimSpace(addrs[0]), opts...)
}

func (g *GnetTcpServer) Stop() {
	g.eng.Stop(context.Background())
}

// OnClose implements gnet.EventHandler.
func (es *GnetTcpServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	switch agent := c.Context().(type) {
	case network.Agent:
		agent.OnClose()
	}
	c.SetContext(nil)
	return gnet.None
}

// OnOpen implements gnet.EventHandler.
func (es *GnetTcpServer) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	if es.eng.CountConnections() > es.conf.MaxConn {
		xlog.Write().Warn("too many connections", zap.Int("max", es.conf.MaxConn))
		return nil, gnet.Close
	}
	conn := NewGnetConn(c, &es.conf.GnetTcpConnConf).WithParse(es.parse)
	agent := es.agent(conn)

	ip, port, _ := network.GetClientIPFromProxyProtocol(c)
	// Set the IP and port in the agent
	clientAddr := network.ClientAddrMessage{IP: *ip, Port: *port}
	agent.OnPreConn(clientAddr)

	conn.withClientAddr(clientAddr)
	c.SetContext(agent)

	return nil, gnet.None
}

// OnShutdown implements gnet.EventHandler.
func (es *GnetTcpServer) OnShutdown(eng gnet.Engine) {
}

// OnTick implements gnet.EventHandler.
func (es *GnetTcpServer) OnTick() (delay time.Duration, action gnet.Action) {
	if !es.conf.Ticker {
		return
	}

	if es.tickFn != nil {
		es.tickFn()
	}

	return es.delay, gnet.None
}

// OnBoot implements gnet.EventHandler.
func (es *GnetTcpServer) OnBoot(eng gnet.Engine) gnet.Action {
	es.eng = eng
	return gnet.None
}

// OnTraffic implements gnet.EventHandler.
func (es *GnetTcpServer) OnTraffic(c gnet.Conn) gnet.Action {
	buf, err := c.Next(-1)
	if err != nil {
		xlog.Write().Error("gnet tcp server read error: %v", zap.Error(err))
		return gnet.Close
	}

	switch agent := c.Context().(type) {
	case network.GnetAgent:
		agent.React(buf)
	default:
		return gnet.Close
	}

	return gnet.None
}

func defaultConf(conf *GnetTcpServerConf) {
	if conf.PendingWrite <= 0 {
		conf.PendingWrite = defaultPendingWrite
	}
	if conf.MsgMinSize <= 0 {
		conf.MsgMinSize = defaultMsgMinSize
	}
	if conf.KeepAlive == 0 {
		conf.KeepAlive = defaultKeepAlive
	}
}
