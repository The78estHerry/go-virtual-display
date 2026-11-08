package ui

import (
	"os"
	"sync"
	"unsafe"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"
	"golang.org/x/sys/windows"

	"github.com/The78estHerry/go-virtual-display/internal/i18n"
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
	procMonitorFromRect  = user32.NewProc("MonitorFromRect")
	procGetMonitorInfo   = user32.NewProc("GetMonitorInfoW")
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
	monitorDefaultNull = 0
)

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

// setupWidgetWindow 讓小工具不出現在工作列、套用 Windows 11 圓角，並移到記住的位置（沒有則放右下角）。
// 記住的位置所在的螢幕已經拔掉時，由 placer 暫時移到主螢幕。
func setupWidgetWindow(hwnd uintptr, x, y int, hasPos bool, placer *widgetPlacer) {
	exStyle := int32(gwlExStyle)
	idx := uintptr(exStyle) // 負數索引轉成 uintptr
	ex, _, _ := procGetWindowLongPtr.Call(hwnd, idx)
	ex = ex&^wsExAppWindow | wsExToolWindow
	procShowWindow.Call(hwnd, swHide)
	procSetWindowLongPtr.Call(hwnd, idx, ex)

	pref := uint32(dwmwcpRound)
	procDwmSetWindowAttr.Call(hwnd, dwmwaCornerPref, uintptr(unsafe.Pointer(&pref)), unsafe.Sizeof(pref))

	if !hasPos {
		x, y = defaultPos(hwnd)
	}
	procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged)
	placer.reset() // 新的視窗：從記住的位置重新判斷
	placer.check(hwnd)
	procShowWindow.Call(hwnd, swShowNoActivate)
}

// defaultPos 是主螢幕工作區（扣掉工作列）的右下角。
func defaultPos(hwnd uintptr) (int, int) {
	var wa, wr rect
	procSystemParamsInfo.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&wa)), 0)
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
	return int(wa.Right) - int(wr.Right-wr.Left) - widgetMarginPixels,
		int(wa.Bottom) - int(wr.Bottom-wr.Top) - widgetMarginPixels
}

func moveWindow(hwnd uintptr, x, y int) {
	procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpNoZOrder|swpNoActivate)
}

// visibleEnough 判斷視窗 r 是否至少一半落在某個螢幕的工作區內。
func visibleEnough(r rect) bool {
	mon, _, _ := procMonitorFromRect.Call(uintptr(unsafe.Pointer(&r)), monitorDefaultNull)
	if mon == 0 {
		return false
	}
	mi := monitorInfo{CbSize: uint32(unsafe.Sizeof(monitorInfo{}))}
	if ok, _, _ := procGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi))); ok == 0 {
		return false
	}
	w := int64(min(r.Right, mi.Work.Right) - max(r.Left, mi.Work.Left))
	h := int64(min(r.Bottom, mi.Work.Bottom) - max(r.Top, mi.Work.Top))
	if w <= 0 || h <= 0 {
		return false
	}
	return 2*w*h >= int64(r.Right-r.Left)*int64(r.Bottom-r.Top)
}

// widgetPlacer 在小工具所在的螢幕被拔掉時，把小工具暫時移到主螢幕右下角；
// 那個螢幕回來時再移回原位。暫時移開期間使用者自己拖曳的話，就改用新的位置。
type widgetPlacer struct {
	mu                   sync.Mutex
	displaced            bool
	preferX, preferY     int // 原本的位置
	fallbackX, fallbackY int // 暫時移到的位置
}

func (p *widgetPlacer) reset() {
	p.mu.Lock()
	p.displaced = false
	p.mu.Unlock()
}

// check 依目前的螢幕配置調整小工具位置，定期呼叫。
func (p *widgetPlacer) check(hwnd uintptr) {
	var r rect
	if hwnd == 0 {
		return
	}
	if ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	x, y := int(r.Left), int(r.Top)
	if !p.displaced {
		if visibleEnough(r) {
			return
		}
		p.preferX, p.preferY = x, y
		p.fallbackX, p.fallbackY = defaultPos(hwnd)
		p.displaced = true
		moveWindow(hwnd, p.fallbackX, p.fallbackY)
		return
	}
	if x != p.fallbackX || y != p.fallbackY {
		p.displaced = false // 使用者拖曳過，以後用新的位置
		return
	}
	dx, dy := int32(p.preferX-x), int32(p.preferY-y)
	want := rect{r.Left + dx, r.Top + dy, r.Right + dx, r.Bottom + dy}
	if visibleEnough(want) {
		p.displaced = false
		moveWindow(hwnd, p.preferX, p.preferY)
	}
}

// savedPos 回傳要存下的位置：暫時移開時存原本的位置，下次開啟時那個螢幕在的話就放回去。
func (p *widgetPlacer) savedPos(hwnd uintptr) (int, int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.displaced {
		return p.preferX, p.preferY, true
	}
	return windowPos(hwnd)
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

var (
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procMonitorFromPoint = user32.NewProc("MonitorFromPoint")
)

const monitorDefaultNearest = 2

// cursorOffsetPixels 是小工具和滑鼠游標之間的距離
const cursorOffsetPixels = 16

// cursorWidgetPos 回傳把小工具放在滑鼠游標旁邊（右下方）時的位置；
// 右邊或下面放不下就改放左邊或上面，最後再限制在那個螢幕的工作區內。
func cursorWidgetPos(hwnd uintptr) (int, int, bool) {
	var pt struct{ X, Y int32 }
	if ok, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt))); ok == 0 {
		return 0, 0, false
	}
	// MonitorFromPoint 的 POINT 以值傳遞：x64 上兩個 int32 合成一個 64 位元參數
	mon, _, _ := procMonitorFromPoint.Call(uintptr(uint32(pt.X))|uintptr(uint32(pt.Y))<<32, monitorDefaultNearest)
	mi := monitorInfo{CbSize: uint32(unsafe.Sizeof(monitorInfo{}))}
	if ok, _, _ := procGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi))); ok == 0 {
		return 0, 0, false
	}
	var wr rect
	if ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr))); ok == 0 {
		return 0, 0, false
	}
	x, y := besideCursor(int(pt.X), int(pt.Y), int(wr.Right-wr.Left), int(wr.Bottom-wr.Top), mi.Work)
	return x, y, true
}
