# Boundary Review — tools#11 (whole-issue close)

| field | value |
|-------|-------|
| issue | 11 — LLM seam: Anthropic client, stateful fake, offline degradation |
| repo | tools |
| issue file | workshop/issues/000011-vocab-llm.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8b60e06b1f1aad127d240647140669bcdd317489..65fa45f3d50f5a8df381fcd28f14b8ef5b7c2dec |
| command | sdlc close --issue 11 |
| reviewer | claude |
| timestamp | 2026-08-23T10:03:37-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The three open Importants are genuinely closed and I verified each by reversion rather than by reading commit messages: deleting the `additionalProperties` arm reddens `TestRequiredWalkCoversEveryNestingKeyword` **and** `FuzzDecode/seed#8` through an oracle written independently of the traversal it checks; the schema-keyword walk is now driven by the generator's vocabulary rather than the shapes prior findings named, which is the fix that finally holds after four instance-fixes. `SkipIfUnreachable` is measured end-to-end: with the proxy up and `DEFINE_LLM_MODEL=claude-opus-9-does-not-exist`, all four drift subtests now **FAIL** where round 8 measured them skipping. `go test ./...` green, `-race` clean, `go vet -tags conformance` clean, live conformance ~40s, `--llm-check` answers PONG against the real proxy with the preamble at 1902 — matching the README example byte for byte. What keeps this off SHIP is two new Importants plus two prior Importants whose *class* was never swept: `define --llm-check` discards the `signal.NotifyContext` that `main()` builds, so Ctrl-C is swallowed for the full 5-minute timeout (measured: survived three SIGINTs over 6s against a hung endpoint); and the live typed-task suite added in the second-to-last commit fails **3 times in 18 runs** on ordinary model variation, asserting judgment its own doc comment disclaims.

## 1. Strengths

- **`internal/llm/task.go:126` — the required-field walk is finally driven by the generator, and it is pinned twice.** The enumeration (`properties`, `items`, `additionalProperties`, with `$ref`/`$defs`/`oneOf`/`anyOf` latent) is written into the doc comment, each arm has a table row, and `TestSchemaKeywordsAreCovered` fails when a schema arrives carrying a keyword the walk does not know. Reversion-verified on both sides — the table test *and* `FuzzDecode/seed#8`, whose oracle `keysAbsentFromInput` (`task_test.go:157`) is a deliberately separate traversal so it can disagree with its subject.
- **`internal/llm/llmtest/reachable.go:45` — the drift guard is measured, not argued.** With a live proxy and a renamed model I reproduced the exact scenario round 8 flagged: it now fails rather than skips. The inline closure that survived the last rewrite is gone (`capture_conformance_test.go:52` is a `mustCall` that Fatalfs), and `reachable_test.go` pins both directions.
- **The issue's Done-when evidence is reproducible.** I re-ran two of the three claims: routing 401/403 to the `ErrRequest` arm reddens `TestClassifyStatus` at both rows, and `go test ./... -list '.*'` returns only `TestMemConformance`/`TestYAMLConformance` — no live-API test runs by default.
- **`internal/llm/llmtest/fake.go:162` — `splitInto` on runes, reversion-verified.** Reverting to `[]byte` reddens `TestTextJoinsMultibyteBlocksWithoutCorruption` while `TestTextJoinsEveryTextBlock` stays green — the ASCII fixture genuinely could not see it, which is the point of the second test.
- **Core concepts table cross-check: clean.** All 11 PURE rows exist at their stated paths, and `render_test.go`, `schema_test.go`, `task_test.go`, `errors_test.go`, `config_test.go` are all in-package with no server, clock or filesystem.

## 2. Critical findings

None.

## 3. Important findings

**N1 — `internal/llm/task_conformance_test.go:88,95` the live typed-task suite fails on ordinary model variation, and asserts the judgment its own doc disclaims.** Measured against the live proxy: **3 failures in 18 runs (~17%)**, in two distinct modes.

```
task_conformance_test.go:97: option[3] = {Word: Why:}: a required field came back empty
```
(the model returned four options where the prompt asks for exactly three, the fourth with empty strings)

```
task_conformance_test.go:88: stem has no blank: "placeholder"
task_conformance_test.go:91: no options returned
```
(the model returned a degenerate stub; `Run[T]` returned it with `err == nil`, correctly — an empty array and an empty string both satisfy JSON Schema `required`)

Three artifacts claim otherwise. The doc comment at `:23` says *"It asserts SHAPE, never the model's judgment"*; `atlas/llm.md:207` repeats it; and the commit that added the file says *"Shape only, never judgment."* `Contains(stem, "___")` is a formatting convention, and non-emptiness is not a shape the layer can guarantee. Two failure messages compound it by misattributing: `:56` says *"the required-field check should have rejected this"* and `:95` says *"Every required field at every depth — the property the walk exists for"*, but `missingRequired` enforces **key presence**, not value non-emptiness — so a reader chasing either message goes hunting in a function that behaved correctly.

**This is the 3rd finding in family `unclassified-failure-mode`.** Do NOT fix these two assertions. The rule, stated at BR-39 as *"a check must distinguish 'dependency unreachable' from 'dependency changed' before reporting either"*, needs its third bucket: **a live check must also distinguish "the dependency changed" from "the dependency behaved normally but differently," and may only assert properties the layer under test guarantees.** The enumeration is mechanically available — for each assertion in the three `-tags conformance` files, name which side guarantees it: the transport/decode layer (assert), or the model (log). `Contains(stem,"___")`, `TrimSpace(o.Word)!=""` and `TrimSpace(got.Reason)!=""` are all model-side and belong in `t.Logf` beside the verdicts that already are. The layer-side properties this test *should* assert and currently does not: `err == nil`, the struct decoded, and that the schema reached the wire. Same sweep covers `mustCall` in `capture_conformance_test.go:52`, which Fatalfs a mid-run 429/529 as drift.

**N2 — `cmd/define/main.go:278` `--llm-check` discards the signal context, so Ctrl-C cannot interrupt it.** `main()` builds `signal.NotifyContext` at `main.go:154` with the comment *"Ctrl-C now cancels the context … This changes the one-shot path too, deliberately"*, and `run()` threads that `ctx` into `defineOnce`/`repl`. The `--llm-check` dispatch drops it — `runLLMCheck(os.Getenv, llm.New, stdout, stderr)` takes no ctx, and `llmcheck.go:35` starts from `context.Background()` with `cfg.Timeout` (5 minutes). Measured against a server that accepts and never answers:

```
SIGINT #1 … SIGINT #2 … SIGINT #3
RESULT: survived 3 SIGINTs over 6s — Ctrl-C is swallowed for the full 5m Timeout
```

Repeated Ctrl-C does not help: `NotifyContext` leaves `signal.Notify` registered until `stop()`, so the default terminate behaviour is never restored and the process is uninterruptible until the deadline.

**This is the 2nd finding in family `diagnostic-ignores-config`.** Do NOT just add a ctx parameter. The rule BR-45 needs, and which covers both: **the diagnostic path must use the inputs the caller already resolved, never reconstruct them.** Prevalence 2, both measured — `llmcheck.go:41` hardcodes `MaxTokens: 2048` instead of `cfg.MaxTokens` (BR-45, still open), and `llmcheck.go:35` reconstructs a context instead of deriving from `run`'s. The enumeration is the argument list of `runLLMCheck` against what `run()` holds: `ctx`, the resolved `cfg`, and the writers. Two of the three are reconstructed.

**BR-27 remains open.** `SlowEvery` was clamped; the class the finding tabulated was not swept. Re-measured on the clean tree: negative `Timeout` → every call returns `llm: unavailable: context deadline exceeded` in 1ms, silently absorbed rather than reported as misconfiguration; negative `MaxTokens` → reaches the wire as `max_tokens: -5` with `err=nil`, and `TestNegativeConfigValuesDoNotPanic` (`anthropic_test.go`) pins that pass-through as *correct* by asserting `err == nil` for `MaxTokens: -5`. Two of the finding's three rows survive, and the rule was explicitly "every `Config` field, every value it can hold."

**BR-28 remains open — the sweep covered one path's fields, not the struct's.** The rule said *"the sweep covers the `Reply` struct field by field rather than the one branch a finding named."* The stream path was swept field by field; the JSON path was not swept at all. Measured, all five on the clean tree:

| scripted | request path | result |
|---|---|---|
| `Reply{Stall:true}` | `Complete` | `err=nil`, `stop="end_turn"`, empty text — silently ignored |
| `Reply{StallEarly:true}` | `Complete` | same |
| `Reply{JunkFrame:true}` | `Complete` | same |
| `Reply{Capture:"stream-sample.sse"}` | `Complete` | SSE bytes served as `application/json` → `llm: unavailable: error parsing response json` |
| `Reply{SplitText:3}` | `Stream` | silently ignored, streams the capture |

The fourth is the literal mirror of the case the fix closed (`.json` on a streaming request → 400 naming the field), and it is worse than the original: it arrives wearing `ErrUnavailable`, the class every consumer absorbs silently.

## 4. Minor findings

- **BR-43** — re-verified by reversion: with sticky reverted, `TestTheLastScriptedReplyIsSticky` reddens while `TestAQueueStillAdvancesWhileItHasEntries` (`fake_test.go:166`) stays green alongside `TestQueueServesInOrder`.
- **BR-44** — re-measured: `define -llm-check hello` runs the check and discards `hello`, exit 0.
- **BR-45** — `llmcheck.go:41` still `MaxTokens: 2048`. Folded into N2's class.
- **BR-46** — `schema.go:100` still unconditional; `SchemaFor[string]()` still returns `additionalProperties:false` on a string schema.
- **BR-53** — re-measured: `go test ./internal/llm/llmtest -update -run TestGoldenDetectsAChangedPrompt` still FAILS ("a changed prompt passed its golden").
- **BR-56** — ran the enumeration: `ErrorForStop` still has 0 non-test refs (`errors.go:92`), and `_ = c` is still at `cassette_test.go:242` guarding an unused `llm.New` at `:231`.
- **BR-61** — now measurably worse than "they agree today": I changed `params()` to send `claude-sonnet-5` while `effective()` hashed the config's model, and the **entire suite stayed green** — the cassette tests, `TestRequestInContextCarriesTheEffectiveModel` and the field-coverage table all pass with the wire body and the cassette key describing different requests.
- `atlas/llm.md:200` says "**Two** tagged suites" above a list of three; the third bullet was inserted by `e4c0364` without touching the count. Its "Asserts shape, never the model's judgment" is the claim N1 falsifies.
- `README.md:115` is 120 characters in a file otherwise wrapped near 80 (base had no line over 102) — an unwrapped edit artifact in the exit-code paragraph, flagged in two prior rounds and still there.
- `internal/llm/llmtest/cassette.go:128-151` — the *record* path still returns raw IO/marshal errors from `RoundTrip`, which classify as `ErrUnavailable`. BR-52 removed the SSE instance; its clause "return a harness error not in the dependency's absorbable class" was applied to the read path only.

## 5. Test coverage notes

Reversion-verified green→red this round, each mutation the honest absence of the fix: the `additionalProperties` arm (table test **and** fuzz seed), `splitInto`'s rune conversion, the 401/403 arm of `classifyStatus`, and `New`'s preservation of a negative `StallAfter`. Verified *not* red where it should not be: the vacuous `TestNegativeStallAfterDisablesTheBound` is gone from the tree entirely (BR-29), leaving only `TestNewPreservesADisabledStallBound`, which does redden. `go test ./...` green; `-race` clean across all four packages; `go vet -tags conformance ./...` clean; tree clean at `65fa45f` after every probe.

Two gaps remain, both measured rather than inferred. The first is BR-61's: a deliberate divergence between the wire body and the hashed effective request is invisible to the whole suite, so the coupling the cassette design rests on is asserted by documentation and by two tests that operate on a hand-populated struct rather than through `Run[T]`. The second is BR-28's: the fake's JSON path accepts five `Reply` fields it cannot serve and says nothing, so a consumer test scripted that way asserts against a response it never asked for — and `Cassettes`/`Transport` still has zero committed artifacts, so replay is only ever exercised against files the same test wrote seconds earlier. That is defensible while no consumer exists, but the first real recording will also be the first real test of that path.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag.** `effective()` (`anthropic.go:70`) and `params()` (`anthropic.go:77`) are two independent defaulting implementations of `Model`/`Effort`/`MaxTokens`, and I have now measured that a divergence between them is caught by nothing. The consolidation is one line — `r = a.effective(r)` at the top of `Complete`/`Stream`, with `params()`'s `cmp.Or` calls deleted — after which the wire body and `RenderRequest` agree by construction rather than by coincidence (BR-61).
- **ARCH-PURE — pass.** No business logic sits inside IO in this window. Every PURE row in the plan's Core concepts table is deterministic and tested with no server, clock or filesystem; `runLLMCheck` is a thin shell over injected `getenv`/`newClient` (its one flaw, N2, is a missing *input*, not misplaced logic); the cassette and reachability seams are injected (`Config.Transport`, an explicit `baseURL`).
- **ARCH-PURPOSE — flag, and it is the same shape the ledger has reported for five rounds.** Two open Importants were answered at the site the finding named while the class it named stayed open, *after the finding text spelled the enumeration out*: BR-27 tabulated three `Config` fields and one was fixed; BR-28 said "field by field" and one path's fields were done. The corrective is mechanical rather than attitudinal — when a finding hands you an enumeration, write it into the issue `## Log` as a checklist before any code moves, and paste the measurement per row. On the issue's own purpose the sweep is clean: the `SchemaFor[T]` single source now has four consumers (wire schema, `RequestHash`, the committed golden, `decode`'s required check) and all four derive.
- **ARCH-MOCK — pass on placement, flag on fidelity.** The fake at the wire, the cassette beneath `Config.Transport`, the taxonomy surviving replay by status, streaming recordable, and a reachability probe tested in both directions is a strong seam — better than the plan asked for, and the drift check now fails on the drift it exists to catch. Two fidelity gaps: the fake silently ignores half its `Reply` surface on the JSON path (BR-28), and the live typed-task check cries wolf at ~17% (N1). Fix N1 before `#10` starts trusting these suites — a check that fails one run in six is a check that gets `-run`-excluded, and then it protects nothing.
- **For the consumers next up:** `Task[T]`/`Run[T]` is stable and I would not change the surface. One contract line is worth writing into the doc before it has five callers, because each consumer will otherwise answer it privately: **`decode` guarantees presence, not non-emptiness** — `{"word":"","why":""}` and `"options":[]` are valid answers that `Run[T]` returns with a nil error, and validating them is the consumer's job. N1 is the first place that ambiguity has already cost something.

## 7. Plan revision recommendations

Append one `## Revisions` entry to `workshop/plans/000011-vocab-llm-plan.md` — the last heading is *2026-08-23 — M2 rounds 6–7*, and two rounds of work have landed since:

- **Round 8 and the post-boundary commit are undeclared.** `missingRequired` is now driven by the generator's nesting vocabulary with `TestSchemaKeywordsAreCovered` as its backstop, and `FuzzDecode` ranges over four result shapes with an independent oracle. Neither appears in Task 10 or in any Revisions entry.
- **`internal/llm/task_conformance_test.go` / `TestTypedTaskAgainstTheLiveService` appears nowhere in the plan** — no Task, no Revisions entry, no Integration points row — although it is the file the issue's close leans on for "the layer consumers write against is proven."
- **Correct plan:2138.** It states "`llmtest.SkipIfUnreachable` is new exported surface (added to the Integration points table above)"; the table at plan:152 has eight rows and neither `SkipIfUnreachable` nor `llm.RequestFromContext` is among them (BR-60). Write the rows first and the sentence second.
- **Task 10 / Core concepts wording.** State that the required check enforces **presence**, not non-emptiness — the distinction N1 turned into a flaky live gate.

```findings
dispose:
  - id: BR-27
    disposition: not-addressed
    note: |
      SlowEvery clamped; the class not swept — negative Timeout still returns ErrUnavailable in 1ms, negative MaxTokens still reaches the wire as -5 with err=nil.
  - id: BR-28
    disposition: not-addressed
    note: |
      Stream path swept field by field; JSON path not swept at all — Stall/StallEarly/JunkFrame silently ignored on Complete, and an .sse capture on Complete surfaces as ErrUnavailable.
  - id: BR-29
    disposition: addressed
    note: |
      The vacuous test is gone from the tree; only TestNewPreservesADisabledStallBound remains, and it reddens when New stops preserving a negative StallAfter.
  - id: BR-30
    disposition: addressed
    note: |
      Reversion-verified — reverting splitInto to []byte reddens TestTextJoinsMultibyteBlocksWithoutCorruption while the ASCII test stays green.
  - id: BR-31
    disposition: addressed
    note: |
      plan:518-520 now declares two phases and no Bytes, matching llm.go.
  - id: BR-32
    disposition: addressed
    note: |
      Reply.Body and Reply.NoThinking are both gone from the struct and from every branch that read them.
  - id: BR-39
    disposition: addressed
    note: |
      Measured live — with the proxy up and a renamed model, all four drift subtests now FAIL where round 8 measured them skipping. No ErrUnavailable skip remains in any conformance file.
  - id: BR-43
    disposition: not-addressed
    note: |
      Re-verified by reversion — with sticky reverted, TestAQueueStillAdvancesWhileItHasEntries (fake_test.go:166) stays green while TestTheLastScriptedReplyIsSticky reddens.
  - id: BR-44
    disposition: not-addressed
    note: |
      Re-measured — `define -llm-check hello` runs the check and discards the word, exit 0.
  - id: BR-45
    disposition: not-addressed
    note: |
      llmcheck.go:41 still hardcodes MaxTokens 2048. Folded into the diagnostic-ignores-config class finding this round.
  - id: BR-46
    disposition: not-addressed
    note: |
      schema.go:100 still unconditional; SchemaFor[string]() still returns additionalProperties:false on a string schema.
  - id: BR-53
    disposition: not-addressed
    note: |
      Re-measured — `go test ./internal/llm/llmtest -update -run TestGoldenDetectsAChangedPrompt` still fails; cassette_test.go:32 restores the literal false.
  - id: BR-56
    disposition: not-addressed
    note: |
      Ran the enumeration — ErrorForStop still 0 non-test refs (errors.go:92); `_ = c` still at cassette_test.go:242 guarding an unused llm.New at :231.
  - id: BR-57
    disposition: not-addressed
    note: |
      Table at plan:152 still lacks SkipIfUnreachable and RequestFromContext, and the new task_conformance_test.go surface is undeclared in the plan too.
  - id: BR-59
    disposition: addressed
    note: |
      Reversion-verified twice — deleting the additionalProperties arm reddens TestRequiredWalkCoversEveryNestingKeyword AND FuzzDecode/seed#8 via an independent oracle.
  - id: BR-60
    disposition: not-addressed
    note: |
      plan:2138 still claims SkipIfUnreachable was "added to the Integration points table above"; the eight-row table at plan:152 does not contain it.
  - id: BR-61
    disposition: not-addressed
    note: |
      Now measured stronger — making params() send a different model than effective() hashes leaves the ENTIRE suite green.
findings:
  - id: new
    severity: Important
    family: unclassified-failure-mode
    title: |
      The live typed-task suite fails on ordinary model variation, asserting judgment its own doc comment disclaims
    detail: |
      Measured against the live proxy: 3 failures in 18 runs (~17%), two distinct modes.
      "task_conformance_test.go:97: option[3] = {Word: Why:}: a required field came back
      empty" (the model returned four options where the prompt asks for three, the fourth
      with empty strings) and "task_conformance_test.go:88: stem has no blank:
      \"placeholder\"" with "no options returned" (a degenerate stub). In both, Run[T]
      returned err=nil correctly — an empty array and an empty string both satisfy JSON
      Schema required. Three artifacts claim the test asserts shape only: the doc comment
      at :23, atlas/llm.md:207, and the commit message of e4c0364. Contains(stem,"___")
      is a formatting convention and non-emptiness is not a shape the layer guarantees.
      Two messages compound it by misattributing to the walk: :56 says "the required-field
      check should have rejected this" and :95 says "Every required field at every depth —
      the property the walk exists for", but missingRequired enforces key PRESENCE, not
      value non-emptiness, so both send the next reader hunting in a function that behaved
      correctly.
      THIS IS THE 3RD FINDING IN FAMILY `unclassified-failure-mode`. Do not fix these two
      assertions. The RULE, stated at BR-39 as "a check must distinguish dependency
      unreachable from dependency changed before reporting either", needs its third
      bucket: a live check must ALSO distinguish "the dependency changed" from "the
      dependency behaved normally but differently", and may only assert properties the
      layer under test guarantees. THE ENUMERATION, mechanically available: for each
      assertion in the three -tags conformance files, name which side guarantees it —
      the transport/decode layer (assert) or the model (log). Contains(stem,"___"),
      TrimSpace(o.Word)!="" and TrimSpace(got.Reason)!="" are all model-side and belong
      beside the verdicts already in t.Logf; the layer-side properties this test should
      assert and does not are err==nil, that the struct decoded, and that the schema
      reached the wire. The same sweep covers mustCall at capture_conformance_test.go:52,
      which Fatalfs a mid-run 429/529 as drift.
  - id: new
    severity: Important
    family: diagnostic-ignores-config
    title: |
      define --llm-check discards run's signal context, so Ctrl-C is swallowed for the full five-minute timeout
    detail: |
      main.go:154 builds signal.NotifyContext with the comment "Ctrl-C now cancels the
      context ... This changes the one-shot path too, deliberately", and run() threads
      that ctx into defineOnce and repl. main.go:278 drops it —
      runLLMCheck(os.Getenv, llm.New, stdout, stderr) takes no ctx — and llmcheck.go:35
      starts from context.Background() with cfg.Timeout, five minutes. Measured against a
      server that accepts and never answers: the process survived three SIGINTs over six
      seconds. Repeated Ctrl-C does not help, because NotifyContext leaves signal.Notify
      registered until stop(), so the default terminate behaviour is never restored and
      the process is uninterruptible until the deadline.
      THIS IS THE 2ND FINDING IN FAMILY `diagnostic-ignores-config`. Do not just add a ctx
      parameter. The RULE that covers both: the diagnostic path must use the inputs the
      caller already resolved, never reconstruct them. Measured prevalence 2 —
      llmcheck.go:41 hardcodes MaxTokens 2048 instead of cfg.MaxTokens (BR-45, still
      open), and llmcheck.go:35 reconstructs a context instead of deriving from run's.
      THE ENUMERATION is runLLMCheck's argument list against what run() already holds:
      ctx, the resolved cfg, and the writers — two of the three are reconstructed.
  - id: new
    severity: Minor
    family: docs-claim-absent-surface
    title: |
      atlas/llm.md says "Two tagged suites" above a list of three, and the third bullet's claim is measurably false
    detail: |
      atlas/llm.md:200 reads "Two tagged suites, both on-demand" and is followed by three
      bullets; e4c0364 inserted TestTypedTaskAgainstTheLiveService without touching the
      count. The same inserted bullet asserts "Asserts shape, never the model's judgment",
      which the ~17% live failure rate falsifies.
      THIS IS THE 6TH FINDING IN FAMILY `docs-claim-absent-surface`. Do not just change
      "Two" to "Three". The RULE, unchanged since BR-34: a doc claim of universal or
      COUNTING form is a claim about an enumeration, so it may only be written after
      running that enumeration — and inserting a member into a list is an edit to every
      counting sentence that governs the list. The cheap enforcement is to grep the
      enclosing section for a cardinal before adding a bullet, and to re-read the sentence
      you are inserting under, not only the ones you wrote.
```

---

## Re-review — 2026-08-23T10:24:15-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 11 — LLM seam: Anthropic client, stateful fake, offline degradation |
| repo | tools |
| issue file | workshop/issues/000011-vocab-llm.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8b60e06b1f1aad127d240647140669bcdd317489..e9a36b3673350bd962626514ab125198e51321bf |
| command | sdlc close --issue 11 |
| reviewer | claude |
| timestamp | 2026-08-23T10:24:15-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The Ctrl-C fix is real and I verified it by reversion, not by reading the commit message: restoring `context.Background()` in `llmcheck.go:39` reddens `TestLLMCheckHonoursCancellation` in 20s with the sentence that names the cause, and the test bounds the call itself rather than leaning on the package timeout — the lesson filed alongside it is honoured in the same commit. `go test ./...` green, `go vet -tags conformance` clean, tree clean at `e9a36b3` after every probe. What keeps this off SHIP is that the round's own record contains a claim I could not reproduce, and that its one behavioural fix introduced a new silent-failure path in the surface whose entire purpose is to be loud. The commit message states BR-27 and BR-28 were "verified fixed by reproducing each finding's OWN case" — but round 9 disposed both `not-addressed` on the *residual*, and no code outside `cmd/define` and `task_conformance_test.go` moved this window. I re-measured the residuals on the clean tree: negative `Timeout` still returns `ErrUnavailable` in 940µs, negative `MaxTokens` still reaches the wire as `-5` with `err=nil`, and the fake still ignores `Stall`/`StallEarly`/`JunkFrame` on `Complete` while an `.sse` capture on `Complete` surfaces as `ErrUnavailable`. And BR-62 is not just open but slightly worse: **3 failures in 15 live runs (20%)**, two of them on the `stem is empty` / `answer is empty` assertions this round *added*.

## 1. Strengths

- **`cmd/define/llmcheck.go:39` — the Ctrl-C fix is genuinely pinned, and the test names its own cause.** Reverting to `context.Background()` produces `runLLMCheck ignored a cancelled context; Ctrl-C would be swallowed for the whole Timeout` in 20s. The `select` on a local timer instead of the 120s package timeout is exactly the correction `workshop/lessons.md` files, applied in the commit that filed it.
- **Core concepts cross-check is clean.** All 14 PURE rows and all 8 INTEGRATION rows exist at their stated paths. `render_test.go`, `schema_test.go`, `task_test.go`, `config_test.go` import nothing IO-shaped; `errors_test.go`'s `net/http` is status constants and `response_test.go`'s `os` reads a committed capture. No table/code contradiction.
- **`internal/llm/task.go:126` — the required-field walk is driven by the generator's vocabulary and pinned twice**, by `TestSchemaKeywordsAreCovered` and by `FuzzDecode` through an oracle (`keysAbsentFromInput`) written as a deliberately separate traversal so it can disagree with its subject.
- **The docs gate passes.** `README.md` has a "Checking the model connection" section and names `--llm-check` in the exit-code paragraph; `atlas/llm.md` exists and `atlas/index.md:13` links it. No new surface is undocumented.
- **The issue's Done-when evidence is reproducible** — I re-ran two rows: `go test ./... -list '.*'` returns only the store's in-memory conformance tests, and every live-client file carries `//go:build conformance`.

## 2. Critical findings

None.

## 3. Important findings

**N1 — `cmd/define/llmcheck.go:49` the guard added this round silences the diagnostic's most likely failure.** The new `if ctx.Err() != nil { return 1 }` cannot distinguish a user interrupt from the deadline expiring, because `ctx` was shadowed four lines earlier by `context.WithTimeout(ctx, cfg.Timeout)` — both cases give a non-nil `Err()`. Measured, two orderings:

```
SDK bound fires first  -> stderr = "define: llm check failed after 2.002s: llm: unavailable: context deadline exceeded"
ctx  bound fires first -> stderr = ""            exit=1
```

Production has the second ordering by construction: both bounds are `cfg.Timeout`, and the context is created at `:39` while the SDK's `WithRequestTimeout` starts when the request is issued at `:42`, so the context deadline is strictly earlier and always wins. A hung proxy therefore prints three config lines and then nothing. Three artifacts promise otherwise — `README.md:108` ("it exits non-zero and **names the reason**"), `atlas/llm.md:186` ("Non-zero and specific when unavailable — the one surface where the seam is LOUD"), and the function's own doc comment. `TestLLMCheckIsNonZeroAndSpecificWhenUnavailable` covers no-key and a closed port, neither of which reaches this branch, and `TestLLMCheckHonoursCancellation` actively pins the silence via `!strings.Contains(errOut, "llm check failed")`. **This is the 4th finding in family `unclassified-failure-mode`** — do not add a second `errors.Is` check here. The rule, stated at BR-39 and extended at BR-62, needs its narrowest form yet: **a guard that suppresses reporting must be narrower than the set of failures it can fire on, and must be tested on the failure it is *not* meant to suppress.** The enumeration is the two things a non-nil `ctx.Err()` can mean here; capture the parent's `Err()` before deriving the timeout so `context.Canceled` from the signal context is distinguishable from `context.DeadlineExceeded` from your own bound, and add the timeout row to the "non-zero and specific" table.

**BR-27 remains open** — the class was never swept. Measured on the clean tree: `Timeout: -1` → `llm: unavailable: context deadline exceeded` in 940µs, absorbed as an outage rather than reported as misconfiguration; `MaxTokens: -5` → reaches the wire as `max_tokens: -5` with `err=nil`. Only `SlowEvery` is clamped (`anthropic.go:42`); `Timeout` and `MaxTokens` use `cmp.Or`, which replaces zero only. `TestNegativeConfigValuesDoNotPanic` pins the `MaxTokens: -5` pass-through as *correct* by asserting `err == nil`. The finding's own table listed all three rows.

**BR-28 remains open** — the stream path was swept field by field, the JSON path not at all. Measured, all on `Complete`:

| scripted | result |
|---|---|
| `Reply{Stall}` / `Reply{StallEarly}` / `Reply{JunkFrame}` | `err=nil`, `stop="end_turn"`, empty text — silently ignored |
| `Reply{Capture:"stream-sample.sse"}` | SSE served as `application/json` → `llm: unavailable: error parsing response json` |
| `Reply{SplitText:3}` on `Stream` | silently ignored |

The fourth row is the exact mirror of the case `fake.go:406` closed (`.json` on a streaming request → 400 naming the field), and it is worse: it arrives wearing `ErrUnavailable`, the class every consumer absorbs silently.

**BR-62 remains open, and this round's change made the class larger.** The finding named three model-side assertions and said "Do not fix these two assertions." One (`Contains(stem,"___")`) was removed; two new ones of the same class were added (`TrimSpace(got.Stem)==""` at `:94`, `TrimSpace(got.Answer)==""` at `:97`); `TrimSpace(o.Word)!=""`/`TrimSpace(o.Why)!=""` at `:106` and `TrimSpace(got.Reason)!=""` at `:56` are untouched, as is `mustCall` at `capture_conformance_test.go:49`, which still `Fatalf`s a mid-run 429/529 as drift. Measured live against the proxy, 15 runs:

```
RUN  3: option[0] = {Word:sycophantic Why:}: a required field came back empty   (stem "placeholder")
RUN  7: stem is empty / answer is empty / no options returned
RUN 13: no options returned
TYPED-TASK LIVE: 12 pass / 3 fail out of 15
```

20%, against round 9's 17%. Note run 3: the degenerate `"placeholder"` stub the removed assertion used to catch is *still* a failure, now via a different line — and runs 7's two failures are on assertions that did not exist before this commit. The misattributing messages the finding quoted survive verbatim at `:56` ("the required-field check should have rejected this") and `:105` ("Every required field at every depth — the property the walk exists for"), both pointing a reader at `missingRequired`, which enforces key **presence** and behaved correctly.

**BR-63 — the ctx half is fixed and reversion-verified; the enumeration it wrote is 1 of 2.** The finding said "Do not just add a ctx parameter" and named `llmcheck.go:41`'s hardcoded `MaxTokens: 2048` as the second member of the same class. `MaxTokens: 2048` is unchanged at `:46`. Reporting this `addressed` would let the class close while half of its own enumeration is open.

## 4. Minor findings

- **BR-43** — re-verified by reversion: with sticky reverted, `TestTheLastScriptedReplyIsSticky` reddens while `TestAQueueStillAdvancesWhileItHasEntries` (`fake_test.go:166`) stays green beside `TestQueueServesInOrder` (`:82`). Same script shape, same two assertions.
- **BR-44** — re-measured: `define -llm-check hello` runs the check and discards `hello`, exit 1 (not the usage 2 that `--forget` gives).
- **BR-45** — `llmcheck.go:46` still `MaxTokens: 2048`. Folded into N1/BR-63's class.
- **BR-46** — `schema.go:100` still unconditional.
- **BR-53** — re-measured: `go test ./internal/llm/llmtest -update -run TestGoldenDetectsAChangedPrompt` still FAILS ("a changed prompt passed its golden").
- **BR-56** — ran the enumeration: `ErrorForStop` (`errors.go:92`) has **0** references tree-wide including tests; `_ = c` still at `cassette_test.go:242` guarding an unused `llm.New` at `:231`.
- **BR-57 / BR-60** — the plan's last `## Revisions` heading is *M2 rounds 6–7*; rounds 8, 9 and 10 are undeclared, and `task_conformance_test.go` / `TestTypedTaskAgainstTheLiveService` appears nowhere in the plan (`grep` returns zero hits). `plan:2138` still claims `SkipIfUnreachable` was "added to the Integration points table above"; the table at `plan:152` has eight rows and contains neither it nor `llm.RequestFromContext`.
- **BR-61** — now measured with a cleaner mutation than round 9's: changing only `params()`'s Config fallback to `cmp.Or(r.Model, "claude-sonnet-5")` (so an explicit `Request.Model` still reaches the wire, keeping the unknown-model obligation intact) puts a different model on the wire than `effective()` hashes, and **`go test ./...` stays fully green**.
- **BR-64** — the count is corrected to "Three", but the same bullet still asserts "Asserts shape, never the model's judgment", which the 3/15 measurement falsifies; the identical claim sits at `task_conformance_test.go:23`.
- `README.md:115` is 120 characters in a file otherwise wrapped near 80 (base had no line over 102) — an unwrapped edit artifact, flagged in three prior rounds and never raised.
- `internal/llm/llmtest/cassette.go:128-151` — the *record* path still returns raw IO/marshal errors from `RoundTrip`, which classify as `ErrUnavailable`. BR-52 removed the SSE instance; its clause "return a harness error not in the dependency's absorbable class" was applied to the read path only.

## 5. Test coverage notes

Reversion-verified green→red this round: the `ctx` threading in `runLLMCheck` (20s, named cause). Verified *not* red where it should be: the `params()`/`effective()` divergence (whole suite green), and every BR-27/BR-28 residual (no test touches them). Full suite green; `go vet -tags conformance` clean; tree clean at `e9a36b3` after every probe.

The structural gap is unchanged and is now the one to fix before consumers arrive: the *effective* cassette key has no field-coverage test. `TestEveryMeaningfulFieldReachesTheHash` (`render_test.go:57`) runs over a hand-populated struct, not through `Run[T]`, which is why a deliberate wire-vs-key divergence is invisible to 100% of the suite. Second: the live typed-task suite is the only place `Run[T]` is proven end to end, and at a 20% failure rate it is a check that will be `-run`-excluded rather than trusted — after which it protects nothing. Third: `Cassettes`/`Transport` still has zero committed artifacts, so replay is only ever exercised against files the same test wrote seconds earlier; defensible while no consumer exists, but the first real recording will also be the first real test of that path.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag.** `effective()` (`anthropic.go:70`) and `params()` (`anthropic.go:77`) remain two independent defaulting implementations of `Model`/`Effort`/`MaxTokens`, and I have now measured that a divergence between them is caught by nothing at all. One line — `r = a.effective(r)` at the top of `Complete`/`Stream`, with `params()`'s `cmp.Or` calls deleted — makes the wire body and `RenderRequest` agree by construction (BR-61).
- **ARCH-PURE — pass.** No business logic sits inside IO in this window. Every PURE row is deterministic and tested with no server, clock or network; `runLLMCheck` is a thin shell over injected `getenv`/`newClient`, and N1 is a defect in how it *reports*, not in where its logic lives.
- **ARCH-PURPOSE — flag, and this round is the sharpest instance the ledger has recorded.** Two open Importants were closed in the commit message on the strength of "reproducing each finding's OWN case" — but the finding's own case was the original instance, fixed rounds ago; what round 9 measured and disposed was the residual, which I re-measured unchanged. A third (BR-62) was answered by deleting one named assertion and adding two of the same class, both of which I measured failing live. The corrective is mechanical and has been recommended twice: when a finding hands you an enumeration, write it into the issue `## Log` as a checklist with a measurement pasted per row *before* any code moves — and when re-verifying a `not-addressed` disposition, reproduce **the disposition's** measurement, not the original finding's.
- **ARCH-MOCK — pass on placement, flag on fidelity.** The fake at the wire, the cassette beneath `Config.Transport`, taxonomy surviving replay by status, streaming recordable, and a reachability probe tested in both directions is a genuinely strong seam. Two fidelity gaps: the fake silently ignores five `Reply` fields on the JSON path (one of them surfacing as `ErrUnavailable`), and the live check that proves the layer to its consumers cries wolf once in five runs.
- **For the consumers next up (`#10`, `#12`, `#13`, `#16`, `#17`):** `Task[T]`/`Run[T]` is stable and I would not change the surface. One contract line belongs in the doc before it has five callers, because each consumer will otherwise answer it privately: **`decode` guarantees presence, not non-emptiness** — `{"word":"","why":""}` and `"options":[]` are valid answers that `Run[T]` returns with a nil error. BR-62 is the first place that ambiguity has already cost something.

## 7. Plan revision recommendations

Append one `## Revisions` entry to `workshop/plans/000011-vocab-llm-plan.md` — the last heading is *2026-08-23 — M2 rounds 6–7*, and three rounds have landed since:

- **Rounds 8–10 are undeclared.** `missingRequired` is now driven by the generator's nesting vocabulary with `TestSchemaKeywordsAreCovered` as its backstop; `FuzzDecode` ranges over four result shapes with an independent oracle; `runLLMCheck` takes `run`'s signal context. None of it appears in any Task or Revisions entry.
- **`internal/llm/task_conformance_test.go` / `TestTypedTaskAgainstTheLiveService` appears nowhere in the plan** — no Task, no Revisions entry, no Integration points row — although the issue's close leans on it for "the layer consumers write against is proven rather than argued."
- **Correct `plan:2138`.** Write the `llmtest.SkipIfUnreachable` and `llm.RequestFromContext` rows into the table at `plan:152` *first*, then the sentence that claims they are there (BR-60).
- **Task 10 / Core concepts wording:** state that the required check enforces **presence**, not non-emptiness — the distinction that turned the live gate flaky.

```findings
dispose:
  - id: BR-27
    disposition: not-addressed
    note: |
      Re-measured on the clean tree — negative Timeout returns ErrUnavailable in 940us, negative MaxTokens reaches the wire as -5 with err=nil; only SlowEvery is clamped, and no code moved this window.
  - id: BR-28
    disposition: not-addressed
    note: |
      Re-measured — Stall/StallEarly/JunkFrame silently ignored on Complete, SplitText silently ignored on Stream, and an .sse capture on Complete surfaces as ErrUnavailable; the JSON path is still unswept.
  - id: BR-43
    disposition: not-addressed
    note: |
      Re-verified by reversion — with sticky reverted, TestAQueueStillAdvancesWhileItHasEntries (fake_test.go:166) stays green while TestTheLastScriptedReplyIsSticky reddens.
  - id: BR-44
    disposition: not-addressed
    note: |
      Re-measured against the built binary — `define -llm-check hello` runs the check and discards the word, exit 1.
  - id: BR-45
    disposition: not-addressed
    note: |
      llmcheck.go:46 still hardcodes MaxTokens 2048; it is the second member of BR-63's own two-item enumeration.
  - id: BR-46
    disposition: not-addressed
    note: |
      schema.go:100 still sets additionalProperties:false unconditionally.
  - id: BR-53
    disposition: not-addressed
    note: |
      Re-measured — `go test ./internal/llm/llmtest -update -run TestGoldenDetectsAChangedPrompt` still fails; cassette_test.go:32 restores the literal false.
  - id: BR-56
    disposition: not-addressed
    note: |
      Ran the enumeration — ErrorForStop (errors.go:92) has 0 references tree-wide including tests; `_ = c` still at cassette_test.go:242.
  - id: BR-57
    disposition: not-addressed
    note: |
      Plan's last Revisions heading is still M2 rounds 6-7; rounds 8/9/10 undeclared, and task_conformance_test.go returns zero grep hits in the plan.
  - id: BR-60
    disposition: not-addressed
    note: |
      plan:2138 still claims SkipIfUnreachable was added to the Integration points table; the eight-row table at plan:152 contains neither it nor RequestFromContext.
  - id: BR-61
    disposition: not-addressed
    note: |
      Cleaner mutation than round 9's — diverging only params()'s Config fallback puts a different model on the wire than effective() hashes, and the whole suite stays green.
  - id: BR-62
    disposition: not-addressed
    note: |
      Measured live 3 fail / 15 runs (20%, vs round 9's 17%); one named assertion removed, two of the same class added at :94 and :97, and both fired.
  - id: BR-63
    disposition: not-addressed
    note: |
      The ctx half is fixed and reversion-verified (20s, named cause), but the finding's own two-item enumeration is 1 of 2 — llmcheck.go:46 still hardcodes MaxTokens 2048.
  - id: BR-64
    disposition: not-addressed
    note: |
      The count is corrected, but the same bullet's "Asserts shape, never the model's judgment" is falsified by the 3/15 live measurement, as is the identical claim at task_conformance_test.go:23.
findings:
  - id: new
    severity: Important
    family: unclassified-failure-mode
    title: |
      The guard added this round silences --llm-check on the deadline it is most likely to hit
    detail: |
      cmd/define/llmcheck.go:49 added `if ctx.Err() != nil { return 1 }`, but ctx was
      shadowed at :39 by context.WithTimeout(ctx, cfg.Timeout), so a non-nil Err() means
      either the user's Ctrl-C or the diagnostic's own deadline. Measured both orderings
      against a socket that accepts and never answers: when the SDK's WithRequestTimeout
      fires first, stderr reads "define: llm check failed after 2.002s: llm: unavailable:
      context deadline exceeded"; when the outer context deadline fires first, stderr is
      EMPTY and exit is 1. Production always has the second ordering by construction —
      both bounds are cfg.Timeout, and the context is created at :39 while the SDK's bound
      starts when the request is issued at :42, so the context deadline is strictly
      earlier. A hung proxy therefore prints three config lines and then nothing, from the
      surface README.md:108 calls "the one surface where an unusable configuration is
      loud: it exits non-zero and names the reason" and atlas/llm.md:186 calls "Non-zero
      and specific when unavailable". TestLLMCheckIsNonZeroAndSpecificWhenUnavailable
      covers no-key and a closed port, neither of which reaches this branch, and
      TestLLMCheckHonoursCancellation pins the silence via
      !strings.Contains(errOut, "llm check failed").
      THIS IS THE 4TH FINDING IN FAMILY `unclassified-failure-mode`. Do not add a second
      errors.Is check at this site. The RULE, stated at BR-39 as "a skip condition must be
      narrower than the failure it protects against" and extended at BR-62, needs its
      narrowest form: a guard that suppresses REPORTING must be narrower than the set of
      failures it can fire on, and must be tested against the failure it is not meant to
      suppress. THE ENUMERATION is the two things a non-nil ctx.Err() can mean here —
      capture the parent context's Err() before deriving the timeout so context.Canceled
      from the signal context is distinguishable from context.DeadlineExceeded from your
      own bound, and add a deadline row to the "non-zero and specific" table so the silent
      branch is covered by something.
```
