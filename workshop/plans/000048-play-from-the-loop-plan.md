# `/play` from inside the loop — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `/play` runs today's sitting from the definition prompt and returns to
it — on finishing the queue, and on Ctrl-C.

**Architecture:** No new machinery. The command RECORDS the intent and the loop
PERFORMS the sitting, which is `cc.replay`'s shape one size up; the sitting runs
on the terminal the loop already holds, through the `playSession` `--play`
already calls. The work is joining three existing seams without opening a fourth.

**Tech Stack:** Go 1.24. No new dependencies. `cmd/define` only.

---

## Core concepts

### Every claim below carries a file:line

`#8`'s plan broke this four times, so the rule is inherited: a declarative
sentence about existing code is verified before it is written. The citations are
the evidence.

### The precedent this issue follows exactly

`/pron` already solved this problem for a smaller verb. `replraw.go:491-508`:

> **RECORDED here, PERFORMED below.** … *a command decides WHAT to replay and the
> loop owns replaying, so `/pron` and a bare Enter remain one path.*

`cc.replay` is set only when the loop can honour it (`if sess.hasCurrent()`,
`replraw.go:499`), the command assigns into a local, and the loop performs it
after `dispatchCommand` returns. **The same sentence, one verb up: the loop owns
running a sitting, so `--play` and `/play` remain one path.**

That shape is not a stylistic preference here. Performing the sitting *inside*
`dispatchCommand` would nest a full-screen loop inside a command's lifetime,
while the keys channel, the console and the interrupter are all locals of
`runEditor` (`replraw.go:253`).

### What is already true, checked

| fact | where | consequence |
|---|---|---|
| `playSession` is the separable half of `runPlay` | `play_loop.go:105` calls it after the guards, `enterRaw` and the console | `/play` calls `playSession`, never `runPlay` |
| both loops build their console from ONE builder | `newConsole` (`replraw.go:47`), called at `replraw.go:33` with `newLiveScreen` and `play_loop.go:106` with `newPinnedScreen` | the sitting's console is the same builder with the other screen constructor — the difference `newConsole`'s doc says "IS the whole difference" |
| `enterAlt` is idempotent | `rawterm.go:147-150` returns early when `r.alt` | building a second console over the live `rawSession` does NOT take a second alternate screen — the hazard the issue feared is already closed |
| one goroutine reads the terminal | `readKeys(ctx, f, interrupts)`, `replraw.go:32` | `/play` MUST reuse that channel. A second reader on one fd races for bytes, and each would see half the keystrokes |
| a nil capability is how a command refuses | `commandCtx.setTimes`/`setLang`, `command.go:160-175`; "a nil setTimes means /sound must REFUSE" | `/play` is nil outside a raw loop, which handles one-shot, piped and line-mode in one rule |
| the interrupter scopes Ctrl-C to something narrower | `interrupter.Set` (`interrupt.go:34`), built in `#16` for a streaming answer | "Ctrl-C ends the sitting, not the program" is `Set`/`restore`, not a new path |

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `runPlayCommand` | `cmd/define/play_cmd.go` | new |
| `commandCtx` | `cmd/define/command.go` | modified |

- **`runPlayCommand(c commandCtx, args []string) int`** — the command. Refuses
  arguments, refuses when the capability is nil, otherwise records the intent.
  - **DRY rationale:** it decides nothing about the sitting. Everything a sitting
    is lives in `playSession` and `todaysQuestions` already.

- **`commandCtx`** *(modified)* — gains one nil-able field.
  ```go
  // startSitting asks the loop to run today's review, and is nil wherever a
  // sitting cannot be run: the one-shot path, a pipe, and the line-mode REPL,
  // which has no raw terminal (replLines, repl.go:249). Nil is the refusal —
  // the rule setTimes states at command.go:160.
  startSitting func()
  ```
  - **Why a `func()` and not a `bool`:** the loop supplies the closure only when
    it can honour it, so "can I" and "do it" are one fact rather than two that
    can disagree. That is `cc.replay`'s shape (`replraw.go:499`).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `sittingInPlace` | `cmd/define/play_cmd.go` | new | the live terminal |

- **`sittingInPlace(ctx, d, opt, sess, keys, interrupts, stdout, stderr) int`** —
  runs one sitting on a terminal that is ALREADY raw.
  - **The sibling is `replayInPlace`** (`replraw.go:506`), and the name is
    deliberate: same position in the loop, same "on the terminal we already
    hold" contract.
  - **It builds the questions and the console, then calls `playSession`.**
    `runPlay`'s body above that call is guards + `enterRaw` + console; here the
    guards are the loop's already, the terminal is held, and only the console
    differs — `newConsole(ctx, d, sess, stdout, newPinnedScreen)`.
  - **Injected into:** nothing. It is the performing half.

**Test surface.** `runPlayCommand` is pure over `commandCtx` and unit-tested
directly. `sittingInPlace` needs a terminal, so its coverage is the
**pty test** the repo already runs under the `pty` tag, plus assertions through
the store that a sitting entered this way records what `--play` records.

### ARCH-ORDER — the events that matter

- **Ctrl-C during a sitting** — `interrupter.Set` installs the sitting's cancel
  and `restore()` hands the loop's back. The failure to avoid is a Ctrl-C that
  escapes the sitting and kills the program, and the one to avoid *harder* is a
  `restore` that does not run: it is a `defer`.
- **Ctrl-C between the command and the sitting** — the window where the intent is
  recorded but the sitting has not started. It cancels the loop, as it does
  today; the recorded intent dies with the iteration because it is a local.
- **A sitting that ends with no questions** — `todaysQuestions` can return none
  (nothing due). The command must say so at the prompt rather than flashing a
  screen; `--play` already has that sentence (`emptyQueueReason`, `play_loop.go:1422`, printed at `play_loop.go:920`).
- **Not applicable, stated:** no concurrency (one loop, one reader goroutine, and
  the sitting borrows it rather than starting a second); no durable state of its
  own (the sitting's writes are `capture`'s, unchanged); no retry.

### ARCH-CONSTRAINTS — the envelope

A sitting is bounded by `opt.count` (default 20, `main.go:451`). Entering one
from the loop adds a console build and a `todaysQuestions` call — one file per
deck word and one per day of log, which `--play` already pays on every
invocation. Nothing here runs per keystroke.

---

## Chunk 1: the command

### Task 1: `/play` records, and refuses where it cannot run

**Files:**
- Create: `cmd/define/play_cmd.go`, `cmd/define/play_cmd_test.go`
- Modify: `cmd/define/command.go` (the row + the field)

- [ ] **Step 1: Write the failing tests**

```go
// A NIL CAPABILITY IS THE REFUSAL, which is how /sound and /lang already handle
// "there is no session here" (command.go:160). It covers the one-shot path, a
// pipe, and the line-mode REPL in one rule rather than three checks.
func TestSlashPlayRefusesWhereItCannotRun(t *testing.T)

// It takes no argument, the rule --play and --stats already state.
func TestSlashPlayRefusesArguments(t *testing.T)

// It RECORDS rather than performs: the command returns and the loop acts.
func TestSlashPlayRecordsTheIntent(t *testing.T)
```

- [ ] **Step 2: Run them, watch them fail.**
- [ ] **Step 3: Add the row, the field and the function.**
- [ ] **Step 4: Run them, watch them pass.**
- [ ] **Step 5: The atlas's command list is DERIVED** (`TestDocsQuoteTheCommandList`,
      `doc_sync_test.go:354`) and will fail until it quotes the new row. Update it;
      that failure is the guard working.
- [ ] **Step 6: Commit.**

## Chunk 2: the sitting

### Task 2: `sittingInPlace`

**Files:**
- Create: the function in `cmd/define/play_cmd.go`
- Modify: `cmd/define/play_loop.go` — extract what `runPlay` and this share

- [ ] **Step 1: Read `runPlay` and take the part below the guards.** The
      questions, the console, the `playSession` call. If that means a helper both
      call, write the helper — a second way to start a sitting is what this issue
      exists to not create.
- [ ] **Step 2: Write the failing pty test.** Type `/play`, answer a question,
      Ctrl-C, and assert the DEFINITION PROMPT is back — not a shell.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: The scoped interrupt.** `interrupts.Set(cancel)` with a deferred
      `restore()`, and a test that Ctrl-C inside the sitting does not cancel the
      loop's context.
- [ ] **Step 5: Assert through the STORE that both entry points record the same
      thing** — a review answered via `/play` and via `--play` produce the same
      event. Not through a fake capturer: `#12`'s mutation sweep found that a fake proves the
      outcome reaches *a* capturer and nothing about what it writes.
- [ ] **Step 6: The derived guard the issue asks for:** nothing may call
      `enterRaw` while a session is live. Parse the call sites the way
      `TestEveryWriteWordsCallSitePassesAVocabulary` does
      (`deckwords_test.go`), and fail closed on the count.
- [ ] **Step 7: Mutation sweep.** Remove the `Set`/`restore` pair and confirm the
      Ctrl-C row reddens; point `/play` at `runPlay` and confirm the
      enterRaw guard reddens.
- [ ] **Step 8: README + atlas.**
- [ ] **Step 9: Commit, then `sdlc close --issue 48`.**

---

## Verification

1. `go test -count=1 ./...`, `go vet` under default, `pty` and `conformance`,
   `gofmt -l` clean.
2. `TestSlashPlayRefusesWhereItCannotRun` — the nil-capability rule.
3. The pty test: `/play`, answer, Ctrl-C, back at the prompt.
4. The store assertion: both entry points record identically.
5. The `enterRaw` guard, mutation-swept.
6. **Manual, once:** `/play` in the smoke deck — answer a question, Ctrl-C, look a
   word up, `/play` again. A terminal is a thing a person has to see.
