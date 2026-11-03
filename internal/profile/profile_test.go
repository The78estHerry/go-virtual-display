package profile

import (
	"os"
	"testing"

	"github.com/The78estHerry/go-virtual-display/internal/edid"
)

func TestPresetsBuildAndParse(t *testing.T) {
	for _, p := range Presets {
		if err := p.Validate(); err != nil {
			t.Errorf("%s：%v", p.Key(), err)
			continue
		}
		b, _ := p.EDID()
		info, err := edid.Parse(b)
		if err != nil {
			t.Errorf("%s：%v", p.Key(), err)
			continue
		}
		if info.Preferred.HActive != p.Width || info.Preferred.VActive != p.Height {
			t.Errorf("%s 解析回來是 %dx%d", p.Key(), info.Preferred.HActive, info.Preferred.VActive)
		}
		// 更新率必須剛好是整數：pixel clock ＝ 總像素 × 總行數 × 更新率
		if tm := info.Preferred; tm.PixelClockKHz*1000 != tm.HTotal()*tm.VTotal()*p.Refresh {
			t.Errorf("%s 的更新率不是剛好 %d Hz：%.4f（%+v）", p.Key(), p.Refresh, tm.RefreshHz(), tm)
		}
		if want := "GVD-" + p.Key()[:len(p.Key())-3]; info.SerialText != want {
			t.Errorf("%s 序號字串 %q，預期 %q", p.Key(), info.SerialText, want)
		}
	}
}

func Test1920x1440KeepsOldEDID(t *testing.T) {
	// 0.2.0 以前固定使用的 EDID（edid.Default）除了實體尺寸外應該相同時序
	b, err := Profile{1920, 1440, 60}.EDID()
	if err != nil {
		t.Fatal(err)
	}
	info, _ := edid.Parse(b)
	if info.Preferred != edid.DMT1920x1440at60 {
		t.Errorf("1920×1440@60 的時序 %+v，預期 DMT", info.Preferred)
	}
}

func TestValidate(t *testing.T) {
	for _, c := range []struct {
		p  Profile
		ok bool
	}{
		{Profile{1366, 768, 75}, true},
		{Profile{1080, 1920, 144}, true},
		{Profile{319, 480, 60}, false},
		{Profile{4096, 2160, 60}, false},
		{Profile{1920, 1080, 241}, false},
		{Profile{3840, 2400, 144}, false}, // pixel clock 超過 655.35 MHz
	} {
		if err := c.p.Validate(); (err == nil) != c.ok {
			t.Errorf("%s：Validate() = %v，預期 ok=%v", c.p.Key(), err, c.ok)
		}
	}
}

func TestStore(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)         // Windows 的 os.UserConfigDir
	t.Setenv("XDG_CONFIG_HOME", dir) // Linux 的 os.UserConfigDir

	if l := Load(); l.Active != Default.Key() || len(l.Profiles) != 1 {
		t.Fatalf("沒有檔案時應該只有預設：%+v", l)
	}
	p := Profile{2560, 1440, 60}
	if _, err := Add(p); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(p); err != nil {
		t.Fatal(err)
	}
	if l := Load(); len(l.Profiles) != 2 {
		t.Fatalf("重複加入不應增加：%+v", l)
	}
	if _, err := SetActive(p); err != nil {
		t.Fatal(err)
	}
	if _, err := Remove(p); err == nil {
		t.Error("使用中的解析度不應能刪除")
	}
	if _, err := Remove(Default); err != nil {
		t.Fatal(err)
	}
	l := Load()
	if l.Active != p.Key() || len(l.Profiles) != 1 || l.Has(Default) {
		t.Errorf("刪除後的清單不對：%+v", l)
	}
	if _, err := Add(Profile{100, 100, 60}); err == nil {
		t.Error("不合法的解析度不應能加入")
	}

	// 損壞的檔案回到預設
	path, _ := Path()
	if err := os.WriteFile(path, []byte("{壞掉"), 0o644); err != nil {
		t.Fatal(err)
	}
	if l := Load(); l.Active != Default.Key() {
		t.Errorf("損壞的檔案應回到預設：%+v", l)
	}
}
