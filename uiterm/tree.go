package uiterm

import (
	"strings"

	"github.com/nsf/termbox-go"
)

type TreeItem interface {
	TreeItemStyle(fg, bg Attribute, active bool) (Attribute, Attribute)
	String() string
}

type renderedTreeItem struct {
	//String string
	Level int
	Item  TreeItem
}

type Tree struct {
	Fg        Attribute
	Bg        Attribute
	Generator func(item TreeItem) []TreeItem
	Listener  func(ui *Ui, tree *Tree, item TreeItem)

	lines      []renderedTreeItem
	activeLine int

	ui             *Ui
	active         bool
	x0, y0, x1, y1 int
}

func bounded(i, lower, upper int) int {
	if i < lower {
		return lower
	}
	if i > upper {
		return upper
	}
	return i
}

func (t *Tree) uiInitialize(ui *Ui) {
	t.ui = ui
}

func (t *Tree) uiSetActive(active bool) {
	t.active = active
	t.uiDraw()
}

func (t *Tree) uiSetBounds(x0, y0, x1, y1 int) {
	t.x0 = x0
	t.y0 = y0
	t.x1 = x1
	t.y1 = y1
	t.uiDraw()
}

func (t *Tree) Rebuild() {
	if t.Generator == nil {
		t.lines = []renderedTreeItem{}
		return
	}

	lines := []renderedTreeItem{}
	for _, item := range t.Generator(nil) {
		children := t.rebuild_rec(item, 0)
		if children != nil {
			lines = append(lines, children...)
		}
	}
	t.lines = lines
	t.activeLine = bounded(t.activeLine, 0, len(t.lines)-1)
	t.uiDraw()
}

func (t *Tree) rebuild_rec(parent TreeItem, level int) []renderedTreeItem {
	if parent == nil {
		return nil
	}
	lines := []renderedTreeItem{
		renderedTreeItem{
			Level: level,
			Item:  parent,
		},
	}
	for _, item := range t.Generator(parent) {
		children := t.rebuild_rec(item, level+1)
		if children != nil {
			lines = append(lines, children...)
		}
	}
	return lines
}

func (t *Tree) uiDraw() {
	t.ui.beginDraw()
	defer t.ui.endDraw()

	if t.lines == nil {
		t.Rebuild()
	}

	line := 0
	for y := t.y0; y < t.y1; y++ {
		var reader *strings.Reader
		var item TreeItem
		level := 0
		if line < len(t.lines) {
			item = t.lines[line].Item
			level = t.lines[line].Level
			reader = strings.NewReader(item.String())
		}
		for x := t.x0; x < t.x1; x++ {
			var chr rune = ' '
			fg := t.Fg
			bg := t.Bg
			dx := x - t.x0
			dy := y - t.y0
			if reader != nil && level*2 <= dx {
				if ch, _, err := reader.ReadRune(); err == nil {
					chr = ch
					fg, bg = item.TreeItemStyle(fg, bg, t.active && t.activeLine == dy)
				}
			}
			termbox.SetCell(x, y, chr, termbox.Attribute(fg), termbox.Attribute(bg))
		}
		line++
	}
}

func (t *Tree) uiKeyEvent(mod Modifier, key Key) {
	switch key {
	case KeyArrowUp:
		t.activeLine = bounded(t.activeLine-1, 0, len(t.lines)-1)
	case KeyArrowDown:
		t.activeLine = bounded(t.activeLine+1, 0, len(t.lines)-1)
	case KeyEnter:
		if t.Listener != nil && t.activeLine >= 0 && t.activeLine < len(t.lines) {
			t.Listener(t.ui, t, t.lines[t.activeLine].Item)
		}
	}
	t.uiDraw()
}

func (t *Tree) uiCharacterEvent(ch rune) {
}
