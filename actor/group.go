package actor

import "context"

// Group manages a collection of actors.
type Group struct {
	actors []Service
	ctx    context.Context
}

// NewGroup creates a new Group with the provided actors.
func NewGroup(ctx context.Context, actors ...Service) *Group {
	return &Group{
		actors: actors,
		ctx:    ctx,
	}
}

// Start starts all actors in the group.
func (g *Group) Start() {
	for _, actor := range g.actors {
		actor.Start()
	}
}

// Stop stops all actors in the group.
func (g *Group) Stop() {
	for _, actor := range g.actors {
		actor.Stop()
	}
}
