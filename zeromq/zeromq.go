// Package zeromq provides a wrapper for ZeroMQ, a messaging library that implements the ZeroMQ protocol.
package zeromq

import (
	"fmt"
	"time"

	"github.com/czx-lab/czx/xlog"
	zmq "github.com/pebbe/zmq4"
	"go.uber.org/zap"
)

const (
	// Default timeout for ZeroMQ connections
	defaultTimeout = 10
)

type (
	// ZeromqConf is the configuration for the Zeromq instance
	ZeromqConf struct {
		// Address to connect to
		Addr string
		// Type of the ZeroMQ socket
		Type zmq.Type
		// Timeout for ZeroMQ connections
		Timeout int
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
	if err = socket.SetConnectTimeout(time.Duration(zq.conf.Timeout) * time.Second); err != nil {
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

// Stop subscribes to a topic on the ZeroMQ socket if the socket type is zmq.SUB
func (zq *Zeromq) Stop() {
	select {
	case <-zq.done:
	default:
		close(zq.done)
	}
}

// Publish sends a message to the ZeroMQ socket if the socket type is zmq.PUB
func (zq *Zeromq) Publish(message []byte, flag zmq.Flag) (int, error) {
	if zq.conf.Type != zmq.PUB {
		return -1, fmt.Errorf("not support")
	}
	return zq.socket.SendBytes(message, flag)
}

// Subscribe receives a message from the ZeroMQ socket if the socket type is zmq.SUB
func (zq *Zeromq) Subscribe(prefix string, flag zmq.Flag, call func([]byte)) error {
	if zq.conf.Type != zmq.SUB {
		return fmt.Errorf("not support")
	}
	if call == nil {
		return fmt.Errorf("call is nil")
	}
	if err := zq.socket.SetSubscribe(prefix); err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-zq.done:
				return
			default:
			}
			msg, err := zq.socket.RecvMessageBytes(flag)
			if err != nil {
				xlog.Write().Sugar().Debugf("zeromq subscribe error: %v", err)
				continue
			}
			for _, bval := range msg {
				call(bval)
			}
		}
	}()

	return nil
}

func defaultConf(conf *ZeromqConf) {
	if conf.Timeout == 0 {
		conf.Timeout = defaultTimeout
	}
}
