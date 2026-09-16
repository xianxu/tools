---
id: 000067
status: working
deps: []
github_issue:
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
started: 2026-09-16T12:50:20-07:00
---

# define: read-along — paste a passage, click or drag what is opaque

## Problem

The operator's use of `define` has outgrown the word as the unit. Reading
astronomical prose they looked up `zenith`, `nadir`, `meridian`, then asked
*"what's progression in this context"* — a question about the passage, which
`define` cannot see.

The answer (observed 2026-09-16) opens with *"I can't see the sentence you're
reading, so this is a best guess from what's around it"* and then spends three of
four paragraphs hedging across an astronomical reading, an ordinary-use reading,
and a guess keyed off the word on screen. One line of the actual source would
collapse all of it into a single confident sentence.

The session's lookups are only a SHADOW of the passage. `askContext` carries the
deck, the learner model, the current entry and this session's words — everything
except the text that prompted the question.

Two further observations from the same answer:

- It highlighted four deck words (`equinox`, `synodic`, `granulation`,
  `epithelium`) inside its own prose, because `highlightWriter` wraps the answer
  stream (atlas §Highlighting, M2/M3). Each is an incidental retrieval event.
  Explanation-in-context is ALREADY a reinforcement surface; it is simply aimed
  at the wrong context.
- It flagged `progression` vs `precession` as a probable misreading. Sense and
  form disambiguation against a real sentence is the thing a dictionary app
  structurally cannot do, and it is the value this issue is for.

## Spec

Draft from the operator's request (2026-09-16); **needs a brainstorm before a
plan.**

**The gesture.** Paste a passage into `define`. It stays on screen. Click a word,
or drag across a phrase, to mark it opaque. `define` explains the marked spans in
the context of that passage.

**Selections are a SET, answered together.** N marked spans in one passage
produce ONE model call, not N. The relations among the marked words are most of
what a reader is missing, and per-span glosses discard exactly that — the same
finding that runs through this whole line of work: the unit is the structure, not
the node.

**The dictionary is the admission gate to the deck.** Every explained span is run
through the dictionary — the classifier `define` already uses for free-form input
(atlas §Free-form input, *"the dictionary is the classifier"*). A hit
(`progression`, `a priori`, `hot dog`) is deck-eligible and enters later recall;
a miss — an arbitrary dragged phrase — is explained and not retained. Operator's
rule, 2026-09-16. Multi-word headwords already work as deck keys: `highlightSpans`
takes longest-match-wins so `hot dog` highlights as a phrase.

**NOAD's role inverts, and that should be deliberate.** Today the entry is the
output and the model supplements it (`askSystem`: *"it is authoritative and you
are not"*). Here the entries of the marked words become CONTEXT and the
contextual explanation is the output, with the full entry one gesture away. A
stance change to state, not to drift into.

**A marked word is stronger deck signal than a typed lookup.** A typed lookup is
ambiguous — curiosity, a spelling check, verification. A word marked because it
blocked a reading is unambiguous, and it carries its sentence as provenance.
Feeds #22 (active learning set) and #17 (the learner model).

### What already exists (survey, 2026-09-16)

The build is much smaller than the feature sounds. Almost every part is a ROW in
a registry that was built anticipating a third consumer (ARCH-DRY):

| capability | where | status |
|---|---|---|
| drag-select a span | `selection.go` — `selectionGesture`, press/motion/release/cancel | built; effects `selectionCopy` today (`selection.go:72`) |
| click a span | same state machine | built; effects `selectionClick` (`selection.go:74`) |
| mouse on the wire | `rawterm.go:131` `\x1b[?1002h\x1b[?1006h` | built — 1002 is button-event motion, i.e. held drags, and the comment already says they "belong to the shared selection router" |
| drag vs click discrimination | `key.go:236` `isClickButton` | built — motion bit 5 is already named *"a selection gesture rather than a click"* |
| actionable spans over rendered text | `Region{Kind,Text,Word,Line,Col,Width}`, `regionsIn`, `screen.addRegions`/`RegionAt`, `markClickable` | built; atlas says *"a third consumer is a row rather than a new feature"* |
| word boundaries | `wordRuns` (the single tokenizer), `highlightSpans` | built, fuzz-defended: concatenating spans reproduces the input |
| "is this span a headword" | NOAD-first classification | built, reusable as the deck admission gate |
| an answer with context | `askContext` / `renderAskPrompt` (pure) | built; needs a passage field |

**What is genuinely missing:**

1. **Bracketed paste.** Mode `2004` is never enabled and nothing accumulates
   between `ESC[200~` and `ESC[201~`. The start marker is only consumed correctly
   by the generic CSI scanner (`key_test.go:71`). Today a pasted sentence arrives
   as loose keystrokes and its first newline submits the line. This is the one
   real build.
2. **A surface that pins the passage** while explanations accumulate, so marking
   a second word does not require scrolling back.
3. **A third `selectionEffect`** (explain) beside `selectionClick`/`selectionCopy`,
   and a region kind for a word inside a passage.
4. **The set-wise contextual answer** — one request carrying the passage, the
   marked spans and their dictionary entries.
5. **Deck admission** on a dictionary hit, with an event kind that distinguishes
   a marked word from a typed lookup.

### Open questions for the brainstorm

- Does a marked span explain on release, or does the reader mark several and then
  ask? (Set-wise answering argues for the second; immediacy argues for the first.)
- Word regions must survive wrapping — same class as `phraseGap`, which already
  stops a wrapped `hot\n  dog` forming a false phrase. Reuse or extend?
- A dragged phrase with no dictionary entry is still learnable (`at the zenith
  of`, idioms, constructions). The operator's rule keeps them out of the deck;
  is that right permanently, or right for M1?
- **Click density is a free readability measure** — four opaque words in a
  twenty-word sentence vs one in forty is a calibrated difficulty reading of real
  text against this learner, far better evidence than the current CEFR guess.
  `define` has never had this. In scope, or its own issue?
- Where does the passage live — session-only, or on disk like every other context
  source? (Compare #56, where the transcript is the one in-memory exception.)
- Does this compose with `/bilingual` and #64's interaction stage?

## Done when

- A pasted passage stays on screen and its words are individually clickable;
  dragging selects a phrase across word boundaries.
- Marking N spans produces ONE explanation that resolves them against the
  passage, asserted through the LLM fake by reading the real request.
- An explained span with a dictionary entry enters the deck and appears in a
  later recall exercise; one without is explained and not retained. Both
  directions covered.
- A marked word is distinguishable from a typed lookup in the event log.
- Pasting multi-line text does not submit on the embedded newline.
- The NOAD-as-context inversion is stated in the atlas, with the route back to
  the full entry.

## Plan

Awaiting brainstorm, then a durable plan in `workshop/plans/`.

- [ ] brainstorm the open questions above
- [ ] `sdlc start-plan`, then write the plan

## Log

### 2026-09-16

Filed from a design conversation that walked words → concept → structure. Two
framings were tried and discarded on the way, both worth not re-deriving:

- **Longer conversational memory (#56).** The operator explicitly does not want a
  chat replacement — *"keep it lean and short... slightly more surrounding
  information than a dictionary app."* Persisting the transcript is a different
  need; it does not serve this one.
- **"Phrase" as the expansion.** A phrase is still a headword (`hot dog`,
  `a priori`) and `define` already looks them up. The real expansion is from node
  to STRUCTURE — the edges between words are what makes a neighborhood cheap to
  learn. Drag-select serves that; it is not the phrase-as-headword case.

Also discarded: piping the harvested `Domain` into `askContext`. Bare session
words were already enough for the model to infer "astronomical prose" unaided, so
that would solve a problem not currently hurting.
