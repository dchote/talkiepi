// Package codec defines Opus encoding profiles for talkiepi voice.
package codec

import "fmt"

// Profile describes PCM/Opus encoding parameters.
type Profile struct {
	SampleRate int
	Channels   int
	FrameMs    int
	Bitrate    int
	Complexity int
}

// Default returns 48 kHz mono 20 ms @ 40 kbps, complexity 5.
func Default() Profile {
	return Profile{
		SampleRate: 48000,
		Channels:   1,
		FrameMs:    20,
		Bitrate:    40000,
		Complexity: 5,
	}
}

// Esp32Compatible returns 16 kHz mono 20 ms @ 16 kbps, complexity 1.
func Esp32Compatible() Profile {
	return Profile{
		SampleRate: 16000,
		Channels:   1,
		FrameMs:    20,
		Bitrate:    16000,
		Complexity: 1,
	}
}

// FrameSamples returns samples per channel per frame.
func (p Profile) FrameSamples() int {
	if p.SampleRate <= 0 || p.FrameMs <= 0 {
		return 0
	}
	return p.SampleRate * p.FrameMs / 1000
}

// Validate checks profile fields for sane Opus values.
func (p Profile) Validate() error {
	switch p.SampleRate {
	case 8000, 12000, 16000, 24000, 48000:
	default:
		return fmt.Errorf("codec: unsupported sample rate %d", p.SampleRate)
	}
	if p.Channels != 1 && p.Channels != 2 {
		return fmt.Errorf("codec: channels must be 1 or 2, got %d", p.Channels)
	}
	switch p.FrameMs {
	case 2, 5, 10, 20, 40, 60:
	default:
		return fmt.Errorf("codec: unsupported frame_ms %d", p.FrameMs)
	}
	if p.Bitrate < 500 || p.Bitrate > 512000 {
		return fmt.Errorf("codec: bitrate out of range: %d", p.Bitrate)
	}
	if p.Complexity < 0 || p.Complexity > 10 {
		return fmt.Errorf("codec: complexity out of range: %d", p.Complexity)
	}
	if p.FrameSamples() <= 0 {
		return fmt.Errorf("codec: invalid frame samples")
	}
	return nil
}
