---
id: 000049
status: working
deps: []
github_issue:
created: 2026-09-09
updated: 2026-09-09
estimate_hours: 1.41
started: 2026-09-09T10:02:30-07:00
---

# publish define through a homebrew tap

## Problem

`define` is installable only by cloning the repo and running `go build`. The
`define-learn` MVP is complete and the operator uses it daily on a real deck, so
the thing that stops anyone else trying it is the install.

## Spec

**`brew tap xianxu/tools && brew install define`.**

### The tap is NEW, and that is the operator's decision

`pair` publishes from `xianxu/homebrew-pair` (tapped as `xianxu/pair`), and the
obvious move is to add a second formula there. **Rejected**: someone who wants
only `define` would have to tap something called `pair`, and that tap's README is
pair-specific. `xianxu/homebrew-tools` matches the repo the code lives in and has
room for anything else `tools` ships.

Cost recorded honestly: one more public repo, and users of both tools tap twice.

### What makes this simpler than pair's formula

`pair` depends on `zellij`, `neovim`, `fzf`, `jq` and `par`, installs three asset
trees under `libexec`, and generates a runtime bundle at build time.

**`define` depends on nothing.** It reads macOS's installed dictionaries through
DictionaryServices and plays audio with `afplay` — both already on the machine —
and its only network use is the pronunciation CDN, which is optional and cached.
So the formula is `go build` and one binary.

**It is macOS-only for the same reason**, and the formula should say so rather
than failing confusingly on Linux.

### `--version`, which does not exist yet

A binary someone installed from a tap must be able to say what it is: a bug
report that cannot name a version costs a round trip to establish what the
reporter is running. Stamped at build time through `-ldflags -X`, the way pair's
formula already sets `main.defaultPairHome` — so the number lives in the git tag
and the formula, never in a constant someone has to remember to bump.

**A build with no stamp says so** rather than claiming a version it does not
have: `go build` from a clone is not a release, and printing `v0.1.0` there would
make the flag a lie exactly where it is most likely to be read.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.* Calibration tagged **stale**; derived against
`#8` (3.71/2.98) and `#48` (3.60/3.40).

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.12 impl=0.04
item: smaller-go-module        design=0.02 impl=0.10
item: skill-or-dispatcher      design=0.03 impl=0.14
item: atlas-docs               design=0.02 impl=0.06
item: ux-rename-iteration      design=0.0  impl=0.15
item: milestone-review         design=0.0  impl=0.30
item: milestone-review         design=0.0  impl=0.40
design-buffer: 0.15
total: 1.41
```

| row | the work |
|---|---|
| `issue-spec` 0.12/0.04 | short: the shape was settled by two operator answers, and the facts were gathered before asking. |
| `smaller-go-module` 0.02/0.10 | `--version`, stamped by ldflags, honest when unstamped. |
| `skill-or-dispatcher` 0.03/0.14 | the tap repo, `Formula/define.rb`, its README — a packaging artifact rather than a Go module, and this is the closest primitive. |
| `atlas-docs` 0.02/0.06 | the README's install section. |
| `ux-rename-iteration` 0.0/0.15 | **installing from the tap on a real machine**, which is the only thing that proves a checksum and a formula. |
| `milestone-review` 0.0/0.30 + 0.0/0.40 | the close pair, BELOW the house 0.60/0.85: the diff is one flag, two new files in a peer repo and a doc section, with no new architectural surface for a review to work against. |

**Reconciliation.** Σdesign = 0.19, Σimpl = 1.19.
0.19 × 1.15 + 1.19 = **1.41**.

**Why it is the smallest issue in the project.** `define` has no brew
dependencies — the formula is `go build` and one binary, where `pair`'s installs
three asset trees and generates a runtime bundle. Most of the cost is the review
pair and the hand-install, which are fixed.

## Done when

- [ ] `brew tap xianxu/tools && brew install define` installs a working `define`
      on a clean machine. **Pending `v0.1.1`** — the mechanism is proven (installed
      from the tap on a clean vanilla VM: install, lookup and audio all confirmed),
      but `v0.1.0` = `7380263` predates the BR-1 fix, so the *published* binary
      still drops a mode. Ticks when `v0.1.1` is installed and verified.
- [x] `define --version` reports the released version, and reports something
      honest (not a fabricated number) when built from a clone.
- [x] The formula declares macOS rather than failing confusingly elsewhere.
- [ ] `tools` carries a release tag, and the formula's `sha256` matches the
      tarball GitHub serves for it — verified by installing from the tap, not by
      assuming the checksum. **Pending `v0.1.1`**: this held for `v0.1.0`
      (checksum independently re-hashed by the boundary review), and re-holds once
      the tag and formula are bumped.
- [x] `cmd/define/README.md` leads with the brew install and keeps the
      build-from-source line for contributors.

## Plan

Single-pass: one boundary, plain checkboxes (AGENTS.md §3).

- [x] `--version`, stamped by ldflags, honest when unstamped.
- [x] Tag `v0.1.0` and confirm the tarball's checksum from GitHub.
- [x] The tap repo, its formula and its README.
- [x] Install from the tap on this machine and run it.
- [ ] **Post-merge:** tag `v0.1.1` on `main`, bump `url` + `sha256` in
      `xianxu/homebrew-tools`, and `brew reinstall` to verify `define v0.1.1` —
      the release that actually contains the BR-1 fix (#49 V).

## Log

### 2026-09-09 — filed

Operator: *"now, we should publish this as a homebrew. I already publish pair, so
we can use the same tap I assume."*

**Checked rather than assumed**, which is the session's own lesson: the tap is
`xianxu/homebrew-pair` holding one formula; `tools` has ZERO git tags, so this
needs a first release; `define` shells out only to `afplay` and has no brew
dependencies at all; and there is no `--version` flag. The operator chose a new
tap and `v0.1.0` when asked.

### 2026-09-09 — installed from the tap on a clean machine

`xianxu/homebrew-tools` is public, `Formula/define.rb` builds from the `v0.1.0`
tarball, and **the checksum was verified the only way that counts** — by
installing from the tap rather than by trusting the number.

**Two things only a real install could have found**, both now in the tap README
and (this round) in `cmd/define/README.md`, which still carried the broken pair:

- `brew trust xianxu/tools` is required first. Without it Homebrew reports
  `invalid syntax in tap!`, which reads like a Ruby error in the formula.
- `brew install define` installs a DIFFERENT program — `define` also exists in
  homebrew-core. The qualified `xianxu/tools/define` is the one that works.

**Smoke-tested in a vanilla VM** (`make tart-clean && VANILLA=1 make tart`), not
a provisioned one, and that distinction turned out to be load-bearing. A normal
`make tart` mounts the workspace, and `construct/dev-aliases.sh` then emits a
`define()` SHELL FUNCTION that rebuilds from source — functions outrank PATH in
zsh, so it silently shadows the bottle and the smoke test measures a local build.
The tell is `--version`: the function builds without ldflags, so it prints
`built from source` where the bottle prints `define v0.1.0`. This trap applies to
every `cmd/X` in every ariadne-styled peer, so it will recur for any future
formula.

Operator verified in the VM: install, lookup and audio. Version stamping verified
here both ways — unstamped `go build` → `define (built from source)`;
`-ldflags -X main.version=v0.1.0` → `define v0.1.0` — which is the formula's own
`test do` assertion.

`v0.1.0` points at `7380263` on the branch and is pushed to origin. `sdlc merge`
uses `gh pr merge --merge`, so the commit becomes an ancestor of `main` and the
tag stays reachable — checked rather than assumed, since a squash would have
stranded it.

### 2026-09-09 — boundary review round 1: REWORK, five blocking findings fixed

**BR-1 (Critical) was real and was the THIRD instance of a named class.** `-version`
dispatched as a mode but was absent from `run()`'s collision list, so
`define --version --play` printed the version, exited 0 and silently dropped
`--play`. `main.go:598` names the same bug twice already.

The guard built to prevent exactly this (`TestEveryDispatchedModeIsInTheCollisionList`)
went green because its AST parse required `return <call>` and the new dispatch was
`return 0` — a `BasicLit`, not a `CallExpr`. **That is the second time this guard was
itself the reason the bug went unseen** (#8 BR-11 was the first: it recognised only
`if *x`, so `-forget`'s `if forgetting` slipped past). So the fix is the CLASS, not the
site (ARCH-PURPOSE): the parse now accepts ANY `return` in a flag-guarded `if`, making
it shape-INDEPENDENT, and it cannot over-match because the condition must already be a
bare flag ident. Mutation-verified — dropping the `{"-version", *versionFlag}` row now
fails with *"run() returns on -version, so it is a MODE, but it is not in the collision
list"*, where before it passed.

**BR-4 swept with it.** The enumerable class is *modes dispatched above the
argument-count switch* — exactly `-llm-check` and `-version`, both of which swallowed a
word (`define --llm-check sycophantic` exited 0). Both moved BELOW the switch rather
than growing a bespoke check, so the switch every other mode is judged by now judges
them too. Still above `withStore`, so the "answers on a broken machine" property holds.

**BR-2: the ldflags path was pinned by nothing** — the unit test assigns the Go
variable, which cannot fail when the LINKER path breaks.
`TestLdflagsStampReachesTheBinary` (conformance) now shells the real toolchain with the
formula's own flag and reads `--version` off the built binary. Mutation-verified with a
realistic refactor (rename `version` → `buildVersion`, update its one test reference):
the package builds, both version unit tests PASS, the shipped binary says
`define (built from source)` — and only the conformance test reddens.

**BR-3:** the root `README.md` gained an Install section; it had offered only
`make build` on a now-public repo whose issue exists to replace that path.

**BR-5:** the `define()` shadow trap moved from this Log to `workshop/lessons.md`,
since Logs archive to `workshop/history/` which AGENTS.md tells agents not to read.

**A guard caught me mid-fix**, which is worth recording: placing `runVersion` above
`var version string` inserted it between that var and its own doc comment, orphaning
the var and handing its prose to the new function — precisely what
`TestADocCommentNamesWhatItSitsOn` exists to catch. It failed, named the defect, and
`runVersion` moved below `versionLine`. Likewise `TestEverySkipIsRoutedOrWaived` caught
the new conformance test's bare `t.Skip`, now routed through `conformance.SkipOrFail`
so `CONFORMANCE_STRICT=1` cannot report green for a check that did not run.

Minors also closed: the two overlapping "macOS only" paragraphs merged, the stray blank
line removed, `fs.Usage` prose names `--version`, and the package-var mutation in
`main_test.go` records its `t.Parallel` dependency.

Verified: `go build ./...`, `go vet ./...`, `go vet -tags conformance ./cmd/define/`,
`gofmt -l` clean, `go test ./...` fully green.

### 2026-09-09 — boundary review round 2: FIX-THEN-SHIP, four Importants closed

Round 2 mutation-verified all five round-1 fixes independently and found no
Criticals. Its four new Importants:

**I-2 — the mode x word half was still hand-enumerated (3rd in family
`two-commands-one-line`).** The mode x mode half already derived every pair from
`declaredModes`, which is why a seventh mode got pair coverage free; the word half
was six hand-written switch arms plus hand-listed tests, so a new mode joined the
pair matrix automatically and got NO word coverage — *exactly* how `-version`
reached production swallowing one, and `-llm-check` before it.
`TestEveryModeRefusesATrailingWord` now iterates the derived list (with
`stringFlags` deriving argv shape, so `-forget` gets a value and the rest do not),
and an eighth mode is covered the day its row is added. Mutation-verified: deleting
the `-version` arm reddens exactly that subtest. It also sets the three
`DEFINE_LLM_*` vars, because the review measured the old test making a **live 1.2s
model call** when `-llm-check` regressed.

**I-3 — an asserted invariant that nothing enforced, both halves.** `main.go` and
`atlas/define.md` both claim `--version` answers ABOVE `withStore`; the reviewer
moved the dispatch below it and the whole suite stayed green, because `deps{}`
leaves `newStore` nil and `withStore` a no-op. The test now injects a `newStore`
that FAILS if built, so the position is pinned rather than described — the
reviewer's exact mutation now reddens. Second half: nothing ran `-tags
conformance`, so BR-2's fix protected only whoever remembered the command. Added
`scripts/merge-checks.d/10-release-stamp.sh`, which the CI workflow already
dispatches through `run-merge-checks.sh`, scoped to `-run Ldflags` so the gate
makes no live model or dictionary calls. Mutation-verified end to end: renaming
`main.version` now fails the MERGE GATE, not just a test somebody might run. The
fleet-wide `CONFORMANCE_STRICT` question stays with ariadne#37; this is the
repo-local floor that does not wait on it.

**I-4 — the compiled-artifact guard was blind to the artifact this window
removed.** `TestNoBinariesInHistory` knew only Mach-O and ELF magic, so a `.pyc`
walked into a now-public repo and the guard reported green. Extended by EXACT
extension (`.pyc`, `.pyo`, `.class`, `.o`, `.a`, `.so`, `.dylib`, `.wasm`) — which
carries none of the imprecision the magic-bytes comment rejected, since a file
named `.pyc` is a compiled module with no judgment involved. The existing blob
`b1fc21e3` (7892 bytes) is recorded in `acceptedCompiledBlobs` as debt ACCEPTED
rather than paid: rewriting would move `7380263`, the commit `v0.1.0` tags and the
formula's `sha256` pins. `TestAcceptedCompiledBlobsAreStillReachable` fails if a
waiver outlives its debt. Both halves mutation-verified.

**I-1 — the released artifact predates the fix. Operator chose to cut `v0.1.1`.**
`v0.1.0` = `7380263`, which contains neither `512c706` (the BR-1/BR-4 fix) nor
`43559b0`. So the tarball the formula pins ships a `define` where
`--version --play` drops `--play`. Done-when #1 is NOT ticked against that
artifact: `v0.1.1` is tagged on main after merge and the tap's `url` + `sha256`
bumped, then verified by reinstalling. Recorded under `### 2026-09-09 — boundary review round 3: FIX-THEN-SHIP, five Importants closed

Round 3 re-verified all eight prior findings by reversion in a scratch copy. Its
five new Importants, each fixed and mutation-verified:

**I — a waiver wider than its own justification.** `acceptedCompiledBlobs` was a
package global consulted inside `scanForExecutables`, which BOTH guards call. Its
argument is purely historical ("rewriting would move the commit `v0.1.0` tags")
and says nothing about the index — so staging a NEW file carrying the same blob
passed `TestNoCommittedBinaries`, the guard `atlas/repo-guards.md` calls "the last
moment the mistake is free". The waiver is now a PARAMETER: `nil` from the index
guard, the list from the history guard. Verified by reproducing the reviewer's
bypass — a fresh `replanted.pyc` with blob `b1fc21e3` is now caught.

**II — a gate that passes green when its check does not run.** `go test -run
<no match>` exits 0, so renaming the test would leave the merge check reporting
`✓ passed` having executed nothing — the exact "a skip reads as green" failure the
script's own header argues against. It now anchors the pattern, sets
`CONFORMANCE_STRICT=1`, and REQUIRES the test's own `--- PASS` line as proof of
execution. Mutation-verified: renaming the test now fails the gate.

**III — atlas prose contradicting the code, 2nd in family `usage-prose-lags-flags`.**
The rule taken, not the two edits: *a statement of a mechanism's decision rule
belongs at the mechanism, and an atlas enumeration of a code-derivable set must be
derived, never retyped.* "What convicts a blob" moved into `scanForExecutables`'
doc comment with `atlas/repo-guards.md` linking rather than restating; the
"on demand, not in CI" claim corrected; and the conformance table — which lagged
by TWO of nine and had since before this window — completed and then PINNED by
`TestAtlasListsEveryConformanceCheck`, which globs the files and fails on any with
no row. Only presence is derived; the prose stays human.

**IV — the last hand-typed mode set, 4th in family `two-commands-one-line`.**
`TestRunRefusesTwoModes` was a literal table, and this issue's own commit series —
which argued that a remembered enumeration IS the defect — added three rows to it
by hand. Now every unordered pair derives from `declaredModes` (21 pairs), so an
eighth mode is covered the day its row is added. Mutation-verified: removing
run()'s `return 2` on a collision reddens 11 of 21.

**V — Done-when ticked against a release that does not exist.** #1 and #4 are
UNTICKED and annotated *pending `v0.1.1`*, with the tag + formula bump + reinstall
added as an explicit post-merge `## Plan` row. That row is deliberately unchecked
at close time — it happens after the merge — so this close passes `--no-plan-check`
with the reason recorded rather than pretending otherwise.

Minors: the two near-identical AST walkers collapsed into one
`flagsDeclaredWith(t, kind)` with `argvForMode` deriving argv shape (ARCH-DRY);
the merge check's always-run behaviour documented as deliberate. Recorded not
fixed: the conformance test's `-ldflags` string is a hand restatement of the peer
formula, which the formula's own `test do` pins from the other side.

## Revisions`.

Minors closed: two stale claims about WHERE `--llm-check` dispatches deleted
rather than corrected (a statement about dispatch position belongs at the dispatch
site); the install recipe reduced to ONE canonical copy — root README keeps the
commands and links `cmd/define/README.md#install` for the why, and the miscounted
"first two lines" (it was lines 1 and 3) is fixed; the dangling `[[...]]` in
lessons.md replaced with the actual lesson; and a note that the conformance build
deliberately omits the formula's `GOFLAGS`, since `-trimpath -mod=readonly` are
orthogonal to the `-X` symbol path.

Verified: `go build ./...`, `go vet ./...`, `go vet -tags conformance`, `gofmt -l`
clean, `go test ./...` green, `go test -tags conformance -run Ldflags` green, and
`bash scripts/run-merge-checks.sh` green.

## Revisions

**2026-09-09 — Done-when #1 is satisfied by `v0.1.1`, not `v0.1.0`.**
*Reason:* the boundary review (I-1) measured that `v0.1.0` = `7380263` predates
`512c706`'s Critical fix, so the published tarball ships a `define` that silently
drops a mode. Ticking "installs a working `define`" against it would record a
claim about an artifact lacking this issue's own work.
*Delta:* the release cut for this issue is `v0.1.1`, tagged on `main` after the
merge, with `url` + `sha256` bumped in `xianxu/homebrew-tools` and verified by
reinstalling from the tap. `v0.1.0` remains tagged and reachable as the first
release; it is superseded rather than withdrawn.
