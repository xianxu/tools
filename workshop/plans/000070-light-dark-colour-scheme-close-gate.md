---
gate: boundary-review
issue: 70
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-17T22:58:39-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: 'The plan restates the diff: full implementations, test cases listed in prose, call sites by line number'
          detail: |-
            The gate asks for named functions plus one strategy line per risky function. It does not block this plan, which can be carried out and whose facts were checked; compress in future plans rather than rewriting this one.
            (carried from plan-quality PQ-4, deferred to the boundary review)
          family: plan-restates-code
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-17T22:58:39-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Minor
          title: schemeState.withChoice accepts any schemeSource, so choice-by-detected, choice-by-default and empty-choice states are representable
          detail: 'The spec''s choice *{value, source: flag|saved|session} is implemented as choice plus chosenBy schemeSource with sourceDefault as the absent marker (cmd/define/scheme.go:30,49). M2''s describeScheme will print this source to the user; give the choice its own source type or use *schemeChoice before M2 builds on it (ARCH-ORDER).'
          family: state-shape-admits-illegal-combinations
          round: 2
        - id: BR-3
          severity: Minor
          title: Two live-edge footer invariants are tested through the test-only boardFooter, not production's boardFooterOutput
          detail: TestBoardFooterPutsTheFormsOwnRowsFirst and the fitsABoard row count (play_loop_test.go:2417,3049) call boardFooter (render_helpers_test.go:82), which also now holds the only copy of the load-bearing rationale. Point them at paintedBoardFooterForTest, move the comment to boardFooterOutput, and delete boardFooter (ARCH-DRY). Pre-existing; M1 moved it into a test file.
          family: tests-pin-a-shadow-of-the-live-path
          round: 2
        - id: BR-4
          severity: Minor
          title: The atlas paragraph on paint-time shade describes saved/session sources and detect/forget as live in M1
          detail: atlas/define.md:503-515 describes the terminal-reported value and the saved and session sources as if they worked now, but none has a production caller until M2/M3. The plan says docs should describe only what each milestone ships; add one clause saying so.
          family: docs-describe-unshipped-surface
          round: 2
        - id: BR-5
          severity: Minor
          title: Done-when says dead paths are deleted with tint assertions ported; the renderers were moved to test helpers and fragment-tint assertions dropped
          detail: The Log's Task 3 entry explains why, but the list of spec revisions the plan makes omits it, so the issue's Done-when (line 320) and Spec section on Repaint still claim what was not done.
          family: spec-done-when-drifts-from-delivery
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: false
    - "n": 3
      timestamp: "2026-09-17T23:46:34-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: The fix belongs in future plans, and no lessons.md rule records it; Chunk 1's code blocks have already drifted from the code (the plan's own Revisions says so), which shows the cost. Minor, not blocking.
          round: 3
      findings:
        - id: BR-6
          severity: Important
          title: M2 plan steps are ticked but the issue Log has no M2 entry
          detail: Task 7 Step 6 says the WriteScheme symlink-to-regular-file behaviour is "noted in the Log"; it is not. Task 11 Step 4 claims the TestPTY conformance run PASSED and should name what ran and what was skipped, but the M1 Log records 13 TestPTY tests already failing, so a clean PASS contradicts it. The mutation record the Done-when requires is also missing. Add an M2 Log entry covering the symlinked-file behaviour, the pty run broken into passed, pre-existing failures and skipped, and the mutation list.
          family: ticked-step-lacks-its-evidence
          round: 3
        - id: BR-7
          severity: Minor
          title: commandCtx session/fullScreen and schemeArg auto/value allow combinations that mean nothing
          detail: 'This is the 2nd finding in this family. Rule: when fields depend on each other, they should be ONE tagged value whose members are exactly the legal combinations. Remaining instances in #70: commandCtx.session plus fullScreen (fullScreen without session is representable), and schemeArg auto plus value (its zero value would save a blank line, which every later startup warns about). Neither is reachable today. Measured prevalence in #70: 3 instances; choice/chosenBy was fixed at M1. Fix: a loopKind enum {oneShot, piped, editor}, and a nil-able choice where nil means auto.'
          family: state-shape-admits-illegal-combinations
          round: 3
        - id: BR-8
          severity: Minor
          title: The startup read skips schemePersister, and the precedence order is tested only through a pty
          detail: 'run() resolves d.configDir and calls store.ReadScheme inline, while deps.schemePersister resolves it again for save and clear (ARCH-DRY). The flag, then saved, then default order lives in run() glue; "flag beats saved" and "garbled file warns once" are pinned only by TestSavedSchemeGovernsALookup through a real pty (ARCH-PURE). Fix: add load() to schemePersister and extract a pure initialSchemeState(flag, persister, warn) that can be unit-tested without a pty.'
          family: store-access-bypasses-its-seam
          round: 3
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 4
      timestamp: "2026-09-18T00:00:08-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'Forward-looking and still true: the 2026-09-18 Revisions entry concedes Chunk 2''s code has drifted too, Chunk 3 still restates code, and no lessons.md rule records it. Minor, never blocks.'
          round: 4
        - id: BR-6
          disposition: addressed
          note: The Log now has an M2 entry covering the symlinked FILE (rename replaces the link; the directory is kept, see TestClearSchemeRemovesOnlyWhatIsOurs), the pty run (1338 passed, 1 skipped, 14 failed, the 14 being the pre-existing set from M1) and the per-task mutation list.
          round: 4
        - id: BR-7
          disposition: not-addressed
          note: 'Both named instances are fixed (schemeArg: empty = auto; loopKind). But the class sweep the fix commit claims was not done: schemeState.detected + heard (scheme.go:60-63) is a third pair, and heard=false with a detected value, or heard=true with an empty one, is representable. The spec says detected *scheme. Enumerated #70 structs (schemeState, schemeChoice, schemeArg, commandCtx, tintPolicy); this is the only remaining instance, so prevalence in #70 is 4. Fix: detected store.Scheme with empty meaning not heard, drop heard.'
          round: 4
        - id: BR-8
          disposition: addressed
          note: schemePersister.load plus the pure initialSchemeState(flag, persister, warn), used in run() at main.go:871. TestInitialSchemeState pins flag-beats-saved without a load and one warning for a garbled file, with no pty. Swapping the precedence in a scratch copy turned it red.
          round: 4
      findings:
        - id: BR-9
          severity: Minor
          title: The atlas says /scheme's loopKind decides "does this loop ask the terminal", which no loop does until M3
          detail: 'This is the 2nd finding in this family (BR-4 at M1 was the 1st). atlas/define.md:1016-1017, added in a4ec10d, drops the "(from M3)" tag that the loopKind code comment keeps. Rule: a doc written at milestone N describes only what N ships, and anything from a later milestone carries that milestone inline. Sweep: grep the milestone''s doc diff for the later milestone''s words (for #70: detect, ask, report, query, OSC, KeyBackground). That sweep over atlas and README finds only this hit; prevalence in #70 is 2.'
          family: docs-describe-unshipped-surface
          round: 4
      boundary: M2
      recipe: milestone-review
      blocked: false
    - "n": 5
      timestamp: "2026-09-18T09:36:44-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: M3 Tasks 13-15 still carry full implementations, and the window adds no lessons.md rule telling future plans to name functions plus one strategy line. Record that rule at close; the plan itself needs no rewrite.
          round: 5
      findings:
        - id: BR-10
          severity: Important
          title: 'The sitting''s repaint after a background reply is unpinned: removing show() at play_loop.go:315-317 leaves the suite green'
          detail: 'This is the 2nd finding in family tests-pin-a-shadow-of-the-live-path. Rule: a consumer''s test checks what that consumer itself outputs (the frame it paints, the answer it records, the notice it posts), never only the shared state it writes. PaintedTranscript is not evidence of a repaint, because it re-reads the holder when called. Consumers of KeyBackground: editor frame pinned (dropping draw() goes red, verified); router drag and drop-site notice pinned; sitting frame NOT pinned (TestASittingIgnoresABackgroundReply checks d.scheme.Scheme(); dropping show() stayed green, verified), and this is the path that paints --play''s first question on a light terminal. TestAReplyDuringPlayReachesTheEditor''s PaintedTranscript check repeats its effective() check. Fix: follow TestASittingPaintsInTheEditorsScheme (a tinted row on the sitting''s screen), send the reply, and assert lastFrame of the tty carries languageLight. Record the rule in lessons.md.'
          family: tests-pin-a-shadow-of-the-live-path
          round: 5
        - id: BR-11
          severity: Important
          title: Ticked M3 steps name evidence the record does not hold, including the three-terminal x two-appearance manual check
          detail: 'This is the 2nd finding in family ticked-step-lacks-its-evidence; the 1st was M2''s BR-6. Rule: tick a step that names evidence (a run result, "name what ran", a mutation list, a recorded matrix) only in the commit that writes that evidence where the step says it goes. If a step was done differently, add a Revisions entry saying what was done instead. Steps to sweep: Task 14 Step 4 (three 30 s fuzz runs); the mutation steps of Tasks 13-17 (the Log gives counts, not names as M1 and M2 did); Task 17 Step 3 (which TestPTY tests ran or skipped under CONFORMANCE_STRICT; TestPTYBackgroundDetection sits behind bilingualNativeProbe); Task 18 step 1 (full suite, -race, vet, linux build, strict results); Task 18 step 2 (the matrix; the atlas says "not itemised"). Done-when bullet 1 still claims Terminal.app, iTerm2 and Ghostty in both appearances. Bullet 6 claims Alt-] and #rrggbb are pinned "through each loop shell", but only the decoder and readInput test them. The README says those three terminals reply. Either record the matrix or add a Revisions entry that narrows bullet 1, and make the README match what is recorded. Record the rule in lessons.md.'
          family: ticked-step-lacks-its-evidence
          round: 5
        - id: BR-12
          severity: Minor
          title: The paired-ink reset in unfill (language_row.go:34-37) is unpinned; removing inkOff leaves the suite green
          detail: 'Verified by mutation. If it regressed, 235 or 252 ink would carry onto a producer''s own background (the \x1b[42mc cell, column 3, in TestLanguageRowStylesAndExclusions) and onto uncoloured excluded answer cells. The atlas says the ink steps aside for either. Fix: assert cells[3].fg == -1 there, and add an uncoloured excluded cell with fg == -1.'
          family: branch-survives-its-mutation
          round: 5
        - id: BR-13
          severity: Minor
          title: The KeyBackground intercept is copied in runEditor (replraw.go:553-561) and playSession (play_loop.go:311-318)
          detail: ARCH-DRY. The neighbouring viewportGesture helper exists precisely so the two loops cannot disagree about a shared key. A one-line helper (apply the report through d.scheme.detect and report whether to repaint) would make the report rule a single source.
          family: duplicated-logic-not-extracted
          round: 5
      boundary: M3
      recipe: milestone-review
      blocked: true
    - "n": 6
      timestamp: "2026-09-18T09:55:13-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: lessons.md now carries "A plan names functions and one strategy line each, not their code (#70)", the rule the prior round asked for at close; the plan itself needs no rewrite.
          round: 6
        - id: BR-10
          disposition: addressed
          note: TestASittingRepaintsOnABackgroundReply reads the sitting's painted frame; removing show() turns it red (timeout). The editor-after-play test reads lastFrame; removing resume()'s repaint turns it red. Both verified by mutation in a scratch copy of HEAD.
          round: 6
        - id: BR-11
          disposition: addressed
          note: 'Log names every M3 mutation, fuzz run, strict set and boundary run (two spot-checked red); bullet 1 narrowed by a Log REVISION and plan Revisions; README no longer names terminals; the loop shells now test #hex and Alt-]. The separate distinguishing-evidence gap is raised as a new finding.'
          round: 6
        - id: BR-12
          disposition: addressed
          note: TestTheInkStepsAsideWithTheTint pins the excluded cell and the producer background at fg -1; removing inkOff reddens both (verified).
          round: 6
        - id: BR-13
          disposition: addressed
          note: terminalReport (replraw.go:144-153) is the one report rule, used by runEditor and playSession.
          round: 6
      findings:
        - id: BR-14
          severity: Important
          title: Done-when bullet 1 cites the operator check as evidence of detection, but the record cannot tell detection from a saved choice
          detail: 'This is the 3rd finding in family tests-pin-a-shadow-of-the-live-path. Rule for the whole family: evidence for a path (a test assertion OR a manual check) must be something only that path can produce; the shared holder, PaintedTranscript''s re-read, and a light tint that a saved choice also paints all fail this. The Log''s evidence for "light -> 254, dark -> 236, no flag, no saved file" ends with "operator check". The only recorded detail of that check is the first run, and the Log says a saved light was in force then ("not yet /scheme auto''d"). The re-check is recorded only as "working", with no /scheme report. The atlas also calls that run "a light profile", while the Log describes a terminal with white default text. Sweep of the Done-when evidence list: only this item fails; the pty (detected) reports, the in-process light-reply frame and the late-reply repaint all distinguish detection. Fix: extend the lessons.md rule from tests to manual evidence. For this instance, either record one real-terminal /scheme report reading "(detected)" in each appearance after /scheme auto, or state that the live check did not establish detection, so detection''s only evidence is the modelled pty terminal and the ARCH-MOCK live check is still owed. Either way, correct the atlas wording.'
          family: tests-pin-a-shadow-of-the-live-path
          round: 6
      boundary: M3
      recipe: milestone-review
      blocked: true
    - "n": 7
      timestamp: "2026-09-18T11:08:59-07:00"
      agent: claude
      dispose:
        - id: BR-14
          disposition: addressed
          note: Log, atlas:547-558 and plan Revisions record a real /scheme reading "dark (detected)" after /scheme auto; light detection is recorded as owed; lessons.md extends the rule to manual evidence; the "light profile" wording is gone.
          round: 7
      findings:
        - id: BR-15
          severity: Minor
          title: paintLanguageRow stores inking, which always equals filled && !coloured (language_row.go:26-27,34)
          detail: 'This is the 3rd finding in family state-shape-admits-illegal-combinations. The line-26 comment states the derivation. unfill() runs before every change to coloured, so the flag is redundant. Scratch probe: dropping it and writing if !coloured { inkOff } left every tint, ink, row, selection and screen test green. Rule for the family: store only independent facts; compute a flag that follows from other state where it is read. Add this to lessons.md. Sweep of the M3 diff: this is the only derived flag.'
          family: state-shape-admits-illegal-combinations
          round: 7
        - id: BR-16
          severity: Minor
          title: 'README scheme section: "suits a dark terminal by default" is out of date and "With nothing chosen ... asks" is too narrow (README.md:343,359)'
          detail: 'This is the 3rd finding in family docs-describe-unshipped-surface. wantsBackground asks whether or not a scheme is chosen; that is what lets /scheme auto show a detected value. And a session no longer just defaults to dark. Rule: when a milestone changes a behaviour, check every doc sentence about it against the code''s actual condition. Docs describe exactly what ships: no future features, no stale defaults, no narrower conditions. Sweep: the atlas detection paragraph and -h are correct; the README is the only instance.'
          family: docs-describe-unshipped-surface
          round: 7
        - id: BR-17
          severity: Minor
          title: The owed real light-terminal "light (detected)" check exists only in atlas prose (atlas/define.md:555)
          detail: 'Once sdlc close runs, nothing owns it. Follow the M1 residue precedent (#76): open a follow-up with sdlc issue new, or attach it to an existing tracker, so the ARCH-MOCK live check has an owner.'
          family: deferred-obligation-lacks-a-tracker
          round: 7
      boundary: M3
      recipe: milestone-review
      blocked: false
    - "n": 8
      timestamp: "2026-09-18T11:19:17-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: addressed
          note: schemeState.choice is *schemeChoice{value, by choiceSource}; choiceSource admits only flag/saved/session (scheme.go:30-58).
          round: 8
        - id: BR-3
          disposition: addressed
          note: boardFooter and practiceChrome are gone; both tests call paintedBoardFooterForTest; the rationale now sits on boardFooterOutput (practice_output.go:155-189).
          round: 8
        - id: BR-4
          disposition: addressed
          note: At close every source and transition the paragraph describes has shipped; atlas/define.md:512-531 matches scheme.go and screen.go.
          round: 8
        - id: BR-5
          disposition: addressed
          note: The issue Log's M1 review entry states the renderers were moved and the fragment-tint assertions dropped, with the live halves ported; the Done-when evidence list cites that reconciliation.
          round: 8
        - id: BR-7
          disposition: addressed
          note: commandCtx uses loopKind {oneShot, piped, editor} (command.go:257-267); schemeArg is one field whose empty value means auto (scheme.go:164-166).
          round: 8
        - id: BR-9
          disposition: addressed
          note: Detection shipped in M3, so the atlas sentence about a loop asking the terminal is now true of loopEditor; no later-milestone claims are left.
          round: 8
        - id: BR-15
          disposition: addressed
          note: inking is removed; unfill writes inkOff when !coloured, which is correct because unfill runs before every change to coloured (language_row.go:24-38).
          round: 8
        - id: BR-16
          disposition: not-addressed
          note: 'Both named sentences are fixed, but the replacement at README.md:359 ("Every full-screen session asks") claims more than the code does: wantsBackground (rawterm.go:162) also requires colour on, -language-tint on and not -raw, while the editor opens on terminalUI && opt.tty alone (repl.go:297). So define -language-tint off and define -raw are full-screen sessions that never ask. Fix: state the predicate as the atlas does ("unless -language-tint off or -raw"), and change lessons.md "no narrower conditions than the code has" to "the code''s condition exactly, neither narrower nor broader". Prevalence in the family: 4.'
          round: 8
        - id: BR-17
          disposition: addressed
          note: '#77 ("confirm light detection in a real light terminal") is committed on origin/main (9b9f691), and atlas/define.md:555 cites it.'
          round: 8
      recipe: milestone-review
      blocked: false
---

# Gate ledger — tools#70 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-17T22:58:39-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `plan-restates-code` The plan restates the diff: full implementations, test cases listed in prose, call sites by line number
  The gate asks for named functions plus one strategy line per risky function. It does not block this plan, which can be carried out and whose facts were checked; compress in future plans rather than rewriting this one.
  (carried from plan-quality PQ-4, deferred to the boundary review)

## Round 2 — 2026-09-17T22:58:39-07:00 (claude) — passed

### Raised

- **BR-2** [Minor] `state-shape-admits-illegal-combinations` schemeState.withChoice accepts any schemeSource, so choice-by-detected, choice-by-default and empty-choice states are representable
  The spec's choice *{value, source: flag|saved|session} is implemented as choice plus chosenBy schemeSource with sourceDefault as the absent marker (cmd/define/scheme.go:30,49). M2's describeScheme will print this source to the user; give the choice its own source type or use *schemeChoice before M2 builds on it (ARCH-ORDER).
- **BR-3** [Minor] `tests-pin-a-shadow-of-the-live-path` Two live-edge footer invariants are tested through the test-only boardFooter, not production's boardFooterOutput
  TestBoardFooterPutsTheFormsOwnRowsFirst and the fitsABoard row count (play_loop_test.go:2417,3049) call boardFooter (render_helpers_test.go:82), which also now holds the only copy of the load-bearing rationale. Point them at paintedBoardFooterForTest, move the comment to boardFooterOutput, and delete boardFooter (ARCH-DRY). Pre-existing; M1 moved it into a test file.
- **BR-4** [Minor] `docs-describe-unshipped-surface` The atlas paragraph on paint-time shade describes saved/session sources and detect/forget as live in M1
  atlas/define.md:503-515 describes the terminal-reported value and the saved and session sources as if they worked now, but none has a production caller until M2/M3. The plan says docs should describe only what each milestone ships; add one clause saying so.
- **BR-5** [Minor] `spec-done-when-drifts-from-delivery` Done-when says dead paths are deleted with tint assertions ported; the renderers were moved to test helpers and fragment-tint assertions dropped
  The Log's Task 3 entry explains why, but the list of spec revisions the plan makes omits it, so the issue's Done-when (line 320) and Spec section on Repaint still claim what was not done.

## Round 3 — 2026-09-17T23:46:34-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — The fix belongs in future plans, and no lessons.md rule records it; Chunk 1's code blocks have already drifted from the code (the plan's own Revisions says so), which shows the cost. Minor, not blocking.

### Raised

- **BR-6** [Important] `ticked-step-lacks-its-evidence` M2 plan steps are ticked but the issue Log has no M2 entry
  Task 7 Step 6 says the WriteScheme symlink-to-regular-file behaviour is "noted in the Log"; it is not. Task 11 Step 4 claims the TestPTY conformance run PASSED and should name what ran and what was skipped, but the M1 Log records 13 TestPTY tests already failing, so a clean PASS contradicts it. The mutation record the Done-when requires is also missing. Add an M2 Log entry covering the symlinked-file behaviour, the pty run broken into passed, pre-existing failures and skipped, and the mutation list.
- **BR-7** [Minor] `state-shape-admits-illegal-combinations` commandCtx session/fullScreen and schemeArg auto/value allow combinations that mean nothing
  This is the 2nd finding in this family. Rule: when fields depend on each other, they should be ONE tagged value whose members are exactly the legal combinations. Remaining instances in #70: commandCtx.session plus fullScreen (fullScreen without session is representable), and schemeArg auto plus value (its zero value would save a blank line, which every later startup warns about). Neither is reachable today. Measured prevalence in #70: 3 instances; choice/chosenBy was fixed at M1. Fix: a loopKind enum {oneShot, piped, editor}, and a nil-able choice where nil means auto.
- **BR-8** [Minor] `store-access-bypasses-its-seam` The startup read skips schemePersister, and the precedence order is tested only through a pty
  run() resolves d.configDir and calls store.ReadScheme inline, while deps.schemePersister resolves it again for save and clear (ARCH-DRY). The flag, then saved, then default order lives in run() glue; "flag beats saved" and "garbled file warns once" are pinned only by TestSavedSchemeGovernsALookup through a real pty (ARCH-PURE). Fix: add load() to schemePersister and extract a pure initialSchemeState(flag, persister, warn) that can be unit-tested without a pty.

## Round 4 — 2026-09-18T00:00:08-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Forward-looking and still true: the 2026-09-18 Revisions entry concedes Chunk 2's code has drifted too, Chunk 3 still restates code, and no lessons.md rule records it. Minor, never blocks.
- BR-6 — addressed — The Log now has an M2 entry covering the symlinked FILE (rename replaces the link; the directory is kept, see TestClearSchemeRemovesOnlyWhatIsOurs), the pty run (1338 passed, 1 skipped, 14 failed, the 14 being the pre-existing set from M1) and the per-task mutation list.
- BR-7 — not-addressed — Both named instances are fixed (schemeArg: empty = auto; loopKind). But the class sweep the fix commit claims was not done: schemeState.detected + heard (scheme.go:60-63) is a third pair, and heard=false with a detected value, or heard=true with an empty one, is representable. The spec says detected *scheme. Enumerated #70 structs (schemeState, schemeChoice, schemeArg, commandCtx, tintPolicy); this is the only remaining instance, so prevalence in #70 is 4. Fix: detected store.Scheme with empty meaning not heard, drop heard.
- BR-8 — addressed — schemePersister.load plus the pure initialSchemeState(flag, persister, warn), used in run() at main.go:871. TestInitialSchemeState pins flag-beats-saved without a load and one warning for a garbled file, with no pty. Swapping the precedence in a scratch copy turned it red.

### Raised

- **BR-9** [Minor] `docs-describe-unshipped-surface` The atlas says /scheme's loopKind decides "does this loop ask the terminal", which no loop does until M3
  This is the 2nd finding in this family (BR-4 at M1 was the 1st). atlas/define.md:1016-1017, added in a4ec10d, drops the "(from M3)" tag that the loopKind code comment keeps. Rule: a doc written at milestone N describes only what N ships, and anything from a later milestone carries that milestone inline. Sweep: grep the milestone's doc diff for the later milestone's words (for #70: detect, ask, report, query, OSC, KeyBackground). That sweep over atlas and README finds only this hit; prevalence in #70 is 2.

## Round 5 — 2026-09-18T09:36:44-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — M3 Tasks 13-15 still carry full implementations, and the window adds no lessons.md rule telling future plans to name functions plus one strategy line. Record that rule at close; the plan itself needs no rewrite.

### Raised

- **BR-10** [Important] `tests-pin-a-shadow-of-the-live-path` The sitting's repaint after a background reply is unpinned: removing show() at play_loop.go:315-317 leaves the suite green
  This is the 2nd finding in family tests-pin-a-shadow-of-the-live-path. Rule: a consumer's test checks what that consumer itself outputs (the frame it paints, the answer it records, the notice it posts), never only the shared state it writes. PaintedTranscript is not evidence of a repaint, because it re-reads the holder when called. Consumers of KeyBackground: editor frame pinned (dropping draw() goes red, verified); router drag and drop-site notice pinned; sitting frame NOT pinned (TestASittingIgnoresABackgroundReply checks d.scheme.Scheme(); dropping show() stayed green, verified), and this is the path that paints --play's first question on a light terminal. TestAReplyDuringPlayReachesTheEditor's PaintedTranscript check repeats its effective() check. Fix: follow TestASittingPaintsInTheEditorsScheme (a tinted row on the sitting's screen), send the reply, and assert lastFrame of the tty carries languageLight. Record the rule in lessons.md.
- **BR-11** [Important] `ticked-step-lacks-its-evidence` Ticked M3 steps name evidence the record does not hold, including the three-terminal x two-appearance manual check
  This is the 2nd finding in family ticked-step-lacks-its-evidence; the 1st was M2's BR-6. Rule: tick a step that names evidence (a run result, "name what ran", a mutation list, a recorded matrix) only in the commit that writes that evidence where the step says it goes. If a step was done differently, add a Revisions entry saying what was done instead. Steps to sweep: Task 14 Step 4 (three 30 s fuzz runs); the mutation steps of Tasks 13-17 (the Log gives counts, not names as M1 and M2 did); Task 17 Step 3 (which TestPTY tests ran or skipped under CONFORMANCE_STRICT; TestPTYBackgroundDetection sits behind bilingualNativeProbe); Task 18 step 1 (full suite, -race, vet, linux build, strict results); Task 18 step 2 (the matrix; the atlas says "not itemised"). Done-when bullet 1 still claims Terminal.app, iTerm2 and Ghostty in both appearances. Bullet 6 claims Alt-] and #rrggbb are pinned "through each loop shell", but only the decoder and readInput test them. The README says those three terminals reply. Either record the matrix or add a Revisions entry that narrows bullet 1, and make the README match what is recorded. Record the rule in lessons.md.
- **BR-12** [Minor] `branch-survives-its-mutation` The paired-ink reset in unfill (language_row.go:34-37) is unpinned; removing inkOff leaves the suite green
  Verified by mutation. If it regressed, 235 or 252 ink would carry onto a producer's own background (the \x1b[42mc cell, column 3, in TestLanguageRowStylesAndExclusions) and onto uncoloured excluded answer cells. The atlas says the ink steps aside for either. Fix: assert cells[3].fg == -1 there, and add an uncoloured excluded cell with fg == -1.
- **BR-13** [Minor] `duplicated-logic-not-extracted` The KeyBackground intercept is copied in runEditor (replraw.go:553-561) and playSession (play_loop.go:311-318)
  ARCH-DRY. The neighbouring viewportGesture helper exists precisely so the two loops cannot disagree about a shared key. A one-line helper (apply the report through d.scheme.detect and report whether to repaint) would make the report rule a single source.

## Round 6 — 2026-09-18T09:55:13-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — lessons.md now carries "A plan names functions and one strategy line each, not their code (#70)", the rule the prior round asked for at close; the plan itself needs no rewrite.
- BR-10 — addressed — TestASittingRepaintsOnABackgroundReply reads the sitting's painted frame; removing show() turns it red (timeout). The editor-after-play test reads lastFrame; removing resume()'s repaint turns it red. Both verified by mutation in a scratch copy of HEAD.
- BR-11 — addressed — Log names every M3 mutation, fuzz run, strict set and boundary run (two spot-checked red); bullet 1 narrowed by a Log REVISION and plan Revisions; README no longer names terminals; the loop shells now test #hex and Alt-]. The separate distinguishing-evidence gap is raised as a new finding.
- BR-12 — addressed — TestTheInkStepsAsideWithTheTint pins the excluded cell and the producer background at fg -1; removing inkOff reddens both (verified).
- BR-13 — addressed — terminalReport (replraw.go:144-153) is the one report rule, used by runEditor and playSession.

### Raised

- **BR-14** [Important] `tests-pin-a-shadow-of-the-live-path` Done-when bullet 1 cites the operator check as evidence of detection, but the record cannot tell detection from a saved choice
  This is the 3rd finding in family tests-pin-a-shadow-of-the-live-path. Rule for the whole family: evidence for a path (a test assertion OR a manual check) must be something only that path can produce; the shared holder, PaintedTranscript's re-read, and a light tint that a saved choice also paints all fail this. The Log's evidence for "light -> 254, dark -> 236, no flag, no saved file" ends with "operator check". The only recorded detail of that check is the first run, and the Log says a saved light was in force then ("not yet /scheme auto'd"). The re-check is recorded only as "working", with no /scheme report. The atlas also calls that run "a light profile", while the Log describes a terminal with white default text. Sweep of the Done-when evidence list: only this item fails; the pty (detected) reports, the in-process light-reply frame and the late-reply repaint all distinguish detection. Fix: extend the lessons.md rule from tests to manual evidence. For this instance, either record one real-terminal /scheme report reading "(detected)" in each appearance after /scheme auto, or state that the live check did not establish detection, so detection's only evidence is the modelled pty terminal and the ARCH-MOCK live check is still owed. Either way, correct the atlas wording.

## Round 7 — 2026-09-18T11:08:59-07:00 (claude) — passed

### Disposed

- BR-14 — addressed — Log, atlas:547-558 and plan Revisions record a real /scheme reading "dark (detected)" after /scheme auto; light detection is recorded as owed; lessons.md extends the rule to manual evidence; the "light profile" wording is gone.

### Raised

- **BR-15** [Minor] `state-shape-admits-illegal-combinations` paintLanguageRow stores inking, which always equals filled && !coloured (language_row.go:26-27,34)
  This is the 3rd finding in family state-shape-admits-illegal-combinations. The line-26 comment states the derivation. unfill() runs before every change to coloured, so the flag is redundant. Scratch probe: dropping it and writing if !coloured { inkOff } left every tint, ink, row, selection and screen test green. Rule for the family: store only independent facts; compute a flag that follows from other state where it is read. Add this to lessons.md. Sweep of the M3 diff: this is the only derived flag.
- **BR-16** [Minor] `docs-describe-unshipped-surface` README scheme section: "suits a dark terminal by default" is out of date and "With nothing chosen ... asks" is too narrow (README.md:343,359)
  This is the 3rd finding in family docs-describe-unshipped-surface. wantsBackground asks whether or not a scheme is chosen; that is what lets /scheme auto show a detected value. And a session no longer just defaults to dark. Rule: when a milestone changes a behaviour, check every doc sentence about it against the code's actual condition. Docs describe exactly what ships: no future features, no stale defaults, no narrower conditions. Sweep: the atlas detection paragraph and -h are correct; the README is the only instance.
- **BR-17** [Minor] `deferred-obligation-lacks-a-tracker` The owed real light-terminal "light (detected)" check exists only in atlas prose (atlas/define.md:555)
  Once sdlc close runs, nothing owns it. Follow the M1 residue precedent (#76): open a follow-up with sdlc issue new, or attach it to an existing tracker, so the ARCH-MOCK live check has an owner.

## Round 8 — 2026-09-18T11:19:17-07:00 (claude) — passed

### Disposed

- BR-2 — addressed — schemeState.choice is *schemeChoice{value, by choiceSource}; choiceSource admits only flag/saved/session (scheme.go:30-58).
- BR-3 — addressed — boardFooter and practiceChrome are gone; both tests call paintedBoardFooterForTest; the rationale now sits on boardFooterOutput (practice_output.go:155-189).
- BR-4 — addressed — At close every source and transition the paragraph describes has shipped; atlas/define.md:512-531 matches scheme.go and screen.go.
- BR-5 — addressed — The issue Log's M1 review entry states the renderers were moved and the fragment-tint assertions dropped, with the live halves ported; the Done-when evidence list cites that reconciliation.
- BR-7 — addressed — commandCtx uses loopKind {oneShot, piped, editor} (command.go:257-267); schemeArg is one field whose empty value means auto (scheme.go:164-166).
- BR-9 — addressed — Detection shipped in M3, so the atlas sentence about a loop asking the terminal is now true of loopEditor; no later-milestone claims are left.
- BR-15 — addressed — inking is removed; unfill writes inkOff when !coloured, which is correct because unfill runs before every change to coloured (language_row.go:24-38).
- BR-16 — not-addressed — Both named sentences are fixed, but the replacement at README.md:359 ("Every full-screen session asks") claims more than the code does: wantsBackground (rawterm.go:162) also requires colour on, -language-tint on and not -raw, while the editor opens on terminalUI && opt.tty alone (repl.go:297). So define -language-tint off and define -raw are full-screen sessions that never ask. Fix: state the predicate as the atlas does ("unless -language-tint off or -raw"), and change lessons.md "no narrower conditions than the code has" to "the code's condition exactly, neither narrower nor broader". Prevalence in the family: 4.
- BR-17 — addressed — #77 ("confirm light detection in a real light terminal") is committed on origin/main (9b9f691), and atlas/define.md:555 cites it.

## Open findings

- **BR-16** [Minor] `docs-describe-unshipped-surface` README scheme section: "suits a dark terminal by default" is out of date and "With nothing chosen ... asks" is too narrow (README.md:343,359)
