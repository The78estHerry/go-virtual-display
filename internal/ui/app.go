package ui

import (
	"os"
	"runtime"

	"fyne.io/systray"
	"gioui.org/app"

	"github.com/The78estHerry/go-virtual-display/internal/display"
)

// App 串起 Model、系統列圖示與桌面小工具。
type App struct {
	model    *Model
	widget   *widgetMgr
	manager  *managerWin
	hotkey   *hotkeyMgr    // 把小工具移到滑鼠所在螢幕的快捷鍵（只有 Windows）
	hkDialog *hotkeyDialog // 設定快捷鍵的視窗
	quitting bool
	onWidget func(bool) // 小工具顯示狀態改變時更新系統列選單
	onHotkey func()     // 快捷鍵改變時更新系統列選單
}

// Run 啟動開關程式，不會返回（結束時直接離開程序）。必須在 main goroutine 呼叫。
// show 收到訊號時顯示小工具（重複啟動程式時），可以是 nil。
func Run(show <-chan struct{}) {
	ctrl, err := display.New()
	a := &App{model: NewModel(ctrl, err)}
	a.widget = newWidgetMgr(a.model, a.widgetClosed)
	a.manager = newManagerWin(a.model)
	if hotkeySupported {
		a.hotkey = startHotkey(hotkeyFromSettings(LoadSettings()), a.summonWidget)
		a.hkDialog = newHotkeyDialog(a.hotkey, func() {
			if a.onHotkey != nil {
				a.onHotkey()
			}
		})
	}

	// 小工具要在系統列之前建立：系統列選單建立時用 a.widget.Visible() 決定「顯示桌面小工具」有沒有勾，
	// 之前順序相反，選單有時先建好，小工具明明顯示著卻沒有勾
	watchDisplayLayout()
	if LoadSettings().ShowWidget {
		a.widget.Show()
	}

	go func() {
		// systray 在 Windows 上需要同一個執行緒建立視窗並處理訊息
		runtime.LockOSThread()
		systray.Run(a.trayReady, nil)
		os.Exit(0)
	}()

	if show != nil {
		go func() {
			for range show {
				a.ShowWidget()
			}
		}()
	}
	app.Main()
}

// summonWidget 是快捷鍵的動作：把小工具移到滑鼠所在的螢幕，隱藏中的話一併顯示。
func (a *App) summonWidget() {
	a.widget.MoveToCursor()
	if !a.widget.Visible() {
		a.setWidget(true)
	}
}

// ShowWidget 顯示小工具（重複啟動程式時也會呼叫）。
func (a *App) ShowWidget() { a.setWidget(true) }

func (a *App) setWidget(on bool) {
	SaveSettings(func(s *Settings) { s.ShowWidget = on })
	if on {
		a.widget.Show()
	} else {
		a.widget.Hide()
	}
	if a.onWidget != nil {
		a.onWidget(on)
	}
}

func (a *App) widgetClosed() {
	if a.quitting {
		return
	}
	// 被 Alt+F4 等方式關閉時，視同使用者隱藏小工具
	SaveSettings(func(s *Settings) { s.ShowWidget = false })
	if a.onWidget != nil {
		a.onWidget(false)
	}
}

func (a *App) quit() {
	a.quitting = true
	a.widget.SavePosition()
	systray.Quit() // systray.Run 返回後離開程序
}
