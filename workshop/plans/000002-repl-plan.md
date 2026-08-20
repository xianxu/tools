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
exits 0. Stating it because it is a behaviour change to code that already
shipped, not a new-feature detail.

---

## Chunk 1: Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `replCommand` | `cmd/define/repl.go` | new |
| `parseREPLLine` | `cmd/define/repl.go` | new |

- **replCommand** — what one line of input means: `cmdDefine{word}`, `cmdReplay`, `cmdQuit`, `cmdNothing`.
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
  - **Failed fetches are not cached**, so a transient outage does not poison the rest of the session.

- **defineOnce** — the existing body of `run()` after flag parsing, extracted verbatim: look up, render, print, speak.
  - **Injected into:** both `run()` (one-shot) and `repl`. This is the ARCH-DRY core of the issue — the REPL must not grow a parallel copy of the define path.

- **repl** — reads lines and dispatches commands.
  - **Owns no temp dir.** `speak` already creates and removes one per call, and an
    MP3 is a few KB, so re-writing it per replay is cheaper than owning session
    state. This keeps Task 1's extraction genuinely verbatim.
  - Takes an `io.Reader`, the writers, and an `interactive bool`, so tests drive it
    from a string with no terminal anywhere.

- **stdinIsTerminal** — injected as a field on `deps` (not called directly), because otherwise "no args on a terminal" is unwritable as a test: the harness's stdin is never a TTY. Note this is a *different question* from the existing stdout check that drives colour.

### Test surface

`parseREPLLine` gets a table test with no IO. `repl` is driven end to end through the existing `fakeDictionary` / `fakeCDN` / `fakePlayer`, asserting on `Requested()` and `Played` — the same fakes #1 already ships, reused rather than re-made.

---

## Chunk 2: Tasks

### Task 1: Extract `defineOnce`

**Files:** modify `cmd/define/main.go`; test `cmd/define/main_test.go`

- [ ] **Step 1: Run the existing suite** — `go test ./cmd/define/`, all green. This is a pure refactor; the existing CLI tests are the regression net.
- [ ] **Step 2: Extract** the post-flag body of `run()` into `defineOnce(ctx, d, opt, word, stdout, stderr) int`, where `opt` carries `raw`, `color`, `noAudio`, `times`, `locale`.
- [ ] **Step 3: Re-run the suite** — still green, no test changes. If a test needed changing, the extraction was not behaviour-preserving.
- [ ] **Step 4: Commit** — `#2: extract defineOnce from run`

### Task 2: `parseREPLLine`

**Files:** create `cmd/define/repl.go`; test `cmd/define/repl_test.go`

- [ ] **Step 1: Write the failing table test.** Obligations: a word → `cmdDefine`; leading/trailing space trimmed; blank with a current word → `cmdReplay`; blank with none → `cmdNothing`; a multi-word line (`hot dog`) → `cmdDefine` with both words, since multi-word headwords are real (#1 ships `hot dog` as a fixture).
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#2: parse REPL input lines`

### Task 3: `cachingAudioSource`

**Files:** modify `cmd/define/fetch.go`; test `cmd/define/fetch_test.go`

- [ ] **Step 1: Write the failing test.** Obligations: two `Fetch` calls with the same candidate list return identical bytes and produce **exactly one** entry in `fakeCDN.Requested()`; a different word does hit the network; a failed fetch is **not** cached (so a transient outage does not poison the session).
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement** — a mutex-guarded map keyed by the joined candidate list.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#2: cache fetched audio behind the AudioSource seam`

### Task 4: The loop

**Files:** create `cmd/define/repl.go`; test `cmd/define/repl_test.go`

- [ ] **Step 1: Write the failing tests**, driven through the existing fakes. The design decisions they pin: an unknown word leaves the *current* word unchanged so a following blank line replays the last good one; a blank line with nothing current is a hint, not an error; a replay costs zero CDN requests.

  Two adversarial classes get named guards rather than good-path coverage:
  - **A line over 64 KB.** `bufio.Scanner` stops with `ErrTooLong`, which looks
    exactly like EOF — a large paste would silently quit the loop. Check
    `scanner.Err()` separately from the loop ending, report it, and continue.
  - **The reader goroutine outlives `repl`** (it stays blocked on stdin after the
    loop returns on cancellation). It must share nothing mutable with the loop —
    the current word lives in the loop only. Run the package under `-race`.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** `bufio.Scanner` in a goroutine feeding a channel; `select` on that channel and `ctx.Done()` so Ctrl-C is not blocked behind a pending read. One `os.MkdirTemp` for the session, removed by `defer`.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#2: REPL loop`

### Task 5: Wire it up

**Files:** modify `cmd/define/main.go`; test `cmd/define/main_test.go`

- [ ] **Step 1: Write the failing tests.** `NArg() == 0` → the loop, whether or not stdin is a terminal; `interactive` only controls the prompt. `echo sycophantic | define` now defines the word instead of printing usage — a **new** capability, asserted as such. Args present → one-shot, unchanged.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** `main()` wraps `signal.NotifyContext(ctx, os.Interrupt)` and sets `deps.stdinIsTerminal`. The prompt (`› `) goes to **stdout** and only when interactive, so piped output carries none.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Manual check** — the one thing tests cannot cover:

```sh
make build
./bin/define              # type sycophantic, hear it; press return, hear it again
                          # (no second fetch — watch it start instantly)
                          # ^C exits; then: ls /var/folders/**/define-audio-* → none
echo sycophantic | ./bin/define   # one-shot, unchanged
```

- [ ] **Step 6: Update `README.md` + `atlas/define.md`** — the REPL is new user-facing surface and a new entry mode.
- [ ] **Step 7: Commit, then `sdlc close --issue 2 --verified '<evidence>'`**

---

## Risks

- **`go test -race ./cmd/define/` is part of Task 4's done**, not an afterthought — the reader goroutine is the only concurrency in this repo.
- **A blocking read swallows Ctrl-C.** Reading in a goroutine with a `select` on `ctx.Done()` is the mitigation; the cancelled-context test pins it.
- **Temp files leak on interrupt.** `signal.NotifyContext` returns normally rather than killing the process, so `defer os.RemoveAll` runs. Verified by hand in Task 5 Step 5 — a test cannot observe a real SIGINT cleanly.
- **The extraction in Task 1 silently changes behaviour.** Mitigated by refusing to touch the existing tests during it; a test that needs editing is the signal.
