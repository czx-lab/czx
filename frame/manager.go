package frame

import (
	"errors"
	"sync"

	"github.com/czx-lab/czx/container/cmap"
)

var (
	ErrLoopExists   = errors.New("loop already exists")
	ErrLoopNotFound = errors.New("loop not found")
)

type LoopManager struct {
	wg    sync.WaitGroup
	mu    sync.RWMutex
	loops *cmap.CMap[string, *Loop]
}

func NewManager() *LoopManager {
	return &LoopManager{
		loops: cmap.New[string, *Loop](),
	}
}

// Add adds a new loop to the manager.
// It starts the loop in a separate goroutine.
func (lm *LoopManager) Add(id string, loop *Loop) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.loops.Has(id) {
		return ErrLoopExists
	}

	lm.loops.Set(id, loop)

	lm.wg.Add(1)
	go func() {
		defer lm.wg.Done()

		// Start the loop
		loop.Start()
	}()

	return nil
}

// Remove removes a loop from the manager by its ID.
// It stops the loop and waits for it to finish processing before removing it.
func (lm *LoopManager) Remove(id string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	loop, exists := lm.loops.Get(id)
	if !exists {
		return ErrLoopNotFound
	}

	// Stop the loop
	loop.Stop()

	// Remove the loop from the manager
	lm.loops.Delete(id)

	return nil
}

// Get retrieves a loop by its ID.
// It returns an error if the loop is not found.
func (lm *LoopManager) Get(id string) (*Loop, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	loop, exists := lm.loops.Get(id)
	if !exists {
		return nil, ErrLoopNotFound
	}

	return loop, nil
}

// Loops returns a slice of all loops managed by the LoopManager.
// It does not lock the manager, so it may return a snapshot of the loops at the time of the call.
func (lm *LoopManager) Loops() []*Loop {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	loops := make([]*Loop, 0, lm.loops.Len())
	lm.loops.Iterator(func(_ string, loop *Loop) bool {
		loops = append(loops, loop)
		return true
	})

	return loops
}

// Has checks if a loop exists in the manager by its ID.
// It returns true if the loop exists, false otherwise.
func (lm *LoopManager) Has(id string) bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return lm.loops.Has(id)
}

// Count returns the number of loops managed by the LoopManager.
// It locks the manager to ensure thread safety while counting.
func (lm *LoopManager) Count() int {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return lm.loops.Len()
}

// ALLID returns a slice of all loop IDs managed by the LoopManager.
// It locks the manager to ensure thread safety while accessing the IDs.
func (lm *LoopManager) ALLID() []string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	ids := make([]string, 0, lm.loops.Len())
	lm.loops.Iterator(func(s string, _ *Loop) bool {
		ids = append(ids, s)
		return true
	})

	return ids
}

// Stop stops all loops and waits for them to finish.
func (lm *LoopManager) Stop() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Stop all loops
	lm.loops.Iterator(func(s string, l *Loop) bool {
		l.Stop()
		return true
	})

	// Clear the loops map
	lm.loops.Clear()
	// Wait for all loops to finish
	lm.wg.Wait()
}
