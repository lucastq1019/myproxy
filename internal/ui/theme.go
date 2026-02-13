package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// MonochromeTheme 实现 Fyne 主题接口。
// 极简黑白灰 + 状态强调色：交互控件黑白灰，仅状态反馈用绿/红/橙。
type MonochromeTheme struct {
	variant fyne.ThemeVariant
}

// 浅色模式 - 极简黑白灰（背景偏白）
const (
	LightBackground   = "#FFFFFF" // 页面最底层
	LightHeader       = "#FCFCFC" // 顶栏（接近纯白）
	LightPrimary      = "#000000" // 主操作（主开关、选中项）
	LightInputButton  = "#FCFCFC" // 输入框/卡片/默认按钮
	LightSeparator    = "#EBEBEB" // 分隔线（更浅）
	LightForeground   = "#212121" // 正文
	LightPlaceholder  = "#9E9E9E" // 占位符/次要文字
	LightSuccess      = "#4CAF50" // 成功（绿条、低延迟）
	LightError        = "#F44336" // 错误
	LightWarning      = "#FF9800" // 警告
	LightSidebar      = "#FCFCFC" // 设置侧边栏（与顶栏一致）
	LightChartSecondary = "#888888" // 流量图次要线
	LightSelection    = "#F2F2F2" // 选中行背景（浅灰，偏白）
)

// 深色模式 - 明暗反转，状态色不变；Primary 用中灰避免白底白字（代理/设置选中按钮）
const (
	DarkBackground   = "#121212"
	DarkHeader       = "#1E1E1E"
	DarkPrimary      = "#505050" // 选中按钮/主强调用中灰，保证浅色字可读
	DarkInputButton  = "#1E1E1E"
	DarkSeparator    = "#424242"
	DarkForeground   = "#E0E0E0"
	DarkPlaceholder  = "#9E9E9E"
	DarkSuccess      = "#4CAF50"
	DarkError        = "#F44336"
	DarkWarning        = "#FF9800"
	DarkChartSecondary = "#757575"
	DarkSelection      = "#2D2D2D" // 选中行背景（略亮于卡片）
)

// 延迟颜色：仅 <50ms 用绿，其余正文/占位
const (
	DelayFast  = "#4CAF50" // <50ms 成功绿
	DelayNone  = "#9E9E9E" // 未测速/超时 占位灰
)

// NewMonochromeTheme 创建主题实例。
func NewMonochromeTheme(variant fyne.ThemeVariant) fyne.Theme {
	return &MonochromeTheme{variant: variant}
}

// CurrentThemeColor 从当前应用主题取色。
func CurrentThemeColor(app fyne.App, name fyne.ThemeColorName) color.Color {
	if app == nil {
		return theme.DefaultTheme().Color(name, theme.VariantLight)
	}
	t := app.Settings().Theme()
	if t == nil {
		return theme.DefaultTheme().Color(name, theme.VariantLight)
	}
	variant := theme.VariantLight
	if mt, ok := t.(*MonochromeTheme); ok {
		variant = mt.variant
	}
	return t.Color(name, variant)
}

// IsDarkTheme 判断当前应用是否为深色主题。
func IsDarkTheme(app fyne.App) bool {
	if app == nil {
		return false
	}
	t := app.Settings().Theme()
	if mt, ok := t.(*MonochromeTheme); ok {
		return mt.variant == theme.VariantDark
	}
	return false
}

// DelayColor 根据延迟(ms)返回节点列表文字颜色。仅 <50ms 为绿，≤0 为占位灰，其余正文色。
func DelayColor(app fyne.App, delayMs int) color.Color {
	if delayMs <= 0 {
		return hexToRGBA(DelayNone)
	}
	if delayMs < 50 {
		return hexToRGBA(DelayFast)
	}
	return CurrentThemeColor(app, theme.ColorNameForeground)
}

// SidebarBackgroundColor 设置页左侧菜单背景（与顶栏一致）。
func SidebarBackgroundColor(app fyne.App) color.Color {
	if app == nil {
		return hexToRGBA(LightHeader)
	}
	return CurrentThemeColor(app, theme.ColorNameHeaderBackground)
}

// ChartUploadColor 流量图上传/入站（主色）。
func ChartUploadColor(app fyne.App) color.Color {
	return CurrentThemeColor(app, theme.ColorNamePrimary)
}

// ChartDownloadColor 流量图下载/出站（灰色，极简无彩色）。
func ChartDownloadColor(app fyne.App) color.Color {
	if IsDarkTheme(app) {
		return hexToRGBA(DarkChartSecondary)
	}
	return hexToRGBA(LightChartSecondary)
}

// MainButtonActiveFill 主开关「开启」时的填充色。浅色下用深灰避免纯黑，深色下用 Primary。
func MainButtonActiveFill(app fyne.App) color.Color {
	if IsDarkTheme(app) {
		return CurrentThemeColor(app, theme.ColorNamePrimary)
	}
	return hexToRGBA("#424242") // 浅色下开启时用深灰，避免纯黑背景
}

// hexToRGBA 将十六进制颜色转换为 NRGBA。
func hexToRGBA(hex string) color.NRGBA {
	var r, g, b uint8
	var a uint8 = 255
	if len(hex) == 7 {
		fmt.Sscanf(hex[1:], "%02x%02x%02x", &r, &g, &b)
	} else if len(hex) == 9 {
		fmt.Sscanf(hex[1:], "%02x%02x%02x%02x", &r, &g, &b, &a)
	}
	return color.NRGBA{R: r, G: g, B: b, A: a}
}

// Color 返回主题颜色。始终使用主题自身的 variant，确保深色模式下全局使用深色配色（不随 Fyne 传入的 variant 漂移）。
func (t *MonochromeTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	variant = t.variant
	switch variant {
	case theme.VariantDark:
		switch name {
		case theme.ColorNameBackground:
			return hexToRGBA(DarkBackground)
		case theme.ColorNameHeaderBackground:
			return hexToRGBA(DarkHeader)
		case theme.ColorNameInputBackground, theme.ColorNameButton:
			return hexToRGBA(DarkInputButton)
		case theme.ColorNameSeparator:
			return hexToRGBA(DarkSeparator)
		case theme.ColorNameDisabled, theme.ColorNamePlaceHolder:
			return hexToRGBA(DarkPlaceholder)
		case theme.ColorNameForeground:
			return hexToRGBA(DarkForeground)
		case theme.ColorNameHyperlink:
			return hexToRGBA(DarkPrimary)
		case theme.ColorNamePrimary:
			return hexToRGBA(DarkPrimary)
		case theme.ColorNameFocus:
			return hexToRGBA(DarkPrimary + "80")
		case theme.ColorNameHover:
			return hexToRGBA(DarkPrimary + "50")
		case theme.ColorNameSelection:
			return hexToRGBA(DarkSelection)
		case theme.ColorNameSuccess:
			return hexToRGBA(DarkSuccess)
		case theme.ColorNameError:
			return hexToRGBA(DarkError)
		case theme.ColorNameWarning:
			return hexToRGBA(DarkWarning)
		}
	case theme.VariantLight:
		switch name {
		case theme.ColorNameBackground:
			return hexToRGBA(LightBackground)
		case theme.ColorNameHeaderBackground:
			return hexToRGBA(LightHeader)
		case theme.ColorNameInputBackground, theme.ColorNameButton:
			return hexToRGBA(LightInputButton)
		case theme.ColorNameSeparator:
			return hexToRGBA(LightSeparator)
		case theme.ColorNameDisabled, theme.ColorNamePlaceHolder:
			return hexToRGBA(LightPlaceholder)
		case theme.ColorNameForeground:
			return hexToRGBA(LightForeground)
		case theme.ColorNameHyperlink:
			return hexToRGBA(LightPrimary)
		case theme.ColorNamePrimary:
			return hexToRGBA(LightPrimary)
		case theme.ColorNameFocus:
			return hexToRGBA(LightPrimary + "80")
		case theme.ColorNameHover:
			return hexToRGBA(LightPrimary + "50")
		case theme.ColorNameSelection:
			return hexToRGBA(LightSelection)
		case theme.ColorNameSuccess:
			return hexToRGBA(LightSuccess)
		case theme.ColorNameError:
			return hexToRGBA(LightError)
		case theme.ColorNameWarning:
			return hexToRGBA(LightWarning)
		}
	}
	return theme.DefaultTheme().Color(name, variant)
}

// Icon 使用默认主题图标
func (t *MonochromeTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// Font 使用默认字体
func (t *MonochromeTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

// Size 使用默认尺寸
func (t *MonochromeTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
