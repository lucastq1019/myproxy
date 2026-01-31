package error

import (
	"fmt"
)

// AppError 定义结构化应用错误
type AppError struct {
	Code    string // 错误码
	Message string // 错误消息
	Err     error  // 原始错误（可选）
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap 接口
func (e *AppError) Unwrap() error {
	return e.Err
}
