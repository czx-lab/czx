package xkcp

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/czx-lab/czx/network"
	"github.com/czx-lab/czx/network/metrics"
	"github.com/czx-lab/czx/network/tcp"
	"github.com/czx-lab/czx/prometheus"
	"github.com/xtaci/kcp-go/v5"
)

const (
	defaultCryptKey     = "default_czx_key" // Default encryption key
	defaultDataShards   = 10                // Default number of data shards
	defaultParityShards = 3                 // Default number of parity shards
	// defaultMaxConn is the default maximum number of connections
	defaultMaxConn = 1000

	// KCP default Parameters
	defaultNoDelay  = 1
	defaultInterval = 10
	defaultResend   = 2
	defaultNC       = 1
)

type (
	KcpServerConf struct {
		tcp.TcpConnConf
		CryptKey []byte // Key for encryption
		Addr     string // Address to listen on
		// Number of data shards
		DataShards int
		// Number of parity shards
		ParityShards int
		// Maximum number of connections
		MaxConn int
		// If ImmediateRelease is true, the server will release resources immediately after stopping.
		// This may lead to abrupt disconnections for active connections.
		// If false, the server will wait for all active connections to close gracefully before releasing resources.
		// Default is false.
		ImmediateRelease bool
		Metrics          metrics.SvrMetricsConf

		// KCP Parameters
		NoDelay  *int
		Interval *int
		Resend   *int
		NC       *int
	}
	KcpServer struct {
		sync.Mutex
		conf     KcpServerConf
		ln       net.Listener
		lnWait   sync.WaitGroup
		connWait sync.WaitGroup
		conns    tcp.Conns // Map of connections
		agent    func(*tcp.TcpConn) network.Agent
		metrics  network.ServerMetrics
	}
)

func NewKcpServer(conf KcpServerConf, agent func(*tcp.TcpConn) network.Agent) *KcpServer {
	defaultConf(&conf)
	var m network.ServerMetrics
	if prometheus.Enabled() {
		m = metrics.NewSvrMetrics(conf.Metrics)
	} else {
		m = &network.NoopServerMetrics{}
	}
	return &KcpServer{
		conf:    conf,
		agent:   agent,
		conns:   make(tcp.Conns),
		metrics: m,
	}
}

// Start initializes the KCP server and starts listening for connections.
func (srv *KcpServer) Start() error {
	block, err := kcp.NewAESBlockCrypt(srv.conf.CryptKey)
	if err != nil {
		return err
	}
	srv.ln, err = kcp.ListenWithOptions(srv.conf.Addr, block, srv.conf.DataShards, srv.conf.ParityShards)
	if err != nil {
		return err
	}

	go srv.run()

	return nil
}

func (srv *KcpServer) run() {
	srv.lnWait.Add(1)
	defer srv.lnWait.Done()

	// Delay for retrying connection acceptance
	var delay time.Duration

	for {
		conn, err := srv.ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if delay == 0 {
				delay = 5 * time.Millisecond
			} else {
				delay *= 2
			}
			if max := 1 * time.Second; delay > max {
				delay = max
			}

			time.Sleep(delay)
			continue
		}

		delay = 0

		if kcpConn, ok := conn.(*kcp.UDPSession); ok {
			kcpConn.SetNoDelay(*srv.conf.NoDelay, *srv.conf.Interval, *srv.conf.Resend, *srv.conf.NC)
		}

		srv.Lock()

		// Check if the maximum number of connections has been reached
		if len(srv.conns) >= srv.conf.MaxConn {
			srv.Unlock()
			srv.metrics.IncFailedConns()
			conn.Close()
			continue
		}

		// Add the new connection to the map
		srv.conns[conn] = struct{}{}
		srv.Unlock()
		srv.metrics.IncConns()
		srv.metrics.IncTotalConns()
		srv.connWait.Add(1)

		kcpconn := tcp.NewTcpConn(conn, &srv.conf.TcpConnConf).WithMetrics(srv.metrics)
		agent := srv.agent(kcpconn)

		ip, port, _ := network.GetClientIPFromProxyProtocol(conn)
		// Set the IP and port in the agent
		clientAddr := network.ClientAddrMessage{IP: *ip, Port: *port}
		kcpconn.WithClientAddr(clientAddr)

		agent.OnPreConn(clientAddr)

		start_t := time.Now()
		go func() {
			defer func() {
				srv.metrics.DecConns()
				srv.metrics.ObserveConnDuration(time.Since(start_t))
			}()
			agent.Run()
			// Release resources based on the ImmediateRelease configuration
			if srv.conf.ImmediateRelease {
				kcpconn.Destroy()
			} else {
				kcpconn.Close()
			}

			srv.Lock()
			delete(srv.conns, conn)
			srv.Unlock()
			agent.OnClose()

			srv.connWait.Done()
		}()
	}
}

// Stop stops the KCP server and closes all connections.
// It waits for all connections to finish processing before returning.
func (srv *KcpServer) Stop() {
	srv.ln.Close()
	srv.lnWait.Wait()

	srv.Lock()

	// Remove all connections from the map
	for conn := range srv.conns {
		conn.Close()
	}
	srv.conns = make(tcp.Conns)

	srv.Unlock()

	srv.connWait.Wait()
}

func defaultConf(conf *KcpServerConf) {
	if conf.DataShards <= 0 {
		conf.DataShards = defaultDataShards
	}
	if conf.ParityShards <= 0 {
		conf.ParityShards = defaultParityShards
	}
	if conf.CryptKey == nil {
		conf.CryptKey = []byte(defaultCryptKey) // Use a default key if none provided
	}
	if conf.MaxConn <= 0 {
		conf.MaxConn = defaultMaxConn
	}
	if conf.NoDelay == nil {
		v := defaultNoDelay
		conf.NoDelay = &v
	}
	if conf.Interval == nil {
		v := defaultInterval
		conf.Interval = &v
	}
	if conf.Resend == nil {
		v := defaultResend
		conf.Resend = &v
	}
	if conf.NC == nil {
		v := defaultNC
		conf.NC = &v
	}
}
