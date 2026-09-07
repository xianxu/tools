---
gate: boundary-review
issue: 12
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-07T11:22:51-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: the flag key `?` is named in no on-screen prompt, and the graded prompt contradicts it
          detail: |-
            play/cloze.go:131 returns "1-4 = pick the word" and play_loop.go:1259 returns the
            constant "any key = next word, ..." once graded, so the only in-TUI documentation of
            the keys never mentions `?` and actively mis-describes it after answering. gradePrompt's
            own doc comment records the prior instance of this class. Fix in keysFor when the form
            can flag, derive livePrompt's graded branch from canFlag(q), and extend doc_sync_test.go.
          family: prompt-line-matches-live-keys
          round: 1
        - id: BR-2
          severity: Important
          title: the graded branch hands ANY rune to Grade, so a stray digit re-picks the answer on a Cloze
          detail: |-
            play/session.go:277 guards on canFlag(q) rather than on the gesture, while its own comment
            claims re-grading is what the guard prevents. Verified with a probe: after a miss, pressing
            `3` moves Cloze.chosen from 0 to 2. Inert today only because the question is already spent
            and Reveal() is not called again. Guard with `in.Rune == FlagKey && canFlag(q)`, or move the
            gesture onto the Flagging interface so the session never re-grades.
          family: capability-guard-too-wide
          round: 1
        - id: BR-3
          severity: Important
          title: clozeAsk swallows an Items() read error where the adjacent lookup failure is reported
          detail: |-
            cmd/define/cloze.go:192 treats `err != nil` and `len(items) == 0` identically and returns
            silently, four lines from play_loop.go:993 which prints a stderr diagnostic for a dictionary
            failure. A corrupt or unreadable items file makes the cloze form vanish for that word with no
            signal, for material that cost a model call to author. Split the branches and warn on the error.
          family: silent-degradation-unreported
          round: 1
        - id: BR-4
          severity: Important
          title: the issue's Log records nothing this boundary did, including the hand-run sitting
          detail: |-
            workshop/issues/000012-vocab-form-cloze.md still ends at the 2026-08-20 creation note while
            every Plan and Done-when row is ticked. The 19-property mutation sweep and its finding exist
            only in 46a769b's commit body, and the plan's Verification row "a real sitting, run by hand"
            has no evidence in the tree — the item the estimate books as named deviation 3, and the one
            that would have surfaced the missing `?` prompt.
          family: boundary-record-unwritten
          round: 1
        - id: BR-5
          severity: Minor
          title: the flag outcome carries Form and CaptureFlag drops it
          detail: |-
            Set at session.go:415 and :514, and ReviewEvent has a Form field, but capture.go:149 does not
            write it. Set at two sites, read at zero. Pass `Form: out.Form`.
          family: outcome-field-unread
          round: 1
        - id: BR-6
          severity: Minor
          title: clozeAsk duplicates ask's render+marks block, and renders twice for an unusable item
          detail: |-
            cmd/define/cloze.go:197 copies play_loop.go:975; a word holding items but no usable cloze item
            renders its entry once in clozeAsk and again in ask. Extract a shared helper, or move the render
            below clozeFor's nil check (ARCH-DRY).
          family: one-place-renders
          round: 1
        - id: BR-7
          severity: Minor
          title: usableItem's TrimSpace(Answer) == "" check is unreachable
          detail: |-
            cmd/define/cloze.go:99 cannot fire: hasLetterOrDigit at line 95 already returned false for a
            blank answer.
          family: dead-guard
          round: 1
        - id: BR-8
          severity: Minor
          title: 'the project''s model-task table still attributes the distractor veto to #12'
          detail: |-
            workshop/projects/define-learn.md:75 reads "veto a distractor | #12", which the issue's
            2026-09-04 revision moved to #10 and this plan's scope note repudiates.
          family: stale-artifact-restatement
          round: 1
        - id: BR-9
          severity: Minor
          title: the plan's Integration points table names ReviewEvent.Flagged; the code ships ReviewEvent.Options
          detail: |-
            store/event.go:93 is Options, with the flagged fact carried by Kind — better than the plan, but
            the table now names a field that does not exist. Same entry should note optionset_test.go was
            deliberately not created and that Tasks 5/6's tests landed in cmd/define/cloze_test.go.
          family: stale-artifact-restatement
          round: 1
      blocked: true
---

# Gate ledger — tools#12 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-07T11:22:51-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `prompt-line-matches-live-keys` the flag key `?` is named in no on-screen prompt, and the graded prompt contradicts it
  play/cloze.go:131 returns "1-4 = pick the word" and play_loop.go:1259 returns the
  constant "any key = next word, ..." once graded, so the only in-TUI documentation of
  the keys never mentions `?` and actively mis-describes it after answering. gradePrompt's
  own doc comment records the prior instance of this class. Fix in keysFor when the form
  can flag, derive livePrompt's graded branch from canFlag(q), and extend doc_sync_test.go.
- **BR-2** [Important] `capability-guard-too-wide` the graded branch hands ANY rune to Grade, so a stray digit re-picks the answer on a Cloze
  play/session.go:277 guards on canFlag(q) rather than on the gesture, while its own comment
  claims re-grading is what the guard prevents. Verified with a probe: after a miss, pressing
  `3` moves Cloze.chosen from 0 to 2. Inert today only because the question is already spent
  and Reveal() is not called again. Guard with `in.Rune == FlagKey && canFlag(q)`, or move the
  gesture onto the Flagging interface so the session never re-grades.
- **BR-3** [Important] `silent-degradation-unreported` clozeAsk swallows an Items() read error where the adjacent lookup failure is reported
  cmd/define/cloze.go:192 treats `err != nil` and `len(items) == 0` identically and returns
  silently, four lines from play_loop.go:993 which prints a stderr diagnostic for a dictionary
  failure. A corrupt or unreadable items file makes the cloze form vanish for that word with no
  signal, for material that cost a model call to author. Split the branches and warn on the error.
- **BR-4** [Important] `boundary-record-unwritten` the issue's Log records nothing this boundary did, including the hand-run sitting
  workshop/issues/000012-vocab-form-cloze.md still ends at the 2026-08-20 creation note while
  every Plan and Done-when row is ticked. The 19-property mutation sweep and its finding exist
  only in 46a769b's commit body, and the plan's Verification row "a real sitting, run by hand"
  has no evidence in the tree — the item the estimate books as named deviation 3, and the one
  that would have surfaced the missing `?` prompt.
- **BR-5** [Minor] `outcome-field-unread` the flag outcome carries Form and CaptureFlag drops it
  Set at session.go:415 and :514, and ReviewEvent has a Form field, but capture.go:149 does not
  write it. Set at two sites, read at zero. Pass `Form: out.Form`.
- **BR-6** [Minor] `one-place-renders` clozeAsk duplicates ask's render+marks block, and renders twice for an unusable item
  cmd/define/cloze.go:197 copies play_loop.go:975; a word holding items but no usable cloze item
  renders its entry once in clozeAsk and again in ask. Extract a shared helper, or move the render
  below clozeFor's nil check (ARCH-DRY).
- **BR-7** [Minor] `dead-guard` usableItem's TrimSpace(Answer) == "" check is unreachable
  cmd/define/cloze.go:99 cannot fire: hasLetterOrDigit at line 95 already returned false for a
  blank answer.
- **BR-8** [Minor] `stale-artifact-restatement` the project's model-task table still attributes the distractor veto to #12
  workshop/projects/define-learn.md:75 reads "veto a distractor | #12", which the issue's
  2026-09-04 revision moved to #10 and this plan's scope note repudiates.
- **BR-9** [Minor] `stale-artifact-restatement` the plan's Integration points table names ReviewEvent.Flagged; the code ships ReviewEvent.Options
  store/event.go:93 is Options, with the flagged fact carried by Kind — better than the plan, but
  the table now names a field that does not exist. Same entry should note optionset_test.go was
  deliberately not created and that Tasks 5/6's tests landed in cmd/define/cloze_test.go.

## Open findings

- **BR-1** [Important] `prompt-line-matches-live-keys` the flag key `?` is named in no on-screen prompt, and the graded prompt contradicts it
- **BR-2** [Important] `capability-guard-too-wide` the graded branch hands ANY rune to Grade, so a stray digit re-picks the answer on a Cloze
- **BR-3** [Important] `silent-degradation-unreported` clozeAsk swallows an Items() read error where the adjacent lookup failure is reported
- **BR-4** [Important] `boundary-record-unwritten` the issue's Log records nothing this boundary did, including the hand-run sitting
- **BR-5** [Minor] `outcome-field-unread` the flag outcome carries Form and CaptureFlag drops it
- **BR-6** [Minor] `one-place-renders` clozeAsk duplicates ask's render+marks block, and renders twice for an unusable item
- **BR-7** [Minor] `dead-guard` usableItem's TrimSpace(Answer) == "" check is unreachable
- **BR-8** [Minor] `stale-artifact-restatement` the project's model-task table still attributes the distractor veto to #12
- **BR-9** [Minor] `stale-artifact-restatement` the plan's Integration points table names ReviewEvent.Flagged; the code ships ReviewEvent.Options
