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
    - "n": 7
      timestamp: "2026-08-29T22:06:21-07:00"
      agent: claude
      boundary: M1
      blocked: false
      protocol_error: no valid findings block
    - "n": 8
      timestamp: "2026-08-29T22:28:12-07:00"
      agent: claude
      dispose:
        - id: BR-32
          disposition: addressed
          note: 'Verified by reversion at HEAD: restoring the menuRows summation in place of fitMenu reddens "more menu than terminal" (8 rows in 5) and "a terminal too short for anything" (3 in 2). readFrame asserts total height, so the rule is now enforced rather than the instance.'
          round: 8
        - id: BR-33
          disposition: addressed
          note: 'Verified by three reversions at HEAD: len(menu) for menuRows+promptRows-1 reddens, deleting the whole "if len(menu) > 0" block reddens five subtests, and menuRows+promptRows reddens three. The frame is decoded into (row, col) as the finding asked.'
          round: 8
        - id: BR-29
          disposition: not-addressed
          note: Unchanged since round 4 — repo_guard_test.go is untouched across the whole window, so the tree-to-table guard still does not exist, and screen.Lines, screen.eraseOpenLine, paintInterval and defaultRows/defaultCols still have no rows.
          round: 8
        - id: BR-26
          disposition: not-addressed
          note: 'Site 5 of its own enumeration survives: editor.go:227 still computes the ESC[nD park as len([]rune(sug)) + (len(e.Line) - e.Cursor), a rune count for a move the terminal makes in columns. The CJK row was added to the frame test; the combining-mark row was not.'
          round: 8
        - id: BR-1
          disposition: not-addressed
          note: M1.1's prose still enumerates the four cases and no chunk-boundary property test for Write exists; FuzzScreenWriteDoesNotPanic checks only for panics. Minor, non-blocking.
          round: 8
      findings:
        - id: BR-34
          severity: Important
          title: Done-when row 1b names TestScreenFrameFitsTheTerminalInDisplayRows, which this window's own commit renamed away
          detail: 'This is the 3rd finding in family plan-table-incomplete. Earlier rounds fixed instances (BR-9 and BR-22 added missing rows, BR-29''s three rows were added). Do NOT just rename the cell. The rule that covers all of them: every symbol a plan names — in ANY column of ANY of its tables, including Done-when''s "pinned by" — must resolve to a declaration in the tree, and every top-level declaration the window adds to a file the tables name must have a row; one guard, both directions, all columns. Measured at HEAD: I checked all 19 test names across M1''s Done-when and exactly one is missing — plan.md:150 names TestScreenFrameFitsTheTerminalInDisplayRows, renamed to TestPaintFitsTheTerminalAndParksTheCursor by aa4fe94, the commit under review. TestPlanTablesNameEntitiesThatExist exists and reads only the Core-concepts table, so the suite stayed green; the plan''s own Revisions already records this class twice ("M1 done-when, two rows named tests that do not exist"), which is why the deliverable is the widened guard rather than the edit.'
          family: plan-table-incomplete
          round: 8
        - id: BR-35
          severity: Important
          title: visibleCells and clipVisible each hand-roll the CSI grammar that scanEscape already owns
          detail: 'This is the 3rd finding in family one-owner-per-invariant. Earlier rounds fixed instances (BR-17 the offset clamp spelled twice, BR-27 the wheel button byte spelled twice). Do NOT just rewrite the two functions. The rule: the escape-sequence grammar has exactly one owner — sgr.go:85 scanEscape — and any site that walks a styled string skips sequences through it rather than re-deriving "ESC, then optional [, then params, then a final byte in 0x40-0x7E". Enumeration measured at HEAD, 2 of 4 non-test sites wrong: highlightwriter.go:88 defers to scanEscape (ok); render_test.go:102 defers to it and its comment says why (ok); render.go:238 visibleCells hand-rolls inEsc/inCSI, and its inCSI branch is NEW in this window; screen.go:484 clipVisible hand-rolls the same state machine plus a `styled` flag. screen_test.go:334 readFrame is a third spelling but legitimately INTERPRETS rather than skips, so it is out of scope for the sweep. The three agree today, which is what makes this cheap now and expensive later: M2.5 splices an underline through sgrState into text clipVisible cuts, so a fourth reading of the same grammar lands exactly where a divergence becomes a rendering bug (ARCH-DRY).'
          family: one-owner-per-invariant
          round: 8
        - id: BR-36
          severity: Minor
          title: The new placement test asserts the cursor column exactly but the row only as "not the last one"
          detail: screen_test.go:453 checks got.cursorRow >= got.rows-1. An off-by-one UPWARD in the walk-back is caught only where the buffer is empty, by readFrame's "moved the cursor above the screen" guard; I confirmed the "a menu under the prompt" fixture passes with menuRows+promptRows. Asserting the cursor is on the prompt's first row (rows - promptRows - menuRows) closes it. Separately, screen_test.go:449's wantCol %= termCols disagrees with readFrame's deferred wrap for a prompt exactly a multiple of the width — a false failure waiting for a fixture, not a false pass.
          family: unfalsifiable-test-pin
          round: 8
      boundary: M1
      blocked: false
    - "n": 9
      timestamp: "2026-08-30T11:42:28-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: M1.1's row still enumerates the four cases verbatim and no chunk-boundary property test for screen.Write exists (screen_test.go has only FuzzScreenWriteDoesNotPanic); M1 is closed, so this is now a carried Minor.
          round: 9
      findings:
        - id: BR-37
          severity: Important
          title: liveScreen.WriteRegions/RegionAtRow — the production click join is pinned by nothing (verified by mutation)
          detail: |-
            Swapping `l.s.addRegions(rs)` and `l.s.Write([]byte(text))` in screen.go:551
            leaves `go test ./cmd/define/` green, though in production it shifts every
            region forward by the entry's line count. 6th in the family: state the rule
            — a "pinned by" claim holds only when mutating the implementing code reddens
            a named test, and a double may not stand in for the object joining two
            separately-pinned halves — then write the enumeration (every Integration-points
            row names the test that runs it) and sweep it this round.
          family: unfalsifiable-test-pin
          round: 9
        - id: BR-38
          severity: Important
          title: The region-registry guard restates RegionKind's extent, so Done-when 7 cannot fire
          detail: |-
            editorloop_test.go:851 loops `kind <= RegionOriginLang`; RegionKind has no
            count sentinel, so a third kind is never exercised and `clicked`'s switch has
            no default. Second instance in the same window: originLineRange (render.go:377)
            re-derives the section boundary by an all-caps heuristic that `e.Sections`
            already owns. 4th/5th in the family: state the rule — the extent and structure
            of a declared set have one owner and every guard derives from it, as
            TestEveryEnabledMouseModeIsDecoded already does with mouseOn — and sweep both.
          family: one-owner-per-invariant
          round: 9
        - id: BR-39
          severity: Important
          title: README.md and atlas/define.md document none of M2's delivered surface
          detail: |-
            README.md is untouched in the window; atlas/define.md changed 11 lines, all
            the console side-quest, and still refers to clicks in the future tense
            (:283, :326, :350). Missing: click-to-play, ORIGIN-language click, the
            underline mark, the scrolled-past degrade, the region registry, writeRendered's
            seam. 2nd in the family: the cause is that M1.6 made the sweep a TASK and
            M2.1-M2.6 carries no docs row — fix that, ideally as a repo_guard_test.go
            window guard, not this instance.
          family: docs-lag-new-surface
          round: 9
        - id: BR-40
          severity: Minor
          title: markClickable emits the underline twice when a span begins at an escape
          detail: |-
            Measured: markClickable("\x1b[1;36mpotassium\x1b[0m is a metal", {Col:0,Width:9})
            yields "\x1b[4m\x1b[1;36m\x1b[4mpotassium...". The column trigger is re-tested
            after each byte step. Idempotent, so cosmetic; fire it once per column.
          family: column-trigger-fires-per-byte
          round: 9
        - id: BR-41
          severity: Minor
          title: screen.LineAt is tabled PURE but clamps and writes back s.offset via Frame()
          family: pure-label-hides-mutation
          round: 9
      boundary: M2
      blocked: true
    - "n": 10
      timestamp: "2026-08-30T12:15:57-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: M1.1's row still enumerates the four cases and no chunk-boundary property exists for screen.Write; M1 is closed, so this stays a carried Minor.
          round: 10
        - id: BR-37
          disposition: addressed
          note: 'Verified by mutation: swapping addRegions/Write reddens TestLiveScreenJoinsRegionsToTheLinesTheyWereRenderedFor; writeRendered''s seam also reddens when disabled. The class enumeration it demanded is raised separately as the 7th in the family.'
          round: 10
        - id: BR-38
          disposition: addressed
          note: numRegionKinds verified by mutation (a third kind reddens all three guards); originLineRange now derives from e.Sections, though reverting it leaves the suite green.
          round: 10
        - id: BR-39
          disposition: addressed
          note: README gains the clickable section, atlas gains "## Clickable regions", and TestAtlasDescribesEveryRegionKind derives from numRegionKinds - verified red for an undescribed third kind.
          round: 10
        - id: BR-40
          disposition: not-addressed
          note: 'Re-measured at HEAD: markClickable("\x1b[1;36mpotassium\x1b[0m is a metal", {Col:0,Width:9}) still yields "\x1b[4m\x1b[1;36m\x1b[4mpotassium...".'
          round: 10
        - id: BR-41
          disposition: not-addressed
          note: The plan's M2 row still tables screen.LineAt as PURE; Frame still clamps and writes back s.offset. C-1 is the cost of that unstated write.
          round: 10
      findings:
        - id: BR-42
          severity: Critical
          title: screen.Frame's fast path returns without clamping, so the click map detaches from the text when the viewport grows
          detail: |-
            5th in the family, so the rule is the deliverable: a derived invariant is
            re-established on every path that reads it, not only on the path that calls
            its owner. Measured on the real liveScreen — 20 lines at 12 rows, Page(10)
            then Resize(30,80): offset stays 9, topLine = -9, and RegionAtRow(0,0) on
            the row that actually shows the headword answers nothing while its underline
            is painted 9 rows lower. Three routine triggers: resize taller while scrolled
            back, the suggestion menu closing, a wrapped prompt killed with Ctrl-U.
            Hoisting s.clamp() above the early return fixes the repro and keeps the suite
            green — which is also the second half of the finding, since nothing pins it.
            Sweep: every early return in screen.go that precedes clamp, and every reader
            of s.offset/s.rows outside Paint. ARCH-PURE: Frame/LineAt are tabled PURE
            while writing back s.offset, which is what hides the missing clamp.
          family: one-owner-per-invariant
          round: 10
        - id: BR-43
          severity: Important
          title: BR-37's enumeration was never written, and two fixes in the same commit ship with nothing that fails without them
          detail: |-
            7th in the family, across 7 rounds on this issue. The rule was stated and the
            named instance pinned, but: Done-when row 1 still names only the four tests
            that stayed green under last round's mutation, and never names
            TestLiveScreenJoinsRegionsToTheLinesTheyWereRenderedFor; reverting
            originLineRange to the exact all-caps heuristic BR-38 named leaves
            `go test ./cmd/define/` green (measured, 107s); and C-1's clamp is green with
            and without its fix. One row IS filled and I checked it: disabling
            writeRendered's regionWriter branch reddens TestALookupHandsItsRegionsToTheScreen.
            Deliverable: for every M2 Core-concepts row and every Done-when "pinned by"
            cell, name the test and record the mutation that reddens it.
          family: unfalsifiable-test-pin
          round: 10
        - id: BR-44
          severity: Minor
          title: decodeWheel's and decodeX10Mouse's comments still say a click stays KeyUnknown, which stopped being true in this window
          detail: |-
            key.go:186 and :196 say the decoder "answers only the WHEEL" and that "a click
            therefore stays KeyUnknown — consumed whole and inert"; key.go:344 repeats it
            for the X10 path. Both functions now return KeyClick, and the name decodeWheel
            is a misnomer for a decoder that also decodes presses. 2nd in the family: the
            rule is that a comment stating what a function does NOT do is swept in the
            commit that makes it do it — sweep both sites and the name.
          family: stale-rationale
          round: 10
        - id: BR-45
          severity: Minor
          title: addRegions shifts base for a partial line but not Col, and the test that looks like it covers this passes Col pre-offset
          detail: |-
            screen.go:143 decrements base when s.partial, so a render starting mid-line
            lands on the right LINE — but Col is left relative to the render, not to the
            buffer line, so the regions sit in the wrong columns. screen_test.go:738 seems
            to cover the branch and instead supplies Col: 12 already offset by hand, so it
            asserts the caller's arithmetic. Not reachable today (the loop writes "\r\n"
            before an entry), but the comment documents one contract and the test another.
            Either enforce the precondition or shift Col by visibleCells of the open line.
          family: vacuous-pin
          round: 10
      boundary: M2
      blocked: true
    - "n": 11
      timestamp: "2026-08-30T12:49:38-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'Still open: no chunk-boundary property over screen.Write exists — TestScreenWriteBuildsLines is still the four enumerated cases.'
          round: 11
        - id: BR-40
          disposition: not-addressed
          note: 'Reproduced at HEAD: markClickable("\x1b[1;36mpotassium\x1b[0m is a metal", {Col:0,Width:9}) = "\x1b[4m\x1b[1;36m\x1b[4mpotassium\x1b[24m\x1b[0m is a metal".'
          round: 11
        - id: BR-41
          disposition: not-addressed
          note: LineAt now reaches clamp through visible() rather than Frame(), so the write-back under a PURE table row remains.
          round: 11
        - id: BR-42
          disposition: addressed
          note: 'Verified by revert: screen.go at 8f6a458 reddens TestClickMapSurvivesTheViewportGrowing with the exact recorded message.'
          round: 11
        - id: BR-43
          disposition: addressed
          note: 'Enumeration exists and holds: seven sampled rows each reddened under the mutation the table names.'
          round: 11
        - id: BR-44
          disposition: not-addressed
          note: key.go:186/196 still say the decoder answers only the wheel and a click stays KeyUnknown; key.go:344 repeats it; decodeWheel is still the name.
          round: 11
        - id: BR-45
          disposition: not-addressed
          note: 'Reproduced: a mid-line render with Col relative to the render resolves at buffer column 0, not at the open line''s width.'
          round: 11
      findings:
        - id: BR-46
          severity: Critical
          title: A multi-word entry's clickable headword is only its first field, so the click plays a different word
          detail: |-
            regionsIn (render.go:307) builds the headword region from individual HeadWord
            tokens and stamps Word: e.Headword(), which parseHead (parse.go:436) fills
            from fields[0] alone. Measured on the committed `hot dog` fixture: a bare
            Enter asks the CDN for hot_dog_en_us_1.mp3, while a click on the same
            entry's underlined headword asks for hot_en_us_1.mp3, hot_en_us_2.mp3 and
            the hot-- fallbacks. `a priori` produces a single region for the letter "a".
            RegionOriginLang carries the same Word, so a French replay on such an entry
            is also the wrong word. The mark is wrong with it: only "hot" is underlined
            in a head line reading "hot dog". Derive the target from the same source the
            gesture it shortcuts uses — lookupAndRender holds the lookup key — and widen
            the span to the whole head phrase. The missing property: for every headword
            region, the first audio candidate for r.Word equals the first candidate a
            bare-Enter replay of that entry produces.
          family: shortcut-rederives-its-target
          round: 11
        - id: BR-47
          severity: Important
          title: The terminal-row to viewport-row joint is pinned by nothing, and runEditor never runs against a real liveScreen
          detail: |-
            8th in this family. Do NOT fix only this instance — the rule the family keeps
            producing is that the enumeration must be of JOINTS, not entities: every place
            two separately-pinned layers exchange a value across a coordinate or unit
            boundary gets a row, and a row earns it only when a test drives both real
            objects. Concretely: newLiveScreen appears in no editor-loop test, both click
            action tests script recordDisplay.RegionAtRow's answer with row/col of the
            test's own choosing, and TestLiveScreenJoins… calls RegionAtRow directly with
            hand-written regions. Nothing asserts Key.Row is the row screen.LineAt indexes.
            I verified it is correct today by probe (real liveScreen as view and stdout, real
            `concrete` lookup, click at the frame cell where French renders → concrete_fr_*),
            so this is coverage, not a defect. Also no pty row sends a real mouse report.
          family: unfalsifiable-test-pin
          round: 11
        - id: BR-48
          severity: Minor
          title: TestClickMapSurvivesTheViewportGrowing's "the command menu closing" subtest is green against the pre-fix code
          detail: |-
            3rd in this family, so the deliverable is the rule: a subtest earns its row in
            the mutation table only if it reddens under the mutation that row names.
            Measured — with screen.go reverted to 8f6a458, "a resize taller" fails and
            "the command menu closing" passes, because at 21 lines into 11 rows the
            un-clamped early return is never taken.
          family: vacuous-pin
          round: 11
        - id: BR-49
          severity: Minor
          title: Four hand-rolled walks of styled text by display cell
          detail: |-
            6th in this family, so state the rule rather than patch a site: visibleCells
            (render.go:466), visibleIndex (render.go), clipVisible (screen.go:658) and
            markClickable (screen.go:288) correctly share escapeLen and cellWidth but each
            re-spells the traversal. The structural fix is one forEachCell(line, fn)
            iterator the four consume, before M2.5's splice adds a fifth reading.
          family: one-owner-per-invariant
          round: 11
        - id: BR-50
          severity: Minor
          title: The issue's Log carries no M2 entry; every M2 discovery lives only in the plan's Revisions
          detail: |-
            3rd in this family, so the rule: the issue Log is one of the sites a fact has
            to reach, not a follow-up. M1 logged eight entries; M2 logged none, so a reader
            of workshop/issues/000030-clickable-regions.md sees M1 close and nothing after,
            while Region.Word, the writeRendered seam, the 24-not-0 decision and the
            degrade-by-routing rule are recorded only in the plan.
          family: docs-lag-new-surface
          round: 11
      boundary: M2
      blocked: true
    - "n": 12
      timestamp: "2026-08-30T14:35:48-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: M1.1 still enumerates the four cases verbatim and screen_test.go still has only FuzzScreenWriteDoesNotPanic — no chunk-boundary property over screen.Write.
          round: 12
        - id: BR-40
          disposition: not-addressed
          note: 'Behaviour is corrected (measured at HEAD: one \x1b[4m, placed after the palette escape) but reverting screen.go to 9469474 leaves go test ./cmd/define/ green, so nothing defends it.'
          round: 12
        - id: BR-41
          disposition: not-addressed
          note: LineAt still reaches clamp through visible(), which writes back s.offset, while the plan's M2 table still labels the group PURE.
          round: 12
        - id: BR-44
          disposition: not-addressed
          note: key.go:186, :198 and :344 unchanged, decodeWheel still the name; three more sites of the same class arrived in this window (see the stale-rationale finding).
          round: 12
        - id: BR-45
          disposition: not-addressed
          note: addRegions still decrements base for a partial line and leaves Col relative to the render; screen_test.go:738 still supplies Col pre-offset by hand.
          round: 12
        - id: BR-46
          disposition: addressed
          note: 'Verified by mutation: word := e.Headword() reddens three tests with the recorded hot dog / a priori / bargainer messages. Corpus probe at three widths shows every region''s Word is the key, none zero-width, none overlapping.'
          round: 12
        - id: BR-47
          disposition: addressed
          note: 'Verified by two mutations on opposite sides of the joint (clickAt''s 1-based conversion; visible()''s top line) — both redden the new test, which drives a real liveScreen as view and stdout. The by-joint regeneration of the table is only partial: BR-45''s joint is still unenumerated.'
          round: 12
        - id: BR-48
          disposition: not-addressed
          note: The "the command menu closing" subtest is unchanged at screen_test.go:906.
          round: 12
        - id: BR-49
          disposition: not-addressed
          note: No forEachCell exists; markClickable's rewrite in this window re-spelled the traversal a fourth time rather than consuming a shared iterator.
          round: 12
        - id: BR-50
          disposition: addressed
          note: 'The issue now carries "2026-08-30 — M2: the clicks, and what the boundary found" with Region.Word, the writeRendered seam, the 24-not-0 decision and degrade-by-routing.'
          round: 12
      findings:
        - id: BR-51
          severity: Critical
          title: Render panics on a one-space entry — regionsIn hands findVisible an empty needle and it indexes cols[0]
          detail: |-
            Introduced by 3b60e1a. regionsIn now passes span STRINGS to findVisible, and
            two can be empty: e.Headword() when the key is not on the head line
            (render.go:334) and word itself on the --play path, which passes no Word
            (play_loop.go:262). strings.Index returns 0 for an empty needle, so
            findVisible (render.go:445) returns cols[0] on an empty cols slice.
            Measured: Render(ParseEntry(" "), RenderOpts{}) and
            Render(ParseEntry(""), RenderOpts{Word: "somekey"}) both panic; the same
            probe against 9469474:render.go passes, so it is new in this window. The
            repo's own FuzzRenderLosesNothing finds it in 0.18s and minimizes the
            crasher to a single space, while go test ./... stays green because it runs
            only the seed corpus. Reachable on the ordinary lookup path:
            selectedDictionary.Lookup returns (text, nil) without requiring non-empty
            text. Fix at the owner — findVisible refuses an empty needle, since an
            empty span is not a thing on screen — and commit the minimized crasher to
            testdata/fuzz/FuzzRenderLosesNothing/ per the target's own convention. The
            rule: a fix that gives an existing helper a new class of input inherits
            that helper's unstated preconditions, so its own commit re-runs the
            property tests covering it, not only the ones the finding named.
          family: helper-precondition-unguarded
          round: 12
        - id: BR-52
          severity: Important
          title: The atlas's clickable-regions section was not swept when the Critical fix changed what a region carries
          detail: |-
            4th in this family, so the deliverable is the rule, not the two lines.
            3b60e1a touched no atlas file: atlas/define.md:383 still prints
            "regionsIn (Entry, rendered) -> []Region" against a function that now takes
            the key, and :409 still explains Region.Word as "the entry it belongs to"
            while the round's durable decision — the target is the LOOKUP KEY carried
            on RenderOpts.Word, because a shortcut must not re-derive its target — lives
            only in the plan and the issue Log though RenderOpts gained a field. Last
            round's guard did not fire because TestAtlasDescribesEveryRegionKind derives
            from numRegionKinds and therefore defends the set of KINDS only; this
            boundary changed a different axis. The rule: a derived docs guard defends
            only the axis it derives from, and every other axis a boundary changes is
            still an owed sweep whose enumerable form is "for every Core-concepts row
            this window edited, re-read the atlas line naming the same entity". Two rows
            changed here and both have a stale atlas line.
          family: docs-lag-new-surface
          round: 12
        - id: BR-53
          severity: Minor
          title: Five comments now state facts their own commits made false, and the enumeration is the fix
          detail: |-
            3rd in this family; BR-44 named two sites and this window added three, so
            prevalence is measurable at five. key.go:186 ("answers only the WHEEL"),
            :198 ("a click therefore stays KeyUnknown") and :344 ("Only the wheel is
            answered") describe a decoder that returns KeyClick, and decodeWheel is a
            misnomer. render.go:287 says Region.Word is "the ENTRY this region belongs
            to … the word to play is still the headword", which 3b60e1a stopped being
            true. internal/llm/config.go:153 says "the parley proxy's key is four
            characters, so it is the common case here" in the very commit that made
            that key "parley-local" — twelve characters, which takes the other branch.
            The rule: a comment stating a fact about a value, or about what a function
            does NOT do, is part of that fact's blast radius; the commit that changes
            the fact sweeps every site stating it, by grep rather than by recollection.
          family: stale-rationale
          round: 12
        - id: BR-54
          severity: Minor
          title: keysOf is added in this window and called from nowhere, and frameCell's doc comment sits above livePromptOf
          detail: |-
            editorloop_test.go:1004 declares keysOf(map[int][]Region) []int; grep finds
            no caller. Left over from an earlier draft of the joint test — Go does not
            complain about an unused function, so nothing else will catch it. At :982
            the paragraph describing frameCell sits above livePromptOf, so both helpers
            are documented by the wrong comment.
          family: dead-test-scaffolding
          round: 12
      boundary: M2
      blocked: true
    - "n": 13
      timestamp: "2026-08-30T15:00:49-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan line 132 still enumerates the four cases verbatim, and screen.Write still has no chunk-independence property (only three hand-picked splits at screen_test.go:32).
          round: 13
        - id: BR-40
          disposition: addressed
          note: 'Measured: markClickable now emits "\x1b[1;36m\x1b[4mpotassium\x1b[24m…" — one underline, escapes stepped before the column trigger.'
          round: 13
        - id: BR-41
          disposition: not-addressed
          note: 'Measured: offset 999 becomes 15 after Frame() and after LineAt(); screen.go:186, :158 and the plan row all still say PURE.'
          round: 13
        - id: BR-44
          disposition: not-addressed
          note: key.go untouched since 251a85a; :186, :198 and :344 all still stale. Subsumed by BR-53.
          round: 13
        - id: BR-45
          disposition: not-addressed
          note: 'screen.go:143 still decrements base without shifting Col; screen_test.go:739 still supplies Col: 12 pre-offset by hand.'
          round: 13
        - id: BR-48
          disposition: not-addressed
          note: Re-measured against 8f6a458:screen.go — "a resize taller" fails, "the command menu closing" passes. The row still does not earn its place.
          round: 13
        - id: BR-49
          disposition: not-addressed
          note: No forEachCell exists; visibleCells, visibleIndex, clipVisible and markClickable still each spell the traversal.
          round: 13
        - id: BR-51
          disposition: addressed
          note: 'Verified by revert: removing the findVisible guard reddens TestRenderSurvivesADegenerateEntry with the original panic at render.go:445. Cell-based rather than byte-based, and the NUL crasher is committed.'
          round: 13
        - id: BR-52
          disposition: addressed
          note: atlas:383 and :401 both swept and a derived guard added — but the guard cannot fire for RenderOpts.Word; raised separately as a vacuous-pin finding rather than re-raised here.
          round: 13
        - id: BR-53
          disposition: not-addressed
          note: 'All five sites still stale (key.go:186/:198/:344, render.go:287, internal/llm/config.go:153), and a sixth: editorloop_test.go:930-950 carries two successive drafts of the same paragraph.'
          round: 13
        - id: BR-54
          disposition: addressed
          note: keysOf removed and the frameCell comment moved above frameCell.
          round: 13
      findings:
        - id: BR-55
          severity: Important
          title: TestAtlasDescribesEveryRenderOpt cannot fire for RenderOpts.Word, the field it was written for
          detail: |-
            doc_sync_test.go:273 accepts a bare "`Word`" anywhere in the atlas, which
            atlas/define.md:1916 supplies in a sentence about Question. Measured twice:
            deleting the RenderOpts.Word row leaves it green, and deleting the whole
            "## Clickable regions" section leaves it green for Word and Vocab, failing
            only on Color and Width. TestAtlasDescribesEveryRegionKind (:224) has the
            same defect for "headword", which occurs 10+ times elsewhere in the atlas.
            4th in this family, so the deliverable is the rule: a derived docs guard
            must search for a token that exists ONLY in the documentation it defends —
            a qualified anchor, never a bare name ordinary prose can supply. Dropping
            the backtick fallback keeps the suite green, since all four fields already
            carry a qualified RenderOpts.X line.
          family: vacuous-pin
          round: 13
        - id: BR-56
          severity: Important
          title: 15 fuzz targets and 12 pty rows run in nothing automated, which is why BR-51 shipped
          detail: |-
            go test ./... exercises fuzz targets against the seed corpus only, and
            there is no -fuzz invocation in Makefile, Makefile.local, Makefile.workflow,
            scripts/, or .github/workflows/merge-check.yml (scripts/merge-checks.d/
            does not exist). BR-51 was a reachable Critical panic that the repo's own
            fuzzer finds in under a second, found instead by a reviewer typing the
            flag — and round 11's answer added a 15th target with the same property.
            The 12 pty rows are the same rule from another angle: all report "no pty
            available: operation not permitted" here, so Done-when 6 and 8 were
            certified this round only by their in-process counterparts. The rule: a
            target that runs only when a human remembers to invoke it is not part of
            the suite. M1 already applied half of it by pinning handBack in process;
            the other half is a bounded `make fuzz` the close gate or CI invokes.
            Reasonably disposed as a follow-up issue rather than work inside #30 — but
            say which, rather than leaving it implicit.
          family: unrun-test-surface
          round: 13
        - id: BR-57
          severity: Minor
          title: The atlas says an empty RenderOpts.Word means "no click map wanted"; measured false
          detail: |-
            atlas/define.md:401. Render(ParseEntry(entry), RenderOpts{Width: 80}) returns
            2 regions, with Word falling back to e.Headword() at render.go:314 — which
            is precisely the shape BR-46 was filed against. Nothing enforces the stated
            guarantee; play_loop.go:262 is the only caller and it happens to discard the
            regions. Either enforce it (empty key => no regions) or state the actual
            fallback behaviour.
          family: doc-overclaim
          round: 13
      boundary: M2
      blocked: false
    - "n": 14
      timestamp: "2026-08-30T15:31:21-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan line 132 still enumerates the four cases verbatim; screen_test.go has only FuzzScreenWriteDoesNotPanic, which asserts no panic, not chunk-independence.
          round: 14
        - id: BR-26
          disposition: addressed
          note: All six enumerated sites measure cells at HEAD — editor.go:232 visibleCells, command.go:308 truncate delegates to clipVisible — and the combining-mark case is pinned at render_test.go:482.
          round: 14
        - id: BR-29
          disposition: not-addressed
          note: No tree-to-table guard exists; measured six new decls with no row — newLiveScreen, liveScreen.throttledPaint, wheelLines, digits, submitLine, headingLine.
          round: 14
        - id: BR-34
          disposition: addressed
          note: Verified by mutation — renaming TestPaintFitsTheTerminalAndParksTheCursor in the plan reddens TestPlanNamedTestsExist with the intended message.
          round: 14
        - id: BR-35
          disposition: addressed
          note: escapeLen (render.go:527) wraps scanEscape and visibleCells, clipVisible, visibleIndex and markClickable all skip through it.
          round: 14
        - id: BR-36
          disposition: addressed
          note: screen_test.go:465 asserts the cursor row exactly, derived by replaying the prompt through readFrame rather than restating Paint's formula.
          round: 14
        - id: BR-41
          disposition: not-addressed
          note: LineAt (screen.go:174) reaches clamp through visible(), which writes s.offset; screen.go:186 and the plan's M2 row both still say PURE.
          round: 14
        - id: BR-44
          disposition: not-addressed
          note: key.go:186, :198 and :344 unchanged and decodeWheel is still the name; subsumed by BR-53.
          round: 14
        - id: BR-45
          disposition: not-addressed
          note: screen.go:143 still decrements base without shifting Col, and screen_test.go:739 still supplies Col 12 pre-offset by hand.
          round: 14
        - id: BR-48
          disposition: not-addressed
          note: The "the command menu closing" subtest at screen_test.go:906 is unchanged.
          round: 14
        - id: BR-49
          disposition: not-addressed
          note: No forEachCell exists; visibleCells (render.go:510), visibleIndex (render.go:471), clipVisible (screen.go:667) and markClickable (screen.go:282) each still spell the traversal.
          round: 14
        - id: BR-53
          disposition: not-addressed
          note: All five sites unchanged; internal/llm/config.go:153 still claims a four-character key against defaultLocalKey = "parley-local", twelve characters.
          round: 14
        - id: BR-55
          disposition: not-addressed
          note: The RenderOpts half is fixed and fires; the sibling the same finding named, doc_sync_test.go:233, still searches the bare k.String() and stays green with the whole atlas section deleted.
          round: 14
        - id: BR-56
          disposition: addressed
          note: Filed as tools#37 and stated explicitly in the close commit and the issue Log; re-measured, all 12 pty rows still skip here.
          round: 14
        - id: BR-57
          disposition: not-addressed
          note: atlas/define.md:401 still says an empty RenderOpts.Word means no click map; render.go:316 still falls back to e.Headword() and returns regions.
          round: 14
      findings:
        - id: BR-58
          severity: Minor
          title: isClickButton accepts extended mouse buttons 8-11 as a left press, against its own "LEFT only" comment
          detail: |-
            key.go:256 rejects only bits 64 (wheel) and 32 (motion) before testing
            b&3 == 0, so button 8 (b=128) satisfies both. Measured: decodeKey("\x1b[<128;5;3M")
            returns KeyClick at row 2 col 4, identical to a left press, and the X10
            form {0x1b,'[','M',160,33,33} does the same. A five-button mouse's back
            button over an underlined headword therefore plays it. The rule: a
            predicate over an external wire encoding is written against the
            encoding's whole defined range, not the values the fixtures happen to
            carry — the same shape as M2.6's mode rule, one level down from the mode
            to the button field.
          family: predicate-narrower-than-its-encoding
          round: 14
        - id: BR-59
          severity: Minor
          title: markClickable abandons every later span on a line when one span's Col is unreachable or overlapping
          detail: |-
            This is the 2nd finding in family helper-precondition-unguarded (BR-51 is
            its sibling). Do NOT just guard the one call — state the rule: a helper
            consuming a region list either enforces its preconditions at the owner
            (regionsIn) or degrades PER REGION, never by abandoning the rest of the
            line. screen.go:306 advances `next` only on an exact `col == spans[next].Col`
            match, so a Col that no cell boundary can equal parks the cursor forever.
            Measured on "日本語 abc": regions at {Col 1, W 2} and {Col 7, W 3} produce
            NO marks at all, and the overlapping triple {0,6},{2,2},{7,3} marks only
            the first. Unreachable from regionsIn today — findVisible returns real
            boundaries and the corpus was probed for overlap at M2 close — which is
            exactly why it is worth stating now: the registry's whole premise is that
            "a third consumer is a row", and a third kind whose span overlaps the
            headword would silently unmark the rest of the line with a green suite.
          family: helper-precondition-unguarded
          round: 14
        - id: BR-60
          severity: Minor
          title: The issue's Done-when checklist is entirely unticked at the boundary that closes it
          detail: |-
            workshop/issues/000030-clickable-regions.md:202-214 — all seven acceptance
            rows are still "- [ ]" while the ## Plan rows above them are all ticked
            and the close commit is next. Every closed issue in workshop/history/issues/
            ticks them (checked #29 and #35). The rows are substantively delivered —
            I verified the ORIGIN click, the multi-language case on piano/ballet, the
            underline mark, and the Option/scrollback documentation in README — so
            this is a bookkeeping gap, not a delivery one. The rule: the boundary that
            closes a claim settles it in writing, either ticked or explicitly recorded
            as not delivered with the reason.
          family: unsettled-claim-at-boundary
          round: 14
      blocked: true
    - "n": 15
      timestamp: "2026-08-30T15:48:07-07:00"
      agent: claude
      dispose:
        - id: BR-55
          disposition: addressed
          note: 'Mutation-verified: deleting the atlas''s RegionHeadword/RegionOriginLang bullets reddens TestAtlasDescribesEveryRegionKind for both, where the old k.String() form stayed green with the whole section deleted.'
          round: 15
        - id: BR-58
          disposition: addressed
          note: 'Mutation-verified: reverting the b&128 guard reddens two of three subtests; accepted Cb over 0..255 is now exactly {0,4,8,12,16,20,24,28}. Residue, not re-raised — Cb 256/512/1024 still decode to KeyClick because the guard is a blacklist; b&^28 == 0 would close it in one line.'
          round: 15
        - id: BR-41
          disposition: addressed
          note: LineAt's comment and the plan's LineAt/visible row now say NOT pure with the BR-42 reason. The class it belongs to is raised separately this round.
          round: 15
        - id: BR-29
          disposition: addressed
          note: 'Deferred to tools#33 explicitly, in the plan''s Revisions, with the class named and six declarations recorded as evidence; #33 exists, predates this issue, and owns the tree-to-table direction as its whole subject. Seventh piece of evidence for it — RegionKind.identifier, added this round, is named in the Revisions prose and has no Core-concepts row.'
          round: 15
        - id: BR-1
          disposition: not-addressed
          note: Plan line 132 still enumerates the four cases verbatim; no chunk-boundary property test for Write exists.
          round: 15
        - id: BR-44
          disposition: not-addressed
          note: key.go:186, :198 and :344 unchanged, decodeWheel still the name; subsumed by BR-53.
          round: 15
        - id: BR-45
          disposition: not-addressed
          note: screen.go:143 still decrements base without shifting Col; screen_test.go still supplies Col pre-offset by hand.
          round: 15
        - id: BR-48
          disposition: not-addressed
          note: screen_test.go:906 unchanged.
          round: 15
        - id: BR-49
          disposition: not-addressed
          note: No forEachCell; render.go:488, render.go:527, screen.go:288 and screen.go:673 each still spell the traversal.
          round: 15
        - id: BR-53
          disposition: not-addressed
          note: All five sites verified unchanged, including internal/llm/config.go:153 claiming a four-character key against defaultLocalKey "parley-local".
          round: 15
        - id: BR-57
          disposition: not-addressed
          note: atlas/define.md:407 still says empty RenderOpts.Word means no click map; render.go:333 still falls back to e.Headword() and returns regions.
          round: 15
        - id: BR-59
          disposition: not-addressed
          note: screen.go:313 still advances next only on an exact col == spans[next].Col match.
          round: 15
        - id: BR-60
          disposition: not-addressed
          note: workshop/issues/000030-clickable-regions.md:202-213 all still "- [ ]" while every Plan row is ticked.
          round: 15
      findings:
        - id: BR-61
          severity: Minor
          title: The PURE relabel swept the one site BR-41 named; six siblings in the same table and the same file still claim PURE while writing the receiver
          detail: |-
            This is the 2nd finding in family pure-label-hides-mutation. Do NOT relabel
            the six sites one at a time — state the rule. Measured prevalence: seven
            sites, one fixed. screen.go:192 says "Frame is the rows to paint, oldest
            first. PURE" two lines above LineAt's new "NOT pure" comment, and both
            reach the same visible() call; measured, Frame moved s.offset from 2 to 0
            on a 5-line buffer scrolled back 2 with rows grown to 10. The plan's table
            likewise still says PURE for screen.Write (81), screen.Frame (82),
            screen.Scroll/clamp (83), screen.Page (84) and screen.Paint (85, "PURE,
            given the writer" — it writes s.cols, s.rows and s.offset), plus the prose
            at line 104. The rule: the status column carries ONE definition of PURE.
            Under ARCH-PURE's sense (no IO, unit-testable with no terminal) all seven
            are PURE and LineAt/visible should go back; under "does not mutate the
            receiver" none of them are. Relabelling one row of seven leaves a reader
            with two meanings of the same word in one table, which is worse than the
            label BR-41 objected to.
          family: pure-label-hides-mutation
          round: 15
        - id: BR-62
          severity: Minor
          title: This round's three fixes are uncommitted, so the gate's evidence is not reproducible from the commit the window names
          detail: |-
            This is the 2nd finding in family verification-claim-unreproduced (BR-23 is
            its sibling — a suite recorded green that was red at HEAD). Do NOT just
            commit these three. The rule: a gate round's evidence must be reproducible
            from the commit the round names. The window is 168b1c9..92e490d, and the
            fixes for BR-55, BR-58 and BR-41 exist only in the working tree — at
            92e490d, TestAtlasDescribesEveryRegionKind still searches k.String() and
            isClickButton still accepts Cb 128. I verified the fixes against the
            working tree by mutation and they hold, so the ledger's "addressed" is
            substantively true and procedurally unreproducible. Satisfy it by
            committing before re-running the gate, or by having the round record the
            tree it measured.
          family: verification-claim-unreproduced
          round: 15
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

## Round 7 — 2026-08-29T22:06:21-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 8 — 2026-08-29T22:28:12-07:00 (claude) — passed

### Disposed

- BR-32 — addressed — Verified by reversion at HEAD: restoring the menuRows summation in place of fitMenu reddens "more menu than terminal" (8 rows in 5) and "a terminal too short for anything" (3 in 2). readFrame asserts total height, so the rule is now enforced rather than the instance.
- BR-33 — addressed — Verified by three reversions at HEAD: len(menu) for menuRows+promptRows-1 reddens, deleting the whole "if len(menu) > 0" block reddens five subtests, and menuRows+promptRows reddens three. The frame is decoded into (row, col) as the finding asked.
- BR-29 — not-addressed — Unchanged since round 4 — repo_guard_test.go is untouched across the whole window, so the tree-to-table guard still does not exist, and screen.Lines, screen.eraseOpenLine, paintInterval and defaultRows/defaultCols still have no rows.
- BR-26 — not-addressed — Site 5 of its own enumeration survives: editor.go:227 still computes the ESC[nD park as len([]rune(sug)) + (len(e.Line) - e.Cursor), a rune count for a move the terminal makes in columns. The CJK row was added to the frame test; the combining-mark row was not.
- BR-1 — not-addressed — M1.1's prose still enumerates the four cases and no chunk-boundary property test for Write exists; FuzzScreenWriteDoesNotPanic checks only for panics. Minor, non-blocking.

### Raised

- **BR-34** [Important] `plan-table-incomplete` Done-when row 1b names TestScreenFrameFitsTheTerminalInDisplayRows, which this window's own commit renamed away
  This is the 3rd finding in family plan-table-incomplete. Earlier rounds fixed instances (BR-9 and BR-22 added missing rows, BR-29's three rows were added). Do NOT just rename the cell. The rule that covers all of them: every symbol a plan names — in ANY column of ANY of its tables, including Done-when's "pinned by" — must resolve to a declaration in the tree, and every top-level declaration the window adds to a file the tables name must have a row; one guard, both directions, all columns. Measured at HEAD: I checked all 19 test names across M1's Done-when and exactly one is missing — plan.md:150 names TestScreenFrameFitsTheTerminalInDisplayRows, renamed to TestPaintFitsTheTerminalAndParksTheCursor by aa4fe94, the commit under review. TestPlanTablesNameEntitiesThatExist exists and reads only the Core-concepts table, so the suite stayed green; the plan's own Revisions already records this class twice ("M1 done-when, two rows named tests that do not exist"), which is why the deliverable is the widened guard rather than the edit.
- **BR-35** [Important] `one-owner-per-invariant` visibleCells and clipVisible each hand-roll the CSI grammar that scanEscape already owns
  This is the 3rd finding in family one-owner-per-invariant. Earlier rounds fixed instances (BR-17 the offset clamp spelled twice, BR-27 the wheel button byte spelled twice). Do NOT just rewrite the two functions. The rule: the escape-sequence grammar has exactly one owner — sgr.go:85 scanEscape — and any site that walks a styled string skips sequences through it rather than re-deriving "ESC, then optional [, then params, then a final byte in 0x40-0x7E". Enumeration measured at HEAD, 2 of 4 non-test sites wrong: highlightwriter.go:88 defers to scanEscape (ok); render_test.go:102 defers to it and its comment says why (ok); render.go:238 visibleCells hand-rolls inEsc/inCSI, and its inCSI branch is NEW in this window; screen.go:484 clipVisible hand-rolls the same state machine plus a `styled` flag. screen_test.go:334 readFrame is a third spelling but legitimately INTERPRETS rather than skips, so it is out of scope for the sweep. The three agree today, which is what makes this cheap now and expensive later: M2.5 splices an underline through sgrState into text clipVisible cuts, so a fourth reading of the same grammar lands exactly where a divergence becomes a rendering bug (ARCH-DRY).
- **BR-36** [Minor] `unfalsifiable-test-pin` The new placement test asserts the cursor column exactly but the row only as "not the last one"
  screen_test.go:453 checks got.cursorRow >= got.rows-1. An off-by-one UPWARD in the walk-back is caught only where the buffer is empty, by readFrame's "moved the cursor above the screen" guard; I confirmed the "a menu under the prompt" fixture passes with menuRows+promptRows. Asserting the cursor is on the prompt's first row (rows - promptRows - menuRows) closes it. Separately, screen_test.go:449's wantCol %= termCols disagrees with readFrame's deferred wrap for a prompt exactly a multiple of the width — a false failure waiting for a fixture, not a false pass.

## Round 9 — 2026-08-30T11:42:28-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — M1.1's row still enumerates the four cases verbatim and no chunk-boundary property test for screen.Write exists (screen_test.go has only FuzzScreenWriteDoesNotPanic); M1 is closed, so this is now a carried Minor.

### Raised

- **BR-37** [Important] `unfalsifiable-test-pin` liveScreen.WriteRegions/RegionAtRow — the production click join is pinned by nothing (verified by mutation)
  Swapping `l.s.addRegions(rs)` and `l.s.Write([]byte(text))` in screen.go:551
  leaves `go test ./cmd/define/` green, though in production it shifts every
  region forward by the entry's line count. 6th in the family: state the rule
  — a "pinned by" claim holds only when mutating the implementing code reddens
  a named test, and a double may not stand in for the object joining two
  separately-pinned halves — then write the enumeration (every Integration-points
  row names the test that runs it) and sweep it this round.
- **BR-38** [Important] `one-owner-per-invariant` The region-registry guard restates RegionKind's extent, so Done-when 7 cannot fire
  editorloop_test.go:851 loops `kind <= RegionOriginLang`; RegionKind has no
  count sentinel, so a third kind is never exercised and `clicked`'s switch has
  no default. Second instance in the same window: originLineRange (render.go:377)
  re-derives the section boundary by an all-caps heuristic that `e.Sections`
  already owns. 4th/5th in the family: state the rule — the extent and structure
  of a declared set have one owner and every guard derives from it, as
  TestEveryEnabledMouseModeIsDecoded already does with mouseOn — and sweep both.
- **BR-39** [Important] `docs-lag-new-surface` README.md and atlas/define.md document none of M2's delivered surface
  README.md is untouched in the window; atlas/define.md changed 11 lines, all
  the console side-quest, and still refers to clicks in the future tense
  (:283, :326, :350). Missing: click-to-play, ORIGIN-language click, the
  underline mark, the scrolled-past degrade, the region registry, writeRendered's
  seam. 2nd in the family: the cause is that M1.6 made the sweep a TASK and
  M2.1-M2.6 carries no docs row — fix that, ideally as a repo_guard_test.go
  window guard, not this instance.
- **BR-40** [Minor] `column-trigger-fires-per-byte` markClickable emits the underline twice when a span begins at an escape
  Measured: markClickable("\x1b[1;36mpotassium\x1b[0m is a metal", {Col:0,Width:9})
  yields "\x1b[4m\x1b[1;36m\x1b[4mpotassium...". The column trigger is re-tested
  after each byte step. Idempotent, so cosmetic; fire it once per column.
- **BR-41** [Minor] `pure-label-hides-mutation` screen.LineAt is tabled PURE but clamps and writes back s.offset via Frame()

## Round 10 — 2026-08-30T12:15:57-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — M1.1's row still enumerates the four cases and no chunk-boundary property exists for screen.Write; M1 is closed, so this stays a carried Minor.
- BR-37 — addressed — Verified by mutation: swapping addRegions/Write reddens TestLiveScreenJoinsRegionsToTheLinesTheyWereRenderedFor; writeRendered's seam also reddens when disabled. The class enumeration it demanded is raised separately as the 7th in the family.
- BR-38 — addressed — numRegionKinds verified by mutation (a third kind reddens all three guards); originLineRange now derives from e.Sections, though reverting it leaves the suite green.
- BR-39 — addressed — README gains the clickable section, atlas gains "## Clickable regions", and TestAtlasDescribesEveryRegionKind derives from numRegionKinds - verified red for an undescribed third kind.
- BR-40 — not-addressed — Re-measured at HEAD: markClickable("\x1b[1;36mpotassium\x1b[0m is a metal", {Col:0,Width:9}) still yields "\x1b[4m\x1b[1;36m\x1b[4mpotassium...".
- BR-41 — not-addressed — The plan's M2 row still tables screen.LineAt as PURE; Frame still clamps and writes back s.offset. C-1 is the cost of that unstated write.

### Raised

- **BR-42** [Critical] `one-owner-per-invariant` screen.Frame's fast path returns without clamping, so the click map detaches from the text when the viewport grows
  5th in the family, so the rule is the deliverable: a derived invariant is
  re-established on every path that reads it, not only on the path that calls
  its owner. Measured on the real liveScreen — 20 lines at 12 rows, Page(10)
  then Resize(30,80): offset stays 9, topLine = -9, and RegionAtRow(0,0) on
  the row that actually shows the headword answers nothing while its underline
  is painted 9 rows lower. Three routine triggers: resize taller while scrolled
  back, the suggestion menu closing, a wrapped prompt killed with Ctrl-U.
  Hoisting s.clamp() above the early return fixes the repro and keeps the suite
  green — which is also the second half of the finding, since nothing pins it.
  Sweep: every early return in screen.go that precedes clamp, and every reader
  of s.offset/s.rows outside Paint. ARCH-PURE: Frame/LineAt are tabled PURE
  while writing back s.offset, which is what hides the missing clamp.
- **BR-43** [Important] `unfalsifiable-test-pin` BR-37's enumeration was never written, and two fixes in the same commit ship with nothing that fails without them
  7th in the family, across 7 rounds on this issue. The rule was stated and the
  named instance pinned, but: Done-when row 1 still names only the four tests
  that stayed green under last round's mutation, and never names
  TestLiveScreenJoinsRegionsToTheLinesTheyWereRenderedFor; reverting
  originLineRange to the exact all-caps heuristic BR-38 named leaves
  `go test ./cmd/define/` green (measured, 107s); and C-1's clamp is green with
  and without its fix. One row IS filled and I checked it: disabling
  writeRendered's regionWriter branch reddens TestALookupHandsItsRegionsToTheScreen.
  Deliverable: for every M2 Core-concepts row and every Done-when "pinned by"
  cell, name the test and record the mutation that reddens it.
- **BR-44** [Minor] `stale-rationale` decodeWheel's and decodeX10Mouse's comments still say a click stays KeyUnknown, which stopped being true in this window
  key.go:186 and :196 say the decoder "answers only the WHEEL" and that "a click
  therefore stays KeyUnknown — consumed whole and inert"; key.go:344 repeats it
  for the X10 path. Both functions now return KeyClick, and the name decodeWheel
  is a misnomer for a decoder that also decodes presses. 2nd in the family: the
  rule is that a comment stating what a function does NOT do is swept in the
  commit that makes it do it — sweep both sites and the name.
- **BR-45** [Minor] `vacuous-pin` addRegions shifts base for a partial line but not Col, and the test that looks like it covers this passes Col pre-offset
  screen.go:143 decrements base when s.partial, so a render starting mid-line
  lands on the right LINE — but Col is left relative to the render, not to the
  buffer line, so the regions sit in the wrong columns. screen_test.go:738 seems
  to cover the branch and instead supplies Col: 12 already offset by hand, so it
  asserts the caller's arithmetic. Not reachable today (the loop writes "\r\n"
  before an entry), but the comment documents one contract and the test another.
  Either enforce the precondition or shift Col by visibleCells of the open line.

## Round 11 — 2026-08-30T12:49:38-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Still open: no chunk-boundary property over screen.Write exists — TestScreenWriteBuildsLines is still the four enumerated cases.
- BR-40 — not-addressed — Reproduced at HEAD: markClickable("\x1b[1;36mpotassium\x1b[0m is a metal", {Col:0,Width:9}) = "\x1b[4m\x1b[1;36m\x1b[4mpotassium\x1b[24m\x1b[0m is a metal".
- BR-41 — not-addressed — LineAt now reaches clamp through visible() rather than Frame(), so the write-back under a PURE table row remains.
- BR-42 — addressed — Verified by revert: screen.go at 8f6a458 reddens TestClickMapSurvivesTheViewportGrowing with the exact recorded message.
- BR-43 — addressed — Enumeration exists and holds: seven sampled rows each reddened under the mutation the table names.
- BR-44 — not-addressed — key.go:186/196 still say the decoder answers only the wheel and a click stays KeyUnknown; key.go:344 repeats it; decodeWheel is still the name.
- BR-45 — not-addressed — Reproduced: a mid-line render with Col relative to the render resolves at buffer column 0, not at the open line's width.

### Raised

- **BR-46** [Critical] `shortcut-rederives-its-target` A multi-word entry's clickable headword is only its first field, so the click plays a different word
  regionsIn (render.go:307) builds the headword region from individual HeadWord
  tokens and stamps Word: e.Headword(), which parseHead (parse.go:436) fills
  from fields[0] alone. Measured on the committed `hot dog` fixture: a bare
  Enter asks the CDN for hot_dog_en_us_1.mp3, while a click on the same
  entry's underlined headword asks for hot_en_us_1.mp3, hot_en_us_2.mp3 and
  the hot-- fallbacks. `a priori` produces a single region for the letter "a".
  RegionOriginLang carries the same Word, so a French replay on such an entry
  is also the wrong word. The mark is wrong with it: only "hot" is underlined
  in a head line reading "hot dog". Derive the target from the same source the
  gesture it shortcuts uses — lookupAndRender holds the lookup key — and widen
  the span to the whole head phrase. The missing property: for every headword
  region, the first audio candidate for r.Word equals the first candidate a
  bare-Enter replay of that entry produces.
- **BR-47** [Important] `unfalsifiable-test-pin` The terminal-row to viewport-row joint is pinned by nothing, and runEditor never runs against a real liveScreen
  8th in this family. Do NOT fix only this instance — the rule the family keeps
  producing is that the enumeration must be of JOINTS, not entities: every place
  two separately-pinned layers exchange a value across a coordinate or unit
  boundary gets a row, and a row earns it only when a test drives both real
  objects. Concretely: newLiveScreen appears in no editor-loop test, both click
  action tests script recordDisplay.RegionAtRow's answer with row/col of the
  test's own choosing, and TestLiveScreenJoins… calls RegionAtRow directly with
  hand-written regions. Nothing asserts Key.Row is the row screen.LineAt indexes.
  I verified it is correct today by probe (real liveScreen as view and stdout, real
  `concrete` lookup, click at the frame cell where French renders → concrete_fr_*),
  so this is coverage, not a defect. Also no pty row sends a real mouse report.
- **BR-48** [Minor] `vacuous-pin` TestClickMapSurvivesTheViewportGrowing's "the command menu closing" subtest is green against the pre-fix code
  3rd in this family, so the deliverable is the rule: a subtest earns its row in
  the mutation table only if it reddens under the mutation that row names.
  Measured — with screen.go reverted to 8f6a458, "a resize taller" fails and
  "the command menu closing" passes, because at 21 lines into 11 rows the
  un-clamped early return is never taken.
- **BR-49** [Minor] `one-owner-per-invariant` Four hand-rolled walks of styled text by display cell
  6th in this family, so state the rule rather than patch a site: visibleCells
  (render.go:466), visibleIndex (render.go), clipVisible (screen.go:658) and
  markClickable (screen.go:288) correctly share escapeLen and cellWidth but each
  re-spells the traversal. The structural fix is one forEachCell(line, fn)
  iterator the four consume, before M2.5's splice adds a fifth reading.
- **BR-50** [Minor] `docs-lag-new-surface` The issue's Log carries no M2 entry; every M2 discovery lives only in the plan's Revisions
  3rd in this family, so the rule: the issue Log is one of the sites a fact has
  to reach, not a follow-up. M1 logged eight entries; M2 logged none, so a reader
  of workshop/issues/000030-clickable-regions.md sees M1 close and nothing after,
  while Region.Word, the writeRendered seam, the 24-not-0 decision and the
  degrade-by-routing rule are recorded only in the plan.

## Round 12 — 2026-08-30T14:35:48-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — M1.1 still enumerates the four cases verbatim and screen_test.go still has only FuzzScreenWriteDoesNotPanic — no chunk-boundary property over screen.Write.
- BR-40 — not-addressed — Behaviour is corrected (measured at HEAD: one \x1b[4m, placed after the palette escape) but reverting screen.go to 9469474 leaves go test ./cmd/define/ green, so nothing defends it.
- BR-41 — not-addressed — LineAt still reaches clamp through visible(), which writes back s.offset, while the plan's M2 table still labels the group PURE.
- BR-44 — not-addressed — key.go:186, :198 and :344 unchanged, decodeWheel still the name; three more sites of the same class arrived in this window (see the stale-rationale finding).
- BR-45 — not-addressed — addRegions still decrements base for a partial line and leaves Col relative to the render; screen_test.go:738 still supplies Col pre-offset by hand.
- BR-46 — addressed — Verified by mutation: word := e.Headword() reddens three tests with the recorded hot dog / a priori / bargainer messages. Corpus probe at three widths shows every region's Word is the key, none zero-width, none overlapping.
- BR-47 — addressed — Verified by two mutations on opposite sides of the joint (clickAt's 1-based conversion; visible()'s top line) — both redden the new test, which drives a real liveScreen as view and stdout. The by-joint regeneration of the table is only partial: BR-45's joint is still unenumerated.
- BR-48 — not-addressed — The "the command menu closing" subtest is unchanged at screen_test.go:906.
- BR-49 — not-addressed — No forEachCell exists; markClickable's rewrite in this window re-spelled the traversal a fourth time rather than consuming a shared iterator.
- BR-50 — addressed — The issue now carries "2026-08-30 — M2: the clicks, and what the boundary found" with Region.Word, the writeRendered seam, the 24-not-0 decision and degrade-by-routing.

### Raised

- **BR-51** [Critical] `helper-precondition-unguarded` Render panics on a one-space entry — regionsIn hands findVisible an empty needle and it indexes cols[0]
  Introduced by 3b60e1a. regionsIn now passes span STRINGS to findVisible, and
  two can be empty: e.Headword() when the key is not on the head line
  (render.go:334) and word itself on the --play path, which passes no Word
  (play_loop.go:262). strings.Index returns 0 for an empty needle, so
  findVisible (render.go:445) returns cols[0] on an empty cols slice.
  Measured: Render(ParseEntry(" "), RenderOpts{}) and
  Render(ParseEntry(""), RenderOpts{Word: "somekey"}) both panic; the same
  probe against 9469474:render.go passes, so it is new in this window. The
  repo's own FuzzRenderLosesNothing finds it in 0.18s and minimizes the
  crasher to a single space, while go test ./... stays green because it runs
  only the seed corpus. Reachable on the ordinary lookup path:
  selectedDictionary.Lookup returns (text, nil) without requiring non-empty
  text. Fix at the owner — findVisible refuses an empty needle, since an
  empty span is not a thing on screen — and commit the minimized crasher to
  testdata/fuzz/FuzzRenderLosesNothing/ per the target's own convention. The
  rule: a fix that gives an existing helper a new class of input inherits
  that helper's unstated preconditions, so its own commit re-runs the
  property tests covering it, not only the ones the finding named.
- **BR-52** [Important] `docs-lag-new-surface` The atlas's clickable-regions section was not swept when the Critical fix changed what a region carries
  4th in this family, so the deliverable is the rule, not the two lines.
  3b60e1a touched no atlas file: atlas/define.md:383 still prints
  "regionsIn (Entry, rendered) -> []Region" against a function that now takes
  the key, and :409 still explains Region.Word as "the entry it belongs to"
  while the round's durable decision — the target is the LOOKUP KEY carried
  on RenderOpts.Word, because a shortcut must not re-derive its target — lives
  only in the plan and the issue Log though RenderOpts gained a field. Last
  round's guard did not fire because TestAtlasDescribesEveryRegionKind derives
  from numRegionKinds and therefore defends the set of KINDS only; this
  boundary changed a different axis. The rule: a derived docs guard defends
  only the axis it derives from, and every other axis a boundary changes is
  still an owed sweep whose enumerable form is "for every Core-concepts row
  this window edited, re-read the atlas line naming the same entity". Two rows
  changed here and both have a stale atlas line.
- **BR-53** [Minor] `stale-rationale` Five comments now state facts their own commits made false, and the enumeration is the fix
  3rd in this family; BR-44 named two sites and this window added three, so
  prevalence is measurable at five. key.go:186 ("answers only the WHEEL"),
  :198 ("a click therefore stays KeyUnknown") and :344 ("Only the wheel is
  answered") describe a decoder that returns KeyClick, and decodeWheel is a
  misnomer. render.go:287 says Region.Word is "the ENTRY this region belongs
  to … the word to play is still the headword", which 3b60e1a stopped being
  true. internal/llm/config.go:153 says "the parley proxy's key is four
  characters, so it is the common case here" in the very commit that made
  that key "parley-local" — twelve characters, which takes the other branch.
  The rule: a comment stating a fact about a value, or about what a function
  does NOT do, is part of that fact's blast radius; the commit that changes
  the fact sweeps every site stating it, by grep rather than by recollection.
- **BR-54** [Minor] `dead-test-scaffolding` keysOf is added in this window and called from nowhere, and frameCell's doc comment sits above livePromptOf
  editorloop_test.go:1004 declares keysOf(map[int][]Region) []int; grep finds
  no caller. Left over from an earlier draft of the joint test — Go does not
  complain about an unused function, so nothing else will catch it. At :982
  the paragraph describing frameCell sits above livePromptOf, so both helpers
  are documented by the wrong comment.

## Round 13 — 2026-08-30T15:00:49-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Plan line 132 still enumerates the four cases verbatim, and screen.Write still has no chunk-independence property (only three hand-picked splits at screen_test.go:32).
- BR-40 — addressed — Measured: markClickable now emits "\x1b[1;36m\x1b[4mpotassium\x1b[24m…" — one underline, escapes stepped before the column trigger.
- BR-41 — not-addressed — Measured: offset 999 becomes 15 after Frame() and after LineAt(); screen.go:186, :158 and the plan row all still say PURE.
- BR-44 — not-addressed — key.go untouched since 251a85a; :186, :198 and :344 all still stale. Subsumed by BR-53.
- BR-45 — not-addressed — screen.go:143 still decrements base without shifting Col; screen_test.go:739 still supplies Col: 12 pre-offset by hand.
- BR-48 — not-addressed — Re-measured against 8f6a458:screen.go — "a resize taller" fails, "the command menu closing" passes. The row still does not earn its place.
- BR-49 — not-addressed — No forEachCell exists; visibleCells, visibleIndex, clipVisible and markClickable still each spell the traversal.
- BR-51 — addressed — Verified by revert: removing the findVisible guard reddens TestRenderSurvivesADegenerateEntry with the original panic at render.go:445. Cell-based rather than byte-based, and the NUL crasher is committed.
- BR-52 — addressed — atlas:383 and :401 both swept and a derived guard added — but the guard cannot fire for RenderOpts.Word; raised separately as a vacuous-pin finding rather than re-raised here.
- BR-53 — not-addressed — All five sites still stale (key.go:186/:198/:344, render.go:287, internal/llm/config.go:153), and a sixth: editorloop_test.go:930-950 carries two successive drafts of the same paragraph.
- BR-54 — addressed — keysOf removed and the frameCell comment moved above frameCell.

### Raised

- **BR-55** [Important] `vacuous-pin` TestAtlasDescribesEveryRenderOpt cannot fire for RenderOpts.Word, the field it was written for
  doc_sync_test.go:273 accepts a bare "`Word`" anywhere in the atlas, which
  atlas/define.md:1916 supplies in a sentence about Question. Measured twice:
  deleting the RenderOpts.Word row leaves it green, and deleting the whole
  "## Clickable regions" section leaves it green for Word and Vocab, failing
  only on Color and Width. TestAtlasDescribesEveryRegionKind (:224) has the
  same defect for "headword", which occurs 10+ times elsewhere in the atlas.
  4th in this family, so the deliverable is the rule: a derived docs guard
  must search for a token that exists ONLY in the documentation it defends —
  a qualified anchor, never a bare name ordinary prose can supply. Dropping
  the backtick fallback keeps the suite green, since all four fields already
  carry a qualified RenderOpts.X line.
- **BR-56** [Important] `unrun-test-surface` 15 fuzz targets and 12 pty rows run in nothing automated, which is why BR-51 shipped
  go test ./... exercises fuzz targets against the seed corpus only, and
  there is no -fuzz invocation in Makefile, Makefile.local, Makefile.workflow,
  scripts/, or .github/workflows/merge-check.yml (scripts/merge-checks.d/
  does not exist). BR-51 was a reachable Critical panic that the repo's own
  fuzzer finds in under a second, found instead by a reviewer typing the
  flag — and round 11's answer added a 15th target with the same property.
  The 12 pty rows are the same rule from another angle: all report "no pty
  available: operation not permitted" here, so Done-when 6 and 8 were
  certified this round only by their in-process counterparts. The rule: a
  target that runs only when a human remembers to invoke it is not part of
  the suite. M1 already applied half of it by pinning handBack in process;
  the other half is a bounded `make fuzz` the close gate or CI invokes.
  Reasonably disposed as a follow-up issue rather than work inside #30 — but
  say which, rather than leaving it implicit.
- **BR-57** [Minor] `doc-overclaim` The atlas says an empty RenderOpts.Word means "no click map wanted"; measured false
  atlas/define.md:401. Render(ParseEntry(entry), RenderOpts{Width: 80}) returns
  2 regions, with Word falling back to e.Headword() at render.go:314 — which
  is precisely the shape BR-46 was filed against. Nothing enforces the stated
  guarantee; play_loop.go:262 is the only caller and it happens to discard the
  regions. Either enforce it (empty key => no regions) or state the actual
  fallback behaviour.

## Round 14 — 2026-08-30T15:31:21-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Plan line 132 still enumerates the four cases verbatim; screen_test.go has only FuzzScreenWriteDoesNotPanic, which asserts no panic, not chunk-independence.
- BR-26 — addressed — All six enumerated sites measure cells at HEAD — editor.go:232 visibleCells, command.go:308 truncate delegates to clipVisible — and the combining-mark case is pinned at render_test.go:482.
- BR-29 — not-addressed — No tree-to-table guard exists; measured six new decls with no row — newLiveScreen, liveScreen.throttledPaint, wheelLines, digits, submitLine, headingLine.
- BR-34 — addressed — Verified by mutation — renaming TestPaintFitsTheTerminalAndParksTheCursor in the plan reddens TestPlanNamedTestsExist with the intended message.
- BR-35 — addressed — escapeLen (render.go:527) wraps scanEscape and visibleCells, clipVisible, visibleIndex and markClickable all skip through it.
- BR-36 — addressed — screen_test.go:465 asserts the cursor row exactly, derived by replaying the prompt through readFrame rather than restating Paint's formula.
- BR-41 — not-addressed — LineAt (screen.go:174) reaches clamp through visible(), which writes s.offset; screen.go:186 and the plan's M2 row both still say PURE.
- BR-44 — not-addressed — key.go:186, :198 and :344 unchanged and decodeWheel is still the name; subsumed by BR-53.
- BR-45 — not-addressed — screen.go:143 still decrements base without shifting Col, and screen_test.go:739 still supplies Col 12 pre-offset by hand.
- BR-48 — not-addressed — The "the command menu closing" subtest at screen_test.go:906 is unchanged.
- BR-49 — not-addressed — No forEachCell exists; visibleCells (render.go:510), visibleIndex (render.go:471), clipVisible (screen.go:667) and markClickable (screen.go:282) each still spell the traversal.
- BR-53 — not-addressed — All five sites unchanged; internal/llm/config.go:153 still claims a four-character key against defaultLocalKey = "parley-local", twelve characters.
- BR-55 — not-addressed — The RenderOpts half is fixed and fires; the sibling the same finding named, doc_sync_test.go:233, still searches the bare k.String() and stays green with the whole atlas section deleted.
- BR-56 — addressed — Filed as tools#37 and stated explicitly in the close commit and the issue Log; re-measured, all 12 pty rows still skip here.
- BR-57 — not-addressed — atlas/define.md:401 still says an empty RenderOpts.Word means no click map; render.go:316 still falls back to e.Headword() and returns regions.

### Raised

- **BR-58** [Minor] `predicate-narrower-than-its-encoding` isClickButton accepts extended mouse buttons 8-11 as a left press, against its own "LEFT only" comment
  key.go:256 rejects only bits 64 (wheel) and 32 (motion) before testing
  b&3 == 0, so button 8 (b=128) satisfies both. Measured: decodeKey("\x1b[<128;5;3M")
  returns KeyClick at row 2 col 4, identical to a left press, and the X10
  form {0x1b,'[','M',160,33,33} does the same. A five-button mouse's back
  button over an underlined headword therefore plays it. The rule: a
  predicate over an external wire encoding is written against the
  encoding's whole defined range, not the values the fixtures happen to
  carry — the same shape as M2.6's mode rule, one level down from the mode
  to the button field.
- **BR-59** [Minor] `helper-precondition-unguarded` markClickable abandons every later span on a line when one span's Col is unreachable or overlapping
  This is the 2nd finding in family helper-precondition-unguarded (BR-51 is
  its sibling). Do NOT just guard the one call — state the rule: a helper
  consuming a region list either enforces its preconditions at the owner
  (regionsIn) or degrades PER REGION, never by abandoning the rest of the
  line. screen.go:306 advances `next` only on an exact `col == spans[next].Col`
  match, so a Col that no cell boundary can equal parks the cursor forever.
  Measured on "日本語 abc": regions at {Col 1, W 2} and {Col 7, W 3} produce
  NO marks at all, and the overlapping triple {0,6},{2,2},{7,3} marks only
  the first. Unreachable from regionsIn today — findVisible returns real
  boundaries and the corpus was probed for overlap at M2 close — which is
  exactly why it is worth stating now: the registry's whole premise is that
  "a third consumer is a row", and a third kind whose span overlaps the
  headword would silently unmark the rest of the line with a green suite.
- **BR-60** [Minor] `unsettled-claim-at-boundary` The issue's Done-when checklist is entirely unticked at the boundary that closes it
  workshop/issues/000030-clickable-regions.md:202-214 — all seven acceptance
  rows are still "- [ ]" while the ## Plan rows above them are all ticked
  and the close commit is next. Every closed issue in workshop/history/issues/
  ticks them (checked #29 and #35). The rows are substantively delivered —
  I verified the ORIGIN click, the multi-language case on piano/ballet, the
  underline mark, and the Option/scrollback documentation in README — so
  this is a bookkeeping gap, not a delivery one. The rule: the boundary that
  closes a claim settles it in writing, either ticked or explicitly recorded
  as not delivered with the reason.

## Round 15 — 2026-08-30T15:48:07-07:00 (claude) — passed

### Disposed

- BR-55 — addressed — Mutation-verified: deleting the atlas's RegionHeadword/RegionOriginLang bullets reddens TestAtlasDescribesEveryRegionKind for both, where the old k.String() form stayed green with the whole section deleted.
- BR-58 — addressed — Mutation-verified: reverting the b&128 guard reddens two of three subtests; accepted Cb over 0..255 is now exactly {0,4,8,12,16,20,24,28}. Residue, not re-raised — Cb 256/512/1024 still decode to KeyClick because the guard is a blacklist; b&^28 == 0 would close it in one line.
- BR-41 — addressed — LineAt's comment and the plan's LineAt/visible row now say NOT pure with the BR-42 reason. The class it belongs to is raised separately this round.
- BR-29 — addressed — Deferred to tools#33 explicitly, in the plan's Revisions, with the class named and six declarations recorded as evidence; #33 exists, predates this issue, and owns the tree-to-table direction as its whole subject. Seventh piece of evidence for it — RegionKind.identifier, added this round, is named in the Revisions prose and has no Core-concepts row.
- BR-1 — not-addressed — Plan line 132 still enumerates the four cases verbatim; no chunk-boundary property test for Write exists.
- BR-44 — not-addressed — key.go:186, :198 and :344 unchanged, decodeWheel still the name; subsumed by BR-53.
- BR-45 — not-addressed — screen.go:143 still decrements base without shifting Col; screen_test.go still supplies Col pre-offset by hand.
- BR-48 — not-addressed — screen_test.go:906 unchanged.
- BR-49 — not-addressed — No forEachCell; render.go:488, render.go:527, screen.go:288 and screen.go:673 each still spell the traversal.
- BR-53 — not-addressed — All five sites verified unchanged, including internal/llm/config.go:153 claiming a four-character key against defaultLocalKey "parley-local".
- BR-57 — not-addressed — atlas/define.md:407 still says empty RenderOpts.Word means no click map; render.go:333 still falls back to e.Headword() and returns regions.
- BR-59 — not-addressed — screen.go:313 still advances next only on an exact col == spans[next].Col match.
- BR-60 — not-addressed — workshop/issues/000030-clickable-regions.md:202-213 all still "- [ ]" while every Plan row is ticked.

### Raised

- **BR-61** [Minor] `pure-label-hides-mutation` The PURE relabel swept the one site BR-41 named; six siblings in the same table and the same file still claim PURE while writing the receiver
  This is the 2nd finding in family pure-label-hides-mutation. Do NOT relabel
  the six sites one at a time — state the rule. Measured prevalence: seven
  sites, one fixed. screen.go:192 says "Frame is the rows to paint, oldest
  first. PURE" two lines above LineAt's new "NOT pure" comment, and both
  reach the same visible() call; measured, Frame moved s.offset from 2 to 0
  on a 5-line buffer scrolled back 2 with rows grown to 10. The plan's table
  likewise still says PURE for screen.Write (81), screen.Frame (82),
  screen.Scroll/clamp (83), screen.Page (84) and screen.Paint (85, "PURE,
  given the writer" — it writes s.cols, s.rows and s.offset), plus the prose
  at line 104. The rule: the status column carries ONE definition of PURE.
  Under ARCH-PURE's sense (no IO, unit-testable with no terminal) all seven
  are PURE and LineAt/visible should go back; under "does not mutate the
  receiver" none of them are. Relabelling one row of seven leaves a reader
  with two meanings of the same word in one table, which is worse than the
  label BR-41 objected to.
- **BR-62** [Minor] `verification-claim-unreproduced` This round's three fixes are uncommitted, so the gate's evidence is not reproducible from the commit the window names
  This is the 2nd finding in family verification-claim-unreproduced (BR-23 is
  its sibling — a suite recorded green that was red at HEAD). Do NOT just
  commit these three. The rule: a gate round's evidence must be reproducible
  from the commit the round names. The window is 168b1c9..92e490d, and the
  fixes for BR-55, BR-58 and BR-41 exist only in the working tree — at
  92e490d, TestAtlasDescribesEveryRegionKind still searches k.String() and
  isClickButton still accepts Cb 128. I verified the fixes against the
  working tree by mutation and they hold, so the ledger's "addressed" is
  substantively true and procedurally unreproducible. Satisfy it by
  committing before re-running the gate, or by having the round record the
  tree it measured.

## Open findings

- **BR-1** [Minor] `test-cases-enumerated-in-prose` M1.1 enumerates four table-test cases in prose; compress to one strategy line per risky function
- **BR-44** [Minor] `stale-rationale` decodeWheel's and decodeX10Mouse's comments still say a click stays KeyUnknown, which stopped being true in this window
- **BR-45** [Minor] `vacuous-pin` addRegions shifts base for a partial line but not Col, and the test that looks like it covers this passes Col pre-offset
- **BR-48** [Minor] `vacuous-pin` TestClickMapSurvivesTheViewportGrowing's "the command menu closing" subtest is green against the pre-fix code
- **BR-49** [Minor] `one-owner-per-invariant` Four hand-rolled walks of styled text by display cell
- **BR-53** [Minor] `stale-rationale` Five comments now state facts their own commits made false, and the enumeration is the fix
- **BR-57** [Minor] `doc-overclaim` The atlas says an empty RenderOpts.Word means "no click map wanted"; measured false
- **BR-59** [Minor] `helper-precondition-unguarded` markClickable abandons every later span on a line when one span's Col is unreachable or overlapping
- **BR-60** [Minor] `unsettled-claim-at-boundary` The issue's Done-when checklist is entirely unticked at the boundary that closes it
- **BR-61** [Minor] `pure-label-hides-mutation` The PURE relabel swept the one site BR-41 named; six siblings in the same table and the same file still claim PURE while writing the receiver
- **BR-62** [Minor] `verification-claim-unreproduced` This round's three fixes are uncommitted, so the gate's evidence is not reproducible from the commit the window names
