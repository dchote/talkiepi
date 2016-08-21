# talkiepi

talkiepi is a fork of [barnard](https://github.com/layeh/barnard) for the Raspberry Pi.  It is a headless capable Mumble client written in Go, adapted for walkie talkie style use on the Pi using GPIO pins for input and LED display.

You can edit your pin assignments in `talkiepi.go`
```go
const (
	OnlineLEDPin       uint = 18
	ParticipantsLEDPin uint = 23
	ButtonPin          uint = 24
)
```

There is an issue I am currently working on regarding openal's stopping and re-starting of an audio stream, which basically leaves this very broken right now until this bug is tracked down and fixed.


## Requirements
Ensure you have the development packages for openal and libopus installed. On debian/raspbian you can do the following
```
apt-get install libopenal-dev libopus-dev
```

- [gumble](https://github.com/layeh/gumble)
- [gopus](https://github.com/layeh/gopus)
- [termbox-go](https://github.com/nsf/termbox-go)
- [gpio](https://github.com/dchote/gpio)
- [go-rpio](github.com/stianeikeland/go-rpio)

## Using talkiepi
I am using a cheap USB speakerphone (US Robotics). If you are using a USB speakerphone device, be sure to update your alsa config so that your system uses that instead of the built in audio on the raspberry pi. You can list your audio devices by running `aplay -l`, find the index of the device (likely 1) and then edit the alsa config (`/usr/share/alsa/alsa.conf`), changing the following:
```
defaults.ctl.card 1
defaults.pcm.card 1
```
_1 being the index of your device_

## License

MPL 2.0

## Author

- Barnard Author - Tim Cooper (<tim.cooper@layeh.com>)
- talkiepi Adaption - [Daniel Chote](https://github.com/dchote)
