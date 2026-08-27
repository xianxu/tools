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
