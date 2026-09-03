---
id: 000045
status: open
deps: []
github_issue:
created: 2026-09-03
updated: 2026-09-03
estimate_hours:
---

# Play pronunciation without blocking the UI

## Problem

Pronunciation playback blocks the interactive loops. `playAnnounced`
(`cmd/define/main.go:892`) calls `speak` synchronously; `speak`
(`main.go:991`) fetches the recording over the network, writes it to a temp
file, and calls `playN`, which runs `afplay` to completion —
`exec.CommandContext(...).Run()` (`player.go:36`) — with a 250ms gap between
repeats. At the default repeat count the terminal is unresponsive for the whole
fetch plus several seconds of audio.

It blocks in seven places: the REPL (`repl.go:316`), the raw REPL
(`replraw.go:608,636,654`), the review loop (`play_loop.go:506`) and the
one-shot path (`main.go:720`).

## Spec

**Play in a goroutine so the loops stay responsive.** Four things make this
more than moving a call, and all four are load-bearing today.

### 1. The temp file is deleted by `speak`'s own defer

`speak` does `defer os.RemoveAll(dir)` (`main.go:996`) and plays inside that
scope. Return before playback finishes and the file is removed while `afplay`
is reading it. **Whatever owns the playback must own the temp directory's
lifetime** — pass ownership to the goroutine and delete after the last repeat,
including on the error and cancellation paths.

### 2. Two things block, and only one is the request

The fetch is network-bound and variable; playback is bounded and predictable.
Making only playback async still stalls the UI for the fetch. **Recommend both
move**, with the caveat in (3) about what may then be reported.

### 3. Announcements are deliberately ordered around completion

`playAnnounced`'s comments state the invariant: an erasable indicator is
"ephemeral UI and may be optimistic", while a non-erasable one (a pipe, or
`-no-color`) "is a RECORD, and a record has to be true: announced only after
something actually played. Otherwise `define <word-with-no-recording> >
out.txt` files a claim that it played three times when it played none."
`reportVoice` likewise runs after success, so it reports the voice that
*answered* rather than the one requested (`#29`).

Going async breaks this unless the ordering is preserved: the goroutine — not
the caller — must own erase, `reportVoice`, and the failure message, writing
them when the outcome is known. **Do not relax the invariant to make the
refactor easy.** A pipe that claims playback which never happened is the exact
regression these comments exist to prevent.

Note the writes then move off the main goroutine: `stdout`/`stderr` need a
single writer or a mutex, since the loops are drawing at the same time.

### 4. The one-shot path must still wait

`define <word>` (`main.go:720`) exits after printing. If playback is async and
`main` returns, the process dies and takes `afplay` with it — silence. **The
one-shot path must wait for playback before exiting**; only the interactive
loops benefit from async. Simplest split: playback returns a handle, the loops
ignore it, the one-shot path waits on it.

### 5. Concurrency policy, to decide in the plan

Today a second playback cannot start because the first blocks. Async makes
overlap possible: in the review loop, advancing to the next word while audio
plays would run two `afplay` processes. **Recommend cancel-previous** — one
playback at a time, a new request cancels the old — which matches what the
synchronous behavior felt like and needs no queue. `playN` already takes a
`ctx` and stops between repeats (`player.go:51-55`), so cancellation lands
where it already works.

Ctrl-C must keep its current meaning: a cancelled context is the user, not an
error (`main.go:914-918`).

## Done when

- In the REPL, the raw REPL and the review loop, the next keystroke is accepted
  while a pronunciation is still playing.
- `define <word>` plays to completion before the process exits, with the audio
  audible to the end — asserted, since this is what a naive async change breaks.
- The temp file outlives playback: a test asserts the file still exists when the
  player is invoked for the last repeat, and is gone afterwards.
- On a pipe (`-no-color`), nothing is announced unless playback actually
  happened, and nothing is announced *before* it happened — the `#29` record
  invariant, unchanged.
- `reportVoice` still names the URL that answered.
- Starting a new playback while one is in flight leaves exactly one player
  running; the superseded one is cancelled, not queued.
- Ctrl-C during playback prints no error.
- Concurrent stdout writes from the playback goroutine and a drawing loop do
  not interleave mid-line.

## Plan

- [ ] Move temp-dir ownership into the playback path so the file outlives the
      player.
- [ ] Extract the announce → play → erase → report sequence so the goroutine
      owns the whole thing, keeping the ordering in (3).
- [ ] Serialize writer access for the off-main writes.
- [ ] Cancel-previous policy, using the existing `ctx` in `playN`.
- [ ] Handle for the one-shot path to wait on; loops discard it.
- [ ] Tests: responsiveness in each loop, one-shot completion, temp-file
      lifetime, pipe-record invariant, single-player-at-a-time, Ctrl-C.

## Log

### 2026-09-03

Raised as "play sound in a separate goroutine, not blocking main UI".

The existing synchronous shape is not accidental — `playAnnounced` is
documented as "the single owner of the announce → play → erase → report
sequence" (`main.go:884`), and that ownership is what makes the pipe-record
invariant hold. The async version has to keep the ownership and move it, rather
than splitting the sequence across a goroutine boundary.

The `Player` interface stays as it is: the repeat loop already lives outside it
"so the number of plays is a property of the shell that a fake can count"
(`player.go:18-20`), which is exactly what makes the concurrency policy
testable against `player_fake_test.go` without real audio.
