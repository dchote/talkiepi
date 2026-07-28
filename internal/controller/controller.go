package controller

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dchote/talkiepi/internal/audio"
	"github.com/dchote/talkiepi/internal/audio/process"
	"github.com/dchote/talkiepi/internal/codec"
	"github.com/dchote/talkiepi/internal/communicator"
	"github.com/dchote/talkiepi/internal/config"
	"github.com/dchote/talkiepi/internal/discovery"
	"github.com/dchote/talkiepi/internal/led"
	"github.com/dchote/talkiepi/internal/ptt"
	"github.com/dchote/talkiepi/internal/update"
	"github.com/dchote/talkiepi/pkg/mumbleclient"
)

// Status is a snapshot of runtime state for the REST API / UI.
type Status struct {
	Connected    bool    `json:"connected"`
	Connecting   bool    `json:"connecting"`
	Address      string  `json:"address"`
	Username     string  `json:"username"`
	Channel      string  `json:"channel"`
	Session      uint32  `json:"session"`
	PingMs       float64 `json:"ping_ms"`
	UDPActive    bool    `json:"udp_active"`
	Transport    string  `json:"transport"`
	Mode         string  `json:"mode"`
	Transmitting bool    `json:"transmitting"`
	Receiving    bool    `json:"receiving"`
	Participants int     `json:"participants"`
	Version      string  `json:"version"`
}

// Controller wires config, mumble client, audio, triggers, LEDs, and updates.
type Controller struct {
	cfg     *config.Config
	log     *slog.Logger
	version string
	browser discovery.Browser
	enum    audio.Enumerator

	mu      sync.RWMutex
	client  *mumbleclient.Client
	engine  *audio.Engine
	comm    *communicator.Communicator
	leds    led.Indicator
	manual  *ptt.Manual
	mux     *ptt.Mux
	updater *update.Checker
	chat    chatStore

	transmitting atomic.Bool
	connecting   atomic.Bool
	rttMs        atomic.Int64
	noEngWarnNS  atomic.Int64

	cancel    context.CancelFunc
	runCtx    context.Context
	wg        sync.WaitGroup
	pttEvents chan ptt.Event

	listenersMu sync.RWMutex
	listeners   []chan any
}

// Options configures the controller.
type Options struct {
	Config     *config.Config
	Logger     *slog.Logger
	Version    string
	LEDs       led.Indicator
	Updater    *update.Checker
	Browser    discovery.Browser
	Enumerator audio.Enumerator
}

// New creates a Controller.
func New(opt Options) *Controller {
	if opt.Logger == nil {
		opt.Logger = slog.Default()
	}
	if opt.LEDs == nil {
		opt.LEDs = led.NewMemory()
	}
	if opt.Browser == nil {
		opt.Browser = discovery.NewMDNSBrowser()
	}
	if opt.Enumerator == nil {
		opt.Enumerator = audio.NewMalgoEnumerator()
	}
	manual := ptt.NewManual("rest")
	return &Controller{
		cfg:     opt.Config,
		log:     opt.Logger,
		version: opt.Version,
		browser: opt.Browser,
		enum:    opt.Enumerator,
		comm:    communicator.New(),
		leds:    opt.LEDs,
		manual:  manual,
		mux:     ptt.NewMux(manual),
		updater: opt.Updater,
	}
}

// Start begins background trigger loops and connects if a server is configured.
func (c *Controller) Start(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)
	c.runCtx = ctx
	c.pttEvents = make(chan ptt.Event, 16)

	c.rebuildMux(ctx)

	c.wg.Add(1)
	go c.triggerLoop(ctx, c.pttEvents)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.maybeAutoDiscover(ctx)
		if c.cfg.Clone().Server.Address == "" {
			return
		}
		if err := c.dialOnce(ctx); err != nil {
			c.log.Warn("initial connect failed", "err", err)
		}
	}()
	return nil
}

func (c *Controller) rebuildMux(ctx context.Context) {
	if c.mux != nil {
		_ = c.mux.Stop()
	}
	sources := []ptt.Source{c.manual}
	tr := c.cfg.Clone().Triggers
	if tr.GPIOEnabled {
		chip := tr.GPIOChip
		if chip == "" {
			chip = "gpiochip0"
		}
		if !strings.Contains(chip, "/") {
			chip = "/dev/" + chip
		}
		gpio, err := ptt.NewGPIO(ptt.GPIOConfig{
			Chip:      chip,
			Line:      tr.GPIOLine,
			ActiveLow: tr.ActiveLow,
			Name:      "gpio",
		})
		if err != nil {
			c.log.Warn("gpio ptt unavailable", "err", err)
		} else {
			sources = append(sources, gpio)
			c.log.Info("gpio ptt enabled", "chip", chip, "line", tr.GPIOLine, "active_low", tr.ActiveLow)
		}
	}
	c.mux = ptt.NewMux(sources...)
	if err := c.mux.Start(ctx, c.pttEvents); err != nil {
		c.log.Warn("ptt mux start", "err", err)
	}
}

// DiscoverServers browses the LAN for _mumble._tcp services.
func (c *Controller) DiscoverServers(ctx context.Context, timeout time.Duration) ([]discovery.Service, error) {
	if c.browser == nil {
		return nil, nil
	}
	return c.browser.Browse(ctx, timeout)
}

func (c *Controller) maybeAutoDiscover(ctx context.Context) {
	cfg := c.cfg.Clone()
	if !cfg.Server.AutoDiscover || cfg.Server.Address != "" {
		return
	}
	services, err := c.DiscoverServers(ctx, 3*time.Second)
	if err != nil {
		c.log.Warn("mDNS discovery failed", "err", err)
		return
	}
	if len(services) == 0 {
		c.log.Info("mDNS discovery: no Mumble servers found")
		return
	}
	svc := services[0]
	host := svc.PreferredHost()
	if host == "" {
		return
	}
	port := svc.Port
	if port <= 0 {
		port = 64738
	}
	c.cfg.SetServerEndpoint(host, port)
	if err := c.cfg.Save(""); err != nil {
		c.log.Warn("config save after discovery", "err", err)
	}
	c.log.Info("mDNS auto-populated server", "name", svc.Name, "address", host, "port", port, "candidates", len(services))
	c.emit(map[string]any{"type": "config", "source": "mdns", "server": svc})
}

// Stop shuts down the controller.
func (c *Controller) Stop() {
	c.stopAudio()
	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Lock()
	if c.client != nil {
		_ = c.client.Close()
		c.client = nil
	}
	c.mu.Unlock()
	_ = c.mux.Stop()
	c.leds.SetOnline(false)
	c.leds.SetParticipants(false)
	c.leds.SetTransmit(false)
	c.wg.Wait()
}

// Config returns the live config.
func (c *Controller) Config() *config.Config { return c.cfg }

// ManualPTT returns the REST/manual trigger source.
func (c *Controller) ManualPTT() *ptt.Manual { return c.manual }

// Communicator returns the half-duplex state machine.
func (c *Controller) Communicator() *communicator.Communicator { return c.comm }

// ToggleCommunicator toggles half-duplex Communicator mode (REST / button).
func (c *Controller) ToggleCommunicator() {
	prev := c.comm.State()
	c.comm.Toggle()
	c.log.Info("communicator toggle", "from", prev.String(), "to", c.comm.State().String(), "source", "rest")
	c.emit(map[string]any{"type": "communicator", "state": c.comm.State().String()})
}

// ChatMessages returns recent chat history, optionally filtered by channel.
func (c *Controller) ChatMessages(channelID *uint32, limit int) []ChatMessage {
	return c.chat.list(channelID, limit)
}

// Subscribe returns a fan-out event channel; caller must Unsubscribe.
func (c *Controller) Subscribe() chan any {
	ch := make(chan any, 32)
	c.listenersMu.Lock()
	c.listeners = append(c.listeners, ch)
	c.listenersMu.Unlock()
	return ch
}

// Unsubscribe removes a previously subscribed channel.
func (c *Controller) Unsubscribe(ch chan any) {
	c.listenersMu.Lock()
	defer c.listenersMu.Unlock()
	for i, l := range c.listeners {
		if l == ch {
			c.listeners = append(c.listeners[:i], c.listeners[i+1:]...)
			close(ch)
			return
		}
	}
}

func (c *Controller) emit(ev any) {
	c.listenersMu.RLock()
	defer c.listenersMu.RUnlock()
	for _, ch := range c.listeners {
		select {
		case ch <- ev:
		default:
		}
	}
}

// Status returns a status snapshot.
func (c *Controller) Status() Status {
	cfg := c.cfg.Clone()
	st := Status{
		Connecting:   c.connecting.Load(),
		Address:      cfg.ServerHostPort(),
		Username:     cfg.Server.Username,
		Mode:         cfg.Voice.Mode,
		Transmitting: c.transmitting.Load() || c.comm.ShouldTransmit(),
		Version:      c.version,
		PingMs:       float64(c.rttMs.Load()),
		Transport:    "TCP",
	}
	c.mu.RLock()
	cl := c.client
	eng := c.engine
	c.mu.RUnlock()
	if eng != nil {
		st.Receiving = eng.Receiving()
	}
	if cl != nil && cl.IsConnected() {
		st.Connected = true
		st.Session = cl.Session()
		st.UDPActive = cl.IsUDPActive()
		if st.UDPActive {
			st.Transport = "UDP"
		}
		chID := cl.ChannelID()
		if ch, ok := cl.Roster().ChannelByID(chID); ok {
			st.Channel = ch.Name
			st.Participants = len(cl.Roster().UsersInChannel(ch.ID))
		}
		if cl.IsTransmitting() || cfg.Voice.Mode == "always_on" {
			st.Transmitting = true
		}
	}
	return st
}

// Connect starts or restarts the mumble connection using current config.
func (c *Controller) Connect(ctx context.Context) error {
	c.Disconnect()
	c.maybeAutoDiscover(ctx)
	c.log.Info("connect requested", "address", c.cfg.Clone().ServerHostPort())
	return c.dialOnce(ctx)
}

// Disconnect closes the mumble client and audio engine.
func (c *Controller) Disconnect() {
	c.log.Info("disconnect")
	c.stopAudio()
	c.comm.Cancel()
	c.transmitting.Store(false)
	c.leds.SetTransmit(false)
	c.mu.Lock()
	cl := c.client
	c.client = nil
	c.mu.Unlock()
	if cl != nil {
		_ = cl.Close()
	}
	c.leds.SetOnline(false)
	c.leds.SetParticipants(false)
	c.emit(map[string]any{"type": "disconnected"})
}

// ApplyConfig saves cfg changes.
func (c *Controller) ApplyConfig(patch []byte) error {
	prev := c.cfg.Clone()
	if err := c.cfg.ApplyPatch(patch); err != nil {
		return err
	}
	if err := c.cfg.Save(""); err != nil {
		c.log.Warn("config save", "err", err)
	}
	next := c.cfg.Clone()
	c.log.Info("config updated",
		"mode", next.Voice.Mode,
		"address", next.ServerHostPort(),
		"input_device", next.Audio.Input.DeviceID,
		"output_device", next.Audio.Output.DeviceID,
	)
	if prev.Triggers != next.Triggers && c.runCtx != nil {
		c.rebuildMux(c.runCtx)
	}
	if prev.Voice.Mode != next.Voice.Mode {
		c.applyVoiceModeSideEffects()
	}
	// Audio device / processing / gain / profile changes require an engine restart while connected.
	if audioPipelineChanged(prev, next) {
		if cl := c.Client(); cl != nil && cl.IsConnected() {
			c.log.Info("restarting audio after config change")
			if err := c.startAudioSafe(cl); err != nil {
				c.log.Warn("audio engine restart failed", "err", err)
			}
		}
	}
	c.emit(map[string]any{"type": "config"})
	return nil
}

func audioPipelineChanged(prev, next *config.Config) bool {
	if prev.Voice.SampleRate != next.Voice.SampleRate ||
		prev.Voice.Bitrate != next.Voice.Bitrate ||
		prev.Voice.FrameMs != next.Voice.FrameMs ||
		prev.Voice.Complexity != next.Voice.Complexity {
		return true
	}
	pi, po := prev.Audio.Input, next.Audio.Input
	if pi.Backend != po.Backend || pi.DeviceID != po.DeviceID || pi.DeviceName != po.DeviceName ||
		pi.GainDB != po.GainDB || pi.SampleRate != po.SampleRate || pi.PeriodFrames != po.PeriodFrames {
		return true
	}
	oi, oo := prev.Audio.Output, next.Audio.Output
	if oi.Backend != oo.Backend || oi.DeviceID != oo.DeviceID || oi.DeviceName != oo.DeviceName ||
		oi.Volume != oo.Volume || oi.SampleRate != oo.SampleRate || oi.PeriodFrames != oo.PeriodFrames {
		return true
	}
	pp, np := prev.Audio.Processing, next.Audio.Processing
	return pp.Backend != np.Backend || pp.EchoCancel != np.EchoCancel || pp.EchoTailMs != np.EchoTailMs ||
		pp.EchoDelayMs != np.EchoDelayMs || pp.NoiseSuppress != np.NoiseSuppress || pp.NSLevel != np.NSLevel ||
		pp.AGC != np.AGC || pp.AGCTargetDBFS != np.AGCTargetDBFS || pp.Dereverb != np.Dereverb ||
		pp.HighPass != np.HighPass || pp.DuplexMode != np.DuplexMode || pp.AdaptiveDelay != np.AdaptiveDelay
}

// Client returns the active mumble client (may be nil).
func (c *Controller) Client() *mumbleclient.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

// Updater returns the update checker (may be nil).
func (c *Controller) Updater() *update.Checker { return c.updater }

// Enumerator returns the audio device enumerator.
func (c *Controller) Enumerator() audio.Enumerator { return c.enum }

// JoinChannel joins a channel by ID and logs the action.
func (c *Controller) JoinChannel(id uint32) error {
	cl := c.Client()
	if cl == nil {
		return mumbleclient.ErrNotConnected
	}
	if err := cl.JoinChannelID(id); err != nil {
		c.log.Warn("join channel failed", "channel_id", id, "err", err)
		return err
	}
	name := ""
	if ch, ok := cl.Roster().ChannelByID(id); ok {
		name = ch.Name
	}
	c.log.Info("joined channel", "channel_id", id, "name", name)
	return nil
}

// SendChat sends a text message, stores it in history, and emits SSE.
func (c *Controller) SendChat(message string, channelID uint32) (ChatMessage, error) {
	cl := c.Client()
	if cl == nil {
		return ChatMessage{}, mumbleclient.ErrNotConnected
	}
	if channelID == 0 {
		channelID = cl.ChannelID()
	}
	if err := cl.SendTextMessage(message, channelID); err != nil {
		c.log.Warn("send chat failed", "err", err)
		return ChatMessage{}, err
	}
	msg := c.chat.append(ChatMessage{
		Actor:      cl.Session(),
		Author:     c.cfg.Clone().Server.Username,
		Message:    message,
		ChannelIDs: []uint32{channelID},
		Outbound:   true,
	})
	c.log.Info("chat sent", "channel_id", channelID, "bytes", len(message))
	c.emit(map[string]any{
		"type":        "text",
		"id":          msg.ID,
		"at":          msg.At,
		"actor":       msg.Actor,
		"author":      msg.Author,
		"message":     msg.Message,
		"channel_ids": msg.ChannelIDs,
		"outbound":    true,
	})
	return msg, nil
}

func (c *Controller) dialOnce(ctx context.Context) error {
	cfg := c.cfg.Clone()
	if cfg.Server.Address == "" {
		c.log.Warn("connect skipped", "reason", "empty server address")
		return fmt.Errorf("server address is empty")
	}
	c.connecting.Store(true)
	defer c.connecting.Store(false)

	modes := mumbleclient.DefaultCryptoModes
	switch cfg.Server.Crypto {
	case "lite":
		modes = mumbleclient.CryptoModeLite
	case "legacy":
		modes = mumbleclient.CryptoModeLegacy
	}

	mcfg := mumbleclient.Config{
		Address:     cfg.ServerHostPort(),
		Username:    cfg.Server.Username,
		Password:    cfg.Server.Password,
		Channel:     cfg.Server.Channel,
		InsecureTLS: cfg.Server.InsecureTLS,
		CertFile:    cfg.Server.Certificate,
		KeyFile:     cfg.Server.Key,
		CryptoModes: modes,
		Release:     "talkiepi",
	}
	if mcfg.Username == "" {
		mcfg.Username = "talkiepi-device"
	}
	cl := mumbleclient.New(mcfg)
	cl.On(c.handleMumbleEvent)
	c.log.Info("connecting", "address", mcfg.Address, "username", mcfg.Username)
	if err := cl.Dial(ctx); err != nil {
		c.log.Warn("connect failed", "err", err)
		return err
	}
	c.mu.Lock()
	c.client = cl
	c.mu.Unlock()
	c.leds.SetOnline(true)
	c.log.Info("connected", "session", cl.Session(), "channel", cfg.Server.Channel)
	if err := c.startAudioSafe(cl); err != nil {
		c.log.Warn("audio engine start failed", "err", err)
	}
	c.emit(map[string]any{"type": "connected", "session": cl.Session()})
	return nil
}

func (c *Controller) startAudioSafe(cl *mumbleclient.Client) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("audio panic: %v", r)
			c.log.Error("audio engine panic", "err", r)
		}
	}()
	return c.startAudio(cl)
}

type clientVoiceSink struct {
	c *mumbleclient.Client
}

func (s clientVoiceSink) SendVoice(payload []byte, terminator bool) error {
	return s.c.SendVoice(payload, terminator)
}

func (c *Controller) startAudio(cl *mumbleclient.Client) error {
	c.stopAudio()
	cfg := c.cfg.Clone()
	profile := codec.Default()
	if cfg.Voice.SampleRate > 0 {
		profile.SampleRate = cfg.Voice.SampleRate
	}
	if cfg.Voice.Bitrate > 0 {
		profile.Bitrate = cfg.Voice.Bitrate
	}
	if cfg.Voice.FrameMs > 0 {
		profile.FrameMs = cfg.Voice.FrameMs
	}
	if cfg.Voice.Complexity > 0 {
		profile.Complexity = cfg.Voice.Complexity
	}
	procCfg := process.DefaultConfig(profile.SampleRate, profile.FrameSamples())
	procCfg.Backend = process.Backend(cfg.Audio.Processing.Backend)
	procCfg.EchoCancel = cfg.Audio.Processing.EchoCancel
	procCfg.EchoTailMs = cfg.Audio.Processing.EchoTailMs
	procCfg.EchoDelayMs = cfg.Audio.Processing.EchoDelayMs
	procCfg.NoiseSuppress = cfg.Audio.Processing.NoiseSuppress
	procCfg.NSLevel = cfg.Audio.Processing.NSLevel
	procCfg.AGC = cfg.Audio.Processing.AGC
	procCfg.AGCTargetDBFS = cfg.Audio.Processing.AGCTargetDBFS
	procCfg.Dereverb = cfg.Audio.Processing.Dereverb
	procCfg.HighPass = cfg.Audio.Processing.HighPass

	outVol := cfg.Audio.Output.Volume
	if outVol <= 0 {
		outVol = 100
	}

	eng, err := audio.NewEngine(audio.EngineConfig{
		Profile:      profile,
		Processing:   procCfg,
		InputGainDB:  cfg.Audio.Input.GainDB,
		OutputVolume: outVol,
		Input: audio.Descriptor{
			Backend:    cfg.Audio.Input.Backend,
			DeviceID:   cfg.Audio.Input.DeviceID,
			DeviceName: cfg.Audio.Input.DeviceName,
		},
		Output: audio.Descriptor{
			Backend:    cfg.Audio.Output.Backend,
			DeviceID:   cfg.Audio.Output.DeviceID,
			DeviceName: cfg.Audio.Output.DeviceName,
		},
		Enumerator: c.enum,
		Sink:       clientVoiceSink{c: cl},
		Log:        c.log,
		Callbacks: audio.EngineCallbacks{
			AllowTX:     c.allowTX,
			OnSpeech:    c.onSpeech,
			SuppressRX:  c.comm.ShouldSuppressRX,
			ChimeState:  c.chimeState,
			OnChimeDone: c.onChimeDone,
		},
	})
	if err != nil {
		return err
	}
	ctx := c.runCtx
	if ctx == nil {
		ctx = context.Background()
	}
	if err := eng.Start(ctx); err != nil {
		_ = eng.Close()
		return err
	}
	c.mu.Lock()
	c.engine = eng
	c.mu.Unlock()
	c.applyVoiceModeSideEffects()
	return nil
}

func (c *Controller) stopAudio() {
	c.mu.Lock()
	eng := c.engine
	c.engine = nil
	c.mu.Unlock()
	if eng != nil {
		_ = eng.Close()
	}
}

func (c *Controller) allowTX() bool {
	mode := c.cfg.Clone().Voice.Mode
	switch mode {
	case "always_on":
		return c.Client() != nil && c.Client().IsConnected()
	case "push_to_talk":
		return c.transmitting.Load()
	case "communicator":
		return c.comm.ShouldTransmit()
	default:
		return c.transmitting.Load()
	}
}

// applyVoiceModeSideEffects syncs client/LED/VAD bookkeeping after connect or mode change.
// Per-frame TX gate hold is handled in Session.ProcessCaptureFrame when allowTX is true.
func (c *Controller) applyVoiceModeSideEffects() {
	mode := c.cfg.Clone().Voice.Mode
	cl := c.Client()
	eng := c.engineSnapshot()
	switch mode {
	case "always_on":
		c.transmitting.Store(true)
		c.leds.SetTransmit(true)
		if eng != nil {
			eng.ForceTransmit()
		}
		if cl != nil {
			cl.StartTransmit()
		}
	case "push_to_talk":
		tx := c.transmitting.Load()
		c.leds.SetTransmit(tx)
		if eng != nil {
			if tx {
				eng.ForceTransmit()
			} else {
				eng.ForceStop()
			}
		}
		if cl != nil {
			if tx {
				cl.StartTransmit()
			} else {
				cl.StopTransmit()
			}
		}
	case "communicator":
		tx := c.comm.ShouldTransmit()
		c.transmitting.Store(tx)
		c.leds.SetTransmit(tx)
		if eng != nil {
			if tx {
				eng.ForceTransmit()
			} else {
				eng.ForceStop()
			}
		}
		if cl != nil {
			if tx {
				cl.StartTransmit()
			} else {
				cl.StopTransmit()
			}
		}
	}
}

func (c *Controller) onSpeech(speech bool) {
	if c.cfg.Clone().Voice.Mode == "communicator" {
		c.comm.OnVoiceActivity(speech)
		c.leds.SetTransmit(c.comm.ShouldTransmit())
		c.transmitting.Store(c.comm.ShouldTransmit())
	}
}

func (c *Controller) chimeState() audio.ChimeKind {
	switch c.comm.State() {
	case communicator.OpenChime:
		return audio.ChimeOpen
	case communicator.CloseChime:
		return audio.ChimeClose
	default:
		return audio.ChimeNone
	}
}

func (c *Controller) onChimeDone() {
	c.comm.OnChimeDone()
	c.log.Info("communicator chime done", "state", c.comm.State().String())
	tx := c.comm.ShouldTransmit()
	c.transmitting.Store(tx)
	c.leds.SetTransmit(tx)
	if cl := c.Client(); cl != nil {
		if tx {
			cl.StartTransmit()
			if eng := c.engineSnapshot(); eng != nil {
				eng.ForceTransmit()
			}
		} else {
			cl.StopTransmit()
			if eng := c.engineSnapshot(); eng != nil {
				eng.ForceStop()
			}
		}
	}
}

func (c *Controller) engineSnapshot() *audio.Engine {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.engine
}

func (c *Controller) handleMumbleEvent(ev mumbleclient.Event) {
	switch e := ev.(type) {
	case mumbleclient.PingRTT:
		c.rttMs.Store(e.RTT.Milliseconds())
	case mumbleclient.Disconnected:
		reason := ""
		if e.Err != nil {
			reason = e.Err.Error()
		}
		c.log.Info("disconnected", "reason", reason)
		c.stopAudio()
		c.comm.Cancel()
		c.transmitting.Store(false)
		c.leds.SetOnline(false)
		c.leds.SetParticipants(false)
		c.leds.SetTransmit(false)
		c.emit(map[string]any{"type": "disconnected", "reason": reason})
	case mumbleclient.UserAdd, mumbleclient.UserRemove, mumbleclient.UserUpdate:
		c.updateParticipantLED()
		c.emit(map[string]any{"type": "roster"})
	case mumbleclient.TextMessage:
		author := c.resolveAuthor(e.Actor)
		msg := c.chat.append(ChatMessage{
			Actor:      e.Actor,
			Author:     author,
			Message:    e.Message,
			ChannelIDs: append([]uint32(nil), e.ChannelIDs...),
			Sessions:   append([]uint32(nil), e.Sessions...),
			TreeIDs:    append([]uint32(nil), e.TreeIDs...),
		})
		c.log.Info("chat received", "actor", e.Actor, "author", author, "channels", e.ChannelIDs)
		c.emit(map[string]any{
			"type":        "text",
			"id":          msg.ID,
			"at":          msg.At,
			"actor":       e.Actor,
			"author":      author,
			"message":     e.Message,
			"channel_ids": e.ChannelIDs,
			"sessions":    e.Sessions,
			"tree_ids":    e.TreeIDs,
		})
	case mumbleclient.VoicePacket:
		eng := c.engineSnapshot()
		if eng == nil {
			now := time.Now().UnixNano()
			prev := c.noEngWarnNS.Load()
			if prev == 0 || now-prev > int64(5*time.Second) {
				if c.noEngWarnNS.CompareAndSwap(prev, now) {
					c.log.Warn("voice rx dropped", "reason", "no_audio_engine")
				}
			}
			return
		}
		eng.HandleVoicePacket(e)
	case mumbleclient.TransportChanged:
		c.log.Info("voice transport", "transport", e.Transport.String())
		c.emit(map[string]any{"type": "transport", "transport": e.Transport.String()})
	case mumbleclient.Connected:
		c.log.Info("server sync", "session", e.Session)
		c.leds.SetOnline(true)
		// Restart audio after auto-reconnect (initial Dial also starts audio after Dial returns;
		// startAudio is idempotent via stop+start). Client is already stored for reconnects.
		if cl := c.Client(); cl != nil {
			if err := c.startAudioSafe(cl); err != nil {
				c.log.Warn("audio engine start failed", "err", err, "phase", "connected")
			}
		} else {
			c.log.Debug("connected before client assigned; dialOnce will start audio")
		}
	default:
		c.emit(map[string]any{"type": "event"})
	}
}

func (c *Controller) resolveAuthor(actor uint32) string {
	cl := c.Client()
	if cl == nil {
		return ""
	}
	if u, ok := cl.Roster().UserBySession(actor); ok {
		return u.Name
	}
	return ""
}

func (c *Controller) updateParticipantLED() {
	cl := c.Client()
	if cl == nil {
		return
	}
	users := cl.Roster().UsersInChannel(cl.ChannelID())
	c.leds.SetParticipants(len(users) > 1)
}

func (c *Controller) triggerLoop(ctx context.Context, events <-chan ptt.Event) {
	defer c.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			c.onTrigger(ev)
		}
	}
}

func (c *Controller) onTrigger(ev ptt.Event) {
	mode := c.cfg.Clone().Voice.Mode
	c.log.Info("ptt", "pressed", ev.Pressed, "source", ev.Source, "mode", mode)
	switch mode {
	case "communicator":
		if ev.Pressed {
			prev := c.comm.State()
			c.comm.Toggle()
			c.log.Info("communicator toggle", "from", prev.String(), "to", c.comm.State().String())
		}
	case "push_to_talk":
		c.transmitting.Store(ev.Pressed)
		c.leds.SetTransmit(ev.Pressed)
		if eng := c.engineSnapshot(); eng != nil {
			if ev.Pressed {
				eng.ForceTransmit()
			} else {
				eng.ForceStop()
			}
		}
		if cl := c.Client(); cl != nil {
			if ev.Pressed {
				cl.StartTransmit()
			} else {
				cl.StopTransmit()
			}
		}
	case "always_on":
		// Hardware/REST PTT is ignored in always-on; keep transmitting.
	}
	c.emit(map[string]any{"type": "ptt", "pressed": ev.Pressed, "source": ev.Source, "at": time.Now().UTC()})
}
