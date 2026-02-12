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

// 品牌色定义 - 符合苹果设计理念的配色
const (
	// BrandPrimary 主品牌色 - 符合苹果设计的蓝色
	BrandPrimary = "#007AFF"
	// BrandSecondary 次要品牌色 - 柔和的灰色
	BrandSecondary = "#8E8E93"
	// BrandAccent 强调色 - 苹果系统绿色
	BrandAccent = "#34C759"
	// BrandError 错误色 - 苹果系统红色
	BrandError = "#FF3B30"
	// BrandWarning 警告色 - 苹果系统橙色
	BrandWarning = "#FF9500"
	// BrandInfo 信息色 - 柔和的蓝色
	BrandInfo = "#5AC8FA"
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
			return color.NRGBA{R: 28, G: 28, B: 30, A: 255} // 苹果深色模式背景
		case theme.ColorNameInputBackground:
			return color.NRGBA{R: 44, G: 44, B: 46, A: 255} // 苹果深色模式输入框背景
		case theme.ColorNameForeground:
			return color.NRGBA{R: 255, G: 255, B: 255, A: 255} // 苹果深色模式文字
		case theme.ColorNameButton:
			return color.NRGBA{R: 44, G: 44, B: 46, A: 255} // 苹果深色模式按钮背景
		case theme.ColorNamePrimary:
			return hexToRGBA(BrandPrimary) // 使用品牌色作为主要元素颜色
		case theme.ColorNameFocus:
			return hexToRGBA(BrandPrimary + "80") // 品牌色半透明作为焦点高亮
		case theme.ColorNameHover:
			return hexToRGBA(BrandPrimary + "50") // 品牌色更透明作为悬停效果
		case theme.ColorNameDisabled:
			return color.NRGBA{R: 174, G: 174, B: 178, A: 255} // 苹果深色模式禁用状态
		case theme.ColorNamePlaceHolder:
			return color.NRGBA{R: 174, G: 174, B: 178, A: 255} // 苹果深色模式占位符文字
		case theme.ColorNameSelection:
			return hexToRGBA(BrandPrimary + "64") // 品牌色半透明作为选中状态
		case theme.ColorNameSeparator:
			return color.NRGBA{R: 56, G: 56, B: 58, A: 255} // 苹果深色模式分隔线
		case theme.ColorNameSuccess:
			return hexToRGBA(BrandAccent) // 成功色
		case theme.ColorNameWarning:
			return hexToRGBA(BrandWarning) // 警告色
		case theme.ColorNameError:
			return hexToRGBA(BrandError) // 错误色
		case theme.ColorNameHeaderBackground:
			return color.NRGBA{R: 32, G: 32, B: 34, A: 255} // 苹果深色模式标题背景
		case theme.ColorNameHyperlink:
			return hexToRGBA(BrandPrimary) // 超链接
		}
	case theme.VariantLight:
		switch name {
		case theme.ColorNameBackground:
			return color.NRGBA{R: 255, G: 255, B: 255, A: 255} // 苹果浅色模式背景
		case theme.ColorNameInputBackground:
			return color.NRGBA{R: 242, G: 242, B: 247, A: 255} // 苹果浅色模式输入框背景
		case theme.ColorNameForeground:
			return color.NRGBA{R: 0, G: 0, B: 0, A: 255} // 苹果浅色模式文字
		case theme.ColorNameButton:
			return color.NRGBA{R: 242, G: 242, B: 247, A: 255} // 苹果浅色模式按钮背景
		case theme.ColorNamePrimary:
			return hexToRGBA(BrandPrimary) // 使用品牌色作为主要元素颜色
		case theme.ColorNameFocus:
			return hexToRGBA(BrandPrimary + "78") // 品牌色半透明作为焦点高亮
		case theme.ColorNameHover:
			return hexToRGBA(BrandPrimary + "50") // 品牌色更透明作为悬停效果
		case theme.ColorNameDisabled:
			return color.NRGBA{R: 199, G: 199, B: 204, A: 255} // 苹果浅色模式禁用状态
		case theme.ColorNamePlaceHolder:
			return color.NRGBA{R: 199, G: 199, B: 204, A: 255} // 苹果浅色模式占位符文字
		case theme.ColorNameSelection:
			return hexToRGBA(BrandPrimary + "64") // 品牌色半透明作为选中状态
		case theme.ColorNameSeparator:
			return color.NRGBA{R: 229, G: 229, B: 234, A: 255} // 苹果浅色模式分隔线
		case theme.ColorNameSuccess:
			return hexToRGBA(BrandAccent) // 成功色
		case theme.ColorNameWarning:
			return hexToRGBA(BrandWarning) // 警告色
		case theme.ColorNameError:
			return hexToRGBA(BrandError) // 错误色
		case theme.ColorNameHeaderBackground:
			return color.NRGBA{R: 242, G: 242, B: 247, A: 255} // 苹果浅色模式标题背景
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

// Size 返回尺寸配置。
// 这里直接使用 Fyne 默认尺寸，避免自定义尺寸导致图标或间距异常放大。
func (t *MonochromeTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
