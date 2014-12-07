package uiterm

type View interface {
	setActive(ui *Ui, active bool)
	setBounds(ui *Ui, x0, y0, x1, y1 int)
	draw(ui *Ui)
	keyEvent(ui *Ui, mod Modifier, key Key)
	characterEvent(ui *Ui, ch rune)
}
