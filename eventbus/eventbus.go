package eventbus

import (
	"slices"
	"sync"
)

const (
	// EvtAgentClose is the event name for when an agent closes.
	EvtAgentClose = "AgentClose"
	//	Event name for when an agent starts.
	EvtNewAgent = "AgentNew"
)

// DefaultCapacity is the default capacity for event channels.
// It defines how many messages can be buffered in each channel before blocking.
var DefaultCapacity = 100

type EventBus struct {
	mu       sync.RWMutex
	handlers map[string][]chan any
	capacity int
}

// DefaultBus is the default instance of EventBus.
// It is used to manage events and their subscribers.
var DefaultBus = NewEventBus(DefaultCapacity)

// NewEventBus creates a new EventBus instance.
// It initializes the handlers map to store event subscribers.
func NewEventBus(cap int) *EventBus {
	return &EventBus{
		handlers: make(map[string][]chan any),
		capacity: cap,
	}
}

// SubscribeOnChannel creates a new channel for the given event and returns it.
// The channel is buffered with a size of 1. It will not unsubscribe itself.
func (eb *EventBus) SubscribeOnChannel(event string) <-chan any {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan any, eb.capacity)
	eb.handlers[event] = append(eb.handlers[event], ch)

	return ch
}

// Subscribe creates a new channel for the given event and returns it.
// The channel is buffered with a size of 1. It will not unsubscribe itself.
func (eb *EventBus) Subscribe(event string, callback func(message any)) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan any, eb.capacity)
	eb.handlers[event] = append(eb.handlers[event], ch)

	go func() {
		for msg := range ch {
			if callback == nil {
				continue
			}
			callback(msg) // Call the callback function with the received message
		}
	}()
}

// Unsubscriben removes the given event from the handler mapping.
// It also closes the channel to indicate that it is no longer needed.
func (eb *EventBus) Unsubscriben(event string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscribers, ok := eb.handlers[event]
	if !ok {
		return
	}

	for _, subscriber := range subscribers {
		close(subscriber)
	}

	delete(eb.handlers, event)
}

// UnsubscribenChannel removes the specified channel for the given event from the handlers map.
// It also closes the channel to signal that it's no longer needed.
func (eb *EventBus) UnsubscribenChannel(event string, ch <-chan any) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscribers, ok := eb.handlers[event]
	if !ok {
		return
	}

	for i, subscriber := range subscribers {
		if subscriber == ch {
			eb.handlers[event] = slices.Delete(subscribers, i, i+1)
			close(subscriber)
			break
		}
	}

	if len(eb.handlers[event]) == 0 {
		delete(eb.handlers, event)
	}
}

// SubscribeOnce creates a new subscription for the given event.
// It will automatically unsubscribe itself after receiving the first message.
func (eb *EventBus) SubscribeOnce(event string, callback func(message any)) {
	ch := make(chan any, eb.capacity)
	eb.mu.Lock()
	eb.handlers[event] = append(eb.handlers[event], ch)
	eb.mu.Unlock()

	go func() {
		for msg := range ch {
			if callback != nil {
				callback(msg)
			}
			eb.UnsubscribenChannel(event, ch)
			// Removed the unconditional break to allow the loop to process all messages.
		}
	}()
}

// SubscribeWithFilter creates a new subscription for the given event with a filter function.
// It will only pass messages that satisfy the filter condition to the callback.
func (eb *EventBus) SubscribeWithFilter(event string, filter func(data any) bool, callback func(message any)) {
	ch := make(chan any, eb.capacity)
	eb.mu.Lock()
	eb.handlers[event] = append(eb.handlers[event], ch)
	eb.mu.Unlock()

	go func() {
		for msg := range ch {
			if filter(msg) && callback != nil {
				callback(msg)
			}
		}
	}()
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
		go func(q chan any) {
			select {
			case ch <- data:
			default:
				// If the channel is full, we skip sending the message.
				// This prevents blocking the publisher if the channel is full.
			}
		}(ch)
	}
}
