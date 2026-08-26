# Boundary Review — tools#17 (milestone M1)

| field | value |
|-------|-------|
| issue | 17 — learner model: batch analysis into a durable user-model.md |
| repo | tools |
| issue file | workshop/issues/000017-user-model.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e777227b8ede348bc3aaf7d223b87c852582f385^..8876276df2bab56ee9cf3c2e7a72dea071486de3 |
| command | sdlc milestone-close --issue 17 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-25T20:47:40-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what the Spec promised: `define --reflect` folds deck + log into `user-model.md`, every claim carries deck words that are *checked* rather than trusted, and `## Corrections` survives regeneration — I verified that last one by reverting `spliceCorrections` in a scratch copy and watching `TestReflectPreservesCorrections`, `TestSpliceCorrections` and 6/9 fuzz seeds go red, so the Done-when row is genuinely pinned. Nothing here blocks the boundary on correctness of shipped behavior. What holds it back from SHIP is a cluster of cheap gaps I confirmed by experiment rather than by reading: two end-to-end tests stay green when the run under test fails and writes nothing (the exact vacuity the issue's own Log records catching by hand); `checkEvidence`'s usability arm covers domain `name`/`directive` but not `level.rationale` or `share`, so the stub shape it was written to kill (`{Band:"x", Rationale:"placeholder"}`) still renders; the prompt golden does not carry the schema it claims to, which is the one artifact that would have surfaced the `evidence` → `evidence_words` bug in a diff; and the README's explicitly-enumerated exit-code table was not swept for four new producers of exit 1 and one of exit 2.

## 1. Strengths

- **`foldLookups` genuinely reuses `summariseLookups`** (`cmd/define/reflect.go:76`) rather than re-folding the log — PQ-1 disposed for real, not in prose. D7's rule (deck decides membership, log decides counts) falls out of the reuse instead of needing a case, and `TestFoldLookupsTakesMembershipFromTheDeckAndCountsFromTheLog` pins the `--forget` interaction directly. ARCH-DRY passes.
- **`checkEvidence` is the right idea in the right place** (`reflect.go:147`): pure, `store.Key`-normalised so capitalised citations aren't destroyed as inventions, partial-support pruning instead of whole-claim drops, and it *returns* what it dropped so `runReflect` can say it out loud. This is the "selected, never invented" rule arriving in a second place, and it is enforced rather than documented.
- **The fuzz property is real coverage, not decoration.** Reverting preservation reddens seeds 0,1,3,4,5,6 — including the CRLF and tilde-fence shapes a table would not have reached. `TestSpliceCorrectionsIsStableOnItsOwnOutput` is the honest regression for the mid-line marker in the file's own header.
- **The live conformance test is correctly shaped** (`reflect_conformance_test.go`): held-out word per cluster, asserts *representation* not wording, skips (verified: it skips cleanly with no key) rather than reddening on a flat network, and its one hard assertion is the invariant — every backticked word above the marker is a deck word. `go vet -tags conformance` is clean, so it will not rot silently.
- **Failure paths degrade the way the rest of `define` does**: nil deck reuses `noDeckMessage`, no model reuses the ask path's sentence, and both floors prefer writing nothing to writing something confident.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `TestReflectIsIdempotent` and `TestReflectPreservesCorrections` pass when the run under test does nothing** (`cmd/define/reflect_run_test.go:79-95`, `:117-140`). Neither checks `runReflect`'s exit code. Proven twice in a scratch copy: (a) injecting an early `return 1` on any run that finds an existing model → both tests still PASS; (b) renaming the `evidence_words` JSON tag so every model call fails → `TestReflectIsIdempotent` still PASSES while all three sibling tests fail. In (b) the "idempotency" being asserted is that two failed runs both wrote nothing. This is the failure the issue's Log records catching by hand ("the second run had failed and written nothing, so nothing could have changed") — the lesson was written down but not encoded. Fix: `if code := runReflect(...); code != 0 { t.Fatalf(...) }` on every call in both tests, and in `TestReflectPreservesCorrections` also assert the analysis above the marker actually changed between runs (the hand-check used 2570 → 2690 bytes; `len(before) != len(after)` or a differing generated half is enough).

**I2 — `checkEvidence` checks two fields of a claim and leaves three unchecked; the class, not the site** (`reflect.go:161-191`). The usability arm was written from a live failure ("a domain named `x`, a rationale of `placeholder`") and applied only to `domainClaim.Name`/`Directive`. Enumerating the fields a reader or `#10` actually depends on: `level.Rationale` (unchecked), `level.Band` (unchecked — no A2–C2 constraint), `domain.Share` (unchecked). Verified: `checkEvidence(learnerModel{Level: {Band:"x", Rationale:"placeholder", EvidenceWords:["certiorari"]}}, deck)` returns the claim kept with `dropped=[]` — the exact stub shape the domain arm exists to kill; an empty rationale renders `**C1** — ` with a dangling em dash; and `Share: 42` renders `4200%`, which the system prompt makes plausible since its own example is written as `"Law, 42%"`. Fix: extend the arm — drop a level whose `Band` or `Rationale` is blank, and either clamp/reject `Share` outside `[0,1]` or say in the prompt that it is a fraction. ARCH-PURPOSE: the fix answered the instance that was observed, not the enumerable class.

**I3 — the prompt golden does not render the request that is sent** (`cmd/define/reflectprompt.go:105`/`testdata/golden/reflect-prompt.txt:42-43`, `reflect.go:93-95`, `:237-250`). `renderReflectPrompt` returns an `llm.Request` with no `Schema`, so the golden's `--- schema ---` section reads `(none)`; `runReflect` discards that Request (keeping only `.Prompt`) and rebuilds `Name`/`System` inside `llm.Task[learnerModel]`, where `Run` attaches the real schema. Two consequences: the comment at `reflect.go:93-95` ("a field added without thought shows up in the golden's diff") is false — no golden covers `SchemaFor[learnerModel]` at all, and the schema is exactly what surfaced this milestone's most expensive bug (`evidence` → `evidence_words`); and `renderReflectPrompt`'s `Task`/`System` are dead on the production path, so `TestReflectRequestNamesItsTask` asserts a field the transport never sees. #16's `ask.go:158` passes the whole `req` to `Stream`, which is why the copied comment was true there. Fix: set `Schema` in `renderReflectPrompt` (`llm.SchemaFor[learnerModel]()`) and have `runReflect` derive the Task from it (`req := renderReflectPrompt(ev)` → `Name: req.Task, System: req.System, Prompt: req.Prompt`), then re-record the golden.

**I4 — README's exit-code enumeration was not swept for the new mode** (`README.md:184-186`). The table is introduced with "What produces each is **enumerated rather than sampled**, because a list of examples goes stale the moment a new one is added and nothing says so." This diff adds four producers of `1` (deck below the floor; no model configured; nothing survived the check; deck/log/file read-write failure) and one of `2` (`--reflect` with a word) — none listed. Same sweep misses `README.md:122`, where the file tree still annotates `user-model.md` as "optional, **yours to write**", which is now only half true. The new prose at `README.md:89-101` is good; the enumerations that already existed are the consumers that weren't updated.

## 4. Minor findings

- `foldLookups`'s `now` parameter is never used (`reflect.go:59-89`), while its doc comment at `:57-58` justifies it as the thing that makes the window table-testable — a comment asserting what the code does not do. Drop the parameter or use it (e.g. clamp `To` to `now`).
- The floor message says "%d words **in the deck**" but prints `len(ev.Words)`, i.e. deck ∩ found-lookup log (`reflect.go:225`). A deck word whose day-file was skipped by `Events`' warn-and-continue path silently shrinks the number a learner is told about their deck.
- A legitimately-omitted level produces `define: dropped level : cites nothing, none of which is in the deck` (`reflect.go:168`) — an empty band renders as a blank in a message whose whole job is to be actionable. Say "level: the model made no level claim" when `Band` is empty.
- `spliceCorrections` silently discards an existing file with no out-of-fence marker (`usermodel.go:120-122`, pinned by the "an existing file with NO marker keeps nothing below" table row). Reachable if a learner deletes the heading, or leaves an unterminated fence above it. One line to `errOut` ("user-model.md had no `## Corrections` heading; it was regenerated whole") makes the loss visible.
- `modelMeta` (`usermodel.go:19`) has no row in the plan's Core-concepts tables, and the enumeration command the plan cites — `git diff <boundary> -- 'cmd/**/*.go' ':!*_test.go' | grep -E '^\+(func|type) '` — still reports it. The plan's Revisions says "Reconciled to empty before this commit"; re-running the stated command shows one row short. `workshop/lessons.md:941` is this exact rule ("a mechanical check can pass vacuously too").
- The issue Spec's frontmatter shape lists `learner: <name>` and plan Task 3 says "Frontmatter **exactly** as the issue's Spec shows"; the renderer omits it (`usermodel.go:47-53`). Reasonable — there is no name source — but it is an undocumented third departure in a Revisions entry that says "Two departures".
- `--reflect` combined with another *mode* is silently one-of-two: verified `define --forget nonexistent --reflect` runs forget only, and `define --llm-check --reflect` runs llm-check only, both without a word about the ignored flag — while `--reflect` + a *word* is correctly rejected at `main.go:327-331` on the reasoning "two commands on one line". The `--llm-check`/`--forget` pair is pre-existing; this diff adds two more. A mode-count guard in the same switch covers all three.
- The project row still reads `**actual:** measured at close` (`workshop/projects/define-learn.md`) while the issue Log records 3.8h; reconcile at close.

## 5. Test coverage notes

- Pure entities are tested without IO throughout (`reflect_test.go`, `usermodel_test.go`) — ARCH-PURE passes for the tested surface, and `runReflect` really is thin glue over them.
- `llmtest.Fake` is a wire-level httptest server, not a stubbed `Client`, and the store is a real `store.YAML` in a `t.TempDir()` — production flow and test flow share the boundary. ARCH-MOCK passes, and the live conformance check is the drift detector the principle asks for.
- Genuinely uncovered: the JSON schema (I3); a level claim with a blank/stub rationale (I2); and the `--reflect`-plus-another-mode path. The `reflectRig` fixtures do catch a `json:` tag rename indirectly, via `requireSchemaFields` — that's real, if accidental, protection.
- `TestReflectRefusesATinyDeck` asserting `len(fake.Requests()) == 0` is the right kind of assertion: it pins that the floor is checked *before* the model is called, which is the thing that matters.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass**, with the one exception in I3 (the request is constructed twice, and the golden asserts the copy that isn't sent).
- **ARCH-PURE — pass.** `runReflect` is read → fold → ask → check → render → splice → write, with every decision in a pure function. Worth keeping as the shape for `#10`'s authoring path.
- **ARCH-PURPOSE — flagged (I2).** M1's purpose is a file whose claims can be checked and acted on; the check covers what was observed failing and not the enumerable siblings. The `directive` fields are aimed at `#10`, so `share` reaching it as `4200%` is a downstream cost, not a cosmetic one.
- **ARCH-MOCK — pass.**
- For M2: `learnerModel` gains `Weaknesses`, which means `checkEvidence` grows a third arm. Fixing I2 as a per-claim "is this actionable" predicate now — rather than a third hand-written `if` later — is the difference between the class and a fourth instance.

## 7. Plan revision recommendations

A `## Revisions` entry in `workshop/plans/000017-user-model-plan.md` recording:
- **The enumeration did not reconcile to empty.** `modelMeta` has no Core-concepts row; add the row and correct the claim (the current entry names eight symbols and asserts closure).
- **A third departure from the plan:** Task 3 specifies frontmatter "exactly as the issue's Spec shows", including `learner:`; the shipped renderer omits it. State that there is no name source and that the Spec's shape is amended, or add the field.
- **`renderReflectPrompt` does not carry the schema**, so Task 5's stated property ("the golden is what gets SENT") holds for #16 and not here — record it with whichever way I3 is resolved.

```findings
findings:
  - id: new
    severity: Important
    family: vacuous-verification
    title: |
      TestReflectIsIdempotent and TestReflectPreservesCorrections pass when runReflect fails and writes nothing
    detail: |
      Neither test checks the exit code (reflect_run_test.go:79-95, :117-140). Verified in a
      scratch copy: injecting an early `return 1` for any run that finds an existing model
      leaves both PASSING, and breaking the model call outright (renaming the evidence_words
      json tag) leaves TestReflectIsIdempotent PASSING while its three siblings fail — the
      "idempotency" asserted is that two failed runs both wrote nothing. This is the exact
      vacuity the issue's Log records catching by hand. Assert code == 0 on every runReflect
      call, and assert the generated half actually changed between the two runs in the
      corrections test.
  - id: new
    severity: Important
    family: fix-the-class-not-the-instance
    title: |
      checkEvidence's usability arm covers domain name and directive but not level rationale, band, or share
    detail: |
      reflect.go:161-191. Verified: checkEvidence with Level{Band:"x", Rationale:"placeholder",
      EvidenceWords:["certiorari"]} returns it KEPT with dropped=[] — the same stub shape the
      domain arm was written from a live failure to kill. An empty rationale renders
      "**C1** — " with a dangling em dash, and Share:42 renders "4200%", which the system
      prompt makes plausible since its own example reads "Law, 42%". Enumerate the fields a
      reader or authoring depends on and sweep them in one predicate, rather than adding a
      third hand-written arm when M2's weakness claims land.
  - id: new
    severity: Important
    family: golden-not-the-sent-request
    title: |
      The prompt golden renders a request with no schema while the wire request carries one
    detail: |
      renderReflectPrompt returns an llm.Request without Schema, so the golden's schema section
      reads "(none)" (testdata/golden/reflect-prompt.txt:42); runReflect discards that Request
      except for .Prompt and rebuilds Name/System in llm.Task, where Run attaches the real
      schema. So reflect.go:93-95's claim that a learnerModel field change "shows up in the
      golden's diff" is false, no artifact covers SchemaFor[learnerModel] — the schema being
      exactly what produced this milestone's evidence/evidence_words bug — and
      TestReflectRequestNamesItsTask asserts a field production never sends. Set Schema in
      renderReflectPrompt and have runReflect derive Name/System/Prompt from that Request.
  - id: new
    severity: Important
    family: docs-enumeration-not-swept
    title: |
      README's exit-code table is not swept for --reflect, which adds four producers of 1 and one of 2
    detail: |
      README.md:184-186 introduces the table as "enumerated rather than sampled, because a list
      of examples goes stale the moment a new one is added and nothing says so". --reflect exits
      1 on a deck below the floor, on no model configured, on nothing surviving the check, and
      on store read/write failure; it exits 2 when combined with a word. None are listed. The
      same sweep misses README.md:122, where user-model.md is still annotated "optional, yours
      to write".
  - id: new
    severity: Minor
    family: comment-outruns-code
    title: |
      foldLookups takes a `now` parameter it never uses, and the doc comment justifies it
    detail: |
      reflect.go:59-89 never reads `now`; :57-58 explains it as what makes the window a table
      row rather than a timing test. Drop the parameter or use it (clamping To to now).
  - id: new
    severity: Minor
    family: message-states-what-it-measures
    title: |
      The floor message says "words in the deck" but counts evidence words, and a dropped empty level prints a blank band
    detail: |
      reflect.go:225 prints len(ev.Words) — deck intersected with the found-lookup log — as
      "%d words in the deck", so a day-file that Events warned past silently shrinks the number
      a learner is told. reflect.go:168 renders "dropped level : cites nothing, none of which is
      in the deck" when the model made no level claim at all, which is the unactionable shape
      that message exists to avoid.
  - id: new
    severity: Minor
    family: silent-data-loss
    title: |
      spliceCorrections discards an existing file with no out-of-fence marker without saying so
    detail: |
      usermodel.go:120-122 returns `generated` whole when firstMarkerOutsideAFence finds nothing
      — reachable if a learner deletes the heading or leaves an unterminated fence above it.
      D3's contract does not promise preservation there, but the destruction should not be
      silent; one line to errOut is enough.
  - id: new
    severity: Minor
    family: entity-table-completeness
    title: |
      modelMeta has no Core-concepts row, and the plan claims the enumeration reconciled to empty
    detail: |
      usermodel.go:19 defines `type modelMeta`, which the plan's own cited command
      (git diff <boundary> -- 'cmd/**/*.go' ':!*_test.go' | grep -E '^\+(func|type) ') still
      reports. The plan's final Revisions entry names eight symbols and says "Reconciled to
      empty before this commit". lessons.md:941 is this rule.
  - id: new
    severity: Minor
    family: fix-the-class-not-the-instance
    title: |
      --reflect combined with another mode silently honours one of them
    detail: |
      Verified: `define --forget nonexistent --reflect` runs forget only and `define --llm-check
      --reflect` runs llm-check only, neither mentioning the ignored flag — while --reflect plus
      a word is correctly rejected at main.go:327-331 on the reasoning that two commands on one
      line must not be silently resolved. The llm-check/forget pair predates this diff; this
      change adds two more. A mode-count guard in the same switch covers all three.
  - id: new
    severity: Minor
    family: spec-drift-undocumented
    title: |
      The rendered frontmatter omits the Spec's `learner:` field and the Revisions entry does not say so
    detail: |
      usermodel.go:47-53 emits type/updated/window/generated_by; the issue Spec's shape and plan
      Task 3 ("Frontmatter exactly as the issue's Spec shows") both include `learner: <name>`.
      Omitting it is defensible — nothing supplies a name — but the plan's Revisions entry says
      "Two departures from the plan as written" and this is a third.
```

---

## Re-review — 2026-08-25T21:04:42-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 17 — learner model: batch analysis into a durable user-model.md |
| repo | tools |
| issue file | workshop/issues/000017-user-model.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e777227b8ede348bc3aaf7d223b87c852582f385^..93515eddc91c99ed56bea0d1cb1e1b9368081bca |
| command | sdlc milestone-close --issue 17 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-25T21:04:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The four Important findings from round 2 were the right ones to take, and three are genuinely fixed — I verified each by experiment rather than by reading the commit message: injecting a failure into `runReflect` now reddens both `TestReflectIsIdempotent` and `TestReflectPreservesCorrections` (BR-2); reverting the level/share arms reddens all three subtests of `TestCheckEvidenceHoldsTheLevelToTheSameBar` (BR-3); renaming the `evidence_words` JSON tag now reddens `TestRenderReflectPrompt` because the golden carries the schema (BR-4). The implementor also correctly *rejected* half of BR-2's fix sketch — asserting the generated half changed between runs contradicts idempotency under a fixed clock — and said so in the test. What holds this back from SHIP: BR-5 was fixed at the site the finding named (the exit-code table) and not at the second site the same finding named (`README.md:122`, still "optional, yours to write"), which is the `docs-enumeration-not-swept` rule failing on its own second instance; and one new defect I demonstrated end-to-end — model free text (`Band`, `Rationale`, `Name`, `Directive`) is rendered verbatim into a markdown table and a marker-delimited file, so a newline in a directive garbles the table and a line-start `## Corrections` in any of those fields permanently freezes the generated half of `user-model.md`. Seven Minors carry forward unaddressed; none blocks.

## 1. Strengths

- **BR-2's fix is real and swept, not patched.** `mustReflect` (`cmd/define/reflect_run_test.go:265-272`) is reachable from both comparison tests, and its failure message states *why* the comparison would prove nothing. `workshop/lessons.md:965-990` generalises it into a mechanical checklist ("comparing a file before/after → does a FAILED run also satisfy it?"), which is the class-level answer the family asked for.
- **The correction inside the fix commit is the good kind.** The first BR-2 fix asserted the analysis changed between runs; the implementor's own test caught that this contradicts idempotency under a fixed clock, and the test now carries a comment saying exactly that (`reflect_run_test.go:136-140`). Rejecting a reviewer's flawed sketch with a reason beats implementing it.
- **BR-4's fix closes the loop at the source.** `renderReflectPrompt` derives the schema from `llm.SchemaFor[learnerModel]()` — the same call `llm.Run` makes at `internal/llm/task.go:35` — so the golden and the wire cannot disagree. (MaxTokens staying out of the golden is `internal/llm/render.go:21-23`'s deliberate, pre-existing decision, not a gap.)
- **`go vet ./...`, `go vet -tags conformance ./cmd/define/`, `go test ./...` and `go test -race ./cmd/define/` are all clean** at `93515ed` — I ran them.

## 2. Critical findings

None.

## 3. Important findings

**N1 — model free text is rendered into a structurally-significant document without being constrained to it** (`cmd/define/usermodel.go:56-70`). `Band`, `Rationale`, `Name` and `Directive` reach `renderUserModel` verbatim; `checkEvidence` only checks them for emptiness. Two demonstrated consequences, both verified in a scratch copy:

- A directive containing a newline breaks the table row (`| law | 50% | \`a\` | line one` / `line two | with a pipe |`). This is not hypothetical — `reflect.go:255-257` justifies `MaxTokens: 16384` precisely because the answer is "four domains with **paragraph-length** directives."
- A field containing a line-start `## Corrections` permanently freezes everything below it. I ran run-1 → learner edit → run-2 with a different model answer: run-1's domain table (`| law | 50% | ...`) survived into the run-2 file, below the injected marker, and will never regenerate. D3's contract — "everything above the marker is replaced" — silently stops holding, and the learner's real section is now preceded by frozen machine text.

`FuzzSpliceCorrections` cannot catch this: it fuzzes `existing` against a fixed `genA` and asserts *preservation below* the marker, never *regeneration above* it. Fix sketch: collapse newlines (and escape `|`) in the four model-supplied fields at render time — this repo already collapses whitespace for the same "changes no meaning" reason (`atlas/define.md:648`), and collapsing newlines closes the injection entirely, since a marker requires a line start and every rendered line is prefixed by `**`, `Read off: ` or `| `. Pin it with a case in `TestSpliceCorrectionsIsStableOnItsOwnOutput`'s neighbourhood: a rendered file whose rationale contains the marker must splice into itself unchanged. ARCH-PURPOSE — the check's stated job is "a claim a reader can check and authoring can act on"; a claim that garbles the document it lands in fails both.

**BR-5 (re-raised, not-addressed) — the enumeration was swept at one of the two sites the finding named.** The exit-code table now carries all five `--reflect` cells (`README.md:184-185`) and I confirmed each against the code. But `README.md:122` still reads `user-model.md    optional, yours to write — read to pitch answers`, which the same finding called out as "the same sweep." **This is the 2nd finding in family `docs-enumeration-not-swept`, and the second one is inside the first one.** The rule, stated: *when a change adds a producer of a documented fact, sweep every place in the docs that enumerates that fact — the exit-code table, the file-tree annotations, and the flag prose are three consumers of one model, and README's own sentence at `:182-183` is the argument for treating them as a set.* Grep for the surface name (`user-model.md`, `--reflect`) across `README.md` and `atlas/` before closing, rather than fixing the line a reviewer quoted. The remaining site needs roughly: "written by `define --reflect`; `## Corrections` is yours."

## 4. Minor findings

- **BR-7 (not-addressed), and the BR-3 fix added two more instances.** `citedOrNothing([]string{m.Level.Band})` (`reflect.go:163`) and `citedOrNothing([]string{d.Name})` (`reflect.go:190`) pass a *one-element* slice, so the `len(words) == 0` guard can never fire. Measured: an empty band prints `define: dropped level : no band or no rationale — nothing a reader could check`; a whitespace name prints `define: dropped domain   : no name or no directive — …`. Together with the two sites BR-7 already named (`reflect.go:225`'s "%d words in the deck" over deck∩log, and `reflect.go:168`'s blank band) the family now measures **four sites**. The rule: *a message names the set it measured, and renders "nothing" when the thing it is naming is absent* — `citedOrNothing` is the right helper called the wrong way; give it a single-value sibling (or trim-and-check before calling) and sweep all four.
- **BR-1 (not-addressed).** `workshop/plans/000017-user-model-plan.md`'s D1 still says `checkEvidence` "drops any claim citing a word the deck does not contain", while the shipped code and `TestCheckEvidenceDropsClaimsTheDeckCannotSupport`'s `"mixed"` case prune-then-drop-if-empty. Third instance of `single-source-of-truth`; the deliverable is D1 becoming the sole statement and the task bodies citing it.
- **BR-6 (not-addressed).** `foldLookups`'s `now` is still unread (`reflect.go:63`) while `reflect.go:57-58` justifies it.
- **BR-8 (not-addressed).** `spliceCorrections` still discards a marker-less existing file in silence (`usermodel.go:120-122`).
- **BR-9 (not-addressed).** `modelMeta` (`usermodel.go:19`) still has no Core-concepts row; the plan's cited enumeration command still reports it, and the Revisions entry still claims "Reconciled to empty."
- **BR-10 (not-addressed).** `--reflect` plus `--forget`/`--llm-check` still silently honours one; `main.go:308-310` and `main.go:348-355` are two dispatches with no mode-count guard between them.
- **BR-11 (not-addressed).** `learner:` still absent from the frontmatter and the Revisions entry still says "Two departures". A third departure, unmentioned: the Spec's `# N lookups, M reviews` ships as `# N lookups, M questions`.
- **N2 — D5's dispatch site is unpinned.** I moved `if *reflect` to just *before* `d = d.withStore(...)` and the entire suite passed, while in production `--reflect` would then refuse every run with "define: no deck in this directory". The wider mis-siting (above the arity switch) *is* caught by `TestReflectWithAWordIsAUsageError`; the narrow one is not. D5 was a blocking plan-gate finding (PQ-2); one `run(ctx, []string{"--reflect"}, …)` happy-path test in a temp dir pins it.

## 5. Test coverage notes

- The three claimed fixes are each pinned by a test that fails without them — verified by reverting, not by reading. That is the standard this gate asks for, and it was met.
- Uncovered, and each is where the next bug lands: model text containing a newline or a marker (N1); the `--reflect` happy path through `run()` (N2); `Deck()`/`Events()`/`UserModel()`/`SetUserModel()` returning errors (four `return 1` paths in `runReflect`, none exercised).
- `TestReflectRefusesATinyDeck`'s `len(fake.Requests()) != 0` assertion remains the best-shaped test in the file: it pins that the floor is checked *before* the model is called, which is the property that matters.
- `TestReflectPreservesCorrections` now proves the run succeeded, but still does not assert the *generated* half was regenerated. That is correctly delegated to `TestReflectWritesAModelFromTheDeck`; noted only so the next reader doesn't re-derive it.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** `foldLookups` → `summariseLookups`; `renderReflectPrompt` → `SchemaFor[learnerModel]`, the same call `llm.Run` makes. `runReflect` rebuilding `Name`/`System` from the `reflectTaskName`/`reflectSystem` constants rather than from the returned `Request` is duplication that cannot drift, so it is not worth a finding — but deriving them from `req` would remove the question.
- **ARCH-PURE — pass.** `runReflect` is read → fold → ask → check → render → splice → write, with every decision in a pure function; every pure entity is tested with no fake.
- **ARCH-PURPOSE — flagged (N1, BR-5).** The purpose is a file a learner can read and authoring can act on. A garbled table and a permanently-frozen section are failures of that purpose, not of formatting. On the finding-response axis: BR-5's fix took the site and left the class, and BR-3's fix — while it does sweep every field the finding enumerated — did it as three hand-written arms, which is the third arm M2's weakness claims will make a fourth.
- **ARCH-MOCK — pass.** `llmtest.Fake` is a wire-level httptest server, `store.YAML` is real in a `t.TempDir()`, and `reflect_conformance_test.go` is the drift detector, correctly skipping rather than reddening when the seam is unreachable.
- For M2: `learnerModel` gains `Weaknesses`. Both N1's sanitiser and BR-3's usability arm should become one per-claim predicate *before* that lands, or the third arm becomes a fourth and the render-safety hole is copied into a new section.

## 7. Plan revision recommendations

Still owed in `workshop/plans/000017-user-model-plan.md` (none were written this round):

- **The enumeration did not reconcile to empty.** Add the `modelMeta` row to the Pure-entities table and correct the final Revisions entry, which names eight symbols and asserts closure while the plan's own cited command still reports a ninth.
- **Departures three and four.** Task 3 specifies frontmatter "exactly as the issue's Spec shows" — `learner:` is omitted and `M reviews` shipped as `M questions`. Either amend the Spec's shape or add the fields; either way the "Two departures" sentence is wrong.
- **D1's semantics.** Restate it as prune-then-drop-if-empty (BR-1) so the task bodies can cite it instead of paraphrasing it.
- **A new entry for round 2's dispositions:** what BR-2..BR-5 changed, which half of BR-2's sketch was rejected and why, and that BR-5's `README.md:122` half remains open.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      D1 and the Task 2 body are unchanged at HEAD; the plan still says "drops any claim citing a word the deck does not contain".
  - id: BR-2
    disposition: addressed
    note: |
      Verified by injecting a failure into runReflect — both tests now redden via mustReflect; the "assert it changed" half was correctly rejected in a comment.
  - id: BR-3
    disposition: addressed
    note: |
      Verified by reverting both arms — all three TestCheckEvidenceHoldsTheLevelToTheSameBar subtests redden; residual: Band is checked non-empty but not against A2..C2.
  - id: BR-4
    disposition: addressed
    note: |
      Verified by renaming the evidence_words json tag — TestRenderReflectPrompt reddens; schema now derived from the same SchemaFor[learnerModel] llm.Run uses.
  - id: BR-5
    disposition: not-addressed
    note: |
      The exit-code table is swept and correct, but README.md:122 — named in the same finding — still reads "optional, yours to write".
  - id: BR-6
    disposition: not-addressed
    note: |
      reflect.go:63 still takes `now` and never reads it; the doc comment at :57-58 still justifies it.
  - id: BR-7
    disposition: not-addressed
    note: |
      Both original sites unchanged, and the BR-3 fix added two more — citedOrNothing over a one-element slice can never return "nothing" (measured: "dropped level : no band or no rationale").
  - id: BR-8
    disposition: not-addressed
    note: |
      usermodel.go:120-122 unchanged; a marker-less existing file is still discarded without a word.
  - id: BR-9
    disposition: not-addressed
    note: |
      No modelMeta row was added; the plan's Revisions entry still claims the enumeration reconciled to empty.
  - id: BR-10
    disposition: not-addressed
    note: |
      main.go still has no mode-count guard; --reflect with --forget or --llm-check silently honours one.
  - id: BR-11
    disposition: not-addressed
    note: |
      Frontmatter still omits `learner:`, the Revisions entry still says "Two departures", and `M reviews` shipped as `M questions` (a fourth).
findings:
  - id: new
    severity: Important
    family: model-text-unconstrained-by-format
    title: |
      Model free text is rendered verbatim into a markdown table and a marker-delimited file
    detail: |
      usermodel.go:56-70 embeds Band, Rationale, Name and Directive unmodified; checkEvidence
      only checks them for emptiness. Verified in a scratch copy: a directive containing a
      newline garbles the table row, and a field containing a line-start "## Corrections"
      permanently freezes everything below it — I ran run-1, a learner edit, then run-2 with a
      different answer and run-1's domain table survived into the new file and will never
      regenerate, silently breaking D3's "everything above the marker is replaced". Paragraph
      -length directives are exactly what reflect.go:255-257 raised MaxTokens for.
      FuzzSpliceCorrections cannot catch it: it asserts preservation below the marker, never
      regeneration above it. Collapse newlines and escape `|` in the four model-supplied
      fields at render time — collapsing newlines closes the injection outright, since every
      rendered line is prefixed by `**`, `Read off: ` or `| `.
  - id: new
    severity: Minor
    family: decision-unpinned-by-test
    title: |
      D5's dispatch site survives being moved before withStore with the whole suite green
    detail: |
      Verified: moving `if *reflect` from main.go:352 to just before `d = d.withStore(...)`
      passes go test ./cmd/define/ entirely, while in production --reflect would then refuse
      every run with "define: no deck in this directory". The wider mis-siting is caught by
      TestReflectWithAWordIsAUsageError; the narrow one is not, and D5 was a blocking
      plan-gate finding (PQ-2). One happy-path run(ctx, []string{"--reflect"}, ...) test in a
      temp dir pins it.
```

---

## Re-review — 2026-08-25T23:46:16-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 17 — learner model: batch analysis into a durable user-model.md |
| repo | tools |
| issue file | workshop/issues/000017-user-model.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e777227b8ede348bc3aaf7d223b87c852582f385^..9bbfd4b07ca177a4727c30f70be19d034e0d0485 |
| command | sdlc milestone-close --issue 17 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-25T23:46:16-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

BR-12's fix is real: `oneLine` collapses newlines and escapes `|` in the four model-supplied fields, and I confirmed by reverting it in a scratch copy that `TestRenderUserModelNeutralisesModelText` goes red. BR-5's second half landed too — `README.md:122` now reads "written by `--reflect` … its `## Corrections` section is yours", and I checked each of the five new exit-code cells against the code. The whole suite is green at `9bbfd4b`: `go vet ./...`, `go vet -tags conformance ./cmd/define/`, `go test ./...`, `go test -race ./cmd/define/`. What holds this back from SHIP is one thing I demonstrated rather than read: **the assertion written to pin BR-12's serious half cannot fire.** `usermodel_test.go:257` searches `got[:i]` — the region *before* the first marker — for a second marker, which is impossible by construction. I removed the sanitiser from the `## Level` section only, reintroducing exactly the "forged marker permanently freezes the file" defect, and the **entire `./cmd/define/` suite passed**, that test included. Alongside it: `EvidenceWords` are model-supplied and bypass `oneLine` entirely (verified: a citation of `hot\ndog` passes `store.Key`, survives `checkEvidence`, and renders a raw newline inside a table cell), and `atlas/define.md` was not swept for the round-2 rule — it still tells the reader the fuzz property "asserts the one thing that must hold", which this very window disproved. Seven Minors carry forward untouched across three rounds; none blocks.

## 1. Strengths

- **The BR-12 code fix is genuinely reachable, not decorative.** Reverting `oneLine` to `return s` reddens the test; reverting it to escape-pipes-only *also* reddens it (the pipe-count arm catches the moved trailing `|`). The behaviour is correct — my finding below is about the guard, not the fix.
- **The fix's conceptual framing is the right one** (`usermodel.go:187-189`): the learner's text is copied byte-for-byte *because they own it*; the model's text is neutralised *because it lands in a structure the file's integrity depends on*. Same file, two opposite rules, stated explicitly. That is the insight the next section (M2's weaknesses) needs to inherit.
- **BR-5 was swept at both sites this round**, which is the class-level answer the family asked for — the exit-code table's five new cells and the persistence block, each verified against the code.
- **D4's idempotency rests on something real.** `summariseLookups` sorts with a total tiebreak (`history_cmd.go:136-142`, "without it, map iteration makes the output flicker between runs"), so `TestReflectIsIdempotent` passes for a reason rather than by luck.
- **`workshop/lessons.md:965-990` generalises BR-2 into a mechanical checklist** ("comparing a file before/after → does a FAILED run also satisfy it?"). Ironically, N1 below is the *next* entry that checklist wants: "asserting an invariant → does the assertion search where the violation would be?"

## 2. Critical findings

None.

## 3. Important findings

**N1 — the forged-marker assertion searches a region where a forged marker cannot be** (`cmd/define/usermodel_test.go:253-259`). The test computes `i := firstMarkerOutsideAFence(got)` — the offset of the *first* marker — then asserts `firstMarkerOutsideAFence(got[:i] + padding) < 0`. A forged marker in `Rationale`/`Directive` renders *above* the real one, so it **is** `i`; `got[:i]` is by definition marker-free. Measured directly: with the sanitiser reverted, the rendered file contains **2** out-of-fence markers (offsets 336 and 385) and the shipped expression computes `j = -1`. The decisive experiment: removing `oneLine` from the Level section only — reinstating BR-12's serious half verbatim — leaves `go test ./cmd/define/` **fully green**, `TestRenderUserModelNeutralisesModelText` included. So the milestone's most serious defect is guarded only by the pipe-count arm, which a future edit touching the Level section will not trip. Fix: count markers instead of prefix-searching — walk `got`, or assert `firstMarkerOutsideAFence(got[i+len(correctionsMarker):]) < 0`. Verify by reverting the Level-section sanitiser and confirming red. **This is the 2nd finding in family `vacuous-verification`** (BR-2 was the first). Earlier rounds fixed instances. Do NOT fix only this assertion — state the rule: *an assertion that a violation is absent must be run over the region the violation would occupy; write the positive control (construct the violation, confirm the assertion fires) before believing it.* `mustReflect` was that rule applied to exit codes; this is the same rule applied to search windows, and `lessons.md`'s mechanical list is where it belongs.

**N2 — `EvidenceWords` are model-supplied and are the two render sites `oneLine` was not applied to** (`cmd/define/usermodel.go:57`, `:67` via `joinWords`, `:89-95`). Verified end-to-end: deck `{"hot dog"}`, model cites `"hot\ndog"` → `store.Key` collapses it to `hot dog` → `checkEvidence` keeps it **verbatim, with the newline** (`reflect.go:151-158` returns `w`, not the key) → `joinWords` renders `` `hot⏎dog` ``, splitting the `Read off:` line and breaking the table row mid-cell. `hot dog` is not a contrived example: it is this repo's own canonical multi-word headword (`README.md:194`). A deck word containing `|` breaks the table the same way. **This is the 2nd finding in family `model-text-unconstrained-by-format`.** Do NOT fix the two call sites — state the rule and enforce it structurally: *every model-supplied string that reaches the rendered file passes the sanitiser, and the sanitiser lives at one boundary rather than at each render site.* Put `oneLine` inside `joinWords`, or normalise citations to `store.Key(w)` in `checkEvidence`'s `supported` (which also makes what the file claims as evidence match what the deck holds); the sprinkle-at-call-sites shape is what let two of six sites be missed and is itself an ARCH-DRY smell. Related, same site: `oneLine` does not touch backticks, and `reflect_conformance_test.go:145-149` scans every backticked token above the marker as an evidence claim — a rationale that quotes `` `sailing` `` would fail the live check with "checkEvidence let an invention through" when nothing went wrong.

**N3 — `atlas/define.md` was not swept for the round-2 rule, and now asserts a property this window disproved** (`atlas/define.md:693-699`). Commit `9bbfd4b` touched `README.md`, code, tests, the issue and two ledgers — not the atlas. Two gaps: the neutralisation rule and its learner/model asymmetry (the one genuinely new architectural idea in this round) appear nowhere; and the splice paragraph still tells the reader the fuzz property "asserts **the one thing that must hold**", which is precisely the claim the issue's own Log now records as false ("a property that guards one direction of an invariant is not a property that guards the invariant"). A reader trusting the atlas would conclude the file's integrity is fully covered. **This is the 3rd finding in family `docs-enumeration-not-swept`** — round 3 stated that family's rule as "grep for the surface name across `README.md` **and `atlas/`** before closing", and this round swept README and not atlas. Do NOT patch the one paragraph: apply the stated rule as a closing step (grep `user-model`, `--reflect`, `Corrections` across `README.md` and `atlas/`), and while there, replace "the one thing that must hold" with the two-directions invariant.

## 4. Minor findings

- **BR-13 re-confirmed.** I moved `if *reflect` to just before `d = d.withStore(...)` and `go test ./cmd/define/` passed entirely; the resulting binary, run in a fresh directory, prints `define: no deck in this directory` and exits 1 for *every* `--reflect` invocation. One happy-path `run(ctx, []string{"--reflect"}, …)` test in a temp dir pins D5.
- **BR-7 (still open, now four sites).** `reflect.go:240` says "%d words in the deck" over deck∩log; `reflect.go:168` and `:190` pass one-element slices to `citedOrNothing`, whose `len == 0` guard can therefore never fire.
- **BR-9 (still open, and it grew).** The plan's Pure-entities table lacks `modelMeta` **and now `oneLine`** — both reported by the plan's own cited enumeration command — while its Revisions entry still says "Reconciled to empty before this commit."
- **BR-6 (still open).** `foldLookups`'s `now` is unread (`reflect.go:59`) while `:57-58` justifies it.
- **BR-8 (still open).** `spliceCorrections` discards a marker-less existing file in silence (`usermodel.go:104-108`).
- **BR-10 (still open).** No mode-count guard; `--reflect` with `--forget`/`--llm-check` silently honours one (`main.go:308`, `:347-355`).
- **BR-11 (still open).** `learner:` absent; `# N lookups, M reviews` shipped as `M questions`; the plan still says "Two departures".
- **BR-1 (still open).** Plan D1 (`plan.md:37-43`) still says "drops any claim citing a word the deck does not contain" against prune-then-drop-if-empty code.
- `checkEvidence` does not dedupe citations, so a model citing one word three times renders it three times in one cell.

## 5. Test coverage notes

- The three round-2/3 fixes (BR-2, BR-3, BR-4) all still hold at HEAD, and the suite is clean under `-race` and under `-tags conformance` vetting.
- The gap that matters is N1: the suite has a test *named* for the forged-marker defect that is blind to it. This is the second time on this issue that a test asserted something a no-op satisfies.
- Still uncovered: the four store-error `return 1` paths in `runReflect` (`Deck`, `Events`, `UserModel`, `SetUserModel` — `reflect.go:225-234`, `:291-307`); no test injects a failing store, so four error messages have never been rendered.
- `TestReflectRefusesATinyDeck`'s `len(fake.Requests()) != 0` remains the best-shaped assertion in the file — it pins that the floor is checked *before* the model is called.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag (minor, inside N2).** `foldLookups` → `summariseLookups` and `renderReflectPrompt` → `SchemaFor[learnerModel]` are both real reuse. The exception is the sanitiser: `oneLine` is called at four render sites and omitted at two, because the rule lives at the call sites rather than at one boundary. One place should decide "model text becomes safe here."
- **ARCH-PURE — pass.** `runReflect` is read → fold → ask → check → render → splice → write; every decision sits in a pure function and every pure entity is tested with no fake.
- **ARCH-PURPOSE — flag (N1, N2, N3).** All three are the same shape as the round-3 note: the fix takes the sites the finding enumerated. BR-12 named four fields → four fields were sanitised, and the two the finding did not enumerate were not. BR-5's rule named README *and atlas* → README was swept. And N1 means the purpose of the fix — "this cannot come back" — is not delivered even for the sites that were swept. Seven Minors have now survived three rounds untouched; at least three of them (BR-6 one-line deletion, BR-8 one line to `errOut`, BR-9 two table rows) cost less to fix than to carry.
- **ARCH-MOCK — pass.** `llmtest.Fake` is a wire-level `httptest` server, `store.YAML` is real in a `t.TempDir()`, and `reflect_conformance_test.go` skips rather than reddens when the seam is unreachable.
- For M2: `learnerModel` gains `Weaknesses`, which adds a *third* claim type with model-supplied free text and a new rendered section. Both the sanitiser boundary (N2) and `checkEvidence`'s per-claim usability predicate should be single functions before that lands, or M2 copies the render-safety hole into a section nobody has looked at yet.

## 7. Plan revision recommendations

A `## Revisions` entry in `workshop/plans/000017-user-model-plan.md` — none was written for boundary rounds 2 or 3, so this is now three rounds of drift:

- **Round 2/3 dispositions:** what BR-2…BR-5 changed, that half of BR-2's sketch was correctly rejected (asserting change-between-runs contradicts idempotency under a fixed clock), and that BR-12's fix introduced `oneLine`.
- **The enumeration still does not reconcile to empty.** Add `modelMeta` *and* `oneLine` to the Pure-entities table, and correct the entry that names eight symbols and asserts closure.
- **Departures three and four.** Task 3 says frontmatter "exactly as the issue's Spec shows"; `learner:` is omitted and `M reviews` shipped as `M questions`. Amend the Spec's shape or add the fields — either way "Two departures" is wrong.
- **D1's semantics** restated as prune-then-drop-if-empty, so the task bodies can cite it instead of paraphrasing it (BR-1).
- **Task 4's fuzz property covers one direction of the invariant.** Record that `FuzzSpliceCorrections` guards preservation-below and that regeneration-above needs its own guard — with N1 noting that the guard as written does not yet fire.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      plan.md:37-43 unchanged; D1 still says "drops any claim citing a word the deck does not contain".
  - id: BR-5
    disposition: addressed
    note: |
      Both sites swept — exit-code table's five new cells verified against the code, and README.md:122 now says "written by --reflect".
  - id: BR-6
    disposition: not-addressed
    note: |
      reflect.go:59 still takes `now` and never reads it; the doc comment at :57-58 still justifies it.
  - id: BR-7
    disposition: not-addressed
    note: |
      All four sites unchanged — reflect.go:240 counts deck-intersect-log as "in the deck", and :168/:190 pass one-element slices to citedOrNothing.
  - id: BR-8
    disposition: not-addressed
    note: |
      usermodel.go:104-108 unchanged; a marker-less existing file is still discarded without a word.
  - id: BR-9
    disposition: not-addressed
    note: |
      No modelMeta row, and 9bbfd4b added a ninth unrostered symbol (oneLine); the Revisions entry still claims the enumeration reconciled to empty.
  - id: BR-10
    disposition: not-addressed
    note: |
      main.go still has no mode-count guard; --reflect with --forget or --llm-check silently honours one.
  - id: BR-11
    disposition: not-addressed
    note: |
      Frontmatter still omits `learner:`, `M reviews` still ships as `M questions`, and the Revisions entry still says "Two departures".
  - id: BR-12
    disposition: addressed
    note: |
      Code fix verified by full revert (test reddens) and by escape-pipes-only revert (also reddens); but its serious half is unpinned — see the new vacuous-verification finding.
  - id: BR-13
    disposition: not-addressed
    note: |
      Re-verified: moving `if *reflect` before withStore leaves the whole suite green, and the resulting binary refuses every --reflect run with "no deck in this directory".
findings:
  - id: new
    severity: Important
    family: vacuous-verification
    title: |
      The forged-marker assertion searches only the region before the first marker, where a forged marker cannot be
    detail: |
      usermodel_test.go:253-259 computes i as the offset of the FIRST out-of-fence marker, then
      asserts no marker exists in got[:i] — marker-free by construction. Measured: with oneLine
      reverted the render holds 2 markers (offsets 336, 385) and the shipped expression yields
      j = -1. Decisive experiment: removing oneLine from the Level section only — reinstating
      BR-12's serious half exactly — leaves go test ./cmd/define/ fully green, that test
      included. So the milestone's worst defect is guarded only by the pipe-count arm. 2nd in
      this family after BR-2; the rule, not the site: an assertion that a violation is absent
      must run over the region the violation would occupy, and the positive control (construct
      the violation, confirm the assertion fires) is written before the assertion is believed.
      Add it to lessons.md's mechanical list beside the before/after-comparison entry.
  - id: new
    severity: Important
    family: model-text-unconstrained-by-format
    title: |
      EvidenceWords are model-supplied and are the two render sites oneLine was not applied to
    detail: |
      usermodel.go:57 and :67 render EvidenceWords through joinWords, which does not sanitise.
      Verified end to end: deck {"hot dog"}, model cites "hot\ndog" — store.Key collapses it so
      checkEvidence KEEPS it verbatim (reflect.go:151-158 returns w, not the key) — and
      joinWords emits a backticked token containing a newline, splitting the "Read off:" line
      and breaking the table row mid-cell. `hot dog` is this repo's own canonical multi-word
      headword. 2nd in this family: do not patch the two call sites — enforce the rule that
      every model-supplied string reaching the file passes one sanitiser at one boundary (put
      oneLine inside joinWords, or normalise citations to store.Key in checkEvidence's
      supported). The sprinkle-at-call-sites shape is what let two of six sites be missed
      (ARCH-DRY). Same site: oneLine does not strip backticks, and
      reflect_conformance_test.go:145-149 treats every backticked token above the marker as an
      evidence claim, so a rationale quoting a word would fail the live check spuriously.
  - id: new
    severity: Important
    family: docs-enumeration-not-swept
    title: |
      atlas/define.md was not swept for the round-2 rule and still asserts a property this window disproved
    detail: |
      Commit 9bbfd4b touched README, code, tests, the issue and two ledgers — not the atlas. The
      neutralisation rule and its learner/model asymmetry, the one new architectural idea this
      round, appear nowhere in atlas/define.md. Worse, atlas/define.md:697-699 still tells the
      reader the fuzz property "asserts the one thing that must hold", which the issue's own Log
      now records as false. 3rd in this family — round 3 stated the rule as "grep for the surface
      name across README.md AND atlas/ before closing", and this round swept README only. Apply
      the stated rule as a closing step rather than patching the paragraph, and replace "the one
      thing that must hold" with the two-directions invariant.
```
