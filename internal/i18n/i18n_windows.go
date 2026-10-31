package i18n

import (
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func installedLanguage() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\go-virtual-display`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return ""
	}
	defer k.Close()
	v, _, err := k.GetStringValue("Language")
	if err != nil {
		return ""
	}
	return v
}

var procGetUserDefaultUILanguage = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")

// systemIsChinese 依使用者的顯示語言判斷。PowerShell 的 Get-UICulture 可能和實際顯示語言不同，所以直接查 API。
func systemIsChinese() bool {
	id, _, _ := procGetUserDefaultUILanguage.Call()
	return id&0x3ff == 0x04 // LANG_CHINESE（不分繁簡）
}
