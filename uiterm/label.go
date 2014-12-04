package uiterm

import (
	"strings"

	"github.com/nsf/termbox-go"
)

type Label struct {
	Text string
	Fg   Attribute
	Bg   Attribute

	x0, y0, x1, y1 int
}

func (l *Label) SetActive(ui *Ui, active bool) {
}

func (l *Label) SetBounds(ui *Ui, x0, y0, x1, y1 int) {
	l.x0 = x0
	l.y0 = y0
	l.x1 = x1
	l.y1 = y1
}

func (l *Label) Draw(ui *Ui) {
	reader := strings.NewReader(l.Text)
	for y := l.y0; y < l.y1; y++ {
		for x := l.x0; x < l.x1; x++ {
			var chr rune
			if ch, _, err := reader.ReadRune(); err != nil {
				chr = ' '
			} else {
				chr = ch
			}
			termbox.SetCell(x, y, chr, termbox.Attribute(l.Fg), termbox.Attribute(l.Bg))
		}
	}
}

func (l *Label) KeyEvent(ui *Ui, mod Modifier, key Key) {
}

func (l *Label) CharacterEvent(ui *Ui, chr rune) {
}
