package actor

import "github.com/google/uuid"

// PID represents the Process Identifier of an actor.
type PID struct {
	ID string
}

// NewPID creates a new PID with the given ID.
func NewPID(id string) *PID {
	return &PID{ID: id}
}

// GetID returns the ID of the PID.
func (p *PID) GetID() string {
	return p.ID
}

// DefaultPID generates a PID with a unique UUID as its ID.
func DefaultPID() *PID {
	return &PID{ID: uuid.New().String()}
}
