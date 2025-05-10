package frame

import (
	"fmt"
	"testing"
	"time"
)

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
func (n *normalProcessorMock) Process(message Message) {
	fmt.Printf("normalProcessorMock process message: %+v\n", message)
}

var _ NormalProcessor = (*normalProcessorMock)(nil)

type frameProcessorMock struct{}

// Resend implements FrameProcessor.
func (f *frameProcessorMock) Resend(playerId string, sequenceID int) {
	panic("unimplemented")
}

// Close implements FrameProcessor.
func (f *frameProcessorMock) Close() {
	fmt.Println("frameProcessorMock closed")

}

// HandleIdle implements FrameProcessor.
func (f *frameProcessorMock) HandleIdle() {
	fmt.Println("frameProcessorMock handle idle")
}

// Process implements FrameProcessor.
func (f *frameProcessorMock) Process(frame Frame) {
	fmt.Printf("frameProcessorMock process frame: %+v\n", frame)
}

var _ FrameProcessor = (*frameProcessorMock)(nil)

func TestLoop(t *testing.T) {
	t.Run("TestNormalLoop", func(t *testing.T) {
		// Create a new loop with default configuration
		loop, _ := NewLoop(LoopConf{
			Frequency:          60,
			HeartbeatFrequency: 5,
			LoopType:           LoopTypeNormal,
		})
		if err := loop.WithNormalProc(&normalProcessorMock{}); err != nil {
			t.Fatalf("Failed to set normal processor: %v", err)
		}
		loop.WithEmptyHandler(&EmptyProcessor{
			Handler: func() {
				fmt.Println("Empty handler called")
			},
			Frequency: 3 * time.Second,
		})

		// Start the loop
		go loop.Start()

		// Simulate sending messages to the loop
		for i := range 10 {
			loop.Receive(Message{
				PlayerID:  fmt.Sprintf("Player%d", i),
				Data:      []byte(fmt.Sprintf("Input from Player%d", i%3)),
				Timestamp: time.Now(),
			})
		}

		time.Sleep(20 * time.Second)

		// Stop the loop after some time
		loop.Stop()
	})

	t.Run("TestSyncLoop", func(t *testing.T) {
		// Create a new loop with default configuration
		loop, _ := NewLoop(LoopConf{
			Frequency:          60,
			HeartbeatFrequency: 5,
			LoopType:           LoopTypeSync,
		})
		if err := loop.WithFrameProc(&frameProcessorMock{}); err != nil {
			t.Fatalf("Failed to set normal processor: %v", err)
		}

		// Start the loop
		go loop.Start()

		// Simulate sending messages to the loop
		for i := range 10 {
			loop.Receive(Message{
				PlayerID:  fmt.Sprintf("Player%d", i),
				Data:      []byte(fmt.Sprintf("Input from Player%d", i%3)),
				Timestamp: time.Now(),
			})
		}

		time.Sleep(40 * time.Millisecond)

		// Simulate sending messages to the loop
		for i := range 10 {
			loop.Receive(Message{
				PlayerID:  fmt.Sprintf("Player_next%d", i),
				Data:      []byte(fmt.Sprintf("Input next from Player%d", i%3)),
				Timestamp: time.Now(),
			})
		}

		time.Sleep(20 * time.Second)

		// Stop the loop after some time
		loop.Stop()
	})
}

func TestMessage(t *testing.T) {
	// Create a new message
	message := Message{
		PlayerID:  "Player1",
		Data:      []byte("Input data"),
		Timestamp: time.Now(),
	}

	// Serialize the message
	data, err := message.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize message: %v", err)
	}

	// Deserialize the message
	var deserializedMessage Message
	if err := deserializedMessage.Deserialize(data); err != nil {
		t.Fatalf("Failed to deserialize message: %v", err)
	}

	// Check if the original and deserialized messages are equal
	if message.PlayerID != deserializedMessage.PlayerID || string(message.Data) != string(deserializedMessage.Data) {
		t.Errorf("Original and deserialized messages are not equal: %+v vs %+v", message, deserializedMessage)
	}
}

func TestFrame(t *testing.T) {
	// Create a new frame
	frame := Frame{
		FrameID: 1,
		Inputs: map[string]Message{
			"Player1": {
				PlayerID:  "Player1",
				Data:      []byte("Input data"),
				Timestamp: time.Now(),
			},
			"Player2": {
				PlayerID:  "Player2",
				Data:      []byte("Input data"),
				Timestamp: time.Now(),
			},
		},
	}

	// Serialize the frame
	data, err := frame.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize frame: %v", err)
	}

	// Deserialize the frame
	var deserializedFrame Frame
	if err := deserializedFrame.Deserialize(data); err != nil {
		t.Fatalf("Failed to deserialize frame: %v", err)
	}

	// Check if the original and deserialized frames are equal
	if frame.FrameID != deserializedFrame.FrameID || len(frame.Inputs) != len(deserializedFrame.Inputs) {
		t.Errorf("Original and deserialized frames are not equal: %+v vs %+v", frame, deserializedFrame)
	}
	for playerID, message := range frame.Inputs {
		deserializedMessage, ok := deserializedFrame.Inputs[playerID]
		if !ok || message.PlayerID != deserializedMessage.PlayerID || string(message.Data) != string(deserializedMessage.Data) {
			t.Errorf("Original and deserialized messages are not equal: %+v vs %+v", message, deserializedMessage)
		}
	}
	// Check if the original and deserialized frames are equal
	if frame.FrameID != deserializedFrame.FrameID || len(frame.Inputs) != len(deserializedFrame.Inputs) {
		t.Errorf("Original and deserialized frames are not equal: %+v vs %+v", frame, deserializedFrame)
	}
}
