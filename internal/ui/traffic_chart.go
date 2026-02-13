package ui

import (
	"fmt"
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// TrafficData 流量数据点
type TrafficData struct {
	Upload   int64 // 上传字节数
	Download int64 // 下载字节数
	Time     time.Time
}

// TrafficChart 实时流量图组件
type TrafficChart struct {
	widget.BaseWidget

	appState *AppState

	// 数据存储（最近的数据点）
	dataPoints []TrafficData
	maxPoints  int // 最大数据点数

	// 当前流量统计
	currentUpload   int64
	currentDownload int64

	// 上一次的流量统计（用于计算实时流量）
	lastUpload   int64
	lastDownload int64
	lastTime     time.Time

	// 锁保护
	mu sync.RWMutex

	// 更新定时器
	updateTicker *time.Ticker
	stopChan     chan struct{}
}

// NewTrafficChart 创建新的流量图组件
func NewTrafficChart(appState *AppState) *TrafficChart {
	tc := &TrafficChart{
		appState:   appState,
		dataPoints: make([]TrafficData, 0),
		maxPoints:  60, // 保留最近60个数据点（约1分钟，假设每秒更新）
		lastTime:   time.Now(),
		stopChan:   make(chan struct{}),
	}
	tc.ExtendBaseWidget(tc)

	// 启动更新定时器（每秒更新一次）
	tc.updateTicker = time.NewTicker(1 * time.Second)
	go tc.updateLoop()

	return tc
}

// updateLoop 更新循环
func (tc *TrafficChart) updateLoop() {
	for {
		select {
		case <-tc.updateTicker.C:
			tc.updateData()
			// 使用 fyne.Do 确保 UI 更新在主线程中执行
			fyne.Do(func() {
				tc.Refresh()
			})
		case <-tc.stopChan:
			return
		}
	}
}

// updateData 更新流量数据
func (tc *TrafficChart) updateData() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	var totalUpload, totalDownload int64
	if tc.appState != nil && tc.appState.XrayControlService != nil && tc.appState.XrayInstance != nil && tc.appState.XrayInstance.IsRunning() {
		totalUpload, totalDownload = tc.appState.XrayControlService.GetTrafficStats(tc.appState.XrayInstance)
	}

	// 计算实时流量（与上一次的差值）
	now := time.Now()
	timeDiff := now.Sub(tc.lastTime).Seconds()
	if timeDiff == 0 {
		timeDiff = 1 // 避免除以零
	}

	// 计算实时流量（字节/秒）
	upload := int64(float64(totalUpload-tc.lastUpload) / timeDiff)
	download := int64(float64(totalDownload-tc.lastDownload) / timeDiff)

	// 确保流量不为负数
	if upload < 0 {
		upload = 0
	}
	if download < 0 {
		download = 0
	}

	// 更新上一次的流量数据和时间
	tc.lastUpload = totalUpload
	tc.lastDownload = totalDownload
	tc.lastTime = now

	// 添加新数据点
	newPoint := TrafficData{
		Upload:   upload,
		Download: download,
		Time:     now,
	}

	tc.dataPoints = append(tc.dataPoints, newPoint)

	// 限制数据点数量
	if len(tc.dataPoints) > tc.maxPoints {
		tc.dataPoints = tc.dataPoints[len(tc.dataPoints)-tc.maxPoints:]
	}

	tc.currentUpload = upload
	tc.currentDownload = download
}

// Stop 停止更新
func (tc *TrafficChart) Stop() {
	if tc.updateTicker != nil {
		tc.updateTicker.Stop()
	}
	close(tc.stopChan)
}

// CreateRenderer 创建渲染器
func (tc *TrafficChart) CreateRenderer() fyne.WidgetRenderer {
	bgColor := CurrentThemeColor(tc.appState.App, theme.ColorNameBackground)
	return &trafficChartRenderer{
		trafficChart:  tc,
		uploadLines:   make([]*canvas.Line, 0),
		downloadLines: make([]*canvas.Line, 0),
		uploadLabel:   widget.NewLabel("上传: 0 KB/s"),
		downloadLabel: widget.NewLabel("下载: 0 KB/s"),
		bgRect:        canvas.NewRectangle(bgColor),
		objects:       make([]fyne.CanvasObject, 0),
	}
}

// trafficChartRenderer 流量图渲染器
type trafficChartRenderer struct {
	trafficChart *TrafficChart

	uploadLines   []*canvas.Line
	downloadLines []*canvas.Line
	uploadLabel   *widget.Label
	downloadLabel *widget.Label
	bgRect        *canvas.Rectangle

	objects []fyne.CanvasObject
}

// MinSize 返回最小尺寸
func (r *trafficChartRenderer) MinSize() fyne.Size {
	return fyne.NewSize(200, 80)
}

// Layout 布局
func (r *trafficChartRenderer) Layout(size fyne.Size) {
	// 背景矩形
	r.bgRect.Move(fyne.NewPos(0, 0))
	r.bgRect.Resize(size)

	// 图表区域（留出标签空间）
	chartHeight := size.Height - 40
	chartWidth := size.Width

	// 绘制折线图
	r.drawChart(chartWidth, chartHeight)

	// 标签位置
	labelY := size.Height - 35
	r.uploadLabel.Move(fyne.NewPos(10, labelY))
	r.uploadLabel.Resize(fyne.NewSize(size.Width/2-10, 20))

	r.downloadLabel.Move(fyne.NewPos(size.Width/2+10, labelY))
	r.downloadLabel.Resize(fyne.NewSize(size.Width/2-10, 20))
}

// drawChart 绘制图表
func (r *trafficChartRenderer) drawChart(width, height float32) {
	r.trafficChart.mu.RLock()
	dataPoints := make([]TrafficData, len(r.trafficChart.dataPoints))
	copy(dataPoints, r.trafficChart.dataPoints)
	r.trafficChart.mu.RUnlock()

	if len(dataPoints) < 2 {
		// 清理旧的线条
		r.uploadLines = r.uploadLines[:0]
		r.downloadLines = r.downloadLines[:0]
		return
	}

	// 找到最大值（用于缩放）
	maxValue := int64(1)
	for _, point := range dataPoints {
		if point.Upload > maxValue {
			maxValue = point.Upload
		}
		if point.Download > maxValue {
			maxValue = point.Download
		}
	}

	// 添加一些余量，确保图表不会贴边
	maxValue = maxValue * 110 / 100
	if maxValue == 0 {
		maxValue = 1
	}

	// 计算点之间的间距
	pointSpacing := width / float32(len(dataPoints)-1)

	// 清理旧的线条
	r.uploadLines = r.uploadLines[:0]
	r.downloadLines = r.downloadLines[:0]

	uploadColor := ChartUploadColor(r.trafficChart.appState.App)
	downloadColor := ChartDownloadColor(r.trafficChart.appState.App)

	// 绘制上传线（连接所有点）
	for i := 0; i < len(dataPoints)-1; i++ {
		x1 := float32(i) * pointSpacing
		y1 := height - float32(dataPoints[i].Upload)*height/float32(maxValue)
		x2 := float32(i+1) * pointSpacing
		y2 := height - float32(dataPoints[i+1].Upload)*height/float32(maxValue)
		line := canvas.NewLine(uploadColor)
		line.Position1 = fyne.NewPos(x1, y1)
		line.Position2 = fyne.NewPos(x2, y2)
		line.StrokeWidth = 2
		r.uploadLines = append(r.uploadLines, line)
	}
	// 绘制下载线（连接所有点）
	for i := 0; i < len(dataPoints)-1; i++ {
		x1 := float32(i) * pointSpacing
		y1 := height - float32(dataPoints[i].Download)*height/float32(maxValue)
		x2 := float32(i+1) * pointSpacing
		y2 := height - float32(dataPoints[i+1].Download)*height/float32(maxValue)
		line := canvas.NewLine(downloadColor)
		line.Position1 = fyne.NewPos(x1, y1)
		line.Position2 = fyne.NewPos(x2, y2)
		line.StrokeWidth = 2
		r.downloadLines = append(r.downloadLines, line)
	}
}

// Refresh 刷新
func (r *trafficChartRenderer) Refresh() {
	r.trafficChart.mu.RLock()
	upload := r.trafficChart.currentUpload
	download := r.trafficChart.currentDownload
	size := r.trafficChart.Size()
	r.trafficChart.mu.RUnlock()

	// 使用当前主题色更新背景，切换主题后能立即生效
	if r.trafficChart.appState != nil && r.trafficChart.appState.App != nil {
		r.bgRect.FillColor = CurrentThemeColor(r.trafficChart.appState.App, theme.ColorNameBackground)
		r.bgRect.Refresh()
	}

	// 更新标签
	r.uploadLabel.SetText(fmt.Sprintf("上传: %s", formatSpeed(upload)))
	r.downloadLabel.SetText(fmt.Sprintf("下载: %s", formatSpeed(download)))

	// 重新绘制图表（折线会使用当前主题色）
	r.Layout(size)

	// 刷新所有对象
	for _, obj := range r.Objects() {
		obj.Refresh()
	}
}

// Objects 返回所有对象
func (r *trafficChartRenderer) Objects() []fyne.CanvasObject {
	// 清空并重新构建对象列表
	r.objects = r.objects[:0]
	r.objects = append(r.objects, r.bgRect)

	// 添加所有上传线
	for _, line := range r.uploadLines {
		r.objects = append(r.objects, line)
	}

	// 添加所有下载线
	for _, line := range r.downloadLines {
		r.objects = append(r.objects, line)
	}

	r.objects = append(r.objects, r.uploadLabel, r.downloadLabel)
	return r.objects
}

// Destroy 销毁
func (r *trafficChartRenderer) Destroy() {
}

// toRGBA 将 theme 返回的 color.Color 转为 color.RGBA，便于 canvas 使用。
// c 为 nil 时使用默认主题背景色。
func toRGBA(c color.Color) color.RGBA {
	if c == nil {
		c = theme.DefaultTheme().Color(theme.ColorNameBackground, theme.VariantDark)
	}
	r, g, b, a := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

// formatSpeed 格式化速度显示
func formatSpeed(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	var value float64
	var unit string

	switch {
	case bytes >= GB:
		value = float64(bytes) / GB
		unit = "GB/s"
	case bytes >= MB:
		value = float64(bytes) / MB
		unit = "MB/s"
	case bytes >= KB:
		value = float64(bytes) / KB
		unit = "KB/s"
	default:
		value = float64(bytes)
		unit = "B/s"
	}

	if value < 10 {
		return fmt.Sprintf("%.2f %s", value, unit)
	} else if value < 100 {
		return fmt.Sprintf("%.1f %s", value, unit)
	} else {
		return fmt.Sprintf("%.0f %s", value, unit)
	}
}
