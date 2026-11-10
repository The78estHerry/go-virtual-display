package i18n

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
)

func TestResolve(t *testing.T) {
	have := []string{English, Chinese}
	for _, c := range []struct {
		env, installed string
		system         []string
		want           string
	}{
		{"", "", []string{"zh-TW", "en-US"}, Chinese},
		{"", "", []string{"zh-CN"}, Chinese}, // 只有繁中時，所有中文語系都用繁中
		{"", "", []string{"ja-JP", "en-US"}, English},
		{"", "", nil, English},
		{"", "zh-TW", []string{"en-US"}, Chinese}, // 安裝時選的優先
		{"", "en", []string{"zh-TW"}, English},
		{"en", "zh-TW", []string{"zh-TW"}, English}, // 環境變數最優先
		{"", "ja", []string{"zh-TW"}, Chinese},      // 不支援的設定值改看系統語言
	} {
		if got := resolve(have, c.env, c.installed, c.system); got != c.want {
			t.Errorf("resolve(%q, %q, %v) = %q，預期 %q", c.env, c.installed, c.system, got, c.want)
		}
	}
}

// TestMessages 檢查兩張訊息表的 key 一致，而且程式裡用到的 key 都存在。
func TestMessages(t *testing.T) {
	for k := range en {
		if _, ok := zhTW[k]; !ok {
			t.Errorf("繁中缺少 %q", k)
		}
	}
	for k := range zhTW {
		if _, ok := en[k]; !ok {
			t.Errorf("英文缺少 %q", k)
		}
	}
	call := regexp.MustCompile(`i18n\.T\("([^"]*)"`)
	var used []string
	err := filepath.WalkDir("../..", func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".go" {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, m := range call.FindAllStringSubmatch(string(b), -1) {
			if _, ok := en[m[1]]; !ok {
				t.Errorf("%s 用了不存在的 key %q", p, m[1])
			}
			used = append(used, m[1])
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for k := range en {
		if !slices.Contains(used, k) {
			t.Errorf("訊息表的 %q 沒有被使用", k)
		}
	}
}
