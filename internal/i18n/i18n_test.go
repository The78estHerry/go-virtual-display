package i18n

import "testing"

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"": "", "en": English, "en-US": English, "zh-TW": Chinese, "zh_CN.UTF-8": Chinese, "ZH": Chinese, "ja": "",
	} {
		if got := normalize(in); got != want {
			t.Errorf("normalize(%q) = %q，預期 %q", in, got, want)
		}
	}
}
