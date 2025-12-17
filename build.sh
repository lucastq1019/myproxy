#!/bin/bash

# SOCKS/Xray 桌面代理客户端 - 跨平台构建脚本
# 支持 Windows, Linux, Mac

set -e

# 项目配置
PROJECT_NAME="proxy-gui"
VERSION="${VERSION:-$(date +%Y%m%d-%H%M%S)}"
BUILD_DIR="dist"
MAIN_PATH="./cmd/gui/main.go"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 Go 环境
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go 未安装或不在 PATH 中"
        exit 1
    fi
    GO_VERSION=$(go version | awk '{print $3}')
    print_info "检测到 Go 版本: $GO_VERSION"
}

# 清理构建目录
clean() {
    if [ -d "$BUILD_DIR" ]; then
        print_info "清理旧的构建目录..."
        rm -rf "$BUILD_DIR"
    fi
    mkdir -p "$BUILD_DIR"
}

# 构建函数
build_target() {
    local os=$1
    local arch=$2
    local ext=$3
    local output_name="${PROJECT_NAME}"
    
    if [ -n "$ext" ]; then
        output_name="${output_name}${ext}"
    fi
    
    local output_path="${BUILD_DIR}/${os}-${arch}/${output_name}"
    
    print_info "构建 ${os}/${arch}..."
    
    export GOOS="$os"
    export GOARCH="$arch"
    export CGO_ENABLED=1
    
    # Fyne 应用需要 CGO，某些平台可能需要特殊处理
    case "$os" in
        windows)
            # Windows 构建
            go build -ldflags="-s -w -X main.version=${VERSION}" \
                -o "$output_path" "$MAIN_PATH"
            ;;
        linux)
            # Linux 构建
            go build -ldflags="-s -w -X main.version=${VERSION}" \
                -o "$output_path" "$MAIN_PATH"
            ;;
        darwin)
            # macOS 构建
            go build -ldflags="-s -w -X main.version=${VERSION}" \
                -o "$output_path" "$MAIN_PATH"
            ;;
        *)
            print_error "不支持的平台: $os"
            return 1
            ;;
    esac
    
    if [ $? -eq 0 ]; then
        print_info "✓ ${os}/${arch} 构建成功: $output_path"
        
        # 显示文件大小
        if command -v du &> /dev/null; then
            size=$(du -h "$output_path" | cut -f1)
            print_info "  文件大小: $size"
        fi
    else
        print_error "✗ ${os}/${arch} 构建失败"
        return 1
    fi
}

# 构建所有目标
build_all() {
    print_info "开始构建所有平台..."
    print_info "版本: $VERSION"
    print_info "构建目录: $BUILD_DIR"
    echo ""
    
    # Windows
    build_target windows amd64 .exe
    build_target windows 386 .exe
    
    # Linux
    build_target linux amd64
    build_target linux arm64
    
    # macOS
    build_target darwin amd64
    build_target darwin arm64
    
    echo ""
    print_info "所有构建完成！"
    print_info "输出目录: $BUILD_DIR"
}

# 构建单个平台
build_single() {
    local platform=$1
    case "$platform" in
        windows|win)
            print_info "构建 Windows 平台..."
            build_target windows amd64 .exe
            build_target windows 386 .exe
            ;;
        linux)
            print_info "构建 Linux 平台..."
            build_target linux amd64
            build_target linux arm64
            ;;
        mac|darwin|macos)
            print_info "构建 macOS 平台..."
            build_target darwin amd64
            build_target darwin arm64
            ;;
        *)
            print_error "未知平台: $platform"
            print_info "支持的平台: windows, linux, mac"
            exit 1
            ;;
    esac
}

# 主函数
main() {
    check_go
    clean
    
    if [ $# -eq 0 ]; then
        # 没有参数，构建所有平台
        build_all
    elif [ "$1" == "clean" ]; then
        print_info "仅清理构建目录"
        exit 0
    elif [ "$1" == "help" ] || [ "$1" == "-h" ] || [ "$1" == "--help" ]; then
        echo "用法: $0 [平台]"
        echo ""
        echo "平台选项:"
        echo "  windows, win    - 构建 Windows 版本"
        echo "  linux           - 构建 Linux 版本"
        echo "  mac, darwin     - 构建 macOS 版本"
        echo "  (无参数)        - 构建所有平台"
        echo ""
        echo "其他选项:"
        echo "  clean           - 仅清理构建目录"
        echo "  help, -h        - 显示此帮助信息"
        echo ""
        echo "环境变量:"
        echo "  VERSION         - 设置版本号 (默认: 时间戳)"
        echo ""
        echo "示例:"
        echo "  $0              # 构建所有平台"
        echo "  $0 windows      # 仅构建 Windows"
        echo "  $0 linux        # 仅构建 Linux"
        echo "  $0 mac          # 仅构建 macOS"
        echo "  VERSION=1.0.0 $0  # 使用指定版本号"
        exit 0
    else
        # 构建指定平台
        build_single "$1"
    fi
    
    echo ""
    print_info "构建完成！文件位于: $BUILD_DIR"
}

# 执行主函数
main "$@"
