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
