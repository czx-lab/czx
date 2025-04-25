package room

import (
	"fmt"
	"testing"
	"time"

	"github.com/czx-lab/czx/frame"
)

type roomprocessor struct{}

// Join implements RoomProcessor.
func (r *roomprocessor) Join(playerID string) error {
	fmt.Println("roomprocessor Join", playerID)
	return nil
}

// Leave implements RoomProcessor.
func (r *roomprocessor) Leave(playerID string) error {
	fmt.Println("roomprocessor Leave", playerID)
	return nil
}

var _ RoomProcessor = (*roomprocessor)(nil)

func TestRoom(t *testing.T) {
	t.Run("room test", func(t *testing.T) {
		opt := NewOption(
			WithRoomID("1"),
			WithMaxPlayer(5),
			WithMaxBufferSize(4096),
			WithFrequency(30*time.Millisecond),
			// WithTimeout(10*time.Second),
			WithHeartbeat(3*time.Second),
		)
		room := NewRoom(&roomprocessor{}, opt)
		if room.ID() != "1" {
			t.Errorf("expected room id 1, got %v", room.ID())
		}

		go func() {
			if err := room.Start(); err != nil {
				t.Errorf("room start err %v", err)
			}
		}()

		time.AfterFunc(2*time.Second, func() {
			if err := room.WriteMessage(frame.Message{
				PlayerID: "1",
				Data:     []byte{'m', 'e'},
			}); err != nil {
				t.Error("write message err", err)
			}
		})

		time.AfterFunc(10*time.Second, func() {
			room.WriteMessage(frame.Message{
				PlayerID: "2",
				Data:     []byte{'m', 'e', '2'},
			})

			room.Join("10")
			room.Join("11")
			room.Join("12")
			time.AfterFunc(time.Second, func() {
				if err := room.Leave("10"); err != nil {
					t.Errorf("leave player 10 err %v", err)
				}
			})
		})
		<-(chan any)(nil)
	})
}

func TestRoomManager(t *testing.T) {
	t.Run("room manager test", func(t *testing.T) {
		rm := NewRoomManager()

		opt := NewOption(
			WithRoomID("1"),
			WithMaxPlayer(5),
			WithMaxBufferSize(4096),
			WithFrequency(30*time.Millisecond),
			// WithTimeout(10*time.Second),
			WithHeartbeat(3*time.Second),
		)
		opt1 := NewOption(
			WithRoomID("2"),
			WithMaxPlayer(5),
			WithMaxBufferSize(4096),
			WithFrequency(30*time.Millisecond),
			// WithTimeout(10*time.Second),
			WithHeartbeat(3*time.Second),
		)
		room := NewRoom(&roomprocessor{}, opt)
		room1 := NewRoom(&roomprocessor{}, opt1)

		if err := rm.Add(room); err != nil {
			t.Errorf("add room err %v", err)
		}
		if err := rm.Add(room1); err != nil {
			t.Errorf("add room1 err %v", err)
		}

		fmt.Println(111111)

		room.WriteMessage(frame.Message{
			PlayerID: "2",
			Data:     []byte{'m', 'e', '2'},
		})

		room.Join("10")
		room.Join("11")
		room.Join("12")

		room.Leave("10")
		rm.Remove("2")

		time.Sleep(10 * time.Second)
		rm.Stop()

		for {

		}
	})
}
