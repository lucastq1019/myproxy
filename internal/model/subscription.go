package model

import "time"

// Subscription 表示一个订阅配置，包含 URL 和标签信息。
type Subscription struct {
	ID        int64     `json:"id"`
	URL       string    `json:"url"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
