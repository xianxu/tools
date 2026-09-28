---
id: 000073
status: open
created: 2026-09-17
updated: 2026-09-17
estimate_hours:
github_issue:
---

# define: record a failed authoring attempt and stop re-asking for it

## Problem

`--harvest` re-pays for every failure it has ever had, on every run, forever.

`runAuthoring` (`harvest.go:293`) rejects a word on four paths — the stem does not
contain the word (`harvest.go:362`), the entail judge refuses it
(`harvest.go:382`), every distractor is vetoed (`harvest.go:438`), or the write
fails. Each one does the same thing: print to `errOut` and `continue`. The word
ends the run **with no item**, so the skip at `harvest.go:342` (`len(existing) >
0`) does not fire on the next run, and authoring re-asks with a byte-identical
prompt. Nothing anywhere records that the attempt happened.

So a word that cannot be authored costs 2 model calls (author + entail) every
single run, indefinitely, and a word whose distractors all get vetoed costs up to
5. Across a deck that accumulates unauthorable words, this is the dominant cost
of `--harvest` and it is invisible: the only trace is a line on stderr, and the
background harvester passes `io.Discard` for both writers (`background.go:224`).

The operator's decision (2026-09-17): **fail closed.** A word whose authoring
failed should simply have no cloze question, and should not be asked again unless
a policy explicitly says to. `--harvest`'s spend should not be a function of how
many times it has been run.
