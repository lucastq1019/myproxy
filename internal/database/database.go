package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"myproxy.com/p/internal/config"
)

// DB 数据库连接
var DB *sql.DB

// InitDB 初始化 SQLite 数据库，创建必要的表结构。
// 如果数据库文件不存在，会自动创建。如果表已存在，不会重复创建。
// 参数：
//   - dbPath: 数据库文件路径
//
// 返回：错误（如果有）
func InitDB(dbPath string) error {
	// 创建目录（如果不存在）
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("创建数据库目录失败: %w", err)
	}

	// 打开数据库连接
	var err error
	DB, err = sql.Open("sqlite3", dbPath+"?_foreign_keys=1")
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}

	// 测试连接
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 创建表
	if err := createTables(); err != nil {
		return fmt.Errorf("创建表失败: %w", err)
	}

	return nil
}

// createTables 创建数据库表
func createTables() error {
	// 创建订阅表
	createSubscriptionsTable := `
	CREATE TABLE IF NOT EXISTS subscriptions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL UNIQUE,
		label TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	// 创建服务器表
	createServersTable := `
	CREATE TABLE IF NOT EXISTS servers (
		id TEXT PRIMARY KEY,
		subscription_id INTEGER,
		name TEXT NOT NULL,
		addr TEXT NOT NULL,
		port INTEGER NOT NULL,
		username TEXT NOT NULL DEFAULT '',
		password TEXT NOT NULL DEFAULT '',
		delay INTEGER NOT NULL DEFAULT 0,
		selected INTEGER NOT NULL DEFAULT 0,
		enabled INTEGER NOT NULL DEFAULT 1,
		node_protocol_type TEXT NOT NULL DEFAULT 'socks5',
		vmess_version TEXT DEFAULT '',
		vmess_uuid TEXT DEFAULT '',
		vmess_alter_id INTEGER DEFAULT 0,
		vmess_security TEXT DEFAULT '',
		vmess_network TEXT DEFAULT '',
		vmess_type TEXT DEFAULT '',
		vmess_host TEXT DEFAULT '',
		vmess_path TEXT DEFAULT '',
		vmess_tls TEXT DEFAULT '',
		ss_method TEXT DEFAULT '',
		ss_plugin TEXT DEFAULT '',
		ss_plugin_opts TEXT DEFAULT '',
		ssr_obfs TEXT DEFAULT '',
		ssr_obfs_param TEXT DEFAULT '',
		ssr_protocol TEXT DEFAULT '',
		ssr_protocol_param TEXT DEFAULT '',
		raw_config TEXT DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE SET NULL
	);`

	// 创建布局配置表（用于存储窗口布局配置）
	createLayoutConfigTable := `
	CREATE TABLE IF NOT EXISTS layout_config (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL UNIQUE,
		value TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	// 创建应用配置表（用于存储应用配置，如日志级别、日志文件路径、主题等）
	createAppConfigTable := `
	CREATE TABLE IF NOT EXISTS app_config (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL UNIQUE,
		value TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	// 创建索引
	createIndexes := `
	CREATE INDEX IF NOT EXISTS idx_servers_subscription_id ON servers(subscription_id);
	CREATE INDEX IF NOT EXISTS idx_servers_enabled ON servers(enabled);
	CREATE INDEX IF NOT EXISTS idx_subscriptions_url ON subscriptions(url);
	CREATE INDEX IF NOT EXISTS idx_layout_config_key ON layout_config(key);
	CREATE INDEX IF NOT EXISTS idx_app_config_key ON app_config(key);
	`

	if _, err := DB.Exec(createSubscriptionsTable); err != nil {
		return fmt.Errorf("创建订阅表失败: %w", err)
	}

	if _, err := DB.Exec(createServersTable); err != nil {
		return fmt.Errorf("创建服务器表失败: %w", err)
	}

	if _, err := DB.Exec(createLayoutConfigTable); err != nil {
		return fmt.Errorf("创建布局配置表失败: %w", err)
	}

	if _, err := DB.Exec(createAppConfigTable); err != nil {
		return fmt.Errorf("创建应用配置表失败: %w", err)
	}

	if _, err := DB.Exec(createIndexes); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	// 迁移已有数据库表结构（如果字段不存在则添加）
	if err := migrateTables(); err != nil {
		return fmt.Errorf("迁移数据库表失败: %w", err)
	}

	return nil
}

// migrateTables 迁移数据库表，添加新字段（如果不存在）
func migrateTables() error {
	// 检查并添加新字段
	migrations := []struct {
		column string
		colType string
	}{
		{"node_protocol_type", "TEXT DEFAULT 'socks5'"},
		{"vmess_version", "TEXT DEFAULT ''"},
		{"vmess_uuid", "TEXT DEFAULT ''"},
		{"vmess_alter_id", "INTEGER DEFAULT 0"},
		{"vmess_security", "TEXT DEFAULT ''"},
		{"vmess_network", "TEXT DEFAULT ''"},
		{"vmess_type", "TEXT DEFAULT ''"},
		{"vmess_host", "TEXT DEFAULT ''"},
		{"vmess_path", "TEXT DEFAULT ''"},
		{"vmess_tls", "TEXT DEFAULT ''"},
		{"ss_method", "TEXT DEFAULT ''"},
		{"ss_plugin", "TEXT DEFAULT ''"},
		{"ss_plugin_opts", "TEXT DEFAULT ''"},
		{"ssr_obfs", "TEXT DEFAULT ''"},
		{"ssr_obfs_param", "TEXT DEFAULT ''"},
		{"ssr_protocol", "TEXT DEFAULT ''"},
		{"ssr_protocol_param", "TEXT DEFAULT ''"},
		{"raw_config", "TEXT DEFAULT ''"},
	}

	// 获取表结构信息
	rows, err := DB.Query("PRAGMA table_info(servers)")
	if err != nil {
		// 表可能不存在，返回 nil（表会在 createTables 中创建）
		return nil
	}
	defer rows.Close()

	existingColumns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notnull int
		var dfltValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &colType, &notnull, &dfltValue, &pk); err != nil {
			continue
		}
		existingColumns[name] = true
	}

	// 添加缺失的字段
	for _, m := range migrations {
		if !existingColumns[m.column] {
			// 字段不存在，添加字段
			_, err := DB.Exec(fmt.Sprintf(
				"ALTER TABLE servers ADD COLUMN %s %s",
				m.column, m.colType,
			))
			if err != nil {
				// 如果添加失败，记录错误但继续
				continue
			}

			// 如果是 node_protocol_type，为已有数据设置默认值
			if m.column == "node_protocol_type" {
				_, _ = DB.Exec("UPDATE servers SET node_protocol_type = 'socks5' WHERE node_protocol_type IS NULL OR node_protocol_type = ''")
			}
		}
	}

	return nil
}

// CloseDB 关闭数据库连接。
// 应该在应用退出时调用此方法以正确释放资源。
// 返回：错误（如果有）
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// Subscription 表示一个订阅配置，包含 URL 和标签信息。
type Subscription struct {
	ID        int64     `json:"id"`
	URL       string    `json:"url"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AddOrUpdateSubscription 添加新订阅或更新现有订阅。
// 如果订阅 URL 已存在，则更新其标签；否则创建新订阅。
// 参数：
//   - url: 订阅 URL
//   - label: 订阅标签
//
// 返回：订阅实例和错误（如果有）
func AddOrUpdateSubscription(url, label string) (*Subscription, error) {
	now := time.Now()

	// 先尝试查询是否存在
	var sub Subscription
	err := DB.QueryRow("SELECT id, url, label, created_at, updated_at FROM subscriptions WHERE url = ?", url).
		Scan(&sub.ID, &sub.URL, &sub.Label, &sub.CreatedAt, &sub.UpdatedAt)

	if err == sql.ErrNoRows {
		// 不存在，插入新记录
		result, err := DB.Exec(
			"INSERT INTO subscriptions (url, label, created_at, updated_at) VALUES (?, ?, ?, ?)",
			url, label, now, now,
		)
		if err != nil {
			return nil, fmt.Errorf("插入订阅失败: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("获取插入ID失败: %w", err)
		}

		sub.ID = id
		sub.URL = url
		sub.Label = label
		sub.CreatedAt = now
		sub.UpdatedAt = now
	} else if err != nil {
		return nil, fmt.Errorf("查询订阅失败: %w", err)
	} else {
		// 存在，更新记录
		if label != sub.Label {
			_, err = DB.Exec(
				"UPDATE subscriptions SET label = ?, updated_at = ? WHERE id = ?",
				label, now, sub.ID,
			)
			if err != nil {
				return nil, fmt.Errorf("更新订阅失败: %w", err)
			}
			sub.Label = label
			sub.UpdatedAt = now
		}
	}

	return &sub, nil
}

// GetSubscriptionByURL 根据 URL 查找订阅。
// 参数：
//   - url: 订阅 URL
//
// 返回：订阅实例和错误（如果未找到或发生错误）
func GetSubscriptionByURL(url string) (*Subscription, error) {
	var sub Subscription
	err := DB.QueryRow(
		"SELECT id, url, label, created_at, updated_at FROM subscriptions WHERE url = ?",
		url,
	).Scan(&sub.ID, &sub.URL, &sub.Label, &sub.CreatedAt, &sub.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询订阅失败: %w", err)
	}

	return &sub, nil
}

// GetAllSubscriptions 获取所有订阅列表。
// 返回：订阅列表和错误（如果有）
func GetAllSubscriptions() ([]*Subscription, error) {
	rows, err := DB.Query("SELECT id, url, label, created_at, updated_at FROM subscriptions ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("查询订阅列表失败: %w", err)
	}
	defer rows.Close()

	var subscriptions []*Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.URL, &sub.Label, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描订阅数据失败: %w", err)
		}
		subscriptions = append(subscriptions, &sub)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历订阅数据失败: %w", err)
	}

	return subscriptions, nil
}

// DeleteSubscription 删除订阅及其关联的所有服务器。
// 参数：
//   - subscriptionID: 订阅 ID
//
// 返回：错误（如果有）
func DeleteSubscription(subscriptionID int64) error {
	// 先删除关联的服务器
	if err := DeleteServersBySubscriptionID(subscriptionID); err != nil {
		return fmt.Errorf("删除订阅关联服务器失败: %w", err)
	}
	
	// 再删除订阅本身
	_, err := DB.Exec("DELETE FROM subscriptions WHERE id = ?", subscriptionID)
	if err != nil {
		return fmt.Errorf("删除订阅失败: %w", err)
	}
	return nil
}

// GetSubscriptionByID 根据 ID 获取订阅。
// 参数：
//   - id: 订阅 ID
//
// 返回：订阅实例和错误（如果未找到或发生错误）
func GetSubscriptionByID(id int64) (*Subscription, error) {
	var sub Subscription
	err := DB.QueryRow(
		"SELECT id, url, label, created_at, updated_at FROM subscriptions WHERE id = ?",
		id,
	).Scan(&sub.ID, &sub.URL, &sub.Label, &sub.CreatedAt, &sub.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询订阅失败: %w", err)
	}

	return &sub, nil
}

// GetServerCountBySubscriptionID 获取指定订阅的服务器数量。
// 参数：
//   - subscriptionID: 订阅 ID
//
// 返回：服务器数量和错误（如果有）
func GetServerCountBySubscriptionID(subscriptionID int64) (int, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM servers WHERE subscription_id = ?", subscriptionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("查询服务器数量失败: %w", err)
	}
	return count, nil
}

// AddOrUpdateServer 添加新服务器或更新现有服务器。
// 如果服务器 ID 已存在，则更新其信息；否则创建新服务器。
// 如果 subscriptionID 为 nil 且服务器已存在，则保持原有的 subscription_id。
// 参数：
//   - server: 服务器配置信息
//   - subscriptionID: 关联的订阅 ID（可选，可为 nil）
//
// 返回：错误（如果有）
func AddOrUpdateServer(server config.Server, subscriptionID *int64) error {
	now := time.Now()

	// 检查服务器是否存在
	var existingID string
	var existingSubscriptionID sql.NullInt64
	err := DB.QueryRow("SELECT id, subscription_id FROM servers WHERE id = ?", server.ID).
		Scan(&existingID, &existingSubscriptionID)

	if err == sql.ErrNoRows {
		// 不存在，插入新记录
		_, err = DB.Exec(
			`INSERT INTO servers (id, subscription_id, name, addr, port, username, password, delay, selected, enabled,
				node_protocol_type, vmess_version, vmess_uuid, vmess_alter_id, vmess_security, vmess_network,
				vmess_type, vmess_host, vmess_path, vmess_tls, ss_method, ss_plugin, ss_plugin_opts,
				ssr_obfs, ssr_obfs_param, ssr_protocol, ssr_protocol_param, raw_config, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			server.ID, subscriptionID, server.Name, server.Addr, server.Port,
			server.Username, server.Password, server.Delay,
			boolToInt(server.Selected), boolToInt(server.Enabled),
			server.ProtocolType, server.VMessVersion, server.VMessUUID, server.VMessAlterID,
			server.VMessSecurity, server.VMessNetwork, server.VMessType, server.VMessHost,
			server.VMessPath, server.VMessTLS, server.SSMethod, server.SSPlugin, server.SSPluginOpts,
			server.SSRObfs, server.SSRObfsParam, server.SSRProtocol, server.SSRProtocolParam,
			server.RawConfig, now, now,
		)
		if err != nil {
			return fmt.Errorf("插入服务器失败: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("查询服务器失败: %w", err)
	} else {
		// 存在，更新记录
		// 如果 subscriptionID 为 nil，保持原有的 subscription_id
		updateSubscriptionID := subscriptionID
		if updateSubscriptionID == nil && existingSubscriptionID.Valid {
			updateSubscriptionID = &existingSubscriptionID.Int64
		}

		_, err = DB.Exec(
			`UPDATE servers SET 
				subscription_id = ?, name = ?, addr = ?, port = ?, username = ?, password = ?,
				delay = ?, selected = ?, enabled = ?,
				node_protocol_type = ?, vmess_version = ?, vmess_uuid = ?, vmess_alter_id = ?, vmess_security = ?,
				vmess_network = ?, vmess_type = ?, vmess_host = ?, vmess_path = ?, vmess_tls = ?,
				ss_method = ?, ss_plugin = ?, ss_plugin_opts = ?,
				ssr_obfs = ?, ssr_obfs_param = ?, ssr_protocol = ?, ssr_protocol_param = ?,
				raw_config = ?, updated_at = ?
			 WHERE id = ?`,
			updateSubscriptionID, server.Name, server.Addr, server.Port,
			server.Username, server.Password, server.Delay,
			boolToInt(server.Selected), boolToInt(server.Enabled),
			server.ProtocolType, server.VMessVersion, server.VMessUUID, server.VMessAlterID,
			server.VMessSecurity, server.VMessNetwork, server.VMessType, server.VMessHost,
			server.VMessPath, server.VMessTLS, server.SSMethod, server.SSPlugin, server.SSPluginOpts,
			server.SSRObfs, server.SSRObfsParam, server.SSRProtocol, server.SSRProtocolParam,
			server.RawConfig, now, server.ID,
		)
		if err != nil {
			return fmt.Errorf("更新服务器失败: %w", err)
		}
	}

	return nil
}

// GetServer 根据 ID 获取服务器信息。
// 参数：
//   - id: 服务器 ID
//
// 返回：服务器实例和错误（如果未找到或发生错误）
func GetServer(id string) (*config.Server, error) {
	var server config.Server
	var selected, enabled int

	err := DB.QueryRow(
		`SELECT id, name, addr, port, username, password, delay, selected, enabled,
			node_protocol_type, vmess_version, vmess_uuid, vmess_alter_id, vmess_security, vmess_network,
			vmess_type, vmess_host, vmess_path, vmess_tls, ss_method, ss_plugin, ss_plugin_opts,
			ssr_obfs, ssr_obfs_param, ssr_protocol, ssr_protocol_param, raw_config
		 FROM servers WHERE id = ?`,
		id,
	).Scan(&server.ID, &server.Name, &server.Addr, &server.Port,
		&server.Username, &server.Password, &server.Delay,
		&selected, &enabled,
		&server.ProtocolType, &server.VMessVersion, &server.VMessUUID, &server.VMessAlterID,
		&server.VMessSecurity, &server.VMessNetwork, &server.VMessType, &server.VMessHost,
		&server.VMessPath, &server.VMessTLS, &server.SSMethod, &server.SSPlugin, &server.SSPluginOpts,
		&server.SSRObfs, &server.SSRObfsParam, &server.SSRProtocol, &server.SSRProtocolParam,
		&server.RawConfig)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("服务器不存在: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询服务器失败: %w", err)
	}

	server.Selected = intToBool(selected)
	server.Enabled = intToBool(enabled)
	
	// 如果 ProtocolType 为空，设置默认值
	if server.ProtocolType == "" {
		server.ProtocolType = "socks5"
	}

	return &server, nil
}

// GetAllServers 获取所有服务器列表。
// 返回：服务器列表和错误（如果有）
func GetAllServers() ([]config.Server, error) {
	rows, err := DB.Query(
		`SELECT id, name, addr, port, username, password, delay, selected, enabled,
			node_protocol_type, vmess_version, vmess_uuid, vmess_alter_id, vmess_security, vmess_network,
			vmess_type, vmess_host, vmess_path, vmess_tls, ss_method, ss_plugin, ss_plugin_opts,
			ssr_obfs, ssr_obfs_param, ssr_protocol, ssr_protocol_param, raw_config
		 FROM servers ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("查询服务器列表失败: %w", err)
	}
	defer rows.Close()

	var servers []config.Server
	for rows.Next() {
		var server config.Server
		var selected, enabled int

		if err := rows.Scan(&server.ID, &server.Name, &server.Addr, &server.Port,
			&server.Username, &server.Password, &server.Delay,
			&selected, &enabled,
			&server.ProtocolType, &server.VMessVersion, &server.VMessUUID, &server.VMessAlterID,
			&server.VMessSecurity, &server.VMessNetwork, &server.VMessType, &server.VMessHost,
			&server.VMessPath, &server.VMessTLS, &server.SSMethod, &server.SSPlugin, &server.SSPluginOpts,
			&server.SSRObfs, &server.SSRObfsParam, &server.SSRProtocol, &server.SSRProtocolParam,
			&server.RawConfig); err != nil {
			return nil, fmt.Errorf("扫描服务器数据失败: %w", err)
		}

		server.Selected = intToBool(selected)
		server.Enabled = intToBool(enabled)
		
		// 如果 ProtocolType 为空，设置默认值
		if server.ProtocolType == "" {
			server.ProtocolType = "socks5"
		}
		
		servers = append(servers, server)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历服务器数据失败: %w", err)
	}

	return servers, nil
}

// GetServersBySubscriptionID 获取指定订阅关联的所有服务器。
// 参数：
//   - subscriptionID: 订阅 ID
//
// 返回：服务器列表和错误（如果有）
func GetServersBySubscriptionID(subscriptionID int64) ([]config.Server, error) {
	rows, err := DB.Query(
		`SELECT id, name, addr, port, username, password, delay, selected, enabled,
			node_protocol_type, vmess_version, vmess_uuid, vmess_alter_id, vmess_security, vmess_network,
			vmess_type, vmess_host, vmess_path, vmess_tls, ss_method, ss_plugin, ss_plugin_opts,
			ssr_obfs, ssr_obfs_param, ssr_protocol, ssr_protocol_param, raw_config
		 FROM servers WHERE subscription_id = ? ORDER BY created_at DESC`,
		subscriptionID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询服务器列表失败: %w", err)
	}
	defer rows.Close()

	var servers []config.Server
	for rows.Next() {
		var server config.Server
		var selected, enabled int

		if err := rows.Scan(&server.ID, &server.Name, &server.Addr, &server.Port,
			&server.Username, &server.Password, &server.Delay,
			&selected, &enabled,
			&server.ProtocolType, &server.VMessVersion, &server.VMessUUID, &server.VMessAlterID,
			&server.VMessSecurity, &server.VMessNetwork, &server.VMessType, &server.VMessHost,
			&server.VMessPath, &server.VMessTLS, &server.SSMethod, &server.SSPlugin, &server.SSPluginOpts,
			&server.SSRObfs, &server.SSRObfsParam, &server.SSRProtocol, &server.SSRProtocolParam,
			&server.RawConfig); err != nil {
			return nil, fmt.Errorf("扫描服务器数据失败: %w", err)
		}

		server.Selected = intToBool(selected)
		server.Enabled = intToBool(enabled)
		
		// 如果 ProtocolType 为空，设置默认值
		if server.ProtocolType == "" {
			server.ProtocolType = "socks5"
		}
		
		servers = append(servers, server)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历服务器数据失败: %w", err)
	}

	return servers, nil
}

// UpdateServerDelay 更新服务器的延迟值。
// 参数：
//   - id: 服务器 ID
//   - delay: 新的延迟值（毫秒）
//
// 返回：错误（如果有）
func UpdateServerDelay(id string, delay int) error {
	_, err := DB.Exec(
		"UPDATE servers SET delay = ?, updated_at = ? WHERE id = ?",
		delay, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("更新服务器延迟失败: %w", err)
	}
	return nil
}

// DeleteServer 删除指定的服务器。
// 参数：
//   - id: 要删除的服务器 ID
//
// 返回：错误（如果有）
func DeleteServer(id string) error {
	_, err := DB.Exec("DELETE FROM servers WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除服务器失败: %w", err)
	}
	return nil
}

// DeleteServersBySubscriptionID 删除指定订阅关联的所有服务器。
// 参数：
//   - subscriptionID: 订阅 ID
//
// 返回：错误（如果有）
func DeleteServersBySubscriptionID(subscriptionID int64) error {
	_, err := DB.Exec("DELETE FROM servers WHERE subscription_id = ?", subscriptionID)
	if err != nil {
		return fmt.Errorf("删除订阅服务器失败: %w", err)
	}
	return nil
}

// SetLayoutConfig 保存布局配置到数据库。
// 参数：
//   - key: 配置键名
//   - value: 配置值（JSON 格式字符串）
//
// 返回：错误（如果有）
func SetLayoutConfig(key, value string) error {
	now := time.Now()
	_, err := DB.Exec(
		`INSERT INTO layout_config (key, value, created_at, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = ?`,
		key, value, now, now, value, now,
	)
	if err != nil {
		return fmt.Errorf("设置布局配置失败: %w", err)
	}
	return nil
}

// GetLayoutConfig 从数据库获取布局配置。
// 参数：
//   - key: 配置键名
//
// 返回：配置值（JSON 格式字符串）和错误（如果未找到或发生错误）
func GetLayoutConfig(key string) (string, error) {
	var value string
	err := DB.QueryRow("SELECT value FROM layout_config WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("获取布局配置失败: %w", err)
	}
	return value, nil
}

// SetAppConfig 保存应用配置到数据库的 app_config 表。
// 参数：
//   - key: 配置键名（如 "logLevel", "logFile", "autoProxyEnabled", "autoProxyPort", "theme"）
//   - value: 配置值（字符串格式）
//
// 返回：错误（如果有）
func SetAppConfig(key, value string) error {
	now := time.Now()
	_, err := DB.Exec(
		`INSERT INTO app_config (key, value, created_at, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = ?`,
		key, value, now, now, value, now,
	)
	if err != nil {
		return fmt.Errorf("设置应用配置失败: %w", err)
	}
	return nil
}

// GetAppConfig 从数据库的 app_config 表获取应用配置。
// 参数：
//   - key: 配置键名
//
// 返回：配置值和错误（如果未找到或发生错误）
func GetAppConfig(key string) (string, error) {
	var value string
	err := DB.QueryRow("SELECT value FROM app_config WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("获取应用配置失败: %w", err)
	}
	return value, nil
}

// GetAppConfigWithDefault 获取应用配置，如果不存在则返回默认值。
// 参数：
//   - key: 配置键名
//   - defaultValue: 默认值（当配置不存在时返回）
//
// 返回：配置值或默认值和错误（如果有）
func GetAppConfigWithDefault(key, defaultValue string) (string, error) {
	value, err := GetAppConfig(key)
	if err != nil {
		return "", err
	}
	if value == "" {
		// 如果不存在，写入默认值
		if err := SetAppConfig(key, defaultValue); err != nil {
			return "", err
		}
		return defaultValue, nil
	}
	return value, nil
}

// boolToInt 将布尔值转换为整数
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// intToBool 将整数转换为布尔值
func intToBool(i int) bool {
	return i != 0
}
