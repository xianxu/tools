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
    - "n": 3
      timestamp: "2026-09-07T12:02:57-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: not-addressed
          note: capture.go:145-149 still omits Form; Outcome.Form is set at session.go:277 and :412 and read at zero sites.
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: cloze.go:208 still renders before clozeFor's nil check at :213, so an items-holding word with no usable cloze item renders twice.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: cloze.go:101 still unreachable behind hasLetterOrDigit at :98.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: workshop/projects/define-learn.md:75 unchanged; the project file received no edit at all in this window.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: plan lines 92 and 101 still name ReviewEvent.Flagged; store/event.go:93 ships Options.
          round: 3
        - id: BR-10
          disposition: addressed
          note: Mutation-verified both halves; the key table's residual `?` row is raised separately as a rule-level finding.
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: 'All three sites stand (session.go:262-264, plan:211, README:498), plus a fourth: README:487 lists three event kinds and flagged is now a fourth.'
          round: 3
        - id: BR-12
          disposition: not-addressed
          note: The six-line OutcomeFlag literal is still built at session.go:273-279 and :409-415.
          round: 3
        - id: BR-13
          disposition: addressed
          note: lessons.md now carries round 2 and the multi-home rule; that rule's own project-file home is still unwritten, which BR-8 tracks.
          round: 3
      findings:
        - id: BR-14
          severity: Critical
          title: the loop registers a headword click region over the cloze prompt, so clicking the blanked sentence speaks the answer and underlines a span as wide as it
          detail: |-
            play_loop.go:234 registers RegionHeadword at Line 1, Col 0, Width visibleCells(q.Word()) for
            every form the board branch did not return on, on a premise only Choice.Prompt() satisfies;
            addRegions (screen.go:165) does not check Text is at those coordinates. Ran it: region
            {Kind:headword Word:sycophantic Line:1 Col:0 Width:11}, tty paints
            "\x1b[4mThe Times d\x1b[24mismissed the interviews as ___.", the click yields "♫ playing 1×"
            and one fakePlayer call. Three leaks before answering: the answer spoken, its length shown as
            an underline (what Blank's comment at cloze.go:13-17 exists to prevent), and the underline on
            arbitrary text (playbar.go:228 calls that worse than none). play/cloze.go:57-60 states the
            premise fails here and nothing acts on it. 2nd in this family after BR-2 — same rule, a guard
            wider than the property it needs. Do not special-case *Cloze: the rule is already at
            play_loop.go:1170 ("the form owns its own layout, and a formula here would be a second copy
            that a new form silently invalidates"). Locate the span as marksIn does, or let the form
            declare its prompt regions, and add the derived guard over docSyncForms(t) asserting every
            registered Region.Text occupies its claimed Line/Col/Width.
          family: capability-guard-too-wide
          round: 3
        - id: BR-15
          severity: Important
          title: item free text reaches the raw terminal with only whitespace collapsed, and the event log's comment claims otherwise
          detail: |-
            store/item.go:181 oneLine is strings.Join(strings.Fields(s), " ") — newlines go, ESC and BEL
            do not. Cloze.Prompt() is the first path putting Item.Stem/Answer/Distractors on a terminal.
            Probed through the real store: Items() returns
            "The aide was sycophantic\x1b[2J\x1b[H to a fault." and "ephemeral\a" unchanged, and clozeFor
            renders both into the prompt; \x1b[2J\x1b[H clears the alt screen mid-sitting. Provenance is
            model output plus a directory the README documents as inspectable and editable, so ARCH-SECURE
            applies. store/event.go:103 asserts the options are "neutralised at the store's write
            (sanitiseItem)", true of newlines only. Fix in oneLine (drop unicode.IsControl runes) — the one
            place sanitiseItem's own comment says every consumer shares — and pin it in storetest so Mem
            and YAML are both held.
          family: untrusted-text-reaches-output
          round: 3
        - id: BR-16
          severity: Important
          title: the README key table still lists no `?`, and its digit row still says only "pick the definition"
          detail: |-
            README.md:199-209 is the table a reader consults for what they can press, and its header
            promises completeness. 3rd in this family: BR-1 fixed the prompt lines, BR-10 fixed the
            enrolment that checks them, and this third home of the same fact is still hand-maintained.
            Do not hand-paste a row. The rule: every enumeration of live keys must derive from the code
            that owns them, and a new key is not shipped until every such enumeration derives — the prompt
            lines already do via gradePrompt/gradedPromptFor, this does not. Derive the `?` row from
            play.FlagKey + CanFlag and widen the digit row now that a second form grades digits, the same
            move TestREADMENamesEveryFallbackReason makes for fallbackReasons.
          family: prompt-line-matches-live-keys
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-07T12:23:35-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: not-addressed
          note: capture.go:148-150 still writes Word/Kind/Found/Options/At; Outcome.Form is set at session.go:277 and :413 and read nowhere, so every flagged event has an empty form — which ReviewEvent.Form's own doc says means "some earlier form".
          round: 4
        - id: BR-6
          disposition: not-addressed
          note: cloze.go:208 still renders and sets marks[key] before clozeFor's nil check, duplicating play_loop.go:962.
          round: 4
        - id: BR-7
          disposition: not-addressed
          note: cloze.go:101's TrimSpace(it.Answer) == "" is still unreachable behind hasLetterOrDigit at :99.
          round: 4
        - id: BR-8
          disposition: not-addressed
          note: 'projects/define-learn.md:75 still reads "veto a distractor | #12". Rolled into the family finding below.'
          round: 4
        - id: BR-9
          disposition: not-addressed
          note: plan lines 92 and 101 still name ReviewEvent.Flagged; store/event.go:103 ships Options. Rolled into the family finding below.
          round: 4
        - id: BR-11
          disposition: not-addressed
          note: All three instances stand (session.go:262-264, plan:211, README.md:499) and the close-time sweep list the finding asked for was not written; the family measured 11 instances at HEAD.
          round: 4
        - id: BR-12
          disposition: not-addressed
          note: The six-line OutcomeFlag literal is still open-coded at session.go:273-279 and :409-415.
          round: 4
        - id: BR-14
          disposition: addressed
          note: 'Verified by revert: deleting the HasPrefix check in promptRegions reddens TestAPromptRegionCoversTheTextItClaims for both *play.Cloze and *play.Board, plus TestAClozePromptOffersNoHeadwordToClick.'
          round: 4
        - id: BR-15
          disposition: addressed
          note: 'Verified by revert: removing the strings.Map from oneLine reddens TestMemConformance and TestYAMLConformance on the ESC/BEL fixtures in both implementations.'
          round: 4
        - id: BR-16
          disposition: addressed
          note: 'Verified by revert: restoring the old digit row and dropping the `?` row reddens TestREADMEKeyTableNamesEveryLiveKey by name for both of Cloze''s pairs.'
          round: 4
      findings:
        - id: BR-17
          severity: Important
          title: the forms extent is scraped by a regex that silently under-derives, so BR-10's and BR-14's guards both lose a form to a formatting choice
          detail: |-
            doc_sync_test.go:136 requires a single-letter receiver, a pointer receiver, a
            one-line body and an all-lowercase return literal, all at once; the test only
            asserts declared is a subset of enrolled, so a member the regex misses is
            silence. Measured at HEAD in a scratch copy: renaming the receiver to `cz`
            (or reformatting Form() onto three lines, still gofmt-clean) AND removing
            Cloze from docSyncForms leaves TestEveryFormIsEnrolled,
            TestREADMEQuotesThePromptsTheLoopActuallyPrints,
            TestREADMEKeyTableNamesEveryLiveKey and TestAPromptRegionCoversTheTextItClaims
            all green — BR-10's damage restored, now also carrying BR-14's Critical guard.
            Un-enrolling alone does redden, so the enrolment is fine and the derivation is
            the weak link. Fix: derive with go/parser + ast.Inspect (any FuncDecl with a
            receiver, Name == "Form", one string result, returning a BasicLit), or make it
            fail closed by asserting len(declared) == len(docSyncForms(t)).
          family: derived-extent-fails-open
          round: 4
        - id: BR-18
          severity: Important
          title: the close-time sweep list BR-11 asked for was never written, and the family grew from 5 open instances to 11 — three of them added by this round's own commits
          detail: |-
            This is the 4th finding in family stale-artifact-restatement. Earlier rounds
            fixed instances; BR-8, BR-9 and BR-11 all stand verbatim at HEAD. Do NOT fix
            these eleven sites. The rule BR-11 stated is correct — a restatement of a fact
            the code owns must derive from it or be swept at the boundary that changed it —
            and what is missing is the ENUMERATION: a close-time sweep list over issue,
            plan, project, README, atlas, and the doc comment on every symbol the diff
            reshaped. Measured at 2c67482: (1) session.go:262-264 keystroke-through-Grade;
            (2) plan:211 Flagged(); (3) README.md:499 "Nothing writes this yet";
            (4) projects/define-learn.md:75 veto attributed to #12; (5) plan:92,101
            ReviewEvent.Flagged; (6) NEW atlas/define.md:2579-2581 restates the exact
            "prompt word is line 1, column 0 / both forms put the headword on their first
            line" premise that 2c67482 deleted from the code, and there are now three forms;
            (7) NEW README.md:491 events block still reads "kinds: looked-up, asked,
            reviewed" with no `flagged` and no `options:` field, and it is the only doc a
            human reading the log has; (8) NEW atlas/define.md:611 store layout reads
            "kinds: looked-up, asked", missing reviewed, flagged, facts/ and items/;
            (9) NEW session.go:158-160 says Outcome.Form is set "in ONE place — see Apply"
            while :277 and :413 set it directly on a non-Record kind; (10) NEW
            play/optionset.go:40 — the pre-#12 text read "#38, which WILL mark the option
            lines clickable" and the Task 1 extraction flipped it to "which marks", a
            false statement of fact (only RegionHeadword and RegionOriginLang exist);
            (11) pre-existing play/choice.go:106 cites recall.go:29, deleted with form 2.1.
            Rows 6, 7 and 8 are also the docs gate: the atlas restates a removed premise and
            the README does not document the flagged kind or options: field #12 persists.
          family: stale-artifact-restatement
          round: 4
      blocked: false
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

## Round 3 — 2026-09-07T12:02:57-07:00 (claude) — BLOCKED

### Disposed

- BR-5 — not-addressed — capture.go:145-149 still omits Form; Outcome.Form is set at session.go:277 and :412 and read at zero sites.
- BR-6 — not-addressed — cloze.go:208 still renders before clozeFor's nil check at :213, so an items-holding word with no usable cloze item renders twice.
- BR-7 — not-addressed — cloze.go:101 still unreachable behind hasLetterOrDigit at :98.
- BR-8 — not-addressed — workshop/projects/define-learn.md:75 unchanged; the project file received no edit at all in this window.
- BR-9 — not-addressed — plan lines 92 and 101 still name ReviewEvent.Flagged; store/event.go:93 ships Options.
- BR-10 — addressed — Mutation-verified both halves; the key table's residual `?` row is raised separately as a rule-level finding.
- BR-11 — not-addressed — All three sites stand (session.go:262-264, plan:211, README:498), plus a fourth: README:487 lists three event kinds and flagged is now a fourth.
- BR-12 — not-addressed — The six-line OutcomeFlag literal is still built at session.go:273-279 and :409-415.
- BR-13 — addressed — lessons.md now carries round 2 and the multi-home rule; that rule's own project-file home is still unwritten, which BR-8 tracks.

### Raised

- **BR-14** [Critical] `capability-guard-too-wide` the loop registers a headword click region over the cloze prompt, so clicking the blanked sentence speaks the answer and underlines a span as wide as it
  play_loop.go:234 registers RegionHeadword at Line 1, Col 0, Width visibleCells(q.Word()) for
  every form the board branch did not return on, on a premise only Choice.Prompt() satisfies;
  addRegions (screen.go:165) does not check Text is at those coordinates. Ran it: region
  {Kind:headword Word:sycophantic Line:1 Col:0 Width:11}, tty paints
  "\x1b[4mThe Times d\x1b[24mismissed the interviews as ___.", the click yields "♫ playing 1×"
  and one fakePlayer call. Three leaks before answering: the answer spoken, its length shown as
  an underline (what Blank's comment at cloze.go:13-17 exists to prevent), and the underline on
  arbitrary text (playbar.go:228 calls that worse than none). play/cloze.go:57-60 states the
  premise fails here and nothing acts on it. 2nd in this family after BR-2 — same rule, a guard
  wider than the property it needs. Do not special-case *Cloze: the rule is already at
  play_loop.go:1170 ("the form owns its own layout, and a formula here would be a second copy
  that a new form silently invalidates"). Locate the span as marksIn does, or let the form
  declare its prompt regions, and add the derived guard over docSyncForms(t) asserting every
  registered Region.Text occupies its claimed Line/Col/Width.
- **BR-15** [Important] `untrusted-text-reaches-output` item free text reaches the raw terminal with only whitespace collapsed, and the event log's comment claims otherwise
  store/item.go:181 oneLine is strings.Join(strings.Fields(s), " ") — newlines go, ESC and BEL
  do not. Cloze.Prompt() is the first path putting Item.Stem/Answer/Distractors on a terminal.
  Probed through the real store: Items() returns
  "The aide was sycophantic\x1b[2J\x1b[H to a fault." and "ephemeral\a" unchanged, and clozeFor
  renders both into the prompt; \x1b[2J\x1b[H clears the alt screen mid-sitting. Provenance is
  model output plus a directory the README documents as inspectable and editable, so ARCH-SECURE
  applies. store/event.go:103 asserts the options are "neutralised at the store's write
  (sanitiseItem)", true of newlines only. Fix in oneLine (drop unicode.IsControl runes) — the one
  place sanitiseItem's own comment says every consumer shares — and pin it in storetest so Mem
  and YAML are both held.
- **BR-16** [Important] `prompt-line-matches-live-keys` the README key table still lists no `?`, and its digit row still says only "pick the definition"
  README.md:199-209 is the table a reader consults for what they can press, and its header
  promises completeness. 3rd in this family: BR-1 fixed the prompt lines, BR-10 fixed the
  enrolment that checks them, and this third home of the same fact is still hand-maintained.
  Do not hand-paste a row. The rule: every enumeration of live keys must derive from the code
  that owns them, and a new key is not shipped until every such enumeration derives — the prompt
  lines already do via gradePrompt/gradedPromptFor, this does not. Derive the `?` row from
  play.FlagKey + CanFlag and widen the digit row now that a second form grades digits, the same
  move TestREADMENamesEveryFallbackReason makes for fallbackReasons.

## Round 4 — 2026-09-07T12:23:35-07:00 (claude) — passed

### Disposed

- BR-5 — not-addressed — capture.go:148-150 still writes Word/Kind/Found/Options/At; Outcome.Form is set at session.go:277 and :413 and read nowhere, so every flagged event has an empty form — which ReviewEvent.Form's own doc says means "some earlier form".
- BR-6 — not-addressed — cloze.go:208 still renders and sets marks[key] before clozeFor's nil check, duplicating play_loop.go:962.
- BR-7 — not-addressed — cloze.go:101's TrimSpace(it.Answer) == "" is still unreachable behind hasLetterOrDigit at :99.
- BR-8 — not-addressed — projects/define-learn.md:75 still reads "veto a distractor | #12". Rolled into the family finding below.
- BR-9 — not-addressed — plan lines 92 and 101 still name ReviewEvent.Flagged; store/event.go:103 ships Options. Rolled into the family finding below.
- BR-11 — not-addressed — All three instances stand (session.go:262-264, plan:211, README.md:499) and the close-time sweep list the finding asked for was not written; the family measured 11 instances at HEAD.
- BR-12 — not-addressed — The six-line OutcomeFlag literal is still open-coded at session.go:273-279 and :409-415.
- BR-14 — addressed — Verified by revert: deleting the HasPrefix check in promptRegions reddens TestAPromptRegionCoversTheTextItClaims for both *play.Cloze and *play.Board, plus TestAClozePromptOffersNoHeadwordToClick.
- BR-15 — addressed — Verified by revert: removing the strings.Map from oneLine reddens TestMemConformance and TestYAMLConformance on the ESC/BEL fixtures in both implementations.
- BR-16 — addressed — Verified by revert: restoring the old digit row and dropping the `?` row reddens TestREADMEKeyTableNamesEveryLiveKey by name for both of Cloze's pairs.

### Raised

- **BR-17** [Important] `derived-extent-fails-open` the forms extent is scraped by a regex that silently under-derives, so BR-10's and BR-14's guards both lose a form to a formatting choice
  doc_sync_test.go:136 requires a single-letter receiver, a pointer receiver, a
  one-line body and an all-lowercase return literal, all at once; the test only
  asserts declared is a subset of enrolled, so a member the regex misses is
  silence. Measured at HEAD in a scratch copy: renaming the receiver to `cz`
  (or reformatting Form() onto three lines, still gofmt-clean) AND removing
  Cloze from docSyncForms leaves TestEveryFormIsEnrolled,
  TestREADMEQuotesThePromptsTheLoopActuallyPrints,
  TestREADMEKeyTableNamesEveryLiveKey and TestAPromptRegionCoversTheTextItClaims
  all green — BR-10's damage restored, now also carrying BR-14's Critical guard.
  Un-enrolling alone does redden, so the enrolment is fine and the derivation is
  the weak link. Fix: derive with go/parser + ast.Inspect (any FuncDecl with a
  receiver, Name == "Form", one string result, returning a BasicLit), or make it
  fail closed by asserting len(declared) == len(docSyncForms(t)).
- **BR-18** [Important] `stale-artifact-restatement` the close-time sweep list BR-11 asked for was never written, and the family grew from 5 open instances to 11 — three of them added by this round's own commits
  This is the 4th finding in family stale-artifact-restatement. Earlier rounds
  fixed instances; BR-8, BR-9 and BR-11 all stand verbatim at HEAD. Do NOT fix
  these eleven sites. The rule BR-11 stated is correct — a restatement of a fact
  the code owns must derive from it or be swept at the boundary that changed it —
  and what is missing is the ENUMERATION: a close-time sweep list over issue,
  plan, project, README, atlas, and the doc comment on every symbol the diff
  reshaped. Measured at 2c67482: (1) session.go:262-264 keystroke-through-Grade;
  (2) plan:211 Flagged(); (3) README.md:499 "Nothing writes this yet";
  (4) projects/define-learn.md:75 veto attributed to #12; (5) plan:92,101
  ReviewEvent.Flagged; (6) NEW atlas/define.md:2579-2581 restates the exact
  "prompt word is line 1, column 0 / both forms put the headword on their first
  line" premise that 2c67482 deleted from the code, and there are now three forms;
  (7) NEW README.md:491 events block still reads "kinds: looked-up, asked,
  reviewed" with no `flagged` and no `options:` field, and it is the only doc a
  human reading the log has; (8) NEW atlas/define.md:611 store layout reads
  "kinds: looked-up, asked", missing reviewed, flagged, facts/ and items/;
  (9) NEW session.go:158-160 says Outcome.Form is set "in ONE place — see Apply"
  while :277 and :413 set it directly on a non-Record kind; (10) NEW
  play/optionset.go:40 — the pre-#12 text read "#38, which WILL mark the option
  lines clickable" and the Task 1 extraction flipped it to "which marks", a
  false statement of fact (only RegionHeadword and RegionOriginLang exist);
  (11) pre-existing play/choice.go:106 cites recall.go:29, deleted with form 2.1.
  Rows 6, 7 and 8 are also the docs gate: the atlas restates a removed premise and
  the README does not document the flagged kind or options: field #12 persists.

## Open findings

- **BR-5** [Minor] `outcome-field-unread` the flag outcome carries Form and CaptureFlag drops it
- **BR-6** [Minor] `one-place-renders` clozeAsk duplicates ask's render+marks block, and renders twice for an unusable item
- **BR-7** [Minor] `dead-guard` usableItem's TrimSpace(Answer) == "" check is unreachable
- **BR-8** [Minor] `stale-artifact-restatement` the project's model-task table still attributes the distractor veto to #12
- **BR-9** [Minor] `stale-artifact-restatement` the plan's Integration points table names ReviewEvent.Flagged; the code ships ReviewEvent.Options
- **BR-11** [Minor] `stale-artifact-restatement` three more restatements now contradict the code, bringing the open family to five
- **BR-12** [Minor] `one-place-renders` the OutcomeFlag literal is now built at two call sites, against the rule the surviving drop comment states
- **BR-17** [Important] `derived-extent-fails-open` the forms extent is scraped by a regex that silently under-derives, so BR-10's and BR-14's guards both lose a form to a formatting choice
- **BR-18** [Important] `stale-artifact-restatement` the close-time sweep list BR-11 asked for was never written, and the family grew from 5 open instances to 11 — three of them added by this round's own commits
