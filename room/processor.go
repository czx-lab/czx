package room

type RoomProcessor interface {
	// Join is called when a player joins the room
	Join(playerID uint64) error
	// Leave is called when a player leaves the room
	Leave(playerID uint64) error
}

type MessageProcessor interface {
	// Process is called when a message is received from the room
	// It is responsible for processing the message and sending it to the appropriate player
	Process(msg Message) error
	// HandleIdle is called when the loop is idle
	// It is responsible for handling idle timeouts and other idle tasks
	HandleIdle() error
	// Stop is called when the loop is stopped
	Close() error
}
