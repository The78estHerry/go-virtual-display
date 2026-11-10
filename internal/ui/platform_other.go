//go:build !windows

package ui

import (
	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/text"
)

// 小工具視窗的位置由 go-gio-kit 的 giowidget 處理（Linux 上由視窗管理員決定）。

func viewHWND(app.ViewEvent) uintptr { return 0 }
func bringToFront(uintptr)           {}

// newShaper 使用 Go 字型，中文由系統字型（如 Noto Sans CJK）補上。
func newShaper() *text.Shaper {
	return text.NewShaper(text.WithCollection(gofont.Collection()))
}
