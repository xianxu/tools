# REPL line editor — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Up/Down walk history, Up with a prefix typed walks only matching entries, and the most recent match appears ahead of the cursor in grey — accepted with Right/End, ignored by anything else.

**Architecture:** A **pure editor state machine** — `(Editor, Key) → Editor` — plus a pure `Render(Editor) string`. Raw mode and the byte→key decoder are the only IO, and both are thin. Everything the issue asks for is then a table test over key sequences with no terminal anywhere; the pty test covers only that raw mode is entered, restored, and decoded.

**Tech Stack:** Go 1.26, `golang.org/x/term` (`MakeRaw`/`Restore` — already a dependency).

**Milestone:** single-pass. One boundary, closed with `sdlc close` — no `Mx` tags.

---

## The dependency decision the issue demanded

**Own it. No new third-party runtime dependency.** Recorded here because the issue
requires the choice be explicit.

- `x/term` already ships `MakeRaw`, `Restore`, `GetSize`, `IsTerminal`, and is
  already in `go.mod`. The raw-mode half is a dependency we already pay for.
- What remains is a **single-line** editor: printable insert, Left/Right/Home/End,
  Backspace/Delete, Up/Down, Ctrl-C/Ctrl-D, Tab. That is a bounded state machine,
  not a project.
- The feature that actually matters here — **grey inline autosuggestion** —
  is not offered by `chzyer/readline` or `peterh/liner`; it would have to be
  built against their internals either way.
- `bubbletea` would do it, but it is a full-screen framework and this REPL is
  line-oriented with real scrollback. Adopting it means rewriting output, not
  just input.
- Owning the render is also what makes `#2`'s cooked-mode workaround deletable
  (below), which a library would not give us.

The cost is honest: ~300 lines of editor and a key decoder we now maintain.

## Deleting the cooked-mode workaround

`#2` placed the "♫ playing N×" indicator over the prompt with cursor arithmetic —
`eraseLineAndStepBack`, `skipPrompt` — because the terminal echoes Enter before we
can draw. It also documented the limitation that this **breaks if the user types
during playback**, since cooked-mode echo moves the cursor asynchronously.

In raw mode nothing is echoed, so there is no echo to undo: the frame is
rendered with the indicator where the prompt would be. **Both `eraseLineAndStepBack`
and `skipPrompt` are deleted in this issue, not ported** — `#14`'s own file
commits to this. Keystrokes arriving during playback are read as keys, not echoed
as bytes, which removes the limitation rather than documenting it.

## Non-goals

Carried from `#2` and still out of scope: multi-line editing, kill-ring,
incremental reverse search (Ctrl-R), vi mode, completion of *words* (`#15` adds
completion of `/`-commands only), and mouse.

---

## Chunk 1: Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Key` | `cmd/define/key.go` | new |
| `decodeKey` | `cmd/define/key.go` | new |
| `Editor` | `cmd/define/editor.go` | new |
| `Apply` | `cmd/define/editor.go` | new |
| `Suggestion` | `cmd/define/editor.go` | new |
| `RenderLine` | `cmd/define/editor.go` | new |
| `History` | `cmd/define/history.go` | new |
| `memHistory` | `cmd/define/history.go` | new |

- **Key** — one decoded keypress: `KeyRune{r}`, `KeyLeft/Right/Up/Down/Home/End`,
  `KeyBackspace`, `KeyDelete`, `KeyEnter`, `KeyTab`, `KeyInterrupt`, `KeyEOF`,
  `KeyUnknown{raw}`.
  - **DRY rationale:** the escape-sequence vocabulary in exactly one place.
    `KeyUnknown` retains the raw bytes so an unhandled sequence is inert rather
    than inserted as garbage — the failure mode of a naive decoder.

- **decodeKey(buf []byte) (Key, consumed int)** — pure: bytes in, key + how many
  bytes it ate. Returns `consumed == 0` when the buffer holds a **prefix** of a
  longer sequence, so the caller reads more rather than guessing.
  - **Future extensions:** bracketed paste, Alt-modified keys — new cases, same
    signature.

- **Editor** — `Line []rune`, `Cursor int`, `Hist *histCursor`. The whole
  editing state, no IO.

- **Apply(e Editor, k Key, h History) (Editor, Action)** — the state machine.
  `Action` is what the *loop* must do: `ActNone`, `ActSubmit{line}`,
  `ActInterrupt`, `ActEOF`. The editor never performs an action itself.
  - **DRY rationale:** every behaviour the issue asks for — history walk, prefix
    search, suggestion accept — is a case here, so all of them are table tests
    over key sequences with no terminal.

- **Suggestion(e Editor, h History) string** — the grey tail: the most recent
  history entry with `Line` as a prefix, minus the typed part. Pure; `""` when
  nothing matches or the cursor is not at end.

- **RenderLine(e Editor, sug string, opt RenderOpts) string** — prompt + typed
  text + grey suggestion + cursor placement, as one frame. Pure — the loop writes
  the string, the editor never touches a writer.
  - **Why a whole frame:** partial-update cursor arithmetic is what `#2` is
    deleting. Rendering the line and letting the terminal redraw it is what makes
    "indicator in place of the prompt" fall out for free.

- **History** — `Prefix(p string) []string` (newest first, deduped) and
  `Add(word string)`. An interface, because the persistent implementation is
  `#3`'s store and must not be re-invented here (`#15` says `/history` reads the
  store, not a private file).
  - **`memHistory` ships now**, seeded per session. **Cross-session persistence
    is `#3`'s to satisfy through this seam** — see Revisions for the Done-when
    change that follows.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `rawTerminal` | `cmd/define/rawterm.go` | new | `x/term` MakeRaw/Restore |
| `keyReader` | `cmd/define/rawterm.go` | new | an `io.Reader` + `decodeKey` |
| `scriptedKeys` | `cmd/define/rawterm_test.go` | new | a scripted key source |

- **rawTerminal** — enters raw mode and restores it. **Restore must survive every
  exit path**, including panic and SIGINT; a terminal left raw is the worst
  failure this tool can produce, worse than any wrong output.
- **keyReader** — feeds bytes to `decodeKey`, handling short reads. Injected as a
  `<-chan Key` into the loop so cancellation is not blocked behind a read — the
  same shape `#2` uses for lines today.

---

## Chunk 2: Tasks

### Task 1: `decodeKey`

**Files:** create `cmd/define/key.go`; test `cmd/define/key_test.go`

- [ ] **Step 1: Write the failing table test.** Obligations: `a` → rune; `\x1b[A/B/C/D` → Up/Down/Right/Left; `\x1b[H`/`\x1b[F` and `\x1bOH`/`\x1bOF` → Home/End; `\x7f` and `\b` → Backspace; `\x1b[3~` → Delete; `\r` and `\n` → Enter; `\x03` → Interrupt; `\x04` → EOF; `\t` → Tab. A **partial** sequence (`\x1b`, `\x1b[`) → `consumed == 0`. An unknown sequence → `KeyUnknown` with its bytes, never a rune. A multi-byte UTF-8 rune (`é`, `♫`) decodes as one rune — the corpus has non-ASCII headwords.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement**
- [ ] **Step 4: Run, expect PASS; add a fuzz target** asserting `decodeKey` never panics and never reports `consumed > len(buf)` — it is the one function fed arbitrary bytes from outside.
- [ ] **Step 5: Commit** — `#14: decode terminal key sequences`

### Task 2: `Editor` + `Apply` — editing without history

**Files:** create `cmd/define/editor.go`; test `cmd/define/editor_test.go`

- [ ] **Step 1: Write the failing tests.** Obligations: insert at cursor, not just append; Left/Right clamp at the ends; Home/End; Backspace at position 0 is a no-op (not a panic); Delete at end likewise; Enter yields `ActSubmit` with the line and clears it; Ctrl-C yields `ActInterrupt`; Ctrl-D on an **empty** line yields `ActEOF` but on a non-empty line does nothing (that asymmetry is standard and easy to get wrong); `KeyUnknown` changes nothing.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement**
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#14: pure line-editor state machine`

### Task 3: History walk + prefix search

**Files:** create `cmd/define/history.go`; modify `editor.go`; test `history_test.go`

- [ ] **Step 1: Write the failing tests.** Obligations: Up from an empty line walks all entries newest-first; Down walks back and past the newest restores **what was typed before walking started** (losing it is the classic bug); with `sy` typed, Up visits only entries starting `sy`, newest first, and leaves the typed prefix in place; exhausting matches is a no-op, not a wrap; duplicates appear once.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** `histCursor` holds the saved draft and the walk index.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#14: history walk and prefix search`

### Task 4: Inline autosuggestion

**Files:** modify `editor.go`; test `editor_test.go`

- [ ] **Step 1: Write the failing tests.** Obligations: typing `sy` with `sycophantic` in history suggests `cophantic`; Right/End at end of line accepts it; Left, Backspace and any other key leave it unaccepted; **Enter submits only what was typed, never the suggestion** (the single most damaging possible bug here — you would look up a word you did not ask for); no suggestion when the cursor is mid-line; no suggestion when nothing matches.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement**
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#14: inline autosuggestion`

### Task 5: `RenderLine`

**Files:** modify `editor.go`; test `render_test.go`

- [ ] **Step 1: Write the failing tests.** Obligations: the suggestion is emitted in grey and the plain-text form contains it exactly once; with `Color:false` **no ANSI at all** (`#2`'s `-no-color` contract, which this must not regress); the cursor is positioned after the typed text, not after the suggestion; a frame ends by clearing to end-of-line so a shortened line leaves no residue.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement**
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#14: render the editor line`

### Task 6: Raw mode, and deleting the workaround

**Files:** create `cmd/define/rawterm.go`; modify `repl.go`, `main.go`; test `repl_test.go`

- [ ] **Step 1: Write the failing tests.** Obligations: the loop drives `Apply` from a scripted key channel and produces the same lookups `#2`'s line loop did; a bare Enter still replays; `terminalUI` still governs every byte of UI; piped stdin still uses the **line** path unchanged — `echo word | define` must not enter raw mode.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** `MakeRaw` on entry, `Restore` deferred **and** on the interrupt path. Delete `eraseLineAndStepBack` and `skipPrompt`; the indicator becomes a rendered frame.
- [ ] **Step 4: Run, expect PASS.** `go test -race` too — there is a reader goroutine.
- [ ] **Step 5: Manual pty check — the things no test reaches**

```sh
make build && ./bin/define
# type "syc" → grey "ophantic" appears; Right accepts; Enter defines
# Up/Down walk; type "sy" then Up → only sy* entries
# press Enter twice DURING playback → no stranded indicator (#2's documented limit, gone)
# ^C → exits, and the terminal is not left raw:  stty -a | grep -q icanon && echo OK
```

- [ ] **Step 6: Update `README.md`, `atlas/define.md`; revise `#2`'s Limits entry**, which documents a cooked-mode limitation that no longer exists.
- [ ] **Step 7: Commit, then `sdlc close --issue 14 --verified '<evidence>'`**

---

## Risks

- **A terminal left in raw mode** is the worst outcome here — the user's shell
  becomes unusable. `Restore` runs from `defer`, from the interrupt path, and is
  asserted by the pty check. Everything else in this issue is recoverable; this
  is not.
- **Enter submitting the suggestion** would look up a word the user never typed.
  Pinned by an explicit test rather than left to the render logic.
- **A partial escape sequence read across two reads** decodes as `Escape` + junk
  if `consumed == 0` is not honoured. The fuzz target and the partial-sequence
  cases cover it.
- **`#2`'s regressions returning:** `-no-color` emitting ANSI, UI gated on stdin
  alone, piped input entering raw mode. All three have existing tests; they must
  stay green rather than be rewritten.
