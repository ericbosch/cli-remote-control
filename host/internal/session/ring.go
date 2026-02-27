package session

import (
	"sync"
)

// RingBuffer keeps the last size bytes written.
type RingBuffer struct {
	mu   sync.RWMutex
	buf  []byte
	size int
}

// NewRingBuffer creates a ring buffer of size bytes.
func NewRingBuffer(size int) *RingBuffer {
	if size <= 0 {
		size = 65536
	}
	return &RingBuffer{size: size}
}

// Write appends data; keeps only the last size bytes.
func (r *RingBuffer) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n = len(p)
	r.buf = append(r.buf, p...)
	if len(r.buf) > r.size {
		r.buf = r.buf[len(r.buf)-r.size:]
	}
	return n, nil
}

// Bytes returns a copy of the buffer contents.
func (r *RingBuffer) Bytes() []byte {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.buf) == 0 {
		return nil
	}
	out := make([]byte, len(r.buf))
	copy(out, r.buf)
	return out
}

// Len returns the number of bytes currently in the buffer.
func (r *RingBuffer) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.buf)
}

// Snapshot returns the last up-to limit bytes (for replay).
func (r *RingBuffer) Snapshot(limit int) []byte {
	b := r.Bytes()
	if limit <= 0 || len(b) <= limit {
		return b
	}
	return b[len(b)-limit:]
}
