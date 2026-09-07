
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
masked grep's exit status. And when a check disagrees with a test, the test is
the ground truth.

This entry originally concluded "copy the file aside and copy it back". That is
the advice five later occurrences disproved; the rule now lives in one place,
*Mutation testing needs a COMMITTED baseline*.

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

**`#30` dissolved this in `define` rather than obeying it better.** Once the app
owns the screen it places every line itself, so no output depends on the line
discipline and raw mode is continuous — there is nothing left to flap. The lesson
stands for any program that drops a terminal mode around a blocking call; the
better move, where you can afford it, is to stop needing the other mode.

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

## `go test` runs in the package directory — third time

This one fact has now cost three review rounds in three disguises:

1. A repo guard resolved `git ls-files` relative to `cmd/define/`, so the artifact
   it hunted arrived as `define` and a "repo root" skip swallowed it. It could
   not fail.
2. `.gitignore` anchored `/words/` and `/events/` at the repo root, so a deck
   written by tests at `cmd/define/words/` matched nothing and was committed.
3. The pty conformance suite launched the binary with the test's cwd, so it wrote
   a deck into the source tree and rewrote it on every run.

Whenever a test touches the filesystem or the repo, state which directory it is
standing in. Resolve repo paths from `git rev-parse --show-toplevel`, give any
child process an explicit `cmd.Dir`, and prefer un-anchored ignore patterns for
runtime output — anchoring only covers the root, and tests do not run there.

## Two implementations of one idea diverge, and the wrong one carries the comment

`historyWindow` used `AddDate` and documented exactly why a Duration is wrong
across DST. `relativeDay`, twenty lines below, divided a Duration by 24h — and
carried a comment claiming it worked on calendar days. Every `/history` date was
off by one for the week after each spring-forward.

Having got a subtlety right once is not protection; it is the thing that makes
the second copy feel safe to write. When a second site needs the same idea, reach
for the same primitive, and give the second site the harder test — the first one
already has it.


## Pin a loop shell's wiring with a test that drives that loop shell

Four separate wirings shipped green with the wiring deleted — a `setTimes`
closure, a one-shot dispatch, an exit-code propagation, a history append. Every
one had tests. The tests built the callee's context by hand, or drove the *other*
loop.

The sharpest was `/sound`: its test drove the piped loop while the requirement
was the raw TUI prompt, so deleting the raw loop's wiring left the feature
silently broken exactly where it had been asked for.

**The rule:** a value or effect that only a loop shell supplies (`runEditor`,
`replLines`, `run`) must be pinned by a test that drives that loop shell. Every
hand-built context literal in a test is a place this breaks invisibly — the test
passes because it supplied the thing the production path forgot to.

And the part that cost an extra round: *fixing the four instances is not closing
the finding.* The same commit that fixed them introduced two more — an unpinned
arity exemption and an unpinned `clearMenu` — because the rule had been applied
to a list rather than adopted. Write the rule down where it will be read again,
then check the diff you are about to commit against it.

## A marker must not match the thing it is meant to distinguish from

Twice in one issue, a test found its marker in the wrong place: `inputOn+"/sound"`
matched the ECHO of the submitted line rather than the recall, and
`"list the commands"` matched the MENU row rather than `/help`'s output. Both
passed with the production wiring deleted.

Before using a string to locate "where the output starts", check it does not also
appear in what comes before. Prefer a marker only the code under test can emit —
or drive a case whose output shares no text with its surroundings.

## An exemption drops invariants you weren't thinking about

To stop a command paying for a log it never read, I exempted it from the shared
setup call. That call also supplied the process clock — so the exemption left it
nil, and any future command that read the clock would have panicked. It also
failed at its own job: the thing being avoided (an eager log read) still happened
for the commands that *weren't* exempted, so `/history` read the log twice.

**The rule:** when a new kind is exempted from a shared setup path, enumerate
everything that path guaranteed and re-supply it. An exemption removes more than
the cost you were trying to avoid.

**The better move, when it is available:** remove the cost at its source instead
of routing around it. Making the log read lazy meant nothing needed exempting,
which fixed three findings at once and deleted the branch that caused the
strand.

## Removing a cost can make an old test vacuous

Reading the log moved out of a constructor, and a test from a previous issue —
one that asserted usage errors do not read the log — silently stopped being able
to fail: the ordering it pinned no longer decided the outcome. It still passed,
and it still looked like a guard.

When you change *when* something happens, re-run the mutations of every test that
existed to pin *that timing*. And where a test asserts an absence ("this did not
happen"), pair it with a control that makes the same thing happen — otherwise a
dead fixture and a working guard are indistinguishable.

## A width is a rune count, and two renderers must agree what it means

One renderer truncated in bytes, the other in runes, and a third place measured
column widths in bytes while `fmt` padded in runes. All of it was invisible
because every test passed width 0 — the truncation branch had never executed.
Mutation-checked afterwards, the byte version produced `"  na\xc3"`: half a
character.

Two rules fell out. **Share the width helper** rather than writing the check at
each call site, because two implementations of "cut this to N" will disagree
about what N counts. And **a branch no test exercises is not covered by the tests
that call the function** — if every call site passes the value that skips it, it
has never run.

## An assertion whose subject is empty cannot fail (define #11)

`TestStreamDoesNotForwardThinkingDeltas` compared what `onDelta` emitted against
the thinking block's text, guarded with `if thinking != ""`. The recorded capture's
`thinking_delta` carries `""` — the model returns an empty thinking body even under
`display: summarized`, only the signature is meaningful — so the guard never opened
and the test could not fail in any universe. Mutation confirmed it: forwarding
thinking deltas to `onDelta` as well left the suite GREEN.

The fix was to assert on an observable that actually *differs* between the two
implementations — the **call count** (5 text deltas, not 6) — and to derive the
expected numbers from the capture rather than a literal, so re-recording cannot
silently rot them.

The general rule: before asserting `A does not contain B`, check that B is
non-empty in the fixture. A containment check against an empty needle is
vacuously true, and it reads exactly like a passing guard.

## A fake grounded in the easy case models the wrong thing (define #11, three rounds)

Three findings in one family, all the same mistake: a probe recorded under
conditions that do not elicit the shape the fake is supposed to model.

1. A trivial prompt returned one `text` block — so the fake was single-block, and
   a client reading `content[0].text` would have returned `""` live while passing
   the whole suite. `claude-opus-5` returns a **thinking block first** on anything
   non-trivial.
2. `max_tokens: 512` with adaptive thinking on let thinking eat the budget. The
   answer came back `stop_reason: max_tokens` carrying `{"verdict":"yes",
   "reason":": Ā"}` — valid JSON, every required field present, cut mid-rune. It
   **decodes**, so a decode-first implementation returns success on garbage.
3. A trivial stream produced no `thinking_delta`/`signature_delta` frames, so a
   `Stream` that dropped thinking blocks would have passed everything.

Each artifact looked like evidence, was committed, and had something modelled on
it. The fix is not a better capture — it is making the required shape **checkable
at record time**: `scripts/llm-probe.sh verify` declares per capture what it must
exhibit, and `record` refuses to promote one that does not. It caught a real
failure on its first run (an `overloaded_error` envelope heading for `testdata/`),
which exposed a second bug in the check itself:

**A verification that runs after the destructive step protects nothing.** The
first version wrote into `testdata/` then verified, so the bad response had already
clobbered a good capture before the check failed. Stage, verify, then promote.

Corollary: with adaptive thinking on, `max_tokens` must cover the thinking AND the
answer. Budget for the answer alone and the answer is what gets cut.

## Two error classes that need opposite responses must not collapse (define #11)

A stream that dies mid-reply and a service that is unreachable are not the same
event: one means "skip this question", the other means "stop trying for a while".
Conflating them either parks a healthy run or abandons a recoverable one (the rule
is kbench's `is_outage`, learned there across four rewrites).

The discriminator that works is cheap: **once a single frame has arrived, the
service is demonstrably reachable**, so any later failure is a truncation. Return
the partial text with `ErrTruncated`, not `ErrUnavailable`.

Same shape as this repo's older `ErrNoAudio`/`ErrFetchFailed` split. When adding an
error, ask what a caller does differently on it — if the answer is "nothing", it
does not need to be a new class; if two existing classes lead to opposite actions,
they must not be one.

## Reversion-check the test you ADD, not only the fix it pins (define #11)

Three times in one boundary review I added a test that could not fail:

1. a containment check against a needle the fixture leaves empty;
2. a stall-default test bounded by a 3s context, which ends the call whether or
   not the default applied;
3. a "negative disables the bound" test with a 2s `Timeout`, which dominates
   either way.

Each read as a guard. Each was green with the thing it "pinned" deleted.

The rule that covers all three: **a bound dominated by a second bound
distinguishes nothing.** After adding a test, revert the fix and watch THAT test
redden — the same discipline already applied to production fixes, applied to the
test itself. And when the honest observable is a config value rather than a
behaviour, assert the config: an in-package check on the constructed value caught
the deletion of *any* default, where the behavioural version caught none.

## A fixture tuned to itself proves nothing (define #11)

Two instances, one round apart:

- The unknown-model fixture was `claude-not-a-real-model`, and the buggy rule it
  was meant to pin special-cased any name containing `not-a-real`. Reverting the
  fix left the suite green. **A fixture must not encode the pre-fix
  implementation's escape hatch** — use ordinary values (`claude-opus-6`).
- A multi-block splitter chopped byte offsets, cutting multibyte text mid-rune.
  Its test passed because the fixture was ASCII, while every real capture contains
  em-dashes. **Pick fixture data from the shape of real inputs, not from what is
  convenient to type.**

## "Captures, never literals" is about content, not framing (define #11)

A rule that a fake must serve only recorded responses — never invented ones —
prevented tests from asserting against a guess at what a model says. Correct, and
over-applied: it left four load-bearing behaviours untestable, because no
committed capture happened to exhibit the needed *shape*. All three captures carry
exactly one text block, so "join every text block" could not be distinguished from
"take the first" — a property asserted in three separate places and pinned by
nothing.

The line: **what the model SAID must come from a capture; how the response is
FRAMED — block count, header presence, stop_details — is transport shape, and
constructing it is legitimate.** Where the code singles a behaviour out as
load-bearing, the fixture must separate it from the alternative it warns against,
even if that fixture has to be built.

## A finding is a claim about the tree; sweep the tree, not the diff (define #11)

Two failures of the same shape in one review:

- I reported an edit as applied when the batch had aborted before writing it. The
  reviewer found the unchanged line by reading the file. **Re-grep after a scripted
  edit; a script that raised halfway is not a partial success.**
- A doc-claim sweep covered `internal/llm/*.go` and stopped, while the finding had
  explicitly named the plan file — and a later round found the same stale sentences
  still alive in the plan's *embedded code blocks*, which the plan itself warns
  "get pasted verbatim".

**Enumerate where the class can live before fixing any instance of it**: source,
tests, embedded code blocks, plan prose, atlas, and the script comments. Then
re-run the finding's own measurement, not a proxy for it.

## The sandbox reports "absent" where the truth is "unreachable" (define #18)

`define madrugar` returned `no dictionary entry` under the sandbox and a full
Oxford entry without it — same binary, same word, same machine.
`DCSCopyTextDefinition` cannot reach the system dictionary assets from inside the
sandbox, and the API has no way to say so: its status codes distinguish
*no entry* from *CoreFoundation failure*, and a denied read looks like the former.

I published a conclusion from that probe — "no Spanish dictionary is enabled on
this Mac" — **after the operator had already said the definition worked.** Their
observation was direct evidence; mine was a probe through a layer I had not
accounted for.

Two rules:

- **When a capability looks absent, check whether it is merely unreachable** before
  concluding it is missing. This is the third instance in this environment (git
  over SSH, pty allocation, now DictionaryServices), so treat "sandboxed probe says
  no" as unproven rather than negative.
- **A user's direct observation outranks your indirect probe.** When they conflict,
  the probe is what needs explaining.

## Fuzz both branches, or half the invariant is unfuzzed (define #11 M2)

`FuzzDecode` asserted the error branch and `return`ed early on success. So the
half that said *"a fully populated T"* was never checked, and `decode` shipped
returning `{Fits:true, Reason:""}` with a **nil error** for `{"fits":true}` —
while three separate artifacts (the function's own invariant comment, the atlas,
the plan) claimed the opposite, and `SchemaFor[T]` already emitted
`"required":["fits","reason"]`.

The single source declared the constraint and the consumer ignored it. Two rules:

- **A fuzz target that returns early on the success path is testing one half.**
  Write the success assertion first — it is the one the happy path exercises
  constantly, so a bug there is both likelier and quieter.
- **`encoding/json` zero-fills.** "Decoded without error" never means "the fields
  are present". Where a schema declares `required`, derive the check from it;
  restating the list in a second place is the drift you were avoiding.

## A double placed above the seam quietly reverses the design (define #11 M2)

M1's central decision was that the test double is an httptest server speaking the
wire protocol, **not** a stubbed `Client` — with a package doc arguing that a
stubbed client cannot see a mis-serialized field or a dropped header. M2 then
added a cassette that replaced `llm.Client` outright. Same package, opposite
decision, no revision entry, and every consumer test written against it would have
lost exactly the coverage the doc promised.

Rebuilt as an `http.RoundTripper` under a `Config.Transport` seam, which cost
about forty lines and bought a property the higher placement could not have:
**the error taxonomy survives replay structurally.** A recorded 400 is replayed as
a 400 and reaches the classifier as "our bug"; the `Client`-level version stored
the error as a string, re-derived the class from the stop reason, and collapsed
every recorded `ErrRequest` into `ErrUnavailable` — the one collapse the taxonomy
exists to prevent.

**The rule:** when adding a second double for a dependency that already has one,
place it at the same seam. If it cannot go there, that is a design change and it
needs saying out loud, not a new file.

## A universal claim is a claim about an enumeration (define #11 M2)

Four doc claims in one milestone were false the moment they were written —
"every failure names the fix" (2 of 7 did), "held by a fuzz target" (one branch),
"reject missing ones" (it did not), and a citation of a golden file that did not
exist anywhere in the tree.

Each was written as a description of intent while the edit was fresh. **"Every",
"always", "never", "is held by" are claims about a set**, so they may only be
written after enumerating that set and checking each member — and a claim naming
an identifier or an on-disk artifact must be grep-verified in the same edit that
writes it. Otherwise the doc records what the author meant to build.

## Run the enumeration a finding hands you (define #11 M2)

A finding said the drift suite reports drift when the proxy is merely unreachable,
and named **two** conformance files. I fixed one. The next round re-raised it as
`not-addressed`, and the fix that finally worked was not a third patch but
`llmtest.SkipIfUnreachable` — one helper both suites call, so the third live suite
cannot forget it.

When a finding writes out an enumeration, the enumeration IS the work item. Run
it, and then run a grep that proves you ran it — mine surfaced a third candidate,
which turned out to be fake-backed and correctly unguarded. Dismissing a candidate
with a reason is part of the sweep; not looking is not.

## Restoring one property can sever another (define #11 M2)

Moving the cassette beneath the transport seam fixed a real architectural problem
— and silently broke the coupling that goldens and cassettes derive from one
renderer, because an `http.RoundTripper` cannot see a field (`Task`) that is never
sent. The key fell back to hashing the wire body, so two requests differing only
in task collided and the second recording overwrote the first, while five
artifacts and two tests went on asserting the coupling. `RequestHash` had zero
non-test callers and a doc comment saying "RequestHash keys a cassette".

The fix was to carry the request down by context so one key definition survives.
The lesson is the check: **after a structural move, re-verify the invariants the
old structure was holding** — list what the moved thing used to guarantee, and
confirm each still holds. A test that pins a now-unused function looks exactly
like a test that pins a used one.

## A fix without a test is a comment (define #11 M2)

I added a defensive `clone` to a memoised value, explained why in six lines, and
wrote no test. A mutation that removed it passed. Worse, my first mutation
attempt removed the clone from only *one* of two return paths and still passed —
so I nearly recorded "verified" on the strength of a mutation that did not
reproduce the bug.

Two clauses: **a fix ships with a test whose failure you have observed**, and
**the mutation must be the honest absence of the fix**, not a partial one. If the
fix has two sites, remove both.

## Write the check against the SHAPE, not the example (define #11 M2)

A finding showed a missing-required-field bug with a nested object. I fixed
nested objects. The next round showed the same bug with an object inside an
**array** — and my comment now claimed the walk covered "EVERY depth", which was
a universal claim about a tree I had only walked two shapes of.

The invariant quantified over the whole schema tree; the fix quantified over the
example. **Before fixing, write down the set the invariant ranges over** — for a
schema that is objects, arrays, nulls and scalars; for a fuzz target it is both
branches; for a seam double it is every method — then confirm a test exists at
every point of it. A fix scoped to the reproduction is a fix that gets re-raised.

## A guard that cannot fail is worse than no guard (define #11 M2)

To stop a live suite reporting "drift" when the proxy was merely stopped, I made
it skip on `ErrUnavailable`. But a **renamed or withdrawn model** answers 502
through that proxy, which classifies as `ErrUnavailable` — so the suite would have
skipped silently on precisely the drift it exists to catch. The guard converted a
loud failure into a green run.

Two rules:

- **A skip condition must be narrower than the failure it protects against.**
  "Cannot establish a connection" means nothing is there; anything that *answers*,
  even with an error, is the dependency talking and belongs to the suite. Probe
  the endpoint, not the API.
- **A guard needs a test in both directions** — skips when it should, and
  *doesn't* when the dependency is merely broken. Mine had neither, and the
  reviewer found it by reading, not running.

**Recurred on this same helper in #25, and the recurrence is the more useful
lesson.** `SkipIfUnreachable` was routed through a new strict-mode guard, whose
whole purpose is "a check that cannot fail is not a check". Five rounds then
found the rule broken by its own implementation, each time one level in:

| the fix | how it failed the rule it implemented |
|---|---|
| the guard | shipped with no test of its own inversion |
| the test of the inversion | asserted `StrictEnv` against its own literal — a tautology |
| the enumeration of sites | written into Done-when, then applied to 2 of 3 |
| the mutation evidence | recorded as bare counts that did not reproduce |
| the test pinning the wiring | its want string was a PREFIX of the other mode's, so one direction could not fail |

Every one was found by a reviewer MUTATING the thing, and every one had been
verified by its author reading it. The author's reading is what produced the
defect; re-reading cannot find it.

**Rule: a fix for a "cannot fail" defect is not exempt from the rule it
implements.** Mutate the fix, not just the code the fix protects — break the
assertion deliberately and confirm it reddens. If the fix is a test, the mutation
is on the thing it claims to pin. Budget for this: the fix is where the next
instance of the bug lives, because it is written by the person who just
demonstrated they hold the wrong model.

**Corollary — a negative assertion needs its positive twin, and vice versa.** "The
message contains X" cannot fail when X is a prefix of what the other mode emits.
Assert what must be ABSENT alongside what must be present, or the check passes in
both modes and pins neither.

## Verifying only what you have tests for proves nothing new (define #11 M2)

Told two findings were still open, I re-ran my own tests, saw green, and reported
that the reviewer was wrong. Both findings were real: they named cases my tests
did not cover (an array of objects; a 502 from a reachable service). My
measurement was a restatement of what I already believed.

**When a finding is re-raised against a fix you believe landed, reproduce the
finding's own case** — not your test suite. The suite is what missed it the first
time.

**And reproduce the finding's case when you ACCEPT it, too (#17 BR-21).** A
review said four sanitiser sites had no positive control. Three did not; the
fourth was already covered by a test added the round before. I fixed all four,
wrote "deleting it left the whole suite green" into the issue Log, the commit
body and a new test's comment — and the added test was a duplicate. The false
claim reached three durable artifacts before anyone measured it, and one of them
is the ledger the process runs on.

**Rule: a claim about what the suite does or does not cover is a MEASUREMENT, and
that includes a claim you are repeating from a reviewer.** Run the mutation
against the tree the finding names before writing the fix, the comment, or the
log line asserting it. Accepting a premise on trust is the same error as
accepting a fix on trust — the reviewer's authority does not convert a claim into
a measurement.

**And the finding's ENUMERATION is a claim too (#17 BR-24, the very next round).**
Correcting the false measurement above, I fixed the three sites BR-21 listed —
issue Log, commit body, test comment — and the next review found a fourth, in
`reflect.go`'s own type comment. I had run the reviewer's list instead of my own
grep, which is the same substitution one level down: trusting an enumeration
because someone authoritative wrote it. A finding tells you a class exists; it
does not tell you where every member is.

**Rule: when a finding names N sites, grep for the class before believing N.** The
sweep costs one command; the alternative is a review round per missed site, and
here it was exactly that.

## Drive a traversal from the generator's vocabulary, not the reproduction (define #11 M2)

Four fixes in a row for one bug — missing required fields decoding to a partial
value with a nil error. Top level, then nested object, then array item, then map
value. Each fix covered the case the finding named and missed the next, and each
time my doc comment made a universal claim ("EVERY depth", "the WHOLE schema
tree") that the code had not earned.

The fix that finally held was not a fifth case. It was writing down **what the
schema generator can emit** — `properties`, `items`, `additionalProperties`, plus
`$ref`/`$defs`/`oneOf`/`anyOf` as latent under the current configuration — putting
that enumeration in the doc comment, covering each arm, and adding a test that
fails when a schema arrives carrying a keyword the walk does not know.

**When a bug recurs at a new shape, stop fixing shapes.** Find the thing that
*produces* the shapes and enumerate its outputs. And the test-side half matters
just as much: my fuzz target asserted a named field of one fixture type, so it
ranged over the one shape that already had cases — which is why every other
vehicle had to be found by a reviewer rather than by 900K executions.

## Fixing the helper is not fixing its inline twin (define #11 M2)

I rewrote `SkipIfUnreachable` so it probes the endpoint rather than the API,
because skipping on `ErrUnavailable` swallowed the drift it existed to catch. Then
I left, ten lines below the call, a local closure doing exactly the thing I had
just removed — so all four drift subtests still skipped on a renamed model.

**After changing a helper's behaviour, grep for the behaviour, not the helper.**
The name changed; the mistake had been copy-pasted before the rename, and it does
not answer to the new name.

## A test bounded only by the package timeout names nothing (define #11 close)

`TestLLMCheckHonoursCancellation` did redden under mutation — by hitting the
120s package timeout, because the code under test waits five minutes and the
elapsed-time assertion after it never ran. A timeout failure is indistinguishable
from an unrelated hang, a slow machine, or a deadlock somewhere else in the
package, and it takes the whole package down with it.

**Bound the call inside the test** — run it in a goroutine, `select` on a timer,
and fail with the sentence that names the cause. Same mutation now reports
*"runLLMCheck ignored a cancelled context; Ctrl-C would be swallowed for the whole
Timeout"* in 20s instead of an anonymous 120s timeout.

## Take the context you were given (define #11 close)

`main` installs `signal.NotifyContext` so Ctrl-C cancels rather than kills. A new
subcommand then built its own with `context.WithTimeout(context.Background(), …)`
— which silently opts out of that, so against a hung endpoint Ctrl-C did nothing
for five minutes. Measured: 5s+ with no exit, versus 250ms after the fix.

**A function that takes no context, or derives from `Background()`, is opting out
of every cancellation the process has arranged.** When a signal context exists,
every path that can block must be reachable from it — and the test for that is
"does Ctrl-C work while it is blocked", not "does it return eventually".

## An untestable branch is an unreachable knob (define #11 close)

A guard was silencing the wrong error, and four attempts to pin it with a test
all passed under mutation. The reason was not the test: `runLLMCheck` derived its
deadline from the resolved `Config`, and **nothing could shorten that** — no env
var, no flag — so the branch could only be reached after five minutes of a hung
proxy. Every fixture I wrote shortened the SDK's *per-request* timeout instead,
which is a different clock.

The fix was to add `DEFINE_LLM_TIMEOUT`. Once the deadline was reachable the test
became one line and the mutation reddened immediately with `stderr = ""` — the
exact silent exit the finding described.

**When a branch resists testing, ask what makes it unreachable.** Usually it is a
value nobody can set, and making it settable is a feature an operator wanted
anyway — here, anyone whose proxy is merely slow.

Corollary, learned four times in one sitting: **a mutation that leaves the test
green is a fact about the test, not a verdict on the fix.** Print what the guard
actually sees before concluding anything. `ctx.Err() = <nil>` answered in one line
what three rounds of reasoning had not.

## A per-request override is a second door into the same field (define #11 close)

`Config.MaxTokens` was normalised against negatives; `Request.MaxTokens` — the
per-request override — was not, so `-5` still reached the wire with a nil error.
Two doors into one field, and the fix went in one of them.

**When a field has both a default and an override, they are two entry points to
one invariant.** Normalising the default is half the work. The same applies to any
`cmp.Or(requestField, configField)` pair: `cmp.Or` replaces only the ZERO value,
so every negative walks straight through both.

## A table field no row sets is dead code that reads as coverage (define #11 close)

Two fields survived in a test-table struct after the row that used them moved to
its own test. The setup they gated — starting a hung listener, overriding a
timeout — then ran for nobody, and a comment still explained a race in a listener
the table no longer started.

**For every field in a table-driven fixture, confirm at least one row sets it.**
The check is mechanical and worth running over the whole repo rather than the file
in front of you; a field nothing sets is a branch nothing enters, and in a test it
looks exactly like a case that is covered.

## A crashed boundary review can leave the tree mutated (define #11 close)

The reviewer verifies findings by reverting a fix and re-running the tests. A
transient API 502 killed it mid-verification, so it never restored — and
`anthropic.go` sat in the working tree with a committed fix silently backed out.
The suite still passed, because the reversion step it was in had already finished.

HEAD was correct, so nothing was lost, but re-running without looking would have
meant testing a tree with a fix removed. **After a boundary review fails or
crashes, `git status` before anything else** — and revert what it left behind,
including the half-written sidecar and ledger entries for the round that never
completed.

The gate's own behaviour here is the model to copy: given no parseable verdict it
recorded `unknown`, said "a gate/prompt bug?", and refused to close. An
unparseable result is not a pass and not a fail.

## A rule stated in a comment is not a rule the suite enforces (define #16 M1)

Four boundary-review rounds on one milestone, and the same two families kept
coming back — `test-asserts-nothing` reached its 4th finding, `doc-overstates-code`
its 4th. Each round I fixed what the finding named, wrote the rule in a comment,
and the next round found the same family somewhere else. The gate's own summary
was the diagnosis: *not converging: fix rules, not instances.*

What actually made them converge was narrower than "state the rule":

**Name the enumeration the rule quantifies over, and make every cell fail on its
own.** "`-raw` never asks" quantifies over `{forced, unforced} × {one-shot, piped,
editor}` — six cells. The guard went on the unforced fallback, the README stated
the absolute, and three cells asked anyway. A table with six rows in the test is
not enough either: two of the six passed for the very failure they existed to
catch, because they asserted the *absence* of a string.

**An assertion that pins "X did not happen" must assert the positive observable
that distinguishes X from every other outcome.** Checking that stderr lacks the
ask message is satisfied by a *different wrong* message. Assert what did happen —
the `no dictionary entry` the scripting contract promises.

**An assertion on a message compares the bytes the user receives to a literal
written in the test.** `note: noteEmptyLiteral` is the constant compared to
itself: it proves a branch was selected and nothing about what the reader sees.
A doubled backslash shipped behind exactly that assertion — `define '\'` printed
`type a word after "\\"` — and four rounds of placement fixes never touched it,
because every round pinned *where* a message was written and none pinned *what
it said*.

**The same trap has a second shape: `Contains(output, someConstant + "text")`.**
Re-made in #21, two issues later, on a feature whose *entire deliverable* is a
colour. `Contains(got, knownOn+"obsequious")` looks like it pins bold-green, but
aliasing `knownOn = inputOn` — which removes every visible highlight — leaves it
green, because both sides move together. The mutation survived the whole suite.
Two rules, and the second is the one that generalises:
- **Write the escape bytes as a literal** (`"\x1b[1;32mobsequious"`). A test is
  the place where the expected value stops being a variable.
- **When the constant IS the deliverable, assert the property that makes it one.**
  Here that is `knownOn != inputOn`: the feature's claim is "easier to spot", and
  a highlight identical to ordinary text satisfies every byte-level assertion
  while delivering nothing.

**A double must fail ONLY the thing under test, or it proves the wrong claim.**
#21 M1: pinning "a word the deck rejected must not enter the highlight set", I
reached for the existing `failingStore` — which fails `AppendEvent` too. `Capture`
returns on that first failure and never reaches the deck write, so the test
passed while the mutation it existed to kill survived. The assertion was true of
"the store is broken", not of "the deck said no". When a function has several
exits, a blanket-failing double stops at the first one; build the double that
isolates the exit you are naming.

**A test that injects a double must inject where production reads, or prove the
injection is live.** `withStore` only fills nils, so a `newStore` supplied
alongside an already-set field is silently discarded and every assertion over the
double ranges over an empty slice. Keep one control assertion that goes red when
the injection dies.

**In a loop that echoes, an assertion on stdout is satisfied by the echo.** The
recall test passed with `hist.Add` deleted, because the raw editor re-renders the
line on every keystroke and the text was there from *typing*.

And one that is not about tests: **a defect fixed once will be reintroduced by
the next branch that needs the same shape.** `replLines` collapsed a dispatch's
exit code into a boolean — the defect a comment 45 lines above names by number,
fixed for commands in #15 and re-made for questions in #16. The fix is a single
sink every branch feeds, not a third careful branch.

## The `Review-Verdict:` trailer marks a boundary — never put it on a fix commit (define #16 M2)

A REWORK verdict prints trailers alongside its findings, the same way
FIX-THEN-SHIP does. I pasted them into the commit that FIXED the findings. The
gate finds the previous boundary with `git log --grep 'Review-Verdict'`, so that
commit became the boundary, and the next review ran over a window of
`471b376..471b376` — **empty**. It reported "0 new findings, converging" while
all eleven prior findings sat undisposed, and produced no verdict at all.

The rule: **that trailer is a claim that this commit CLOSES a boundary.**

- **FIX-THEN-SHIP** — the verdict sanctions shipping after the fixes, so the
  fixes and the close mutations are ONE commit and it carries the trailer.
- **REWORK** — the verdict is "address the findings, then re-run". The fix
  commit carries no trailer; the trailer arrives with the close that follows.

Costly in a quiet way: nothing errored, the gate said "converging", and only the
window in the output — start SHA equal to end SHA — showed the review had been
handed nothing to look at. **When a review reports zero findings on a diff you
know is large, read the window before believing it.**

## Use `git stash` for mutation safety, not a `wip` commit (define #6 close)

After a scratch-copy restore silently deleted a function, I adopted the rule
"commit, then mutate, then `git checkout HEAD -- <file>`". The rule is right — a
restore has to target something versioned — but the mechanism I chose leaves
unexplained commits in history. Seven of them here, one 507 lines across three
files, and the close review flagged it: a reviewer reading the branch finds half
a milestone's work under the message `wip`.

Mechanism folded into *Mutation testing needs a COMMITTED baseline*, which is the
one home for this rule. The squash recipe lives there too.

## Never report a boundary closed without READING the verdict (define #6)

I ran `sdlc milestone-close` for M1 in the background, the completion
notification arrived while I was mid-M2, and I never opened the output. I then
told the operator "M1 built", ticked the `- [x] M1` row, and worked on top of it
for the rest of the session. The close had FAILED with five open Importants — one
of which was that I had claimed a package's guards were "verified in the tree"
when the package had no tests at all.

Two rounds later the close review caught the tick itself: *"the M1 row is ticked
although the M1 boundary review blocked with four Importants that are still open,
and no verdict trailer or close line exists for it."*

- **A backgrounded gate is not a completed gate.** The notification says the
  COMMAND finished, not that it succeeded. Read the output before saying anything
  about the milestone, and before ticking anything.
- **The tick is a claim, and it is checkable.** A ticked `Mx` with no
  `Review-Verdict:` trailer and no close line in the Log is a claim with no
  evidence behind it — which is exactly what a later review looks for.
- **The cost is not the lost round; it is the work built on top.** Everything in
  M2 sat on a milestone that had not passed, so its findings arrived after the
  code that inherited them.

## A guard nothing has SEEN fail is indistinguishable from one that cannot (define #6 M1)

At `#5`'s close I recorded a gap: every "the mutant reddens it" claim was verified
in a scratch copy and thrown away, so nothing in the tree proved the purity guards
could fail. At `#6` M1 I extracted those guards into a package, verified them the
same way — mutate, observe, delete — and wrote in the issue Log that the negative
cases were "verified in the tree". **The package had no test file at all.**

So the fix for a gap reproduced the gap, and described itself as the fix.

- **"Verified" means a committed artifact re-runs the check.** A scratch mutation
  I watched fail is evidence for me, once. It is not evidence for the next reader,
  the next change, or CI.
- **A guard needs its own known-bad fixture.** `puretest/testdata/impure` imports
  `os` and calls `store.NewYAML`; `testdata/clocky` imports only `time` and calls
  `time.Since` — the case the import allowlist structurally cannot catch. Each
  guard is now run against them and asserted to FAIL.
- **Take the minimal interface, not `*testing.T`.** That one change is what makes
  a guard testable at all, because a recorder can stand in for the `T` and capture
  the failures instead of failing.

## A guard that allows a PACKAGE allows everything in it (define #5 close)

`schedule`'s purity had two guards — an import allowlist and a wall-clock grep —
and both passed a package that called `store.NewYAML(dir, w)`. That constructor
opens a directory and reads and writes files. The import guard allowed it because
`store` is on the allowlist; the clock guard allowed it because it names no time
function. **The purity claim would have been false with every guard green.**

- **Allowlisting a package grants its whole surface, including the parts that
  contradict what you were guarding.** `store` holds both the pure types this
  package needs and the disk this package must not touch.
- **When a dependency is mixed, guard SYMBOLS, not packages.** The new guard lists
  the eight pure things `schedule` may name; anything else fails. Adding a ninth
  becomes a visible decision rather than an implicit one.
- **Two guards agreeing is not two independent checks** if they share the same
  blind spot. Both of these reasoned about names — one about package names, one
  about function names — and neither about what the named thing DOES.

## A t.Skip on the only pin for a fix is not a pin (define #5 close)

The Critical this round found — `Due` firing on the day of review — was pinned by
two tests, and BOTH began `if err != nil { t.Skipf("no tzdata") }`. On a machine
without the system timezone database, the only checks on a real correctness fix
would report green while verifying nothing.

The repo had already solved this: `history_cmd_test.go` imports `_ "time/tzdata"`,
embedding the database so its zone tests RUN. I wrote the skip instead, one file
away from the fix.

- **A skip is an admission the test might not run. For a test that pins a
  correctness fix, that is the same as not having it.** Make the dependency
  available instead — here, one blank import — and turn the guard into `t.Fatal`,
  so absence becomes a failure rather than a shrug.
- **Before writing a skip, grep for how the repo handles that dependency
  already.** The answer existed and cost one line.
- **And the follow-up fix has to be verified, not assumed.** My first pass added
  the import to one of the two files and wrote the explanatory comment into BOTH.
  The second file then carried a comment stating the import was there when it was
  not — a false claim minted by the fix for a false claim.

## Consolidating two implementations? Keep the one whose comment explains itself (define #5 close)

`#15` computed "which local day is this" twice for `/history`: once building a
local midnight and subtracting, once projecting the LOCAL date onto a UTC day
index. I consolidated them into `store.DaysBetween` and kept the shape that read
more cleanly — a loop stepping the calendar with `AddDate`.

It was WRONG, and the discarded one was right. Stepping in `a`'s zone while
comparing instants against `b`'s makes two instants on the same local day count
as one day apart. Consequence at the product level: a word reviewed this morning
was offered again this afternoon. Store stamps carry FIXED offsets (yaml.v3
parses them that way) and `now` comes from `time.Local`, so **mixed locations are
the normal state, not an edge case** — and no existing test crossed zones, so it
was green.

The discarded implementation's comment said exactly why it had its shape:
*"Projecting the LOCAL date onto a UTC day index takes DST out of the arithmetic
instead of compensating for it."*

- **A comment explaining a non-obvious SHAPE is evidence the shape was chosen,
  not stumbled into.** When two implementations disagree in form, the one that
  documents its own weirdness is the one that met the hard case.
- **Consolidation is a behaviour change until proven otherwise.** The regression
  net was `/history`'s tests, and they passed — because `relativeDay` normalises
  its arguments first (`at.In(now.Location())`) and so never exercised the bug.
  A refactor's regression net only covers what the OLD callers did.
- **Ask what the new caller does differently.** `Due` compares a stored stamp
  against the system clock. That pairing did not exist before, and it is exactly
  where the bug lived.

## Restore from git, not from a scratch copy (define #9 M1)

Recurrence of the entry below, one issue later and with a new cause. Mutation
testing means copy-file, mutate, test, copy-back — and I took the backup at the
START of a milestone, then kept writing code. Several functions later, a
copy-back silently DELETED `bothSources`, because the backup predated it. The
build broke immediately, which is lucky: a deletion inside a rarely-run branch
would have shipped.

- **Third occurrence, #5's close: a `git checkout HEAD` run to revert one
  mutation also reverted an unrelated, uncommitted fix in the same file** — the
  `LastBox` const change vanished and was only noticed because the follow-up
  verification came back empty rather than red. **An empty result from a check
  that should have failed is itself a finding.** The two-step rule below is not
  optional discipline; skipping the commit is how work disappears silently.
- **`git checkout HEAD -- <file>` is the correct restore, and only if the target
  is COMMITTED.** Evidence for the rule stated in *Mutation testing needs a
  COMMITTED baseline*; see that entry, and do not restate the rule here.

## A backup is only as good as the tree it was taken from (define #16 M2)

M1's lesson was *make the backup first, and restore from the backup, not from
git*. That is necessary and not sufficient. Here is the shape it missed:

1. A background mutation job was killed mid-flight, before its restore step, so
   it left `ask.go` MUTATED.
2. The next command started with `cp cmd/define/ask.go /tmp/…` — snapshotting the
   corrupted tree as its "known good" copy.
3. Restoring from that snapshot put a `restore := func() {}` stub into the
   working tree, disabling the very interrupt scoping the round had just added a
   test for.

Nothing errored. The command printed `tree restored, builds`, and it was true —
it built fine, with the mechanism disabled.

Evidence for *Mutation testing needs a COMMITTED baseline* — the provenance of a
scratch copy is exactly what cannot be vouched for, which is why the committed
baseline is the rule. After any killed job, restore from GIT and re-apply
intended edits by hand. The
mutation experiments in this repo take longer than the tool timeout, so they run
in the background, which makes "is anything else editing this file right now" a
real question rather than a rhetorical one.

The generalisable half: **`git status` and "it builds" both pass on a tree with a
mechanism silently removed.** What catches it is diffing the file against HEAD
and reading every hunk — which is also what caught it here, one step before a
commit.

## A mechanical check can pass vacuously too (define #16 close)

The entity-table check failed five rounds running, and each fix made it *more*
mechanical: run the enumeration → run it against the working tree, not HEAD →
reconcile by set difference rather than by eye. The sixth round found the set
difference **passing because the tables contain the string `cmd/define/ask.go`**,
which matches `\bask\b`. The function `ask` — the single entry into the question
path — was in no row, and the check said everything was covered.

Two rules, and the second is the general one:

- **Match where a thing is NAMED, not anywhere in the document.** Table rows and
  bullet headers name entities; prose contains the English word "ask" and paths
  contain `ask.go`. Both satisfied a whole-text search.
- **A check you wrote to defend a finding is itself a claim, so probe it.** The
  fix for "the tables drift" was a script, and the script needed exactly the
  falsification test its own finding was about: feed it something you KNOW is
  missing and confirm it says so. I never did, so it reported "none missing" for
  three rounds while three symbols were missing.

Same rule as the mutation-table and the message-count entries above, applied one
level up: the check is a claim, and an unfalsified claim is scaffolding whatever
language it is written in.

## A class found in one place is not fixed until you look for it in the others (define #17 M1)

Smoke-testing `--reflect` live, I caught my own verification being vacuous: the
corrections check printed "preserved", but the second run had FAILED and written
nothing, so nothing could have changed. I fixed that check, reported it, and
moved on.

The boundary review then found the identical hole in two unit tests standing
three feet away — `TestReflectIsIdempotent` and `TestReflectPreservesCorrections`
both compared a file before and after, and a run that fails writes nothing, so
"unchanged" and "suffix preserved" are satisfied by a file nobody touched.

I had named the class out loud and swept exactly one instance of it.

**When a finding is about a SHAPE rather than a line — a comparison that a
no-op satisfies, an assertion the echo already satisfies, a claim that cites
nothing — grep the shape before closing it.** The cost of looking is one search;
the cost of not looking showed up as a blocking finding in the next round, twice
in two issues.

The mechanical version, for the shapes seen so far:

- comparing a file before/after → does a FAILED run also satisfy it?
- asserting on output a loop echoes → does typing alone satisfy it?
- asserting a value reached an output → could a different source supply it?
- a table of examples over human-edited text → is there a malformed class?

## Sanitise where the structure is built, not at each place text is used (define #17 M1)

A finding said "model text is rendered verbatim into a markdown table". I
sanitised the four fields it listed. The next round found the evidence words —
same text, two more render sites, not on the finding's list. A sweep after that
found the frontmatter's model name and every diagnostic message, which go to a
terminal one line each and can forge a `define: …` line a reader cannot tell from
a real one.

Three rounds, one class, because each fix was a LIST of call sites and the list
drifts from the type.

**The fix that ends it is structural: one pass over the whole struct, at the
point the structure is created.** `renderUserModel` is where the file's shape
exists, so it sanitises the model it was handed before rendering anything. A
render site added later is safe without anyone remembering.

Two general forms worth carrying:

- **"Untrusted text in a table" is never the class.** The class is untrusted text
  reaching any structured output — the file, the frontmatter, the diagnostics,
  the terminal. Enumerate the SINKS, not the fields.
- **Collapsing newlines beats filtering for the dangerous string.** Every line
  the renderer emits is prefixed by `**`, `Read off: ` or `| `, so text that
  cannot start a line cannot forge structure *nobody has thought of yet*.
  Filtering for `## Corrections` would have to be updated for the next marker.

## Choose injection text that does not satisfy your own assertion (define #17 M1)

Testing the above, I wrote two assertions in a row that could not fail:

1. *"every stderr line starts with `define: `"* — with injection text that itself
   began `define: `, the forged line passed.
2. *`Contains(injected + "\n")`* — missed, because the forged line carries the
   rest of the message after it.

The mutation passed both. What works is asserting the thing injection actually
changes: it adds LINES, so count them.

**When testing an injection, ask what the injection changes that the assertion
measures** — and pick payload text that is unmistakable and inert (`FORGED-LEVEL`),
never text shaped like the thing you are checking for.

## A guard whose effect is ABSENCE needs a counting double (define #21 M2)

Seventh finding in one family across three rounds, and the survivors had a shape
my earlier enumerations could not reach. Every rule I had written quantified over
things that change OUTPUT. These do not:

- a **wiring argument** — passing `p.ex` as the style to resume. Replace it with
  `""` and production bytes change (the rest of the example goes unstyled) but no
  test looked at those bytes, only at whether a highlight appeared.
- a **guard whose only effect is work not happening** — `!opt.color` in
  `vocabularyFor`. Delete it and the whole deck is read under `-no-color`. No
  output assertion can ever see that, because the output is identical.

The rule that reaches both: **for each behaviour a comment claims, name the
observation that would falsify it.** Production output bytes for wiring; a
counting or spying double for a guard. The package already had `countingDeck`
doing exactly this for "read the deck once", and the colour gate reused it
verbatim — the tool existed, the enumeration just never asked for it.

Sharper still: that colour gate had been found and fixed one milestone earlier.
A refactor moved it, and nothing was watching, because nothing ever had been.
**A fix without a test is a fix with a half-life.**

## A property is only as wide as its fixtures — the deck is input too (define #21 M2)

`FuzzHighlightWriterIsChunkIndependent` asserted that splitting a stream anywhere
produces identical bytes. 803k execs, clean. It proved almost nothing: the
vocabulary was pinned as a constant — `vocab("obsequious", "hot dog", "hot")` —
and with no joiner-bearing or multi-byte entry in it, the entire class of "a
chunk splits inside a joiner or mid-rune" was **unreachable at any exec count**.
Two real data-loss bugs sat underneath: `don'`+`t` lost `don't`, and a chunk cut
mid-rune lost `café`.

- **Everything the function reads is input, not just the fuzzed argument.** A
  fixture held constant silently removes a dimension from the property. If the
  behaviour depends on it, fuzz it or derive it.
- **Derive fixtures from the package's own enumeration.** `TestWordRuns`' table
  already listed every word-character class this code distinguishes — apostrophe,
  hyphen, digits, multi-byte. Hand-picking a deck instead of reusing that table
  is how the gap got in.
- **Exec count is not coverage.** "803k execs clean" reads like assurance and
  measures only how long an unreachable class stayed unreachable.
- **Enumerate class × POSITION, not class alone.** The round-1 fix derived the
  deck from the tokenizer's character classes and still missed the bug: `café`
  was the only multi-byte entry and its multi-byte rune is word-FINAL, so a
  word-INITIAL one was unreachable — and that was precisely the shape the broken
  code path needed. Where a character sits in a token is part of the class.

## Enumerate the production chain from the ENTRY POINTS (define #21 M2)

I wrote this rule at M1 — *enumerate the production dependency chain, not the
comments* — and the same family came back one milestone later, because my chain
started in the wrong place. M1's table began at `openStore`. But `openStore` is
itself a hop *in*: the thing being enumerated has to start where the PROCESS
starts.

M2 wired highlighting into `lookupAndRender` while `Load()` sat in `runEditor`.
Three entry paths reach that render — one-shot `define <word>`, piped stdin, and
the raw editor — and only the third loaded the set. Two of three were dead, the
suite was green, and the README sentence I had just written was false for the
exact command it named.

- **A test that injects a filled dependency begins after the thing that fills
  it.** Every test set `rig.deps.vocab` to a pre-populated set, so the load hop
  was invisible in exactly the way `withStore`'s merge had been invisible one
  round earlier. Drive at least one case with the dependency in its REAL initial
  state (here: an unloaded store vocabulary).
- **The enumeration is "every entry path that reaches this surface", and it
  belongs in a table test.** A new entry path then either appears as a row or is
  conspicuously missing.
- **When a rule recurs, the rule was too narrow — do not just re-apply it
  harder.** Twice now the fix was to widen where the enumeration STARTS.
- **The axis is entry path × RENDER SURFACE.** I wrote the entry-path table
  specifically so a render path could not be added without a row — and then added
  a whole new surface (the answer stream) one milestone later and did not widen
  it. Three of six cells, while the atlas called it the guard for "every render
  path". A table guards the axes it enumerates and nothing else, so when you add
  a dimension, the table is stale even though every row in it still passes.

## A bound derived from an input set must come from the subset that can use it (define #21 close)

`MaxPhraseWords` is the only input to the streaming writer's hold arithmetic, and
it counted the tokens of EVERY deck key. But a key whose tokens cannot rejoin —
`e.g.`, or `rock 'n' roll`, whose gaps carry an apostrophe — can never match
anything, so it widened the tail every stream holds in exchange for nothing.
Measured: a single-word deck held 5 bytes; adding the unmatchable `e.g.` held 9,
the same cost as a real three-token phrase.

- **A maximum taken over a set is a claim about that set's useful members.** Ask
  which members can actually exercise the bound, and take the max over those.
- **The subtle member is the one that LOOKS usable.** `rock 'n' roll` tokenizes
  to three words, so a punctuation check would have admitted it; only asking "do
  these tokens rejoin under the real matching rule" rejects it.

## A concurrency comment is a claim, and the suite cannot falsify it by accident (define #21 close)

`memVocabulary`'s doc comment gave a premise ("the two accesses are genuinely
concurrent") and a conclusion ("so it is mutex-guarded"). Measurement contradicted
both: the loop's goroutines carry values over channels and none touch a
vocabulary, so every access ran on one goroutine — and `storeVocabulary` had added
an UNGUARDED `loaded bool` that `vocabularyFor` writes on every render. Driving it
from eight goroutines under `-race` reported a race immediately.

- **`go test -race` proves nothing about code no test runs concurrently.** A
  clean race run over single-goroutine tests is evidence about the tests, not the
  type. A type that claims safety needs one driver that would fail without it.
- **Embedding inherits the lock but not the discipline.** The mutex was on the
  embedded struct; the new field beside it was bare, and nothing connected them.

## Line numbers and mutation claims in a plan are code that nothing compiles (define #21 close)

One finding stayed open for FIVE rounds — the longest-lived of the issue — because
each round I fixed the divergences the note listed and the next round found more.
The instances were never the point. A plan file accumulates three kinds of claim
about code, and all three rot silently:

- **Line numbers.** `ask.go:160` became `ask.go:171` the moment a wrapper landed
  above it. Nothing checks them, and being *nearly* right is worse than being
  absent — a reader follows one to the wrong function. Cite the file and the
  symbol; drop the number.
- **Contracts stated twice.** Rule 4 said "counts in the caller's units" in two
  places. I corrected one and the other kept promising the opposite for three
  more rounds.
- **Mutation results.** A ticked "mutation-check that X reddens a named test" is
  an assertion, and mine was FALSE — deleting the flush reddens nothing. When
  measurement disagrees with the step, correct the step; do not tick it because
  the work was done.

The rule the recurrence taught: **sweep the class of claim, not the list in the
finding.** A note that names three divergences is a sample, and treating it as
the enumeration is how one finding survives five rounds.

And it survived two more, for a reason worth its own line: **a fix can create a
fresh instance of the finding it is fixing.** I corrected a plan step to say "this
mutation reddens nothing" — true and honest when written. The next commit added
the test that makes it redden, so my correction became false in the same change
that made it obsolete. A statement about what a mutation does is invalidated by
any change to the code it describes, *including one that improves it*. That is
why the stale-claim sweep belongs on every commit touching the code, not only on
the commits where a reviewer lists instances.

## An assertion guarded on the run's own output is not an assertion (define #21 close)

A row meant to pin "an interrupted stream leaves nothing dangling" read
`if tc.cancel && got != "" && !strings.HasSuffix(got, "\n")`. With an
already-cancelled context `runAsk` returns before any delta, so `got` is empty,
the guard never fires, and the row asserted nothing — while the test's own
comment said it pinned that no path leaves text dangling.

- **Guarding on the fixture is fine; guarding on the RUN's output is not.** The
  first is a precondition you control, the second silently converts "the
  behaviour held" into "the behaviour never happened".
- **Write the expectation per row, including the empty one.** `wantEmpty: true`
  is an assertion; `if got != ""` is an escape hatch.
- **When the observable is empty, `t.Fatal`.** The package already had that idiom
  at five sites; the vacuous row was the one place it was missing.
- **A comparison over a corpus needs a hit COUNT, not just a pass.**
  `TestHighlightingLosesNothing` compared highlighted output against plain across
  32 entries — but only 6 of them contain a deck word, and for the other 26 it
  compared two identical strings. Swapping the deck for an unmatchable word left
  it green. Count the entries that actually exercised the behaviour and fail at
  zero; log the number so a corpus refresh that quietly stops matching is
  visible.

## The vocabulary is withheld per region, decided at the boundary (define #21 M2)

Highlighting wrapped the whole rendered definition, so a word the learner
revisits — the common case for a learning tool — rendered green inside its own
bold-cyan headword, which the Spec explicitly puts out of scope.

- **A finished string has no structure left to consult.** Per-region decisions
  have to be made where the regions still exist. `RenderOpts.Vocab` reaches
  `Render`; `admitsHighlight`'s doc comment is the admit/withhold table.
- **Do not write a COUNT beside an enumeration.** "the ten-region table" was
  wrong within one round — the correction that completed the table also split two
  rows, and the arithmetic was not redone. A number in prose next to a list is a
  second source of truth that nothing checks; delete it and let the derived test
  be the record.
- **State the decision for every region, including the obvious ones.** "Prose is
  admitted, labels are withheld" is a rule; "I wrapped the string I had" is not,
  and cannot be reviewed.

## Enumerate the production chain, not the comments (define #21 M1)

Four findings across two boundary rounds, one family: a behaviour with no test
that fails when you break it. Round 1 fixed two instances and stated the
enumeration as *"for each behaviour the diff states in a comment, is there a
mutation that falsifies it and a named test that reddens?"* Round 2 found the
family at full strength again — because the missing hop, `withStore`'s one-line
merge of `sd.vocab` into `deps.vocab`, **carries no comment and makes no claim**.
A comment-driven sweep is structurally blind to it. Deleting that line kills the
feature outright in production and leaves the entire suite green.

- **The enumeration is over hops, not sentences.** For every seam a feature
  introduces, write out each hop from construction to use, and require one test
  per hop that crosses it *through production code*. Writing the chain down is
  what makes a hole visible; a prose rule about comments is what hides one.

  | # | hop | pinned by |
  |---|---|---|
  | 1 | `openStore` builds one set, hands it to capturer + `storeDeps` | ✓ |
  | 2 | `withStore` merges `sd.vocab` → `deps.vocab` | **was nothing** |
  | 3 | `runEditor` reads `d.vocab`, calls `Load()` | ✓ |
  | 4 | `RenderLine` consumes it → stdout | ✓ |
  | 5 | `Capture` → `Add` → next frame | ✓ |

- **A test that sets `rig.deps.X` directly begins AFTER the wiring hops.** Both
  loop tests did, which is exactly why hop 2 stayed invisible while looking well
  covered. At least one test per seam must build deps the way a production entry
  point builds them (`deps{newStore: openStore}.withStore(...)`).
- **A stated rule that does not name its enumeration will be declared swept while
  the family is still live.** "I applied the rule" is a claim about the set you
  enumerated, not about the class.
- **Sweep by GREPPING THE CONCEPT, not a remembered phrase.** #5's close: I
  "swept" a stale claim across artifacts, ran a residue check that came back
  empty, and reported it done — while three of five sites stood, because my grep
  matched one exact wording and the others said the same thing differently. The
  residue check inherited the same blind spot as the sweep. Search for the
  distinctive TOKEN (`store` near `time`), not the sentence.

## A new runtime directory has three homes that cannot see each other (define #9 close)

`usage/` joined `words/` and `events/` as a directory `define` writes into the
working directory — and reached none of the three places that needed it:
`.gitignore`, the index guard, and the history guard. Both guards hardcoded
`p == "words" || p == "events"`, and `.gitignore` listed the same names again.
Three copies, nothing keeping them in step.

This is the deck-in-git class, and the `.gitignore` comment already records that
it cost three review rounds before this one — including the subtlety that the
patterns must be UN-ANCHORED, because `go test` runs with cwd set to the package
directory, so an anchored pattern misses `cmd/define/words/`.

- **Single-source the list where the writer lives.** `store.RuntimeDirs` is the
  one place; both guards ask it instead of repeating it.
- **Close the loop the compiler cannot.** `.gitignore` is not Go, so nothing
  makes it follow that list — except a test that reads the file and asserts an
  un-anchored entry for every name. Adding a fourth directory and forgetting the
  ignore now fails, and so does re-making the anchoring mistake.
- **When a fact lives in a comment because no test could hold it, ask again.**
  The anchoring rule was a well-written comment that had already failed three
  times. A comment explains; only a test enforces.

## Adding a field is not wiring it — the third recurrence (define #9 close)

I added `warn io.Writer` to the usage source, wrote the warn-once logic, and
tested it by passing a buffer into a hand-built struct. **Both production sites
left it nil**, so a broken feed degraded exactly as silently as before, and the
test was green throughout.

Third time in two issues, same shape every time:

| issue | field added | production sites that got it |
|---|---|---|
| #21 | `deps.vocab` needs `Load()` | 1 of 3 entry paths |
| #9 M3 | `deps.usage` | 0 of 1 on the no-capture path |
| #9 close | `bothSources.warn` | 0 of 2 |

- **A test that CONSTRUCTS the struct begins after the hop that fills it.** That
  sentence is the whole family. `&bothSources{warn: &buf}` proves the field is
  read; it says nothing about whether anything writes it.
- **When you add a field, the test is `deps{newStore: openStore}.withStore(...)`
  and an assertion the field arrived.** Not the struct literal. The first form
  fails when a production site is missed; the second cannot.
- **Enumerate the construction sites, not just the type.** There were two here
  and I would have found both by grepping for the literal — which is a ten-second
  check I did not run because the feature "worked".

## A degrading fallback hides a test that reaches the network (define #9 close)

`TestWithStoreCarriesTheUsageSourceThrough` drove production wiring and then
called the seam — so every plain `go test ./cmd/define/` fetched
news.google.com and wrote ~48 KB of live headlines into a temp dir. Its own
comment said "reaches the dictionary half without a network" and its failure
message said "no usages offline". Both false.

What hid it is the feature working correctly: `bothSources` degrades to the
dictionary when the feed fails, so the assertion `len(got) != 0` was satisfied
either way. **The test was green with and without a network, and therefore
verified neither.**

- **A test whose assertion survives the dependency being absent is not testing
  the dependency.** Ask what would change if the network were unplugged. If the
  answer is nothing, the test is about something else.
- **Split the claims.** The WIRING hop belongs on production deps and asserts
  only that the seam is built and carried. The OFFLINE claim belongs on a source
  constructed with no feed at all, where "offline" is a property of the code
  rather than of the machine.
- **Coverage is the cheap detector.** `Fetch` reporting 72.7% under an untagged
  run is impossible unless the default suite calls it. It reads 0.0% now.

## A comment saying "on failure X we do Y" needs a test that reddens without Y (define #9 close)

Sibling to the output-field rule. Every fallback in `news.go` was described in a
comment and none was pinned: inverting "an unreadable cache is a miss, not a
failure" into `return nil, err` — the opposite policy — left the whole suite
green. The `failingStore` fixture that drives it was already in the tree, unused.

- **The enumeration is the coverage profile.** Every uncovered block in a new
  file is a claim nothing checks, and reading them off took one command.
- **A fallback is a behaviour, not an implementation detail.** "Degrades on
  failure" is a promise to the caller and deserves the same pinning as a
  returned value.
- **Degrading SILENTLY is a different design from degrading.** A permanently
  broken feed was indistinguishable from "this word is not in the news" — for
  the reader and for the downstream consumer. The package already had the shape
  (`warnTo`, warn-once); the new code just did not use it.

## A fuzz property may assert only YOUR contract, never the input's textual form (define #9 M1)

Three properties on one target, each wrong the same way, before the rule was
clear. All three were claims about the BYTES rather than about my code, and a
decoder is entitled to transform bytes:

1. **"a title is a SUBSTRING of the input"** — mixed content concatenates:
   `0<![CDATA[0]]>` is legitimately `00`.
2. **"...a SUBSEQUENCE of the input"** — entities decode: `&#65;` is `A`, `&#39;`
   is `'` (what Google News actually emits), a lone CR becomes LF.
3. **"no more items than `<item` tags"** — `encoding/xml` matches by LOCAL name,
   so `<x:item>` is an item no textual count can see. The fixture already
   declares a namespace prefix, so a two-byte edit reaches it.

Each fix produced the next failure, and only at the third did the shape become
obvious: **ask whose contract the property tests.** Character provenance inside
an XML element is the standard library's contract, not mine. What is mine is that
a failed parse returns no items, and that a date I cannot read becomes the zero
time rather than a guess — and the second of those deserved its own target on the
STRING, where no decoder stands between the property and the code it describes.

- **A property that reddens on correct input is a liability.** The danger is the
  response to it: weakening what it defended, or skipping the inputs that trip
  it. Both leave something that looks like coverage.
- **When a property needs a decoder to hold, test the decoder's side separately.**
  Splitting `parsePubDate` out gave a target that kills the guessing mutant on the
  SEED corpus — no `-fuzz` run required.
- **Seed the corpus with whatever refuted the last property**, so reinstating it
  fails locally instead of in the wild.

## (superseded) An earlier statement of the rule above (define #9 M1)

Kept because the count-bound it recommends was ITSELF wrong — the third failure,
not the fix. Read the entry above instead.

`FuzzParseRSS` carried three properties before one was sound, and the first two
failed the same way: too strong, red against entirely correct parsing.

1. **"a parsed title is a substring of the input"** — died in two seconds to
   `<title>0<![CDATA[0]]></title>`, which XML legitimately concatenates to `00`.
2. **"...is a SUBSEQUENCE of the input"** — survived that and dies to entity
   decoding: `&#65;` yields `A`, `&#39;` yields `'`, and a lone `\r` yields `\n`
   under XML line-ending normalisation. None are subsequences of their input, and
   `&#39;` is exactly what Google News emits for apostrophes — so re-capturing the
   fixture would have turned it red against working code.
3. **"there cannot be more items than item tags"** — claimed sound under any
   decoding. It is NOT: `<x:item>` defeats it. This is where the entry above
   picks up.

- **The danger is not the false failure; it is the response to it.** A property
  that reddens on good input invites weakening the thing it was defending, or
  skipping the inputs that trip it. Both leave you with a test that looks like
  coverage.
- **Ask whose contract you are testing.** Character provenance inside an XML
  element is `encoding/xml`'s contract, not mine. What is mine is how many items
  I report and what I do with a date I cannot read — and those are exactly what
  the sound property pins.
- **Seed the corpus with what refuted the old property.** Those three shapes are
  now seeds, so a future weakening fails here rather than in the wild.

## The atlas is due at EACH milestone — twice refused now (define #5 M1)

Second identical occurrence. `#21`'s plan scheduled all atlas work at M3 and the
close gate refused M1 for it; I recorded the correction in that plan. `#5`'s plan
then scheduled all atlas work at M2 and the gate refused M1 again.

- **A plan that lists "atlas" once, at the end, is already wrong** whenever the
  work has more than one milestone. AGENTS.md §8 says each close, and the gate
  enforces it — so the plan should carry an atlas step per milestone from the
  first draft, not acquire one after a refusal.
- **Correcting the instance in one plan does not carry to the next plan.** The
  fix lived in `#21`'s revision history where writing `#5`'s plan never looked.
  A lesson entry is where a rule has to go to survive into the next issue, and
  this is that entry.

## Doc prose at a boundary describes what THAT milestone shipped (define #21 M1)

Round 1 flagged an atlas sentence claiming a path a later milestone builds. The
fix commit corrected that sentence **and wrote a fresh instance of the same
defect into README.md in the same commit** — "highlighting appears in the line
you type, definition bodies, and answers", when only the typed line existed. A
reader following it would look up a word, read a definition, and see no green.

- **The site was fixed; the class was never enumerated.** The enumeration is
  every doc file the boundary window touches × every sentence describing the
  feature, each checked against what is reachable in code at HEAD.
- **Write the milestone's scope into the sentence, or mark the rest as future.**
  The atlas sentence that survived says "used by the prompt line today and by the
  definition and answer paths from M2". That form cannot rot into a lie.

## A stale property stays green until you actually re-fuzz (define #21 M1)

I changed `wordRuns` to trim quotes and hyphens off token edges — a real fix,
since `'obsequious'` otherwise never matches a deck key. That deliberately makes
runs non-maximal, and `FuzzWordRuns` asserted maximality. **The target was red at
HEAD and I did not know**, because `go test ./...` runs a fuzz target against its
SEED CORPUS only, and no seed happened to place a joiner beside a kept run.

- **Changing a contract means re-running the property that asserts it, with
  `-fuzz`, not with `go test`.** Seeds passing is not the property holding.
- **When you fix a tokenizer, seed the corpus with the shape you just changed.**
  `'0`, `-a`, `a-` are three characters each and would have caught it instantly.

## A test helper that skips the production resolution makes every test under it vacuous (define #20)

`typeKeys`, the helper every editor test drives through, resolved candidates with
`h.Prefix(e.WalkBase())`. Production resolves them with `completionsFor`, which
picks a *namespace* first. So the helper had been quietly testing a path that
does not exist since #15: no editor test could see command mode at all.

Two tests written specifically to pin #20's recall/complete split passed the
moment they were written, before the split existed. They were asserting against a
history-only resolution that never had the bug.

- **A helper is part of the production path or it is a second implementation of
  it.** Wire helpers to the same function the loop calls. If the helper needs
  arguments the loop has (here, `commands`), give it them — the seam that is
  awkward to reach in a test is usually the one carrying the behaviour.
- **A new test that passes before the code exists is a finding, not luck.** That
  is the cheapest possible signal that the test is not connected to the change.
  Stop and find out what it is really asserting.
- **Fixing the helper is not fixing the class.** The close review found four more
  `Suggestion(e, h.Prefix(...))` sites in the same file — sites the Plan had
  ENUMERATED by line number in the row I ticked. I fixed the helper, wrote this
  lesson about it, and walked past the four siblings the lesson describes. When
  you can write the enumeration, sweep the enumeration in the same round; a
  lesson recorded is not a sweep performed.

## A regression test needs data that tells the two implementations apart (define #20)

`TestCommandCompletionIsUnchanged` typed `/his` and `/history 7` against a
history holding `historic`, and asserted the tail. It was written to defend the
exact hazard the plan gate had named: the segment loop must not re-enter the
command namespace. Mutating the namespace order — history tried before commands —
left it green, because with that history both orders returned the same answer.

The fix was history that only ONE order can produce: `sevenfold` in the deck and
`/history seven` typed. Correct code offers nothing; the mutant offers `fold`.

- **"I wrote a test for that finding" is a claim about the test's data, not about
  the assertion.** The assertion can be perfect and still never run over a state
  where the implementations differ.
- **Mutate along the axis the finding named, not just any axis.** Four mutations
  of the floor and the markers all died here while the ordering mutant lived.
  Killing mutants elsewhere in the file says nothing about this one.
- **The same defect recurred on a second axis of the same commit, after this
  lesson was written.** `TestWholeLineBeatsAnInnerSegment` used
  `hist("island", "hot dog and fries")` against the line `hot dog`: `island` is
  inert, so segment-precedence — the rule the Spec, the doc comment and the atlas
  all state — had zero coverage, and my mutation "check" of it passed because I
  mutated `trailingSegments`' output order (which the table test catches) rather
  than the ITERATION order in `historyCompletions` (which nothing caught). Two
  rules fall out: **write the mutant at the site that implements the rule, not at
  a site the rule flows through**, and **when a fixture has two entries, check
  that BOTH can match** — a decorative entry is how a discriminating test quietly
  becomes a tautology.

## Re-read the log you are citing; do not cite it from memory (define #20)

Deriving #20's estimate, I priced review rounds below the only measured figure
and justified it: "#17 came in under, three boundary rounds included, measured
~3.8h." #17's log — written by me the day before — says the review cost "is still
unmeasured for this issue", that M1 had not been through even one round at the
time of writing, and that 3.8h was hand-recorded as a wall-clock upper bound
after `sdlc actual` returned an impossible value. Every clause of my citation was
wrong, and it moved the estimate in the wrong direction.

- **A remembered number loses its qualifiers first.** "3.8h" survived; "not a
  measurement", "feature work only", "reviews not yet run" did not. Those
  qualifiers were the entire reason the number existed.
- **When a past issue is your evidence, open it.** The cost is one `sed -n`. The
  cost of not doing it is an estimate that pollutes calibration while carrying a
  citation that makes it look grounded.
- **Check which direction the repo's drift actually runs before correcting for
  it.** The ledger had six of seven `tools` rows under 1.0 — systematic
  under-estimation — and the nearest analogue (#15, the same functions) at 0.27×.
  I was correcting downward.

## An enumeration written from memory is not a sweep (#6, three times)

BR-3, BR-29 and BR-42 are one rule found three times: I wrote "these four
artifacts claim X", "three pinnable, three unpinnable", and each time built the
list from memory of what I had just done instead of from the diff. Each time the
list was missing an item that was in the commit.

**Rule:** when a note enumerates sites ("N files say X", "these are the ones
covered"), build the list by RUNNING something — `grep`, `go tool cover`,
`git show --stat` — and paste what it returned. If a count appears in prose,
the command that produced it belongs next to it. A number typed from memory is
a claim, and it has been wrong every time it has been checked here.

Corollary: a count that varies by context ("three purity guards") should be
stated per-context, not once. `schedule` takes three guards and `play` takes
two; a single sentence about "the guards" was wrong in whichever place it was
copied to second.

**BR-44 is the fourth instance, and it happened in the round that wrote this
rule down.** Un-ticking #6's milestone rows meant correcting three artifacts —
issue, plan, project — and I did two. The project file still carried a close
date and a hand-typed 0.8h for the boundary the other two now said had never
closed, which would have double-counted against the measured actual. The
enumeration was one grep long: `grep -rn "tools#6" workshop/projects/`.

So the rule needs its trigger sharpened. It is not only "when a note enumerates
sites"; it is **whenever a fact changes, grep for the fact before editing, and
edit from what the grep returned.** The failure mode is not forgetting that
other copies exist — it is remembering two of them and never asking.

## A test double must defend itself against the obvious alternative (#6, BR-43)

`missingDict` duplicates something `fakeDictionary` can already do, and the
review flagged it. It was justified — using a corpus-absent word would make the
test depend on a fixture's contents rather than on the behaviour it names, which
is the fault the `-count`/`obsequious` test already had — but NONE of that was
written down, so the duplication looked unexamined.

**Rule:** when adding a double next to one that nearly fits, the comment says
why the near-fit was rejected. If it cannot, use the existing one.

## Back up with git, not with cp to /tmp (#6)

`cp x /tmp/x.bak` before a revert-measurement failed silently under the sandbox
(`/tmp` is not writable; the scratchpad is), leaving the mutation in the tree
with no backup.

Evidence only — the RULE for this lives in one place, *Mutation testing needs a
COMMITTED baseline*. Do not restate it here; this entry is the third of five
occurrences and the restatements are what made them contradict.

## A live conformance check that is never run is not a check (#6, BR-45)

`--play` shipped as the newest raw-terminal surface with no pty conformance
test, though `startDefine` and five sibling conformance files already existed.
The one defect it shipped was the CRLF cascade — and it was found by the
operator on a real terminal, not by the suite.

Worse, when the `--play` test was finally written, running the whole pty suite
showed `TestPTYSuggestionAndAcceptance` had been RED since #21. It asserts on
raw frame bytes, and #21's deck-word highlighting inserts an SGR sequence inside
the typed line, so `"what is a sycophantic"` no longer matched as a substring
even though Tab had accepted correctly. **Both #20 and #21 merged with it red**,
because the suite is behind `-tags conformance` and neither close ran it.

**Rules:**
1. A new raw-terminal, network, or external-binary surface gets a conformance
   test in the SAME milestone. The harness existing is not the same as it being
   used, and "smoke-tested by hand" is the scratch-verify pattern.
2. Run the on-demand suites at a close, not only the default `go test ./...`.
   An opt-in suite decays silently — it reports nothing while it is failing.
3. Assert on TEXT with styling stripped, unless styling is the subject. A raw
   byte assertion is a test of the renderer's current escape sequences, and any
   feature that adds a colour will break it without a behaviour changing.

## "Nothing due today" named the wrong cause three times (#6, BR-46)

One message served an empty deck, a zero budget, and a genuinely clear schedule,
so `--play -count 0` told the learner their deck was clear while every word in
it was outstanding — the learner's own input handed back wearing the schedule's
clothes. The existing test asserted "nothing due" for a deck with NO WORDS, so
it encoded the conflation rather than catching it.

**Rule:** when a result is empty, the message names WHICH cause produced it, and
the reassuring sentence is reserved for the reassuring case. A message shared by
an error path and a success path will be read as the success.

## The claim → test map, and the check that makes it real (#6, BR-48)

BR-48 was the SIXTH finding in `claim-without-failing-test` and the rule behind
it had never actually been executed: *a claim in an artifact is complete only
when a NAMED test fails without it, and the map from claim to test is written
where the claim lives.*

Two things made this the sharpest finding of the issue.

**The commit that closed one unpinned fix shipped another.** `-count must not be
negative`, added to fix BR-46, went in at coverage 0 — so the round that closed
the family's previous instance created a new member of it. A fix is not done
because the defect is gone; it is done when something fails without it.

**A test can configure a seam the code under test cannot reach.**
`TestSessionRunsWithTheModelUnavailable` set `d.newLLM` to a failing client and
asserted the session finished — but the play path never reads `d.newLLM` at all,
so a WORKING client produced the same result. The test was byte-identical in
meaning to the one above it, and the Done-when row it stood for could not fail.
"Degrades when the model is unavailable" and "never reaches for a model" are
different claims; only a double that fails WHEN USED asserts the second.

**And then, writing the map: four of the fourteen test names I typed did not
exist.** In the artifact whose whole purpose was to close the
write-it-from-memory family. A loop over `grep -qE "func <name>\("` caught them.

**Rules:**
1. Build the enumeration by RUNNING `go tool cover` over the close window and
   reading the zero-count blocks. Never from memory of what was written.
2. Every name you write into an artifact — test, function, file, flag — gets
   grepped before the artifact is committed. The error rate on names typed from
   memory in this session was 4 in 14.
3. A seam a test configures must be READ by the code under test. Check with
   grep; an unreachable seam makes the assertion vacuous while looking rigorous.
4. State the SCOPE a mutation proved. Mutating `runPlay` left the session test
   green — correctly, since it drives `playSession`. Knowing which is which is
   the difference between a pin and a belief.

## Mutation testing needs a COMMITTED baseline (#24)

**The single home for the mutate/restore rule.** SIX occurrences — #2 round 4,
#9 M1, #16 M2, #5's close, #6 close, #24 — and it recurs because each was written
as a separate entry, together giving four different answers, so the next round
read a contradiction and picked one. #2 said copy the file aside; #6 said never
copy, use git; #24 said copy again. Add evidence here; do not append a sibling.

(The review that caught this enumerated four of the six. Grepping the headings
for `restore|backup|baseline|mutation` found the other two — the same "the
enumeration is the deliverable" rule this file states twice elsewhere.)

The rule, whole:

**Commit the implementation BEFORE the first mutation, then restore with
`git checkout HEAD -- <path>`.** Both halves, or neither works:

- `git checkout` restores to the last COMMIT. Restoring a file whose work is
  uncommitted erases that work — #5's close lost an unrelated `LastBox` fix this
  way, and #24 lost a whole implementation (a new field, a new function, two
  rewritten switch arms) with the tests still passing against nothing.
- A `cp` snapshot is not the escape. It goes stale (#9 M1: a backup taken at the
  start of a milestone deleted `bothSources` on copy-back) and it can snapshot a
  tree that is already corrupt (#16 M2: a killed background job left the file
  mutated, and the next command snapshotted THAT as "known good", disabling a
  mechanism while printing `tree restored, builds`). It also fails silently under
  the sandbox, where `/tmp` is not writable.

**The mechanism, so "commit first" does not mean polluting the branch** (#6
close, where seven `wip` commits reached a review — one of them 507 lines):

- The file you mutate must be versioned AT THE MOMENT you mutate it, so
  `git checkout HEAD -- <path>` is an exact restore.
- Getting there is a real commit, and a noisy one is squashed before the
  boundary, not avoided: `GIT_SEQUENCE_EDITOR="sed -E 's/^pick (sha1|sha2)/fixup \1/'" git rebase -i <base>`,
  with a backup branch first and `git diff backup --stat` after to prove the tree
  is unchanged.
- `git stash` protects UNRELATED uncommitted work in the same file from a stray
  checkout — stash, mutate a committed file, restore, pop. It is not a way to
  mutate uncommitted work, since stashing removes the very thing under test.

So the fix for "I cannot `git checkout`, my work is uncommitted" is **commit**,
not a scratch copy. #24's close round reached for the copy instead, appended a
lesson saying so, and thereby contradicted three existing entries — which is the
finding that produced this consolidation. *The rule already existed; a sibling
got written instead of read.*

**Before appending a lesson, grep this file for the rule you are about to state.
If it is here, REVISE that entry with the new evidence and resolve any
contradiction inside it.** A rules file read at session start hands the next
agent every version it contains.

**Corollary — a mutation that reddens nothing has two explanations, and the
likelier one is that it did not apply.** Two of #24's nine reported "0 tests
reddened" and both were failed string replacements, not weak tests; two more in
the close round reported a clean pass after a restore had reverted the code under
test, with the guard's `AssertionError: target missing` printing into the same
output. Assert the target text is present before rewriting it, re-verify after
restoring, and **read the FAILURE, not the count** — enumerate which tests fired.
An empty result from a check that should have failed is itself a finding.

**Corollary — a count is not a measurement when the suite is
environment-dependent.** #24's strict-mode grep count held at 11 across a fix
that genuinely added two sites, because network reachability differs between
runs. Enumerate what fired, by name.

## A prompt that names its keys is a state machine's public surface (#24)

`--play` asked "Enter or space to reveal" and only offered `y`/`n` afterwards, so
every correct answer cost a keystroke that carried no information — and the slow
one, since a reveal fetches and plays audio. The session refused to grade an
unrevealed word and argued it in a comment: *"a learner cannot rate what they
have not seen."*

That is true of a RECOGNITION test and false of a RECALL test. The learner rates
their own recall, which they know before checking; the definition is FEEDBACK,
not stimulus. The comment was confident, load-bearing, and had been read past
several times.

**Rule:** when a comment justifies a restriction with a claim about the user,
check the claim against what the feature actually tests. A plausible sentence
next to the code that implements it is the easiest kind of wrong to preserve.

## Prove the file list, not just the pattern (#24 BR-1)

#6 produced the rule "build the doc-sweep list by running a grep, and prove the
pattern reaches known-stale sites first". #24 did exactly that — and still
shipped stale docs, because the proof covered the PATTERN and the bug was in the
FILE LIST: the sweep passed `cmd/define/*.go`, and the form's own doc comments
live in `cmd/define/play/`. A glob is not a tree.

Same shape as #6 BR-44 (a correction reaching two artifacts of three) and #6
BR-48 (four of fourteen names typed from memory). Three issues, three variants,
one rule: **the enumeration is the deliverable, and every part of it — pattern,
paths, names — has to be produced by something that ran.**

**Rule:** sweep with `grep -r` over DIRECTORIES, never a `*.go` glob, and put the
excludes in explicitly (`| grep -v _test`) so what is left out is visible rather
than accidental.

## A skip reads as green (#24, four rounds)

The pty conformance suite skipped when no terminal was available, so a run
without one reported success for a suite that never executed — a suite that
exists because #6's CRLF defect was invisible to everything that WAS running.
`CONFORMANCE_STRICT=1` turns that skip into a failure.

**Rule:** any test that can skip itself needs a mode where the skip is an error,
or "green" silently means "did not run".

**This rule then took four review rounds to actually land, and the interest is in
HOW each round failed** — every one of them applied the rule correctly to the
sites its enumeration reached, and the enumeration was wrong in a new way each
time:

1. Fixed the one pty site. Six suites kept skipping silently.
2. Routed all seven, enumerated by `grep 't\.Skipf\?('` — which cannot see the
   MIRROR defect. Three suites wrote an absent dependency as an unconditional
   `Fatalf`, so the offline suite was red rather than skipped.
3. The carve-out excluded a file by NAME. `render_test.go` skipped on an absent
   COMMITTED fixture, silently retiring coverage while the package reported ok.
4. The sweep covered `cmd/define` while the README it added claimed `./...`.
   Measured: the documented strict command reported `ok` with `internal/llm`'s
   four suites skipped — a green that meant nothing, which is the precise false
   assurance the rule exists to remove.

**Rule:** a "green means it ran" guarantee is a claim about an ENUMERATION, and
its claimed scope may not exceed its swept scope. Four rounds of re-running a
sweep by hand is the signal to stop sweeping: the fix is a meta-test that walks
the tree and FAILS on any unrouted skip
(`internal/conformance.TestEverySkipIsRoutedOrWaived`). Same move as
*A restatement drifts; a consumer fails the build* — a grep cannot fail a build.

**Corollary — ask the site the question; do not grep the spelling.** "What does
this do when its dependency is missing?" sorts every site into four classes, and
only the first is a skip: absent EXTERNAL dependency (skip; fail under strict),
absent IN-REPO artifact (always fail — a committed file that is gone is a deleted
file), SHAPE drift (always fail — it is what the check is FOR), inapplicable
table row (skip, permanently, marked with a reason at the site).

## The same rule fails in two directions, and a grep sees one (#24 BR-9)

"A skip reads as green" (above) was fixed at one pty site, then generalised: route
every conformance skip through one `skipOrFail` helper, enumerated with
`grep -rn 't\.Skipf\?(' cmd/define/*_test.go`. Seven sites, all routed, rule
applied to the class rather than the instance — the lesson from the round before,
correctly learned.

It still missed three sites. `dict_conformance`, `news_conformance` and
`live_property` wrote an absent dependency as an unconditional `t.Fatalf`, so the
NON-strict suite could never be green offline — the mirror of the same bug, and
invisible to a grep for the word "Skip". It surfaced only because a full offline
run came back `FAIL` and the number was chased instead of shrugged at.

The enumeration lesson keeps getting sharper: #6 said prove the pattern, BR-1 said
prove the file list, this says **the pattern encodes an assumption about how the
bug is spelled**. A grep finds instances of a SHAPE; a class of bug is a
QUESTION. Ask each site the question — "what does this do when its dependency is
missing?" — and read the answer.

**Rule:** when a fix is applied to a class, enumerate by asking every member the
question the rule is about, not by grepping the spelling the first instance used.
If the rule has two failure directions, one grep sees one of them.

## A restatement drifts; a consumer fails the build (#24 BR-10)

Three findings in the `doc-sweep-incomplete` family landed on one screen of
`--play`: the reversal reached README and atlas but not the form's doc comments,
then the doc comments but not two test citations, then the README's audio
sentence still described a fetch a `y` no longer performs. Each was found by a
human re-reading prose, and each was fixed by another sweep — so the next edit
restarts the cycle.

The prompt lines are now `const gradePrompt` / `gradedPrompt` in `play_loop.go`,
and `TestREADMEQuotesThePromptsTheLoopActuallyPrints` asserts README.md contains
them verbatim. It failed on its first run, catching a paraphrase the three
preceding sweeps had all read past. A grep cannot fail a build; a test can.

**Rule:** when a doc restates a fact the code owns, and it has drifted twice,
stop sweeping and make the doc a CONSUMER — a test that reads the doc and asserts
the code's literal. Pin what the user reads off the screen and types against, not
the surrounding prose, which should stay free to be rewritten.

## A line number in a comment is a restatement too (#24 BR-1)

`play_loop_test.go` cited "play_loop.go:182 — space reveals" and "README:54". The
reversal moved both: :182 became a different case, and README:54 became a table
header. Same drift as the prose in the family above, with nothing to catch it —
a comment cannot be wrong enough to fail a build, and a citation that is *almost*
right is worse than none, because it reads as verified.

**Rule:** cite by NAME — the function, the const, the table — never by line
number. Names move with the thing they name.

## Enumerate the derived set, or you will sweep it by hand and miss one (#23 C1, BR-2)

`/lang` had to re-derive everything downstream of the language. I found the
members one at a time — the deck, the capturer, the vocabulary, and the highlight
set the raw editor caches in a local — swept them at the call site, and never
wrote the list down anywhere. Two of the members were missing.

`opt.voice` shipped un-re-derived, so a mid-session `/lang es` left the fetch loop
asking for the four **English** URLs, including the two legacy ones the same
range had just gated to English for costing ~450 ms per guaranteed miss. The deck
went Spanish and the pronunciation did not. Then `user-model.md` — derived from
the language-scoped deck but stored flat — turned out to mean a Spanish
`--reflect` replaced the English learner model.

Every unit test was green throughout, because each member was individually
correct. The defect lived in the *set*, and a set nobody wrote down has no place
a reviewer can check it against.

**Rule:** when a change makes N things depend on one value, write the enumeration
into ONE function with the rule that generates it, and name the members that will
join it later. A hand-sweep at the call site is unreviewable — "did you get them
all" is unanswerable against a list that does not exist.

**Corollary — a derived value needs one deriving function.** `opt.voice` was
computed by one expression at the boundary and should have been computed by
another at the switch; that is not a missing call, it is two sources for one
fact. One function, two callers, and they cannot drift.

**Corollary — "not scoped" needs its own justification per artifact.** `events/`
is unscoped because an event is a fact about a moment. The learner model is a
SUMMARY OF A DECK, so that argument does not transfer — but the atlas had already
reused it, and the reused sentence read as settled. When a rule is extended to a
second artifact, re-derive it there rather than citing the first.

## Test what the pure function's CALLER does, not just the function (#23 C1)

`voiceFor` was always right. `TestLocaleFor` and `TestAudioCandidatesSpanish`
passed the whole time the bug existed, because nothing called `voiceFor` again
after a language switch. The only test that could have caught it drives `/lang es`
through `run()` and asserts what the fake CDN was **asked for**.

**Rule:** for a pure function whose value is CACHED by its caller, a unit test
pins the function and says nothing about the cache. Add one assertion at the
altitude where the cached value is consumed — and give it a non-vacuity check, or
a session that fetched nothing at all passes it.

## A rule stated in prose is enforced where you can run it (#23 BR-6, BR-11)

The artifact-name rule was written correctly the first time: *no output line,
comment, README line, atlas line or plan line may spell a name the code owns.*
It was then swept by hand, and the sweep was 3 of ~12 — including the
user-facing line, which told the learner to edit a file the tool had not
written.

Mechanising it over Go source immediately found four more sites the manual
enumeration had missed. One round later the same family returned, because the
ratchet stopped at `*.go` while the rule bound prose: the project file still
claimed a single shared learner model after it had become one per language.

**Rule:** when a review finding names a rule rather than a site, ship the
ENFORCEMENT in the same commit, and make its scope match the rule's own words.
A rule enforced over a subset of what it claims to bind reads as settled while
the unenforced half keeps drifting — and the half nobody checks is the half that
goes stale.

**Corollary — enforcement needs an explicit theory of records.** Docs that
describe the tool as it IS must be swept; docs that RECORD what was true when
written must not, or the fix is falsifying history. Identify records by shape
(`## Revisions`, `## Log`, a block carrying `**closed:**`) rather than by a list
of filenames, so a new one is covered without anyone remembering it.

## Move the test to where the paths come from (#23, round 4)

`TestYAMLIgnoresInterruptedWrites` planted its fixture at `words/.tmp-halfwritten`
from outside the package. When the deck moved to `words/<lang>/`, the fixture
stayed put, `Deck()` stopped listing it, and the test kept passing while
exercising nothing.

Worse, it had never been load-bearing: the fixture body was unparseable, so
deleting the suffix check the test exists to pin left it warned-and-skipped and
the suite green either way.

**Rule:** an external test that hardcodes an internal path will silently stop
testing when the layout moves. Put it in the package and obtain the path from the
function that produces it. And make the fixture VALID except for the one property
under test — an invalid fixture passes for whichever reason comes first, which
may not be the one the test names.

## An enumeration in a comment is not structural (#23 C1, BR-2, BR-13)

Three findings, one shape. `/lang` must re-derive everything downstream of the
language, and the set was written as a list in `applyLang`'s doc comment. It was
wrong three times: `opt.voice` at M1's boundary, the learner model one round
later, and `d.usage` at the close — the last one added by *the same milestone*
whose comment still called it "not language-scoped".

Each fix added the missing member to the list. The list kept going stale because
nothing made adding a member at the boundary imply switching it.

**Rule:** when N things derive from one value, make the set a TYPE built by ONE
function that both the construction site and the switch call. A struct returned
from one builder cannot be half-adopted; a comment listing the same members can,
and will, within a milestone.

**Corollary:** "deliberately NOT in this set" is a claim with a shelf life. Every
exclusion needs its reason re-checked when the thing it excludes changes — the
`usage` exclusion was true when written and false three commits later.

## Give a count one producer, or delete the count (#23 BR-14)

"The nine symbols" appeared in four documents and was wrong in all four: nine is
how many the *survey* found, while the resolver needs three. Nothing checked it,
because a number in prose has no consumer.

**Rule:** a count restated in prose is drift waiting to happen. Either give it one
producer the docs derive from, or — usually better — remove the number and name
the list, so there is nothing to go stale. `dcsPrivateSymbols` is the list; no
document counts it.

## Make the plan a consumer of the tree (#23 BR-14)

Plans named `deckDeps`, `MigrateFlatDeck`, `dictChoice` and `dcsDictionaries` —
four entities the tree did not have, across four review rounds, each fixed by
hand-sweeping the instance.

A plan's Core-concepts table already states "this identifier lives at this path"
in machine-readable form. `TestPlanTablesNameEntitiesThatExist` reads it and
checks the file declares the name.

**Rule:** when a document restates a fact the code owns in a STRUCTURED form,
make the document a consumer. The unstructured half stays a review problem; the
structured half becomes a build failure. A row whose file does not exist yet is
skipped — a plan precedes its code, and only a row pointing at a real file makes
a checkable claim.

## Pure code behind a build tag is not pure enough (#23 BR-23)

Three parsers carried doc comments saying "pure, so the cgo boundary's format is
testable without CoreServices" — while sitting behind `//go:build darwin`. Their
test was untagged, so `GOOS=linux go vet` failed on them, and `dict_stub.go`
claimed in its own comment to keep exactly that green.

Worse, a fix shipped INOPERATIVE in the same file: the status-folding rule
("an absence never overwrites a real failure") was corrected in a commit, and the
correction did nothing, because the branch it needed to guard was still
unconditional and nothing could test it where it lived.

**Rule:** if a function's doc says it is pure and testable off-platform, it must
COMPILE off-platform — move it out of the tagged file. Untestable code is where a
fix can look right in review and do nothing at runtime; "I extracted the policy"
is only true once the policy has a test that fails without it.

## A struct is not enough if both sides spell out its fields (#23 BR-22)

The language-derived set went from a doc comment to a `langDeps` struct, and the
next review found the same class again: `openStore` and `applyLang` each copied
its four fields by hand, so a fifth field was still forgettable in two places.

Embedding the struct is what finally fixed it — adoption became one assignment,
and a member added later is adopted with no edit at either site.

**Rule:** "make it a type" is half the fix. Ask where the type is CONSUMED: if
every consumer enumerates its fields, the type is documentation and the
enumeration is still the real interface. Embed it, or give it one adoption
method, so adding a member cannot be half done.

## Never let a test reimplement the code it pins (#23 BR-22b)

A new test for "the dictionary is built for the session's language" replicated
the boundary derivation inline instead of calling `run()`. Deleting that
derivation from production left the test green — it was asserting against its own
copy.

**Rule:** a test that reproduces the production wiring tests the reproduction.
Drive the real entry point, and verify by DELETING the production line and
watching the test fail. If it does not fail, the test is pinning a copy.

## When a corpus gains a dimension, every check over it gains one (#23 BR-31)

M2 made the fixture corpus per-language and added five real Spanish captures.
Every check over it kept naming English: `loadFakeDictionary(…, DefaultLang)` at
two sites, and the no-data-loss invariant through `testDict`, which is English by
definition. So the new half was byte-compared to nothing and never parsed — while
the atlas and the plan both described the corpus as conformance-checked.

**Rule:** a check that spells one value of a dimension is blind to the rest of it.
Read the SET from the artifact — here, the language directories — so adding a
member brings it under every check with no edit. And when adding the dimension,
grep the checks for the old constant: each surviving mention is a check that
silently narrowed.

## A guard is code, and gets the same scrutiny (#23 BR-20, BR-32)

Three guards written to close review families each shipped with a hole a review
found: one passed on a COMMENT mention, one read only the first name in a row
naming three, and two more had unrouted skips that would report green for checks
that never ran.

**Rule:** the test you add to stop a class of bug is not exempt from that class.
Before trusting a new guard, mutate the thing it claims to catch and watch it
fail — and check its own skips, its own scope, and whether a weaker match
satisfies it.

## A branch that tests position is a catch-all wearing a signature (#26 BR-1, BR-14)

`classifyRawNotation` was written to make a drifting count attributable. Its
first version had two catch-all branches, so every input the oracle passed it
landed in a named bucket BY CONSTRUCTION — the live assertion
`unclassified == 0` could not fail, and I reported its passing as evidence the
taxonomy was total. A new shape would have been absorbed into whichever bucket
it resembled: the exact failure the classifier existed to prevent, built as its
opposite.

Fixing three of four branches was not enough. The fourth tested `i < 128` —
POSITION, not shape — and so absorbed every polysyllabic form of the very shape
the issue was filed about. The test that was supposed to catch this passed
because its fixture was a monosyllable with no stress mark, reaching the residue
through a different branch entirely.

**Rule:** every branch of a classifier needs a POSITIVE signature of the thing it
names. A predicate testing position, length, or "everything else" is a catch-all,
and a catch-all makes the residue unreachable — which makes any assertion about
the residue vacuous. Ask of each branch: *what input would this refuse?* If the
answer is "nothing that got this far", it is not a test.

**Corollary — a residue bucket earns its keep only if it can fire.** Before
trusting "unclassified is zero", construct an input that SHOULD be unclassified
and watch it land there. That probe is the difference between a measurement and
a coincidence.

## Read the signature off the data, not off your memory of the data (#26)

Two discriminators in this classifier were guessed and both were wrong: a
position fraction (`idx < len/3`) that misclassified its own exemplar, and a
lookup into the wrong coordinate space that silently returned -1 every time
because `strayStress` windows the STRIPPED text.

Dumping the real values took one throwaway test and settled both: byte 27 of
1828 versus 2436 of 4116, and the actual shape `(aˈhəndrədzˈhəndrəd/)` versus
`| AmE …, BrE … |`.

**Rule:** when a predicate keys on the shape of real data, print the real data
first. A guess that happens to pass its exemplar is indistinguishable from a
correct rule until the population disagrees — and the population is where the
cost lands.

## The human half of a mechanical guard is the half that fails (#27 BR-8)

`#26` built `retiredSymbolNames` + `TestNoArtifactNamesARetiredSymbol` so a
rename sweeps every prose restatement of the old symbol. Its own comment says why
one step must stay manual: *a rename cannot be detected automatically, because
only the person doing it knows the old name.*

One issue later I renamed a test, skipped the row, and left a stale mention —
**in the same commit that widened the test being renamed.** The guard could not
fire, because the guard's input is the thing I did not supply.

**Rule:** when a mechanism has one human step, that step is where it will fail,
and "I built the guard" is not the same as "the guard is armed". Make the manual
step part of the same edit as the thing that triggers it — rename and row in one
commit — and treat a mechanism with an unsupplied input as unprotected rather
than protected.


## A structural argument is not a pin (define #29, close review round 2)

Done-when 6 was "covered" by a sentence: *`voiceFor` is unchanged, so the locale
is literally `#27`'s.* True, and worthless — rewriting the CALL SITE to
`voiceFor(pron, "")` left the whole suite green. **A reused function cannot vouch
for a new caller.** The reuse is real; the wiring to it is new code and can be
wrong without touching the thing reused.

The rule, and it is decidable at plan time: **every Done-when coverage cell names
a `Test…` symbol, and that test is OBSERVED red with the wiring removed.** A cell
that can only name a task, a decision, or a design property is a cell with no
pin, and should say so rather than implying one.

Measured on that issue: of six Done-whens, the two whose cells named a task or a
decision instead of a test were *exactly* the two that turned out unpinned. Two
for two, found one review round apart.

## "unchanged" is a claim about the diff, and git already knows (define #29)

A plan's Core-concepts status column says `new`/`modified`/`unchanged`. Those are
not opinions about behaviour — they say whether this window touched the symbol,
which is mechanically checkable. Four of twenty rows were wrong across two review
rounds, and the round that fixed them BY HAND caught three of four: the fourth
had been sitting there the whole time claiming "unchanged" about a symbol the
same plan's own doc sweep rewrote.

Two details decide whether the check is usable:

- **Declaration level, not file level.** `voice.go` changed while `voiceFor`,
  `localeFor`, `defaultLocale` and `applyVoice` did not, and that row was
  correct. A file-level check condemns it.
- **The doc comment is part of the declaration.** A symbol whose comment this
  window rewrote is not "unchanged" to the reader the plan is written for.

And when the mechanism landed it immediately caught two rows the *previous* round
had added by hand — both naming a TYPE while what changed was its method.

## A heuristic that fails in both directions wants a vocabulary (define #29)

The same status cell got two heuristics. `Contains(lower(cell), "new")` also
matched "renewed". The fix — first word equals "new" — then missed `**new**`, and
bold cells are this repo's live convention. Two failures in opposite directions
is the signal to stop guessing at the shape: it is a **controlled vocabulary**
(`new`/`modified`/`unchanged`/`deleted`), so normalise the cell and match the
set, and **fail loudly on anything outside it** rather than letting an
unrecognised value fall into whichever branch the heuristic happens to pick.

## A negative check must cover the whole thing it claims (define #29)

Three live conformance rows pinned claims about a WALK — "Italian is absent",
"French coverage is partial", "`jalapeno_es_es` is a 404" — by probing
`AudioCandidates(word, voice)[0]`. One URL. A recording appearing only at the
`_2` suffix would leave every row green while the fallback quietly stopped
firing, and the row would be pinning a strictly smaller claim than its name.

Assert through the production shape — `Fetch(ctx, AudioCandidates(…))` returning
`ErrNoAudio` — so the check and the code walk the same list.

## A doc that ENUMERATES something must derive from it (define #29)

The atlas listed three of five commands, and the two missing were the two most
recently added — `/lang` and `/pron`, two for two. Every author added a registry
row and did not know the table existed. Prose *about* a mechanism can be written
by hand; a prose *enumeration of its members* cannot, because the members grow
and the prose does not. Generate it from the registry and pin it, exactly as
`localeHelp`/`pronHelp` are pinned.

## Mutation testing, two more ways to get a false reading (define #29)

Both happened in one session, on top of the four already recorded:

- **Mutating BEFORE the thing that overwrites it.** A mutation setting
  `opt.voice` was placed above `applyVoice`, which recomputes it — so the suite
  stayed green and read as "this assertion is blind". The assertion was fine; the
  mutation never survived to the code under test. Put it where the value is
  actually read.
- **`git checkout <file>` to revert, on an UNCOMMITTED baseline.** This is
  already in this file, and it happened again anyway — twice more in the same
  session, the second time discarding ~150 lines of guard work written minutes
  earlier, and a FOURTH an hour after that entry was rewritten — which is the
  real finding: restating the rule did not change the behaviour. **Four
  occurrences in one session, twice after writing the rule down** — and then a
  FIFTH, after the "mechanical trigger" sentence was added. Five is enough to
  say the trigger was still the wrong shape: it asked me to remember to run a
  check, which is the same class of thing as remembering not to type the
  command.
  **What actually worked: stop using `git checkout` to revert a mutation at
  all.** `cp f "$TMPDIR/f.bak"` before, `cp "$TMPDIR/f.bak" f` after. It restores
  the file to what it was rather than to what was committed, which is the
  property the task needs and the one `git checkout` does not have at any point.
  The underlying framing that keeps failing is:
  *`git checkout` is not a revert tool — it is a "discard everything since the
  last commit in this file" tool.* Before typing it, `git status --short` the
  file. Better: commit, THEN mutate, and treat "I want to mutate an uncommitted
  file" as the signal to commit first, not as a thing to be careful about.

## Run the chain against the real dependency before believing the probe (define #29)

Nine CDN probes said `rôle_fr_fr` is a 200 and `role_fr_fr` a 404, so "the
accented spelling wins" went into three files as a 9-of-9 rule. Running the
finished pipeline against the live dictionary showed `SourceSpellings("role", …)`
returns `["role"]`: NOAD heads the entry `role`, has no `(also rôle)`, and spells
the accented form only inside ORIGIN prose. The probe measured the CDN correctly
and said nothing about whether the code could ever *reach* that URL.

**A probe of a dependency is not a test of the path to it.** The rule is 8 of 8,
and `role` is a recorded limitation.

## A fake that diverges from its dependency blocks the test you need (define #29)

Three doubles were wrong in ways that had been harmless until this issue made
them load-bearing, and each one blocked the end-to-end test rather than merely
being imprecise:

- `fakeDictionary` was keyed by exact spelling while NOAD is accent-insensitive.
  With it unfixed, the test would have had to type `jalapeño` — where typed and
  headword agree and the entire mechanism goes unexercised. **A fake that cannot
  represent the case the feature is FOR turns the test into a tautology.**
- `fakeCDN` keyed on `r.URL.Path`, which Go percent-DECODES, so every URL with a
  non-ASCII character 404'd there while the real CDN serves it.
- `rebasedSource` returned its own rewritten URL as "the one that answered" —
  harmless while the value was discarded, wrong the moment it was read.

The pattern: a double's divergence is invisible until a feature depends on the
part that diverges. When a new feature makes a previously-ignored value
load-bearing, **check what the doubles do with that value first.**

## Run the suite the checklist names, not the one you have been running (define #29)

Three close-review rounds ran on `go test ./cmd/define`. The plan's own
`## Verification before close` says `go test ./...`, and when the reviewer ran
that it was **RED** — a repo-wide guard (`TestEverySkipIsRoutedOrWaived`) had
been failing since the previous round on two `t.Skip` sites added in the commit
that answered it.

Two things generalise:

- **A narrower run is not weaker evidence, it is DIFFERENT evidence.** Package
  tests cannot see a guard that walks the whole tree, and the guards most worth
  having are exactly the tree-walking ones.
- **Evidence has a timestamp.** The `--verified` text was measured before the
  last two commits, which is the same class as a commit message asserting a
  deletion: re-run on the FINAL head, after the last fix, not after the round
  the fixes answered.

Corollary found the hard way: `internal/conformance`'s waiver marker must sit on
the skip line or within **three lines above it**. A four-line comment block whose
first line carries the marker does not count, and the failure message does not
say so.

## A mutation that changes two things proves nothing about either (define #35)

To show a word-boundary regex was load-bearing I removed the boundary AND
deleted the redundant mask entry in the same edit, saw five subtests redden, and
wrote "dropping the boundary reddens five subtests" into a comment and an issue
Log. The close review measured the truth: with only the regex changed, **nothing
reddened** — the mask was holding those rows green the whole time.

Two mechanisms that both prevent the same failure are **mutually redundant**, and
each hides the other's removal. The tell is that the case you are testing with is
covered by both. The fix is a case only ONE of them can save: `"a town in
Germany"` is in no mask list, so only `\b` stops it matching `German`.

**Rule: one mutation, one change.** If the thing you removed is not the only
thing preventing the failure, you have measured the pair, not the part.

## A test column nobody asserts is documentation (define #35)

`TestPronReportsWhyItCannotInfer` had a `{word, because}` table and a doc comment
saying "it declines with the reason, on the two shapes that differ" — and the
body only checked that stderr mentioned the command. `because` was never read.

The cost was not hypothetical: it let a wrong message ship. An entry naming no
language at all was told its ORIGIN "names only historical stages or cognates",
which is a record that is not true — the exact property the surrounding design
insists on. Asserting the column turned it red immediately.

**Before adding a field to a test table, grep the body for it.** An unread field
reads as coverage in every review and provides none.

## Enumerate the category, not the instances you happened to meet (define #35)

A mask list held `Old French`, `Old English` and `Middle Dutch` — the stages the
corpus happened to contain — under a decision that promised to exclude
superseded stages *as a category*. `Old Italian`, `Middle French`, `Old Spanish`
and `Low German` all sailed through, each inferring a modern recording for a dead
stage and each REPORTING the modern language, so the record was wrong too.

When a rule says "category", derive the members from whatever defines the
category. Here every mapped language generates its own stages by prefix, and the
hand list shrinks to the stages with no modern member to generate from — which is
also the only part a reader has to check.

## The plan-table guards are answered by the COMMIT (define #30)

`TestPlanTableStatusMatchesTheChangeWindow` compares a plan's `modified` /
`unchanged` claims against `git diff <base>..HEAD` — **committed** changes. So a
plan row and a code edit that are both in the working tree pass, and the same
tree fails the moment it becomes a commit. Running the suite before committing
proves nothing about these guards.

Twice in one issue this produced a Critical review finding: "`go test ./...` is
red at HEAD and the Log records it green." Both times the suite HAD been run —
one commit too early.

The rule: after editing a plan's tables, commit, then run the suite; amend if it
reddens. More generally, a guard that reads git history has to be run against the
history, not the tree.

## A docs guard must look for what only THAT documentation would contain (define #30)

Two guards were written to stop the atlas lagging new surface, and both were
vacuous for the exact thing they were written for. `TestAtlasDescribesEveryRenderOpt`
accepted a bare `` `Word` ``; `TestAtlasDescribesEveryRegionKind` searched for
`k.String()`, and "headword" occurs in that atlas nineteen times for unrelated
reasons. Deleting the whole section they defended left both green.

A prose word is not evidence that something was documented — it is evidence that
English was used. A **qualified Go identifier** is: `RegionHeadword` appears
where someone meant that kind. Where the two differ, keep both and say why —
`String()` names a thing for a reader, `identifier()` is what a check can look
for.

The wider rule this issue kept re-learning: **writing a guard is not the same as
checking the guard can fail.** Delete the thing it defends and watch it go red,
in the same sitting you write it.

## Verify by exit status, not by grepping output (define #30)

`go test ./... 2>&1 | grep -v "^ok" | head -3 && git commit` commits on a RED
suite: the pipeline's status is `head`'s, which is 0. This shipped a commit with
five failing tests, twice in one session.

`go test ./... >/dev/null 2>&1; echo $?` — or just let the command fail. A check
whose result you read with your eyes is a check that passes whenever you are
tired.

## A shortcut must not RE-DERIVE its target (define #30)

Clicking a headword is a shortcut for the bare Enter beside it. Enter replays the
session's current word — the lookup key — while the click derived its target from
the entry, `Entry.Headword()`, which is `fields[0]` alone. So `hot dog` played
"hot", `a priori` reduced to the letter "a", and `bargainer` — an inflected form
finding its base entry, the COMMON case — played "bargain".

Two gestures that mean one thing must read one source. When the shortcut cannot
reach that source, pass it: the key belongs to the caller, so it travels on
`RenderOpts.Word` rather than being guessed from what the callee happens to hold.

## Tests that only run when someone types a flag defend nothing (define #30)

The repo has fifteen fuzz targets written with real care — one found two genuine
decoder defects the day it was run by hand. A Critical panic that the fuzzer
finds in **under a second** still shipped through eleven review rounds, because
`go test ./...` exercises a fuzz target against its seed corpus only and nothing
anywhere passes `-fuzz`. The twelve pty rows have the same shape: correctly
written to skip when no pty exists, and therefore silently uncertified wherever
that is true — including inside a boundary review.

The tests were not missing. The schedule was. Filed as `#37`.

## A test that asserts what the implementation's comment claims pins the assumption (define, side-quest 2026-08-30)

`AudioCandidates` carried the comment *"Multi-word headwords are spelled with
underscores on the CDN"* and built `hot_dog_en_us_1.mp3`. The unit test asserted
`hot_dog_en_us_1.mp3`. Both were written in the same sitting, from the same
guess, and the server was never asked. `hot_dog` is a 404; the real key is
`hotdog`. Every multi-word headword in the corpus — `hot dog`, `a priori` — had
missed for the life of the feature.

Three separate things kept it invisible, and each is reusable:

1. **The test's source was the code, not the system.** For an EXTERNAL contract,
   a unit test can only pin what someone already believed. It cannot discover
   that the belief is wrong, so it converts a guess into a regression guard
   pointing the wrong way. The rule is not "don't unit-test the builder" — it is
   that a unit test asserting an external key MUST cite a live measurement, and
   name the conformance row that re-checks it.

2. **The sibling assertion looked like coverage and discriminated nothing.** The
   test also checked "must not produce a URL with a space" — satisfied by `_`,
   by `-`, and by the correct answer alike. A negative that every candidate
   passes is not evidence; it is a row that makes the file look tested. Same
   class as *A negative check must cover the whole thing it claims* (#29), met
   here in the weaker form where the check is real but its alternatives are all
   equivalent.

3. **A miss was a SUPPORTED outcome on that path**, so the bug had no symptom.
   Nothing distinguished "the CDN has no recording for this word" — routine, and
   genuinely true for most phrases — from "we asked for a filename that cannot
   exist". When absence is a legal answer, a wrong request is indistinguishable
   from a correct one, and only a positive control finds it: a word the server
   IS known to serve, asked for through the production path.

The fix pins the rule live, and asserts the negative too (`hot_dog` and `hot-dog`
must stay 404). Asserting only that `hotdog` answers would stay green if the CDN
started accepting several spellings, and the day it narrowed again would be the
day phrases broke with nothing to say why.

**Found by a user question, not by the suite** — *"'ne plus ultra' no
pronunciation found, can you figure out if you can find it?"*. The reported word
turned out to have no recording under any spelling, so the original report was
not a bug at all; the bug was three feet to the left, and only surfaced because
the first move was to probe a KNOWN-GOOD control rather than the reported word.
When a miss is reported on a path where missing is normal, establish that the
path works at all before investigating the input.

## An unticked checkbox can silently DISABLE a repo guard (define #7, close round 4)

`TestPlanTablesNameEntitiesThatExist` exempts rows whose status is `new` while
the plan still has any `- [ ] ` line — reasonably, since a plan under
construction names entities that do not exist yet. `#7` ticked the tasks in the
ISSUE and not in the PLAN DOC, so `inProgress` stayed true through four close
review rounds and every `new` row went unchecked. The table drifted three times
— `shuffleOptions` deleted, `entryDefines` added, five renames — and a reviewer
found it each round while the guard written for exactly that sat idle.

Two things to carry:

1. **A conditional exemption is a switch, and something has to turn it off.**
   "Skip while in progress" is right, but nothing made "no longer in progress"
   happen — closing ticks the issue, and the guard reads the plan. The moment
   the plan's tasks were ticked the guard fired immediately and correctly.
   **Tick the plan document's own task list at close, not just the issue's.**

2. **Reviewers found what the guard would have.** Three rounds of
   `plan-artifact-must-match-tree` findings were a human doing, by hand and
   imperfectly, a job already automated and switched off. When a finding names a
   class the repo already guards, the first question is not "how do I fix the
   instances" but "why did the guard not fire" — the answer is worth more than
   the fix.

**And the sibling guard was simply broken.** `TestPlanNamedTestsExist` globbed
`cmd/define/*_test.go` — flat — so a plan pinning a test in `play/` was told it
does not exist. `#7` tripped it with six at once, because form 2.3's selection is
pure and its tests live in `play/` BY DESIGN. A guard that fails on the
arrangement the architecture asks for trains people to weaken the guard, so it
was fixed to walk the tree rather than the plan being edited to appease it.

## Three Criticals, one missing distinction: a deck holds KEYS, a dictionary holds ENTRIES (define #7)

`#7`'s close took six review rounds. Three of them raised a Critical that looked
new each time and was the same defect:

| round | symptom | what I added |
|---|---|---|
| 4 | one entry supplied two options under two spellings | dedup on `Word` |
| 5 | two options carried byte-identical glosses | dedup on `Gloss` |
| 6 | one entry supplied two options with DIFFERENT senses, both defining the prompted word, one marked wrong | dedup on `Source` — the entry |

`Word` was a DECK KEY being used as if it identified a meaning. The mapping from
keys to entries is many-to-one (`jalapeño` and `jalapeno` are two keys and one
entry), so every attribute of the OPTION — its word, its text — was a proxy that
would eventually come apart from the thing that actually matters, which is the
entry.

**The tell was in my own commit messages.** Rounds 4 and 5 both claimed to have
"fixed the class". A fix that claims the class and is followed by another
instance of it did not fix the class; it fixed a bigger symptom. When a finding
recurs after a class-fix, the class was named at the wrong level — go up one.

**The fact that explains all three was already written down in this repo.** `#29`
exists precisely because the dictionary resolves several spellings to one entry;
`dict_fake_test.go` models it and `TestLiveDictionaryResolvesAnUnaccentedQuery`
pins it live. The cost of not asking "what does the rest of the codebase already
know about identity here?" was three rounds of Critical findings.

**The question that would have short-circuited it** is not "why are these two
options the same?" but "what makes two options the same?" — the first invites a
key per symptom, the second forces you to name the unit of meaning. Ask the
second one first when deduplicating anything.

## Verify each pin against its own mutation, and name the mutation (define #7, BR-27)

I wrote two tests for one fix, ran the mutation once, watched ONE of them go red,
and recorded "both mutation-verified" in the plan. The other passed with the fix
removed entirely — its fixture could not produce the defect — so a test that
pinned nothing was on the record as a pin, which is worse than no test at all.

Two rules, and the second is the one I keep needing:

1. **A test written to pin a fix is verified against THAT fix, individually.**
   Reverting the fix and watching "the suite" go red proves only that something
   in the suite covers it.
2. **A "mutation-verified" claim in a durable artifact must NAME the mutation.**
   "Mutation-verified" is unfalsifiable prose; *"dropping `usedSource` from
   `free`, and deleting `Source:` from `optionCandidates`, each turn it red"* is
   a claim a later reader can re-run in a minute. Write the second one.

The tell that should have caught it: the fixture was chosen to illustrate the
BUG REPORT (`jalapeño`/`jalapeno`, the pair the finding named) rather than the
DEFECT CLASS (one entry with two differently-glossed senses under two deck keys).
Those coincided in the finding's prose and came apart in the corpus — the
jalapeño entry has one usable sense, so gloss-dedup already covered it. **Pick a
fixture from the class, then check the reported instance is an example of it.**

## A golden test enshrines whatever was there, bug included (define, side-quest 2026-08-31)

`potassium` rendered as a gloss of `(Symbol` with the entire definition thrown
into a quoted example. `newSense` split gloss from example on the FIRST colon
anywhere, and NOAD writes `(Symbol: K)` — a colon inside brackets.

**`minute` in the committed corpus had exactly the same defect, and
`TestRenderOutputMatchesTheCorpusGolden` was GREEN on it.** The golden was
generated from real output at a moment when the bug was already present, so it
recorded `(symbol` + `"ʹ): Delta Lyrae…"` as the expected rendering. Regenerating
it after the fix produced a 17-line diff over one entry, every line an
improvement.

The lesson is not "goldens are bad" — that golden is the only thing asserting
the rendered bytes, and it earns its place. It is that **a golden pins CHANGE,
never correctness.** It can only ever tell you the output differs from the day it
was captured; it cannot tell you the output was wrong that day. So:

- **A green golden is not evidence the output is right**, and a comment saying it
  was "generated from the commit before the change" makes it evidence about that
  commit, not about the product.
- **When a golden diff appears, read every line of it.** The diff is the only
  moment anyone looks at the bytes, so it is the only moment a pre-existing bug
  is visible. Here 17 lines took a minute to read and confirmed the fix; a
  regenerate-and-move-on would have shipped the same evidence unexamined.
- **Pair a golden with unit tests that assert PROPERTIES**, which can be wrong in
  a way a reader notices. `TestSenseSplitIgnoresColonsInsideBrackets` states what
  a colon inside brackets means; the golden only states what bytes came out.

**Found by a user on a word not in the corpus**, which is the other half: the
corpus is 34 entries chosen for the shapes someone thought to collect. The fix
reached for `delimiterDepths`, which was already in the same file doing exactly
this job for `firstSenseNumber` — so the tool existed and the second site never
got it. When adding a scanner for brackets, grep for the ones already there.

## Adopting an existing seam inherits its behaviour on inputs the previous consumer never sent it (`#41`)

`#41` made `--play` the second consumer of `#30`'s `screen`, and three of the
boundary review's Importants were the same shape: a behaviour that was correct
for the FIRST consumer, silently wrong for the second, and never examined because
"adopt the editor's seam" reads as inheriting a solved problem.

- `Paint` CLIPS a buffer line at the terminal width. Correct for a REPL, whose
  lines are all pre-wrapped by `Render`. `--play` had one line-kind that was not,
  and an unwrapped line stopped being ugly and started being missing.
- The editor gates its full-screen surface on `interactive && opt.tty`. `--play`
  gated on stdin alone, which was harmless while it emitted no escapes at all and
  a regression the moment it took the alternate screen.
- `enterMouse` costs drag-select, which the editor's `/help` documents. A review
  sitting inherited that cost with nothing saying so.

**Before adopting a seam, enumerate what its current consumer feeds it and what
yours will feed it differently.** The answer is usually one or two things, and
they are exactly the ones that will surface as findings.

## Fix the class by ENUMERATING its sites, not by fixing the one you were shown (`#41`)

The same family — `frame-clips-unwrapped-text` — produced three findings across
an operator report and two review rounds, because each fix closed the instance in
front of it: the option gloss at the startup width, then the same lines after a
resize, then the rendered definition a reveal writes. The rule that covered all
of them was available at the first fix: *anything written into a clipping frame
must be wrapped at the moment of WRITING, not of rendering.*

Two habits that would have caught it:

- **When a finding names a class, write the enumeration down before fixing**, and
  put the ENUMERATION in the test — one predicate over every line the loop
  writes, rather than one assertion per line-kind. That test reddens for the
  sixth site the day someone adds it.
- **`grep -l` is the enumeration.** `#41`'s plan promised to "re-examine the
  tests that assert over the surface this issue changes" and named three pty
  rows from memory; `grep -l TestPTYPlay` has four, and the fourth was the one
  frames broke.

## A plan that names an anti-pattern is not protection against writing it (`#41`)

`#41` D1 said: *"the honest move is to widen the shared seam rather than grow a
parallel one — a second way to draw is the thing this issue exists to remove, not
to add."* The first implementation then copied `replRaw`'s six-statement console
construction verbatim (differing in one token) and the editor's four-case
viewport switch. Both were caught by review, not by the plan that forbade them.

**When a decision forbids duplication, the diff is where it is enforced.** After
writing a block that mirrors an existing one, diff them literally before
committing — if they differ in one token, that token is the parameter.

## A guard added to protect a special case must name the CASE, not a mechanism it happens to use (`#41`)

`--play` wraps what it writes, because a frame clips an over-wide line. The
`♫ playing 3×` indicator carries the screen's `\r\x1b[K` take-that-line-back
marker, which a wrap would scatter across a break — so the wrap skipped it. The
guard was written as *"skip lines carrying an escape"*, which is a mechanism the
case happens to use.

Every rendered definition line carries colour. `--play` refuses to run with
`-no-color`. So the guard exempted the entire class the wrap exists for, and the
suite stayed green — see the next entry for why.

**The case was `strings.Contains(line, eraseLine)` and it was one call away.**
When adding a skip, write down the sentence describing what must be protected,
then encode THAT sentence. If the encoding is broader than the sentence, the
difference is what will break.

## A test rig's defaults must be reachable from the flag parse of the command under test (`#41`)

`playRig` returned `options{color: false}`. A `--play` gate landed mid-issue that
REFUSES unless `opt.tty`, and `tty` and `color` are the same expression at the
flag parse — so from that moment no in-process sitting test drove a configuration
production can produce. Two tests written specifically to pin the wrap were green
over uncoloured text while the wrap was broken for every coloured line.

**A default production cannot produce is a suite testing a state that does not
exist, and it fails by passing.** This is repo-general, not a `cmd/define`
quirk: when a command grows a precondition, grep the rigs for defaults that now
violate it. Deriving the rig's options from the same helper the flag parse uses
removes the question.

**It recurred one issue later, on the same rig, through a different field.**
`#38` made the words in a sitting clickable; `playRig` still carried `width: 0`,
which is `terminalWidth`'s "do not wrap" sentinel. A sitting always has a real
width, so the wrap the new click map has to survive was simply OFF in every
test — and the map was silently dropped on every multiple-choice question,
because a gloss below the headword wraps. Four tests written specifically for the
feature were green while the feature was inert in its commonest case, and the
operator found it on the first real sitting.

**So the rule is stronger than "check the rig when a precondition lands":
EVERY sentinel-valued default in a rig is a state production may not have.** `0`
meaning "off", `""` meaning "none", a nil clock — each one turns some
downstream behaviour off, and the test then asserts over a path with that
behaviour missing. Ask of every field: *can the flag parse produce this value for
this command?*

## The wrap and the map must be measured by one ruler, and only the owner holds it (`#38`)

A click map's coordinates are relative to the text they were computed from. A
pinned screen wraps between the caller and the buffer, so those coordinates move.
Two versions of the rule shipped and both were wrong:

- **All-or-nothing** — drop the map if the wrap changed anything — made the
  clicked-on word inert in the commonest case, because one long line beside it
  wrapped.
- **Per line, measured by the caller's width** — the caller held a width fixed at
  startup while the screen re-measures on every resize. After a resize the two
  disagreed and regions landed on lines that did not contain their text, which is
  the wrong-click bug the rule exists to forbid.

The fix is not a better calculation. **Move the calculation to whoever owns the
number**, so a second ruler is unexpressible rather than merely unused — the same
move `#41` made putting the wrap itself on the screen's `Write` after finding a
helper writing around it. When two things must agree about a measurement, one of
them owns it and the other asks.

## Adopting a mechanism means adopting its documented obligations, as checkable rows (`#38`)

`#38`'s T5 said "the revealed definition carries its regions, through
`writeRendered`". `WriteRegions` and `RenderOpts` document three obligations
between them, and the task carried none: the write's leading newline moves every
region down a line; `RenderOpts.Word` must be the caller's KEY, because empty
falls back to the entry's headword and `jalapeno` against `jalapeño` are
different URLs at the CDN; and a pinned screen's wrap moves the map.

Two of the three were live defects. **Before writing a task that adopts an
existing mechanism, read that mechanism's doc comments AT HEAD and turn each
obligation into a row.** A plan written before the mechanism's latest change
never sees the obligations that change added.

## A scripted edit must assert on what it expects to find (`#38`)

Ticking seven task rows in a plan silently did nothing: the edit used a
find-and-replace against text an earlier edit had already changed, so every
substitution matched nothing and the script reported success. The review found
the rows still unticked two rounds later.

**Every scripted edit to an artifact asserts the anchor is present before
replacing it.** A `replace` that matches nothing is indistinguishable from one
that worked, and the failure surfaces at a gate rather than at the keyboard. This
is the same discipline the repo's own guards enforce on prose — applied to the
tool doing the editing.

## Fixing an obligation is not discharging it (`#38`)

`RenderOpts.Word` was left empty, so region words fell back to the entry's
headword — `jalapeño` where the deck holds `jalapeno`, which are different URLs
at the CDN. The fix was one field, the commit message called it "a live defect",
and deleting the field again left the entire suite green.

**A claim is discharged by something that can fail.** A one-line fix earns a test
exactly as a feature does, and the cheapest moment to write it is while the
divergence is still in your head — the fixture is the thing you just reproduced.

## A `red when` cell is a mutation, and it has to be RUN (`#38`)

`#38`'s Done-when row 2 read *"a click NEVER answers — red when: the click reaches
`play.Apply`"*. The boundary review executed that literal mutation and the test
stayed green: the property was delivered by `toInput`'s default (it returns false
for a click), not by the guard the row was written for. The row pinned something
the code under test did not provide.

The second attempt still survived, for a subtler reason: with a ONE-word deck a
click that advanced simply ended the sitting, which is indistinguishable from not
advancing. It needed two questions before the advance was observable.

**Sweep the whole Done-when table as mutations before crossing a boundary**, and
when a mutation does not redden, ask which of the two things it means: the test
is weak, or the fixture cannot express the failure. The second is the one that
hides.

**And record WHICH rows were swept, not that the table was.** Claiming the sweep
covered everything is only as true as the weakest row, and a row pinned by a test
that SKIPs where the sweep runs — every pty row here — cannot be part of it. The
same finding twice in one issue: a blanket claim over a table is a citation that
does not point at anything.

## Two orphaned doc comments were found by a 50-line AST guard (`#38`)

A comment block acquires the wrong owner when a declaration is inserted between
it and its function: the new one arrives undocumented and the old one's prose now
describes its neighbour. `go vet` does not look, and the exported-comment linters
do not reach unexported declarations — which is most of `cmd/define`.

`TestADocCommentNamesWhatItSitsOn` walks the package with `go/ast` and fires when
a function's doc opens with the name of another function IN THE SAME FILE — the
shape an insertion produces. Same-file, because a first word naming something two
files away is prose (`newStoreCapturer`'s doc opens by naming its `vocab`
parameter). Test functions are exempt: their docs name the subject by convention.

It found five, two of them a day old and two nobody had noticed — `openStore`'s
doc had drifted onto `newsFeedFor`, `checkPlanName`'s onto `coreConceptsSection`.
**When a finding is "a comment is in the wrong place", ask whether the class is
walkable; here it was fifty lines.**


## An enumeration that lives in prose fails silently the next time the set grows (`#40`)

`#40` D12 wrote down FOUR `Apply` paths that must consult the `Batch` capability,
found by measurement, and the plan treated that list as the deliverable. The real
shape is `InputKind × Batch`, and `InputReveal` was a fifth cell nobody had
counted — so space on a board set `Revealed`, handed the loop an arbitrary cell's
word to pronounce and a blank reveal to file in the append-only buffer. The
boundary review reproduced it by execution.

**The deliverable is the enumeration, not the guard.** `numInputKinds` is now a
sentinel and a table test ranges over it, so the next kind added arrives with no
expectation and fails. `choice.go`'s `numAxes` had already established the
pattern in the same package — *"the guard derives the set from this, never from a
list"* — and the list was written down anyway.

Ask, of any "the following N places must X": **what makes N?** If it is a
property of a type, derive it. If it is prose, it is already stale.

## A constant standing in for a measurement someone else already computes (`#40`)

`boardChromeRows = 2` charged a board's keys prompt one row. That line is 76
columns wide and the board was offered from 20, so below 76 the live edge was
under-budgeted and the footer silently dropped rows from the end — the bar, then
the panel, then the mode toggle, which is the ONE owner of which mark is live on
a surface where every mark is irreversible.

`displayRows` already answers "how tall is this line at this width". The
constant was a second, implicit owner of it. **When a budget charges a component
a fixed height, ask whether anything in the tree already measures that
component** — and if the answer is yes, the constant is the bug waiting.

The fix kept ONE constant deliberately, and the asymmetry is the interesting
part: the bar keeps a one-row minimum because `fitFooter` drops from the END and
the bar is last, so its real height cannot cost the board anything. A budget only
has to measure what it can be squeezed by.

## A `## Revisions` section placed mid-document truncates every guard that reads it (`#40`)

`currentTruthOnly` cuts a plan at its first `## Revisions`, so the guards see only
what is above it. `#40`'s plan grew two such sections, the first sitting above
`## Done when` — and both `TestPlanTableStatusMatchesTheChangeWindow` and the new
`TestPlanCitesTestsThatExist` were reading a truncated file and passing on it.

Found only because the new guard was mutation-checked: citing a test that does
not exist left it green. **A guard added without a mutation is a guard nobody has
seen fail**, and this one would have shipped certifying nothing.

Revisions are APPENDED, once, at the END. AGENTS.md already says "append"; this
is what the word is doing.

## An in-memory double cannot pin a claim about persistence (`#40`)

`yaml:"-"` on a new `ReviewEvent` field left the whole suite green: the loop's
tests read through `store.Mem`, which keeps events in memory, so the field
reached every assertion without ever reaching a file. The field exists to be
queried months later, and a field that reaches only memory answers nothing.

Same shape as `#30`'s rule about doubles standing in for the object that joins
two separately-pinned halves — here the halves are the struct and the file, and
the thing between them is the tag. **A claim about what SURVIVES needs the real
writer.**

## A Done-when that names a proxy is a hypothesis, and measuring it can falsify the proxy (`#40`)

Done-when 13 asked for "materially fewer KEYSTROKES than form 2.3". Measured:
1.00 per word against 1.00, or 1.12 against 1.25 with misses. A wash. The claim
the Spec actually makes — *"a hundred mature words cost what ten fragile ones
cost"* — is about what the learner READS, and there the ratio is 20-40x.

The wrong response is to find a framing under which the proxy passes. **Say which
measurement the claim rests on, and move the row.** A proxy that survives its own
measurement unexamined is how a Done-when becomes decoration.

## A layout fixed at selection time survives exactly until the window does not (`#40` R9)

`Board` computed its columns once, in its constructor, and the field comment said
the width "cannot change". The terminal is resized under a live board: a layout
for eighty columns has 74-column rows, at forty each wraps into two, a footer
entry stops being one physical row, and a click on the continuation carries a
column that means a different word. Permanently — the mark is already in the log.

**The rule: every quantity a click map or a frame budget depends on must be read
from the terminal AS IT IS at draw and click time.** Not at selection time, not
at construction. The enumeration is small and worth writing out for any
live-edge surface: the prompt's height, the form's layout width, the fit
re-check after a resize, and the column translation for an entry that wrapped.

Two defences, and both earn their place: the form relays out (closing it at the
root) and the loop refuses a click on any continuation row (closing it at the
seam, for the next multi-row entry and for the day someone forgets to pass the
resize on). **"Should never happen" is not a thing to bet an irreversible action
on.**

## A guard that reports success about a file it never saw is worse than no guard (`#40` R10)

`currentTruthOnly` truncates an artifact at its first `## Revisions` — sound only
while records come last. A plan grew a second one higher up, and every guard
reading it saw a file that stopped before the section it existed to check. Four
of the eight then reported "nothing to check" and SKIPPED.

The first fix was to reorder that one plan. The shape comes back tomorrow, in any
artifact, and it comes back as success.

**A guard reading a FILTERED view must assert its premise about that view and
FAIL — never Skip — when the filter removed its subject.** The filter is the
right place for it: one check there covered all eight call sites. And a guard
whose "nothing to check" branch cannot distinguish *there was nothing* from
*I was handed nothing* has a hole exactly the size of its own filter.

## Fixing the instance is how a family reaches round two (`#40`)

Round 1 of `#40`'s boundary review found a prose enumeration that had gone stale
(`Batch` "consulted at FOUR points", with a fifth path unguarded) and a constant
standing in for a measurement. Both were fixed at the class — a sentinel plus a
matrix test, a measured height — and round 2 still returned three REPEAT families:
the same `frame-budget-hardcoded-not-measured` through the resize door, the same
`plan-citations-unenforced` through the skip, and the same
`comment-asserts-absent-behaviour` in the very comment that had declared prose
the culprit while still saying "four".

**When a review names a family, enumerate every member before fixing one.** The
question is not "where else does this exact bug appear" but "what else is this
quantity read from, and when". Writing the enumeration into the fix — as a
derived set, a measured value, or a table in the revision — is what stops round
three.

## Put the irreplaceable thing on the row that survives (`#40` R11)

The board's mode toggle had its own footer row, because the live edge is where
things that change belong. `fitFooter` drops footer rows from the END, so a
narrowing resize dropped the panel and then the toggle — leaving a grid on screen
with no statement of what the next click would MEAN, while every mark is
irreversible.

`Paint`'s order of sacrifice is a design surface, not an implementation detail.
**Ask, of every element on a live edge: what does its absence cost, and where in
the drop order does that put it?** The mode moved to the prompt row, which Paint
clips last. Still one owner — the question was never whether to duplicate it, but
which row it should be on.

The general form: an element whose absence makes the remaining UI *misleading*
outranks every element whose absence merely makes it *smaller*.

## A test that hangs on the defect is barely better than one that passes on it (`#40` R12)

A resize test drove its input from a helper goroutine that called `waitFor`,
whose timeout is `t.Fatal`. `FailNow` off the test goroutine is a `Goexit`: the
goroutine died without closing the key channel, the loop blocked forever, and the
mutation that should have reddened the test hung the run instead — five minutes,
no output, no signal.

**A driver goroutine closes its channel with `defer`, always, and reports nothing
itself.** Assertions belong on the test goroutine, where a failure is a failure.
The same test also asserted over an event set it never produced (`for _, e :=
range events` with no count check) — so: **a test whose subject is an event must
assert the event happened**, before it asserts anything about it.

## A filter that discards silently cannot be audited by reading it (`#40` R14)

`currentTruthOnly` strips the record sections out of an artifact before the name
guards read it, and it found closed blocks with `strings.Contains(sec,
"**closed:**")`. `atlas/repo-guards.md` DOCUMENTS that rule, so the marker
appears in its prose — and the whole guard inventory had been discarded from
every guard reading current truth. Two of them were blind over that page for
months, and unblinding it turned up five retired identifiers on the page whose
subject is retired identifiers.

Nobody found it by reading the filter, twice over: the fix that added a premise
assertion to the OTHER discarding rule was written on the belief there was no
live instance, and it took thirty seconds to write and fired immediately.

**A discard is a decision, and a decision that never speaks cannot be reviewed.**
Where a filter drops something its caller was going to check, say so and fail.
And match markers the way they are WRITTEN — anchored — because prose about a
marker is not a marker, and documentation of a rule is the first place that
distinction bites.

## A frame's lines are not the terminal's rows (`#40` R15)

`FooterRowAt` answers in the rows the terminal reports for a click. A frame
string split on `\r\n` gives LOGICAL lines. They agree only while nothing wraps —
and the keys prompt is one logical line and two physical rows in a narrow window,
so from that point down the two indices differ by one and a click placed by frame
index lands a row high.

Three attempts at one test driver, each a different way of being wrong about
this: scraping the frame raced the redraw; asking the screen alone raced the
other way, because it answers from the last paint; and requiring both, matched by
PREFIX, admitted the stale state anyway — the narrow layout's first row is a
prefix of the wide one's.

**Drive a screen through the screen.** If a test needs to know where something
was drawn, ask the object that drew it, and detect state changes by something the
old state cannot produce — here the footer's entry COUNT, which grows when the
board relays out. And never let a test goroutine touch a form the loop owns:
`-race` says so, and production has one goroutine on it for the same reason.

## A review checks that the code does what the plan says; a sitting checks whether the plan was right (`#40` R16)

Three boundary-review rounds on `#40` found a Critical reproduced by execution, a
resize that landed a permanent mark on the wrong word, and two guards that had
been certifying nothing. The operator's FIRST real sitting found four things none
of them did:

- the mark replaced the cell's key, which is how a mouse-less terminal reaches it
- the label sequence skipped `d` to protect a key that did nothing on that screen
- the grid began flush against the previous question, with no separator
- (and the fix for the second broke three restatements of the old sequence)

Every one is *correct code that reads wrong to the person using it*. A reviewer
reads the plan and the diff and checks they agree; only the person holding the
keyboard can tell you the agreement was on the wrong thing.

**So: get it in front of the operator before the boundary review, not after.** A
round of review spent on a design a sitting would have changed is a round spent
polishing the wrong object — and the plan's `ux-rename-iteration` line, priced for
"3–5 rounds per TUI-heavy milestone", is an estimate of exactly this and was
still treated as if reviews could substitute for it.

## Let the plan-status guard adjudicate, and commit first (`#40`)

`livePrompt`/`gradePrompt` flipped status four times in one issue. Twice that was
real churn — the code genuinely changed across review rounds. Twice it was me
reading `git diff` hunk headers by hand and losing to the guard, which locates a
declaration in the CURRENT file and compares it against the diff from the
merge-base: while edits sit uncommitted, the two can disagree about which
function a hunk lands in.

**Commit, then run the guard, then write what it says.** Arguing with a
mechanism that reads the tree from an argument about line numbers is a way to
spend a round and be wrong at the end of it.

## "Harmless" is a claim about a mechanism, not about a state (`#40` R17)

A shrunken terminal drops trailing footer rows, so some of a board's grid rows go
unpainted. I checked that against the click map — `FooterRowAt` answers nothing
for a row that was never drawn, so a click cannot reach one — and wrote the
losses down as harmless in three places.

Enter does not go through the click map. It sweeps every unmarked word as wrong,
including the ones that were never on screen: boxes halved on one keystroke, for
words the learner had no chance to look at.

The word "harmless" was carried from the mechanism it was verified against to a
different one, silently. **A safety claim names the path it was checked on.** If
a second path reaches the same state, it is a second claim and needs its own
check — and the enumeration of paths is the deliverable, exactly as it was for
`InputKind × Batch`.

## Read the terminal at DRAW time, and only there (`#40` R17)

Two separate bugs, one shape: a fact about the terminal read once, where it had
to be read every frame. The resize handler told the CURRENT form its new width —
fixing the board on screen at that instant and no other, while the next board was
built at the old width and painted too wide.

**The place that draws is the only place that can promise anything about how
things are drawn.** Ask the screen for its shape there; do not keep a copy, which
is correct until the first resize you miss. Make the setter idempotent so calling
it every frame costs a comparison.

## The sweep set for a drawn-or-keyed contract is the same every time (`#40` R18)

Five findings in one issue said "prose asserts behaviour that is absent". The
fifth one finally named the fix: **stop patching sites and write the enumeration
down.** A change to what is drawn, or to what a key does, goes stale in exactly
seven places:

1. the comment where the mechanism moved FROM — it describes a call that is gone
2. the comment where it moved TO — it inherits the old reasoning verbatim
3. `atlas/*.md`, which names call sites and repeats safety words
4. README prose, which states contracts unconditionally
5. the README key table — a key that gained a condition still reads absolute
6. the README example block — a picture with no consumer
7. the plan's Done-when AND the issue's Done-when, both

Run it in the SAME commit. What no guard catches is shipped behaviour with **no**
citation — a guard can only check that a citation resolves, and both of R17's
tests existed while neither was named. The checklist is the substitute.

Row 6 is the one that can stop being prose: make the example DERIVE. `#40`'s
README board block now builds a real form and asserts the fenced rows are what it
draws — it had gone stale within hours of the sitting that changed the design,
while the prose eight lines below contradicted it.

## A deleted element leaves no symbol to rename, so write the PHRASE down (`#40` BR-15)

The rule above says run the sweep in the same commit. It was written down and
then not run, twice — which means the sweep needed a mechanism, not a better
reminder.

`retiredSymbolNames` already turns a RENAME into a build failure: the human adds
one row, and every later commit is swept mechanically. A DELETION of something
drawn gets none of that. `#40` R11 removed a footer row and moved what it said
onto the prompt row; the row's only identifier was unexported, so
`isCitableName` filtered it out and nothing mechanical ever saw the change. Five
comments went on describing a row that is not drawn — one of them contradicting
its own owner twenty lines below it — and it took five rounds of one boundary
review to enumerate them.

**So: `retiredPhrases` is the same mechanism for the half a compiler cannot
reach.** A phrase naming a drawn element the tool no longer has, mapped to what
states that fact now, swept over every current-truth artifact by
`TestNoArtifactDescribesARetiredDrawnElement`. Keys are PHRASES rather than
words, because the word usually survives the row — the verb that named the
deleted row still names what `Tab` does.

Two things fall out of it. **A doc page that documents the guard must not spell
the retired phrase** — the same convention `retiredSymbolNames` already has, for
the same reason: the page would become the next stale artifact. And **prose that
restates a table's ORDER is a second owner of that order** — "rows three to six"
renumbers itself silently the day a row is inserted. Name the rows.

## An open ledger row is a question, not an answer (`#40`, round 6)

A boundary review round read the whole window and returned no machine-readable
findings block. The gate converged anyway and printed seven findings as still
open, two of them Important and demoted past the round cap with the explicit
warning that no later gate picks them up.

Measured against the tree at the publish gate: **five of the seven were already
fixed** by the two commits after the round that raised them, and unrecorded only
because a round that names nothing can dispose nothing. One was real.

**Before crossing a boundary on a demoted finding, re-measure it against the
tree.** Neither trusting the ledger nor dismissing it is available: the ledger
records what a reviewer saw at some past HEAD, and it is the tree that ships. And
record every round's outcome in the ISSUE, not only in the plan and the gate
files — the tracker is the artifact a reader opens first, and it was four rounds
stale while three other files were current.

## A guard's SCOPE is a rule too, and an inclusion list is the failure it exists to catch (`#44`, rounds 1–3)

`#44`'s whole thesis was that a sweep is not a fix: five call sites had drifted
three ways, so the deliverable was a source-level guard rather than five
corrections. The guard then scoped itself with `screenHostedFiles = {…}` — a
hand-maintained enumeration, which is exactly the shape it was written to
replace, one level up. A sixth screen-hosted file would simply not be checked.

Inverted, it became `nonScreenFiles`: **scan everything, exempt by name, and make
each exemption carry its reason.** A new file then defaults INTO the rule.

The instructive part is what happened next. A sibling guard written *in the same
file, one screen below that comment*, hardcoded a single filename — the inclusion
shape again, by the same hand, minutes later. Then the shared body of the two was
found to be a hand copy in which the copy had **dropped a Fatal**, turning
"package main parsed to no files" from a diagnosis into a nil panic.

**Three rules, and the third is the one that generalises:**
- A guard's file scope is an exemption list, never an enumeration. Its
  `scanned == 0` and `checked == 0` cases must FATAL, and an exemption naming a
  file that no longer exists must fail — a guard that certifies nothing is worse
  than no guard, because it reads as evidence.
- Every guard's predicate is stated over the PROPERTY, not over a name that
  happens to have it today. "Every call to `playAnnounced`" was blind to a
  forwarder; "every argument of type `indicator`" was not.
- **Writing a rule down does not install it.** Both rules above failed on their
  first application, in the same session that invented them. The second copy of a
  guard is a helper, extracted then, not the third time.

## A pin that cannot fail is not a pin — check it by breaking the code (`#44`, rounds 1–2)

Two boundary rounds produced six findings in one family: a test that passed for a
reason other than the one it claimed.

- A footer-click test asked `FooterRowAt(footerTop + i) == i`, and `FooterRowAt`
  *is* `row - footerTop` — it passed with the field set to nonsense.
- Three tests built `screen{pinned: true, gap: chromeGap}` by hand, so deleting
  the production wiring `newPinnedScreen` sets left the whole suite green — on the
  issue's headline behaviour.
- A styling test checked "this line carries a dim escape", and `Paint` reprints
  the prompt onto the same `\n`-split line as the bar, so the prompt's dim
  satisfied it while the bar had none.
- A Done-when row ("the special case is deleted") and a threshold constant
  (`>= want+1`) were both ticked with nothing able to notice them being undone.

**The rule: before ticking a Done-when row or landing a threshold, revert the code
and watch the named test go red.** Record the mutation beside the claim. Three
corollaries, each earned here:
- **Construct through the PRODUCTION constructor.** A test that builds the object
  by hand does not test the wiring that builds it in production.
- **Anchor an assertion to the text, not to the row** — a terminal frame puts
  several things on one line.
- **Measure a DIFFERENCE where the absolute number is noisy.** "Same sitting,
  audible vs silent" isolated one leaked row out of 547; counting blank lines
  could not, because a dictionary entry is full of them.

## A retraction is not done until `git grep` over the TREE is clean (`#44`, rounds 2–3)

A plan's `## Revisions` entry recorded that `fitsABoard` would NOT charge the gap
after all. Two rounds later, `grantedGap`'s own doc still advertised `fitsABoard`
as its second consumer — which reads as an instruction to add the term back — and
the atlas still described `playRegion`'s deleted parameter as a deliberate
difference between the two loops: **the bug, written down as a design note, in the
document whose job is telling the next reader how this works.**

**When a design decision is retracted, `git grep <entity>` over the whole tree is
the enumeration.** One grep per revised entity. Scoping it to the directory you
happen to be editing is the same defect as scoping a guard to the files you happen
to remember — the first pass here ran over `cmd/` and missed both atlas hits.

And: **a rule recorded only in a plan is a rule that will not be read.** Plans are
archived to `workshop/history/` at close, which `AGENTS.md` §2 tells the next agent
not to read. If a round produced a rule, it belongs HERE.

## Mutation testing has to be done, and the tooling for it has to be safe (`#42`, close round 1)

Two failures in one round, both about the same habit.

**A mutation check you did not run is worse than none**, because the tick claims
it. `#42`'s plan named three required mutation checks for the board's drop; I ran
two and ticked all three, and the boundary review found that the untested one —
the palette — could be deleted twice over with the whole suite green. A dropped
cell would have painted identically to an untouched one while the README promised
it was struck out.

The fix that generalises is not "run the third check". It is that a palette test
listing three fields says nothing about a fourth mark, so the check DERIVES from
the mark set: `play.Marks()` is the extent, `Palette.For` is the one owner of
mark → sequence, and a new mark with no colour now fails the day it is declared.
Same shape as `numRegionKinds` guarding the click registry.

**And the scripted revert must not use an empty replacement.** A helper doing
`s.replace(from, to)` to mutate and `s.replace(to, from)` to restore silently
PREPENDS the original text at byte 0 when `to` is `""` — Python's `str.replace`
matches the empty string at every position. Two source files were corrupted into
`Drop: "\x1b[2;9m", package main`. Mutate by replacing a line with a *different*
line (`if cond {` → `if false {`), or copy the file aside and copy it back. Never
restore by replacing an empty string.

(The corruption did prove the point: with both arms missing, exactly the two
tests that should have failed did.)

## A deletion's blast radius is every artifact that named the thing (`#42`, close rounds 1–2)

Three consecutive review rounds found the same shape: a sweep that fixed the sites
the previous round named and not the class. `PQ-7` named four files; I fixed four
and five were left. `BR-3` named five; I fixed five and eleven were left, spread
over `boardsFor` (deleted in the same window) and comments restating a count that
had changed.

**The guard that should have caught it existed and could not see it.**
`TestARemovedDeclarationIsSweptOrRetired` sweeps every artifact for names the
window removed — but gated on `isCitableName`, which required an EXPORTED or
`Test*` name, on the reasoning that unexported helpers are not cited in prose.
That is true of `ids` and `binds` and false of exactly the helpers a codebase
argues about: `boardsFor` stayed the current account of selection in two atlas
paragraphs, so the atlas held two contradictory accounts of the rule the issue
existed to change.

**The interior capital is the discriminator.** A prose-cited unexported name here
is a compound (`boardsFor`, `choiceFor`, `optionCandidates`); a single lowercase
word (`ids`, `paint`) is both uncited and a substring of ordinary English. Widening
on that keeps the noise out and lets the citations in.

**Two more rules from the same rounds:**

- **A comment must not restate a count the code enumerates.** `Mark`'s own doc
  said "TWO marks and an ABSENCE" three lines above the const block declaring a
  third. The fix is not the edit — it is that the extent became `Marks()` and the
  prose defers to it, exactly as `Keys()` already declines to enumerate the label
  set.
- **When two guards disagree about one artifact, settle it where "is this a
  record?" is already decided.** The plan-table guard REQUIRES a plan to name what
  the window deleted; the retired-symbol guard forbids naming a retired symbol.
  Neither could yield alone. It belongs in `currentTruthOnly`, so both inherit one
  answer — and the exemption is self-limiting: only a document carrying a
  `| deleted |` row gets it, and only for the symbol that row names.

**And the guard's own name for the failure was right:** *"a guard that depends on
someone remembering has now been remembered late twice."* Every fix above replaces
remembering with a build failure.

## Fixing a class means pinning the fix, not just widening the rule (`#42`, close round 4)

Round 2 answered "the sweep was the instance, not the class" by widening
`isCitableName` so the removed-declaration guard could see unexported compound
names. Round 4 reverted that widening and **the entire suite stayed green** — the
symbol it was written for had just been swept, so nothing in the tree exercised
the new clause. The fix that closed a class was itself unpinned, which is the
family the previous entry in this file is about.

**When a rule's triggering input no longer exists in the tree, the pin is a
fixture table.** The repo already had the precedent (`TestPlanStatusNormalisesToTheVocabulary`
exists because no plan writes a bolded status), and the mechanism is eight lines
from the rule it defends. Supply the input rather than hoping the tree contains
it.

**Two more from the same round:**

- **Re-wording a comment does not close a "prose restates a count" class.** The
  replacement said *"THREE marks and an ABSENCE, and `Marks()` below is the EXTENT
  — a count spelled in prose is a second owner of it"* — spelling the count inside
  the sentence forbidding it. And it claimed its test "derives its loop from the
  cycle" while the test read `for range 3`. **Make the code carry the extent and
  the prose name nothing**: the spelling table is now keyed by mark, the test walks
  `Marks()`, and a mark with no spelling fails the build instead of silently
  drawing the default row.
- **Tense is the discriminator when a concept is retired but its history is worth
  keeping.** Banning the form's NAME would have reddened ~8 legitimate historical
  mentions, so round 1 declined the ban — and left seven present-tense claims
  standing three rounds later. Keying the guard on `"form 2.1 is"`, `"has"`,
  `"cannot"` catches the claims and leaves `"was"`, `"used to"`, `"before #42"`
  alone by construction. It found three more the hand-grep had missed.

**And a test that HANGS on the defect is worse than one that misses it.** The
first version of the mark-spelling walk was `for b.Mode() != m { b.Toggle() }`,
which never terminates for a mark `Toggle` cannot reach — exactly the mark the
test exists to catch. A red says what is wrong; a hang says nothing and takes the
suite with it. Bound every search whose termination depends on the property under
test.

## #10 M1 — four boundary rounds, and one class that kept coming back

**A finding is disposed by CODE, not by the paragraph promising it.** `#10`'s
plan-quality gate raised a neutralisation finding; the plan answered it with a
paragraph specifying `sanitiseFacts`/`sanitiseItem`, the gate recorded it
`addressed`, the step was ticked, and neither function was ever written. The
boundary review found it two rounds later. **A ticked checkbox is the weakest
evidence in the loop, because ticking it is the cheapest thing in the loop** — so
when disposing a finding, name the symbol and grep for it.

**"Fix the class, not the site" fails in a specific, predictable way: you fix the
instances the finding NAMED.** Three rounds in a row on this issue:

- a mode-collision guard covered `--play` and `--reflect` — the two the finding
  listed — while `-forget` and `--llm-check`, which dispatch *above* that switch,
  still swallowed the new mode in silence;
- a "properties without pins" finding listed three, they were pinned, and the
  round-4 review found the enumeration the round-3 finding had itself written
  down had seven members and one was swept;
- a read-side canonicalisation rule was stated on one accessor while its sibling
  forty lines up returned the raw record.

**The tell is that the finding hands you the enumeration and you use it as a
list of sites instead of as a specification.** When a finding says "2nd in
family", write the enumeration down as an object the code shares — a `modes`
slice both the check and its table test walk — so the next member is covered by
construction rather than by the next reviewer.

**A seam only constrains callers that go through it.** `bandTask` existed
precisely so `--harvest` and its measurement mode could not ask different
questions, and the conformance row broke that from OUTSIDE by handing the seam
different arguments: it floored a bare word while production sends a dictionary
gloss. Every number the milestone reported as *measured* was a number about a
prompt nobody runs. **When a test asserts a property of production, derive its
inputs the way production derives them** — the row now calls the same
`senseFacts` the harvest loop does.

**Do not write the calibration prose before the gate runs.** The project's M1
paragraph said "est 3.94 / actual 2.16 = 1.82, and the milestone had no
remediation round at all" — committed before the boundary review, which then took
three more rounds and 1.9h. Predeclaring an outcome and then measuring it is how
a calibration ledger stops being evidence.

**And a `go test` that takes ~110s is a review-agent hazard.** One boundary round
produced no verdict at all: the reviewer spent its budget waiting on repeated
full-suite runs and was cut off mid-sentence. Not a code defect, but it cost a
round — worth knowing before blaming the diff.

## #10 M2 — what reading real output found that a green suite could not

**Stating a rule in a prompt is not enforcing it.** The author system prompt said
*"You never explain the word, and you never write a definition"* and half of the
first twenty items came back as appositive glosses — *"the alewife, the small
silver herring"*. The model honoured the letter and wrote a definition in a form
the sentence didn't call a definition. **Showing three wrong shapes and two right
ones is what worked**, plus a judge field that asks about the shape directly.

**Two requirements can be individually right and jointly impossible.** "The stem
must entail its answer" and "the stem must never define the word" are each
defensible, and together they are unsatisfiable for any concrete noun: the
cheapest way to make a sentence entail a word IS to define it. The second batch
said so out loud — items rejected because *"no definition is supplied"*, the
judge citing the absence of the thing the other rule forbids. **When a judge's
rejection reason cites a rule you deliberately broke, the requirements are
fighting, not the output.**

**And the resolution was to re-read what the form actually is.** The bar
"recoverable from the sentence alone" is a fill-in-the-blank criterion, and this
is MULTIPLE CHOICE — the learner sees four options, so "does any other word fit"
is a question about the OPTIONS, which the veto already asked per pair. One judge
was doing the other's job badly. Check what the artifact is before specifying
what makes it good.

**Ask the free question before the paid one.** An item shipped with `___` already
in its stem, against an explicit instruction, and BOTH model judges passed it
because neither was asked. `strings.Contains` would have caught it. Every
model-judged property should be preceded by the deterministic checks that are
cheaper and stronger — the same rule as not asking a model for a domain the
dictionary printed.

**A per-item constraint says nothing about a batch.** Every selection rule was
about one question, so one eligible word served as the wrong answer in 8 of 20
items. Nothing was violated; the property nobody had stated was the one that
mattered. When output is generated in batches, at least one measure has to be
taken over the batch.

**And a mechanism can be correct while the data starves it.** The diversity fix
demonstrably works (worst-case reuse 6→4 on a homogeneous pool) and did almost
nothing on the real deck, because for a C1 Nautical word the whole eligible tier
was two words. **Measure the fix on the shape that motivated it, not only on a
constructed one** — and when the limit is the input rather than the code, say so
instead of adding machinery.

## #10 M2's review — the sweep row, and pins that had tests

**A milestone's Verification sweep is not satisfied by the previous milestone's
table.** I ran the mutation sweep for M1, wrote its 13-row table into the plan,
and then ticked the same row at M2's boundary. Three headline M2 properties had
pins that could not fail. The row is per-milestone; enumerate THIS milestone's
properties from the diff's branches, not from the atlas's claims.

**Two of the four unpinned properties HAD tests, which is the failure worth
recognising.** `prune`'s determinism was asserted by pruning the same slice
twice — so a prune that returned its input unchanged agreed with itself
perfectly. The `## Corrections` guard used a fixture domain the parse refuses
anyway, so it could not distinguish the guard from the parse. **A pin whose
fixture cannot reach the branch is the same failure as no pin**, and reading the
test does not reveal it — only reverting the code does.

**A rig too small to run the pass makes every assertion about that pass
vacuous.** `harvestRig(t, 1)` gave a pool of one, so authoring bailed before
running and "nothing was authored" passed for an unrelated reason. Fixed at the
class level: the rig now REFUSES a size that cannot exercise the path, and the
skipped pass says so on stdout where a test can read it. Prefer making the
degenerate case loud over remembering not to construct it.

**Assert a comparison, not a threshold, when the claim is "X improves Y."** The
diversity pin checked `worst > 3`, which both branches satisfied. Run the code
with the feature off and on over the same inputs and assert the difference —
otherwise the test measures the fixture.

**A flag's counter must be the resource the flag names.** `-limit` documented a
ceiling on model calls and counted successes per pass, so rejected words charged
nothing and a run cost one call per deck word regardless. It also spent the same
value twice, once per pass. One budget, threaded, charged next to every call
site including inner loops.

**A value parsed and read by nothing is not a delivered consumer.** The learner's
domains were parsed into a struct field whose doc comment said selection did
arithmetic on it; grep found zero production readers. Either wire it or delete
it — and when the Spec names it, wiring it is the deliverable, not a follow-up.

## #10 M2 rounds 7-8 — a pin written to dispose a finding, itself unfalsifiable

**The sharpest one in this issue.** Round 7 found three fixes with no
revert-check; I wrote pins for all three; round 8 reverted them and found one
still green. `TestTheTierReportCountsOnlyWrittenItems` asserted that a
fully-vetoed batch prints no tier line — and the report only ever printed the
WIDENED tiers, so a batch that never widened printed nothing either way. The
assertion was true before the fix and after it.

Two lessons, and the second is the general one:

- **A pin written to dispose a finding gets the same revert-check as the fix.**
  Otherwise the finding is disposed by an assertion, which is what it was
  complaining about.
- **When a report is filtered, a test over the unfiltered case sees nothing.**
  The report listed four of five tiers; the test's fixture produced the fifth.
  Any assertion of the form "X does not appear" needs a sibling asserting that X
  appears when it should, or it passes for the wrong reason forever.

**And a doc claim can be the tell.** The atlas said "the tier reached is printed
for every item" while the code printed four tiers of five. Writing the sentence
is what should have surfaced the gap; instead the sentence was written from the
intent and the code kept its filter. **When you document a claim, check the code
makes it true — a doc sweep is a chance to find bugs, not just to describe.**

**A flag×resource table is worth writing once.** `-limit` took three rounds and
two Criticals: it counted successes rather than calls, then charged without
gating (N+1). What finally fixed it was making the charge structural — one
function is the only path to the model, it charges before calling, and its
refusal returns as an error the caller already handles. **A budget you can
charge without gating on is a budget somebody will charge without gating on.**

## #10's close — Forget, and a guard that classified on one axis

**A new persisted surface is a new thing every per-word verb must reach.** `#10`
added `facts/` and `items/`, and `Forget` removed only the deck entry — so a
forgotten word kept the cached band and authored items that made it worth
forgetting, and `--harvest` then skipped it as already done. *"Forget this word,
its material is bad"* was the one thing forgetting could not do. `usage/` had the
same bug and had it first, from `#9`. **When you add a directory keyed by an
existing noun, enumerate the verbs that act on that noun** — the create path is
the one everybody remembers.

**And the guard I wrote for it classified on ONE axis while the surfaces vary on
two.** "Per-word or history" decided whether `Forget` touches a directory;
"scoped or flat" decides *whose copy* it touches, and `usage/` is per-word and
flat, so forgetting in Spanish reached the English cache. A guard that enumerates
a set is only as good as the number of questions it asks about each member.

**A stale ledger entry is worth proving, not arguing.** `BR-41` blocked three
rounds after it was fixed. What settled it was a grep over every statement of the
flag's meaning — 20 hits, all saying "calls", none saying "words" — recorded in
`--verified` alongside the one precise `--no-ledger`. Bypass one gate for one
finding with the evidence attached; never `--force`.

## #12 — a fuzz that paid twice, and a pin the gate had already asked for

**A gate finding is not disposed by a test that cannot reach the line it named.**
Plan-quality's PQ-2 said a flag recorded as a review would demote the word. I
built the whole path as specified, wrote an end-to-end test AND a `storetest`
row — and the mutation sweep found that changing the real capturer to write
`EventReviewed` left everything green. The end-to-end test drove a FAKE
capturer, so it proved the outcome reaches *a* capturer and nothing about what
that capturer writes; the suite row asserted a hand-written event round-trips,
which is a third claim again. **When a finding names a line, the disposing test
must fail when that line changes** — check it by changing the line.

The fix was also the better test: assert the property through its CONSUMER.
`schedule.Fold` must read nothing from a flag-only log. That is what PQ-2 was
about; the event's `Kind` field was only how it would have gone wrong.

**Fuzz the function whose failure is silent.** `blankStem` renders and grades
perfectly whether or not it leaks its answer, which is exactly the shape a table
of examples cannot cover. Two minutes of fuzzing found a HANG (invalid UTF-8
decodes to `RuneError`, which matches itself and is not a word rune, so the
match had no word run and the loop never advanced) and then a wrong INVARIANT
(`blankStem("_","_") = "___"` — `_` is not a word rune, so "does the answer
occur as a word" is ill-defined for it).

**And the second one is the more useful pattern: a fuzz failure is not always a
code bug.** Sometimes the invariant is wrong, and narrowing it is right — but
only when the narrowing names a real defect it exposed. Here it did: an answer
with no letter and no digit is not a word, so `usableItem` refuses one now.
Narrowing an invariant without finding the defect underneath is how a fuzz gets
trained to pass.

**Then check the fix one predicate over.** `hasWordRune` looked right and let
`---` through, because `isWordRune` counts hyphens as INSIDE a word — correct for
tokenising `hot-dog`, wrong for "is this a word". Joiners are not what a word is
made of.

**A test rig too small can make a row pass for the wrong reason.** Form 2.3 draws
distractors from the sitting's pool, so a one-word rig cannot build one and every
word falls to the board. The two "still takes 2.3" rows of the selection rule
passed on a one-word deck — for `#42`'s reason, not the rule's. Size the fixture
to the path under test.

**A guard that names its own residual has told you where the next bug is.**
`doc_sync_test.go` checked that every enrolled form's prompt lines appear in the
README, and its comment said the quiet part: *"a form added to play and not added
to this slice is not checked here. That half is human."* `Cloze` was then added
to `play/` and not to the slice — so both of its prompt lines went unchecked, and
neither was in the README. The close review found it, not the guard.

**A guard whose extent is hand-maintained is half a guard.** The fix is the move
`numRegionKinds` already makes for region kinds: DERIVE the extent from the code
rather than restating it. `TestEveryFormIsEnrolled` regexes `func (x *T) Form()
string { return "…" }` out of `play/*.go` and fails when a declared form is not
enrolled. When you catch yourself writing "that half is human" in a test comment,
that sentence is the finding — write the test that closes it instead.

**A boundary's durable record has more than one home**, and it is not written
until all of them are: the issue `## Log`, `workshop/lessons.md` when a review
found something (AGENTS.md §4), the plan's `## Revisions`, and the project file.

**Documenting a hazard is not fixing it, and the comment is evidence you saw it.**
`Cloze`'s doc comment said "#38's clickable-prompt premise (line 0, column 0,
width len(word)) is Choice's and does not hold here" — and nothing acted on it,
so the loop underlined the first eleven cells of the blanked sentence and spoke
the answer on a click. The close review found it as a Critical. **When you write
"this does not hold here", the next thing you write is the code that makes it not
matter** — or a failing test if you cannot.

**A region, a coordinate, an offset: check the claim, do not compute it.**
The formula was `Choice.Prompt()`'s layout read as every form's. The fix is not a
special case for the form that broke it but a predicate that IS the region's own
claim — "line 0 begins with the headword" — plus a guard that reads every form's
region coordinates back out of the text actually written. That guard found a
second instance (`Board`) the moment it existed, which is how you know it was
sized to the class and not to the bug.

**And do not "just search for it".** The obvious alternative — find the word in
the prompt — would have been worse than the formula it replaced: a cloze prompt
contains its answer among the options, so the search would have underlined the
correct one. When the claim is about a POSITION, check the position.

**Three homes of one fact is a pattern, not three findings.** `?` was missing
from the prompt line (round 1), from the enrolment that checks the prompt line
(round 2), and from the README key table (round 3) — one gesture, three
hand-maintained enumerations, three rounds. The rule: **every enumeration of live
keys derives from the code that owns them, and a new key is not shipped until
every such enumeration derives.** When a gate finds the same fact missing twice,
stop fixing homes and go count them.

**A derivation that can under-derive silently is the hand-maintained list with
extra steps.** `TestEveryFormIsEnrolled` scraped `Form()` with a regex requiring
a single-letter pointer receiver, a one-line body and a lowercase literal all at
once — and asserted only `declared ⊆ enrolled`, so a form the regex missed was
SILENCE. Renaming a receiver would have re-opened the finding the guard was
written to close, plus the Critical guard built on the same extent. Parse the
code (`go/parser`) rather than matching its formatting, and **make it fail
closed**: assert the counts match, because "everything I found is enrolled" is
satisfied by finding nothing.

That is the second time in two rounds. Both times the guard was mine, and both
times the flaw was the same shape: **I checked that the guard fires, and not that
it fires on everything it claims to cover.**

**When a gate finds the same family four times, the finding is the missing
ENUMERATION.** Stale-restatement was raised in rounds 1, 2, 3 and 4 of one issue;
each round fixed instances and the open count went from five to eleven, three
added by the fixing commits themselves. Instances are not the bug. Split the
family: derive every half that is machine-readable (a guard per fact — see the
table in `workshop/targets/derived-restatement.md`), and put the prose half on a
checklist run at every boundary. Then keep moving rows from the checklist into
the table.

**And check the doc comment on the CALLER.** The commonest stale restatement is a
comment on code the diff did not touch, which is exactly the set `git diff` will
never show you.

**A mutation that does not COMPILE is not a passing mutation.** Sweeping `#46`'s
audio key, I replaced the digest input and grepped the output for the assertion
text. Nothing matched, so it read as "the test did not catch this" — but the
package had failed to BUILD (the mutation orphaned an import), so no test ran at
all. The two outcomes look identical through a narrow grep and mean opposite
things.

Two rules from it: **grep the sweep's output for `build failed` and `FAIL`, not
just for the assertion's own words**; and **write the mutation so it still
compiles** — key on `word + strings.Join(urls[:0], "")` rather than deleting the
argument — so the run is a real one. Also `go test` serves CACHED results: a
sweep needs `-count=1` or it may report the pre-mutation verdict.

**A capability asked for by type assertion must be asserted at COMPILE TIME.**
`#46`'s disk cache is reached through `wordFiler`, an optional-capability
interface the seam asks for with `if wf, ok := inner.(wordFiler); ok`. The first
`forWord` returned `*diskAudioCache` instead of `AudioSource` — a signature Go
accepts everywhere except as an implementation of that interface. So the
assertion never matched, every fetch bypassed the disk, and **the cache did
nothing at all** while compiling and running cleanly.

One line prevents the whole family: `var _ wordFiler = (*diskAudioCache)(nil)`.
Write it beside every type that exists to satisfy an optional interface — the
failure mode is silence, and silence is what a type assertion returns when a
signature drifts.

**Three fixture-cannot-reach-the-branch failures in one issue.** `#46`'s
`re`/`re-` collision needed the FORGOTTEN word's slug plus the separator to be a
prefix of the neighbour's; my fixture forgot `red`, which never globs `re--`, so
the mutation passed and the bug would have shipped. Before that, a
`many`-axis probe took its file NAMES from the declaration it was testing —
self-fulfilling, green under mutation. Before that, a test named for a loop
never entered the loop.

The pattern is one question, asked too late: **"what exact input reaches the
line I am claiming to pin?"** Write the mutation first, watch it redden, and if
it does not, suspect the fixture before the code.

**And a probe whose shape comes from the thing under test proves nothing.** If
the declaration says `many` and the probe therefore plants prefixed files, both
branches agree with themselves. What knows the naming convention is the code that
WRITES it — so the pin belongs in a conformance row driving the real API, not in
a guard planting files it invented.

**A guard that reads prose cannot tell a citation from a reminiscence.** The
plan-cites-tests guard fired on "the obvious name is `TestFoo`" in a paragraph
explaining why that test was NOT written. Backticks are the guard's whole signal,
so historical mentions have to drop them — cheaper than teaching the guard about
tense.
