package eventbus

import (
	"slices"
	"sync"
	"sync/atomic"

	"github.com/czx-lab/czx/container/cqueue"
	"github.com/czx-lab/czx/container/recycler"
	"github.com/czx-lab/czx/xlog"
)

const (
	// EvtAgentClose is the event name for when an agent closes.
	EvtAgentClose = "AgentClose"
	//	Event name for when an agent starts.
	EvtNewAgent = "AgentNew"
	// EvtDefaultType is the default name for the event bus.
	EvtDefaultType EvtType = "channel"
	EvtXqueueType  EvtType = "xqueue"
)

type (
	EvtType  string
	EventBus struct {
		mu            sync.RWMutex
		chanHandlers  map[string][]chan any
		queueHandlers map[string][]*cqueue.Queue[any]
		capacity      int32
		typ           EvtType
		recycler      recycler.Recycler
	}
)

var (
	// DefaultCapacity is the default capacity for event channels.
	// It defines how many messages can be buffered in each channel before blocking.
	defaultCapacity int32 = 100
	// DefaultBus is the default instance of EventBus.
	// It is used to manage events and their subscribers.
	DefaultBus              = NewEventBus(defaultCapacity, EvtDefaultType)
	busMu      sync.RWMutex // Mutex to protect the DefaultBus instance
)

// LoadCapacity sets the default capacity for event channels.
func LoadCapacity(cap int, typ EvtType, r recycler.Recycler) {
	atomic.StoreInt32(&defaultCapacity, int32(cap))
	busMu.Lock()
	defer busMu.Unlock()

	DefaultBus = NewEventBus(defaultCapacity, typ).WithRecycler(r)
}

// NewEventBus creates a new EventBus instance.
// It initializes the handlers map to store event subscribers.
func NewEventBus(cap int32, typ EvtType) *EventBus {
	return &EventBus{
		chanHandlers:  make(map[string][]chan any),
		queueHandlers: make(map[string][]*cqueue.Queue[any]),
		capacity:      cap,
		typ:           typ,
	}
}

func (eb *EventBus) WithRecycler(r recycler.Recycler) *EventBus {
	eb.recycler = r
	return eb
}

// Type returns the name of the event bus.
func (eb *EventBus) Type() EvtType {
	return eb.typ
}

// SubscribeOnChannel creates a new channel for the given event and returns it.
// The channel is buffered with a size of 1. It will not unsubscribe itself.
func (eb *EventBus) SubscribeOnChannel(event string) <-chan any {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan any, eb.capacity)
	eb.chanHandlers[event] = append(eb.chanHandlers[event], ch)

	return ch
}

// Subscribe creates a new channel for the given event and returns it.
// The channel is buffered with a size of 1. It will not unsubscribe itself.
func (eb *EventBus) Subscribe(event string, callback func(message any)) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan any, eb.capacity)
	eb.chanHandlers[event] = append(eb.chanHandlers[event], ch)

	go func() {
		for msg := range ch {
			if callback == nil {
				continue
			}
			callback(msg) // Call the callback function with the received message
		}
	}()
}

// QueueSubscribe creates a new queue for the given event and starts a goroutine to process messages.
// It allows for processing messages in a queue-like manner, where messages are processed in the order they are received.
func (eb *EventBus) QueueSubscribe(event string, callback func(message any)) {
	eb.mu.Lock()
	queue := cqueue.NewQueue[any](int(eb.capacity)).WithRecycler(eb.recycler)
	eb.queueHandlers[event] = append(eb.queueHandlers[event], queue)
	eb.mu.Unlock()

	go func() {
		for {
			// Try to dequeue a message
			msg := queue.WaitPop()

			if callback != nil {
				callback(msg) // Call the callback function with the received message
			}
		}
	}()
}

// SubscribeOnQueue creates a new queue for the given event and returns it.
func (eb *EventBus) SubscribeOnQueue(event string) *cqueue.Queue[any] {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	queue := cqueue.NewQueue[any](int(eb.capacity)).WithRecycler(eb.recycler)
	eb.queueHandlers[event] = append(eb.queueHandlers[event], queue)

	return queue
}

// Unsubscriben removes the given event from the handler mapping.
// This function will clear all channels and queues associated with the event.
// It is used to clean up resources when an event is no longer needed.
func (eb *EventBus) Unsubscriben(event string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscribers, ok := eb.chanHandlers[event]
	if !ok {
		goto UnsubscribenQueue
	}

	for _, subscriber := range subscribers {
		close(subscriber)
	}

	delete(eb.chanHandlers, event)

	// unsubscribe from queues if they exist
UnsubscribenQueue:
	if len(eb.queueHandlers[event]) == 0 {
		return
	}

	queueSubs := eb.queueHandlers[event]
	for _, queue := range queueSubs {
		queue.Clear()
	}

	delete(eb.queueHandlers, event)
}

// UnsubscribenChannel removes the specified channel for the given event from the handlers map.
// It also closes the channel to signal that it's no longer needed.
func (eb *EventBus) UnsubscribenChannel(event string, ch <-chan any) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscribers, ok := eb.chanHandlers[event]
	if !ok {
		return
	}

	for i, subscriber := range subscribers {
		if subscriber == ch {
			eb.chanHandlers[event] = slices.Delete(subscribers, i, i+1)
			close(subscriber)
			break
		}
	}

	if len(eb.chanHandlers[event]) == 0 {
		delete(eb.chanHandlers, event)
	}
}

// UnsubscribeQueue removes the specified queue for the given event from the queue handlers map.
func (eb *EventBus) UnsubscribeQueue(event string, queue *cqueue.Queue[any]) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	queues, ok := eb.queueHandlers[event]
	if !ok {
		return
	}

	for i, q := range queues {
		if q == queue {
			eb.queueHandlers[event] = slices.Delete(queues, i, i+1)
			break
		}
	}

	if len(eb.queueHandlers[event]) == 0 {
		delete(eb.queueHandlers, event)
	}
}

// SubscribeOnce creates a new subscription for the given event.
// It will automatically unsubscribe itself after receiving the first message.
func (eb *EventBus) SubscribeOnce(event string, callback func(message any)) {
	ch := make(chan any, eb.capacity)
	eb.mu.Lock()
	eb.chanHandlers[event] = append(eb.chanHandlers[event], ch)
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
	eb.chanHandlers[event] = append(eb.chanHandlers[event], ch)
	eb.mu.Unlock()

	go func() {
		for msg := range ch {
			if filter(msg) && callback != nil {
				callback(msg)
			}
		}
	}()
}

// PublishWithQueue sends the data to all queues subscribed to the given event.
// If there are no queues, it does nothing.
func (eb *EventBus) PublishWithQueue(event string, data any) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	queues, ok := eb.queueHandlers[event]
	if !ok {
		return
	}

	for _, queue := range queues {
		if err := queue.Push(data); err != nil {
			xlog.Write().Sugar().Errorf("EventBus: failed to push data to queue for event %s: %v", event, err)
			continue
		}
	}
}

// Publish sends the data to all subscribers of the given event.
// If there are no subscribers, it does nothing.
func (eb *EventBus) Publish(event string, data any) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	subscribers, ok := eb.chanHandlers[event]
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
				xlog.Write().Sugar().Warnf("EventBus: channel full, skipping message for event %s", event)
			}
		}(ch)
	}
}
