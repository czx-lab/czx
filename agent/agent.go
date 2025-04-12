package agent

import (
	"czx/eventbus"
	"czx/network"
	"czx/network/tcp"
	"czx/network/ws"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var (
	ErrProcessorNotFound = errors.New("processor not found")
)

type (
	GateConf struct {
		ws.WsServerConf
		tcp.TcpServerConf
	}
	Gate struct {
		option    GateConf
		processor network.Processor
		wsSrv     *ws.WsServer
		tcpSrv    *tcp.TcpServer
		eventBus  *eventbus.EventBus
	}
	// agent implements network.Agent interface
	// It is used to handle the connection and process messages.
	agent struct {
		conn     network.Conn
		gate     *Gate
		userdata any
	}
)

var _ network.Agent = (*agent)(nil)

func NewGate(opt GateConf) *Gate {
	return &Gate{
		option: opt,
	}
}

// WithProcessor sets the processor for the Gate instance.
// The processor is responsible for marshaling and unmarshaling messages.
func (g *Gate) WithProcessor(processor network.Processor) *Gate {
	g.processor = processor
	return g
}

// WithEventBus sets the event bus for the Gate instance.
// The event bus is used for publishing and subscribing to events.
func (g *Gate) WithEventBus(bus *eventbus.EventBus) *Gate {
	g.eventBus = bus
	return g
}

func (g *Gate) Start() {
	// Default event bus
	// If no event bus is provided, use the default event bus.
	if g.eventBus == nil {
		g.eventBus = eventbus.DefaultBus
	}

	if len(g.option.WsServerConf.Addr) > 0 {
		g.wsSrv = ws.NewServer(&g.option.WsServerConf, func(wc *ws.WsConn) network.Agent {
			a := &agent{conn: wc, gate: g}
			if a.gate.eventBus != nil {
				a.gate.eventBus.Publish(eventbus.EvtNewAgent, a)
			}

			return a
		})
		g.wsSrv.Start()
	}
	if len(g.option.TcpServerConf.Addr) > 0 {
		g.tcpSrv = tcp.NewServer(&g.option.TcpServerConf, func(tc *tcp.TcpConn) network.Agent {
			a := &agent{conn: tc, gate: g}
			if a.gate.eventBus != nil {
				a.gate.eventBus.Publish(eventbus.EvtNewAgent, a)
			}

			return a
		})
		g.tcpSrv.Start()
	}

	// Handle graceful shutdown on Ctrl+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-sig
	if g.wsSrv != nil {
		g.wsSrv.Stop()
	}
	if g.tcpSrv != nil {
		g.tcpSrv.Stop()
	}
}

// OnClose implements network.Agent.
func (a *agent) OnClose() {
	if a.gate.eventBus == nil {
		return
	}

	a.gate.eventBus.Publish(eventbus.EvtAgentClose, a)
}

func (a *agent) Run() {
	for {
		data, err := a.conn.ReadMessage()
		if err != nil {
			// Log or handle the error
			break
		}

		if a.gate.processor != nil {
			msg, err := a.gate.processor.Unmarshal(data)
			if err != nil {
				// Log or handle the error
				break
			}
			if err = a.gate.processor.Process(msg); err != nil {
				// Log or handle the error
				break
			}
		}
	}
}

// Write implements network.Agent.
func (a *agent) Write(msg any) error {
	if a.gate.processor == nil {
		return ErrProcessorNotFound
	}

	data, err := a.gate.processor.Marshal(msg)
	if err != nil {
		return err
	}
	return a.conn.WriteMessage(data...)
}

// WriteWithCode implements network.Agent.
func (a *agent) WriteWithCode(code uint16, msg any) error {
	if a.gate.processor == nil {
		return ErrProcessorNotFound
	}

	data, err := a.gate.processor.MarshalWithCode(code, msg)
	if err != nil {
		return err
	}

	return a.conn.WriteMessage(data...)
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

// GetUserData implements network.Agent.
func (a *agent) GetUserData() any {
	return a.userdata
}

// SetUserData implements network.Agent.
func (a *agent) SetUserData(data any) {
	a.userdata = data
}
