package cqueue

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/czx-lab/czx/algo/random"
	"github.com/czx-lab/czx/container/recycler"
)

// go test -v -bench=BenchmarkQueue -benchmem -benchtime=50000000x -count=1 -timeout=0 ./container/cqueue
type Data struct {
	ID int
}

type Recycler struct{}

// Shrink implements recycler.Recycler.
func (r *Recycler) Shrink(len_ int, cap_ int) bool {
	ok := cap_ > len_ && cap_ > len_+10
	if ok {
		fmt.Println("Shrink called:", ok, "Len:", len_, "Cap:", cap_)
	}
	return ok
}

var _ recycler.Recycler = (*Recycler)(nil)

func TestQueueMemStats(t *testing.T) {
	var m runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&m)
	fmt.Printf("Start: Alloc = %v MB\n", m.Alloc/(1024*1024))

	q := NewQueue[*Data](0).WithRecycler(&Recycler{})
	count := 10000000

	var num int
	go func() {
		for {
			time.Sleep(time.Second)
			num++
			pnum := random.RangeRandom(200000, 300000)
			// if rand.Float64() < 0.01 {
			// 	pnum = q.Len()
			// }
			q.PopBatch(pnum)

			runtime.GC()

			runtime.ReadMemStats(&m)
			fmt.Printf("After Pop: Alloc = %v MB, Len = %d\n", m.Alloc/(1024*1024), q.Len())

			_num := random.RangeRandom(100000, 200000)
			if num%10 == 0 {
				for i := 0; i < _num; i++ {
					q.Push(&Data{ID: i}) // Push a dummy value to trigger memory allocation
				}
			}

		}
	}()

	// Push
	for i := 0; i < count; i++ {
		q.Push(&Data{ID: i})
	}
	runtime.ReadMemStats(&m)
	fmt.Printf("After Push: Alloc = %v MB, Len = %d\n", m.Alloc/(1024*1024), q.Len())

	select {}
	// Pop
	// for i := 0; i < 18000; i++ {
	// 	q.Pop()
	// }
	// runtime.GC()
	// runtime.ReadMemStats(&m)
	// fmt.Printf("After Pop: Alloc = %v MB, Len = %d\n", m.Alloc/(1024*1024), q.Len())

	// Shrink
	q.Shrink()
	runtime.GC()
	runtime.ReadMemStats(&m)
	fmt.Printf("After Shrink: Alloc = %v MB, Len = %d\n", m.Alloc/(1024*1024), q.Len())
}

// func BenchmarkQueue(b *testing.B) {

// 	queue := NewQueue[*Data](0)

// 	pool := sync.Pool{
// 		New: func() any {
// 			return new(Data)
// 		},
// 	}

// 	// 启动消费者
// 	done := make(chan struct{})
// 	go func() {
// 		count := 0
// 		for {
// 			data, ok := queue.Pop()
// 			if ok && data != nil {
// 				pool.Put(data)
// 			}
// 			count++
// 			if count >= b.N {
// 				break
// 			}
// 		}
// 		close(done)
// 	}()

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		q := pool.Get().(*Data)
// 		q.ID = i
// 		queue.Push(q)
// 	}
// 	b.StopTimer()

// 	// 关闭输入，等待消费完毕
// 	select {
// 	case <-done:
// 	case <-time.After(5 * time.Second):
// 		b.Fatal("timeout waiting for consumer")
// 	}
// }
