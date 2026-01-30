package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// MonochromeTheme 实现 Fyne 主题接口，提供黑白两套主题（Dark/Light）。
// 该主题使用优化的配色方案，增强对比度和层次感，提供更好的视觉体验。
type MonochromeTheme struct {
	variant fyne.ThemeVariant
}

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
			// MediumImportance 按钮使用此颜色，加深20%使其更突出
			// 原色：R:255, G:255, B:255 (白色)
			// 加深20%：255 * 0.8 = 204，但由于是白色，加深意味着降低亮度
			// 实际上，对于白色来说，加深20%意味着使用更深的灰色
			// 为了保持对比度，我们使用一个更深的灰色：255 * 0.7 = 178.5 ≈ 179
			return color.NRGBA{R: 204, G: 204, B: 204, A: 255} // 主要元素（MediumImportance按钮）使用加深20%的颜色
		case theme.ColorNameFocus:
			return color.NRGBA{R: 255, G: 255, B: 255, A: 128} // 焦点高亮，更明显
		case theme.ColorNameHover:
			return color.NRGBA{R: 255, G: 255, B: 255, A: 80} // 悬停效果，适中
		case theme.ColorNameDisabled:
			return color.NRGBA{R: 100, G: 100, B: 100, A: 255} // 禁用状态，降低对比
		case theme.ColorNamePlaceHolder:
			return color.NRGBA{R: 140, G: 140, B: 140, A: 255} // 占位符文字
		case theme.ColorNameSelection:
			return color.NRGBA{R: 255, G: 255, B: 255, A: 100} // 选中状态
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
			// MediumImportance 按钮使用此颜色，加深20%使其更突出
			// 原色：R:30, G:30, B:30 (深灰色)
			// 加深20%：30 * 0.8 = 24，但由于是深色，加深意味着更深的黑色
			// 实际上，对于深色来说，加深20%意味着更接近黑色
			// 计算：30 * 0.8 = 24，但我们想要更明显的加深效果
			// 使用更深的颜色：R:24, G:24, B:24
			return color.NRGBA{R: 24, G: 24, B: 24, A: 255} // 主要元素（MediumImportance按钮）使用加深20%的颜色
		case theme.ColorNameFocus:
			return color.NRGBA{R: 0, G: 0, B: 0, A: 120} // 焦点高亮，更明显
		case theme.ColorNameHover:
			return color.NRGBA{R: 0, G: 0, B: 0, A: 80} // 悬停效果，更明显
		case theme.ColorNameDisabled:
			return color.NRGBA{R: 180, G: 180, B: 180, A: 255} // 禁用状态
		case theme.ColorNamePlaceHolder:
			return color.NRGBA{R: 150, G: 150, B: 150, A: 255} // 占位符文字
		case theme.ColorNameSelection:
			return color.NRGBA{R: 0, G: 0, B: 0, A: 100} // 选中状态，更明显
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
		return 3  // 小滚动条
	case theme.SizeNameSeparatorThickness:
		return 1  // 分隔线更细
	case theme.SizeNameInputBorder:
		return 1  // 输入框边框
	case theme.SizeNameInputRadius:
		return 6  // 输入框圆角，更圆润
	case theme.SizeNameSelectionRadius:
		return 6  // 选中圆角，更圆润
	case theme.SizeNameInlineIcon:
		return 20 // 内联图标
	}
	// 其他尺寸使用默认值
	return theme.DefaultTheme().Size(name)
}
