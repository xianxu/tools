# Boundary Review — tools#9 (whole-issue close)

| field | value |
|-------|-------|
| issue | 9 — news seam: Google News RSS client with a stateful fake |
| repo | tools |
| issue file | workshop/issues/000009-vocab-news.md |
| boundary | whole-issue close |
| milestone | — |
| window | 361136b3e46f21e67dd307a6c5ed31c62b70148f..b2456479d2144998b57c6c0d71ef8256cf48da90 |
| command | sdlc close --issue 9 |
| reviewer | claude |
| timestamp | 2026-08-26T21:07:45-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The close window is correct this time — its base equals `git merge-base main HEAD`, so all eight of #9's commits including the previously-stranded `b0e907e`/`61c9774`/`6ee2044` are in it, and I reviewed them. The shipped production code is sound: `go build`, `go vet` and `go test ./...` all pass, 91.7% statement coverage in `cmd/define`, every Done-when row is delivered, and the two Importants from round 2 are genuinely fixed — I confirmed both by reverting (blanking `Usage.URL`/`At` reddens `usage_test.go:270`; a `parsePubDate` that guesses reddens `rss_test.go:87` *and* two `FuzzParsePubDate` seeds). What blocks SHIP is one thing I measured rather than inferred: `TestWithStoreCarriesTheUsageSourceThrough` — an untagged unit test whose own comment says it "reaches the dictionary half **without a network**" — drives production wiring straight into `httpFeed` and makes a live request to news.google.com. I probed it: 95 news usages, 48 KB of real Google News written to the temp dir, and `httpFeed.Fetch` reports 72.7% coverage in the *untagged* run, which is only possible if the default suite is fetching. The test passes with or without the network (NOAD alone satisfies `len(got) != 0`), so the call is pure side effect and the "offline" half of the assertion proves nothing. That is ARCH-MOCK's central case, with the stateful fake sitting one file away. Behind it, every error branch in `news.go` is uncovered and I verified one survives inversion. Five Minors from round 1 (BR-5 through BR-9) are untouched after two rounds and are crossing the last gate.

## 1. Strengths

- **The cache asserts on the fake's counter, not on output** — `news_test.go:31,50,68,88`. `f.fetches(word)` is exactly the right oracle: "served from cache" and "re-fetched and got the same answer" are indistinguishable from the returned items, and this is the one design that tells them apart. `fakeFeed` serving **bytes** rather than parsed items keeps `parseRSS` in both the production and test paths (`news_fake_test.go:10-14`), which is what makes the malformed-feed Done-when row testable at the seam.
- **The three cache outcomes are real, and the third is the non-obvious one** (`news.go:106-127`). Outcome 2 — a successful fetch with zero items is cached *with its timestamp* — is what stops the feed's worst words from re-fetching forever, and `storetest/suite.go:288-338` pins "never fetched" vs "fetched and found nothing" as a store-level contract both `Mem` and `YAML` must answer. That distinction is the reason `NewsItems` returns a `time.Time` at all.
- **`containsWord` delegating to `highlightSpans`** (`usage.go:55-67`) remains the best decision in the issue, and `TestContainsWordAgreesWithTheHighlighter` is a genuine pin on it rather than a restatement. ARCH-DRY earning its keep.
- **BR-11's fix swept the class, not the site.** `TestEveryUsageFieldCarriesThrough` (`usage_test.go:243`) is a table over the fields, so a *new* field is conspicuous by absence — and `FuzzParsePubDate` (`rss_test.go:194`) puts the property on the string where no decoder stands between it and the code. Both kill their mutants on the seed corpus with no `-fuzz` run.
- **The live conformance check asserts the licence terms** (`news_conformance_test.go:71`). Asserting that the feed body still says "personal, non-commercial use" makes the thing we are agreeing to falsifiable rather than a comment. That is the right instinct.

## 2. Critical findings

**C1 — `cmd/define/news_test.go:228` makes a live HTTP request to news.google.com on every `go test ./cmd/define/`, and the test claims the opposite.**

The test builds production wiring (`deps{newStore: openStore}.withStore(...)`), which at `main.go:215` constructs `&bothSources{news: newCachingFeed(newHTTPFeed(), st, clk)}` — the *real* `httpFeed`. Then line 228 calls `Usages`, which misses the fresh temp-dir cache and fetches. Measured with a scratch probe replicating the same wiring:

```
usages=101 err=<nil>
news usages = 95
wrote bank.yaml (48334 bytes)
```

and per-function coverage of the untagged run confirms it independently:

```
news.go:44:  Fetch   72.7%      # 0% if nothing untagged called it
```

Three things follow. (a) The comment at `news_test.go:226-227` — "it reaches the dictionary half **without a network**" — and the message at `:233` — "no usages **offline**" — are both false; the assertion `len(got) != 0` is satisfied by NOAD alone, so the test is green with or without network and verifies neither the offline claim nor the degradation. (b) `news_conformance_test.go:7-9` states the repo's own rule — network "does not belong in merge-check.yml" — and this test is untagged, so it opts out of that rule silently. (c) ARCH-MOCK: `fakeFeed` exists and is stateful, but here production flow and test flow do **not** share the boundary.

Fix sketch: keep the wiring assertion (`d.usage == nil`) exactly as it is — that is the #21 lesson and it is worth having — and replace the `Usages` call with one that cannot reach the network. Either assert on the *shape* of what `openStore` built without invoking it, or add a `newStore` variant that injects `newFakeFeed(...)` at the `feed` seam and then assert `len(got) != 0` **and** that every usage is `usageNOAD` when the fake is scripted to fail. The second form actually pins the offline claim the current comment makes.

## 3. Important findings

**I1 — every error branch in `news.go` is uncovered, and I verified one survives inversion.** `cmd/define/news.go:96-101`: the comment states "an unreadable cache is a miss, not a failure". Replacing that block with `return nil, err` — the exact opposite policy — leaves the **entire** `cmd/define` suite green (measured, full 94s run). `failingStore` (`history_store_test.go:78-81`) already implements `NewsItems`/`SetNewsItems` returning `errFail`, so the fixture to close this is already in the tree and unused for this purpose.

The rule, since this is the sibling of BR-11's: **a comment that states "on failure X we do Y instead" needs one test that goes red when Y is removed.** BR-11 swept that rule over output *fields*; the degradation *paths* were never enumerated. The enumeration, taken from the coverage profile — every uncovered block in the two new files:

| site | documented contract | pinned? |
|---|---|---|
| `news.go:51-53` | `NewRequestWithContext` error propagates | no |
| `news.go:55-57` | transport failure propagates | no |
| `news.go:59-61` | non-200 is an error | no |
| `news.go:97-101` | unreadable cache = miss | **no — mutation verified** |
| `news.go:114-118` | parse failure falls back to stale, else errors | no |
| `news.go:122-126` | a write failure still returns the answer | no |
| `usage.go:99-101` | empty publisher = no stripping | no |

Rows 1–3 close with one `httptest.NewServer` test on `httpFeed` (which today has **no** hermetic test at all — its URL construction is also unpinned, and the quoted-word query at `news.go:48` is a Spec-measured, load-bearing detail that a mutation dropping the quotes would ship green). Rows 4–6 close with `failingStore` and a `fakeFeed` body of non-XML bytes. Row 7 is one line.

**I2 — `cmd/define/usage.go:152` discards the feed error with no signal at all, and `UsageSource.Usages` can never return one.** `if items, err := b.news.items(ctx, word); err == nil { ... }` — the error is dropped, nothing is warned, and the method's own `error` return is unconditionally `nil` on every path. Degradation is the right *design*; silent degradation is not this package's shape. `openStore` warns when it falls back (`main.go:184`), `YAML.NewsItems` warns on a corrupt cache (`yaml.go:388`), and `cachingAudioSource` propagates its error and lets the caller decide (`fetch.go:122-131`). Here a permanently broken feed — wrong URL, TLS failure, Google blocking the UA — is indistinguishable from "this word is not in the news", for both the user and #10, which is the stated consumer. Fix sketch: give `bothSources` the `warn io.Writer` the rest of the package already threads and warn once per failed fetch; if the `error` return is genuinely meant to always be nil, say so in the interface doc so #10 does not write a dead error branch against it.

## 4. Minor findings

- `news.go:122-126` — `if err := c.st.SetNewsItems(...); err != nil { return items, nil }` followed by `return items, nil`. Both branches are identical, so the `if` is dead and unmutatable; `_ = c.st.SetNewsItems(...)` with the same comment says it honestly.
- `rss.go:16-38` — `parseRSS` never checks the root element, so any well-formed non-RSS XML returns `(empty, nil)`. Round 2 noted this as M2 scope; M2 is in this window, and at M2 it becomes cache-outcome-2 — a wrong-content 200 frozen as "no news for this word" for a full `cacheTTL`.
- `rss_test.go:181` — `FuzzParseRSS`'s remaining property (`err != nil && items != nil`) can never fire: `parseRSS`'s only error return is `return nil, err`. The target is now a no-panic smoke test. **Not a recommendation to write a fourth property** — BR-10 was right about that. The recommendation is to record in the plan/Log that the Done-when's second clause ("never returns an item it did not find in the input") is pinned by the exact-count and exact-title assertions in `TestParseRSSOverACapturedFeed` and `TestParseRSSDecisions`, not by the fuzz target.
- `atlas/define.md:216-219` — the canonical on-disk layout block still lists only `words/`, `events/`, `user-model.md`, while the new section at `:1064` says `usage/<slug>.yaml` sits "beside `words/` and `events/`". One fact, two places, one stale.
- `storetest/suite.go:381` — stray blank line before the closing brace.
- No README finding: `deps.usage` has zero consumers, so `define` never creates `usage/` in production today, and the plan's "No README line: nothing user-facing changed in this window" is the correct call. It becomes a README row the moment #10 lands.

## 5. Test coverage notes

Coverage is 91.7% of statements in `cmd/define`, and the gap is not random — it is exactly the failure paths (I1). The happy paths of the cache are pinned unusually well, on the fake's counter rather than on output, and the storetest conformance rows mean `Mem` and `YAML` genuinely answer one contract. Mutation results this round, all against a clean tree with `git checkout HEAD --` restores verified: **dead** — blank `Usage.URL`/`At`; `parsePubDate` returning a fixed date. **Survived** — `entryUsages` restricted to `e.Blocks[:1]` (BR-5, re-confirmed against the full suite); `cachingFeed` propagating the store-read error instead of treating it as a miss. On BR-10's enumeration: I independently read all 12 fuzz targets in the tree (`cmd/define` ×10, `store` ×1, `internal/llm` ×1) and none carries the input-textual-form defect — each asserts its own function's contract with no third-party decoder between the property and the code. `FuzzDecode`'s independent-oracle presence check and `FuzzRenderLosesNothing`'s alnum-subsequence property both survive the rule because the transformer in between is ours, not `encoding/xml`'s. The class is clean.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag (BR-6, still open).** `renderSpans` (`usage_test.go:74`) is byte-identical to `marked` (`highlight_test.go:110`) apart from returning `[]byte`, and the one call site wraps it back in `string(...)`. Two rounds old, in the milestone whose thesis is that the matcher must not be reimplemented. Otherwise clean: `containsWord`→`highlightSpans` and `parseRSS` returning `store.NewsItem` rather than an `encoding/xml` struct are both the right seams.
- **ARCH-PURE — pass.** `parseRSS`, `parsePubDate`, `containsWord`, `usagesFrom`, `withoutAttribution`, `entryUsages` are all pure and all tested with no doubles at all; the IO is confined to `httpFeed` behind the `feed` byte interface. The layering (`bothSources` → `cachingFeed` → `feed`) mirrors `fetch.go` as the plan promised.
- **ARCH-PURPOSE — flag.** All seven Done-when rows are delivered and the seam is genuinely complete, which is the pass part. The flag is the pattern: BR-11 was answered by sweeping the *field* enumeration, and the adjacent class — documented degradation paths — was never enumerated, which is I1. Separately, five Minors named in round 1 (BR-5, 6, 7, 8, 9) are unchanged at the final gate; four of them are one-line fixes.
- **ARCH-MOCK — flag, and it is C1.** The fake is stateful and byte-shaped, the counter is the right assertion, and the live conformance check is real and asserts a floor plus the licence terms — all correct. But the principle's own closing sentence is "a fake satisfies this only when production flow and test flow share the same boundary", and `news_test.go:228` is the counter-example: the one test driven through production wiring bypasses the fake entirely. `httpFeed` also has no double-backed test, so the seam's boundary is exercised *only* live.
- For #10: `Usage.At` is the item's pubDate and can legitimately be the zero time (unreadable date). If #10 sorts by recency, decide now whether zero-time usages sort last or are dropped — the parser deliberately hands you "we do not know when" rather than a guess, and that is only useful if the consumer reads it.

## 7. Plan revision recommendations

`workshop/plans/000009-vocab-news-plan.md` still has **zero** ticked checkboxes across all six tasks, and a `## Revisions` section containing only the plan-quality round-1 entry. Beyond BR-9's five prose drifts, the Integration table at `:88` lists `newsUsageSource | cmd/define/news.go | new` — an entity that **exists nowhere in the tree**. That is a core-concepts-table-vs-code contradiction. Add a `## Revisions` entry:

```
### 2026-08-26 — as built (M1–M3)

- `Usage` ships `{Text, Source, Publisher, URL, At}`, not `{…Title…}`.
- `store.NewsItem` ships `{Title, URL, Source, At}`; the prose omits `Source`,
  which `withoutAttribution` depends on.
- `entryUsages(e Entry, word string)`, not `entryUsages(e Entry)`.
- The Integration table's `newsUsageSource` row was never built: `bothSources`
  wraps `cachingFeed` directly. Delete the row.
- The fuzz property is not "title is a substring of the input". Substring,
  subsequence and the count-bound were all retired for failing on correct
  parsing; record the RULE, not a fourth attempt.
- Task 2 Step 5's "12–99 RANGE" is impossible against a committed 13-item
  fixture; the exact-10 assertion the code ships is correct.
- Tick Task 1 Steps 1–7, Task 2 Steps 1–7, Task 3 Steps 1–6, Task 4 Steps 1–9,
  Task 5 Steps 1–3, Task 6 Steps 1–2.
```

`workshop/issues/000009-vocab-news.md:57-58` needs the same treatment — the Plan rows are ticked while naming three things that do not exist: `Store.Usages` (shipped as `Store.NewsItems`/`SetNewsItems`), the `NewsSource` seam (shipped as `feed` + `UsageSource`), and `/usage` (**deliberately dropped**; the plan-gate's PQ-4 disposition already said this row "should be tidied to match" and it never was). A ticked box claiming a subcommand that does not exist is the worst of the three.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      Verified by reverting: blanking URL and At reddens usage_test.go:270 on both fields.
  - id: BR-5
    disposition: not-addressed
    note: |
      Re-verified against the FULL suite: e.Blocks[:1] leaves all of cmd/define green.
  - id: BR-6
    disposition: not-addressed
    note: |
      renderSpans (usage_test.go:74) is still byte-identical to marked (highlight_test.go:110).
  - id: BR-7
    disposition: not-addressed
    note: |
      store/news.go:15 still names the publisher field Source, one line from Usage.Source.
  - id: BR-8
    disposition: not-addressed
    note: |
      Re-verified false for "e.g.", "9/11", "rock 'n' roll"; still undocumented in usage.go and atlas.
  - id: BR-9
    disposition: not-addressed
    note: |
      Worse than reported - the Integration table names newsUsageSource, which exists nowhere, and the issue's own M2/M3 rows are ticked while naming Store.Usages, NewsSource and /usage, none of which exist.
  - id: BR-10
    disposition: addressed
    note: |
      Count-bound gone; rule stated in lessons.md and the target's doc. I read all 12 fuzz targets in the tree - no other instance of the class.
  - id: BR-11
    disposition: addressed
    note: |
      Verified by reverting: the guessing mutant reddens rss_test.go:87 and FuzzParsePubDate seeds 1 and 6.
  - id: BR-12
    disposition: addressed
    note: |
      This window's base equals git merge-base main HEAD, so all eight #9 commits including b0e907e are covered; no M2/M3 commit carries a verdict trailer.
findings:
  - id: new
    severity: Critical
    family: unit-test-reaches-real-dependency
    title: |
      TestWithStoreCarriesTheUsageSourceThrough makes a live request to news.google.com on every untagged test run
    detail: |
      cmd/define/news_test.go:228 drives production wiring (openStore builds newHTTPFeed at
      main.go:215) and then calls Usages, so the default `go test ./cmd/define/` fetches the
      real feed. Measured with a scratch probe on the same wiring: 101 usages, 95 of them
      usageNews, 48334 bytes of live Google News written into the temp dir; and untagged
      coverage reports news.go:44 Fetch at 72.7%, which is impossible unless the default suite
      calls it. The test's own comment at :226-227 says it "reaches the dictionary half without
      a network" and its failure message at :233 says "no usages offline" - both false. Because
      the assertion is only len(got) != 0, NOAD alone satisfies it, so the test is green with or
      without the network and verifies neither the offline claim nor the degradation. This is
      ARCH-MOCK's central case with fakeFeed sitting one file away, and it silently opts out of
      the rule news_conformance_test.go:7-9 states in as many words. Fix: keep the d.usage != nil
      wiring assertion, and inject the fake at the feed seam so the offline claim is actually
      exercised - script it to fail and assert every returned usage is usageNOAD.
  - id: new
    severity: Important
    family: documented-degradation-unpinned
    title: |
      Every error branch in news.go is uncovered, and the store-read degradation survives inversion
    detail: |
      Measured: replacing news.go:97-101 ("an unreadable cache is a miss, not a failure") with
      `return nil, err` - the opposite policy - leaves the entire cmd/define suite green over a
      full run. failingStore (history_store_test.go:78-81) already returns errFail for both new
      methods, so the fixture is in the tree and unused. The rule, sibling to BR-11's: a comment
      stating "on failure X we do Y instead" needs one test that reddens when Y is removed.
      BR-11 swept that rule over output FIELDS; the degradation PATHS were never enumerated.
      Enumeration, from the coverage profile - every uncovered block in the two new files:
      news.go:51-53 (NewRequestWithContext error), 55-57 (transport failure), 59-61 (non-200 is
      an error), 97-101 (unreadable cache = miss, verified unpinned), 114-118 (parse failure
      falls back to stale else errors), 122-126 (write failure still returns the answer), and
      usage.go:99-101 (empty publisher = no stripping). Rows 1-3 close with one httptest.Server
      test on httpFeed, which today has NO hermetic test at all - its URL construction is also
      unpinned, and the quoted-word query at news.go:48 is a Spec-measured, load-bearing detail
      a mutation could drop while staying green. Rows 4-6 close with failingStore plus a fakeFeed
      body of non-XML bytes.
  - id: new
    severity: Important
    family: degradation-without-signal
    title: |
      bothSources discards the feed error with no warning, and UsageSource.Usages can never return one
    detail: |
      cmd/define/usage.go:152 - `if items, err := b.news.items(ctx, word); err == nil` drops the
      error, warns nothing, and the method returns nil error on every path, so the interface's
      error result is vestigial. Degrading is the right design; degrading SILENTLY is not this
      package's shape: openStore warns when it falls back (main.go:184), YAML.NewsItems warns on
      a corrupt cache (yaml.go:388), and cachingAudioSource propagates and lets the caller decide
      (fetch.go:122-131). As shipped, a permanently broken feed - wrong URL, TLS failure, Google
      blocking us - is indistinguishable from "this word is not in the news" for both the user
      and for #10, the stated consumer. Fix: thread the warn io.Writer the package already
      carries and warn once per failed fetch; if the error return is genuinely always nil, say so
      in the UsageSource doc so #10 does not write a dead branch against it.
  - id: new
    severity: Minor
    family: dead-branch
    title: |
      cachingFeed's SetNewsItems error handler and its fallthrough return the same value
    detail: |
      news.go:122-126 is `if err := c.st.SetNewsItems(...); err != nil { return items, nil }`
      followed by `return items, nil`. Both branches are identical, so the guard is dead and
      cannot be mutated. `_ = c.st.SetNewsItems(...)` with the same comment says it honestly.
  - id: new
    severity: Minor
    family: parser-accepts-wrong-document
    title: |
      parseRSS never checks the root element, so non-RSS XML becomes a cached "no news"
    detail: |
      rss.go:16-38 unmarshals into a struct keyed only on channel>item, so any well-formed XML
      that is not an RSS feed returns (empty, nil). Round 2 noted this as M2 scope; M2 is in this
      window, and at M2 it becomes cache outcome 2 - a wrong-content 200 frozen as "no news for
      this word" for a full cacheTTL.
  - id: new
    severity: Minor
    family: assertion-cannot-fire
    title: |
      FuzzParseRSS's remaining property can never fire, so the target is now a no-panic smoke test
    detail: |
      rss_test.go:181 asserts `err != nil && items != nil`, but parseRSS's only error return is
      `return nil, err`, so the guard is unreachable by construction. This is NOT a request for a
      fourth property - BR-10 was right about that. The recommendation is to record, in the plan
      and the Log, that the Done-when's second clause ("never returns an item it did not find in
      the input") is pinned by the exact-count and exact-title assertions in
      TestParseRSSOverACapturedFeed and TestParseRSSDecisions, not by the fuzz target.
  - id: new
    severity: Minor
    family: fact-documented-in-two-places
    title: |
      The atlas's canonical on-disk layout block does not list usage/
    detail: |
      atlas/define.md:216-219 shows words/, events/ and user-model.md as the layout, while the
      new section at :1064 says usage/<slug>.yaml sits "beside words/ and events/". One fact,
      two places, one stale - a reader who consults the layout map will not see the new
      directory.
```
