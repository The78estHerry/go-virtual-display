package ui

import (
	"os"
	"unsafe"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"
	"golang.org/x/sys/windows"

	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/i18n"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	dwmapi               = windows.NewLazySystemDLL("dwmapi.dll")
	procGetWindowLongPtr = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtr = user32.NewProc("SetWindowLongPtrW")
	procSetWindowPos     = user32.NewProc("SetWindowPos")
	procGetWindowRect    = user32.NewProc("GetWindowRect")
	procShowWindow       = user32.NewProc("ShowWindow")
	procSetForeground    = user32.NewProc("SetForegroundWindow")
	procSystemParamsInfo = user32.NewProc("SystemParametersInfoW")
	procDwmSetWindowAttr = dwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	gwlExStyle         = -20
	wsExToolWindow     = 0x00000080
	wsExAppWindow      = 0x00040000
	swpNoSize          = 0x0001
	swpNoMove          = 0x0002
	swpNoZOrder        = 0x0004
	swpNoActivate      = 0x0010
	swpFrameChanged    = 0x0020
	swHide             = 0
	swShowNoActivate   = 4
	spiGetWorkArea     = 0x0030
	dwmwaCornerPref    = 33
	dwmwcpRound        = 2
	widgetMarginPixels = 24
)

type rect struct{ Left, Top, Right, Bottom int32 }

func viewHWND(e app.ViewEvent) uintptr {
	if v, ok := e.(app.Win32ViewEvent); ok {
		return v.HWND
	}
	return 0
}

// setupWidgetWindow 讓小工具不出現在工作列、套用 Windows 11 圓角，並移到記住的位置（沒有則放右下角）。
func setupWidgetWindow(hwnd uintptr, x, y int, hasPos bool) {
	exStyle := int32(gwlExStyle)
	idx := uintptr(exStyle) // 負數索引轉成 uintptr
	ex, _, _ := procGetWindowLongPtr.Call(hwnd, idx)
	ex = ex&^wsExAppWindow | wsExToolWindow
	procShowWindow.Call(hwnd, swHide)
	procSetWindowLongPtr.Call(hwnd, idx, ex)

	pref := uint32(dwmwcpRound)
	procDwmSetWindowAttr.Call(hwnd, dwmwaCornerPref, uintptr(unsafe.Pointer(&pref)), unsafe.Sizeof(pref))

	if !hasPos {
		var wa, wr rect
		procSystemParamsInfo.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&wa)), 0)
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
		x = int(wa.Right) - int(wr.Right-wr.Left) - widgetMarginPixels
		y = int(wa.Bottom) - int(wr.Bottom-wr.Top) - widgetMarginPixels
	}
	procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged)
	procShowWindow.Call(hwnd, swShowNoActivate)
}

func windowPos(hwnd uintptr) (int, int, bool) {
	if hwnd == 0 {
		return 0, 0, false
	}
	var r rect
	if ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return 0, 0, false
	}
	return int(r.Left), int(r.Top), true
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
