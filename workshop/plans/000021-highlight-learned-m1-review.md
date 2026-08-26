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

---

## Re-review — 2026-08-26T15:11:28-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 21 — highlight the words you are learning wherever they appear |
| repo | tools |
| issue file | workshop/issues/000021-highlight-learned.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | a0ae0d43301570e6b1895c43f8e8ff08f37822b5^..83d0322d974e0c3d6f62951ab8814f0c6f7ceea5 |
| command | sdlc milestone-close --issue 21 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-26T15:11:28-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All eight round-1 findings are genuinely fixed, and I verified each by reverting it in a scratch copy rather than trusting the commit message — six mutations, six deaths against named tests. The pure core is solid (`FuzzHighlightSpans` clean at 1.3M execs, `-race` clean, `gofmt`/`vet` clean, suite green). What blocks the boundary is that the BR-5 fix — trimming joiners off token edges — changed `wordRuns`' contract and **broke its own fuzz property**: `FuzzWordRuns` fails in 0.34s on `"'0"`, and on `'obsequious'`, the exact input the fix was written for. `go test` stayed green because fuzz targets only run their seed corpus, so a committed red test shipped inside the commit that closed a round about tests that cannot fail. Beyond that, the family sweep the last round asked for was performed over *comments*, and that enumeration is structurally blind to the one remaining hole: `withStore`'s vocab merge — the sole production link between `openStore`'s wired set and the renderer — can be deleted and the whole suite stays green.

## 1. Strengths

- **The BR-1 fix was designed to kill its specific mutant, and does.** `TestEditorLoopHighlightsADeckWordOnScreen` (`highlight_test.go:265`) deliberately uses a `storeVocabulary` over a seeded `store.NewMem()` rather than a pre-filled `memVocabulary` — which is what makes deleting `voc.Load()` redden it. Both mutations the round named die (MUT-A, MUT-B).
- **BR-2's fix required, and got, an isolating double.** `deckRejects` (`vocab_test.go:181`) fails only `Upsert`, embedding a working `store.NewMem()` so `AppendEvent` still succeeds and `Capture` actually reaches the deck write. The Log is honest that the first attempt was vacuous for exactly this reason, and `workshop/lessons.md:889` records the general rule. Three purpose-built doubles now exist — `errDeck`, `deckRejects`, `countingDeck` — each isolating one exit.
- **`highlightSpans` handles the empty-vocabulary case correctly and for free** (`highlight.go:117-126`): `maxWords == 0` makes the inner loop never execute, so an empty deck costs no `store.Key` calls at all.
- **`TestHighlightingDoesNotMoveTheCursor`** (`highlight_test.go:229`) pins the right property the right way — comparing the park sequence of a highlighted render against an unhighlighted one, rather than asserting a magic column count.
- **The atlas section added at `atlas/define.md:422-465` is architecture, not restatement** — it explains *why* the seam is a predicate and why ANSI's lack of nesting shapes every renderer, which is what M2 will actually need.

## 2. Critical findings

**a. `FuzzWordRuns` is red at HEAD and was never re-run after the fix that broke it — `cmd/define/highlight_test.go:83`, `:89`.**

```
$ go test ./cmd/define/ -run XXX -fuzz FuzzWordRuns -fuzztime 90s
--- FAIL: FuzzWordRuns (0.34s)
    highlight_test.go:83: run 0 of "'0" is not maximal on the left
```

The property asserts every run is maximal — the rune before `r.start` and after `r.end` must not be a word rune. The BR-5 fix (`appendTrimmed`, `highlight.go:65-76`) deliberately trims `'` and `-` off token edges, which makes runs non-maximal *by design* whenever a joiner sits adjacent:

| input | run | left neighbour | property |
|---|---|---|---|
| `'0` | `"0"` | `'` (word rune) | **fails** |
| `'obsequious'` | `"obsequious"` | `'` (word rune) | **fails** |
| `-a` | `"a"` | `-` (word rune) | **fails** |
| `a-` | `"a"` | right: `-` | **fails** |

The production behaviour is correct — trimming is what makes `'obsequious'` match a deck key, which is the whole point of BR-5. It is the property that is stale. It stayed invisible because `go test` runs a fuzz target only against its seed corpus, and the seeds (`"a-b'c"`, `"--"`, `"café"`, …) all happen to avoid a joiner adjacent to a kept run.

Fix sketch — replace the two maximality checks with the invariant the trimmed tokenizer actually holds: *no run's neighbour is a **non-joiner** word rune*, i.e. `isWordRune(neighbour) && !isJoiner(neighbour)` is the failure condition. Then re-fuzz (`-fuzztime 90s`) and commit any corpus entry it finds. Note `FuzzHighlightSpans` is unaffected and clean.

This is a new family, `property-test-not-rerun-after-change`: the rule is that a change to a function must re-run the property test guarding it in the same round, because the default test run does not exercise fuzz targets beyond seeds. It is not `behaviour-claimed-without-a-failing-test` — here the test exists and *does* fail; it was simply never run.

## 3. Important findings

**a. `withStore`'s vocabulary merge is unpinned; deleting it kills highlighting in production and the entire suite stays green — `cmd/define/main.go:133-135`.**

Mutation (MUT-G): replace `d.vocab = orElse[Vocabulary](sd.vocab, &memVocabulary{})` with `d.vocab = &memVocabulary{}`. Full suite: **green** (the only failures are the four `TestNoCommitted*` git-hygiene tests, which fail identically in an unmutated scratch copy with no `.git`). In production this is total feature death: `main.go:359` is the single site (`d = d.withStore(opt, stderr)`), so the capturer would hold `openStore`'s real set while the renderer read a fresh empty one — deck words never highlight, and in-session lookups never highlight either.

> **This is the 4th finding in family `behaviour-claimed-without-a-failing-test`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

**The rule, and why the last sweep could not have found this.** Round 1 stated the enumeration as *"for each behaviour the diff states in a comment, is there a mutation that falsifies it and a named test that reddens?"* — an enumeration over **comments**. `withStore`'s vocab merge carries no comment and makes no claim, so a comment-driven sweep is structurally blind to it. That is why the family recurred at full strength after being declared swept.

The rule that covers all four: **enumerate the production dependency chain, not the comments.** For each seam a feature introduces, write out every hop from construction to use, and require one test per hop that crosses it *through production code* rather than injecting past it. For `Vocabulary`:

| # | hop | pinned by |
|---|---|---|
| 1 | `openStore` builds one set, hands it to capturer + `storeDeps` | `TestOpenStoreSharesOneHighlightSet` |
| 2 | `withStore` merges `sd.vocab` → `deps.vocab` | **nothing** |
| 3 | `runEditor` reads `d.vocab`, calls `Load()` | `TestEditorLoopHighlightsADeckWordOnScreen` |
| 4 | `RenderLine` consumes it → stdout | same |
| 5 | `Capture` → `Add` → next frame | `TestEditorLoopHighlightsAWordLookedUpThisSession` |

Hop 2 is unpinned *precisely because* both new loop tests set `rig.deps.vocab` directly — they begin after hop 2. Writing the chain out is what makes the hole visible; writing the comment enumeration is what hid it. Prevalence: 4 findings across 2 rounds, all one missing-hop shape. Supporting evidence that this hop is a known hazard in this package: `command_test.go:130` already carries a comment recording BR-31, where "a fix that exempted commands from `withStore` dropped an invariant" — same function, same class, a previous issue. The pattern that pins it already exists at `command_test.go:137`: `deps{newStore: openStore}.withStore(options{}, io.Discard)`.

**b. README documents definition bodies and streamed answers as highlighting, which M1 does not build — `README.md:54-57`.**

> "**Words you have looked up show in green.** Anywhere they appear — the line you are typing, definition bodies, and answers…"

At HEAD `highlightWriter`, `sgrState` and `highlightText` do not exist, and `highlightSpans` has exactly one production caller — `editor.go:206`, the typed line. The definition print (`main.go:502`) goes straight through `Render(...)`; the answer stream is untouched. A reader who follows the README, looks up a word and reads a definition sees no green.

> **This is the 2nd finding in family `atlas-claims-unbuilt-surface`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

**The rule:** *doc prose written at a milestone boundary describes only what that milestone shipped; a surface a later milestone builds is marked as future.* The enumeration it implies: every doc file the boundary window touches × every sentence describing the feature, each checked against what is reachable in code at HEAD. Applied now — `atlas/define.md:444` ✓ (correctly reads "from M2"), `README.md:54-57` ✗.

This is the ARCH-PURPOSE signature and worth naming plainly: commit `83d0322` fixed the atlas sentence BR-6 named **and wrote a fresh instance of the same defect into README.md in the same commit**. The site was fixed; the class was never enumerated.

## 4. Minor findings

- `cmd/define/vocab.go:100` — `storeVocabulary.loaded` is an unguarded `bool` mutated in `Load()`, embedded in a type whose doc comment (`vocab.go:46-47`) advertises mutex-guarded concurrent access. Harmless today (single `Load` call site, on the loop goroutine, before the loop starts — `-race` is clean), but the type now advertises a safety property one of its own fields does not hold.
- `cmd/define/vocab.go:76` — a key that can never match still inflates `MaxPhraseWords`: `e.g.` counts as 2 tokens, so every deck holding one makes `highlightSpans` try a 2-token candidate at every position forever. Correctly documented, but it becomes a real cost in M2, where `MaxPhraseWords` drives stream hold-back.

**Plan-record drift** — `workshop/plans/000021-highlight-learned-plan.md:34` still reads ``**`wordRuns(text string) []run`** — the tokenizer: maximal runs of word characters…``. Code returns `[]wordRun`, and the runs are no longer maximal.

> **This is the 2nd finding in family `plan-record-not-updated`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

**The rule:** *the durable plan is a record of the code, so a change that alters an entity's signature or contract updates the Core-concepts row in the same commit, with a `## Revisions` entry whenever the change was not the plan's original intent.* The enumeration: for each table row, does the entity exist at the stated path, with the stated name, type and described contract? Run at HEAD, every other row checks out (`sgrState`/`highlightWriter` correctly absent — M2). Worth noting the root cause is shared with Critical (a): **three artifacts state `wordRuns`' contract — the plan table, the fuzz property, and the doc comment — and the BR-5 fix updated only the doc comment.** That is one enumeration, not three findings.

## 5. Test coverage notes

Every mutation below was run against the full `./cmd/define/` suite in a clean scratch copy of HEAD. The four `TestNoCommittedBinaries`/`TestNoBinariesInHistory`/`TestNoTrackedRuntimeState`/`TestNoRuntimeStateInHistory` failures appear in *every* scratch run including the unmutated control (no `.git` present) and are excluded as environment noise.

| mutation | result | killed by |
|---|---|---|
| `nil` for `voc` at both `RenderLine` call sites | **dies** | `TestEditorLoopHighlightsADeckWordOnScreen`, `…AWordLookedUpThisSession` |
| delete `voc.Load()` (`replraw.go:82`) | **dies** | `TestEditorLoopHighlightsADeckWordOnScreen` |
| hoist `vocab.Add` above the `Upsert` error check | **dies** | `TestCaptureDoesNotAddAWordTheDeckRejected` |
| drop the `loaded` guard (`vocab.go:112`) | **dies** | `TestStoreVocabularyReadsTheDeckOnlyOnce` |
| remove joiner trimming from `appendTrimmed` | **dies** | `TestWordRuns` |
| remove `vocab.Add` from `Capture` entirely | **dies** | 3 tests |
| drop the blank-key guard in `Add` | **dies** | `TestMemVocabularyIgnoresBlankWords` |
| `withStore` stops propagating `sd.vocab` | **survives** | — (Important a) |

Other verification: `go vet` clean; `gofmt -l ./cmd ./internal` empty (BR-4 confirmed); `go test -race ./cmd/define/` passes in 65s; `FuzzHighlightSpans` 90s / 1.3M execs / 65 new interesting inputs, no failures; `FuzzWordRuns` **fails in 0.34s** (Critical a). I removed the `cmd/define/testdata/fuzz/FuzzWordRuns/` artifact my run created — the tree is as I found it, and the failure reproduces from clean in under a second.

## 6. Architectural notes

- **ARCH-DRY — pass.** `storeVocabulary` embeds `memVocabulary` (`vocab.go:97`), so the set has one implementation; `wordRuns` is one tokenizer; `nonEmptySpan` holds the no-empty-span rule in one place. The `&memVocabulary{}` fallback now appears at four sites (`main.go:134`, `:167`, `:172`, `replraw.go:80`), mirroring the established `memHistory` idiom — still not drift, but M2 adds a fifth and that is the point to extract a shared null object.
- **ARCH-PURE — pass.** `wordRuns`, `appendTrimmed`, `phraseGap`, `phraseRunsJoin`, `nonEmptySpan`, `highlightSpans`, `RenderLine` are deterministic; their tests run against `memVocabulary` with no clock, filesystem or network. IO is confined to `storeVocabulary.Load`'s single `Deck()` call and `storeCapturer.Capture`. The plan's PURE/INTEGRATION split matches the code at every M1 row.
- **ARCH-PURPOSE — flag.** The shadow-sweep over M1's one consumer (`RenderLine`) shows both call sites deriving from `d.vocab`, no hand-maintained restatement of the set anywhere, and Task 4 Step 7 — the step the last round found skipped — now genuinely delivered. The flag is the *fix-the-instance-not-the-class* pattern, twice in one commit: the atlas sentence was corrected while the identical defect was written into README (Important b), and the comment-scoped enumeration was swept while the production-chain hop it could not see was left open (Important a).
- **ARCH-MOCK — pass.** `store.NewMem()` is the stateful fake behind the same `store.Store` seam production uses; `errDeck`, `deckRejects` and `countingDeck` each model one specific failure/observation state rather than blanket-failing. `TestOpenStoreSharesOneHighlightSet` boots the real `openStore` against a real YAML store under `t.Chdir(t.TempDir())` — portable, non-production storage, production and test flow crossing the same boundary. No new external binary or service.
- **For M2:** `MaxPhraseWords` is about to become load-bearing — it drives how many tokens `highlightWriter` holds back on every write. Two things make it noisier than it looks: an unmatchable punctuated key (`e.g.`) inflates it permanently, and the plan's own Risks section already flags long phrases. Consider having `Add` skip the `maxWords` bump for keys whose `wordRuns` cannot re-form the key (i.e. keys that `phraseGap` can never rejoin), which would keep the look-ahead honest.

## 7. Plan revision recommendations

1. **A `## Revisions` entry for the `wordRuns` contract change.** The BR-5 fix made runs non-maximal at joiner edges. That contract is stated in three places and only one was updated. The entry should record: the new contract ("maximal runs of word characters, with leading and trailing joiners trimmed — therefore NOT maximal when a joiner is adjacent"), that the Core-concepts row at line 34 is corrected to `[]wordRun` and re-worded, and that `FuzzWordRuns`' maximality property was updated and re-fuzzed. This is the fix for both Critical (a) and the Minor plan-record item.
2. **A `## Revisions` entry recording the boundary's test-completeness rule, replacing the comment-scoped one.** The existing M1-boundary entry records the round-1 enumeration as *"for each behaviour the diff states in a comment…"*. That enumeration is now known to be blind to unclaimed wiring hops. Replace it with the production-chain enumeration in Important (a), including the five-hop `Vocabulary` table, so M2/M3 inherit the working rule rather than the one that let hop 2 through.
3. **Task 8 Step 7's README line is now partly spent, and wrongly.** The step says "README gains a line under 'On a terminal'." That line has already been written at M1 and over-claims M2/M3 behaviour. Record that the README paragraph exists as of M1 covering the typed line only, and that Step 7's remaining job is to *widen* it as M2 and M3 land — otherwise M3 finds the step already ticked-looking and never revisits the sentence.
