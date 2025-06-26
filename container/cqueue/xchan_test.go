package cqueue

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestXchan(t *testing.T) {
	t.Run("TestXchan", func(t *testing.T) {
		xchan := NewXchan[int](context.TODO(), XchanConf{
			Bufsize: 100,
			Insize:  1,
			Outsize: 1,
		})

		go func() {
			for v := range 1000 {
				xchan.In() <- v + 1
			}
		}()

		go func() {
			for {
				time.Sleep(time.Second)
				val := <-xchan.Out()
				fmt.Printf("Received value: %d\n", val)
			}
		}()

		<-(chan any)(nil)
	})
}

func TestXchanContext(t *testing.T) {
	t.Run("TestXchanContext", func(t *testing.T) {
		parent := context.Background()
		ctx, cancel := context.WithCancel(parent)
		xchan := NewXchan[int](ctx, XchanConf{
			Bufsize: 100,
			Insize:  10,
			Outsize: 10,
		})

		go func() {
			for v := range 1000 {
				xchan.In() <- v + 1
				if v == 99 {
					cancel()
				}
			}
		}()

		go func() {
			for {
				val := <-xchan.Out()
				if val != 0 {
					fmt.Printf("Received value: %d\n", val)
				}
			}
		}()

		<-(chan any)(nil)
	})
}
