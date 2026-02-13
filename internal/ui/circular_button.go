package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// CircularButton 圆形按钮组件。极简黑白灰：开启时黑色填充，关闭时边框黑+透明填充。
type CircularButton struct {
	widget.BaseWidget
	icon     fyne.Resource
	onTapped func()
	size     float32
	appState *AppState
	isActive bool // 是否处于开启状态（代理运行中），用于配色
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
		icon:     icon,
		onTapped: onTapped,
		size:     size,
		appState: appState,
	}
	btn.ExtendBaseWidget(btn)
	return btn
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

// SetActive 设置是否处于开启状态（代理运行中），用于切换 Primary / Separator 配色。
func (cb *CircularButton) SetActive(active bool) {
	if cb.isActive == active {
		return
	}
	cb.isActive = active
	cb.Refresh()
}

// MinSize 返回最小尺寸
func (cb *CircularButton) MinSize() fyne.Size {
	return fyne.NewSize(cb.size, cb.size)
}

// CreateRenderer 创建渲染器
func (cb *CircularButton) CreateRenderer() fyne.WidgetRenderer {
	var app fyne.App
	if cb.appState != nil {
		app = cb.appState.App
	}
	fill, stroke, strokeW := circularButtonStyle(app, cb.isActive)
	circle := canvas.NewCircle(fill)
	circle.StrokeColor = stroke
	circle.StrokeWidth = strokeW

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

	iconSize := size.Width
	if size.Height < size.Width {
		iconSize = size.Height
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
	var app fyne.App
	if r.button.appState != nil {
		app = r.button.appState.App
	}
	fill, stroke, strokeW := circularButtonStyle(app, r.button.isActive)
	r.circle.FillColor = fill
	r.circle.StrokeColor = stroke
	r.circle.StrokeWidth = strokeW

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

// circularButtonStyle 返回 (填充色, 描边色, 描边宽度)。开启=主题主按钮激活色，关闭=背景填充+主色描边。
func circularButtonStyle(app fyne.App, active bool) (fill, stroke color.Color, strokeWidth float32) {
	primary := CurrentThemeColor(app, theme.ColorNamePrimary)
	if active {
		return MainButtonActiveFill(app), primary, 0
	}
	return CurrentThemeColor(app, theme.ColorNameBackground), primary, 2
}
