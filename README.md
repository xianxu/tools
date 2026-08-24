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
define                      # interactive: type a word, / for commands, ^C to quit
echo sycophantic | define   # or feed it words on stdin
define sycophantic          # definition + /ˌsikəˈfan(t)ik/, played 3x
define /history             # a command works as an argument too
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

**Type a question and it is answered instead of looked up.** There is no mode and
no prefix to remember:

```
› sycophantic                            # a word: the dictionary entry
› what's the difference to obsequious?   # a question: answered by the model
› hot dog                                # still a word — two of them
```

The dictionary decides which is which, and that is why multi-word headwords keep
working: `define` asks it first, and only classifies what it does not have. So
`hot dog` and `a priori` are definitions, while a line it has no entry for that
reads as a question — a wh-word, a question mark, or a request like `use it in a
sentence` — goes to the model. Anything else is still a miss, so a typo says
`no dictionary entry` rather than starting a conversation.

Both directions have a one-key escape, and neither is the only way to reach its
outcome:

| prefix | means |
|---|---|
| `?` | ask, even if it is a word — `?why` asks about *why* instead of defining it |
| `\` | define, even if it reads as a question — `\how so` answers `no dictionary entry` |

The answer is streamed, and **Ctrl-C stops the answer rather than the session** —
you land back at the prompt with the word you were reading still current.

What the model is told is the directory you are in: the word on screen and its
dictionary entry, what you have looked up this session, your recent deck,
`user-model.md` if you keep one, and the earlier questions in this session — so a
follow-up like `give me two more examples` resolves against the answer before it.
Nothing is remembered between runs except the files, which means a fresh process
answers as well as a long-running one and you can read the context with `cat`.

Questions need a model configured (see `--llm-check` below); without one, `define`
says so and exits `1` rather than looking up a sentence.

**`-raw` never asks**, on either route: it is the scripting form, so an unforced
miss stays a miss, and an explicit `?` alongside it is a usage error (exit `2`)
rather than a guess at which of the two contradicting flags you meant.

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
--sound 1` opens the loop with single playback.

## Checking the model connection

`define` can use a language model for the parts a dictionary cannot do. Every one
of those features **degrades silently by design** — no key or no network means
they are skipped, not failed, so a review session is never blocked on a third
party. That makes a misconfiguration invisible, which is what this flag is for:

```sh
define --llm-check
```

```
  base url  http://127.0.0.1:8317
  model     claude-opus-5 (effort high)
  key       (set, short)
  latency   1.379s
  tokens    22 in, 5 out (0 thinking)
  preamble  1902 tokens injected upstream (not ours)
  answer    "PONG"
  ok
```

It is the one surface where an unusable configuration is **loud**: it exits
non-zero and names the reason. Configure it with `DEFINE_LLM_API_KEY` (or `ANTHROPIC_API_KEY`),
`DEFINE_LLM_BASE_URL`, `DEFINE_LLM_MODEL`, `DEFINE_LLM_EFFORT` and
`DEFINE_LLM_TIMEOUT` (a duration, e.g. `90s`) — all five the tool reads. The
default base URL is a local proxy on `127.0.0.1:8317`.

Exit codes: `0` success; `1` the request failed (no dictionary entry, a question
with no model configured, `--forget` found nothing to remove, or `--llm-check`
found no usable model configuration); `2` usage error, which includes an unknown
`/command` and a bare `?` or `\` with nothing after it. A piped run exits `1` if any word failed and `2` if a command was
malformed, so `echo "$w" | define || …` works in a script; an interactive typo
does not fail the session.

A line beginning with `/` is a command rather than a word — `/` is safe as a
marker because no English headword starts with one, and `define` needs whole
lines for multi-word headwords like `hot dog`. The same reasoning picks `?` and
`\` for the two question hatches above: no headword begins with either. Type `/` to see what there is,
Tab to complete, `/help` to list them. It works the same from every entry mode:
`define /help`, `echo /help | define`, and `/help` typed at the prompt are one
thing.

`/history [N]` lists what you looked up in the last N days — two by default,
counted as local calendar days rather than N×24 hours. `N` can be written three
ways, so it reads the same whichever you reach for: `/history 7`,
`/history --days 7`, `/history --days=7`. It works from every entry mode, so
`define /history 7` and `echo '/history 7' | define` mean the same thing.

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
