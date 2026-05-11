# MyProxy - 现代化跨平台代理客户端

<div align="center">

![License](https://img.shields.io/badge/License-MIT-green.svg)
![Language](https://img.shields.io/badge/Language-Go-blue.svg)
![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey.svg)

**简洁、高效、易用的桌面代理管理工具**

[English](#english) | [中文](#中文)

</div>

## 中文

### 📋 项目介绍

MyProxy 是一款现代化的跨平台代理客户端，基于 Go 语言开发，集成 xray-core 代理引擎，提供简洁直观的图形化界面。无论是个人用户还是开发者，都能轻松管理和使用代理服务。

### ✨ 核心特性

- **🎨 现代化 GUI**
  - 基于 Fyne 框架打造流畅的用户界面
  - 支持浅色/深色主题切换
  - 响应式布局，窗口状态自动保存

- **⚡ 强大的代理引擎**
  - 内置 xray-core，开箱即用
  - 支持 SOCKS5、VMess 等多种协议
  - 本地 10808 端口监听

- **📡 灵活的订阅管理**
  - 支持 VMess、SOCKS5 协议
  - 兼容 JSON 和 Base64 格式
  - 多标签分类管理订阅源

- **🔧 完整的功能模块**
  - 订阅管理与更新
  - 服务器列表展示
  - 延迟测试与服务器筛选
  - 实时日志监控
  - 系统代理配置（macOS、Windows）
  - 环境变量代理设置（跨平台）

- **💾 数据持久化**
  - SQLite 数据库存储
  - 配置自动保存
  - 断点恢复

### 🚀 快速开始

#### 系统要求
- **Go** 1.25.4 或更高版本
- **操作系统**: macOS / Windows / Linux（需要图形环境）

#### 安装与运行

```bash
# 克隆项目
git clone https://github.com/lucastq1019/myproxy.git
cd myproxy

# 下载依赖
go mod download

# 运行应用
go run ./cmd/gui/main.go
```

或者编译后独立运行：

```bash
# 编译
go build -o myproxy ./cmd/gui

# 运行
./myproxy
```

首次启动会自动创建数据库并初始化配置。

### 📖 使用指南

#### 基本流程

1. **添加订阅** 📌
   - 在订阅面板输入订阅 URL
   - 可选：添加自定义标签便于分类
   - 应用会自动解析订阅内容

2. **查看服务器** 📍
   - 订阅解析完成后，服务器列表自动更新
   - 显示服务器名称、协议、地区等信息

3. **测试延迟** 📊
   - 点击服务器列表中的测试按钮
   - 获取实时延迟数据
   - 根据延迟选择最优服务器

4. **启动代理** ▶️
   - 选择目标服务器
   - 点击启动按钮
   - 默认在本地 10808 端口监听

5. **查看日志** 📋
   - 实时监控应用和代理引擎日志
   - 快速定位问题

#### 系统代理设置

在代理运行中，可选择启用系统代理：
- **macOS**: 自动配置系统网络设置
- **Windows**: 通过注册表配置系统代理
- **Linux**: 支持环境变量代理

### ⚙️ 配置说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| 监听地址 | SOCKS5 代理监听地址 | 127.0.0.1 |
| 监听端口 | SOCKS5 代理监听端口 | 10808 |
| 数据库 | 配置和订阅数据存储位置 | `./data/myproxy.db` |
| 日志文件 | 应用运行日志 | `./myproxy.log` |

### 🏗️ 技术架构

```
┌─────────────────────────────┐
│      GUI 层 (Fyne)           │
│  用户界面 / 事件处理          │
└──────────────┬──────────────┘
               │
┌──────────────▼──────────────┐
│    业务逻辑层                  │
│ 订阅管理 / 代理控制 / 配置      │
└──────────────┬──────────────┘
               │
┌──────────────▼──────────────┐
│   代理引擎层 (xray-core)      │
│  SOCKS5 / VMess 协议支持      │
└──────────────┬──────────────┘
               │
┌──────────────▼──────────────┐
│  存储层 (SQLite)             │
│  持久化数据存储               │
└─────────────────────────────┘
```

**核心技术栈**
- **语言**: Go (1.25.4+)
- **GUI框架**: Fyne
- **代理引擎**: xray-core (库模式)
- **数据存储**: SQLite 3
- **订阅解析**: 自定义解析器

### 📋 平台支持

| 平台 | 系统代理 | 环境变量代理 | 状态 |
|------|--------|-----------|------|
| macOS | ✅ | ✅ | 完整支持 |
| Windows | ✅ | ✅ | 完整支持 |
| Linux | ⏳ | ✅ | 部分支持 |

### 🤝 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

### 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

### ⚖️ 免责声明

**重要**: 本项目仅供学习和研究使用。用户需自行确保使用本项目符合所在地的法律法规。开发者不对任何不当使用行为负责。

### 📞 联系方式

- **GitHub Issues**: [报告问题](https://github.com/lucastq1019/myproxy/issues)
- **讨论区**: [GitHub Discussions](https://github.com/lucastq1019/myproxy/discussions)

---

## English

### 📋 Project Overview

MyProxy is a modern cross-platform proxy client developed in Go, integrated with xray-core proxy engine, providing a clean and intuitive graphical interface. Whether you're an individual user or developer, you can easily manage and use proxy services.

### ✨ Key Features

- **🎨 Modern GUI**
  - Built with Fyne framework for smooth user experience
  - Support for light/dark theme switching
  - Responsive layout with automatic window state preservation

- **⚡ Powerful Proxy Engine**
  - Integrated xray-core, ready to use out of the box
  - Support for SOCKS5, VMess and other protocols
  - Listen on local port 10808

- **📡 Flexible Subscription Management**
  - Support for VMess and SOCKS5 protocols
  - Compatible with JSON and Base64 formats
  - Multi-label classification for subscription sources

- **🔧 Complete Feature Set**
  - Subscription management and updates
  - Server list display
  - Latency testing and server filtering
  - Real-time log monitoring
  - System proxy configuration (macOS, Windows)
  - Environment variable proxy setup (cross-platform)

- **💾 Data Persistence**
  - SQLite database storage
  - Automatic configuration saving
  - Breakpoint recovery

### 🚀 Quick Start

#### System Requirements
- **Go** 1.25.4 or higher
- **Operating System**: macOS / Windows / Linux (requires graphical environment)

#### Installation & Running

```bash
# Clone the project
git clone https://github.com/lucastq1019/myproxy.git
cd myproxy

# Download dependencies
go mod download

# Run the application
go run ./cmd/gui/main.go
```

Or compile and run standalone:

```bash
# Compile
go build -o myproxy ./cmd/gui

# Run
./myproxy
```

The database will be automatically created and initialized on first launch.

### 📖 Usage Guide

#### Basic Workflow

1. **Add Subscription** 📌
   - Enter subscription URL in the subscription panel
   - Optional: Add custom tags for classification
   - Application automatically parses subscription content

2. **View Servers** 📍
   - Server list updates automatically after subscription parsing
   - Displays server name, protocol, location and other information

3. **Test Latency** 📊
   - Click the test button in server list
   - Get real-time latency data
   - Select optimal server based on latency

4. **Start Proxy** ▶️
   - Select target server
   - Click start button
   - Listens on local port 10808 by default

5. **View Logs** 📋
   - Monitor application and proxy engine logs in real-time
   - Quickly identify and resolve issues

#### System Proxy Configuration

While the proxy is running, you can optionally enable system proxy:
- **macOS**: Automatically configure system network settings
- **Windows**: Configure system proxy via registry
- **Linux**: Support environment variable proxy

### ⚙️ Configuration

| Item | Description | Default |
|------|-------------|---------|
| Listen Address | SOCKS5 proxy listen address | 127.0.0.1 |
| Listen Port | SOCKS5 proxy listen port | 10808 |
| Database | Configuration and subscription data storage | `./data/myproxy.db` |
| Log File | Application log file | `./myproxy.log` |

### 🏗️ Architecture

```
┌─────────────────────────────┐
│      GUI Layer (Fyne)        │
│  User Interface / Events     │
└──────────────┬──────────────┘
               │
┌──────────────▼──────────────┐
│      Business Logic Layer     │
│ Subscriptions / Proxy Control │
└──────────────┬──────────────┘
               │
┌──────────────▼──────────────┐
│  Proxy Engine (xray-core)    │
│  SOCKS5 / VMess Support      │
└──────────────┬──────────────┘
               │
┌──────────────▼──────────────┐
│    Storage Layer (SQLite)    │
│   Persistent Data Storage    │
└─────────────────────────────┘
```

**Technology Stack**
- **Language**: Go (1.25.4+)
- **GUI Framework**: Fyne
- **Proxy Engine**: xray-core (library mode)
- **Data Storage**: SQLite 3
- **Subscription Parser**: Custom parser

### 📋 Platform Support

| Platform | System Proxy | Environment Proxy | Status |
|----------|-------------|------------------|--------|
| macOS | ✅ | ✅ | Fully Supported |
| Windows | ✅ | ✅ | Fully Supported |
| Linux | ⏳ | ✅ | Partial Support |

### 🤝 Contributing

We welcome issues and pull requests!

1. Fork this repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

### ⚖️ Disclaimer

**Important**: This project is for learning and research purposes only. Users are responsible for ensuring their use complies with applicable laws and regulations in their jurisdiction. The developer is not responsible for any misuse.

### 📞 Contact

- **GitHub Issues**: [Report Issues](https://github.com/lucastq1019/myproxy/issues)
- **Discussions**: [GitHub Discussions](https://github.com/lucastq1019/myproxy/discussions)
