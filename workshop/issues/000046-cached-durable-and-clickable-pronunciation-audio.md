---
id: 000046
status: working
deps: []
github_issue:
created: 2026-09-07
updated: 2026-09-07
estimate_hours:
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

### A cloze's option words are clickable

Needs three things, and the first two are the real work:

- a third `RegionKind` — the registry is built for this ("a third consumer is a
  row rather than a new feature"), and `String`/`identifier`/the atlas guard and
  `TestEveryRegionKindIsActionable` all pick it up the moment it is declared.
- **regions produced for a PROMPT**, which nothing does today. `marksIn` locates
  `Render(entry)`'s text inside what is written and shifts the lines; a cloze
  prompt contains no rendered entry, so it gets nil. The form knows where its
  option lines are (`optionLine`, `Options()`), and `play` is mechanically
  guarded pure so it cannot hold a `Region` — the coordinates have to be built
  in main from the same string the form rendered. `clozeAsk` is where that
  happens, for the reason it already gives: one place renders.
- the screen wiring, which is the existing path once the regions exist.

**Not in scope:** `#45` (async playback). This is about what is fetched and what
is clickable, not about when the audio plays.

## Done when

- [ ] Every loop that can play audio goes through the cache, and a guard names
      that set rather than a person remembering it — `runPlay` is the one that
      does not today, and fixing only `runPlay` leaves the next loop free to
      repeat it.
- [ ] A recording fetched in one sitting is not fetched again in the next: the
      bytes are on disk, in the directory `define` was started from.
- [ ] A word the CDN has no recording for is asked for ONCE, not four candidate
      URLs per replay per day. Pinned through the existing `fakeCDN` request
      recorder, which is what makes "no second request" assertable without new
      scaffolding.
- [ ] `Forget` takes a word's recordings with it, and
      `TestPerWordDirsCoverEveryRuntimeDir` is what says so — the new directory
      is classified on both axes, not just declared.
- [ ] A cloze's option words are clickable and each plays its own word.
- [ ] The new region kind is a ROW, not a special case: `numRegionKinds` picks
      it up, `TestEveryRegionKindIsActionable`, `TestAtlasDescribesEveryRegionKind`
      and `TestAPromptRegionCoversTheTextItClaims` all exercise it without being
      edited to know about it.
- [ ] No region claims text it does not cover — the `#12` BR-14 invariant holds
      for the new kind, which is the one that puts regions on a PROMPT for the
      first time.

## Plan

Two boundaries, because the work splits cleanly at a seam and neither half needs
the other. Not three: the two cache gaps are one change to one type in one file,
and closing them separately would buy a redundant review (AGENTS.md §3).

- [ ] M1 — the cache reaches every loop that plays audio, and survives the
      process. `fetch.go`, a new `RuntimeDirs` entry, `perWordDirs`, `Forget`.
- [ ] M2 — a cloze's option words are clickable. A third `RegionKind`, and the
      first regions ever produced for a PROMPT.

## Log

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

`#45` (async playback) is adjacent and separate: this issue is about WHAT is
fetched and what is clickable, not about when the audio plays.
