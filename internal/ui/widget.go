package ui

import (
	"image"
	"image/color"
	"strconv"
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

	"github.com/The78estHerry/go-gio-kit/giowidget"

	"github.com/The78estHerry/go-virtual-display/internal/display"
	"github.com/The78estHerry/go-virtual-display/internal/i18n"
)

const (
	widgetH        = 64 // 沒裝 Synergy 時的高度
	widgetHSynergy = 84 // 多一行 Synergy 狀態
	animDur        = 160 * time.Millisecond
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
// 視窗的無邊框、不出現在工作列、位置記憶與螢幕拔掉時暫時移開，由 go-gio-kit 的 giowidget 處理。
type widgetMgr struct {
	model    *Model
	theme    *material.Theme
	onClosed func() // 視窗被關閉（含 Alt+F4）時呼叫

	mu       sync.Mutex
	gw       *giowidget.Widget
	toCursor bool // 視窗建立後移到滑鼠游標旁邊（快捷鍵叫出隱藏中的小工具時）
}

// savePosDelay：giowidget 在背景移動視窗，等它移完再記住位置。
const savePosDelay = 300 * time.Millisecond

// MoveToCursor 把小工具移到滑鼠游標旁邊並叫到前面，記住新位置。
// 小工具隱藏中時，下次建立視窗時再移（呼叫端接著要 Show）。
func (w *widgetMgr) MoveToCursor() {
	w.mu.Lock()
	gw := w.gw
	if gw == nil {
		w.toCursor = true
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()
	gw.MoveToCursor()
	time.AfterFunc(savePosDelay, w.SavePosition)
}

func newWidgetMgr(m *Model, onClosed func()) *widgetMgr {
	th := material.NewTheme()
	th.Shaper = newShaper()
	w := &widgetMgr{model: m, theme: th, onClosed: onClosed}
	m.Subscribe(func(Snapshot) {
		w.mu.Lock()
		gw := w.gw
		w.mu.Unlock()
		if gw != nil {
			gw.Window().Invalidate()
		}
	})
	return w
}

func (w *widgetMgr) Visible() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.gw != nil
}

func (w *widgetMgr) Show() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.gw != nil {
		w.gw.BringToFront()
		return
	}
	s := LoadSettings()
	opts := giowidget.Options{
		Title:   i18n.T("virtual_display_switch"),
		Width:   unit.Dp(widgetW),
		Height:  unit.Dp(widgetHeight(w.model.Snapshot())),
		TopMost: s.TopMost,
	}
	if s.HasPos {
		opts.Position = &image.Point{X: s.X, Y: s.Y}
	}
	if w.toCursor {
		w.toCursor = false
		opts.NearCursor = true
		time.AfterFunc(savePosDelay*3, w.SavePosition) // 視窗建立並移到游標旁之後
	}
	gw := giowidget.New(opts)
	w.gw = gw
	go w.loop(gw)
}

// widgetHeight 有安裝 Synergy 時多一行 Synergy 狀態。
func widgetHeight(s Snapshot) float32 {
	if s.Synergy == display.SynergyNone {
		return widgetH
	}
	return widgetHSynergy
}

func (w *widgetMgr) Hide() {
	w.mu.Lock()
	gw := w.gw
	w.mu.Unlock()
	if gw != nil {
		w.SavePosition()
		gw.Window().Perform(system.ActionClose)
	}
}

func (w *widgetMgr) SetTopMost(on bool) {
	w.mu.Lock()
	gw := w.gw
	w.mu.Unlock()
	if gw != nil {
		gw.SetTopMost(on)
	}
}

// SavePosition 記住目前視窗位置（只在能取得位置的平台有效）。所在螢幕暫時被拔掉時記住原本的位置。
func (w *widgetMgr) SavePosition() {
	w.mu.Lock()
	gw := w.gw
	w.mu.Unlock()
	if gw == nil {
		return
	}
	if p, ok := gw.Position(); ok {
		SaveSettings(func(s *Settings) { s.X, s.Y, s.HasPos = p.X, p.Y, true })
	}
}

func (w *widgetMgr) loop(gw *giowidget.Widget) {
	var (
		ops    op.Ops
		toggle widget.Clickable
		knob   float32 // 0 = 左（拔掉），1 = 右（插入）
		last   time.Time
		inited bool
		height = widgetHeight(w.model.Snapshot()) // 目前視窗的高度（dp）
	)
	for {
		e := gw.Window().Event()
		gw.HandleEvent(e)
		switch e := e.(type) {
		case app.DestroyEvent:
			w.mu.Lock()
			if w.gw == gw {
				w.gw = nil
			}
			w.mu.Unlock()
			w.onClosed()
			return
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			snap := w.model.Snapshot()
			if h := widgetHeight(snap); h != height {
				// 安裝或移除 Synergy 後改變高度（SetSize 不會在事件迴圈裡等待視窗執行緒）
				height = h
				gw.SetSize(unit.Dp(widgetW), unit.Dp(h))
			}
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
						l := material.Label(w.theme, unit.Sp(14), i18n.T("virtual_display")+snap.Profiles.ActiveProfile().Size())
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
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return w.synergyRow(gtx, snap)
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
		return i18n.T("reboot_to_apply")
	}
	switch {
	case s.Applying:
		return i18n.T("applying")
	case s.Busy:
		return i18n.T("switching")
	case s.Err != nil:
		return s.Err.Error()
	case s.State == display.Plugged:
		return i18n.T("plugged_in")
	case s.State == display.Unplugged:
		return i18n.T("unplugged")
	}
	return i18n.T("unknown_state")
}

// synergyName 是「Synergy 3」這樣的名稱，只顯示主要版號；讀不到版本時只寫 Synergy。
func synergyName(s Snapshot) string {
	if s.SynergyMajor > 0 {
		return "Synergy " + strconv.Itoa(s.SynergyMajor)
	}
	return "Synergy"
}

// synergyText 是 Synergy 狀態的文字：系統列選單和小工具共用。
func synergyText(running bool) string {
	if running {
		return i18n.T("running")
	}
	return i18n.T("not_running")
}

// synergyRow 是狀態下面的一行：Synergy 圖示＋「Synergy 執行中／未執行」，沒裝 Synergy 時不畫。
// 圖示從使用者安裝的 Synergy 讀取，執行中是彩色、未執行是灰色；讀不到時改畫小圓點。
func (w *widgetMgr) synergyRow(gtx layout.Context, snap Snapshot) layout.Dimensions {
	if snap.Synergy == display.SynergyNone {
		return layout.Dimensions{}
	}
	running := snap.Synergy == display.SynergyRunning
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			size := gtx.Dp(14)
			if img := synergyIcon(snap.SynergyDir, size, running); img != nil {
				paint.NewImageOp(img).Add(gtx.Ops)
				defer clip.Rect{Max: image.Pt(size, size)}.Push(gtx.Ops).Pop()
				paint.PaintOp{}.Add(gtx.Ops)
			} else {
				c := colorOff
				if running {
					c = colorOn
				}
				d := gtx.Dp(8)
				o := (size - d) / 2
				paint.FillShape(gtx.Ops, c, clip.Ellipse(image.Rect(o, o, o+d, o+d)).Op(gtx.Ops))
			}
			return layout.Dimensions{Size: image.Pt(size, size)}
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			l := material.Label(w.theme, unit.Sp(12), synergyName(snap)+" "+synergyText(running))
			l.Color = colorSub
			if running {
				l.Color = colorTitle
			}
			l.MaxLines = 1
			return l.Layout(gtx)
		}),
	)
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
