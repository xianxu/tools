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
