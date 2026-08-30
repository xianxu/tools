---
gate: boundary-review
issue: 30
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-29T20:00:44-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: M1.1 enumerates four table-test cases in prose; compress to one strategy line per risky function
          detail: |-
            "a partial write continues the last line; a write containing \n\n appends
            an empty line; Frame clamps the offset at both ends; a viewport taller than
            the buffer pads" will be code within the hour. The chunk-boundary property
            for Write (any split of the same byte stream yields the same lines) is
            worth more than the four cases and is what they are groping at.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: test-cases-enumerated-in-prose
          round: 1
        - id: BR-2
          severity: Minor
          title: screen.lines and D3's exit transcript are unbounded
          detail: |-
            The buffer holds every line of a session and D3 replays all of it into the
            normal buffer on exit. A long session dumps thousands of lines at quit.
            State a cap, or state deliberately that there is none.
            (carried from plan-quality PQ-8, deferred to the boundary review)
          family: unbounded-buffer
          round: 1
        - id: BR-3
          severity: Minor
          title: M1.3's "only the destination changes" understates deleting cooked
          detail: |-
            cooked is a parameter of runEditor and submitLine; editorRig
            (editorloop_test.go:29) returns it and 47 test call sites pass it. The
            churn is mechanical, but it is a signature change across the suite, not a
            writer swap — say so, so the reviewer at the M1 boundary is not surprised
            by the diff size.
            (carried from plan-quality PQ-9, deferred to the boundary review)
          family: blast-radius-understated
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-08-29T20:00:44-07:00"
      agent: claude
      findings:
        - id: BR-4
          severity: Important
          title: TestRestoreLeavesTheAlternateScreen passes with leaveAlt and leaveMouse deleted from restore()
          detail: |-
            rawterm_test.go:110 builds `&rawSession{f: nil}`, so enterAlt returns at the
            nil-file guard and `alt` is never set; the assertion checks a field that was
            never true. Verified by reverting: replacing `r.leaveMouse()` and `r.leaveAlt()`
            in restore() (rawterm.go:44-48) leaves the entire in-process suite green (only
            the 12 git-dependent repo guards fail, as they do on an unmodified scratch copy).
            M1 done-when rows 3 and 3b therefore rest solely on conformance-tagged pty rows,
            which skip when no pty is available. Fix: type rawSession's escape-output target
            as io.Writer (keeping fd int for term.Restore) and assert the bytes AND the
            order, mouseOff before altScreenOff, in-process. ARCH-MOCK.
          family: unfalsifiable-test-pin
          round: 2
        - id: BR-5
          severity: Important
          title: Mode 1000 is enabled but its native X10 encoding is undecoded, so a click types characters into the line
          detail: |-
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
          family: decoder-consumes-whole-sequence
          round: 2
        - id: BR-6
          severity: Important
          title: Paint counts the frame in logical lines, not display rows, so a wide buffer line scrolls the terminal
          detail: |-
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
          family: frame-fits-the-terminal
          round: 2
        - id: BR-7
          severity: Minor
          title: 'TestRawLoopMessagePlacement''s wantErase row passes with eraseLine removed from both define: sites'
          detail: |-
            Removing eraseLine from replraw.go:372 and replraw.go:419 leaves the full suite
            green. The test gives stderr its own screen, so the erase has nothing to take
            back; in production stdout and stderr are one liveScreen. Separately, after
            view.Draw("", nil) blanks the live edge and the cmdNothing/cmdReplay branch skips
            writing the committed line, those two eraseLine prefixes look vestigial — while
            their comments still assert the pre-screen rationale. Decide which: pin it
            against a shared screen, or drop the gesture and the comment.
          family: unfalsifiable-test-pin
          round: 2
        - id: BR-8
          severity: Minor
          title: A whole-frame repaint per streamed delta, with no coalescing and no synchronized-output guard
          detail: |-
            liveScreen.Write (screen.go:270) repaints on every write, so a streamed answer
            redraws up to `rows` lines per token via cursorHome + eraseDown with no
            ESC[?2026h/l bracket. Related, same envelope: screen.lines grows unboundedly for
            the session and is dumped whole at exit. The plan declares no operating envelope
            for what is now a full-screen program. Not measured — flagged as undeclared
            rather than as observed flicker. ARCH-CONSTRAINTS.
          family: coalesce-ui-work
          round: 2
        - id: BR-9
          severity: Minor
          title: M1's Core-concepts table omits liveScreen, display, enterMouse/leaveMouse, decodeWheel, screen.Page
          detail: |-
            Those entities are named only in prose or in later Revisions entries, but the
            table is the greppable registry the boundary cross-check reads. Add rows with
            kind, path and status — liveScreen INTEGRATION, decodeWheel PURE — alongside
            winSize and terminalRows.
          family: plan-table-under-declares
          round: 2
        - id: BR-10
          severity: Minor
          title: Stale cooked-mode prose and small residue left by D4
          detail: |-
            editorloop_test.go:187 still says "before dropping back to cooked mode" for a
            mode D4 deleted, and its assertion pins \r\n bytes that screen.Write strips.
            Also: rawterm.go:134-192 writes terminal-state escapes to the STDIN handle while
            frames and the height probe go to stdout (a new assumption, correct only while
            both are the same tty); rawterm.go:222-238 nests four selects where a drain-then-
            send pair suffices with a single producer; watchResize never signal.Stops its
            channel; Paint discards the tty write error entirely.
          family: stale-rationale
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-08-29T20:08:39-07:00"
      agent: claude
      findings:
        - id: BR-11
          severity: Critical
          title: a legacy X10 mouse report injects three characters into the line being typed
          detail: |-
            replRaw enables ?1000h and ?1006h, but a terminal honouring 1000 without 1006
            sends "ESC [ M Cb Cx Cy". decodeEscape (key.go:130) finds M as the final byte,
            decodeWheel rejects the 3-byte sequence on len(seq) < 4, and the three
            coordinate bytes then decode as KeyRune. Probe-verified against the tree: a
            wheel-up at col 10 row 5 yields KeyUnknown plus the runes ` * %. This is the
            exact #14 under-consumption class the plan claims to have defended. Fix:
            special-case ESC[M before the parameter scan, require six bytes (return 0
            consumed if fewer), decode Cb-32 through the same wheel logic, and consume all
            six for a button. Add a decoder row and two fuzz seeds.
          family: escape-sequence-underconsumption
          round: 3
        - id: BR-12
          severity: Important
          title: Paint budgets the frame in logical lines and never consults a column width
          detail: |-
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
          family: app-owns-every-row
          round: 3
        - id: BR-13
          severity: Important
          title: TestRestoreLeavesTheAlternateScreen asserts nothing; deleting leaveAlt from restore stays green
          detail: |-
            The test builds &rawSession{f: nil}, so enterAlt() returns at its f == nil guard
            and r.alt is never set; the closing "if r.alt" is trivially false. Verified by
            mutation: with r.leaveAlt() removed from restore(), the whole default
            go test ./cmd/define/ suite passes. Done-when row 3 therefore rests only on the
            pty Fatal in TestPTYTranscriptIsPrintedOnExit, which skips without a pty (it
            skipped in this review environment), as do rows 3b, 5 and 6. Fix: widen
            rawSession.f to an io.Writer so restore's sequence and ORDER (mouseOff before
            altScreenOff) can be asserted in-process. Also drop the dead
            "var b strings.Builder; _ = b".
          family: vacuous-pin
          round: 3
        - id: BR-14
          severity: Important
          title: five sites still claim the raw loop wraps stdout in crlfWriter, which D5 removed
          detail: |-
            No caller of ask wraps stdout in crlfWriter any more (main.go:659,686,
            repl.go:299, replraw.go:223 all verified). Still asserting otherwise: ask.go:32,
            ask.go:168, askhighlight_test.go:240, atlas/define.md:708, atlas/define.md:747.
            TestHighlightingNestsInsideCRLFTranslation now pins a composition no production
            path builds. The repo guards cannot catch it because crlfWriter still exists for
            --play. M1.6 swept the sites it looked at rather than enumerating the class.
          family: stale-prose-after-removal
          round: 3
        - id: BR-15
          severity: Important
          title: two new tests race; go test -race fails where the base commit is clean
          detail: |-
            TestEditorResizeRedrawsForTheNewShape (editorloop_test.go:494) polls view.rows,
            written by recordDisplay.Resize on the loop goroutine; TestWatchResizeCoalesces
            (:530) polls `measured`, incremented by the watchResize goroutine. Both are
            reported by -race; the same command on base 168b1c9 reports no DATA RACE. Beyond
            tooling, waitFor spinning on an unsynchronised variable can spin to its 5s
            t.Fatal. Fix: mutex-guard recordDisplay and the counter, or hand the resize back
            over a buffered channel the test receives from.
          family: unsynchronised-test-observation
          round: 3
        - id: BR-16
          severity: Important
          title: every streamed delta triggers a full-screen clear and redraw, with no coalescing
          detail: |-
            runAsk writes once per streaming delta (ask.go:190); each write reaches
            liveScreen.Write -> repaint -> Paint, which emits ESC[H ESC[J plus the whole
            visible frame. Hundreds of deltas per answer means hundreds of full-screen
            erase-and-redraw cycles and megabytes to the tty for one answer. The plan
            declares no operating envelope for this path (ARCH-CONSTRAINTS). Fix: coalesce
            in liveScreen on a trailing timer with a forced flush on Draw/Page/Scroll/Stop,
            and write the budget into the plan.
          family: unbounded-ui-repaint
          round: 3
        - id: BR-17
          severity: Minor
          title: the offset clamp is spelled twice and Frame does not write its clamp back
          detail: |-
            Frame (screen.go:126) clamps a local `off`; Scroll (screen.go:147) clamps
            s.offset. After a widening resize s.offset can stay out of range, so the first
            wheel-down is a visual no-op. One clampedOffset() owner fixes both (ARCH-DRY).
          family: one-owner-per-invariant
          round: 3
        - id: BR-18
          severity: Minor
          title: the handedBack once-only guard in finish cannot fire today
          detail: |-
            Every finish() call site (replraw.go:238, :262, :290) returns immediately after
            it, so finish cannot run twice. Defensible as forward protection, but it
            currently reads as a guarantee nothing exercises.
          family: unreachable-guard
          round: 3
        - id: BR-19
          severity: Minor
          title: terminal-control sequences go to the stdin handle while frames go to stdout, errors discarded
          detail: |-
            rawterm.go:138,149,182,191 write altScreen/mouse sequences to rawSession.f, the
            stdin *os.File, while liveScreen paints to stdout; the fmt.Fprint errors are
            dropped and a failed write still sets r.alt/r.mouse. Write them to the handle
            the frames go to, or check the error.
          family: terminal-writes-off-seam
          round: 3
        - id: BR-20
          severity: Minor
          title: RenderLine's trailing cursor-back escape is stored in the buffer and the transcript
          detail: |-
            With the cursor mid-line at submit, replraw.go:314 buffers a committed line
            ending in ESC[<n>D. Harmless today because a \r\n follows, but
            TestEditorLoopWritesThroughAScreen only guards against ESC[K.
          family: control-sequence-buffered-as-text
          round: 3
        - id: BR-21
          severity: Minor
          title: README says the transcript is your scrollback "exactly as it was before"
          detail: |-
            The erased indicator and the live edge (prompt and menu) are deliberately absent
            from the transcript, so "exactly as it was" overstates what D3 restores.
          family: doc-overclaim
          round: 3
        - id: BR-22
          severity: Minor
          title: M1's Core concepts tables omit liveScreen, display, enterMouse/leaveMouse, terminalRows, decodeWheel and the four new KeyKinds
          detail: |-
            All are new entities M1 ships and all appear only in Revisions prose, not in the
            greppable tables a reader and the plan-table guards check. Add them with
            kind/location/status in a "## Revisions" entry.
          family: plan-table-incomplete
          round: 3
      boundary: M1
      blocked: true
    - "n": 4
      timestamp: "2026-08-29T20:48:44-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: M1.1's prose is unchanged and no chunk-boundary property test for Write was written; Minor, non-blocking.
          round: 4
        - id: BR-2
          disposition: addressed
          note: The plan now states the uncapped buffer deliberately, with the reason and a size estimate.
          round: 4
        - id: BR-3
          disposition: withdrawn
          note: 'Overtaken: M1.3 is built and its diff is in front of the reviewer, so the size warning is moot.'
          round: 4
        - id: BR-4
          disposition: addressed
          note: Verified by mutation — deleting both leaves from restore() now reddens three assertions.
          round: 4
        - id: BR-5
          disposition: addressed
          note: Verified by mutation — removing the decodeX10Mouse dispatch reddens TestDecodeX10Mouse and TestX10ClickTypesNothing.
          round: 4
        - id: BR-6
          disposition: addressed
          note: Both routes it named are fixed and mutation-verified; the cols==0 residual is carried on BR-12.
          round: 4
        - id: BR-7
          disposition: addressed
          note: Both eraseLine prefixes are gone and the comments now explain why the screen took that job.
          round: 4
        - id: BR-8
          disposition: addressed
          note: Throttle plus trailing flush, mutation-verified; the uncapped buffer is now a stated decision.
          round: 4
        - id: BR-9
          disposition: addressed
          note: Rows added — but the added Kind column broke the guard that reads the table; see the new Critical.
          round: 4
        - id: BR-10
          disposition: not-addressed
          note: 'Two of five remain: Paint still discards the tty write error, and pty_conformance_test.go:216 still states "render cooked, play raw" as the current fix.'
          round: 4
        - id: BR-11
          disposition: addressed
          note: Mutation-verified; the X10 payload is consumed whole or not at all.
          round: 4
        - id: BR-12
          disposition: not-addressed
          note: The third route it named survives — terminalWidth returns 0 below 20 columns or on probe failure, and with cols==0 displayRows charges one row per line and clipVisible returns the line unclipped; measured, a 10-row/80-column frame then needs 28 display rows.
          round: 4
        - id: BR-13
          disposition: addressed
          note: rawSession.control is an io.Writer and the restore protocol is asserted in process, bytes and order.
          round: 4
        - id: BR-14
          disposition: addressed
          note: All five sites fixed and the test now pins the production composition; issue 32's Spec still describes the old nesting, which D5a deliberately deferred.
          round: 4
        - id: BR-15
          disposition: addressed
          note: go test -race ./cmd/define/ is green; one remaining unsynchronised read is raised separately as the family rule.
          round: 4
        - id: BR-16
          disposition: addressed
          note: Mutation-verified — removing the throttle reddens the frame-count assertion.
          round: 4
        - id: BR-17
          disposition: addressed
          note: One clamp owner, and Frame writes back; the behavioural half has no pin (reverting to a local clamp leaves TestScreen green).
          round: 4
        - id: BR-18
          disposition: addressed
          note: sync.OnceFunc with the transcript stated as its reason; still unreachable today, which the comment now owns.
          round: 4
        - id: BR-19
          disposition: addressed
          note: Mode sequences go to the same stream as the frames, and a failed write no longer claims the state.
          round: 4
        - id: BR-20
          disposition: not-addressed
          note: The fix is present at replraw.go:304 but no test pins it — deleting the line leaves the whole suite green.
          round: 4
        - id: BR-21
          disposition: addressed
          note: README now says what the transcript keeps and what it deliberately drops.
          round: 4
        - id: BR-22
          disposition: addressed
          note: Rows added; the resulting guard breakage is the new Critical.
          round: 4
      findings:
        - id: BR-23
          severity: Critical
          title: go test ./... is red at HEAD on two plan-table guards, and the Log records it green
          detail: |-
            TestPlanTablesNameEntitiesThatExist fails on all fourteen M1 Pure-entities rows:
            plan.md:78 added a Kind column, which lands in the guard's third-cell Status slot,
            so every row is rejected as an out-of-vocabulary status AND skipped unchecked — the
            Core-concepts cross-check is blind to M1's whole table. TestPlanTableStatusMatchesTheChangeWindow
            fails on plan.md:167, the M2 Render row, because this window touched render.go
            without touching Render's declaration. Base 168b1c9 passes the first guard and
            skips the second. Fix the table shape (Status third, as the Integration table at
            plan.md:104 already is), decide how a not-yet-started milestone's modified rows
            should read, then re-run and re-record.
          family: verification-claim-unreproduced
          round: 4
        - id: BR-24
          severity: Important
          title: replRaw's exit sequence is pinned only by pty rows that skip, and the BR-20 fix is pinned by nothing
          detail: |-
            Third in this family. The rule: every behavioural claim M1 makes needs a pin that
            runs under plain go test ./cmd/define/; a conformance-tagged pty row is a live
            conformance check, not the pin. replRaw has no in-process caller, so enterAlt +
            enterMouse on entry and finish's Stop -> restore -> print-transcript ordering and
            once-only property rest solely on TestPTYTranscriptIsPrintedOnExit and
            TestPTYMouseTrackingIsAskedForAndGivenBack, both of which skipped here ("no pty
            available: operation not permitted"). Separately replraw.go:304's
            submitted.Cursor = len(submitted.Line) can be deleted with the suite still green.
            Extract finish's body over an interface and assert the order in process.
          family: unfalsifiable-test-pin
          round: 4
        - id: BR-25
          severity: Important
          title: atlas/ was not updated for the display-row budget, the paint clip, the repaint throttle or the X10 fallback
          detail: |-
            The rework commit touched atlas/define.md only for the two crlfWriter prose sites.
            "The screen" (atlas/define.md:265-340) does not say that the frame is budgeted in
            display rows, that buffer lines are CLIPPED to the terminal width at paint time
            while the transcript keeps the full text, or that a write now repaints at most
            once per 16 ms — it still asserts "A write REPAINTS" flatly. The X10 fallback and
            the rule it leaves behind are also absent, though 1000 and 1006 are named. All
            three are surface a reader of the atlas would be wrong about.
          family: docs-lag-new-surface
          round: 4
        - id: BR-26
          severity: Minor
          title: 'Three counters answer "how wide is this" differently: runes, a sentinel column count, and logical menu rows'
          detail: |-
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
          family: frame-fits-the-terminal
          round: 4
        - id: BR-27
          severity: Minor
          title: The wheel button-byte decode is spelled twice, in decodeWheel and decodeX10Mouse
          detail: |-
            Second in this family. The rule: one function owns "what does this mouse button
            byte mean" and every encoding calls it. key.go:201-207 and key.go:250-254 both
            spell b&64 for the wheel bit and b&3 for the direction. M2.2 adds button decoding,
            which would make it three spellings of one fact across two encodings. Extract
            wheelFromButton(b int) (Key, bool) now.
          family: one-owner-per-invariant
          round: 4
        - id: BR-28
          severity: Minor
          title: screen_test.go:457 reads tty.frames without the lock, and :464 reads l.pending without l.mu
          detail: |-
            Second in this family. The rule: a field written by a timer or loop goroutine is
            read only through its accessor. Every other site in the same test uses the locked
            tty.painted(); line 457 reaches the field directly while the trailing paint timer
            may be running. -race does not report it because the timer reliably fires after
            the read, which is exactly why the guarantee has to be structural.
          family: unsynchronised-test-observation
          round: 4
      boundary: M1
      blocked: true
    - "n": 5
      timestamp: "2026-08-29T21:18:49-07:00"
      agent: claude
      dispose:
        - id: BR-23
          disposition: not-addressed
          note: 'go test ./... is still RED at HEAD: TestPlanTableStatusMatchesTheChangeWindow now fails on the M2 Render row for the OPPOSITE reason — it was flipped to "unchanged" while render.go:172 (inside Render, 97-222) changed visibleLen to visibleCells in this window. The Kind-column half is fixed and TestPlanTablesNameEntitiesThatExist passes; nothing else in ./... fails.'
          round: 5
        - id: BR-12
          disposition: not-addressed
          note: 'The budget half landed (cols is real, displayRows charges wrapped height, the clip runs at paint time) but the frame still overflows: clipVisible''s cut cursor counts RUNES, so clipVisible(strings.Repeat("日",100), 80) returns 200 cells and a 10-line buffer of those needs 19 display rows in a 10-row terminal. Measured at HEAD. TestScreenFrameFitsTheTerminalInDisplayRows would catch it — all four of its fixtures are ASCII.'
          round: 5
        - id: BR-26
          disposition: not-addressed
          note: 'Third in family one-owner-per-invariant, and the RULE is still the deliverable rather than the sites. Rule: every cursor that walks a string against a column budget advances by cellWidth(r) — no site in the paint path may increment a column counter per rune, per byte, or per index. Enumeration measured at HEAD, 3 of 6 wrong: render.go:256 visibleCells CELLS ok; render.go:332,335 wrapText CELLS ok; screen.go:412 displayRows CELLS ok; screen.go:465 clipVisible RUNES wrong (cuts a 20-cell/30-rune line to 8 cells at width 12, and lets a 200-cell line through a width-80 clip); editor.go:227 RenderLine''s ESC[nD park counts runes for a move the terminal makes in columns; command.go:305 truncate counts runes and its doc comment still asserts "a column is a rune". Write the enumeration into the plan and sweep it, and add one wide-rune and one combining-mark row to TestScreenFrameFitsTheTerminalInDisplayRows so the class cannot come back.'
          round: 5
        - id: BR-20
          disposition: addressed
          note: 'Verified by reversion: replacing submitted.Cursor = len(submitted.Line) with a no-op reddens TestACommittedLineCarriesNoCursorEscape with "…\x1b[3D".'
          round: 5
        - id: BR-24
          disposition: addressed
          note: 'handBack/onceHandBack are named, take two small interfaces, and are pinned in process by two tests that ran here while every pty row skipped. Residue: replRaw''s own enterAlt/enterMouse on ENTRY are still pinned only by pty rows.'
          round: 5
        - id: BR-25
          disposition: addressed
          note: atlas/define.md gains "The screen" with the display-row budget, the clip, the cell-width owner, the 16 ms throttle and its trailing flush, the X10 fallback and handBack. One sentence now overclaims — "every clip reads that one function" is false while clipVisible cuts by rune.
          round: 5
        - id: BR-27
          disposition: addressed
          note: wheelFromButton is the single owner; decodeWheel and decodeX10Mouse both call it.
          round: 5
        - id: BR-28
          disposition: addressed
          note: countingWriter.painted() is used at every read site, recordDisplay is mutex-guarded with reader methods, and the watcher reports measurements over a channel. go test -race is green on the M1 suites.
          round: 5
        - id: BR-10
          disposition: addressed
          note: 'Mode sequences go to rawSession.control rather than the stdin handle; the nested selects are a drain-then-send pair with the single-producer reasoning stated; the missing signal.Stop is now a recorded consequence of the seam; the stale cooked-mode prose is gone. Still open, and folded into the Minor list: Paint discards the tty write error.'
          round: 5
        - id: BR-1
          disposition: not-addressed
          note: M1.1 still enumerates the four cases in prose, and the chunk-boundary property the finding actually asked for (any split of the same byte stream yields the same lines) has no test — FuzzScreenWriteDoesNotPanic only checks for panics.
          round: 5
      findings:
        - id: BR-29
          severity: Important
          title: handBack, onceHandBack and wheelFromButton are absent from M1's Core-concepts tables
          detail: 'This is the 2nd finding in family plan-table-incomplete (BR-9 and BR-22 are its siblings under plan-table-under-declares — three rounds, same class). Do NOT just add the three rows. The rule: the Core-concepts table is populated FROM THE DIFF, not from memory — every top-level declaration this window adds to a file the tables name gets a row. TestPlanTablesNameEntitiesThatExist already checks table-to-tree; the direction that keeps failing is tree-to-table, and repo_guard_test.go already has changedLines() and repoRoot() to build the inverse guard with. Measured at HEAD, the omissions are handBack, onceHandBack (replraw.go), wheelFromButton (key.go), plus screen.Lines, screen.eraseOpenLine, paintInterval, defaultRows/defaultCols — and the first three are exactly the deliverables round 3''s own Log names as its fixes for BR-24 and BR-27.'
          family: plan-table-incomplete
          round: 5
        - id: BR-30
          severity: Minor
          title: Paint's cursor-up omits the prompt's own display height, so a wrapping prompt is reprinted over the menu
          detail: 'This is the 2nd finding in family app-owns-every-row. Do NOT fix the instance. The rule: the frame''s row accounting is ONE pass — the same enumeration that budgets rows derives where the cursor must return to. Today screen.go:216-222 and screen.go:239-243 are two separate summations of the same quantity, and the second omits the prompt. Measured: Paint(&b, 10, 20, "> "+strings.Repeat("z",30), []string{"m1","m2"}) emits ESC[2A, which lands on the prompt''s SECOND row; the reprint then covers menu row m1 and leaves the cursor a row low — the same off-by-a-row limit the whole-frame redraw claims in its own comment to have deleted. Reachable on a terminal narrow enough that a partial /command wraps; terminalCols has no 20-column floor.'
          family: app-owns-every-row
          round: 5
        - id: BR-31
          severity: Minor
          title: The throttle test's Stop-flush assertion depends on wall-clock ordering it does not control
          detail: screen_test.go:465-470 writes "the last word", snapshots the frame count, then requires l.Stop() to paint. That only holds while the write lands inside paintInterval of the trailing flush waitFor just observed; if the goroutine is descheduled past 16 ms the write paints itself, pending is false, and Stop correctly does nothing while the test reports "Stop left a pending frame unpainted". Set l.painted deliberately (as TestLiveScreenShowsWhatIsWrittenToIt already does) rather than racing the interval.
          family: timing-dependent-assertion
          round: 5
      boundary: M1
      blocked: true
    - "n": 6
      timestamp: "2026-08-29T21:50:28-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: M1.1's prose still enumerates the four cases and the chunk-boundary property still has no test — FuzzScreenWriteDoesNotPanic checks only for panics. Minor, non-blocking.
          round: 6
        - id: BR-12
          disposition: addressed
          note: 'Verified at HEAD: clipVisible cuts by cells (100 CJK runes -> 80 cells/40 runes) and a 10-line CJK buffer needs exactly 10 rows in a 10-row terminal; the remaining overflow route is the live edge, raised separately.'
          round: 6
        - id: BR-23
          disposition: addressed
          note: 'Reproduced green at HEAD: go test ./... (102s) and go test -race ./cmd/define/ (109s); both plan-table guards pass, and the "guards read the commit" lesson is in workshop/lessons.md.'
          round: 6
        - id: BR-26
          disposition: not-addressed
          note: Site 5 of its own enumeration survives — editor.go:227 counts runes for a column move (NFD "cafe" cursor=2 emits ESC[3D for a 2-column move; "日本語" cursor=1 emits ESC[2D for 4) — and the wide-rune/combining-mark rows it asked for were not added to TestScreenFrameFitsTheTerminalInDisplayRows, whose four fixtures are still ASCII.
          round: 6
        - id: BR-29
          disposition: not-addressed
          note: 'The three named rows were added; the class was not. repo_guard_test.go is untouched in this window so the tree-to-table guard does not exist, and four symbols the finding itself enumerated still have no row: screen.Lines, screen.eraseOpenLine, paintInterval, defaultRows/defaultCols.'
          round: 6
        - id: BR-30
          disposition: addressed
          note: 'Verified at HEAD: Paint(&b,10,20,"> "+30xz,["m1","m2"]) now emits ESC[3A, the prompt''s first row, from one summation that also budgets the buffer. No test fails without it — raised as a separate finding.'
          round: 6
        - id: BR-31
          disposition: addressed
          note: interval is a field set to time.Hour in the throttle test, so the Stop-flush case no longer races the real 16 ms window.
          round: 6
      findings:
        - id: BR-32
          severity: Important
          title: The live edge is charged display rows but never budgeted by them, so the frame still overflows the terminal
          detail: 'This is the 3rd finding in family frame-fits-the-terminal. Earlier rounds fixed instances (BR-6 taught Paint display rows, BR-12 made cols real, BR-26 made the clip count cells). Do NOT fix this instance. The rule: the frame''s TOTAL display height is asserted against termRows before it is written — every component, buffer AND live edge — or it is not a budget. Today screen.go:226 clamps only the buffer''s share while prompt and menu are written unclipped and unlimited. Measured at HEAD: a 12-row/15-column terminal with "/" typed needs 17 display rows (opt.width is 0 below 20 columns, so truncate leaves 36-column menu rows to wrap three ways), and a 5-row/80-column terminal needs 6. The terminal then scrolls, every placed row moves, and M2''s RegionAt maps a click to the wrong buffer line — the exact property M1 exists to establish. Fix: clip the live edge by height in Paint, and make TestScreenFrameFitsTheTerminalInDisplayRows a property over shapes rather than four ASCII fixtures.'
          family: frame-fits-the-terminal
          round: 6
        - id: BR-33
          severity: Important
          title: Nothing asserts where Paint leaves the cursor; the whole cursor-up-and-reprint block is deletable with a green suite
          detail: 'This is the 4th finding in family unfalsifiable-test-pin. Earlier rounds fixed instances (BR-13, BR-7, BR-24). Do NOT just add a test for the cursor-up count. The rule: every byte Paint emits that POSITIONS the cursor is asserted in process — a frame is a placement, not a set of substrings. Verified by reversion in a scratch copy of HEAD: replacing menuRows+promptRows-1 with len(menu) at screen.go:244 (the exact BR-30 regression) leaves go test ./cmd/define/ green, and deleting the entire `if len(menu) > 0` block at screen.go:239-249 — which would leave the cursor at the end of the last menu row so every keystroke redraws in the wrong place — also leaves it green. TestScreenPaintSplitsTheHeight only checks that named lines and "PROMPT" appear and that the frame starts with home+erase. Decode the emitted frame into (row, col) and assert the final cursor position; that one assertion covers BR-30, the prompt reprint, and M2.5''s underline splice.'
          family: unfalsifiable-test-pin
          round: 6
      boundary: M1
      blocked: false
---

# Gate ledger — tools#30 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-29T20:00:44-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `test-cases-enumerated-in-prose` M1.1 enumerates four table-test cases in prose; compress to one strategy line per risky function
  "a partial write continues the last line; a write containing \n\n appends
  an empty line; Frame clamps the offset at both ends; a viewport taller than
  the buffer pads" will be code within the hour. The chunk-boundary property
  for Write (any split of the same byte stream yields the same lines) is
  worth more than the four cases and is what they are groping at.
  (carried from plan-quality PQ-7, deferred to the boundary review)
- **BR-2** [Minor] `unbounded-buffer` screen.lines and D3's exit transcript are unbounded
  The buffer holds every line of a session and D3 replays all of it into the
  normal buffer on exit. A long session dumps thousands of lines at quit.
  State a cap, or state deliberately that there is none.
  (carried from plan-quality PQ-8, deferred to the boundary review)
- **BR-3** [Minor] `blast-radius-understated` M1.3's "only the destination changes" understates deleting cooked
  cooked is a parameter of runEditor and submitLine; editorRig
  (editorloop_test.go:29) returns it and 47 test call sites pass it. The
  churn is mechanical, but it is a signature change across the suite, not a
  writer swap — say so, so the reviewer at the M1 boundary is not surprised
  by the diff size.
  (carried from plan-quality PQ-9, deferred to the boundary review)

## Round 2 — 2026-08-29T20:00:44-07:00 (claude) — BLOCKED

### Raised

- **BR-4** [Important] `unfalsifiable-test-pin` TestRestoreLeavesTheAlternateScreen passes with leaveAlt and leaveMouse deleted from restore()
  rawterm_test.go:110 builds `&rawSession{f: nil}`, so enterAlt returns at the
  nil-file guard and `alt` is never set; the assertion checks a field that was
  never true. Verified by reverting: replacing `r.leaveMouse()` and `r.leaveAlt()`
  in restore() (rawterm.go:44-48) leaves the entire in-process suite green (only
  the 12 git-dependent repo guards fail, as they do on an unmodified scratch copy).
  M1 done-when rows 3 and 3b therefore rest solely on conformance-tagged pty rows,
  which skip when no pty is available. Fix: type rawSession's escape-output target
  as io.Writer (keeping fd int for term.Restore) and assert the bytes AND the
  order, mouseOff before altScreenOff, in-process. ARCH-MOCK.
- **BR-5** [Important] `decoder-consumes-whole-sequence` Mode 1000 is enabled but its native X10 encoding is undecoded, so a click types characters into the line
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
- **BR-6** [Important] `frame-fits-the-terminal` Paint counts the frame in logical lines, not display rows, so a wide buffer line scrolls the terminal
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
- **BR-7** [Minor] `unfalsifiable-test-pin` TestRawLoopMessagePlacement's wantErase row passes with eraseLine removed from both define: sites
  Removing eraseLine from replraw.go:372 and replraw.go:419 leaves the full suite
  green. The test gives stderr its own screen, so the erase has nothing to take
  back; in production stdout and stderr are one liveScreen. Separately, after
  view.Draw("", nil) blanks the live edge and the cmdNothing/cmdReplay branch skips
  writing the committed line, those two eraseLine prefixes look vestigial — while
  their comments still assert the pre-screen rationale. Decide which: pin it
  against a shared screen, or drop the gesture and the comment.
- **BR-8** [Minor] `coalesce-ui-work` A whole-frame repaint per streamed delta, with no coalescing and no synchronized-output guard
  liveScreen.Write (screen.go:270) repaints on every write, so a streamed answer
  redraws up to `rows` lines per token via cursorHome + eraseDown with no
  ESC[?2026h/l bracket. Related, same envelope: screen.lines grows unboundedly for
  the session and is dumped whole at exit. The plan declares no operating envelope
  for what is now a full-screen program. Not measured — flagged as undeclared
  rather than as observed flicker. ARCH-CONSTRAINTS.
- **BR-9** [Minor] `plan-table-under-declares` M1's Core-concepts table omits liveScreen, display, enterMouse/leaveMouse, decodeWheel, screen.Page
  Those entities are named only in prose or in later Revisions entries, but the
  table is the greppable registry the boundary cross-check reads. Add rows with
  kind, path and status — liveScreen INTEGRATION, decodeWheel PURE — alongside
  winSize and terminalRows.
- **BR-10** [Minor] `stale-rationale` Stale cooked-mode prose and small residue left by D4
  editorloop_test.go:187 still says "before dropping back to cooked mode" for a
  mode D4 deleted, and its assertion pins \r\n bytes that screen.Write strips.
  Also: rawterm.go:134-192 writes terminal-state escapes to the STDIN handle while
  frames and the height probe go to stdout (a new assumption, correct only while
  both are the same tty); rawterm.go:222-238 nests four selects where a drain-then-
  send pair suffices with a single producer; watchResize never signal.Stops its
  channel; Paint discards the tty write error entirely.

## Round 3 — 2026-08-29T20:08:39-07:00 (claude) — BLOCKED

### Raised

- **BR-11** [Critical] `escape-sequence-underconsumption` a legacy X10 mouse report injects three characters into the line being typed
  replRaw enables ?1000h and ?1006h, but a terminal honouring 1000 without 1006
  sends "ESC [ M Cb Cx Cy". decodeEscape (key.go:130) finds M as the final byte,
  decodeWheel rejects the 3-byte sequence on len(seq) < 4, and the three
  coordinate bytes then decode as KeyRune. Probe-verified against the tree: a
  wheel-up at col 10 row 5 yields KeyUnknown plus the runes ` * %. This is the
  exact #14 under-consumption class the plan claims to have defended. Fix:
  special-case ESC[M before the parameter scan, require six bytes (return 0
  consumed if fewer), decode Cb-32 through the same wheel logic, and consume all
  six for a button. Add a decoder row and two fuzz seeds.
- **BR-12** [Important] `app-owns-every-row` Paint budgets the frame in logical lines and never consults a column width
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
- **BR-13** [Important] `vacuous-pin` TestRestoreLeavesTheAlternateScreen asserts nothing; deleting leaveAlt from restore stays green
  The test builds &rawSession{f: nil}, so enterAlt() returns at its f == nil guard
  and r.alt is never set; the closing "if r.alt" is trivially false. Verified by
  mutation: with r.leaveAlt() removed from restore(), the whole default
  go test ./cmd/define/ suite passes. Done-when row 3 therefore rests only on the
  pty Fatal in TestPTYTranscriptIsPrintedOnExit, which skips without a pty (it
  skipped in this review environment), as do rows 3b, 5 and 6. Fix: widen
  rawSession.f to an io.Writer so restore's sequence and ORDER (mouseOff before
  altScreenOff) can be asserted in-process. Also drop the dead
  "var b strings.Builder; _ = b".
- **BR-14** [Important] `stale-prose-after-removal` five sites still claim the raw loop wraps stdout in crlfWriter, which D5 removed
  No caller of ask wraps stdout in crlfWriter any more (main.go:659,686,
  repl.go:299, replraw.go:223 all verified). Still asserting otherwise: ask.go:32,
  ask.go:168, askhighlight_test.go:240, atlas/define.md:708, atlas/define.md:747.
  TestHighlightingNestsInsideCRLFTranslation now pins a composition no production
  path builds. The repo guards cannot catch it because crlfWriter still exists for
  --play. M1.6 swept the sites it looked at rather than enumerating the class.
- **BR-15** [Important] `unsynchronised-test-observation` two new tests race; go test -race fails where the base commit is clean
  TestEditorResizeRedrawsForTheNewShape (editorloop_test.go:494) polls view.rows,
  written by recordDisplay.Resize on the loop goroutine; TestWatchResizeCoalesces
  (:530) polls `measured`, incremented by the watchResize goroutine. Both are
  reported by -race; the same command on base 168b1c9 reports no DATA RACE. Beyond
  tooling, waitFor spinning on an unsynchronised variable can spin to its 5s
  t.Fatal. Fix: mutex-guard recordDisplay and the counter, or hand the resize back
  over a buffered channel the test receives from.
- **BR-16** [Important] `unbounded-ui-repaint` every streamed delta triggers a full-screen clear and redraw, with no coalescing
  runAsk writes once per streaming delta (ask.go:190); each write reaches
  liveScreen.Write -> repaint -> Paint, which emits ESC[H ESC[J plus the whole
  visible frame. Hundreds of deltas per answer means hundreds of full-screen
  erase-and-redraw cycles and megabytes to the tty for one answer. The plan
  declares no operating envelope for this path (ARCH-CONSTRAINTS). Fix: coalesce
  in liveScreen on a trailing timer with a forced flush on Draw/Page/Scroll/Stop,
  and write the budget into the plan.
- **BR-17** [Minor] `one-owner-per-invariant` the offset clamp is spelled twice and Frame does not write its clamp back
  Frame (screen.go:126) clamps a local `off`; Scroll (screen.go:147) clamps
  s.offset. After a widening resize s.offset can stay out of range, so the first
  wheel-down is a visual no-op. One clampedOffset() owner fixes both (ARCH-DRY).
- **BR-18** [Minor] `unreachable-guard` the handedBack once-only guard in finish cannot fire today
  Every finish() call site (replraw.go:238, :262, :290) returns immediately after
  it, so finish cannot run twice. Defensible as forward protection, but it
  currently reads as a guarantee nothing exercises.
- **BR-19** [Minor] `terminal-writes-off-seam` terminal-control sequences go to the stdin handle while frames go to stdout, errors discarded
  rawterm.go:138,149,182,191 write altScreen/mouse sequences to rawSession.f, the
  stdin *os.File, while liveScreen paints to stdout; the fmt.Fprint errors are
  dropped and a failed write still sets r.alt/r.mouse. Write them to the handle
  the frames go to, or check the error.
- **BR-20** [Minor] `control-sequence-buffered-as-text` RenderLine's trailing cursor-back escape is stored in the buffer and the transcript
  With the cursor mid-line at submit, replraw.go:314 buffers a committed line
  ending in ESC[<n>D. Harmless today because a \r\n follows, but
  TestEditorLoopWritesThroughAScreen only guards against ESC[K.
- **BR-21** [Minor] `doc-overclaim` README says the transcript is your scrollback "exactly as it was before"
  The erased indicator and the live edge (prompt and menu) are deliberately absent
  from the transcript, so "exactly as it was" overstates what D3 restores.
- **BR-22** [Minor] `plan-table-incomplete` M1's Core concepts tables omit liveScreen, display, enterMouse/leaveMouse, terminalRows, decodeWheel and the four new KeyKinds
  All are new entities M1 ships and all appear only in Revisions prose, not in the
  greppable tables a reader and the plan-table guards check. Add them with
  kind/location/status in a "## Revisions" entry.

## Round 4 — 2026-08-29T20:48:44-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — M1.1's prose is unchanged and no chunk-boundary property test for Write was written; Minor, non-blocking.
- BR-2 — addressed — The plan now states the uncapped buffer deliberately, with the reason and a size estimate.
- BR-3 — withdrawn — Overtaken: M1.3 is built and its diff is in front of the reviewer, so the size warning is moot.
- BR-4 — addressed — Verified by mutation — deleting both leaves from restore() now reddens three assertions.
- BR-5 — addressed — Verified by mutation — removing the decodeX10Mouse dispatch reddens TestDecodeX10Mouse and TestX10ClickTypesNothing.
- BR-6 — addressed — Both routes it named are fixed and mutation-verified; the cols==0 residual is carried on BR-12.
- BR-7 — addressed — Both eraseLine prefixes are gone and the comments now explain why the screen took that job.
- BR-8 — addressed — Throttle plus trailing flush, mutation-verified; the uncapped buffer is now a stated decision.
- BR-9 — addressed — Rows added — but the added Kind column broke the guard that reads the table; see the new Critical.
- BR-10 — not-addressed — Two of five remain: Paint still discards the tty write error, and pty_conformance_test.go:216 still states "render cooked, play raw" as the current fix.
- BR-11 — addressed — Mutation-verified; the X10 payload is consumed whole or not at all.
- BR-12 — not-addressed — The third route it named survives — terminalWidth returns 0 below 20 columns or on probe failure, and with cols==0 displayRows charges one row per line and clipVisible returns the line unclipped; measured, a 10-row/80-column frame then needs 28 display rows.
- BR-13 — addressed — rawSession.control is an io.Writer and the restore protocol is asserted in process, bytes and order.
- BR-14 — addressed — All five sites fixed and the test now pins the production composition; issue 32's Spec still describes the old nesting, which D5a deliberately deferred.
- BR-15 — addressed — go test -race ./cmd/define/ is green; one remaining unsynchronised read is raised separately as the family rule.
- BR-16 — addressed — Mutation-verified — removing the throttle reddens the frame-count assertion.
- BR-17 — addressed — One clamp owner, and Frame writes back; the behavioural half has no pin (reverting to a local clamp leaves TestScreen green).
- BR-18 — addressed — sync.OnceFunc with the transcript stated as its reason; still unreachable today, which the comment now owns.
- BR-19 — addressed — Mode sequences go to the same stream as the frames, and a failed write no longer claims the state.
- BR-20 — not-addressed — The fix is present at replraw.go:304 but no test pins it — deleting the line leaves the whole suite green.
- BR-21 — addressed — README now says what the transcript keeps and what it deliberately drops.
- BR-22 — addressed — Rows added; the resulting guard breakage is the new Critical.

### Raised

- **BR-23** [Critical] `verification-claim-unreproduced` go test ./... is red at HEAD on two plan-table guards, and the Log records it green
  TestPlanTablesNameEntitiesThatExist fails on all fourteen M1 Pure-entities rows:
  plan.md:78 added a Kind column, which lands in the guard's third-cell Status slot,
  so every row is rejected as an out-of-vocabulary status AND skipped unchecked — the
  Core-concepts cross-check is blind to M1's whole table. TestPlanTableStatusMatchesTheChangeWindow
  fails on plan.md:167, the M2 Render row, because this window touched render.go
  without touching Render's declaration. Base 168b1c9 passes the first guard and
  skips the second. Fix the table shape (Status third, as the Integration table at
  plan.md:104 already is), decide how a not-yet-started milestone's modified rows
  should read, then re-run and re-record.
- **BR-24** [Important] `unfalsifiable-test-pin` replRaw's exit sequence is pinned only by pty rows that skip, and the BR-20 fix is pinned by nothing
  Third in this family. The rule: every behavioural claim M1 makes needs a pin that
  runs under plain go test ./cmd/define/; a conformance-tagged pty row is a live
  conformance check, not the pin. replRaw has no in-process caller, so enterAlt +
  enterMouse on entry and finish's Stop -> restore -> print-transcript ordering and
  once-only property rest solely on TestPTYTranscriptIsPrintedOnExit and
  TestPTYMouseTrackingIsAskedForAndGivenBack, both of which skipped here ("no pty
  available: operation not permitted"). Separately replraw.go:304's
  submitted.Cursor = len(submitted.Line) can be deleted with the suite still green.
  Extract finish's body over an interface and assert the order in process.
- **BR-25** [Important] `docs-lag-new-surface` atlas/ was not updated for the display-row budget, the paint clip, the repaint throttle or the X10 fallback
  The rework commit touched atlas/define.md only for the two crlfWriter prose sites.
  "The screen" (atlas/define.md:265-340) does not say that the frame is budgeted in
  display rows, that buffer lines are CLIPPED to the terminal width at paint time
  while the transcript keeps the full text, or that a write now repaints at most
  once per 16 ms — it still asserts "A write REPAINTS" flatly. The X10 fallback and
  the rule it leaves behind are also absent, though 1000 and 1006 are named. All
  three are surface a reader of the atlas would be wrong about.
- **BR-26** [Minor] `frame-fits-the-terminal` Three counters answer "how wide is this" differently: runes, a sentinel column count, and logical menu rows
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
- **BR-27** [Minor] `one-owner-per-invariant` The wheel button-byte decode is spelled twice, in decodeWheel and decodeX10Mouse
  Second in this family. The rule: one function owns "what does this mouse button
  byte mean" and every encoding calls it. key.go:201-207 and key.go:250-254 both
  spell b&64 for the wheel bit and b&3 for the direction. M2.2 adds button decoding,
  which would make it three spellings of one fact across two encodings. Extract
  wheelFromButton(b int) (Key, bool) now.
- **BR-28** [Minor] `unsynchronised-test-observation` screen_test.go:457 reads tty.frames without the lock, and :464 reads l.pending without l.mu
  Second in this family. The rule: a field written by a timer or loop goroutine is
  read only through its accessor. Every other site in the same test uses the locked
  tty.painted(); line 457 reaches the field directly while the trailing paint timer
  may be running. -race does not report it because the timer reliably fires after
  the read, which is exactly why the guarantee has to be structural.

## Round 5 — 2026-08-29T21:18:49-07:00 (claude) — BLOCKED

### Disposed

- BR-23 — not-addressed — go test ./... is still RED at HEAD: TestPlanTableStatusMatchesTheChangeWindow now fails on the M2 Render row for the OPPOSITE reason — it was flipped to "unchanged" while render.go:172 (inside Render, 97-222) changed visibleLen to visibleCells in this window. The Kind-column half is fixed and TestPlanTablesNameEntitiesThatExist passes; nothing else in ./... fails.
- BR-12 — not-addressed — The budget half landed (cols is real, displayRows charges wrapped height, the clip runs at paint time) but the frame still overflows: clipVisible's cut cursor counts RUNES, so clipVisible(strings.Repeat("日",100), 80) returns 200 cells and a 10-line buffer of those needs 19 display rows in a 10-row terminal. Measured at HEAD. TestScreenFrameFitsTheTerminalInDisplayRows would catch it — all four of its fixtures are ASCII.
- BR-26 — not-addressed — Third in family one-owner-per-invariant, and the RULE is still the deliverable rather than the sites. Rule: every cursor that walks a string against a column budget advances by cellWidth(r) — no site in the paint path may increment a column counter per rune, per byte, or per index. Enumeration measured at HEAD, 3 of 6 wrong: render.go:256 visibleCells CELLS ok; render.go:332,335 wrapText CELLS ok; screen.go:412 displayRows CELLS ok; screen.go:465 clipVisible RUNES wrong (cuts a 20-cell/30-rune line to 8 cells at width 12, and lets a 200-cell line through a width-80 clip); editor.go:227 RenderLine's ESC[nD park counts runes for a move the terminal makes in columns; command.go:305 truncate counts runes and its doc comment still asserts "a column is a rune". Write the enumeration into the plan and sweep it, and add one wide-rune and one combining-mark row to TestScreenFrameFitsTheTerminalInDisplayRows so the class cannot come back.
- BR-20 — addressed — Verified by reversion: replacing submitted.Cursor = len(submitted.Line) with a no-op reddens TestACommittedLineCarriesNoCursorEscape with "…\x1b[3D".
- BR-24 — addressed — handBack/onceHandBack are named, take two small interfaces, and are pinned in process by two tests that ran here while every pty row skipped. Residue: replRaw's own enterAlt/enterMouse on ENTRY are still pinned only by pty rows.
- BR-25 — addressed — atlas/define.md gains "The screen" with the display-row budget, the clip, the cell-width owner, the 16 ms throttle and its trailing flush, the X10 fallback and handBack. One sentence now overclaims — "every clip reads that one function" is false while clipVisible cuts by rune.
- BR-27 — addressed — wheelFromButton is the single owner; decodeWheel and decodeX10Mouse both call it.
- BR-28 — addressed — countingWriter.painted() is used at every read site, recordDisplay is mutex-guarded with reader methods, and the watcher reports measurements over a channel. go test -race is green on the M1 suites.
- BR-10 — addressed — Mode sequences go to rawSession.control rather than the stdin handle; the nested selects are a drain-then-send pair with the single-producer reasoning stated; the missing signal.Stop is now a recorded consequence of the seam; the stale cooked-mode prose is gone. Still open, and folded into the Minor list: Paint discards the tty write error.
- BR-1 — not-addressed — M1.1 still enumerates the four cases in prose, and the chunk-boundary property the finding actually asked for (any split of the same byte stream yields the same lines) has no test — FuzzScreenWriteDoesNotPanic only checks for panics.

### Raised

- **BR-29** [Important] `plan-table-incomplete` handBack, onceHandBack and wheelFromButton are absent from M1's Core-concepts tables
  This is the 2nd finding in family plan-table-incomplete (BR-9 and BR-22 are its siblings under plan-table-under-declares — three rounds, same class). Do NOT just add the three rows. The rule: the Core-concepts table is populated FROM THE DIFF, not from memory — every top-level declaration this window adds to a file the tables name gets a row. TestPlanTablesNameEntitiesThatExist already checks table-to-tree; the direction that keeps failing is tree-to-table, and repo_guard_test.go already has changedLines() and repoRoot() to build the inverse guard with. Measured at HEAD, the omissions are handBack, onceHandBack (replraw.go), wheelFromButton (key.go), plus screen.Lines, screen.eraseOpenLine, paintInterval, defaultRows/defaultCols — and the first three are exactly the deliverables round 3's own Log names as its fixes for BR-24 and BR-27.
- **BR-30** [Minor] `app-owns-every-row` Paint's cursor-up omits the prompt's own display height, so a wrapping prompt is reprinted over the menu
  This is the 2nd finding in family app-owns-every-row. Do NOT fix the instance. The rule: the frame's row accounting is ONE pass — the same enumeration that budgets rows derives where the cursor must return to. Today screen.go:216-222 and screen.go:239-243 are two separate summations of the same quantity, and the second omits the prompt. Measured: Paint(&b, 10, 20, "> "+strings.Repeat("z",30), []string{"m1","m2"}) emits ESC[2A, which lands on the prompt's SECOND row; the reprint then covers menu row m1 and leaves the cursor a row low — the same off-by-a-row limit the whole-frame redraw claims in its own comment to have deleted. Reachable on a terminal narrow enough that a partial /command wraps; terminalCols has no 20-column floor.
- **BR-31** [Minor] `timing-dependent-assertion` The throttle test's Stop-flush assertion depends on wall-clock ordering it does not control
  screen_test.go:465-470 writes "the last word", snapshots the frame count, then requires l.Stop() to paint. That only holds while the write lands inside paintInterval of the trailing flush waitFor just observed; if the goroutine is descheduled past 16 ms the write paints itself, pending is false, and Stop correctly does nothing while the test reports "Stop left a pending frame unpainted". Set l.painted deliberately (as TestLiveScreenShowsWhatIsWrittenToIt already does) rather than racing the interval.

## Round 6 — 2026-08-29T21:50:28-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — M1.1's prose still enumerates the four cases and the chunk-boundary property still has no test — FuzzScreenWriteDoesNotPanic checks only for panics. Minor, non-blocking.
- BR-12 — addressed — Verified at HEAD: clipVisible cuts by cells (100 CJK runes -> 80 cells/40 runes) and a 10-line CJK buffer needs exactly 10 rows in a 10-row terminal; the remaining overflow route is the live edge, raised separately.
- BR-23 — addressed — Reproduced green at HEAD: go test ./... (102s) and go test -race ./cmd/define/ (109s); both plan-table guards pass, and the "guards read the commit" lesson is in workshop/lessons.md.
- BR-26 — not-addressed — Site 5 of its own enumeration survives — editor.go:227 counts runes for a column move (NFD "cafe" cursor=2 emits ESC[3D for a 2-column move; "日本語" cursor=1 emits ESC[2D for 4) — and the wide-rune/combining-mark rows it asked for were not added to TestScreenFrameFitsTheTerminalInDisplayRows, whose four fixtures are still ASCII.
- BR-29 — not-addressed — The three named rows were added; the class was not. repo_guard_test.go is untouched in this window so the tree-to-table guard does not exist, and four symbols the finding itself enumerated still have no row: screen.Lines, screen.eraseOpenLine, paintInterval, defaultRows/defaultCols.
- BR-30 — addressed — Verified at HEAD: Paint(&b,10,20,"> "+30xz,["m1","m2"]) now emits ESC[3A, the prompt's first row, from one summation that also budgets the buffer. No test fails without it — raised as a separate finding.
- BR-31 — addressed — interval is a field set to time.Hour in the throttle test, so the Stop-flush case no longer races the real 16 ms window.

### Raised

- **BR-32** [Important] `frame-fits-the-terminal` The live edge is charged display rows but never budgeted by them, so the frame still overflows the terminal
  This is the 3rd finding in family frame-fits-the-terminal. Earlier rounds fixed instances (BR-6 taught Paint display rows, BR-12 made cols real, BR-26 made the clip count cells). Do NOT fix this instance. The rule: the frame's TOTAL display height is asserted against termRows before it is written — every component, buffer AND live edge — or it is not a budget. Today screen.go:226 clamps only the buffer's share while prompt and menu are written unclipped and unlimited. Measured at HEAD: a 12-row/15-column terminal with "/" typed needs 17 display rows (opt.width is 0 below 20 columns, so truncate leaves 36-column menu rows to wrap three ways), and a 5-row/80-column terminal needs 6. The terminal then scrolls, every placed row moves, and M2's RegionAt maps a click to the wrong buffer line — the exact property M1 exists to establish. Fix: clip the live edge by height in Paint, and make TestScreenFrameFitsTheTerminalInDisplayRows a property over shapes rather than four ASCII fixtures.
- **BR-33** [Important] `unfalsifiable-test-pin` Nothing asserts where Paint leaves the cursor; the whole cursor-up-and-reprint block is deletable with a green suite
  This is the 4th finding in family unfalsifiable-test-pin. Earlier rounds fixed instances (BR-13, BR-7, BR-24). Do NOT just add a test for the cursor-up count. The rule: every byte Paint emits that POSITIONS the cursor is asserted in process — a frame is a placement, not a set of substrings. Verified by reversion in a scratch copy of HEAD: replacing menuRows+promptRows-1 with len(menu) at screen.go:244 (the exact BR-30 regression) leaves go test ./cmd/define/ green, and deleting the entire `if len(menu) > 0` block at screen.go:239-249 — which would leave the cursor at the end of the last menu row so every keystroke redraws in the wrong place — also leaves it green. TestScreenPaintSplitsTheHeight only checks that named lines and "PROMPT" appear and that the frame starts with home+erase. Decode the emitted frame into (row, col) and assert the final cursor position; that one assertion covers BR-30, the prompt reprint, and M2.5's underline splice.

## Open findings

- **BR-1** [Minor] `test-cases-enumerated-in-prose` M1.1 enumerates four table-test cases in prose; compress to one strategy line per risky function
- **BR-26** [Minor] `frame-fits-the-terminal` Three counters answer "how wide is this" differently: runes, a sentinel column count, and logical menu rows
- **BR-29** [Important] `plan-table-incomplete` handBack, onceHandBack and wheelFromButton are absent from M1's Core-concepts tables
- **BR-32** [Important] `frame-fits-the-terminal` The live edge is charged display rows but never budgeted by them, so the frame still overflows the terminal
- **BR-33** [Important] `unfalsifiable-test-pin` Nothing asserts where Paint leaves the cursor; the whole cursor-up-and-reprint block is deletable with a green suite
