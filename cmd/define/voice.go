package main

import "github.com/xianxu/tools/cmd/define/store"

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
// This is the DEFAULT only — `-locale` overrides it for any language. #27
// replaced the interim rule that made the flag English-only; the exception here
// survives because it is a fact about the CDN, not a policy: English recordings
// are keyed _en_us_ and _en_gb_, never _en_en_.
func defaultLocale(l store.Lang) string {
	if l == store.DefaultLang {
		return "us"
	}
	return string(l)
}

// localeFor applies -locale to a language. Pure — a language and a flag in, a
// locale out, no io.Writer (ARCH-PURE).
//
// `-locale` is honoured for EVERY language. It was English-only for one
// milestone: #23 M1's D2 shipped that as an explicit interim rule, written to be
// replaced here, because #23 had measured `_en_us_` and `_es_es_` but had no
// policy for the variant within a language.
//
// NO whitelist of valid language/locale pairs, and that is a decision rather
// than an omission. A closed table would restate a fact the CDN owns and go
// stale the moment Google adds a variant — the same argument ParseLang makes for
// not enumerating languages, and using a different philosophy for the adjacent
// field would be the inconsistency, not the safety. A table would also have no
// answer for `fr`, which ParseLang admits and the CDN serves.
//
// So an unserved pair — `-lang es -locale gb` builds `madrugar_es_gb` — 404s and
// degrades to the warning every missing recording already produces. The CDN
// stays the authority on what exists.
// An empty flag means "not given": #27 changed the flag's default from "us" to
// "", so the value alone carries that. The separate flagSet bool it used to take
// became inert in the same change and was removed rather than kept warm — the
// default was "us" before, which is a real locale, so only fs.Visit could tell
// "asked for American" from "said nothing".
func localeFor(l store.Lang, flag string) string {
	if flag == "" {
		return defaultLocale(l)
	}
	return flag
}

// voiceFor is the one place a voice is built from the session's language and
// flags, so the two fields cannot be assembled inconsistently at a call site.
func voiceFor(l store.Lang, flag string) voice {
	return voice{Lang: l, Locale: localeFor(l, flag)}
}

// applyVoice derives the session's voice from a language.
//
// One derivation with two callers — the boundary in run(), and applyLang for a
// mid-session /lang. They existed as two expressions for exactly one commit, and
// in that commit only the first ran: a /lang es session kept asking the CDN for
// English URLs, including the legacy pair that had just been gated to English.
// A derived value with one deriving function cannot drift like that.
//
// It took an io.Writer while localeFor could complain about a locale it refused.
// #27 removed the refusal — the CDN decides what exists — so there is nothing
// left to say and the parameter went with it rather than being kept warm for a
// hypothetical caller.
func applyVoice(opt *options, l store.Lang) {
	opt.voice = voiceFor(l, opt.locale)
}

// localeHelp is THE statement of what -locale means, and the one source for it.
//
// The policy was written in four places — this flag, localeFor, the README and
// the atlas — with nothing keeping them in step, which is the family this repo
// already mechanised for the play-loop prompts. TestDocsQuoteTheLocaleHelp
// makes both the README and atlas/define.md consumers of this string.
//
// It gives EXAMPLES rather than an enumeration, because localeFor does not
// whitelist: the CDN decides what exists, so a help text claiming a closed set
// would be the same restatement in prose.
//
// The Spanish pair is named with its PHONEMIC content rather than two country
// codes, because that is the actual choice: es_es is Castilian, distinguishing
// cazar /θ/ from casar /s/; es_us is Latin American seseo, where both are /s/.
// Choosing one chooses which sound system a learner acquires. "us or gb" said
// none of that, and omitted Spanish entirely.
const localeHelp = "regional variant of the pronunciation, per language: " +
	"en us|gb; es es (Castilian, cazar /θ/) or us (seseo, /s/). " +
	"Others exist — the CDN decides, not a list here"

// pronHelp is THE statement of what -pron means, and the one source for it.
//
// Same mechanism as localeHelp above, for the same reason: the -locale policy
// was once written in four places with nothing keeping them in step, so
// TestDocsQuoteThePronHelp makes the README and the atlas CONSUMERS of this
// string rather than restatements of it.
//
// It says "this lookup" because that is the whole distinction from -lang. A mode
// moves the deck, the dictionary and the highlight set; this moves nothing but
// the recording. And it names ORIGIN, because that is where a reader finds the
// language to type — #29 chose a declared language over an inferred one, so the
// help has to say where the answer is.
const pronHelp = "hear THIS lookup in another language without switching the " +
	"session: -pron fr arrondissement. The entry's ORIGIN says which. Falls back " +
	"to the session's recording, and says so, when the source has none"
