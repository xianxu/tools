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
  - **Relationships:** a property of a WRITE SITE, not of a form. `play` never
    sees it; `play` is pure and holds no presentation.

- **`audioName` / `audioVerdict`** — the on-disk shape of one word's audio: the
  recording, or a dated record that the CDN has none.
  - **Why the verdict is dated.** A miss cached forever is a word that can never
    start working, and the CDN gains recordings over time. A verdict carries the
    day it was reached and is re-asked after `audioVerdictTTL` (30 days). The
    in-memory `misses` set needs no TTL because it dies with the process; the
    durable one does, and that difference is the reason this is a new type rather
    than the old map serialised.

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

### Task 1: One construction site for the audio source

**Files:**
- Modify: `cmd/define/main.go` (`realDeps`)
- Modify: `cmd/define/repl.go:257`, `cmd/define/replraw.go:264` (remove the wraps)
- Test: `cmd/define/fetch_test.go`

- [ ] **Step 1: Write the failing guard** — the property is "no loop can hold an
      uncached source", and the way to state that is that the raw source is
      constructed once and immediately wrapped.

```go
// THE WRAP IS STRUCTURAL, NOT REMEMBERED (#46).
//
// It used to be a line each loop wrote for itself, and `runPlay` — the loop that
// replays the same handful of words all sitting — never wrote it. So a word with
// no recording cost four candidate requests on EVERY replay, because the misses
// set that exists to prevent exactly that was never constructed.
//
// Fixing runPlay fixes the site. This fixes the class: the raw source is built
// in one place and wrapped there, so there is no unwrapped source for a loop to
// hold, and no enumeration of loops for anyone to keep current.
func TestTheRawAudioSourceIsConstructedOnceAndWrapped(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	var sites []string
	for _, pkg := range pkgs {
		for path, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "newHTTPAudioSource" {
					sites = append(sites, fmt.Sprintf("%s:%d", path, fset.Position(call.Pos()).Line))
				}
				return true
			})
		}
	}
	if len(sites) != 1 {
		t.Errorf("newHTTPAudioSource is constructed at %d sites %v; want exactly 1.\n"+
			"A second construction is a source that reaches a loop unwrapped, "+
			"which is how --play spent a sitting re-fetching every recording.", len(sites), sites)
	}
}

// And the thing realDeps hands out is the wrapped one.
func TestRealDepsCarriesACachingAudioSource(t *testing.T) {
	if _, ok := realDeps().audio.(*cachingAudioSource); !ok {
		t.Errorf("realDeps().audio is %T, not a caching source", realDeps().audio)
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define/ -run 'RawAudioSource|RealDepsCarries' -v`
Expected: FAIL — `realDeps().audio` is `*httpAudioSource`.

- [ ] **Step 3: Move the wrap into `realDeps`**

```go
// audio is wrapped HERE and nowhere else.
//
// It used to be wrapped by each loop, with repl.go carrying #2's I-1 reason —
// "so the line the tests exercise is the line production runs". That reason is
// preserved and strengthened: a test driving any loop through realDeps now gets
// the same source production gets, and a loop that forgets to wrap is not a
// thing that can exist.
audio: newCachingAudioSource(newHTTPAudioSource()),
```

Delete the wrap lines at `repl.go:257` and `replraw.go:264`, keeping their
comments' reasoning in the new site.

- [ ] **Step 4: Run the guards and the existing fetch suite**

Run: `go test ./cmd/define/ -run 'AudioSource|RealDeps|Caching' -v`
Expected: PASS.

- [ ] **Step 5: Prove the sitting is actually cached now, through the CDN recorder**

```go
// The regression itself, stated at the loop rather than at the seam: a sitting
// that shows the same word twice fetches it once.
func TestASittingFetchesARecordingOnce(t *testing.T) { /* fakeCDN + runPlay over a two-question deck */ }
```

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "#46 M1: the audio wrap is structural, not remembered"
```

### Task 2: A durable home for the recording and the verdict

**Files:**
- Create: `cmd/define/store/audio.go`, `cmd/define/store/audio_test.go`
- Modify: `cmd/define/store/yaml.go` (`RuntimeDirs`, `audioDir`, `perWordDirs`)
- Modify: `cmd/define/store/mem.go` (the in-memory twin)
- Modify: `cmd/define/store/storetest/suite.go` (the conformance rows)

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

- [ ] **Step 4: The verdict type, with the TTL as a named constant**

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

- [ ] **Step 5: Conformance rows in `storetest`, so Mem and YAML are both held**

Rows: a recording round-trips; a verdict round-trips with its date; a verdict
older than the TTL reads as absent; `Forget` takes both.

- [ ] **Step 6: Commit**

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
- [ ] **Step 5: Commit.**

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
func TestBoardCellsAreNotColoured(t *testing.T)

// And nothing is coloured twice: Render owns the entry's colour, because it
// colours with a per-region BASE and ANSI does not nest.
func TestTheRenderedEntryIsNotRecoloured(t *testing.T)
```

- [ ] **Step 2: Run them, watch them fail.**
- [ ] **Step 3: Give `writeRendered` the vocabulary and the surface.** It appends
      `wordRegions(text, v)` to the regions it was given, and colours the text
      OUTSIDE any embedded render — the range `marksIn` already locates, so the
      boundary is found rather than assumed.
- [ ] **Step 4: Set the surface at each call site.** `surfaceDeck` for a cloze
      prompt and a board; `surfaceProse` everywhere else.
- [ ] **Step 5: The derived guard, so a new write site is covered by
      construction.** Walk the surfaces the way `docSyncForms` walks the forms:
      every `writeRendered` call site in production must pass a vocabulary and a
      surface, checked by parsing rather than by a list. *A hand-maintained
      extent is half a guard* — `#12` BR-17.
- [ ] **Step 6: Run the whole suite, plus `go vet` under all three tag sets.**
- [ ] **Step 7: Mutation sweep** — turn the surface rule off and confirm the
      cloze row reddens; drop `wordRegions` and confirm the clickability row
      reddens.
- [ ] **Step 8: Update `README.md` and `atlas/define.md`.** The key table and the
      click sentence both describe what a click does; the derived guards from
      `#12` will hold them to it.
- [ ] **Step 9: Commit, then `sdlc close --issue 46`.**

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
