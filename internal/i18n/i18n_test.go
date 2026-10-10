package i18n

import "testing"

func TestAllKeysHaveBothLanguages(t *testing.T) {
	for key, entry := range messages {
		if entry[LangZH] == "" {
			t.Errorf("key %q missing zh translation", key)
		}
		if entry[LangEN] == "" {
			t.Errorf("key %q missing en translation", key)
		}
	}
}

func TestIsChineseLocale(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"zh_CN.UTF-8", true},
		{"zh-Hans-CN", true},
		{"zh_TW", true},
		{"en_US.UTF-8", false},
		{"en", false},
		{"", false},
		{"C", false},
		{"POSIX", false},
		{"ja_JP.UTF-8", false},
	}
	for _, test := range tests {
		if got := isChineseLocale(test.input); got != test.want {
			t.Errorf("isChineseLocale(%q) = %v, want %v", test.input, got, test.want)
		}
	}
}

func TestSystemLangPrefersChineseEnvOverDefaults(t *testing.T) {
	// 环境变量优先级高于系统 defaults，中文环境应直接返回中文。
	t.Setenv("LC_ALL", "zh_CN.UTF-8")
	t.Setenv("LANG", "zh_CN.UTF-8")
	if got := SystemLang(); got != LangZH {
		t.Fatalf("SystemLang() = %q, want %q", got, LangZH)
	}
}

func TestLocalizerResolvesAuto(t *testing.T) {
	t.Setenv("LC_ALL", "zh_CN.UTF-8")
	l := New(LangAuto)
	if l.Lang() != LangZH {
		t.Fatalf("auto resolved to %q, want %q", l.Lang(), LangZH)
	}
}

func TestLocalizerSetLangOverridesAuto(t *testing.T) {
	t.Setenv("LC_ALL", "zh_CN.UTF-8")
	l := New(LangZH).SetLang(LangEN)
	if l.Lang() != LangEN {
		t.Fatalf("SetLang(en) = %q, want %q", l.Lang(), LangEN)
	}
}

func TestLocalizerTReturnsKeyWhenMissing(t *testing.T) {
	l := New(LangZH)
	if got := l.T("nonexistent.key"); got != "nonexistent.key" {
		t.Fatalf("missing key returned %q", got)
	}
}

func TestLocalizerTfFormats(t *testing.T) {
	l := New(LangZH)
	if got := l.Tf("menu.tooltip.download", "1.2MB/s"); got != "下载 1.2MB/s" {
		t.Fatalf("Tf = %q", got)
	}
}
