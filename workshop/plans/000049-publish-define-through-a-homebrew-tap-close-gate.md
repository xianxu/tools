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
    - "n": 4
      timestamp: "2026-09-09T15:31:16-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: All six -version mode pairs exit 2 on the built binary; dropping the list row now reddens the guard.
          round: 4
        - id: BR-2
          disposition: addressed
          note: Renaming main.version keeps units green and reddens TestLdflagsStampReachesTheBinary plus the merge gate.
          round: 4
        - id: BR-3
          disposition: addressed
          note: Root README.md now leads with an Install section carrying the three brew lines.
          round: 4
        - id: BR-4
          disposition: addressed
          note: Both -version cat and -llm-check cat exit 2; the word matrix is derived from declaredModes.
          round: 4
        - id: BR-5
          disposition: addressed
          note: The define() shadow trap is now a workshop/lessons.md entry with the generalisation stated.
          round: 4
        - id: BR-6
          disposition: addressed
          note: One macOS paragraph, CoreServices reasoning kept, no stray blank pair.
          round: 4
        - id: BR-7
          disposition: addressed
          note: fs.Usage prose now names --version (see the separate Minor about its printed width).
          round: 4
        - id: BR-8
          disposition: addressed
          note: The t.Parallel dependency is recorded in a comment at the mutation site.
          round: 4
      findings:
        - id: BR-9
          severity: Important
          title: TestAtlasListsEveryConformanceCheck matches the filename anywhere, so the one table row the merge gate depends on can be deleted while it stays green
          detail: |-
            Demonstrated at efd8392: deleting only atlas/define.md:2085 (the version_conformance_test.go
            table row) leaves the test green, because the prose mention at atlas/define.md:2066 satisfies
            strings.Contains; deleting the harvest row, which has no prose mention, does fail it. Require
            the row form (a line matching "^\| `<name>` \|") instead of a bare Contains.
          family: guard-fails-open
          round: 4
        - id: BR-10
          severity: Important
          title: scanForExecutables convicts by extension but want maps a blob to only ONE path, so the same blob tracked as x.pyc and x.txt is not caught
          detail: |-
            The comment "identical content at two paths collapses; either name locates it"
            (repo_guard_test.go:313) was true only while conviction was content-only; it stopped being
            true when filepath.Ext(want[sha]) became half the predicate. Reproduced in a clone of efd8392:
            cmd/define/zz_art.pyc alone fails TestNoCommittedBinaries, the same blob also staged as
            zz_art.txt passes. Make want a sha -> []string and convict if ANY path carries a compiled
            extension. This is the 2nd finding in the family I am coining: the RULE is that a guard must
            be mutation-verified against the invariant it NAMES in its general form, not against the
            single reproduction that motivated it.
          family: guard-fails-open
          round: 4
        - id: BR-11
          severity: Important
          title: --llm-check's "answers above withStore" is asserted in main.go twice and in the atlas, and pinned by nothing
          detail: |-
            Round 2's I-3 pinned exactly this for --version with a newStore that t.Errors if built, and
            the fix's own comment at main.go:713 names BOTH modes in one sentence. Verified by mutation:
            moving `if *llmCheck { return runLLMCheck(...) }` below `d = d.withStore(opt, stderr)` builds
            clean and leaves the whole cmd/define package green. ARCH-PURPOSE — the class was enumerable
            and two long, and the second member is still open. Table both modes over the failing-newStore
            deps, asserting "the store was not built" rather than exit 0.
          family: asserted-invariant-unpinned
          round: 4
        - id: BR-12
          severity: Minor
          title: the three brew commands are duplicated verbatim in README.md and cmd/define/README.md with nothing keeping them in sync
          detail: |-
            2nd in this family, so the rule rather than the instance: a user-facing recipe present in two
            docs needs a doc-sync pin, and doc_sync_test.go already has the machinery. The root README
            claims cmd/define's is "the one copy that does" — true of the explanation, not of the commands.
          family: doc-restates-itself
          round: 4
        - id: BR-13
          severity: Minor
          title: the issue file has an unterminated inline code span that swallows round 3's Log heading and emits a phantom second "## Revisions"
          detail: |-
            workshop/issues/000049-…md:274 ends "Recorded under `" and the span closes only at line 324
            ("## Revisions`."), so round 3's ### heading is not a heading, ~50 lines render as one code
            span, and a naive ^## section split finds Revisions at 324 rather than 339. The intended
            sentence is "Recorded under `## Revisions`."
          family: artifact-record-malformed
          round: 4
        - id: BR-14
          severity: Minor
          title: version_conformance_test.go runs the built binary with cwd inherited from go test (= cmd/define/)
          detail: |-
            Harmless while --version returns above withStore, but this repo's own history is "three deck
            files reached commits because go test runs with cwd set to cmd/define/". Set cmd.Dir =
            t.TempDir() on the run; the build still needs the package dir.
          family: test-touches-real-state
          round: 4
        - id: BR-15
          severity: Minor
          title: main.go:600 is 127 chars after prose was inserted without re-wrapping, and the merged usage line at main.go:488 prints ~100 columns
          detail: |-
            The file wraps at ~80 everywhere else, and the usage line will soft-wrap in an 80-column
            terminal unlike the lines around it.
          family: comment-wrap-drift
          round: 4
        - id: BR-16
          severity: Minor
          title: the "derived %d modes; run() declares at least seven" floor is byte-identical in two tests, and the atlas guard's floor is 7 against 9 files
          detail: |-
            harvest_test.go:684 and :826 should share one declaredModesOrFail(t) helper (ARCH-DRY);
            doc_sync_test.go:727 could lose two conformance files to a broken glob and still certify.
          family: derived-set-floor-duplicated
          round: 4
      blocked: false
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

## Round 4 — 2026-09-09T15:31:16-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — All six -version mode pairs exit 2 on the built binary; dropping the list row now reddens the guard.
- BR-2 — addressed — Renaming main.version keeps units green and reddens TestLdflagsStampReachesTheBinary plus the merge gate.
- BR-3 — addressed — Root README.md now leads with an Install section carrying the three brew lines.
- BR-4 — addressed — Both -version cat and -llm-check cat exit 2; the word matrix is derived from declaredModes.
- BR-5 — addressed — The define() shadow trap is now a workshop/lessons.md entry with the generalisation stated.
- BR-6 — addressed — One macOS paragraph, CoreServices reasoning kept, no stray blank pair.
- BR-7 — addressed — fs.Usage prose now names --version (see the separate Minor about its printed width).
- BR-8 — addressed — The t.Parallel dependency is recorded in a comment at the mutation site.

### Raised

- **BR-9** [Important] `guard-fails-open` TestAtlasListsEveryConformanceCheck matches the filename anywhere, so the one table row the merge gate depends on can be deleted while it stays green
  Demonstrated at efd8392: deleting only atlas/define.md:2085 (the version_conformance_test.go
  table row) leaves the test green, because the prose mention at atlas/define.md:2066 satisfies
  strings.Contains; deleting the harvest row, which has no prose mention, does fail it. Require
  the row form (a line matching "^\| `<name>` \|") instead of a bare Contains.
- **BR-10** [Important] `guard-fails-open` scanForExecutables convicts by extension but want maps a blob to only ONE path, so the same blob tracked as x.pyc and x.txt is not caught
  The comment "identical content at two paths collapses; either name locates it"
  (repo_guard_test.go:313) was true only while conviction was content-only; it stopped being
  true when filepath.Ext(want[sha]) became half the predicate. Reproduced in a clone of efd8392:
  cmd/define/zz_art.pyc alone fails TestNoCommittedBinaries, the same blob also staged as
  zz_art.txt passes. Make want a sha -> []string and convict if ANY path carries a compiled
  extension. This is the 2nd finding in the family I am coining: the RULE is that a guard must
  be mutation-verified against the invariant it NAMES in its general form, not against the
  single reproduction that motivated it.
- **BR-11** [Important] `asserted-invariant-unpinned` --llm-check's "answers above withStore" is asserted in main.go twice and in the atlas, and pinned by nothing
  Round 2's I-3 pinned exactly this for --version with a newStore that t.Errors if built, and
  the fix's own comment at main.go:713 names BOTH modes in one sentence. Verified by mutation:
  moving `if *llmCheck { return runLLMCheck(...) }` below `d = d.withStore(opt, stderr)` builds
  clean and leaves the whole cmd/define package green. ARCH-PURPOSE — the class was enumerable
  and two long, and the second member is still open. Table both modes over the failing-newStore
  deps, asserting "the store was not built" rather than exit 0.
- **BR-12** [Minor] `doc-restates-itself` the three brew commands are duplicated verbatim in README.md and cmd/define/README.md with nothing keeping them in sync
  2nd in this family, so the rule rather than the instance: a user-facing recipe present in two
  docs needs a doc-sync pin, and doc_sync_test.go already has the machinery. The root README
  claims cmd/define's is "the one copy that does" — true of the explanation, not of the commands.
- **BR-13** [Minor] `artifact-record-malformed` the issue file has an unterminated inline code span that swallows round 3's Log heading and emits a phantom second "## Revisions"
  workshop/issues/000049-…md:274 ends "Recorded under `" and the span closes only at line 324
  ("## Revisions`."), so round 3's ### heading is not a heading, ~50 lines render as one code
  span, and a naive ^## section split finds Revisions at 324 rather than 339. The intended
  sentence is "Recorded under `## Revisions`."
- **BR-14** [Minor] `test-touches-real-state` version_conformance_test.go runs the built binary with cwd inherited from go test (= cmd/define/)
  Harmless while --version returns above withStore, but this repo's own history is "three deck
  files reached commits because go test runs with cwd set to cmd/define/". Set cmd.Dir =
  t.TempDir() on the run; the build still needs the package dir.
- **BR-15** [Minor] `comment-wrap-drift` main.go:600 is 127 chars after prose was inserted without re-wrapping, and the merged usage line at main.go:488 prints ~100 columns
  The file wraps at ~80 everywhere else, and the usage line will soft-wrap in an 80-column
  terminal unlike the lines around it.
- **BR-16** [Minor] `derived-set-floor-duplicated` the "derived %d modes; run() declares at least seven" floor is byte-identical in two tests, and the atlas guard's floor is 7 against 9 files
  harvest_test.go:684 and :826 should share one declaredModesOrFail(t) helper (ARCH-DRY);
  doc_sync_test.go:727 could lose two conformance files to a broken glob and still certify.

## Open findings

- **BR-9** [Important] `guard-fails-open` TestAtlasListsEveryConformanceCheck matches the filename anywhere, so the one table row the merge gate depends on can be deleted while it stays green
- **BR-10** [Important] `guard-fails-open` scanForExecutables convicts by extension but want maps a blob to only ONE path, so the same blob tracked as x.pyc and x.txt is not caught
- **BR-11** [Important] `asserted-invariant-unpinned` --llm-check's "answers above withStore" is asserted in main.go twice and in the atlas, and pinned by nothing
- **BR-12** [Minor] `doc-restates-itself` the three brew commands are duplicated verbatim in README.md and cmd/define/README.md with nothing keeping them in sync
- **BR-13** [Minor] `artifact-record-malformed` the issue file has an unterminated inline code span that swallows round 3's Log heading and emits a phantom second "## Revisions"
- **BR-14** [Minor] `test-touches-real-state` version_conformance_test.go runs the built binary with cwd inherited from go test (= cmd/define/)
- **BR-15** [Minor] `comment-wrap-drift` main.go:600 is 127 chars after prose was inserted without re-wrapping, and the merged usage line at main.go:488 prints ~100 columns
- **BR-16** [Minor] `derived-set-floor-duplicated` the "derived %d modes; run() declares at least seven" floor is byte-identical in two tests, and the atlas guard's floor is 7 against 9 files
