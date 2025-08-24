package actor

type (
	option func(o *options)

	options struct {
		Mailbox mboxOpts
	}
	MboxOpt option

	mboxOpts struct {
		Capacity int
	}
)

func Cap(cap int) option {
	return func(o *options) {
		o.Mailbox.Capacity = cap
	}
}

func newOptions(opts ...option) options {
	return options{}
}
