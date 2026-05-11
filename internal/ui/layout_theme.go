package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// innerPadding 返回当前主题下的内边距（theme.SizeNameInnerPadding），供 newPaddedWithSize、compactVBoxLayout 使用。
func innerPadding(appState *AppState) float32 {
	if appState != nil && appState.App != nil {
		return appState.App.Settings().Theme().Size(theme.SizeNameInnerPadding)
	}
	return theme.DefaultTheme().Size(theme.SizeNameInnerPadding)
}

type uniformPadLayout struct {
	padding float32
}

func (u uniformPadLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	if len(objs) == 0 {
		return
	}
	p := u.padding
	if p < 0 {
		p = 0
	}
	innerW := size.Width - 2*p
	innerH := size.Height - 2*p
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}
	objs[0].Move(fyne.NewPos(p, p))
	objs[0].Resize(fyne.NewSize(innerW, innerH))
}

func (u uniformPadLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	if len(objs) == 0 {
		return fyne.NewSize(0, 0)
	}
	p := u.padding
	if p < 0 {
		p = 0
	}
	m := objs[0].MinSize()
	return fyne.NewSize(m.Width+2*p, m.Height+2*p)
}

// newPaddedWithSize 按指定四边留白包裹单个子控件，padding 应来自 innerPadding。
func newPaddedWithSize(content fyne.CanvasObject, padding float32) fyne.CanvasObject {
	return container.New(&uniformPadLayout{padding: padding}, content)
}

// compactVBoxLayout 纵向排列子控件，相邻可见子项之间使用固定 spacing（通常取 innerPadding）。
type compactVBoxLayout struct {
	spacing float32
}

func (l compactVBoxLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	var y float32
	first := true
	for _, child := range objs {
		if child == nil || !child.Visible() {
			continue
		}
		if !first {
			y += l.spacing
		}
		first = false
		h := child.MinSize().Height
		child.Move(fyne.NewPos(0, y))
		child.Resize(fyne.NewSize(size.Width, h))
		y += h
	}
}

func (l compactVBoxLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var w, h float32
	first := true
	for _, child := range objs {
		if child == nil || !child.Visible() {
			continue
		}
		if !first {
			h += l.spacing
		}
		first = false
		ms := child.MinSize()
		w = max(w, ms.Width)
		h += ms.Height
	}
	return fyne.NewSize(w, h)
}

// newCompactVBox 垂直排列子控件，子项间距为 spacing。
func newCompactVBox(spacing float32, objects ...fyne.CanvasObject) *fyne.Container {
	return container.New(&compactVBoxLayout{spacing: spacing}, objects...)
}
