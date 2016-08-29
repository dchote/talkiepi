package talkiepi

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/dchote/gumble/gumble"
	"github.com/dchote/gumble/gumbleopenal"
	"github.com/dchote/gumble/gumbleutil"
)

func (b *Talkiepi) start() {
	b.Config.Attach(gumbleutil.AutoBitrate)
	b.Config.Attach(b)

	b.initGPIO()

	var err error
	_, err = gumble.DialWithDialer(new(net.Dialer), b.Address, b.Config, &b.TLSConfig)
	if err != nil {
		go func() {
			b.AddOutputLine("Connection to %s failed, attempting again in 10 seconds...", b.Address)
			time.Sleep(10 * time.Second)
			b.start()
		}()
		return
	}

	b.OpenStream()
}

func (b *Talkiepi) OpenStream() {
	// Audio
	if os.Getenv("ALSOFT_LOGLEVEL") == "" {
		os.Setenv("ALSOFT_LOGLEVEL", "0")
	}

	if stream, err := gumbleopenal.New(b.Client); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	} else {
		b.Stream = stream
	}
}

func (b *Talkiepi) ResetStream() {
	b.Stream.Destroy()

	// Sleep a bit and re-open
	time.Sleep(50 * time.Millisecond)

	b.OpenStream()
}

func (b *Talkiepi) TransmitStart() {
	b.transmitting = true

	// turn on our transmit LED
	b.LEDOn(b.transmitLED)

	b.StatusStartVoiceSend()
	b.Stream.StartSource()
}

func (b *Talkiepi) TransmitStop() {
	b.StatusStopVoiceSend()
	b.Stream.StopSource()

	b.LEDOff(b.transmitLED)

	b.transmitting = false
}

func (b *Talkiepi) OnConnect(e *gumble.ConnectEvent) {
	b.Client = e.Client

	// turn on our online LED
	b.LEDOn(b.onlineLED)

	b.Ui.SetActive(uiViewInput)

	b.UpdateInputStatus(fmt.Sprintf("To: %s", e.Client.Self.Channel.Name))
	b.AddOutputLine(fmt.Sprintf("Connected to %s", b.Client.Conn.RemoteAddr()))
	if e.WelcomeMessage != nil {
		b.AddOutputLine(fmt.Sprintf("Welcome message: %s", esc(*e.WelcomeMessage)))
	}

	if b.ChannelName != "" {
		b.ChangeChannel(b.ChannelName)
	}

	b.UiTree.Rebuild()
	b.Ui.Refresh()
}

func (b *Talkiepi) OnDisconnect(e *gumble.DisconnectEvent) {
	var reason string
	switch e.Type {
	case gumble.DisconnectError:
		reason = "connection error"
	}

	// turn off our online LED
	b.LEDOff(b.onlineLED)

	if reason == "" {
		b.AddOutputLine("Disconnected")
	} else {
		b.AddOutputLine("Disconnected: " + reason)
	}

	b.UiTree.Rebuild()
	b.Ui.Refresh()

	go func() {
		b.AddOutputLine("Attempting reconnect in 10 seconds")
		time.Sleep(10 * time.Second)
		b.start()
	}()
}

func (b *Talkiepi) ChangeChannel(ChannelName string) {
	channel := b.Client.Self.Channel.Find(ChannelName)
	if channel != nil {
		b.Client.Self.Move(channel)
	} else {
		b.AddOutputLine(fmt.Sprintf("Unable to find channel: %s", ChannelName))
	}
}

func (b *Talkiepi) OnTextMessage(e *gumble.TextMessageEvent) {
	b.AddOutputMessage(e.Sender, e.Message)
}

func (b *Talkiepi) OnUserChange(e *gumble.UserChangeEvent) {
	if e.Type.Has(gumble.UserChangeChannel) && e.User == b.Client.Self {
		b.UpdateInputStatus(fmt.Sprintf("To: %s", e.User.Channel.Name))
	}

	// If we have more than just ourselves in the channel, turn on the participants LED, otherwise, turn it off
	if len(e.User.Channel.Users) > 1 {
		b.LEDOn(b.participantsLED)
	} else {
		b.LEDOff(b.participantsLED)
	}

	b.UiTree.Rebuild()
	b.Ui.Refresh()
}

func (b *Talkiepi) OnChannelChange(e *gumble.ChannelChangeEvent) {
	b.UiTree.Rebuild()
	b.Ui.Refresh()
}

func (b *Talkiepi) OnPermissionDenied(e *gumble.PermissionDeniedEvent) {
	var info string
	switch e.Type {
	case gumble.PermissionDeniedOther:
		info = e.String
	case gumble.PermissionDeniedPermission:
		info = "insufficient permissions"
	case gumble.PermissionDeniedSuperUser:
		info = "cannot modify SuperUser"
	case gumble.PermissionDeniedInvalidChannelName:
		info = "invalid channel name"
	case gumble.PermissionDeniedTextTooLong:
		info = "text too long"
	case gumble.PermissionDeniedTemporaryChannel:
		info = "temporary channel"
	case gumble.PermissionDeniedMissingCertificate:
		info = "missing certificate"
	case gumble.PermissionDeniedInvalidUserName:
		info = "invalid user name"
	case gumble.PermissionDeniedChannelFull:
		info = "channel full"
	case gumble.PermissionDeniedNestingLimit:
		info = "nesting limit"
	}
	b.AddOutputLine(fmt.Sprintf("Permission denied: %s", info))
}

func (b *Talkiepi) OnUserList(e *gumble.UserListEvent) {
}

func (b *Talkiepi) OnACL(e *gumble.ACLEvent) {
}

func (b *Talkiepi) OnBanList(e *gumble.BanListEvent) {
}

func (b *Talkiepi) OnContextActionChange(e *gumble.ContextActionChangeEvent) {
}

func (b *Talkiepi) OnServerConfig(e *gumble.ServerConfigEvent) {
}
