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
