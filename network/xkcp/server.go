package xkcp

import (
	"net"
	"sync"
	"time"

	"github.com/czx-lab/czx/network"
	"github.com/czx-lab/czx/network/tcp"
	"github.com/xtaci/kcp-go/v5"
)

const (
	defaultCryptKey     = "default_czx_key" // Default encryption key
	defaultDataShards   = 10                // Default number of data shards
	defaultParityShards = 3                 // Default number of parity shards
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
	}
	KcpServer struct {
		sync.Mutex
		conf     KcpServerConf
		ln       net.Listener
		lnWait   sync.WaitGroup
		connWait sync.WaitGroup
		conns    tcp.Conns // Map of connections
		agent    func(*tcp.TcpConn) network.Agent
	}
)

func NewKcpServer(conf KcpServerConf, agent func(*tcp.TcpConn) network.Agent) *KcpServer {
	defaultConf(&conf)

	return &KcpServer{
		conf:  conf,
		agent: agent,
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
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
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
			return
		}

		delay = 0

		srv.Lock()

		// Check if the maximum number of connections has been reached
		if len(srv.conns) >= srv.conf.MaxConn {
			srv.Unlock()
			conn.Close()
			continue
		}

		// Add the new connection to the map
		srv.conns[conn] = struct{}{}
		srv.Unlock()
		srv.connWait.Add(1)

		kcpconn := tcp.NewTcpConn(conn, &srv.conf.TcpConnConf)
		agent := srv.agent(kcpconn)

		ip, port, _ := network.GetClientIPFromProxyProtocol(conn)
		// Set the IP and port in the agent
		clientAddr := network.ClientAddrMessage{IP: *ip, Port: *port}
		kcpconn.WithClientAddr(clientAddr)

		agent.OnPreConn(clientAddr)

		go func() {
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
}
