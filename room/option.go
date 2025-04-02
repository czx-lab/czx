package room

import (
	"time"
)

const (
	// default room id
	defaultRoomID uint64 = 1
	// game logic frame processing frequency
	frequency = time.Second / 30
	// game logic frame processing timeout
	timeout = time.Minute * 5
	// default max player count
	defaultMaxPlayer = 5
	// default max buffer size
	defaultMaxBufferSize = 4096
)

type (
	// Option defines custom room options.
	RoomConf struct {
		// size of the message buffer
		maxBufferSize uint64
		// room id
		roomID uint64
		// max player count
		maxPlayer int
		// frequency of game logic frame processing
		frequency time.Duration
		// timeout for game logic frame processing
		timeout time.Duration
	}

	// OptionFunc defines the method to customize a Option.
	OptionFunc func(*RoomConf)
	IOption    interface {
		apply(*RoomConf)
	}
)

func (fn OptionFunc) apply(opt *RoomConf) {
	fn(opt)
}

func NewOption(opts ...IOption) *RoomConf {
	opt := &RoomConf{
		roomID:        defaultRoomID,
		frequency:     frequency,
		timeout:       timeout,
		maxBufferSize: defaultMaxBufferSize,
		maxPlayer:     defaultMaxPlayer,
	}
	for _, v := range opts {
		v.apply(opt)
	}

	return opt
}

func WithRoomID(id uint64) OptionFunc {
	return func(o *RoomConf) {
		o.roomID = id
	}
}

func WithFrequency(frequency time.Duration) OptionFunc {
	return func(o *RoomConf) {
		o.frequency = frequency
	}
}

func WithTimeout(timeout time.Duration) OptionFunc {
	return func(o *RoomConf) {
		o.timeout = timeout
	}
}

func WithMaxPlayer(maxPlayer int) OptionFunc {
	return func(o *RoomConf) {
		o.maxPlayer = maxPlayer
	}
}

func WithMaxBufferSize(maxBufferSize uint64) OptionFunc {
	return func(o *RoomConf) {
		o.maxBufferSize = maxBufferSize
	}
}
