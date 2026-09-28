---
id: '000020'
status: done
started: 2026-08-26T11:03:49-07:00
created: 2026-08-26
updated: 2026-08-26
estimate_hours: 5.24
actual_hours: 1.35
---

# typeahead beyond the first word: complete deck words anywhere in the line

## Problem

The grey inline suggestion only fires on the first word of a line.

`Suggestion` (editor.go:145) byte-prefix-matches candidates against the **whole
typed line**, and `completionsFor` (command.go:41) returns whole past lines. So
typing `syco` completes to `sycophantic` — a past line that starts with `syco` —
but typing `what's the difference to obseq` needs a past line starting with that
entire text. A sentence you have never typed before matches nothing, so the grey
tail silently stops existing the moment you type a space.

That is exactly backwards for the words this tool exists to teach: the long,
hard-to-spell ones (`obsequious`, `certiorari`, `defenestrate`) are hardest to
type precisely when you are asking a free-form question ABOUT them, which is the
one place completion currently never fires.
