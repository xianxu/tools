# `/play` from inside the loop — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `/play` runs today's sitting from the definition prompt and returns to
it — on finishing the queue, and on Ctrl-C.

**Architecture:** The command RECORDS the intent and the loop PERFORMS the
sitting — `cc.replay`'s shape one size up — through the `playSession` `--play`
already calls.

**It is NOT "no new machinery", which the first draft claimed.** Plan-quality
round 1 found two Criticals in that claim, and both are about who owns the
terminal. A sitting entered from the loop needs a console that does NOT hand the
terminal back when it ends, and it needs the REPL's screen to stop painting while
it runs. Neither exists today. The new machinery is small and it is named below,
rather than discovered during implementation.

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
| `enterAlt` is idempotent | `rawterm.go:147-150` returns early when `r.alt` | building a second console over the live `rawSession` does NOT take a second alternate screen. This is the ONLY half of the terminal question that was already safe — the teardown half was not, see below |
| `console.finish` RESTORES the shared session | `finish: onceHandBack(live, sess, stdout)` (`replraw.go:85`) → `handBack` = `live.Stop(); sess.restore(); print transcript` (`replraw.go:235-242`) | a nested console's `finish` would put the terminal back into cooked mode MID-REPL. PQ-1 |
| `runEditor` never sees the `rawSession` | its parameters are `(ctx, keys, interrupts, d, opt, con)` (`replraw.go:253`); `sess` is `replRaw`'s local (`replraw.go:25`) | the performing half cannot build a console where it stands. PQ-2 |
| `liveScreen.Stop` is TERMINAL, and there is an async painter | `Stop` sets `stopped = true` with no resume (`screen.go:841-852`); `l.timer = time.AfterFunc(…, l.flush)` (`screen.go:719`) | the REPL's screen cannot be paused and resumed today, and a pending throttled paint would flush INTO the sitting's frame. PQ-4 |
| one goroutine reads the terminal | `readKeys(ctx, f, interrupts)`, `replraw.go:32` | `/play` MUST reuse that channel. A second reader on one fd races for bytes, and each would see half the keystrokes |
| a nil capability is how a command refuses | `commandCtx.setTimes`/`setLang`, `command.go:160-175`; "a nil setTimes means /sound must REFUSE" | `/play` is nil outside a raw loop, which handles one-shot, piped and line-mode in one rule |
| the interrupter scopes Ctrl-C to something narrower | `interrupter.Set` (`interrupt.go:34`), built in `#16` for a streaming answer | "Ctrl-C ends the sitting, not the program" is `Set`/`restore`, not a new path |

### The three things that must be built

**1. A console that borrows the terminal, with its own `finish` — NOT `handBack`
with a no-op restorer.** That was the first revision's answer and it is wrong:
`handBack` is `live.Stop(); sess.restore(); print transcript` (`replraw.go:239-241`),
and the ORDER is the point — the transcript is printed after the terminal is back
in cooked mode and out of the alternate screen. A no-op restorer keeps the order
and breaks its precondition, printing raw text into the alternate screen.

So the sitting's `finish` is a different three lines:

```go
// stop this screen's painter, then hand the summary UP rather than out.
sitting.Stop()
repl.Write(sitting.Transcript())
```

**And that answers where the summary lands** (PQ-1's second half): into the
REPL's own buffer, so it is there in the scrollback when the loop resumes and the
learner can page back to it — which is what `--play` achieves by printing into
the normal buffer on exit, reached the only way that works when the REPL still
owns the terminal.

**2. A way for the loop to build one.** `runEditor` has no `sess`, and widening
its signature would leak the terminal back into a function whose whole point is
that it does not have one (`replraw.go:244`: *"the editor loop with the terminal
factored out"*). So `console` carries the factory instead — built by
`newConsole`, where `sess` IS in scope:

```go
// newSitting builds a console for a full-screen sitting on THIS terminal, or is
// nil where one cannot run. The closure holds the rawSession so runEditor never
// has to.
newSitting func() console
```

**3. Suspend and resume on `liveScreen`.** Two screens will share one terminal,
and `Stop` is one-way. The REPL's screen must stop PAINTING for the duration —
including its throttled `time.AfterFunc` flush, which would otherwise land in the
middle of a sitting's frame — and then resume with its buffer and viewport
intact. `Stop` is not that: it is the end of a screen's life.

This is the smallest piece that cannot be avoided, and it is why this issue is
not the table row it first looked like.

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `runPlayCommand` | `cmd/define/play_cmd.go` | new |
| `commandCtx` | `cmd/define/command.go` | modified |

- **`runPlayCommand(c commandCtx, args []string) int`** — the command. Refuses
  arguments, refuses when the capability is nil, otherwise records the intent.
  - **A NIL DECK IS ALSO A REFUSAL** (PQ-3). `todaysQuestions` reads
    `d.deck.Deck()`, so a sitting without one panics rather than degrading. The
    capability being non-nil says the TERMINAL can host a sitting; it says
    nothing about there being a deck to review. Two questions, two checks — and
    the deck one uses `noDeckMessage(c.noCapture)`, which `#8` established is
    unanimous across five callers.
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
| `applyShape` | `cmd/define/replraw.go` | new | the terminal's shape |

- **`sittingInPlace(ctx, d, opt, sess, keys, interrupts, stdout, stderr) int`** —
  runs one sitting on a terminal that is ALREADY raw.
  - **The sibling is `replayInPlace`** (`replraw.go:506`), and the name is
    deliberate: same position in the loop, same "on the terminal we already
    hold" contract.
  - **It does NOT call `newConsole`** (PQ-8). `newConsole` acquires three things
    a borrower must not take, and the first revision's row called it anyway while
    the prose above described the opposite:

    | `newConsole` does | why a borrower must not |
    |---|---|
    | `finish: onceHandBack(live, sess, stdout)` (`replraw.go:85`) | restores the SHARED session and prints the transcript to a cooked terminal — the precondition PQ-1 is about |
    | `watchResize(ctx, …)` (`replraw.go:72`) | a SECOND SIGWINCH goroutine over one terminal, outliving the sitting |
    | `sess.enterMouse()` (`replraw.go:67`) | already reported; asking twice is a second acquisition of a thing already held |

    So `newSitting` assembles the `console` struct directly — a new pinned screen
    over the same tty, the REPL's own `resizes` channel BORROWED, and the
    three-line `finish` above. `enterAlt` is not called either: the loop is
    already in the alternate screen, and idempotence is a safety net rather than
    a reason to ask.
  - **Injected into:** nothing. It is the performing half.

**Test surface.**

- `runPlayCommand` — pure over `commandCtx`, unit-tested directly.
- `liveScreen.suspend`/`resume` — unit-tested against a buffer tty with a real
  timer: write, suspend, assert NOTHING more reaches the buffer even after the
  throttle window elapses, resume, assert the frame returns. That last clause is
  what distinguishes suspend from `Stop`, and a test without it passes on a
  `Stop` in disguise.
- `sittingInPlace` — needs a terminal, so the **pty test** under the `pty` tag,
  plus an assertion through the STORE that a sitting entered this way records
  what `--play` records. Not through a fake capturer: `#12`'s sweep found a fake
  proves the outcome reaches *a* capturer and nothing about what it writes.

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
- **CONCURRENCY IS NOT N/A, and the first draft said it was** (PQ-4). Two
  independent asynchronous things exist per console, and a sitting must add
  exactly one of them:

  **The painter.** Each `liveScreen` carries a throttled `time.AfterFunc` →
  `flush` (`screen.go:719`) that paints without the loop asking. During a sitting
  there are two screens over one terminal, so the REPL's is SUSPENDED — timer
  disarmed, `pending` preserved, not flushed (flushing would put the REPL's frame
  on top of the sitting's) — and resumed after, buffer and viewport intact.

  **The resize watcher.** `newConsole` starts one goroutine per console
  (`watchResize`, `replraw.go:72`). A sitting must NOT start a second: two
  watchers on one SIGWINCH means the shape is delivered to two channels and the
  suspended screen acts on it. The sitting BORROWS the REPL's `resizes` channel,
  so there is exactly one watcher for the process's life, owned by `replRaw`.

  A resize during a sitting therefore arrives on that one channel, is read by the
  sitting's loop, and resizes the sitting's screen; the suspended screen takes the
  new shape on `resume`, which repaints unconditionally anyway.
- **EXTENT — who is still running when `sittingInPlace` returns.** Nothing of the
  sitting's: its screen is stopped (timer disarmed, no goroutine outlives the
  call) and its console is dropped. The REPL's screen is resumed and its timer
  re-arms on the next write. The ONE thing that spans both is the key reader
  goroutine, which belongs to `replRaw` and is borrowed rather than duplicated —
  it is still running because it was running before, and `sittingInPlace` never
  owned it.
- **Still N/A, stated:** durable state (the sitting's writes are `capture`'s,
  unchanged, and this issue adds none) and retry.

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

- [x] **Step 1: Write the failing tests.** Three, named for what each asserts —
      written as prose because a plan citing a test that does not exist yet turns
      the suite red (see Verification):

      1. **a nil capability is the refusal**, which is how `/sound` and `/lang`
         already handle "there is no session here" (`command.go:160`). One rule
         covers the one-shot path, a pipe, and the line-mode REPL.
      2. **a nil DECK is a second, separate refusal** — `todaysQuestions` reads
         `d.deck.Deck()`, so the capability being present says the terminal can
         host a sitting and nothing about there being one to review.
      3. **it takes no argument**, the rule `--play` and `--stats` state, and it
         RECORDS rather than performs: the command returns, the loop acts.

- [x] **Step 2: Run them, watch them fail.**
- [x] **Step 3: Add the row, the field and the function.**
- [x] **Step 4: Run them, watch them pass.**
- [x] **Step 5: The atlas's command list is DERIVED** (`TestDocsQuoteTheCommandList`,
      `doc_sync_test.go:354`) and will fail until it quotes the new row. Update it;
      that failure is the guard working.
- [x] **Step 6: Commit.**

## Chunk 2: the sitting

### Task 2: `sittingInPlace`

**Files:**
- Create: the function in `cmd/define/play_cmd.go`
- Modify: `cmd/define/play_loop.go` — extract what `runPlay` and this share

- [x] **Step 1: Read `runPlay` and take the part below the guards.** The
      questions, the console, the `playSession` call. If that means a helper both
      call, write the helper — a second way to start a sitting is what this issue
      exists to not create.
- [x] **Step 2: Write the failing pty test.** Type `/play`, answer a question,
      Ctrl-C, and assert the DEFINITION PROMPT is back — not a shell. And the
      suspend/resume rows named in the Test surface, including the clause that
      separates suspend from `Stop`: after resume the frame comes BACK.
- [x] **Step 3: Implement.**
- [x] **Step 4: The scoped interrupt.** `interrupts.Set(cancel)` with a deferred
      `restore()`, and a test that Ctrl-C inside the sitting does not cancel the
      loop's context.
- [x] **Step 5: Assert through the STORE that both entry points record the same
      thing** — a review answered via `/play` and via `--play` produce the same
      event. Not through a fake capturer: `#12`'s mutation sweep found that a fake proves the
      outcome reaches *a* capturer and nothing about what it writes.
- [x] **Step 6: The derived guard the issue asks for:** nothing may call
      `enterRaw` while a session is live. Parse the call sites the way
      `TestEveryWriteWordsCallSitePassesAVocabulary` does
      (`deckwords_test.go`), and fail closed on the count.
- [x] **Step 7: Mutation sweep.** Remove the `Set`/`restore` pair and confirm the
      Ctrl-C row reddens; point `/play` at `runPlay` and confirm the
      enterRaw guard reddens.
- [x] **Step 8: README + atlas.**
- [x] **Step 9: Commit, then `sdlc close --issue 48`.**

---

## Verification

1. `go test -count=1 ./...`, `go vet` under default, `pty` and `conformance`,
   `gofmt -l` clean.
2. The nil-capability rule — a refusal where no sitting can run.
3. The suspend/resume rows, including the clause that separates suspend from
   `Stop`: after resume, the frame comes back.
4. The pty test: `/play`, answer, Ctrl-C, back at the prompt.
5. The store assertion: both entry points record identically.
6. The `enterRaw` guard, mutation-swept.

**The test NAMES are deliberately un-backticked above and in the tasks below.**
`TestPlanCitesTestsThatExist` (`repo_guard_test.go:1306`) walks every plan in the
tree and requires every backticked test name in it to exist — so a plan written before its
code, cited in its own convention, turns the suite red for whatever issue is
closing. This plan tripped that guard while citing it. The names are written as
prose until the tests land; the tasks say what each asserts, which is what a
reader needs anyway.
6. **Manual, once:** `/play` in the smoke deck — answer a question, Ctrl-C, look a
   word up, `/play` again. A terminal is a thing a person has to see.

## Revisions

### 2026-09-08 — plan-quality round 1: 2 Critical, 2 Important

**The architecture line was wrong, and that is the finding.** It said "no new
machinery… joining three existing seams without opening a fourth". Two Criticals
say otherwise, and both are about terminal OWNERSHIP — the half I checked was
setup (`enterAlt` is idempotent, so no second alternate screen) and the half I
did not check was teardown.

- **PQ-1** — `console.finish` is `onceHandBack(live, sess, stdout)`, and
  `handBack` calls `sess.restore()`. A nested console ending would hand the
  shared terminal back to cooked mode mid-REPL. Fixed by passing a no-op
  restorer: `handBack`'s parameter is already an interface, so borrowing rather
  than owning is an argument, not a branch.
- **PQ-2** — `runEditor` has no `rawSession`; it is `replRaw`'s local, and the
  function's own doc says it is "the editor loop with the terminal factored out".
  So the console carries a `newSitting` factory built where `sess` is in scope,
  and the terminal stays factored out.
- **PQ-3** — the nil-capability refusal covered "no terminal" and not "no deck",
  which `todaysQuestions` dereferences. Two questions, two checks.
- **PQ-4** — "no concurrency" was false: each screen has a throttled painter and
  the loop watches SIGWINCH, so a pending flush would land inside a sitting's
  frame. `liveScreen` needs suspend/resume; `Stop` is one-way and is the end of a
  screen's life, not a pause.

**What this changes about the issue's shape.** It was filed as three seams
meeting, and the operator read it as straightforward — as did I. The command half
is straightforward. The terminal half needs one genuinely new capability
(suspend/resume on a screen) and two small ones, all three named above rather
than met during implementation.

### 2026-09-08 — plan-quality round 2

Two findings survived round 1's revision, and both survived for the same reason:
I answered the half of the question I had checked.

**PQ-1 — the no-op restorer broke a precondition instead of a rule.** `handBack`
is `live.Stop(); sess.restore(); print transcript`, and the ORDER is load-bearing:
the transcript is printed once the terminal is back in cooked mode and out of the
alternate screen. Handing it a restorer that does nothing keeps the order and
removes the thing the order was for, so raw text would land inside the alternate
screen. The sitting's `finish` is now its own three lines, and the summary goes
UP into the REPL's buffer rather than out to the terminal — which also answers
the half of PQ-1 the first revision left unwritten: where the summary lands.

**PQ-4 — the extent clause.** "Two painters" was handled; "who is still running
when `sittingInPlace` returns" was not. Written now: nothing of the sitting's
outlives the call, the REPL's screen resumes, and the one thing spanning both is
the key reader goroutine — borrowed, never owned, still running because it was
running before.

**PQ-5 — the new capability had no spec.** `suspend`/`resume` was named as "the
smallest piece that cannot be avoided" and then given no entity row, no test and
no state model. It has all three now, including the clause that distinguishes it
from `Stop`: after `resume`, the frame comes BACK. A test without that clause
passes on a `Stop` in disguise.

**PQ-6 — a Done-when row rested on a false claim about existing guards.** It said
the README's command list is guarded by a derived test. It is not: the guard
reads `atlas/define.md`. `#8` learned that by adding `/stats` and watching which
document failed, and this plan restated the wrong version anyway — the
file:line rule at the top of this document exists precisely for that.

### 2026-09-08 — close review rounds 1 and 2

**BR-5 (Critical) and BR-14 are one bug found twice**, and the second is the
reason to record them together. The sitting BORROWS the resize channel — one
watcher for the process, which is right — and a borrowed channel is CONSUMED, not
shared: a SIGWINCH during a sitting is read by the sitting and reaches the loop by
no other route. This plan claimed "the suspended screen takes the new shape on
resume". It does not; resume repaints, and repainting does not change rows and
cols.

Round 1 fixed the instance — hand the screen's shape back — and round 2 found the
class: the loop also derives `opt.width` from a shape, with a below-the-floor
policy, so an entry looked up after a sitting wrapped at the pre-sitting width.
`applyShape` is now the one place a shape means anything, called by both routes,
and `TestBothShapeRoutesGoThroughOnePlace` derives that from the source so a
third route is covered when it arrives.

**BR-7 took four wrong tests, each of which PASSED.** Feeding
`Key{KeyInterrupt}` into the channel bypasses the interrupter entirely; waiting
on a `HasScope` helper the test had itself made true fired before the sitting was
in the picture; racing two goroutines over one `bytes.Buffer` hung; and asserting
only the restore passed with `Set` AND `restore` both deleted, because never
scoping also leaves the loop's cancel installed. What works orders the fire
deterministically by making the sitting consume a resize first — a channel read
the test can observe, unlike a buffer it must not race — and asserts BOTH halves.

**BR-8** — a test named `...FromEitherDoor` called `playSession` twice and
invoked neither, with a comment rationalising it. **BR-6** — the three refusals
this plan named in Chunk 1 were never written. **BR-9** — the atlas had no entry
for the borrow design. **BR-15** — these boxes.

**The entity table gained what shipped**: `applyShape`, and `sittingInPlace`'s
signature returning the shape via NAMED returns, because a plain return evaluates
before the deferred hand-back and would return the pre-sitting size — the very
bug it exists to fix, one level in.
