
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

## Raw mode: render cooked, play raw (define #14)

In raw mode Ctrl-C is byte `0x03`, not a signal, so `signal.NotifyContext` never
fires and the **key reader** must own cancellation — it can act while the loop is
blocked. But that is only half of it: restoring cooked mode around a long
operation hands Ctrl-C back to the line discipline, which swallows the byte, and
the reader sees nothing. Printing needs cooked (newline translation); blocking
work must stay raw. Split the two.

Also: a pty master does not honour `SetReadDeadline`, so a foreground read loop
in a pty test hangs rather than times out. Use a background reader plus a
snapshot.

## Don't assume an escape sequence's length (define #14)

`ESC[3~` is Delete; `ESC[3;5~` is Ctrl-Delete. Special-casing the four-byte form
consumed four bytes of a six-byte sequence and inserted `5~` into the word being
typed. Scan to the CSI final byte (0x40–0x7E after parameter/intermediate bytes)
instead of matching a prefix and assuming a length.

## Two loops, one decision table (define #14)

A second input path was added and re-implemented "what does this line mean"
inline, so the interactive path stopped trimming and stopped collapsing
`hot  dog` into the multi-word headword. If two paths take user input, they route
through one parser or they will drift — the divergence is silent because each
path is individually tested.

## Deleting a test needs the same evidence as writing one

Six tests were removed with a note saying a design change had superseded them.
Five still passed verbatim: they exercised a path the change left intact. Before
deleting, **run them against the new code** — "this test is obsolete" is a claim,
and it is checkable in seconds.

## Say what a design buys, not what it feels like it buys (define #3)

"One file per word so a synced directory never conflicts" was false: the same
word, or the same day, on two machines still conflicts. What the layout actually
changes is the *rate* — with one big file, every write on the second machine
conflicts. The precise claim is still a good reason for the design; the loose one
would have been quoted back later as a guarantee the code never made.

## Detecting a truncated record needs a round trip (define #3)

Two plausible tests both fail, and each took a review round to disprove:

1. **"it parsed"** — a cut leaves valid YAML. `- word: thi` unmarshals into an
   event with no timestamp.
2. **"the fields are present"** — a cut *inside a timestamp* can leave a shorter
   date that parses fine, so every field is populated and a fabricated event is
   admitted.

Re-marshal what was parsed and compare it to the bytes on disk: a fragment cannot
reproduce itself. And prefer one parsing path to a fast-path-plus-fallback — the
two-path version double-counted whatever the failed parse had collected, and left
the fallback unreachable for any input that stayed syntactically valid.
