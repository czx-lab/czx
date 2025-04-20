package jsonx

import (
	"czx/network"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
)

type (
	Processor struct {
		conf network.ProcessorConf
		// message name registered by id
		// ids is used for json messages that are registered by id
		ids map[string]uint16
		// messages registered by id
		messages map[uint16]*message
		//	used for json messages that are not registered by id
		messagesByName map[string]*message
	}
	message struct {
		id         uint16
		name       string
		msgtype    reflect.Type
		handler    network.Handler
		rawHandler network.Handler
	}
	Raw struct {
		id   uint16
		name string
		data json.RawMessage
	}
)

var _ network.JsonProcessor = (*Processor)(nil)

// NewProcessor creates a new json processor.
// It is used for json messages that are registered by id.
func NewProcessor(conf network.ProcessorConf) *Processor {
	return &Processor{
		conf:           conf,
		messages:       make(map[uint16]*message),
		messagesByName: make(map[string]*message),
		ids:            make(map[string]uint16),
	}
}

// Marshal implements network.Processor.
func (p *Processor) Marshal(msgs any) ([][]byte, error) {
	msgtype := reflect.TypeOf(msgs)
	if msgtype == nil || msgtype.Kind() != reflect.Ptr {
		return nil, errors.New("json message pointer required")
	}

	msgname := msgtype.Elem().Name()
	if _, ok := p.ids[msgname]; ok {
		goto MarshalExec
	}
	if _, ok := p.messagesByName[msgname]; !ok {
		return nil, fmt.Errorf("message %v not registered", msgname)
	}

MarshalExec:
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

	return [][]byte{msgs[0], msgcode, msgs[1]}, nil
}

// Process implements network.Processor.
func (p *Processor) Process(data any) error {
	if raw, ok := data.(Raw); ok {
		rawinfo, ok := p.messages[raw.id]
		if ok {
			goto RawHandlerExec
		}

		// check if the message is registered by name
		// this is used for json messages that are not registered by id
		rawinfo, ok = p.messagesByName[raw.name]
		if !ok {
			return fmt.Errorf("message %v not registered", raw.name)
		}

	RawHandlerExec:
		if rawinfo.rawHandler != nil {
			rawinfo.rawHandler([]any{raw.id, raw.data})
		}
		return nil
	}

	// check if the message is registered by id
	var info *message
	msgname := reflect.TypeOf(data).Elem().Name()
	id, ok := p.ids[msgname]
	if ok {
		info, ok = p.messages[id]
		if !ok {
			return fmt.Errorf("message %s not registered", msgname)
		}

		goto HandlerExec
	}

	// this is used for json messages that are not registered by id
	info, ok = p.messagesByName[msgname]
	if !ok {
		return fmt.Errorf("message %s not registered", msgname)
	}

HandlerExec:
	if info.handler != nil {
		info.handler([]any{data})
	}

	return nil
}

// Register implements network.Processor.
func (p *Processor) Register(id uint16, msg any) error {
	msgtype := reflect.TypeOf(msg)
	if msgtype == nil || msgtype.Kind() != reflect.Ptr {
		return errors.New("json message pointer required")
	}

	// check if the message is registered
	msgname := msgtype.Elem().Name()
	if len(msgname) == 0 {
		return errors.New("unnamed json message")
	}
	if _, ok := p.messages[id]; ok {
		return fmt.Errorf("message %v is already registered", msgname)
	}

	i := new(message)
	i.id = id
	i.msgtype = msgtype
	p.messages[id] = i
	p.ids[msgname] = id

	return nil
}

// RegisterExceptID implements network.JsonProcessor.
func (p *Processor) RegisterExceptID(msg any) error {
	msgtype := reflect.TypeOf(msg)
	if msgtype == nil || msgtype.Kind() != reflect.Ptr {
		log.Fatal("json message pointer required")
	}

	// check if the message is registered
	msgname := msgtype.Elem().Name()
	if len(msgname) == 0 {
		return errors.New("unnamed json message")
	}
	if _, ok := p.messagesByName[msgname]; ok {
		return fmt.Errorf("message %v is already registered", msgname)
	}

	i := new(message)
	i.name = msgname
	i.msgtype = msgtype
	p.messagesByName[msgname] = i

	return nil
}

// RegisterHandler implements network.Processor.
func (p *Processor) RegisterHandler(msg any, handler network.Handler) error {
	msgtype := reflect.TypeOf(msg)
	if msgtype == nil || msgtype.Kind() != reflect.Ptr {
		return errors.New("json message pointer required")
	}

	var info *message
	msgname := msgtype.Elem().Name()
	id, ok := p.ids[msgname]
	if ok {
		info, ok = p.messages[id]
		if !ok {
			return fmt.Errorf("message %v not registered", id)
		}

		goto WithHandlerExec
	}

	info, ok = p.messagesByName[msgname]
	if !ok {
		return fmt.Errorf("message %v not registered", msgname)
	}

WithHandlerExec:
	info.handler = handler

	return nil
}

// Unmarshal implements network.Processor.
func (p *Processor) Unmarshal(data []byte) (any, error) {
	var m map[any]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if len(m) != 1 {
		return nil, errors.New("invalid json data")
	}

	for msgid, data := range m {
		var info *message
		var mid uint16
		var mname string

		id, ok := msgid.(uint16)
		name, nameok := msgid.(string)

		// check if the message is registered by id
		if ok {
			info, ok = p.messages[id]
			if !ok {
				return nil, fmt.Errorf("message %v not registered", msgid)
			}
			goto UnmarshalExec
		}

		if !nameok {
			return nil, fmt.Errorf("message %v not registered", msgid)
		}

		info, ok = p.messagesByName[name]
		if !ok {
			return nil, fmt.Errorf("message %v not registered", msgid)
		}

	UnmarshalExec:
		if info.rawHandler != nil {
			return Raw{mid, mname, data}, nil
		} else {
			msg := reflect.New(info.msgtype.Elem()).Interface()
			return msg, json.Unmarshal(data, msg)
		}
	}

	return nil, errors.New("invalid json data")
}
