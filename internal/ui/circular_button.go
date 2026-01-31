package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// CircularButton 圆形按钮组件
type CircularButton struct {
	widget.BaseWidget
	icon       fyne.Resource
	onTapped   func()
	importance widget.Importance
	size       float32
	appState   *AppState
}

// NewCircularButton 创建新的圆形按钮
// 参数：
//   - icon: 图标资源
//   - onTapped: 点击回调函数
//   - size: 按钮尺寸（直径）
//   - appState: 应用状态，用于获取主题颜色
//
// 返回：圆形按钮实例
func NewCircularButton(icon fyne.Resource, onTapped func(), size float32, appState *AppState) *CircularButton {
	btn := &CircularButton{
		icon:       icon,
		onTapped:   onTapped,
		importance: widget.LowImportance,
		size:       size,
		appState:   appState,
	}
	btn.ExtendBaseWidget(btn)
	return btn
}

// SetImportance 设置按钮重要性
func (cb *CircularButton) SetImportance(importance widget.Importance) {
	cb.importance = importance
	cb.Refresh()
}

// SetIcon 设置图标
func (cb *CircularButton) SetIcon(icon fyne.Resource) {
	cb.icon = icon
	cb.Refresh()
}

// SetSize 设置按钮尺寸
func (cb *CircularButton) SetSize(size float32) {
	cb.size = size
	cb.Refresh()
}

// MinSize 返回最小尺寸
func (cb *CircularButton) MinSize() fyne.Size {
	return fyne.NewSize(cb.size, cb.size)
}

// CreateRenderer 创建渲染器
func (cb *CircularButton) CreateRenderer() fyne.WidgetRenderer {
	var bgColor color.Color
	if cb.appState != nil && cb.appState.App != nil {
		bgColor = CurrentThemeColor(cb.appState.App, theme.ColorNameButton)
	} else {
		bgColor = theme.Color(theme.ColorNameButton)
	}
	circle := canvas.NewCircle(bgColor)
	circle.StrokeWidth = 0

	// 创建图标
	iconImg := canvas.NewImageFromResource(cb.icon)
	iconImg.FillMode = canvas.ImageFillContain

	return &circularButtonRenderer{
		button:  cb,
		circle:  circle,
		iconImg: iconImg,
		objects: []fyne.CanvasObject{circle, iconImg},
	}
}

// Tapped 处理点击事件
func (cb *CircularButton) Tapped(*fyne.PointEvent) {
	if cb.onTapped != nil {
		cb.onTapped()
	}
}

// circularButtonRenderer 圆形按钮渲染器
type circularButtonRenderer struct {
	button  *CircularButton
	circle  *canvas.Circle
	iconImg *canvas.Image
	objects []fyne.CanvasObject
}

// Layout 布局
func (r *circularButtonRenderer) Layout(size fyne.Size) {
	// 圆形背景占满整个区域
	r.circle.Resize(size)
	r.circle.Move(fyne.NewPos(0, 0))

	// 图标居中，大小为按钮的 60%
	iconSize := size.Width * 0.6
	if size.Height < size.Width {
		iconSize = size.Height * 0.6
	}

	iconX := (size.Width - iconSize) / 2
	iconY := (size.Height - iconSize) / 2

	r.iconImg.Resize(fyne.NewSize(iconSize, iconSize))
	r.iconImg.Move(fyne.NewPos(iconX, iconY))
}

// MinSize 返回最小尺寸
func (r *circularButtonRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.button.size, r.button.size)
}

// Objects 返回所有对象
func (r *circularButtonRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

// Refresh 刷新渲染
func (r *circularButtonRenderer) Refresh() {
	// 根据重要性设置颜色
	var bgColor color.Color
	switch r.button.importance {
	case widget.SuccessImportance:
		// 成功状态使用主题色（通常是绿色或蓝色，取决于主题）
		// 如果没有主题色，使用 PrimaryColor
		if r.button.appState != nil && r.button.appState.App != nil {
			bgColor = CurrentThemeColor(r.button.appState.App, theme.ColorNameSuccess)
		} else {
			bgColor = theme.Color(theme.ColorNameSuccess)
		}
	case widget.HighImportance:
		if r.button.appState != nil && r.button.appState.App != nil {
			bgColor = CurrentThemeColor(r.button.appState.App, theme.ColorNamePrimary)
		} else {
			bgColor = theme.Color(theme.ColorNamePrimary)
		}
	case widget.MediumImportance:
		if r.button.appState != nil && r.button.appState.App != nil {
			bgColor = CurrentThemeColor(r.button.appState.App, theme.ColorNameButton)
		} else {
			bgColor = theme.Color(theme.ColorNameButton)
		}
	case widget.LowImportance:
		if r.button.appState != nil && r.button.appState.App != nil {
			bgColor = CurrentThemeColor(r.button.appState.App, theme.ColorNameInputBackground)
		} else {
			bgColor = theme.Color(theme.ColorNameInputBackground)
		}
	default:
		if r.button.appState != nil && r.button.appState.App != nil {
			bgColor = CurrentThemeColor(r.button.appState.App, theme.ColorNameButton)
		} else {
			bgColor = theme.Color(theme.ColorNameButton)
		}
	}

	r.circle.FillColor = bgColor
	r.circle.StrokeColor = bgColor

	// 更新图标
	if r.button.icon != nil {
		r.iconImg.Resource = r.button.icon
	}

	r.circle.Refresh()
	r.iconImg.Refresh()
}

// Destroy 销毁渲染器
func (r *circularButtonRenderer) Destroy() {
	// 清理资源
}
