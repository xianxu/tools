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
- [ ] Every deck word is clickable wherever it is written — a guard walks the
      surfaces rather than a person listing them, so a new write site is covered
      by construction.
- [ ] Colour and clicks come from ONE span walk, not two producers that can
      disagree about where a word is.
- [ ] Colour is off exactly where the text IS the deck (cloze options, board
      cells) and on everywhere else, and the guard states that as the rule
      rather than naming the two surfaces.
- [ ] The new region kind is a ROW, not a special case: `numRegionKinds` picks
      it up, `TestEveryRegionKindIsActionable`, `TestAtlasDescribesEveryRegionKind`
      and `TestAPromptRegionCoversTheTextItClaims` all exercise it without being
      edited to know about it.
- [ ] No region claims text it does not cover — the `#12` BR-14 invariant holds
      for the new kind, which is the one that puts regions on a PROMPT for the
      first time, and holds on text that is ALREADY coloured.
- [ ] A span walk over text carrying ANSI never lands a region inside an escape
      sequence, and never nests colour.

## Plan

Two boundaries, because the work splits cleanly at a seam and neither half needs
the other. Not three: the two cache gaps are one change to one type in one file,
and closing them separately would buy a redundant review (AGENTS.md §3).

Durable design: `workshop/plans/000046-audio-cache-and-deck-words-plan.md`.

- [ ] M1 — the cache reaches every loop that plays audio, and survives the
      process. `fetch.go`, a new `RuntimeDirs` entry, `perWordDirs`, `Forget`.
- [ ] M2 — one span walk feeds both colour and clicks, and every surface goes
      through it. A third `RegionKind`, the first regions ever produced for a
      PROMPT, and the discovery rule that turns colour off where the text is
      the deck.

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
