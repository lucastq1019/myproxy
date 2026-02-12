package model

import "time"

// AccessRecord 访问记录，用于流量分析。
// 记录访问的网站及累计访问次数，便于后续分析。
type AccessRecord struct {
	ID           int64     `json:"id"`
	Domain       string    `json:"domain"`        // 访问的域名（兼容旧数据，新数据同 Address 的 host 部分）
	Address      string    `json:"address"`       // 完整地址 host:port，如 api2.cursor.sh:443
	AccessCount  int64     `json:"accessCount"`  // 累计访问次数
	UploadBytes  int64     `json:"uploadBytes"`  // 累计上传字节（暂不支持，保留字段）
	DownloadBytes int64    `json:"downloadBytes"` // 累计下载字节（暂不支持，保留字段）
	FirstSeen    time.Time `json:"firstSeen"`   // 首次访问时间
	LastSeen     time.Time `json:"lastSeen"`    // 最近访问时间
}
