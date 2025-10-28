package xnats

import (
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
)

type JetStream struct {
	js nats.JetStreamContext
}

func NewJetStream(nc *nats.Conn, opts ...nats.JSOpt) (*JetStream, error) {
	js, err := nc.JetStream(opts...)
	if err != nil {
		return nil, err
	}

	return &JetStream{js: js}, nil
}

// GetJetStreamContext returns the underlying JetStream context.
// This method allows access to the underlying JetStream context for advanced operations.
func (js *JetStream) GetJetStreamContext() nats.JetStreamContext {
	return js.js
}

// AddStream creates a new JetStream stream with the given configuration.
// If the stream already exists, it returns nil without error.
func (js *JetStream) AddStream(conf *nats.StreamConfig, opts ...nats.JSOpt) error {
	_, err := js.js.StreamInfo(conf.Name)
	if err == nil {
		return nil // Stream already exists
	}
	if err != nats.ErrStreamNotFound {
		return err // Some other error occurred
	}

	// Add the stream
	_, err = js.js.AddStream(conf, opts...)

	return err
}

// AddConsumer creates a new JetStream consumer for the specified stream with the given configuration.
// If the consumer already exists, it returns nil without error.
func (js *JetStream) AddConsumer(stream, consumer string, cconf *nats.ConsumerConfig, opts ...nats.JSOpt) error {
	if cconf == nil {
		return fmt.Errorf("consumer config is nil")
	}
	// If consumer name is empty, create a new consumer with a generated name
	if consumer == "" {
		_, err := js.js.AddConsumer(stream, cconf, opts...)
		return err
	}
	// Check if the consumer already exists
	_, err := js.js.ConsumerInfo(stream, consumer)
	if err == nil {
		return nil // Consumer already exists
	}
	if errors.Is(err, nats.ErrConsumerNotFound) {
		return err // Some other error occurred
	}

	// Add the consumer
	_, err = js.js.AddConsumer(stream, cconf, opts...)

	return err
}

// Publish sends a message to the specified subject in JetStream.
// It returns a PubAck on success or an error if the publish fails.
func (js *JetStream) Publish(subject string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error) {
	return js.js.Publish(subject, data, opts...)
}

// PublishMsg sends a NATS message to the specified subject in JetStream.
// It returns a PubAck on success or an error if the publish fails.
func (js *JetStream) PublishMsg(msg *nats.Msg, opts ...nats.PubOpt) (*nats.PubAck, error) {
	return js.js.PublishMsg(msg, opts...)
}

// PublishAsync sends a message to the specified subject in JetStream asynchronously.
// It returns a PubAckFuture that can be used to wait for the publish acknowledgment.
func (js *JetStream) PublishAsync(subject string, data []byte, opts ...nats.PubOpt) (nats.PubAckFuture, error) {
	return js.js.PublishAsync(subject, data, opts...)
}

// PublishMsgAsync sends a NATS message to the specified subject in JetStream asynchronously.
func (js *JetStream) PublishMsgAsync(msg *nats.Msg, opts ...nats.PubOpt) (nats.PubAckFuture, error) {
	return js.js.PublishMsgAsync(msg, opts...)
}

// Subscribe subscribes to messages on the specified subject in JetStream.
// It returns a Subscription on success or an error if the subscription fails.
func (js *JetStream) Subscribe(subject string, handler nats.MsgHandler, opts ...nats.SubOpt) (*nats.Subscription, error) {
	return js.js.Subscribe(subject, handler, opts...)
}

// QueueSubscribe subscribes to messages on the specified subject with a queue group in JetStream.
// It returns a Subscription on success or an error if the subscription fails.
func (js *JetStream) QueueSubscribe(subject, queue string, handler nats.MsgHandler, opts ...nats.SubOpt) (*nats.Subscription, error) {
	return js.js.QueueSubscribe(subject, queue, handler, opts...)
}
