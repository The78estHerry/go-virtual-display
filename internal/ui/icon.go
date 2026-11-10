package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"

	"github.com/The78estHerry/go-virtual-display/internal/display"
)

var (
	colorOn      = color.NRGBA{R: 0x34, G: 0xC7, B: 0x59, A: 0xFF} // 綠：已插入
	colorOff     = color.NRGBA{R: 0x8E, G: 0x8E, B: 0x93, A: 0xFF} // 灰：已拔掉
	colorBusy    = color.NRGBA{R: 0xFF, G: 0x9F, B: 0x0A, A: 0xFF} // 橘：切換中
	colorUnknown = color.NRGBA{R: 0xFF, G: 0x45, B: 0x3A, A: 0xFF} // 紅：錯誤
)

func stateColor(s Snapshot) color.NRGBA {
	switch {
	case s.Busy:
		return colorBusy
	case s.State == display.Plugged:
		return colorOn
	case s.State == display.Unplugged:
		return colorOff
	}
	return colorUnknown
}

// iconPNG 繪製一個螢幕形狀的圖示：外框白色，畫面填入狀態顏色。
func iconPNG(c color.NRGBA) []byte {
	const n = 32
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	white := color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	dark := color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xFF}
	fill := func(x0, y0, x1, y1 int, col color.NRGBA) {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				img.SetNRGBA(x, y, col)
			}
		}
	}
	fill(1, 3, 31, 24, dark)  // 外框陰影
	fill(2, 4, 30, 23, white) // 外框
	fill(4, 6, 28, 21, c)     // 畫面
	fill(13, 23, 19, 27, white)
	fill(8, 27, 24, 29, white) // 底座
	if c == colorOff || c == colorUnknown {
		// 斜線表示「沒有訊號」
		for i := 0; i < 15; i++ {
			x := 9 + i
			y := 7 + int(math.Round(float64(i)*13/14))
			fill(x, y, x+2, y+1, white)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
