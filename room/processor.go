package room

type RoomProcessor interface {
	// Join is called when a player joins the room
	Join(playerID string) error
	// Leave is called when a player leaves the room
	Leave(playerID string) error
}
