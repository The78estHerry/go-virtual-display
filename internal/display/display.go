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

// New 回傳目前平台的 Controller。
func New() (Controller, error) { return newController() }
