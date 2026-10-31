// Package i18n 決定介面語言（英文或繁體中文）並提供翻譯。
//
// 語言以安裝時使用者選的為準（Windows：HKLM\SOFTWARE\go-virtual-display 的 Language；
// Linux：/etc/go-virtual-display.conf 的 LANGUAGE），沒有設定時依系統顯示語言：
// 中文語系用繁體中文，其他一律英文。環境變數 GO_VIRTUAL_DISPLAY_LANG（en／zh-TW）優先，供測試用。
package i18n

import (
	"os"
	"strings"
)

const (
	English = "en"
	Chinese = "zh-TW"
)

var current = resolve()

func resolve() string {
	if l := normalize(os.Getenv("GO_VIRTUAL_DISPLAY_LANG")); l != "" { // 測試用
		return l
	}
	if l := normalize(installedLanguage()); l != "" {
		return l
	}
	if systemIsChinese() {
		return Chinese
	}
	return English
}

func normalize(s string) string {
	switch {
	case s == "":
		return ""
	case strings.HasPrefix(strings.ToLower(s), "zh"):
		return Chinese
	case strings.HasPrefix(strings.ToLower(s), "en"):
		return English
	}
	return ""
}

// Current 回傳目前的介面語言（English 或 Chinese）。
func Current() string { return current }

// IsChinese 回傳介面是否為繁體中文。
func IsChinese() bool { return current == Chinese }

// T 依介面語言回傳英文或繁體中文的字串。
func T(en, zh string) string {
	if current == Chinese {
		return zh
	}
	return en
}
