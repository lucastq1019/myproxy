@echo off
REM SOCKS/Xray 桌面代理客户端 - Windows 构建脚本
REM 支持 Windows, Linux, Mac 交叉编译

setlocal enabledelayedexpansion

REM 项目配置
set PROJECT_NAME=proxy-gui
set VERSION=%VERSION%
if "%VERSION%"=="" set VERSION=%date:~0,4%%date:~5,2%%date:~8,2%-%time:~0,2%%time:~3,2%%time:~6,2%
set VERSION=%VERSION: =0%
set BUILD_DIR=dist
set MAIN_PATH=./cmd/gui/main.go

REM 检查 Go 环境
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Go 未安装或不在 PATH 中
    exit /b 1
)

for /f "tokens=3" %%i in ('go version') do set GO_VERSION=%%i
echo [INFO] 检测到 Go 版本: %GO_VERSION%

REM 清理构建目录
if exist "%BUILD_DIR%" (
    echo [INFO] 清理旧的构建目录...
    rmdir /s /q "%BUILD_DIR%"
)
mkdir "%BUILD_DIR%"

REM 构建函数
goto :main

:build_target
set OS=%1
set ARCH=%2
set EXT=%3
set OUTPUT_NAME=%PROJECT_NAME%%EXT%

echo [INFO] 构建 %OS%/%ARCH%...

set GOOS=%OS%
set GOARCH=%ARCH%
set CGO_ENABLED=1

go build -ldflags="-s -w -X main.version=%VERSION%" -o "%BUILD_DIR%\%OS%-%ARCH%\%OUTPUT_NAME%" %MAIN_PATH%

if %errorlevel% equ 0 (
    echo [INFO] ✓ %OS%/%ARCH% 构建成功: %BUILD_DIR%\%OS%-%ARCH%\%OUTPUT_NAME%
) else (
    echo [ERROR] ✗ %OS%/%ARCH% 构建失败
    exit /b 1
)
goto :eof

:main
if "%1"=="" goto :build_all
if "%1"=="clean" (
    echo [INFO] 仅清理构建目录
    exit /b 0
)
if "%1"=="help" goto :help
if "%1"=="-h" goto :help
if "%1"=="--help" goto :help
if "%1"=="windows" goto :build_windows
if "%1"=="win" goto :build_windows
if "%1"=="linux" goto :build_linux
if "%1"=="mac" goto :build_mac
if "%1"=="darwin" goto :build_mac
if "%1"=="macos" goto :build_mac

echo [ERROR] 未知平台: %1
echo [INFO] 支持的平台: windows, linux, mac
exit /b 1

:build_all
echo [INFO] 开始构建所有平台...
echo [INFO] 版本: %VERSION%
echo [INFO] 构建目录: %BUILD_DIR%
echo.

call :build_target windows amd64 .exe
call :build_target windows 386 .exe
call :build_target linux amd64 ""
call :build_target linux arm64 ""
call :build_target darwin amd64 ""
call :build_target darwin arm64 ""

echo.
echo [INFO] 所有构建完成！
goto :end

:build_windows
echo [INFO] 构建 Windows 平台...
call :build_target windows amd64 .exe
call :build_target windows 386 .exe
goto :end

:build_linux
echo [INFO] 构建 Linux 平台...
call :build_target linux amd64 ""
call :build_target linux arm64 ""
goto :end

:build_mac
echo [INFO] 构建 macOS 平台...
call :build_target darwin amd64 ""
call :build_target darwin arm64 ""
goto :end

:help
echo 用法: %0 [平台]
echo.
echo 平台选项:
echo   windows, win    - 构建 Windows 版本
echo   linux           - 构建 Linux 版本
echo   mac, darwin     - 构建 macOS 版本
echo   (无参数)        - 构建所有平台
echo.
echo 其他选项:
echo   clean           - 仅清理构建目录
echo   help, -h        - 显示此帮助信息
echo.
echo 环境变量:
echo   VERSION         - 设置版本号 (默认: 时间戳)
echo.
echo 示例:
echo   %0              # 构建所有平台
echo   %0 windows      # 仅构建 Windows
echo   %0 linux        # 仅构建 Linux
echo   %0 mac          # 仅构建 macOS
echo   set VERSION=1.0.0 ^& %0  # 使用指定版本号
goto :end

:end
echo.
echo [INFO] 构建完成！文件位于: %BUILD_DIR%
endlocal
