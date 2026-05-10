package ui

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"myproxy.com/p/internal/model"
)

// DiagnosticsPage 展示运行时诊断信息。
type DiagnosticsPage struct {
	appState *AppState
	content  fyne.CanvasObject

	pprofCheck    *widget.Check
	pprofAddr     *widget.Entry
	samplingSel   *widget.Select
	overviewLabel *widget.Label
	exportLabel   *widget.Label
	memChart      *MetricChart
	gorChart      *MetricChart

	ticker      *time.Ticker
	stopCh      chan struct{}
	cleanupOnce sync.Once
}

// NewDiagnosticsPage 创建诊断页。
func NewDiagnosticsPage(appState *AppState) *DiagnosticsPage {
	return &DiagnosticsPage{appState: appState}
}

// Build 构建诊断页。
func (dp *DiagnosticsPage) Build() fyne.CanvasObject {
	if dp.content != nil {
		dp.Refresh()
		return dp.content
	}

	spacing := innerPadding(dp.appState)
	dp.overviewLabel = widget.NewLabel("")
	dp.overviewLabel.Wrapping = fyne.TextWrapWord
	dp.exportLabel = widget.NewLabel("")
	dp.exportLabel.Wrapping = fyne.TextWrapWord

	pprofEnabled := false
	pprofAddr := "127.0.0.1:6060"
	samplingSeconds := "5 秒"
	if dp.appState != nil && dp.appState.DiagnosticsService != nil {
		pprofEnabled = dp.appState.DiagnosticsService.IsPprofEnabled()
		pprofAddr = dp.appState.DiagnosticsService.GetPprofAddr()
	}
	if dp.appState != nil && dp.appState.ConfigService != nil {
		samplingSeconds = fmt.Sprintf("%d 秒", dp.appState.ConfigService.GetDiagnosticsSamplingSeconds())
	}

	dp.pprofCheck = widget.NewCheck("启用本地 pprof", func(enabled bool) {
		if dp.appState == nil || dp.appState.ConfigService == nil || dp.appState.DiagnosticsService == nil {
			return
		}
		_ = dp.appState.ConfigService.SetDebugPprofEnabled(enabled)
		if err := dp.appState.DiagnosticsService.ApplyPprofConfig(); err != nil {
			dp.showError(err)
		}
		dp.Refresh()
	})

	dp.pprofAddr = widget.NewEntry()
	dp.pprofAddr.SetText(pprofAddr)
	savePprofBtn := widget.NewButtonWithIcon("保存地址", theme.DocumentSaveIcon(), func() {
		if dp.appState == nil || dp.appState.ConfigService == nil || dp.appState.DiagnosticsService == nil {
			return
		}
		if err := dp.appState.ConfigService.SetDebugPprofAddr(dp.pprofAddr.Text); err != nil {
			dp.showError(err)
			return
		}
		if err := dp.appState.DiagnosticsService.ApplyPprofConfig(); err != nil {
			dp.showError(err)
			return
		}
		dp.setExportStatus("pprof 地址已更新")
		dp.Refresh()
	})
	savePprofBtn.Importance = widget.LowImportance

	dp.samplingSel = widget.NewSelect([]string{"1 秒", "5 秒", "10 秒"}, func(value string) {
		if dp.appState == nil || dp.appState.ConfigService == nil {
			return
		}
		seconds := 5
		switch value {
		case "1 秒":
			seconds = 1
		case "10 秒":
			seconds = 10
		}
		_ = dp.appState.ConfigService.SetDiagnosticsSamplingSeconds(seconds)
		dp.setExportStatus("采样周期已保存，重启应用后完全生效")
	})

	dp.memChart = NewMetricChart(dp.appState, "内存趋势", ChartUploadColor(dp.appState.App))
	dp.gorChart = NewMetricChart(dp.appState, "Goroutine 趋势", ChartDownloadColor(dp.appState.App))
	dp.pprofCheck.SetChecked(pprofEnabled)
	dp.samplingSel.SetSelected(samplingSeconds)

	browserRow := container.NewGridWithColumns(3,
		widget.NewButtonWithIcon("浏览器：pprof 首页", theme.ComputerIcon(), func() {
			if dp.appState == nil || dp.appState.DiagnosticsService == nil {
				return
			}
			raw, err := dp.appState.DiagnosticsService.PprofIndexPageURL()
			dp.openPprofURL(raw, err)
		}),
		widget.NewButtonWithIcon("浏览器：堆内存(文本)", theme.DocumentIcon(), func() {
			if dp.appState == nil || dp.appState.DiagnosticsService == nil {
				return
			}
			raw, err := dp.appState.DiagnosticsService.PprofHeapTextURL()
			dp.openPprofURL(raw, err)
		}),
		widget.NewButtonWithIcon("浏览器：交互+火焰图", theme.ViewFullScreenIcon(), func() {
			if dp.appState == nil || dp.appState.DiagnosticsService == nil {
				return
			}
			dp.setExportStatus("正在启动 go tool pprof...")
			go func() {
				viewerURL, err := dp.appState.DiagnosticsService.StartGoToolPprofHeapWebViewer()
				fyne.Do(func() {
					if err != nil {
						dp.showError(err)
						dp.setExportStatus(err.Error())
						return
					}
					u, perr := url.Parse(viewerURL)
					if perr != nil {
						dp.showError(perr)
						return
					}
					if dp.appState.App == nil {
						return
					}
					if err := dp.appState.App.OpenURL(u); err != nil {
						dp.showError(err)
						return
					}
					dp.setExportStatus("已在浏览器打开交互式剖面: " + viewerURL)
				})
			}()
		}),
	)

	buttonsRow1 := container.NewGridWithColumns(2,
		widget.NewButtonWithIcon("导出堆快照", theme.DownloadIcon(), func() {
			dp.runAsyncAction("正在导出堆快照...", func() (string, error) {
				path, err := dp.appState.DiagnosticsService.ExportHeapProfile()
				if err != nil {
					return "", err
				}
				summary := dp.currentSummary()
				_, _ = dp.appState.DiagnosticsService.ExportSummaryJSON(summary)
				return "堆快照已导出: " + path, nil
			})
		}),
		widget.NewButtonWithIcon("导出 Goroutine 快照", theme.DownloadIcon(), func() {
			dp.runAsyncAction("正在导出 Goroutine 快照...", func() (string, error) {
				path, err := dp.appState.DiagnosticsService.ExportGoroutineProfile()
				if err != nil {
					return "", err
				}
				summary := dp.currentSummary()
				_, _ = dp.appState.DiagnosticsService.ExportSummaryJSON(summary)
				return "Goroutine 快照已导出: " + path, nil
			})
		}),
	)
	buttonsRow2 := container.NewGridWithColumns(2,
		widget.NewButtonWithIcon("生成火焰图", theme.MediaPlayIcon(), func() {
			dp.runAsyncAction("正在生成火焰图...", func() (string, error) {
				profilePath, svgPath, err := dp.appState.DiagnosticsService.GenerateHeapFlameGraph()
				if err != nil {
					if profilePath != "" {
						return "", fmt.Errorf("%w（已保留 profile: %s）", err, profilePath)
					}
					return "", err
				}
				summary := dp.currentSummary()
				_, _ = dp.appState.DiagnosticsService.ExportSummaryJSON(summary)
				return "火焰图已生成: " + svgPath, nil
			})
		}),
		widget.NewButtonWithIcon("打开诊断目录", theme.FolderOpenIcon(), func() {
			dp.runAsyncAction("正在打开诊断目录...", func() (string, error) {
				if err := dp.appState.DiagnosticsService.OpenDiagnosticsDirectory(); err != nil {
					return "", err
				}
				return "诊断目录已打开: " + dp.currentSummary().DiagnosticsDir, nil
			})
		}),
	)
	buttonsRow3 := container.NewGridWithColumns(2,
		widget.NewButtonWithIcon("复制诊断摘要", theme.ContentCopyIcon(), func() {
			summary := formatDiagnosticSummary(dp.currentSummary())
			if dp.appState != nil && dp.appState.Window != nil {
				dp.appState.Window.Clipboard().SetContent(summary)
			}
			dp.setExportStatus("诊断摘要已复制到剪贴板")
		}),
		widget.NewButtonWithIcon("导出摘要 JSON", theme.DocumentCreateIcon(), func() {
			dp.runAsyncAction("正在导出诊断摘要...", func() (string, error) {
				path, err := dp.appState.DiagnosticsService.ExportSummaryJSON(dp.currentSummary())
				if err != nil {
					return "", err
				}
				return "诊断摘要已导出: " + path, nil
			})
		}),
	)

	configCard := container.NewVBox(
		widget.NewLabelWithStyle("诊断配置", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		dp.pprofCheck,
		container.NewBorder(nil, nil, nil, savePprofBtn, dp.pprofAddr),
		widget.NewLabel("采样周期"),
		dp.samplingSel,
		widget.NewLabelWithStyle("浏览器调试（需已启用 pprof；火焰图需本机安装 Go）", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		browserRow,
	)

	overviewCard := container.NewVBox(
		widget.NewLabelWithStyle("运行概览", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		dp.overviewLabel,
		widget.NewSeparator(),
		dp.exportLabel,
	)

	content := newCompactVBox(spacing,
		configCard,
		widget.NewSeparator(),
		overviewCard,
		widget.NewSeparator(),
		dp.memChart,
		dp.gorChart,
		widget.NewSeparator(),
		buttonsRow1,
		buttonsRow2,
		buttonsRow3,
	)

	dp.content = newPaddedWithSize(content, spacing)
	dp.startAutoRefresh()
	dp.Refresh()
	return dp.content
}

// Refresh 刷新诊断页。
func (dp *DiagnosticsPage) Refresh() {
	if dp.appState == nil || dp.appState.DiagnosticsService == nil || dp.overviewLabel == nil {
		return
	}

	summary := dp.currentSummary()
	dp.overviewLabel.SetText(formatDiagnosticSummary(summary))
	dp.refreshCharts(dp.appState.DiagnosticsService.History())

	if !summary.PprofEnabled {
		dp.setExportStatusIfEmpty("pprof 未启用，仍可导出 profile，但 HTTP 调试端口不会监听。")
	}
}

// Cleanup 停止自动刷新（可重复调用；仅首次关闭 ticker 与 stopCh）。
func (dp *DiagnosticsPage) Cleanup() {
	if dp == nil {
		return
	}
	dp.cleanupOnce.Do(func() {
		if dp.ticker != nil {
			dp.ticker.Stop()
			dp.ticker = nil
		}
		if dp.stopCh != nil {
			close(dp.stopCh)
			dp.stopCh = nil
		}
	})
}

func (dp *DiagnosticsPage) startAutoRefresh() {
	if dp.ticker != nil {
		return
	}
	dp.ticker = time.NewTicker(5 * time.Second)
	dp.stopCh = make(chan struct{})
	go func() {
		for {
			select {
			case <-dp.ticker.C:
				fyne.Do(func() { dp.Refresh() })
			case <-dp.stopCh:
				return
			}
		}
	}()
}

func (dp *DiagnosticsPage) refreshCharts(history []model.DiagnosticSnapshot) {
	if dp == nil || dp.memChart == nil || dp.gorChart == nil || dp.appState == nil || dp.appState.DiagnosticsService == nil {
		return
	}

	memSeries := make([]float64, 0, len(history))
	gorSeries := make([]float64, 0, len(history))
	for _, item := range history {
		memSeries = append(memSeries, float64(item.HeapInuse))
		gorSeries = append(gorSeries, float64(item.Goroutines))
	}

	current := dp.appState.DiagnosticsService.CurrentSnapshot()
	dp.memChart.SetData(memSeries, "当前 "+formatBytes(current.HeapInuse))
	dp.gorChart.SetData(gorSeries, fmt.Sprintf("当前 %d", current.Goroutines))
}

func (dp *DiagnosticsPage) currentSummary() model.DiagnosticSummary {
	serverName := "无"
	proxyRunning := false
	proxyPort := 0
	if dp.appState != nil {
		if dp.appState.Store != nil && dp.appState.Store.Nodes != nil {
			if selected := dp.appState.Store.Nodes.GetSelected(); selected != nil {
				serverName = selected.Name
			}
		}
		if dp.appState.XrayInstance != nil && dp.appState.XrayInstance.IsRunning() {
			proxyRunning = true
			proxyPort = dp.appState.XrayInstance.GetPort()
		}
	}
	return dp.appState.DiagnosticsService.GetSummary(proxyRunning, proxyPort, serverName)
}

// openPprofURL 在系统默认浏览器中打开诊断相关 URL（raw 为完整 http(s) 地址）。
func (dp *DiagnosticsPage) openPprofURL(raw string, err error) {
	if err != nil {
		dp.showError(err)
		return
	}
	u, err := url.Parse(raw)
	if err != nil {
		dp.showError(err)
		return
	}
	if dp.appState == nil || dp.appState.App == nil {
		dp.showError(fmt.Errorf("应用未就绪"))
		return
	}
	if err := dp.appState.App.OpenURL(u); err != nil {
		dp.showError(err)
		return
	}
	dp.setExportStatus("已在浏览器打开: " + raw)
}

func (dp *DiagnosticsPage) runAsyncAction(startText string, fn func() (string, error)) {
	dp.setExportStatus(startText)
	go func() {
		message, err := fn()
		fyne.Do(func() {
			if err != nil {
				dp.showError(err)
				dp.setExportStatus(err.Error())
				return
			}
			dp.setExportStatus(message)
			dp.Refresh()
		})
	}()
}

func (dp *DiagnosticsPage) showError(err error) {
	if dp.appState != nil && dp.appState.Window != nil {
		dialog.ShowError(err, dp.appState.Window)
	}
	if dp.appState != nil {
		dp.appState.AppendLog("ERROR", "app", err.Error())
	}
}

func (dp *DiagnosticsPage) setExportStatus(message string) {
	if dp.exportLabel != nil {
		dp.exportLabel.SetText(message)
	}
}

func (dp *DiagnosticsPage) setExportStatusIfEmpty(message string) {
	if dp.exportLabel != nil && strings.TrimSpace(dp.exportLabel.Text) == "" {
		dp.exportLabel.SetText(message)
	}
}

func formatDiagnosticSummary(summary model.DiagnosticSummary) string {
	lastNodeSwitch := "未记录"
	if !summary.LastNodeSwitchAt.IsZero() {
		lastNodeSwitch = summary.LastNodeSwitchAt.Format("2006-01-02 15:04:05")
	}
	lastSubscriptionUpdate := "未记录"
	if !summary.LastSubscriptionUpdateAt.IsZero() {
		lastSubscriptionUpdate = summary.LastSubscriptionUpdateAt.Format("2006-01-02 15:04:05")
	}
	lastExport := summary.LastDiagnosticExport
	if lastExport == "" {
		lastExport = "未导出"
	}

	return fmt.Sprintf(
		"代理状态: %s\n当前节点: %s\n监听端口: %d\nHeapInuse: %s\nAlloc: %s\nSys: %s\nGoroutines: %d\nGC 次数: %d\npprof: %t (%s)\n最近节点切换: %s\n最近订阅更新: %s\n最近诊断导出: %s",
		boolText(summary.ProxyRunning, "运行中", "未运行"),
		summary.CurrentServerName,
		summary.ProxyPort,
		formatBytes(summary.Current.HeapInuse),
		formatBytes(summary.Current.Alloc),
		formatBytes(summary.Current.Sys),
		summary.Current.Goroutines,
		summary.Current.NumGC,
		summary.PprofEnabled,
		summary.PprofAddr,
		lastNodeSwitch,
		lastSubscriptionUpdate,
		lastExport,
	)
}

func formatBytes(value uint64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case value >= gb:
		return fmt.Sprintf("%.2f GB", float64(value)/float64(gb))
	case value >= mb:
		return fmt.Sprintf("%.2f MB", float64(value)/float64(mb))
	case value >= kb:
		return fmt.Sprintf("%.2f KB", float64(value)/float64(kb))
	default:
		return fmt.Sprintf("%d B", value)
	}
}

func boolText(v bool, yes, no string) string {
	if v {
		return yes
	}
	return no
}
