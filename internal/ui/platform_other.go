//go:build !windows

package ui

import (
	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/text"
)

// Linux（X11／Wayland）上 Gio 沒有提供設定視窗位置的方法，小工具由視窗管理員決定位置。

func viewHWND(app.ViewEvent) uintptr                           { return 0 }
func setupWidgetWindow(uintptr, int, int, bool, *widgetPlacer) {}
func windowPos(uintptr) (int, int, bool)                       { return 0, 0, false }
func bringToFront(uintptr)                                     {}
func moveWindow(uintptr, int, int)                             {}

// widgetPlacer 只有 Windows 會移動小工具，這裡不做事。
type widgetPlacer struct{}

func (*widgetPlacer) check(uintptr)                       {}
func (*widgetPlacer) reset()                              {}
func (*widgetPlacer) savedPos(h uintptr) (int, int, bool) { return windowPos(h) }

// newShaper 使用 Go 字型，中文由系統字型（如 Noto Sans CJK）補上。
func newShaper() *text.Shaper {
	return text.NewShaper(text.WithCollection(gofont.Collection()))
}
