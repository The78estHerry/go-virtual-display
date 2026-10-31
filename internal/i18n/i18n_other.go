//go:build !windows

package i18n

import (
	"bufio"
	"os"
	"strings"
)

func installedLanguage() string {
	f, err := os.Open("/etc/go-virtual-display.conf")
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if v, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "LANGUAGE="); ok {
			return v
		}
	}
	return ""
}

func systemIsChinese() bool {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(k); v != "" {
			return strings.HasPrefix(strings.ToLower(v), "zh")
		}
	}
	return false
}
