package vad

import (
	"math"
)

// Defaults match esp32-mumble voice capture constants.
const (
	DefaultCalibrationFrames = 12
	DefaultAttackFrames      = 3
	DefaultHangoverFrames    = 15
	DefaultMinThreshold      = 120.0
	DefaultMargin            = 2.5
	DefaultFloorDown         = 0.25
	DefaultFloorUpSpeech     = 0.05
	DefaultFloorUpSilence    = 0.01
)

// State is the transmit decision for a frame.
type State int

const (
	StateIdle State = iota
	StateCapturing
	StateTransmitting
)

// Config tunes VAD behavior.
type Config struct {
	CalibrationFrames int
	AttackFrames      int
	HangoverFrames    int
	MinThreshold      float64
	Margin            float64
	FloorDown         float64
	FloorUpSpeech     float64
	FloorUpSilence    float64
}

// DefaultConfig returns esp32-compatible defaults.
func DefaultConfig() Config {
	return Config{
		CalibrationFrames: DefaultCalibrationFrames,
		AttackFrames:      DefaultAttackFrames,
		HangoverFrames:    DefaultHangoverFrames,
		MinThreshold:      DefaultMinThreshold,
		Margin:            DefaultMargin,
		FloorDown:         DefaultFloorDown,
		FloorUpSpeech:     DefaultFloorUpSpeech,
		FloorUpSilence:    DefaultFloorUpSilence,
	}
}

// Detector is an adaptive energy VAD with hangover and terminator signaling.
type Detector struct {
	cfg            Config
	state          State
	noiseFloor     float64
	calFrames      int
	calSum         float64
	aboveCount     int
	silenceCount   int
	calibrated     bool
	sendTerminator bool
	lastRMS        float64
}

// New creates a Detector.
func New(cfg Config) *Detector {
	if cfg.CalibrationFrames < 1 {
		cfg.CalibrationFrames = DefaultCalibrationFrames
	}
	if cfg.AttackFrames < 1 {
		cfg.AttackFrames = DefaultAttackFrames
	}
	if cfg.HangoverFrames < 1 {
		cfg.HangoverFrames = DefaultHangoverFrames
	}
	if cfg.MinThreshold <= 0 {
		cfg.MinThreshold = DefaultMinThreshold
	}
	if cfg.Margin <= 0 {
		cfg.Margin = DefaultMargin
	}
	return &Detector{cfg: cfg, state: StateIdle}
}

// Result is the per-frame VAD decision.
type Result struct {
	State          State
	RMS            float64
	NoiseFloor     float64
	Speech         bool
	Transmit       bool
	SendTerminator bool
}

// Process evaluates one PCM frame and returns the transmit decision.
// When SendTerminator is true, the caller must send a terminator packet
// even if Transmit is false.
func (d *Detector) Process(samples []int16) Result {
	rms := RMS(samples)
	d.lastRMS = rms

	if !d.calibrated {
		d.calSum += rms
		d.calFrames++
		if d.calFrames >= d.cfg.CalibrationFrames {
			d.noiseFloor = d.calSum / float64(d.calFrames)
			if d.noiseFloor < d.cfg.MinThreshold {
				d.noiseFloor = d.cfg.MinThreshold
			}
			d.calibrated = true
			d.state = StateCapturing
		}
		return Result{State: d.state, RMS: rms, NoiseFloor: d.noiseFloor}
	}

	threshold := math.Max(d.cfg.MinThreshold, d.noiseFloor*d.cfg.Margin)
	speech := rms >= threshold

	// Adaptive floor
	if speech {
		d.noiseFloor += (rms - d.noiseFloor) * d.cfg.FloorUpSpeech
	} else if rms < d.noiseFloor {
		d.noiseFloor += (rms - d.noiseFloor) * d.cfg.FloorDown
	} else {
		d.noiseFloor += (rms - d.noiseFloor) * d.cfg.FloorUpSilence
	}
	if d.noiseFloor < d.cfg.MinThreshold {
		d.noiseFloor = d.cfg.MinThreshold
	}

	d.sendTerminator = false
	switch d.state {
	case StateCapturing:
		if speech {
			d.aboveCount++
			if d.aboveCount >= d.cfg.AttackFrames {
				d.state = StateTransmitting
				d.silenceCount = 0
			}
		} else {
			d.aboveCount = 0
		}
	case StateTransmitting:
		if speech {
			d.silenceCount = 0
		} else {
			d.silenceCount++
			if d.silenceCount >= d.cfg.HangoverFrames {
				d.state = StateCapturing
				d.aboveCount = 0
				d.sendTerminator = true
			}
		}
	}

	return Result{
		State:          d.state,
		RMS:            rms,
		NoiseFloor:     d.noiseFloor,
		Speech:         speech,
		Transmit:       d.state == StateTransmitting,
		SendTerminator: d.sendTerminator,
	}
}

// ForceTransmit starts transmitting immediately (PTT press).
func (d *Detector) ForceTransmit() {
	d.state = StateTransmitting
	d.silenceCount = 0
	d.aboveCount = d.cfg.AttackFrames
	if !d.calibrated {
		d.calibrated = true
		d.noiseFloor = d.cfg.MinThreshold
	}
}

// ForceStop ends transmitting and requests a terminator.
func (d *Detector) ForceStop() {
	if d.state == StateTransmitting {
		d.sendTerminator = true
	}
	d.state = StateCapturing
	d.aboveCount = 0
	d.silenceCount = 0
}

// TakeTerminator returns and clears the pending terminator flag.
func (d *Detector) TakeTerminator() bool {
	v := d.sendTerminator
	d.sendTerminator = false
	return v
}

// Reset clears calibration and state.
func (d *Detector) Reset() {
	*d = *New(d.cfg)
}

// RMS computes root-mean-square amplitude of samples.
func RMS(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		v := float64(s)
		sum += v * v
	}
	return math.Sqrt(sum / float64(len(samples)))
}
