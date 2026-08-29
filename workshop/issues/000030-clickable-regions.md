---
id: 000030
status: open
deps: [tools#29]
github_issue:
created: 2026-08-29
updated: 2026-08-29
estimate_hours:
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

- **`French` after `ORIGIN`** → play the recording in that language. `#29` builds
  the mechanism (`-pron fr` / `/pron fr`); this is the gesture that reaches it
  without typing a language code.
- **the IPA notation** → replay the recording. Today playback is automatic on
  every lookup; a click target is what would let it stop being automatic.

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

### The hard part is scrollback, and it should be decided rather than assumed

Coordinates go stale as soon as the next word is typed and the entry scrolls up.
The raw loop knows how many lines it printed, so tracking a delta is *possible*,
but a resize, a wrap, or scrollback the user scrolled by hand all break it. The
plausible answers — regions valid only for the most recent entry; an alternate
screen buffer; re-rendering on click — are different sizes and different
products. This is the design question, not the mouse decoding.

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

## Plan

- [ ] Blocked on `#29` for the pronunciation-language mechanism. Wire the
      fr/it/de dictionaries into `curated` first — the notation row above cannot
      be measured honestly until then.
- [ ] Design via `sdlc start-plan`.

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
