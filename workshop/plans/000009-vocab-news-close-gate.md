---
gate: boundary-review
issue: 9
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-26T20:12:36-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: usagesFrom's strip-then-filter order is unpinned; inverting it yields usages whose text lacks the word
          detail: |-
            cmd/define/usage.go:74-77. Verified: changing `containsWord(text, word)` to
            `containsWord(it.Title, word)` leaves the whole cmd/define suite green, and these
            functions have no callers outside usage.go/usage_test.go so nothing else could catch
            it. Measured consequence with a scratch fixture — {Title: "Best startups of 2026 -
            Ephemeral Media", Source: "Ephemeral Media"} yields Text="Best startups of 2026" with
            containsWord=false, violating the Done-when row "only headlines that genuinely contain
            the word become usages". TestUsagesFromTheCapturedFeed:100 already holds the right
            assertion; it cannot fire because no fixture row puts the word only in the attribution.
            One row closes it.
          family: fixture-does-not-discriminate
          round: 1
        - id: BR-2
          severity: Important
          title: FuzzParseRSS's subsequence property reports correct XML parsing as invented content
          detail: |-
            cmd/define/rss_test.go:156. The comment names entity decoding as the reason substring
            is too strong, but subsequence does not survive it either. Verified by seeding a
            scratch copy: `&#65;` -> "A", `&#39;q&#39;` -> "'q'", and a lone `\r` -> "\n" (XML
            line-ending normalisation) each fail with "the parser invented content" against
            entirely correct parsing. `&#39;` is what Google News emits for apostrophes, and the
            plan's Task 1 Step 1 makes re-capturing the fixture routine, so this can go red
            deterministically; the plan's own "fuzz for 60s" step will find it eventually. The
            hazard is the response to a false failure — weakening the property that defends a
            Done-when row. Skip when the input carries `&#` or `\r`, or normalise before
            comparing.
          family: property-rejects-correct-behaviour
          round: 1
        - id: BR-3
          severity: Important
          title: The atlas describes M2's cache in the present tense at the M1 boundary
          detail: |-
            atlas/define.md, "Raw is cached; usable is derived" — at 98d8ccc there is no
            Store.NewsItems, no cachingFeed and no httpFeed, yet the section states that
            store.NewsItem "is what goes to disk", that "every word already cached" improves, and
            that "the feed is queried", with no scope marker. This is verbatim the class
            workshop/lessons.md records from #21 M1, whose stated fix is to write the milestone
            into the sentence. Overtaken in fact: b0e907e (outside this window) landed the cache,
            so the claim is accurate at current HEAD — dispose withdrawn if preferred. Reported
            because the rule recurred, and the closing step is what prevents the next instance.
          family: docs-describe-unshipped-milestone
          round: 1
        - id: BR-4
          severity: Minor
          title: Usage.URL and Usage.At are never asserted, so blanking them is invisible
          detail: |-
            cmd/define/usage.go:82-83. Replacing them with zero values leaves the suite green,
            and they are the two fields #10 needs to attribute a sentence to its source.
          family: output-field-unasserted
          round: 1
        - id: BR-5
          severity: Minor
          title: entryUsages' multi-block traversal is unpinned by an at-least-one assertion
          detail: |-
            Restricting the loop to e.Blocks[0] survives the suite, because
            TestEntryUsagesLiftsNOADExamples:175 asserts only len(got) != 0 plus per-item
            properties. A count, or an example known to live under a later block, closes it.
          family: at-least-one-hides-undercollection
          round: 1
        - id: BR-6
          severity: Minor
          title: renderSpans duplicates marked from highlight_test.go in the same package
          detail: |-
            cmd/define/usage_test.go:73 is a byte-for-byte reimplementation of `marked`, which is
            already in package main and directly usable — ARCH-DRY, in the milestone whose thesis
            is that the matcher must not be reimplemented.
          family: helper-duplicated-in-package
          round: 1
        - id: BR-7
          severity: Minor
          title: store.NewsItem.Source is the publisher while Usage.Source is provenance
          detail: |-
            usagesFrom writes `Publisher: it.Source` — the same identifier carries two meanings one
            line apart, and #10 will read both types. Renaming the store field Publisher removes
            the trap before it has a second reader.
          family: name-means-two-things
          round: 1
        - id: BR-8
          severity: Minor
          title: A punctuated deck key can never yield a usage, and nothing says so
          detail: |-
            Verified: containsWord returns false for "e.g.", "9/11" and "rock 'n' roll" even when
            the text contains them verbatim, inherited from #21's phraseRunsJoin bound on
            MaxPhraseWords. In #21 the cost was "not highlighted"; here it is "this word can never
            be taught from a real sentence", which is a different stake and is documented in
            neither the code comment nor the atlas.
          family: matcher-limit-silently-drops-input
          round: 1
        - id: BR-9
          severity: Minor
          title: Every plan checkbox is unticked at the boundary, and five prose claims contradict the code
          detail: |-
            workshop/plans/000009-vocab-news-plan.md: all of Task 1's and Task 2's steps are
            unticked while the issue marks M1 [x]. The prose also states Usage{...Title...}
            (shipped: Publisher), store.NewsItem{Title, URL, At} (shipped: plus Source),
            entryUsages(e Entry) (shipped: plus word), a "substring" fuzz property (shipped:
            subsequence, correction recorded in the Log but not the plan), and Task 2 Step 5's
            "12-99 RANGE" which is impossible against a 13-item fixture — the code's exact-10
            assertion is the better call and should be recorded as such.
          family: plan-record-drift
          round: 1
      boundary: M1
      blocked: true
---

# Gate ledger — tools#9 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-26T20:12:36-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `fixture-does-not-discriminate` usagesFrom's strip-then-filter order is unpinned; inverting it yields usages whose text lacks the word
  cmd/define/usage.go:74-77. Verified: changing `containsWord(text, word)` to
  `containsWord(it.Title, word)` leaves the whole cmd/define suite green, and these
  functions have no callers outside usage.go/usage_test.go so nothing else could catch
  it. Measured consequence with a scratch fixture — {Title: "Best startups of 2026 -
  Ephemeral Media", Source: "Ephemeral Media"} yields Text="Best startups of 2026" with
  containsWord=false, violating the Done-when row "only headlines that genuinely contain
  the word become usages". TestUsagesFromTheCapturedFeed:100 already holds the right
  assertion; it cannot fire because no fixture row puts the word only in the attribution.
  One row closes it.
- **BR-2** [Important] `property-rejects-correct-behaviour` FuzzParseRSS's subsequence property reports correct XML parsing as invented content
  cmd/define/rss_test.go:156. The comment names entity decoding as the reason substring
  is too strong, but subsequence does not survive it either. Verified by seeding a
  scratch copy: `&#65;` -> "A", `&#39;q&#39;` -> "'q'", and a lone `\r` -> "\n" (XML
  line-ending normalisation) each fail with "the parser invented content" against
  entirely correct parsing. `&#39;` is what Google News emits for apostrophes, and the
  plan's Task 1 Step 1 makes re-capturing the fixture routine, so this can go red
  deterministically; the plan's own "fuzz for 60s" step will find it eventually. The
  hazard is the response to a false failure — weakening the property that defends a
  Done-when row. Skip when the input carries `&#` or `\r`, or normalise before
  comparing.
- **BR-3** [Important] `docs-describe-unshipped-milestone` The atlas describes M2's cache in the present tense at the M1 boundary
  atlas/define.md, "Raw is cached; usable is derived" — at 98d8ccc there is no
  Store.NewsItems, no cachingFeed and no httpFeed, yet the section states that
  store.NewsItem "is what goes to disk", that "every word already cached" improves, and
  that "the feed is queried", with no scope marker. This is verbatim the class
  workshop/lessons.md records from #21 M1, whose stated fix is to write the milestone
  into the sentence. Overtaken in fact: b0e907e (outside this window) landed the cache,
  so the claim is accurate at current HEAD — dispose withdrawn if preferred. Reported
  because the rule recurred, and the closing step is what prevents the next instance.
- **BR-4** [Minor] `output-field-unasserted` Usage.URL and Usage.At are never asserted, so blanking them is invisible
  cmd/define/usage.go:82-83. Replacing them with zero values leaves the suite green,
  and they are the two fields #10 needs to attribute a sentence to its source.
- **BR-5** [Minor] `at-least-one-hides-undercollection` entryUsages' multi-block traversal is unpinned by an at-least-one assertion
  Restricting the loop to e.Blocks[0] survives the suite, because
  TestEntryUsagesLiftsNOADExamples:175 asserts only len(got) != 0 plus per-item
  properties. A count, or an example known to live under a later block, closes it.
- **BR-6** [Minor] `helper-duplicated-in-package` renderSpans duplicates marked from highlight_test.go in the same package
  cmd/define/usage_test.go:73 is a byte-for-byte reimplementation of `marked`, which is
  already in package main and directly usable — ARCH-DRY, in the milestone whose thesis
  is that the matcher must not be reimplemented.
- **BR-7** [Minor] `name-means-two-things` store.NewsItem.Source is the publisher while Usage.Source is provenance
  usagesFrom writes `Publisher: it.Source` — the same identifier carries two meanings one
  line apart, and #10 will read both types. Renaming the store field Publisher removes
  the trap before it has a second reader.
- **BR-8** [Minor] `matcher-limit-silently-drops-input` A punctuated deck key can never yield a usage, and nothing says so
  Verified: containsWord returns false for "e.g.", "9/11" and "rock 'n' roll" even when
  the text contains them verbatim, inherited from #21's phraseRunsJoin bound on
  MaxPhraseWords. In #21 the cost was "not highlighted"; here it is "this word can never
  be taught from a real sentence", which is a different stake and is documented in
  neither the code comment nor the atlas.
- **BR-9** [Minor] `plan-record-drift` Every plan checkbox is unticked at the boundary, and five prose claims contradict the code
  workshop/plans/000009-vocab-news-plan.md: all of Task 1's and Task 2's steps are
  unticked while the issue marks M1 [x]. The prose also states Usage{...Title...}
  (shipped: Publisher), store.NewsItem{Title, URL, At} (shipped: plus Source),
  entryUsages(e Entry) (shipped: plus word), a "substring" fuzz property (shipped:
  subsequence, correction recorded in the Log but not the plan), and Task 2 Step 5's
  "12-99 RANGE" which is impossible against a 13-item fixture — the code's exact-10
  assertion is the better call and should be recorded as such.

## Open findings

- **BR-1** [Important] `fixture-does-not-discriminate` usagesFrom's strip-then-filter order is unpinned; inverting it yields usages whose text lacks the word
- **BR-2** [Important] `property-rejects-correct-behaviour` FuzzParseRSS's subsequence property reports correct XML parsing as invented content
- **BR-3** [Important] `docs-describe-unshipped-milestone` The atlas describes M2's cache in the present tense at the M1 boundary
- **BR-4** [Minor] `output-field-unasserted` Usage.URL and Usage.At are never asserted, so blanking them is invisible
- **BR-5** [Minor] `at-least-one-hides-undercollection` entryUsages' multi-block traversal is unpinned by an at-least-one assertion
- **BR-6** [Minor] `helper-duplicated-in-package` renderSpans duplicates marked from highlight_test.go in the same package
- **BR-7** [Minor] `name-means-two-things` store.NewsItem.Source is the publisher while Usage.Source is provenance
- **BR-8** [Minor] `matcher-limit-silently-drops-input` A punctuated deck key can never yield a usage, and nothing says so
- **BR-9** [Minor] `plan-record-drift` Every plan checkbox is unticked at the boundary, and five prose claims contradict the code
