package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// MonochromeTheme 实现 Fyne 主题接口，提供黑白两套主题（Dark/Light）。
// 该主题使用优化的配色方案，增强对比度和层次感，提供更好的视觉体验。
type MonochromeTheme struct {
	variant fyne.ThemeVariant
}

// 品牌色定义
const (
	// BrandPrimary 主品牌色
	BrandPrimary = "#3B82F6"
	// BrandSecondary 次要品牌色
	BrandSecondary = "#10B981"
	// BrandAccent 强调色
	BrandAccent = "#8B5CF6"
	// BrandError 错误色
	BrandError = "#EF4444"
	// BrandWarning 警告色
	BrandWarning = "#F59E0B"
	// BrandInfo 信息色
	BrandInfo = "#3B82F6"
)

// NewMonochromeTheme 创建黑白主题实例。
// 参数：
//   - variant: 主题变体，支持 fyne.ThemeVariantDark（黑色）或 fyne.ThemeVariantLight（白色）
//
// 返回：主题实例
func NewMonochromeTheme(variant fyne.ThemeVariant) fyne.Theme {
	return &MonochromeTheme{variant: variant}
}

// CurrentThemeColor 从当前应用主题取色，供自定义组件在绘制/刷新时使用，切换主题后可立即生效。
// 若 app 为 nil 或未设置主题，则回退到默认主题的深色变体。
func CurrentThemeColor(app fyne.App, name fyne.ThemeColorName) color.Color {
	if app == nil {
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
	t := app.Settings().Theme()
	if t == nil {
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
	variant := theme.VariantDark
	if mt, ok := t.(*MonochromeTheme); ok {
		variant = mt.variant
	}
	return t.Color(name, variant)
}

// hexToRGBA 将十六进制颜色转换为 RGBA
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

// Color 返回自定义颜色，未覆盖的颜色使用默认主题
func (t *MonochromeTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// 以传入 variant 优先，其次使用主题自身 variant
	if variant == fyne.ThemeVariant(0) {
		variant = t.variant
	}

	switch variant {
	case theme.VariantDark:
		switch name {
		case theme.ColorNameBackground:
			return color.NRGBA{R: 23, G: 23, B: 23, A: 255} // 深灰背景，增强层次
		case theme.ColorNameInputBackground:
			return color.NRGBA{R: 30, G: 30, B: 30, A: 255} // 稍亮的输入背景，形成层次
		case theme.ColorNameForeground:
			return color.NRGBA{R: 240, G: 240, B: 240, A: 255} // 更亮的文字，增强对比度
		case theme.ColorNameButton:
			return color.NRGBA{R: 45, G: 45, B: 45, A: 255} // 按钮背景，与输入框区分
		case theme.ColorNamePrimary:
			return hexToRGBA(BrandPrimary) // 使用品牌色作为主要元素颜色
		case theme.ColorNameFocus:
			return hexToRGBA(BrandPrimary + "80") // 品牌色半透明作为焦点高亮
		case theme.ColorNameHover:
			return hexToRGBA(BrandPrimary + "50") // 品牌色更透明作为悬停效果
		case theme.ColorNameDisabled:
			return color.NRGBA{R: 100, G: 100, B: 100, A: 255} // 禁用状态，降低对比
		case theme.ColorNamePlaceHolder:
			return color.NRGBA{R: 140, G: 140, B: 140, A: 255} // 占位符文字
		case theme.ColorNameSelection:
			return hexToRGBA(BrandPrimary + "64") // 品牌色半透明作为选中状态
		case theme.ColorNameSeparator:
			return color.NRGBA{R: 60, G: 60, B: 60, A: 255} // 分隔线
		case theme.ColorNameSuccess:
			return hexToRGBA(BrandSecondary) // 成功色
		case theme.ColorNameWarning:
			return hexToRGBA(BrandWarning) // 警告色
		case theme.ColorNameError:
			return hexToRGBA(BrandError) // 错误色
		case theme.ColorNameHeaderBackground:
			return color.NRGBA{R: 30, G: 30, B: 30, A: 255} // 标题背景
		case theme.ColorNameHyperlink:
			return hexToRGBA(BrandPrimary) // 超链接
		}
	case theme.VariantLight:
		switch name {
		case theme.ColorNameBackground:
			return color.NRGBA{R: 255, G: 255, B: 255, A: 255} // 白色背景
		case theme.ColorNameInputBackground:
			return color.NRGBA{R: 252, G: 252, B: 252, A: 255} // 极浅灰输入背景，形成层次
		case theme.ColorNameForeground:
			return color.NRGBA{R: 20, G: 20, B: 20, A: 255} // 深色文字，增强对比
		case theme.ColorNameButton:
			return color.NRGBA{R: 245, G: 245, B: 245, A: 255} // 浅灰按钮背景，更柔和
		case theme.ColorNamePrimary:
			return hexToRGBA(BrandPrimary) // 使用品牌色作为主要元素颜色
		case theme.ColorNameFocus:
			return hexToRGBA(BrandPrimary + "78") // 品牌色半透明作为焦点高亮
		case theme.ColorNameHover:
			return hexToRGBA(BrandPrimary + "50") // 品牌色更透明作为悬停效果
		case theme.ColorNameDisabled:
			return color.NRGBA{R: 180, G: 180, B: 180, A: 255} // 禁用状态
		case theme.ColorNamePlaceHolder:
			return color.NRGBA{R: 150, G: 150, B: 150, A: 255} // 占位符文字
		case theme.ColorNameSelection:
			return hexToRGBA(BrandPrimary + "64") // 品牌色半透明作为选中状态
		case theme.ColorNameSeparator:
			return color.NRGBA{R: 220, G: 220, B: 220, A: 255} // 分隔线
		case theme.ColorNameSuccess:
			return hexToRGBA(BrandSecondary) // 成功色
		case theme.ColorNameWarning:
			return hexToRGBA(BrandWarning) // 警告色
		case theme.ColorNameError:
			return hexToRGBA(BrandError) // 错误色
		case theme.ColorNameHeaderBackground:
			return color.NRGBA{R: 248, G: 248, B: 248, A: 255} // 标题背景
		case theme.ColorNameHyperlink:
			return hexToRGBA(BrandPrimary) // 超链接
		}
	}

	// 其他颜色使用默认主题
	return theme.DefaultTheme().Color(name, variant)
}

// Icon 使用默认主题图标
func (t *MonochromeTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// Font 使用默认字体，保持兼容
func (t *MonochromeTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

// Size 返回自定义尺寸，增加内边距和间距以提升视觉体验
func (t *MonochromeTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 12 // 增加内边距（默认8）
	case theme.SizeNameScrollBar:
		return 16 // 滚动条宽度
	case theme.SizeNameScrollBarSmall:
		return 3 // 小滚动条
	case theme.SizeNameSeparatorThickness:
		return 1 // 分隔线更细
	case theme.SizeNameInputBorder:
		return 1 // 输入框边框
	case theme.SizeNameInputRadius:
		return 6 // 输入框圆角，更圆润
	case theme.SizeNameSelectionRadius:
		return 6 // 选中圆角，更圆润
	case theme.SizeNameInlineIcon:
		return 20 // 内联图标
	}
	// 其他尺寸使用默认值
	return theme.DefaultTheme().Size(name)
}
