# News Seam Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fetch, parse and cache real current sentences containing a word, from Google News RSS, behind a seam with a stateful fake — so `#10`'s authoring step has usage to work from and a review session never touches the network.

**Architecture:** A pure RSS parser over bytes; a `NewsSource` seam with an HTTP implementation, a store-backed cache, and a stateful fake; and a second usage source — NOAD's own example sentences — that costs no network at all. The cache lives in the store beside `words/` and `events/`, so "works offline" is a property of the directory rather than of a process that happens to be warm.

**Tech Stack:** Go 1.26, `encoding/xml`, the existing `store.Store` seam, the `AudioSource`/`Dictionary` seam conventions in `cmd/define`.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `store.NewsItem` | `cmd/define/store/news.go` | new |
| `Usage` | `cmd/define/usage.go` | new |
| `parseRSS` | `cmd/define/rss.go` | new |
| `usagesFrom` | `cmd/define/usage.go` | new |
| `containsWord` | `cmd/define/usage.go` | new |
| `entryUsages` | `cmd/define/usage.go` | new |

**The split between raw and derived is the load-bearing decision, and the first
draft of this plan got it wrong twice in one move.** It put `Usage` in
`package main` while giving `store.Store` a `Usages()` method — which cannot
compile — and it cached post-filter results, which the Spec explicitly does not
say. Both fix together:

- **`store.NewsItem{Title, URL, At}`** — one raw feed item, and the ONLY thing
  cached. In `package store` because the store returns it. It carries no notion of
  usage, filtering or provenance: the store's job is "what did the feed say", not
  "what is worth teaching".
- **`Usage{Text, Source, Title, URL, At}`** — a sentence worth showing, derived at
  READ time in `package main`. `Source` is `"news"` or `"noad"`.

Deriving on read is what the Spec's "raw feed items" buys: improving
`containsWord` improves every word already cached, with no re-fetch. Caching the
filtered result would freeze today's judgment onto disk.

- **`Usage`** — **Relationships:** N per word; assembled from two sources.
  **DRY rationale:** one shape for both, so `#10` gets "sentences for this word,
  tagged by provenance" rather than two parallel lists to join.
  **Future extensions:** a `Lang` field when `#18` makes the deck
  language-aware.

- **`parseRSS(data []byte) ([]store.NewsItem, error)`** — RSS 2.0 into items,
  pure over bytes, so the malformed-feed requirement is a table plus a fuzz
  target with no network. The only XML knowledge in the package.

- **`containsWord(text, word string) bool`** — whether a headline genuinely
  contains the word. The feed is queried with the word quoted and still returns
  items that do not contain it (measured 12–99 matching out of 41–100), so this
  filter is not optional.

  **It delegates to `highlightSpans` (`highlight.go`) with a one-word
  vocabulary**, rather than reimplementing matching over `wordRuns`. That is the
  whole matcher — `store.Key` normalisation for case, `phraseGap`/`phraseRunsJoin`
  for multi-word entries like `hot dog`, joiner trimming so `hot-dog` and
  `'ephemeral'` behave. Reusing only the tokenizer would re-commit #21's
  consolidation finding one level up, and the divergence would be **visible to the
  user**: a headline whose word renders green while `containsWord` rejects the
  same headline. One matcher, one answer.

  Inflections stay OUT, same call as #21: `ephemerally` does not match
  `ephemeral`, because a false positive puts a sentence about a different word in
  front of the learner.

- **`usagesFrom(items []store.NewsItem, word string) []Usage`** — filter and
  shape. Pure.

- **`entryUsages(e Entry) []Usage`** — NOAD's own examples in the same shape. No
  network, offline by construction, and the reason a word is not taught only
  through this week's news cycle.

**Test surface.** `rss_test.go`, `usage_test.go`, colocated, no IO.
`FuzzParseRSS`: never panic, and never return an item whose title is not a
substring of the input — a parser that invents content is worse than one that
finds none.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `UsageSource` | `cmd/define/usage.go` | new | both sources |
| `newsUsageSource` | `cmd/define/news.go` | new | the feed + the cache |
| `httpFeed` | `cmd/define/news.go` | new | `net/http` |
| `cachingFeed` | `cmd/define/news.go` | new | `feed` + `store.Store` |
| `bothSources` | `cmd/define/usage.go` | new | news + NOAD |
| `fakeFeed` | `cmd/define/news_fake_test.go` | new | nothing (the double) |
| `Store.NewsItems` / `SetNewsItems` | `cmd/define/store/store.go` | modified | the cache on disk |
| `failingStore` | `cmd/define/history_store_test.go` | modified | (needs the new methods) |
| `deps.usage` | `cmd/define/main.go` | modified | dependency wiring |

- **`UsageSource`** — `Usages(ctx, word string, e Entry) ([]Usage, error)`.
  **The seam is the Done-when's promise**: "NOAD's own example sentences are
  available through the same seam". A bare `entryUsages` helper beside a
  news-only seam would have left the consumer asking two shapes, which is not a
  seam. `bothSources` merges them, news first, NOAD always — so a feed outage
  degrades to the dictionary rather than to nothing.
  - **Injected into:** `#10`'s authoring step. This issue delivers the seam; see
    the note on `/usage` below for why nothing user-facing consumes it yet.

- **`cachingFeed`** — the three outcomes, stated because the first draft modelled
  two:
  1. **Fetch succeeds with items** → cache, serve.
  2. **Fetch succeeds with ZERO items** → cache it, with its timestamp. A real
     answer, not a failure: some words are simply not in the news. Not caching it
     would re-fetch on every call forever for exactly the words the feed is
     worst at.
  3. **Fetch FAILS** → do not cache. A network blip must not become a permanent
     empty answer.

  Because (2) is cached, entries carry `FetchedAt` and a `cacheTTL`: a stale
  entry is re-fetched, and **a failed re-fetch falls back to the stale copy**.
  That is what keeps "works offline" true while letting a word that had no news
  last week pick some up this week.

- **`Store.NewsItems(key) ([]NewsItem, time.Time, error)` / `SetNewsItems(key, items, at)`**
  — `usage/<slug>.yaml`, beside `words/` and `events/`.
  - **Record shape, stated because the plan gate caught the first draft
    describing the wrong one:** this is a WHOLE-FILE record like `words/`, written
    via the existing `writeBytesAtomic` and therefore untearable. The failure mode
    to test is a corrupt or truncated file from outside, and the existing
    discipline for that shape is **warn and skip the whole file**
    (`yaml.go`) — not the terminator-plus-completeness rule, which belongs to the
    append-only day log in `event.go`.
  - **Conformance:** both methods get rows in `storetest.Suite`, so `Mem` and
    `YAML` answer the same contract.

- **`fakeFeed`** — serves canned RSS **bytes**, not parsed items: a fake returning
  the parsed type would skip `parseRSS`, and the parser is where the risk is.
  Counts fetches per word and can be scripted to fail, so "served from cache" is
  asserted on its counter rather than inferred from output.

- **Live conformance** (`//go:build conformance`) — the feed still parses and
  still yields a coverage FLOOR. On-demand; it needs network, which does not
  belong in merge-check.

**No `/usage` command, and the first draft was wrong to call it cheap.** It said
the command "costs one row in the command table". It does not: `commandCtx`
(`command.go`) carries no `context.Context` and no usage dependency, so a command
that fetches means widening the context every command shares — a change to the
command contract, made for a debug affordance, ahead of the real consumer. `#10`
needs that wiring anyway and can do it with the context it actually requires. In
this window the thing is exercised by its tests and by the live conformance check,
which prints what it fetched.

---

## Chunk 1: M1 — the parser and the usage shape

### Task 1: `parseRSS`

**Files:**
- Create: `cmd/define/rss.go`, `cmd/define/store/news.go` (`NewsItem`)
- Test: `cmd/define/rss_test.go`
- Fixture: `cmd/define/testdata/news/ephemeral.rss` — a real captured feed

- [ ] **Step 1: Capture a real feed as the fixture.** `curl` the URL for
      `ephemeral`, commit it trimmed. A captured artifact, not an invented one:
      same rule as #16's streaming capture — you cannot fake the shape of
      something you have not looked at.

- [ ] **Step 2: Write the failing tests.** Strategy: one table whose rows are the
      parser's DECISIONS (an item missing `pubDate` parses with a zero time
      rather than failing; CDATA is unwrapped; an unparseable date does not fail
      the feed), plus `FuzzParseRSS` for the malformed class — never panic, and
      never return a title that is not a substring of the input.

- [ ] **Step 3: Run to verify they fail. Step 4: Implement** with `encoding/xml`
      into unexported structs, returning `[]store.NewsItem`. **Step 5: Verify.**

- [ ] **Step 6: Fuzz** for 60s; expect clean. **Step 7: Commit.**

### Task 2: `containsWord`, `usagesFrom`, `entryUsages`

**Files:**
- Create: `cmd/define/usage.go`
- Test: `cmd/define/usage_test.go`

- [ ] **Step 1: Write the failing tests.** Strategy: a table for `containsWord`
      whose rows are the boundary decisions, and — because it delegates to
      `highlightSpans` — one row per property that delegation is supposed to
      inherit: case, punctuation edges, hyphenated words, and a multi-word entry
      matching as a phrase. Plus one AGREEMENT test asserting `containsWord` and
      the highlighter answer the same for the same text, which is the property
      that makes reusing the matcher worth doing.

- [ ] **Step 2: Run to verify it fails. Step 3: Implement. Step 4: Verify.**

- [ ] **Step 5: `usagesFrom` and `entryUsages`.** Test `usagesFrom` against the
      captured fixture and assert the count in the measured 12–99 RANGE, not an
      exact number — an exact count pins the fixture, not the filter. Test
      `entryUsages` against an existing dictionary fixture and assert the source
      tag distinguishes them.

- [ ] **Step 6: Mutation-check** that dropping the filter reddens a named test,
      that `containsWord` returning true unconditionally reddens a different one,
      and that replacing the `highlightSpans` delegation with a naive
      `strings.Contains` reddens the agreement test.

- [ ] **Step 7: `sdlc milestone-close --issue 9 --milestone M1`.**

## Chunk 2: M2 — the cache, the seam, and the fake

### Task 3: `Store.NewsItems` / `SetNewsItems`

**Files:**
- Modify: `cmd/define/store/store.go`, `mem.go`, `yaml.go`, `storetest/suite.go`
- Modify: `cmd/define/history_store_test.go` — `failingStore` implements the
  method set explicitly rather than embedding `store.Store`, so widening the
  interface breaks it. Compiler-caught, but it is a file this task edits.

- [ ] **Step 1: Add the conformance rows FIRST** — empty is empty and not an
      error; set-then-get round-trips including the timestamp; a second set
      REPLACES rather than appends; keys are `store.Key`-normalised; an empty
      item list round-trips as empty-and-fetched, distinct from never-fetched
      (outcome 2 in the cache model, and the one a `[]NewsItem` alone cannot
      express).

- [ ] **Step 2: Verify both implementations fail. Step 3: Implement in `Mem`,
      then `YAML`. Step 4: Verify the suite passes for both.**

- [ ] **Step 5: Corrupt-file test** for `YAML`: this is a whole-file record, so
      the discipline is warn-and-skip the whole file. Assert a corrupt
      `usage/<slug>.yaml` warns and reads as never-fetched, rather than failing
      the session or returning half a record.

- [ ] **Step 6: Commit.**

### Task 4: the feed, the cache wrapper, and the fake

**Files:**
- Create: `cmd/define/news.go`, `cmd/define/news_fake_test.go`
- Test: `cmd/define/news_test.go`

- [ ] **Step 1: Write `fakeFeed`** — serves canned RSS bytes from the fixture,
      counts fetches per word, scriptable to fail.

- [ ] **Step 2: Write the failing cache tests, one per outcome in the model**
      (success-with-items, success-empty, failure), plus staleness: a fresh
      entry serves from cache with the fetch counter unmoved; a stale entry
      re-fetches; **a stale entry whose re-fetch FAILS serves the stale copy**,
      which is what keeps offline true.

- [ ] **Step 3: Verify they fail. Step 4: Implement `cachingFeed`. Step 5: Verify.**

- [ ] **Step 6: Implement `httpFeed`** — build the URL, bounded read
      (`maxFeedBytes`, mirroring `maxAudioBytes` in `fetch.go`), context honoured,
      non-200 is an error.

- [ ] **Step 7: Implement `bothSources`** and test that a failing news source
      still yields NOAD usages — the degradation the seam exists to provide.

- [ ] **Step 8: Mutation-check** each cache outcome and the stale-fallback rule;
      each must redden a named test.

- [ ] **Step 9: `sdlc milestone-close --issue 9 --milestone M2`.**

## Chunk 3: M3 — wiring, live conformance, docs

### Task 5: Wire the seam

**Files:**
- Modify: `cmd/define/main.go` (`deps.usage`, `storeDeps`, `openStore`, `withStore`)

- [ ] **Step 1: Write the wiring test FIRST**, as a table over every process
      entry path that can reach the seam, each driven through production wiring
      with the dependency in its REAL initial state. This is #21's lesson applied
      before the fact rather than after: #21 shipped two dead entry paths because
      its tests injected a pre-filled dependency and so began after the hop that
      fills it. There is no user-facing consumer yet, so the assertion is that
      `deps.usage` is non-nil and reaches the store after `withStore` — the same
      hop `TestWithStoreCarriesTheHighlightSetThrough` pins for #21.

- [ ] **Step 2: Wire it**, following `deps.vocab`'s shape: one instance built in
      `openStore`, merged in `withStore`, nil meaning "no usage source".

- [ ] **Step 3: Mutation-check** that dropping the `withStore` merge reddens the
      wiring test.

### Task 6: Live conformance + docs

**Files:**
- Create: `cmd/define/news_conformance_test.go`
- Modify: `atlas/define.md`, `atlas/index.md` if a new file needs linking

- [ ] **Step 1: Live check** — fetch a known-common word, assert the feed parses
      and yields usages above a FLOOR, and print what it got so the check doubles
      as the way to look at real output. Assert a floor, not an exact count.

- [ ] **Step 2: Atlas** — the seam, the two sources and why both, the raw-vs-derived
      split and what it buys, the three cache outcomes, and the personal-use terms
      note. No README line: nothing user-facing changed in this window.

- [ ] **Step 3: `sdlc close --issue 9 --verified '<evidence>'`.**

---

## Risks

- **The feed is not a contract.** Google can change or withdraw it. That is why the seam exists, why the cache is on disk, and why live conformance is a separate on-demand test rather than something merge-check depends on. If it goes away, `entryUsages` still works and authoring degrades rather than breaks.
- **Thematic collapse is measured and real** — 10 of 14 `sycophantic` headlines were about AI chatbots. That is a `#10` problem (it decides what to author from these), but it is the reason NOAD's examples are in scope here rather than deferred: the second source is what keeps a word from being taught only through this week's news cycle.
- **Headlines are not sentences.** They are clipped, verbless, sometimes all-caps. `#10` will need to judge usability; this issue's job is to deliver them tagged and honest, not to pre-filter on quality it cannot assess.
- **Personal-use terms.** The feed is licensed for personal feed-reader use. The atlas note is the durable record of that, and nothing in this issue redistributes content.

## Revisions

### 2026-08-26 — plan-quality round 1

Ten findings, six blocking. All accepted; three were design errors rather than
omissions.

- **PQ-1 (Critical) `cross-package-type`** — the plan put `Usage` in
  `package main` and gave `store.Store` a `Usages()` method returning it. That
  cannot compile, and the fix is not a move but a SPLIT: `store.NewsItem` is the
  raw cached thing in `package store`, `Usage` is derived at read time in
  `package main`.
- **PQ-2 `spec-divergence`** — addressed by the same split. The Spec says raw feed
  items are cached; my plan cached post-filter `[]Usage` without recording a
  reason. The Spec is also RIGHT: deriving on read means improving `containsWord`
  improves every word already cached, with no re-fetch.
- **PQ-3 `seam-promised-not-delivered`** — the Done-when says NOAD's examples are
  available "through the same seam" and the plan gave them a bare function beside
  a news-only interface, leaving the consumer to ask two shapes. `UsageSource`
  with a `bothSources` composite is the seam; news failing now degrades to the
  dictionary rather than to nothing.
- **PQ-4 `cost-understated`** — I wrote that `/usage` "costs one row in the
  command table". It does not: `commandCtx` carries no `context.Context` and no
  usage dependency, so the command means widening the contract every command
  shares, for a debug affordance, ahead of the real consumer. Dropped, with the
  reason recorded — #10 needs that wiring anyway.
- **PQ-5 `incomplete-state-model`** — the cache had two outcomes and needs three.
  A fetch that succeeds with ZERO items is a real answer, not a failure: not
  caching it re-fetches forever for exactly the words the feed is worst at, and
  caching it forever freezes them empty. Resolved with `FetchedAt` + a TTL, and
  a stale re-fetch that fails falls back to the stale copy.
- **PQ-6 `partial-reuse`** — `containsWord` needs phrase matching, case folding
  and joiner handling, all of which `highlightSpans` already does over
  `store.Key`. Naming only `wordRuns` would have re-committed #21's consolidation
  finding one level up, and the divergence is user-visible: a headline whose word
  renders green while `containsWord` rejects it. It now delegates to the matcher,
  with an AGREEMENT test as the pin.
- **Minors** — the torn-record test described the append-only day log's rule for
  what is actually a whole-file record (warn-and-skip is the right discipline);
  the prose test enumerations are compressed to one strategy line per risky
  function; `failingStore` implements `store.Store` explicitly rather than
  embedding it, so widening the interface breaks it and its file joins Task 3.
