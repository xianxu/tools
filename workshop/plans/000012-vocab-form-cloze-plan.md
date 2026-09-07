# Form 2.2 — the cloze question — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A review question that shows the word's own sentence with the word blanked out and four words to choose from — consuming the items `#10` authored, so the sitting stays instant, free and offline.

**Architecture:** `#10 M2` already writes finished items to `items/<lang>/`: stem, answer, and distractors already selected and already vetoed. So this issue builds no selection and no model call. It is three things: a `Cloze` form in `play`, a BLANKER in `main` that hides the answer without leaking it, and the wiring that prefers an authored item over a definition match. Plus the bad-question gesture, which `#12`'s Spec credited to `#6` and which does not exist.

**Tech Stack:** Go. `cmd/define/play` (the pure form, imports nothing — D5a), `cmd/define` (the blanker and the wiring), `cmd/define/store` (one event field). No new dependencies, no new model calls.

---

## What this issue is NOT, and why the scope is small

Three of `#12`'s five original Done-when rows moved to `#10 M2` with the selection they belong to, and are recorded as satisfied on the issue: options are never model-generated (they are selected offline from the banded deck and stored finished), the near-synonym case (`sycophantic`/`obsequious` is `#10`'s committed known-bad veto row), and the form working with the seam unavailable (an item read from disk never reaches for a model, so there is no veto step to skip).

**What remains is rendering and wiring**, plus the flag. That is the whole of this plan.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Cloze` | `cmd/define/play/cloze.go` | new |
| `optionSet` | `cmd/define/play/optionset.go` | new |
| `Choice` | `cmd/define/play/choice.go` | modified |
| `Flagging` | `cmd/define/play/session.go` | new |
| `blankStem` | `cmd/define/cloze.go` | new |
| `wordRunEnd` | `cmd/define/harvest_item.go` | new |
| `clozeFor` | `cmd/define/cloze.go` | new |
| `usableItem` | `cmd/define/cloze.go` | new |

- **`Cloze`** — one cloze question: a blanked stem and N words to choose from.
  - **Relationships:** 1:1 with a `store.Item` of `FormCloze`. Holds no store handle — the item arrives finished, exactly as `Choice` takes finished options (D5: prose does not cross into this package).
  - **DRY rationale:** see `optionSet` below. What is genuinely NEW here is the prompt (a sentence with a hole in it, not a headword) and the reveal.
  - **THE REVEAL CARRIES THE DEFINITION TOO (PQ-5), and that is a decision.** The restored sentence is the payload — seeing the word in its own context is why `#10` authored a sentence rather than a definition. But `choice.go:82` argues at length that a recognition form revealing LESS than the retired form 2.1 *"would have taught less than the easier form did"*, and a learner who just missed a word wants its senses, examples and origin. So the reveal is: the restored sentence, then what they picked, then the rendered entry — the same three-part shape `Choice.Reveal` has.
    **Consequence, which is why this is settled at plan time rather than in Task 2:** a word taking cloze STILL needs `d.dict.Lookup` and `Render` in `todaysQuestions`, so Task 5 does not get to skip the lookup for cloze words. Deciding it later would have been discovering it later.
  - **Future extensions:** `#13`'s free-sentence form is a different shape and should not reuse this; `Item.Form` is what keeps them apart in one store.

- **`optionSet`** — the numbered, digit-graded, remembers-what-you-picked machinery both option forms share.
  - **Relationships:** embedded by `Choice` and `Cloze`, 1:1 with each.
  - **DRY rationale, and the alternative considered.** `Choice` and `Cloze` differ in what an option SAYS (a definition vs a word), in the prompt above it, and in whether `Axis` means anything. They do NOT differ in the mechanical contract, which is subtle and worth having once: options are numbered from 1, `Grade` is asked BEFORE the reveal, the reserved keys (`Enter`, space, `d`, Ctrl-C) must not be built into the answer set, `Keys()` names only the digits that actually work, and the pick is remembered for the reveal. Two copies of that contract is two places for it to drift, and the second copy would be written by someone reading the first.
    - **The alternative — leave them separate and accept ~30 duplicated lines** — is defensible and was rejected for one measured reason: `Choice.Keys()` carries a comment explaining why it does NOT branch below two options, and `Grade`'s contract is documented on the `Question` interface across four paragraphs. A second implementation is where a documented subtlety goes to be forgotten.
  - **Future extensions:** if `#13` ever needs a graded option set it embeds this rather than copying either form.

- **`Choice`** (modified) — form 2.3, unchanged in behaviour, embedding `optionSet` instead of holding `options`/`chosen` itself.
  - **This is a REFACTOR with no behaviour change, and its whole test surface already exists.** `#7`'s suite is the proof: if any of it moves, the extraction was wrong.

- **`Flagging`** — the optional capability a form implements when its last keystroke said *"this question is broken"*.
  - **Relationships:** implemented by `Cloze` only, for now.
  - **THE WHOLE SIGNAL PATH, named (PQ-2).** The first draft specified the gesture and the log field and nothing between them, and the obvious wiring is actively wrong: `CaptureReview` hardcodes `Kind: store.EventReviewed` and `Correct: verdict == Correct` (`capture.go:143`), `Fold` folds every reviewed event (`progress.go:155`), and `GradeOf(correct=false)` is `GradeWrong` (`progress.go:180`) — **so a flag written as a reviewed event DEMOTES the word**, which is what this plan's own Task 6 forbids. On an append-only log that is not cheaply reversed. The path is therefore named end to end, and it copies `OutcomeDrop`'s exactly:
    | stage | drop (existing) | flag (new) |
    |---|---|---|
    | capability | `Dropping` | `Flagging` |
    | outcome | `OutcomeDrop` | `OutcomeFlag`, carrying Word + the options |
    | loop arm | calls `d.deck.Forget` | calls `CaptureFlag` |
    | store verb | `Forget` | `CaptureFlag`, a new verb |
    | event kind | none written | `EventFlagged`, a new kind |
    **`Fold` needs no change and must get none**: it already skips every kind that is not `EventReviewed`, so the ladder ignores a flag BY CONSTRUCTION rather than by a filter someone has to maintain. That is the property to pin.
  - **DRY rationale:** this is the FIFTH member of an established pattern — `Missed`, `Dropping`, `Batch`, `Grid`, `Moded`, `SelfRated` are all optional interfaces the session asks for via a helper, and `session.go` twice states that a type switch on a concrete form is the thing `#6`'s Done-when forbids. A new gesture takes the same shape or it is a new precedent nobody chose.
  - **Future extensions:** any form can flag; the board is the obvious next one.

- **`blankStem`** — replaces the answer in a stem with `___`, everywhere it occurs, without leaking it.
  - **Relationships:** pure, `main`-side. Called once per question at build time, not at render time.
  - **DRY rationale, CORRECTED (PQ-4).** The first draft kept `#10`'s `blankOut` as a separate "deliberately simple" helper on the grounds that a miss there only *"costs the judge some context"*. **That reason was wrong.** `blankOut` blanks the FIRST occurrence only and consumes only the answer's own match, so a stem using the word twice shows the veto judge the answer verbatim, and `keels` becomes `___s`. The veto prompt asks whether a candidate would ALSO fit the blank — a blank that still contains or half-shows the answer is a different question, and `#10`'s Revisions make that veto load-bearing rather than advisory.
    **So `blankOut` DELEGATES to `blankStem`.** One blanker, one leak table, one fuzz target. The committed `testdata/golden/veto-prompt.txt` has a single uninflected occurrence, so delegating moves the golden not at all — which is the cheap confirmation that this is a strictly-better implementation of the same intent rather than a behaviour change smuggled in.
  - **Future extensions:** `#13` may want the same blanking to seed a prompt; it will get the one that exists.

- **`clozeFor`** — turns a word's stored items into one question, or reports that it cannot.
  - **Relationships:** 1:1 with a due word; reads N items and picks one.
  - **THE DETERMINISM ROW LIVES HERE (PQ-1).** Done-when says *"deterministic under a fixed seed"*, and until this entity existed nothing named the seed, the option order, or which item. Three decisions, all mirroring `#7`:
    - **Which item:** the newest `FormCloze` one. `Items()` returns newest-first (a `Store` promise, held by `storetest`) and `prune` keeps the newest, so "the first one of the right form" is already the newest and needs no second rule.
    - **The seed:** `seedFor(key, day)`, exactly as `#7` seeds `optionsFor` at `play_loop.go:957` — the same word gets the same question all day and a different one tomorrow.
    - **The order:** SHUFFLED. Without it the answer is at position 1 in every question, and a learner would answer the whole sitting by pressing 1. `#7` pinned this with `TestPickOptionsMovesTheAnswerAround`; this needs the same pin, and it is the row most likely to be skipped because a question with the answer first still *looks* right.
  - **DRY rationale:** it does NOT reuse `play.PickOptions` — the options are already chosen, by `#10`, offline. This only orders them. `#10`'s recorded finding on `pickDistractors` vs `PickOptions` covers why selection is not shared; this is a third thing again and shares only the shuffle.

- **`usableItem`** — whether a stored item can be rendered as a question at all.
  - **THE CLASS, not one instance.** `sanitiseItem` neutralises on read but does not RE-VALIDATE, the README documents `items/` as inspectable, and `runAuthoring` can legitimately write an item with one surviving distractor. So the floor is enumerated rather than exemplified: a usable item has a non-empty answer, a stem that contains it (`stemUsesTheWord`), at least one distractor, and **no distractor equal to the answer** — the last is the nastiest, because it renders as two correct options with one marked wrong, which is the exact failure `#10`'s veto exists to prevent arriving by the one path the veto never sees.
  - **Future extensions:** `#13`'s `FormSentence` items have a different floor and get their own predicate rather than widening this one.

- **`wordRunEnd`** — where the word-run starting at an offset ends.
  - **DRY rationale:** `wordIndexIn` finds the START of a match and returns the length of the ANSWER, which is not the length of what must be hidden: `#10`'s author prompt permits inflection to follow (`keels`, `runs`), and blanking `keel` out of `keels` leaves `___s`. This is the other half of the same question and belongs beside it, using the same `isWordRune` predicate `highlight.go` owns.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `todaysQuestions` | `cmd/define/play_loop.go` | modified | the store and the dictionary |
| `ReviewEvent.Flagged` | `cmd/define/store/event.go` | modified | the event log |

- **`todaysQuestions`** — gains a third source: `d.deck.Items(key)`.
  - **Injected into:** nothing new; it already holds `store.Store`.
  - **THE FORM-SELECTION RULE, extended by one clause.** Today it is *"2.3 tests you, the board triages you"*: a mature word goes to the board, a young word with buildable options goes to 2.3, and a young word without goes to the board. The new sentence is **"an authored item beats a definition match"** — a young word holding a `FormCloze` item takes cloze, and everything else is unchanged.
    - **Why cloze outranks 2.3 rather than the reverse:** `#10` exists because a definition match is the weaker test. The item was authored to be a better question, and preferring the weaker one when both are available would make `#10` decoration.
    - **Why mature words still go to the board, even holding an item:** that rule is `#42`'s and was hard-won. Changing it is a different issue with its own evidence.
  - **Future extensions:** `#13`'s form joins the same rule with `FormSentence`.

- **`ReviewEvent.Flagged`** — records that the learner called a question broken, with the option set that made it broken.
  - **Injected into:** the existing capture path; nothing new is threaded.
  - **WHY IT CARRIES THE OPTIONS, when `Missed` deliberately does not.** `Missed` records the AXIS rather than the distractor's word, because *"picked larceny is a fact about one question whose option set no longer exists"*. A FLAG inverts that: the option set is the evidence, the whole point is to diagnose the question, and a flag naming no options is *"something was wrong once"*. The two fields differ because their purposes do.
  - **ARCH-SECURE:** the options are model-authored text reaching a structured YAML log. They are already neutralised at the store's write (`sanitiseItem`), which is why this can be recorded at all — but the event log is a second structured output and the plan's Task 5 checks it rather than assuming.
  - **The field goes BEFORE `At`.** `event.go` states the rule: *"At stays LAST, and a field added after it would break the torn-record rule silently."*

**Operating envelope (ARCH-CONSTRAINTS).** This form makes NO model call and NO network call, ever. It is the cheapest form in the program: the item is already on disk, selection already happened offline, the veto already ran. Per-question cost is one `Items()` read; per-sitting it is one read per due word, which is the same order as the existing `Deck()` and `Events()` reads. There is no new unbounded loop, no new concurrency, and no new failure mode reachable from a sitting.

**Test surface.** `Cloze`, `optionSet` and `blankStem` are pure and unit-tested with no store, no dictionary and no model. `play` is mechanically guarded pure (`purity_test.go`), so `Cloze` cannot import anything. The wiring is covered through `todaysQuestions` and through `run()` for the flag's keystroke, and the event field gets a `storetest` row because it is a `Store`-surface promise.

---

## Chunk 1: the form

### Task 1: extract `optionSet` from `Choice`, with no behaviour change

**Files:**
- Create: `cmd/define/play/optionset.go`, `cmd/define/play/optionset_test.go`
- Modify: `cmd/define/play/choice.go`

- [ ] **Step 1: Run `#7`'s existing suite and record it green**

Run: `go test ./cmd/define/play/ -run 'Choice|Option|Pick' -v`
Expected: PASS. **This is the refactor's oracle** — it was written against behaviour that must not move, and a refactor whose test suite changes is not a refactor.

- [ ] **Step 2: Write `optionSet` with the contract in one place**

```go
// optionSet is the numbered, digit-graded machinery both option forms share.
//
// The mechanical contract is subtle and documented across four paragraphs of
// Question's doc comment, so it lives once: options number from 1, Grade is
// asked BEFORE the reveal, the session's reserved keys are never in the answer
// set, Keys names only the digits that work, and the pick is remembered.
type optionSet struct {
	options []Option
	chosen  int // -1 until graded
}
```

- [ ] **Step 3: Embed it in `Choice`, delete the duplicated fields**

`Choice` keeps `word`, `definition`, and its own `Prompt`/`Reveal`/`Form`. Everything mechanical moves.

- [ ] **Step 4: Run `#7`'s suite again, unchanged**

Run: `go test ./cmd/define/play/ ./cmd/define/`
Expected: PASS, **with no test file edited**. If a test needed changing, the extraction changed behaviour — revert and reconsider.

- [ ] **Step 5: Commit**

```bash
git commit -m "#12: extract the option-set contract Choice and Cloze both need"
```

---

### Task 2: `Cloze`

**Files:**
- Create: `cmd/define/play/cloze.go`, `cmd/define/play/cloze_test.go`

- [ ] **Step 1: Write the failing tests for the three things that are NOT Choice**

```go
// The PROMPT is the sentence with the hole, then the words. Not the headword —
// showing it would answer the question.
// The REVEAL restores the sentence, which is the pedagogical payload: the
// learner sees the word in its context, which is the whole reason #10 authored
// a sentence rather than a definition.
// The prompt NEVER contains the answer, asserted over a table including the
// answer inflected, capitalised and appearing twice.
```

- [ ] **Step 2: Run, watch it fail to compile**

- [ ] **Step 3: Implement `Cloze` over `optionSet`**

`Word()`, `Grade()`, `Keys()` come from the embedded set. `Prompt()`, `Reveal()`, `Form()` are its own. `Form()` returns `"cloze"` — a NAME, for the reason `#40 D4a` gives: a log read years later needs no atlas to decode `cloze`, while `2.2` does.

- [ ] **Step 4: Run, then commit**

---

### Task 3: `Flagging`, the bad-question gesture

**Files:**
- Modify: `cmd/define/play/session.go` (the interface + helper), `cmd/define/play/cloze.go`
- Test: `cmd/define/play/cloze_test.go`, `cmd/define/play/session_test.go`

- [ ] **Step 1: Write the failing test for the capability, not the form**

```go
// Asked through the HELPER, exactly as missedAxis and droppedBy are. A form
// that does not flag answers false and the session never learns which forms
// can — session.go states twice that a type switch on a concrete form is what
// #6's Done-when forbids.
```

- [ ] **Step 2: Add `Flagging` beside `Missed` and `Dropping`, with the same shape**

```go
// Flagging is implemented by a form whose last keystroke said the QUESTION is
// broken, rather than that the answer was wrong.
//
// NOT A VERDICT, for the same reason Dropping is not: Verdict is what an ANSWER
// meant and Fold moves boxes by it. "This question is bad" is a statement about
// the material, and a fourth verdict would put material curation in front of
// the ladder.
type Flagging interface {
	Flagged() ([]string, bool)
}
```

- [ ] **Step 3: Enumerate the SESSION STATES `?` can arrive in (PQ-3)**

Collision is not the risk — `toInput` passes `?` through as an ordinary
`InputRune` (`play_loop.go:548-559`), so nothing else claims it. **The risk is
`session.go:242`:** once `s.Graded`, every `InputRune` advances to the next word.
So `?` pressed *after* answering would silently advance — and after answering,
having just read the reveal, is exactly when a learner discovers the question was
broken. Enumerate and pin all three:

| state | what `?` must do |
|---|---|
| before answering | flag, and advance without a verdict |
| after answering (`s.Graded`) | flag, and advance — **must not fall through to the any-key-advances branch** |
| a form that does not implement `Flagging` | nothing; the existing behaviour is unchanged |

**The precedent is written in the code that causes the problem.** `session.go`'s
own comment keeps `InputDrop` and `InputQuit` OUTSIDE the graded branch, saying
*"this word is not mine and stop are still true after a verdict, and routing
them here would silently turn a drop into a plain advance"*. A flag is true after
a verdict for the same reason and is routed the same way.

**What a flag does to the question**, stated so it is not decided by accident: it
records the flag, advances, and **scores nothing** — no `OutcomeRecord`, so no
verdict reaches `Fold` and the word neither promotes nor demotes. A broken
question is not evidence about the learner.

- [ ] **Step 4: Run, then commit**

---

## Chunk 2: the blanking

### Task 4: `blankStem`, and the leak table

**Files:**
- Create: `cmd/define/cloze.go`, `cmd/define/cloze_test.go`
- Modify: `cmd/define/harvest_item.go` (add `wordRunEnd` beside `wordIndexIn`)

- [ ] **Step 1: Write the leak table FIRST — it is this task's whole specification**

Done-when: *"the blanked sentence never leaks the answer (stem, plural, hyphenation)"*. Each row is a way it could:

```go
// | stem                                              | answer   | want
// | "The aide was sycophantic to a fault."            | sycophantic | "The aide was ___ to a fault."
// | "Sycophantic aides surrounded him."               | sycophantic | "___ aides surrounded him."          (capitalised)
// | "Shipwrights laid the keels of three frigates."   | keel        | "...laid the ___ of three frigates." (INFLECTED — the tail goes too)
// | "The keel cracked; the keel was replaced."        | keel        | both blanked                          (EVERY occurrence)
// | "Sunset over the set of the play."                | set         | "Sunset over the ___ of the play."   (substring is NOT the word)
// | "The keel's timbers rotted."                      | keel        | "The ___ timbers rotted."             (possessive)
```

**Every-occurrence is the row most likely to be missed** and the one that leaks hardest: a sentence using the word twice hands it over in full.

- [ ] **Step 2: Run, watch it fail**

- [ ] **Step 3: Implement over `wordIndexIn` + `wordRunEnd`**

`wordIndexIn` (from `#10`, already mutation-tested) finds a match on a word boundary and refuses substrings. `wordRunEnd` extends to the end of the word RUN using `highlight.go`'s `isWordRune`, which counts apostrophes and hyphens as inside a word — so the inflected tail and the possessive go with it.

**The hyphen case needs a decision, not an accident.** For answer `hot` in `"a hot-dog stand"`, the run ends after `dog`, so the whole compound is blanked. That is the conservative reading and it is CORRECT here: `___-dog` would narrow the answer to one word, which is a leak in the other direction. Record it; it is a judgement, not a fact.

- [ ] **Step 4: Assert the property, not only the table**

```go
// The property behind every row: the blanked stem, lowercased, never contains
// the answer on a word boundary. Asserted over the table AND over every stem in
// the committed corpus's items fixture, so a new fixture cannot leak silently.
```

- [ ] **Step 5: FUZZ it (PQ-7)**

The repo already has nine fuzz targets, and this is the function that earns a
tenth: its input is MODEL-AUTHORED text, its sibling `blankOut` shipped a
slice-bounds panic on `Ⱥ`/`İ` two weeks ago, and a leak here is silent — a
question that gives away its answer still renders perfectly.

```go
// FuzzBlankStem, seeded from the leak table. Three invariants:
//   - it never panics (the Ⱥ/İ class);
//   - the output is valid UTF-8;
//   - no word-boundary occurrence of the answer survives.
// The third is the leak itself, stated as a property rather than as rows.
```

- [ ] **Step 6: Run, then commit**

---

## Chunk 3: the wiring

### Task 5: `todaysQuestions` prefers an authored item

**Files:**
- Modify: `cmd/define/play_loop.go`
- Test: `cmd/define/play_loop_test.go`

- [ ] **Step 1: Write the failing tests for the RULE, not the branch**

```go
// A young word WITH an authored item gets cloze.
// A young word WITHOUT one still gets 2.3 — #7's behaviour is unchanged, which
// is the row that catches a rule written as "always cloze".
// A MATURE word still goes to the board even holding an item (#42's rule).
// An UNUSABLE item falls back to 2.3 rather than producing a broken question,
// over usableItem's whole enumeration rather than one example: no answer, a
// stem that does not contain it, no distractors, and a distractor EQUAL to the
// answer — the last renders as two correct options with one marked wrong, which
// is what #10's veto exists to prevent, arriving by the one path the veto never
// sees.
// The ANSWER MOVES: over several seeds it does not always sit at position 1.
// #7 pinned the same property with TestPickOptionsMovesTheAnswerAround, and
// without it a learner answers the sitting by pressing 1.
```

- [ ] **Step 2: Add the one clause, and the drop for an unusable item**

- [ ] **Step 3: Assert the sitting still makes NO model call**

The seam is made to PANIC, not nil — `#10`'s Done-when 1 test is the precedent and the reason: nil passes on a loop that reaches for the model behind a `!= nil` guard.

- [ ] **Step 4: Run, then commit**

---

### Task 6: the flag reaches the log

**Files:**
- Modify: `cmd/define/store/event.go`, `cmd/define/store/storetest/suite.go`, `cmd/define/play_loop.go`, `cmd/define/capture.go`
- Test: `cmd/define/play_loop_test.go`

- [ ] **Step 1: Write the failing `storetest` row**

In the SUITE, so both implementations are held to it — `#10`'s BR-2 is the precedent: a promise stated on the `Store` interface that landed in one implementation's test only, where `Mem` then diverged in the permissive direction.

- [ ] **Step 2: Add `Flagged`, BEFORE `At`**

- [ ] **Step 3: Wire the keystroke through the capability helper**

- [ ] **Step 4: Assert a flagged question is NOT also graded**

A flag says the question was broken; recording a verdict beside it would feed the ladder an answer the learner never gave.

- [ ] **Step 5: Run, then commit**

---

### Task 7: docs, and the mutation sweep

- [ ] **Step 1: README + atlas**

The README's `--play` section names three kinds of question; there are now four. The atlas needs the form-selection rule's new clause and the flag.

- [ ] **Step 2: THE MUTATION SWEEP, for THIS issue**

`#10` learned this twice: a milestone's sweep is not satisfied by another's, and a pin whose fixture cannot reach the branch is the same failure as no pin. Enumerate every property this issue states, revert each, record the named test that reddens.

- [ ] **Step 3: `sdlc close --issue 12`**

---

## Verification

- [ ] `go test ./...` green; `go vet ./...` and `go vet -tags conformance ./...` clean; `gofmt -l` clean.
- [ ] **`#7`'s suite unchanged** — the Task 1 refactor's oracle.
- [ ] Every Done-when row ticked with the mutation that proved it.
- [ ] **A real sitting, run by hand** against the batch `#10`'s checkpoint generated, to see a cloze question rendered rather than asserted.

## Revisions

### 2026-09-06 — round 1 of the plan-quality gate: 4 Important, 3 Minor

**Reason.** `sdlc change-code --issue 12` refused. Every finding was checked
against the tree before being acted on; all seven held. Ledger:
`workshop/plans/000012-vocab-form-cloze-plan-gate.md`.

**Two of the four would have caused real damage, and both were gaps rather than
errors — the plan specified the endpoints and not the path between them.**

- **PQ-2 — the flag would have DEMOTED the word.** The gesture was specified and
  the log field was specified and nothing in between. The obvious wiring is
  wrong: `CaptureReview` hardcodes `EventReviewed`, `Fold` folds every reviewed
  event, and `GradeOf(correct=false)` is `GradeWrong` — so a flag written the
  natural way marks the word wrong, which this plan's own Task 6 forbids, on an
  append-only log. The path is now named end to end and copies `OutcomeDrop`'s,
  including the reason `Fold` is safe BY CONSTRUCTION rather than by a filter.
- **PQ-3 — `?` after answering would have silently advanced.** The plan checked
  that the key collides with nothing, which was never the risk; the risk is
  `session.go:242`, where once graded every rune advances. That is the exact
  moment a learner discovers a question is broken — having just read the reveal.
  The three session states are enumerated, and the routing follows the precedent
  `InputDrop` and `InputQuit` already set for the same reason.

**PQ-1 — a Done-when row with no step.** *"Deterministic under a fixed seed"* had
nothing naming the seed, the option order, or which item. Without a shuffle the
answer sits at position 1 in every question and the sitting is answerable by
pressing 1 — a question that still LOOKS right, which is why `#7` pinned it
explicitly. `clozeFor` now owns all three decisions.

**PQ-4 — my stated reason for keeping two blankers was wrong.** I wrote that a
miss in `#10`'s `blankOut` costs the veto judge *"some context"*. It costs more
than that: it blanks the first occurrence only and consumes only the answer's own
match, so a twice-using stem shows the judge the answer verbatim and `keels`
becomes `___s` — and the veto is load-bearing. `blankOut` now delegates.
**A DRY argument decided on a wrong cost estimate is the shape to watch**: the
duplication was justified by a claim about consequences that nobody had checked.

**Minors, all taken.** The reveal carries the definition (settled at plan time
because it decides whether Task 5 keeps the dictionary lookup); `usableItem`
enumerates what a hand-edited item can break rather than naming one instance; and
`blankStem` gets a fuzz target — it is model-authored input whose sibling shipped
a slice-bounds panic on folding runes two weeks ago, and a leak here renders
perfectly while giving the answer away.
