package ui

import (
	"bytes"
	"image/png"
	"sync"

	"github.com/The78estHerry/go-desktop-kit/tray"

	"github.com/The78estHerry/go-virtual-display/internal/display"
	"github.com/The78estHerry/go-virtual-display/internal/i18n"
	"github.com/The78estHerry/go-virtual-display/internal/profile"
)

// trayIcon 回傳系統列圖示的 PNG（Windows 需要的 ICO 由 tray 套件轉換）。
func trayIcon(s Snapshot) []byte { return iconPNG(stateColor(s)) }

func (a *App) trayReady(t *tray.Tray) {
	a.tray = t
	t.SetTitle(i18n.T("virtual_display_switch"))
	t.OnTapped(a.model.Toggle) // 左鍵點一下切換；右鍵開選單

	status := t.AddItem("", nil)
	status.SetEnabled(false)
	// Synergy 狀態：有安裝才顯示；點了不做事。不停用這個項目，停用時 Windows 會把圖示畫成灰色，看不出運作中
	synergy := t.AddItem("Synergy", nil)
	synergy.SetVisible(false)
	t.AddSeparator()
	toggle := t.AddItem(i18n.T("plug_in"), a.model.Toggle)
	var res *resolutionMenu
	if a.model.ProfilesSupported() {
		res = a.newResolutionMenu(t)
	}
	widgetItem := t.AddCheckbox(i18n.T("show_desktop_widget"), a.widget.Visible(), nil)
	widgetItem.OnClick(func() { a.setWidget(!widgetItem.Checked()) })
	top := t.AddCheckbox(i18n.T("keep_widget_on_top"), LoadSettings().TopMost, nil)
	top.OnClick(func() {
		on := !top.Checked()
		SaveSettings(func(s *Settings) { s.TopMost = on })
		a.widget.SetTopMost(on)
		top.SetChecked(on)
	})
	auto := t.AddCheckbox(i18n.T("start_at_sign_in"), AutostartEnabled(), nil)
	auto.OnClick(func() {
		on := !auto.Checked()
		if err := SetAutostart(on); err == nil {
			auto.SetChecked(on)
		}
	})
	hotkeyItem := t.AddItem("", func() {
		if a.hkDialog != nil {
			a.hkDialog.Show()
		}
	})
	hotkeyItem.SetTooltip(i18n.T("moves_the_widget_next_to_the"))
	updateHotkey := func() {
		h, err := a.hotkey.Status()
		title := i18n.T("widget_hotkey_prefix") + h.String()
		if err != nil {
			title += i18n.T("unavailable")
		}
		hotkeyItem.SetTitle(title + "…")
	}
	if a.hotkey != nil {
		updateHotkey()
		a.onHotkey = updateHotkey
	} else {
		hotkeyItem.SetVisible(false)
	}
	t.AddSeparator()
	uninstall := t.AddItem(i18n.T("uninstall"), func() { startUninstall() }) // 移除腳本會關掉這個程式
	if !uninstallSupported() {
		uninstall.SetVisible(false)
	}
	t.AddItem(i18n.T("exit"), a.quit)

	a.onWidget = widgetItem.SetChecked

	var synShown, synMajorShown = display.SynergyNone, 0 // 上次顯示的狀態，只在改變時重設圖示和文字
	updateSynergy := func(s Snapshot) {
		if s.Synergy == synShown && s.SynergyMajor == synMajorShown {
			return
		}
		synShown, synMajorShown = s.Synergy, s.SynergyMajor
		if s.Synergy == display.SynergyNone {
			synergy.SetVisible(false)
			return
		}
		running := s.Synergy == display.SynergyRunning
		synergy.SetTitle(synergyName(s) + i18n.T("sep_colon") + synergyText(running))
		if img := synergyIcon(s.SynergyDir, smallIconPx(), running); img != nil {
			var buf bytes.Buffer
			if png.Encode(&buf, img) == nil {
				synergy.SetIcon(buf.Bytes())
			}
		}
		synergy.SetVisible(true)
	}

	a.model.Subscribe(func(s Snapshot) {
		t.SetIcon(trayIcon(s))
		text := i18n.T("virtual_display") + s.Profiles.ActiveProfile().Size() + i18n.T("sep_colon") + statusText(s)
		t.SetTooltip(text)
		if res != nil {
			res.update(s)
		}
		status.SetTitle(text)
		updateSynergy(s)
		switch {
		case s.Busy || s.State == display.Unknown:
			toggle.SetEnabled(false)
		case s.State == display.Plugged:
			toggle.SetTitle(i18n.T("unplug"))
			toggle.SetEnabled(true)
		default:
			toggle.SetTitle(i18n.T("plug_in"))
			toggle.SetEnabled(true)
		}
	})
}

// resolutionMenu 是系統列的「解析度」子選單：最上面是「管理解析度…」，分隔線下面列出存下的解析度，
// 使用中的打勾，點一下就切換。清單改變時移除舊項目重建（分隔線無法移除，所以放在固定項目之後）。
type resolutionMenu struct {
	app    *App
	parent *tray.Item

	mu    sync.Mutex
	key   string // 上次建立時的清單與使用中狀態
	busy  bool
	items []*tray.Item
}

func (a *App) newResolutionMenu(t *tray.Tray) *resolutionMenu {
	parent := t.AddItem(i18n.T("resolution"), nil)
	parent.AddItem(i18n.T("manage_resolutions_menu"), a.manager.Show)
	parent.AddSeparator()
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
			it.SetEnabled(!s.Busy)
		}
		r.busy = s.Busy
	}
}

func (r *resolutionMenu) rebuild(l profile.List) {
	for _, it := range r.items {
		it.Remove()
	}
	r.items = r.items[:0]
	r.busy = false
	for _, p := range l.Profiles {
		r.items = append(r.items, r.parent.AddCheckbox(p.Label(), p.Key() == l.Active, func() { r.app.model.UseProfile(p) }))
	}
}
