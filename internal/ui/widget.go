package ui

import (
	"image"
	"image/color"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/display"
	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/i18n"
)

const (
	widgetH = 64
	animDur = 160 * time.Millisecond
)

// widgetW 英文字比較寬，寬度依介面語言決定
var widgetW = func() float32 {
	if i18n.IsChinese() {
		return 248
	}
	return 288
}()

var (
	colorCard  = color.NRGBA{R: 0x22, G: 0x25, B: 0x2B, A: 0xFF}
	colorTitle = color.NRGBA{R: 0xF2, G: 0xF2, B: 0xF5, A: 0xFF}
	colorSub   = color.NRGBA{R: 0xA8, G: 0xAB, B: 0xB3, A: 0xFF}
	colorKnob  = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

// widgetMgr 管理桌面小工具視窗；隱藏時關閉視窗，顯示時重新建立。
type widgetMgr struct {
	model    *Model
	theme    *material.Theme
	onClosed func() // 視窗被關閉（含 Alt+F4）時呼叫

	mu   sync.Mutex
	win  *app.Window
	hwnd uintptr
}

func newWidgetMgr(m *Model, onClosed func()) *widgetMgr {
	th := material.NewTheme()
	th.Shaper = newShaper()
	w := &widgetMgr{model: m, theme: th, onClosed: onClosed}
	m.Subscribe(func(Snapshot) {
		w.mu.Lock()
		win := w.win
		w.mu.Unlock()
		if win != nil {
			win.Invalidate()
		}
	})
	return w
}

func (w *widgetMgr) Visible() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.win != nil
}

func (w *widgetMgr) Show() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.win != nil {
		if w.hwnd != 0 {
			bringToFront(w.hwnd)
		}
		return
	}
	s := LoadSettings()
	win := new(app.Window)
	win.Option(
		app.Title(i18n.T("Virtual Display Switch", "虛擬螢幕開關")),
		app.Size(unit.Dp(widgetW), unit.Dp(widgetH)),
		app.MinSize(unit.Dp(widgetW), unit.Dp(widgetH)),
		app.MaxSize(unit.Dp(widgetW), unit.Dp(widgetH)),
		app.Decorated(false),
		app.TopMost(s.TopMost),
	)
	w.win = win
	go w.loop(win)
}

func (w *widgetMgr) Hide() {
	w.mu.Lock()
	win := w.win
	w.mu.Unlock()
	if win != nil {
		w.SavePosition()
		win.Perform(system.ActionClose)
	}
}

func (w *widgetMgr) SetTopMost(on bool) {
	w.mu.Lock()
	win := w.win
	w.mu.Unlock()
	if win != nil {
		win.Option(app.TopMost(on))
	}
}

// SavePosition 記住目前視窗位置（只在能取得位置的平台有效）。
func (w *widgetMgr) SavePosition() {
	w.mu.Lock()
	hwnd := w.hwnd
	w.mu.Unlock()
	if x, y, ok := windowPos(hwnd); ok {
		SaveSettings(func(s *Settings) { s.X, s.Y, s.HasPos = x, y, true })
	}
}

func (w *widgetMgr) loop(win *app.Window) {
	var (
		ops    op.Ops
		toggle widget.Clickable
		knob   float32 // 0 = 左（拔掉），1 = 右（插入）
		last   time.Time
		inited bool
	)
	for {
		switch e := win.Event().(type) {
		case app.DestroyEvent:
			w.mu.Lock()
			if w.win == win {
				w.win, w.hwnd = nil, 0
			}
			w.mu.Unlock()
			w.onClosed()
			return
		case app.ViewEvent:
			if h := viewHWND(e); h != 0 {
				w.mu.Lock()
				w.hwnd = h
				w.mu.Unlock()
				s := LoadSettings()
				// 必須另開 goroutine：視窗執行緒正等著這個迴圈處理完事件，
				// 在這裡同步呼叫會送訊息給該視窗的 Win32 函式會互相等待而卡死
				go setupWidgetWindow(h, s.X, s.Y, s.HasPos)
			}
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			snap := w.model.Snapshot()
			if toggle.Clicked(gtx) {
				w.model.Toggle()
				snap = w.model.Snapshot()
			}
			var target float32
			if snap.State == display.Plugged {
				target = 1
			}
			if !inited {
				knob, inited = target, true
			} else if knob != target {
				step := float32(gtx.Now.Sub(last)) / float32(animDur)
				if knob < target {
					knob = min(target, knob+step)
				} else {
					knob = max(target, knob-step)
				}
			}
			last = gtx.Now
			if knob != target {
				gtx.Execute(op.InvalidateCmd{})
			}
			w.layout(gtx, snap, &toggle, knob)
			e.Frame(gtx.Ops)
		}
	}
}

func (w *widgetMgr) layout(gtx layout.Context, snap Snapshot, toggle *widget.Clickable, knob float32) layout.Dimensions {
	size := gtx.Constraints.Max
	paint.FillShape(gtx.Ops, colorCard, clip.UniformRRect(image.Rectangle{Max: size}, gtx.Dp(10)).Op(gtx.Ops))

	return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			// 左側文字區：拖曳這裡可以移動視窗
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min = gtx.Constraints.Max
				area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
				system.ActionInputOp(system.ActionMove).Add(gtx.Ops)
				area.Pop()
				return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceAround}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(w.theme, unit.Sp(14), i18n.T("Virtual display ", "虛擬螢幕 ")+snap.Profiles.ActiveProfile().Size())
						l.Color = colorTitle
						l.MaxLines = 1
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(w.theme, unit.Sp(12), statusText(snap))
						l.Color = stateColor(snap)
						if snap.Err == nil && snap.State == display.Unplugged && !snap.Busy {
							l.Color = colorSub
						}
						l.MaxLines = 1
						return l.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return toggle.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return drawSwitch(gtx, snap, knob)
				})
			}),
		)
	})
}

func statusText(s Snapshot) string {
	if s.RebootPending && !s.Busy && s.Err == nil {
		return i18n.T("Reboot to apply", "重新開機後套用")
	}
	switch {
	case s.Applying:
		return i18n.T("Applying…", "套用中…")
	case s.Busy:
		return i18n.T("Switching…", "切換中…")
	case s.Err != nil:
		return s.Err.Error()
	case s.State == display.Plugged:
		return i18n.T("Plugged in", "已插入")
	case s.State == display.Unplugged:
		return i18n.T("Unplugged", "已拔掉")
	}
	return i18n.T("Unknown state", "狀態未知")
}

// drawSwitch 畫出滑動開關；knob 介於 0（左）與 1（右）之間。
func drawSwitch(gtx layout.Context, snap Snapshot, knob float32) layout.Dimensions {
	tw, th := gtx.Dp(52), gtx.Dp(30)
	track := lerpColor(colorOff, colorOn, knob)
	if snap.Busy {
		track = colorBusy
	} else if snap.State == display.Unknown {
		track = colorUnknown
	}
	paint.FillShape(gtx.Ops, track, clip.UniformRRect(image.Rect(0, 0, tw, th), th/2).Op(gtx.Ops))

	pad := gtx.Dp(3)
	d := th - 2*pad
	x := pad + int(knob*float32(tw-2*pad-d))
	paint.FillShape(gtx.Ops, colorKnob, clip.Ellipse(image.Rect(x, pad, x+d, pad+d)).Op(gtx.Ops))

	area := clip.Rect{Max: image.Pt(tw, th)}.Push(gtx.Ops)
	pointer.CursorPointer.Add(gtx.Ops)
	area.Pop()
	return layout.Dimensions{Size: image.Pt(tw, th)}
}

func lerpColor(a, b color.NRGBA, t float32) color.NRGBA {
	l := func(x, y uint8) uint8 { return uint8(float32(x) + (float32(y)-float32(x))*t) }
	return color.NRGBA{R: l(a.R, b.R), G: l(a.G, b.G), B: l(a.B, b.B), A: 0xFF}
}
