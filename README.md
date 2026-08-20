# tools

Small Go binaries — one job each, installed onto `$PATH`.

An [ariadne](../ariadne) derivative: the SDLC spine (`sdlc`), the agentic
context (`AGENTS.md`, skills), and the build/merge machinery are inherited
through `construct/deps` (`substrate ../ariadne`) and recomposed by `weave`.
This repo declares only its own binaries.

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

```sh
define sycophantic          # definition + /ˌsikəˈfan(t)ik/, played 3x
define -times 1 record      # play once instead of three times
define -no-audio bank       # no fetch, no sound
define -locale gb colour    # British pronunciation
define -raw record          # the unparsed dictionary entry
define -no-color bank       # never emit ANSI (also automatic when piped)
```

Exit codes: `0` success, `1` no dictionary entry, `2` usage error.

Lookup goes through macOS's CoreServices, which searches **every active
dictionary** rather than NOAD specifically — the SDK offers no way to pick one.
NOAD answers for ordinary English words (hence the Google-matching notation), but
`iPhone` comes from Apple Dictionary, and enabling the Chinese dictionaries will
return entries this tool does not format. Adjust the set in Dictionary.app.

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
go test -tags conformance ./...
```

## Platform

Several tools bind macOS system frameworks (`define` reads the bundled New
Oxford American Dictionary through CoreServices). Those files carry a
`//go:build darwin` tag and a non-darwin stub, so `go build ./...` and
`go vet ./...` stay green on any host; the binary just reports the tool is
unavailable there.
