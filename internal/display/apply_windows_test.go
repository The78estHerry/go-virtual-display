package display

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/profile"
)

func TestSetVDDResolution(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "windows", "scripts", "vdd_settings.xml"))
	if err != nil {
		t.Fatal(err)
	}
	// 模擬原版設定有多個 global 更新率的情況
	tmpl := strings.Replace(string(src), "<g_refresh_rate>60</g_refresh_rate>",
		"<g_refresh_rate>60</g_refresh_rate>\n<g_refresh_rate>90</g_refresh_rate>", 1)
	path := filepath.Join(t.TempDir(), "vdd_settings.xml")
	if err := os.WriteFile(path, []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := setVDDResolution(path, profile.Profile{Width: 2560, Height: 1440, Refresh: 75}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	s := string(b)
	for _, want := range []string{"<width>2560</width>", "<height>1440</height>", "<refresh_rate>75</refresh_rate>", "<g_refresh_rate>75</g_refresh_rate>", "<CustomEdid>true</CustomEdid>"} {
		if !strings.Contains(s, want) {
			t.Errorf("結果缺少 %s", want)
		}
	}
	if n := len(regexp.MustCompile(`<resolution>`).FindAllString(s, -1)); n != 1 {
		t.Errorf("應該只剩 1 組解析度，實際 %d 組", n)
	}
	if n := strings.Count(s, "<g_refresh_rate>"); n != 1 {
		t.Errorf("應該只剩 1 個 global 更新率，實際 %d 個", n)
	}
}

// 安裝用的 vdd_settings.xml 必須是格式正確的 XML，否則驅動不會建立螢幕（0.3.0 測試版就是註解裡寫了 "--"）
func TestVDDSettingsTemplateIsWellFormed(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "windows", "scripts", "vdd_settings.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := checkXML(src); err != nil {
		t.Fatalf("vdd_settings.xml 格式錯誤：%v", err)
	}
	if strings.Contains(string(src), "<logging>true</logging>") {
		t.Error("發佈用的設定不應開啟 logging")
	}
}

func TestCheckXMLRejectsDoubleHyphenInComment(t *testing.T) {
	if checkXML([]byte("<a><!-- vdswitch.exe --apply --></a>")) == nil {
		t.Error("註解裡有 -- 應該回報錯誤")
	}
}

func TestSetVDDResolutionRejectsUnknownFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vdd_settings.xml")
	if err := os.WriteFile(path, []byte("<vdd_settings></vdd_settings>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := setVDDResolution(path, profile.Default); err == nil {
		t.Error("格式不符時應回報錯誤")
	}
}
