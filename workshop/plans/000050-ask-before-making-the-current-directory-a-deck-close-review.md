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

---

## Re-review — 2026-09-10T23:56:17-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 50 — ask before making the current directory a deck |
| repo | tools |
| issue file | workshop/issues/000050-ask-before-making-the-current-directory-a-deck.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3ce76e8b43d4995f8d78e70523c671327763d75a..a30cb79e8366c421fbbf01085e96faa522c2e897 |
| command | sdlc close --issue 50 |
| reviewer | claude |
| timestamp | 2026-09-10T23:56:17-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The feature itself is sound and the round's headline fixes are real — I re-ran the prior reviewer's mutations in a scratch worktree and BR-1/BR-7's dual-write and reclassification mutations now redden `TestCreatingMethodsWriteWhenAllowedAndNotWhenDenied/SetItems`, and reverting `readLineUnbuffered` to `bufio` reddens the stdin-remainder test. What blocks SHIP is that three of the open findings are still open *as rules*, measured not read: (1) BR-10's per-instance rule holds for the seven creating methods but not for the other two absence/ordering claims — moving `resolve()` into `replLines` leaves the whole 111 s suite green while `replRaw`'s own key reader loses the pre-resolution, and making `YAML.Forget` `MkdirAll` leaves the whole suite green while a declined directory grows `words/en/`; (2) BR-12's cited consequence survives the DRY fix — `define word` piped in a fresh directory prints no explanation while `echo word | define` does, because a read settles the state silently through `settleQuietly` and `deckAsker` (the only thing that says why) is never reached; (3) BR-11's sweep still hasn't happened — `cmd/define/README.md:586` still says *"**Every** successful lookup — one-shot, **piped**, or in the editor — records the word"*. Plus one new behaviour defect: `define -raw`, the mode documented as never writing and never asking, now asks (verified: answering `y` creates nothing) and tells piped scripts to use `--here`.

### 1. Strengths

- `gated_store_test.go:223` — the per-method positive control is the right shape and it works: I mutated `SetItems` to dual-write past the gate and the subtest named it (`SetItems created [items] in a directory the learner declined`). `TestEveryCreatingMethodHasASample` deriving the table from `createsOnDisk` is what keeps it from rotting.
- `deckperm.go:233` `readLineUnbuffered` — verified by reversion; swapping in `bufio.NewReader(in).ReadString` makes `TestTheAnswerDoesNotSwallowTheRestOfStdin` fail on the *remainder*, which is the observable that actually distinguishes the two implementations.
- `main.go:341-353` — the language-keyed fallback map, with `TestADeniedSessionSurvivesALanguageSwitch` driving `newLangDeps` rather than one wrapper. That test cannot pass without the fix.
- `gated_store_test.go:285-300` — running the wrapper through `storetest.Suite` both allowed *and* denied is the strongest available check that delegation is faithful, and the denied run is the evidence that "declined" means empty, not broken.
- `store/isdeck.go:55` + `isdeck_test.go:103` — `filepath.Match` against entry names, with the bracket/star/question rows placing a runtime **file** (the only branch that globs). The comment records why the first draft of that row proved nothing (ARCH-SECURE: the cwd is input this code did not choose, and it never enters a pattern).

### 2. Critical findings

None.

### 3. Important findings

**a. `-raw` asks a question it can never act on** — `main.go:806`, `capture.go:26-33`.
Measured at the pinned head: `define -raw` on a terminal in a fresh directory prints `… is not a deck yet. Create one here? [y/N]`, and answering **y** leaves the directory empty (`created=[]`) — because `decideCapture` returns `captureNothing` for `opt.raw`. Piped, the same run prints `… nothing will be saved (use --here to create one)` on every invocation, advising a flag that would change nothing. The Spec's own rule is *"a prompt for a command that would not have written anything is a false alarm, and false alarms train people to hit `y`"*, and `cmd/define/README.md:702` already tells the reader `-raw` *"records nothing … and neither does it ask."* `run()` guards the permission on `!opt.noCapture` only; `capture.go:192` already records that `DEFINE_NO_CAPTURE` and `-raw` are one class. Fix at the class, not the flag: derive "will this invocation write?" from the existing capture policy (`decideCapture(…) == captureNothing`) at the one site that decides whether to build a permission at all.

**b. The deck question cannot be interrupted** — `deckperm.go:179`, `repl.go:245`, `main.go:401`.
`main()` installs `signal.NotifyContext`, and `repl()` calls `detachedInterrupts` *before* `resolve()`, so SIGINT is intercepted process-wide by the time the question is on screen. `deckAsker` takes no context and blocks in `readLineUnbuffered` until a newline or EOF, so Ctrl-C at the prompt fires `cancel()` into a context nobody is watching and the question stays up; the user's only exits are Enter (which silently *declines*) or Ctrl-D. This is a new blocking prompt on the interactive path in a program that otherwise defines what Ctrl-C means. Cheapest honest fix: give the asker the cancellation channel and treat a cancelled context as a decline with a printed reason.

### 4. Minor findings

- The plan's `## Core concepts` never gained a row for `deckPolicy` / `withQuiet` / `settleQuietly` — entities M2 introduced after smoke testing — and the `store.IsDeck` bullet still sits under `### Pure entities` (plan:41) while its table row moved to Integration points, so the plan lists it in both sections.
- `renderStats`' third positional `bool` now reads as a bare `true` at six call sites (`stats_test.go:22,54,113,135,137,257,263`).
- `storeInterfaceMethods` still reads only `m.Names` (embedded interfaces contribute zero) and the floor is still a hand-typed `< 15`.
- `IsDeck` still uses `os.ReadDir` (materialise + sort). Measured cost per process: 2 calls for a terminal one-shot, 3 for terminal `--stats`, 1 piped — bounded, so the cost is small, but `f.ReadDir(-1)` would drop the sort.

### 5. Test coverage notes

- The Done-when claims `--play` and `/history` "report EMPTY rather than refusing"; only `--stats` has an end-to-end assertion (`TestStatsInAnUnsavedDirectoryReportsEmpty`). The `Mem` fallback makes it structurally true, but nothing pins it.
- `TestBothLoopShellsResolveBeforeReading`'s "raw shell" subtest takes `replRaw`'s *fallback* to `replLines` (its own name says so), so both subtests observe the same code path. `pty_conformance_test.go` (`//go:build darwin && conformance`) is the existing seam that drives the real raw shell; it is the natural home for the raw-path ordering control, noting that only one anchored conformance test runs at the merge gate today.
- Nothing asserts that a `doesNotCreate` method creates nothing on disk — the bucket's claim is verified only as "it does not consult the permission".

### 6. Architectural notes

- **ARCH-DRY** — pass on the precedence (`deckPolicy` is now the one encoding) and on `deckPrompt`. Flagged once: `run()` restates half of the write-nothing class (finding 3a).
- **ARCH-PURE** — pass. Policy in the command, bytes in the store; `gatedStore` wraps rather than edits `YAML`; `renderStats` takes the flag rather than re-deriving the policy.
- **ARCH-PURPOSE** — flag. The shadow-sweep over "every surface derives from one source" finds `cmd/define/README.md:586-587` still hand-asserting the behaviour this issue changed, and `TestEverySurfaceDescribingCaptureMentionsTheQuestion` checks only that the string `-here` appears *somewhere* in each file, which a file can satisfy while contradicting itself 540 lines later — exactly what happened.
- **ARCH-MOCK** — pass. `storetest.Suite` over the wrapper both ways; the terminal is an injected predicate and an `io.Reader`; disk assertions run against a real `t.TempDir` rather than an in-memory lookalike.
- **ARCH-CONSTRAINTS** — pass with the `IsDeck` note above; the prompt blocks only where a human is present.
- **ARCH-SECURE** — pass. The cwd never enters a glob pattern; an unreadable directory reads false and therefore asks; no credentials on this path.
- **ARCH-ORDER** — flag, three ways: the raw shell's ordering claim has no control that can redden (BR-10); the deny transition's user-visible effect is attached to one path rather than to the transition (BR-12); and the question has no cancellation arm (finding 3b).

### 7. Plan revision recommendations

- `## Revisions` — "M2 introduced `deckPolicy`, `withQuiet` and `settleQuietly` after smoke testing; Core concepts gains `deckPolicy` (integration — it calls `store.IsDeck`) and the `store.IsDeck` bullet moves out of `### Pure entities` to match its table row."
- `## Revisions` — "Task 8's 'one test per shell' is not what shipped: both subtests reach `replLines` (the raw one via `replRaw`'s non-file-stdin fallback). The raw path's control belongs in the pty conformance suite."
- `## Non-goals` / Task 7 — record the `-raw` case explicitly: either "`-raw` never reaches the gate, derived from `decideCapture`" once fixed, or the current behaviour as a known deviation from the Spec's false-alarm rule.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Verified by mutation: a dual-writing SetItems now reddens TestCreatingMethodsWriteWhenAllowedAndNotWhenDenied/SetItems.
  - id: BR-4
    disposition: addressed
    note: |
      Both artifacts gained "## Revisions"; the residual Pure-entities bullet for IsDeck is a new Minor.
  - id: BR-6
    disposition: addressed
    note: |
      Verified by reversion to bufio: the remainder assertion goes red.
  - id: BR-7
    disposition: addressed
    note: |
      Verified: moving SetItems to doesNotCreate plus routing via reading() reddens, because sampleCalls is independent of the buckets.
  - id: BR-8
    disposition: not-addressed
    note: |
      Still only m.Names, and the floor is still a hand-typed "< 15".
  - id: BR-9
    disposition: not-addressed
    note: |
      Still os.ReadDir. Measured cost: 2 calls per terminal one-shot, 3 per terminal --stats, 1 piped.
  - id: BR-10
    disposition: not-addressed
    note: |
      The rule holds for the 7 creating methods only. Measured at the pinned head in a scratch
      worktree: (1) moving repl.go:245's resolve() into replLines leaves the entire suite green,
      so the "both loop shells" ordering claim is still pinned for 0 of 1 raw shells — both
      subtests of TestBothLoopShellsResolveBeforeReading reach replLines, the "raw" one via
      replRaw's non-file-stdin fallback, as its own name concedes; (2) making YAML.Forget call
      MkdirAll(wordsDir) leaves the entire suite green while a declined directory grows words/en/,
      so the doesNotCreate bucket's absence claim has no control at all — only "does not consult
      the permission" is asserted. The enumeration BR-1 asked for was storeInterfaceMethods, and
      what shipped enumerates sampleCalls, whose completeness is checked only against createsOnDisk.
      Write it once, for the class: a sample per INTERFACE method with a per-method positive
      control and a byte-identical denied directory, and a raw-path ordering observation in the
      existing pty conformance seam.
  - id: BR-11
    disposition: not-addressed
    note: |
      --help was fixed, the sweep was not. cmd/define/README.md:586-587 still reads "Every
      successful lookup - one-shot, piped, or in the editor - records the word where you started
      define, so your deck and history build themselves", which is false in the third state and
      names the piped case by name; :702 "-raw records nothing ... and neither does it ask" is now
      contradicted by the program (see the new finding). The test written to close this family
      asserts only that each surface contains the substring "-here", which a document can satisfy
      while asserting the old behaviour elsewhere in the same file. The rule, restated so it can
      catch the next instance: a surface test must assert the OLD claim is GONE, not that a new
      word is present - enumerate the sentences that assert the superseded behaviour and pin their
      absence.
  - id: BR-12
    disposition: not-addressed
    note: |
      The duplicate encoding is genuinely gone (deckAsker switches on deckPolicy), but the
      consequence the finding cited survives. Measured with the real wiring: `define sycophantic`
      with stdin not a terminal in a fresh directory prints NO explanation (stderr is only the
      pronunciation warning), while `echo sycophantic | define` prints "nothing will be saved (use
      --here to create one)". Traced: the one-shot path has a READ settle the state to deckDeny
      through settleQuietly - silently - so the later write finds it decided and deckAsker, the
      only thing that says why, is never invoked. The effect belongs to the TRANSITION into
      deckDeny, not to one of the paths that can cause it; deckPermission should emit it once when
      it settles to deny, whichever path settles it. The same gap has a second face: only the EMPTY
      --stats screen says nothing is being saved, so a declined session that has looked three words
      up shows ordinary figures with no hint they are session-only.
  - id: BR-13
    disposition: not-addressed
    note: |
      Still a bare positional bool; six call sites read renderStats(s, now, true).
findings:
  - id: new
    severity: Important
    family: policy-restated-not-derived
    title: |
      -raw prompts to create a deck, and tells piped scripts to use --here, though it writes nothing
    detail: |
      This is the 2nd finding in family policy-restated-not-derived (BR-12 is the 1st, and it is
      re-raised above). Do NOT fix this instance alone. The rule: a policy this codebase already
      owns in one place must be DERIVED at every new site - including the site that decides whether
      to build the permission at all. decideCapture (capture.go:26-33) is the single source of "this
      invocation writes nothing", and capture.go:192 already states that DEFINE_NO_CAPTURE and -raw
      are one class; main.go:806 restates half of it as `!opt.noCapture`, so -raw falls through into
      the gate. Measured at the pinned head with the real wiring: `define -raw` on a terminal in a
      fresh directory prints "... is not a deck yet. Create one here? [y/N]", answering y creates
      nothing at all, and the question consumes the session's first line of stdin as its answer;
      piped, every run prints "nothing will be saved (use --here to create one)" to stderr,
      recommending a flag that would not make -raw save anything. The Spec forbids exactly this
      ("a prompt for a command that would not have written anything is a false alarm"), and
      cmd/define/README.md:702 already documents -raw as never asking. Enumerate the members of the
      write-nothing class from decideCapture and gate on the class.
  - id: new
    severity: Important
    family: blocking-prompt-ignores-cancellation
    title: |
      Ctrl-C at the deck question does nothing - the prompt blocks with no cancellation arm
    detail: |
      main.go:401 installs signal.NotifyContext, and repl.go calls detachedInterrupts BEFORE
      repl.go:245's resolve(), so by the time the question is on screen SIGINT is intercepted
      process-wide and delivered to a context nobody is watching. deckAsker (deckperm.go:179) takes
      no context and blocks in readLineUnbuffered until a newline or EOF, so Ctrl-C at the prompt
      leaves the question up; the only exits are Enter, which silently declines, and Ctrl-D. The
      program otherwise owns what Ctrl-C means, which is why a reader will expect it to work here.
      Give the asker the cancellation channel and treat a cancelled context as an explicit,
      printed decline.
  - id: new
    severity: Minor
    family: artifact-claims-what-code-does-not
    title: |
      the plan's Core concepts never gained deckPolicy, and still lists IsDeck under Pure entities
    detail: |
      This is the 2nd finding in family artifact-claims-what-code-does-not (BR-4 was the 1st and is
      disposed addressed). Do NOT fix only this row. The rule: the Core-concepts table is the
      greppable contract a boundary review cross-checks, so any entity the implementation ADDS or
      RECLASSIFIES lands in the table with a "## Revisions" entry in the same commit that adds it.
      Instances: deckPolicy, withQuiet and settleQuietly - introduced in M2 after smoke testing -
      appear in no table; and while BR-4 moved the store.IsDeck table row to Integration points, the
      bullet describing it (plan:41) still sits under the "### Pure entities" heading, so the plan
      files it in both sections.
```
