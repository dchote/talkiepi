package talkiepi

import (
	"crypto/tls"

	"github.com/dchote/gpio"
	"github.com/dchote/gumble/gumble"
	"github.com/dchote/gumble/gumbleopenal"
)

// Raspberry Pi GPIO pin assignments (CPU pin definitions)
const (
	OnlineLEDPin        uint = 18
	ParticipantsLEDPin  uint = 23
	TransmitLEDPin      uint = 24
	TransmitButtonPin   uint = 25
	VolumeUpButtonPin   uint = 22
	VolumeDownButtonPin uint = 27
	MaxVolume						int = 90
	MinVolume						int = 10
	VolumeIncrement			int = 10
)

type TalkieButton struct {
	Pin    gpio.Pin
	State  uint
	OnPress func ()
	OnRelease  func ()
}

type Talkiepi struct {
	Config *gumble.Config
	Client *gumble.Client

	Address   string
	TLSConfig tls.Config

	ConnectAttempts uint

	Stream *gumbleopenal.Stream

	ChannelName    string
	IsConnected    bool
	IsTransmitting bool

	GPIOEnabled     bool
	OnlineLED       gpio.Pin
	ParticipantsLED gpio.Pin
	TransmitLED     gpio.Pin

	Buttons		[]TalkieButton
}
