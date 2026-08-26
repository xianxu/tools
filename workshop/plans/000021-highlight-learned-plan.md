# Highlight the Words You Are Learning — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render every word the learner has looked up in bold green, wherever it appears — the line they are typing, definition bodies, and streamed LLM answers.

**Architecture:** One predicate (`Vocabulary.Has`) behind which #22 can later swap an active-learning subset; one pure matcher (`highlightSpans`) that answers "which runs of this text are known words"; and two renderers over it — `RenderLine` for the single-frame prompt, and `highlightWriter` for text that carries ANSI codes and arrives in chunks.

**Tech Stack:** Go 1.26, `cmd/define` (package main), existing `store.Store` / `History` seam conventions, `go test` + native fuzzing.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `span` | `cmd/define/highlight.go` | new |
| `highlightSpans` | `cmd/define/highlight.go` | new |
| `wordRuns` / `wordRun` | `cmd/define/highlight.go` | new |
| `sgrState` | `cmd/define/highlight.go` | new |
| `RenderLine` | `cmd/define/editor.go` | modified |

- **`span`** — one run of text plus whether it is a word the learner knows.
  - **Relationships:** N spans per rendered line; concatenating their `text` reproduces the input exactly. That invariant is the whole safety story, exactly as `head+text == line` was in #20 — a renderer joins spans back into what the user sees, so a lossy split silently corrupts a definition. Fuzz target, not a table row.
  - **DRY rationale:** The single answer to "which parts of this text are known", consumed by both renderers. Without it, the prompt line and the definition path would each grow their own matcher and drift.
  - **Future extensions:** A `reason` field if #22 wants to distinguish "learning" from "mastered" with two colours.

- **`highlightSpans(text string, v Vocabulary) []span`** — splits text into alternating known/unknown runs.
  - **Relationships:** Depends on `Vocabulary` (an interface), so it is pure over injected data and unit-testable with `memVocabulary` and no IO.
  - **DRY rationale:** Longest-match phrase logic lives here once. `hot dog` must beat `hot`, and both renderers need that rule.

- **`wordRuns(text string) []wordRun`** — the tokenizer: runs of word characters (Unicode letter, digit, apostrophe, hyphen) with their byte offsets, **with leading and trailing joiners trimmed**. Apostrophes and hyphens are word characters only INSIDE a word, so `don't` is one token but `'obsequious'` yields the word without its quotes. Runs are therefore maximal only modulo that trim.
  - **DRY rationale:** `highlightSpans` and `highlightWriter` must agree byte-for-byte on where a word begins and ends, or a word highlighted in a definition would not highlight at the prompt. One tokenizer, two callers.
  - **Future extensions:** The word-character set is one predicate; a locale that needs different rules changes it in one place.

- **`sgrState`** — the active SGR style, updated by feeding it escape sequences.
  - **DRY rationale:** ANSI has no nesting. Injecting green inside `\x1b[3;32m` (an example) must emit `green + word + reset + \x1b[3;32m` to resume. Something has to remember the enclosing style; this is that something, and it is pure.

**Test surface implied by the table.** `highlight_test.go` colocated; every test runs with `memVocabulary` and no IO. `FuzzHighlightSpans` asserts span concatenation reproduces the input and that no known span is empty. `RenderLine`'s existing tests in `editor_test.go` extend.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Vocabulary` | `cmd/define/vocab.go` | new | the deck |
| `storeVocabulary` | `cmd/define/vocab.go` | new | `store.Store` |
| `memVocabulary` | `cmd/define/vocab.go` | new | nothing (the double) |
| `highlightWriter` | `cmd/define/highlight.go` | new | `io.Writer` |
| `storeCapturer.Capture` | `cmd/define/capture.go` | modified | store writes |
| `deps.vocab` | `cmd/define/main.go` | modified | dependency wiring |

- **`Vocabulary`** — `Load()`, `Add(word string)`, `Has(key string) bool`, `MaxPhraseWords() int`.
  - **Injected into:** `highlightSpans`, `highlightWriter`, `RenderLine`. This is the seam #22 replaces: today `storeVocabulary` fills the set from `Deck()`; later it fills from the active-learning subset, and nothing above changes.
  - **Why not reuse `History`:** different source (`words/` vs the event log), different query (set membership vs ordered prefix search), and — decisive — different *contents*. History carries typos deliberately so they stay recallable (#20); highlighting a typo as a known word is the opposite of reinforcement. The operator named this distinction: *"not history, but correct form of the words."*
  - **`MaxPhraseWords` is not a leak.** The streaming writer must know how many tokens to hold back before it can rule out a phrase match; for a deck of single words it is 1 and the writer holds back almost nothing.
  - **Future extensions:** `Has` widening to return a category for #22's two-tier colouring.

- **`storeVocabulary`** — reads `Deck()` once into a set. `Load()` is separate from construction for the same reason `storeHistory.Load` is: a one-shot `define /help` must not pay to read the deck.
  - **Injected into:** `deps.vocab`, filled at the boundary in `main.go` beside `deck`.

- **`memVocabulary`** — the in-memory double (ARCH-MOCK). Production and tests share the `Vocabulary` boundary; no test reaches around it.

- **`highlightWriter`** — stateful `io.Writer` that rewrites known words in a byte stream that already carries ANSI codes.
  - **Injected into:** the definition print site (`main.go:488`) and the answer stream sink (`ask.go:160`).
  - **State model:** active SGR (`sgrState`); a held-back tail beginning at the first token that could still extend into a phrase; a partial escape sequence. `Write` always reports `len(p)` consumed even while holding bytes back — the `io.Writer` contract, and the same problem `crlfWriter.consumed` already solves in this tree.
  - **Future extensions:** any other styled stream (a `--stats` table) wraps the same writer.

### `highlightWriter`'s contract (normative — tasks refer here, not restate)

**1. Nothing overtakes held text.** The writer holds a tail back while a phrase
is still possible. If an escape sequence arrives during a hold, it must NOT be
passed straight through — that reorders the output. Concretely, deck `hot dog`
against `Render`'s headword bytes `\x1b[1;36mhot\x1b[0m dog`: holding `hot` and
emitting `\x1b[0m` immediately puts the reset before the word it was closing.
The rule: an escape encountered during a hold **resolves the hold first**, then
passes through.

**2. The phrase window is broken by anything but spaces and tabs.** A multi-word
key may span only ` ` and `\t` between its tokens. An escape, a newline, or
punctuation resolves the window immediately. Two consequences worth stating
because they are behaviour, not accidents:
- `hot\x1b[0m dog` does not match `hot dog` — a phrase whose halves are styled
  differently is not a phrase. This is what makes rule 1 implementable.
- A phrase does not match across a line break. `store.Key` collapses all
  whitespace (`store/word.go:29-31`), so a `Render`-wrapped `hot\n  dog` would
  otherwise form the candidate `hot dog` and paint a green run straight through
  the wrap indent.

**3. Escape-stripped equality is not enough.** Both properties the first draft
proposed — visible-text equality and one-call equivalence — are blind to a
reorder of escapes against text. Byte-for-byte assertions on full output, for at
least the styled-headword case above.

**4. Downstream errors propagate; counts are in the caller's units.** The writer
emits more bytes than it receives (added escapes) and sometimes fewer (held
back), so the count returned to the caller can never be the count the downstream
writer returned. `Write(p)` returns `(len(p), nil)` when every byte of `p` has
been accepted or held. On a downstream error it returns `(n, err)` where `n` is
the number of `p`'s bytes whose output was fully written, and it must not
double-emit those bytes on a subsequent call. This is not hypothetical:
`highlightWriter` will wrap `crlfWriter` (`replraw.go:166`), which returns short
counts on partial writes and is already tested for it (`crlf_test.go:50, :78`).
Byte loss is this feature's worst failure mode.

**5. `Flush` is part of the contract, not a convenience.** Held text is invisible
until flushed. Every exit path flushes — see Task 8 Step 4.

---

## Chunk 1: M1 — the seam, the matcher, the typed line

### Task 1: The `Vocabulary` seam

**Files:**
- Create: `cmd/define/vocab.go`
- Test: `cmd/define/vocab_test.go`

- [x] **Step 1: Write the failing tests**

```go
func TestMemVocabularyNormalisesOnTheWayIn(t *testing.T) {
	v := &memVocabulary{}
	v.Add("Hot  Dog")
	if !v.Has(store.Key("hot dog")) {
		t.Error("Add must normalise through store.Key, as the deck does")
	}
}

func TestMaxPhraseWordsTracksTheLongestEntry(t *testing.T) {
	v := &memVocabulary{}
	v.Add("obsequious")
	if got := v.MaxPhraseWords(); got != 1 {
		t.Errorf("MaxPhraseWords = %d, want 1", got)
	}
	v.Add("a priori")
	if got := v.MaxPhraseWords(); got != 2 {
		t.Errorf("MaxPhraseWords = %d, want 2", got)
	}
}

func TestEmptyVocabularyHasNothingAndNeedsNoLookahead(t *testing.T) {
	v := &memVocabulary{}
	if v.Has("") || v.Has("obsequious") || v.MaxPhraseWords() != 0 {
		t.Error("an empty vocabulary must answer no to everything")
	}
}
```

- [x] **Step 2: Run to verify it fails.** `go test ./cmd/define/ -run TestMemVocabulary` → build failure, `undefined: memVocabulary`.

- [x] **Step 3: Implement.** Interface + `memVocabulary` (a `map[string]bool` plus `maxWords int`, mutex-guarded because `Add` is called from the capture path while the editor reads). `Add` routes through `store.Key`; `Has` takes an already-normalised key.

- [x] **Step 4: Run to verify it passes.**

- [x] **Step 5: Add `storeVocabulary`, EMBEDDING `memVocabulary`.** It contributes exactly one thing — `Load()` reads `Deck()` once and `Add`s each `Word.Text`; `Add`/`Has`/`MaxPhraseWords` are inherited, so the set has one implementation rather than two that must agree (ARCH-DRY). On a `Deck()` error it warns via the injected writer and leaves the set empty: a deck that cannot be read degrades to "nothing highlighted", never to a crash. Mirror `storeHistory`'s `loaded` guard so a second `Load` is free.

- [x] **Step 6: Test the degrade path** with a store whose `Deck()` returns an error; assert a warning is written and `Has` answers false rather than panicking.

- [x] **Step 7: Commit.** `#21 M1: the Vocabulary seam — a predicate, so #22 swaps one place`

### Task 2: The tokenizer

**Files:**
- Create: `cmd/define/highlight.go`
- Test: `cmd/define/highlight_test.go`

- [x] **Step 1: Write the failing tests.** Strategy: a table pinning the word-character DECISIONS (apostrophe and hyphen are inside a word, so `don't` and `hot-dog` are each one token; punctuation and whitespace are not), plus a property for the offsets. Do not enumerate every punctuation mark — the property covers the class.

- [x] **Step 2: Add `FuzzWordRuns`.** `wordRuns` is the byte-offset source of truth for both consumers, so it needs a property of its own: offsets strictly increasing, runs non-overlapping and non-empty, every run slicing the input without panicking, and every sliced run containing only word characters. Seed with multi-byte input (`café`, `¿qué`) — byte offsets over multi-byte runes are where this breaks.

- [x] **Step 3: Run to verify they fail.**

- [x] **Step 4: Implement `wordRuns`** returning `[]run{start, end int}` byte offsets, using `unicode.IsLetter || unicode.IsDigit || r == '\'' || r == '-'`.

- [x] **Step 5: Run to verify they pass.**

- [x] **Step 6: Commit.** `#21 M1: one tokenizer, so the prompt and definitions agree on where a word is`

### Task 3: `highlightSpans`, with longest-match phrases

**Files:**
- Modify: `cmd/define/highlight.go`
- Test: `cmd/define/highlight_test.go`

- [x] **Step 1: Write the failing tests**

```go
func TestHighlightSpansMarksAKnownWord(t *testing.T) {
	v := vocab("obsequious")
	got := highlightSpans("his obsequious manner", v)
	want := []span{{"his ", false}, {"obsequious", true}, {" manner", false}}
	// assert element-wise
}

func TestHighlightSpansPrefersTheLongestPhrase(t *testing.T) {
	v := vocab("hot", "hot dog")   // BOTH present: the fixture must be able to
	                               // tell the two rules apart, or it is a
	                               // tautology (lessons.md, #20).
	got := highlightSpans("one hot dog please", v)
	// "hot dog" is one known span, NOT "hot" followed by unknown " dog"
}

func TestHighlightSpansIsCaseInsensitive(t *testing.T) { /* "Obsequious" matches */ }

func TestHighlightSpansLeavesInflectionsAlone(t *testing.T) {
	// deck: obsequious; text: "obsequiousness" -> no known span. The operator
	// chose exact match over a suffix list; this is the pin on that decision.
}

func TestHighlightSpansIgnoresPunctuationAroundAWord(t *testing.T) {
	// "(obsequious)," -> the parens and comma are unknown spans, the word is known
}
```

- [x] **Step 2: Run to verify it fails.**

- [x] **Step 3: Implement.** Walk `wordRuns`; at each token try phrases of `MaxPhraseWords()` tokens down to 1, joining with single spaces to form the candidate key, and take the first `Has` hit. Emit the text between the previous emit point and the match as an unknown span, the match as known, and continue after it.

- [x] **Step 4: Run to verify it passes.**

- [x] **Step 5: Add `FuzzHighlightSpans`.** Assert: concatenated span texts equal the input exactly; no span has empty text; no two adjacent spans share a `known` value (spans are maximal, so a bug that emits one span per character is caught). Seed with the table cases plus `""`, `"   "`, `"café"`, `"a-b'c"`.

- [x] **Step 6: Run the fuzzer** for 45s; expect clean.

- [x] **Step 7: Commit.** `#21 M1: highlightSpans — the one answer to "which words here are yours"`

### Task 4: The typed line

**Files:**
- Modify: `cmd/define/editor.go` (`RenderLine`, palette)
- Modify: `cmd/define/replraw.go` (pass the vocabulary)
- Test: `cmd/define/editor_test.go`

- [x] **Step 1: Write the failing test.** Drive `RenderLine` with a vocabulary containing `obsequious`, a line `what is obsequious`, `color=true`; assert the output contains `knownOn + "obsequious"` and that the run AFTER it returns to `inputOn` — the resume is the part a naive implementation gets wrong, and it is what keeps the rest of the line bold.

- [x] **Step 2: Write the no-colour test.** `color=false` must contain no escape codes at all. Assert on the absence of `"\x1b"`, not on the absence of the green code specifically — a test that only checks for green passes while emitting bold.

- [x] **Step 3: Run to verify both fail.**

- [x] **Step 4: Implement.** Add `knownOn = "\x1b[1;32m"`. Change `RenderLine(e Editor, sug string, v Vocabulary, color bool)`; it computes its own spans from `e.Line` — never accepts a precomputed list, for the reason `draw` does not accept a precomputed match list (#15/#20: a stale list rendered against the previous line). `v == nil` means no highlighting.

- [x] **Step 5: Run to verify they pass.** Fix the cursor-parking arithmetic if it moved: the parked position is computed from *rune counts of the typed text*, which highlighting must not change. There is an existing assertion on this; if it does not cover a highlighted line, add one.

- [x] **Step 6: Wire `deps.vocab`** in `main.go` beside `deck`, built by `newStore`; `runEditor` calls `vocab.Load()` next to `hist.Load()`.

- [x] **Step 7: In-session growth.** In `storeCapturer.Capture`, when the decision is `captureEventAndWord`, also `vocab.Add(word)`. One recorder, and it is the only place that already knows a lookup both succeeded and earned a deck entry. Test: look up a word through the fake, assert it highlights on the next render without a reload.

- [x] **Step 8: Mutation-check.** (a) `knownOn` → `inputOn`: the highlight test must redden. (b) drop the style-resume after a known span: the "returns to bold" assertion must redden. (c) `Has` always false: the highlight test reddens and the no-colour test does not. Each mutation must kill a *named* test; a mutation that kills nothing means the assertion is decorative.

- [x] **Step 8b: Update `atlas/define.md` with M1's surface.** The plan originally deferred all atlas work to M3 Step 7; AGENTS.md §8 requires it at EACH milestone close, and the close gate enforces it. M1 introduces the predicate seam, the matcher and the tokenizer — real architectural surface, and deferring it is exactly the end-of-project sweep §8 forbids. M2 and M3 extend the same section.

- [x] **Step 9: `sdlc milestone-close --issue 21 --milestone M1`,** fix findings, commit with the verdict trailer.

## Chunk 2: M2 — `highlightWriter` and definitions

### Task 5: SGR tracking

**Files:**
- Modify: `cmd/define/highlight.go`
- Test: `cmd/define/highlight_test.go`

- [x] **Step 1: Write the failing tests.** Strategy: one table over "sequence in → resume code out", whose rows are the DISTINCTIONS (SGR sets the resume; reset clears it; a non-SGR CSI leaves it alone; an unterminated escape is retained rather than read as text). One row per rule, not one per escape code.

- [x] **Step 2: Run to verify it fails. Step 3: Implement. Step 4: Verify passing.**

- [x] **Step 5: Commit.** `#21 M2: remember the enclosing style, because ANSI does not nest`

### Task 6: `highlightWriter`

**Files:**
- Modify: `cmd/define/highlight.go`
- Test: `cmd/define/highlight_test.go`

- [x] **Step 1: Write the failing tests.** Strategy: one BYTE-EXACT table over the contract rules above — a known word in one call; the same word split across two `Write` calls; an escape split across two calls; a known word inside a styled run (asserting the enclosing style resumes after it); and the rule-1 ordering case, deck `hot dog` against `\x1b[1;36mhot\x1b[0m dog`, whose whole point is that the reset must not move. Assert full output bytes, per contract rule 3 — escape-stripped comparison cannot see a reorder.

- [x] **Step 2: Write the downstream-contract tests** (rule 4), against a writer that (a) short-writes and (b) returns an error mid-word: assert no byte is emitted twice across the retry, that the error reaches the caller, and that the returned count is in the caller's units. Reuse `crlf_test.go:50,:78`'s short-writer fixture rather than writing a second one.

- [x] **Step 3: Run to verify they fail. Step 4: Implement.** Hold back from the first token that could still begin a phrase; resolve when the window reaches `MaxPhraseWords()` tokens, or when contract rule 2's window-breaker arrives, or on flush.

- [x] **Step 5: Verify passing.**

- [x] **Step 6: Add `FuzzHighlightWriter`.** Split a random input at a random index into two writes; assert the flushed bytes are IDENTICAL to writing it in one call. Chunk-independence is the property, and byte identity — not visible-text equality — is what makes it able to see a reorder.

- [x] **Step 7: Commit.**

### Task 7: Definitions

**Files:**
- Modify: `cmd/define/main.go:488`
- Test: `cmd/define/render_test.go` or `highlight_test.go`

- [x] **Step 1: Write the failing test.** Render a real parsed entry with a vocabulary containing a word that appears in its *body*, assert the body occurrence is highlighted.

- [x] **Step 2: Write the invariant test.** `invariant_test.go` holds rendered letters/digits against the raw entry as an ordered subsequence. Add a case with highlighting ON. If the existing helper does not strip escapes, it must — otherwise green codes read as data and the invariant is meaningless.

- [x] **Step 3: Run to verify. Step 4: Implement** via `highlightText(s, v, on)` — `highlightWriter` over a `bytes.Buffer`, flushed. When colour is off, do not wrap at all: a pass-through that emits nothing is weaker than not being in the path.

- [x] **Step 5: Verify, including `-no-color` and piped output emitting zero escapes.**

- [x] **Step 6: Mutation-check** that the invariant test bites: make `highlightText` drop a byte and confirm it reddens.

- [x] **Step 7: `sdlc milestone-close --issue 21 --milestone M2`.**

## Chunk 3: M3 — streamed answers

### Task 8: The answer stream

**Files:**
- Modify: `cmd/define/ask.go` (the `Stream` sink, ~:160)
- Test: `cmd/define/askrun_test.go`

- [ ] **Step 1: Write the failing test — but NOT by scripting the answer.** `llmtest.Fake` cannot serve invented streamed text: `misapplied()` (`internal/llm/llmtest/fake.go:188-215`) rejects a scripted `Reply{Text}` on a streaming request with a 400, and `serveStream` (`:461-472`) always replays the committed capture. So the test must take its word FROM the capture. Add a helper that reads `stream-sample.sse`, reconstructs the full text and the delta boundaries, and returns a word that a boundary splits — the capture currently splits `rather` (`...authority r` / `ather than`) and `painstakingly` (`(painstak` / `ingly`), but derive it, do not hardcode it. Seed the vocabulary with what the helper returns. If the helper finds no split word, `t.Fatalf` with "re-record the capture or pick another" — a `t.Skip` there would let the test go quietly inert, which is the failure mode this plan's own lessons keep naming.

- [ ] **Step 2: Run to verify it fails. Step 3: Implement** by wrapping `out` in a `highlightWriter` for the duration of the stream.

- [ ] **Step 4: Flush on every exit path.** The stream ends normally, on `ErrTruncated`, and on interrupt (`askScoped` cancels mid-stream). Each must flush, or the last partial token is silently dropped — text loss, the worst failure this feature can have. Enumerate the paths from `runAsk` and test each; do not assume the happy path covers them.

- [ ] **Step 5: Interaction with `crlfWriter`.** The raw loop already wraps `out`. Determine and TEST the order: highlighting must see logical text, and CRLF translation must apply to the final bytes, so `highlightWriter` wraps *inside* `crlfWriter`. Assert a highlighted word emitted during raw mode carries correct line endings.

- [ ] **Step 6: Mutation-check** that dropping the flush on the interrupt path reddens a named test.

- [ ] **Step 7: Atlas + README.** `atlas/define.md`'s highlight section (started at M1, extended at M2) gains the streaming half. The README paragraph ALREADY EXISTS as of M1 and covers the typed line only — WIDEN it to name definitions and answers; do not treat this step as spent because a paragraph is there. README gains a line under "On a terminal". Note explicitly that the set is the whole deck *today* and #22 narrows it.

- [ ] **Step 8: `sdlc close --issue 21 --verified '<evidence>'`.**

---

## Risks

- **Text loss is the real danger.** Every other bug is cosmetic; a dropped byte in a definition or an answer is not. The no-data-loss invariant and the escape-stripped-visible-text assertions are the defence, and they belong in M2 before the streaming path exists, not after.
- **`MaxPhraseWords` grows with the deck.** A deck containing one long phrase makes the writer hold back that many tokens on every stream. Acceptable at real deck sizes; worth a note if a phrase of five or more words ever lands.
- **Green on green.** Definition examples are already italic green (`\x1b[3;32m`). A known word inside an example will be bold green inside italic green — legible, but the one place the palette nearly collides. Check it on a real terminal during M2 rather than reasoning about it.

---

## Revisions

### 2026-08-26 — plan-quality round 1

Six findings, all accepted; the three blocking ones were real holes.

- **PQ-1 `fake-capability-mismatch`** — addressed. M3's end-to-end test was
  written against a capability `llmtest.Fake` does not have: it rejects a
  scripted reply on a streaming request and always replays the committed
  capture. Verified both by reading the fake and by dumping the capture's delta
  boundaries. The test now derives its seeded word from the capture, and fails
  loudly rather than skipping if no word is split.
- **PQ-2 `writer-emission-order`** — addressed as the class. Two rules I had
  written separately ("pass escapes through", "hold back a possible phrase")
  contradict whenever an escape arrives mid-hold, and my proposed properties were
  both blind to the resulting reorder. Rather than patch the one case, the
  writer's contract is now a normative section: nothing overtakes held text, and
  the phrase window is broken by anything but spaces and tabs. That single rule
  also disposes of PQ-6.
- **PQ-3 `writer-downstream-contract`** — addressed. The plan said "returns
  `len(p)` and a nil error", which reads as "always nil". The writer will wrap
  `crlfWriter`, which short-writes and is tested for it, and byte loss is the
  worst failure this feature can have. Contract rule 4 now specifies error
  propagation, caller-unit counts, and no double-emit on retry, with a test
  against a short/erroring writer reusing the existing fixture.
- **PQ-4 `test-enumeration-in-prose`** — addressed. Tasks 2, 5 and 6 now name a
  strategy per risky function instead of listing cases, and `wordRuns` — the
  byte-offset source of truth for both consumers — gained its own fuzz property.
- **PQ-5 `duplicate-implementation`** — addressed. `storeVocabulary` embeds
  `memVocabulary`, so `Add`/`Has`/`MaxPhraseWords` have one implementation.
- **PQ-6 `unstated-matching-rule`** — addressed by PQ-2's rule: a phrase may not
  span a line break, so a wrapped `hot\n  dog` cannot paint a green run through
  the wrap indent.

### 2026-08-26 — M1 boundary review

Verdict FIX-THEN-SHIP; eight findings, two blocking. Both blocking ones were the
same rule, and the reviewer named the enumeration to sweep: *for each behaviour
the diff states in a comment, is there a mutation that makes it false and a named
test that reddens?*

- **BR-1a/b (Important)** — the set→screen link, which IS the M1 Done-when, was
  pinned by nothing. Passing `nil` instead of the vocabulary at both `RenderLine`
  call sites, and deleting `voc.Load()`, each survived the whole suite: every
  test called `RenderLine` directly and so could not see the loop's wiring break.
  Exactly #20's `typeKeys` class. Two loop-level tests added, one of them driving
  the in-session-growth path end to end. This is also **Task 4 Step 7, which the
  milestone skipped** — what shipped asserted the capturer→set mechanism, not the
  set→screen link the step specified.
- **BR-2 (Important)** — `Capture`'s "must not claim a word the deck rejected"
  was a comment with no test. Added one, and it was **vacuous on the first
  attempt**: `failingStore` fails `AppendEvent` too, so `Capture` returned before
  reaching the deck write and the mutant survived anyway. Needed a double that
  fails ONLY `Upsert`. Fourth vacuous assertion caught by mutation this session.
- **Minors, all addressed** — the `loaded` guard survived removal because `Add`
  is idempotent (now counted with a `countingDeck` double); `gofmt` flagged
  `capture.go`; `isWordRune` admitted `'` and `-` at token EDGES, so
  `'obsequious'` never matched — real before M2, since bodies quote and dash
  routinely, now trimmed; `MaxPhraseWords` counted with `strings.Fields`,
  disagreeing with `wordRuns` (now counted with `wordRuns`, and the
  punctuated-key limitation is pinned as known behaviour rather than left for M2
  to rediscover); the atlas claimed the definition path in the present tense;
  dead `_ = unicode.IsLetter` scaffolding dropped.
- **README** — the reviewer noted the plan defers it to M3 while #20's analogous
  prompt-line behaviour IS documented, and Step 8b had already overridden the
  identical atlas deferral. Written now for the same reason.

### 2026-08-26 — Task 4 Step 2 was infeasible as written

The step said to assert the no-colour render contains no `\x1b` at all. It cannot:
`eraseLine` is `\r\x1b[K` and the cursor park is `\x1b[ND`, both structural and
both present with colour off. The shipped test asserts the absence of each STYLE
constant instead, which is the property that was meant.

### 2026-08-26 — M1 boundary round 2 (REWORK), and the wordRuns contract change

- **The `wordRuns` contract changed at round 1 and only one of its three
  statements was updated.** The BR-5 trim makes runs non-maximal by design
  whenever a joiner sits adjacent. The new contract: *maximal runs of word
  characters, with leading and trailing joiners trimmed — therefore NOT maximal
  when a joiner is adjacent.* Corrected here in Core concepts (the row now reads
  `[]wordRun` and states the trim), in the doc comment, and in
  `FuzzWordRuns`, whose maximality property asserted the OLD contract and was
  **red at HEAD**. It stayed invisible because `go test` runs a fuzz target
  against its seed corpus only and no seed placed a joiner beside a kept run.
  Property rewritten to "maximal modulo the trim, and no run edge is a joiner",
  seeds gained `'0`, `-a`, `a-`, `'obsequious'`, and re-fuzzed for real: 2.24M
  execs clean.

- **The boundary's test-completeness rule is replaced, not amended.** Round 1
  stated it as an enumeration over COMMENTS. Round 2 found the same family at
  full strength, because `withStore`'s vocab merge carries no comment and makes
  no claim — a comment-driven sweep is structurally blind to it. The rule M2 and
  M3 inherit is the production-chain enumeration: for every seam, write out each
  hop from construction to use and require one test per hop that crosses it
  through production code. The five-hop `Vocabulary` table is in lessons.md.
  Hop 2 is now pinned by `TestWithStoreCarriesTheHighlightSetThrough`, built on
  the `deps{newStore: openStore}.withStore(...)` pattern BR-31 already left in
  `command_test.go:137`.

- **Task 8 Step 7's README job is now partly spent, and was spent wrongly.** A
  README paragraph was written at M1 that described definition bodies and
  streamed answers as highlighting — surface M1 does not build. It has been
  rewritten to cover the typed line only. Step 7's remaining job is to WIDEN that
  paragraph as M2 and M3 land, not to write a new one; without this note M3 would
  find the step looking done and never revisit the sentence.
