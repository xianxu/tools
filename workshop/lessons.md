
## Oracles (define #1, five review rounds)

- **A check must not consult the function it checks.** The raw-notation test
  asked `isPronunciation` whether its own output was right, so it could only
  detect false positives. It read 0% while 2.2% of entries showed raw pipes —
  and that false 0% was published in the atlas.
- **An honest oracle can still be a narrow one.** `strayStress` only saw notation
  carrying a stress mark; example-separator pipes carry none. Adding a second
  oracle that consults nothing (`IndexByte(out, '|')`) found what it missed.
- **A one-directional oracle only catches one direction.** The no-data-loss
  invariant was a subsequence check, so `%q`'s escape sequences — which *insert*
  characters — passed five rounds. Assert counts, not just containment.
- **A claim must not outrun the width it was measured at.** Three rewrites of the
  same atlas section: 0 at 2,749 entries (3.8%) was 27 at full width. Sweep
  everything when the full pass is cheap; here it was 38 seconds.
- **Pin a known limitation as a ratchet, not a comment.** A test that asserts the
  exact known-bad count fails on regression *and* forces the number down when
  fixed. A `Limits` note alone drifts.

## Test helpers that hide the bug (define #2)

- **Never couple two independent conditions in a rig.** `replRig` set the stdin
  and stdout terminal checks from one parameter, commented "the real-world
  pairing." Escapes then leaked into `define > out.txt` and no test could reach
  it — the bug lives precisely where the two disagree.
- **Capture streams the way the user sees them.** A defect where a diagnostic
  overwrites a redrawn prompt is invisible with stdout and stderr in separate
  buffers: both the fixed and broken versions emit identical bytes. It exists
  only in the interleaving, so tee both into one buffer — a terminal is one
  stream.
- **Three terminal questions, not one:** may I colour (stdout), may I erase
  (stdout), is a human typing (stdin). Conflating any two is a bug; this repo
  has `color`, `tty` and `stdinIsTerminal` for that reason.
- **A guard with no test is not a guard.** Both `ctx.Err() == nil` suppressions
  and the whole >64 KB branch were dead to the suite — deleting them was green —
  after being added specifically to fix a shipped regression.

## Ephemeral UI vs. a record (define #2, close round 3)

An indicator that can be **erased** is ephemeral UI and may be optimistic — if
the action fails it is taken back and never seen. One that **cannot** be erased
(piped output, `-no-color`) is a *record*, and a record has to be true. Moving
the "♫ playing 3×" line ahead of playback made `define <word> > out.txt` claim
three plays where there were zero. Announce optimistically only when you can
take it back.

Corollary: a flag named for one effect (`-no-color`) must not leave a related
one on. It suppresses cursor control too, or the name is a lie.

## Reverting a mutation (define #2, close round 4)

`git checkout <file>` to undo a deliberate test-mutation **also discards every
other uncommitted edit in that file** — it silently reverted a fix made minutes
earlier, and the follow-up "verification" read as passing because a `|| true`
masked grep's exit status. Copy the file aside and copy it back. And when a check
disagrees with a test, the test is the ground truth.

## Verify the deletion, don't assert it (define #2, close round 5)

A commit message claimed a false sentence had been removed from the atlas. The
string replacement never matched, nothing checked, and the sentence survived two
more review rounds. **After a scripted edit, grep for the thing you removed** —
`replace()` on text that has since been reflowed is a silent no-op, and a commit
message is not evidence.

## One predicate, not three fixes (define #2, rounds 2/4/5)

The same rule went missing three times: *interactive UI is written to stdout, so
stdout must be a terminal; it answers a human, so stdin must be one too.* It was
fixed for cursor control (round 2), then in prose (round 4), then for the prompt
(round 5) — three findings, three patches, one missing predicate. When a finding
looks familiar, state the rule and apply it everywhere instead of fixing the
instance in front of you.
