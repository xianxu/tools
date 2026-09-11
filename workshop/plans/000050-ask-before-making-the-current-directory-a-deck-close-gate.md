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

## Open findings

- **BR-1** [Important] `guard-fails-open` "nothing reaches a declined directory" is pinned for 2 of 7 creating methods, and the guard fails open
- **BR-2** [Important] `process-scoped-state-per-wrapper` the denied session's fallback store is per-wrapper, so a /lang rebuild discards it
- **BR-3** [Important] `unshipped-surface-claimed-as-current` M2 surface (-here, deckAsker) landed at the M1 boundary, inert and untested
- **BR-4** [Important] `artifact-claims-what-code-does-not` Done-when, the M1 Plan row, and the plan's Core-concepts table each state what the code deliberately does not do
- **BR-5** [Important] `readme-gate` README update missing for -here, and the plan has no README step at all
- **BR-6** [Minor] `stdin-over-read` deckAsker reads the answer through a throwaway bufio.Reader over shared stdin
- **BR-7** [Minor] `guard-fails-open` the createsOnDisk/doesNotCreate split is hand-maintained; a coordinated reclassification is invisible
- **BR-8** [Minor] `derivation-under-derives` storeInterfaceMethods ignores embedded interfaces and the "< 15" floor is hand-bumped
- **BR-9** [Minor] `startup-path-cost` IsDeck uses os.ReadDir, which reads and sorts every entry in the working directory
