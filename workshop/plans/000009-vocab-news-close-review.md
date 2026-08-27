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

---

## Re-review — 2026-08-26T21:40:19-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 9 — news seam: Google News RSS client with a stateful fake |
| repo | tools |
| issue file | workshop/issues/000009-vocab-news.md |
| boundary | whole-issue close |
| milestone | — |
| window | 361136b3e46f21e67dd307a6c5ed31c62b70148f..efb806cdc983aeb61f3e441934eed28225f70ce1 |
| command | sdlc close --issue 9 |
| reviewer | claude |
| timestamp | 2026-08-26T21:40:19-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I've finished the review. Tree restored clean (`git status` verified after every mutation).

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The Critical from round 3 is genuinely dead: I patched `httpFeed.Fetch` to panic whenever `base == ""` — i.e. on any real-network call — and the full `go test ./cmd/define/` stayed green, so the default suite makes **zero** live requests to news.google.com. The wiring hop and the offline claim are now two tests, neither lying, and `httpFeed`'s error branches are driven against `httptest` instead of Google. `go build`, `go vet` (both untagged and `-tags conformance`) and `go test ./...` all pass; coverage is 92.1% in `cmd/define` with `cachingFeed.items` at 100%. Two Importants survive, both because the fix stopped one step short of the thing the finding named. **BR-15's warn writer is threaded to nothing**: `bothSources.warn` is nil at both production construction sites (`main.go:180`, `main.go:215` — probed, both `nil`), and `warnTo` returns early on nil, so a permanently broken feed is still perfectly silent in production. The field, the `warnOnce` method and its mutex all exist and are exercised only by a test that injects the writer itself — protection that reads as real and does nothing. **BR-14's own text named the quoted-word query as a load-bearing detail a mutation could drop while staying green**; I dropped the quotes and the whole suite is still green, because all three new `httptest` handlers take `_ *http.Request` and never look at what was requested. Behind those, nine Minors from rounds 1 and 3 are unchanged at the final gate, and the durable record is the worst of them: **zero of the plan's 26 checkboxes are ticked**, the Integration table still lists `newsUsageSource` (exists nowhere), and the issue's ticked M2/M3 rows still name `Store.Usages`, the `NewsSource` seam and `/usage` — none of which were built, `/usage` deliberately.

## 1. Strengths

- **BR-13's fix is real, and it is the right shape.** Splitting the one test into a wiring assertion on production deps (`news_test.go:216`) and an offline assertion on a feedless source (`news_test.go:240`) makes "offline" a property of the code rather than of the test machine. Verified by probe, not by reading: `Fetch` is never entered without a `base`.
- **BR-14's central claim is genuinely closed and I confirmed it by reverting.** Replacing `news.go:108-112` with `return nil, err` reddens `TestUnreadableCacheIsAMissNotAFailure` at `news_test.go:285`. `cachingFeed.items` went from a comment-documented policy to 100% covered, and `failingStore` — which had sat unused — is now the fixture that drives it.
- **The cache asserts on the fake's counter** (`news_test.go:31,50,68,88`). "Served from cache" and "re-fetched and got the same answer" are indistinguishable from returned items; `f.fetches(word)` is the only oracle that tells them apart, and `fakeFeed` serving *bytes* keeps `parseRSS` in both the production and test paths.
- **`containsWord` delegating to `highlightSpans`** (`usage.go:63`) with `TestContainsWordAgreesWithTheHighlighter` as the pin remains the best decision in the issue — ARCH-DRY where a divergence would be visible to the learner as one word in two states on one screen.
- **The live conformance check asserts the licence terms** (`news_conformance_test.go:71`). Making "personal, non-commercial use" falsifiable rather than a comment is the right instinct, and the check is correctly behind a build tag.

## 2. Critical findings

None. BR-13 is disposed `addressed`.

## 3. Important findings

Both are dispositions of prior findings, not new ones — see the block below.

**BR-15 — not addressed; the fix is unreachable.** `usage.go:153-159` adds `warn io.Writer` to `bothSources`, and `usage.go:180-188` adds `warnOnce`. Production never sets the field:

```
openStore path:    bothSources.warn == nil ? true      # main.go:215
sessionUsage path: bothSources.warn == nil ? true      # main.go:180
```

`warnTo` (`vocab.go:165`) returns immediately on a nil writer, so the production behaviour is byte-identical to the silent version the finding was raised against. The only non-nil writer is the test's own at `news_test.go:147`. `openStore` already has `warn io.Writer` in its signature and passes it to `store.NewYAML` one line above — threading it is two identifiers; `sessionUsage(clk)` needs the parameter added. Separately, the "once per session" claim is at zero coverage (`usage.go:183-185`): no test calls `Usages` twice on a failing feed, so removing the `warned` guard is invisible.

**BR-14 — not addressed; two rows of the finding's own enumeration are open.** (a) The quoted-word query. `news.go:59` builds `q="<word>"`; I replaced `url.QueryEscape(`"`+word+`"`)` with `url.QueryEscape(word)` and the entire suite stayed green. All three subtests of `TestHTTPFeedErrors` (`news_test.go:342,353,366`) declare the handler as `func(w http.ResponseWriter, _ *http.Request)` — the `base` seam was added expressly so the URL could be driven locally, and nothing inspects it. One handler capturing `r.URL.RawQuery` closes it. (b) `usage.go:107-109` (`withoutAttribution`'s empty-publisher guard) is still at zero coverage — the 7th row the finding tabulated.

## 4. Minor findings

- **New:** `cmd/define/store/yaml.go:356-361` — `newsFile` was spliced in directly under `Forget`'s doc comment with no blank line, so `YAML.Forget` (`:417`) now has **no** documentation and `newsFile` carries a doc describing a different function. `go doc store.YAML.Forget` prints only the signature; at the base SHA the comment was attached at `:353-358`.
- **New:** neither `YAML.Forget` (`yaml.go:417`) nor `Mem.Forget` (`mem.go:138`) removes `usage/<slug>.yaml`. `Store.Forget`'s doc deliberately enumerates what it leaves behind ("does NOT remove events: … the log is history") and that enumeration is now incomplete for a record that is a *cache*, not history — so the decision is neither stated nor pinned by a `storetest` row.
- BR-5 through BR-9 and BR-16 through BR-19 are all unchanged in the tree; see dispositions. BR-5 re-verified by mutation against the full suite; BR-8 re-verified (`containsWord` is `false` for `e.g.`, `9/11`, `rock 'n' roll` against text containing them verbatim).
- `cmd/define/store/storetest/suite.go:381` — stray blank line before the closing brace, unchanged since round 3.
- `Mem.NewsItems` (`mem.go:112`) does not guard `k == ""` while `Mem.SetNewsItems` and both `YAML` methods do. Harmless (a map miss), but the four methods should agree.

## 5. Test coverage notes

92.1% of statements in `cmd/define`; only three blocks in the new files are uncovered, and two of them are findings above (`usage.go:107-109`, `usage.go:183-185`). The third, `news.go:62-64` (`NewRequestWithContext` error), is near-unreachable and fine to leave. Mutation results this round, every one restored with `git checkout HEAD --` on a committed tree and the tree verified clean afterwards: **dead** — inverting the unreadable-cache policy. **Survived** — dropping the quotes from the feed query; `entryUsages` restricted to `e.Blocks[:1]`. **Never fired** — the live-network probe panic, which is the measurement that closes BR-13. The `storetest` rows for `NewsItems`/`SetNewsItems` are the strongest new tests in the diff: never-fetched vs fetched-empty is pinned as a contract both `Mem` and `YAML` must answer, which is the reason the method returns a `time.Time` at all.

## 6. Architectural notes for upcoming work

- **ARCH-MOCK — pass.** This is the round it earned it. Production flow and test flow now share the `feed` boundary, `httpFeed` has hermetic tests via `httptest`, the fake is stateful and byte-shaped, and the live conformance check is tagged out of the default suite. The one residual is that the *request* the fake and the server receive is never asserted, which is the BR-14 residual above.
- **ARCH-PURE — pass.** `parseRSS`, `parsePubDate`, `containsWord`, `usagesFrom`, `withoutAttribution`, `entryUsages` are all pure and all tested with no doubles; IO is confined to `httpFeed`. `bothSources` → `cachingFeed` → `feed` mirrors `fetch.go` as the plan promised.
- **ARCH-DRY — flag (BR-6).** `renderSpans` (`usage_test.go:74`) is still byte-for-byte `marked` (`highlight_test.go:110`), and the single call site at `:64` wraps the result back in `string(...)`. Four rounds old, in the milestone whose thesis is that the matcher must not be reimplemented.
- **ARCH-PURPOSE — flag, and this is the round's theme.** BR-15 is the pattern inverted: the *shape* of the fix landed (field, method, mutex, test) while the purpose — production stops being silent — did not. A field set at zero production call sites passes every suite while doing nothing, which is exactly the failure mode the gate protocol names. BR-14 is the same axis on the enumeration it wrote itself: 6 of 7 rows swept, and the one it called out in prose as load-bearing left open. And nine Minors named in rounds 1 and 3, four of them one-line fixes, are crossing the final gate untouched.
- For #10: `Usage.At` can legitimately be the zero time (unreadable pubDate — the parser hands you "we do not know when" rather than a guess). Decide at design time whether zero-time usages sort last or are dropped. And note that the seam currently degrades silently, so #10 cannot distinguish "no news" from "feed broken" unless BR-15 is actually threaded.

## 7. Plan revision recommendations

`workshop/plans/000009-vocab-news-plan.md` has **0 ticked / 26 unticked** checkboxes at a whole-issue close, and a `## Revisions` section containing only the plan-quality round-1 entry. The Integration table at `:88` still lists `newsUsageSource | cmd/define/news.go | new` — an entity that exists nowhere in the tree; that is a core-concepts-table-vs-code contradiction at the final gate. Add:

```
### 2026-08-26 — as built (M1–M3)

- `Usage` ships `{Text, Source, Publisher, URL, At}`, not `{…Title…}`.
- `store.NewsItem` ships `{Title, URL, Source, At}`; the prose omits `Source`,
  which `withoutAttribution` depends on.
- `entryUsages(e Entry, word string)`, not `entryUsages(e Entry)`.
- Delete the Integration table's `newsUsageSource` row — it was never built;
  `bothSources` wraps `cachingFeed` directly.
- `UsageSource.Usages` returns `[]Usage` with NO error (BR-15): the dictionary
  half always answers, so there is nothing to fail. The prose still says
  `([]Usage, error)`.
- The fuzz property is not "title is a substring of the input". Substring,
  subsequence and the count-bound were all retired for failing on correct
  parsing; record the RULE, and record that the Done-when's "never returns an
  item it did not find in the input" is pinned by the exact-count and
  exact-title assertions in TestParseRSSOverACapturedFeed and
  TestParseRSSDecisions, not by FuzzParseRSS (BR-18).
- Task 2 Step 5's "12–99 RANGE" is impossible against a committed 13-item
  fixture; the exact-10 assertion the code ships is correct.
- Tick Task 1 Steps 1–7, Task 2 Steps 1–7, Task 3 Steps 1–6, Task 4 Steps 1–9,
  Task 5 Steps 1–3, Task 6 Steps 1–2.
```

`workshop/issues/000009-vocab-news.md:57-58` needs the same: the M2 row is `[x]` while naming `Store.Usages` (shipped as `NewsItems`/`SetNewsItems`) and the `NewsSource` seam (shipped as `feed` + `UsageSource`), and the M3 row is `[x]` while naming `/usage`, which plan-quality PQ-4 deliberately dropped and whose disposition already said the row "should be tidied to match". A ticked box claiming a subcommand that does not exist is the worst of the three.

```findings
dispose:
  - id: BR-13
    disposition: addressed
    note: |
      Verified by probe: panicking in Fetch when base=="" never fires over the full suite — zero live requests.
  - id: BR-14
    disposition: not-addressed
    note: |
      news.go's branches are pinned (verified by reverting), but the quoted-word query still drops green under mutation and usage.go:107-109 is at zero coverage — both named in the finding's own text.
  - id: BR-15
    disposition: not-addressed
    note: |
      bothSources.warn is nil at BOTH production sites (main.go:180, main.go:215, probed); warnTo no-ops on nil, so production degrades exactly as silently as before.
  - id: BR-5
    disposition: not-addressed
    note: |
      Re-verified by mutation: e.Blocks[:1] leaves the entire cmd/define suite green.
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
      Re-verified false for "e.g.", "9/11", "rock 'n' roll" against text containing them verbatim; still undocumented in usage.go and the atlas.
  - id: BR-9
    disposition: not-addressed
    note: |
      Measured at the close: 0 of 26 plan checkboxes ticked, newsUsageSource still at plan line 88, and the issue's ticked M2/M3 rows still name Store.Usages, NewsSource and /usage.
  - id: BR-16
    disposition: not-addressed
    note: |
      news.go:133-138 unchanged — both branches still return items, nil.
  - id: BR-17
    disposition: not-addressed
    note: |
      rss.go:16-38 unchanged; well-formed non-RSS XML still returns (empty, nil) and is cacheable as "no news".
  - id: BR-18
    disposition: not-addressed
    note: |
      rss_test.go property unchanged and still unreachable; neither the plan nor the Log records what actually pins the Done-when clause.
  - id: BR-19
    disposition: not-addressed
    note: |
      atlas/define.md:210-220 layout block still lists only words/, events/ and user-model.md.
findings:
  - id: new
    severity: Minor
    family: doc-comment-detached-from-declaration
    title: |
      newsFile was spliced under Forget's doc comment, so YAML.Forget lost its documentation
    detail: |
      cmd/define/store/yaml.go:356-361. The four-line comment documenting Forget ("Forget removes
      one word file... Filename derivation goes through wordFileName...") is immediately followed,
      with no blank line, by the newsFile type declaration — so godoc attaches it to newsFile, which
      it does not describe, and func (y *YAML) Forget at :417 has no doc at all. Verified:
      `go doc store.YAML.Forget` prints only the signature, and `git show 361136b:cmd/define/store/yaml.go`
      has the comment attached at :353-358. A regression introduced by this window. Fix: move the
      Forget doc back above :417 and give newsFile its own.
  - id: new
    severity: Minor
    family: delete-scope-unstated-for-new-record
    title: |
      Forget does not remove usage/<slug>.yaml, and neither the doc nor storetest says so
    detail: |
      cmd/define/store/yaml.go:417 removes only words/<slug>.yaml; cmd/define/store/mem.go:138 only
      deletes from m.words. Store.Forget's doc deliberately enumerates what it leaves behind ("does
      NOT remove events: the deck is a working set, the log is history"), and this window added a
      per-word on-disk record that the enumeration does not mention. Unlike events, the news cache is
      derived and refetchable, so the argument for keeping it is weaker — a user who forgets a word
      still has its headlines on disk. The rule: adding a per-word record obliges the delete verb to
      declare whether it is removed, and a storetest row to pin the answer. Decide either way, but say
      it in the interface doc and pin it in the suite.
```

---

## Re-review — 2026-08-26T22:00:09-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 9 — news seam: Google News RSS client with a stateful fake |
| repo | tools |
| issue file | workshop/issues/000009-vocab-news.md |
| boundary | whole-issue close |
| milestone | — |
| window | 361136b3e46f21e67dd307a6c5ed31c62b70148f..cce956fb9bfa180ece877c2a96f5e1cdab3855b6 |
| command | sdlc close --issue 9 |
| reviewer | claude |
| timestamp | 2026-08-26T22:00:09-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both Importants from round 4 are genuinely fixed, and I verified each by reverting rather than by reading. `bothSources.warn` is now non-nil at both production sites — unwiring `openStore` reddens `news_test.go:439/with a durable store`, unwiring `sessionUsage` reddens the `no-capture` subtest. BR-14's two residuals are closed: dropping the quotes from the feed query now reddens `TestHTTPFeedErrors/the word is quoted in the query`, and removing `withoutAttribution`'s empty-publisher guard reddens `TestNoPublisherLeavesTheTitleAlone`. `go build`, `go vet` (untagged and `-tags conformance`) and `go test ./...` all pass; coverage is 92.1% in `cmd/define` with exactly two uncovered blocks in the new files. And the ARCH-MOCK probe holds — patching `Fetch` to panic when `base == ""` never fires over a full run, so the default suite makes zero live requests. What keeps this off SHIP is one thing I measured and did not expect: this window adds a **third runtime directory**, `usage/`, and `git check-ignore` says `cmd/define/usage/ephemeral.yaml` is **not** ignored, while `cmd/define/words/x.yaml` matches `.gitignore:29`. Both repo guards hardcode `p == "words" || p == "events"`, so neither sees it either. The `.gitignore` comment in this repo says in as many words that this class "has cost a review round" three times — this is the fourth, and it is a one-line fix plus two enumerations. Behind it, thirteen Minors from rounds 1, 3 and 4 are crossing the final gate untouched, and the durable record is the worst of them: **0 of 27 plan checkboxes ticked**, a Core-concepts row naming `newsUsageSource` which exists nowhere, and ticked issue rows naming `Store.Usages`, `NewsSource` and `/usage` — none of which were built.

## 1. Strengths

- **BR-15's fix is the right shape, not just the right field.** `TestProductionWiringGivesTheUsageSourceItsWarnWriter` (`cmd/define/news_test.go:420`) builds `deps` the way a process does, over both `options{}` and `options{noCapture: true}`, and then proves the writer is *the caller's* by writing a probe through it. That is the exact test form the wiring-hop family needed, and it fails when either site is unwired — I checked both.
- **The quoted-word query is now pinned against a captured request** (`news_test.go:366`). The `base` seam existed for three rounds before anything looked at what was actually asked for; `r.URL.Query().Get("q")` is what a fake structurally cannot tell you, and the comment says so.
- **`cachingFeed.items` is at 100% and every branch is a behaviour test.** `TestUnreadableCacheIsAMissNotAFailure` drives `failingStore` — a fixture that had sat unused in the tree — and `TestUnparseableBodyFallsBackOrErrors` splits stale-fallback from hard error. The fake asserting on its own counter (`f.fetches(word)`) remains the only oracle that distinguishes "served from cache" from "re-fetched the same answer."
- **`containsWord` delegating to `highlightSpans`** (`usage.go:63`) with `TestContainsWordAgreesWithTheHighlighter` as the pin is still the best decision in the issue — the divergence it prevents is visible to the learner as one word in two states on one screen.
- **`workshop/lessons.md` earned its keep this round.** The three new entries — the wiring-hop table, "a degrading fallback hides a test that reaches the network", and the fuzz-property rule — each name the class rather than the site, with the measurement attached.

## 2. Critical findings

None.

## 3. Important findings

**`usage/` is a new runtime directory in the working directory, and none of the places that enumerate runtime directories were swept.** Measured:

```
git check-ignore -v cmd/define/usage/ephemeral.yaml   → exit 1 (NOT ignored)
git check-ignore -v cmd/define/words/x.yaml           → .gitignore:29  words/
```

`YAML.SetNewsItems` (`cmd/define/store/yaml.go:407`) does `os.MkdirAll(y.usageDir())` under `y.dir`, which in production is `os.Getwd()` — the same directory `words/` and `events/` land in. The fact "these are the runtime directories" is currently written in five places, and this window edited none of them:

| site | lists `usage/`? |
|---|---|
| `.gitignore:29-30` (`words/`, `events/`) | no |
| `repo_guard_test.go:207` `TestNoTrackedRuntimeState` | no — `p == "words" \|\| p == "events"` |
| `repo_guard_test.go:250` `TestNoRuntimeStateInHistory` | no — same literal pair |
| `README.md:142-145` on-disk layout | no |
| `atlas/define.md:215-219` on-disk layout | no (this is BR-19) |

It is unreachable *today* only because nothing calls `Usages` — which is precisely the situation #10 ends. The `.gitignore` comment already records what happens then: "`go test` runs with cwd set to the PACKAGE directory … so a deck appeared at `cmd/define/words/` and `cmd/define/events/`, which no pattern matched, and `git add -A` committed the developer's own vocabulary. … Third time this class has cost a review round."

**This is the class BR-19 and BR-21 are also instances of** — the atlas layout block and `Store.Forget`'s enumeration are two more sites that enumerate on-disk record types and were not swept for the record type this window added. Per ARCH-PURPOSE, the deliverable is the enumeration, not the site: sweep all six in one pass. ARCH-DRY applies too — `{words, events}` written five times is why a sixth reader was easy to miss; the two guards should share one list.

## 4. Minor findings

- **`cmd/define/main.go:179` — `sessionUsage` was spliced under `openStore`'s doc comment, so `openStore` lost its documentation.** `go doc -u ./cmd/define sessionUsage` prints "*openStore builds the store-backed dependencies over the WORKING DIRECTORY*" as its first line; `git show 361136b:cmd/define/main.go:154` has that comment attached to `openStore`. Introduced by `06312f4`. **This is the 2nd finding in family `doc-comment-detached-from-declaration`** (BR-20 named `yaml.go:366`). Do not fix this instance alone — state the rule and sweep it. The rule: *a declaration inserted into an existing file must not be placed inside another declaration's doc comment; a doc comment whose first word is not the declaration's name is the detector.* I ran that detector mechanically (go/ast, first doc word vs. declaration name, all four packages) and it takes about thirty seconds: exactly two production instances exist in this window — `store/yaml.go:366` and `main.go:179` — and every other hit is a legitimate prose doc on a const block. The enumeration is complete; sweep both.
- **`cmd/define/usage.go:183` — the "once per session" guard is at zero coverage, so removing it is invisible.** The comment at `:171` cites its own model: "Once per session, like storeCapturer's write warning." That precedent *is* pinned — `TestStoreCapturerDegradesOnWriteFailure` (`capture_test.go:60`) calls `Capture` five times and asserts exactly one `define:` line. **This is the 2nd finding in family `documented-degradation-unpinned`** (BR-14 was the first). The rule stands as BR-14 wrote it — *a comment stating a degradation policy needs one test that reddens when the policy is removed* — and the measured prevalence is now stateable: two warn-once sites in `cmd/define`, one pinned, and the unpinned one is the new code that names the pinned one as its model. Copy the precedent's test, six lines.
- **The atlas records the seam as it was two rounds ago.** `atlas/define.md:1080-1082` says "a feed outage degrades to the dictionary rather than propagating" and stops there — no mention that it now warns once per session, and no mention that `UsageSource.Usages` returns **no error**, which is the single most consumer-relevant fact for #10. **This is the 2nd finding in family `docs-describe-unshipped-milestone`** — the inverse direction of the same rule (*doc prose at a boundary describes what that boundary shipped*, no more and no less). Don't patch one sentence: the enumeration is the atlas's `Usage sources` section read against the three things rounds 4-5 changed — the warn signal, the dropped error return, and `usage/` in the layout block (BR-19). One pass.
- BR-5 through BR-9 and BR-16 through BR-21 are all unchanged in the tree; see dispositions. BR-5 re-verified by mutation against the full 94s suite; BR-8 re-probed (`containsWord` is `false` for `e.g.`, `9/11`, `rock 'n' roll` against text containing them verbatim).

## 5. Test coverage notes

92.1% of statements in `cmd/define`. Exactly two blocks in the three new files are uncovered: `news.go:62-64` (`NewRequestWithContext` error — near-unreachable, fine to leave) and `usage.go:183-185` (the `warned` guard, a finding above). Mutation results this round, all on a committed tree with `git checkout HEAD --` restores and `git status` verified clean afterwards — **dead:** unwire `warn` at `openStore`; unwire `warn` at `sessionUsage`; drop the quotes from the feed query; remove `withoutAttribution`'s empty-publisher guard; invert the unreadable-cache policy. **survived:** `entryUsages` restricted to `e.Blocks[:1]` (BR-5, re-confirmed against the full suite). **never fired:** the live-network panic probe. The storetest conformance rows for `NewsItems`/`SetNewsItems` remain the strongest tests in the diff — never-fetched vs fetched-empty is pinned as a contract both `Mem` and `YAML` must answer, which is the reason the method returns a `time.Time` at all.

## 6. Architectural notes for upcoming work

- **ARCH-MOCK — pass.** Production and test share the `feed` boundary; the fake is stateful and byte-shaped so `parseRSS` stays in both paths; `httpFeed` has hermetic `httptest` tests including a captured request; live conformance is tagged out of the default suite and asserts the licence terms. Verified by probe, not by reading.
- **ARCH-PURE — pass.** `parseRSS`, `parsePubDate`, `containsWord`, `usagesFrom`, `withoutAttribution`, `entryUsages` are all pure and all tested with no doubles at all. IO is confined to `httpFeed` and the store; `bothSources → cachingFeed → feed` mirrors `fetch.go` as the plan promised.
- **ARCH-DRY — flag, twice.** BR-6 is four rounds old: `renderSpans` (`usage_test.go:74`) is still byte-for-byte `marked` (`highlight_test.go:110`), and its one call site wraps the result back in `string(...)` — in the milestone whose thesis is that the matcher must not be reimplemented. The second is new and structural: the runtime-directory list is written five times, which is why a sixth reader was easy to miss (Important above).
- **ARCH-PURPOSE — flag, and it is the round's shape.** BR-15 and BR-14 were both answered well *at the sites the findings named*. But two of this round's three Minors are second instances of families whose enumerations were never written, and both enumerations are mechanically cheap — a thirty-second AST sweep for the doc detachments, a two-line grep for the warn-once sites. The Important finding is the same axis at the record level: BR-19 and BR-21 each named one site of "the enumeration of on-disk record types was not swept," and the third site (`.gitignore` + both repo guards) was found by asking what else enumerates them rather than by a new symptom appearing.
- For #10: `Usages` returns `[]Usage` with **no error** and warns once per session on a feed failure — so the consumer cannot distinguish "no news for this word" from "the feed is broken" except by reading stderr. That is a deliberate contract, but it is currently recorded only in `usage.go`'s doc comment and the issue Log, not the atlas. Also `Usage.At` can legitimately be the zero time (unreadable pubDate); decide at design time whether zero-time usages sort last or are dropped.

## 7. Plan revision recommendations

`workshop/plans/000009-vocab-news-plan.md` is at **0 ticked / 27 unticked** at a whole-issue close, with a `## Revisions` section containing only the plan-quality round-1 entry. Line 88 still lists `newsUsageSource | cmd/define/news.go | new` — a Core-concepts-table row for an entity that exists nowhere in the tree, which is a table-vs-code contradiction at the final gate. Add:

```
### 2026-08-26 — as built (M1–M3)

- `Usage` ships `{Text, Source, Publisher, URL, At}`, not `{…Title…}` (plan:36).
- `store.NewsItem` ships `{Title, URL, Source, At}`; plan:32 omits `Source`,
  which `withoutAttribution` depends on.
- `entryUsages(e Entry, word string)`, not `entryUsages(e Entry)` (plan:74).
- Delete the Integration table's `newsUsageSource` row (plan:88) — never built;
  `bothSources` wraps `cachingFeed` directly.
- `UsageSource.Usages` returns `[]Usage` with NO error (plan:97 still says
  `([]Usage, error)`). The dictionary half always answers, so there is nothing
  to fail; a feed failure warns once per session on the warn writer.
- The fuzz property is not "a title is a substring of the input" (plan:80,
  plan:171). Substring, subsequence and the count-bound were all retired for
  failing on correct parsing. Record the RULE, and record that the Done-when's
  "never returns an item it did not find in the input" is pinned by the
  exact-count and exact-title assertions in TestParseRSSOverACapturedFeed and
  TestParseRSSDecisions, not by FuzzParseRSS (BR-18).
- Task 2 Step 5's "12–99 RANGE" (plan:195) is impossible against a committed
  13-item fixture; the exact-10 assertion the code ships is correct.
- Task 6 Step 2's "No README line" still holds — nothing calls Usages — but
  `usage/` needs a .gitignore line and a repo-guard row regardless.
- Tick Task 1 Steps 1–7, Task 2 Steps 1–7, Task 3 Steps 1–6, Task 4 Steps 1–9,
  Task 5 Steps 1–3, Task 6 Steps 1–2.
```

`workshop/issues/000009-vocab-news.md:57-58` needs the same treatment: the M2 row is `[x]` while naming `Store.Usages` (shipped as `NewsItems`/`SetNewsItems`) and the `NewsSource` seam (shipped as `feed` + `UsageSource`), and the M3 row is `[x]` while naming `/usage`, which plan-quality PQ-4 deliberately dropped and whose own disposition said the row "should be tidied to match." Separately, the `## Log` entry for the M1 boundary still reads "three findings, all addressed" — round 1 raised nine, six of which are still open at this gate.

```findings
dispose:
  - id: BR-14
    disposition: addressed
    note: |
      Verified by reverting: dropping the query quotes reddens news_test.go:366, and removing the empty-publisher guard reddens TestNoPublisherLeavesTheTitleAlone.
  - id: BR-15
    disposition: addressed
    note: |
      Verified by reverting at BOTH production sites — unwiring main.go:215 or main.go:180 reddens news_test.go:439; the vestigial error return is gone from the interface.
  - id: BR-5
    disposition: not-addressed
    note: |
      Re-verified by mutation against the full 94s suite: e.Blocks[:1] leaves all of cmd/define green.
  - id: BR-6
    disposition: not-addressed
    note: |
      renderSpans (usage_test.go:74) is still byte-identical to marked (highlight_test.go:110).
  - id: BR-7
    disposition: not-addressed
    note: |
      store/news.go:15 still names the publisher field Source, one line from Usage.Source.
  - id: BR-9
    disposition: not-addressed
    note: |
      Measured at the close - 0 of 27 plan checkboxes ticked, newsUsageSource still at plan:88, plan:97 still says Usages returns an error, the issue's ticked M2/M3 rows still name Store.Usages/NewsSource//usage, and the Log still says M1 round 1 had "three findings" when it raised nine.
  - id: BR-8
    disposition: not-addressed
    note: |
      Re-probed at HEAD - containsWord is false for "e.g.", "9/11" and "rock 'n' roll" against text containing them verbatim; still undocumented in usage.go and the atlas.
  - id: BR-16
    disposition: not-addressed
    note: |
      news.go:133-138 unchanged - both branches still return items, nil, so the guard cannot be mutated.
  - id: BR-17
    disposition: not-addressed
    note: |
      rss.go:16-38 unchanged; the anonymous struct has no XMLName, so any well-formed non-RSS XML still returns (empty, nil).
  - id: BR-18
    disposition: not-addressed
    note: |
      rss_test.go property still unreachable, and neither the plan nor the Log records what actually pins the Done-when clause (grepped both for TestParseRSSOverACapturedFeed - no hits).
  - id: BR-19
    disposition: not-addressed
    note: |
      atlas/define.md:215-219 layout block still lists only words/, events/ and user-model.md.
  - id: BR-20
    disposition: not-addressed
    note: |
      store/yaml.go:366 unchanged; a mechanical AST sweep found a SECOND instance in this window at main.go:179, raised below.
  - id: BR-21
    disposition: not-addressed
    note: |
      yaml.go:417 still removes only words/; Store.Forget's doc enumeration still omits the new per-word record and storetest has no row for it.
findings:
  - id: new
    severity: Important
    family: record-type-enumeration-unswept
    title: |
      usage/ is a new runtime directory in the working directory and is in neither .gitignore nor either repo guard
    detail: |
      Measured - `git check-ignore -v cmd/define/usage/ephemeral.yaml` exits 1 (NOT ignored) while
      `cmd/define/words/x.yaml` matches .gitignore:29. YAML.SetNewsItems (store/yaml.go:407)
      MkdirAll's usage/ under y.dir, which in production is os.Getwd() - the same directory
      words/ and events/ land in. Both guards hardcode the pair: repo_guard_test.go:207 and :250
      test `p == "words" || p == "events"`, so a committed usage/ file trips neither the index
      guard nor the history guard. Unreachable only until #10 wires the consumer this issue
      exists to serve; the .gitignore comment already records what happens then, verbatim -
      "go test runs with cwd set to the PACKAGE directory ... a deck appeared at
      cmd/define/words/ ... git add -A committed the developer's own vocabulary. Third time
      this class has cost a review round." This is the fourth. THE CLASS, and BR-19 and BR-21
      are instances of it: adding a new on-disk record type obliges sweeping every site that
      enumerates them. The enumeration, complete - .gitignore:29-30, repo_guard_test.go:207,
      repo_guard_test.go:250, README.md:142-145, atlas/define.md:215-219, and Store.Forget's
      doc plus a storetest row. Zero of the six were edited by this window. ARCH-DRY applies
      too: the two guards should read one shared list rather than repeating the literal pair.
  - id: new
    severity: Minor
    family: doc-comment-detached-from-declaration
    title: |
      sessionUsage was spliced under openStore's doc comment, so openStore lost its documentation
    detail: |
      cmd/define/main.go:165-183. `go doc -u ./cmd/define sessionUsage` prints "openStore builds
      the store-backed dependencies over the WORKING DIRECTORY." as its FIRST line, and
      `func openStore` at :183 prints with no doc at all; `git show 361136b:cmd/define/main.go`
      has that comment attached at :154. Introduced by 06312f4. THIS IS THE 2ND FINDING IN
      FAMILY doc-comment-detached-from-declaration - do NOT fix this instance alone. The rule:
      a declaration inserted into an existing file must never be placed inside another
      declaration's doc comment, and the detector is mechanical - a doc comment whose first
      word is not the declaration's own name. I ran that detector (go/ast walk over all four
      packages, first doc word vs. declaration name, ~30 seconds) and the enumeration is
      COMPLETE: exactly two production instances exist in this window - store/yaml.go:366
      (BR-20) and main.go:179 - every other hit being a legitimate prose doc on a const block.
      Sweep both, and consider keeping the detector.
  - id: new
    severity: Minor
    family: documented-degradation-unpinned
    title: |
      warnOnce's once-per-session guard is at zero coverage, so deleting it is invisible
    detail: |
      cmd/define/usage.go:183-185 is the only uncovered block in the new files besides the
      near-unreachable NewRequestWithContext error - proof from the profile that no test calls
      Usages twice on a failing feed, so removing the `warned` guard changes nothing any test
      observes. THIS IS THE 2ND FINDING IN FAMILY documented-degradation-unpinned. The rule is
      the one BR-14 already stated - a comment describing a degradation policy needs one test
      that reddens when the policy is removed - and what is new is the measured prevalence,
      which makes the enumeration trivially checkable: there are exactly two warn-once sites
      in cmd/define (capture.go:127 and usage.go:183), one is pinned and one is not, and the
      unpinned one's own comment at usage.go:171 names the pinned one as its model
      ("Once per session, like storeCapturer's write warning"). TestStoreCapturerDegradesOnWriteFailure
      (capture_test.go:60) loops five times and asserts exactly one "define:" line; copying
      that shape onto bothSources is six lines.
  - id: new
    severity: Minor
    family: docs-describe-unshipped-milestone
    title: |
      The atlas records the seam as it was two rounds ago — no warn signal, no dropped error return
    detail: |
      atlas/define.md:1080-1082 says "a feed outage degrades to the dictionary rather than
      propagating" and stops there. Two behaviours this window's last two rounds added are
      absent: the once-per-session warning on the warn writer (BR-15's whole remedy, and
      lessons.md now records "degrading SILENTLY is a different design from degrading"), and
      the fact that UsageSource.Usages returns NO error - the single most consumer-relevant
      fact for #10, currently recorded only in a code comment and the issue Log. THIS IS THE
      2ND FINDING IN FAMILY docs-describe-unshipped-milestone, in the inverse direction: the
      rule is that doc prose at a boundary describes what that boundary shipped, no more AND
      no less, and BR-3 was the over-description arm. Do not patch one sentence - the
      enumeration is the atlas's "Usage sources" section read against the three things rounds
      4-5 changed: the warn signal, the dropped error return, and usage/ missing from the
      layout block (BR-19). One pass closes all three.
```

---

## Re-review — 2026-08-26T22:19:23-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 9 — news seam: Google News RSS client with a stateful fake |
| repo | tools |
| issue file | workshop/issues/000009-vocab-news.md |
| boundary | whole-issue close |
| milestone | — |
| window | 361136b3e46f21e67dd307a6c5ed31c62b70148f..2ce38c2e285499027141e41dddab535eb4555b53 |
| command | sdlc close --issue 9 |
| reviewer | claude |
| timestamp | 2026-08-26T22:19:23-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I've completed the review — build, vet, full suite, coverage profile, four mutations (each restored via `git checkout HEAD --`, tree verified clean after every one), a live-network probe, a mechanical AST detector re-run, and two planted-file probes against the repo guards.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The one Important from round 5 is genuinely fixed and I verified it three ways rather than by reading: removing `usage/` from `.gitignore` reddens `TestGitignoreCoversRuntimeDirs`, planting a tracked `cmd/define/usage/probe.yaml` reddens `TestNoTrackedRuntimeState` through the new `isRuntimeDir`, and `git check-ignore` now matches `.gitignore:34`. `store.RuntimeDirs` is the right shape for that class — one source where the writer lives, plus a test closing the loop the compiler cannot. `go build`, `go vet` (untagged and `-tags conformance`) and `go test ./...` all pass; coverage is 92.1% in `cmd/define` with exactly two uncovered blocks in the new files; and the ARCH-MOCK property still holds — patching `Fetch` to panic when `base == ""` never fires over a full 94s run, so the default suite makes zero live requests. What keeps this off SHIP is that the enumeration the fix created stops one artifact short, and the artifact it misses is **live rather than latent**: `define --reflect` writes `user-model.md` into the working directory today, and I measured that a staged `cmd/define/user-model.md` is not ignored and passes **both** repo guards, because they match path *segments* and the third artifact is a file. `RuntimeDirs`' own doc says "every **directory** define writes" — that word is where it fell out, eight lines above `userModelFile`, which the same file calls "the third artifact in the directory". Behind it, three of the six sites BR-22 itself enumerated are still unswept, and fourteen Minors from rounds 1, 3, 4 and 5 are crossing the final gate untouched — the worst being the durable record: **0 of 26 plan checkboxes ticked**, and a Core-concepts table row for `newsUsageSource`, which exists nowhere in the tree.

## 1. Strengths

- **The `RuntimeDirs` fix is pinned in all three directions, and I checked each.** `cmd/define/store/yaml.go:41` is the single source; `repo_guard_test.go:267` makes both guards ask it; `repo_guard_test.go:283` reads the actual `.gitignore` and asserts the un-anchored form *and* rejects the anchored one — which encodes the subtlety (`go test` runs in the package directory) that cost three earlier rounds and had, until now, only ever been a comment.
- **The default suite is provably offline.** The panic probe on `httpFeed.Fetch` never fires. That is the strongest single property in this diff and it is the one that was false two rounds ago.
- **`cachingFeed.items` is 100% covered and every branch is a behaviour test**, driven by `failingStore` (a fixture that had sat unused in the tree) and by an `httptest` server that captures the request — so the quoted-word query at `news.go:59` is pinned against what was actually asked for, which a fake structurally cannot tell you.
- **`containsWord` delegating to `highlightSpans`** (`cmd/define/usage.go:63`) with `TestContainsWordAgreesWithTheHighlighter` as the pin is still the best decision in the issue: the divergence it prevents is visible to the learner as one word in two states on one screen.
- **The storetest conformance rows** (`storetest/suite.go:288`) pin never-fetched vs fetched-empty as a contract both `Mem` and `YAML` must answer — which is the reason `NewsItems` returns a `time.Time` at all, and the reason cache outcome 2 is expressible.

## 2. Critical findings

None.

## 3. Important findings

**`user-model.md` is a runtime artifact `--reflect` writes into the working directory, and it is in neither `.gitignore` nor either guard.** Measured, with the file staged exactly where `go test` and the pty suite would leave it:

```
git ls-files                    → cmd/define/user-model.md          (in the index)
git check-ignore -v …           → exit 1                            (NOT ignored)
TestNoTrackedRuntimeState       → ok
TestGitignoreCoversRuntimeDirs  → ok
```

Both guards split a path on `/` and compare **segments** against `RuntimeDirs`, so a file has no segment to match. `reflect.go:324` calls `SetUserModel` in production and `README.md:113` documents `--reflect` as a shipped feature, so unlike `usage/` — which BR-22 rated Important while it was still latent — this one is reachable today, and what would be committed is the learner's own inferred model.

**This is the 2nd finding in family `record-type-enumeration-unswept`.** Do not fix this instance alone. The rule: *the single-sourced list must enumerate every runtime **artifact** define writes into the working directory — files as well as directories — and both the ignore loop and the guards must be driven from it.* `RuntimeDirs`' doc says "every **directory**", and `yaml.go:46` calls `user-model.md` "the third artifact in the directory" eight lines below it; the noun mismatch is the whole defect. The measured prevalence is 3 artifacts, 2 covered. Shape of the fix: a `RuntimeFiles` (or a `RuntimeArtifacts` list carrying a dir/file kind), a `.gitignore` line the same test asserts, and a guard arm matching the final path element. One caveat worth stating so the fix does not break a passing test: `cmd/define/testdata/golden/user-model.md` is a legitimate tracked fixture, so the guard needs a `testdata/` exception.

## 4. Minor findings

- **New:** `cmd/define/store/yaml.go:43,44,52` consume the shared list by **position** — `RuntimeDirs[0]`, `[1]`, `[2]` — so the slice's order carries meaning that only a comment records. Honest measurement: reordering to `{"events", "words", "usage"}` *does* redden the suite, but incidentally, via path literals in `store/yaml_test.go:220`, not via anything that knows about the coupling; `TestGitignoreCoversRuntimeDirs` and both guards only check membership. Named constants used both in the accessors and in the slice literal cost one line and remove the coupling.
- BR-5 through BR-9 and BR-16 through BR-21 and BR-23 through BR-25 are all unchanged in the tree — see dispositions, each re-verified this round rather than carried forward.

## 5. Test coverage notes

92.1% of statements in `cmd/define`. Exactly two blocks in the three new files are uncovered: `news.go:62-64` (the `NewRequestWithContext` error — near-unreachable, fine to leave) and `usage.go:183-185` (the `warned` guard, which is BR-24: an uncovered block cannot be observed by any test, so deleting it is invisible by construction). Mutations this round, every one on a committed tree with `git checkout HEAD --` restores and `git status` verified clean afterwards — **dead:** removing `usage/` from `.gitignore`; reordering `RuntimeDirs`. **survived:** `entryUsages` restricted to `e.Blocks[:1]` (BR-5, re-confirmed against the full 94s suite). **never fired:** the live-network panic probe. Two planted-file probes: a tracked `usage/` file reddens the index guard (the BR-22 fix is reachable), a tracked `user-model.md` does not (the Important above). The `FuzzParsePubDate` target remains the right shape — it kills the guessing mutant on the seed corpus with no decoder standing between the property and the code.

## 6. Architectural notes for upcoming work

- **ARCH-MOCK — pass.** Production and test share the `feed` boundary; the fake serves bytes so `parseRSS` stays in both paths; `httpFeed` has hermetic `httptest` tests including a captured request; live conformance is tagged out of the default suite and asserts the licence terms. Verified by probe, not by reading.
- **ARCH-PURE — pass.** `parseRSS`, `parsePubDate`, `containsWord`, `usagesFrom`, `withoutAttribution`, `entryUsages` are all pure and all tested with no doubles at all; IO is confined to `httpFeed` and the store. `bothSources → cachingFeed → feed` mirrors `fetch.go` as the plan promised.
- **ARCH-DRY — flag, twice.** `store.RuntimeDirs` is ARCH-DRY done right and it is the best thing in this window — but it is consumed positionally (Minor above), and the fact it single-sources is "runtime **directories**" rather than "runtime **artifacts**", which is why a third one stayed outside it. Separately BR-6 is five rounds old: `renderSpans` (`usage_test.go:74`) is still byte-for-byte `marked` (`highlight_test.go:110`), in the milestone whose thesis is that the matcher must not be reimplemented.
- **ARCH-PURPOSE — flag, and it is the round's shape again.** BR-22's own text enumerated six sites and called the enumeration complete; three landed. The three that did not — `README.md:142-145`, `atlas/define.md:215-219` (BR-19), and `Store.Forget`'s doc plus a storetest row (BR-21) — are the same class, already have ids, and were named by the finding that was answered. And the class statement itself was one artifact short. Naming a class is not the same as writing the enumeration that closes it; the test for "did I fix the class" is whether a *new* instance of it can still be found, and this round found one in thirty seconds.
- **On the doc-detachment family (BR-23):** I re-ran the mechanical detector (go/ast, first doc word vs. declaration name, all packages). BR-23's claim about *this window* holds — exactly two production instances, `main.go:179` and `store/yaml.go:375`. But the same detector finds a **third outside the window** at `cmd/define/sgr.go:5`, where `sgrState`'s doc is attached to `maxOpenSGR` and `sgrState` has none (`go doc -u ./cmd/define maxOpenSGR` prints the `sgrState` prose). That is the argument for BR-23's parenthetical — *keep the detector* — rather than making two edits: 3 instances tree-wide, none of which any test can see.
- For #10: `Usages` returns `[]Usage` with **no error** and warns once per session on a feed failure, so the consumer cannot distinguish "no news for this word" from "the feed is broken" except by reading stderr. That is a deliberate contract recorded only in `usage.go`'s doc comment and the issue Log (BR-25). Also `Usage.At` can legitimately be the zero time; decide at design time whether zero-time usages sort last or are dropped.

## 7. Plan revision recommendations

`workshop/plans/000009-vocab-news-plan.md` is at **0 ticked / 26 unticked** at a whole-issue close, and its `## Revisions` section still contains only the plan-quality round-1 entry. Line 88 lists `newsUsageSource | cmd/define/news.go | new` — I grepped the whole tree for both `newsUsageSource` and `NewsSource`: neither exists in any `.go` file. That is a Core-concepts-table-vs-code contradiction at the final gate. Every other table row I checked *does* exist at its stated path. Add:

```
### 2026-08-26 — as built (M1–M3)

- `Usage` ships `{Text, Source, Publisher, URL, At}`, not `{…Title…}` (plan:36).
- `store.NewsItem` ships `{Title, URL, Source, At}`; plan:32 omits `Source`,
  which `withoutAttribution` depends on.
- `entryUsages(e Entry, word string)`, not `entryUsages(e Entry)` (plan:74).
- Delete the Integration table's `newsUsageSource` row (plan:88) — never built;
  `bothSources` wraps `cachingFeed` directly.
- `UsageSource.Usages` returns `[]Usage` with NO error (plan:97 still says
  `([]Usage, error)`); a feed failure warns once per session instead.
- The fuzz property is not "a title is a substring of the input" (plan:80,
  plan:171). Substring, subsequence and the count-bound were all retired for
  failing on correct parsing. Record the RULE, and record that the Done-when's
  "never returns an item it did not find in the input" is pinned by the
  exact-count and exact-title assertions in TestParseRSSOverACapturedFeed and
  TestParseRSSDecisions, not by FuzzParseRSS (BR-18).
- Task 2 Step 5's "12–99 RANGE" (plan:195) is impossible against a committed
  13-item fixture; the exact-10 assertion the code ships is correct.
- Task 6 Step 2's "No README line" still holds — nothing calls Usages — but the
  runtime-artifact enumeration (.gitignore + both guards) needed usage/ AND
  still needs user-model.md.
- Tick Task 1 Steps 1–7, Task 2 Steps 1–7, Task 3 Steps 1–6, Task 4 Steps 1–9,
  Task 5 Steps 1–3, Task 6 Steps 1–2.
```

`workshop/issues/000009-vocab-news.md:57-58` needs the same: the M2 row is `[x]` while naming `Store.Usages` (shipped as `NewsItems`/`SetNewsItems`) and the `NewsSource` seam (shipped as `feed` + `UsageSource`), and the M3 row is `[x]` while naming `/usage`, which plan-quality PQ-4 deliberately dropped and whose own disposition said the row "should be tidied to match." The `## Log` entry for the M1 boundary still reads "three findings, all addressed" — round 1 raised nine, and eleven findings from all rounds remain open at this gate.

```findings
dispose:
  - id: BR-22
    disposition: not-addressed
    note: |
      Three of the six sites it enumerated landed and are pinned (verified by reverting and by planting a tracked usage/ file); README.md:142-145, atlas/define.md:215-219 and Forget's doc plus a storetest row remain, and the list it created omits user-model.md — raised below.
  - id: BR-5
    disposition: not-addressed
    note: |
      Re-verified by mutation against the full 94s suite: e.Blocks[:1] leaves all of cmd/define green.
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
      Re-probed at HEAD - containsWord is false for "e.g.", "9/11" and "rock 'n' roll" against text containing them verbatim; still undocumented in usage.go and the atlas.
  - id: BR-9
    disposition: not-addressed
    note: |
      Measured at the close - 0 of 26 plan checkboxes ticked; grepped the whole tree for newsUsageSource and NewsSource, neither exists in any .go file, yet plan:88 still lists the row and the issue's ticked M2/M3 rows still name Store.Usages, NewsSource and /usage.
  - id: BR-16
    disposition: not-addressed
    note: |
      news.go:133-138 unchanged - both branches still return items, nil, so the guard cannot be mutated.
  - id: BR-17
    disposition: not-addressed
    note: |
      rss.go:16-38 unchanged; the anonymous struct still has no XMLName, so any well-formed non-RSS XML returns (empty, nil).
  - id: BR-18
    disposition: not-addressed
    note: |
      rss_test.go:181 property unchanged and still unreachable, and neither the plan nor the Log records what actually pins the Done-when clause.
  - id: BR-19
    disposition: not-addressed
    note: |
      atlas/define.md:216-219 layout block still lists only words/, events/ and user-model.md.
  - id: BR-20
    disposition: not-addressed
    note: |
      Re-verified - `go doc store.YAML.Forget` still prints only the signature; the comment is still attached to newsFile at yaml.go:375.
  - id: BR-21
    disposition: not-addressed
    note: |
      yaml.go:426 still removes only words/; Store.Forget's doc enumeration at store.go:45 still omits the new per-word record and storetest has no row for it.
  - id: BR-23
    disposition: not-addressed
    note: |
      Re-verified by go doc; I also re-ran the mechanical AST detector and it finds a THIRD instance outside this window at cmd/define/sgr.go:5 (sgrState's doc attached to maxOpenSGR) - which is the argument for keeping the detector rather than making two edits.
  - id: BR-24
    disposition: not-addressed
    note: |
      Coverage profile at HEAD - usage.go:183-185 is still one of only two uncovered blocks in the new files, so deleting the warned guard is unobservable by construction.
  - id: BR-25
    disposition: not-addressed
    note: |
      atlas/define.md:1080-1085 still says only "degrades to the dictionary rather than propagating" - no warn signal, no mention that Usages returns no error.
findings:
  - id: new
    severity: Important
    family: record-type-enumeration-unswept
    title: |
      user-model.md is a runtime artifact --reflect writes today, and it is in neither .gitignore nor either guard
    detail: |
      Measured by planting the file where go test and the pty suite would leave it:
      `cmd/define/user-model.md` staged appears in `git ls-files`, `git check-ignore -v`
      exits 1 (NOT ignored), and BOTH TestNoTrackedRuntimeState and
      TestGitignoreCoversRuntimeDirs PASS. The guards split a path on "/" and compare
      SEGMENTS against store.RuntimeDirs, so a file has no segment to match.
      YAML.SetUserModel (yaml.go:60) writes it into y.dir = os.Getwd(), reflect.go:324
      calls it in production, and README.md:113 documents `define --reflect` as shipped -
      so unlike usage/, which BR-22 rated Important while still latent, this one is
      reachable today and what gets committed is the learner's own inferred model.
      THIS IS THE 2ND FINDING IN FAMILY record-type-enumeration-unswept - do NOT fix this
      instance alone. The rule: the single-sourced list must enumerate every runtime
      ARTIFACT define writes into the working directory, files as well as directories, and
      both the ignore loop and the guards must be driven from it. RuntimeDirs' doc
      (yaml.go:34) says "every DIRECTORY define writes", and yaml.go:46 calls user-model.md
      "the third artifact in the directory" eight lines below it - the noun mismatch is the
      whole defect. Measured prevalence: 3 artifacts, 2 covered. Caveat for the fix -
      cmd/define/testdata/golden/user-model.md is a legitimate tracked fixture, so the
      guard arm needs a testdata/ exception or it reddens on a passing test.
  - id: new
    severity: Minor
    family: shared-list-indexed-by-position
    title: |
      RuntimeDirs is consumed by index position, so the slice's order silently carries meaning
    detail: |
      cmd/define/store/yaml.go:43, :44 and :52 read RuntimeDirs[0], [1] and [2] for
      wordsDir, eventsDir and usageDir, so reordering the literal at :41 changes which
      on-disk directory each accessor names. Honest measurement: the reorder DOES redden
      the suite - but incidentally, through path literals in store/yaml_test.go:220, not
      through anything that knows about the coupling; TestGitignoreCoversRuntimeDirs and
      both repo guards only check membership. Named constants used in both the accessors
      and the slice literal cost one line and remove the coupling entirely.
```
