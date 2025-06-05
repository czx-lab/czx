package xnats

import (
	"strings"

	"github.com/nats-io/nats.go"
)

type (
	NatsConf struct {
		// NATS server hosts (comma-separated or slice)
		Hosts []string
	}
	XNats struct {
		conf NatsConf
		conn *nats.Conn
	}
)

func NewNats(conf NatsConf, opts ...nats.Option) (*XNats, error) {
	// Connect to multiple NATS servers
	nc, err := nats.Connect(strings.Join(conf.Hosts, ","), opts...)
	if err != nil {
		return nil, err
	}

	return &XNats{conf: conf, conn: nc}, nil
}

// GetConnection returns the existing NATS connection
// This method allows access to the underlying NATS connection
func (n *XNats) GetConnection() *nats.Conn {
	return n.conn
}

// Publish sends a message to the specified subject
func (n *XNats) Publish(subject string, data []byte) error {
	return n.conn.Publish(subject, data)
}

// PublishNatsMsg sends a NATS message to the specified subject
func (n *XNats) PublishMsg(message *nats.Msg) error {
	return n.conn.PublishMsg(message)
}

// Request sends a request to the specified subject and waits for a response
// with a default timeout of 2 second
// Note: The default timeout is set to 2 second in the NATS library
func (n *XNats) Request(subj string, data []byte) (*nats.Msg, error) {
	return n.conn.Request(subj, data, nats.DefaultTimeout)
}

// Subscribe listens for messages on the specified subject
func (n *XNats) Subscribe(subject string, handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	return n.conn.Subscribe(subject, handler)
}

// QueueSubscribe listens for messages on the specified subject with a queue group
func (n *XNats) QueueSubscribe(subject, queue string, handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	return n.conn.QueueSubscribe(subject, queue, handler)
}

// IsConnected checks if the NATS connection is active
func (n *XNats) IsConnected() bool {
	return n.conn.IsConnected()
}

// Close closes the NATS connection
func (n *XNats) Close() {
	if n.conn == nil {
		return
	}
	n.conn.Close()
}
