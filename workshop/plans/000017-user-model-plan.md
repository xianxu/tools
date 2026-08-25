# Learner model from lookups — Implementation Plan (M1)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Issue:** [tools#17](../issues/000017-user-model.md) · **Project:** [define-learn](../projects/define-learn.md)

**Goal:** `define --reflect` folds the deck and the lookup log into a durable
`user-model.md` whose every claim names the deck words it was derived from, and
whose `## Corrections` section survives regeneration byte-for-byte.

**Architecture:** A batch fold, a typed model call, and a pure renderer, in that
order and with nothing else between them. `foldLookups` turns the store's raw
rows into the evidence the prompt carries; `llm.Run[learnerModel]` returns a
typed answer; `checkEvidence` **drops any claim whose evidence is not in the
deck** before anything is written; `renderUserModel` and `spliceCorrections` are
pure text. The whole path is one mode flag beside `--llm-check` — no model call
ever moves onto the lookup or review path (ARCH-PURE).

**Tech Stack:** Go, `internal/llm` (`Run[T]`, the wire-level `llmtest` fake),
`cmd/define/store` (deck + events + `UserModel`/`SetUserModel`, both
implementations under one conformance suite).

---

## Scope check

The issue carries two milestones and this plan is **M1 only**. M2 (the weakness
taxonomy) needs review events that #6 does not yet produce; planning it now would
be planning against a data shape nobody has seen. M1 stands alone: it ships a
file that #16's ask path already reads today.

---

## Decisions

**D1 — The model's evidence is CHECKED, not trusted.** The typed answer carries
the deck words behind each claim, and `checkEvidence` drops any claim citing a
word the deck does not contain. This is the project's existing rule — *distractors
are selected, never invented* (`define-learn`, 2026-08-20) — applied to the
learner model: the model may **read** the deck and **not** add to it. Without the
check, "every claim names its evidence" is a formatting convention that a
plausible hallucination satisfies.

**D2 — A floor, because a model built from four lookups is noise.** Below
`minDeckForReflection` (12 words) `--reflect` writes nothing and says why. The
alternative is a confidently generic file that then steers authoring, which is
worse than no file — and `gatherAskContext` already degrades cleanly on absence.

**D3 — `## Corrections` is spliced, not re-rendered.** Everything above the
marker is replaced; the marker and everything below it is copied byte-for-byte.
The marker is matched **only outside fenced code blocks**, because the file
documents its own format and a fence containing `## Corrections` must not split
it. First run writes an empty `## Corrections` section so the human-owned space
exists before anyone needs it.

**D4 — Idempotency is a property of fixed inputs, and the clock is one of them.**
`updated:` and `window:` come from `d.clock` and the store, both already
injectable; the model's answer comes from a committed capture under the fake. Two
runs against the same three produce byte-identical files, asserted.

**D5 — `--reflect` is a mode, dispatched beside `--llm-check`.** Same shape: it
answers a question about the directory rather than looking a word up, so it is
validated and dispatched before the argument count is judged. It is also the
second consumer of `deps.getenv`/`newLLM`, which #16 introduced.

**D6 — The prompt carries counts and dates, never the raw event log.** A year of
events is thousands of rows; the fold summarises to one line per word plus the
window. That is also what makes `foldLookups` a pure function worth testing.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `deckEvidence` | `cmd/define/reflect.go` | new |
| `foldLookups` | `cmd/define/reflect.go` | new |
| `learnerModel` / `domainClaim` | `cmd/define/reflect.go` | new |
| `checkEvidence` | `cmd/define/reflect.go` | new |
| `renderUserModel` | `cmd/define/usermodel.go` | new |
| `spliceCorrections` | `cmd/define/usermodel.go` | new |
| `minDeckForReflection` | `cmd/define/reflect.go` | new |

- **deckEvidence** — the folded input: one row per word (text, lookups,
  first/last seen), plus the window and the totals. What the prompt carries.
  - **Relationships:** 1:1 with a `--reflect` run; built from N `store.Word` and
    M `store.ReviewEvent`.
  - **DRY rationale:** #10's authoring prompt wants the same summary of "what has
    this learner been looking at". First occurrence of a shape that recurs.
  - **Future extensions:** M2 adds review events to the same struct rather than a
    parallel one; the fold gains a second source, not a second function.

- **foldLookups** — `([]store.Word, []store.ReviewEvent, time.Time) → deckEvidence`.
  Pure: no store, no clock (the instant is a parameter).
  - **DRY rationale:** the one place that decides what "recent" and "the window"
    mean for the model, rather than each prompt deciding again.

- **learnerModel / domainClaim** — the typed answer: a level band with its
  rationale and evidence words, and N domain claims each with a share, evidence
  words, and the authoring directive that follows from it.
  - **Relationships:** 1 model : N domains. `SchemaFor[learnerModel]` reflects the
    JSON schema, so the struct is the single source of the shape.
  - **Future extensions:** M2 adds `Weaknesses []weaknessClaim`; the renderer
    gains a section and the schema follows the struct.

- **checkEvidence** — `(learnerModel, map[string]bool) → (learnerModel, []string)`.
  Drops claims whose evidence words are absent from the deck; returns what it
  dropped so the caller can say so out loud.
  - **DRY rationale:** the veto shape #12 will need for distractors, arriving
    here first. Both are "the model may select from what exists, not invent".

- **renderUserModel** — `(learnerModel, modelMeta) → string`, the markdown above
  `## Corrections`. Pure, golden-tested.

- **spliceCorrections** — `(existing, generated string) → string`. The only place
  that knows the human-owned section's boundary.
  - **DRY rationale:** first occurrence, and deliberately separate from the
    renderer: one function decides what to WRITE, another decides what to KEEP.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `runReflect` | `cmd/define/reflect.go` | new | `llm.Run` + the store |
| `reflectTask` | `cmd/define/reflectprompt.go` | new | the prompt (domain knowledge) |
| `--reflect` dispatch | `cmd/define/main.go` | modified | flag parsing |

- **runReflect** — read store → fold → `llm.Run[learnerModel]` → check → render →
  splice → `SetUserModel`. Thin: every decision it appears to make belongs to one
  of the pure entities above.
  - **Injected into:** nothing; it composes them. Tests drive it against the
    wire-level `llmtest.Fake` and a real `store.YAML` in a temp dir.

- **reflectTask** — the prompt lives with the consumer, never in `internal/llm`
  (`AGENTS.local.md`). Golden-tested through `llmtest.AssertGolden`, which renders
  the `llm.Request` the transport will actually send.

**Test surface.** Every pure entity gets a table test with no fake. `runReflect`
is exercised against `llmtest.Fake` (an httptest server speaking the wire
protocol — never a stubbed `Client`) plus `store.YAML` in a `t.TempDir()`. The
conformance suite gains nothing here: `UserModel`/`SetUserModel` landed in #16
and are already covered against both implementations.

---

## Chunk 1 — the fold and the evidence check

### Task 1: `deckEvidence` and `foldLookups`

**Files:**
- Create: `cmd/define/reflect.go`, `cmd/define/reflect_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestFoldLookups(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	day := func(n int) time.Time { return now.AddDate(0, 0, -n) }
	deck := []store.Word{ // Deck() is newest-first
		{Text: "certiorari", FirstSeen: day(3), LastSeen: day(1), Lookups: 4},
		{Text: "ephemeral", FirstSeen: day(9), LastSeen: day(9), Lookups: 1},
	}
	events := []store.ReviewEvent{
		{Word: "certiorari", Kind: store.EventLookedUp, Found: true, At: day(3)},
		{Word: "zzzz", Kind: store.EventLookedUp, Found: false, At: day(2)},
		{Word: "certiorari", Kind: store.EventAsked, Question: "vs cert?", At: day(1)},
	}

	got := foldLookups(deck, events, now)

	if got.Words[0].Text != "certiorari" || got.Words[0].Lookups != 4 {
		t.Errorf("words = %+v, want the deck rows carried through", got.Words)
	}
	// A MISS is not vocabulary and an ASK is not a lookup: neither may inflate
	// the counts a claim is later checked against.
	if got.Lookups != 1 {
		t.Errorf("lookups = %d, want only the found looked-up events", got.Lookups)
	}
	if got.Questions != 1 {
		t.Errorf("questions = %d, want the asked events counted separately", got.Questions)
	}
	if got.From != day(9) || got.To != day(1) {
		t.Errorf("window = %v..%v, want first-seen..last-seen across the deck", got.From, got.To)
	}
}

func TestFoldLookupsOnAnEmptyDeck(t *testing.T) {
	got := foldLookups(nil, nil, time.Now())
	if len(got.Words) != 0 || !got.From.IsZero() {
		t.Errorf("got %+v, want an empty fold rather than a zero-day window", got)
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define/ -run TestFoldLookups -v`
Expected: FAIL — `undefined: foldLookups`.

- [ ] **Step 3: Implement**

```go
// deckEvidence is what the model is shown: one row per word plus the totals.
//
// NOT the raw event log — a year of it is thousands of rows, and the claims we
// want back are about words, not about individual lookups. Summarising here is
// also what keeps the decision testable without a store (#17 D6).
type deckEvidence struct {
	Words     []wordRow
	Lookups   int       // found lookups only: a miss is not vocabulary
	Questions int       // #16's asked events, counted apart from lookups
	From, To  time.Time // the window the claims may speak about
}

type wordRow struct {
	Text     string
	Lookups  int
	FirstAt  time.Time
	LastAt   time.Time
}

// foldLookups summarises the store for one --reflect run.
//
// Pure — the instant is a parameter, not a clock — so "what does the window mean
// when the deck spans one day" is a table row rather than a timing test.
func foldLookups(deck []store.Word, events []store.ReviewEvent, now time.Time) deckEvidence
```

⚠️ Count `EventLookedUp && Found` for `Lookups` and `EventAsked` for `Questions`.
Counting all events conflates three different facts and inflates every share the
model is asked to reason about.

- [ ] **Step 4: Run the tests** — `go test ./cmd/define/ -run TestFoldLookups -v` → PASS
- [ ] **Step 5: Commit**

```bash
git add cmd/define/reflect.go cmd/define/reflect_test.go
git commit -m "#17 M1: foldLookups — the deck as evidence, not as a log"
```

### Task 2: `learnerModel` and `checkEvidence`

**Files:**
- Modify: `cmd/define/reflect.go`, `cmd/define/reflect_test.go`

- [ ] **Step 1: Write the failing test**

```go
// D1: the model may READ the deck and may not ADD to it. A claim citing a word
// nobody looked up is a hallucination wearing the shape of evidence, and the
// file's whole promise is that a claim can be checked.
func TestCheckEvidenceDropsClaimsTheDeckCannotSupport(t *testing.T) {
	deck := map[string]bool{"certiorari": true, "dicta": true, "ephemeral": true}
	in := learnerModel{
		Level:       levelClaim{Band: "C1", Evidence: []string{"certiorari", "dicta"}},
		Domains: []domainClaim{
			{Name: "law", Share: 0.6, Evidence: []string{"certiorari", "dicta"}},
			{Name: "sailing", Share: 0.2, Evidence: []string{"luffing", "clew"}},
			{Name: "mixed", Share: 0.2, Evidence: []string{"ephemeral", "luffing"}},
		},
	}

	got, dropped := checkEvidence(in, deck)

	if len(got.Domains) != 2 {
		t.Fatalf("domains = %+v, want the unsupported one dropped", got.Domains)
	}
	if got.Domains[0].Name != "law" {
		t.Errorf("kept %q, want law", got.Domains[0].Name)
	}
	// A PARTIALLY supported claim keeps only the evidence that exists, rather
	// than being dropped whole: "mixed" is real, "luffing" is not.
	if len(got.Domains[1].Evidence) != 1 || got.Domains[1].Evidence[0] != "ephemeral" {
		t.Errorf("mixed evidence = %v, want only the deck word", got.Domains[1].Evidence)
	}
	if len(dropped) == 0 {
		t.Error("nothing reported: a dropped claim must be sayable out loud")
	}
}

func TestCheckEvidenceDropsALevelClaimWithNoSupport(t *testing.T) {
	// The level band is a claim like any other. Unsupported, the file must not
	// carry it — an asserted band is exactly what the issue's spec forbids.
}

func TestCheckEvidenceMatchesOnTheDeckKEY(t *testing.T) {
	// "Certiorari" and "certiorari" are one deck entry (store.Key), so evidence
	// must match the same way or every capitalised citation is dropped.
}
```

- [ ] **Step 2: Run and watch it fail.**
- [ ] **Step 3: Implement.**

```go
// learnerModel is the typed answer. SchemaFor reflects the JSON schema from this
// struct, so the shape has one source and a field added without thought shows up
// in the golden's diff.
type learnerModel struct {
	Level   levelClaim    `json:"level"`
	Domains []domainClaim `json:"domains"`
}

type levelClaim struct {
	Band      string   `json:"band"`      // CEFR-ish: B2, C1, C2
	Rationale string   `json:"rationale"`
	Evidence  []string `json:"evidence"`  // deck words, checked against the deck
}

type domainClaim struct {
	Name      string   `json:"name"`
	Share     float64  `json:"share"`
	Evidence  []string `json:"evidence"`
	Directive string   `json:"directive"` // what authoring should DO about it
}

// checkEvidence enforces D1: a claim may cite only words the deck holds.
func checkEvidence(m learnerModel, deck map[string]bool) (learnerModel, []string)
```

⚠️ Match on `store.Key(word)`, not the raw string — the deck's identity is
case- and space-normalised, and matching raw drops every capitalised citation.

- [ ] **Step 4: Run the tests.**
- [ ] **Step 5: Commit.**

```bash
git commit -am "#17 M1: checkEvidence — the model may read the deck, not add to it"
```

## Chunk 2 — the file

### Task 3: `renderUserModel`

**Files:**
- Create: `cmd/define/usermodel.go`, `cmd/define/usermodel_test.go`,
  `cmd/define/testdata/golden/user-model.md`

- [ ] **Step 1: Write the failing test** — a golden over a fully populated model,
      plus rows for the shapes that must not render.

```go
func TestRenderUserModel(t *testing.T) {
	got := renderUserModel(sampleLearnerModel(), modelMeta{
		Updated: time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
		From:    time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		To:      time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
		Lookups: 84, Questions: 6, Model: "claude-opus-5",
	})
	assertGoldenFile(t, "testdata/golden/user-model.md", got)
}

func TestRenderUserModelNamesEvidenceForEveryClaim(t *testing.T) {
	// Each rendered claim line contains at least one deck word. The issue's rule
	// is "a claim that cannot name the events behind it does not go in the file"
	// — asserted over the OUTPUT, because that is where a reader checks it.
}

func TestRenderUserModelEndsWithTheCorrectionsMarker(t *testing.T) {
	// The human-owned section exists from the first run, so nobody has to know
	// the marker's spelling to use it.
}
```

- [ ] **Step 2: Run and watch it fail.**
- [ ] **Step 3: Implement.** Frontmatter exactly as the issue's Spec shows
      (`type`, `learner`, `updated`, `window`, `generated_by`), then `## Level`,
      `## Domains they read in` (a table: domain, share, evidence, directive),
      then the `## Corrections` marker with a one-line invitation under it.
- [ ] **Step 4: Record the golden** — `go test ./cmd/define/ -run TestRenderUserModel -update`
      then READ the file and check it is what a person would want to be handed.
- [ ] **Step 5: Commit.**

```bash
git commit -am "#17 M1: the learner model, rendered"
```

### Task 4: `spliceCorrections`

**Files:**
- Modify: `cmd/define/usermodel.go`, `cmd/define/usermodel_test.go`

- [ ] **Step 1: Write the failing test**

```go
// The human-owned half. A learner who writes "I read these for pleasure, not for
// the bar exam" must find it there tomorrow, byte for byte.
func TestSpliceCorrections(t *testing.T) {
	for _, tc := range []struct{ name, existing, generated, want string }{
		{"no existing file: the generated text stands alone", "", genA, genA},
		{"corrections are preserved verbatim", oldWithCorrections, genA, genA + correctionsBody},
		{"an existing file with NO marker keeps nothing below", oldNoMarker, genA, genA},
		{"trailing whitespace inside corrections survives", oldWithTrailing, genA, genA + trailingBody},
		{"a marker inside a fenced block is not the marker", oldWithFencedMarker, genA, genA + realBody},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := spliceCorrections(tc.existing, tc.generated); got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run and watch it fail.**
- [ ] **Step 3: Implement.** Scan line by line, tracking fence state (```` ``` ````
      toggles it); the first `## Corrections` **outside a fence** is the boundary.
      Everything from that line on is copied verbatim.

⚠️ The fence case is not hypothetical: this file's own format is documented in
the issue using a fenced block that contains `## Corrections`, and a learner may
well paste that block into their corrections while arguing with it.

- [ ] **Step 4: Run the tests.**
- [ ] **Step 5: Commit.**

```bash
git commit -am "#17 M1: corrections are copied, never regenerated"
```

## Chunk 3 — the run

### Task 5: `reflectTask` — the prompt

**Files:**
- Create: `cmd/define/reflectprompt.go`, `cmd/define/reflectprompt_test.go`,
  `cmd/define/testdata/golden/reflect-prompt.txt`

- [ ] **Step 1: Write the failing test** — the golden renders the `llm.Request`
      through `llmtest.AssertGolden`, and a table asserts the evidence reaches it.
- [ ] **Step 2: Run and watch it fail.**
- [ ] **Step 3: Implement.** The system prompt states the two rules the checker
      then enforces: **cite only words from the list**, and **say what authoring
      should do about each domain**. Say them because a model that follows them
      produces fewer dropped claims; enforce them anyway because saying is not
      guaranteeing (D1).
- [ ] **Step 4: Record the golden and read it.**
- [ ] **Step 5: Commit.**

### Task 6: `runReflect` and the `--reflect` flag

**Files:**
- Modify: `cmd/define/reflect.go`, `cmd/define/main.go:221-225` (flags),
  `cmd/define/main.go:300-312` (mode dispatch)
- Test: `cmd/define/reflect_run_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestReflectWritesAModelFromTheDeck(t *testing.T)      // against the wire fake
func TestReflectIsIdempotent(t *testing.T)                 // two runs, byte-identical
func TestReflectPreservesCorrections(t *testing.T)         // end to end, through the store
func TestReflectRefusesATinyDeck(t *testing.T)             // D2's floor, with the count in the message
func TestReflectSaysWhatItDropped(t *testing.T)            // D1's warning reaches stderr
func TestReflectDegradesWithNoModel(t *testing.T)          // no seam: says so, writes nothing, exit 1
func TestReflectIsAModeFromEveryEntryPoint(t *testing.T)   // `define --reflect extra-arg` is a usage error
```

- [ ] **Step 2: Run and watch them fail.**
- [ ] **Step 3: Implement.** `runReflect` composes the pure parts; the flag sits
      beside `--llm-check` and dispatches before the argument count is judged.
- [ ] **Step 4: Run the tests + `go test -race ./cmd/define/`.**
- [ ] **Step 5: Commit.**

### Task 7: the live conformance check

**Files:**
- Create: `cmd/define/reflect_conformance_test.go` (`//go:build conformance`)

- [ ] **Step 1: Write it.** A curated deck with three known domains and a
      **held-out** word per domain. Run against the live model; assert the
      inferred domains cover the known ones and that every cited word is in the
      deck. Shape, not wording — the same bar the `llmtest` suite sets.
- [ ] **Step 2: Run it** — `go test -tags conformance -run Reflect ./cmd/define/`
- [ ] **Step 3: Commit.**

⚠️ This is the Done-when row that says domain inference is *checked, not
asserted*. It is on-demand like every other conformance suite here, and it must
SKIP (not fail) when the seam is unreachable — "not running" is not "wrong".

### Task 8: close M1

- [ ] `go test ./... && go vet ./... && go test -race ./cmd/define/`
- [ ] Hand-run against the live proxy in a real deck directory; read the file it
      writes and record in `## Log` whether it is worth being handed.
- [ ] Confirm #16's ask path picks it up: ask a question and check the recorded
      prompt carries the new file.
- [ ] Update `atlas/define.md` (a `--reflect` section) and the project row.
- [ ] Run the entity enumeration LAST, against the working tree, as a set
      difference — see `000016-console-qa-plan.md`'s final Revisions entry for the
      exact command and why the reconciliation must not be an eye pass.
- [ ] `sdlc milestone-close --issue 17 --milestone M1`

---

## Done-when → task map

| M1 Done-when row | Task |
|---|---|
| `--reflect` writes a model whose claims name their deck words | 3, 6 |
| regeneration idempotent against fake seam, fixed clock, fixed store | 6 |
| a hand-written `## Corrections` survives byte-for-byte | 4, 6 |
| authoring reads the file; absence degrades rather than errors | 8 (verified against #16's ask path, which already reads it) |
| domain inference checked against a held-out sample | 7 |

## Notes for the reviewer

- **The fourth Done-when row names #10, which does not exist.** #16's
  `gatherAskContext` reads `user-model.md` today and degrades on absence, so the
  row's substance is already satisfied by a live consumer; #10 inherits it. Task
  8 verifies it rather than assuming it.
- **M2 is deliberately unplanned.** It needs review events from #6.
