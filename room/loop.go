package room

import (
	"time"
)

type Loop struct {
	opt  *RoomConf
	quit chan struct{}
}

func NewLoop(opt *RoomConf) *Loop {
	return &Loop{
		opt: opt,
	}
}

func (l *Loop) Start() {
	l.loop()
}

func (l *Loop) loop() {
	tickerTick := time.NewTicker(l.opt.frequency)
	defer tickerTick.Stop()

	timeoutTimer := time.NewTimer(l.opt.timeout)

LOOP:
	for {
		select {
		case <-timeoutTimer.C:
			break LOOP
		case <-tickerTick.C:
		case <-l.quit:
			break LOOP
		}
	}
}

func (l *Loop) Stop() {
	close(l.quit)
}
