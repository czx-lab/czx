package frame

import (
	"testing"
)

// recorder is a FrameProcessor that records executed frames and closes.
type recorder struct {
	frames []uint64
	closed int
}

func (r *recorder) Process(frame Frame)                 { r.frames = append(r.frames, frame.FrameID) }
func (r *recorder) Resend(playerId string, frameId int) {}
func (r *recorder) OnClose()                            { r.closed++ }

// newTestLoop returns a loop with two players, driven manually through
// exec() — no ticker, fully deterministic.
func newTestLoop(t *testing.T, delay uint) (*FrameLoop, *recorder) {
	t.Helper()

	rec := &recorder{}
	loop := NewFrameLoop(FrameConf{Frequency: 10, InputDelay: delay}).WithProc(rec)
	loop.RegisterPlayer("p1")
	loop.RegisterPlayer("p2")

	return loop, rec
}

// continuous reports whether executed frame IDs advance by exactly one.
func continuous(frames []uint64) bool {
	for i := 1; i < len(frames); i++ {
		if frames[i] != frames[i-1]+1 {
			return false
		}
	}

	return true
}

func TestExecWarmupThenSteadyPace(t *testing.T) {
	loop, rec := newTestLoop(t, 2)

	for i := 0; i < 30; i++ {
		loop.exec()
	}

	// With InputDelay 2 the first frame executes on tick 3 and every tick
	// after that executes exactly one frame.
	if len(rec.frames) != 28 || rec.frames[0] != 1 || rec.frames[27] != 28 {
		t.Fatalf("expected frames 1..28, got %v", rec.frames)
	}
	if !continuous(rec.frames) {
		t.Fatalf("frames not sequential: %v", rec.frames)
	}
}

func TestResetResumesAtNextFrame(t *testing.T) {
	loop, rec := newTestLoop(t, 2)

	for i := 0; i < 10; i++ {
		loop.exec()
	}
	rec.frames = nil

	loop.Reset(100)

	// frameId is 100 and execution resumes at 101, held by the delay window
	// until tick 103; frames 99 and 100 must not be re-run with empty inputs.
	for i := 0; i < 5; i++ {
		loop.exec()
	}

	if len(rec.frames) == 0 || rec.frames[0] != 101 {
		t.Fatalf("expected execution to resume at frame 101, got %v", rec.frames)
	}
	if !continuous(rec.frames) {
		t.Fatalf("frames not sequential: %v", rec.frames)
	}
}

func TestWriteRejectsPastFrame(t *testing.T) {
	loop, _ := newTestLoop(t, 2)

	for i := 0; i < 10; i++ {
		loop.exec()
	}

	// Frames 1..8 executed, so 8 is past and 9 is still eligible.
	if err := loop.Write(Message{PlayerID: "p1", FrameID: 8}); err == nil {
		t.Fatal("expected past-frame message to be rejected")
	}
	if err := loop.Write(Message{PlayerID: "p1", FrameID: 9}); err != nil {
		t.Fatalf("expected frame 9 to be accepted: %v", err)
	}
}

func TestStopDropsFutureFrames(t *testing.T) {
	loop, rec := newTestLoop(t, 2)

	for i := 0; i < 10; i++ {
		loop.exec()
	}

	// Frames 1..8 executed; inputs for frames beyond the due one are future
	// frames and must be discarded on stop, not executed with empty fills.
	for frame := uint64(9); frame <= 15; frame++ {
		if err := loop.Write(Message{PlayerID: "p1", FrameID: frame}); err != nil {
			t.Fatalf("expected frame %d to be accepted: %v", frame, err)
		}
	}

	loop.Stop()

	if len(rec.frames) != 8 {
		t.Fatalf("expected no frames executed after stop, got %v", rec.frames)
	}
	if rec.closed != 1 {
		t.Fatalf("expected OnClose once, got %d", rec.closed)
	}
}
