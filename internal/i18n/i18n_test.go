package i18n

import (
	"reflect"
	"testing"
)

func TestDetect(t *testing.T) {
	t.Setenv("GH_NEW_REPO_LANG", "")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "")

	tests := []struct {
		name string
		pref string
		lang string // LANG 環境変数
		want Lang
	}{
		{"explicit ja", "ja", "", JA},
		{"explicit en", "en", "ja_JP.UTF-8", EN},
		{"auto falls back to LANG ja", "auto", "ja_JP.UTF-8", JA},
		{"auto with no locale", "auto", "", EN},
		{"empty pref uses LANG", "", "en_US.UTF-8", EN},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LANG", tt.lang)
			if got := Detect(tt.pref); got != tt.want {
				t.Errorf("Detect(%q) with LANG=%q = %v, want %v", tt.pref, tt.lang, got, tt.want)
			}
		})
	}
}

func TestStringsComplete(t *testing.T) {
	// 英語と日本語の全フィールドを確認し、翻訳漏れを検出します。
	for name, s := range map[string]Strings{"en": en, "ja": ja} {
		v := reflect.ValueOf(s)
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).String() == "" {
				t.Errorf("%s: Strings.%s is empty", name, v.Type().Field(i).Name)
			}
		}
	}
}
