---
id: 000009
status: working
deps: ["tools#3"]
github_issue:
created: 2026-08-20
updated: 2026-08-26
estimate_hours: 8.22
started: 2026-08-26T19:29:35-07:00
---

# news seam: Google News RSS client with a stateful fake

## Problem

Generated questions are better when the sentence is real and current. That
needs a news source that can be fetched without a key and without scraping.

## Spec

Google News **RSS**, behind a seam.

- `news.google.com/rss/search?q="<word>"&hl=en-US&gl=US&ceid=US:en`.
- **Measured before planning:** 41–100 items per word; headlines containing the
  word range 12 (`defenestrate`) to 99 (`ephemeral`). Structured XML, key-free.
- **The SERP is not an option, and this was measured, not assumed:** an earlier
  probe of `google.com/search` returned a 91 KB JS shell (`enablejs`) with zero
  usable content. Scraping it needs a headless browser.
- Terms note: the feed is licensed for personal, non-commercial feed-reader use.
  A personal vocabulary tool fits; do not redistribute the content.
- Results **cached in the store**, so review works offline and a session does not
  hit the network per question.
- Stateful fake serving canned RSS; live conformance check on the feed shape,
  on-demand like the other seams.

## Done when

- [x] Fetches, parses and caches; a second request for the same word is served
      from cache — asserted on the fake's fetch COUNTER, not inferred from output.
- [x] A failed fetch is NOT cached, so a network blip does not become a permanent
      empty answer for that word.
- [x] The parser survives a malformed feed without taking down the session, and
      never returns an item it did not find in the input.
- [x] Only headlines that genuinely contain the word become usages — the feed
      returns many that do not (measured 12-99 matching out of 41-100).
- [x] NOAD's own example sentences are available through the same seam and the
      same `Usage` shape, tagged by source, with no network.
- [x] Live conformance asserts the feed shape and a coverage FLOOR.
- [x] `Mem` and `YAML` both satisfy the new store contract via `storetest.Suite`.

## Plan

Durable plan: `workshop/plans/000009-vocab-news-plan.md` (three milestones; each
`Mx` row is its own review boundary).

- [x] M1 — `parseRSS`, `Usage`, `containsWord`, and NOAD's examples as the second source
- [x] M2 — `Store.Usages`, the `NewsSource` seam, the caching wrapper, the stateful fake
- [x] M3 — wiring, `/usage` to see it, live conformance, docs

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

Derived after the plan cleared plan-quality (#187), one item per task in
`workshop/plans/000009-vocab-news-plan.md` plus the process work inside the
measured window. Design carries v2's ×0.2 spec-quality discount on every code
item — the plan pre-resolves the raw/derived split, the three cache outcomes, the
matcher delegation and the record shape, so what is left at design time is
reading. Implementation is v3.1's 40% of the v2 table. Familiarity 1.0: same
package, and `fetch.go`'s `httpAudioSource`/`cachingAudioSource` pair is the
literal shape `httpFeed`/`cachingFeed` copies.

`issue-spec` is `design=0.75`, which is the band's MIDPOINT × 1.5 — not "half the
band", which an earlier draft claimed and which would be 0.5. The derivation is
proportional: #21's block priced 1.50 for a 646-line plan; this one is 351 lines
against a Spec that was already written and already revised once (2026-08-22,
settling RSS-not-SERP and the second source), so roughly half of #21's authoring
work. `impl=0.08` is the unhalved 40%-of-mid, because ticking and committing a
plan costs the same whatever its length.

**`real-api-discovery` is included at its FLOOR (0.12), consciously rather than
by omission.** The v2 table carries a per-external-API discovery budget and this
issue integrates a live feed — capturing a real fixture, building the URL, handling
non-200, and a live conformance check against real Google output. Most of that
discovery is already SPENT: the Spec measured the feed before planning (41–100
items per word, 12–99 containing it) and measured the SERP alternative well enough
to rule it out. What remains is mechanical, so the floor rather than the mid.

**Review rounds are priced at #20's measured 0.5h — the only DIRECT measurement
this repo has** — as `design=0.20 impl=0.30`. An earlier draft of this block cut
that to 0.30h and claimed the basis was "#21's measurement". That overstated the
evidence, and the correction is worth recording because it is the same optimistic
drift I have been criticising elsewhere: #21 gives an upper BOUND (eleven rounds
inside a 6.0h total ⇒ rounds under ~0.55h), which refutes #16's 0.9h but does not
select 0.30 over 0.5. Halving the only measured value on the strength of a bound
that merely permits it is not a derivation. Eight rounds are budgeted: two per
boundary across M1, M2 and the close, plus the two plan rounds already spent.

Sensitivity, stated because this line dominates the estimate: at 0.5h/round the
eight rounds are 4.0h of an 8.22h total. Eleven rounds — #21's actual count —
would add ~1.5h.

**The two most recent rows disagree about direction, and my first attempt to
reconcile them did not survive the ledger.** tools#21 est 9.28 / actual 6.00
(1.55×, trusted) says I over-price; tools#16 est 6.49 / actual 17.63 (0.37×,
trusted) says I under-price. The reconciliation worth testing is the
estimate-quality judge's from #21: over-pricing in **design/process**,
under-pricing in **implementation**. But the placement argument I built for it was
wrong — I wrote that this block "sits between" a 54%-design #21 and an
implementation-heavy #16, and the ledger's own columns say #16 was `3.12/6.49` =
**48% design**, essentially this block's 49%. The spread is 48%→54% and this
block sits on top of the 0.37× row, not between the two. The hypothesis may still
hold; the argument that this estimate is safely positioned inside it does not, and
the honest statement is that the design/impl split does not separate these two
rows at all. Something else does, and this issue is another data point toward
finding out what.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec             design=0.75 impl=0.08
item: milestone-review       design=0.20 impl=0.30
item: milestone-review       design=0.20 impl=0.30
item: greenfield-go-module   design=0.25 impl=0.22
item: greenfield-go-module   design=0.25 impl=0.22
item: milestone-review       design=0.20 impl=0.30
item: milestone-review       design=0.20 impl=0.30
item: greenfield-go-module   design=0.25 impl=0.22
item: smaller-go-module      design=0.03 impl=0.14
item: greenfield-go-module   design=0.25 impl=0.22
item: smaller-go-module      design=0.03 impl=0.14
item: milestone-review       design=0.20 impl=0.30
item: milestone-review       design=0.20 impl=0.30
item: cross-cutting-refactor design=0.12 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: real-api-discovery     design=0.00 impl=0.12
item: atlas-docs             design=0.03 impl=0.05
item: milestone-review       design=0.20 impl=0.30
item: milestone-review       design=0.20 impl=0.30
design-buffer: 0.15
total: 8.22
```

Item-to-task map. Process: plan authoring, then plan rounds 1 and 2 (both spent).
**M1** — `greenfield` ×2 = Task 1 (`parseRSS` + its fuzz) and Task 2
(`containsWord`/`usagesFrom`/`entryUsages`); then the M1 boundary and one
fix-then-re-review. **M2** — `smaller` ×2 = Task 3's store methods and its
conformance rows plus the corrupt-file test, itemised apart because the suite is
work the methods do not do; `greenfield` = Task 4's `httpFeed` + `cachingFeed`
with the three-outcome model; `smaller` = the fake; then the M2 boundary and one
fix round. **M3** — `cross-cutting` = Task 5's wiring across `deps`, `storeDeps`,
`openStore` and `withStore`; `smaller` = live conformance; `atlas-docs` = Task 6
Step 2; then the close boundary and one fix round.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

## Revisions

### 2026-08-22 — the consumer changed; NOAD's own examples join the feed

**Reason.** #10 now authors finished items rather than harvesting a word pool, and
the operator raised general web search as an alternative usage source.

**Delta.**

- **General Google search stays out, on the measurement already recorded above** —
  the SERP is a 91 KB JS shell needing a headless browser. That finding is why this
  issue is RSS-shaped, and it has not changed.
- **NOAD's own example sentences are a second usage source, and a free one.** The
  entry is already fetched, already parsed, offline, editorially curated, and
  register-correct. It complements the feed exactly where the feed is weakest: the
  measured thematic collapse (10 of 14 `sycophantic` headlines were about AI
  chatbots) is a *current-events* artifact, and the dictionary's examples are not
  current, which is the point. Authoring gets both.
- **What is cached is unchanged** (raw feed items), but the downstream consumer is
  now the authoring step, not a question at review time.

**Unchanged.** Feed shape, the personal-use terms note, the stateful fake, the
malformed-feed requirement, and live conformance on shape and coverage.

### 2026-08-26

Claimed and planned. The Spec needed no revision — the 2026-08-22 entry already
settled the two questions that mattered (RSS not the SERP, measured; and NOAD's
examples as a second source). What the plan adds is where the judgment lives:
`containsWord` is the one place that decides whether a headline really contains
the word, and it reuses `wordRuns` from #21 rather than growing the package's
third tokenizer.

Two decisions worth recording before implementation:

- **The fake serves BYTES, not parsed usages.** A fake returning `[]Usage` would
  skip `parseRSS` entirely, and the parser is where the risk is — malformed feeds
  are a Done-when row.
- **A failed fetch is not cached.** Caching it would turn one network blip into a
  permanently empty answer for that word. That asymmetry is why the cache is its
  own type rather than a flag on the HTTP source.

- 2026-08-26: M1 — `parseRSS` (`rss.go`), `store.NewsItem`, and the usage layer
  (`usage.go`). Capturing a real feed paid for itself twice: it turned up the
  `" - Publisher"` suffix the plan had not anticipated, and it showed the first
  fixture slice was 10 copies of one article — the thematic collapse the Spec
  measured, in miniature. The committed fixture is 13 items chosen for diversity:
  10 matches across 10 publishers plus 3 non-matches, so the filter has work to do
  and a test over it can tell `containsWord` from `return true`.
  The fuzz property was WRONG on its first run and the fuzzer said so in two
  seconds: a parsed title must be an ordered SUBSEQUENCE of the input, not a
  substring, because mixed content like `0<![CDATA[0]]>` legitimately
  concatenates to `00`. 1.69M execs clean after the correction.
  Five mutations run; four died first time. The survivor was "strip anything after
  the last dash" — my fixture ended in the publisher, so both rules cut at the
  same place. Fixed with a title carrying NO attribution, where the correct rule
  keeps the headline and the mutant truncates it.

- 2026-08-26: M1 boundary round 1 — FIX-THEN-SHIP, three findings, all addressed.
  BR-1: `usagesFrom` strips attribution then filters, and the ORDER is
  load-bearing — a headline whose only occurrence of the word is inside the
  publisher name survives the inverted order with text that does not contain the
  word. `TestUsagesFromTheCapturedFeed` already asserted every usage contains its
  word, but the captured feed holds no such headline, so the assertion never ran
  over the discriminating state. The assertion was right; the input never reached
  it. BR-2: the fuzz property was STILL wrong — subsequence dies to entity
  decoding (`&#39;`, which is what Google News emits for apostrophes) and to XML
  line-ending normalisation. Replaced with the Done-when's own claim as a bound:
  there cannot be more items than item tags. Sound under any decoding, and the
  three refuting shapes are now seeds. 2.2M execs clean. BR-3: the atlas described
  M2's cache in the present tense at the M1 boundary — the
  `atlas-claims-unbuilt-surface` family I recorded a lesson about in #21, back
  again. Corrected to name what M1 actually ships.

  Also, during that round a mutation copy-back silently deleted `bothSources`:
  the scratch backup predated it. That is the "a backup is only as good as the
  tree it was taken from" lesson recurring with a new cause, and the fix is a rule
  rather than more care — restore with `git checkout HEAD -- <file>`, which cannot
  be stale, and re-verify the mutation afterwards in case the restore undid the
  fix it was checking.

- 2026-08-26: M1 boundary round 2 — three more, all real, and two are mine
  recurring. BR-10: the count-bound I wrote in round 1 was the THIRD wrong
  property on one target, not the fix — `encoding/xml` matches by local name, so
  `<x:item>` is an item no textual count sees, and the fixture already declares a
  namespace prefix. Also `1026` as a year is a date the input genuinely supplied,
  which my plausibility check called invented. The rule now stated rather than a
  fourth property: a fuzz property may assert only what THIS CODE guarantees over
  arbitrary input, never a property of the input's textual form. `FuzzParseRSS`
  keeps one own-contract claim; `parsePubDate` got its own target on the STRING,
  where it kills the guessing mutant on the seed corpus with no `-fuzz` run.
  BR-11: that guessing mutant survived the entire M1 suite, because my date
  assertion was guarded on `!At.IsZero()` — skipping exactly the case it claimed
  to pin. Second in the `output-field-unasserted` family, so the enumeration was
  swept: `Usage{Text,Source,Publisher,URL,At}` and `NewsItem.At`-on-unreadable now
  each have an assertion that reddens when the field is blanked or invented.
  BR-12: I put the `Review-Verdict:` trailer on a FIX commit — the exact failure
  recorded in lessons.md from #16 M2. Consequence: round 2's window resolved to
  empty, and `b0e907e` (M2's code) plus `61c9774` sat in NO review window. Fixed
  forward by reviewing M2 over a WIDENED window from M1's real boundary rather
  than by rewriting history — see the M2 close.

- 2026-08-26: M2 + M3 — the cache with its three outcomes, the stateful fake, the
  seam, the wiring, and live conformance. The wiring test was written FIRST this
  time (#21's lesson applied before the fact rather than after) and immediately
  earned it: the no-capture path had no usage source at all. `DEFINE_NO_CAPTURE`
  means "write nothing into this directory", not "the feed does not exist" — the
  same reading that gives that path a `memHistory` — so it now gets the seam
  cached in memory.
  Live conformance run against the real feed: 100 items, 96 usages for
  `ephemeral`, attribution stripped, and the personal-use terms still present in
  the body. It PRINTS what it fetched, which is how a person looks at real output
  until #10 exists.
  A second restore hazard, different from this morning's: `git checkout HEAD --`
  reverted UNCOMMITTED wiring. "Restore from git" only holds if the thing you are
  restoring TO is committed — so commit before mutating, which is now the rule.

- 2026-08-26: close boundary round 1 — FIX-THEN-SHIP, one Critical, all
  addressed. BR-13 (Critical): my wiring test drove production wiring and then
  CALLED the seam, so every plain `go test` fetched news.google.com and wrote
  ~48 KB of live headlines into a temp dir — while its own comment claimed
  "without a network". What hid it is the feature working: `bothSources` degrades
  to the dictionary, so `len(got) != 0` held either way and the test was green
  with and without a network. Split into two: the wiring hop on production deps,
  the offline claim on a source built with no feed. `Fetch` coverage went 72.7% →
  0.0% under an untagged run, which is the measurement that proves it.
  BR-14: every fallback in `news.go` was a comment nothing checked — inverting
  "an unreadable cache is a miss, not a failure" left the suite green, with the
  `failingStore` fixture sitting unused in the tree. The enumeration was the
  coverage profile; `items` is now 100% and `Fetch` 92.9%, driven against an
  `httptest` server rather than Google.
  BR-15: `Usages` returned an error that was nil on every path — a dead branch
  for #10 — and dropped the feed error silently. The error is gone from the
  interface (the dictionary half always answers, so there is nothing to fail) and
  a failed fetch now warns once per session, the shape `storeCapturer` already
  uses.

- 2026-08-26: close round 2 — the two Importants were not addressed after all,
  and the reviewer was right on both. BR-15: I added `bothSources.warn`, wrote the
  warn-once logic, and tested it with a buffer passed into a HAND-BUILT struct —
  while both production sites left it nil. A broken feed degraded exactly as
  silently as before. Third recurrence of one shape in two issues (#21's vocab
  Load reached 1 of 3 entry paths; #9 M3's no-capture path had no seam; this one
  0 of 2 sites), and the sentence that covers all three is: a test that
  CONSTRUCTS the struct begins after the hop that fills it. Now wired at both
  sites and pinned by a test that builds deps the way a process does; unwiring
  either site reddens it. BR-14: the branches WERE pinned, but two items named in
  the finding's own text were not swept — the quoted-word query (dropping the
  quotes changed no test, because a fake serves its fixture whatever was asked
  for; now pinned against a captured request) and `withoutAttribution`'s
  empty-publisher arm. That second fixture did not discriminate on the first
  attempt: without a publisher the strip becomes `CutSuffix(title, " - ")`, which
  leaves an ordinary title alone either way, so the title has to END in " - " for
  the guard to be observable.

- 2026-08-26: close round 3 — one blocking finding, and a good one. `usage/` is a
  third directory `define` writes into the working directory, and it reached none
  of the three places that needed it: `.gitignore`, the index guard, the history
  guard. Both guards hardcoded the two older names and `.gitignore` listed them
  again — three copies with nothing keeping them in step. This is the deck-in-git
  class, which `.gitignore`'s own comment records as having cost three review
  rounds before this one. Fixed as the class: `store.RuntimeDirs` is the single
  source where the writer lives, both guards ask it, and
  `TestGitignoreCoversRuntimeDirs` closes the loop the compiler cannot — adding a
  fourth directory and forgetting the ignore now fails, and so does re-making the
  un-anchored/anchored mistake that caused the earlier rounds.
