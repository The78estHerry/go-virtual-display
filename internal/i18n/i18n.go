// Package i18n 決定介面語言（英文或繁體中文）並提供翻譯。
//
// 翻譯用 go-desktop-kit 的 desktop/i18n 訊息表（messages.go），程式裡用 T("key") 取字串；
// 缺翻譯時退回英文。之後加語言只要多一張表。
//
// 語言以安裝時使用者選的為準（Windows：HKLM\SOFTWARE\go-virtual-display 的 Language；
// Linux：/etc/go-virtual-display.conf 的 LANGUAGE），沒有設定時依系統顯示語言挑選：
// 中文語系用繁體中文，其他一律英文。環境變數 GO_VIRTUAL_DISPLAY_LANG（en／zh-TW）優先，供測試用。
package i18n

import (
	"os"

	kit "github.com/The78estHerry/go-desktop-kit/desktop/i18n"
)

const (
	English = "en"
	Chinese = "zh-TW"
)

// catalog 在套件初始化時就決定語言：其他套件的套件層級變數（例如 display.ErrNotInstalled）
// 在初始化時就會呼叫 T。
var catalog = func() *kit.Catalog {
	c := kit.New(English)
	c.Add(English, en)
	c.Add(Chinese, zhTW)
	c.SetLanguage(resolve(c.Languages(), os.Getenv("GO_VIRTUAL_DISPLAY_LANG"), installedLanguage(), kit.SystemLanguages()))
	return c
}()

// resolve 依序看環境變數、安裝時選的語言，都沒有（或不是支援的語言）時依系統顯示語言。
func resolve(have []string, env, installed string, system []string) string {
	for _, l := range []string{env, installed} {
		if m := kit.Match([]string{l}, have, ""); m != "" {
			return m
		}
	}
	return kit.Match(system, have, English)
}

// Current 回傳目前的介面語言（English 或 Chinese）。
func Current() string { return catalog.Language() }

// IsChinese 回傳介面是否為繁體中文。
func IsChinese() bool { return Current() == Chinese }

// T 回傳 key 在目前介面語言的字串；有 args 時當成 fmt 格式字串。
func T(key string, args ...any) string { return catalog.T(key, args...) }
