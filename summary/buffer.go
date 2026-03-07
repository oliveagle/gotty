package summary

import (
	"sync"
)

// RingBuffer is a circular buffer for storing terminal output
type RingBuffer struct {
	data  []byte
	size  int
	start int
	end   int
	full  bool
	mu    sync.Mutex
}

// NewRingBuffer creates a new ring buffer with the given size
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]byte, size),
		size: size,
	}
}

// Write adds data to the buffer, overwriting old data if necessary
func (rb *RingBuffer) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	rb.mu.Lock()
	defer rb.mu.Unlock()

	// If data is larger than buffer, just keep the last part
	if len(data) >= rb.size {
		copy(rb.data, data[len(data)-rb.size:])
		rb.start = 0
		rb.end = rb.size
		rb.full = true
		return len(data), nil
	}

	// Write data
	for _, b := range data {
		rb.data[rb.end] = b
		rb.end++
		if rb.end >= rb.size {
			rb.end = 0
		}
		if rb.end == rb.start {
			rb.full = true
			rb.start++
			if rb.start >= rb.size {
				rb.start = 0
			}
		}
	}

	return len(data), nil
}

// Bytes returns the current buffer contents
func (rb *RingBuffer) Bytes() []byte {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.start == rb.end && !rb.full {
		return nil
	}

	var result []byte
	if rb.start < rb.end {
		result = make([]byte, rb.end-rb.start)
		copy(result, rb.data[rb.start:rb.end])
	} else {
		result = make([]byte, rb.size-rb.start+rb.end)
		copy(result, rb.data[rb.start:])
		copy(result[rb.size-rb.start:], rb.data[:rb.end])
	}

	return result
}

// Reset clears the buffer
func (rb *RingBuffer) Reset() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.start = 0
	rb.end = 0
	rb.full = false
}

// Used returns the number of bytes currently in the buffer
func (rb *RingBuffer) Used() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.start == rb.end && !rb.full {
		return 0
	}
	if rb.start < rb.end {
		return rb.end - rb.start
	}
	return rb.size - rb.start + rb.end
}

// Size returns the total buffer size
func (rb *RingBuffer) Size() int {
	return rb.size
}
