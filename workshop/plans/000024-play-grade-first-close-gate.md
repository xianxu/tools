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

## Open findings

- **BR-1** [Important] `doc-sweep-incomplete` The flow reversal reached README and atlas but not the form's own doc comments in cmd/define/play
- **BR-2** [Important] `plan-artifact-not-ticked` The durable plan has all 42 step checkboxes unticked at close
- **BR-3** [Minor] `doc-sweep-incomplete` README still implies pronunciation audio is fetched for every word in a sitting
- **BR-4** [Minor] `outcome-carries-its-own-subject` The OutcomeReveal arm would nil-panic, not just mis-name the word, if an input ever advanced and revealed
- **BR-5** [Minor] `duplicated-test-rig` Four loop tests repeat the same five-line audio rig instead of sharing a helper
- **BR-6** [Minor] `check-that-cannot-fail-reads-as-green` The pty conformance suite SKIPs when no pty is available, so a sandboxed run reads as passing
- **BR-7** [Minor] `donewhen-describes-a-mechanism-not-delivered` Done-when says the no-audio assertion uses a player double that fails if called; it records and is checked after
