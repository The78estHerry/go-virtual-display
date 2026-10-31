package ui

import (
	"runtime"

	"fyne.io/systray"

	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/display"
	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/i18n"
)

func trayIcon(s Snapshot) []byte {
	p := iconPNG(stateColor(s))
	if runtime.GOOS == "windows" {
		return pngToICO(p, 32)
	}
	return p
}

func (a *App) trayReady() {
	systray.SetTitle(i18n.T("Virtual Display Switch", "虛擬螢幕開關"))
	systray.SetOnTapped(a.model.Toggle) // 左鍵點一下切換；右鍵開選單

	mStatus := systray.AddMenuItem("", "")
	mStatus.Disable()
	systray.AddSeparator()
	mToggle := systray.AddMenuItem(i18n.T("Plug in", "插入"), "")
	mWidget := systray.AddMenuItemCheckbox(i18n.T("Show desktop widget", "顯示桌面小工具"), "", a.widget.Visible())
	mTop := systray.AddMenuItemCheckbox(i18n.T("Keep widget on top", "小工具固定在最上層"), "", LoadSettings().TopMost)
	mAuto := systray.AddMenuItemCheckbox(i18n.T("Start at sign-in", "登入時自動啟動"), "", AutostartEnabled())
	systray.AddSeparator()
	mUninstall := systray.AddMenuItem(i18n.T("Uninstall…", "解除安裝…"), "")
	if !uninstallSupported() {
		mUninstall.Hide()
	}
	mQuit := systray.AddMenuItem(i18n.T("Exit", "結束"), "")

	setCheck := func(m *systray.MenuItem, on bool) {
		if on {
			m.Check()
		} else {
			m.Uncheck()
		}
	}
	a.onWidget = func(on bool) { setCheck(mWidget, on) }

	a.model.Subscribe(func(s Snapshot) {
		systray.SetIcon(trayIcon(s))
		text := i18n.T("Virtual display 1920×1440: ", "虛擬螢幕 1920×1440：") + statusText(s)
		systray.SetTooltip(text)
		mStatus.SetTitle(text)
		switch {
		case s.Busy || s.State == display.Unknown:
			mToggle.Disable()
		case s.State == display.Plugged:
			mToggle.SetTitle(i18n.T("Unplug", "拔掉"))
			mToggle.Enable()
		default:
			mToggle.SetTitle(i18n.T("Plug in", "插入"))
			mToggle.Enable()
		}
	})

	for {
		select {
		case <-mToggle.ClickedCh:
			a.model.Toggle()
		case <-mWidget.ClickedCh:
			a.setWidget(!mWidget.Checked())
		case <-mTop.ClickedCh:
			on := !mTop.Checked()
			SaveSettings(func(s *Settings) { s.TopMost = on })
			a.widget.SetTopMost(on)
			setCheck(mTop, on)
		case <-mAuto.ClickedCh:
			on := !mAuto.Checked()
			if err := SetAutostart(on); err == nil {
				setCheck(mAuto, on)
			}
		case <-mUninstall.ClickedCh:
			startUninstall() // 移除腳本會關掉這個程式
		case <-mQuit.ClickedCh:
			a.quit()
			return
		}
	}
}
