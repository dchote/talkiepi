package talkiepi

import (
	"fmt"
	"github.com/layeh/gumble/gumble"
	"github.com/layeh/gumble/gumbleopenal"
	"github.com/layeh/gumble/gumbleutil"
	"net"
	"os"
)

func (b *Talkiepi) start() {
	b.Config.Attach(gumbleutil.AutoBitrate)
	b.Config.Attach(b)

	var err error
	_, err = gumble.DialWithDialer(new(net.Dialer), b.Address, b.Config, &b.TLSConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

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

	if err := b.GPIO.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	defer b.GPIO.Close()

	b.GPIO.Output(18)
	b.GPIO.Output(23)

	// turn off leds for now
	b.GPIO.Low(18)
	b.GPIO.Low(23)

	//b.GPIO.High(18)
}

func (b *Talkiepi) OnConnect(e *gumble.ConnectEvent) {
	b.Client = e.Client

	b.Ui.SetActive(uiViewInput)
	b.UiTree.Rebuild()
	b.Ui.Refresh()

	b.UpdateInputStatus(fmt.Sprintf("To: %s", e.Client.Self.Channel.Name))
	b.AddOutputLine(fmt.Sprintf("Connected to %s", b.Client.Conn.RemoteAddr()))
	if e.WelcomeMessage != nil {
		b.AddOutputLine(fmt.Sprintf("Welcome message: %s", esc(*e.WelcomeMessage)))
	}

	b.GPIO.High(18)

}

func (b *Talkiepi) OnDisconnect(e *gumble.DisconnectEvent) {
	var reason string
	switch e.Type {
	case gumble.DisconnectError:
		reason = "connection error"
	}
	if reason == "" {
		b.AddOutputLine("Disconnected")
	} else {
		b.AddOutputLine("Disconnected: " + reason)
	}

	b.UiTree.Rebuild()
	b.Ui.Refresh()

	/*
		if err := rpio.Open(); err != nil {
			return
		}

		defer rpio.Close()

		ledConnectedPin.Output()
		ledConnectedPin.Low()

		rpio.Close()
	*/
}

func (b *Talkiepi) OnTextMessage(e *gumble.TextMessageEvent) {
	b.AddOutputMessage(e.Sender, e.Message)
}

func (b *Talkiepi) OnUserChange(e *gumble.UserChangeEvent) {
	if e.Type.Has(gumble.UserChangeChannel) && e.User == b.Client.Self {
		b.UpdateInputStatus(fmt.Sprintf("To: %s", e.User.Channel.Name))
	}
	/*
		if len(e.User.Channel.Users) > 0 {
			ledParticipantsPin.High()
		} else {
			ledParticipantsPin.Low()
		}
	*/
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
