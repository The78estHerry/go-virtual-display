package main

import (
	"os"

	"golang.org/x/sys/windows"

	"github.com/The78estHerry/go-virtual-display/internal/display"
	"github.com/The78estHerry/go-virtual-display/internal/i18n"
)

// runSubcommand 處理以最高權限排程工作執行的子命令（安裝腳本也會執行 --apply）：
//   - --apply：套用使用中的解析度
//   - --plug／--unplug：啟用／停用虛擬螢幕裝置（0.4.4 起，之前是 pnputil）
//   - --request-unplug：開始功能表的「Unplug Virtual Display」捷徑，以一般權限觸發排程工作 Unplug
//
// 有處理時回傳 true。
func runSubcommand() bool {
	if len(os.Args) < 2 {
		return false
	}
	var err error
	switch os.Args[1] {
	case "--apply":
		err = display.ApplyElevated()
	case "--plug":
		err = display.SetEnabledElevated(true)
	case "--unplug":
		err = display.SetEnabledElevated(false)
	case "--request-unplug":
		// 這時通常看不到主螢幕，但失敗時仍跳訊息方塊，至少遠端或切回來時看得到原因
		if err = display.RequestUnplug(); err != nil {
			title := i18n.T("Unplug Virtual Display", "拔掉虛擬螢幕")
			_, _ = windows.MessageBox(0, windows.StringToUTF16Ptr(err.Error()), windows.StringToUTF16Ptr(title), windows.MB_OK|windows.MB_ICONERROR|windows.MB_TOPMOST)
		}
	default:
		return false
	}
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
	return true
}
