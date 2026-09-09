---
gate: boundary-review
issue: 49
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-09T13:44:06-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: -version dispatches as a mode but is absent from run()'s collision list, so it silently swallows every other mode
          detail: |-
            Verified on the built binary: `define -version -play`, `-version -harvest`,
            `-version -forget cat` and `-version -llm-check` all print the version and
            exit 0, dropping the other command. main.go:598 states the rule and names
            two prior instances. TestEveryDispatchedModeIsInTheCollisionList missed it
            because its parse requires `return <call>` and the new dispatch is
            `return 0` — proven by adding the list row alone, which fails with
            "missing a dispatch SHAPE (found 6 … declares 7)". Fix both halves:
            extract `runVersion(stdout) int` so the dispatch shape is recognised, and
            add {"-version", *versionFlag} to modes.
          family: two-commands-one-line
          round: 1
        - id: BR-2
          severity: Important
          title: the -ldflags stamp path is pinned by no test; renaming main.version ships an unversioned release silently
          detail: |-
            main_test.go:464 assigns the Go variable, which cannot fail when the linker
            path breaks. Demonstrated: renaming version to buildVersion leaves the
            package green while `go build -ldflags "-X main.version=v9.9.9"` exits 0 and
            prints "define (built from source)". Production flow (linker) and test flow
            (assignment) do not share a boundary. Add a //go:build conformance test that
            builds with -ldflags -X main.version=vTEST into t.TempDir() and asserts the
            output.
          family: cross-boundary-contract-untested
          round: 1
        - id: BR-3
          severity: Important
          title: root README.md still offers only make build / make install; the brew install appears only in cmd/define/README.md
          detail: |-
            The issue's Problem is that the install is what stops anyone trying define,
            and 7380263 made the repo public. A reader landing on
            github.com/xianxu/tools gets the clone-and-build path the issue exists to
            replace. Add the three brew lines to the "### define" section or a short
            Install section above Build.
          family: readme-front-door
          round: 1
        - id: BR-4
          severity: Important
          title: define --version <word> silently swallows the word, as --llm-check already does; sweep the class not the instance
          detail: |-
            Confirmed: `define -version sycophantic` and `define -llm-check
            sycophantic` both exit 0 honouring one of two commands, while --stats,
            --play and --reflect refuse. The enumerable class is modes dispatched
            ABOVE the argument-count switch — exactly -llm-check and -version. Fix
            both in this round (ARCH-PURPOSE).
          family: two-commands-one-line
          round: 1
        - id: BR-5
          severity: Important
          title: the dev-aliases define() shadow trap is recorded only in the issue Log, which archives to workshop/history
          detail: |-
            The Log itself says the trap "applies to every cmd/X in every
            ariadne-styled peer, so it will recur for any future formula" — a
            generalisable rule filed in a directory AGENTS.md tells agents not to
            read. Per AGENTS.md 4 it belongs in workshop/lessons.md, or as a
            workshop/targets/ invariant.
          family: discovery-lands-in-history
          round: 1
        - id: BR-6
          severity: Minor
          title: cmd/define/README.md now has two "macOS only" paragraphs ten lines apart, plus a stray double blank line
          detail: |-
            README.md:22 and :34 overlap; merge them keeping the CoreServices
            reasoning. Blank-line pair at :32-33 is insertion residue.
          family: doc-restates-itself
          round: 1
        - id: BR-7
          severity: Minor
          title: fs.Usage prose names --llm-check but not --version
          detail: PrintDefaults lists the flag, so this is consistency only.
          family: usage-prose-lags-flags
          round: 1
        - id: BR-8
          severity: Minor
          title: TestVersionIsHonestAboutUnstampedBuilds mutates package-level version, safe only because nothing here calls t.Parallel
          detail: |-
            Confirmed no t.Parallel in cmd/define today. One line of comment records
            the dependency for whoever adds the first parallel test.
          family: test-mutates-package-state
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-09T14:13:14-07:00"
      agent: claude
      blocked: true
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-09-09T15:08:59-07:00"
      agent: claude
      blocked: true
      protocol_error: no valid findings block
---

# Gate ledger — tools#49 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-09T13:44:06-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `two-commands-one-line` -version dispatches as a mode but is absent from run()'s collision list, so it silently swallows every other mode
  Verified on the built binary: `define -version -play`, `-version -harvest`,
  `-version -forget cat` and `-version -llm-check` all print the version and
  exit 0, dropping the other command. main.go:598 states the rule and names
  two prior instances. TestEveryDispatchedModeIsInTheCollisionList missed it
  because its parse requires `return <call>` and the new dispatch is
  `return 0` — proven by adding the list row alone, which fails with
  "missing a dispatch SHAPE (found 6 … declares 7)". Fix both halves:
  extract `runVersion(stdout) int` so the dispatch shape is recognised, and
  add {"-version", *versionFlag} to modes.
- **BR-2** [Important] `cross-boundary-contract-untested` the -ldflags stamp path is pinned by no test; renaming main.version ships an unversioned release silently
  main_test.go:464 assigns the Go variable, which cannot fail when the linker
  path breaks. Demonstrated: renaming version to buildVersion leaves the
  package green while `go build -ldflags "-X main.version=v9.9.9"` exits 0 and
  prints "define (built from source)". Production flow (linker) and test flow
  (assignment) do not share a boundary. Add a //go:build conformance test that
  builds with -ldflags -X main.version=vTEST into t.TempDir() and asserts the
  output.
- **BR-3** [Important] `readme-front-door` root README.md still offers only make build / make install; the brew install appears only in cmd/define/README.md
  The issue's Problem is that the install is what stops anyone trying define,
  and 7380263 made the repo public. A reader landing on
  github.com/xianxu/tools gets the clone-and-build path the issue exists to
  replace. Add the three brew lines to the "### define" section or a short
  Install section above Build.
- **BR-4** [Important] `two-commands-one-line` define --version <word> silently swallows the word, as --llm-check already does; sweep the class not the instance
  Confirmed: `define -version sycophantic` and `define -llm-check
  sycophantic` both exit 0 honouring one of two commands, while --stats,
  --play and --reflect refuse. The enumerable class is modes dispatched
  ABOVE the argument-count switch — exactly -llm-check and -version. Fix
  both in this round (ARCH-PURPOSE).
- **BR-5** [Important] `discovery-lands-in-history` the dev-aliases define() shadow trap is recorded only in the issue Log, which archives to workshop/history
  The Log itself says the trap "applies to every cmd/X in every
  ariadne-styled peer, so it will recur for any future formula" — a
  generalisable rule filed in a directory AGENTS.md tells agents not to
  read. Per AGENTS.md 4 it belongs in workshop/lessons.md, or as a
  workshop/targets/ invariant.
- **BR-6** [Minor] `doc-restates-itself` cmd/define/README.md now has two "macOS only" paragraphs ten lines apart, plus a stray double blank line
  README.md:22 and :34 overlap; merge them keeping the CoreServices
  reasoning. Blank-line pair at :32-33 is insertion residue.
- **BR-7** [Minor] `usage-prose-lags-flags` fs.Usage prose names --llm-check but not --version
  PrintDefaults lists the flag, so this is consistency only.
- **BR-8** [Minor] `test-mutates-package-state` TestVersionIsHonestAboutUnstampedBuilds mutates package-level version, safe only because nothing here calls t.Parallel
  Confirmed no t.Parallel in cmd/define today. One line of comment records
  the dependency for whoever adds the first parallel test.

## Round 2 — 2026-09-09T14:13:14-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-09-09T15:08:59-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

- **BR-1** [Critical] `two-commands-one-line` -version dispatches as a mode but is absent from run()'s collision list, so it silently swallows every other mode
- **BR-2** [Important] `cross-boundary-contract-untested` the -ldflags stamp path is pinned by no test; renaming main.version ships an unversioned release silently
- **BR-3** [Important] `readme-front-door` root README.md still offers only make build / make install; the brew install appears only in cmd/define/README.md
- **BR-4** [Important] `two-commands-one-line` define --version <word> silently swallows the word, as --llm-check already does; sweep the class not the instance
- **BR-5** [Important] `discovery-lands-in-history` the dev-aliases define() shadow trap is recorded only in the issue Log, which archives to workshop/history
- **BR-6** [Minor] `doc-restates-itself` cmd/define/README.md now has two "macOS only" paragraphs ten lines apart, plus a stray double blank line
- **BR-7** [Minor] `usage-prose-lags-flags` fs.Usage prose names --llm-check but not --version
- **BR-8** [Minor] `test-mutates-package-state` TestVersionIsHonestAboutUnstampedBuilds mutates package-level version, safe only because nothing here calls t.Parallel
