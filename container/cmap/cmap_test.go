package cmap

import (
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/czx-lab/czx/container/recycler"
)

type Data struct {
	ID      int
	Content string
}

type Recycler struct{}

var gc bool

// Shrink implements recycler.Recycler.
func (r *Recycler) Shrink(len_ int, cap_ int) bool {
	ok := cap_ > len_ && cap_ > len_+50000
	if ok {
		fmt.Println("Shrink called:", ok, "Len:", len_, "Cap:", cap_)
	}
	gc = ok
	return ok
}

var _ recycler.Recycler = (*Recycler)(nil)

func TestCmapMemoStats(t *testing.T) {
	pool := sync.Pool{
		New: func() any {
			return new(Data)
		},
	}
	var m runtime.MemStats

	// 记录初始内存
	runtime.ReadMemStats(&m)
	fmt.Printf("Start: Alloc = %v MB\n", m.Alloc/(1024*1024))

	q := New[int, *Data]().WithRecycler(&Recycler{})
	for i := 0; i < 10000000; i++ {
		d := pool.Get().(*Data)
		d.ID = i
		d.Content = "content"
		q.Set(i, d)
	}

	// 记录插入后的内存
	runtime.ReadMemStats(&m)
	fmt.Printf("After Set: Alloc = %v MB, Len = %d\n", m.Alloc/(1024*1024), q.Len())

	for i := 0; i < 5000000; i++ {
		if data, ok := q.Get(i); ok {
			pool.Put(data)
		}
		q.Delete(i)

		// 记录弹出后的内存
		if gc {
			runtime.GC()
			runtime.ReadMemStats(&m)
			fmt.Printf("After Delete: Alloc = %v MB, Len = %d\n", m.Alloc/(1024*1024), q.Len())
		}
	}

	q.Shrink()
	runtime.GC()
	runtime.ReadMemStats(&m)
	fmt.Printf("After Shrink: Alloc = %v MB, Len = %d\n", m.Alloc/(1024*1024), q.Len())
}
