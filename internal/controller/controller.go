package controller

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dchote/talkiepi/internal/communicator"
	"github.com/dchote/talkiepi/internal/config"
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
	Participants int     `json:"participants"`
	Version      string  `json:"version"`
}

// Controller wires config, mumble client, triggers, LEDs, and updates.
type Controller struct {
	cfg     *config.Config
	log     *slog.Logger
	version string

	mu      sync.RWMutex
	client  *mumbleclient.Client
	comm    *communicator.Communicator
	leds    led.Indicator
	manual  *ptt.Manual
	mux     *ptt.Mux
	updater *update.Checker

	transmitting atomic.Bool
	connecting   atomic.Bool
	rttMs        atomic.Int64

	cancel context.CancelFunc
	wg     sync.WaitGroup

	listenersMu sync.RWMutex
	listeners   []chan any
}

// Options configures the controller.
type Options struct {
	Config  *config.Config
	Logger  *slog.Logger
	Version string
	LEDs    led.Indicator
	Updater *update.Checker
}

// New creates a Controller.
func New(opt Options) *Controller {
	if opt.Logger == nil {
		opt.Logger = slog.Default()
	}
	if opt.LEDs == nil {
		opt.LEDs = led.NewMemory()
	}
	manual := ptt.NewManual("rest")
	return &Controller{
		cfg:     opt.Config,
		log:     opt.Logger,
		version: opt.Version,
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
	events := make(chan ptt.Event, 16)
	if err := c.mux.Start(ctx, events); err != nil {
		c.log.Warn("ptt mux start", "err", err)
	}
	c.wg.Add(1)
	go c.triggerLoop(ctx, events)

	if c.cfg.Clone().Server.Address != "" {
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			if err := c.dialOnce(ctx); err != nil {
				c.log.Warn("initial connect failed", "err", err)
			}
		}()
	}
	return nil
}

// Stop shuts down the controller.
func (c *Controller) Stop() {
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
		Transmitting: c.transmitting.Load(),
		Version:      c.version,
		PingMs:       float64(c.rttMs.Load()),
		Transport:    "TCP",
	}
	c.mu.RLock()
	cl := c.client
	c.mu.RUnlock()
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
	}
	return st
}

// Connect starts or restarts the mumble connection using current config.
func (c *Controller) Connect(ctx context.Context) error {
	c.Disconnect()
	return c.dialOnce(ctx)
}

// Disconnect closes the mumble client.
func (c *Controller) Disconnect() {
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
	if err := c.cfg.ApplyPatch(patch); err != nil {
		return err
	}
	if err := c.cfg.Save(""); err != nil {
		c.log.Warn("config save", "err", err)
	}
	c.emit(map[string]any{"type": "config"})
	return nil
}

// Client returns the active mumble client (may be nil).
func (c *Controller) Client() *mumbleclient.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

// Updater returns the update checker (may be nil).
func (c *Controller) Updater() *update.Checker { return c.updater }

func (c *Controller) dialOnce(ctx context.Context) error {
	cfg := c.cfg.Clone()
	if cfg.Server.Address == "" {
		return nil
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
		mcfg.Username = "talkiepi"
	}
	cl := mumbleclient.New(mcfg)
	cl.On(c.handleMumbleEvent)
	if err := cl.Dial(ctx); err != nil {
		return err
	}
	c.mu.Lock()
	c.client = cl
	c.mu.Unlock()
	c.leds.SetOnline(true)
	c.emit(map[string]any{"type": "connected", "session": cl.Session()})
	return nil
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
		c.leds.SetOnline(false)
		c.leds.SetParticipants(false)
		c.emit(map[string]any{"type": "disconnected", "reason": reason})
	case mumbleclient.UserAdd, mumbleclient.UserRemove, mumbleclient.UserUpdate:
		c.updateParticipantLED()
		c.emit(map[string]any{"type": "roster"})
	case mumbleclient.TextMessage:
		c.emit(map[string]any{"type": "text", "actor": e.Actor, "message": e.Message})
	case mumbleclient.TransportChanged:
		c.emit(map[string]any{"type": "transport", "transport": e.Transport.String()})
	default:
		c.emit(map[string]any{"type": "event"})
	}
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
	switch mode {
	case "communicator":
		if ev.Pressed {
			c.comm.Toggle()
		}
	case "push_to_talk":
		c.transmitting.Store(ev.Pressed)
		c.leds.SetTransmit(ev.Pressed)
		if cl := c.Client(); cl != nil {
			if ev.Pressed {
				cl.StartTransmit()
			} else {
				cl.StopTransmit()
			}
		}
	}
	c.emit(map[string]any{"type": "ptt", "pressed": ev.Pressed, "source": ev.Source, "at": time.Now().UTC()})
}
