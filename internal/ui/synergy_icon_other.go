//go:build !windows

package ui

import "image"

// Synergy 圖示只有 Windows 版會顯示。
func synergyIcon(string, int, bool) image.Image { return nil }
func smallIconPx() int                          { return 16 }
