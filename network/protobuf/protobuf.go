package protobuf

import (
	"czx/network"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"

	"google.golang.org/protobuf/proto"
)

type (
	message struct {
		id         uint16
		msgtype    reflect.Type
		handler    network.Handler
		rawHandler network.Handler
	}
	raw struct {
		id   uint16
		data []byte
	}
	Processor struct {
		ids      map[reflect.Type]uint16
		messages map[uint16]*message
		option   network.ProcessorConf
	}
)

func NewProcessor() *Processor {
	return &Processor{
		ids:      make(map[reflect.Type]uint16),
		messages: make(map[uint16]*message),
	}
}

// Marshal implements network.Processor.
func (p *Processor) Marshal(msg any) ([][]byte, error) {
	msgtype := reflect.TypeOf(msg)
	id, ok := p.ids[msgtype]
	if !ok {
		return nil, fmt.Errorf("protobuf: message %v not registered", msgtype)
	}

	bid := make([]byte, 2)
	if p.option.LittleEndian {
		binary.LittleEndian.PutUint16(bid, id)
	} else {
		binary.BigEndian.PutUint16(bid, id)
	}

	data, err := proto.Marshal(msg.(proto.Message))
	return [][]byte{bid, data}, err
}

// Process implements network.Processor.
func (p *Processor) Process(data any) error {
	if raw, ok := data.(raw); ok {
		info, ok := p.messages[raw.id]
		if !ok {
			return fmt.Errorf("message id %v not registered", raw.id)
		}
		if info.rawHandler != nil {
			info.rawHandler([]any{raw.id, raw.data})
		}
		return nil
	}

	msgtype := reflect.TypeOf(data)
	id, ok := p.ids[msgtype]
	if !ok {
		return fmt.Errorf("message %s not registered", msgtype)
	}

	info := p.messages[id]
	if info.handler != nil {
		info.handler([]any{data})
	}

	return nil
}

// Unmarshal implements network.Processor.
func (p *Processor) Unmarshal(data []byte) (any, error) {
	if len(data) < 2 {
		return nil, errors.New("protobuf data too short")
	}

	var id uint16
	if p.option.LittleEndian {
		id = binary.LittleEndian.Uint16(data)
	} else {
		id = binary.BigEndian.Uint16(data)
	}

	info, ok := p.messages[id]
	if !ok {
		return nil, fmt.Errorf("protobuf: message ID %d not registered", id)
	}
	if info.rawHandler != nil {
		return raw{id, data[2:]}, nil
	}

	msg := reflect.New(info.msgtype.Elem()).Interface()
	return msg, proto.Unmarshal(data[2:], msg.(proto.Message))
}

// Register implements network.Processor.
func (p *Processor) Register(id uint16, msg proto.Message) error {
	msgtype := reflect.TypeOf(msg)
	if msgtype == nil || msgtype.Kind() != reflect.Ptr {
		log.Fatal("protobuf: message must be a pointer")
	}

	_, ok := p.ids[msgtype]
	if ok {
		return fmt.Errorf("protobuf: message %v is already registered", msgtype)
	}
	if len(p.messages) >= math.MaxUint16 {
		return fmt.Errorf("too many protobuf messages (max = %v)", math.MaxUint16)
	}

	p.messages[id] = &message{
		msgtype: msgtype,
		id:      id,
	}
	p.ids[msgtype] = id
	return nil
}

// RegisterHandler implements network.Processor.
func (p *Processor) RegisterHandler(id uint16, msg proto.Message, handler network.Handler) error {
	msgtype := reflect.TypeOf(msg)
	id, ok := p.ids[msgtype]
	if !ok {
		return fmt.Errorf("protobuf: message %s not registered", msgtype)
	}

	p.messages[id].handler = handler
	return nil
}

// Range implements network.Processor.
func (p *Processor) Range(fn func(id uint16, msgtyoe reflect.Type)) {
	for _, i := range p.messages {
		fn(uint16(i.id), i.msgtype)
	}
}

var _ network.Processor = (*Processor)(nil)
