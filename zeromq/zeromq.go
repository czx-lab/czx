// Package zeromq provides a wrapper for ZeroMQ, a messaging library that implements the ZeroMQ protocol,
// It provides functions for sending and receiving messages using ZeroMQ, and supports DEALER, ROUTER, PUB, SUB modes.
package zeromq

import (
	"fmt"
	"time"

	"github.com/czx-lab/czx/xlog"
	zmq "github.com/pebbe/zmq4"
	"go.uber.org/zap"
)

const (
	// Default timeout for ZeroMQ connections, in seconds
	defaultTimeout          = 30
	defaultHeartbeatIvl     = 30
	defaultHeartbeatTimeout = 60
)

type (
	// ZeromqConf is the configuration for the Zeromq instance
	ZeromqConf struct {
		// Address to connect to
		Addr string
		// Type of the ZeroMQ socket
		Type zmq.Type
		// Timeout for ZeroMQ connections, in seconds, default 30
		Timeout int
		// HeartbeatIvl for ZeroMQ connections, in seconds, default 30
		HeartbeatIvl int
		// HeartbeatTimeout for ZeroMQ connections, in seconds, default 60
		HeartbeatTimeout int
		// Identity for ZeroMQ sockets
		Identity string
	}
	// Zeromq is the ZeroMQ instance that provides socket and context
	Zeromq struct {
		conf   ZeromqConf
		socket *zmq.Socket
		done   chan struct{}
	}
)

// NewZeromq creates a new Zeromq instance with the specified configuration
func NewZeromq(conf ZeromqConf) (*Zeromq, error) {
	defaultConf(&conf)

	zq := &Zeromq{
		conf: conf,
		done: make(chan struct{}),
	}
	socket, err := zq.connect()
	if err != nil {
		return nil, err
	}
	zq.socket = socket
	return zq, nil
}

// connect creates a new ZeroMQ socket and binds it to the specified address
func (zq *Zeromq) connect() (socket *zmq.Socket, err error) {
	ctx, err := zmq.NewContext()
	if err != nil {
		return
	}
	socket, err = ctx.NewSocket(zq.conf.Type)
	if err != nil {
		return
	}
	if zq.conf.Type == zmq.DEALER && len(zq.conf.Identity) > 0 {
		if err = socket.SetIdentity(zq.conf.Identity); err != nil {
			return
		}
	}
	if err = socket.SetConnectTimeout(time.Duration(zq.conf.Timeout) * time.Second); err != nil {
		return
	}
	if err = socket.SetHeartbeatIvl(time.Duration(zq.conf.HeartbeatIvl) * time.Second); err != nil {
		return
	}
	if err = socket.SetHeartbeatTimeout(time.Duration(zq.conf.HeartbeatTimeout) * time.Second); err != nil {
		return
	}
	caddr := fmt.Sprintf("tcp://%s", zq.conf.Addr)
	switch zq.conf.Type {
	case zmq.PUB, zmq.REP, zmq.ROUTER, zmq.PUSH:
		err = socket.Bind(caddr)
	case zmq.SUB, zmq.REQ, zmq.DEALER, zmq.PULL:
		err = socket.Connect(caddr)
	}
	return
}

// Socket returns the ZeroMQ socket
func (zq *Zeromq) Socket() *zmq.Socket {
	return zq.socket
}

// Context returns the ZeroMQ context
func (zq *Zeromq) Context() (*zmq.Context, error) {
	return zq.socket.Context()
}

// Close closes the ZeroMQ socket and context if they are not nil
func (zq *Zeromq) Close() {
	if zq.socket == nil {
		return
	}

	zq.Stop()
	if err := zq.socket.Close(); err != nil {
		xlog.Write().Error("zeromq close error", zap.Error(err))
	}
}

// Stop stops the ZeroMQ socket and context if they are not nil
func (zq *Zeromq) Stop() {
	select {
	case <-zq.done:
	default:
		close(zq.done)
	}
}

// Dealer sends a message to the ZeroMQ socket if the socket type is zmq.DEALER
func (zq *Zeromq) Dealer(message []byte) (int, error) {
	if zq.conf.Type != zmq.DEALER {
		return 0, fmt.Errorf("socket type %v does not support Dealer mode", zq.conf.Type)
	}
	return zq.socket.SendMessage([]byte{}, message)
}

// DealerRecv receives a message from the ZeroMQ socket if the socket type is zmq.DEALER
func (zq *Zeromq) DealerRecv(flag zmq.Flag) ([]byte, error) {
	if zq.conf.Type != zmq.DEALER {
		return nil, fmt.Errorf("socket type %v does not support Dealer mode", zq.conf.Type)
	}
	message, err := zq.socket.RecvMessageBytes(flag)
	if err != nil {
		return nil, err
	}
	if len(message) < 2 {
		return nil, fmt.Errorf("invalid message")
	}
	return message[1], nil
}

// Router receives a message from the ZeroMQ socket if the socket type is zmq.ROUTER
func (zq *Zeromq) Router(call func(identity []byte, message []byte) []byte) error {
	if zq.conf.Type != zmq.ROUTER {
		return fmt.Errorf("socket type %v does not support Router mode", zq.conf.Type)
	}
	go func() {
		for {
			select {
			case <-zq.done:
				return
			default:
			}
			message, err := zq.socket.RecvMessageBytes(0)
			if err != nil {
				xlog.Write().Sugar().Debugf("zeromq router error: %v", err)
				continue
			}
			identity := message[0]
			delimiter := message[1]
			body := message[2]
			resp := call(identity, body)
			// send false to client
			zq.socket.SendBytes(identity, zmq.SNDMORE)
			zq.socket.SendBytes(delimiter, zmq.SNDMORE)
			zq.socket.SendBytes(resp, 0)
		}
	}()
	return nil
}

// Pub sends a message to the ZeroMQ socket if the socket type is zmq.PUB
func (zq *Zeromq) Pub(topic string, message []byte) error {
	if zq.conf.Type != zmq.PUB {
		return fmt.Errorf("socket type %v does not support Pub mode", zq.conf.Type)
	}
	_, err := zq.socket.SendBytes([]byte(topic), zmq.SNDMORE)
	if err != nil {
		return err
	}
	_, err = zq.socket.SendBytes(message, 0)
	return err
}

// Sub receives a message from the ZeroMQ socket if the socket type is zmq.SUB
func (zq *Zeromq) Sub(topic string, call func([]byte)) error {
	if zq.conf.Type != zmq.SUB {
		return fmt.Errorf("socket type %v does not support Sub mode", zq.conf.Type)
	}
	if err := zq.socket.SetSubscribe(topic); err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-zq.done:
				return
			default:
			}
			msg, err := zq.socket.RecvMessageBytes(0)
			if err != nil {
				xlog.Write().Sugar().Debugf("zeromq subscribe error: %v", err)
				continue
			}
			if len(msg) < 2 {
				xlog.Write().Sugar().Debugf("zeromq subscribe invalid message")
				continue
			}
			call(msg[1])
		}
	}()
	return nil
}

// SubMulti receives a message from the ZeroMQ socket if the socket type is zmq.SUB
func (zq *Zeromq) SubMulti(topics []string, call func(topic string, message []byte)) error {
	if zq.conf.Type != zmq.SUB {
		return fmt.Errorf("socket type %v does not support Sub mode", zq.conf.Type)
	}
	if call == nil {
		return fmt.Errorf("call is nil")
	}
	if len(topics) == 0 {
		return fmt.Errorf("topics is empty")
	}
	for _, topic := range topics {
		if err := zq.socket.SetSubscribe(topic); err != nil {
			return err
		}
	}
	go func() {
		for {
			select {
			case <-zq.done:
				return
			default:
			}
			msg, err := zq.socket.RecvMessageBytes(0)
			if err != nil {
				xlog.Write().Sugar().Debugf("zeromq subscribe error: %v", err)
				continue
			}
			if len(msg) < 2 {
				xlog.Write().Sugar().Debugf("zeromq subscribe invalid message")
				continue
			}
			call(string(msg[0]), msg[1])
		}
	}()
	return nil
}

func defaultConf(conf *ZeromqConf) {
	if conf.Timeout == 0 {
		conf.Timeout = defaultTimeout
	}
	if conf.HeartbeatIvl == 0 {
		conf.HeartbeatIvl = defaultHeartbeatIvl
	}
	if conf.HeartbeatTimeout == 0 {
		conf.HeartbeatTimeout = defaultHeartbeatTimeout
	}
}
