package main

import (
	"os"

	"github.com/The78estHerry/go-virtual-display/internal/display"
)

// runSubcommand 處理以最高權限排程工作執行的子命令（安裝腳本也會執行 --apply）：
//   - --apply：套用使用中的解析度
//   - --plug／--unplug：啟用／停用虛擬螢幕裝置（0.4.4 起，之前是 pnputil）
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
	default:
		return false
	}
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
	return true
}
