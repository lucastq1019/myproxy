package systemproxy

import (
	"fmt"
	"os/exec"
)

// SetGitGlobalProxy 执行 git config --global 写入 http.proxy / https.proxy。
// 未在 PATH 中找到 git 可执行文件时静默返回 nil（不报错）。
func SetGitGlobalProxy(host string, port int, proxyType string) error {
	git, ok := lookPathGit()
	if !ok {
		return nil
	}
	url := TerminalProxyURL(host, port, proxyType)
	if out, err := exec.Command(git, "config", "--global", "http.proxy", url).CombinedOutput(); err != nil {
		return fmt.Errorf("git config http.proxy: %w: %s", err, string(out))
	}
	if out, err := exec.Command(git, "config", "--global", "https.proxy", url).CombinedOutput(); err != nil {
		return fmt.Errorf("git config https.proxy: %w: %s", err, string(out))
	}
	return nil
}

// ClearGitGlobalProxy 取消全局 http.proxy / https.proxy。
// 未在 PATH 中找到 git 时静默返回 nil。键不存在等错误忽略。
func ClearGitGlobalProxy() error {
	git, ok := lookPathGit()
	if !ok {
		return nil
	}
	_ = exec.Command(git, "config", "--global", "--unset", "http.proxy").Run()
	_ = exec.Command(git, "config", "--global", "--unset", "https.proxy").Run()
	return nil
}

func lookPathGit() (string, bool) {
	p, err := exec.LookPath("git")
	if err != nil || p == "" {
		return "", false
	}
	return p, true
}
