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

---

## Re-review — 2026-09-09T15:08:59-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 49 — publish define through a homebrew tap |
| repo | tools |
| issue file | workshop/issues/000049-publish-define-through-a-homebrew-tap.md |
| boundary | whole-issue close |
| milestone | — |
| window | 12aacfb887f7c58401aa0fa5b164b8ef00973ea0..18ebf80764b3c0ec3ed7409be50d4b844c7f7a86 |
| command | sdlc close --issue 49 |
| reviewer | claude |
| timestamp | 2026-09-09T15:08:59-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All eight prior findings are genuinely addressed, and I verified the five substantive ones by reverting them in a scratch copy rather than reading the commit messages: dropping `{"-version", *versionFlag}` now reddens `TestEveryDispatchedModeIsInTheCollisionList`; renaming `main.version` leaves both unit tests green while `TestLdflagsStampReachesTheBinary` alone fails; moving the `--version` dispatch below `withStore` reddens the position test; deleting the `-version` word arm reddens exactly that subtest of the derived `TestEveryModeRefusesATrailingWord`; and emptying `acceptedCompiledBlobs` reddens `TestNoBinariesInHistory` naming the `.pyc`. The built binary refuses `-version -play`, `-version -harvest`, `-version -forget cat`, `-version -llm-check`, `-version cat` and `-llm-check cat` — all exit 2. `go build`, `go vet`, `gofmt`, `go test ./...`, `GOOS=linux go vet -tags conformance`, and `bash scripts/run-merge-checks.sh` are all clean. Nothing here blocks SHIP. What holds it back from SHIP outright is four cheap Importants: a waiver whose scope is wider than its own justification (confirmed — a freshly staged `.pyc` with the same blob passes the *index* guard), a merge gate that exits 0 when its `-run` pattern matches nothing (measured), atlas prose that now contradicts the code in two places, and an issue whose Done-when boxes are ticked against a release tag that does not exist yet.

## 1. Strengths

- **`cmd/define/harvest_test.go:1053` — the guard was widened at the class, not the site.** Requiring `return <call>` was what let `-version` through; accepting *any* single-result `return` under a bare flag ident makes it shape-independent, and the "cannot over-match" argument holds — I checked `declaredModes`' two condition shapes and a validation branch like `*harvest && fs.NArg() != 0` is a `BinaryExpr` that never reaches the body walk.
- **`cmd/define/main_test.go:504` — a `newStore` that fails if it is built.** This is the difference between asserting an invariant and enforcing one: `deps{}` alone left `withStore` a no-op, so the position claim in `main.go` and `atlas/define.md` was decorative. Now the reviewer's own mutation reddens.
- **`cmd/define/version_conformance_test.go:39` — production and test flow finally share the linker.** It shells the real toolchain with `-X main.version=`, and the version it stamps is deliberately fake so the test does not become a third place the release number lives. That reasoning is right and rare.
- **`cmd/define/harvest_test.go:743` — the word-refusal half is now derived from `declaredModes`**, and it sets `DEFINE_LLM_*` so an `-llm-check` regression cannot quietly spend tokens. Both halves of that are good instincts.
- **`cmd/define/repo_guard_test.go:31` — the "exact extension, not a heuristic" distinction is correctly drawn.** The old comment rejected *guessing* from filenames; `.pyc` is not a guess. Extending by extension rather than by more magic bytes is the right call.

## 2. Critical findings

None.

## 3. Important findings

**I. `cmd/define/repo_guard_test.go:257` — the history waiver also silences the index guard.**
`acceptedCompiledBlobs` is consulted inside `scanForExecutables`, which both `TestNoCommittedBinaries` (index) and `TestNoBinariesInHistory` (history) call. The waiver's justification is purely historical — *"rewriting would move the commit `v0.1.0` tags"* — and that argument says nothing about the index, which `atlas/repo-guards.md` calls "the last moment the mistake is free." Confirmed by reproduction: staging `cmd/define/testdata/replanted.pyc` containing the identical blob `b1fc21e3` leaves `TestNoCommittedBinaries` green, while a *different* `.pyc` is caught (`blob 8eb0aaa3, 24 bytes`). Fix: make the waiver a parameter — `scanForExecutables(t, dir, want, waived map[string]string)` — passing `nil` from the index guard and `acceptedCompiledBlobs` from the history guard. Family: `waiver-wider-than-its-reason`.

**II. `scripts/merge-checks.d/10-release-stamp.sh:28` — the gate passes green when its check does not run.**
`go test -run <no match>` exits 0 (`ok … [no tests to run]`, measured). Rename or delete `TestLdflagsStampReachesTheBinary` and this merge check reports `✓ merge-check passed` having executed nothing — the exact "a skip reads as green" failure the script's own 20-line header argues against. `CONFORMANCE_STRICT` is also unset here, so a routed skip added inside that path later would read as pass too. Fix: `CONFORMANCE_STRICT=1 go test -tags conformance -run '^TestLdflagsStampReachesTheBinary$' -v -count=1 ./cmd/define/` piped through an assertion that `--- PASS: TestLdflagsStampReachesTheBinary` appeared. Family: `gate-green-without-running`.

**III. `atlas/repo-guards.md:32` and `atlas/define.md:2063` — the atlas now states two things the code contradicts. This is the 2nd finding in family `usage-prose-lags-flags`.**
Per the family rule I am not asking for these two edits — here is the rule.
Instances measured: (a) *"Both decide by **magic bytes** … not by filename"* is false as of `compiledExtensions`, and neither `acceptedCompiledBlobs` nor `TestAcceptedCompiledBlobsAreStillReachable` appears anywhere in `atlas/`; (b) *"run **on demand, not in CI** … neither belongs in `merge-check.yml`"* is false as of `10-release-stamp.sh`; (c) the 7-row conformance table omits `version_conformance_test.go` **and** `harvest_conformance_test.go` — it lags by 2 of 9, and has since before this window.
**Rule:** *a statement of a mechanism's decision rule belongs at the mechanism, and an atlas enumeration of a code-derivable set must be derived or pinned, never retyped.* Two mechanical consequences: move "what convicts a blob" into `scanForExecutables`' own doc comment and have `atlas/repo-guards.md` link rather than restate — the same move BR-7 made for `fs.Usage`, and `TestADocCommentNamesWhatItSitsOn` already enforces the co-location one level down; and pin the conformance table in `cmd/define/doc_sync_test.go` (which already pins counts in `atlas/define.md`) by globbing `*_conformance_test.go` + `live_property_test.go` and failing on any file with no table row. Item (c) is the evidence that a hand-typed table does not survive even one unrelated issue.

**IV. `cmd/define/harvest_test.go:679` — `TestRunRefusesTwoModes` is still a hand-typed mode set, and this window added three rows to it by hand. This is the 4th finding in family `two-commands-one-line`.**
Rows `{"-stats","-harvest"}`, `{"-version","-harvest"}`, `{"-version","-play"}` were typed in the same commit series that argued (correctly, at `harvest_test.go:731`) that a remembered enumeration is the defect. The file itself states why this table is not redundant: *"modeCollision being right is not the same claim as run() calling it."* So the through-`run()` pair claim is the one enumeration still carried by memory, and an 8th mode gets no row.
**Rule:** *every test that enumerates the mode SET iterates `declaredModes(t)`; a mode-name string literal inside a test table is the defect, not a missing row.* (Per-mode behaviour tests naming one mode — `stats_test.go:229`, `capture_test.go:236` — are outside it; the rule is about set-enumerations.) Measured prevalence after I-2: exactly one hand-enumerated mode set remains, this one. Enforceable form, if you want it mechanical: an AST guard over `*_test.go` failing on any composite literal holding ≥2 names from `declaredModes` inside a func that does not call `declaredModes`.

**V. `workshop/issues/000049-publish-define-through-a-homebrew-tap.md:106,113` — Done-when #1 and #4 are ticked against a release that does not exist.**
The issue's own `## Revisions` says #1 is satisfied by `v0.1.1`; `git tag -l` shows only `v0.1.0 -> 7380263`, and the installed `xianxu/homebrew-tools/Formula/define.rb` still pins `url … v0.1.0.tar.gz` / `sha256 47f486f6…` — the tarball that ships the BR-1 defect. Closing with these boxes ticked records a verified claim about an artifact this issue's own review disqualified. Fix: untick #1 (or annotate it *pending `v0.1.1`*), add the tag + formula bump + reinstall as an explicit post-merge `## Plan` row, and keep it out of `--verified`. Family: `checkbox-outruns-artifact`.

## 4. Minor findings

- `cmd/define/harvest_test.go:707` — `stringFlags` is a near-verbatim copy of the `fs.Bool` walker at `harvest_test.go:965`; they differ only in `"String"` vs `"Bool"` and the map shape. ARCH-DRY: one `flagsDeclaredWith(t, kind)`.
- `scripts/merge-checks.d/10-release-stamp.sh` ignores the `$BASE $HEAD` the runner passes it, so it rebuilds `cmd/define` on every PR including docs-only ones. Either scope it, or say in the header that always-run is deliberate.
- `cmd/define/stats_test.go:240` is now covered by `TestEveryModeRefusesATrailingWord/-stats`; harmless, but it is the shape the derived test replaces.
- `version_conformance_test.go:53`'s `-ldflags` string is a hand restatement of `Formula/define.rb` in the peer tap; nothing here derives it. This would be a repeat in family `cross-boundary-contract-untested`, so recording rather than raising — and the formula's own `test do` asserts `define v#{version}`, which pins it from the other side.

## 5. Test coverage notes

Coverage on the code this window ships is strong and, unusually, *demonstrated* — every fix I checked reddens under a realistic mutation, not just under a contrived one (the `version` → `buildVersion` rename is the kind of refactor someone would actually do, and only the conformance test noticed). Full suite green in 112s; `-tags conformance -run Ldflags` green in ~4s. The gaps are all one layer out from the code: the merge gate that runs the conformance test cannot tell "passed" from "did not exist" (Important II), the index half of the compiled-artifact guard has a content-keyed hole (Important I), and the through-`run()` pair matrix is the one mode enumeration still typed by hand (Important IV). `TestAcceptedCompiledBlobsAreStillReachable` does what it claims — I confirmed it fails on a bogus sha.

## 6. Architectural notes

- **ARCH-DRY — flag (Minor).** The version number has exactly one source (tag → formula → linker), and the mode list one source; the duplication is the two AST walkers.
- **ARCH-PURE — pass.** `versionLine()` is pure and unit-tested without IO; `runVersion(io.Writer)` is the thin shell; the position test injects a failing seam rather than mocking.
- **ARCH-PURPOSE — flag (Importants III, IV, V).** The shadow-sweep over "the mode set" and "the conformance suite set" found two hand-maintained restatements left; the sweep over "the version number" found none. V is the purpose itself: the issue is *"brew install works"*, and the published artifact still predates the fix.
- **ARCH-MOCK — pass.** The linker is correctly treated as an external dependency: real toolchain, real flag, real binary, and a gate that runs it. The only unmodelled surface is the peer formula, mitigated by its `test do`.
- **ARCH-CONSTRAINTS — flag (Minor).** `-run Ldflags` scoping deliberately keeps live model and dictionary calls out of the gate, and the derived word test pins `DEFINE_LLM_*` so a regression cannot spend tokens — both good. The gate is unconditioned on the diff range.
- **ARCH-SECURE — pass.** `version` is linker-supplied; no credential reaches a log, argv or fixture; the conformance build writes only into `t.TempDir()`. Worth noting that Important I *is* a provenance error in miniature: the waiver keys on content hash when its justification is about a commit, and content is exactly the wrong key for "this specific historical debt."
- **ARCH-ORDER — pass, non-vacuously.** `run()` holds no state between events: it is one-shot and returns on the first mode it dispatches, so there is no `(state, event)` space for `--version` to enter. The one ordering that *does* matter — dispatch position relative to `withStore` — is now pinned by a seam that fails when crossed, which is the right shape.

## 7. Plan revision recommendations

- **`## Revisions` — "Done-when #1/#4 remain unticked until `v0.1.1` exists."** Reason: the Revisions entry already says #1 is satisfied by `v0.1.1`, but the checkboxes above it still claim satisfaction, and only `v0.1.0` is tagged. Delta: untick #1 and #4, add `- [ ] Tag v0.1.1 on main after merge; bump url + sha256 in xianxu/homebrew-tools; reinstall from the tap to verify` to `## Plan`, and keep that claim out of `sdlc close --verified`.
- **`## Revisions` — "the mode enumeration is derived at every consumer except `TestRunRefusesTwoModes`."** Reason: the round-2 Log states the enumeration problem as solved ("an eighth mode is covered the day its row is added"), which is true of the word half and the pair half in `modeCollision`, but not of the through-`run()` pair table — and this window added rows to it by hand. Delta: record the remaining consumer and the rule that covers it (Important IV), so the next mode does not have to rediscover which half was derived.
