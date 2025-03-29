package room

import "time"

const (
	// default room id
	defaultRoomID uint64 = 1

	// game logic frame processing frequency
	frequency = time.Second / 30

	// game logic frame processing timeout
	timeout = time.Minute * 5
)

type (
	// Option defines custom room options.
	Option struct {
		roomID    uint64
		frequency time.Duration
		timeout   time.Duration
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
		roomID:    defaultRoomID,
		frequency: frequency,
		timeout:   timeout,
	}
	for _, v := range opts {
		v.apply(opt)
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
