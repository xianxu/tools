---
gate: boundary-review
issue: 17
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-25T20:47:40-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: D1 says checkEvidence drops a claim citing an absent word; Task 2's test keeps it with evidence pruned
          detail: |-
            Third instance in this family (PQ-1 duplicated fold, PQ-4 two count sources, PQ-6 test restating D7),
            so the deliverable is the rule, not the site: a `## Decisions` entry is the sole statement of its fact
            and task bodies cite it (D1/D7) rather than restate it. D1's sentence needs the prune-then-drop-if-empty
            semantics that `TestCheckEvidenceDropsALevelClaimWithNoSupport` and the `"mixed"` case actually specify,
            then the task bodies can cite D1 instead of paraphrasing it.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: single-source-of-truth
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-08-25T20:47:40-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: TestReflectIsIdempotent and TestReflectPreservesCorrections pass when runReflect fails and writes nothing
          detail: |-
            Neither test checks the exit code (reflect_run_test.go:79-95, :117-140). Verified in a
            scratch copy: injecting an early `return 1` for any run that finds an existing model
            leaves both PASSING, and breaking the model call outright (renaming the evidence_words
            json tag) leaves TestReflectIsIdempotent PASSING while its three siblings fail — the
            "idempotency" asserted is that two failed runs both wrote nothing. This is the exact
            vacuity the issue's Log records catching by hand. Assert code == 0 on every runReflect
            call, and assert the generated half actually changed between the two runs in the
            corrections test.
          family: vacuous-verification
          round: 2
        - id: BR-3
          severity: Important
          title: checkEvidence's usability arm covers domain name and directive but not level rationale, band, or share
          detail: |-
            reflect.go:161-191. Verified: checkEvidence with Level{Band:"x", Rationale:"placeholder",
            EvidenceWords:["certiorari"]} returns it KEPT with dropped=[] — the same stub shape the
            domain arm was written from a live failure to kill. An empty rationale renders
            "**C1** — " with a dangling em dash, and Share:42 renders "4200%", which the system
            prompt makes plausible since its own example reads "Law, 42%". Enumerate the fields a
            reader or authoring depends on and sweep them in one predicate, rather than adding a
            third hand-written arm when M2's weakness claims land.
          family: fix-the-class-not-the-instance
          round: 2
        - id: BR-4
          severity: Important
          title: The prompt golden renders a request with no schema while the wire request carries one
          detail: |-
            renderReflectPrompt returns an llm.Request without Schema, so the golden's schema section
            reads "(none)" (testdata/golden/reflect-prompt.txt:42); runReflect discards that Request
            except for .Prompt and rebuilds Name/System in llm.Task, where Run attaches the real
            schema. So reflect.go:93-95's claim that a learnerModel field change "shows up in the
            golden's diff" is false, no artifact covers SchemaFor[learnerModel] — the schema being
            exactly what produced this milestone's evidence/evidence_words bug — and
            TestReflectRequestNamesItsTask asserts a field production never sends. Set Schema in
            renderReflectPrompt and have runReflect derive Name/System/Prompt from that Request.
          family: golden-not-the-sent-request
          round: 2
        - id: BR-5
          severity: Important
          title: README's exit-code table is not swept for --reflect, which adds four producers of 1 and one of 2
          detail: |-
            README.md:184-186 introduces the table as "enumerated rather than sampled, because a list
            of examples goes stale the moment a new one is added and nothing says so". --reflect exits
            1 on a deck below the floor, on no model configured, on nothing surviving the check, and
            on store read/write failure; it exits 2 when combined with a word. None are listed. The
            same sweep misses README.md:122, where user-model.md is still annotated "optional, yours
            to write".
          family: docs-enumeration-not-swept
          round: 2
        - id: BR-6
          severity: Minor
          title: foldLookups takes a `now` parameter it never uses, and the doc comment justifies it
          detail: |-
            reflect.go:59-89 never reads `now`; :57-58 explains it as what makes the window a table
            row rather than a timing test. Drop the parameter or use it (clamping To to now).
          family: comment-outruns-code
          round: 2
        - id: BR-7
          severity: Minor
          title: The floor message says "words in the deck" but counts evidence words, and a dropped empty level prints a blank band
          detail: |-
            reflect.go:225 prints len(ev.Words) — deck intersected with the found-lookup log — as
            "%d words in the deck", so a day-file that Events warned past silently shrinks the number
            a learner is told. reflect.go:168 renders "dropped level : cites nothing, none of which is
            in the deck" when the model made no level claim at all, which is the unactionable shape
            that message exists to avoid.
          family: message-states-what-it-measures
          round: 2
        - id: BR-8
          severity: Minor
          title: spliceCorrections discards an existing file with no out-of-fence marker without saying so
          detail: |-
            usermodel.go:120-122 returns `generated` whole when firstMarkerOutsideAFence finds nothing
            — reachable if a learner deletes the heading or leaves an unterminated fence above it.
            D3's contract does not promise preservation there, but the destruction should not be
            silent; one line to errOut is enough.
          family: silent-data-loss
          round: 2
        - id: BR-9
          severity: Minor
          title: modelMeta has no Core-concepts row, and the plan claims the enumeration reconciled to empty
          detail: |-
            usermodel.go:19 defines `type modelMeta`, which the plan's own cited command
            (git diff <boundary> -- 'cmd/**/*.go' ':!*_test.go' | grep -E '^\+(func|type) ') still
            reports. The plan's final Revisions entry names eight symbols and says "Reconciled to
            empty before this commit". lessons.md:941 is this rule.
          family: entity-table-completeness
          round: 2
        - id: BR-10
          severity: Minor
          title: --reflect combined with another mode silently honours one of them
          detail: |-
            Verified: `define --forget nonexistent --reflect` runs forget only and `define --llm-check
            --reflect` runs llm-check only, neither mentioning the ignored flag — while --reflect plus
            a word is correctly rejected at main.go:327-331 on the reasoning that two commands on one
            line must not be silently resolved. The llm-check/forget pair predates this diff; this
            change adds two more. A mode-count guard in the same switch covers all three.
          family: fix-the-class-not-the-instance
          round: 2
        - id: BR-11
          severity: Minor
          title: The rendered frontmatter omits the Spec's `learner:` field and the Revisions entry does not say so
          detail: |-
            usermodel.go:47-53 emits type/updated/window/generated_by; the issue Spec's shape and plan
            Task 3 ("Frontmatter exactly as the issue's Spec shows") both include `learner: <name>`.
            Omitting it is defensible — nothing supplies a name — but the plan's Revisions entry says
            "Two departures from the plan as written" and this is a third.
          family: spec-drift-undocumented
          round: 2
      boundary: M1
      blocked: true
---

# Gate ledger — tools#17 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-25T20:47:40-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `single-source-of-truth` D1 says checkEvidence drops a claim citing an absent word; Task 2's test keeps it with evidence pruned
  Third instance in this family (PQ-1 duplicated fold, PQ-4 two count sources, PQ-6 test restating D7),
  so the deliverable is the rule, not the site: a `## Decisions` entry is the sole statement of its fact
  and task bodies cite it (D1/D7) rather than restate it. D1's sentence needs the prune-then-drop-if-empty
  semantics that `TestCheckEvidenceDropsALevelClaimWithNoSupport` and the `"mixed"` case actually specify,
  then the task bodies can cite D1 instead of paraphrasing it.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-08-25T20:47:40-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `vacuous-verification` TestReflectIsIdempotent and TestReflectPreservesCorrections pass when runReflect fails and writes nothing
  Neither test checks the exit code (reflect_run_test.go:79-95, :117-140). Verified in a
  scratch copy: injecting an early `return 1` for any run that finds an existing model
  leaves both PASSING, and breaking the model call outright (renaming the evidence_words
  json tag) leaves TestReflectIsIdempotent PASSING while its three siblings fail — the
  "idempotency" asserted is that two failed runs both wrote nothing. This is the exact
  vacuity the issue's Log records catching by hand. Assert code == 0 on every runReflect
  call, and assert the generated half actually changed between the two runs in the
  corrections test.
- **BR-3** [Important] `fix-the-class-not-the-instance` checkEvidence's usability arm covers domain name and directive but not level rationale, band, or share
  reflect.go:161-191. Verified: checkEvidence with Level{Band:"x", Rationale:"placeholder",
  EvidenceWords:["certiorari"]} returns it KEPT with dropped=[] — the same stub shape the
  domain arm was written from a live failure to kill. An empty rationale renders
  "**C1** — " with a dangling em dash, and Share:42 renders "4200%", which the system
  prompt makes plausible since its own example reads "Law, 42%". Enumerate the fields a
  reader or authoring depends on and sweep them in one predicate, rather than adding a
  third hand-written arm when M2's weakness claims land.
- **BR-4** [Important] `golden-not-the-sent-request` The prompt golden renders a request with no schema while the wire request carries one
  renderReflectPrompt returns an llm.Request without Schema, so the golden's schema section
  reads "(none)" (testdata/golden/reflect-prompt.txt:42); runReflect discards that Request
  except for .Prompt and rebuilds Name/System in llm.Task, where Run attaches the real
  schema. So reflect.go:93-95's claim that a learnerModel field change "shows up in the
  golden's diff" is false, no artifact covers SchemaFor[learnerModel] — the schema being
  exactly what produced this milestone's evidence/evidence_words bug — and
  TestReflectRequestNamesItsTask asserts a field production never sends. Set Schema in
  renderReflectPrompt and have runReflect derive Name/System/Prompt from that Request.
- **BR-5** [Important] `docs-enumeration-not-swept` README's exit-code table is not swept for --reflect, which adds four producers of 1 and one of 2
  README.md:184-186 introduces the table as "enumerated rather than sampled, because a list
  of examples goes stale the moment a new one is added and nothing says so". --reflect exits
  1 on a deck below the floor, on no model configured, on nothing surviving the check, and
  on store read/write failure; it exits 2 when combined with a word. None are listed. The
  same sweep misses README.md:122, where user-model.md is still annotated "optional, yours
  to write".
- **BR-6** [Minor] `comment-outruns-code` foldLookups takes a `now` parameter it never uses, and the doc comment justifies it
  reflect.go:59-89 never reads `now`; :57-58 explains it as what makes the window a table
  row rather than a timing test. Drop the parameter or use it (clamping To to now).
- **BR-7** [Minor] `message-states-what-it-measures` The floor message says "words in the deck" but counts evidence words, and a dropped empty level prints a blank band
  reflect.go:225 prints len(ev.Words) — deck intersected with the found-lookup log — as
  "%d words in the deck", so a day-file that Events warned past silently shrinks the number
  a learner is told. reflect.go:168 renders "dropped level : cites nothing, none of which is
  in the deck" when the model made no level claim at all, which is the unactionable shape
  that message exists to avoid.
- **BR-8** [Minor] `silent-data-loss` spliceCorrections discards an existing file with no out-of-fence marker without saying so
  usermodel.go:120-122 returns `generated` whole when firstMarkerOutsideAFence finds nothing
  — reachable if a learner deletes the heading or leaves an unterminated fence above it.
  D3's contract does not promise preservation there, but the destruction should not be
  silent; one line to errOut is enough.
- **BR-9** [Minor] `entity-table-completeness` modelMeta has no Core-concepts row, and the plan claims the enumeration reconciled to empty
  usermodel.go:19 defines `type modelMeta`, which the plan's own cited command
  (git diff <boundary> -- 'cmd/**/*.go' ':!*_test.go' | grep -E '^\+(func|type) ') still
  reports. The plan's final Revisions entry names eight symbols and says "Reconciled to
  empty before this commit". lessons.md:941 is this rule.
- **BR-10** [Minor] `fix-the-class-not-the-instance` --reflect combined with another mode silently honours one of them
  Verified: `define --forget nonexistent --reflect` runs forget only and `define --llm-check
  --reflect` runs llm-check only, neither mentioning the ignored flag — while --reflect plus
  a word is correctly rejected at main.go:327-331 on the reasoning that two commands on one
  line must not be silently resolved. The llm-check/forget pair predates this diff; this
  change adds two more. A mode-count guard in the same switch covers all three.
- **BR-11** [Minor] `spec-drift-undocumented` The rendered frontmatter omits the Spec's `learner:` field and the Revisions entry does not say so
  usermodel.go:47-53 emits type/updated/window/generated_by; the issue Spec's shape and plan
  Task 3 ("Frontmatter exactly as the issue's Spec shows") both include `learner: <name>`.
  Omitting it is defensible — nothing supplies a name — but the plan's Revisions entry says
  "Two departures from the plan as written" and this is a third.

## Open findings

- **BR-1** [Minor] `single-source-of-truth` D1 says checkEvidence drops a claim citing an absent word; Task 2's test keeps it with evidence pruned
- **BR-2** [Important] `vacuous-verification` TestReflectIsIdempotent and TestReflectPreservesCorrections pass when runReflect fails and writes nothing
- **BR-3** [Important] `fix-the-class-not-the-instance` checkEvidence's usability arm covers domain name and directive but not level rationale, band, or share
- **BR-4** [Important] `golden-not-the-sent-request` The prompt golden renders a request with no schema while the wire request carries one
- **BR-5** [Important] `docs-enumeration-not-swept` README's exit-code table is not swept for --reflect, which adds four producers of 1 and one of 2
- **BR-6** [Minor] `comment-outruns-code` foldLookups takes a `now` parameter it never uses, and the doc comment justifies it
- **BR-7** [Minor] `message-states-what-it-measures` The floor message says "words in the deck" but counts evidence words, and a dropped empty level prints a blank band
- **BR-8** [Minor] `silent-data-loss` spliceCorrections discards an existing file with no out-of-fence marker without saying so
- **BR-9** [Minor] `entity-table-completeness` modelMeta has no Core-concepts row, and the plan claims the enumeration reconciled to empty
- **BR-10** [Minor] `fix-the-class-not-the-instance` --reflect combined with another mode silently honours one of them
- **BR-11** [Minor] `spec-drift-undocumented` The rendered frontmatter omits the Spec's `learner:` field and the Revisions entry does not say so
