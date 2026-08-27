# Boundary Review — tools#6 (milestone M1)

| field | value |
|-------|-------|
| issue | 6 — define --play: review loop + form 2.1 quick pass |
| repo | tools |
| issue file | workshop/issues/000006-vocab-play.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 4cac0b95d83d54eab57b8747d40d09a2196bca1e..dd4b6cb8796e2574e6a5fba9b9914653b6fcc0ab |
| command | sdlc milestone-close --issue 6 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-27T08:51:51-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M1 core is genuinely good work: `play` is the purest package in the tree (it imports *nothing*), `Apply` is a clean input-in/state-out machine with every effect named as an `Outcome`, and I mutation-verified in a scratch copy that the skip filter, the reveal gate, reveal idempotence and the form-agnostic property each redden a *named* test — Plan Task 2 Step 7's obligation is really met, not merely asserted. What blocks SHIP is one thing, and it is the thing this gate exists for: the `puretest` extraction is documented — in the issue `## Log` *and* in the project file — as having closed `#5`'s gap that "nothing in the tree proved the guards could fail," and `cmd/define/puretest` has **zero test files**. I gutted `ImportsOnly` and `NoWallClock` to report nothing and both callers' purity suites stayed green. The extraction is real and correct; the claim about its verification is not. Secondary: the plan's Core-concepts table and Task 2/Task 3 steps have drifted from the code M2 will be written against, including the still-open plan-gate finding PQ-3.

### 1. Strengths

- **The purity-guard extraction is the ARCH-DRY exemplar.** `cmd/define/schedule/purity_test.go` goes 193 → 38 lines and becomes three declarative allowlist calls; `eachSourceFile` (`cmd/define/puretest/puretest.go:120`) is shared by `NoWallClock` and `StoreSymbolsOnly` rather than copied. With `#7`/`#12`/`#13` each adding a form package, this was the right moment.
- **`play` imports literally nothing** — `go list` returns an empty import set, and `purity_test.go:21` passes an empty allowlist to match. ARCH-PURE in its strongest form; the caller supplies the already-rendered definition (`recall.go:16`) precisely so the dictionary and `RenderOpts` stay outside.
- **The zero-imports vacuity correction is the right call, well reasoned.** `puretest.go:23-33` explains why "zero imports" is a legitimate answer rather than a failed measurement, and I confirmed the replacement protection holds: `go list` on a nonexistent path exits 1 and the guard fatals, so a typo'd import path still cannot pass silently.
- **`TestSessionIsFormAgnostic` (`session_test.go:158`) is an honest test of an un-testable-looking claim.** I injected a hardcoded `if in.Rune == 'y'` into `Apply` and it reddened at `session_test.go:197` — the Done-when property is pinned, not asserted.
- **`main.Key` stopping at the package boundary** (`session.go:3-11`) is the right resolution of PQ-1, and the atlas records *why* the compile error was the design talking.

### 2. Critical findings

None.

### 3. Important findings

**A. `cmd/define/puretest/puretest.go:1` — the shared guard body has no negative coverage, and two durable artifacts claim it does.**

`go test ./cmd/define/...` reports `? github.com/xianxu/tools/cmd/define/puretest [no test files]`. I verified by mutation in a scratch copy: replacing the `t.Errorf` in `ImportsOnly` (`puretest.go:52`) and in `NoWallClock` (`puretest.go:76`) with no-ops leaves **both** `play` and `schedule` green. The guards do work when the *subject* is impure — I confirmed `TestPlayPurity/imports` fires on an injected `import "os"` and `TestPlayPurity/no_wall_clock` fires on an injected `time.Since(` — but that is exactly the scratch-copy-and-discard verification `#5`'s close recorded as a gap, repeated. Meanwhile `workshop/issues/000006-vocab-play.md:180` states the extraction closed it "with the negative cases verified in the tree," and `workshop/projects/define-learn.md:527` repeats it. Fix sketch: change the guards to take a minimal `reporter` interface (`Helper()`, `Errorf`, `Fatalf`) instead of `*testing.T`, add `cmd/define/puretest/puretest_test.go` driving each guard against a known-bad subject with a recording reporter, and assert it reports. A `testdata/` package that imports `os` and calls `time.Now()` gives a stable subject without depending on another package staying impure. Until that exists, correct the two artifacts in the same round rather than only one (ARCH-PURPOSE: the class is "no claim of verification without the failing test," and it has two instances in this window).

**B. `cmd/define/play/session.go:139-158` — the session has two termination signals and documents only one.**

Answering the final question sets `s.Done = true` inside `advance` but returns `Outcome{Kind: OutcomeRecord}`; a final skip returns `OutcomeNone`. `OutcomeDone` is only ever produced by a *subsequent* `Apply` call. So a loop written as `for { in := <-keys; s, o = Apply(s, in); switch o.Kind { … case OutcomeDone: return } }` blocks on the key channel after the last answer and the session appears to hang; a loop written `for !s.Done { … }` is correct. Nothing in the diff says which — `OutcomeDone`'s doc (`session.go:44`) says "the session is over," and `Session.Done` carries no doc comment at all. M2's Task 4 is the consumer. Fix sketch: either add `Outcome.SessionDone bool` set from the returned state so one switch suffices, or document the contract on `Apply` and pin it with a test asserting that the outcome of the final answer is `OutcomeRecord` *and* `s.Done` is true.

**C. `workshop/plans/000006-vocab-play-plan.md:21,27` — the Core-concepts table and its surrounding prose contradict the code.**

`Verdict` is listed at `cmd/define/play/session.go`; it is declared at `cmd/define/play/question.go:21`. The paragraph under the table states "`play` imports `store`, `schedule` and pure stdlib" and the Test-surface note promises "including the SYMBOL guard, since `play` will import `store` for `ReviewEvent`" — `play` imports nothing, and `purity_test.go:13-19` deliberately omits `StoreSymbolsOnly` with a documented reason. And `puretest`, a whole new package introduced at this boundary that shells out to the `go` binary, has no row in either table. The review checklist defaults a table/code contradiction to Critical; I am not filing it there because every entity exists and is exercised — only the locations and the import claim are stale — but the table no longer enumerates the boundary's surface, which is what makes it useful to `#7`.

**D. `workshop/plans/000006-vocab-play-plan.md:98,122` — the plan still instructs M2 to build the skip rule the code correctly rejects.**

This is the plan-quality ledger's open PQ-3, `signature-cannot-express-contract`, not-addressed across rounds 2–5 and still open at `workshop/plans/000006-vocab-play-plan-gate.md`'s "Open findings". Task 2 Step 1 says "a skip emits a record outcome with `Skipped`"; the code emits `OutcomeNone` (`session.go:152`), which is what the Done-when and the Core-concepts section require. Task 3 Step 1 says "the loop drops it," contradicting "ONE filter, in `Apply`" three sections earlier. The same step also says "the last answer emits quit," which is finding B's behaviour. M1 is where these should be corrected, because M2 is executed from this file. Fix sketch: rewrite both steps to match the implemented contract and dispose PQ-3 `addressed` with the two line references.

### 4. Minor findings

- `cmd/define/play/session_test.go:70` — `TestSkipAdvancesButRecordsNothing` asserts the opposite of its name: the body drives `'s'`, which `Recall` does not grade, and asserts `s.Index != 0` is a failure (it did *not* advance). Rename to `TestUngradedKeyIsIgnored`; the skip contract is already covered by `TestSkippedVerdictRecordsNothing` at line 84.
- `cmd/define/play/recall.go:33` — form 2.1 has no skip key, so `Verdict.Skipped` is unreachable in production until `#7`. Defensible against the Spec ("self-rate right/wrong"), but the issue Log at `000006-vocab-play.md:150` frames skipping as a learner action ("a learner who skips a word they half-know"), so a reader will expect a key. Record the decision in the Log or the plan.
- `cmd/define/play/session_test.go:216` — `skipForm` duplicates capability `fakeForm` already has (`Grade('3')` returns `(Skipped, true)` at line 211); `TestSkippedVerdictRecordsNothing` could drive `fakeForm` and one double could go (ARCH-DRY, test-local).
- `cmd/define/play/question.go:45` — `Grade` overloads `Skipped` as both "the learner skipped" and the zero value returned with `ok == false`. Harmless today because callers check `ok` first, but worth a sentence in the doc comment since `Verdict`'s zero value is `Skipped`.
- `cmd/define/puretest/puretest.go:45,122` — `exec.Command(...).Output()` discards stderr, so a `go list` failure fatals with only the exit status. Capture `exitErr.Stderr` into the message.
- `workshop/plans/000006-vocab-play-plan.md` — every Chunk 1 step is still `- [ ]` although M1 is complete. The issue's `## Plan` M1 row is ticked; the durable plan's steps are the traceability record and were not.

### 5. Test coverage notes

Mutation checks I ran (scratch module under `$TMPDIR`, discarded — nothing in the repo was modified):

| Mutation | Result |
|---|---|
| `advance`'s `if v == Skipped` disabled | `TestSkippedVerdictRecordsNothing` reddens |
| `Apply`'s `!s.Revealed` gate disabled | `TestGradingBeforeRevealIsIgnored` reddens |
| reveal-idempotence branch removed | `TestRevealIsIdempotent` reddens |
| `if in.Rune == 'y'` hardcoded into `Apply` | `TestSessionIsFormAgnostic` reddens |
| `ImportsOnly` + `NoWallClock` error paths gutted | **`play` and `schedule` both stay green** |

The first four are Plan Task 2 Step 7's obligation and it is genuinely satisfied — but no `## Log` entry records that it was run, so the evidence lives only in this review. The fifth is finding A. Uncovered and low-value: `Apply` with an unrecognised `InputKind`, `Current()` with a nil element in `Questions` (it degrades to `OutcomeDone` rather than panicking — fine), and `StoreSymbolsOnly`'s `found == 0` fatal, which no caller can currently reach.

### 6. Architectural notes

- **ARCH-DRY — pass.** The extraction is the finding PQ-6 asked for, delivered on the `storetest.Suite` precedent. Only test-local duplication remains (`skipForm`).
- **ARCH-PURE — pass, with one note.** `play` is the cleanest instance of the principle in this repo. The note is inverted: `puretest` is the IO shell, and it hard-wires *both* its IO (`exec`, `os.ReadDir`) and its reporter (`*testing.T`) — which is the structural reason its own logic cannot be exercised. The reporter seam in finding A is the ARCH-PURE fix as much as the coverage fix.
- **ARCH-PURPOSE — flag (mild).** M1's stated deliverables all landed. The class-vs-instance lens flags two spots: the puretest claim delivers the extraction (instance) while asserting the verification (class) that does not exist; and PQ-3 fixed the prose site while two sibling sites in the same document still instruct the opposite — a family repeating for a fifth round is the ledger reporting the enumeration was never written.
- **ARCH-MOCK — flag (mild).** `puretest` invokes the `go` binary directly with no seam. That is the *correct* call — a faked `go list` would prove nothing about the real build graph, and the toolchain is the measurement. But the boundary should own the cost: the fix is the portable-fixture shape the principle names — a `testdata/` folder of deliberately impure packages plus the reporter seam — so the guards can be driven against known-bad input in-tree without faking the toolchain. `play` itself consumes no external dependency, so nothing else in this window is in scope.

### 7. Plan revision recommendations

A `## Revisions` entry in `workshop/plans/000006-vocab-play-plan.md`, dated, covering:

1. **Core-concepts table** — move `Verdict` to `cmd/define/play/question.go`; add rows for `Input`/`InputKind` and for the `cmd/define/puretest` package (with its kind, since it is neither pure nor a production integration point).
2. **Import claim** — replace "`play` imports `store`, `schedule` and pure stdlib" with "`play` imports nothing," and replace the Test-surface promise of the SYMBOL guard with the reason it is deliberately absent (already written correctly at `purity_test.go:13-19`).
3. **Task 2 Step 1** — delete "a skip emits a record outcome with `Skipped`" (it emits `OutcomeNone`) and "the last answer emits quit" (it emits `OutcomeRecord` with `Session.Done` set); this closes PQ-3's first residue.
4. **Task 3 Step 1** — delete "the loop drops it"; the filter is in `Apply`. This closes PQ-3's second residue, after which PQ-3 can be disposed `addressed`.
5. **Task 2, new step** — the negative-case tests for `puretest` (finding A), so M2 does not inherit an unverified guard that five packages will depend on.
6. **Chunk 1 checkboxes** — tick the delivered steps.

```findings
findings:
  - id: new
    severity: Important
    family: claim-without-failing-test
    title: |
      puretest has no tests, yet the issue Log and project entry both claim its negative cases are verified in the tree
    detail: |
      cmd/define/puretest reports "[no test files]". Gutting the t.Errorf in ImportsOnly (puretest.go:52) and
      NoWallClock (puretest.go:76) leaves both play and schedule green, so the shared guard body five packages
      will depend on has zero negative coverage. workshop/issues/000006-vocab-play.md:180 states the extraction
      closed that gap "with the negative cases verified in the tree" and workshop/projects/define-learn.md:527
      repeats it. Fix: take a reporter interface instead of *testing.T, add puretest_test.go driving each guard
      against a testdata subject that is deliberately impure, and assert it reports. Correct BOTH artifacts, not
      one (ARCH-PURPOSE).
  - id: new
    severity: Important
    family: two-signal-termination
    title: |
      Answering the last question sets Session.Done but returns OutcomeRecord, so the outcome alone never says the session ended
    detail: |
      session.go:139-158 — advance sets s.Done and returns OutcomeRecord (or OutcomeNone for a skip);
      OutcomeDone is only produced by a subsequent Apply. A loop switching solely on Outcome.Kind blocks on
      the key channel after the final answer. Session.Done has no doc comment and OutcomeDone's says only
      "the session is over". M2's runPlay is the consumer. Add Outcome.SessionDone, or document the
      two-signal contract on Apply and pin it with a test.
  - id: new
    severity: Important
    family: plan-table-contradicts-code
    title: |
      The Core concepts table places Verdict in session.go, claims play imports store, and omits the puretest package
    detail: |
      plan:21 lists Verdict at cmd/define/play/session.go; it is declared at question.go:21. plan:27 says
      "play imports store, schedule and pure stdlib" and the Test-surface note promises the store SYMBOL
      guard; play imports nothing and purity_test.go:13-19 deliberately omits StoreSymbolsOnly. puretest, a
      new package introduced at this boundary that shells out to the go binary, has no row in either table.
      Every entity exists and is exercised, which is why this is not filed Critical.
  - id: new
    severity: Important
    family: signature-cannot-express-contract
    title: |
      Plan Task 2 Step 1 and Task 3 Step 1 still instruct the superseded skip contract, leaving PQ-3 open into M2
    detail: |
      plan:98 says "a skip emits a record outcome with Skipped"; session.go:152 emits OutcomeNone, which is
      what the Done-when requires. plan:122 says "the loop drops it", contradicting "ONE filter, in Apply".
      plan:98 also says "the last answer emits quit". The plan-quality ledger lists PQ-3 as open after five
      rounds. M2 is executed from this file, so correct it at this boundary.
  - id: new
    severity: Minor
    family: test-name-contradicts-assertion
    title: |
      TestSkipAdvancesButRecordsNothing drives an ungraded key and asserts the session did NOT advance
    detail: |
      session_test.go:70 — the body drives 's', which Recall does not grade, and fails if s.Index != 0. The
      skip contract is covered at line 84. Rename to TestUngradedKeyIsIgnored.
  - id: new
    severity: Minor
    family: unrecorded-scope-decision
    title: |
      Form 2.1 has no skip key, so Verdict.Skipped is unreachable in production, and no artifact records the decision
    detail: |
      recall.go:33 grades only y/Y/n/N. Defensible against the Spec's "self-rate right/wrong", but the issue
      Log at 000006-vocab-play.md:150 frames skipping as a learner action. Record it in the Log or the plan.
  - id: new
    severity: Minor
    family: duplicated-guard-logic
    title: |
      skipForm duplicates a capability fakeForm already has
    detail: |
      session_test.go:216 — fakeForm.Grade('3') already returns (Skipped, true) at line 211, so
      TestSkippedVerdictRecordsNothing could drive fakeForm and one double could go.
  - id: new
    severity: Minor
    family: overloaded-zero-value
    title: |
      Grade overloads Skipped as both a real verdict and the zero value returned with ok == false
    detail: |
      question.go:45 — callers check ok first so nothing is wrong today, but Verdict's zero value being
      Skipped deserves a sentence in the doc comment.
  - id: new
    severity: Minor
    family: discarded-error-detail
    title: |
      puretest discards go list stderr, so a failed measurement fatals with only an exit status
    detail: |
      puretest.go:45 and :122 use exec.Command(...).Output() and report only err. Capture
      exitErr.Stderr into the fatal message.
  - id: new
    severity: Minor
    family: plan-checkboxes-not-ticked
    title: |
      Every Chunk 1 step in the durable plan is still unticked although M1 is complete
    detail: |
      The issue's Plan M1 row is [x] but workshop/plans/000006-vocab-play-plan.md Chunk 1 Steps 1-9 remain
      "- [ ]". The durable plan is the traceability record.
```
