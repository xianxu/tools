# define REPL — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bare `define` on a terminal opens a loop: type a word to define and hear it, press return to hear it again, Ctrl-C to quit.

**Architecture:** The one-shot path is factored into `defineOnce`, and the loop calls it — the REPL adds a reader, not a second implementation of `define` (ARCH-DRY). The loop reads **stdin unconditionally** and prints a prompt only when stdin is a terminal, so there is no interactive/batch branch to keep in sync and the whole loop is testable from a string. Replay-without-refetch comes from a `cachingAudioSource` **decorator** on the existing `AudioSource` seam, so the fake CDN's own request recorder is the assertion.

**Correction to the issue as filed.** The Spec claimed `echo word | define` "still" works. It does not and never has: `run()` requires `NArg() == 1`, so no-args exits 2 with usage. Reading words from stdin is therefore **new behaviour introduced here**, not behaviour preserved. Verified at `bin/define` before planning.

**Tech Stack:** Go 1.26, `os/signal.NotifyContext`, `golang.org/x/term` (already a dependency).

**Milestone:** single-pass. One boundary, closed with `sdlc close` — no `Mx` tags.

---

## Non-goals

A REPL is where scope creeps, so the boundary is stated before any code:

- **No line editing, history, or completion.** No readline, no arrow keys, no
  `~/.define_history`. A word is short and retyping it is cheap; the moment this
  needs a line editor it needs a dependency, and that is a separate decision.
- **No multi-line input, no `:` commands.** One line is one word. `:forget` and
  friends arrive with the deck (#4), where there is something to forget.
- **No session state beyond the current word** — no per-session deck, no counters.
  The deck is #3/#4's job and must not be pre-empted here.
- **No pager, no screen clearing, no cursor control.** Output scrolls.

## Flags inside the loop

`-times`, `-locale`, `-no-audio`, `-raw` and `-no-color` are **session settings**:
parsed once, applied to every word. `define -times 1` with no word opens the loop
with single playback — `NArg() == 0` is the trigger, independent of flags.

## Cancellation contract

`main()` wraps the context with `signal.NotifyContext(ctx, os.Interrupt)`. This
changes the **shipped one-shot path** too, and that is intended: today Ctrl-C
during playback kills the process outright; afterwards it cancels the context,
`exec.CommandContext` stops `afplay`, deferred cleanup runs, and the process
exits 0 **printing nothing**. That last part is not incidental: killing `afplay`
is *how* cancellation is implemented, so the naive version reported
`define: afplay: signal: killed` and told the user their own keypress had failed. Stating it because it is a behaviour change to code that already
shipped, not a new-feature detail.

---

## Chunk 1: Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `replCommand` | `cmd/define/repl.go` | new |
| `parseREPLLine` | `cmd/define/repl.go` | new |

- **replCommand** — what one line of input means: `cmdDefine{word}`, `cmdReplay`, `cmdNothing`. (No `cmdQuit`: quitting is EOF or context cancellation, never a parsed line — an earlier draft listed one.)
  - **DRY rationale:** the loop's decision table in one testable place. Without it, "blank line means replay" is an `if` buried in an IO loop and only reachable through a fake terminal.
  - **Future extensions:** `:help`, `:forget` when the deck lands (#4) — new cases, same seam.

- **parseREPLLine(line string, hasCurrent bool) replCommand** — pure: text in, command out. `hasCurrent` is passed rather than read from session state, so the function has no memory and the "blank line with nothing to replay" case is a plain unit test.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `cachingAudioSource` | `cmd/define/fetch.go` | new | an `AudioSource` |
| `defineOnce` | `cmd/define/main.go` | new (extracted) | the existing lookup→render→speak path |
| `stdinIsTerminal` | `cmd/define/main.go` | new | `term.IsTerminal` on **stdin** |
| `repl` | `cmd/define/repl.go` | new | stdin + the session |

- **cachingAudioSource** — decorates any `AudioSource`, memoising by candidate-list key.
  - **Injected into:** applied by `repl` itself to whatever `AudioSource` it is handed. Not in `realDeps()`: wiring it there would leave the production composition untested, since every test builds its own deps. Wrapping inside `repl` means the test path and the production path are the same line of code.
  - **Why a decorator and not a map in the loop:** the cache sits *behind the seam*, so `fakeCDN.Requested()` — which already records every request in order — is the assertion. A map inside the loop would need a bespoke test.

- **defineOnce** — the existing body of `run()` after flag parsing, extracted verbatim: look up, render, print, speak.
  - **Injected into:** both `run()` (one-shot) and `repl`. This is the ARCH-DRY core of the issue — the REPL must not grow a parallel copy of the define path.
  - **`run` changes signature**, and this is the only signature change in the issue:

    ```go
    func run(args []string, d deps, stdout, stderr io.Writer) int                    // before
    func run(ctx context.Context, args []string, d deps, stdin io.Reader, stdout, stderr io.Writer) int  // after
    ```

    Both additions are forced. `ctx` because `run` currently manufactures
    `context.Background()` inline (`main.go:87`), which no test can cancel — the
    cancellation contract above is untestable until the caller owns it. `stdin`
    because the loop must read from somewhere a test can supply. Every existing
    call site in `main_test.go` updates mechanically; that churn is expected and
    is not the signal Task 1 Step 3 describes.

- **repl** — reads lines and dispatches commands.
  - **Owns no temp dir.** `speak` already creates and removes one per call, and an
    MP3 is a few KB, so re-writing it per replay is cheaper than owning session
    state. This keeps Task 1's extraction genuinely verbatim.
  - Takes an `io.Reader` and the writers — `repl(ctx, d, opt, stdin, stdout, stderr) int`.
    Interactivity is derived from `d.stdinIsTerminal`, not passed as a parameter, so
    `run`'s remains the only signature change in the issue.

- **stdinIsTerminal** — injected as a field on `deps` (not called directly), because otherwise "no args on a terminal" is unwritable as a test: the harness's stdin is never a TTY. Note this is a *different question* from the existing stdout check that drives colour.

### Test surface

`parseREPLLine` gets a table test with no IO. `repl` is driven end to end through the existing `fakeDictionary` / `fakeCDN` / `fakePlayer`, asserting on `Requested()` and `Played` — the same fakes #1 already ships, reused rather than re-made.

---

## Chunk 2: Tasks

### Task 1: Extract `defineOnce`

**Files:** modify `cmd/define/main.go`; test `cmd/define/main_test.go`

- [x] **Step 1: Run the existing suite** — `go test ./cmd/define/`, all green. This is a pure refactor; the existing CLI tests are the regression net.
- [x] **Step 2: Extract** the post-flag body of `run()` into `defineOnce(ctx, d, opt, word, stdout, stderr) int`, where `opt` carries `raw`, `color`, `noAudio`, `times`, `locale`.
- [x] **Step 3: Re-run the suite** — still green, no test changes. If a test needed changing, the extraction was not behaviour-preserving.

  This norm applies to **Task 1 only**. Task 5 deliberately changes behaviour and
  must therefore change a test: `TestRunNoArgsIsUsageError` (`main_test.go:87`)
  asserts that no-args exits 2, which is exactly what this issue removes. It gets
  **rewritten, not deleted** — into `TestRunNoArgsEntersTheLoop` — so the no-args
  branch keeps a test at all times and the diff shows the contract moving rather
  than a test quietly disappearing.
- [x] **Step 4: Commit** — `#2: extract defineOnce from run`

### Task 2: `parseREPLLine`

**Files:** create `cmd/define/repl.go`; test `cmd/define/repl_test.go`

- [x] **Step 1: Write the failing table test.** Obligations: a word → `cmdDefine`; leading/trailing space trimmed; blank with a current word → `cmdReplay`; blank with none → `cmdNothing`; a multi-word line (`hot dog`) → `cmdDefine` with both words, since multi-word headwords are real (#1 ships `hot dog` as a fixture).
- [x] **Step 2: Run, expect FAIL**
- [x] **Step 3: Implement.**
- [x] **Step 4: Run, expect PASS**
- [x] **Step 5: Commit** — `#2: parse REPL input lines`

### Task 3: `cachingAudioSource`

**Files:** modify `cmd/define/fetch.go`; test `cmd/define/fetch_test.go`

- [x] **Step 1: Write the failing test.** Obligations: two `Fetch` calls with the same candidate list return identical bytes and produce **exactly one** entry in `fakeCDN.Requested()`; a different word does hit the network; a failed fetch is **not** cached (so a transient outage does not poison the session).
- [x] **Step 2: Run, expect FAIL**
- [x] **Step 3: Implement** — a mutex-guarded map keyed by the joined candidate list.
- [x] **Step 4: Run, expect PASS**
- [x] **Step 5: Commit** — `#2: cache fetched audio behind the AudioSource seam`

### Task 4: The loop

**Files:** create `cmd/define/repl.go`; test `cmd/define/repl_test.go`

- [x] **Step 1: Write the failing tests**, driven through the existing fakes. The design decisions they pin: an unknown word leaves the *current* word unchanged so a following blank line replays the last good one; a blank line with nothing current is a hint, not an error; a replay costs zero CDN requests.

  Two adversarial classes get named guards rather than good-path coverage:
  - **A line over 64 KB.** `bufio.Scanner` stops with `ErrTooLong`, which looks
    exactly like EOF — a large paste would silently quit the loop. Raise the
    scanner's buffer cap and check `scanner.Err()` separately from the loop
    ending. It cannot be *recovered* from: once the scanner returns `ErrTooLong`
    every later `Scan()` returns false, so the loop reports and exits rather than
    continuing (an earlier draft said "report it, and continue", which is not
    implementable).
  - **The reader goroutine outlives `repl`** (it stays blocked on stdin after the
    loop returns on cancellation). It must share nothing mutable with the loop —
    the current word lives in the loop only. Run the package under `-race`.
- [x] **Step 2: Run, expect FAIL**
- [x] **Step 3: Implement.** `bufio.Scanner` in a goroutine feeding a channel; `select` on that channel and `ctx.Done()` so Ctrl-C is not blocked behind a pending read.

  **No session temp dir** — `speak` keeps creating and removing its own per call
  (Chunk 1). Raise the scanner's limit with `scanner.Buffer(make([]byte, 0, 64<<10), 1<<20)`
  rather than trying to recover from `ErrTooLong`: once the scanner returns that
  error every later `Scan()` returns false, so "report it and continue" would
  either spin or quit anyway. A line past 1 MB ends the loop with a diagnostic —
  the honest outcome, since the reader cannot be resumed.
- [x] **Step 4: Run, expect PASS**
- [x] **Step 5: Commit** — `#2: REPL loop`

### Task 5: Wire it up

**Files:** modify `cmd/define/main.go`; test `cmd/define/main_test.go`

- [x] **Step 1: Write the failing tests.** `NArg() == 0` → the loop, whether or not stdin is a terminal; `interactive` only controls the prompt. `echo sycophantic | define` now defines the word instead of printing usage — a **new** capability, asserted as such. Args present → one-shot, unchanged.
- [x] **Step 2: Run, expect FAIL**
- [x] **Step 3: Implement.** `main()` wraps `signal.NotifyContext(ctx, os.Interrupt)` and sets `deps.stdinIsTerminal`. The prompt (`› `) goes to **stdout** and only when interactive, so piped output carries none.
- [x] **Step 4: Run, expect PASS**
- [x] **Step 5: Manual check** — the one thing tests cannot cover:

```sh
make build
./bin/define              # type sycophantic, hear it; press return, hear it again
                          # (no second fetch — watch it start instantly)
                          # ^C exits; then: ls /var/folders/**/define-audio-* → none
echo sycophantic | ./bin/define   # one-shot, unchanged
```

- [x] **Step 6: Update `README.md` + `atlas/define.md`** — the REPL is new user-facing surface and a new entry mode.
- [x] **Step 7: Commit, then `sdlc close --issue 2 --verified '<evidence>'`**

---

## Risks

- **`go test -race ./cmd/define/` is part of Task 4's done**, not an afterthought — the reader goroutine is the only concurrency in this repo.
- **A blocking read swallows Ctrl-C.** Reading in a goroutine with a `select` on `ctx.Done()` is the mitigation; the cancelled-context test pins it.
- **Temp files leak on interrupt.** `signal.NotifyContext` returns normally rather than killing the process, so `defer os.RemoveAll` runs. Verified by hand in Task 5 Step 5 — a test cannot observe a real SIGINT cleanly.
- **The extraction in Task 1 silently changes behaviour.** Mitigated by refusing to touch the existing tests during it; a test that needs editing is the signal.

---

## Revisions

### 2026-08-20 — operator refinements + two close-review rounds

The shipped code has moved twice since Chunk 1 was written. Reconciling here
rather than leaving the plan describing a design that no longer exists.

**Replay is audio-only, then silent, then transient** (three operator steps).
`speak` writes nothing at all; `defineOnce` owns the "♫ playing N×" line; and the
indicator is *ephemeral* — shown while the sound plays, erased when it finishes,
because it acknowledges a keypress rather than recording anything. On a terminal
the replay indicator now **replaces the prompt**: the terminal has already echoed
Enter onto a new line, so the loop steps back over both that echo and the prompt
before drawing. While the sound plays there is no prompt, which is honest —
input is not accepted during playback.

**Cursor control is no longer a non-goal.** Chunk 1's Non-goals ruled it out;
the operator lifted it for exactly this (2026-08-20). `#14` inherits the rest of
that list (line editing, history, completion) and lifts those too.

**`playAnnounced` is new and unplanned.** The announce → play → erase → report
sequence existed twice and had diverged three ways: which terminal it gated on,
whether the audio-off guard applied, and a duplicated literal. One owner now,
with the erase style as the only intended difference (ARCH-DRY).

**`options.tty` is new.** There are three distinct terminal questions in this
tool and conflating any two is a bug: `color` (stdout, presentation), `tty`
(stdout, may I erase), and `stdinIsTerminal` (stdin, is there a human typing).
Gating cursor control on stdin alone leaked escapes into `define > out.txt`.

**Close review round 1 (4 Important).** Ctrl-C printed `define: afplay: signal:
killed` — a regression from this issue's own `signal.NotifyContext`, since
killing `afplay` is *how* cancellation is implemented. The >64 KB guard shipped
untested. `ErrNoAudio` was retried though it is permanent. `-h` documented only
the one-shot form.

**Close review round 2 (5 Important).** Escapes leaked into a redirected stdout;
a failed replay wrote its diagnostic onto the redrawn prompt and then swallowed
the next one; the duplicated sequence above; the Ctrl-C guard was still dead to
the suite; and this section did not exist.

**Close review round 3 (4 Important).** A word with no recording announced
playback that never happened, and on a pipe the false line persisted —
`playAnnounced` had moved the announcement ahead of `speak`. The rule now: an
*erasable* indicator is ephemeral UI and may be optimistic, because a failure
takes it back; a *non-erasable* one is a record, and a record has to be true.
Also: `echo rizz | define` exited 0 where `define rizz` exits 1, though README
presents them as interchangeable; and `-no-color` still emitted cursor-control
escapes, so `options.tty` now derives from `!noColor && isTerminal(stdout)` —
the flag means "no ANSI", not "no colour".

**Line-level reconciliations (flagged at every prior boundary, landed here):**
`cmdQuit` removed from the `replCommand` bullet (it never existed); `repl`'s
signature corrected — interactivity comes from `d.stdinIsTerminal`, not a
parameter; and Task 4's "report it, and continue" corrected, which was also an
undisposed plan-gate carry-forward (PQ-10).

**Two testing lessons, both from assertions that could not fail:**

1. `replRig` hard-coupled the stdin and stdout terminal checks, commented as
   "the real-world pairing." That assumption *was* the blind spot — it made the
   mismatched-streams bug unreachable from the suite. Test helpers must not
   couple the conditions whose disagreement is the defect.
2. The first attempt at the failed-replay test captured stdout and stderr
   separately, where the fixed and broken versions emit identical bytes. The
   defect only exists in the *interleaving*, so the test tees both into one
   buffer — which is what a terminal actually is.

### 2026-08-20 — close rounds 4 and 5

**Round 4 (4 Important).** `-raw` meant two things — it returned before playing
one-shot while the loop's replay branch ignored it and fetched; decided once at
flag parse now. The record form had no shape assertion: mutating its trailing
newline away left the suite green, which would glue `define word | cat` output to
whatever followed. Three atlas sentences were false, one contradicting its own
heading. And the erase arithmetic was documented as unconditional when it assumes
no input arrives during playback — cooked-mode echo moves the cursor and the
erase then clears the wrong line.

**Round 5 (3 Important).** Two are the same story told twice more:

- **One family, third instance.** UI is written to STDOUT but was gated on
  STDIN. Round 2 caught it for cursor control, round 4 caught the atlas
  describing the weaker gate, round 5 caught the *prompt* — so
  `define > out.txt` from a terminal polluted the file and showed the human
  nothing. Fixed once now: a single `terminalUI := interactive && opt.tty`
  governs every byte of interactive UI. `interactive` alone survives only where
  the question genuinely is about stdin (whether a failed lookup sets the exit
  code). **Three rounds to state a one-line rule** is the cost `ariadne#195`
  exists to remove.
- **A recorded fix that never landed.** Round 4's commit message says a false
  atlas sentence was deleted; it was not — the replacement text did not match and
  nothing checked. It survived at `atlas:210` and `plan:74` into round 5. This is
  the same defect class as the plan-gate carry-forward in round 3, and the same
  one `ariadne#195` describes: a disposition asserted rather than verified.
