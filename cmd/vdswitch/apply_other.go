//go:build !windows

package main

// Linux 版還不支援切換解析度，沒有子命令。
func runSubcommand() bool { return false }
