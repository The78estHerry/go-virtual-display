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
