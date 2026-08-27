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

---

## Re-review — 2026-08-27T12:08:51-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 6 — define --play: review loop + form 2.1 quick pass |
| repo | tools |
| issue file | workshop/issues/000006-vocab-play.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 787824810e8a371c3f2c7c7e7899943d91dfbb39..14e5e97cbe10a104462abcb72005b445889770df |
| command | sdlc milestone-close --issue 6 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-27T12:08:51-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

All inspections completed: stat, name-status, targeted patches, full test run, coverage measurement, and revert-verification of every claimed fix in a scratch copy at HEAD.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1's actual deliverable — `Question`/`Verdict`, `Recall`, the `Session`/`Apply` state machine, the extracted `puretest` guards and the atlas — is correct, pure, and now genuinely pinned. I revert-verified the two headline fixes rather than trusting the commit message: gutting each of `puretest`'s three `t.Errorf` bodies reddens its own named test, and dropping `SessionDone` from `advance`'s record return reddens `TestTheOutcomeThatEndsTheSessionSaysSo`. Eight of ten prior findings are addressed; BR-5 and BR-7 remain open. Nothing blocks the boundary on correctness. What holds it back from SHIP is that BR-4's fix corrected the plan's "three guards" claim and left the identical false claim in three live artifacts (`question.go`, `purity_test.go`, the atlas), and that four of the five code fixes in the sweep commit land in branches measured at coverage 0 — including two that are pinnable today.

**1. Strengths**

- `cmd/define/puretest/puretest_test.go` is the right answer to BR-2, not a plausible-looking one. Taking a minimal `T` interface instead of `*testing.T` is what makes a guard testable at all, and `testdata/clocky` (imports only `time`, calls `time.Since`) is precisely the hazard an import allowlist structurally cannot see. Verified: all three guards redden when gutted.
- The `puretest` extraction is this window's best structural work (ARCH-DRY). `schedule/purity_test.go` is now a 38-line call and `play/purity_test.go` an 8-line one; I grepped the tree and there is no surviving copy of the guard bodies.
- `play` importing **nothing at all** (`purity_test.go:21`, empty allowlist) is the strongest form of the purity claim, and `TestImportsOnlyAcceptsAPurePackage` proves the guard tolerates it after the vacuity-check correction.
- The skip filter genuinely lives in one place (`session.go:170`). Mutating `if v == Skipped` to `if false && …` reddens `TestSkippedVerdictRecordsNothing` and nothing else — one rule, one site, one test.
- `TestSessionIsFormAgnostic` (`session_test.go:176`) is the honest test of the Done-when property before a second form exists, and its second half — 2.1's own `y` meaning nothing to `fakeForm` — is the half most implementations would omit.
- `workshop/lessons.md` gained both rules this round (AGENTS.md §4), and the "backgrounded gate is not a completed gate" entry names the actual failure rather than a sanitised version of it.

**2. Critical findings** — none.

**3. Important findings**

**3.1 — `play` runs TWO purity guards; three live artifacts still say three.** *(2nd in family `plan-table-contradicts-code`.)*
- `cmd/define/play/question.go:7-8` — "Three guards enforce it — an import allowlist, a wall-clock grep, and a store-SYMBOL allowlist"
- `cmd/define/play/purity_test.go:11` — "Same three guards as schedule" — contradicted two lines later by its own "the store guard is deliberately absent"
- `atlas/define.md:1179` — "`#6` needs the same three"

`purity_test.go:19-26` runs `ImportsOnly` and `NoWallClock` only. BR-4 named the plan's copy of this sentence, and the plan was corrected (`plan:56`); the siblings were not. Per the family rule, don't fix the site — the rule is: **a claim about what is enforced is checked against the enforcing code before it is written, and the fix sweeps every artifact stating it, not the one the finding named.** The enumeration is greppable in one command (`grep -rn "three guards\|store-SYMBOL"`), and it returns exactly these three plus `schedule/purity_test.go:11`, where the claim is true. The package doc is the artifact `#7`/`#12`/`#13`'s author reads first, and it overstates enforcement of exactly the guard whose *absence* is the interesting decision (ARCH-PURPOSE).

**3.2 — Four of the five code fixes in `14e5e97` are in branches measured at coverage 0; two are pinnable today.** *(4th in family `claim-without-failing-test`.)*
Measured with `go test ./cmd/define/ -coverprofile`:

| fix | site | coverage |
|---|---|---|
| BR-17 `rawTerm.f` + BR-24 exit 1 | `play_loop.go:146-166` | 0 |
| BR-25 "none could be looked up" + exit 1 | `play_loop.go:234-240` | 0 |
| BR-23 duplicate `withStore` removed | `main.go:416-422` | 0 |
| `f.Stat()` error returned | `store/yaml.go:158` | not practically forceable |

Every `playSession` test passes `rawTerm{}`, so `raw.sess` is nil and the playback re-entry block — which is where both BR-17's and BR-24's fixes live — is never entered by anything. Do not fix these instances. The rule is already written in `lessons.md` ("a claim is complete only when a named test in the tree fails without it"); what is missing is the step that makes it operable: **before closing a round, run coverage over the changed lines and list any fix that lands in a zero-coverage branch, marking each either pinned-this-round or explicitly verified-by-reading in the artifact.** BR-21 asked for that enumeration and got a prose table of `err == nil` sites instead of a coverage sweep, which is why the same family returns. Of the four, BR-25's is pinnable now (a deck whose only due word the fake dictionary lacks — `TestCountBoundsTheSession`'s own comment already identifies that fixture) and BR-23's is pinnable now (a counting `newStore`, which BR-23 itself used to measure the defect).

**4. Minor findings**

- **`probabilistic-regression-test` (2nd in family).** `play_loop_test.go:218` now claims "DETERMINISTIC by construction" and "near-certain rather than occasional". Measured with the pre-select guard removed: **8/24 runs red** with the new four-key buffer, **7/24** with the original single `\ry`. The extra keys change nothing — recording a review still requires two consecutive `keys` wins in the `select` (P≈0.25), independent of buffer depth. The rule: a regression test for a nondeterministic bug removes the nondeterminism (loop the scenario N times inside the test, or extract the stop decision into a pure function and assert it directly) rather than raising the odds; and a comment must not assert a property that a 24-run measurement contradicts.
- **`docs-not-updated-for-new-surface` (3rd in family).** This window's new architectural surface absent from `atlas/define.md`: `Outcome.SessionDone` (new exported field, justified as the affordance downstream forms consume), `puretest.T` and the `testdata/` known-bad fixture convention that `#7`/`#12`/`#13` must follow when adding a guard, and the new all-lookups-fail exit-1 path. The rule: the atlas entry for a surface is edited in the same round the surface changes, and "new surface" includes exported fields and test-fixture conventions, not just user-typed commands.
- **`production-code-used-as-test-fixture` (new family).** `puretest_test.go:57` uses the live `cmd/define/play` package as its known-*good* fixture. If `#7` ever gives `play` a legitimate import, `puretest`'s own suite reddens pointing at the wrong package. `testdata/` already exists; a five-line `testdata/pure` makes the fixture owned and portable (ARCH-MOCK's portable-fixture clause).
- **`commit-not-attributable-to-issue` (new family).** `2a2c837 "wip"` carries 520 lines of substantive M2 work with no issue reference, no body, and no `Co-Authored-By` trailer — invisible to `git log --grep "^#6"`, which AGENTS.md §12 makes the retrieval path.
- `session_test.go:72-73` — the doc comment above the renamed `TestAnUngradedKeyDoesNotAdvance` still describes skip semantics ("`schedule.Fold` would read a recorded skip as a miss"). Fold into BR-6's edit.
- `play_loop.go:189` — `toInput`'s space→`InputReveal` case is at coverage 0; every test reveals with `\r`. One table row.

**5. Test coverage notes**

- `puretest` went from zero tests to seven with committed known-bad fixtures, and I confirmed each of the four assertions is load-bearing by mutation (three guards + `stderrOf`).
- `TestGuardsRefuseToPassVacuously` covers `StoreSymbolsOnly`'s `found == 0` fatal and the `go list` failure path. It does not cover `eachSourceFile`'s `seen == 0` fatal (a package dir with only `_test.go` files) — cheap to add, and it is the vacuity case most likely to bite when a form package is scaffolded test-first.
- `Question.Grade`'s "when `ok` is false the Verdict is ignored" contract (`question.go:49-52`) is only exercised with `(Skipped, false)`. A `fakeForm` key returning `(Correct, false)` would pin that `Apply` really ignores the verdict — the exact hazard BR-9's overloaded zero value creates.
- `runPlay`'s terminal setup (`play_loop.go:38-73`) and the `--play` dispatch (`main.go:416-422`) remain entirely uncovered; BR-13 is unchanged.

**6. Architectural notes for upcoming work**

- **ARCH-DRY — pass, with the flag at 3.1.** `puretest` is the exemplar; the only duplication left in the window is the *claim about* the guards, restated in three places instead of derived.
- **ARCH-PURE — pass.** `play` imports nothing, `Apply` is a total function of `(Session, Input)`, all IO sits in `runPlay`/`playSession`, and the split is machine-enforced rather than asserted. `NewRecall` taking an already-rendered string is the right call and keeps `RenderOpts` out of the pure core.
- **ARCH-PURPOSE — flag (3.1, 3.2).** Both new Importants are the same shape: a prior finding named one instance, the instance was fixed, and enumerable siblings survived. The shadow-sweep on `puretest` as single source passes (both consumers derive); the shadow-sweep on the *reserved-keys* fact is weaker — `toInput` is the source and `question.go:54`, `plan:22` and `atlas:1216` are three hand-maintained restatements with nothing that reddens when the reserved set changes. Prose is the right medium here, but a table-driven `toInput` test naming the reserved set would give the restatements something to derive from before `#7` adds a form.
- **ARCH-MOCK — pass, with the note at 4.3.** `puretest` shelling out to the real `go list` is correct: a fake `go` would defeat the measurement, and the committed `testdata/` packages *are* the stateful fixture behind the seam. The gap is that its known-good fixture is a moving production package rather than an owned one.
- For `#7`: `Outcome.SessionDone` has **zero production consumers** today — `playSession` still loops on `!s.Done`. That is defensible as a published contract for form authors, but it means the field's only proof of usefulness is its own tests. When `#7` lands, either the loop switches to reading `out.SessionDone` or the field should be reconsidered.

**7. Plan revision recommendations**

- **`plan:132`** — "the loop drops it" is the last surviving sentence of the superseded skip contract and contradicts both `plan:76` ("ONE filter, in `Apply`") and `session.go:170-176`. Replace with "`Apply` emits no record outcome for it". This is why BR-5 is disposed `not-addressed`.
- **`plan:56`** is now correct about the two guards; add a sentence naming the `testdata/` known-bad-fixture obligation so `#7`/`#12`/`#13` inherit it, since that convention is the durable output of BR-2.
- **A `## Revisions` entry the plan still does not have.** The durable plan was rewritten in place again this window — the Core-concepts table, the Test-surface paragraph, the reserved-keys block, and 20 checkboxes — with no `## Revisions` section anywhere in the file. AGENTS.md §1 requires timestamp + reason + delta on a mid-stream revision. (Not filed as a finding: `BR-27` in the close ledger already names this and is open.)
- **Record the two unrecorded decisions** BR-7 asks for: form 2.1 offers no key that returns `(Skipped, true)`, so `Verdict.Skipped` as a *real* verdict is reachable only from `fakeForm` today — deliberate, and it should say so in the issue Log or under `Verdict` in the plan.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      Revert-verified in a scratch copy at HEAD: gutting the Errorf in ImportsOnly, NoWallClock and StoreSymbolsOnly each reddens its own named test; issue Log and project entry both now true.
  - id: BR-3
    disposition: addressed
    note: |
      Revert-verified: dropping SessionDone from advance's OutcomeRecord return reddens TestTheOutcomeThatEndsTheSessionSaysSo.
  - id: BR-4
    disposition: addressed
    note: |
      Table corrected on all four counts; but the same store-SYMBOL claim survives in question.go, purity_test.go and the atlas — raised as a new finding in this family.
  - id: BR-5
    disposition: not-addressed
    note: |
      Task 2 Step 1 fixed; plan:132 still reads "the loop drops it", contradicting plan:76 and session.go:170.
  - id: BR-6
    disposition: addressed
    note: |
      Renamed to TestAnUngradedKeyDoesNotAdvance; the doc comment above it at session_test.go:72 still describes skip semantics — fold into the same edit.
  - id: BR-7
    disposition: not-addressed
    note: |
      No artifact records that form 2.1 offers no skip key; question.go:24 documents only the zero value.
  - id: BR-8
    disposition: addressed
    note: |
      skipForm deleted; TestSkippedVerdictRecordsNothing drives fakeForm's '3' and still reddens when the skip filter is mutated.
  - id: BR-9
    disposition: addressed
    note: |
      question.go:24-27 now states Skipped is deliberately the zero value and why.
  - id: BR-10
    disposition: addressed
    note: |
      stderrOf carries ExitError.Stderr at both sites; revert-verified — disabling it reddens TestGuardFailureNamesTheUnderlyingError.
  - id: BR-11
    disposition: addressed
    note: |
      All Chunk 1 and Chunk 2 checkboxes ticked.
findings:
  - id: new
    severity: Important
    family: plan-table-contradicts-code
    title: |
      play runs TWO purity guards, but question.go, purity_test.go and the atlas all still claim three
    detail: |
      2nd in this family (BR-4 was the plan's copy of the same sentence). Do NOT fix the
      instance. Live sites: cmd/define/play/question.go:7-8 "Three guards enforce it — an
      import allowlist, a wall-clock grep, and a store-SYMBOL allowlist"; purity_test.go:11
      "Same three guards as schedule", contradicted two lines later by its own "the store
      guard is deliberately absent"; atlas/define.md:1179 "#6 needs the same three".
      purity_test.go:19-26 runs ImportsOnly and NoWallClock only. The rule: a claim about
      what is ENFORCED is checked against the enforcing code before it is written, and the
      fix sweeps the whole greppable enumeration rather than the site the finding named
      (ARCH-PURPOSE). grep -rn "three guards|store-SYMBOL" returns exactly these three plus
      schedule/purity_test.go:11, where the claim is true.
  - id: new
    severity: Important
    family: claim-without-failing-test
    title: |
      Four of the five code fixes in the sweep commit land in branches measured at coverage 0, and two are pinnable today
    detail: |
      4th in this family. Do NOT fix the instances. Measured with go test -coverprofile:
      play_loop.go:146-166 (BR-17's rawTerm.f and BR-24's exit 1) coverage 0 — every
      playSession test passes rawTerm{}, so raw.sess is nil and that whole block is entered
      by nothing; play_loop.go:234-240 (BR-25's message and exit 1) coverage 0;
      main.go:416-422 (BR-23's duplicate withStore removal) coverage 0; store/yaml.go:158
      not practically forceable. The rule is already in lessons.md; what is missing is the
      operable step — before closing a round, run coverage over the CHANGED lines and list
      every fix landing in a zero-coverage branch as either pinned-this-round or explicitly
      verified-by-reading in the artifact. BR-21 asked for this enumeration and received a
      prose table of err == nil sites instead of a coverage sweep, which is why the family
      recurs. BR-25's is pinnable now (a deck whose only due word the fake dictionary lacks,
      the fixture TestCountBoundsTheSession's comment already names) and BR-23's is pinnable
      now (a counting newStore, which BR-23 itself used to measure the defect).
  - id: new
    severity: Minor
    family: probabilistic-regression-test
    title: |
      The cancel-before-select test now claims determinism, and measurement contradicts it
    detail: |
      2nd in this family (BR-18). Do NOT fix the instance. play_loop_test.go:218 says
      "DETERMINISTIC by construction" and "near-certain rather than occasional". With the
      pre-select guard removed I measured 8/24 runs red using the new four-key buffer and
      7/24 using the original single key — statistically identical, because recording a
      review still needs two consecutive keys wins in the select regardless of buffer depth.
      The rule: a regression test for a nondeterministic bug removes the nondeterminism
      (repeat the scenario N times in-test, or extract the stop decision into a pure
      function and assert it), and no comment asserts a property a measurement refutes.
  - id: new
    severity: Minor
    family: docs-not-updated-for-new-surface
    title: |
      The atlas does not record Outcome.SessionDone, the puretest.T seam, or the testdata fixture convention this window introduced
    detail: |
      3rd in this family. Do NOT fix the instance. The rule: the atlas entry for a surface
      is edited in the same round the surface changes, and "new surface" includes exported
      fields and test-fixture conventions, not only user-typed commands. This window's
      enumeration of new architectural surface absent from atlas/define.md: Outcome
      .SessionDone (new exported field, justified as the affordance downstream forms
      consume), puretest.T plus the testdata known-bad-fixture obligation that a form
      package adding a guard must follow, and the new all-lookups-fail exit-1 path.
  - id: new
    severity: Minor
    family: production-code-used-as-test-fixture
    title: |
      puretest's known-GOOD fixture is the live play package, so an unrelated change to play reddens puretest's own suite
    detail: |
      puretest_test.go:57 points TestImportsOnlyAcceptsAPurePackage and
      TestNoWallClockAcceptsAPurePackage at cmd/define/play with an empty allowlist. If #7
      gives play a legitimate import, puretest's suite fails with a message about the wrong
      package. testdata/ already holds the known-bad fixtures; a five-line testdata/pure
      makes the good one owned and portable too (ARCH-MOCK's portable-fixture clause).
  - id: new
    severity: Minor
    family: commit-not-attributable-to-issue
    title: |
      2a2c837 "wip" carries 520 lines of M2 work with no issue reference, no body and no Co-Authored-By trailer
    detail: |
      AGENTS.md section 12 makes git log --grep "^#6" the retrieval path for an issue's
      history, and this commit is invisible to it. It touches main.go, play_loop.go,
      play_loop_test.go and both gate ledgers — the largest single hunk of the window.
```

---

## Re-review — 2026-08-27T12:27:09-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 6 — define --play: review loop + form 2.1 quick pass |
| repo | tools |
| issue file | workshop/issues/000006-vocab-play.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 787824810e8a371c3f2c7c7e7899943d91dfbb39..5ffaec23a3f0237b28c4696e116347f9dd2e4a0f |
| command | sdlc milestone-close --issue 6 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-27T12:27:09-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1's actual deliverable is sound and I verified it rather than trusting the commit messages: `puretest`'s guards genuinely fail when gutted, `SessionDone` genuinely reddens `TestTheOutcomeThatEndsTheSessionSaysSo`, and both of BR-30's newly-pinned fixes bite under revert. Build, `go vet` and the full suite are green. What blocks SHIP is that of seven open findings, four are re-raisable on measurement, and two of those are the *same* failure the findings themselves named: BR-29 asked for a greppable enumeration to be swept and one of four sites was fixed (the atlas still says "`#6` needs the same three" three lines above its own correction, and `schedule/box.go` says TWO where three run); BR-31 asserted a determinism property that I measured at 119/400 (≈30 %) red with the fix reverted — the comment is still there, still refuted. Additionally, the artifact that carries BR-30's own accounting — the issue Log's round-6 entry — silently lost seven inline code spans and now reads "The first attempt used , which FAILS…".

**1. Strengths**

- `cmd/define/puretest/puretest_test.go` is the right answer to BR-2 and I confirmed it load-bearing: neutering `NoWallClock`'s banned list reddens `TestNoWallClockRejectsTimeSince`. `testdata/clocky` (imports only `time`, calls `time.Since`) is precisely the hazard an import allowlist structurally cannot see, and `TestGuardsRefuseToPassVacuously` pins the fatal-on-nothing-to-check behaviour. Taking a minimal `T` instead of `*testing.T` is what makes any of it possible.
- `cmd/define/play` is genuinely pure — it imports **nothing at all** — and that is enforced, not asserted (`play/purity_test.go:19-26`). ARCH-PURE is exemplary here.
- BR-25 and BR-24 are both real fixes with real tests. Reverting `play_loop.go:234-240` to `"nothing due today"` reddens `TestAllLookupsFailingIsNotNothingDue` on all three assertions; reverting `finish(...); return 1` to `return finish(...)` reddens `TestLosingTheTerminalAfterPlaybackExitsOne`. The `missingDict`-not-`refusingDict` reasoning at `play_loop_test.go:451-453` is the kind of note that saves the next reader an hour.
- `store/yaml.go:158-165` — the `Stat` error is now returned rather than skipped past, and the comment explains *why* it matters (bypassing the torn-record guard) rather than restating the code.
- `todaysQuestions` keeps the `len(keys) == 0 → exit 0` path separate from the new all-lookups-failed path (`play_loop.go:213-217` vs `232-240`), so the "nothing due today" Done-when row survives the change.

**2. Critical findings** — none.

**3. Important findings**

- `cmd/define/play/purity_test.go:11`, `atlas/define.md:1179`, `cmd/define/schedule/box.go:12` — BR-29's class is unswept; see disposition below.
- `workshop/plans/000006-vocab-play-plan.md:24` files `puretest` guards under **Pure entities (the conceptual core)**, but `puretest.go:148` execs `go list` and `eachSourceFile` reads the filesystem. Its own tests shell out to the real `go` toolchain. That is INTEGRATION, and the row was *added* by BR-4's fix.
- `workshop/issues/000006-vocab-play.md:335-359` — seven backtick-quoted identifiers were dropped when the round-6 Log entry was written.
- `workshop/plans/000006-vocab-play-plan.md` has no `## Revisions` section despite 64 substantive lines changing this window (AGENTS.md §1).
- `workshop/plans/000006-vocab-play-plan.md:120,163` tick `sdlc milestone-close` and `sdlc close` as done; neither has run.

**4. Minor findings**

- `cmd/define/play_loop.go:74-80` — `playSession`'s doc comment now sits above `type rawTerm`, so `playSession` has none and `rawTerm`'s godoc opens "playSession drives the state machine…".
- `cmd/define/play/session_test.go:72-73` — the renamed `TestAnUngradedKeyDoesNotAdvance` kept `TestSkipAdvancesButRecordsNothing`'s doc comment about `schedule.Fold` demoting a skipped word.
- `workshop/issues/000006-vocab-play.md:349-351` — the "other three are honestly unpinnable" enumeration omits `store/yaml.go:163-165`, which I measured at coverage 0.

**5. Test coverage notes**

Measured with `go test -coverprofile` over the changed ranges (max across package runs): `play_loop.go:152-165` (lost-terminal) and `play_loop.go:234-240` (all-lookups-fail) are now **count 1** — the two BR-30 asked for. Still count 0: `main.go:416-422` (the `--play` dispatch body) and `store/yaml.go:163-165` (the `Stat` error return). One caveat on the lost-terminal test — it pins the *exit code* but not BR-17's descriptor fix: I replaced `enterRaw(raw.f)` with `enterRaw(os.Stdin)` in a scratch copy and `TestLosingTheTerminalAfterPlaybackExitsOne` stayed **green**, because `os.Stdin` under `go test` is also not a terminal. The Log already concedes that one as unpinnable, so this is confirmation rather than a new finding.

**6. Architectural notes**

- **ARCH-DRY — flag.** The guard-count fact is restated in four artifacts (`question.go:7`, `play/purity_test.go:11`, `atlas:1142`, `atlas:1179/1187`); three are now wrong and two contradict each other about `schedule`. The single-source fix is for prose to point at the enforcing `TestXPurity` body rather than restate a count.
- **ARCH-PURE — pass for `play`, flag for `puretest`.** `play` imports nothing; that is the strongest form of the claim. `puretest` is IO (exec + filesystem) filed as pure in the plan table.
- **ARCH-PURPOSE — flag.** BR-29 and BR-32 both named a class with a greppable enumeration; both rounds fixed the site the finding cited and left the siblings. That is the third round in a row this axis has been the blocker.
- **ARCH-MOCK — flag (Minor).** `puretest` shells out to the `go` binary with no named seam and no fake, and its known-*good* fixture is the live `play` package (BR-33) rather than a portable `testdata/` package — the portable-fixture clause. `#7` giving `play` one legitimate import will redden `puretest`'s own suite with a message about the wrong package.

**7. Plan revision recommendations**

- Add a `## Revisions` entry dated 2026-08-27 recording: the Core-concepts table corrections (BR-4), the "play imports NOTHING" reversal, the reserved-keys paragraph, the two-guards correction, and the Task 2/Task 3 skip-contract rewrite (BR-5) — reason + delta, per AGENTS.md §1.
- Move the `puretest` guards row out of **Pure entities** into **Integration points** (wraps: the `go` toolchain + the filesystem), with a line naming the dependency surface and the `testdata/` fixture obligation `#7`/`#12`/`#13` inherit.
- Untick `Chunk 1 Step 9` and `Chunk 2 Step 11` until each gate has actually produced its `Review-Verdict:` trailer and `## Log` close line.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      plan:105 now reads "a SKIP emits NO record outcome" and plan:130 replaces "the loop drops it" with "Apply emits no record outcome for one"; no residue found by grep.
  - id: BR-7
    disposition: not-addressed
    note: |
      question.go:30-33 documents why Skipped is the zero value (BR-9's ask), but no artifact records that no shipped form produces Skipped with ok==true; grep of plan/issue/atlas returns nothing.
  - id: BR-29
    disposition: not-addressed
    note: |
      Only question.go was corrected. play/purity_test.go:11 still says "Same three guards as schedule"; atlas:1179 still says "#6 needs the same three" three lines above its own correction at 1187; and schedule/box.go:12 says "TWO guards enforce it" for a package whose purity_test runs three.
  - id: BR-30
    disposition: addressed
    note: |
      Verified by revert in a scratch copy - both new tests redden. Coverage of the two branches is now count 1. Note the enumeration omits store/yaml.go:163-165, also measured count 0.
  - id: BR-31
    disposition: not-addressed
    note: |
      Measured in-process, N=400, pre-select guard removed - 119/400 red (~30%), matching the theoretical 0.25 for two consecutive select wins. play_loop_test.go:219 still says "DETERMINISTIC by construction" and :232 still says "near-certain rather than occasional".
  - id: BR-32
    disposition: not-addressed
    note: |
      grep of atlas/define.md returns no SessionDone, no puretest.T, no testdata known-bad-fixture convention and no all-lookups-fail exit-1 path. The window's atlas edit covers only the guard count and the reserved keys.
  - id: BR-33
    disposition: not-addressed
    note: |
      puretest_test.go:57 still points the known-good fixtures at cmd/define/play with an empty allowlist.
  - id: BR-34
    disposition: not-addressed
    note: |
      A fresh instance landed in this window - 4cf2616 "wip", 507 lines including question.go, play_loop_test.go and the atlas, with no issue reference, no body and no Co-Authored-By trailer.
findings:
  - id: new
    severity: Important
    family: plan-table-contradicts-code
    title: |
      The plan files puretest under "Pure entities" but it execs `go list` and reads the filesystem
    detail: |
      This is the 3rd finding in family `plan-table-contradicts-code`. Earlier rounds fixed
      instances. Do NOT fix this instance. The rule: a Core-concepts row's KIND is
      determined by reading the entity's imports, not by where the entity conceptually
      belongs - and the check is mechanical, so it applies to every row at once.
      Measured prevalence: plan:24 lists `puretest` guards under "Pure entities (the
      conceptual core)"; puretest.go:148 and :168 run `exec.Command("go", "list", ...)`
      and `os.ReadFile`, and puretest_test.go drives the real toolchain against real
      packages. The row was ADDED by BR-4's fix, so a fix in this family created a new
      member of it. ARCH-PURE at-review, and ARCH-MOCK: the `go` binary is an external
      dependency consumed with no named seam and no fake.
  - id: new
    severity: Important
    family: unreadable-artifact-edit
    title: |
      The issue Log's round-6 entry silently lost seven inline code spans, making BR-30's own accounting unreadable
    detail: |
      workshop/issues/000006-vocab-play.md:340-358. Raw text reads "The first attempt
      used , which FAILS the test when consulted", "handing the session a  whose file is
      , so the re-entry genuinely fails", "the duplicate  is idempotent by
      construction", "BR-29:  and the atlas both claimed THREE purity guards where \n
      runs two", and "the decision moved to  - the same two-places-for-one-rule". Line
      341 is also a dangling indented fragment. Seven identifiers are gone. This is the
      artifact BR-30 asked to carry the pinned-vs-verified-by-reading enumeration, so
      the record of the disposition is itself unreadable. The rule: an artifact edit is
      re-read from the file after writing - a write that drops content is not a record,
      and the check is one `sed -n` away.
  - id: new
    severity: Important
    family: artifact-revised-without-revision-entry
    title: |
      The durable plan was substantively revised this window with no `## Revisions` entry
    detail: |
      This is the 2nd finding in family `artifact-revised-without-revision-entry`.
      Earlier rounds fixed instances. Do NOT fix this instance. The rule (AGENTS.md
      section 1): a plan artifact revised mid-stream gets an appended `## Revisions`
      entry - timestamp, reason, delta - and the trigger is "the diff touched the
      artifact", not "the change felt large". Measured: `grep -n "^## "` on
      workshop/plans/000006-vocab-play-plan.md returns Core concepts / Chunk 1 / Chunk 2
      / Risks and no Revisions section at all, while the window changed 64 lines
      including the Core-concepts table, the "play imports NOTHING" reversal, the
      two-guards paragraph, the reserved-keys paragraph and both skip-contract steps.
      The issue file has a `## Revisions` section covering the same period; the plan
      does not.
  - id: new
    severity: Important
    family: plan-checkboxes-not-ticked
    title: |
      The plan ticks its two gate-invocation steps and the issue ticks M2, though neither gate has produced a verdict trailer or close line
    detail: |
      This is the 3rd finding in family `plan-checkboxes-not-ticked`. Earlier rounds
      fixed instances. Do NOT fix this instance. The rule: a checkbox asserts an EVENT
      happened, and for a gate step the event has a checkable artifact - so a `- [x]`
      on a `sdlc milestone-close` / `sdlc close` row is only writable after that
      command's `Review-Verdict:` trailer and `## Log` close line exist. Measured:
      `git log main..HEAD --format='%(trailers:key=Review-Verdict)'` is empty for all 20
      branch commits; `grep "closed M1\|closed M2"` on the issue returns nothing; the
      issue frontmatter is still `status: working`. Yet plan:120 is `- [x] Step 9:
      sdlc milestone-close --issue 6 --milestone M1`, plan:163 is `- [x] Step 11: sdlc
      close --issue 6`, and issue:69-70 tick both M1 and M2. BR-11's fix ticked every
      plan checkbox including the two gate rows, which is how fixing one member of this
      family created another.
  - id: new
    severity: Minor
    family: comment-detached-from-its-declaration
    title: |
      playSession's doc comment is now attached to `type rawTerm`, leaving playSession undocumented
    detail: |
      cmd/define/play_loop.go:74-90. Inserting rawTerm between playSession's comment and
      its declaration merged the two comment blocks, so godoc renders rawTerm as
      "playSession drives the state machine and performs its outcomes. Split from
      runPlay so a test can drive a whole session ... rawTerm is the terminal a session
      borrows ...". Move the playSession block back down to sit immediately above
      `func playSession`.
  - id: new
    severity: Minor
    family: test-name-contradicts-assertion
    title: |
      TestAnUngradedKeyDoesNotAdvance kept the skip-rule doc comment from the name it replaced
    detail: |
      This is the 2nd finding in family `test-name-contradicts-assertion`. Earlier
      rounds fixed instances. Do NOT fix this instance. The rule: a test's NAME and its
      DOC COMMENT are both claims about what the body asserts, so renaming a test means
      re-reading its comment in the same edit. Measured: session_test.go:72-73 still
      reads "A skip is not an assessment: schedule.Fold would read a recorded skip as a
      miss and demote a word the learner was honest about" above a test whose body
      (:75-84) asserts that an UNGRADED key does not advance - the skip claim it
      describes is now tested two functions down in TestSkippedVerdictRecordsNothing.
      BR-6's fix renamed the function and left the comment.
```
