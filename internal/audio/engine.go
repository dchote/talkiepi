package audio

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	mumbleaudio "github.com/dchote/go-mumble-server/pkg/mumble/audio"

	"github.com/dchote/talkiepi/internal/audio/process"
	"github.com/dchote/talkiepi/internal/codec"
	"github.com/dchote/talkiepi/internal/jitter"
	"github.com/dchote/talkiepi/pkg/mumbleclient"
)

// rxHangover is how long Receiving stays true after the last inbound voice frame.
const rxHangover = 400 * time.Millisecond

// EngineCallbacks let the controller gate TX and drive communicator mode.
type EngineCallbacks struct {
	AllowTX     func() bool
	OnSpeech    func(speech bool)
	SuppressRX  func() bool
	ChimeState  func() ChimeKind // Idle/None, Open, Close
	OnChimeDone func()
}

// ChimeKind selects which local chime to play.
type ChimeKind int

const (
	ChimeNone ChimeKind = iota
	ChimeOpen
	ChimeClose
)

// EngineConfig configures a live voice engine.
type EngineConfig struct {
	Profile    codec.Profile
	Processing process.Config
	Input      Descriptor
	Output     Descriptor
	Enumerator Enumerator
	Sink       VoiceSink
	Log        *slog.Logger
	Callbacks  EngineCallbacks
	// InputGainDB is applied after AEC/NS (0 = unity).
	InputGainDB float64
	// OutputVolume is 0–100 playback level (100 = unity).
	OutputVolume float64
}

// Engine runs capture → process → encode → sink and RX → decode → jitter → play.
type Engine struct {
	cfg     EngineConfig
	session *Session
	enc     *codec.Encoder
	devices DeviceIO
	frame   int
	rate    int
	outGain float64

	mu       sync.Mutex
	decoders map[uint32]*codec.Decoder
	jitters  map[uint32]*jitter.Buffer
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	chimeMu   sync.Mutex
	chimePCM  []int16
	chimePos  int
	chimeKind ChimeKind
	lastChime ChimeKind

	lastRXUnixNano atomic.Int64

	rateLog   *rateLimitedLog
	txActive  atomic.Bool
	rxLogged  sync.Map // session -> struct{}
	emptyCapN atomic.Uint64
}

// NewEngine builds an engine (devices are opened on Start).
func NewEngine(cfg EngineConfig) (*Engine, error) {
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	if cfg.Profile.SampleRate == 0 {
		cfg.Profile = codec.Default()
	}
	if cfg.Processing.SampleRate == 0 {
		cfg.Processing = process.DefaultConfig(cfg.Profile.SampleRate, cfg.Profile.FrameSamples())
	}
	sess, err := NewSession(SessionConfig{
		Profile:    cfg.Profile,
		Processing: cfg.Processing,
		Input:      cfg.Input,
		Output:     cfg.Output,
		InputGain:  GainLinearFromDB(cfg.InputGainDB),
	}, cfg.Sink)
	if err != nil {
		return nil, err
	}
	enc, err := codec.NewEncoder(cfg.Profile)
	if err != nil {
		_ = sess.Close()
		return nil, err
	}
	outVol := cfg.OutputVolume
	if outVol <= 0 {
		outVol = 100
	}
	return &Engine{
		cfg:      cfg,
		session:  sess,
		enc:      enc,
		frame:    cfg.Profile.FrameSamples(),
		rate:     cfg.Profile.SampleRate,
		outGain:  VolumeLinearFromPercent(outVol),
		decoders: make(map[uint32]*codec.Decoder),
		jitters:  make(map[uint32]*jitter.Buffer),
		rateLog:  newRateLimitedLog(5 * time.Second),
	}, nil
}

// Start opens devices and begins the frame loop.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.cancel != nil {
		e.mu.Unlock()
		return fmt.Errorf("audio: engine already started")
	}
	dev, err := OpenDevices(DeviceOpenConfig{
		Enumerator:   e.cfg.Enumerator,
		Input:        e.cfg.Input,
		Output:       e.cfg.Output,
		SampleRate:   uint32(e.rate),
		Channels:     1,
		PeriodFrames: uint32(e.frame),
		CaptureRing:  e.session.CaptureRing(),
		PlaybackRing: e.session.PlaybackRing(),
	})
	if err != nil {
		e.mu.Unlock()
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	e.devices = dev
	e.cancel = cancel
	e.mu.Unlock()

	if err := dev.Start(runCtx); err != nil {
		_ = e.Stop()
		return err
	}
	e.cfg.Log.Info("audio engine started",
		"sample_rate", e.rate,
		"frame", e.frame,
		"input", e.cfg.Input.DeviceName,
		"output", e.cfg.Output.DeviceName,
		"input_gain_db", e.cfg.InputGainDB,
		"output_volume", e.cfg.OutputVolume,
	)

	e.wg.Add(1)
	go e.loop(runCtx)
	return nil
}

// Stop tears down the frame loop and devices.
func (e *Engine) Stop() error {
	e.mu.Lock()
	cancel := e.cancel
	dev := e.devices
	e.cancel = nil
	e.devices = nil
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	e.wg.Wait()

	var first error
	if dev != nil {
		_ = dev.Stop()
		if err := dev.Close(); err != nil && first == nil {
			first = err
		}
	}
	// Send terminator if we were mid-TX.
	if e.cfg.Sink != nil {
		_ = e.cfg.Sink.SendVoice(nil, true)
	}
	e.cfg.Log.Info("audio engine stopped")
	return first
}

// Close releases codec resources.
func (e *Engine) Close() error {
	_ = e.Stop()
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, d := range e.decoders {
		_ = d.Close()
	}
	e.decoders = nil
	e.jitters = nil
	if e.enc != nil {
		_ = e.enc.Close()
		e.enc = nil
	}
	if e.session != nil {
		_ = e.session.Close()
		e.session = nil
	}
	return nil
}

// HandleVoicePacket ingests an inbound Mumble voice packet.
func (e *Engine) HandleVoicePacket(pkt mumbleclient.VoicePacket) {
	sv, err := mumbleclient.ParseServerVoice(pkt.Data)
	if err != nil || sv == nil {
		e.rateLog.Warn(e.cfg.Log, "rx-parse", "voice rx drop", "reason", "parse", "err", err)
		return
	}
	if sv.Codec != mumbleaudio.CodecOpus {
		e.rateLog.Warn(e.cfg.Log, "rx-codec", "voice rx drop", "reason", "codec", "codec", sv.Codec)
		return
	}
	if e.cfg.Callbacks.SuppressRX != nil && e.cfg.Callbacks.SuppressRX() {
		e.rateLog.Info(e.cfg.Log, "rx-suppress", "voice rx suppressed", "session", sv.Session)
		return
	}
	if sv.IsTerminator && len(sv.Payload) == 0 {
		e.mu.Lock()
		if j := e.jitters[sv.Session]; j != nil {
			j.Reset()
		}
		e.mu.Unlock()
		e.cfg.Log.Debug("voice rx terminator", "session", sv.Session)
		return
	}
	dec, jbuf := e.streamFor(sv.Session)
	if dec == nil || jbuf == nil {
		e.rateLog.Warn(e.cfg.Log, "rx-stream", "voice rx drop", "reason", "decoder", "session", sv.Session)
		return
	}
	pcm, err := dec.Decode(sv.Payload)
	if err != nil {
		e.rateLog.Warn(e.cfg.Log, "rx-decode", "opus decode", "err", err, "session", sv.Session)
		return
	}
	jbuf.Push(sv.Sequence, pcm)
	e.lastRXUnixNano.Store(time.Now().UnixNano())
	if _, loaded := e.rxLogged.LoadOrStore(sv.Session, struct{}{}); !loaded {
		e.cfg.Log.Info("voice rx started", "session", sv.Session)
	}
}

// Receiving reports whether inbound voice was decoded within the hangover window.
func (e *Engine) Receiving() bool {
	last := e.lastRXUnixNano.Load()
	if last == 0 {
		return false
	}
	return time.Since(time.Unix(0, last)) < rxHangover
}

func (e *Engine) streamFor(session uint32) (*codec.Decoder, *jitter.Buffer) {
	e.mu.Lock()
	defer e.mu.Unlock()
	dec := e.decoders[session]
	if dec == nil {
		d, err := codec.NewDecoder(e.cfg.Profile)
		if err != nil {
			e.cfg.Log.Warn("opus decoder", "err", err)
			return nil, nil
		}
		e.decoders[session] = d
		dec = d
	}
	j := e.jitters[session]
	if j == nil {
		j = jitter.New(24, 3)
		e.jitters[session] = j
	}
	return dec, j
}

func (e *Engine) loop(ctx context.Context) {
	defer e.wg.Done()
	ticker := time.NewTicker(time.Duration(e.cfg.Profile.FrameMs) * time.Millisecond)
	defer ticker.Stop()

	capFrame := make([]int16, e.frame)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.tick(capFrame)
		}
	}
}

func (e *Engine) tick(capFrame []int16) {
	// Playback mix from jitter buffers + Opus PLC on underrun + optional chime.
	mix := e.session.Mixer()
	e.mu.Lock()
	for session, j := range e.jitters {
		if pcm, ok := j.Pop(); ok {
			mix.Add(pcm)
			continue
		}
		if !j.Primed() {
			continue
		}
		// Gap: synthesize with Opus PLC when available.
		if dec := e.decoders[session]; dec != nil {
			if plc, err := dec.DecodePLC(); err == nil {
				mix.Add(plc)
			}
		}
	}
	e.mu.Unlock()

	e.feedChime(mix)

	out := mix.Render()
	ApplyLinearGain(out, e.outGain)
	_ = e.session.FeedFarEnd(out)
	_ = e.session.PlaybackRing().Write(out)

	// Capture path.
	if e.session.CaptureRing().Len() < e.frame {
		n := e.emptyCapN.Add(1)
		if n == 1 || n%250 == 0 { // ~5s at 20 ms frames after first
			e.rateLog.Warn(e.cfg.Log, "cap-empty", "capture ring underrun",
				"need", e.frame, "have", e.session.CaptureRing().Len(), "count", n)
		}
		return
	}
	e.emptyCapN.Store(0)
	n := e.session.CaptureRing().Read(capFrame)
	if n < e.frame {
		for i := n; i < e.frame; i++ {
			capFrame[i] = 0
		}
	}

	allow := false
	if e.cfg.Callbacks.AllowTX != nil {
		allow = e.cfg.Callbacks.AllowTX()
	}

	encode := func(pcm []int16) ([]byte, error) { return e.enc.Encode(pcm) }
	speech, err := e.session.ProcessCaptureFrame(capFrame, encode, allow)
	if err != nil {
		e.cfg.Log.Warn("capture frame", "err", err)
	} else if allow {
		if !e.txActive.Swap(true) {
			e.cfg.Log.Info("voice tx started")
		}
	} else if e.txActive.Swap(false) {
		e.cfg.Log.Info("voice tx stopped")
	}
	if e.cfg.Callbacks.OnSpeech != nil {
		e.cfg.Callbacks.OnSpeech(speech)
	}
}

// ForceTransmit opens the VAD TX gate (used by PTT / always_on).
func (e *Engine) ForceTransmit() {
	if e.session != nil {
		e.session.ForceTransmit()
	}
}

// ForceStop closes the VAD TX gate.
func (e *Engine) ForceStop() {
	if e.session != nil {
		e.session.ForceStop()
	}
}

func (e *Engine) feedChime(mix *Mixer) {
	want := ChimeNone
	if e.cfg.Callbacks.ChimeState != nil {
		want = e.cfg.Callbacks.ChimeState()
	}

	e.chimeMu.Lock()
	defer e.chimeMu.Unlock()

	if want != e.lastChime {
		e.lastChime = want
		e.chimePos = 0
		switch want {
		case ChimeOpen:
			e.chimePCM = OpenChimePCM(e.rate)
			e.chimeKind = ChimeOpen
			if len(e.chimePCM) == 0 {
				e.cfg.Log.Warn("communicator open chime missing; skipping")
			} else {
				e.cfg.Log.Info("communicator chime start", "kind", "open", "samples", len(e.chimePCM), "rate", e.rate)
			}
		case ChimeClose:
			e.chimePCM = CloseChimePCM(e.rate)
			e.chimeKind = ChimeClose
			if len(e.chimePCM) == 0 {
				e.cfg.Log.Warn("communicator close chime missing; skipping")
			} else {
				e.cfg.Log.Info("communicator chime start", "kind", "close", "samples", len(e.chimePCM), "rate", e.rate)
			}
		default:
			e.chimePCM = nil
			e.chimeKind = ChimeNone
		}
	}

	// Empty PCM must still advance the communicator FSM (esp32 skips when size==0).
	if want == ChimeOpen || want == ChimeClose {
		if len(e.chimePCM) == 0 {
			kind := e.chimeKind
			e.chimeKind = ChimeNone
			e.lastChime = ChimeNone
			e.chimePCM = nil
			if kind != ChimeNone && e.cfg.Callbacks.OnChimeDone != nil {
				e.cfg.Callbacks.OnChimeDone()
			}
			return
		}
	} else if len(e.chimePCM) == 0 {
		return
	}

	frame := make([]int16, e.frame)
	n := copy(frame, e.chimePCM[e.chimePos:])
	e.chimePos += n
	mix.Add(frame)
	if e.chimePos >= len(e.chimePCM) {
		e.chimePCM = nil
		kind := e.chimeKind
		e.chimeKind = ChimeNone
		e.lastChime = ChimeNone
		e.cfg.Log.Info("communicator chime done", "kind", chimeKindName(kind))
		if kind != ChimeNone && e.cfg.Callbacks.OnChimeDone != nil {
			e.cfg.Callbacks.OnChimeDone()
		}
	}
}

func chimeKindName(k ChimeKind) string {
	switch k {
	case ChimeOpen:
		return "open"
	case ChimeClose:
		return "close"
	default:
		return "none"
	}
}
