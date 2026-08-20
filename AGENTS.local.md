# Local Extensions

## Repo-specific rules

### This repo is a binary collection, not an application

Each `cmd/<name>` is an independent tool with its own issue history. There is
no shared runtime, no daemon, no cross-tool coupling beyond `internal/`.

- **One tool per issue.** A new binary is a new issue, not a milestone on an
  existing one.
- **`internal/` is earned, not anticipated.** Code moves there on the *second*
  consumer, never the first (ARCH-DRY cuts both ways — a one-caller "shared"
  package is speculative generality).
- **A tool must justify a slot on `$PATH`** in one sentence. If it can't, it
  belongs in `construct/dev-aliases.sh` as a shell function instead.

### Platform-bound code

Tools that bind macOS frameworks (cgo, `-framework …`) must keep
`go build ./...` and `go vet ./...` green on non-darwin hosts:

- Put the platform call behind a small interface in the tool's own package.
- `//go:build darwin` on the cgo file; a sibling `//go:build !darwin` stub
  returning a "not supported on $GOOS" error.
- Never let a `#cgo LDFLAGS` line reach a portable file.

### Output conventions

These are interactive terminal tools, so:

- Human-readable output on **stdout**, diagnostics on **stderr**, meaningful
  exit codes (a lookup miss is a non-zero exit, not a cheerful empty result).
- Colour and other terminal niceties degrade when stdout is not a TTY, so
  piping into another tool yields clean text.
- Anything that reaches the network or plays audio states what it is doing, and
  is suppressible by a flag.
