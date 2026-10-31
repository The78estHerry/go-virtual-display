package edid

import (
	"math"
	"testing"
)

// 已知值取自 VESA CVT 1.2 reduced blanking（與 Linux cvt -r 的結果相同）
func TestCVTReducedBlankingKnownValues(t *testing.T) {
	for _, c := range []struct {
		w, h           int
		clockKHz       int
		hTotal, vTotal int
		vSync          int
	}{
		{1920, 1080, 138500, 2080, 1111, 5},
		{1920, 1200, 154000, 2080, 1235, 6},
		{2560, 1440, 241500, 2720, 1481, 5},
		{1280, 1024, 91000, 1440, 1054, 7},
	} {
		tm, err := CVTReducedBlanking(c.w, c.h, 60)
		if err != nil {
			t.Fatal(err)
		}
		if tm.PixelClockKHz != c.clockKHz || tm.HTotal() != c.hTotal || tm.VTotal() != c.vTotal || tm.VSync != c.vSync {
			t.Errorf("%dx%d：clock %d、總 %dx%d、vsync %d，預期 %d、%dx%d、%d",
				c.w, c.h, tm.PixelClockKHz, tm.HTotal(), tm.VTotal(), tm.VSync, c.clockKHz, c.hTotal, c.vTotal, c.vSync)
		}
		if hz := tm.RefreshHz(); math.Abs(hz-60) > 0.1 {
			t.Errorf("%dx%d 更新率 %.3f，預期約 60", c.w, c.h, hz)
		}
	}
}

func TestRangeLimitsOffsetForHighHFreq(t *testing.T) {
	// 直式 1080×1920@144 的水平頻率約 290 kHz，需要 offset 旗標
	tm, err := CVTReducedBlanking(1080, 1920, 144)
	if err != nil {
		t.Fatal(err)
	}
	p := Default
	p.Timing = tm
	e, err := Build(p)
	if err != nil {
		t.Fatal(err)
	}
	rl := e[72:90]
	if rl[4]&0x08 == 0 {
		t.Errorf("水平上限超過 255 kHz 卻沒有設 offset 旗標（byte 4 = %02X）", rl[4])
	}
	_, _, _, maxH := rangeLimits(tm)
	if int(rl[8])+255 != maxH {
		t.Errorf("水平上限 %d+255，預期 %d", rl[8], maxH)
	}
}

func TestBuildRejectsUnrepresentableRange(t *testing.T) {
	tm, err := CVTReducedBlanking(4000, 4000, 240)
	if err != nil {
		t.Fatal(err)
	}
	p := Default
	p.Timing = tm
	if _, err := Build(p); err == nil {
		t.Error("超出範圍的時序沒有被擋下")
	}
}
