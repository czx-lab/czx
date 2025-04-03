package room

import (
	"fmt"
	"testing"
	"time"
)

type msgprocessor struct{}

// Close implements MessageProcessor.
func (m *msgprocessor) Close() error {
	fmt.Println("msgprocessor closed")
	return nil
}

// HandleIdle implements MessageProcessor.
func (m *msgprocessor) HandleIdle() error {
	fmt.Println("msgprocessor HandleIdle")
	return nil
}

// Process implements MessageProcessor.
func (m *msgprocessor) Process(msg Message) error {
	fmt.Println("msgprocessor Process", msg.PlayerID, string(msg.Msg))
	return nil
}

var _ MessageProcessor = (*msgprocessor)(nil)

type roomprocessor struct{}

// Join implements RoomProcessor.
func (r *roomprocessor) Join(playerID uint64) error {
	fmt.Println("roomprocessor Join", playerID)
	return nil
}

// Leave implements RoomProcessor.
func (r *roomprocessor) Leave(playerID uint64) error {
	fmt.Println("roomprocessor Leave", playerID)
	return nil
}

var _ RoomProcessor = (*roomprocessor)(nil)

func TestRoom(t *testing.T) {
	t.Run("room test", func(t *testing.T) {
		opt := NewOption(
			WithRoomID(1),
			WithMaxPlayer(5),
			WithMaxBufferSize(4096),
			WithFrequency(30*time.Millisecond),
			// WithTimeout(10*time.Second),
			WithHeartbeat(3*time.Second),
		)
		room := NewRoom(&roomprocessor{}, &msgprocessor{}, opt)
		if room.ID() != 1 {
			t.Errorf("expected room id 1, got %d", room.ID())
		}

		if err := room.Start(); err != nil {
			t.Errorf("room start err %v", err)
		}

		room.WriteMessage(Message{
			PlayerID: 1,
			Msg:      []byte{'m', 'e'},
		})

		time.AfterFunc(10*time.Second, func() {
			room.WriteMessage(Message{
				PlayerID: 2,
				Msg:      []byte{'m', 'e', '2'},
			})

			room.Join(10)
			room.Join(11)
			room.Join(12)
			time.AfterFunc(time.Second, func() {
				room.Leave(10)
			})
		})
		for {

		}
	})
}

func TestRoomManager(t *testing.T) {
	t.Run("room manager test", func(t *testing.T) {
		rm := NewRoomManager()

		opt := NewOption(
			WithRoomID(1),
			WithMaxPlayer(5),
			WithMaxBufferSize(4096),
			WithFrequency(30*time.Millisecond),
			// WithTimeout(10*time.Second),
			WithHeartbeat(3*time.Second),
		)
		room := NewRoom(&roomprocessor{}, &msgprocessor{}, opt)

		if err := rm.AddRoom(room); err != nil {
			t.Errorf("add room err %v", err)
		}

		fmt.Println(111111)

		room.WriteMessage(Message{
			PlayerID: 2,
			Msg:      []byte{'m', 'e', '2'},
		})

		room.Join(10)
		room.Join(11)
		room.Join(12)
		time.AfterFunc(time.Second, func() {
			room.Leave(10)

			// rm.Stop()
		})

		rm.Stop()

		for {

		}
	})
}
