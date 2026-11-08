package ui

import (
	"image/color"
	"sync"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/The78estHerry/go-virtual-display/internal/i18n"
)

// hotkeyDialog 是「設定快捷鍵」視窗：按下新的組合鍵後儲存。只開一個。
// 開著的時候暫停全域快捷鍵，否則按目前的組合鍵會被系統攔走，視窗收不到。
type hotkeyDialog struct {
	hk        *hotkeyMgr
	theme     *material.Theme
	onChanged func() // 儲存成功後（更新系統列選單）

	mu  sync.Mutex
	win *app.Window
}

func newHotkeyDialog(hk *hotkeyMgr, onChanged func()) *hotkeyDialog {
	th := material.NewTheme()
	th.Shaper = newShaper()
	return &hotkeyDialog{hk: hk, theme: th, onChanged: onChanged}
}

// Show 開啟視窗；已開啟時叫到前面。
func (d *hotkeyDialog) Show() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.win != nil {
		d.win.Perform(system.ActionRaise)
		return
	}
	win := new(app.Window)
	win.Option(
		app.Title(i18n.T("Widget hotkey", "小工具快捷鍵")),
		app.Size(unit.Dp(440), unit.Dp(300)),
		app.MinSize(unit.Dp(400), unit.Dp(280)),
	)
	d.win = win
	d.hk.Suspend()
	go d.loop(win)
}

type hotkeyDialogState struct {
	captured             Hotkey
	save, reset, cancel  widget.Clickable
	message              string
	messageIsError, done bool
}

const hotkeyModsAll = key.ModCtrl | key.ModShift | key.ModAlt | key.ModSuper

func (d *hotkeyDialog) loop(win *app.Window) {
	var ops op.Ops
	cur, _ := d.hk.Status()
	st := &hotkeyDialogState{captured: cur}
	for {
		switch e := win.Event().(type) {
		case app.DestroyEvent:
			d.hk.Resume()
			d.mu.Lock()
			d.win = nil
			d.mu.Unlock()
			return
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			d.handle(gtx, st)
			if st.done {
				st.done = false
				win.Perform(system.ActionClose)
			}
			d.layout(gtx, st)
			e.Frame(gtx.Ops)
		}
	}
}

func (d *hotkeyDialog) handle(gtx layout.Context, st *hotkeyDialogState) {
	// 名稱留空的 Filter 收得到所有沒有被其他 Filter 處理的按鍵
	for {
		ev, ok := gtx.Event(key.Filter{Focus: st, Optional: hotkeyModsAll})
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		switch e.Name {
		case key.NameCtrl, key.NameShift, key.NameAlt, key.NameSuper:
			continue // 只按修飾鍵時等主鍵
		case key.NameEscape:
			st.done = true
			return
		}
		h := Hotkey{Mods: e.Modifiers & hotkeyModsAll, Key: string(e.Name)}
		if err := h.Validate(); err != nil {
			st.message, st.messageIsError = h.String()+i18n.T(": ", "：")+err.Error(), true
			continue
		}
		st.captured, st.message, st.messageIsError = h, "", false
	}
	if st.reset.Clicked(gtx) {
		st.captured, _ = ParseHotkey(DefaultHotkey)
		st.message = ""
	}
	if st.cancel.Clicked(gtx) {
		st.done = true
	}
	if st.save.Clicked(gtx) {
		h := st.captured
		if err := d.hk.Set(h); err != nil {
			d.hk.Suspend() // Set 失敗時恢復了原本的註冊，視窗還開著所以再暫停
			st.message, st.messageIsError = err.Error(), true
			return
		}
		SaveSettings(func(s *Settings) {
			s.Hotkey = h.String()
			if s.Hotkey == DefaultHotkey {
				s.Hotkey = ""
			}
		})
		if d.onChanged != nil {
			d.onChanged()
		}
		st.done = true
	}
}

func (d *hotkeyDialog) layout(gtx layout.Context, st *hotkeyDialogState) layout.Dimensions {
	paint.Fill(gtx.Ops, colorPage)
	// 整個視窗都是按鍵的接收區，並一直保持焦點
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	event.Op(gtx.Ops, st)
	area.Pop()
	gtx.Execute(key.FocusCmd{Tag: st})

	label := func(size unit.Sp, txt string, c color.NRGBA) layout.FlexChild {
		return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.Label(d.theme, size, txt)
			l.Color = c
			return l.Layout(gtx)
		})
	}
	gap := func(h unit.Dp) layout.FlexChild { return layout.Rigid(layout.Spacer{Height: h}.Layout) }
	msgColor := colorMuted
	if st.messageIsError {
		msgColor = colorError
	}
	return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			label(14, i18n.T(
				"Moves the desktop widget to the screen under the mouse. Useful when the virtual display is the main display and the switch is on a screen you cannot see.",
				"把桌面小工具移到滑鼠所在的螢幕。虛擬螢幕是主螢幕、開關在看不到的螢幕上時使用。"), colorText),
			gap(12),
			label(13, i18n.T(
				"Press the new combination (Ctrl, Alt or Win plus a letter, digit or F1–F12). Esc closes.",
				"請按下新的組合鍵（Ctrl、Alt 或 Win，加上英文字母、數字或 F1～F12）。按 Esc 關閉。"), colorMuted),
			gap(16),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(d.theme, unit.Sp(22), st.captured.String())
				l.Color = colorText
				return layout.Center.Layout(gtx, l.Layout)
			}),
			gap(8),
			label(12, st.message, msgColor),
			layout.Flexed(1, layout.Spacer{}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Spacing: layout.SpaceStart}.Layout(gtx,
					layout.Rigid(material.Button(d.theme, &st.reset, i18n.T("Default", "預設值")).Layout),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(material.Button(d.theme, &st.cancel, i18n.T("Cancel", "取消")).Layout),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(material.Button(d.theme, &st.save, i18n.T("Save", "儲存")).Layout),
				)
			}),
		)
	})
}
