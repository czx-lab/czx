package eventbus

import (
	"testing"
)

func TestEventbus(t *testing.T) {
	t.Run("TestSubscribe", func(t *testing.T) {
		ch_1 := DefaultBus.Subscribe("test")
		if ch_1 == nil {
			t.Error("Expected non-nil channel, got nil")
		}

		ch_2 := DefaultBus.Subscribe("test")
		if ch_2 == nil {
			t.Error("Expected non-nil channel, got nil")
		}

		go func() {
			for {
				select {
				case msg := <-ch_1:
					t.Logf("Received message on channel 1: %v", msg)
				case msg := <-ch_2:
					t.Logf("Received message on channel 2: %v", msg)
				}
			}
		}()

		DefaultBus.Publish("test", "test message")

		<-(chan any)(nil)
	})
}
