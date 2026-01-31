package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	// LevelDebug 调试级别
	LevelDebug LogLevel = iota
	// LevelInfo 信息级别
	LevelInfo
	// LevelWarn 警告级别
	LevelWarn
	// LevelError 错误级别
	LevelError
	// LevelFatal 致命级别
	LevelFatal
)

var levelNames = map[LogLevel]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
	LevelFatal: "FATAL",
}

// LogType 日志类型
type LogType string

const (
	// LogTypeApp 应用程序日志
	LogTypeApp LogType = "app"
	// LogTypeProxy 代理转发日志
	LogTypeProxy LogType = "proxy"
)

// LogPanelCallback 日志面板回调函数类型
// 当有新日志写入时，会调用此回调来更新UI
type LogPanelCallback func(level, logType, message, logLine string)

// Logger 日志记录器
// 负责统一管理日志文件的写入和UI显示，确保两者一致
type Logger struct {
	level         LogLevel
	file          *os.File // 单一日志文件
	console       bool
	mutex         sync.Mutex
	logFilePath   string
	logDir        string
	panelCallback LogPanelCallback // UI面板回调函数（用于实时更新UI）
}

const (
	// MaxLogFileSize 单个日志文件最大大小（10MB）
	MaxLogFileSize int64 = 10 * 1024 * 1024
)

// NewLogger 创建新的日志记录器
// 参数：
//   - logFilePath: 日志文件路径
//   - console: 是否输出到控制台
//   - level: 日志级别
//   - panelCallback: UI面板回调函数（可选，用于实时更新UI显示）
func NewLogger(logFilePath string, console bool, level string, panelCallback ...LogPanelCallback) (*Logger, error) {
	// 解析日志级别
	logLevel, err := parseLogLevel(level)
	if err != nil {
		return nil, err
	}

	// 获取日志目录
	logDir := filepath.Dir(logFilePath)
	baseName := filepath.Base(logFilePath)
	// 移除扩展名以获取基本名称
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// 构建统一的日志文件路径
	unifiedLogPath := logFilePath
	// 如果路径没有扩展名，添加 .log
	if filepath.Ext(unifiedLogPath) == "" {
		unifiedLogPath = unifiedLogPath + ".log"
	}

	logger := &Logger{
		level:       logLevel,
		console:     console,
		logFilePath: unifiedLogPath,
		logDir:      logDir,
	}

	// 设置UI面板回调（如果提供）
	if len(panelCallback) > 0 && panelCallback[0] != nil {
		logger.panelCallback = panelCallback[0]
	}

	// 创建目录（如果不存在）
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 启动时如果日志文件存在则归档
	if err := logger.archiveIfExists(unifiedLogPath); err != nil {
		return nil, fmt.Errorf("归档日志文件失败: %w", err)
	}

	// 打开统一的日志文件
	logFile, err := os.OpenFile(unifiedLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}
	logger.file = logFile

	return logger, nil
}

// archiveIfExists 如果日志文件存在则归档（启动时使用）
func (l *Logger) archiveIfExists(logPath string) error {
	// 检查文件是否存在
	fileInfo, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，不需要归档
			return nil
		}
		return err
	}

	// 如果文件存在且大小大于0，则归档
	if fileInfo.Size() > 0 {
		timestamp := time.Now().Format("20060102_150405")
		backupPath := fmt.Sprintf("%s.%s", logPath, timestamp)

		// 重命名文件为归档文件
		if err := os.Rename(logPath, backupPath); err != nil {
			return fmt.Errorf("归档日志文件失败: %w", err)
		}
	}

	return nil
}

// rotateIfNeeded 检查日志文件大小，如果超过阈值则归档（运行时使用）
func (l *Logger) rotateIfNeeded(logPath string) error {
	// 检查文件是否存在
	fileInfo, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，不需要归档
			return nil
		}
		return err
	}

	// 检查文件大小
	if fileInfo.Size() < MaxLogFileSize {
		// 文件大小未超过阈值，不需要归档
		return nil
	}

	// 文件大小超过阈值，进行归档
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.%s", logPath, timestamp)

	// 重命名文件为归档文件
	if err := os.Rename(logPath, backupPath); err != nil {
		return fmt.Errorf("归档日志文件失败: %w", err)
	}

	return nil
}

// parseLogLevel 解析日志级别字符串
func parseLogLevel(level string) (LogLevel, error) {
	level = strings.ToLower(level)
	// 如果日志级别为空，返回默认级别
	if level == "" {
		return LevelInfo, nil
	}
	switch level {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	case "fatal":
		return LevelFatal, nil
	default:
		return LevelInfo, fmt.Errorf("无效的日志级别: %s", level)
	}
}

// log 记录日志
func (l *Logger) log(level LogLevel, logType LogType, format string, args ...interface{}) {
	// 检查日志级别
	if level < l.level {
		return
	}

	// 规范化日志类型：仅保留 app / xray，其他归并为 app
	logTypeStr := strings.ToLower(string(logType))
	if logTypeStr != "xray" {
		logTypeStr = "app"
	}

	// 生成日志消息
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelName := levelNames[level]
	message := fmt.Sprintf(format, args...)
	// 在日志中添加类型标识
	logLine := fmt.Sprintf("%s [%s] [%s] %s\n", timestamp, levelName, logTypeStr, message)

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 输出到控制台
	if l.console {
		fmt.Print(logLine)
	}

	// 输出到统一的日志文件
	if l.file != nil {
		if _, err := l.file.WriteString(logLine); err != nil {
			// 如果写入文件失败，尝试重新打开文件
			l.reopenFile()
			// 再次尝试写入
			if l.file != nil {
				l.file.WriteString(logLine)
			}
		}
	}

	// 通知UI面板更新（确保文件写入和UI显示一致）
	if l.panelCallback != nil {
		// 移除末尾的换行符，因为UI显示不需要
		logLineForUI := strings.TrimRight(logLine, "\n")
		l.panelCallback(levelName, logTypeStr, message, logLineForUI)
	}

	// 如果是致命错误，退出程序
	if level == LevelFatal {
		os.Exit(1)
	}
}

// SetPanelCallback 设置UI面板回调函数
func (l *Logger) SetPanelCallback(callback LogPanelCallback) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.panelCallback = callback
}

// reopenFile 重新打开日志文件
func (l *Logger) reopenFile() {
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}

	newFile, err := os.OpenFile(l.logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		l.file = newFile
	}
}

// InfoWithType 记录指定类型的信息日志
func (l *Logger) InfoWithType(logType LogType, format string, args ...interface{}) {
	l.log(LevelInfo, logType, format, args...)
}

// Error 记录错误日志（默认应用日志）
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, LogTypeApp, format, args...)
}

// GetLogLevel 获取当前日志级别
func (l *Logger) GetLogLevel() string {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return strings.ToLower(levelNames[l.level])
}

// SetLogLevel 设置日志级别
func (l *Logger) SetLogLevel(level string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if logLevel, err := parseLogLevel(level); err == nil {
		l.level = logLevel
	}
}

// Close 关闭日志记录器
func (l *Logger) Close() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 关闭日志文件
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}
}

// GetLogFilePath 获取日志文件路径
func (l *Logger) GetLogFilePath() string {
	return l.logFilePath
}

// Log 记录日志（通用方法，支持外部调用）
func (l *Logger) Log(level, logType, message string) {
	// 解析日志级别
	logLevel, err := parseLogLevel(level)
	if err != nil {
		logLevel = LevelInfo
	}

	// 解析日志类型
	var lt LogType = LogTypeApp
	if strings.ToLower(logType) == "xray" {
		lt = LogTypeProxy
	}

	l.log(logLevel, lt, "%s", message)
}

// SafeLogger 安全日志包装器，处理 Logger 为 nil 的情况
type SafeLogger struct {
	logger *Logger
}

// NewSafeLogger 创建安全日志包装器
func NewSafeLogger(logger *Logger) *SafeLogger {
	return &SafeLogger{
		logger: logger,
	}
}

// Log 记录日志（安全方法，处理 logger 为 nil 的情况）
func (sl *SafeLogger) Log(level, logType, message string) {
	if sl.logger != nil {
		sl.logger.Log(level, logType, message)
	}
}

// Info 记录信息日志
func (sl *SafeLogger) Info(message string) {
	sl.Log("INFO", "app", message)
}

// Error 记录错误日志
func (sl *SafeLogger) Error(message string) {
	sl.Log("ERROR", "app", message)
}

// Warn 记录警告日志
func (sl *SafeLogger) Warn(message string) {
	sl.Log("WARN", "app", message)
}

// Debug 记录调试日志
func (sl *SafeLogger) Debug(message string) {
	sl.Log("DEBUG", "app", message)
}

// IsReady 检查 Logger 是否已初始化
func (sl *SafeLogger) IsReady() bool {
	return sl.logger != nil
}

// SetLogger 设置底层 Logger
func (sl *SafeLogger) SetLogger(logger *Logger) {
	sl.logger = logger
}
