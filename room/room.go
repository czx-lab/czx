package room

import (
	"sync"
	"time"

	"github.com/IBM/sarama"
)

type Room struct {
	wg sync.WaitGroup

	opt *Option

	producer sarama.AsyncProducer
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

LOOP:
	for {
		select {
		case <-timeoutTimer.C:
			break LOOP
		case <-tickerTick.C:
			r.producer.Input() <- &sarama.ProducerMessage{
				Topic:     r.opt.kafkaTopic,
				Key:       sarama.StringEncoder(r.opt.pushKey),
				Value:     sarama.ByteEncoder(nil),
				Timestamp: time.Now(),
			}
		}
	}

	r.Stop()
}

func (r *Room) Stop() {
	r.wg.Wait()
}
