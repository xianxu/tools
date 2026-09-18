# Light/Dark Colour Scheme Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `define` paints its target-language tint in the shade that suits the terminal's background — detected in a session, overridable by `-scheme` and a saved `/scheme` — and a switch recolours everything already on screen.

**Architecture:** A row records only WHETHER it is tinted (a role); the shade resolves at paint from ONE per-process scheme holder (`atomic.Pointer` to an immutable `schemeState`, pure transitions). The saved choice is a one-word file under the user's config directory. Detection is an `OSC 11` query sent at raw-mode entry whose reply the key decoder turns into a `KeyBackground` event — nothing waits for it.

**Tech Stack:** Go 1.26, `golang.org/x/term`, `sync/atomic`, `github.com/creack/pty` (conformance only).

**Spec:** `workshop/issues/000070-light-dark-colour-scheme.md` — `## Spec` and `## Done when` are the contract; the Log's 2026-09-17 plan-review entry records where this plan revises the spec.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Scheme` | `cmd/define/store/scheme.go` | new |
| `ParseScheme` | `cmd/define/store/scheme.go` | new |
| `schemeState` | `cmd/define/scheme.go` | new |
| `schemeSource` | `cmd/define/scheme.go` | new |
| `schemeArg` | `cmd/define/scheme.go` | new |
| `parseSchemeArg` | `cmd/define/scheme.go` | new |
| `parseTintFlag` | `cmd/define/scheme.go` | new |
| `schemeTint` | `cmd/define/language_style.go` | new |
| `rowPaint` | `cmd/define/output_layout.go` | modified |
| `tintPolicy` | `cmd/define/language_style.go` | modified |
| `PresentationRegion` | `cmd/define/play/presentation.go` | modified |
| `paintLanguageRow` | `cmd/define/language_row.go` | modified |
| `describeScheme` (M2) | `cmd/define/scheme.go` | new |
| `applyScheme` (M2) | `cmd/define/scheme.go` | new |
| `configDirFrom` (M2) | `cmd/define/scheme.go` | new |
| `runScheme` (M2) | `cmd/define/scheme_cmd.go` | new |
| `commandCtx` (M2) | `cmd/define/command.go` | modified |
| `parseBackgroundColour` (M3) | `cmd/define/scheme_detect.go` | new |
| `decodeOSC` (M3) | `cmd/define/key.go` | new |
| `KeyBackground` (M3) | `cmd/define/key.go` | new |
| `styleLanguageText` | `cmd/define/language_style.go` | deleted |
| `lineInkBounds` | `cmd/define/language_style.go` | deleted |
| `validateLanguageText` | `cmd/define/language_text.go` | deleted |
| `dictionaryFragment` | `cmd/define/dictionary_language.go` | deleted |
| `dictionaryText` | `cmd/define/dictionary_language.go` | deleted |
| `styledBoardPrompt` | `cmd/define/practice_language.go` | deleted |
| `TestLanguageTintStyle` | `cmd/define/language_style_test.go` | deleted |
| `TestLanguageTintMixedAndSelection` | `cmd/define/language_style_test.go` | deleted |
| `TestLanguageTextValidation` | `cmd/define/language_style_test.go` | deleted |
| `TestDictionaryCapturedMixedOwnership` | `cmd/define/dictionary_language_test.go` | deleted |
| `TestDictionaryInlinePronunciationRemainsNeutral` | `cmd/define/dictionary_language_test.go` | deleted |
| `tintProfile` | `cmd/define/language_style.go` | deleted |
| `TestLanguageTintProfile` | `cmd/define/language_style_test.go` | deleted |
| `boardFooter` | `cmd/define/render_helpers_test.go` | deleted |
| `practiceChrome` | `cmd/define/render_helpers_test.go` | deleted |

Rows for DELETED symbols are added by the task that deletes them, in the same commit (Task 3): a `| deleted |` row asserts the symbol is already gone (`TestPlanTablesNameEntitiesThatExist`), and it is also what exempts this plan's prose from `TestARemovedDeclarationIsSweptOrRetired`.

- **`Scheme`** — the closed enum naming the terminal background, as persisted. One enum for the file, the flag, the command and the decoder, on `store.Lang`'s precedent (the persisted enum lives in `store`). A third value widens here and in `schemeTint`.
- **`schemeState`** — immutable: an explicit choice (source flag, saved or session) and a detected value; `effective()` = choice, else detected, else dark, plus the source. All transitions return a new value (ARCH-ORDER). DEC mode 2031 notifications would be one more `withDetected` event.
- **`applyScheme`** — `/scheme`'s transition, persisting first (the `/bilingual` rule): the holder, the parsed arg, a `schemePersister` (nil = nowhere to save), and whether a session exists.
- **`parseBackgroundColour`** — `rgb:R/G/B` (1–4 hex digits each) → Rec. 601 luma → dark or light.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `schemeHolder` | `cmd/define/scheme.go` | new | cross-goroutine state |
| `deps` | `cmd/define/main.go` | modified | gains `scheme` (M1) and `configDir` (M2) |
| `screen` | `cmd/define/screen.go` | modified | gains the holder |
| `newConsole` | `cmd/define/replraw.go` | modified | attaches the holder (M1); asks the question (M3) |
| `ReadScheme` (M2) | `cmd/define/store/scheme.go` | new | filesystem |
| `WriteScheme` (M2) | `cmd/define/store/scheme.go` | new | filesystem |
| `ClearScheme` (M2) | `cmd/define/store/scheme.go` | new | filesystem |
| `schemePersister` (M2) | `cmd/define/scheme.go` | new | the store functions |
| `backgroundQuery` (M3) | `cmd/define/rawterm.go` | new | the terminal |
| `terminalQueries` (M3) | `cmd/define/rawterm.go` | new | the terminal |

- **`schemeHolder`** — `atomic.Pointer[schemeState]`. ONE writer at a time (the loop goroutine in force; a `/play` sitting runs on the editor loop's goroutine); five painting goroutines read. Nil holder = dark, read-only. Built by `run()`, attached to every production screen (`liveScreen.attachScheme`), carried by `tintPolicy` to the non-screen writer.
- **`schemePersister`** — `save`, `clear`. Production: `dirSchemePersister` over the store functions. Tests: a stateful `fakePersister` (holds the saved value, can fail). The filesystem half is covered by `store` tests on `t.TempDir()` (ARCH-MOCK).
- **The terminal (M3)** — the external dependency. Seam: bytes in (`readInput`'s reader) and bytes out (`rawSession.control`). Fake: in-process tests feed reply BYTES through `readInput`; pty conformance plays a light, a dark and a silent terminal. Live conformance: the manual check in Task 18.

### Conventions for every task

- Run tests from the repo root: `go test ./cmd/define/... -count=1`. After any task touching a tagged file, also `go vet -tags conformance ./cmd/define` — the default build never compiles `//go:build darwin && conformance` files (lessons: "Name the suite a swept file actually runs in").
- **Every commit is green.** No step commits a known failure.
- **Mutation checks** follow lessons.md "Mutation testing needs a COMMITTED baseline": the task's commit comes FIRST, then mutate one thing, run with `-count=1`, confirm the mutation APPLIED and COMPILED, see red, restore with `git checkout HEAD -- <path>`. One mutation, one change. A mutation that stays green means the test is blind: fix the test and commit that.
- Tests not yet written are named in bold, never in backticks — `TestPlanCitesTestsThatExist` reads backticked `Test*` names as claims that they exist.
- Commits: `#70 Mx: <subject>`, body = why, trailer `Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`. Read `git show --stat` before each commit.

---

## Chunk 1: M1 — the tint is a role, the scheme is state

Behaviour after M1: identical to today except the flags — `-scheme dark|light|auto` picks the shade (auto means dark until M3 detects), `-language-tint on|off`, and `-language-tint dark|light` is refused naming `-scheme`.

### Task 1: `Scheme`

**Files:** Create `cmd/define/store/scheme.go`; test `cmd/define/store/scheme_test.go`.

- [x] **Step 1: Write the failing test** **TestParseScheme**:

```go
package store

import "testing"

func TestParseScheme(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Scheme
		ok   bool
	}{
		{"dark", SchemeDark, true}, {"light", SchemeLight, true},
		{" Light\n", SchemeLight, true}, {"DARK", SchemeDark, true},
		{"", "", false}, {"auto", "", false}, {"solarized", "", false},
	} {
		got, err := ParseScheme(tc.in)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("ParseScheme(%q) = %q, %v; want %q, ok=%v", tc.in, got, err, tc.want, tc.ok)
		}
	}
}
```

- [x] **Step 2: Run — expect FAIL** (`undefined: ParseScheme`): `go test ./cmd/define/store -run TestParseScheme -count=1`
- [x] **Step 3: Implement**

```go
package store

import (
	"fmt"
	"strings"
)

// Scheme names the terminal's background. A closed set: anything else read from
// a file, a flag or a command is refused at the boundary (ARCH-SECURE), so every
// value past ParseScheme is one of these two.
type Scheme string

const (
	SchemeDark  Scheme = "dark"
	SchemeLight Scheme = "light"
)

// ParseScheme accepts either value, trimmed and in any case.
func ParseScheme(s string) (Scheme, error) {
	switch v := Scheme(strings.ToLower(strings.TrimSpace(s))); v {
	case SchemeDark, SchemeLight:
		return v, nil
	}
	return "", fmt.Errorf("%q is not a colour scheme; use dark or light", s)
}
```

- [x] **Step 4: Run — PASS.** **Step 5: Commit** `#70 M1: store: a Scheme enum for the terminal background`
- [x] **Step 6: Mutations** (after the commit): drop `strings.ToLower` → the `DARK`/` Light` rows redden; drop `TrimSpace` → ` Light\n` reddens. Restore each.

### Task 2: `schemeState`, `schemeHolder`, the two flag parsers

**Files:** Create `cmd/define/scheme.go`; test `cmd/define/scheme_test.go`.

- [x] **Step 1: Write the failing tests** — **TestSchemeStateSequences** (event sequences, ARCH-ORDER), **TestSchemeHolder**, **TestSchemeHolderConcurrentReaders**, **TestParseSchemeArg**, **TestParseTintFlag**:

```go
package main

import (
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

func TestSchemeStateSequences(t *testing.T) {
	light, dark := store.SchemeLight, store.SchemeDark
	type step struct {
		apply func(schemeState) schemeState
		want  store.Scheme
		src   schemeSource
	}
	for _, tc := range []struct {
		name  string
		steps []step
	}{
		{"nothing known is dark by default", nil},
		{"a reply decides while nothing is chosen", []step{
			{func(s schemeState) schemeState { return s.withDetected(light) }, light, sourceDetected}}},
		{"a later reply overwrites an earlier one", []step{
			{func(s schemeState) schemeState { return s.withDetected(light) }, light, sourceDetected},
			{func(s schemeState) schemeState { return s.withDetected(dark) }, dark, sourceDetected}}},
		{"a choice outranks a reply that arrives after it", []step{
			{func(s schemeState) schemeState { return s.withChoice(dark, sourceFlag) }, dark, sourceFlag},
			{func(s schemeState) schemeState { return s.withDetected(light) }, dark, sourceFlag}}},
		{"clearing the choice reveals the reply kept underneath", []step{
			{func(s schemeState) schemeState { return s.withDetected(light) }, light, sourceDetected},
			{func(s schemeState) schemeState { return s.withChoice(dark, sourceSaved) }, dark, sourceSaved},
			{func(s schemeState) schemeState { return s.withoutChoice() }, light, sourceDetected}}},
		{"a session choice replaces a flag", []step{
			{func(s schemeState) schemeState { return s.withChoice(dark, sourceFlag) }, dark, sourceFlag},
			{func(s schemeState) schemeState { return s.withChoice(light, sourceSession) }, light, sourceSession}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var s schemeState
			if v, src := s.effective(); v != dark || src != sourceDefault {
				t.Fatalf("zero state = %s/%v, want dark/default", v, src)
			}
			for i, st := range tc.steps {
				s = st.apply(s)
				if v, src := s.effective(); v != st.want || src != st.src {
					t.Fatalf("step %d: %s/%v, want %s/%v", i, v, src, st.want, st.src)
				}
			}
		})
	}
}

func TestSchemeHolder(t *testing.T) {
	var nilHolder *schemeHolder
	if got := nilHolder.Scheme(); got != store.SchemeDark {
		t.Errorf("nil holder paints %s, want dark", got)
	}
	if nilHolder.detect(store.SchemeLight) {
		t.Error("a nil holder is read-only; detect must report no change")
	}
	h := newSchemeHolder(schemeState{})
	if !h.detect(store.SchemeLight) || h.Scheme() != store.SchemeLight {
		t.Error("a reply on an undecided holder must change what is painted")
	}
	if h.detect(store.SchemeLight) {
		t.Error("the same reply twice is not a change — it must not force a repaint")
	}
	// A reply that DIFFERS from the one heard before, under a choice: nothing
	// visible changes. (Repeating the earlier reply could not tell a choice that
	// outranks a reply from one that does not.)
	if !h.choose(store.SchemeDark, sourceSaved) {
		t.Error("choosing dark over a detected light changes what is painted")
	}
	if h.detect(store.SchemeDark) || h.Scheme() != store.SchemeDark {
		t.Error("with a choice in force a reply changes nothing visible")
	}
	if h.choose(store.SchemeDark, sourceFlag) {
		t.Error("re-choosing the shade already in force changes nothing visible")
	}
	// The last reply was dark, so forgetting a dark choice paints nothing new...
	if h.forget() || h.Scheme() != store.SchemeDark {
		t.Error("forgetting a dark choice over a dark reply changes nothing painted")
	}
	// ...while forgetting a light choice reveals that dark reply.
	h.choose(store.SchemeLight, sourceSession)
	if !h.forget() || h.Scheme() != store.SchemeDark {
		t.Error("forgetting a light choice must reveal the dark reply underneath")
	}
}

// The holder's claim is concurrent safety; this is the driver that would fail
// without it (run under -race).
func TestSchemeHolderConcurrentReaders(t *testing.T) {
	h := newSchemeHolder(schemeState{})
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_ = h.Scheme()
				}
			}
		}()
	}
	for i := range 1000 {
		v := store.SchemeDark
		if i%2 == 0 {
			v = store.SchemeLight
		}
		h.detect(v)
	}
	close(stop)
	wg.Wait()
}

func TestParseSchemeArg(t *testing.T) {
	for in, want := range map[string]schemeArg{
		"auto": {auto: true}, "AUTO": {auto: true},
		"dark": {value: store.SchemeDark}, "light": {value: store.SchemeLight},
	} {
		if got, err := parseSchemeArg(in); err != nil || got != want {
			t.Errorf("parseSchemeArg(%q) = %+v, %v", in, got, err)
		}
	}
	if _, err := parseSchemeArg("sepia"); err == nil {
		t.Error("an unknown scheme must be refused")
	}
}

func TestParseTintFlag(t *testing.T) {
	for in, want := range map[string]bool{"on": true, "off": false, "ON": true} {
		if got, err := parseTintFlag(in); err != nil || got != want {
			t.Errorf("parseTintFlag(%q) = %v, %v", in, got, err)
		}
	}
	for _, old := range []string{"dark", "light"} {
		_, err := parseTintFlag(old)
		if err == nil || !strings.Contains(err.Error(), "-scheme "+old) {
			t.Errorf("-language-tint %s must be refused naming -scheme %s, got %v", old, old, err)
		}
	}
	if _, err := parseTintFlag("bogus"); err == nil {
		t.Error("an unknown value must be refused")
	}
}
```

- [x] **Step 2: Run — expect FAIL** (undefined): `go test ./cmd/define -run 'TestSchemeState|TestSchemeHolder|TestParseSchemeArg|TestParseTintFlag' -count=1`
- [x] **Step 3: Implement `cmd/define/scheme.go`**

```go
package main

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/xianxu/tools/cmd/define/store"
)

// schemeSource says where the scheme in effect came from, so /scheme can say
// it truthfully. sourceDefault means nothing was chosen and nothing detected.
type schemeSource int

const (
	sourceDefault schemeSource = iota
	sourceDetected
	sourceFlag
	sourceSaved
	sourceSession
)

// schemeState is the whole of what the process knows about its scheme (#70).
//
// IMMUTABLE: every transition returns a new value, and schemeHolder swaps it in.
// Two independent facts, so their product is the legal state space: an explicit
// choice (chosenBy is flag, saved or session; sourceDefault means none), and
// what the terminal last reported (heard). The choice outranks the report; the
// report is kept underneath, so clearing the choice reveals it.
type schemeState struct {
	choice   store.Scheme
	chosenBy schemeSource
	detected store.Scheme
	heard    bool
}

func (s schemeState) effective() (store.Scheme, schemeSource) {
	if s.chosenBy != sourceDefault {
		return s.choice, s.chosenBy
	}
	if s.heard {
		return s.detected, sourceDetected
	}
	return store.SchemeDark, sourceDefault
}

// withChoice records an explicit choice. by is sourceFlag, sourceSaved or
// sourceSession; it replaces any earlier choice whatever its source.
func (s schemeState) withChoice(v store.Scheme, by schemeSource) schemeState {
	s.choice, s.chosenBy = v, by
	return s
}

func (s schemeState) withoutChoice() schemeState {
	s.choice, s.chosenBy = "", sourceDefault
	return s
}

func (s schemeState) withDetected(v store.Scheme) schemeState {
	s.detected, s.heard = v, true
	return s
}

// schemeHolder is the ONE per-process home of schemeState, shared by the editor,
// a sitting it starts, and every screen and writer that paints a tint.
//
// An atomic pointer to an immutable value rather than a mutex: five goroutines
// read it while painting under the screen lock (the loop, the throttle timer,
// the activity ticker, readInput's pointer routing, clipboard completion), and a
// load takes no lock, so there is no ordering to get wrong against l.mu or the
// pointer router's. There is exactly ONE writer at a time — the loop goroutine in
// force (a /play sitting runs on the editor loop's own goroutine) — so a
// load-transition-store needs no compare-and-swap.
//
// A nil holder paints dark and cannot change: that is a test's bare deps, and
// run() always builds one.
type schemeHolder struct{ p atomic.Pointer[schemeState] }

func newSchemeHolder(s schemeState) *schemeHolder {
	h := &schemeHolder{}
	h.p.Store(&s)
	return h
}

func (h *schemeHolder) Load() schemeState {
	if h == nil {
		return schemeState{}
	}
	if s := h.p.Load(); s != nil {
		return *s
	}
	return schemeState{}
}

// The holder's TRANSITIONS are its only writers (ARCH-ORDER structural
// enforcement): choose, forget and detect, each applying one pure schemeState
// transition and reporting whether the painted shade changed. set is their
// shared step; nothing outside this file calls it. Callers run on the loop in
// force, and a nil holder changes nothing.
func (h *schemeHolder) set(next schemeState) bool {
	a, _ := h.Load().effective()
	h.p.Store(&next)
	b, _ := next.effective()
	return a != b
}

func (h *schemeHolder) choose(v store.Scheme, by schemeSource) bool {
	return h != nil && h.set(h.Load().withChoice(v, by))
}

func (h *schemeHolder) forget() bool {
	return h != nil && h.set(h.Load().withoutChoice())
}

// Scheme is the value to paint with now.
func (h *schemeHolder) Scheme() store.Scheme {
	v, _ := h.Load().effective()
	return v
}

// detect applies a terminal report and says whether what is painted changed —
// the only case worth a repaint.
func (h *schemeHolder) detect(v store.Scheme) bool {
	return h != nil && h.set(h.Load().withDetected(v))
}

// schemeArg is what -scheme and /scheme accept: a scheme, or auto (no choice).
// ONE parser for both (ARCH-DRY), so the flag and the command cannot disagree.
type schemeArg struct {
	auto  bool
	value store.Scheme
}

func parseSchemeArg(s string) (schemeArg, error) {
	if strings.EqualFold(strings.TrimSpace(s), "auto") {
		return schemeArg{auto: true}, nil
	}
	v, err := store.ParseScheme(s)
	if err != nil {
		return schemeArg{}, fmt.Errorf("%q is not a colour scheme; use light, dark or auto", s)
	}
	return schemeArg{value: v}, nil
}

// parseTintFlag reads -language-tint, which #70 narrowed to on|off. Its old
// values name the shade, which -scheme owns now, so they are refused by name
// rather than guessed at (operator decision: narrowed, not aliased).
func parseTintFlag(s string) (bool, error) {
	switch v := strings.ToLower(strings.TrimSpace(s)); v {
	case "on":
		return true, nil
	case "off":
		return false, nil
	case "dark", "light":
		return false, fmt.Errorf("-language-tint is on or off now; the shade follows the colour scheme: use -scheme %s", v)
	}
	return false, fmt.Errorf("invalid -language-tint %q: use on or off", s)
}
```

- [x] **Step 4: Run — PASS**, including `go test ./cmd/define -race -run TestSchemeHolderConcurrentReaders -count=1`; then `go vet ./cmd/define`.
- [x] **Step 5: Commit** `#70 M1: scheme state as an immutable value behind one atomic holder`
- [x] **Step 6: Mutations**, one at a time: (a) `effective()` checks `heard` before `chosenBy` → the "choice outranks" and holder cases redden; (b) `withoutChoice` leaves `chosenBy` → "clearing reveals" reddens; (c) `detect` returns `true` unconditionally → the "same reply twice" case reddens; (d) change the holder to a plain `*schemeState` field (no atomic) → `-race` on **TestSchemeHolderConcurrentReaders** reports a race; (f) `choose` returns `true` unconditionally → the holder test's "changes nothing visible" case reddens (add that assertion on `choose`'s result); (e) `parseTintFlag` drops the `dark, light` case → the named-refusal assertion reddens. Restore each.

### Task 3: Delete the tint paths production never reached

Do this BEFORE the role change (Task 4), so nothing is migrated only to be deleted.

**Files:** `cmd/define/language_style.go`, `cmd/define/dictionary_language.go`, `cmd/define/render.go`, `cmd/define/output_screen.go`, `cmd/define/definitions.go`, `cmd/define/practice_language.go`, `cmd/define/play_loop.go`; create `cmd/define/render_helpers_test.go`; tests `language_style_test.go`, `dictionary_language_test.go`, `bilingual_conformance_test.go` (tagged). NOT `dict_test.go` or `dictionary_source_test.go`: they go through `renderDefinitions` → `renderDefinitionOutput`, where the tint returns as section row paint — the LIVE path.

- [x] **Step 1: Evidence, by SIMULATING the deletion** (lessons "Deleting a test needs the same evidence as writing one"; a `panic` would trip the zero-tint calls production DOES make and abort the test binary). On the committed baseline, make `styleLanguageText` return `t.text` as its first statement. Run through `go test` ONLY — `go vet` would report the now-unreachable body, which is the simulation, not a finding: `go test ./cmd/define/... -count=1` and `go test -tags conformance ./cmd/define -run TestBilingualNativeLanguageOwnership -count=1` (say whether it ran or skipped). Every failure must be a test that calls `Render` or `styleLanguageText` directly with a NON-ZERO tint; a reviewer measured exactly five in the default suite, on a run WITHOUT the pty-backed tests (`TestLanguageTintInvocation`, `TestLanguagePromptStartup` — expected to pass, since they render with a zero tint; a sixth failure in a full environment is investigated, not accepted) — `TestDictionaryCapturedMixedOwnership`, `TestDictionarySourceProvenanceCorpus`, `TestDictionaryMonolingualOriginAndDisabledTint`, `TestLanguageTintStyle`, `TestLanguageTintMixedAndSelection`. Record the list in the issue Log. Any OTHER failure means a live path — STOP and re-plan. Restore with `git checkout HEAD -- cmd/define/language_style.go`.
- [x] **Step 2: Remove the tint from `Render`** (`render.go`). Each `opt.dictionaryText(e, <original>, <rendered>, …)` returned `<rendered>` whenever the tint is zero (`projectLanguageText` returns `text: rendered`), so replace each call with its `<rendered>` argument — at ~203 `opt.prose(wrapText(body, opt.Width, lead), "")`, at ~221 `opt.prose(wrapText(prettyPronunciations(ex.Text, p), opt.Width, len(indent)+2), p.ex)`. At ~147-154 the call sat in an `if … { text = … }` block that becomes a no-op: delete the block and `headAt`, which loses its only reader. At ~243-245 delete the statement and the `if` left empty around it.
- [x] **Step 3: Delete** `RenderOpts.dictionaryText`, `dictionaryFragment`, `styleLanguageText`, and each helper left with no reader — check `lineInkBounds` and `validateLanguageText` with `grep -rnw <name> cmd/define --include='*.go'`. KEEP `sourceBackground` (read by `paintLanguageRow` and `answerwrap.go:129`). Delete `styledBoardPrompt` (no callers at all).
- [x] **Step 4: Record the residue, don't widen the task.** Also name, in the Log, the tests that will then check ONLY the residue — the `entry.source.spans` check in `TestBilingualNativeLanguageOwnership`, `TestDictionaryParserSourceOffsets`, the `projectDictionaryText` half kept in Step 6, and whatever remains of the three `TestDictionary*` tests — so they go with the chain when it goes. After Step 3 a provenance chain loses its last reader: `RenderOpts.Language` (read only at the deleted `dictionary_language.go:141`), `definitions.go:88-89`, `Entry.source` (`parse.go:46`, `definitions.go:94-95`), `sourceAt`/`sourceKnown` (`parse.go:187-202,760,813`), `definitionSection.source` (`definitions.go:150,152`), `bilingualDocument.native` and `projectDictionaryText` (`bilingual_layout.go:105`). Go does not report unused struct fields, so nothing forces this. Verify each member's readers by grep, list the chain in the issue Log, and raise the follow-up (delete #66's source-provenance data, or give it a consumer) with the operator at the M1 boundary. It is separable from #70 (ARCH-PURPOSE: it is not the purpose), and deleting it touches the parser.
- [x] **Step 5: Move the test-only renderers into `render_helpers_test.go`** with their names and signatures unchanged — `renderOutputText`, `renderDefinitions`, `renderPracticePresentation`, `practiceChrome`, `boardFooter` (their only callers are tests; still declared, so no removed-name sweep). Delete them from production.
- [x] **Step 6: Tests.** `language_style_test.go`: delete the direct `styleLanguageText` unit tests (`TestLanguageTintStyle`, `TestLanguageTintMixedAndSelection`, and whatever of `TestLanguageTextValidation` exercised a deleted helper); `TestLanguageTintProfile` stays until Task 4. Step 1's list is authoritative for the rest: the per-fragment tint assertions that went through `Render` with a non-zero tint (the three `TestDictionary*` tests it names, and the tagged `TestBilingualNativeLanguageOwnership` at `bilingual_conformance_test.go:129`) describe behaviour production never had — it tints whole SECTIONS (`definitions.go:123-127`), pinned by `TestDefinitionOutputUniformSections`. Remove the tint half of each; keep what they assert about text and regions. `TestDictionaryProjectionExactOccurrenceAndFallback` calls `dictionaryFragment` directly: delete that half, keep its `projectDictionaryText` half. A test left with nothing to assert is deleted and gets a `| deleted |` row (Step 7). Name each in the Log.
- [x] **Step 7: Add one `| deleted |` row per removed citable symbol** to this plan's Pure-entities table, each alone in its first cell with a repo-relative path — e.g. `` | `styleLanguageText` | `cmd/define/language_style.go` | deleted | `` — for `styleLanguageText`, `dictionaryFragment`, `dictionaryText` (`cmd/define/dictionary_language.go`), `styledBoardPrompt` (`cmd/define/practice_language.go`), `lineInkBounds` / `validateLanguageText` if removed, AND every deleted test function this plan names (`TestLanguageTintStyle`, `TestLanguageTintMixedAndSelection`, and `TestLanguageTextValidation` if it goes — path `cmd/define/language_style_test.go`). A removed `Test*` name the plan still mentions fails both `TestPlanCitesTestsThatExist` and `TestARemovedDeclarationIsSweptOrRetired`; the row is what exempts it.
- [x] **Step 8: Verify.** `go build ./cmd/define/...` (only a build catches production still calling a helper that moved into a `_test.go`), `go test ./cmd/define/... -count=1`, `go vet ./...`, `go vet -tags conformance ./cmd/define`, and `go test -tags conformance ./cmd/define -run TestBilingualNativeLanguageOwnership -count=1` (ran or skipped — say which) — PASS. Sweep: `grep -rnw 'styleLanguageText\|dictionaryFragment\|dictionaryText\|lineInkBounds\|validateLanguageText\|styledBoardPrompt\|headAt' cmd/define atlas README.md` — expected output: EMPTY.
- [x] **Step 9: Commit** `#70 M1: delete the tint paths production never reached`
- [x] **Step 10: After the commit, run the whole package again** — `TestARemovedDeclarationIsSweptOrRetired` and `TestPlanTableStatusMatchesTheChangeWindow` read `base..HEAD`, so before the commit they cannot see the deletion (lessons #53: "After a commit, run the whole package"). A failure here is fixed in a follow-up commit, not by amending history that has been read.

### Task 4: The tint becomes a role; the flags choose the shade

ONE commit: the role, the holder in `run()`, the flags, and every test migration. The type change breaks the build until all of it lands, and the flags cannot move separately (`tintProfile` produces the escape string the role replaces).

**Production files** (every site, from the reviews):
- `output_layout.go` — `rowPaint{tinted bool; exclusions []cellRange}`; `validRowPaint` drops the background clause (~24); `layoutOutput` copies `tinted` (~80); `serializeOutput(o renderedOutput, width int, sc store.Scheme)`. Rewrite the doc comment (~10): *"WHETHER a row is tinted is decided when it is produced, so later policy changes (a /lang switch) do not recolour history; WHICH shade a tint is resolves at paint from the scheme in effect (#70), so a /scheme switch does."*
- `language_row.go` — `paintLanguageRow(text string, paint rowPaint, width int, sc store.Scheme)`: `if !paint.tinted { return text }`; write `schemeTint(sc)` where it wrote `paint.background`.
- `language_style.go` — `tintPolicy{lang store.Lang; on bool; scheme *schemeHolder}`; add `func schemeTint(s store.Scheme) string` (light → `languageLight`, else `languageDark`); replace the `options.tintFor` method with `func tintFor(d deps, opt options) tintPolicy { return tintPolicy{lang: d.lang, on: opt.color && opt.tintOn, scheme: d.scheme} }`; delete `tintProfile` (add its `| deleted |` row, and one for `TestLanguageTintProfile` at `cmd/define/language_style_test.go`, as Task 3 Step 7).
- `output_screen.go` — `writeOutput(w io.Writer, o renderedOutput, width int, sc store.Scheme)` (the `outputWriter` branch ignores `sc`: a screen reads its own holder); `paintOutputChunk(text string, p rowPaint, width int, sc store.Scheme)` (`if !p.tinted`); `sliceRowPaint` copies `tinted`; `paintedTranscript` reads `sc := s.scheme.Scheme()` ONCE before its loop.
- `screen.go` — `screen` gains `scheme *schemeHolder`; `selectionLayout` gains `scheme store.Scheme`, set ONCE at the top of `layoutSelectionFrame` (`layout := selectionLayout{scheme: s.scheme.Scheme()}`, replacing `layout := selectionLayout{}`), so a transition mid-frame cannot mix two shades; `selectionLayout.paint` passes `layout.scheme` to both painters. Add:

```go
// attachScheme gives the screen the process's scheme holder (#70). Call it
// BEFORE the screen is shared with another goroutine — the pointer router, the
// resize watcher, the throttle timer — because this field is not atomic; the
// holder it points at is.
func (l *liveScreen) attachScheme(h *schemeHolder) { l.s.scheme = h }
```
- `replraw.go` `newConsole` — `live.attachScheme(d.scheme)` right after `live := newScreen(...)`, before `newPointerRouter`. `play_cmd.go` `sittingInPlace` — `sitting.attachScheme(d.scheme)` right after `sitting := newPinnedScreen(...)`, before `pointer.activate(sitting)`. (These are the only production screen constructions: `replraw.go:33` and `play_loop.go:115` via `newConsole`, and `play_cmd.go:112`.)
- `selection_frame.go:190` — compare `r.paint.tinted != s.paint.tinted`.
- `answerwrap.go` — ~228 `p.background = ""` → `p.on = false`; `finalize`: `tinted := !owner.mixed && owner.lang == target && w.policy.on`, `rows: []rowPaint{{tinted: tinted}}`, `writeOutput(w.out, …, w.width, w.policy.scheme.Scheme())`. (Lines ~129/140 are a different field — leave them.)
- `definitions.go:125` — `paints[i].tinted = opt.Tint.on`.
- `practice_output.go` — ~96 `paint.tinted = policy.on`; ~116 `sectionBase.rows[row].tinted = region.Tinted`; ~142 `Tinted: paintAt(o.rows, i).tinted`; ~150/157 `tintFor(d, opt)`.
- `play/presentation.go:26` — `Background string` → `Tinted bool` (doc: "a role; the shade is main's business").
- Every `opt.tintFor(d.lang)` → `tintFor(d, opt)`: `ask.go:164`, `cloze.go:236`, `main.go:1076`, `play_loop.go:1083`, `practice_language.go:49,150` (and the moved helpers in `render_helpers_test.go`).
- `writeOutput` callers pass the scheme: `main.go:1150`, `practice_language.go:55,150` → `d.scheme.Scheme()`.
- `main.go` — `deps` gains `scheme *schemeHolder` (doc: "a pointer, on practiceHelp's precedent, so a nested sitting and the editor share it — NOT bilingual's, whose setter replaces the pointer in a by-value copy of deps"). `options.tintBackground string` → `tintOn bool` (doc: "-language-tint; false under TERM=dumb"). Flags, replacing the `-language-tint` definition and the `tintProfile` block (~524, ~605-611):

```go
schemeFlag := fs.String("scheme", "auto", "terminal background: auto, dark, or light")
languageTint := fs.String("language-tint", "on", "tint the target language's rows: on or off")
```
```go
tintOn, err := parseTintFlag(*languageTint)
if err != nil {
	fmt.Fprintf(stderr, "define: %v\n", err)
	return 2
}
schemeChoice, err := parseSchemeArg(*schemeFlag)
if err != nil {
	fmt.Fprintf(stderr, "define: -scheme: %v\n", err)
	return 2
}
if os.Getenv("TERM") == "dumb" {
	tintOn = false // a dumb terminal prints escapes as text (and M3's query with them)
}
```
  `opt` literal: `tintOn: tintOn`. The holder, AFTER the `--llm-check` return and BEFORE `d = d.withStore` (~882) — every consumer is downstream (the one-shot dispatch ~942, repl, `--play`, `defineOnce`, ask), and `--version`/`--llm-check` never read it; an injected holder wins, per the repo convention (~867) — which also means a test that injects a holder has `-scheme` ignored; say so in the comment:

```go
if d.scheme == nil {
	var st schemeState
	if !schemeChoice.auto {
		st = st.withChoice(schemeChoice.value, sourceFlag)
	}
	d.scheme = newSchemeHolder(st)
}
```
  `-h` prose: one sentence — *"-scheme light or dark picks the shade of the language tint to suit the terminal's background; -language-tint off turns the tint off."* (M2 and M3 extend it; no mention of detection yet.)
- `cmd/define/README.md:344-345` — `-scheme light` and `-language-tint off` replace `-language-tint=light|off`.

- [x] **Step 1: Write the failing tests** (they will not compile until the type exists — that is the RED):

```go
// language_row_test.go — the role is frozen at production; the shade resolves at paint.
func TestATintedRowTakesTheShadeOfTheSchemeItIsPaintedIn(t *testing.T) {
	p := rowPaint{tinted: true}
	if got := paintLanguageRow("hola", p, 6, store.SchemeDark); !strings.Contains(got, languageDark) || strings.Contains(got, languageLight) {
		t.Errorf("dark: %q", got)
	}
	if got := paintLanguageRow("hola", p, 6, store.SchemeLight); !strings.Contains(got, languageLight) || strings.Contains(got, languageDark) {
		t.Errorf("light: %q", got)
	}
	if got := paintLanguageRow("hola", rowPaint{}, 6, store.SchemeLight); got != "hola" {
		t.Errorf("an untinted row is untouched: %q", got)
	}
}

// output_screen_test.go — history repaints in the scheme in force at paint time.
func TestAScreenRepaintsHistoryInTheCurrentScheme(t *testing.T) {
	var tty bytes.Buffer
	h := newSchemeHolder(schemeState{})
	l := newLiveScreen(&tty, 10, 20)
	l.interval = -1
	l.attachScheme(h)
	if err := l.WriteOutput(renderedOutput{text: "hola\n", rows: []rowPaint{{tinted: true}}}); err != nil {
		t.Fatal(err)
	}
	h.choose(store.SchemeLight, sourceSession)
	tty.Reset()
	l.Draw("› ", nil)
	if !strings.Contains(tty.String(), languageLight) || strings.Contains(tty.String(), languageDark) {
		t.Errorf("the repaint kept the old shade: %q", tty.String())
	}
	if tr := l.PaintedTranscript(); !strings.Contains(tr, languageLight) {
		t.Errorf("the exit transcript kept the old shade: %q", tr)
	}
}
```
  And rewrite `TestLanguageTintInvocation` / `TestLanguageTintInvalidFlagBeforeStore` (`language_style_paths_test.go`) as the flag table (args → exit, shade): none → 0, `languageDark`; `-scheme dark` → dark; `-scheme light` → `languageLight`; `-scheme auto` → dark; `-language-tint off` → no tint escape; `-language-tint light` → exit 2, stderr contains `-scheme light`, NO store directory created; `-language-tint bogus` → exit 2, `invalid -language-tint`; `-scheme sepia` → exit 2, `not a colour scheme`; `TERM=dumb` → no tint; and KEEP today's `plain` (`-no-color`) and `redirect` rows (`language_style_paths_test.go:21,23`). They cannot pin `tintFor`'s colour gate, though: the lookup path has its own (`definitions.go:123`, `opt.Color && …`). The gate matters where there is no second check — practice output (`practice_output.go:96`) and the answer writer use `policy.on` alone, so `define --play -no-color` would print tint escapes without it. So add **TestTintForGatesOnColour**: `tintFor(deps{lang: "es"}, options{color: false, tintOn: true}).on` is false, and with `color: true` it is true — the direct pin `TestLanguageTintProfile`'s second half used to be.
- [x] **Step 2: Make every production change listed above.** `go build ./cmd/define/...` until clean.
- [x] **Step 3: Migrate every test the compiler rejects — default AND tagged** (`go vet -tags conformance ./cmd/define` finds the tagged ones: `bilingual_layout_conformance_test.go:55-61` and the `assertDefinitionSectionCells` helper it calls in `definitions_output_test.go:22`). Rules, so each test asserts what it asserted before:
  1. `rowPaint{background: languageDark|languageLight}` → `rowPaint{tinted: true}`; the paint call that follows receives the matching `store.SchemeDark` / `store.SchemeLight`; `rowPaint{background: ""}` → `rowPaint{}`.
  2. `tintPolicy{lang, languageX}` → `tintPolicy{lang: lang, on: true, scheme: holderFor(store.SchemeX)}`; `tintPolicy{lang, ""}` → `tintPolicy{lang: lang}`.
  3. `options{tintBackground: languageX}` → `options{tintOn: true}` plus `d.scheme = holderFor(store.SchemeX)` where the shade matters; `opt.tintFor(x)` → `tintFor(d, opt)` with `d.lang = x`.
  4. `.background` compared on a paint → `.tinted`; where the test really asserts the SHADE, assert it on the painted bytes.
  5. `play.PresentationRegion{Background: languageX}` → `{Tinted: true}`.
  6. Loops over `{dark: languageDark, light: languageLight}` profiles → loops over `store.Scheme` values asserting `schemeTint(sc)`.
  7. Calls to `paintLanguageRow`/`serializeOutput`/`writeOutput`/`paintOutputChunk` and the moved `renderOutputText` gain the scheme argument (`renderOutputText(o, sc)`; `renderDefinitions` passes `opt.Tint.scheme.Scheme()`, `renderPracticePresentation` passes `policy.scheme.Scheme()`).
  8. `TestLanguageTintProfile` (`language_style_test.go`) tested the deleted `tintProfile`: delete it; the flag table in Step 1 replaces it.
  9. pty: `TestPTYLanguageTint` profiles become `{dark: "-scheme=dark"}`, `{light: "-scheme=light"}`, `{off: "-language-tint=off"}`; same for `TestPTYNativeRendirSectionLayout:40-43`.
  Add once, in `render_helpers_test.go`:

```go
// holderFor is a holder already set to one scheme, for tests that paint a shade.
func holderFor(s store.Scheme) *schemeHolder {
	return newSchemeHolder(schemeState{}.withChoice(s, sourceFlag))
}
```
- [x] **Step 4: Verify.** `go build ./cmd/define/...`, `go test ./cmd/define/... -count=1`, `go vet ./...`, `go vet -tags conformance ./cmd/define` — PASS. Conformance: `go test -tags conformance ./cmd/define -run 'TestPTYLanguageTint|TestPTYNativeRendir|TestBilingualNativeRendirLayout' -count=1` — PASS; say which ran and which skipped (they need the Oxford ES dictionary, `bilingualNativeProbe`). Grep: `grep -rn 'tintBackground\|tintProfile\|paint\.background\|p\.background\b' cmd/define --include='*.go'` — expected: EMPTY (`answerwrap.go`'s `w.background` is a different field and does not match; a pty test's `profile.background` field should be renamed by rule 9 anyway).
- [x] **Step 5: Commit** `#70 M1: a row records whether it is tinted; -scheme picks the shade at paint`, then run the whole package again (the window guards read `base..HEAD`).
- [x] **Step 6: Mutations**, one at a time: `paintLanguageRow` always writes `languageDark` → both new tests redden; `paintedTranscript` reads nothing (uses `store.SchemeDark`) → the transcript assertion reddens; skip `withChoice` for the flag → the `-scheme light` row reddens; `tintFor` ignores `opt.tintOn` → the `-language-tint off` row reddens; `tintFor` ignores `opt.color` → **TestTintForGatesOnColour** reddens; drop the `TERM=dumb` line → its row reddens. Restore each.

### Task 5: Pin the holder's wiring in the loop shells

**TestAScreenRepaintsHistoryInTheCurrentScheme** attaches the holder itself, so it cannot see a missing `attachScheme` in production (lessons: "Pin a loop shell's wiring with a test that drives that loop shell"; `TestNewConsoleEnablesEveryMode`'s comment records the same failure for modes).

**Files:** test `cmd/define/rawterm_test.go`, `cmd/define/selection_nested_test.go` (or a new `scheme_wiring_test.go`).

- [x] **Step 1: Write** **TestNewConsoleAttachesTheSchemeHolder**: build a console through `newConsole` exactly as `TestNewConsoleEnablesEveryMode` does, with `d := testDeps(t); d.scheme = holderFor(store.SchemeLight)` and a capturing `newScreen` closure that keeps the `*liveScreen`; write a tinted row through the console's `stdout` (`writeOutput(con.stdout, renderedOutput{text: "hola\n", rows: []rowPaint{{tinted: true}}}, 20, store.SchemeDark)` — the scheme argument is ignored by a screen) and assert the captured screen's `PaintedTranscript()` carries `languageLight` (read through the screen's lock, not a plain `bytes.Buffer` a throttled flush may be writing — Task 6 runs `-race`).
- [x] **Step 2: Write** **TestASittingPaintsInTheEditorsScheme**: on the `selection_nested_test.go:56` pattern (a parent `*liveScreen`, the real `sittingInPlace` on a goroutine, `playRig` for a due word), with `d.scheme = holderFor(store.SchemeLight)`, wait for the nested screen (`router.active != parent`) and assert a tinted row painted there carries `languageLight`. If the sitting's first question has no tinted row, write one through the nested screen's `WriteOutput`.
- [x] **Step 3: Run — PASS. Commit** `#70 M1: pin the scheme holder where production attaches it`, then run the whole package again.
- [x] **Step 4: Mutations:** delete `live.attachScheme(d.scheme)` in `newConsole` → Step 1's test reddens; delete `sitting.attachScheme(d.scheme)` → Step 2's reddens. Restore each.

### Task 6: M1 boundary

- [x] `go test ./... -count=1`, `go test ./cmd/define -race -count=1`, `go vet ./...`, `go vet -tags conformance ./cmd/define`, `go test -tags conformance ./cmd/define -count=1` (report which tests skipped), `GOOS=linux go build ./...` — all green; quote the counts.
- [x] `atlas/define.md`: rewrite the `-language-tint` line (~2270) and the `RenderOpts.Tint` row (~578) for the role/shade split; the `RenderOpts.Language` row (~577, "explicit mixed-source ranges take precedence") is false after Task 3 but `TestAtlasDescribesEveryRenderOpt` keeps the row mandatory — reword it "unread since #70 (residue; see the issue Log)"; `renderDefinitions` (~2256) is now a TEST helper — say so or drop it; add a short "The shade is a paint-time decision (#70)" paragraph under **The screen**: the holder, `attachScheme` before sharing, the once-per-frame read.
- [x] Raise Task 3 Step 4's residue chain with the operator; on their yes, file it with `sdlc issue new` so it outlives the session (the Log names the tests that go with it).
- [x] `sdlc milestone-close --issue 70 --milestone M1` — read the verdict before ticking `M1`; fix Critical/Important first.

---

## Chunk 2: M2 — `/scheme` and the saved choice

Docs in this chunk describe what M2 ships — flag, saved, dark — and do not mention detection; M3 adds it.

### Task 7: `ReadScheme` / `WriteScheme` / `ClearScheme`

**Files:** Modify `cmd/define/store/scheme.go`; test `cmd/define/store/scheme_test.go`.

- [x] **Step 1: Failing tests** against `t.TempDir()`:
  - missing file → `("", false, nil)`;
  - `WriteScheme(dir, SchemeLight)` into a `dir` that does not exist yet → `ReadScheme` gives `(SchemeLight, true, nil)`; the file is exactly `light\n`; `dir` holds exactly ONE entry afterwards (no `.tmp-*` left, the `bilingual_test.go:35-41` precedent);
  - `"  DARK \n"` → `SchemeDark`;
  - `"sepia"` and `""` → `("", false, err)`, and the error names the file's path;
  - THE CAP, with a valid word so only the cap can refuse it: `"light"` + 59 spaces (64 bytes) → `SchemeLight`; `"light"` + 60 spaces (65 bytes) → error (the `store/bilingual_test.go:45` pattern);
  - `ClearScheme` removes the file AND the now-empty `dir`; a second `ClearScheme` → nil; with a foreign file also in `dir`, only `scheme` goes and `dir` stays;
  - **a SYMLINKED `dir`** (a dotfile manager's `~/.config/define` → elsewhere, the target holding a foreign file): after `ClearScheme` the LINK still exists and the foreign file survives. `os.Remove` unlinks a symlink even when its target is full, so a bare `os.Remove(dir)` fails this.
- [x] **Step 2: Run — FAIL.** `go test ./cmd/define/store -run Scheme -count=1`
- [x] **Step 3: Implement** (imports: `errors`, `fmt`, `io`, `io/fs`, `os`, `path/filepath`, `strings`):

```go
const schemeFileName = "scheme"
const maxSchemeSettingBytes = 64

// ReadScheme reads the saved scheme in dir (define's USER config directory, not a
// deck). found is false with a nil error when nothing is saved. Anything
// unreadable, oversized or outside the enum is an ERROR naming the file, so the
// caller can warn once and treat it as unset (ARCH-SECURE: a hand-edited or
// truncated file is untrusted input, parsed into the closed enum here).
func ReadScheme(dir string) (Scheme, bool, error) {
	path := filepath.Join(dir, schemeFileName)
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxSchemeSettingBytes+1))
	if err != nil {
		return "", false, fmt.Errorf("%s: %w", path, err)
	}
	if len(b) > maxSchemeSettingBytes {
		return "", false, fmt.Errorf("%s is larger than a scheme name", path)
	}
	s, err := ParseScheme(string(b))
	if err != nil {
		return "", false, fmt.Errorf("%s: %w", path, err)
	}
	return s, true, nil
}

// WriteScheme saves s atomically, creating dir if needed.
func WriteScheme(dir string, s Scheme) error {
	return writeBytesAtomic(filepath.Join(dir, schemeFileName), []byte(string(s)+"\n"))
}

// ClearScheme forgets the saved scheme, then the directory if that left it
// empty (ARCH-FUNERAL: the residue is at most this file and its directory).
//
// It removes only what is OURS to remove. The file is the saved choice itself,
// link or not, so forgetting the choice removes it. The directory goes only if
// it is a REAL directory that is now empty: os.Remove unlinks a symlink even
// when its target is full, which would break a dotfile manager's link (stow,
// chezmoi) — so Lstat first, and a non-empty real directory refuses on its own.
func ClearScheme(dir string) error {
	if err := os.Remove(filepath.Join(dir, schemeFileName)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if fi, err := os.Lstat(dir); err == nil && fi.IsDir() {
		_ = os.Remove(dir) // ENOTEMPTY when something else lives there: kept
	}
	return nil
}
```
- [x] **Step 4: PASS. Step 5: Commit** `#70 M2: store: the saved scheme, one word in the user's config directory`
- [x] **Step 6: Mutations:** drop the `len(b) >` check → the 65-byte case reddens; drop `os.Remove(dir)` → the empty-dir case reddens; drop the `Lstat`/`IsDir` guard → the symlinked-dir case reddens; drop the path wrap → the names-the-file case reddens. Restore each. The class (cleanup removes only owned residue) has no other instance in #70: the only other removals are `t.TempDir()`s the tests own. `WriteScheme`'s atomic rename replaces a symlinked `scheme` FILE with a regular one — the store's behaviour for every setting (`bilingual.txt` too), noted in the Log rather than changed here.

### Task 8: The config-directory seam and the startup read

**Files:** Modify `cmd/define/scheme.go`, `cmd/define/main.go`; tests `cmd/define/scheme_test.go`, `cmd/define/language_style_paths_test.go`.

- [x] **Step 1: Failing tests.** **TestConfigDirFrom** (pure; `configDirFrom` needs `path/filepath` in `scheme.go`):

```go
func TestConfigDirFrom(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	for _, tc := range []struct {
		name string
		env  map[string]string
		want string
		ok   bool
	}{
		{"xdg wins", map[string]string{"XDG_CONFIG_HOME": "/x", "HOME": "/h"}, "/x/define", true},
		{"home fallback", map[string]string{"HOME": "/h"}, "/h/.config/define", true},
		{"relative xdg is ignored", map[string]string{"XDG_CONFIG_HOME": "rel", "HOME": "/h"}, "/h/.config/define", true},
		{"nothing usable", map[string]string{"HOME": "rel"}, "", false},
		{"empty", nil, "", false},
	} {
		if got, ok := configDirFrom(env(tc.env)); got != tc.want || ok != tc.ok {
			t.Errorf("%s: got %q,%v want %q,%v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}
```
  **TestRealDepsConfigDirReadsXDG**: `t.Setenv("XDG_CONFIG_HOME", dir)` → `realDeps().configDir()` returns `dir/define`, true (the production wiring, in process — lessons "Adding a field is not wiring it").
  Through `run()` (with `d.configDir` → a temp dir): a saved `light`, no flag → `languageLight` in a lookup; `-scheme dark` beats a saved `light`; a garbled file → exactly one `define: ignoring saved scheme:` line on stderr and the dark shade; `d.configDir == nil` → dark, no stderr.
- [x] **Step 2: Run — FAIL. Step 3: Implement.**

```go
// configDirFrom resolves define's user config directory. Only ABSOLUTE bases
// count: a relative XDG_CONFIG_HOME or HOME would put the file wherever the
// process happens to stand — the deck's directory, the one place this setting
// must not live.
func configDirFrom(getenv func(string) string) (string, bool) {
	if x := getenv("XDG_CONFIG_HOME"); filepath.IsAbs(x) {
		return filepath.Join(x, "define"), true
	}
	if h := getenv("HOME"); filepath.IsAbs(h) {
		return filepath.Join(h, ".config", "define"), true
	}
	return "", false
}
```
  `deps` gains `configDir func() (string, bool)` — its OWN seam, not `getenv` (the model seam; some tests make it panic, `practice_help_paths_test.go:66`). `realDeps`: `configDir: func() (string, bool) { return configDirFrom(os.Getenv) }`. Test deps leave it nil (no in-process test reads the real `~/.config`: `realDeps()` is called only by `main()` and `dict_test.go:52`, which uses `newDict`). In `run()`, Task 4's holder block becomes:

```go
if d.scheme == nil {
	var st schemeState
	switch {
	case !schemeChoice.auto:
		st = st.withChoice(schemeChoice.value, choiceFlag)
	case d.configDir != nil:
		if dir, ok := d.configDir(); ok {
			if v, found, err := store.ReadScheme(dir); err != nil {
				fmt.Fprintf(stderr, "define: ignoring saved scheme: %v\n", err)
			} else if found {
				st = st.withChoice(v, choiceSaved)
			}
		}
	}
	d.scheme = newSchemeHolder(st)
}
```
  It stays after the `--llm-check` return, so `--version` never reads the user's config (`version_conformance_test.go:73-81` runs with the inherited environment).
- [x] **Step 4: PASS. Step 5: Commit** `#70 M2: a saved scheme is read at startup, from its own seam`
- [x] **Step 6: Mutations:** swap the flag and saved cases → "flag beats saved" reddens; drop the warning → the garbled case reddens; `realDeps` without `configDir` → **TestRealDepsConfigDirReadsXDG** reddens (it asserts `realDeps().configDir != nil` first, so the failure is a message, not a nil-call panic). Restore each.

### Task 9: `describeScheme` and `applyScheme`

**Files:** Modify `cmd/define/scheme.go` (add `errors`); test `cmd/define/scheme_test.go`.

- [x] **Step 1: Failing tests.** A stateful fake:

```go
type fakePersister struct {
	saved   store.Scheme
	failing error
}

func (f *fakePersister) save(s store.Scheme) error {
	if f.failing != nil {
		return f.failing
	}
	f.saved = s
	return nil
}

func (f *fakePersister) clear() error {
	if f.failing != nil {
		return f.failing
	}
	f.saved = ""
	return nil
}
```
  **TestApplyScheme** rows (holder before → arg, persister, session → effective/source after, fake's saved, error):
  - empty → `light`, fake, session → light/saved, saved=light;
  - empty → `light`, failing fake, session → error, holder UNCHANGED, saved unchanged;
  - empty → `light`, nil, session → light/session;
  - empty → `light`, nil, one-shot → `errNowhereToSave`, holder unchanged;
  - flag light → `dark`, fake, session → dark/saved (the Done-when's "-scheme light then /scheme dark");
  - saved light over detected dark → `auto`, fake, session → dark/detected, saved="";
  - saved light → `auto`, failing fake, session → error, holder unchanged (the same failure rule for clearing);
  - nil holder → `errNoScheme`.
  **TestDescribeScheme**, each string exact: `light (saved)`, `dark (detected)`, `light (-scheme flag)`, `light (session only; not saved)`, `dark (default: the terminal has not reported its background)` (full-screen), `dark (default: detected only in a full-screen session)` (otherwise — the spec's "interactive session" wording is revised to this in the issue Log: the piped loop is a session and never detects).
- [x] **Step 2: Run — FAIL. Step 3: Implement.**

```go
var (
	errNoScheme      = errors.New("there is no colour scheme to change here")
	errNowhereToSave = errors.New("nowhere to save it: $XDG_CONFIG_HOME and $HOME are unset or not absolute")
)

// schemePersister is the durable half of /scheme. nil means there is nowhere to
// save (no config directory).
type schemePersister interface {
	save(store.Scheme) error
	clear() error
}

type dirSchemePersister string

func (d dirSchemePersister) save(s store.Scheme) error { return store.WriteScheme(string(d), s) }
func (d dirSchemePersister) clear() error             { return store.ClearScheme(string(d)) }

func (d deps) schemePersister() schemePersister {
	if d.configDir == nil {
		return nil
	}
	if dir, ok := d.configDir(); ok {
		return dirSchemePersister(dir)
	}
	return nil
}

// applyScheme is /scheme's transition. PERSIST, THEN SWITCH — /bilingual's rule
// (bilingual_cmd.go:37), not a second one: a failed write changes nothing, so the
// message can never claim a switch that did not persist. With nowhere to save, a
// session switches for itself alone and says so; a one-shot has nothing else to
// change, so it refuses.
func applyScheme(h *schemeHolder, arg schemeArg, p schemePersister, session bool) (schemeState, error) {
	if h == nil {
		return schemeState{}, errNoScheme
	}
	switch {
	case p == nil && !session:
		return h.Load(), errNowhereToSave
	case p == nil && arg.auto:
		h.forget()
	case p == nil:
		h.choose(arg.value, choiceSession)
	case arg.auto:
		if err := p.clear(); err != nil {
			return h.Load(), err
		}
		h.forget()
	default:
		if err := p.save(arg.value); err != nil {
			return h.Load(), err
		}
		h.choose(arg.value, choiceSaved)
	}
	return h.Load(), nil
}

// describeScheme is /scheme's report, and every wording is TRUE of its state:
// "has not reported" holds whether the reply is pending, unsupported or never
// asked for; outside a full-screen session nothing asks.
func describeScheme(s schemeState, fullScreen bool) string {
	v, src := s.effective()
	switch src {
	case sourceDetected:
		return string(v) + " (detected)"
	case sourceFlag:
		return string(v) + " (-scheme flag)"
	case sourceSaved:
		return string(v) + " (saved)"
	case sourceSession:
		return string(v) + " (session only; not saved)"
	}
	if fullScreen {
		return string(v) + " (default: the terminal has not reported its background)"
	}
	return string(v) + " (default: detected only in a full-screen session)"
}
```
- [x] **Step 4: PASS. Step 5: Commit** `#70 M2: /scheme's transition persists first, and its report is always true`
- [x] **Step 6: Mutations:** `h.choose` moved before `p.save` → the failing-fake rows redden; the `p == nil && !session` case removed → the one-shot row reddens. Restore each.

### Task 10: The `/scheme` command in all three contexts, with its docs

**Files:** Create `cmd/define/scheme_cmd.go` (imports `fmt`, `strings`); modify `cmd/define/command.go` (registry, `commandCtx`, `newCommandCtx`), `cmd/define/replraw.go` (~691), `cmd/define/repl.go` (~419), `cmd/define/README.md` and `atlas/define.md` (the command-list and command-usage spans), `cmd/define/main.go` (`-h` prose); tests `cmd/define/scheme_cmd_test.go`.

- [x] **Step 1: Failing tests — every wiring gets a test driving the shell that supplies it** (lessons "Pin a loop shell's wiring…"). Editor tests use a console whose `view`/`stdout`/`stderr` is a real `newLiveScreen(&tty, 20, 60)` with `interval = -1`, as `bilingual_paths_test.go:34` does, and read frames with `lastFrame` (`screen_test.go:552`); the rig sets `d.scheme = newSchemeHolder(schemeState{})` and `attachScheme`s it; lookups need `opt.color`, `opt.tintOn` and a `tintSourceFixture` (`language_style_paths_test.go:28`) so rows are tinted at all.
  1. **TestRawEditorSchemeRepaintsWhatIsOnScreen** — `d.configDir` → a temp dir; pre-write a tinted row (`l.WriteOutput(renderedOutput{text: "hola\n", rows: []rowPaint{{tinted: true}}})`); drive `runEditor` with `/scheme light⏎` then Ctrl-C. The last frame's `hola` row carries `languageLight` and no `languageDark`; `l.PaintedTranscript()` carries `languageLight`; `<tmp>/scheme` reads `light`; the output contains `scheme light (saved)`.
  2. **TestRawEditorSchemeWithNowhereToSave** — `configDir` nil → `light (session only; not saved)`; the repaint is light. (Pins `cc.session` in the editor.)
  3. **TestRawEditorSchemeWriteErrorChangesNothing** — `configDir` → a path under a regular FILE, so `MkdirAll` fails → stderr `define: /scheme:`, the shade stays dark, nothing says saved.
  4. **TestRawEditorSchemeReportsNothingDetected** — bare `/scheme` → `dark (default: the terminal has not reported its background)`. (Pins `cc.fullScreen`.)
  Piped tests (5, 6) call `replLines` directly, so `run()` never builds a holder for them: set `d.scheme = newSchemeHolder(schemeState{})` there too, or `/scheme` fails with `errNoScheme`.
  5. **TestPipedSchemeSaves** — `replLines`: `/scheme light` → `scheme light (saved)`; a following lookup's tinted rows carry `languageLight`.
  6. **TestPipedSchemeWithNowhereToSave** — `replLines`, `configDir` nil → `light (session only; not saved)`, and a following lookup is light. (Pins the piped loop's `cc.session`.)
  7. **TestOneShotScheme** — `run([]string{"/scheme", "light"}, …)`: file written, exit 0; `run([]string{"/scheme"})` with the file → `light (saved)`; `configDir` nil → exit 2, stderr names `$XDG_CONFIG_HOME`; `run([]string{"-scheme", "light", "/scheme", "dark"})` → `dark (saved)`.
- [x] **Step 2: Run — FAIL. Step 3: Implement.**

```go
// M2's wording. M3 adds "so define follows what the terminal reports" to auto.
const schemeUsage = "With nothing, the colour scheme in use and where it came from. light or dark sets it and saves it for every session; auto forgets the saved choice. The scheme picks the shade of the language tint: dark grey on a dark background, light grey on a light one."

// runScheme is /scheme (#70).
func runScheme(c commandCtx, args []string) int {
	if len(args) > 1 {
		fmt.Fprintf(c.stderr, "define: /scheme takes one of light, dark or auto, not %q\n", strings.Join(args, " "))
		return 2
	}
	if len(args) == 0 {
		fmt.Fprintf(c.stdout, "  scheme %s\n", describeScheme(c.scheme.Load(), c.fullScreen))
		return 0
	}
	arg, err := parseSchemeArg(args[0])
	if err != nil {
		fmt.Fprintf(c.stderr, "define: /scheme: %v\n", err)
		return 2
	}
	st, err := applyScheme(c.scheme, arg, c.schemePersister, c.session)
	if err != nil {
		fmt.Fprintf(c.stderr, "define: /scheme: %v\n", err)
		return 2
	}
	fmt.Fprintf(c.stdout, "  scheme %s\n", describeScheme(st, c.fullScreen))
	return 0
}
```
  Registry: append `{name: "scheme", summary: "light or dark terminal background", args: "[light|dark|auto]", usage: schemeUsage, run: runScheme}` at the END of `commands` (the table has no order; `/help` and the docs list it in table order, the menu sorts). `commandCtx` fields, documented in the file's style: `scheme *schemeHolder`; `schemePersister schemePersister`; `session bool` ("a loop exists for a session-only choice to live in; false for the one-shot"); `fullScreen bool` ("the raw editor: the loop that asks the terminal for its background, from M3"). `newCommandCtx`: `scheme: d.scheme, schemePersister: d.schemePersister()`. Editor (`replraw.go`, beside `cc.setTimes`): `cc.session = true` and `cc.fullScreen = true` on SEPARATE lines — the existing `draw()` after dispatch repaints from the holder, which is the whole recolour. Piped loop (`repl.go`, beside `cc.setTimes`): `cc.session = true`.
  Docs, in this commit so it stays green: run `go test ./cmd/define -run 'TestDocs' -count=1` — `TestDocsQuoteTheCommandList` and `TestDocsQuoteTheCommandUsage` name the README and atlas spans to update; update them. Add a README "Light or dark" paragraph for M2 (the shade follows `-scheme`, then the saved `/scheme`, then dark; the file's location; `/scheme auto`), and one `-h` sentence naming `/scheme`.
- [x] **Step 4: PASS**, full package. **Step 5: Commit** `#70 M2: /scheme switches, saves, and repaints what is already on screen`
- [x] **Step 6: Mutations**, one at a time: delete the editor's `cc.session = true` → test 2 reddens; delete `cc.fullScreen = true` → test 4 reddens; delete the piped loop's `cc.session = true` → test 6 reddens; make `applyScheme` skip its `h.choose` → test 1 reddens (no repaint, no transcript change). Restore each.

### Task 11: Harness isolation

**Files:** Modify `cmd/define/pty_conformance_test.go` (`startDefineBinary`), `cmd/define/pty_layout_conformance_test.go` (its own `exec.Command`).

- [x] **Step 1:** `startDefineBinary` ALWAYS sets `cmd.Env = append(append(os.Environ(), "XDG_CONFIG_HOME="+t.TempDir()), env...)` — the caller's `env` comes last, so a test can override it (`os/exec` keeps the LAST duplicate key). The layout test does the same. The comment says why: a developer's saved scheme would otherwise flip every "default" expectation.
- [x] **Step 2: A pty case that depends on the harness DEFAULT** — without one, nothing notices if the harness line goes (every shade-checking pty test passes a flag, which beats a saved choice). Add a `default` subtest to `TestPTYLanguageTint` with NO scheme flag, expecting `languageDark` and a bare `/scheme` reporting `dark (default: the terminal has not reported its background)`.
- [x] **Step 3:** **TestPTYSavedSchemeSurvivesARestart** (tag `darwin && conformance`; skip via `bilingualNativeProbe` like `TestPTYLanguageTint`), all three runs passing ONE shared `XDG_CONFIG_HOME=<dir>` in `env` and one Spanish deck: run 1 `/scheme light`, wait for `scheme light (saved)`, quit; run 2 looks up `red` → the Spanish section carries `languageLight`; `/scheme auto`, wait for its report, quit; run 3 → `languageDark`.
- [x] **Step 4:** `go test ./cmd/define/... -count=1`; `go test -tags conformance ./cmd/define -run 'TestPTY' -count=1` — PASS; name what ran vs skipped.
- [x] **Step 5: Commit** `#70 M2: the pty harness gets its own config directory`
- [x] **Step 6: Mutation — never touching the real config:** drop the harness's `XDG_CONFIG_HOME` line, then run `XDG_CONFIG_HOME=$(mktemp -d) sh -c 'mkdir -p "$XDG_CONFIG_HOME/define" && echo light > "$XDG_CONFIG_HOME/define/scheme" && go test -tags conformance ./cmd/define -run TestPTYLanguageTint/default -count=1'` (the harness appends `os.Environ()`, so the binary inherits it) → the `default` subtest reddens. Restore.

### Task 12: M2 boundary

- [ ] Full suite, `-race`, `go vet ./...`, `go vet -tags conformance ./cmd/define`, `GOOS=linux go build ./...`.
- [ ] Atlas: `/scheme` under **Command mode**; the saved file under **The store** ("not the deck — the user's config directory", precedence flag → saved → dark).
- [x] `sdlc milestone-close --issue 70 --milestone M2` — read the verdict before ticking.

---

## Chunk 3: M3 — detection

### Task 13: `parseBackgroundColour`

**Files:** Create `cmd/define/scheme_detect.go` (imports `strconv`, `strings`, `store`); test `cmd/define/scheme_detect_test.go`.

- [x] **Step 1: Failing test** **TestParseBackgroundColour** (table): `rgb:ffff/ffff/ffff` → light; `rgb:0000/0000/0000` → dark; `rgb:1e1e/1e1e/1e1e` → dark; `rgb:fdf6/e3e3/e3e3` → light; `rgb:f/f/f` → light; `rgb:80/80/80` → light (0.502) and `rgb:7f/7f/7f` → dark (0.498) — the boundary that separates Rec. 601 on encoded values from linear luminance; rejects: `rgba:ffff/ffff/ffff/ffff`, `#ffffff`, `rgb:fffff/0/0`, `rgb:ff/ff`, `rgb:gg/00/00`, `rgb:`, empty.
- [x] **Step 2: FAIL. Step 3: Implement.**

```go
// parseBackgroundColour reads an OSC 11 reply's payload (#70). Only the rgb:
// form with 1-4 hex digits per component is a colour; anything else (rgba:, #hex,
// garbage) is not an answer, and the caller swallows it without detecting.
//
// Classification is Rec. 601 luma on the gamma-ENCODED components, below 0.5
// dark — Neovim's background heuristic. Linear luminance would call a mid-grey
// terminal dark, which is not what its user calls it.
func parseBackgroundColour(payload string) (store.Scheme, bool) {
	rest, ok := strings.CutPrefix(payload, "rgb:")
	if !ok {
		return "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) != 3 {
		return "", false
	}
	var c [3]float64
	for i, p := range parts {
		if len(p) < 1 || len(p) > 4 {
			return "", false
		}
		v, err := strconv.ParseUint(p, 16, 16)
		if err != nil {
			return "", false
		}
		c[i] = float64(v) / float64(uint64(1)<<(4*len(p))-1)
	}
	if 0.299*c[0]+0.587*c[1]+0.114*c[2] < 0.5 {
		return store.SchemeDark, true
	}
	return store.SchemeLight, true
}
```
- [x] **Step 4: PASS. Step 5: Commit** `#70 M3: read a terminal's background colour as light or dark`
- [ ] **Step 6: Mutations:** replace the Rec. 601 sum with linear luminance (square each component, weights 0.2126/0.7152/0.0722) → the `80`/`7f` pair reddens; accept 5 digits → the 5-digit reject reddens. Restore each.

### Task 14: The decoder swallows the reply, then parses it

**Files:** Modify `cmd/define/key.go` (`KeyBackground` before the `numKeyKinds` sentinel; `Key.Background store.Scheme`; `case ']'` in `decodeEscape`), `cmd/define/play_loop.go` (`sittingKeyHandling` row only); tests `cmd/define/key_test.go`, the `readInput` tests.

- [ ] **Step 1: Failing tests.**
  - **TestDecodeBackgroundReply**: `"\x1b]11;rgb:ffff/ffff/ffff\x07"` → `KeyBackground`/light, consumed = len; the same with ST `"\x1b\\"` and with the 8-bit ST `"\x9c"`; EVERY strict prefix of each → `used == 0`.
  - **TestDecodeBackgroundReplyOtherFormats**: `rgba:…` and `#ffffff` payloads → ONE `KeyUnknown` consuming the whole reply.
  - **TestDecodeOSCAbortsAsToday**: `"\x1b]x"` → `KeyUnknown` (2 bytes), then `KeyRune 'x'`; `"\x1b]11;rgb\x03"` → `KeyUnknown` (2) and, decoding on, `KeyInterrupt` with nothing waiting; DEL (`0x7f`) and `0x80` in the payload abort; `ESC` then anything but `\` aborts.
  - **TestDecodeOSCCap**: a 64-byte reply cannot carry a valid `rgb:` payload, so assert the CONSUMED COUNT, not the kind: exactly 64 bytes is consumed whole (`n == 64`, `KeyUnknown`) with BEL and with ST; 65 bytes is refused with each (`n == 2`).
  - **TestReadInputBackgroundAcrossWrites** (`io.Pipe`, separate writes): `"\x1b]"` then `"\x03"` → `KeyUnknown`, `KeyInterrupt`; `"\x1b]"` then `"a"` → `KeyUnknown`, `KeyRune 'a'`; a reply split in three writes → one `KeyBackground`.
  - `TestEveryKeyKindIsDecidedForASitting` reddens as soon as `KeyBackground` exists — watch it fail, then add the row.
  - Fuzz: add seeds `"\x1b]"`, `"\x1b]11;"`, `"\x1b]11;rgb:ffff/ffff/ffff\x07"`, `"\x1b]11;rgba:0/0/0/0\x1b\\"` to all three targets (`FuzzDecodeKeyNeverLeaksEscapeTails`, `FuzzDecodeKey`, `FuzzDecodeMouseIsBounded`), and to `FuzzDecodeKey` the invariant: a `KeyBackground` only when the consumed bytes start with `"\x1b]11;rgb:"` and end in BEL or ST.
- [ ] **Step 2: FAIL. Step 3: Implement** in `key.go`:

```go
// oscBackgroundReply is the front of the terminal's answer to backgroundQuery.
const oscBackgroundReply = "\x1b]11;"

// maxOSCReply bounds a reply, terminator included. A real one is about 25 bytes
// (rgba: about 30); past this it is not a reply.
const maxOSCReply = 64

// decodeOSC decodes ESC ] … (#70). It SWALLOWS only the reply to the question
// this program asks, and in two steps so no reply FORMAT can leak as typing — in
// a sitting a leaked character is an answer:
//
//  1. swallow: ESC ] 11 ; then bytes in 0x20-0x7E up to BEL, ST (ESC \) or the
//     8-bit ST 0x9C (never legal inside a payload), at most maxOSCReply bytes in
//     all. A sequence longer than that is not a reply (a real one is ~25 bytes),
//     so it decodes as it always has — the one way past this rule, and it takes
//     a terminal no one ships. Any other byte — Ctrl-C, Enter, DEL, 0x80+,
//     an ESC not followed by \ — or the cap ABORTS, and the input decodes exactly
//     as it always has: ESC ] as a 2-byte KeyUnknown, then the rest. So Alt-]
//     with meta-sends-escape, then typing or Ctrl-C, behaves as before; a user
//     would have to type "11;" straight after Alt-] to enter the swallow at all.
//  2. parse: rgb: → KeyBackground; any other payload → KeyUnknown, swallowed,
//     nothing detected.
//
// While the buffer is still a PREFIX of a reply it returns 0 and waits. 0x03 is
// never part of one, so Ctrl-C always aborts in the same pass (lessons: "Trusted
// ANSI parsing is not untrusted control filtering").
func decodeOSC(buf []byte) (Key, int) {
	abort := func() (Key, int) { return Key{Kind: KeyUnknown, Raw: buf[:2]}, 2 }
	for i := 2; i < len(oscBackgroundReply); i++ {
		if i >= len(buf) {
			return Key{}, 0
		}
		if buf[i] != oscBackgroundReply[i] {
			return abort()
		}
	}
	for i := len(oscBackgroundReply); i < len(buf); i++ {
		if i >= maxOSCReply {
			return abort()
		}
		switch c := buf[i]; {
		case c == 0x07 || c == 0x9c:
			return backgroundKey(buf[len(oscBackgroundReply):i], buf[:i+1]), i + 1
		case c == 0x1b:
			if i+1 >= maxOSCReply {
				return abort() // ST would end past the cap
			}
			if i+1 == len(buf) {
				return Key{}, 0
			}
			if buf[i+1] != '\\' {
				return abort()
			}
			return backgroundKey(buf[len(oscBackgroundReply):i], buf[:i+2]), i + 2
		case c < 0x20 || c > 0x7e:
			return abort()
		}
	}
	if len(buf) >= maxOSCReply {
		return abort()
	}
	return Key{}, 0
}

func backgroundKey(payload, raw []byte) Key {
	if s, ok := parseBackgroundColour(string(payload)); ok {
		return Key{Kind: KeyBackground, Background: s, Raw: raw}
	}
	return Key{Kind: KeyUnknown, Raw: raw}
}
```
  `decodeEscape`: add `case ']': return decodeOSC(buf)` (after the `len(buf) < 2` guard). `KeyKind`: `KeyBackground` with the doc "a terminal REPORT, not a keystroke: the answer to backgroundQuery. Never typing, never an answer, never cancels a gesture." `sittingKeyHandling`: `KeyBackground: false` under "a terminal report, intercepted before toInput (Task 16) — never an answer".
- [ ] **Step 4: PASS**: `go test ./cmd/define -run 'Key|Decode|Fuzz|Sitting|ReadInput' -count=1`; then fuzz each target 30 s, anchored so only one matches: `go test ./cmd/define -run '^$' -fuzz '^FuzzDecodeKey$' -fuzztime 30s`, likewise `'^FuzzDecodeKeyNeverLeaksEscapeTails$'` and `'^FuzzDecodeMouseIsBounded$'`.
- [ ] **Step 5: Commit** `#70 M3: the key decoder reads a background report, bounded byte by byte`
- [ ] **Step 6: Mutations:** drop the `c < 0x20 || c > 0x7e` abort → the DEL/Ctrl-C cases redden; drop the ESC-case cap check → the 65-byte ST case reddens; make `backgroundKey` return `KeyBackground` for any payload → the `rgba` case reddens. Restore each.

### Task 15: Ask the question at raw-mode entry

**Files:** Modify `cmd/define/rawterm.go`, `cmd/define/replraw.go` (`newConsole`, `replRaw`), `cmd/define/play_loop.go:115`, and every test caller of `newConsole(` (`grep -n 'newConsole(' cmd/define/*_test.go`); tests `cmd/define/rawterm_test.go`, `cmd/define/key_test.go`.

- [ ] **Step 1: Failing tests.**
  - **TestNewConsoleAsksEveryQuery**: one recorder for BOTH `control` and the console's stdout (production writes both to the tty); with `askTerminal=true`, everything `newConsole` writes to control after the mode enables is exactly the concatenation of `terminalQueries` (so the sends and the list cannot drift); with `false`, none of it.
  - **TestWantsBackground**: true for `{tty: true, color: true, tintOn: true}`; false with `tintOn` false, with `raw` true, with `tty` false.
  - **TestASittingDoesNotAskAgain**: `/play` from an editor built through `newConsole` with `askTerminal=true` (Task 5's pattern) → exactly ONE `backgroundQuery` in the shared recorder.
  - Extend `TestEveryEnabledInputModeIsDecoded` with a SEPARATE loop over `terminalQueries` keyed by name (its regex reads `?NNNNh` modes and cannot match an OSC query): the reply table gains `"background colour": {{"rgb with BEL", "\x1b]11;rgb:ffff/ffff/ffff\x07"}, {"rgb with ST", "\x1b]11;rgb:0/0/0\x1b\\"}, {"rgba", "\x1b]11;rgba:ffff/ffff/ffff/ffff\x1b\\"}}`; a query with no row FAILS (closed); every sample decodes with no `KeyRune`.
- [ ] **Step 2: FAIL. Step 3: Implement.**

```go
// backgroundQuery asks the terminal for its background colour (OSC 11, #70).
// The answer arrives as input, whenever it arrives; nothing waits for it.
const backgroundQuery = "\x1b]11;?\x1b\\"

// terminalQueries is every QUESTION this program asks a terminal. Not modes —
// there is nothing to tear down — but they share enabledModes' obligation: a
// terminal that answers owes the decoder a case, and
// TestEveryEnabledInputModeIsDecoded derives from both lists. Every query here
// is about the tint today, so one condition (wantsBackground) governs them all;
// a query with a different condition gets its own.
var terminalQueries = []struct{ name, query string }{
	{"background colour", backgroundQuery},
}

// ask writes a query where the modes go. NEVER through a screen: scanEscape
// reads ESC ] as a 2-byte escape, so the rest would be painted as text.
func (r *rawSession) ask(query string) {
	if r == nil || r.control == nil {
		return
	}
	fmt.Fprint(r.control, query)
}

// wantsBackground: ask only where a tint can appear — colour on, the tint on,
// not -raw. TERM=dumb already turned tintOn off at flag parse.
func wantsBackground(opt options) bool { return opt.tty && opt.color && opt.tintOn && !opt.raw }
```
  `newConsole(ctx, d, sess, stdout, newScreen, askTerminal bool)` (not `ask`, which would shadow the package's `ask` function): after `sess.enterModes()`, `if askTerminal { for _, q := range terminalQueries { sess.ask(q.query) } }`. Callers: `replRaw` and `runPlay` pass `wantsBackground(opt)`; test callers pass `false` unless testing the query. `sittingInPlace` has no `sess`, so it cannot ask.
- [ ] **Step 4: PASS. Step 5: Commit** `#70 M3: a full-screen session asks the terminal for its background, once`
- [ ] **Step 6: Mutations:** drop the `askTerminal` loop → **TestNewConsoleAsksEveryQuery** reddens; `wantsBackground` ignores `tintOn` → **TestWantsBackground** reddens. (The callers' `wantsBackground(opt)` arguments — `replRaw` AND `runPlay` — are pinned by Task 17's **TestPTYNoQueryWithoutATint**.) Restore each.

### Task 16: Every consumer of the new key kind

**Files:** Modify `cmd/define/replraw.go` (`runEditor`), `cmd/define/play_loop.go` (`playSession` intercept), `cmd/define/selection_input.go` (`cancelPointerInput`, the length-check drop site); tests beside the existing loop, sitting and saturation tests.

- [ ] **Step 1: Failing tests** — the loops take a key channel built by **`readInput` over reply BYTES**, not a hand-built `Key`, so the decoder, the loop and the consumer are one path (Done-when: "driving `runEditor` with the reply as input").
  1. **TestRawEditorBackgroundReplyRepaints**: a live-screen console (Task 10's pattern) with a tinted row on screen; input bytes = `"\x1b]11;rgb:ffff/ffff/ffff\x1b\\" + "parrot\r"` then EOF (`parrot` has a fixture, so the lookup hits) → the row repaints in `languageLight`, and the word looked up is exactly `parrot` — wrap `d.dict` in the existing `countingDict` (`optionpool_test.go:17-27`, which records `words`) and assert `words == ["parrot"]`: no reply byte reached the line. Keep background preparation OFF in this rig (it can call `d.dict.Lookup`, `harvest.go:590`, and `countingDict` is not goroutine-safe). Repeat with BEL, and with an `rgba:` reply (→ looked up `parrot`, shade stays dark). With the holder chosen by flag dark, the rgb light reply → the row stays `languageDark`.
  2. **TestASittingIgnoresABackgroundReply**: `playSession` over `readInput` of `reply + "1"` (or the rig's first valid answer key) → exactly one answer recorded, the intended one; the sitting's screen repaints light. Repeat with `rgba:` → one answer, and the shade stays dark (no `languageLight` — the answer key itself repaints, so "no repaint" is not the observable).
  3. **TestAReplyDuringPlayReachesTheEditor**: through `newConsole` with `playRig` and the real `sittingInPlace` (`selection_nested_test.go:56` pattern), keys from a scripted channel (`scriptKeys`/`keySeq`) so the order is explicit: `/play⏎`, the decoded `KeyBackground` light, then Ctrl-C to end the sitting; after it ends, `/scheme` in the editor reports `light (detected)` and the editor's frame paints light.
  4. **TestAReplyMidDragKeepsTheSelection**: press + motion, then a `KeyBackground` through `route`, then release → the gesture still completes to a copy.
  5. **TestADroppedReplyIsSilent**, deterministic on an `io.Pipe`: write 256 × `x`; write the reply; then a ZERO-LENGTH write as the barrier — `io.Pipe` delivers it as a `Read`, so its return proves `readInput` finished the reply chunk and came back for more; assert `selectionNotice == ""` (no notice for the reply); THEN write `y` and `waitFor` the "input full" notice (the dropped `y` posts it, so the reply did not set `saturated`); close. (Asserting after `y`'s write instead races `y`'s own drop notice.)
- [ ] **Step 2: FAIL. Step 3: Implement.**
  - `runEditor`, first in `case k, open := <-keys:` after the `!open` check:

```go
// A terminal REPORT, not a keystroke (#70): apply it and repaint only if the
// shade on screen changed. Never Apply, never history, never a viewport key.
if k.Kind == KeyBackground {
	if d.scheme.detect(k.Background) {
		draw()
	}
	continue
}
```
  - `playSession`: the same intercept where it intercepts `KeyClick` and paging before `toInput`, calling its `show()`.
  - `selection_input.go` — ONE guard, in `cancelPointerInput` (both `route` and `cancelInput` reach it; a second guard in `route` would hide a mutation of this one): `if k.Kind == KeyUnknown || k.Kind == KeyBackground { return }`. At the length-check drop site (`if !isPointerKey(k.Kind) && len(out) == cap(out) {`), first statement: `if k.Kind == KeyBackground { continue } // a report, not typing: no notice, and saturated untouched`. The `select`'s `default:` arm is NOT guarded: `readInput`'s goroutine is `out`'s only sender and the length check diverts every non-pointer key first, so only a router-made `KeyClick` reaches it — a guard there could never be exercised (lessons "An untestable branch is an unreachable knob"). Say so in a comment there, and record in the issue Log that this revises the spec's "both drop sites".
- [ ] **Step 4: PASS. Step 5: Commit** `#70 M3: a background report reaches every consumer as a report, never as typing`
- [ ] **Step 6: Mutations**, one at a time: remove the `runEditor` intercept → test 1 reddens (the lookup is not `parrot`, or no repaint); remove the `playSession` intercept → test 2 reddens; remove the `cancelPointerInput` clause → test 4 reddens; remove the length-check clause → test 5 reddens. Restore each.

### Task 17: Conformance and docs

**Files:** `cmd/define/pty_conformance_test.go`; the test terminal readers; `cmd/define/README.md`; `atlas/define.md`; `cmd/define/main.go` (`-h`, flag help); `cmd/define/scheme_cmd.go` (usage).

- [ ] **Step 1: Test readers.** No current reader sees the query today (in-process tests use a separate `control`; the pty readers go through `lastFrame`, and the query precedes the first frame), so **TestReadersSkipOSC** is the proof: a captured stream beginning with `backgroundQuery` reads the same frame and the same `unstyled` text as without it. Make `lastFrame`/`readFrame`/`rowTestCells` skip `ESC ] … BEL|ST` as a terminal does, and add OSC stripping to `unstyled` (`screen_test.go:532,544` — an SGR-only regex today; its doc protects cursor and erase sequences, which this leaves alone). (`scanEscape`/`stripANSI` treat `ESC ]` as 2 bytes — do not assert on a raw stream through them.)
- [ ] **Step 2: pty tests** (tag `darwin && conformance`, `bilingualNativeProbe` skip, a Spanish deck as `TestPTYLanguageTint` sets up). `awaitActivityPTY` drains what it reads, so wait for the query AND the prompt in ONE predicate.
  - **TestPTYBackgroundDetection**: subtests light (`"\x1b]11;rgb:ffff/ffff/ffff\x1b\\"`), dark (`"\x1b]11;rgb:0000/0000/0000\x07"`), silent (no reply): look up `red` → the Spanish section carries `languageLight` / `languageDark` / `languageDark`; `/scheme` → `light (detected)` / `dark (detected)` / `dark (default: the terminal has not reported its background)`. And a late reply: light AFTER the entry is on screen → it repaints light.
  - **TestPTYNoQueryWithoutATint** (no `bilingualNativeProbe` gate — it needs no Oxford dictionary; every case except the dumb one sets `TERM=xterm-256color` explicitly, as `TestPTYLanguageTint` does, so an inherited `TERM` cannot flip a default case): for the EDITOR, the query IS sent by default and is NOT sent with `-language-tint=off`, with `TERM=dumb`, or with `-raw` (wait for the prompt, then assert the stream so far has no `"\x1b]11;?"`); for `--play` (seed a due word as the existing `--play` pty tests do), sent by default and not with `-language-tint=off`.
- [ ] **Step 3:** `go test -tags conformance ./cmd/define -run 'TestPTY' -count=1`, then with `CONFORMANCE_STRICT=1` — PASS; name what ran.
- [ ] **Step 4: Mutations:** `replRaw` passes `false` → **TestPTYNoQueryWithoutATint**'s editor default case reddens (and **TestPTYBackgroundDetection** times out where it runs); `runPlay` passes `false` → its `--play` default case reddens; `runPlay` passes `true` → its `-language-tint=off` case reddens. Restore each.
- [ ] **Step 5: Docs.** `schemeUsage`: auto "forgets the saved choice, so define follows what the terminal reports". `-scheme` flag help: "auto (ask the terminal), dark, or light". README "Light or dark": detection in full-screen sessions, one-shot uses flag → saved → dark, the late-reply-after-a-fast-quit limit. Atlas: **The screen** gains detection (the query at mode entry via `rawSession.control`, the reply as `KeyBackground`, its consumers); **The line editor** gets the bounded OSC swallow. `go test ./cmd/define -run TestDocs -count=1` — PASS.
- [ ] **Step 6: Commit** `#70 M3: conformance plays light, dark and silent terminals; docs for detection`

### Task 18: M3 boundary and close

- [ ] Full suite, `-race`, `go vet ./...`, `go vet -tags conformance ./cmd/define`, `GOOS=linux go build ./...`, conformance with `CONFORMANCE_STRICT=1`.
- [ ] **Manual live conformance** (the real external dependency): in Terminal.app, iTerm2 and Ghostty, each in a light and a dark profile — `define`, `/lang es`, look up `red`; check the tint's shade and `/scheme`'s report; `/scheme light|dark|auto` and watch the repaint; quit and check the transcript's shade. Note any terminal that answers `rgba:` or nothing. RECORD the terminal × appearance matrix with the date in `atlas/define.md` (a short "Terminals checked" table beside the detection paragraph), and state there when it is re-run: when a terminal is added to the matrix, when a detection bug is reported, or when `decodeOSC`, `parseBackgroundColour` or `backgroundQuery` changes.
- [ ] Walk every `## Done when` bullet and name the test (or manual check) that proves it.
- [ ] `sdlc milestone-close --issue 70 --milestone M3`, read the verdict; then `sdlc close --issue 70 --verified '<evidence>'`.

## Revisions

- **2026-09-17, M1 boundary review (Minor, ARCH-ORDER).** `schemeState` shipped as `choice store.Scheme` + `chosenBy schemeSource`, which let a choice claim to be detected or default. It is now `choice *schemeChoice{value, by choiceSource}`, with `choiceSource` ∈ {`choiceFlag`, `choiceSaved`, `choiceSession`} mapping to the report's `schemeSource`. `withChoice` and `choose` take a `choiceSource`. Chunk 1's code above shows the shape as first written; Chunk 2's code is updated to the new names.
- **2026-09-17, M1 boundary review (Minor, ARCH-DRY).** The test-only `boardFooter` duplicated production's `boardFooterOutput` and held the only copy of its rationale; the two tests that used it now go through `paintedBoardFooterForTest` (the production path), the rationale moved onto `boardFooterOutput`, and `boardFooter` and `practiceChrome` are deleted.
- **2026-09-18, M2 boundary review (FIX-THEN-SHIP).** BR-6 (Important): the M2 evidence the ticked steps cite is now in the issue Log. The state-shape family's 2nd finding: `schemeArg` is `struct{ value store.Scheme }` with `auto()` (empty = auto) instead of `auto bool` + `value`; `commandCtx.session`/`fullScreen` are one `loop loopKind` {`loopOneShot`, `loopPiped`, `loopEditor`}, set by each loop (`cc.loop = loopEditor` / `loopPiped`). `schemePersister` gains `load()`, and the startup read is `initialSchemeState(flag, persister, warn)` — pure, pinned by `TestInitialSchemeState` — replacing the inline read in `run()`. Chunk 2's code above shows the first shapes.
- **2026-09-18, M2 close.** `schemeState` drops `heard`: `detected store.Scheme` is empty for nothing heard, so the reply and whether one arrived cannot disagree (the state-shape family's last instance in #70). M3's `detect` and `withDetected(v)` are unchanged in signature.
