package uiterm

import (
	"strings"

	"github.com/nsf/termbox-go"
)

type Textview struct {
	Lines       []string
	CurrentLine int
	Fg          Attribute
	Bg          Attribute

	parsedLines    []string

	ui *Ui
	x0, y0, x1, y1 int
}

func (t *Textview) uiInitialize(ui *Ui) {
	t.ui = ui
}

func (t *Textview) uiSetActive(active bool) {
}

func (t *Textview) uiSetBounds(x0, y0, x1, y1 int) {
	t.x0 = x0
	t.y0 = y0
	t.x1 = x1
	t.y1 = y1
	t.updateParsedLines()
}

func (t *Textview) ScrollUp() {
	if newLine := t.CurrentLine + 1; newLine < len(t.parsedLines) {
		t.CurrentLine = newLine
	}
}

func (t *Textview) ScrollDown() {
	if newLine := t.CurrentLine - 1; newLine >= 0 {
		t.CurrentLine = newLine
	}
}

func (t *Textview) ScrollTop() {
	if newLine := len(t.parsedLines) - 1; newLine > 0 {
		t.CurrentLine = newLine
	} else {
		t.CurrentLine = 0
	}
}

func (t *Textview) ScrollBottom() {
	t.CurrentLine = 0
}

func (t *Textview) updateParsedLines() {
	width := t.x1 - t.x0 - 3

	if t.Lines == nil || width <= 0 {
		t.parsedLines = nil
		return
	}

	parsed := make([]string, 0, len(t.Lines))
	for _, line := range t.Lines {
		current := ""
		chars := 0
		reader := strings.NewReader(line)
		for {
			if chars >= width {
				parsed = append(parsed, current)
				chars = 0
				current = ""
			}
			if reader.Len() <= 0 {
				if chars > 0 {
					parsed = append(parsed, current)
				}
				break
			}
			if ch, _, err := reader.ReadRune(); err == nil {
				current = current + string(ch)
				chars++
			}
		}
	}
	t.parsedLines = parsed
}

func (t *Textview) AddLine(line string) {
	t.Lines = append(t.Lines, line)
	t.updateParsedLines()
}

func (t *Textview) Clear() {
	t.Lines = nil
	t.CurrentLine = 0
	t.parsedLines = nil
}

func (t *Textview) uiDraw() {
	var reader *strings.Reader
	line := len(t.parsedLines) - 1 - t.CurrentLine
	if line < 0 {
		line = 0
	}
	totalLines := len(t.parsedLines)
	if totalLines == 0 {
		totalLines = 1
	}
	currentScrollLine := t.y1 - 1 - int((float32(t.CurrentLine)/float32(totalLines))*float32(t.y1-t.y0))
	for y := t.y1 - 1; y >= t.y0; y-- {
		if t.parsedLines != nil && line >= 0 {
			reader = strings.NewReader(t.parsedLines[line])
		} else {
			reader = nil
		}
		for x := t.x0; x < t.x1; x++ {
			var chr rune = ' '
			if x == t.x1-1 { // scrollbar
				if y == currentScrollLine {
					chr = '█'
				} else {
					chr = '░'
				}
			} else if x < t.x1-3 {
				if reader != nil {
					if ch, _, err := reader.ReadRune(); err == nil {
						chr = ch
					}
				}
			}
			termbox.SetCell(x, y, chr, termbox.Attribute(t.Fg), termbox.Attribute(t.Bg))
		}
		line--
	}
}

func (t *Textview) uiKeyEvent(mod Modifier, key Key) {
}

func (t *Textview) uiCharacterEvent(chr rune) {
}
