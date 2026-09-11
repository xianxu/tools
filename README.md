# tools

Small Go binaries — one job each, installed onto `$PATH`.

An [ariadne](../ariadne) derivative: the SDLC spine (`sdlc`), the agentic
context (`AGENTS.md`, skills), and the build/merge machinery are inherited
through `construct/deps` (`substrate ../ariadne`) and recomposed by `weave`.
This repo declares only its own binaries.

## Install

`define` is published through a Homebrew tap:

```sh
brew trust xianxu/tools          # third-party taps are untrusted by default
brew tap xianxu/tools
brew install xianxu/tools/define
```

The `brew trust` line and the *qualified* formula name are both load-bearing;
[cmd/define/README.md](cmd/define/README.md#install) says why, and is the one
copy that does — this section is deliberately just the commands.

macOS only. For contributors, or for a binary not yet tapped, `make build` /
`make install` build everything here from source.

The directory you run `define` in *is* its deck, so it asks before writing to a
new one (`--here` to skip the question; declining still answers the lookup). See
[cmd/define/README.md](cmd/define/README.md).

## Layout

```
cmd/<name>/      one directory per binary — the whole public surface
internal/<pkg>/  shared code; import-fenced to this module by the Go toolchain
bin/             build output (gitignored)
```

A tool earns a `cmd/` directory when it is worth a name on `$PATH`. Anything
smaller stays a shell function in `construct/dev-aliases.sh`.

## Tools

| binary | what it does |
|---|---|
| `define` | Print a word's dictionary definition with Google-style IPA, and play its pronunciation. |

### define

Print a word's dictionary definition with Google-style IPA, and play its
pronunciation — plus a spaced-repetition review loop, a learner model, and
questions answered by a language model when a dictionary cannot.

```sh
define                      # interactive: type a word, / for commands, ^C to quit
define sycophantic          # definition + /ˌsikəˈfan(t)ik/, played 3x
define --play               # review what is due today
```

**[cmd/define/README.md](cmd/define/README.md)** is its documentation — the flags,
the review loop, the clickable regions, what it writes in your directory, and the
decisions behind each. `atlas/define.md` is the map of how it is built.

## Build

```sh
make build     # compile every cmd/ into bin/ (target inherited from ariadne)
go test ./...  # unit tests, including the fuzz corpus
```

```sh
make install   # symlink bin/* into ~/.local/bin (already on PATH)
```

Live conformance checks sit behind a build tag and must run unsandboxed:

```sh
go test -tags conformance ./...                     # skips what it cannot reach
CONFORMANCE_STRICT=1 go test -tags conformance ./...   # a skip is a failure
```

Without the variable a missing dependency — no network, no NOAD, no terminal, no
`afplay`, no model — SKIPS, so the suite is still useful offline. That makes a
sandboxed run report success for checks that never executed, which is the wrong
answer for CI or for a close that has to mean something: set `CONFORMANCE_STRICT`
there and green means it ran.

The guarantee covers `./...` because it is enforced rather than swept.
`internal/conformance` owns the decision, and its `TestEverySkipIsRoutedOrWaived`
walks the tree and FAILS on any `t.Skip` that is neither routed through it nor
marked `conformance:inapplicable` with a reason — so a new suite in a package
nobody thought to sweep cannot quietly opt out.

## Platform

Several tools bind macOS system frameworks (`define` reads the bundled New
Oxford American Dictionary through CoreServices). Those files carry a
`//go:build darwin` tag and a non-darwin stub, so `go build ./...` and
`go vet ./...` stay green on any host; the binary just reports the tool is
unavailable there.
