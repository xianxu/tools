---
gate: boundary-review
issue: 24
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-27T16:42:01-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: The flow reversal reached README and atlas but not the form's own doc comments in cmd/define/play
          detail: |-
            recall.go:3 still says "show the word, let the learner try to remember it,
            reveal the definition, self-rate" — the exact order #24 inverted. Same class:
            question.go:60 ("Reveal is what they see after asking to see it"),
            question.go:68 ("Enter and space reveal"), and play_loop_test.go:595,600
            ("play_loop.go:182 — space reveals", now :194 after this diff). All four are
            returned by the plan's own Task 9 grep pattern over cmd/define/, and the
            issue's Done-when claims the site list was built from that grep. Sweep the
            enumeration in one pass, not just the site named here.
          family: doc-sweep-incomplete
          round: 1
        - id: BR-2
          severity: Important
          title: The durable plan has all 42 step checkboxes unticked at close
          detail: |-
            6c29da2 ticked the issue's Plan rows only; workshop/plans/000024-play-grade-first-plan.md
            still reads as unstarted. Every recently closed plan in workshop/history/plans/
            is fully ticked (000021 52/0, 000016 60/0, 000015 39/0), so this breaks the
            standing convention and the plan artifact no longer records what was delivered.
          family: plan-artifact-not-ticked
          round: 1
        - id: BR-3
          severity: Minor
          title: README still implies pronunciation audio is fetched for every word in a sitting
          detail: |-
            README.md:65-66 reads "Pronunciation audio IS fetched over the network as each
            word is revealed". Under grade-first a `y` reveals nothing and so fetches and
            plays nothing — the issue's headline speedup, unstated in the README.
          family: doc-sweep-incomplete
          round: 1
        - id: BR-4
          severity: Minor
          title: The OutcomeReveal arm would nil-panic, not just mis-name the word, if an input ever advanced and revealed
          detail: |-
            play_loop.go:143 calls s.Current().Word() after Apply returned. Correct today
            (verified: OutcomeReveal is emitted only from the two non-advancing paths), but
            Current() returns a nil interface once the session is Done, so the failure mode
            is a panic rather than the wrong word the comment describes. Carrying Word on
            OutcomeReveal, as OutcomeRecord and OutcomeDrop already do, removes it.
          family: outcome-carries-its-own-subject
          round: 1
        - id: BR-5
          severity: Minor
          title: Four loop tests repeat the same five-line audio rig instead of sharing a helper
          detail: |-
            TestCorrectAnswerPlaysNoAudio, TestAMissPlaysThePronunciation,
            TestRevealPlaysThePronunciationByDefault and TestNoAudioSilencesTheSession each
            re-set opt.noAudio/times, a fakePlayer and okAudio{}. An audioRig(t, word)
            helper would make the PQ-6 trap (a rig whose player is unreachable) impossible
            to reintroduce rather than merely fixed at four sites (ARCH-DRY).
          family: duplicated-test-rig
          round: 1
        - id: BR-6
          severity: Minor
          title: The pty conformance suite SKIPs when no pty is available, so a sandboxed run reads as passing
          detail: |-
            startDefineInDir calls t.Skipf on "no pty available: operation not permitted",
            which is what every in-sandbox run gets — `ok` plus 7 SKIPs. Verified: the suite
            is 7/7 PASS outside the sandbox. Given #6 found this suite red across two closes,
            an opt-in env gate (fail instead of skip when it is set) would make green mean
            "it ran" (ARCH-MOCK: a conformance check that cannot fail is not a check).
          family: check-that-cannot-fail-reads-as-green
          round: 1
        - id: BR-7
          severity: Minor
          title: Done-when says the no-audio assertion uses a player double that fails if called; it records and is checked after
          detail: |-
            The delivered mechanism (fakePlayer.Played asserted empty, with okAudio installed
            so the branch is reachable) is equivalent and mutation-verified, and the plan
            chose it deliberately per #6 BR-43. The issue text should say so.
          family: donewhen-describes-a-mechanism-not-delivered
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-27T17:01:30-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: recall.go and question.go fixed; play_loop_test.go:595,600 still cite play_loop.go:182 (now :195) and README:54 (now :58) — the fix's own sweep excludes _test, where the two remaining enumerated sites live.
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'Verified: 42 of 42 step checkboxes ticked, 0 unticked (the one remaining "- [ ]" is the boilerplate that names the syntax).'
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: README.md:65-66 unchanged; the audio sentence still does not say a `y` fetches and plays nothing.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: play_loop.go:143 unchanged; the comment predicting "the next word" still describes a nil-interface panic, and OutcomeReveal still carries no Word.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: No audioRig helper; the rig is now at five sites, not four (play_loop_test.go:343, 360, 519, 648, 669).
          round: 2
        - id: BR-6
          disposition: addressed
          note: 'Verified reachable and failing: DEFINE_CONFORMANCE_STRICT=1 reddens all 7 pty tests here, where the unset run reports ok. See the new class finding.'
          round: 2
        - id: BR-7
          disposition: addressed
          note: 'The Done-when now names fakePlayer.Played + okAudio and cites #6 BR-43; I confirmed the mechanism is mutation-verified.'
          round: 2
      findings:
        - id: BR-8
          severity: Important
          title: The loop must perform BOTH outcomes of a miss; only the second is pinned, so dropping every miss from the event log is invisible to the whole suite
          detail: |-
            Apply returning []Outcome exists so `n` on a hidden word both records and reveals.
            Mutating play_loop.go:120 to `for _, out := range outs[len(outs)-1:]` drops the
            OutcomeRecord — no miss ever reaches events/ or the schedule — and
            `go test ./cmd/define/ -run 'Play|Session|Miss|Correct|Drop|Reveal|Capturer|Prompt|Record|Interrupt'`
            stays ok. The pty suite cannot catch it either: it asserts "0 right, 1 wrong",
            which finish() reads from the session tally that score() sets inside Apply, and
            it makes no store assertion at all. The mirror mutation `outs[:1]` IS caught by
            TestAMissPlaysThePronunciation. Fix: that test already calls playRig and throws
            away st — keep it and assert reviewEvents(t, st) is one event with Correct false,
            so one test pins both halves of the slice.
          family: widened-contract-unpinned-at-the-consumer
          round: 2
        - id: BR-9
          severity: Minor
          title: DEFINE_CONFORMANCE_STRICT is honoured at one of seven conformance skip sites, so the strict run still reports green for four suites that did not execute
          detail: |-
            This is the 2nd finding in family `check-that-cannot-fail-reads-as-green`. The
            earlier round fixed the pty instance and lessons.md:1880 wrote the general rule
            ("any test that can skip itself needs a mode where the skip is an error"), but
            the env var is read only at pty_conformance_test.go:94. Measured prevalence:
            fetch_conformance_test.go:27,62; player_conformance_test.go:23,33;
            reflect_conformance_test.go:47; live_property_test.go:38; pty_conformance_test.go:271
            all still skip unconditionally. Do not patch another site — route every
            conformance skip through one skipOrFail(t, reason, err) helper that consults the
            env var, and enumerate with `grep -rn 't\.Skipf\?(' cmd/define/*_test.go` so the
            next suite inherits the rule.
          family: check-that-cannot-fail-reads-as-green
          round: 2
        - id: BR-10
          severity: Minor
          title: The new DEFINE_CONFORMANCE_STRICT surface reached no doc, and the atlas conformance table still says three seams while listing neither pty suite
          detail: |-
            This is the 3rd finding in family `doc-sweep-incomplete`. atlas/define.md:1002
            reads "All three seams have one" over a table omitting the pty suite and this
            issue's TestPTYPlayGradeFirst; README.md:291-294 still shows the bare
            `go test -tags conformance ./...` with no mention of the strict mode. The rule
            that covers all three findings in this family is the ARCH-PURPOSE one: a
            hand-maintained restatement of a fact the code owns will drift, so make it
            derive. The cheapest enforcement available is a test asserting README.md contains
            the literal prompt string draw emits (play_loop.go:281 vs README.md:51) — that
            converts the prompt line from prose-to-be-swept into a pinned consumer, which is
            the exact site whose drift produced BR-1.
          family: doc-sweep-incomplete
          round: 2
        - id: BR-11
          severity: Minor
          title: The plan was revised mid-stream with no "## Revisions" entry, and the issue's Log section is an empty date header after five gate rounds
          detail: |-
            This is the 2nd finding in family `plan-artifact-not-ticked`. 4c24763 rewrote
            Task 9 Step 1 in place; AGENTS.md section 1 requires an appended "## Revisions"
            entry (timestamp, reason, delta) rather than an overwrite, and workshop/history
            shows the convention live (000015's plan carries entries for rounds 5 and 6).
            Separately the issue's "## Log" holds only "### 2026-08-27" with no body, after
            four plan-quality rounds and one boundary round. Do not fix the two spots — state
            the rule: every gate round ends with one artifact-completion pass over issue and
            plan covering checkboxes, "## Revisions", and "## Log", so the round that flips a
            box is the round that records why.
          family: plan-artifact-not-ticked
          round: 2
        - id: BR-12
          severity: Minor
          title: The graded state's "any key moves on" rule is implemented identically in both the InputReveal and InputRune arms (ARCH-DRY)
          detail: |-
            session.go:146 and session.go:162 are the same four lines with different comments,
            implementing the single rule the prompt states as "any key = next word". The
            plan's own architecture note cites #6 BR-5 against exactly this shape while
            arguing for []Outcome over a Reveal flag. Hoisting
            `if s.Graded && (in.Kind == InputRune || in.Kind == InputReveal)` above the switch
            gives it one home and keeps InputDrop and InputQuit correctly outside it.
          family: one-rule-two-places
          round: 2
      forced: '--no-ledger (or --force): Operator tested the shipped flow and confirmed it works. Ran define --play through a pty against a copy of the real deck: grading keys offered up front, y advancing with no definition and no audio, n showing the definition with "any key = next word", space moving on, Ctrl-C reporting "1 right, 1 wrong", zero bare newlines. go build, go vet, gofmt, go test ./... green; full -tags conformance pty suite green. Nine mutations run, each reddening the test that names it. --no-ledger: BR-1 and BR-2 are both FIXED in this same commit per the FIX-THEN-SHIP protocol (#174) rather than disposed — the form doc comments in cmd/define/play now describe grade-first, the plan sweep recurses over directories, and all 42 plan steps are ticked. Both Minors were also taken: DEFINE_CONFORMANCE_STRICT turns a no-pty skip into a failure, verified by running it where pty allocation is denied, and the Done-when now records the delivered player mechanism.'
      blocked: true
    - "n": 3
      timestamp: "2026-08-27T18:55:55-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: recall.go:3, question.go:60/74 and the play_loop_test citation all rewritten; the plan's recursive sweep returns zero stale hits.
          round: 3
        - id: BR-3
          disposition: addressed
          note: README.md:71-73 now states audio is fetched only on a reveal.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Word carried on both OutcomeReveal sites, loop reads out.Word; stripping Word reddens TestEveryWordOutcomeNamesItsWord.
          round: 3
        - id: BR-5
          disposition: addressed
          note: audible(&d, &opt) at five sites; the forgettable d.audio line now lives in one place.
          round: 3
        - id: BR-8
          disposition: addressed
          note: 'Verified by mutation: both outs[:1] and outs[len(outs)-1:] redden TestAMissPlaysThePronunciationAndRecordsIt.'
          round: 3
        - id: BR-9
          disposition: addressed
          note: Ten sites route through skipOrFail; measured here, non-strict ok with 7 pty skips, strict fails naming each.
          round: 3
        - id: BR-10
          disposition: addressed
          note: doc_sync_test.go makes README a consumer of the prompt consts; atlas table now lists seven seams.
          round: 3
        - id: BR-11
          disposition: addressed
          note: Four Revisions entries and a full Log entry; see the new Minor for the one scope item the pass missed.
          round: 3
        - id: BR-12
          disposition: addressed
          note: Rule hoisted above the switch; three narrowing mutations each redden a different named test.
          round: 3
      findings:
        - id: BR-13
          severity: Important
          title: The loop may perform an input's outcomes in any order and the whole suite stays green
          detail: |-
            This is the 2nd finding in family widened-contract-unpinned-at-the-consumer.
            BR-8 swept the slice's MEMBERSHIP at the consumer; ORDER is the enumerable
            sibling and was left in the tree. Measured: rewriting play_loop.go:120 to
            `for i := len(outs) - 1; i >= 0; i--` leaves go test ./cmd/define/
            ./cmd/define/play/ fully green, though session.go:127 and the issue's
            Done-when both call the ordering load-bearing. Failure: on a miss the
            reveal arm restores the terminal and shells out to afplay, and re-entry
            failure takes the early `return 1` at play_loop.go:167 with the record not
            yet performed, so the miss is lost and the process exits 1. Write the
            enumeration the widened contract implies and pin each row at the consumer;
            the order row is three lines on TestLosingTheTerminalAfterPlaybackExitsOne
            (play_loop_test.go:529) — drive keysFor("n"), keep st, assert one
            reviewEvent. Verified: green today, red under the reversal.
          family: widened-contract-unpinned-at-the-consumer
          round: 3
        - id: BR-14
          severity: Important
          title: The atlas pass missed the round's own new doc-derivation convention, and describes an InputReveal case BR-12 removed
          detail: |-
            This is the 4th finding in family doc-sweep-incomplete. Earlier rounds
            fixed instances and BR-10 shipped the general fix for the DERIVABLE half
            (make the restatement a consumer); this is the other half, so state the
            rule rather than patching these two spots. (i) doc_sync_test.go plus the
            gradePrompt/gradedPrompt consts introduce a convention that constrains
            every future prompt edit, and it appears nowhere in atlas/ or README.md —
            while the sibling convention born in the same round (skipOrFail +
            DEFINE_CONFORMANCE_STRICT) got a full atlas paragraph at
            atlas/define.md:1009-1024. (ii) atlas/define.md:1319 says the graded state
            "needs its own case there" in the InputReveal arm, the shape BR-12 removed
            later in the same range. Rule: the atlas enumeration must be produced by
            something that ran — new package-level identifiers, env vars and test
            conventions taken from git diff over the window, each grepped against
            atlas/ and either documented or explicitly waived.
          family: doc-sweep-incomplete
          round: 3
        - id: BR-15
          severity: Important
          title: lessons.md now holds four mutation-restore rules giving three different answers, two added this round
          detail: |-
            This is the 2nd finding in family one-rule-two-places, so the ask is the
            rule, not the edit. Measured prevalence, four entries: :1080 "commit, then
            mutate, then git checkout — neither half works alone"; :1739 "back up with
            git, not with cp to /tmp"; :1820 (new) "the baseline is a commit, not a
            working tree"; :1939 (new) "snapshot to a temp file and restore from that".
            :1820 re-derives :1080 and then mis-describes it ("the previous lesson
            stopped one clause short" — it did not), and :1939 contradicts :1739,
            :1107 and its own sibling 119 lines above. AGENTS.md section 4 has agents
            read this file at session start, so the next mutation round is handed
            contradictory instructions for the exact procedure that burned this one
            (ARCH-DRY). Rule: before appending a lesson, grep lessons.md for the rule
            you are about to state; if it exists, REVISE that entry with the new
            evidence instead of appending a sibling, and resolve any contradiction
            inside the one entry.
          family: one-rule-two-places
          round: 3
        - id: BR-16
          severity: Minor
          title: render_test.go skips on an absent COMMITTED fixture, and the skipOrFail carve-out excludes it by filename rather than by the question
          detail: |-
            This is the 3rd finding in family check-that-cannot-fail-reads-as-green.
            skipOrFail's doc excludes render_test.go on the grounds that "table rows
            that do not apply are not absent dependencies", but render_test.go:205
            skips when a committed fixture is missing. Measured: deleting
            cmd/define/testdata/entries/subject.txt makes TestCorpusBlockStructure
            /subject skip and the package still report ok, silently retiring the
            phantom-block coverage that fixture exists for. The rule is three classes,
            not two — absent EXTERNAL dependency (skip; fail under strict), absent
            IN-REPO artifact (always fail, which doc_sync_test.go:29 already gets
            right), shape drift (always fail) — and each exclusion should be justified
            by asking the site that question rather than by naming the file.
          family: check-that-cannot-fail-reads-as-green
          round: 3
        - id: BR-17
          severity: Minor
          title: The plan's Revisions section records three of the four scope-growth items from this round
          detail: |-
            This is the 3rd finding in family plan-artifact-not-ticked. The four
            entries cover BR-1, BR-11, BR-9 and BR-4; the BR-10 work is absent though
            it added a file the plan never named (cmd/define/doc_sync_test.go) and two
            new package consts, and left Task 7 Step 3's draw sketch showing the
            literals inline. Same rule as the atlas finding: the Revisions enumeration
            is built from git diff --name-status over the round's commits, not from
            memory — the round that grows the scope is the round that records what
            grew.
          family: plan-artifact-not-ticked
          round: 3
        - id: BR-18
          severity: Minor
          title: atlas/define.md still says "The third is the least obvious" after the table grew from three rows to seven
          detail: |-
            atlas/define.md:1002 correctly changed to "Every seam has one", but the
            paragraph below it opens with an ordinal into the old three-row table.
            Name the check (player_conformance_test.go) instead of its position, per
            the round's own "cite by NAME, never by line number" lesson.
          family: doc-sweep-incomplete
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-27T19:21:42-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
---

# Gate ledger — tools#24 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-27T16:42:01-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `doc-sweep-incomplete` The flow reversal reached README and atlas but not the form's own doc comments in cmd/define/play
  recall.go:3 still says "show the word, let the learner try to remember it,
  reveal the definition, self-rate" — the exact order #24 inverted. Same class:
  question.go:60 ("Reveal is what they see after asking to see it"),
  question.go:68 ("Enter and space reveal"), and play_loop_test.go:595,600
  ("play_loop.go:182 — space reveals", now :194 after this diff). All four are
  returned by the plan's own Task 9 grep pattern over cmd/define/, and the
  issue's Done-when claims the site list was built from that grep. Sweep the
  enumeration in one pass, not just the site named here.
- **BR-2** [Important] `plan-artifact-not-ticked` The durable plan has all 42 step checkboxes unticked at close
  6c29da2 ticked the issue's Plan rows only; workshop/plans/000024-play-grade-first-plan.md
  still reads as unstarted. Every recently closed plan in workshop/history/plans/
  is fully ticked (000021 52/0, 000016 60/0, 000015 39/0), so this breaks the
  standing convention and the plan artifact no longer records what was delivered.
- **BR-3** [Minor] `doc-sweep-incomplete` README still implies pronunciation audio is fetched for every word in a sitting
  README.md:65-66 reads "Pronunciation audio IS fetched over the network as each
  word is revealed". Under grade-first a `y` reveals nothing and so fetches and
  plays nothing — the issue's headline speedup, unstated in the README.
- **BR-4** [Minor] `outcome-carries-its-own-subject` The OutcomeReveal arm would nil-panic, not just mis-name the word, if an input ever advanced and revealed
  play_loop.go:143 calls s.Current().Word() after Apply returned. Correct today
  (verified: OutcomeReveal is emitted only from the two non-advancing paths), but
  Current() returns a nil interface once the session is Done, so the failure mode
  is a panic rather than the wrong word the comment describes. Carrying Word on
  OutcomeReveal, as OutcomeRecord and OutcomeDrop already do, removes it.
- **BR-5** [Minor] `duplicated-test-rig` Four loop tests repeat the same five-line audio rig instead of sharing a helper
  TestCorrectAnswerPlaysNoAudio, TestAMissPlaysThePronunciation,
  TestRevealPlaysThePronunciationByDefault and TestNoAudioSilencesTheSession each
  re-set opt.noAudio/times, a fakePlayer and okAudio{}. An audioRig(t, word)
  helper would make the PQ-6 trap (a rig whose player is unreachable) impossible
  to reintroduce rather than merely fixed at four sites (ARCH-DRY).
- **BR-6** [Minor] `check-that-cannot-fail-reads-as-green` The pty conformance suite SKIPs when no pty is available, so a sandboxed run reads as passing
  startDefineInDir calls t.Skipf on "no pty available: operation not permitted",
  which is what every in-sandbox run gets — `ok` plus 7 SKIPs. Verified: the suite
  is 7/7 PASS outside the sandbox. Given #6 found this suite red across two closes,
  an opt-in env gate (fail instead of skip when it is set) would make green mean
  "it ran" (ARCH-MOCK: a conformance check that cannot fail is not a check).
- **BR-7** [Minor] `donewhen-describes-a-mechanism-not-delivered` Done-when says the no-audio assertion uses a player double that fails if called; it records and is checked after
  The delivered mechanism (fakePlayer.Played asserted empty, with okAudio installed
  so the branch is reachable) is equivalent and mutation-verified, and the plan
  chose it deliberately per #6 BR-43. The issue text should say so.

## Round 2 — 2026-08-27T17:01:30-07:00 (claude) — BLOCKED

**Forced past** (`--force`): --no-ledger (or --force): Operator tested the shipped flow and confirmed it works. Ran define --play through a pty against a copy of the real deck: grading keys offered up front, y advancing with no definition and no audio, n showing the definition with "any key = next word", space moving on, Ctrl-C reporting "1 right, 1 wrong", zero bare newlines. go build, go vet, gofmt, go test ./... green; full -tags conformance pty suite green. Nine mutations run, each reddening the test that names it. --no-ledger: BR-1 and BR-2 are both FIXED in this same commit per the FIX-THEN-SHIP protocol (#174) rather than disposed — the form doc comments in cmd/define/play now describe grade-first, the plan sweep recurses over directories, and all 42 plan steps are ticked. Both Minors were also taken: DEFINE_CONFORMANCE_STRICT turns a no-pty skip into a failure, verified by running it where pty allocation is denied, and the Done-when now records the delivered player mechanism.

### Disposed

- BR-1 — not-addressed — recall.go and question.go fixed; play_loop_test.go:595,600 still cite play_loop.go:182 (now :195) and README:54 (now :58) — the fix's own sweep excludes _test, where the two remaining enumerated sites live.
- BR-2 — addressed — Verified: 42 of 42 step checkboxes ticked, 0 unticked (the one remaining "- [ ]" is the boilerplate that names the syntax).
- BR-3 — not-addressed — README.md:65-66 unchanged; the audio sentence still does not say a `y` fetches and plays nothing.
- BR-4 — not-addressed — play_loop.go:143 unchanged; the comment predicting "the next word" still describes a nil-interface panic, and OutcomeReveal still carries no Word.
- BR-5 — not-addressed — No audioRig helper; the rig is now at five sites, not four (play_loop_test.go:343, 360, 519, 648, 669).
- BR-6 — addressed — Verified reachable and failing: DEFINE_CONFORMANCE_STRICT=1 reddens all 7 pty tests here, where the unset run reports ok. See the new class finding.
- BR-7 — addressed — The Done-when now names fakePlayer.Played + okAudio and cites #6 BR-43; I confirmed the mechanism is mutation-verified.

### Raised

- **BR-8** [Important] `widened-contract-unpinned-at-the-consumer` The loop must perform BOTH outcomes of a miss; only the second is pinned, so dropping every miss from the event log is invisible to the whole suite
  Apply returning []Outcome exists so `n` on a hidden word both records and reveals.
  Mutating play_loop.go:120 to `for _, out := range outs[len(outs)-1:]` drops the
  OutcomeRecord — no miss ever reaches events/ or the schedule — and
  `go test ./cmd/define/ -run 'Play|Session|Miss|Correct|Drop|Reveal|Capturer|Prompt|Record|Interrupt'`
  stays ok. The pty suite cannot catch it either: it asserts "0 right, 1 wrong",
  which finish() reads from the session tally that score() sets inside Apply, and
  it makes no store assertion at all. The mirror mutation `outs[:1]` IS caught by
  TestAMissPlaysThePronunciation. Fix: that test already calls playRig and throws
  away st — keep it and assert reviewEvents(t, st) is one event with Correct false,
  so one test pins both halves of the slice.
- **BR-9** [Minor] `check-that-cannot-fail-reads-as-green` DEFINE_CONFORMANCE_STRICT is honoured at one of seven conformance skip sites, so the strict run still reports green for four suites that did not execute
  This is the 2nd finding in family `check-that-cannot-fail-reads-as-green`. The
  earlier round fixed the pty instance and lessons.md:1880 wrote the general rule
  ("any test that can skip itself needs a mode where the skip is an error"), but
  the env var is read only at pty_conformance_test.go:94. Measured prevalence:
  fetch_conformance_test.go:27,62; player_conformance_test.go:23,33;
  reflect_conformance_test.go:47; live_property_test.go:38; pty_conformance_test.go:271
  all still skip unconditionally. Do not patch another site — route every
  conformance skip through one skipOrFail(t, reason, err) helper that consults the
  env var, and enumerate with `grep -rn 't\.Skipf\?(' cmd/define/*_test.go` so the
  next suite inherits the rule.
- **BR-10** [Minor] `doc-sweep-incomplete` The new DEFINE_CONFORMANCE_STRICT surface reached no doc, and the atlas conformance table still says three seams while listing neither pty suite
  This is the 3rd finding in family `doc-sweep-incomplete`. atlas/define.md:1002
  reads "All three seams have one" over a table omitting the pty suite and this
  issue's TestPTYPlayGradeFirst; README.md:291-294 still shows the bare
  `go test -tags conformance ./...` with no mention of the strict mode. The rule
  that covers all three findings in this family is the ARCH-PURPOSE one: a
  hand-maintained restatement of a fact the code owns will drift, so make it
  derive. The cheapest enforcement available is a test asserting README.md contains
  the literal prompt string draw emits (play_loop.go:281 vs README.md:51) — that
  converts the prompt line from prose-to-be-swept into a pinned consumer, which is
  the exact site whose drift produced BR-1.
- **BR-11** [Minor] `plan-artifact-not-ticked` The plan was revised mid-stream with no "## Revisions" entry, and the issue's Log section is an empty date header after five gate rounds
  This is the 2nd finding in family `plan-artifact-not-ticked`. 4c24763 rewrote
  Task 9 Step 1 in place; AGENTS.md section 1 requires an appended "## Revisions"
  entry (timestamp, reason, delta) rather than an overwrite, and workshop/history
  shows the convention live (000015's plan carries entries for rounds 5 and 6).
  Separately the issue's "## Log" holds only "### 2026-08-27" with no body, after
  four plan-quality rounds and one boundary round. Do not fix the two spots — state
  the rule: every gate round ends with one artifact-completion pass over issue and
  plan covering checkboxes, "## Revisions", and "## Log", so the round that flips a
  box is the round that records why.
- **BR-12** [Minor] `one-rule-two-places` The graded state's "any key moves on" rule is implemented identically in both the InputReveal and InputRune arms (ARCH-DRY)
  session.go:146 and session.go:162 are the same four lines with different comments,
  implementing the single rule the prompt states as "any key = next word". The
  plan's own architecture note cites #6 BR-5 against exactly this shape while
  arguing for []Outcome over a Reveal flag. Hoisting
  `if s.Graded && (in.Kind == InputRune || in.Kind == InputReveal)` above the switch
  gives it one home and keeps InputDrop and InputQuit correctly outside it.

## Round 3 — 2026-08-27T18:55:55-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — recall.go:3, question.go:60/74 and the play_loop_test citation all rewritten; the plan's recursive sweep returns zero stale hits.
- BR-3 — addressed — README.md:71-73 now states audio is fetched only on a reveal.
- BR-4 — addressed — Word carried on both OutcomeReveal sites, loop reads out.Word; stripping Word reddens TestEveryWordOutcomeNamesItsWord.
- BR-5 — addressed — audible(&d, &opt) at five sites; the forgettable d.audio line now lives in one place.
- BR-8 — addressed — Verified by mutation: both outs[:1] and outs[len(outs)-1:] redden TestAMissPlaysThePronunciationAndRecordsIt.
- BR-9 — addressed — Ten sites route through skipOrFail; measured here, non-strict ok with 7 pty skips, strict fails naming each.
- BR-10 — addressed — doc_sync_test.go makes README a consumer of the prompt consts; atlas table now lists seven seams.
- BR-11 — addressed — Four Revisions entries and a full Log entry; see the new Minor for the one scope item the pass missed.
- BR-12 — addressed — Rule hoisted above the switch; three narrowing mutations each redden a different named test.

### Raised

- **BR-13** [Important] `widened-contract-unpinned-at-the-consumer` The loop may perform an input's outcomes in any order and the whole suite stays green
  This is the 2nd finding in family widened-contract-unpinned-at-the-consumer.
  BR-8 swept the slice's MEMBERSHIP at the consumer; ORDER is the enumerable
  sibling and was left in the tree. Measured: rewriting play_loop.go:120 to
  `for i := len(outs) - 1; i >= 0; i--` leaves go test ./cmd/define/
  ./cmd/define/play/ fully green, though session.go:127 and the issue's
  Done-when both call the ordering load-bearing. Failure: on a miss the
  reveal arm restores the terminal and shells out to afplay, and re-entry
  failure takes the early `return 1` at play_loop.go:167 with the record not
  yet performed, so the miss is lost and the process exits 1. Write the
  enumeration the widened contract implies and pin each row at the consumer;
  the order row is three lines on TestLosingTheTerminalAfterPlaybackExitsOne
  (play_loop_test.go:529) — drive keysFor("n"), keep st, assert one
  reviewEvent. Verified: green today, red under the reversal.
- **BR-14** [Important] `doc-sweep-incomplete` The atlas pass missed the round's own new doc-derivation convention, and describes an InputReveal case BR-12 removed
  This is the 4th finding in family doc-sweep-incomplete. Earlier rounds
  fixed instances and BR-10 shipped the general fix for the DERIVABLE half
  (make the restatement a consumer); this is the other half, so state the
  rule rather than patching these two spots. (i) doc_sync_test.go plus the
  gradePrompt/gradedPrompt consts introduce a convention that constrains
  every future prompt edit, and it appears nowhere in atlas/ or README.md —
  while the sibling convention born in the same round (skipOrFail +
  DEFINE_CONFORMANCE_STRICT) got a full atlas paragraph at
  atlas/define.md:1009-1024. (ii) atlas/define.md:1319 says the graded state
  "needs its own case there" in the InputReveal arm, the shape BR-12 removed
  later in the same range. Rule: the atlas enumeration must be produced by
  something that ran — new package-level identifiers, env vars and test
  conventions taken from git diff over the window, each grepped against
  atlas/ and either documented or explicitly waived.
- **BR-15** [Important] `one-rule-two-places` lessons.md now holds four mutation-restore rules giving three different answers, two added this round
  This is the 2nd finding in family one-rule-two-places, so the ask is the
  rule, not the edit. Measured prevalence, four entries: :1080 "commit, then
  mutate, then git checkout — neither half works alone"; :1739 "back up with
  git, not with cp to /tmp"; :1820 (new) "the baseline is a commit, not a
  working tree"; :1939 (new) "snapshot to a temp file and restore from that".
  :1820 re-derives :1080 and then mis-describes it ("the previous lesson
  stopped one clause short" — it did not), and :1939 contradicts :1739,
  :1107 and its own sibling 119 lines above. AGENTS.md section 4 has agents
  read this file at session start, so the next mutation round is handed
  contradictory instructions for the exact procedure that burned this one
  (ARCH-DRY). Rule: before appending a lesson, grep lessons.md for the rule
  you are about to state; if it exists, REVISE that entry with the new
  evidence instead of appending a sibling, and resolve any contradiction
  inside the one entry.
- **BR-16** [Minor] `check-that-cannot-fail-reads-as-green` render_test.go skips on an absent COMMITTED fixture, and the skipOrFail carve-out excludes it by filename rather than by the question
  This is the 3rd finding in family check-that-cannot-fail-reads-as-green.
  skipOrFail's doc excludes render_test.go on the grounds that "table rows
  that do not apply are not absent dependencies", but render_test.go:205
  skips when a committed fixture is missing. Measured: deleting
  cmd/define/testdata/entries/subject.txt makes TestCorpusBlockStructure
  /subject skip and the package still report ok, silently retiring the
  phantom-block coverage that fixture exists for. The rule is three classes,
  not two — absent EXTERNAL dependency (skip; fail under strict), absent
  IN-REPO artifact (always fail, which doc_sync_test.go:29 already gets
  right), shape drift (always fail) — and each exclusion should be justified
  by asking the site that question rather than by naming the file.
- **BR-17** [Minor] `plan-artifact-not-ticked` The plan's Revisions section records three of the four scope-growth items from this round
  This is the 3rd finding in family plan-artifact-not-ticked. The four
  entries cover BR-1, BR-11, BR-9 and BR-4; the BR-10 work is absent though
  it added a file the plan never named (cmd/define/doc_sync_test.go) and two
  new package consts, and left Task 7 Step 3's draw sketch showing the
  literals inline. Same rule as the atlas finding: the Revisions enumeration
  is built from git diff --name-status over the round's commits, not from
  memory — the round that grows the scope is the round that records what
  grew.
- **BR-18** [Minor] `doc-sweep-incomplete` atlas/define.md still says "The third is the least obvious" after the table grew from three rows to seven
  atlas/define.md:1002 correctly changed to "Every seam has one", but the
  paragraph below it opens with an ordinal into the old three-row table.
  Name the check (player_conformance_test.go) instead of its position, per
  the round's own "cite by NAME, never by line number" lesson.

## Round 4 — 2026-08-27T19:21:42-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

- **BR-13** [Important] `widened-contract-unpinned-at-the-consumer` The loop may perform an input's outcomes in any order and the whole suite stays green
- **BR-14** [Important] `doc-sweep-incomplete` The atlas pass missed the round's own new doc-derivation convention, and describes an InputReveal case BR-12 removed
- **BR-15** [Important] `one-rule-two-places` lessons.md now holds four mutation-restore rules giving three different answers, two added this round
- **BR-16** [Minor] `check-that-cannot-fail-reads-as-green` render_test.go skips on an absent COMMITTED fixture, and the skipOrFail carve-out excludes it by filename rather than by the question
- **BR-17** [Minor] `plan-artifact-not-ticked` The plan's Revisions section records three of the four scope-growth items from this round
- **BR-18** [Minor] `doc-sweep-incomplete` atlas/define.md still says "The third is the least obvious" after the table grew from three rows to seven
