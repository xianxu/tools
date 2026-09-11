# Boundary Review — tools#50 (whole-issue close)

| field | value |
|-------|-------|
| issue | 50 — ask before making the current directory a deck |
| repo | tools |
| issue file | workshop/issues/000050-ask-before-making-the-current-directory-a-deck.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3ce76e8b43d4995f8d78e70523c671327763d75a..35c9c0723c048f4e8943a65d1d963a042e1b503d |
| command | sdlc close --issue 50 |
| reviewer | claude |
| timestamp | 2026-09-10T23:26:34-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The shipped behaviour is right and I confirmed it on the real binary, not only in tests: piped in a fresh directory `define` creates nothing, says why, and still answers; `--stats` prints *"Nothing yet, and nothing is being saved here"* and exits 0; `--here` creates `events/` unasked; the suite is green at the pinned head (`go build`, `go vet`, `go vet -tags conformance`, `gofmt` all clean; `cmd/define` ok in 111s). BR-2 is genuinely fixed — I reverted the `(dir, lang)`-keyed fallback in a scratch worktree and `TestADeniedSessionSurvivesALanguageSwitch` went red with the exact message it was written for. What blocks SHIP is that **BR-1's fix is vacuous for 5 of the 7 methods it claims to cover**: I re-applied BR-1's own mutation (`_ = g.disk.SetItems(key, items)` before the gated call) and the **entire** `cmd/define` suite stayed green, including the new `TestNoCreatingMethodTouchesADeclinedDirectory` — because `callStoreMethod` passes zero-value args and, measured, only `AppendEvent` and `SetUserModel` write to disk under those args. The companion test written to prevent exactly this waives the per-method check (`asserts the SET is non-trivial rather than demanding all seven`), which is what let the class through; the `## Log`'s claim that it has "a companion test so the denial cannot pass by the call doing nothing at all" is false as written. Second, M2's own central ordering claim is unpinned: with `d.deckPermission.resolve()` moved *below* both loop shells, `TestBothLoopShellsResolveBeforeReading` passes and so does the full 111s suite. Third, the durable artifacts still state what the code deliberately does not do, with no `## Revisions` — and one of those lines is the Done-when the close gate reads.

## 1. Strengths

- **`cmd/define/main.go:324-343`** — the `(dir, lang)`-keyed fallback is a real fix to a real bug, and it is pinned by a test that drives `newLangDeps` the way `/lang` does. Reverting it to `store.NewMem()` per wrapper reddens `gated_store_test.go:405` immediately. That is what a closed finding looks like.
- **`cmd/define/deckperm.go:139-157`** — splitting `deckPolicy` out of `deckAsker` is the right response to the smoke-test defect: the three answers that need nobody settle quietly, the one that would prompt does not. `TestStatsIsHonestWhereTheAnswerNeedsNobody` pins all three cases *and* that settling asks nobody.
- **`cmd/define/deckperm.go:213-239`** — `readLineUnbuffered` is the correct fix for BR-6, and the comment explains the failure it prevents rather than just asserting the choice.
- **`cmd/define/store/isdeck_test.go:103-130`** — the bracket/star/question/brace × two-artifact table is the PQ-5 guard done properly: it tests the *file* branch, which is the only branch a pattern lives in. Eight subtests where a lesser version would have had one.
- **`cmd/define/gated_store_test.go:232-247`** — running the wrapper through `storetest.Suite` in **both** modes. ARCH-MOCK in one move: `store.Mem` is a stateful fake held to the same conformance suite as `YAML`, and production and test flow share the boundary.
- **`cmd/define/deckperm_e2e_test.go:15-28`** — clearing `d.capture` so the e2e tests actually reach the write path, with the comment recording that three of them previously "passed" by never exercising the feature. That correction is worth more than the tests it fixed.
- **`cmd/define/README.md:44-72`** and **`atlas/define.md`** — the new prose is accurate, concrete and reader-facing, and BR-5 is properly closed at both levels.

## 2. Critical findings

None. No shipped behaviour is wrong; every gap below is in the guards or the artifacts.

## 3. Important findings

**(A) `guard-fails-open` — third round. Do not fix the instance; write the enumeration.** `cmd/define/gated_store_test.go:167,178-199`, `cmd/define/deckasker_test.go:91-128`

Measured this round, in a scratch worktree at the pinned head:

| mutation | result |
|---|---|
| `_ = g.disk.SetItems(key, items)` before the gated call (BR-1's own mutation) | **entire `cmd/define` suite green**, exit 0, 111s |
| zero-arg reach of each creating method on a real `YAML` | `AppendEvent`, `SetUserModel` write; `Upsert`, `SetNewsItems`, `SetWordFacts`, `SetItems`, `SetAudio` write **nothing** → 5 of 7 subtests vacuous |
| `resolve()` moved below both loop shells in `repl.go` | `TestBothLoopShellsResolveBeforeReading` **passes**; full suite green |

The rule that covers all three (and BR-7): **an assertion of an absence or an ordering is worth nothing without a per-instance control that the mutation making it false actually reddens that named assertion.** An aggregate control — `TestCreatingMethodsDoReachAnAllowedDirectory`'s "the SET is non-trivial rather than demanding all seven" — certifies seven claims with a sample of one, which is the same error at test level that `guard-fails-open` names at code level. The enumeration this implies, to be written in one pass rather than one site at a time: for each of the 7 `createsOnDisk` methods, a `sampleArgs` entry that provably writes when allowed (assert it per method, not `any`), and for the ordering claim, a seam — an `io.Reader` wrapper recording when the shell first calls `Read` — so "settled before the shell read a key" is falsifiable rather than confirmed by whichever interleaving ran.

**(B) `readme-gate` — second round. The sweep, not the file.** `cmd/define/main.go:528-531`, `cmd/define/README.md:52`

BR-5 was answered at both READMEs, but `--help` — the first surface a user types — still says *"define records what you look up under words/ and events/ in the CURRENT DIRECTORY … A word that was found is added to the deck"*, unconditionally, which is now false in the third state. The rule: **when a behaviour changes, enumerate every place the old behaviour is asserted and sweep them in the same round** — here that set is `README.md`, `cmd/define/README.md`, `atlas/define.md`, the `fs.Usage` prose, the issue's Done-when and the plan. Related and cheap: `cmd/define/README.md:52` quotes the literal prompt string that `deckperm.go:195` prints, with nothing keeping them in step — this package already owns `doc_sync_test.go` for exactly that class (its own comment records the flag/README/atlas drift it was written for), so the quoted line should derive.

**(C) The three-way precedence is encoded twice.** `cmd/define/deckperm.go:149-157` vs `:181-194` — ARCH-DRY

`deckPolicy` and `deckAsker` each independently implement *already-a-deck → yes / `--here` → yes / no terminal → no / otherwise ask*. They agree today; nothing makes them. One observable consequence already exists: only `deckAsker` prints the *"nothing will be saved (use `--here` to create one)"* explanation, so whether a piped user is told depends on which encoding settles the state first — I confirmed `echo word | define` prints it (the REPL resolves first) while `define --forget cat` piped does not (a read settles it quietly). Consolidation: `deckAsker` switches on `deckPolicy(dir, opt, stdinIsTerminal)` and owns only the `deckUndecided` arm plus the printing.

## 4. Minor findings

- `cmd/define/stats.go:87,101` — `renderStats(s, now, true)` at six call sites: a bare positional bool that a reader has to look up. A named type or an options struct field reads better.
- `cmd/define/deckperm.go:224` — `readLineUnbuffered` has no length bound; only reachable on a terminal, so low risk, but a `4096`-byte cap costs a line.
- `workshop/lessons.md` has an uncommitted 34-line addition in the working tree (the "run the program" lesson). It is outside the reviewed window and will not be in the close's recorded range unless committed first.

## 5. Test coverage notes

- `isdeck_test.go`, `deckperm_test.go` and `deckasker_test.go`'s policy table are strong: derived tables, no IO in the permission tests, and each asserts the negative (`saving()` must not consult `ask`) rather than only the happy path.
- The e2e file is the right shape — it lists the directory rather than trusting the gate, and `TestAcceptingCreatesTheDeck` is a real positive control for `TestDecliningCreatesNothingOnDisk`.
- The uniform gap is (A): **consultation** is pinned mechanically, **routing** is pinned by the conformance suite, but **"no bytes reached the declined directory"** — the Done-when's own wording — is still pinned for 2 of 7, and **ordering** is pinned for 0 of 2 shells.
- `readLineUnbuffered`'s no-read-ahead property has no test. `in := strings.NewReader("y\ndog\n")`, run the asker, then assert the remainder reads `"dog\n"` — one assertion, and it is the whole of BR-6.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag (C)**, plus four hand-maintained prose restatements of one behaviour in a package that already has the derivation machinery.
- **ARCH-PURE — pass.** Policy in the command, bytes in the store, decision travelling as a value; `deckPermission`/`deckDecision` test with no IO at all. Only the plan's label for `store.IsDeck` is wrong (BR-4).
- **ARCH-PURPOSE — flag (A), (B).** Both are the instance-vs-class axis on its second round: BR-1's answer was made class-*shaped* (it iterates a map) without being class-*effective* (the args make 5 of 7 subtests no-ops), and BR-5's answer covered the two files named and not the surface class they belong to.
- **ARCH-MOCK — pass.** `store.Mem` behind the same seam, same conformance suite, run allowed and denied; moving the denial test onto a real `store.NewYAML(t.TempDir(), …)` was the right call independent of the args problem.
- **ARCH-CONSTRAINTS — flag, Minor.** BR-9 stands, and it is now slightly worse than "once per process": `settleQuietly()` sits on `reading()`, so while the decision is undecided every read re-evaluates `deckPolicy` → `IsDeck` → `os.ReadDir` + sort of the whole cwd (measured: 5 evaluations for 5 reads). Measured cost in a real one-shot terminal lookup is 1, which is why this stays Minor rather than escalating the family.
- **ARCH-SECURE — pass.** The cwd is kept out of glob patterns and that is pinned in 8 subtests; the question and the decline go to stderr so a redirected stdout stays clean; no credentials anywhere; tests use `t.TempDir`/`t.Chdir` and cannot reach real user state; an unreadable directory degrades to *ask* rather than to a fabricated answer.
- **ARCH-ORDER — flag (A).** The tagged `deckDecision` is exemplary and the reasoning is recorded where the next reader hits it. But the diff's own ordering claim is observed as an end state, not an ordering — the highest-leverage flag in this entry, and the M1 review predicted it in §6. Separately, I verified there is no concurrency exposure to fix: the only goroutines in `cmd/define` are `repl.go:413`, `rawterm.go:87/242` and `interrupt.go:83`, none of which touch the store, so "NOT SAFE FOR CONCURRENT USE" is honestly discharged.

## 7. Plan revision recommendations

BR-4 is unaddressed and no `## Revisions` section exists in either artifact. Append one to each (timestamp + reason + delta):

- `workshop/issues/000050-…-deck.md` — (i) Done-when bullet 6: *"Every **write-shaped** method"* → *"Every **creating** method … `Forget` is write-shaped and deliberately ungated (PQ-8)"*; (ii) Plan M1 row 2: *"8 gated writes, 7 ungated reads"* → *"7 gated creating writes, 8 ungated (7 reads + `Forget`)"*.
- `workshop/plans/000050-…-plan.md` — (i) move `store.IsDeck` from **Pure entities** to **Integration points**; (ii) `newGatedStore(disk store.Store, perm)` → `newGatedStore(disk, mem store.Store, perm)`, and add the `(dir, lang)`-keyed fallback to Task 5; (iii) add `deckPolicy`, `withQuiet`/`settleQuietly` and `readLineUnbuffered` to the Core-concepts table — three entities shipped that the table does not list; (iv) Task 9 names `cmd/define/play_loop.go` as a file to modify; it was not touched — drop it or say why; (v) add the README step (Task 10 Step 7 still names only the atlas).

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Re-measured: BR-1's own mutation leaves the whole suite green; zero-value args make 5 of 7 subtests vacuous and the positive control waives per-method checking.
  - id: BR-2
    disposition: addressed
    note: |
      Revert-verified — restoring the per-wrapper store.NewMem() reddens gated_store_test.go:405 with its own message.
  - id: BR-3
    disposition: addressed
    note: |
      deckasker_test.go is committed, -here has production readers, and --here creates events/ unasked on the real binary.
  - id: BR-4
    disposition: not-addressed
    note: |
      Done-when still says "write-shaped", the Plan row still says 8/7, IsDeck is still listed PURE, and neither artifact has a "## Revisions" section.
  - id: BR-5
    disposition: addressed
    note: |
      Both READMEs document the question, the decline and --here; --help is a separate surface, raised below.
  - id: BR-6
    disposition: not-addressed
    note: |
      readLineUnbuffered is the right code but no test fails without it; one assertion on the unread remainder closes it.
  - id: BR-7
    disposition: not-addressed
    note: |
      The class test iterates createsOnDisk, not storeInterfaceMethods, so a coordinated bucket-move plus creating()->reading() is still invisible.
  - id: BR-8
    disposition: not-addressed
    note: |
      storeInterfaceMethods still reads only m.Names and the floor is still a hand-typed "< 15".
  - id: BR-9
    disposition: not-addressed
    note: |
      Still os.ReadDir, and settleQuietly now sits on the read path so it is no longer once per process while undecided (measured 5 evaluations for 5 reads; 1 in a real one-shot lookup).
findings:
  - id: new
    severity: Important
    family: guard-fails-open
    title: |
      third round of guard-fails-open — the rule is that an absence or ordering claim needs a per-instance control, not an aggregate one
    detail: |
      This is the 3rd+ finding in family guard-fails-open (BR-1, BR-7, and now the
      loop-shell ordering claim). Do NOT fix the instances. Measured at the pinned head
      in a scratch worktree: (1) BR-1's dual-write SetItems mutation leaves the entire
      cmd/define suite green, exit 0; (2) of the 7 createsOnDisk methods only
      AppendEvent and SetUserModel write anything under callStoreMethod's zero-value
      args, so 5 of 7 subtests of TestNoCreatingMethodTouchesADeclinedDirectory are
      no-ops, and TestCreatingMethodsDoReachAnAllowedDirectory explicitly waives the
      per-method check that would have caught it; (3) moving repl.go:245's
      resolve() below both shells leaves TestBothLoopShellsResolveBeforeReading and the
      full 111s suite green, so PQ-3's ordering claim is pinned for 0 of 2 shells. The
      rule: every assertion of an absence or an ordering must be paired with a mutation
      that makes it false and shown to redden THAT assertion, per instance. Write the
      enumeration (a sampleArgs entry per creating method with a per-method positive
      assertion, and a Read-recording seam for the ordering) in one pass.
  - id: new
    severity: Important
    family: readme-gate
    title: |
      second round of readme-gate — --help still promises unconditional recording, and the README quotes a prompt string nothing keeps in step
    detail: |
      This is the 2nd finding in family readme-gate. Do NOT fix only the instance.
      main.go:528-531's usage prose still says define "records what you look up under
      words/ and events/ in the CURRENT DIRECTORY … A word that was found is added to
      the deck", which is false in the third state; --help is the first surface a user
      types. The rule: when behaviour changes, enumerate every place the old behaviour
      is asserted and sweep them in one round — README.md, cmd/define/README.md,
      atlas/define.md, the fs.Usage prose, the issue's Done-when, the plan. Cheap
      structural half: cmd/define/README.md:52 quotes the literal string deckperm.go:195
      prints, and this package already owns doc_sync_test.go for exactly that class.
  - id: new
    severity: Important
    family: policy-restated-not-derived
    title: |
      deckPolicy and deckAsker independently encode the same three-way precedence
    detail: |
      deckperm.go:149-157 and :181-194 each implement already-a-deck / --here /
      no-terminal. They agree today and nothing makes them. Observable consequence
      already present: only deckAsker prints the "nothing will be saved (use --here)"
      explanation, so whether a piped user is told depends on which encoding settles the
      state first — confirmed on the real binary (echo word | define prints it,
      define --forget cat piped does not). ARCH-DRY: deckAsker should switch on
      deckPolicy and own only the deckUndecided arm.
  - id: new
    severity: Minor
    family: positional-bool-parameter
    title: |
      renderStats gained a bare positional bool, read at six call sites as a literal true
    detail: |
      stats.go:101's `saving bool` appears as renderStats(s, now, true) in stats_test.go
      and deckasker_test.go. A named type or a field makes the call sites self-describing.
```
