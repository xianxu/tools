---
id: '000050'
status: done
started: 2026-09-10T09:41:40-07:00
created: 2026-09-10
updated: 2026-09-11
estimate_hours: 4.19
actual_hours: 5.64
---

# ask before making the current directory a deck

## Problem

`define` makes the current directory a deck by writing into it, and it never
asks. Six `os.MkdirAll` calls scattered through `store/yaml.go` each fire on
their own first write, so `words/`, `events/`, `usage/`, `facts/`, `items/` and
`audio/` appear silently — along with `lang.txt` and `user-model.<lang>.md`,
which carries inferred claims about the learner.

The directory IS the deck (that is the design), which makes running `define` in
the wrong shell a real and quiet accident: you get a stray deck somewhere you
never meant, and you find out later.
