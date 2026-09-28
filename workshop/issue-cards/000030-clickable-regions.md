---
id: '000030'
status: done
started: 2026-08-29T16:24:35-07:00
created: 2026-08-29
updated: 2026-08-30
estimate_hours: 3.19
actual_hours: 10.91
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
