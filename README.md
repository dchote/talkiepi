# talkiepi

talkiepi is a fork of [barnard](https://github.com/layeh/barnard) for the Raspberry Pi.  It is a headless capable Mumble client written in Go, adapted for walkie talkie style use on the Pi using GPIO pins for input and LED display.

You can edit your pin assignments in talkiepi.go 
```go
const (
	OnlineLEDPin       uint = 18
	ParticipantsLEDPin uint = 23
	ButtonPin          uint = 24
)
```

There is an issue I am currently working on regarding openal's stopping and re-starting of an audio stream, which basically leaves this very broken right now until this bug is tracked down and fixed.


## Requirements

- [gumble](https://github.com/layeh/gumble/tree/master/gumble)
- [gumbleopenal](https://github.com/layeh/gumble/tree/master/gumbleopenal)
- [termbox-go](https://github.com/nsf/termbox-go)

## License

MPL 2.0

## Author

- Barnard Author - Tim Cooper (<tim.cooper@layeh.com>)
- talkiepi Adaption - Daniel Chote
