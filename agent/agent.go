package agent

import (
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/czx-lab/czx/eventbus"
	gnetcp "github.com/czx-lab/czx/gnetx/tcp"
	"github.com/czx-lab/czx/network"
	xtcp "github.com/czx-lab/czx/network/tcp"
	"github.com/czx-lab/czx/network/ws"
	"github.com/czx-lab/czx/xlog"

	"go.uber.org/zap"
)

var (
	ErrProcessorNotFound = errors.New("processor not found")
)

type (
	GateConf struct {
		ws.WsServerConf
		xtcp.TcpServerConf
		gnetcp.GnetTcpServerConf
	}
	Gate struct {
		option    GateConf
		processor network.Processor
		wsSrv     *ws.WsServer
		tcpSrv    *xtcp.TcpServer
		gnetcpSrv *gnetcp.GnetTcpServer
		eventBus  *eventbus.EventBus
		preConn   network.PreConnHandler

		flag chan struct{}
	}
	// agent implements network.Agent interface
	// It is used to handle the connection and process messages.
	agent struct {
		conn       network.Conn
		gate       *Gate
		clientAddr network.ClientAddrMessage
		userdata   any
	}
)

var _ network.Agent = (*agent)(nil)
var _ network.GnetAgent = (*agent)(nil)

func NewGate(opt GateConf) *Gate {
	return &Gate{
		option: opt,
	}
}

func (g *Gate) WithFlag(flag chan struct{}) *Gate {
	g.flag = flag
	return g
}

// WithProcessor sets the processor for the Gate instance.
// The processor is responsible for marshaling and unmarshaling messages.
func (g *Gate) WithProcessor(processor network.Processor) *Gate {
	g.processor = processor
	return g
}

// WithPreConn sets the pre-connection function for the Gate instance.
// The pre-connection function is called before a new connection is established.
func (g *Gate) WithPreConn(fn network.PreConnHandler) *Gate {
	g.preConn = fn
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
		g.tcpSrv = xtcp.NewServer(&g.option.TcpServerConf, func(tc *xtcp.TcpConn) network.Agent {
			a := &agent{conn: tc, gate: g}
			if a.gate.eventBus != nil {
				a.gate.eventBus.Publish(eventbus.EvtNewAgent, a)
			}

			return a
		})
		g.tcpSrv.Start()
	}
	if len(g.option.GnetTcpServerConf.Addr) > 0 {
		g.gnetcpSrv = gnetcp.NewGNetTcpServer(&g.option.GnetTcpServerConf, func(c network.Conn) network.Agent {
			a := &agent{conn: c, gate: g}
			if a.gate.eventBus != nil {
				a.gate.eventBus.Publish(eventbus.EvtNewAgent, a)
			}

			return a
		})
		g.gnetcpSrv.Start()
	}

	// Handle graceful shutdown on Ctrl+C
	if g.flag != nil {
		<-g.flag
	} else {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		<-sig
	}

	if g.wsSrv != nil {
		g.wsSrv.Stop()
	}
	if g.tcpSrv != nil {
		g.tcpSrv.Stop()
	}
	if g.gnetcpSrv != nil {
		g.gnetcpSrv.Stop()
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
			xlog.Write().Debug("network read message error", zap.Error(err))
			break
		}

		if a.gate.processor != nil {
			msg, err := a.gate.processor.Unmarshal(data)
			if err != nil {
				xlog.Write().Debug("network processor message decoding error", zap.Error(err))
				break
			}
			if err = a.gate.processor.Process(msg, a); err != nil {
				xlog.Write().Debug("network message processor error", zap.Error(err))
				break
			}
		}
	}
}

// ClientAddr implements network.Agent.
func (a *agent) ClientAddr() network.ClientAddrMessage {
	return a.clientAddr
}

// OnPreConn implements network.Agent.
func (a *agent) OnPreConn(data network.ClientAddrMessage) {
	a.clientAddr = data

	if a.gate.preConn == nil {
		return
	}

	a.gate.preConn(a, data)
}

// React implements network.GnetAgent.
func (a *agent) React(data []byte) {
	if a.gate.processor == nil {
		a.conn.Close()
		return
	}

	msg, err := a.gate.processor.Unmarshal(data)
	if err != nil {
		xlog.Write().Debug("network processor message decoding error", zap.Error(err))
		return
	}
	if err = a.gate.processor.Process(msg, a); err != nil {
		xlog.Write().Debug("network message processor error", zap.Error(err))
		return
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
