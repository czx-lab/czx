package room

import (
	"time"

	"github.com/IBM/sarama"
)

type Loop struct {
	opt      *Option
	producer sarama.AsyncProducer
	quit     chan struct{}
}

func NewLoop(opt *Option) *Loop {
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
			l.producer.Input() <- &sarama.ProducerMessage{
				Topic:     l.opt.kafkaTopic,
				Key:       sarama.StringEncoder(l.opt.pushKey),
				Value:     sarama.ByteEncoder(nil),
				Timestamp: time.Now(),
			}
		case <-l.quit:
			break LOOP
		}
	}
}

func (l *Loop) Stop() {
	close(l.quit)
}
