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
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var (
	trayIconCache  fyne.Resource
	appIconCache   fyne.Resource
	iconCacheMutex sync.Mutex
)

// getIconDir returns the asset directory, compatible with both `go run` and `go build`.
func getIconDir() string {
	exec, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return filepath.Join(wd, "assets")
	}
	dir := filepath.Dir(exec)
	if strings.Contains(strings.ToLower(dir), "go-build") {
		if wd, err := os.Getwd(); err == nil {
			return filepath.Join(wd, "assets")
		}
	}
	return filepath.Join(dir, "assets")
}

// ClearIconCaches clears cached icons; call this after a theme change.
func ClearIconCaches() {
	iconCacheMutex.Lock()
	defer iconCacheMutex.Unlock()
	trayIconCache = nil
	appIconCache = nil
}

// resolveVariant converts the AppState theme setting to a fyne.ThemeVariant.
func resolveVariant(appState *AppState) fyne.ThemeVariant {
	if appState == nil {
		return theme.VariantDark
	}
	switch appState.GetTheme() {
	case ThemeLight:
		return theme.VariantLight
	case ThemeSystem:
		if appState.App != nil {
			return appState.App.Settings().ThemeVariant()
		}
	}
	return theme.VariantDark
}

// createAppIcon returns the 228×228 window icon (cached after first call).
func createAppIcon(appState *AppState) fyne.Resource {
	iconCacheMutex.Lock()
	defer iconCacheMutex.Unlock()
	if appIconCache == nil {
		appIconCache = buildIcon(228, "app-icon-bw.png", resolveVariant(appState))
	}
	return appIconCache
}

// createTrayIconResource returns the 32×32 system tray icon (cached after first call).
func createTrayIconResource(appState *AppState) fyne.Resource {
	iconCacheMutex.Lock()
	defer iconCacheMutex.Unlock()
	if trayIconCache == nil {
		trayIconCache = buildIcon(32, "tray-icon-bw.png", resolveVariant(appState))
	}
	return trayIconCache
}

// createHomeLogo returns the 32×32 home-page logo for the current theme.
func createHomeLogo(appState *AppState) fyne.Resource {
	variant := resolveVariant(appState)
	varStr := "dark"
	if variant == theme.VariantLight {
		varStr = "light"
	}
	themeStr := "dark"
	if appState != nil {
		themeStr = string(appState.GetTheme())
	}
	name := fmt.Sprintf("home-logo-%s-draw-%s.png", themeStr, varStr)
	return buildIcon(32, name, variant)
}

// buildIcon loads an icon from the disk cache, or renders and saves it.
func buildIcon(size int, name string, variant fyne.ThemeVariant) fyne.Resource {
	path := filepath.Join(getIconDir(), name)
	if data, err := os.ReadFile(path); err == nil {
		return fyne.NewStaticResource(name, data)
	}
	img := renderIcon(size, variant)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		fmt.Printf("png encode failed (%s): %v\n", name, err)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err == nil {
		_ = os.WriteFile(path, buf.Bytes(), 0644)
	}
	return fyne.NewStaticResource(name, buf.Bytes())
}

// renderIcon rasterises the VPN "L-in-circle" icon at any square size.
//
// ── Design spec (base 32×32 grid, scaled uniformly to `size`) ──────────────
//
//	Canvas:  size×size px, 1 px padding → circle touches all four inset edges
//	Circle:  center (16,16), radius 15, diameter 30
//
//	L shape (hollowed negative space), "Regular" weight, corner radius 1.5 u:
//	  Vertical bar:    x=[10,14]  y=[7,25]   — full height of the L
//	  Horizontal bar:  x=[14,23]  y=[21,25]  — foot, shares right edge of vert bar
//
//	Gap-split effect:
//	  The two bars overlap in the region x=[10,14] y=[21,25].
//	  Instead of merging seamlessly, two hairline seams are cut at the junction:
//	    Vertical seam:   x ∈ [vx1−gV, vx1+gV],  y ∈ [hy0, vy1]
//	                     (right edge of vertical bar / left edge of horizontal bar)
//	    Horizontal seam: y ∈ [hy0−gH, hy0+gH],  x ∈ [vx0, vx1]
//	                     (top edge of horizontal bar, within vertical bar width)
//	  Pixels inside a seam are transparent — everything else inside the L union
//	  retains its fill colour.  The overlap region itself is therefore coloured
//	  (not hollow), except for the two thin seam lines.
//
//	Anti-aliasing: 4×4 supersampling; circle edge smooth, L edges crisp.
//	Outside circle: always alpha = 0 (transparent) for both variants.
//	Dark:  opaque black disk, transparent L cutout and seams.
//	Light: opaque white disk, transparent L cutout and seams.
func renderIcon(size int, variant fyne.ThemeVariant) *image.RGBA {
	const base = 32.0
	const ss = 4

	sc := float64(size) / base
	light := variant == theme.VariantLight

	// Circle (pixel space).
	cx, cy, cr := 16*sc, 16*sc, 15*sc
	aa := 0.7 * sc

	// L bars (base-32 units).
	const (
		rr  = 1.5 // corner radius
		vx0 = 10.0; vx1 = 14.0; vy0 = 7.0; vy1 = 25.0
		hx0 = 14.0; hx1 = 23.0; hy0 = 21.0; hy1 = 25.0
	)

	// Seam half-widths (base-32 units). ~0.5 u ≈ 1 px at 32 px output.
	const gV = 0.5 // vertical seam half-width   (along x-axis)
	const gH = 0.5 // horizontal seam half-width  (along y-axis)

	// Seam regions (base-32 units).
	// Vertical seam: right edge of vert bar meets left edge of horiz bar.
	const svX0, svX1 = vx1 - gV, vx1 + gV
	const svY0, svY1 = hy0, vy1

	// Horizontal seam: top edge of horiz bar, only within vert bar x-range.
	const shX0, shX1 = vx0, vx1
	const shY0, shY1 = hy0 - gH, hy0 + gH

	inSeam := func(bx, by float64) bool {
		sv := bx >= svX0 && bx <= svX1 && by >= svY0 && by <= svY1
		sh := bx >= shX0 && bx <= shX1 && by >= shY0 && by <= shY1
		return sv || sh
	}

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	step := 1.0 / float64(ss)

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var sDisk, sL float64

			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					px := float64(x) + (float64(sx)+0.5)*step
					py := float64(y) + (float64(sy)+0.5)*step
					bx, by := px/sc, py/sc

					// Circle coverage (smooth).
					d := math.Hypot(px-cx, py-cy)
					var disk float64
					switch {
					case d <= cr-aa:
						disk = 1
					case d < cr+aa:
						t := (d - (cr - aa)) / (2 * aa)
						disk = 1 - t*t*(3-2*t)
					}

					// L membership: union of both bars, minus seam lines.
					inShape := inRR(bx, by, vx0, vy0, vx1, vy1, rr) ||
						inRR(bx, by, hx0, hy0, hx1, hy1, rr)
					inLetter := inShape && !inSeam(bx, by)

					sDisk += disk
					if inLetter {
						sL++
					}
				}
			}

			n := float64(ss * ss)
			disk := sDisk / n
			lett := sL / n

			// solid = inside circle AND inside the L shape (coloured region).
			// L is negative space: circle minus L = filled; L itself = transparent.
			solid := disk * (1 - lett)

			var c color.RGBA
			if light {
				c = color.RGBA{R: 255, G: 255, B: 255, A: u8(255 * solid)}
			} else {
				c = color.RGBA{A: u8(255 * solid)}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// inRR reports whether (px, py) lies inside an axis-aligned rounded rectangle.
func inRR(px, py, x0, y0, x1, y1, r float64) bool {
	if px < x0 || px > x1 || py < y0 || py > y1 {
		return false
	}
	cr := math.Min(r, 0.5*math.Min(x1-x0, y1-y0))
	if px < x0+cr && py < y0+cr {
		return math.Hypot(px-(x0+cr), py-(y0+cr)) <= cr
	}
	if px > x1-cr && py < y0+cr {
		return math.Hypot(px-(x1-cr), py-(y0+cr)) <= cr
	}
	if px < x0+cr && py > y1-cr {
		return math.Hypot(px-(x0+cr), py-(y1-cr)) <= cr
	}
	if px > x1-cr && py > y1-cr {
		return math.Hypot(px-(x1-cr), py-(y1-cr)) <= cr
	}
	return true
}

// u8 rounds and clamps a float64 to [0, 255].
func u8(v float64) uint8 {
	return uint8(math.Max(0, math.Min(255, math.Round(v))))
}
