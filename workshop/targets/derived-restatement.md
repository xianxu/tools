---
slug: derived-restatement
kind: target
created: 2026-09-07
origin: tools#12 BR-8, BR-9, BR-11, BR-18 — one family, four rounds, never closed
---

# No artifact restates a fact the code owns

**The invariant.** A statement of fact about this system lives in exactly one
place. Every other appearance of it either DERIVES from that place — a guard
reads the code and checks the document — or is SWEPT at the boundary that
changed the fact.

**Why it needs defending.** It is the highest-recurrence family this repo has.
`#12` alone raised it four times across four review rounds, and the count of open
instances went UP, from five to eleven, with three of those added by the commits
that were fixing the other findings. Fixing instances does not shrink it, because
the instances are not the bug: the absence of an enumeration is.

Drift here is quiet and it is expensive twice over. A stale doc comment misleads
the next reader into building on a premise the code abandoned — `#12`'s Critical
was exactly that, a region formula justified by "both forms put the headword on
their first line" that a third form falsified. And a stale README is worse than
no README: the event log's block still read `kinds: looked-up, asked` while the
code wrote four kinds, and that block is the ONLY documentation a human reading
their own log has.

## Derive wherever the fact is machine-readable

Prefer this always. It costs one test and it never drifts again. What already
derives, as the pattern to copy:

| the fact | its one source | the guard |
|---|---|---|
| which forms exist | `Form()` in `play/*.go`, parsed | `TestEveryFormIsEnrolled` |
| what each form's prompt says | `livePrompt` / `gradedPromptFor` | `TestREADMEQuotesThePromptsTheLoopActuallyPrints` |
| what each key does | each form's `Keys()` | `TestREADMEKeyTableNamesEveryLiveKey` |
| which region kinds exist | `numRegionKinds` | `TestEveryRegionKindIsActionable`, `TestAtlasDescribesEveryRegionKind` |
| which event kinds exist | `store.EventKinds()` | `TestStoreLayoutDocsNameEveryEventKind` |
| what a region claims | the region's own coordinates | `TestAPromptRegionCoversTheTextItClaims` |
| which runtime dirs exist | `store.RuntimeDirs` | `TestPerWordDirsCoverEveryRuntimeDir` |
| why a form was not offered | `fallbackReasons` | `TestREADMENamesEveryFallbackReason` |

Two rules learned the hard way, both from guards in that table:

- **Scope the guard to the block.** A word occurs in the atlas for unrelated
  reasons; a docs guard matching the whole file passed with its section deleted.
- **Fail closed.** A guard asserting "declared ⊆ documented" is satisfied by
  deriving nothing. Assert the counts match too — `#12` BR-17 found a regex that
  silently under-derived and took two other guards down with it.

## Sweep what cannot derive

Prose is not machine-readable, so the rest is a checklist, run at **every**
`milestone-close` and `close` — not at an end-of-project sweep, which is where
this family grew to eleven. For each fact the window's diff changed, check:

- [ ] **the doc comment on every symbol the diff reshaped** — including the ones
      it did not edit. A comment on the *caller* of a changed function is the
      commonest instance, and `git diff` will not show it to you.
- [ ] **comments that cite a symbol or file** — a citation to something deleted
      or renamed (`play/choice.go` still cites `recall.go`, deleted with form 2.1).
- [ ] **tense** — "#38 WILL mark the option lines clickable" is true until someone
      mechanically rewrites it to "which marks", which is then a false statement
      of fact. Reword rather than re-tense.
- [ ] **`README.md`** — every block a user reads to interpret their own data.
- [ ] **`atlas/`** — surface, flow, terminology, and the premises it records.
- [ ] **the issue's `## Spec` and `## Done when`** — a row resolved elsewhere.
- [ ] **the plan** — a `## Revisions` entry, never an overwrite.
- [ ] **the project file** — scope events and ticks.

**And ask the derivation question first.** Every row above that turns out to be
machine-readable belongs in the table, not the checklist. The checklist should be
getting shorter.
