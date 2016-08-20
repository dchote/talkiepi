package talkiepi

import (
	"crypto/tls"

	"github.com/dchote/go-rpio"
	"github.com/layeh/barnard/uiterm"
	"github.com/layeh/gumble/gumble"
	"github.com/layeh/gumble/gumbleopenal"
)

type Talkiepi struct {
	Config *gumble.Config
	Client *gumble.Client

	Address   string
	TLSConfig tls.Config

	Stream *gumbleopenal.Stream

	GPIO *rpio.GPIO

	Ui            *uiterm.Ui
	UiOutput      uiterm.Textview
	UiInput       uiterm.Textbox
	UiStatus      uiterm.Label
	UiTree        uiterm.Tree
	UiInputStatus uiterm.Label
}
