package uiterm

import (
	"strings"

	"github.com/nsf/termbox-go"
)

type Label struct {
	Text   string
	Fg, Bg Attribute

	ui             *Ui
	x0, y0, x1, y1 int
}

func (l *Label) uiInitialize(ui *Ui) {
	l.ui = ui
}

func (l *Label) uiSetActive(active bool) {
}

func (l *Label) uiSetBounds(x0, y0, x1, y1 int) {
	l.x0 = x0
	l.y0 = y0
	l.x1 = x1
	l.y1 = y1
	l.uiDraw()
}

func (l *Label) uiDraw() {
	l.ui.beginDraw()
	defer l.ui.endDraw()

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

func (l *Label) uiKeyEvent(mod Modifier, key Key) {
}

func (l *Label) uiCharacterEvent(chr rune) {
}
