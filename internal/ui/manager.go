package ui

import (
	"image"
	"image/color"
	"strconv"
	"sync"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/The78estHerry/go-virtual-display/internal/i18n"
	"github.com/The78estHerry/go-virtual-display/internal/profile"
)

const presetColumns = 3

var (
	colorPage  = color.NRGBA{R: 0xF7, G: 0xF7, B: 0xF9, A: 0xFF}
	colorText  = color.NRGBA{R: 0x22, G: 0x25, B: 0x2B, A: 0xFF}
	colorMuted = color.NRGBA{R: 0x6B, G: 0x6F, B: 0x78, A: 0xFF}
	colorError = color.NRGBA{R: 0xC6, G: 0x28, B: 0x28, A: 0xFF}
)

// managerWin 是「管理解析度」視窗：列出存下的解析度（使用／刪除）、加入預設、手動輸入。只開一個。
type managerWin struct {
	model *Model
	theme *material.Theme

	mu  sync.Mutex
	win *app.Window
}

func newManagerWin(m *Model) *managerWin {
	th := material.NewTheme()
	th.Shaper = newShaper()
	mw := &managerWin{model: m, theme: th}
	m.Subscribe(func(Snapshot) {
		mw.mu.Lock()
		win := mw.win
		mw.mu.Unlock()
		if win != nil {
			win.Invalidate()
		}
	})
	return mw
}

// Show 開啟視窗；已開啟時叫到前面。
func (mw *managerWin) Show() {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	if mw.win != nil {
		mw.win.Perform(system.ActionRaise)
		return
	}
	win := new(app.Window)
	win.Option(
		app.Title(i18n.T("manage_resolutions")),
		app.Size(unit.Dp(460), unit.Dp(640)),
		app.MinSize(unit.Dp(400), unit.Dp(420)),
	)
	mw.win = win
	go mw.loop(win)
}

// managerState 是視窗內的元件狀態。
type managerState struct {
	list           widget.List
	use, del       map[string]*widget.Clickable // 依 profile.Key()
	presets        map[string]*widget.Clickable
	width, height  widget.Editor
	refresh        widget.Editor
	add            widget.Clickable
	message        string
	messageIsError bool
}

func newManagerState() *managerState {
	s := &managerState{
		use: map[string]*widget.Clickable{}, del: map[string]*widget.Clickable{}, presets: map[string]*widget.Clickable{},
	}
	s.list.Axis = layout.Vertical
	for _, e := range []*widget.Editor{&s.width, &s.height, &s.refresh} {
		e.SingleLine, e.Filter, e.MaxLen = true, "0123456789", 4
	}
	s.refresh.SetText("60")
	return s
}

func clickable(m map[string]*widget.Clickable, key string) *widget.Clickable {
	c, ok := m[key]
	if !ok {
		c = new(widget.Clickable)
		m[key] = c
	}
	return c
}

func (mw *managerWin) loop(win *app.Window) {
	var ops op.Ops
	st := newManagerState()
	for {
		switch e := win.Event().(type) {
		case app.DestroyEvent:
			mw.mu.Lock()
			mw.win = nil
			mw.mu.Unlock()
			return
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			mw.handle(gtx, st)
			mw.layout(gtx, st)
			e.Frame(gtx.Ops)
		}
	}
}

func (st *managerState) report(err error, ok string) {
	if err != nil {
		st.message, st.messageIsError = err.Error(), true
	} else {
		st.message, st.messageIsError = ok, false
	}
}

// handle 處理按鈕點擊。
func (mw *managerWin) handle(gtx layout.Context, st *managerState) {
	snap := mw.model.Snapshot()
	for _, p := range snap.Profiles.Profiles {
		if clickable(st.use, p.Key()).Clicked(gtx) {
			mw.model.UseProfile(p)
			st.report(nil, "")
		}
		if clickable(st.del, p.Key()).Clicked(gtx) {
			st.report(mw.model.RemoveProfile(p), i18n.T("deleted")+p.Label())
		}
	}
	for _, p := range profile.Presets {
		if clickable(st.presets, p.Key()).Clicked(gtx) {
			st.report(mw.model.AddProfile(p), i18n.T("added")+p.Label())
		}
	}
	if st.add.Clicked(gtx) {
		atoi := func(e *widget.Editor) int { n, _ := strconv.Atoi(e.Text()); return n }
		p := profile.Profile{Width: atoi(&st.width), Height: atoi(&st.height), Refresh: atoi(&st.refresh)}
		if snap.Profiles.Has(p) {
			st.report(nil, p.Label()+i18n.T("is_already_in_the_list"))
		} else if err := mw.model.AddProfile(p); err != nil {
			st.report(err, "")
		} else {
			st.report(nil, i18n.T("added")+p.Label())
			st.width.SetText("")
			st.height.SetText("")
		}
	}
}

func (mw *managerWin) layout(gtx layout.Context, st *managerState) {
	th := mw.theme
	snap := mw.model.Snapshot()
	paint.Fill(gtx.Ops, colorPage)

	var rows []layout.Widget
	heading := func(text string) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				l := material.Label(th, unit.Sp(16), text)
				l.Color = colorText
				return l.Layout(gtx)
			})
		}
	}

	// 已存的解析度
	rows = append(rows, heading(i18n.T("saved_resolutions")))
	for _, p := range snap.Profiles.Profiles {
		p := p
		active := p.Key() == snap.Profiles.Active
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						text := p.Label()
						if active {
							text += i18n.T("in_use")
						}
						l := material.Body1(th, text)
						l.Color = colorText
						if !active {
							l.Color = colorMuted
						}
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if active || snap.Busy {
							gtx = gtx.Disabled()
						}
						return smallButton(th, clickable(st.use, p.Key()), i18n.T("use")).Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if active {
							gtx = gtx.Disabled()
						}
						b := smallButton(th, clickable(st.del, p.Key()), i18n.T("delete"))
						b.Background = colorError
						return b.Layout(gtx)
					}),
				)
			})
		})
	}
	if snap.Applying {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			l := material.Body2(th, i18n.T("applying_the_virtual_display_restarts_briefly"))
			l.Color = colorMuted
			return l.Layout(gtx)
		})
	}

	// 預設清單中還沒加入的
	var avail []profile.Profile
	for _, p := range profile.Presets {
		if !snap.Profiles.Has(p) {
			avail = append(avail, p)
		}
	}
	rows = append(rows, heading(i18n.T("add_a_common_resolution_60_hz")))
	if len(avail) == 0 {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			l := material.Body2(th, i18n.T("all_common_resolutions_are_already_added"))
			l.Color = colorMuted
			return l.Layout(gtx)
		})
	}
	for i := 0; i < len(avail); i += presetColumns {
		chunk := avail[i:min(i+presetColumns, len(avail))]
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, presetColumns*2)
			for j := 0; j < presetColumns; j++ {
				if j > 0 {
					children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout))
				}
				if j >= len(chunk) {
					children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }))
					continue
				}
				p := chunk[j]
				children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					b := material.Button(th, clickable(st.presets, p.Key()), "+ "+p.Size())
					b.TextSize = unit.Sp(13)
					b.Background = color.NRGBA{R: 0x4A, G: 0x5A, B: 0xC8, A: 0xFF}
					return b.Layout(gtx)
				}))
			}
			return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{}.Layout(gtx, children...)
			})
		})
	}

	// 手動輸入
	rows = append(rows, heading(i18n.T("add_a_custom_resolution")))
	rows = append(rows, func(gtx layout.Context) layout.Dimensions {
		field := func(e *widget.Editor, hint string) layout.FlexChild {
			return layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return editorBox(gtx, th, e, hint)
			})
		}
		sep := func(s string) layout.FlexChild {
			return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, material.Body1(th, s).Layout)
			})
		}
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			field(&st.width, i18n.T("width")),
			sep("×"),
			field(&st.height, i18n.T("height")),
			sep("@"),
			field(&st.refresh, "Hz"),
			sep("Hz"),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return smallButton(th, &st.add, i18n.T("add")).Layout(gtx)
			}),
		)
	})
	rows = append(rows, func(gtx layout.Context) layout.Dimensions {
		l := material.Caption(th, i18n.T("width_height_320_4095_refresh_24"))
		l.Color = colorMuted
		return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, l.Layout)
	})
	if st.message != "" {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			l := material.Body2(th, st.message)
			l.Color = colorMuted
			if st.messageIsError {
				l.Color = colorError
			}
			return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, l.Layout)
		})
	}

	layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.List(th, &st.list).Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
			return rows[i](gtx)
		})
	})
}

func smallButton(th *material.Theme, c *widget.Clickable, text string) material.ButtonStyle {
	b := material.Button(th, c, text)
	b.TextSize = unit.Sp(13)
	b.Inset = layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}
	return b
}

// editorBox 畫出有白底和外框的輸入欄。
func editorBox(gtx layout.Context, th *material.Theme, e *widget.Editor, hint string) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(8)).Layout(gtx, material.Editor(th, e, hint).Layout)
	call := macro.Stop()
	rr := gtx.Dp(4)
	rect := image.Rectangle{Max: dims.Size}
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0xC8, G: 0xCB, B: 0xD2, A: 0xFF}, clip.UniformRRect(rect, rr).Op(gtx.Ops))
	inner := image.Rectangle{Min: image.Pt(1, 1), Max: dims.Size.Sub(image.Pt(1, 1))}
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, clip.UniformRRect(inner, rr).Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}
