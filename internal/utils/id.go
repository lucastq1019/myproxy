package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateServerID 生成服务器唯一ID。
// 参数：
//   - addr: 服务器地址
//   - port: 服务器端口
//   - username: 用户名（用于唯一性）
// 返回：服务器唯一ID（MD5哈希）
func GenerateServerID(addr string, port int, username string) string {
	// 使用地址、端口和用户名生成唯一ID
	data := fmt.Sprintf("%s:%d:%s:%d", addr, port, username, time.Now().UnixNano())
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

