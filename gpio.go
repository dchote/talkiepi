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

	TransmitButtonPinPullUp := rpio.Pin(TransmitButtonPin)
	TransmitButtonPinPullUp.PullUp()

	rpio.Close()


	b.Buttons = []TalkieButton{
		TalkieButton{
			Pin: gpio.NewInput(TransmitButtonPin),
			State: 1,
			OnPress: func() {
				fmt.Printf("Transmit is pressed\n")
				b.TransmitStart()
			},
			OnRelease: func() {
				fmt.Printf("Transmit is released\n")
				b.TransmitStop()
			},
		},
	}

	// unfortunately the gpio watcher stuff doesnt work for me in this context, so we have to poll the buttons instead
	go func() {
		for {

			for i := 0; i < len(b.Buttons); i++ {
				currentState, err := b.Buttons[i].Pin.Read()

				if currentState != b.Buttons[i].State && err == nil {
					b.Buttons[i].State = currentState

					if b.Stream != nil {
						if b.Buttons[i].State == 1 {
							b.Buttons[i].OnRelease()
						} else {
							b.Buttons[i].OnPress()
						}
					}
				}
			}

			time.Sleep(500 * time.Millisecond)
		}
	}()

	// then we can do our gpio stuff
	b.OnlineLED = gpio.NewOutput(OnlineLEDPin, false)
	b.ParticipantsLED = gpio.NewOutput(ParticipantsLEDPin, false)
	b.TransmitLED = gpio.NewOutput(TransmitLEDPin, false)


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
