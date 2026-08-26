# Boundary Review — tools#21 (milestone M1)

| field | value |
|-------|-------|
| issue | 21 — highlight the words you are learning wherever they appear |
| repo | tools |
| issue file | workshop/issues/000021-highlight-learned.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | a0ae0d43301570e6b1895c43f8e8ff08f37822b5^..740e7faacaf873b73af515428a08f05e6f42969e |
| command | sdlc milestone-close --issue 21 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-26T14:52:43-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The pure core of this milestone is genuinely well built and genuinely well pinned: I ran six mutations against `highlightSpans`/`RenderLine` and every one died against a *named* test — including the two the Log claims were re-pinned after the `knownOn`/`inputOn` aliasing mutant survived, which I verified by reverting them in a scratch copy. Fuzzers are clean (`FuzzHighlightSpans` 438k execs, `FuzzWordRuns` 198k execs, no failures), `-race` is clean, `go vet` is clean, the full suite passes. What blocks SHIP is not correctness but proof of the *wiring*: I disabled the entire M1 deliverable in the editor loop — passed `nil` for the vocabulary at both `RenderLine` call sites in `replraw.go`, and separately deleted `voc.Load()` — and the whole suite stayed green both times. Nothing drives `runEditor` with a vocabulary and asserts a green byte reaches stdout, which is also the one Plan step (Task 4 Step 7) that this boundary claims but did not deliver. Two smaller claims are in the same class (a comment asserting behaviour that no test can fail on). All of it is cheap — `editorRig`/`runEditor` and `failingStore{}` already exist.

## 1. Strengths

- **The matcher's tests are not tautologies.** `highlight_test.go:113` seeds *both* `hot` and `hot dog` so the longest-match rule and the shortest-match rule produce different output — I inverted the loop to count up and `longest phrase wins` reddened with a legible diff. Likewise `phraseGap` accepting any gap reddened `a phrase does not span a newline`.
- **The lessons.md fix is real, not narrated.** Aliasing `knownOn = inputOn` in a scratch copy failed *two* named tests (`highlight_test.go:189` on literal escape bytes, and the `knownOn == inputOn` property at :194). The constant-vs-itself trap the Log describes is actually closed.
- **`errDeck` (`vocab_test.go:100`) embeds a nil `store.Store` deliberately**, so a `Vocabulary` that reaches past `Deck()` panics loudly instead of silently reading a zero value. That is the right shape for a failure double (ARCH-MOCK).
- **`TestOpenStoreSharesOneHighlightSet` (`vocab_test.go:145`) is live**: handing the capturer its own `newStoreVocabulary` reddens it. The atlas cites this test and the citation holds.
- **PQ-5 was genuinely delivered** — `storeVocabulary` embeds `memVocabulary` (`vocab.go:88`), so `Add`/`Has`/`MaxPhraseWords` have one implementation (ARCH-DRY).

## 2. Critical findings

None.

## 3. Important findings

**a. The editor loop's use of the vocabulary is pinned by nothing — `cmd/define/replraw.go:138`, `:216`.**
Two independent mutations survive the *entire* suite:
- replace `voc` with `nil` at both `RenderLine` call sites (highlighting fully off in production) → `ok`
- delete `voc.Load()` at `replraw.go:82` (no deck word ever enters the set) → `ok`

So the chain deck → set → capturer is pinned (`TestOpenStoreSharesOneHighlightSet`, `TestCaptureGrowsTheHighlightSet`) but the set → screen link is not, and that link *is* the M1 Done-when. This is also the plan item the boundary skipped: Task 4 Step 7 specifies "look up a word through the fake, assert it highlights on the next render without a reload" — what shipped (`vocab_test.go:112`) asserts the capturer→set mechanism instead.

Fix sketch — `editorRig` already returns `options{color: true}` and `runEditor` already takes a scripted key channel and an `out` buffer (see `editorloop_test.go:38`):
```go
rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
rig.deps.vocab = vocab("sycophantic")
var out, errb bytes.Buffer
runEditor(t.Context(), scriptKeys("sycophantic"), nil, rig.deps, opt, cooked, finish, &out, &errb)
if !strings.Contains(out.String(), "\x1b[1;32msycophantic") { t.Error(...) }
```
A second case scripting `"sycophantic\rsycophantic"` covers the in-session-growth Done-when (the first submit echoes *before* the lookup runs, so the highlight can only appear on the retype — which is exactly the behaviour worth pinning).

**b. `Capture`'s "must not claim a word the deck rejected" is a comment, not a contract — `cmd/define/capture.go:96-104`.**
Moving `c.vocab.Add(word)` above the `Upsert` error check (so a rejected write still grows the highlight set) survives the whole suite. The fixture is already in the file: `failingStore{}` (`capture_test.go:62`). Two lines:
```go
v := &memVocabulary{}
c := newStoreCapturer(failingStore{}, store.FixedClock(time.Now()), nil, v)
c.Capture("sycophantic", true, options{})
if v.Has(store.Key("sycophantic")) { t.Error("a word the deck rejected entered the highlight set") }
```

Both (a) and (b) are instances of one rule, and the enumeration is short enough to sweep in this round rather than the next: for each behaviour the diff states in a comment, is there a mutation that makes it false and a named test that reddens? Applying that enumeration across the diff turns up exactly (a), (b) and Minor (c) below.

## 4. Minor findings

- `cmd/define/vocab.go:107` — the `loaded` guard survives removal; `vocab_test.go:74`'s "second Load is free and must not double anything" cannot fail, because `Add` is idempotent and `maxWords` is a max. A counting `store.Store` double would make it bite. Same family as the two Important findings.
- `cmd/define/capture.go:70` — `gofmt -l` flags this file (`vocab Vocabulary` misaligned); it is the only unformatted file in the package. No CI gate catches it.
- `cmd/define/highlight.go:25` — `isWordRune` admits `'` and `-` at token *edges*, so `'obsequious'` tokenizes as one run and `word--word` as another; neither matches a deck key. Harmless on the prompt line, but body text in M2 routinely quotes and dashes.
- `cmd/define/vocab.go:70` — `MaxPhraseWords` counts with `strings.Fields`, which disagrees with `wordRuns` for any key holding other punctuation (`e.g.`, `9/11`): such an entry is permanently unmatchable. Edge case, worth knowing before M2 widens the input.
- `atlas/define.md:444` — "the single tokenizer both the prompt **and the definition path** use" is present tense for a path M2 hasn't built yet; the M2/M3 forward references elsewhere in the section are correctly marked as future.
- `cmd/define/highlight_test.go:92` — `_ = unicode.IsLetter` exists only to keep an otherwise-unused import; drop both.
- README is untouched. The plan schedules it at M3 Step 7, and highlighting adds no flag or keybinding, so this is defensible — but note #20's directly analogous prompt-line behaviour *is* documented under "On a terminal" (README.md:54), and Step 8b overrode the identical atlas deferral for exactly this reason.

## 5. Test coverage notes

Empirically verified this round — mutations run in a scratch copy, full suite each time:

| mutation | result |
|---|---|
| `knownOn` → `inputOn` | **dies** (2 tests) |
| drop `+ inputOn` resume | **dies** |
| `phraseGap` accepts any gap | **dies** |
| longest-match → shortest-match | **dies** |
| `openStore` hands two instances | **dies** |
| `Capture` stops calling `vocab.Add` | **dies** |
| `RenderLine(…, nil, …)` in the loop | **survives** |
| drop `voc.Load()` in the loop | **survives** |
| `Add` before the `Upsert` error check | **survives** |
| drop the `loaded` guard | **survives** |

The pure core is thoroughly covered; every surviving mutant is at the IO/wiring boundary. Fuzz corpus growth was healthy (20 new interesting inputs for `FuzzHighlightSpans`), and the span-concatenation and span-maximality invariants held throughout.

## 6. Architectural notes

- **ARCH-DRY — pass.** `storeVocabulary` embeds `memVocabulary`; `wordRuns` is one tokenizer; `nonEmptySpan` holds the empty-span rule in one place. The `&memVocabulary{}` fallback recurring at four construction sites (`main.go:134`, `:167`, `:172`, `replraw.go:80`) mirrors the established `memHistory` idiom rather than introducing new drift — not flagged, but if a fifth appears in M2/M3 it is worth a shared null object.
- **ARCH-PURE — pass.** `wordRuns`, `phraseGap`, `phraseRunsJoin`, `nonEmptySpan`, `highlightSpans` and `RenderLine` are deterministic and their tests run with `memVocabulary` and no filesystem, clock or network. IO is confined to `storeVocabulary.Load`'s single `Deck()` call and `storeCapturer.Capture`. The plan's PURE/INTEGRATION table matches the code at every M1 row (`sgrState` and `highlightWriter` are correctly absent — M2).
- **ARCH-PURPOSE — pass on scope, gap on proof.** The shadow-sweep finds one consumer at M1 (`RenderLine`), both call sites deriving from `d.vocab`; no hand-maintained restatement of the set exists anywhere. Nothing was deferred that is the point of the milestone. The gap is Important finding (a): the last link is real in code but asserted nowhere.
- **ARCH-MOCK — pass.** `store.NewMem()` is the stateful fake behind the same `store.Store` seam production uses; `errDeck` models the read-failure path; `TestOpenStoreSharesOneHighlightSet` runs the real `openStore` against a real YAML store in `t.Chdir(t.TempDir())`, so production flow and test flow cross the same boundary. No new external binary or service is introduced.
- **For M2/M3:** `highlightSpans` calls `store.Key` once per candidate per token — O(tokens × `MaxPhraseWords`) allocations. Invisible on a prompt line; a full definition body re-highlighted per stream chunk is a different profile. Worth measuring before optimising, not worth designing around now. Separately, `memVocabulary`'s mutex is currently defensive only — all three `Capture` call sites (`main.go:491`, `:499`, `:503`) are on the loop's own goroutine — so keep it, but don't let the doc comment's "genuinely concurrent" claim justify a design that assumes concurrency exists.

## 7. Plan revision recommendations

1. **Tick M1's boxes.** All 26 `- [ ]` steps in Tasks 1–4 of `workshop/plans/000021-highlight-learned-plan.md` are still unchecked at HEAD while the issue's `## Plan` says `- [x] M1`. The durable plan is the record of where the work stopped; M2's executor cannot read it as-is.
2. **A `## Revisions` entry for Task 4 Step 2.** It requires the no-colour test to "assert on the absence of `"\x1b"`", which is impossible — `RenderLine` emits `eraseLine` and the cursor-park escape on every path. What shipped (`highlight_test.go:214`, absence of each named style constant) is the right test; record that the step as written was infeasible so M2/M3 don't inherit the instruction.
3. **A `## Revisions` entry for Task 4 Step 7**, recording that its stated test ("assert it highlights on the next render") was not delivered, and what replaced it — or better, delete the entry by writing the test per Important finding (a).
4. Minor, if you touch the table anyway: the plan names the tokenizer's return type `[]run`; the code calls it `wordRun`. And Task 4 places the `RenderLine` tests in `editor_test.go`; they landed in `highlight_test.go`, which reads better — worth recording as a deliberate deviation rather than leaving the table wrong.

```findings
findings:
  - id: new
    severity: Important
    family: behaviour-claimed-without-a-failing-test
    title: |
      The editor loop's use of the vocabulary is pinned by no test — highlighting can be switched fully off and the suite stays green
    detail: |
      Two mutations survive the entire suite: replacing `voc` with `nil` at both
      RenderLine call sites (replraw.go:138, :216), and deleting voc.Load()
      (replraw.go:82). Nothing drives runEditor with a vocabulary and asserts a
      green byte reaches stdout, so the M1 Done-when has no proof. This is also
      Plan Task 4 Step 7's own stated test ("assert it highlights on the next
      render without a reload"), which was replaced by a capturer-to-set unit
      test. editorRig already supplies color:true and runEditor already takes a
      scripted key channel plus an out buffer, so the fix is ~10 lines.
  - id: new
    severity: Important
    family: behaviour-claimed-without-a-failing-test
    title: |
      Capture's "must not claim a word the deck rejected" is a comment with no test behind it
    detail: |
      capture.go:96-104 documents that vocab.Add runs only after Upsert
      succeeds, but moving the Add above the error check survives the whole
      suite. The fixture already exists — failingStore{} at capture_test.go:62 —
      so the pin costs four lines. Same rule as the editor-loop finding: sweep
      the diff for behaviours stated only in comments rather than fixing these
      two sites alone.
  - id: new
    severity: Minor
    family: behaviour-claimed-without-a-failing-test
    title: |
      storeVocabulary's once-only Load guard survives removal; the test asserting it cannot fail
    detail: |
      vocab_test.go:74 comments "second Load is free and must not double
      anything", but Add is idempotent and maxWords is a max, so deleting the
      `loaded` guard at vocab.go:107 keeps every assertion green. A counting
      store double would make the assertion bite.
  - id: new
    severity: Minor
    family: unformatted-source
    title: |
      cmd/define/capture.go is not gofmt-clean
    detail: |
      The new `vocab Vocabulary` field at capture.go:70 is misaligned; gofmt -l
      flags this file and no other in the package. No CI gate catches it.
  - id: new
    severity: Minor
    family: tokenizer-edge-admits-punctuation
    title: |
      isWordRune admits apostrophe and hyphen at token edges, so 'obsequious' and word--word never match
    detail: |
      highlight.go:25 treats ' and - as word runes unconditionally, so a quoted
      or double-dashed occurrence tokenizes into a run whose store.Key differs
      from the deck key. Harmless on the prompt line; definition bodies in M2
      routinely quote and dash. Relatedly, MaxPhraseWords counts with
      strings.Fields (vocab.go:70), which disagrees with wordRuns for keys
      holding other punctuation (e.g., 9/11) — such entries are unmatchable.
  - id: new
    severity: Minor
    family: atlas-claims-unbuilt-surface
    title: |
      atlas/define.md describes the definition path in the present tense before M2 builds it
    detail: |
      Line 444 says wordRuns is "the single tokenizer both the prompt and the
      definition path use". The M2/M3 forward references elsewhere in the same
      section are correctly marked as future; this one is not.
  - id: new
    severity: Minor
    family: plan-record-not-updated
    title: |
      All 26 M1 plan steps remain unchecked while the issue marks M1 complete
    detail: |
      workshop/plans/000021-highlight-learned-plan.md Tasks 1-4 are entirely
      `- [ ]` at HEAD. The plan also needs Revisions entries for Task 4 Step 2
      (its "assert absence of \x1b" instruction is infeasible — eraseLine and
      the cursor park are escapes) and Task 4 Step 7 (test not delivered).
  - id: new
    severity: Minor
    family: dead-test-scaffolding
    title: |
      `_ = unicode.IsLetter` in highlight_test.go:92 exists only to justify an unused import
    detail: |
      Drop the statement and the `unicode` import together.
```
