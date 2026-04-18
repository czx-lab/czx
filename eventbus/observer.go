package eventbus

import "sync"

type (
	// Observer is an interface that defines the methods for an event observer.
	// It is a generic interface that can be implemented for any type of event.
	Observer[T any] interface {
		// Name returns the name of the observer.
		// This is used to identify the observer when subscribing and unsubscribing.
		Name() string
		// OnNotify is called when an event is published. It receives the event data as a parameter.
		OnNotify(T)
	}

	// SubjectFace is an interface that defines the methods for an event subject.
	// It allows observers to subscribe and unsubscribe from events, and notifies them when an event occurs.
	SubjectFace[T any] interface {
		// Subscribe adds an observer to the subject's list of observers.
		Subscribe(Observer[T])
		// Unsubscribe removes an observer from the subject's list of observers.
		Unsubscribe(Observer[T])
		// Notify sends an event to all subscribed observers.
		Notify(T)
		// Count returns the number of observers currently subscribed to the subject.
		Count() int
	}

	// Subject is a struct that implements the SubjectFace interface.
	// It maintains a list of observers and provides methods to subscribe, unsubscribe, and notify them.
	Subject[T any] struct {
		rmu       sync.RWMutex
		observers map[string]Observer[T]
	}
)

func NewSubject[T any]() *Subject[T] {
	return &Subject[T]{
		observers: make(map[string]Observer[T]),
	}
}

// Count implements [SubjectFace].
func (s *Subject[T]) Count() int {
	s.rmu.RLock()
	defer s.rmu.RUnlock()
	return len(s.observers)
}

// Notify implements [SubjectFace].
func (s *Subject[T]) Notify(event T) {
	s.rmu.RLock()

	snapshot := make([]Observer[T], 0, len(s.observers))
	for _, observer := range s.observers {
		snapshot = append(snapshot, observer)
	}

	s.rmu.RUnlock()

	for _, observer := range snapshot {
		observer.OnNotify(event)
	}
}

// Subscribe implements [SubjectFace].
func (s *Subject[T]) Subscribe(observer Observer[T]) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.observers[observer.Name()] = observer
}

// Unsubscribe implements [SubjectFace].
func (s *Subject[T]) Unsubscribe(observer Observer[T]) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	delete(s.observers, observer.Name())
}

var _ SubjectFace[any] = (*Subject[any])(nil)
