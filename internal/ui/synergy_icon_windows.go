package ui

import (
	"image"
	"path/filepath"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Synergy 的圖示在執行時從使用者電腦上已安裝的 Synergy 讀取，不把對方的商標圖檔放進本程式或安裝包。

var (
	gdi32                  = windows.NewLazySystemDLL("gdi32.dll")
	procPrivateExtractIcon = user32.NewProc("PrivateExtractIconsW")
	procGetIconInfo        = user32.NewProc("GetIconInfo")
	procDestroyIcon        = user32.NewProc("DestroyIcon")
	procGetDC              = user32.NewProc("GetDC")
	procReleaseDC          = user32.NewProc("ReleaseDC")
	procGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
	procGetDIBits          = gdi32.NewProc("GetDIBits")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
)

const smCxSmIcon = 49

// synergyExes 依序嘗試讀取圖示的執行檔。
var synergyExes = []string{"Synergy.exe", "synergy-tray.exe", "synergy-daemon.exe"}

type iconKey struct {
	dir string
	px  int
}

type iconPair struct{ color, gray image.Image }

var (
	iconMu    sync.Mutex
	iconCache = map[iconKey]iconPair{} // 讀不到也記下來（nil），不要每次都重讀
)

// smallIconPx 是系統列選單圖示的大小（依 DPI 而定）。
func smallIconPx() int {
	n, _, _ := procGetSystemMetrics.Call(smCxSmIcon)
	if n == 0 {
		return 16
	}
	return int(n)
}

// synergyIcon 回傳 px×px 的 Synergy 圖示，active 為 false 時是灰色版。讀不到時回傳 nil。
func synergyIcon(dir string, px int, active bool) image.Image {
	if dir == "" || px <= 0 {
		return nil
	}
	key := iconKey{dir, px}
	iconMu.Lock()
	defer iconMu.Unlock()
	pair, ok := iconCache[key]
	if !ok {
		if c := loadExeIcon(dir, px); c != nil {
			pair = iconPair{c, grayed(c)}
		}
		iconCache[key] = pair
	}
	if active {
		return pair.color
	}
	return pair.gray
}

func loadExeIcon(dir string, px int) *image.NRGBA {
	for _, name := range synergyExes {
		if img := extractIcon(filepath.Join(dir, name), px); img != nil {
			return img
		}
	}
	return nil
}

type iconInfo struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// extractIcon 從執行檔讀出第一個圖示，轉成 px×px 的 NRGBA。
func extractIcon(path string, px int) *image.NRGBA {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil
	}
	var hicon uintptr
	var id uint32
	n, _, _ := procPrivateExtractIcon.Call(uintptr(unsafe.Pointer(p)), 0, uintptr(px), uintptr(px),
		uintptr(unsafe.Pointer(&hicon)), uintptr(unsafe.Pointer(&id)), 1, 0)
	if n == 0 || n == 0xFFFFFFFF || hicon == 0 {
		return nil
	}
	defer procDestroyIcon.Call(hicon)

	var ii iconInfo
	if ok, _, _ := procGetIconInfo.Call(hicon, uintptr(unsafe.Pointer(&ii))); ok == 0 {
		return nil
	}
	defer procDeleteObject.Call(ii.HbmMask)
	if ii.HbmColor == 0 {
		return nil // 單色圖示，不處理
	}
	defer procDeleteObject.Call(ii.HbmColor)

	hdc, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, hdc)
	color := dibits(hdc, ii.HbmColor, px)
	if color == nil {
		return nil
	}
	img := image.NewNRGBA(image.Rect(0, 0, px, px))
	hasAlpha := false
	for i := 0; i < len(color); i += 4 {
		if color[i+3] != 0 {
			hasAlpha = true
			break
		}
	}
	var mask []byte
	if !hasAlpha {
		mask = dibits(hdc, ii.HbmMask, px) // 舊式圖示：遮罩白色的地方是透明
	}
	for i := 0; i < px*px; i++ {
		b, g, r, a := color[4*i], color[4*i+1], color[4*i+2], color[4*i+3]
		if !hasAlpha {
			a = 0xFF
			if mask != nil && mask[4*i] != 0 {
				a = 0
			}
		}
		img.Pix[4*i], img.Pix[4*i+1], img.Pix[4*i+2], img.Pix[4*i+3] = r, g, b, a
	}
	return img
}

// dibits 把點陣圖讀成 32 bpp、由上而下的 BGRA。
func dibits(hdc, hbm uintptr, px int) []byte {
	bi := bitmapInfoHeader{Width: int32(px), Height: -int32(px), Planes: 1, BitCount: 32}
	bi.Size = uint32(unsafe.Sizeof(bi))
	// BITMAPINFO 後面還有色盤，32 bpp 用不到，但多留空間比較保險
	buf := make([]byte, unsafe.Sizeof(bi)+16)
	*(*bitmapInfoHeader)(unsafe.Pointer(&buf[0])) = bi
	out := make([]byte, 4*px*px)
	n, _, _ := procGetDIBits.Call(hdc, hbm, 0, uintptr(px), uintptr(unsafe.Pointer(&out[0])), uintptr(unsafe.Pointer(&buf[0])), 0)
	if n == 0 {
		return nil
	}
	return out
}

// grayed 轉成灰階並變淡，表示 Synergy 沒有在執行。
func grayed(src *image.NRGBA) *image.NRGBA {
	dst := image.NewNRGBA(src.Rect)
	for i := 0; i < len(src.Pix); i += 4 {
		r, g, b, a := src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3]
		y := uint8((299*uint32(r) + 587*uint32(g) + 114*uint32(b)) / 1000)
		dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = y, y, y, uint8(uint32(a)*45/100)
	}
	return dst
}
