package ws

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/czx-lab/czx/network"
	"github.com/czx-lab/czx/network/metrics"
	"github.com/czx-lab/czx/prometheus"
	"github.com/czx-lab/czx/xlog"
	"go.uber.org/zap"

	"github.com/gorilla/websocket"
)

type WsServerConf struct {
	Addr            string
	CertFile        string
	KeyFile         string
	MaxConn         int
	PendingWriteNum int
	Timeout         int
	MaxMsgSize      uint32
	// If ImmediateRelease is true, the server will release resources immediately after stopping.
	// This may lead to abrupt disconnections for active connections.
	// If false, the server will wait for all active connections to close gracefully before releasing resources.
	// Default is false.
	ImmediateRelease bool
	// Metrics configuration
	Metrics metrics.SvrMetricsConf
}

type WsHandler struct {
	opt      *WsServerConf
	mu       sync.Mutex
	wg       sync.WaitGroup
	upgrader websocket.Upgrader
	conns    WsConns
	agent    func(*WsConn) network.Agent
	metrics  network.ServerMetrics
}

type WsServer struct {
	opt     *WsServerConf
	ln      net.Listener
	handler *WsHandler
}

func NewServer(opt *WsServerConf, agent func(*WsConn) network.Agent) *WsServer {
	var m network.ServerMetrics
	if prometheus.Enabled() {
		m = metrics.NewSvrMetrics(opt.Metrics)
	} else {
		m = &network.NoopServerMetrics{}
	}
	return &WsServer{
		opt: opt,
		handler: &WsHandler{
			opt:     opt,
			agent:   agent,
			conns:   make(WsConns),
			metrics: m,
		},
	}
}

func (handler *WsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// todo:: Allow all origins for simplicity; customize as needed
	handler.upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	start_t := time.Now()
	conn, err := handler.upgrader.Upgrade(w, r, nil)
	if err != nil {
		handler.metrics.IncFailedConns()
		http.Error(w, fmt.Sprintf("Failed to upgrade connection: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	handler.wg.Add(1)
	defer func() {
		handler.metrics.DecConns()
		handler.metrics.ObserveConnDuration(time.Since(start_t))
		handler.wg.Done()
	}()

	handler.mu.Lock()

	// Check if the maximum number of connections has been reached
	// If so, close the connection and return an error response
	if len(handler.conns) >= handler.opt.MaxConn {
		xlog.Write().Warn("too many connections", zap.Int("max", handler.opt.MaxConn))
		handler.mu.Unlock()
		handler.metrics.IncFailedConns()
		conn.Close()
		return
	}

	handler.conns[conn] = struct{}{}

	handler.mu.Unlock()
	handler.metrics.IncConns()
	handler.metrics.IncTotalConns()

	wsconn := NewConn(conn, &WsConnConf{
		MaxMsgSize:      handler.opt.MaxMsgSize,
		PendingWriteNum: handler.opt.PendingWriteNum,
	}).WithMetrics(handler.metrics)

	agent := handler.agent(wsconn)
	ip, port := network.GetClientIP(r)

	// Set the IP and port in the agent
	clentAddr := network.ClientAddrMessage{IP: *ip, Port: *port, Req: r}
	wsconn.withClientAddr(clentAddr)
	agent.OnPreConn(clentAddr)
	agent.Run()

	// Release resources based on the ImmediateRelease configuration
	if handler.opt.ImmediateRelease {
		wsconn.Destroy()
	} else {
		wsconn.Close()
	}

	handler.mu.Lock()
	delete(handler.conns, conn)
	handler.mu.Unlock()
	agent.OnClose()
}

// Start starts the WebSocket server and listens for incoming connections.
// It will use the provided address and TLS configuration if specified.
func (server *WsServer) Start() error {
	ln, err := net.Listen("tcp", server.opt.Addr)
	if err != nil {
		return err
	}

	if len(server.opt.CertFile) > 0 || len(server.opt.KeyFile) > 0 {
		config := &tls.Config{
			NextProtos: []string{"http/1.1"},
		}

		var err error
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0], err = tls.LoadX509KeyPair(server.opt.CertFile, server.opt.KeyFile)
		if err != nil {
			return err
		}

		ln = tls.NewListener(ln, config)
	}

	server.ln = ln

	httpServer := &http.Server{
		Addr:           server.opt.Addr,
		Handler:        server.handler,
		ReadTimeout:    time.Duration(server.opt.Timeout) * time.Second,
		WriteTimeout:   time.Duration(server.opt.Timeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go httpServer.Serve(ln)
	return nil
}

// Stop stops the WebSocket server and closes all connections.
// It will also wait for all connections to be closed before returning.
func (server *WsServer) Stop() {
	if server.ln != nil {
		server.ln.Close()
	}

	server.handler.mu.Lock()

	for conn := range server.handler.conns {
		conn.Close()
	}

	// Clear the connections map to prevent new connections from being accepted
	// and to allow the goroutine to exit cleanly.
	server.handler.conns = nil
	server.handler.mu.Unlock()

	server.handler.wg.Wait()
}
