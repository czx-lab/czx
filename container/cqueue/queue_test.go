package cqueue

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// go test -v -bench=BenchmarkQueue -benchmem -benchtime=50000000x -count=1 -timeout=0 ./container/cqueue
type Data struct {
	ID int
}

func TestQueueMemStats(t *testing.T) {
	var m runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&m)
	fmt.Printf("Start: Alloc = %v MB\n", m.Alloc/(1024*1024))

	q := NewQueue[int](0)
	count := 10000000

	// Push
	for i := 0; i < count; i++ {
		q.Push(i)
	}
	runtime.ReadMemStats(&m)
	fmt.Printf("After Push: Alloc = %v MB, Len = %d\n", m.Alloc/(1024*1024), q.Len())

	// Pop
	for i := 0; i < 500000; i++ {
		q.Pop()
	}
	runtime.GC()
	runtime.ReadMemStats(&m)
	fmt.Printf("After Pop: Alloc = %v MB, Len = %d\n", m.Alloc/(1024*1024), q.Len())

	// Shrink
	q.Shrink()
	runtime.GC()
	runtime.ReadMemStats(&m)
	fmt.Printf("After Shrink: Alloc = %v MB, Len = %d\n", m.Alloc/(1024*1024), q.Len())
}

func BenchmarkQueue(b *testing.B) {

	queue := NewQueue[*Data](0)

	pool := sync.Pool{
		New: func() any {
			return new(Data)
		},
	}

	// 启动消费者
	done := make(chan struct{})
	go func() {
		count := 0
		for {
			data, ok := queue.Pop()
			if ok && data != nil {
				pool.Put(data)
			}
			count++
			if count >= b.N {
				break
			}
		}
		close(done)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := pool.Get().(*Data)
		q.ID = i
		queue.Push(q)
	}
	b.StopTimer()

	// 关闭输入，等待消费完毕
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		b.Fatal("timeout waiting for consumer")
	}
}
