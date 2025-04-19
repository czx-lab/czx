package ws

import (
	"crypto/tls"
	"czx/network"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WsServerConf struct {
	Addr            string
	CertFile        string
	KeyFile         string
	MaxConn         int
	PendingWriteNum int
	MaxMsgSize      uint32
	Timeout         time.Duration
}

type WsHandler struct {
	opt      *WsServerConf
	mu       sync.Mutex
	wg       sync.WaitGroup
	upgrader websocket.Upgrader
	conns    WsConns
	agent    func(*WsConn) network.Agent
}

type WsServer struct {
	opt     *WsServerConf
	ln      net.Listener
	handler *WsHandler
}

func NewServer(opt *WsServerConf, agent func(*WsConn) network.Agent) *WsServer {
	return &WsServer{
		opt: opt,
		handler: &WsHandler{
			opt:   opt,
			agent: agent,
			conns: make(WsConns),
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

	conn, err := handler.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to upgrade connection: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	handler.wg.Add(1)
	defer handler.wg.Done()

	handler.mu.Lock()

	// Check if the maximum number of connections has been reached
	// If so, close the connection and return an error response
	if len(handler.conns) >= handler.opt.MaxConn {
		handler.mu.Unlock()
		conn.Close()
		http.Error(w, "Too many connections", http.StatusServiceUnavailable)
		return
	}

	handler.conns[conn] = struct{}{}

	handler.mu.Unlock()

	wsconn := NewConn(conn, &WsConnConf{
		MaxMsgSize:      handler.opt.MaxMsgSize,
		pendingWriteNum: handler.opt.PendingWriteNum,
	})

	agent := handler.agent(wsconn)
	ip, port := network.GetClientIP(r)

	// Set the IP and port in the agent
	clentAddr := network.ClientAddrMessage{IP: *ip, Port: *port, Req: r}
	wsconn.withClientAddr(clentAddr)
	agent.OnPreConn(clentAddr)
	agent.Run()

	wsconn.Close()

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
		ReadTimeout:    server.opt.Timeout,
		WriteTimeout:   server.opt.Timeout,
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
