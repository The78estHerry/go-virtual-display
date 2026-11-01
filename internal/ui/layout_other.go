//go:build !windows

package ui

// watchDisplayLayout 只有 Windows 需要（Synergy 子機的問題），其他平台不做事。
func watchDisplayLayout() {}
