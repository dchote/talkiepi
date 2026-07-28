package ptt

// GPIOConfig configures a GPIO PTT line.
type GPIOConfig struct {
	Chip      string
	Line      int
	ActiveLow bool
	Name      string
}
