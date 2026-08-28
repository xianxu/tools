# Boundary Review — tools#17 (whole-issue close)

| field | value |
|-------|-------|
| issue | 17 — learner model: batch analysis into a durable user-model.md |
| repo | tools |
| issue file | workshop/issues/000017-user-model.md |
| boundary | whole-issue close |
| milestone | — |
| window | 145a0e74438c3a94b0b2ae7995330ebc67f6e1de..c179efab165263ba0889fedb3567b8f22bed2f7c |
| command | sdlc close --issue 17 |
| reviewer | claude |
| timestamp | 2026-08-27T22:29:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M1 feature is genuinely shipped and sound: `foldLookups`/`checkEvidence`/`renderUserModel`/`spliceCorrections` are a real pure core behind a thin `runReflect` shell, the wire-level fake plus a live conformance check satisfy ARCH-MOCK properly, and the whole suite plus `go vet -tags conformance` is green (94s, no failures). Round 5's one Important finding (BR-17) was half-fixed and I verified the half that landed by mutation — deleting `sanitiseMeta`'s body now reddens `TestEveryUntrustedFieldIsNeutralised`. What blocks SHIP is that the *other* half of the same finding is still green under mutation (unsanitising three of `checkEvidence`'s five diagnostic arms leaves `./cmd/define/` entirely passing), and that the scope decision this close round exists to record — M2 descoped to #7 — was written into two issue files and into none of the four places `workshop/projects/define-learn.md` still calls it planned work. Both are cheap; neither is a correctness bug in shipped behaviour.

## 1. Strengths

- **`reflect_conformance_test.go:129-153` — the hard invariant is the right one.** "Every backticked word above the marker is a deck word" proves `checkEvidence` against a live model rather than a scripted answer, and the two scope corrections (above the marker only; drop the HTML comment) show it was actually run and debugged, not written and hoped for.
- **`usermodel.go:36-45` — sanitisation moved from N call sites to one pass over the struct.** This is the ARCH-DRY fix done at the right altitude: `renderUserModel` is where the file's structure is created, so a render site added later is safe by construction. `TestEveryUntrustedFieldIsNeutralised` (usermodel_test.go:331) gives it one row per field, and I confirmed the frontmatter row fires.
- **`reflect_run_test.go:264-276` — `mustReflect`.** The helper that fixes BR-2's class rather than its two instances, with the reason (a failed run writes nothing, so "unchanged" proves nothing) stated where the next person will read it.
- **`reflect.go:43-58` — `foldLookups` calls `summariseLookups` instead of restating it** (ARCH-DRY, cited in the code). D7 is a real decision with a real consequence: `--forget` drops evidence and keeps history.
- **The descope itself is well-argued.** Verifying the blocker rather than assuming it (`store/event.go:27` — I checked, `Correct bool` is accurate) turned "#6 unblocked M2" into "the schema still can't carry a kind", and carrying it to #7 as a Done-when row is the right artifact.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `workshop/projects/define-learn.md`: the descope is recorded in the issues and nowhere else.**
The close round's whole content is a scope change, and the portfolio view still contradicts it in four places: `:171-173` ("`#17 M2` deepens it once `#6` is producing misses"), `:194` (`- [ ] learner model — weakness taxonomy … [tools#17 M2]`), `:383-386` (a detail section whose `**status:** blocked — needs review events from [tools#6]` is now factually false — those events exist; the blocker is the event *schema*), and `:613`. This is the 3rd finding in family `docs-enumeration-not-swept`, so per the escalation rule the deliverable is the enumeration, not these four lines: run `grep -rn '#17\|tools#17' workshop/ atlas/ README.md` and reconcile every hit before the close is recorded. I ran it — 24 hits, of which the four above plus the `[tools#17 M2]: #tools-17-m2` anchor at `:563` need a scope-event edit; the rest (`:74`, `:77`, `atlas/define.md:715,766,771,1147`, `issues/000010`, `000012`, `000018`) are still true and need nothing. Note also that `sdlc close` auto-ticks referencing project rows: if it ticks `:194`, the portfolio will claim a descoped milestone was delivered.

**I-2 — M1 Done-when row 5 is ticked and nothing asserts it.**
`workshop/issues/000017-user-model.md:96-97` claims "Domain inference is checked against a held-out sample of deck words, not asserted." In `reflect_conformance_test.go` the held-out word is in the deck, in the prompt, *and* in the representation loop (`:95` ranges over `append(append([]string{}, c.words...), c.heldOut)`), and the only statement about it is a `t.Logf` at `:125` with the comment "Stated as an observation rather than a failure." Nothing is held out of anything. This is the 3rd finding in family `vacuous-verification` (after BR-2 and BR-14), so the deliverable is the rule: **a Done-when row is ticked only when a named test fails if the property is removed.** I ran that enumeration over M1's five rows — rows 1, 2 and 3 are pinned by named tests, row 4 I mutation-verified (setting `c.UserModel` aside in `ask.go:270` reddens `TestAskStreamsAnAnswerWithTheDirectoryAsContext`), row 5 is the only one unpinned. Cheapest real fix: count `seen` over `c.words` only, so a run that finds a cluster *only* by citing its held-out word fails.

## 4. Minor findings

- **M-1 — `- [x] M2 — weaknesses. DESCOPED at close, not delivered.`** (`000017-user-model.md:212`). A ticked box states delivery; this one's own text denies it, and the Done-when M2 rows five sections up are still `- [ ]`, so the same file answers "was M2 done" both ways. The gate has `--no-plan-check` for exactly this — leave the box unticked and put the descope in `--verified`.

## 5. Test coverage notes

- **The gap BR-13 names is the one this diff is most likely to ship again.** I re-verified on current code: moving `if *reflect` from `main.go:432` to just above `d = d.withStore(opt, stderr)` leaves `./cmd/define/` fully green, while the built binary would then refuse every `--reflect` run with "no deck in this directory". Every `runReflect` test calls the function directly; the only `run()`-level test is the usage-error path. One happy-path `run(ctx, []string{"--reflect"}, …)` in a temp dir closes it.
- **Mutation results this round**, all against the full `./cmd/define/` suite in a scratch copy: delete `sanitiseMeta` body → **RED** (good, BR-17's render half landed); unsanitise `cite(m.Level.Band)` at `reflect.go:188`, `cite(d.Name)` at `:210` and `:216` → **GREEN**; move the `--reflect` dispatch above `withStore` → **GREEN**; drop `c.UserModel` in the ask path → **RED** (good).
- `citeAll`'s per-word `oneLine` (`reflect.go:144-153`) has no injected fixture either — `TestDroppedClaimDiagnosticsCannotForgeALine` passes `evidence_words` with no newline in them.
- Live-behaviour repro for BR-7, since it is cheap to see: a directory with 13 word files and no events prints `define: 0 words in the deck; --reflect needs 12`. The number is `deck ∩ found-lookup log`, and the message calls it the deck.

## 6. Architectural notes

- **ARCH-DRY — flag.** `checkEvidence` neutralises at five hand-written `cite(...)` call sites (`reflect.go:188, 197, 210, 216, 221`), which is precisely the drifting list `renderUserModel` was refactored away from in the same milestone. Making `dropped` carry a `{subject, reason}` struct with one formatter that calls `oneLine` on the subject removes the class and makes one injection fixture cover all of them. Everything else passes: `foldLookups`→`summariseLookups`, one `noDeckMessage`, one `unavailableToReflect`.
- **ARCH-PURE — pass.** `foldLookups`, `checkEvidence`, `renderUserModel`, `spliceCorrections`, `firstMarkerOutsideAFence` are deterministic and tested with no fake; `runReflect` is read→fold→ask→check→render→splice→write and makes no decision of its own. The one residue is BR-6's unused `now` parameter, which advertises an injected clock the function does not have.
- **ARCH-PURPOSE — pass on the feature, flag on the finding.** The shadow-sweep is clean: `user-model.md` has exactly one consumer today and it derives (`ask.go:263-270`, mutation-confirmed). M2 is a separable extension with a real data blocker and a carried row, not the deferred point of the issue. The flag is on the *findings* axis — BR-17 was raised explicitly as "the deliverable is the rule, not these four sites" and was answered by fixing four sites.
- **ARCH-MOCK — pass, and it is the strongest part of the diff.** `llmtest.Fake` is a wire-level httptest server (never a stubbed client), `store.YAML` in `t.TempDir()` is the portable backend, production and test share the `deps.newLLM`/`deps.deck` boundary, and `reflect_conformance_test.go` is a real live check with a stated on-demand cadence that skips rather than fails on an unreachable seam.
- **For M2, whenever #7 delivers it:** `learnerModel` gains `Weaknesses`, and `sanitiseModel`'s comment (`usermodel.go:210-211`) says adding a field "makes this function fail to compile" — it does not, so a weakness claim's free text would reach the file unsanitised. That is the BR-17 residue with a name attached.

## 7. Plan revision recommendations

- **Core concepts, `## Revisions`.** The final entry says the enumeration was "Reconciled to empty before this commit." It is not: `git diff <base> <head> -- cmd/define/reflect.go cmd/define/usermodel.go cmd/define/reflectprompt.go | grep -E '^\+(func|type) '` still reports `modelMeta`, `cite`, `citeAll`, `oneLine`, `oneLineAll`, `sanitiseModel`, `sanitiseMeta` with no row, and the table's `citedOrNothing` was deleted in `692ec09`. Add the seven rows, drop the stale one, and correct the entry's claim.
- **D1.** Still reads "`checkEvidence` drops any claim citing a word the deck does not contain." The code prunes unsupported citations and drops only when nothing survives (`reflect.go:186-199`, `:219-225`) — which is what `TestCheckEvidenceDropsALevelClaimWithNoSupport` and the `"mixed"` case specify. Restate D1 with prune-then-drop-if-empty semantics so the task bodies can cite it instead of paraphrasing it.
- **The M1-shipped Revisions entry says "Two departures from the plan as written."** There are at least four: the frontmatter omits the Spec's `learner:` field, and `window:` renders `M questions` where the Spec shows `M reviews`. Either restore them or record them as departures.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      plan.md D1 unchanged — still "drops any claim citing a word the deck does not contain"; the code prunes and drops only when nothing survives.
  - id: BR-6
    disposition: not-addressed
    note: |
      reflect.go:59 still takes `now`; lines 60-88 never read it and the doc comment at :57-58 still justifies it as what makes the window a table row.
  - id: BR-7
    disposition: not-addressed
    note: |
      reflect.go:260 and :329 still call deck-intersect-log "words in the deck"; reproduced live — 13 word files and no events prints "define: 0 words in the deck".
  - id: BR-8
    disposition: not-addressed
    note: |
      usermodel.go:114-118 unchanged; an existing file with no out-of-fence marker is still returned over without a word to errOut.
  - id: BR-9
    disposition: not-addressed
    note: |
      Re-ran the plan's own cited command — seven symbols still have no row (modelMeta, cite, citeAll, oneLine, oneLineAll, sanitiseModel, sanitiseMeta) and citedOrNothing is stale.
  - id: BR-10
    disposition: not-addressed
    note: |
      Re-verified with the built binary — `--forget nonexistent --reflect` runs forget only, `--llm-check --reflect` runs llm-check only, neither mentioning the ignored flag; --play is now a fourth mode with no count guard.
  - id: BR-11
    disposition: not-addressed
    note: |
      usermodel.go:49-55 still omits `learner:` and still renders "M questions" where the Spec shows "M reviews"; the Revisions entry still says "Two departures".
  - id: BR-13
    disposition: not-addressed
    note: |
      Re-verified by mutation on current code — moving `if *reflect` above withStore leaves the whole ./cmd/define/ suite green while the binary would refuse every run.
  - id: BR-17
    disposition: not-addressed
    note: |
      Render half landed and is mutation-verified (deleting sanitiseMeta's body now reddens); the diagnostic half did not — unsanitising cite() at reflect.go:188, :210 and :216 leaves the full suite GREEN, and the one-formatter fix was not made.
findings:
  - id: new
    severity: Important
    family: docs-enumeration-not-swept
    title: |
      The M2 descope is recorded in two issue files and in none of the four places the project still calls it planned work
    detail: |
      This is the 3rd finding in family `docs-enumeration-not-swept`, so the deliverable is the
      enumeration, not the four lines. Rule: at a boundary that changes a fact, grep for every
      artifact that restates it and reconcile each hit before the verdict is recorded. Measured for
      this close: `grep -rn '#17|tools#17' workshop/ atlas/ README.md` returns 24 hits;
      workshop/projects/define-learn.md:171-173, :194, :383-386, :563 and :613 all still describe
      #17 M2 as planned or blocked-on-#6 work, and :385's "blocked — needs review events from
      tools#6" is now false. The remaining hits are still true. Note that sdlc close auto-ticks
      referencing project rows, so :194 risks being marked delivered.
  - id: new
    severity: Important
    family: vacuous-verification
    title: |
      M1 Done-when row 5 is ticked and the held-out property is a t.Logf, not an assertion
    detail: |
      This is the 3rd finding in family `vacuous-verification` after BR-2 and BR-14, so the
      deliverable is the rule: a Done-when row is ticked only when a named test FAILS if the
      property is removed. I ran that enumeration over M1's five rows — rows 1, 2, 3 are pinned by
      named tests, row 4 is pinned (mutation-verified: setting c.UserModel aside in ask.go:270
      reddens TestAskStreamsAnAnswerWithTheDirectoryAsContext), and row 5 is the only unpinned one.
      In reflect_conformance_test.go the held-out word is in the deck, in the prompt AND in the
      representation loop at :95, and the only statement about it is a t.Logf at :125. Cheapest
      real fix: count `seen` over c.words only.
  - id: new
    severity: Minor
    family: gate-satisfied-not-met
    title: |
      The M2 Plan row is ticked while its own text says "not delivered", and the M2 Done-when rows stay unticked
    detail: |
      000017-user-model.md:212 marks `- [x] M2 — weaknesses. DESCOPED at close, not delivered.`
      while :98-105 leave the M2 Done-when rows `- [ ]`, so one file answers "was M2 done" both
      ways and a `grep '\- \[ \]'` over the tracker no longer means what it did. The plan-unchecked
      gate has `--no-plan-check` for precisely this case — leave the box unticked and put the
      descope reasoning in `--verified`.
```

---

## Re-review — 2026-08-27T22:52:05-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 17 — learner model: batch analysis into a durable user-model.md |
| repo | tools |
| issue file | workshop/issues/000017-user-model.md |
| boundary | whole-issue close |
| milestone | — |
| window | 145a0e74438c3a94b0b2ae7995330ebc67f6e1de..1ecf05ac2fc9672e80307047424de7414e8ab15e |
| command | sdlc close --issue 17 |
| reviewer | claude |
| timestamp | 2026-08-27T22:52:05-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The three Important findings this round (BR-17, BR-18, BR-19) are genuinely disposed and I mutation-verified each: unsanitising `dropClaim.String` reddens all five new positive-control rows, the `#17` enumeration re-run reconciles every hit, and the conformance `seen` loop now excludes the held-out word so the row it ticks is load-bearing. `go test ./...` and `go vet ./...` are green in the real tree. Two things block a clean SHIP, both cheap: BR-17's frontmatter half was measured wrong — I confirmed at the parent commit (`c179efa`) that deleting `sanitiseMeta`'s body *already* reddened `TestEveryUntrustedFieldIsNeutralised/the frontmatter's model name`, so the round's Log, commit message and new test comment all assert a suite state that was false; and the `dropClaim` refactor silently changed a rendered diagnostic (`level C1: cites`, sentence trailing off) with no test covering the empty-citation shape. Eight prior Minor findings remain open and unfixed across five rounds — none blocking, but they are now the standing residue on this issue.

## 1. Strengths

- **`dropClaim` is the right shape, not a patch** (`cmd/define/reflect.go:166-180`). Five inline message constructions collapse into one formatter whose `String()` is the only path to text. I mutation-verified it: replacing `cite(d.Subject)` with `d.Subject` reddens all five subtests of `TestEveryDropDiagnosticNeutralisesItsSubject` plus `TestDroppedClaimDiagnosticsCannotForgeALine`. This is the list-that-drifts problem genuinely removed by construction (ARCH-DRY).
- **The positive controls are real, not decorative** (`cmd/define/reflect_run_test.go:331-374`). Every arm gets its own row with injection text that does not satisfy the assertion, and the "neutralised, not discarded" half is asserted separately from the "no newline" half.
- **BR-18's enumeration was actually run, not recalled.** I re-ran `grep -rnE '#17|tools#17' workshop/ atlas/ README.md`: all six project-file sites reconcile, and the mvp checkbox at `workshop/projects/define-learn.md:193` no longer contains the `[tools#17 M2]` link `sdlc close` auto-ticks — the specific auto-tick hazard the finding named is closed.
- **BR-19's fix is the cheapest correct one** (`cmd/define/reflect_conformance_test.go:106-117`). Counting `seen` over `c.words` alone makes the held-out design load-bearing; the `t.Logf` at :134 is now an observation *beside* an assertion rather than instead of one.
- **The descope was verified, not assumed.** `store/event.go`'s `Correct bool` is a real obstacle, and `workshop/issues/000007-vocab-form-meaning.md:32-49,54-55` carries the prerequisite as a Done-when row with the reasoning intact (ARCH-PURPOSE: the deferred part is separable, and it landed somewhere).

## 2. Critical findings

None.

## 3. Important findings

**N1 — BR-17's frontmatter premise was false, and three artifacts now record the false measurement.** `cmd/define/reflect_run_test.go:369-374`. I extracted `c179efa` to a scratch tree and deleted `sanitiseMeta`'s body: `TestEveryUntrustedFieldIsNeutralised/the frontmatter's model name` (added in round 4, `cmd/define/usermodel_test.go:367-371`) fails there. The suite was *not* green. The same mutation on the three diagnostic arms *was* green at the parent, so BR-17's diagnostics half was real — only the frontmatter half was wrong, and it is the half asserted verbatim in `workshop/issues/000017-user-model.md` ("deleting `sanitiseMeta`'s body … left the whole `./cmd/define/` suite green", "Measured: … each **now** redden 2"), in the commit body, and in the new test's own doc comment. Fix: correct the Log/commit note to say the frontmatter site was already pinned, and decide whether `TestModelNameCannotBreakTheFrontmatter` stays (its line-count assertion is a different shape, so keeping it is defensible — the false justification is not).

**N2 — the `dropClaim` refactor changed a rendered diagnostic and nothing noticed.** `cmd/define/reflect.go:174-180`. When a claim carries an empty `evidence_words` array, `Cited` is empty, so `String()` stops at `Reason` and stderr prints `define: dropped level C1: cites` — a sentence that trails off. Before the refactor it read `… cites nothing, none of which is in the deck`. `citeAll`'s `len(words) == 0 → "nothing"` branch (`:144-147`) is now unreachable: its only caller guards on `len(d.Cited) > 0`, so the comment above it ("a claim that cited NOTHING is a different failure … the message must tell them apart") describes dead code. Verified by scratch test on both the level and domain arms.

## 4. Minor findings

- **N3 — the durable plan still claims 40 outstanding steps.** `workshop/plans/000017-user-model-plan.md` has 40 `- [ ]` step boxes and zero `- [x]`, while its own Revisions entry says "Tasks 1–8 done." AGENTS.md §1 makes this file the record of truth.
- (Eight prior Minor findings re-raised below as `not-addressed`; see the dispose block.)

## 5. Test coverage notes

Coverage on the pure entities is strong — 22 named tests plus a fuzz property over `spliceCorrections`, all running without IO. Two gaps this window:

- No test constructs a claim with an **empty** `evidence_words` array. Every `checkEvidence` test supplies at least one word, which is why N2 shipped unseen. A table row per `dropClaim` shape (empty subject / empty citations / both) asserting the full rendered line would close it and make `citeAll`'s empty branch reachable again.
- `--reflect`'s happy path is never driven through `run(ctx, []string{"--reflect"}, …)`. I re-confirmed BR-13 by moving the dispatch above `d = d.withStore(opt, stderr)` in a scratch copy: `go test ./cmd/define/` stays green apart from the repo-guard tests, which fail only because a `git archive` tree is not a repo. One happy-path `run` test in a `t.TempDir()` pins D5.

## 6. Architectural notes

- **ARCH-DRY — pass, with one residue.** The `dropClaim` consolidation is the principle applied correctly. The residue is `citeAll`'s now-dead branch (N2) and the partial overlap between `TestModelNameCannotBreakTheFrontmatter` and the existing per-field row (N1).
- **ARCH-PURE — pass.** `foldLookups`, `checkEvidence`, `dropClaim.String`, `renderUserModel`, `spliceCorrections` are all deterministic and tested without fakes; `runReflect` is the thin shell. The one blemish is BR-6: `foldLookups`'s `now` parameter is read at zero sites while the doc comment at `reflect.go:57-58` cites it as what makes purity observable.
- **ARCH-PURPOSE — pass on the descope, flag on the enumeration.** M2 is deferred for a verified data-shape reason and lands in `#7`'s Done-when plus an unticked project row, so it is tracked. But the round answered BR-18 by enumerating *textual mentions of `#17`* — which cannot see a state restatement that never names the issue, which is exactly why the plan's 40 unticked boxes (N3) survived the sweep. The class is "artifacts that restate this boundary's state," not "artifacts that mention this issue id."
- **ARCH-MOCK — pass.** `llmtest.Fake` is wire-level, `store.YAML` runs in a `t.TempDir()`, and the live check skips through `conformance.SkipOrFail` rather than reddening on a flat network. Production and test flows share the `runReflect` boundary.

## 7. Plan revision recommendations

The plan has had no `## Revisions` entry since 2026-08-25, across three boundary rounds. It needs:

- **`### 2026-08-27 — rounds 3–5, and the Core concepts reconciliation that was claimed and not run.`** The table's `checkEvidence / citedOrNothing` row names a symbol that does not exist (renamed to `cite`/`citeAll`), and eleven symbols have no row: `modelMeta`, `cite`, `citeAll`, `dropClaim`, `dropClaim.String`, `oneLine`, `sanitiseModel`, `sanitiseMeta`, `oneLineAll`, `reflectTaskName`, `reflectSystem`. `dropClaim` and `String` were added *by this round*. The existing entry's "Reconciled to empty before this commit" is false against the tree.
- **D1's sentence** rewritten to the prune-then-drop-if-empty semantics the code and `TestCheckEvidenceDropsClaimsTheDeckCannotSupport` actually implement, so Task 2's body can cite D1 instead of paraphrasing it (BR-1).
- **A note on the frontmatter departure:** the rendered frontmatter omits the Spec's `learner:` field, which the existing entry's "Two departures from the plan as written" does not cover (BR-11).
- **Tick the 40 step boxes** or record why they stay open (N3).

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      D1 still reads "drops any claim citing a word the deck does not contain"; the mixed case prunes and keeps.
  - id: BR-6
    disposition: not-addressed
    note: |
      reflect.go:59-89 still never reads `now`; the purity comment at :57-58 still cites it.
  - id: BR-7
    disposition: not-addressed
    note: |
      Both halves stand, and this round's refactor made the second worse — see new finding N2.
  - id: BR-8
    disposition: not-addressed
    note: |
      usermodel.go:115-118 still returns `generated` whole with nothing written to errOut.
  - id: BR-9
    disposition: not-addressed
    note: |
      Still no modelMeta row; the table also names `citedOrNothing`, which does not exist, and this round added dropClaim + String with no row.
  - id: BR-10
    disposition: not-addressed
    note: |
      Re-verified by running the binary: --forget X --reflect, --llm-check --reflect and --play --reflect each honour one mode silently. Three pairs now, not two.
  - id: BR-11
    disposition: not-addressed
    note: |
      usermodel.go:49-55 still emits four keys; no `learner:`, and no Revisions entry saying so.
  - id: BR-13
    disposition: not-addressed
    note: |
      Re-verified at HEAD: moving the dispatch above withStore leaves ./cmd/define/ green apart from the scratch-tree repo guards.
  - id: BR-17
    disposition: addressed
    note: |
      Diagnostics half verified — unsanitising dropClaim.String reddens all five arms. The frontmatter half's premise was false; see N1.
  - id: BR-18
    disposition: addressed
    note: |
      Re-ran the grep: all six project sites reconcile and the auto-tick checkbox no longer matches [tools#17 M2].
  - id: BR-19
    disposition: addressed
    note: |
      seen is counted over c.words only, so the domain must be inferable from the words that were not withheld.
  - id: BR-20
    disposition: not-addressed
    note: |
      000017-user-model.md:212 is still `- [x]` on work its own text calls not delivered; the new Done-when header clarifies prose but not the grep.
findings:
  - id: new
    severity: Important
    family: comment-outruns-code
    title: |
      BR-17's frontmatter half was measured wrong, and the Log, commit body and new test comment all record the false measurement
    detail: |
      This is the 2nd finding in family `comment-outruns-code`. Do not fix the instance — state the rule.
      Measured at the parent commit c179efa in a scratch tree: deleting sanitiseMeta's body already
      reddened TestEveryUntrustedFieldIsNeutralised/"the frontmatter's model name" (usermodel_test.go:367),
      added in round 4. The suite was not green. The same mutation on the three diagnostic arms WAS green,
      so the diagnostics half of BR-17 was real and the frontmatter half was not. The false half is now
      asserted in three places: the issue Log ("deleting sanitiseMeta's body ... left the whole ./cmd/define/
      suite green" and "each NOW redden 2"), the commit body, and reflect_run_test.go:369-374's own comment.
      The rule the class needs: a claim about what the suite does or does not cover is a MEASUREMENT — run
      the mutation against the tree the finding names before writing the fix, the comment, or the Log line
      that asserts it. Accepting a finding's premise on trust is the same error as accepting a fix on trust,
      and it costs a duplicate test plus a false entry in the ledger the process runs on.
  - id: new
    severity: Important
    family: decision-unpinned-by-test
    title: |
      The dropClaim refactor truncated the cited-nothing diagnostic and left citeAll's empty branch dead, with no test over the shape
    detail: |
      This is the 2nd finding in family `decision-unpinned-by-test`. Do not fix only the site — state the rule.
      reflect.go:174-180 appends the ", none of which is in the deck" clause only when len(d.Cited) > 0, so a
      claim whose evidence_words array is empty renders as "define: dropped level C1: cites" — a sentence that
      stops mid-clause. Verified by scratch test on both the level arm (checkEvidence default branch) and the
      domain arm. Before the refactor the same input produced "... cites nothing, none of which is in the deck".
      citeAll's len(words)==0 -> "nothing" branch (reflect.go:144-147) is now unreachable, so its comment —
      "a claim that cited NOTHING is a different failure from one that cited words we do not have, and the
      message must tell them apart" — documents dead code. No test in reflect_test.go or reflect_run_test.go
      supplies an empty evidence_words array; every case carries at least one word, which is why a refactor
      landed under a fix-the-rule banner and silently changed observable output. The rule: a formatter that
      branches on input shape gets one table row per shape it can receive, empty included, asserting the whole
      rendered line — otherwise consolidating call sites into one formatter trades five remembering-sites for
      one unpinned one.
  - id: new
    severity: Minor
    family: docs-enumeration-not-swept
    title: |
      The durable plan still shows 40 unticked step boxes while its own Revisions entry says Tasks 1-8 are done
    detail: |
      This is the 4th finding in family `docs-enumeration-not-swept`. Do not fix the instance — the rule is what
      is missing. Measured: workshop/plans/000017-user-model-plan.md has 40 `- [ ]` and 0 `- [x]`, and AGENTS.md
      section 1 makes that file the record of truth, not the ephemeral harness plan. Why the round-5 sweep could
      not see it: BR-18's enumeration was `grep -rnE '#17|tools#17'`, which finds artifacts that MENTION the
      issue id. A checkbox restates the issue's STATE without naming it, so no grep over the id can reach it.
      The rule the class needs is therefore not "grep the id" but "enumerate the artifacts that restate this
      boundary's state — issue Plan, issue Done-when, durable plan steps, project rows, atlas prose — and
      reconcile each before the verdict is recorded." Same family fired on the project file this round; the
      enumeration was written for one axis and the other axis is where the drift was.
```

---

## Re-review — 2026-08-27T23:11:57-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 17 — learner model: batch analysis into a durable user-model.md |
| repo | tools |
| issue file | workshop/issues/000017-user-model.md |
| boundary | whole-issue close |
| milestone | — |
| window | 145a0e74438c3a94b0b2ae7995330ebc67f6e1de..34c47cbc2405395d594f22fda60b5194ead6bf24 |
| command | sdlc close --issue 17 |
| reviewer | claude |
| timestamp | 2026-08-27T23:11:57-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 7's two Important findings are genuinely and verifiably fixed — I reverted each in a scratch worktree and confirmed the named tests go red, which is the bar the claimed-fixes protocol sets. The shipped code is correct: `go test ./...` and `go vet ./...` are green, `runReflect` is a thin IO shell over pure `foldLookups`/`checkEvidence`/`renderUserModel`/`spliceCorrections`, and the atlas + README both carry the `--reflect` surface including its exit codes. What keeps this off SHIP is not the code: it is that the round-6 commit answered BR-21 by correcting the three artifacts the *finding* enumerated and did not enumerate the rest — `reflect.go:161` still asserts a whole-suite measurement that I measured false at the exact commit it names. That is the third instance of `comment-outruns-code`, landing inside the commit that wrote the rule into `lessons.md`. Ten Minor findings (BR-1, 6, 7, 8, 9, 10, 11, 13, 20, 23) are now carried un-disposed across four rounds; none blocks the gate, but seven of them are one class — a durable artifact restating a fact the tree contradicts — and that class is what BR-23 asked for a rule about.

## 1. Strengths

- **`TestDropDiagnosticRendersEveryShape`** (`cmd/define/reflect_run_test.go:383`) is exactly the right shape for BR-22: one row per input shape the formatter can receive (nil / empty / populated / no-citation), asserting the *whole* rendered line rather than a substring. Measured: re-introducing the `len(d.Cited)` branch reddens two named rows with a diff that reads as the bug (`"level C1: cites"` vs the full sentence).
- **The `Cites` flag** (`cmd/define/reflect.go:171-183`) is the right fix rather than the cheap one — it makes "rejected *for* its citations" a distinct state from "cited nothing", which is the distinction `citeAll`'s empty branch existed to serve and which `len()` collapsed.
- **BR-21's self-correction is honest and correct.** I verified it independently: deleting `sanitiseMeta`'s body at HEAD reddens `TestEveryUntrustedFieldIsNeutralised/"the frontmatter's model name"` (`usermodel_test.go:331`), so the removed test really was a duplicate and the reviewer's premise really was half wrong. Retracting a fix you already shipped, in the ledger, is the expensive direction.
- **`sanitiseModel`/`sanitiseMeta` as one pass over the struct** (`usermodel.go:213-243`) is the structural answer to the list-that-drifts failure — a new render site is safe by construction, not by remembering. `ARCH-DRY` pass.
- **`foldLookups` calls `summariseLookups`** rather than re-deriving the per-word fold (`reflect.go:59-89`). `ARCH-DRY` pass, and the plan's D6 states why.

## 2. Critical findings

None.

## 3. Important findings

**`cmd/define/reflect.go:161` — the `dropClaim` doc comment asserts a whole-suite measurement that is false at the commit it cites.**

The comment reads: *"deleting the sanitiser left the whole suite green, because no test constructed a subject that carried a newline at three of the five arms."* The subordinate clause contradicts the main clause — if three of five arms were uncovered, two were covered — and I measured the main clause directly. In a scratch worktree at `c179efa` (the tree BR-17's premise named), removing `oneLine` from both `cite` and `citeAll`:

```
--- FAIL: TestDroppedClaimDiagnosticsCannotForgeALine
    reflect_run_test.go:305: stderr has 5 lines, want 3
    reflect_run_test.go:310: a diagnostic line does not start with the program name: "FORGED-LEVEL: ..."
```

The project's own round-5 Log records the accurate per-arm version (*"unsanitising the share-out-of-range arm reddened 0 tests"*). The comment overstated it into a suite claim.

Fix sketch: this is not a sentence edit. Round 6 disposed BR-21 by correcting the three artifacts BR-21's text listed and stopped there; the rule it wrote into `lessons.md:761` implies an enumeration nobody ran. Run it — `grep -rniE 'suite green|left the suite|reddens? [0-9]|no test .*(constructed|supplied)' cmd/ internal/ atlas/ workshop/` returns ~25 sites — and either measure each claim or rewrite it to the scope actually measured. Two sites are in this window: `reflect.go:161` (false as written) and `reflect_run_test.go:322-324` (defensible — "them" scopes to the three unpinned sanitisers, but it sits one line from the false one).

## 4. Minor findings

*(No new Minor findings; the ten carried ones are dispositioned in the block below rather than restated here.)*

## 5. Test coverage notes

- Suite green at HEAD: `go test ./cmd/define/... ./internal/...` all `ok` (`cmd/define` 94.4s), `go vet ./...` silent.
- Both round-7 fixes are mutation-verified, not read-verified. `ARCH-MOCK` pass: `reflectRig` drives the real YAML store in a temp dir behind the wire-level `llmtest.Fake`, and `reflect_conformance_test.go` is the live drift check.
- The gap BR-13 names is still real and I re-measured its shape: the only test routing `--reflect` through `run()` is `TestReflectWithAWordIsAUsageError` (`reflect_run_test.go:207`), the usage-error path. Every happy-path test calls `runReflect` directly, so D5's "dispatched after `withStore`" decision is pinned by nothing.

## 6. Architectural notes for upcoming work

- `ARCH-DRY` — pass. `dropClaim.String` is the consolidation the round-5 refactor promised, and it now has the per-branch rows that make consolidation safe.
- `ARCH-PURE` — pass, with one artifact: `foldLookups`'s unused `now` parameter (BR-6) is purity theatre — a parameter added to *look* injected. Either clamp `To` to it or drop it; a pure signature that lies is worse than a smaller one.
- `ARCH-PURPOSE` — flag, and it is the shape of this whole gate. M1's purpose is delivered and consumed (`ask.go:268`). But findings on this issue keep being answered at the site they name: BR-17 → four sites, not the class; BR-21 → three artifacts, not the enumeration. `dropClaim.Reason` being documented as *"unused when `Cites` is set"* is the same axis in the type system — a struct with mutually exclusive fields where a sum type would make the invalid state unrepresentable. Worth reaching for on the next formatter.
- `ARCH-MOCK` — pass.

## 7. Plan revision recommendations

`workshop/plans/000017-user-model-plan.md` needs one `## Revisions` entry that closes three still-open contradictions at once, rather than three entries:

- **Core concepts** — delete the `citedOrNothing` row (the symbol was removed in `692ec09`); add rows for `modelMeta`, `dropClaim`, `dropClaim.String`, `cite`, `citeAll`, `oneLine`, `oneLineAll`, `sanitiseModel`, `sanitiseMeta`. Correct the entry that asserts *"Reconciled to empty before this commit"* — re-running the plan's own cited command still reports the gap, four rounds after it was first raised.
- **D1** — restate it as the prune-then-drop-if-empty semantics `checkEvidence` actually implements (`reflect.go:224-231` keeps a partially-supported claim), so `D1` is the single statement of its fact and the task bodies can cite it.
- **Task checkboxes** — 41 `- [ ]` and 0 `- [x]` against a Revisions entry saying *"Tasks 1–8 done"*. Tick them, or state in the entry that the boxes are not maintained and the Revisions section is the record.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      D1 still reads "drops any claim citing a word the deck does not contain"; checkEvidence prunes and keeps.
  - id: BR-6
    disposition: not-addressed
    note: |
      reflect.go:59-89 still never reads `now`; the purity comment at :57-58 still justifies it.
  - id: BR-7
    disposition: not-addressed
    note: |
      Half 1 stands — reflect.go:302 prints len(ev.Words) as "words in the deck", and YAML.Events skips a warned day file, so the number shrinks silently. Half 2 is now unreachable (the empty-band arm fires first).
  - id: BR-8
    disposition: not-addressed
    note: |
      usermodel.go:120-122 still returns `generated` whole with nothing written to errOut.
  - id: BR-9
    disposition: not-addressed
    note: |
      Table still names citedOrNothing (deleted in 692ec09); modelMeta, cite, citeAll, dropClaim, String, oneLine, oneLineAll, sanitiseModel, sanitiseMeta have no row; the Revisions entry still claims "Reconciled to empty".
  - id: BR-10
    disposition: not-addressed
    note: |
      No mode-count guard. main.go:372 dispatches --llm-check before the switch, :419 --forget and :430 --play before `if *reflect` at :432; each pair silently honours one mode.
  - id: BR-11
    disposition: not-addressed
    note: |
      usermodel.go:49-55 still emits four keys; no `learner:`, and no Revisions entry recording the departure.
  - id: BR-13
    disposition: not-addressed
    note: |
      The only run()-level --reflect test is TestReflectWithAWordIsAUsageError (reflect_run_test.go:207); every happy-path test calls runReflect directly, so D5's dispatch site is unpinned.
  - id: BR-20
    disposition: not-addressed
    note: |
      000017-user-model.md:212 is still `- [x]` on work its own text calls "not delivered"; the Done-when header clarifies prose but not the grep.
  - id: BR-21
    disposition: addressed
    note: |
      Verified by mutation at HEAD — deleting sanitiseMeta's body reddens TestEveryUntrustedFieldIsNeutralised/"the frontmatter's model name", so the duplicate test is correctly removed and the Log correction is right. See the new finding for the residue.
  - id: BR-22
    disposition: addressed
    note: |
      Verified by reverting String() to the len(Cited) branch in a scratch worktree — TestDropDiagnosticRendersEveryShape's "cited NOTHING" and "cited an empty list" rows both go red with the truncated sentence.
  - id: BR-23
    disposition: not-addressed
    note: |
      Re-measured: workshop/plans/000017-user-model-plan.md has 41 unticked boxes and 0 ticked, against a Revisions entry saying "Tasks 1-8 done".
findings:
  - id: new
    severity: Important
    family: comment-outruns-code
    title: |
      reflect.go's dropClaim comment still asserts "deleting the sanitiser left the whole suite green" — measured false at the commit it names
    detail: |
      This is the 3rd finding in family `comment-outruns-code` (BR-6, BR-21, this).
      Earlier rounds fixed instances. Do NOT fix this instance — state the rule that
      covers all of them, and fix that.
      Measured, in a scratch worktree at c179efa (the tree BR-17's premise named and the
      tree this comment describes): removing oneLine from cite() and citeAll() reddens
      TestDroppedClaimDiagnosticsCannotForgeALine ("stderr has 5 lines, want 3"). The suite
      was NOT green. reflect.go:159-162 says it was, and its own subordinate clause
      contradicts it — three of five arms uncovered means two were covered. The issue Log's
      round-5 entry states the accurate per-arm version ("unsanitising the share-out-of-range
      arm reddened 0 tests"); the code comment inflated it into a suite claim.
      Why this is the class and not the site: round 6 disposed BR-21 by correcting the three
      artifacts BR-21's own text listed (Log, commit body, test comment) and wrote the rule
      into lessons.md:761 — then did not run the enumeration that rule implies. This
      sentence was in the diff of the same commit. The enumeration exists and is cheap:
      grep -rniE 'suite green|left the suite|reddens? [0-9]|no test .*(constructed|supplied)'
      over cmd/ internal/ atlas/ workshop/ returns ~25 sites, two of them in this window
      (reflect.go:161 false as written, reflect_run_test.go:322-324 defensible but adjacent).
      The deliverable is that sweep plus the rule stated as an enumeration, not a reworded
      sentence.
```
