// vdswitch 是虛擬螢幕的拔插開關：系統列圖示＋桌面小工具。
package main

import (
	"github.com/The78estHerry/go-virtual-display/internal/ui"
)

func main() {
	if runSubcommand() {
		return
	}
	show, ok := acquireSingleInstance()
	if !ok {
		return // 已有一份在執行，已通知它顯示小工具
	}
	ui.Run(show)
}
