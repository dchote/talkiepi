package controller

// GPIOPin is a selectable BCM GPIO line for PTT.
type GPIOPin struct {
	Line  int    `json:"line"`
	Label string `json:"label"`
}

// DefaultGPIOPins returns common Raspberry Pi BCM GPIO lines for PTT wiring.
func DefaultGPIOPins() []GPIOPin {
	return []GPIOPin{
		{Line: 5, Label: "BCM 5 (pin 29)"},
		{Line: 6, Label: "BCM 6 (pin 31)"},
		{Line: 12, Label: "BCM 12 (pin 32)"},
		{Line: 13, Label: "BCM 13 (pin 33)"},
		{Line: 16, Label: "BCM 16 (pin 36)"},
		{Line: 17, Label: "BCM 17 (pin 11)"},
		{Line: 18, Label: "BCM 18 (pin 12)"},
		{Line: 19, Label: "BCM 19 (pin 35)"},
		{Line: 20, Label: "BCM 20 (pin 38)"},
		{Line: 21, Label: "BCM 21 (pin 40)"},
		{Line: 22, Label: "BCM 22 (pin 15)"},
		{Line: 23, Label: "BCM 23 (pin 16)"},
		{Line: 24, Label: "BCM 24 (pin 18)"},
		{Line: 25, Label: "BCM 25 (pin 22)"},
		{Line: 26, Label: "BCM 26 (pin 37)"},
		{Line: 27, Label: "BCM 27 (pin 13)"},
	}
}
