package uiterm

type View interface {
	uiInitialize(ui *Ui)
	setActive(active bool)
	setBounds(x0, y0, x1, y1 int)
	draw()
	keyEvent(mod Modifier, key Key)
	characterEvent(ch rune)
}
