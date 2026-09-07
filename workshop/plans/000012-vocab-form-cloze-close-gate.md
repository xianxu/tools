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
    - "n": 2
      timestamp: "2026-09-07T11:41:55-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: |-
            Keys() and gradedPromptFor both derive from the form; reverting each independently
            reddens a named test. The class-level gap (doc_sync enrollment) is raised separately.
          round: 2
        - id: BR-2
          disposition: addressed
          note: |-
            Flagging now takes the rune, so Grade never sees the key; reverting reddens
            TestAStrayDigitAfterAnsweringDoesNotRePick, which names the moved pick.
          round: 2
        - id: BR-3
          disposition: addressed
          note: |-
            err and len==0 split; reverting the warn reddens TestAnUnreadableItemsFileIsReported,
            and TestAWordWithNoItemsSaysNothing pins the silent ordinary case.
          round: 2
        - id: BR-4
          disposition: addressed
          note: |-
            The 2026-09-07 Log entry records the sweep, the pty transcript, and that the first
            "hand-run" was a programmatic render. See the new lessons.md finding for the sibling.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: 'capture.go:149 still omits Form: out.Form; set at two sites, read at zero.'
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: cloze.go:207 still copies play_loop.go:975 and still renders above clozeFor's nil check.
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: cloze.go:101 TrimSpace(Answer)=="" is still unreachable behind hasLetterOrDigit.
          round: 2
        - id: BR-8
          disposition: not-addressed
          note: 'define-learn.md:75 still reads "veto a distractor | #12".'
          round: 2
        - id: BR-9
          disposition: not-addressed
          note: No Revisions entry was added; the plan still names ReviewEvent.Flagged.
          round: 2
      findings:
        - id: BR-10
          severity: Important
          title: Cloze was never enrolled in doc_sync_test's forms slice, so neither of its prompt lines is a README consumer
          detail: |-
            2nd in this family — round 1 fixed the instance, not the class. doc_sync_test.go:56
            pins Choice and Board and its own comment names the residual ("a form added to play
            and not added to this slice is not checked here. That half is human"); this diff added
            the third shipped form and skipped it. Probed: README.md contains neither
            "1-4 = pick the word, ? = bad question, ..." nor "any key = next word, ? = bad question,
            ...", and the key table at README.md:187 never lists `?`. Do not hand-paste the lines —
            enroll every shipped form by construction and make the key table's `?` row derive from
            play.FlagKey.
          family: prompt-line-matches-live-keys
          round: 2
        - id: BR-11
          severity: Minor
          title: three more restatements now contradict the code, bringing the open family to five
          detail: |-
            3rd in this family. play/session.go:262-264 still says the keystroke reaches the form
            through Grade and the flag comes out of advance — both false since 0698b27. The plan's
            Flagging snippet (line 211) still declares Flagged(). README.md:486 says items/ has
            "Nothing writes this yet" three paragraphs below the README's own "It then writes the
            practice items". With BR-8 and BR-9 open that is five instances. Fix the rule, not the
            five: a restatement of a fact the code owns must derive from it or be swept at the
            boundary that changed it — write the close-time sweep list (issue, plan, project,
            README, atlas, and the doc comments on every symbol the diff re-shaped).
          family: stale-artifact-restatement
          round: 2
        - id: BR-12
          severity: Minor
          title: the OutcomeFlag literal is now built at two call sites, against the rule the surviving drop comment states
          detail: |-
            2nd in this family, and a regression. 0698b27 deleted flaggedBy from advance and
            open-coded the identical six-line OutcomeFlag at play/session.go:273-279 and :409-415.
            The deleted block's comment justified the single site ("the one place both mark paths
            meet ... rather than at two call sites that could drift"), and the drop's version of
            that comment is still at :498-503. With BR-6 open the rule to state is: one constructor
            per outcome/render called from N sites, never N constructions — a flagOutcome helper
            and BR-6's renderInto are the same fix twice.
          family: one-place-renders
          round: 2
        - id: BR-13
          severity: Minor
          title: lessons.md carries nothing from close-review round 1, whose commit body states the rule
          detail: |-
            2nd in this family. 46a769b added the #12 lessons section; 0698b27 fixed two
            user-facing bugs and added none, though its own body states the transferable rule
            ("a programmatic render cannot see a prompt line"; booking a deviation is not running
            it). AGENTS.md section 4 requires it. The rule covering this and BR-4: a boundary's
            durable record has more than one home — issue Log, lessons.md when a review found
            something, the plan's Revisions, the project file — and it is not written until all of
            them are.
          family: boundary-record-unwritten
          round: 2
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

## Round 2 — 2026-09-07T11:41:55-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Keys() and gradedPromptFor both derive from the form; reverting each independently
reddens a named test. The class-level gap (doc_sync enrollment) is raised separately.
- BR-2 — addressed — Flagging now takes the rune, so Grade never sees the key; reverting reddens
TestAStrayDigitAfterAnsweringDoesNotRePick, which names the moved pick.
- BR-3 — addressed — err and len==0 split; reverting the warn reddens TestAnUnreadableItemsFileIsReported,
and TestAWordWithNoItemsSaysNothing pins the silent ordinary case.
- BR-4 — addressed — The 2026-09-07 Log entry records the sweep, the pty transcript, and that the first
"hand-run" was a programmatic render. See the new lessons.md finding for the sibling.
- BR-5 — not-addressed — capture.go:149 still omits Form: out.Form; set at two sites, read at zero.
- BR-6 — not-addressed — cloze.go:207 still copies play_loop.go:975 and still renders above clozeFor's nil check.
- BR-7 — not-addressed — cloze.go:101 TrimSpace(Answer)=="" is still unreachable behind hasLetterOrDigit.
- BR-8 — not-addressed — define-learn.md:75 still reads "veto a distractor | #12".
- BR-9 — not-addressed — No Revisions entry was added; the plan still names ReviewEvent.Flagged.

### Raised

- **BR-10** [Important] `prompt-line-matches-live-keys` Cloze was never enrolled in doc_sync_test's forms slice, so neither of its prompt lines is a README consumer
  2nd in this family — round 1 fixed the instance, not the class. doc_sync_test.go:56
  pins Choice and Board and its own comment names the residual ("a form added to play
  and not added to this slice is not checked here. That half is human"); this diff added
  the third shipped form and skipped it. Probed: README.md contains neither
  "1-4 = pick the word, ? = bad question, ..." nor "any key = next word, ? = bad question,
  ...", and the key table at README.md:187 never lists `?`. Do not hand-paste the lines —
  enroll every shipped form by construction and make the key table's `?` row derive from
  play.FlagKey.
- **BR-11** [Minor] `stale-artifact-restatement` three more restatements now contradict the code, bringing the open family to five
  3rd in this family. play/session.go:262-264 still says the keystroke reaches the form
  through Grade and the flag comes out of advance — both false since 0698b27. The plan's
  Flagging snippet (line 211) still declares Flagged(). README.md:486 says items/ has
  "Nothing writes this yet" three paragraphs below the README's own "It then writes the
  practice items". With BR-8 and BR-9 open that is five instances. Fix the rule, not the
  five: a restatement of a fact the code owns must derive from it or be swept at the
  boundary that changed it — write the close-time sweep list (issue, plan, project,
  README, atlas, and the doc comments on every symbol the diff re-shaped).
- **BR-12** [Minor] `one-place-renders` the OutcomeFlag literal is now built at two call sites, against the rule the surviving drop comment states
  2nd in this family, and a regression. 0698b27 deleted flaggedBy from advance and
  open-coded the identical six-line OutcomeFlag at play/session.go:273-279 and :409-415.
  The deleted block's comment justified the single site ("the one place both mark paths
  meet ... rather than at two call sites that could drift"), and the drop's version of
  that comment is still at :498-503. With BR-6 open the rule to state is: one constructor
  per outcome/render called from N sites, never N constructions — a flagOutcome helper
  and BR-6's renderInto are the same fix twice.
- **BR-13** [Minor] `boundary-record-unwritten` lessons.md carries nothing from close-review round 1, whose commit body states the rule
  2nd in this family. 46a769b added the #12 lessons section; 0698b27 fixed two
  user-facing bugs and added none, though its own body states the transferable rule
  ("a programmatic render cannot see a prompt line"; booking a deviation is not running
  it). AGENTS.md section 4 requires it. The rule covering this and BR-4: a boundary's
  durable record has more than one home — issue Log, lessons.md when a review found
  something, the plan's Revisions, the project file — and it is not written until all of
  them are.

## Open findings

- **BR-5** [Minor] `outcome-field-unread` the flag outcome carries Form and CaptureFlag drops it
- **BR-6** [Minor] `one-place-renders` clozeAsk duplicates ask's render+marks block, and renders twice for an unusable item
- **BR-7** [Minor] `dead-guard` usableItem's TrimSpace(Answer) == "" check is unreachable
- **BR-8** [Minor] `stale-artifact-restatement` the project's model-task table still attributes the distractor veto to #12
- **BR-9** [Minor] `stale-artifact-restatement` the plan's Integration points table names ReviewEvent.Flagged; the code ships ReviewEvent.Options
- **BR-10** [Important] `prompt-line-matches-live-keys` Cloze was never enrolled in doc_sync_test's forms slice, so neither of its prompt lines is a README consumer
- **BR-11** [Minor] `stale-artifact-restatement` three more restatements now contradict the code, bringing the open family to five
- **BR-12** [Minor] `one-place-renders` the OutcomeFlag literal is now built at two call sites, against the rule the surviving drop comment states
- **BR-13** [Minor] `boundary-record-unwritten` lessons.md carries nothing from close-review round 1, whose commit body states the rule
