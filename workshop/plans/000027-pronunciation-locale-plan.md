# Pronunciation Locale Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the recording's language a parameter instead of a literal, so a Spanish word is asked for as Spanish.

**Architecture:** A pure `voice{Lang, Locale}` value replaces the bare `locale string`, and a pure `voices()` function owns the ordering policy — which languages to try, in what order. `AudioCandidates` builds URLs for that ordered list. Nothing about the fetch loop changes: it already walks candidates until the first 200, so widening the list costs no new mechanism.

**Tech Stack:** Go, no new dependencies. Existing `fakeCDN` stateful fake for integration tests; existing `-tags conformance` suite for the live CDN.

---

## Measurements this plan rests on

Every number below was re-measured on **2026-08-28** against the live CDN, not carried from the issue (which measured on 2026-08-22). Three of these are NEW and change the design; the issue did not have them.

| claim | measured | consequence |
|---|---|---|
| `madrugar_en_us_1` | **404** | today's builder can never find a Spanish recording |
| `madrugar_es_es_1` / `madrugar_es_us_1` | **200 / 200** | both variants exist |
| `es_419`, `es_mx`, `es_ar` | **404, 404, 404** | only `es`/`us` are real Spanish locales |
| `sobremesa`, `empalagoso`, `chapucero`, `desvelarse` on `es_es` | **200 ×4** | coverage is real, not incidental |
| **`sycophantic_es_es_1`** | **404** | **languages are DISJOINT — trying a second language can never return the wrong word's audio** |
| **`madrugar--_us_1`, `madrugar--_es_1`** | **404, 404** | **the legacy `/sounds/oxford/` path is ENGLISH-ONLY — Spanish candidates on it are guaranteed waste** |
| **a 404 vs a 200** | **~300–600ms vs ~40ms** | **a wasted candidate costs ~10× a hit; ordering is a performance decision, not only a correctness one** |
| `defenestrate` modern_1 | **200** (legacy_1 also 200) | the modern path now serves every sampled word; legacy is pure fallback and belongs last |

The three bolded rows are why this plan is not simply "append `es` candidates to the list".

Reproduce with (unsandboxed — the sandbox cannot reach the CDN):

```sh
B=https://ssl.gstatic.com/dictionary/static/pronunciation/2022-03-02/audio
curl -s -o /dev/null -w '%{http_code} %{time_total}\n' "$B/ma/madrugar_es_es_1.mp3"
curl -s -o /dev/null -w '%{http_code} %{time_total}\n' "$B/sy/sycophantic_es_es_1.mp3"
```

## Scope check

One subsystem: the audio URL builder and the flag that feeds it. `#18 M2` (deck language dimension, lemma identity, gender, agreement-safe distractors) is explicitly NOT here and remains blocked on `#10`/`#12`. `#26` (NOAD dual-locale rendering) depends on this landing but is not part of it.

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `voice` | `cmd/define/voice.go` | new |
| ~~voices~~ (never built) | `cmd/define/voice.go` | **superseded by `#23`** — see Revisions |
| `AudioCandidates` | `cmd/define/audiourl.go` | modified |

- **`voice`** — the language plus regional variant a recording is asked for: `voice{Lang: "es", Locale: "es"}`.
  - **Relationships:** 1:N with the candidate URLs it produces. Held by `options`, passed to `speak` and `AudioCandidates`.
  - **DRY rationale:** First occurrence, and it exists to kill a *mix-up hazard rather than a duplication*. `"es"` is a valid value of BOTH fields — `AudioCandidates(w, "es", "es")` (Castilian) and a transposed `AudioCandidates(w, "es", "en")` are indistinguishable at the call site, and there are 20+ call sites in tests. A two-field struct makes the transposition unspellable.
  - **Future extensions:** `#18 M2` gives the deck a language, so a word will carry its own `voice`; this is the type that lands on it. A third field (dialect, speaker sex) widens here without touching callers.

- **~~voices~~ — DELETED, not adapted.** `#23` made the language a declared MODE, so there is nothing to order: the mode supplies exactly one language and `AudioCandidates` builds for it alone. What shipped instead is `localeFor`, the interim locale rule `#27` inherits.
  - **Relationships:** pure `voice → []voice`. Called only by `AudioCandidates`.
  - **DRY rationale:** Splitting the policy from the URL construction is what makes the policy testable as a table without asserting on URL strings. `AudioCandidates` then has one job (spell a URL) and `voices` has one job (decide what to ask for).
  - **Why it is its own function, not an `if` inside `AudioCandidates`:** the ordering is the part with measured facts behind it and the part most likely to change (a third language, a cheaper probe, a per-deck default). Burying it in string concatenation is how the `_en_` literal happened in the first place.
  - **Future extensions:** when the deck knows a word's language, an unspecified `voice` stops meaning "guess" and starts meaning "ask the deck" — that is a change to this function alone.

- **`AudioCandidates`** — unchanged in purpose: the ordered CDN URLs to try. Now takes a `voice`, and emits legacy `/sounds/oxford/` candidates **only for English**, because they are measured English-only.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `-lang` flag | `cmd/define/main.go` | new | operator input |
| `fakeCDN` | `cmd/define/fetch_fake_test.go` | reused | Google's pronunciation CDN |
| `TestCDNStillServesSpanishOnTheExpectedPaths` | `cmd/define/fetch_conformance_test.go` | **landed in `#23 M1`** | the live CDN |

- **`-lang` flag** — selects the language; empty means unspecified.
  - **Injected into:** `options.voice`, then `speak`, then `AudioCandidates`. The pure builder never reads a flag.
  - **Future extensions:** a per-deck default from `#18 M2` supersedes the flag's default without changing its meaning.

- **`fakeCDN`** — the existing stateful fake. **Reused, not extended**: it already records every requested path in order, which is exactly what proves a Spanish word's walk skips the English-only legacy path. No new fake is needed and adding one would be the near-fit double `#6 BR-43` warns about.
  - **State model:** a set of present paths plus an ordered request log.

- **`TestCDNStillServesSpanishOnTheExpectedPaths`** — live conformance beside `TestCDNStillServesTheExpectedPaths`, pinning the facts the gate rests on. Landed in `#23 M1` rather than here, since `#23` needed the same measurement.
  - **Cadence:** on-demand with the rest of `-tags conformance`; routes its dependency probe through `conformance.SkipOrFail` (`#25`) so an unreachable CDN SKIPS by default and FAILS under `CONFORMANCE_STRICT`.

---

## Chunk 1: the pure core

### Task 1: `voice` and the ordering policy

**Files:**
- Create: `cmd/define/voice.go`
- Test: `cmd/define/voice_test.go`

- [ ] **Step 1: Write the failing test**

```go
package main

import (
	"slices"
	"testing"
)

// The ordering policy, as a table over the cases that actually occur.
//
// Each row is justified by a measurement recorded in the plan, not by taste:
// languages are disjoint on the CDN, so a second language is safe to try; a 404
// costs ~10x a hit, so the order is a performance decision; and the legacy path
// is English-only, so Spanish never belongs on it.
func TestVoicesOrdering(t *testing.T) {
	for _, tc := range []struct {
		name string
		want voice
		got  []voice
	}{
		{
			// Today's behaviour must not regress: an unspecified language asks
			// English FIRST, so the existing corpus still hits on candidate 1.
			name: "unspecified tries English first, then Spanish",
			want: voice{Lang: "", Locale: "us"},
			got:  []voice{{Lang: "en", Locale: "us"}, {Lang: "es", Locale: "us"}},
		},
		{
			// -lang reorders rather than restricts: a mis-tagged word still
			// resolves, it just pays for the miss.
			name: "an explicit language goes first, the other still follows",
			want: voice{Lang: "es", Locale: "es"},
			got:  []voice{{Lang: "es", Locale: "es"}, {Lang: "en", Locale: "es"}},
		},
		{
			name: "an explicit English is the plain case",
			want: voice{Lang: "en", Locale: "gb"},
			got:  []voice{{Lang: "en", Locale: "gb"}, {Lang: "es", Locale: "gb"}},
		},
		{
			// An empty locale must not spell "_en__1".
			name: "an empty locale defaults to us",
			want: voice{Lang: "en", Locale: ""},
			got:  []voice{{Lang: "en", Locale: "us"}, {Lang: "es", Locale: "us"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := voices(tc.want); !slices.Equal(got, tc.got) {
				t.Errorf("voices(%+v) =\n  %+v\nwant\n  %+v", tc.want, got, tc.got)
			}
		})
	}
}

// An unknown language is passed through rather than rejected.
//
// The CDN answers 404 for a language it does not have, and the fetch loop
// already degrades to a warning with exit 0. Rejecting here would mean
// maintaining a list of "real" languages that goes stale the moment Google adds
// one — a restatement of a fact the CDN owns.
func TestVoicesPassesAnUnknownLanguageThrough(t *testing.T) {
	got := voices(voice{Lang: "de", Locale: "de"})
	if len(got) == 0 || got[0].Lang != "de" {
		t.Errorf("voices() = %+v, want an unknown language tried first", got)
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define/ -run TestVoices -v`
Expected: FAIL — `undefined: voice`, `undefined: voices`

- [ ] **Step 3: Write the minimal implementation**

```go
package main

// voice is the language and regional variant a recording is asked for.
//
// Two fields rather than two positional strings because "es" is a valid value of
// BOTH — AudioCandidates(w, "es", "es") and a transposed AudioCandidates(w,
// "es", "en") are indistinguishable at a call site, and there are 20+ of them in
// tests. voice{Lang: "es", Locale: "es"} cannot be written backwards.
type voice struct {
	// Lang is the CDN's language segment: "en", "es". Empty means unspecified,
	// which asks several — see voices.
	Lang string
	// Locale is the regional variant: "us", "gb" for English; "es", "us" for
	// Spanish. Empty defaults to "us".
	//
	// The Spanish pair is a PHONEMIC split, not an accent flavour: es_es is
	// Castilian (cazar /θ/ != casar /s/) and es_us is Latin American seseo (both
	// /s/). Choosing one chooses which sound system a learner acquires, which is
	// why the flag's help says so rather than naming two country codes.
	// Measured: es_419, es_mx and es_ar are all 404.
	Locale string
}

// defaultLocale is what an unspecified locale means. "us" for both languages,
// matching -locale's existing default, so the flag means the same thing whatever
// the language is.
const defaultLocale = "us"

// languageOrder is every language the CDN is asked for, most likely first.
//
// English leads because it is the existing corpus and today's only behaviour;
// an unspecified word must still hit on the first candidate, which measurement
// makes a real requirement rather than a preference: a 404 costs ~300-600ms
// against ~40ms for a hit.
var languageOrder = []string{"en", "es"}

// voices is the ORDERING POLICY: which languages to ask, in what order.
//
// Trying a second language is SAFE because the languages are disjoint on the
// CDN — measured, sycophantic_es_es_1 is 404 and madrugar_en_us_1 is 404 — so a
// fallback can never return the wrong word's recording. It is not free, though,
// which is why an explicit -lang REORDERS rather than merely appends: a Spanish
// word under -lang es hits on candidate 1 instead of paying ~1s of English
// misses first.
func voices(want voice) []voice {
	if want.Locale == "" {
		want.Locale = defaultLocale
	}

	out := []voice{}
	seen := map[string]bool{}
	if want.Lang != "" {
		out = append(out, voice{Lang: want.Lang, Locale: want.Locale})
		seen[want.Lang] = true
	}
	for _, l := range languageOrder {
		if !seen[l] {
			out = append(out, voice{Lang: l, Locale: want.Locale})
		}
	}
	return out
}
```

- [ ] **Step 4: Run the tests and watch them pass**

Run: `go test ./cmd/define/ -run TestVoices -v`
Expected: PASS, all five subtests.

- [ ] **Step 5: Mutation-check that the ordering test actually bites**

Run each, confirm it reddens a NAMED test, then restore:

```sh
cp cmd/define/voice.go "$TMPDIR/voice.base"
# 1. Reverse the language order — the unspecified row must fail.
sed -i '' 's/{"en", "es"}/{"es", "en"}/' cmd/define/voice.go
go test ./cmd/define/ -run TestVoices -v 2>&1 | grep -E '^\s+--- FAIL'
cp "$TMPDIR/voice.base" cmd/define/voice.go
# 2. Ignore the explicit language — the reorder row must fail.
sed -i '' 's/if want.Lang != "" {/if false {/' cmd/define/voice.go
go test ./cmd/define/ -run TestVoices -v 2>&1 | grep -E '^\s+--- FAIL'
cp "$TMPDIR/voice.base" cmd/define/voice.go
# 3. Drop the locale default — the empty-locale row must fail.
sed -i '' 's/want.Locale = defaultLocale/_ = defaultLocale/' cmd/define/voice.go
go test ./cmd/define/ -run TestVoices -v 2>&1 | grep -E '^\s+--- FAIL'
cp "$TMPDIR/voice.base" cmd/define/voice.go
```

Expected: each mutation names at least one failing subtest. **A mutation that reddens nothing is a finding, not a pass** — and the likelier explanation is that the `sed` did not match, so confirm the file actually changed before believing a zero (`workshop/lessons.md`, *Mutation testing needs a COMMITTED baseline*).

- [ ] **Step 6: Commit**

```bash
git add cmd/define/voice.go cmd/define/voice_test.go
git commit -m "#27: voice and the ordering policy, with the measurements behind it"
```

### Task 2: `AudioCandidates` takes a `voice`

**Files:**
- Modify: `cmd/define/audiourl.go`
- Modify: `cmd/define/audiourl_test.go`

- [ ] **Step 1: Write the failing test**

Replace `TestAudioCandidatesOrder` and `TestAudioCandidatesLocale`; keep the short-word and multi-word tests, updating their calls.

```go
// The full ordered list for an unspecified language.
//
// Spanish sits between the modern English candidates and the legacy ones, and
// the legacy path carries NO Spanish entry at all — measured, madrugar--_us_1
// and madrugar--_es_1 are both 404 while sycophantic--_us_1 is 200. Emitting
// Spanish legacy candidates would be two guaranteed misses at ~450ms each.
func TestAudioCandidatesOrder(t *testing.T) {
	got := AudioCandidates("Sycophantic", voice{}) // input case must not matter
	want := []string{
		audioBase + "/pronunciation/2022-03-02/audio/sy/sycophantic_en_us_1.mp3",
		audioBase + "/pronunciation/2022-03-02/audio/sy/sycophantic_en_us_2.mp3",
		audioBase + "/pronunciation/2022-03-02/audio/sy/sycophantic_es_us_1.mp3",
		audioBase + "/pronunciation/2022-03-02/audio/sy/sycophantic_es_us_2.mp3",
		audioBase + "/sounds/oxford/sycophantic--_us_1.mp3",
		audioBase + "/sounds/oxford/sycophantic--_us_2.mp3",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// -lang es puts Spanish first, and the English legacy fallback still trails.
func TestAudioCandidatesSpanishFirst(t *testing.T) {
	got := AudioCandidates("madrugar", voice{Lang: "es", Locale: "es"})
	if len(got) == 0 || !strings.HasSuffix(got[0], "/ma/madrugar_es_es_1.mp3") {
		t.Fatalf("first candidate = %q, want the Castilian recording", got[0])
	}
	// The legacy path is English-only; no Spanish candidate may appear on it.
	for _, u := range got {
		if strings.Contains(u, "/sounds/oxford/") && strings.Contains(u, "_es_") {
			t.Errorf("Spanish on the English-only legacy path: %s", u)
		}
	}
}

// Exactly one legacy pair, regardless of how many languages are tried.
func TestAudioCandidatesEmitsLegacyOnceForEnglishOnly(t *testing.T) {
	var legacy int
	for _, u := range AudioCandidates("sycophantic", voice{}) {
		if strings.Contains(u, "/sounds/oxford/") {
			legacy++
		}
	}
	if legacy != 2 {
		t.Errorf("got %d legacy candidates, want 2 — one pair, English only", legacy)
	}
}

func TestAudioCandidatesLocale(t *testing.T) {
	for _, u := range AudioCandidates("sycophantic", voice{Lang: "en", Locale: "gb"}) {
		if !strings.Contains(u, "_gb_") && !strings.Contains(u, "--_gb_") {
			t.Errorf("locale not applied: %s", u)
		}
	}
	// An empty locale defaults to us rather than producing "_en__1".
	for _, u := range AudioCandidates("sycophantic", voice{}) {
		if strings.Contains(u, "__") {
			t.Errorf("empty locale leaked into %s", u)
		}
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define/ -run TestAudioCandidates -v`
Expected: FAIL — `cannot use voice{} (value of type voice) as string`.

- [ ] **Step 3: Rewrite `AudioCandidates`**

```go
// AudioCandidates returns the CDN URLs to try, in order, for a word's recorded
// pronunciation. Pure and offline — no request is made here, so the ordering is
// unit-testable without a server.
//
// The order comes from measurement, and the facts have grown since the first
// version (which wrote the language as the literal "_en_", so a Spanish word was
// only ever asked for as English):
//
//   - The 2022 generation strictly dominates the legacy sounds/oxford paths, so
//     modern candidates lead. Re-measured 2026-08-28: the modern path now serves
//     every sampled word including defenestrate, so legacy is pure fallback.
//   - The legacy path is ENGLISH-ONLY — madrugar--_us_1 and madrugar--_es_1 are
//     both 404 while sycophantic--_us_1 is 200 — so it is emitted once, for
//     English, whatever languages are tried.
//   - Languages are DISJOINT, so a fallback language can never return the wrong
//     word's recording; and a 404 costs ~10x a hit, so which language leads is a
//     performance decision. voices owns that.
func AudioCandidates(word string, v voice) []string {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return nil
	}
	// Multi-word headwords are spelled with underscores on the CDN.
	slug := strings.ReplaceAll(word, " ", "_")
	esc := url.PathEscape(slug)

	// The shard is the first two letters — or one, for a single-letter word.
	shard := esc
	if r := []rune(slug); len(r) >= 2 {
		shard = url.PathEscape(string(r[:2]))
	}

	var out []string
	var locale string
	for _, want := range voices(v) {
		locale = want.Locale // every voice carries the same locale; kept for the legacy path
		for _, n := range []string{"1", "2"} {
			out = append(out, audioBase+"/pronunciation/2022-03-02/audio/"+shard+"/"+
				esc+"_"+want.Lang+"_"+want.Locale+"_"+n+".mp3")
		}
	}
	// The legacy pair, LAST and English-only. Not inside the loop above: it has
	// no language segment to vary, so one pair is all there is.
	for _, n := range []string{"1", "2"} {
		out = append(out, audioBase+"/sounds/oxford/"+esc+"--_"+locale+"_"+n+".mp3")
	}
	return out
}
```

- [ ] **Step 4: Fix every call site**

`AudioCandidates` has call sites in `main.go`, `main_test.go`, `player_conformance_test.go`, `fetch_conformance_test.go`. Enumerate them rather than remembering:

```sh
grep -rn "AudioCandidates(" --include='*.go' . | grep -v "^./cmd/define/audiourl.go"
```

Each `AudioCandidates(w, "us")` becomes `AudioCandidates(w, voice{})`; `speak`'s `locale string` parameter becomes `v voice`.

- [ ] **Step 5: Run the whole package**

Run: `go test ./cmd/define/ && go vet -tags conformance ./cmd/define/`
Expected: PASS, and vet clean under both tag sets.

- [ ] **Step 6: Commit**

```bash
git add cmd/define/audiourl.go cmd/define/audiourl_test.go cmd/define/main.go cmd/define/main_test.go
git commit -m "#27: AudioCandidates takes a voice; legacy stays English-only"
```

## Chunk 2: the flag, the fake, and the live check

### Task 3: the `-lang` flag and its help text

**Files:**
- Modify: `cmd/define/main.go` (flag block ~:272, `options` ~:237)
- Test: `cmd/define/main_test.go`

- [ ] **Step 1: Write the failing test**

```go
// The help text must say what the Spanish variants ARE, not name two country
// codes. The choice is phonemic — it decides which sound system a learner
// acquires — and "es or us" tells a learner nothing they can act on.
func TestLangHelpExplainsTheVariants(t *testing.T) {
	// run(ctx, args, deps, stdin, stdout, stderr) — signature checked at
	// main.go:261; fs.Usage writes to stderr (main.go:279).
	d := testDeps(t)
	var out, errb bytes.Buffer
	run(t.Context(), []string{"-h"}, d, strings.NewReader(""), &out, &errb)
	help := out.String() + errb.String()

	for _, want := range []string{"-lang", "seseo"} {
		if !strings.Contains(help, want) {
			t.Errorf("help does not mention %q:\n%s", want, help)
		}
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define/ -run TestLangHelp -v`
Expected: FAIL — help mentions neither.

- [ ] **Step 3: Add the flag**

In the flag block, beside `-locale`:

```go
lang := fs.String("lang", "", "pronunciation language: en, es (default: try en then es)")
locale := fs.String("locale", "us", "pronunciation variant: en → us|gb; es → es (Castilian, cazar /θ/) | us (seseo, both /s/)")
```

Add `voice voice` to `options`, drop the `locale string` field, and set it where options are built:

```go
opt.voice = voice{Lang: *lang, Locale: *locale}
```

- [ ] **Step 4: Run it and watch it pass**

Run: `go test ./cmd/define/ -run TestLangHelp -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/define/main.go cmd/define/main_test.go
git commit -m "#27: -lang, and a -locale help line that says what the choice is"
```

### Task 4: the walk skips what cannot exist (fake-backed)

**Files:**
- Test: `cmd/define/fetch_fake_test.go`

- [ ] **Step 1: Write the failing test**

The existing `fakeCDN` records requested paths in order — reuse it rather than
adding a near-fit double (`#6 BR-43`).

```go
// A Spanish word's walk asks for Spanish paths and NEVER for a Spanish legacy
// path, which is measured not to exist.
//
// This is the fake earning its place over a function-call mock: the assertion is
// about the ORDER and MEMBERSHIP of the requests, which only a stateful double
// can show.
func TestSpanishWalkSkipsTheEnglishOnlyLegacyPath(t *testing.T) {
	want := "/pronunciation/2022-03-02/audio/ma/madrugar_es_es_1.mp3"
	cdn := newFakeCDN(t, map[string][]byte{want: []byte("audio")})

	// AudioCandidates builds absolute gstatic URLs; the fake serves paths. Strip
	// the real host and re-point at the fake with the helpers this file already
	// has — stripHost, urls, source. VERIFIED present at fetch_fake_test.go:50-64;
	// httpAudioSource has no base field, only a client.
	var paths []string
	for _, u := range AudioCandidates("madrugar", voice{Lang: "es", Locale: "es"}) {
		paths = append(paths, stripHost(t, u, audioBase))
	}
	_, from, err := cdn.source().Fetch(t.Context(), cdn.urls(paths...))
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !strings.HasSuffix(from, want) {
		t.Errorf("fetched %q, want %q", from, want)
	}
	for _, p := range cdn.Requested() {
		if strings.Contains(p, "/sounds/oxford/") && strings.Contains(p, "_es_") {
			t.Errorf("asked the CDN for a Spanish legacy path that cannot exist: %s", p)
		}
	}
	// -lang es must HIT FIRST: the whole point of reordering is not paying for
	// four English misses at ~450ms each.
	if got := cdn.Requested(); len(got) != 1 {
		t.Errorf("made %d requests before the hit, want 1:\n%v", len(got), got)
	}
}
```

- [ ] **Step 2: Run, implement any needed helper, re-run**

Run: `go test ./cmd/define/ -run TestSpanishWalk -v`
No new helper is needed: `stripHost`, `(*fakeCDN).urls` and `(*fakeCDN).source`
already exist (`fetch_fake_test.go:50-64`) and were checked before this task was
written, not assumed.

- [ ] **Step 3: Commit**

```bash
git add cmd/define/fetch_fake_test.go
git commit -m "#27: the Spanish walk hits first and never asks for a legacy Spanish path"
```

### Task 5: live conformance for the Spanish paths

**Files:**
- Modify: `cmd/define/fetch_conformance_test.go`

- [ ] **Step 1: Write the test**

```go
// The facts the Spanish ordering rests on, measured against the live CDN.
//
// Beside TestCDNStillServesTheExpectedPaths, which pins the English ordering the
// same way. Both route their dependency probe through conformance.SkipOrFail
// (#25): an unreachable CDN SKIPS by default and FAILS under CONFORMANCE_STRICT,
// so a sandboxed run cannot report green for a check that never ran.
func TestCDNServesSpanish(t *testing.T) {
	// Fact 1: the Spanish path serves a word the English path does not.
	es := AudioCandidates("madrugar", voice{Lang: "es", Locale: "es"})[0]
	if got := head(t, es); got != http.StatusOK {
		t.Errorf("Castilian madrugar = %d, want 200 — %s", got, es)
	}
	en := AudioCandidates("madrugar", voice{Lang: "en", Locale: "us"})[0]
	if got := head(t, en); got != http.StatusNotFound {
		t.Errorf("English madrugar = %d, want 404 — if this became 200 the "+
			"languages are no longer disjoint and the fallback could return the "+
			"wrong recording", got)
	}

	// Fact 2: languages are DISJOINT in both directions. This is what makes
	// trying a second language safe rather than merely unhelpful.
	syEs := AudioCandidates("sycophantic", voice{Lang: "es", Locale: "es"})[0]
	if got := head(t, syEs); got != http.StatusNotFound {
		t.Errorf("Spanish sycophantic = %d, want 404 — %s", got, syEs)
	}

	// Fact 3: the legacy path is English-only, which is why AudioCandidates
	// emits no Spanish candidate on it.
	legacy := audioBase + "/sounds/oxford/madrugar--_es_1.mp3"
	if got := head(t, legacy); got != http.StatusNotFound {
		t.Errorf("legacy Spanish = %d, want 404 — the legacy path grew a language "+
			"segment and AudioCandidates should now use it", got)
	}
}
```

- [ ] **Step 2: Run it UNSANDBOXED, in both modes**

```sh
go test -tags conformance ./cmd/define/ -run 'CDN' -v
CONFORMANCE_STRICT=1 go test -tags conformance ./cmd/define/ -run 'CDN' -v
```

Expected: PASS unsandboxed. Sandboxed it must SKIP by default and FAIL under strict — that pairing is the `#25` guarantee and is worth confirming here rather than assuming.

- [ ] **Step 3: Commit**

```bash
git add cmd/define/fetch_conformance_test.go
git commit -m "#27: live conformance for the Spanish paths and the disjointness the fallback rests on"
```

### Task 6: a Spanish entry has no pronunciation, and that is correct

**Files:**
- Modify: `cmd/define/render_test.go` (or the nearest fixture-backed render test)

- [ ] **Step 1: Write the test**

```go
// Spanish orthography is phonemic — spelling plus the written accent determines
// pronunciation exactly — so Spanish dictionary entries carry NO phonetic
// notation, unlike NOAD's "lig·a·ment | ˈliɡəmənt |".
//
// Verified on the live dictionary: `define madrugar` renders no pronunciation
// line while `define ligament` renders /ˈliɡəmənt/. This test exists so the
// absence reads as EXPECTED rather than as a gap someone later "fixes" —
// and it is why the recording is more load-bearing for Spanish than English:
// it is the only place the information exists.
func TestASpanishEntryRendersNoPronunciation(t *testing.T) {
	// Uses the committed Spanish fixture; an absent fixture is a deleted file,
	// not a missing dependency, so this FAILS rather than skipping (#25's
	// second class).
	raw, err := testDict(t).Lookup("madrugar")
	if err != nil {
		t.Fatalf("fixture absent for madrugar: %v — testdata/entries is committed", err)
	}
	if got := ParseEntry(raw).IPA; got != "" {
		t.Errorf("IPA = %q, want empty — Spanish entries carry no "+
			"phonetic notation, and inventing one would be wrong, not helpful", got)
	}
}
```

- [ ] **Step 2: Capture the fixture**

`madrugar` is not in `testdata/entries`. Capture it the way the corpus is captured:

```sh
grep -n "madrugar" cmd/define/testdata/entries/* 2>/dev/null || \
  echo "capture needed — see cmd/define/testdata/capture.sh"
```

Run `capture.sh` (unsandboxed; it enforces a byte floor precisely because a
sandboxed `DCSCopyTextDefinition` returns silence rather than an error).

- [ ] **Step 3: Run, then commit**

```bash
go test ./cmd/define/ -run TestASpanishEntry -v
git add cmd/define/render_test.go cmd/define/testdata/entries/
git commit -m "#27: a Spanish entry has no pronunciation, and a test says so"
```

### Task 7: docs

**Files:**
- Modify: `README.md`
- Modify: `atlas/define.md`

- [ ] **Step 1: Build the site list by running the sweep, not from memory**

```sh
grep -rniE "locale|pronunciation" README.md atlas/ --include='*.md'
```

`#24 BR-1`'s rule: the enumeration is the deliverable, and the exclude list is
part of it. Recurse over directories; do not use a `*.md` glob.

- [ ] **Step 2: Write the changes**

- README: `-lang` beside `-locale`, and the θ/seseo sentence.
- atlas: a short paragraph under the audio section — language is a parameter,
  `voices` owns the ordering, the legacy path is English-only, and the three
  measured facts with their dates.

- [ ] **Step 3: Verify the sweep is clean, then commit**

```bash
grep -rn "us or gb" README.md atlas/   # the old -locale help wording must be gone
git add README.md atlas/define.md
git commit -m "#27: docs — language as a parameter, and what the Spanish variants are"
```

## Risks

**The unspecified-language path costs a Spanish learner ~1s per word.** With no
`-lang`, a Spanish word pays two English 404s (~900ms) before its first Spanish
candidate. That is the measured price of "reorders rather than restricts", and
it is why `-lang` exists. It is acceptable *now* because the deck is
English-dominant; it stops being acceptable when `#18 M2` gives the deck a
language, at which point `voices` should read the word's own language instead of
guessing. Recorded here so the next issue meets a decision rather than a mystery.

**`-locale gb` with `-lang es` produces `es_gb`, which 404s.** Deliberately not
validated: a table of valid language/locale pairs is a restatement of a fact the
CDN owns, and it goes stale the moment Google adds a variant. The failure is a
warning and exit 0, which is the existing behaviour for any missing recording.
If this proves confusing in use, the fix is a warning naming the pair, not a
rejection.

**The legacy path could grow a language segment.** `TestCDNServesSpanish`'s third
fact is what would catch it, and its failure message says what to do.

## Done-when → task map

| Done-when row | Task |
|---|---|
| `define madrugar` plays a recording; asserted against the fake | 4 |
| conformance test measures the live CDN like the English ordering | 5 |
| `-lang` selects language, `-locale` still selects variant | 3 |
| help says what the difference *is* (θ vs seseo) | 3 |
| an English word with no recording still warns, exit 0 | 2 (no change to the fetch loop; `go test ./cmd/define/` covers it) |
| no pronunciation for a Spanish entry, and a test says that is expected | 6 |
| new conformance assertions route through `conformance.SkipOrFail` | 5 |

## Notes for the reviewer

- **Every measurement in this plan was re-run on 2026-08-28**, not carried from the
  issue's 2026-08-22 session. Three findings are new and changed the design:
  languages are disjoint, the legacy path is English-only, and a 404 costs ~10× a
  hit. The first makes the fallback safe, the second removes two guaranteed
  misses per Spanish word, and the third turns ordering into a performance
  decision. `#17 BR-21` is the reason for re-running rather than citing.
- **`voice` is justified by a mix-up hazard, not by duplication.** If that reads as
  over-engineering for two strings, the counter-argument is that `"es"` is a legal
  value of both fields and there are 20+ call sites; the struct makes the
  transposition unspellable. I would rather defend this than a third positional
  string.
- **No new fake.** `fakeCDN` already records ordered requests, which is exactly
  what Task 4 asserts.
- **Every symbol this plan names was checked against the tree before the plan was
  saved**, after two of them turned out to be wrong on the first pass:
  `httpAudioSource` has no `base` field (only `client`), and the entry's
  pronunciation field is `IPA`, not `Pronunciation`. A plan that spells a
  non-existent symbol costs a gate round; `#21`'s lesson is that line numbers and
  API names in a plan are code nothing compiles.

---

## Revisions

### 2026-08-28 — the multi-language fallback is deleted; `#23`'s language mode replaces it

**Reason.** Operator, before implementation started:

> *we will split `define` into language specific thing, so every `define`
> invocation will operate in 1 language only. it can switch in a TUI app, but at
> any given time one language only. make this change first.*

That design is `#23`, filed the day before with the same words, and this plan
was written without reconciling against it. **The plan's central mechanism was
designed for a question the mode answers.**

**Delta — what is now wrong above:**

- **`voices()` is deleted, not adjusted.** Its whole job was ordering an
  unspecified language across `en` then `es`. Under a mode there is no
  unspecified language: the invocation has exactly one, either persisted in the
  vocab directory or given by `-lang`. Task 1's table, its three mutations, and
  the `languageOrder` var all go with it.
- **The ~900ms fallback cost in Risks stops existing.** It was the price of
  guessing; a mode does not guess. The Risks entry predicting it "stops being
  acceptable when `#18 M2` gives the deck a language" was directionally right and
  arrived at the wrong remedy — the answer was not a better guess, it was not
  guessing.
- **`AudioCandidates` builds candidates for ONE language.** Same signature
  (`word string, v voice`), simpler body: one modern pair for `v.Lang`, plus the
  legacy pair only when `v.Lang == "en"`.
- **The disjointness measurement changes role, not truth.** It justified the
  fallback being *safe*; with no fallback it instead justifies a mode being
  *sufficient* — asking the wrong language returns nothing rather than the wrong
  word's audio, so a mis-set mode is a visible miss, not a silent wrong answer.

**What survives unchanged, and is the reusable part.** Every measurement in
*"Measurements this plan rests on"*, re-run 2026-08-28. Three are load-bearing
for `#23`'s audio task whatever the sequencing:

1. languages are **disjoint** on the CDN,
2. the legacy `/sounds/oxford/` path is **English-only** (so Spanish candidates
   on it are guaranteed waste),
3. a 404 costs **~10×** a hit (~300–600ms vs ~40ms).

Also surviving: the `voice{Lang, Locale}` type and its mix-up argument, the
θ/seseo semantics of the Spanish locale pair, the `fakeCDN`-reuse decision, and
the "a Spanish entry has no IPA, and that is correct" test.

**Sequencing (operator's call).** `#23` end-to-end first, with the audio language
plumbing as one task inside it. `#27` is `blocked` on `#23`; what plausibly
remains here afterwards is the *variant* half — `es_es` vs `es_us`, the help text
that says what the choice is, and the live CDN conformance — reassessed at
`#23`'s close rather than assumed now.

**The process lesson, since this is the second time today.** This plan reached
727 lines before anything reconciled it against the open issue that already
specified the design. `#23` was on the board, was filed from the operator's own
words, and its Spec says *"everything inherits the mode … the audio asks for that
language's recording"*. The `sdlc state` board was read at the start of this
session and the overlap still went unnoticed, because I read `#27` and the code
and never re-read the neighbours it names. **Before planning an issue, read the
issues it touches** — `#27`'s own Log named `#18` and `#26`, and `#23` was one
`grep -l lang workshop/issues/` away.

## Revisions

### 2026-08-28 — two rows this plan named were superseded by `#23`

**Reason.** `#23`'s close review found the symbol half of the artifact-name rule
unenforced for the fourth time, and the fix — a guard asserting every
Core-concepts row names an entity the tree actually has — flagged two rows here.

**Delta.** `voices` was DELETED rather than adapted: `#23` made the language a
declared mode, so the ordering policy this plan designed has nothing left to
order. `TestCDNServesSpanish` shipped as
`TestCDNStillServesSpanishOnTheExpectedPaths` in `#23 M1`, because `#23` needed
the same measurement. Both rows now say so instead of naming absent entities.
