package ui

import (
	"github.com/The78estHerry/go-desktop-kit/desktop/autostart"

	"github.com/The78estHerry/go-virtual-display/internal/i18n"
)

// autostartID 是 Windows Run 機碼的值名稱與 Linux 的 go-virtual-display.desktop，
// 安裝與移除腳本也用這個名字，不能改。
const autostartID = "go-virtual-display"

// AutostartEnabled 回傳是否已設定登入時自動啟動。
func AutostartEnabled() bool {
	on, _ := autostart.Enabled(autostartID)
	return on
}

// SetAutostart 開啟或關閉登入時自動啟動（Windows：HKCU Run；macOS：LaunchAgent；Linux：XDG autostart）。
func SetAutostart(on bool) error {
	if !on {
		return autostart.Disable(autostartID)
	}
	return autostart.Enable(autostart.Entry{ID: autostartID, Name: i18n.T("virtual_display_switch")})
}
