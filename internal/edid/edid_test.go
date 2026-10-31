package edid

import (
	"math"
	"testing"
)

func TestBuildDefaultRoundTrip(t *testing.T) {
	e, err := Build(Default)
	if err != nil {
		t.Fatal(err)
	}
	if len(e) != Size {
		t.Fatalf("長度 %d，預期 %d", len(e), Size)
	}
	info, err := Parse(e)
	if err != nil {
		t.Fatal(err)
	}
	if info.Preferred != DMT1920x1440at60 {
		t.Errorf("解析回來的時序 %+v，預期 %+v", info.Preferred, DMT1920x1440at60)
	}
	if info.Manufacturer != "VDS" || info.Name != "VirtualDisp" || info.SerialText != "GVD-1920x1440" {
		t.Errorf("識別資訊不符：%+v", info)
	}
	if info.Version != "1.4" || info.ProductCode != 0x1440 || info.Serial != 1 {
		t.Errorf("版本或代碼不符：%+v", info)
	}
	if hz := info.Preferred.RefreshHz(); math.Abs(hz-60) > 0.01 {
		t.Errorf("更新率 %.3f Hz，預期 60", hz)
	}
}

func TestStandardTiming1920x1440(t *testing.T) {
	e, err := Build(Default)
	if err != nil {
		t.Fatal(err)
	}
	// (1920/8)-31 = 209；4:3 → aspect 01；60 Hz → 0
	if e[38] != 0xD1 || e[39] != 0x40 {
		t.Errorf("standard timing = %02X %02X，預期 D1 40", e[38], e[39])
	}
}

func TestParseRejectsBadChecksum(t *testing.T) {
	e, _ := Build(Default)
	e[127]++
	if _, err := Parse(e); err == nil {
		t.Error("checksum 錯誤卻沒有回報")
	}
}

func TestBuildValidates(t *testing.T) {
	p := Default
	p.Manufacturer = "vd"
	if _, err := Build(p); err == nil {
		t.Error("不合法的 manufacturer 沒有被擋下")
	}
	p = Default
	p.Name = "名稱太長太長太長太長"
	if _, err := Build(p); err == nil {
		t.Error("過長的名稱沒有被擋下")
	}
}
