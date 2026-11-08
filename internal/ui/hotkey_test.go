package ui

import (
	"testing"

	"gioui.org/io/key"
)

func TestParseHotkey(t *testing.T) {
	h, err := ParseHotkey(DefaultHotkey)
	if err != nil {
		t.Fatal(err)
	}
	if h.Mods != key.ModCtrl|key.ModAlt|key.ModShift || h.Key != "V" {
		t.Fatalf("預設快捷鍵解析錯誤：%+v", h)
	}
	if h.String() != DefaultHotkey {
		t.Fatalf("String() = %q", h.String())
	}
	// 不分大小寫、修飾鍵順序不同也一樣
	if h2, err := ParseHotkey("shift + v + ctrl+alt"); err != nil || h2 != h {
		t.Fatalf("ParseHotkey 不分順序失敗：%+v %v", h2, err)
	}
	for _, s := range []string{"Win+F12", "Ctrl+0", "Alt+F1"} {
		if _, err := ParseHotkey(s); err != nil {
			t.Errorf("%s 應該合法：%v", s, err)
		}
	}
	for _, s := range []string{"", "V", "Shift+V", "Ctrl+Alt", "Ctrl+F13", "Ctrl+Space", "Ctrl+A+B", "Ctrl+F0"} {
		if _, err := ParseHotkey(s); err == nil {
			t.Errorf("%q 應該不合法", s)
		}
	}
}

func TestHotkeyFromSettings(t *testing.T) {
	if h := hotkeyFromSettings(Settings{}); h.String() != DefaultHotkey {
		t.Fatalf("沒設定時應該用預設值，得到 %s", h)
	}
	if h := hotkeyFromSettings(Settings{Hotkey: "Ctrl+Alt+F9"}); h.String() != "Ctrl+Alt+F9" {
		t.Fatalf("得到 %s", h)
	}
	if h := hotkeyFromSettings(Settings{Hotkey: "壞掉"}); h.String() != DefaultHotkey {
		t.Fatalf("格式錯誤時應該用預設值，得到 %s", h)
	}
}
