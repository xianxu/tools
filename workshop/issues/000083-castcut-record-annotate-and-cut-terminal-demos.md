---
id: 000083
status: open
deps: []
github_issue:
created: 2026-10-08
updated: 2026-10-08
estimate_hours:
card_mirror: 'd026bad11e0353558f16f8cdabf70d1f45bb5996' # card fields mirrored from issue-cards; edit via sdlc
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
- `castcut cut` output is byte-identical to `cut.py` on a small fixture cast plus
  captions (golden produced by `cut.py` and checked in; the parley-nvim-v1 raw take
  no longer exists).
- The parley.nvim demo README points at `castcut`, and its `cut.py`/`viewer.html` are removed (follow-up in parley.nvim).
- A couch broadcast take is recorded, annotated and cut end to end with `castcut`.

## Plan

- [ ]

## Log

### 2026-10-08
