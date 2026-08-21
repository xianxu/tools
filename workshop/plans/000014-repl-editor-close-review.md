# Boundary Review — tools#14 (whole-issue close)

| field | value |
|-------|-------|
| issue | 14 — REPL line editor: history, prefix search, inline autosuggestion |
| repo | tools |
| issue file | workshop/issues/000014-repl-editor.md |
| boundary | whole-issue close |
| milestone | — |
| window | e85a0d077b3a16afd79727650733bbeb19e9d02a..HEAD |
| command | sdlc close --issue 14 |
| reviewer | claude |
| timestamp | 2026-08-20T17:23:17-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The editor is well-built: the pure core (`decodeKey`, `Apply`, `Suggestion`, `RenderLine`) is genuinely IO-free and table-tested, `runEditor` factors the terminal out behind `cooked`/`finish`/`<-chan Key` so the production loop is the tested loop, and the hardest constraint in the issue — Ctrl-C-as-a-byte during playback — is solved correctly (reader owns `cancel`, render cooked / play raw) and confirmed by a real pty run I executed: all three `TestPTY*` pass unsandboxed against the current binary (8.5s / 2.5s / 1.0s). Two correctness bugs must be fixed before the boundary: the key decoder leaks the tail of modified-Delete escape sequences into the line as literal text, and the raw loop bypasses `parseREPLLine`, so the interactive path no longer trims/collapses whitespace the way the piped path (and `#2`'s tested contract) does. Beyond that, the main gaps are test coverage that was deleted rather than moved (five deleted tests still pass verbatim against today's code — I re-ran them) and undeclared scope: wrapping, Tab, Ctrl-U, coloured prompt landed with no `## Revisions` entry and no README/atlas coverage.

## 1. Strengths

- **`cmd/define/main.go:149` — `lookupAndRender` split out of `defineOnce`.** The render-cooked/play-raw seam is the right factoring, and it keeps one define path instead of a parallel copy (ARCH-DRY). The `## Log` entry recording that the first attempt hung is honest and matched what I could verify on the pty.
- **`cmd/define/rawterm.go:43` — `readKeys` owns cancellation.** `cancel()` fires from the reader on `KeyInterrupt`, so it reaches a loop blocked inside `speak`. Verified end-to-end: `TestPTYCtrlCDuringPlaybackExitsPromptly` passes in 2.5s.
- **`cmd/define/replraw.go:32` — `cooked` mutates `*sess` in place** so the `finish` method value stays bound to live state. Easy to get wrong (`sess = s` would have silently broken restore); this is right.
- **`cmd/define/editor.go:43,48,54` — three-index slicing forces a copy on every line mutation**, so the `submitted := e` snapshot at `replraw.go:84` can't be aliased out from under the committed-line redraw. The classic `[]rune` editor bug is absent.
- **`cmd/define/repl.go:92-100` — `pipedInput` / `showPrompt` kept as two parameters with the four-regression history written down.** The lesson from `#2` was actually applied rather than restated.

## 2. Critical findings

**C-1 — `decodeKey` leaks the tail of modified-Delete sequences into the line as text.** `cmd/define/key.go:100-107`

The `case '3'` branch assumes byte 4 terminates the sequence. For `\x1b[3;5~` (Ctrl+Delete), `\x1b[3;2~` (Shift+Delete), `\x1b[3;3~` it returns `KeyUnknown` after **4** bytes, leaving `5~` / `2~` / `3~` in the buffer, which `readKeys` then decodes as two `KeyRune`s and inserts into the word. This violates the decoder's own doc comment ("swallow through its final byte so the tail never reaches the line as text") and the plan's Task 1 obligation ("An unknown sequence → `KeyUnknown` with its bytes, never a rune"). Verified:

```
decodeKey("\x1b[3;5~") = kind KeyUnknown consumed 4 ; remainder "5~"
```

`\x1b[1;5C` is handled correctly because it falls through to the generic final-byte scan — only the `'3'` special case is holed.

Fix sketch: in `case '3'`, return `KeyDelete` only when `buf[3] == '~'`; otherwise **break out of the switch** and let the generic `for i := 2; …` scan find the real final byte. Add `\x1b[3;5~` to `TestDecodeKeyUnknownSequencesAreInert`, and add a fuzz invariant that a buffer starting `\x1b[` never yields a `KeyRune`.

**C-2 — the raw loop bypasses `parseREPLLine`, so the interactive path silently changed what a line means.** `cmd/define/replraw.go:80,86-97` vs `cmd/define/repl.go:30`

`runEditor` uses `line := e.String()` and `if line == ""` where the line loop uses the pure decision table. The two loops now disagree, on the path humans actually use. Verified against the current code:

| typed | raw loop (today) | `parseREPLLine` contract (`repl_test.go:18-21`) |
|---|---|---|
| `sycophantic ` | `define: sycophantic : no dictionary entry` | trimmed → defines `sycophantic` |
| `␣␣` + Enter | `define:   : no dictionary entry` | `cmdReplay` — replays |
| `hot  dog` | looked up verbatim | collapsed → `hot dog` |

That last row is the multi-word-headword case the table exists for. `parseREPLLine` is documented as "the loop's decision table, kept pure so it is a unit test rather than something only reachable through a fake terminal" — and the new loop reimplements a subset of it inline (ARCH-DRY).

Fix sketch: in `runEditor`'s `ActSubmit`, run `parseREPLLine(e.String(), current != "")` and switch on `cmd.kind` exactly as `replLines` does; feed `cmd.word` to `submitLine` and to `hist.Add`. Add a raw-loop test for the trailing-space and whitespace-only cases.

## 3. Important findings

**I-1 — five deleted tests still pass verbatim; they were line-loop tests, not raw-loop tests.** `cmd/define/repl_test.go:49-53`

The NOTE says `#2`'s interactive tests "drove `repl()` with a `strings.Reader`. That path is now the LINE loop … so the editor's behaviour is exercised through `runEditor` instead." But all five were created with `replRig(…, interactive=false)` → `stdinTTY=false` → they always ran on the line loop, which this diff leaves intact. I restored them (adding only the new `cancel` argument) and ran them against HEAD:

```
--- PASS: TestZZREPLSecondWordBecomesCurrent
--- PASS: TestZZREPLUnknownWordLeavesCurrentUnchanged
--- PASS: TestZZREPLBlankWithNothingCurrentIsAHint
--- PASS: TestZZREPLReplayWithAudioOffIsAHint
--- PASS: TestZZREPLBareReturnReplaysWithoutRefetching
```

Only `TestREPLReplayFlashesThenRestoresThePrompt` genuinely died with `eraseLineAndStepBack`. Fix: restore the other five and correct the NOTE. This is plan-gate **PQ-7** ("does not name the existing tests that necessarily die"), still open — and the answer turns out to be "one, not six".

**I-2 — the raw loop's word-state and replay-hint branches have no coverage at all.** `cmd/define/replraw.go:118-127,143-146`

Untested on the path humans use: a failed lookup must not overwrite `current` (`submitLine:144`); `replayInPlace`'s "type a word, or press return to replay" branch; its "audio is off" branch; second-word-becomes-current. All four had line-loop equivalents (see I-1) and none were re-expressed against `runEditor`. `editorRig` already makes each a ~6-line test.

**I-3 — `readKeys` / `enterRaw` / `restore` have zero tests in the default build.** `cmd/define/rawterm.go:21-75`

The issue names Ctrl-C-in-raw-mode as "the single most likely regression", and its only coverage is `//go:build darwin && conformance`, which is on-demand — under the sandbox it doesn't even run (`pty.Start` → `operation not permitted` → `t.Skipf`, suite green). `readKeys` takes an `io.Reader`, so an `io.Pipe` test can pin: bytes split mid-escape reassemble into one `KeyUp`; `0x03` calls `cancel` *and* emits `KeyInterrupt`; a read error closes the channel. That is the fake-seam test ARCH-MOCK asks for at this boundary.

**I-4 — the `History` seam is declared but not wired.** `cmd/define/replraw.go:54,131` vs `cmd/define/history.go:15`

`runEditor` does `hist := &memHistory{}` and `submitLine` takes `hist *memHistory`; the `History` interface is referenced only by test helpers (`editor_test.go:11,28`). The Done-when — "History is consumed through a `History` seam, so `#3`'s store satisfies persistence" — is ticked, but `#3` will have to change these signatures to land its store. Fix: type both parameters `History` and take it as a `runEditor` argument.

**I-5 — undeclared scope, no `## Revisions` entry.** `workshop/plans/000014-repl-editor-plan.md`, `workshop/issues/000014-repl-editor.md`

Word wrapping + `terminalWidth` + `RenderOpts.Width`, Tab-accepts-suggestion, `KeyKillLine` (Ctrl-U / Cmd+Delete), and the coloured/bold prompt are all absent from the Spec and the Plan. AGENTS.md §1: "Revising a plan artifact mid-stream: append a `## Revisions` section (timestamp + reason + delta), don't overwrite." Wrapping in particular changes `define <word>` output on every terminal, one-shot path included — a user-visible format change carried in on a line-editor issue.

**I-6 — README and atlas miss the new user-facing surface.** `README.md:40-42`, `atlas/define.md:154-186`

README documents Up/Down/Right only. Missing: **Tab** and **End** accept the suggestion, **Ctrl-U / Cmd+Delete** clears the line, Ctrl-D on an empty line quits, and output is now hard-wrapped to the terminal width. The atlas section likewise names only Right/End and says nothing about wrapping, `Width`, or `terminalWidth`. README's "On a terminal, `define` … opens a line editor" also needs the `-no-color` caveat the atlas states correctly (with `-no-color`, raw mode is never entered and there is no prompt).

**I-7 — the atlas states a property the code doesn't have.** `atlas/define.md:169`, `cmd/define/editor.go:9-10`

"Candidates are passed in as a plain slice rather than a `History` handle, so **no store query runs per keystroke** once `#3` fills that seam." `runEditor` calls `hist.Prefix(e.WalkBase())` twice on every keystroke (`replraw.go:57` and `:73`). `Apply` is pure — that part is true — but the query count claim is false and will mislead `#3`. Fix: reword to "`Apply` never queries; the loop resolves candidates once per keystroke", and resolve `matches` once per iteration instead of twice.

**I-8 — silent error swallow re-entering raw mode.** `cmd/define/replraw.go:35`

`if s, err := enterRaw(f); err == nil { *sess = *s }` discards the error. If re-entry fails after a lookup, the editor keeps drawing frames while the tty is cooked: input becomes line-buffered, keystrokes echo over the definition, and nothing tells the user. Fix: on error, report on stderr and return to the line loop (or exit), rather than continuing in a state the loop's model doesn't describe.

## 4. Minor findings

- `cmd/define/main.go:210` — `playAnnounced`'s error write uses `\n`, but on the raw path it executes in raw mode; every other raw-mode write in this diff uses `\r\n`. An audio failure leaves a ragged extra row.
- `cmd/define/replraw.go:121-123` — `replayInPlace` writes `eraseLine` (ANSI) to **stderr**; `define 2>err.txt` on a terminal captures escapes.
- `cmd/define/replraw.go:107` — Ctrl-C during playback draws one more prompt frame before the next `select` sees `ctx.Done()`, leaving a stray prompt above the shell prompt.
- `cmd/define/editor.go:177,197` — `RenderLine`'s `\r\x1b[K` plus `\x1b[ND` assume the frame fits one terminal row; a word longer than the width leaves residue. Undocumented.
- `cmd/define/editor.go:151` — `len(m) > len(typed) && m[:len(typed)] == typed` is `strings.HasPrefix` plus a length test (ARCH-DRY nit).
- `cmd/define/render.go:187` — `wrapText` uses `strings.Fields`, collapsing runs of whitespace inside a gloss; the no-loss invariant compares alnum runes only, so it cannot see this.
- `cmd/define/render.go:123-124` — example lines pass `len(indent)+2` as the wrap lead, but the rendered line also carries `“` and an optional label, so the first line can exceed the width by 1 + label.
- `cmd/define/render.go:162` — `visibleLen` counts runes; CJK/double-width entries (README notes other dictionaries can answer) will wrap short.
- `cmd/define/main.go:245` — width is sampled once at startup; SIGWINCH isn't handled.
- `cmd/define/key.go:65` — Ctrl-A / Ctrl-E / Ctrl-W are inert. Out of plan scope; worth a line in `#15`'s spec since they're muscle memory next to Ctrl-U, which *is* now bound.
- `cmd/define/history.go:28` — `Add` discards `found`; nothing pins the flag's contract until `#3` stores it.
- `cmd/define/editor.go:184` — the `color == false` branch of `RenderLine` is unreachable in production (`opt.color` and `opt.tty` are set from the same expression at `main.go:98,102`, and raw mode requires `opt.tty`), and `RenderLine` emits `\x1b[K` regardless. The plan's Task 5 obligation "with `Color:false` **no ANSI at all**" is therefore vacuous rather than met.
- `cmd/define/pty_conformance_test.go:85` — `t.Skipf` on `pty.Start` failure means a sandboxed or CI run reports green having asserted nothing. Consider `t.Fatal` when a `CONFORMANCE_STRICT`-style env var is set.

## 5. Test coverage notes

- Pure core is well covered: `key_test.go` (table + partials + unknown-sequence + fuzz), `editor_test.go` (walk, draft restore, prefix search, dedup, suggestion accept/reject, kill-line, the "Enter never submits the suggestion" test the plan called the most damaging possible bug). `TestSuggestionNotAcceptedByOtherKeys` and `TestKillLineEndsTheHistoryWalk` are exactly the right shape — they pin behaviour, not implementation.
- `-race ./cmd/define/` passes; `go vet` clean with and without the conformance tag.
- Real gaps, in priority order: C-1's escape-tail case (no test would have caught it), C-2's whitespace contract on the raw path (the contract is tested only for the loop that no longer serves humans), I-3's `readKeys`, I-2's four behavioural branches, I-1's five recoverable tests.
- The fuzz target asserts `consumed` bounds but not the decoder's actual contract. `if n > 0 && buf[0] == 0x1b && k.Kind == KeyRune { t.Fatal }` would have found C-1 immediately.

## 6. Architectural notes

- **ARCH-DRY — flag.** C-2 (`parseREPLLine` bypassed; the two loops disagree on what a line means) and the `strings.HasPrefix` reimplementation. Positives worth keeping: `lookupAndRender`/`defineOnce`, `playAnnounced` still the single owner of announce→play→erase, and `newCachingAudioSource` wired inside *both* loops rather than at the caller.
- **ARCH-PURE — pass, and this is the diff's strongest axis.** `decodeKey`, `Apply`, `Suggestion`, `RenderLine`, `wrapText`, `visibleLen` are pure and tested with no terminal; `runEditor` takes the terminal as `cooked`/`finish`/`<-chan Key`; `terminalWidth` stays at the boundary and hands `Render` an int. The plan's stated goal — "a pty test covers only the raw-mode plumbing" — is achieved.
- **ARCH-PURPOSE — flag (minor).** Every Spec item ships. But the `History` seam is nominal (I-4): the Done-when's point was that `#3` can drop its store in without touching the editor, and today it must edit `runEditor`/`submitLine` signatures. Run the shadow-sweep before closing: the seam has exactly one consumer, and that consumer doesn't derive from it.
- **ARCH-MOCK — pass with a gap.** The scripted key channel is a proper fake at the same boundary production uses (`runEditor` is shared), and the live pty conformance check exists with a stated on-demand cadence — I ran it and it passes. The gap: the fake seam is entered *below* `readKeys`, so the byte→key→cancel layer has no fake-driven test and its only coverage is the pty run that silently skips when unavailable. An `io.Pipe`-driven `readKeys` test closes it (I-3).

## 7. Plan revision recommendations

Append to `workshop/plans/000014-repl-editor-plan.md` `## Revisions`:

1. **Core concepts table vs code.** `rawTerminal` → `rawSession`; `keyReader` → `readKeys`; `scriptedKeys` in `cmd/define/rawterm_test.go` → `scriptKeys` in `cmd/define/editorloop_test.go`; `Editor`'s `Hist *histCursor` → inline `draft`/`walkIdx`/`walkBase` (no `histCursor` type exists); `RenderLine(e, sug string, opt RenderOpts)` → `RenderLine(e, sug string, color bool)`. State that `RenderOpts` was deliberately **not** reused (this is the resolution of open gate finding **PQ-6**, which is otherwise still listed as unaddressed).
2. **Test file locations.** Task 3 names `history_test.go` (never created — history tests live in `editor_test.go`); Task 5 names `render_test.go` (`RenderLine`'s test is `TestRenderLineMakesTheInputLineDistinct` in `editorloop_test.go`).
3. **Scope added after approval** (I-5): `KeyKillLine` (Ctrl-U / Cmd+Delete), Tab-accepts-suggestion, coloured/bold prompt, and word wrapping (`RenderOpts.Width`, `wrapText`, `visibleLen`, `terminalWidth`) — with the operator-feedback reason and the note that wrapping changes one-shot `define <word>` output too. Mirror a short entry in the issue's `## Revisions`.
4. **PQ-7 resolution** (I-1): name the one test that genuinely died (`TestREPLReplayFlashesThenRestoresThePrompt`) and record that the other five were line-loop tests deleted unnecessarily and restored.
5. **Task 5's `Color:false` obligation** is unreachable in production (Minor above) — restate it as "`-no-color` never enters raw mode; `RenderLine` is only exercised with colour on" so the plan stops implying a contract the frame-control `\x1b[K` doesn't meet.
