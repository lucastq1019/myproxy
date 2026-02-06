package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var (
	// 图标缓存
	trayIconCache     fyne.Resource
	appIconCache      fyne.Resource
	settingsLogoCache fyne.Resource
	iconCacheMutex    sync.Mutex
)

// getIconDir 获取图标存储目录
func getIconDir() string {
	// 获取可执行文件所在目录
	execPath, err := os.Executable()
	if err != nil {
		// 如果获取失败，使用当前工作目录
		wd, _ := os.Getwd()
		return filepath.Join(wd, "assets")
	}
	execDir := filepath.Dir(execPath)
	return filepath.Join(execDir, "assets")
}

// createAppIcon 创建应用图标资源（用于窗口图标，228x228）
// 参数：
//   - appState: 应用状态（用于获取主题配置）
func createAppIcon(appState *AppState) fyne.Resource {
	iconCacheMutex.Lock()
	defer iconCacheMutex.Unlock()

	if appIconCache != nil {
		return appIconCache
	}

	appIconCache = createLShapeIcon(228, "app-icon.png", appState)
	return appIconCache
}

// createTrayIconResource 创建系统托盘图标资源（32x32，L形布局）
// 参数：
//   - appState: 应用状态（用于获取主题配置）
func createTrayIconResource(appState *AppState) fyne.Resource {
	iconCacheMutex.Lock()
	defer iconCacheMutex.Unlock()

	if trayIconCache != nil {
		return trayIconCache
	}

	trayIconCache = createLShapeIcon(32, "tray-icon.png", appState)
	return trayIconCache
}

// createSettingsLogo 创建设置页面logo资源（64x64，根据主题变化）
// 参数：
//   - appState: 应用状态（用于获取主题配置）
// 注意：logo颜色与主题色相同，背景色根据主题变化
func createSettingsLogo(appState *AppState) fyne.Resource {
	// 获取当前主题variant，确保文件名包含完整的主题信息
	themeVariant := theme.VariantDark
	themeStr := ThemeDark
	if appState != nil {
		themeStr = appState.GetTheme()
		switch themeStr {
		case ThemeLight:
			themeVariant = theme.VariantLight
		case ThemeSystem:
			if appState.App != nil {
				themeVariant = appState.App.Settings().ThemeVariant()
			}
		default:
			themeVariant = theme.VariantDark
		}
	}
	// 文件名包含主题字符串和variant信息，确保不同主题使用不同文件
	variantStr := "dark"
	if themeVariant == theme.VariantLight {
		variantStr = "light"
	}
	fileName := fmt.Sprintf("settings-logo-%s-%s.png", themeStr, variantStr)
	return createLShapeIconWithVariant(64, fileName, themeVariant)
}

// createHomeLogo 创建主页logo资源（32x32，根据主题变化）
// 参数：
//   - appState: 应用状态（用于获取主题配置）
// 注意：logo颜色与主题色相反（light主题用dark色，dark主题用light色）
func createHomeLogo(appState *AppState) fyne.Resource {
	// 获取当前主题
	currentTheme := ThemeDark
	if appState != nil {
		currentTheme = appState.GetTheme()
	}

	// 确定相反的主题variant
	var oppositeVariant fyne.ThemeVariant
	if currentTheme == ThemeLight {
		oppositeVariant = theme.VariantDark
	} else if currentTheme == ThemeSystem {
		// 如果是系统主题，需要判断当前系统主题
		if appState != nil && appState.App != nil {
			systemVariant := appState.App.Settings().ThemeVariant()
			if systemVariant == theme.VariantLight {
				oppositeVariant = theme.VariantDark
			} else {
				oppositeVariant = theme.VariantLight
			}
		} else {
			oppositeVariant = theme.VariantDark
		}
	} else {
		// ThemeDark
		oppositeVariant = theme.VariantLight
	}

	// 生成文件名，使用相反主题，包含variant信息确保不同主题使用不同文件
	variantStr := "dark"
	if oppositeVariant == theme.VariantLight {
		variantStr = "light"
	}
	fileName := fmt.Sprintf("home-logo-%s-opposite-%s.png", currentTheme, variantStr)
	return createLShapeIconWithVariant(32, fileName, oppositeVariant)
}

// createLShapeIconWithVariant 创建下L形布局的图标（使用指定的主题variant）
// 参数：
//   - size: 图标尺寸（正方形）
//   - name: 资源名称
//   - variant: 主题变体（用于确定logo颜色）
func createLShapeIconWithVariant(size int, name string, variant fyne.ThemeVariant) fyne.Resource {
	// 检查文件是否已存在
	iconDir := getIconDir()
	iconPath := filepath.Join(iconDir, name)

	// 如果文件存在，直接加载（文件名已包含主题信息，确保不同主题使用不同文件）
	if _, err := os.Stat(iconPath); err == nil {
		fmt.Printf("图标文件已存在，加载: %s\n", iconPath)
		if data, err := os.ReadFile(iconPath); err == nil {
			return fyne.NewStaticResource(name, data)
		}
		fmt.Printf("读取图标文件失败，重新生成: %v\n", err)
	}

	// 创建主题实例并使用 Color 方法获取背景色
	monochromeTheme := NewMonochromeTheme(variant)
	bgColorValue := monochromeTheme.Color(theme.ColorNameBackground, variant)

	// 转换为 RGBA，失败时使用默认主题背景色
	var bgColor color.RGBA
	if nrgba, ok := bgColorValue.(color.NRGBA); ok {
		bgColor = color.RGBA{R: nrgba.R, G: nrgba.G, B: nrgba.B, A: nrgba.A}
	} else if rgba, ok := bgColorValue.(color.RGBA); ok {
		bgColor = rgba
	} else {
		fallback := theme.DefaultTheme().Color(theme.ColorNameBackground, variant)
		r, g, b, a := fallback.RGBA()
		bgColor = color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	}

	// 使用新的绘制方式创建图标
	img := createIconImage(size, bgColor)

	// 将图片编码为 PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		fmt.Printf("编码 PNG 失败 (%s): %v\n", name, err)
		return nil
	}

	// 保存到文件系统
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		fmt.Printf("创建图标目录失败: %v\n", err)
	} else {
		if err := os.WriteFile(iconPath, buf.Bytes(), 0644); err != nil {
			fmt.Printf("保存图标文件失败 (%s): %v\n", iconPath, err)
		} else {
			fmt.Printf("图标已保存到文件: %s\n", iconPath)
		}
	}

	fmt.Printf("图标创建成功 (%s, %dx%d)，大小: %d 字节\n", name, size, size, buf.Len())
	return fyne.NewStaticResource(name, buf.Bytes())
}

// createLShapeIcon 创建下L形布局的图标（使用水滴形状）
// 参数：
//   - size: 图标尺寸（正方形）
//   - name: 资源名称
//   - appState: 应用状态（用于获取主题配置）
func createLShapeIcon(size int, name string, appState *AppState) fyne.Resource {
	// 从主题获取背景色
	// 从 ConfigService 读取主题配置
	themeVariant := theme.VariantDark
	if appState != nil {
		themeStr := appState.GetTheme()
		switch themeStr {
		case ThemeLight:
			themeVariant = theme.VariantLight
		case ThemeSystem:
			// 如果是系统主题，需要判断当前系统主题
			if appState.App != nil {
				themeVariant = appState.App.Settings().ThemeVariant()
			}
		default:
			themeVariant = theme.VariantDark
		}
	}
	return createLShapeIconWithVariant(size, name, themeVariant)
}

// createIconImage 使用透明区域方式生成图标
// size: 图标尺寸（正方形）
// bgColor: 背景颜色（从主题获取）
func createIconImage(size int, bgColor color.RGBA) *image.RGBA {
	transparent := color.RGBA{0, 0, 0, 0} // 透明色

	// 1. 创建画布
	rect := image.Rect(0, 0, size, size)
	canvas := image.NewRGBA(rect)

	// 2. 计算 L 形尺寸和位置
	center := float64(size) / 2.0
	radius := center
	R := float64(size) / 6.0
	H := float64(size) / 2.0
	G := float64(size) / 50.0     // 间隙缩小一半（原来是 size/10）
	L_line := float64(size) / 3.0 // 横线长度缩短1/2（原来是 size/2）
	W_line := R * 2.0 / 4.0       // 线宽

	// 计算L形的整体尺寸，用于居中定位
	// L形总宽度 = 左侧宽度(R*2) + 间隙(G) + 横线长度(L_line)
	lTotalWidth := R*2.0 + G + L_line
	// L形总高度 = 三角形高度(H) + 圆的半径(R)
	lTotalHeight := H + R

	// L形整体居中定位
	lStartX := center - lTotalWidth/2.0
	lStartY := center - lTotalHeight/2.0

	// 左侧部分（圆和三角形）的位置
	// 圆心X坐标：L形起始X + 圆的半径
	Cx := lStartX + R
	// 圆心Y坐标：L形起始Y + L形总高度 - 圆的半径（底部对齐）
	Cy := lStartY + lTotalHeight - R
	// 三角形底部Y坐标（与圆心重叠）
	Ty := Cy

	// 矩形（横线）的位置
	lineStart := lStartX + R*2.0 + G
	lineEnd := lineStart + L_line
	lineTop := lStartY + lTotalHeight - W_line
	lineBottom := lStartY + lTotalHeight

	// 3. 绘制背景圆，在三角形、圆形、矩形区域内保持透明
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			xf := float64(x)
			yf := float64(y)

			// 判断是否在圆形背景内
			distToCenter := math.Sqrt(math.Pow(xf-center, 2) + math.Pow(yf-center, 2))
			if distToCenter > radius {
				// 不在圆形背景内，保持透明
				canvas.Set(x, y, transparent)
				continue
			}

			// 判断1: 是否在三角形内
			inTriangle := false
			if yf >= Ty-H && yf <= Ty {
				maxWidth := R * 2.0
				currentWidth := maxWidth * (yf - (Ty - H)) / H
				if xf >= Cx-currentWidth/2.0 && xf <= Cx+currentWidth/2.0 {
					inTriangle = true
				}
			}

			// 判断2: 是否在圆形内
			distToCircle := math.Sqrt(math.Pow(xf-Cx, 2) + math.Pow(yf-Cy, 2))
			inCircle := distToCircle <= R

			// 判断3: 是否在矩形内
			inRectangle := xf >= lineStart && xf <= lineEnd && yf >= lineTop && yf <= lineBottom

			// 如果在三角形、圆形或矩形内，保持透明；否则绘制背景色
			if inTriangle || inCircle || inRectangle {
				canvas.Set(x, y, transparent)
			} else {
				canvas.Set(x, y, bgColor)
			}
		}
	}

	return canvas
}
