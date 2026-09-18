---
id: 000070
status: working
deps: []
github_issue:
created: 2026-09-16
updated: 2026-09-17
estimate_hours: 6.1
started: 2026-09-17T17:20:31-07:00
flow: {kind: full, provenance: inferred}
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

ONE holder per process — `deps.scheme`, a stable pointer on the
`deps.practiceHelp` precedent (`main.go:25-27`: "a pointer, so a nested sitting
and the next one share"), NOT the `deps.bilingual` one, whose setter replaces the
pointer in a by-value copy of `deps` and so never reaches the editor from a
sitting. The holder is an `atomic.Pointer` to an IMMUTABLE `schemeState`; a pure
transition function builds the next value and the loop stores it:

```
schemeState{ choice *{value, source: flag|saved|session}; detected *scheme }
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

**Nothing holds a resolved copy.** Every screen (the editor's, a sitting's, a
suspended one) and the non-screen writer hold the holder and read `effective()`
when they paint or write. So a transition anywhere is in force everywhere at the
next paint, and the only effect a transition has is to repaint the ACTIVE screen:
a suspended editor screen repaints on `resume` (`play_cmd.go:148`) with whatever
is in force then, and a sitting's new screen starts with it. `paintLanguageRow`
still takes the scheme as an explicit parameter (ARCH-PURE); only its callers read
the holder. A nil holder means dark, read-only: it paints dark, and `/scheme`
refuses with a message rather than silently doing nothing (only reachable from
tests that drive a loop with bare test deps; `run()` always builds the holder).

Concurrency: exactly ONE writer at a time — the loop goroutine in force (the
editor's, the piped loop's, or `--play`'s; a `/play` sitting runs on the editor
loop's own goroutine). FIVE goroutines read while painting under the screen lock
`l.mu`: the loop (`Draw`/`DrawOutput`/`WriteOutput` → `repaint`), the throttle
timer (`flush`, `screen.go:935,944-950`), the activity ticker (`activity.go:54` →
`throttledPaint`), `readInput` (pointer routing → `repaint`,
`selection_input.go:28-49`), and clipboard completion (`copyFinishedLocked`,
`selection_screen.go:141-152`). An atomic load of an immutable value has no lock to
order against `l.mu` and the pointer router's lock, which is why it is an atomic
pointer rather than a mutex. A frame reads `effective()` ONCE, in
`layoutSelectionFrame`, and passes the value down, so a transition landing
mid-frame cannot mix two tints in one frame.

`/scheme` reports `effective` and its source, and each report is true:
`light (detected)`, `dark (detected)`, `dark (saved)`, `light (-scheme flag)`,
`light (session only; not saved)`, and `dark (default: the terminal has not
reported its background)` while nothing is detected — true whether the reply is
pending, unsupported, or never asked for. The one-shot form, which has no session
to detect in, says `(default: detected only in an interactive session)`.

### Detection: a query sent in raw mode, a reply read as a key (ARCH-MOCK)

- **Sent once per raw session**, at mode entry, to `rawSession.control`
  (`rawterm.go:31`) beside `enterModes` — never through a screen, where
  `scanEscape` would read `ESC ]` as a 2-byte escape (`sgr.go:92-95`) and leave
  `11;?` in the frame and the exit transcript. A `/play` sitting borrows the
  editor's raw session and does not send it again; `--play` on its own sends it.
- **Sent only when a tint can appear:** colour on, `-language-tint on`, not `-raw`,
  and `TERM` not `dumb` (which already turns the tint off, `main.go:609-611`, and
  would print the query as text).
- The query is `OSC 11 ?` (`ESC ] 11 ; ? ESC \`). Nothing waits for an answer: the
  first frame paints with `effective` as it stands, and a reply is just another
  input event.

**The swallow and the colour parse are two steps, so no reply format can leak.**
Once the query is sent a reply is certain, and in a sitting a leaked character is
an ANSWER (digits pick options; `d` removes the word).

1. **Swallow** — `ESC ] 11 ;` then bytes in 0x20–0x7E up to BEL or ST
   (`ESC \`), at most 64 bytes, possibly split across reads. Any byte outside
   0x20–0x7E other than the terminators (Ctrl-C, Enter, DEL, 0x80+, an `ESC` not
   followed by `\`) or the cap aborts, and the input
   decodes exactly as today: `ESC ]` as a 2-byte `KeyUnknown` (`key.go:240`), then
   the rest. Before `11;` is complete the decoder waits for the next byte, which
   holds a lone Alt-] until the next key (it is inert either way). A user would
   have to type `11;` straight after Alt-] to enter the swallow at all.
2. **Parse** the swallowed payload: `rgb:R/G/B` with 1–4 hex digits per component
   → `KeyBackground` carrying the scheme. Anything else — `rgba:`, `#rrggbb`,
   garbage — → `KeyUnknown`: swallowed, nothing detected.

Classification: Rec. 601 luma on the gamma-encoded components,
`0.299R + 0.587G + 0.114B`, normalised to 0–1; below 0.5 → dark, otherwise light.
This is Neovim's background heuristic (prior art), chosen over linear luminance
because a mid-grey terminal is what users call mid, not dark.

**`KeyBackground` is a new key kind, so every consumer of keys is enumerated**
(ARCH-PURPOSE):

- `runEditor` applies it to the holder BEFORE `Apply` — it is never typing.
- The sitting applies it via its `sittingKeyHandling` entry (`play_loop.go:570`),
  which `TestEveryKeyKindIsDecidedForASitting` forces — it is never an answer.
- The pointer router classes it as NOT input, like `KeyUnknown`, so a reply
  mid-drag does not cancel the selection or its notice (`selection_input.go:48-49`,
  `:234-243`).
- BOTH full-channel drop sites — the length check (`selection_input.go:192-200`)
  and the `select`'s `default:` arm (`:216-221`) — drop it SILENTLY: no
  "input full" notice, no pointer cancel, and the `saturated` latch NOT set, or a
  dropped reply would suppress the notice for the next real key dropped. A
  dropped reply leaves the scheme where it was, which is an accepted outcome.

The query's reply owes the decoder a case — the obligation `enabledModes`'
`replies` flag records (`rawterm.go:118`). The query is not a mode (no teardown),
so it gets its own list, and `TestEveryEnabledInputModeIsDecoded` derives from both.

Paths with no raw session never send it: one-shot lookups (including one-shot
`-raw`), the piped loop, `--version`, `--stats`, `--forget`, `--llm-check`.
Operator decision: one-shot uses flag, then saved, then dark — no blocking probe.

Late replies, the whole class:
- **Inside the session** (any time before exit): an event → at most a repaint.
- **During a `/play` sitting**: the sitting's loop applies it to the shared holder;
  the editor sees it on resume.
- **After exit** (the session ended within one terminal round trip of starting —
  a fast quit over a slow link): the terminal is cooked again, so the reply is
  echoed as visible junk and reaches whatever reads next, usually the shell.
  Accepted and documented; it needs a quit faster than the terminal's answer.
- Nothing reads stdin in cooked mode between sending the query and exit: the deck
  question (`repl.go:295`) and `--play`'s queue build (`play_loop.go:84`) both
  finish before `enterRaw`, and the only reader until the restore is `readInput`
  (`replraw.go:34`, `play_loop.go:117`).

### Repaint: the role is frozen, the colour is not

- `rowPaint.background` holds the escape string today, fixed when the row is
  produced (`output_layout.go:10`: "later policy changes do not recolor history").
  It becomes a ROLE (`tinted bool`). WHETHER a row is tinted stays frozen at
  production — that is the `/lang` rule, and it survives. WHICH colour a tint is
  gets resolved at paint.
- Every carrier of the escape changes with it: `options.tintBackground`
  (`main.go:462,672`, the root) becomes the tint on/off alone — no resolved scheme;
  `tintPolicy.background`; `play.PresentationRegion.Background` (a question holds
  it until reveal); and the equality checks against `languageDark`/`languageLight`
  in `validRowPaint` (`output_layout.go:24`) and `answerwrap.go:352`. The `play`
  package stops carrying background escapes.
- **The scheme reaches the painter as an explicit parameter, never a global**
  (ARCH-PURE): `paintLanguageRow` takes it, and so do its callers.
  - The screen (`*screen`) holds the `deps.scheme` holder — attached in
    `newConsole` and `sittingInPlace`, not threaded through the constructors
    (`newLiveScreen`/`newPinnedScreen` have ~108 test call sites, and
    `newConsole` takes the constructor as a value, `replraw.go:77-78`) — and
    reads `effective()` once per frame. It is attached BEFORE the screen is
    shared with another goroutine — in `newConsole` before `newPointerRouter` and
    `watchResize` (`replraw.go:81-100`), in `sittingInPlace` before
    `pointer.activate` (`play_cmd.go:112-119`) — because the screen's field is
    not atomic.
    That covers every `*screen` painter: the frame (`layoutSelectionFrame`,
    `selectionLayout.paint` → `paintOutputChunk`), `paintActivity`, and the exit
    transcript (`paintedTranscript`, which already paints from `screen.paints`).
    The display fakes (`recordDisplay`, `editorloop_test.go:275`) gain whatever
    repaint hook the loops call.
  - The non-screen path (`writeOutput` → `serializeOutput`: one-shot, piped and
    answer output) reads `effective()` from the same holder when the output is
    written. `options` no longer carries a scheme, so `tintPolicy` carries the
    holder from `d` (`tintFor` takes it; `ownedAnswerWrapWriter` sees only its
    `tintPolicy`, `answerwrap.go:358`).
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
  bare `define /scheme` reports. With no config directory it REFUSES (exit 2,
  naming `$XDG_CONFIG_HOME`/`$HOME`), as `/bilingual`'s one-shot does
  (`bilingual_cmd.go:58-61`) — there is no session for "session only" to mean.

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
  "session only; not saved" in a session, and the one-shot `define /scheme light`
  exits 2 naming `$XDG_CONFIG_HOME`/`$HOME`. A write error fails the command and
  changes nothing. A garbled file warns once and is ignored.
- A reply dropped on a full input channel posts no notice and leaves the next
  real dropped key's "input full" notice intact.
- An OSC 11 reply in the editor inserts no text, and in a sitting records no
  answer — for an `rgb:` reply AND for an `rgba:` / `#rrggbb` one (swallowed,
  nothing detected). A reply mid-drag leaves the selection intact. Alt-] followed
  by typing, and Alt-] followed by Ctrl-C sent as SEPARATE writes (so the decoder
  is really waiting), behave exactly as today. Pinned in the decoder, the pointer
  router, and through each loop shell.
- A reply applied during a `/play` sitting is in force in the editor after the
  sitting ends, and a sitting starts with the editor's scheme.
- `-language-tint off`, `-raw` and `TERM=dumb` send no query; `-language-tint off`
  disables the tint. `-language-tint light` is
  refused naming `-scheme light`. `-scheme light` then `/scheme dark` reports
  `dark (saved)`. The state transitions are unit-tested as event sequences
  (reply before/after a choice, a duplicate reply, a late reply).
- Every dead path listed is deleted, with its tint assertions ported and passing.
- Every new test has been observed failing with its fix removed (mutation
  applied, compiled, run with `-count=1`).
- README, `-h` and atlas are updated, and the doc-sync tests are green.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec              design=1.5  impl=0.1
item: scope-pivot             design=0.35 impl=0.08
item: smaller-go-module       design=0.05 impl=0.2
item: cross-cutting-refactor  design=0.1  impl=0.2
item: cross-cutting-refactor  design=0.1  impl=0.2
item: atlas-docs              design=0.05 impl=0.08
item: milestone-review        design=0.0  impl=0.2
item: smaller-go-module       design=0.05 impl=0.2
item: smaller-go-module       design=0.05 impl=0.2
item: smaller-go-module       design=0.0  impl=0.12
item: atlas-docs              design=0.05 impl=0.08
item: milestone-review        design=0.0  impl=0.2
item: greenfield-go-module    design=0.2  impl=0.3
item: tui-screen              design=0.2  impl=0.3
item: real-api-discovery      design=0.0  impl=0.2
item: atlas-docs              design=0.05 impl=0.08
item: milestone-review        design=0.0  impl=0.2
design-buffer: 0.15
total: 6.10
```

Derivation, row by row (v2 ranges; `impl=` written at 40% of them per v3.1):
- **Design already spent** — `issue-spec` at the top of its range (1.5): the brainstorm, four spec review rounds, three plan review rounds and the plan gate. `scope-pivot` (mid of 0.2–0.5) for the redesign round 1 forced: blocking startup probe → reply read as a key event.
- **M1** — `smaller-go-module` for `Scheme` + `schemeState`/holder + parsers (Tasks 1–2); two `cross-cutting-refactor`s, one for the dead-path deletion (Task 3) and one for the role type change with its ~155 test references and the flags (Tasks 4–5), each at the top of the impl range; `atlas-docs`; one `milestone-review`.
- **M2** — `smaller-go-module` for the store functions + config seam + `applyScheme`/`describeScheme` (Tasks 7–9); another for `/scheme` in three contexts with its loop-shell tests (Task 10); a smaller one for the pty harness isolation (Task 11); `atlas-docs`; `milestone-review`.
- **M3** — `greenfield-go-module` for the OSC decoder, colour parse and query (Tasks 13–15; Step 2.5: `termenv` can query OSC 11 but only blocking, which the spec rejects, so no halving); `tui-screen` for the four consumers in the two loops (Task 16); `real-api-discovery` for real terminals' replies (conformance + the manual three-terminal check); `atlas-docs`; `milestone-review` (+ close).
- **Step 3** — the plan pre-resolves the M1–M3 design, so those primitives take ×0.2 design; hence the +15% buffer, not +30%. **Familiarity** 1.0: the same codebase and seams (`screen`, `readInput`, the command registry) as the last several issues.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* (The calibration doc is flagged stale by `sdlc estimate-source`; numbers provisional.)

## Plan

Durable plan: `workshop/plans/000070-light-dark-colour-scheme-plan.md` (tasks,
code, tests, mutation checks). Three review boundaries:

- [x] M1 — the tint is a role, the scheme is state: `store.Scheme`,
  `schemeState` + atomic `schemeHolder`, `rowPaint.tinted` resolved at paint,
  dead tint paths deleted (renderers moved to test helpers), `-scheme` and
  `-language-tint on|off` (plan Tasks 1–6)
- [x] M2 — `/scheme` and the saved choice: `store.Read/Write/ClearScheme`, the
  `deps.configDir` seam, `applyScheme` (persist-then-switch) and
  `describeScheme`, the command in editor/piped/one-shot, pty harness
  isolation, docs (Tasks 7–12)
- [ ] M3 — detection: `parseBackgroundColour`, the bounded OSC decoder and
  `KeyBackground`, the query at raw-mode entry, every consumer of the key kind,
  conformance terminals, manual check in three terminals (Tasks 13–18)

## Log

### 2026-09-16

### 2026-09-17
- 2026-09-17: closed M2 — M2: /scheme (report | light|dark saves+repaints | auto forgets) in raw editor, piped loop and one-shot; saved file under $XDG_CONFIG_HOME/define via one schemePersister seam (load/save/clear; capped, enum-parsed; ClearScheme never removes a symlinked dir); persist-then-switch per /bilingual. Review round 1 fixed: M2 Log entry (BR-6), schemeArg one field, loopKind replaces session+fullScreen, pure initialSchemeState. go test ./... green outside sandbox; pty conformance incl. TestPTYSavedSchemeSurvivesARestart green on built binary; full tagged suite and -race fail only the pre-existing set. 19 mutations each applied, compiled, reddened.; review verdict: FIX-THEN-SHIP
- 2026-09-17: closed M1 — M1: tint is a role (rowPaint.tinted), shade resolves at paint from one atomic schemeHolder attached in newConsole+sittingInPlace; -scheme dark|light|auto, -language-tint on|off. go test ./... green (pty tests outside sandbox); -race and tagged conformance: only failures reproduce on branch point 75370a2 (13 pty + live-LLM + a -race timing test) — logged; TestPTYLanguageTint/TestPTYNativeRendirSectionLayout green on the built binary. 27 mutations each applied, compiled and reddened their pin.; review verdict: SHIP

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

Spec review, round 2: round 1 all resolved; four new. (1) Several copies of the
scheme with nothing keeping them in step — the `deps.bilingual` precedent
replaces a pointer in a by-value `deps`, so a sitting's reply would never reach
the editor → ONE holder on the `deps.practiceHelp` precedent, and nothing holds
a resolved copy: every screen and writer reads `effective()` at paint, so the
only effect is repainting the active screen. (2) `KeyBackground` is a new key
kind with four consumers, not two — the pointer router and the full-channel
branch would have treated it as input. (3) The swallow grammar WAS the `rgb:`
grammar, so any other reply format (`rgba:`) leaked as typing — and as ANSWERS
in a sitting; now swallow-then-parse. (4) One-shot `/scheme` with no config dir
refuses, as `/bilingual`'s does. Also: query goes to `rawSession.control`, never
the screen; not sent under `-raw` or `TERM=dumb`; `asked` dropped (no reader).

Spec review, round 3: round-2 fixes verified against the code; two small issues.
(1) Holder concurrency named one reader where there are five painting goroutines
under `l.mu` → an `atomic.Pointer` to an immutable state (single writer: the loop
in force), so there is no lock to order; one `effective()` read per frame.
(2) The full-channel class had two drop sites, not one (`:216-221`), and a silent
drop must not set the `saturated` latch. Advisories adopted: nil holder = dark;
holder attached in `newConsole`/`sittingInPlace`, not the ~108-call-site
constructors; `tintPolicy` carries the holder to the non-screen path; the swallow
byte range stated exactly; a Done-when for the one-shot refusal.

Spec review, round 4: ✅ approved. Two advisories folded in — attach the holder
before any painter goroutine can see the screen, and `/scheme` on a nil holder
refuses rather than silently doing nothing.

Plan written (`workshop/plans/000070-light-dark-colour-scheme-plan.md`) and
reviewed per chunk by three code-verifying reviewers; all three found issues,
fixed in one rewrite. The committed first draft had turned the suite RED: the
repo's plan guards parse plans (`TestPlanTablesNameEntitiesThatExist` wants one
symbol and one path per Core-concepts row; `TestPlanCitesTestsThatExist` reads a
backticked `Test*` as a claim it exists; a `| deleted |` row asserts absence, so
it lands with its deletion). Spec revisions the plan makes, recorded here
because an issue cannot carry both `## Revisions` and `## Log`:
- `/scheme`'s default report outside the raw editor says "detected only in a
  full-screen session", not "an interactive session" — the piped loop is a
  session and never detects.
- Only the full-channel LENGTH-CHECK drop site is guarded for a report. The
  `select`'s `default:` arm cannot see one (`readInput` is `out`'s only sender
  and the length check diverts every non-pointer key first); a guard there
  would be untestable.
- "`projectDictionaryText` stays (live)" was wrong: once `styleLanguageText`
  goes, a #66 source-provenance chain (`Entry.source`, `sourceAt`/`sourceKnown`,
  `RenderOpts.Language`, `bilingualDocument.native`, `projectDictionaryText`)
  loses its last reader. Recorded as residue and raised at the M1 boundary
  rather than deleted here — separable, and it touches the parser.
- Deletion runs BEFORE the role change; the flags move into the role commit
  (every commit green); docs describe only what each milestone ships.

`sdlc change-code`: plan-quality cleared in 2 rounds (PQ-1 Important — a
symlinked config dir unlinked by `ClearScheme` — fixed and probed in a scratch
worktree; PQ-4 "plan restates code" carried to the close review as a Minor).
Estimate 6.1h (v3.1, Method A). The estimate-quality judge passed it (INFO) but
forecast ~9–11h measured: `sdlc actual` already read 4.01h of DESIGN before any
code, against 1.85h the table priced, and several build rows hold more than one
commit. Kept as derived rather than re-fitted after the gate; when the ledger
row lands, read the gap as the design half first. Branch created in place.

M1 Task 3 — the dead tint paths. EVIDENCE (simulated deletion: the tint
painter's first statement made `return t.text`, run through `go test` only):
exactly five default-suite failures — `TestDictionaryCapturedMixedOwnership`,
`TestDictionarySourceProvenanceCorpus`, `TestDictionaryMonolingualOriginAndDisabledTint`,
and the painter's own two unit tests — plus the tagged `TestBilingualNativeLanguageOwnership`
(ran; failed under simulation, passed on the baseline). The two pty-backed
tests passed under simulation outside the sandbox. Nothing live. Then:
deleted the fragment-tint painter, its two helpers, `Render`'s four calls into
it (each replaced by the rendered text it returned under a zero tint) and the
`headAt` bookkeeping that fed only the first; moved the five test-only string
renderers into `render_helpers_test.go` unchanged; deleted the one with no
callers. Tests: the painter's unit tests go, but their LIVE halves were ported —
the producer-background cases into `TestSourceBackgroundTracksTheProducersBackground`
(nothing tested `sourceBackground` directly) and copy/highlight over a tinted
row into `TestSelectionCopiesATintedRowsText`, both on the live painter. Two
dictionary tests deleted: one held only fragment-tint assertions, and the
inline-pronunciation one asserted only an ABSENCE that can no longer occur (a
test that cannot fail). Three kept their live halves (provenance corpus,
no-colour, projection fallback); the tagged Oxford test keeps its provenance
check. Section tint stays pinned by `TestDefinitionOutputUniformSections`.

RESIDUE (raised at the M1 boundary, not deleted here): #66's source-provenance
chain now has no production reader — `RenderOpts.Language`, `Entry.source`,
`sourceAt`/`sourceKnown`, `definitionSection.source`, `bilingualDocument.native`,
`projectDictionaryText`, `bilingualLanguageText`. Tests that will then check
only the residue, to go with it: `TestDictionaryParserSourceOffsets`,
`TestDictionarySourceProvenanceCorpus`, the kept half of
`TestDictionaryProjectionExactOccurrenceAndFallback`, and the provenance check
left in `TestBilingualNativeLanguageOwnership`.

M1 Task 4 — the tint is a role. `rowPaint.background` (an escape string) became
`rowPaint.tinted` (a bit); the shade resolves in `paintLanguageRow` from the
scheme, read ONCE per frame from the holder the screen was given
(`attachScheme`, in `newConsole` and `sittingInPlace`). `-scheme` and
`-language-tint on|off` landed in the same commit. The test migration (136
compile errors, ~20 files + tagged pty/layout tests) was delegated to a
subagent under the plan's rules and reviewed. Two cases fed a foreign
background STRING to check it was dropped; the new type makes that input
unrepresentable, so those two list items are gone rather than rewritten.
`TestOutputScreenImmutablePolicyAndCurrentWidthTranscript` asserted each row
kept its own SHADE — the opposite of #70 — and now asserts each row keeps its
own tint BIT and that a producer mutating its slice after writing does not
reach the screen. pty conformance (`TestPTYLanguageTint`, `TestPTYNativeRendirSectionLayout`)
ran against the built binary with `-scheme dark|light` and `-language-tint off`: green.

M1 boundary. Operator decision on the residue: file a follow-up — #76
("delete or re-use #66's orphaned source-provenance chain"), carrying the member
list and the four tests that go with it. Pre-existing failures found while
verifying, all reproduced on the branch point 75370a2 (so not #70's): 13 pty
conformance tests fail in this environment (`TestPTYSuggestionAndAcceptance`,
`TestPTYCommandMenuAppearsAndClears`, `TestPTYTranscriptIsPrintedOnExit`,
`TestPTYWithoutMouseBehavesAsBefore`, `TestPTYMouseTrackingIsAskedForAndGivenBack`,
`TestPTYCtrlCMidAnswerKeepsTheSession` and seven `TestPTYPlay*`), and
`TestPlayClickOnThePromptWordPlaysIt` fails under `-race` only (3/3 on the base:
its 5 s wait is too short for the race detector). `TestLongPassageStreamsAgainstLiveService`
asks a live model and failed on answer shape, not code.

M1 review (SHIP, 4 Minor) — all four fixed in the close commit: the choice got
its own source type (a choice can no longer claim to be detected or default);
the board-footer tests go through the production path and the test-only copy is
deleted; the atlas paragraph now says what M1 does NOT ship yet. And the spec
reconciliation it asked for: the Spec's "dead code is deleted … tint assertions
PORTED" and the matching Done-when bullet describe more than happened. The
test-only renderers were MOVED to `render_helpers_test.go` as test helpers (not
deleted — tests still use them), and the per-fragment tint assertions were
DROPPED, not ported: they described behaviour production never had (it tints
whole sections, pinned by `TestDefinitionOutputUniformSections`); their live
halves (producer backgrounds, copy over tint) were ported.

M2 — `/scheme` and the saved choice. What the ticked plan steps rest on:
- **Symlinked FILE (Task 7 Step 6):** `WriteScheme`'s atomic rename REPLACES a
  symlinked `scheme` file with a regular file, so a dotfile manager that links
  the file itself (not its directory) loses the link on `/scheme light`. That is
  the store's behaviour for every setting (`bilingual.txt` too), not changed in
  #70; `ClearScheme` removing the file is intended (it IS the saved choice). A
  symlinked config DIRECTORY is kept — `TestClearSchemeRemovesOnlyWhatIsOurs`.
- **pty (Task 11 Step 4), precisely:** at the step I ran only the three affected
  tests on the built binary — `TestPTYLanguageTint` (default/dark/light/off),
  `TestPTYNativeRendirSectionLayout` (6 subtests), `TestPTYSavedSchemeSurvivesARestart`
  — all passed. The FULL tagged suite ran at the boundary: 1338 passed, 1 skipped
  (`TestPlanNamedTestsExist`), 14 failed, all 14 the pre-existing set logged at
  M1 and reproduced on the branch point 75370a2. `-race`: only the pre-existing
  `TestPlayClickOnThePromptWordPlaysIt`.
- **Mutations**, each applied, compiled, run with `-count=1`, red, restored:
  Task 7 — the size cap, the empty-dir removal, the `Lstat` guard, the path in
  the error; Task 8 — flag and saved swapped, the garbled-file warning dropped,
  `realDeps` without `configDir`; Task 9 — `choose` before `save`, the one-shot
  refusal removed; Task 10 — the editor's `session`, the editor's `fullScreen`,
  the piped loop's `session`, `applyScheme` skipping `choose`; Task 11 — the
  harness's `XDG_CONFIG_HOME` dropped under a fake config holding `light`.

M2 review: FIX-THEN-SHIP; the ledger blocked the close on BR-6 (this entry was
missing). Also fixed, the family rule rather than the instances (the
state-shape family's 2nd finding): `schemeArg` is ONE field (empty = auto — the
zero value forgets instead of saving a blank line); `commandCtx`'s `session` +
`fullScreen` became one `loopKind` {one-shot, piped, editor}; and the startup
read goes through the same `schemePersister` seam as save and clear, as a pure
`initialSchemeState` pinned by `TestInitialSchemeState` with no pty.

M2 closed on review round 2 (FIX-THEN-SHIP, BR-6 and BR-8 disposed). Its two
advisories fixed in the close commit: BR-7 — the state-shape family's third
instance, `schemeState.detected` + `heard`, is now one field (empty = nothing
heard); the #70 structs are enumerated (`schemeState`, `schemeChoice`,
`schemeArg`, `commandCtx`, `tintPolicy`) and this was the last pair. And the
atlas's `loopKind` sentence carries "(from M3)" again — a sweep of M2's doc diff
for detect/ask/report/query/OSC/KeyBackground finds nothing else untagged.

2026-09-18 — M3 manual check, first result, and a design change. The operator's
screenshot: light tint (254) with the terminal's DEFAULT text colour white — body
text and the dimmed syllables almost invisible, coloured text fine. Cause: a saved
`light` (their own `/scheme light` at 07:25, not yet `/scheme auto`'d) over a
terminal whose default foreground is white. Root cause, not the setting: the tint
is a FIXED background drawn under the terminal's DEFAULT foreground, which is
chosen for the terminal's background, not ours — so any scheme/terminal mismatch
is unreadable. Operator decision: pair it like the mark. Spec revision (approach A
said foregrounds belong to the terminal's theme): text with NO colour of its own on
a tinted row now takes `schemeInk` — 235 on the light tint, 252 on the dark;
producer colours (headword, IPA, examples, deck words) keep theirs.
`sourceColours` replaces `sourceBackground`'s parse with one that tracks both, so
the ink steps aside for a producer's foreground as the tint does for its
background. Pinned by `TestATintedRowCarriesItsOwnTextColour` and
`TestSourceColoursTracksTheProducersForeground`.

2026-09-18 — M3 manual live check: the operator verified the rebuilt binary
"working" (after the paired-ink change; their earlier light-profile screenshot is
what found it). Per-terminal reply strings were not reported, and the atlas says
so rather than inventing a matrix; it names the re-check triggers.

Done-when, each with its evidence:
- Light terminal → 254, dark → 236, no flag, no saved file: in process via
  `TestRawEditorBackgroundReplyRepaints` (reply bytes through `readInput` into
  `runEditor`); pty `TestPTYBackgroundDetection` light/dark/silent; operator check.
- A late reply repaints; a choice in force outranks it: `TestRawEditorBackgroundReplyRepaints`
  (flag-choice case), `TestPTYBackgroundDetection/late`, `TestSchemeStateSequences`.
- `/scheme light` repaints the screen AND the exit transcript: `TestRawEditorSchemeRepaintsWhatIsOnScreen`;
  piped: `TestPipedSchemeSaves`, `TestPipedSchemeWithNowhereToSave`; sitting:
  `TestASittingIgnoresABackgroundReply`, `TestAReplyDuringPlayReachesTheEditor`.
- Persistence: `TestPTYSavedSchemeSurvivesARestart`; session-only / one-shot refusal /
  write error / garbled file: `TestRawEditorSchemeWithNowhereToSave`, `TestOneShotScheme`,
  `TestRawEditorSchemeWriteErrorChangesNothing`, `TestInitialSchemeState`, `TestSavedSchemeGovernsALookup`.
- A dropped reply is silent and keeps the next notice: `TestADroppedReplyIsSilent`.
- No leak for rgb, rgba or #hex; mid-drag; Alt-] then typing / Ctrl-C in separate
  writes: `TestDecodeBackgroundReply`, `TestDecodeBackgroundReplyOtherFormats`,
  `TestDecodeOSCAbortsAsToday`, `TestReadInputBackgroundAcrossWrites`,
  `TestAReplyMidDragKeepsTheSelection`, and through both loops above.
- A /play reply reaches the editor; a sitting starts in the editor's scheme:
  `TestAReplyDuringPlayReachesTheEditor`, `TestASittingPaintsInTheEditorsScheme`.
- No query under `-language-tint off`, `-raw`, `TERM=dumb`; the tint-flag refusal;
  `-scheme light` then `/scheme dark` → `dark (saved)`; transition sequences:
  `TestPTYNoQueryWithoutATint`, `TestWantsBackground`, `TestLanguageTintInvalidFlagBeforeStore`,
  `TestOneShotScheme`, `TestSchemeStateSequences`, `TestSchemeHolder`.
- Dead paths deleted, live halves ported (the spec's "ported" reconciled in the
  M1 review entry above): M1 Task 3 entry.
- Every new test seen failing with its fix removed: the mutation lists in the M1,
  M2 and M3 entries (M3: colour parse ×2, decoder ×3, query ×2, consumers ×4,
  caller wiring ×3, paired ink ×3).
- README, `-h`, atlas updated; doc-sync tests green.
