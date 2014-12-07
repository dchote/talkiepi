package uiterm

import (
	"github.com/nsf/termbox-go"
)

type LayoutFunc func(ui *Ui, width, height int)

type KeyListener func(ui *Ui, key Key)

type UiManager interface {
	OnUiInitialize(ui *Ui)
	OnUiResize(ui *Ui, width, height int)
}

type Ui struct {
	close   chan bool
	manager UiManager

	elements      map[string]*uiElement
	activeElement *uiElement

	keyListeners map[Key][]KeyListener

	fg Attribute
	bg Attribute
}

type uiElement struct {
	X0, Y0, X1, Y1 int
	View           View
}

func New(manager UiManager) *Ui {
	ui := &Ui{
		close:        make(chan bool, 10),
		elements:     make(map[string]*uiElement),
		manager:      manager,
		keyListeners: make(map[Key][]KeyListener),
	}
	return ui
}

func (ui *Ui) Close() {
	if termbox.IsInit {
		ui.close <- true
	}
}

func (ui *Ui) Refresh() {
	if termbox.IsInit {
		termbox.Clear(termbox.Attribute(ui.fg), termbox.Attribute(ui.bg))
		termbox.HideCursor()
		for _, element := range ui.elements {
			element.View.draw(ui)
		}
		termbox.Flush()
	}
}

func (ui *Ui) Active() View {
	return ui.activeElement.View
}

func (ui *Ui) SetActive(name string) {
	element, _ := ui.elements[name]
	if ui.activeElement != nil {
		ui.activeElement.View.setActive(ui, false)
	}
	ui.activeElement = element
	if element != nil {
		element.View.setActive(ui, true)
	}
	ui.Refresh()
}

func (ui *Ui) SetClear(fg, bg Attribute) {
	ui.fg = fg
	ui.bg = bg
}

func (ui *Ui) Run() error {
	if termbox.IsInit {
		return nil
	}
	if err := termbox.Init(); err != nil {
		return nil
	}
	defer termbox.Close()
	termbox.SetInputMode(termbox.InputAlt)

	events := make(chan termbox.Event)
	go func() {
		for {
			events <- termbox.PollEvent()
		}
	}()

	ui.manager.OnUiInitialize(ui)
	width, height := termbox.Size()
	ui.manager.OnUiResize(ui, width, height)
	ui.Refresh()

	for {
		select {
		case <-ui.close:
			return nil
		case event := <-events:
			switch event.Type {
			case termbox.EventResize:
				ui.manager.OnUiResize(ui, event.Width, event.Height)
				ui.Refresh()
			case termbox.EventKey:
				if event.Ch != 0 {
					ui.onCharacterEvent(event.Ch)
				} else {
					ui.onKeyEvent(Modifier(event.Mod), Key(event.Key))
				}
			}
		}
	}
}

func (ui *Ui) onCharacterEvent(ch rune) {
	if ui.activeElement != nil {
		ui.activeElement.View.characterEvent(ui, ch)
	}
}

func (ui *Ui) onKeyEvent(mod Modifier, key Key) {
	if ui.keyListeners[key] != nil {
		for _, listener := range ui.keyListeners[key] {
			listener(ui, key)
		}
	}
	if ui.activeElement != nil {
		ui.activeElement.View.keyEvent(ui, mod, key)
	}
}

func (ui *Ui) SetView(name string, x0, y0, x1, y1 int, view View) {
	if element, ok := ui.elements[name]; ok {
		element.X0 = x0
		element.Y0 = y0
		element.X1 = x1
		element.Y1 = y1
		view = element.View
	} else {
		ui.elements[name] = &uiElement{
			X0:   x0,
			Y0:   y0,
			X1:   x1,
			Y1:   y1,
			View: view,
		}
	}
	view.setBounds(ui, x0, y0, x1, y1)
}

func (ui *Ui) View(name string) View {
	if element, ok := ui.elements[name]; !ok {
		return nil
	} else {
		return element.View
	}
}

func (ui *Ui) AddKeyListener(listener KeyListener, key Key) {
	ui.keyListeners[key] = append(ui.keyListeners[key], listener)
}
