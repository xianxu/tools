---
id: 000070
status: working
deps: []
github_issue:
created: 2026-09-16
updated: 2026-09-17
estimate_hours:
started: 2026-09-17T17:20:31-07:00
---

# define: switch between a light and a dark colour scheme

## Problem

`define` is drawn for a dark terminal. The one colour that cannot follow a light
terminal is the target-language tint: xterm-256 colour 236, a FIXED code the
terminal's theme does not remap, so on a white background every tinted row is a
dark bar. `-language-tint light` exists (#66), but only as a launch flag that
defaults to `dark` — nothing notices the terminal is light, nothing remembers the
choice, and nothing changes it mid-session.

## Spec

### What a scheme is

`scheme` is a closed enum `{dark, light}` naming the terminal's background. It
resolves exactly ONE colour: the target-language tint — 236 on dark, 254 on light.

Deliberately not scheme-dependent:

- **The 16 ANSI foregrounds** (cyan headword, yellow part of speech, green deck
  words, grey suggestion). The terminal's theme remaps these; that is what a light
  theme is for.
- **The mark** (`markOn`, 48;5;24 + 38;5;231). A self-contained fg/bg pair,
  legible on either background.
- **Attributes** — inverse selection, dim, bold, italic, underline.

Operator decision (brainstorm 2026-09-17, approach A): swap only the fixed
colours. Per-scheme foreground palettes (B: new output only; C: recolour
everything) were rejected as larger than the problem.

### Scheme state (ARCH-ORDER)

One pure state value per process, shared by the editor and any sitting it starts
(a pointer, the way `deps.bilingual` is shared):

```
schemeState{ choice *{value, source: flag|saved|session}; detected *scheme; asked bool }
effective(state) = choice.value ?? detected ?? dark        (+ the source it came from)
```

- Start: `choice` = `-scheme dark|light` (source `flag`), else the saved file
  (source `saved`), else nil. `-scheme auto` is the same as not giving the flag.
- Event `detected(s)` sets `detected`; it may arrive at any time, late, or twice.
  A later reply overwrites an earlier one. It changes what is on screen only while
  `choice` is nil.
- `/scheme dark|light` REPLACES `choice`, whatever its source — including a flag,
  so `-scheme dark` then `/scheme light` shows light and says so. Persistence
  follows `/bilingual`'s rule (`bilingual_cmd.go:37`), not a new one: a
  successful save → source `saved`; no config directory at all → source `session`,
  reported "(session only; not saved)"; a real write error → the command FAILS
  and nothing changes.
- `/scheme auto` deletes the saved file (same failure rule) and clears `choice`,
  so detection — or the dark fallback — governs again.
- Every transition that changes `effective` has one effect: repaint.

`/scheme` reports `effective` and its source, and each report is true:
`light (detected)`, `dark (detected)`, `dark (saved)`, `light (-scheme flag)`,
`light (session only; not saved)`, and `dark (default: the terminal has not
reported its background)` while nothing is detected — true whether the reply is
pending, unsupported, or was never asked for. Outside an interactive session it
says `(default: detected only in an interactive session)`.

### Detection: a query sent in raw mode, a reply read as a key (ARCH-MOCK)

- Every raw session (the editor, `--play`) writes `OSC 11 ?` (`ESC ] 11 ; ? ESC \`)
  once, right after entering its terminal modes — only when colour is on and the
  tint is on (`-language-tint off` leaves the scheme nothing to change). Nothing
  waits for an answer: the first frame paints with `effective` as it stands, and a
  reply is just another input event.
- The key decoder learns the reply: `ESC ] 11 ; rgb:R/G/B` with 1–4 hex digits
  per component, terminated by BEL or ST (`ESC \`), possibly split across reads.
  It decodes to a `KeyBackground` event carrying the scheme. Both loops (editor,
  sitting) apply it to the shared state; neither ever treats it as typing or an
  answer.
- **The swallow is bounded, byte by byte.** Past `ESC ]` the bytes must continue
  the reply grammar exactly; the first byte that cannot — any other C0 (Ctrl-C
  included), a letter where a hex digit belongs, or a 64-byte cap — aborts, and the
  input decodes exactly as today (`ESC ]` as a 2-byte `KeyUnknown`, `key.go:240`,
  then the rest). So Alt-] with meta-sends-escape, followed by typing or Ctrl-C,
  behaves as it does now (`lessons.md`: "Trusted ANSI parsing is not untrusted
  control filtering").
- A reply that parses as `rgb:` but carries no usable colour is `KeyUnknown`:
  swallowed, nothing detected.
- Classification: Rec. 601 luma on the gamma-encoded components,
  `0.299R + 0.587G + 0.114B`, normalised to 0–1; below 0.5 → dark, otherwise
  light. This is Neovim's background heuristic (prior art), chosen over linear
  luminance because a mid-grey terminal is what users call mid, not dark.
- The query's reply owes the decoder a case, the same obligation
  `enabledModes`' `replies` flag records (`rawterm.go:118`). The query is not a
  mode (no teardown), so it gets its own list, and
  `TestEveryEnabledInputModeIsDecoded` derives from both.
- Paths with no raw session never probe: one-shot lookups, the piped loop,
  `--version`, `--stats`, `--forget`, `--llm-check`, `-raw`. Operator decision:
  one-shot uses flag, then saved, then dark — no blocking probe.

Late replies, the whole class:
- **Inside the session** (any time before exit): an event → at most a repaint.
- **During a `/play` sitting**: the sitting's loop applies it to the shared state.
- **After exit** (the session ended within one terminal round trip of starting —
  a fast quit over a slow link): the reply reaches whatever reads the terminal
  next, usually the shell. Accepted and documented; it needs a quit faster than
  the terminal's answer.
- Nothing reads stdin in cooked mode after the query is sent: the deck question
  (`repl.go:295`) and `--play`'s queue build (`play_loop.go:84`) both finish
  before raw mode begins.

### Repaint: the role is frozen, the colour is not

- `rowPaint.background` holds the escape string today, fixed when the row is
  produced (`output_layout.go:10`: "later policy changes do not recolor history").
  It becomes a ROLE (`tinted bool`). WHETHER a row is tinted stays frozen at
  production — that is the `/lang` rule, and it survives. WHICH colour a tint is
  gets resolved at paint.
- Every carrier of the escape changes with it: `options.tintBackground`
  (`main.go:462,672`, the root) becomes the tint on/off plus the resolved scheme;
  `tintPolicy.background`; `play.PresentationRegion.Background` (a question holds
  it until reveal); and the equality checks against `languageDark`/`languageLight`
  in `validRowPaint` (`output_layout.go:24`) and `answerwrap.go:352`. The `play`
  package stops carrying background escapes.
- **The scheme reaches the painter as an explicit parameter, never a global**
  (ARCH-PURE): `paintLanguageRow` takes it, and so do its callers.
  - The screen (`*screen`) holds the scheme in effect; `liveScreen.SetScheme`
    stores it and repaints. That covers every `*screen` painter: the frame
    (`layoutSelectionFrame`, `selectionLayout.paint` → `paintOutputChunk`),
    `paintActivity`, and the exit transcript (`paintedTranscript`, which already
    paints from `screen.paints`).
  - The non-screen path (`writeOutput` → `serializeOutput`: one-shot, piped and
    answer output) takes the scheme from `options`, resolved when the output is
    written.
- **Dead code is deleted, not converted**, once each is shown unreachable in
  production (both reviewers confirmed): `styleLanguageText` (only production
  caller runs with `ro.Tint` zeroed, `definitions.go:89`) with `dictionaryFragment`
  and the tint half of `dictionaryText` (`dictionary_language.go:116-144`;
  `projectDictionaryText` stays, `bilingual_layout.go:105`); the test-only baked
  renderers `renderOutputText` (`output_screen.go:146`), `renderDefinitions`
  (`definitions.go:55`), `renderPracticePresentation` (`practice_language.go:19`),
  `practiceChrome` (`:125`), `boardFooter` (`play_loop.go:781`); and
  `styledBoardPrompt` (`practice_language.go:129`, no callers at all). The tint
  assertions their tests make are PORTED to `renderDefinitionOutput` +
  `serializeOutput`, not dropped (`dictionary_source_test.go`, `dict_test.go`,
  `dictionary_language_test.go`, `practice_language_test.go`).

### Persistence (ARCH-SECURE, ARCH-FUNERAL)

- A one-word file, `<config>/define/scheme`, holding `dark` or `light`. It follows
  the one-file-per-setting precedent: `store.ReadScheme` / `WriteScheme` /
  `ClearScheme` in `store/scheme.go` beside `store/bilingual.go`, taking a
  directory and reusing `writeBytesAtomic` (`store/yaml.go:419`).
- It is define's first USER-level setting, because a scheme belongs to the
  terminal, not the deck.
- `<config>` comes from its own seam, `deps.configDir`, not from `getenv`: `getenv`
  is the model seam, and some tests make it panic (`practice_help_paths_test.go:66`).
  `realDeps` resolves `$XDG_CONFIG_HOME` if it is absolute, else `$HOME/.config` if
  `$HOME` is absolute, else none. None means "no saved setting" and "session only".
  Test deps leave it unset, so no test can reach a real config.
- The file is untrusted input (hand-edited, truncated, or written by a newer
  version). Reads are capped at 64 bytes, trimmed, and parsed into the closed
  enum. Anything else → one warning on stderr, treated as unset.
- `ClearScheme` removes the file and then the `define/` directory if it is empty.
  Residue: at most one directory and one file of ≤ 6 bytes.
- The one-shot form saves, like `/lang` and `/bilingual` (`command.go:214-219`):
  `define /scheme light` writes the file, `define /scheme auto` clears it, and a
  bare `define /scheme` reports.

### Flags

- `-scheme auto|dark|light`, default `auto`.
- `-language-tint on|off`, default `on`. Operator decision: narrowed, not aliased.
  `-language-tint dark|light` is a usage error that names `-scheme dark|light`.
- Docs: `cmd/define/README.md:344`, the `-h` prose (`main.go:560`), the
  command-list spans pinned by `doc_sync_test.go`, and `atlas/define.md` (incl.
  the `-language-tint` line at `:2270`).

### Test migration forced by the change

- Move off `-language-tint dark|light`: `TestPTYLanguageTint`
  (`pty_conformance_test.go:1245`), `TestPTYNativeRendirSectionLayout`
  (`pty_layout_conformance_test.go:43`), `TestLanguageTintInvocation`
  (`language_style_paths_test.go:19`). `TestLanguageTintInvalidFlagBeforeStore`
  (`:93`) asserts the old error wording.
- The pty harness always sets an isolated `XDG_CONFIG_HOME`: `startDefine*` sets
  `cmd.Env` only when `env` is non-empty (`pty_conformance_test.go:89`), and the
  layout test builds its own `exec.Command`. Otherwise a developer's saved scheme
  flips the "default" expectations.
- Captured raw-session transcripts now begin with the OSC 11 query. The tests'
  terminal readers (`readFrame`, `rowTestCells` in `language_row_test.go:26`)
  must skip OSC sequences the way a terminal does, not count them as cells.
- A pty terminal that does not answer is the default, and it is what existing
  expectations (dark) already assume. Tests that play a light or dark terminal
  write the reply themselves, as `TestPTYLanguageTint` already writes input.

### Operating envelope (ARCH-CONSTRAINTS)

- **Startup:** no added latency — the query is written and nothing waits for it.
  8 bytes per raw session.
- **A reply:** one decode plus at most one repaint (the cost of a scroll), and a
  repaint only when `effective` changes.
- **`/scheme`:** one file write plus one repaint.
- **Decoder:** at most 64 bytes buffered for a pending reply.

### Out of scope

- Following a live appearance change mid-session (DEC mode 2031 notifications) —
  the paint-time role plus the `KeyBackground` event make it cheap to add later.
- Per-scheme foreground palettes.
- Detection outside a raw session (one-shot, piped).

## Done when

- A session on a terminal that reports a light background paints tinted rows in
  254, and on a dark one in 236, with no flag and no saved file. Pinned in-process
  (driving `runEditor` with the reply as input) and by pty tests that play each
  terminal; checked by hand in Terminal.app, iTerm2 and Ghostty, both appearances.
- A reply arriving after content is on screen repaints the rows already there;
  with a `choice` in force, it changes nothing visible.
- `/scheme light` in the editor repaints rows already on screen AND the exit
  transcript in 254 — a test driving `runEditor`. `/scheme` in the piped loop is
  pinned through `replLines`; in a sitting, the reply is applied by the sitting's
  loop.
- `/scheme light` persists (a new process with no flag uses it). `/scheme auto`
  removes it and detection governs again. A missing config directory reports
  "session only; not saved". A write error fails the command and changes nothing.
  A garbled file warns once and is ignored.
- An OSC 11 reply in the editor inserts no text, and in a sitting records no
  answer. Alt-] followed by typing, and Alt-] followed by Ctrl-C, behave exactly
  as today. Pinned in the decoder, and through each loop shell.
- `-language-tint off` disables the tint and the query. `-language-tint light` is
  refused naming `-scheme light`. `-scheme light` then `/scheme dark` reports
  `dark (saved)`. The state transitions are unit-tested as event sequences
  (reply before/after a choice, a duplicate reply, a late reply).
- Every dead path listed is deleted, with its tint assertions ported and passing.
- Every new test has been observed failing with its fix removed (mutation
  applied, compiled, run with `-count=1`).
- README, `-h` and atlas are updated, and the doc-sync tests are green.

## Plan

- [ ]

## Log

### 2026-09-16

### 2026-09-17

Brainstorm. The only fixed colour a light terminal cannot remap is the language
tint (236/254); foregrounds are the terminal theme's job, so the scheme swaps the
tint alone (approach A). An explorer mapped the tint: it reaches `screen.lines`
only as `rowPaint` metadata, but as a literal escape frozen at production — so
recolouring history means making the background a role resolved at paint, not
re-rendering. Found: a mid-session OSC reply would be typed into the prompt
(`key.go:240`), and `styleLanguageText` is unreachable in production
(`definitions.go:89`). Operator chose auto-detect with overrides, a user-level
saved file (not an env var), and narrowing `-language-tint` to `on|off`.

Spec review, round 1 (fresh-context reviewer, code-verified): 10 issues, most
from ONE decision — a blocking startup probe that owns stdin before the session
does. It forced a typeahead handoff that cannot be expressed (`replRaw` needs an
`*os.File`; the deck question is cooked), turned Ctrl-C into a byte, scattered
late replies across the deck question, one-shot output and the shell, and pulled
in per-OS termios code. Pivot (the reviewer's alternative): send the query AFTER
raw mode and read the reply as a key event; since the tint colour now resolves
at paint, a reply at any time is just a repaint. Operator chose sessions-only
detection (one-shot: flag, then saved, then dark). Also fixed: a dark reply was
"unknown" (now `dark (detected)`, Rec. 601 luma); the source enum could not say
`/scheme` after a flag or a session-only switch (now a `choice` that `/scheme`
replaces); the OSC swallow is bounded byte-by-byte so Alt-] and Ctrl-C are
unchanged; the scheme reaches painters as a parameter; the save-failure rule is
`/bilingual`'s rather than a new one; dead renderers and test migrations are
enumerated. DA1 dropped — it existed only to end a wait that no longer exists.
