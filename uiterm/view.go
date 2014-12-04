package uiterm

type View interface {
	SetBounds(ui *Ui, x0, y0, x1, y1 int)
	Draw(ui *Ui)
	SetActive(ui *Ui, active bool)
	KeyEvent(ui *Ui, mod Modifier, key Key)
	CharacterEvent(ui *Ui, ch rune)
}
