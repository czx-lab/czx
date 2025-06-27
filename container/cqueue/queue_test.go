package cqueue

import (
	"testing"
	"time"
)

// go test -v -bench=BenchmarkQueue -benchtime=50000000x -count=1 -timeout=0 ./container/cqueue
func BenchmarkQueue(b *testing.B) {
	queue := NewQueue[int](0)

	// 启动消费者
	done := make(chan struct{})
	go func() {
		count := 0
		for {
			queue.Pop()
			count++
			if count >= b.N {
				break
			}
		}
		close(done)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.Push(i)
	}
	b.StopTimer()

	// 关闭输入，等待消费完毕
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		b.Fatal("timeout waiting for consumer")
	}
}
