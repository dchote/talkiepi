package audio

import (
	"sync"
)

// Ring is a thread-safe PCM sample ring buffer (int16).
type Ring struct {
	mu   sync.Mutex
	buf  []int16
	r, w int
	size int
	full bool
}

// NewRing creates a ring with capacity samples.
func NewRing(capacity int) *Ring {
	if capacity < 1 {
		capacity = 4800
	}
	return &Ring{buf: make([]int16, capacity), size: capacity}
}

// Write appends samples, dropping oldest on overflow. Returns samples written.
func (r *Ring) Write(samples []int16) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, s := range samples {
		if r.full {
			r.r = (r.r + 1) % r.size
		}
		r.buf[r.w] = s
		r.w = (r.w + 1) % r.size
		r.full = r.w == r.r
		n++
	}
	return n
}

// Read copies up to len(dst) samples. Returns count read.
func (r *Ring) Read(dst []int16) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for n < len(dst) && (r.full || r.r != r.w) {
		dst[n] = r.buf[r.r]
		r.r = (r.r + 1) % r.size
		r.full = false
		n++
	}
	return n
}

// Len returns available samples.
func (r *Ring) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.full {
		return r.size
	}
	if r.w >= r.r {
		return r.w - r.r
	}
	return r.size - r.r + r.w
}

// Clear empties the ring.
func (r *Ring) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.r, r.w = 0, 0
	r.full = false
}

// PeekRMS returns RMS of available samples without consuming.
func (r *Ring) PeekRMS() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := r.lenUnlocked()
	if n == 0 {
		return 0
	}
	var sum float64
	idx := r.r
	for i := 0; i < n; i++ {
		v := float64(r.buf[idx])
		sum += v * v
		idx = (idx + 1) % r.size
	}
	return sqrt(sum / float64(n))
}

func (r *Ring) lenUnlocked() int {
	if r.full {
		return r.size
	}
	if r.w >= r.r {
		return r.w - r.r
	}
	return r.size - r.r + r.w
}

func sqrt(x float64) float64 {
	// Local tiny sqrt to avoid importing math in hot path tests; math is fine too.
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 8; i++ {
		z = 0.5 * (z + x/z)
	}
	return z
}
