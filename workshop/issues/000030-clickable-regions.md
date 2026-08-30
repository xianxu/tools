---
id: 000030
status: working
deps: [tools#29, tools#35]
github_issue:
created: 2026-08-29
updated: 2026-08-29
estimate_hours: 3.19
started: 2026-08-29T16:24:35-07:00
---

# clickable regions in the terminal: click ORIGIN French to hear it, click the IPA to replay

## Problem

Operator, filing this while `#29` was being designed:

> ideally, we would highlight `ORIGIN French`, maybe just `French` that follows
> immediate the ORIGIN tag, and allow user to mouse click on it. make this
> "mouse click" highlight uniform across, as there might be others, for example,
> instead of auto play sound, we can make clickable highlight `/pəˈtasēəm/`, and
> if user click on it, the pronunciation is read out.

The shape is ONE affordance with several consumers, not two features. A rendered
entry contains tokens that *mean something the tool can act on*; today they are
inert text and the action has to be retyped as a command.

Two consumers are already named:

- **the HEADWORD** → play the recording. The operator's clarification, and the
  right primary target: every entry has one, in every language, whereas the IPA
  is English-only (`#31` measured it — Spanish writes none, Italian writes
  syllabification rather than transcription). `Entry.Headword()` and
  `Entry.Syllables()` already expose both forms as separate head tokens.
- **`French` after `ORIGIN`** → play the recording in that language. `#29` built
  the mechanism and `#35` makes it inferrable, so this click is a thin wrapper
  over `/pron` rather than the thing that introduces inference.
- **the IPA notation** → a second target where it exists, not the mechanism.

## Spec

Not designed. What follows is measured, so the design starts from facts.

### Clicking dissolves the ambiguity that killed ORIGIN inference

`#29` chose a DECLARED language (`-pron fr`) over one inferred from the ORIGIN
section. One of the two arguments against inference was that an entry can name
more than one language and something has to pick:

```
piano   ORIGIN either from French, or an abbreviation of pianoforte;
        Italian piano is not attested until later in this sense.
ballet  ORIGIN early 17th century: from French, from Italian balletto…
opera   ORIGIN mid 17th century: from Italian, from Latin…
```

**A click has nothing to pick.** `French` and `Italian` are two separate targets
and the user points at the one they meant. So this issue may revisit the
language-name → code table (`French`→`fr`, `Mexican Spanish`→`es`,
`Japanese`→`ja`) with that objection removed — but the OTHER argument still
stands and is not dissolved by clicking: such a table is a closed list of a fact
something else owns, which is what `ParseLang` and `-locale` both refused
(`atlas/define.md`, "Nothing whitelists which pairs exist").

The second `#29` argument — that origin audio must not be AUTOMATIC, measured
from `police_fr_fr`/`restaurant_fr_fr`/`machine_fr_fr` all being 200 — is
untouched here: a click is opt-in by construction.

### The IPA is not a uniform target, measured 2026-08-29

| dictionary | notation | evidence |
|---|---|---|
| English NOAD | **always** | `potassium \| pəˈtasēəm \|` |
| Spanish Larousse `es>es` | **never** | `madrugar`, `casa`, `cazar` carry no `\| … \|`; `TestNonEnglishEntriesCarryNoPronunciationNotation` asserts it over the whole corpus |
| French / Italian / German | **unknown** | unmeasurable today — see below |

Spanish has no notation because its orthography is phonemic: the dictionary has
nothing to write. So "the IPA is the click target for playback" is an
English-shaped rule.

**The operator's answer, and it is the better one: click the WORD.**

> if spanish doesn't have notation, we can just use the click on the word
> itself, e.g. `potassium`, or `po·tas·si·um`

That generalises where "click the IPA" does not. Every entry has a headword and
`Entry.Headword()` already exposes it; `Entry.Syllables()` exposes the
syllabified form (`po·tas·si·um`) as a separate head token, in SOURCE ORDER, so
either is addressable today without touching the parser. The IPA then stops
being the mechanism and becomes at most a second target in English, where it
happens to exist.

So the uniform thing is "a region that offers an action", and the FIRST region is
the headword — present in every entry, in every language. Which of headword or
syllabification carries the click (or both) is a design question, not a blocker.

French/Italian/German are unmeasurable because `fr.Multi`, `it.Devoto-Oli` and
`de.DDDSI` are installed but absent from `curated` in `cmd/define/dictselect.go`,
so `-lang fr bonjour` silently falls back to NOAD and returns the ENGLISH entry
(`bonjour | bänˈZHo͝or |` is NOAD's anglicisation, not French). Wiring those three
in is the one-line-each win `#29` split out and nobody has taken yet; it is a
PRECONDITION for measuring this row honestly.

### What does not exist yet

`cmd/define` has **no mouse code at all** (grepped 2026-08-29: no `1006`, no
`1000h`, no `Mouse`). The pieces this needs:

- **SGR 1006 mouse tracking** — `ESC[?1006h` on entry, off on restore, and
  decoding `ESC[<b;x;yM`/`m`. `key.go` already scans to a CSI final byte (`#14`),
  so the decoder has somewhere to live, but the enable/disable has to be part of
  `rawSession` or a Ctrl-C leaves the terminal in mouse mode.
- **Only the raw editor loop can own it.** The piped loop and the one-shot have
  no terminal; `--play`'s session borrows and returns raw mode around playback
  (`play_loop.go`), so the enable/disable has to survive that borrow.
- **`Render` must emit a region map.** It returns a `string` today. Regions mean
  spans → row/col, which changes its contract. It is pure, so this half is
  unit-testable without a terminal — keep it that way.

### The scrollback question is DECIDED: `define` becomes a TUI and owns the screen

Operator, 2026-08-29, after the alternatives were laid out:

> I don't mean OSC 8 style link. rather define become TUI program and owns all
> the rendering, so you know what is rendered precisely?

Yes — and it removes the problem by construction rather than managing it. The
interactive loop enters the alternate screen, keeps its own buffer of rendered
lines plus a viewport offset, and draws everything. A mouse click at viewport row
R maps to buffer line `R + scrollOffset`, and that mapping is exact because
`define` caused every line and every scroll.

**Three alternatives were considered and rejected, so the next reader does not
re-open them:**

- **Colour as the carrier.** The operator's first proposal, and the instinct
  behind it is right — the palette in `newPalette` is already markup by MEANING
  (`head`, `ipa`, `pos`, `num`, `ex`, `sect`), not colours that acquired
  meanings. It fails on one fact: **there is no escape sequence for "report the
  attributes at row R, column C".** Mouse reporting sends coordinates and a
  button. The terminal remembers the colour and can never be asked about it, so
  colour is markup a HUMAN reads, not markup the app can query.
- **OSC 8 hyperlinks.** The correct implementation of "markup the terminal
  carries", and genuinely scroll-agnostic. Rejected on cost: Terminal.app does
  not support OSC 8 at all, and a click opens a URL through the OS, so reaching
  the RUNNING `define` needs a custom scheme plus a helper that talks back into
  the session.
- **Most-recent-entry only.** Cheapest, and honest, but it makes the affordance
  disappear the moment you look anything else up.

### What a TUI actually costs, and what already exists

**Already in the tree**, which is why this is an extension rather than a rewrite:
raw mode (`enterRaw`/`restore`), key decoding that scans to the CSI final byte
(`#14`) — so `ESC[<b;x;yM` mouse sequences already parse structurally and are
merely discarded — and a width probe.

**Missing:** the alternate screen, a line buffer + viewport + scrolling, SIGWINCH
(there is none today; width is read ONCE at flag parse), mouse decoding, and the
region map.

**`Render` does not change**, and that is the load-bearing good news. It returns
a string; the screen layer splits it into lines and owns placement. The parser,
the renderer, the no-data-loss invariant and the whole fixture corpus are
untouched. This is a layer BENEATH them.

**Three costs to decide, not discover:**

1. **Terminal scrollback after exit.** The alternate screen tears down on quit,
   so the session's entries vanish from the terminal's history — today
   `define arrondissement` leaves the entry where you can scroll back to it
   tomorrow. The standard mitigation is to print the transcript into the normal
   buffer on exit. Decide it.
2. **Copy/paste.** With mouse tracking on, drag-select goes to the app. Either
   the user holds Option, or `define` implements selection, which is a real chunk
   of work.
3. **ONLY the interactive loop becomes a TUI.** `define <word>`, `echo w |
   define`, `-raw` and `> out.txt` stay exactly as they are. That also means the
   "ephemeral UI vs record" doctrine keeps applying on those paths while becoming
   vacuous inside the alternate screen, where everything is ephemeral. State the
   split; do not discard the doctrine.

**One fact to MEASURE before sizing the scroll machinery:** with mouse tracking
enabled, most terminals send wheel events to the application instead of scrolling
the viewport. If that holds on the operator's terminal, `define` causes every
scroll and the viewport model is bookkeeping. If it does not, the app must track
an offset it never observes, which is the hard version. Measure it; do not reason
about it.

### Cost that must be accepted deliberately

Enabling mouse reporting **takes drag-select away from the terminal** in most
emulators: copying text then needs Option (iTerm2/Terminal.app) or Shift. That is
a real regression for anyone who copies definitions, and it applies to the whole
session, not just the clickable tokens. A decision, not a footnote.

**OSC 8 hyperlinks are NOT an alternative.** They open a URL; they cannot call
back into a running `define`. Terminal.app does not support them at all.

## Done when

- [ ] Clicking the language after `ORIGIN` plays the recording in that language,
      through `#29`'s mechanism rather than a second one.
- [ ] The affordance is ONE mechanism with a registry of regions, so a third
      consumer is a row rather than a new feature.
- [ ] A clicked region is discoverable before it is clicked — a reader who never
      moves the mouse must be able to tell the token is live.
- [ ] The scrollback answer is a DECISION with its reason recorded, not an
      assumption that coordinates stay valid.
- [ ] Losing drag-select is decided deliberately and said out loud, with the
      escape (Option/Shift) documented where a user will see it.
- [ ] A terminal without mouse support degrades to exactly today's behaviour.
- [ ] The playback target is the HEADWORD, so it exists in every entry and every
      language — Spanish, which has no IPA, is not left with a dead affordance.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.50 impl=0.08
item: smaller-go-module        design=0.02 impl=0.14
item: smaller-go-module        design=0.02 impl=0.12
item: greenfield-go-module     design=0.06 impl=0.24
item: smaller-go-module        design=0.01 impl=0.08
item: smaller-go-module        design=0.02 impl=0.10
item: smaller-go-module        design=0.01 impl=0.08
item: cross-cutting-refactor    design=0.03 impl=0.12
item: milestone-review         design=0.00 impl=0.16
item: milestone-review         design=0.00 impl=0.12
item: cross-cutting-refactor    design=0.04 impl=0.18
item: smaller-go-module        design=0.02 impl=0.12
item: smaller-go-module        design=0.01 impl=0.08
item: smaller-go-module        design=0.02 impl=0.10
item: smaller-go-module        design=0.02 impl=0.10
item: smaller-go-module        design=0.01 impl=0.08
item: milestone-review         design=0.00 impl=0.16
item: milestone-review         design=0.00 impl=0.12
item: milestone-review         design=0.00 impl=0.10
design-buffer: 0.15
total: 3.19
```

Derivation notes.

- **`issue-spec` design 0.50 is the measured window**, `16:00`–`16:38`: the
  scrollback analysis, rejecting colour-as-carrier and OSC 8 with reasons, the
  `file:line` sweep PQ-10 demanded, and three plan-gate rounds. Larger than
  `#35`'s 0.35 because the design question was open — the operator chose the TUI
  in conversation but the consequences (cooked/raw, stderr, `#32`'s seam,
  transcript-on-exit) were all decided here.

- **`M1.3` is the only `greenfield-go-module`**, and it is the risk the milestone
  split exists for: the editor's output path is rewritten and `cooked()` is
  deleted, which removes a mechanism `#14` established and `#29` had to defer a
  replay around. 0.24 impl is the top of the scaled band, not the middle.

- **Two `cross-cutting-refactor`s, and they are genuinely different in kind from
  the modules.** `M1.6` rewrites the atlas's raw-mode section, whose subject
  D4 deletes. `M2.1` changes `Render`'s signature — `editorloop_test.go:29`
  returns it and the suite calls it from dozens of sites, so the churn is
  mechanical but wide, which the plan now says out loud so the boundary reviewer
  is not surprised by the diff size.

- **Two `milestone-review` pairs plus a close row**, because there are two
  boundaries and each returns work. Every boundary review this session has: `#29`
  four rounds, `#31` four, `#35` one with four findings. Pricing remediation at
  zero is the one thing this session's history rules out, so each milestone
  carries 0.16 to run and 0.12 to remediate, and the close carries 0.10 for the
  manual pass on a real terminal — which is the only place clicking can be
  verified at all.

- **No `TUI screen + state machine` primitive**, despite this being a TUI. That
  primitive is for a screen with its own input state machine; `M1` adds a
  viewport and a paint to an editor loop that already owns keys, frames and raw
  mode. `M1.3` is priced as greenfield precisely because it is the part that is
  genuinely new.

- **Step 2.5, the library-availability check, which the first draft SKIPPED.**
  v2.1 makes it a required step and its own examples read *"Cross-platform
  terminal UIs → bubbletea/lipgloss/bubbles"* — and `M1`'s four missing pieces
  (alt screen, viewport/scroll, SIGWINCH, SGR-1006 decoding) are all primitives
  those libraries ship. Skipping it is what drove v2's worst outlier, so:

  **Checked, and hand-rolled deliberately.** Step 2.5 exempts "choosing to do it
  from scratch for control or footprint reasons", and both apply. This is not a
  greenfield terminal layer with gaps to fill — it is an EXISTING one with four
  pieces missing: raw mode (`rawterm.go`), a CSI scanner that survived `#14`'s
  length bug (`key.go`), a pure `Editor`/`RenderLine` pair, and a `puretest`
  package that mechanically enforces the pure/IO split. Adopting bubbletea means
  replacing all of that with its Model/Update/View, discarding four issues' worth
  of measured decisions and the purity guard with them, for a CLI that today has
  four direct dependencies. The design hours are NOT halved.

  Recorded because the step's value is the visibility, not the arithmetic — it
  moves ~0.03 either way.

- **Expect this one to run long rather than short**, and here is the number so
  the close can tell a calibration miss from a confirmed prediction. 3.19 is only
  **1.15×** `#29`'s estimate (2.78) and **below** `#29`'s measured actual (3.34),
  while carrying 13 tasks across two boundaries. At this session's own over-run
  rate the actual lands near **3.7–3.9h**. The estimate is NOT padded toward
  that — v3.1 is applied as written, or the ledger row means nothing.

## Plan

- [x] Blocked on `#29` for the pronunciation-language mechanism — shipped.
- [x] The notation row is measured: `#31` did it. English IPA always, Spanish
      none, Italian syllabification-not-transcription, French none, German real
      but lossy. Curating fr/de turned out NOT to be a precondition, and is
      `#34`.
- [ ] Blocked on `#35`, so the click is a wrapper over an existing gesture.
- [x] MEASURE the wheel-capture fact — NOT NEEDED, and the plan records why: the
      alternate screen has no scrollback, so there is no offset define does not
      own. The wheel survives as a UX question, not a correctness one.
- [x] Design via `sdlc start-plan`. Plan:
      `workshop/plans/000030-clickable-regions-plan.md` (M1 the screen layer,
      M2 the clicks; two boundaries, one publish).

## Log

### 2026-08-29

Filed from the operator's request during `#29`'s design. Measurements taken
before filing: the absence of any mouse code in `cmd/define`, the notation table
above, and the observation that `-lang fr` currently answers from NOAD because
the French dictionary is not curated — which is why that row reads "unknown"
rather than a number.

The insight worth keeping: clicking removes the AMBIGUITY objection to
ORIGIN-inference (`piano` names two languages; a pointer picks one) but not the
CLOSED-TABLE objection. `#29` recorded both; only one is dissolved here.

### 2026-08-29 — M1.3: the cooked/raw dance is gone

`cooked()` is deleted (D4) and the editor's output goes through the screen. The
loop's writers are unchanged; `runEditor` swapped one closure for another —
`cooked func(func()) error` became `paint func(prompt string, menu []string)` —
and stdout/stderr are the screen in production, plain buffers in tests.

Three things fell out that were not in the plan:

- **`liveScreen`**, because a buffer is invisible: a streamed answer arrives per
  token and the `♫ playing 3×` has to appear while playback blocks. A write
  repaints; `screen` stays pure. It also owns `Stop()`, so nothing paints after
  the terminal is handed back — a frame drawn then lands on the NORMAL screen.
- **The buffer honours `eraseLine`.** Otherwise the indicator survives into the
  exit transcript and the record claims playback that may not have happened.
- **`lostTerminal` is gone**, and with it "raw mode could not be re-entered" —
  there is no re-entry. Same for the three farewell newlines: leaving the
  alternate screen restores the shell's own last line.

Deleted along with the mechanism they pinned: the cooked-block instruments in
two tests and `assertCRLFTerminated`/`streamedAnswer`. Each was rewritten to its
successor property rather than dropped — detail in the plan's Revisions.

Verified: `go test ./...` green; `go test -tags conformance -run PTY` green (8/8,
including the menu and terminal-restore rows, through a real pty and the alt
screen). A pty smoke run rendered `arrondissement` as one frame with the
committed line above the entry and the prompt below it, alt screen entered once
and left once.

## Revisions

### 2026-08-29 — the scrollback question is answered, and the target changed

**Reason:** discussed with the operator after `#29` and `#31` shipped. Two
decisions and one correction.

- **DECIDED: `define` becomes a TUI for the interactive loop**, owning the screen
  so click coordinates are exact by construction. Colour-as-carrier and OSC 8
  are recorded above as considered-and-rejected with their reasons.
- **The primary click target is the HEADWORD, not the IPA.** `#31` measured why:
  the IPA is an English-only affordance. Spanish writes no notation at all
  (phonemic orthography) and Italian writes syllabification, which is not a
  transcription. A headword exists in every entry in every language.
- **Sequenced behind `#35`.** With `/pron` inferring, both click targets become
  thin wrappers over gestures that already exist, and the inference question is
  settled before the screen work starts rather than tangled into it.

The Spec's earlier claim that this issue was blocked on curating fr/de was
wrong — `#31` answered the notation question by measurement without curating
either, and split them to `#34`.
