---
id: 000083
status: working
deps: []
github_issue:
created: 2026-10-08
updated: 2026-10-08
estimate_hours: 4.5
card_mirror: 'beb7cfd83f897cb45dfb3dc09f3e3da04886cb2c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-08T12:16:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: tools:0
    worktree: /Users/xianxu/workspace/tools
    repository: github.com/xianxu/tools
flow: {kind: full, provenance: inferred}
---

# castcut: record, annotate and cut terminal demos

## Problem

Recording a captioned terminal demo for the blog is a manual pipeline that lives
inside parley.nvim (`demo/README.md`, `demo/viewer.html`, `demo/cut.py`) and is
remembered only by rereading it. The tooling has no parley-specific parts, and
pair (couch broadcast) and nous need the same thing. Copying it per repo would
let the copies drift, and the `captions` header contract it shares with the blog's
`CastEmbed.astro` would drift with them.

A TUI weave layer was considered and deferred: two files are not worth a layer.
Revisit if a second shared TUI convention appears (pty test harness, keybinding rules).

## Spec

One `castcut` binary that wraps the whole flow, one subcommand per step:

- `castcut record [--cols C --rows R] -- <command> <out.cast>`: runs `asciinema rec`
  (an external dependency, not reimplemented) with a readable default size
  (95x36 worked for the parley post) and puts the recording where it's easy to find later.
  Consider `--capture-input` so keystrokes can be shown without an app-specific
  plugin such as Screenkey.
- `castcut annotate <take.cast>`: serves the embedded viewer (port of
  `viewer.html`, `go:embed`) on localhost with the cast preloaded. Alt+T stamps
  `~m:ss.s` and the notes download as `captions.txt`.
- `castcut cut <take.cast> <captions.txt> -o <cut.cast>`: Go port of `cut.py`
  with the same flags (`--speed --idle --lead --min-hold --wps --beat`) and
  output: asciicast v3 with `captions: [{start, end, text}]` in the header and
  `m` markers.
- `castcut --help` is written as agent instructions. It walks the operator through
  record → annotate → cut → embed. It documents the captions header contract and how
  to embed a cut cast in different destinations: xianxu.dev (`public/casts/` +
  `<div class="cast-embed" data-cast=…>`, rendered by
  `src/components/blog/CastEmbed.astro`), plain asciinema-player, or another blog
  format that needs its own overlay.

Per-app parts stay in each app repo: the isolated demo launcher (e.g.
`parley_app --demo`) and the shot list.

## Done when

- `castcut record|annotate|cut` and `castcut --help` exist. `brew install xianxu/tools/castcut` installs it.
- `castcut cut` produces an asciicast v3 whose `captions` header and `m` markers satisfy
  the contract `CastEmbed.astro` reads, and its timing model is pinned by property tests:
  each caption window plays in real time, idle between captions is squeezed and sped up,
  event order and count are preserved. (`cut.py` was the prototype, not a spec; no
  byte-identity, no Python.)
- The parley.nvim demo README points at `castcut`, and its `cut.py`/`viewer.html` are removed (follow-up in parley.nvim).
- A couch broadcast take is recorded, annotated and cut end to end with `castcut`.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec                design=0.5 impl=0.04
item: greenfield-go-module      design=0.5 impl=0.24
item: smaller-go-module         design=0.1 impl=0.12
item: smaller-go-module         design=0.1 impl=0.12
item: greenfield-go-module      design=0.5 impl=0.24
item: skill-or-dispatcher       design=0.3 impl=0.08
item: real-api-discovery        design=0.0 impl=0.16
item: milestone-review          design=0.0 impl=0.14
item: milestone-review          design=0.0 impl=0.14
item: milestone-review          design=0.0 impl=0.14
item: atlas-docs                design=0.1 impl=0.04
item: scope-pivot               design=0.3 impl=0.08
item: cross-repo-refactor-small design=0.1 impl=0.08
design-buffer: 0.15
total: 4.50
```

Items in order: spec; cut timing model (greenfield); cast+captions parse; record; annotate server + viewer
(greenfield); `--help` as agent instructions; asciinema discovery; M1–M3 reviews; atlas; the byte-identity
drop; homebrew tap + parley.nvim follow-up. Design 2.5 × 1.15 = 2.875, impl 1.62.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Plan

Durable plan: [workshop/plans/000083-castcut-plan.md](../plans/000083-castcut-plan.md) (decisions D1–D7 there).

- [x] M1 — `castcut cut`: the timing model (pure, property + fuzz tests), stdlib JSON, CLI wiring
- [x] M2 — `castcut record` (fake asciinema on PATH), `castcut annotate` (localhost server, notes sidecar), `castcut --help` as agent instructions
- [x] M3 — Homebrew formula + tag (operator-confirmed), parley.nvim follow-up issue, couch broadcast end-to-end take

## Log

### 2026-10-08
- 2026-10-08: closed M2 — go test -count=1 ./cmd/castcut green; BR-13: record status from the take x event — fake failure mode test + mutation red, live asciinema 3.2.1 strict conformance green, sandboxed EPERM repro reports incomplete take exit 1; BR-14: in-browser check moved to M3 in plan Revisions (no browser driver here; server via handler tests + real-binary smoke, JS node --check); minors (disabled-until-loaded, sidecar mode, stale tmp sweep) fixed. actual = measured 1.12h - 0.74h M1 measurement.; review verdict: SHIP
- 2026-10-08: closed M1 — go test -count=1 ./cmd/castcut green at HEAD c48afec (run alone, not piped); BR-10 fixed (rows split: per-gap vs cumulative); negative idle_time_limit refusal recorded + tested; one inRange predicate at every duration boundary, guards mutation-checked; 500 property takes, 2e5 envelope, naive-warp ref, CLI wiring; FuzzCut 30s clean; review verdict: FIX-THEN-SHIP

- Claimed; plan drafted. asciinema 3.2.1 installed (`--window-size`, `--capture-input`, v3 default). Byte-identity was first planned (Python-compatible JSON seam), then dropped by the operator: castcut is a new tool grown from the prototype (see Revisions).
- Plan-quality round: windows past the take's end now hold the last frame (prototype truncated them); warp is O(log S) per lookup via cumulative output + binary search instead of the prototype's scan-from-zero (the plan said cursor; binary search is stateless and meets the same envelope).
- M1: property tests over 500 random takes + 2e5-event envelope (0.85 s) + naive-warp reference; fuzz 20 s × 3 targets clean. Mutation checks, each red then reverted: drop overlap clamp → (c)/(e); markers after output → marker example row (first attempt was vacuous: speed 5 made the tie unreachable through rounding — fixed with speed 1); ignore idle → (d); no hold event → late-caption row; window rate ≠ 1 → (c); HTML escaping on → CLI row.
- Correction (actuals): M1's close recorded `--actual 1.25`, hand-typed across re-close rounds (0.9→1.25) instead of measured. The measurement at the first M1 attempt was 0.74h and the whole issue measured 1.05h at M2's close, so M1's 1.25 overstates; treat M1 ≈ 0.74h + its rework share. M2 is recorded as the measured increment.
- M2: record via a fake asciinema that runs the command through `sh` + live conformance against asciinema 3.2.1 (headless, unsandboxed, strict) — same `recordContract`; dropping `--return` reds both. annotate: real handler on a real listener; Host/Origin/size guards each mutation-checked red (the first Origin mutation didn't compile — redone). Real-binary smoke: serve → PUT sidecar → foreign Host 403 → cut reads the sidecar → Ctrl-C exit 0. Viewer JS `node --check` clean; in-browser behaviour (player, Alt+T, autosave) not yet exercised — M3's couch take does that. `help_test.go` derives every flag from each command's `-h`; it caught an undocumented `--idle-time-limit`.
- Side note: a heredoc turned `\u003c` in a Go raw string into `<`, so the no-escape assertion first checked the wrong thing; caught because the test failed on correct output.

## Revisions

### 2026-10-08 — castcut is a new tool, not a port

Reason (operator): `cut.py` was a quick prototype; castcut is a new feature built from it. Delta:
"Done when" byte-identity bullet replaced by a contract + timing-property bullet; M1 no longer
carries the Python-compatible JSON seam or a python3 oracle. Plan revision of the same date has the detail.
- M3 started: `../homebrew-tools/Formula/castcut.rb` drafted (uncommitted; sha256 pending the v0.1.8 tag), `ruby -c` OK, its test-block expectation checked against the real cut output. `make install` links `castcut` into ~/.local/bin for the couch take. Waiting on the operator for the couch take and for each outward step (PR/merge, tag, tap push, parley.nvim issue).
- M3 couch end-to-end (operator-driven, 2026-10-08): `castcut record -- couch` → annotate → cut, run by the operator in `brain/` (take-01, take-02 + captions + take-02-cut.cast observed under `brain/recordings/`, since removed by the operator). Operator verdict: "works well". Feedback folded in during the take: record at the terminal's own size (size flags deleted), annotate prints the next step, cut names its output `<take>-cut.cast`. The in-browser checks moved here from M2 (Alt+T stamping, sidecar autosave, cut preview) are covered by that run; reload-persistence was not separately confirmed.
- M3 done (2026-10-08): PR #57 merged (21ee5e9) after a side-quest deps fix (main's merge-check had been red since 975ddb6: `construct/deps` lacked ariadne's source URL); tag v0.1.8 pushed; tap `castcut: release v0.1.8` (0ef958d) pushed; `brew install xianxu/tools/castcut` → `castcut v0.1.8`, `brew test` exit 0; parley.nvim#311 filed. M3 is ticked by hand rather than milestone-closed: its review is the issue-close boundary review (#175).
