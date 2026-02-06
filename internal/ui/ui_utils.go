package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const (
	SpacingSmall  = 4.0
	SpacingMedium = 8.0
	SpacingLarge  = 12.0
)

func NewSpacer(width float32) fyne.CanvasObject {
	_ = width
	return layout.NewSpacer()
}

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

// compactVBoxLayout 紧凑的 VBox 布局，减少组件间距
type compactVBoxLayout struct {
	spacing float32
}

func (c compactVBoxLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	y := float32(0)
	for _, obj := range objects {
		if obj == nil {
			continue
		}
		objMin := obj.MinSize()
		objHeight := objMin.Height
		if objHeight < 0 {
			objHeight = 0
		}
		obj.Resize(fyne.NewSize(size.Width, objHeight))
		obj.Move(fyne.NewPos(0, y))
		y += objHeight + c.spacing
	}
}

func (c compactVBoxLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	width := float32(0)
	height := float32(0)
	for i, obj := range objects {
		if obj == nil {
			continue
		}
		objMin := obj.MinSize()
		if objMin.Width > width {
			width = objMin.Width
		}
		height += objMin.Height
		if i < len(objects)-1 {
			height += c.spacing
		}
	}
	return fyne.NewSize(width, height)
}

// paddedLayout 自定义内边距布局
type paddedLayout struct {
	padding float32
}

func (p paddedLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 1 {
		return
	}
	obj := objects[0]
	if obj == nil {
		return
	}
	contentWidth := size.Width - 2*p.padding
	contentHeight := size.Height - 2*p.padding
	if contentWidth < 0 {
		contentWidth = 0
	}
	if contentHeight < 0 {
		contentHeight = 0
	}
	obj.Resize(fyne.NewSize(contentWidth, contentHeight))
	obj.Move(fyne.NewPos(p.padding, p.padding))
}

func (p paddedLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) != 1 || objects[0] == nil {
		return fyne.NewSize(0, 0)
	}
	min := objects[0].MinSize()
	return fyne.NewSize(min.Width+2*p.padding, min.Height+2*p.padding)
}

// newPaddedWithSize 使用指定间距创建带内边距的容器
func newPaddedWithSize(content fyne.CanvasObject, padding float32) fyne.CanvasObject {
	if content == nil {
		// 如果内容为 nil，返回一个空的容器
		return container.NewWithoutLayout()
	}
	c := container.NewWithoutLayout(content)
	c.Layout = paddedLayout{padding: padding}
	return c
}

// noSpacingBorderLayout 无间距的 Border 布局，移除 headerBar 下方的多余空白
type noSpacingBorderLayout struct{}

func (n noSpacingBorderLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 5 {
		return
	}
	top := objects[0]
	bottom := objects[1]
	left := objects[2]
	right := objects[3]
	center := objects[4]

	topHeight := float32(0)
	if top != nil {
		topMin := top.MinSize()
		topHeight = topMin.Height
		if topHeight > size.Height {
			topHeight = size.Height
		}
		top.Resize(fyne.NewSize(size.Width, topHeight))
		top.Move(fyne.NewPos(0, 0))
	}

	bottomHeight := float32(0)
	if bottom != nil {
		bottomMin := bottom.MinSize()
		bottomHeight = bottomMin.Height
		if bottomHeight > size.Height-topHeight {
			bottomHeight = size.Height - topHeight
		}
		if bottomHeight < 0 {
			bottomHeight = 0
		}
		bottom.Resize(fyne.NewSize(size.Width, bottomHeight))
		bottom.Move(fyne.NewPos(0, size.Height-bottomHeight))
	}

	leftWidth := float32(0)
	if left != nil {
		leftMin := left.MinSize()
		leftWidth = leftMin.Width
		if leftWidth > size.Width {
			leftWidth = size.Width
		}
		centerHeight := size.Height - topHeight - bottomHeight
		if centerHeight < 0 {
			centerHeight = 0
		}
		left.Resize(fyne.NewSize(leftWidth, centerHeight))
		left.Move(fyne.NewPos(0, topHeight))
	}

	rightWidth := float32(0)
	if right != nil {
		rightMin := right.MinSize()
		rightWidth = rightMin.Width
		if rightWidth > size.Width-leftWidth {
			rightWidth = size.Width - leftWidth
		}
		if rightWidth < 0 {
			rightWidth = 0
		}
		centerHeight := size.Height - topHeight - bottomHeight
		if centerHeight < 0 {
			centerHeight = 0
		}
		right.Resize(fyne.NewSize(rightWidth, centerHeight))
		right.Move(fyne.NewPos(size.Width-rightWidth, topHeight))
	}

	if center != nil {
		centerWidth := size.Width - leftWidth - rightWidth
		centerHeight := size.Height - topHeight - bottomHeight
		if centerWidth < 0 {
			centerWidth = 0
		}
		if centerHeight < 0 {
			centerHeight = 0
		}
		center.Resize(fyne.NewSize(centerWidth, centerHeight))
		center.Move(fyne.NewPos(leftWidth, topHeight))
	}
}

func (n noSpacingBorderLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) != 5 {
		return fyne.NewSize(0, 0)
	}
	topMin := float32(0)
	bottomMin := float32(0)
	leftMin := float32(0)
	rightMin := float32(0)
	centerMin := fyne.NewSize(0, 0)

	if objects[0] != nil {
		topMin = objects[0].MinSize().Height
	}
	if objects[1] != nil {
		bottomMin = objects[1].MinSize().Height
	}
	if objects[2] != nil {
		leftMin = objects[2].MinSize().Width
	}
	if objects[3] != nil {
		rightMin = objects[3].MinSize().Width
	}
	if objects[4] != nil {
		centerMin = objects[4].MinSize()
	}

	width := leftMin + centerMin.Width + rightMin
	height := topMin + centerMin.Height + bottomMin
	return fyne.NewSize(width, height)
}

