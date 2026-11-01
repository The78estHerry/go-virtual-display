// Package display 查詢與切換虛擬螢幕的「插入／拔掉」狀態，實作依平台而異。
package display

import (
	"errors"

	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/i18n"
)

// State 是虛擬螢幕目前的狀態。
type State int

const (
	Unknown   State = iota // 查不到（驅動未安裝、設定錯誤等）
	Plugged                // 已插入
	Unplugged              // 已拔掉
)

func (s State) String() string {
	switch s {
	case Plugged:
		return i18n.T("Plugged in", "已插入")
	case Unplugged:
		return i18n.T("Unplugged", "已拔掉")
	}
	return i18n.T("Unknown", "未知")
}

// ErrNotInstalled 表示找不到虛擬螢幕（驅動或接頭沒有設定好）。
var ErrNotInstalled = errors.New(i18n.T("Virtual display not found. Please run the installer first.", "找不到虛擬螢幕，請先執行安裝腳本"))

// Controller 是平台相關的實作。
type Controller interface {
	Status() (State, error)
	Plug() error
	Unplug() error
}

// ErrRebootRequired 表示新的解析度已經存好，但要重新開機才會生效（Linux 沒有 debugfs 可以立即套用時）。
// 這不算失敗：使用中的解析度維持新選的那一組。
var ErrRebootRequired = errors.New(i18n.T("Saved; reboot to apply the new resolution", "已存好，重新開機後套用新解析度"))

// ProfileApplier 由支援切換解析度的平台實作（Windows、Linux）。
// 使用前先用 profile 套件把要用的解析度設為使用中。
type ProfileApplier interface {
	// ApplyActive 讓虛擬螢幕改用 profiles.json 裡使用中的解析度，完成或失敗後才返回。
	ApplyActive() error
}

// New 回傳目前平台的 Controller。
func New() (Controller, error) { return newController() }

// SynergyState 是 Synergy 服務的狀態，用來在系統列和小工具上顯示 Synergy 圖示（只有 Windows）。
type SynergyState int

const (
	SynergyNone    SynergyState = iota // 沒有安裝
	SynergyStopped                     // 有安裝但沒有在共用滑鼠鍵盤
	SynergyRunning                     // 服務和 synergy-core.exe 都在執行
)
