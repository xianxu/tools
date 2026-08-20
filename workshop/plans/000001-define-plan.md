# define — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `define <word>` prints the macOS-bundled New Oxford American Dictionary entry with Google's exact `/ˌsikəˈfan(t)ik/` notation, then plays the recorded Oxford pronunciation three times.

**Architecture:** A pure core (parse flat NOAD text → `Entry`; render `Entry` → string; derive audio URL candidates) surrounded by three thin, injected IO seams (`Dictionary` over CoreServices, `AudioSource` over the gstatic CDN, `Player` over `afplay`). Each seam ships a stateful fake as part of the deliverable, so the playback count and the CDN fallback order are asserted by tests rather than by ear (ARCH-PURE, ARCH-MOCK). The parser is best-effort over a schema-less string, so it is anchored by a **no-data-loss invariant** rather than by a claim of completeness.

**Tech Stack:** Go 1.26, cgo (`-framework CoreServices`), `net/http` + `httptest`, `afplay(1)`.

**Milestones:** M1 (definition pipeline) and M2 (pronunciation) are separate review boundaries — M1 delivers a useful `define` on its own, and the parser deserves fresh eyes before audio work lands on top of it.

**Testing note:** tests state a *strategy*, not a transcript. Each risky function gets one obligation line; the test body is written at the keyboard. The two things reproduced verbatim below are the ones that were expensive to learn: the cgo body (with its do-not-redeclare trap) and the header/pipe tables, which are specification rather than test data.

---

## Non-goals

These are limitations of the source, not deferred work. They are stated because
the issue's goal is "reproduce Google's dictionary panel," and NOAD-via-CoreServices
does not reach all of it:

- **Only one homograph is reachable.** `DCSCopyTextDefinition("bank")` returns
  `bank 1 | baNGk | noun 1 the land alongside…` and ends at `ORIGIN`. The
  financial-institution sense (`bank 2`) is simply not in the response —
  verified, zero occurrences. `define bank` will therefore show the riverbank
  and nothing else, where Google's panel shows both. **Mitigation:** render the
  homograph number, so the output says `bank 1` and the truncation is visible
  rather than silent.
- **The no-data-loss invariant guarantees fidelity, not completeness.** It
  asserts that nothing NOAD returned is dropped on the way to the screen. It
  says nothing about what NOAD declined to return. These are different claims
  and the tests must not be read as making the stronger one.
- **No second dictionary source.** NOAD-only was decided with the operator.
  Words absent from NOAD (`rizz`, `unalive`) exit non-zero. Their audio is
  missing too — the gaps correlate, since both trace back to Oxford.

---

## Chunk 1: Core concepts

### Package layout

Everything lives in `cmd/define/` as `package main`. Per `AGENTS.local.md`, `internal/` is earned on the **second** consumer, and there is no second consumer yet. Go runs colocated `_test.go` files in `package main` normally, so this costs no testability.

> When a second tool needs NOAD, promote `dict_darwin.go` + `dict_stub.go` to `internal/noad/` — that is the natural extraction point, and the `Dictionary` interface is already the seam.

`make build` is inherited from ariadne's `Makefile.workflow:735-757`: it walks `cmd/*/`, builds each `main.go` into `bin/<name>`, and needs no new target (ARCH-DRY — do not write a build rule).

### The two parsing rules that are actually hard

Both were derived by calling the real API, and both are the specification the
parser is written against.

**Rule A — the header.** Tokens before the first IPA span, after `Headword`:

| Case | Raw head | `Headword` | `Syllables` | `Homograph` | Glued POS |
|---|---|---|---|---|---|
| simple | `sycophantic syc·o·phan·tic` | `sycophantic` | `syc·o·phan·tic` | — | — |
| homograph, monosyllable | `bank 1` | `bank` | — | `1` | — |
| POS glued to syllabification | `record rec·ordnoun` | `record` | `rec·ord` | — | `noun` |

Token rules, applied in order: digits-only → `Homograph`; equal to `Headword`
after stripping `·` → `Syllables`; *starts with* `Headword` after stripping `·`
→ `Syllables` plus a glued POS suffix; anything else → `HeadExtra` (kept, never
dropped).

**The glued POS is not decoration — it is the first block's POS.** `record`'s
body begins `1 a thing constituting…` with no POS token of its own, because the
head consumed it. A parser that drops it yields a first `Block` with an empty
`POS`. So: glued POS + head IPA together open block 0.

**Rule B — `|` is overloaded.** In `record` the same delimiter does two jobs:

```
record rec·ordnoun | ˈrekərd | 1 a thing … : you should keep a written record
  | identification was made through dental records | a record of meter readings.
  … verb [with object] | rəˈkôrd | 1 set down …
```

`| ˈrekərd |` and `| rəˈkôrd |` delimit pronunciations; the middle pipes
separate examples. Discriminating on "contains no ASCII letters" does **not**
work — `ˈrekərd` and `baNGk` are mostly ASCII letters. The rule that does:

> A `|…|` span is a **pronunciation** iff every comma-separated part of its
> trimmed content is a single token (contains no internal space). Otherwise the
> pipes are **example separators**.

Verified against every pipe span in `sycophantic, record, bank, ephemeral, run, set`:

| span | verdict |
|---|---|
| `ˈrekərd`, `baNGk`, `rən`, `set`, `əˈfem(ə)rəl`, `ˌsikəˈfan(t)ik` | pronunciation |
| `ˌsikəˈfan(t)ək(ə)lē, -ˈfantik(ə)lē` (comma-separated variants) | pronunciation |
| `identification was made through dental records` | example |
| `[as modifier] : record profits` | example |
| `she ran the last few yards, breathing heavily` | example |

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `Entry` | `cmd/define/parse.go` | new |
| `Block` | `cmd/define/parse.go` | new |
| `Sense` | `cmd/define/parse.go` | new |
| `Section` | `cmd/define/parse.go` | new |
| `ParseEntry` | `cmd/define/parse.go` | new |
| `isPronunciation` | `cmd/define/parse.go` | new |
| `Render` | `cmd/define/render.go` | new |
| `AudioCandidates` | `cmd/define/audiourl.go` | new |

- **Entry** — one parsed NOAD headword: `Headword`, `Syllables`, `Homograph`, `IPA`, `HeadExtra []string`, `Blocks []Block`, `Sections []Section`, `Raw string` (untouched input, retained so the invariant test and `--raw` both reach it).
  - **Relationships:** 1:N with `Block`; 1:N with `Section`. `Entry` owns both.
  - **DRY rationale:** First occurrence of a pattern likely to recur — a second source would produce the same shape, so `Render` never learns where an entry came from.

- **Block** — one part-of-speech run: `POS string`, **`IPA string`**, `Senses []Sense`.
  - **The per-block `IPA` is required, not speculative.** `record`'s verb block carries `| rəˈkôrd |`, distinct from the head's `| ˈrekərd |`. Empty means "same as `Entry.IPA`"; `Render` falls back accordingly and prints a block-level pronunciation only when it differs.
  - **Relationships:** N:1 with `Entry`; 1:N with `Sense`.

- **Sense** — a numbered sense or `•` sub-sense: `Number string`, `Sub bool`, `Gloss string`, `Examples []string`.

- **Section** — a trailing all-caps block: `Name` (`DERIVATIVES`, `ORIGIN`, `PHRASES`, `PHRASAL VERBS`, `USAGE`), `Text`.

- **ParseEntry(raw string) Entry** — the whole parser: Rule A, then split trailing sections, then split blocks on POS tokens, then senses. Pure: string in, struct out.
  - **DRY rationale:** The single place that knows NOAD's conventions. Nothing else may inspect `Raw`.

- **isPronunciation(inner string) bool** — Rule B, factored out because both the header split and the body scan need it. Its table above *is* its test corpus.

- **Render(e Entry, opt RenderOpts) string** — `Entry` + `RenderOpts{Color bool}` → printed block. Pure; no `os.Stdout`, no TTY probing (caller decides `Color`). Renders `Homograph` (see Non-goals).
  - **Future extensions:** `RenderOpts.Width` for wrapping; a JSON renderer for `--json`.

- **AudioCandidates(word, locale string) []string** — ordered CDN URL candidates. Pure and offline.
  - **DRY rationale:** The survey (issue `## Log`) proved one URL is not enough — `defenestrate` 404s on the legacy `_1` path, `gaslighting` exists only on the 2022 path. Encoding that order once keeps fallback policy out of the fetch loop.

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
  - **Injected into:** `run()` (the thin shell), never into `ParseEntry` — which is what keeps the parser's tests IO-free.
  - **Fake state model:** `map[string]string` loaded from `testdata/entries/*.txt` — real captured output, not hand-written approximations. Unknown word → `ErrNoEntry`, exactly as the real one behaves for `rizz`.

- **AudioSource** — `Fetch(ctx, urls []string) (data []byte, from string, err error)`; walks candidates in order, returns the first 200.
  - **Injected into:** `run()`. `AudioCandidates` stays pure and offline so its ordering is unit-tested without a server.
  - **Fake state model:** `httptest.Server` + a `map[string]bool` of existing paths + a mutex-guarded `[]string` recording **every path requested, in order**. That ordering record is the point — it proves the walk stops at the first hit, which a function-call mock cannot show.

- **Player** — `Play(ctx, path string) error`.
  - **Injected into:** `run()`. The repeat loop lives in the shell so the fake can count.
  - **Fake state model:** records played paths across calls, and can be armed with `FailOn: N` so partial-playback handling is testable.

**Conformance cadence.** Both conformance checks (`//go:build conformance`) are deliberately **on-demand, not CI-scheduled** — they depend on the host's macOS dictionary assets and on reaching Google's CDN, neither of which belongs in `merge-check.yml` (it would make a green build depend on network weather and on a runner that has NOAD installed). The cadence is: run after a macOS upgrade, and whenever an audio or parse bug is reported. `testdata/capture.sh` re-captures the corpus in the same breath. Documented in `atlas/define.md` so the trigger is discoverable rather than folklore.

---

## Chunk 2: M1 — Definition pipeline

### Task 1: Capture the fixture corpus

**Files:**
- Create: `cmd/define/testdata/capture.py`, `cmd/define/testdata/capture.sh`
- Create: `cmd/define/testdata/entries/*.txt`

The corpus must include the structurally awkward entries. Fixtures are committed — a captured trace nobody can reproduce or re-read is worthless.

**The capture must not depend on the binary.** `capture.py` calls CoreServices directly through `ctypes`, so there is no Task 1 ⇄ Task 6 ordering knot: the corpus exists before any Go code does, which is what lets Task 2 onward be TDD at all.

- [ ] **Step 1: Write `capture.py`** — `ctypes` → `CoreFoundation` + `CoreServices`, calling `DCSCopyTextDefinition(NULL, word, {0, len})`, printing the returned text. (This is the same call the cgo path makes in Task 3; the duplication is deliberate and one-directional — the capture tool must not depend on the artifact it captures for.)
- [ ] **Step 2: Write `capture.sh` so a bad capture fails loudly**

```bash
#!/usr/bin/env bash
# Capture real NOAD output for the parser fixture corpus.
# MUST run OUTSIDE a sandbox — DCSCopyTextDefinition needs real access to
# /System/Library/AssetsV2 and silently returns nothing without it. That
# silence is exactly why this script fails hard on a short capture: a
# directory of zero-byte fixtures makes TestRenderLosesNothing vacuously green.
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p entries
words=(sycophantic quokka ephemeral defenestrate bank record run gaslighting set)
MIN_BYTES=40
for w in "${words[@]}"; do
    out="entries/$w.txt"
    if ! python3 capture.py "$w" > "$out.tmp"; then
        rm -f "$out.tmp"; echo "capture failed: $w" >&2; exit 1
    fi
    n=$(wc -c < "$out.tmp" | tr -d ' ')
    if [ "$n" -lt "$MIN_BYTES" ]; then
        rm -f "$out.tmp"
        echo "capture too short for '$w' ($n bytes) — sandboxed, or NOAD is absent." >&2
        exit 1
    fi
    mv "$out.tmp" "$out"
done
echo "captured ${#words[@]} entries:"; wc -c entries/*.txt
```

- [ ] **Step 3: Run it outside the sandbox.** Confirm `bank.txt` starts `bank 1 | baNGk |` and `record.txt` contains `rec·ordnoun` — the two cases the parser is designed against. Confirm no file is 0 bytes.
- [ ] **Step 4: Commit** — `#1 M1: capture NOAD fixture corpus`

### Task 2: The `Dictionary` seam and its fake

**Files:** create `cmd/define/dict.go`, `cmd/define/dict_fake.go`; test `cmd/define/dict_fake_test.go`

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

- [ ] **Step 1: Write the failing test.** Obligation: `fakeDictionary` loads every `testdata/entries/*.txt` keyed by filename stem, returns the real IPA for a known word, and returns `ErrNoEntry` for an absent one. **Assert the corpus is non-empty** — a fake that silently loads zero fixtures would make every later test vacuous.
- [ ] **Step 2: Run, expect a compile failure**
- [ ] **Step 3: Implement**
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#1 M1: Dictionary seam + fixture-backed fake`

### Task 3: The darwin implementation and the non-darwin stub

**Files:** create `cmd/define/dict_darwin.go`, `cmd/define/dict_stub.go`; test `cmd/define/dict_conformance_test.go`

The cgo body below is **verified working**. Do not redeclare `DCSCopyTextDefinition` — the SDK header already declares it and a duplicate `extern` is a compile error (that failure is how the header was confirmed public).

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

- [ ] **Step 2: Write `dict_stub.go`** — `//go:build !darwin`; `systemDictionary()` returns a `Dictionary` whose `Lookup` reports `"the system dictionary is only available on macOS (GOOS=%s)"`.
- [ ] **Step 3: Verify both build paths**

```sh
go build ./... && go vet ./...
GOOS=linux CGO_ENABLED=0 go build ./...   # must stay green — the stub carries it
```

- [ ] **Step 4: Write the conformance test** (`//go:build darwin && conformance`): live output is byte-identical to every fixture.
- [ ] **Step 5: Run `go test -tags conformance ./cmd/define/` outside the sandbox, expect PASS**
- [ ] **Step 6: Commit** — `#1 M1: NOAD lookup via CoreServices + non-darwin stub`

### Task 4: `ParseEntry`

**Files:** create `cmd/define/parse.go`; test `cmd/define/parse_test.go`

- [ ] **Step 1: Write the failing tests.** Obligations, hardest first:
  - `isPronunciation` reproduces the Rule B verdict table exactly — that table is the test corpus, table-driven.
  - Rule A's three header cases produce the fields in the Rule A table.
  - `record` parses to **two blocks**, and `Blocks[0].POS == "noun"` (from the glued suffix) while `Blocks[1].IPA == "rəˈkôrd"` — the finding that reshaped `Block`.
  - `record`'s sense 1 examples split into three on the interior pipes, and none of them is `[as modifier] : record profits` mis-read as a pronunciation.
  - `ephemeral` yields blocks `[adjective, noun]` and sections `[DERIVATIVES, ORIGIN]`.
- [ ] **Step 2: Run, expect FAIL (undefined: ParseEntry)**
- [ ] **Step 3: Implement.** Constants: `posWords` (noun, verb, adjective, adverb, pronoun, preposition, conjunction, interjection, exclamation, determiner, abbreviation, prefix, suffix, symbol, contraction, plural noun) and `sectionWords` (DERIVATIVES, ORIGIN, PHRASES, PHRASAL VERBS, USAGE), matched on whole tokens only. Always set `Raw`.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#1 M1: parse NOAD flat text into Entry`

### Task 5: `Render` and the no-data-loss invariant

**Files:** create `cmd/define/render.go`; test `cmd/define/render_test.go`, `cmd/define/invariant_test.go`

The invariant is the load-bearing test: **every letter and digit of the raw entry appears, in order, in the rendered output.** Punctuation may be restructured (`:` becomes a line break, examples gain quotes); words may never vanish and never reorder. It is what makes a best-effort parser safe against entries nobody sampled — an unrecognized construct degrades to a paragraph instead of disappearing.

It guarantees **fidelity, not completeness** (see Non-goals). Say so in the test file, so a future reader does not over-read it.

Consequence for `Render`: it must not change case, abbreviate, truncate, or reorder. That is a correctness constraint, not a style preference — note it in a comment.

- [ ] **Step 1: Write the invariant test** over the whole corpus: reduce raw and rendered to letters+digits, assert raw is a subsequence of rendered. On failure report the first unmatched rune **with surrounding context** — a bare `false` is unreadable. Guard that the corpus is non-empty.
- [ ] **Step 2: Run, expect FAIL (undefined: Render)**
- [ ] **Step 3: Implement `Render`** — header line (`headword  syllables`, plus homograph), `/IPA/` on its own line, then per block: POS, its own `/IPA/` when it differs from the entry's, senses indented by number, `•` sub-senses one level deeper, examples quoted one level deeper again; then sections. ANSI colour only when `opt.Color`.
- [ ] **Step 4: Run both, expect PASS.** If the invariant fails, fix the **parser** to retain the unmatched text (as `HeadExtra` or a trailing paragraph) — never weaken the test.
- [ ] **Step 5: Commit** — `#1 M1: render Entry with no-data-loss invariant`

### Task 6: CLI wiring — M1 exit

**Files:** create `cmd/define/main.go`; test `cmd/define/main_test.go`

- [ ] **Step 1: Write the failing test.** Obligation: `run(args, deps, stdout, stderr) int` returns `0` for a known word and `1` with a stderr diagnostic for an unknown one, driven by `fakeDictionary`. `run` is the seam that makes `main()` a two-liner.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** Flags: `--raw`, `--no-color`. Colour defaults on only when stdout is a TTY. `main()` = `os.Exit(run(os.Args[1:], realDeps(), os.Stdout, os.Stderr))`.
- [ ] **Step 4: Run tests; then `make build && ./bin/define sycophantic` and eyeball it**
- [ ] **Step 5: Commit, then `sdlc milestone-close --issue 1 --milestone M1`**

---

## Chunk 3: M2 — Pronunciation

### Task 7: `AudioCandidates`

**Files:** create `cmd/define/audiourl.go`; test `cmd/define/audiourl_test.go`

Ordering comes from the measured survey (issue `## Log`): the 2022 path first (strictly best coverage), then legacy `sounds/oxford`, `_1` before `_2` at each generation.

- [ ] **Step 1: Write the failing test.** Obligations: exact ordered list for `sycophantic`; input is lower-cased (`Sycophantic` → same list); a one-letter word (`a`) does not panic on the two-letter shard prefix.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement** — lowercase, `url.PathEscape`, shard = first ≤2 letters:

```
https://ssl.gstatic.com/dictionary/static/pronunciation/2022-03-02/audio/sy/sycophantic_en_us_1.mp3
https://ssl.gstatic.com/dictionary/static/pronunciation/2022-03-02/audio/sy/sycophantic_en_us_2.mp3
https://ssl.gstatic.com/dictionary/static/sounds/oxford/sycophantic--_us_1.mp3
https://ssl.gstatic.com/dictionary/static/sounds/oxford/sycophantic--_us_2.mp3
```

- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#1 M2: derive CDN audio URL candidates`

### Task 8: `AudioSource` + the stateful fake CDN

**Files:** create `cmd/define/fetch.go`, `cmd/define/fetch_fake.go`; test `cmd/define/fetch_test.go`, `cmd/define/fetch_conformance_test.go`

- [ ] **Step 1: Write the failing tests.** Obligations: with only the third candidate present, `Fetch` returns its bytes **and** `fakeCDN.Requested()` is exactly the first three paths in order (the walk order is the assertion, not just the result); when all 404, `ErrNoAudio` and every candidate attempted.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement** — context-aware, stop at first 200, `ErrNoAudio` when all miss. `fakeCDN` wraps `httptest.NewServer`, records each path under a mutex, exposes `Requested()`.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Write the conformance test** (`//go:build conformance`): the real CDN returns 200 for `sycophantic`, and `gaslighting` is 200 on the 2022 path but 404 on `sounds/oxford` — the two survey facts the ordering rests on.
- [ ] **Step 6: Commit** — `#1 M2: CDN audio fetch behind a seam + stateful fake`

### Task 9: `Player` and playing three times

**Files:** create `cmd/define/player.go`, `cmd/define/player_fake.go`; test `cmd/define/player_test.go`

- [ ] **Step 1: Write the failing tests** — this is the "3 times" acceptance criterion. Obligations: `playN(ctx, p, path, 3)` records exactly 3 plays; with `fakePlayer{FailOn: 2}` it returns an error and records exactly 2 (stops rather than pressing on).
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** `playN` loops `n` times with a 250 ms gap, skipped after the final play, so repeats are distinguishable by ear. `afplayPlayer.Play` runs `exec.CommandContext(ctx, "afplay", path)`; a missing `afplay` returns a typed error the shell downgrades to a warning.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#1 M2: Player seam + playN with count assertion`

### Task 10: Wire audio into the CLI

**Files:** modify `cmd/define/main.go`; test `cmd/define/main_test.go`

- [ ] **Step 1: Write the failing tests.** Obligations: default run plays 3×; `--no-audio` plays 0× **and makes zero CDN requests** (assert `Requested()` is empty — no wasted fetch); `--times 1` plays once; an audio failure still prints the definition and exits **0** with a stderr warning (the definition is the deliverable; a missing recording is not a failed lookup).
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** Add `--no-audio`, `--times N` (default 3), `--locale us|gb`. Print the definition, then `♫ playing 3×`, then play. Write the MP3 under `os.MkdirTemp`; clean up.
- [ ] **Step 4: Run tests, then the real end-to-end check**

```sh
make build
./bin/define sycophantic      # hear it 3×, see /ˌsikəˈfan(t)ik/
./bin/define --no-audio bank  # renders "bank 1" — homograph visible
./bin/define --no-audio record # two blocks, verb shows its own /rəˈkôrd/
./bin/define rizz; echo $?    # → clean diagnostic, 1
```

- [ ] **Step 5: Add a `make install` target** in `Makefile.local` symlinking `bin/*` into `~/.local/bin` (already on PATH), and correct `README.md`, which currently claims a `make install` that does not exist.
- [ ] **Step 6: Update `atlas/`** — `atlas/define.md` (the three seams, the two parsing rules, the CDN survey, the conformance trigger, the Non-goals) plus an `atlas/index.md` link.
- [ ] **Step 7: Commit, then `sdlc close --issue 1 --verified '<evidence>'`**

---

## Risks

- **NOAD text has no schema.** Mitigated by the invariant test, not by claiming the grammar is complete. Unparsed constructs degrade to paragraphs.
- **A macOS upgrade may reship NOAD.** The conformance test detects it; `capture.sh` re-captures.
- **The gstatic paths are undocumented.** Stable for a decade and through two migrations, but unowned. The candidate list absorbs a third; conformance detects one.
- **`DCSCopyTextDefinition` returns nothing under a sandbox.** Not a code bug, and the reason `capture.sh` fails hard on short output. Fixture-backed tests are unaffected; conformance and capture must run unsandboxed.

---

## Revisions

### 2026-08-20 — plan-quality gate round 1 (4 Important, 2 Minor)

Findings PQ-1..PQ-4 were raised by the `sdlc change-code` judge, which called
the live API rather than trusting the plan. All four factual claims were
independently re-verified before acting.

- **PQ-1 (`Block` had nowhere to put a per-block pronunciation).** Confirmed:
  `record`'s verb block carries `| rəˈkôrd |` against the head's `| ˈrekərd |`.
  Added `Block.IPA`, and made the head's glued POS explicitly open block 0 —
  previously it was extracted and then dropped, which would have yielded an
  empty first `POS`.
- **PQ-2 (`|` overloaded, no disambiguation rule).** Confirmed. Added Rule B as
  a named pure function `isPronunciation` with a verified verdict table.
  **The judge's suggested rule was wrong** — it proposed "IPA iff no ASCII
  letters", but `ˈrekərd` and `baNGk` are mostly ASCII letters. Re-measured
  across six entries and adopted a word-shape rule instead: a span is a
  pronunciation iff every comma-separated part is a single space-free token.
- **PQ-3 (`bank` returns homograph 1 only).** Confirmed — zero occurrences of
  the financial sense. Added a `## Non-goals` section stating the limitation,
  and made `Render` print the homograph number so the truncation is visible.
  Also distinguished fidelity from completeness where the invariant is defined.
- **PQ-4 (`capture.sh` could emit zero-byte fixtures).** Confirmed by
  inspection: `>` creates the file before the command runs and `||` swallowed
  the failure, so a sandboxed run would have left an empty corpus and a
  vacuously-green invariant test. Rewritten to capture to `.tmp`, enforce a
  minimum byte count, and `exit 1`. Added a non-empty-corpus assertion in the
  fake as a second line of defence.
- **PQ-4b (Task 1 ⇄ Task 6 circularity, raised inside PQ-4).** Confirmed — Step 2 misreferenced
  Task 5, and the "verified spike" lived only in a scratchpad, not the repo.
  Capture is now a self-contained `capture.py` (`ctypes` → CoreServices),
  depending on no Go code at all.
- **PQ-5 (inline test transcripts).** Accepted. Test bodies collapsed to
  obligation lines; kept the cgo body and the Rule A/Rule B tables, which are
  specification rather than test data.
- **PQ-6 (conformance cadence).** Accepted. Stated on-demand as a deliberate
  choice, with the reason (host NOAD + live network do not belong in
  `merge-check.yml`) and the trigger recorded in `atlas/define.md`.

### 2026-08-20 — M1 boundary review (4 Critical, 8 Important, 7 Minor) → REWORK

The review captured 170 live NOAD entries and ran M1's own invariant predicate
over them: **12 failed (7%)**. All four Criticals were reproduced locally before
fixing. The verdict was correct and the diff was reworked rather than argued.

**Core concepts drift, now reconciled (Chunk 1 above is updated in place):**

- `Entry` no longer carries `Headword` / `Syllables` / `Homograph` / `HeadPOS` /
  `HeadExtra` as parallel fields. It carries **`Head []HeadTok`** in source
  order, with those four as *accessors* derived from it. This is the root-cause
  fix for C1 and C3: NOAD has no fixed head field order (`present 1 pres·ent`
  vs `record rec·ordnoun` vs `read verb (past … read | red |)`), so a renderer
  emitting fields in a guessed order reorders every entry that disagrees. One
  representation, walked by `Render`, cannot drift (ARCH-DRY).
- `Block` gained `FromHead bool`, `Label string`, and `IPA` is now stored
  **unconditionally** (I4 — suppressing a block pronunciation equal to the
  entry's silently dropped it).
- New pure function `splitFirstToken` replaces `firstToken`/`trimFirstToken`,
  which re-derived the same boundary and **disagreed on whether a newline
  counts** — that disagreement *was* C2, dropping the leading "A" from every
  entry with no pronunciation span (`iPhone`, `iPad`, `MacBook`). Two helpers
  independently deriving one rule is the ARCH-DRY failure mode, not a style nit.
- `findPronunciation` (was `splitLeadingPronunciation`) now tracks **paren
  depth**, so `read`'s parenthesised inflected-form pronunciation is not
  mistaken for the entry's.
- `posAt`'s trailing boundary is whitespace-or-EOS; allowing `]`/`)` made
  `(banked as adjective)` open a phantom top-level block (C4).
- `parseSenses` accepts a numbered split only when it opens the block or
  continues the sequence — bare numerals in prose ("the 200 meters") were
  becoming sense numbers (I5).

**Test surface, widened.** The invariant was *corpus-scoped* while its own
documentation claimed it made the parser "safe against entries nobody sampled"
(I1 — the ARCH-PURPOSE finding, and the fair one). It is now checked at three
widths: the 21-fixture corpus, a corpus-seeded `FuzzRenderLosesNothing`, and
`TestRenderLosesNothingOverLiveEntries`, which walks a stride sample of
`/usr/share/dict/words` through the real dictionary. Plus `render_test.go`
(never created in M1, I6) with structural goldens for block shape and sense
numbering — the classes the alnum property is *blind* to, because they preserve
letter order.

**Corpus (I2)** grew from 9 to 21 entries, adding every shape that shipped a bug.

**Also noted by the review and applied:** `go mod tidy`; the cgo bridge now
distinguishes "no entry" from a CoreFoundation failure; the fake moved into a
`_test.go` file so it no longer links into the shipped binary; `-h` exits 0;
README documents the flags and no longer claims a `make install` that does not
exist; committed build-artifact blobs dropped from the branch history.

**Deferred to M2 with reason:** nothing. All Critical and Important findings are
addressed in this milestone.
