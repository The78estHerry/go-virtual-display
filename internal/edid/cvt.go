package edid

import (
	"errors"
	"math"
)

// CVTReducedBlanking 依 VESA CVT 1.2 reduced blanking（v1）計算 w×h@hz 的時序。
// 這是現代 LCD 常用的時序：水平 blanking 固定 160 像素，垂直 blanking 至少 460 µs。
func CVTReducedBlanking(w, h int, hz float64) (Timing, error) {
	const (
		minVBlankUS = 460.0 // RB_MIN_V_BLANK
		vFront      = 3     // RB_V_FPORCH
		minVBack    = 6     // MIN_V_BPORCH
		hFront      = 48
		hSync       = 32 // RB_H_SYNC
		hBack       = 80 // RB_H_BLANK 160 = 48 + 32 + 80
		clockStep   = 0.25
	)
	if w <= 0 || h <= 0 || hz <= 0 {
		return Timing{}, errors.New("寬、高、更新率必須大於 0")
	}
	frameUS := 1e6 / hz
	if frameUS <= minVBlankUS {
		return Timing{}, errors.New("更新率太高")
	}
	vSync := cvtVSync(w, h)
	hPeriodEst := (frameUS - minVBlankUS) / float64(h)
	vbiLines := int(minVBlankUS/hPeriodEst) + 1
	if minVBI := vFront + vSync + minVBack; vbiLines < minVBI {
		vbiLines = minVBI
	}
	totalV := h + vbiLines
	totalH := w + hFront + hSync + hBack
	mhz := clockStep * math.Floor(hz*float64(totalV)*float64(totalH)/1e6/clockStep)
	return Timing{
		PixelClockKHz: int(math.Round(mhz * 1000)),
		HActive:       w, HFront: hFront, HSync: hSync, HBack: hBack,
		VActive: h, VFront: vFront, VSync: vSync, VBack: vbiLines - vFront - vSync,
		HSyncPositive: true, VSyncPositive: false,
	}, nil
}

// cvtVSync 是 CVT 依畫面比例規定的垂直同步寬度；不是標準比例時用 10。
func cvtVSync(w, h int) int {
	switch {
	case w*3 == h*4:
		return 4
	case w*9 == h*16:
		return 5
	case w*10 == h*16:
		return 6
	case w*4 == h*5, w*9 == h*15:
		return 7
	}
	return 10
}

// ExactRefresh 微調 t 的總像素與總行數（加大水平 front porch、垂直 back porch），
// 讓更新率剛好等於整數 hz。DTD 的 pixel clock 以 10 kHz 為單位，所以要找
// 「總像素 × 總行數 × hz」是 10000 倍數的最小組合。
// Virtual Display Driver 以設定檔的整數更新率建立模式，EDID 的首選時序必須完全一致才會建立螢幕。
func ExactRefresh(t Timing, hz int) (Timing, error) {
	if hz <= 0 {
		return t, errors.New("更新率必須大於 0")
	}
	h0, v0 := t.HTotal(), t.VTotal()
	best, bestH, bestV := 0, 0, 0
	for dv := 0; dv <= 200; dv++ {
		for dh := 0; dh <= 200 && t.HFront+dh <= 1023; dh++ {
			h, v := h0+dh, v0+dv
			if h*v*hz%10000 != 0 {
				continue
			}
			if best == 0 || h*v < best {
				best, bestH, bestV = h*v, h, v
			}
			break // 同一個 dv 只需要最小的 dh
		}
	}
	if best == 0 {
		return t, errors.New("找不到更新率剛好是整數的時序")
	}
	t.HFront += bestH - h0
	t.VBack += bestV - v0
	t.PixelClockKHz = bestH * bestV * hz / 1000
	return t, nil
}
