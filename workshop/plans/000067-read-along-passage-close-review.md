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
