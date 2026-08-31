
### 2026-08-30 — read this before resuming: #7 lands first and changes the prompt

Parked at `punt` after T0. While it is parked, `#7` (review form 2.3, meaning
multiple choice) is being built, and it touches this issue in two places — found
by `#7`'s plan-quality gate rather than at a merge conflict.

1. **`todaysQuestions` (`play_loop.go:233-265`) is edited by both.** `#7`'s T5
   builds the distractor pool there; this issue's T5 rewrites the same function.
   `#7` lands first, so re-read that function before resuming — do not apply T5
   from the plan as written.

2. **A second form arrives whose prompt is NOT just the word, and this issue's
   region arithmetic assumes it is.** `play/recall.go:29` makes the prompt the
   headword alone, so T4 draws the region at line 0, column 0, width
   `visibleCells(word)`. `Choice`'s prompt is multi-line: the word, a blank, then
   four numbered options.

   `#7` accepted a CONSTRAINT to keep this working — the target word stays alone
   on the first line — so the arithmetic still holds and T4 needs no change. It
   holds because `#7` chose to protect it, not because it is inherent, so if this
   issue ever generalises the region beyond line 0 it should stop depending on
   the constraint and read the form's own declaration instead.

   The upside: `Choice`'s options are additional clickable material this issue
   can mark later, with no new decision needed.
