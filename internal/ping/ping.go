package ping

import (
	"fmt"
	"net"
	"sync"
	"time"

	"myproxy.com/p/internal/config"
	"myproxy.com/p/internal/server"
)

// PingManager 延迟测试管理器
type PingManager struct {
	serverManager *server.ServerManager
}

// NewPingManager 创建新的延迟测试管理器
func NewPingManager(serverManager *server.ServerManager) *PingManager {
	return &PingManager{
		serverManager: serverManager,
	}
}

// TestServerDelay 测试单个服务器延迟
func (pm *PingManager) TestServerDelay(server config.Server) (int, error) {
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

// TestAllServersDelay 测试所有服务器延迟
func (pm *PingManager) TestAllServersDelay() map[string]int {
	// 获取所有服务器
	servers := pm.serverManager.ListServers()
	results := make(map[string]int)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 并发测试每个服务器
	for _, s := range servers {
		if !s.Enabled {
			continue
		}

		wg.Add(1)
		go func(server config.Server) {
			defer wg.Done()

			delay, err := pm.TestServerDelay(server)
			mu.Lock()
			if err != nil {
				results[server.ID] = -1
			} else {
				results[server.ID] = delay
				// 更新服务器延迟
				pm.serverManager.UpdateServerDelay(server.ID, delay)
			}
			mu.Unlock()
		}(s)
	}

	// 等待所有测试完成
	wg.Wait()

	return results
}
