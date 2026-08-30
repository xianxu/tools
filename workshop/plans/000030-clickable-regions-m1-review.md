# Boundary Review — tools#30 (milestone M1)

| field | value |
|-------|-------|
| issue | 30 — clickable regions in the terminal: click ORIGIN French to hear it, click the IPA to replay |
| repo | tools |
| issue file | workshop/issues/000030-clickable-regions.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 168b1c9f3ed7367f122af4795002ad336fd41e02..943f10a9265ea9747ac741e247047d44ba013fa5 |
| command | sdlc milestone-close --issue 30 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-29T20:00:44-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what it committed to: the alternate screen, a pure line-buffer/viewport (`screen`) with a thin IO shell (`liveScreen`), whole-frame paint, `cooked()` deleted, PageUp/PageDown + wheel scrolling, SIGWINCH, and the exit transcript — with the atlas and README rewritten rather than appended to. `go test ./...` is green (101s); the pty conformance rows compile and are correctly routed through `conformance.SkipOrFail`, but **could not be executed in this review environment** ("no pty available: operation not permitted"), so every claim resting solely on a pty row is unverified here and I say so per-finding. Three Important findings block a clean SHIP: the in-process test the plan names as the pin for terminal restoration cannot fail (I deleted `leaveAlt()`+`leaveMouse()` from `restore()` and the whole suite stayed green); enabling mode `1000` without a decoder for its native X10 encoding types characters into the line on any terminal that ignores `1006`; and `Paint` measures the frame in logical lines rather than display rows, so a buffer line wider than the terminal makes the frame overflow and the terminal scroll — the exact failure M1 exists to prevent and M2's hit test will depend on.

## 1. Strengths

- **`screen`/`liveScreen`/`display` is a textbook pure-core/thin-shell split (ARCH-PURE).** `screen.go:25-160` does no terminal IO at all; `screen_test.go` drives every piece of the scroll arithmetic with a `strings.Builder`. `runEditor` takes a `display` interface (`replraw.go:103-117`) and a key channel, so `commandloop_test.go`/`askroute_test.go` drive the whole loop with no terminal anywhere.
- **The M1.3b prompt fix is genuinely pinned.** I reverted `view.Draw("", nil)` (`replraw.go:312`) in a scratch copy: `TestNothingIsWrittenWhileAPromptIsShown` goes red with the exact stale prompt from the operator's screenshot (`"\r\x1b[K…› sycophantic"`). Same for the wheel — deleting the `KeyWheelUp/Down` cases reddens `TestWheelScrollsRatherThanWalkingHistory` with `moved [] want [3 -3]`.
- **The rewritten tests are honest about what they lost.** `TestSubmitClearsTheMenuBeforeOutput` → `TestSubmitLeavesNoMenuInTheFrame` and `TestRawLoopMessagePlacement`'s move to a `screen` as stderr each carry the reason the old observable died, rather than being quietly deleted with the mechanism.
- **`decodeWheel` reuses the existing CSI scan rather than a second delimiter** (`key.go:144`, `170-196`), and `TestDecodeKeyPageKeys` explicitly pins `ESC[5;5~` → `KeyUnknown`/6 bytes, which is #14's bug asserted as a negative.
- **Docs are ahead of the code, not behind it.** `atlas/define.md` gains a full "The screen" section and rewrites the now-false raw-mode prose as history; `lessons.md` keeps "render cooked, play raw" with the note that #30 dissolved it; README gains the scroll keys, the resize behaviour, the Option escape and the exit transcript.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `cmd/define/rawterm_test.go:110-127`: the named pin for terminal restoration cannot fail.**
`TestRestoreLeavesTheAlternateScreen` builds `&rawSession{f: nil}`, so `enterAlt()` returns at the `r.f == nil` guard (`rawterm.go:135`) and `r.alt` is never set; `restore()`'s `leaveAlt()` returns at the same guard; the assertion `if r.alt` then checks a field that was never true. Verified by reverting: I replaced `r.leaveMouse()` and `r.leaveAlt()` in `restore()` (`rawterm.go:44-48`) with comments and ran the full in-process suite — the only failures were the 12 git-dependent repo guards that also fail on an unmodified scratch copy. M1 done-when rows 3 and 3b therefore rest entirely on `-tags conformance` pty rows I could not run here. The vestigial `var b strings.Builder; _ = b` at line 118 shows the author reached for an injected writer and gave up because `rawSession.f` is typed `*os.File`.
*Fix sketch:* give `rawSession` a `w io.Writer` for escape output (keeping `fd int` for `term.Restore`), then assert the bytes AND the order — `mouseOff` before `altScreenOff` — in-process. ARCH-MOCK: the terminal has a stateful double for keystrokes but none for the three-state restore protocol.

**I2 — `cmd/define/key.go:130-147`: mode `1000` is enabled but its native X10 encoding is undecoded, so a click types characters into the line.**
`rawterm.go:170` sends `\x1b[?1000h\x1b[?1006h`. A terminal that honours `1000` and ignores `1006` reports in X10: `ESC[M` + three raw bytes. `decodeEscape` treats `M` (0x4D) as the final byte, returns `KeyUnknown` after 3 bytes, and the three payload bytes fall through to `KeyRune`. Measured on a scratch copy: a left click at (1,1) decodes as `KeyUnknown` then `' '`, `'!'`, `'!'` — it types `" !!"` into the word being looked up. A wheel notch is worse: `0x60` → `` `!! `` three times per notch. This is #14's family exactly (`key.go:162-164` cites it), and the module comment's claim that a click is "consumed whole and inert" holds only for SGR-encoded reports. Not Critical because I have no evidence a terminal in the supported set (iTerm2/Terminal.app/Ghostty) lacks `1006`; the fix is ~5 lines.
*Fix sketch:* in `decodeEscape`, special-case `seq == "\x1b[M"` — require `len(buf) >= 6`, return `Key{}, 0` if short (the partial-sequence protocol already exists), else consume 6 and return `KeyUnknown`, or decode the X10 wheel. Add a `TestDecodeKey` row asserting a click consumes 6 bytes and emits no rune.

**I3 — `cmd/define/screen.go:196-224`: the frame is counted in logical lines, not display rows, so a wide buffer line makes the terminal scroll.**
`Paint` sets `s.rows = termRows - 1 - len(menu)` and writes `line + "\r\n"` with no clipping to the terminal's width. Measured: 10 buffer lines of 200 characters, painted at `termRows=10` into an 80-column terminal, needs **28 display rows**. Two reachable triggers, both routine: (a) a narrowing resize — `replraw.go:256` updates `opt.width` for future entries only, and the plan deliberately decided "lines already in the buffer keep their wrapping", which guarantees over-wide lines after any shrink; (b) typing a line longer than the terminal width, since the committed line goes to the buffer via `fmt.Fprint(stdout, RenderLine(...))` (`replraw.go:314`) with no truncation. The consequence is precisely what `atlas/define.md` names as the reason resize matters — "a frame one row too tall makes the terminal scroll, which moves every row the app believes it placed" — i.e. the exact-coordinate property that is M1's entire purpose and M2's `RegionAt` precondition. `TestPTYResizeRepaints` cannot catch it: it counts `\r\n` (logical rows) and resizes rows only, holding cols at 80. `screen.cols` is declared at `screen.go:37`, named in the plan's Core-concepts entry, and never set or read anywhere in the tree. (ARCH-CONSTRAINTS, ARCH-PURPOSE)
*Fix sketch:* set `s.cols` from the resize and clip each frame line to it in `Frame`/`Paint` using an ANSI-aware width (`truncate` in `command.go:305` is the nearest existing helper — check it counts escape bytes as zero-width). Clipping at paint time keeps the buffer's full text for the transcript. Add a unit test asserting total display rows ≤ `termRows` for a buffer of over-wide lines.

## 4. Minor findings

- `cmd/define/askroute_test.go:386-393` — the `wantErase` row is unfalsifiable: I removed `eraseLine` from **both** `define: …` sites in `replraw.go:372,419` and the full suite stayed green. The test gives stderr its own `screen`, so the erase has nothing to take back; in production stdout and stderr are one `liveScreen`. Related: after `view.Draw("", nil)` blanks the live edge and the `cmdNothing`/`cmdReplay` branch skips writing the committed line, those two `eraseLine` prefixes appear to be vestigial — but their code comments still assert the pre-screen rationale ("without it the note is appended to the line the user typed"). Decide which: pin it against a shared screen, or drop the gesture and the comment.
- `cmd/define/screen.go:270` (`liveScreen.Write`) — a whole-frame repaint per write, so a streamed answer redraws up to `rows` lines per delta, with no coalescing and no synchronized-output guard (`\x1b[?2026h/l`) around `cursorHome + eraseDown`. Cheap mitigations exist; I did not measure flicker. (ARCH-CONSTRAINTS)
- `cmd/define/screen.go:29` — `screen.lines` grows unboundedly for the session and is printed whole at exit; no cap, no envelope declared in the plan.
- `cmd/define/rawterm.go:134-192` — the alt-screen and mouse sequences are written to `r.f`, which is **stdin**, while frames and the height probe go to stdout. Correct whenever both are the same tty (which `terminalUI` nearly guarantees), but it is a new assumption this diff introduces; previously `rawSession` only did `tcsetattr` on that fd.
- `cmd/define/rawterm.go:222-238` — the coalescing hand-off is a four-level nested select where `select { case <-out: default: }` then `select { case out <- sz: default: }` suffices (single producer).
- `cmd/define/rawterm.go:203` — `watchResize` never `signal.Stop`s its channel on `ctx.Done()`; harmless for a CLI, but the registration outlives the goroutine.
- `cmd/define/editorloop_test.go:187` — stale rationale: "anything written before dropping back to cooked mode" describes a mode D4 deleted. The assertion itself now pins `\r\n` bytes that `screen.Write` strips.
- `cmd/define/screen.go:224` — `Paint` discards the `fmt.Fprint` error entirely, so a failing tty is invisible. Consistent with the surrounding code; noting it because the checklist asks about swallowed errors.

## 5. Test coverage notes

- **Falsifiability checked by reverting, in scratch copies (working tree untouched):** M1.3b's prompt blanking → red ✅; the wheel cases → red ✅; `restore()`'s `leaveAlt`/`leaveMouse` → **green** ❌ (I1); `eraseLine` on the `define:` notes → **green** ❌ (Minor). The 12 repo-guard failures in a scratch tree are git-dependent and present on an unmodified copy too.
- **Not runnable here:** all 11 `TestPTY*` rows skip with "no pty available: operation not permitted". That covers M1 done-when rows 3 (partly), 3b, 5 and 6 — including the only test of the D3 exit transcript and the only test of `mouseOff`. The `CONFORMANCE_STRICT` mechanism means CI would catch a skip, so this is an environment limit rather than a design gap, but no one has *seen* those rows pass in this review.
- **`finish()`'s once-only guard (`replraw.go:55-78`) has no test at all.** Printing the session twice is exactly what an idempotent call does not fix — the reason the guard exists — and nothing pins it; `replRaw` is unreachable in-process because it demands `stdin.(*os.File)` (an `os.Pipe` would suffice).
- Coverage that is genuinely strong: `TestScreenFrame` (both clamps), `TestScreenTakesBackAnErasedLine` (including the "erase with no open line" case), `FuzzDecodeWheelIsBounded`, `TestCtrlDStillEndsTheSession`'s key-behind-the-Ctrl-D discriminator, and `TestWatchResizeCoalesces`.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** One line-ending owner per path (`crlfWriter` scoped to `--play` by D5a), `menuDrawn`/`clearMenu` deleted rather than ported, `replayInPlace` still the single replay for both a bare Enter and `/pron`.
- **ARCH-PURE — pass**, and the strongest property of the diff. Keep it that way in M2: `Region` positions are computed after wrapping, so make the region walk a pure function over the rendered string and let `screen` own the splice (PQ-11 already says this).
- **ARCH-PURPOSE — flag.** M1.4b answered a class ("a terminal gesture arrives as bytes the editor mistakes for typing") at the site the operator reported. I2 is a live sibling of that same class, in the same decoder, introduced by the same commit that enabled tracking. The enumeration the finding implies is: for every mode we enable, does the decoder handle **every** encoding that mode can answer in? Sweep it now, not in M2.6.
- **ARCH-MOCK — partial.** The pty harness is the right seam and M1 added three rows to it. The gap is `rawSession`: it is the one integration point with no in-process double, which is why I1 exists. Fixing I1 fixes the pattern.
- **ARCH-CONSTRAINTS — flag.** The plan declares no operating envelope for a program that is now full-screen. The three concrete gaps are I3 (no display-row budget, and `cols` inert), repaint-per-delta, and unbounded buffer growth. M2 adds a per-line region list to the same buffer, so state the envelope before that lands.
- **API shape for M2:** `runEditor` is at 10 positional parameters and `display` at 4 methods; M2 adds at least `RegionAt` plus a click action. Consider folding `view`/`finish`/`stdout`/`stderr` into one struct before the M2 churn rather than after — `M2.1` already promises a wide mechanical diff.

## 7. Plan revision recommendations

Add one `## Revisions` entry to `workshop/plans/000030-clickable-regions-plan.md`:

1. **M1's Core-concepts table under-declares what M1 shipped.** `liveScreen`, `display`, `screen.Page`, `screen.Transcript`, `enterMouse`/`leaveMouse`, `decodeWheel`/`atoiPrefix`, `winSize` and `terminalRows` are all new entities named only in prose or in later Revisions entries. The table is the greppable registry the boundary cross-check reads; add the rows with kind (PURE/INTEGRATION), path and status. `liveScreen` is INTEGRATION, `decodeWheel` PURE.
2. **`screen.cols` is declared in the plan's model ("`lines []string`, `offset int`, `rows, cols int`") and inert in the code.** Either record that the frame is measured in logical lines and that clipping to `cols` is deferred to M2.1 as a stated precondition of `RegionAt`, or fix it in M1 and say so. Leaving the plan claiming a width-aware viewport that does not exist is the `unbacked-existing-behavior` class the plan spent PQ-10 on, one layer up.
3. **Done-when row 3's in-process pin does not discriminate.** The row names `TestRestoreLeavesTheAlternateScreen`; that test passes with `leaveAlt` deleted from `restore()`. This is the third instance of the family the "two rows named tests that do not exist" revision opened — record it and the fix, so row 3b (`mouseOff`) is not left pty-only either.
4. **M2.6 ("degrade") should name the X10 case explicitly.** As written it covers "a terminal that reports no mouse"; the failing case is a terminal that reports the mouse in an encoding we did not ask for, and the exposure started in M1.4b, not M2.

```findings
findings:
  - id: new
    severity: Important
    family: unfalsifiable-test-pin
    title: |
      TestRestoreLeavesTheAlternateScreen passes with leaveAlt and leaveMouse deleted from restore()
    detail: |
      rawterm_test.go:110 builds `&rawSession{f: nil}`, so enterAlt returns at the
      nil-file guard and `alt` is never set; the assertion checks a field that was
      never true. Verified by reverting: replacing `r.leaveMouse()` and `r.leaveAlt()`
      in restore() (rawterm.go:44-48) leaves the entire in-process suite green (only
      the 12 git-dependent repo guards fail, as they do on an unmodified scratch copy).
      M1 done-when rows 3 and 3b therefore rest solely on conformance-tagged pty rows,
      which skip when no pty is available. Fix: type rawSession's escape-output target
      as io.Writer (keeping fd int for term.Restore) and assert the bytes AND the
      order, mouseOff before altScreenOff, in-process. ARCH-MOCK.
  - id: new
    severity: Important
    family: decoder-consumes-whole-sequence
    title: |
      Mode 1000 is enabled but its native X10 encoding is undecoded, so a click types characters into the line
    detail: |
      rawterm.go:170 sends ESC[?1000h ESC[?1006h. A terminal that honours 1000 and
      ignores 1006 reports in X10 as ESC[M plus three raw bytes. decodeEscape
      (key.go:130) treats M as the final byte, returns KeyUnknown after 3 bytes, and
      the three payload bytes fall through to KeyRune. Measured: a left click at (1,1)
      types " !!" into the word being looked up; an X10 wheel notch types backtick-bang-bang
      three times. This is exactly the family key.go:162 cites from issue 14, and the
      module comment's claim that a click is "consumed whole and inert" holds only for
      SGR reports. Fix: special-case seq == ESC[M in decodeEscape, return Key{},0 when
      fewer than 6 bytes are buffered, else consume 6 and return KeyUnknown; add a
      decodeKey row asserting 6 bytes consumed and no rune emitted.
  - id: new
    severity: Important
    family: frame-fits-the-terminal
    title: |
      Paint counts the frame in logical lines, not display rows, so a wide buffer line scrolls the terminal
    detail: |
      screen.go:196 sets rows = termRows - 1 - len(menu) and writes each line without
      clipping to the terminal width. Measured: 10 buffer lines of 200 characters at
      termRows=10 in an 80-column terminal needs 28 display rows. Reachable two ways,
      both routine — a narrowing resize (replraw.go:256 updates opt.width for future
      entries only, and the plan deliberately keeps existing lines' wrapping), and a
      typed line longer than the terminal width, since the committed line reaches the
      buffer un-truncated at replraw.go:314. The consequence is the failure the atlas
      names as the reason resize matters: the terminal scrolls and every row the app
      believes it placed moves, which is the exact-coordinate property M1 exists to
      establish and M2's RegionAt depends on. TestPTYResizeRepaints cannot catch it —
      it counts logical rows and resizes height only. screen.cols is declared at
      screen.go:37, named in the plan's model, and never set or read. Fix: set cols on
      resize and clip frame lines with an ANSI-aware width at paint time, keeping the
      buffer's full text for the transcript. ARCH-CONSTRAINTS, ARCH-PURPOSE.
  - id: new
    severity: Minor
    family: unfalsifiable-test-pin
    title: |
      TestRawLoopMessagePlacement's wantErase row passes with eraseLine removed from both define: sites
    detail: |
      Removing eraseLine from replraw.go:372 and replraw.go:419 leaves the full suite
      green. The test gives stderr its own screen, so the erase has nothing to take
      back; in production stdout and stderr are one liveScreen. Separately, after
      view.Draw("", nil) blanks the live edge and the cmdNothing/cmdReplay branch skips
      writing the committed line, those two eraseLine prefixes look vestigial — while
      their comments still assert the pre-screen rationale. Decide which: pin it
      against a shared screen, or drop the gesture and the comment.
  - id: new
    severity: Minor
    family: coalesce-ui-work
    title: |
      A whole-frame repaint per streamed delta, with no coalescing and no synchronized-output guard
    detail: |
      liveScreen.Write (screen.go:270) repaints on every write, so a streamed answer
      redraws up to `rows` lines per token via cursorHome + eraseDown with no
      ESC[?2026h/l bracket. Related, same envelope: screen.lines grows unboundedly for
      the session and is dumped whole at exit. The plan declares no operating envelope
      for what is now a full-screen program. Not measured — flagged as undeclared
      rather than as observed flicker. ARCH-CONSTRAINTS.
  - id: new
    severity: Minor
    family: plan-table-under-declares
    title: |
      M1's Core-concepts table omits liveScreen, display, enterMouse/leaveMouse, decodeWheel, screen.Page
    detail: |
      Those entities are named only in prose or in later Revisions entries, but the
      table is the greppable registry the boundary cross-check reads. Add rows with
      kind, path and status — liveScreen INTEGRATION, decodeWheel PURE — alongside
      winSize and terminalRows.
  - id: new
    severity: Minor
    family: stale-rationale
    title: |
      Stale cooked-mode prose and small residue left by D4
    detail: |
      editorloop_test.go:187 still says "before dropping back to cooked mode" for a
      mode D4 deleted, and its assertion pins \r\n bytes that screen.Write strips.
      Also: rawterm.go:134-192 writes terminal-state escapes to the STDIN handle while
      frames and the height probe go to stdout (a new assumption, correct only while
      both are the same tty); rawterm.go:222-238 nests four selects where a drain-then-
      send pair suffices with a single producer; watchResize never signal.Stops its
      channel; Paint discards the tty write error entirely.
```

---

## Re-review — 2026-08-29T20:08:39-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 30 — clickable regions in the terminal: click ORIGIN French to hear it, click the IPA to replay |
| repo | tools |
| issue file | workshop/issues/000030-clickable-regions.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 168b1c9f3ed7367f122af4795002ad336fd41e02..943f10a9265ea9747ac741e247047d44ba013fa5 |
| command | sdlc milestone-close --issue 30 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-29T20:08:39-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 lands a genuinely well-shaped screen layer: `screen` is a pure line buffer + viewport with real table tests and no pty anywhere, `liveScreen` is a thin IO shell, and the `display` interface lets `runEditor` be driven with no terminal at all — the ARCH-PURE split is the best thing in this diff. The two operator-reported regressions (the duplicated prompt, the wheel walking history) are both fixed and both pinned by tests I mutation-verified go red. What blocks SHIP is one correctness bug and one falsified invariant: enabling `?1000h` without handling the **legacy X10 mouse encoding** means any terminal that honours 1000 but ignores 1006 injects three literal characters into the line being typed on every wheel notch (probe-verified: `` ` ``, `*`, `%` arrive as `KeyRune`) — the exact `#14` under-consumption class the plan claims to have defended; and `Paint` budgets the frame in **logical lines, never columns** (`screen.cols` is declared and never assigned or read), so any line wider than the terminal makes the frame taller than the viewport and the alternate screen scrolls — which is precisely the invariant M1 exists to establish and M2's hit test will depend on. Beside those, the in-process pin the plan names for done-when row 3 is vacuous (mutation-verified: deleting `r.leaveAlt()` from `restore()` leaves the whole default suite green), the M1.6 prose sweep fixed the sites it looked at and left five more claiming the raw loop wraps stdout in `crlfWriter`, and this window introduces two data races that make `go test -race ./cmd/define/` fail where the base commit is clean.

### 1. Strengths

- **`cmd/define/screen.go:126-160` — the pure core is real, not nominal.** `Frame`/`Scroll`/`Page` are unit-tested with no terminal, no mock, no pty (`screen_test.go`), and `liveScreen` holds the only IO. The `display` interface (`replraw.go:103-117`) is the right seam — `runEditor` is fully drivable from a scripted channel, which is what let the M1.4a/M1.4b behaviour be pinned in-process at all.
- **`cmd/define/replraw.go:312` — the live-edge blank is a real fix with a real pin.** I mutated it out; `TestNothingIsWrittenWhileAPromptIsShown` reddens with the exact stale prompt from the operator's screenshot (`write 1 of 7 landed with a prompt on the frame: "…› …sycophantic"`). And the test pins the *property* ("nothing is written while a prompt is on the frame") plus the guard that the prompt returns, so blanking it forever cannot pass.
- **`cmd/define/key.go:170-196` — the wheel decoder is delimited, not length-guessed**, and `TestDecodeKeyPageKeys` correctly rejects `ESC[5;5~`. Mutating the wheel kinds back to `KeyUp`/`KeyDown` reddens six `TestDecodeWheel` rows.
- **`cmd/define/rawterm.go:38-56` — one restore guarantee for three terminal states, unwound in the right order** (mouse, then alt screen, then line discipline), with the reason for the ordering written down.
- **`cmd/define/screen.go:56-90` — honouring `eraseLine` in the buffer** is the correct call: it keeps `♫ playing 3×` out of the exit transcript, which is the "ephemeral UI vs record" doctrine holding in the direction it exists for.

### 2. Critical findings

**C1 — `cmd/define/key.go:130-147`: a legacy X10 mouse report types three characters into the line.**
`replRaw` enables `?1000h` *and* `?1006h`, but a terminal that honours 1000 and ignores 1006 sends the legacy form `ESC [ M Cb Cx Cy` (each coordinate byte offset by 32). `decodeEscape` finds `M` at index 2 as the final byte, hands `"\x1b[M"` to `decodeWheel`, which rejects it on `len(seq) < 4`, and returns `KeyUnknown` consuming **3** bytes. The three coordinate bytes then decode as ordinary text. Probe-verified against the tree:

```
in: 1b 5b 4d 60 2a 25   (wheel-up at col 10, row 5)
key 0: kind=KeyUnknown raw="\x1b[M"
key 1: kind=KeyRune rune='`'
key 2: kind=KeyRune rune='*'
key 3: kind=KeyRune rune='%'
```

macOS Terminal.app is the commonly-cited terminal in this category, and the code comment at `rawterm.go:166` names it as a target. On such a terminal M1 pays D7's drag-select cost and gets garbage instead of scrolling. Fix sketch: in `decodeEscape`, special-case `ESC[M` before the parameter scan — require 6 bytes total (return `0` consumed if fewer, so a split read waits), decode `Cb-32` through the same wheel logic as `decodeWheel`, and return `KeyUnknown` with all 6 bytes consumed for a button. Add a `TestDecodeLegacyMouse` row and extend `FuzzDecodeWheelIsBounded`'s seeds with `"\x1b[M"` and `"\x1b[M\x60\x2a\x25"`.

### 3. Important findings

**I1 — `cmd/define/screen.go:37,200`: the frame's row budget ignores columns, so the app does not in fact own every row (ARCH-PURPOSE, ARCH-CONSTRAINTS).**
`Paint` sets `s.rows = termRows - 1 - len(menu)` and emits one `\r\n`-terminated line per *logical* buffer line. It never consults a width; `screen.cols` (line 37) is declared and **never assigned or read anywhere in the tree**. Any buffer line wider than the terminal wraps, so the frame occupies more terminal rows than `termRows`, the alt screen scrolls, and every row the app believes it placed moves — the exact failure the atlas and plan describe as the reason resize matters. Reachable three ways today: (a) a long committed line, since `RenderLine(submitted, …)` at `replraw.go:314` is buffered verbatim and the user can type any length; (b) after *any* narrowing resize, because "lines already in the buffer keep their wrapping" is a declared decision and nothing compensates for it in the row budget; (c) `terminalWidth` returns `0` ("do not wrap") below 20 columns, so a narrow terminal buffers unwrapped entries. `TestPTYResizeRepaints` cannot catch this — it counts `\r\n`, i.e. logical lines. Fix sketch: give `screen` a real `cols` (set from `Paint`'s caller alongside `termRows`) and have `Frame` charge each line `ceil(visibleWidth(line)/cols)` rows against the budget, using the existing SGR-aware width helper rather than `len`. This should land before M2, whose `RegionAt(row, col)` is only correct if the row budget is.

**I2 — `cmd/define/rawterm_test.go:112`: `TestRestoreLeavesTheAlternateScreen` asserts nothing (ARCH-MOCK).**
It builds `&rawSession{f: nil}`, so `enterAlt()` returns at its `r.f == nil` guard and `r.alt` is never set; the final `if r.alt` is then trivially false. Mutation-verified: I deleted `r.leaveAlt()` from `restore()` and the entire default `go test ./cmd/define/` suite stayed green (only the git-dependent repo guards fail in a non-git scratch, unrelated). The test also carries dead code (`var b strings.Builder; _ = b`). The plan's done-when row 3 therefore rests solely on `TestPTYTranscriptIsPrintedOnExit`, which is `//go:build darwin && conformance` and skips when no pty is available — as it does in this review environment (`no pty available: operation not permitted`), so I could not independently confirm any of rows 3, 3b, 5 or 6. Fix sketch: `rawSession.f` is used only as an `fmt.Fprint` target; widen it to an `io.Writer` (keeping the `*os.File` for `term.Restore` via `fd`) and assert in-process that `restore()` emits `mouseOff` before `altScreenOff`, and that a second call emits nothing. That makes the ordering — currently pinned nowhere in-process — testable.

**I3 — stale prose the M1.6 sweep left behind: five sites still say the raw loop wraps stdout in `crlfWriter` (ARCH-PURPOSE — the instance, not the class).**
D5 removed `crlfWriter` from the interactive path, and no caller of `ask` wraps stdout in it any more (verified: `main.go:659,686`, `repl.go:299`, `replraw.go:223` all pass the raw writer). Still claiming otherwise: `cmd/define/ask.go:32` ("a raw terminal needs crlfWriter and a prompt redrawn"), `cmd/define/ask.go:168` ("the raw loop has already wrapped stdout in crlfWriter (replraw.go), so this sits INSIDE it"), `cmd/define/askhighlight_test.go:240` (same claim in the doc comment), `atlas/define.md:708` ("`crlfWriter`, which the raw loop nests this inside"), `atlas/define.md:747` ("The raw loop has already wrapped stdout in `crlfWriter` before `ask` is called"). `TestHighlightingNestsInsideCRLFTranslation` now pins a composition no production path builds. The repo guards cannot catch this, because `crlfWriter` still exists for `--play`. Fix sketch: rewrite the five sites to say the interactive path streams into the screen and `--play` keeps `crlfWriter`; either re-scope the test's comment to `--play` or drive it through `--play`'s actual writer stack.

**I4 — `cmd/define/editorloop_test.go:494,530`: two data races introduced by this window.**
`TestEditorResizeRedrawsForTheNewShape` polls `view.rows` (written by `recordDisplay.Resize` on the loop goroutine, line 82) through `waitFor` on the test goroutine; `TestWatchResizeCoalesces` polls `measured`, which the `watchResize` goroutine increments. `go test -race -count=1 ./cmd/define/` fails on both; the same command on base `168b1c9` reports no `DATA RACE` at all, so these are new. Beyond the tooling, `waitFor` spinning on an unsynchronised variable can legitimately spin to its 5s `t.Fatal`. Fix sketch: guard both with a `sync.Mutex` (or make `recordDisplay` mutex-protected wholesale, since `prompts`/`menus` are read the same way) or have `Resize` also send on a buffered channel the test receives from.

**I5 — `cmd/define/screen.go:262-301` + `cmd/define/ask.go:190`: one full-screen clear-and-redraw per streamed delta, with no coalescing (ARCH-CONSTRAINTS).**
`runAsk` calls `fmt.Fprint(out, delta)` once per streaming delta; every such write goes through `liveScreen.Write` → `repaint()` → `Paint`, which emits `\x1b[H\x1b[J` plus the entire visible frame. A few hundred deltas per answer on a 50-row terminal is hundreds of full-screen erase+redraw cycles and megabytes to the tty for one answer. The plan declares no operating envelope for this path at all — no latency budget, no bound on repaint work, no statement of what happens under a fast stream. Fix sketch: coalesce in `liveScreen` — repaint on a trailing timer (~30–60 Hz) with a forced flush on `Draw`, `Page`, `Scroll` and `Stop`, so a burst of deltas costs one frame; and state the budget in the plan (keystroke → one frame; stream → ≤N frames/sec).

### 4. Minor findings

- `screen.go:126-160` — the offset clamp is spelled twice (`Frame` clamps a *local* `off`; `Scroll` clamps `s.offset`). Because `Frame` doesn't write back, after a widening resize `s.offset` can stay out of range and the first wheel-down is a visual no-op. One `clampedOffset()` owner would fix both (ARCH-DRY).
- `replraw.go:55-60` — `handedBack` is unreachable: every `finish()` call site (`:238`, `:262`, `:290`) returns immediately after, so `finish` cannot run twice today. Defensible as forward protection, but it currently reads as a guarantee nothing exercises.
- `rawterm.go:138,149,182,191` — the alt-screen and mouse sequences are written to the **stdin** handle while frames go to stdout, and the `fmt.Fprint` errors are discarded; a failed write still sets `r.alt`/`r.mouse`. Writing them to the same handle the frames go to (or checking the error) removes the asymmetry.
- `replraw.go:314` — when the cursor is mid-line at submit, `RenderLine`'s trailing `\x1b[<n>D` is stored in the buffer and re-emitted into the exit transcript. Harmless today (a `\r\n` follows), but it is a control sequence buffered as text, which `TestEditorLoopWritesThroughAScreen` only checks for `\x1b[K`.
- `README.md` — "so it is in your scrollback exactly as it was before" overstates: the erased indicator and the live edge (prompt/menu) are deliberately absent from the transcript.
- `rawterm_test.go:112` — dead `var b strings.Builder; _ = b`.

### 5. Test coverage notes

- **Pinned and mutation-verified by me:** the live-edge blank (`TestNothingIsWrittenWhileAPromptIsShown`), the wheel decoder (`TestDecodeWheel`, six rows red under mutation). The `screen` arithmetic tests are genuinely pure — no IO, no mocks — which satisfies the PURE half of the core-concepts table.
- **Not pinned:** `leaveAlt` in the restore path (I2, verified vacuous). `finish`'s once-only property. The `replRaw` wiring as a whole — `enterAlt`, `enterMouse`, the restore *ordering*, and the transcript print — has no in-process coverage because `rawSession` and `replRaw` are typed to `*os.File`. Rows 3, 3b, 5 and 6 of M1's done-when all depend on pty conformance rows that skip without a pty; I could not run them here, so the M1.3–M1.5 "verified on a real pty" Log claims are unconfirmed by this review.
- **Coverage gap the diff could ship:** no test exercises a buffer line wider than the terminal (I1), and none exercises a non-SGR mouse report (C1). Both are the kind of bug this diff actually ships.
- `TestScreenPaintSplitsTheHeight`'s third and fourth rows pass `wantLines: nil`, so their loop body never executes — they assert only that a prompt is present. Worth an explicit "the frame contains no buffer line" assertion.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass, one nit.** Replacing `crlfWriter` on the interactive path rather than running two line-ending owners is the right call, and `replayInPlace` stays the single replay path. The duplicated offset clamp is the only repetition.
- **ARCH-PURE — pass, and it is the diff's strongest axis.** `screen` (pure) / `liveScreen` (IO) / `display` (injected) is a clean three-way split and the tests prove it. The one place purity leaks is `rawSession`, which is typed to `*os.File` where an `io.Writer` would do — that is what makes I2 possible.
- **ARCH-PURPOSE — flag (I1, I3).** M1's purpose is "the app places every row"; the column dimension was designed in (`cols`) and then dropped, so the invariant is not enforced. And M1.6 answered "rewrite the prose D4 made false" at the sites it looked at rather than by enumerating the class — five sites survive.
- **ARCH-MOCK — flag (I2).** The pty harness is the right stateful double for the terminal and gained the right rows, but the in-process substitute at the `rawSession` seam is vacuous, and production flow and test flow do not share a boundary there (production writes to a `*os.File`; the test can only pass `nil`).
- **ARCH-CONSTRAINTS — flag (I5).** No envelope is declared for the full-screen path. Before M2, the relevant budgets to write down are: keystroke → one frame; streamed delta → coalesced frames; and the frame-fits-the-viewport invariant that I1 breaks, since M2's `RegionAt(row, col)` is only sound while it holds. `readKeys`' escape buffer is also unbounded on a never-terminating CSI — pre-existing, but mouse reports make it a busier path.

### 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000030-clickable-regions-plan.md` covering:

- **The M1 Core concepts tables are incomplete.** `liveScreen`, `display`, `enterMouse`/`leaveMouse`, `terminalRows`/`defaultRows`, `decodeWheel`/`atoiPrefix`, and the four new `KeyKind`s (`KeyPageUp`, `KeyPageDown`, `KeyWheelUp`, `KeyWheelDown`) are all new entities M1 ships. They are described in the Revisions prose but absent from the greppable tables, which is what a reader (and the plan-table guards) actually check. Add them with kind/location/status.
- **`screen.cols` is listed implicitly as part of the pure model but is dead.** Either the table row should say the viewport is rows-only, or — better, per I1 — the plan should commit to a column-aware frame budget and say so.
- **The M1 done-when table's row 3 names a pin that does not pin.** The commit `aa7e2b8` corrected two invented test names; row 3's replacement (`TestRestoreLeavesTheAlternateScreen`) is vacuous under mutation. The row should either name only the pty Fatal, or the plan should carry the in-process assertion I2 recommends.
- **M2.6 ("degrade — a terminal that reports no mouse must behave exactly as M1 does") does not cover the case C1 describes** — a terminal that *does* report the mouse but not in SGR. Add a row for the legacy X10 encoding, and note that M1.4b moved the tracking enable into M1 while leaving its degrade row in M2.

```findings
findings:
  - id: new
    severity: Critical
    family: escape-sequence-underconsumption
    title: |
      a legacy X10 mouse report injects three characters into the line being typed
    detail: |
      replRaw enables ?1000h and ?1006h, but a terminal honouring 1000 without 1006
      sends "ESC [ M Cb Cx Cy". decodeEscape (key.go:130) finds M as the final byte,
      decodeWheel rejects the 3-byte sequence on len(seq) < 4, and the three
      coordinate bytes then decode as KeyRune. Probe-verified against the tree: a
      wheel-up at col 10 row 5 yields KeyUnknown plus the runes ` * %. This is the
      exact #14 under-consumption class the plan claims to have defended. Fix:
      special-case ESC[M before the parameter scan, require six bytes (return 0
      consumed if fewer), decode Cb-32 through the same wheel logic, and consume all
      six for a button. Add a decoder row and two fuzz seeds.
  - id: new
    severity: Important
    family: app-owns-every-row
    title: |
      Paint budgets the frame in logical lines and never consults a column width
    detail: |
      screen.cols (screen.go:37) is declared and never assigned or read anywhere in
      the tree, and Paint (screen.go:200) charges one row per logical buffer line.
      Any line wider than the terminal wraps, so the frame exceeds termRows and the
      alternate screen scrolls, moving every row the app believes it placed — the
      invariant M1 exists to establish and M2's RegionAt depends on. Reachable via a
      long committed line (replraw.go:314 buffers RenderLine verbatim), via any
      narrowing resize (buffer lines keep their old wrapping by decision), and below
      20 columns where terminalWidth returns 0 meaning "do not wrap".
      TestPTYResizeRepaints counts \r\n, so it measures logical lines and cannot see
      this. Fix: set a real cols alongside termRows and charge each line
      ceil(visibleWidth/cols) against the budget.
  - id: new
    severity: Important
    family: vacuous-pin
    title: |
      TestRestoreLeavesTheAlternateScreen asserts nothing; deleting leaveAlt from restore stays green
    detail: |
      The test builds &rawSession{f: nil}, so enterAlt() returns at its f == nil guard
      and r.alt is never set; the closing "if r.alt" is trivially false. Verified by
      mutation: with r.leaveAlt() removed from restore(), the whole default
      go test ./cmd/define/ suite passes. Done-when row 3 therefore rests only on the
      pty Fatal in TestPTYTranscriptIsPrintedOnExit, which skips without a pty (it
      skipped in this review environment), as do rows 3b, 5 and 6. Fix: widen
      rawSession.f to an io.Writer so restore's sequence and ORDER (mouseOff before
      altScreenOff) can be asserted in-process. Also drop the dead
      "var b strings.Builder; _ = b".
  - id: new
    severity: Important
    family: stale-prose-after-removal
    title: |
      five sites still claim the raw loop wraps stdout in crlfWriter, which D5 removed
    detail: |
      No caller of ask wraps stdout in crlfWriter any more (main.go:659,686,
      repl.go:299, replraw.go:223 all verified). Still asserting otherwise: ask.go:32,
      ask.go:168, askhighlight_test.go:240, atlas/define.md:708, atlas/define.md:747.
      TestHighlightingNestsInsideCRLFTranslation now pins a composition no production
      path builds. The repo guards cannot catch it because crlfWriter still exists for
      --play. M1.6 swept the sites it looked at rather than enumerating the class.
  - id: new
    severity: Important
    family: unsynchronised-test-observation
    title: |
      two new tests race; go test -race fails where the base commit is clean
    detail: |
      TestEditorResizeRedrawsForTheNewShape (editorloop_test.go:494) polls view.rows,
      written by recordDisplay.Resize on the loop goroutine; TestWatchResizeCoalesces
      (:530) polls `measured`, incremented by the watchResize goroutine. Both are
      reported by -race; the same command on base 168b1c9 reports no DATA RACE. Beyond
      tooling, waitFor spinning on an unsynchronised variable can spin to its 5s
      t.Fatal. Fix: mutex-guard recordDisplay and the counter, or hand the resize back
      over a buffered channel the test receives from.
  - id: new
    severity: Important
    family: unbounded-ui-repaint
    title: |
      every streamed delta triggers a full-screen clear and redraw, with no coalescing
    detail: |
      runAsk writes once per streaming delta (ask.go:190); each write reaches
      liveScreen.Write -> repaint -> Paint, which emits ESC[H ESC[J plus the whole
      visible frame. Hundreds of deltas per answer means hundreds of full-screen
      erase-and-redraw cycles and megabytes to the tty for one answer. The plan
      declares no operating envelope for this path (ARCH-CONSTRAINTS). Fix: coalesce
      in liveScreen on a trailing timer with a forced flush on Draw/Page/Scroll/Stop,
      and write the budget into the plan.
  - id: new
    severity: Minor
    family: one-owner-per-invariant
    title: |
      the offset clamp is spelled twice and Frame does not write its clamp back
    detail: |
      Frame (screen.go:126) clamps a local `off`; Scroll (screen.go:147) clamps
      s.offset. After a widening resize s.offset can stay out of range, so the first
      wheel-down is a visual no-op. One clampedOffset() owner fixes both (ARCH-DRY).
  - id: new
    severity: Minor
    family: unreachable-guard
    title: |
      the handedBack once-only guard in finish cannot fire today
    detail: |
      Every finish() call site (replraw.go:238, :262, :290) returns immediately after
      it, so finish cannot run twice. Defensible as forward protection, but it
      currently reads as a guarantee nothing exercises.
  - id: new
    severity: Minor
    family: terminal-writes-off-seam
    title: |
      terminal-control sequences go to the stdin handle while frames go to stdout, errors discarded
    detail: |
      rawterm.go:138,149,182,191 write altScreen/mouse sequences to rawSession.f, the
      stdin *os.File, while liveScreen paints to stdout; the fmt.Fprint errors are
      dropped and a failed write still sets r.alt/r.mouse. Write them to the handle
      the frames go to, or check the error.
  - id: new
    severity: Minor
    family: control-sequence-buffered-as-text
    title: |
      RenderLine's trailing cursor-back escape is stored in the buffer and the transcript
    detail: |
      With the cursor mid-line at submit, replraw.go:314 buffers a committed line
      ending in ESC[<n>D. Harmless today because a \r\n follows, but
      TestEditorLoopWritesThroughAScreen only guards against ESC[K.
  - id: new
    severity: Minor
    family: doc-overclaim
    title: |
      README says the transcript is your scrollback "exactly as it was before"
    detail: |
      The erased indicator and the live edge (prompt and menu) are deliberately absent
      from the transcript, so "exactly as it was" overstates what D3 restores.
  - id: new
    severity: Minor
    family: plan-table-incomplete
    title: |
      M1's Core concepts tables omit liveScreen, display, enterMouse/leaveMouse, terminalRows, decodeWheel and the four new KeyKinds
    detail: |
      All are new entities M1 ships and all appear only in Revisions prose, not in the
      greppable tables a reader and the plan-table guards check. Add them with
      kind/location/status in a "## Revisions" entry.
```
