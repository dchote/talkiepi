package talkiepi

import (
	"fmt"
	"time"

	"github.com/dchote/gpio"
	"github.com/stianeikeland/go-rpio"
)

func (b *Talkiepi) initGPIO() {
	// we need to pull in rpio to pullup our button pin
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		b.GPIOEnabled = false
		return
	} else {
		b.GPIOEnabled = true
	}

	buttonPinPullUp := rpio.Pin(ButtonPin)
	buttonPinPullUp.PullUp()

	rpio.Close()

	// unfortunately the gpio watcher stuff doesnt work for me in this context, so we have to poll the button instead
	b.button = gpio.NewInput(ButtonPin)
	go func() {
		for {
			currentState, err := b.button.Read()

			if currentState != b.buttonState && err == nil {
				b.buttonState = currentState

				if b.Stream != nil {
					if b.buttonState == 1 {
						fmt.Printf("Button is released\n")
						b.TransmitStop()
					} else {
						fmt.Printf("Button is pressed\n")
						b.TransmitStart()
					}
				}

			}

			time.Sleep(500 * time.Millisecond)
		}
	}()

	// then we can do our gpio stuff
	b.onlineLED = gpio.NewOutput(OnlineLEDPin, false)
	b.participantsLED = gpio.NewOutput(ParticipantsLEDPin, false)
	b.transmitLED = gpio.NewOutput(TransmitLEDPin, false)
}

func (b *Talkiepi) LEDOn(LED gpio.Pin) {
	if b.GPIOEnabled == false {
		return
	}

	LED.High()
}

func (b *Talkiepi) LEDOff(LED gpio.Pin) {
	if b.GPIOEnabled == false {
		return
	}

	LED.Low()
}
