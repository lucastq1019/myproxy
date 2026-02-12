package service

import (
	"strings"
	"sync"

	"myproxy.com/p/internal/store"
)

// AccessRecordService 访问记录服务，提供从日志解析并记录访问记录的能力。
type AccessRecordService struct {
	store *store.Store

	// 批量模式：用于 loadInitialLogs 等场景，避免逐行写入 DB
	mu           sync.Mutex
	batchMode    bool
	batchCounts  map[string]int64
}

// xray 访问日志格式（空格分割）：第 6 个字段为 host:port
// 示例: 2026/02/12 10:20:40.159520 from tcp:127.0.0.1:52101 accepted tcp:api2.cursor.sh:443 [socks-in -> proxy]
// 示例: 2026/02/12 10:20:42.465015 from 127.0.0.1:52117 accepted //www.google.com:443 [socks-in -> proxy]
// 字段索引: 0          1               2    3                   4        5

// NewAccessRecordService 创建访问记录服务实例。
func NewAccessRecordService(store *store.Store) *AccessRecordService {
	return &AccessRecordService{store: store}
}

// StartBatch 开启批量模式，后续 RecordAccessFromLogLine 将累积到内存，由 EndBatch 统一写入。
func (ars *AccessRecordService) StartBatch() {
	ars.mu.Lock()
	ars.batchMode = true
	ars.batchCounts = make(map[string]int64)
	ars.mu.Unlock()
}

// EndBatch 结束批量模式，将累积的访问记录写入 DB。
func (ars *AccessRecordService) EndBatch() error {
	ars.mu.Lock()
	batchMode := ars.batchMode
	counts := ars.batchCounts
	ars.batchMode = false
	ars.batchCounts = nil
	ars.mu.Unlock()

	if !batchMode || len(counts) == 0 || ars.store == nil || ars.store.AccessRecords == nil {
		return nil
	}
	return ars.store.AccessRecords.RecordAccessBatch(counts)
}

// RecordAccessFromLogLine 解析日志行，若为 xray 访问日志则提取 address (host:port) 并记录。
// 批量模式下累积到内存；否则立即写入 DB。
// 返回：是否成功记录（true 表示解析到并记录了地址）。
func (ars *AccessRecordService) RecordAccessFromLogLine(line string) bool {
	address := extractAddressFromXrayAccessLine(line)
	if address == "" {
		return false
	}
	if ars.store == nil || ars.store.AccessRecords == nil {
		return false
	}

	ars.mu.Lock()
	if ars.batchMode {
		ars.batchCounts[address]++
		ars.mu.Unlock()
		return true
	}
	ars.mu.Unlock()

	_ = ars.store.AccessRecords.RecordAccess(address, 1, 0, 0)
	return true
}

// ExtractAddressFromLogLine 解析日志行提取 address (host:port)，供批量处理使用。
func (ars *AccessRecordService) ExtractAddressFromLogLine(line string) string {
	return extractAddressFromXrayAccessLine(line)
}

// RecordAccessBatchFromLines 批量解析日志行并记录访问。
func (ars *AccessRecordService) RecordAccessBatchFromLines(lines []string) error {
	if ars.store == nil || ars.store.AccessRecords == nil {
		return nil
	}
	addressCounts := make(map[string]int64)
	for _, line := range lines {
		addr := extractAddressFromXrayAccessLine(line)
		if addr != "" {
			addressCounts[addr]++
		}
	}
	return ars.store.AccessRecords.RecordAccessBatch(addressCounts)
}

// RecordAccessBatchFromAddressCounts 根据已统计的地址计数批量记录。
func (ars *AccessRecordService) RecordAccessBatchFromAddressCounts(addressCounts map[string]int64) error {
	if ars.store == nil || ars.store.AccessRecords == nil {
		return nil
	}
	return ars.store.AccessRecords.RecordAccessBatch(addressCounts)
}

// extractAddressFromXrayAccessLine 从 xray 访问日志行提取 address (host:port)，保留端口信息。
// 仅解析包含 "accepted" 的 xray 代理访问日志，排除 app 日志和 xray 启动等日志。
// 规则：定位 "accepted" 后取其后的第一个 token 为 host:port，兼容有无时间戳两种格式：
//   - 有 timestamp: 2026/02/12 10:43:05.230386 from tcp:127.0.0.1:59593 accepted tcp:api2.cursor.sh:443
//   - 无 timestamp: from tcp:127.0.0.1:49379 accepted tcp:api2.cursor.sh:443
func extractAddressFromXrayAccessLine(line string) string {
	idx := strings.Index(line, "accepted")
	if idx == -1 {
		return ""
	}
	rest := strings.TrimSpace(line[idx+len("accepted"):])
	fields := strings.Fields(rest)
	if len(fields) < 1 {
		return ""
	}
	hostPort := fields[0]
	// 去掉 tcp:/udp: 前缀
	if strings.HasPrefix(hostPort, "tcp:") || strings.HasPrefix(hostPort, "udp:") {
		hostPort = hostPort[4:]
	}
	// 去掉 // 前缀
	hostPort = strings.TrimPrefix(hostPort, "//")
	// 保留 :port，不剥离
	address := strings.TrimSpace(hostPort)
	if address == "" || len(address) > 268 || strings.Contains(address, " ") {
		return ""
	}
	// 跳过纯 IP（不含端口则无法判断，含端口时 host 部分可能是 IP）
	host := address
	if idx := strings.LastIndex(address, ":"); idx > 0 {
		host = address[:idx]
	}
	if isIPLike(host) {
		return ""
	}
	return address
}

func isIPLike(s string) bool {
	// 简单判断：仅含数字和点
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			continue
		}
		return false
	}
	return strings.Contains(s, ".")
}
