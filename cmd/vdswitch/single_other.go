//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"syscall"
)

// acquireSingleInstance 以 flock 確保只執行一份；第二份直接結束。
func acquireSingleInstance() (<-chan struct{}, bool) {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	f, err := os.OpenFile(filepath.Join(dir, "go-virtual-display.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, true
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, false
	}
	// f 刻意不關閉，鎖在程序結束時自動釋放
	return nil, true
}
