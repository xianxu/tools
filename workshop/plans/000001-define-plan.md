# define — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `define <word>` prints the macOS-bundled New Oxford American Dictionary entry with Google's exact `/ˌsikəˈfan(t)ik/` notation, then plays the recorded Oxford pronunciation three times.

**Architecture:** A pure core (parse flat NOAD text → `Entry`; render `Entry` → string; derive audio URL candidates) surrounded by three thin, injected IO seams (`Dictionary` over CoreServices, `AudioSource` over the gstatic CDN, `Player` over `afplay`). Each seam ships a stateful fake as part of the deliverable, so the playback count and the CDN fallback order are asserted by tests rather than by ear (ARCH-PURE, ARCH-MOCK). The parser is best-effort over a schema-less string, so it is anchored by a **no-data-loss invariant** rather than by a claim of completeness.

**Tech Stack:** Go 1.26, cgo (`-framework CoreServices`), `net/http` + `httptest`, `afplay(1)`.

**Milestones:** M1 (definition pipeline) and M2 (pronunciation) are separate review boundaries — M1 delivers a useful `define` on its own, and the parser deserves fresh eyes before audio work lands on top of it.

---

## Chunk 1: Core concepts

### Package layout

Everything lives in `cmd/define/` as `package main`. Per `AGENTS.local.md`, `internal/` is earned on the **second** consumer, and there is no second consumer yet. Go runs colocated `_test.go` files in `package main` normally, so this costs no testability.

> When a second tool needs NOAD, promote `dict_darwin.go` + `dict_stub.go` to `internal/noad/` — that is the natural extraction point, and the `Dictionary` interface is already the seam.

`make build` is inherited from ariadne's `Makefile.workflow`: it walks `cmd/*/`, builds each `main.go` into `bin/<name>`, and needs no new target (ARCH-DRY — do not write a build rule).

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `Entry` | `cmd/define/parse.go` | new |
| `Block` | `cmd/define/parse.go` | new |
| `Sense` | `cmd/define/parse.go` | new |
| `Section` | `cmd/define/parse.go` | new |
| `ParseEntry` | `cmd/define/parse.go` | new |
| `Render` | `cmd/define/render.go` | new |
| `AudioCandidates` | `cmd/define/audiourl.go` | new |

- **Entry** — one parsed NOAD headword: `Headword`, `Syllables`, `Homograph`, `IPA`, `HeadExtra []string`, `Blocks []Block`, `Sections []Section`, and `Raw string` (the untouched input, retained so the invariant test and `--raw` can both reach it).
  - **Relationships:** 1:N with `Block`; 1:N with `Section`. `Entry` owns both.
  - **DRY rationale:** First occurrence of a pattern likely to recur — a second dictionary source would produce the same shape, so `Render` never learns where an entry came from.
  - **Future extensions:** A `Source` field when a non-NOAD provider is added (explicitly out of scope, see issue).

- **Block** — one part-of-speech run: `POS string`, `Senses []Sense`.
  - **Relationships:** N:1 with `Entry`; 1:N with `Sense`.

- **Sense** — one numbered sense or `•` sub-sense: `Number string`, `Sub bool`, `Gloss string`, `Examples []string`.
  - **Relationships:** N:1 with `Block`.

- **Section** — a trailing all-caps block: `Name` (`DERIVATIVES`, `ORIGIN`, `PHRASES`, `PHRASAL VERBS`, `USAGE`), `Text`.

- **ParseEntry(raw string) Entry** — the whole parser. Pure: string in, struct out, no IO.
  - **DRY rationale:** The single place that knows NOAD's flat-text conventions. Nothing else in the tool may inspect `Raw`.
  - **Future extensions:** Splitting a multi-headword response (`DCSCopyTextDefinition` returns one entry today; verified across the fixture corpus).

- **Render(e Entry, opt RenderOpts) string** — `Entry` + `RenderOpts{Color bool, Indent int}` → the printed block. Pure; no `os.Stdout`, no TTY probing (the caller decides `Color`).
  - **Relationships:** consumes `Entry`; produces a string the thin shell prints.
  - **Future extensions:** `RenderOpts.Width` for wrapping; a JSON renderer for `--json`.

- **AudioCandidates(word, locale string) []string** — ordered CDN URL candidates. Pure and offline: no request is made here.
  - **DRY rationale:** The path survey (see issue `## Log`) proved one URL is not enough — `defenestrate` 404s on the legacy `_1` path, `gaslighting` exists only on the 2022 path. Encoding that order once keeps the fallback policy out of the fetch loop.
  - **Future extensions:** More locales; a newer CDN generation prepends to the list.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Dictionary` | `cmd/define/dict.go` | new | interface (seam) |
| `noadDictionary` | `cmd/define/dict_darwin.go` | new | CoreServices `DCSCopyTextDefinition` |
| `unsupportedDictionary` | `cmd/define/dict_stub.go` | new | non-darwin stub |
| `fakeDictionary` | `cmd/define/dict_fake.go` | new | fixture corpus in `testdata/entries/` |
| `AudioSource` | `cmd/define/fetch.go` | new | interface (seam) |
| `httpAudioSource` | `cmd/define/fetch.go` | new | gstatic CDN over `net/http` |
| `fakeCDN` | `cmd/define/fetch_fake.go` | new | stateful `httptest` CDN |
| `Player` | `cmd/define/player.go` | new | interface (seam) |
| `afplayPlayer` | `cmd/define/player.go` | new | `afplay(1)` |
| `fakePlayer` | `cmd/define/player_fake.go` | new | stateful play recorder |

- **Dictionary** — `Lookup(word string) (string, error)`, returning raw NOAD text or `ErrNoEntry`.
  - **Injected into:** `run()` (the thin shell), never into `ParseEntry`. The parser only ever sees a string, which is what keeps its tests IO-free.
  - **Fake state model:** `fakeDictionary` is a `map[string]string` loaded from `testdata/entries/*.txt` — real captured output, not hand-written approximations. Unknown word → `ErrNoEntry`, exactly as the real one behaves for `rizz`.
  - **Live conformance:** `dict_conformance_test.go`, `//go:build darwin && conformance`, asserts the real `DCSCopyTextDefinition` still returns byte-identical text for every fixture. Run on demand (`go test -tags conformance ./...`); it is the drift detector for a macOS upgrade shipping a new NOAD.

- **AudioSource** — `Fetch(ctx, urls []string) (data []byte, from string, err error)`; walks candidates in order, returns the first 200.
  - **Injected into:** `run()`. `AudioCandidates` stays pure and offline so its ordering is unit-tested without a server.
  - **Fake state model:** `fakeCDN` is an `httptest.Server` plus a `map[string]bool` of which paths exist and a `[]string` recording **every path requested, in order**. That ordering record is the point: it proves the fallback walks candidates in the surveyed order and stops at the first hit, which a function-call mock would not catch.
  - **Live conformance:** `fetch_conformance_test.go`, `//go:build conformance`, asserts the real CDN still returns 200 for `sycophantic` and that `gaslighting` is still 2022-path-only — the two facts the candidate ordering rests on.

- **Player** — `Play(ctx context.Context, path string) error`.
  - **Injected into:** `run()`. The repeat loop lives in the shell, so `fakePlayer` can assert the count.
  - **Fake state model:** `fakePlayer` records `[]string` of played paths (stateful across calls) and can be armed to fail on the Nth call, so partial-playback handling is testable.
  - **Live conformance:** covered by the `afplay` presence check in `Play`; a missing `afplay` is a warning, not a crash.

---

## Chunk 2: M1 — Definition pipeline

### Task 1: Capture the fixture corpus

**Files:**
- Create: `cmd/define/testdata/capture.sh`
- Create: `cmd/define/testdata/entries/*.txt`

The corpus must include the structurally awkward entries, not just easy ones. Fixtures are committed (a captured trace is worthless if it cannot be reproduced or re-read).

- [ ] **Step 1: Write the capture script**

```bash
#!/usr/bin/env bash
# Capture real NOAD output for the parser fixture corpus.
# Re-run after a macOS upgrade; dict_conformance_test.go detects the drift.
# NOTE: must run OUTSIDE a sandbox — DCSCopyTextDefinition needs real access
# to /System/Library/AssetsV2 and silently returns nothing without it.
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p entries
words=(sycophantic quokka ephemeral defenestrate bank record run gaslighting set)
for w in "${words[@]}"; do
    go run ../ --raw "$w" > "entries/$w.txt" || echo "no entry: $w" >&2
done
wc -c entries/*.txt
```

- [ ] **Step 2: Ship `--raw` first (it is what the script needs)** — implement in Task 5; until then capture with the verified spike, whose exact cgo body is reproduced in Task 3.
- [ ] **Step 3: Run it, confirm `bank.txt` starts `bank 1 | baNGk |` and `record.txt` contains `rec·ordnoun`** — these are the two cases the parser is designed against.
- [ ] **Step 4: Commit** — `git add cmd/define/testdata && git commit -m "#1 M1: capture NOAD fixture corpus"`

### Task 2: The `Dictionary` seam and its fake

**Files:**
- Create: `cmd/define/dict.go`, `cmd/define/dict_fake.go`
- Test: `cmd/define/dict_fake_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestFakeDictionaryLoadsFixtures(t *testing.T) {
	d := newFakeDictionary(t)
	got, err := d.Lookup("sycophantic")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if !strings.Contains(got, "ˌsikəˈfan(t)ik") {
		t.Errorf("fixture missing IPA, got %q", got)
	}
	if _, err := d.Lookup("rizz"); !errors.Is(err, ErrNoEntry) {
		t.Errorf("want ErrNoEntry for rizz, got %v", err)
	}
}
```

- [ ] **Step 2: Run it, expect a compile failure** — `go test ./cmd/define/ -run FakeDictionary`
- [ ] **Step 3: Implement**

```go
// dict.go
var ErrNoEntry = errors.New("no dictionary entry")

// Dictionary resolves a word to a raw dictionary entry. The seam exists so the
// parser never touches CoreServices and the tests never touch the system
// dictionary (ARCH-PURE).
type Dictionary interface {
	Lookup(word string) (string, error)
}
```

`dict_fake.go` reads `testdata/entries/*.txt` into a map, keyed by filename stem.

- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#1 M1: Dictionary seam + fixture-backed fake`

### Task 3: The darwin implementation and the non-darwin stub

**Files:**
- Create: `cmd/define/dict_darwin.go`, `cmd/define/dict_stub.go`
- Test: `cmd/define/dict_conformance_test.go`

The cgo body below is **verified working** — do not redeclare `DCSCopyTextDefinition`; the SDK header already declares it and a duplicate `extern` is a compile error.

- [ ] **Step 1: Write `dict_darwin.go`**

```go
//go:build darwin

package main

/*
#cgo LDFLAGS: -framework CoreServices -framework CoreFoundation
#include <CoreServices/CoreServices.h>
#include <stdlib.h>

// DCSCopyTextDefinition is declared by the DictionaryServices SDK header
// included above. Only two functions are public; structured markup is not
// among them, which is why the caller parses flat text.
char *noad_lookup(const char *word) {
    CFStringRef s = CFStringCreateWithCString(NULL, word, kCFStringEncodingUTF8);
    if (!s) return NULL;
    CFRange r = CFRangeMake(0, CFStringGetLength(s));
    CFStringRef def = DCSCopyTextDefinition(NULL, s, r);
    CFRelease(s);
    if (!def) return NULL;
    CFIndex max = CFStringGetMaximumSizeForEncoding(CFStringGetLength(def), kCFStringEncodingUTF8) + 1;
    char *buf = malloc(max);
    if (buf && !CFStringGetCString(def, buf, max, kCFStringEncodingUTF8)) { free(buf); buf = NULL; }
    CFRelease(def);
    return buf;
}
*/
import "C"

import "unsafe"

type noadDictionary struct{}

func (noadDictionary) Lookup(word string) (string, error) {
	cw := C.CString(word)
	defer C.free(unsafe.Pointer(cw))
	res := C.noad_lookup(cw)
	if res == nil {
		return "", ErrNoEntry
	}
	defer C.free(unsafe.Pointer(res))
	return C.GoString(res), nil
}

func systemDictionary() Dictionary { return noadDictionary{} }
```

- [ ] **Step 2: Write `dict_stub.go`** — `//go:build !darwin`, a `systemDictionary()` returning a `Dictionary` whose `Lookup` reports `"the system dictionary is only available on macOS (GOOS=%s)"`.
- [ ] **Step 3: Verify both build paths**

```sh
go build ./... && go vet ./...
GOOS=linux CGO_ENABLED=0 go build ./...   # must stay green — the stub carries it
```

- [ ] **Step 4: Write the conformance test** (`//go:build darwin && conformance`) asserting live output is byte-identical to every fixture.
- [ ] **Step 5: Run `go test -tags conformance ./cmd/define/` outside the sandbox, expect PASS**
- [ ] **Step 6: Commit** — `#1 M1: NOAD lookup via CoreServices + non-darwin stub`

### Task 4: `ParseEntry` — TDD, hardest cases first

**Files:**
- Create: `cmd/define/parse.go`
- Test: `cmd/define/parse_test.go`

Parsing order: (1) split the header on the first `|…|` pair; (2) tokenize the head; (3) split trailing all-caps sections off the body; (4) split the remaining body into POS blocks; (5) split each block into senses.

Head tokenizing rules, derived from the corpus:
- token 0 → `Headword`
- a token of only digits → `Homograph` (`bank 1`)
- a token equal to `Headword` after removing `·` → `Syllables`
- a token that *starts with* `Headword` after removing `·` → `Syllables` + a glued leading POS (`rec·ordnoun` → `rec·ord` + `noun`)
- anything else → appended to `HeadExtra`, never dropped

- [ ] **Step 1: Write the failing tests — the awkward cases are the spec**

```go
func TestParseHeader(t *testing.T) {
	tests := []struct{ name, raw, word, syl, homo, ipa, leadPOS string }{
		{"simple", "sycophantic syc·o·phan·tic | ˌsikəˈfan(t)ik | adjective x",
			"sycophantic", "syc·o·phan·tic", "", "ˌsikəˈfan(t)ik", ""},
		{"homograph, no syllabification", "bank 1 | baNGk | noun 1 the land x",
			"bank", "", "1", "baNGk", ""},
		{"POS glued to syllabification", "record rec·ordnoun | ˈrekərd | 1 a thing x",
			"record", "rec·ord", "", "ˈrekərd", "noun"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := ParseEntry(tc.raw)
			if e.Headword != tc.word || e.Syllables != tc.syl ||
				e.Homograph != tc.homo || e.IPA != tc.ipa {
				t.Errorf("got %+v", e)
			}
		})
	}
}

func TestParseSensesAndSections(t *testing.T) {
	e := ParseEntry(mustFixture(t, "ephemeral"))
	if len(e.Blocks) != 2 { // adjective, noun
		t.Fatalf("want 2 blocks, got %d", len(e.Blocks))
	}
	if e.Blocks[0].POS != "adjective" {
		t.Errorf("POS = %q", e.Blocks[0].POS)
	}
	if got := e.Blocks[0].Senses[0].Examples; len(got) != 1 || got[0] != "fashions are ephemeral" {
		t.Errorf("examples = %q", got)
	}
	if names := sectionNames(e); !slices.Equal(names, []string{"DERIVATIVES", "ORIGIN"}) {
		t.Errorf("sections = %v", names)
	}
}
```

- [ ] **Step 2: Run, expect FAIL (undefined: ParseEntry)**
- [ ] **Step 3: Implement `ParseEntry` per the rules above.** Constants: `posWords` (noun, verb, adjective, adverb, pronoun, preposition, conjunction, interjection, exclamation, determiner, abbreviation, prefix, suffix, symbol, contraction, plural noun) and `sectionWords` (DERIVATIVES, ORIGIN, PHRASES, PHRASAL VERBS, USAGE). Match on whole tokens only. Always set `Raw`.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#1 M1: parse NOAD flat text into Entry`

### Task 5: `Render` and the no-data-loss invariant

**Files:**
- Create: `cmd/define/render.go`
- Test: `cmd/define/render_test.go`, `cmd/define/invariant_test.go`

The invariant is the load-bearing test: **every letter and digit of the raw entry appears, in order, in the rendered output.** Punctuation may be restructured (`:` becomes a line break, examples gain quotes); words may never vanish and may never be reordered. This is what makes a best-effort parser safe against entries nobody sampled — an unrecognized construct degrades to a paragraph instead of disappearing.

Consequence for `Render`: it must not change case, abbreviate, truncate, or reorder. Note that in a comment; it is a real constraint, not a style preference.

- [ ] **Step 1: Write the invariant test over the whole corpus**

```go
// alnum reduces a string to its letters and digits, so the comparison ignores
// punctuation the renderer is allowed to restructure.
func alnum(s string) []rune {
	var out []rune
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, r)
		}
	}
	return out
}

func TestRenderLosesNothing(t *testing.T) {
	for _, name := range fixtureNames(t) {
		t.Run(name, func(t *testing.T) {
			raw := mustFixture(t, name)
			out := Render(ParseEntry(raw), RenderOpts{Color: false})
			if missing, ok := isSubsequence(alnum(raw), alnum(out)); !ok {
				t.Errorf("renderer dropped content near %q", missing)
			}
		})
	}
}
```

`isSubsequence` returns the first raw rune that could not be matched, plus context — a bare `false` would make failures unreadable.

- [ ] **Step 2: Run, expect FAIL (undefined: Render)**
- [ ] **Step 3: Implement `Render`** — header line (`headword  syllables` + homograph), `/IPA/` on its own line, then per block: POS, senses indented by number, `•` sub-senses one level deeper, examples quoted one level deeper again; then sections. Colour via ANSI only when `opt.Color`.
- [ ] **Step 4: Run both render tests, expect PASS.** If the invariant fails, fix the *parser* to keep the unmatched text (as `HeadExtra` or a trailing paragraph) — never weaken the test.
- [ ] **Step 5: Commit** — `#1 M1: render Entry with no-data-loss invariant`

### Task 6: CLI wiring — M1 exit

**Files:**
- Create: `cmd/define/main.go`
- Test: `cmd/define/main_test.go`

- [ ] **Step 1: Write the failing test** — a `run(args, deps, stdout, stderr) int` returning `0` for `sycophantic`, and `1` with a stderr diagnostic for `rizz`, driven by `fakeDictionary`. `run` is the seam that makes `main()` a two-liner.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** Flags: `--raw` (print the unparsed entry — Task 1 depends on it), `--no-color`. Colour defaults on only when stdout is a TTY. `main()` = `os.Exit(run(os.Args[1:], realDeps(), os.Stdout, os.Stderr))`.
- [ ] **Step 4: Run tests; then `make build && ./bin/define sycophantic` and eyeball it**
- [ ] **Step 5: Commit, then close the milestone**

```sh
sdlc milestone-close --issue 1 --milestone M1
```

---

## Chunk 3: M2 — Pronunciation

### Task 7: `AudioCandidates`

**Files:**
- Create: `cmd/define/audiourl.go`
- Test: `cmd/define/audiourl_test.go`

Ordering comes from the measured survey in the issue `## Log`: the 2022 path first (strictly best coverage), then the legacy `oxford` path, with `_1` before `_2` at each generation.

- [ ] **Step 1: Write the failing test**

```go
func TestAudioCandidates(t *testing.T) {
	got := AudioCandidates("Sycophantic", "us") // case-insensitive
	want := []string{
		"https://ssl.gstatic.com/dictionary/static/pronunciation/2022-03-02/audio/sy/sycophantic_en_us_1.mp3",
		"https://ssl.gstatic.com/dictionary/static/pronunciation/2022-03-02/audio/sy/sycophantic_en_us_2.mp3",
		"https://ssl.gstatic.com/dictionary/static/sounds/oxford/sycophantic--_us_1.mp3",
		"https://ssl.gstatic.com/dictionary/static/sounds/oxford/sycophantic--_us_2.mp3",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %#v", got)
	}
}
```

Add a one-letter word case (`a`) — the two-letter shard prefix must not panic on a short word.

- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement** — lowercase, `url.PathEscape` the word, shard = first ≤2 letters.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#1 M2: derive CDN audio URL candidates`

### Task 8: `AudioSource` + the stateful fake CDN

**Files:**
- Create: `cmd/define/fetch.go`, `cmd/define/fetch_fake.go`
- Test: `cmd/define/fetch_test.go`, `cmd/define/fetch_conformance_test.go`

- [ ] **Step 1: Write the failing test — assert the walk order, not just the result**

```go
func TestFetchWalksCandidatesInOrder(t *testing.T) {
	cdn := newFakeCDN(t, map[string][]byte{"/c.mp3": []byte("ID3audio")})
	src := &httpAudioSource{client: cdn.Client()}
	data, from, err := src.Fetch(t.Context(), []string{cdn.URL+"/a.mp3", cdn.URL+"/b.mp3", cdn.URL+"/c.mp3"})
	if err != nil || string(data) != "ID3audio" || from != cdn.URL+"/c.mp3" {
		t.Fatalf("data=%q from=%q err=%v", data, from, err)
	}
	if want := []string{"/a.mp3", "/b.mp3", "/c.mp3"}; !slices.Equal(cdn.Requested(), want) {
		t.Errorf("walk order = %v, want %v", cdn.Requested(), want)
	}
}

func TestFetchAllMissing(t *testing.T) { /* → ErrNoAudio, all candidates attempted */ }
```

- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement `httpAudioSource.Fetch`** — context-aware, stops at the first 200, returns `ErrNoAudio` when every candidate 404s. `fakeCDN` wraps `httptest.NewServer`, records each requested path under a mutex, and exposes `Requested()`.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Write the conformance test** (`//go:build conformance`): real CDN returns 200 for `sycophantic`, and `gaslighting` is 200 on the 2022 path but 404 on `sounds/oxford` — the survey facts the ordering depends on.
- [ ] **Step 6: Commit** — `#1 M2: CDN audio fetch behind a seam + stateful fake`

### Task 9: `Player` and playing three times

**Files:**
- Create: `cmd/define/player.go`, `cmd/define/player_fake.go`
- Test: `cmd/define/player_test.go`

- [ ] **Step 1: Write the failing test — this is the "3 times" acceptance criterion**

```go
func TestPlaysRequestedNumberOfTimes(t *testing.T) {
	p := &fakePlayer{}
	if err := playN(t.Context(), p, "/tmp/x.mp3", 3); err != nil {
		t.Fatal(err)
	}
	if len(p.Played) != 3 {
		t.Errorf("played %d times, want 3", len(p.Played))
	}
}

func TestPlayNStopsOnError(t *testing.T) {
	p := &fakePlayer{FailOn: 2}
	err := playN(t.Context(), p, "/tmp/x.mp3", 3)
	if err == nil {
		t.Fatal("want error")
	}
	if len(p.Played) != 2 { // attempted 1 and 2, stopped before 3
		t.Errorf("played %d times, want 2", len(p.Played))
	}
}
```

- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** `playN` loops `n` times with a short gap (`gap = 250ms`, skipped after the final play) so the repeats are distinguishable by ear. `afplayPlayer.Play` writes the bytes to a temp file once (caller's job) and runs `exec.CommandContext(ctx, "afplay", path)`; a missing `afplay` returns a typed error the shell downgrades to a warning.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#1 M2: Player seam + playN with count assertion`

### Task 10: Wire audio into the CLI

**Files:**
- Modify: `cmd/define/main.go`
- Test: `cmd/define/main_test.go`

- [ ] **Step 1: Write the failing tests**
  - default run plays 3× via `fakePlayer`
  - `--no-audio` plays 0× and makes **zero** CDN requests (assert `fakeCDN.Requested()` is empty — no wasted fetch)
  - `--times 1` plays once
  - audio failure still prints the definition and exits **0**, with a warning on stderr (the definition is the primary deliverable; a missing recording is not a failed lookup)
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** Add `--no-audio`, `--times N` (default 3), `--locale us|gb`. Print the definition first, then `♫ playing 3×`, then play. Write the MP3 to `os.MkdirTemp` and clean up.
- [ ] **Step 4: Run tests; then the real end-to-end check**

```sh
make build
./bin/define sycophantic      # hear it 3×, see /ˌsikəˈfan(t)ik/
./bin/define --no-audio bank  # homograph renders sanely
./bin/define rizz; echo $?    # → clean diagnostic, 1
```

- [ ] **Step 5: Add a `make install` target** in `Makefile.local` symlinking `bin/*` into `~/.local/bin` (already on PATH), and correct `README.md`, which currently claims a `make install` that does not exist.
- [ ] **Step 6: Update `atlas/`** — a `atlas/define.md` sketch (the three seams, the parser's invariant, the CDN path survey) plus an `atlas/index.md` link.
- [ ] **Step 7: Commit and close**

```sh
sdlc close --issue 1 --verified '<evidence>'
```

---

## Risks

- **NOAD text has no schema.** Mitigated by the invariant test, not by claiming the grammar is complete. An unparsed construct degrades to a paragraph; it never disappears.
- **A macOS upgrade may reship NOAD.** The conformance test detects it; `testdata/capture.sh` re-captures.
- **The gstatic paths are undocumented.** Stable for a decade and through two path migrations, but unowned. The candidate list absorbs a third migration; the conformance test detects one.
- **`DCSCopyTextDefinition` returns nothing under a sandbox.** Not a code bug. Fixture-backed tests are unaffected; conformance tests must run unsandboxed.
