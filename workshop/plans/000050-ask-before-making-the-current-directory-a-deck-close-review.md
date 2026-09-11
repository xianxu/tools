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

---

## Re-review — 2026-09-11T00:28:01-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 50 — ask before making the current directory a deck |
| repo | tools |
| issue file | workshop/issues/000050-ask-before-making-the-current-directory-a-deck.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3ce76e8b43d4995f8d78e70523c671327763d75a..6b512c04fb75fc9e0ef6722772fca0c744f11e82 |
| command | sdlc close --issue 50 |
| reviewer | claude |
| timestamp | 2026-09-11T00:28:01-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The feature works on the real binary. In a fresh directory, piped and redirected lookups each explain once and create nothing, `-raw` stays silent, `--stats` shows the honest screen, and `--here` creates `events/`. All five Important findings from round 3 (BR-10, BR-11, BR-12, BR-14, BR-15) are fixed, and I checked each by reverting the fix in a scratch worktree: every one goes red. The full `cmd/define` suite passes at the pinned head (111s). Two new Important findings stand between this and SHIP. Both are cheap to fix:
- **`--stats` isn't pinned.** The ticked Done-when row "`--stats` in an unsaved directory does not claim a word will join your deck" has no test behind it. Two mutations of the `--stats` wiring leave the whole package green.
- **`/lang` claims a save that never happened.** In a declined directory it reports a switch, or "now on the record", when nothing was written. The Spec names this exact kind of message ("The message that would become a lie"), and only `--stats` got fixed.

### 1. Strengths
- **Every store method now has its own check.** Each creating method's test first proves the call really writes when allowed, then proves nothing appears when denied (`gated_store_test.go:223-252`). The non-creating methods got the same treatment in reverse (`:509-528`). Making `YAML.Forget` call `MkdirAll` now reddens `TestNonCreatingMethodsCreateNothing/Forget`.
- **The ordering test watches the right moment.** `observingReader` (`deckasker_test.go:168-234`) records whether the question was settled when the shell first reads stdin, not after the loop ends. Removing `resolve()` reddens both subtests.
- **The explanation follows the decision, not the path.** The reason now travels with the decision (`deckReason`), and the permission says why exactly once, whichever path settles it (`deckperm.go:79-140`). Both explanation mutations redden. On the real binary, `define cat </dev/null` and `echo cat | define` now print the same explanation.
- **BR-11's guard checks that old claims are gone.** It doesn't just look for "-here". Exact reverts of the `--help` prose and the README paragraph each redden it.
- **The stdin readers are sound.** `readLineUnbuffered` is pinned on what's left unread afterwards. `readLineCancellable`'s channel is buffered, so the abandoned goroutine can't block on send.

### 2. Critical findings
None.

### 3. Important findings
- **Done-when row 7 is pinned by nothing (4th `guard-fails-open`).** I tried two mutations, and each left the whole package green (110s):
  - making `printStats` always render the "saving" screen (`stats.go:87`);
  - passing a nil permission at both doors (`stats.go:47`, `:260`).

  Two reasons nothing catches this:
  - `deckperm_e2e_test.go:174` checks for "Nothing yet", which both screens print.
  - `TestStatsIsHonestWhereTheAnswerNeedsNobody` computes its own copy of `!decided || allowed` (`deckasker_test.go:269`) instead of calling `printStats`.

  **The rule:** a ticked Done-when row is only pinned by a test that fails when that row is made false through the production wiring. The fix is to add a table to the Log (row → test → wiring mutation → observed red) for all seven rows. My sweep of those rows:
  - Rows 1, 3, 5 and 6 already have tests that would catch a regression.
  - Row 7 has none.
  - Row 2's `--play` and `/history` half is pinned only structurally: the deck is always a non-nil `gatedStore`, but no test runs either command under a declined permission.
- **`/lang` claims an effect that was discarded.** `persistLang` returns nil when declined (`main.go:385-389`), and `TestPersistLangIsGated` asserts that nil. So `lang_cmd.go:72` and `:69` report success. On the real binary:
  - `define /lang es` prints "now defining in es", exits 0 and writes nothing; the next `define /lang` still says `en`.
  - `define /lang en` prints "still defining in en, now on the record".
  - The pre-#50 binary wrote `lang.txt`.

  Everything else that could claim this:
  - The other "saved / wrote / removed" messages all need a non-empty deck, which a declined one-shot can't have.
  - Ctrl-C at the REPL's opening question prints "the lookup still works" (`deckperm.go:263`). In a scratch test the session then exited (run returned 0) with no lookup, because the same SIGINT also fires the loop's own cancel.
  - Docs: `cmd/define/README.md:649-651` ("the setting stays with the directory") and `atlas/define.md:1128` ("persists when given").

  **Fix:** have `persistLang` return a "not saved" sentinel instead of nil, and have `/lang` say the language applies to this session only. The one-shot form can reuse the existing no-directory wording at `lang_cmd.go:57`. Pin the stdout.

### 4. Minor findings
- **`-raw` is restated rather than derived (3rd `policy-restated-not-derived`).** `deckperm.go:210` checks `opt.raw` directly. It should ask `decideCapture(true, opt) == captureNothing`, which `capture.go:198` already does for the same question.
- **Wrong README link.** `cmd/define/README.md:590` points to `#install`, not the section that describes the question.
- **`deckAsker`'s doc comment is stale.** It says "FOUR INPUTS" (there are five now, with `-raw`), "no, with a warning" (the warning moved out), and cites `deckPermission.settle`, which doesn't exist.
- BR-8, BR-9 and BR-13 are unchanged.

### 5. Test coverage notes
- **The raw-shell ordering pin has never run.** `TestPTYDeckQuestionArrivesBeforeTheEditor` can't run here either: `openpty` fails with EPERM even outside the sandbox. It has never been seen red or green. Before merge, run on a machine with a pty: `CONFORMANCE_STRICT=1 go test -tags conformance -run TestPTYDeckQuestion ./cmd/define/`
- The in-process ordering test only reaches `replLines`, so moving `resolve()` into `replLines` would still pass in-process.
- Lookups on the real binary said "no dictionary entry" because no dictionary is installed here. That's unrelated to #50; the deck behaviour could still be checked.

### 6. Architectural notes
- **ARCH-DRY: pass.** `deckAsker` now delegates to `deckPolicy`. On the ask path `deckPolicy` runs twice, so `IsDeck` reads the directory twice.
- **ARCH-PURE: pass, with a note.** `deckPolicy` calls `store.IsDeck` itself. Taking an `isDeck` input instead would make the five-input policy a pure table test.
- **ARCH-PURPOSE: flag.** The "lying message" class was swept for `--stats` only (Important #2).
- **ARCH-MOCK: pass, with a caveat.** The terminal is covered through the pty seam, which hasn't been executed.
- **ARCH-CONSTRAINTS:** BR-9 is still open. The cost is bounded at a few directory reads per invocation.
- **ARCH-SECURE: pass.** The working directory never becomes part of a pattern, and answers are parsed strictly.
- **ARCH-ORDER: pass, with a note.** The decision is an explicit three-value type, and ordering is sampled at the first read. The abandoned reader goroutine is only safe in the REPL because the same SIGINT also ends the session. Nothing states or tests that link.

### 7. Plan revision recommendations
- **Core concepts:** add `deckPolicy`, `deckReason`, `explainDenial`, `readLineUnbuffered`/`readLineCancellable`, and `withQuiet`/`settleQuietly`. Move the `IsDeck` bullet out of Pure entities. Give `deckAsker` its `ctx` parameter and five inputs.
- **Issue `## Revisions`:** the REPL resolves the question at session start, which contradicts the Spec's "Not at startup". `-raw` is a fifth input.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      Still only m.Names and a hand-typed "< 15" floor; latent, since store.Store embeds nothing today.
  - id: BR-9
    disposition: not-addressed
    note: |
      Still os.ReadDir, and deckPolicy runs twice on the ask path (quiet, then ask), so two ReadDirs there.
  - id: BR-10
    disposition: addressed
    note: |
      Revert-verified: Forget->MkdirAll reddens TestNonCreatingMethodsCreateNothing/Forget; removing resolve() reddens both TestTheLineShellResolvesBeforeReading subtests. The raw-shell pty pin could not run here either (openpty EPERM, unsandboxed), so it has never been observed; run it with CONFORMANCE_STRICT=1 on a pty-capable machine before merge.
  - id: BR-11
    disposition: addressed
    note: |
      Exact literal reverts of the fs.Usage prose and the README paragraph each redden TestEverySurfaceDescribingCaptureMentionsTheQuestion; deckPrompt is pinned against the README. The /lang surfaces belong to the new confirmation finding.
  - id: BR-12
    disposition: addressed
    note: |
      Both explanation mutations (no sayWhy on settle; resolve skipping the quiet half) redden; real binary: redirected and piped lookups each explain once. The non-empty /stats "second face" is outside the Spec, which asked only for the empty message.
  - id: BR-13
    disposition: not-addressed
    note: |
      Still renderStats(s, now, true) at every call site.
  - id: BR-14
    disposition: addressed
    note: |
      Reverting the -raw branch reddens TestRawNeitherAsksNorAdvises; real binary: -raw piped and one-shot print no advice and create nothing. The derivation the finding asked for was not done; raised below as a Minor.
  - id: BR-15
    disposition: addressed
    note: |
      Reverting readLineCancellable hangs TestInterruptAtTheQuestionDeclines 5s and reddens; reachable, because deckAsker holds run()'s NotifyContext ctx, which SIGINT still cancels. The post-cancel message is part of the new confirmation finding.
  - id: BR-16
    disposition: not-addressed
    note: |
      Plan untouched since a30cb79: deckPolicy, deckReason, explainDenial, readLineCancellable, withQuiet, settleQuietly absent; IsDeck bullet still under Pure entities; plan's deckAsker lacks ctx and says four inputs (five now), and deckAsker's own doc comment says FOUR INPUTS and cites a nonexistent deckPermission.settle.
findings:
  - id: new
    severity: Important
    family: guard-fails-open
    title: |
      4th guard-fails-open: the issue's Done-when list is the enumeration never written, and its --stats row is pinned by nothing
    detail: |
      This is the 4th finding in family guard-fails-open. Earlier rounds applied the per-instance rule to components (store methods, buckets, loop shells), and those now hold. The enumeration never written is the issue's own Done-when list, where the feature's claims live. Measured at the pinned head: making printStats render the saving screen unconditionally (stats.go:87), or handing both doors a nil permission (stats.go:47 and :260), each leaves the whole cmd/define package green (110s). So the ticked row "--stats in an unsaved directory does not claim a word will join your deck", the exact defect smoke testing found, is pinned by nothing through production wiring. deckperm_e2e_test.go:174 asserts "Nothing yet", which both screens print, and TestStatsIsHonestWhereTheAnswerNeedsNobody computes its own copy of the formula (deckasker_test.go:269) instead of calling printStats. The rule: a ticked Done-when row is pinned only by a test that fails when the row is made false through production wiring. Write a Log table for all seven rows: row, test, wiring mutation, observed red. Sweep result: rows 1, 3, 5 and 6 have discriminating controls; row 7 has none; row 2's --play and /history half is pinned only structurally (non-nil gatedStore), with no test running either under a declined permission.
  - id: new
    severity: Important
    family: confirmation-not-derived-from-effect
    title: |
      /lang reports a switch and "now on the record" in a declined directory, because persistLang swallows the write its only caller reports
    detail: |
      The Spec names this class ("The message that would become a lie") and --stats was fixed; /lang was not. persistLang returns nil when declined (main.go:385-389), and TestPersistLangIsGated pins that nil, so lang_cmd.go:72 and :69 report success. Real binary, fresh directory, stdin redirected: define /lang es prints "now defining in es", exit 0, writes nothing, and the next define /lang says en. define /lang en prints "still defining in en, now on the record". A one-shot /lang has only a durable effect, so in the third state it is a no-op reporting a switch. The pre-#50 binary wrote lang.txt, and the no-directory case already refuses honestly (lang_cmd.go:57). TestLangAfterADeclineWritesNothing never reads stdout. Enumeration: lang_cmd.go:69 and :72 are reachable. The harvest, reflect, forget and play "saved/wrote/removed" messages all need a non-empty deck, which a declined one-shot cannot have. deckperm.go:263 prints "the lookup still works" on Ctrl-C, but in the REPL the same SIGINT fires detachedInterrupts' default cancel and the session exits with no lookup (scratch test: run returned 0). Docs asserting the old behaviour: cmd/define/README.md:649-651 and atlas/define.md:1128. Fix: persistLang reports a declined sentinel instead of nil; /lang says the language applies to this session only and was not saved; pin it on stdout. While in that README, :590 links the question to the Install anchor instead of the section that describes it.
  - id: new
    severity: Minor
    family: policy-restated-not-derived
    title: |
      deckPolicy re-tests opt.raw instead of consuming decideCapture, the single owner of "this invocation writes nothing"
    detail: |
      This is the 3rd finding in family policy-restated-not-derived (BR-12, BR-14). The rule: a decision this codebase already owns is consumed by calling its owner, never by re-testing the owner's inputs at a new site. BR-14's behaviour is fixed, but deckperm.go:210 re-tests opt.raw instead of asking decideCapture (capture.go:26), which capture.go:198 already consults for the same question. A third captureNothing member would reach the REPL's up-front question again. Enumerated: this is the only site in the diff restating decideCapture. main.go:806's noCapture check mirrors openStore's "anywhere to read at all" and is correct as is. One line: decideCapture(true, opt) == captureNothing, with reasonRaw renamed for the class.
```
