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
