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

type normalProcessorMock struct{}

// Close implements NormalProcessor.
func (n *normalProcessorMock) Close() {
	fmt.Println("normalProcessorMock closed")
}

// HandleIdle implements NormalProcessor.
func (n *normalProcessorMock) HandleIdle() {
	fmt.Println("normalProcessorMock handle idle")

}

// Process implements NormalProcessor.
func (n *normalProcessorMock) Process(message frame.Message) {
	fmt.Printf("normalProcessorMock process message: %+v\n", message)
}

var _ frame.NormalProcessor = (*normalProcessorMock)(nil)

func TestRoom(t *testing.T) {
	t.Run("room test", func(t *testing.T) {
		room := NewRoom(&roomprocessor{}, RoomConf{})
		loop := frame.NewLoop(frame.LoopConf{})
		loop.WithNormalProc(&normalProcessorMock{})
		room.WithLoop(loop)
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

				room.Stop()
			})
		})

		for {
		}
	})
}

func TestRoomManager(t *testing.T) {
	t.Run("room manager test", func(t *testing.T) {
		rm := NewRoomManager()
		room := NewRoom(&roomprocessor{}, RoomConf{
			RoomID: "2",
		})

		loop := frame.NewLoop(frame.LoopConf{})
		loop.WithNormalProc(&normalProcessorMock{})
		room.WithLoop(loop)

		room1 := NewRoom(&roomprocessor{}, RoomConf{
			RoomID: "3",
		})

		if err := rm.Add(room); err != nil {
			t.Errorf("add room err %v", err)
		}
		if err := rm.Add(room1); err != nil {
			t.Errorf("add room1 err %v", err)
		}

		fmt.Println(111111)

		time.Sleep(2 * time.Second)
		if err := room.WriteMessage(frame.Message{
			PlayerID: "2",
			Data:     []byte{'m', 'e', '2'},
		}); err != nil {
			t.Fatalf("write message err %v", err)
		}

		room.Join("10")
		room.Join("11")
		room.Join("12")

		room.Leave("10")
		time.Sleep(100 * time.Second)
		fmt.Println("222222")
		rm.Remove("2")

		time.Sleep(10 * time.Second)
		rm.Stop()

		for {

		}
	})
}
