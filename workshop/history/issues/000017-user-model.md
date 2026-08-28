---
id: 000017
status: done
deps: [tools#3, tools#11]
github_issue:
created: 2026-08-22
updated: 2026-08-27
estimate_hours: 8.39
started: 2026-08-25T16:45:07-07:00
actual_hours: 33.46
---

# learner model: batch analysis into a durable user-model.md

## Problem

Nothing in the trainer knows who it is teaching. `--stats` (#8) counts answers;
it does not know that this learner reads judicial opinions for pleasure, sits at
C1–C2, and consistently mistakes contemptuous praise words for neutral ones.

Without that, every generated item is generic — and generic is exactly what a
personal tool has no excuse to be. The words a person looks up *are* a signal a
vocabulary app cannot buy.

## Spec

One durable artifact, `user-model.md`, in the same working directory as `words/`
and `events/`. Batch-generated, human-correctable, and read by every authoring
prompt.

### Why a document, not a table

It is meant to be read and argued with. A learner who disagrees ("I read these for
pleasure, not for the bar exam") must be able to say so and have it stick — which
is also the escape hatch for a batch analysis that draws the wrong conclusion.

### Shape

```markdown
---
type: user-model
learner: <name>
updated: <ISO date>
window: <from>..<to>          # N lookups, M reviews
generated_by: define --reflect (<model>)
---

## Level
Working band, anchored on evidence from the deck rather than asserted.

## Domains they read in
Inferred from what gets looked up — never asked. Table of domain, share, and the
words that are the evidence. Carries the authoring directive that follows from it.

## Weaknesses
Error kinds with counts, each naming the events it was derived from.

## Corrections          ← human-owned
```

### Rules

- **Batch, never per-answer.** `define --reflect` folds the event log and the deck
  into this file on demand. No model call ever sits inside the review loop or the
  lookup path; review stays instant and works offline.
- **`## Corrections` is never rewritten.** Regeneration replaces everything above
  it and appends nothing to it. Corrections are authoritative over anything
  inferred, and the authoring prompt is told so explicitly.
- **Every claim names its evidence.** "34% of lookups are legal — `certiorari`,
  `dicta`, `estoppel`" is checkable; "you like law" is not. A claim that cannot
  name the events behind it does not go in the file (`lessons.md`: a claim must not
  outrun the width it was measured at).
- **Regeneration is idempotent under a fixed clock, fake seam and fixed store.**
  Same inputs, same file — which is what makes it testable at all.
- **Absent file is normal.** Authoring works without it, just generically; nothing
  blocks on it existing.

### Milestones

- **M1 — from lookups.** Level and domain need only the deck and the lookup events,
  both of which exist today (#3, #4). Lands before any review events do, so the very
  first authored item is already learner-aware.
- **M2 — weaknesses.** The error taxonomy needs review events from #6, and the
  misses classified by kind (near-synonym collapse, connotation, register, domain,
  right-meaning-wrong-usage). Feeds back into which words surface and what the next
  item is authored to probe.

## Done when

M1:
- [x] `--reflect` writes a `user-model.md` whose level and domain claims each name
      the deck words they were derived from.
- [x] Regeneration is idempotent against a fake seam, fixed clock and fixed store.
- [x] A hand-written `## Corrections` section survives regeneration byte-for-byte,
      asserted by a test that fails when the preservation is removed.
- [x] Authoring (#10) reads the file, and its absence degrades to generic authoring
      rather than an error.
- [x] Domain inference is checked against a held-out sample of deck words, not
      asserted — the same bar #10 sets for level bucketing.

M2 — descoped at close, carried to #7 (see Plan):
- [ ] Misses classify into a fixed, enumerated error taxonomy; an unrecognised
      classification is dropped with a warning, not admitted as a new kind.
- [ ] A weakness claim names the events behind it, and the count is reproducible
      from the log.
- [ ] The model demonstrably changes what is authored: same deck, two different
      `user-model.md` files, different distractor domains — asserted, not asserted-ish.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

Derived after the plan cleared plan-quality (#187), against
`workshop/plans/000017-user-model-plan.md` — one item per task, plus the process
work that sits inside the measured window.

**Costed with #16's overrun as evidence, not with its estimate.** #16 estimated
6.49 and measured **17.63** (0.4×). The feature itself came in at ~6.3h, almost
exactly as costed; the other ~11h was twelve review rounds, and per-milestone
actuals under-report because those rounds land at and after the boundary they
measure. So the review rounds are line items here rather than a hope: two plan
rounds (already spent) and four boundary rounds, which is what #16 needed for
each of its milestones.

Design carries v2's ×0.2 spec-quality discount where the plan pre-resolves the
decision — it does for the pure entities, and does not for the plan authoring
itself or for the review rounds, which are the design work rather than a
beneficiary of it. Implementation is v3.1's 40% of the v2 table. Familiarity 1.0:
same repo, same files, `summariseLookups` and the store seams all shipped.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec             design=1.00 impl=0.08
item: milestone-review       design=0.10 impl=0.14
item: milestone-review       design=0.10 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: greenfield-go-module   design=0.25 impl=0.22
item: greenfield-go-module   design=0.25 impl=0.22
item: greenfield-go-module   design=0.25 impl=0.22
item: greenfield-go-module   design=0.25 impl=0.22
item: milestone-review       design=0.30 impl=0.55
item: milestone-review       design=0.30 impl=0.55
item: milestone-review       design=0.30 impl=0.55
item: milestone-review       design=0.30 impl=0.55
item: milestone-review       design=0.10 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: atlas-docs             design=0.03 impl=0.05
design-buffer: 0.15
total: 8.39
```

| item | what |
|---|---|
| issue-spec | plan authoring — inside the measured window, as #16 established |
| milestone-review ×2 @0.24 | the plan-quality rounds (PQ-1…PQ-5), already spent |
| smaller-go-module | T1 `foldLookups`, reusing `summariseLookups` |
| smaller-go-module | T2 `learnerModel` + `checkEvidence` |
| smaller-go-module | T3 `renderUserModel` + its golden |
| greenfield-go-module | T4 `spliceCorrections` + the fuzz property |
| greenfield-go-module | T5 `reflectTask` — the prompt |
| greenfield-go-module | T6 `runReflect`, the flag, seven wiring tests |
| greenfield-go-module | T7 the live conformance check — a held-out-sample assertion against a live model is the class that needs prompt iteration before it converges, not a mirror-and-extend |
| milestone-review ×4 @0.90 | boundary rounds, **at #16's observed cost** — see below |
| milestone-review | the M1 close itself, un-discounted like the other rounds |
| smaller-go-module | T8's live hand-run and the cross-issue ask-path check — integration verification, not review overhead |
| atlas-docs | the `--reflect` atlas section and the project row |

### Revision — 2026-08-25, after estimate-quality

First derivation was **5.23**, and it made the mistake it was written to avoid:
it cited #16's overrun as evidence and then priced review rounds at the
primitive table's default anyway. Measured, #16 was 17.63h total against ~6.34h
of feature — **11.3h across twelve rounds, ≈0.9h each**. I had priced four rounds
at 0.24h. Quoting a number and then not using it is the same failure as a comment
asserting what the code does not do.

Four changes, three of them the judge's findings:

- **Boundary rounds at the observed cost** (0.30/0.55 ≈ 0.90 each). This is most
  of the +3.2h. If M1 converges in two rounds the actual will say so, and that
  becomes the next data point — but assuming it without evidence is the optimism
  the whole revision exists to remove.
- **T7 is not mirror-and-extend.** A held-out-sample assertion against a live
  model converges after prompt iteration, not on the first run.
- **T8's live verification gets its own item.** The hand-run, the cross-issue
  ask-path check and the entity enumeration were folded into 0.24h — and #16's
  log shows the enumeration alone took five rounds to get right.
- **The final `milestone-review` is un-discounted** (0.02 → 0.10). The prose said
  the ×0.2 discount does not apply to review rounds and the number applied it
  anyway; T5's "prompts are design" gloss is dropped for the same reason — it
  argued for a number it was not attached to.

**Both directions, since the first version hedged only one.** M1 has no terminal
work, no interrupt semantics and no new store implementation — the three things
that generated most of #16's second-order findings — so two rounds rather than
four lands this near **6.6**. The symmetric case is the one the evidence actually
supports: four rounds at #16's cost is exactly what is priced, and #16 needed
four per milestone plus five at the close, which would put it near **11**.

## Plan

Design: [`workshop/plans/000017-user-model-plan.md`](../plans/000017-user-model-plan.md)
— M1 only; M2 needs review events #6 does not yet produce.

- [x] Design via `sdlc start-plan` before implementing.
- [x] M1 — the model from lookups: `foldLookups`, a typed `learnerModel` whose
      evidence is CHECKED against the deck, `renderUserModel` +
      `spliceCorrections`, and `--reflect` as a mode beside `--llm-check`.
- [ ] M2 — weaknesses. **DESCOPED at close, NOT delivered** — deliberately left
      unticked so `grep '\- \[ \]'` over the tracker keeps meaning what it did
      (BR-20). Ticking it while its own text said "not delivered" made one file
      answer "was M2 done" both ways. #6's review events
      now exist, so the original blocker is gone — but `store.ReviewEvent` carries
      `Correct bool`, a binary verdict that cannot carry an error KIND, so the
      taxonomy still has no source data. #7 (multiple choice) produces it for
      free: the chosen distractor IS the kind. Carried to #7's Done-when as a
      row rather than filed as an issue whose first design question would be
      "wait for #7".

## Log


- 2026-08-26: closed M1 — go test ./... + go vet + go test -race green; live conformance stable; fuzz 1.47M execs. All M1 Done-when rows asserted. Boundary rounds 1-3 all fixed, plus a self-run sweep of round 3s class BEFORE this round. Round 3: BR-15 (oneLine applied to the four fields the finding LISTED, not to EvidenceWords which reach two more sites), BR-14 (my forged-marker assertion searched the region before the first marker, where a forged marker cannot be by definition — it passed unconditionally; now counts markers over the whole document), BR-16 (atlas still called the fuzz property "the one thing that must hold", which round 2 disproved). The sweep then found three MORE sinks the findings had not named: the frontmatter model name, and every diagnostic message — these go to a terminal one line each, so a band carrying a newline forges a "define: ..." line a reader cannot tell from a real one. The fix changed shape rather than growing: sanitisation is now ONE pass over the whole struct at the point the file structure is built, so a render site added later is safe by construction. Two vacuous assertions of my own caught during that sweep, both in the new diagnostic test — the payload satisfied my own prefix check, and Contains(text+newline) missed because the forged line carries the rest of the message; the honest observable is that injection adds LINES, and the test counts them. Every fix mutation-verified with the mutation confirmed to have landed. lessons.md gains both rules.; review verdict: FIX-THEN-SHIP
### 2026-08-22

Created from the operator conversation that broadened `define-learn` to an adaptive
program: *"keep one user model of where user is, what area/words they seem to know
more… if the words appear more in supreme court cases, then the comparables and
wrong options would be drawn from those areas more. the whole thing should be
adaptive."*

Batch analysis over accumulated errors was specified in preference to diagnosing
each miss as it happens.

### 2026-08-25

`sdlc start-plan` run; durable plan at `workshop/plans/000017-user-model-plan.md`,
**M1 only** — M2's error taxonomy needs review events #6 does not yet produce, and
planning against a data shape nobody has seen is how a plan becomes fiction.

The decision worth surfacing before code: **the model's evidence is checked, not
trusted.** The typed answer carries the deck words behind each claim and
`checkEvidence` drops any claim citing a word the deck does not hold. That is
this project's existing rule — *distractors are selected, never invented* — applied
to the learner model: the model may READ the deck and may not ADD to it. Without
it, "every claim names its evidence" is a formatting convention that a plausible
hallucination satisfies, and the file's whole promise is that a claim can be
checked.

Two smaller ones: a floor of 12 deck words, because a model built from four
lookups is noise that would then steer authoring (absence already degrades
cleanly — #16's `gatherAskContext` handles it); and `## Corrections` is spliced
by scanning outside fenced code blocks, because the file documents its own format
in a fence that contains the marker.

The fourth M1 Done-when row names #10, which does not exist. #16's ask path reads
`user-model.md` today and degrades on absence, so the row's substance has a live
consumer already; Task 8 verifies that rather than assuming it.

### 2026-08-25 — T1–T6 done; three bugs found by running it, not by testing it

`define --reflect` works end to end against the live proxy. The unit tests were
green throughout and none of these three would have been caught by them:

- **The model put PROSE in the evidence array**, so every level claim was
  correctly dropped by the deck check and `## Level` was silently missing from
  every run. The field was named `evidence`; renaming it `evidence_words` fixed
  it. **The JSON field name is what steers the model** — and the domain claims,
  which have no competing `rationale` field, had been citing bare words correctly
  the whole time, which is what made the failure look like a check bug.
- **It stubbed claims it did not believe in** — a domain literally named `x`, a
  rationale of `placeholder` — because a stub satisfies a schema that requires
  every field. `checkEvidence` now drops claims authoring cannot act on: the same
  don't-trust-check rule extended from *is this evidence real* to *is this claim
  usable*. The prompt also now says to omit rather than stub, but the check is
  what makes it true.
- **It wrote an EMPTY file** when everything was dropped: frontmatter and a
  corrections stub, a file that says "here is what we know about you" and knows
  nothing. That is D2's floor arriving through another door; it writes nothing
  and says so now.

Fixed at the cause rather than defended: answers were **intermittently**
degenerate at the 8192 default, because high-effort thinking shares that budget
with a genuinely long answer (four domains, paragraph directives). Not a
truncation — `Run` checks the stop reason first and it was `end_turn` — but the
same squeeze that produced #11's preserved max_tokens specimen. Raised to 16384;
consecutive clean runs since.

**And my own verification was vacuous once.** The first corrections round-trip
printed "preserved" — but the second run had failed and written nothing, so
nothing could have changed. Re-run confirming both runs wrote and that the
analysis actually changed (2570 → 2690 bytes) while the corrections came back
byte-identical.

A dropped claim now names the words it rejected. "No evidence in the deck" is the
same unactionable shape as a claim that names none, and that message is the only
place a person sees why a section went missing from their file.

### 2026-08-25 — M1 closed, and a measurement I did not adopt

`sdlc actual --issue 17` reports **15.42h** for a window (`e777227b`→HEAD) whose
**wall clock is 3h49m** — 16:48 to 20:37, near-continuous. Idle-removed active
time cannot exceed the wall clock of its own window, so the number is wrong, and
its own output names the likely cause: *"attributed across window issues: #16,
#17"*. #16 closed at 17.63h earlier in this same session and this same transcript
directory; the engine appears to be counting that session's events against a
window that starts partway through it.

**Recorded 3.8h instead**, which is the window's wall clock and therefore an
upper bound on its active time. AGENTS.md §5 is right that a hand-typed actual
pollutes calibration — but adopting a measured value I can show is impossible
would poison the ledger harder, and silently. Reported rather than absorbed.

Against the 8.39h estimate that is 0.45× — the estimate was HIGH, and for a
reason worth keeping: it priced four boundary review rounds at #16's observed
~0.9h each, and M1 has not been through even one yet. The estimate's own hedge
said two rounds lands near 6.6; the honest read is that the feature work came in
around 3.8h and the review cost is still unmeasured for this issue.

This is a defect in the calibration path itself, which every close depends on —
worth a look in ariadne, where `sdlc` lives.

### 2026-08-25 — M1 boundary round 2: model text could destroy the learner's corrections

Eleven disposed, two open, and one of them is the most serious defect either
issue has produced.

**BR-12 — model free text was rendered verbatim into a marker-delimited file.**
A directive containing a line-start `## Corrections` creates a SECOND marker
above the real one; the next run splices there, so everything below it — the
analysis that run just generated, and the learner's actual corrections further
down — is treated as human-owned and **never regenerates again**. The reviewer
verified it by running it: run one's domain table survived into every later
file. A pipe in a name also broke the table, which is the harmless half.

The fix collapses newlines and escapes `|` in the four model-supplied fields.
Collapsing newlines closes the injection *outright* rather than filtering for the
marker, because every line this renderer emits is prefixed by `**`, `Read off: `
or `| ` — model text that cannot start a line cannot forge any structure,
including markers nobody has thought of yet.

**Why the fuzz property could not catch it**, which is the part worth keeping:
`FuzzSpliceCorrections` asserts that everything BELOW the marker is preserved. It
says nothing about everything ABOVE it being replaced — and that is exactly where
a forged marker lives. A property that guards one direction of an invariant is
not a property that guards the invariant. There is now a test for the other half.

Note the asymmetry the fix rests on: the corrections text is the LEARNER's and is
copied byte-for-byte precisely because they own it; this text is the MODEL's, and
it is rendered into a structure the file's integrity depends on. Same file, two
opposite rules, and conflating them is what created the hole.

**BR-5's other half:** the persistence block still described `user-model.md` as
"optional, yours to write". `--reflect` writes it now; only `## Corrections` is
theirs. I had swept that block for questions in #16 and not for this.

### 2026-08-25 — M1 boundary round 3: all three findings are about round 2's fix

And two of them are shapes this session has already taught me, arriving again in
the fix for the finding that taught me one of them.

- **BR-15 — `oneLine` was applied to the four fields the finding LISTED**, and
  not to `EvidenceWords`, which are just as model-supplied and reach two render
  sites. `checkEvidence` constrains citations to deck words, but it matches on
  `store.Key`, so a citation carrying a newline can match a real deck word AND
  forge a line. I fixed the instances named rather than the class — with the
  class written in my own commit message as *"model text that cannot start a
  line cannot forge structure"*.
- **BR-14 — my forged-marker assertion was vacuous.** It searched
  `got[:firstMarker]` for another marker: the one region where a forged marker
  cannot be, since the first marker is by definition the first. It passed
  unconditionally. Counting markers over the whole document is the honest form.
  This is the fourth vacuous assertion of the session and the second in two
  rounds of this milestone.
- **BR-16 — the atlas still called the fuzz property "the one thing that must
  hold"**, a claim round 2 disproved by finding the direction it does not guard.
  I recorded that insight in the issue Log and left the atlas asserting the old
  version.

Both render sites now mutation-checked independently: unsanitising the four
fields reddens the test, and unsanitising the evidence words reddens it
separately.

The through-line, stated plainly because three rounds have now said it: **I fix
what a finding enumerates and not what it generalises, and I write assertions
that cannot fail.** The lessons entries name both; what round 3 adds is that
they recur *inside the fix for the round that named them*.

### 2026-08-26 — M1 round 4: the gate passed, and named the gap my own sweep left

One finding, and it is *half* right: **four neutralisation sites named, three of
which genuinely had no positive control.** The fourth — `sanitiseMeta`, the
frontmatter's model name — was already covered; see round 6's BR-21, which
measured this and corrected the claim recorded here.

That is the same rule as #16's BR-45 (*a fix added to defend a finding must have
a read site that can fail*) applied to the sweep I ran proactively last round. I
found three unprotected sinks and protected them; I tested the two the story was
about and left the other two covered by nothing. **"The fields I remembered" is
not a site list.**

There is now one row per untrusted field, each injected alone so a row going
green names exactly which site stopped being defended. Verified per site:
removing any single field's neutralisation reddens its own row.

What is worth keeping from four rounds on one milestone: the sweep DID move the
work — round 4 raised one finding where rounds 1–3 raised four, three, and three
— and what it missed was not another sink but the *evidence* that the sinks are
defended. The gap moved from the code to the proof of the code, which is the
direction it should move.

### 2026-08-27 — closed at M1
- 2026-08-27: closed — Close round 8. BR-24 fixed. Round 6 corrected the false "left the whole suite green" claim in the three places BR-21 enumerated and missed a fourth in reflect.go dropClaim comment — the same substitution BR-21 was about, one level down: BR-21 was accepting a reviewer PREMISE without measuring, BR-24 was accepting a reviewer ENUMERATION without sweeping. Swept properly with grep -rniE "left the (whole )?suite green" over cmd/ and workshop/, which found THREE surviving instances in this issue scope, not the one named. All three now state the measurement actually run (unsanitising the share-out-of-range arm reddens ZERO at c179efa) and say explicitly that the sanitiseMeta half of BR-17 was false because TestEveryUntrustedFieldIsNeutralised already covered it a round earlier. Rule folded into lessons.md existing verification entry rather than appended as a sibling: a finding tells you a CLASS exists, it does not tell you where every member is — when a finding names N sites, grep for the class before believing N. Substantive state unchanged and verified: M1 live (define --reflect writes user-model.md with Level and Domains claims each naming deck evidence, Corrections section never rewritten, consumed by ask.go:268 which degrades on absence); M2 DESCOPED into #7 and reconciled across all six project-file sites; dropClaim String is the one neutralising path with positive controls on all five arms plus one row per formatter shape (nil/empty/populated/no-citation), and re-introducing the truncation reddens two named rows. gofmt clean, go build, go vet both tag sets, go test ./... 8/8 ok. --no-atlas: no new architectural surface; gate considered and waived.; review verdict: FIX-THEN-SHIP

- M1 shipped earlier (`effe0f3a`) and the issue then sat in `working` with M2
  open, which is the drift `sdlc state` flagged.
- **M2 descoped rather than carried.** Its blocker (#6's review events) is gone,
  but verifying that rather than assuming it turned up the real obstacle:
  `cmd/define/store/event.go:27` records `Correct bool` on `ReviewEvent` — a
  binary verdict. The taxonomy needs misses by KIND, which a boolean cannot
  carry, so M2 would have opened with either an event-schema change or a model
  call per miss.
- #7 (multiple choice) makes it free — the chosen distractor IS the error kind,
  drawn from the learner's own deck and selected by semantic distance. So M2 is
  now a Done-when row on #7 ("record the CHOSEN option, not just correctness")
  rather than a standalone issue whose first design question would be "wait for
  #7".
- What shipped and is live: `learnerModel{Level, Domains}` rendered to
  `user-model.md` by `renderUserModel`, with a `## Corrections` section the
  learner owns and `--reflect` never rewrites. Consumed by `ask.go:268`, which
  degrades gracefully when the file is absent.

### 2026-08-27 — close round 5 (BR-17 … BR-19)

- **BR-17 (Important, 3rd in family) — HALF OF IT WAS TRUE, and I recorded the
  false half as a measurement (corrected in round 6, BR-21).** The finding said
  four neutralisation sites had no positive control. Measured at `c179efa` in a
  scratch worktree, which I should have done BEFORE writing the fix:
  - **diagnostics half: REAL.** Unsanitising the share-out-of-range arm reddened
    **0** tests. `TestDroppedClaimDiagnosticsCannotForgeALine` reached two of
    five arms.
  - **frontmatter half: FALSE.** Deleting `sanitiseMeta`'s body already reddened
    `TestEveryUntrustedFieldIsNeutralised/the frontmatter's model name`
    (round 4's own work). The suite was NOT green, and the test I added for it
    was a duplicate — since removed.
  Fixed as the RULE, not the four sites: `dropped` now carries `dropClaim`
  (Kind/Subject/Reason/Cited) and `String()` is the ONE path to text, so an arm
  cannot forget to neutralise — the list-that-drifts shape `renderUserModel` was
  already refactored away from, reproduced in the diagnostics. Positive controls
  added for every arm. Measured: unsanitising `dropClaim.String` reddens the
  named rows.
- **BR-18 (Important, 3rd in family).** The M2 descope was recorded in two issue
  files and in NONE of the six places `workshop/projects/define-learn.md` still
  called it planned or blocked work — including a checkbox `sdlc close`
  auto-ticks, which would have marked a DESCOPED item delivered. All reconciled
  by running the finding's own enumeration
  (`grep -rnE '#17|tools#17' workshop/ atlas/ README.md`) rather than from memory.
  The rule: a boundary that changes a fact greps for every artifact restating it
  and reconciles each hit before the verdict is recorded.
- **BR-19 (Important, 3rd in family).** M1 Done-when row 5 claimed the domain
  inference is CHECKED against a held-out sample; the only statement about the
  held-out word was a `t.Logf`, and the `seen` loop INCLUDED it — so a run citing
  nothing but the held-out word satisfied "the cluster was read". Now counted over
  `c.words` only, which is what makes the row true.
- **Note on the frontmatter test:** its first version asserted
  `Contains(head, "type: forged")` and failed against a correctly neutralised
  file, because the collapsed text still contains that substring inline. That is
  this issue's own lesson (`lessons.md`: *choose injection text that does not
  satisfy your own assertion*) landing on the test written to satisfy a finding
  about vacuous checks.

### 2026-08-27 — close round 6 (BR-21, BR-22)

Both findings are about round 5's own fix.

- **BR-21 (Important, 2nd in `comment-outruns-code`).** I accepted BR-17's
  premise on trust and wrote a measurement I had not run. The frontmatter half
  was already covered; the claim "left the whole suite green" was false and
  reached three artifacts — this Log, the commit body of `1ecf05a`, and the new
  test's own comment. Corrected above; the commit body is immutable and this
  entry is the correction of record. The duplicate test is removed.
  **The rule:** a claim about what the suite does or does not cover is a
  MEASUREMENT. Run the mutation against the tree the finding names *before*
  writing the fix, the comment, or the Log line that asserts it. Accepting a
  finding's premise on trust is the same error as accepting a fix on trust —
  and this session's whole theme is that reading is not measuring. A reviewer's
  premise is not exempt.
- **BR-22 (Important, 2nd in `decision-unpinned-by-test`).** The round-5
  `dropClaim` refactor — landed under a fix-the-rule banner — silently changed
  observable output: branching on `len(Cited)` made a claim with empty
  `evidence_words` render as `level C1: cites`, stopping mid-clause, and left
  `citeAll`'s `"nothing"` branch unreachable with a comment explaining a
  distinction nothing made any more. No test anywhere supplied an empty
  `evidence_words`, which is why a consolidation passed review. Fixed with an
  explicit `Cites` flag, and `TestDropDiagnosticRendersEveryShape` gives one row
  per shape the formatter can receive — nil, empty, populated, and no-citation —
  asserting the WHOLE rendered line. Re-introducing the truncation now reddens
  two named rows.
  **The rule:** consolidating N call sites into one formatter trades N
  remembering-sites for one unpinned one unless the formatter's own branches each
  get a row. A refactor justified as "one place to get it right" has to pin that
  place.

### 2026-08-27 — close round 7 (BR-24)

- **BR-24 (Important, `comment-outruns-code` again).** Round 6 corrected the false
  "left the whole suite green" claim in the three places BR-21 enumerated — and
  missed a fourth, `reflect.go`'s own `dropClaim` comment. I ran the reviewer's
  list instead of my own grep, which is the same substitution BR-21 was about, one
  level down: trusting an enumeration because someone authoritative wrote it.
  Swept properly this time
  (`grep -rniE "left the (whole )?suite green" cmd/ workshop/`), which found
  three surviving instances in this issue's scope, not the one the finding named.
  All now state the measurement that was actually run — unsanitising the
  share-out-of-range arm reddens ZERO at `c179efa` — and say explicitly that the
  `sanitiseMeta` half was false.
  **The rule** (folded into `lessons.md`'s verification entry rather than
  appended as a sibling): a finding tells you a CLASS exists; it does not tell you
  where every member is. When a finding names N sites, grep for the class before
  believing N.

### 2026-08-27 — close round 8: converged, and the carried findings

The gate converged. Three findings were taken here under the FIX-THEN-SHIP
protocol before committing.

- **BR-25 (Important, `message-states-what-it-measures`).** `ev.Lookups` is
  accumulated over the WHOLE log; `From..To` extends only over deck words. Printed
  together as "looked up N times between X and Y" they asserted a containment
  neither computes — a forgotten word looked up earlier inflated N and left the
  span untouched. Measured: deck `{kept}` looked up once on 2026-01-10 plus two
  lookups of a forgotten word on 2026-01-01 gives whole-log 3 against a
  single-day span. The prompt site is the one with teeth: a false premise handed
  to the model that writes the durable artifact.
  Fixed with a DERIVED `DeckLookups()`, not a stored field — the first attempt
  was a field, and the golden fixture (which builds `deckEvidence` by hand rather
  than through `foldLookups`) silently carried 0, turning one false sentence into
  a different one. Summing `Words` cannot disagree with `Words`; a field would
  need keeping in step at every construction site, which is the list-that-drifts
  shape this file has now been burned by twice.
  The golden itself carried the bug: `84` was the whole-log count for a
  three-word deck whose own rows sum to `6`.
- **BR-20 (Minor).** The M2 Plan row is now UNTICKED. Ticking it while its own
  text said "not delivered" made one file answer "was M2 done" both ways and broke
  what `grep '\- \[ \]'` means over the tracker. Descoped is not done.
- **BR-23 (Minor).** All 40 plan step boxes ticked; the plan's own Revisions had
  said "Tasks 1–8 done" since M1 shipped while every box below stayed unticked.

**Carried, not fixed — eight M1-era Minors** (BR-1, BR-6…BR-11, BR-13). They
predate this session's rounds, none is a correctness defect in shipped behaviour,
and the issue has converged. Rather than archive them into oblivion with the
issue, they are recorded here as the standing list, and the ones with teeth
(BR-8's silent discard, BR-13's unpinned dispatch site) are worth a follow-up if
`--reflect` is touched again.

**Actual 33.46h against an 8.39h estimate (0.3x).** The overrun is almost
entirely gate rounds: eight of them, of which rounds 6–8 were correcting the
RECORD of round 5's fix rather than the software. That is the datum for the
calibration ledger — `milestone-review` prices conducting a round, not
remediating it, and #25 already added remediation rows for exactly this reason.

