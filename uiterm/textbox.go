package uiterm

import (
	"strings"
	"unicode/utf8"

	"github.com/nsf/termbox-go"
)

type Textbox struct {
	Text string
	Fg   Attribute
	Bg   Attribute

	Input func(ui *Ui, textbox *Textbox, text string)

	ui             *Ui
	active         bool
	x0, y0, x1, y1 int
}

func (t *Textbox) uiInitialize(ui *Ui) {
	t.ui = ui
}

func (t *Textbox) uiSetActive(active bool) {
	t.active = active
	t.uiDraw()
}

func (t *Textbox) uiSetBounds(x0, y0, x1, y1 int) {
	t.x0 = x0
	t.y0 = y0
	t.x1 = x1
	t.y1 = y1
	t.uiDraw()
}

func (t *Textbox) uiDraw() {
	t.ui.beginDraw()
	defer t.ui.endDraw()

	var setCursor = false
	reader := strings.NewReader(t.Text)
	for y := t.y0; y < t.y1; y++ {
		for x := t.x0; x < t.x1; x++ {
			var chr rune
			if ch, _, err := reader.ReadRune(); err != nil {
				if t.active && !setCursor {
					termbox.SetCursor(x, y)
					setCursor = true
				}
				chr = ' '
			} else {
				chr = ch
			}
			termbox.SetCell(x, y, chr, termbox.Attribute(t.Fg), termbox.Attribute(t.Bg))
		}
	}
}

func (t *Textbox) uiKeyEvent(mod Modifier, key Key) {
	redraw := false
	switch key {
	case KeyCtrlC:
		t.Text = ""
		redraw = true
	case KeyEnter:
		if t.Input != nil {
			t.Input(t.ui, t, t.Text)
		}
		t.Text = ""
		redraw = true
	case KeySpace:
		t.Text = t.Text + " "
		redraw = true
	case KeyBackspace:
	case KeyBackspace2:
		if len(t.Text) > 0 {
			if r, size := utf8.DecodeLastRuneInString(t.Text); r != utf8.RuneError {
				t.Text = t.Text[:len(t.Text)-size]
				redraw = true
			}
		}
	}
	if redraw {
		t.uiDraw()
	}
}

func (t *Textbox) uiCharacterEvent(chr rune) {
	t.Text = t.Text + string(chr)
	t.uiDraw()
}
