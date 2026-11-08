package ui

import (
	"errors"
	"strings"

	"gioui.org/io/key"

	"github.com/The78estHerry/go-virtual-display/internal/i18n"
)

// 快捷鍵（0.4.4 起，目前只有 Windows）：把小工具移到滑鼠游標旁邊。
// 虛擬螢幕被設成主螢幕、系統列和小工具都在看不到的螢幕上時，用它把開關叫回來。
// 刻意不做成「拔掉」：誤按也只是移動小工具，不會影響畫面。

// DefaultHotkey 是預設的快捷鍵。
const DefaultHotkey = "Ctrl+Alt+Shift+V"

// Hotkey 是一組組合鍵：修飾鍵＋一個主鍵（A～Z、0～9、F1～F12）。
type Hotkey struct {
	Mods key.Modifiers
	Key  string
}

// 修飾鍵的固定順序與名稱
var hotkeyMods = []struct {
	mod  key.Modifiers
	name string
}{
	{key.ModCtrl, "Ctrl"},
	{key.ModAlt, "Alt"},
	{key.ModShift, "Shift"},
	{key.ModSuper, "Win"},
}

func (h Hotkey) String() string {
	var parts []string
	for _, m := range hotkeyMods {
		if h.Mods&m.mod != 0 {
			parts = append(parts, m.name)
		}
	}
	return strings.Join(append(parts, h.Key), "+")
}

// IsZero 表示沒有設定（暫停時用）。
func (h Hotkey) IsZero() bool { return h.Key == "" }

// validHotkeyKey 判斷 k 能不能當主鍵。
func validHotkeyKey(k string) bool {
	if len(k) == 1 {
		c := k[0]
		return c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
	}
	if len(k) >= 2 && len(k) <= 3 && k[0] == 'F' {
		n := 0
		for _, c := range k[1:] {
			if c < '0' || c > '9' {
				return false
			}
			n = n*10 + int(c-'0')
		}
		return n >= 1 && n <= 12
	}
	return false
}

// Validate 檢查組合鍵：主鍵要合法，而且至少有 Ctrl、Alt、Win 其中一個，
// 只有 Shift 會干擾一般打字。
func (h Hotkey) Validate() error {
	if !validHotkeyKey(h.Key) {
		return errors.New(i18n.T("Use a letter, a digit or F1–F12 as the key", "主鍵請用英文字母、數字或 F1～F12"))
	}
	if h.Mods&(key.ModCtrl|key.ModAlt|key.ModSuper) == 0 {
		return errors.New(i18n.T("Include Ctrl, Alt or Win", "至少要包含 Ctrl、Alt 或 Win"))
	}
	return nil
}

// ParseHotkey 解析「Ctrl+Alt+Shift+V」這樣的字串，不分大小寫。
func ParseHotkey(s string) (Hotkey, error) {
	var h Hotkey
	for _, p := range strings.Split(s, "+") {
		p = strings.ToUpper(strings.TrimSpace(p))
		found := false
		for _, m := range hotkeyMods {
			if p == strings.ToUpper(m.name) {
				h.Mods |= m.mod
				found = true
			}
		}
		if found {
			continue
		}
		if h.Key != "" {
			return Hotkey{}, errors.New("hotkey has more than one key: " + s)
		}
		h.Key = p
	}
	if err := h.Validate(); err != nil {
		return Hotkey{}, err
	}
	return h, nil
}

// hotkeyFromSettings 回傳設定裡的快捷鍵；沒設定或格式錯誤時用預設值。
func hotkeyFromSettings(s Settings) Hotkey {
	if h, err := ParseHotkey(s.Hotkey); err == nil {
		return h
	}
	h, _ := ParseHotkey(DefaultHotkey)
	return h
}
