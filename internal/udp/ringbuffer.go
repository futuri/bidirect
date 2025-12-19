package udp

import (
	"sync"
	"sync/atomic"
)

type Frame struct {
	Data   []byte
	Width  int
	Height int
}

type RingBuffer struct {
	frames    [3]*Frame
	writeIdx  uint64
	readIdx   uint64
	mu        sync.RWMutex
	hasFrames atomic.Bool
}

func NewRingBuffer() *RingBuffer {
	rb := &RingBuffer{}
	for i := range rb.frames {
		rb.frames[i] = &Frame{
			Data: make([]byte, 0, 4*1024*1024), // 4MB pre-allocated
		}
	}
	return rb
}

func (rb *RingBuffer) Write(data []byte, width, height int) {
	rb.mu.Lock()
	idx := rb.writeIdx % 3
	frame := rb.frames[idx]

	if cap(frame.Data) < len(data) {
		frame.Data = make([]byte, len(data))
	} else {
		frame.Data = frame.Data[:len(data)]
	}
	copy(frame.Data, data)
	frame.Width = width
	frame.Height = height

	rb.writeIdx++
	rb.hasFrames.Store(true)
	rb.mu.Unlock()
}

func (rb *RingBuffer) ReadLatest() (*Frame, bool) {
	if !rb.hasFrames.Load() {
		return nil, false
	}

	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.writeIdx == 0 {
		return nil, false
	}

	idx := (rb.writeIdx - 1) % 3
	frame := rb.frames[idx]

	if len(frame.Data) == 0 {
		return nil, false
	}

	return frame, true
}

func (rb *RingBuffer) HasFrames() bool {
	return rb.hasFrames.Load()
}
