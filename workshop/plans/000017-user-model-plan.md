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
pure text. The whole path is one mode flag, dispatched where the store exists
(beside `--forget`, see D5) — no model call ever moves onto the lookup or review
path (ARCH-PURE).

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

**D5 — `--reflect` is a mode dispatched where the store EXISTS, which is beside
`--forget` and not beside `--llm-check`.** The two modes look alike and are not:
`--llm-check` runs at `main.go:307` with `os.Getenv`/`llm.New` precisely because
it needs no directory, while `d = d.withStore(opt, stderr)` does not run until
`main.go:339`. Dispatching there would hand `--reflect` a nil `d.deck` and a nil
`d.clock` — the two things D4's idempotency rests on (`PQ-2`). So it follows
`--forget`: arity judged in the switch, dispatched after `withStore`.

A nil deck is still reachable after that, under `DEFINE_NO_CAPTURE`. `--reflect`
then refuses through the existing `noDeckMessage(opt.noCapture)`
(`main.go:596`) — the same sentence `--forget` and `/history` use, because a
third phrasing of one fact is how #4's atlas contradictions started.

It is the second consumer of `deps.getenv`/`newLLM`, which #16 introduced.

**D6 — The prompt carries counts and dates, never the raw event log**, and the
fold that produces them **already exists**. `summariseLookups`
(`history_cmd.go:105-144`) folds the log into per-word rows with the same
`EventLookedUp && Found` filter, the same `store.Key` keying and the same
FirstAt/LastAt/Lookups accumulation that `/history` needs; `historyRow` is the
row this wanted. `foldLookups` calls it with a zero `since` (the whole log) and
adds only what is genuinely new: the questions count and the window. Writing a
second fold would have been a second answer to "how many times has this learner
looked this up" (ARCH-DRY, `PQ-1`).

**D7 — Two sources, one rule: the DECK decides which words are evidence, the LOG
decides how many times and when.** `store.Word` carries a `Lookups` count and so
does the log, and a plan that reads both without saying which wins is a plan with
two answers to one question (`PQ-4`). The rule follows from what `--forget`
already promises: it removes a word from the deck and deliberately leaves its
events, because *"the deck is a working set, the log is history"*. So a forgotten
word stops being evidence for a claim while its history survives — which is the
behaviour a learner asking to forget something expects, and it falls out of the
rule rather than needing a case.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `deckEvidence` | `cmd/define/reflect.go` | new |
| `foldLookups` | `cmd/define/reflect.go` | new |
| `summariseLookups` / `historyRow` | `cmd/define/history_cmd.go` | reused, unchanged |
| `learnerModel` / `levelClaim` / `domainClaim` | `cmd/define/reflect.go` | new |
| `checkEvidence` / `citedOrNothing` | `cmd/define/reflect.go` | new |
| `renderUserModel` / `dateOrNone` / `joinWords` | `cmd/define/usermodel.go` | new |
| `spliceCorrections` / `firstMarkerOutsideAFence` / `isCorrectionsMarker` | `cmd/define/usermodel.go` | new |
| `minDeckForReflection` | `cmd/define/reflect.go` | new |
| `assertGoldenFile` | `cmd/define/usermodel_test.go` | new |

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
  - **DRY rationale:** it **calls `summariseLookups`** rather than repeating it.
    What is left over — the questions count, the window, and the deck-membership
    filter D7 defines — is genuinely new, and that is the whole of what this
    function adds.
  - **Note:** `wordRow` from the first draft is **deleted before it exists**;
    `historyRow` is that row and already carries `Word`/`FirstAt`/`LastAt`/`Lookups`.

- **assertGoldenFile** — `(t, path, got string)`, for the markdown golden.
  `llmtest.AssertGolden` takes an `llm.Request` and renders it, so it cannot
  compare a rendered file.
  - **DRY rationale:** it reads `llmtest.Updating()` rather than registering a
    second `-update` flag — two flags of that name in one test binary is a panic
    at init, and one of them refreshing half the artifacts is worse.

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
| `runReflect` / `unavailableToReflect` | `cmd/define/reflect.go` | new | `llm.Run` + the store |
| `renderReflectPrompt` | `cmd/define/reflectprompt.go` | new | the prompt (domain knowledge) |
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

- [x] **Step 1: Write the failing test**

```go
// The sketches below assert the PROPERTIES D6 and D7 state, not a field list.
// The first draft asserted Words[0].Lookups == 4 from store.Word while the log
// held one found lookup — a test that could only pass with the second fold PQ-1
// removed. D7 is the single statement of where each field comes from; a test
// that restates it can disagree with it.

// D7: the DECK decides which words are evidence, the LOG decides how many times
// and when. --forget removes a word from the deck and deliberately keeps its
// events, so a forgotten word must stop being evidence while its history still
// counts toward the totals.
func TestFoldLookupsTakesMembershipFromTheDeckAndCountsFromTheLog(t *testing.T)

// D6: EventAsked is counted apart from lookups, and a miss is not vocabulary.
// Conflating them inflates every share the model is asked to reason about.
func TestFoldLookupsCountsAsksAndMissesApartFromLookups(t *testing.T)

// The window is the span the claims may speak about; an empty deck yields an
// empty fold rather than a zero-day window the prompt would then assert over.
func TestFoldLookupsWindowSpansTheEvidence(t *testing.T)
func TestFoldLookupsOnAnEmptyDeck(t *testing.T)
```

Each is a table over `([]store.Word, []store.ReviewEvent, now)`, asserting the
stated property rather than a field-by-field snapshot — the same shape Task 4's
fuzz property takes, and for the same reason: a snapshot of a struct I am about
to change is a test of my memory.

- [x] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define/ -run TestFoldLookups -v`
Expected: FAIL — `undefined: foldLookups`.

- [x] **Step 3: Implement**

```go
// deckEvidence is what the model is shown: one row per word plus the totals.
//
// NOT the raw event log — a year of it is thousands of rows, and the claims we
// want back are about words, not about individual lookups (#17 D6).
type deckEvidence struct {
	Words     []historyRow // reused: /history's row already IS this row
	Lookups   int          // found lookups only: a miss is not vocabulary
	Questions int          // #16's asked events, counted apart from lookups
	From, To  time.Time    // the window the claims may speak about
}

// foldLookups summarises the store for one --reflect run.
//
// The per-word fold is summariseLookups', called with a zero `since` so it
// covers the whole log: same filter, same keying, same accumulation that
// /history needs, and writing a second one would be a second answer to "how many
// times has this learner looked this up" (ARCH-DRY, PQ-1).
//
// What this adds is the part that is genuinely new: the questions count, the
// window, and D7's rule — the DECK decides which words are evidence, the LOG
// decides how many times and when. A word removed by --forget therefore stops
// being evidence while its history survives, which is what --forget promises.
//
// Pure — the instant is a parameter, not a clock — so "what does the window mean
// when the deck spans one day" is a table row rather than a timing test.
func foldLookups(deck []store.Word, events []store.ReviewEvent, now time.Time) deckEvidence
```

⚠️ `Questions` counts `EventAsked` and must NOT reach `summariseLookups`, which
filters to found lookups by design. Conflating the two inflates every share the
model is asked to reason about, and #16's asked events made that reachable.

⚠️ Filter the folded rows by deck membership (D7) — `summariseLookups` reads the
whole log, which still holds words `--forget` removed.

- [x] **Step 4: Run the tests** — `go test ./cmd/define/ -run TestFoldLookups -v` → PASS
- [x] **Step 5: Commit**

```bash
git add cmd/define/reflect.go cmd/define/reflect_test.go
git commit -m "#17 M1: foldLookups — the deck as evidence, not as a log"
```

### Task 2: `learnerModel` and `checkEvidence`

**Files:**
- Modify: `cmd/define/reflect.go`, `cmd/define/reflect_test.go`

- [x] **Step 1: Write the failing test**

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

- [x] **Step 2: Run and watch it fail.**
- [x] **Step 3: Implement.**

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

- [x] **Step 4: Run the tests.**
- [x] **Step 5: Commit.**

```bash
git commit -am "#17 M1: checkEvidence — the model may read the deck, not add to it"
```

## Chunk 2 — the file

### Task 3: `renderUserModel`

**Files:**
- Create: `cmd/define/usermodel.go`, `cmd/define/usermodel_test.go`,
  `cmd/define/testdata/golden/user-model.md`

- [x] **Step 1: Write the failing test** — a golden over a fully populated model,
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

- [x] **Step 2: Run and watch it fail.**
- [x] **Step 3: Implement.** Frontmatter exactly as the issue's Spec shows
      (`type`, `learner`, `updated`, `window`, `generated_by`), then `## Level`,
      `## Domains they read in` (a table: domain, share, evidence, directive),
      then the `## Corrections` marker with a one-line invitation under it.
- [x] **Step 4: Record the golden** — `go test ./cmd/define/ -run TestRenderUserModel -update`
      then READ the file and check it is what a person would want to be handed.
- [x] **Step 5: Commit.**

```bash
git commit -am "#17 M1: the learner model, rendered"
```

### Task 4: `spliceCorrections`

**Files:**
- Modify: `cmd/define/usermodel.go`, `cmd/define/usermodel_test.go`

- [x] **Step 1: Write the failing test**

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

- [x] **Step 1b: Write the PROPERTY, because the examples miss a class**

Five hand-picked cases cannot cover malformed human-edited text, and the failure
mode here is silently discarding the learner's own writing — with byte-for-byte
survival as a Done-when row (`PQ-3`). The property is one line and it quantifies
over inputs nobody thought to type:

```go
// FuzzSpliceCorrectionsPreservesEverythingBelowTheMarker asserts the ONE thing
// that must hold for any existing file: whatever follows the first out-of-fence
// marker comes out byte-identical.
//
// Seeded with the shapes a table would not have reached — tilde fences, an
// UNTERMINATED fence, an indented fence, a marker with trailing whitespace, and
// CRLF line endings, which this repo already handles elsewhere (crlf.go) and so
// will certainly meet here.
func FuzzSpliceCorrectionsPreservesEverythingBelowTheMarker(f *testing.F) {
	for _, seed := range []string{
		"## Corrections\nkeep me\n",
		"~~~\n## Corrections\n~~~\n## Corrections\nreal\n",
		"```\nunterminated fence\n## Corrections\nstill inside\n",
		"    ```\n    indented\n    ```\n## Corrections\nreal\n",
		"## Corrections   \ntrailing space on the marker\n",
		"## Corrections\r\nCRLF body\r\n",
		"no marker at all\n",
		"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, existing string) {
		got := spliceCorrections(existing, "GENERATED\n")
		i := firstMarkerOutsideAFence(existing)
		if i < 0 {
			if got != "GENERATED\n" {
				t.Fatalf("no marker, yet something was carried over: %q", got)
			}
			return
		}
		if tail := existing[i:]; !strings.HasSuffix(got, tail) {
			t.Fatalf("the learner's own text was altered\nwant suffix: %q\ngot: %q", tail, got)
		}
	})
}
```

⚠️ `firstMarkerOutsideAFence` is the same function `spliceCorrections` uses — the
property asserts the SPLICE against the SCANNER, not the scanner against itself.
That is deliberate and it is the honest scope: the scanner's own rules are the
table's job; the property's job is that nothing below the boundary is ever
rewritten, whatever the boundary turns out to be.

- [x] **Step 2: Run and watch them fail.**
- [x] **Step 3: Implement.** Scan line by line tracking fence state — ```` ``` ````
      **and** `~~~`, since both are markdown fences, and an indented fence still
      opens one. The first `## Corrections` **outside a fence** is the boundary;
      everything from that line on is copied verbatim. Trailing whitespace on the
      marker line still makes it a marker; a marker inside an unterminated fence
      is not one.

⚠️ The fence case is not hypothetical: this file's own format is documented in
the issue using a fenced block that contains `## Corrections`, and a learner may
well paste that block into their corrections while arguing with it.

- [x] **Step 4: Run the tests.**
- [x] **Step 5: Commit.**

```bash
git commit -am "#17 M1: corrections are copied, never regenerated"
```

## Chunk 3 — the run

### Task 5: `reflectTask` — the prompt

**Files:**
- Create: `cmd/define/reflectprompt.go`, `cmd/define/reflectprompt_test.go`,
  `cmd/define/testdata/golden/reflect-prompt.txt`

- [x] **Step 1: Write the failing test** — the golden renders the `llm.Request`
      through `llmtest.AssertGolden`, and a table asserts the evidence reaches it.
- [x] **Step 2: Run and watch it fail.**
- [x] **Step 3: Implement.** The system prompt states the two rules the checker
      then enforces: **cite only words from the list**, and **say what authoring
      should do about each domain**. Say them because a model that follows them
      produces fewer dropped claims; enforce them anyway because saying is not
      guaranteeing (D1).
- [x] **Step 4: Record the golden and read it.**
- [x] **Step 5: Commit.**

### Task 6: `runReflect` and the `--reflect` flag

**Files:**
- Modify: `cmd/define/reflect.go`, `cmd/define/main.go:221-225` (flags),
  `cmd/define/main.go:300-312` (mode dispatch)
- Test: `cmd/define/reflect_run_test.go`

- [x] **Step 1: Write the failing tests**

```go
func TestReflectWritesAModelFromTheDeck(t *testing.T)      // against the wire fake
func TestReflectIsIdempotent(t *testing.T)                 // two runs, byte-identical
func TestReflectPreservesCorrections(t *testing.T)         // end to end, through the store
func TestReflectRefusesATinyDeck(t *testing.T)             // D2's floor, with the count in the message
func TestReflectSaysWhatItDropped(t *testing.T)            // D1's warning reaches stderr
func TestReflectDegradesWithNoModel(t *testing.T)          // no seam: says so, writes nothing, exit 1
func TestReflectIsAModeFromEveryEntryPoint(t *testing.T)   // `define --reflect extra-arg` is a usage error
```

- [x] **Step 2: Run and watch them fail.**
- [x] **Step 3: Implement.** `runReflect` composes the pure parts. The flag is
      declared beside `--llm-check` but DISPATCHED beside `--forget`, after
      `d = d.withStore(opt, stderr)` — see D5 for why the two modes look alike
      and are not. A nil deck refuses through `noDeckMessage(opt.noCapture)`.
- [x] **Step 4: Run the tests + `go test -race ./cmd/define/`.**
- [x] **Step 5: Commit.**

### Task 7: the live conformance check

**Files:**
- Create: `cmd/define/reflect_conformance_test.go` (`//go:build conformance`)

- [x] **Step 1: Write it.** A curated deck with three known domains and a
      **held-out** word per domain. Run against the live model; assert the
      inferred domains cover the known ones and that every cited word is in the
      deck. Shape, not wording — the same bar the `llmtest` suite sets.
- [x] **Step 2: Run it** — `go test -tags conformance -run Reflect ./cmd/define/`
- [x] **Step 3: Commit.**

⚠️ This is the Done-when row that says domain inference is *checked, not
asserted*. It is on-demand like every other conformance suite here, and it must
SKIP (not fail) when the seam is unreachable — "not running" is not "wrong".

### Task 8: close M1

- [x] `go test ./... && go vet ./... && go test -race ./cmd/define/`
- [x] Hand-run against the live proxy in a real deck directory; read the file it
      writes and record in `## Log` whether it is worth being handed.
- [x] Confirm #16's ask path picks it up: ask a question and check the recorded
      prompt carries the new file.
- [x] Update `atlas/define.md` (a `--reflect` section) and the project row.
- [x] Run the entity enumeration LAST, against the working tree, as a set
      difference — see `000016-console-qa-plan.md`'s final Revisions entry for the
      exact command and why the reconciliation must not be an eye pass.
- [x] `sdlc milestone-close --issue 17 --milestone M1`

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

## Revisions

### 2026-08-25 — plan-quality round 1 (PQ-1 … PQ-5)

**Reason:** `sdlc change-code` plan gate, three blocking findings and two Minor.
Ledger: `workshop/plans/000017-user-model-plan-gate.md`.

- **PQ-1 (Important) — addressed.** `foldLookups` was re-implementing
  `summariseLookups`: same `EventLookedUp && Found` filter, same `store.Key`
  keying, same accumulation, and `historyRow` was the `wordRow` I had declared.
  It now calls it, and `wordRow` is deleted before it existed. The DRY rationale
  claiming to create "the one place" was the tell — that place was already there,
  in the file `/history` reads.
- **PQ-2 (Important) — addressed.** D5 had `--reflect` dispatching beside
  `--llm-check`, which runs BEFORE `withStore` precisely because it needs no
  directory — so the mode whose idempotency rests on `d.clock` and the store
  would have got nil for both. It follows `--forget` now, and a nil deck under
  `DEFINE_NO_CAPTURE` refuses through the existing `noDeckMessage` rather than a
  third phrasing of one fact. The header and Task 6 were swept for the same
  claim, not just D5 — that restatement drift is the family #16 hit nine times.
- **PQ-3 (Important) — addressed.** `spliceCorrections` is a line scanner over
  human-edited text whose failure mode is silently discarding the learner's own
  writing. Five examples cannot cover that class, so there is now a fuzz property
  — *everything below the first out-of-fence marker survives byte-identical* —
  seeded with tilde fences, an unterminated fence, an indented fence, a marker
  with trailing whitespace and CRLF. The implementation step gained the rules
  those seeds imply.
- **PQ-4 (Minor) — addressed as D7.** The draft read per-word counts from
  `store.Word` and the total from the log, which is two answers to one question.
  The rule now follows from what `--forget` already promises: the DECK decides
  which words are evidence, the LOG decides how many times and when.
- **PQ-5 (Minor) — addressed.** `assertGoldenFile` is genuinely new surface
  (`llmtest.AssertGolden` takes an `llm.Request`, not a string) and has a table
  row. It reads `llmtest.Updating()` rather than registering a second `-update`,
  which would be a flag redefinition panic in a binary that links both.

### 2026-08-25 — M1 shipped; the plan's own tasks, revised by what running it found

Tasks 1–8 done. Two departures from the plan as written, both forced by
measurement rather than by taste:

1. **`evidence` became `evidence_words`, and `checkEvidence` grew a second arm.**
   The plan had the check enforcing one rule (cite only deck words). Live, the
   model put whole sentences in the array — so every level claim was correctly
   dropped and `## Level` went missing from every run — and under a schema
   requiring every field it stubbed claims it did not believe in. The field name
   is what steers; the check now also drops claims authoring cannot act on.
2. **A second floor.** D2 covered a deck too small to reflect on. Nothing covered
   an ANSWER with nothing left in it, and `--reflect` wrote frontmatter plus an
   empty promise — a file that reads as an answer. It writes nothing now.

Also: `llm.Task.MaxTokens` is set explicitly (16384). At the 8192 default,
answers were intermittently degenerate because high-effort thinking shares that
budget with a genuinely long answer — not a truncation, which `Run` catches
first, but the same squeeze that produced #11's preserved specimen.

**The enumeration, run LAST against the working tree, found eight symbols with no
row** — `citedOrNothing`, `dateOrNone`, `firstMarkerOutsideAFence`,
`isCorrectionsMarker`, `joinWords`, `levelClaim`, `renderReflectPrompt`,
`unavailableToReflect` — every one of them created after the tables were written.
That is the check working as #16 finally learned to run it: not the command, but
the MOMENT. Reconciled to empty before this commit.

### 2026-08-27 — close round 8 (BR-23)

All 40 step checkboxes ticked to match delivery. The plan's own Revisions entry
had said "Tasks 1–8 done" since M1 shipped, while every box below it stayed
unticked — the plan read as unstarted next to a paragraph saying it was finished
(2nd in the `plan-artifact-not-ticked` family across this repo). Every box in
this document belongs to Tasks 1–8; M2 was deliberately unplanned here and is now
descoped into `tools#7`, so nothing in this plan is outstanding.

