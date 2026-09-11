---
gate: boundary-review
issue: 50
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-10T17:56:03-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: '"nothing reaches a declined directory" is pinned for 2 of 7 creating methods, and the guard fails open'
          detail: |-
            Verified by mutation in a scratch worktree: making SetItems write to g.disk AND
            route through g.creating() leaves the entire suite green while a declined
            directory gains an items/ tree. A scratch test listing the directory after a
            denied SetItems went red under the mutation and green after revert. Fix as a
            class: iterate storeInterfaceMethods (not the createsOnDisk map) with a
            sampleArgs table and assert the directory is byte-identical under denial.
          family: guard-fails-open
          round: 1
        - id: BR-2
          severity: Important
          title: the denied session's fallback store is per-wrapper, so a /lang rebuild discards it
          detail: |-
            newGatedStore allocates store.NewMem() per wrapper (gated_store.go:32) and
            newLangDeps rebuilds the wrapper (main.go:339). Measured: allowed keeps 1 word
            across a rebuild, denied drops to 0 — contradicting "a denied session still
            recalls itself". This is PQ-2's rule applied to the decision but not to the
            store it swaps in. Key the fallback by (dir, lang), memoized in openStore.
          family: process-scoped-state-per-wrapper
          round: 1
        - id: BR-3
          severity: Important
          title: M2 surface (-here, deckAsker) landed at the M1 boundary, inert and untested
          detail: |-
            -here prints in --help as "make this directory a deck without asking" but has
            no production reader (main.go:414,474,601); deckAsker (deckperm.go:132) has no
            caller and no committed test (deckasker_test.go is untracked). The new atlas
            section describes the question and the decline as current truth, which the
            committed binary cannot produce. M1 is stated to be a true no-op.
          family: unshipped-surface-claimed-as-current
          round: 1
        - id: BR-4
          severity: Important
          title: Done-when, the M1 Plan row, and the plan's Core-concepts table each state what the code deliberately does not do
          detail: |-
            Done-when says "every write-shaped method consults the gate" — PQ-8 made that
            false on purpose (Forget). The Plan row says "8 gated writes, 7 ungated reads";
            the code ships 7 gated / 8 ungated. The plan's table lists store.IsDeck as PURE
            though it calls os.ReadDir and its tests need a mutable filesystem. No
            "## Revisions" section exists in either artifact.
          family: artifact-claims-what-code-does-not
          round: 1
        - id: BR-5
          severity: Important
          title: README update missing for -here, and the plan has no README step at all
          detail: |-
            cmd/define/README.md is the binding user documentation and gains no mention of
            -here, which the Spec calls "the only path automation has". Task 10 Step 7
            names only the atlas, so nothing remaining in M2 will catch this before the
            merge-time specs judge.
          family: readme-gate
          round: 1
        - id: BR-6
          severity: Minor
          title: deckAsker reads the answer through a throwaway bufio.Reader over shared stdin
          detail: |-
            deckperm.go:149 may consume up to 4 KiB past the answer and discard it. M2 Task
            8 resolves the permission immediately before the REPL reads the same stdin, so
            a pasted "y\ndog\n" would lose the word. Read byte-wise to '\n', or share the
            loop's reader.
          family: stdin-over-read
          round: 1
        - id: BR-7
          severity: Minor
          title: the createsOnDisk/doesNotCreate split is hand-maintained; a coordinated reclassification is invisible
          detail: |-
            The AST guard forces every method into a bucket and the consultation test forces
            the bucket to match the code, but moving a method to the wrong bucket AND
            switching creating() to reading() together passes everything. The class-level
            denied-directory test above closes this as a side effect.
          family: guard-fails-open
          round: 1
        - id: BR-8
          severity: Minor
          title: storeInterfaceMethods ignores embedded interfaces and the "< 15" floor is hand-bumped
          detail: |-
            Only m.Names is read, so an embedded interface contributes zero methods. Deriving
            the floor from reflect.TypeOf((*store.Store)(nil)).Elem().NumMethod() would make
            it self-maintaining.
          family: derivation-under-derives
          round: 1
        - id: BR-9
          severity: Minor
          title: IsDeck uses os.ReadDir, which reads and sorts every entry in the working directory
          detail: |-
            Early exit does not help: ReadDir materialises and sorts the full list first. In a
            home directory or monorepo root that is a needless O(n log n) on the startup path.
            f.ReadDir(-1) unsorted avoids the sort. ARCH-CONSTRAINTS, once per process.
          family: startup-path-cost
          round: 1
      boundary: M1
      blocked: true
    - "n": 2
      timestamp: "2026-09-10T23:26:34-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'Re-measured: BR-1''s own mutation leaves the whole suite green; zero-value args make 5 of 7 subtests vacuous and the positive control waives per-method checking.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: Revert-verified — restoring the per-wrapper store.NewMem() reddens gated_store_test.go:405 with its own message.
          round: 2
        - id: BR-3
          disposition: addressed
          note: deckasker_test.go is committed, -here has production readers, and --here creates events/ unasked on the real binary.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: Done-when still says "write-shaped", the Plan row still says 8/7, IsDeck is still listed PURE, and neither artifact has a "## Revisions" section.
          round: 2
        - id: BR-5
          disposition: addressed
          note: Both READMEs document the question, the decline and --here; --help is a separate surface, raised below.
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: readLineUnbuffered is the right code but no test fails without it; one assertion on the unread remainder closes it.
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: The class test iterates createsOnDisk, not storeInterfaceMethods, so a coordinated bucket-move plus creating()->reading() is still invisible.
          round: 2
        - id: BR-8
          disposition: not-addressed
          note: storeInterfaceMethods still reads only m.Names and the floor is still a hand-typed "< 15".
          round: 2
        - id: BR-9
          disposition: not-addressed
          note: Still os.ReadDir, and settleQuietly now sits on the read path so it is no longer once per process while undecided (measured 5 evaluations for 5 reads; 1 in a real one-shot lookup).
          round: 2
      findings:
        - id: BR-10
          severity: Important
          title: third round of guard-fails-open — the rule is that an absence or ordering claim needs a per-instance control, not an aggregate one
          detail: |-
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
          family: guard-fails-open
          round: 2
        - id: BR-11
          severity: Important
          title: second round of readme-gate — --help still promises unconditional recording, and the README quotes a prompt string nothing keeps in step
          detail: |-
            This is the 2nd finding in family readme-gate. Do NOT fix only the instance.
            main.go:528-531's usage prose still says define "records what you look up under
            words/ and events/ in the CURRENT DIRECTORY … A word that was found is added to
            the deck", which is false in the third state; --help is the first surface a user
            types. The rule: when behaviour changes, enumerate every place the old behaviour
            is asserted and sweep them in one round — README.md, cmd/define/README.md,
            atlas/define.md, the fs.Usage prose, the issue's Done-when, the plan. Cheap
            structural half: cmd/define/README.md:52 quotes the literal string deckperm.go:195
            prints, and this package already owns doc_sync_test.go for exactly that class.
          family: readme-gate
          round: 2
        - id: BR-12
          severity: Important
          title: deckPolicy and deckAsker independently encode the same three-way precedence
          detail: |-
            deckperm.go:149-157 and :181-194 each implement already-a-deck / --here /
            no-terminal. They agree today and nothing makes them. Observable consequence
            already present: only deckAsker prints the "nothing will be saved (use --here)"
            explanation, so whether a piped user is told depends on which encoding settles the
            state first — confirmed on the real binary (echo word | define prints it,
            define --forget cat piped does not). ARCH-DRY: deckAsker should switch on
            deckPolicy and own only the deckUndecided arm.
          family: policy-restated-not-derived
          round: 2
        - id: BR-13
          severity: Minor
          title: renderStats gained a bare positional bool, read at six call sites as a literal true
          detail: |-
            stats.go:101's `saving bool` appears as renderStats(s, now, true) in stats_test.go
            and deckasker_test.go. A named type or a field makes the call sites self-describing.
          family: positional-bool-parameter
          round: 2
      blocked: true
---

# Gate ledger — tools#50 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-10T17:56:03-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `guard-fails-open` "nothing reaches a declined directory" is pinned for 2 of 7 creating methods, and the guard fails open
  Verified by mutation in a scratch worktree: making SetItems write to g.disk AND
  route through g.creating() leaves the entire suite green while a declined
  directory gains an items/ tree. A scratch test listing the directory after a
  denied SetItems went red under the mutation and green after revert. Fix as a
  class: iterate storeInterfaceMethods (not the createsOnDisk map) with a
  sampleArgs table and assert the directory is byte-identical under denial.
- **BR-2** [Important] `process-scoped-state-per-wrapper` the denied session's fallback store is per-wrapper, so a /lang rebuild discards it
  newGatedStore allocates store.NewMem() per wrapper (gated_store.go:32) and
  newLangDeps rebuilds the wrapper (main.go:339). Measured: allowed keeps 1 word
  across a rebuild, denied drops to 0 — contradicting "a denied session still
  recalls itself". This is PQ-2's rule applied to the decision but not to the
  store it swaps in. Key the fallback by (dir, lang), memoized in openStore.
- **BR-3** [Important] `unshipped-surface-claimed-as-current` M2 surface (-here, deckAsker) landed at the M1 boundary, inert and untested
  -here prints in --help as "make this directory a deck without asking" but has
  no production reader (main.go:414,474,601); deckAsker (deckperm.go:132) has no
  caller and no committed test (deckasker_test.go is untracked). The new atlas
  section describes the question and the decline as current truth, which the
  committed binary cannot produce. M1 is stated to be a true no-op.
- **BR-4** [Important] `artifact-claims-what-code-does-not` Done-when, the M1 Plan row, and the plan's Core-concepts table each state what the code deliberately does not do
  Done-when says "every write-shaped method consults the gate" — PQ-8 made that
  false on purpose (Forget). The Plan row says "8 gated writes, 7 ungated reads";
  the code ships 7 gated / 8 ungated. The plan's table lists store.IsDeck as PURE
  though it calls os.ReadDir and its tests need a mutable filesystem. No
  "## Revisions" section exists in either artifact.
- **BR-5** [Important] `readme-gate` README update missing for -here, and the plan has no README step at all
  cmd/define/README.md is the binding user documentation and gains no mention of
  -here, which the Spec calls "the only path automation has". Task 10 Step 7
  names only the atlas, so nothing remaining in M2 will catch this before the
  merge-time specs judge.
- **BR-6** [Minor] `stdin-over-read` deckAsker reads the answer through a throwaway bufio.Reader over shared stdin
  deckperm.go:149 may consume up to 4 KiB past the answer and discard it. M2 Task
  8 resolves the permission immediately before the REPL reads the same stdin, so
  a pasted "y\ndog\n" would lose the word. Read byte-wise to '\n', or share the
  loop's reader.
- **BR-7** [Minor] `guard-fails-open` the createsOnDisk/doesNotCreate split is hand-maintained; a coordinated reclassification is invisible
  The AST guard forces every method into a bucket and the consultation test forces
  the bucket to match the code, but moving a method to the wrong bucket AND
  switching creating() to reading() together passes everything. The class-level
  denied-directory test above closes this as a side effect.
- **BR-8** [Minor] `derivation-under-derives` storeInterfaceMethods ignores embedded interfaces and the "< 15" floor is hand-bumped
  Only m.Names is read, so an embedded interface contributes zero methods. Deriving
  the floor from reflect.TypeOf((*store.Store)(nil)).Elem().NumMethod() would make
  it self-maintaining.
- **BR-9** [Minor] `startup-path-cost` IsDeck uses os.ReadDir, which reads and sorts every entry in the working directory
  Early exit does not help: ReadDir materialises and sorts the full list first. In a
  home directory or monorepo root that is a needless O(n log n) on the startup path.
  f.ReadDir(-1) unsorted avoids the sort. ARCH-CONSTRAINTS, once per process.

## Round 2 — 2026-09-10T23:26:34-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Re-measured: BR-1's own mutation leaves the whole suite green; zero-value args make 5 of 7 subtests vacuous and the positive control waives per-method checking.
- BR-2 — addressed — Revert-verified — restoring the per-wrapper store.NewMem() reddens gated_store_test.go:405 with its own message.
- BR-3 — addressed — deckasker_test.go is committed, -here has production readers, and --here creates events/ unasked on the real binary.
- BR-4 — not-addressed — Done-when still says "write-shaped", the Plan row still says 8/7, IsDeck is still listed PURE, and neither artifact has a "## Revisions" section.
- BR-5 — addressed — Both READMEs document the question, the decline and --here; --help is a separate surface, raised below.
- BR-6 — not-addressed — readLineUnbuffered is the right code but no test fails without it; one assertion on the unread remainder closes it.
- BR-7 — not-addressed — The class test iterates createsOnDisk, not storeInterfaceMethods, so a coordinated bucket-move plus creating()->reading() is still invisible.
- BR-8 — not-addressed — storeInterfaceMethods still reads only m.Names and the floor is still a hand-typed "< 15".
- BR-9 — not-addressed — Still os.ReadDir, and settleQuietly now sits on the read path so it is no longer once per process while undecided (measured 5 evaluations for 5 reads; 1 in a real one-shot lookup).

### Raised

- **BR-10** [Important] `guard-fails-open` third round of guard-fails-open — the rule is that an absence or ordering claim needs a per-instance control, not an aggregate one
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
- **BR-11** [Important] `readme-gate` second round of readme-gate — --help still promises unconditional recording, and the README quotes a prompt string nothing keeps in step
  This is the 2nd finding in family readme-gate. Do NOT fix only the instance.
  main.go:528-531's usage prose still says define "records what you look up under
  words/ and events/ in the CURRENT DIRECTORY … A word that was found is added to
  the deck", which is false in the third state; --help is the first surface a user
  types. The rule: when behaviour changes, enumerate every place the old behaviour
  is asserted and sweep them in one round — README.md, cmd/define/README.md,
  atlas/define.md, the fs.Usage prose, the issue's Done-when, the plan. Cheap
  structural half: cmd/define/README.md:52 quotes the literal string deckperm.go:195
  prints, and this package already owns doc_sync_test.go for exactly that class.
- **BR-12** [Important] `policy-restated-not-derived` deckPolicy and deckAsker independently encode the same three-way precedence
  deckperm.go:149-157 and :181-194 each implement already-a-deck / --here /
  no-terminal. They agree today and nothing makes them. Observable consequence
  already present: only deckAsker prints the "nothing will be saved (use --here)"
  explanation, so whether a piped user is told depends on which encoding settles the
  state first — confirmed on the real binary (echo word | define prints it,
  define --forget cat piped does not). ARCH-DRY: deckAsker should switch on
  deckPolicy and own only the deckUndecided arm.
- **BR-13** [Minor] `positional-bool-parameter` renderStats gained a bare positional bool, read at six call sites as a literal true
  stats.go:101's `saving bool` appears as renderStats(s, now, true) in stats_test.go
  and deckasker_test.go. A named type or a field makes the call sites self-describing.

## Open findings

- **BR-1** [Important] `guard-fails-open` "nothing reaches a declined directory" is pinned for 2 of 7 creating methods, and the guard fails open
- **BR-4** [Important] `artifact-claims-what-code-does-not` Done-when, the M1 Plan row, and the plan's Core-concepts table each state what the code deliberately does not do
- **BR-6** [Minor] `stdin-over-read` deckAsker reads the answer through a throwaway bufio.Reader over shared stdin
- **BR-7** [Minor] `guard-fails-open` the createsOnDisk/doesNotCreate split is hand-maintained; a coordinated reclassification is invisible
- **BR-8** [Minor] `derivation-under-derives` storeInterfaceMethods ignores embedded interfaces and the "< 15" floor is hand-bumped
- **BR-9** [Minor] `startup-path-cost` IsDeck uses os.ReadDir, which reads and sorts every entry in the working directory
- **BR-10** [Important] `guard-fails-open` third round of guard-fails-open — the rule is that an absence or ordering claim needs a per-instance control, not an aggregate one
- **BR-11** [Important] `readme-gate` second round of readme-gate — --help still promises unconditional recording, and the README quotes a prompt string nothing keeps in step
- **BR-12** [Important] `policy-restated-not-derived` deckPolicy and deckAsker independently encode the same three-way precedence
- **BR-13** [Minor] `positional-bool-parameter` renderStats gained a bare positional bool, read at six call sites as a literal true
