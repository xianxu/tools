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

### Resolution

`effective = -scheme flag ?? saved ?? detected ?? dark` — a pure function that also
returns the SOURCE (`flag | saved | detected | fallback`), so `/scheme` can say
where the value came from.

### Detection — the terminal is an external dependency (ARCH-MOCK)

- Runs ONCE, early in `run`, before anything reads stdin, and only when stdin AND
  stdout are terminals, colour is on (no `-no-color`, `TERM != dumb`), and `-scheme`
  was not given. Otherwise `detected` is unknown.
- Sends `OSC 11 ?` then DA1 (`ESC [ c`) in raw mode. Terminals answer in order and
  practically all answer DA1, so **the DA1 reply is the fence**: it ends the wait
  whether or not an OSC 11 reply came first. A terminal that ignores OSC 11 costs
  one round trip, not a timeout.
- Hard cap 250 ms, reached only by a terminal that answers neither. Reads use a
  kernel read timeout (termios `VMIN=0`/`VTIME`), so no goroutine outlives the
  probe (ARCH-ORDER: extent is lexical).
- Pure parser: `rgb:R/G/B` with 1–4 hex digits per component, terminated by BEL or
  ST (`ESC \`), possibly split across reads. Luminance > 0.5 → light; anything
  else → unknown.
- Bytes read during the probe that are not part of a reply (typeahead) are handed
  to the next reader, in order — never discarded.
- **The key reader swallows a whole OSC sequence** (`ESC ] … BEL|ST`) as one
  ignored key. Today `ESC ]` decodes as a 2-byte `KeyUnknown` and the payload
  arrives as `KeyRune`s (`key.go:240`, `key.go:159-166`), so a late reply would be
  typed into the prompt, or read as an answer in a sitting.
- Known limit: a reply arriving AFTER the cap, while the cooked-mode deck question
  is open, is echoed into that answer. It needs a terminal that answers neither
  query within 250 ms.

### Repaint: the role is frozen, the colour is not

- `rowPaint.background` holds the escape string today, fixed when the row is
  produced (`output_layout.go:10`: "later policy changes do not recolor history").
  It becomes a ROLE (tinted or not). WHETHER a row is tinted stays frozen at
  production — that is the `/lang` rule, and it survives. WHICH colour a tint is
  resolves in `paintLanguageRow`, from the active scheme, at paint.
- The same change applies to the copies that carry the escape today:
  `tintPolicy.background` and `play.PresentationRegion.Background` (a question
  holds it until reveal). The `play` package stops carrying background escapes.
- `liveScreen` holds the active scheme. `/scheme` sets it and repaints, so history
  and the exit transcript (`paintedTranscript`, painted from `screen.paints`) take
  the new colour. One-shot and piped output resolve with the scheme in effect at
  write time.
- Dead paths are DELETED rather than converted, once shown unreachable:
  `styleLanguageText` (its only production caller runs with a zeroed tint,
  `definitions.go:89`) and the test-only baked renderers.

### Persistence and `/scheme`

- The saved value is a one-word file: `$XDG_CONFIG_HOME/define/scheme`, else
  `$HOME/.config/define/scheme`. It follows the one-file-per-setting precedent
  (`bilingual.txt`, the lang file), and it is define's first USER-level setting,
  because a scheme belongs to the terminal, not the deck. Written atomically
  (temp + rename, reusing the store's helper). Read capped at 64 bytes; anything
  unparseable → one warning on stderr, treated as unset.
- The path resolves through `deps.getenv`, so no test reaches the real config. The
  pty harness sets an isolated `XDG_CONFIG_HOME`.
- `/scheme` reports: `light (detected)`, `dark (saved)`, `dark (-scheme flag)`,
  `dark (default: the terminal did not report its background)`.
- `/scheme light|dark` switches, repaints and saves. `/scheme auto` deletes the file
  and returns to the detected value (or the dark fallback).
- A failed save still switches the session and SAYS it was not saved — the message
  is a record and must be true.
- Registered like `/sound`, so both loops get it (editor and piped). A sitting
  inherits the scheme in effect when it starts.

### Flags

- `-scheme auto|dark|light`, default `auto`.
- `-language-tint on|off`, default `on`. Operator decision: narrowed, not aliased —
  `-language-tint dark|light` is a usage error that names `-scheme dark|light`.
- Docs: `cmd/define/README.md:344`, the `-h` prose, the command-list spans pinned
  by `doc_sync_test.go`, `atlas/define.md`.

### Operating envelope (ARCH-CONSTRAINTS)

- **Startup path.** The probe costs one terminal round trip (local ≈ 1 ms; over SSH,
  the link RTT) on every launch that probes, one-shot lookups included. The 250 ms
  cap is an operator-tunable guess, reached only by silent terminals.
- **`/scheme`**: one repaint, the cost of a scroll.
- **Residue (ARCH-FUNERAL):** one file of at most 6 bytes, removed by `/scheme auto`.
  Nothing else durable.

### Out of scope

- Following a live appearance change mid-session (DEC mode 2031 notifications) —
  the paint-time role makes it cheap to add later.
- Per-scheme foreground palettes.
- Probing when stdin is not a terminal (`echo word | define` on a TTY): falls back
  to saved, then dark.

## Done when

- With no flag and no saved file, a terminal reporting a light background gets
  tinted rows in 254 and a dark one gets 236 — pty tests that play each terminal,
  plus a manual check in Terminal.app, iTerm2 and Ghostty in both appearances.
- A terminal answering only DA1 resolves `dark` without waiting for the cap; a
  silent one resolves `dark` within the cap (timed pty tests).
- `/scheme light` in the editor repaints rows already on screen AND the exit
  transcript in 254 — pinned by a test that drives `runEditor`; `/scheme` in the
  piped loop pinned through `replLines`.
- `/scheme light` persists: a new process with no flag uses it. `/scheme auto`
  removes the file and detection governs again. A garbled file warns once and is
  ignored.
- An OSC 11 reply delivered mid-session to the editor and to a sitting changes
  nothing: no inserted text, no answer.
- Typeahead during the probe survives into the first line read.
- `-language-tint off` disables the tint; `-language-tint light` is refused naming
  `-scheme light`. The precedence table is unit-tested.
- Every new test has been observed failing with its fix removed (mutation applied,
  compiled, run with `-count=1`).
- README, `-h` and atlas updated; the doc-sync tests are green.

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
