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

---

## Re-review — 2026-08-20T15:40:33-07:00 (FIX-THEN-SHIP)

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
| timestamp | 2026-08-20T15:40:33-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The issue's purpose is delivered and the architecture is real, not asserted: `defineOnce` is genuinely the single define path, `playAnnounced` collapsed the duplication that round 2 flagged, `parseREPLLine` is pure and table-tested with no IO, and the cache decorator sits *inside* `repl` so `fakeCDN.Requested()` is a production-path assertion. I re-verified the environment rather than trusting the log: `go vet` clean, `go test -race -count=1 ./cmd/define/` passes (14.0s), `GOOS=linux CGO_ENABLED=0 go build ./...` green, and all four round-1 fixes plus round-2's I-4 (the Ctrl-C guard) are now genuinely covered — `TestCancellationPrintsNoDiagnostic` fails if the `ctx.Err() == nil` suppression is removed. What blocks a bare SHIP is that the operator's "announcement moves out of `speak`" refinement changed *when* the indicator is printed relative to the fetch, and nothing tested the consequence: I executed both paths and confirmed a word with no recording now writes a false `♫ playing 3×` to a redirected stdout while playing zero times, and `echo rizz | define` exits 0 where `define rizz` exits 1 — the exit-code contract README documents three lines below the newly-added piped example. Both are on the shipped one-shot/piped surface, both are cheap, neither is a crash. Alongside them, the plan's `## Revisions` section landed but skipped three of the four specific corrections the last two boundaries asked for, and the issue Spec still states something the code no longer does.

## 1. Strengths

- **`cmd/define/main.go:159-178`** — `playAnnounced` is a real consolidation, not a rename. The three-way divergence round 2 found (terminal gate, audio-off guard, duplicated literal) is genuinely gone, and the `indicator` struct makes the erase style the only remaining difference between the two callers. This is the ARCH-DRY win the issue set out to get.
- **`cmd/define/repl.go:80`** — the cache applied inside `repl` rather than in `realDeps()`. Production and test wiring are the same line, which is what makes `TestREPLBareReturnReplaysWithoutRefetching`'s "exactly 1 request" a real assertion instead of scaffolding.
- **`cmd/define/fetch.go:124-131`** — the cache derives permanence from the pre-existing `ErrNoAudio`/`ErrFetchFailed` taxonomy instead of re-deciding what "failed" means, and `TestCachingAudioSourceDoesNotCacheTransportFailures` pins the retryable half. Correct resolution of round-1 I-3.
- **`cmd/define/repl_test.go:207-227`** — the over-cap *and* just-under-cap pair. Pinning that the >64 KB guard is not over-eager, not merely that it fires, is what keeps a future `maxLineBytes` tweak honest.
- **`cmd/define/repl.go:76`** and **`repl_test.go:34-41`** — `transient := interactive && opt.tty`, with `replRigStreams` deliberately decoupling the two probes and a comment naming the old coupling as the thing that hid the bug. Round 2's I-1 is fixed at the level of the test *helper*, which is the durable fix.
- **`workshop/lessons.md:21-37`** — four rules, each traceable to a specific defect in this window (AGENTS.md §4 satisfied).

## 2. Critical findings

None.

## 3. Important findings

**I-1 — A word with no recording now announces playback that never happens; the line persists on a pipe. Behaviour drift on the shipped one-shot path.**
`cmd/define/main.go:132-137` and `cmd/define/main.go:159-165`

Before this window, `speak` printed `\n  ♫ playing %d×\n` *after* a successful `Fetch` and `os.WriteFile`, immediately before `playN` (base `main.go:110`). `playAnnounced` now prints it *before* calling `speak`, so every pre-playback failure — no recording, transport error, `MkdirTemp`/`WriteFile` failure — emits the announcement first. On a tty it is erased and invisible; on a pipe `ind.erase` is `""`, so the false line is kept.

Confirmed by execution (throwaway probe against `newAudioRig(t, "sycophantic", false)`, since the sandbox has no working CoreServices dictionary):

```
PLAYED=0  ANNOUNCED=true  stderr="define: sycophantic: no recorded pronunciation\n"
tail of stdout: "n(t)ək(ə)lē, -ˈfantik(ə)lē/ adverb\n\n  ♫ playing 3×\n"
```

So `define <word-with-no-recording> > out.txt` now records that the word played three times when it played zero. `TestRunMissingAudioStillSucceeds` (`main_test.go:191-206`) drives exactly this scenario and asserts `player.count() == 0`, but never asserts the absence of `playing` — even though `TestRunNoAudioMakesNoRequests:185` already uses precisely that assertion for the `-no-audio` case. The pattern was available and not applied.

Fix sketch — the indicator is *ephemeral UI* when erasable and a *record* when not, and a record has to be true. Announce optimistically only when it can be taken back:

```go
// in playAnnounced: cursor positioning stays unconditional (I-2's fix depends
// on it), but the text waits for something to actually play when it cannot be erased.
if ind.show { fmt.Fprint(stdout, ind.before) }
announceNow := ind.show && ind.erase != ""
if announceNow { fmt.Fprintf(stdout, "  ♫ playing %d×", opt.times) }
err := speak(...)
if ind.show && !announceNow && err == nil {
    fmt.Fprintf(stdout, "\n  ♫ playing %d×%s", opt.times, ind.trail)  // restores base behaviour
}
```

Keep `ind.before` unconditional or `TestREPLFailedReplayDoesNotWriteOntoThePrompt` regresses — with no step-back the diagnostic lands directly after the prompt, which is the defect that test exists to catch. Add the missing assertion to `TestRunMissingAudioStillSucceeds`, and state the resulting rule in `atlas/define.md`'s ephemeral-indicator paragraph.

**I-2 — `echo rizz | define` exits 0 where `define rizz` exits 1, contradicting the documented exit-code table for surface this issue introduced.**
`cmd/define/repl.go:130-135` and `cmd/define/repl.go:97-105`; `README.md:46`

`repl` discards `defineOnce`'s return value except to decide `current`, and returns 0 at EOF unconditionally. Verified against the built binary:

```
$ define -no-audio rizz      → define: rizz: no dictionary entry   exit=1
$ echo rizz | define -no-audio → define: rizz: no dictionary entry   exit=0
```

`README.md:31` now advertises `echo sycophantic | define   # or feed it words on stdin` directly beneath `define sycophantic`, presenting them as interchangeable, and `README.md:46` states `Exit codes: 0 success, 1 no dictionary entry, 2 usage error`. A script doing `echo "$w" | define || …` gets no signal. Exiting 0 at EOF is right for the *interactive* loop; it is wrong for the piped one-shot form the Done-when added.

Fix sketch — one bool, non-interactive only:

```go
var anyFailed bool
…
case cmdDefine:
    if defineOnce(...) == 0 { current = cmd.word } else { anyFailed = true }
…
// on the EOF arm:
if !interactive && anyFailed { return 1 }
return 0
```

Alternatively decide the divergence is intended and say so in README + Spec — but it should be a decision, not an omission. Either way it needs a test; there is none today for the exit code of a piped failed lookup.

**I-3 — `-no-color` still emits ANSI on a terminal, which README says it never does.**
`cmd/define/main.go:95-96`, `cmd/define/main.go:132-136`; `README.md:37`

`opt.tty` is `isTerminal(stdout)` with no reference to `noColor`, so `defineOnce` sets `ind.erase = eraseLine` (`"\r\x1b[K"`) regardless. `define -no-color <word>` on a tty therefore emits two escape sequences, while `README.md:37` documents the flag as "never emit ANSI (also automatic when piped)". Round 2 recorded this as Minor; I'm raising it because the doc line is unchanged and it was *this diff* that falsified it — that's drift from a stated contract, and `-no-color` exists precisely for terminals that mangle escapes.

Fix: either `tty: !*noColor && isTerminal(stdout)` (one line; the indicator then stays a plain line for those users, which is the correct degradation), or narrow README to "never emit ANSI **colour**". I'd take the former — it keeps the flag meaning one thing.

**I-4 — Plan and Spec still contradict the code in four named places; three were recommended at both prior boundaries and the `## Revisions` entry that landed does not cover them.**

The `## Revisions` section at `workshop/plans/000002-repl-plan.md:202` is well written and correctly handles the cursor-control non-goal, `playAnnounced`, `options.tty`, and both review rounds — that mechanism is now right. But it skipped the specific line-level corrections:

- `plan:56` — `replCommand` is still documented as including **`cmdQuit`**. The shipped type (`repl.go:19-23`) has only `cmdNothing`/`cmdDefine`/`cmdReplay`; quitting is EOF or cancellation, never a parsed line. Recommended at boundary 1 (item 1) and boundary 2 (item 2).
- `plan:96` — "Takes an `io.Reader`, the writers, and an **`interactive bool`**". The shipped signature is `repl(ctx, d, opt, stdin, stdout, stderr) int`, deriving interactivity from `d.stdinIsTerminal` at `repl.go:71`. The plan elsewhere asserts `run`'s change is "the only signature change in the issue", so this leaves a parameter documented that does not exist. Recommended twice.
- `plan:154` — still instructs "check `scanner.Err()` separately from the loop ending, **report it, and continue**", which Step 3 and the implementation deliberately contradict. This one is a **plan-gate carry-forward**: PQ-10's round-3 disposition in `000002-repl-plan-gate.md` reads *"drop Step 1's leftover 'report it, and continue' clause when writing the test"*. The test was written; the clause was not dropped. The ledger's `## Open findings` now says "(none — every finding has been disposed)" while recording a disposition action that is not true on disk.
- `workshop/issues/000002-repl.md:38` — the Spec bullet still says a bare return "writes **nothing to stdout at all** — no definition, no '♫ playing' line". Interactively it now writes `eraseLineAndStepBack`, the flash, `eraseLine`, and a redrawn prompt. The *settled screen* is unchanged, which is the real (and good) contract; the Spec as written is false, and the issue has no `## Revisions` section. Recommended at boundary 2.

Not blocking, but these are the artifacts the next issue (#14) reads to decide what to keep — and #14's own file already commits to deleting `eraseLineAndStepBack`, so a plan that still calls cursor control a non-goal in one place and describes a `cmdQuit` that never existed is the wrong handoff.

## 4. Minor findings

- `atlas/define.md:200` — "Failed fetches are not cached, so a transient outage does not poison a session" directly contradicts the paragraph's own heading four lines above ("**'No recording' is cached; a transport failure is not**") and duplicates the outage clause. Stale leftover from before I-3's fix; flagged at boundary 2 and not removed.
- `atlas/define.md:190` — "both report sites suppress a diagnostic" — after the `playAnnounced` consolidation there is exactly one.
- `atlas/define.md:186` — "Gated on interactive" — it is gated on `interactive && opt.tty`, which is the whole point of round 2's I-1. Also `atlas/define.md:198-199` is a run-on/mis-wrapped line ("…re-deciding what \"failed\" means. Replay costs no network: `cachingAudioSource`…").
- `cmd/define/repl.go:112` — "nothing to replay: audio is off" also fires for `-times 0`, where audio is not off.
- `-raw` inside the loop: `defineOnce` returns at `main.go:122-125` before the audio block, so typing a word plays nothing — yet a bare return then *does* play it, and pays a fetch the cache never warmed. Undeclared in the Spec's "flags are session settings".
- `cmd/define/repl_test.go:158` — the timeout arm selects on `t.Context().Done()`, which cannot fire while the test is blocked in that select. A regression hangs to the package timeout rather than failing "promptly"; use `time.After(2*time.Second)`.
- `cmd/define/main_test.go:12` — "a nil one would make run() panic rather than fail a test" is now false: `repl.go:71` tolerates a nil `stdinIsTerminal` and silently treats it as non-interactive. `testDeps` relies on that.
- `cmd/define/main.go:203` — `isTerminal(w io.Writer)` is now called on `os.Stdin`; it works, but the parameter name no longer describes the use.
- `cmd/define/repl.go:93` / `repl.go:102` — duplicated `if interactive { fmt.Fprintln(stdout) }` across the two exit arms.
- `cmd/define/main.go:144-149` — `indicator.trail` is documented as "written instead of erase when there is nothing to erase", but it is written *before* playback, alongside the text. Rename or reword.
- `cmd/define/player_fake_test.go:18` — `fakePlayer.Play` discards `ctx`, so cancel-*during-playback* is still unreachable from the suite (the guard is now covered via a failing fetch instead, which is adequate but not the real path).
- A second Ctrl-C is swallowed: `stop()` is deferred to `main` exit, so `NotifyContext` keeps absorbing SIGINT and there is no force-quit if playback ever hangs.
- `cachingAudioSource.hits` is unbounded for the session lifetime. Fine at a few KB per word; noting so it isn't rediscovered.
- `README.md:40-44` wraps mid-sentence at an odd column; reflow.
- The window bundles unrelated tracker work (commit `cbcd30c`: `#14`/`#15` issue files, project rows) with `#2`'s implementation — harmless, but it widens the reviewed boundary.

## 5. Test coverage notes

The suite asserts through fakes rather than restating the implementation — player counts, `cdn.Requested()` lengths, byte-identical stdout across two whole runs — which is the right posture, and I confirmed the Ctrl-C guard is no longer dead (removing `&& ctx.Err() == nil` breaks `TestCancellationPrintsNoDiagnostic`). Gaps, in priority order:

1. **The announcement's position relative to the fetch** (I-1). `TestRunMissingAudioStillSucceeds` drives the exact scenario and stops one assertion short. Add `if strings.Contains(out.String(), "playing")` there — mirroring `main_test.go:185`.
2. **Exit code of a failed lookup on the piped path** (I-2). No test asserts `repl`'s return value for anything other than success, EOF, and the over-cap read error.
3. **The non-tty define path's `trail` newline.** `TestRunPlaysThreeTimesByDefault:157` only does `Contains(out, "playing 3")`, so a regression setting `trail = ""` would glue the indicator to whatever follows and stay green.
4. `-raw` inside the REPL — untested; the audio asymmetry above went unnoticed.
5. `opt.times == 0` routed into the replay branch — untested.
6. `TestREPLReplayFlashesThenRestoresThePrompt:246` asserts `Count(s, eraseLine) >= 2`, but `eraseLineAndStepBack` itself contains `eraseLine` twice, so the threshold is met by the step-back alone. Weak, not wrong.

`-race` is clean and the goroutine sharing model holds on inspection: `d` is a value copy so the `d.audio` rewrite at `repl.go:80` is unshared, and both channels carry values.

## 6. Architectural notes

- **ARCH-DRY — pass.** `defineOnce` is the single define path and `playAnnounced` is now the single owner of announce → play → erase → report; I checked the two call sites and the only remaining difference is the `indicator` value, which is the intent. `cachingAudioSource` is a decorator on the existing seam and derives its permanence rule from the existing error taxonomy rather than restating it. One trivial duplication remains (`repl.go:93`/`102`).
- **ARCH-PURE — pass, with a forward note.** `parseREPLLine` is deterministic, memory-free (`hasCurrent` passed in), and its table test touches no IO, exec, or fs — genuinely PURE, matching the plan's table. The note carried forward from boundary 2 and now *sharper*: the terminal state machine — when to flash, which erase, whether to `skipPrompt`, and (after I-1) whether the announcement is a record or ephemeral UI — is non-trivial policy living in the IO loop, assertable only by `strings.Count` over a buffer. Both of this boundary's behaviour findings live there. `#14`'s file already commits to deleting `eraseLineAndStepBack` and the `skipPrompt` bookkeeping under raw mode, which is the right instinct; pair it with a pure `func replFrame(ev event, interactive, tty bool) string` so the screen contract becomes a table test.
- **ARCH-PURPOSE — pass with a flag.** Every Done-when item is delivered and independently verified. Shadow-sweep over the consumers of "what invocations and behaviours exist": `-h` (`main.go:75-80`) ✓ updated, `README.md` ✓ updated, `atlas/define.md` ✓ updated but carrying three stale sentences, `workshop/plans/…-plan.md` ✗ three stale claims, `workshop/issues/000002-repl.md` Spec ✗ one false claim (I-4). Consumers of the flag set — `run`, `defineOnce`, `repl`, `playAnnounced` — all derive from the single `options` construction at `main.go:93-100`; no hand-maintained restatement survives there. The exit-code contract (I-2) is the one behavioural consumer that did not derive.
- **ARCH-MOCK — pass with a note.** All new external surface runs behind existing seams: `cachingAudioSource` against the stateful `fakeCDN` with its ordered request recorder, the REPL end-to-end against `fakeDictionary`/`fakeCDN`/`fakePlayer`, and production/test share the same wiring line. The note, repeated from boundary 2 and still open: the Ctrl-C contract now depends on `exec.CommandContext` actually terminating a real `afplay` mid-file, and `player_conformance_test.go` covers only `TestAfplayBlocksUntilPlaybackCompletes`. That is the one real-binary behaviour this issue newly relies on and does not verify against the real binary. A `ctx` cancelled after ~100 ms asserting `Play` returns early is a five-line addition to the existing conformance file. Non-blocking — the operator exercised it on a pty — but it is the drift-detection this principle asks for.

## 7. Plan revision recommendations

Append to the **existing** `## Revisions` section of `workshop/plans/000002-repl-plan.md` (a new dated sub-entry; don't rewrite the one that's there — it's good):

1. **`plan:56`** — drop `cmdQuit` from the `replCommand` enumeration; quitting is EOF or context cancellation, never a parsed line. *Third time recommended.*
2. **`plan:96`** — correct "Takes an `io.Reader`, the writers, and an `interactive bool`" to the shipped signature, which derives interactivity from `d.stdinIsTerminal` internally. *Third time recommended*; it currently contradicts the plan's own "the only signature change in the issue".
3. **`plan:154`** — drop the leftover "report it, and continue" clause. This is the plan-gate carry-forward: PQ-10's round-3 disposition committed to exactly this edit, so until it lands the gate ledger records a disposition that is false on disk. Note the obligation is now discharged by `TestREPLOverlongLineIsReportedNotSilentEOF`.
4. **Cancellation contract section (`plan:36-43`)** — state what Ctrl-C *prints* (nothing, from the single suppression in `playAnnounced`), not only that it exits 0. The Revisions entry records the fix narratively; the contract section still reads as if only the exit code matters. This is PQ-4's round-2 residue.
5. **New, from I-1** — record when the announcement is written relative to the fetch, and why it differs by erasability. The move out of `speak` is documented as a refinement; its interaction with a failed fetch is not, and that gap is the finding.
6. **New, from I-2** — state the exit-code contract for the loop explicitly (interactive → always 0; piped → propagate a failed lookup, or deliberately not, with the reason).

Separately, append a `## Revisions` entry to **`workshop/issues/000002-repl.md`** amending the Spec bullet at line 38: the surviving contract is "the *settled screen* is unchanged and no definition is reprinted", not "writes nothing to stdout at all". *Second time recommended.* The issue currently has no `## Revisions` section at all, so AGENTS.md §1's append-don't-overwrite rule has not been applied to it.

---

## Re-review — 2026-08-20T15:53:46-07:00 (FIX-THEN-SHIP)

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
| timestamp | 2026-08-20T15:53:46-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The issue's purpose is delivered and the architecture is real rather than asserted, and I verified the environment rather than trusting the log: `go vet` clean, `go test -race -count=1 ./cmd/define/` passes (15.5s), `GOOS=linux CGO_ENABLED=0 go build ./...` green. More usefully, I mutation-tested the fixes from the three prior rounds and they are genuinely pinned — dropping `&& transient` from the replay redraw fails `TestREPLReplayWritesNothingToStdout`; reverting `transient` to `interactive` alone fails `TestREPLMismatchedStreamsEmitNoEscapes`; moving the announcement back ahead of `speak` fails `TestRunMissingAudioStillSucceeds`; removing the `ctx.Err() == nil` suppression fails `TestCancellationPrintsNoDiagnostic`. The plan-gate ledger's last live carry-forward (PQ-10's "drop the *report it, and continue* clause") is now actually true on disk. Nothing here is Critical and nothing blocks the gate. What keeps it off a bare SHIP is one shipped-behaviour limitation that no artifact records — terminal echo during playback moves the cursor out from under the erase, so an impatient second Enter strands the indicator and skews the step-back arithmetic, while `atlas/define.md` states the "view never scrolls" invariant unconditionally — plus three atlas sentences that are now factually false about the code they describe, one output-format property that survives mutation untested, and a confirmed `-raw` asymmetry between the two entry modes (probed: `define -raw w` plays 0×, the same flag in the loop plays 3× on a bare return and issues a CDN request).

## 1. Strengths

- **`cmd/define/main.go:159-178`** — `playAnnounced` as the single owner of announce → play → erase → report. I read both call sites: the only difference is now the `indicator` value, which is exactly what round 2's I-3 asked for. The `erasable` split ("ephemeral UI may be optimistic; a record has to be true") is a genuinely good distinction, and mutation-testing confirms it's enforced, not just commented.
- **`cmd/define/repl.go:80`** — the cache applied inside `repl` rather than in `realDeps()`. Production and test wiring are the same line, which is *why* `fakeCDN.Requested()` is a real assertion; M3/M4 mutations prove the tests reach the production path.
- **`cmd/define/fetch.go:124-131`** — permanence derived from the pre-existing `ErrNoAudio`/`ErrFetchFailed` taxonomy instead of re-deciding what "failed" means, with both halves pinned (`TestCachingAudioSourceCachesErrNoAudio` / `…DoesNotCacheTransportFailures`).
- **`cmd/define/repl.go:30-41`** — `parseREPLLine` with `hasCurrent` passed in. Genuinely PURE; its table test imports only `bytes`/`context`/`strings`/`testing`.
- **`cmd/define/repl_test.go:34-41`** — `replRigStreams` decoupling the two terminal probes, with a comment naming the old coupling as what hid the bug. Fixing a defect at the level of the *test helper* is the durable form, and `workshop/lessons.md:21-37` generalises it into four rules that each trace to a specific defect in this window (AGENTS.md §4 satisfied).

## 2. Critical findings

None.

## 3. Important findings

**I-1 — Terminal echo during playback breaks the erase arithmetic; the limitation is recorded nowhere while the atlas asserts the invariant unconditionally.**
`cmd/define/main.go:170`, `cmd/define/repl.go:126`; `atlas/define.md:180-187`

`eraseLine` (`\r\x1b[K`) acts on whatever line the cursor is *currently* on. The loop is blocked inside `speak` for ~3 seconds, and the tty is in cooked mode with ECHO on, so anything typed during that window is echoed by the driver immediately and asynchronously to our writes. Enter is the only key this REPL responds to, so a second impatient Enter during playback is ordinary use, not a corner case: the echoed newline moves the cursor down, the post-playback `ind.erase` then clears the echoed line instead of the indicator, and `♫ playing 3×` is stranded on screen permanently. The queued newline is then read as another replay whose `eraseLineAndStepBack` steps back over lines that no longer hold what it assumes.

This is by construction rather than by observation — the sandbox has no CDN access and the test harness has no echo, so I could not reproduce it here; on a pty it is `define`, a word, then two quick Returns during playback. `atlas/define.md:184-186` says "the view never scrolls" and "the settled screen shows only the definition" with no qualification, and `README.md:40` says "the screen does not change".

The real fix is raw mode, which is `#14`'s job and which `workshop/issues/000014-repl-editor.md:69-77` already commits to (delete the workaround, don't port it). The fix *at this boundary* is to stop claiming an invariant the code only holds when the user waits: add a sentence to `atlas/define.md`'s ephemeral-indicator paragraph (and one line to `#14`'s inherited-workaround section) recording that the arithmetic assumes no input arrives during playback, and that cooked-mode echo is what makes that assumption breakable.

**I-2 — Three sentences in `atlas/define.md` are now false about the code they describe.**
`atlas/define.md:186`, `:190`, `:200`

The atlas is what the next agent reads to decide what to keep; each of these was flagged at a prior boundary and none was fixed.

- `:200` — "Failed fetches are not cached, so a transient outage does not poison a session." This directly contradicts its own paragraph heading four lines above ("**'No recording' is cached; a transport failure is not**") — `ErrNoAudio` *is* cached (`fetch.go:124-128`) — and duplicates the outage clause already stated at `:196`. Delete the sentence.
- `:190` — "both report sites suppress a diagnostic when `ctx.Err() != nil`". After the `playAnnounced` consolidation there is exactly one (`main.go:179-184`). The sentence describes the pre-round-2 shape.
- `:186` — "Gated on interactive" — it is gated on `interactive && opt.tty` (`repl.go:76`), and gating on `interactive` alone *was* round 2's I-1. Stating the weaker gate in the atlas is how that bug comes back.

Same false claim also survives in the plan at `workshop/plans/000002-repl-plan.md:74` (see §7). `:198-199` is additionally a mis-wrapped run-on line; reflow while you're there.

**I-3 — The record-form indicator's output format has no test; a mutation dropping its trailing newline is green.**
`cmd/define/main.go:137`, `cmd/define/main.go:175`; `cmd/define/main_test.go:157`

I changed `ind.erase, ind.trail = "", "\n"` to `"", ""` and the whole suite still passed. That regression makes `define word | cat` emit `…adverb\n\n  ♫ playing 3×` with no terminating newline, gluing the record to whatever follows it. Every existing assertion on this line is a `Contains(out, "playing 3")` or a `Count(s, "♫")` (grepped: `main_test.go:157,185,209`, `repl_test.go:70,249,262,293,331`) — none pins the shape. This is precisely the class of bug the boundary already shipped once at round 3 (the false announcement on a pipe), in the same four lines of code.

Fix: assert the exact record form on the non-tty path, e.g. in `TestRunPlaysThreeTimesByDefault`:
```go
if !strings.HasSuffix(out.String(), "\n  ♫ playing 3×\n") {
    t.Errorf("record form changed: %q", out.String())
}
```

**I-4 — `-raw` plays audio in the loop but never one-shot; confirmed by execution.**
`cmd/define/main.go:122-125` and `cmd/define/repl.go:110-113`

`defineOnce` returns before the audio block under `-raw`, so typing a word in a `-raw` session prints the unparsed entry and plays nothing — but the replay branch never consults `opt.raw`, so a bare return fetches and plays. Probed against the fake rig:

```
-raw in the REPL   : plays=3  cdnRequests=1
-raw one-shot      : plays=0  cdnRequests=0
```

So the flag means two different things depending on which line you're on, in an issue whose stated architecture is that both modes run the same define path, and `README.md:39-40` tells the user a bare return means "nothing is re-fetched" while under `-raw` it is the *first* fetch. Raised as Minor at boundaries 2 and 3; carrying it a fourth time without a decision is what makes it a finding rather than a note. Cheapest fix is to make the flag mean one thing:

```go
if opt.raw || opt.noAudio || opt.times <= 0 {
    fmt.Fprintln(stderr, "define: nothing to replay: audio is off")
    break
}
```
— or decide `-raw` should not suppress audio at all and change `defineOnce`. Either way it needs one sentence in the Spec's "flags are session settings" and a test.

## 4. Minor findings

- `cmd/define/repl.go:112` — "nothing to replay: audio is off" also fires for `-times 0`, where audio is not off (probed: exact message confirmed). Fourth boundary.
- `cmd/define/repl.go:96-108` — `select` gives `ctx.Done()` no priority over `lines`, so after Ctrl-C with type-ahead queued the loop may define one more word before quitting. A non-blocking `if ctx.Err() != nil { … }` at the top of the iteration removes the coin flip.
- `cmd/define/main.go:93-99` — `color` and `tty` are now the *identical* expression `!*noColor && isTerminal(stdout)`. The comment insists they are different questions; the code no longer distinguishes them. Either derive one from the other with a named helper or note that they coincide today by choice.
- `cmd/define/main.go:167,176` — the `"  ♫ playing %d×"` literal appears twice inside `playAnnounced` itself; a local `line := fmt.Sprintf(...)` removes the last copy.
- `cmd/define/main.go:144` — `indicator.trail` is documented as "written instead of erase when there is nothing to erase", but it is written *after* playback alongside the text, not instead of anything. Reword.
- `cmd/define/repl.go:93,102` — duplicated `if interactive { fmt.Fprintln(stdout) }` across the two exit arms.
- `cmd/define/repl.go:71` — `d.stdinIsTerminal != nil` silently defaults to non-interactive, which falsifies `main_test.go:12`'s stated norm ("a nil one would make run() panic rather than fail a test"). `testDeps` now relies on the tolerance.
- `cmd/define/repl_test.go:158` — the timeout arm selects on `t.Context().Done()`, which cannot fire while the test is blocked in that same select; a regression hangs to the package timeout instead of failing "promptly". Use `time.After(2*time.Second)`. Third boundary.
- `cmd/define/repl_test.go:249` — `Count(s, eraseLine) >= 2` is satisfied by `eraseLineAndStepBack` alone (it contains `eraseLine` twice). Weak threshold, not wrong.
- `README.md:46` — the exit-code table says `1` = no dictionary entry, but the interactive loop deliberately exits 0 on a typo (`repl.go:100-104`). The issue's Revisions records the divergence; README, which sits three lines below the piped example, does not.
- `cmd/define/main.go:203` — `isTerminal(w io.Writer)` is now called on `os.Stdin`; it works, but the parameter name no longer describes the use.
- `cmd/define/player_fake_test.go:18` — `fakePlayer.Play` discards `ctx`, so cancel-*during-playback* is still unreachable from the suite (the guard is covered via a failing fetch instead — adequate, not the real path).
- A second Ctrl-C is swallowed: `stop()` is deferred to `main` exit, so there is no force-quit if playback ever hangs.
- `cachingAudioSource.hits` is unbounded for the session; a few KB per word makes this fine, noting so it isn't rediscovered.
- `define hot dog` is a usage error while typing `hot dog` in the loop works (`repl.go:39`). Quoting is ordinary shell convention, but the two entry modes disagree about a case the plan calls out as real.
- The window bundles unrelated tracker work (`cbcd30c`: `#14`/`#15` issue files, project rows) with `#2`'s implementation.

## 5. Test coverage notes

The suite asserts through fakes rather than restating the implementation, and I confirmed by mutation that four of the previously-shipped bugs are now genuinely pinned (details in the summary). Remaining gaps, in priority order:

1. **The record-form indicator's exact bytes** (I-3) — a dropped trailing newline is green today.
2. **`-raw` inside the REPL** (I-4) — untested, which is why the asymmetry has survived three boundaries.
3. **`opt.times == 0` routed into the replay branch** — untested; the misleading message rides on it.
4. **Define-path failure with `opt.tty == true`** — no test covers the erasable indicator being taken back on a *define* (only on replay, via `TestREPLFailedReplayDoesNotWriteOntoThePrompt`).
5. **Ctrl-C priority with queued input** — no test; the non-deterministic select above is invisible.

`-race` is clean and the sharing model holds on inspection: `deps` is passed by value so the `d.audio` rewrite at `repl.go:80` is unshared, and both channels carry values.

## 6. Architectural notes

- **ARCH-DRY — pass.** I checked both `playAnnounced` call sites: `defineOnce` and the replay branch differ only in the `indicator` value, so the three-way divergence round 2 found is genuinely gone, and `cachingAudioSource` derives permanence from the existing error taxonomy rather than restating it. The residue is cosmetic (duplicated literal inside the helper, duplicated interactive-newline, `color`/`tty` identical expressions) — all Minor above.
- **ARCH-PURE — pass, with the same forward note now sharper.** `parseREPLLine` is deterministic and its table test touches no IO, exec, or fs. But the terminal state machine — when to flash, which erase, whether to `skipPrompt`, whether the announcement is a record or ephemeral — is non-trivial policy living in the IO shell, assertable only by `strings.Count` over a buffer, and this round's I-1 and I-3 both live there. Before `#14` adds a line editor and `#15` adds `/`-command type-ahead to the same loop, extract a pure `func replFrame(ev event, interactive, tty bool) string` so the screen contract becomes a table test. `#14`'s plan to delete the cooked-mode arithmetic under raw mode is the right instinct; pair it with the pure renderer or the same class of bug returns in raw mode.
- **ARCH-PURPOSE — pass; shadow-sweep flags the doc consumers.** Every Done-when item is delivered and verified by something other than the implementor's word. Consumers of the flag set (`run`, `defineOnce`, `repl`, `playAnnounced`) all derive from the single `options` construction at `main.go:90-100` — no hand-maintained restatement survives. Consumers of "what invocations and behaviours exist": `-h` (`main.go:75-80`) ✓, `README.md` ✓ (one exit-code nuance short), `workshop/issues/000002-repl.md` Spec ✓ (properly amended with a `## Revisions` section), `atlas/define.md` ✗ three false sentences (I-2), `workshop/plans/000002-repl-plan.md` ✗ one false sentence (§7). The behavioural consumer that still doesn't derive is `-raw` (I-4).
- **ARCH-MOCK — pass with one standing note.** All new external surface runs behind existing seams, and the fact that M3/M4 mutations *fail* proves the fake sits on the production path rather than beside it. Still open from rounds 2 and 3: the Ctrl-C contract now depends on `exec.CommandContext` actually terminating a real `afplay` mid-file, and `cmd/define/player_conformance_test.go` contains only `TestAfplayBlocksUntilPlaybackCompletes`. That is the one real-binary behaviour this issue newly relies on and does not verify against the real binary. A `ctx` cancelled after ~100 ms asserting `Play` returns early is a five-line addition to the file that already exists. Non-blocking — the operator exercised it on a pty — but it is exactly the drift detection this principle asks for, and it has now been deferred three times.

**Plan-gate carry-forward:** `workshop/plans/000002-repl-plan-gate.md` lists no open findings, and I verified PQ-10's disposition action is now genuinely performed — Task 4 Step 1 (`plan:154-160`) no longer says "report it, and continue" and explains why it cannot. The ledger is honest as of this boundary.

**Core-concepts cross-check:** all six rows verified at their stated paths with their stated status (`replCommand`, `parseREPLLine` in `repl.go`; `cachingAudioSource` in `fetch.go`; `defineOnce`, `stdinIsTerminal` in `main.go`; `repl` in `repl.go`). PURE rows test without IO; the INTEGRATION row is injected at `repl.go:80` rather than called from business logic. No contradictions.

## 7. Plan revision recommendations

Append a new dated sub-entry to the existing `## Revisions` section of `workshop/plans/000002-repl-plan.md` (the entry that's there is good — don't rewrite it):

1. **`plan:74`** — "**Failed fetches are not cached**, so a transient outage does not poison the rest of the session" is now false: `ErrNoAudio` is cached as permanent, `ErrFetchFailed` is not. Same sentence as `atlas/define.md:200` (I-2); correct both.
2. **Cancellation contract (`plan:36-43`)** — state what Ctrl-C *prints* (nothing, from the single suppression in `playAnnounced`), not only that it exits 0. *Fourth boundary this has been recommended*; the Revisions entry covers it narratively while the contract section still reads as if only the exit code matters. This is PQ-4's original round-2 residue.
3. **Chunk 1 integration table** — `indicator` and `scanLines` are shipped entities absent from the table (`playAnnounced` and `options` at least appear in Revisions). Add the two rows so the table remains the greppable inventory it's meant to be.
4. **New, from I-1** — record that the cursor arithmetic assumes no input arrives during playback, and that cooked-mode echo is what breaks the assumption. This is the invariant the atlas currently states unconditionally, and it is the piece `#14` needs in order to know *why* raw mode is the fix rather than a preference.
5. **New, from I-4** — state what `-raw` means inside the loop, since one-shot and replay currently disagree.

No revision needed to `workshop/issues/000002-repl.md` — its Spec was properly amended and its `## Revisions` section now matches the code, which resolves the finding carried from the two previous boundaries.

---

## Re-review — 2026-08-20T16:05:50-07:00 (FIX-THEN-SHIP)

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
| timestamp | 2026-08-20T16:05:50-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The issue's purpose is delivered and the architecture is real rather than asserted. I verified the environment myself rather than trusting the Log: `go vet` clean, `go test -race -count=1 ./cmd/define/` passes (15.5s), `GOOS=linux CGO_ENABLED=0 go build ./...` green, tree clean. I mutation-tested both round-4 fixes and both are genuinely pinned — reverting `noAudio: *noAudio || *raw` fails `TestRawNeverPlays`, and dropping the record's trailing newline fails `TestRunPlaysThreeTimesByDefault`. The plan-gate ledger's last carry-forward (PQ-10) is now true on disk. What keeps it off a bare SHIP is one behaviour gap I reproduced by execution: the prompt itself is still gated on the **stdin** probe while being written to **stdout**, so `define > out.txt` from a terminal writes three `› ` tokens plus a stray newline into the file while the human at the keyboard sees no prompt at all — the same stream-mismatch class round 2's I-1 fixed for escape sequences and left behind for the prompt, on the same configuration a test already covers. Alongside that, commit `c5607ab`'s message claims it deleted three false atlas sentences; it deleted two, and the third is still on disk contradicting its own paragraph heading four lines above. Neither is a crash, both are cheap.

## 1. Strengths

- **`cmd/define/main.go:159-197`** — `playAnnounced` as the single owner of announce → play → erase → report, with the `erasable` split ("ephemeral UI may be optimistic; a record has to be true") as the only policy. I read both call sites: the `indicator` value is the sole difference. That distinction is now enforced, not just commented.
- **`cmd/define/main.go:105`** — deciding `-raw ⇒ noAudio` once at flag parse rather than at each branch. The flag now means one thing regardless of which line you're on, and `TestRawNeverPlays` drives both entry modes through the same rig. Clean resolution of round 4's I-4.
- **`cmd/define/repl.go:80`** — the cache decorator applied inside `repl` rather than in `realDeps()`, so production and test wiring are the same line. This is what makes `fakeCDN.Requested()` a real assertion instead of scaffolding.
- **`cmd/define/repl.go:30-41`** — `parseREPLLine` with `hasCurrent` passed in. Genuinely PURE; the table test imports only `bytes`/`context`/`strings`/`testing`.
- **`cmd/define/repl_test.go:34-41`** — `replRigStreams` decoupling the two terminal probes, with the comment naming the old coupling as the blind spot. Fixing a defect at the level of the test *helper* is the durable form, and `workshop/lessons.md:21-37` generalises it into four rules each traceable to a specific defect in this window (AGENTS.md §4 satisfied).
- **`workshop/issues/000014-repl-editor.md:69-86`** — the cooked-mode workaround recorded as inherited debt with an explicit instruction to *delete* rather than port it. That is the right handoff.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — The prompt is gated on stdin but written to stdout, so `define > out.txt` pollutes the file and shows the human nothing.**
`cmd/define/repl.go:92`, `cmd/define/repl.go:98`, `cmd/define/repl.go:107`

`interactive` comes from `d.stdinIsTerminal` (repl.go:71) and gates three stdout writes: the prompt, and the newline on each of the two exit arms. Round 2's I-1 fixed exactly this mistake for the *escape sequences* by introducing `transient := interactive && opt.tty`; the prompt was left on the old gate. Reproduced with a throwaway probe (stdin TTY, `opt.tty` false — `define > out.txt`), stdout was:

```
"› sycophantic  syc·o·phan·tic\n…\n\n  ♫ playing 3×\n› › \n"      promptCount=3
```

So the redirected file carries three `› ` tokens and a trailing newline, and the person typing gets no prompt on their screen. `README.md:43` states "The prompt appears only on a terminal, so piping stays clean" — true for `echo w | define`, false for this pairing. `TestREPLMismatchedStreamsEmitNoEscapes` (`repl_test.go:305-313`) drives this exact configuration and asserts only the absence of `\x1b[`; `TestREPLPromptOnlyWhenInteractive` (`repl_test.go:176-185`) uses the coupling helper, so it can't see it either.

The fix needs care: gating on `opt.tty` would suppress the prompt under `-no-color`, since round 3's I-3 folded `!noColor` into that field. "Will a human see this?" is a **fourth** question, independent of the colour flag. Fix sketch — add the probe to `deps` alongside its sibling:

```go
// deps
stdoutIsTerminal func() bool   // realDeps: func() bool { return isTerminal(os.Stdout) }

// repl
visible := interactive && d.stdoutIsTerminal()   // prompt + the exit newline
transient := visible && opt.tty                  // escapes: also honours -no-color
```

Then extend `TestREPLMismatchedStreamsEmitNoEscapes` with `strings.Contains(out.String(), prompt) == false`, which is the assertion that was one line away.

**I-2 — `atlas/define.md:210` still carries the false sentence commit `c5607ab` says it deleted; the same sentence survives at `plan:74`.**
`atlas/define.md:210`, `workshop/plans/000002-repl-plan.md:74`

> "Failed fetches are not cached, so a transient outage does not poison a session."

`ErrNoAudio` *is* cached (`fetch.go:124-128`), which is what the section heading fourteen lines above says: "**'No recording' is cached; a transport failure is not**". The commit message for `c5607ab` reads "three atlas sentences were false: 'failed fetches are not cached' contradicted its own heading, 'both report sites' …, 'gated on interactive' …". The second and third are genuinely fixed; the first was not — the paragraph was restructured around it and the sentence rode along to the end of the new one. Flagged at boundaries 2, 3 and 4. Delete both occurrences; the clause is already correctly stated at `atlas/define.md:196-199`.

This one matters beyond tidiness: the atlas is what `#14` reads, and the sentence asserts the exact opposite of the cache rule this issue introduced.

**I-3 — The plan has no `## Revisions` entry for round 4, and the Cancellation contract still omits what Ctrl-C prints.**
`workshop/plans/000002-repl-plan.md:36-43`, `:202-…`

AGENTS.md §1 requires an appended revision entry when a plan artifact moves mid-stream. The existing entry (`:204`) is good and covers rounds 1–3 and the operator refinements, but the code moved again in `c5607ab` (`-raw` semantics, the record-form pin, the cooked-mode limitation) and the section stops at round 3.

Within that, `## Cancellation contract` (`:36-43`) still describes only the exit code — "deferred cleanup runs, and the process exits 0" — with no statement that Ctrl-C is *silent*. That is PQ-4's original round-2 residue and it has now been recommended at all four prior boundaries. The behaviour is implemented and tested (`main.go:186-189`, `TestCancellationPrintsNoDiagnostic`); only the contract statement is missing. One sentence closes a finding that has been carried five times.

## 4. Minor findings

- `atlas/define.md:203-210` — the new erase-arithmetic paragraph and the caching paragraph are welded into one run-on ("…which is `#14`'s job. Replay costs no network: `cachingAudioSource` decorates…"). Split them while removing I-2's sentence.
- Dropping `fmt.Fprint(stdout, ind.before)` from the record path (`main.go:193`) is a **green mutation** — verified. The `HasSuffix("\n  ♫ playing 3×\n")` assertion added at round 4 pins the tail but not the blank separator line, so `define word | cat` losing that line is undetected. One character away from the bug round 4 fixed.
- `-no-color` on a terminal leaves a replay with **zero** acknowledgement: probed at playCount 6, one `♫` on screen (from the define path only). The indicator's stated purpose is "show the program responded to a keypress"; under `-no-color` that acknowledgement is gone rather than degraded to a plain line. Knock-on of round 3's I-3; arguably correct under "the settled screen is unchanged", but undeclared.
- On a pipe the record says `playing 3×` while 6 plays occurred (define records, replay is silent). Under-reporting rather than false, but "a record has to be true" is now an explicit principle in both the code comment and `lessons.md`.
- `indicator.before` (`main.go:154`) carries two meanings — escape-sequence cursor positioning on the replay path, a layout newline on the define path — and which is safe depends on `erase != ""`. The struct permits the unsafe combination (`show: true, erase: "", before: eraseLineAndStepBack`) that would write escapes to a pipe.
- `indicator.trail` (`main.go:156`) is documented "written instead of erase when there is nothing to erase", but it is written *after* the text on the success path, not instead of anything.
- `cmd/define/repl.go:120` — "nothing to replay: audio is off" now also fires for `-times 0` and for `-raw`, where audio is not "off" in the user's sense. Fourth boundary.
- `cmd/define/repl.go:96-114` — `select` gives `ctx.Done()` no priority over `lines`, so after Ctrl-C with type-ahead queued the loop may define one more word. A non-blocking `if ctx.Err() != nil` at the top of the iteration removes the coin flip.
- `cmd/define/main.go:95` and `:99` — `color` and `tty` are now the identical expression while the comment insists they are different questions. With I-1 the taxonomy is four questions, not three; worth naming them once.
- `cmd/define/main.go:176,194` — the `"  ♫ playing %d×"` literal appears twice inside `playAnnounced` itself.
- `cmd/define/repl.go:99,108` — duplicated `if interactive { fmt.Fprintln(stdout) }` across the two exit arms.
- `cmd/define/repl_test.go:158` — the timeout arm selects on `t.Context().Done()`, which cannot fire while the test is blocked in that same select; a regression hangs to the package timeout rather than failing "promptly". Third boundary.
- `cmd/define/main_test.go:12` — "a nil one would make run() panic rather than fail a test" is false now that `repl.go:71` tolerates a nil `stdinIsTerminal`; `testDeps` relies on the tolerance.
- `cmd/define/main.go:222` — `isTerminal(w io.Writer)` is called on `os.Stdin`; it works, but the parameter name no longer describes the use.
- `README.md:46` — the exit-code table doesn't note that the interactive loop deliberately exits 0 on a typo (`repl.go:110-113`). The issue's Revisions records the divergence; the README, three lines below the piped example, does not.
- `cmd/define/player_fake_test.go:18` — `fakePlayer.Play` discards `ctx`, so cancel-*during-playback* is still unreachable from the suite (the guard is covered via a failing fetch instead — adequate, not the real path).
- A second Ctrl-C is swallowed: `stop()` is deferred to `main` exit, so there is no force-quit if playback hangs.
- `cachingAudioSource.hits` is unbounded for the session; fine at a few KB per word, noting so it isn't rediscovered.
- `define hot dog` is a usage error while typing `hot dog` in the loop works (`repl.go:40`) — the two entry modes disagree about a case the plan calls out as real.
- The window bundles unrelated tracker work (`cbcd30c`: `#14`/`#15` issue files, project rows) with `#2`'s implementation.

## 5. Test coverage notes

The suite asserts through fakes rather than restating the implementation, and I confirmed by mutation that the two round-4 fixes are genuinely pinned (`-raw` → `TestRawNeverPlays`; record newline → `TestRunPlaysThreeTimesByDefault`). Remaining gaps, in priority order:

1. **Mismatched-stream prompt** (I-1) — a test for exactly this configuration exists and stops one assertion short of catching it.
2. **The record's blank separator line** — verified green mutation; the shape assertion pins the suffix only.
3. **`-no-color` replay acknowledgement** — no test observes that a replay under `-no-color` produces nothing at all.
4. **`opt.times == 0` routed into the replay branch** — untested; the misleading message rides on it.
5. **Ctrl-C priority with queued input** — no test; the non-deterministic select is invisible.
6. **Define-path failure with `opt.tty == true`** — the erasable indicator being taken back is covered only on the replay path.

`-race` is clean and the sharing model holds on inspection: `deps` is passed by value so the `d.audio` rewrite at `repl.go:80` is unshared, and both channels carry values. The reader goroutine only sends on `errc` after every line has been consumed, so there is no last-line/EOF ordering hazard.

## 6. Architectural notes

- **ARCH-DRY — pass.** `defineOnce` is the single define path; `playAnnounced` is the single owner of announce → play → erase → report and I verified the two call sites differ only in the `indicator` value; `cachingAudioSource` derives permanence from the existing `ErrNoAudio`/`ErrFetchFailed` taxonomy rather than restating it. Residue is cosmetic (duplicated literal inside the helper, duplicated exit-arm newline, `color`/`tty` identical expressions) — all Minor above.
- **ARCH-PURE — pass, forward note now concrete.** `parseREPLLine` is deterministic and its table test touches no IO, exec, or fs. But the terminal state machine — which gate applies to which write, whether the announcement is a record or ephemeral, whether to `skipPrompt` — is non-trivial policy living in the IO shell, assertable only by `strings.Count` over a buffer, and I-1 lives there. Before `#14` adds a line editor and `#15` adds `/`-command type-ahead to the same loop, extract a pure `func replFrame(ev event, visible, transient bool) string`; the screen contract then becomes a table test and the four terminal questions become named parameters instead of expressions scattered across two files.
- **ARCH-PURPOSE — pass with a flag.** Every Done-when item is delivered and independently verified. Consumers of the flag set (`run`, `defineOnce`, `repl`, `playAnnounced`) all derive from the single `options` construction at `main.go:93-108`; no hand-maintained restatement survives. Shadow-sweep over the consumers of "what invocations and behaviours exist": `-h` (`main.go:75-80`) ✓, `README.md` ✓ (one exit-code nuance and one now-false prompt claim short), `workshop/issues/000002-repl.md` ✓ (properly amended, its own `## Revisions` section), `workshop/issues/000014-repl-editor.md` ✓, `atlas/define.md` ✗ one false sentence (I-2), `workshop/plans/000002-repl-plan.md` ✗ one false sentence plus a stale contract (I-2, I-3). The behavioural consumer that still doesn't derive is the prompt's terminal gate (I-1).
- **ARCH-MOCK — pass with one standing note.** All new external surface runs behind existing seams, and the fact that reverting the production wiring *fails* the suite proves the fake sits on the production path rather than beside it. Still open from rounds 2, 3 and 4: the Ctrl-C contract depends on `exec.CommandContext` actually terminating a real `afplay` mid-file, and `cmd/define/player_conformance_test.go` contains only `TestAfplayBlocksUntilPlaybackCompletes`. That is the one real-binary behaviour this issue newly relies on and does not verify against the real binary. A `ctx` cancelled after ~100 ms asserting `Play` returns early is a five-line addition to a file that already exists. Non-blocking — the operator exercised it on a pty — but this is the fourth deferral of the drift detection the principle asks for.

**Plan-gate carry-forward:** `workshop/plans/000002-repl-plan-gate.md` lists no open findings, and I re-verified PQ-10's disposition is now genuinely true on disk (`plan:154-160` no longer says "report it, and continue" and explains why it cannot). The ledger is honest.

**Core-concepts cross-check:** all six rows verified at their stated paths with their stated status — `replCommand` and `parseREPLLine` (PURE, `repl.go:12-41`), `cachingAudioSource` (`fetch.go:87-131`), `defineOnce` and `stdinIsTerminal` (`main.go:124`, `main.go:27`), `repl` (`repl.go:70`). The PURE rows test without IO; the INTEGRATION row is injected at `repl.go:80` rather than called from business logic. No contradictions.

## 7. Plan revision recommendations

Append a new dated sub-entry to the existing `## Revisions` section of `workshop/plans/000002-repl-plan.md` — don't rewrite the entry that's there:

1. **`plan:74`** — delete "**Failed fetches are not cached**, so a transient outage does not poison the rest of the session"; `ErrNoAudio` is cached as permanent, `ErrFetchFailed` is not. Same sentence as `atlas/define.md:210` (I-2). *Recommended at boundaries 2, 3 and 4.*
2. **Cancellation contract (`plan:36-43`)** — state what Ctrl-C *prints* (nothing, from the single suppression in `playAnnounced`), not only that it exits 0. *Fifth boundary.* This is PQ-4's original residue.
3. **New — round-4 delta.** Record `-raw ⇒ noAudio` decided once at flag parse (so the flag means one thing on both paths), the record-form pin, and the cooked-mode echo limitation now documented in the atlas and `#14`.
4. **New — the terminal question taxonomy is four, not three.** `color` (stdout, presentation), `tty` (stdout **and** `!-no-color`, may I erase), `stdinIsTerminal` (stdin, is a human typing), and — from I-1 — "will the human see this at all" (stdout is a terminal, independent of `-no-color`), which is what the prompt should gate on. Writing this down is what keeps `#14`/`#15` from re-breaking it.
5. **Chunk 1 integration table** — `indicator`, `playAnnounced`, `options` and `scanLines` are shipped entities absent from the table (the first three appear only in Revisions prose). Add the rows so the table stays the greppable inventory it is meant to be.

No revision needed to `workshop/issues/000002-repl.md` — its Spec is amended and its `## Revisions` section matches the code.
