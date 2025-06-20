package eventbus

import (
	"fmt"
	"testing"
	"time"
)

func TestEventbus(t *testing.T) {
	t.Run("TestSubscribe", func(t *testing.T) {
		ch_1 := DefaultBus.SubscribeOnChannel("test")
		if ch_1 == nil {
			t.Error("Expected non-nil channel, got nil")
		}

		ch_2 := DefaultBus.SubscribeOnChannel("test")
		if ch_2 == nil {
			t.Error("Expected non-nil channel, got nil")
		}

		DefaultBus.QueueSubscribe("queue.test", func(message any) {
			t.Logf("Received message on queue: %v", message)
		})

		go func() {
			for {
				select {
				case msg, ok := <-ch_1:
					if !ok {
						t.Logf("Received message on channel 1: %v closed: %v", msg, ok)
						break
					}
					t.Logf("Received message on channel 1: %v", msg)

					DefaultBus.Publish("test", "test message 1")

					time.Sleep(time.Second)
					fmt.Println("Unsubscribing channel 1")
					DefaultBus.UnsubscribenChannel("test", ch_1)
				case msg, ok := <-ch_2:
					if !ok {
						t.Logf("Received message on channel 2: %v closed: %v", msg, ok)
						break
					}
					time.Sleep(time.Second)
					DefaultBus.Publish("test", "test message 2")
					DefaultBus.PublishWithQueue("queue.test", "test message for queue")
					t.Logf("Received message on channel 2: %v", msg)
				}
			}
		}()

		DefaultBus.Publish("test", "test message")

		time.Sleep(10 * time.Second)
		DefaultBus.Unsubscriben("queue.test")

		<-(chan any)(nil)
	})
}
