
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

3. **byte-identical round trip** — catches both, and destroys the history it
   protects: every record in a log a person or a sync tool ever reformatted is
   discarded. The strictest rule was the most dangerous one.

What works is termination plus completeness: a whole record ends with its
terminator and carries every field. Two corollaries learned the hard way — the
writer must REPAIR a missing terminator before appending, or one interrupted
write costs two events; and a splitter must preserve line endings
(`SplitAfter`, not `Split` plus re-adding `\n`) or it hands a terminator to the
fragment and erases the signal.

Prefer one parsing path to a fast-path-plus-fallback: the two-path version
double-counted whatever the failed parse had collected, and left the fallback
unreachable for any input that stayed syntactically valid.

## A fake at the seam cannot see bugs below it (define #4)

The plan said "driven through a counting **store**"; the implementation used a
counting **capturer**, injected at the seam. It counted `Capture` calls, so it
could not see the failure it existed for — two writers *below* the seam, each
called once. Restoring the double write left it green while the deck
double-counted.

Where you inject the double decides what the test can see. To catch "the wrong
component wrote", the fake has to sit **beneath** every component involved. And
when a plan names a specific seam for a test, substituting a different one is a
design change, not an implementation detail.

## Before ticking a box, delete the line and run the suite (define #4)

*A fix ships with a test whose failure you have observed by removing the fix.*
Applied selectively, three fixes shipped as documentation: the `-raw` capture
call, `openStore`'s opt-out branch, and a traversal guard that was **unreachable
in principle** — sitting behind a sanitiser, no test at that API could tell the
guard from the sanitiser, and deleting it left everything green.

That last one forces an honest choice rather than a patch: either restructure so
the guard is exercisable at its own level (extract a pure function taking the
already-sanitised value, and feed it hostile input), or drop the claim that it is
asserted. Defence in depth that cannot be tested is decoration.

**And verify the mutation applied.** One check here printed GREEN because the
text substitution silently missed, not because the test was blind. A mutation
that did not land is not a result.

## A family finding is closed when every instance it lists is closed (define #4)

Told "this is the 4th finding in family X; fix the rule, not the instance", I
fixed **3 of the 10 instances the findings enumerated — and all three were the
ones in the titles.** That is the same substitution the escalation exists to
prevent: patch the named thing, leave the class.

Two operational forms:

- **Reply instance-by-instance, not finding-by-finding.** Marking a family
  finding addressed asserts the *family* is closed.
- **Before ticking a Done-when, name the symbol your evidence exercises and grep
  that the production path reaches it.** One fix here added a guard and wired
  only *one* of its two call sites — the one that was not the subject of the
  finding — while the record claimed both.

## A mutation is a result only if it applied AND compiled (define #4)

Two false readings in one session: a substitution that silently missed printed
GREEN ("test is blind"), and one that broke the build printed RED ("test caught
it"). Neither was true. Confirm the mutation landed and the package still
compiles, *then* read the suite.

## The family rule applies to artifacts, not just code (define #4)

Told to close a family instance-by-instance, I closed **10 of 10 code instances
and 0 of 7 plan instances** — the same fix-what-the-diff-touches substitution,
one layer out. A finding whose instances live in a markdown file is not a lesser
finding; it is the one that misleads the *next* issue, because that is what the
next issue reads.

Two operational forms:

- **Sweep the file the fix touched, not the line the finding named.** Two stale
  comments were *created* by a round whose whole subject was stale prose — one
  asserted a security property the same commit retracted, and one was orphaned
  onto a struct by an insertion above it. `git diff --stat`, then read each file.
- **When instances live in a plan, the closing move is an AGENTS.md §1
  `## Revisions` entry, not a checkbox tick.** Ticking boxes records that work
  happened; it does not correct what the document claims.

## `go test` runs in the package directory (define #4)

A repo-wide guard test ran `git ls-files` and got paths relative to `cmd/define/`,
not the repo root — so `cmd/define/define` arrived as `define`, and a "files at
the repo root are fine" skip swallowed the very artifact the test existed to
catch. It printed `ok` and I nearly accepted it.

Resolve the root explicitly (`git rev-parse --show-toplevel`, then `git -C`), and
**verify a guard by planting what it hunts** — clean passes, planted fails,
removed passes. Two runs, not one.

## Test the property, not a proxy for it (define #4)

The same guard's first working version flagged "extensionless file in a source
directory" and immediately false-positived on a tracked symlink. The question was
never about filenames: it is *is this file an executable image*, which the magic
bytes answer exactly. A proxy that is cheap to write is expensive to keep.

## Read `git show --stat` before committing (define #4)

`Bin 0 -> 9616546 bytes` is visible at a glance in `git show --stat`, and it was
in the commit that added a 9.6 MB binary. Nothing read it. This is the cheapest
possible check for the whole class of "`git add -A` swept in something I did not
mean to send", and it costs one command before the commit rather than a history
rewrite after it.

Corollary, learned the expensive way: **removing the file in a follow-up commit
does not remove the cost.** The blob stays reachable, so every clone still pays
— measured here at 5.9 MB against main's 604 KB while `git status` was clean and
`git ls-files` reported the file zero times. The fix is to rewrite the commit
that *adds* it, while the branch is still unpushed.

## A finding is closed only when you have re-run the measurement that produced it

Across one close, ten open findings entered a round and zero were closed, twice
in a row — while both rounds *felt* productive, because each fixed the visible
half. The shape: a finding arrives with a measurement (a clone size, a grep
list, an enumerated set of sites), and I fixed the part that shows up in
`git status` or in the title, then wrote a commit message claiming the finding
was addressed.

Re-running the finding's own measurement takes one command and would have shown
the claim false *before* the commit asserted it. Do that, and quote the number
in the commit — "5.9M → 712K, blob absent from a fresh clone", not "removed".

## `-count=1` belongs in the mutation recipe

A mutation test's planted run printed `ok (cached)`, which reads exactly like a
blind test and is neither. The recipe is now three clauses: confirm the mutation
**applied**, confirm it **compiled**, and run with **`-count=1`** so the result
is not Go's cache answering a question about the previous source.

## Say what a test reads before naming it as enforcement

A `.gitignore` comment named `TestNoCommittedBinaries` as enforcing "the general
case". The test reads `git ls-files` — the index — while the class's cost lives
in history, so it was green on a repo carrying exactly the artifact it existed
to prevent. Before citing a test as the guard for a class, state what the test
reads and check the class lives there. Scope mismatch passes every review that
only asks "is there a test?"

## A guard must assert it consumed the whole work list

A history-scanning guard enumerated every object in the repo, checked each one,
and reported a clean result — from a partial scan. Nothing compared records read
against records requested, and the subprocess's exit status was `defer`red and
dropped. Feeding it the first five objects left it green with a planted binary
sitting in history.

Two clauses, both cheap:

- If a test builds a work list, assert it reached the end of it. `seen ==
  len(want)`, and fail with the counts — `scanned 5 of 761 objects` names the
  problem instantly.
- Check the exit status of every process the test depends on. `defer cmd.Wait()`
  discards it; so does `_ = cmd.Wait()`. A crashed child and a clean one are
  otherwise indistinguishable, and the crash is the interesting case.

This one is worth internalising because of where it was found: in the fix for the
*previous* round's version of the same rule. Writing the rule down does not
execute it.

## Name the suite a swept file actually runs in

A sweep claimed three instances of a class fixed, and reported the suite green in
the same breath. The third instance lived behind `//go:build darwin && conformance`
and skipped without a built binary — so it never ran in the suite being reported.
The fix was correct; the claim about it was not. When a sweep touches a file with
a build tag, an env guard, or a skip, say which suite it runs in and run that one.

## Mutation-check a claimed behaviour against its OWN code path

A test asserted "the process exited cleanly" and was credited with pinning "Ctrl-C
arrives as a byte, not a signal". Both paths produce that observable, and so does
a crash — the assertion separated only the crash. An observable that two code
paths both produce cannot distinguish between them, however true the assertion is.
Mutate the specific branch the prose names and watch THAT test redden.

The same episode is a caution in the other direction: a reviewer measured the byte
path as "asserted by nothing" from a run scoped to one build tag, when the full
suite reddens on that mutation. Before accepting a negative finding, re-run its
measurement at full scope — a finding is a measurement, and measurements have
scopes.


## Render from the state you just changed, not the state you read

A key loop computed its candidate list, applied the keystroke, then drew the
suggestion using the list from *before* the keystroke. It was invisible for as
long as both the old and new lists came from the same source — a stale superset
usually shares its first element. It became a reported bug the moment the two
lists came from different **namespaces**: the grey tail offered one completion
and Tab accepted another.

The fix that removes the class rather than the instance: the drawing function
computes what it needs instead of accepting it. A parameter is a place a stale
value can enter; if a function can derive its input from current state, let it.

Corollary worth keeping: "it was already like that and nothing broke" is not
evidence of correctness. It is evidence that nothing has yet varied the thing the
latent bug depends on.
