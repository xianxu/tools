# Boundary Review — tools#72 (whole-issue close)

| field | value |
|-------|-------|
| issue | 72 — define: stream the model's answer to the screen as it arrives |
| repo | tools |
| issue file | workshop/issues/000072-stream-answer-display.md |
| boundary | whole-issue close |
| milestone | — |
| window | 615f88e82f5b067810e36624f2bc46902bdc1ab7..423553495085a83e6336fe827665b0ce6552fc1e |
| command | sdlc close --issue 72 |
| reviewer | claude |
| timestamp | 2026-09-17T16:11:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Strong, well-evidenced work. The decoder change is exactly right: `stepLanguageDecode` is now total, the effect table matches the Spec cell-for-cell, `d.lang` has one clearing rule, and `d.body`/`languageBodyLimit`/`decodeLimit` are genuinely deleted rather than orphaned (grep-verified). I mutation-tested the two new guards in a scratch worktree and both discriminate exactly as the `## Log` claims — restoring the body buffer gives `TestALongPassageReachesTheScreenInPieces: a 1262-byte answer arrived in 11 writes, the largest 818 bytes` and `TestOwnedTextIsEmittedBeforeItsCloseMarker: nothing reached the sink while the passage was still open`; restoring the one-shot `highlightRegion` on the owned path reds both `TestOwnedPassageHighlightsAWordSplitAcrossDeltas` and `TestLanguageAnswerForeignHomographDoesNotUseTargetVocabulary`. `go vet ./...` clean; `go test ./cmd/define` passes except `TestLanguagePromptStartup`/`TestLanguageTintInvocation`, which fail identically at the base commit (`pty.Open()` → operation not permitted — an environment limit, not this diff). 20 s of `FuzzLanguageDecoderChunks` with the retention invariant: clean. What keeps this off SHIP is one traceability gap: the Plan's end-to-end guard — `Reply{AfterText, FinishRelease}`, the barrier PQ-5 was disposed `addressed` on — is not in the diff, and I verified by mutation that the test shipped in its place does not cover the ordering claim the Done-when makes.

### 1. Strengths

- `cmd/define/language_decode.go:90` — the transition function became **total** by deleting an event rather than by adding a default arm, and `language_decode_test.go:148` restates the 3×5 matrix independently (literal enum constants, not read off the implementation). This is the ARCH-ORDER pattern done properly.
- `cmd/define/answer_language.go:68` — the PQ-1 fix removes a mechanism instead of adding one. Owned and neutral text now share the single streaming `highlightWriter`, and the flush-then-replace ordering (flush while `a.ownership` still names the old run, *then* swap) is both correct and explained at the point it matters.
- `cmd/define/language_decode.go:12` — `maxLanguageDecoderRetained` stated as a **total** rather than per-field, with the reason written down: `d.body.Len() < languageBodyLimit` named a field and so stopped checking when the field went away. `retained()` covers all five components, asserted after every fuzz chunk.
- `cmd/define/language_decode_test.go:29` — the three fixture rows whose expectations flipped each carry a comment tying the change to the Spec's accepted consequence, so a future reader can't mistake them for tests bent to fit.
- `internal/llm/llmtest/testdata/README.md:50` + `language_conformance_test.go:104` — the new capture ships with a recorder that **refuses to promote a non-dominant passage**, and the README records why an English session produces the wrong shape. That is the right answer to "a capture is evidence only for the shape it was recorded in".
- Fixture hygiene: `stream-long-passage.sse` is committed 100644, contains only SSE event/data lines, and carries no credentials (scanned).

### 2. Critical findings

None.

### 3. Important findings

**`cmd/define/ask_language_test.go:139` — the production-chain ordering assertion the Done-when names is not delivered.**

Done-when line 1: *"A test asserts text reaches the sink **before the stream ends**, on annotated input, through the production chain."* Plan row 6 and PQ-5's `addressed` disposition both name the mechanism: `Reply{AfterText, FinishRelease}` (it exists at `internal/llm/llmtest/fake.go:115-117` and is already used by `cmd/define/llm_activity_test.go:26` and `activity_conformance_test.go:28`).

What shipped instead is a post-hoc largest-write check (`sink.largest > 64`) with the stream already closed. I mutated the decoder to buffer the passage and release it **rune-by-rune** at the close marker, then ran the whole package: `TestALongPassageReachesTheScreenInPieces` **passed**. Only the pure decoder test caught it. So no test through the production chain asserts the ordering property, and a downstream hold (in `languageAnswer` or the wrap writer, where the decoder test cannot see) would ship green.

Fix sketch: script the capture with `AfterText`/`FinishRelease`, assert `sink.Len() > 0` (or `sink.writes > 1`) at the `AfterText` rendezvous, then close `FinishRelease`. Keep the largest-write check — it is a good granularity oracle and it is the one the mutation confirms; it just isn't a timing oracle.

### 4. Minor findings

- `cmd/define/ask_language_test.go:147` — the e2e streaming test runs with `options{color: true}`, so `opt.width == 0` and `WriteOwned` is a pass-through: `ownedAnswerWrapWriter`'s row-commit path never runs. The envelope's "~0.5 s per row at width 100" and "first row at 0.3 s" stay Log-only facts (ARCH-CONSTRAINTS).
- `cmd/define/askhighlight_test.go:29` — `splitWordInsideAPassage` requires only that the word sit inside *some* `[lang=..]` region, but the test sets `d.lang = "en"` and so needs an **`en`** region; an `es` one yields `vocabularyFor → nil` and no highlight. Today every split-word candidate in `stream-language.sse` happens to be in an `en` region (I enumerated them: `phrase`, `that's`, `greeting`, `fixed`, `waking`, `roughly`, `lunchtime`, `stretch`, `bakery`, `strangers`, `midday`, `feminine`), so it passes — but a re-record can turn a correct implementation into a misleading `was not highlighted` failure. Pass the wanted lang into the predicate.
- `cmd/define/language_decode_test.go:93` — `decodedChunksBounded` is a verbatim copy of `decodedChunks` (`:9`) plus three lines of assertion, and the conformance test at `language_conformance_test.go:120` open-codes a third copy of the same span-offset accumulator. One helper with an optional per-chunk hook and a merge flag (ARCH-DRY).
- Two statements of one rule: `assertDominantPassage` (`ask_language_test.go:174`) requires longest ≥ 50 % of **raw** bytes including marker text, while the recorder (`language_conformance_test.go:143`) requires ≥ 60 % of **decoded** text. Same predicate, two denominators — they can disagree about the same capture.
- `cmd/define/askhighlight_test.go:49` — `annotatedRegions` doesn't handle nesting. In `stream-long-passage.sse` the nested `[lang=en]Sycophant[lang=es][/lang][/lang]` produces a region whose body contains marker bytes. Harmless for the current thresholds, wrong as a general helper.
- `cmd/define/answer_language.go:79` — the new method `(*languageAnswer).vocabularyFor` shares its name with the package-level `vocabularyFor(d, opt)` called at `ask.go:164`. Legal, but two different things named the same in one package.
- Prose nit: the write count is given as 1,232 in `workshop/lessons.md` and the issue `## Log`, 1262 bytes in the `countingSink` comment; the run logs `1262 bytes in 1233 writes`.

### 5. Test coverage notes

- Both new guards are mutation-proven (above) — this is the standard the checklist asks for and it was met for the behavior that shipped.
- The enumeration `{neutral, owned} × {split across deltas}` is now complete, and the boundary invariant (`TestLanguageAnswerForeignHomographDoesNotUseTargetVocabulary`) reds under the reverted fix, so it is load-bearing rather than incidental.
- Uncovered cell: the ordering property through the production chain (finding 3.1).
- Uncovered cell: ownership transitions at `width > 0`. Every `answer_rows_test.go` case uses widths 20–40 but predates own-at-open; no test drives a *streamed* owned run through the wrap writer at a realistic width, which is where a row could now legitimately turn up mixed-ownership where it previously arrived whole. Worth one row.

### 6. Architectural notes

Marker-by-marker, at-review lens:

- **ARCH-DRY** — pass on production code; the change *removes* a parallel mechanism (`highlightRegion` retired from the answer path, still correctly serving its seven other callers). Flagged twice in test scaffolding (4.3, 4.4).
- **ARCH-PURE** — pass. Decoder and `stepLanguageDecode` are pure and tested with no clock, socket or fs; `retained()` is a pure oracle; `languageAnswer` is the thin adapter.
- **ARCH-PURPOSE** — pass, including the shadow-sweep: `d.body`, `languageBodyLimit` and `decodeLimit` are all gone, not merely unused; the deferred tint question is genuinely recorded in `workshop/issues/000064-...md:74`; the out-of-scope row hold is scoped out with a measurement rather than a hand-wave.
- **ARCH-MOCK** — pass with the finding above as the caveat. Production and test share the capture/fake seam, and a live conformance check with a promotion guard exists. But the fake's *ordering* barrier — the half of the seam that makes timing observable without a clock — was specified and not used.
- **ARCH-CONSTRAINTS** — flagged (4.1). The envelope is well-written and the first-byte figure is measured, but nothing mechanical enforces it, which is the state PQ-5 raised and the plan promised to leave.
- **ARCH-SECURE** — pass. The decoder parses untrusted model output; language codes go through `store.ParseLang`; retention is now bounded by grammar and asserted per chunk by the fuzz target, which is strictly stronger than the deleted size check. New fixture carries no credentials.
- **ARCH-ORDER** — pass, and the strongest part of the diff. One transition function, total, independently-stated matrix, one `d.lang` clearing rule replacing two divergent sites. The new cross-event field `languageAnswer.ownership` has a single mutator (`own`) and a readable invariant; it is not a boolean constellation.
- **ARCH-FUNERAL** — pass. Net deletion of a growing buffer; one new 25 KB fixture with a documented re-record command and no per-run growth; `DEFINE_LONG_PASSAGE_CAPTURE` writes only where the operator points it.

For upcoming work (#64): `languageAnswer.ownership` is now the single place answer-level language state lives. When the stage model arrives and a reply language is actually *requested*, `vocabularyFor` is the one function that needs to change — it currently hard-codes "neutral prose keeps the session's vocabulary", which is a policy statement that belongs with the stage model.

### 7. Plan revision recommendations

One `## Revisions` entry, if finding 3.1 is disposed as a deferral rather than fixed:

> **2026-09-17 — close review.** The end-to-end guard shipped as a largest-single-write assertion (`TestALongPassageReachesTheScreenInPieces`) rather than the `Reply{AfterText, FinishRelease}` ordering barrier plan row 6 named and PQ-5 was disposed on. Mutation-checked: the delivered test catches a buffer-released-in-one-write regression (818 bytes) but passes a buffer-released-rune-by-rune one, which only the pure decoder test catches. Done-when line 1 ("text reaches the sink before the stream ends … through the production chain") is therefore satisfied at the decoder, not at the chain. [State here whether the barrier is being added now or carried, and to where.]

If the barrier is added instead, no plan revision is needed — the plan and code will agree.

```findings
findings:
  - id: new
    severity: Important
    family: envelope-guard-unautomated
    title: |
      the production-chain ordering assertion the Done-when names is not delivered
    detail: |
      Done-when line 1 and plan row 6 require a test that text reaches the sink
      BEFORE the stream ends, through the production chain, driven by
      Reply{AfterText, FinishRelease} — the barrier PQ-5 was disposed `addressed`
      on. The delivered TestALongPassageReachesTheScreenInPieces
      (cmd/define/ask_language_test.go:139) instead checks largest single write
      after the run completes. Verified by mutation in a scratch worktree: a
      decoder that buffers the passage and releases it rune-by-rune at the close
      marker PASSES that test; only the pure decoder test catches it. So no
      production-chain test asserts the ordering property, and a hold introduced
      downstream of the decoder would ship green. The barrier already exists at
      internal/llm/llmtest/fake.go:115-117 and is used by llm_activity_test.go:26.
  - id: new
    severity: Minor
    family: envelope-guard-unautomated
    title: |
      the e2e streaming test runs at width 0, so the wrap writer is never exercised
    detail: |
      options{color: true} leaves opt.width at 0, so WriteOwned is a pass-through
      and ownedAnswerWrapWriter's row-commit path — the remaining hold, and the
      subject of the envelope's "first row 0.3 s, ~0.5 s per row at width 100"
      budget — is not on the tested path. Those figures stay Log-only facts
      (ARCH-CONSTRAINTS).
  - id: new
    severity: Minor
    family: test-helper-underconstrained
    title: |
      splitWordInsideAPassage does not require the passage language to match the session
    detail: |
      cmd/define/askhighlight_test.go:29 accepts a word in ANY [lang=..] region,
      but TestOwnedPassageHighlightsAWordSplitAcrossDeltas sets d.lang = "en" and
      only an `en` region can highlight (vocabularyFor returns nil otherwise). All
      twelve split-word candidates in stream-language.sse happen to sit in `en`
      regions today, so it passes; a re-record can turn a correct implementation
      into a misleading "was not highlighted" failure. Pass the wanted lang into
      the predicate.
  - id: new
    severity: Minor
    family: duplicated-test-helper
    title: |
      three copies of the span-offset accumulator, and decodedChunksBounded clones decodedChunks
    detail: |
      cmd/define/language_decode_test.go:93 is a verbatim copy of :9 plus three
      lines of assertion, and language_conformance_test.go:120 open-codes a third
      copy with merging added. One helper with an optional per-chunk hook and a
      merge flag (ARCH-DRY).
  - id: new
    severity: Minor
    family: one-rule-two-statements
    title: |
      "dominant passage" is defined twice, over different denominators
    detail: |
      assertDominantPassage (cmd/define/ask_language_test.go:174) requires the
      longest region to be at least half the RAW capture including marker bytes;
      the recorder (language_conformance_test.go:143) requires 60 percent of
      DECODED text. The replay guard and the promotion guard can disagree about
      the same capture.
  - id: new
    severity: Minor
    family: one-rule-two-statements
    title: |
      annotatedRegions does not handle nested markers
    detail: |
      cmd/define/askhighlight_test.go:49 takes the first [/lang] after an open, so
      the nested [lang=en]Sycophant[lang=es][/lang][/lang] in
      stream-long-passage.sse yields a region whose body contains marker bytes.
      Harmless at the current thresholds, wrong as a general helper.
  - id: new
    severity: Minor
    family: shadowed-name
    title: |
      (*languageAnswer).vocabularyFor shares its name with the package-level vocabularyFor
    detail: |
      cmd/define/answer_language.go:79 versus the package function called at
      ask.go:164. Legal Go, but two unrelated things named the same in one
      package.
```

---

## Re-review — 2026-09-17T16:33:13-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 72 — define: stream the model's answer to the screen as it arrives |
| repo | tools |
| issue file | workshop/issues/000072-stream-answer-display.md |
| boundary | whole-issue close |
| milestone | — |
| window | 615f88e82f5b067810e36624f2bc46902bdc1ab7..456a01cabc94aa2650f06042098cec1499263431 |
| command | sdlc close --issue 72 |
| reviewer | claude |
| timestamp | 2026-09-17T16:33:13-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All seven prior findings are genuinely addressed, and BR-1 — the only blocking one — is addressed with real regression evidence: I rebuilt the "buffer the passage, dribble it rune-by-rune at the close marker" mutation in a scratch worktree at HEAD and `TestALongPassageReachesTheScreenInPieces` now fails on both subtests at `nothing reached the screen until delta 102 of 161`, where the round-1 version passed. The production change itself is clean and the rare kind that *removes* mechanism: `stepLanguageDecode` loses two effects, one event and the `ok` return and becomes total; `d.lang` gets one clearing rule; the owned path stops being a second highlighter. `go build ./...`, `go vet` (both tag sets) and `go test ./cmd/define` are clean apart from `TestLanguagePromptStartup`/`TestLanguageTintInvocation`, which fail at `pty.Open(): operation not permitted` — an environment restriction in files this window does not touch, matching the Log's note. What stops a bare SHIP is one user-facing documentation error: `cmd/define/README.md:351` still tells the reader "incomplete annotations fall back to neutral text", which is precisely the rule this issue inverted and which `cmd/define/ask_language_test.go:88` now asserts the opposite of.

**1. Strengths**

- `cmd/define/language_decode.go:72-95` — the effect table is smaller than what it replaced, and the accepted behaviour change (a nested/bad/unterminated segment keeps its announced language) is written into the function's own doc comment rather than left for a reader to infer from a diff.
- `cmd/define/language_decode.go:145-151` — the single `old == decodeSegment && next != decodeSegment` clearing rule genuinely collapses four exits into one site, and no effect arm can observe a half-cleared `d.lang` (the only `decodeEmitOwned` cell is segment→segment).
- `cmd/define/language_decode.go:13-25` + `:267` — `maxLanguageDecoderRetained` stated as a **total** with the reason recorded (`d.body.Len() < languageBodyLimit` named a field, so it stopped checking instead of failing when the field died). I checked each component against the code: `marker` ≤63 (`lex:201`), `entity` ≤63 (`output:245`), two `pending` ≤3 each (`answer_text.go:30-32`), `literalText` = 0 at every event boundary via `output`'s deferred flush. Worst case 132 against a bound of 136 — correct, and asserted after every fuzz chunk.
- `cmd/define/answer_language.go:68-75` — flush-before-reassign is subtle and the comment earns its length: the flush runs while `a.ownership` still names the old run, so held text is tagged with the run it arrived in. `TestLanguageAnswerForeignHomographDoesNotUseTargetVocabulary` is preserved unchanged as the boundary invariant rather than rewritten to fit.
- `cmd/define/ask_language_test.go:219-235` — `assertDominantPassage` is the right shape of guard: the ordering test is only meaningful while the fixture is one long passage, and neutral prose streamed per-rune even *before* the fix, so without this the test would go silently inert on a re-record. I measured the committed capture at 804/1232 decoded bytes (0.653) against the 0.60 threshold.
- `cmd/define/language_conformance_test.go:102-156` — a recorder that refuses to promote a capture that cannot exhibit the defect, sharing `dominantPassage` with the replay guard. This is the ARCH-MOCK seam done properly: live conformance behind a build tag, stateful fake for the suite, one predicate governing promotion and replay.

**2. Critical findings**

None.

**3. Important findings**

- **`cmd/define/README.md:351` — the user-facing statement of the rule this issue inverted was not swept.**
  > **This is the 1st finding in family `docs-not-swept-with-behavior`** (new family; it is *not* `one-rule-two-statements`, whose two prior instances are executable duplication — a predicate implemented twice, a grammar re-implemented in a helper. The rule here is different: when a behaviour changes, every *statement* of it must be swept, not only the one the Done-when named.)

  The Done-when named `atlas/define.md`'s "malformed/nested/incomplete segments preserve neutral prose" sentence, and that one was correctly replaced (`atlas/define.md:2313-2318`). But the rule is stated in three places and only two were swept. README:351 still reads "Model language annotations are removed before display and storage, even with tint off; **incomplete annotations fall back to neutral text**" — false since this diff: `cmd/define/language_decode_test.go:92` pins `[lang=es]unfinished[/lang` → owned `unfinished`, and `cmd/define/ask_language_test.go:88` asserts that a Ctrl-C'd unterminated passage still carries `languageDark`. A reader who hits Ctrl-C mid-answer sees a tinted fragment the README says will be neutral.

  Fix as the class, not the instance: the enumeration is `grep -rin "fall back to neutral\|preserve neutral prose\|incomplete annotation"` over `atlas/`, `cmd/define/README.md`, `README.md` and Go doc comments — it returns exactly one unswept member today, so the sweep is cheap and the enumeration is worth writing into the issue's Done-when for next time. Suggested replacement for the clause: *"an incomplete or malformed annotation keeps the language its opening marker announced; text arriving after the malformed marker is neutral."* (ARCH-PURPOSE at-review: the instance the Done-when named was fixed, an enumerable sibling was not.)

**4. Minor findings**

- **`cmd/define/askhighlight_test.go:89-96` — `annotatedRegions` still reports as owned the region that *follows* a nested open, which the parser treats as recovery.**
  > **This is the 2nd finding in family `test-helper-underconstrained`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

  The rule: *a helper that nominates candidates for a production assertion must be no more permissive than the production path, or a correct implementation fails as "was not highlighted."* BR-3 was that rule applied to language; this is the same rule applied to nesting. The helper correctly ends a region at a nested open (BR-6's fix), then sets `at = body+i` and rescans from the nested marker, emitting a region for it — but in `language_decode.go:83-85` a nested open enters `decodeRecovery`, which only leaves on close/finish, so `[lang=en]a[lang=es]bbb[/lang]` leaves `bbb` unowned while the helper calls it an owned `es` region. The comment at `:63-65` states the parser's rule correctly and the code below it does not follow through. Measured prevalence: unreachable today — `stream-language.sse` (the only capture this helper runs against) has no nested markers at all, and the nested region in `stream-long-passage.sse` is degenerate/empty. The durable fix for the family is to stop re-implementing the grammar: drive the capture through `newLanguageDecoder` + `spanAccumulator` and read `spans`, carrying a raw→decoded offset map, rather than maintaining a second marker parser in test code.

- **`cmd/define/language_conformance_test.go:51-59` — one open-coded span accumulator survives the consolidation.**
  > **This is the 2nd finding in family `duplicated-test-helper`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

  The rule: *there is one way to reassemble a `languageText` from decoder emissions, and it is `spanAccumulator`.* BR-4 swept three copies into one; this one was out of its stated scope (pre-existing, non-merging) and sits 70 lines above a comment reading "The SHARED accumulator, which merges adjacent runs". Enumeration: `grep -n 'got\.text += v\.text\|out\.text +=' cmd/define/*_test.go` — one member remains. It is still *correct* (it concatenates per-language spans, and concatenating adjacent rune-spans yields the same text), so this is tidiness, not a bug; but leaving it is how the third copy comes back.

- `cmd/define/language_decode_test.go:35-44` and `:139-147` — the same four-sentence paragraph about why the retention invariant lives in the fuzz path is written twice verbatim on two adjacent helpers. One of them should be a cross-reference.

- `cmd/define/language_decode_test.go:42-44` — "calling `after()` at every point the decoder is between events" overclaims slightly: the hook fires at the two chunk boundaries and after `Finish`, not between events *within* a chunk. The bound that matters is the one held across arrivals, so the coverage is right; the sentence isn't.

- `cmd/define/language_decode.go:267` — `retained()` has no production consumer. Justifiable as the envelope's declared oracle, but worth a one-line comment saying so explicitly so a future sweep for dead methods doesn't delete the guard's only seam.

**5. Test coverage notes**

- The ordering property is now delivered and *kills* the mutation that defeated round 1's version. I confirmed independently that the sink receives nothing but answer bytes: `activityEnabled` is `opt.tty && !opt.raw` (`activity.go:31`) and the test leaves `tty` false, so no spinner write can make `firstVisible` trivially 1. The observed margins are healthy — delta 2/161 piped, 13/161 at width 100, against a `deltas/4` threshold.
- BR-2's width-100 row is real coverage, not ceremony: the terminal subtest goes through `ownedAnswerWrapWriter`'s row-commit path (17 writes, largest 135) rather than the width-0 pass-through, so the second hold is now on the tested path.
- I checked the per-rune write granularity against `answerwrap.go:271-284`: `acceptUnit` still buffers a whole word in `w.word` before `emitWord`, so word-wrap is unaffected by the change in call granularity, and `advanceRowOwnership` consumes display units rather than writes — no row-tint drift from the finer calls.
- Remaining gap, stated rather than raised (see §6): the envelope's wall-clock figures (first row 0.3 s, ~0.5 s per row) are still `## Log` facts with no mechanical guard. That was a deliberate PQ-5 decision — an ordering assertion where a timing assertion would be flaky — and I agree with it; I am not re-raising `envelope-guard-unautomated`.

**6. Architectural notes for upcoming work**

- **ARCH-DRY — pass** on production code, emphatically: the diff deletes a parallel mechanism (`highlightRegion` retired from the answer path; its seven other callers untouched) rather than adding one. Flagged twice in test scaffolding only (§4).
- **ARCH-PURE — pass.** The decoder and `stepLanguageDecode` are pure and unit-tested with no clock, socket or fs; `languageAnswer` is a thin adapter; IO lives in the wrap writer. The new ordering test needs a client wrapper, but that wrapper only *observes* — the stack still runs against the llmtest fake.
- **ARCH-PURPOSE — pass with the §3 exception.** Shadow-sweep of the single-source change: `d.body`, `languageBodyLimit`, `decodeLimit`, `decodeAppend`, `decodeFlushOwned`, `decodeFlushNeutral` are all gone from the tree, not merely unused (verified by grep); the `#64` handoff is genuinely recorded at `workshop/issues/000064-...:74`; the out-of-scope row hold is scoped out with a measurement rather than a hand-wave. The one consumer of the changed rule that did not derive from it is `cmd/define/README.md`.
- **ARCH-MOCK — pass.** Production and test flow share the `llm.Client` boundary; the fake is stateful and capture-driven; a live conformance check exists with a documented re-record command and a promotion guard. The `#72` lesson — *a capture is evidence only for the shape it was recorded in* — is the strongest structural idea in this diff and is worth generalising: the next capture-backed guard should carry its own "can this fixture still exhibit the defect" assertion the way `assertDominantPassage` does.
- **ARCH-CONSTRAINTS — pass.** Envelope declared per interaction path; retention now has a stated total with a fuzz-asserted oracle; both width regimes measured. The per-rune `WriteOwned` at width 0 means ~1233 small writes where there used to be 2 — over a ~10 s generation that is noise, and at real widths the wrap writer coalesces to ~17. No action.
- **ARCH-SECURE — pass.** `lex` is the rune scanner over untrusted model output and is fuzzed with a retention invariant after every chunk; `store.ParseLang` is the typed boundary for a language code; the new capture contains response SSE only (I checked for `authorization`/`api-key`/`bearer`/`sk-` — zero hits), consistent with the transport teeing only the response body.
- **ARCH-ORDER — pass, and this is the diff's best work.** The `(state, event) → (state, effect)` enumeration is explicit, total, owned by the pure core, and pinned by an *independently stated* matrix (`language_decode_test.go:188-197`) rather than one read off the implementation. The interleaving seam is real and injectable: `TestLanguageDecoderRecoveryAndSplits` re-runs every fixture at every byte split, and the fuzz target asserts chunk-independence — so a failing ordering is reproducible, not a sample of size one. The interrupting event that matters here (cancellation mid-passage) is enumerated and asserted at `ask_language_test.go:83-97`, including the accepted consequence.
- **ARCH-FUNERAL — pass.** The diff's net effect on residue is negative: the 16 KiB segment body is deleted and replaced with a 136-byte stated total. New durable artifacts are one fixture file and one gate ledger, both fixed-size and archived with the issue.

**7. Plan revision recommendations**

- Add a `## Revisions` entry — *2026-09-17 — boundary review round 1* — recording that **Plan row 6's named mechanism changed**. The row still reads "Drive it through `Reply{AfterText, FinishRelease}` and assert the sink holds text while the stream is still open", and PQ-5 was disposed `addressed` on exactly that barrier. What shipped is `deltaObserver` (`cmd/define/ask_language_test.go:146-158`), because `AfterText` holds after the *first* text delta, which in this capture is the bare `[lang=es]` marker — so the sink is legitimately empty there with or without the bug. The Log tells this story well; the Plan does not, and per AGENTS.md the plan artifact is what should stop claiming what the code doesn't deliver.
- In the same entry, note that the envelope's "Total ≤ 200 B" retention figure was implemented as `maxLanguageDecoderRetained = 136` (`2*languageHeaderLimit + 2*utf8.UTFMax`). The implementation is tighter than the Spec, so nothing is wrong — but the Spec's number is now the looser of two statements of one bound, which is the shape of a future drift.
- Extend the Done-when line "atlas/define.md states the streaming behaviour and replaces the sentence…" to name the **enumeration** rather than the single file, per the §3 finding: atlas + `cmd/define/README.md` + the decoder doc comment.

---

## Re-review — 2026-09-17T16:47:12-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 72 — define: stream the model's answer to the screen as it arrives |
| repo | tools |
| issue file | workshop/issues/000072-stream-answer-display.md |
| boundary | whole-issue close |
| milestone | — |
| window | 615f88e82f5b067810e36624f2bc46902bdc1ab7..5705992f12d849f58c13ecbd50296301228ff3c8 |
| command | sdlc close --issue 72 |
| reviewer | claude |
| timestamp | 2026-09-17T16:47:12-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All seven round-1 findings are genuinely fixed, and I confirmed the two that mattered by mutation in a scratch worktree rather than by reading: restoring a buffer-then-dribble decoder now reds `TestALongPassageReachesTheScreenInPieces` at *both* widths ("nothing reached the screen until delta 102 of 161") and `TestOwnedTextIsEmittedBeforeItsCloseMarker`; restoring the one-shot `highlightRegion` on the owned path reds `TestOwnedPassageHighlightsAWordSplitAcrossDeltas` and `TestLanguageAnswerForeignHomographDoesNotUseTargetVocabulary`. The BR-1 gap — a granularity oracle standing in for an ordering claim — is closed with a real production-chain ordering assertion. `stepLanguageDecode` is total, matches the Spec table cell-for-cell against the independently-stated test matrix, `d.lang` has exactly one clearing rule, and `d.body`/`languageBodyLimit`/`decodeLimit` are gone from the tree (grep-verified). `go vet ./cmd/define` clean under both build tags; the full package passes except `TestLanguagePromptStartup`/`TestLanguageTintInvocation`, which fail identically at the base commit with `pty.Open()` → operation not permitted (environment, not this diff — I ran them at 615f88e to check). 20 s / 258 k executions of `FuzzLanguageDecoderChunks` with the retention invariant: clean. What keeps this off SHIP is the shadow-sweep: the enumeration of "documents stating the replaced rule" covered the three prose consumers and missed the one that is an executable contract — the live ask system prompt still tells the model to keep passages under 4000 characters, which is the prompt-side spelling of the 16 KiB body bound this issue deleted (both were introduced in the same commit, 62c6a66 / #65).

## 1. Strengths

- **`cmd/define/language_decode.go:141-161`** — the `d.lang` clearing rule is stated once, structurally (`old == decodeSegment && next != decodeSegment`), so the four exits from a segment cannot drift apart. This is the ARCH-ORDER win the Spec promised, and it is enforced by shape rather than by discipline.
- **`cmd/define/language_decode.go:13-25`** — `maxLanguageDecoderRetained` as a *total* with the reason written down ("`d.body.Len() < languageBodyLimit` named one field, so it became uncheckable the moment that field went away instead of failing"). `retained()` covers all five components; asserted after every fuzz chunk, not just in a hand-written case.
- **`cmd/define/ask_language_test.go:190-213`** — the ordering guard is the right oracle and it discriminates. `firstVisible*4 > deltas` is a ratio over the production delta stream, not a wall clock, so it is deterministic; the `deltaObserver` seam observes without holding. Recording in the Plan that this is *not* the `Reply{AfterText, FinishRelease}` barrier PQ-5 was disposed on, with the reason, is exactly the right disposal of a mechanism that turned out not to work.
- **`cmd/define/answer_language.go:57-75`** — the fix deletes a mechanism instead of adding one, and the comment on `own` explains why *both* halves of flush-then-replace are load-bearing (held text belongs to the run it arrived in; vocabulary swap is what stops a phrase spanning a boundary). `TestLanguageAnswerForeignHomographDoesNotUseTargetVocabulary` is preserved unchanged as the boundary invariant rather than adjusted.
- **`cmd/define/language_decode_test.go:180-200`** — the transition matrix is restated by hand, column-labelled, with the changed cell called out. An oracle read off the implementation would assert nothing, and this one isn't.
- The three `workshop/lessons.md` entries are unusually good: "a double that measures something must implement nothing it did not write" and "a threshold that happens to pass is not evidence it discriminates" are both transferable rules, not war stories.

## 2. Critical findings

None.

## 3. Important findings

### 3.1 — `cmd/define/askctx.go:176`, `cmd/define/language_decode.go:245` — the shadow-sweep missed the executable consumer of the deleted bound

**This is the 3rd finding in family `one-rule-two-statements`.** Round 1 fixed two instances (the envelope's "200 B" vs `maxLanguageDecoderRetained` = 136; "dominant passage" defined over two denominators). Per the repeat protocol I am not asking for this instance to be patched — the rule is:

> **A quantitative bound in this subsystem has exactly one statement. Every other site derives from it or cites it by name. A prompt is a site.**

Measured prevalence — the enumeration this rule implies, over the whole `[lang=..]` subsystem:

| bound | statements | status |
|---|---|---|
| header candidate, 64 B | `languageHeaderLimit` only | single-sourced ✅ |
| decoder retention, 136 B | constant + Spec envelope | fixed round 1 ✅ |
| dominant passage, 500 B / 60 % | `dominantPassage()` | fixed round 1 ✅ |
| **passage length, 16 KiB** | `languageBodyLimit` (deleted) **+ "Keep passages below 4000 characters" in `sharedLanguageGrammar`** | **open** |
| **entity candidate, 64 B** | bare literal at `language_decode.go:245` **+ `2*languageHeaderLimit` in `maxLanguageDecoderRetained`** | **open** |

Two of five members were never enumerated, and both are in the direction the rule predicts — the surviving statement is the one nobody could grep for.

The passage-length member is the substantive one. `sharedLanguageGrammar` carries its own comment saying it "is a WIRE FORMAT, so a second spelling of it is a second format", and 4000 characters is 16000 bytes — the UTF-8 worst-case fit inside `languageBodyLimit`. `git log -S` puts the prompt clause and the constant in the same commit (62c6a66, #65). With the body gone the clause protects nothing, still costs prompt tokens, and *pushes the model toward the passage shape this issue's fixture and conformance recorder have to fight* — the `## Log` records one recording whose longest span was 3 bytes, and `TestLongPassageStreamsAgainstLiveService` exists precisely to refuse those. It also quietly contradicts the atlas line landed in this same diff ("the only bounds now are the 64-byte header candidate and a stated total retention"). Either delete the clause or restate what it now buys.

The entity member is cheap and was *introduced* by this diff: `maxLanguageDecoderRetained = 2*languageHeaderLimit + 2*utf8.UTFMax` now depends on the entity cap being `languageHeaderLimit`, while the entity check is a bare `64`. It fails loudly rather than silently (the fuzz invariant would red), so it is the mild member — but it is the member the rule predicts, so sweep it with the same pass.

## 4. Minor findings

### 4.1 — `cmd/define/askhighlight_test.go:70-99` — `annotatedRegions` still reports text after a nested open as owned

**This is the 2nd finding in family `test-helper-underconstrained`.** BR-3 fixed the instance (pass the wanted lang in). The rule that covers both:

> **A test helper that must know where the parser puts a boundary derives that boundary from the parser, not from a second scan.**

BR-6's fix ended a region at a nested open, and the comment claims the helper now does "what `language_decode.go` does, where a nested open drops into recovery and the text after it is no longer owned." It then sets `next = body+i` and rescans *from* that nested open, so the nested passage is emitted as a region. Measured in a scratch test:

```
input "[lang=en]foo[lang=es]bar[/lang] tail"
  annotatedRegions: en "foo", es "bar"
  parser  (spans): en "foo"          ← "bar" is in recovery, owned by nobody
```

Harmless on today's capture (the nested region in `stream-long-passage.sse` is empty), and the failure mode after a re-record is a false red, not a false green. But it is a second grammar, and the derivation is available: run the capture through `languageDecoder` one delta at a time, record the decoded length at each delta boundary, and read both "split across deltas" *and* "owned by lang" off the emissions. That collapses `annotatedRegions`, the raw-vs-decoded offset mismatch and BR-3's lang parameter into one mechanism that cannot disagree with production.

### 4.2 — `cmd/define/language_decode_test.go:31-40` — the retention-invariant rationale is duplicated verbatim onto a function that doesn't assert it

**This is the 2nd finding in family `duplicated-test-helper`** (BR-4 was the three span accumulators). Same rule — one statement. The 8-line comment above `decodeChunks` is character-for-character the comment above `decodedChunksBounded` at :138-147, and `decodeChunks` takes a no-op hook in its `decodedChunks` form and asserts nothing. Delete the copy at :31-40; the invariant's home is `decodedChunksBounded`.

### 4.3 — `cmd/define/ask_language_test.go:222-225`

`assertDominantPassage`'s second sentence ("Stated twice over two denominators — raw bytes here, decoded text there — …") reads as a statement of the current state rather than the counterfactual it means. It describes the bug BR-5 fixed, in the present tense, in the fix.

## 5. Test coverage notes

- The two new guards discriminate; I verified both by mutation rather than by reading, and against the *dribble* variant specifically (the one that defeated the round-1 test), not just the original buffer.
- BR-2 is genuinely closed: the `terminal` subtest runs at width 100 and produces 17 writes / largest 135 against a 1742-byte render, so `ownedAnswerWrapWriter`'s row-commit path is on the tested path.
- `assertDominantPassage` is the right kind of guard — it stops the streaming test going inert after a re-record. Margin is real but not large: 818 / 1261 decoded bytes = 64.9 % against a 60 % floor. Worth knowing when the capture is next re-recorded.
- End-to-end tint coverage for a *well-formed closed* passage is thinner than for the unterminated one, but it exists (`TestLanguageAnswerWrapClosesBackgroundBeforePhysicalNewline` at width 20, `TestAskAnnotatedCancellationFlushesBeforeHistory` for the open case), and both exercise the new `a.ownership` tagging path. No gap worth a finding.
- `TestLanguageAnswerWriteFailureStillFinishesCleanTranscript` still pins `w.calls == 1` through the new flush-and-replace, so the poison contract survives the highlighter being swapped mid-stream. That one was easy to break and wasn't.

## 6. Architecture

- **ARCH-DRY** — pass on production: the change *removes* a parallel mechanism (`highlightRegion` retired from the answer path, still correctly serving its seven other callers). Test scaffolding is much better than round 1 (one `spanAccumulator`, one `decodeChunks`, one `dominantPassage`). Flagged at 4.2.
- **ARCH-PURE** — pass. `stepLanguageDecode` is a total pure function; `languageDecoder` holds no IO; `languageAnswer` is pure over an injected `io.Writer`. Every new decoder test runs without a clock, socket or fs beyond testdata.
- **ARCH-PURPOSE** — **flag (3.1)**. The code half of the sweep is complete and verified: `d.body`, `languageBodyLimit` and `decodeLimit` are deleted, not orphaned, and the `#64` tint question is genuinely recorded (`workshop/issues/000064-…md:73-82`). The prose half is complete (atlas, both READMEs, doc comment). The *executable* half is not — the prompt is a consumer of the deleted bound and was never enumerated. This is the shadow-sweep's own failure mode: the consumer that doesn't look like a document.
- **ARCH-MOCK** — pass. The `llmtest` fake is the seam for both new tests; `deltaObserver` wraps `llm.New(cfg)` through `d.newLLM` rather than bypassing it. The live conformance check exists (`TestLongPassageStreamsAgainstLiveService`) with a promotion guard and a documented re-record command, and it refuses to promote a capture that can't exhibit the defect.
- **ARCH-CONSTRAINTS** — pass with the round-1 caveat unchanged. The envelope's ordering claim now has a mechanical guard at both widths; the 0.3 s / 0.5 s-per-row figures stay `## Log` facts, which was disposed at BR-2 and I am not re-raising. Retention is bounded and asserted per chunk. No unbounded fan-out; the per-ownership-run `highlightWriter` allocation is bounded by span count, and spans merge.
- **ARCH-SECURE** — pass. Model output is the untrusted input, and the diff *strengthens* the parse: the bound moved from one field to a total over all five retained components, asserted by the fuzz target after every chunk with four own-at-open seeds. The new capture carries no credentials (checked). The accepted widening — a malformed annotation keeps its announced language — is cosmetic (a background colour), stated in three places, and cannot revoke or mislabel stored history, which stays clean.
- **ARCH-ORDER** — pass, and this is the diff's strongest axis. The `(state, event) → (state, effect)` table is explicit, total, owned by one function, independently restated as the test oracle, and the one piece of cross-event state (`d.lang`) has exactly one clearing rule tied to the transition rather than to an effect. `languageAnswer.ownership` is a second piece of cross-event state with a single mutator (`own`), consistent with its `highlight` by construction. No boolean constellation.
- **ARCH-FUNERAL** — pass. One new committed fixture (25 KB, re-record command documented), one gate ledger under `workshop/plans/` that archives with the issue. No new growing family, no writer whose per-event growth increased.

## 7. Plan revision recommendations

The Plan now matches the code; round 1's `## Revisions` entry already corrected the row-6 mechanism, the retention figure and the Done-when enumeration. One addition:

- **`## Revisions` — Done-when line 8, the enumeration was incomplete.** It names `atlas/define.md`, `cmd/define/README.md` and `stepLanguageDecode`'s doc comment as the consumers of the replaced rule. A fourth consumer is `sharedLanguageGrammar` (`cmd/define/askctx.go:173-180`), whose "Keep passages below 4000 characters" is the prompt-side spelling of `languageBodyLimit` — same origin commit, 62c6a66 / #65. It is an executable contract, not prose, which is why an enumeration of *documents* missed it. State the rule (a bound has one statement; a prompt is a site) and record the two-member sweep.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Mutation-verified: a buffer-then-dribble decoder reds both widths at ask_language_test.go:207 ("nothing reached the screen until delta 102 of 161"); production chain via deltaObserver, no clock.
  - id: BR-2
    disposition: addressed
    note: |
      The terminal subtest runs at width 100; 1742 bytes in 17 writes, largest 135, so the row-commit path is on the tested path.
  - id: BR-3
    disposition: addressed
    note: |
      splitWordInsideAPassage takes the wanted lang and matches it against the region; residual helper/parser divergence raised separately at 4.1.
  - id: BR-4
    disposition: addressed
    note: |
      One spanAccumulator and one decodeChunks with a per-chunk hook; the conformance recorder now uses the shared one. Duplicated doc comment noted at 4.2.
  - id: BR-5
    disposition: addressed
    note: |
      dominantPassage/longestPassage stated once over decoded text, used by both the promotion guard and the replay guard.
  - id: BR-6
    disposition: addressed
    note: |
      A region now ends at a nested open; the specific defect (region body containing marker bytes) is gone, verified against the nested form in stream-long-passage.sse.
  - id: BR-7
    disposition: addressed
    note: |
      The method is runVocabulary, with a comment explaining why it is not vocabularyFor.
findings:
  - id: new
    severity: Important
    family: one-rule-two-statements
    title: |
      the shadow-sweep enumerated prose consumers and missed the executable one: the ask prompt still restates the deleted body bound
    detail: |
      3rd finding in this family, so per the repeat protocol the deliverable is the RULE, not this site.
      Rule - a quantitative bound in this subsystem has exactly one statement, and every other site derives
      from it or cites it by name; a prompt is a site. Measured prevalence, five bounds in the subsystem -
      header candidate (single-sourced), decoder retention (fixed round 1), dominant passage (fixed round 1),
      passage length and entity candidate (both open). Passage length - sharedLanguageGrammar
      (cmd/define/askctx.go:176) still says "Keep passages below 4000 characters", which is 16000 bytes, the
      UTF-8 worst-case fit inside the deleted languageBodyLimit; git log -S puts the clause and the constant
      in the same commit 62c6a66 (issue 65). It now protects nothing, contradicts the atlas line landed in
      this diff ("the only bounds now are the 64-byte header candidate and a stated total retention"), and
      pushes the model toward the fragmented passage shape TestLongPassageStreamsAgainstLiveService refuses.
      Entity candidate - maxLanguageDecoderRetained is 2*languageHeaderLimit + 2*utf8.UTFMax and so now
      depends on the entity cap being languageHeaderLimit, but the entity check at
      cmd/define/language_decode.go:245 is a bare literal 64; this coupling was introduced by this diff. It
      fails loudly via the fuzz invariant rather than silently, so it is the mild member - sweep it in the
      same pass.
  - id: new
    severity: Minor
    family: test-helper-underconstrained
    title: |
      annotatedRegions reports text after a nested open as owned, where the parser has it in recovery
    detail: |
      2nd in this family, so the deliverable is the rule - a test helper that must know where the parser puts
      a boundary derives that boundary from the parser, not from a second scan. cmd/define/askhighlight_test.go:90
      ends a region at a nested open (BR-6's fix) but then rescans FROM that open, emitting the nested passage
      as a region. Measured in a scratch test: "[lang=en]foo[lang=es]bar[/lang] tail" yields regions en "foo"
      and es "bar", while the decoder yields only en "foo" - "bar" is in recovery and owned by nobody. Harmless
      on today's capture (the nested region there is empty) and the failure mode is a false red, not a false
      green. The derivation is available: run the capture through languageDecoder one delta at a time and record
      the decoded length at each delta boundary, which yields both "split across deltas" and "owned by lang" from
      production, collapsing annotatedRegions, the raw-vs-decoded offset mismatch and BR-3's lang parameter into
      one mechanism.
  - id: new
    severity: Minor
    family: duplicated-test-helper
    title: |
      the retention-invariant rationale is duplicated verbatim onto decodeChunks, which does not assert it
    detail: |
      2nd in this family; same one-statement rule as BR-4. The eight-line comment at
      cmd/define/language_decode_test.go:31-40 is character-for-character the one at :138-147, and decodeChunks
      in its decodedChunks form takes a no-op hook and asserts nothing. Delete the copy at :31-40 - the
      invariant's home is decodedChunksBounded. Separately, assertDominantPassage's comment
      (cmd/define/ask_language_test.go:222-225) describes the bug BR-5 fixed in the present tense, inside the fix.
```

---

## Re-review — 2026-09-17T17:07:07-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 72 — define: stream the model's answer to the screen as it arrives |
| repo | tools |
| issue file | workshop/issues/000072-stream-answer-display.md |
| boundary | whole-issue close |
| milestone | — |
| window | 615f88e82f5b067810e36624f2bc46902bdc1ab7..841c14d1a9e82cac90a1f0f303bf3372b307b175 |
| command | sdlc close --issue 72 |
| reviewer | claude |
| timestamp | 2026-09-17T17:07:07-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All three open findings (BR-8, BR-9, BR-10) are genuinely addressed, and I verified the production change with mutation testing rather than taking the Log at its word: restoring the segment buffer in a scratch worktree at HEAD turns `TestOwnedTextIsEmittedBeforeItsCloseMarker`, both subtests of `TestALongPassageReachesTheScreenInPieces` ("nothing reached the screen until delta 102 of 161") and `TestOwnedPassageHighlightsAWordSplitAcrossDeltas` red; restoring the one-shot `highlightRegion` on the owned path turns `TestOwnedPassageHighlightsAWordSplitAcrossDeltas` and `TestLanguageAnswerForeignHomographDoesNotUseTargetVocabulary` red. BR-8's sweep is complete — a tree-wide grep for `4000`/`16 KiB`/`languageBodyLimit`/`d.body`/`decodeLimit` returns only historical comments naming what was deleted, the prompt clause is gone with both goldens re-recorded (`TestRenderAskPrompt`/`TestRenderPassagePrompt` pass and are themselves the wording regression test), and the entity cap now cites `languageHeaderLimit`. `go build ./...` and `go vet` are clean; `go test ./cmd/define` passes except `TestLanguagePromptStartup`/`TestLanguageTintInvocation`, which fail at `pty.Open(): operation not permitted` — an environment restriction in files this window does not touch — and a 45 s fuzz run (507k execs) holds the retention invariant. Nothing blocks; three Minor notes follow, two of them family repeats stated as rules.

### 1. Strengths

- **The fix removes mechanism instead of adding it.** `stepLanguageDecode` loses two effects, one event and its `ok` return and becomes total (`cmd/define/language_decode.go:72`); `d.lang` gets one clearing rule at `:146-152` replacing two flush-effect sites; the owned path stops being a second highlighter (`cmd/define/answer_language.go:68`). The unchanged-but-still-green `TestLanguageAnswerForeignHomographDoesNotUseTargetVocabulary` is the strongest evidence the boundary invariant survived a mechanism swap.
- **BR-9's fix is the right shape, not just a patch.** `splitWordOwnedBy` (`cmd/define/askhighlight_test.go:29`) now runs the capture through the real decoder one delta at a time, yielding "where the deltas fell" and "who owns each byte" from production itself — collapsing `annotatedRegions`, `splitWordMatching`, the raw-vs-decoded offset mismatch and BR-3's language parameter into one mechanism that cannot disagree with the code it tests. 137 lines out, 60 in.
- **The ordering guard is the observable, not a proxy.** `deltaObserver` (`cmd/define/ask_language_test.go:146`) reports each delta after the real writers handled it — no clock, no held connection — and both properties run at width 0 and width 100, putting the wrap writer's row-commit path on the tested path.
- **The envelope's retention bound is now a grammar property, not a field check.** `maxLanguageDecoderRetained` = `2*languageHeaderLimit + 2*utf8.UTFMax` (`language_decode.go:26`) asserted after every chunk by `decodedChunksBounded`, with the constant cited at both grammar sites. I traced both: marker retains ≤ 63 B and entity ≤ 63 B between events, so 136 is tight rather than vacuous.
- **`lessons.md` earns its four entries** — particularly "A double that measures something must implement nothing it did not write" (the embedded `bytes.Buffer` promoting `WriteString`), which is a durable trap, and "A capture is evidence only for the shape it was recorded in".

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

- `cmd/define/answer_language.go:32` — the constructor restates `runVocabulary("")` as a bare `v`, so the `(ownership, highlight)` pair is set once outside `own()`, its only transition function (ARCH-ORDER bypass, ARCH-DRY). 4th in `one-rule-two-statements`; see the rule below.
- `atlas/define.md:2317` — "the 64-byte header candidate" states the bound numerically while naming `maxLanguageDecoderRetained` for the other; same family, same rule.
- `cmd/define/askhighlight_test.go:68` — a production streaming regression surfaces as "re-record the capture or pick another". 3rd in `test-helper-underconstrained`; see the rule below.
- `cmd/define/ask_language_test.go:206` — `firstVisible*4 > deltas` is a bare magic 4 with no stated basis; one clause saying "within the first quarter of the stream" would make it reviewable.
- `cmd/define/README.md:337` — the inserted clause leaves a 96-column line mid-paragraph; cosmetic.
- No durable plan at `workshop/plans/000072-stream-answer-display-plan.md` for a 15-file / 1459-line change (AGENTS.md §1). The issue's `## Plan` is complete and the Estimate section flags this as a known judgment call, so I record it rather than ask for a move at close.

### 5. Test coverage notes

Mutation-verified above. The enumeration the plan committed to — {neutral, owned} × {split across deltas} — is delivered and both cells fail under the relevant mutation. One uncovered cell I judged not worth a finding: {owned} × {Ctrl-C mid-passage} asserts tint survival (`ask_language_test.go:88`) but not vocabulary highlighting; `TestInterruptedMidStreamKeepsWhatArrivedHighlighted` covers only the neutral half. The paths now share one `highlightWriter` flushed unconditionally in `Finish()`, so the risk is low. Separately, at width 100 the granularity half of `TestALongPassageReachesTheScreenInPieces` would not catch the buffered shape on its own (the wrap writer chops the released passage into ~135-byte rows, under the 400 threshold) — the ordering half is what catches it there, which the test's own comment anticipates.

### 6. Architectural notes

ARCH-DRY pass (the diff is net-subtractive on duplication; two restatement sites noted). ARCH-PURE pass — `stepLanguageDecode` is a pure total function with an independently stated oracle table, and every decoder test runs without IO. ARCH-PURPOSE pass — the shadow-sweep now reaches the executable consumer, and grep confirms no live restatement survives. ARCH-MOCK pass — `llmtest.Fake` is the seam, `deltaObserver` wraps at the client seam rather than beside it, and the new live conformance recorder refuses to promote a non-dominant passage with a documented re-record command. ARCH-CONSTRAINTS mostly pass; the one gap is the 0.5 s first-paint budget (below). ARCH-SECURE pass — untrusted model output goes through a total transition function with a fuzz-asserted retention bound, and `languageCaptureTransport` tees the response body only, never request credentials; the capture write is env-gated behind the `conformance` tag at 0600. ARCH-ORDER pass with the constructor note. ARCH-FUNERAL pass — the diff deletes residue (the segment buffer) and adds one 25 KB committed fixture with a re-record path; no new log, ledger, cache or growing family.

For **#64**: the inherited question is recorded in the Spec, and the tint's fate now composes with a real streaming path — the answer is painted as it arrives, so any #64 decision to withhold the tint until a reply-language policy exists has to be a decision made *at the open*, not a revocation.

### 7. Plan revision recommendations

None required — the Plan's six rows all match the code, including row 6's already-revised `deltaObserver` mechanism. One optional addition if the envelope finding is acted on: a `## Revisions` entry noting that the 0.5 s first-paint basis was measured on the wrapper in isolation and that the composed path's post-change signal at width 100 is delta 13 of 161.

```findings
dispose:
  - id: BR-8
    disposition: addressed
    note: |
      Both sites fixed and the class swept: the "Keep passages below 4000 characters" clause is gone from sharedLanguageGrammar (cmd/define/askctx.go:180) with both goldens re-recorded and TestRenderAskPrompt/TestRenderPassagePrompt green, and the entity cap at cmd/define/language_decode.go:246 now cites languageHeaderLimit; grep for 4000/16 KiB/languageBodyLimit/d.body/decodeLimit returns only historical comments.
  - id: BR-9
    disposition: addressed
    note: |
      annotatedRegions, splitWordMatching and splitWordInsideAPassage are deleted; splitWordOwnedBy (cmd/define/askhighlight_test.go:29) derives both the delta boundaries and the ownership by running the capture through the real languageDecoder one delta at a time, so the helper cannot disagree with the parser. Verified reachable: the owned-path test still finds "phrase" and still goes red when highlightRegion is restored.
  - id: BR-10
    disposition: addressed
    note: |
      The duplicated eight-line retention rationale is deleted from decodeChunks (the invariant's only statement is now decodedChunksBounded, cmd/define/language_decode_test.go:132-140), and assertDominantPassage's comment (cmd/define/ask_language_test.go:227-230) no longer describes the fixed bug in the present tense.
findings:
  - id: new
    severity: Minor
    family: one-rule-two-statements
    title: |
      the run highlighter's vocabulary is derived in own() and restated in the constructor
    detail: |
      This is the 4th finding in family `one-rule-two-statements`. Earlier rounds fixed instances
      (decoder retention, dominant passage, passage length, entity candidate). Do NOT fix these two
      instances alone — state the rule and sweep it.
      Rule: a fact with one authority has exactly one derivation site; every other site calls that
      derivation or names it. Measured prevalence, two open sites. (1) cmd/define/answer_language.go:74
      derives the run's vocabulary as a.runVocabulary(lang); cmd/define/answer_language.go:32 restates
      it as the raw v. They agree today only because runVocabulary("") returns a.vocab unconditionally,
      so if the neutral-run rule ever changes (plausibly in #64) an answer that opens neutral silently
      keeps the old vocabulary for its first run and one that opens inside a passage does not. It is
      also an ARCH-ORDER bypass: own() is the sole transition for the (ownership, highlight) pair and
      the constructor sets it directly. Fix is one line — newHighlightWriter(a, a.runVocabulary(""), knownOn).
      (2) atlas/define.md:2317 says "the only bounds now are the 64-byte header candidate and a stated
      total retention (maxLanguageDecoderRetained)" — it names the constant for one bound and spells
      the number for the other; "the 64-byte header candidate (languageHeaderLimit)" closes it.
  - id: new
    severity: Minor
    family: test-helper-underconstrained
    title: |
      splitWordOwnedBy reports a production streaming regression as "re-record the capture"
    detail: |
      This is the 3rd finding in family `test-helper-underconstrained` (BR-3: the helper could hand back
      a foreign-region word and report a correct implementation as "was not highlighted"; BR-9: the
      helper disagreed with the parser about a boundary). Do NOT fix this instance alone — state the rule.
      Rule: a helper that computes a test's precondition from production code must separate "the fixture
      lacks the case" from "production stopped producing the case", because after BR-9 those two share a
      single derivation and therefore a single failure message. Measured: with the segment buffer restored
      in a scratch worktree at HEAD, cmd/define/askhighlight_test.go:68 fires with "no word in
      stream-language.sse is split across deltas and owned by \"en\" — re-record the capture or pick
      another". The test does go red, so the gate is safe; but the message directs a maintainer to
      re-record a capture that is not the problem. The split is cheap and stays inside BR-9's rule: assert
      first that the decoded capture carries any span owned by lang (a fixture property — still true under
      the buffering mutation, since spans exist, they just arrive at the close), then that a delta boundary
      falls inside such a word (the production property), with a message naming streaming.
  - id: new
    severity: Minor
    family: envelope-guard-unautomated
    title: |
      the 0.5 s first-paint budget has no post-change measurement at width 100 and no guard in its own units
    detail: |
      This is the 3rd finding in family `envelope-guard-unautomated`. Do NOT add a wall-clock assertion —
      the operator already decided at PQ-5 that ordering is the deterministic stand-in and the second
      figure stays a `## Log` fact. State the rule instead.
      Rule: every budget in an `Operating envelope` block names, in the block itself, either the test that
      enforces it or the `## Log` measurement that is its evidence after the change; a budget with neither
      is an assumption, not a bound. Measured prevalence in this envelope: retention -> guarded
      (FuzzLanguageDecoderChunks, verified over 507k execs); piped first-byte -> measured after
      (0.917 s, `## Log`); row cadence -> measured, wrapper untouched; first text visible <= 0.5 s at
      width 100 -> neither. Its stated basis (0.3 s) was measured on the wrap writer in isolation before
      the decoder hold was removed, so it never described the composed path; the only post-change signal
      is TestALongPassageReachesTheScreenInPieces logging "first visible at delta 13 of 161" at width 100
      against delta 2 at width 0, and nothing translates 13 deltas into the envelope's units. The ordering
      guard at cmd/define/ask_language_test.go:206 admits anything up to delta 40, so it does not stand in
      for 0.5 s. Either record the composed width-100 figure in `## Log` beside the piped one, or mark the
      budget as carried by the ordering guard and drop the second-level number.
```
