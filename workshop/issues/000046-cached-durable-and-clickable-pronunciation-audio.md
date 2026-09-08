---
id: 000046
status: working
deps: []
github_issue:
created: 2026-09-07
updated: 2026-09-07
estimate_hours: 7.02
started: 2026-09-07T12:42:34-07:00
---

# cached, durable and clickable pronunciation audio

## Problem

Three gaps found by hand-running a sitting, all on the same axis: **the sound a
learner wants is either not reachable, or fetched again.**

1. **A sitting fetches every recording afresh.** `repl.go:257` and
   `replraw.go:264` each wrap `d.audio = newCachingAudioSource(d.audio)`.
   `runPlay` (`play_loop.go:24`) does not. So the one loop that plays the SAME
   handful of words over and over is the one loop with no cache — and a word the
   CDN has no recording for costs four candidate requests on every replay,
   because the `misses` set that exists to prevent exactly that is never
   constructed.

2. **The cache dies with the process.** `cachingAudioSource` already models both
   levels the learner asked for — `hits` (the bytes) and `misses` (whether a
   fetch is known to be pointless) — but in two maps that live as long as one
   command. A deck reviewed daily re-fetches the same recordings daily, and
   re-asks the same permanent 404s daily. The distinction the type documents
   ("a miss is PERMANENT — unlike a transport failure") is the argument for
   persisting it, and the type makes it without acting on it.

3. **A cloze's option words cannot be clicked.** Every other word on screen can:
   `Region` is a click registry, `playRegion` already fetches and plays, and
   `TestEveryRegionKindIsActionable` derives its loop from `numRegionKinds`. But
   regions are produced by `Render(entry)` alone, and `marksIn` finds them by
   locating that rendered entry inside the written text — so they attach to the
   REVEAL and never to a prompt. A cloze prompt is a blanked sentence and four
   words, none of which is the rendered entry; the four words a learner is
   choosing between are the four they most want to hear.

## Spec

**One sentence: the sound is fetched once, kept, and reachable from the words a
learner is actually looking at.**

### The cache reaches the sitting, and stops being a call-site decision

The bug is not the missing line — it is that "wrap the seam" is a thing each
loop remembers to do. `repl.go`'s comment gives the reason it lives there (#2's
I-1: the line tests exercise is the line production runs), which is a good
reason not to move it to `realDeps` and a bad reason to leave the set
unenumerated. Close it the way `numRegionKinds` and `TestEveryFormIsEnrolled`
close theirs: a guard that names every loop that can play audio and fails when
one of them reaches the CDN twice for the same word. Fixing only `runPlay`
leaves a fourth loop free to repeat it (ARCH-PURPOSE).

### The cache is durable, in the directory `define` was started from

Two levels, and they are the two the type already has:

- **the verdict** — whether a fetch is known to be pointless. Permanent, cheap,
  and the one that saves four requests per replay of an unrecorded word.
- **the bytes** — the recording itself, so a replayed word costs no network.

It lands under the store's working directory, a new `RuntimeDirs` entry
appended AT THE TAIL (`yaml.go`:57's positional-index rule). Audio is keyed by
`AudioCandidates(word, voice)`, so it is per-word and language-scoped — a
`perWordDir{scoped: true}` — and `#10`'s `TestPerWordDirsCoverEveryRuntimeDir`
will fail until it is classified. That is the guard working, and the property it
buys is the one `#10` fought for: **`Forget` takes a word's recordings with it**,
exactly as it now takes its facts and its items.

Open questions for the plan, not decided here:

- eviction, or none. A recording is ~10-30KB and a deck is hundreds of words, so
  "none" may be the right answer for the MVP — but it should be a decision with
  a number attached, not an omission.
- whether a stale verdict expires. The CDN gains recordings over time, so a
  `miss` cached forever is a word that can never start working. A dated verdict
  with a long floor is the likely shape.
- the on-disk form: the bytes are not YAML, and every other runtime dir is. And
  a word can have more than one recording within one language, because `voice`
  carries a Locale as well as a Lang — so "one file per word" may not hold, which
  is the assumption `perWordDir`'s filename derivation currently makes.

### Every deck word is clickable, and coloured where colour means something

**Widened 2026-09-07 by the operator, and the wider version is the simpler one**
— see the revision below. The ask is not "cloze options are clickable" but
*click to pronounce and colour for the words I am learning, in all places*, with
colour off in the cloze.

**The unification.** A deck word in rendered text is ONE span that offers TWO
things: a colour and a click. Today two independent producers find that span and
they cover different surfaces, which is the whole bug:

| | finds spans by | reaches |
|---|---|---|
| colour | `highlightSpans` / `highlightWriter`, escape-aware, longest-phrase-wins | the lookup entry, the REPL input line, `--ask` answers, and the entry embedded in a sitting's reveal |
| clicks | `regionsIn`, escape-aware via `findVisible` | the headword and the ORIGIN languages of a rendered entry, plus `promptRegions`' headword |

Neither reaches a form's own text — the option lines, the restored sentence, the
`you chose` line — so the words a learner is actively choosing between are the
words they can neither hear nor see marked.

So: **one span producer, two consumers.** `highlightSpans` already computes
exactly the right set (deck words, longest phrase wins) and `findVisible`
already locates a span in text that is already coloured. What is missing is the
`Region` half of the same walk, and one write door that applies both.

- a third `RegionKind` — the registry is built for this ("a third consumer is a
  row rather than a new feature"), and `String`/`identifier`/the atlas guard and
  `TestEveryRegionKindIsActionable` all pick it up the moment it is declared.
  Distinct from `RegionHeadword`, which carries the entry's LOOKUP KEY and may
  differ from the text under it (`define jalapeno` renders `jalapeño`); a deck
  word in prose is matched by its own text.
- **regions produced for a PROMPT**, which nothing does today. `marksIn` locates
  `Render(entry)`'s text inside what is written and shifts the lines; a cloze
  prompt contains no rendered entry, so it gets nil. `play` is mechanically
  guarded pure and cannot hold a `Region`, so the coordinates are built in main
  from the same string the form rendered — one place renders.
- the screen wiring, which is the existing path once the regions exist.

### Colour is a DISCOVERY signal, so it is off where everything is a deck word

The operator's rule, and it generalises past the case that prompted it: *no
colour in the cloze, because everything there is a new word and colour would
clog things up.*

Stated as a property rather than an exception: **colour marks a deck word inside
PROSE, where finding one is a discovery. Where the text IS the deck, colour
marks everything and therefore distinguishes nothing.** That covers two surfaces
with one rule instead of two special cases:

- a **cloze's option words** — four words drawn from the banded deck;
- a **board's cells** — every cell is a deck word by construction.

And it leaves colour ON where it earns its place: a `Choice`'s option glosses
are definitions, i.e. prose in which a known word is worth spotting; so are the
reveal's restored sentence and every rendered entry.

**Clicks are NOT subject to this rule.** A click is per-word and costs the
learner nothing when it is everywhere; colour is a field the eye reads at once
and degrades when it is everywhere. Cloze options are the case the operator
asked for FIRST: clickable, uncoloured.

**Not in scope:** `#45` (async playback). This is about what is fetched and what
is clickable, not about when the audio plays.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.* The calibration doc is tagged **stale** by
`sdlc estimate-source`, so the per-primitive hours are provisional; they are
derived against `#10` and `#12`, the two most recent closes in this repo, whose
blocks use the same primitives on the same codebase.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.60 impl=0.10
item: cross-cutting-refactor   design=0.06 impl=0.24
item: smaller-go-module        design=0.02 impl=0.12
item: greenfield-go-module     design=0.06 impl=0.28
item: cross-cutting-refactor   design=0.04 impl=0.20
item: smaller-go-module        design=0.02 impl=0.14
item: greenfield-go-module     design=0.05 impl=0.24
item: smaller-go-module        design=0.02 impl=0.12
item: milestone-review         design=0.0  impl=0.45
item: milestone-review         design=0.0  impl=0.60
item: greenfield-go-module     design=0.06 impl=0.28
item: smaller-go-module        design=0.03 impl=0.14
item: smaller-go-module        design=0.04 impl=0.14
item: smaller-go-module        design=0.03 impl=0.14
item: cross-cutting-refactor   design=0.05 impl=0.24
item: atlas-docs               design=0.03 impl=0.06
item: smaller-go-module        design=0.0  impl=0.20
item: smaller-go-module        design=0.0  impl=0.20
item: smaller-go-module        design=0.0  impl=0.20
item: ux-rename-iteration      design=0.0  impl=0.20
item: milestone-review         design=0.0  impl=0.60
item: milestone-review         design=0.0  impl=0.85
design-buffer: 0.15
total: 7.02
```

**What each row is**, in plan order, so the derivation is checkable rather than
asserted:

| row | the work |
|---|---|
| `issue-spec` 0.60/0.10 | the design carrier: the Spec, an operator scope widening mid-plan, and FOUR plan-quality rounds. **Above** `#10`'s 0.50 — 4 rounds against its 2, an 861-line plan against its 693, and a widening neither neighbour had. |
| `cross-cutting-refactor` 0.06/0.24 | `deps.audio` becomes `*audioSeam`. Mechanical but wide: the compiler names every construction site, including the test literals. |
| `smaller-go-module` 0.02/0.12 | the memo logic relocated into `audioSeam` — existing code, existing tests, new home. |
| `greenfield-go-module` 0.06/0.28 | `store/audio.go`: `audioKey`'s digest, `audioRecord`, the TTL. |
| `cross-cutting-refactor` 0.04/0.20 | `RuntimeDirs`' four consumers — `perWordDirs` prefix glob, the `Store` interface, the `Mem` twin, `.gitignore`. |
| `smaller-go-module` 0.02/0.14 | the `storetest` conformance rows, so `Mem` and YAML are both held. |
| `greenfield-go-module` 0.05/0.24 | `diskAudioCache`. |
| `smaller-go-module` 0.02/0.12 | the degrade-never-fail paths and the `--forget` end-to-end. |
| `milestone-review` 0.0/0.45 + 0.0/0.60 | **M1's boundary, priced as run + remediation** — the house convention `#12` and `#42` set, which the first draft misread as one row per boundary. Below the close's pair, because M1 changes one seam where M2 changes what every write site does. |
| `greenfield-go-module` 0.06/0.28 | `deckSpans` — the coordinate layer, and the escape-awareness that is the design's load-bearing row. |
| `smaller-go-module` 0.03/0.14 | `RegionWord`: the kind, `playRegion`'s row, `String`/`identifier`, the atlas. |
| `smaller-go-module` 0.04/0.14 | `mergeRegions` and the disjointness property. A sort-and-drop helper plus one property test — `#12`'s `Flagging` shape, not `diskAudioCache`'s; the first draft priced it as greenfield. |
| `smaller-go-module` 0.03/0.14 | `surface`/`surfaceOf` and `TestEveryFormHasASurface`. |
| `cross-cutting-refactor` 0.05/0.24 | the write door: three call sites, and colouring outside the embedded render. |
| `atlas-docs` 0.03/0.06 | README's key table and click sentence; the atlas. |
| three × `smaller-go-module` 0.0/0.20 | **three DISTINCT mutation sweeps**, not one: revert `deckSpans`' escape-awareness; append an overlapping region and confirm the golden loses an underline; turn the surface rule off. `#12` booked 0.20 for a single sweep. |
| `ux-rename-iteration` 0.0/0.20 | **the pty hand-run.** M2 is entirely about what a person SEES — colour on or off, which words are click targets — and this issue was born from exactly such a sitting. Both neighbours added this row on review; `#12`'s deviation 3 records that pricing it at nothing is the omission `#10` had already made. |
| `milestone-review` 0.0/0.60 + 0.0/0.85 | the close boundary, run + remediation. This is `#12`'s pair for one boundary, which is the right comparable for M2's. |

**Reconciliation.** Σdesign = 1.11, Σimpl = 5.74.
1.11 × 1.15 + 5.74 × 1.0 = **7.02**.

**Trailing ledger read**, which is better evidence than a bracket between two
neighbours. This repo's rows: `#42` 0.36, `#44` 0.47, `#10` 0.62, `#12` 1.23 —
median ≈ 0.55, which at 7.02 would predict roughly 13h actual. **That gap is not
a reason to inflate the primitives.** Per the model's own unit note it is the
within-session parallelism and overlap that `#117`'s ledger exists to instrument,
and multiplying the rows to meet it would destroy the only signal the ledger
carries. The estimate is the derivation; the ratio is the measurement; the two
are supposed to differ and be recorded.

**Deviations from the neighbours, named rather than absorbed:**

1. **`issue-spec` above `#10`'s**, where both neighbours' rows sat at or below
   0.50. Four plan-quality rounds, three of them re-answering ONE finding, plus a
   mid-plan scope widening. This row is partly retrospective — the rounds are in
   `git log` — so it is checkable rather than predicted.
2. **Two `milestone-review` PAIRS**, one per boundary. The first draft booked one
   row per boundary and cited `#12`'s 0.85 as a single-boundary comparable; 0.85
   was `#12`'s remediation half, with 0.60 for the run.
3. **Three sweep rows.** The plan commits to three distinct mutation sweeps and
   the first draft priced them as one.

## Done when

- [x] No loop can play audio through an uncached source. **Reworded 2026-09-07:
      the row asked for "a guard that names that set", and four plan-gate rounds
      established that no such set is derivable — every predicate over functions
      caught dispatchers or missed a wrap site. The shipped design makes the
      question unaskable instead: `deps.audio` is a `*audioSeam`, so there is no
      unwrapped source to hold and the compiler enumerates the construction
      sites.** Ticking the row as written would have claimed a guard that does
      not exist and should not.
- [x] A recording fetched in one sitting is not fetched again in the next: the
      bytes are on disk, in the directory `define` was started from.
- [x] A word the CDN has no recording for is asked for ONCE, not four candidate
      URLs per replay per day. Pinned through the existing `fakeCDN` request
      recorder, which is what makes "no second request" assertable without new
      scaffolding.
- [x] `Forget` takes a word's recordings with it, on every voice and from every
      language. **Reworded 2026-09-07: "both axes" is now THREE** — the boundary
      review added `many` (a word owns several files here) after finding that a
      declared axis nothing checks is decoration, and found two bugs the axes
      caught: a language shelf that stranded recordings across `/lang`, and a
      `<slug>--` prefix glob that took `re-`'s recordings when forgetting `re`.
- [x] A cloze's option words are clickable and each plays its own word.
- [x] Every deck word is clickable wherever it is written, on every surface the
      write door serves — `TestEveryDeckWordInASittingIsClickable` drives it
      through `writeWords` rather than through the rule, so it fails if a call
      site stops passing the vocabulary. **The row asked for a guard over write
      SITES and that is NOT what shipped:** the derived guard is over FORMS
      (`TestEveryFormHasASurface`, hung on `docSyncForms`). A new `writeWords`
      call site is covered by review, not by construction — carried to `#30`'s
      region work rather than claimed here.
- [x] Colour and clicks come from ONE span walk, not two producers that can
      disagree about where a word is.
- [x] Colour is off exactly where the text IS the deck (cloze options, board
      cells) and on everywhere else, and the guard states that as the rule
      rather than naming the two surfaces.
- [x] The new region kind is a ROW, not a special case: `numRegionKinds` picks
      it up, `TestEveryRegionKindIsActionable`, `TestAtlasDescribesEveryRegionKind`
      and `TestAPromptRegionCoversTheTextItClaims` all exercise it without being
      edited to know about it.
- [x] No region claims text it does not cover — the `#12` BR-14 invariant holds
      for the new kind, which is the one that puts regions on a PROMPT for the
      first time, and holds on text that is ALREADY coloured.
- [x] A span walk over text carrying ANSI never lands a region inside an escape
      sequence, and never nests colour.

## Plan

Two boundaries, because the work splits cleanly at a seam and neither half needs
the other. Not three: the two cache gaps are one change to one type in one file,
and closing them separately would buy a redundant review (AGENTS.md §3).

Durable design: `workshop/plans/000046-audio-cache-and-deck-words-plan.md`.

- [x] M1 — the cache reaches every loop that plays audio, and survives the
      process. `fetch.go`, a new `RuntimeDirs` entry, `perWordDirs`, `Forget`.
- [x] M2 — one span walk feeds both colour and clicks, and every surface goes
      through it. A third `RegionKind`, the first regions ever produced for a
      PROMPT, and the discovery rule that turns colour off where the text is
      the deck.

## Log

### 2026-09-07 — built through both milestones, then smoke-tested
- 2026-09-07: closed M1 — M1 ships after two REWORK rounds (round 3 did not run — the reviewer failed to authenticate, OAuth expiry, not a finding). The memo IS the seam: deps.audio is a *audioSeam, so runPlay needed no wrap line and the compiler enumerated the construction sites. Pinned: TestOneWordCostsOneFetchHoweverOftenItIsPlayed (the memo) and TestWithStorePutsTheDiskCacheUnderTheMemo (the WIRING — round 1 found deleting the whole withStore block left the suite green; deleting it now reddens by name). Durable: TestASecondRunReusesTheRecordingOnDisk (two fresh seams over one directory = one CDN request, and from survives so spokeSource/reportVoice stay true), stale-verdict re-ask, outage-not-recorded, PlaybackSurvivesAnUnusableCache driven against failingStore. Round 2 fixes: audio/ is FLAT (a language shelf stranded recordings across /lang while --forget reported success, pinned by TestForgetTakesARecordingFetchedInAnotherLanguage); a word owns a DIRECTORY not a filename prefix (the <slug>--<digest> scheme rested on the false claim that a slug cannot contain "--" — re- slugs to re--ddf427, so forgetting re took its recordings; storetest drives that exact pair on both twins and the prefix scheme reddens it); perWordDir carries THREE axes, each added by a bug the previous set could not see; the blob read is capped at 4MB like the network path; a verdict REMOVES the recording it supersedes, pinned through the FILESYSTEM because asserting through Audio cannot see the stale file (mutation-verified: reverting the removal reddens it by name). TestAudioDegradesOnEveryDamagedFile drives all four branches the "degrades, never fails" promise names — corrupt record, vanished blob, unwritable path, oversized blob — two of which previously carried a promise nothing checked. BR-16 swept: the filing reversal had left ten restatements of the superseded scheme; the grep that finds them is now in lessons.md with the rule to write it before the fix, and the derived consumer (.gitignore from RuntimeDirs) was right untouched while every hand-maintained one was wrong. Plan revised by APPENDED Revisions entry per AGENTS.md §1, all 45 step checkboxes ticked, entity table names match the code. go test -count=1 ./... green; go vet clean under default, pty and conformance; gofmt clean. ACTUAL OMITTED DELIBERATELY: M1 and M2 both landed before any boundary review (the operator asked for a runnable sitting first), so this window is not separable and sdlc actual has no sub-range flag; hand-splitting would be the guessed value the gate exists to prevent. The issue close measures and adopts the real figure.; review verdict: FIX-THEN-SHIP

**Both milestones landed before any boundary review**, because the operator asked
for a runnable sitting ("go ahead till I can smoke test"). So M1's boundary
review sees M2's diff as well. Recorded rather than hidden: the review is wider
than the milestone, and the close's delta is correspondingly small.

**The manual verification the plan commits to (steps 8-10) is done, by the
operator, on a generated deck** — `scratchpad/smoke`, ten words and five authored
cloze items written THROUGH the store package rather than by hand, with the word
list chosen so the definitions cross-reference each other (`sycophantic`'s NOAD
gloss reads "behaving in an *obsequious* way", and `obsequious` is in the deck).
Verdict: **passed**.

Independently confirmed over a pty before handing it over, which is what makes
the two halves of the operator's rule checkable rather than asserted:

- the cloze frame carries **four underlined option words and no `knownOn`
  escape anywhere in it** — clickable, uncoloured;
- the reveal frame **does** carry `knownOn`, on a deck word inside the gloss.

**Three things changed during implementation, all recorded in the plan's
Revisions or below:**

1. **The disk cache did nothing at all in its first version, silently.**
   `forWord` returned `*diskAudioCache` where `wordFiler` wanted `AudioSource` —
   a signature Go accepts everywhere except as an implementation of that
   interface. The assertion never matched, every fetch bypassed the disk, and it
   compiled and ran cleanly. Only the request-count assertions caught it. A
   `var _ wordFiler = (*diskAudioCache)(nil)` now stands beside it, and
   `lessons.md` carries the rule: the failure mode of an optional-capability
   interface is SILENCE.
2. **Clicks were about to depend on colour.** `vocabularyFor` returns nil when
   colour is off — correct for its own caller, since loading the deck to inject
   an invisible style is IO for a disabled feature. Taking the click vocabulary
   from the same place would have made `-no-color` silently remove every click
   target. Split into `deckVocabulary`, pinned by
   `TestClicksSurviveNoColour`.
3. **The lookup path was wired too**, not only the sitting. "All places" includes
   `define <word>`, where a click previously reached only the headword and the
   ORIGIN languages.

### 2026-09-07 — filed

Three gaps found by hand-running `--play` at the end of `#12`, kept as one issue
because they are one complaint: the sound is either fetched again or not
reachable.

`#12`'s BR-14 fix is the groundwork for M2 and should be read before starting
it: `promptRegions` is now the one place a prompt declares what it offers, and
`TestAPromptRegionCoversTheTextItClaims` will hold the new kind to the same
invariant that Critical was about — a region must not claim text it does not
cover. Producing regions for the option lines is exactly the case that made the
old formula dangerous, so it is the case the guard was built for.

### 2026-09-07 — widened: click and colour everywhere, not just cloze options

**Reason.** The operator, on picking the issue up: *"generally speaking I'd like
the click to pronounce and coloring of words I'm learning in all places. note no
need for coloring in cloze because everything there is 'new words', thus having
color would clog things up."*

**Delta.** M2 was "a cloze's option words are clickable" — one surface, one
form. It is now the general property, and the general version is SIMPLER to
build than the special case would have been: the special case needed a bespoke
span producer for cloze option lines, and the general one needs the span
producer that already exists to grow a second consumer. `#12`'s BR-14 fix is
what makes this affordable — `promptRegions` established that a prompt declares
what it offers and that a region must cover the text it claims.

**What the operator's colour exception bought.** Read narrowly it is "skip the
cloze". Read as a property it is *colour is a discovery signal*, which also
explains the board — every cell there is a deck word too — and predicts the
answer for any surface added later. That reading is in the Spec above; the
narrow one would have been a flag.

`#45` (async playback) is adjacent and separate: this issue is about WHAT is
fetched and what is clickable, not about when the audio plays.
