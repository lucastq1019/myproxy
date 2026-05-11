package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// MetricChart 是一个轻量趋势图组件。
type MetricChart struct {
	widget.BaseWidget

	appState     *AppState
	title        string
	currentValue string
	points       []float64
	lineColor    color.Color
}

// NewMetricChart 创建趋势图组件。
func NewMetricChart(appState *AppState, title string, lineColor color.Color) *MetricChart {
	mc := &MetricChart{
		appState:  appState,
		title:     title,
		lineColor: lineColor,
		points:    make([]float64, 0),
	}
	mc.ExtendBaseWidget(mc)
	return mc
}

// SetData 更新图表数据。
func (mc *MetricChart) SetData(points []float64, currentValue string) {
	mc.points = append(mc.points[:0], points...)
	mc.currentValue = currentValue
	mc.Refresh()
}

func (mc *MetricChart) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(CurrentThemeColor(mc.appState.App, theme.ColorNameInputBackground))
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = CurrentThemeColor(mc.appState.App, theme.ColorNameSeparator)
	border.StrokeWidth = 1
	return &metricChartRenderer{
		chart:        mc,
		bg:           bg,
		border:       border,
		titleLabel:   widget.NewLabel(mc.title),
		currentLabel: widget.NewLabel(""),
		objects:      make([]fyne.CanvasObject, 0),
	}
}

type metricChartRenderer struct {
	chart        *MetricChart
	bg           *canvas.Rectangle
	border       *canvas.Rectangle
	titleLabel   *widget.Label
	currentLabel *widget.Label
	lines        []*canvas.Line
	objects      []fyne.CanvasObject
}

func (r *metricChartRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.border.Resize(size)

	padding := float32(8)
	headerHeight := float32(20)
	r.titleLabel.Move(fyne.NewPos(padding, padding))
	r.titleLabel.Resize(fyne.NewSize(size.Width/2, headerHeight))
	r.currentLabel.Move(fyne.NewPos(size.Width/2, padding))
	r.currentLabel.Resize(fyne.NewSize(size.Width/2-padding, headerHeight))

	chartTop := padding + headerHeight + 6
	chartHeight := size.Height - chartTop - padding
	chartWidth := size.Width - 2*padding
	if chartWidth <= 0 || chartHeight <= 0 {
		return
	}

	r.lines = r.lines[:0]
	points := r.chart.points
	if len(points) < 2 {
		return
	}

	minValue := points[0]
	maxValue := points[0]
	for _, point := range points[1:] {
		if point < minValue {
			minValue = point
		}
		if point > maxValue {
			maxValue = point
		}
	}
	if maxValue == minValue {
		maxValue++
	}

	step := chartWidth / float32(len(points)-1)
	for i := 0; i < len(points)-1; i++ {
		x1 := padding + float32(i)*step
		x2 := padding + float32(i+1)*step
		y1 := chartTop + chartHeight - float32((points[i]-minValue)/(maxValue-minValue))*chartHeight
		y2 := chartTop + chartHeight - float32((points[i+1]-minValue)/(maxValue-minValue))*chartHeight
		line := canvas.NewLine(r.chart.lineColor)
		line.Position1 = fyne.NewPos(x1, y1)
		line.Position2 = fyne.NewPos(x2, y2)
		line.StrokeWidth = 2
		r.lines = append(r.lines, line)
	}
}

func (r *metricChartRenderer) MinSize() fyne.Size {
	return fyne.NewSize(220, 120)
}

func (r *metricChartRenderer) Refresh() {
	r.bg.FillColor = CurrentThemeColor(r.chart.appState.App, theme.ColorNameInputBackground)
	r.border.StrokeColor = CurrentThemeColor(r.chart.appState.App, theme.ColorNameSeparator)
	r.titleLabel.SetText(r.chart.title)
	r.currentLabel.SetText(r.chart.currentValue)
	r.currentLabel.Alignment = fyne.TextAlignTrailing
	r.Layout(r.chart.Size())
	canvas.Refresh(r.bg)
	canvas.Refresh(r.border)
	r.titleLabel.Refresh()
	r.currentLabel.Refresh()
	for _, obj := range r.lines {
		obj.Refresh()
	}
}

func (r *metricChartRenderer) Objects() []fyne.CanvasObject {
	r.objects = r.objects[:0]
	r.objects = append(r.objects, r.bg, r.border, r.titleLabel, r.currentLabel)
	for _, line := range r.lines {
		r.objects = append(r.objects, line)
	}
	return r.objects
}

func (r *metricChartRenderer) Destroy() {}

func (r *metricChartRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

func (r *metricChartRenderer) String() string {
	return fmt.Sprintf("MetricChart(%s)", r.chart.title)
}
