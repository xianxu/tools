# Boundary Review — tools#67 (milestone M1)

| field | value |
|-------|-------|
| issue | 67 — define: read-along — paste a passage, click or drag what is opaque |
| repo | tools |
| issue file | workshop/issues/000067-read-along-passage.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 98f5c779b468ada00c087bde6bd43cca9b0892cc..c1844b33a9c9f8f7e9efb70d6930fed332233c49 |
| command | sdlc milestone-close --issue 67 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-16T15:27:51-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The paste scanner itself is the strongest part of this milestone — the "accumulates nothing" contract against `readInput`'s re-presenting buffer is correct, well-argued in prose, and pinned by tests that drive it the way the caller actually drives it, including a cross-call fuzz target (30s, no crashers). The terminal-mode work is exemplary: `enterPaste`/`leavePaste` mirror the mouse pair exactly, the ordered teardown is asserted, the flag is set only on a successful write, and widening `TestEveryEnabledMouseModeIsDecoded` → `TestEveryEnabledInputModeIsDecoded` closes a real hole in the guard rather than working around it. Two things block SHIP. First, **the suite is red at HEAD**: `TestARemovedDeclarationIsSweptOrRetired` fails because `render.go:284` still names the test this window renamed — the issue Log's "full suite green" claim does not hold at the pinned commit (I reproduced it in a clean checkout of `c1844b3`). Second, an unterminated `ESC[200~` now **permanently deafens the input path**, swallowing Enter and Ctrl-C — I reproduced this through the real `readKeys`/`readInput` goroutine; it is a regression against the base, the plan named the case and promised a pinning test, and neither the test nor the documented limit shipped. Separately, the one piece of production state this milestone introduced (`readInput`'s long-lived `keyDecoder`) has no test at all: I reverted it to the stateless `decodeKey(buf)` in a scratch worktree and the **entire suite stayed green**.

## 1. Strengths

- **`paste.go:50-95` — the scanner's statelessness is the right call and is defended by the right test.** `TestPasteScannerSurvivesAReadBoundary` and `TestPasteScannerTakesAPasteDeliveredInChunks` (`paste_test.go:33,52`) drive one scanner with a growing, unconsumed buffer — the exact shape `selection_input.go:174` presents. This is a regression test that would actually go red.
- **`key.go:87-101` — the hook condition is `d.paste.draining || buf[0] == 0x1b`, not "on ESC".** The comment states why, and `TestAnOversizePasteKeepsDrainingAcrossReads` (`paste_test.go`) is a real pin for it: a read boundary landing on ordinary body text.
- **`key.go:79-90` — keeping a package-level `decodeKey` wrapper** leaves 38 call sites and both fuzz targets untouched and makes "a fresh decoder cannot be mid-paste" structurally true. `TestAFreshDecodeKeyIsNotMidPaste` pins it.
- **`key_test.go:402-462` — the widened mode guard is the fix, not a waiver.** It reads `mouseOn + pasteOn`, exercises the 2004 rows through `decodeKey`, and names why 1049 is excluded. The rows are reachable (they run and assert full consumption).
- **`rawterm.go:160-195` + `rawterm_test.go:119-150` — ARCH-FUNERAL done properly for a terminal mode**: paste-off is emitted in the ordered teardown before the alt screen, with an ordering assertion, not just a "we call it somewhere" check.

## 2. Critical findings

**C1 — `cmd/define/paste.go:79-93` (+ `key.go:87`): an unterminated `ESC[200~` locks the input path forever, swallowing Enter and Ctrl-C.**

Verified empirically. Driving `readKeys(ctx, strings.NewReader("\x1b[200~" + strings.Repeat("x", maxPasteRunes+50) + "hello\r\x03"), ...)` through the real `readInput` goroutine yields **two inert `KeyUnknown` keys and nothing else** — the `\r` and the `\x03` never emerge. Because raw mode disables `ISIG`, `interrupts.Fire()` is only reachable from a decoded `KeyInterrupt`, so the program cannot be quit from the keyboard.

Two phases, both unbounded:
- Under the cap, `scan` returns `used == 0` forever (`paste.go:83`), `readInput` never advances `buf` (`selection_input.go:178-180`), and every key typed afterwards joins that buffer and is re-scanned.
- Over the cap, `draining` latches (`paste.go:90`) and every subsequent byte is consumed and discarded until a closer that will never arrive.

This is a strict regression: at the base commit `\x1b[200~` decoded as an inert 6-byte `KeyUnknown` and everything after it worked. The plan flagged exactly this (Chunk 1, Task 1.2b item 1) and called it "a real, bounded degradation; **state it as a known limit with a test**" — but the bound does not exist, the promised `TestAnUnterminatedPasteRecoversAtTheCap` was not written, and `atlas/define.md` does not record the limit either. ARCH-SECURE: input this process did not produce is trusted to be well-formed, and the failure path hangs rather than degrading visibly. ARCH-ORDER: a latched state with no exit event other than one the peer may never send.

*Fix sketch:* give the drain a bounded exit — e.g. a byte budget after which `draining` clears and the scanner returns `KeyPasteRefused` (self-healing), or treat `\x03`/`\r` seen while draining as an abort. Under the cap, the same budget should convert a stalled open marker into a refusal rather than silence. Then write the promised pin: drive `readKeys` over an `io.Pipe` and assert a `KeyInterrupt` still arrives after a never-closed paste.

**C2 — `cmd/define/render.go:284`: the suite is red at HEAD; a renamed test was not swept.**

```
--- FAIL: TestARemovedDeclarationIsSweptOrRetired
    repo_guard_test.go:1719: cmd/define/render.go names "TestEveryEnabledMouseModeIsDecoded",
    which this window REMOVED and which is not in retiredSymbolNames.
```

The guard **skips** at the base commit ("no change window: HEAD is the merge-base with main") and **fails** at `c1844b3`, so this window introduced it. `doc_sync_test.go:390` and `:427` carry the same stale name (outside `currentTruthFiles`, so unflagged, but equally stale), as does the key_test doc comment at `key_test.go:399` ("Adding a mode to **mouseOn** without adding a row here…", now `mouseOn + pasteOn`).

*Fix sketch:* sweep the three mentions to `TestEveryEnabledInputModeIsDecoded` (or add the mapping to `retiredSymbolNames`), and re-run. The only other failures on this host are `TestLanguagePromptStartup` / `TestLanguageTintInvocation`, which fail at `pty.Open(): operation not permitted` — an environment limit, not this diff.

## 3. Important findings

**I1 — `cmd/define/selection_input.go:165,178`: the one piece of production state this milestone adds is untested.** In a scratch worktree at `c1844b3` I replaced `dec.decode(buf)` with `decodeKey(buf)` — reverting the long-lived decoder — and the **full package suite produced the identical three failures, zero new ones**. That revert reintroduces precisely the bug the drain exists to prevent (with a fresh decoder per call, the held-back tail and every later chunk of an over-cap paste decode as `KeyRune`). Every paste test calls `scan`/`decode` directly and simulates re-presentation by hand; the production seam is never driven. ARCH-ORDER: the oracle only observes an interleaving the author constructed. The seam already exists and is already used — `readKeys` + `io.Pipe` in `TestReadKeysHandlesSequencesSplitAcrossReads` (`rawterm_test.go:50`). Add a paste-split-across-reads row and an oversize-paste row there.

**I2 — `workshop/plans/000067-read-along-passage-plan.md`: the durable plan contradicts the shipped code and has no `## Revisions` entry.** Every M1 checkbox (Tasks 1.1–1.5, lines 123–469) is still `- [ ]` while the issue marks `- [x] M1`. Worse, the plan's prose now describes code that does not exist: Task 1.2b states "**So `newPassage` is the parse boundary**" and its tests call `newPassage(...)`, but the boundary shipped as `sanitisePasteBody` in `paste.go:109`; Task 1.4 Step 3 specifies `Apply` gets `case KeyPaste, KeyPasteRefused: return e, ActNone` and that `runEditor` intercepts **both**, but `editor.go:58-70` inserts the paste and `replraw.go:534` intercepts only `KeyPasteRefused`. The deviations are recorded in the issue `## Log`, which is the right place for the narrative — but AGENTS.md §1 requires the plan artifact itself to carry a `## Revisions` entry, and M2 will be read off this plan. The Pure-entities table also omits three new pure entities this milestone added: `sanitisePasteBody`, `pasteLineRunes` (both `cmd/define/paste.go`) and the `keyDecoder` type (`cmd/define/key.go`).

**I3 — `cmd/define/README.md`: no README update for user-visible paste behaviour (Docs gate).** That README documents interactive input in detail — a Keys table at `:113-119` and "Select and copy text" at `:63-74`, including the "type-ahead is full" notice. This window changes what a user sees when they paste: a multi-line paste now lands as **one line with newlines and tabs turned into spaces** (`paste.go:143`), instead of submitting mid-paste; and a paste over 1000 characters is refused with `define: that paste is longer than 1000 characters; paste less` (`replraw.go:539`). Neither the flattening nor the limit and its message appears anywhere a user reads. `atlas/define.md` covers the mechanism well but also omits `pasteLineRunes` — that newlines become spaces is the one behaviour a reader will actually hit, and it is the documented deviation from the plan.

**I4 — `cmd/define/paste_test.go`: the plan's "decide it, don't inherit it" pin for a paste during a live drag is missing.** Task 1.4's preamble is explicit: a `KeyPaste` reaches `pointerRouter.route` (`selection_input.go:28`) and, being non-pointer and non-`KeyUnknown`, hits `cancelPointerInput(l, k, true)` at `:47` — cancelling a live drag — and "it must be a decision with a test, not an inherited side effect." Step 1 lists the test; no test in the diff touches it. Same class as C1's missing pin: the plan enumerated the decisions this milestone had to nail down and two of them shipped unnailed.

## 4. Minor findings

- `cmd/define/paste.go:128` — `unicode.IsControl` is category Cc only, so bidi/format controls (U+202E `RLO`, ZWJ, U+200B) survive the boundary. `practice_help.go:140` already treats `unicode.Bidi_Control` as a distinct hazard for terminal-bound text; decide this before M2 puts the passage on screen and M4 sends it to the model.
- `cmd/define/key.go:66` — the `Key` struct doc still says "Raw carries the bytes of an unmodelled sequence so it can be ignored rather than inserted as garbage". For `KeyPaste`, `Raw` is sanitised text that *is* inserted (`editor.go:64`). The field now carries two trust levels; the doc asserts one.
- `cmd/define/paste_test.go:163` — `TestTheBoundaryKeepsNewlinesAndDropsOtherControls` accepts either `"a\nb\tc de"` or `"a\nb\tcde"`, so it pins neither behaviour for NUL/BEL. Pick one.
- `workshop/issues/000067-read-along-passage.md` M1 row — says "the 1000-**byte** cap"; the code, the plan and the atlas all say runes, and the rune-vs-byte choice is the argued point.
- `cmd/define/paste_test.go:134` — `TestTheDrainDoesNotCutAStraddlingCloser`'s second `scan` is handed a fresh `"tail"+pasteEnd` rather than the leftover the caller would actually re-present; the assertion holds but not for the drive pattern it names. The fuzz target covers the real shape.
- `cmd/define/play_loop.go:566` — `toInput` returns `false` for `KeyPaste`, so a paste during practice is silently dropped. Almost certainly right (practice keys are single keystrokes), but undecided-by-default rather than decided.

## 5. Test coverage notes

- Coverage of the scanner as a unit is genuinely good: 12 unit tests plus `FuzzPasteScannerAcrossCalls` (ran 30s clean; `FuzzDecodeKey` 25s clean). The refusal, the drain, the straddling closer, the embedded closer, the rune-vs-byte cap and the escape boundary all have real assertions.
- The gap is the seam, not the unit (I1). The single highest-value addition is a `readKeys` + `io.Pipe` test: paste split across reads, and an oversize paste whose tail must not reach the line. That one test would cover I1, and with a bounded drain it covers C1 too.
- Two pins the plan promised and the milestone did not deliver: `TestAnUnterminatedPasteRecoversAtTheCap` (C1) and the paste-cancels-a-live-drag test (I4).

## 6. Architectural notes

- **ARCH-DRY — pass.** `sanitisePasteBody` routes through `escapeLen` (`render.go:573`) rather than writing a second escape grammar; `decodeKey` is kept as a wrapper rather than churning 38 call sites; the mode guard reads off the constants instead of restating them.
- **ARCH-PURE — pass.** The scanner is a pure function of `(draining, buffer)`; every test runs with no IO. The IO shell is four small edits at existing seams.
- **ARCH-PURPOSE — flag (C1/I4).** M1's stated purpose is delivered, and the deviation that keeps a pasted word working is the right call. But Task 1.2b named a **class** — three degenerate inputs — and swept two: the embedded closer and body escapes shipped with tests, the unterminated paste shipped with neither a bound nor a pin. That is the instance, not the class.
- **ARCH-MOCK — flag (Minor).** Mode 2004 is an external terminal behaviour this feature rests on entirely, asserted only through the `control io.Writer` seam on the *write* side. There is a pty conformance harness (`pty_conformance_test.go`) and no row asserting a real terminal round-trips a bracketed paste. Those rows skip in this environment (#37), so this is a note, not a gate.
- **ARCH-CONSTRAINTS — mostly pass, one flag.** The rune cap is enforced and the drain keeps `readInput`'s buffer bounded (~4 KB while waiting, ~261 B while draining) — that is the bound the comment claims, and it holds. Flag: the drain emits one `KeyUnknown` per read into a 256-cap channel whose overflow fires "input full — newest key ignored" (`selection_input.go:190`), so a very large refused paste can surface that notice rather than, or before, the refusal message.
- **ARCH-SECURE — flag (C1, Minor bidi).** `sanitisePasteBody` is a genuine parse-at-the-boundary, which is the right shape. The failure path is the problem: malformed input hangs instead of degrading visibly, and the rune class the boundary rejects is narrower than the hazard class.
- **ARCH-ORDER — flag (C1, I1).** Two states with the transitions embedded in control flow is fine at this size, and the cross-call fuzz is the right instinct. What is missing is the production ordering seam under test, and an exit from `draining` that does not depend on a peer's cooperation.
- **ARCH-FUNERAL — pass.** Nothing durable is created; the scanner dies with the goroutine; mode 2004's end is explicit, ordered and asserted.

## 7. Plan revision recommendations

Append a `## Revisions` section to `workshop/plans/000067-read-along-passage-plan.md` (it currently has none) with a `2026-09-16 — M1 as built` entry covering:

1. **The parse boundary moved from `newPassage` to the scanner.** Rewrite Task 1.2b: `sanitisePasteBody` in `cmd/define/paste.go` is the boundary; its three tests are `TestAPastedEscapeNeverLeavesTheBoundary`, `TestTheBoundaryKeepsNewlinesAndDropsOtherControls`, `TestAnEmbeddedCloserEndsThePaste`. Record the reason already in the issue Log (the planned tests referenced an M2 symbol).
2. **`Apply` inserts a paste rather than returning `ActNone`.** Rewrite Task 1.4 Step 3: `Apply` handles `KeyPaste` via `pasteLineRunes` (newlines/tabs → spaces) and `KeyPasteRefused` as a no-op; `runEditor` intercepts only `KeyPasteRefused`. Record why (a `KeyPaste` with no destination would regress the working single-word paste).
3. **Carry the two unshipped pins forward, or restate them as M1 rework:** `TestAnUnterminatedPasteRecoversAtTheCap` and the paste-cancels-a-live-drag test. Correct the Task 1.2b claim that the wait is "a real, bounded degradation" — after the cap the drain is unbounded.
4. **Add the M1 entities to the Core concepts tables:** `sanitisePasteBody` and `pasteLineRunes` (`cmd/define/paste.go`, new, pure) and `keyDecoder` (`cmd/define/key.go`, new).
5. **Tick the M1 checkboxes** (Tasks 1.1–1.5) so the plan stops claiming the milestone is unstarted.

Also correct the issue's M1 row: "the 1000-**byte** cap" → "the 1000-**rune** cap".

```findings
findings:
  - id: new
    severity: Critical
    family: external-state-needs-bounded-exit
    title: |
      An unterminated ESC[200~ permanently deafens the input path, swallowing Enter and Ctrl-C
    detail: |
      Verified through the real readKeys/readInput goroutine: feeding "\x1b[200~" + 1050 'x'
      + "hello\r\x03" yields two inert KeyUnknown keys and nothing else — the carriage return
      and the interrupt never emerge. Under the cap scan returns used==0 forever so readInput
      never advances buf (selection_input.go:178) and later keystrokes join the same buffer;
      over the cap `draining` latches (paste.go:90) and every byte is discarded until a closer
      that never arrives. Raw mode disables ISIG, so Ctrl-C is only reachable via a decoded
      KeyInterrupt and the program cannot be quit from the keyboard. A regression against the
      base, where ESC[200~ was an inert 6-byte KeyUnknown. The plan (Task 1.2b item 1) named
      this case and promised TestAnUnterminatedPasteRecoversAtTheCap; neither the bound, the
      test, nor the atlas note shipped. ARCH-SECURE, ARCH-ORDER.
  - id: new
    severity: Critical
    family: removed-symbol-unswept
    title: |
      Suite is red at HEAD — render.go still names the test this window renamed
    detail: |
      TestARemovedDeclarationIsSweptOrRetired fails at c1844b3 in a clean checkout:
      cmd/define/render.go:284 names TestEveryEnabledMouseModeIsDecoded, removed by this
      window and absent from retiredSymbolNames. The guard SKIPS at the base commit, so this
      window introduced the failure, and the issue Log's "full suite green" claim does not
      hold. doc_sync_test.go:390 and :427 carry the same stale name, and key_test.go:399's doc
      comment still says "Adding a mode to mouseOn" after the constant became mouseOn+pasteOn.
  - id: new
    severity: Important
    family: production-seam-untested
    title: |
      readInput's long-lived keyDecoder has no test — reverting it leaves the whole suite green
    detail: |
      In a scratch worktree at c1844b3 I replaced dec.decode(buf) with decodeKey(buf) at
      selection_input.go:178, reverting the stateful wiring this milestone added. The full
      cmd/define suite produced the identical three failures and zero new ones. That revert
      reintroduces exactly what the drain exists to prevent: with a fresh decoder per call the
      held-back tail and every later chunk of an over-cap paste decode as KeyRune. Every paste
      test drives scan/decode directly and simulates re-presentation by hand. The seam already
      exists and is already used by TestReadKeysHandlesSequencesSplitAcrossReads
      (rawterm_test.go:50); add a paste-split-across-reads row and an oversize-paste row there.
      ARCH-ORDER — the oracle observes only the interleaving the author constructed.
  - id: new
    severity: Important
    family: plan-artifact-stale
    title: |
      The durable plan contradicts the shipped code and carries no Revisions entry
    detail: |
      workshop/plans/000067-read-along-passage-plan.md has all M1 checkboxes (Tasks 1.1-1.5,
      lines 123-469) still unticked while the issue marks M1 done, and no "## Revisions"
      section at all (AGENTS.md section 1). Its prose describes code that does not exist: Task
      1.2b states "newPassage is the parse boundary" and its tests call newPassage(), but the
      boundary shipped as sanitisePasteBody (paste.go:109); Task 1.4 Step 3 specifies Apply
      returns ActNone for KeyPaste and that runEditor intercepts both kinds, but editor.go:58
      inserts the paste and replraw.go:534 intercepts only KeyPasteRefused. The Pure entities
      table also omits sanitisePasteBody, pasteLineRunes and the keyDecoder type. M2 will be
      read off this plan.
  - id: new
    severity: Important
    family: readme-surface-undocumented
    title: |
      cmd/define/README.md not updated for the paste behaviour or the 1000-character refusal
    detail: |
      That README documents interactive input in detail (Keys table at :113-119, "Select and
      copy text" at :63-74, including the type-ahead-full notice). This window changes what a
      user sees: a multi-line paste now lands as ONE line with newlines and tabs turned into
      spaces (paste.go:143) instead of submitting mid-paste, and a paste over 1000 characters
      is refused with "define: that paste is longer than 1000 characters; paste less"
      (replraw.go:539). Neither appears in the README. atlas/define.md covers the mechanism
      well but also omits pasteLineRunes — the flattening is the one behaviour a user hits and
      it is the documented deviation from the plan.
  - id: new
    severity: Important
    family: decided-behaviour-unpinned
    title: |
      No test for a paste arriving during a live drag, which the plan required as a decision
    detail: |
      Task 1.4's preamble: a KeyPaste reaches pointerRouter.route (selection_input.go:28) and,
      being non-pointer and non-KeyUnknown, hits cancelPointerInput(l, k, true) at :47 —
      cancelling a live drag — and "it must be a decision with a test, not an inherited side
      effect." Step 1 lists the test; nothing in the diff exercises it. Same class as the
      missing unterminated-paste pin.
  - id: new
    severity: Minor
    family: boundary-parses-partial-class
    title: |
      sanitisePasteBody drops Cc controls but lets bidi/format controls through
    detail: |
      unicode.IsControl (paste.go:128) is category Cc only, so U+202E RLO, ZWJ and U+200B
      survive the boundary. practice_help.go:140 already treats unicode.Bidi_Control as a
      distinct hazard for terminal-bound text. Decide before M2 puts the passage on screen and
      M4 sends it to the model. ARCH-SECURE.
  - id: new
    severity: Minor
    family: doc-contradicts-type
    title: |
      Key struct doc still says Raw is an unmodelled sequence to be ignored, not inserted
    detail: |
      key.go:66. For KeyPaste, Raw is sanitised text that IS inserted (editor.go:64). The
      field now carries two trust levels and the doc asserts one; the KeyPaste const comment
      says the right thing but the struct doc above it does not.
  - id: new
    severity: Minor
    family: test-accepts-two-outcomes
    title: |
      TestTheBoundaryKeepsNewlinesAndDropsOtherControls accepts either outcome, pinning neither
    detail: |
      paste_test.go:163 passes for both "a\nb\tc de" and "a\nb\tcde", so NUL/BEL handling is
      unpinned. Pick one. Relatedly, TestTheDrainDoesNotCutAStraddlingCloser at :134 hands the
      second scan a fresh buffer rather than the leftover the caller would re-present.
  - id: new
    severity: Minor
    family: issue-row-stale
    title: |
      The issue's M1 row says "1000-byte cap"; the code, plan and atlas all say runes
    detail: |
      workshop/issues/000067-read-along-passage.md, M1 checkbox. Rune-vs-byte is the argued
      point of the cap, so the tracker row inverts it.
```
