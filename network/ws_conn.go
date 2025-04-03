package network

import (
	"errors"
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	// ErrConnClosed is returned when the connection is closed.
	ErrConnClosed   = errors.New("connection closed")
	MessageTooLong  = errors.New("message too long")
	MessageTooShort = errors.New("message too short")
)

type (
	WsConns    map[*websocket.Conn]struct{}
	WsConnConf struct {
		MaxMsgSize      uint32
		pendingWriteNum int
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
		closeFlag bool
	}
)

var _ Conn = (*WsConn)(nil)

func NewWsConn(conn *websocket.Conn, opt *WsConnConf) *WsConn {
	wsConn := &WsConn{
		opt:       opt,
		conn:      conn,
		writeChan: make(chan []byte, opt.pendingWriteNum),
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

// Read implements Conn.
func (w *WsConn) Read() ([]byte, error) {
	_, b, err := w.conn.ReadMessage()
	return b, err
}

// Write implements Conn.
func (w *WsConn) Write(args ...[]byte) error {
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
		return MessageTooLong
	}

	if msgLen < 1 {
		return MessageTooShort
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
