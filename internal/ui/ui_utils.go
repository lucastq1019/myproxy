package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func NewButtonWithIcon(text string, icon fyne.Resource, onTapped func()) *widget.Button {
	btn := widget.NewButton(text, onTapped)
	if icon != nil {
		btn.SetIcon(icon)
	}
	return btn
}

func NewIconButton(icon fyne.Resource, onTapped func()) *widget.Button {
	btn := widget.NewButton("", onTapped)
	if icon != nil {
		btn.SetIcon(icon)
	}
	return btn
}

func NewTitleLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Bold: true}
	return label
}

func NewSeparator() *widget.Separator {
	return widget.NewSeparator()
}
