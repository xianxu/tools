# Cached audio and deck-word marks — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A recording is fetched once and kept; and every word from the learner's
deck is clickable wherever it is written, and coloured wherever colour still
means something.

**Architecture:** Two independent halves. **M1** makes `cachingAudioSource`
structural rather than remembered — one construction site, so no loop can hold
an uncached source — and gives it a home on disk under the working directory,
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
| `wordRegions` | `cmd/define/deckwords.go` | new |
| `RegionWord` | `cmd/define/render.go` | new |
| `surface` | `cmd/define/deckwords.go` | new |
| `audioName` | `cmd/define/store/audio.go` | new |
| `audioVerdict` | `cmd/define/store/audio.go` | new |
| `perWordDir` | `cmd/define/store/yaml.go` | modified |
| `RenderOpts` | `cmd/define/render.go` | modified |

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

- **`audioKey`** — the identity of one cached recording, and **it is the SEAM's
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

- **`audioRecord`** — what is stored beside the bytes: `from`, the day, and
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
  `<slug>.yaml`. Audio is the first per-word directory whose file is not a single
  YAML: it is `<slug>.mp3` or `<slug>.none`. `Forget` removes the word's whole
  set.

- **`RenderOpts`** *(modified)* — no field changes; `Vocab` STAYS. `Render` keeps
  colouring the entry itself, because it colours with a per-region BASE style
  (amber part-of-speech labels, the `p.ex` example style) and ANSI does not nest
  — a flat pass at the write door cannot reproduce those resume rules. What
  changes is that the write door now colours the text OUTSIDE the render.

**Test surface.** Every entity above is PURE and gets a colocated unit test that
runs with no IO and no fake: `deckwords_test.go`, `store/audio_test.go`. The one
exception is `perWordDir`, whose test is the existing
`TestPerWordDirsCoverEveryRuntimeDir` — a guard, not a new file.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `diskAudioCache` | `cmd/define/audiodisk.go` | new | the working directory |
| `cachingAudioSource` | `cmd/define/fetch.go` | modified | an `AudioSource` |
| `audioDir` | `cmd/define/store/yaml.go` | new | the filesystem |
| `writeRendered` | `cmd/define/main.go` | modified | stdout + the click map |

- **`diskAudioCache`** — an `AudioSource` decorator that reads and writes
  recordings under the store's directory.
  - **Injected into:** nothing pure; it is the outermost decorator on
    `deps.audio`, so every existing consumer is unchanged.
  - **A DECORATOR, LIKE ITS SIBLING.** `cachingAudioSource`'s doc comment records
    why the memo is a decorator rather than a map inside the REPL loop: "the
    existing `fakeCDN` request recorder is the assertion that a replay costs no
    second request — no bespoke test scaffolding". That argument carries over
    exactly, and it is the reason this half needs no new fake: `fakeCDN` plus a
    `t.TempDir()` store is the whole rig.
  - **Future extensions:** eviction. Deliberately NOT built — a recording is
    10–30KB and a 1000-word deck is roughly 30MB, which is smaller than the
    dictionary it sits beside. Recorded as a number rather than omitted, so the
    decision is reviewable.

- **`cachingAudioSource`** *(modified)* — unchanged behaviour; what changes is
  that it is constructed ONCE.
  - **THE BUG IS THAT WRAPPING IS REMEMBERED.** `repl.go:257` and
    `replraw.go:264` each wrap; `runPlay` does not, so the one loop that replays
    the same handful of words is the one with no cache. Fixing `runPlay` fixes
    the site; making the source impossible to obtain unwrapped fixes the class.
  - **Injected into:** `deps.audio`, in `realDeps()`.

- **`audioDir`** — the per-language audio directory, appended to `RuntimeDirs`
  AT THE TAIL (the file's own positional-index rule).

- **`writeRendered`** *(modified)* — the one door for text that reaches the
  sitting's scrollback, so it is where the span walk lands. It gains the
  vocabulary and the surface.

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

**PQ-3 REVERSED THIS TASK'S FIRST DRAFT, and it was right to.** The draft moved
the wrap into `realDeps()` and claimed "a test driving any loop through realDeps
now gets the same source production gets" — but **no test calls `realDeps()`**;
the suite builds `deps{...}` literals at 21 sites. Moving the wrap there would
have left every loop test on an uncached source, which is precisely the `#2` I-1
failure `repl.go:256` exists to record: the line the tests exercise must be the
line production runs.

So the wrap stays in the loop, where both paths traverse it, and **the derived
thing is the guard**. That is the same correction `#12` BR-17 made: the extent
must be parsed, not remembered.

- [ ] **Step 1: Write the derived guard, and watch it fail on `runPlay`**

```go
// EVERY LOOP WRAPS, and the set of loops is PARSED rather than listed (#46 PQ-3).
//
// The bug this exists for: repl and replraw each wrapped and runPlay did not, so
// the one loop that replays the same handful of words all sitting was the one
// with no cache — and an unrecorded word cost four candidate requests on every
// replay, because the misses set that prevents exactly that was never built.
//
// A hand-listed set of loops would be the same bug one level up (#12 BR-17: a
// hand-maintained extent is half a guard). The loops are exactly the functions
// run() dispatches to that take a deps, so that is what this reads.
func TestEveryLoopWrapsTheAudioSource(t *testing.T) {
	for _, fn := range loopsDispatchedByRun(t) { // AST: callees of run() taking deps
		if !callsIn(t, fn, "newCachingAudioSource") {
			t.Errorf("%s takes deps and is dispatched by run(), but never wraps "+
				"d.audio — it will re-fetch every recording, and an unrecorded "+
				"word will cost four requests per replay.", fn.Name.Name)
		}
	}
	if len(loopsDispatchedByRun(t)) < 2 {
		t.Fatal("found fewer than two loops; the derivation is under-deriving " +
			"and this guard would certify nothing")
	}
}
```

The `len(...) < 2` floor is the fail-closed clause `#12` BR-17 taught: a
derivation satisfied by finding nothing is not a derivation.

- [ ] **Step 2: Run it**

Run: `go test ./cmd/define/ -run EveryLoopWraps -v`
Expected: FAIL, naming `runPlay` and nothing else.

- [ ] **Step 3: Add the wrap to `runPlay`**, carrying `repl.go:256`'s reason
      across verbatim — it is the same reason.

- [ ] **Step 4: Run it again.** Expected: PASS.

- [ ] **Step 5: Mutation-sweep the guard, not just the fix.** Delete the wrap
      from `repl.go` and confirm the guard names `repl`; restore. *A pin that
      cannot fail is not a pin*, and a guard that only ever names the loop you
      already fixed is sized to the bug rather than the class.

- [ ] **Step 6: The behavioural pin, at the loop, through the CDN recorder**

```go
// The regression itself, stated where a learner would meet it: a sitting that
// shows the same word twice fetches it once.
func TestASittingFetchesARecordingOnce(t *testing.T) { /* fakeCDN + runPlay, two questions on one word */ }
```

- [ ] **Step 7: Commit**

```bash
git add -A && git commit -m "#46 M1: the loop that replays words is the one that never cached them"
```

### Task 2: A durable home for the recording and the verdict

**Files:**
- Create: `cmd/define/store/audio.go`, `cmd/define/store/audio_test.go`
- Modify: `cmd/define/store/yaml.go` (`RuntimeDirs`, `audioDir`, `perWordDirs`)
- Modify: `cmd/define/store/store.go` (the `Store` interface both twins satisfy)
- Modify: `cmd/define/store/mem.go` (the in-memory twin)
- Modify: `cmd/define/store/storetest/suite.go` (the conformance rows)
- Modify: `.gitignore` — it single-sources from `RuntimeDirs`, so adding a
  directory reddens `TestGitignoreCoversRuntimeDirs`. **That is the fan-out this
  file list exists to make visible:** `RuntimeDirs` has four consumers, and the
  first draft named two.

- [ ] **Step 1: Append to `RuntimeDirs` and watch the classification guard fail**

```go
var RuntimeDirs = []string{"words", "events", "usage", "facts", "items", "audio"}
```

Run: `go test ./cmd/define/store/ -run PerWordDirs -v`
Expected: FAIL — "audio is not classified in perWordDirs". **That failure is the
guard doing its job**, and it is the reason this step comes before the code: it
is what makes `Forget` take a word's recordings with it rather than leaving them
for someone to notice (`#10`'s BR-45, where a forgotten word kept the facts and
items that made it worth forgetting).

- [ ] **Step 2: Classify it, on BOTH axes**

`{path: y.audioDir(), scoped: true}` — per-word, and language-scoped, because
`AudioCandidates(word, voice)` keys on the voice's Lang and Locale: the English
and Spanish recordings of `red` are different files and forgetting one must not
take the other.

- [ ] **Step 3: Teach `perWordDir` that a word can own more than one file**

Audio is the first per-word directory whose file is not `<slug>.yaml`: a hit is
`<slug>.mp3`, a verdict is `<slug>.none`. Give `perWordDir` the extension set it
owns and have `Forget` remove each. Pin it: forgetting a word with BOTH a
recording and a stale verdict leaves neither.

- [ ] **Step 4: The KEY, derived from the seam rather than from the word**

The filename is `<slug>--<digest>` where the digest is over the candidate list
`Fetch` is actually given — `strings.Join(urls, "\n")`, the very string
`cachingAudioSource` keys its memo on. Two things fall out and both are the
point: `-locale gb` and `-locale us` are different files rather than one wrong
one, and `Forget` still globs `<slug>--*` because the word is what the file is
filed under.

Pin the collision directly, because it is the Critical this task exists to
avoid: two utterances for the SAME word with different voices must not read each
other's bytes.

- [ ] **Step 5: The record, carrying `from`, with the TTL as a named constant**

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

- [ ] **Step 6: ARCH-ORDER — what happens when the process dies mid-write**

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

- [ ] **Step 7: Conformance rows in `storetest`, so Mem and YAML are both held**

Rows: a recording round-trips WITH its `from`; two voices of one word do not
collide; a verdict round-trips with its date; a verdict older than the TTL reads
as absent; `Forget` takes every file a word owns, hit and verdict together.

- [ ] **Step 8: Commit**

### Task 3: The disk decorator

**Files:**
- Create: `cmd/define/audiodisk.go`, `cmd/define/audiodisk_test.go`
- Modify: `cmd/define/main.go` (wrap when a store exists)

- [ ] **Step 1: Write the failing test, against the real fake CDN and a real temp dir**

```go
// THE POINT OF THE WHOLE MILESTONE: a second PROCESS pays nothing.
func TestASecondRunReusesTheRecordingOnDisk(t *testing.T) {
	cdn := newFakeCDN(t)
	dir := t.TempDir()
	first := newDiskAudioCache(openStoreIn(t, dir), newCachingAudioSource(cdn.source()))
	mustFetch(t, first, "sycophantic")
	// A NEW decorator over a NEW memo — everything in memory is gone, exactly as
	// it is between two runs of the binary.
	second := newDiskAudioCache(openStoreIn(t, dir), newCachingAudioSource(cdn.source()))
	mustFetch(t, second, "sycophantic")
	if got := cdn.requests(); got != 1 {
		t.Errorf("the CDN saw %d requests across two processes; want 1", got)
	}
}
```

- [ ] **Step 2: Run it, watch it fail** (`newDiskAudioCache` undefined).
- [ ] **Step 3: Implement the decorator.** Read-through on `Fetch`; write the
      bytes on a hit; write a dated verdict on `ErrNoAudio`; leave
      `ErrFetchFailed` alone, because a transient outage must not be recorded as
      a permanent absence — the taxonomy `cachingAudioSource` already documents
      is the single source of that distinction and this derives from it.
- [ ] **Step 4: The verdict half of the same test** — an unrecorded word costs
      four candidate requests once, and zero on the second process.
- [ ] **Step 5: Degrade, never fail.** An unwritable directory, a corrupt file, a
      store that could not open: the recording still plays, from the network.
      Pin each — a cache that can break playback is worse than no cache.
- [ ] **Step 6: `--forget` end to end.** Forget a word whose recording is on
      disk; the file is gone.
- [ ] **Step 7: Commit, then `sdlc milestone-close --issue 46 --milestone M1`.**

---

## Chunk 2: M2 — one span walk, two consumers

### Task 4: `deckSpans` — the walk, in visible coordinates

**Files:**
- Create: `cmd/define/deckwords.go`, `cmd/define/deckwords_test.go`

- [ ] **Step 1: Write the failing tests.** The rows that matter are the ones a
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

- [ ] **Step 2: Run them, watch them fail.**
- [ ] **Step 3: Implement.** Per line: `plain, cols := visibleIndex(line)`, run
      the EXISTING `highlightSpans(plain, v)`, map each `known` span's byte range
      through `cols`. No new tokeniser, no change to `highlightSpans` — it stays
      the one producer and this is the coordinate layer over it.
- [ ] **Step 4: Run them, watch them pass.**
- [ ] **Step 5: Mutation sweep.** Revert the escape-awareness (walk the raw line)
      and confirm `TestASpanWalkSkipsEscapeSequences` reddens by name. *A pin
      that cannot fail is not a pin.*
- [ ] **Step 6: Commit.**

### Task 5: `RegionWord`, and the invariant it must satisfy

**Files:**
- Modify: `cmd/define/render.go` (the kind, `String`, `identifier`)
- Modify: `cmd/define/replraw.go` (`playRegion`'s registry)
- Modify: `atlas/define.md`
- Test: `cmd/define/editorloop_test.go`, `cmd/define/deckwords_test.go`

- [ ] **Step 1: Declare the kind above `numRegionKinds`** and run the suite
      WITHOUT touching anything else.
      Expected: `TestEveryRegionKindIsActionable`,
      `TestEveryRegionKindIsNamed` and `TestAtlasDescribesEveryRegionKind` all
      fail, unedited. That is the registry working — the kind is exercised the
      moment it is declared.
- [ ] **Step 2: Add the `playRegion` row.** A `RegionWord` plays `r.Word` in the
      session's voice — the same resolution `RegionHeadword` gets, because they
      offer the same action.
- [ ] **Step 3: Name it and describe it in the atlas.**
- [ ] **Step 4: Hold it to `#12`'s BR-14 invariant.** Extend
      `TestAPromptRegionCoversTheTextItClaims` to every region the write door
      produces, not just the prompt's, and run it over coloured text — this is
      the first kind whose regions are computed on a string that already carries
      escapes, which is exactly where "the region covers what it claims" is
      easiest to get wrong.
- [ ] **Step 5: Write `markClickable`'s precondition down, and check it (PQ-2).**

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

- [ ] **Step 6: Mutation-sweep it** — append an overlapping region by hand and
      confirm the property test reddens, and that a later region on that line
      loses its underline in the golden. The second half is what makes the
      finding's *consequence* visible rather than just its cause.
- [ ] **Step 7: Commit.**

### Task 6: The write door applies both

**Files:**
- Modify: `cmd/define/main.go` (`writeRendered`), `cmd/define/play_loop.go`
- Test: `cmd/define/play_loop_test.go`

- [ ] **Step 1: Write the failing tests, one per surface.**

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

- [ ] **Step 2: Run them, watch them fail.**
- [ ] **Step 3: Give `writeRendered` the vocabulary and the surface.** It appends
      `wordRegions(text, v)` to the regions it was given, and colours the text
      OUTSIDE any embedded render — the range `marksIn` already locates, so the
      boundary is found rather than assumed.
- [ ] **Step 4: Derive the surface from the FORM, not the call site (PQ-4).**
      There is ONE prompt write site serving both `Choice` and `Cloze`, so the
      site cannot tell them apart. `surfaceOf(q)` switches on `q.Form()` — the
      NAME, a string; never a type switch on a concrete form, which `#6`'s
      Done-when forbids.
- [ ] **Step 5: Hang the classification on the extent that is already derived.**
      `TestEveryFormHasASurface` walks `docSyncForms`, whose completeness
      `TestEveryFormIsEnrolled` already guarantees by parsing `play/*.go`. So a
      new form cannot arrive unclassified, and no second list is created. Assert
      the guard fires: adding a form with no classification must redden it.
      *A hand-maintained extent is half a guard* — `#12` BR-17.
- [ ] **Step 6: Run the whole suite, plus `go vet` under all three tag sets.**
- [ ] **Step 7: Mutation sweep** — turn the surface rule off and confirm the
      cloze row reddens; drop `wordRegions` and confirm the clickability row
      reddens.
- [ ] **Step 8: Update `README.md` and `atlas/define.md`.** The key table and the
      click sentence both describe what a click does; the derived guards from
      `#12` will hold them to it.
- [ ] **Step 9: State M2's operating envelope (ARCH-CONSTRAINTS).** M1 carries
      one and M2 did not. `deckSpans` runs **once per write** — per question, per
      reveal, per lookup — never per keystroke: the sitting redraws from a buffer
      and the REPL's input line uses the streaming `highlightSpans` path it
      already had. The largest input is a full-entry reveal, a few hundred lines,
      walked once against a deck of at most a few thousand keys via a map lookup
      per token. That is the bound; if a future surface wants this per keystroke,
      that is a different design and should be a finding, not a silent regression.
- [ ] **Step 10: Commit, then `sdlc close --issue 46`.**

---

## Verification

Automated, and each row names the thing it would catch:

1. `go test ./...`, `go vet ./...` under default, `pty` and `conformance` tags,
   `gofmt -l` clean.
2. `TestASittingFetchesARecordingOnce` — the M1 regression at the loop.
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
