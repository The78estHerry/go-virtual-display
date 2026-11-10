package ui

import (
	"os"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"
	"golang.org/x/sys/windows"

	"github.com/The78estHerry/go-virtual-display/internal/i18n"
)

// 小工具視窗的位置與樣式由 go-gio-kit 的 giowidget 處理；這裡只剩其他視窗與螢幕配置偵測
// （layout_windows.go）用到的 Win32 函式。

var (
	user32             = windows.NewLazySystemDLL("user32.dll")
	procShowWindow     = user32.NewProc("ShowWindow")
	procSetForeground  = user32.NewProc("SetForegroundWindow")
	procGetMonitorInfo = user32.NewProc("GetMonitorInfoW")
)

const swShowNoActivate = 4

type rect struct{ Left, Top, Right, Bottom int32 }

type monitorInfo struct {
	CbSize  uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

func viewHWND(e app.ViewEvent) uintptr {
	if v, ok := e.(app.Win32ViewEvent); ok {
		return v.HWND
	}
	return 0
}

func bringToFront(hwnd uintptr) {
	procShowWindow.Call(hwnd, swShowNoActivate)
	procSetForeground.Call(hwnd)
}

// newShaper 英文介面只用 Go 字型；中文介面另外載入微軟正黑體，找不到時改用系統字型。
func newShaper() *text.Shaper {
	coll := gofont.Collection()
	if !i18n.IsChinese() {
		return text.NewShaper(text.NoSystemFonts(), text.WithCollection(coll))
	}
	b, err := os.ReadFile(os.Getenv("WINDIR") + `\Fonts\msjh.ttc`)
	if err == nil {
		if faces, err := opentype.ParseCollection(b); err == nil {
			coll = append(coll, faces...)
			return text.NewShaper(text.NoSystemFonts(), text.WithCollection(coll))
		}
	}
	return text.NewShaper(text.WithCollection(coll))
}
