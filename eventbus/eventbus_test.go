package eventbus

import (
	"testing"
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

		go func() {
			for {
				select {
				case msg, ok := <-ch_1:
					t.Logf("Received message on channel 1: %v closed: %v", msg, ok)
					DefaultBus.UnsubscribenChannel("test", ch_1)

					DefaultBus.Publish("test", "test message 2")
				case msg, ok := <-ch_2:
					t.Logf("Received message on channel 2: %v closed: %v", msg, ok)
				}
			}
		}()

		DefaultBus.Publish("test", "test message")

		<-(chan any)(nil)
	})
}
