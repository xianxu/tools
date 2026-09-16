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
or drag across a phrase, to mark it opaque. Marks ACCUMULATE. When the reader
asks, `define` explains the passage with the marked spans called out.

**A click is a one-word drag** (operator, 2026-09-16). It exists so selecting a
single word is easy, not so it means something different — both gestures produce
a SPAN, differing only in how the span is derived: word-snapped via `wordRuns`
for a click, anchor→end for a drag. One path, not two (ARCH-DRY).

**Marks persist and are visible as a set.** A second selection does not replace
the first: earlier marks stay on screen in a distinct treatment (operator
proposes a blue background) so the reader can see what is currently being asked
about. Selecting is therefore a TOGGLE over a set — clicking a marked word
unmarks it — not an append-only list.

**The request is the whole sentence with its marks inline** (operator,
2026-09-16). The reader's actual question is:

> what does this mean: "this is a whole sentence about [some topic] that user
> doesn't [understand]"

So the model answers BOTH — the passage as a whole, AND each bracketed span. This
settles the set-vs-per-span question: you cannot answer "what does this sentence
mean" N times. ONE call, one passage, marks carried inline.

Inline markers rather than a passage plus a list of spans, because position is
then unambiguous: a word occurring twice in the passage needs no occurrence
index, the bracket is already at the right one.

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

- ~~Explain on release, or mark several then ask?~~ **Settled 2026-09-16:**
  accumulate, then ask — the request is the whole sentence with marks in it.
- ~~What is the ask gesture?~~ **Settled 2026-09-16: a bare Enter.** See below.
- Still open: a blank Enter with a passage on screen but ZERO marks — explain the
  whole passage, or fall through to replay? (Operator's call.)
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
- What marker does a selection use? **Not a bare `[...]`** — see the collision
  below. And what escapes a literal bracket already in the passage?
- Does an explained passage become the example sentence for the deck items its
  marks produce? (See "an authentic sentence" below.)

### Three collisions the design has to answer

1. **The bracket grammar is already taken.** `askSystem` reserves
   `[lang=es]…[/lang]` in the ANSWER direction and instructs the model to escape
   literal brackets as `&#91;`/`&#93;` (`askctx.go:153`). Introducing a second,
   different meaning for `[...]` in the PROMPT direction invites the model to
   confuse the two or echo them back. The selection marker must be chosen against
   that existing grammar, and the passage's own literal brackets need an escaping
   rule in the prompt direction too — which today only exists for the answer.

2. **Two decorations on one token, and ANSI does not nest.** A marked word may
   ALSO be a deck word — exactly what the 2026-09-16 screenshot shows, where
   `equinox` and `synodic` came back green. Blue background plus green foreground
   have to compose on the same token, and the atlas is explicit about why that is
   hard: *"`RenderLine` writes `knownOn + word + sgrOff + inputOn` for each known
   span: without re-opening `inputOn`, everything after the first highlighted word
   goes plain."* A `sgrOff` emitted by the deck highlighter would kill a selection
   background set by a different writer. There must be ONE composition point, not
   two writers layering escapes independently — same reason `highlightWriter`
   exists rather than definitions and answers each growing their own.

3. **"A click on ordinary text is NOTHING" stops being true.** That is a stated
   invariant (atlas §Clickable regions: *"no beep, no message. Pointing at a word
   that offers nothing is not an error"*), and inside a passage every word now
   offers something. This is a real change to a documented rule, scoped to the
   passage surface — it should be written down as scoped, not quietly broken. Note
   the same gesture also keeps its old meaning elsewhere: a click on a rendered
   entry's headword still plays it.

### Enter is the ask, and it is a row in the decision table

Operator, 2026-09-16: the ask gesture is **a bare Enter**. Only the BLANK line is
claimed — a non-empty line submits as it does today, and the natural reading is
that a typed question carries the marks as context ("explain these, specifically
answering this").

It belongs in `parseREPLLine`, not in either loop. That function is the single
decision table both loops route through, and the atlas gives the reason in its
own words: *"a prefix checked in either loop alone makes the loops disagree about
what a line means."* Enter-asks is a ROW, and `TestConsoleDecisionTable` gains
rows rather than the loops gaining branches.

**The conflict is narrower than it looks.** A blank Enter today means *replay the
current word*, guarded by `hasCurrent` (`repl.go:104`). But `session.current` is
set only by a SUCCESSFUL LOOKUP — so in the common flow (paste → mark → Enter)
there is no current word and no conflict at all. It arises only when a lookup
preceded the paste.

Proposed rule, with precedent: **marks win over replay**, and pasting a passage
clears `session.current` the same way a question deliberately never becomes it
(atlas: *"A question does not become the current word either"*). Replay stays
reachable via `/pron`.

**Two consequences to carry into the plan:**

- `parseREPLLine(line string, hasCurrent bool)` would grow a second boolean, and
  two bools next to each other encode a precedence nobody declared. Pass the
  session state as ONE value instead — the same consolidation `session` itself
  was created for.
- It scopes the headword-click shortcut. The atlas states a click on the headword
  *"is a shortcut for the bare Enter beside it"*; if Enter asks while marks are
  present, the two diverge unless the scoping is written down. Same family as
  collision 3 above — one invariant, two places it is now conditional.

### An authentic sentence is better material than an authored one

Today `items/<lang>/` holds practice sentences the MODEL writes, gated by an
entailment judge and a veto (atlas §Authored items). A passage the learner
actually met needs no authoring, no judge and no veto — it is real, it is the
context in which they first hit the word, and it is already on screen. If a
marked word enters the deck, the sentence it was marked in is the obvious example
to carry with it. That is a quality improvement and a model-call saving at once;
worth deciding in the brainstorm rather than discovering later.

## Done when

- A pasted passage stays on screen and its words are individually clickable;
  dragging selects a phrase across word boundaries; a click selects exactly the
  word under it.
- Marks accumulate, stay visible in their own treatment, and clicking a marked
  span unmarks it.
- A marked word that is ALSO a deck word renders both treatments correctly, and
  the token after it is not left plain (the ANSI-nesting regression).
- Marking N spans produces ONE explanation covering the passage AND each mark,
  asserted through the LLM fake by reading the real request — including that the
  marks arrive positioned within the passage, with a passage containing a literal
  bracket among the rows.
- An explained span with a dictionary entry enters the deck and appears in a
  later recall exercise; one without is explained and not retained. Both
  directions covered.
- A marked word is distinguishable from a typed lookup in the event log.
- Pasting multi-line text does not submit on the embedded newline.
- A bare Enter with marks present asks; with a passage but no marks it does the
  decided thing; with no passage it still replays the current word. All three are
  rows in `TestConsoleDecisionTable`, not branches in a loop.
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

## Revisions

### 2026-09-16 — operator refinement of the selection model

Reason: four follow-up asks in the filing conversation, before any brainstorm.

Delta:
- Multi-select made explicit; marks ACCUMULATE and persist with their own visual
  treatment; selection is a toggle over a set.
- A click is defined as a one-word drag — a convenience, not a second meaning.
- The request representation is now the whole passage with marks INLINE, and the
  model answers the passage as well as each mark. This CLOSES the open question
  of explain-on-release vs accumulate-then-ask, in favour of the latter.
- Added three collisions the earlier draft missed: the reserved `[lang=…]` bracket
  grammar, ANSI composition of a selection background with the deck highlight,
  and the now-scoped "a click on ordinary text is nothing" invariant.
- Added the authentic-example-sentence opportunity against `items/<lang>/`.

### 2026-09-16 — the ask gesture is a bare Enter

Reason: operator answered the one question left open by the previous revision.

Delta: Enter-asks specified as a row in `parseREPLLine`'s decision table, with
the `hasCurrent` replay conflict analysed (narrow — `session.current` is set only
by a successful lookup) and a precedent-backed precedence proposed. Two knock-ons
recorded: the two-boolean signature smell, and the scoping of the "headword click
is a shortcut for bare Enter" invariant. One sub-case returned to the operator:
blank Enter with a passage but no marks.
