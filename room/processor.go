package room

type RoomProcessor interface {
	// Join is called when a player joins the room
	Join(playerID uint64) error
	// Leave is called when a player leaves the room
	Leave(playerID uint64) error
}

type MessageProcessor interface {
	// ProcessMessage is called when a message is received from a player
	// It is responsible for processing the message and sending it to the appropriate player
	Message(playerID uint64, msg []byte) error
	// Process is called when a message is received from the room
	// It is responsible for processing the message and sending it to the appropriate player
	Process(msg []byte) error
	// HandleIdle is called when the loop is idle
	// It is responsible for handling idle timeouts and other idle tasks
	HandleIdle() error

	Close() error
	// Stop is called when the loop is stopped
}
