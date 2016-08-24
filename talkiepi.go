package talkiepi

import (
	"crypto/tls"

	"github.com/dchote/gpio"
	"github.com/dchote/gumble/gumble"
	"github.com/dchote/gumble/gumbleopenal"
	"github.com/layeh/barnard/uiterm"
)

// Raspberry Pi GPIO pin assignments (CPU pin definitions)
const (
	OnlineLEDPin       uint = 18
	ParticipantsLEDPin uint = 23
	TransmitLEDPin     uint = 24
	ButtonPin          uint = 25
)

type Talkiepi struct {
	Config *gumble.Config
	Client *gumble.Client

	Address   string
	TLSConfig tls.Config

	Stream *gumbleopenal.Stream

	transmitting bool

	GPIOEnabled     bool
	onlineLED       gpio.Pin
	participantsLED gpio.Pin
	transmitLED     gpio.Pin
	button          gpio.Pin
	buttonState     uint

	Ui            *uiterm.Ui
	UiOutput      uiterm.Textview
	UiInput       uiterm.Textbox
	UiStatus      uiterm.Label
	UiTree        uiterm.Tree
	UiInputStatus uiterm.Label
}
