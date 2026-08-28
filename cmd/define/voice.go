package main

import (
	"fmt"

	"github.com/xianxu/tools/cmd/define/store"
)

// voice is language plus regional variant for a recording: the Spanish of Spain
// is voice{Lang: "es", Locale: "es"}, American English is {"en", "us"}.
//
// A struct rather than two strings because "es" is a legal value of BOTH fields.
// Two positional strings are transposable at every call site and the compiler
// cannot tell; a struct makes the mistake unspellable.
type voice struct {
	Lang   store.Lang
	Locale string
}

// defaultLocale is the locale a language implies when nobody says otherwise.
//
// ONE rule with ONE exception: the locale is the language code (es -> es_es),
// except English, whose CDN recordings are _en_us_ and _en_gb_ rather than
// _en_en_. Both halves are measured, not assumed — madrugar_es_es_1.mp3 is a 200
// and sycophantic_en_us_1.mp3 is a 200.
//
// #27 owns real locale policy (es_es vs es_us, the θ/seseo split). This is the
// interim rule it inherits, written down rather than left to fall out of a
// default.
func defaultLocale(l store.Lang) string {
	if l == store.DefaultLang {
		return "us"
	}
	return string(l)
}

// localeFor applies -locale to a language, and returns any complaint rather than
// printing one — the policy is pure, the writing is the caller's (ARCH-PURE).
//
// -locale is documented as "us or gb", which are ENGLISH variants. Honouring it
// for another language would build madrugar_es_gb_1.mp3: a URL form nothing has
// measured, and a ~450ms miss when it 404s. So it applies to English only, and
// says so instead of being silently dropped — a flag that is quietly ignored is
// worse than one that is refused.
func localeFor(l store.Lang, flag string, flagSet bool) (locale string, complaint string) {
	if !flagSet || flag == "" {
		return defaultLocale(l), ""
	}
	if l == store.DefaultLang {
		return flag, ""
	}
	return defaultLocale(l), fmt.Sprintf(
		"-locale %s is an English variant; %s recordings use %s_%s. Locale variants for other "+
			"languages are not supported yet", flag, l, l, defaultLocale(l))
}

// voiceFor is the one place a voice is built from the session's language and
// flags, so the two fields cannot be assembled inconsistently at a call site.
func voiceFor(l store.Lang, flag string, flagSet bool) (voice, string) {
	locale, complaint := localeFor(l, flag, flagSet)
	return voice{Lang: l, Locale: locale}, complaint
}
