// vdswitch 是虛擬螢幕的拔插開關：系統列圖示＋桌面小工具。
package main

import (
	"errors"

	"github.com/The78estHerry/go-desktop-kit/desktop/single"

	"github.com/The78estHerry/go-virtual-display/internal/ui"
)

// instanceID 也是登入時自動啟動的名稱，見 internal/ui/autostart.go。
const instanceID = "go-virtual-display"

func main() {
	if runSubcommand() {
		return
	}
	// 只執行一份：第二份啟動時通知第一份顯示小工具後結束。
	// 取得鎖失敗（不是已有一份在執行）時照常啟動，只是少了這個保護。
	var show <-chan struct{}
	inst, err := single.Acquire(instanceID)
	switch {
	case errors.Is(err, single.ErrAlreadyRunning):
		return
	case err == nil:
		defer inst.Release()
		show = inst.Activated()
	}
	ui.Run(show)
}
