package jsonx

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/czx-lab/czx/network"
)

type (
	Processor struct {
		conf network.ProcessorConf
		// messages registered by id
		messages map[string]*message
	}
	message struct {
		name       string
		msgtype    reflect.Type
		handler    network.Handler
		rawHandler network.Handler
	}
	Raw struct {
		name string
		data json.RawMessage
	}
)

var _ network.Processor = (*Processor)(nil)

// NewProcessor creates a new json processor.
// It is used for json messages that are registered by id.
func NewProcessor(conf network.ProcessorConf) *Processor {
	return &Processor{
		conf:     conf,
		messages: make(map[string]*message),
	}
}

// Marshal implements network.Processor.
func (p *Processor) Marshal(msgs any) ([][]byte, error) {
	msgtype := reflect.TypeOf(msgs)
	if msgtype == nil || msgtype.Kind() != reflect.Ptr {
		return nil, errors.New("json message pointer required")
	}

	msgname := msgtype.Elem().Name()
	if _, ok := p.messages[msgname]; !ok {
		return nil, fmt.Errorf("message %v not registered", msgname)
	}

	m := map[string]any{msgname: msgs}
	data, err := json.Marshal(m)
	return [][]byte{data}, err
}

// MarshalWithCode implements network.Processor.
func (p *Processor) MarshalWithCode(code uint16, msg any) ([][]byte, error) {
	msgs, err := p.Marshal(msg)
	if err != nil {
		return nil, err
	}

	msgcode := make([]byte, 2)
	if p.conf.LittleEndian {
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
	if raw, ok := data.(Raw); ok {
		rawinfo, ok := p.messages[raw.name]
		if !ok {
			return fmt.Errorf("message %v not registered", raw.name)
		}
		if rawinfo.rawHandler != nil {
			rawinfo.rawHandler([]any{raw.name, raw.data, agent})
		}

		return nil
	}

	msgname := reflect.TypeOf(data).Elem().Name()
	info, ok := p.messages[msgname]
	if !ok {
		return fmt.Errorf("message %s not registered", msgname)
	}
	if info.handler != nil {
		info.handler([]any{data, agent})
	}

	return nil
}

// Register implements network.Processor.
func (p *Processor) Register(msg network.Message) error {
	msgtype := reflect.TypeOf(msg.Data)
	if msgtype == nil || msgtype.Kind() != reflect.Ptr {
		return errors.New("json message pointer required")
	}

	// check if the message is registered
	msgname := msgtype.Elem().Name()
	if len(msgname) == 0 {
		return errors.New("unnamed json message")
	}
	if _, ok := p.messages[msgname]; ok {
		return fmt.Errorf("message %v is already registered", msgname)
	}

	i := new(message)
	i.name = msgname
	i.msgtype = msgtype
	p.messages[msgname] = i

	return nil
}

// RegisterHandler implements network.Processor.
func (p *Processor) RegisterHandler(msg any, handler network.Handler) error {
	msgtype := reflect.TypeOf(msg)
	if msgtype == nil || msgtype.Kind() != reflect.Ptr {
		return errors.New("json message pointer required")
	}

	msgname := msgtype.Elem().Name()
	info, ok := p.messages[msgname]
	if !ok {
		return fmt.Errorf("message %v not registered", msgname)
	}

	info.handler = handler
	return nil
}

// RegisterRawHandler implements network.JsonProcessor.
func (p *Processor) RegisterRawHandler(msg any, handler network.Handler) error {
	msgtype := reflect.TypeOf(msg)
	if msgtype == nil || msgtype.Kind() != reflect.Ptr {
		return errors.New("json message pointer required")
	}

	msgname := msgtype.Elem().Name()
	info, ok := p.messages[msgname]
	if !ok {
		return fmt.Errorf("message %v not registered", msgname)
	}

	info.rawHandler = handler
	return nil
}

// Unmarshal implements network.Processor.
func (p *Processor) Unmarshal(data []byte) (any, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if len(m) != 1 {
		return nil, errors.New("invalid json data")
	}

	for msgname, data := range m {
		info, ok := p.messages[msgname]
		if !ok {
			return nil, fmt.Errorf("message %v not registered", msgname)
		}

		if info.rawHandler != nil {
			return Raw{msgname, data}, nil
		} else {
			msg := reflect.New(info.msgtype.Elem()).Interface()
			return msg, json.Unmarshal(data, msg)
		}
	}

	return nil, errors.New("invalid json data")
}
