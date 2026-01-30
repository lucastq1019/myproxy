# 系统代理管理模块

## 架构设计

本模块使用**策略模式**实现多平台支持，避免了大量的 `if/else` 判断。

### 设计优势

1. **可扩展性**：添加新平台只需实现 `PlatformProxy` 接口，无需修改现有代码
2. **可维护性**：每个平台的实现独立，代码清晰
3. **可测试性**：可以轻松为每个平台编写单元测试
4. **类型安全**：编译时检查，避免运行时错误

### 架构图

```
SystemProxy (统一接口)
    ↓
PlatformProxy (接口)
    ├── DarwinProxy (macOS)
    ├── LinuxProxy (Linux)
    ├── WindowsProxy (Windows)
    └── UnsupportedProxy (不支持的系统)
```

## 环境变量代理方案对比

### 方案一：直接修改 Shell 配置文件

**实现方式**：直接在 `~/.zshrc` 或 `~/.bashrc` 中追加代理环境变量

**优点**：
- ✅ 实现简单，直接写入
- ✅ 不需要额外的文件
- ✅ 配置集中在一个文件

**缺点**：
- ❌ 直接修改用户配置文件，风险较高
- ❌ 清除时需要解析文件，容易出错
- ❌ 如果用户手动修改了配置，可能冲突
- ❌ 难以区分哪些是我们添加的配置

### 方案二：外部 Shell 文件（推荐）⭐

**实现方式**：
1. 创建独立的代理配置文件 `~/.myproxy_proxy.sh`
2. 在 shell 配置文件中添加 `source ~/.myproxy_proxy.sh`

**优点**：
- ✅ **隔离性好**：代理配置独立文件，不影响用户其他配置
- ✅ **易于管理**：清除时只需删除一个文件，简单可靠
- ✅ **可读性强**：用户可以看到明确的 source 语句
- ✅ **安全性高**：不会误删用户的其他配置
- ✅ **易于调试**：可以单独查看代理配置文件

**缺点**：
- ⚠️ 需要管理两个文件（代理文件 + shell 配置文件）
- ⚠️ 需要处理 source 语句的添加/删除

### 推荐方案

**当前实现使用方案二（外部 Shell 文件）**，原因：

1. **更安全**：不会污染用户的 shell 配置文件
2. **更可靠**：清除时只需删除独立文件，不会误删其他配置
3. **更专业**：符合 Unix/Linux 的配置管理最佳实践

### 实现细节

#### 设置代理时：
1. 创建 `~/.myproxy_proxy.sh` 文件，包含所有代理环境变量
2. 在 `~/.zshrc` 或 `~/.bashrc` 中添加：
   ```bash
   # Source myproxy proxy settings
   source ~/.myproxy_proxy.sh
   ```

#### 清除代理时：
1. 删除 `~/.myproxy_proxy.sh` 文件
2. 从 shell 配置文件中移除 source 语句

## 使用示例

```go
// 创建代理管理器（自动检测平台）
proxy := systemproxy.NewSystemProxy("127.0.0.1", 10808)

// 设置系统代理
err := proxy.SetSystemProxy()

// 设置环境变量代理（使用外部文件方案）
err = proxy.SetTerminalProxy()

// 清除所有代理
err = proxy.ClearSystemProxy()
err = proxy.ClearTerminalProxy()

// 动态更新代理地址
proxy.UpdateProxy("127.0.0.1", 10808)
```

## 平台支持

- ✅ **macOS**: 完整支持（系统代理 + 环境变量代理）
- ⏳ **Linux**: 环境变量代理支持，系统代理待实现
- ✅ **Windows**: 完整支持（系统代理 + 环境变量代理）

## Windows 实现说明

### 系统代理设置

Windows 系统代理通过修改注册表实现：

**注册表路径**：
```
HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Internet Settings
```

**关键值**：
- `ProxyEnable` (DWORD): 1 启用，0 禁用
- `ProxyServer` (String): 代理服务器地址，格式 `host:port`
- `ProxyOverride` (String): 代理覆盖列表，默认 `<local>` 表示本地地址不使用代理

**实现方式**：
- 使用 `golang.org/x/sys/windows/registry` 包操作注册表
- 设置代理时：写入 `ProxyServer` 和 `ProxyEnable=1`
- 清除代理时：设置 `ProxyEnable=0`

**注意事项**：
- 修改注册表后，系统可能需要刷新才能生效
- 某些应用程序可能需要重启才能读取新的代理设置
- 可以通过发送 `WM_SETTINGCHANGE` 消息通知系统（当前实现中简化处理）

### 环境变量代理设置

Windows 环境变量代理通过用户环境变量实现持久化：

**注册表路径**：
```
HKEY_CURRENT_USER\Environment
```

**实现方式**：
1. 设置当前进程环境变量（立即生效）
2. 在注册表中设置用户环境变量（持久化）
   - `HTTP_PROXY`, `HTTPS_PROXY`, `http_proxy`, `https_proxy`
   - `ALL_PROXY`, `all_proxy`

**清除方式**：
1. 清除当前进程环境变量
2. 从注册表中删除用户环境变量

**注意事项**：
- 新打开的终端/程序会自动读取新的环境变量
- 当前已打开的终端需要重新加载环境变量才能生效

