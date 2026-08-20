# Boundary Review — tools#2 (whole-issue close)

| field | value |
|-------|-------|
| issue | 2 — define REPL: bare invocation reads words, defines and speaks them |
| repo | tools |
| issue file | workshop/issues/000002-repl.md |
| boundary | whole-issue close |
| milestone | — |
| window | 65e91604bc62f5017131f80ab17f46302beca82e..HEAD |
| command | sdlc close --issue 2 |
| reviewer | claude |
| timestamp | 2026-08-20T15:13:04-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The diff delivers the issue's actual purpose: bare `define` opens a loop, a bare return replays audio and writes literally nothing to stdout, and the "no second fetch" property is proved through the pre-existing `fakeCDN.Requested()` recorder rather than bespoke scaffolding. `go vet` is clean and `go test -race -count=1 ./cmd/define/` passes (10.5s). The ARCH-DRY core — one `defineOnce` shared by both entry modes — is real, not asserted; `parseREPLLine` is genuinely pure and table-tested with no IO; and putting `cachingAudioSource` *inside* `repl` (repl.go:61) rather than in `realDeps()` means production and test wiring are the same line, which is the cleanest resolution of the gate's PQ-5. Nothing here is Critical. What holds it back from a bare SHIP is the one behaviour the plan gate explicitly deferred to a manual check that didn't cover it: Ctrl-C now emits a spurious error diagnostic on stderr, on both the REPL quit path and the *shipped* one-shot path — a user-visible regression to code that already existed. Alongside that, the plan's own named adversarial guard (a >64 KB line) shipped with zero test coverage, and the binary's `-h` text still documents only the one-shot form.

## 1. Strengths

- **`cmd/define/repl.go:61`** — the cache decorator applied inside `repl` instead of `realDeps()`. This is the difference between "the fake proves it" and "a test hand-wires something production doesn't do." A regression that dropped the decorator would now fail `TestREPLBareReturnReplaysWithoutRefetching`.
- **`cmd/define/main.go:130-136`** — moving the `♫ playing N×` line out of `speak` into the caller is the right seam for the "replay writes nothing" contract. `speak` is silent *by construction*, so the loop cannot accidentally reprint; the test at `repl_test.go:180` (byte-identical stdout across two replays) is the strongest possible statement of it.
- **`cmd/define/repl.go:52-58`** — reading stdin unconditionally and making only the *prompt* TTY-conditional eliminates the interactive/batch branch entirely. The whole loop is drivable from `strings.NewReader`, which is why the test file needs no pty.
- **`cmd/define/main_test.go:87-90`** — `TestRunNoArgsIsUsageError` rewritten rather than deleted, with a comment saying why. The contract change is visible in the diff instead of a test quietly vanishing.
- **`cmd/define/repl.go:96-99`** — "only a successful lookup becomes the current word" is a real design decision (a typo doesn't cost you the word you were listening to), and `TestREPLUnknownWordLeavesCurrentUnchanged` pins it via player count rather than by inspecting internal state.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — Ctrl-C prints a spurious error diagnostic; affects the shipped one-shot path too.**
`cmd/define/main.go:127` and `cmd/define/repl.go:95`

Neither call site distinguishes cancellation from failure. With `signal.NotifyContext` now in `main()`, a SIGINT during playback takes this path: `exec.CommandContext` kills `afplay` → `player.go:36` wraps it as `afplay: signal: killed` → `playN` → `speak` → `defineOnce` prints `define: afplay: signal: killed`. Cancelling during the inter-play gap yields `define: context canceled` (`player.go:53`); during the fetch, `define: sycophantic: audio fetch failed: …: context canceled`.

So `define sycophantic` + Ctrl-C, which previously died silently, now prints a confusing failure line and exits **0**. The Spec says "Ctrl-C exits cleanly"; the plan's Cancellation contract says "the process exits 0" — neither anticipates the diagnostic. This is exactly the residue the gate flagged: PQ-4's round-2 disposition reads "*The stderr line Ctrl-C now emits is left to the manual check*", and Task 5 Step 5's manual check inspects only the exit and leftover temp dirs, never stderr. The deferral was never actually collected.

Fix sketch — suppress at both report sites:
```go
if err := speak(ctx, d, word, opt.locale, opt.times); err != nil && ctx.Err() == nil {
    fmt.Fprintf(stderr, "define: %s\n", err)
}
```
Same guard at `repl.go:95`. Then state the resulting contract ("Ctrl-C prints nothing and exits 0") in the plan's Cancellation contract section and in `atlas/define.md`'s Entry-modes paragraph, which currently describes the cancellation change without mentioning output.

**I-2 — The named >64 KB adversarial guard shipped with no test.**
`cmd/define/repl.go:76-80` and `cmd/define/repl.go:126`

Plan Task 4 Step 1 is checked `[x]` and commits to "*Two adversarial classes get named guards rather than good-path coverage*", the first being a line over 64 KB. The mechanical guard is in the code (`sc.Buffer(..., maxLineBytes)` plus `sc.Err()` reported separately from EOF), but `grep` finds no test touching `maxLineBytes`, `ErrTooLong`, or the `reading input` message. The entire `err != nil → exit 1` branch at repl.go:77-80 is dead to the suite. This is precisely the bug class the guard exists to prevent — a large paste silently ending the loop, indistinguishable from EOF — and it is currently unpinned.

Fix sketch:
```go
func TestREPLOverlongLineIsReportedNotSilentEOF(t *testing.T) {
    rig, opt := replRig(t, "sycophantic", true, false)
    var out, errb bytes.Buffer
    huge := strings.Repeat("a", maxLineBytes+1) + "\n"
    if code := repl(t.Context(), rig.deps, opt, strings.NewReader(huge), &out, &errb); code != 1 {
        t.Errorf("exit = %d, want 1 — an unreadable line must not look like EOF", code)
    }
    if !strings.Contains(errb.String(), "reading input") { … }
}
```

**I-3 — `ErrNoAudio` is treated as retryable, so replaying a word with no recording re-issues every candidate request.**
`cmd/define/fetch.go:104-108`

The comment justifies not caching failures with "a transient outage must not poison the rest of the session" — correct for `ErrFetchFailed`, wrong for `ErrNoAudio`, which is permanent. `AudioCandidates` returns 4 URLs (`audiourl.go:39-46`), so each bare return on a recording-less word costs 4 HTTP round-trips that can never succeed, and the user's only feedback is the same stderr line repeated. The package already draws this exact distinction and documents why it matters (`fetch.go:15-24`); the cache is the natural consumer of it and currently ignores it (ARCH-DRY: the error taxonomy is the single source, and this new code doesn't derive from it).

Fix sketch:
```go
if err != nil {
    if errors.Is(err, ErrNoAudio) {
        c.mu.Lock(); c.misses[key] = struct{}{}; c.mu.Unlock() // permanent: no recording exists
    }
    return nil, "", err // ErrFetchFailed stays retryable
}
```
Add a fourth case to the existing `fetch_test.go` trio asserting that a second `Fetch` after `ErrNoAudio` makes no new requests, while an `ErrFetchFailed` still retries.

**I-4 — `-h` help text still documents only the one-shot form.**
`cmd/define/main.go:71`

`usage: define [flags] <word>` is unchanged, so the binary's own help — the first place a user looks — says nothing about bare invocation being the friendly entry point. README.md and atlas/define.md were both updated; the in-binary surface was missed, which is the same class of gap the docs gate is meant to catch, just one layer in. Fix: `usage: define [flags] [word]`, plus one line ("With no word, reads words from stdin; on a terminal this is an interactive loop — return replays, Ctrl-C quits.").

## 4. Minor findings

- `repl.go:111` — "nothing to replay: audio is off" also fires for `-times 0`, where audio is not off; the message misdescribes half the branch it guards.
- `-raw` + REPL asymmetry: `defineOnce` returns at `main.go:115-118` before the audio block, so typing a word under `-raw` plays nothing — but a bare return then *does* play it, and pays a fetch because the cache was never warmed. Undeclared in the Spec's "flags are session settings".
- `repl_test.go:158` — the timeout arm of `TestREPLCancelledContextReturnsPromptly` selects on `t.Context().Done()`, which cannot fire while the test function is blocked in that very select. A regression hangs to the 10-minute `go test` panic instead of failing "promptly". Use `time.After(2*time.Second)`.
- `player_fake_test.go:18` — `fakePlayer.Play` discards its `ctx`, so no test can drive a cancel-during-playback through the REPL. This is structurally why I-1 is invisible to a green suite.
- `repl.go:70` and `repl.go:82` — `if interactive { fmt.Fprintln(stdout) }` duplicated across the two exit arms; a single deferred/named `finish()` would read better.
- `main.go:33` — `isTerminal(w io.Writer)` is now called on `os.Stdin`. It works (`*os.File` satisfies `io.Writer`), but the signature now lies about its use; `func isTerminal(f any) bool` or an `*os.File` parameter would be honest.
- A second Ctrl-C is swallowed: `stop()` is deferred to `main` exit, so `NotifyContext` keeps absorbing SIGINT and there's no force-quit if playback ever hangs. The usual idiom calls `stop()` on first signal.
- `cachingAudioSource.hits` is unbounded for the session lifetime. A few KB per word makes this fine in practice — noting only so it isn't rediscovered later.
- README.md:41-44 wraps mid-sentence at an odd column ("…and Ctrl-C quits. The prompt appears / only on a terminal"); reflow.

## 5. Test coverage notes

Coverage of the *stated* contracts is genuinely good, and the tests assert through fakes rather than restating the implementation: player counts, `cdn.Requested()` lengths, and byte-identical stdout are all external observations. `TestREPLReplayWritesNothingToStdout` comparing two whole runs is a notably strong formulation.

Gaps, in priority order:
1. The read-error / `ErrTooLong` branch (`repl.go:77-80`) — zero coverage despite being a named plan deliverable (I-2).
2. Cancel-during-playback — unreachable while `fakePlayer` ignores `ctx`; wiring `ctx` into the fake would let I-1 be pinned as a test rather than a manual pty check.
3. `-raw` inside the REPL — no test; the asymmetry above went unnoticed.
4. `defineOnce`'s `-times 0` path through `replay` — no test.
5. `parseREPLLine`'s table omits a line that is a single interior-whitespace run vs. a tab-only line; `"   \t "` is covered, so this is marginal.

The `-race` requirement in Done-when is satisfied and the reader-goroutine sharing model (channels carry values; `current` lives only in `repl`) holds up on inspection — `d` is a value copy, so the `d.audio` rewrite at repl.go:61 is not shared either.

## 6. Architectural notes

- **ARCH-DRY — pass.** `defineOnce` is the single define path; the loop calls it rather than reimplementing it, which was the issue's stated architectural point. `cachingAudioSource` is a decorator on the existing `AudioSource` seam rather than a parallel map. One trivial duplication noted in Minor. The one place DRY is *under*-served is I-3: the `ErrNoAudio`/`ErrFetchFailed` distinction is an existing source of truth the new cache doesn't derive from.
- **ARCH-PURE — pass.** `parseREPLLine` is deterministic, memory-free (`hasCurrent` passed in, not read from state), and its table test touches no IO, no exec, no fs. `repl` is a thin shell over it. One observation for later: the "is there anything to replay" policy at `repl.go:110-112` is a *decision* living in the IO shell; when `:` commands arrive with #4, that check belongs in `parseREPLLine` (as an `audioOn bool` parameter) so the command table stays the single decision point rather than splitting across two files.
- **ARCH-PURPOSE — pass, shadow-sweep clean.** Every Done-when item is delivered and verified by something other than the implementor's word: the loop opens on no-args, replay costs one CDN request across two definitions, `echo … | define` is asserted as new capability, `-race` is clean, and the rewritten no-args test keeps that branch covered. Consumers of the new `options` struct: `run`, `defineOnce`, `repl`, `replay` — all four derive from the single parse at `main.go:83-89`; no hand-maintained restatement of the flag set survives. The one purpose-adjacent shortfall is I-4 — the binary's own help is a consumer of "what invocations exist" that was left as a stale restatement while README and atlas were updated.
- **ARCH-MOCK — pass with a note.** The new external surface is exercised entirely behind existing seams: `cachingAudioSource` is tested against the stateful `fakeCDN` (which records requests *in order*), and the REPL runs end-to-end against `fakeDictionary`/`fakeCDN`/`fakePlayer`. Conformance checks exist for the dictionary, fetch, and player. The note: the Ctrl-C contract now depends on `exec.CommandContext`'s kill-on-cancel behaviour, and `player_conformance_test.go` covers only `TestAfplayBlocksUntilPlaybackCompletes` — there's no live conformance check that a cancelled context actually terminates a real `afplay` mid-file. That is the one real-binary behaviour this issue newly depends on and does not verify against the real binary. Worth a conformance case; not blocking, since the manual pty check exercised it once.

## 7. Plan revision recommendations

Four items for a `## Revisions` entry on `workshop/plans/000002-repl-plan.md` (append, don't overwrite, per AGENTS.md §1):

1. **Chunk 1, `replCommand` bullet** — enumerates `cmdDefine{word}`, `cmdReplay`, **`cmdQuit`**, `cmdNothing`. The shipped type (`repl.go:18-23`) has only `cmdNothing`, `cmdDefine`, `cmdReplay`; quitting is EOF or context cancellation, never a parsed line. The code is right and the plan is stale — record that `cmdQuit` was dropped and why.
2. **Chunk 1, `repl` bullet** — "*Takes an `io.Reader`, the writers, and an `interactive bool`*". The shipped signature is `repl(ctx, d, opt, stdin, stdout, stderr) int`; interactivity is derived from `d.stdinIsTerminal` inside the function (`repl.go:56`). Since the plan elsewhere states that `run`'s signature change is "the only signature change in the issue", this discrepancy should be corrected rather than left implying a parameter that doesn't exist.
3. **Task 4 Step 1** — still instructs "*Check `scanner.Err()` separately from the loop ending, report it, and continue.*" Step 3 and the implementation deliberately do the opposite (raise the cap; a line past 1 MB ends the loop with a diagnostic). PQ-10's round-3 disposition explicitly said "*drop Step 1's leftover 'report it, and continue' clause when writing the test*" — that action was not performed, so the plan now contradicts itself and the ledger records a disposition that isn't true on disk. Drop the clause and note the test obligation from I-2.
4. **Cancellation contract section** — state what Ctrl-C *prints*, not just what it returns. This closes PQ-4's round-2 residue, which was parked on a manual check that never inspected stderr. Pair with the I-1 fix so the plan and the code agree that Ctrl-C is silent.

No revision needed to the issue's Spec or Done-when — those match the code as shipped.

---

## Re-review — 2026-08-20T15:25:53-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 2 — define REPL: bare invocation reads words, defines and speaks them |
| repo | tools |
| issue file | workshop/issues/000002-repl.md |
| boundary | whole-issue close |
| milestone | — |
| window | 65e91604bc62f5017131f80ab17f46302beca82e..HEAD |
| command | sdlc close --issue 2 |
| reviewer | claude |
| timestamp | 2026-08-20T15:25:53-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All four Important findings from the earlier boundary review are genuinely fixed in the code: the Ctrl-C suppression guards are present at both report sites, the >64 KB guard is now pinned by two tests, `ErrNoAudio` is cached while `ErrFetchFailed` stays retryable (I mutation-tested the cache — flipping it to cache all failures does fail `TestCachingAudioSourceDoesNotCacheTransportFailures`), and `-h` documents the loop. `go vet` is clean, `go test -race -count=1 ./cmd/define/` passes (13.1s), and `GOOS=linux CGO_ENABLED=0 go build ./...` is green. Every Done-when item is delivered and verified by something other than the implementor's word. What holds it back from SHIP is the operator's two screen-control refinements, which landed after the last review and introduced a second copy of the "announce → speak → erase → report" sequence that has silently diverged from the first in three ways — two of which I reproduced as real misbehaviour: ANSI escapes leak into a redirected stdout when stdin is still a terminal (`define | tee log`), and a failed interactive replay prints its diagnostic *on top of* the redrawn prompt and then suppresses the next prompt entirely. Neither is a crash and both are cheap; alongside them, the plan and Spec still describe cursor control as a non-goal and the prior review's four plan revisions were never applied.

## 1. Strengths

- **`cmd/define/repl.go:75`** — the cache decorator applied inside `repl` rather than in `realDeps()`. Production and test wiring are the same line, so `fakeCDN.Requested()` is a real assertion rather than scaffolding. Confirmed by mutation: this is what makes `TestREPLBareReturnReplaysWithoutRefetching` meaningful.
- **`cmd/define/fetch.go:122-132`** — the cache derives permanence from the existing `ErrNoAudio`/`ErrFetchFailed` taxonomy instead of re-deciding what "failed" means. I mutated it to cache every failure; the test suite caught it immediately.
- **`cmd/define/main.go:154-157`** — `speak` prints nothing *by construction*. That's what makes "a replay writes nothing to stdout" structurally true rather than a discipline, and `TestREPLReplayWritesNothingToStdout` (comparing two whole runs byte-for-byte) is the strongest available formulation of it.
- **`cmd/define/repl.go:30-41`** — `parseREPLLine` with `hasCurrent` passed in rather than read from state. Genuinely pure, table-tested, no IO anywhere in the test. This is the ARCH-PURE core of the diff.
- **`cmd/define/repl_test.go:219-227`** — `TestREPLLongButReadableLineIsJustAWord` alongside the over-cap test. Pinning that the guard is not *over*-eager, not just that it fires, is the kind of pairing that survives refactoring.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — Cursor control is gated on stdin being a TTY, so ANSI escapes leak into a redirected stdout.**
`cmd/define/repl.go:111-118`

`repl` derives `interactive` from `d.stdinIsTerminal` (repl.go:71) and uses it to gate both the flash and `eraseLineAndStepBack`, which are written to **stdout**. `defineOnce` gates the same class of output on `opt.tty` (stdout). The two disagree whenever the streams differ — `define | tee log`, `define > out.txt` — which is an ordinary interactive-capture pairing, not a corner case.

Reproduced with a throwaway probe (stdin TTY, `opt.tty` false), stdout was:

```
…  ♫ playing 3×\n›   ♫ playing 3×\r\x1b[K\x1b[A\r\x1b[K› \n
```

`atlas/define.md:186` claims "Gated on interactive: piped output contains no escape sequence, asserted" and `atlas/define.md:177` claims "Piped output keeps the indicator as a plain line and emits no escape sequence, asserted"; `README.md:43` says "piping stays clean". `TestREPLNonInteractiveReplayEmitsNoEscapes` only covers the both-non-terminal case because `replRig` (`repl_test.go:34-41`) hard-couples `tty` to `interactive` — its comment, "the real-world pairing", is the assumption that hides this.

Fix sketch — the step-back is only meaningful when the tty echoed Enter *and* stdout is that tty, so both conditions are required:

```go
canErase := interactive && opt.tty
if canErase { fmt.Fprintf(stdout, "  ♫ playing %d×", opt.times) }
…
if canErase { fmt.Fprint(stdout, eraseLineAndStepBack+prompt); skipPrompt = true }
```

Then give `replRig` independent `interactive`/`tty` parameters and add a mismatched-streams case asserting no `\x1b[` in stdout.

**I-2 — A failed interactive replay writes its diagnostic onto the redrawn prompt, then eats the next prompt.**
`cmd/define/repl.go:116-121`

The prompt is redrawn and `skipPrompt` set *before* the error is reported. Probed with stdout and stderr teed to one buffer (the real-terminal arrangement), `define -no-audio` + Enter produces:

```
…\r\x1b[K\x1b[A\r\x1b[K› define: nothing to replay: audio is off\n
```

So the user sees `› define: nothing to replay: audio is off`, and because `skipPrompt` is true the next iteration draws no prompt at all — they type into a bare line. The same applies to a transient `ErrFetchFailed` on replay, which is the more likely trigger in real use.

Fix sketch — erase, report, and only claim the prompt when there was nothing to report:

```go
if canErase { fmt.Fprint(stdout, eraseLineAndStepBack) }
if err != nil && ctx.Err() == nil {
    fmt.Fprintf(stderr, "define: %s\n", err)   // leaves skipPrompt false: loop redraws normally
} else if canErase {
    fmt.Fprint(stdout, prompt)
    skipPrompt = true
}
```

**I-3 — ARCH-DRY: the announce → speak → erase → report sequence exists twice and has diverged three ways.**
`cmd/define/main.go:135-149` and `cmd/define/repl.go:111-121`

Both sites run the identical four-step shape. The copies differ in: (a) the terminal gate (`opt.tty` vs `interactive`) — that divergence *is* I-1; (b) the erase sequence (`eraseLine` vs `eraseLineAndStepBack+prompt`) — legitimately different, but nothing marks it as the only intended difference; (c) the `!opt.noAudio && opt.times > 0` guard, present in `defineOnce` and absent around the replay flash — which is why `-no-audio` still flashes "♫ playing 3×" before erroring. The `"  ♫ playing %d×"` literal is duplicated, and `TestREPLReplayFlashesThenRestoresThePrompt` counts `♫` occurrences across both, so a change to one silently shifts the other's assertion.

Fix sketch: one helper owning the sequence, with the erase style as its only parameter:

```go
func playAnnounced(ctx context.Context, d deps, opt options, word string, erase string, stdout, stderr io.Writer)
```

`defineOnce` passes `eraseLine`; the replay path passes `eraseLineAndStepBack+prompt`. This closes I-1, I-2 and the `-no-audio` flash in one edit rather than three.

**I-4 — The Ctrl-C suppression guard — the regression the last review caught — still has no test.**
`cmd/define/main.go:147`, `cmd/define/repl.go:119`

`fakePlayer.Play` (`player_fake_test.go:18`) discards its `ctx`, so no test drives a cancel-during-playback. Both `ctx.Err() == nil` guards are dead to the suite; re-deleting them is green. This is the exact bug class that shipped once already.

It's cheap to cover without touching the fake — a pre-cancelled context makes `Fetch` fail through `rebasedSource`, so the guard is exercised end to end:

```go
func TestDefineOnceSuppressesDiagnosticOnCancellation(t *testing.T) {
    rig := newAudioRig(t, "sycophantic", true)
    ctx, cancel := context.WithCancel(t.Context()); cancel()
    var out, errb bytes.Buffer
    if code := defineOnce(ctx, rig.deps, options{times: 3, locale: "us"}, "sycophantic", &out, &errb); code != 0 {
        t.Fatalf("exit = %d", code)
    }
    if errb.Len() != 0 {
        t.Errorf("Ctrl-C printed a diagnostic: %q", errb.String())
    }
}
```

**I-5 — Plan and Spec contradict the shipped code; the prior review's four revisions were never applied, and no artifact has a `## Revisions` section.**

AGENTS.md §1 requires an appended `## Revisions` entry when a plan artifact is revised mid-stream. `grep "## Revisions"` returns nothing in either `workshop/plans/000002-repl-plan.md` or `workshop/issues/000002-repl.md`, while the code has moved twice since the plan was written. Live contradictions:

- `plan:28` — "**No pager, no screen clearing, no cursor control.** Output scrolls." The tool now ships `eraseLine` and `eraseLineAndStepBack`, and the whole point of the second refinement is that output *doesn't* scroll. `repl.go:60` correctly records the operator lifting it; the plan does not.
- `plan:56` — `replCommand` is documented as including `cmdQuit`; the shipped type (`repl.go:19-23`) has only `cmdNothing`/`cmdDefine`/`cmdReplay`.
- `plan:96` — "Takes an `io.Reader`, the writers, and an `interactive bool`"; the shipped signature is `repl(ctx, d, opt, stdin, stdout, stderr) int`, deriving interactivity internally.
- `plan:154` — still says "report it, and continue", which Step 3 and the implementation deliberately contradict. PQ-10's own round-3 disposition in the gate ledger reads "*drop Step 1's leftover 'report it, and continue' clause when writing the test*" — the test was written, the clause was not dropped, so the ledger now records a disposition that isn't true on disk.
- **Cancellation contract** section still describes only the exit code, not that Ctrl-C is silent — the residue PQ-4 parked on a manual check.
- Issue Spec, `workshop/issues/000002-repl.md` — the bullet still asserts a bare return "writes **nothing to stdout at all** — no definition, no '♫ playing' line". Interactively it now writes a flash plus escapes and erases them. The *settled screen* is unchanged, which is the intent, but the Spec as written is false and the Log's operator-refinement notes at the bottom don't amend it.

## 4. Minor findings

- `README.md:37` — `-no-color` is documented as "never emit ANSI", but `defineOnce` emits `\r\x1b[K` on a tty regardless of `-no-color`. Narrow the wording (or gate `opt.tty` on `!noColor`).
- `repl.go:136` — "nothing to replay: audio is off" also fires for `-times 0`, where audio is not off.
- `-raw` inside the loop: `defineOnce` returns at `main.go:122-125` before the audio block, so typing a word plays nothing — yet a bare return then *does* play it, and pays a fetch the cache never warmed.
- `repl.go:71` — `d.stdinIsTerminal != nil` silently defaults to non-interactive, contrary to `main_test.go:11-14`'s stated norm that "a nil one would make run() panic rather than fail a test".
- `repl_test.go:160` — the timeout arm selects on `t.Context().Done()`, which cannot fire while the test is blocked in that very select. A regression hangs to the package timeout instead of failing "promptly"; use `time.After(2*time.Second)`.
- `atlas/define.md:200` — "Failed fetches are not cached" is now unqualified-false (`ErrNoAudio` *is* cached, per the sentence three lines above), and "a transient outage does not poison a session" appears twice in the same paragraph.
- `repl.go:88-90` / `repl.go:97-99` — duplicated `if interactive { fmt.Fprintln(stdout) }` across the two exit arms.
- `main.go:177` — `isTerminal(w io.Writer)` is now called on `os.Stdin`; it works, but the parameter name no longer describes the use.
- A second Ctrl-C is swallowed: `stop()` is deferred to `main` exit, so there is no force-quit if playback ever hangs. The usual idiom calls `stop()` on the first signal.
- `cachingAudioSource.hits` is unbounded for the session. Fine in practice at a few KB per word; noting so it isn't rediscovered.
- The close window bundles unrelated tracker work (commit `cbcd30c`: `#14`/`#15` issue files and project rows) with `#2`'s implementation. Harmless, but it widens the boundary being reviewed.

## 5. Test coverage notes

The tests assert through fakes rather than restating the implementation — player counts, `cdn.Requested()` lengths, byte-identical stdout — which is the right posture, and I verified two of them survive mutation. Gaps, in priority order:

1. **Cancel-during-playback / the `ctx.Err() == nil` guards** — zero coverage (I-4). Both guards can be deleted with a green suite, and this is the one bug that already shipped.
2. **Mismatched stream terminality** — `replRig` (`repl_test.go:34-41`) makes `tty == interactive` structurally, so I-1 is invisible. Parameterize the two.
3. **Interactive error paths** — no test runs a replay failure with `interactive=true`, which is why I-2 went unnoticed. Assert stderr ordering relative to the prompt redraw.
4. `-raw` inside the REPL — untested; the audio asymmetry above went unnoticed.
5. `defineOnce`'s `opt.times == 0` route into `replay` — untested.

`-race` is clean, and the reader-goroutine sharing model holds up on inspection: `d` is a value copy, so the `d.audio` rewrite at `repl.go:75` isn't shared, and the channels carry values only.

## 6. Architectural notes

- **ARCH-DRY — flag (I-3).** `defineOnce` as the single define path is real and is the issue's architectural win; `cachingAudioSource` as a decorator on the existing seam rather than a map in the loop is the right call. But the two operator refinements re-introduced duplication at exactly the level the issue set out to eliminate, and both I-1 and I-2 live in the divergence between the copies. Consolidating is the fix for all three.
- **ARCH-PURE — pass, with a forward note.** `parseREPLLine` is deterministic, memory-free, and table-tested with no IO. The note: the *terminal-state machine* — when to flash, which erase to emit, whether to `skipPrompt` — is now non-trivial policy living in the IO loop, assertable only by counting substrings in a buffer. That's where both of today's bugs are. Before #14 (line editor) and #15 (`/`-commands) add history rendering and type-ahead to the same loop, extract a pure `func replRender(ev event, interactive, tty bool) string`; the loop then just writes what it returns, and the screen contract becomes a table test instead of `strings.Count`.
- **ARCH-PURPOSE — pass; shadow-sweep clean.** Every Done-when item is delivered and independently verified. Consumers of the flag set — `run`, `defineOnce`, `repl`, `replay` — all derive from the single `options` construction at `main.go:93-100`; no hand-maintained restatement survives. The three documentation consumers (`-h` at `main.go:75-80`, README, atlas) were all updated, closing the prior I-4. The `interactive`/`tty` split at `repl.go:71` vs `options.tty` is the one place where a single question ("may I write escape sequences?") is answered from two sources.
- **ARCH-MOCK — pass with a note.** All new external surface is exercised behind existing seams: `cachingAudioSource` against the stateful `fakeCDN` with its ordered request recorder, the REPL end to end against `fakeDictionary`/`fakeCDN`/`fakePlayer`. The note: the Ctrl-C contract now depends on `exec.CommandContext` actually terminating a real `afplay` mid-file, and `player_conformance_test.go` covers only `TestAfplayBlocksUntilPlaybackCompletes`. That is the one real-binary behaviour this issue newly relies on and does not verify against the real binary — worth a conformance case (`ctx` cancelled after ~100ms, assert `Play` returns early). Non-blocking, since the operator exercised it once on a pty.

## 7. Plan revision recommendations

Append one `## Revisions` entry to `workshop/plans/000002-repl-plan.md` (timestamp + reason + delta, per AGENTS.md §1) covering all six — items 1–4 were recommended at the previous boundary and were not applied, so they are repeat findings at their original severity:

1. **Non-goals, line 28** — "No pager, no screen clearing, no cursor control. Output scrolls" is superseded. Record that the operator lifted the cursor-control non-goal on 2026-08-20 for the ephemeral indicator and the flash-and-restore replay, and that the new invariant is "the settled screen is unchanged", not "output scrolls".
2. **Chunk 1, `replCommand` bullet (line 56)** — drop `cmdQuit`; quitting is EOF or context cancellation, never a parsed line. The code is right, the plan is stale.
3. **Chunk 1, `repl` bullet (line 96)** — correct "Takes an `io.Reader`, the writers, and an `interactive bool`" to the shipped signature, which derives interactivity from `d.stdinIsTerminal` internally. The plan elsewhere states `run`'s change is "the only signature change in the issue", so this discrepancy needs correcting rather than leaving a parameter that doesn't exist.
4. **Task 4 Step 1 (line 154)** — drop the leftover "report it, and continue" clause, which Step 3 and the implementation deliberately contradict. PQ-10's round-3 disposition already committed to this and it was not done; note the test obligation is now discharged by `TestREPLOverlongLineIsReportedNotSilentEOF`.
5. **Cancellation contract section** — state what Ctrl-C *prints* (nothing, on both report sites), not only that it exits 0. This finally closes PQ-4's round-2 residue, which was parked on a manual check that never inspected stderr.
6. **New: the terminal-question taxonomy** — record that there are three distinct probes (`color`, `options.tty`, `deps.stdinIsTerminal`) and which one gates escape-sequence emission. This is the invariant I-1 violates; writing it down is what keeps #14/#15 from re-breaking it.

Separately, append a `## Revisions` entry to `workshop/issues/000002-repl.md` amending the Spec bullet that says a bare return "writes nothing to stdout at all" — the settled-screen contract is what survived the operator refinements, and the Spec should say that rather than something the code no longer does.
