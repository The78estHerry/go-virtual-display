//go:build !windows

package ui

// Linux 用 linux/uninstall.sh 移除，系統列不提供解除安裝。
func uninstallSupported() bool { return false }

func startUninstall() bool { return false }
