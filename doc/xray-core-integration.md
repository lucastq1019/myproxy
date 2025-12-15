# Xray-core 集成指南

本文档介绍在 Go 项目中集成 xray-core 的几种方式及推荐方案。

## 集成方式概览

### 方式一：直接调用二进制文件（最简单但不推荐用于生产）

**优点：**
- 实现简单，无需修改代码结构
- 利用 xray-core 的完整功能
- 可以独立升级 xray-core

**缺点：**
- 需要通过进程间通信，性能开销较大
- 需要管理子进程生命周期
- 错误处理复杂
- 部署时需要包含 xray-core 二进制文件

**实现示例：**
```go
package xray

import (
    "context"
    "os/exec"
    "time"
)

type XrayBinary struct {
    binPath string
    configPath string
    cmd *exec.Cmd
}

func NewXrayBinary(binPath, configPath string) *XrayBinary {
    return &XrayBinary{
        binPath: binPath,
        configPath: configPath,
    }
}

func (x *XrayBinary) Start(ctx context.Context) error {
    x.cmd = exec.CommandContext(ctx, x.binPath, "-config", x.configPath)
    return x.cmd.Start()
}

func (x *XrayBinary) Stop() error {
    if x.cmd != nil && x.cmd.Process != nil {
        return x.cmd.Process.Kill()
    }
    return nil
}
```

### 方式二：作为库集成（推荐）

**优点：**
- 性能最优，无进程间通信开销
- 完全的程序化控制
- 可以直接调用内部 API
- 更好的错误处理和资源管理
- 单二进制部署

**缺点：**
- 需要理解 xray-core 的内部结构
- 编译时间可能较长
- 升级 xray-core 需要重新编译

**实现步骤：**

1. **添加依赖到 go.mod：**
```bash
go get github.com/xtls/xray-core@latest
```

2. **核心集成代码示例：**
```go
package xray

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "os"

    "github.com/xtls/xray-core/app/proxyman"
    "github.com/xtls/xray-core/app/stats"
    "github.com/xtls/xray-core/common/net"
    "github.com/xtls/xray-core/common/serial"
    "github.com/xtls/xray-core/core"
    "github.com/xtls/xray-core/infra/conf"
)

type XrayInstance struct {
    instance *core.Instance
    ctx      context.Context
    cancel   context.CancelFunc
}

// NewXrayInstance 创建 xray-core 实例
func NewXrayInstance(configPath string) (*XrayInstance, error) {
    // 读取配置文件
    configBytes, err := os.ReadFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("读取配置文件失败: %w", err)
    }

    // 解析配置
    config, err := conf.ParseJSON(configBytes)
    if err != nil {
        return nil, fmt.Errorf("解析配置失败: %w", err)
    }

    // 创建 xray-core 实例
    pbConfig, err := config.Build()
    if err != nil {
        return nil, fmt.Errorf("构建配置失败: %w", err)
    }

    instance, err := core.New(pbConfig)
    if err != nil {
        return nil, fmt.Errorf("创建实例失败: %w", err)
    }

    ctx, cancel := context.WithCancel(context.Background())

    return &XrayInstance{
        instance: instance,
        ctx:      ctx,
        cancel:   cancel,
    }, nil
}

// NewXrayInstanceFromJSON 从 JSON 配置创建实例
func NewXrayInstanceFromJSON(configJSON []byte) (*XrayInstance, error) {
    config, err := conf.ParseJSON(configJSON)
    if err != nil {
        return nil, fmt.Errorf("解析配置失败: %w", err)
    }

    pbConfig, err := config.Build()
    if err != nil {
        return nil, fmt.Errorf("构建配置失败: %w", err)
    }

    instance, err := core.New(pbConfig)
    if err != nil {
        return nil, fmt.Errorf("创建实例失败: %w", err)
    }

    ctx, cancel := context.WithCancel(context.Background())

    return &XrayInstance{
        instance: instance,
        ctx:      ctx,
        cancel:   cancel,
    }, nil
}

// Start 启动 xray-core 实例
func (xi *XrayInstance) Start() error {
    if err := xi.instance.Start(); err != nil {
        return fmt.Errorf("启动失败: %w", err)
    }
    return nil
}

// Stop 停止 xray-core 实例
func (xi *XrayInstance) Stop() error {
    xi.cancel()
    if xi.instance != nil {
        xi.instance.Close()
    }
    return nil
}

// CreateOutboundHandler 创建出站处理器（用于动态路由）
func (xi *XrayInstance) CreateOutboundHandler(tag string, outboundConfig *conf.OutboundDetourConfig) error {
    // 这个方法允许动态创建出站连接
    // 适用于需要根据目标地址动态选择代理的场景
    pbConfig, err := outboundConfig.Build()
    if err != nil {
        return err
    }

    // 获取 OutboundHandlerManager
    ohm := xi.instance.GetFeature(proxyman.OutboundHandlerManagerType()).(proxyman.OutboundHandlerManager)
    
    return ohm.AddHandler(context.Background(), &core.OutboundHandlerConfig{
        Tag:              tag,
        SenderSettings:   serial.ToTypedMessage(&proxyman.SenderConfig{}),
        ProxySettings:    pbConfig,
    })
}

// 创建一个简单的 SOCKS5 出站配置
func CreateSOCKS5OutboundConfig(address string, port int, username, password string) *conf.OutboundDetourConfig {
    outboundConfig := &conf.OutboundDetourConfig{
        Protocol: "socks",
        Tag:      "socks-out",
        StreamSetting: &conf.StreamConfig{
            Network: "tcp",
        },
        Settings: &serial.TypedMessage{
            Type: "v2ray.core.proxy.socks.Config",
            Value: mustMarshalJSON(map[string]interface{}{
                "auth": map[string]string{
                    "user": username,
                    "pass": password,
                },
                "servers": []map[string]interface{}{
                    {
                        "address": address,
                        "port":    port,
                    },
                },
            }),
        },
    }
    return outboundConfig
}

func mustMarshalJSON(v interface{}) []byte {
    data, _ := json.Marshal(v)
    return data
}
```

### 方式三：使用 xray-core 作为 SOCKS5 客户端（轻量级集成）

如果你只需要使用 xray-core 的协议实现（如 VMess、VLESS 等），可以将其作为客户端库使用：

```go
package xray

import (
    "context"
    "net"

    "github.com/xtls/xray-core/common/net"
    "github.com/xtls/xray-core/core"
    "github.com/xtls/xray-core/proxy/vmess/outbound"
)

type XrayOutbound struct {
    outbound *outbound.Handler
}

// Dial 通过 xray-core 协议连接到目标
func (xo *XrayOutbound) Dial(ctx context.Context, dest net.Destination) (net.Conn, error) {
    return xo.outbound.Dial(ctx, dest)
}
```

## 推荐方案

**针对你的项目，推荐使用方式二（作为库集成），原因如下：**

1. **性能优势**：你的项目需要处理大量连接，库集成避免了进程间通信开销
2. **集成简单**：你的项目已经是 Go 项目，集成 Go 库非常自然
3. **控制精细**：可以完全控制 xray-core 的生命周期和配置
4. **功能完整**：可以获得 xray-core 的所有功能（路由、负载均衡、统计等）

## 集成到现有项目的建议

### 1. 创建 xray 包装模块

在你的项目中创建 `internal/xray/` 目录，封装 xray-core 的使用：

```go
// internal/xray/xray.go
package xray

import (
    "context"
    "fmt"
    
    "github.com/xtls/xray-core/core"
)

type XrayProxy struct {
    instance *core.Instance
    ctx      context.Context
    cancel   context.CancelFunc
}

// NewXrayProxy 创建 xray 代理实例
func NewXrayProxy(configJSON []byte) (*XrayProxy, error) {
    // 实现细节...
}

// Dial 创建出站连接（可以替换现有的 SOCKS5Client.Dial）
func (xp *XrayProxy) Dial(network, address string) (net.Conn, error) {
    // 使用 xray-core 的出站连接
}
```

### 2. 在 forwarder.go 中集成

修改 `internal/proxy/forwarder.go`，添加对 xray-core 的支持：

```go
type Forwarder struct {
    SOCKS5Client   *socks5.SOCKS5Client
    XrayProxy      *xray.XrayProxy  // 新增
    UseXray        bool              // 新增：是否使用 xray
    // ... 其他字段
}

func (f *Forwarder) handleTCPConnection(localConn net.Conn) {
    var proxyConn net.Conn
    var err error
    
    if f.UseXray {
        // 使用 xray-core
        proxyConn, err = f.XrayProxy.Dial("tcp", f.RemoteAddr)
    } else {
        // 使用现有的 SOCKS5 客户端
        proxyConn, err = f.SOCKS5Client.Dial("tcp", f.RemoteAddr)
    }
    
    // ... 后续处理
}
```

### 3. 配置文件支持

在 `config.json` 中添加 xray 配置选项：

```json
{
  "proxy": {
    "type": "xray",
    "xray_config": {
      "inbounds": [...],
      "outbounds": [...],
      "routing": {...}
    }
  }
}
```

## 依赖添加

在项目根目录执行：

```bash
go get github.com/xtls/xray-core@latest
go get github.com/xtls/xray-core/common/net
go get github.com/xtls/xray-core/core
go get github.com/xtls/xray-core/infra/conf
```

## 注意事项

1. **编译时间**：xray-core 是一个大型项目，首次编译可能需要较长时间
2. **CGO 依赖**：某些功能可能需要 CGO，确保你的环境支持
3. **版本兼容**：建议锁定 xray-core 版本，避免自动升级带来的问题
4. **资源管理**：确保正确关闭 xray-core 实例，避免资源泄漏

## 参考资源

- xray-core GitHub: https://github.com/xtls/xray-core
- xray-core 文档: https://xtls.github.io/
- 配置格式参考: https://xtls.github.io/config/

## 示例：完整的集成代码

参考 `internal/xray/xray_integration_example.go`（如果需要，可以创建这个文件）

