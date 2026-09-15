package main

import "github.com/xianxu/tools/cmd/define/store"

// These are presentation conventions, independent of dictionary availability
// and pronunciation locale. Unknown languages keep their readable code.
var languageFlags = map[store.Lang]string{
	"en": "🇺🇸", "es": "🇪🇸", "it": "🇮🇹", "fr": "🇫🇷", "de": "🇩🇪",
	"pt": "🇵🇹", "zh": "🇨🇳", "ja": "🇯🇵", "ko": "🇰🇷",
}

func languagePrompt(lang store.Lang, flags bool, cols int) string {
	if lang == "" {
		lang = store.DefaultLang
	}
	normalized, err := store.ParseLang(string(lang))
	if err != nil {
		return "[??] " + prompt
	}
	if flag := languageFlags[normalized]; flags && cols >= 2 && flag != "" {
		return flag + " " + prompt
	}
	return "[" + string(normalized) + "] " + prompt
}
