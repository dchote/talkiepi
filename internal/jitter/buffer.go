package jitter

import (
	"sync"
)

// Buffer is a sequence-indexed jitter buffer for PCM frames.
// Late packets (sequence < next) are dropped. Gaps produce zero frames
// (caller may replace with Opus PLC before Push of decoded PCM).
type Buffer struct {
	mu        sync.Mutex
	frames    map[int64][]int16
	capacity  int
	target    int
	next      int64
	started   bool
	primed    bool
	drops     uint64
	underruns uint64
}

// New creates a jitter buffer.
// capacity is max frames held; target is frames to accumulate before first pop.
func New(capacity, target int) *Buffer {
	if capacity < 1 {
		capacity = 16
	}
	if target < 1 {
		target = 2
	}
	if target > capacity {
		target = capacity
	}
	return &Buffer{
		frames:   make(map[int64][]int16, capacity),
		capacity: capacity,
		target:   target,
	}
}

// Push inserts a decoded PCM frame for sequence. Copies samples.
func (b *Buffer) Push(sequence int64, samples []int16) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.started {
		// Anchor to the lowest sequence seen during prebuffer so out-of-order
		// first packets (e.g. 1 then 0) still play in order.
		b.next = sequence
		b.started = true
	} else if !b.primed && sequence < b.next {
		// Still prebuffering: accept earlier sequences and rewind next.
		b.next = sequence
	}
	if sequence < b.next {
		b.drops++
		return
	}
	cp := make([]int16, len(samples))
	copy(cp, samples)
	b.frames[sequence] = cp

	// Evict oldest if over capacity.
	for len(b.frames) > b.capacity {
		oldest := b.next
		for seq := range b.frames {
			if seq < oldest {
				oldest = seq
			}
		}
		if oldest == b.next && b.frames[oldest] == nil {
			// Find actual oldest key
			first := true
			for seq := range b.frames {
				if first || seq < oldest {
					oldest = seq
					first = false
				}
			}
		}
		delete(b.frames, oldest)
		if oldest >= b.next {
			b.drops++
		}
	}

	if !b.primed && len(b.frames) >= b.target {
		b.primed = true
	}
}

// Pop returns the next frame if primed and available.
// ok=false means underrun (silence should be played).
func (b *Buffer) Pop() (samples []int16, ok bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.primed {
		return nil, false
	}
	frame, found := b.frames[b.next]
	if !found {
		b.underruns++
		b.next++
		return nil, false
	}
	delete(b.frames, b.next)
	b.next++
	return frame, true
}

// Reset clears state (e.g. on stream restart).
func (b *Buffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.frames = make(map[int64][]int16, b.capacity)
	b.next = 0
	b.started = false
	b.primed = false
}

// Stats returns drop and underrun counters.
func (b *Buffer) Stats() (drops, underruns uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.drops, b.underruns
}

// Len returns buffered frame count.
func (b *Buffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.frames)
}

// Primed reports whether the buffer has reached its target depth.
func (b *Buffer) Primed() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.primed
}
