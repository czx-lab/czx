package agent

import (
	"czx/network"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var (
	ProcessorErr = errors.New("processor is nil")
)

type (
	GateConf struct {
		network.WsServerConf
	}
	Gate struct {
		option    GateConf
		processor network.Processor
		wsSrv     *network.WsServer
	}
	// agent implements network.Agent interface
	// It is used to handle the connection and process messages.
	agent struct {
		conn network.Conn
		gate *Gate
	}
)

var _ network.Agent = (*agent)(nil)

func NewGate(opt GateConf) *Gate {
	return &Gate{
		option: opt,
	}
}

func (g *Gate) Start() {
	var wsServer *network.WsServer
	if len(g.option.WsServerConf.Addr) > 0 {
		wsServer = network.NewWSServer(&g.option.WsServerConf)
		wsServer.Start()
	}

	// Handle graceful shutdown on Ctrl+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-sig
	g.wsSrv.Stop()
}

func (g *Gate) Stop() {
}

func (a *agent) Run() {
	for {
		data, err := a.conn.Read()
		if err != nil {
			// todo:: Handle error
			break
		}

		if a.gate.processor != nil {
			msg, err := a.gate.processor.Unmarshal(data)
			if err != nil {
				// todo:: Handle error
				break
			}
			if err = a.gate.processor.Process(msg); err != nil {
				//	todo:: Handle error
				break
			}
		}
	}
}

// Write implements network.Agent.
func (a *agent) Write(msg any) error {
	if a.gate.processor == nil {
		return ProcessorErr
	}

	data, err := a.gate.processor.Marshal(msg)
	if err != nil {
		return err
	}
	return a.conn.Write(data...)
}

// WriteWithCode implements network.Agent.
func (a *agent) WriteWithCode(code uint16, msg any) error {
	if a.gate.processor == nil {
		return ProcessorErr
	}

	data, err := a.gate.processor.MarshalWithCode(code, msg)
	if err != nil {
		return err
	}

	return a.conn.Write(data...)
}

// Close implements Agent.
func (a *agent) Close() {
	a.conn.Close()
}

// Destroy implements network.Agent.
func (a *agent) Destroy() {
	a.conn.Destroy()
}

// LocalAddr implements Agent.
func (a *agent) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}

// RemoteAddr implements Agent.
func (a *agent) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}
