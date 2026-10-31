package main

import (
	"os"

	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/display"
)

// runSubcommand 處理 vdswitch.exe --apply（由最高權限排程工作 Apply 與安裝腳本執行）。
// 有處理時回傳 true。
func runSubcommand() bool {
	if len(os.Args) < 2 || os.Args[1] != "--apply" {
		return false
	}
	if err := display.ApplyElevated(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
	return true
}
