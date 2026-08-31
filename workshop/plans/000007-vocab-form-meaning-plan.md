# Review Form 2.3 Implementation Plan (`#7`)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A review question that tests RECOGNITION — given a word, pick its definition from four — built from the learner's own deck with no model and no network, and recording WHICH option was chosen so a reduced error taxonomy falls out for free.

**Architecture:** The session needs no change: `play.Question` was designed with this form in mind and its doc comment already names it. Form 2.3 is a new pure type in `play` beside `Recall`; the option material — glosses and NOAD's own labels — is assembled in `main`, which is where the dictionary lives. The one seam that widens is the review event, because a chosen option cannot be recorded as a boolean.

**Tech Stack:** Go 1.26. No new dependency. No model, no network — that is the point of the form.

---

## Decisions

**D1 — the distractor tension is settled: SAFE AXES ONLY, and the issue's own Revisions carry the reasoning.** The far-in-meaning guard stays, so every question has exactly one defensible answer; the three distractors are chosen to VARY along axes NOAD labels itself, so a miss still says which kind. Near-synonym collapse and connotation are out of reach without semantics and belong to `#12`/`#13`, which have a model veto. Recorded in the issue on 2026-08-30 with the measurement behind it.

**D1a — THE WORD STAYS ALONE ON `Prompt`'s FIRST LINE, and that is a constraint `#38` imposes on this issue.** `#38` (parked after T0) makes `--play`'s prompt word clickable, and its whole premise is that *a recall form's prompt IS the word* (`play/recall.go:29`) — so the region is line 0, column 0, width `visibleCells(word)`. A form whose `Prompt` opens with anything else silently breaks that arithmetic in a parked issue nobody is looking at.

So `Choice.Prompt` is:

```
sycophantic
                              ← blank
1  behaving or done in an obsequious way …
2  the offence of taking property …
3  a fool or simpleton …
4  a light meal eaten in the afternoon …
```

The word alone, then the options. `#38`'s region math holds unchanged, and its T4 gains options it can mark LATER without re-deciding anything.

**Also flagged, because parking `#38` created it:** T5 edits `todaysQuestions` (`play_loop.go:233-265`), which `#38`'s T5 also rewrites. `#7` lands first, so `#38`'s plan must be RE-READ before it resumes — its "the prompt word is a region" task now meets a multi-line prompt. Recorded in `#38`'s Log at the same time as this decision, so a resumer meets it rather than discovers it.

**D2 — NOAD's labels lead the gloss, which is what makes this cheap.** Measured over the committed corpus: `"dated a manservant or valet."`, `"archaic form (something) into a mass"`, `"Grammar (of a tense or participle)…"`, `"Law historical the benefit or profit of lands…"`, `"informal, mainly North American English used…"`. So extracting one is a PREFIX match against a closed set at the head of `Sense.Gloss` (`parse.go:181-186`) — not a search, not a heuristic over prose.

**D3 — the label set is a closed table this repo may own, and `#35` already argued why.** `originLanguages` (`origin.go:27`) faced the same objection and answered it: a table restating a fact the CDN or the dictionaries own would go stale, but *"this table reads NOAD's EDITORIAL PROSE: the set of language names a dictionary writes in its etymologies, which is stable, small, and ours to read."* Register and domain labels are the same kind of fact — NOAD's own style vocabulary. A missing row costs a distractor its axis, which degrades to `general`, not to a wrong question.

**D2a — the axes are BEST-EFFORT, and the yield is measured rather than assumed.** Counted over the committed corpus, 2026-08-30: **12 of 34 entries carry a labelled sense (35%)**, and the two label families are not equally common — register is frequent (51 `informal`, plus `formal`, `archaic`, `dated`, `dialect`, `rare`, `humorous`) while domain is sparse (`Law` 4, `Grammar` 3, `Nautical` 2, `Music`/`Military`/`Computing` 1 each).

What follows, and it narrows the taxonomy a second time:

- A deck of a dozen or more will usually supply ONE labelled distractor per question, so "picked the labelled one" is reachable.
- TWO distinct axes in one question — a domain distractor AND a register one — will be uncommon, because domain labels are sparse.
- So the taxonomy this form realistically produces is **register confusion, occasional domain confusion, and did-not-know-it.** That is less than D1 implied and more than a boolean, and saying so here is the point: a decision made on a measurement should carry the measurement's limits with it.

`PickOptions` therefore fills axes in a fixed priority — domain, register, general — and falls back to `general` whenever the pool has nothing labelled. A question is never blocked on an axis being available.

**D4 — the gloss is the option text, and it already exists.** `Sense.Gloss` is a single clean line (`"the land alongside or sloping down to a river or lake"`), which is exactly what an option needs. No new rendering, no truncation policy to invent — and it keeps `Render` out of this entirely, so nothing here can affect what a lookup prints.

**D3a — the near-synonym guard has a MECHANISM, and it is the dictionary's own cross-references.** D1 said "never a near-synonym" and named no way to know. Without semantics there is no general test — but there is a specific, cheap and measured one: **NOAD defines near-synonyms by reference to each other.** `sycophantic` is glossed *"behaving or done in an obsequious way…"*. So a candidate is EXCLUDED when its headword appears in the target's gloss, or the target's headword appears in its gloss.

Measured over the corpus: this fires on 3 of 34 first-glosses, and all three are the harmless direction (`record`, `subject` and `use` contain the word `thing`) — so a minimum length and a word-boundary match keep it from excluding half the deck.

**It is a reduction, not a proof**, and the plan says so rather than implying a guarantee: two deck words can be near-synonyms NOAD never cross-references. The residual is accepted because the option set is DEFINITIONS — the Revisions' own argument that *"two different words rarely share one"* — and because the form has no model veto by design. What this mechanism removes is the case that is both most likely and most visible: the pair the learner met through each other's entry.

**D5 — the option material is assembled in `main`, never in `play`.** `play` is pure and must not import `main`: `Sense`, `Entry` and the dictionary all live there. `Recall` already set this precedent — *"Rendering happens in the caller, deliberately: Render needs RenderOpts and the dictionary, both IO-shaped, and pulling them in here would end this package's purity"* (`recall.go:19-24`). Form 2.3 takes finished options the same way.

**D4a — WHICH SENSE becomes an option is defined, because determinism and the axis both rest on it.** `bank` has a dozen senses; picking one arbitrarily makes the question non-reproducible and the axis meaningless.

- **The correct option is the target's first USABLE sense in document order** — the first block's, whenever that block has one. NOAD orders senses by centrality, so this is the meaning a learner is most likely to have met. It takes the first usable sense of ANY axis: a `rare` or `Law` sense is still what the entry leads with, and skipping to a later unlabelled one would ask about a meaning NOAD does not put first.
- **A distractor's sense is the one CARRYING the axis being sought**: for `AxisDomain`, the first sense whose gloss leads with a domain label; for `AxisRegister`, likewise; for `AxisGeneral`, **the first UNLABELLED usable sense** — across all blocks, not "the first sense of the first block".

  *Corrected 2026-08-30 (BR-12b). The original said "the first sense of the first block" for both, which is wrong for any entry whose opening sense is labelled: `defenestrate`'s general candidate is a later sense, because its first usable one is `rare`. The code was right and this text was stale — and `#12` reuses this rule, so the imprecision would have propagated.*
- A candidate that cannot supply the sought axis is not a candidate FOR THAT SLOT, which is what makes the fallback to `general` a selection outcome rather than a special case.

**D5a — `play` imports NOTHING, and the new code KEEPS it that way rather than widening the allowlist.** Measured: `go list -f '{{join .Imports}}' ./cmd/define/play` returns empty, and `play/purity_test.go:21` is `ImportsOnly(t, playPkg, []string{})` — `puretest` calls an empty allowlist *"the strongest possible version of the claim"*, and `#30` BR-41 added a dedicated fixture because `play` importing nothing was that guard's only pin. Adding `fmt`, `strings` and `math/rand` to build a form would quietly retire it.

The alternative is not asceticism — the constraint FORCES the seam D5 already asked for, which is why it is worth keeping:

| what it needs | where it goes | why no import |
|---|---|---|
| numbering the options `1`–`4` | `play` | `byte('0'+n)`, concatenated with `+`. `fmt` buys nothing for one digit |
| a seeded shuffle | `play` | a hand-rolled xorshift64, ~4 lines. **Better than `math/rand` here**: Done-when 3 claims determinism under a fixed seed, and a PRNG defined in this repo is pinned by this repo rather than by `math/rand`'s cross-version behaviour |
| the near-synonym gloss match (D3a) | **`main`** | word-boundary scanning over dictionary prose is `strings` work — and it is DICTIONARY work, which D5 already places in `main` |
| the label table (D3) | **`main`** | `readGloss` and the three `noad*Labels` tables are text parsing, sited in `cmd/define/glosslabel.go` |

So `PickOptions` receives candidates that are ALREADY filtered and labelled — gloss, headword, `Axis` — and does selection and formatting only. It never sees prose. That is the same division `Recall` uses (`play/recall.go:19-24`: already-rendered text in, no parsing), so this is the existing precedent applied rather than a new rule invented to satisfy a guard.

**Hand-rolling `strings` inside `play` would be the wrong answer** and is explicitly rejected: re-implementing a standard search to keep an allowlist empty is the guard wagging the design (ARCH-DRY). The guard stays intact because the text work MOVED, not because it was rewritten badly.

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
| `Sense.Gloss` is a single clean definition line | `parse.go:181-186` | **FALSE — corrected 2026-08-30 during T1.** Measured over the whole corpus, not three entries: `bank`, `complete`, `concrete` and `defenestrate` each carry a sense whose gloss is exactly `[with object]`, `man` carries `(plural men /men/)`, and `alewife`/`bases`/`record` carry cross-references (`another term for menhaden`). See the Revisions |
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

*Corrected 2026-08-30 after the close review (BR-2): the rows below name what
SHIPPED. The originals named five entities the tree does not have, because the
2026-08-30 design revisions renamed them and the table was never updated.*

| Name | Lives in | Status | Kind |
|------|----------|--------|------|
| `Choice` | `cmd/define/play/choice.go` | new | PURE — form 2.3, implementing `Question` |
| `Option` | `cmd/define/play/choice.go` | new | PURE — one answer line: its gloss and why it is in the set |
| `Axis` | `cmd/define/play/choice.go` | new | PURE — the reduced taxonomy: `AxisDomain`, `AxisRegister`, `AxisGeneral`, with a `numAxes` sentinel the guard derives from |
| `Candidate` | `cmd/define/play/pick.go` | new | PURE, **EXPORTED** — one possible distractor (word, gloss, axis), pre-expanded by `main`. Not in the original table |
| `PickOptions` | `cmd/define/play/pick.go` | new | PURE, **EXPORTED** — target + candidates + seed → options. Was planned as unexported `pickOptions` in `choice.go`; `main` has to call it |
| `prng` / `shuffle` | `cmd/define/play/pick.go` | new | PURE — xorshift64, and one GENERIC Fisher-Yates over it. The planned bare `shuffle` became a type plus a generic helper once the sampler and the selector both needed the same stream (ARCH-DRY) |
| `SampleStrings` | `cmd/define/play/pick.go` | new | PURE, **EXPORTED** — partial Fisher-Yates so `main` samples the deck with `play`'s PRNG rather than growing a second one. Not in the original table, and the one row that is genuinely new downstream API |
| `readGloss` / `glossFacts` | `cmd/define/glosslabel.go` | new | PURE — replaces the planned `senseLabel`. It WALKS the head of a gloss rather than matching a prefix, and returns axis, label, text and `Usable` from one pass — see the 2026-08-30 Revisions for the measurement that forced this |
| `leadingLabel` / `hasLabelPrefix` | `cmd/define/glosslabel.go` | new | PURE — longest-match on a word boundary |
| `noadDomainLabels`, `noadRegisterLabels`, `noadRegionalLabels` | `cmd/define/glosslabel.go` | new | PURE — the closed tables (D3). THREE, not the planned single `noadLabels`: regional had to be recognized in order to be scanned past without being an axis |
| `crossReferenced` / `mentions` | `cmd/define/glosslabel.go` | new | PURE — D3a's near-synonym guard, planned as `excludeCrossReferenced` |
| `optionCandidates` / `targetCandidate` / `choiceFor` | `cmd/define/optionpool.go` | new | PURE — D4a's sense selection and the per-question near-synonym filter. A file the plan did not anticipate |
| `entryDefines` | `cmd/define/optionpool.go` | new | PURE — the guard both producers call: a gloss is only ever attributed to the word whose entry defines it (BR-15/BR-17). Not anticipated by any plan row |
| `seedFor` | `cmd/define/optionpool.go` | new | PURE — FNV-1a over word + day |

- **`Choice`** — shows a word and four glosses, and remembers which was picked.
  - **Relationships:** 1:1 with a due word; holds N `Option`s (4, or fewer per D9).
  - **DRY rationale:** first of the three remaining forms (`#12`, `#13`). What it establishes — options in `Prompt`, digits in `Grade`, the chosen option surviving into the outcome — is what those inherit rather than re-invent.
  - **Future extensions:** `#12`'s cloze is the same shape with a different `Prompt`; if a form ever needs five options the count is data, not structure.

- **`pickOptions`** — the selection rule, and the whole of D1 in one function.
  - **DRY rationale:** `#12` will need the same axis-varying selection over a different pool (authored items rather than the deck), so the signature takes candidates rather than reaching for a deck. Taking finished candidates is also what lets it live in `play`: the dictionary work happens in `main` (D5) and only the RULE crosses.
  - **Future extensions:** a fourth axis when a form gains a model veto; the axis set is a type, so adding one is a row and a test.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `CaptureReview` | `cmd/define/capture.go` | modified | the event log — widened to carry the chosen axis (D7) |
| `ReviewEvent.Missed` | `cmd/define/store/event.go` | new | the field carrying the chosen axis, placed ABOVE `At` (D6) |
| `Outcome` | `cmd/define/play/session.go` | modified | carries the chosen axis out of `Apply` |
| `Missed` | `cmd/define/play/session.go` | new | **EXPORTED** — the optional capability a form implements to say WHY it was missed. `Apply` asks; form 2.1 does not implement it, because a failed recall has no kind. Missed from round 1's table patch, which is what BR-12 named |
| `todaysQuestions` | `cmd/define/play_loop.go` | modified | builds the option pool from the deck it already walks |
| `buildPool` | `cmd/define/optionpool.go` | new | the dictionary — one lookup per sampled pool word, capped at `poolCap` |
| `Question` | `cmd/define/play/question.go` | modified | gains `Keys()`, so a form describes its own answer keys. NOT anticipated by the plan: the loop's grading prompt was a const spelling form 2.1's `y`/`n`, and it was printed under form 2.3's numbered options |

**ARCH-MOCK.** No new external dependency: the dictionary is the only one and its fake (`dict_fake_test.go`) plus the committed corpus already back every form test. The deck's fake store is likewise in place. This is the form that needs neither network nor key, so the whole thing is testable with what exists — which is also `#6`'s Done-when about a session with the model seam unavailable.

**ARCH-CONSTRAINTS.** The interaction path is a keystroke and a screen redraw. The new cost is building the option pool: one dictionary lookup per POOL word, on top of the one per DUE word `--play` already does. `DCSCopyTextDefinition` is a local framework call with no network. Budget: the pool is capped at the deck words needed to fill the axes for a sitting bounded by `-count` (default 20), so worst case is a few dozen local lookups at session start — the same order as today's, which is already accepted. If a deck is large the pool is SAMPLED under the seed rather than walked whole, so cost is bounded by the cap and not by deck size. Nothing here is on the per-keystroke path.

---

## Tasks

Plain checkboxes: single-pass work with ONE boundary (AGENTS.md §3).

- [x] **T1 — `senseLabel` and the closed table** (D2, D3). Prefix match at the head of a gloss; stacked labels (`"Law historical"`) take the first; `"informal, mainly North American English"` matches `informal`. Table tests over real corpus glosses, plus the negative: a gloss that merely CONTAINS a label word later on is unlabelled.
- [x] **T2 — `Option`, `Axis`, `Choice`** (D4, D5). `Prompt` renders the word and the numbered glosses; `Grade` maps `1`–`4` and nothing else — the reserved keys are the session's (`play/question.go:74-77`). `Reveal` names the right answer. The chosen index is remembered.
- [x] **T3 — `pickOptions`** (D1). Target + candidates + seed → the correct option plus up to three distractors varying by axis, never a near-synonym. Deterministic: same seed, same set, same order. Table tests including D9's small decks.
- [x] **T4 — the choice survives into the record.** `Outcome` carries the axis, `CaptureReview` takes it, `ReviewEvent` gains a field ABOVE `At` (D6), and a correct answer writes none (D8). The torn-record test is what pins the ordering.
- [x] **T5 — wire it into `--play`.** `todaysQuestions` (`play_loop.go:233`) builds the pool and constructs `Choice` where the deck allows and `Recall` where it does not (D9). Nothing in `playSession` changes, which is the claim `#6` made and this is the first chance to test.
- [x] **T6 — docs**: `cmd/define/README.md`'s review section gains the form; `atlas/define.md` gains the selection rule and the reduced taxonomy; the project file's breakdown row ticks.

## Done when

**Every row's pin is a PREDICATE OVER BEHAVIOUR — a named test, or a grep for a property — and never "file X is unchanged".** Stated as a rule because this table got it wrong twice in one review: first pinning `play_loop_test.go` unchanged when T5 must edit it, then pinning `play/*` unchanged when T2, T3 and T4 all write there. The defect is structural, not clerical: a plan's tasks and its Done-when rows are written at different moments, so any row phrased as file-state decays the instant the task list grows. A behavioural pin cannot decay that way — it goes red for the reason it names.

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | up to four options, exactly one correct, from the deck, and no two saying the same thing | `TestPickOptionsHasOneAnswer` (the count and the single `Correct` flag) **and** `TestPickOptionsNeverRepeatsAGloss` (the duplicate-gloss half, which the first cannot see) | a set carries two `Correct` options, or two options with the same gloss — the `jalapeño`/`jalapeno` case, where one entry answers two deck keys |
| 2 | the CHOSEN option is recorded, not just correctness | `TestAMissRecordsTheAxisItChose` | the axis is collapsed to a boolean |
| 3 | deterministic under a fixed seed | `TestPickOptionsIsDeterministic`, `TestPRNGSequenceIsPinned`, `TestSeedForIsPinned` | selection reaches for map order or wall-clock, **or the PRNG/hash constants move** — the goldens are what make "ours, not the runtime's" true rather than asserted |
| 4 | degrades below four deck words | `TestPickOptionsWithATinyDeck`, `TestASittingFallsBackToRecall` | a two-word deck produces a broken question or a skipped word |
| 5 | works with the network off | `TestSittingWithNoModelAndNoNetwork` | any path here reaches the model seam |
| 6 | the axes are the reduced taxonomy, and only that | `TestEveryAxisIsSelectable` — derived from the `Axis` set, as `#30`'s registry guards are | an axis is added that nothing can select |
| 7 | the SESSION did not change to accept a second form | `TestSessionIsFormAgnostic` (`play/session_test.go:327`) green, AND no form name (`Choice`, `Recall`) appears inside `playSession` or `play.Apply` — a grep, not a file-state claim | either predicate fails |
| 9 | a SITTING is not one question repeated: the distractors vary per question | `TestPickOptionsVariesTheDistractorsAcrossASitting` — 20 targets over one pool must give ≥8 distinct distractor sets | the seed reaches only the final shuffle, so every target takes the same first-matching candidate out of the once-per-sitting pool |
| 8 | `At` stays last in the event record | the existing torn-record test green, AND `at:` is the last key of a written record — the property, not the test file's mtime | the new field is appended after `at:` |

---

## Verification before close

```bash
go test ./... && go test ./cmd/define/ -race
go test -tags conformance ./cmd/define/    # unsandboxed
```

Then, on a real terminal with a deck of a dozen words: `define --play`, answer a 2.3 question wrongly on purpose, and confirm the event log records which axis was chosen; answer one correctly and confirm no axis is written; run a sitting with the network off.

**Close:** one boundary, one `sdlc close`, one publish.

## Revisions

### 2026-08-30 — plan-quality round 1 (PQ-1…PQ-5 and a Minor)

- **PQ-1 (Critical) is a conflict I created by parking `#38` mid-flight.** T5 edits
  the same function `#38`'s T5 rewrites, and — worse — `Choice`'s multi-line
  prompt would silently break `#38`'s load-bearing premise that the prompt IS the
  word. Resolved by CONSTRAINT rather than by sequencing: the word stays alone on
  the first line (D1a), so `#38`'s region math holds unchanged. The ordering note
  is recorded in `#38`'s Log too, so a resumer meets it.
- **PQ-2 demanded a measurement the decision had skipped.** 12 of 34 corpus
  entries carry a labelled sense, and register labels vastly outnumber domain
  ones — so the taxonomy narrows a second time, to *register confusion,
  occasional domain confusion, did-not-know-it*. D2a states it. A decision made
  on a measurement has to carry that measurement's limits.
- **PQ-3 — "never a near-synonym" named no mechanism.** Now it does, and it is the
  dictionary's own cross-references: NOAD glosses near-synonyms through each other
  (`sycophantic` → *"in an obsequious way"*). Measured to fire on 3 of 34 glosses.
  Stated as a REDUCTION rather than a proof.
- **PQ-4 — which sense of a multi-sense entry becomes an option was undefined**,
  and determinism, one-correct-answer and the axis all rest on it. D4a defines it.
- **PQ-5 — Done-when 7 forbade editing the very file T5 must edit.** The claim was
  always about the SESSION, not the loop's wiring; the row now says so and names
  the form-agnostic test that actually pins it.
- **Minor: `pickOptions` moves into `play`**, which is mechanically guarded pure —
  so the determinism claim sits inside the guard that enforces it.

### 2026-08-30 — plan-quality round 2 (PQ-7, PQ-8, PQ-9)

- **PQ-7 — `play`'s purity guard allows ZERO imports, and the plan had not said
  which side of it the new code lands on.** Decided: keep the guard intact and
  move the text work to `main` (D5a). The constraint turned out to force the seam
  D5 already wanted — `pickOptions` never sees prose — so this is the `Recall`
  precedent applied rather than a concession. Hand-rolling `strings` inside `play`
  is explicitly rejected as the guard wagging the design.
- **PQ-8 — the same Done-when defect reappeared one directory up**, so it is
  fixed as a RULE above the table rather than as a third instance: a pin is a
  predicate over behaviour, never a file-unchanged claim. Rows 7 and 8 both
  rewritten; rows 1-6 were already named tests and pass the rule.
- **PQ-9 was my own breakage, and the gate caught it before the tree did.** The
  `#38` coupling note from round 1 was appended to
  `000038-play-clickable-words.md` — a filename that does not exist — creating a
  second, frontmatter-less issue file for `38` and taking `sdlc issue show 38`
  down with it. The note is now in `000038-play-clickable.md` and resolution
  works. **The rule, since the instance is trivial and the class is not:** a
  peer-issue write is not done until the peer file is read back and
  `sdlc issue show <peer>` still resolves to exactly one file. An append with `>>`
  CREATES on a typo, so it cannot fail loudly — which is exactly why round 1's
  claim that the note was "recorded where a resumer meets it" read as true while
  being false of the tree.

### 2026-08-30 — two claims the corpus disproved during T1

Both were asserted from three entries and are false over the whole corpus. Found
by dumping every parsed gloss before writing the code, which is the only reason
they were found before the form was built on them.

**D2 said labels LEAD the gloss, so extracting one is "a PREFIX match at the
head".** True of most senses, false of enough to matter — a GRAMMAR BRACKET or a
parenthetical can come first:

```
[no object] Military (of a soldier) illegally run away from military service
[with adjective] informal a book considered in terms of its readability
(the runs) informal diarrhea.
```

A prefix match written from the decision would have called every one of these
unlabelled, losing exactly the domain and register senses the axes are built
from. `readGloss` walks the head instead, stripping `[...]`, `(...)` and a
leading `/pronunciation/` until it reaches a label or the definition. Also
discovered in the same dump: NOAD stacks a REGIONAL label in front of a real one
("North American English informal a person who shows off"), so regional labels
have to be recognized in order to be scanned past — they are not an axis (the
settled taxonomy has three values), but a scanner that stopped at the first
unknown word would never reach the register label behind them.

**D4 said `Sense.Gloss` is "a single clean definition line", and the verified
table called it measured.** It was measured on `bank`, `record` and `run` — and
`bank` itself has a sense whose gloss is exactly `[with object]`. Over the whole
corpus there are three kinds of non-definition:

| shape | example | why it cannot be an option |
|---|---|---|
| a bare grammar bracket | `[with object]` | an option reading "[with object]" makes the form look broken |
| an inflection note | `(plural men /men/)`, `(evener, evenest)` | not a meaning |
| a cross-reference | `another term for menhaden`, `short for criminal record` | tests nothing about meaning; unanswerable as the ANSWER |

`readGloss` therefore returns `Usable`, and T5 must filter on it. This is a real
addition to T1's scope, not a detail: without it the first sitting on a deck
containing `bank` would show a multiple choice with `[with object]` as an option.

**Method note, since it generalises.** Both errors have the same cause — a claim
checked against the entries that came to mind rather than against the set. The
corpus is 34 files and dumping every gloss took one throwaway test. `lessons.md`
already carries *"Enumerate the category, not the instances you happened to
meet"* (#35); this is that lesson recurring in a plan's verified-claims table,
which is the one place designed to stop it.

### 2026-08-30 — close review round 1: four corrections, one of them a real defect

**BR-1 (Critical) — selection was not seeded, only ORDERING was.** `PickOptions`
took `seed` and spent it entirely on the final shuffle; both selection passes
walked the pool in fixed order, and the pool is built once per sitting. Measured
by the reviewer over a 20-word deck: **17 of 20 questions shared one distractor
set**, every day sampled. The learner could answer everything after question one
by elimination, and the recorded axis — the thing this issue was folded around —
was a choice among the same three glosses each time.

This is the same class the pool sampler was written to prevent. `SampleStrings`'
own comment says *"a deck's alphabetically first forty words would otherwise
supply every distractor forever"*; the fix removed that ACROSS sittings and I
reproduced it WITHIN one. Fixed by walking a seed-shuffled index permutation in
both passes — a permutation rather than shuffling the slice, because `choiceFor`
reuses the sitting's pool and reordering it under each question would make every
selection depend on the ones before.

**Done-when 1 and 3 were both green on the defect**, which is the more useful
finding. "Exactly one correct option" and "deterministic under a fixed seed" are
each true of a sitting that asks the same question twenty times. A new pin,
`TestPickOptionsVariesTheDistractorsAcrossASitting`, asserts the property the
two of them together do not: 20 targets over one pool must yield ≥8 distinct
distractor sets. Mutation-verified — reverting to fixed order gives 6 sets, all
of them the same three words in different orders.

**BR-2 — the Core concepts table named five entities the tree does not have.**
Corrected above. The renames were all explained in the design revisions and none
of them reached the table, so the plan asserted a surface the code never had —
including two exported types (`Candidate`, `SampleStrings`) that no plan row ever
reviewed, which is how new downstream API arrives unnoticed.

**BR-3 — Done-when 6's guard was vacuous for the property it claimed.** The
reviewer added an axis to the const block, left it out of `distractorAxes`, and
`TestEveryAxisIsSelectable` stayed green: the test's pool held only the axis
under test, and `PickOptions`' second pass fills any remaining slot from any
candidate. Reshaping the pool does not fix it — pass 2 can always reach the
candidate, by design — so the test now asserts MEMBERSHIP of `distractorAxes`
directly, derived from the `numAxes` sentinel, and keeps the behavioural check
beside it. Mutation-verified.

**BR-4 — the argument for hand-rolling the PRNG and the hash was not backed.**
D5a rejects `math/rand` because a fixed seed must reproduce a question *forever*,
"pinned by this repo's tests" — and nothing pinned the sequence. The reviewer
changed a shift constant and the FNV offset basis together and the suite stayed
green. `TestPickOptionsIsDeterministic` compared two runs inside one binary,
which `math/rand` also satisfies. Golden assertions now pin both.

**BR-5 — every seam in `optionpool.go` was untested**, including the
ARCH-CONSTRAINTS budget, which was declared and implemented and enforced by
nothing. A counting dictionary now bounds a sitting at `poolCap + count`
lookups, and asserts the pool is not silently empty either.

**BR-6 — the README's fallback threshold was off by one.** Corrected, along with
a note that a young deck gets two or three options rather than four.

### 2026-08-30 — close review round 2: the RULE behind BR-2, and four stale comments

**BR-12 is the second finding in `plan-artifact-must-match-tree`, so the
deliverable is the rule, not three more patches.**

> **A plan's entity tables and Done-when rows are DERIVED from the tree at the
> close, never hand-patched against a reviewer's enumeration.** For exported
> surface that means running `go doc -short` on each touched package and
> checking every name against the table; for pins, grepping the test names the
> rows cite; for decisions, re-reading each `D` against the function that
> implements it.

Round 1 fixed exactly the five rows BR-2 listed and left its siblings, which is
the failure mode the rule exists to end — a reviewer's list is a SAMPLE, and
patching the sample is how the same finding returns. Applied here:

```
$ go doc -short ./cmd/define/play | grep -oE '^(type|func) [A-Z][A-Za-z]*'
```

Fifteen exported names. Eleven are in the table; four are not — `Input`,
`InputKind`, `OutcomeKind` and `Session` — and those four are `#6`'s, untouched
by this window, so they correctly belong to no row here. The one real gap the
derivation found is `Missed`, now added. That is what running the check buys
over reading the diff: it distinguishes "missing" from "not mine".

**BR-12b — D4a was stale**, and the code was right. Corrected above.

**BR-12c — the round-1 pin never became a Done-when row.** The Revisions
described `TestPickOptionsVariesTheDistractorsAcrossASitting` and the table kept
eight rows, so the artifact still asserted that rows 1 and 3 covered selection —
the wording that was green on a Critical defect. Row 9 added, row 3 widened to
name the goldens.

**Five doc comments stated behaviour the code does not have** (3rd finding in
`docs-restate-behaviour-inaccurately`), so the rule there too: *a comment making
a falsifiable behavioural claim must be derived, pinned, or weakened to the true
claim.* The worst of the five was not a slip but an OVERCLAIM I had repeated in
three places — that the hand-rolled PRNG and hash make a question "reproducible
from a log indefinitely". They do not and cannot: the option set also depends on
the pool, which is the deck's membership and `LastSeen` ordering at that moment,
and the event log records none of it. The true claim — the same deck on the same
day yields the same sitting, so a restart re-asks rather than reshuffles — is
narrower, still worth the code, and now what the comments say.

### 2026-08-30 — close review round 3: a correctness bug the form's premise hid

**BR-15 (Important, and really a correctness defect) — the option material is
only valid when the entry DEFINES the prompted word.** D4 says *"the gloss is the
option text, and it already exists"* and D4a says which sense to take; both quietly
assume the looked-up word and the entry's headword are the same word. NOAD
redirects derived forms to their base: looking up `bargainer` returns the
`bargain` entry, so form 2.3 offered *"an agreement between two or more parties
as to what each party will do for the other"* as the correct answer to *"what
does bargainer mean?"*, recorded `Correct`, and promoted the word in the
schedule. One of 34 corpus entries has this shape.

**Form 2.1 was immune, which is why nothing caught it.** Recall shows the whole
rendered entry — DERIVATIVES line included — and the learner reads it as the
answer to "did you know this word". Form 2.3 asserts that ONE gloss *is* the
meaning. The same dictionary behaviour is harmless under one form and wrong under
the other, and the assumption was inherited rather than re-examined when the form
changed.

`entryDefines` gates it, and falls back to form 2.1 — the route `bases` already
takes. `Headword()` alone is not the check: the head is built from `fields[0]`
(`parse.go:436`), so it returns `hot` for `hot dog` and `a` for `a priori`, and
gating on it would send every multi-word headword to the fallback. The check
accumulates `HeadWord` plus the following `HeadOther` tokens and compares each
prefix, which reconstructs the phrases and stops at the syllable token — exactly
where `bargainer` fails.

**BR-13 closed as a SWEEP, since a list is what let it recur.** The rule from
round 2 (*a comment making a falsifiable behavioural claim must be derived,
pinned, or weakened*) had been recorded and not EXECUTED, so round 2 fixed five
named comments and left the rest. The enumeration, run over `cmd/define/`,
`atlas/` and both `#7` artifacts:

```
grep -rn "first sense of the first block|reproduc(ed|ible) from a log|four definitions|1-4 =" 
grep -rn "senseLabel|excludeCrossReferenced|noadLabels|pickOptions"
```

Four survivors it found, none of which any reviewer had listed: `choice.go`'s
package doc still said "offer four definitions"; `pick_test.go`'s new golden test
repeated the very reproducible-from-a-log claim it was written to replace;
`seedFor` said "the same four definitions"; and D5a still named `pickOptions`,
`senseLabel` and `noadLabels`. Re-running the sweep is now clean. The lesson is
narrow and worth keeping: **recording a rule is not applying it** — the round
that writes the rule must also run it, or the next round finds the instance the
rule was written to catch.

### 2026-08-30 — close review round 4: the guard was switched off the whole time

**BR-17 (Critical) — the round-3 fix was applied to the caller, so it caught the
target and not the distractors.** `choiceFor` checked `entryDefines`; `buildPool`
did not. So `bargain`'s glosses kept flowing into every question labelled
`bargainer`, and a learner picking that distractor recorded a choice whose word
and meaning came from different entries.

The class fix is not a third call site. **Both PRODUCERS of a (word, gloss) pair
now guard themselves** — `optionCandidates` and `targetCandidate` — and
`choiceFor`'s own check is deleted, because a check at the caller is precisely
what let the second path through. The rule is now true of every path rather than
of the paths a finding happened to name, and
`TestNoCandidateEverCarriesAnotherWordsGloss` states it as a property over the
whole corpus, so a fixture added later cannot reintroduce it.

**BR-18 was the fourth `plan-artifact-must-match-tree` finding, and the cause was
not carelessness — the guard for it was disabled.**
`TestPlanTablesNameEntitiesThatExist` exempts `new` rows while the plan has any
unticked `- [ ] `. The tasks were ticked in the ISSUE and not in THIS DOCUMENT,
so the exemption held through four rounds and every `new` row went unchecked
while reviewers found the drift by hand each time. Ticking the plan's own task
list made the guard fire on `shuffleOptions` within seconds.

That is the real deliverable of this round, and it supersedes round 2's rule.
Round 2 said *derive the table from the tree at close*, done by hand. The tree
already had a guard that does it. **The rule is: tick the PLAN's task list at
close — that is what arms the check — and when a reviewer names a class the repo
already guards, ask why the guard did not fire before fixing the instances.**

**And its sibling was broken outright.** `TestPlanNamedTestsExist` globbed
`cmd/define/*_test.go`, flat, so the six tests this plan pins in `play/` were
reported as nonexistent. Form 2.3's selection is pure and its tests live there BY
DESIGN — ARCH-PURE asks for exactly that arrangement — so the guard was fixed to
walk the tree. A guard that fails on the structure the architecture requires
teaches people to edit the plan until the guard shuts up, which is worse than no
guard.

### 2026-08-30 — close review round 5: the second clause, and enumerations that derive

**BR-17's second clause was the live one, and I had implemented only the first.**
The finding said *"gate on the entry defining the prompted word"* AND *"no two
options in one set may share a gloss"*. Round 4 did the first and the second
stayed reachable: `jalapeño` and `jalapeno` are separate deck keys — `store.Key`
folds case and whitespace but not diacritics — and the dictionary answers both
with the same entry, which is documented production behaviour with its own live
conformance check. Both survive `entryDefines`, `crossReferenced` cannot see it
because neither headword appears in the shared gloss, and `PickOptions` deduped
on `Word` alone. The set then carried byte-identical options with one marked
correct, so a learner who read both and picked the other was recorded as a miss,
given an axis they never chose, and had the word demoted for being right.

Fixed by deduping on gloss as well as word, seeded with the ANSWER's gloss.
`TestPickOptionsNeverRepeatsAGloss` pins it and is mutation-verified. Reverting
to word-only dedup turns it red.

**A fixture lied and the new rule exposed it.** `TestPickOptionsWithATinyDeck`
gave every candidate the gloss `"g"`, so gloss-dedup collapsed a nine-candidate
pool to two options and the test failed. The test was right and the fixture was
wrong — a pool of identical definitions cannot occur in a deck, and using one had
made the sizing assertions meaningless in a way nothing would have shown until
the rule that cares about glosses arrived.

**BR-21 — three files enumerated the fallback reasons and all three disagreed**
(code branched on three, README named two, atlas named one). 4th finding in
`docs-restate-behaviour-inaccurately`, so the rule: **a doc sentence that
ENUMERATES is a closed claim about the code and must DERIVE from the same list
the code branches on, or be written open.** Implemented, not just stated:
`fallbackReasons` is now a declared list in `optionpool.go` and
`TestREADMENamesEveryFallbackReason` checks the README against it — the same move
`doc_sync_test.go` already made for the prompt lines. The README's neighbouring
"never offered as a distractor" was corrected too: `mentions` ignores headwords
under six characters, so "never" was an overclaim.

**BR-23 — the redirect model had no live check.** `entryDefines` rests on a
belief about macOS's dictionary that only a committed fixture asserted, and a
fixture is a copy of a belief. `TestLiveDictionaryRedirectsADerivedForm` measures
it against the real dictionary, in both directions: `bargainer` must not get its
own entry, and `bargain` must still pass, so the gate cannot quietly become a ban.
