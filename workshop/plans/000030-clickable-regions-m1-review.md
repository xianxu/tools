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

---

## Re-review — 2026-08-29T20:48:44-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 30 — clickable regions in the terminal: click ORIGIN French to hear it, click the IPA to replay |
| repo | tools |
| issue file | workshop/issues/000030-clickable-regions.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 168b1c9f3ed7367f122af4795002ad336fd41e02..72ad3a17e2ad2eb89fe2744d08817c90416008d7 |
| command | sdlc milestone-close --issue 30 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-29T20:48:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

`go test ./...` is **red at HEAD** — two repo guards fail (`TestPlanTablesNameEntitiesThatExist`, `TestPlanTableStatusMatchesTheChangeWindow`), and base `168b1c9` is green on the one that runs there. The issue's `## Log` records "Re-verified: `go test ./...` and `-race` green" for this window, which does not reproduce. That alone blocks the boundary, and the cause matters more than the symptom: the fix for BR-9/BR-22 added a `Kind` column to M1's Pure-entities table, which lands in the guard's Status slot, so all fourteen M1 rows are rejected as out-of-vocabulary *and* go unchecked — the plan-table cross-check that this very gate reads is currently blind to M1's entire table. Everything else is in good shape. I mutation-verified the four blocking fixes this round claims: deleting the X10 dispatch, deleting `leaveAlt`/`leaveMouse` from `restore`, reverting `Paint` to a logical-line budget, and removing the repaint throttle each redden a named test. `-race` is clean and `go vet` is silent. Three residuals remain from prior findings (`terminalWidth`'s `0` sentinel still defeats the display-row budget below 20 columns, `Paint`'s discarded write error plus one stale "render cooked, play raw" comment, and the BR-20 cursor fix which no test pins), and the rework commit added architectural surface — the display-row budget, paint-time clipping, the 16 ms throttle, the X10 fallback — without touching `atlas/` for any of it.

## 1. Strengths

- **`cmd/define/key.go:245` `decodeX10Mouse` is the right shape for the Critical it answers.** Six bytes or none, with `Key{}, 0` reusing `decodeKey`'s existing partial-sequence protocol rather than inventing a second one. `TestX10ClickTypesNothing` (`key_test.go:267`) asserts the *observable* — nothing reaches the line — instead of a byte count, and reverting the dispatch reddens it with the exact `" !!"` the operator would have seen.
- **`cmd/define/rawterm.go:32` `rawSession.control` is a structural fix, not a test patch.** Typing the mode-sequence target as `io.Writer` makes the restore protocol assertable in process; `TestRestoreHandsBackEveryTerminalState` (`rawterm_test.go:114`) pins the bytes *and* the order, and `TestRestoreSendsNothingItDidNotTake` / `TestEnterDoesNotClaimAStateItCouldNotWrite` close the two adjacent holes. Deleting both leaves from `restore()` now reddens three assertions.
- **`screen.go:211` `Paint` charges the live edge its real height and clips buffer lines at paint time**, keeping the full text in the buffer — so the transcript and M2's future click map are unaffected by the view's truncation. `TestScreenClipsTheViewNotTheBuffer` pins both halves of that split.
- **`liveScreen`'s throttle has the trailing flush** (`screen.go:340`), and the test names why it matters: `TestLiveScreenThrottlesTheRepaintButNeverLosesTheLastWord` fails on *both* mutations — remove the throttle and the frame count blows up; the indicator case would hang without the trailing timer.
- **The `crlfWriter` sweep replaced a dead test with a live one.** `TestHighlightingSeesLogicalTextAndTheScreenPlacesIt` (`askhighlight_test.go:255`) drives `runAsk` into a real `screen`, which is the composition production actually builds — a genuine improvement over the composition it used to assert.

## 2. Critical findings

**`go test ./...` fails at HEAD; the recorded verification does not reproduce** — `workshop/plans/000030-clickable-regions-plan.md:78`, `:93`, `:167`

Two failures, both introduced by this window (base `168b1c9` passes `TestPlanTablesNameEntitiesThatExist`; the window guard skips there because HEAD == merge-base):

1. `plan.md:78` — the Pure-entities table is now `| Name | Lives in | Kind | Status |`. `repo_guard_test.go`'s row regex reads the **third** cell as Status, so it sees `"PURE"` and fails closed on all fourteen rows, including `crlfWriter`'s `"—"` (`plan.md:93`). The rows are then `continue`d past, so *no M1 name is checked against the tree at all*. The Integration table at `plan.md:104` (`| Name | Lives in | Status | Wraps |`) keeps Status third and passes — that is the shape the guard expects.
2. `plan.md:167` — `| Render | cmd/define/render.go | modified — also returns regions |` is an **M2** row. This window touched `render.go` (`visibleLen`), so the guard now evaluates it and reports "the row describes work that did not happen."

Fix sketch: for (1), either move `Kind` after `Status` or fold it into the Status prose so the third cell stays in the controlled vocabulary — and re-run to confirm the fourteen rows now actually get *checked*, not merely stop erroring. For (2), the M2 row should read `new`/`unchanged` until M2 lands, or the guard needs a milestone-scoped exemption (`workshop/issues/000033-plan-table-both-directions.md` is the natural home for that design). Then re-run `go test ./...` and `-race` and re-record the Log line. I manually cross-checked every M1 Core-concepts row against the tree while the guard was down — all entities exist at their stated paths — but that is a one-off, not the mechanism.

## 3. Important findings

**`replRaw`'s exit sequence has no pin that runs in the default build, and the BR-20 fix has none at all** — `cmd/define/replraw.go:56`, `:304`

> **This is the 3rd finding in family `unfalsifiable-test-pin`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

The rule: **every behavioural claim M1 makes must be pinned by a test that runs under plain `go test ./cmd/define/`; a `conformance`-tagged pty row is a live conformance check, not the pin.** BR-4/BR-13 established exactly this for `restore`, and the same gap remains one level up. `replRaw` has zero in-process callers (`grep replRaw(` → `repl.go:233` and the definition), so everything it uniquely owns — `enterAlt`+`enterMouse` on entry, and `finish`'s ordering `Stop() → restore() → Fprint(transcript)` plus its once-only property — is asserted only by `TestPTYTranscriptIsPrintedOnExit` and `TestPTYMouseTrackingIsAskedForAndGivenBack`, both of which skip when no pty is available (`pty_conformance_test.go:59: no pty available: operation not permitted` in this environment). M1 done-when rows 3b and 6 therefore have no runnable pin here. Separately, `replraw.go:304`'s `submitted.Cursor = len(submitted.Line)` — this round's fix for BR-20 — is unpinned: I deleted it in a scratch copy and the whole suite stayed green. Fix sketch: extract `finish`'s body into a small function over `(display-with-Transcript, restorer, io.Writer)` and assert the three-step order and once-only in process; add a `[D` assertion to `TestEditorLoopWritesThroughAScreen` driven by a submit with the cursor mid-line.

**`atlas/` was not updated for the surface the rework commit introduced** — `atlas/define.md:265-340`

`git show 72ad3a1 -- atlas/define.md` touches only the two `crlfWriter` prose sites. Three genuinely new architectural facts from the same commit are absent from "The screen":

- **The frame is budgeted in display rows and buffer lines are clipped to the terminal width at paint time.** This is user-visible (an over-wide line is *cut*, not wrapped, in the view while the transcript keeps it) and it is the invariant M2's `RegionAt` rests on. The section documents resize's row/column reasoning but never says the view clips.
- **The 16 ms repaint throttle with its trailing flush.** The atlas currently states "**A write REPAINTS**" without qualification, which is now incomplete — a write repaints at most once per `paintInterval`. This is the declared ARCH-CONSTRAINTS envelope and it lives only in a plan Revisions bullet.
- **The X10 fallback and the rule it leaves** ("for every mode we enable, the decoder answers every encoding that mode can reply in"). The atlas names `1000` + `1006` and stops there.

## 4. Minor findings

- **One display-cell measurement, not three.** > **This is the 4th finding in family `frame-fits-the-terminal`.** Do NOT fix these instances — state the rule and fix that. The rule: *every row and column count in the paint path comes from one owner that measures display cells and cannot return a sentinel.* Three counters currently disagree. (a) `visibleLen` (`render.go:230`) counts **runes**: measured, `"日本語のテキストです"` is 20 columns and reports 10 — a frame budgeted from that is too tall, which is the BR-6/BR-12 failure exactly; `"bänˈZHo͝or"` is 9 columns and reports 10, so `clipVisible` cuts a line that fits. No wide characters exist in the `en`/`es`/`it` fixture corpus, so today's vector is the free-form `?` ask stream and typed input, not the dictionary. (b) `Paint`'s `\x1b[%dA` (`screen.go:235`) counts **logical** menu rows: measured with a 45-column menu row in a 20-column terminal, the menu occupied 3 display rows and the cursor moved up 1, reprinting the prompt over the menu — the same off-by-a-row limit the whole-frame redraw was supposed to delete, and its comment claims it "never counts rows it drew earlier." Latent only because `menuLines` truncates to `opt.width`, which happens to equal `termCols` today.
- **The wheel bit-decode is spelled twice** — `key.go:201-207` and `key.go:250-254`. > **This is the 2nd finding in family `one-owner-per-invariant`.** The rule: *one function owns "what does this mouse button byte mean", and every encoding calls it.* `b&64` / `b&3` is the same fact in two places, and M2.2 adds button decoding to both. Extract `wheelFromButton(b int) (Key, bool)` before M2 makes it three.
- **One unsynchronised test observation remains** — `screen_test.go:457` reads `tty.frames` directly while every other site uses the locked `tty.painted()`; `:464` reads `l.pending` outside `l.mu`. > **This is the 2nd finding in family `unsynchronised-test-observation`.** The rule: *a field written by a timer or loop goroutine is read only through its accessor.* `-race` does not report it (the timer reliably fires after the read), which is precisely why the rule has to be structural rather than detected.

## 5. Test coverage notes

- Mutation-verified this round: X10 dispatch, `restore`'s two leaves, the display-row budget + clip, and the repaint throttle all redden named tests. Those four fixes are real.
- Mutation-verified as **unpinned**: `submitted.Cursor = len(submitted.Line)` (raised above) and `Frame`'s clamp write-back — reverting `screen.go:132` to a local clamp leaves `TestScreen*` green. The DRY half of BR-17 is structurally verifiable by reading; the behavioural half ("after a widening resize the first wheel-down is a visual no-op") has no assertion.
- BR-1's chunk-boundary property for `Write` — any split of the same byte stream yields the same lines — still does not exist. `TestScreenWriteBuildsLines` has a token-stream row and a CRLF-split row; a property test over arbitrary split points would subsume both and is cheap next to `FuzzScreenWriteDoesNotPanic`, which only asserts no panic.
- `pty_conformance_test.go` rows all SKIP in this environment (`operation not permitted`), so I could not verify the 11 green PTY rows the Log records; I am reporting that as unverified, not as failing.

## 6. Architectural notes for upcoming work

- **ARCH-PURE — pass.** `screen` is a pure model, `liveScreen` is the only IO, `display` (`replraw.go:100`) is the injected seam, and `screen_test.go` runs the whole arithmetic with a `strings.Builder`. `Paint` taking its writer is the right call.
- **ARCH-MOCK — pass with a caveat.** The terminal's double is the `creack/pty` harness plus the new in-process `io.Writer` seams; production and test share `rawSession.control` and `display`. The caveat is the coverage gap above: `replRaw` sits outside both seams.
- **ARCH-DRY — flagged** (wheel bit-decode, above). Otherwise good: one `clamp()` owner, one `visibleLen`, one `playAnnounced`.
- **ARCH-PURPOSE — flagged.** The round's headline was "prose is swept by CLASS or not at all", and the enumeration still stopped at `cmd/define` + `atlas`. Two siblings of the same class survive: `workshop/issues/000032-crlf-seam.md:36,49` still describes `replraw.go` wrapping stderr in `crlfWriter` and proposes extending it (D5a deliberately deferred `#32`, so this is a documented deferral rather than a miss), and `pty_conformance_test.go:216-217` still states "the fix is to render cooked and play raw" in the present tense for a design D4 deleted. When writing the enumeration, include other issues' Spec sections and the conformance tests.
- **ARCH-CONSTRAINTS — flagged.** The envelope is declared (16 ms + trailing flush, uncapped buffer with a stated reason) and enforced, which answers BR-16 and BR-8. What is not declared is the *column* envelope: `terminalWidth` returning `0` below 20 columns silently drops the frame out of its budget, and no test covers a terminal narrower than 20.
- **For M2:** `RegionAt(row, col)` needs the same column arithmetic as the clip, so fixing the display-cell measurement first is cheaper than fixing it twice. Also note that `screen.rows`/`cols` are set as a side effect of `Paint`, so a hit test that runs before the first paint sees zeroes — worth an explicit invariant when the region map lands.

## 7. Plan revision recommendations

- **`## Revisions` — "the Core-concepts table's Kind column broke the guard that reads it."** Record why the table shape matters (the third cell is the controlled-vocabulary Status slot), what the fix was, and that the fourteen M1 rows were unchecked for the length of the boundary review. This is the second time in this issue that a fix for a plan-table finding produced a new plan-table defect.
- **`## Revisions` — the M2 `Render` row's status.** State how a not-yet-started milestone's `modified` rows are meant to read while the window is open, so the window guard and the plan agree.
- **`## Revisions` — the operating envelope gains its column half.** The envelope bullet declares a repaint budget and a buffer policy but no minimum measurable width; `terminalWidth`'s `0` sentinel needs a stated behaviour ("below 20 columns the screen does X"), since the current answer is "silently revert to the pre-rework budget."
- **`M1.6` — widen the docs row.** It currently reads as the atlas's raw-mode rewrite. The rework added surface after M1.6 was ticked, and the row should say that a docs sweep follows any post-review code change, not just the one it was written for.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      M1.1's prose is unchanged and no chunk-boundary property test for Write was written; Minor, non-blocking.
  - id: BR-2
    disposition: addressed
    note: |
      The plan now states the uncapped buffer deliberately, with the reason and a size estimate.
  - id: BR-3
    disposition: withdrawn
    note: |
      Overtaken: M1.3 is built and its diff is in front of the reviewer, so the size warning is moot.
  - id: BR-4
    disposition: addressed
    note: |
      Verified by mutation — deleting both leaves from restore() now reddens three assertions.
  - id: BR-5
    disposition: addressed
    note: |
      Verified by mutation — removing the decodeX10Mouse dispatch reddens TestDecodeX10Mouse and TestX10ClickTypesNothing.
  - id: BR-6
    disposition: addressed
    note: |
      Both routes it named are fixed and mutation-verified; the cols==0 residual is carried on BR-12.
  - id: BR-7
    disposition: addressed
    note: |
      Both eraseLine prefixes are gone and the comments now explain why the screen took that job.
  - id: BR-8
    disposition: addressed
    note: |
      Throttle plus trailing flush, mutation-verified; the uncapped buffer is now a stated decision.
  - id: BR-9
    disposition: addressed
    note: |
      Rows added — but the added Kind column broke the guard that reads the table; see the new Critical.
  - id: BR-10
    disposition: not-addressed
    note: |
      Two of five remain: Paint still discards the tty write error, and pty_conformance_test.go:216 still states "render cooked, play raw" as the current fix.
  - id: BR-11
    disposition: addressed
    note: |
      Mutation-verified; the X10 payload is consumed whole or not at all.
  - id: BR-12
    disposition: not-addressed
    note: |
      The third route it named survives — terminalWidth returns 0 below 20 columns or on probe failure, and with cols==0 displayRows charges one row per line and clipVisible returns the line unclipped; measured, a 10-row/80-column frame then needs 28 display rows.
  - id: BR-13
    disposition: addressed
    note: |
      rawSession.control is an io.Writer and the restore protocol is asserted in process, bytes and order.
  - id: BR-14
    disposition: addressed
    note: |
      All five sites fixed and the test now pins the production composition; issue 32's Spec still describes the old nesting, which D5a deliberately deferred.
  - id: BR-15
    disposition: addressed
    note: |
      go test -race ./cmd/define/ is green; one remaining unsynchronised read is raised separately as the family rule.
  - id: BR-16
    disposition: addressed
    note: |
      Mutation-verified — removing the throttle reddens the frame-count assertion.
  - id: BR-17
    disposition: addressed
    note: |
      One clamp owner, and Frame writes back; the behavioural half has no pin (reverting to a local clamp leaves TestScreen green).
  - id: BR-18
    disposition: addressed
    note: |
      sync.OnceFunc with the transcript stated as its reason; still unreachable today, which the comment now owns.
  - id: BR-19
    disposition: addressed
    note: |
      Mode sequences go to the same stream as the frames, and a failed write no longer claims the state.
  - id: BR-20
    disposition: not-addressed
    note: |
      The fix is present at replraw.go:304 but no test pins it — deleting the line leaves the whole suite green.
  - id: BR-21
    disposition: addressed
    note: |
      README now says what the transcript keeps and what it deliberately drops.
  - id: BR-22
    disposition: addressed
    note: |
      Rows added; the resulting guard breakage is the new Critical.
findings:
  - id: new
    severity: Critical
    family: verification-claim-unreproduced
    title: |
      go test ./... is red at HEAD on two plan-table guards, and the Log records it green
    detail: |
      TestPlanTablesNameEntitiesThatExist fails on all fourteen M1 Pure-entities rows:
      plan.md:78 added a Kind column, which lands in the guard's third-cell Status slot,
      so every row is rejected as an out-of-vocabulary status AND skipped unchecked — the
      Core-concepts cross-check is blind to M1's whole table. TestPlanTableStatusMatchesTheChangeWindow
      fails on plan.md:167, the M2 Render row, because this window touched render.go
      without touching Render's declaration. Base 168b1c9 passes the first guard and
      skips the second. Fix the table shape (Status third, as the Integration table at
      plan.md:104 already is), decide how a not-yet-started milestone's modified rows
      should read, then re-run and re-record.
  - id: new
    severity: Important
    family: unfalsifiable-test-pin
    title: |
      replRaw's exit sequence is pinned only by pty rows that skip, and the BR-20 fix is pinned by nothing
    detail: |
      Third in this family. The rule: every behavioural claim M1 makes needs a pin that
      runs under plain go test ./cmd/define/; a conformance-tagged pty row is a live
      conformance check, not the pin. replRaw has no in-process caller, so enterAlt +
      enterMouse on entry and finish's Stop -> restore -> print-transcript ordering and
      once-only property rest solely on TestPTYTranscriptIsPrintedOnExit and
      TestPTYMouseTrackingIsAskedForAndGivenBack, both of which skipped here ("no pty
      available: operation not permitted"). Separately replraw.go:304's
      submitted.Cursor = len(submitted.Line) can be deleted with the suite still green.
      Extract finish's body over an interface and assert the order in process.
  - id: new
    severity: Important
    family: docs-lag-new-surface
    title: |
      atlas/ was not updated for the display-row budget, the paint clip, the repaint throttle or the X10 fallback
    detail: |
      The rework commit touched atlas/define.md only for the two crlfWriter prose sites.
      "The screen" (atlas/define.md:265-340) does not say that the frame is budgeted in
      display rows, that buffer lines are CLIPPED to the terminal width at paint time
      while the transcript keeps the full text, or that a write now repaints at most
      once per 16 ms — it still asserts "A write REPAINTS" flatly. The X10 fallback and
      the rule it leaves behind are also absent, though 1000 and 1006 are named. All
      three are surface a reader of the atlas would be wrong about.
  - id: new
    severity: Minor
    family: frame-fits-the-terminal
    title: |
      Three counters answer "how wide is this" differently: runes, a sentinel column count, and logical menu rows
    detail: |
      Fourth in this family. Do not fix these instances — the rule is that every row and
      column count in the paint path comes from one owner measuring display cells that
      cannot return a sentinel. Measured: visibleLen (render.go:230) counts runes, so
      "日本語のテキストです" reports 10 for 20 columns (frame too tall, the BR-6/BR-12
      failure) and "bänˈZHo͝or" reports 10 for 9 (clipVisible cuts text that fits);
      Paint's cursor-up (screen.go:235) uses len(menu) while the terminal moved
      sum(displayRows(menu)) rows, so a 45-column menu row in a 20-column terminal
      leaves the cursor two rows low and the prompt is reprinted over the menu — the
      same off-by-a-row limit the whole-frame redraw claims in its own comment to have
      deleted. Latent today only because menuLines truncates to opt.width, which
      happens to equal termCols.
  - id: new
    severity: Minor
    family: one-owner-per-invariant
    title: |
      The wheel button-byte decode is spelled twice, in decodeWheel and decodeX10Mouse
    detail: |
      Second in this family. The rule: one function owns "what does this mouse button
      byte mean" and every encoding calls it. key.go:201-207 and key.go:250-254 both
      spell b&64 for the wheel bit and b&3 for the direction. M2.2 adds button decoding,
      which would make it three spellings of one fact across two encodings. Extract
      wheelFromButton(b int) (Key, bool) now.
  - id: new
    severity: Minor
    family: unsynchronised-test-observation
    title: |
      screen_test.go:457 reads tty.frames without the lock, and :464 reads l.pending without l.mu
    detail: |
      Second in this family. The rule: a field written by a timer or loop goroutine is
      read only through its accessor. Every other site in the same test uses the locked
      tty.painted(); line 457 reaches the field directly while the trailing paint timer
      may be running. -race does not report it because the timer reliably fires after
      the read, which is exactly why the guarantee has to be structural.
```

---

## Re-review — 2026-08-29T21:18:49-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 30 — clickable regions in the terminal: click ORIGIN French to hear it, click the IPA to replay |
| repo | tools |
| issue file | workshop/issues/000030-clickable-regions.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 168b1c9f3ed7367f122af4795002ad336fd41e02..26102ae3964670f141b8394e3ffb46afe113edf8 |
| command | sdlc milestone-close --issue 30 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-29T21:18:49-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 is substantively built and most of round 3's claimed fixes hold up under reversion: `handBack`/`onceHandBack` are genuinely pinned in process, `TestACommittedLineCarriesNoCursorEscape` goes red when I delete `submitted.Cursor = len(submitted.Line)`, `TestRestoreHandsBackEveryTerminalState` asserts real bytes through a real seam, `wheelFromButton` is one owner, and the atlas/README now describe the screen. Two things block the boundary. **`go test ./...` is RED at HEAD** — `TestPlanTableStatusMatchesTheChangeWindow` fails because the plan's M2 `Render` row now says `unchanged` while this window edits `Render`'s body at `render.go:172`; that is BR-23's exact family, second consecutive round, and the round-3 Log records the rule ("re-run the suite after editing a plan") without the suite having been re-run. And **the frame still does not fit the terminal**: `clipVisible`'s cut cursor counts *runes* (`screen.go:465`) while everything around it counts *cells*, so the one-owner rule BR-26 asked for is implemented at 3 of 6 sites and a 100-rune CJK line still paints 19 display rows into a 10-row terminal — the invariant M1 exists to establish.

## 1. Strengths

- **`handBack`/`onceHandBack` (`replraw.go:101-140`) is the right answer to BR-24.** Extracting the exit sequence over two tiny interfaces makes the ordering (stop painting → restore → print transcript) assertable with no pty, and `TestHandBackStopsPaintingRestoresThenPrintsTheSession` checks all three steps independently. The pty rows all skipped in this environment, so without this the whole exit path would again be unverified at the gate.
- **`rawSession.control io.Writer` (`rawterm.go:32`) is a structural fix, not a patched assertion.** `TestRestoreSendsNothingItDidNotTake` and `TestEnterDoesNotClaimAStateItCouldNotWrite` pin the two failure modes the previous vacuous test could not reach, and the enter-side error handling ("the flag records the terminal's state, not the attempt") is exactly right.
- **The X10 fallback is complete and adversarially tested.** `decodeX10Mouse` consumes six-or-none, `TestX10ClickTypesNothing` asserts the *observable* (nothing typed) rather than a byte count, and `FuzzDecodeWheelIsBounded` checks the wheel answer can only come from a sequence that actually ended in `M`/`m`.
- **Verified falsifiable by reversion:** replacing `submitted.Cursor = len(submitted.Line)` with a no-op reddens `TestACommittedLineCarriesNoCursorEscape` with `"…\x1b[3D"`. BR-20's fix is real.
- **`recordDisplay` and `countingWriter` (`editorloop_test.go:54`, `screen_test.go:475`) apply BR-28's rule structurally** — every field behind `mu`, every read through an accessor, including the one line the finding named.

## 2. Critical findings

**`go test ./...` is red at HEAD, on the same guard family as last round** — see `dispose: BR-23 not-addressed` below. `workshop/plans/000030-clickable-regions-plan.md:167` claims `Render` is `unchanged`; `render.go:172` (inside `Render`, lines 97-222) changed `visibleLen(marker)` → `visibleCells(marker)` in this window. Fix: set the status to `modified` and say why (the `visibleLen`→`visibleCells` rename swept its body; the *signature* change is still M2.1's), then re-run the suite **after** the edit.

## 3. Important findings

**The display-row budget is still breakable** — see `dispose: BR-12 not-addressed`. Measured at HEAD: `clipVisible(strings.Repeat("日",100), 80)` returns 200 cells, and a 10-line buffer of those painted into a 10×80 terminal needs **19 display rows**. Reachable via streamed model answers containing emoji or CJK (`isWide` covers `0x1f300-0x1f64f`), via a typed line, and via any narrowing resize.

**The Core-concepts tables omit this round's own deliverables** — `handBack`, `onceHandBack` (`replraw.go`) and `wheelFromButton` (`key.go`) appear only in the plan's Revisions prose. See finding 1 below; the deliverable is the inverse guard, not three rows.

## 4. Minor findings

- `screen.go:243` — `\x1b[%dA` walks back only the menu's rows, so a prompt that wraps is reprinted from its *last* row and overwrites the first menu row (measured at 20 cols with a 32-cell prompt: `…m1\r\nm2\x1b[2A\r> zzz…`).
- `screen_test.go:465-470` — the `l.Stop()` flush assertion depends on the write landing inside the 16 ms window after `waitFor` returns; a scheduling hiccup makes it a false failure.
- `command.go:300` — `truncate`'s doc comment still asserts "a width is a column count, and a column is a rune", which round 3 made false.
- `screen.go:249` / `screen.go:402-408` — `Paint`'s `fmt.Fprint(w, …)` error is discarded (BR-10 residue; defensible, but it is the one item of that finding left standing).
- The `#### Pure entities` table carries a row whose Kind is `INTEGRATION` (`terminalSize`/`terminalCols`), and `crlfWriter`'s row has no Kind at all.

## 5. Test coverage notes

`TestScreenFrameFitsTheTerminalInDisplayRows` is well designed — it re-derives the row count with `displayRows` over what `clipVisible` actually emitted, so it *would* catch the cell/rune split. Every one of its four fixtures is ASCII (`x`, `z`, `m`, `c`). One row with `strings.Repeat("日", 100)` and one with a combining-mark string reddens it immediately; that single fixture gap is what let round 3's fix ship half-done. `FuzzScreenWriteDoesNotPanic` covers crashes but not BR-1's chunk-boundary property (any split of the same byte stream yields the same lines), which is the one property worth more than the four enumerated cases.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag.** Three column counters, one fact. See the enumeration in BR-26's disposition.
- **ARCH-PURE — pass.** `screen` is pure and unit-tested against `strings.Builder`; `liveScreen` is the only IO; `display` keeps `runEditor` drivable with no terminal. Clean.
- **ARCH-PURPOSE — flag.** Round 3's stated motivation for `cellWidth` was `bänˈZHo͝or` and Japanese entries; the fix landed at `visibleCells`/`wrapText`/`displayRows` and stopped at the clip. That is the instance, not the class — and it is the same shape ARCH-PURPOSE names.
- **ARCH-MOCK — pass, with a note.** The pty harness is the live conformance check and `conformance.SkipOrFail` + `CONFORMANCE_STRICT` correctly stop a skip reading as green; every pty row skipped here ("no pty available"), which is why BR-24's in-process extraction mattered. For M2: there is still no in-process fake that *models terminal wrapping*. Every escape from this family so far (BR-6, BR-12, and today's) is "our arithmetic disagreed with what the terminal would do". A ~40-line fake terminal that consumes a frame and reports where the cursor ended up would make `RegionAt`'s hit test checkable end-to-end without a pty.
- **ARCH-CONSTRAINTS — pass.** The 16 ms throttle with a trailing flush is correctly reasoned (the indicator case is the load-bearing half), the uncapped buffer is a stated decision with its bound, and `watchResize` coalesces with a genuinely non-blocking two-step hand-off.

## 7. Plan revision recommendations

- **`## Revisions` — "M2's `Render` row is a claim about the diff."** Record that `Render` is `modified` in this window (the `visibleCells` rename), that the guard reads the third cell, and that the suite runs **after** a plan edit. This is the second round the same guard has been red at HEAD.
- **`## Revisions` — "one owner, and the enumeration."** State the rule as an enumeration rather than a principle: every cursor that walks a string against a column budget advances by `cellWidth(r)`. List the six sites and their status. A principle stated without its enumeration is what let the clip survive round 3.
- **Core-concepts tables** — add `handBack`, `onceHandBack`, `wheelFromButton`, and note the guard that would make the omission impossible.

```findings
dispose:
  - id: BR-23
    disposition: not-addressed
    note: |
      go test ./... is still RED at HEAD: TestPlanTableStatusMatchesTheChangeWindow now fails on the M2 Render row for the OPPOSITE reason — it was flipped to "unchanged" while render.go:172 (inside Render, 97-222) changed visibleLen to visibleCells in this window. The Kind-column half is fixed and TestPlanTablesNameEntitiesThatExist passes; nothing else in ./... fails.
  - id: BR-12
    disposition: not-addressed
    note: |
      The budget half landed (cols is real, displayRows charges wrapped height, the clip runs at paint time) but the frame still overflows: clipVisible's cut cursor counts RUNES, so clipVisible(strings.Repeat("日",100), 80) returns 200 cells and a 10-line buffer of those needs 19 display rows in a 10-row terminal. Measured at HEAD. TestScreenFrameFitsTheTerminalInDisplayRows would catch it — all four of its fixtures are ASCII.
  - id: BR-26
    disposition: not-addressed
    note: |
      Third in family one-owner-per-invariant, and the RULE is still the deliverable rather than the sites. Rule: every cursor that walks a string against a column budget advances by cellWidth(r) — no site in the paint path may increment a column counter per rune, per byte, or per index. Enumeration measured at HEAD, 3 of 6 wrong: render.go:256 visibleCells CELLS ok; render.go:332,335 wrapText CELLS ok; screen.go:412 displayRows CELLS ok; screen.go:465 clipVisible RUNES wrong (cuts a 20-cell/30-rune line to 8 cells at width 12, and lets a 200-cell line through a width-80 clip); editor.go:227 RenderLine's ESC[nD park counts runes for a move the terminal makes in columns; command.go:305 truncate counts runes and its doc comment still asserts "a column is a rune". Write the enumeration into the plan and sweep it, and add one wide-rune and one combining-mark row to TestScreenFrameFitsTheTerminalInDisplayRows so the class cannot come back.
  - id: BR-20
    disposition: addressed
    note: |
      Verified by reversion: replacing submitted.Cursor = len(submitted.Line) with a no-op reddens TestACommittedLineCarriesNoCursorEscape with "…\x1b[3D".
  - id: BR-24
    disposition: addressed
    note: |
      handBack/onceHandBack are named, take two small interfaces, and are pinned in process by two tests that ran here while every pty row skipped. Residue: replRaw's own enterAlt/enterMouse on ENTRY are still pinned only by pty rows.
  - id: BR-25
    disposition: addressed
    note: |
      atlas/define.md gains "The screen" with the display-row budget, the clip, the cell-width owner, the 16 ms throttle and its trailing flush, the X10 fallback and handBack. One sentence now overclaims — "every clip reads that one function" is false while clipVisible cuts by rune.
  - id: BR-27
    disposition: addressed
    note: |
      wheelFromButton is the single owner; decodeWheel and decodeX10Mouse both call it.
  - id: BR-28
    disposition: addressed
    note: |
      countingWriter.painted() is used at every read site, recordDisplay is mutex-guarded with reader methods, and the watcher reports measurements over a channel. go test -race is green on the M1 suites.
  - id: BR-10
    disposition: addressed
    note: |
      Mode sequences go to rawSession.control rather than the stdin handle; the nested selects are a drain-then-send pair with the single-producer reasoning stated; the missing signal.Stop is now a recorded consequence of the seam; the stale cooked-mode prose is gone. Still open, and folded into the Minor list: Paint discards the tty write error.
  - id: BR-1
    disposition: not-addressed
    note: |
      M1.1 still enumerates the four cases in prose, and the chunk-boundary property the finding actually asked for (any split of the same byte stream yields the same lines) has no test — FuzzScreenWriteDoesNotPanic only checks for panics.
findings:
  - id: new
    severity: Important
    family: plan-table-incomplete
    title: |
      handBack, onceHandBack and wheelFromButton are absent from M1's Core-concepts tables
    detail: |
      This is the 2nd finding in family plan-table-incomplete (BR-9 and BR-22 are its siblings under plan-table-under-declares — three rounds, same class). Do NOT just add the three rows. The rule: the Core-concepts table is populated FROM THE DIFF, not from memory — every top-level declaration this window adds to a file the tables name gets a row. TestPlanTablesNameEntitiesThatExist already checks table-to-tree; the direction that keeps failing is tree-to-table, and repo_guard_test.go already has changedLines() and repoRoot() to build the inverse guard with. Measured at HEAD, the omissions are handBack, onceHandBack (replraw.go), wheelFromButton (key.go), plus screen.Lines, screen.eraseOpenLine, paintInterval, defaultRows/defaultCols — and the first three are exactly the deliverables round 3's own Log names as its fixes for BR-24 and BR-27.
  - id: new
    severity: Minor
    family: app-owns-every-row
    title: |
      Paint's cursor-up omits the prompt's own display height, so a wrapping prompt is reprinted over the menu
    detail: |
      This is the 2nd finding in family app-owns-every-row. Do NOT fix the instance. The rule: the frame's row accounting is ONE pass — the same enumeration that budgets rows derives where the cursor must return to. Today screen.go:216-222 and screen.go:239-243 are two separate summations of the same quantity, and the second omits the prompt. Measured: Paint(&b, 10, 20, "> "+strings.Repeat("z",30), []string{"m1","m2"}) emits ESC[2A, which lands on the prompt's SECOND row; the reprint then covers menu row m1 and leaves the cursor a row low — the same off-by-a-row limit the whole-frame redraw claims in its own comment to have deleted. Reachable on a terminal narrow enough that a partial /command wraps; terminalCols has no 20-column floor.
  - id: new
    severity: Minor
    family: timing-dependent-assertion
    title: |
      The throttle test's Stop-flush assertion depends on wall-clock ordering it does not control
    detail: |
      screen_test.go:465-470 writes "the last word", snapshots the frame count, then requires l.Stop() to paint. That only holds while the write lands inside paintInterval of the trailing flush waitFor just observed; if the goroutine is descheduled past 16 ms the write paints itself, pending is false, and Stop correctly does nothing while the test reports "Stop left a pending frame unpainted". Set l.painted deliberately (as TestLiveScreenShowsWhatIsWrittenToIt already does) rather than racing the interval.
```

---

## Re-review — 2026-08-29T21:50:28-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 30 — clickable regions in the terminal: click ORIGIN French to hear it, click the IPA to replay |
| repo | tools |
| issue file | workshop/issues/000030-clickable-regions.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 168b1c9f3ed7367f122af4795002ad336fd41e02..b88accf0fde0cccbe1a1ed7c344c61e0283c1128 |
| command | sdlc milestone-close --issue 30 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-29T21:50:28-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The milestone's substance is in good shape: `go test ./...` is green at HEAD (verified, 102s) and `go test -race ./cmd/define/` is green (109s), so the round-3 Critical (BR-23) is genuinely cleared and the lesson behind it is recorded. The pure/IO split (`screen` vs `liveScreen`), the `rawSession.control` seam, `handBack` over two tiny interfaces, and `wheelFromButton` as the one button decoder are all the right shapes, and every M1 done-when row names a test that exists. What blocks SHIP is that M1's *central* invariant — the frame fits the terminal, which is the property the alternate screen was taken for and which M2's `RegionAt` rests on — is still violable through a route no fix has covered: the live edge (prompt + menu) is charged rows but never *budgeted* by them, so a 15×12 terminal with `/` typed emits a 17-row frame into a 12-row window (measured at HEAD). That is the third round this same invariant has been broken by a different route. Alongside it, `Paint`'s cursor geometry has no in-process assertion at all — deleting the whole cursor-up-and-reprint block leaves `go test ./cmd/define/` green — and two prior findings (BR-26, BR-29) were answered at the instance they named rather than at the class they explicitly asked for.

## 1. Strengths

- **`screen` / `liveScreen` is a textbook ARCH-PURE split.** `screen.go:27-255` is a pure line buffer + viewport with `Paint` taking an `io.Writer`; `liveScreen` (`screen.go:271`) is the only thing that holds a tty. Every scroll, clamp, budget and clip case in `screen_test.go` runs with a `strings.Builder` and no pty — which is exactly why the review could measure the failures above without a terminal.
- **The BR-13/BR-4 fix is structural, not a patched assertion.** Moving mode sequences to `rawSession.control io.Writer` (`rawterm.go:31`) makes the restore protocol assertable in-process, bytes *and* order (`rawterm_test.go:114`), and `TestEnterDoesNotClaimAStateItCouldNotWrite` closes the "flag records the attempt, not the terminal" hole. This is the right answer to "the pin could not fail."
- **`handBack` / `onceHandBack` (`replraw.go:110-140`) is the correct extraction.** `replRaw` has no in-process caller, so the exit ordering used to rest entirely on pty rows — and every pty row skipped in this review environment (`no pty available: operation not permitted`). The two in-process tests in `replraw_test.go` ran here.
- **One width owner, and it is measured in cells.** `visibleCells` + `cellWidth` (`render.go:228-311`) with `TestVisibleCellsCountsColumns` covering the combining breve and CJK; `clipVisible` now cuts by cells and never splits a two-cell rune — verified: `clipVisible(strings.Repeat("日",100), 80)` returns 80 cells / 40 runes, and a 10-line CJK buffer needs exactly 10 rows in a 10-row terminal.
- **The decoder answers both mouse encodings whole.** `TestX10ClickTypesNothing` (`key_test.go`) asserts the observable the operator would meet — characters in the word — rather than a byte count, and `FuzzDecodeWheelIsBounded` pins the consumption bound `#14` shipped wrong once.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `cmd/define/screen.go:221-229`: the live edge is charged rows but never budgeted by them, so the frame still overflows the terminal.**
*(3rd finding in family `frame-fits-the-terminal`.)* Earlier rounds fixed instances: BR-6 taught `Paint` about display rows, BR-12 made `cols` real, BR-26 made the clip count cells. Do **not** fix this instance either. The rule that covers all of them: **the frame's total display height is asserted against `termRows` before it is written — every component, buffer *and* live edge, or it is not a budget.** Today only the buffer's share is clamped (`s.rows = termRows - promptRows - menuRows`, floored at 0); `prompt` and `menu` are written unclipped and unlimited. Measured at HEAD:

| terminal | `opt.width` | menu rows | frame needs |
|---|---|---|---|
| 12×15 | 0 (the "do not wrap" policy answer below 20 cols) | 5 logical → 15 display | **17 rows in a 12-row budget** |
| 5×80 | 80 | 5 | **6 rows in a 5-row budget** |

Reachable with one keystroke (`/` enters command mode; `menuLines` returns all five commands) in a narrow tmux pane, and the narrow case is *caused* by the correct decision that `opt.width` goes to 0 below 20 columns — `truncate` then leaves 36-column menu rows to wrap three ways. The consequence is the one M1 exists to prevent: the terminal scrolls, every row the app believes it placed moves, and `M2`'s `RegionAt(row, col)` maps a click to the wrong buffer line. Fix at the rule level: have `Paint` compute the whole frame's height and clip the live edge (drop menu rows from the bottom, clip the prompt) until it fits, and assert `total ≤ termRows` as a property in `TestScreenFrameFitsTheTerminalInDisplayRows` rather than per-fixture.

**I-2 — `cmd/define/screen.go:239-249`: nothing asserts where `Paint` leaves the cursor; the whole block is deletable with a green suite.**
*(4th finding in family `unfalsifiable-test-pin`.)* Earlier rounds fixed instances (BR-13's restore pin, BR-7's erase row, BR-24's pty-only exit sequence). Do **not** just add a test for the cursor-up count. The rule: **every byte `Paint` emits that positions the cursor is asserted in-process — the frame is a placement, not a set of substrings.** Verified by reversion in a scratch copy: (a) replacing `menuRows+promptRows-1` with `len(menu)` — the exact BR-30 regression — leaves `go test ./cmd/define/` green; (b) deleting the entire `if len(menu) > 0 { … }` block, which would leave the cursor parked at the end of the last menu row so every keystroke redraws at the wrong place, also leaves it green. `TestScreenPaintSplitsTheHeight` (`screen_test.go:119`) checks only that named buffer lines and `"PROMPT"` appear somewhere and that the frame starts with home+erase. Add a geometry assertion that decodes the emitted frame into (row, col) and checks the final cursor position, and it covers BR-30, the reprint, and M2's underline splice at once.

**I-3 — BR-29 re-raised as `not-addressed`: the class fix was specified and not built.**
See dispositions below. `repo_guard_test.go` is untouched in this window, so the tree→table guard the finding named (using the `changedLines()` / `repoRoot()` helpers it pointed at) does not exist, and four of the symbols BR-29 itself enumerated still have no row.

## 4. Minor findings

- `cmd/define/editor.go:227` — `RenderLine`'s cursor park counts runes for a move the terminal makes in columns. Carried on BR-26 below; measured NFD `café` @cursor=2 → `ESC[3D` for a 2-column move, `日本語` @cursor=1 → `ESC[2D` for a 4-column move.
- `screen.go:453-493` and `render.go:236-250` each hand-roll a CSI-skipping scanner. The *measurement* has one owner (BR-27/BR-26 fixed that); the *scan* does not. Not worth a change now, but `M2.5` splices an SGR attribute into already-styled text and will want a third — extract one `forEachVisibleRune` seam when `M2.5` lands (ARCH-DRY).
- `screen_test.go:481` — `countingWriter` counts `Write` calls as "frames"; `Paint` happens to emit one `Write` per frame, so the count is right only by construction. A comment or a sentinel-prefix count would make it robust.
- `editorloop_test.go` — `TestEditorResizeRedrawsForTheNewShape` asserts menu width with `len([]rune(m)) > 40`, i.e. runes, in the same window that established cells as the one owner. Cosmetic in a test, but it is the measurement the milestone just retired.
- `liveScreen.Write` after `Stop()` still arms a `time.AfterFunc` that can only no-op (`screen.go:343-348`, `repaint` returns early on `stopped`). Harmless; a `stopped` check before arming would be tidier.

## 5. Test coverage notes

- **Green and reproduced:** `go test ./...` 102s, all packages ok; `go test -race ./cmd/define/` 109s ok. The round-3/round-4 Log claims check out.
- **Every pty row skipped here** (`no pty available: operation not permitted`) — all 11, including the two M1 rows that are the *sole* pin for done-when 3b and 5. BR-24's stated residue stands: `replRaw`'s own `enterAlt`/`enterMouse` on entry (`replraw.go:38,44`) are still pinned only by rows that skip. Per ARCH-MOCK a live conformance check that silently skips in every non-pty environment is not yet a conformance check; consider making the skip loud in CI, or asserting entry through the same `control` writer that made restore assertable.
- **`TestScreenFrameFitsTheTerminalInDisplayRows` fixtures are all ASCII** (`"x"×200`, `"z"×55`, `"m"×45`, coloured `"c"×100`). BR-26's round-5 note asked for one wide-rune and one combining-mark row precisely so the class could not come back; they are not there. The cell-width behaviour is pinned only by `TestVisibleCellsCountsColumns`, which does not exercise the frame.
- **No chunk-boundary property for `Write`** (BR-1). `FuzzScreenWriteDoesNotPanic` checks only for panics; `TestScreenWriteBuildsLines` uses fixed splits. The property — any split of the same byte stream yields the same lines — is one `f.Fuzz` over `(stream, splitPoints)` and would subsume four of the table rows.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass, with one open site.** `wheelFromButton`, `visibleCells`, `clamp`, `terminalSize` and the single row accounting are all one-owner now. The remaining violation is `editor.go:227` computing a column offset outside the owner (BR-26).
- **ARCH-PURE — pass.** Best feature of the diff. Keep it for M2: `RegionAt` belongs on `screen` (pure), and the underline splice belongs in `Paint`/`clipVisible` — note that `clipVisible` will need to *not* cut a region span in half, which the current per-rune loop makes easy to add.
- **ARCH-PURPOSE — flagged.** Two findings (BR-26, BR-29) were answered at the named instance while the enumeration each finding supplied was left partly unswept and the mechanism each asked for was not written. This is the "instance, not the class" pattern the principle names, and it is why both families are now three rounds deep.
- **ARCH-MOCK — pass, with the skip caveat above.** The `rawSession.control` seam is exactly the right boundary; production and test flow now share it.
- **ARCH-CONSTRAINTS — flagged.** The repaint envelope (16 ms + trailing flush), the uncapped buffer and the column half are all stated *and* enforced. The **row** half is stated and not enforced (I-1). Before M2, also decide the synchronized-output question: `Paint` emits `ESC[H ESC[J` + a full redraw up to 60×/s with no `ESC[?2026h` guard and no cursor hide, so a streamed answer tears and the cursor strobes. Not a correctness bug and BR-8's coalescing half is done — but M2 adds an underline attribute to that same frame, which makes the flicker more visible.

## 7. Plan revision recommendations

Add one `## Revisions` entry, `2026-08-29 — M1 review round 5`, carrying:

1. **The frame-fits rule, restated as a budget rather than a charge**, with the two measured overflow shapes (12×15 → 17 rows; 5×80 → 6 rows) and the design change: `Paint` clips the live edge by height, and `TestScreenFrameFitsTheTerminalInDisplayRows` becomes a property over shapes rather than four fixtures.
2. **BR-26's six-site enumeration, written into the plan** as the finding asked, with each site's current state (`visibleCells` ✓, `wrapText` ✓, `displayRows` ✓, `clipVisible` ✓, `truncate` ✓, `RenderLine`'s `ESC[nD` park ✗) — plus the two test rows (wide rune, combining mark) as an explicit deliverable, not prose.
3. **BR-29's tree→table direction as a guard, not a sweep**, and the M1 Core-concepts tables completed from the diff: `screen.Lines`, `screen.eraseOpenLine`, `paintInterval`, `defaultRows` / `defaultCols` (`main.go:1023-1024`) all currently have no row. Note the escape valve if the guard is judged too noisy — say so explicitly and say which declarations it would exempt, rather than leaving the direction unmechanised for a fourth round.
4. **M1.1's task line compressed** to the chunk-boundary property (BR-1), so the plan names the strategy rather than four cases the code has long since chosen.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      M1.1's prose still enumerates the four cases and the chunk-boundary property still has no test — FuzzScreenWriteDoesNotPanic checks only for panics. Minor, non-blocking.
  - id: BR-12
    disposition: addressed
    note: |
      Verified at HEAD: clipVisible cuts by cells (100 CJK runes -> 80 cells/40 runes) and a 10-line CJK buffer needs exactly 10 rows in a 10-row terminal; the remaining overflow route is the live edge, raised separately.
  - id: BR-23
    disposition: addressed
    note: |
      Reproduced green at HEAD: go test ./... (102s) and go test -race ./cmd/define/ (109s); both plan-table guards pass, and the "guards read the commit" lesson is in workshop/lessons.md.
  - id: BR-26
    disposition: not-addressed
    note: |
      Site 5 of its own enumeration survives — editor.go:227 counts runes for a column move (NFD "cafe" cursor=2 emits ESC[3D for a 2-column move; "日本語" cursor=1 emits ESC[2D for 4) — and the wide-rune/combining-mark rows it asked for were not added to TestScreenFrameFitsTheTerminalInDisplayRows, whose four fixtures are still ASCII.
  - id: BR-29
    disposition: not-addressed
    note: |
      The three named rows were added; the class was not. repo_guard_test.go is untouched in this window so the tree-to-table guard does not exist, and four symbols the finding itself enumerated still have no row: screen.Lines, screen.eraseOpenLine, paintInterval, defaultRows/defaultCols.
  - id: BR-30
    disposition: addressed
    note: |
      Verified at HEAD: Paint(&b,10,20,"> "+30xz,["m1","m2"]) now emits ESC[3A, the prompt's first row, from one summation that also budgets the buffer. No test fails without it — raised as a separate finding.
  - id: BR-31
    disposition: addressed
    note: |
      interval is a field set to time.Hour in the throttle test, so the Stop-flush case no longer races the real 16 ms window.
findings:
  - id: new
    severity: Important
    family: frame-fits-the-terminal
    title: |
      The live edge is charged display rows but never budgeted by them, so the frame still overflows the terminal
    detail: |
      This is the 3rd finding in family frame-fits-the-terminal. Earlier rounds fixed instances (BR-6 taught Paint display rows, BR-12 made cols real, BR-26 made the clip count cells). Do NOT fix this instance. The rule: the frame's TOTAL display height is asserted against termRows before it is written — every component, buffer AND live edge — or it is not a budget. Today screen.go:226 clamps only the buffer's share while prompt and menu are written unclipped and unlimited. Measured at HEAD: a 12-row/15-column terminal with "/" typed needs 17 display rows (opt.width is 0 below 20 columns, so truncate leaves 36-column menu rows to wrap three ways), and a 5-row/80-column terminal needs 6. The terminal then scrolls, every placed row moves, and M2's RegionAt maps a click to the wrong buffer line — the exact property M1 exists to establish. Fix: clip the live edge by height in Paint, and make TestScreenFrameFitsTheTerminalInDisplayRows a property over shapes rather than four ASCII fixtures.
  - id: new
    severity: Important
    family: unfalsifiable-test-pin
    title: |
      Nothing asserts where Paint leaves the cursor; the whole cursor-up-and-reprint block is deletable with a green suite
    detail: |
      This is the 4th finding in family unfalsifiable-test-pin. Earlier rounds fixed instances (BR-13, BR-7, BR-24). Do NOT just add a test for the cursor-up count. The rule: every byte Paint emits that POSITIONS the cursor is asserted in process — a frame is a placement, not a set of substrings. Verified by reversion in a scratch copy of HEAD: replacing menuRows+promptRows-1 with len(menu) at screen.go:244 (the exact BR-30 regression) leaves go test ./cmd/define/ green, and deleting the entire `if len(menu) > 0` block at screen.go:239-249 — which would leave the cursor at the end of the last menu row so every keystroke redraws in the wrong place — also leaves it green. TestScreenPaintSplitsTheHeight only checks that named lines and "PROMPT" appear and that the frame starts with home+erase. Decode the emitted frame into (row, col) and assert the final cursor position; that one assertion covers BR-30, the prompt reprint, and M2.5's underline splice.
```
