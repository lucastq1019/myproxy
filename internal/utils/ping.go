package utils

import (
	"fmt"
	"net"
	"sync"
	"time"

	"myproxy.com/p/internal/model"
)

// Ping 延迟测试工具。
// 负责测试服务器延迟，不涉及数据更新操作。
type Ping struct {
}

// NewPing 创建新的延迟测试工具实例。
// 返回：初始化后的 Ping 实例
func NewPing() *Ping {
	return &Ping{}
}

// TestServerDelay 测试单个服务器延迟。
// 参数：
//   - server: 服务器节点
//
// 返回：延迟值（毫秒）和错误（如果有）
func (p *Ping) TestServerDelay(server model.Node) (int, error) {
	// 使用TCP连接测试延迟
	addr := fmt.Sprintf("%s:%d", server.Addr, server.Port)
	start := time.Now()

	// 尝试建立TCP连接
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return -1, fmt.Errorf("连接服务器失败: %w", err)
	}
	defer conn.Close()

	// 计算延迟
	delay := int(time.Since(start).Milliseconds())
	return delay, nil
}

// TestAllServersDelay 测试多个服务器延迟。
// 参数：
//   - servers: 服务器节点列表
//
// 返回：服务器ID到延迟值的映射（-1表示测试失败）
func (p *Ping) TestAllServersDelay(servers []model.Node) map[string]int {
	results := make(map[string]int)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 并发测试每个服务器
	for _, server := range servers {
		if !server.Enabled {
			continue
		}

		wg.Add(1)
		go func(s model.Node) {
			defer wg.Done()

			delay, err := p.TestServerDelay(s)
			mu.Lock()
			if err != nil {
				results[s.ID] = -1
			} else {
				results[s.ID] = delay
			}
			mu.Unlock()
		}(server)
	}

	// 等待所有测试完成
	wg.Wait()

	return results
}
