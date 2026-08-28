---
id: 000028
status: open
deps: [tools#17]
github_issue:
created: 2026-08-28
updated: 2026-08-28
estimate_hours:
---

# reflect: eight carried Minors from #17's close

## Problem

`#17` (learner model) closed at M1 after eight gate rounds. Eight Minor findings
were recorded past the round cap and never disposed — none is a correctness
defect in shipped behaviour, which is why the issue converged without them, but
archiving them with `#17` would have dropped them silently.

They are carried here verbatim so they are a tracked list rather than a footnote
in `workshop/history/`. `dep: tools#17` records where they came from; the issue
is closed, so nothing blocks.

**Do not treat this as a batch to grind through.** Two have teeth and the rest
are cheap; the sensible trigger is *"`--reflect` is being touched again"*, at
which point the whole list costs little. Picking them off in isolation is likely
to cost more in gate rounds than the defects are worth — which is the lesson
`#17` itself paid 33.46h against an 8.39h estimate to learn.

## Spec

### The two with teeth

- **BR-8 — `silent-data-loss`.** `usermodel.go:120-122` returns `generated` whole
  when `firstMarkerOutsideAFence` finds nothing — reachable if a learner deletes
  the Corrections heading or leaves an unterminated fence above it. D3's contract
  does not promise preservation there, but the destruction should not be SILENT.
  One line to `errOut` is enough. *This one discards a learner's own writing.*
- **BR-13 — `decision-unpinned-by-test`.** Moving `if *reflect` from
  `main.go:352` to just before `d = d.withStore(...)` passes `go test
  ./cmd/define/` entirely, while in production `--reflect` would then refuse every
  run with "define: no deck in this directory". The wider mis-siting is caught by
  `TestReflectWithAWordIsAUsageError`; the narrow one is not, and D5 was a
  blocking plan-gate finding (PQ-2). One happy-path
  `run(ctx, []string{"--reflect"}, ...)` test in a temp dir pins it.

### The rest

- **BR-1 — `single-source-of-truth`.** D1 says `checkEvidence` drops a claim
  citing an absent word; Task 2's test keeps it with evidence pruned. D1's
  sentence needs the prune-then-drop-if-empty semantics that
  `TestCheckEvidenceDropsALevelClaimWithNoSupport` and the `"mixed"` case actually
  specify. Third instance in its family, so the deliverable is the rule: a
  `## Decisions` entry is the sole statement of its fact and task bodies CITE it
  rather than restate it.
- **BR-6 — `comment-outruns-code`.** `foldLookups` takes a `now` parameter it
  never reads (`reflect.go:59-89`), and the doc comment justifies it. Drop the
  parameter or use it (clamping `To` to `now`).
- **BR-7 — `message-states-what-it-measures`.** `reflect.go:225` prints
  `len(ev.Words)` — deck intersected with the found-lookup log — as `"%d words in
  the deck"`, so a day-file that `Events` warned past silently shrinks the number
  a learner is told. Separately `reflect.go:168` renders `"dropped level : cites
  nothing, none of which is in the deck"` when the model made no level claim at
  all — the unactionable shape that message exists to avoid. **Same family as
  BR-25, which #17 DID fix**, so the rule is already stated; these are the
  remaining members.
- **BR-9 — `entity-table-completeness`.** `usermodel.go:19` defines `modelMeta`,
  which the plan's own cited enumeration command still reports, while the plan's
  final Revisions entry names eight symbols and says "Reconciled to empty".
- **BR-10 — `fix-the-class-not-the-instance`.** `--reflect` combined with another
  mode silently honours one: `define --forget nonexistent --reflect` runs forget
  only, `define --llm-check --reflect` runs llm-check only, neither mentioning the
  ignored flag — while `--reflect` plus a word IS rejected (`main.go:327-331`) on
  the reasoning that two commands on one line must not be silently resolved. The
  llm-check/forget pair predates #17; #17 added two more. A mode-count guard in
  the same switch covers all three.
- **BR-11 — `spec-drift-undocumented`.** The rendered frontmatter omits the Spec's
  `learner:` field. Omitting it is defensible — nothing supplies a name — but the
  plan's Revisions says "Two departures from the plan as written" and this is a
  third.

## Done when

- [ ] BR-8 says something when it discards a learner's file.
- [ ] BR-13's dispatch site is pinned by a happy-path run.
- [ ] Each remaining finding is either fixed or explicitly withdrawn with a
      reason recorded here — a carried finding that is silently dropped a second
      time is worse than one never filed.

## Plan

- [ ] Trigger: pick this up when `--reflect` is next being changed, not before.
- [ ] Design via `sdlc start-plan` if more than the two with teeth are taken.

## Log

### 2026-08-28

- Filed at the operator's request rather than leaving the list in
  `workshop/history/issues/000017-user-model.md`. Findings copied verbatim from
  `workshop/history/plans/000017-user-model-close-gate.md`; each was measured by
  the reviewer that raised it, not by me — **that provenance matters given #17's
  own BR-21, where a reviewer's premise was accepted and recorded as a
  measurement without being run.** Re-measure before fixing.
