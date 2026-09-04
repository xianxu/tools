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
| `Band` | `cmd/define/store/item.go` | new |
| `WordFacts` | `cmd/define/store/item.go` | new |
| `Item` | `cmd/define/store/item.go` | new |
| `agreement` | `cmd/define/harvest_band.go` | new |
| `pickDistractors` | `cmd/define/harvest_item.go` | new |
| `topicSpread` | `cmd/define/harvest_judge.go` | new |
| `prune` | `cmd/define/store/item.go` | new |

- **`Band`** — a CEFR level, `A1`…`C2`, as a string type with a parse that REFUSES anything else.
  - **DRY rationale:** the scale is the one thing `#17`'s learner band and a word's band must share; two spellings of it is how "the learner's band, or one below" comes to compare incomparable things. `#17` already assigns the learner one — this reuses that vocabulary rather than defining a parallel enum.
  - **Future extensions:** if a downloadable CEFR asset ever arrives (the issue DEFERS rather than rejects one), it produces `Band` values and nothing else changes.

- **`WordFacts`** — what is cached per word forever: `Band`, `Domain`, and `At`.
  - **Relationships:** 1:1 with a deck word, keyed by `store.Key` like `words/`.
  - **DRY rationale:** one artifact rather than two caches, because they are assigned in one call and expire together (never).

- **`Item`** — a finished practice item: `Word`, `Stem`, `Answer`, `Distractors`, `At`, and the `Form` it was authored for.
  - **Relationships:** 1:N with a word — a word may hold several items, and `#12` picks among them.
  - **Future extensions:** `#13`'s free-sentence form wants a different shape; `Form` is the field that lets one store hold both rather than a second store appearing beside this one.

- **`agreement`** — over N repeated band assignments for one word, the fraction that agree with the modal answer. **This is the measured claim M1 makes**, and it is deliberately narrow — see "What the measure does not say" below.

- **`pickDistractors`** — same domain (or general vocabulary), at the learner's band or one below, never the answer. Pure over an already-banded candidate list.
  - **DRY rationale:** `play.PickOptions` already selects options at review time from a live pool; this is its authoring-time counterpart and must NOT become a second copy of it. The plan's Task 6 checks whether the two can share.

- **`topicSpread`** — distinct domains over a batch of items, as a fraction. **Measured with no model at all**, which is what Done-when 5 asks for: a judge scoring its own batch's variety is the oracle problem the atlas already records.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `store.WordFacts`/`Items` accessors | `cmd/define/store/store.go` | modified | the working directory |
| `bandTask` / `authorTask` / `vetoTask` | `cmd/define/harvest_*.go` | new | the model, via `llm.Task[T]` |
| `runHarvest` | `cmd/define/harvest.go` | new | the batch mode |

- **`store.WordFacts`/`Items` accessors** — four methods on the existing `Store` interface, following `NewsItems`/`SetNewsItems` exactly.
  - **Injected into:** every consumer already takes `store.Store`; nothing new is threaded.
  - **ARCH-MOCK:** `store.Mem` implements them and `storetest/suite.go` gains rows, so the fake and the YAML store are held to one contract — the suite exists for precisely this and a new surface that skipped it would be the gap `#16 M2` BR-45 closed.
  - **The new directory joins `RuntimeDirs`**, which every guard, migration and test derives from. A hand-typed name would be the second source that comment warns about.

- **`bandTask` / `authorTask` / `vetoTask`** — three typed `llm.Task[T]`s.
  - **Injected into:** `llm.Run`, through `deps.newLLM` — the seam `#11` built and `#16` already uses.
  - **ARCH-MOCK:** each gets a `llmtest` golden (the prompt is pinned) and runs against `llmtest.Fake` in unit tests; the live conformance check follows `#11`'s existing `-tags conformance` pattern, so drift between the fake and the real service is detected rather than assumed.

- **`runHarvest`** — the batch mode, dispatched like `--forget` and `--llm-check`: a MODE, validated apart from the argument count.
  - **ARCH-CONSTRAINTS:** see the envelope below.

**Operating envelope (ARCH-CONSTRAINTS).** `--harvest` is a BATCH path and the only one in this program that may block: a sitting must never wait on it, which Done-when 1 states and a test enforces by driving a sitting with the model seam made to panic.
- *Latency:* unbounded by design, bounded in practice by the deck. One band call per unbanded word, one author call per item.
- *Scale:* a deck of a few thousand. Work is per NEW word, so a second run over an unchanged deck makes ZERO model calls — that is Done-when 2 and it is what keeps the cost from growing with time.
- *Overload:* a model outage leaves the store untouched and harvesting stops (Done-when 6). `#19` is open on `ErrRequest` mis-classification and is a real risk here, since this is the first path that makes many calls in a row.
- *Disk:* bounded by `prune`, deterministic, tested (Done-when 7).

**Test surface.** `Band`, `agreement`, `pickDistractors`, `topicSpread` and `prune` are pure and unit-tested with no model and no disk. The store surface is covered by `storetest`, so `Mem` and `YAML` cannot diverge. The three tasks have goldens plus fake-driven tests, and a live conformance row each.

---

## Chunk 1: M1 — a band and a domain per word

### Task 1: the artifact and its store surface

**Files:**
- Create: `cmd/define/store/item.go`, `cmd/define/store/item_test.go`
- Modify: `cmd/define/store/store.go` (interface), `mem.go`, `yaml.go` (incl. `RuntimeDirs`), `storetest/suite.go`

- [ ] **Step 1: Write the failing conformance rows**

In `storetest/suite.go`, so BOTH implementations are held to them at once:

```go
// Absent facts are not an error — the normal state of every word before the
// first harvest, exactly as an absent user model is (#16 M2).
// Written facts read back identically.
// A second write REPLACES rather than merging: a band is assigned once, and a
// re-harvest that changed one would break the cache's whole premise.
```

- [ ] **Step 2: Run against `Mem` and watch it fail to compile**

Run: `go test ./cmd/define/store/...`
Expected: FAIL — `WordFacts` undefined.

- [ ] **Step 3: Define the types, parse-refusing on `Band`**

`Band` is a string type whose parse refuses anything outside `A1`…`C2`. **A model
returning `B2+` or `intermediate` is a real answer to a badly-posed question**,
and accepting it would put an unorderable value into arithmetic that compares
bands. Refuse at the boundary, count the refusal, leave the word unbanded.

- [ ] **Step 4: Implement on `Mem`, then `YAML`**

`YAML` writes `facts/<key>.yaml` through the SAME atomic-write helper `words/`
uses — the shadow-file prefix is already single-sourced, and this must not become
a second writer. Add `"facts"` to `RuntimeDirs`.

- [ ] **Step 5: Run, then commit**

```bash
go test ./cmd/define/store/... && go test ./cmd/define/...
git commit -m "#10 M1: a band and a domain per word, cached like the deck"
```

---

### Task 2: the band task, and the stability measure

**Files:**
- Create: `cmd/define/harvest_band.go`, `cmd/define/harvest_band_test.go`
- Create: golden under `internal/llm/llmtest/testdata/` per that package's convention

- [ ] **Step 1: Write `agreement` and its table test**

Pure, and worth its own test before any model exists:

```go
// agreement is the fraction of N assignments that match the modal band.
// 1.0 is perfect stability; 1/N is noise.
//
// THE MODE, not the first answer: "what does this model usually say" is the
// question the cache's premise rests on, and anchoring on a single run would
// measure that run's luck.
```

- [ ] **Step 2: Write the task, with the golden**

```go
// bandTask asks for a CEFR band and a domain in ONE call, because they are one
// judgement about the word and two calls would double the cost of the only
// per-word work this issue does.
```

- [ ] **Step 3: The measure, and what it does not say**

```go
// TestBandAssignmentIsStable drives the same word N times through a fake seeded
// to vary, and asserts agreement above a floor.
//
// STABILITY, NOT CORRECTNESS, and the distinction is the operator's decision of
// 2026-09-04 recorded on the issue. This measures the property the cache
// depends on — a band assigned once and reused forever must be the band this
// model usually gives — and it CANNOT detect a model that is confidently and
// consistently wrong. Every downstream use of a band rests on the scale being
// right, so the number this reports is agreement, and it is reported as that
// with no claim about correctness.
```

**Report the number, do not bury it.** `--harvest` prints the agreement it
measured, so a scale that starts drifting is visible without anyone re-reading
this comment.

- [ ] **Step 4: A live conformance row**

Follow `#11`'s existing `-tags conformance` pattern: the same word, against the
real service, must land in the same band across runs. That is the only check that
can see the fake and the service disagreeing about the shape of the answer.

- [ ] **Step 5: Run, then commit**

---

### Task 3: `--harvest`, as a mode

**Files:**
- Create: `cmd/define/harvest.go`, `cmd/define/harvest_test.go`
- Modify: `cmd/define/main.go` (flag + dispatch), `cmd/define/README.md`

- [ ] **Step 1: Write the failing test for Done-when 1 and 2**

```go
// A SITTING NEVER WAITS ON HARVESTING. Driven with the model seam made to PANIC,
// not nil: nil passes on a loop that reaches for the model behind a `!= nil`
// guard, which is how a network dependency creeps into an offline path.
//
// AND A SECOND RUN MAKES NO CALLS. The counting fake asserts zero — "cached
// forever" is a claim about calls, not about the file existing.
```

- [ ] **Step 2: Dispatch it as a mode**

Beside `--forget` and `--llm-check`, which are validated apart from the argument
count. Reuse that path rather than adding a fourth shape.

- [ ] **Step 3: Outage leaves the store untouched (Done-when 6)**

Fail the fake mid-batch; assert the words banded before the failure are intact and
nothing partial was written. The atomic write already gives this — the test is
what makes it a property rather than an accident.

- [ ] **Step 4: README + atlas, then close M1**

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
generic authoring, not an error**, which is the normal first-run state.

Two constraints go in the prompt as REQUIREMENTS, because a model asked for a
natural sentence drifts to the neutral and unnamed:
- the stem must ENTAIL its answer;
- it must name real people, places or institutions.

- [ ] **Step 2: `topicSpread`, measured with no model**

Done-when 5's second half. A judge scoring its own batch's variety is the
self-oracle problem `atlas/define.md` already records under "Its own oracle" —
distinct domains over a batch is arithmetic and cannot flatter itself.

- [ ] **Step 3: The entailment judge, on a batch**

An `llm.Task` scoring a batch, with a COMMITTED known-bad stem that must be
rejected — the same shape Done-when 4 asks of the veto, and the only thing that
makes a judge's pass meaningful.

---

### Task 5: distractors SELECTED, and vetoed

**Files:**
- Create: `cmd/define/harvest_judge.go`, `cmd/define/harvest_judge_test.go`

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
