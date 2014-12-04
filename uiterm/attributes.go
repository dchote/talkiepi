package uiterm

/*
 * Source: https://godoc.org/github.com/nsf/termbox-go
 */

type Attribute int

const (
	ColorDefault Attribute = iota
	ColorBlack
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
)

const (
	AttrBold Attribute = 1 << (iota + 4)
	AttrUnderline
	AttrReverse
)
