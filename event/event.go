package event

import "sync"

const (
	// EvtAgentClose is the event name for when an agent closes.
	EvtAgentClose = "AgentClose"
)

type EventBus struct {
	wg       sync.WaitGroup
	mu       sync.RWMutex
	handlers map[string][]chan any
}

// DefaultBus is the default instance of EventBus.
// It is used to manage events and their subscribers.
var DefaultBus = NewEventBus()

// NewEventBus creates a new EventBus instance.
// It initializes the handlers map to store event subscribers.
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]chan any),
	}
}

// Subscribe creates a new channel for the given event and returns it.
// The channel is buffered with a size of 1.
func (eb *EventBus) Subscribe(event string) <-chan any {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan any, 1)
	eb.handlers[event] = append(eb.handlers[event], ch)
	return ch
}

// Unsubscribe removes the channel for the given event.
// If the event does not exist, it does nothing.
func (eb *EventBus) Unsubscribe(event string, ch <-chan any) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscribers, ok := eb.handlers[event]
	if !ok {
		return
	}

	for i, subscriber := range subscribers {
		if subscriber != ch {
			continue
		}

		eb.handlers[event] = append(subscribers[:i], subscribers[i+1:]...)
		close(subscriber) // Close the channel to signal that it's no longer needed
		break
	}

	if len(eb.handlers[event]) == 0 {
		delete(eb.handlers, event)
	}
}

// SubscribeOnce creates a new channel for the given event and returns it.
// The channel is buffered with a size of 1. It will unsubscribe itself after receiving the first message.
func (eb *EventBus) SubscribeOnce(event string) <-chan any {
	ch := eb.Subscribe(event)
	go func() {
		for range ch {
			eb.Unsubscribe(event, ch)
			break
		}
	}()
	return ch
}

// SubscribeWithFilter creates a new channel for the given event and returns it.
// The channel is buffered with a size of 1. It will only receive messages that pass the filter function.
func (eb *EventBus) SubscribeWithFilter(event string, filter func(data any) bool) <-chan any {
	ch := make(chan any, 1)
	go func() {
		for msg := range eb.Subscribe(event) {
			if filter(msg) {
				ch <- msg
			}
		}
	}()
	return ch
}

// Publish sends the data to all subscribers of the given event.
// If there are no subscribers, it does nothing.
func (eb *EventBus) Publish(event string, data any) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	subscribers, ok := eb.handlers[event]
	if !ok {
		return
	}

	for _, ch := range subscribers {
		go func(c chan any) {
			select {
			case c <- data: // Send data to the channel
			default: // If the channel is full, skip sending
			}
		}(ch)
	}
}
