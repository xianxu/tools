# Ask Before Making the Current Directory a Deck — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `define` asks before it turns the current directory into a deck; when it cannot ask, it creates nothing, still answers lookups, and reads as an empty deck.

**Architecture:** One **decision**, resolved at most once per process and shared by everything that could write. A `store.Store` wrapper carries it to the store; the `persistLang` closure carries it to the one write path that is not a `Store` method. On denial the wrapper serves `store.Mem`, so "reads as empty" costs no changes at the eight sites that refuse a `nil` deck.

**Tech Stack:** Go 1.26 (toolchain 1.27.1), `cmd/define` (command + IO seams), `cmd/define/store` (YAML + Mem, both run through `storetest.Suite`).

> **Plan-quality round 1 (9 findings) is folded in.** The corrections that changed the
> DESIGN, not just the prose: the decision moved out of the wrapper into a shared object
> (PQ-2), `persistLang` is gated explicitly (PQ-1), pre-resolution moved above the
> loop-shell choice (PQ-3), `IsDeck` stopped putting a path into a glob pattern (PQ-5),
> and the gated set is classified by **"creates"** rather than "is write-shaped" (PQ-8).

---

## Non-goals

Stated so a later round does not reopen them (PQ-10):

- **Not changing `DEFINE_NO_CAPTURE`.** It keeps its `nil` deck and its exit 1. An explicit instruction deserves a direct answer; an accident deserves graceful degradation. The divergence is intended.
- **Not teaching the `nil`-deck sites a new meaning.** Serving `store.Mem` is what makes those sites need no changes.
- **Not adding a marker file.** `IsDeck` derives from artifacts that already exist; a `.deck` file would be a new runtime artifact to gitignore, migrate and explain.
- **Not gating reads.** A read creates nothing, so a question about creating is a false alarm — and false alarms train people to answer yes.
- **Not remembering the answer on disk.** It lives for the process. Persisting a "no" would be new state to invalidate when the user changes their mind.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `deckDecision` | `cmd/define/deckperm.go` | new |
| `deckPermission` | `cmd/define/deckperm.go` | new |
| `renderStats` | `cmd/define/stats.go` | modified |

- **`store.IsDeck(dir string) bool`** — has `define` already made this directory its own?
  - **Relationships:** 1:1 with a directory. Derives from `RuntimeDirs` + `RuntimeFiles`.
  - **DRY rationale:** those two lists already enumerate everything `define` writes, and they grew three times across three issues with each growth finding a place that had not been told. A predicate that re-listed them would be the fourth.
  - **Adversarial input (PQ-5):** the directory comes from `os.Getwd()` and **must never become part of a glob pattern**. A cwd containing `[` makes `filepath.Glob` return `ErrBadPattern`, the branch is skipped, an established deck reads as *not* a deck — and a "yes" then `MkdirAll`s over a live deck. `*`/`?` could match a sibling. Strategy: `os.ReadDir(dir)` once, then `filepath.Match(pat, entry.Name())` against **entry names only**.
  - **Any artifact counts, not all.** A directory holding only `audio/` is one `define` wrote into; asking to create a deck there asks about something that exists.

- **`deckDecision`** — `deckUndecided` / `deckAllow` / `deckDeny`.
  - **DRY rationale:** three values, not `decided bool` + `allowed bool`, which makes `{false, true}` representable and meaningless.

- **`deckPermission`** — **the** decision, resolved at most once per process.
  - **Relationships:** 1:N. ONE instance; every `gatedStore` and the `persistLang` closure hold a pointer.
  - **Why not a field on the wrapper (PQ-2):** a session builds **two** YAML stores — `flat` (`main.go:323`, backing `newStoreHistory` and `newCachingFeed`) and the per-language `st` (`main.go:326`) — and `/lang` rebuilds `langDeps` (`command.go:426`), producing more. A per-wrapper state would ask once *per wrapper*: answer, switch language, get asked again. Worse, a fresh wrapper starts `deckUndecided` and discards the pre-resolution.
  - **Three operations, and the third is the subtle one:**
    - `allowed() bool` — resolve (memoized) and answer. Called by creating writes.
    - `resolve()` — force resolution now. Called once above the loop-shell choice.
    - `saving() (bool, bool)` — **read the state WITHOUT resolving**, returning `(allowed, decided)`. This is what `--stats` uses (PQ-6); reading via `allowed()` would make `--stats` prompt, the very false alarm this design prevents.

- **`renderStats`** — the empty screen stops promising a deck that will not exist.
  - Its new input comes from `deckPermission.saving()` threaded through `printStats` (PQ-6). Not re-derived in `stats.go`: restating the policy there would be a second answer to "is anything being saved".

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `store.IsDeck` | `cmd/define/store/isdeck.go` | new | the filesystem |
| `gatedStore` | `cmd/define/gated_store.go` | new | `store.Store` |
| `deckAsker` | `cmd/define/deckperm.go` | new | stdin + stderr |
| `openStore` | `cmd/define/main.go:269` | modified | working directory |
| `persistLang` closure | `cmd/define/main.go:350` | modified | `store.WriteLang` |
| loop-shell choice | `cmd/define/repl.go:229` | modified | the terminal |

- **`gatedStore`** — implements all 15 `Store` methods; the **creating** ones consult the permission.
  - **Injected into:** both YAML stores `openStore` builds, so `newStoreHistory` and the caching feed are covered. Wrapping only the per-language one lets `cachingFeed.SetNewsItems` (`news.go:133` → `yaml.go:561` `MkdirAll`) bypass the gate (PQ-2).
  - **Why a wrapper, not a change to `YAML`:** the store package's only UI is a `warn io.Writer` — it must not prompt. Policy in the command, bytes in the store (ARCH-PURE).

- **`persistLang` closure** — gated explicitly, because a `Store` wrapper **cannot** reach it.
  - **This is PQ-1, and the previous draft's Architecture paragraph was false:** a wrapper catches every disk-creating operation *that is a `Store` method*. `store.WriteLang` is a free function (`lang.go:98` → `writeBytesAtomic`), reached from `openStore`'s closure at `main.go:350` and handed to a one-shot `/lang` at `command.go:232`. Ungated, `define /lang es` writes `lang.txt` having asked nothing — and so does `/lang` after a decline.
  - **The complete enumeration of non-`Store` write paths** (PQ-1 asks for it):
    1. `store.WriteLang` via `persistLang` — **gated here**.
    2. `store.MigrateToLanguages` (`main.go:299`) — **provably safe, not gated**: `migrateFlatDeck` returns `nil` before any `MkdirAll` when `words/` is absent (`migrate.go:44-48`), and again when no flat `.yaml` files exist. If `words/` *does* exist the directory is already a deck, so `IsDeck` is true and the permission allows silently. Pinned by a test in Task 5 rather than trusted as prose.

- **`deckAsker`** — prints the question, reads a line.
  - **Reuses the terminal predicate that exists.** `deps.stdinIsTerminal` (`main.go:95`) is already injected — *"Injected rather than"* calling `isTerminal(os.Stdin)`. Do **not** add a bare `tty bool`: this repo has three distinct terminal questions (`opt.tty` for stdout at `main.go:403`, `d.stdinIsTerminal`, colour) and `repl.go:228` already computes `terminalUI`. Conflating two is a bug it has made three times (PQ-4).

- **loop-shell choice (`repl.go:229`)** — resolves the permission ONCE, before either loop starts.
  - **Above the `replRaw`/`replLines` branch, not inside `runEditor` (PQ-3).** Both shells read stdin on their own goroutine — `replRaw` owns the key channel, `replLines` reads through `scanLines` (`repl.go:396`) — so a prompt firing mid-loop races the reader in **either**. `replraw.go:23` and `:28` also fall back to `replLines`, so a fix in `runEditor` alone misses the fallback.

**Test surface.** `IsDeck`: table over `t.TempDir()`, including a directory whose **name contains `[`**. `deckPermission`: pure, no terminal — a `func() bool` double. `gatedStore`: `storetest.Suite` plus a recording permission. The classification guard is an AST test over the `Store` interface, in the shape of `declaredModes` (`harvest_test.go:893`).

---

## Chunk 1: the seam (M1)

M1 ships with a permission that always allows. **No user-visible behaviour change** — so the wiring is proven before any policy rides on it. A **nil** permission means allow, which keeps the ~15 existing `deps{newStore: openStore}` sites behaving exactly as today.

### Task 1: `store.IsDeck`

**Files:** create `cmd/define/store/isdeck.go`, `cmd/define/store/isdeck_test.go`

```go
func IsDeck(dir string) bool
```

- [ ] **Step 1: failing test.** One subtest per `RuntimeDirs` entry (proves derivation); one per runtime-file family including the per-language `user-model.??.md` pattern; an empty dir; a missing dir; and **a directory whose name contains `[` holding a real deck** (PQ-5 — the row that fails if the path reaches a pattern).
- [ ] **Step 2:** run, watch it fail (`undefined: IsDeck`).
- [ ] **Step 3: implement.** `os.ReadDir(dir)` once; a `RuntimeDirs` hit is an entry that `IsDir()`; a `RuntimeFiles` hit is `filepath.Match(pat, entry.Name())`. **Skip `tmpPattern`** — the atomic-write shadow is debris from a crashed write, and treating it as an established deck would skip the question. Unreadable dir → `false` (safe direction: the response to false is to *ask*).
- [ ] **Step 4:** run, watch it pass.
- [ ] **Step 5: mutate.** Replace `Match` with `strings.HasPrefix` — the `user-model.??.md` row must redden. Revert. Append `"scratch"` to `RuntimeDirs` — a new subtest must pass with no edit to `isdeck.go`. Revert.
- [ ] **Step 6:** `git commit -m "#50 M1: a deck is any artifact define wrote, derived not listed"`

### Task 2: `deckPermission`

**Files:** create `cmd/define/deckperm.go`, `cmd/define/deckperm_test.go`

```go
type deckDecision int // deckUndecided | deckAllow | deckDeny

func newDeckPermission(ask func() bool) *deckPermission
func (p *deckPermission) allowed() bool        // resolves, memoized
func (p *deckPermission) resolve()             // force resolution now
func (p *deckPermission) saving() (bool, bool) // (allowed, decided) — NEVER resolves
```

- [ ] **Step 1: failing test.** `allowed()` twice consults `ask` once; a **nil receiver and a nil `ask`** both mean allow (this is what keeps M1 a no-op); `saving()` on an undecided permission reports `decided == false` and **does not** consult `ask`; `resolve()` makes a later `saving()` report the answer.
- [ ] **Step 2–4:** fail, implement, pass.
- [ ] **Step 5: mutate.** Make `saving()` call `allowed()` — the "does not consult" assertion must redden. That is the PQ-6 defect; if it stays green the test is decorative.
- [ ] **Step 6:** `git commit -m "#50 M1: one decision, resolved at most once"`

### Task 3: `gatedStore`

**Files:** create `cmd/define/gated_store.go`, `cmd/define/gated_store_test.go`

```go
func newGatedStore(disk store.Store, perm *deckPermission) *gatedStore
var _ store.Store = (*gatedStore)(nil)
```

- **Strategy.** Reads delegate to `disk` unless the permission is decided-deny, in which case they serve the in-memory fallback. Creating writes call `perm.allowed()` first and route to `disk` or the fallback. **A declined write is not an error** — the user was asked and answered; an error would put "could not save" in front of someone who said not to.
- **Classification is by "creates", not "is write-shaped" (PQ-8).** `Forget` only `os.Remove`/`os.RemoveAll` (`yaml.go:789, 820`) — no `MkdirAll` — so gating it makes `define --forget x` in a non-deck directory ask permission to **create** a deck in order to delete nothing: exactly the false alarm the lazy rule prevents. `Forget` is ungated, and the buckets are named `createsOnDisk` / `doesNotCreate` so the next reader is not tempted by "write".

- [ ] **Step 1: failing test.** Reads never consult the permission; N creating writes consult it once; denial reaches the backing store not at all (asserted against the *backing* store, not the wrapper); a denied session still recalls itself; `Forget` does not consult it.
- [ ] **Step 2–4:** fail, implement, pass.
- [ ] **Step 5:** `git commit -m "#50 M1: the wrapper asks before it creates"`

### Task 4: the classification guard

**Files:** modify `cmd/define/gated_store_test.go`

- **Strategy.** Parse `type Store interface` out of `cmd/define/store/store.go` with `go/parser` — the `declaredModes` shape (`harvest_test.go:893`) — and assert every method lands in exactly one of `createsOnDisk` / `doesNotCreate`. An unknown method **fails**; defaulting is how a set nobody chose gets certified. A floor (`< 15`) catches an under-deriving parse. A second test calls each `createsOnDisk` method by reflection and asserts the permission was consulted.

- [ ] **Step 1–4:** failing test, helpers (`storeInterfaceMethods`, `callStoreMethod`), pass.
- [ ] **Step 5: MUTATE IN EVERY SHAPE THE CLAIM HAS** (#49 `guard-fails-open` — *mutation-verify a guard against the invariant it NAMES, in its general form, not the single reproduction that motivated it*). Four shapes, each must redden alone:
  1. Add a 16th `Store` method → classification fails naming it.
  2. Move a method between buckets → the both/neither check fails.
  3. Make one `createsOnDisk` method skip the permission → the consultation test fails **for that method**.
  4. Make the parse return a hardcoded list → the floor or shape 1 fails.
  Record all four in `## Log`. A shape that stays green means the guard is decorative.
- [ ] **Step 6:** `git commit -m "#50 M1: the creating set is derived from the interface"`

### Task 5: wire it in, always allowing

**Files:** modify `cmd/define/main.go` (`deps.newStore`, `openStore`, `withStore`, `persistLang`)

- **Arity, stated once so Tasks 4 and 5 cannot disagree (PQ-4):**
  ```go
  // both change TOGETHER
  newStore func(options, io.Writer, *deckPermission) storeDeps
  func openStore(opt options, warn io.Writer, perm *deckPermission) storeDeps
  ```
  The ~15 sites mentioning `newStore` (`lang_cmd_test.go:149`, `command_test.go:137`, `harvest_test.go:894`, …) **assign** `openStore` rather than call it, so they keep compiling untouched when both sides change together. The only invocation is `withStore`, passing `d.deckPermission`. A **nil** permission allows, so every existing test is unaffected.

- **Where the wrap goes, and why not one level up.** Inside `newLangDeps` — the closure `/lang` re-invokes — because its own comment states the rule: *"anything constructed here is necessarily re-derived there. Adding a member cannot be half done."* Wrapping in `withStore` would give a switched language an **ungated** deck: decline, type `/lang es`, and it starts writing. Wrap `flat` too (PQ-2), or the caching feed's `SetNewsItems` bypasses the gate.

- **`persistLang` is gated here (PQ-1):** the closure consults the same permission before `store.WriteLang`. Declining means the language applies to the session and is not persisted — consistent with everything else being session-only.

- [ ] **Step 1: failing tests.** (a) the deck is a `*gatedStore`; (b) **`flat`'s consumers are gated too** — assert the history store's backing is wrapped; (c) `persistLang` under a denying permission writes no `lang.txt`, verified by listing the directory; (d) **`MigrateToLanguages` creates nothing in a non-deck directory** — the claim the Integration-points section makes, pinned rather than asserted in prose.
- [ ] **Step 2–4:** fail, implement, pass.
- [ ] **Step 5: run the whole suite.** `go test ./...` must be green. Red here means the seam changed behaviour it should not have.
- [ ] **Step 6:** `git commit -m "#50 M1: every write path carries the same decision"`

### Task 6: conformance

**Files:** modify `cmd/define/gated_store_test.go`

- **Strategy.** `storetest.Suite` (`storetest/suite.go`, 38 KB) already defines correct `Store` behaviour and runs over both `YAML` (`yaml_test.go:19`) and `Mem` (`mem_test.go:11`). A wrapper's only failure mode is delegating wrong, so run it through the same suite — allowed (must behave exactly like what it wraps) and denied (must behave like an empty store that keeps the session).
- [ ] **Step 1–2:** write, run. **If the denied case fails**, the suite asserts a durability property the fallback cannot honour — a finding about the design, not a test to weaken. Stop and record it.
- [ ] **Step 3:** `git commit -m "#50 M1: the wrapper answers to the same suite both stores do"`

- [ ] **M1 — `sdlc milestone-close --issue 50 --milestone M1`**

---

## Chunk 2: the policy (M2)

### Task 7: `--here` and the question

**Files:** modify `cmd/define/main.go` (flag, `options.here`); modify `cmd/define/deckperm.go`

```go
func deckAsker(dir string, opt options, in io.Reader, out io.Writer, stdinIsTerminal func() bool) func() bool
```

- **Four inputs, in this order, and the order is the design:** already a deck → yes, silently (nothing to ask about); `--here` → yes, silently (**the only path automation has**, since a non-terminal no longer gets a deck); not a terminal → no, with a warning (a question nobody can answer must not become a hang); otherwise ask, **defaulting to no** (a wrong "yes" is a stray deck in someone's home directory; a wrong "no" is re-running one command).
- [ ] **Step 1: failing table** over those four, plus bare Enter declines, EOF mid-question declines, `--here` with no terminal, an existing deck never asked about. Each row asserts **both** whether it asked and what it answered.
- [ ] **Step 2–4:** fail, implement, pass.
- [ ] **Step 5:** `git commit`

### Task 8: resolve above the loop-shell choice

**Files:** modify `cmd/define/repl.go` (~229, above the `replRaw`/`replLines` branch)

- **Strategy (PQ-3).** One `perm.resolve()` before either shell starts. Both read stdin on their own goroutine, so a prompt firing mid-loop races the reader in either; and `replraw.go:23`/`:28` fall back to `replLines`, so resolving inside `runEditor` misses the fallback.
- [ ] **Step 1: failing test** driving **each** loop shell, asserting the permission was already decided before the shell read its first key. One test per shell — a single test covers the shape only for whichever branch it happened to take.
- [ ] **Step 2–4:** fail, implement, pass.
- [ ] **Step 5:** `git commit`

### Task 9: the honest empty screen

**Files:** modify `cmd/define/stats.go` (`printStats`, `renderStats`), `cmd/define/play_loop.go`

- **Strategy (PQ-6).** `printStats` gains the `(allowed, decided)` pair from `deckPermission.saving()` and passes it to `renderStats`. It **must not** call `allowed()` — that would make `--stats` prompt. When decided-deny, the empty screen says nothing is being saved and how to fix it, instead of *"look a word up and it joins your deck"*, which is false there and lands in front of exactly the confused user this feature exists for.
- [ ] **Step 1: failing test** — the empty screen must not contain "joins your deck" when not saving, and must say so; rendering with `decided == false` is unchanged from today.
- [ ] **Step 2–4:** fail, implement, pass.
- [ ] **Step 5: mutate.** Make `printStats` read the flag via `allowed()` — a test asserting `--stats` never prompts must redden.
- [ ] **Step 6:** `git commit`

### Task 10: end-to-end on a real directory

**Files:** create `cmd/define/deckperm_e2e_test.go`

Done-when says *verified by listing the directory afterwards, not by reading the code*.

- [ ] **Step 1: failing tests.** Declining leaves the directory byte-identical (list before/after) while the lookup still prints and exits 0; a piped lookup creates nothing and asks nothing; `--here` creates unasked; **`/lang es` after a decline writes no `lang.txt`** (the PQ-1 path, which a one-shot lookup test would not see); a second run in a now-real deck asks nothing.
- [ ] **Step 2–4:** fail, implement, pass.
- [ ] **Step 5: full verification.**
  ```bash
  go build ./... && go vet ./... && go vet -tags conformance ./cmd/define/ && gofmt -l cmd/ internal/
  go test ./...
  bash scripts/run-merge-checks.sh "$(git merge-base main HEAD)" HEAD
  ```
- [ ] **Step 6: manual smoke** in `$(mktemp -d)`: decline → `ls -a` empty; `--stats` says nothing is saved, exit 0; piped → silent, nothing created; `/lang es` → no `lang.txt`; `--here` → creates; re-run → no question.
- [ ] **Step 7: atlas** — the three states, that `IsDeck` derives, and that `persistLang` is the one non-`Store` write path carrying the decision.
- [ ] **Step 8:** `git commit`

- [ ] **M2 — `sdlc close --issue 50 --verified '<evidence>'`**

---

## Corrections to earlier claims (PQ-7)

Both were mine, and both are now checked against the code rather than argued:

- **Go version.** The module declares `go 1.26` (toolchain 1.27.1), not 1.24. `t.Chdir` is available — confirmed by compiling it in a scratch module — so that risk is closed, not open. The earlier draft left it OPEN because an unasserted string replacement silently failed to apply, which is its own lesson.
- **The `nil`-deck sites are 8 refusals and 3 degradations, not 11 refusals.** Measured per site: `ask.go:267` returns a context without the learner model, `cloze.go:190` returns a nil question, and `main.go:176` is an *assignment* inside `withStore` — not a check at all. The design conclusion is unchanged (serving an empty store avoids all eleven), and the three degraders get strictly better: a real empty store means `ask.go` gets a `UserModel` call that succeeds and `cloze.go` an empty item list rather than a nil question.

## Risks

- **RESOLVED — `store.Mem` is a total `Store`.** `var _ Store = NewMem()` compiles, no stubs, and `storetest.Suite` runs over it *and* `YAML`. The denied path inherits asserted behaviour rather than a lookalike. The whole design rests on this.
- **RESOLVED — the audio cache takes the interface.** `newDiskAudioCache(st store.Store, …)`, field `st store.Store`, so `audio/` goes through the wrapper. Checked specifically: this seam once had `forWord` returning a concrete type, so a type assertion silently never matched and the disk cache did nothing.
- **RESOLVED — `MigrateToLanguages` creates nothing in a non-deck directory.** `migrate.go:44-48` returns before any `MkdirAll`. Pinned by a test in Task 5 rather than left as prose.
- **OPEN — the question goes to stderr while lookups go to stdout.** A prompt in a redirected stream is a hang with no visible cause. Task 7's table covers the non-tty case; verify by hand that `define word > out.txt` on a terminal still shows the question.
- **OPEN — `/lang` rebuilds `langDeps`.** Task 5 wraps inside `newLangDeps` so the rebuild is covered, but the *test* must actually perform a switch; asserting only the first build would pass while a switched language writes ungated.

## Revisions

**2026-09-10 — `store.IsDeck` moved from Pure entities to Integration points (BR-4).**
*Reason:* it calls `os.ReadDir` and its tests need a real mutable filesystem, which is
this skill's own definition of an integration point. The row's "PURE-with-IO note"
conceded the substance while the table kept claiming otherwise, and that mislabel
survived two review rounds.
*Delta:* listed under Integration points, wrapping the filesystem. Its tests still need
no mocks — which is why it looked pure.

**2026-09-10 — the gated set is 7 creates / 8 non-creates, not 8 writes / 7 reads
(BR-4).** *Reason:* PQ-8. `Forget` only removes, so it is ungated. *Delta:* the counts
above are corrected and derived from the interface by test.
