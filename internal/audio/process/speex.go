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
	} else {
		C.speex_preprocess_ctl(pp, C.SPEEX_PREPROCESS_SET_DENOISE, unsafe.Pointer(&zero))
	}
	if cfg.AGC {
		C.speex_preprocess_ctl(pp, C.SPEEX_PREPROCESS_SET_AGC, unsafe.Pointer(&one))
	}
	if cfg.Dereverb {
		C.speex_preprocess_ctl(pp, C.SPEEX_PREPROCESS_SET_DEREVERB, unsafe.Pointer(&one))
	}

	return &speexProcessor{cfg: cfg, echo: echo, preprocess: pp, frame: frame}, nil
}

func (p *speexProcessor) ProcessRender(far []int16) error {
	if !p.cfg.EchoCancel || p.echo == nil {
		return nil
	}
	if len(far) < p.frame {
		return nil
	}
	// speex expects frame-sized buffers; process first frame only here.
	C.speex_echo_playback(p.echo, (*C.spx_int16_t)(unsafe.Pointer(&far[0])))
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
	if len(near) > p.frame {
		copy(out[p.frame:], near[p.frame:])
	}
	return out, nil
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
