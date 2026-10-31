package ui

import (
	"os"
	"runtime"

	"fyne.io/systray"
	"gioui.org/app"

	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/display"
)

// App 串起 Model、系統列圖示與桌面小工具。
type App struct {
	model    *Model
	widget   *widgetMgr
	manager  *managerWin
	quitting bool
	onWidget func(bool) // 小工具顯示狀態改變時更新系統列選單
}

// Run 啟動開關程式，不會返回（結束時直接離開程序）。必須在 main goroutine 呼叫。
// show 收到訊號時顯示小工具（重複啟動程式時），可以是 nil。
func Run(show <-chan struct{}) {
	ctrl, err := display.New()
	a := &App{model: NewModel(ctrl, err)}
	a.widget = newWidgetMgr(a.model, a.widgetClosed)
	a.manager = newManagerWin(a.model)

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
	if LoadSettings().ShowWidget {
		a.widget.Show()
	}
	app.Main()
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
