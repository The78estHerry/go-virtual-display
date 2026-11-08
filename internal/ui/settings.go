package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Settings 是使用者偏好，存在 <使用者設定目錄>/go-virtual-display/widget.json。
type Settings struct {
	ShowWidget bool `json:"show_widget"`
	TopMost    bool `json:"top_most"`
	HasPos     bool `json:"has_pos"`
	X          int  `json:"x"`
	Y          int  `json:"y"`
	// Hotkey 是把小工具移到滑鼠所在螢幕的快捷鍵，空字串表示預設值（DefaultHotkey）
	Hotkey string `json:"hotkey,omitempty"`
}

var settingsMu sync.Mutex

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "go-virtual-display", "widget.json"), nil
}

// LoadSettings 讀取設定；檔案不存在或損壞時回傳預設值（顯示小工具）。
func LoadSettings() Settings {
	s := Settings{ShowWidget: true}
	p, err := settingsPath()
	if err != nil {
		return s
	}
	if b, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	return s
}

// SaveSettings 以 update 修改設定後寫回檔案。
func SaveSettings(update func(*Settings)) {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	s := LoadSettings()
	update(&s)
	p, err := settingsPath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	if b, err := json.MarshalIndent(s, "", "  "); err == nil {
		_ = os.WriteFile(p, b, 0o644)
	}
}
