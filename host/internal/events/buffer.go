package events

import (
	"sync"
)

type Buffer struct {
	mu      sync.RWMutex
	cap     int
	buf     []SessionEvent
	start   int
	size    int
	nextSeq uint64
}

func NewBuffer(capacity int) *Buffer {
	if capacity <= 0 {
		capacity = 256
	}
	return &Buffer{
		cap: capacity,
		buf: make([]SessionEvent, capacity),
	}
}

func (b *Buffer) Append(ev SessionEvent) SessionEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextSeq++
	ev.Seq = b.nextSeq
	if ev.TsMS <= 0 {
		ev.TsMS = NowMS()
	}

	if b.size < b.cap {
		idx := (b.start + b.size) % b.cap
		b.buf[idx] = ev
		b.size++
		return ev
	}

	b.buf[b.start] = ev
	b.start = (b.start + 1) % b.cap
	return ev
}

func (b *Buffer) LastSeq() uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.nextSeq
}

func (b *Buffer) ReplayFromSeq(from uint64) []SessionEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.size == 0 {
		return nil
	}
	out := make([]SessionEvent, 0, b.size)
	for i := 0; i < b.size; i++ {
		ev := b.buf[(b.start+i)%b.cap]
		if ev.Seq > from {
			out = append(out, ev)
		}
	}
	return out
}

func (b *Buffer) ReplayLastN(n int) []SessionEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if n <= 0 || b.size == 0 {
		return nil
	}
	if n > b.size {
		n = b.size
	}
	out := make([]SessionEvent, 0, n)
	start := b.size - n
	for i := start; i < b.size; i++ {
		out = append(out, b.buf[(b.start+i)%b.cap])
	}
	return out
}
