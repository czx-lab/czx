package tcp

import (
	"io"
	"sync"
)

type GBuffer struct {
	mu     sync.Mutex
	buffer []byte
}

var (
	_ io.Reader = (*GBuffer)(nil)
	_ io.Writer = (*GBuffer)(nil)
)

func NewGBuffer() *GBuffer {
	return &GBuffer{
		buffer: make([]byte, 0),
	}
}

// Reset clears the buffer.
func (gb *GBuffer) Reset() {
	gb.mu.Lock()
	defer gb.mu.Unlock()

	gb.buffer = gb.buffer[:0]
}

// Write implements io.Writer.
func (gb *GBuffer) Write(p []byte) (n int, err error) {
	gb.mu.Lock()
	defer gb.mu.Unlock()

	gb.buffer = append(gb.buffer, p...)
	return len(p), nil
}

// Read implements io.Reader.
func (gb *GBuffer) Read(p []byte) (n int, err error) {
	gb.mu.Lock()
	defer gb.mu.Unlock()

	if len(gb.buffer) == 0 {
		return 0, io.EOF
	}
	n = copy(p, gb.buffer)
	gb.buffer = gb.buffer[n:]
	if n < len(p) {
		err = io.EOF
	}
	return
}
