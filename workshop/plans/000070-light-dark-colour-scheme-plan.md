# Light/Dark Colour Scheme Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `define` paints its target-language tint in the shade that suits the terminal's background — detected in a session, overridable by `-scheme` and a saved `/scheme` — and a switch recolours everything already on screen.

**Architecture:** A row records only WHETHER it is tinted (a role); the colour resolves at paint from ONE per-process scheme holder (`atomic.Pointer` to an immutable `schemeState`, pure transitions). The saved choice is a one-word file under the user's config directory. Detection is an `OSC 11` query sent at raw-mode entry whose reply the key decoder turns into a `KeyBackground` event — nothing ever waits for it.

**Tech Stack:** Go 1.26, `golang.org/x/term`, `sync/atomic`, `github.com/creack/pty` (conformance only).

**Spec:** `workshop/issues/000070-light-dark-colour-scheme.md` — `## Spec` and `## Done when` are the contract; this plan does not restate their reasoning.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `store.Scheme` (`SchemeDark`, `SchemeLight`, `ParseScheme`) | `cmd/define/store/scheme.go` | new |
| `schemeState`, `schemeSource`, `effective`, `withChoice`, `withoutChoice`, `withDetected` | `cmd/define/scheme.go` | new |
| `schemeArg`, `parseSchemeArg` | `cmd/define/scheme.go` | new |
| `parseTintFlag` | `cmd/define/scheme.go` | new |
| `describeScheme` | `cmd/define/scheme.go` | new (M2) |
| `applyScheme` | `cmd/define/scheme.go` | new (M2) |
| `configDirFrom` | `cmd/define/scheme.go` | new (M2) |
| `parseBackgroundColour` | `cmd/define/scheme_detect.go` | new (M3) |
| `decodeOSC` / `KeyBackground` / `Key.Background` | `cmd/define/key.go` | new (M3) |
| `rowPaint` (`background string` → `tinted bool`) | `cmd/define/output_layout.go` | modified |
| `tintPolicy` (`background string` → `on bool` + `scheme *schemeHolder`) | `cmd/define/language_style.go` | modified |
| `play.PresentationRegion` (`Background string` → `Tinted bool`) | `cmd/define/play/presentation.go` | modified |
| `paintLanguageRow` (+ `sc store.Scheme`) | `cmd/define/language_row.go` | modified |
| `schemeTint` | `cmd/define/language_style.go` | new |
| `styleLanguageText`, `lineInkBounds`, `tintProfile`, `dictionaryFragment`, `RenderOpts.dictionaryText` | `language_style.go`, `dictionary_language.go` | deleted |
| `renderOutputText`, `renderDefinitions`, `renderPracticePresentation`, `practiceChrome`, `boardFooter`, `styledBoardPrompt` | production files | deleted (the used ones move to `render_helpers_test.go`) |

- **`store.Scheme`** — the closed enum naming the terminal background, as persisted.
  - **Relationships:** read by `schemeState`, `ReadScheme`/`WriteScheme`, `Key.Background`.
  - **DRY rationale:** one enum for the file, the flag, the command and the decoder — `store.Lang`'s shape (the persisted enum lives in `store`).
  - **Future extensions:** a third value (e.g. high-contrast) widens here and in `schemeTint`.
- **`schemeState`** — immutable value: an explicit `choice` (source `flag|saved|session`) and a `detected` value. `effective()` = choice, else detected, else dark, plus the source. ARCH-ORDER: all transitions are methods returning a new value; nothing mutates in place.
  - **Relationships:** 1 per process, held by `schemeHolder`.
  - **Future extensions:** DEC mode 2031 notifications are another `withDetected` event.
- **`applyScheme`** — the `/scheme` transition with persistence first (the `/bilingual` rule): takes the holder, the parsed arg, a `schemePersister` (nil = nowhere to save) and whether a session exists.
- **`parseBackgroundColour`** — `rgb:R/G/B` (1–4 hex digits each) → Rec. 601 luma → dark/light.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `schemeHolder` | `cmd/define/scheme.go` | new | cross-goroutine state (atomic pointer) |
| `store.ReadScheme` / `WriteScheme` / `ClearScheme` | `cmd/define/store/scheme.go` | new (M2) | filesystem |
| `deps.configDir` | `cmd/define/main.go` | new (M2) | `$XDG_CONFIG_HOME` / `$HOME` |
| `schemePersister` (`dirSchemePersister`) | `cmd/define/scheme.go` | new (M2) | the store functions |
| `deps.scheme` | `cmd/define/main.go` | new | the process's holder |
| `screen.scheme` + `liveScreen.attachScheme` | `cmd/define/screen.go` | new | the painter's read of the holder |
| `backgroundQuery` / `terminalQueries` / `rawSession.ask` | `cmd/define/rawterm.go` | new (M3) | the terminal |

- **`schemeHolder`** — `atomic.Pointer[schemeState]`. ONE writer at a time (the loop goroutine in force); five painting goroutines read. Nil holder = dark, read-only.
  - **Injected into:** `deps` (built by `run()`), attached to every production screen, carried by `tintPolicy` to the non-screen writer.
- **`schemePersister`** — `save(store.Scheme) error`, `clear() error`. Production: `dirSchemePersister(dir)` over the store functions. Tests: `fakePersister` (stateful: holds the saved value, can be told to fail). **ARCH-MOCK** — the filesystem half is covered by `store` tests against `t.TempDir()`; the command logic runs against the fake.
- **The terminal (M3)** — the external dependency. Seam: bytes in (`readInput`'s reader / a key channel) and bytes out (`rawSession.control`). Fake: in-process tests write the reply bytes into the input; pty conformance tests play a light, a dark and a silent terminal. Live conformance: the manual check in Terminal.app, iTerm2 and Ghostty (Task 18).

### Conventions for every task

- Run tests from the repo root: `go test ./cmd/define/... -count=1` (the package dir is where `go test` stands — lessons "`go test` runs in the package directory").
- **Mutation checks follow lessons.md "Mutation testing needs a COMMITTED baseline":** commit first, mutate, run with `-count=1`, confirm the mutation APPLIED and COMPILED, see red, restore with `git checkout HEAD -- <path>`.
- Commits: `#70 Mx: <subject>`, body = why, trailer `Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`. Read `git show --stat` before each commit.

---

## Chunk 1: M1 — the tint is a role, the scheme is state

Behaviour after M1: identical to today except the flags — `-scheme dark|light|auto` picks the shade (auto = dark until M3), `-language-tint on|off`, and `-language-tint dark|light` is refused naming `-scheme`.

### Task 1: `store.Scheme`

**Files:**
- Create: `cmd/define/store/scheme.go`
- Test: `cmd/define/store/scheme_test.go`

- [ ] **Step 1: Write the failing test**

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

- [ ] **Step 2: Run it — expect FAIL (`undefined: ParseScheme`)**

Run: `go test ./cmd/define/store -run TestParseScheme -count=1`

- [ ] **Step 3: Implement**

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

- [ ] **Step 4: Run it — expect PASS.** Same command.
- [ ] **Step 5: Commit** `#70 M1: store: a Scheme enum for the terminal background`

### Task 2: `schemeState` and `schemeHolder`

**Files:**
- Create: `cmd/define/scheme.go`
- Test: `cmd/define/scheme_test.go`

- [ ] **Step 1: Write the failing tests** — the transition table as event sequences (ARCH-ORDER), and the holder.

```go
package main

import (
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
	h.Store(h.Load().withChoice(store.SchemeDark, sourceSaved))
	if h.detect(store.SchemeLight) {
		t.Error("with a choice in force a reply changes nothing visible")
	}
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
(add `"strings"` to the imports)

- [ ] **Step 2: Run — expect FAIL (undefined symbols).** `go test ./cmd/define -run 'TestSchemeState|TestSchemeHolder|TestParseSchemeArg|TestParseTintFlag' -count=1`

- [ ] **Step 3: Implement `cmd/define/scheme.go`**

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

// Store must only be called on a non-nil holder by the loop in force.
func (h *schemeHolder) Store(s schemeState) { h.p.Store(&s) }

// Scheme is the value to paint with now.
func (h *schemeHolder) Scheme() store.Scheme {
	v, _ := h.Load().effective()
	return v
}

// detect applies a terminal report and says whether what is painted changed —
// the only case that is worth a repaint.
func (h *schemeHolder) detect(v store.Scheme) bool {
	if h == nil {
		return false
	}
	before := h.Load()
	next := before.withDetected(v)
	h.Store(next)
	a, _ := before.effective()
	b, _ := next.effective()
	return a != b
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

- [ ] **Step 4: Run — expect PASS.** Same command, then `go vet ./cmd/define`.
- [ ] **Step 5: Commit** `#70 M1: scheme state as an immutable value behind one atomic holder`

### Task 3: The tint becomes a role, resolved at paint

This is a TYPE change; the compiler enumerates the sites. Do it in one commit so the tree never builds half-migrated.

**Files (production, each site named by the spec review):**
- Modify: `cmd/define/output_layout.go` — `rowPaint{background string}` → `rowPaint{tinted bool}`; `validRowPaint` drops the background clause (line 24); `layoutOutput` copies `tinted` (line 80); `serializeOutput(o, width, sc store.Scheme)` passes `sc` to `paintLanguageRow`. Update the `renderedOutput` doc comment (line 10) to: *"WHETHER a row is tinted is decided when it is produced, so later policy changes (a /lang switch) do not recolour history; WHICH shade a tint is resolves at paint from the scheme in effect (#70), so a /scheme switch does."*
- Modify: `cmd/define/language_row.go` — `paintLanguageRow(text string, paint rowPaint, width int, sc store.Scheme)`: `if !paint.tinted { return text }`; write `schemeTint(sc)` where it wrote `paint.background`.
- Modify: `cmd/define/language_style.go` — `tintPolicy{lang store.Lang; on bool; scheme *schemeHolder}`; add `schemeTint(s store.Scheme) string` (light → `languageLight`, else `languageDark`); replace `options.tintFor` with `func tintFor(d deps, opt options) tintPolicy { return tintPolicy{lang: d.lang, on: opt.color && opt.tintOn, scheme: d.scheme} }`; delete `tintProfile`.
- Modify: `cmd/define/output_screen.go` — `writeOutput(w io.Writer, o renderedOutput, width int, sc store.Scheme)` (the `outputWriter` branch ignores `sc`: a screen reads its own holder); `paintOutputChunk(text, p, width, sc)` (`if !p.tinted`); `sliceRowPaint` copies `tinted`; `paintedTranscript` reads `sc := s.scheme.Scheme()` ONCE before the loop.
- Modify: `cmd/define/screen.go` — `screen` gains `scheme *schemeHolder`; `selectionLayout` gains `scheme store.Scheme`, set ONCE at the top of `layoutSelectionFrame` (`layout := selectionLayout{scheme: s.scheme.Scheme()}` — move the existing `layout := selectionLayout{}` line); `selectionLayout.paint` passes `layout.scheme` to both painters. Add:

```go
// attachScheme gives the screen the process's scheme holder (#70). Call it
// BEFORE the screen is shared with another goroutine — the pointer router, the
// resize watcher, the throttle timer — because this field is not atomic; the
// holder it points at is.
func (l *liveScreen) attachScheme(h *schemeHolder) { l.s.scheme = h }
```
- Modify: `cmd/define/replraw.go` `newConsole` — `live.attachScheme(d.scheme)` immediately after `live := newScreen(...)`, before `newPointerRouter`.
- Modify: `cmd/define/play_cmd.go` `sittingInPlace` — `sitting.attachScheme(d.scheme)` immediately after `sitting := newPinnedScreen(...)`, before `pointer.activate(sitting)`.
- Modify: `cmd/define/selection_frame.go:190` — compare `r.paint.tinted != s.paint.tinted`.
- Modify: `cmd/define/answerwrap.go` — line 228 `p.background = ""` → `p.on = false`; `finalize`: `tinted := !owner.mixed && owner.lang == target && w.policy.on`, `rows: []rowPaint{{tinted: tinted}}`, `writeOutput(w.out, ..., w.width, w.policy.scheme.Scheme())`.
- Modify: `cmd/define/definitions.go:125` — `paints[i].tinted = opt.Tint.on`.
- Modify: `cmd/define/practice_output.go` — line 96 `paint.tinted = policy.on`; line 116 `sectionBase.rows[row].tinted = region.Tinted`; line 142 `Tinted: paintAt(o.rows, i).tinted`; lines 150/157 `tintFor(d, opt)`.
- Modify: `cmd/define/play/presentation.go:26` — `Background string` → `Tinted bool` (doc: "a role; the shade is main's business").
- Modify: every `opt.tintFor(d.lang)` → `tintFor(d, opt)`: `ask.go:164`, `cloze.go:236`, `main.go:1076`, `play_loop.go:1083`, `practice_language.go:49,150`.
- Modify: `writeOutput` callers pass the scheme: `main.go:1150` and `practice_language.go:55,150` → `d.scheme.Scheme()`.
- Modify: `cmd/define/main.go` — `deps` gains `scheme *schemeHolder` with a doc comment citing the `practiceHelp` precedent ("a pointer, so a nested sitting and the editor share it; NOT the bilingual pattern, whose setter replaces the pointer in a by-value copy"). `options.tintBackground string` → `tintOn bool` (doc: "-language-tint; false under TERM=dumb").

- [ ] **Step 1: Write the failing tests** (in `cmd/define/language_row_test.go` and `cmd/define/output_screen_test.go`):

```go
// The role is frozen at production; the shade is resolved at paint (#70).
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

// A screen repaints history in the scheme in force at paint time, not the one in
// force when the row was written — the property /scheme depends on.
func TestAScreenRepaintsHistoryInTheCurrentScheme(t *testing.T) {
	var tty bytes.Buffer
	h := newSchemeHolder(schemeState{})
	l := newLiveScreen(&tty, 10, 20)
	l.interval = -1
	l.attachScheme(h)
	if err := l.WriteOutput(renderedOutput{text: "hola\n", rows: []rowPaint{{tinted: true}}}); err != nil {
		t.Fatal(err)
	}
	h.Store(h.Load().withChoice(store.SchemeLight, sourceSession))
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

- [ ] **Step 2: Run — expect FAIL to COMPILE** (`unknown field tinted`). `go test ./cmd/define -run 'TestATintedRow|TestAScreenRepaints' -count=1`
- [ ] **Step 3: Make the production changes listed above.** `go build ./cmd/define/...` until clean.
- [ ] **Step 4: Migrate the tests the compiler now rejects** (22 files, ~155 references — `grep -c 'languageLight\|languageDark\|tintPolicy{\|background:'`). Rules, applied so each test asserts what it asserted before:
  1. `rowPaint{background: languageDark|languageLight}` → `rowPaint{tinted: true}`; a later paint call receives `store.SchemeDark` / `store.SchemeLight` matching the constant the test used; `rowPaint{background: ""}` → `rowPaint{}`.
  2. `tintPolicy{lang, languageX}` → `tintPolicy{lang: lang, on: true, scheme: holderFor(store.SchemeX)}`; `tintPolicy{lang, ""}` → `tintPolicy{lang: lang}`.
  3. `.background != ""` / `== languageX` on a paint → `.tinted`; where the test really asserts the SHADE, assert it on the painted bytes.
  4. `play.PresentationRegion{Background: languageX}` → `{Tinted: true}`.
  5. Loops over `{dark: languageDark, light: languageLight}` profiles become loops over `store.Scheme` values, asserting `schemeTint(sc)` in the output.
  6. `paintLanguageRow`/`serializeOutput`/`writeOutput`/`paintOutputChunk` calls gain the scheme argument.
  Add the helper once, in `cmd/define/render_helpers_test.go`:

```go
// holderFor is a holder already set to one scheme, for tests that paint a shade.
func holderFor(s store.Scheme) *schemeHolder {
	return newSchemeHolder(schemeState{}.withChoice(s, sourceFlag))
}
```
  Do NOT touch `pty_*` or `language_style_paths_test.go` here — Task 5 owns the flag tests.
- [ ] **Step 5: Run the package** — `go test ./cmd/define/... -count=1`. Expect PASS except tests that go through `styleLanguageText` or the baked renderers (Task 4 owns them); record their names.
- [ ] **Step 6: Mutation check** (committed baseline first): make `paintLanguageRow` always use `languageDark`; `TestATintedRowTakesTheShade...` and `TestAScreenRepaintsHistory...` must redden. Then delete `live.attachScheme(d.scheme)` in `newConsole`: `TestAScreenRepaints...` does NOT see that (it attaches itself) — Task 11's loop-shell test is the pin for that wiring; note it in the Log. Restore with `git checkout HEAD -- <path>`.
- [ ] **Step 7: Commit** `#70 M1: a row records whether it is tinted; the shade resolves at paint`

### Task 4: Delete the dead tint paths

**Files:**
- Modify: `cmd/define/render.go` (lines 150, 203, 221, 244), `cmd/define/dictionary_language.go`, `cmd/define/language_style.go`, `cmd/define/output_screen.go`, `cmd/define/definitions.go`, `cmd/define/practice_language.go`, `cmd/define/play_loop.go`
- Create/extend: `cmd/define/render_helpers_test.go`

- [ ] **Step 1: Evidence before deletion** (lessons "Deleting a test needs the same evidence as writing one"). On a committed baseline, replace `styleLanguageText`'s body with `panic("unreachable in production (#70)")` and run `go test ./cmd/define/... -count=1`. Expected: ONLY `language_style_test.go`'s direct unit tests and tests that call `Render` with a non-zero `Tint` fail. Record the list in the issue Log; any OTHER failure means a live path — STOP and re-plan. Restore.
- [ ] **Step 2: Remove the tint from `Render`.** Replace each `opt.dictionaryText(e, <original>, <rendered>, …)` with its `<rendered>` argument (150: `t.Text`; 203: `opt.prose(wrapText(body, opt.Width, lead), "")`; 221: `opt.prose(wrapText(prettyPronunciations(ex.Text, p), opt.Width, len(indent)+2), p.ex)`; 244: delete the statement). Delete `RenderOpts.dictionaryText`, `dictionaryFragment`, `styleLanguageText`, and any helper left with no reference (`lineInkBounds`, and `validateLanguageText` if unreferenced — `grep -n` each). Keep `projectDictionaryText` (live at `bilingual_layout.go:105`) and `sourceBackground` (live in `paintLanguageRow`, `answerwrap.go:129`).
- [ ] **Step 3: Move the test-only renderers into `render_helpers_test.go`**, re-expressed over the production path, and delete them from production (`renderOutputText`, `renderDefinitions`, `renderPracticePresentation`, `practiceChrome`, `boardFooter`; delete `styledBoardPrompt` outright — no callers at all):

```go
// renderOutputTextIn is the tests' string view of a renderedOutput: each row
// painted at its own width, under an explicit scheme. Production paints only at
// the terminal boundary (serializeOutput, the screen).
func renderOutputTextIn(o renderedOutput, sc store.Scheme) string {
	lines := outputStyledRows(o.text)
	for i, line := range lines {
		lines[i] = paintLanguageRow(line, paintAt(o.rows, i), visibleCells(line), sc)
	}
	return strings.Join(lines, "\n")
}

// renderDefinitions renders through renderDefinitionOutput and paints in the
// scheme the policy carries (dark for a nil holder).
func renderDefinitions(set definitionSet, opt RenderOpts) (string, []Region) {
	o := renderDefinitionOutput(set, opt)
	return renderOutputTextIn(o, opt.Tint.scheme.Scheme()), o.regions
}

func renderPracticePresentation(p play.Presentation, lang, source store.Lang, policy tintPolicy, vocab Vocabulary, sf surface, subject string) string {
	return renderOutputTextIn(renderPracticeOutput(p, lang, source, policy, vocab, sf, subject, 0), policy.scheme.Scheme())
}
```
  `practiceChrome` and `boardFooter` move verbatim, calling these and `tintFor(d, opt)`.
- [ ] **Step 4: Port the dead-path assertions.** Tests that asserted a tint through `Render` or `styleLanguageText` (Step 1's list; the reviewer named `dictionary_source_test.go:14,66`, `dict_test.go:70,76`, `dictionary_language_test.go:146-201`, `language_style_test.go`) are rewritten against `renderDefinitionOutput` + `serializeOutput` (or `renderDefinitions` above) so the property — which rows are tinted, in which shade — is asserted on the live path. A `styleLanguageText` unit test with no live-path equivalent is deleted, and the Log names it and why.
- [ ] **Step 5: Run** `go test ./cmd/define/... -count=1` and `go vet ./...` — expect PASS. `grep -rn 'styleLanguageText\|dictionaryFragment\|renderOutputText\|styledBoardPrompt' cmd/define --include='*.go' | grep -v _test.go` — expect no output (lessons "Verify the deletion").
- [ ] **Step 6: Commit** `#70 M1: delete the tint paths production never reached`

### Task 5: Flags — `-scheme`, `-language-tint on|off`

**Files:**
- Modify: `cmd/define/main.go` (flag block ~line 524, validation ~605-611, `opt` literal ~660-690, usage text ~560)
- Modify: `cmd/define/language_style_paths_test.go`, `cmd/define/pty_conformance_test.go:1239-1245`, `cmd/define/pty_layout_conformance_test.go:40-43`

- [ ] **Step 1: Rewrite `TestLanguageTintInvocation` / `TestLanguageTintInvalidFlagBeforeStore` as the failing tests.** Cases (args → exit, shade in output):
  - none → 0, `languageDark` (auto with nothing detected);
  - `-scheme dark` → dark; `-scheme light` → `languageLight`; `-scheme auto` → dark;
  - `-language-tint off` → no tint escape at all;
  - `-language-tint light` → exit 2, stderr contains `-scheme light`, no store directory created;
  - `-language-tint bogus` → exit 2, `invalid -language-tint`;
  - `-scheme sepia` → exit 2, `not a colour scheme`;
  - `TERM=dumb` → no tint.
- [ ] **Step 2: Run — expect FAIL** (`flag provided but not defined: -scheme`). `go test ./cmd/define -run 'TestLanguageTint' -count=1`
- [ ] **Step 3: Implement in `run()`:**

```go
schemeFlag := fs.String("scheme", "auto", "terminal background: auto (ask the terminal), dark, or light")
languageTint := fs.String("language-tint", "on", "tint the target language's rows: on or off")
```
  After `fs.Parse`, replacing the `tintProfile` block:

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
  In the `opt` literal: `tintOn: tintOn`. Build the holder BEFORE anything renders (next to `opt`):

```go
var st schemeState
if !schemeChoice.auto {
	st = st.withChoice(schemeChoice.value, sourceFlag)
}
d.scheme = newSchemeHolder(st)
```
  Add one sentence to the `-h` prose: *"The language tint's shade follows the terminal's background: -scheme light or dark says which; the default asks the terminal."* (M2/M3 extend it.)
- [ ] **Step 4: Migrate the pty tests.** `TestPTYLanguageTint` profiles: `{dark: "-scheme=dark"}`, `{light: "-scheme=light"}`, `{off: "-language-tint=off"}`, asserting `schemeTint(...)`/no tint. Same for `TestPTYNativeRendirSectionLayout:43`.
- [ ] **Step 5: Run** `go test ./cmd/define/... -count=1` → PASS. Then conformance: `go test -tags conformance ./cmd/define -run 'TestPTYLanguageTint|TestPTYNativeRendir' -count=1` → PASS (skips are reported, not passes — say which ran).
- [ ] **Step 6: Mutation check:** drop `tintOn = false` for `TERM=dumb` → the dumb case reddens; restore.
- [ ] **Step 7: Commit** `#70 M1: -scheme picks the shade; -language-tint is on or off`

### Task 6: M1 boundary

- [ ] `go test ./... -count=1`, `go vet ./...`, `GOOS=linux go build ./...` — all green; quote the counts.
- [ ] `atlas/define.md`: rewrite the `-language-tint` line (`:2270`) and the `RenderOpts.Tint` row (`:578`) for the role/shade split, and add a short "The shade is a paint-time decision (#70)" paragraph under **The screen** naming `schemeHolder`, `attachScheme`, and the once-per-frame read.
- [ ] `sdlc milestone-close --issue 70 --milestone M1` — read the verdict before ticking `M1`; fix Critical/Important first.

---

## Chunk 2: M2 — `/scheme` and the saved choice

### Task 7: `store.ReadScheme` / `WriteScheme` / `ClearScheme`

**Files:** Modify `cmd/define/store/scheme.go`; test `cmd/define/store/scheme_test.go`.

- [ ] **Step 1: Failing tests** against `t.TempDir()`:
  - missing file → `("", false, nil)`;
  - `WriteScheme(dir, SchemeLight)` then `ReadScheme` → `(SchemeLight, true, nil)`, file content exactly `light\n`, and `dir` created by the write (it does not exist beforehand);
  - `"  DARK \n"` → `SchemeDark`;
  - `"sepia"`, `""`, 65 bytes of `a` → `("", false, err)` — a hand-edited or truncated file is an ERROR the caller warns about, never a silent default;
  - `ClearScheme` removes the file AND the now-empty `dir`; a second `ClearScheme` → nil; with a foreign file also in `dir`, `ClearScheme` removes only `scheme` and keeps `dir`.
- [ ] **Step 2: Run — expect FAIL.** `go test ./cmd/define/store -run Scheme -count=1`
- [ ] **Step 3: Implement**

```go
const schemeFileName = "scheme"
const maxSchemeSettingBytes = 64

// ReadScheme reads the saved scheme in dir (the user's config directory for
// define, NOT a deck). found is false with a nil error when nothing is saved.
// Anything unreadable, oversized or outside the enum is an error, so the caller
// can warn once and treat it as unset (ARCH-SECURE: a hand-edited or truncated
// file is untrusted input, parsed into the closed enum here).
func ReadScheme(dir string) (Scheme, bool, error) {
	f, err := os.Open(filepath.Join(dir, schemeFileName))
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxSchemeSettingBytes+1))
	if err != nil {
		return "", false, err
	}
	if len(b) > maxSchemeSettingBytes {
		return "", false, fmt.Errorf("%s is larger than a scheme name", filepath.Join(dir, schemeFileName))
	}
	s, err := ParseScheme(string(b))
	if err != nil {
		return "", false, err
	}
	return s, true, nil
}

// WriteScheme saves s atomically, creating dir if needed.
func WriteScheme(dir string, s Scheme) error {
	return writeBytesAtomic(filepath.Join(dir, schemeFileName), []byte(string(s)+"\n"))
}

// ClearScheme forgets the saved scheme and then the directory if that left it
// empty (ARCH-FUNERAL: the residue is at most this one file and its directory).
func ClearScheme(dir string) error {
	if err := os.Remove(filepath.Join(dir, schemeFileName)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	_ = os.Remove(dir) // fails harmlessly when something else lives there
	return nil
}
```
- [ ] **Step 4: Run — PASS.** **Step 5: Commit** `#70 M2: store: the saved scheme, one word in the user's config directory`

### Task 8: The config-directory seam and startup read

**Files:** Modify `cmd/define/scheme.go`, `cmd/define/main.go` (`deps`, `realDeps`, `run()`); test `cmd/define/scheme_test.go`, `cmd/define/language_style_paths_test.go`.

- [ ] **Step 1: Failing tests.** Pure `configDirFrom`:

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
  And through `run()` (in `language_style_paths_test.go`, with `d.configDir` pointing at a temp dir): a saved `light` with no flag → `languageLight` in a lookup; `-scheme dark` beats a saved `light`; a garbled file → a single `define: ignoring saved scheme` line on stderr and the dark shade; `d.configDir == nil` → dark, no error.
- [ ] **Step 2: Run — FAIL.**
- [ ] **Step 3: Implement.**

```go
// configDirFrom resolves define's user config directory. Only ABSOLUTE bases
// count: a relative XDG_CONFIG_HOME or HOME would put the file wherever the
// process happens to stand, which is the deck's directory — the one place this
// setting must not live.
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
  `deps` gains `configDir func() (string, bool)` — its OWN seam, not `getenv` (which is the model seam, and some tests make it panic: `practice_help_paths_test.go:66`). `realDeps`: `configDir: func() (string, bool) { return configDirFrom(os.Getenv) }`. Test deps leave it nil = no config. In `run()`, where Task 5 builds the holder:

```go
var st schemeState
switch {
case !schemeChoice.auto:
	st = st.withChoice(schemeChoice.value, sourceFlag)
case d.configDir != nil:
	if dir, ok := d.configDir(); ok {
		if v, found, err := store.ReadScheme(dir); err != nil {
			fmt.Fprintf(stderr, "define: ignoring saved scheme: %v\n", err)
		} else if found {
			st = st.withChoice(v, sourceSaved)
		}
	}
}
d.scheme = newSchemeHolder(st)
```
- [ ] **Step 4: Run — PASS. Step 5: Commit** `#70 M2: a saved scheme is read at startup, from its own seam`

### Task 9: `describeScheme` and `applyScheme`

**Files:** Modify `cmd/define/scheme.go`; test `cmd/define/scheme_test.go`.

- [ ] **Step 1: Failing tests.** A stateful fake persister:

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
  Table for `applyScheme(h, arg, p, session)` → resulting `effective()` + source, the fake's `saved`, and the error:
  - `light`, fake, session → light/`sourceSaved`, saved=light;
  - `light`, fake failing, session → error, holder UNCHANGED, saved unchanged;
  - `light`, nil persister, session → light/`sourceSession`;
  - `light`, nil persister, one-shot → `errNowhereToSave`, holder unchanged;
  - `auto` after a saved light with a detected dark underneath → dark/`sourceDetected`, saved="";
  - nil holder → `errNoScheme`.
  `describeScheme` table, each string exact: `light (saved)`, `dark (detected)`, `light (-scheme flag)`, `light (session only; not saved)`, `dark (default: the terminal has not reported its background)` (full-screen), `dark (default: detected only in a full-screen session)` (otherwise).
- [ ] **Step 2: Run — FAIL. Step 3: Implement.**

```go
var (
	errNoScheme      = errors.New("there is no colour scheme to change here")
	errNowhereToSave = errors.New("nowhere to save it: $XDG_CONFIG_HOME and $HOME are unset or not absolute")
)

// schemePersister is the durable half of /scheme. nil means there is nowhere
// to save (no config directory).
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
	st := h.Load()
	switch {
	case p == nil && !session:
		return st, errNowhereToSave
	case p == nil && arg.auto:
		st = st.withoutChoice()
	case p == nil:
		st = st.withChoice(arg.value, sourceSession)
	case arg.auto:
		if err := p.clear(); err != nil {
			return h.Load(), err
		}
		st = st.withoutChoice()
	default:
		if err := p.save(arg.value); err != nil {
			return h.Load(), err
		}
		st = st.withChoice(arg.value, sourceSaved)
	}
	h.Store(st)
	return st, nil
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
- [ ] **Step 4: Run — PASS. Step 5: Commit** `#70 M2: /scheme's transition persists first, and its report is always true`

### Task 10: The `/scheme` command, wired into all three contexts

**Files:** Create `cmd/define/scheme_cmd.go`; modify `cmd/define/command.go` (registry, `commandCtx`, `newCommandCtx`), `cmd/define/replraw.go` (~line 691), `cmd/define/repl.go` (~line 419); test `cmd/define/scheme_cmd_test.go`, `cmd/define/commandloop_test.go`.

- [ ] **Step 1: Failing tests — every loop shell that supplies wiring gets a test that drives THAT shell** (lessons "Pin a loop shell's wiring with a test that drives that loop shell"):
  1. **Editor** (`TestRawEditorSchemeRepaintsWhatIsOnScreen`): a console whose `view`/`stdout`/`stderr` is a real `newLiveScreen(&tty, 20, 60)` with `interval = -1`; `rig.deps.scheme = newSchemeHolder(schemeState{})`; `rig.deps.configDir` → a temp dir; `attachScheme` it; pre-write a tinted row (`l.WriteOutput(renderedOutput{text: "hola\n", rows: []rowPaint{{tinted: true}}})`); drive `runEditor` with `/scheme light⏎` then Ctrl-C. Assert: the last frame's `hola` row carries `languageLight` and no `languageDark`; `l.PaintedTranscript()` carries `languageLight`; `<tmp>/scheme` reads `light`; the output contains `scheme light (saved)`.
  2. **Editor, nowhere to save**: `configDir` nil → output `light (session only; not saved)`, repaint still light.
  3. **Editor, write error**: `configDir` → a path under a FILE (so `MkdirAll` fails) → stderr `define: /scheme:`, shade still dark, nothing claims saved.
  4. **Piped loop** (`replLines`): `/scheme light` → `scheme light (saved)`; a following lookup's tinted rows carry `languageLight`.
  5. **One-shot** via `run([]string{"/scheme", "light"}, …)`: file written, exit 0; `run([]string{"/scheme"})` with the file → `light (saved)`; with `configDir` nil → exit 2, stderr names `$XDG_CONFIG_HOME`.
  6. **Report in the editor** with nothing detected → `dark (default: the terminal has not reported its background)`.
- [ ] **Step 2: Run — FAIL. Step 3: Implement.**

```go
const schemeUsage = "With nothing, the colour scheme in use and where it came from. light or dark sets it and saves it for every session; auto forgets the saved choice, so define follows what the terminal reports. The scheme picks the shade of the language tint: dark grey on a dark background, light grey on a light one."

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
  Registry row (keep alphabetical position as the table does): `{name: "scheme", summary: "light or dark terminal background", args: "[light|dark|auto]", usage: schemeUsage, run: runScheme}`.
  `commandCtx` fields, with doc comments in the file's style: `scheme *schemeHolder`, `schemePersister schemePersister`, `session bool` ("there is a loop whose state a session-only choice can live in"), `fullScreen bool` ("this loop asks the terminal for its background — the raw editor, from M3"). `newCommandCtx`: `scheme: d.scheme, schemePersister: d.schemePersister()` (session false, fullScreen false: the one-shot). Editor (`replraw.go` beside `cc.setTimes`): `cc.session, cc.fullScreen = true, true` — the existing `draw()` after dispatch repaints from the holder, which is the whole of the recolour. Piped loop (`repl.go` beside `cc.setTimes`): `cc.session = true`.
- [ ] **Step 4: Run — PASS.** Then mutation checks (committed baseline): delete `cc.session, cc.fullScreen = true, true` in `replraw.go` → tests 2 and 6 redden; delete `live.attachScheme(d.scheme)` in `newConsole` — if no in-process test reddens, add one that builds the console through `newConsole` (as `rawterm_test.go:225` does) and asserts a tinted row paints in the holder's shade; restore.
- [ ] **Step 5: Commit** `#70 M2: /scheme switches, saves, and repaints what is already on screen`

### Task 11: Harness isolation and docs

**Files:** Modify `cmd/define/pty_conformance_test.go` (`startDefineBinary`), `cmd/define/pty_layout_conformance_test.go` (its own `exec.Command`), `cmd/define/README.md`, `atlas/define.md`, `cmd/define/main.go` (`-h` prose).

- [ ] **Step 1:** `startDefineBinary` ALWAYS sets `cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+t.TempDir())` followed by the caller's `env` (so a caller can still override); the layout test does the same. Why, in the comment: a developer's saved scheme would otherwise flip every "default" expectation.
- [ ] **Step 2:** Add a pty test `TestPTYSavedSchemeSurvivesARestart`: run 1 `/scheme light` then quit; run 2 in the same `XDG_CONFIG_HOME` looks up a word and shows `languageLight`; `/scheme auto`; run 3 shows `languageDark`.
- [ ] **Step 3: Docs.** Run `go test ./cmd/define -run 'TestDocs' -count=1` first — it names the README/atlas spans the new command must appear in (`TestDocsQuoteTheCommandList`, `TestDocsQuoteTheCommandUsage`). Update those spans; replace `README.md:344-345`'s `-language-tint=light` example with `-scheme light` and `-language-tint off`; add a "Light or dark" paragraph (precedence: flag, saved, detected, dark; where the file lives; `/scheme auto`); extend the `-h` prose with `/scheme`.
- [ ] **Step 4:** `go test ./cmd/define/... -count=1` and `go test -tags conformance ./cmd/define -run 'TestPTY' -count=1` — PASS (state which ran vs skipped).
- [ ] **Step 5: Commit** `#70 M2: the pty harness gets its own config directory; docs for /scheme`

### Task 12: M2 boundary

- [ ] Full suite, `go vet ./...`, `GOOS=linux go build ./...`.
- [ ] Atlas: the `/scheme` command under **Command mode**, the saved file under **The store** (a one-line "not the deck — the user's config directory" note), and the precedence.
- [ ] `sdlc milestone-close --issue 70 --milestone M2` — read the verdict before ticking.

---

## Chunk 3: M3 — detection

### Task 13: `parseBackgroundColour`

**Files:** Create `cmd/define/scheme_detect.go`; test `cmd/define/scheme_detect_test.go`.

- [ ] **Step 1: Failing test** (table): `rgb:ffff/ffff/ffff` → light; `rgb:0000/0000/0000` → dark; `rgb:1e1e/1e1e/1e1e` → dark; `rgb:fdf6/e3e3/e3e3`(Solarized-light-ish) → light; one-digit `rgb:f/f/f` → light; two-digit `rgb:80/80/80` → light (0.502, Rec. 601 on encoded values — the Neovim heuristic; this is the boundary case that distinguishes it from linear luminance); `rgb:7f/7f/7f` → dark; rejects: `rgba:ffff/ffff/ffff/ffff`, `#ffffff`, `rgb:fffff/0/0` (5 digits), `rgb:ff/ff`, `rgb:gg/00/00`, `rgb:`, ``.
- [ ] **Step 2: Run — FAIL. Step 3: Implement.**

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
- [ ] **Step 4: PASS. Step 5: Commit** `#70 M3: read a terminal's background colour as light or dark`

### Task 14: The decoder swallows the reply, then parses it

**Files:** Modify `cmd/define/key.go` (`KeyBackground` before the `numKeyKinds` sentinel; `Key.Background store.Scheme`; `decodeEscape` case `']'`); test `cmd/define/key_test.go`, `cmd/define/selection_input_test.go` (or wherever `readInput` is tested).

- [ ] **Step 1: Failing tests.**
  - `decodeKey("\x1b]11;rgb:ffff/ffff/ffff\x07")` → `KeyBackground`/light, consumed = len; same with ST `\x1b\\`.
  - Split across reads: every prefix of that reply → `used == 0` (waits); the whole → one key.
  - `rgba:…` and `#ffffff` payloads → ONE `KeyUnknown` consuming the whole reply (no runes leak).
  - Aborts decode exactly as today: `"\x1b]x"` → `KeyUnknown` (2 bytes) then `KeyRune 'x'`; `"\x1b]11;rgb\x03"` → `KeyUnknown` (2) … then `KeyInterrupt` in the same pass; DEL (`0x7f`) and `0x80` inside the payload abort; `ESC` followed by anything but `\` aborts; 64 bytes with no terminator abort.
  - Through `readInput` with the bytes as SEPARATE writes (an `io.Pipe`): `"\x1b]"` then `"\x03"` → the channel yields `KeyUnknown` then `KeyInterrupt` (the decoder really waited); `"\x1b]"` then `"a"` → `KeyUnknown`, `KeyRune 'a'`.
  - Extend `TestEveryEnabledInputModeIsDecoded`: iterate `terminalQueries` (Task 15) alongside `enabledModes`; the table gains `"background colour": {{"rgb with BEL", "\x1b]11;rgb:ffff/ffff/ffff\x07"}, {"rgb with ST", "\x1b]11;rgb:0/0/0\x1b\\"}, {"rgba", "\x1b]11;rgba:ffff/ffff/ffff/ffff\x1b\\"}}`, and every sample must decode to keys with no `KeyRune`. (Write this test in Task 15 if `terminalQueries` does not exist yet — it must FAIL CLOSED on a query with no row.)
- [ ] **Step 2: Run — FAIL. Step 3: Implement** in `key.go`:

```go
// oscBackgroundReply is the front of the terminal's answer to backgroundQuery.
const oscBackgroundReply = "\x1b]11;"

// maxOSCReply bounds what the decoder will hold while a reply arrives. A real
// one is about 25 bytes (rgba: about 30); past this it is not a reply.
const maxOSCReply = 64

// decodeOSC decodes ESC ] … (#70). It SWALLOWS only the reply to the question
// this program asks, and it is two steps so no reply FORMAT can leak as typing
// — in a sitting a leaked character is an answer:
//
//  1. swallow: ESC ] 11 ; then bytes in 0x20-0x7E up to BEL or ST (ESC \), at
//     most maxOSCReply bytes in all. Any other byte — Ctrl-C, Enter, DEL, 0x80+,
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
		case c == 0x07:
			return backgroundKey(buf[len(oscBackgroundReply):i], buf[:i+1]), i + 1
		case c == 0x1b:
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
  In `decodeEscape`, before the `'[', 'O'` case: `case ']': return decodeOSC(buf)`.
  Add `KeyBackground` to the `KeyKind` block (doc: "a terminal REPORT, not a keystroke: the answer to backgroundQuery. Never typing, never an answer, never cancels a gesture.").
- [ ] **Step 4: Run** the decoder tests and the existing fuzz seeds (`go test ./cmd/define -run 'Key|Decode|Fuzz' -count=1`) — PASS. Run each fuzz target for 30 s (`go test ./cmd/define -run '^$' -fuzz <Target> -fuzztime 30s` for each `Fuzz*` in `key_test.go`).
- [ ] **Step 5: Commit** `#70 M3: the key decoder reads a background report, bounded byte by byte`

### Task 15: Ask the question at raw-mode entry

**Files:** Modify `cmd/define/rawterm.go`, `cmd/define/replraw.go` (`newConsole` signature + `replRaw`), `cmd/define/play_loop.go:115`, `cmd/define/rawterm_test.go:225`; test `cmd/define/rawterm_test.go`, `cmd/define/key_test.go`.

- [ ] **Step 1: Failing tests.**
  - `newConsole(..., ask=true)` writes `backgroundQuery` to the session's `control` AFTER the mode enables (assert order on a recording writer); `ask=false` writes nothing of it.
  - `wantsBackground(opt)`: true for `{tty: true, color: true, tintOn: true}`; false with `tintOn` false, with `raw` true, with `tty` false.
  - `sittingInPlace` never writes the query (drive `/play` from the editor with a recording control; exactly ONE query in the stream).
- [ ] **Step 2: Run — FAIL. Step 3: Implement.**

```go
// backgroundQuery asks the terminal for its background colour (OSC 11, #70).
// The answer arrives as input, whenever it arrives; nothing waits for it.
const backgroundQuery = "\x1b]11;?\x1b\\"

// terminalQueries is every QUESTION this program asks a terminal. Not modes —
// there is nothing to tear down — but they share enabledModes' obligation: a
// terminal that answers owes the decoder a case, and
// TestEveryEnabledInputModeIsDecoded derives from both lists.
var terminalQueries = []struct{ name, query string }{
	{"background colour", backgroundQuery},
}

// ask writes a query to where the modes go. NEVER through a screen: scanEscape
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
  `newConsole(ctx, d, sess, stdout, newScreen, askBackground bool)`: after `sess.enterModes()`, `if askBackground { sess.ask(backgroundQuery) }`. Callers: `replRaw` → `wantsBackground(opt)`; `runPlay` (`play_loop.go:115`) → `wantsBackground(opt)`; `rawterm_test.go:225` → `false`.
- [ ] **Step 4: PASS. Step 5: Commit** `#70 M3: a full-screen session asks the terminal for its background, once`

### Task 16: Every consumer of the new key kind

**Files:** Modify `cmd/define/replraw.go` (`runEditor` key case), `cmd/define/play_loop.go` (`playSession` intercept, `sittingKeyHandling`), `cmd/define/selection_input.go` (`route`, `cancelPointerInput`, both full-channel sites); tests in `editorloop_test.go`/`commandloop_test.go`, `play_loop_test.go`, `selection_input_test.go`.

- [ ] **Step 1: Failing tests.**
  1. **Editor** (drive `runEditor`, live-screen console as in Task 10): a tinted row on screen, then `Key{Kind: KeyBackground, Background: store.SchemeLight}`, then Ctrl-C → the row repaints in `languageLight`; the prompt line contains no `rgb`; the history has no new entry. With `-scheme dark` in force (holder chosen by flag) the same key → the row stays `languageDark`.
  2. **Sitting** (drive `playSession` with a scripted key channel): the key between two answers → no answer recorded, the session's next question unchanged, the sitting's screen repaints in the new shade.
  3. **`/play` from the editor**: the key arrives during the sitting; after it ends, `/scheme` in the editor reports `light (detected)` and the editor's frame paints light; and a sitting started after a detected light starts light.
  4. **Pointer router**: during a drag (press + motion), route a `KeyBackground` → the selection is intact (the gesture still completes to a copy on release).
  5. **Full channel, both sites**: fill `out` (256) then feed a reply → no "input full" notice, `saturated` not set: a REAL key dropped next DOES post the notice. For the `select`'s `default:` arm, make the test deterministic with a `router` hook or a zero-capacity-at-send channel as the existing saturation tests do (mirror them — `grep -n 'input full' *_test.go`).
  6. `TestEveryKeyKindIsDecidedForASitting` reddens until the map has a row.
- [ ] **Step 2: Run — FAIL. Step 3: Implement.**
  - `runEditor`, first thing in `case k, open := <-keys:` after the `!open` check:

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
  - `playSession`: the same intercept where it intercepts `KeyClick`/paging before `toInput`, calling its `show()`; `sittingKeyHandling`: `KeyBackground: false` under a comment "a terminal report, intercepted before toInput — never an answer".
  - `selection_input.go`:

```go
// isTerminalReport is input the terminal SENT rather than the user typed (#70).
// It never cancels a gesture and it is never worth an "input full" notice.
func isTerminalReport(k KeyKind) bool { return k == KeyBackground }
```
    `route`: `} else if k.Kind != KeyUnknown && !isTerminalReport(k.Kind) {`; `cancelPointerInput`: `if k.Kind == KeyUnknown || isTerminalReport(k.Kind) { return }`; length-check site: `if isTerminalReport(k.Kind) { continue }` as the FIRST statement inside `if !isPointerKey(k.Kind) && len(out) == cap(out) {`; `default:` arm: `if isTerminalReport(k.Kind) { continue }` before the notice, so `saturated` is untouched. (`continue` inside `select` continues the inner `for` — confirm by the test, not by reading.)
- [ ] **Step 4: Run — PASS.** Mutation checks (committed baseline), each must redden its own test: remove the `runEditor` intercept (test 1: `rgb` text in the prompt, or no repaint); remove the `playSession` intercept (test 2); remove the `route` clause (test 4); remove each full-channel clause separately (test 5 — both sites, one at a time). Restore each.
- [ ] **Step 5: Commit** `#70 M3: a background report reaches every consumer as a report, never as typing`

### Task 17: Conformance, the test terminal readers, docs

**Files:** `cmd/define/pty_conformance_test.go`, `cmd/define/language_row_test.go` (`rowTestCells`), the `readFrame` helper (`grep -n 'func readFrame' cmd/define/*_test.go`), `cmd/define/README.md`, `atlas/define.md`.

- [ ] **Step 1:** Teach the test terminal readers (`readFrame`, `rowTestCells`, `unstyled` if it parses) to skip OSC sequences (`ESC ] … BEL|ST`) the way a terminal does. A unit test: a captured stream beginning with `backgroundQuery` reads the same frame as without it.
- [ ] **Step 2: `TestPTYBackgroundDetection`** (tag `darwin && conformance`), three subtests on a Spanish deck as `TestPTYLanguageTint` sets one up: wait until the master has received `\x1b]11;?`; reply `\x1b]11;rgb:ffff/ffff/ffff\x1b\\` (light) / `\x1b]11;rgb:0000/0000/0000\x07` (dark) / nothing (silent); look up `red`; assert the Spanish section's rows carry `languageLight` / `languageDark` / `languageDark`; `/scheme` prints `light (detected)` / `dark (detected)` / `dark (default: the terminal has not reported its background)`. A fourth: reply AFTER the entry is on screen → the entry repaints light.
- [ ] **Step 3:** Run `go test -tags conformance ./cmd/define -run 'TestPTY' -count=1`, then `CONFORMANCE_STRICT=1` with the same run — PASS; name what ran.
- [ ] **Step 4: Docs.** README "Light or dark": detection in full-screen sessions, one-shot uses flag/saved/dark, the late-reply-after-a-fast-quit limit. Atlas: **The screen** paragraph gains detection (query at mode entry via `rawSession.control`, reply as `KeyBackground`, the four consumers); **The line editor** gets the bounded OSC swallow; update `atlas/index.md` only if a new file is added.
- [ ] **Step 5: Commit** `#70 M3: conformance plays light, dark and silent terminals`

### Task 18: M3 boundary and close

- [ ] Full suite, `go vet ./...`, `GOOS=linux go build ./...`, conformance with `CONFORMANCE_STRICT=1`.
- [ ] **Manual live conformance** (the real external dependency; record results in the Log): in Terminal.app, iTerm2 and Ghostty, each in a light and a dark profile — `define`, look up `red` after `/lang es`, check the tint's shade and `/scheme`'s report; switch `/scheme light|dark|auto` and confirm the repaint; quit and confirm the transcript's shade. Note any terminal that answers `rgba:` or not at all.
- [ ] Walk every `## Done when` bullet and name the test (or manual check) that proves it.
- [ ] `sdlc milestone-close --issue 70 --milestone M3`, read the verdict; then `sdlc close --issue 70 --verified '<evidence>'`.
