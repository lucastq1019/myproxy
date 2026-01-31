package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// 统一的间距常量
const (
	SpacingSmall  = 4.0  // 小间距
	SpacingMedium = 8.0  // 中间距
	SpacingLarge  = 12.0 // 大间距
)

// NewSpacer 创建间距（参数保留用于未来扩展，当前使用弹性间距）
func NewSpacer(width float32) fyne.CanvasObject {
	// 注意：当前使用弹性间距，width 参数保留用于未来可能需要固定宽度间距的场景
	_ = width // 明确标记参数当前未使用
	return layout.NewSpacer()
}

// NewButtonWithIcon 创建带图标的按钮
func NewButtonWithIcon(text string, icon fyne.Resource, onTapped func()) *widget.Button {
	btn := widget.NewButton(text, onTapped)
	if icon != nil {
		btn.SetIcon(icon)
	}
	return btn
}

// NewIconButton 创建纯图标按钮
func NewIconButton(icon fyne.Resource, onTapped func()) *widget.Button {
	btn := widget.NewButton("", onTapped)
	if icon != nil {
		btn.SetIcon(icon)
	}
	return btn
}

// NewStyledButton 创建带样式的按钮（圆角、图标等）
func NewStyledButton(text string, icon fyne.Resource, onTapped func()) *widget.Button {
	btn := NewButtonWithIcon(text, icon, onTapped)
	// 注意：Fyne 的按钮圆角由主题控制，这里主要是设置图标
	return btn
}

// NewTitleLabel 创建标题样式的标签（更大、加粗）
func NewTitleLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Bold: true}
	return label
}

// NewSubtitleLabel 创建副标题样式的标签
func NewSubtitleLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	return label
}

// NewSeparator 创建优化的分隔线
func NewSeparator() *widget.Separator {
	return widget.NewSeparator()
}

// NewStyledSelect 创建带样式的下拉框
func NewStyledSelect(options []string, onChanged func(string)) *widget.Select {
	selectWidget := widget.NewSelect(options, onChanged)
	return selectWidget
}
