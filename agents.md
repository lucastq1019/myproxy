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

```
UI Layer (ui/)          → Service Layer (service/) → Store Layer (store/) → Database Layer (database/)
                              ↓                           ↓
                         Model Layer (model/)      Error Layer (error/)
```

### 依赖规则

1. **UI 层**: 可依赖 Service、Store、Model、Error 层；禁止直接访问 Database 层
2. **Service 层**: 可依赖 Store、Model、Error 层；禁止依赖 UI、Database 层
3. **Store 层**: 可依赖 Database、Model、Error 层；禁止依赖 UI、Service 层
4. **Database 层**: 仅可依赖 Model 层
5. **Model/Error 层**: 不依赖任何层
6. **工具层** (utils/, xray/, systemproxy/): 仅可依赖 Model 层；通过参数传入数据，不持有业务数据

### Xray 实例管理

- xray 属于工具层，实例生命周期 = 代理运行生命周期
- 实例由 AppState 临时持有，禁止由 Store 层持有
- 启动/切换节点时：创建新实例；停止时：销毁实例（设为 nil）
- Service 层通过 `AppState.XrayInstance` 访问实例

### 数据访问规则

- UI 层：通过 `AppState.Store` 和 `AppState.Service` 访问
- Service 层：通过 Store 层访问
- 数据更新流程：UI → Service → Store → Database
- 统一使用 `model` 包的数据模型

### 重构原则

- 职责单一：每个包/层只负责自己的职责
- 依赖倒置：通过参数传递数据，不在内部获取
- 工具类无状态：如 `utils.Ping`，通过方法参数传递依赖

示例：
```go
// 错误：内部获取数据
func (p *Ping) TestAllServersDelay() map[string]int {
    servers := p.serverService.ListServers()
}

// 正确：通过参数传入
func (p *Ping) TestAllServersDelay(servers []model.Node) map[string]int {
}
```

## UI布局规范

**核心原则：禁止使用固定间距值，必须使用 Fyne 系统提供的间距**

- 使用 `theme.SizeNameInnerPadding` 获取最小间距
- 使用 `compactVBoxLayout` 减少组件间距
- 使用 `newPaddedWithSize()` 替代 `container.NewPadded()`
- 使用 `noSpacingBorderLayout` 移除 headerBar 下方的多余空白

获取间距方法：
```go
func (sp *SettingsPage) getInnerPadding() float32 {
    if sp.appState != nil && sp.appState.App != nil {
        return sp.appState.App.Settings().Theme().Size(theme.SizeNameInnerPadding)
    }
    return theme.DefaultTheme().Size(theme.SizeNameInnerPadding)
}
```

使用示例：
```go
// 正确
spacing := sp.getInnerPadding()
content := container.NewWithoutLayout(items...)
content.Layout = compactVBoxLayout{spacing: spacing}
paddedContent := newPaddedWithSize(content, spacing)

// 错误：禁止使用固定值
content.Layout = compactVBoxLayout{spacing: 4}  // 禁止
container.NewPadded(content)  // 禁止
```

## 编码规范

### 命名

- 包名：小写简短
- 导出标识符：首字母大写（类型、函数、常量）
- 私有标识符：首字母小写
- 结构体：PascalCase，单数形式
- 函数命名：`New<Type>()`, `Get<Field>()`, `Set<Field>()`, 动词动作, `Is*/Has*/Can*` 布尔

### 代码格式

- 注释：函数描述 + 参数/返回值说明
- 错误处理：使用 `error.Wrap()` 和 `error.New()`，错误消息使用中文
- JSON标签：camelCase
- 导入顺序：标准库 → 第三方库 → 项目内部包
- 方法接收者：指针类型，使用类型缩写

### 代码规则

- 构造函数：`New<Type>()` 模式，返回指针
- 数据库：使用预编译语句，及时关闭连接
- 日志：使用 `internal/logging` 包，优先使用 `SafeLogger`
- 配置：优先从数据库读取
- UI：禁止直接访问 `database` 包，必须通过 Store 或 Service 层
- 并发：UI操作在主goroutine，Store层使用读写锁

## 启动和构建

### 启动
```bash
go run ./cmd/gui/main.go
go run ./cmd/gui/main.go /path/to/config.json
```

启动行为：初始化数据库 `./data/myproxy.db`，读取配置，归档旧日志，加载服务器和订阅

### 构建
```bash
# Windows
build.bat [windows|linux|mac|clean]
set VERSION=1.0.0 && build.bat

# Linux/macOS
./build.sh [windows|linux|mac|clean]
VERSION=1.0.0 ./build.sh
```

构建输出: `dist/<OS>-<ARCH>/proxy-gui[.exe]`  
构建目标: windows(amd64,386), linux(amd64,arm64), darwin(amd64,arm64)  
构建参数: CGO_ENABLED=1, ldflags: -s -w -X main.version=$VERSION

## 约束

- 唯一入口: cmd/gui/main.go
- 数据库路径: ./data/myproxy.db（相对于config.json目录）
- 日志自动归档
- 构建需要CGO支持
- 版本号: 环境变量VERSION或时间戳
