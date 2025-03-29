package room

import (
	"sync"
	"time"
)

type Room struct {
	wg sync.WaitGroup

	opt *Option
}

func NewRoom(opt *Option) *Room {
	return &Room{
		opt: opt,
	}
}

func (r *Room) Run() {
	r.wg.Add(1)
	defer r.wg.Done()

	tickerTick := time.NewTicker(r.opt.frequency)
	defer tickerTick.Stop()

	timeoutTimer := time.NewTimer(r.opt.timeout)

	for {
		select {
		case <-timeoutTimer.C:

		case <-tickerTick.C:

		}
	}
}

func (r *Room) Stop() {
	r.wg.Wait()
}
