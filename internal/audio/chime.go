package audio

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
)

//go:embed sounds/communicator.wav
var communicatorWAV []byte

// Matches esp32-mumble CHIME_VOLUME_SCALE.
const chimeVolumeScale = 0.25

var (
	chimeOnce   sync.Once
	chimeSource []int16 // native PCM from WAV (16 kHz mono)
	chimeRate   int
	chimeErr    error
	chimeCache  sync.Map // sampleRate -> []int16
)

// loadCommunicatorChime parses the embedded esp32-mumble communicator.wav (16-bit mono PCM).
func loadCommunicatorChime() {
	chimeOnce.Do(func() {
		pcm, rate, err := parseWAVPCM16Mono(communicatorWAV)
		if err != nil {
			chimeErr = err
			return
		}
		chimeSource = pcm
		chimeRate = rate
	})
}

// CommunicatorChimePCM returns the esp32-mumble communicator chime resampled to sampleRate
// with the same volume scale used on device. Open and close use the same asset (esp32 behavior).
func CommunicatorChimePCM(sampleRate int) []int16 {
	loadCommunicatorChime()
	if chimeErr != nil || len(chimeSource) == 0 {
		return nil
	}
	if sampleRate <= 0 {
		sampleRate = chimeRate
	}
	if v, ok := chimeCache.Load(sampleRate); ok {
		src := v.([]int16)
		out := make([]int16, len(src))
		copy(out, src)
		return out
	}
	pcm := resampleLinear(chimeSource, chimeRate, sampleRate)
	ApplyLinearGain(pcm, chimeVolumeScale)
	chimeCache.Store(sampleRate, pcm)
	out := make([]int16, len(pcm))
	copy(out, pcm)
	return out
}

// OpenChimePCM returns the communicator open chime (esp32 communicator.wav).
func OpenChimePCM(sampleRate int) []int16 {
	return CommunicatorChimePCM(sampleRate)
}

// CloseChimePCM returns the communicator close chime (same WAV as open; esp32 behavior).
func CloseChimePCM(sampleRate int) []int16 {
	return CommunicatorChimePCM(sampleRate)
}

// GenerateTone builds a mono int16 sine tone (fallback / tests).
func GenerateTone(sampleRate, durationMs int, hz float64, amplitude float64) []int16 {
	if sampleRate <= 0 || durationMs <= 0 || hz <= 0 {
		return nil
	}
	if amplitude <= 0 {
		amplitude = 0.25
	}
	n := sampleRate * durationMs / 1000
	out := make([]int16, n)
	for i := 0; i < n; i++ {
		env := 1.0
		fade := sampleRate / 200 // 5 ms
		if fade < 1 {
			fade = 1
		}
		if i < fade {
			env = float64(i) / float64(fade)
		} else if i > n-fade {
			env = float64(n-i) / float64(fade)
		}
		v := math.Sin(2 * math.Pi * hz * float64(i) / float64(sampleRate))
		out[i] = int16(v * amplitude * env * 32767)
	}
	return out
}

func parseWAVPCM16Mono(data []byte) ([]int16, int, error) {
	if len(data) < 44 {
		return nil, 0, fmt.Errorf("audio: wav too small")
	}
	if !bytes.Equal(data[0:4], []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WAVE")) {
		return nil, 0, fmt.Errorf("audio: not a RIFF WAVE")
	}
	pos := 12
	var (
		pcm        []byte
		sampleRate int
		channels   int
		bits       int
	)
	for pos+8 <= len(data) {
		chunkID := string(data[pos : pos+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		pos += 8
		if pos+chunkSize > len(data) {
			break
		}
		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return nil, 0, fmt.Errorf("audio: fmt chunk too small")
			}
			audioFormat := binary.LittleEndian.Uint16(data[pos : pos+2])
			channels = int(binary.LittleEndian.Uint16(data[pos+2 : pos+4]))
			sampleRate = int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
			bits = int(binary.LittleEndian.Uint16(data[pos+14 : pos+16]))
			if audioFormat != 1 {
				return nil, 0, fmt.Errorf("audio: unsupported wav format %d", audioFormat)
			}
		case "data":
			pcm = data[pos : pos+chunkSize]
		}
		pos += chunkSize
		if chunkSize%2 == 1 {
			pos++ // pad byte
		}
	}
	if pcm == nil {
		return nil, 0, fmt.Errorf("audio: wav missing data chunk")
	}
	if channels != 1 || bits != 16 {
		return nil, 0, fmt.Errorf("audio: expected 16-bit mono wav, got %d-bit %d ch", bits, channels)
	}
	if sampleRate <= 0 {
		return nil, 0, fmt.Errorf("audio: invalid sample rate")
	}
	if len(pcm)%2 != 0 {
		return nil, 0, fmt.Errorf("audio: odd pcm length")
	}
	samples := make([]int16, len(pcm)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2 : i*2+2]))
	}
	return samples, sampleRate, nil
}

func resampleLinear(src []int16, fromRate, toRate int) []int16 {
	if fromRate <= 0 || toRate <= 0 || len(src) == 0 {
		return nil
	}
	if fromRate == toRate {
		out := make([]int16, len(src))
		copy(out, src)
		return out
	}
	n := int(float64(len(src)) * float64(toRate) / float64(fromRate))
	if n < 1 {
		n = 1
	}
	out := make([]int16, n)
	for i := 0; i < n; i++ {
		srcPos := float64(i) * float64(fromRate) / float64(toRate)
		i0 := int(srcPos)
		if i0 >= len(src)-1 {
			out[i] = src[len(src)-1]
			continue
		}
		frac := srcPos - float64(i0)
		a := float64(src[i0])
		b := float64(src[i0+1])
		out[i] = int16(a + (b-a)*frac)
	}
	return out
}
