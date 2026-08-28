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

## Open findings

- **BR-1** [Important] `doc-sweep-incomplete` The flow reversal reached README and atlas but not the form's own doc comments in cmd/define/play
- **BR-3** [Minor] `doc-sweep-incomplete` README still implies pronunciation audio is fetched for every word in a sitting
- **BR-4** [Minor] `outcome-carries-its-own-subject` The OutcomeReveal arm would nil-panic, not just mis-name the word, if an input ever advanced and revealed
- **BR-5** [Minor] `duplicated-test-rig` Four loop tests repeat the same five-line audio rig instead of sharing a helper
- **BR-8** [Important] `widened-contract-unpinned-at-the-consumer` The loop must perform BOTH outcomes of a miss; only the second is pinned, so dropping every miss from the event log is invisible to the whole suite
- **BR-9** [Minor] `check-that-cannot-fail-reads-as-green` DEFINE_CONFORMANCE_STRICT is honoured at one of seven conformance skip sites, so the strict run still reports green for four suites that did not execute
- **BR-10** [Minor] `doc-sweep-incomplete` The new DEFINE_CONFORMANCE_STRICT surface reached no doc, and the atlas conformance table still says three seams while listing neither pty suite
- **BR-11** [Minor] `plan-artifact-not-ticked` The plan was revised mid-stream with no "## Revisions" entry, and the issue's Log section is an empty date header after five gate rounds
- **BR-12** [Minor] `one-rule-two-places` The graded state's "any key moves on" rule is implemented identically in both the InputReveal and InputRune arms (ARCH-DRY)
