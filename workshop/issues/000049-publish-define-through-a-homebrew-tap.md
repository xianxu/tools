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

- [x] `brew tap xianxu/tools && brew install define` installs a working `define`
      on a clean machine.
- [x] `define --version` reports the released version, and reports something
      honest (not a fabricated number) when built from a clone.
- [x] The formula declares macOS rather than failing confusingly elsewhere.
- [x] `tools` carries a `v0.1.0` tag, and the formula's `sha256` matches the
      tarball GitHub serves for it — verified by installing from the tap, not by
      assuming the checksum.
- [x] `cmd/define/README.md` leads with the brew install and keeps the
      build-from-source line for contributors.

## Plan

Single-pass: one boundary, plain checkboxes (AGENTS.md §3).

- [x] `--version`, stamped by ldflags, honest when unstamped.
- [x] Tag `v0.1.0` and confirm the tarball's checksum from GitHub.
- [x] The tap repo, its formula and its README.
- [x] Install from the tap on this machine and run it.

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
