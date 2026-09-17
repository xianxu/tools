---
id: 000067
status: working
deps: []
github_issue:
created: 2026-09-16
updated: 2026-09-16
estimate_hours: 7.78
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
- ~~Blank Enter with a passage but ZERO marks?~~ **Settled 2026-09-16:** neither
  explain-everything nor replay — a local nudge. See below.
Resolved at `start-plan`, 2026-09-16 — recorded with rationale so the plan does
not re-litigate them. Any of these is cheap to revisit; none is load-bearing
enough to block design.

- **Selection marker: `[sel]…[/sel]`.** Same family as the reserved
  `[lang=xx]…[/lang]` rather than a second bracket dialect (ARCH-DRY), so the model
  meets one grammar. A literal bracket in the passage escapes as `&#91;`/`&#93;` —
  the rule `askSystem` already states for the ANSWER direction, now extended to the
  prompt direction, which is where it was always missing.
- **The passage lives in the SESSION, not on disk.** It is transient reading
  material, not learner data. What deserves to persist is the residue — the marked
  word plus the sentence it was marked in — and that lands on the deck word. So
  #56's asymmetry does not apply here: nothing valuable is trapped in memory,
  because the valuable part is extracted before the process ends. Also answers
  ARCH-FUNERAL: the passage dies with the session; the example sentence is bounded
  by the deck that holds it.
- **Click density: OUT of scope, its own issue.** A genuinely separable extension
  rather than the deferred point of this one (ARCH-PURPOSE). It needs the marking
  data this issue produces, so it is a natural successor, not a parallel concern.
- **The passage becomes the example sentence: IN scope**, final milestone, as a
  step that can be dropped on its own. The operator asked for marked words to enter
  recall; a real sentence is materially better material than an authored one and it
  is already on screen. Left out, the recall item is worse for no reason.
- **A lookup while a passage is on screen carries the passage as context: YES.** It
  is one more optional field on `askContext`, which is already "whatever this
  directory happens to hold". No new mechanism.
- **`/bilingual` and #64 compose for free.** The passage explanation renders through
  `renderAskPrompt` like every other question, so it inherits whatever language and
  level policy exists then. No work in this issue; #64 will find one more consumer
  already deriving from its record rather than restating it.
- **A dragged phrase with no dictionary entry stays out of the deck** — the
  operator's rule holds for this issue. Recorded as revisitable: `at the zenith of`
  is learnable and NOAD has no entry for it, so the rule may want relaxing once
  there is usage data.
- **Word regions must survive wrapping** — reuse `phraseGap`'s discipline rather
  than reinventing it; the plan names the shared helper.

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
   `equinox` and `synodic` came back green.

   **Superseded in part (see Revisions, 2026-09-16, mark precedence):** they do
   NOT compose. An explicit fg/bg pair overrides what it re-asserts over, so a
   marked deck word renders in the MARK treatment and loses its green for as long
   as the mark lasts. The mark is the salient state and it is short-lived.

   What remains true, and is the whole reason this is a collision, is the ANSI
   hazard underneath: *"`RenderLine` writes `knownOn + word + sgrOff + inputOn` for each known
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

### The line classifier, corrected — and where marking actually lives

Stated during the filing conversation and corrected against the code, because the
obvious framing ("one word = lookup, many words = infer") is the one the design
deliberately rejects. **Word count is not the signal — the dictionary is.**
`hot dog` and `a priori` are multi-word LOOKUPS; only a MISS is classified. A
one-word miss is *not found*, never a question.

| input | decided by | outcome |
|---|---|---|
| `/lang es` | `/` in column 1 | command |
| `?hot dog` | forced hatch | question, dictionary never consulted |
| `\how so` | forced hatch | lookup, question fallback suppressed |
| any line NOAD has | **dictionary hit** | lookup — any word count |
| miss, reads interrogative | `readsAsQuestion` | question |
| miss, otherwise | — | not found |
| blank, current word | `hasCurrent` | replay |
| blank, nothing | — | nothing |
| **blank, marks present** | **passage state** | **implicit "what does this mean"** |
| **typed line, marks present** | **passage state** | **that question, passage + marks as context** |

**Marking is a classifier INPUT that arrives out-of-band.** Two half-truths were
traded before this landed, and the synthesis is sharper than either:

- Marks are not text. The parser cannot read them off the line, so they are not
  syntax and passage state must not be smuggled into the line string.
- But marks are decisive EVIDENCE about the text. Operator, 2026-09-16: *"a line
  of text with user marking is surely a sentence; a piece of text without it, you
  are much less sure."* That is exactly right. `readsAsQuestion` exists to GUESS
  prose-or-headword from shape — four words, a trailing `?`, a leading
  interrogative. A mark is PROOF of prose, so the guess is not needed.

So the heuristic arms of `readsAsQuestion` are the fallback for UNMARKED text, and
marking short-circuits them. This is not a separate branch and it is not new
syntax: it makes an existing classification certain. It also means the marked case
is not only the blank-line branch — marked-and-typed is one decision too, which is
a further argument for passing session state as ONE value rather than growing a
second boolean.

The last two rows stay separate on purpose: an implicit *what does this mean* and
an explicit typed question over the same marks are different requests, and the
second is the more valuable one.

### A one-word miss could be a did-you-mean (separate issue)

Operator, 2026-09-16: rather than reporting *not found*, let the model correct a
typo — in a dictionary context it will find words that look or sound like it.

Right direction, and there is no cheaper offline route: `complete.go` is PREFIX
typeahead over History, not a speller, and Dictionary.app matches inflections
(`bargainer` → `bargain`) but not misspellings. So a model call is the honest
mechanism.

**No local speller** (operator, 2026-09-16 — a local edit-distance pass over the
deck/History was proposed and rejected). The deck is ALREADY in the prompt, so the
model can spot the misspelling with no new machinery. Keeping it simple wins twice:

- Edit-distance would need a threshold, tie-breaking, an auto-correct-vs-suggest
  decision and tests around the near-misses — real surface for a case that is
  otherwise free.
- And it would cover the NARROWER half. It can only find a typo of a word the
  learner has already seen; a learner mistyping a word they are meeting for the
  FIRST time — the more common case — finds nothing locally and round-trips anyway.

It is also BETTER, not merely simpler: the deck is a personalized ranking signal.
`sycophanti` resolves to `sycophantic` because that word is in THIS learner's deck,
where a generic speller would have no reason to prefer it over any other near
neighbour.

The plumbing already exists. `question` carries HOW it arrived, and the atlas notes
that for an unforced question *"is not a word is true by construction"* — the
provenance that makes a did-you-mean honest travels with the question already. What
is needed is a prompt line licensing the correction (today `askSystem` says only
*"Say plainly when you are unsure"*) plus routing a one-word miss to the ask.

Note in passing that `sycophanti` is a literal prefix of `sycophantic`, so typeahead
already offers it WHILE typing; the miss only survives when the suggestion is
ignored.

**Three constraints keep this from being a blanket change** — none of them touched
by dropping the local speller, since they govern WHETHER to ask, not how to correct:

1. **`-raw` never asks.** A stated absolute enforced at `mayAsk`, precisely because
   guarding only one route left three of six cells asking anyway. A typo in
   scripting mode stays not-found, exit 1.
2. **History keeps typos on purpose** (#20) so they stay Up-arrow recallable, and
   the atlas notes that highlighting a misspelling as a known word *"is the
   opposite of reinforcement."* If a typo auto-resolves, `recallLine` has to decide
   which form is the canonical re-submittable one — the typo or the correction.
3. **The event log distinguishes a not-found lookup from an ask**, deliberately:
   *"a question recorded as a not-found lookup is data that was never a lookup."* A
   did-you-mean is arguably neither, and #17 folds over these events.

**This belongs in its own issue**, not #67: it changes the console classifier
globally rather than the read-along surface.

**Undecided:** what a plain LOOKUP means while a passage is on screen. Mark
`precession`, then type `zenith` — still a lookup, but should it carry the passage
as context? Nearly free, probably right, nobody has decided it.

### The default level, when there is no learner model

Operator, 2026-09-16: absent a learner model, assume **a curious high school
student**.

This REVERSES a stated rule. `askSystem` currently says *"If the learner model is
absent, write for a capable adult reader and do not guess at their level"*
(`askctx.go:153`) — a deliberate refusal to guess, sitting under a comment calling
the level sentence *"the sentence that makes the whole adaptive loop worth
building."* Changing it is a reversal to record, not a tweak to slip in.

Sharpened by the operator moments later: **college-bound, but without the
sophistication or the knowledge yet — and curious.** That is a persona with two
dials set in OPPOSITE directions, and reading it as one dial is the failure mode:

- **Language: hold it.** They are going to college. Do not simplify the prose,
  shorten the sentences, or swap a hard word for an easy one. In a VOCABULARY
  tool, replacing the difficult word with a plain synonym defeats the entire
  purpose — a real risk, because "high school student" read carelessly means
  exactly that to a model.
- **Background: assume none.** Do not take the ecliptic, the equatorial bulge or
  the synodic month as known. This is the dial the feature exists to move, and it
  is the operator's original thesis restated as a persona: *learning words is not
  just learning words, but some base level information around those words.*
- **Curious: go one step past the question.** Volunteer the connecting fact, which
  is what "slightly more surrounding information than a dictionary app" asks for.

Worth a live conformance check rather than a prompt line alone: an explanation of
a hard word must not paraphrase the hard word away. The repo already measures
prompts against the live proxy this way.

It is the better default here, and for the reason the atlas already gives in a
neighbouring case: an abstract instruction produces *"a confident generic
answer."* "Capable adult reader" is exactly that shape. "Curious high school
student" is concrete enough for a model to act on. Keep the word **curious** — it
licenses going a little past the question, which is the whole point of the feature
(*"slightly more surrounding information than a dictionary app"*).

**Two things to settle:**

1. **Scope: GLOBAL** (operator, 2026-09-16). One answer to "what level do we
   assume," stated once — a one-line reversal in `askSystem` that read-along
   inherits, not a second default in a second prompt (ARCH-DRY). Split only if it
   proves wrong in practice. Longer term it is the zero value of the per-deck level
   record #64 introduces, and both surfaces consume that rather than restate it.
2. **It is not a CEFR band.** `store.Band` is A1–C2, a LANGUAGE-PROFICIENCY scale,
   and `ParseBand` maps #17's prose onto it. "Curious high school student" is an
   assumption about PRIOR KNOWLEDGE, not vocabulary difficulty — a C1 reader can
   still lack the astronomy. Those are separate axes, and collapsing the new
   default into a band would lose precisely the thing this feature exists to
   supply. Whatever holds it, it is not `Band`.

### Blank Enter, passage present, nothing marked — the tool answers, not the model

Operator proposed replying to the effect of *"what do you want to know about it?"*
That is the right RESPONSE from the wrong PRODUCER.

It is a deterministic answer to a deterministic state — a passage is loaded and the
mark set is empty. Routing it through the model buys latency, cost and
nondeterminism (it may decide to explain the passage anyway) to produce what is
essentially a UI hint. This repo already guards that instinct elsewhere: the sitting
path pins its model seam to PANIC rather than nil so *"a network dependency [cannot]
creep into a path that promises to be offline."*

**It already has a home.** `nothingSays` is, in its own words, *"the ONE answer to
'this line meant nothing — why, and what should the user do about it'"* — which is
precisely the question here, and it is local by construction. Its note-less default
is already a nudge rather than an error (*"type a word, or press return to replay
the last one"*); only the hatch-with-no-payload notes carry exit 2. So this is a new
`note`, not a new mechanism and not an error path.

**Instruct rather than ask.** "Click or drag what you don't understand, then press
return" beats "what do you want to know about it?" — the user just pressed Enter, so
bouncing a question back is a small dead end, and this is a brand-new gesture nobody
discovers unaided. The nudge teaches it at exactly the moment it is needed.

**The alternative being declined, for the record:** treat zero marks as *the whole
passage is marked*. It needs no new code at all — the request shape is identical,
simply with no brackets in it, and it is the literal reading of "what does this
mean: <sentence>". Declined because Enter is cheap to press and a passage is an
expensive call, but it is a real option if the nudge proves annoying in practice.

### The mark becomes deck membership (operator, 2026-09-16)

**After an ask, marks CLEAR and the passage re-renders under the normal rules — so
the words just asked about now show as DECK words.** The transient state converts
into the durable one, visibly, at the moment it happens.

Three problems dissolve at once:

- **No accidental re-ask.** A bare Enter after an answer finds no marks and hits the
  local nudge, which is free. The "Enter fires an expensive call with no visible
  state change" hazard never arises.
- **No re-marking friction.** You do not need to remember what you covered: it is
  green. Follow-up means marking only what is new, which is exactly the iterative
  reading loop the feature is for.
- **The mark's whole job is now legible.** It is a short-lived request, not a
  persistent annotation — it exists only between marking and asking, which is also
  why it must read as clearly DISTINCT from deck green.

**Passage size: a sentence to a paragraph** (operator, same exchange) — roughly
1000 characters. Everything stays visible, so every word is reachable without
scrolling the passage.

**This settles the layout question by implication, not by preference.**
`screen.lines` is append-only and **immutable once written** (`screen.go:71`), and
deck colour is baked in at WRITE time by `highlightRegion`, while only `regions` and
`paints` are paint-time and mutable. So a passage printed inline as ordinary
scrollback can never re-render: the marks could be dropped (paint-time) but the
words could not turn green (baked). "Re-render under the normal rules" therefore
REQUIRES a live region the frame rebuilds — pinned, like the practice playbar's
`newPinnedScreen` (`screen.go:832`). Inline is off the table for a structural
reason rather than a taste one.

**One consequence to carry into the plan (ARCH-DRY).** Admission must go through
`Capturer`, not a second `Upsert`. There is exactly one production `Upsert`
(`capture.go:123`) and the interface doc states why: *"capture is the only thing
that records, and a second appender beside it is how that stops being true without
anyone noticing."* It also orders the writes deliberately — `vocab.Add` runs only
after the deck accepted the word, *"so it must not claim a word the deck rejected"*
— which is precisely the ordering that makes "the word turns green" honest.

**And one loose end, noted rather than solved.** The admission rule is dictionary-hit
only, so a marked phrase with no entry reverts to PLAIN rather than turning green.
That difference is meaningful — one is a word you are now learning, the other was a
phrase you needed explained once — but nothing on screen says so. Worth a look once
there is usage; not worth machinery now.

### Survey findings, 2026-09-16 — three that change the design

Three parallel code surveys ran at `start-plan`. Most of the earlier capability
table held up. These did not:

**1. `RegionWord` ALREADY EXISTS, and clicking one plays audio.** The registry is
`RegionHeadword`, `RegionOriginLang`, `RegionWord` (`render.go:254`). `RegionWord`
is a DECK word in prose, produced by `deckSpans` → `wordRegions`
(`deckwords.go:59,90`), and `playRegion` (`replraw.go:707`) plays it in the
session's voice — so clicking `equinox` in an answer already plays it today.

This is why the click-to-copy-in-the-answer request was WITHDRAWN (see Revisions):
it would have collided with a shipped gesture on exactly the words most likely to
appear in an answer, and "regions win" would have made a click mean *play* or
*copy* depending on whether the word happened to be in the deck — a distinction
invisible except for the colour. Outside a passage, a click keeps every meaning it
has today.

**2. `TestEveryRegionKindIsActionable` asserts *actionable == plays audio***
(`editorloop_test.go:953`, plus `...ThroughTheSharedRegistry` at `:985`). It derives
its loop from `numRegionKinds`, so a new kind is exercised the moment it is added —
and a passage-word kind whose action is "mark, do not play" FAILS it as written.
The guard has to be generalised from "plays" to "does something observable", which
is a deliberate change to a test that is doing real work, not a fix-up.

**3. A per-span BACKGROUND is not expressible today.** `rowPaint` carries ONE
`background` for a whole row plus *exclusions* — holes, not colours — and
`validRowPaint` (`output_layout.go:23`) rejects anything but `languageDark` /
`languageLight` / `""`. Worse, the selection highlight is `\x1b[7m` INVERSE VIDEO
(`selection_frame.go:241`), not a background: a deck-green word inside a selection
renders as green *background*, because inverse swaps foreground and background. So
"blue background on the marked words" cannot be built from either existing channel
without new machinery. See Open decisions.

**Two more structural gaps, both real but decidable in the plan:**

- **`highlightRow` takes two points, not a set.** Its signature is
  `(row int, a, b selectionPoint) string` (`selection_frame.go:208`) — one range per
  row. Accumulating marks IS a set of ranges, so this is the one place the selection
  side genuinely has to widen.
- **Word snapping does not exist in the selection path.** `selectionCell` is
  per-display-column and knows nothing about words; word boundaries live in
  `wordRuns` (byte offsets, plain text, `highlight.go:33`) and reach the screen only
  as `Region{Col,Width}` through `deckSpans`. "Click snaps to a word" needs that
  bridge built — it is the shared primitive this issue and the answer-area request
  both want.

**What the surveys CONFIRMED, and can be relied on:**

- The single composition point exists: `selectionLayout.paint` (`screen.go:512`) is
  the only place holding the styled bytes, the row's `rowPaint`, the frame's cell
  table AND the live gesture at once. Collision 2's "one composition point, not two
  writers" has a home.
- Both surviving decoration layers work by RE-ASSERTING their attribute after every
  foreign SGR (`language_row.go:45`, `selection_frame.go:243`) rather than trusting
  nesting. Any fourth decoration must do the same or compose at that one point.
- `markClickable` (`screen.go:362`) is the precedent for decorating a span safely:
  attributes only (`\x1b[4m`/`\x1b[24m`), never colour, closing with `24` rather
  than `0` so the palette survives.
- Mode 2004 is absent. `TestEveryEnabledMouseModeIsDecoded` (`key_test.go:400`)
  states the right rule — "for every mode we ENABLE, the decoder answers every
  encoding it can reply in" — but **it would NOT catch this**: it derives its modes
  by regex over `mouseOn` alone (`key_test.go:417`), so a separate `pasteOn`
  constant is invisible to it. Corrected 2026-09-16 by a plan review; an earlier
  revision of this section claimed the guard covered it. Widening the guard's
  source is therefore part of the work, not a nicety — otherwise 2004 is the first
  mode enabled outside the one test written to prevent exactly that.
  `ESC[200~` is currently PINNED as `KeyUnknown` at `key_test.go:71`; that
  assertion must be rewritten deliberately.
- **A paste must not arrive as N keystrokes.** `readInput` delivers into a 256-key
  channel with a hard drop-newest policy (`selection_input.go:157,204`), and `Apply`
  inserts one rune per key with no bulk path (`editor.go:55`). A pasted paragraph
  would lose its tail behind one "input full" notice (ARCH-CONSTRAINTS).
- `question` is `{text, forced}` with six composite-literal construction sites, so a
  new field defaults to zero at all six. `askTask` keys both the golden filename and
  the cassette, so the passage question takes its own task name and leaves
  `ask-prompt.txt` untouched.
- `store.Item{Form: FormCloze}` holds the example sentence AS `Stem`
  (`harvest.go:445`) — no new datatype needed. But `oneLine` (`store/item.go:200`)
  flattens whitespace on read AND write, and distractors still have to come from
  somewhere.
- `storeHistory.Load` filters `EventLookedUp` only (`history_store.go:64`) — a new
  event kind is invisible to Up-arrow recall unless added there.
- The piped loop never decodes escapes at all, so paste is raw-mode-only by
  construction, and `mayAsk`/`-raw` already scope the ask.

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
- A marked word that is ALSO a deck word renders in the MARK treatment, not in
  both and not in green — and the token AFTER the mark still carries the style it
  had (the ANSI-nesting regression). The second half needs a test that inspects
  the style after the span: stripping escapes is exactly what hides a lost one.
- Marking N spans produces ONE explanation covering the passage AND each mark,
  asserted through the LLM fake by reading the real request — including that the
  marks arrive positioned within the passage, with a passage containing a literal
  bracket among the rows.
- An explained span with a dictionary entry enters the deck and is SCHEDULABLE —
  it reaches recall by the ordinary route, because `harvest` authors items for
  deck words. One without an entry is explained and not retained. Both directions
  covered. (Reaching recall with the PASSAGE's own sentence as material is a
  separate issue; see Revisions.)
- A marked word is distinguishable from a typed lookup in the event log.
- Pasting multi-line text does not submit on the embedded newline.
- A bare Enter with marks present asks; with a passage but no marks it does the
  decided thing; with no passage it still replays the current word. All three are
  rows in `TestConsoleDecisionTable`, not branches in a loop.
- The NOAD-as-context inversion is stated in the atlas, with the route back to
  the full entry.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.90 impl=0.08
item: milestone-review         design=0.10 impl=0.12
item: milestone-review         design=0.10 impl=0.12
item: greenfield-go-module     design=0.16 impl=0.20
item: cross-cutting-refactor   design=0.08 impl=0.16
item: smaller-go-module        design=0.03 impl=0.16
item: smaller-go-module        design=0.03 impl=0.12
item: smaller-go-module        design=0.03 impl=0.10
item: atlas-docs               design=0.02 impl=0.05
item: milestone-review         design=0.00 impl=0.14
item: greenfield-go-module     design=0.16 impl=0.24
item: cross-cutting-refactor   design=0.08 impl=0.16
item: smaller-go-module        design=0.03 impl=0.10
item: atlas-docs               design=0.02 impl=0.05
item: milestone-review         design=0.00 impl=0.14
item: smaller-go-module        design=0.03 impl=0.12
item: cross-cutting-refactor   design=0.08 impl=0.18
item: tui-screen               design=0.12 impl=0.24
item: tui-screen               design=0.10 impl=0.20
item: atlas-docs               design=0.02 impl=0.05
item: milestone-review         design=0.00 impl=0.14
item: smaller-go-module        design=0.04 impl=0.16
item: cross-cutting-refactor   design=0.08 impl=0.16
item: smaller-go-module        design=0.05 impl=0.18
item: smaller-go-module        design=0.03 impl=0.12
item: real-api-discovery       design=0.00 impl=0.18
item: atlas-docs               design=0.02 impl=0.05
item: milestone-review         design=0.00 impl=0.14
item: smaller-go-module        design=0.04 impl=0.16
item: smaller-go-module        design=0.02 impl=0.14
item: atlas-docs               design=0.02 impl=0.05
item: milestone-review         design=0.00 impl=0.24
item: ux-rename-iteration      design=0.10 impl=0.08
item: ux-rename-iteration      design=0.10 impl=0.08
item: ux-rename-iteration      design=0.10 impl=0.08
design-buffer: 0.15
total: 7.78
```

Derived after the plan cleared plan-quality (#187), in plan-task order: rows 4–10
are M1, 11–15 M2, 16–21 M3, 22–28 M4, 29–32 M5; rows 33–35 are the TUI iteration
rounds M2 and M3 will draw on.

Familiarity **1.0**. The design is warm — three code surveys in this session read
the input path, the screen/selection stack and the ask/store path end to end — but
**no code has been written**, so there is no editing warmth to discount for. The
plan carries the `pasteScanner` implementation verbatim and full test bodies for
M1/M2; that is spec quality, priced through the ×0.2 design discount, not
familiarity.

Design carries v2's ×0.2 spec-quality discount on every code row: the plan
pre-resolves the scanner's contract, the three-space coordinate mapping, the mark
precedence rules, the `[sel]` grammar and the ask-outcome predicate. Implementation
is v3.1's 40% of the v2 table.

- **`issue-spec` is NOT discounted — it IS the design**, and it is priced inside the
  table's undiscounted 0.5–1.5 band. **0.90** covers a long exploration that moved
  the unit twice (word → concept → structure), four operator refinements to the
  selection model, three parallel code surveys, the durable plan, and two
  fresh-eyes plan-document reviews whose findings were substantive — one reviewer
  built and ran the paste scanner and measured its failure. Near the top of the
  band because two *design* errors were found and corrected here rather than in
  code: a scanner that double-counted re-presented bytes, and a `phraseGap` claim
  that did not hold. At 0.90 it sits just BELOW the band's midpoint — deliberately
  low, since the exploration is fully spent and measurable rather than forecast.
  (An earlier draft of this note called it "near the top of the band", which the
  estimate-quality judge correctly flagged as prose arguing for a bigger number
  than the row carries.)
- **Rows 2–3 are the two plan-quality rounds, counted as SPENT, not budgeted** —
  priced as `milestone-review`, which is the primitive #24 used for exactly this
  (a plan round is a review round). Round 1 returned four Importants: the drain
  state unreachable through an ESC-only hook, an unhandled untrusted-input class
  at the paste boundary, the passage/footer/frame coordinate mapping unstated, and
  the ask's five non-success outcomes uncollapsed. Round 2 disposed of all six and
  passed. #24's rule applies — a round this block can already see is counted.
- **Two `greenfield-go-module` rows, and only two.** `paste.go` and `passage.go`
  are new files with new state and no mirror in the tree. Everything else extends
  something that exists, which is `smaller-go-module` territory.
- **Four `cross-cutting-refactor` rows**, each earning it by call-site count rather
  than by feel: `decodeKey` → method (39 sites, plus fuzz-freshness), the three
  `view.Draw` sites, `highlightRow`'s signature plus its new inverse, and
  `parseREPLLine`'s `hasCurrent` → session-state across three non-test and ~11 test
  callers.
- **Two `tui-screen` rows** for Tasks 3.3 and 3.4 — mark painting and the
  click/drag gesture are screen state machines with their own tests, which is what
  that primitive names.
- **One `real-api-discovery`** for Task 4.4's live conformance row: it reaches the
  real proxy, and the persona it defends ("hold the language, drop the background")
  is exactly the kind of prompt claim that needs a real answer to falsify.
- **One `milestone-review` per Mx, five in total**, because each `Mx` row in the
  Plan commits to its own `sdlc milestone-close`. The last is priced slightly
  higher (0.16) as the issue close rather than a milestone.
- **Three `ux-rename-iteration` rows, added after the estimate-quality check.** The
  first version priced ZERO, on an issue with two TUI-heavy milestones, against a
  baseline that says in as many words: *"Plan for 3–5 rounds per TUI-heavy
  milestone, not 1"* (`baseline-v2.1.md:75`) — the documented systematic miss for
  exactly this shape. The evidence is already in this file: seven operator-driven
  `## Revisions` entries at SPEC time, before a pixel exists. The mark treatment is
  the obvious candidate — Task 3.3 pins "white-on-blue" in a test name, but the
  operator only ever *proposed* a blue background, and the survey found no channel
  expresses a per-span background at all. Three rounds, priced at the low end
  (design 0.10, impl 0.08) because each round is a colour or precedence tweak, not
  a re-design.
- **The issue close is 0.24, not a milestone's 0.14.** Task 5.3 carries strictly
  more than a boundary review: the full suite re-run UNSANDBOXED, a
  `-tags conformance` run, a nine-row Done-when audit, the atlas pass, project
  ticking, and the close gate's own fresh-eyes review with remediation. The first
  version priced it at 0.16 — a milestone plus two minutes.
- **Two counts corrected** from the estimate-quality check: `parseREPLLine` has
  **14** test callers, not ~11; and there are **four** non-test `view.Draw` sites,
  of which this issue scopes three (`replraw.go:408,506,567`) and deliberately
  leaves `practice_output.go:189` alone — the practice playbar is not this surface.
- Design buffer **+15%** for a thorough plan doc. 2.69 × 1.15 + 4.69 = 7.78.

**Empirical cross-check.** `sdlc actual --issue 67` read **1.28h** at the moment
this block was written, against the 1.58h rows 1–3 budget for the same spent
design window — about 19% conservative in the same unit the model is calibrated
in. Roughly a fifth of the total is consumed before the first line of feature
code, which is the honest shape of a five-milestone issue whose hard parts were
found at design time.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

## Plan

Durable plan: `workshop/plans/000067-read-along-passage-plan.md`. Five review
boundaries; each `Mx` row closes with its own `sdlc milestone-close`.

- [x] brainstorm the open questions (2026-09-16, in-session; decisions recorded above)
- [x] `sdlc start-plan`, write the plan
- [x] paste: mode 2004, the paste scanner, `KeyPaste` as ONE key, the 1000-RUNE
      cap (runes, not bytes — a byte cap refuses a CJK paragraph at a third of its
      length), the abandon rule, and `scan`'s exits as an enumeration.
- [x] the passage on screen: `passage` + `wordAtCell`, rendered into the screen's
      `footer` channel (chrome, not buffer text, because `screen.lines` is
      immutable), under the normal deck-highlighting rules.
- [x] `markSet` and drag-to-span.
- [x] marks painted: `highlightRow` widened from one range to a set, the three
      precedence rules.
- [x] click and drag produce marks.
- [x] the ask: `renderPassagePrompt` with `[sel]` + bracket escaping under its own
      task name, `parseREPLLine` marks-aware, the local nudge, the global
      level-default reversal.
- [x] the words become deck words: `CaptureMarked` through the one `Upsert`, the
      word becoming schedulable, the passage re-rendering green.
- [x] atlas

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

### 2026-09-16 — line classifier corrected, default level set

Reason: operator restated the decision table and set a default level.

Delta:
- Recorded the corrected classifier. The count-based framing is rejected: the
  dictionary is the classifier, hits win at any word count, and a one-word miss is
  not-found rather than a question.
- Established that marking extends the BLANK-LINE branch, not the word/question
  classifier — it is out-of-band state, not text. This changes where the code goes.
- Split implicit "what does this mean" from an explicit typed question over the
  same marks; they are different requests.
- New undecided row: does a lookup carry the passage as context?
- Default level with no learner model is "a curious high school student",
  REVERSING `askSystem`'s "do not guess at their level". Scope (global vs
  read-along) returned to the operator, and flagged that it is not a CEFR band.
- Persona sharpened: college-bound, no sophistication or knowledge yet, curious.
  Two dials in opposite directions — HOLD the language level, DROP the assumed
  background. Guarding the failure mode (simplifying the hard word away) is a
  live conformance check, not just a prompt line.

### 2026-09-16 — marking reclassified, did-you-mean split out

Reason: operator pushed back on two points; both improved the model.

Delta:
- Marking is neither syntax nor merely a separate branch: it is a classifier INPUT
  arriving out-of-band, and it makes `readsAsQuestion`'s heuristics unnecessary
  rather than competing with them. The heuristics are the unmarked fallback.
  Supersedes the previous revision's "extends the blank-line branch" framing,
  which was true but too narrow — marked-and-typed is one decision too.
- Recorded the did-you-mean proposal for one-word misses, with its three
  constraints (`-raw` asks never, History keeps typos deliberately, the event log
  separates not-found from ask). Flagged as ITS OWN ISSUE — it changes the console
  classifier globally, not this surface.

### 2026-09-16 — default level scoped global; empty-mark Enter settled

Reason: operator answered the two remaining interaction questions.

Delta:
- The no-model default level is GLOBAL — one `askSystem` line, read-along inherits.
  DRY first; split only if practice shows it wrong.
- Blank Enter with a passage and no marks produces a LOCAL nudge via `nothingSays`,
  not a model round-trip: deterministic state, deterministic answer, and that
  function already exists to answer "what should the user do about it". Phrased as
  an instruction ("click or drag what you don't understand, then press return")
  rather than a question back. The zero-marks-means-whole-passage alternative is
  recorded as declined-but-available.

### 2026-09-16 — no local speller

Reason: operator rejected the proposed edit-distance pass in favour of letting the
model correct, since the deck is already in the prompt.

Delta: local spell-correction dropped. Recorded the two reasons it is the better
call — edit-distance covers only typos of words already seen, which is the narrower
half, and the deck in context is a personalized ranking signal a generic speller
could not match. Also recorded that `question` already carries the "the dictionary
missed this" provenance, so the change is a prompt line plus routing, not plumbing.
The three constraints are unaffected: they govern whether to ask, not how to correct.

### 2026-09-16 — open questions resolved at start-plan

Reason: entering design; the remaining questions were decidable without further
operator input and are recorded with rationale rather than left to the plan.

Delta: settled the selection marker (`[sel]`, one grammar), the passage's home
(session; the residue persists, not the passage), density (out, own issue), the
example sentence (in, droppable step), passage-as-lookup-context (yes), #64/bilingual
composition (free), the non-headword rule (holds, revisitable) and wrapping (reuse
`phraseGap`).

### 2026-09-16 — click-to-select generalised to the answer area

Reason: operator asked for click-to-select-and-copy in the answer area, not only in
a passage.

Delta: added it as a consumer of the SAME word-snap helper, to be built early since
it ships independently (a drag already copies; only span derivation is new).
Supersedes the scoped treatment of collision 3: "a click on ordinary text is
nothing" becomes "a click on ordinary text selects that word" uniformly, with
regions taking precedence where they exist.

### 2026-09-16 — click-to-copy in the answer area withdrawn

Reason: operator withdrew the request after the survey showed `RegionWord` already
claims clicks on deck words in prose and plays them.

Delta: the answer-area click section is removed from the Spec. Collision 3 returns
to its SCOPED form — "a click on ordinary text is nothing" becomes false only inside
a passage, and every click outside one keeps its current meaning. The word-snap
helper now has ONE consumer (the passage surface) rather than two, so it is no
longer a candidate to build first for its own sake; it is built where it is used.
Supersedes the revision directly above it.

### 2026-09-16 — marks clear into deck membership; layout follows

Reason: operator settled the post-ask behaviour and the passage size.

Delta:
- Marks CLEAR after an ask and the passage re-renders normally, so asked-about words
  appear as deck words. Better than either option offered: it removes the accidental
  re-ask, removes the re-marking friction, and makes the mark's transience legible.
- Passage bounded to a sentence/paragraph (~1000 chars), fully visible.
- Layout is DERIVED, not chosen: an append-only buffer with write-time baked colour
  cannot re-render, so the passage must be a pinned live region. Inline is
  structurally impossible for this behaviour.
- Admission routes through `Capturer` (one `Upsert`, `vocab.Add` only after the deck
  accepts) — which is also what makes the green honest.

### 2026-09-16 — mark precedence recorded; the authentic sentence split out

Reason: a fresh-eyes plan review caught the plan citing this issue for a decision
this issue stated the opposite of; and investigating distractors showed the
authentic-sentence task was larger than scoped.

Delta:
- **Mark precedence is now recorded here, where the plan wrongly claimed it already
  was.** A marked deck word renders in the mark treatment and loses its green while
  marked; the two do not compose. Collision 2 and the matching Done-when are
  amended. The ANSI hazard under collision 2 is unchanged and still governs.
- **The authentic sentence as practice material is OUT of this issue.** It cannot
  stand alone: distractors come from the deck's BANDED words via `pickDistractors`
  (`harvest.go:390`) and are then vetoed, so the sentence replaces the authoring
  step, not the pipeline. Worse, wild prose routinely FAILS the author prompt's own
  requirements — it must point at the word without defining it, and never gloss it,
  *"not as an appositive, not as a relative clause"* — and expository prose defines
  terms in place, which is exactly the appositive that makes a reading test rather
  than a vocabulary test. A real sentence needs MORE judging, not less. This
  corrects the earlier "an authentic sentence is better material than an authored
  one" note above: it is more REAL, not automatically better as a stem.
- It also lands on an unfinished thread: `usage/` already caches real sentences per
  word (news + NOAD examples) and `bothSources` is wired into `deps`, but `Usages`
  has NO CALLER — the atlas's *"#10's authoring step is the consumer"* never
  happened. Absorbing it here would have made this issue a second stalled producer.
  Filed separately so one consumer serves all three sources.
- Done-when for recall is corrected accordingly: admission alone makes a word
  schedulable, because `harvest` authors items for deck words by the ordinary route.

### 2026-09-16 — M1 implemented

Bracketed paste lands. Four tasks, TDD throughout, full suite green (unsandboxed;
`TestLanguageTintInvocation` and the `language_prompt_paths` rows fail under the
Bash sandbox and pass on the host — see `MEMORY.md`).

**One deviation from the plan, deliberate.** The plan had `runEditor` intercept
`KeyPaste` and leave it without a destination until M2's passage surface. That
would have REGRESSED the working case: before this milestone a pasted word typed
itself into the line correctly, and only a pasted newline misbehaved. So M1 makes
a paste insert into the line atomically, with interior newlines becoming spaces
(`pasteLineRunes`). M2 will redirect a passage-shaped paste to the passage; the
line insertion stays as the fallback. Shipping a milestone that makes an existing
gesture do nothing is not a smaller step, it is a worse one.

**`sanitisePasteBody` landed inside Task 1.1 rather than as its own task.** The
plan put the parse boundary in `newPassage`, which does not exist until M2 — so
Task 1.2b's tests referenced a symbol a later milestone creates. Putting the
boundary in the scanner is better anyway: the bytes become a typed value at the
moment they stop being a wire format, and nothing downstream can forget to ask.

**Two repo guards fired and both were right.** `TestPlanTablesNameEntitiesThatExist`
rejected two plan rows marked `modified` for entities that are NEW in existing
files — the status column describes the entity, not the file. Then
`TestPlanCitesTestsThatExist` rejected the name I guessed for the first guard. The
plan now names both correctly.

### 2026-09-16 — milestones collapsed to ONE boundary (operator)

Reason: the operator chose a single review over per-milestone gates, after M1
alone took four boundary-review rounds and the feature was still not
smoke-testable.

Delta: the `Mx` tags are removed and the Plan is plain checkboxes, which is what
AGENTS.md §3 prescribes for work closing at one boundary — *"tagging a one-shot
task M1 forces a redundant milestone-close + issue-close double-log"*. The
mandatory fresh-eyes review still runs, once, at `sdlc close`.

The M1 work is already committed and has been through four review rounds
(BR-1..BR-17, two Criticals, all disposed or fixed); its findings and the review
sidecar stay in the record. What changes is only that its close folds into the
issue close rather than running as a fifth round of its own.

### 2026-09-16 — the passage is a record, not chrome (operator, from smoke test)

Reason: the operator ran the binary and reported, with screenshots, that the
passage stayed welded to the prompt forever, then that it did not wrap and that
every word was underlined.

Delta, and the first item REVERSES a decision recorded above:

- **The passage is BUFFER text and scrolls away**, like a definition or an answer.
  It was footer chrome, which is redrawn every frame and never scrolls. I chose
  the footer to honour "marks clear and the asked-about words turn green", which
  needs a region the frame rebuilds — `screen.lines` is immutable once written.
  The two requirements were in direct conflict and I picked the wrong one to
  honour. **The green re-render is dropped** (operator: "let's remove that
  requirement so it's ok for it to scroll off"). Rationale recorded: finding
  earlier content is navigation's job, not something to solve by pinning — the
  operator points at an "outline" feature for that, and at #69's header, which
  implies the whole screen becoming a screen program later. Out of scope here.
- **The passage wraps at construction**, so a passage line is a buffer line is a
  Region line. Unwrapped, long lines ran off the right edge and the words past the
  margin could not be clicked at all.
- **Passage words are not underlined.** `markClickable` marks a span that offers
  something its neighbours do not; in a passage every word does, so it said nothing
  and made the text unreadable. `regionUnderlines` declares the split beside
  `regionPlaysAudio`.
- **`RegionPassageWord` is back.** It was resolved away when the passage was a
  surface; with the passage in the buffer, the click map IS how its content is
  reached. It is the first kind the audio registry does not answer for, which is
  what forced both declarations to be explicit rather than assumed total.
- Smoke test passed on the operator's machine after these three fixes.
