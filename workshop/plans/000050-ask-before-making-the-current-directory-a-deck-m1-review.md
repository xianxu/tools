# Boundary Review — tools#50 (milestone M1)

| field | value |
|-------|-------|
| issue | 50 — ask before making the current directory a deck |
| repo | tools |
| issue file | workshop/issues/000050-ask-before-making-the-current-directory-a-deck.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 3ce76e8b43d4995f8d78e70523c671327763d75a..3d520a53400717189c5efa9a6bf3660139088d0b |
| command | sdlc milestone-close --issue 50 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-10T17:56:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1's core claim holds: the seam is real, wired at every `Store` path plus the one free function a wrapper cannot reach, and the whole suite is green at the pinned commit (`go build`/`go vet`/`go vet -tags conformance`/`gofmt` all clean; `go test ./...` ok, 112s for `cmd/define`). The derivations are genuinely derived — `IsDeck` off `RuntimeDirs`+`RuntimeFiles`, the gated set off the `Store` interface by AST — and I confirmed by mutation that the denied conformance run is a real oracle (routing `SetItems` to `disk` under denial reddens 7 subtests). What blocks SHIP is that the guard protecting the issue's central Done-when ("answering no leaves the directory untouched") is verified for 2 of 7 creating methods: I applied a dual-write mutation (`_ = g.disk.SetItems(...)` then `g.creating().SetItems(...)`) and **every test in this diff stayed green while a declined directory grew an `items/` tree** — the exact `guard-fails-open` shape #49 paid four review rounds for. Second, the denied session's fallback store is per-wrapper, so a `/lang` rebuild silently discards it (measured: allowed keeps 1 word, denied drops to 0) — that is PQ-2's own rule applied to the decision but not to the store it swaps in. Everything else is Important-but-cheap or documentation truth.

## 1. Strengths

- **`store/isdeck.go:44-60`** — the PQ-5 fix is real and the test earns it. `os.ReadDir` once, `filepath.Match` against entry **names**, so the cwd never enters a pattern; `isdeck_test.go:103-129` tests the file branch (not the directory branch) in `br[acket`/`sta*r`/`quest?ion`/`bra]ce`, which is the only shape that reaches the globbing. The `## Log` entry admitting the first draft passed under its own defect is exactly the discipline the repo's lessons ask for.
- **`gated_store_test.go:187-202`** — running the wrapper through `storetest.Suite` **in both modes** is the strongest available delegation check, and it is not decorative: I verified it reddens under a mis-routing mutation. Denied conforming is the evidence that "a declined store is empty, not broken."
- **`gated_store.go:70-78` (`Forget`)** — classifying by *what a method does to the disk* rather than by being write-shaped is the right call, and the reasoning (`--forget` would ask permission to create a deck in order to delete nothing) is recorded where the next reader will hit it.
- **`deckperm.go:18-24`** — a three-valued `deckDecision` instead of `decided bool` + `allowed bool`, with the reason written down. Textbook ARCH-ORDER: the illegal `{false, true}` is unrepresentable rather than merely unreached.
- **`main.go:331` / `:339` / `:366-372`** — `flat` gated as well as the per-language store, the wrap inside `newLangDeps` (not `withStore`), and `persistLang` gated explicitly. `TestPersistLangIsGated` verifies by **listing the directory**, not by reading the code.
- **`workshop/lessons.md`** — the unasserted-string-replacement lesson, and its mutation-testing corollary, is a genuinely load-bearing addition.

## 2. Critical findings

None.

## 3. Important findings

**(a) The "nothing reaches a declined directory" guard is pinned per-instance, not per-class — and it fails open. `cmd/define/gated_store_test.go:152,250`**

`TestDeniedWritesNeverReachTheDisk` covers `Upsert` (against a `Mem` backing, so it cannot see the filesystem at all) and `TestTheNewsCacheCannotCreateWhenDenied` covers `SetNewsItems`. The other five creating methods have no assertion that a declined directory stays empty. Verified by mutation in a scratch worktree:

```go
func (g *gatedStore) SetItems(key string, items []store.Item) error {
	_ = g.disk.SetItems(key, items)          // reaches the declined directory
	return g.creating().SetItems(key, items) // …and still consults + routes correctly
}
```
→ `go test ./cmd/define/` **passes**, including both conformance runs and both classification guards. A scratch test doing `newGatedStore(store.NewYAML(dir,…), deny)` → `SetItems` with a real item → `lsNames(dir)` went **red** (`a declined SetItems created [items]`) under the mutation and green after reverting.

Fix sketch: one reflection-driven test iterating `storeInterfaceMethods(t)` — **not** the `createsOnDisk` map — asserting that under a denying permission over a real `store.NewYAML(t.TempDir(),…)`, calling any method leaves the directory byte-identical. Back it with a `sampleArgs map[string][]any` and a guard that every derived method has a sample (zero-value args do not make YAML write, which is why `callStoreMethod` cannot be reused as-is). That single test closes this and (b) below.

**(b) The denied session's fallback store is per-wrapper, so `/lang` discards it. `cmd/define/gated_store.go:32`, `cmd/define/main.go:339`**

`newGatedStore` builds `store.NewMem()` per wrapper, and `newLangDeps` — the closure `/lang` re-invokes — builds a fresh wrapper. Measured:

```
allow=true   before=1  after-rebuild=1
allow=false  before=1  after-rebuild=0   ← the session forgot itself
```

(rebuilding for the *same* language is enough; an actual `es`→`en` round-trip is the user path). This contradicts `gated_store.go:20` ("still recalls what it did"), the plan's Task 3 acceptance ("a denied session still recalls itself") and the atlas's "an empty deck, not a missing one". It is precisely the PQ-2 argument — per-wrapper state does not survive the `/lang` rebuild — applied to the decision but not to the store the decision swaps in. Fix: give the fallback the same identity the disk store has, `(dir, lang)`, by memoizing it in `openStore` (`map[store.Lang]store.Store`, `flat` taking the `DefaultLang` entry) and passing it into `newGatedStore`. Note in a comment that events are language-blind on disk but per-`Mem` in memory — a residual fidelity gap a single shared `Mem` would fix at the cost of merging decks across languages, which is worse.

**(c) M2's user-visible surface landed at the M1 boundary, inert and untested. `cmd/define/main.go:414,474,601`, `cmd/define/deckperm.go:132`**

M1 is stated to be "no user-visible behaviour change at all" — the property it was split out to prove. But `-here` is registered and prints in `--help` as *"make this directory a deck without asking"*, while `opt.here` has exactly one reader: `deckAsker`, which has **no production caller** at this commit. `define -here` today does nothing. `deckAsker` itself is ~35 lines of M2 policy committed with no test in the window (`cmd/define/deckasker_test.go` is untracked in the working tree), and it appears in none of the five M1 Plan rows or the `## Log`. The new `atlas/define.md` section likewise describes the question, the decline, and the three states as current truth, which the committed binary cannot produce — the repo has an explicit rule that atlas/README prose describes the tool *as it is* (`repo_guard_test.go:632-650`). Cheapest disposition: commit the asker test with M2 and note in `## Log` that the flag + asker landed early; if M1 is meant to stay a true no-op, move them to the M2 commit.

**(d) Durable artifacts state what the code deliberately does not do, with no `## Revisions`. `workshop/issues/000050-…-deck.md:194-196,210`, `workshop/plans/000050-…-plan.md:37`**

Three instances of one rule:
- Done-when: *"Every **write-shaped** method on `store.Store` consults the gate"* — PQ-8 deliberately made this false (`Forget` mutates and is ungated by design). The close gate will read this line.
- Plan row: *"8 gated writes, 7 ungated reads"* — the code ships **7 gated / 8 ungated** (`Forget` moved buckets).
- Core-concepts table lists `store.IsDeck` under **Pure entities**, but it calls `os.ReadDir` and its tests need a mutable filesystem (`t.TempDir`, `os.MkdirAll`, `os.WriteFile`). The review contract's default for a table/code contradiction is Critical; I am lowering it one notch because the entity is correct, fast, and injected only at the boundary — the defect is the label, not the design. It belongs in **Integration points** (it is the design's only non-`Store` filesystem read).

Per AGENTS.md, append a `## Revisions` section to both artifacts rather than overwriting.

**(e) README gate: `-here` is new user-facing surface and the plan has no README step at all. `cmd/define/README.md`**

`cmd/define/README.md` is the binding user documentation (the guard binds by `/README.md` suffix). `-here` gains help text in this range and becomes, per the Spec, *"the only path automation has"* — a reader who pipes into `define` will need it. The plan's Task 10 Step 7 names only the atlas, so nothing in the remaining M2 work will catch this; the gap would surface at the merge-time `specs` judge.

## 4. Minor findings

- `deckperm.go:149` — `bufio.NewReader(in).ReadString('\n')` is a throwaway buffered reader over shared stdin; it may consume up to 4 KiB past the answer and discard it. Harmless while nothing calls it, but M2 Task 8 resolves this *immediately before* the REPL starts reading the same stdin — a pasted `y⏎dog⏎` would lose `dog`. Read byte-wise to `\n`, or hand it the reader the loop will use.
- `gated_store_test.go:23-31` — the `createsOnDisk`/`doesNotCreate` split is hand-maintained. The AST guard forces every method into *a* bucket and the consultation test forces the bucket to match the code, but moving a method to the wrong bucket *and* changing `creating()`→`reading()` together passes everything. Finding (a)'s class-level test (iterating the interface, not the map) closes this too.
- `gated_store_test.go:36-62` — `storeInterfaceMethods` reads only `m.Names`, so an embedded interface contributes zero methods; the `< 15` floor is a hand-bumped magic number that would not catch a small under-derivation. Consider deriving the floor from `reflect.TypeOf((*store.Store)(nil)).Elem().NumMethod()`.
- `store/isdeck.go:33` — `os.ReadDir` reads *and sorts* every entry before the first `Match`; in a home directory or monorepo root that is a needless O(n log n) on the startup path. `f.ReadDir(-1)` (unsorted) or a chunked loop with early exit would avoid it. ARCH-CONSTRAINTS, startup-path only, once per process — note for later, not now.
- `gated_store.go:32` — `store.NewMem()` is allocated for every wrapper even when the permission allows. Trivial, but it disappears for free with fix (b).

## 5. Test coverage notes

- `isdeck_test.go` and `deckperm_test.go` are strong: derived tables, no IO in the permission tests, and each asserts the *negative* (`saving()` must not consult `ask`) rather than only the happy path.
- The mutation record in `## Log` is credible where I spot-checked it. The Log's own admission that two wiring tests passed while asserting nothing, and were replaced, is the right outcome recorded the right way.
- The gap is uniform and named in (a): **consultation** is pinned mechanically, **routing** is pinned behaviourally by the conformance suite, but **"no bytes reached the declined directory"** — the Done-when's own wording — is pinned for 2 of 7 methods and fails open for the other 5.
- `deckAsker` has zero committed coverage in this window.
- Nothing yet exercises a `/lang` switch under denial; that fixture would have caught (b), and M2 Task 10 already promises `/lang es` after a decline — extend it to switch *back*.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass (with the Minor above).** `RuntimeDirs`/`RuntimeFiles` reuse is exemplary; the shadow-sweep of non-`Store` write paths checks out — I enumerated every `MkdirAll`/`WriteFile`/`writeBytesAtomic` under `cmd/define/store/` and the only two outside a `Store` method are `WriteLang` (gated) and `MigrateToLanguages` (pinned). `main.go:1232`'s `os.WriteFile` writes into `os.MkdirTemp("")`, not the deck. The enumeration in the plan is complete.
- **ARCH-PURE — flag (d).** Policy in the command, bytes in the store, decision travelling as a value: correct. Only the table's label is wrong.
- **ARCH-PURPOSE — flag (a), (b).** Both are the instance-vs-class axis: a guard written for the two sites that motivated it, and PQ-2's rule applied to the decision but not to the object it swaps in. Answer both as classes, in this round.
- **ARCH-MOCK — pass.** `store.Mem` is a stateful fake behind the same seam, held to the same conformance suite as `YAML`, and production and test flow share that boundary. This is what makes the third state cheap.
- **ARCH-CONSTRAINTS — pass.** One question per process, resolved lazily, no fan-out. Only the `ReadDir` Minor.
- **ARCH-SECURE — pass.** The cwd is treated as input the code did not choose and is kept out of glob patterns; no credentials; tests use `t.TempDir`/`t.Chdir` and cannot reach real user state; unreadable-directory degrades to `false`, whose consequence is *ask*, not a fabricated answer.
- **ARCH-ORDER — pass at M1, with an M2 warning.** The tagged enum is right. But `deckperm.go:35` documents "NOT SAFE FOR CONCURRENT USE" and discharges it with a pre-resolution that **does not exist yet** (M2 Task 8). At M1 that is sound only because the production permission is always nil. When Task 8 lands, the per-shell tests it specifies are the minimum, and they can only observe the interleaving they happened to take — add a seam (or run the loop tests under `-race`) so the claim is falsifiable rather than confirmed by one sample.

## 7. Plan revision recommendations

Append a `## Revisions` section to **both** artifacts (timestamp + reason + delta, per AGENTS.md):

- `workshop/issues/000050-…-deck.md` — (i) Done-when bullet 6: replace *"Every write-shaped method"* with *"Every **creating** method … `Forget` is write-shaped and deliberately ungated (PQ-8)"*; (ii) Plan M1 row 2: *"8 gated writes, 7 ungated reads"* → *"7 gated creating writes, 8 ungated (7 reads + `Forget`)"*; (iii) record that `-here` and `deckAsker` (Task 7 surface) landed in the M1 commits.
- `workshop/plans/000050-…-plan.md` — (i) move `store.IsDeck` from **Pure entities** to **Integration points** (it is the design's only non-`Store` filesystem read; its tests require a mutable fs); (ii) add to Task 6 or a new M1 task the class-level *"under denial, any `Store` method leaves the directory byte-identical"* test, derived from the interface rather than from `createsOnDisk`; (iii) add the `(dir, lang)`-keyed fallback store to the Task 5 wiring, with the `/lang`-round-trip test; (iv) add a README step to Task 10 (`cmd/define/README.md` currently appears nowhere in the plan).

```findings
findings:
  - id: new
    severity: Important
    family: guard-fails-open
    title: |
      "nothing reaches a declined directory" is pinned for 2 of 7 creating methods, and the guard fails open
    detail: |
      Verified by mutation in a scratch worktree: making SetItems write to g.disk AND
      route through g.creating() leaves the entire suite green while a declined
      directory gains an items/ tree. A scratch test listing the directory after a
      denied SetItems went red under the mutation and green after revert. Fix as a
      class: iterate storeInterfaceMethods (not the createsOnDisk map) with a
      sampleArgs table and assert the directory is byte-identical under denial.
  - id: new
    severity: Important
    family: process-scoped-state-per-wrapper
    title: |
      the denied session's fallback store is per-wrapper, so a /lang rebuild discards it
    detail: |
      newGatedStore allocates store.NewMem() per wrapper (gated_store.go:32) and
      newLangDeps rebuilds the wrapper (main.go:339). Measured: allowed keeps 1 word
      across a rebuild, denied drops to 0 — contradicting "a denied session still
      recalls itself". This is PQ-2's rule applied to the decision but not to the
      store it swaps in. Key the fallback by (dir, lang), memoized in openStore.
  - id: new
    severity: Important
    family: unshipped-surface-claimed-as-current
    title: |
      M2 surface (-here, deckAsker) landed at the M1 boundary, inert and untested
    detail: |
      -here prints in --help as "make this directory a deck without asking" but has
      no production reader (main.go:414,474,601); deckAsker (deckperm.go:132) has no
      caller and no committed test (deckasker_test.go is untracked). The new atlas
      section describes the question and the decline as current truth, which the
      committed binary cannot produce. M1 is stated to be a true no-op.
  - id: new
    severity: Important
    family: artifact-claims-what-code-does-not
    title: |
      Done-when, the M1 Plan row, and the plan's Core-concepts table each state what the code deliberately does not do
    detail: |
      Done-when says "every write-shaped method consults the gate" — PQ-8 made that
      false on purpose (Forget). The Plan row says "8 gated writes, 7 ungated reads";
      the code ships 7 gated / 8 ungated. The plan's table lists store.IsDeck as PURE
      though it calls os.ReadDir and its tests need a mutable filesystem. No
      "## Revisions" section exists in either artifact.
  - id: new
    severity: Important
    family: readme-gate
    title: |
      README update missing for -here, and the plan has no README step at all
    detail: |
      cmd/define/README.md is the binding user documentation and gains no mention of
      -here, which the Spec calls "the only path automation has". Task 10 Step 7
      names only the atlas, so nothing remaining in M2 will catch this before the
      merge-time specs judge.
  - id: new
    severity: Minor
    family: stdin-over-read
    title: |
      deckAsker reads the answer through a throwaway bufio.Reader over shared stdin
    detail: |
      deckperm.go:149 may consume up to 4 KiB past the answer and discard it. M2 Task
      8 resolves the permission immediately before the REPL reads the same stdin, so
      a pasted "y\ndog\n" would lose the word. Read byte-wise to '\n', or share the
      loop's reader.
  - id: new
    severity: Minor
    family: guard-fails-open
    title: |
      the createsOnDisk/doesNotCreate split is hand-maintained; a coordinated reclassification is invisible
    detail: |
      The AST guard forces every method into a bucket and the consultation test forces
      the bucket to match the code, but moving a method to the wrong bucket AND
      switching creating() to reading() together passes everything. The class-level
      denied-directory test above closes this as a side effect.
  - id: new
    severity: Minor
    family: derivation-under-derives
    title: |
      storeInterfaceMethods ignores embedded interfaces and the "< 15" floor is hand-bumped
    detail: |
      Only m.Names is read, so an embedded interface contributes zero methods. Deriving
      the floor from reflect.TypeOf((*store.Store)(nil)).Elem().NumMethod() would make
      it self-maintaining.
  - id: new
    severity: Minor
    family: startup-path-cost
    title: |
      IsDeck uses os.ReadDir, which reads and sorts every entry in the working directory
    detail: |
      Early exit does not help: ReadDir materialises and sorts the full list first. In a
      home directory or monorepo root that is a needless O(n log n) on the startup path.
      f.ReadDir(-1) unsorted avoids the sort. ARCH-CONSTRAINTS, once per process.
```
