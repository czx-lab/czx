package actor

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
)

type producerWorker struct {
	mbox Mailbox[int]
	num  int
}

// GetId implements Worker.
func (p *producerWorker) GetId() string {
	return "producer"
}

// OnStart implements StartableWorker.
func (p *producerWorker) OnStart(ctx context.Context) {
	fmt.Println("producer started")
}

// Exec implements Worker.
func (p *producerWorker) Exec(ctx context.Context) WorkerState {
	select {
	case <-ctx.Done():
		return WorkerStopped
	case <-time.After(200 * time.Millisecond):
		p.num++
		_ = p.mbox.Write(ctx, p.num)
		return WorkerRunning
	}
}

var _ Worker = (*producerWorker)(nil)
var _ StartableWorker = (*producerWorker)(nil)

type consumerWorker struct {
	mbox    Mailbox[int]
	id      int
	ispanic bool
}

// GetId implements Worker.
func (c *consumerWorker) GetId() string {
	return fmt.Sprintf("consumer-%d", c.id)
}

// OnStart implements StartableWorker.
func (c *consumerWorker) OnStart(context.Context) {
	fmt.Println("consumer", c.id, "started")
}

// Exec implements Worker.
func (c *consumerWorker) Exec(ctx context.Context) WorkerState {
	select {
	case <-ctx.Done():
		return WorkerStopped
	case data := <-c.mbox.Receive():
		println("consumer", c.id, "received:", data)
		dts := []int{2, 5, 10, 30, 35, 40}
		if c.ispanic && slices.Contains(dts, data) {
			panic("consumer panic")
		}
		return WorkerRunning
	}
}

var _ Worker = (*consumerWorker)(nil)
var _ StartableWorker = (*consumerWorker)(nil)

func TestActor(t *testing.T) {
	t.Run("TestActor", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		mbox := NewMailbox[int](ctx, Cap(16))
		producer := New(ctx, &producerWorker{
			mbox: mbox,
		})

		producer.Start()
		defer producer.Stop()

		consumer1 := New(ctx, &consumerWorker{
			mbox: mbox,
			id:   1,
		})
		consumer1.Start()
		defer consumer1.Stop()

		consumer2 := New(ctx, &consumerWorker{
			mbox: mbox,
			id:   2,
		})

		consumer2.Start()
		defer consumer2.Stop()

		mbox.Start()
		defer mbox.Stop()

		time.AfterFunc(11*time.Second, cancel)
		<-ctx.Done()
	})

	t.Run("TestGroup", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		mbox := NewMailbox[int](ctx, Cap(16))
		producer := New(ctx, &producerWorker{
			mbox: mbox,
		})

		consumer1 := New(ctx, &consumerWorker{
			mbox: mbox,
			id:   1,
		})

		consumer2 := New(ctx, &consumerWorker{
			mbox: mbox,
			id:   2,
		})

		group := NewGroup(ctx, producer, consumer1, consumer2, mbox)
		group.Start()
		defer group.Stop()

		time.AfterFunc(11*time.Second, cancel)
		<-ctx.Done()
	})

	t.Run("TestSupervisor", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		mbox := NewMailbox[int](ctx, Cap(16))
		supervisor := NewSupervisor(ctx, SupervisorConf{
			RestartOnPanic: true,
			MaxRestarts:    2,
			TimeWindow:     5 * time.Second,
		})

		supervisor.SpawnChild(&producerWorker{
			mbox: mbox,
		})

		supervisor.SpawnChild(&consumerWorker{
			mbox:    mbox,
			id:      1,
			ispanic: true,
		}, &consumerWorker{
			mbox:    mbox,
			id:      2,
			ispanic: true,
		})

		supervisor.Start()
		defer supervisor.Stop()
		defer mbox.Stop()

		mbox.Start()
		defer mbox.Stop()

		time.AfterFunc(11*time.Second, cancel)
		<-ctx.Done()
	})
}
