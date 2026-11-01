package ui

import (
	"bytes"
	"image/png"
	"runtime"
	"sync"

	"fyne.io/systray"

	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/display"
	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/i18n"
	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/profile"
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
	// Synergy 狀態：有安裝才顯示。不停用這個項目，停用時 Windows 會把圖示畫成灰色，看不出運作中
	mSynergy := systray.AddMenuItem("Synergy", "")
	mSynergy.Hide()
	systray.AddSeparator()
	mToggle := systray.AddMenuItem(i18n.T("Plug in", "插入"), "")
	var res *resolutionMenu
	if a.model.ProfilesSupported() {
		res = a.newResolutionMenu()
	}
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

	var synShown, synMajorShown = display.SynergyNone, 0 // 上次顯示的狀態，只在改變時重設圖示和文字
	updateSynergy := func(s Snapshot) {
		if s.Synergy == synShown && s.SynergyMajor == synMajorShown {
			return
		}
		synShown, synMajorShown = s.Synergy, s.SynergyMajor
		if s.Synergy == display.SynergyNone {
			mSynergy.Hide()
			return
		}
		running := s.Synergy == display.SynergyRunning
		if running {
			mSynergy.SetTitle(synergyName(s) + i18n.T(": ", "：") + synergyText(true))
		} else {
			mSynergy.SetTitle(synergyName(s) + i18n.T(": ", "：") + synergyText(false))
		}
		px := smallIconPx()
		if img := synergyIcon(s.SynergyDir, px, running); img != nil {
			var buf bytes.Buffer
			if png.Encode(&buf, img) == nil {
				mSynergy.SetIcon(pngToICO(buf.Bytes(), px))
			}
		}
		mSynergy.Show()
	}

	a.model.Subscribe(func(s Snapshot) {
		systray.SetIcon(trayIcon(s))
		text := i18n.T("Virtual display ", "虛擬螢幕 ") + s.Profiles.ActiveProfile().Size() + i18n.T(": ", "：") + statusText(s)
		systray.SetTooltip(text)
		if res != nil {
			res.update(s)
		}
		mStatus.SetTitle(text)
		updateSynergy(s)
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
		case <-mSynergy.ClickedCh:
			// 只是顯示狀態，點了不做事
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

// resolutionMenu 是系統列的「解析度」子選單：最上面是「管理解析度…」，分隔線下面列出存下的解析度，
// 使用中的打勾，點一下就切換。清單改變時移除舊項目重建（分隔線無法移除，所以放在固定項目之後）。
type resolutionMenu struct {
	app    *App
	parent *systray.MenuItem

	mu    sync.Mutex
	key   string // 上次建立時的清單與使用中狀態
	busy  bool
	items []*systray.MenuItem
	stop  chan struct{}
}

func (a *App) newResolutionMenu() *resolutionMenu {
	parent := systray.AddMenuItem(i18n.T("Resolution", "解析度"), "")
	manage := parent.AddSubMenuItem(i18n.T("Manage resolutions…", "管理解析度…"), "")
	parent.AddSeparator()
	go func() {
		for range manage.ClickedCh {
			a.manager.Show()
		}
	}()
	return &resolutionMenu{app: a, parent: parent}
}

func (r *resolutionMenu) update(s Snapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if key := profilesKey(s.Profiles); key != r.key {
		r.rebuild(s.Profiles)
		r.key = key
	}
	if s.Busy != r.busy {
		for _, it := range r.items {
			if s.Busy {
				it.Disable()
			} else {
				it.Enable()
			}
		}
		r.busy = s.Busy
	}
}

func (r *resolutionMenu) rebuild(l profile.List) {
	if r.stop != nil {
		close(r.stop)
	}
	for _, it := range r.items {
		it.Remove()
	}
	r.items = r.items[:0]
	r.stop = make(chan struct{})
	r.busy = false
	for _, p := range l.Profiles {
		it := r.parent.AddSubMenuItemCheckbox(p.Label(), "", p.Key() == l.Active)
		r.items = append(r.items, it)
		go func(p profile.Profile, it *systray.MenuItem, stop chan struct{}) {
			for {
				select {
				case <-it.ClickedCh:
					r.app.model.UseProfile(p)
				case <-stop:
					return
				}
			}
		}(p, it, r.stop)
	}
}
