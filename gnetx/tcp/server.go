package tcp

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/czx-lab/czx/network"
	"github.com/czx-lab/czx/network/metrics"
	xtcp "github.com/czx-lab/czx/network/tcp"
	"github.com/czx-lab/czx/prometheus"
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
		Metrics          metrics.SvrMetricsConf
	}
	GnetTcpServer struct {
		mu       sync.Mutex
		conf     *GnetTcpServerConf
		eng      gnet.Engine
		agent    func(network.Conn) network.Agent
		delay    time.Duration
		tickFn   func()
		parse    *xtcp.MessageParser
		connWait sync.WaitGroup
		conns    Conns
		metrics  network.ServerMetrics
	}
)

var _ gnet.EventHandler = (*GnetTcpServer)(nil)

func NewGNetTcpServer(conf *GnetTcpServerConf, agent func(network.Conn) network.Agent) *GnetTcpServer {
	defaultConf(conf)
	var m network.ServerMetrics
	if prometheus.Enabled() {
		m = metrics.NewSvrMetrics(conf.Metrics)
	} else {
		m = &network.NoopServerMetrics{}
	}

	return &GnetTcpServer{
		conf:    conf,
		agent:   agent,
		parse:   xtcp.NewParse(&conf.MessageParserConf),
		conns:   make(Conns),
		metrics: m,
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
	if g.conf.ImmediateRelease {
		// Immediately close all connections
		g.mu.Lock()
		for conn := range g.conns {
			conn.Destroy()
		}
		g.mu.Unlock()
	}

	// Stop the engine, which will trigger OnClose for all connections
	g.eng.Stop(context.Background())

	// Wait for all agent goroutines to finish
	g.connWait.Wait()

	// Clear the connection map
	g.mu.Lock()
	g.conns = make(Conns)
	g.mu.Unlock()
}

// OnClose implements gnet.EventHandler.
func (es *GnetTcpServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	// Get the connection from context and close it
	if conn, ok := c.Context().(*GnetConn); ok && conn != nil {
		// Close the connection to signal agent.Run() to exit
		if es.conf.ImmediateRelease {
			conn.Destroy()
		} else {
			conn.Close()
		}
	}
	c.SetContext(nil)
	return gnet.None
}

// OnOpen implements gnet.EventHandler.
func (es *GnetTcpServer) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	if es.eng.CountConnections() > es.conf.MaxConn {
		es.metrics.IncFailedConns()
		xlog.Write().Warn("too many connections", zap.Int("max", es.conf.MaxConn))
		return nil, gnet.Close
	}
	conn := NewGnetConn(c, &es.conf.GnetTcpConnConf).WithParse(es.parse)

	ip, port, _ := network.GetClientIPFromProxyProtocol(c)
	clientAddr := network.ClientAddrMessage{IP: *ip, Port: *port}
	conn.withClientAddr(clientAddr)

	// Store conn in context for fast lookup in OnTraffic
	c.SetContext(conn)

	agent := es.agent(conn)
	es.mu.Lock()
	es.conns[conn] = struct{}{}
	es.mu.Unlock()

	es.metrics.IncConns()
	es.metrics.IncTotalConns()

	agent.OnPreConn(clientAddr)

	es.connWait.Add(1)
	start_t := time.Now()
	go func() {
		defer func() {
			es.metrics.DecConns()
			es.metrics.ObserveConnDuration(time.Since(start_t))
		}()

		agent.Run()

		// agent.Run() has exited, clean up resources
		es.mu.Lock()
		delete(es.conns, conn)
		es.mu.Unlock()

		agent.OnClose()

		es.connWait.Done()
	}()

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

	// Fast lookup: get GnetConn directly from context
	conn, ok := c.Context().(*GnetConn)
	if !ok || conn == nil {
		return gnet.Close
	}

	// Write data to buffer
	if _, err := conn.WriteBuffer(buf); err != nil {
		xlog.Write().Error("gnet tcp server write buffer error: %v", zap.Error(err))
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
