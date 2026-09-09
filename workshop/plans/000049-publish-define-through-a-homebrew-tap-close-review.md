# Boundary Review — tools#49 (whole-issue close)

| field | value |
|-------|-------|
| issue | 49 — publish define through a homebrew tap |
| repo | tools |
| issue file | workshop/issues/000049-publish-define-through-a-homebrew-tap.md |
| boundary | whole-issue close |
| milestone | — |
| window | 12aacfb887f7c58401aa0fa5b164b8ef00973ea0..644ab63ed4eecc7103df51f9a6e41a9317446c85 |
| command | sdlc close --issue 49 |
| reviewer | claude |
| timestamp | 2026-09-09T13:44:06-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The release mechanics are genuinely done and I verified them independently: the `v0.1.0` tag exists and is an ancestor of HEAD, the formula's `sha256` matches the tarball GitHub actually serves (I fetched and hashed it), and `go build -ldflags "-X main.version=v9.9.9"` really does produce `define v9.9.9` while a plain build says `define (built from source)`. What blocks SHIP is one thing: `-version` is a *dispatching mode* that was never added to `run()`'s collision list, so `define --version --play` and `define --version --forget cat` print the version, exit 0, and silently drop the other command — the exact failure this file spends three comment blocks and two guards preventing. The guard built to catch it (`TestEveryDispatchedModeIsInTheCollisionList`) only recognises `if *x { return <call>(…) }`, and the new dispatch is `return 0`, so it counted 6 modes and passed. I proved the blindness: adding `{"-version", *versionFlag}` to the list alone makes the guard fail with *"the dispatch parse found 6 modes … but the collision list declares 7 … this guard is missing a dispatch SHAPE."* The fix has to close both halves.

### 1. Strengths

- **The unstamped-build honesty is real, and pure.** `versionLine()` (`cmd/define/main.go:1345`) is a pure function with no IO, and `TestVersionAnswersWithNoDictionaryOrDeck` (`cmd/define/main_test.go:489`) runs the whole of `run()` with a literal `deps{}` — an actual empty-dependency assertion, not a mock reasserting the implementation. ARCH-PURE passes cleanly.
- **The checksum claim survives independent verification.** `curl` of `https://github.com/xianxu/tools/archive/refs/tags/v0.1.0.tar.gz` → `47f486f6a0c0…bfd8f`, byte-identical to the formula's `sha256`. `git merge-base --is-ancestor v0.1.0 644ab63` → yes, so the Log's reasoning about `gh pr merge --merge` keeping the tag reachable holds.
- **The version number has no third copy.** `grep` across `*.go`/`*.md`/`*.rb` finds `v0.1.0` only in the test's stamp fixture and in prose — nothing to bump. The atlas entry (`atlas/define.md:2605`) records *where* it lives rather than *what* it is, which is the right thing to write down.
- **The formula's `test do` asserts the stamp off the built binary**, not off a variable — the one place a broken ldflags path is currently caught.
- **The README documents the two failures only a real install finds** (`brew trust`, and homebrew-core's own `define` shadowing the unqualified name). Both are the kind of thing that makes a working formula look broken; `brew trust` is a real command on Homebrew 6.x, confirmed.

### 2. Critical findings

**`cmd/define/main.go:620–628` — `-version` dispatches as a mode but is absent from the collision list.**

Verified against the built binary in an empty directory:

```
define -version -play        → "define (built from source)"   exit 0
define -version -harvest     → "define (built from source)"   exit 0
define -version -forget cat  → "define (built from source)"   exit 0
define -version -llm-check   → "define (built from source)"   exit 0
```

Every one of those should be `exit 2` with *"…are both modes; run them separately"*. `main.go:598` states the rule and names its history: *"-harvest was refused beside -play and -reflect and silently swallowed by -forget and -llm-check, because those two dispatch above the switch and nobody enumerated them."* `-version` is now the seventh mode and the third instance of that same bug.

The reason the boundary went green is the guard's dispatch parse (`cmd/define/harvest_test.go:963-970`) requires the body to `return` a `*ast.CallExpr`; `return 0` is a `BasicLit`, so `-version` never enters `dispatched`, `len(dispatched) == len(listed) == 6`, and the floor check passes.

Fix sketch — both halves, or the guard stays blind:

```go
// beside runLLMCheck, so the dispatch shape is one the guard recognises
func runVersion(stdout io.Writer) int {
	fmt.Fprintln(stdout, versionLine())
	return 0
}
```
```go
modes := []mode{
	…
	{"-stats", *statsFlag},
	{"-version", *versionFlag},
}
…
if *versionFlag {
	return runVersion(stdout)
}
```
Add `{"-version", "-harvest"}` to `TestRunRefusesTwoModes`'s table (`harvest_test.go:679`) so the run()-path guard names it too.

### 3. Important findings

**`cmd/define/main_test.go:464` — the ldflags stamp path, which is the point of the issue, is pinned by nothing.** `TestVersionIsHonestAboutUnstampedBuilds` sets the Go variable directly, which cannot fail when the *linker* path breaks. Demonstrated in a scratch copy: rename `version` → `buildVersion`, and `go build -ldflags "-X main.version=v9.9.9"` emits no error, exits 0, and prints `define (built from source)` — a silently unversioned release — while the package stays green. Production flow (linker stamp) and test flow (Go assignment) do not share a boundary, which is precisely ARCH-MOCK's failure condition. The repo already has `go test -tags conformance ./...`; a `//go:build conformance` test that shells `go build -ldflags -X main.version=vTEST` into `t.TempDir()` and asserts `define vTEST` would fail on that rename in under a second.

**`README.md:3` — the repo front door still tells only the source-build story.** The root README leads with *"Small Go binaries — one job each, installed onto `$PATH`"* and offers `make build` / `make install`; `brew tap xianxu/tools` appears only in `cmd/define/README.md`. The issue's stated Problem is *"the thing that stops anyone else trying it is the install,"* and `7380263` made this repo public — a reader landing on github.com/xianxu/tools gets the clone-and-build path the issue exists to replace. Add the three brew lines to the `### define` section or to a short Install section above Build.

**`cmd/define/main.go:620` — `define --version <word>` silently swallows the word**, exit 0. Same shape as the pre-existing `define --llm-check sycophantic` (confirmed: prints the config report and exits 0), while `--stats`, `--play` and `--reflect` all refuse. The enumerable class is *modes dispatched above the argument-count switch* — currently exactly `-llm-check` and `-version`. Sweep both now rather than fixing the new site alone (ARCH-PURPOSE: the deliverable is the class, not the instance).

**The `define()` shell-function shadow trap is recorded only where nobody will read it.** The issue Log says it plainly — `construct/dev-aliases.sh` emits a function per `cmd/X`, functions outrank PATH in zsh, so a provisioned VM smoke-tests a local build instead of the bottle — and then says *"This trap applies to every `cmd/X` in every ariadne-styled peer, so it will recur for any future formula."* That Log archives to `workshop/history/`, which AGENTS.md instructs agents not to read. Per §4 it belongs in `workshop/lessons.md` (or a `workshop/targets/` invariant), where the next formula author will meet it.

### 4. Minor findings

- `cmd/define/README.md:22` and `:34` now carry two "macOS only" paragraphs ten lines apart with overlapping content (ARCH-DRY, docs); merge them and keep the `CoreServices` reasoning.
- `cmd/define/README.md:32-33` — stray double blank line left by the insertion.
- `fs.Usage`'s prose (`main.go:492`) names `--llm-check` but not `--version`; `PrintDefaults` covers it, so cosmetic only.
- `main_test.go:478` mutates package-level `version` with save/restore — safe today only because nothing in `cmd/define` calls `t.Parallel()`; worth one line saying so.

### 5. Test coverage notes

`go test ./cmd/define/` passes (110s). The two new tests are well-aimed at what they cover: one pins the pure function both ways, one pins the *envelope* (answers with no dictionary, no deck, no clock) rather than just the happy path. The gaps are the two above — nothing exercises `-version` beside another mode or beside a word, and nothing exercises the linker. A single table row in `TestRunRefusesTwoModes` plus one conformance test closes both.

### 6. Architectural notes

Per-marker: **ARCH-DRY** flag (minor, doc duplication; the version number itself is correctly single-sourced). **ARCH-PURE** pass. **ARCH-PURPOSE** flag (README front door; instance-not-class on the word swallow). The shadow-sweep on "one version, no third copy" comes back clean in the tree — but the derivation is *unenforced*, which is finding 2. **ARCH-MOCK** flag — Homebrew and GitHub are external dependencies and the only conformance check lives in the peer formula; `tools` never runs the seam production uses. **ARCH-CONSTRAINTS** pass — the "answers on a broken machine" envelope is declared in the comment and asserted by a test; the formula documents the ~150 MB first-install Go toolchain. **ARCH-SECURE** pass — release integrity pinned by a checksum verified against what GitHub serves, and `brew trust` presented as the trust decision it is rather than as boilerplate; no credentials or untrusted input in the diff. **ARCH-ORDER** pass on state (nothing carried between events); the one ordering that matters is dispatch order inside `run()`, and that is the Critical.

For the next formula: the mode-set guard is the most valuable thing in this package and it just failed open on a new dispatch *shape*. Widening the parse to accept any `return` in a flag-guarded `if` — not only a call — makes it shape-independent, so the eighth mode cannot repeat this.

### 7. Plan revision recommendations

None for accuracy — all four Plan items and all five Done-when items are genuinely delivered, and I verified the two that are checkable (the tag/checksum, and both stamp paths). No `## Revisions` entry is needed. Record the mode-set closure as a `## Log` line when the Critical is fixed.

```findings
findings:
  - id: new
    severity: Critical
    family: two-commands-one-line
    title: |
      -version dispatches as a mode but is absent from run()'s collision list, so it silently swallows every other mode
    detail: |
      Verified on the built binary: `define -version -play`, `-version -harvest`,
      `-version -forget cat` and `-version -llm-check` all print the version and
      exit 0, dropping the other command. main.go:598 states the rule and names
      two prior instances. TestEveryDispatchedModeIsInTheCollisionList missed it
      because its parse requires `return <call>` and the new dispatch is
      `return 0` — proven by adding the list row alone, which fails with
      "missing a dispatch SHAPE (found 6 … declares 7)". Fix both halves:
      extract `runVersion(stdout) int` so the dispatch shape is recognised, and
      add {"-version", *versionFlag} to modes.
  - id: new
    severity: Important
    family: cross-boundary-contract-untested
    title: |
      the -ldflags stamp path is pinned by no test; renaming main.version ships an unversioned release silently
    detail: |
      main_test.go:464 assigns the Go variable, which cannot fail when the linker
      path breaks. Demonstrated: renaming version to buildVersion leaves the
      package green while `go build -ldflags "-X main.version=v9.9.9"` exits 0 and
      prints "define (built from source)". Production flow (linker) and test flow
      (assignment) do not share a boundary. Add a //go:build conformance test that
      builds with -ldflags -X main.version=vTEST into t.TempDir() and asserts the
      output.
  - id: new
    severity: Important
    family: readme-front-door
    title: |
      root README.md still offers only make build / make install; the brew install appears only in cmd/define/README.md
    detail: |
      The issue's Problem is that the install is what stops anyone trying define,
      and 7380263 made the repo public. A reader landing on
      github.com/xianxu/tools gets the clone-and-build path the issue exists to
      replace. Add the three brew lines to the "### define" section or a short
      Install section above Build.
  - id: new
    severity: Important
    family: two-commands-one-line
    title: |
      define --version <word> silently swallows the word, as --llm-check already does; sweep the class not the instance
    detail: |
      Confirmed: `define -version sycophantic` and `define -llm-check
      sycophantic` both exit 0 honouring one of two commands, while --stats,
      --play and --reflect refuse. The enumerable class is modes dispatched
      ABOVE the argument-count switch — exactly -llm-check and -version. Fix
      both in this round (ARCH-PURPOSE).
  - id: new
    severity: Important
    family: discovery-lands-in-history
    title: |
      the dev-aliases define() shadow trap is recorded only in the issue Log, which archives to workshop/history
    detail: |
      The Log itself says the trap "applies to every cmd/X in every
      ariadne-styled peer, so it will recur for any future formula" — a
      generalisable rule filed in a directory AGENTS.md tells agents not to
      read. Per AGENTS.md 4 it belongs in workshop/lessons.md, or as a
      workshop/targets/ invariant.
  - id: new
    severity: Minor
    family: doc-restates-itself
    title: |
      cmd/define/README.md now has two "macOS only" paragraphs ten lines apart, plus a stray double blank line
    detail: |
      README.md:22 and :34 overlap; merge them keeping the CoreServices
      reasoning. Blank-line pair at :32-33 is insertion residue.
  - id: new
    severity: Minor
    family: usage-prose-lags-flags
    title: |
      fs.Usage prose names --llm-check but not --version
    detail: |
      PrintDefaults lists the flag, so this is consistency only.
  - id: new
    severity: Minor
    family: test-mutates-package-state
    title: |
      TestVersionIsHonestAboutUnstampedBuilds mutates package-level version, safe only because nothing here calls t.Parallel
    detail: |
      Confirmed no t.Parallel in cmd/define today. One line of comment records
      the dependency for whoever adds the first parallel test.
```

---

## Re-review — 2026-09-09T14:13:14-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 49 — publish define through a homebrew tap |
| repo | tools |
| issue file | workshop/issues/000049-publish-define-through-a-homebrew-tap.md |
| boundary | whole-issue close |
| milestone | — |
| window | 12aacfb887f7c58401aa0fa5b164b8ef00973ea0..512c7065a0fc1746fdc3d17c5b5f77f7a7ee3ed1 |
| command | sdlc close --issue 49 |
| reviewer | claude |
| timestamp | 2026-09-09T14:13:14-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 1's Critical is genuinely fixed and I mutation-verified every claim rather than reading the commit messages: dropping `{"-version", *versionFlag}` from `modes` now reddens `TestEveryDispatchedModeIsInTheCollisionList` with *"run() returns on -version, so it is a MODE"* (it passed before the parse was widened), deleting the two new switch arms reddens `TestModesAboveTheSwitchRefuseAWord`, and renaming `version` → `buildVersion` leaves both unit tests green while `TestLdflagsStampReachesTheBinary` is the only thing that fails — exactly the boundary BR-2 said was missing. `go test ./cmd/define/` is green (109s) and `go test -tags conformance -run Ldflags` passes. Nothing here blocks at the gate, but four things want a decision before the close is recorded: the tag the formula installs (`v0.1.0` = `7380263`) **predates** the Critical fix, so `brew install` today still ships the swallowing `--version`; the mode×word half of the collision family is still hand-enumerated after three rounds; the new conformance guard sits behind a build tag no gate runs; and the `.pyc` side-quest used the exact remedy its own guard's error message says does not work.

### 1. Strengths

- **The guard fix is the class, not the site.** `harvest_test.go:993-1013` widens the dispatch parse from `return <call>` to any single-valued `return` inside a bare-flag `if`. I checked the over-match claim independently: `dispatched` and `listed` are both 7 today, and the only non-mode bool flags (`raw`, `no-color`, `no-audio`) never appear as `if *x { return … }` in `run()`. The claim in the comment holds.
- **BR-4 was swept as an enumeration, not patched at the named site.** Both `--version` and `--llm-check` moved *below* the argument-count switch rather than growing a bespoke word check, so the switch every other mode is judged by now judges them (`main.go:670-684`). That is the right shape.
- **`TestLdflagsStampReachesTheBinary` is a real conformance test.** It shells the actual toolchain with the actual `-X main.version=` flag and reads the answer off the built binary, and its `stampedVersion = "v9.9.9-conformance"` deliberately avoids becoming a third place the release number lives. I verified against the installed formula: `Formula/define.rb` passes `-X main.version=v#{version}` — the flag string matches, and the formula's own `test do` asserts the stamp off the binary.
- **The skip is routed.** `conformance.SkipOrFail` rather than a bare `t.Skip`, and I confirmed `TestEverySkipIsRoutedOrWaived` walks `.go` files textually, so it does see a file behind a build tag.
- **`versionLine()` is genuinely PURE** and `TestVersionAnswersWithNoDictionaryOrDeck` drives all of `run()` with a literal `deps{}` — no mocks, ARCH-PURE passes.

### 2. Critical findings

None.

### 3. Important findings

**I-1 — the released artifact predates the fix (`cmd/define/main.go:710`, new family `shipped-artifact-lags-the-fix`).** `git rev-list -n1 v0.1.0` → `7380263`. The two commits after it — `644ab63` (README install line) and `512c706` (the BR-1/BR-4 fix) — are not in the tarball the formula's `sha256` pins. So `brew install xianxu/tools/define` today installs a binary where `define --version --play` prints the version, exits 0 and drops `--play`, and `define --version cat` swallows the word. Closing #49 with Done-when #1 ticked ("installs a working `define`") records a claim about an artifact that does not contain this round's work. Re-tag `v0.1.1` after merge, bump `url` + `sha256` in `xianxu/homebrew-tools`, and say so in `## Log` — or tick the box with an explicit note that the fix lands in the next release.

**I-2 — the mode×word half is still hand-enumerated. This is the 3rd finding in family `two-commands-one-line`** (`cmd/define/main.go:632-696`, `harvest_test.go:703-728`). Do not fix another instance. The rule: **every row in `modes` refuses a trailing word, derived from the list rather than remembered.** The mode×mode half already closes this way — `TestModeCollision` builds every pair from `declaredModes(t)`, which is why a 7th mode gets pair coverage for free. The word half does not: `run()` carries six hand-written `case *X && fs.NArg() != 0:` arms, and the tests hand-list which modes they check. A mode added tomorrow joins the pair matrix automatically and gets *no* word coverage — which is precisely how `-version` shipped. Close it the same way: iterate `declaredModes(t)`, build argv per mode (the AST pass already distinguishes `fs.Bool` from `fs.String`, so `-forget` gets `{"-forget","x","cat"}` and the rest get `{name,"cat"}`), and assert exit 2. Measured prevalence: 3 rounds, 6 hand-written arms, 2 modes that reached production without one.

**I-3 — a depended-on contract is asserted but not enforced by anything a gate runs. This is the 2nd finding in family `cross-boundary-contract-untested`.** Rule: **a contract this code depends on must be pinned by a test that some gate actually executes.** Two measured instances:
  - `cmd/define/version_conformance_test.go:1` is behind `//go:build conformance`. I grepped `Makefile*`, `scripts/parallel-checks.sh`, `scripts/pre-merge-checks.sh` and `.github/workflows/merge-check.yml`: nothing runs `-tags conformance`, and nothing sets `CONFORMANCE_STRICT`. BR-2's fix therefore exists but is never run by the close or the merge — it protects only whoever remembers the command. (`#37` already tracks *where* `CONFORMANCE_STRICT` gets set; this boundary is the reason to land it.)
  - `main.go:706-712` and `atlas/define.md:2612` both assert `--version` answers *above* `withStore`. I moved the dispatch below `d = d.withStore(...)` in a scratch copy: the entire package stayed green, and `./define --version` in an empty dir behaved identically. The stated invariant is pinned by nothing, and the mutation shows the code does not actually depend on it — so either pin it or soften the prose to match.

**I-4 — the compiled-artifact guard cannot see the artifact this window removed (`cmd/define/repo_guard_test.go:24-28`, new family `guard-blind-to-an-unlisted-shape`).** `7380263` deleted `cmd/define/testdata/__pycache__/capture.cpython-314.pyc` in a follow-up commit — the exact remedy `TestNoBinariesInHistory`'s own message says does not work ("deleting it in a later commit does not remove the cost"). Blob `b1fc21e`, 7892 bytes, magic `2b 0e 0d 0a`, is still reachable from HEAD and fetched by every clone of a now-public repo, and the guard reports green because `executableMagics` knows only Mach-O and ELF. The cost is 7.9 KB, so I would **not** rewrite history (it would move `7380263` and strand the formula's tarball). The class-level fix is the guard: extend it to compiled artifacts by exact extension (`.pyc`, `.pyo`, `.class`, `.o`, `.a`, `.so`, `.dylib`, `.wasm`) alongside the magic test — extension is exact here and does not reintroduce the symlink false-positive the magic-byte comment rejected — then record the accepted 7.9 KB as known debt.

### 4. Minor findings

- `main.go:583-585` still says `--llm-check` "is dispatched before the argument count is judged"; `main.go:593` reads the same way in the present tense. **2nd in family `usage-prose-lags-flags`** — rule: a claim about *where* a mode dispatches belongs only at the dispatch site. Prevalence measured at 2 sites, both about the pair this commit moved; delete both and let `main.go:670` and `main.go:706` carry it.
- `harvest_test.go:713` — `{"-llm-check", "sycophantic"}` sets no environment, so if the guard ever regresses the test makes a live model call (I saw it: `http://127.0.0.1:8317`, 1.199s, real tokens). `llmcheck_test.go:135-141` already writes the rule down. Add the same three `t.Setenv` lines. (ARCH-MOCK.)
- **2nd in family `doc-restates-itself`**: the install recipe + its two caveat paragraphs now exist in `README.md:9-30`, `cmd/define/README.md:10-24` and the tap's README, and have already diverged — both in-repo copies say "Both of the *first two* lines are load-bearing" and then explain lines 1 and 3. Rule: one canonical copy; the root README should keep the three commands and link `cmd/define/README.md` for the why, or vice versa. Fix the sentence in whichever survives.
- `workshop/lessons.md:4185` — `[[validate-the-system-not-your-model-of-it]]` is the only `[[…]]` in the file and matches no heading. Either add the target or make it a plain pointer.
- The conformance build omits the formula's `GOFLAGS = "-trimpath -mod=readonly"`; harmless for the `-X` path, worth one line of comment saying so.

### 5. Test coverage notes

Coverage of what this window changed is good, and I verified it by reversion rather than by reading: three separate mutations each redden exactly one test and no others. The gap is not in what the tests assert but in **which of them anything runs** (I-3) and in **what derives the list they iterate** (I-2). One test-hygiene item: `TestVersionIsHonestAboutUnstampedBuilds` mutates the package var and documents the `t.Parallel` dependency — BR-8 addressed, and the comment is the right artifact.

### 6. Architectural notes

- **ARCH-DRY** — flag (I-4 guard list, Minor doc triplication). The switch arms are the file's established idiom, so I-2 is filed as a coverage rule, not a refactor demand.
- **ARCH-PURE** — pass. `versionLine()` is deterministic, `runVersion` is a two-line IO shell, tests need no mocks.
- **ARCH-PURPOSE** — flag (I-1). The shadow-sweep on the version single-source is otherwise clean: binary derives via `-ldflags`, atlas records *where* not *what*, conformance test uses its own fixture, no README states a number. The one consumer that does not derive from HEAD is the released tarball.
- **ARCH-MOCK** — pass with a Minor. The linker now has a seam and a live conformance check; the cadence is the gap (I-3).
- **ARCH-CONSTRAINTS** — pass. `--version` is O(1) with no IO; the conformance test shells one `go build` (~2.8s measured), bounded and tag-gated.
- **ARCH-SECURE** — pass. `--version` prints a compile-time constant and degrades visibly when unstamped rather than fabricating a number; the formula's `sha256` pins the tarball. The `.pyc` in public history (I-4) is hygiene, not exposure — it is a compiled testdata capture script.
- **ARCH-ORDER** — pass. `run()` holds no state between events; the conformance test's subprocess extent is lexically bounded by `t.Context()` through `CommandContext`.

### 7. Plan revision recommendations

The `## Plan` and `## Done when` boxes match the code. One `## Revisions` entry is warranted on the issue, for I-1: Done-when #1 and #2 are ticked against `v0.1.0` = `7380263`, which does not contain `512c706`'s fix — record either the follow-up `v0.1.1` re-tag + formula bump, or an explicit note that the released binary lags the branch and why that was accepted.
