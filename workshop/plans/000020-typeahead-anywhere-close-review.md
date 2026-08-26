# Boundary Review — tools#20 (whole-issue close)

| field | value |
|-------|-------|
| issue | 20 — typeahead beyond the first word: complete deck words anywhere in the line |
| repo | tools |
| issue file | workshop/issues/000020-typeahead-anywhere.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0207e7ef24414534b308a2296fb148b1740814ee..3f884697463a5ee3900196513fcd83da63f7d5e8 |
| command | sdlc close --issue 20 |
| reviewer | claude |
| timestamp | 2026-08-26T12:16:33-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation is correct and the design is the right one — the segment expansion lives entirely in the history arm, the editor never learns about word boundaries, and the `candidates{recall, complete}` split is a real fix for the class PQ-1 named rather than a patch at the glue site. I mutation-tested the six axes the work claims to defend; five die loudly (floor in both directions, namespace order, mid-line `/` re-entry, `walk`→`complete`, each submission marker). **One survives: segment precedence.** Reversing `historyCompletions` to try the shortest segment first — with floor semantics preserved — leaves the entire suite green, including `TestWholeLineBeatsAnInnerSegment`, whose fixture (`hist("island", "hot dog and fries")` against the line `hot dog`) has no inner-segment match at all, so it never runs over a state where the two orderings differ. Plan row 7 claims that exact axis was mutation-checked and Done-when row 3 ticks it. That plus two unswept sites the Plan itself enumerated (four `h.Prefix(...)` test sites, the atlas editor-model paragraph) are what stand between this and SHIP. All are cheap; nothing here is a shipped correctness bug.

## 1. Strengths

- **The namespace boundary is genuinely pinned, and the second attempt was the honest one.** Mutating `completionsFor` to try history before commands reddens `TestACommandArgumentDoesNotDrawFromHistory`; injecting a per-segment `parseCommandLine` into the loop reddens `TestASlashSegmentMidLineDoesNotCompleteACommand` (both verified). The Log's admission that the first version of that test passed over its own bug, and the fixture change to `sevenfold`/`/history seven`, is the discipline the checklist asks for.
- **The recall/complete split is the strongest-pinned thing in the diff.** Routing `walk` to `c.complete` (editor.go:91-94) reddens five tests, including both new ones. `candidates` as a two-field struct rather than a special-cased glue path is the class-level fix.
- **`FuzzTrailingSegments` (complete_test.go:68) is the right tool on the right invariant.** 426k execs clean in 20s here; it asserts `head+text == line`, non-empty text, and strict shortening — three properties a table cannot cover.
- **ARCH-PURE holds cleanly.** `trailingSegments`, `matchesFor`, `historyCompletions`, `glue` are pure over an injected `History`; every test runs with `hist(...)` and no IO. `Suggestion`/`RenderLine`/`acceptSuggestion` are byte-for-byte untouched, which is the claim the atlas makes and it is true.
- **`typeKeys` wired to `candidatesFor` (editor_test.go:20)** — fixing the helper rather than the two symptoms is correct, and the lessons.md entry that came out of it is the most useful thing in this diff.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `TestWholeLineBeatsAnInnerSegment` does not discriminate; segment precedence is unpinned** (`cmd/define/complete_test.go:131-138`)

Verified by mutation. With `historyCompletions` rewritten to iterate segments shortest-first (floor semantics preserved via `i > 0` + `continue`), `go test ./cmd/define/` passes in full — the precedence rule stated in the Spec ("The first segment with a match wins"), in `historyCompletions`' doc comment, and in atlas/define.md:384-386 has zero coverage. The fixture's `"island"` entry is inert: for the line `hot dog` the only inner segment is `dog`, which matches nothing, so both orderings return the same answer.

Fix (verified red under the mutant, green on HEAD):
```go
h := hist("dog house", "hot dog and fries")   // was hist("island", ...)
// "hot dog": segment 0 -> " and fries"; reversed -> " house"
```
Also un-tick Done-when row 3 and Plan row 7's precedence clause until this bites — Plan row 7 currently asserts a mutation check that a mutant survives.

**I2 — atlas/define.md still states the pre-#20 editor model** (`atlas/define.md:177-183`)

> The editor is a pure state machine — `Apply(Editor, Key, matches) → (Editor, Action)` … the loop resolves matches once per keystroke and hands **the same slice** to both the state machine and the suggestion.

Both clauses are now false: the signature is `Apply(Editor, Key, candidates)`, and "the same slice to both" is precisely what #20 stopped doing. This is the atlas's *primary* description of the editor; the (good) new prose at :375-417 sits 200 lines below it, so a reader hits the stale model first. Plan row 8 scoped the sweep to the "matches the whole typed line" phrase and missed the model statement. Same paragraph family at :444-446 ("Apply still receives the pre-keystroke list") should say "pair".

**I3 — four test sites the Plan enumerated still bypass the production resolution** (`cmd/define/editor_test.go:163, 245, 249, 253`)

Plan row 1 says "Plus ~5 test sites (editor_test.go:15,157,239,243,247)". Site 15 (`typeKeys`) was fixed; the other four — all `Suggestion(e, h.Prefix(e.WalkBase()))` — were not. They are the exact anti-pattern this diff's own lessons.md entry names ("A test helper that skips the production resolution makes every test under it vacuous"), at four enumerable sibling sites in the same file, left in the same round. `TestSuggestionSuppressedMidLineAndWhenUnmatched`'s "no match" case (`zzz`) is the one that stings: under production resolution it would exercise the segment path and could catch "an unmatched line now suggests something glued"; as written it cannot. Fix: `Suggestion(e, completionsFor(e.WalkBase(), h, commands))` at all four. (ARCH-PURPOSE — instance vs class.)

**I4 — `?` is a bare literal at the three sites that must agree, while `\` got a shared constant, in the same commit** (`cmd/define/complete.go:71`, `repl.go:69`, `repl.go:140`)

The commit's own justification for `forceLiteral` is *"One constant because three places have to agree on it — the parser, `recallLine`'s canonical form, and the completion namespace."* `?` has identically three sites (parser at repl.go:69, `recallLine` at repl.go:140, `submissionMarkers` at complete.go:71) and this diff added the third one — as a literal. `submissionMarkers` is literally the class variable, and one of its two members is a constant. PQ-3 named the `\` instance; the class is "submission markers". Fix: `const forceAsk = "?"` beside `forceLiteral`, used at all three. (ARCH-DRY, ARCH-PURPOSE.)

## 4. Minor findings

- `repl.go:122-135` — the `const forceLiteral` declaration was inserted *between* `recallLine`'s doc comment and the function, so godoc now renders the whole BR-12 meaning-preservation rule as documentation for a one-character constant, and `recallLine` is undocumented. Move the const above the comment block.
- `editor.go:44` — `candidates`' doc says "Both are newest-first and deduped". `matchesFor` appends `?`- then `\`-unwrapped matches *after* all bare matches, so `complete` is grouped-by-marker, not newest-first. Deduped is still true. Either state the grouping or drop the claim.
- `complete.go:110-119` — `historyCompletions` returns on the first segment with *any* match, but `Suggestion` needs a match strictly longer than the line. A segment whose only match equals the segment itself silently blocks every shorter segment. Benign in the cases I could construct, but "first match" ≠ "first useful match".
- `README.md` — the (well-written) §"On a terminal" addition is **in the working tree but uncommitted**; it is not in the reviewed range `0207e7e..3f88469`. Commit it before closing, or the docs gate is satisfied only locally.
- `replraw.go:126` vs `:180` — `draw()` calls `completionsFor` while the key path calls `candidatesFor`. The pre-vs-post-keystroke split is deliberate and documented, but `candidatesFor(...).complete` would keep one resolution site. ARCH-DRY nit.
- `complete.go:112` — the floor counts trailing whitespace: `len([]rune("to ")) == 3` clears `minInnerSegment`, so a two-letter word plus a space is above a floor whose stated intent is word length. Requires a history entry starting `"to "` to matter, so harmless today.

## 5. Test coverage notes

- Mutants killed (all verified in a scratch copy): floor removed → `TestShortInnerSegmentsDoNotComplete`; floor applied globally → `TestGreyTailStillCompletesASingleWordAtTwoRunes` + `TestCompletionsFor`; namespace order swapped → `TestACommandArgumentDoesNotDrawFromHistory`; per-segment command re-entry → `TestASlashSegmentMidLineDoesNotCompleteACommand`; `walk`→`c.complete` → 5 tests; `?` dropped → `TestPastQuestionCompletesWhenRetypedBare`; `\` dropped → `TestForcedLiteralLookupCompletesAsThePlainWord`; empty trailing segment allowed → `TestTrailingSegments`; `glue` drops the head → 2 tests. Mutant surviving: **segment precedence** (I1).
- Done-when row 2 ("Tab / Right / End accept it … Tab asserted through a pty") rests entirely on `TestPTYSuggestionAndAcceptance`, which is behind `//go:build darwin && conformance` and **skipped in this environment** ("no pty available: operation not permitted"), so I could not verify the claim. There is no in-process test that accepts a *glued* candidate — `TestSuggestionAcceptedByRightEndAndTab` only covers segment 0. A ~10-line test (type `what is a obseq`, Tab, assert `e.String() == "what is a obsequious"` and `e.Cursor == len(e.Line)`) closes it in the default suite; I wrote and ran it, it passes and it goes red under the glue mutant.
- `matchesFor`'s cross-marker dedup (history holding both `obsequious` and `?obsequious`) is untested.
- Default suite is green (`go test ./cmd/define/` → ok, 63.6s); `go vet` clean; fuzz clean at 426k execs.

## 6. Architectural notes

- **ARCH-DRY — flag.** I4 (`?` literal vs `forceLiteral` constant) and the two-site resolution of `complete` (Minor). Everything else consolidates well: `prefixMatch` remains the one Prefix contract, `completionsFor` remains the one namespace decision.
- **ARCH-PURE — pass.** All four new functions are pure over an injected `History` whose `Prefix` is IO-free by contract; no test needs a mock to run them. `Apply` stays a state machine over plain data — `candidates` is a data struct, not a handle, so the "Apply never queries" property survives.
- **ARCH-PURPOSE — flag (partial).** The purpose (complete deck words mid-line, *including in free-form questions*) is fulfilled, and the `?`-unwrap reversal recorded in `## Revisions` is exactly the right call — leaving `?` alone would have shipped the easy subset. But two fixes landed at the instance rather than the class: the marker constant (I4) and the test-helper sweep (I3), where the diff's own lesson enumerates the class and four siblings were left standing.
- **ARCH-MOCK — pass.** No new external dependency. The terminal seam already has its pty conformance suite and #20 extended it rather than adding a stateless double; `History` is an injected interface with an in-memory implementation and the durable store behind the same seam, and production and test flow share that boundary.
- The plan carries no Core-concepts table (single-file work, Spec-as-plan). All named entities exist at their stated paths: `segment`, `trailingSegments`, `matchesFor`, `historyCompletions`, `glue`, `minInnerSegment` in `cmd/define/complete.go`; `candidates` in `editor.go`; `candidatesFor` in `command.go`; `forceLiteral` in `repl.go`. No table/code contradiction.
- For upcoming work: `matchesFor` is the natural place a `Found` filter would go if failed lookups prove noisy (the Spec anticipates it landing in `storeHistory.Load` instead — worth deciding once rather than twice), and the per-keystroke cost is now ~3×segments `Prefix` scans, resolved twice per key. Irrelevant at today's deck sizes; worth a note if the deck ever gets large.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/issues/000020-typeahead-anywhere.md` dated 2026-08-26 (close review), recording:

- **Plan row 7 amended** — the precedence axis was *not* successfully mutation-checked. A shortest-segment-first mutant survives the full suite because `TestWholeLineBeatsAnInnerSegment`'s fixture has no inner-segment match. Fixture changed to `hist("dog house", "hot dog and fries")`; Done-when row 3 re-ticked only after the mutant dies. (This is the same class as the lesson recorded this round — "a regression test needs data that tells the two implementations apart" — reappearing on a second axis of the same commit.)
- **Plan row 1 amended** — four of the five enumerated test sites (`editor_test.go:163,245,249,253`) were not updated; they still resolve via `h.Prefix(...)` instead of `completionsFor`.
- **Plan row 8 amended** — the atlas sweep was scoped to one phrase; `atlas/define.md:177-183` (and :444-446) still describe the pre-#20 `Apply` signature and the single shared slice.
- **New scope note** — a README §"On a terminal" paragraph was written for the mid-line suggestion; confirm it is committed inside the close window rather than left in the working tree.
