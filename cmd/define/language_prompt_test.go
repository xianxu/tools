package main

import (
	"github.com/xianxu/tools/cmd/define/store"
	"testing"
)

func TestLanguagePrompt(t *testing.T) {
	for _, tc := range []struct {
		lang  store.Lang
		flags bool
		cols  int
		want  string
	}{
		{"", true, 80, "🇺🇸 › "}, {"en", true, 80, "🇺🇸 › "},
		{"es", true, 80, "🇪🇸 › "}, {"it", true, 80, "🇮🇹 › "},
		{"fr", true, 80, "🇫🇷 › "}, {"de", true, 80, "🇩🇪 › "},
		{"pt", true, 80, "🇵🇹 › "}, {"zh", true, 80, "🇨🇳 › "},
		{"ja", true, 80, "🇯🇵 › "}, {"ko", true, 80, "🇰🇷 › "},
		{" ES ", true, 80, "🇪🇸 › "}, {"xx", true, 80, "[xx] › "},
		{"es", false, 80, "[es] › "}, {"es", true, 1, "[es] › "},
		{"es", true, 2, "🇪🇸 › "}, {"\x1b[31m", true, 80, "[??] › "},
	} {
		if got := languagePrompt(tc.lang, tc.flags, tc.cols); got != tc.want {
			t.Errorf("%q flags=%v cols=%d: %q, want %q", tc.lang, tc.flags, tc.cols, got, tc.want)
		}
	}
}
