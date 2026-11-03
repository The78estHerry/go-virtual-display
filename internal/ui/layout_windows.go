package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/The78estHerry/go-virtual-display/internal/display"
)

const (
	layoutPoll   = time.Second
	layoutSettle = 2 * time.Second // 拖曳排列、插入螢幕時配置會連續變好幾次，穩定後才處理
)

var (
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")

	layoutMu  sync.Mutex
	layoutBuf strings.Builder
	// layoutCallback 只建立一次：windows.NewCallback 建立的數量有上限
	layoutCallback = windows.NewCallback(func(hmon, hdc, lprc, lparam uintptr) uintptr {
		mi := monitorInfo{CbSize: uint32(unsafe.Sizeof(monitorInfo{}))}
		if ok, _, _ := procGetMonitorInfo.Call(hmon, uintptr(unsafe.Pointer(&mi))); ok != 0 {
			r := mi.Monitor
			fmt.Fprintf(&layoutBuf, "%d,%d,%d,%d;", r.Left, r.Top, r.Right, r.Bottom)
		}
		return 1
	})
)

// displayLayout 回傳所有螢幕位置組成的字串，用來判斷螢幕配置是否改變。
func displayLayout() string {
	layoutMu.Lock()
	defer layoutMu.Unlock()
	layoutBuf.Reset()
	procEnumDisplayMonitors.Call(0, 0, layoutCallback, 0)
	return layoutBuf.String()
}

// watchDisplayLayout 在螢幕配置改變（拔插、切換解析度、在 Windows 設定裡調整排列）並穩定後，
// 重新啟動 Synergy 等服務。Synergy 子機只在啟動時讀取桌面大小，不重新啟動的話滑鼠會卡在舊的邊緣。
// 沒有對應的排程工作（沒裝這些服務）時不監看。
func watchDisplayLayout() {
	if !display.HasServiceRestart() {
		return
	}
	go func() {
		handled := displayLayout() // 啟動時的配置只記錄
		cur, changedAt := handled, time.Time{}
		for range time.Tick(layoutPoll) {
			if l := displayLayout(); l != cur {
				cur, changedAt = l, time.Now()
				continue
			}
			if cur != handled && time.Since(changedAt) >= layoutSettle {
				handled = cur
				// 使用者自己關掉 Synergy 時（服務還在、synergy-core.exe 沒在執行）不要重新啟動，
				// 否則會把使用者關掉的 Synergy 又帶起來。沒裝 Synergy 時照樣執行，處理 $RestartServices 裡的其他服務
				if st, _ := display.Synergy(); st != display.SynergyStopped {
					_ = display.RestartDependentServices() // 失敗也不影響開關，下次改變時會再試
				}
			}
		}
	}()
}
