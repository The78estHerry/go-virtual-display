package ui

import (
	"fmt"
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

// hotkeyDialog 是「設定快捷鍵」視窗：在視窗裡按下新的組合鍵就立刻套用並儲存，不用再按按鈕。只開一個。
// 開著的時候暫停全域快捷鍵，否則按目前的組合鍵會被系統攔走，視窗收不到。
type hotkeyDialog struct {
	hk        *hotkeyMgr
	theme     *material.Theme
	onChanged func() // 儲存成功後（更新系統列選單）

	mu      sync.Mutex
	win     *app.Window
	applied Hotkey // 最後一次在視窗裡套用成功的（測試用）
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
	reset, close         widget.Clickable
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
		case app.ViewEvent:
			// 從系統列開啟時視窗不一定拿到鍵盤焦點，按鍵會送到別的視窗。
			// 另開 goroutine：不能在事件迴圈裡同步呼叫會送訊息給這個視窗的 Win32 函式
			if h := viewHWND(e); h != 0 {
				go bringToFront(h)
			}
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
	// Gio 只會把焦點交給登記過 FocusFilter 的對象；少了這段 FocusCmd 不會生效，視窗收不到按鍵
	for {
		if _, ok := gtx.Event(key.FocusFilter{Target: st}); !ok {
			break
		}
	}
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
		d.apply(st, h)
	}
	if st.reset.Clicked(gtx) {
		h, _ := ParseHotkey(DefaultHotkey)
		d.apply(st, h)
	}
	if st.close.Clicked(gtx) {
		st.done = true
	}
}

// apply 立刻改用 h 並儲存。視窗還開著，所以成功或失敗後都再暫停全域快捷鍵，繼續接收按鍵。
func (d *hotkeyDialog) apply(st *hotkeyDialogState, h Hotkey) {
	err := d.hk.Set(h) // 失敗時會恢復原本的註冊
	d.hk.Suspend()
	if err != nil {
		st.message, st.messageIsError = err.Error(), true
		return
	}
	SaveSettings(func(s *Settings) {
		s.Hotkey = h.String()
		if s.Hotkey == DefaultHotkey {
			s.Hotkey = ""
		}
	})
	st.captured = h
	st.message, st.messageIsError = fmt.Sprintf(i18n.T("Saved: %s. You can close this window.", "已改成 %s，可以關閉這個視窗。"), h), false
	d.mu.Lock()
	d.applied = h
	d.mu.Unlock()
	if d.onChanged != nil {
		d.onChanged()
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
				"Moves the desktop widget next to the mouse pointer. Useful when the virtual display is the main display and the switch is on a screen you cannot see.",
				"把桌面小工具移到滑鼠游標旁邊。虛擬螢幕是主螢幕、開關在看不到的螢幕上時使用。"), colorText),
			gap(12),
			label(13, i18n.T(
				"Press the new combination in this window; it is saved right away (Ctrl, Alt or Win plus a letter, digit or F1–F12). Esc closes.",
				"在這個視窗裡直接按下新的組合鍵，會立刻儲存（Ctrl、Alt 或 Win，加上英文字母、數字或 F1～F12）。按 Esc 關閉。"), colorMuted),
			gap(16),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(d.theme, unit.Sp(22), st.captured.String())
				l.Color = colorText
				return layout.Center.Layout(gtx, l.Layout)
			}),
			gap(8),
			label(13, st.message, msgColor),
			layout.Flexed(1, layout.Spacer{}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Spacing: layout.SpaceStart}.Layout(gtx,
					layout.Rigid(material.Button(d.theme, &st.reset, i18n.T("Restore default", "還原預設值")).Layout),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(material.Button(d.theme, &st.close, i18n.T("Close", "關閉")).Layout),
				)
			}),
		)
	})
}
