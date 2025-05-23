package tcp

import (
	"net"
	"sync"
	"time"

	"github.com/czx-lab/czx/network"
)

var (
	defaultMaxConn = 1000
)

type (
	TcpServerConf struct {
		TcpConnConf
		MessageParserConf
		// TCP server address
		Addr string
		// Maximum number of connections
		MaxConn int
	}
	TcpServer struct {
		sync.Mutex
		connWait sync.WaitGroup
		lnWait   sync.WaitGroup
		conf     *TcpServerConf
		// Map of connections
		conns Conns
		ln    net.Listener

		agent func(*TcpConn) network.Agent
		parse *MessageParser
	}
)

func NewServer(conf *TcpServerConf, agent func(*TcpConn) network.Agent) *TcpServer {
	defaultConf(conf)

	return &TcpServer{
		conf:  conf,
		conns: make(Conns),
		agent: agent,
		parse: NewParse(&conf.MessageParserConf),
	}
}

// Start starts the TCP server and begins accepting connections
// It returns an error if the server fails to start
func (srv *TcpServer) Start() error {
	ln, err := net.Listen("tcp", srv.conf.Addr)
	if err != nil {
		return err
	}

	srv.ln = ln

	go srv.run()
	return nil
}

func (srv *TcpServer) run() {
	srv.lnWait.Add(1)
	defer srv.lnWait.Done()

	// Delay for retrying connection acceptance
	var delay time.Duration

	// Accept connections in a loop
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

		srv.conns[conn] = struct{}{}
		srv.Unlock()
		srv.connWait.Add(1)

		tcpconn := NewTcpConn(conn, &srv.conf.TcpConnConf).WithParse(srv.parse)
		agent := srv.agent(tcpconn)

		ip, port, _ := network.GetClientIPFromProxyProtocol(conn)
		// Set the IP and port in the agent
		clientAddr := network.ClientAddrMessage{IP: *ip, Port: *port}
		tcpconn.withClientAddr(clientAddr)

		agent.OnPreConn(clientAddr)

		go func() {
			agent.Run()

			tcpconn.Close()
			srv.Lock()
			delete(srv.conns, conn)
			srv.Unlock()
			agent.OnClose()

			srv.connWait.Done()
		}()
	}
}

// Close closes the server and all connections
func (srv *TcpServer) Stop() {
	srv.ln.Close()
	srv.lnWait.Wait()

	srv.Lock()

	// Remove all connections from the map
	for conn := range srv.conns {
		conn.Close()
	}
	srv.conns = make(Conns)

	srv.Unlock()

	srv.connWait.Wait()
}

func defaultConf(conf *TcpServerConf) {
	if conf.MaxConn <= 0 {
		conf.MaxConn = defaultMaxConn
	}
	if conf.PendingWrite <= 0 {
		conf.PendingWrite = defaultPendingWrite
	}
	if conf.MsgMinSize <= 0 {
		conf.MsgMinSize = defaultMsgMinSize
	}
}
