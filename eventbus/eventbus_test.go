package eventbus

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSubscribe(t *testing.T) {
	eb := NewEventBus(10, EvtDefaultType)

	var received atomic.Int32
	cancel := eb.Subscribe("test-subscribe", func(message any) {
		received.Add(1)
	})

	// Publish messages
	eb.Publish("test-subscribe", "msg1")
	eb.Publish("test-subscribe", "msg2")
	eb.Publish("test-subscribe", "msg3")

	// Wait for messages to be processed
	time.Sleep(50 * time.Millisecond)

	if received.Load() != 3 {
		t.Errorf("Expected 3 messages, got %d", received.Load())
	}

	// Cancel subscription
	cancel()

	// Verify goroutine cleanup by publishing more messages (should not be received)
	beforeCancel := received.Load()
	eb.Publish("test-subscribe", "msg4")
	time.Sleep(50 * time.Millisecond)

	if received.Load() != beforeCancel {
		t.Errorf("Expected no more messages after cancel, got %d", received.Load()-beforeCancel)
	}
}

func TestSubscribeCancelFunction(t *testing.T) {
	eb := NewEventBus(10, EvtDefaultType)

	var wg sync.WaitGroup
	wg.Add(1)

	cancel := eb.Subscribe("test-cancel", func(message any) {
		wg.Done()
	})

	eb.Publish("test-cancel", "test")
	wg.Wait()

	// Cancel should complete without blocking forever (goroutine cleanup)
	done := make(chan struct{})
	go func() {
		cancel()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Cancel function blocked for too long, possible goroutine leak")
	}
}

func TestQueueSubscribe(t *testing.T) {
	eb := NewEventBus(10, EvtXqueueType)

	var received atomic.Int32
	cancel := eb.QueueSubscribe("test-queue", func(message any) {
		received.Add(1)
	})

	// Publish messages
	eb.PublishWithQueue("test-queue", "msg1")
	eb.PublishWithQueue("test-queue", "msg2")
	eb.PublishWithQueue("test-queue", "msg3")

	// Wait for messages to be processed
	time.Sleep(100 * time.Millisecond)

	if received.Load() != 3 {
		t.Errorf("Expected 3 messages, got %d", received.Load())
	}

	// Cancel subscription
	done := make(chan struct{})
	go func() {
		cancel()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Cancel function blocked for too long, possible goroutine leak")
	}
}

func TestSubscribeOnce(t *testing.T) {
	eb := NewEventBus(10, EvtDefaultType)

	var received atomic.Int32
	_ = eb.SubscribeOnce("test-once", func(message any) {
		received.Add(1)
	})

	// Publish multiple messages
	eb.Publish("test-once", "msg1")
	eb.Publish("test-once", "msg2")
	eb.Publish("test-once", "msg3")

	// Wait for message processing
	time.Sleep(100 * time.Millisecond)

	// Should only receive ONE message (SubscribeOnce semantics)
	if received.Load() != 1 {
		t.Errorf("SubscribeOnce should receive exactly 1 message, got %d", received.Load())
	}
}

func TestSubscribeOnceCancelBeforeReceive(t *testing.T) {
	eb := NewEventBus(10, EvtDefaultType)

	var received atomic.Int32
	cancel := eb.SubscribeOnce("test-once-cancel", func(message any) {
		received.Add(1)
	})

	// Cancel before publishing
	done := make(chan struct{})
	go func() {
		cancel()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Cancel function blocked for too long")
	}

	// Publish should not cause any issues
	eb.Publish("test-once-cancel", "msg1")
	time.Sleep(50 * time.Millisecond)

	if received.Load() != 0 {
		t.Errorf("Expected 0 messages after cancel, got %d", received.Load())
	}
}

func TestSubscribeWithFilter(t *testing.T) {
	eb := NewEventBus(10, EvtDefaultType)

	var received atomic.Int32
	cancel := eb.SubscribeWithFilter("test-filter", func(data any) bool {
		val, ok := data.(int)
		return ok && val > 5
	}, func(message any) {
		received.Add(1)
	})

	// Publish messages with different values
	eb.Publish("test-filter", 1)
	eb.Publish("test-filter", 10)
	eb.Publish("test-filter", 3)
	eb.Publish("test-filter", 20)
	eb.Publish("test-filter", 5)
	eb.Publish("test-filter", 6)

	time.Sleep(100 * time.Millisecond)

	// Only 10, 20, and 6 should pass the filter (> 5)
	if received.Load() != 3 {
		t.Errorf("Expected 3 filtered messages, got %d", received.Load())
	}

	cancel()
}

func TestPublishNoGoroutineLeak(t *testing.T) {
	// Use larger capacity to handle burst publishing
	eb := NewEventBus(1000, EvtDefaultType)

	var received atomic.Int32
	cancel := eb.Subscribe("test-publish", func(message any) {
		received.Add(1)
	})

	// Publish many messages
	for i := 0; i < 100; i++ {
		eb.Publish("test-publish", i)
	}

	time.Sleep(200 * time.Millisecond)

	if received.Load() != 100 {
		t.Errorf("Expected 100 messages, got %d", received.Load())
	}

	cancel()
}

func TestUnsubscribeDeprecatedAlias(t *testing.T) {
	eb := NewEventBus(10, EvtDefaultType)

	_ = eb.SubscribeOnChannel("test-deprecated")
	_ = eb.SubscribeOnChannel("test-deprecated")

	// Use deprecated method
	eb.Unsubscriben("test-deprecated")

	// Verify cleanup
	eb.mu.RLock()
	subs := len(eb.chanHandlers["test-deprecated"])
	eb.mu.RUnlock()

	if subs != 0 {
		t.Errorf("Expected 0 subscribers after Unsubscriben, got %d", subs)
	}
}

func TestUnsubscribeChannelDeprecatedAlias(t *testing.T) {
	eb := NewEventBus(10, EvtDefaultType)

	ch := eb.SubscribeOnChannel("test-deprecated-channel")

	// Use deprecated method
	eb.UnsubscribenChannel("test-deprecated-channel", ch)

	// Verify cleanup
	eb.mu.RLock()
	subs := len(eb.chanHandlers["test-deprecated-channel"])
	eb.mu.RUnlock()

	if subs != 0 {
		t.Errorf("Expected 0 subscribers after UnsubscribenChannel, got %d", subs)
	}
}

func TestConcurrentPublish(t *testing.T) {
	// Use larger capacity to handle concurrent publishing
	eb := NewEventBus(1000, EvtDefaultType)

	var received atomic.Int32
	cancel := eb.Subscribe("test-concurrent", func(message any) {
		received.Add(1)
	})

	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				eb.Publish("test-concurrent", j)
			}
		}()
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	expected := int32(numGoroutines * messagesPerGoroutine)
	if received.Load() != expected {
		t.Errorf("Expected %d messages, got %d", expected, received.Load())
	}

	cancel()
}

func TestMultipleSubscribersReceiveAllMessages(t *testing.T) {
	eb := NewEventBus(10, EvtDefaultType)

	var received1, received2 atomic.Int32

	cancel1 := eb.Subscribe("test-multi", func(message any) {
		received1.Add(1)
	})
	cancel2 := eb.Subscribe("test-multi", func(message any) {
		received2.Add(1)
	})

	eb.Publish("test-multi", "msg1")
	eb.Publish("test-multi", "msg2")

	time.Sleep(100 * time.Millisecond)

	if received1.Load() != 2 {
		t.Errorf("Subscriber 1 expected 2 messages, got %d", received1.Load())
	}
	if received2.Load() != 2 {
		t.Errorf("Subscriber 2 expected 2 messages, got %d", received2.Load())
	}

	cancel1()
	cancel2()
}

func TestUnsubscribe(t *testing.T) {
	eb := NewEventBus(10, EvtDefaultType)

	_ = eb.SubscribeOnChannel("test-unsub")
	_ = eb.SubscribeOnChannel("test-unsub")
	_ = eb.QueueSubscribe("test-unsub", func(message any) {})

	// Unsubscribe all
	eb.Unsubscribe("test-unsub")

	// Verify cleanup
	eb.mu.RLock()
	chanSubs := len(eb.chanHandlers["test-unsub"])
	queueSubs := len(eb.queueHandlers["test-unsub"])
	eb.mu.RUnlock()

	if chanSubs != 0 {
		t.Errorf("Expected 0 channel subscribers, got %d", chanSubs)
	}
	if queueSubs != 0 {
		t.Errorf("Expected 0 queue subscribers, got %d", queueSubs)
	}
}
