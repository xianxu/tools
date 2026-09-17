# Read-along passage Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Paste a sentence or paragraph into `define`, click or drag the words you do not understand, and get one explanation of the passage with those spans called out — after which the marks clear and the words you asked about show as deck words.

**Architecture:** One review boundary (the milestones were collapsed — see Revisions). The passage is a **RECORD**: it is written to the buffer and scrolls away like a definition or an answer, with its deck colour baked in at write time. Marks are PAINT-TIME over those immutable bytes, the mechanism `markClickable` already uses for the clickable underline. It was footer chrome first, to make "the asked-about words turn green afterwards" possible; when that collided with "the passage should scroll like everything else", the green re-render was the requirement dropped. Every new decision surface is a pure function over data (`pasteScanner`, `passage`, `markSet`, `renderPassagePrompt`); the IO shell is small edits to existing seams (terminal mode, key decoder, the buffer write, `Capturer`).

**Tech Stack:** Go, `package main` in `cmd/define/`. Terminal: raw mode + SGR. Model: `internal/llm` with `llmtest.Fake` (httptest, Anthropic wire protocol) and `llmtest.AssertGolden`. Store: `cmd/define/store` (YAML).

**Issue:** `workshop/issues/000067-read-along-passage.md` — read its Spec, its "Three collisions", and its "Survey findings" section before starting. Every decision below is recorded there with rationale; this plan does not re-argue them.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `pasteScanner` | `cmd/define/paste.go` | new |
| `sanitisePasteBody` | `cmd/define/paste.go` | new |
| `indexPasteAbandon` | `cmd/define/paste.go` | new |
| `pasteLineRunes` | `cmd/define/paste.go` | new |
| `keyDecoder` | `cmd/define/key.go` | new |
| `passage` | `cmd/define/passage.go` | new |
| `wordAtCell` | `cmd/define/passage.go` | new |
| `markSet` | `cmd/define/marks.go` | new |
| `renderPassagePrompt` | `cmd/define/passageprompt.go` | new |
| `escapeReservedBrackets` | `cmd/define/passageprompt.go` | new |
| `paintMarks` | `cmd/define/passage.go` | new |
| `markCellRanges` | `cmd/define/passage.go` | new |
| `passageRegions` | `cmd/define/passage.go` | new |
| `regionPlaysAudio` | `cmd/define/replraw.go` | new |
| `regionUnderlines` | `cmd/define/replraw.go` | new |
| `sittingKeyHandling` | `cmd/define/play_loop.go` | new |
| `lineState` | `cmd/define/repl.go` | new |
| `parseREPLLine` | `cmd/define/repl.go` | modified |
| `askSystem` | `cmd/define/askctx.go` | modified |

- **pasteScanner** — the bracketed-paste state machine: bytes in, either "still accumulating" or one finished paste.
  - **Relationships:** 1:1 with the decode loop; holds no reference to anything.
  - **DRY rationale:** First occurrence of an explicitly enumerated `(state, event) -> (state, effect)` machine on the *input* path. `selectionStep` is the same shape on the pointer path and is the model to copy (ARCH-ORDER).
  - **Future extensions:** Other DCS/OSC sequences that carry a payload rather than a meaning.

- **passage** — the pasted text plus everything derived from it: its lines, its word runs per line, and its rendered form for a given width, mark set and vocabulary.
  - **Relationships:** 1:1 with a session (at most one live passage); 1:N with word runs; referenced by `markSet` only by index, never by pointer.
  - **DRY rationale:** Reuses `wordRuns` (`highlight.go:33`) for tokenising and `highlightSpans` for deck colour, rather than growing a second tokeniser. The divergence this prevents is the one the atlas already names: *"If two paths take user input, they route through one parser or they will drift."*
  - **Future extensions:** A source field (file, URL, clipboard) if `/read` is ever added; a language field if a passage is ever not in the deck's language.

- **wordAtCell** — given a passage line and a display column, the word run under it. The bridge the survey found missing: `selectionCell` is per-column and knows nothing about words; `wordRuns` is per-byte and knows nothing about cells.
  - **Relationships:** pure function, no state.
  - **DRY rationale:** The single definition of "a click snaps to this word". Both the click path and any future keyboard mark path must use it or they will disagree about what a click selected.
  - **Future extensions:** Snap to a phrase when the cell falls inside a multi-word deck key.

- **markSet** — the spans currently marked, as `(line, startByte, endByte)`, ordered by position.
  - **Relationships:** N:1 with `passage`. Owned by the session; cleared wholesale after an ask.
  - **DRY rationale:** One definition of "what is marked", read by the renderer, the prompt builder and the deck admission. Three readers is exactly the count at which a second definition starts to drift.
  - **Future extensions:** A per-mark note or reason, if marks ever survive an ask.

- **renderPassagePrompt** — `passageAsk` (the passage, the marks, and the ordinary `askContext`) → `llm.Request`. Pure, so the golden can snapshot the real request.
  - **Relationships:** 1:1 with a submitted ask.
  - **DRY rationale:** Sits beside `renderAskPrompt` and shares its section helper and headers; it must NOT fork the context blocks, which are the same blocks.
  - **Future extensions:** #64's interaction stage lands here as one more derived field, not a new renderer.

- **escapeReservedBrackets** — turns `[` and `]` in passage text into `&#91;`/`&#93;` so a literal bracket cannot be read as a `[sel]` or `[lang=…]` marker.
  - **DRY rationale:** `askSystem` already states this escape for the *answer* direction. This is the same rule, finally applied in the prompt direction. Note there is no constant pair to reuse: `&#91;`/`&#93;` exist only as prose inside `askSystem`'s raw string literal (`askctx.go:173-174`), and the decode side uses generic `html.UnescapeString` (`language_decode.go:237`). They must be EXTRACTED, which means breaking that raw literal.

### Integration points

The Status column describes the ENTITY, not the file: a mode enable is a new
function in an existing file, so it is `new`. `TestPlanTablesNameEntitiesThatExist`
(`repo_guard_test.go:897`) reads it that way — it exempts `new` rows while a plan
is in progress and checks everything else against the tree.

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `enabledModes` + `enterModes`/`leaveModes` | `cmd/define/rawterm.go` | new | every terminal mode, derived from one list |
| `KeyPaste` + decode | `cmd/define/key.go` | modified | the byte stream |
| passage-in-the-buffer | `cmd/define/replraw.go` | modified | `liveScreen.WriteRegions` |
| `CaptureMarked` | `cmd/define/capture.go` | new | `store.Store` |

- **enabledModes** — every mode the program asks a terminal for, in TEARDOWN order, each marked with whether the terminal replies in it.
  - **Injected into:** nothing; `enterModes`/`leaveModes` loop it, `restore()` calls the latter, the decoder guard derives its encodings from the `replies` half, and both the in-process and PTY assertions loop it.
  - **Future extensions:** a fourth mode is a row, and every guard picks it up — which is the point. It began as a pair of constants plus a hand-written enable method, and the three hand-written pairs were the duplication BR-13 named.

- **KeyPaste** — a new `KeyKind` whose `Raw` carries the whole pasted text, produced only by `pasteScanner`.
  - **Injected into:** `Apply` (which must NOT insert it as runes) and `runEditor` (which routes it to the passage).
  - **Future extensions:** A paste into the *line* rather than the passage, if that is ever wanted, is a second case on the same key.

- **passage-in-the-buffer** — the passage written to the record with one clickable `Region` per word.
  - **Injected into:** `console.view.WriteRegions`, unchanged in signature. Marks are a paint-time overlay handed down by `SetPassage` on every draw, together with the live passage's buffer range — one call, because both answer "is the live passage reachable here?".
  - **Future extensions:** #69's header, which turns the whole screen into a screen program and would let a passage be revisited rather than scrolled past.
  - **Superseded:** this row said `passage-as-footer` and described `Draw`. See Revisions.

- **CaptureMarked** — admission of a marked word into the deck.
  - **Injected into:** nothing new; it is a fifth verb on the existing `Capturer` interface, and it MUST reuse `decideCapture` and the existing `Upsert` → `vocab.Add` ordering rather than adding a second appender (`capture.go:47` states why).
  - **Future extensions:** Carrying the source sentence, which M5 does.

**Test surface.** Every pure entity above gets a colocated `_test.go` that runs with no IO and no fakes. The integration points are exercised through the existing doubles: `recordDisplay` (`editorloop_test.go:54`) for the screen, `scriptedPointer`/`completedPointerClick` (`editorloop_test.go:780,798`) for clicks, `llmtest.Fake` for the wire, and a real `store.YAML` in `t.TempDir()` for the deck (the `askRig` pattern, `askrun_test.go:25`).

**ARCH-CONSTRAINTS.** Interaction path: keystroke and UI response. A passage is bounded to **1000 runes** (operator decision: a sentence to a paragraph). Runes, not bytes — a byte cap would refuse a CJK paragraph at roughly a third of its length, on exactly the decks `/lang` exists for. A paste beyond the cap is refused with a message rather than truncated, because a silently half-taken passage would produce an answer about text the reader cannot see; the refusal still CONSUMES its bytes, so `readInput`'s buffer cannot grow with the input. Model calls: exactly one per ask, never one per mark. The existing `maxSelectionCells = 262144` and `maxSelectionSource = 1<<20` bounds are unchanged and sit far above the passage cap.

**ARCH-FUNERAL.** The passage and its marks are in-memory, die with the session, and are replaced wholesale by the next paste — no removal path needed. The durable residue is a deck word (bounded by the deck, which already has its own lifecycle) and at most one `store.Item` per marked word (bounded by the existing `ItemCap = 4`). Nothing else is created.

**ARCH-SECURE.** The passage is **untrusted input**: it is arbitrary text from the user's clipboard and it flows into a model prompt. Two boundaries: `escapeReservedBrackets` at the prompt boundary (so pasted text cannot forge a `[sel]` or `[lang=…]` marker), and the existing `oneLine`/`sanitiseItem` on the store boundary (so a pasted control character cannot reach a YAML record). No credentials are touched.

---

> **These task bodies are PRE-IMPLEMENTATION intent.** Several were superseded
> while the work ran — most of all by the footer-to-buffer reversal (see
> Revisions) — and the illustrative Go blocks have been removed because every test
> name in them was a planned name, not a written one: 23 of them ended up in no
> file. The record of what SHIPPED, with the real test beside each obligation, is
> the issue's `## Done when` audit, which `TestPlanCitesTestsThatExist` reads.
>
> What is still load-bearing here is the *reasoning*: the Read-firsts, the design
> arguments, and the corrections recorded where they were made.

## Chunk 1: M1 — bracketed paste

**Why first:** it is the only genuinely missing mechanism, it is testable with no UI, and it fixes a standing bug on its own — today a pasted newline submits the line mid-paste (`key.go:83` → `editor.go:96`).

**Read first:** `key.go:66-107` (the `(Key, int)` protocol, where `used == 0` means "need more bytes"), `key.go:330-352` (`decodeX10Mouse` — the precedent for a sequence consumed whole or not at all), and **`selection_input.go:169-178`, which is the contract the scanner must satisfy:**


**The caller RE-PRESENTS the whole buffer after a short read.** It advances only when bytes are consumed. A decoder that also accumulates internally therefore sees every byte twice. A first draft of this plan got this wrong and a plan review caught it by building the code and running it: split at a read boundary it produced `"hello\x1b[200~hello world"`, and a 900-byte paste arriving in `readInput`'s real 256-byte chunks (`selection_input.go:161`) was **refused** because the internal buffer grew quadratically past the cap. With a 256-byte chunk and a 1000-rune cap, multi-read is the *normal* path, not the edge.

So: **the scanner accumulates nothing.** It is a function of the buffer it is shown, and its only state is the one thing the buffer cannot tell it — whether it is discarding an oversize paste.

### Task 1.1: The paste scanner

**Files:**
- Create: `cmd/define/paste.go`, `cmd/define/paste_test.go`

- [x] **Step 1: Write the failing tests**


- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ -run TestPasteScanner -v`
Expected: FAIL — `undefined: pasteScanner`

- [x] **Step 3: Implement**


Add `KeyPaste` and `KeyPasteRefused` to the `KeyKind` block (`key.go:9-51`).

- [x] **Step 4: Run to verify they pass**

Run: `go test ./cmd/define/ -run TestPasteScanner -v`
Expected: PASS (5 tests)

- [x] **Step 5: Commit**

### Task 1.2: Hook the scanner into `decodeKey`, and rewrite the pinned assertion

**Files:**
- Modify: `cmd/define/key.go:72` (`decodeKey`)
- Modify: `cmd/define/key_test.go:71` — **this line pins `ESC[200~` as `KeyUnknown`/6 and must be deliberately rewritten, not deleted**

> **Read `workshop/lessons.md:114` § "Deleting a test needs the same evidence as writing one" first.** That assertion is not stale scaffolding; it encodes the old behaviour. Replace it with one asserting the new behaviour (a lone start marker now consumes **0** — it is a prefix), and run the surrounding table to confirm nothing else moved.

> **`decodeKey` must become stateful, and that has two costs the first draft missed.** There are **39 call sites across 6 files** (34 in `key_test.go`). Prefer keeping a package-level `decodeKey(buf)` wrapper that allocates a fresh decoder, and give `readInput` a long-lived one — that leaves 38 call sites untouched and is equally correct, because only the streaming caller needs continuity. **`FuzzDecodeKey` and `FuzzDecodeKeyNeverLeaksEscapeTails` (`key_test.go:128,147`) must build a fresh decoder per iteration** or paste state leaks between inputs; the seed corpus already contains `"\x1b[200~"`.

- [x] **Step 1: Write the failing tests**


- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ -run 'TestDecodeKeyTakesAPaste|TestAPastedNewline|TestAFreshDecoder' -v`
Expected: FAIL — `undefined: KeyPaste`

- [x] **Step 3: Implement — and state the hook rule, because it is not "on ESC"**

**The scanner takes EVERY byte while draining, and only `0x1b` otherwise.** Hooking it on `0x1b` alone makes the drain unreachable: `draining` is entered while the buffer head is ordinary text mid-paste, so `decodeKey` would never consult it again and the rest of an oversize paste would arrive as `KeyRune` — the exact failure the drain exists to prevent. Every test above starts at an ESC, which is why none of them catches it.


Add the test that would have caught it:


- [x] **Step 4: Run — naming the tests explicitly, because a pattern that looks right can select nothing**

Run: `go test ./cmd/define/ -run 'Key|Paste|Decode' -v` then
`go test ./cmd/define/ -run 'TestEveryEnabledInputModeIsDecoded|FuzzDecodeKey' -v`
Expected: PASS. (Verify the selection with `go test ./cmd/define/ -list 'Key|Paste|Decode'` — `-run 'Key|Fuzz'` does **not** match `TestEveryEnabledInputModeIsDecoded`.)

- [x] **Step 5: Commit**

### Task 1.2b: The paste body is UNTRUSTED — name the boundary (ARCH-SECURE)

**Files:** Modify `cmd/define/paste.go`; create the fuzz target in `cmd/define/paste_test.go`

The first draft's ARCH-SECURE note named the prompt and store boundaries and stopped at the terminal frame. Three degenerate inputs were unhandled, and the third is the dangerous one:

1. **A paste that never closes.** Under the cap, `scan` returns 0 forever, `readInput` never advances `buf` (`selection_input.go:174`), and **the editor goes deaf** — including to keys typed afterwards, which join the same buffer and are re-scanned. Bound it: the wait is bounded by the cap, so a never-closed paste blocks input until `maxPasteRunes` accumulates and the drain takes over. That is a real, bounded degradation; **state it as a known limit with a test**, rather than leaving it to be discovered.
2. **An embedded `ESC[201~` in the payload** ends the paste early and delivers the remainder as live keys — including `\r`, which submits. Terminals are expected to strip it; the clipboard is the user's own, so the threat model is low. **Decide it explicitly** (the closer wins; the remainder is ordinary input) rather than inheriting it.
3. **Escape sequences in the body reach the SCREEN, which passes producer SGR through by construction** (`selection_frame.go:236-245`). A pasted `\x1b[31m` would recolour the text and can defeat any decoration layered over it, because a mark re-asserts over *known* producer SGR, not over arbitrary injected state.

**So `newPassage` is the parse boundary: untrusted bytes in, a typed `passage` out.** It strips escape sequences (`escapeLen`, `render.go:582` — do not write a second escape grammar) and non-newline control runes, exactly as `oneLine` (`store/item.go:200`) does at the store boundary. Invalid state becomes unrepresentable rather than checked downstream.

- [x] **Step 1: Write the failing tests**


- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ -run 'PastedEscape|Unterminated|EmbeddedCloser' -v`
Expected: FAIL

- [x] **Step 3: Implement**
- [x] **Step 4: Run**

Run: `go test ./cmd/define/ -run 'Paste|Passage' -v` then `go test ./cmd/define/ -fuzz FuzzPasteScannerAcrossCalls -fuzztime 30s`
Expected: PASS, no crashers

- [x] **Step 5: Commit**

### Task 1.3: Enable mode 2004 — and widen the guard that should have covered it

**Files:**
- Modify: `cmd/define/rawterm.go:130-160`, `cmd/define/replraw.go:88`
- Modify: `cmd/define/key_test.go:400-420` (`TestEveryEnabledInputModeIsDecoded`)

> **A plan review corrected this plan's claim here.** The first draft said enabling 2004 without decoding it would fail `TestEveryEnabledInputModeIsDecoded` "by design". It would not: that test derives its modes by regex over **`mouseOn` only** (`key_test.go:417`), so a separate `pasteOn` constant is invisible to it. The plan would have enabled a mode outside the one guard written to prevent exactly that. **Widening the guard is a step of this task, not a nicety.**

- [x] **Step 1: Write the failing tests** — (a) widen the guard's source to `mouseOn + pasteOn` and add a `"2004"` row to its `replies` table; (b) assert `restore()` emits paste-off **before** raw mode ends, in the same ordered teardown as mouse reporting; (c) model the "flag set only on a successful write" rule on `TestEnterDoesNotClaimAStateItCouldNotWrite` (`rawterm_test.go:161`).
- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ -run 'TestEveryEnabledInputModeIsDecoded|TestEnterDoesNotClaim|TestLeaveModesIsIdempotent|Paste' -v`
Expected: FAIL

- [x] **Step 3: Implement** — `pasteOn = "\x1b[?2004h"` / `pasteOff = "\x1b[?2004l"` beside `mouseOn`/`mouseOff`, as a row in `enabledModes`; `enterModes`/`leaveModes` loop that list and record each mode only on a successful write; `newConsole` calls `enterModes`. (Round 6 replaced the three hand-written pairs this step originally described — see Revisions.)
- [x] **Step 4: Run** the same selection; Expected: PASS
- [x] **Step 5: Commit**

### Task 1.4: Route `KeyPaste` through the editor

**Files:** Modify `cmd/define/editor.go:52-112` (`Apply`), `cmd/define/replraw.go:511-541` (`runEditor`); test `cmd/define/editorloop_test.go`

> **Decide explicitly:** a `KeyPaste` reaches `pointerRouter.route` (`selection_input.go:28`), which for a non-pointer, non-`KeyUnknown` key calls `cancelPointerInput(l, k, true)` — **cancelling a live drag**. That is almost certainly right (a paste replaces the passage, so a drag over the old one is meaningless), but it must be a decision with a test, not an inherited side effect.

- [x] **Step 1: Write the failing tests** — a `KeyPaste` leaves the line unchanged and does not submit; a `KeyPasteRefused` prints a message naming the limit; a paste during a live drag cancels the drag.
- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ -run 'EditorLoop|Paste' -v`
Expected: FAIL

- [x] **Step 3: Implement** — `Apply` gets explicit `case KeyPaste, KeyPasteRefused: return e, ActNone` (explicit, not fall-through, so the intent is readable); `runEditor` intercepts both **before** `Apply`, where `viewportGesture` and `KeyClick` are already intercepted.
- [x] **Step 4: Run** the same selection; Expected: PASS
- [x] **Step 5: Commit**

### Task 1.5: Milestone close

- [x] Atlas (`atlas/define.md`, under *The line editor (raw mode)*): mode 2004 is enabled; a paste is ONE key and **why** (the 256-key drop-newest channel); the rune cap and its refusal; and that the scanner accumulates nothing because `readInput` re-presents.
- [x] `sdlc milestone-close --issue 67 --milestone M1`

---

## Chunk 2: M2 — the passage on screen

**Read first:** `screen.go:920` (`Draw`), `screen.go:226-262` (`FooterRowAt` — its doc says a dropped row must not resolve because *"inventing an entry for it would mark a word that is not on screen"*, and its `(entry, offset)` return exists so a caller can correct a column on a **continuation row**: *"column 4 of a continuation is column cols+4 of the entry"*), `screen.go:1146` (`fitFooter`), `screen.go:74-77` (the append-only buffer, which says in as many words that *"a surface with marks that change colour has to live in the footer instead"*), `highlight.go:33-160` (`wordRuns`, `phraseGap`, `highlightSpans`).

### Task 2.1: `passage` and `wordAtCell`

**Files:** Create `cmd/define/passage.go`, `cmd/define/passage_test.go`

- [x] **Step 1: Write the failing tests**


- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ -run 'TestPassage|TestWordAtCell' -v`
Expected: FAIL — `undefined: newPassage`

- [x] **Step 3: Implement** — `passage` holds the raw text, its lines and `[][]wordRun`, with `runs(line) []wordRun` and `text(wordRun) string`. `wordAtCell` maps a display column to a byte offset using `visibleIndex` (`render.go:520`) and `nextDisplayUnit` (`display_unit.go:8`) — **do not write new cell arithmetic**.

> **THE COORDINATE MAPPING, stated once — three spaces, two conversions.**
> A passage has **logical lines** (what was pasted, split on `\n`). Each becomes **one footer entry**. The screen wraps each entry into **frame rows**. So:
>
> | space | unit | who owns it |
> |---|---|---|
> | passage | logical line + byte offset | `passage`, `wordRuns` |
> | footer | entry index + row offset | `FooterRowAt` (`screen.go:247`) |
> | frame | physical row + display cell | `selectionFrame`, `highlightRow` |
>
> **Forward** (a click): frame row → `FooterRowAt` → `(entry, offset)` → `wordAtCell(p, entry, col + offset*cols)`. **Inverse** (painting marks): a `mark`'s byte offsets → per-frame-row `[]cellRange` for the widened `highlightRow`. **The inverse was named nowhere in the first draft and every Chunk 3 test used a single-row frame** — build it here, beside `wordAtCell`, as its stated inverse, and give Tasks 3.2/3.3 at least one **multi-row** frame each.
>
> **Correction:** the first draft said to reuse `phraseGap` to stop a wrapped `hot\n  dog` forming a phrase. That was wrong, and the gate caught it. `phraseGap` (`highlight.go:95-105`) rejects a gap containing a newline — but under logical-line footer entries a display wrap inserts **no newline**, so `phraseGap` does not guard it at all. Deck-phrase highlighting runs on the **logical line**, before the screen wraps it, which is the correct place and needs no new rule. The issue's "reuse `phraseGap`" instruction is answered by this paragraph: the helper is *already* doing its job one layer up, and adding a second wrap-aware rule would be the duplication ARCH-DRY warns about.

> **There is no spans→styled-string helper to reuse.** `highlightSpans` returns `[]span`, and both existing consumers open-code the loop (`editor.go:206-213`, `highlightwriter.go:141`), each with the `inputOn` re-open hazard the atlas names. Extract **one** helper here and have the passage use it; a fourth open-coded loop is the thing ARCH-DRY is for.

- [x] **Step 4: Run** `go test ./cmd/define/ -run 'TestPassage|TestWordAtCell|Highlight' -v`; Expected: PASS
- [x] **Step 5: Commit**

### Task 2.2: Write the passage into the buffer, with a region per word

**Files:**
- Modify: `cmd/define/replraw.go:398-410` (the `draw()` closure), **and `replraw.go:506` and `replraw.go:567`**
- Modify: `cmd/define/session.go` — the session gains the passage and its marks
- Test: `cmd/define/editorloop_test.go`

> **The first draft covered one of three `Draw` call sites.** `replraw.go:506` and `:567` both call `view.Draw("", nil)` — a **nil footer**. `:567` fires on every submit, so the passage would vanish for the entire lookup/answer/playback window: exactly the interval the feature exists for. Either route all three through one helper that supplies the current footer, or make the footer the screen's own state rather than a `Draw` argument. **Route them through one helper** — three callers each remembering to pass the passage is the same shape of bug as three loops each deciding what a line means.

- [x] **Step 1: Write the failing tests**


- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ -run 'ThePassageRenders|ThePassageSurvives' -v`
Expected: FAIL

- [x] **Step 3: Implement** — the passage is written to the BUFFER with one Region per word (superseded: this step said footer entries). Colour them with `deckVocabulary(d)` (**`vocab.go:266`**, not `vocabularyFor`) — `vocab.go:255` says why: *"the same set WITHOUT the colour condition"*, because a word is clickable whether or not it is coloured, and `vocabularyFor` (`vocab.go:248`) returns nil when `!opt.color`.
- [x] **Step 4: Run** `go test ./cmd/define/ -run 'EditorLoop|Passage' -v`; Expected: PASS
- [x] **Step 5: Commit**

### Task 2.3: A passage that does not fit

**Files:** Modify `cmd/define/replraw.go`; test `cmd/define/editorloop_test.go`

- [x] **Step 1: Write the failing tests** — superseded by the footer-to-buffer reversal: a passage in the buffer simply scrolls, and a click outside the live range is refused by `ownsBufferLine`.
- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ -run 'PassageTooTall|FooterRow' -v`
Expected: FAIL

- [x] **Step 3: Implement**
- [x] **Step 4: Run** the same selection; Expected: PASS
- [x] **Step 5: Commit**

### Task 2.4: Milestone close

- [x] Atlas: a new section *Read-along* — the passage as a RECORD in the buffer, and the reversal that got there.
- [x] `sdlc milestone-close --issue 67 --milestone M2`

---

## Chunk 3: M3 — marks

**Read first:** `selection.go` (whole file), `selection_frame.go:208-261` (`highlightRow` — the re-assert-after-every-foreign-SGR discipline), `screen.go:512-524` (`selectionLayout.paint`, the single composition point), `language_row.go:8-74`.

### The decision this chunk rests on: the passage is a SURFACE, not a set of regions

The first draft of this plan proposed *both* a new `RegionKind` and a third `selectionEffect` for the same job — click-to-mark — without saying which owned the gesture. A plan review caught it. Resolved as follows, and the resolution removes work rather than adding it:

**No new `RegionKind`** — *and this was REVERSED.* It held while the passage was footer chrome: every word offers marking, so a per-span registry carried no information. Moving the passage into the BUFFER made the click map the only way to reach its content, so `RegionPassageWord` shipped. See Revisions.

Avoiding a new kind also avoids four obligations the review enumerated, none of which buys anything here:

- `RegionKind.String()` (`render.go:293`), or `TestEveryRegionKindIsNamed` (`render_test.go:642`) fails;
- `RegionKind.identifier()` (`render.go:312`);
- an atlas mention of the kind's **Go identifier**, because `TestAtlasDescribesEveryRegionKind` (`doc_sync_test.go:395-412`) greps `atlas/define.md` for it — M3 could not otherwise close;
- and the hard one: `TestEveryRegionKindIsActionableThroughTheSharedRegistry` (`editorloop_test.go:985`) drives `playRegion(ctx, d, opt, Region, entry, stdout, stderr)` (`replraw.go:707`) directly. That signature carries no session, no passage and no `markSet`, so a "mark, do not play" kind **cannot** produce an observable effect through it without widening a shared registry for one caller.

It is also right visually: `markClickable` underlines regions, and underlining every word of a passage would be noise.

**Click needs no new channel either** — though what it carries changed with the reversal: `pointerClick` now carries the absolute buffer `line`, because a `Region`'s own `Line` is relative to its render and cannot say which passage it belongs to.

**Only the DRAG needs a new effect.** A drag currently always yields `selectionCopy` (`selection.go:72`). On passage rows it must mark instead. That is one new `selectionEffect`, keyed on whether the gesture's ANCHOR row is a passage row — decided in `pointerLocked`, where the frame is already in hand.

So: one new effect, no new region kind, no change to `playRegion`, no change to the actionability guards.

### Task 3.1: `markSet`

**Files:**
- Create: `cmd/define/marks.go`, `cmd/define/marks_test.go`

- [x] **Step 1: Write the failing tests**


- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ -run 'TestMarkSet|TestAClickMark|TestADragAcross' -v`
Expected: FAIL — `undefined: markSet`

- [x] **Step 3: Implement** — `mark{line, start, end int}` (byte offsets into that passage line, matching `wordRuns`' units), `markSet` with `toggle`, `ordered`, `empty`, `clear`. `marksForDrag` converts two cells to per-line marks, snapping each end outward to whole words via `wordAtCell` — a drag that starts mid-word marks the whole word, because the unit everywhere else is a word.

- [x] **Step 4: Run to verify they pass**

Run: `go test ./cmd/define/ -run 'TestMarkSet|TestAClickMark|TestADragAcross' -v`
Expected: PASS (3 tests)

- [x] **Step 5: Commit**

### Task 3.2: ~~Widen `highlightRow` from one range to a set~~ — SUPERSEDED

**Not done, and not to be done.** The steps that were here described changing
`highlightRow`'s signature to `(row, ranges, on)` for a paint-time mark layer over
the FOOTER. The footer-to-buffer reversal (see Revisions) removed the consumer:
marks paint over buffer bytes through `paintMarks`, and `selection_frame.go` is
untouched by this window.

The steps are DELETED rather than left ticked or unticked. Ticked, they claimed a
signature change that never happened and named three tests that exist in no file —
a plan is the one artifact a reader trusts to describe the design. Unticked, they
would switch off `TestPlanTablesNameEntitiesThatExist`, which exempts `new` rows
while any box remains open. Neither state is honest, so there is no box.

### Task 3.3: Paint marks, with precedence

**Files:**
- Modify: `cmd/define/screen.go:512-524` (`selectionLayout.paint`)
- Modify: `cmd/define/selection_frame.go` — the mark colour pair lives **here**, beside the `\x1b[7m` it parallels, not in `language_style.go` (that file is the bilingual row-tint module; a mark is a selection-frame concern)
- Test: `cmd/define/selection_frame_test.go`, `cmd/define/screen_test.go`

Three rules, one test each:

- [x] **Step 1: Write the failing tests**


- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ -run 'AMarkedDeckWord|ALiveDrag|RowTintIsNotInjected' -v`
Expected: FAIL

- [x] **Step 3: Implement** — `selectionLayout.paint`'s condition widens from *"is there a live gesture"* to *"has this row anything to paint"*: marks paint whether or not a gesture is active, which is required because marks persist between gestures.

- [x] **Step 4: Run**

Run: `go test ./cmd/define/ -run 'Selection|Screen|Language' -v`
Expected: PASS

- [x] **Step 5: Commit**

### Task 3.4: Click and drag produce marks

**Files:**
- Modify: `cmd/define/selection.go` (one new `selectionEffect`, for the DRAG only)
- Modify: `cmd/define/selection_screen.go:37-89` (`pointerLocked` — key the effect on whether the anchor row is a passage row)
- Modify: `cmd/define/replraw.go:422-438` (`clicked` — one new case)
- Test: `cmd/define/editorloop_test.go`, `cmd/define/selection_test.go`

- [x] **Step 1: Write the failing tests**


- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ -run 'ThePassageMarks|UnmarksIt|OutsideThePassage' -v`
Expected: FAIL

- [x] **Step 3: Implement** — in `pointerLocked`, a drag whose anchor row is a passage row AND inside the live buffer range returns the mark effect instead of `selectionCopy`; a click carries the absolute buffer line and `passageSpanAt` resolves it.

- [x] **Step 4: Run**

Run: `go test ./cmd/define/ -run 'Selection|EditorLoop|Pointer' -v`
Expected: PASS

- [x] **Step 5: Commit**

### Task 3.5: Milestone close

- [x] Atlas: the mark treatment, the three precedence rules, and **why the passage is a surface rather than a set of regions** — the reasoning above is the kind that gets re-litigated if it is not written down.
- [x] `sdlc milestone-close --issue 67 --milestone M3`

---

## Chunk 4: M4 — the ask

**Read first:** `askctx.go` (whole file), `ask.go:13-23,80-96,146-221`, `repl.go:68-137`, `route_test.go:16` (`TestConsoleDecisionTable`), `testdata/golden/ask-prompt.txt`.

### Task 4.1: The prompt, with its own task name

**Files:**
- Create: `cmd/define/passageprompt.go`, `cmd/define/passageprompt_test.go`
- Create: `cmd/define/testdata/golden/passage-prompt.txt` (first generation via `-update`)
- Modify: `cmd/define/askctx.go:153` — **introduce** `escLeft = "&#91;"` / `escRight = "&#93;"` as constants and interpolate them into `askSystem`. They exist today only as prose inside its raw string literal (`askctx.go:173-174`), so this is an extraction, not a reuse.

> A **new `Task`** (`"passage-question"`) keys a new golden and a new cassette, leaving `ask-prompt.txt` untouched. Reuse `renderAskPrompt`'s `section` helper and its five header constants — the context blocks are the same blocks and must not fork (ARCH-DRY).

- [x] **Step 1: Write the failing tests**


- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ -run 'PassagePrompt|SecondOccurrence|LiteralBracket|NoMarksStillRenders' -v`
Expected: FAIL — `undefined: renderPassagePrompt`

- [x] **Step 3: Implement.** Generate the golden once: `go test ./cmd/define/ -run TestRenderPassagePrompt -update`, then **read the generated file** before committing it — a golden accepted unread is a snapshot of whatever the code did.

- [x] **Step 4: Run**

Run: `go test ./cmd/define/ -run 'PassagePrompt|Ask' -v`
Expected: PASS, and `ask-prompt.txt` unchanged (`git diff --exit-code cmd/define/testdata/golden/ask-prompt.txt`)

- [x] **Step 5: Commit**

### Task 4.2: `parseREPLLine` learns about marks

**Files:**
- Modify: `cmd/define/repl.go:68` (`parseREPLLine`)
- Modify: **all three non-test callers** — `main.go:750`, `repl.go:346`, `replraw.go:546`
- Modify: the test call sites in `route_test.go`, **`repl_test.go:58`**, `askroute_test.go`, `commandloop_test.go`, `command_test.go` — `repl_test.go` was omitted from the first draft's list and the signature change breaks its build
- Test: **the mark-me nudge row belongs in `repl_test.go`**, which field-compares `note`; `TestConsoleDecisionTable` (`route_test.go:16`) reduces to one of four outcome strings through the real dictionary and cannot see a note
- Test: `cmd/define/route_test.go:16` (`TestConsoleDecisionTable` — **this file, not `repl_test.go`**)

> Replace the `hasCurrent bool` parameter with a single session-state value. **Do not add a second boolean** — two bools side by side encode a precedence nobody declared, which is the consolidation `session` itself was created for. Expect a wide, mechanical diff across the call sites; that breadth is the reason to do it as one change rather than adding a parameter now and consolidating later.

- [x] **Step 1: Write the failing tests** — extend `TestConsoleDecisionTable` with five rows:

| input | state | outcome |
|---|---|---|
| blank | marks present | ask about the passage |
| blank | passage, no marks | `cmdNothing` + the mark-me nudge |
| blank | no passage, current word | `cmdReplay` (unchanged) |
| typed question | marks present | that question, passage + marks as context |
| `/lang es` | marks present | `cmdCommand` — `/` still wins in column 1 |

- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ -run TestConsoleDecisionTable -v`
Expected: FAIL

- [x] **Step 3: Implement** — the nudge is a new `note` on `cmdNothing` routed through `nothingSays` (`repl.go:129`), phrased as an **instruction** (*"click or drag what you don't understand, then press return"*), **local, no model call**.

- [x] **Step 4: Run**

Run: `go test ./cmd/define/ -run 'ConsoleDecisionTable|Route|Command' -v`
Expected: PASS

- [x] **Step 5: Commit**

### Task 4.3: Wire the ask, and clear the marks

**Files:**
- Modify: `cmd/define/ask.go:13-23` (a `passage` field on `question`; all six existing construction sites keep their zero value), `cmd/define/ask.go:146-221` (route to the passage renderer), `cmd/define/replraw.go:542-682`
- Modify: `cmd/define/askctx.go` — `askContext` gains an optional passage, so **a plain LOOKUP while a passage is on screen carries it as context** (resolved in-scope at start-plan, and otherwise silently dropped)
- Test: `cmd/define/askrun_test.go`

- [x] **Step 1: Write the failing tests**


> Use a `.sse` capture (`streamCapture = "stream-sample.sse"`, `askrun_test.go:43`) — `llmtest.Fake` 400s a non-streaming `Reply` scripted onto a streaming request, by design.

- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ -run 'SendsOneRequest|MarksInPlace|MarksClear|LookupCarries' -v`
Expected: FAIL

- [x] **Step 3: Implement**
- [x] **Step 4: Run** — `go test ./cmd/define/ -run Ask -v`; Expected: PASS
- [x] **Step 5: Commit**

### Task 4.4: The global default level

**Files:**
- Modify: `cmd/define/askctx.go:153` (`askSystem`)
- Regenerate: `cmd/define/testdata/golden/ask-prompt.txt` **and** `passage-prompt.txt`
- Create: a row in an existing `*_conformance_test.go`

> This **reverses** a stated rule — *"If the learner model is absent, write for a capable adult reader and do not guess at their level."* Global, one statement, both surfaces inherit (operator decision). Two dials in opposite directions: **hold the language level** (college-bound; do not simplify, do not swap a hard word for an easy one), **drop the assumed background** (do not take the ecliptic as known), and **be curious** (volunteer the connecting fact).

- [x] **Step 1: Write the failing conformance test** — assert that an explanation of a hard word **still contains that word** rather than paraphrasing it away. That is the failure mode the persona invites, and a prompt line alone does not defend it.

> **Conformance tests are build-tagged.** They carry `//go:build darwin && conformance` and do not compile into an ordinary run — so without the tag, "run it and watch it fail" reports PASS vacuously. Run them as:
> ```
> go test -tags conformance ./cmd/define/ -run TestDefaultLevelKeepsTheHardWord -v
> ```
> and run them **unsandboxed** — they reach the real dictionary and the live proxy.

- [x] **Step 2: Run to verify it fails**

Run: `go test -tags conformance ./cmd/define/ -run TestDefaultLevelKeepsTheHardWord -v` (unsandboxed)
Expected: FAIL — the current prompt writes for "a capable adult reader"

- [x] **Step 3: Implement** the `askSystem` change.
- [x] **Step 4: Run** the conformance row, then `go test ./cmd/define/ -run 'Ask|Golden' -v`, then regenerate both goldens with `-update` and **read both diffs**.
- [x] **Step 5: Commit**

### Task 4.5: Milestone close

- [x] Atlas: the decision table's new rows; the `[sel]` grammar and its escape; the reversed level default; **and the NOAD-as-context inversion with the route back to the full entry** (Done-when 9 — it has no other home).
- [x] Atlas: **the headword-click shortcut is now scoped.** The atlas states a headword click *"is a shortcut for the bare Enter beside it"*; with marks present Enter asks instead, so the two diverge unless the scoping is written down (one of the issue's two "consequences to carry into the plan").
- [x] `sdlc milestone-close --issue 67 --milestone M4`

---

## Chunk 5: M5 — the words become deck words

**Read first:** `capture.go:11-132,191-205`, `store/event.go:6-140`, `history_store.go:45-75`.

### Task 5.1: `CaptureMarked`

**Files:**
- Modify: `cmd/define/capture.go` — the `Capturer` interface (`:47`), `storeCapturer`, `noopCapturer`
- Modify: `cmd/define/store/event.go` if a new `EventKind` is added — append to `eventKinds` (`:35`) and leave `ReviewEvent.At` **last** (`:122-126`)
- Test: `cmd/define/capture_test.go`

> **One `Upsert`.** The interface doc states why: *"capture is the only thing that records, and a second appender beside it is how that stops being true without anyone noticing."* Reuse `decideCapture` and keep `vocab.Add` (`capture.go:130`) **after** the `Upsert` (`capture.go:123`) — that ordering is what makes "the word turns green" honest rather than optimistic.

- [x] **Step 1: Write the failing tests**


- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/define/ ./cmd/define/store/ -run 'Marked|Capture' -v`
Expected: FAIL

- [x] **Step 3: Implement**
- [x] **Step 4: Run** — `go test ./cmd/define/... -run 'Capture|Event|History' -v`; Expected: PASS
- [x] **Step 5: Commit**

### Task 5.2: The passage re-renders, and the words are green

**Files:** `cmd/define/editorloop_test.go`

- [x] **Step 1: Write the failing test** — the end-to-end row this issue is named for:


- [x] **Step 2: Run to verify they fail**
- [x] **Step 3: Implement** — superseded: the buffer does not re-render, which is why the green re-render was dropped. `vocab.Add` still runs, so the word is green the next time it is rendered anywhere.
- [x] **Step 4: Run** — `go test ./cmd/define/ -run 'PasteMarkAsk|Schedulable' -v`; Expected: PASS
- [x] **Step 5: Commit**

### Task 5.3: Issue close

- [x] Atlas: admission, the event kind, and the passage's lifecycle (replaced by the next paste, dies with the session).
- [x] Confirm every `## Done when` row in the issue has a test naming it.
- [x] Full suite: `go test ./cmd/define/...` — **note the sandbox**: `TestLanguageTintInvocation` and the `language_prompt_paths` rows fail with "operation not permitted" under the Bash sandbox and pass on the host. Re-run unsandboxed before diagnosing (`workshop/lessons.md`).
- [x] Conformance: `go test -tags conformance ./cmd/define/` (unsandboxed).
- [x] `sdlc close --issue 67 --verified '<evidence>'` — let it measure `--actual`; do not hand-type hours.

---

## What this plan does NOT do

Recorded so a reviewer does not read them as omissions (ARCH-PURPOSE — these are separable extensions, not the deferred point of the issue):

- **Click density as a readability measure.** Own issue. It needs the marking data this one produces, so it is a successor.
- **Did-you-mean for one-word misses.** Own issue — it changes the console classifier globally, not this surface.
- **Click-to-copy in the answer area.** Withdrawn by the operator after the survey showed `RegionWord` already claims those clicks.
- **The authentic sentence as practice material.** Moved to **#68** (`use real sentences as cloze material`). It cannot stand alone: distractors come from the deck's banded words and are then vetoed, and — the finding that actually killed it — expository prose routinely *glosses* the word it uses, which is precisely the appositive the author prompt bans. A real sentence needs more judging, not less. #68 also finishes the unused `usage/` cache, which is the same problem one layer down.
- ~~**A new `RegionKind` for passage words.**~~ **Shipped as `RegionPassageWord`.** It was resolved away while the passage was footer chrome, where every word being clickable made a per-span registry carry no information. Moving the passage into the BUFFER reversed that: the click map is how buffer content is reached. It is the first kind the audio registry does not answer for, which forced `regionPlaysAudio` and `regionUnderlines` to be declared rather than assumed.
- **Persisting the passage across restarts.** The passage is transient; the residue (the deck word) is what persists.
- **Relaxing the dictionary-hit admission rule** so non-headword phrases can be learned. Recorded as revisitable once there is usage data.

---

## Revisions

### 2026-09-16 — M1 as shipped

Recorded because a plan a reader trusts must not contradict the tree. Each item is
a deliberate departure, not drift.

- **The parse boundary is `sanitisePasteBody` in `paste.go`, not `newPassage`.**
  Task 1.2b put it in a function M2 creates, so its tests referenced a symbol that
  did not exist yet. The scanner is the better home anyway: the bytes become a
  typed value at the moment they stop being a wire format, and nothing downstream
  can forget to ask.
- **A paste inserts into the line.** The plan had `runEditor` intercept `KeyPaste`
  and leave it without a destination until M2. That would have REGRESSED the case
  that already worked — a pasted word typed itself in fine; only a pasted newline
  misbehaved. `Apply` now takes a paste as one atomic insertion.
- **No `spansToStyled`.** The plan's entity table added one, on the claim that no
  spans-to-styled helper existed. `highlightRegion` (`highlightwriter.go:269`) is
  exactly that and already has seven callers; the row is removed and the passage
  uses it.
- **`TestEveryEnabledMouseModeIsDecoded` is now `TestEveryEnabledInputModeIsDecoded`,**
  since it no longer reads only `mouseOn`.
- **A new rule the plan did not have: a paste is abandoned on a control byte.**
  The M1 boundary review found (C1/BR-1) that an unterminated `ESC[200~`
  permanently deafened the input path — raw mode disables ISIG, so Ctrl-C is
  reachable only as a decoded `KeyInterrupt`, and the program could not be quit
  from the keyboard. The plan named this case as a "known limit" and promised a
  pinning test; neither shipped, and the limit was worse than stated because the
  drain latched. `indexPasteAbandon` ends an open paste at the first byte that
  cannot be paste text, which restores exactly the pre-milestone behaviour.
- **A second departure found in review round 2 (BR-12): the cap is two predicates,
  not one.** `maxPasteRunes` is semantic and can only be judged on COMPLETE text,
  at the closer; `maxPasteBytes` is the memory bound and is judged while the text
  is still arriving. One predicate served both, and since `utf8.RuneCount` counts
  each orphan byte of a split rune as a `RuneError`, a legal 1000-rune CJK paste
  split at the wrong byte counted 1001 and was refused — on exactly the decks the
  rune cap exists for.

### 2026-09-16 — the footer-to-buffer reversal, recorded as a Revision

Reason: the rule BR-24 names has two clauses, and the first pass followed only
one. Ticking the boxes and correcting the tables is the first; **appending the
departure here instead of overwriting the prose** is the second, and the
Architecture paragraph was rewritten in place with no entry.

Delta, stated as a reversal rather than as if it had always been so:

- The passage was designed as **footer chrome**, redrawn every frame, to make
  "marks clear and the asked-about words turn green" possible — `screen.lines` is
  immutable once written. The operator ran it and the passage was welded to the
  prompt forever, under every later lookup. The green re-render was dropped and
  the passage became a **buffer record**.
- That reversed two further decisions in this plan. `highlightRow` was to be
  widened to a set of ranges for a paint-time mark layer; with marks painted over
  buffer bytes by `paintMarks`, it has no consumer and `selection_frame.go` is
  untouched by this window. And `RegionPassageWord`, "resolved away in Chunk 3",
  is shipped — the click map is how buffer content is reached.
- Tasks 2.2 through 2.4 were written against the footer and their titles and steps
  are corrected in place, which is the part that cannot be appended: a task
  description that describes work nobody did is not history, it is a false claim
  about the tree that this repo's guards read.
