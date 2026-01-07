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
│  - SubscriptionService               │
│  - ConfigService                     │
│  - ProxyService                      │
├─────────────────────────────────────┤
│      Store Layer (store/)           │  ← 数据访问和绑定管理
│  - NodesStore                       │
│  - SubscriptionsStore               │
│  - ConfigStore                      │
├─────────────────────────────────────┤
│   Database Layer (database/)        │  ← 数据库访问
├─────────────────────────────────────┤
│      Model Layer (model/)           │  ← 数据模型
└─────────────────────────────────────┘
```

### 依赖规则

**严格遵循以下依赖规则**：

1. **UI 层 (ui/)**：
   - ✅ 可以依赖：Service 层、Store 层、Model 层
   - ❌ 禁止直接依赖：Database 层
   - 职责：只负责 UI 展示和事件转发，不包含业务逻辑

2. **Service 层 (service/)**：
   - ✅ 可以依赖：Store 层、Model 层
   - ❌ 禁止依赖：UI 层、Database 层（通过 Store 访问）
   - 职责：包含业务逻辑，协调 Store 层完成业务操作

3. **Store 层 (store/)**：
   - ✅ 可以依赖：Database 层、Model 层
   - ❌ 禁止依赖：UI 层、Service 层
   - 职责：数据访问和双向绑定管理，不包含业务逻辑

4. **Database 层 (database/)**：
   - ✅ 可以依赖：Model 层
   - ❌ 禁止依赖：其他任何层
   - 职责：纯粹的数据库 CRUD 操作

5. **Model 层 (model/)**：
   - ✅ 不依赖任何层（纯数据结构）
   - 职责：定义数据模型，供各层使用

6. **工具层 (utils/, subscription/, xray/, systemproxy/)**：
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
   - `ProxyService` 通过 `AppState.XrayInstance` 访问 xray 实例（可能为 nil）
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
- ❌ 禁止直接访问 `database` 包，必须通过 `Store` 或 `Service` 层

架构分层:
- 严格遵循分层架构和依赖规则
- 工具类（如 `ping`）应该无状态，不依赖其他 Service
- 数据通过参数传入，而不是在内部获取
- 数据更新返回结果，由调用者决定如何更新

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
