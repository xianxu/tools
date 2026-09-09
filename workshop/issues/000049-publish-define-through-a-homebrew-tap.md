---
id: 000049
status: working
deps: []
github_issue:
created: 2026-09-09
updated: 2026-09-09
estimate_hours:
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

## Done when

- [ ] `brew tap xianxu/tools && brew install define` installs a working `define`
      on a clean machine.
- [ ] `define --version` reports the released version, and reports something
      honest (not a fabricated number) when built from a clone.
- [ ] The formula declares macOS rather than failing confusingly elsewhere.
- [ ] `tools` carries a `v0.1.0` tag, and the formula's `sha256` matches the
      tarball GitHub serves for it — verified by installing from the tap, not by
      assuming the checksum.
- [ ] `cmd/define/README.md` leads with the brew install and keeps the
      build-from-source line for contributors.

## Plan

Single-pass: one boundary, plain checkboxes (AGENTS.md §3).

- [ ] `--version`, stamped by ldflags, honest when unstamped.
- [ ] Tag `v0.1.0` and confirm the tarball's checksum from GitHub.
- [ ] The tap repo, its formula and its README.
- [ ] Install from the tap on this machine and run it.

## Log

### 2026-09-09 — filed

Operator: *"now, we should publish this as a homebrew. I already publish pair, so
we can use the same tap I assume."*

**Checked rather than assumed**, which is the session's own lesson: the tap is
`xianxu/homebrew-pair` holding one formula; `tools` has ZERO git tags, so this
needs a first release; `define` shells out only to `afplay` and has no brew
dependencies at all; and there is no `--version` flag. The operator chose a new
tap and `v0.1.0` when asked.
