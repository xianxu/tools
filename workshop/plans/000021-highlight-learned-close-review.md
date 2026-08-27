# Boundary Review — tools#21 (whole-issue close)

| field | value |
|-------|-------|
| issue | 21 — highlight the words you are learning wherever they appear |
| repo | tools |
| issue file | workshop/issues/000021-highlight-learned.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0207e7ef24414534b308a2296fb148b1740814ee..92a39fff86e96278a4ff732202199418478ebd87 |
| command | sdlc close --issue 21 |
| reviewer | claude |
| timestamp | 2026-08-26T17:41:46-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M3 is correct and its headline wiring is genuinely pinned — I verified by mutation, not by reading commit messages: replacing the answer stream's vocabulary with `nil`, or dropping `out = hw`, each reddens three named tests; dropping the `!opt.color` gate reddens `TestStreamedAnswerCarriesNoEscapesWithoutColour`. The writer itself held up under an **independent** differential I wrote from scratch (deck derived from each text's own 1–3-token windows, N-way random chunking, 570k execs) plus the four shipped fuzz targets at ~1.1–1.25M execs each — all clean, `-race` clean, `go vet`/`gofmt` clean, suite green, tree clean. Six of nine open findings are properly closed (each verified by revert). What keeps this from SHIP is one coverage gap in the family this issue keeps recurring on: **M3 added a second render surface and the entry-path enumeration written at M2 to prevent exactly this was not widened** — deleting `Load()` from the answer path alone leaves the entire suite green, which is BR-14's shipped-Critical shape one surface over. Nothing here is a live defect; all of it is cheap.

## 1. Strengths

- **The answer path's wiring is pinned three ways over.** `cmd/define/ask.go:171` — `newHighlightWriter(out, vocabularyFor(d, opt), knownOn)`. Mutating the vocabulary to `nil` kills `TestStreamedAnswerHighlightsAWordSplitAcrossDeltas`, `TestEveryStreamExitPathFlushes` and `TestHighlightingNestsInsideCRLFTranslation`; `out = hw` → `_ = hw` kills the same three; `vocabularyFor` → `d.vocab` kills the no-colour test. All verified.
- **`splitWordInCapture` (`askhighlight_test.go:14`) derives its seed word from the artifact and `t.Fatalf`s rather than skipping.** That is the plan's Step 1 followed to the letter, and it is the difference between a test that rots loudly and one that goes inert. Measured: the capture is 5 deltas / 532 bytes and splits `rather` as `…authority r` + `ather than`.
- **The CRLF nesting assertion is not vacuous.** Measured on the production path: 6 newlines, 6 CRLF, so `strings.Count(got,"\n") != strings.Count(got,"\r\n")` is a check that can actually fire.
- **The honest-scope comment at `askhighlight_test.go:99-110` is the right call** — it records that deleting the deferred `Flush` leaves the table green and names what a fake would need to reach the fifth path, instead of implying coverage. I confirmed the claim by mutation. That is the behaviour this issue's own lessons asked for.
- **ARCH-PURPOSE shadow-sweep is clean.** One `Vocabulary.Has` call site in the whole tree (`highlight.go:132`); `knownOn` appears at exactly three production sites (`editor.go:208`, `render.go:93`, `ask.go:171`), each deriving from the seam. Done-when row 8 is genuinely satisfied, not asserted.

## 2. Critical findings

None.

## 3. Important findings

**The entry-path enumeration was not widened when M3 added a render surface — `cmd/define/ask.go:171`, `cmd/define/vocab_test.go:242`.**

> **This is the 8th finding in family `behaviour-claimed-without-a-failing-test`.** Do NOT fix only this instance.

Measured at HEAD: replace `vocabularyFor(d, opt)` in `runAsk` with an inline equivalent that keeps the `d.vocab == nil || !opt.color` gate but drops `d.vocab.Load()` — **the full suite passes**. In production that mutant means a streamed answer never highlights on the piped-stdin and one-shot question paths, because neither loads the deck before reaching `ask`; only `runEditor` does (`replraw.go:79`). That is BR-14's bug, one surface over, and it survives because every test in `askhighlight_test.go` injects `d.vocab = vocab(word)` — a pre-filled `memVocabulary` — and so begins one hop after the thing that fills it.

**The rule, widened:** the enumeration's axis is **entry path × render surface**, not entry path alone. `TestEveryEntryPathHighlightsDefinitions` covers 3 of 6 cells (definitions × three paths); answers × three paths is 0, and the atlas paragraph at `atlas/define.md:517` nevertheless calls that table the guard for "*every render path*". A surface added later gets a column, and every column needs at least one row driven with the dependency in its **real initial state**.

Fix, verified in both directions: seeding one existing test with an unloaded store vocabulary kills the mutant —

```go
st := store.NewMem(); st.Upsert(store.Word{Text: word})
d.vocab = newStoreVocabulary(st, nil)   // deliberately NOT loaded
```

I ran that against HEAD: it passes (`vocabularyFor` loads it) and it is the single assertion the mutant cannot survive. The class-level fix is to make `TestEveryEntryPathHighlightsDefinitions` a surface×path table, or add its answer-surface sibling beside it, and say so in the atlas sentence.

## 4. Minor findings

- **`askhighlight_test.go:143` — the "interrupted mid-stream" row asserts nothing.** Its only check is behind `tc.cancel && got != ""`, and with an already-cancelled context `runAsk` returns before any delta, so `out` is empty (measured: `code=0 out="" len=0`). The guard never fires. **This is the 9th finding in family `behaviour-claimed-without-a-failing-test`** — the rule the family had not yet named: *an assertion guarded on the run's own output is not an assertion until something proves the guard fires.* Measured prevalence: 4 output-conditional assertions in `cmd/define`; three (`commandloop_test.go:140,143`, `usermodel_test.go:390`) guard on a **fixture** the author controls and are legitimate; this one guards on the observed output and is the only vacuous one. The package already owns the correct idiom at 5 sites (`highlightwriter_test.go:328,433,483`, `invariant_test.go:79`, `dict_fake_test.go:69`) — a `t.Fatal` when the observable is empty. Same file, same rule: `askhighlight_test.go:163`'s "the capture's final characters, **which only a flush can emit**" is refuted by mutation — deleting `defer hw.Flush()` leaves that test green, because the trailing `Fprintln` emits it.
- **`askhighlight_test.go:66` — `func max(a, b int) int` shadows the Go builtin for the whole package's test build.** Deleting it leaves `go vet ./cmd/define/` clean (verified), so it is pure shadowing: 8 other call sites (`invariant_test.go:43,95`, `live_property_test.go:51,81,101`, `render_test.go:235`, `editorloop_test.go:155`) silently resolve to it now, and `min` beside them still resolves to the builtin. Go 1.26 per `go.mod`. **3rd in family `copy-pasted-helper`** — the rule ("grep the package for one with the same job before adding a helper") needs one word added: *the language's own builtins are part of what you are grepping.*
- **`cmd/define/ask.go:171` — every error the writer is designed to report is discarded by its only production caller.** `fmt.Fprint(out, delta)` ignores its error and `defer hw.Flush()` discards its return, so a poisoned writer silently drops the rest of an answer with no message — and M3 turns "lose one delta" into "lose the remainder", because the writer poisons on first failure by contract (`highlightwriter.go:120`). Practically unreachable with `*os.File`, but it is precisely the question the M2 round-3 review left as a note for Task 8 Step 4 ("what does the user see when the writer is already poisoned"), and nothing in the window records a decision. Related, same defer: on the `default:` path the error line reaches `errOut` *before* the deferred flush pushes held text to `out`, so the last partial word lands after the error rather than before it.
- **`atlas/define.md` §"Highlighting the words you are learning" never mentions `highlightSetFor`.** The command-namespace withhold (a deck word named `history` is not green inside `/history 7`, `highlight.go:173`) is a real shipped rule, and the section documents every other rule in the feature. New family `atlas-omits-a-shipped-rule` — the inverse of `atlas-claims-unbuilt-surface`, so it needs its own slug to match a recurrence.

## 5. Test coverage notes

- Mutations run this round (full suite each, in a scratch copy; the four `repo_guard` failures are `.git`-absence noise present in the unmutated control): **7 killed** — answer vocabulary → nil (3 tests), `out = hw` dropped (3), colour gate dropped (1), `!opt.color` reverted (2), `highlightSetFor` reverted (1), reset-clears-base reverted (2), plus the existing suite. **2 survived** — `Load()` dropped from the answer path (Important above) and `defer hw.Flush()` deleted (documented honestly in-file).
- Fuzz: `FuzzHighlightWriterIsChunkIndependent` 1.12M, `FuzzHighlightSpans` 1.25M, `FuzzWordRuns` 1.25M, `FuzzTrailingSegments` 1.24M — all clean, corpus additions in GOCACHE, tree unchanged. `fuzzDeck.MaxPhraseWords() == 3`, so BR-29's phrase-length axis is genuinely live.
- Independent property I wrote (not in the tree): deck derived from each text's own 1–3-token windows × N-way random chunk schedules, 570k execs clean; plus the real capture replayed at its actual delta boundaries with every answer word in the deck — byte-identical to one call.
- `TestEveryStreamExitPathFlushes` names five paths in prose and tables two. `Stall` and `JunkFrame` both classify as `ErrTruncated` (`llmtest/fake.go:127,138`), so the truncated row is reachable cheaply; `unavailable-after-sending` is too. That is Plan Task 8 Step 4's "enumerate the paths and test each", ticked.

## 6. Architectural notes

- **ARCH-DRY — flag (minor).** The `max` shadow above, and `warnTo`'s residue (below). Otherwise clean: one tokenizer, one matcher, one writer serving both styled surfaces, `phraseGapOrEmpty` delegating to `phraseGap`, `stripANSI` deferring to `scanEscape`. Worth noting as a seam question rather than a finding: `cmd/define/askhighlight_test.go:23-45` now decodes `llmtest`'s own SSE artifact with an ad-hoc parser, while `internal/llm/anthropic_test.go:259` does a weaker version of the same thing. `llmtest` owns the capture; "what deltas does this capture deliver" belongs behind that seam (`llmtest.CaptureDeltas`). The current drift mode is loud (`t.Fatalf`), which is why this is a note and not a finding.
- **ARCH-PURE — pass.** `wordRuns`, `highlightSpans`, `sgrState`, `scanEscape`, `decidedEnd`, `tokenStillOpen` are pure and unit-tested with no IO. `runAsk` stays the thin shell: the writer is constructed at the boundary and injected as an `io.Writer`, and `answer` accumulates **raw** deltas, so no escape can leak into `recordExchange` → `gatherAskContext` → the next prompt.
- **ARCH-PURPOSE — flag on the enumeration axis only.** The purpose is delivered: all three Spec surfaces highlight, through one predicate, on all entry paths (verified end to end against production wiring with an unloaded set). The flag is the Important finding — an enumeration that names a class and then covers a subset of it, which is the axis this issue has recurred on at every boundary.
- **ARCH-MOCK — pass.** M3 adds no external dependency and does not reach around a seam: the answer arrives through `llmtest.Fake` serving a committed capture, the test refuses to script invented streamed text (it cannot), and `crlfWriter`/`shortWriter`/`failAfter` sit exactly at the `io.Writer` boundary production uses. `store.NewMem()` remains the portable non-production backend.
- **For #22:** the swap is one constructor and the seam holds — but note `memVocabulary` has no removal path (`Add` only). Narrowing to actively-learned words means the set must *shrink* mid-session as a word graduates, and today nothing above the seam is prepared for a `Has` that goes from true to false while a stream is mid-hold. Worth deciding at #22's plan gate rather than discovering in its writer.

## 7. Plan revision recommendations

A `## Revisions` entry — "M3 / close boundary: the plan-record sweep, run to the end this time" — covering what BR-16 still names plus what M3 added:

- `plan:270` and `plan:282` — Task 5 and Task 6 `Test:` still say `cmd/define/highlight_test.go`; actual `sgr_test.go` and `highlightwriter_test.go`. (Tasks 2/3 at `:174`/`:192` are correct and should stay.)
- `plan:286` — Task 6 Step 2 still requires "the returned count is in the caller's units", which contract rule 4 (`plan:100-110`) and `TestHighlightWriterPropagatesDownstreamErrors` both contradict with `(0, err)`. Delete the clause and point the step at rule 4.
- `plan:70` — Core concepts gives the answer-stream injection point as `ask.go:160`; actual `ask.go:171`.
- **Task 8 Step 4 is ticked but its stated requirement was not delivered** — "Enumerate the paths from `runAsk` and test each" landed as 2 of 5 rows, one of which asserts nothing. Either add the two cheap rows (`Stall`/`JunkFrame` → `ErrTruncated`; unavailable-after-sending) or record the deviation.
- **Task 8 Step 6 is ticked and its mutation check fails** — "dropping the flush on the interrupt path reddens a named test": measured, deleting `defer hw.Flush()` reddens nothing. The in-file comment already says so honestly; the plan tick says the opposite. Untick or record.
- Add the surface×path widening from the Important finding as a new Task 8 step, so #22 inherits the working enumeration rather than the definitions-only one.

```findings
dispose:
  - id: BR-9
    disposition: not-addressed
    note: |
      Two of three seams wrap warnTo; capture.go:132 still writes "define: " itself, and warnTo's own comment (vocab.go:130) claims to be the one place it is written.
  - id: BR-10
    disposition: addressed
    note: |
      Verified by revert — dropping !opt.color from vocabularyFor reddens TestNoColourReadsNoDeck and TestStreamedAnswerCarriesNoEscapesWithoutColour.
  - id: BR-11
    disposition: addressed
    note: |
      nil is the single representation; no memVocabulary{} fallback remains in production and the unreachable replraw.go guard is gone.
  - id: BR-12
    disposition: addressed
    note: |
      Verified by revert — neutering highlightSetFor's parseCommandLine test reddens TestACommandLineIsNotHighlighted.
  - id: BR-16
    disposition: not-addressed
    note: |
      4th round. plan:270/:282 still name highlight_test.go, plan:286 still promises caller-unit counts, plan:70 still says ask.go:160, and Task 8 Steps 4 and 6 are ticked while undelivered (2 of 5 paths tabled; the Step 6 mutation reddens nothing).
  - id: BR-20
    disposition: not-addressed
    note: |
      3rd round for the vacuity guard — measured 6 of 32 corpus entries highlight, and swapping the deck for an unmatchable word leaves TestHighlightingLosesNothing green.
  - id: BR-27
    disposition: addressed
    note: |
      Counts deleted from render.go and atlas/define.md; lessons.md:1139's "ten-region" is the lesson quoting its own mistake, which is the record rather than a restatement.
  - id: BR-28
    disposition: addressed
    note: |
      Verified by revert — leaving base set on a reset reddens TestAnInnerResetClearsTheEnclosingStyle and TestHighlightAfterAnInnerResetDoesNotRestylePlainText.
  - id: BR-29
    disposition: addressed
    note: |
      fuzzDeck.MaxPhraseWords() measured at 3 with seeds; 1.12M execs clean this round.
findings:
  - id: new
    severity: Important
    family: behaviour-claimed-without-a-failing-test
    title: |
      M3 added a render surface and the entry-path enumeration written to prevent exactly this was not widened
    detail: |
      This is the 8th finding in family `behaviour-claimed-without-a-failing-test`. Do NOT
      fix only this instance. Measured at HEAD: replacing vocabularyFor(d, opt) in runAsk
      (ask.go:171) with an inline equivalent that keeps the nil/colour gate but drops
      d.vocab.Load() passes the ENTIRE suite. In production that mutant means a streamed
      answer never highlights on piped stdin or one-shot, since only runEditor loads —
      BR-14's shipped Critical, one surface over. It survives because every askhighlight
      test injects a pre-filled memVocabulary and so begins after the hop that fills it.
      THE RULE, widened: the enumeration's axis is entry path x RENDER SURFACE, not entry
      path alone. TestEveryEntryPathHighlightsDefinitions (vocab_test.go:242) covers 3 of
      6 cells while atlas/define.md:517 calls it the guard for "every render path". Each
      surface needs at least one row driven with the dependency in its real initial state;
      verified in both directions that swapping vocab(word) for an unloaded
      newStoreVocabulary passes on HEAD and kills the mutant.
  - id: new
    severity: Minor
    family: behaviour-claimed-without-a-failing-test
    title: |
      An assertion guarded on the run's own output never fires, and two comments claim what it does not pin
    detail: |
      This is the 9th finding in family `behaviour-claimed-without-a-failing-test`. Do NOT
      fix only this instance. askhighlight_test.go:143's only check sits behind
      `tc.cancel && got != ""`, and with an already-cancelled context runAsk returns before
      any delta, so out is empty (measured: code=0 out="" len=0) and the "interrupted
      mid-stream" row asserts nothing — while the test's own comment says the table pins
      that no path leaves text dangling. Same file, :163: "the capture's final characters,
      which only a flush can emit" is refuted by mutation — deleting `defer hw.Flush()`
      leaves that test green because the trailing Fprintln emits them. THE RULE the family
      had not yet named: an assertion guarded on the run's OWN output is not an assertion
      until something proves the guard fires. Measured prevalence: 4 output-conditional
      assertions in cmd/define; three guard on a fixture the author controls and are
      legitimate, this one is the only vacuous one. The package already owns the fix idiom
      at 5 sites (highlightwriter_test.go:328/433/483, invariant_test.go:79,
      dict_fake_test.go:69) — a t.Fatal when the observable is empty.
  - id: new
    severity: Minor
    family: copy-pasted-helper
    title: |
      func max in askhighlight_test.go shadows the Go builtin across the whole package's test build
    detail: |
      This is the 3rd finding in family `copy-pasted-helper`. Do NOT fix only this
      instance. askhighlight_test.go:66 redeclares max(a, b int); deleting it leaves
      `go vet ./cmd/define/` clean (verified), so it is pure shadowing on Go 1.26. Eight
      other call sites now resolve to it — invariant_test.go:43,95, live_property_test.go
      :51,81,101, render_test.go:235, editorloop_test.go:155 — while min beside them still
      resolves to the builtin, so the package's two halves of the same idiom now come from
      different places. THE RULE needs one word: the language's own builtins are part of
      what you grep before adding a helper.
  - id: new
    severity: Minor
    family: contract-error-unread-by-consumer
    title: |
      Every error highlightWriter is designed to report is discarded by its only production caller
    detail: |
      ask.go:171. `fmt.Fprint(out, delta)` ignores its error and `defer hw.Flush()`
      discards its return, while the writer poisons on first failure by contract
      (highlightwriter.go:120) — so one downstream failure silently drops the REST of an
      answer with no message, where before M3 it would have lost one delta. Practically
      unreachable with an os.File, but it is exactly the question M2 round 3 left as a note
      for Task 8 Step 4 ("what does the user see when the writer is already poisoned") and
      nothing in the window records a decision. Same defer, related: on the `default:` path
      the error line reaches errOut BEFORE the deferred flush pushes held text to out, so
      the last partial word lands after the error rather than before it.
  - id: new
    severity: Minor
    family: atlas-omits-a-shipped-rule
    title: |
      The atlas highlight section never mentions highlightSetFor, the command-namespace withhold
    detail: |
      highlight.go:173 withholds the vocabulary on a command line, so a deck word named
      like a command is not green inside "/history 7" — a real user-visible rule shipped at
      M1 round 3. atlas/define.md's "Highlighting the words you are learning" section
      documents every other rule in the feature and omits this one. New slug rather than
      atlas-claims-unbuilt-surface: that family names prose asserting surface that does not
      exist; this is the inverse, and the two need different slugs to match a recurrence.
```

---

## Re-review — 2026-08-26T18:07:12-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 21 — highlight the words you are learning wherever they appear |
| repo | tools |
| issue file | workshop/issues/000021-highlight-learned.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0207e7ef24414534b308a2296fb148b1740814ee..e1080f44e74fa2cb8164339fdb11140c1cdac7a4 |
| command | sdlc close --issue 21 |
| reviewer | claude |
| timestamp | 2026-08-26T18:07:12-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The headline fix of this round is real and I verified it by mutation rather than by reading the commit message: replacing `vocabularyFor(d, opt)` in `runAsk` with an inline gate that keeps the nil/colour check but drops `d.vocab.Load()` now reddens `TestEveryEntryPathHighlightsAnswers` on exactly the one-shot and piped-stdin rows, and correctly leaves the raw-editor row green — that is BR-30 closed at the class level, on the right axis. BR-31 and BR-32 are likewise genuinely closed, and I re-confirmed the in-file honest-scope claim (deleting `defer hw.Flush()` entirely reddens nothing) rather than taking the comment's word for it. Suite green, `-race` clean, `go vet`/`gofmt` clean, all four fuzz targets clean at ~600k execs each, tree clean, all eight Done-when rows delivered with a named pin, and the ARCH-PURPOSE shadow-sweep is exact — one `Vocabulary.Has` call site, three production `knownOn` sites, each deriving from the seam. What keeps this from SHIP is that **BR-33's fix ships unpinned**: reverting the reported-Flush-error to `defer hw.Flush()` leaves the *entire* suite green (full run, verified), so by the round's own rule the fix is not addressed — and I verified in both directions that a 12-line test using the package's existing `shortWriter` kills it. That, plus BR-16 at a fifth round and BR-20 at a fourth, and a sentence the close commit broke in the atlas.

## 1. Strengths

- **BR-30's fix is on the right axis and the mutant dies where production breaks.** `cmd/define/askhighlight_test.go:246` drives each entry path with `newStoreVocabulary(st, nil)` — deliberately unloaded — so the row begins *before* the hop that fills it. The Load-dropping mutant kills the one-shot and piped rows and spares the raw-editor row, which is exactly the production truth (`runEditor` loads at `replraw.go:79`).
- **The `wantEmpty` rewrite is a real assertion, and the surviving conditional is not vacuous.** I instrumented `TestEveryStreamExitPathFlushes` and measured which rows enter the word-arrival guard: `clean completion` → `wordArrived=true`, `truncated mid-answer` → `false`. So the guard fires, which is precisely what BR-31's rule demands of an output-conditional check.
- **The refuted comment was corrected in place rather than quietly dropped** (`askhighlight_test.go:180`). Deleting the defer leaves that test green and the comment now says so. Recording a coverage limit honestly is worth more than a test that implies coverage it lacks.
- **`splitWordInCapture` (`askhighlight_test.go:14`) tokenises with production `wordRuns`**, so the test cannot disagree with production about where a word is, and it `t.Fatalf`s rather than skipping when the capture stops splitting a word.
- **`Stall` → `JunkFrame` in the truncated row** is a good call — same `ErrTruncated` classification, without waiting out the client's 30s stall timeout.
- Core-concepts cross-check: all 15 rows exist at their stated paths (`span`, `highlightSpans`, `wordRuns`/`wordRun`, `sgrState`, `RenderLine`, `admitsHighlight`, `tokenStillOpen`, `Vocabulary`, `storeVocabulary`, `memVocabulary`, `highlightWriter`, `storeCapturer.Capture`, `deps.vocab`, `vocabularyFor`, `Render`). No table/code contradiction beyond the line-number drift below.

## 2. Critical findings

None.

## 3. Important findings

None newly raised. **BR-16 remains open at its fifth round** and is disposed `not-addressed` below — three concrete residues at HEAD, each measured.

## 4. Minor findings

- **`atlas/define.md:513-515` — the close commit's own edit left a broken sentence.** It reads `One function now owns "loaded, and only with colour", and` / `The enumeration that` / `guards it is **entry path × render surface**…` — a dangling conjunction followed by an orphaned capitalised clause, with the line break mid-phrase. The replaced text was well-formed; the replacement was not read back. New family `doc-edit-not-read-back`, because the two existing atlas families name *what the prose claims* (unbuilt surface / omitted rule), not a mechanically malformed edit.
- **`cmd/define/askhighlight_test.go:3-13` — a third-party import sits inside the stdlib group**, and `encoding/json`/`strings`/`testing` are split from the first stdlib block for no reason. Every other file in the package (e.g. `vocab_test.go:3`, `highlightwriter_test.go:3`) groups stdlib then third-party. **2nd in family `unformatted-source`** — so the fix is not this file. The rule: `gofmt -l` is clean here and always was, because nothing in the repo *runs* it. There is no formatting gate in `Makefile`, `Makefile.workflow`, `scripts/`, or CI (verified by grep); both instances were caught by a reviewer typing the command by hand. Put `gofmt -l` — and `goimports -l`, which is what catches this variant — in a gate.
- **The interrupt-with-held-text exit path lost its row when BR-31's fix renamed it.** **This is the 10th finding in family `behaviour-claimed-without-a-failing-test`.** Do NOT fix this instance. The row was `interrupted mid-stream` and became `cancelled before any delta` — honest, and strictly weaker: an already-cancelled context returns before any delta, so held text and cancellation never interact. The path Task 8 Step 4 names by name (`askScoped` cancels mid-stream, `ctx.Err() != nil && answer.Len() > 0`, which writes `Fprintln(out)` through the held writer) is now exercised only at `options{}` — colour off, `vocabularyFor` nil, nothing held — by `TestAQuestionIsRecordedWhateverBecameOfTheAnswer/cut off mid-answer` (`askrun_test.go:590`). I drove it with `color: true` and `Stall: true`: correct (43 bytes, ends `\n`, highlight intact), so this is coverage, not a bug — and it is ~20 lines using `syncBuf` and `Stall`, both already in the package. **THE RULE the family had not yet named: a fix that makes a test *honest* can also make it *narrower*, and the narrowing is silent because every remaining assertion still passes.** When a row is renamed or a guard is tightened, state what it stopped covering. Measured prevalence in this window: 1 of 3 rewritten rows lost a case.
- **`cmd/define/vocab.go:76` — `MaxPhraseWords` is inflated by deck keys that can never match, and M3 turned that from a curiosity into streaming latency.** `phraseGap` admits only spaces and tabs, so a key with internal punctuation is structurally unmatchable — which `TestAPunctuatedKeyIsNotMatchable` pins — yet `Add` still counts its tokens into the look-ahead budget the stream holds against. Measured on `"the quick brown fox jumps"`: deck of single words holds 5 bytes; adding the unmatchable `e.g.` holds 9; adding the unmatchable `rock 'n' roll` holds 15. The comment at `vocab.go:76` says such a key "is currently UNMATCHABLE whichever way it is counted" — true at M1, false since M3, because the count is now what the reader waits on. Fix: bump `maxWords` only for keys whose own tokens rejoin.

## 5. Test coverage notes

- Mutations this round (full suite each, in a scratch export; the four `repo_guard` failures are `.git`-absence noise present in the unmutated control): **3 killed** — Load dropped on the answer path (2 rows), plus the standing suite; **2 survived by design and are documented as such** — `defer hw.Flush()` deleted entirely (conceded in-file); **1 survived undocumented** — the BR-33 error-reporting branch (below).
- The proposed BR-33 pin, verified both ways: with `runAsk(..., &shortWriter{limit: 2}, &errOut)` and an assertion on `"could not be fully written"`, the test passes on HEAD and fails with `errOut=""` when the fix is reverted. `shortWriter` already exists in `crlf_test.go`.
- `TestHighlightingLosesNothing` measured: 6 of 32 corpus entries highlight; swapping its deck for `vocab("zzzznotinanycorpusentry")` leaves every assertion green. Fourth round for that guard.
- Fuzz sanity at HEAD: `FuzzHighlightWriterIsChunkIndependent` 617k, `FuzzHighlightSpans` 684k, `FuzzWordRuns` 685k, `FuzzTrailingSegments` 464k — all clean, corpus additions in GOCACHE, tree unchanged. `go test -race ./cmd/define/` green (65.8s); `go test ./...` green.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag (BR-9, disposed below).** Two of three warning seams wrap `warnTo`; `capture.go:132` still writes `"define: "+format+"\n"` itself while `warnTo`'s own doc comment (`vocab.go:130`) claims to be "the one place" it is written. Otherwise the package is in good shape: one tokenizer, one matcher, one writer for both styled surfaces, `phraseGapOrEmpty` delegating to `phraseGap`, `stripANSI` deferring to `scanEscape`.
- **ARCH-PURE — pass.** `wordRuns`, `highlightSpans`, `sgrState`, `scanEscape`, `decidedEnd`, `tokenStillOpen` are pure and tested with no IO. `runAsk` stays the thin shell: the writer is built at the boundary and injected as an `io.Writer`, and `answer` accumulates raw deltas, so no escape reaches `recordExchange` → `gatherAskContext` → the next prompt. I confirmed `onDelta` is invoked synchronously on the caller's goroutine (`internal/llm/anthropic.go:169-188`), so the unguarded `highlightWriter` state is not racy — and `-race` agrees.
- **ARCH-PURPOSE — pass.** Shadow-sweep is exact: `Vocabulary.Has` has one call site (`highlight.go:132`); `knownOn` has three production sites (`editor.go:208`, `render.go:93`, `ask.go:171`), each reaching the seam. No hand-maintained restatement of the set anywhere. Done-when row 8 is satisfied, not asserted.
- **ARCH-MOCK — pass.** No new external dependency in M3. The answer arrives through `llmtest.Fake` replaying a committed capture, `store.NewMem()` is the portable non-production backend, and `crlfWriter`/`shortWriter`/`failAfter` sit at the same `io.Writer` boundary production uses. Live conformance exists for the surfaces this depends on (`internal/llm/capture_conformance_test.go`, `conformance_test.go`, `cmd/define/pty_conformance_test.go`).
- **Seam-ownership note for the next capture consumer** (not a finding): `splitWordInCapture` decodes `llmtest`'s SSE artifact with a parser living in `cmd/define`. It is currently the only full decoder — `anthropic_test.go:259` does a weaker `strings.Count` — so this is layering, not duplication, and its drift mode is a loud `t.Fatalf`. If a second consumer needs delta boundaries, the parser belongs behind the seam that owns the format (`llmtest.CaptureDeltas`) before it gets copied.
- **For #22:** `memVocabulary` still has no removal path. Narrowing to actively-learned words means the set must *shrink* mid-session as a word graduates, and nothing above the seam is prepared for a `Has` that flips true→false while the writer is mid-hold. Decide that at #22's plan gate, not in its writer.

## 7. Plan revision recommendations

One `## Revisions` entry — "close boundary round 2: the plan-record residues, finished" — covering exactly what remains, all measured at HEAD:

- `plan:70` — Core concepts still gives the answer-stream injection point as `ask.go:160`; actual `ask.go:171`.
- `plan:286` — Task 6 Step 2 still requires "that the returned count is in the caller's units", which contract rule 4 (`plan:100-110`) and `TestHighlightWriterPropagatesDownstreamErrors` both contradict with `(0, err)`. Delete the clause and point the step at rule 4. (Task 5/6/7/8 `Files` blocks *were* corrected this round — that half is done.)
- **Task 8 Step 6 is ticked and its stated mutation check fails.** "Mutation-check that dropping the flush on the interrupt path reddens a named test": measured, deleting `defer hw.Flush()` reddens nothing. The test file says so honestly; the plan tick says the opposite. Untick or record the deviation.
- **Task 8 Step 4 is ticked at 3 of 5 paths** (up from 2). The in-file comment correctly argues the `default:` path is unreachable through `llmtest`; the *interrupt-with-held-text* path is reachable and was dropped. Add the row or record what the tick covers.
- Record BR-33's decision as delivered *and unpinned*, with the `shortWriter` test as the outstanding step — a decision documented in Revisions but absent from the atlas's own `Flush` paragraph is the same shape as BR-34.

```findings
dispose:
  - id: BR-9
    disposition: not-addressed
    note: |
      Unchanged since round 8 — capture.go:132 still writes "define: " itself while warnTo's comment (vocab.go:130) claims to be the one place it is written.
  - id: BR-16
    disposition: not-addressed
    note: |
      5th round. Files blocks fixed; plan:70 still says ask.go:160 (actual 171), plan:286 still promises caller-unit counts, and Task 8 Step 6's mutation claim is false — deleting the defer reddens nothing.
  - id: BR-20
    disposition: not-addressed
    note: |
      4th round for the vacuity guard — measured 6 of 32 corpus entries highlight, and swapping the deck for an unmatchable word leaves TestHighlightingLosesNothing green.
  - id: BR-30
    disposition: addressed
    note: |
      Verified by mutation — an inline gate keeping nil/colour but dropping Load reddens TestEveryEntryPathHighlightsAnswers on exactly the one-shot and piped-stdin rows.
  - id: BR-31
    disposition: addressed
    note: |
      wantEmpty is asserted unconditionally with a t.Fatal on empty output; probed that the surviving output-conditional guard fires on the clean row; the refuted comment is corrected in place.
  - id: BR-32
    disposition: addressed
    note: |
      No func max/min remains in cmd/define; max(0, len(deltas)-1) resolves to the Go 1.26 builtin; vet, build and suite clean.
  - id: BR-33
    disposition: not-addressed
    note: |
      The code change is right but nothing pins it — reverting to `defer hw.Flush()` leaves the ENTIRE suite green (full run). A 12-line test with the package's existing shortWriter kills the mutant, verified both ways. The errOut-before-flush ordering half is also unchanged and unrecorded.
  - id: BR-34
    disposition: not-addressed
    note: |
      atlas/define.md:422-545 still never mentions highlightSetFor or the command-namespace withhold; the close commit added a second omission in the same family by recording the poisoned-writer decision only in the plan's Revisions, not in the atlas Flush paragraph where it belongs.
findings:
  - id: new
    severity: Minor
    family: doc-edit-not-read-back
    title: |
      The atlas sentence the close commit itself edited is malformed prose
    detail: |
      atlas/define.md:513-515 now reads `One function now owns "loaded, and only with
      colour", and` / `The enumeration that` / `guards it is **entry path x render
      surface**...` — a dangling conjunction followed by an orphaned capitalised clause,
      broken mid-phrase across lines. The text it replaced was well-formed; the
      replacement was written but not read back. New slug rather than
      atlas-claims-unbuilt-surface or atlas-omits-a-shipped-rule: those two name what the
      prose CLAIMS (surface that does not exist, a rule left out); this is a mechanically
      malformed edit, and would recur in any doc file with any content.
  - id: new
    severity: Minor
    family: unformatted-source
    title: |
      A third-party import sits inside the stdlib group, and nothing in the repo enforces import grouping
    detail: |
      This is the 2nd finding in family `unformatted-source`. Do NOT fix only this
      instance. askhighlight_test.go:3-13 puts github.com/xianxu/tools/cmd/define/store
      between encoding/json and strings, and splits the stdlib block for no reason; every
      other file in the package groups stdlib then third-party (vocab_test.go:3,
      highlightwriter_test.go:3). gofmt does not catch this and goimports would. THE RULE:
      both instances of this family were caught by a reviewer typing the command by hand —
      verified that no Makefile, Makefile.workflow, scripts/ or CI target runs gofmt or
      goimports anywhere in the repo. Put `gofmt -l` and `goimports -l` in a gate; fixing
      the file again leaves the next instance to the next reviewer.
  - id: new
    severity: Minor
    family: behaviour-claimed-without-a-failing-test
    title: |
      The interrupt-with-held-text exit path lost its row when BR-31's fix renamed it, and nothing noticed
    detail: |
      This is the 10th finding in family `behaviour-claimed-without-a-failing-test`. Do NOT
      fix only this instance. The row `interrupted mid-stream` became `cancelled before any
      delta` — honest, and strictly weaker: an already-cancelled context returns before any
      delta, so held text and cancellation never interact. The path Task 8 Step 4 names by
      name (askScoped cancels mid-stream; ctx.Err() != nil with answer.Len() > 0, writing
      Fprintln through the held writer) is now exercised only at options{} — colour off,
      vocabularyFor nil, nothing held — by askrun_test.go:590. Measured with color:true and
      Stall:true it is CORRECT (43 bytes, ends in a newline, highlight intact), so this is
      coverage rather than a bug, and it is ~20 lines using syncBuf and Stall, both already
      in the package. THE RULE the family had not yet named: a fix that makes a test HONEST
      can also make it NARROWER, and the narrowing is silent because every remaining
      assertion still passes. When a row is renamed or a guard tightened, state what it
      stopped covering. Measured prevalence this window: 1 of 3 rewritten rows lost a case.
  - id: new
    severity: Minor
    family: bound-includes-unusable-input
    title: |
      MaxPhraseWords is inflated by deck keys that can never match, and M3 made that streaming latency
    detail: |
      vocab.go:76. phraseGap admits only spaces and tabs, so a key with internal
      punctuation is structurally unmatchable — TestAPunctuatedKeyIsNotMatchable pins
      exactly that — yet Add still counts its tokens into the look-ahead budget
      decidedEnd holds against. Measured on "the quick brown fox jumps": single-word deck
      holds 5 bytes, adding the unmatchable `e.g.` holds 9, adding the unmatchable
      `rock 'n' roll` holds 15, so the reader waits on words for a match that cannot
      occur. The comment at vocab.go:76 says such a key "is currently UNMATCHABLE
      whichever way it is counted" — true at M1, false since M3 wired the stream, and
      nothing revisited it. THE RULE: a bound derived from an input set must be derived
      from the subset that can actually exercise it — bump maxWords only for keys whose
      own tokens rejoin.
```

---

## Re-review — 2026-08-26T18:34:03-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 21 — highlight the words you are learning wherever they appear |
| repo | tools |
| issue file | workshop/issues/000021-highlight-learned.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0207e7ef24414534b308a2296fb148b1740814ee..6300136c98a7c976996ea5ab2ed170aae4cfcc0c |
| command | sdlc close --issue 21 |
| reviewer | claude |
| timestamp | 2026-08-26T18:34:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The feature is done and I have strong independent evidence it is correct — not from the commit messages, but from an adversarial differential I wrote from scratch: every one of the 32 real corpus entries rendered with colour ON, deck derived from that entry's own 1–3-token windows, fed through the writer **byte-at-a-time** and compared against one call — 0 mismatches, no visible text changed. On top of that: four fuzz targets clean at ~0.9–1.6M execs each, `go test -race ./cmd/define/` green, full suite green, `go vet`/`gofmt`/build clean, tree clean, all 8 Done-when rows and all plan steps ticked. Two of the nine open findings are genuinely closed this round and I verified both by mutation (BR-20's vacuity guard now fires; BR-16's three *named* residues are gone). What keeps this from SHIP is that **the other seven open findings were not touched at all** — the round's single commit changed one test file and four docs — and one of them, BR-33, is a correct fix shipping unpinned for the third consecutive round: I reverted it to `defer hw.Flush()` and the *entire* suite stayed green, then confirmed in both directions that a 12-line test using the package's existing `shortWriter` kills that mutant. Nothing here is a correctness bug; all of it is pinning and record-keeping.

## 1. Strengths

- **BR-20's guard is real and it bites.** `cmd/define/highlightwriter_test.go:350,357` — swapping the deck for `vocab("zzzznotinanycorpusentry")` now fails with *"no entry in the corpus highlighted anything"*. Measured at HEAD: 6 of 32 entries highlight, logged. That closes a finding that was open four rounds, and the `t.Logf` makes a future corpus refresh visible rather than silent.
- **The writer holds up under a corpus differential it was never tuned for.** `fuzzDeck` is hand-curated (class × position × phrase length); my probe replaced it with decks derived from each entry's own text — every phrase the corpus actually contains — and split at every byte. Clean across all 32. That is independent confirmation that BR-13/BR-23's release rules are right, not just that the shipped fixtures agree with them.
- **`Task 8 Step 6` was corrected instead of ticked.** The plan step claimed dropping the flush reddens a named test; measurement says it reddens nothing, and the step now says so at length. A ticked mutation claim *is* an assertion, and correcting one against measurement is exactly the discipline this issue's ledger has been asking for.
- **The ARCH-PURPOSE shadow-sweep is exact.** One `Vocabulary.Has` call site in the whole tree (`highlight.go:132`); three production `knownOn` consumers (`editor.go:208`, `render.go:93`, `ask.go:171`); one `Load()` site, inside `vocabularyFor` (`vocab.go:160`). No hand-maintained restatement of the set anywhere — #22's swap really is one constructor.
- **`Render`'s `prose(s, base)` wiring is correct at every emit site**, which I checked rather than assumed: the gloss follows `marker`'s `p.off` (plain → `base: ""`), the example sits inside `p.ex` (→ `base: p.ex`), section text follows the section name's `p.off` (plain → `""`). The BR-24-class "wrong base" bug has no second instance.

## 2. Critical findings

None.

## 3. Important findings

None newly raised. **BR-16 remains open at its sixth round** and is disposed `not-addressed` below, with three live residues measured at HEAD.

## 4. Minor findings

**The `Vocabulary` seam advertises concurrency safety it does not have, and a concurrency claim is the one kind of behaviour claim nothing in this package can falsify** — `cmd/define/vocab.go:46-47`, `:100`, `:112-115`.

> **This is the 11th finding in family `behaviour-claimed-without-a-failing-test`.** Do NOT fix only this instance.

`memVocabulary`'s doc comment states a premise and a conclusion, and measurement contradicts both:

- *Premise* — "the two accesses are **genuinely concurrent**: the capture path Adds a word as a lookup completes while the editor's render reads." Not true today. All three production goroutines (`rawterm.go:55` key reader, `repl.go:196` signal fan-out, `repl.go:362` `scanLines`) carry values over channels and touch no vocabulary; every `Add`/`Has` runs on the loop goroutine.
- *Conclusion* — that the type is therefore safe. Also not true. `storeVocabulary` embeds `memVocabulary` and adds an **unguarded** `loaded bool`, and `vocabularyFor` calls `Load()` on *every* render. Driving `Load`/`Add`/`Has` from 8 goroutines under `-race` reports a data race immediately: read at `vocab.go:112`, write at `vocab.go:115`. The suite's `-race` run is clean only because the premise is false.

So the comment is a safety licence that would be acted on and then fail. **THE RULE, one notch wider than the family has yet stated it:** round 6 established that the falsifying observation is *production output bytes* for a wiring claim and *a counting/spying double* for a guard whose effect is absence. This adds the third kind — **a concurrency claim's falsifier is a `-race` driver**, and this package owns none, so every such claim in it is unpinned by construction. The enumeration to sweep is the synchronisation claims in `cmd/define`: `memVocabulary.mu`, `storeVocabulary.loaded`, `storeHistory.mu`, `storeCapturer.mu`/`warned`. Either guard `loaded` (`sync.Once`, or the existing mutex) and add one `-race` driver that would fail without it, or state the real invariant — "single-goroutine today; the mutex is defensive" — so the comment stops licensing something the type cannot support.

## 5. Test coverage notes

- **Mutations run this round** (full suite each, in a scratch export of HEAD; the four `repo_guard` failures are `.git`-absence noise present in the unmutated control): **3 killed** — unmatchable deck → `TestHighlightingLosesNothing`; `highlightRegion` drops a byte → same test (so plan Task 7 Step 6's ticked mutation claim is *true*, unlike Step 6 of Task 8); `-no-color` prints no answer → 7 tests. **1 survived** — BR-33's `defer hw.Flush()` revert, full suite green.
- **One absence assertion in the window has no liveness guard, and its own sibling does.** `TestStreamedAnswerCarriesNoEscapesWithoutColour` (`askhighlight_test.go:199`) asserts only the *absence* of `\x1b`, so an empty `out` passes it. `TestDefinitionHighlightingIsOffWithoutColour` (`highlightwriter_test.go:314-319`), written in the previous milestone, pairs the same absence check with `if !strings.Contains(out.String(), "obsequious")`. Measured prevalence: 1 of 5 absence-of-escape assertions in `cmd/define` lacks a liveness guard. I am **not** raising this as a finding — I mutated production so `-no-color` prints nothing at all, and 7 other tests caught it, so the behaviour is covered and this is hygiene, not a hole. Worth one line when the file is next touched.
- Fuzz at HEAD: `FuzzHighlightWriterIsChunkIndependent` 950k, `FuzzHighlightSpans` 1.12M, `FuzzWordRuns` 942k, `FuzzTrailingSegments` 1.57M — all clean; corpus additions land in GOCACHE, tree unchanged.
- `TestEveryStreamExitPathFlushes` still names five paths in prose and tables three (BR-37).

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag (BR-9 only).** `capture.go:132` writes `"define: "+format+"\n"` itself while `warnTo`'s doc comment (`vocab.go:130`) says it is "the one place the prefix and the trailing newline are written". Two of three seams wrap it; the comment is the finding as much as the literal is. Everything else consolidates cleanly: one tokenizer, one matcher, one writer for both styled surfaces, `phraseGapOrEmpty` delegating to `phraseGap`, `stripANSI` deferring to `scanEscape`, `storeVocabulary` embedding `memVocabulary`, `wordsOf` in the leak test calling production `wordRuns`.
- **ARCH-PURE — pass.** `wordRuns`, `highlightSpans`, `sgrState`, `scanEscape`, `decidedEnd`, `tokenStillOpen`, `trailingSegments`, `matchesFor`, `historyCompletions` are pure and unit-tested with no IO. `Render` takes `Vocabulary` as injected data; the single IO decision (load + colour) is isolated in `vocabularyFor` and pinned by a counting double. `runAsk` stays the thin shell — the writer is built at the boundary and injected as an `io.Writer`, and `answer` accumulates raw deltas, so no escape reaches `recordExchange` → `gatherAskContext` → the next prompt.
- **ARCH-PURPOSE — pass on the feature, flag on the record.** Shadow-sweep exact (§1). All three Spec surfaces highlight through one predicate on all three entry paths, verified end-to-end with an *unloaded* set. The flag is the disposition pattern: seven of nine open findings were not worked at all this round, and BR-33 answers the finding's *site* (the code is right) without its *class* (nothing pins it) — three rounds running.
- **ARCH-MOCK — pass.** No new external dependency. The answer arrives through `llmtest.Fake` replaying a committed capture (and `splitWordInCapture` refuses to invent streamed text, which the fake genuinely cannot serve); `store.NewMem()` is the portable non-production backend; `crlfWriter`/`shortWriter`/`failAfter` sit at the same `io.Writer` boundary production uses. Live conformance suites for the dependencies (`internal/llm/capture_conformance_test.go`, `cmd/define/pty_conformance_test.go`) are untouched and still apply.
- **For #22:** `memVocabulary` still has no removal path — `Add` only. Narrowing to actively-learned words means the set must *shrink* mid-session as a word graduates, and nothing above the seam is prepared for a `Has` that flips true→false while the writer is mid-hold. Decide that at #22's plan gate, not in its writer. BR-38 is the other thing to settle first: `MaxPhraseWords` is the streaming hold budget now, and #22 changing what is in the set changes that budget.

## 7. Plan revision recommendations

One `## Revisions` entry — "close boundary round 3: the plan-claim sweep, finished by its own rule":

- **`plan:43` and `plan:241`** both place `RenderLine`'s tests in `cmd/define/editor_test.go`; they are in `cmd/define/highlight_test.go`. This is the same Files-block class BR-16 named and round 7 fixed for Tasks 5/6/7/8 — Task 4 was skipped, and it is the *only* Task whose `Test:` line is still wrong.
- **`plan:243` (Task 4 Step 1, ticked)** instructs "assert the output contains `knownOn + "obsequious"`". `workshop/lessons.md` — extended in this very commit — names that exact shape as the trap ("aliasing `knownOn = inputOn` … leaves it green"), and the shipped test correctly writes the literal `"\x1b[1;32mobsequious"`. A ticked step whose stated method the issue's own lesson forbids is the third kind of rotting claim, alongside line numbers and twice-stated contracts.
- **`plan:91` and `plan:330`** still carry code line numbers (`store/word.go:29-31`, `internal/llm/llmtest/fake.go:188-215`, `:461-472`) against the rule this commit wrote down — "cite the file and the symbol; drop the number". I verified all three are accurate *today*; they are cross-package references this issue does not own, so they are the ones most likely to rot first.
- Record BR-33's decision as **delivered and unpinned**, with the `shortWriter` test as the outstanding step, and put the decision in the atlas's own `Flush` paragraph rather than only in the plan — a decision recorded where the reader of the mechanism will not find it is the same shape as BR-34.

```findings
dispose:
  - id: BR-9
    disposition: not-addressed
    note: |
      Unchanged for a third round — capture.go:132 still writes "define: " itself while warnTo's comment (vocab.go:130) claims to be the one place it is written.
  - id: BR-16
    disposition: not-addressed
    note: |
      6th round. The three named residues ARE fixed; three live ones remain by the commit's own stated rule — plan:43/:241 place RenderLine's tests in editor_test.go (they are in highlight_test.go), plan:243 instructs the knownOn+"text" assertion shape that this commit's own lessons.md entry forbids, and plan:91/:330 still carry code line numbers.
  - id: BR-20
    disposition: addressed
    note: |
      Verified by mutation — an unmatchable deck now reddens TestHighlightingLosesNothing at the hits==0 guard; measured 6 of 32 entries highlight and the count is logged.
  - id: BR-33
    disposition: not-addressed
    note: |
      3rd round. Reverting to `defer hw.Flush()` leaves the ENTIRE suite green (full run at HEAD). Verified both ways that a 12-line test driving runAsk into the package's existing shortWriter and asserting "could not be fully written" passes on HEAD and fails on the revert. The errOut-before-flush ordering half is also still unchanged and unrecorded.
  - id: BR-34
    disposition: not-addressed
    note: |
      Unchanged — atlas/define.md's highlight section never mentions highlightSetFor or the command-namespace withhold, and the Flush paragraph still describes the defer without the stderr report the code now emits.
  - id: BR-35
    disposition: not-addressed
    note: |
      Unchanged — atlas/define.md:513-515 still reads `...and` / `The enumeration that` / `guards it is ...`, a dangling conjunction followed by an orphaned capitalised clause broken mid-phrase.
  - id: BR-36
    disposition: not-addressed
    note: |
      Unchanged — askhighlight_test.go:3-13 still puts the store import between encoding/json and strings, and re-verified that no Makefile, Makefile.workflow, scripts/ or .github/workflows target runs gofmt or goimports anywhere in the repo.
  - id: BR-37
    disposition: not-addressed
    note: |
      Unchanged — the exit-path table is still {clean, cancelled-before-any-delta, truncated}. Re-measured the missing path with color:true and Stall:true through the real wiring: 43 bytes, ends in a newline, highlight intact, so it is coverage rather than a bug.
  - id: BR-38
    disposition: not-addressed
    note: |
      Unchanged — reproduced the measurement at HEAD on "the quick brown fox jumps": single-word deck holds 5 bytes, adding the unmatchable `e.g.` holds 9, adding the unmatchable `rock 'n' roll` holds 15, identical to a REAL three-token phrase.
findings:
  - id: new
    severity: Minor
    family: behaviour-claimed-without-a-failing-test
    title: |
      The Vocabulary seam advertises concurrency safety it does not have, and nothing in the package can falsify a concurrency claim
    detail: |
      This is the 11th finding in family `behaviour-claimed-without-a-failing-test`. Do NOT
      fix only this instance. memVocabulary's doc comment (vocab.go:46-47) states a premise
      and a conclusion and measurement contradicts both. Premise: "the two accesses are
      genuinely concurrent" — false today; the three production goroutines (rawterm.go:55,
      repl.go:196, repl.go:362) carry values over channels and touch no vocabulary, so every
      Add/Has runs on the loop goroutine. Conclusion, that the type is therefore safe —
      also false: storeVocabulary embeds it and adds an UNGUARDED `loaded bool`
      (vocab.go:100), while vocabularyFor calls Load() on every render. Driving
      Load/Add/Has from 8 goroutines under -race reports a data race immediately: read at
      vocab.go:112, write at vocab.go:115. The suite's -race run is clean only because the
      premise is false, so the comment is a safety licence that fails the moment anyone
      acts on it. THE RULE, one notch wider than the family has stated it: round 6
      established the falsifier is production output bytes for a wiring claim and a
      counting/spying double for a guard whose effect is absence. A CONCURRENCY claim's
      falsifier is a -race driver, and this package owns none, so every such claim in it is
      unpinned by construction. The enumeration to sweep is the synchronisation claims in
      cmd/define: memVocabulary.mu, storeVocabulary.loaded, storeHistory.mu,
      storeCapturer.mu/warned. Either guard `loaded` and add one -race driver that fails
      without it, or state the real invariant so the comment stops licensing what the type
      cannot support.
```

---

## Re-review — 2026-08-26T18:58:22-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 21 — highlight the words you are learning wherever they appear |
| repo | tools |
| issue file | workshop/issues/000021-highlight-learned.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0207e7ef24414534b308a2296fb148b1740814ee..16c6e1c5f885691bfb00e53f09660fdc8887349d |
| command | sdlc close --issue 21 |
| reviewer | claude |
| timestamp | 2026-08-26T18:58:22-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both real defects this round are genuinely fixed and I verified each by reverting it, not by reading the commit message: reverting the `loaded` guard produces a data race at `vocab.go:130/133` and reddens `TestVocabularyIsSafeUnderConcurrency` under `-race`; reverting the `phraseRunsJoin` condition reddens `TestAnUnmatchableKeyDoesNotWidenTheHoldWindow` on both assertions; reverting the poison report — and separately deleting the whole defer — reddens `TestAPoisonedWriterIsReported`. Build/`gofmt`/`vet` clean, `go test ./...` green, `go test -race ./cmd/define/` green, four fuzz targets clean at 2.24M/1.55M/2.43M/4.66M execs, tree clean, Core-concepts table matches the filesystem at all 15 rows, and the ARCH-PURPOSE shadow-sweep is exact (one `Vocabulary.Has` call site, three `knownOn` production consumers, one `Load()` site). What keeps this from SHIP is three measurements, all made by running code: **BR-39's own enumeration was not swept** — `storeHistory.Load` has the identical unguarded-`loaded` shape and reports a data race under exactly the driver the finding named as the falsifier; **the commit that swept BR-16's claim-rot created a fresh instance of it** — `TestAPoisonedWriterIsReported` makes plan Task 8 Step 6 and `atlas/define.md:533` (“deleting the defer leaves the suite green”) measurably false, and neither was re-measured; and **the row added to close BR-37 covers nothing** — instrumented, `stalled upstream` lands on `case err == nil:`, the same branch as the `clean completion` row above it, for 30.01s of wall clock. None of this is a correctness bug in shipped behaviour.

## 1. Strengths

- **The BR-38 fix reuses the matcher's own predicate rather than inventing a punctuation test.** `vocab.go:91` guards the `maxWords` bump with `phraseRunsJoin(key, runs)` — the same function `highlightSpans` uses at `highlight.go:128` — so "can these tokens rejoin" has one implementation and the bound cannot drift from the matching rule (ARCH-DRY). That is what rejects `rock 'n' roll`, which a punctuation check would have admitted; `TestAnUnmatchableKeyDoesNotWidenTheHoldWindow` pins that case explicitly and it reddens on revert.
- **The `-race` driver is the right falsifier and it works.** Verified in both directions: with the guard reverted, `-race` reports `Read at vocab.go:130 / Write at vocab.go:133` and the test fails; without `-race` the same mutant passes 5/5 runs. That asymmetry is exactly the point the finding made, and the fix is pinned by the only observation that can pin it.
- **The phrase-length fuzz axis survived the bound change.** `fuzzDeck.MaxPhraseWords()` still measures 3 at HEAD — `in spite of` rejoins, so the BR-29 axis was not silently collapsed by narrowing the bound. Worth stating because a fix to a `max` is exactly where a fixture axis quietly dies.
- **`TestAPoisonedWriterIsReported` reuses `crlf_test.go`'s `shortWriter`** instead of writing a fourth failing-writer double, and drives `runAsk` end to end rather than the writer directly — so it crosses the wiring hop, which is the class this issue kept being bitten by.
- **The atlas gained the `highlightSetFor` paragraph and the malformed sentence was repaired** — both BR-34 and BR-35 closed cleanly.

## 2. Critical findings

None.

## 3. Important findings

**BR-39's enumeration was not swept: `storeHistory.Load` races in the same shape, in a seam the finding named by name — `cmd/define/history_store.go:39-43`.**

> **This is the 12th finding in family `behaviour-claimed-without-a-failing-test`.** Do NOT fix only this instance — this *is* the instance-vs-class shape, one round after the finding wrote the class out longhand.

BR-39 said, verbatim: *"The enumeration to sweep is the synchronisation claims in cmd/define: `memVocabulary.mu`, `storeVocabulary.loaded`, `storeHistory.mu`, `storeCapturer.mu/warned`."* One of the four was fixed. Measured at HEAD with an 8-goroutine driver:

```
WARNING: DATA RACE
Read at  ... history_store.go:40   (if h.loaded || h.st == nil)
Previous write at ... history_store.go:43   (h.loaded = true)
```

`h.lines = append(h.lines, e.Word)` at `:50` is likewise outside the mutex that `Add` (`:66`) and `Prefix` (`:72`) both take — the identical inconsistency that made `storeVocabulary` a race. There is no live race today, for the same reason `storeVocabulary` had none: every caller is on the loop goroutine. That is precisely the premise BR-39 established is load-bearing and undocumented. `storeCapturer` I checked and it is fine — `mu` covers `warned`, and everything `Capture` reads is immutable after construction. So the sweep is one seam, and the driver that proves it is 12 lines of the one just written.

## 4. Minor findings

- **The row added to close BR-37 enters no branch the table did not already cover, and costs 30 seconds — `cmd/define/askhighlight_test.go:128-135`.** Instrumented every exit of `runAsk` and ran the table: `clean completion` → `err=nil ctxErr=nil answerLen=532` → `case err == nil:`; `stalled upstream` → `err=nil ctxErr=nil answerLen=31` → **`case err == nil:`**. Same branch, one row apart. The row's comment calls it *"the fifth cell … the one remaining exit path"*, and the comment eight lines above it says Stall and JunkFrame are *"Same path"* and rejects Stall as *"two orders of magnitude"* more expensive — then this row pays that cost anyway: `TestEveryStreamExitPathFlushes` takes 30.02s, of which `stalled_upstream` is 30.01s, and `go test ./cmd/define/` is now 93.6s against the 63.6s recorded at #20's close. **THE RULE, which this family had not stated for tests:** a row added to close a coverage finding is a claim about which branch it reaches, and its falsifier is instrumentation, not the fake's documentation — measure the branch before writing "the one remaining exit path". Measured prevalence this window: 1 of 1 rows added to answer a coverage finding entered a branch already covered.
- **`atlas/define.md:533` and plan Task 8 Step 6 (`plan:340`) both assert a mutation result that the same commit falsified.** Both say deleting the deferred `Flush` reddens nothing; measured, deleting it reddens `TestAPoisonedWriterIsReported` — the test that commit added. The in-file comment at `askhighlight_test.go:104` is still correct because it is scoped to *"this table"*. This is BR-16's third kind of rotting claim, created by the commit that swept for the other two.
- **`memVocabulary`'s doc comment still licenses more than the type delivers — `cmd/define/vocab.go:50`.** *"The lock is here so the type stays safe if that changes"* is true only in the data-race sense. `Load` sets `loaded` and releases the mutex before reading the deck, so it is not a barrier: driving two goroutines through `Load` then `Has`, **97 of 200 trials** had a goroutine whose `Load()` returned before the deck was readable, and it would render against an empty set. Race-free ≠ correct under concurrency, and this matters for #22, which makes the set change mid-session.
- `cmd/define/capture.go:132` still writes `"define: "+format+"\n"` itself while `warnTo`'s doc comment at `vocab.go:151` claims to be *"the one place"* it is written (BR-9, fourth round). The comment is as much the defect as the literal.
- `cmd/define/askhighlight_test.go:3-13` still puts the `store` import between `encoding/json` and `strings` (BR-36). Re-verified that no `Makefile`, `Makefile.workflow`, `Makefile.local`, `scripts/` or `.github/` target runs `gofmt`, `goimports`, `go vet`, `go test` **or `-race`** anywhere in the repo — which now also means BR-39's fix is only falsifiable if the next person remembers to type `-race`.

## 5. Test coverage notes

- Mutations run this round (full `./cmd/define/` suite each, in a `git archive` scratch export; the four `repo_guard` failures are `.git`-absence noise present in the unmutated control): **3 killed** — `defer hw.Flush()` revert and full-defer deletion both → `TestAPoisonedWriterIsReported`; `phraseRunsJoin` guard removed → `TestAnUnmatchableKeyDoesNotWidenTheHoldWindow` (both assertions, `MaxPhraseWords` 2 and 3 where 1 is wanted); `loaded` unguarded → `TestVocabularyIsSafeUnderConcurrency` under `-race`. **0 survived** among the fixes claimed this round.
- The interrupt-with-held-text path (`ctx.Err() != nil` with `answer.Len() > 0`) is still exercised only at `color=false vocabNil=true` — measured `PROBE-EXIT ctxErr answerLen=31 color=false vocabNil=true` from `askrun_test.go`'s `cut off mid-answer` row, and nowhere with a held writer.
- `go test -race ./cmd/define/` green (98.0s). Fuzz clean: `FuzzHighlightWriterIsChunkIndependent` 2.24M, `FuzzHighlightSpans` 1.55M, `FuzzWordRuns` 2.43M, `FuzzTrailingSegments` 4.66M; corpus additions land in GOCACHE and the tree is unchanged.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag (BR-9 only).** The `warnTo` residue is the single live duplication. Everything else consolidates well, and this round improved it: `phraseRunsJoin` now serves both the matcher and the bound, so the "can these tokens rejoin" rule has one home.
- **ARCH-PURE — pass.** `wordRuns`, `highlightSpans`, `phraseRunsJoin`, `sgrState`, `scanEscape`, `decidedEnd`, `tokenStillOpen` are pure and unit-tested with no IO. `memVocabulary.Add` calls only pure functions. The one IO decision (load + colour) stays isolated in `vocabularyFor` and is pinned by a counting double.
- **ARCH-PURPOSE — pass on the feature, flag on the answering pattern.** Shadow-sweep exact: one `Has` call site, three `knownOn` consumers each deriving from the seam, one `Load()`. No hand-maintained restatement of the set anywhere; all eight Done-when rows have a named pin. The flag is that two of the round's answers were instance-shaped — BR-39's enumeration and BR-37's row — in the round whose own commit message says "sweep the class, not the list."
- **ARCH-MOCK — pass.** No new external dependency. The concurrency driver runs against `store.NewMem()` behind the same seam production uses; `llmtest.Fake` replays a committed capture; `shortWriter` sits at the `io.Writer` boundary. The gap is process, not design: with no `-race` in any gate, the stateful-double discipline is intact but the check that exercises it is not automated.
- **For #22, two things to settle at its plan gate.** (1) `Load` is not a barrier (97/200 above) and `memVocabulary` still has no removal path — narrowing to actively-learned words means the set must *shrink* mid-session, and nothing above the seam is prepared for a `Has` that flips true→false while `highlightWriter` is mid-hold. (2) `MaxPhraseWords` is now the streaming hold budget, and #22 changes what is in the set, so it changes that budget; decide once whether the bound is recomputed on removal or only ever grows.

## 7. Plan revision recommendations

One `## Revisions` entry — "close boundary round 4: the claim-rot the sweep itself created":

- **Task 8 Step 6 (`plan:340`) is ticked and its recorded mutation result is now false.** It says deleting the defer "does NOT redden anything"; measured, it reddens `TestAPoisonedWriterIsReported`. Correct it, and correct the matching sentence at `atlas/define.md:533`. The step's own closing line — *"A plan step that asserts a mutation result is a claim like any other and gets corrected when measurement disagrees"* — is the rule; apply it to itself.
- **Record BR-39 as delivered at the instance and open at the class**, naming `storeHistory.loaded`/`.lines` as the unswept sibling with the measured race at `history_store.go:40/43`, so the sweep is a written enumeration rather than a remembered one.
- **Record what `stalled upstream` actually covers.** Task 8 Step 4's path enumeration should say the table reaches three distinct branches (`err == nil`, `ctxErr`, `ErrTruncated`) across four rows, that the `default:` path is unreachable through `llmtest`, and that the interrupt-with-held-text path is exercised only colour-off — rather than implying five cells.

```findings
dispose:
  - id: BR-9
    disposition: not-addressed
    note: |
      4th round, unchanged — capture.go:132 still writes "define: " and the trailing newline itself while warnTo's comment (vocab.go:151) claims to be the one place both are written.
  - id: BR-16
    disposition: not-addressed
    note: |
      7th round. The three round-10 residues ARE fixed (no .go:NNN left in the plan, Task 4's Files/Step 1 corrected). But the same commit created a fresh instance: TestAPoisonedWriterIsReported makes plan Task 8 Step 6 and atlas/define.md:533 measurably false — verified by mutation that deleting the defer now reddens a named test.
  - id: BR-33
    disposition: addressed
    note: |
      Verified both ways — reverting to `defer hw.Flush()` AND deleting the defer entirely each redden TestAPoisonedWriterIsReported, via the package's existing shortWriter.
  - id: BR-34
    disposition: addressed
    note: |
      atlas/define.md:538-543 now documents highlightSetFor and the command-namespace withhold.
  - id: BR-35
    disposition: addressed
    note: |
      The dangling conjunction and orphaned clause are gone; the sentence reads correctly at atlas/define.md:512-514.
  - id: BR-36
    disposition: not-addressed
    note: |
      Unchanged — askhighlight_test.go:3-13 still splits the stdlib block around the store import, and re-verified no Makefile/Makefile.workflow/Makefile.local/scripts/.github target runs gofmt, goimports, go vet, go test or -race anywhere in the repo. That last one now also leaves BR-39's fix falsifiable only by hand.
  - id: BR-37
    disposition: not-addressed
    note: |
      The added row does not cover the named path. Instrumented every runAsk exit: `stalled upstream` returns err=nil ctxErr=nil, i.e. `case err == nil:` — the same branch as `clean completion`. The interrupt-with-held-text path is still exercised only at color=false with a nil vocabulary (measured: ctxErr answerLen=31 color=false vocabNil=true).
  - id: BR-38
    disposition: addressed
    note: |
      Verified by revert — dropping the phraseRunsJoin condition reddens TestAnUnmatchableKeyDoesNotWidenTheHoldWindow on both assertions; fuzzDeck.MaxPhraseWords() still measures 3, so the phrase-length axis survived the narrowing.
  - id: BR-39
    disposition: not-addressed
    note: |
      The instance is fixed and verified (unguarding `loaded` reports a race at vocab.go:130/133 and reddens the new test under -race; it passes without -race). The enumeration the finding wrote was not swept: storeHistory.Load is the same shape and races under the same driver — read at history_store.go:40, write at :43, with h.lines appended outside the mutex Add/Prefix both take. storeCapturer is clean. Separately, the comment still over-licenses: Load releases the mutex before reading the deck, so it is not a barrier — 97 of 200 two-goroutine trials had a Load() return before the deck was readable.
findings:
  - id: new
    severity: Minor
    family: behaviour-claimed-without-a-failing-test
    title: |
      The row added to close BR-37 enters no new branch and costs 30 seconds of every suite run
    detail: |
      This is the 12th finding in family `behaviour-claimed-without-a-failing-test`. Do NOT
      fix only this instance. askhighlight_test.go:128-135. Instrumented every exit of
      runAsk and ran the table: `clean completion` returns err=nil ctxErr=nil answerLen=532
      and `stalled upstream` returns err=nil ctxErr=nil answerLen=31 — both take
      `case err == nil:`. The row's comment calls it "the fifth cell ... the one remaining
      exit path" while the comment eight lines above rejects Stall for JunkFrame as "Same
      path" and "two orders of magnitude" more expensive; the row then pays that cost.
      Measured: TestEveryStreamExitPathFlushes is 30.02s, of which stalled_upstream is
      30.01s, and go test ./cmd/define/ is 93.6s against the 63.6s recorded at #20's close.
      THE RULE, which this family had not stated for TESTS rather than production code: a
      row added to close a coverage finding is a claim about which branch it reaches, and
      its falsifier is instrumentation, not the fake's documentation. Measure the branch
      before writing "the one remaining exit path". Measured prevalence this window: 1 of 1
      rows added to answer a coverage finding entered a branch already covered.
```
