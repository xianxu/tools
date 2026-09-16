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

### Click-to-select applies to the ANSWER area too (operator, 2026-09-16)

Once a click selects a word in a passage, the same gesture should work wherever
text is on screen: **click a word in an answer and it is selected and copied to the
clipboard.**

This is the same rule, not a second feature — *a click is a one-word drag* — and it
makes the codebase MORE consistent rather than less (ARCH-DRY):

- A drag in the answer area already copies (`selectionCopy`, `selection.go:72`), and
  `clipboardWriter` already exists for it. Click-to-copy reuses that effect
  wholesale; only the span derivation is new, and it is the same word-snap the
  passage surface needs.
- So the word-snap helper is built ONCE and has two consumers from the start: copy
  in an answer, mark in a passage. That is a good reason to build it EARLY, before
  the passage surface, where it ships value on its own.

**Precedence: regions win.** A click on a headword still plays it
(`RegionHeadword`), a click on an ORIGIN language still plays it there. Word-snap
applies to cells no region claims.

**This generalises collision 3 rather than adding to it.** The invariant *"a click
on ordinary text is NOTHING"* now becomes *"a click on ordinary text selects that
word"* — everywhere, not just inside a passage. One uniform rule is easier to state
and easier to test than a surface-conditional one, so this REPLACES the scoped
exception the earlier revision proposed.

### Survey findings, 2026-09-16 — three that change the design

Three parallel code surveys ran at `start-plan`. Most of the earlier capability
table held up. These did not:

**1. `RegionWord` ALREADY EXISTS, and clicking one plays audio.** The registry is
`RegionHeadword`, `RegionOriginLang`, `RegionWord` (`render.go:254`). `RegionWord`
is a DECK word in prose, produced by `deckSpans` → `wordRegions`
(`deckwords.go:59,90`), and `playRegion` (`replraw.go:707`) plays it in the
session's voice. So in today's answer area, clicking `equinox` already plays it.
The operator's click-to-copy therefore does not land on empty ground — it lands on
a shipped behaviour, and "regions win" would mean deck words play while every
other word copies. That is a fork, not a detail; see Open decisions below.

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
- Mode 2004 is absent, and `TestEveryEnabledMouseModeIsDecoded` (`key_test.go:400`)
  already encodes "for every mode we ENABLE, the decoder answers every encoding it
  can reply in" — so a half-done paste fails a test by design. `ESC[200~` is
  currently PINNED as `KeyUnknown` at `key_test.go:71`; that assertion must be
  rewritten deliberately.
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
