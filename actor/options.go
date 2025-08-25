package actor

import "github.com/czx-lab/czx/container/recycler"

// mboxChanCap is the capacity of the mailbox channel.
const mboxChanCap = 1024

type (
	option func(o *options)
	Option option

	// options holds all configuration options.
	options struct {
		Mailbox mboxOpts
	}

	MboxOpt option
	// mboxOpts holds options for mailbox configuration.
	mboxOpts struct {
		ID       string
		Capacity int
		Recycler recycler.Recycler
	}
)

// Id sets the ID of the mailbox.
func Id(id string) MboxOpt {
	return func(o *options) {
		o.Mailbox.ID = id
	}
}

// Cap sets the capacity of the mailbox channel.
func Cap(cap int) MboxOpt {
	return func(o *options) {
		o.Mailbox.Capacity = cap
	}
}

// Recycler sets the recycler for the mailbox's internal queue.
func Recycler(r recycler.Recycler) MboxOpt {
	return func(o *options) {
		o.Mailbox.Recycler = r
	}
}

func newOptions[T ~func(o *options)](opts ...T) options {
	opt := defaultOptions()
	for _, v := range opts {
		v(opt)
	}
	return *opt
}

func defaultOptions() *options {
	return &options{
		Mailbox: mboxOpts{
			Capacity: mboxChanCap,
			Recycler: NewRecycler(),
		},
	}
}
