package frame

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type (
	NormalConf struct {
		// Capacity of the normal message queue
		QueueCap  int
		Frequency uint // Frequency of processing normal messages (in Hz)
		BatchSize int  // Number of messages to process in each batch
	}
	// Normal is a loop type that processes normal messages from the input queue.
	// It is suitable for scenarios where the processing of messages is not time-sensitive and can be handled in a regular loop.
	Normal struct {
		mu     sync.RWMutex
		conf   NormalConf
		adjust chan struct{} // Channel for adjusting the frequency dynamically
		queue  chan Message
		proc   NormalProcessor
		done   chan struct{}
		once   sync.Once
		flag   atomic.Uint32
		wg     sync.WaitGroup
	}
)

func NewNormal(conf NormalConf) *Normal {
	defaultNormalConf(&conf)

	return &Normal{
		conf:   conf,
		queue:  make(chan Message, conf.QueueCap),
		adjust: make(chan struct{}, 1), // Add buffer to avoid blocking
		done:   make(chan struct{}),
	}
}

// WithProc sets the normal processor for the normal loop.
func (n *Normal) WithProc(proc NormalProcessor) *Normal {
	n.mu.Lock()
	n.proc = proc
	n.mu.Unlock()

	return n
}

// Start implements [LoopFace].
func (n *Normal) Start(ctx context.Context) error {
	if !n.flag.CompareAndSwap(0, flagStarted) {
		return errors.New("loop already started")
	}

	defer n.flag.Store(0)

	n.mu.RLock()
	if n.proc == nil {
		n.mu.RUnlock()
		return errors.New("processor is not set")
	}

	n.mu.RUnlock()

	n.wg.Add(1)
	defer n.wg.Done()

	frequency := time.Second / time.Duration(n.conf.Frequency)
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-n.done:
			return nil
		case <-ticker.C:
			if n.flag.Load() == flagPaused {
				continue
			}

			n.exec()
		case <-n.adjust:
			n.mu.RLock()
			frequency := time.Second / time.Duration(n.conf.Frequency)
			n.mu.RUnlock()

			ticker.Reset(frequency)
		}
	}
}

// exec processes messages from the queue in batches based on the configured batch size and frequency.
// It continues to process messages until the context is canceled or there are no more messages to process.
// The function uses a select statement to handle both the context cancellation and the ticker for timing the processing intervals.
func (n *Normal) exec() {
	var processed int

	n.mu.RLock()
	batchSize := n.conf.BatchSize
	proc := n.proc
	n.mu.RUnlock()

	for {
		// If we've processed enough messages for this batch, break out of the loop
		if batchSize > 0 && processed >= batchSize {
			return
		}

		select {
		case <-n.done:
			return
		case data, ok := <-n.queue:
			if !ok {
				return
			}

			// Process the message
			proc.Process(data)
			processed++
		default:
			// No more messages to process, break out of the loop
			return
		}
	}
}

// Stop implements [LoopFace].
func (n *Normal) Stop() {
	n.once.Do(func() {
		close(n.done)
		n.wg.Wait()
		close(n.queue)

		n.mu.RLock()
		proc := n.proc
		n.mu.RUnlock()

		if proc == nil {
			return
		}

		// Drain the queue to ensure all messages are processed before stopping
		for data := range n.queue {
			proc.Process(data)
		}

		proc.OnClose()
	})
}

// Pause implements [NormalFace].
func (n *Normal) Pause() bool {
	return n.flag.CompareAndSwap(flagStarted, flagPaused)
}

// Resume implements [NormalFace].
func (n *Normal) Resume() bool {
	return n.flag.CompareAndSwap(flagPaused, flagStarted)
}

// Write implements [LoopFace].
func (n *Normal) Write(msg Message) error {
	select {
	case <-n.done:
		return errors.New("loop is closed")
	case n.queue <- msg:
		return nil
	default:
		return errors.New("queue is full")
	}
}

// WriteTimeout implements [LoopFace].
func (n *Normal) WriteTimeout(in Message, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-n.done:
		return errors.New("loop is closed")
	case n.queue <- in:
		return nil
	case <-timer.C:
		return errors.New("write timeout")
	}
}

// Frequency implements [LoopFace].
func (n *Normal) Frequency(frequency uint) error {
	if frequency == 0 {
		return errors.New("frequency must be greater than 0")
	}

	n.mu.Lock()
	n.conf.Frequency = frequency
	n.mu.Unlock()

	select {
	case n.adjust <- struct{}{}:
	default:
	}

	return nil
}

// WithBatchSize allows you to change the batch size of the normal loop at runtime.
func (n *Normal) WithBatchSize(batchSize int) error {
	if batchSize < 0 {
		return errors.New("batch size must be non-negative")
	}

	n.mu.Lock()
	n.conf.BatchSize = batchSize
	n.mu.Unlock()

	return nil
}

func defaultNormalConf(conf *NormalConf) {
	if conf.QueueCap <= 0 {
		conf.QueueCap = queueCap
	}

	if conf.Frequency == 0 {
		conf.Frequency = frequency
	}

	if conf.BatchSize < 0 {
		conf.BatchSize = 0
	}
}

var _ NormalFace = (*Normal)(nil)
