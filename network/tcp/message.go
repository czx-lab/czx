package tcp

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

var (
	defaultMsgMinSize uint32 = 1
	defaultMsgMaxSize uint32 = 4096

	ErrMessageTooLong  = errors.New("message too long")
	ErrMessageTooShort = errors.New("message too short")
)

const (
	LenType8  LenType = iota + 1 // 1 bytes
	LenType16                    // 2 bytes
	LenType32 LenType = iota + 2 // 4 bytes
)

type (
	// Length type for message length field
	LenType           uint
	MessageParserConf struct {
		// Type of the message length field (0: uint8, 1: uint16, 2: uint32)
		MsgLengthType LenType
		// Minimum message size
		MsgMinSize uint32
		// Maximum message size (0: no limit)
		MsgMaxSize   uint32
		LittleEndian bool
	}
	MessageParser struct {
		conf *MessageParserConf
	}
)

func NewParse(conf *MessageParserConf) *MessageParser {
	defaultParseConf(conf)

	return &MessageParser{
		conf: conf,
	}
}

// Read message from connection, the first 1/2/4 bytes is the length of the message
func (m *MessageParser) Read(conn *TcpConn) ([]byte, error) {
	var b [4]byte
	bufMsgLen := b[:m.conf.MsgLengthType]
	if _, err := io.ReadFull(conn, bufMsgLen); err != nil {
		return nil, err
	}

	var msgLen uint32
	switch m.conf.MsgLengthType {
	case LenType8:
		msgLen = uint32(bufMsgLen[0])
	case LenType16:
		if m.conf.LittleEndian {
			msgLen = uint32(binary.LittleEndian.Uint16(bufMsgLen))
		} else {
			msgLen = uint32(binary.BigEndian.Uint16(bufMsgLen))
		}
	case LenType32:
		if m.conf.LittleEndian {
			msgLen = binary.LittleEndian.Uint32(bufMsgLen)
		} else {
			msgLen = binary.BigEndian.Uint32(bufMsgLen)
		}
	}

	if msgLen > m.conf.MsgMaxSize {
		return nil, ErrMessageTooLong
	}
	if msgLen < m.conf.MsgMinSize {
		return nil, ErrMessageTooShort
	}

	data := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, err
	}

	return data, nil
}

// Write Message
func (m *MessageParser) Write(conn *TcpConn, args ...[]byte) error {
	var msgLen uint32
	for i := range args {
		msgLen += uint32(len(args[i]))
	}
	if msgLen > m.conf.MsgMaxSize {
		return ErrMessageTooLong
	}
	if msgLen < m.conf.MsgMinSize {
		return ErrMessageTooShort
	}

	msg := make([]byte, uint32(m.conf.MsgLengthType)+msgLen)
	switch m.conf.MsgLengthType {
	case LenType8:
		msg[0] = byte(msgLen)
	case LenType16:
		if m.conf.LittleEndian {
			binary.LittleEndian.PutUint16(msg, uint16(msgLen))
		} else {
			binary.BigEndian.PutUint16(msg, uint16(msgLen))
		}
	case LenType32:
		if m.conf.LittleEndian {
			binary.LittleEndian.PutUint32(msg, msgLen)
		} else {
			binary.BigEndian.PutUint32(msg, msgLen)
		}
	}

	l := int(m.conf.MsgLengthType)
	for i := range args {
		copy(msg[l:], args[i])
		l += len(args[i])
	}

	conn.Write(msg)

	return nil
}

func defaultParseConf(conf *MessageParserConf) {
	if conf.MsgMaxSize <= 0 {
		conf.MsgMaxSize = defaultMsgMaxSize
	}
	if conf.MsgMinSize <= 0 {
		conf.MsgMinSize = defaultMsgMinSize
	}

	var max uint32
	switch conf.MsgLengthType {
	case LenType8:
		max = math.MaxUint8
	case LenType16:
		max = math.MaxUint16
	case LenType32:
		max = math.MaxUint32
	}
	if conf.MsgMinSize > max {
		conf.MsgMinSize = max
	}
	if conf.MsgMaxSize > max {
		conf.MsgMaxSize = max
	}
}
