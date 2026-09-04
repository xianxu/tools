# Authored Practice Items Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `define --harvest` writes finished practice material to disk ahead of time — a CEFR band and a domain per word, and stems the model authors — so a review sitting stays instant, free and offline, and `#12`/`#13` have something worth consuming.

**Architecture:** Two milestones with a real stop between them. **M1** is the cheap durable half: a band and a domain per word, assigned once and cached forever, behind a new store surface that follows `words/` and `events/` exactly. **M2** is the expensive judged half: authored stems, and distractors SELECTED from the deck with a model vetoing any that would also fit. The model is reached only through `#11`'s `llm.Task[T]`, so every task is a typed unit with a golden and a wire fake; nothing here invents a second transport.

**Tech Stack:** Go. `cmd/define/store` (the artifact), `internal/llm` (the transport, unchanged), `cmd/define` (the `--harvest` mode). No new dependencies.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Band` | `cmd/define/store/vocab.go` | new |
| `Domain` | `cmd/define/store/vocab.go` | new |
| `WordFacts` | `cmd/define/store/item.go` | new |
| `Item` | `cmd/define/store/item.go` | new |
| `parseLearnerBand` | `cmd/define/usermodel.go` | new |
| `agreement` | `cmd/define/harvest_band.go` | new |
| `pickDistractors` | `cmd/define/harvest_item.go` | new |
| `topicSpread` | `cmd/define/harvest_judge.go` | new |
| `prune` | `cmd/define/store/item.go` | new |

- **`Band`** — a CEFR level, `A1`…`C2`, as a string type with a parse that REFUSES anything else.
  - **DRY rationale, corrected (PQ-1).** The earlier draft said this "reuses" a scale `#17` already assigns. **That was false and the correction is load-bearing.** `#17`'s `levelClaim.Band` is a bare `string` (`cmd/define/reflect.go:128`) with no CEFR validation, and it reaches disk as prose under `## Level`; `Store.UserModel()` returns the whole file as raw markdown (`cmd/define/store/store.go:28`) and its only consumer pastes it into a prompt (`cmd/define/ask.go:268`). There is no typed learner band in the tree. So a `store.Band` added beside that would BE the parallel spelling the rationale claimed to avoid.
  - **Therefore the single source is enforced, not asserted (ARCH-PURPOSE).** Two consumers derive from `Band` in THIS issue, because single-sourcing is not done until the motivating consumer derives:
    1. **`#17` writes through it.** `--reflect` parses the model's band claim through `ParseBand`. An unparseable band DROPS the level claim, which is the behaviour `levelClaim` already documents for evidence words that aren't in the deck ("is dropped if none of them exist") — the same rule, one member further out.
    2. **Authoring reads it back through it.** See `parseLearnerBand` below.
  - **Future extensions:** if a downloadable CEFR asset ever arrives (the issue DEFERS rather than rejects one), it produces `Band` values and nothing else changes.

- **`Domain`** — a subject field, drawn from a CLOSED set, with `general` as the explicit fallback.
  - **The set is `noadDomainLabels`, and it is not a third vocabulary (PQ-4).** The tree already holds two: `noadDomainLabels` (`cmd/define/glosslabel.go:47`), a closed 37-row table this repo owns and reads off NOAD's editorial prose, and `#17`'s `domainClaim.Name` (`cmd/define/reflect.go:143`), free model text that renders as `law`, `business news` in the committed golden. Adding a third — an open string the model fills — is what would let `Medicine`/`medicine`/`med` count as three in `topicSpread`, which is the ONE measure taken with no model and therefore the one that must not be inflatable.
  - **A domain is mostly not a model call at all.** `readGloss` (`cmd/define/glosslabel.go:115`) already yields `Label` + `Axis` per sense; a word whose NOAD entry carries a field label gets its domain from the DICTIONARY, offline and free. The model is asked only when the dictionary supplies none, and its answer is parsed through the same closed set — refused values fall back to `general` rather than widening the vocabulary.
  - **Where the set lives.** The persisted vocabulary belongs to `store`, because the store is what decides which values may be written. `glosslabel.go` keeps its NOAD-specific SCANNING concerns — longest-first ordering, NOAD's capitalization — but derives its label list from `store`'s set instead of restating it. That respects D5 (prose parsing stays on the dictionary side of the seam) while leaving one source for what a `Domain` may be.
  - **The learner-side join is a pure fold.** `#17`'s free-text domain claims map onto the closed set case-insensitively; an unmapped claim is IGNORED, not an error, and not a new value. Recorded because it is a real narrowing: a learner domain NOAD has no label for ("business news") contributes nothing to selection until someone decides it deserves a row.

- **`parseLearnerBand`** — reads the learner's CEFR band back out of the learner model, purely, from the markdown `UserModel()` already returns.
  - **Why a frontmatter field rather than a second artifact.** `renderUserModel` emits `level: <band>` beside `type:`/`updated:`, and this reads it. One file, no second thing to drift; the human-facing `## Level` prose is untouched. `renderUserModel` is already "where the file's structure is created", which is the comment `sanitiseModel` cites for doing its work there — a machine-readable field belongs in the same place.
  - **Absent or unparseable means GENERIC, never an error** — the Spec's rule for an absent learner model, and it covers every learner-model file written before this issue.

- **`WordFacts`** — what is cached per word forever: `Band`, `Domain`, and `At`.
  - **Relationships:** 1:1 with a deck word, keyed by `store.Key` like `words/`.
  - **DRY rationale:** one artifact rather than two caches, because they are assigned in one call and expire together (never).

- **`Item`** — a finished practice item: `Word`, `Stem`, `Answer`, `Distractors`, `At`, and the `Form` it was authored for.
  - **Relationships:** 1:N with a word — a word may hold several items, and `#12` picks among them.
  - **Future extensions:** `#13`'s free-sentence form wants a different shape; `Form` is the field that lets one store hold both rather than a second store appearing beside this one.

- **`agreement`** — over N repeated band assignments for one word, the fraction that agree with the modal answer. **This is the measured claim M1 makes**, and it is deliberately narrow — see "What the measure does not say" below.
  - **It runs on a SAMPLE, in its own mode, and never on the cached path (PQ-5).** The earlier draft asked `--harvest` to print an agreement it measured while also promising one call per unbanded word and zero calls on a second run; those cannot all hold. Resolved by separating them: `--harvest --agreement[=N]` re-asks a fixed sample of K already-banded words N times and **writes nothing**. Default off. The ordinary path keeps its one-call-per-unbanded-word cost and its zero-call second run, both untouched.
  - **The unit test measures the ARITHMETIC, not the model.** `agreement` is a pure function over a slice of bands, table-tested on synthetic input — a fake "seeded to vary" would only report how the fake was seeded. The floor (`≥ 0.8` over `N=5`, `K=20`) is asserted in the **conformance** row, against the real service, which is the only place the number means anything.

- **`pickDistractors`** — same domain (or general vocabulary), at the learner's band or one below, never the answer. Pure over an already-banded candidate list.
  - **DRY rationale:** `play.PickOptions` already selects options at review time from a live pool; this is its authoring-time counterpart and must NOT become a second copy of it. The plan's Task 5 Step 3 checks whether the two can share.
  - **This MOVES ownership from `#12`, and the move is declared (PQ-3).** `#12`'s Done-when rows own "options are drawn from the pool + deck, never model-generated", the `sycophantic`/`obsequious` near-synonym rejection, and "the form works with the LLM seam unavailable (veto step skipped)"; `internal/llm/golden_schema_test.go:11` records the veto verdict type as "#12 will define its own". Building selection and the veto here takes all of that. **What happens to `#12`:** its rows are not deleted, they are SATISFIED BY CONSTRUCTION and rewritten to say so — options selected offline from the banded deck are never model-generated in a stronger sense than review-time filtering achieved, and a form reading finished items from disk works with the seam unavailable because it never reaches for it. `#12` keeps the rendering job: blanking a stem without leaking the answer, and determinism under a seed. The near-synonym case moves here as the veto's committed known-bad row. Recorded as a Revision on `#12` in the same round as this plan, and the `golden_schema_test.go` comment is corrected to name `#10`.

- **`topicSpread`** — distinct domains over a batch of items, as a fraction. **Measured with no model at all**, which is what Done-when 5 asks for: a judge scoring its own batch's variety is the oracle problem the atlas already records.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Store` | `cmd/define/store/store.go` | modified | the working directory |
| `bandTask` / `authorTask` / `vetoTask` | `cmd/define/harvest_*.go` | new | the model, via `llm.Task[T]` |
| `runHarvest` | `cmd/define/harvest.go` | new | the batch mode |

- **`Store`** — gains four methods (`WordFacts`/`SetWordFacts`, `Items`/`SetItems`), following `NewsItems`/`SetNewsItems` exactly.
  - **Injected into:** every consumer already takes `store.Store`; nothing new is threaded.
  - **ARCH-MOCK:** `store.Mem` implements them and `storetest/suite.go` gains rows, so the fake and the YAML store are held to one contract — the suite exists for precisely this and a new surface that skipped it would be the gap `#16 M2` BR-45 closed.
  - **The new directory joins `RuntimeDirs`**, which every guard, migration and test derives from. A hand-typed name would be the second source that comment warns about.
  - **PER-LANGUAGE: `facts/<lang>/<key>.yaml`, like `words/` (PQ-2).** `cmd/define/store/yaml.go:126` states this repo's rule and the dividing line is DERIVATION, not storage — `words/` and the learner model are scoped, `events/` and `usage/` are flat, and the comment records that a shared learner model is what a Spanish `--reflect` overwriting the English one cost (`#23`). A band and a domain are derived from the word IN A LANGUAGE: `red`, `once`, `actual` and `sensible` are all real words in English and Spanish with different bands and different meanings. Flat would collide them — and because facts are cached FOREVER and a second write REPLACES, the collision is permanent and unwinding it is a migration. `#18` (Spanish) is open, so this is a live path, not a hypothetical. `factsDir()` mirrors `wordsDir()` and goes through the same atomic-write helper.
  - **Model text is neutralised in ONE pass over the struct, at the write (PQ-6).** `sanitiseFacts` and `sanitiseItem` sit where the artifact's structure is created, which is exactly the placement `sanitiseModel` (`cmd/define/usermodel.go:213`) argues for and the reason it exists: two earlier rounds sanitised the fields a finding happened to list and missed the ones it didn't. This issue persists four new free-text model fields — `Item.Stem`, `Answer`, `Distractors`, and a `Domain` that fell back to the model — and later renders them onto the board, so per-site calls would be the same list-that-drifts failure a third time. **`Band` and `Domain` are additionally parse-refusing**, which is a narrower guarantee than neutralisation and not a substitute for it.
  - **A truncated or hand-edited `facts/*.yaml` degrades to UNBANDED (ARCH-SECURE).** The refusing parse rejects the file's band, the word is treated as never harvested, and the next `--harvest` re-asks. No crash, and no half-trusted record: a facts file that will not parse is worth exactly as much as an absent one.

- **`bandTask` / `authorTask` / `vetoTask`** — three typed `llm.Task[T]`s.
  - **Injected into:** `llm.Run`, through `deps.newLLM` — the seam `#11` built and `#16` already uses.
  - **ARCH-MOCK:** each gets a `llmtest` golden (the prompt is pinned) and runs against `llmtest.Fake` in unit tests; the live conformance check follows `#11`'s existing `-tags conformance` pattern, so drift between the fake and the real service is detected rather than assumed.

- **`runHarvest`** — the batch mode, dispatched like `--forget` and `--llm-check`: a MODE, validated apart from the argument count.
  - **ARCH-CONSTRAINTS:** see the envelope below.

**Operating envelope (ARCH-CONSTRAINTS).** `--harvest` is a BATCH path and the only one in this program that may block: a sitting must never wait on it, which Done-when 1 states and a test enforces by driving a sitting with the model seam made to panic.
- *Latency:* bounded by `--limit` (below), not "by the deck". One band call per unbanded word **that the dictionary could not already supply a domain for**, one author call per item.
- *Scale:* a deck of a few thousand. Work is per NEW word, so a second run over an unchanged deck makes ZERO model calls — that is Done-when 2 and it is what keeps the cost from growing with time.
- *A per-run bound, stated rather than implied.* `--limit` caps the words one `--harvest` run will ask about, defaulting to 200. `poolCap = 40` (`cmd/define/optionpool.go:24`) is this repo's own precedent for bounding batch work with a reason attached; "bounded in practice by the deck" was not a bound, and the first run against a deck of thousands is precisely where an unbounded loop is discovered. Harvesting is resumable by construction — the cache is the progress marker — so a capped run is a partial run, not a failed one.
- *Overload:* a model outage leaves the store untouched and harvesting stops (Done-when 6). `#19` is open on `ErrRequest` mis-classification and is a real risk here, since this is the first path that makes many calls in a row.
- *Disk:* bounded by `prune`, deterministic, tested (Done-when 7).

**Test surface.** `Band`, `agreement`, `pickDistractors`, `topicSpread` and `prune` are pure and unit-tested with no model and no disk. The store surface is covered by `storetest`, so `Mem` and `YAML` cannot diverge. The three tasks have goldens plus fake-driven tests, and a live conformance row each.

---

## Chunk 1: M1 — a band and a domain per word

### Task 1: the artifact and its store surface

**Files:**
- Create: `cmd/define/store/vocab.go`, `cmd/define/store/vocab_test.go` (`Band`, `Domain` and their refusing parses)
- Create: `cmd/define/store/item.go`, `cmd/define/store/item_test.go`
- Modify: `cmd/define/store/store.go` (interface), `mem.go`, `yaml.go` (incl. `RuntimeDirs` + `factsDir`), `storetest/suite.go`
- Modify: `cmd/define/glosslabel.go` — `noadDomainLabels` DERIVES from `store`'s closed set rather than restating it (ARCH-DRY; the ordering and NOAD's capitalization stay here, the vocabulary does not)

- [x] **Step 1: Write the failing conformance rows**

In `storetest/suite.go`, so BOTH implementations are held to them at once:

```go
// Absent facts are not an error — the normal state of every word before the
// first harvest, exactly as an absent user model is (#16 M2).
// Written facts read back identically.
// A second write REPLACES rather than merging: a band is assigned once, and a
// re-harvest that changed one would break the cache's whole premise.
// FACTS ARE PER-LANGUAGE: the same key written under two languages yields two
// records, neither overwriting the other. `red` is a word in English and in
// Spanish, with different bands and unrelated meanings — and because facts are
// cached forever and replaced rather than merged, a collision here is permanent.
// This is yaml.go's derivation rule (#23), one member further out.
// A facts file that will not parse reads as ABSENT, not as an error: a band
// that fails ParseBand leaves the word unbanded and re-harvestable.
```

- [x] **Step 2: Run against `Mem` and watch it fail to compile**

Run: `go test ./cmd/define/store/...`
Expected: FAIL — `WordFacts` undefined.

- [x] **Step 3: Define the types, parse-refusing on `Band` and `Domain`**

`Band` is a string type whose parse refuses anything outside `A1`…`C2`. **A model
returning `B2+` or `intermediate` is a real answer to a badly-posed question**,
and accepting it would put an unorderable value into arithmetic that compares
bands. Refuse at the boundary, count the refusal, leave the word unbanded.

`Domain` is the same shape over the CLOSED set, with `general` as the fallback an
unrecognised answer lands on rather than widening the vocabulary. Then make
`glosslabel.go` derive its label list from this set — the single-source move is
not done while a second copy of the vocabulary sits in `main` (ARCH-PURPOSE), and
that file's ordering + capitalization concerns stay where they are.

- [x] **Step 4: Implement on `Mem`, then `YAML`**

`YAML` writes `facts/<lang>/<key>.yaml` through the SAME atomic-write helper
`words/` uses — the shadow-file prefix is already single-sourced, and this must
not become a second writer. `factsDir()` mirrors `wordsDir()`; add `"facts"` to
`RuntimeDirs`.

**`"facts"` goes at the END of `RuntimeDirs` (`store/yaml.go:53`).** `wordsDir`,
`eventsDir` and `usageDir` index that slice POSITIONALLY at `[0]`/`[1]`/`[2]`;
inserting anywhere but the tail silently repoints three directories at each
other. And `TestGitignoreCoversRuntimeDirs` (`repo_guard_test.go`) will fail
until `.gitignore` gains the matching row — that is the guard working, and the
loop `RuntimeDirs`' own comment says has now cost four review rounds across
three issues.

`sanitiseFacts` runs on the write, one pass over the struct, for the reason
`sanitiseModel` gives at `usermodel.go:213`.

- [x] **Step 5: Run, then commit**

```bash
go test ./cmd/define/store/... && go test ./cmd/define/...
git commit -m "#10 M1: a band and a domain per word, cached like the deck"
```

---

### Task 2: the band task, and the stability measure

**Files:**
- Create: `cmd/define/harvest_band.go`, `cmd/define/harvest_band_test.go`
- Create: golden at `cmd/define/testdata/golden/band-prompt.txt` — `llmtest.AssertGolden(t, "testdata", …)` writes into the CALLING package, which is where `askctx_test.go:29` and `reflectprompt_test.go:24` land theirs. `internal/llm/llmtest/testdata/` holds `llmtest`'s own wire fixtures (`stream-sample.sse`, `message-schema.json`) and is not the place for a prompt golden.
- Modify: `cmd/define/reflect.go` + `cmd/define/usermodel.go` — `#17` writes its band THROUGH `store.ParseBand`, and `renderUserModel` emits `level:` into the frontmatter for `parseLearnerBand` to read back

- [x] **Step 0: Close the learner-band loop, so `Band` has two deriving consumers**

The Critical finding of round 1, and it comes first because `pickDistractors`
cannot be written until the learner's band has a source.

- `--reflect` parses its band claim through `store.ParseBand`; unparseable DROPS
  the level claim, matching `levelClaim`'s existing rule for evidence words that
  aren't in the deck.
- `renderUserModel` emits `level: <band>` in the frontmatter, beside `type:` and
  `updated:` — the same function that neutralises model text, because it is the
  one place this file's structure is made.
- `parseLearnerBand` reads it back out of the markdown `UserModel()` already
  returns. Pure, table-tested, and **absent or unparseable means generic
  authoring** — which is also every learner-model file written before this issue.
- The `user-model.golden.md` diff is the artifact that proves the field landed.

- [x] **Step 1: Write `agreement` and its table test**

Pure, and worth its own test before any model exists:

```go
// agreement is the fraction of N assignments that match the modal band.
// 1.0 is perfect stability; 1/N is noise.
//
// THE MODE, not the first answer: "what does this model usually say" is the
// question the cache's premise rests on, and anchoring on a single run would
// measure that run's luck.
//
// Table-tested over synthetic []Band. NOT over a fake seeded to vary: that
// measures how the fake was seeded, which is a fact about the test. The number
// only means something against the real service, so the FLOOR lives in the
// conformance row (Step 4) and this test owns the arithmetic.
```

- [x] **Step 2: Write the task, with the golden**

```go
// bandTask asks for a CEFR band and a domain in ONE call, because they are one
// judgement about the word and two calls would double the cost of the only
// per-word work this issue does.
//
// The DOMAIN half is skipped when readGloss already supplied one: NOAD labels a
// specialist sense itself, offline and free, and asking a model to re-derive a
// fact the dictionary printed is the call this issue should not make. The model
// answers for words NOAD leaves unlabelled, and its answer is parsed through the
// same closed set — unrecognised lands on `general` rather than widening it.
```

- [x] **Step 3: The measure, in its own mode**

`--harvest --agreement[=N]` re-asks a fixed sample of K already-banded words N
times and **writes nothing** (defaults `N=5`, `K=20`; off unless asked for).

**Why a separate mode rather than a number `--harvest` prints.** The two claims
in the envelope — one call per unbanded word, and zero calls on a second run —
are incompatible with measuring N assignments on the same pass. Splitting them
keeps both properties true of the path that runs every day, and puts the cost of
measurement where someone is choosing to pay it.

```go
// STABILITY, NOT CORRECTNESS, and the distinction is the operator's decision of
// 2026-09-04 recorded on the issue. This measures the property the cache
// depends on — a band assigned once and reused forever must be the band this
// model usually gives — and it CANNOT detect a model that is confidently and
// consistently wrong. Every downstream use of a band rests on the scale being
// right, so the number this reports is agreement, and it is reported as that
// with no claim about correctness.
```

- [x] **Step 4: A live conformance row, and the floor**

Follow `#11`'s existing `-tags conformance` pattern. **The floor is `agreement ≥
0.8` over `N=5` on the sample**, asserted here rather than in the unit test —
against the real service is the only place the fraction reports anything about
the model. A binary "same band twice" would not have been the measure Done-when 3
asks for. This is also the only check that can see the fake and the service
disagreeing about the shape of the answer.

- [x] **Step 5: Run, then commit**

---

### Task 3: `--harvest`, as a mode

**Files:**
- Create: `cmd/define/harvest.go`, `cmd/define/harvest_test.go`
- Modify: `cmd/define/main.go` (flag + dispatch), `cmd/define/README.md`

- [x] **Step 1: Write the failing test for Done-when 1 and 2**

```go
// A SITTING NEVER WAITS ON HARVESTING. Driven with the model seam made to PANIC,
// not nil: nil passes on a loop that reaches for the model behind a `!= nil`
// guard, which is how a network dependency creeps into an offline path.
//
// AND A SECOND RUN MAKES NO CALLS. The counting fake asserts zero — "cached
// forever" is a claim about calls, not about the file existing.
```

- [x] **Step 2: Dispatch it as a mode, with its bound**

Beside `--forget` and `--llm-check`, which are validated apart from the argument
count. Reuse that path rather than adding a fourth shape.

`--limit` (default 200) caps the words one run asks about, and `--agreement[=N]`
selects the measurement mode from Task 2. A test asserts the cap holds on a deck
larger than it — `poolCap` has one for the same reason.

- [x] **Step 3: Outage leaves the store untouched (Done-when 6)**

Fail the fake mid-batch; assert the words banded before the failure are intact and
nothing partial was written. The atomic write already gives this — the test is
what makes it a property rather than an accident.

- [x] **Step 4: README + atlas, then close M1**

```bash
sdlc milestone-close --issue 10 --milestone M1
```

---

## Chunk 2: M2 — authored items, and the veto

**STOP AT THE END OF THIS CHUNK.** Read real generated items before `#12` and
`#13` consume them — the project's own instruction, and the reason this issue is
milestoned.

### Task 4: authored stems

**Files:**
- Create: `cmd/define/harvest_item.go`, `cmd/define/harvest_item_test.go`

- [ ] **Step 1: The task, learner-aware**

Reads `#17`'s `user-model.md` through `store.UserModel()` — **an absent file means
generic authoring, not an error**, which is the normal first-run state. The band
comes through `parseLearnerBand` (Task 2 Step 0), the domains through the
case-insensitive fold onto the closed set; an unmapped learner domain is ignored
rather than becoming a new value.

Two constraints go in the prompt as REQUIREMENTS, because a model asked for a
natural sentence drifts to the neutral and unnamed:
- the stem must ENTAIL its answer;
- it must name real people, places or institutions.

- [ ] **Step 2: `topicSpread`, measured with no model**

Done-when 5's second half. A judge scoring its own batch's variety is the
self-oracle problem `atlas/define.md` already records under "Its own oracle" —
distinct domains over a batch is arithmetic and cannot flatter itself.

**The arithmetic only holds because `Domain` is closed.** Counting distinct
values over free model text would let `Medicine`, `medicine` and `med` report a
spread of three where there is one — the measure taken with no model is exactly
the one nothing else can catch being wrong. A test asserts that casing and
whitespace variants collapse.

- [ ] **Step 3: The entailment judge, on a batch**

An `llm.Task` scoring a batch, with a COMMITTED known-bad stem that must be
rejected — the same shape Done-when 4 asks of the veto, and the only thing that
makes a judge's pass meaningful.

---

### Task 5: distractors SELECTED, and vetoed

**Files:**
- Create: `cmd/define/harvest_judge.go`, `cmd/define/harvest_judge_test.go`

- [ ] **Step 0: Declare the move from `#12`, in writing, before building it**

This task takes three things `#12`'s Done-when currently owns. Do the paperwork
first, so the ownership is recorded at the moment it moves rather than discovered
at `#12`'s close:

- Append a `## Revisions` entry to `workshop/issues/000012-vocab-form-cloze.md`
  rewriting its rows: "options drawn from the pool + deck, never model-generated"
  and "works with the LLM seam unavailable" become **satisfied by construction**
  (an item read from disk carries finished options and never reaches for a
  model), and `#12` keeps the rendering job — blanking without leaking the
  answer, determinism under a seed.
- The `sycophantic`/`obsequious` near-synonym case moves HERE, as Step 1's
  committed known-bad row. It is the same assertion at a different time.
- Correct `internal/llm/golden_schema_test.go:11`, which says the veto verdict
  type is one "#12 will define its own". `#10` defines it.

- [ ] **Step 1: A known-bad case, committed**

Done-when 4 names this precisely: *"the veto is exercised by a committed
known-bad case"*. A veto that has never rejected anything is a veto nobody has
seen work.

- [ ] **Step 2: Selection at the learner's band or one below**

One band BELOW rather than above: a distractor the learner does not know is
unrejectable — they eliminate it by ignorance rather than by knowing it does not
fit.

- [ ] **Step 3: Check `play.PickOptions` for reuse (ARCH-DRY)**

`#7` already selects options from a live pool at review time. **Before writing a
second selector, establish whether these are one function with two callers or two
genuinely different rules** — and write the answer down either way. Authoring
selects from BANDED candidates offline and review selects from a rendered pool,
which may be different enough; that is a finding to record, not to assume.

---

### Task 6: pruning, docs, and the stop

- [ ] **Step 1: `prune`, deterministic and bounded (Done-when 7)**

Deterministic means the same store prunes to the same result — tested by pruning
twice, not by inspecting one run.

- [ ] **Step 2: README + atlas**

- [ ] **Step 3: GENERATE A REAL BATCH AND READ IT**

The checkpoint. Not a test: run `--harvest` against the real model on the real
deck, and read the items. The question is the project's central one — *how good
can the material get* — and no green suite answers it.

- [ ] **Step 4: `sdlc close --issue 10`**

---

## Verification

- [ ] `go test ./...` green; `go vet ./...` and `gofmt -l` clean.
- [ ] `go test -tags conformance ./...` green — the band task's live row is new, and `#11`'s existing rows must still pass.
- [ ] Every Done-when row ticked with the mutation that proved it — revert the code, watch the named test redden (`workshop/lessons.md`, "A pin that cannot fail is not a pin").
- [ ] **The generated batch, read by the operator.** The one row no test replaces.

---

## Revisions

### 2026-09-04 — round 1 of the plan-quality gate: 1 Critical, 5 Important, 2 Minor

**Reason.** `sdlc change-code --issue 10` refused. Every finding was checked
against the tree before being acted on; all eight held, so none was withdrawn.
Ledger: `workshop/plans/000010-vocab-harvest-plan-gate.md`.

**Deltas.**

- **PQ-1 (Critical) — `Band`'s DRY rationale rested on a false claim.** The draft
  said the type "reuses" a scale `#17` already assigns. It does not: `#17`'s band
  is a bare `string` that reaches disk as prose, and no typed learner band exists
  in the tree — so `store.Band` would have BEEN the parallel spelling. Fixed by
  making the single source enforced rather than asserted (ARCH-PURPOSE): `#17`
  now writes its band through `ParseBand`, `renderUserModel` emits a `level:`
  frontmatter field, and `parseLearnerBand` reads it back. That also supplies
  `pickDistractors`' second input, which previously had no source at all.
- **PQ-2 (Important) — `facts/` is per-language.** `yaml.go:126` states the rule:
  per-language iff DERIVED from the language-scoped deck. A band is. `red`,
  `once`, `actual` and `sensible` collide EN↔ES, and "cached forever + second
  write replaces" would make the collision permanent.
- **PQ-3 (Important) — the move from `#12` is declared.** Selection and the veto
  are three of `#12`'s Done-when rows. Task 5 gains a Step 0 that rewrites them
  as satisfied-by-construction, moves the `sycophantic`/`obsequious` case here,
  and corrects `golden_schema_test.go:11`.
- **PQ-4 (Important) — one domain vocabulary, not a third.** `Domain` is a closed
  set with a refusing parse; `glosslabel.go` derives its labels from it instead
  of owning them. **A design improvement fell out of this:** `readGloss` already
  yields a NOAD field label, so most words get a domain from the DICTIONARY with
  no model call — the model is now the fallback, not the source.
- **PQ-5 (Important) — the agreement measure has a mechanism.** It moves to its
  own `--agreement[=N]` mode over a sample of K, writing nothing, so the daily
  path keeps both of its stated properties. The floor (`≥ 0.8`, `N=5`) moves to
  the conformance row; the unit test owns the arithmetic, since a fake seeded to
  vary only measures its own seed.
- **PQ-6 (Important) — neutralisation is named.** `sanitiseFacts`/`sanitiseItem`,
  one pass over the struct at the write, per `sanitiseModel`'s stated reason. An
  unparseable facts file reads as absent, not as an error.
- **Minor 1 — the golden lands in `cmd/define/testdata/golden/`**, where
  `AssertGolden`'s callers put theirs.
- **Minor 2 — `--limit` (default 200)** bounds a run's model calls, with
  `poolCap = 40` as the precedent for bounding batch work explicitly.

### 2026-09-04 — M1 boundary review: 4 Important, 6 Minor, all addressed

**Reason.** `sdlc milestone-close --issue 10 --milestone M1` returned
FIX-THEN-SHIP. Sidecar: `workshop/plans/000010-vocab-harvest-m1-review.md`. Every
finding was checked against the tree; all held.

**The one worth remembering: BR-4, a plan step ticked for code that was never
written.** Task 1 Step 4 named `sanitiseFacts`/`sanitiseItem`, the plan had
pre-rejected "the parse covers it" in writing, and neither function existed. The
gate's PQ-6 was recorded as *addressed* on the strength of plan prose alone. **A
finding is disposed by the CODE, not by the paragraph promising it** — and a
ticked checkbox is the weakest possible evidence, because ticking it is the
cheapest thing in the loop.

**Deltas.**

- **BR-1 — `agreement` keyed the RAW band**, so `["C1","c1","C1"]` scored 0.67
  and a perfectly stable model could fail the 0.8 floor whose prescribed remedy
  is the expensive hand-labelled sample. Keyed on the parsed value; two
  regression rows added. The dead tie-break `sort` went with it.
- **BR-2 + BR-4 share one fix, which is the class.** `sanitiseFacts`/
  `sanitiseItem` now live in `store` and are called by BOTH implementations at
  the write, so canonicalisation and neutralisation are the INTERFACE's
  guarantee rather than YAML's. That also closes the divergence BR-2 found —
  `Mem` returned an off-scale band as harvested where `YAML` refused it, the fake
  being the permissive one, which is the direction that hides bugs. Three rows
  moved into `storetest/suite.go`, where the plan said to put them.
- **BR-3 — the band prompt hardcoded English** while `facts/<lang>/` exists
  precisely because Spanish decks are live. The language is threaded and asserted;
  the system prompt names no language.
- **Minors** — the unreachable `-agreement` default is wired, `-agreement` is
  bounded above, `-limit` with `-agreement` and `--harvest` with `--play`/
  `--reflect` now refuse instead of silently dropping a mode, the loop-invariant
  dictionary lookup is hoisted, `contains` is `strings.Contains`, and the
  **computed longest-first ordering is pinned** — the review verified an inverted
  comparator left the whole package green, and it now reddens.
