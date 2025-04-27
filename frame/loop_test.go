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
		// loop.WithEmptyHandler(func() {
		// 	fmt.Println("Empty handler called")
		// })

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
