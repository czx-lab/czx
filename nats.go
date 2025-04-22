package czx

import (
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

type (
	NatsConf struct {
		// NATS server hosts (comma-separated or slice)
		Hosts []string
		// Optional reconnect options
		MaxReconnects int
		ReconnectWait int
	}
	Nats struct {
		conf *NatsConf
		conn *nats.Conn
	}
)

// GetConnection returns the existing NATS connection
func (n *Nats) GetConnection() *nats.Conn {
	return n.conn
}

// NewNats creates a new Nats instance and connects to the servers
func NewNats(conf *NatsConf) (*Nats, error) {
	if conf == nil {
		conf = new(NatsConf)
	}

	opts := []nats.Option{
		nats.MaxReconnects(conf.MaxReconnects),
		nats.ReconnectWait(time.Duration(conf.ReconnectWait) * time.Second),
	}

	// Connect to multiple NATS servers
	nc, err := nats.Connect(strings.Join(conf.Hosts, ","), opts...)
	if err != nil {
		return nil, err
	}

	return &Nats{conf: conf, conn: nc}, nil
}

// Publish sends a message to the specified subject
func (n *Nats) Publish(subject string, data []byte) error {
	return n.conn.Publish(subject, data)
}

// PublishNatsMsg sends a NATS message to the specified subject
func (n *Nats) PublishMsg(message *nats.Msg) error {
	return n.conn.PublishMsg(message)
}

// Request sends a request to the specified subject and waits for a response
// with a default timeout of 2 second
// Note: The default timeout is set to 2 second in the NATS library
func (n *Nats) Request(subj string, data []byte) (*nats.Msg, error) {
	return n.conn.Request(subj, data, nats.DefaultTimeout)
}

// Subscribe listens for messages on the specified subject
func (n *Nats) Subscribe(subject string, handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	return n.conn.Subscribe(subject, handler)
}

// QueueSubscribe listens for messages on the specified subject with a queue group
func (n *Nats) QueueSubscribe(subject, queue string, handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	return n.conn.QueueSubscribe(subject, queue, handler)
}

// IsConnected checks if the NATS connection is active
func (n *Nats) IsConnected() bool {
	return n.conn.IsConnected()
}

// Close closes the NATS connection
func (n *Nats) Close() {
	if n.conn == nil {
		return
	}

	n.conn.Close()
}
