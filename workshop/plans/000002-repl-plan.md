# define REPL — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bare `define` on a terminal opens a loop: type a word to define and hear it, press return to hear it again, Ctrl-C to quit.

**Architecture:** The one-shot path is factored into `defineOnce`, and the loop calls it — the REPL adds a reader and a session, not a second implementation of `define` (ARCH-DRY). Replay-without-refetch comes from a `cachingAudioSource` **decorator** on the existing `AudioSource` seam, so the fake CDN's request recorder proves the cache works rather than a hand-inspected log.

**Tech Stack:** Go 1.26, `os/signal.NotifyContext`, `golang.org/x/term` (already a dependency).

**Milestone:** single-pass. One boundary, closed with `sdlc close` — no `Mx` tags.

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
| `repl` | `cmd/define/repl.go` | new | stdin + the session |

- **cachingAudioSource** — decorates any `AudioSource`, memoising by candidate-list key for the process lifetime.
  - **Injected into:** `realDeps()` for the REPL only; the one-shot path fetches once anyway, so caching there is dead weight.
  - **Why a decorator and not a map in the loop:** the cache then sits *behind the seam*, so `fakeCDN.Requested()` — which already records every request in order — is the assertion that a replay makes no second request. A map inside the loop would need its own bespoke test.

- **defineOnce** — the existing body of `run()` after flag parsing, extracted verbatim: look up, render, print, speak.
  - **Injected into:** both `run()` (one-shot) and `repl`. This is the ARCH-DRY core of the issue — the REPL must not grow a parallel copy of the define path.

- **repl** — reads lines, dispatches commands, owns the session temp dir.
  - Takes an `io.Reader` and writers, so tests drive it with a scripted script and no terminal.

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

- [ ] **Step 1: Write the failing tests.** Obligations, driven through the fakes:
  - `"sycophantic\n\n"` → 2 plays-sets (6 plays at the default 3×) and **1** CDN request.
  - `"sycophantic\nephemeral\n"` → 2 CDN requests; the second word becomes current.
  - a blank first line → a hint on stderr, no lookup, loop continues.
  - an unknown word → diagnostic on stderr, loop continues, current word unchanged.
  - EOF (the reader runs out) → returns 0.
  - a cancelled context → returns promptly without consuming further input.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** `bufio.Scanner` in a goroutine feeding a channel; `select` on that channel and `ctx.Done()` so Ctrl-C is not blocked behind a pending read. One `os.MkdirTemp` for the session, removed by `defer`.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#2: REPL loop`

### Task 5: Wire it up

**Files:** modify `cmd/define/main.go`; test `cmd/define/main_test.go`

- [ ] **Step 1: Write the failing tests.** Obligations: no args + a TTY-ish stdin → the loop; no args + **piped** stdin → the one-shot path, so `echo sycophantic | define` still works and stays scriptable; args present → one-shot regardless.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** `main()` wraps `signal.NotifyContext(ctx, os.Interrupt)`. The TTY test is on **stdin**, not stdout — colour keys off stdout and these are different questions.
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

- **A blocking read swallows Ctrl-C.** Reading in a goroutine with a `select` on `ctx.Done()` is the mitigation; the cancelled-context test pins it.
- **Temp files leak on interrupt.** `signal.NotifyContext` returns normally rather than killing the process, so `defer os.RemoveAll` runs. Verified by hand in Task 5 Step 5 — a test cannot observe a real SIGINT cleanly.
- **The extraction in Task 1 silently changes behaviour.** Mitigated by refusing to touch the existing tests during it; a test that needs editing is the signal.
