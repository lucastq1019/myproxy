# Agents 开发规范

## 项目信息

- 语言: Go 1.25.4+
- 模块: myproxy.com/p
- UI: Fyne v2.7.1
- 数据库: SQLite3
- 核心: xray-core v1.251208.0
- 入口: cmd/gui/main.go

## 项目结构

```
cmd/gui/                 # 唯一入口
internal/
  config/                # 配置定义
  database/              # SQLite封装
  logging/               # 日志管理
  ping/                  # 延迟测试
  subscription/          # 订阅解析
  systemproxy/           # 系统代理（跨平台）
  ui/                    # Fyne界面组件
  xray/                  # xray-core封装
data/                    # 数据库目录（运行时生成）
config.json              # 运行时配置
```

## 启动命令

```bash
# 开发
go run ./cmd/gui/main.go
go run ./cmd/gui/main.go /path/to/config.json

# 编译
go build -o gui ./cmd/gui
./gui [config_path]
```

启动行为：
- 初始化数据库: ./data/myproxy.db
- 读取配置: config.json 或命令行参数
- 归档旧日志
- 加载数据库中的服务器和订阅

## 打包命令

### Windows (build.bat)
```batch
build.bat                  # 构建所有平台
build.bat windows          # Windows
build.bat linux            # Linux
build.bat mac              # macOS
build.bat clean            # 清理
set VERSION=1.0.0 && build.bat
```

### Linux/macOS (build.sh)
```bash
./build.sh                 # 构建所有平台
./build.sh windows
./build.sh linux
./build.sh mac
./build.sh clean
VERSION=1.0.0 ./build.sh
```

构建输出: `dist/<OS>-<ARCH>/proxy-gui[.exe]`

构建目标:
- windows: amd64, 386
- linux: amd64, arm64
- darwin: amd64, arm64

构建参数:
- CGO_ENABLED=1
- ldflags: -s -w -X main.version=$VERSION
- VERSION: 默认时间戳，可通过环境变量设置

## 编码规范

### 命名

包名: 小写，简短 (`config`, `database`, `ui`)

导出标识符 (首字母大写):
- 类型: `Config`, `Server`, `MainWindow`
- 函数: `NewMainWindow()`, `LoadConfig()`, `DefaultConfig()`
- 常量: `LogLevel`, `LogType`

私有标识符 (首字母小写):
- 变量: `appState`, `layoutConfig`
- 函数: `loadLayoutConfig()`, `saveLayoutConfig()`
- 字段: 通常私有，JSON序列化时公开

结构体: PascalCase，单数形式 (`Server`, `Config`, `MainWindow`)

函数命名模式:
- 构造函数: `New<Type>()`
- Getter: `Get<Field>()`
- Setter: `Set<Field>()`
- 动作: 动词 (`LoadConfig()`, `SaveConfig()`, `InitDB()`)
- 布尔: `Is*`, `Has*`, `Can*`

### 代码格式

注释格式:
```go
// FunctionName 函数描述。
// 参数说明（如有）
// 返回值说明（如有）
func FunctionName(...) {...}

// TypeName 类型描述。
type TypeName struct {
    Field Type `json:"field"` // 字段说明
}
```

错误处理:
- 使用 `fmt.Errorf("描述: %w", err)`
- 错误消息使用中文

JSON标签: camelCase (`json:"protocol_type"`, `json:"vmess_uuid,omitempty"`)

导入顺序:
1. 标准库
2. 第三方库
3. 项目内部包 (myproxy.com/p/...)

文件结构:
1. package
2. imports
3. constants
4. variables
5. types
6. functions/methods

方法接收者: 指针类型 (`*Type`)，使用类型缩写 (`c *Config`, `mw *MainWindow`)

### 代码规则

构造函数:
- 使用 `New<Type>()` 模式
- 返回指针类型

数据库:
- 使用预编译语句
- 及时关闭连接和结果集

日志:
- 使用 `internal/logging` 包
- 级别: debug, info, warn, error, fatal
- 输出到文件和UI面板

配置:
- 优先从数据库读取
- 支持JSON文件迁移到数据库

UI:
- Fyne框架
- 组件在 `internal/ui` 包
- UI逻辑与业务逻辑分离

并发:
- UI操作必须在主goroutine
- 使用通道或回调在goroutine间通信

## 测试

```bash
go test ./...
go test ./internal/database
go test -cover ./...
```

## 约束

- 唯一入口: cmd/gui/main.go
- 数据库路径: ./data/myproxy.db（相对于config.json目录）
- 日志自动归档
- 构建需要CGO支持
- 版本号: 环境变量VERSION或时间戳
