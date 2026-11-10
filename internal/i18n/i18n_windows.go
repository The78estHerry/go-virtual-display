package i18n

import "golang.org/x/sys/windows/registry"

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
