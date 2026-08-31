# Review Form 2.3 Implementation Plan (`#7`)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A review question that tests RECOGNITION — given a word, pick its definition from four — built from the learner's own deck with no model and no network, and recording WHICH option was chosen so a reduced error taxonomy falls out for free.

**Architecture:** The session needs no change: `play.Question` was designed with this form in mind and its doc comment already names it. Form 2.3 is a new pure type in `play` beside `Recall`; the option material — glosses and NOAD's own labels — is assembled in `main`, which is where the dictionary lives. The one seam that widens is the review event, because a chosen option cannot be recorded as a boolean.

**Tech Stack:** Go 1.26. No new dependency. No model, no network — that is the point of the form.

---

## Decisions

**D1 — the distractor tension is settled: SAFE AXES ONLY, and the issue's own Revisions carry the reasoning.** The far-in-meaning guard stays, so every question has exactly one defensible answer; the three distractors are chosen to VARY along axes NOAD labels itself, so a miss still says which kind. Near-synonym collapse and connotation are out of reach without semantics and belong to `#12`/`#13`, which have a model veto. Recorded in the issue on 2026-08-30 with the measurement behind it.

**D2 — NOAD's labels lead the gloss, which is what makes this cheap.** Measured over the committed corpus: `"dated a manservant or valet."`, `"archaic form (something) into a mass"`, `"Grammar (of a tense or participle)…"`, `"Law historical the benefit or profit of lands…"`, `"informal, mainly North American English used…"`. So extracting one is a PREFIX match against a closed set at the head of `Sense.Gloss` (`parse.go:181-186`) — not a search, not a heuristic over prose.

**D3 — the label set is a closed table this repo may own, and `#35` already argued why.** `originLanguages` (`origin.go:27`) faced the same objection and answered it: a table restating a fact the CDN or the dictionaries own would go stale, but *"this table reads NOAD's EDITORIAL PROSE: the set of language names a dictionary writes in its etymologies, which is stable, small, and ours to read."* Register and domain labels are the same kind of fact — NOAD's own style vocabulary. A missing row costs a distractor its axis, which degrades to `general`, not to a wrong question.

**D4 — the gloss is the option text, and it already exists.** `Sense.Gloss` is a single clean line (`"the land alongside or sloping down to a river or lake"`), which is exactly what an option needs. No new rendering, no truncation policy to invent — and it keeps `Render` out of this entirely, so nothing here can affect what a lookup prints.

**D5 — the option material is assembled in `main`, never in `play`.** `play` is pure and must not import `main`: `Sense`, `Entry` and the dictionary all live there. `Recall` already set this precedent — *"Rendering happens in the caller, deliberately: Render needs RenderOpts and the dictionary, both IO-shaped, and pulling them in here would end this package's purity"* (`recall.go:19-24`). Form 2.3 takes finished options the same way.

**D6 — `ReviewEvent` gains a field BEFORE `At`, and that ordering is load-bearing.** `event.go:32-35` states it: *"At stays LAST, and a field added after it would break the torn-record rule silently… completeness leans on a cut record losing its timestamp. A field written after `at:` would survive the cut that drops `at`, and a fragment would"* read as complete. So the new field goes above `At`, and the torn-record test is the pin that says so.

**D7 — the recorded thing is the AXIS, not the distractor's word.** Two candidates: store which word's gloss was picked, or store why that option was in the set. The axis is what `#17 M2` reads — *"picked the `Law` one"* is the finding; *"picked `larceny`"* is a fact about one question that a later reader cannot interpret without rebuilding the option set. Storing the axis also keeps the event small and stable while the deck churns underneath it.

**D8 — a correct answer records no axis.** The field is meaningful only for a miss, and writing `axis: correct` on every right answer would put a word in the log that the taxonomy then has to filter out. `omitempty` and silence.

**D9 — fewer than four deck words is a NORMAL state, not an error.** A learner three lookups in has a deck of three. The form offers what it can — a question with two options is still a question — and below two it is not a question at all, so the session falls back to form 2.1 for that word rather than skipping it. Falling back is invisible to the learner and keeps the sitting the length the schedule asked for.

---

## What this plan asserts about the existing tree, verified

| claim | verified at | status |
|---|---|---|
| `Question` is the whole of what the session knows about a form | `play/question.go:53-77` | true — and its doc names *"form 2.3's 1/2/3/4"* and *"2.3's options"* |
| a form puts unanswerable-unseen content in `Prompt` | `play/question.go:69-72` | true — so the options go there |
| the session RESERVES Enter, space, `d` and Ctrl-C | `play/question.go:74-77` | true — so the answer keys must avoid them; `1`–`4` do |
| `Sense.Gloss` is a single clean definition line | `parse.go:181-186` | true — measured on `bank`, `record`, `run` |
| labels LEAD the gloss | measured over the corpus, 2026-08-30 | true — `dated`, `archaic`, `Grammar`, `Law historical`, `informal, mainly…` |
| the deck stores no definitions | `store/word.go:20-25` | true — `Text`, `FirstSeen`, `LastSeen`, `Lookups` only, so options come from lookups |
| `--play` already looks every due word up | `play_loop.go:255-265` | true — the pool extends that walk rather than adding a mechanism |
| `Outcome` carries `Kind`, `Word`, `Verdict`, `SessionDone` | `play/session.go:80-85` | true — it must carry the choice too |
| `CaptureReview` takes a bool | `capture.go:66` | true — the seam that widens |
| `At` must stay last in `ReviewEvent` | `store/event.go:32-35` | true — stated there, with the torn-record reason |
| `Recall` takes already-rendered text to stay pure | `play/recall.go:19-24` | true — the precedent D5 follows |

---

## Core concepts

### Pure entities

| Name | Lives in | Status | Kind |
|------|----------|--------|------|
| `Choice` | `cmd/define/play/choice.go` | new | PURE — form 2.3, implementing `Question` |
| `Option` | `cmd/define/play/choice.go` | new | PURE — one answer line: its gloss and why it is in the set |
| `Axis` | `cmd/define/play/choice.go` | new | PURE — the reduced taxonomy: `AxisDomain`, `AxisRegister`, `AxisGeneral` |
| `senseLabel` | `cmd/define/glosslabel.go` | new | PURE — the leading NOAD label of a gloss, or none |
| `noadLabels` | `cmd/define/glosslabel.go` | new | PURE — the closed table (D3) |
| `pickOptions` | `cmd/define/choicebuild.go` | new | PURE — deck + target + seed → four options, deterministic |

- **`Choice`** — shows a word and four glosses, and remembers which was picked.
  - **Relationships:** 1:1 with a due word; holds N `Option`s (4, or fewer per D9).
  - **DRY rationale:** first of the three remaining forms (`#12`, `#13`). What it establishes — options in `Prompt`, digits in `Grade`, the chosen option surviving into the outcome — is what those inherit rather than re-invent.
  - **Future extensions:** `#12`'s cloze is the same shape with a different `Prompt`; if a form ever needs five options the count is data, not structure.

- **`pickOptions`** — the selection rule, and the whole of D1 in one function.
  - **DRY rationale:** `#12` will need the same axis-varying selection over a different pool (authored items rather than the deck), so the signature takes candidates rather than reaching for a deck.
  - **Future extensions:** a fourth axis when a form gains a model veto; the axis set is a type, so adding one is a row and a test.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `CaptureReview` | `cmd/define/capture.go` | modified | the event log — widened to carry the chosen axis (D7) |
| `ReviewEvent.Missed` | `cmd/define/store/event.go` | new | the field carrying the chosen axis, placed ABOVE `At` (D6) |
| `Outcome` | `cmd/define/play/session.go` | modified | carries the chosen axis out of `Apply` |
| `todaysQuestions` | `cmd/define/play_loop.go` | modified | builds the option pool from the deck it already walks |

**ARCH-MOCK.** No new external dependency: the dictionary is the only one and its fake (`dict_fake_test.go`) plus the committed corpus already back every form test. The deck's fake store is likewise in place. This is the form that needs neither network nor key, so the whole thing is testable with what exists — which is also `#6`'s Done-when about a session with the model seam unavailable.

**ARCH-CONSTRAINTS.** The interaction path is a keystroke and a screen redraw. The new cost is building the option pool: one dictionary lookup per POOL word, on top of the one per DUE word `--play` already does. `DCSCopyTextDefinition` is a local framework call with no network. Budget: the pool is capped at the deck words needed to fill the axes for a sitting bounded by `-count` (default 20), so worst case is a few dozen local lookups at session start — the same order as today's, which is already accepted. If a deck is large the pool is SAMPLED under the seed rather than walked whole, so cost is bounded by the cap and not by deck size. Nothing here is on the per-keystroke path.

---

## Tasks

Plain checkboxes: single-pass work with ONE boundary (AGENTS.md §3).

- [ ] **T1 — `senseLabel` and the closed table** (D2, D3). Prefix match at the head of a gloss; stacked labels (`"Law historical"`) take the first; `"informal, mainly North American English"` matches `informal`. Table tests over real corpus glosses, plus the negative: a gloss that merely CONTAINS a label word later on is unlabelled.
- [ ] **T2 — `Option`, `Axis`, `Choice`** (D4, D5). `Prompt` renders the word and the numbered glosses; `Grade` maps `1`–`4` and nothing else — the reserved keys are the session's (`play/question.go:74-77`). `Reveal` names the right answer. The chosen index is remembered.
- [ ] **T3 — `pickOptions`** (D1). Target + candidates + seed → the correct option plus up to three distractors varying by axis, never a near-synonym. Deterministic: same seed, same set, same order. Table tests including D9's small decks.
- [ ] **T4 — the choice survives into the record.** `Outcome` carries the axis, `CaptureReview` takes it, `ReviewEvent` gains a field ABOVE `At` (D6), and a correct answer writes none (D8). The torn-record test is what pins the ordering.
- [ ] **T5 — wire it into `--play`.** `todaysQuestions` (`play_loop.go:233`) builds the pool and constructs `Choice` where the deck allows and `Recall` where it does not (D9). Nothing in `playSession` changes, which is the claim `#6` made and this is the first chance to test.
- [ ] **T6 — docs**: `cmd/define/README.md`'s review section gains the form; `atlas/define.md` gains the selection rule and the reduced taxonomy; the project file's breakdown row ticks.

## Done when

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | four options, exactly one correct, from the deck | `TestPickOptionsHasOneAnswer` | a second option's gloss is the target's |
| 2 | the CHOSEN option is recorded, not just correctness | `TestAMissRecordsTheAxisItChose` | the axis is collapsed to a boolean |
| 3 | deterministic under a fixed seed | `TestPickOptionsIsDeterministic` | selection reaches for map order or wall-clock |
| 4 | degrades below four deck words | `TestPickOptionsWithATinyDeck`, `TestASittingFallsBackToRecall` | a two-word deck produces a broken question or a skipped word |
| 5 | works with the network off | `TestSittingWithNoModelAndNoNetwork` | any path here reaches the model seam |
| 6 | the axes are the reduced taxonomy, and only that | `TestEveryAxisIsSelectable` — derived from the `Axis` set, as `#30`'s registry guards are | an axis is added that nothing can select |
| 7 | the session did NOT change to accept a second form | `play_loop_test.go` unchanged for this issue | `playSession` grows a case for a form |
| 8 | `At` stays last in the event record | the existing torn-record test, unchanged | the new field is appended after `at:` |

---

## Verification before close

```bash
go test ./... && go test ./cmd/define/ -race
go test -tags conformance ./cmd/define/    # unsandboxed
```

Then, on a real terminal with a deck of a dozen words: `define --play`, answer a 2.3 question wrongly on purpose, and confirm the event log records which axis was chosen; answer one correctly and confirm no axis is written; run a sitting with the network off.

**Close:** one boundary, one `sdlc close`, one publish.
