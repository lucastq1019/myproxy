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
  database/              # SQLite封装（数据库访问层）
  error/                 # 结构化错误系统
  logging/               # 日志管理
  model/                 # 数据模型层
  service/               # 业务逻辑层（Service层）
  store/                 # 数据访问和绑定管理（Store层）
  subscription/          # 订阅解析
  systemproxy/           # 系统代理（跨平台）
  ui/                    # Fyne界面组件（UI层）
  utils/                 # 工具函数（延迟测试等）
  xray/                  # xray-core封装
data/                    # 数据库目录（运行时生成）
config.json              # 运行时配置
```

## 架构设计

### 分层架构

项目采用分层架构，各层职责明确，依赖关系清晰：

```
┌─────────────────────────────────────┐
│         UI Layer (ui/)              │  ← 只负责 UI 展示和事件转发
├─────────────────────────────────────┤
│      Service Layer (service/)       │  ← 业务逻辑层
│  - ServerService                    │
│  - SubscriptionService              │
│  - ConfigService                    │
│  - ProxyService                     │
│  - XrayControlService               │
├─────────────────────────────────────┤
│      Store Layer (store/)           │  ← 数据访问和绑定管理
│  - NodesStore                       │
│  - SubscriptionsStore               │
│  - ConfigStore                      │
│  - ProxyStatusStore                 │
│  - AppConfigStore                   │
│  - LayoutStore                      │
├─────────────────────────────────────┤
│   Database Layer (database/)        │  ← 数据库访问
├─────────────────────────────────────┤
│      Model Layer (model/)           │  ← 数据模型
├─────────────────────────────────────┤
│      Error Layer (error/)           │  ← 结构化错误系统
└─────────────────────────────────────┘
```

### 依赖规则

**严格遵循以下依赖规则**：

1. **UI 层 (ui/)**：
   - ✅ 可以依赖：Service 层、Store 层、Model 层、Error 层
   - ❌ 禁止直接依赖：Database 层
   - 职责：只负责 UI 展示和事件转发，不包含业务逻辑

2. **Service 层 (service/)**：
   - ✅ 可以依赖：Store 层、Model 层、Error 层
   - ❌ 禁止依赖：UI 层、Database 层（通过 Store 访问）
   - 职责：包含业务逻辑，协调 Store 层完成业务操作

3. **Store 层 (store/)**：
   - ✅ 可以依赖：Database 层、Model 层、Error 层
   - ❌ 禁止依赖：UI 层、Service 层
   - 职责：数据访问和双向绑定管理，不包含业务逻辑

4. **Database 层 (database/)**：
   - ✅ 可以依赖：Model 层
   - ❌ 禁止依赖：其他任何层
   - 职责：纯粹的数据库 CRUD 操作

5. **Model 层 (model/)**：
   - ✅ 不依赖任何层（纯数据结构）
   - 职责：定义数据模型，供各层使用

6. **Error 层 (error/)**：
   - ✅ 不依赖任何层（纯错误定义）
   - 职责：定义错误代码和错误包装工具

7. **工具层 (utils/, subscription/, xray/, systemproxy/)**：
   - ✅ 可以依赖：Model 层
   - ❌ 禁止依赖：UI 层、Service 层、Store 层、Database 层
   - 职责：提供独立的功能模块，不涉及数据更新
   - 示例：`utils.Ping` 用于延迟测试，`xray.XrayInstance` 用于代理服务
   - **xray 包说明**：
     - xray 是工具层，与 `utils.Ping` 类似：通过参数传入数据，不持有业务数据
     - xray 实例生命周期 = 代理运行生命周期（启动代理时创建，停止代理时销毁）
     - 切换节点时：停止旧实例 → 销毁 → 创建新实例 → 启动（xray-core 限制）
     - 实例对象由 App 层临时持有（用于状态检查和 Service 访问），但生命周期由代理行为决定

### Xray 实例管理规则

1. **xray 定位**：
   - xray 属于工具层 (`internal/xray/`)，与 `utils.Ping` 类似
   - 通过参数传入配置，不持有业务数据
   - 实例生命周期 = 代理运行生命周期（随代理行为创建/销毁）

2. **xray 实例持有**：
   - xray 实例对象由 **App 层临时持有**（`AppState.XrayInstance`）
   - 生命周期：由代理行为决定，不是 App 生命周期
   - App 启动时：字段初始化为 nil
   - 启动代理时：创建实例并保存到 AppState
   - 停止代理时：销毁实例（设为 nil）
   - 切换节点时：销毁旧实例，创建新实例
   - ❌ 禁止由 Store 层持有（Store 只管理数据，不管理业务服务）

3. **xray 实例创建时机**：
   - **启动代理时**：根据选中节点创建配置 → 创建 xray 实例 → 启动 → 保存到 AppState
   - **切换节点时**：停止当前实例 → 销毁实例 → 根据新节点创建配置 → 创建新实例 → 启动
     - 注意：由于 xray-core 限制，配置变化必须重新创建实例
   - **停止代理时**：停止实例运行 → 销毁实例（设为 nil）
   - **App 退出时**：如果实例存在，停止并销毁

4. **xray 与 Service 层关系**：
   - `ProxyService` 和 `XrayControlService` 通过 `AppState.XrayInstance` 访问 xray 实例（可能为 nil）
   - Service 层提供业务方法（如 `StartProxy(node)`），内部操作 App 临时持有的 xray 实例

### 数据访问规则

1. **UI 层数据访问**：
   - 通过 `AppState.Store` 访问数据
   - 通过 `AppState.Service` 执行业务操作
   - ❌ 禁止直接调用 `database` 包

2. **Service 层数据访问**：
   - 通过 `Store` 层访问数据
   - ❌ 禁止直接调用 `database` 包

3. **数据更新流程**：
   ```
   UI 层 → Service 层 → Store 层 → Database 层
   ```

4. **数据模型使用**：
   - 各层统一使用 `model.Node`、`model.Subscription` 等
   - ❌ 禁止直接使用 `database.Node`（虽然它是别名，但应使用 model 包）

### 重构原则

1. **职责单一原则**：
   - 每个包/层只负责自己的职责
   - 例如：`utils.Ping` 只负责延迟测试，不负责数据更新

2. **依赖倒置原则**：
   - 高层模块不依赖低层模块，都依赖抽象（Model 层）
   - 通过参数传递数据，而不是在内部获取

3. **数据流向清晰**：
   - 数据获取：通过参数传入，而不是在内部调用其他层获取
   - 数据更新：返回结果，由调用者决定如何更新

4. **示例：ping 工具重构**：
   ```go
   // ❌ 错误：依赖 Service 层，内部获取数据
   func (p *Ping) TestAllServersDelay() map[string]int {
       servers := p.serverService.ListServers()  // 错误：内部获取数据
       // ...
   }
   
   // ✅ 正确：通过参数传入，只负责测试
   func (p *Ping) TestAllServersDelay(servers []model.Node) map[string]int {
       // 只负责测试延迟，不涉及数据更新
   }
   ```

5. **构造函数设计**：
   - 避免在构造函数中传入其他 Service 的依赖
   - 工具类（如 `utils.Ping`）应该无状态，不需要依赖注入
   - 如果确实需要依赖，应该通过方法参数传递，而不是结构体字段

### 架构改进要点

1. **初始化状态检查**：
   - 所有核心组件（Store、AppState、Service）都应添加初始化状态标记
   - 防止重复初始化导致的资源泄露和冲突
   - 提供 `IsInitialized()` 和 `Reset()` 方法

2. **统一错误处理**：
   - 使用 `error` 包实现结构化错误系统
   - 定义错误代码常量，方便错误分类和处理
   - 使用 `error.Wrap()` 包装错误，保留错误链
   - 错误消息使用中文，便于用户理解

3. **接口隔离**：
   - 定义清晰的接口，提高代码可测试性和可维护性
   - 为核心组件（ServerService、SubscriptionService、ConfigService 等）定义接口
   - 接口应简洁明了，只包含必要的方法

4. **并发安全**：
   - Store 层应添加读写锁，确保并发安全
   - 保护节点和订阅数据的并发访问
   - 使用 `sync.RWMutex` 优化读多写少的场景

5. **日志系统增强**：
   - 添加 `SafeLogger`，处理未初始化的日志记录器
   - 提供安全的日志记录接口，防止 nil 指针异常
   - 支持日志回调，实现日志实时更新到 UI

6. **资源管理**：
   - 改进 `Cleanup` 方法，确保资源正确释放
   - 清理 Xray 实例、Logger、Store 和服务层资源
   - 使用 `defer` 确保资源释放

7. **应用工厂模式**：
   - 使用 `ApplicationFactory` 集中管理应用组件的创建和初始化
   - 确保依赖关系正确，初始化顺序合理
   - 提供统一的应用初始化入口

8. **页面导航**：
   - AppState 应持有 MainWindow 实例，方便页面导航
   - 实现页面栈管理，支持返回操作
   - 页面组件应通过 MainWindow 访问其他页面

### 接口定义

#### Service 层接口

```go
// ServerServiceInterface 定义服务器服务接口
type ServerServiceInterface interface {
    GetAllServers() ([]*model.Node, error)
    GetServerByID(id string) (*model.Node, error)
    GetServersBySubscriptionID(subscriptionID int64) ([]model.Node, error)
    UpdateServerDelay(id string, delay int) error
    DeleteServer(id string) error
    AddOrUpdateServer(node model.Node, subscriptionID *int64) error
    ListServers() []model.Node
    GetSelectedSubscriptionID() int64
    SetSelectedSubscriptionID(subscriptionID int64)
}

// SubscriptionServiceInterface 定义订阅服务接口
type SubscriptionServiceInterface interface {
    GetAllSubscriptions() ([]*database.Subscription, error)
    GetSubscriptionByID(id int64) (*database.Subscription, error)
    GetSubscriptionByURL(url string) (*database.Subscription, error)
    AddSubscription(url, label string) (*database.Subscription, error)
    UpdateSubscription(id int64, url, label string) error
    DeleteSubscription(id int64) error
    UpdateSubscriptionByID(id int64) error
    FetchSubscription(url string, label ...string) error
}

// ConfigServiceInterface 定义配置服务接口
type ConfigServiceInterface interface {
    GetConfig() (*config.Config, error)
    SaveConfig(config *config.Config) error
    LoadConfig() (*config.Config, error)
    GetDefaultConfig() *config.Config
}

// ProxyServiceInterface 定义代理服务接口
type ProxyServiceInterface interface {
    StartProxy(node *model.Node) error
    StopProxy() error
    RestartProxy(node *model.Node) error
    UpdateXrayInstance(instance *xray.XrayInstance)
}

// XrayControlServiceInterface 定义 Xray 控制服务接口
type XrayControlServiceInterface interface {
    StartProxy(instance *xray.XrayInstance, logPath string) *XrayControlResult
    StopProxy(instance *xray.XrayInstance) error
    RestartProxy(instance *xray.XrayInstance, logPath string) *XrayControlResult
}
```

#### Store 层接口

```go
// NodesStoreInterface 定义节点存储接口
type NodesStoreInterface interface {
    Load() error
    GetAll() []*model.Node
    Get(id string) (*model.Node, error)
    Select(id string) error
    UpdateDelay(id string, delay int) error
    Delete(id string) error
    Add(node *model.Node) error
    Update(node *model.Node) error
    GetBySubscriptionID(subscriptionID int64) ([]*model.Node, error)
    GetSelected() *model.Node
    IsInitialized() bool
    Reset()
}

// SubscriptionsStoreInterface 定义订阅存储接口
type SubscriptionsStoreInterface interface {
    Load() error
    GetAll() []*database.Subscription
    Get(id int64) (*database.Subscription, error)
    GetByURL(url string) (*database.Subscription, error)
    Add(url, label string) (*database.Subscription, error)
    Update(id int64, url, label string) error
    Delete(id int64) error
    UpdateByID(id int64) error
    Fetch(url string, label ...string) error
    IsInitialized() bool
    Reset()
}

// AppConfigStoreInterface 定义应用配置存储接口
type AppConfigStoreInterface interface {
    Get(key string) (string, error)
    Set(key, value string) error
    GetWithDefault(key, defaultValue string) (string, error)
    SaveWindowSize(size fyne.Size)
    GetWindowSize(defaultSize fyne.Size) fyne.Size
}

// LayoutStoreInterface 定义布局配置存储接口
type LayoutStoreInterface interface {
    Get() *LayoutConfig
    Save(config *LayoutConfig) error
    Load() error
}

// ProxyStatusStoreInterface 定义代理状态存储接口
type ProxyStatusStoreInterface interface {
    UpdateProxyStatus(xrayInstance *xray.XrayInstance, nodesStore *NodesStore)
    GetProxyStatus() string
    GetPort() string
    GetServerName() string
}
```

### 错误代码定义

```go
// 错误代码常量
const (
    ErrCodeNone             = "NONE"             // 无错误
    ErrCodeInit             = "INIT"             // 初始化错误
    ErrCodeDatabase         = "DATABASE"         // 数据库错误
    ErrCodeInternal         = "INTERNAL"         // 内部错误
    ErrCodeInvalidInput     = "INVALID_INPUT"    // 无效输入
    ErrCodeNotFound         = "NOT_FOUND"        // 资源不存在
    ErrCodeSubscription     = "SUBSCRIPTION"     // 订阅错误
    ErrCodeSubscriptionFetch = "SUBSCRIPTION_FETCH" // 订阅获取错误
    ErrCodeProxy            = "PROXY"            // 代理错误
    ErrCodeSystemProxy      = "SYSTEM_PROXY"     // 系统代理错误
    ErrCodeXray             = "XRAY"             // Xray 错误
    ErrCodeAlreadyInit      = "ALREADY_INIT"     // 已经初始化
)
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
- 使用 `error.Wrap(err, error.ErrCodeInternal, "描述")`
- 使用 `error.New(error.ErrCodeNotFound, "描述")`
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
- 优先使用 `SafeLogger`，防止 nil 指针异常

配置:
- 优先从数据库读取
- 支持JSON文件迁移到数据库

UI:
- Fyne框架
- 组件在 `internal/ui` 包
- UI逻辑与业务逻辑分离
- ❌ 禁止直接访问 `database` 包，必须通过 `Store` 或 `Service` 层

架构分层:
- 严格遵循分层架构和依赖规则
- 工具类（如 `ping`）应该无状态，不依赖其他 Service
- 数据通过参数传入，而不是在内部获取
- 数据更新返回结果，由调用者决定如何更新

并发:
- UI操作必须在主goroutine
- 使用通道或回调在goroutine间通信
- Store 层应添加读写锁，确保并发安全

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