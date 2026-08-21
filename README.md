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
define                      # interactive: type a word, press return to replay, ^C to quit
define /help                # / starts a command instead of a word; /help lists them
echo sycophantic | define   # or feed it words on stdin
define sycophantic          # definition + /ˌsikəˈfan(t)ik/, played 3x
define --sound 1 record     # play once instead of three times
define -no-audio bank       # no fetch, no sound
define -locale gb colour    # British pronunciation
define -raw record          # the unparsed dictionary entry
define -no-color bank       # never emit ANSI (also automatic when piped)
```

On a terminal, `define` with no word opens a line editor:

| key | does |
|---|---|
| Up / Down | walk history — narrowed to what you have typed |
| Right / End / Tab | accept the grey suggestion |
| Enter | define what you typed (never the suggestion) |
| Enter on an empty line | replay the pronunciation, without moving the screen |
| Cmd+Delete (Ctrl-U) | clear the line |
| Ctrl-C | quit, including mid-playback |

Definitions wrap to your terminal width at word boundaries.

**`define` writes to the current directory.** *Every* successful lookup — one-shot,
piped, or in the editor — records the word under `words/` and `events/` where you
started `define`, so your deck and history build themselves:

```
words/sycophantic.yaml     one file per word
events/2026-08-21.yaml     append-only, one file per day (named in UTC)
```

A failed lookup is recorded as history but never enters the deck, so typos are
recallable with Up-arrow without becoming vocabulary. `-raw` records nothing —
scripting a dictionary should not mutate a deck.

```sh
define --forget sycophantic   # drop a word from the deck (history is kept)
DEFINE_NO_CAPTURE=1 define …  # write nothing in this directory
```

`DEFINE_NO_CAPTURE=1` means *nothing at all*, and that includes the event log —
which is what persists your history, so with it set, history is session-only.

The directory *is* the deck: run `define` somewhere else and you get a different
one. If that directory happens to be synced, so is your vocabulary; `define`
neither knows nor cares.

With no word and no terminal, `define` reads stdin: a word defines and speaks it, a bare return
replays the *pronunciation* of the current one — nothing is re-fetched, and the
screen is left as it was provided you let the sound finish — and Ctrl-C quits
silently. `-raw` prints the unparsed entry and never plays. The prompt appears
only on a terminal, so piping stays clean. Flags are session settings — `define
-times 1` opens the loop with single playback.

Exit codes: `0` success; `1` the request failed (no dictionary entry, or
`--forget` found nothing to remove); `2` usage error, which includes an unknown
`/command`. A piped run exits `1` if any word failed and `2` if a command was
malformed, so `echo "$w" | define || …` works in a script; an interactive typo
does not fail the session.

A line beginning with `/` is a command rather than a word — `/` is safe as a
marker because no English headword starts with one, and `define` needs whole
lines for multi-word headwords like `hot dog`. Type `/` to see what there is,
Tab to complete, `/help` to list them. It works the same from every entry mode:
`define /help`, `echo /help | define`, and `/help` typed at the prompt are one
thing.

`/history [N]` lists what you looked up in the last N days — two by default,
counted as local calendar days rather than N×24 hours:

```
  defenestrate  today
  sycophantic   yesterday   2×
  perennial     Aug 1       2×
```

Deduped, and ordered by when each word was **first** seen, so one you keep
returning to holds its place instead of jumping to the top; the count is how
often you have looked it up. Words the dictionary could not find are kept for
up-arrow recall but never listed here — a typo is not vocabulary.

`/sound N` changes how many times a pronunciation plays for the rest of the
session; `/sound` on its own reports it, and `0` turns playback off. It is the
in-session form of `--sound`, which sets it for one run. (`-times` is the older
name for `--sound` and still works; passing both is a usage error rather than a
guess at which you meant.)

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
