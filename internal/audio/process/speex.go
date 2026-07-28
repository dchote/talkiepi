//go:build cgo && speexdsp

package process

/*
#cgo pkg-config: speexdsp
#include <speex/speex_echo.h>
#include <speex/speex_preprocess.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"math"
	"strings"
	"unsafe"
)

type speexProcessor struct {
	cfg        Config
	echo       *C.SpeexEchoState
	preprocess *C.SpeexPreprocessState
	frame      int
	erle       float64
	nearPow    float64
	errPow     float64

	// Fixed far-end delay line for echo_delay_ms alignment.
	delayBuf []int16
	delayPos int

	// One-pole high-pass state (when HighPass is enabled).
	hpX, hpY float64
}

func newSpeex(cfg Config) (Processor, error) {
	frame := cfg.FrameSamples
	tail := cfg.EchoTailMs * cfg.SampleRate / 1000
	if tail < frame {
		tail = frame * 5
	}
	filterLen := C.int(tail)

	echo := C.speex_echo_state_init(C.int(frame), filterLen)
	if echo == nil {
		return nil, fmt.Errorf("process: speex_echo_state_init failed")
	}
	sr := C.int(cfg.SampleRate)
	C.speex_echo_ctl(echo, C.SPEEX_ECHO_SET_SAMPLING_RATE, unsafe.Pointer(&sr))

	pp := C.speex_preprocess_state_init(C.int(frame), C.int(cfg.SampleRate))
	if pp == nil {
		C.speex_echo_state_destroy(echo)
		return nil, fmt.Errorf("process: speex_preprocess_state_init failed")
	}
	C.speex_preprocess_ctl(pp, C.SPEEX_PREPROCESS_SET_ECHO_STATE, unsafe.Pointer(echo))

	one := C.int(1)
	zero := C.int(0)
	if cfg.NoiseSuppress {
		C.speex_preprocess_ctl(pp, C.SPEEX_PREPROCESS_SET_DENOISE, unsafe.Pointer(&one))
		ns := C.int(nsLevelDB(cfg.NSLevel))
		C.speex_preprocess_ctl(pp, C.SPEEX_PREPROCESS_SET_NOISE_SUPPRESS, unsafe.Pointer(&ns))
	} else {
		C.speex_preprocess_ctl(pp, C.SPEEX_PREPROCESS_SET_DENOISE, unsafe.Pointer(&zero))
	}
	if cfg.AGC {
		C.speex_preprocess_ctl(pp, C.SPEEX_PREPROCESS_SET_AGC, unsafe.Pointer(&one))
		if cfg.AGCTargetDBFS != 0 {
			level := C.float(math.Max(1, math.Min(32768, math.Pow(10, cfg.AGCTargetDBFS/20)*32768)))
			C.speex_preprocess_ctl(pp, C.SPEEX_PREPROCESS_SET_AGC_LEVEL, unsafe.Pointer(&level))
		}
	} else {
		C.speex_preprocess_ctl(pp, C.SPEEX_PREPROCESS_SET_AGC, unsafe.Pointer(&zero))
	}
	if cfg.Dereverb {
		C.speex_preprocess_ctl(pp, C.SPEEX_PREPROCESS_SET_DEREVERB, unsafe.Pointer(&one))
	} else {
		C.speex_preprocess_ctl(pp, C.SPEEX_PREPROCESS_SET_DEREVERB, unsafe.Pointer(&zero))
	}

	delaySamples := cfg.EchoDelayMs * cfg.SampleRate / 1000
	if delaySamples < 0 {
		delaySamples = 0
	}

	return &speexProcessor{
		cfg:        cfg,
		echo:       echo,
		preprocess: pp,
		frame:      frame,
		delayBuf:   make([]int16, delaySamples+1),
	}, nil
}

func nsLevelDB(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "mild", "low":
		return -15
	case "aggressive", "high":
		return -45
	default:
		return -25
	}
}

func (p *speexProcessor) ProcessRender(far []int16) error {
	if !p.cfg.EchoCancel || p.echo == nil {
		return nil
	}
	if len(far) < p.frame {
		return nil
	}
	frame := far[:p.frame]
	if len(p.delayBuf) <= 1 {
		C.speex_echo_playback(p.echo, (*C.spx_int16_t)(unsafe.Pointer(&frame[0])))
		return nil
	}
	// Push frame through delay line sample-by-sample, feeding Speex the delayed path.
	delayed := make([]int16, p.frame)
	for i := 0; i < p.frame; i++ {
		delayed[i] = p.delayBuf[p.delayPos]
		p.delayBuf[p.delayPos] = frame[i]
		p.delayPos++
		if p.delayPos >= len(p.delayBuf) {
			p.delayPos = 0
		}
	}
	C.speex_echo_playback(p.echo, (*C.spx_int16_t)(unsafe.Pointer(&delayed[0])))
	return nil
}

func (p *speexProcessor) ProcessCapture(near []int16) ([]int16, error) {
	out := make([]int16, len(near))
	if len(near) < p.frame {
		copy(out, near)
		return out, nil
	}
	frame := near[:p.frame]
	if p.cfg.EchoCancel && p.echo != nil {
		var nearPow, errPow float64
		for _, s := range frame {
			v := float64(s)
			nearPow += v * v
		}
		C.speex_echo_capture(p.echo, (*C.spx_int16_t)(unsafe.Pointer(&frame[0])), (*C.spx_int16_t)(unsafe.Pointer(&out[0])))
		for _, s := range out[:p.frame] {
			v := float64(s)
			errPow += v * v
		}
		p.nearPow = 0.9*p.nearPow + 0.1*nearPow
		p.errPow = 0.9*p.errPow + 0.1*errPow
		if p.errPow > 1 && p.nearPow > p.errPow {
			p.erle = 10 * math.Log10(p.nearPow/p.errPow)
		}
	} else {
		copy(out, near)
	}
	if p.preprocess != nil {
		C.speex_preprocess_run(p.preprocess, (*C.spx_int16_t)(unsafe.Pointer(&out[0])))
	}
	if p.cfg.HighPass {
		p.applyHighPass(out[:p.frame])
	}
	if len(near) > p.frame {
		copy(out[p.frame:], near[p.frame:])
	}
	return out, nil
}

// applyHighPass is a simple 1-pole HPF (~80 Hz at 48 kHz) for rumble rejection.
func (p *speexProcessor) applyHighPass(pcm []int16) {
	const rc = 1.0 / (2 * math.Pi * 80)
	dt := 1.0 / float64(p.cfg.SampleRate)
	alpha := rc / (rc + dt)
	for i, s := range pcm {
		x := float64(s)
		y := alpha * (p.hpY + x - p.hpX)
		p.hpX = x
		p.hpY = y
		if y > 32767 {
			y = 32767
		} else if y < -32768 {
			y = -32768
		}
		pcm[i] = int16(y)
	}
}

func (p *speexProcessor) Stats() Stats {
	return Stats{Backend: string(BackendSpeex), ERLE: p.erle, Active: p.cfg.EchoCancel}
}

func (p *speexProcessor) Close() error {
	if p.preprocess != nil {
		C.speex_preprocess_state_destroy(p.preprocess)
		p.preprocess = nil
	}
	if p.echo != nil {
		C.speex_echo_state_destroy(p.echo)
		p.echo = nil
	}
	return nil
}
