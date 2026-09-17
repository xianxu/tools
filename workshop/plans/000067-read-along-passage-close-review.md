# Boundary Review — tools#67 (whole-issue close)

| field | value |
|-------|-------|
| issue | 67 — define: read-along — paste a passage, click or drag what is opaque |
| repo | tools |
| issue file | workshop/issues/000067-read-along-passage.md |
| boundary | whole-issue close |
| milestone | — |
| window | 98f5c779b468ada00c087bde6bd43cca9b0892cc..11a279fc1eee73de979a47d011594e2b401fc90c |
| command | sdlc close --issue 67 |
| reviewer | claude |
| timestamp | 2026-09-16T21:51:08-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The paste layer this window built on top of M1 is genuinely strong — the `pasteExit` enumeration with a guard derived from `numPasteExits`, the two-predicate cap, the `regionPlaysAudio`/`regionUnderlines` split that keeps the actionability guards total, and an atlas section that explains every non-obvious choice including the ones that were reversed. Two confirmed correctness defects block SHIP, both in the click-to-mark path the issue is named for. First, a superseded passage's `Region`s stay clickable and carry only a passage-*relative* `Line`, so after a second paste clicking a word in the earlier passage marks an unrelated word in the current one (measured: `alpha`→`zulu`, `beta`→`yankee`, `epsilon`→`whiskey`) or silently does nothing. Second, a drag produces one mark *per word* rather than the span the Spec decided, so the prompt reads `[sel]at[/sel] [sel]the[/sel] [sel]zenith[/sel] [sel]of[/sel]` and deck admission runs the dictionary on each function word separately — while `marksForDrag`, which implements the decided single-span behaviour and is pinned by three tests, has no production caller at all and produces spans that `markedPassageText` silently drops. Of the eight prior open findings, BR-1 and BR-17 are now mutation-verified fixed; BR-8, BR-11, BR-13, BR-14, BR-15 and BR-16 remain open, four of them because the class fix the finding asked for was not written.

## 1. Strengths

- **`pasteExit` + `TestEveryPasteExitIsExercised` (`cmd/define/paste.go:64-75`, `paste_test.go:445`)** — naming the seven exits and deriving the guard from `numPasteExits` is the right answer to round 4's "the subtest named *draining* never drained". I confirmed it works: adding `&& !s.draining` to the abandon condition (`paste.go:113`) now reddens `TestAnUnterminatedPasteDoesNotSwallowEnterOrInterrupt/after_the_drain_has_started` rather than leaving the suite green.
- **The two-predicate cap (`paste.go:19-28,150`)** — `maxPasteRunes` judged only on complete text at the closer, `maxPasteBytes` as the memory bound while bytes are still arriving, with the `RuneError`-counting reason stated at the site. This is the structural fix BR-12 asked for, not a patch.
- **`regionPlaysAudio` / `regionUnderlines` (`replraw.go:788-817`)** — the audio registry stopped being total, and instead of weakening `TestEveryRegionKindIsActionable` to "does something somewhere", the split is declared and both guards consult it (`editorloop_test.go:996-1055`). Correct handling of a hard constraint the survey predicted.
- **`paintMarks` (`passage.go:221-259`)** — re-asserting `markOn` after every producer SGR and restoring with `sgrOff + style.resume()` is the discipline collision 2 demanded, and the explicit fg/bg pair (rather than `\x1b[7m`) is argued from `sourceBackground` recognising `48` and not `7`.
- **BR-17's fix (`replraw.go:614-620`)** — mutation-verified: removing the `view.Draw("", nil)` before the notice reddens `TestTheRefusalNoticeDoesNotLandInsideThePrompt` with "the loop drew 1 prompts".
- **`atlas/define.md`** — both new sections carry the *reversals* (footer→buffer, the dropped green re-render, the level default) with the reasoning, which is exactly what the atlas is for.

## 2. Critical findings

**C-A. A superseded passage's regions resolve against the current passage — `cmd/define/passage.go:367` + `cmd/define/replraw.go:593-608`.**
`passageRegions` sets `Region.Line` to the *passage-relative* line; `addRegions` (`screen.go:195-198`) keys the map by absolute buffer line but stores the region unmodified. Nothing removes the first passage's regions when a second is pasted, and `passageSpanOf` resolves `r.Line`/`r.Col` against `sess.passage` — whichever passage is current — with no identity check. Measured against HEAD:

```
click on OLD "alpha"   (col  0) -> CURRENT "zulu"    (ok=true)
click on OLD "beta"    (col  6) -> CURRENT "yankee"  (ok=true)
click on OLD "gamma"   (col 11) -> CURRENT ""        (ok=false)   // silent no-op
click on OLD "epsilon" (col 23) -> CURRENT "whiskey" (ok=true)
```
The mark then paints at `sess.passageBase + line`, i.e. on the *new* passage, so the highlight appears where the reader did not click, and the next bare Enter brackets a word they never marked and admits it to the deck. Two pastes in one session is the feature's own use case ("Follow-up means marking only what is new"). ARCH-ORDER: *"an observation is evidence about an exact entity at a point in time, not permanent authority to act on it."*
*Fix sketch:* give a passage an identity (a monotonic generation stamped into the `Region`, or resolve through the absolute buffer line and require `0 <= line-passageBase < lineCount`), and reject a click whose region does not belong to the live passage. Pin it with a two-paste regression test through `runEditor`.

**C-B. A drag marks each word separately, so the decided span never exists — `cmd/define/replraw.go:568-576`, `cmd/define/marks.go:68`.**
Production drags go `passageWordsInLocked` → `passageSpanOf` → `toggle` *per Region*, so a drag over the issue's own example yields four marks:

```
production drag  -> he stopped [sel]at[/sel] [sel]the[/sel] [sel]zenith[/sel] [sel]of[/sel] the arc
marksForDrag     -> 1 span ["at the zenith of"]  ... rendered: he stopped at the zenith of the arc  (no [sel] at all)
```
Three consequences: (1) the Spec's *"both gestures produce a SPAN, differing only in how the span is derived"* is not delivered; (2) `admitMarkedWords` (`ask.go:302-312`) runs the dictionary per word, so with NOAD installed every function word of a dragged phrase (`at`, `the`, `of`) is a hit and enters the deck as `EventMarked` — durable state, undoable only via `/forget` — which inverts the Done-when *"a dragged phrase with no dictionary entry stays out of the deck"*; (3) `marksForDrag`, `passageCell`, `firstWordFrom`, `lastWordTo` and `wrappedColumn` have **no production caller**, and `markedPassageText` (`passageprompt.go:88-97`) matches a mark only when it equals a whole word run — so the tested helper, if wired, would send an ask with no marks. `marks_test.go:70-109` pins behaviour production does not have. Also in the same decision: the plan specified *"a drag whose ANCHOR ROW is a passage row"*, but `passageWordsInLocked` tests overlap across every covered row, so a drag that starts in an answer and ends over the passage silently loses its copy and marks passage words instead.
*Fix sketch:* pick one representation. Either wire `marksForDrag` and teach `markedPassageText`/`admitMarkedWords` to handle multi-word spans (dictionary-gate the *phrase*, which is what the admission rule was written for), or delete the unwired helpers, record the per-word decision in the issue's Revisions and Done-when, and add a phrase-level guard that a drag does not admit function words. Either way no entity in the plan's table may keep a zero-consumer test.

## 3. Important findings

**I-1. `cmd/define/passage.go:143` (`byteAtCell`) — a tab is one cell here and eight on the terminal, and an unbreakable token overflows the width.** `sanitisePasteBody` deliberately keeps tabs, and only the *line* path flattens them (`pasteLineRunes`). Measured: `newPassage("\tthe slow precession", 0)`, a click at display column 8 — where the terminal draws `the` — resolves to `"slow"`. Separately, `wrapPassageLines` cannot break a long token: a 70-cell URL stays one 70-cell line at width 30, and `clipVisible` then truncates it, so its tail is neither readable nor clickable (the operator-reported bug the wrapping fix was for, still reachable). **This is the 2nd finding in family `boundary-parses-partial-class`** (BR-7 was the 1st), so do not fix the tab alone — state the rule: *every character class the paste boundary ADMITS must be representable by every downstream consumer of the passage's cell arithmetic.* The enumeration is short and mechanical — the classes `sanitisePasteBody` admits (newline, tab, `Cf`, wide glyphs, combining marks) × the consumers (`wrapPassageLines`, `byteAtCell`, `spanCells`, `paintMarks`) — and a table test over it closes the class.

**I-2. `cmd/define/passageprompt.go:115-141` — `passageSystem` restates askSystem's shared policy instead of composing it.** `renderPassagePrompt` replaces `req.System` wholesale, so the level-default paragraph, *"Never invent a definition that contradicts a dictionary entry"*, the `[lang=xx]` annotation grammar and the `&#91;`/`&#93;` escape rule now exist twice (visible side by side in `testdata/golden/passage-prompt.txt` and `ask-prompt.txt`). The issue decided the opposite in as many words: *"One answer to 'what level do we assume,' stated once — a one-line reversal in `askSystem` that read-along inherits, **not a second default in a second prompt** (ARCH-DRY)."* The language block is load-bearing for `language_decode.go`, so drift there breaks bilingual rendering of passage answers only. **This is the 2nd finding in family `repeated-shape-not-extracted`** (BR-13 is the 1st and still open), so do not patch this site alone — the rule: *a prompt paragraph consumed by more than one task is a named constant composed into each system prompt, and the shared set is derived, not remembered.* BR-13's three terminal-mode pairs and this are the same rule at two altitudes.

**I-3. Done-when rows are unpinned, and the audit that would have found it was not run.** `workshop/plans/…-plan.md:1128` (*"Confirm every `## Done when` row in the issue has a test naming it"*) is unchecked while the issue's Plan is fully ticked. Two rows fail: (a) *"the token AFTER the mark still carries the style it had (the ANSI-nesting regression) … needs a test that inspects the style after the span"* — mutation-verified: replacing both `sgrOff + style.resume()` writes in `paintMarks` with bare `sgrOff` leaves the whole suite green; (b) the dragged-phrase admission row, which C-B shows cannot hold as written. **This is the 2nd finding in family `decided-behaviour-unpinned`** (BR-6 was the 1st), so the rule rather than the two tests: *a Done-when row is a test obligation, and the close step's audit is the enumeration — run it row by row and record the test name beside each row, so a row with no test is visible rather than asserted.*

**I-4. `cmd/define/README.md:76-90, 439, 441` — the paste section is now wrong and the read-along surface is undocumented.** *"The whole paste arrives at once and goes in at the cursor; newlines and tabs inside it become spaces"* is true only for a headword-shaped paste; four words or any newline becomes a passage (`pasteIsPassage`). The key table still says a click *"plays the word"* everywhere and Enter *"see the answer, like space"* — both false inside a passage. Nothing documents click/drag-to-mark, the marks-win-over-replay rule, the `noteNothingMarked` nudge, that marks clear after an ask, or that a marked word enters the deck: the entire user-facing feature. **This is the 2nd finding in family `readme-surface-undocumented`** (BR-5 was the 1st), so state the rule: *a gesture or key the window changes is a README row, and the enumeration is derivable — `RegionKind` × `regionPlaysAudio`/`regionUnderlines` for clicks and `replKind` for Enter, the same derivation `TestAtlasDescribesEveryRegionKind` already runs against the atlas.* Extending that guard to the README closes the class.

**I-5. `workshop/plans/000067-read-along-passage-plan.md` contradicts the tree, and its staleness silences two guards.** Chunks 2–5 are entirely unchecked though the work shipped; the Architecture paragraph still says the passage *"is chrome, not scrollback … it lives in the screen's existing `footer []string` channel"*; "What this plan does NOT do" still says *"A new `RegionKind` for passage words. Resolved away in Chunk 3"* while `RegionPassageWord` is shipped; the Core-concepts table claims `selectionFrame.highlightRow | selection_frame.go | modified` for a file this window never touched; and Task 2.2 still declares `TestThePassageSurvivesALookup`, which the tree does not have. The unchecked boxes are not cosmetic: `TestPlanTablesNameEntitiesThatExist` exempts `new` rows whenever the plan contains any `- [ ]` (`repo_guard_test.go:896`), and `TestPlanTableStatusMatchesTheChangeWindow` skips a row whose file the window did not touch (`:1378`) — so exactly the two claims above are invisible. **This is the 2nd finding in family `plan-artifact-stale`** (BR-4 was the 1st), so the rule: *a plan's checkboxes and tables are claims about the tree that the repo's guards read, and leaving them behind disables the guards — tick the boxes and append a `## Revisions` entry at the same commit that lands the departure, not at the close.*

**I-6. `cmd/define/ask.go:165-172` — `gatherAskContext` runs twice on every passage ask.** `req := renderAskPrompt(gatherAskContext(...))` is computed and thrown away, then the passage branch calls `gatherAskContext` again. It is the IO step: two `UserModel()` reads and two full `Deck()` reads per ask, and if either fails the user sees *"define: could not read the deck (…); answering without it"* twice. *Fix:* gather once into a local, then choose the renderer.

## 4. Minor findings

- `cmd/define/screen.go:619-625` — `paintMarks` is applied with no reference to `opt.color`, so marking a word under `-no-color` emits `\x1b[48;5;24m\x1b[38;5;231m`. Unlike the selection's inverse video, `markOn` *is* colour.
- `cmd/define/replraw.go:605` — every paste appends ~one `Region` per word (a 1000-char passage ≈ 170) to `screen.regions` with no removal path, so the plan's ARCH-FUNERAL line (*"replaced wholesale by the next paste"*) holds for `sess.passage` but not for the screen's copy; per-session growth per paste is larger than before this window.
- `cmd/define/selection_screen.go:83` — a passage drag returns a `pointerClick` with a zero `point`, so `resolvePointerLocked` fills its `region`/`footer`/`retry` fields from frame row 0. Harmless only because the `dragged` branch is checked first.
- `cmd/define/paste.go:96-99` — the comment says an open paste is abandoned on "a control byte", but `indexPasteAbandon` matches only `0x03`/`0x04`; `\r` (which `sanitisePasteBody` strips) does not abandon, so a quiet unterminated paste still swallows Enter until the byte bound. The narrower rule is the right one — the comment should say so.
- `cmd/define/passageprompt.go:54-59` — pasted text containing `## The question` reaches the prompt as a header; brackets are escaped but Markdown headers are not. The clipboard is the user's own, so this is a note, not a hole.

## 5. Test coverage notes

- Suite state at HEAD: `go test ./cmd/define/...` is green except `TestLanguagePromptStartup` and `TestLanguageTintInvocation`, which fail at `pty.Open()` with "operation not permitted". Pre-existing and environmental — the base commit's copy of both files is byte-identical, and the failure is the sandbox-independent inability to allocate a pty in-process here (my Bash sandbox failed to initialise and was disabled for the whole session, so this is an unsandboxed run).
- **The conformance rows cannot be run in this environment and did not run.** `go test -tags conformance -run 'TestPTYBracketedPaste|TestPTYAPastedPassage'` reports `ok`, but `-v` shows `SKIP: no pty available: operation not permitted`. Any close evidence claiming the conformance suite passed should say "skipped, pty unavailable" instead.
- Mutation results: BR-1's fix reddens (good); BR-17's fix reddens (good); removing `sess.enterPaste()` from `newConsole` leaves the full in-process suite green (BR-16 stands); removing the `toInput` paste case leaves the suite green (BR-15 stands); removing `style.resume()` from `paintMarks` leaves the suite green (I-3).
- Missing coverage the diff could ship: a second paste (C-A), a drag at the prompt/deck level rather than at `passageWordsInLocked` (C-B), a passage containing a tab (I-1), and `paintMarks` under `-no-color`.

## 6. Architectural notes

- **ARCH-DRY — flag.** `passageSystem` duplicates `askSystem`'s policy (I-2); `marksForDrag` and the production drag path are two answers to "what does a drag mark" (C-B); tabs are flattened on one path and kept on the other (I-1). `wordAtCell`/`spanCells` as declared inverses and the reuse of `wordRuns`, `highlightRegion`, `wrapText`, `escapeLen` and `cellRange` are all correct reuse.
- **ARCH-PURE — pass.** `pasteScanner`, `passage`, `markSet`, `renderPassagePrompt` are pure and tested without IO; the IO shell is four small edits. `admitMarkedWords` sits on the right side of the seam (it takes `deps`).
- **ARCH-PURPOSE — flag.** The purpose is "drag across a phrase, get it explained, keep what is learnable"; what shipped marks words and admits function words (C-B). Separately, the level-default reversal was scoped GLOBAL precisely so read-along would *inherit* it, and read-along restates it instead (I-2) — the consumer does not derive from the source.
- **ARCH-MOCK — flag (unchanged from BR-16).** Mode 2004 is enabled by production wiring no runnable test exercises, and the mode set is still hand-written in two places rather than derived from the enable constants.
- **ARCH-CONSTRAINTS — pass, with a note.** The 1000-rune cap, the byte bound, one model call per ask and `markCellRanges` recomputed per draw are all bounded and argued. The unbounded growth is residue, not load (see ARCH-FUNERAL).
- **ARCH-SECURE — pass.** `sanitisePasteBody` is a real parse boundary with the decision recorded; `escapeReservedBrackets` closes the marker-forging path; no credentials touched. The residue is representational, not a trust hole: a tab survives the boundary that the cell arithmetic cannot represent (I-1).
- **ARCH-ORDER — flag.** C-A is the entry's own example: a `Region` is an observation of a passage that has since been replaced, acted on with no identity or version check. `pasteExit` is the counter-example done right.
- **ARCH-FUNERAL — flag (Minor).** The passage dies with the session, but its `Region`s do not: they accumulate in `screen.regions` per paste with no removal path, which the plan's note does not cover.

## 7. Plan revision recommendations

A `## Revisions` entry in `workshop/plans/000067-read-along-passage-plan.md` (and the matching corrections in the issue) should record, at minimum:

- **"The passage is buffer text, not footer chrome."** Rewrite the Architecture paragraph and the Chunk 2 tasks; the `passage-as-footer` integration row is wrong, and Task 2.4's atlas item ("footer chrome rather than buffer text") says the opposite of what shipped.
- **"`RegionPassageWord` is back."** Delete the "What this plan does NOT do" bullet claiming it was resolved away.
- **"`highlightRow` was not widened."** Remove the `selectionFrame.highlightRow | modified` row and all of Task 3.2; marks are painted by `paintMarks` over finished bytes at `screen.go:619-625` instead, and `selection_frame.go` is untouched.
- **"The green re-render is dropped."** Task 5.2 and the issue's Plan checkbox still claim "the passage re-rendering green", which the 2026-09-16 revision removed from scope.
- **Tick Chunks 2–5**, or state which steps did not happen — while any `- [ ]` remains, two repo guards exempt the plan's own tables (I-5).
- **Remove or rename `TestThePassageSurvivesALookup`** (plan:577) — the tree has no such test.
- **Record the drag decision** once C-B is resolved: whether a drag yields one span or N word marks, and what that means for the admission gate.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Mutation-verified at HEAD: adding "&& !s.draining" to the abandon condition (paste.go:113) reddens TestAnUnterminatedPasteDoesNotSwallowEnterOrInterrupt/after_the_drain_has_started ("Enter never emerged"); the pasteExit enumeration plus TestEveryPasteExitIsExercised closes round 4's untested-drain residual. Residual noted as Minor: the comment claims "a control byte" while indexPasteAbandon matches only 0x03/0x04.
  - id: BR-8
    disposition: not-addressed
    note: |
      key.go:66-68 is still byte-identical to the base — the struct doc says Raw carries an unmodelled sequence "so it can be ignored rather than inserted as garbage" while editor.go:64 inserts it for KeyPaste.
  - id: BR-11
    disposition: not-addressed
    note: |
      The named instances are repaired and the BACKWARD half of the class landed (currentTruthFiles now binds *_test.go, repo_guard_test.go:1738-1755). The FORWARD half was not written, and prevalence at HEAD is 3 new sites committed in this window: passage.go:16-22 ("It is CHROME, not scrollback ... It lives in the footer" — it is written to the buffer by replraw.go:605), session.go:24 ("pinned in the footer"), and README.md:76-82 ("goes in at the cursor", false for any 4-word or multi-line paste).
  - id: BR-13
    disposition: not-addressed
    note: |
      rawterm.go:105-192 still holds three hand-written enter/leave pairs and three independent bools, and restore() still hand-orders the three leaves; nothing in this window changes the shape. Newly-raised I-2 is the same rule one altitude up (duplicated prompt policy), which is worth fixing together.
  - id: BR-14
    disposition: not-addressed
    note: |
      Still no "## Log" entry for any boundary-review round (the rounds appear only in Revisions), and issue:937 still claims "full suite green" for c1844b3 where BR-2 proved it red. A third instance now: the Plan checkbox still claims "the passage re-rendering green", which the 2026-09-16 "passage is a record" revision explicitly dropped.
  - id: BR-15
    disposition: not-addressed
    note: |
      Mutation-verified at HEAD: deleting the "case KeyPaste, KeyPasteRefused" arm from toInput (play_loop.go:593) leaves go test ./cmd/define/ -run 'Sitting|Play|Paste|Input' green, because TestAPasteDuringASittingIsIgnored asserts the pre-existing default. The durable ask is untouched — grep shows no numKeyKinds sentinel in key.go, so the next KeyKind meets the same silence.
  - id: BR-16
    disposition: not-addressed
    note: |
      Mutation-verified at HEAD: commenting out sess.enterPaste() (replraw.go:89) leaves the full in-process suite green. TestNewConsoleEnablesBracketedPaste (paste_test.go:612) never calls newConsole — it builds a bare rawSession and calls the three enters by hand, pinning what rawterm_test.go already pinned. The PTY row does cover the call site but SKIPS here ("no pty available: operation not permitted"), verified with -v, so no runnable test covers it. The enumeration is still hand-written in two places (key_test.go:436, rawterm_test.go:119-152) rather than derived from the enable constants.
  - id: BR-17
    disposition: addressed
    note: |
      Mutation-verified: removing the view.Draw("", nil) before the notice (replraw.go:614) reddens TestTheRefusalNoticeDoesNotLandInsideThePrompt with "the loop drew 1 prompts; the notice did not clear the frame". Clear-write-redraw now matches the bgResults precedent.
findings:
  - id: new
    severity: Critical
    family: observation-outlives-its-subject
    title: |
      A superseded passage's regions stay clickable and resolve against the CURRENT passage, marking an unrelated word
    detail: |
      passageRegions stores Region.Line relative to its own passage; addRegions keys the screen map by absolute buffer line but leaves that field alone; nothing removes an old passage's regions on the next paste; and passageSpanOf (passage.go:367) resolves r.Line/r.Col against sess.passage with no identity check. Measured at HEAD with two passages: clicking "alpha" in the old one resolves to "zulu" in the current one, "beta" to "yankee", "epsilon" to "whiskey", and "gamma" to nothing at all (a silent no-op with no message). The mark then paints at passageBase+line, i.e. on the new passage, and the next bare Enter brackets and admits a word the reader never marked. Fix: stamp a passage generation into the Region (or resolve via the absolute buffer line and require 0 <= line-passageBase < lineCount) and reject a click that does not belong to the live passage; pin it with a two-paste regression through runEditor. ARCH-ORDER.
  - id: new
    severity: Critical
    family: tested-entity-not-wired
    title: |
      A drag marks each word separately, so the decided single span exists only in marksForDrag, which production never calls
    detail: |
      Production drags go passageWordsInLocked -> passageSpanOf -> toggle per Region (replraw.go:568-576), so a drag over the issue's own example yields four marks and the prompt reads "he stopped [sel]at[/sel] [sel]the[/sel] [sel]zenith[/sel] [sel]of[/sel] the arc" (measured). The Spec decided the opposite: "both gestures produce a SPAN, differing only in how the span is derived". Consequences: admitMarkedWords (ask.go:302) runs the dictionary per word, so with NOAD installed "at", "the" and "of" are hits and enter the deck as EventMarked — durable state, undoable only via /forget — inverting the Done-when "a dragged phrase with no dictionary entry stays out of the deck". Meanwhile marksForDrag, passageCell, firstWordFrom, lastWordTo and wrappedColumn have zero production callers, p.raw() has none at all, and marksForDrag's correct single span ("at the zenith of") is SILENTLY DROPPED by markedPassageText, which matches a mark only when it equals a whole word run — so the three tests at marks_test.go:70-109 certify behaviour the program does not have and could not send. Same decision, second site: the plan specified "a drag whose ANCHOR ROW is a passage row", but passageWordsInLocked tests overlap across every covered row, so a drag from an answer into the passage loses its copy and marks passage words instead. Fix the class in one pass: pick one representation, wire it, and let no entity in the plan's table keep a zero-consumer test. ARCH-DRY, ARCH-PURPOSE.
  - id: new
    severity: Important
    family: boundary-parses-partial-class
    title: |
      A tab survives the paste boundary but counts as one cell, so clicks land on the wrong word; an unbreakable token overflows the wrap
    detail: |
      This is the 2nd finding in family `boundary-parses-partial-class` (BR-7 was the 1st), so do NOT fix the tab alone. The rule that covers both: every character class sanitisePasteBody ADMITS must be representable by every downstream consumer of the passage's cell arithmetic. Measured: newPassage("\tthe slow precession", 0) — a click at display column 8, where the terminal draws "the", resolves to "slow", because cellWidth gives a tab 1 cell and the terminal gives it 8; pasteLineRunes flattens tabs on the LINE path and the passage keeps them, so the same input has two meanings. Second instance of the same rule: wrapPassageLines cannot break a long token, so a 70-cell URL stays one line at width 30 and clipVisible truncates it — the tail is neither readable nor clickable, which is the operator-reported bug the wrapping fix was for. The enumeration is short: the classes the boundary admits (newline, tab, Cf, wide glyphs, combining marks) x the consumers (wrapPassageLines, byteAtCell, spanCells, paintMarks), as one table test.
  - id: new
    severity: Important
    family: repeated-shape-not-extracted
    title: |
      passageSystem restates askSystem's level default, dictionary authority and language grammar instead of composing them
    detail: |
      This is the 2nd finding in family `repeated-shape-not-extracted` (BR-13 is the 1st and still open), so do NOT fix this site alone. The rule: a prompt paragraph consumed by more than one task is a named constant composed into each system prompt, and the shared set is derived rather than remembered. renderPassagePrompt replaces req.System wholesale (passageprompt.go:48), so the reversed level default, "Never invent a definition that contradicts a dictionary entry", the [lang=xx] annotation grammar and the &#91;/&#93; escape rule exist twice — visible side by side in testdata/golden/passage-prompt.txt and ask-prompt.txt. The issue forbade exactly this: "One answer to 'what level do we assume,' stated once — a one-line reversal in askSystem that read-along inherits, not a second default in a second prompt (ARCH-DRY)". The language block is load-bearing for language_decode.go, so drift breaks bilingual rendering of passage answers only. BR-13's three terminal-mode triples are the same rule at the other altitude; one extraction pattern should serve both.
  - id: new
    severity: Important
    family: decided-behaviour-unpinned
    title: |
      The Done-when audit was not run: the style-after-the-mark regression is mutation-green, and the dragged-phrase row cannot hold as written
    detail: |
      This is the 2nd finding in family `decided-behaviour-unpinned` (BR-6 was the 1st), so do NOT just add the two tests. The rule: a Done-when row is a test obligation, and the close step's audit is its enumeration — walk the rows and record the pinning test's name beside each, so a row with no test is visible rather than asserted. Evidence: plan.md:1128 ("Confirm every ## Done when row in the issue has a test naming it") is unchecked while the issue's Plan is fully ticked, and the row that names its own test — "the token AFTER the mark still carries the style it had ... needs a test that inspects the style after the span: stripping escapes is exactly what hides a lost one" — is unpinned: replacing both "sgrOff + style.resume()" writes in paintMarks with bare sgrOff leaves the whole suite green. The second unpinned row is the dragged-phrase admission rule, which the Critical above shows the code contradicts.
  - id: new
    severity: Important
    family: readme-surface-undocumented
    title: |
      README's paste section is now wrong and the whole read-along surface — click, drag, Enter-asks, the nudge, admission — is undocumented
    detail: |
      This is the 2nd finding in family `readme-surface-undocumented` (BR-5 was the 1st), so do NOT patch the paragraph alone. The rule: a gesture or key the window changes is a README row, and the enumeration is derivable — RegionKind x regionPlaysAudio/regionUnderlines for clicks and replKind for Enter, the same derivation TestAtlasDescribesEveryRegionKind already runs against the atlas; extend that guard to README.md. Sites at HEAD: README.md:76-82 says "The whole paste arrives at once and goes in at the cursor; newlines and tabs inside it become spaces", true only for a 1-3 word single-line paste since pasteIsPassage routes everything else to the passage; README.md:439 still says a click "plays the word" everywhere; README.md:441's Enter row does not mention that marks make Enter an ask. Nothing documents click/drag-to-mark, marks-win-over-replay, noteNothingMarked, marks clearing after an ask, or deck admission — the entire feature the issue is named for.
  - id: new
    severity: Important
    family: plan-artifact-stale
    title: |
      The durable plan contradicts the tree in five places, and its unticked boxes switch off the two guards that would have caught two of them
    detail: |
      This is the 2nd finding in family `plan-artifact-stale` (BR-4 was the 1st), so do NOT patch the rows one at a time. The rule: a plan's checkboxes and tables are claims about the tree that this repo's guards READ, so leaving them behind disables the guards — tick the boxes and append the Revisions entry at the commit that lands the departure, not at the close. Sites: Chunks 2-5 are entirely unticked though the work shipped; the Architecture paragraph still says the passage "is chrome, not scrollback ... it lives in the screen's existing footer []string channel"; "What this plan does NOT do" still says a passage-word RegionKind was "resolved away in Chunk 3" while RegionPassageWord is shipped; the Core-concepts table claims `selectionFrame.highlightRow | selection_frame.go | modified` for a file this window never touched; Task 2.2 declares TestThePassageSurvivesALookup, which the tree does not have. The guard interaction is the teeth: TestPlanTablesNameEntitiesThatExist exempts `new` rows whenever any "- [ ]" remains (repo_guard_test.go:896), and TestPlanTableStatusMatchesTheChangeWindow skips a row whose file the window did not touch (:1378), so exactly those two claims are invisible.
  - id: new
    severity: Important
    family: work-repeated-on-one-path
    title: |
      gatherAskContext runs twice on every passage ask, doubling the deck and learner-model reads and any warning they print
    detail: |
      ask.go:165 computes req := renderAskPrompt(gatherAskContext(d, sess, q, errOut)) and throws it away; ask.go:170 calls gatherAskContext again for the passage renderer. gatherAskContext is the IO step — d.deck.UserModel() and d.deck.Deck() — and it warns on failure, so an unreadable deck prints "define: could not read the deck (...); answering without it" twice to the user. Fix: gather once into a local, then choose the renderer.
  - id: new
    severity: Minor
    family: decoration-ignores-color-option
    title: |
      paintMarks emits a 256-colour SGR pair regardless of opt.color, so marking a word under -no-color produces colour
    detail: |
      draw() calls view.SetMarks unconditionally (replraw.go:425) and layoutSelectionFrame applies paintMarks whenever a row has marks (screen.go:622), with no reference to opt.color. Unlike the selection's inverse video, markOn is "\x1b[48;5;24m\x1b[38;5;231m". passageText already honours the flag, so the two halves of the same surface disagree.
  - id: new
    severity: Minor
    family: artifact-family-without-removal
    title: |
      Each paste appends one Region per word to screen.regions with no removal path, which the plan's ARCH-FUNERAL note does not cover
    detail: |
      replraw.go:605 writes passageRegions into the screen on every passage paste — roughly 170 entries for a 1000-character passage — and screen.regions is only ever appended to. The plan states "The passage and its marks are in-memory, die with the session, and are replaced wholesale by the next paste", which holds for sess.passage but not for the screen's copy: per-session growth per paste is larger than before this window. Note it and state the bound, or clear the superseded passage's regions (which would also help the stale-region Critical).
```

---

## Re-review — 2026-09-16T23:00:17-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 67 — define: read-along — paste a passage, click or drag what is opaque |
| repo | tools |
| issue file | workshop/issues/000067-read-along-passage.md |
| boundary | whole-issue close |
| milestone | — |
| window | 98f5c779b468ada00c087bde6bd43cca9b0892cc..c7de6a0b2cf8dc0aec653ea77b1c14f2b0cd81c6 |
| command | sdlc close --issue 67 |
| reviewer | claude |
| timestamp | 2026-09-16T23:00:17-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The feature is genuinely delivered end to end — paste → passage in the buffer → click/drag marks → one prompt with `[sel]` spans in place → dictionary-gated admission — and this round fixed real things with real evidence: `enabledModes` + a call-site-derived guard (mutation-verified), the prompt clauses single-sourced so `askSystem` and `passageSystem` compose rather than restate, `gatherAskContext` gathered once, the drag-is-one-span Critical wired into production, and `TestTheTokenAfterAMarkKeepsItsStyle` mutation-red under a bare `sgrOff`. What blocks SHIP is that the *class* behind last round's stale-passage Critical was not swept: the identity question is asked in exactly one of the three places that consume "there is a live passage". I measured a drag anchored in a superseded passage marking `"zulu yankee xray"` in the current one — the same failure BR-18 named, one gesture over — and a single paste permanently shadows Enter-replay for the rest of the session because `hasPassage` never expires. Nine prior findings also remain open, four of them Important (the documentation-class rule fixes were written as prose repairs, not as the derived guards the rules named, and five *new* contradicting prose sites shipped in this same window).

## 1. Strengths

- **`pasteExit` + `numPasteExits` + `TestEveryPasteExitIsExercised`** (`paste.go:58-77`, `paste_test.go:451`) is the best thing in the diff: the enumeration fails *closed* — a new exit no row produces reddens the guard. Same for `TestEveryRegionKindIsActionable` (`editorloop_test.go:1007`), which now requires a non-audio kind to *mark* and so still reddens for an undeclared kind.
- **`TestNewConsoleEnablesEveryMode`** (`rawterm_test.go:205`) pins the call site, not the method. Verified by deleting `sess.enterPaste()` (`replraw.go:89`): `newConsole never enabled bracketed paste … "\x1b[?1049h\x1b[?1002h\x1b[?1006h"`. BR-16's exact ask.
- **Shared prompt clauses** (`askctx.go:147-191`) — `sharedLevel` / `sharedAuthority` / `sharedLanguageGrammar` plus the extracted `escLeft`/`escRight`, with both goldens now showing the identical text. The reversed level default exists once.
- **`markedPassageText` matches by position, not by word run** (`passageprompt.go:73-99`) — that is what makes a dragged phrase survive to the wire as one span, and `TestADraggedPhraseIsAdmittedAsAPhraseOrNotAtAll` reads it off the recorded request.
- **`record()` extracted in `capture.go:120`** so `CaptureMarked` shares the one `AppendEvent → Upsert → vocab.Add` ordering rather than copying it.

## 2. Critical findings

**C-1 — The live-passage identity gate exists on the click path only; the drag path clamps into the current passage, and `hasPassage` never expires** (`session.go:89`, `session.go:50`, `replraw.go:573`).

This is the **2nd finding in family `observation-outlives-its-subject`**, so do not patch the drag call alone. The rule: *the session must answer ONE question — "is the live passage reachable at this point?" — and every consumer of passage state must route through it.* The enumeration is three long:

| consumer | gate |
|---|---|
| `passageSpanAt` (click) | `ownsBufferLine` ✓ |
| `passageCell` ×2 (drag) | **clamps instead of refusing** ✗ |
| `lineState().hasPassage` (Enter) | **never expires** ✗ |

Measured at HEAD, two passages, drag anchored on the old one's buffer row 10 with the current passage at base 40: `passageCell(10,0) = {line:0 col:0}` → `marksForDrag` returns one span → **`MARKED "zulu yankee xray"` of the current passage**. `passageDragLocked` (`selection_screen.go:177`) only checks that the anchor row carries *some* `RegionPassageWord`, and a superseded passage's regions stay in `screen.regions` forever (BR-27), so the gate it claims to be is satisfied by the wrong passage. Two harms: those words are bracketed in the next prompt and, on a dictionary hit, enter the deck as durable `EventMarked` records; and the copy the reader asked for is swallowed by the `continue`.

Second site, same rule: `parseREPLLine` (`repl.go:140`) treats `hasPassage` as authority over the Enter key. Measured: with `current` set, a bare Enter is `cmdReplay`; after one paste it becomes `cmdNothing`/`noteNothingMarked` and **never returns to replay for the rest of the session** — long after the passage has scrolled away, and while the nudge tells the reader to click something that may be off-screen. This decision was correct while the passage was pinned footer chrome; the footer→buffer reversal changed the surface's lifetime and the predicate was not re-derived.

Fix sketch: make `passageCell` return `(passageCell, bool)` and refuse a row outside `ownsBufferLine`, and make `hasPassage` mean *the live passage is on screen* (derive it from the viewport, or clear `sess.passage` when it leaves). Pin both with a two-passage regression through `runEditor`, and one asserting replay returns once the passage is gone. Note `pointerClick.line` is useless on the drag path — `resolvePointerLocked` (`selection_screen.go:116`) overwrites it with `LineAt(click.point.row)` and `point` is the zero value for a drag, so the field that should carry identity carries row 0's line and `hasRegion` can be spuriously true. `hasRegion`/`hasDrag`/`footer`/`retry`/`line` is now a 5-field constellation whose legal combinations are unwritten — collapse it into a tagged variant (ARCH-ORDER). **ARCH-ORDER, ARCH-PURPOSE.**

## 3. Important findings

**I-1 — The one line that makes Enter ask is deletable-green, and `replKind` has no sentinel** (`replraw.go:667`).

**3rd finding in family `production-seam-untested`** (BR-3, BR-16), so do not just add a test for this line. Measured: deleting the whole `if cmd.kind == cmdAskPassage { askInSession(…) }` block leaves `go test ./cmd/define/...` green apart from the two pty-denied rows. Every ask test constructs `question{passage: …}` by hand (`passageprompt_test.go:171-340`); `TestABlankLineWithMarksAsksAboutThePassage` stops at `parseREPLLine`. Nothing connects the two, so the feature's headline gesture has no end-to-end pin. The rule is the one BR-16 established, one altitude up: *a decision table's kinds are a registry, so every kind's disposition in every loop is declared and derived, not hand-written per loop.* Second site proving it is a class: `replLines` (`repl.go:384`) has **no case for `cmdAskPassage`** — unreachable today only because the piped loop never decodes a paste. `replKind` (`repl.go:45`) has no `numReplKinds`, which is why nothing asked. The move is already in this window three times (`numPasteExits`, `numRegionKinds`, `numKeyKinds`) — add the fourth and drive both loops from it.

## 4. Minor findings

- `passage.raw()` (`passage.go:75`) has zero consumers in the tree — production or test. BR-19 named it and it survived; delete it rather than keeping it for symmetry.
- `renderAskPrompt` runs twice on every passage ask (`ask.go:167` then again inside `renderPassagePrompt`). Pure and cheap, but it is the same shape BR-25 fixed one layer up: choose the renderer before building a prompt you discard.
- `regionPlaysAudio`/`regionUnderlines` live in `replraw.go` (the IO shell) while `RegionKind` and `numRegionKinds` live in `render.go`. The declaration belongs beside the registry it partitions (ARCH-PURE placement nit; matches the plan's table, so it is a plan question too).
- `gofmt -l cmd/define/` reports `selection_paths_test.go` (`questionsFor(t,d,opt)`). Pre-existing — this window did not touch the file — but it is unformatted at a close boundary.

## 5. Test coverage notes

- Suite at HEAD: green except `TestLanguagePromptStartup` and `TestLanguageTintInvocation`, both failing at `pty.Open()` with "operation not permitted". Environmental, not from this diff, and consistent with the documented note — but note the sandbox was **disabled** for this session and they still fail, so "fails under the Bash sandbox" is not the whole explanation.
- **Every PTY conformance row skips in this environment**, including all three added this window (`TestPTYBracketedPasteIsAskedForAndGivenBack`, `TestPTYAPastedPassageAppearsAndDoesNotSubmit`, `TestPTYEveryEnabledModeIsAskedForAndGivenBack`). They compile under `-tags conformance` (`go vet -tags conformance` clean) and the derivation from `enabledModes` is correct by reading, but the issue's Done-when cites `TestPTYAPastedPassageAppearsAndDoesNotSubmit` as live evidence and it has not run here.
- Mutation evidence gathered: `sgrOff + style.resume()` → bare `sgrOff` reddens `TestTheTokenAfterAMarkKeepsItsStyle` ✓; `sess.enterPaste()` removed reddens `TestNewConsoleEnablesEveryMode` ✓; a new `KeyKind` added before `numKeyKinds` leaves `TestEveryKeyKindIsDecidedForASitting` **green** ✗; the `cmdAskPassage` branch removed leaves the suite **green** ✗.
- Two Done-when rows cite tests that do not exist: `TestThePassageIsWrittenToTheBufferNotTheFooter` and `TestAMarkedDeckWordRendersAsAMarkNotAsADeckWord`. The passage-in-buffer claim and the marked-deck-word precedence rule are therefore unpinned at the level the audit asserts. No guard reads `workshop/issues/`, which is why the audit could assert them.
- Untested and broken: a passage line containing an unbreakable token longer than the terminal width. Measured — a 67-cell URL at width 30 becomes one 67-cell passage line, and `clipVisible` draws `"https://example.com/a/very/lon"`. The tail is neither readable nor clickable.

## 6. Architectural notes

- **ARCH-DRY** — flag (BR-13, BR-21 residual). `enabledModes` (`rawterm.go:183`) is the right single list but nothing derives from it except the two new guards: `enterAlt/enterMouse/enterPaste` are still three hand-written pairs over three independent bools, `restore()` still hand-calls its teardown in an order the list does not express (the list is alt/mouse/paste; teardown is mouse/paste/alt), and `TestEveryEnabledInputModeIsDecoded` still hand-concatenates `const inputModes = mouseOn + pasteOn` (`key_test.go:431`). A fourth mode added to `enabledModes` is invisible to that guard.
- **ARCH-PURE** — pass. `pasteScanner`, `passage`, `markSet`, `paintMarks`, `renderPassagePrompt` are pure and unit-tested with no IO; the shell is `enterPaste`, `WriteRegions`, `SetMarks`, `CaptureMarked`.
- **ARCH-PURPOSE** — flag. Shadow-sweep on the single-sourcing: prompt clauses ✓, bracket escapes ✓, event kinds ✓. Still hand-maintained restatements of the model: README's two tables, the atlas's tab claim, the plan's footer sections, three production doc comments. And C-1/I-1 are both "the instance, not the class" — the site a prior finding named was fixed while enumerable siblings remained.
- **ARCH-MOCK** — pass, with the environment caveat above. The terminal is treated as the external dependency; both an in-process assertion and a live row exist per enabled mode; `passage_conformance_test.go` defends the persona the prompt cannot.
- **ARCH-CONSTRAINTS** — pass. Two predicates (`maxPasteRunes` semantic, `maxPasteBytes` memory) with the split argued; the drain consumes so the caller's buffer cannot grow; one model call per ask, never per mark; `markCellRanges` allocates nothing when unmarked.
- **ARCH-SECURE** — pass. `sanitisePasteBody` is a real parse boundary (Cc + escapes out, Cf deliberately kept, tab→space), `newPassage` does not depend on it having run, brackets escaped before the prompt, no credentials.
- **ARCH-ORDER** — flag, twice. `pasteScanner` is exemplary: an explicit `(state, event) → (state, exit)` enumeration with a sentinel and a closed guard. Against that, `pointerClick`'s field constellation and `TestEveryKeyKindIsDecidedForASitting` comparing two switches that both default to `false` — a guard whose stated purpose ("so a new kind cannot slip past") it cannot serve, which is the oracle failing rather than the code.
- **ARCH-FUNERAL** — flag (BR-27 open). `screen.regions` gains ~one entry per passage word per paste with no removal path; the plan's ARCH-FUNERAL paragraph still says "no removal path needed", which is true of `sess.passage` and false of the screen's copy. Clearing a superseded passage's regions would also shrink C-1's blast radius.

## 7. Plan revision recommendations

Append one `## Revisions` entry — *"the passage is a record, not chrome (and what that reversed)"* — rather than continuing to overwrite prose in place (AGENTS.md §1). It needs to carry:

- The Integration points row `passage-as-footer | cmd/define/replraw.go | modified | liveScreen.Draw` (`plan:80`) and its bullet (`:91`) → the passage is written to the buffer via `WriteRegions`; `Draw`'s footer carries nothing of it.
- Task 2.2's title and Step 3 (`:591`, "passage lines become footer entries ahead of `menuLines`"), Task 2.3 (`FooterRowAt` refuses undrawn rows), and Task 2.4's ticked atlas row (`:611`, "footer chrome rather than buffer text") — all three describe the superseded design; the atlas says the opposite.
- **Task 3.2 "Widen `highlightRow` from one range to a set" (`:707`) is fully ticked and did not happen.** `highlightRow` is still `(row int, a, b selectionPoint)` at `selection_frame.go:208`, and this window does not touch that file. Marks are painted by `paintMarks` over finished bytes instead. The estimate priced this as one of four `cross-cutting-refactor` rows.
- `TestThePassageIsWrittenToTheBufferNotTheFooter` (`:583`) and `TestThePassageRendersWithDeckColour` (`:579`) are not in the tree; the shipped names are `TestThePassageIsWrittenWithDeckColour` and (for the scroll-survival claim) nothing.
- Also correct the issue, in the same edit: `## Plan` row `:757` still says the passage is "rendered into the screen's `footer` channel (chrome, not buffer text)" and `:760` still claims "`highlightRow` widened from one range to a set"; the Log's "full suite green" at `:961` was false at the commit it describes; and the two Done-when rows above cite tests that do not exist.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      key.go:74-76 is byte-identical; the Key doc still says Raw is an unmodelled sequence to be ignored while KeyPaste's Raw is sanitised text that Apply inserts.
  - id: BR-11
    disposition: not-addressed
    note: |
      The three named sites were repaired, but no forward-direction guard was written and five NEW contradicting sites shipped in this window: passage.go:17-21 ("It is CHROME, not scrollback ... It lives in the footer"), session.go:24 ("pinned in the footer"), replraw.go:558-562 ("the hit test is FooterRowAt plus wordAtCell, and no new RegionKind exists", three lines above code keying on RegionPassageWord), atlas/define.md:313-314 ("Newlines and tabs survive" while sanitisePasteBody turns tabs into spaces and paste_test.go:183 pins that), and the plan's footer sections. currentTruthFiles now binds *_test.go (the backward half), which is real progress; the forward half — every identifier prose names must be declared at HEAD — does not exist.
  - id: BR-13
    disposition: not-addressed
    note: |
      enabledModes (rawterm.go:183) is the right list but nothing derives from it beyond the two new guards: three hand-written enter/leave pairs over three bools remain, restore() still hand-calls its teardown in an order the list does not express (list is alt/mouse/paste, teardown is mouse/paste/alt), and key_test.go:431 still hand-concatenates `const inputModes = mouseOn + pasteOn`, so a fourth mode is invisible to it.
  - id: BR-14
    disposition: not-addressed
    note: |
      issue:961 still claims "full suite green" for the commit BR-2 proved red, and ## Log still holds exactly one dated section with no per-round boundary-review entry (five rounds have now run). The milestone-collapse Revisions entry mentions "four review rounds" in passing, which is the nearest thing to a record.
  - id: BR-15
    disposition: not-addressed
    note: |
      The case and the declaration landed (play_loop.go:571, :617) and TestAPasteDuringASittingIsIgnored pins the behaviour — but the guard, which was the finding, cannot fail for the failure it names. Measured: adding a KeyKind before numKeyKinds leaves TestEveryKeyKindIsDecidedForASitting GREEN, because keyBecomesASittingInput and toInput both default to false, so a kind wired into neither agrees with itself. numPasteExits and numRegionKinds fail closed in this same window; this one fails open.
  - id: BR-16
    disposition: addressed
    note: |
      Mutation-verified: deleting sess.enterPaste() (replraw.go:89) reddens TestNewConsoleEnablesEveryMode with "newConsole never enabled bracketed paste". The PTY row TestPTYEveryEnabledModeIsAskedForAndGivenBack exists and loops enabledModes; it compiles under -tags conformance (go vet clean) and skips here because pty.Open is denied.
  - id: BR-18
    disposition: addressed
    note: |
      The CLICK path is genuinely fixed and pinned: passageSpanAt gates on ownsBufferLine (session.go:79,99) and TestAStalePassagesRegionsDoNotMarkTheCurrentOne asserts a stale region refuses. The enumerable siblings of the class are raised fresh below rather than re-raised here.
  - id: BR-19
    disposition: addressed
    note: |
      Production drags now go through marksForDrag (replraw.go:573) and yield ONE span; the anchor gate is enforced in passageDragLocked (selection_screen.go:177) so a drag from an answer keeps its copy; markedPassageText matches by position so a phrase reaches the wire as one bracket; TestADraggedPhraseIsAdmittedAsAPhraseOrNotAtAll reads both the deck and the recorded prompt. Residual: passage.raw() still has zero consumers anywhere (raised Minor).
  - id: BR-20
    disposition: not-addressed
    note: |
      The tab instance is fixed and pinned to ONE outcome (paste.go sanitisePasteBody, paste_test.go:168-185). The rule fix was not written: no classes-by-consumers table test exists, and the second instance stands. Measured at HEAD: newPassage("see <67-cell URL> for more", 30) yields a 67-cell line and the frame draws "https://example.com/a/very/lon" — clipVisible truncates because wrapText cannot break an unbreakable token, so the tail is neither readable nor clickable. That is the operator-reported wrapping bug, half fixed.
  - id: BR-21
    disposition: addressed
    note: |
      sharedLevel / sharedAuthority / sharedLanguageGrammar extracted in askctx.go:147-191 and composed by both askSystem and passageSystem; escLeft/escRight extracted and consumed by both the prompt escape and the answer-direction rule. Both goldens now show the identical clauses.
  - id: BR-22
    disposition: not-addressed
    note: |
      Half addressed with real evidence: TestTheTokenAfterAMarkKeepsItsStyle is mutation-red (replacing both "sgrOff + style.resume()" writes with bare sgrOff fails it), and the dragged-phrase row now has a test. But the audit asserts pins that do not exist — TestThePassageIsWrittenToTheBufferNotTheFooter and TestAMarkedDeckWordRendersAsAMarkNotAsADeckWord are in no _test.go file — so two rows are still asserted rather than visible, which is the rule's own failure mode. No guard reads workshop/issues/; TestPlanCitesTestsThatExist globs workshop/plans/*-plan.md only, which is where the teeth belong.
  - id: BR-23
    disposition: not-addressed
    note: |
      The new "Read along" section (README.md:76-121) is accurate and thorough, and the old paste paragraph is gone. But the derived guard the rule named was not written (TestAtlasDescribesEveryRegionKind still reads only the atlas), and two rows now contradict the new section: README.md:163 "Enter on an empty line | replay the pronunciation" is unconditional, and README.md:169-174 still says "Underlined words are clickable" and "A click on ordinary text does nothing" while passage words are clickable and deliberately NOT underlined.
  - id: BR-24
    disposition: not-addressed
    note: |
      Boxes ticked and three of five sites repaired (Architecture paragraph, RegionPassageWord bullet, selectionFrame.highlightRow row removed). Remaining: Integration points row "passage-as-footer" (plan:80) and its bullet (:91); Task 2.2 title plus Step 3 (:591), Task 2.3, and Task 2.4's ticked atlas row (:611) all still say footer; Task 3.2 "Widen highlightRow from one range to a set" (:707) is fully ticked though highlightRow is unchanged at selection_frame.go:208 and this window never touches that file; plan:579 and :583 declare TestThePassageRendersWithDeckColour and TestThePassageIsWrittenToTheBufferNotTheFooter, neither in the tree. And the rule's own second clause was not followed: the footer-to-buffer reversal was applied by OVERWRITING the Architecture paragraph, with no Revisions entry appended.
  - id: BR-25
    disposition: addressed
    note: |
      ask.go:167 gathers once into askCtx and both renderers take it; the double deck/learner-model read and the duplicated warning are gone. Residual (Minor below): renderAskPrompt itself still runs twice on the passage path, purely.
  - id: BR-26
    disposition: not-addressed
    note: |
      replraw.go:425 calls SetMarks(markCellRanges(...)) and screen.go:622-624 applies paintMarks whenever a row has marks, with no reference to opt.color anywhere on the path; markOn is "\x1b[48;5;24m\x1b[38;5;231m". passageText still honours the flag, so the two halves of the same surface disagree under -no-color.
  - id: BR-27
    disposition: not-addressed
    note: |
      replraw.go:609 still writes passageRegions into screen.regions on every passage paste with no removal path, and the plan's ARCH-FUNERAL paragraph is unchanged — it still says "no removal path needed", which holds for sess.passage and not for the screen's copy. Clearing a superseded passage's regions would also shrink the new Critical's blast radius.
findings:
  - id: new
    severity: Critical
    family: observation-outlives-its-subject
    title: |
      The live-passage gate is on the click path only — a drag in a superseded passage marks the CURRENT one, and hasPassage never expires
    detail: |
      This is the 2nd finding in family `observation-outlives-its-subject`, so do NOT patch the drag call alone. The rule that covers both sites: the session must answer ONE question — "is the live passage reachable at this point?" — and every consumer of passage state routes through it. The enumeration is three long and two are ungated. (1) `passageSpanAt` (session.go:99) gates on `ownsBufferLine` — correct, and BR-18's fix. (2) `passageCell` (session.go:89) CLAMPS instead of refusing, and `passageDragLocked` (selection_screen.go:177) only checks that the anchor row carries some `RegionPassageWord`, which a superseded passage's rows still do because `screen.regions` is never pruned. Measured at HEAD with two passages, drag anchored on the old one's buffer row 10 against a current passage at base 40: `passageCell(10,0) = {line:0 col:0}` and `marksForDrag` returns one span — MARKED "zulu yankee xray" of the current passage. Those words are then bracketed in the next prompt and, on a dictionary hit, admitted to the deck as durable EventMarked records; the copy the reader asked for is swallowed by the `continue` at replraw.go:576. (3) `lineState().hasPassage` (session.go:50) treats "a passage was once pasted" as permanent authority over Enter. Measured: with `current` set a bare Enter is `cmdReplay`; after one paste it is `cmdNothing`/`noteNothingMarked` and never returns to replay for the rest of the session — long after the passage has scrolled away, while the nudge points at something possibly off-screen. That decision was right when the passage was pinned footer chrome; the footer-to-buffer reversal changed the surface's lifetime and the predicate was not re-derived. Fix: `passageCell` returns `(passageCell, bool)` and refuses a row outside `ownsBufferLine`; `hasPassage` means the live passage is on screen. Pin with a two-passage drag regression through runEditor and one asserting replay returns once the passage is gone. Note `pointerClick.line` cannot serve as the gate as written: resolvePointerLocked (selection_screen.go:116) overwrites it from `click.point`, which is the zero value on the drag path, so it carries row 0's buffer line and `hasRegion` can be spuriously true — hasRegion/hasDrag/footer/retry/line is a five-field constellation whose legal combinations are unwritten and should collapse into a tagged variant. ARCH-ORDER, ARCH-PURPOSE.
  - id: new
    severity: Important
    family: production-seam-untested
    title: |
      Deleting the cmdAskPassage branch in runEditor leaves the suite green, and replKind has no sentinel so replLines silently has no case for it
    detail: |
      This is the 3rd finding in family `production-seam-untested` (BR-3, BR-16), so do NOT just add a test for this line. Measured: removing the whole `if cmd.kind == cmdAskPassage { askInSession(...) }` block at replraw.go:667-675 leaves `go test ./cmd/define/...` green apart from the two pty-denied rows — the one line that turns "Enter with marks" into an actual model call is unpinned. Every ask test builds `question{passage: ...}` by hand (passageprompt_test.go:171-340) and TestABlankLineWithMarksAsksAboutThePassage stops at parseREPLLine, so nothing joins the decision to the effect. The rule, which is BR-16's one altitude up: a decision table's kinds are a registry, so each kind's disposition in each loop is DECLARED and DERIVED, not hand-written per loop. Second site proving it is a class: `replLines` (repl.go:384) has no case for cmdAskPassage at all — unreachable today only because the piped loop never decodes a paste — and `replKind` (repl.go:45) has no `numReplKinds`, which is why nothing forced the question. This window makes exactly that move three times (numPasteExits, numRegionKinds, numKeyKinds); make it a fourth and drive both loops from it, with the guard failing CLOSED (see the BR-15 disposition for the failure mode to avoid).
  - id: new
    severity: Minor
    family: tested-entity-not-wired
    title: |
      passage.raw() has zero consumers anywhere in the tree
    detail: |
      passage.go:75. BR-19 named it and it survived the round: no production caller and no test caller. Delete it rather than keeping it for symmetry with lineCount/line/spans/text, all of which are used.
```
