package room

import (
	"fmt"
	"time"
)

const (
	// default room id
	defaultRoomID uint64 = 1
	// kafka message topic
	topic = "room_frames"
	// game logic frame processing frequency
	frequency = time.Second / 30
	// game logic frame processing timeout
	timeout = time.Minute * 5
)

type (
	// Option defines custom room options.
	Option struct {
		kafkaTopic string
		pushKey    string
		roomID     uint64
		frequency  time.Duration
		timeout    time.Duration
	}

	// OptionFunc defines the method to customize a Option.
	OptionFunc func(*Option)
	IOption    interface {
		apply(*Option)
	}
)

func (fn OptionFunc) apply(opt *Option) {
	fn(opt)
}

func NewOption(opts ...IOption) *Option {
	opt := &Option{
		roomID:     defaultRoomID,
		frequency:  frequency,
		timeout:    timeout,
		kafkaTopic: topic,
	}
	for _, v := range opts {
		v.apply(opt)
	}

	if len(opt.pushKey) == 0 {
		opt.pushKey = fmt.Sprint(opt.roomID)
	}
	return opt
}

func WithRoomID(id uint64) OptionFunc {
	return func(o *Option) {
		o.roomID = id
	}
}

func WithFrequency(frequency time.Duration) OptionFunc {
	return func(o *Option) {
		o.frequency = frequency
	}
}

func WithTimeout(timeout time.Duration) OptionFunc {
	return func(o *Option) {
		o.timeout = timeout
	}
}

func WithTopic(topic string) OptionFunc {
	return func(o *Option) {
		o.kafkaTopic = topic

	}
}

func WithKey(key string) OptionFunc {
	return func(o *Option) {
		o.pushKey = key
	}
}
