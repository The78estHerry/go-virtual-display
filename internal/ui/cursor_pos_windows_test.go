package ui

import "testing"

func TestBesideCursor(t *testing.T) {
	work := rect{Left: 0, Top: 0, Right: 1920, Bottom: 1040}
	const w, h = 300, 80
	o := cursorOffsetPixels
	cases := []struct {
		name         string
		cx, cy       int
		wantX, wantY int
	}{
		{"中間：右下方", 500, 400, 500 + o, 400 + o},
		{"靠右：改到左邊", 1900, 400, 1900 - o - w, 400 + o},
		{"靠下：改到上面", 500, 1030, 500 + o, 1030 - o - h},
		{"右下角：左上方", 1910, 1030, 1910 - o - w, 1030 - o - h},
		{"左上角：右下方", 0, 0, o, o},
	}
	for _, c := range cases {
		x, y := besideCursor(c.cx, c.cy, w, h, work)
		if x != c.wantX || y != c.wantY {
			t.Errorf("%s：得到 (%d,%d)，要 (%d,%d)", c.name, x, y, c.wantX, c.wantY)
		}
	}
	// 另一個螢幕（座標為負）也要落在工作區內
	neg := rect{Left: -2160, Top: 0, Right: 0, Bottom: 1320}
	if x, y := besideCursor(-10, 1310, w, h, neg); x < -2160 || x+w > 0 || y < 0 || y+h > 1320 {
		t.Errorf("負座標螢幕超出工作區：(%d,%d)", x, y)
	}
}
