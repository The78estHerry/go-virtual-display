//go:build !windows

package main

import (
	"fmt"
	"os"

	"github.com/The78estHerry/go-virtual-display/internal/profile"
)

// runSubcommand 處理 vdswitch --write-edid <檔案>：依目前使用者 profiles.json 使用中的解析度
// （沒有的話是預設的 1920×1080）產生 EDID 寫到檔案。install.sh 用它讓安裝後的 EDID 和使用者選的解析度一致。
// 有處理時回傳 true。
func runSubcommand() bool {
	if len(os.Args) < 2 || os.Args[1] != "--write-edid" {
		return false
	}
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: vdswitch --write-edid <file>")
		os.Exit(2)
	}
	p := profile.Load().ActiveProfile()
	b, err := p.EDID()
	if err == nil {
		err = os.WriteFile(os.Args[2], b, 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(p.Key())
	os.Exit(0)
	return true
}
