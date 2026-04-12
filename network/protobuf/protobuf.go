package protobuf

import (
	"errors"
	"fmt"
	"math"
	"reflect"

	"github.com/czx-lab/czx/network"

	"google.golang.org/protobuf/proto"
)

type (
	// message represents a protobuf message with its ID, type, and handler.
	// It also contains a raw handler for processing raw data.
	message struct {
		id      uint
		msgtype reflect.Type
		handler network.Handler
	}
	// Processor is a protobuf message processor that handles marshalling,
	// unmarshalling, and processing of protobuf messages.
	Processor struct {
		ids      map[reflect.Type]uint
		messages map[uint]*message
		option   network.ProcessorConf
	}
)

func NewProcessor(opt network.ProcessorConf) *Processor {
	return &Processor{
		ids:      make(map[reflect.Type]uint),
		messages: make(map[uint]*message),
		option:   opt,
	}
}

// Marshal implements network.Processor.
func (p *Processor) Marshal(msg any) ([][]byte, error) {
	msgtype := reflect.TypeOf(msg)
	id, ok := p.ids[msgtype]
	if !ok {
		return nil, fmt.Errorf("protobuf: message %v not registered", msgtype)
	}

	msgid := make([]byte, p.option.IDLength)

	network.PutID(msgid, id, p.option)

	data, err := proto.Marshal(msg.(proto.Message))
	return [][]byte{msgid, data}, err
}

// MarshalWithCode implements network.Processor.
func (p *Processor) MarshalWithCode(code uint, msg any) ([][]byte, error) {
	msgs, err := p.Marshal(msg)
	if err != nil {
		return nil, err
	}

	mcode := make([]byte, p.option.CodeLength)
	network.PutCode(mcode, code, p.option)

	smsgs := [][]byte{mcode}
	smsgs = append(smsgs, msgs...)
	return smsgs, nil
}

// Process implements network.Processor.
func (p *Processor) Process(data any, agent network.Agent) error {
	msgtype := reflect.TypeOf(data)
	id, ok := p.ids[msgtype]
	if !ok {
		return fmt.Errorf("message %s not registered", msgtype)
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

// Unmarshal implements network.Processor.
func (p *Processor) Unmarshal(data []byte) (any, error) {
	id, err := network.GetID(data, p.option)
	if err != nil {
		return nil, err
	}

	info, ok := p.messages[id]
	if !ok {
		return nil, fmt.Errorf("protobuf: message ID %d not registered", id)
	}

	msg := reflect.New(info.msgtype.Elem()).Interface()
	return msg, proto.Unmarshal(data[p.option.IDLength:], msg.(proto.Message))
}

// Register implements network.Processor.
func (p *Processor) Register(msg network.Message) error {
	msgtype := reflect.TypeOf(msg.Data)
	if msgtype == nil || msgtype.Kind() != reflect.Pointer {
		return errors.New("protobuf: message must be a pointer")
	}
	if _, ok := p.ids[msgtype]; ok {
		return fmt.Errorf("protobuf: message %v is already registered", msgtype)
	}
	if len(p.messages) >= math.MaxInt {
		return fmt.Errorf("too many protobuf messages (max = %v)", math.MaxInt)
	}

	p.messages[msg.ID] = &message{
		msgtype: msgtype,
		id:      msg.ID,
	}
	p.ids[msgtype] = msg.ID
	return nil
}

// RegisterHandler implements network.Processor.
func (p *Processor) RegisterHandler(msg any, handler network.Handler) error {
	msgtype := reflect.TypeOf(msg)
	id, ok := p.ids[msgtype]
	if !ok {
		return fmt.Errorf("protobuf: message %s not registered", msgtype)
	}

	p.messages[id].handler = handler
	return nil
}

// Range implements network.Processor.
func (p *Processor) Range(fn func(id uint, msgtype reflect.Type)) {
	for _, i := range p.messages {
		fn(i.id, i.msgtype)
	}
}

var _ network.Processor = (*Processor)(nil)
