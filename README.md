# SOCKS/Xray 桌面代理客户端

基于 Fyne 的跨平台桌面应用，整合 xray-core 作为核心代理引擎，支持订阅、服务器管理、自动代理、本地日志与布局持久化。项目以单一 GUI 入口提供完整的代理体验，其余 CLI 程序仅用于开发调试。

## 功能总览
- GUI：订阅管理、服务器列表、延迟测试、启动/停止代理、实时日志、状态栏，窗口布局自动保存。
- 代理引擎：内置 xray-core（库方式集成），默认开启本地 SOCKS5 入站，出站可选 SOCKS5/VMess（支持 TLS/WS/H2/gRPC 等常见参数）。
- 自动代理：以选中服务器生成 xray 配置并启动本地 10080 端口（可自定义），UI 实时回显端口与状态。
- 订阅与服务器：支持 VMess、SOCKS5、JSON/Base64 订阅，数据存入 SQLite；可为订阅加标签，右键/菜单管理服务器。
- 日志与主题：应用日志+代理日志集中显示，支持级别/类型过滤；主题（浅/深色）和布局比例持久化到数据库。
- 向后兼容：保留旧版 SOCKS5 转发器（`internal/proxy/forwarder`），但默认路径使用 xray-core。

## 目录结构
```
├── cmd/
│   └── gui/                 # ✅ 唯一正式入口
├── doc/
│   ├── xray-core-integration.md
│   └── xray-usage-example.go
├── internal/
│   ├── config/              # 应用配置（日志/端口）与协议字段定义
│   ├── database/            # SQLite 封装（订阅、服务器、布局、主题）
│   ├── logging/             # 日志与归档
│   ├── ping/                # 延迟测试
│   ├── proxy/               # 旧版 SOCKS5 转发器（兼容）
│   ├── server/              # 服务器管理
│   ├── socks5/              # SOCKS5 客户端实现
│   ├── subscription/        # 订阅解析与入库
│   ├── ui/                  # Fyne 界面组件与布局
│   └── xray/                # xray-core 封装与动态配置
├── data/                    # 默认数据库目录（运行时生成）
├── config.json              # 运行时配置（日志/自动代理）
└── README.md
```

## 快速开始
### 环境要求
- Go 1.25.4+
- macOS / Windows / Linux（需图形环境）

### 安装与运行
```bash
cd /Users/test/work/project/proxy
go mod download
go run ./cmd/gui/main.go           # 或 go build -o gui ./cmd/gui && ./gui

# 自定义配置文件路径
go run ./cmd/gui/main.go /path/to/config.json
```
启动时会自动：
1) 初始化 SQLite 数据库到 `./data/myproxy.db`（不存在则创建）；  
2) 读取配置（默认 `config.json`）；  
3) 归档旧日志，应用主题与布局设置。

### 使用流程（GUI）
1. 在订阅面板添加订阅 URL（支持 VMess/SOCKS5/JSON/Base64），可为订阅设置标签。
2. 等待服务器入库后，在列表中选择需要的节点并测试延迟。
3. 点击“启动代理”启动本地 SOCKS5（默认 10080，可在配置中调整）。
4. 系统/浏览器代理指向 `127.0.0.1:<本地端口>`，日志面板实时查看运行状态。

## 配置说明
应用配置主要用于日志与自动代理端口，服务器与订阅存放在数据库：
```json
{
  "autoProxyEnabled": false,
  "autoProxyPort": 10080,
  "logLevel": "info",
  "logFile": "myproxy.log",
  "selectedServerID": ""
}
```

### 服务器字段（存储在数据库）
- 通用：`id` `name` `addr` `port` `username` `password` `delay` `enabled` `selected`
- 协议标识：`protocol_type` = `socks5` | `vmess` | `ss` | `ssr`（扩展中）
- VMess 相关：`vmess_uuid` `vmess_security` `vmess_network` (`tcp/ws/h2/grpc…`) `vmess_host` `vmess_path` `vmess_tls`
- Shadowsocks/SSR 相关字段保留以便兼容（`ss_method`、`ssr_obfs` 等）
- `raw_config`：保留完整原始配置，便于未来扩展/导入

### xray 工作方式
- 通过 `internal/xray` 生成本地 SOCKS5 入站 + 选中服务器的出站配置：
  - 入站：`port = autoProxyPort`（默认 10080），UDP 支持开启
  - 出站：根据服务器协议自动生成 SOCKS5/VMess 配置，支持 TLS/WS/H2/gRPC 参数
- 若需要自定义，可参考 `doc/xray-core-integration.md` 和示例 `doc/xray-usage-example.go`。
- 旧版转发逻辑依然保留在 `internal/proxy/forwarder`，但 GUI 默认走 xray-core。

## 开发者提示
- 入口：`cmd/gui/main.go`
- 日志：`logging` 模块负责分级输出与归档；xray 日志可通过回调接入 UI。
- 布局/主题：保存在数据库的 `layout_config` 与 `app_config` 表，首次运行会写入默认值。
- 数据库：`data/myproxy.db` 自动创建；关闭应用时会关闭连接。
- 测试：`go test ./...`

## 许可证
MIT License

## 重要声明
- 本项目仅供学习与研究，请勿用于任何违反当地法律法规的用途。
- GUI 是唯一正式入口，其余命令行/测试代码仅用于开发调试。

## TODO list
- 日志区域可完全折叠 done
- 界面美化
- 系统启动时根据上次的状态自动加载
- 按规则进行代理，内置白名单，哪些不走代理
- 