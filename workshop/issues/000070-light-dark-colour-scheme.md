---
id: 000070
status: open
deps: []
github_issue:
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
---

# define: switch between a light and a dark colour scheme

## Problem

`newPalette(on bool)` (`cmd/define/render.go:46`) returns ONE hardcoded palette —
bright cyan headword, magenta IPA, italic green examples, SGR `2` for dim — and
those values were chosen for a terminal with a dark background. On a light
background `\x1b[2m` dim grey on white is close to unreadable and bright cyan is
worse. The only escape is `-no-color`, which is all-or-nothing: the reader's
choice today is "colours built for someone else's terminal" or "no colours".

The one light/dark axis that already exists is `-language-tint dark|light|off`
(`main.go:524`, `language_style.go:145`) — a background tint behind
target-language text, `48;5;236` vs `48;5;254`. That is the right pair of values
bound to the wrong scope: it themes one feature rather than the session, and its
default (`dark`) silently asserts a fact about the terminal that nothing else in
the program is allowed to know.

Two more surfaces are about to make this worse: the passage marks in #67, and the
markdown scheme in #71, which is specified as one palette for light and one for
dark and has nowhere to read the answer from.

## Spec

**One theme, whole-session.** A `theme` with two values, `dark` and `light`, that
selects every colour define emits: the entry palette, `highlight.go`'s `knownOn`,
the language tint, the playbar, #67's passage marks, #71's markdown scheme. One
source for the light/dark fact (ARCH-DRY) — today it is a per-feature flag, and
each new feature that hardcodes an SGR is another place the fact lives.

**Two surfaces, the split define already documents.** `-theme dark|light` applies
to this invocation and does not persist; `/theme` persists. That is verbatim the
`-lang` / `/lang` division main.go:533 writes out ("`-lang` exists so a script can
ask a question without mutating state … `/lang` is the other half"), and the
`-sound` / `/sound` pair is the second instance. `/theme` with no argument
reports the current one, like `/sound`.

**Persistence** in the deck's settings, the way `store/lang.go` and
`store/bilingual.go` already do it. Precedence: flag, then the stored setting,
then the default — the same three-step `openStore` applies to the language
(main.go:503).

**The default stays `dark`**, which is today's behaviour, so no existing terminal
changes appearance. Auto-detection is DEFERRED, deliberately, and this is the
operating envelope the design commits to (ARCH-CONSTRAINTS): `COLORFGBG` is set
by roughly one terminal family and lies after a theme change, and an OSC 11
query is a round trip that some terminals never answer, which means a timeout on
the startup path of a program whose whole job is to answer in under a second.
Both are worth having and neither is worth blocking a readable light theme on.
If detection later lands it becomes a third value, `auto`, resolving to one of
these two — the two-value enum is not disturbed by it.

**`-no-color` stays orthogonal.** It answers "may I colour at all"; the theme
answers "which colours". `lessons.md` already records the cost of conflating
terminal questions ("Three terminal questions, not one") — this is a fourth, and
it composes with the other three rather than joining them.

**`-language-tint` derives.** Once a theme exists, `dark`/`light` on that flag
are a second source of truth for the same fact. The flag keeps working (it is
documented, and `off` has no theme equivalent), but the tint's light/dark value
comes from the theme unless the flag was given explicitly, and the precedence is
written down and tested rather than left to whichever assignment runs last.

**Pure palette, thin shell** (ARCH-PURE). Choosing a palette is a pure function
of `(theme, colorOn)` returning a value; the IO shell reads flag + setting and
passes it in. No package-level palette variable — `options` already carries
`color` and `tintBackground` and is the seam.

Open questions for the brainstorm:
- Does `/theme` re-render what is already on screen, or take effect from the next
  entry? `screen.lines` is immutable with colour baked in at write time
  (`passage.go:17`), so re-rendering scrollback is not available; "from the next
  entry" is the honest answer and should be stated in `/theme`'s reply.
- Is the light palette a remap of the same seven roles, or do some roles collapse?

## Done when

- `-theme light` and `-theme dark` both produce a full render, and every colour
  define emits comes from the selected palette. Verified by an enumeration test
  over the palette's roles + a guard that greps `cmd/define` for SGR literals
  outside the palette's own file — a spot check here is exactly the "subset
  pretending to be the whole" `render.go:57` already warns about.
- `/theme light` changes the colours of the next entry in a live session, and
  `/theme` with no argument reports the current theme.
- The choice persists: a second `define` invocation in the same deck starts in
  the stored theme.
- Precedence is tested: `-theme` beats the stored setting, which beats the
  default.
- `-language-tint` derives from the theme when not given explicitly; giving both
  resolves by the documented rule, with a test for each combination.
- `-no-color` suppresses colour under either theme, and `define <word> | cat` is
  byte-identical under both themes.
- `atlas/define.md` records the theme axis and where the palette lives.

## Plan

- [ ] brainstorm the two open questions above
- [ ] `sdlc start-plan`, then the durable plan in `workshop/plans/`
- [ ] the palette becomes a value selected by `(theme, color)`; enumeration test
      over its roles
- [ ] sweep every hardcoded SGR under `cmd/define` into the palette; add the
      repo guard that keeps them there
- [ ] `-theme`, `/theme`, persistence, precedence
- [ ] `-language-tint` derives from the theme
- [ ] atlas, then `sdlc close`

## Log

### 2026-09-16

Filed alongside #71 (markdown rendering), which needs a light/dark axis to hang
its scheme on and is blocked on this one.
