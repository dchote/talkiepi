package talkiepi

import (
	"fmt"
	"strings"
	"time"

	"github.com/dchote/gumble/gumble"
	"github.com/kennygrant/sanitize"
	"github.com/layeh/barnard/uiterm"
)

const (
	uiViewLogo        = "logo"
	uiViewTop         = "top"
	uiViewStatus      = "status"
	uiViewInput       = "input"
	uiViewInputStatus = "inputstatus"
	uiViewOutput      = "output"
	uiViewTree        = "tree"
)

func esc(str string) string {
	return sanitize.HTML(str)
}

func (b *Talkiepi) UpdateInputStatus(status string) {
	b.UiInputStatus.Text = status
	b.UiTree.Rebuild()
	b.Ui.Refresh()
}

func (b *Talkiepi) AddOutputLine(line string) {
	now := time.Now()
	b.UiOutput.AddLine(fmt.Sprintf("[%02d:%02d:%02d] %s", now.Hour(), now.Minute(), now.Second(), line))
}

func (b *Talkiepi) AddOutputMessage(sender *gumble.User, message string) {
	if sender == nil {
		b.AddOutputLine(message)
	} else {
		b.AddOutputLine(fmt.Sprintf("%s: %s", sender.Name, strings.TrimSpace(esc(message))))
	}
}

func (b *Talkiepi) OnVoiceToggle(ui *uiterm.Ui, key uiterm.Key) {
	if b.transmitting == true {
		b.TransmitStop()
	} else {
		b.TransmitStart()
	}
}

func (b *Talkiepi) StatusStartVoiceSend() {
	b.UiStatus.Fg = uiterm.ColorWhite | uiterm.AttrBold
	b.UiStatus.Bg = uiterm.ColorRed
	b.UiStatus.Text = "  Tx  "

	b.Ui.Refresh()
}

func (b *Talkiepi) StatusStopVoiceSend() {
	b.UiStatus.Text = " Idle "
	b.UiStatus.Fg = uiterm.ColorBlack
	b.UiStatus.Bg = uiterm.ColorWhite

	b.Ui.Refresh()
}

func (b *Talkiepi) OnQuitPress(ui *uiterm.Ui, key uiterm.Key) {
	b.Client.Disconnect()
	b.Ui.Close()
}

func (b *Talkiepi) OnClearPress(ui *uiterm.Ui, key uiterm.Key) {
	b.UiOutput.Clear()
}

func (b *Talkiepi) OnScrollOutputUp(ui *uiterm.Ui, key uiterm.Key) {
	b.UiOutput.ScrollUp()
}

func (b *Talkiepi) OnScrollOutputDown(ui *uiterm.Ui, key uiterm.Key) {
	b.UiOutput.ScrollDown()
}

func (b *Talkiepi) OnScrollOutputTop(ui *uiterm.Ui, key uiterm.Key) {
	b.UiOutput.ScrollTop()
}

func (b *Talkiepi) OnScrollOutputBottom(ui *uiterm.Ui, key uiterm.Key) {
	b.UiOutput.ScrollBottom()
}

func (b *Talkiepi) OnFocusPress(ui *uiterm.Ui, key uiterm.Key) {
	active := b.Ui.Active()
	if active == uiViewInput {
		b.Ui.SetActive(uiViewTree)
	} else if active == uiViewTree {
		b.Ui.SetActive(uiViewInput)
	}
}

func (b *Talkiepi) OnTextInput(ui *uiterm.Ui, textbox *uiterm.Textbox, text string) {
	if text == "" {
		return
	}
	if b.Client != nil && b.Client.Self != nil {
		b.Client.Self.Channel.Send(text, false)
		b.AddOutputMessage(b.Client.Self, text)
	}
}

func (b *Talkiepi) OnUiInitialize(ui *uiterm.Ui) {
	ui.Add(uiViewLogo, &uiterm.Label{
		Text: " barnard ",
		Fg:   uiterm.ColorWhite | uiterm.AttrBold,
		Bg:   uiterm.ColorMagenta,
	})

	ui.Add(uiViewTop, &uiterm.Label{
		Fg: uiterm.ColorWhite,
		Bg: uiterm.ColorBlue,
	})

	b.UiStatus = uiterm.Label{
		Text: " Idle ",
		Fg:   uiterm.ColorBlack,
		Bg:   uiterm.ColorWhite,
	}
	ui.Add(uiViewStatus, &b.UiStatus)

	b.UiInput = uiterm.Textbox{
		Fg:    uiterm.ColorWhite,
		Bg:    uiterm.ColorBlack,
		Input: b.OnTextInput,
	}
	ui.Add(uiViewInput, &b.UiInput)

	b.UiInputStatus = uiterm.Label{
		Fg: uiterm.ColorBlack,
		Bg: uiterm.ColorWhite,
	}
	ui.Add(uiViewInputStatus, &b.UiInputStatus)

	b.UiOutput = uiterm.Textview{
		Fg: uiterm.ColorWhite,
		Bg: uiterm.ColorBlack,
	}
	ui.Add(uiViewOutput, &b.UiOutput)

	b.UiTree = uiterm.Tree{
		Generator: b.TreeItem,
		Listener:  b.TreeItemSelect,
		Fg:        uiterm.ColorWhite,
		Bg:        uiterm.ColorBlack,
	}
	ui.Add(uiViewTree, &b.UiTree)

	b.Ui.AddKeyListener(b.OnFocusPress, uiterm.KeyTab)
	b.Ui.AddKeyListener(b.OnVoiceToggle, uiterm.KeyF1)
	b.Ui.AddKeyListener(b.OnQuitPress, uiterm.KeyF10)
	b.Ui.AddKeyListener(b.OnClearPress, uiterm.KeyCtrlL)
	b.Ui.AddKeyListener(b.OnScrollOutputUp, uiterm.KeyPgup)
	b.Ui.AddKeyListener(b.OnScrollOutputDown, uiterm.KeyPgdn)
	b.Ui.AddKeyListener(b.OnScrollOutputTop, uiterm.KeyHome)
	b.Ui.AddKeyListener(b.OnScrollOutputBottom, uiterm.KeyEnd)

	b.start()
}

func (b *Talkiepi) OnUiResize(ui *uiterm.Ui, width, height int) {
	ui.SetBounds(uiViewLogo, 0, 0, 9, 1)
	ui.SetBounds(uiViewTop, 9, 0, width-6, 1)
	ui.SetBounds(uiViewStatus, width-6, 0, width, 1)
	ui.SetBounds(uiViewInput, 0, height-1, width, height)
	ui.SetBounds(uiViewInputStatus, 0, height-2, width, height-1)
	ui.SetBounds(uiViewOutput, 0, 1, width-20, height-2)
	ui.SetBounds(uiViewTree, width-20, 1, width, height-2)
}
