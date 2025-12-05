package eventbus

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// waitFor waits for a condition to become true within a timeout
func waitFor(t *testing.T, timeout time.Duration, condition func() bool, errorMsg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Error(errorMsg)
}

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
	waitFor(t, time.Second, func() bool {
		return received.Load() == 3
	}, "Expected 3 messages")

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
	waitFor(t, time.Second, func() bool {
		return received.Load() == 3
	}, "Expected 3 messages")

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

	// Wait for message processing - should only receive ONE message
	waitFor(t, time.Second, func() bool {
		return received.Load() >= 1
	}, "Expected at least 1 message")

	// Give some time to ensure no additional messages are processed
	time.Sleep(50 * time.Millisecond)

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

	// Only 10, 20, and 6 should pass the filter (> 5)
	waitFor(t, time.Second, func() bool {
		return received.Load() == 3
	}, "Expected 3 filtered messages")

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

	waitFor(t, time.Second, func() bool {
		return received.Load() == 100
	}, "Expected 100 messages")

	cancel()
}

func TestUnsubscribeNewMethod(t *testing.T) {
	eb := NewEventBus(10, EvtDefaultType)

	_ = eb.SubscribeOnChannel("test-unsub-new")
	_ = eb.SubscribeOnChannel("test-unsub-new")

	// Use new correctly spelled method
	eb.Unsubscribe("test-unsub-new")

	// Verify cleanup
	eb.mu.RLock()
	subs := len(eb.chanHandlers["test-unsub-new"])
	eb.mu.RUnlock()

	if subs != 0 {
		t.Errorf("Expected 0 subscribers after Unsubscribe, got %d", subs)
	}
}

func TestUnsubscribeDeprecatedAlias(t *testing.T) {
	eb := NewEventBus(10, EvtDefaultType)

	_ = eb.SubscribeOnChannel("test-deprecated")
	_ = eb.SubscribeOnChannel("test-deprecated")

	// Use deprecated method to verify backward compatibility
	eb.Unsubscriben("test-deprecated")

	// Verify cleanup
	eb.mu.RLock()
	subs := len(eb.chanHandlers["test-deprecated"])
	eb.mu.RUnlock()

	if subs != 0 {
		t.Errorf("Expected 0 subscribers after Unsubscriben, got %d", subs)
	}
}

func TestUnsubscribeChannelNewMethod(t *testing.T) {
	eb := NewEventBus(10, EvtDefaultType)

	ch := eb.SubscribeOnChannel("test-unsub-channel-new")

	// Use new correctly spelled method
	eb.UnsubscribeChannel("test-unsub-channel-new", ch)

	// Verify cleanup
	eb.mu.RLock()
	subs := len(eb.chanHandlers["test-unsub-channel-new"])
	eb.mu.RUnlock()

	if subs != 0 {
		t.Errorf("Expected 0 subscribers after UnsubscribeChannel, got %d", subs)
	}
}

func TestUnsubscribeChannelDeprecatedAlias(t *testing.T) {
	eb := NewEventBus(10, EvtDefaultType)

	ch := eb.SubscribeOnChannel("test-deprecated-channel")

	// Use deprecated method to verify backward compatibility
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

	expected := int32(numGoroutines * messagesPerGoroutine)
	waitFor(t, time.Second, func() bool {
		return received.Load() == expected
	}, "Expected all concurrent messages to be received")

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

	waitFor(t, time.Second, func() bool {
		return received1.Load() == 2 && received2.Load() == 2
	}, "Expected both subscribers to receive 2 messages each")

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

func TestSubscribeWithNilCallback(t *testing.T) {
	eb := NewEventBus(10, EvtDefaultType)

	// Subscribe with nil callback should not panic
	cancel := eb.Subscribe("test-nil-callback", nil)

	// Publish should not panic
	eb.Publish("test-nil-callback", "msg1")

	time.Sleep(50 * time.Millisecond)

	// Cancel should complete without issues
	done := make(chan struct{})
	go func() {
		cancel()
		close(done)
	}()

	select {
	case <-done:
		// Success - no panic
	case <-time.After(2 * time.Second):
		t.Error("Cancel function blocked for too long")
	}
}
