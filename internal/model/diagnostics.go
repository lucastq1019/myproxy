package model

import "time"

// DiagnosticSnapshot 表示一次运行时采样结果。
type DiagnosticSnapshot struct {
	Timestamp  time.Time `json:"timestamp"`
	Alloc      uint64    `json:"alloc"`
	HeapAlloc  uint64    `json:"heapAlloc"`
	HeapInuse  uint64    `json:"heapInuse"`
	Sys        uint64    `json:"sys"`
	NumGC      uint32    `json:"numGC"`
	Goroutines int       `json:"goroutines"`
}

// DiagnosticSummary 表示诊断页展示和导出的汇总信息。
type DiagnosticSummary struct {
	Timestamp                time.Time          `json:"timestamp"`
	GoVersion                string             `json:"goVersion"`
	ExecutablePath           string             `json:"executablePath"`
	DiagnosticsDir           string             `json:"diagnosticsDir"`
	PprofEnabled             bool               `json:"pprofEnabled"`
	PprofAddr                string             `json:"pprofAddr"`
	ProxyRunning             bool               `json:"proxyRunning"`
	ProxyPort                int                `json:"proxyPort"`
	CurrentServerName        string             `json:"currentServerName"`
	LastNodeSwitchAt         time.Time          `json:"lastNodeSwitchAt"`
	LastSubscriptionUpdateAt time.Time          `json:"lastSubscriptionUpdateAt"`
	LastDiagnosticExport     string             `json:"lastDiagnosticExport"`
	Current                  DiagnosticSnapshot `json:"current"`
}
