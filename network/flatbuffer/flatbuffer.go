package flatbuffer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"

	"github.com/czx-lab/czx/network"
	fb "github.com/google/flatbuffers/go"
)

type (
	message_t struct {
		id      uint16
		type_   reflect.Type
		handler network.Handler
	}
	Processor struct {
		ids      map[reflect.Type]uint16
		messages map[uint16]*message_t
		option   network.ProcessorConf
	}
)

func NewProcessor(opt network.ProcessorConf) *Processor {
	return &Processor{
		ids:      make(map[reflect.Type]uint16),
		messages: make(map[uint16]*message_t),
		option:   opt,
	}
}

// Marshal implements network.Processor.
func (p *Processor) Marshal(msgs any) ([][]byte, error) {
	type_t := reflect.TypeOf(msgs)
	id, ok := p.ids[type_t]
	if !ok {
		return nil, fmt.Errorf("flatbuffers: message %v not registered", type_t)
	}

	msgid := make([]byte, 2)
	if p.option.LittleEndian {
		binary.LittleEndian.PutUint16(msgid, id)
	} else {
		binary.BigEndian.PutUint16(msgid, id)
	}
	fn, ok := msgs.(interface{ Bytes() []byte })
	if ok {
		data := fn.Bytes()
		return [][]byte{msgid, data}, nil
	}
	return nil, errors.New("flatbuffers: message does not implement Bytes() []byte")
}

// MarshalWithCode implements network.Processor.
func (p *Processor) MarshalWithCode(code uint16, msg any) ([][]byte, error) {
	msgs, err := p.Marshal(msg)
	if err != nil {
		return nil, err
	}

	msgcode := make([]byte, 2)
	if p.option.LittleEndian {
		binary.LittleEndian.PutUint16(msgcode, code)
	} else {
		binary.BigEndian.PutUint16(msgcode, code)
	}

	smsgs := [][]byte{msgcode}
	smsgs = append(smsgs, msgs...)
	return smsgs, nil
}

// Process implements network.Processor.
func (p *Processor) Process(data any, agent network.Agent) error {
	type_t := reflect.TypeOf(data)
	id, ok := p.ids[type_t]
	if !ok {
		return fmt.Errorf("message %s not registered", type_t)
	}

	info, ok := p.messages[id]
	if !ok {
		return fmt.Errorf("message id %v not registered", id)
	}
	if info.handler != nil {
		info.handler([]any{data, agent})
	}

	return nil
}

// Register implements network.Processor.
func (p *Processor) Register(msg network.Message) error {
	type_t := reflect.TypeOf(msg.Data)
	if type_t == nil || type_t.Kind() != reflect.Pointer {
		return errors.New("flatbuffers: message must be a pointer")
	}
	if _, ok := p.ids[type_t]; ok {
		return fmt.Errorf("flatbuffers: message %v is already registered", type_t)
	}
	if len(p.messages) >= math.MaxUint16 {
		return fmt.Errorf("too many flatbuffers messages (max = %v)", math.MaxUint16)
	}

	p.messages[msg.ID] = &message_t{
		type_: type_t,
		id:    msg.ID,
	}
	p.ids[type_t] = msg.ID
	return nil
}

// RegisterHandler implements network.Processor.
func (p *Processor) RegisterHandler(msg any, handler network.Handler) error {
	type_t := reflect.TypeOf(msg)
	id, ok := p.ids[type_t]
	if !ok {
		return fmt.Errorf("flatbuffers: message %s not registered", type_t)
	}

	p.messages[id].handler = handler
	return nil
}

// Unmarshal implements network.Processor.
func (p *Processor) Unmarshal(data []byte) (any, error) {
	if len(data) < 2 {
		return nil, errors.New("flatbuffers data too short")
	}

	var id uint16
	if p.option.LittleEndian {
		id = binary.LittleEndian.Uint16(data)
	} else {
		id = binary.BigEndian.Uint16(data)
	}

	info, ok := p.messages[id]
	if !ok {
		return nil, fmt.Errorf("flatbuffers: message ID %d not registered", id)
	}
	msg := reflect.New(info.type_.Elem()).Interface()
	method := reflect.ValueOf(msg).MethodByName("Init")
	if !method.IsValid() {
		return nil, fmt.Errorf("flatbuffers: message %s does not have Init method", info.type_)
	}

	buf := data[2:]
	pos := fb.GetUOffsetT(buf)
	_ = method.Call([]reflect.Value{reflect.ValueOf(buf), reflect.ValueOf(pos)})
	return msg, nil
}

var _ network.Processor = (*Processor)(nil)
