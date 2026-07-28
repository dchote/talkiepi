package audio

import "sync"

// Mixer sums multiple int16 PCM streams into a single output frame.
type Mixer struct {
	mu     sync.Mutex
	frame  int
	accum  []int32
	active int
}

// NewMixer creates a mixer for frame-sized buffers.
func NewMixer(frameSamples int) *Mixer {
	if frameSamples < 1 {
		frameSamples = 960
	}
	return &Mixer{frame: frameSamples, accum: make([]int32, frameSamples)}
}

// Add mixes one source frame into the accumulator.
func (m *Mixer) Add(samples []int16) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := m.frame
	if len(samples) < n {
		n = len(samples)
	}
	for i := 0; i < n; i++ {
		m.accum[i] += int32(samples[i])
	}
	m.active++
}

// Render returns the mixed frame clamped to int16 and clears the accumulator.
func (m *Mixer) Render() []int16 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]int16, m.frame)
	for i := 0; i < m.frame; i++ {
		v := m.accum[i]
		m.accum[i] = 0
		if v > 32767 {
			v = 32767
		} else if v < -32768 {
			v = -32768
		}
		out[i] = int16(v)
	}
	m.active = 0
	return out
}
