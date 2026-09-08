# Cached audio and deck-word marks — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A recording is fetched once and kept; and every word from the learner's
deck is clickable wherever it is written, and coloured wherever colour still
means something.

**Architecture:** Two independent halves. **M1** makes the memo part of the
SEAM rather than a decorator anyone must remember to apply: `deps.audio` becomes
a `*audioSeam` that is cached by construction, so there is no unwrapped source to
hold and the compiler enumerates the construction sites — and gives it a home on disk under the working directory,
which makes it a `perWordDir` and therefore something `Forget` takes with a
word. **M2** collapses two span producers into one: `highlightSpans` already
finds deck words in text with longest-phrase-wins, and `deckSpans` puts that walk
into VISIBLE coordinates so the same span can be both coloured and clicked.
Colour then becomes a property of the surface — on in prose, off where the text
is the deck itself.

**Tech Stack:** Go 1.24. No new dependencies. `cmd/define` (main),
`cmd/define/store` (YAML), `cmd/define/play` (pure, imports nothing).

---

## Core concepts

### Three layers, and where each one stops

Stated before the entity table because the table's rows land in three different
places, and flattening them is the likeliest way this milestone goes wrong. The
work is mostly INSERTING the middle layer and re-pointing what already exists
through it — layers 1 and 3-as-colour are built and wired point-to-point today,
which is exactly why colour reaches the entry and clicks reach the headword and
neither reaches a form's own text.

**Layer 1 — the matcher.** `wordRuns` + `highlightSpans`, both existing and both
unchanged by this plan. Takes text and a phrase set, returns which BYTE RANGES
are known, longest phrase winning. No ANSI, no coordinates, no `Region`, no
terminal.

- **Keep it that way.** This is the only layer that would travel to another
  program, and every terminal concern pushed down into it is a concern that has
  to be unpicked later. If Task 4 finds itself editing `highlightSpans`, that is
  the signal something belongs in layer 2 instead.
- **Do NOT extract it into a shared package here.** No second consumer outside
  this binary exists yet, and an interface guessed for an imaginary caller is
  guessed wrong. The cross-program version of this idea is the VOCABULARY AS
  DATA — a directory another program is handed and walks itself — which is the
  operator's own framing, works whatever language that program is written in,
  and needs no code shared at all. Post-MVP, and unblocked by this milestone
  rather than part of it.

**Layer 2 — the locator.** `deckSpans`, the one genuinely new thing. Answers
"where is that on screen": visible cells, line indices, and skipping escape
sequences. Terminal-shaped by nature and it will not travel — but it works on
ANY string this program writes, which is the reach that matters here.

**Layer 3 — the consumers.** Colour and clicks, each a loop over layer 2's
output. Both become thin, and that is the test of whether the seam is right: a
consumer that needs to re-tokenise, re-scan, or re-decide what a word is has been
handed the wrong thing.

- **Future extensions:** the third consumer is `#13`, marking which deck words a
  learner actually used in a written sentence. If that is a loop, this milestone
  succeeded.

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `deckSpan` | `cmd/define/deckwords.go` | new |
| `deckSpans` | `cmd/define/deckwords.go` | new |
| `mergeRegions` | `cmd/define/deckwords.go` | new |
| `wordRegions` | `cmd/define/deckwords.go` | new |
| `RegionWord` | `cmd/define/render.go` | new |
| `surface` | `cmd/define/deckwords.go` | new |
| `AudioKey` | `cmd/define/store/audio.go` | new |
| `AudioRecord` | `cmd/define/store/audio.go` | new |
| `perWordDir` | `cmd/define/store/yaml.go` | modified |

- **`deckSpan`** — one deck word located in RENDERED text: `{Line, Col, Width,
  Word, Text}`. Coordinates are display cells and line indices, the units
  `Region` already speaks in.
  - **Relationships:** N per rendered string; 1:1 with the `span{known:true}`
    that `highlightSpans` already emits. It adds coordinates and nothing else.
  - **DRY rationale:** this is the whole point of the milestone. Colour and
    clicks are today found by two different walks (`highlightSpans` for colour,
    `regionsIn` for clicks) that cover different surfaces and can disagree about
    where a word is. One walk, two consumers.
  - **Future extensions:** a third consumer — `#13`'s written-sentence form wants
    to mark which deck words a learner actually used.

- **`deckSpans(rendered string, v Vocabulary) []deckSpan`** — the walk.
  - **ESCAPE-AWARE BY CONSTRUCTION, without teaching `highlightSpans` about
    ANSI.** Per line it takes `plain, cols := visibleIndex(line)`, runs the
    EXISTING `highlightSpans` over `plain`, and maps each known span's byte
    offset through `cols`. That is `findVisible`'s trick generalised from one
    needle to the whole vocabulary, and it matters because the reveal contains an
    entry `Render` has already coloured: a walk over raw bytes would match inside
    an escape sequence and produce a region that underlines a `[1;32m`.
  - **Relationships:** consumes `Vocabulary` (already the deck, via
    `storeVocabulary.Load`). Produces spans for `wordRegions` and for the colour
    pass.

- **`wordRegions(rendered string, v Vocabulary) []Region`** — `deckSpans` mapped
  to `Region{Kind: RegionWord}`.
  - **DRY rationale:** keeps `Region` construction in one place so the `#12`
    BR-14 invariant (a region covers the text it claims) is provable once.

- **`RegionWord`** — a word from the learner's deck, wherever it appears.
  - **Why not reuse `RegionHeadword`.** The headword region carries the entry's
    LOOKUP KEY, which can differ from the text under it: `define jalapeno`
    renders `jalapeño`, and `Region.Word` is what gets played. A deck word in
    prose is matched by its own text and played by its own text. Same action,
    different provenance — and the registry's own doc says a third consumer is a
    row rather than a new feature.
  - **Future extensions:** `String()`, `identifier()` and the atlas guard all
    pick it up from `numRegionKinds` with no edit.

- **`surface`** — what a piece of text IS, which decides whether colour applies.
  Two values to start: `surfaceProse` and `surfaceDeck`.
  - **THE OPERATOR'S RULE, AS A PROPERTY.** The ask was "no colour in the cloze,
    because everything there is new words and colour would clog things up". Read
    narrowly that is a flag on one form. Read as a property it is *colour marks a
    deck word inside prose, where finding one is a discovery; where the text IS
    the deck, colour marks everything and distinguishes nothing* — which also
    answers the board, whose every cell is a deck word, and predicts the answer
    for a surface nobody has written yet.
  - **DERIVED FROM THE FORM, NOT THE WRITE SITE** (PQ-4). The first draft said
    "a property of a WRITE SITE" and then asked Task 6 to pass `surfaceDeck` for
    a cloze — but there is ONE prompt write site (`play_loop.go:226`) serving
    both `Choice` and `Cloze`, so it cannot tell them apart. `surfaceOf(q)`
    switches on `q.Form()` — the NAME, a string, never a type switch on a
    concrete form, which `#6`'s Done-when forbids outright.
  - **`play` still never sees it.** The classification lives in main; `play` stays
    pure and holds no presentation. What main gains is a total function over the
    form names, and its extent is `docSyncForms` — so
    `TestEveryFormIsEnrolled`'s AST walk already guarantees a new form cannot
    arrive unclassified.

- **`AudioKey`** — the identity of one cached recording, and **it is the SEAM's
  key, not the word** (PQ-1).
  - **THE FINDING THAT RESHAPED M1.** The first draft keyed on
    `audio/<lang>/<slug>`, justified by "AudioCandidates keys on the voice's Lang
    and Locale". That is false twice: `audioDir` would scope on the STORE's
    language, not the voice's, and locale appears nowhere in the path. The seam's
    real key is `strings.Join(urls, "\n")` (`fetch.go:109`) over a list
    `utterance.Candidates()` builds from a SOURCE voice, a SESSION voice and
    `u.Spellings`. So `-pron en red` and a plain `red` in one Spanish directory,
    and `-locale gb` against `-locale us`, would all have mapped to one file and
    served the wrong recording — a cache that lies, which is worse than no cache.
  - **So the key is a digest of the candidate list**, and the word is what the
    file is FILED under rather than what identifies it:
    `audio/<lang>/<slug>--<digest>.yaml` (the record) beside
    `audio/<lang>/<slug>--<digest>.mp3` (the bytes). Two axes, both served: the
    digest is the seam's identity, and the `<slug>--` prefix is what `Forget`
    globs.
  - **Deriving the digest from the candidate list rather than from the voice** is
    the same rule `spokeSource` states one file over: "by MEMBERSHIP in the list
    actually built, never by parsing the URL". A key rebuilt from `voice` fields
    would be a second, driftable statement of what identifies a recording.

- **`AudioRecord`** — what is stored beside the bytes: `from`, the day, and
  whether the verdict was "no recording".
  - **`from` IS LOAD-BEARING AND THE FIRST DRAFT DROPPED IT** (PQ-1). `Fetch`
    returns `([]byte, from string, error)`, and `from` is not decoration:
    `utterance.spokeSource(from)` decides it by MEMBERSHIP in
    `sourceCandidates()`, and `reportVoice` (`main.go:1014`) prints a RECORD from
    it — one that "survives on a pipe and cannot be taken back". A cache hit that
    cannot say which URL answered makes that record silent or false, so the
    cached artifact is bytes AND provenance or it is not a cache of this seam.
  - **One shape for a hit and a miss.** A record always exists; a hit's names its
    `from` and has a blob beside it, a miss's has neither. That is one artifact to
    read, one to write and one for `Forget` to remove, rather than a `.mp3`/`.none`
    pair whose two shapes drift apart.
  - **Why the verdict is dated.** A miss cached forever is a word that can never
    start working, and the CDN gains recordings over time. A verdict carries the
    day it was reached and is re-asked after `audioVerdictTTL` (30 days). The
    in-memory `misses` set needs no TTL because it dies with the process; the
    durable one does, and that difference is the reason this is a new type rather
    than the old map serialised.

- **`mergeRegions(into, add []Region) []Region`** — the precedence rule for two
  region producers writing to one line (PQ-2).
  - **`markClickable` HAS AN UNWRITTEN PRECONDITION, and M2 is the first thing
    that can violate it.** It sorts by `Col` and walks ONE `next` cursor left to
    right (`screen.go:359`), so two regions on the same column mean the second can
    never match — and **every later region on that line silently loses its
    underline**. `RegionAt` (`screen.go:192`) compounds it: it returns the first
    region in APPEND ORDER that contains the column, not the narrowest. Neither
    was wrong before, because one producer never overlapped itself.
  - **Precedence: the existing region wins.** A `RegionHeadword` carries the
    entry's LOOKUP KEY, which is more specific than a text match — `define
    jalapeno` renders `jalapeño`, and the headword region plays the right thing
    where a text-matched `RegionWord` would play what is on screen.
  - **The precondition becomes a checked property, not a comment:** every line's
    regions are disjoint and ascending by column. That is the class — it holds for
    the third producer too, and `#12` BR-14's guard is its sibling (a region must
    cover the text it claims; now also: no two may cover the same cell).

- **`perWordDir`** *(modified)* — gains the ability to remove more than
  `<slug>.yaml`. Audio is the first per-word directory where a word owns SEVERAL
  files, because one word has a recording per voice — see `audioKey` for the
  shape, which this bullet deliberately does not restate. `Forget` globs the
  word's prefix and removes the set.

**`RenderOpts` is UNCHANGED, and that is a decision rather than an omission.**
`Render` keeps colouring the entry itself, because it colours with a per-region
BASE style (amber part-of-speech labels, the `p.ex` example style) and ANSI does
not nest — a flat pass at the write door cannot reproduce those resume rules. So
the write door colours the text OUTSIDE the render instead, locating the boundary
rather than assuming it. The first draft listed this type as `modified`, which
`TestPlanTableStatusMatchesTheChangeWindow` correctly called work that did not
happen.

**Test surface.** Every entity above is PURE and gets a colocated unit test that
runs with no IO and no fake: `deckwords_test.go`, `store/audio_test.go`. The one
exception is `perWordDir`, whose test is the existing
`TestPerWordDirsCoverEveryRuntimeDir` — a guard, not a new file.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `diskAudioCache` | `cmd/define/audiodisk.go` | new | the working directory |
| `audioSeam` | `cmd/define/fetch.go` | new | an `AudioSource`, with its memo |
| `audioDir` | `cmd/define/store/yaml.go` | new | the filesystem |
| `writeWords` | `cmd/define/main.go` | new | stdout + the click map |

- **`diskAudioCache`** — an `AudioSource` decorator that reads and writes
  recordings under the store's directory.
  - **Injected into:** nothing pure; it is the outermost decorator on
    `deps.audio`, so every existing consumer is unchanged.
  - **A DECORATOR, LIKE ITS SIBLING.** The memo's own doc comment records
    why the memo is a decorator rather than a map inside the REPL loop: "the
    existing `fakeCDN` request recorder is the assertion that a replay costs no
    second request — no bespoke test scaffolding". That argument carries over
    exactly, and it is the reason this half needs no new fake: `fakeCDN` plus a
    `t.TempDir()` store is the whole rig.
  - **Future extensions:** eviction. Deliberately NOT built — a recording is
    10–30KB and a 1000-word deck is roughly 30MB, which is smaller than the
    dictionary it sits beside. Recorded as a number rather than omitted, so the
    decision is reviewable.

- **`audioSeam`** — the source and its memo as one value, replacing the
  decorator this issue removed.
  - **THE BUG WAS THAT WRAPPING WAS REMEMBERED.** `replLines` and `runEditor`
    each wrapped; `runPlay` did not, so the one loop that replays the same
    handful of words was the one with no cache. Fixing `runPlay` fixes the site.
    Making the source impossible to obtain unwrapped fixes the class — and four
    plan-gate rounds established that no predicate over functions can do it.
  - **Injected into:** `deps.audio`, whose TYPE is now `*audioSeam`. A pointer,
    so every by-value copy of `deps` shares one memo.

- **`audioDir`** — the per-language audio directory, appended to `RuntimeDirs`
  AT THE TAIL (the file's own positional-index rule).

- **`writeWords`** — the one door for text that reaches the sitting's scrollback
  with the deck walk applied. It takes the vocabulary and the surface.
  - **NEW, beside `writeRendered` rather than replacing it.** The plain door
    still serves callers with nothing to mark, and keeping them separate means
    the walk is opted into at a site rather than imposed on every write — which
    is what let the lookup path keep `Render`'s own per-region colouring
    untouched.

**Test surface for integration points.** `fakeCDN` (existing, wire-level) plus a
real `t.TempDir()` store. No function-call mocks: a request recorder is what
proves "no second fetch", and a real directory is what proves the bytes came back
after the process ended.

### Explicit non-goal: the board's click still MARKS

A click on a board cell marks that word — it is the board's ANSWER gesture, and
the README documents it as the exception ("board: mark that word. Anywhere else
in a sitting, a click plays the word"). Clicking to hear a word and clicking to
answer about it cannot both own the same gesture on the same surface, and the
answer is the one the learner came for.

This is the one place "click to pronounce in all places" does not hold, and it is
called out here rather than discovered later.

---

## Chunk 1: M1 — the cache is structural and durable

### Task 1: No loop can reach the network twice for the same word

**Files:**
- Modify: `cmd/define/play_loop.go` (`runPlay` — the missing wrap)
- Test: `cmd/define/fetch_test.go`, `cmd/define/play_loop_test.go`

**THREE ROUNDS REVERSED THIS TASK, and every reversal was the same mistake: a
statement about WHERE THE WRAP LIVES that was never checked against the code.**

| round | the claim | what the tree says |
|---|---|---|
| 1 (PQ-3) | move the wrap to `realDeps()`; tests traverse it | no test calls `realDeps()` — 21 `deps{...}` literals |
| 2 (PQ-9) | loops are `run()`'s deps-taking callees | that set is seven one-shot commands and neither wrap site |
| 3 (PQ-9) | the wraps live in `replLines` and `replRaw` | `replraw.go:264` is inside **`runEditor`** (`replraw.go:253`), not `replRaw` |

Three for three. The third is the one that shows why this cannot be fixed by
being more careful: `runEditor`'s signature is `(ctx, keys <-chan Key,
interrupts, d deps, opt options, con console) int` — **no `io.Reader`, no
`io.Writer`** — so the round-2 predicate ("takes a `deps`, an `io.Reader` and
writers") excludes it outright, while roughly 55 tests drive it directly
(`askroute_test.go:40`, `editorloop_test.go:252`, `commandloop_test.go:87`).
Deleting its wrap would leave the guard green and every raw-loop test on an
uncached source: `#2` I-1 again, which PQ-3 already rejected once.

**FOUR OF FOUR prose statements of the wrap set have been wrong** — `realDeps`,
`run()`'s callees, `replRaw`, and then "~8 functions" — which the gate measured
and found three times larger. The fourth is the decisive one: it was made while
ARGUING that the predicate should be generous, and still got the size wrong. Step
4 would have written the wrap line into most of
those functions, which never read `d.audio` at all (`vocabularyFor`, `clozeAsk`, `todaysQuestions`,
`newCommandCtx`, `playRegion`, `submitLine`, …), contradicting this plan's own
Architecture line — *one construction site* — and leaving a guard that checks a
TOKEN APPEARS IN A BODY rather than that a caller's source is cached.

**So M1 takes the third shape, and the plan states it once: `d.audio` becomes
unreachable except through a seam that is already cached.**

```go
// audioSeam is the audio source AND its memo, as one value.
//
// A POINTER, so every by-value copy of deps shares one memo — which is what the
// decorator could never guarantee, because deps is copied at every call.
//
// This is the shape four rounds of the plan gate converged on, and the reason is
// that the previous shape asked a question with no derivable answer. "Which
// functions must remember to wrap the source" was answered wrongly four times:
// realDeps (no test calls it), run()'s callees (seven one-shot commands),
// replRaw (the wrap is in runEditor), and "~8 functions" (measured: three times
// that). Every answer was a statement about the code the code did not support.
//
// A field of this type does not ask the question. There is no unwrapped source
// to hold, so there is no wrap to forget, no predicate to derive, and no guard
// to keep honest — the COMPILER enumerates the construction sites, which is the
// only enumeration in this program that cannot drift. #2's I-1 lesson is
// satisfied absolutely rather than by convention: a test cannot build a deps
// whose audio differs in kind from production's.
type audioSeam struct {
	inner AudioSource
	mu    sync.Mutex
	hits  map[string]cachedAudio
	misses map[string]missRecord
}
```

- [x] **Step 1: Change the field's TYPE, and let the compiler find the sites**

`deps.audio` becomes `*audioSeam`. Build the package and the tests: every site
that assigns an `AudioSource` is now a compile error, and that list is the
construction enumeration — complete, by definition, and free.

Run: `go build ./... && go vet ./...`
Expected: errors at `realDeps` and at every test literal. **Do not count them in
prose.** Paste the compiler's list into the Log; a number written by hand here
would be the fifth wrong statement of this set.

- [x] **Step 2: Give tests one door.** `newAudioSeam(src AudioSource) *audioSeam`,
      and a test helper for the silent source that `noAudioSource{}` served.
      Mechanical, and the churn is the price of the class: it is paid once, by the
      compiler, instead of every time someone adds an entry point.

- [x] **Step 3: Delete the wraps.** `repl.go:257` and `replraw.go:264` go, along
      with the decorator itself — its memo logic moves into
      `audioSeam` unchanged, including the hits/misses split and the
      `ErrNoAudio`-vs-`ErrFetchFailed` taxonomy, which is the single source of
      what "permanent" means and is not re-decided here.

- [x] **Step 4: The property, now stated where it is TRUE by construction**

```go
// There is no unwrapped source to hold. This asserts the shape rather than a
// habit: deps.audio is a *audioSeam, so a caller cannot reach the network twice
// for one key however it obtained its deps.
func TestASecondFetchOfOneKeyDoesNotReachTheSource(t *testing.T)
```

- [x] **Step 5: The regression that started M1, at the loop that had it**

`runPlay` needed no wrap line and never will. Pin the behaviour anyway, because
the behaviour is what the learner meets and the type is only how it is
guaranteed.

- [x] **Step 6: The behavioural pins — the memo, and the WIRING**

```go
// One word costs one fetch, however often it is played.
func TestOneWordCostsOneFetchHoweverOftenItIsPlayed(t *testing.T)

// And something USES it: build deps the way a loop does, run withStore, and
// require a fetch to reach the disk. The boundary review found that deleting the
// whole production wiring left the suite green, because every disk test built
// the layering by hand.
func TestWithStorePutsTheDiskCacheUnderTheMemo(t *testing.T)
```

**NOT a test at the loop.** The obvious name would put "a sitting" in it, and it
would be the wrong test: with `deps.audio` a `*audioSeam` there is no wrap line
in `runPlay` to forget, so the loop-specific failure mode no longer exists. What
can still regress is the wiring, and that is what the second row pins.

`runPlay` itself is also awkward to drive — `isTerminal(stdout)` is a real
syscall with no seam, so the entry point refuses before it reaches a question.
**`playSession` one level down IS drivable in-process**, so "a sitting cannot be
tested" would be too broad a claim; what is true is that the thing this row would
have pinned is no longer a thing that can break.

- [x] **Step 7: Commit**

```bash
git add -A && git commit -m "#46 M1: the loop that replays words is the one that never cached them"
```

### Task 2: A durable home for the recording and the verdict

**Files:**
- Create: `cmd/define/store/audio.go` (its rows live in `storetest`, which holds
  both twins — a `store/audio_test.go` would test one)
- Modify: `cmd/define/store/yaml.go` (`RuntimeDirs`, `audioDir`, `perWordDirs`)
- Modify: `cmd/define/store/store.go` (the `Store` interface both twins satisfy)
- Modify: `cmd/define/store/mem.go` (the in-memory twin)
- Modify: `cmd/define/store/storetest/suite.go` (the conformance rows)
- Modify: `.gitignore` — it single-sources from `RuntimeDirs`, so adding a
  directory reddens `TestGitignoreCoversRuntimeDirs`. **That is the fan-out this
  file list exists to make visible:** `RuntimeDirs` has four consumers, and the
  first draft named two.

- [x] **Step 1: Append to `RuntimeDirs` and watch the classification guard fail**

```go
var RuntimeDirs = []string{"words", "events", "usage", "facts", "items", "audio"}
```

Run: `go test ./cmd/define/store/ -run PerWordDirs -v`
Expected: FAIL — "audio is not classified in perWordDirs". **That failure is the
guard doing its job**, and it is the reason this step comes before the code: it
is what makes `Forget` take a word's recordings with it rather than leaving them
for someone to notice (`#10`'s BR-45, where a forgotten word kept the facts and
items that made it worth forgetting).

- [x] **Step 2: Classify it, on BOTH axes**

`{path: y.audioDir(), scoped: true}` — per-word, and language-scoped **the way
its siblings are**: `audioDir()` scopes on the STORE's language (`y.lang`,
`yaml.go:178`), exactly like `factsDir` and `itemsDir`. It is a shelf, not the
identity — the identity is `audioKey`, and the first draft's justification here
("AudioCandidates keys on the voice's Lang and Locale") was the false sentence
PQ-1 quoted. Do not restate the key in this step; it is defined once, above.

- [x] **Step 3: Teach `perWordDir` that a word can own SEVERAL files**

One word has one file per voice, plus a blob beside each record — the shape
`audioKey`/`audioRecord` define and that nothing else in this plan restates.
`perWordDir` gains a prefix glob rather than an exact name, and `Forget` removes
everything the word owns. Pin it: forget a word holding two voices' recordings
AND a stale verdict, and nothing of it survives.

- [x] **Step 4: The KEY, derived from the seam rather than from the word**

The filename is `<slug>--<digest>` where the digest is over the candidate list
`Fetch` is actually given — `strings.Join(urls, "\n")`, the very string
the seam keys its memo on. Two things fall out and both are the
point: `-locale gb` and `-locale us` are different files rather than one wrong
one, and `Forget` still globs `<slug>--*` because the word is what the file is
filed under.

Pin the collision directly, because it is the Critical this task exists to
avoid: two utterances for the SAME word with different voices must not read each
other's bytes.

- [x] **Step 5: The record, carrying `from`, with the TTL as a named constant**

```go
// audioVerdictTTL is how long "the CDN has no recording for this" is believed.
//
// NOT FOREVER, which is what the in-memory misses set can afford and this
// cannot. The CDN gains recordings over time, so a permanent verdict is a word
// that can never start working — and the words most likely to gain one are the
// rare words a learner is most likely to want. Thirty days costs four candidate
// requests a month per unrecorded word and bounds the staleness.
const audioVerdictTTL = 30 * 24 * time.Hour
```

- [x] **Step 6: ARCH-ORDER — what happens when the process dies mid-write**

M1 is the first DURABLE state this issue adds, and the first BINARY artifact the
working directory holds, so the ordering is named rather than left to be
discovered:

- **process death mid-write** — the bytes go through `writeBytesAtomic`, whose
  `.tmp-*` shadow is already covered by `RuntimeFiles`. A half-written recording
  is therefore not representable; a missing one re-fetches, which is the
  behaviour with no cache at all.
- **two `define` processes in one directory** — both may fetch and both may
  write. Last writer wins and the content is identical, because the key IS the
  candidate list; there is no merge and nothing to lose.
- **record and blob written separately** — write the BLOB first, then the record.
  A record naming a blob that does not exist would serve an empty recording; a
  blob with no record is invisible and re-fetched. Order the pair so the failure
  mode is a wasted fetch rather than silence.
- **not applicable, and why:** no cancellation path (a `Fetch` that returns is
  done), and no ordering between words (each key is independent).

- [x] **Step 7: Conformance rows in `storetest`, so Mem and YAML are both held**

Rows: a recording round-trips WITH its `from`; two voices of one word do not
collide; a verdict round-trips with its date; a verdict older than the TTL reads
as absent; `Forget` takes every file a word owns, hit and verdict together.

- [x] **Step 8: Commit**

### Task 3: The disk decorator

**Files:**
- Create: `cmd/define/audiodisk.go`, `cmd/define/audiodisk_test.go`
- Modify: `cmd/define/main.go` (wrap when a store exists)

- [x] **Step 1: Write the failing test, against the real fake CDN and a real temp dir**

```go
// THE POINT OF THE WHOLE MILESTONE: a second PROCESS pays nothing.
func TestASecondRunReusesTheRecordingOnDisk(t *testing.T) {
	cdn := newFakeCDN(t)
	dir := t.TempDir()
	// THE PRODUCTION LAYERING, exactly: memo OUTERMOST over disk over HTTP, so a
	// repeat within one process never touches the filesystem. The disk layer is
	// applied where the store is known; the memo is applied at the entry point.
	// Wiring the test the other way round would make "the line tests exercise is
	// the line production runs" false in the very task that argues for it.
	first := newAudioSeam(newDiskAudioCache(openStoreIn(t, dir), cdn.source()))
	mustFetch(t, first, "sycophantic")
	// A NEW decorator over a NEW memo — everything in memory is gone, exactly as
	// it is between two runs of the binary.
	second := newAudioSeam(newDiskAudioCache(openStoreIn(t, dir), cdn.source()))
	mustFetch(t, second, "sycophantic")
	if got := cdn.requests(); got != 1 {
		t.Errorf("the CDN saw %d requests across two processes; want 1", got)
	}
}
```

- [x] **Step 2: Run it, watch it fail** (`newDiskAudioCache` undefined).
- [x] **Step 3: Implement the decorator.** Read-through on `Fetch`; write the
      bytes on a hit; write a dated verdict on `ErrNoAudio`; leave
      `ErrFetchFailed` alone, because a transient outage must not be recorded as
      a permanent absence — the taxonomy `fetch.go` already documents
      is the single source of that distinction and this derives from it.
- [x] **Step 4: The verdict half of the same test** — an unrecorded word costs
      four candidate requests once, and zero on the second process.
- [x] **Step 5: Degrade, never fail.** An unwritable directory, a corrupt file, a
      store that could not open: the recording still plays, from the network.
      Pin each — a cache that can break playback is worse than no cache.
- [x] **Step 6: `--forget` end to end.** Forget a word whose recording is on
      disk; the file is gone.
- [x] **Step 7: Commit, then `sdlc milestone-close --issue 46 --milestone M1`.**

---

## Chunk 2: M2 — one span walk, two consumers

### Task 4: `deckSpans` — the walk, in visible coordinates

**Files:**
- Create: `cmd/define/deckwords.go`, `cmd/define/deckwords_test.go`

- [x] **Step 1: Write the failing tests.** The rows that matter are the ones a
      naive implementation gets wrong:

```go
// A span walk over ALREADY-COLOURED text must not land inside an escape.
//
// This is the row that decides the design. The reveal contains an entry Render
// has already highlighted, so the text carries "\x1b[1;32m" runs — and a walk
// over raw bytes tokenises `1`, `32m` and the letters of the escape as words,
// then reports columns that count escape bytes as cells. Everything downstream
// is then wrong in a way that still renders.
func TestASpanWalkSkipsEscapeSequences(t *testing.T)

// Longest phrase wins, inherited from highlightSpans rather than re-decided.
func TestAPhraseBeatsItsFirstWord(t *testing.T)

// Coordinates are DISPLAY CELLS, on the line the word is actually on.
func TestSpansAreLocatedOnTheirOwnLine(t *testing.T)

// A word that is not in the deck produces nothing — the walk marks what the
// learner is learning, not every word.
func TestOnlyDeckWordsAreSpanned(t *testing.T)
```

- [x] **Step 2: Run them, watch them fail.**
- [x] **Step 3: Implement.** Per line: `plain, cols := visibleIndex(line)`, run
      the EXISTING `highlightSpans(plain, v)`, map each `known` span's byte range
      through `cols`. No new tokeniser, no change to `highlightSpans` — it stays
      the one producer and this is the coordinate layer over it.
- [x] **Step 4: Run them, watch them pass.**
- [x] **Step 5: Mutation sweep.** Revert the escape-awareness (walk the raw line)
      and confirm `TestASpanWalkSkipsEscapeSequences` reddens by name. *A pin
      that cannot fail is not a pin.*
- [x] **Step 6: Commit.**

### Task 5: `RegionWord`, and the invariant it must satisfy

**Files:**
- Modify: `cmd/define/render.go` (the kind, `String`, `identifier`)
- Modify: `cmd/define/replraw.go` (`playRegion`'s registry)
- Modify: `atlas/define.md`
- Test: `cmd/define/editorloop_test.go`, `cmd/define/deckwords_test.go`

- [x] **Step 1: Declare the kind above `numRegionKinds`** and run the suite
      WITHOUT touching anything else.
      Expected: `TestEveryRegionKindIsActionable`,
      `TestEveryRegionKindIsNamed` and `TestAtlasDescribesEveryRegionKind` all
      fail, unedited. That is the registry working — the kind is exercised the
      moment it is declared.
- [x] **Step 2: Add the `playRegion` row.** A `RegionWord` plays `r.Word` in the
      session's voice — the same resolution `RegionHeadword` gets, because they
      offer the same action.
- [x] **Step 3: Name it and describe it in the atlas.**
- [x] **Step 4: Hold it to `#12`'s BR-14 invariant.** Extend
      `TestAPromptRegionCoversTheTextItClaims` to every region the write door
      produces, not just the prompt's, and run it over coloured text — this is
      the first kind whose regions are computed on a string that already carries
      escapes, which is exactly where "the region covers what it claims" is
      easiest to get wrong.
- [x] **Step 5: Write `markClickable`'s precondition down, and check it (PQ-2).**

```go
// TWO PRODUCERS, ONE LINE. markClickable sorts by Col and walks ONE cursor left
// to right, so a duplicate span at a column the cursor has already passed can
// never match — and every later region on that line silently loses its
// underline. RegionAt compounds it by resolving in APPEND ORDER rather than by
// narrowest span. Neither was wrong while one producer owned the line.
//
// A Choice prompt is the concrete case: RegionHeadword sits at Line 1 Col 0 and
// the headword is a deck word, so wordRegions would emit a second region on the
// same cells.
func TestALinesRegionsAreDisjointAndAscending(t *testing.T)
```

Enforce it in `mergeRegions`, with the existing region winning: a
`RegionHeadword` carries the entry's lookup key, which is more specific than a
text match (`define jalapeno` renders `jalapeño`).

- [x] **Step 6: Mutation-sweep it** — append an overlapping region by hand and
      confirm the property test reddens, and that a later region on that line
      loses its underline in the golden. The second half is what makes the
      finding's *consequence* visible rather than just its cause.
- [x] **Step 7: Commit.**

### Task 6: The write door applies both

**Files:**
- Modify: `cmd/define/main.go` (`writeRendered`), `cmd/define/play_loop.go`
- Test: `cmd/define/play_loop_test.go`

- [x] **Step 1: Write the failing tests, one per surface.**

```go
// Every word the learner is learning is clickable, wherever it is written.
func TestEveryDeckWordInASittingIsClickable(t *testing.T)

// COLOUR IS A DISCOVERY SIGNAL. On in prose, off where the text is the deck.
func TestClozeOptionsAreClickableButNotColoured(t *testing.T)
func TestChoiceOptionGlossesAreColoured(t *testing.T)

// EVERY FORM IS CLASSIFIED, over docSyncForms — so a form added to play cannot
// arrive without an answer to "is this text the deck itself".
func TestEveryFormHasASurface(t *testing.T)

// The board renders through boardFooter into the FOOTER and never reaches the
// write door, so it is uncoloured by construction today. Asserting "the board is
// not coloured" would therefore be a pin that cannot fail (PQ-4). This asserts
// the reachable thing instead: cells that ARE deck words still carry no knownOn,
// which reddens the day someone routes the footer through the colour pass.
func TestBoardCellsCarryNoDeckColourEvenWhenTheyAreDeckWords(t *testing.T)

// And nothing is coloured twice: Render owns the entry's colour, because it
// colours with a per-region BASE and ANSI does not nest.
func TestTheRenderedEntryIsNotRecoloured(t *testing.T)
```

- [x] **Step 2: Run them, watch them fail.**
- [x] **Step 3: Give `writeRendered` the vocabulary and the surface.** It appends
      `wordRegions(text, v)` to the regions it was given, and colours the text
      OUTSIDE any embedded render — the range `marksIn` already locates, so the
      boundary is found rather than assumed.
- [x] **Step 4: Derive the surface from the FORM, not the call site (PQ-4).**
      There is ONE prompt write site serving both `Choice` and `Cloze`, so the
      site cannot tell them apart. `surfaceOf(q)` switches on `q.Form()` — the
      NAME, a string; never a type switch on a concrete form, which `#6`'s
      Done-when forbids.
- [x] **Step 5: Hang the classification on the extent that is already derived.**
      `TestEveryFormHasASurface` walks `docSyncForms`, whose completeness
      `TestEveryFormIsEnrolled` already guarantees by parsing `play/*.go`. So a
      new form cannot arrive unclassified, and no second list is created. Assert
      the guard fires: adding a form with no classification must redden it.
      *A hand-maintained extent is half a guard* — `#12` BR-17.
- [x] **Step 6: Run the whole suite, plus `go vet` under all three tag sets.**
- [x] **Step 7: Mutation sweep** — turn the surface rule off and confirm the
      cloze row reddens; drop `wordRegions` and confirm the clickability row
      reddens.
- [x] **Step 8: Update `README.md` and `atlas/define.md`.** The key table and the
      click sentence both describe what a click does; the derived guards from
      `#12` will hold them to it.
- [x] **Step 9: State M2's operating envelope (ARCH-CONSTRAINTS).** M1 carries
      one and M2 did not. `deckSpans` runs **once per write** — per question, per
      reveal, per lookup — never per keystroke: the sitting redraws from a buffer
      and the REPL's input line uses the streaming `highlightSpans` path it
      already had. The largest input is a full-entry reveal, a few hundred lines,
      walked once against a deck of at most a few thousand keys via a map lookup
      per token. That is the bound; if a future surface wants this per keystroke,
      that is a different design and should be a finding, not a silent regression.
- [x] **Step 10: Commit, then `sdlc close --issue 46`.**

---

## Verification

Automated, and each row names the thing it would catch:

1. `go test ./...`, `go vet ./...` under default, `pty` and `conformance` tags,
   `gofmt -l` clean.
2. `TestOneWordCostsOneFetchHoweverOftenItIsPlayed` and
   `TestWithStorePutsTheDiskCacheUnderTheMemo` — the memo, and that something uses it.
3. `TestASecondRunReusesTheRecordingOnDisk` — two processes, one request.
4. `TestPerWordDirsCoverEveryRuntimeDir` — `Forget` takes the audio.
5. `TestASpanWalkSkipsEscapeSequences` — the design's load-bearing row.
6. `TestAPromptRegionCoversTheTextItClaims`, widened — no region claims text it
   does not cover, now on coloured text.
7. The surface rows — clickable-not-coloured in a cloze, coloured in a gloss.

Manual, once, over a pty, because a click and a colour are things a person sees:

8. `--play` a deck with an authored item: click each of the four option words and
   hear four different recordings; confirm none of them is green.
9. Look up a word whose definition contains another deck word: it is green AND
   clicking it plays it.
10. Replay the same sitting after quitting: no network traffic for the audio.

## Revisions

### 2026-09-07 — plan-quality round 1: 1 Critical, 3 Important, 3 Minor

**Reason.** `sdlc change-code --issue 46` refused. Every finding was checked
against the code before being accepted; all four blockers were correct, and two
of them reversed decisions this plan had argued for.

**PQ-1 (Critical) — the durable key was the word; the seam's key is the
candidate list.** M1 keyed on `audio/<lang>/<slug>` and justified it with a claim
that is false twice over: `audioDir` scopes on the STORE's language rather than
the voice's, and locale never appeared in the path at all. The seam keys on
`strings.Join(urls, "\n")` over a list built from a source voice, a session
voice and the spellings — so `-locale gb` and `-locale us`, and `-pron en red`
beside a plain `red`, would have shared one file and served the wrong recording.
The key is now a digest of that list, with `<slug>--` kept as the prefix `Forget`
globs. The same finding caught that `from` was dropped, which `spokeSource` and
`reportVoice` depend on — so the artifact is bytes AND provenance.

**PQ-2 (Important) — `markClickable` had an unwritten precondition that M2 is
the first thing able to violate.** One cursor, left to right; a duplicate span at
an already-passed column silently kills every later underline on the line. A
`Choice` prompt is the concrete case, since its headword is a deck word and would
get a region from both producers. Now a stated precedence rule (the existing
region wins, because it carries the lookup key) and a checked property
(disjoint, ascending) rather than a comment.

**PQ-3 (Important) — moving the wrap into `realDeps` would have reversed the
lesson it cited.** No test calls `realDeps()`; the suite builds `deps{...}`
literals at 21 sites, so every loop test would have run on an uncached source —
exactly the `#2` I-1 failure `repl.go:256` records. The wrap stays in the loop
and **the guard becomes the derived thing**: the loops are parsed as the callees
of `run()` that take a `deps`, with a fail-closed floor. That is `#12` BR-17's
correction applied before the bug rather than after.

**PQ-4 (Important) — the surface is a property of the form, not the write
site.** One prompt write site serves both `Choice` and `Cloze`, so it cannot tell
them apart; and the board never reaches the write door at all, which made
`TestBoardCellsAreNotColoured` a pin that could not fail. The classification now
comes from `q.Form()` and hangs on `docSyncForms`, whose completeness is already
derived — and the board's row asserts the reachable fact instead.

**Minors, all taken rather than carried:** Task 1's "Expected: FAIL" now names
which test fails and adds the mutation sweep; Task 2's file list gains
`.gitignore` and `store/store.go`, the two `RuntimeDirs` consumers it had missed;
M1 gains an ARCH-ORDER statement (process death mid-write, two processes in one
directory, blob-before-record); M2 gains the ARCH-CONSTRAINTS envelope M1 already
had.

### 2026-09-07 — plan-quality round 2: 1 Critical, 1 Important

**PQ-9 (Critical) — the derivation invented in round 1 was wrong, and measurably
so.** It read "loops" as `run()`'s deps-taking callees; that set is seven
one-shot commands (`ask`, `defineOnce`, `forgetWord`, `runHarvest`,
`runReflect`, …) and contains neither function that actually wraps — the wraps
live in `replLines` and `replRaw`, one level below what `run` dispatches to.

**The lesson is bigger than the finding, and it is about my own two rounds.**
Round 1 replaced a bad wrap site with a derived guard; round 2 found the
derivation was worse than the thing it replaced. Checking properly: there is no
clean syntactic derivation of "loop" here — a dispatcher and a loop share a
signature, and tests drive BOTH levels directly, so hoisting the wrap to `repl`
would reproduce PQ-3's failure one level down (`askroute_test.go:31` calls
`replLines`).

**So the fix is not a better enumeration but an operation that does not need
one.** The wrap becomes idempotent, over-application becomes a
no-op, and the guard is then free to over-derive: every function taking a `deps`,
an `io.Reader` and writers must wrap. A false positive costs a type assertion. An
enumeration that cannot be wrong beats one that is exactly right and
hand-maintained — and the sweep now runs in BOTH directions, because a guard that
cannot be shown to EXCLUDE anything has not been shown to derive anything.

**PQ-10 (Important) — I fixed PQ-1 at the site it named and left four
restatements of the superseded shape.** The entity table still declared
`audioName`/`audioVerdict`, the `perWordDir` bullet still said "`<slug>.mp3` or
`<slug>.none`", Task 2 Step 3 repeated that pair eight lines before Step 4
contradicted it, and Step 2 still carried the exact sentence PQ-1 had quoted as
false. An implementer working Task 2 in order would have built the rejected
design.

**This is `#12`'s stale-restatement family, one issue later, in a document rather
than in code** — and `workshop/targets/derived-restatement.md` exists precisely
because fixing named sites does not close it. The rule applied here: the
artifact's key and on-disk shape are stated ONCE, in `audioKey`/`audioRecord`,
and every other mention references that definition rather than repeating it. The
target's checklist gains "the plan's own earlier sections" as a sweep row.

### 2026-09-07 — plan-quality round 3: 1 Critical (PQ-9, third disposition)

**PQ-10 addressed. PQ-9 not, and the reason is worth more than the fix.**

The plan said the wraps live in `replLines` and `replRaw`. `replraw.go:264` is
inside **`runEditor`** (`replraw.go:253`), not `replRaw` (`replraw.go:18`) — and
the finding's own text had said so. **Three of three statements this plan has
made about where the wrap lives have been wrong**: round 1's `realDeps`, round
2's `run()` callees, round 3's `replRaw`.

That is no longer a run of carelessness, it is evidence the question is the wrong
one. `runEditor` takes `(ctx, keys <-chan Key, interrupts, d deps, opt options,
con console) int` — no `io.Reader`, no `io.Writer` — so round 2's predicate
excluded it structurally while ~55 tests drive it directly. Being more careful
would have produced a fourth answer, not a right one.

**Delta.** The predicate widens to every function taking a `deps` and an
`options`, which needs no judgement about what a loop is; idempotence makes that
generosity free. And the guard now **checks itself**: the enclosing functions of
every non-test wrap call must be a subset of its own
membership, so a wrap site the predicate cannot see is reported as a failure of
the guard. The sweep exercises both mechanisms rather than both directions of
one, and step (c) deliberately re-narrows the predicate to prove the blind-spot
half fires.

**Folded in while here:** Task 3's test wired the disk cache outermost while
production yields the memo outermost. Both work; they are not the same wiring,
and this plan leans on "the line tests exercise is the line production runs"
twice. Production layering — memo over disk over HTTP — is now stated once and
the test matches it.

### 2026-09-07 — plan-quality round 4: cleared, and the recorded finding taken anyway

**Plan-quality passed** (round cap reached; one finding recorded, not blocking).
Taking it before implementation regardless, because it names a SEAM decision and
those are expensive to reverse once sites carry them.

**The finding: "that is ~8 functions" was the 4th wrong statement of the wrap
set; the gate measured the predicate and found three times that.** And Step 4
would have written the wrap into most of them, functions that never read
`d.audio` (`vocabularyFor`, `clozeAsk`, `todaysQuestions`,
`newCommandCtx`, `playRegion`, `submitLine`), contradicting the plan's own
Architecture line and leaving a guard that checks a token appears in a body.

**Four wrong statements about one set is not four mistakes; it is the wrong
question asked four times.** The finding offered three shapes and the third is
the only one that stops asking it: make `d.audio` unreachable except through a
seam that is already cached. `deps.audio` becomes a `*audioSeam` — a pointer, so
every by-value copy of `deps` shares one memo, which the decorator could never
guarantee. Then there is no wrap to forget, no predicate to derive, no guard to
keep honest, and **the compiler enumerates the construction sites** — the only
enumeration in this program that cannot drift. `#2`'s I-1 lesson stops being a
convention: a test cannot build a `deps` whose audio differs in kind from
production's.

The cost is mechanical churn across the test literals, paid once by the compiler.
The plan now carries no prose statement of the set's membership or size, which is
the rule the finding actually asked for.

### 2026-09-07 — M1 boundary review, rounds 1 and 2

**Appended rather than edited in place, which round 2 (BR-9) had to say twice.**
Round 1's fixes were made by editing the body — and one of them was a blind
symbol substitution that left five passages ungrammatical and introduced a fresh
break at Task 1 Step 3. AGENTS.md §1 says revisions are APPENDED; the reason is
exactly this, that an in-place edit of a document nobody re-reads is how a
sentence stops being English without anyone noticing.

**Round 1: 3 Critical, 6 Important, 5 Minor.** The Criticals: the entity table
called `RenderOpts` modified when `writeWords` was added beside `writeRendered`
instead; deleting the entire production wiring of `diskAudioCache` from
`withStore` left the suite green, because every disk test built the layering by
hand; and a `/lang` switch stranded recordings on the previous language's shelf
while `--forget` reported success.

**Round 2: the filing scheme reversed, and the reversal itself drifted.** BR-10
found that `<slug>--<digest>` rested on a false claim — a slug CAN contain `--`
(`re-` slugs to `re--ddf427`), so forgetting `re` took a different word's
recordings. The shipped layout is `audio/<slug>/<digest>.{mp3,yaml}`, removed by
`RemoveAll` on an exact directory name. **Then BR-16 found ten restatements of
the superseded scheme left behind**, two of them doc comments on the very field
the change was about, sixty lines above the function that contradicted them.

**Delta to this plan**, all of it recorded here rather than rewritten above:

- `audio/` is FLAT, not per-language (BR-2). A shelf would be a second statement
  of which voice a recording is for, and the digest already is the first.
- A word owns a DIRECTORY, not a filename prefix (BR-10). Any separator has to
  be reasoned about against `Slug`'s alphabet, and that reasoning is what was
  wrong; a directory name admits no ambiguity.
- `perWordDir` carries THREE axes. Each was added by a bug the previous set could
  not see, which is now the type's doc comment.
- The blob read is capped at 4MB, mirroring the network path — the directory is
  documented as hand-editable, so its contents are untrusted input.
- A verdict REMOVES the recording it supersedes, pinned through the filesystem
  because asserting through `Audio` cannot see the stale file (BR-12).
- Every degrade branch in `YAML.Audio` is driven: corrupt record, vanished blob,
  unwritable path, oversized blob (BR-6). Two of them previously carried a
  promise nothing checked.

### 2026-09-07 — close rounds 3-7

**Appended, and this time for rounds that had none.** Round 2 of M1 already
made this point (BR-9) and the close gate had to make it again: five rounds of
close review changed the design in ways the body above no longer describes.

**The `writeWords` signature above is superseded.** It is documented as taking a
`Vocabulary` and a colour flag; it takes the `deps` and the `options`, derives
both, and additionally takes the SUBJECT — the word the text is asking about.
Each change came from a finding:

- **BR-21 — the vocabulary argument was removed** because a guard over an
  argument can only check its source token. `nil` reddened the guard;
  `vocabularyFor(d, opt)` did not, though that returns nil whenever colour is off
  and would silently remove every click target on `--no-color`. A parameter with
  one correct value is a parameter that will eventually be given another. When
  `deps{}` slipped through in its place, the guard grew a clause requiring the
  identifier the loop holds.
- **BR-29 — the subject argument was added.** Colouring reached a meaning
  question's headword, so the word under test was painted deck-green: "you have
  looked this up before", told to a learner being asked whether they know it.
  The rule is now stated once — **a question never marks its own subject as
  known** — and the reveal is deliberately exempt, because there the word has
  been shown and its sentence restored. The subject is held out of the COLOUR
  pass only; its click target survives, since a learner may well want to hear
  the word they are being asked about.
- **BR-22 — the empty-payload predicate reached all three layers.** Fixed at the
  store in round 4, the memo in round 5, and finally the fetch in round 6, where
  an empty 200 had also been ABANDONING THE REMAINING CANDIDATES — so a word
  whose recording sat one URL later played nothing at all. An empty body is now
  exactly a 404: skip it, keep walking, and record a verdict with the usual TTL
  if nothing answers.
- **BR-30 — the store-layout blocks now DERIVE from `store.RuntimeDirs`.** Both
  restated it by hand and the README had already fallen behind by one directory.
  Fourth finding in the family `workshop/targets/derived-restatement.md` exists
  for, and its rule is what was applied: derive wherever the fact is
  machine-readable.
