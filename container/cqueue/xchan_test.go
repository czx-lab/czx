package cqueue

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// go test -v -bench=BenchmarkXchan -benchtime=50000000x -count=1 -timeout=0 ./container/cqueue
func BenchmarkXchan(b *testing.B) {
	ctx := b.Context()

	xch := NewXchan[int](ctx, XchanConf{
		Bufsize: 1024,
		Insize:  128,
		Outsize: 128,
	})

	// 启动消费者
	done := make(chan struct{})
	go func() {
		count := 0
		for range xch.Out() {
			count++
			if count >= b.N {
				break
			}
		}
		close(done)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xch.In() <- i
	}
	b.StopTimer()

	// 关闭输入，等待消费完毕
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		b.Fatal("timeout waiting for consumer")
	}
}

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
