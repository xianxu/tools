
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

## Verifying only what you have tests for proves nothing new (define #11 M2)

Told two findings were still open, I re-ran my own tests, saw green, and reported
that the reviewer was wrong. Both findings were real: they named cases my tests
did not cover (an array of objects; a 502 from a reachable service). My
measurement was a restatement of what I already believed.

**When a finding is re-raised against a fix you believe landed, reproduce the
finding's own case** — not your test suite. The suite is what missed it the first
time.

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

- **`git stash` gives the same guarantee without writing anything permanent.**
  Stash, mutate, `git checkout`, `git stash pop`.
- **If a wip commit does happen, squash it before the boundary.** Non-interactive
  rebase works: `GIT_SEQUENCE_EDITOR="sed -E 's/^pick (sha1|sha2)/fixup \1/'" git
  rebase -i <base>`, with a backup branch first and a `git diff backup --stat`
  after to prove the tree is unchanged.

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
  is COMMITTED.** It cannot go stale the way a scratch copy can. But the same
  session then hit the other half of the trap: restoring uncommitted wiring
  reverted the work itself, because HEAD did not have it yet. So the rule is two
  steps — **commit, then mutate, then `git checkout`** — and neither half works
  alone.
- **Re-verify the mutation AFTER restoring.** The restore can undo the fix the
  mutation was checking, and then both the fix and its pin are gone with the
  suite green.

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

**Before snapshotting a file as a backup, confirm nothing else is mid-mutation on
it**, and after any killed job, restore from GIT (plus re-apply intended edits by
hand) rather than from a snapshot whose provenance you cannot vouch for. The
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
with no backup. `git checkout -- <path>` needs no backup step, cannot land
outside the repo, and is already the restore mechanism.
