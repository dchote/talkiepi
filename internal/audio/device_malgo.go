//go:build cgo

package audio

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync"
	"unsafe"

	"github.com/gen2brain/malgo"
)

type malgoIO struct {
	cfg DeviceOpenConfig

	mu      sync.Mutex
	ctx     *malgo.AllocatedContext
	capture *malgo.Device
	play    *malgo.Device
	capID   unsafe.Pointer // C-allocated ma_device_id (from DeviceID.Pointer)
	playID  unsafe.Pointer
	started bool
}

func openDevices(cfg DeviceOpenConfig) (DeviceIO, error) {
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 48000
	}
	if cfg.Channels == 0 {
		cfg.Channels = 1
	}
	if cfg.PeriodFrames == 0 {
		cfg.PeriodFrames = cfg.SampleRate / 50
	}
	if cfg.CaptureRing == nil || cfg.PlaybackRing == nil {
		return nil, fmt.Errorf("audio: capture and playback rings required")
	}

	backends, err := malgoBackends(cfg.Input.Backend)
	if err != nil {
		return nil, err
	}
	if cfg.Output.Backend != "" && cfg.Output.Backend != BackendAuto && cfg.Output.Backend != cfg.Input.Backend {
		if b2, err2 := malgoBackends(cfg.Output.Backend); err2 == nil && len(b2) > 0 {
			backends = b2
		}
	}

	actx, err := malgo.InitContext(backends, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("audio: init context: %w", err)
	}

	io := &malgoIO{cfg: cfg, ctx: actx}

	capID, capDesc := resolveDeviceID(cfg.Enumerator, cfg.Input, DirectionCapture)
	playID, playDesc := resolveDeviceID(cfg.Enumerator, cfg.Output, DirectionPlayback)
	if capDesc.DeviceID != "" {
		cfg.Input = capDesc
	}
	if playDesc.DeviceID != "" {
		cfg.Output = playDesc
	}
	io.cfg = cfg

	// DeviceID must be a C heap pointer — Go pointers inside DeviceConfig panic under cgo checks.
	if capID != (malgo.DeviceID{}) {
		io.capID = capID.Pointer()
	}
	if playID != (malgo.DeviceID{}) {
		io.playID = playID.Pointer()
	}

	captureRing := cfg.CaptureRing
	playbackRing := cfg.PlaybackRing
	channels := cfg.Channels

	capCfg := malgo.DefaultDeviceConfig(malgo.Capture)
	capCfg.Capture.Format = malgo.FormatS16
	capCfg.Capture.Channels = channels
	capCfg.SampleRate = cfg.SampleRate
	capCfg.PeriodSizeInFrames = cfg.PeriodFrames
	capCfg.Alsa.NoMMap = 1
	if io.capID != nil {
		capCfg.Capture.DeviceID = io.capID
	}

	onCapture := func(_, input []byte, frameCount uint32) {
		if len(input) == 0 || frameCount == 0 {
			return
		}
		n := int(frameCount * channels)
		_ = captureRing.Write(bytesToInt16(input, n))
	}

	capDev, err := malgo.InitDevice(actx.Context, capCfg, malgo.DeviceCallbacks{Data: onCapture})
	if err != nil {
		io.freeIDs()
		_ = actx.Uninit()
		actx.Free()
		return nil, fmt.Errorf("audio: open capture: %w", err)
	}
	io.capture = capDev

	playCfg := malgo.DefaultDeviceConfig(malgo.Playback)
	playCfg.Playback.Format = malgo.FormatS16
	playCfg.Playback.Channels = channels
	playCfg.SampleRate = cfg.SampleRate
	playCfg.PeriodSizeInFrames = cfg.PeriodFrames
	playCfg.Alsa.NoMMap = 1
	if io.playID != nil {
		playCfg.Playback.DeviceID = io.playID
	}

	onPlayback := func(output, _ []byte, frameCount uint32) {
		if len(output) == 0 || frameCount == 0 {
			return
		}
		need := int(frameCount * channels)
		samples := make([]int16, need)
		n := playbackRing.Read(samples)
		if n < need {
			for i := n; i < need; i++ {
				samples[i] = 0
			}
		}
		int16ToBytes(samples, output)
	}

	playDev, err := malgo.InitDevice(actx.Context, playCfg, malgo.DeviceCallbacks{Data: onPlayback})
	if err != nil {
		capDev.Uninit()
		io.freeIDs()
		_ = actx.Uninit()
		actx.Free()
		return nil, fmt.Errorf("audio: open playback: %w", err)
	}
	io.play = playDev
	return io, nil
}

func resolveDeviceID(enum Enumerator, want Descriptor, dir Direction) (malgo.DeviceID, Descriptor) {
	if id, ok := decodeMalgoID(want.DeviceID); ok {
		return id, want
	}
	if enum == nil {
		return malgo.DeviceID{}, want
	}
	devices, err := enum.Devices(want.Backend, dir)
	if err != nil {
		return malgo.DeviceID{}, want
	}
	d, rule := Resolve(want, devices)
	if rule == MatchNone {
		return malgo.DeviceID{}, want
	}
	id, ok := decodeMalgoID(d.DeviceID)
	if !ok {
		return malgo.DeviceID{}, d.Descriptor
	}
	return id, d.Descriptor
}

func (d *malgoIO) freeIDs() {
	if d.capID != nil {
		C.free(d.capID)
		d.capID = nil
	}
	if d.playID != nil {
		C.free(d.playID)
		d.playID = nil
	}
}

func (d *malgoIO) Start(context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.started {
		return nil
	}
	if err := d.capture.Start(); err != nil {
		return fmt.Errorf("audio: start capture: %w", err)
	}
	if err := d.play.Start(); err != nil {
		_ = d.capture.Stop()
		return fmt.Errorf("audio: start playback: %w", err)
	}
	d.started = true
	return nil
}

func (d *malgoIO) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.started {
		return nil
	}
	var first error
	if d.play != nil {
		if err := d.play.Stop(); err != nil && first == nil {
			first = err
		}
	}
	if d.capture != nil {
		if err := d.capture.Stop(); err != nil && first == nil {
			first = err
		}
	}
	d.started = false
	return first
}

func (d *malgoIO) Close() error {
	_ = d.Stop()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.play != nil {
		d.play.Uninit()
		d.play = nil
	}
	if d.capture != nil {
		d.capture.Uninit()
		d.capture = nil
	}
	d.freeIDs()
	if d.ctx != nil {
		_ = d.ctx.Uninit()
		d.ctx.Free()
		d.ctx = nil
	}
	return nil
}

func decodeMalgoID(s string) (malgo.DeviceID, bool) {
	if s == "" {
		return malgo.DeviceID{}, false
	}
	b, err := hex.DecodeString(s)
	if err != nil || len(b) == 0 {
		return malgo.DeviceID{}, false
	}
	var id malgo.DeviceID
	copy(id[:], b)
	return id, true
}

func bytesToInt16(b []byte, nSamples int) []int16 {
	max := len(b) / 2
	if nSamples > max {
		nSamples = max
	}
	out := make([]int16, nSamples)
	for i := 0; i < nSamples; i++ {
		out[i] = int16(binary.LittleEndian.Uint16(b[i*2:]))
	}
	return out
}

func int16ToBytes(samples []int16, dst []byte) {
	n := len(samples)
	if n*2 > len(dst) {
		n = len(dst) / 2
	}
	for i := 0; i < n; i++ {
		binary.LittleEndian.PutUint16(dst[i*2:], uint16(samples[i]))
	}
}
