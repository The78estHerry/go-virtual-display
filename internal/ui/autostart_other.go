//go:build !windows

package ui

import (
	"fmt"
	"os"
	"path/filepath"
)

func autostartFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "autostart", "go-virtual-display.desktop"), nil
}

// AutostartEnabled 回傳是否已設定登入時自動啟動（XDG autostart）。
func AutostartEnabled() bool {
	p, err := autostartFile()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// SetAutostart 開啟或關閉登入時自動啟動。
func SetAutostart(on bool) error {
	p, err := autostartFile()
	if err != nil {
		return err
	}
	if !on {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	desktop := fmt.Sprintf("[Desktop Entry]\nType=Application\nName=虛擬螢幕開關\nExec=%q\nX-GNOME-Autostart-enabled=true\n", exe)
	return os.WriteFile(p, []byte(desktop), 0o644)
}
