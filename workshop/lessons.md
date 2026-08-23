
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
