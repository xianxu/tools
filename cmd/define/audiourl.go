package main

import (
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/xianxu/tools/cmd/define/store"
)

// audioBase is Google's dictionary pronunciation CDN. The paths are
// undocumented but have been stable for a decade and through two migrations;
// fetch.go's conformance test is what detects a third.
const audioBase = "https://ssl.gstatic.com/dictionary/static"

// SourceSpellings returns the spellings a word's SOURCE language might key its
// recording under, best first. Pure and offline, like everything else here.
//
// THE CONSTRAINT THIS EXISTS FOR (#29): the CDN keys a source recording on the
// source ORTHOGRAPHY, not on what a person types. Measured 2026-08-28/29 —
//
//	jalapeno_en_us  200   ← what define asked for before #29
//	jalapeño_es_es  200   ← the Spanish recording
//	jalapeno_es_es  404   ← the same word, Spanish locale, unaccented
//
// and the same split for piñata/pinata, señor/senor, café/cafe, naïve/naive,
// façade/facade, cliché/cliche, fiancé/fiance. Swapping the language field in
// the URL builder is therefore not enough; the spelling is a second input.
//
// Three sources: the HEADWORD, the `(also …)` alternatives that differ from it
// only by diacritics (AlsoSpellings owns that filter), and the TYPED word as the
// safety net for an entry whose head did not parse — which is also all a caller
// with no entry at all has.
//
// ACCENTED SPELLINGS GO FIRST, and that beats "the dictionary's own form first"
// on measurement: NOAD files the accent on either side of the headword, so
// headword-first is right 5 times in 8 (jalapeño, piñata, señor, cliché, fiancé
// are headwords) and wrong 3 (café, naïve, façade, where the head is unaccented
// and the accent is an alternative). Ranking by "carries a non-ASCII rune" is
// right 8 times in 8 and saves that second class two 404s. Where nothing carries
// an accent the order is untouched, so this costs nothing in the common case.
//
// KNOWN LIMITATION, measured against the live dictionary rather than reasoned:
// `role` is a ninth borrowing whose French recording exists (rôle_fr_fr is a
// 200, role_fr_fr a 404) and which NO rule here reaches. NOAD heads it `role`,
// offers no `(also rôle)`, and spells the accented form only inside the ORIGIN
// prose — "from French rôle, from obsolete French roule 'roll'". Mining ORIGIN
// is the natural fourth source and is deliberately not done: that one sentence
// offers three candidate tokens, so extracting the right one is a parsing
// problem rather than another lookup, and `role` degrades to the English
// recording and says so, like any other miss.
//
// A stable PARTITION rather than a sort: among spellings that are alike, source
// order stands — headword, then alternatives, then what was typed.
//
// Lowercased and deduped case-insensitively. Both matter: Señor_es_es is a 404
// where señor_es_es is a 200, and for arrondissement all three sources agree, so
// without the dedupe the walk asks the CDN the same question three times.
func SourceSpellings(typed string, e Entry) []string {
	var out []string
	seen := map[string]bool{}
	add := func(s string) {
		s = strings.ToLower(strings.Join(strings.Fields(s), " "))
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	add(e.Headword())
	for _, alt := range e.AlsoSpellings() {
		add(alt)
	}
	add(typed)
	slices.SortStableFunc(out, func(a, b string) int {
		switch aa, ba := isASCIIOnly(a), isASCIIOnly(b); {
		case aa == ba:
			return 0
		case ba:
			return -1 // a carries an accent and b does not: a first
		default:
			return 1
		}
	})
	return out
}

// isASCIIOnly reports whether s carries no rune outside ASCII, which is this
// package's stand-in for "wears no accent".
func isASCIIOnly(s string) bool { return len(s) == utf8.RuneCountInString(s) }

// AudioCandidates returns the CDN URLs to try, in order, for a word's recorded
// pronunciation. Pure and offline — no request is made here, so the ordering is
// unit-testable without a server.
//
// The order comes from a survey across 10 words: the 2022 generation strictly
// dominates the legacy sounds/oxford paths (gaslighting exists only on 2022;
// defenestrate needs the _2 suffix on the legacy path), so one URL is not
// enough and the fallback policy belongs here rather than in the fetch loop.
//
// ONE language, and that is still true OF THIS FUNCTION: it builds for the voice
// it is handed and never searches across languages. #23 makes the session's
// language a declared mode, so there is nothing to guess, which is what deleted
// #27's planned voices() fallback.
//
// The cross-language walk lives one level up, in utterance.Candidates (#29), and
// it is not a search either — it runs only when -pron or /pron named a language
// outright, and it reports which voice actually answered. Guessing is still what
// this package does not do; being TOLD is new.
func AudioCandidates(word string, v voice) []string {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return nil
	}
	v.Lang = v.langOrDefault()
	if v.Locale == "" {
		v.Locale = defaultLocale(v.Lang)
	}
	// Multi-word headwords are spelled with the spaces REMOVED, not with
	// underscores — the CDN key for "hot dog" is `hotdog`.
	//
	// MEASURED 2026-08-30, because the underscore here was an assumption that had
	// never met the server and silently broke every phrase in the corpus:
	//
	//	hotdog_en_us_1   200      hot_dog_en_us_1   404      hot-dog_en_us_1   404
	//	defacto_en_us_1  200      de_facto_en_us_1  404      de-facto_en_us_1  404
	//
	// Coverage for phrases is genuinely sparse even so — `adhoc`, `statusquo`,
	// `coupdetat` and `neplusultra` are all 404 under every spelling — so a miss
	// on a phrase is still the normal outcome. That is the difference this fix
	// makes readable: a phrase that HAS a recording now gets it, instead of every
	// phrase missing for the same wrong reason.
	slug := strings.ReplaceAll(word, " ", "")
	esc := url.PathEscape(slug)
	// The locale is user input too — it comes straight from -locale — so it is
	// escaped like the word. Unescaped, a value containing "/" would rewrite the
	// path rather than 404 cleanly, and the tool's contract for an unserved
	// locale is a clean miss (nothing whitelists which locales exist).
	loc := url.PathEscape(v.Locale)

	// The shard is the first two letters — or one, for a single-letter word.
	shard := esc
	if r := []rune(slug); len(r) >= 2 {
		shard = url.PathEscape(string(r[:2]))
	}

	var out []string
	for _, n := range []string{"1", "2"} {
		out = append(out, audioBase+"/pronunciation/2022-03-02/audio/"+shard+"/"+esc+"_"+string(v.Lang)+"_"+loc+"_"+n+".mp3")
	}
	// The legacy generation is ENGLISH-ONLY. Measured 2026-08-28: madrugar--_us_1
	// and madrugar--_es_1 are both 404 while sycophantic--_us_1 is 200. Asking
	// anyway would cost ~450ms per lookup for a guaranteed miss — the exact cost
	// #23's declared mode exists to stop paying.
	if v.Lang == store.DefaultLang {
		for _, n := range []string{"1", "2"} {
			out = append(out, audioBase+"/sounds/oxford/"+esc+"--_"+loc+"_"+n+".mp3")
		}
	}
	return out
}

// utterance is one request for a recording: the word, the spellings its source
// language might key it under, and the two voices to try in order (#29).
//
// A struct rather than positional arguments because Source and Session are BOTH
// `voice` and therefore transposable at every call site — the same argument that
// made `voice` itself a struct, one level up. The compiler cannot tell two
// voices apart; a field name can.
type utterance struct {
	// Word is what the user typed. The SESSION's recording is keyed on it,
	// exactly as it was before #29 — that path is deliberately unchanged.
	Word string
	// Spellings are the SOURCE orthographies, best first, from SourceSpellings.
	Spellings []string
	// Source is the voice -pron asked for. A zero Lang means nothing was asked
	// for, and then this is exactly the pre-#29 single-voice request.
	Source voice
	// Session is the voice this session would have used anyway: the fallback,
	// and the only voice when Source is zero.
	Session voice
}

// sourceCandidates is every URL for the source voice: one AudioCandidates run
// per spelling, in order.
//
// Empty when no source was asked for, and empty when the source voice IS the
// session's — `/pron es` inside a Spanish session asks for the recording already
// being fetched, and asking twice is two requests for one answer.
func (u utterance) sourceCandidates() []string {
	if !u.askedForSource() {
		return nil
	}
	var out []string
	for _, s := range u.Spellings {
		out = append(out, AudioCandidates(s, u.Source)...)
	}
	return out
}

// Candidates is the whole walk: the source attempts, then the session's own
// recording as the fallback.
//
// THIS IS A CROSS-LANGUAGE FALLBACK, which #23 removed and the atlas recorded as
// "One language, no fallback". Reintroducing it is deliberate, and the two cases
// are not the same one. #23's fallback would have been a GUESS about which
// language a word belongs to, run on every lookup and paid for by every user —
// and measurement says such a guess is often wrong: police_fr_fr,
// restaurant_fr_fr and machine_fr_fr are all 200 for words nobody wants said in
// French. This one runs ONLY when -pron or /pron was typed, and what actually
// answered is reported (reportVoice) rather than left to sound like success. A
// documented degradation from an explicit request, not a guess.
//
// It has to exist because source coverage is partial, measured 2026-08-28/29:
// hotel_fr_fr and debut_fr_fr are 404 while their _en_us_ are 200, Italian is
// absent from this CDN generation entirely (ciao, pizza, espresso, opera), and
// so is Japanese (karaoke, tsunami). Without the fallback those words play
// nothing at all.
func (u utterance) Candidates() []string {
	src := u.sourceCandidates()
	ses := AudioCandidates(u.Word, u.Session)
	// A fresh slice: appending onto what sourceCandidates returned would let a
	// caller's retained result alias this one.
	out := make([]string, 0, len(src)+len(ses))
	out = append(out, src...)
	return append(out, ses...)
}

// spokeSource reports whether the URL that answered was a source attempt.
//
// By MEMBERSHIP in the list actually built, never by parsing the URL. The report
// this feeds is a RECORD — it survives on a pipe and cannot be taken back — and
// a record has to be true. Reading a language back out of a path would be a
// second, driftable statement of what a source URL looks like, and it would go
// wrong the day the CDN generation moves.
func (u utterance) spokeSource(from string) bool {
	return slices.Contains(u.sourceCandidates(), from)
}

// askedForSource reports whether this utterance requested another language at
// all, treating "the source IS the session" as not having asked.
//
// THE predicate, not a copy of one: sourceCandidates calls it rather than
// spelling the same condition negated, which is how the two came to need a
// comment promising they agreed (ARCH-DRY).
func (u utterance) askedForSource() bool {
	return u.Source.Lang != "" && u.Source != u.Session
}
