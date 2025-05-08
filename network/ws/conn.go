package ws

import (
	"errors"
	"net"
	"sync"

	"github.com/czx-lab/czx/network"
	"github.com/czx-lab/czx/xlog"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var (
	// ErrConnClosed is returned when the connection is closed.
	ErrConnClosed      = errors.New("connection closed")
	ErrMessageTooLong  = errors.New("message too long")
	ErrMessageTooShort = errors.New("message too short")
)

type (
	WsConns    map[*websocket.Conn]struct{}
	WsConnConf struct {
		MaxMsgSize      uint32
		PendingWriteNum int
	}

	// WsConn represents a WebSocket connection with a mutex for thread-safe access.
	// It wraps the gorilla/websocket.Conn type to provide additional functionality.
	WsConn struct {
		mu   sync.Mutex
		opt  *WsConnConf
		conn *websocket.Conn
		// Channel for writing messages to the connection
		writeChan chan []byte
		// Flag to indicate if the connection is closed
		closeFlag  bool
		clientAddr network.ClientAddrMessage // Client address message
	}
)

var _ network.Conn = (*WsConn)(nil)

func NewConn(conn *websocket.Conn, opt *WsConnConf) *WsConn {
	wsConn := &WsConn{
		opt:       opt,
		conn:      conn,
		writeChan: make(chan []byte, opt.PendingWriteNum),
	}

	go func() {
		defer func() {
			conn.Close()

			wsConn.mu.Lock()
			wsConn.closeFlag = true
			wsConn.mu.Unlock()
		}()

		for v := range wsConn.writeChan {
			if v == nil {
				break
			}

			if err := wsConn.conn.WriteMessage(websocket.BinaryMessage, v); err != nil {
				xlog.Write().Error("ws conn write error", zap.Error(err))
				break
			}
		}
	}()

	return wsConn
}

// Close implements Conn.
func (w *WsConn) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closeFlag {
		return
	}

	w.doWrite(nil)
	w.closeFlag = true
}

func (w *WsConn) doWrite(b []byte) {
	if len(w.writeChan) == cap(w.writeChan) {
		// Channel is full, cannot write more messages
		w.doDestroy()
		return
	}

	w.writeChan <- b
}

// Destroy implements Conn.
func (w *WsConn) Destroy() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.doDestroy()
}

// This method is responsible for closing the WebSocket connection and cleaning up resources.
func (w *WsConn) doDestroy() {
	if w.closeFlag {
		return
	}

	// Close the connection and clean up resources
	w.conn.UnderlyingConn().(*net.TCPConn).SetLinger(0)
	w.conn.Close()

	close(w.writeChan)
	w.closeFlag = true
}

// LocalAddr implements Conn.
func (w *WsConn) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

// RemoteAddr implements Conn.
func (w *WsConn) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

// ClientAddr implements network.Conn.
func (w *WsConn) ClientAddr() network.ClientAddrMessage {
	return w.clientAddr
}

// withClientAddr sets the client address message for the GnetConn instance
// This allows the user to specify the client address information associated with the connection
func (w *WsConn) withClientAddr(msg network.ClientAddrMessage) {
	w.clientAddr = msg
}

// ReadMessage implements Conn.
func (w *WsConn) ReadMessage() ([]byte, error) {
	_, b, err := w.conn.ReadMessage()
	return b, err
}

// WriteMessage implements Conn.
func (w *WsConn) WriteMessage(args ...[]byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closeFlag {
		return ErrConnClosed
	}

	var msgLen uint32
	for i := range args {
		msgLen += uint32(len(args[i]))
	}

	if msgLen > w.opt.MaxMsgSize {
		return ErrMessageTooLong
	}

	if msgLen < 1 {
		return ErrMessageTooShort
	}

	if len(args) == 1 {
		w.doWrite(args[0])
		return nil
	}

	msg := make([]byte, msgLen)
	l := 0
	for i := range args {
		copy(msg[l:], args[i])
		l += len(args[i])
	}

	w.doWrite(msg)

	return nil
}
