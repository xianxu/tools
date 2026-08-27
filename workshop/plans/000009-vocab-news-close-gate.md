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
    - "n": 2
      timestamp: "2026-08-26T20:35:48-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Verified by reverting: inverting the order reddens usage_test.go:200 and :216.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: Subsequence property removed; its replacement has a new defect, raised separately.
          round: 2
        - id: BR-3
          disposition: addressed
          note: atlas/define.md:1035 now names M1 vs M2 in the sentence itself.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: 'Re-verified clean: blanking URL and At leaves the whole suite green.'
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: 'Re-verified clean: e.Blocks[:1] leaves the whole suite green.'
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: renderSpans is still at usage_test.go:73, still byte-identical to marked.
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: store/news.go:15 still names the publisher field Source.
          round: 2
        - id: BR-8
          disposition: not-addressed
          note: 'Re-verified: containsWord is false for "e.g.", "9/11", "rock ''n'' roll"; still undocumented.'
          round: 2
        - id: BR-9
          disposition: not-addressed
          note: Plan untouched since plan-quality round 1; all five prose claims and every checkbox stand.
          round: 2
      findings:
        - id: BR-10
          severity: Important
          title: The count-bound that replaced the subsequence property also fails on correct parsing — 3rd in family
          detail: |-
            cmd/define/rss_test.go:157. Measured two refuting shapes against correct code.
            (1) `<rss><channel><x:item><title>t</title></x:item></channel></rss>` yields items=1,
            strings.Count(data,"<item")=0, so `len(items) > tags` fires "the parser invented items" —
            encoding/xml matches `channel>item` by LOCAL name, and the committed fixture already
            declares xmlns:media, so the prefix is in the seed corpus and a 2-byte insertion reaches it.
            (2) One byte of the seeded fixture, "Mon, 24 Aug 2026" -> "1026", parses to year 1026 and
            trips the `it.At.Year() < 1900` assertion against a date the input actually supplied. The
            commit's "Sound under any decoding" is false. THIS IS THE 3RD FINDING IN THIS FAMILY on one
            target (substring, subsequence, count-bound) — do not write a fourth property. State the
            rule: a fuzz property may assert only what the code's own contract guarantees over arbitrary
            input, never a property of the input's textual form the decoder may transform (entities,
            namespace prefixes, line endings) and never a plausibility judgment about a supplied value.
            Sound form: count items with an xml.Decoder token walk, the same authority parseRSS uses.
            Enumeration to sweep this round: the 10 fuzz targets in cmd/define plus internal/llm
            (complete_test.go:67, usermodel_test.go:198, key_test.go:98 and :117, rss_test.go:143,
            highlightwriter_test.go:228, invariant_test.go:120, highlight_test.go:58 and :160,
            store/word_test.go:88, internal/llm/task_test.go:106).
          family: property-rejects-correct-behaviour
          round: 2
        - id: BR-11
          severity: Important
          title: A parsePubDate that GUESSES a date survives the whole M1 suite — 2nd in family
          detail: |-
            cmd/define/rss_test.go:174 guards the date assertion with `!it.At.IsZero()`, which skips
            exactly the unreadable-date case the comment above it claims to pin, and the table row at
            rss_test.go:75 asserts only the item count. Measured: replacing parsePubDate's final
            `return time.Time{}` with `return time.Date(2000,1,1,...)` — contradicting rss.go:50-56's
            stated contract that an unreadable date is the zero time, never a guess — leaves the entire
            cmd/define suite green. THIS IS THE 2ND FINDING IN THIS FAMILY (BR-4 named Usage.URL/At).
            Do not fix this site alone. Rule: every output field with a documented contract needs one
            assertion that goes red when the field is blanked or invented. Enumeration: Usage{Text,
            Source, Publisher, URL, At} and store.NewsItem{Title, URL, Source, At} — the three
            currently unpinned are Usage.URL, Usage.At, and NewsItem.At-on-unreadable.
          family: output-field-unasserted
          round: 2
        - id: BR-12
          severity: Important
          title: The Review-Verdict trailer is on a fix commit, so this window is empty and b0e907e is in none
          detail: |-
            `git log --grep 'Review-Verdict' | head -1` returns 61c9774 — the FIX commit — so this
            round's base resolved to HEAD and the diff handed to me was empty. workshop/lessons.md:914
            records this exact failure from define #16 M2, and `sdlc milestone-close --help` states the
            protocol: on FIX-THEN-SHIP the fixes bundle into the ONE milestone-close commit and it is
            not re-run. Here `- [x] M1` was ticked at 98d8ccc, the fixes landed separately at 61c9774
            carrying the trailer, and milestone-close ran again. Consequence: round 1 covered
            68e5551..98d8ccc, this round covers nothing, and M2's window starts at this boundary — so
            b0e907e (news.go 128 lines including net/http, news_test.go 203, news_fake_test.go 60,
            plus store.go/mem.go/yaml.go/storetest/suite.go) and 61c9774's own 103 test/atlas lines are
            in NO review window. Fix: close M2 with an explicit widened window,
            `sdlc judge milestone-review --base 98d8ccc7 --head <M2 close>`, and record it in the Log.
          family: boundary-marker-strands-commits
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-08-26T21:07:45-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: addressed
          note: 'Verified by reverting: blanking URL and At reddens usage_test.go:270 on both fields.'
          round: 3
        - id: BR-5
          disposition: not-addressed
          note: 'Re-verified against the FULL suite: e.Blocks[:1] leaves all of cmd/define green.'
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: renderSpans (usage_test.go:74) is still byte-identical to marked (highlight_test.go:110).
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: store/news.go:15 still names the publisher field Source, one line from Usage.Source.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: Re-verified false for "e.g.", "9/11", "rock 'n' roll"; still undocumented in usage.go and atlas.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: Worse than reported - the Integration table names newsUsageSource, which exists nowhere, and the issue's own M2/M3 rows are ticked while naming Store.Usages, NewsSource and /usage, none of which exist.
          round: 3
        - id: BR-10
          disposition: addressed
          note: Count-bound gone; rule stated in lessons.md and the target's doc. I read all 12 fuzz targets in the tree - no other instance of the class.
          round: 3
        - id: BR-11
          disposition: addressed
          note: 'Verified by reverting: the guessing mutant reddens rss_test.go:87 and FuzzParsePubDate seeds 1 and 6.'
          round: 3
        - id: BR-12
          disposition: addressed
          note: 'This window''s base equals git merge-base main HEAD, so all eight #9 commits including b0e907e are covered; no M2/M3 commit carries a verdict trailer.'
          round: 3
      findings:
        - id: BR-13
          severity: Critical
          title: TestWithStoreCarriesTheUsageSourceThrough makes a live request to news.google.com on every untagged test run
          detail: |-
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
          family: unit-test-reaches-real-dependency
          round: 3
        - id: BR-14
          severity: Important
          title: Every error branch in news.go is uncovered, and the store-read degradation survives inversion
          detail: |-
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
          family: documented-degradation-unpinned
          round: 3
        - id: BR-15
          severity: Important
          title: bothSources discards the feed error with no warning, and UsageSource.Usages can never return one
          detail: |-
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
          family: degradation-without-signal
          round: 3
        - id: BR-16
          severity: Minor
          title: cachingFeed's SetNewsItems error handler and its fallthrough return the same value
          detail: |-
            news.go:122-126 is `if err := c.st.SetNewsItems(...); err != nil { return items, nil }`
            followed by `return items, nil`. Both branches are identical, so the guard is dead and
            cannot be mutated. `_ = c.st.SetNewsItems(...)` with the same comment says it honestly.
          family: dead-branch
          round: 3
        - id: BR-17
          severity: Minor
          title: parseRSS never checks the root element, so non-RSS XML becomes a cached "no news"
          detail: |-
            rss.go:16-38 unmarshals into a struct keyed only on channel>item, so any well-formed XML
            that is not an RSS feed returns (empty, nil). Round 2 noted this as M2 scope; M2 is in this
            window, and at M2 it becomes cache outcome 2 - a wrong-content 200 frozen as "no news for
            this word" for a full cacheTTL.
          family: parser-accepts-wrong-document
          round: 3
        - id: BR-18
          severity: Minor
          title: FuzzParseRSS's remaining property can never fire, so the target is now a no-panic smoke test
          detail: |-
            rss_test.go:181 asserts `err != nil && items != nil`, but parseRSS's only error return is
            `return nil, err`, so the guard is unreachable by construction. This is NOT a request for a
            fourth property - BR-10 was right about that. The recommendation is to record, in the plan
            and the Log, that the Done-when's second clause ("never returns an item it did not find in
            the input") is pinned by the exact-count and exact-title assertions in
            TestParseRSSOverACapturedFeed and TestParseRSSDecisions, not by the fuzz target.
          family: assertion-cannot-fire
          round: 3
        - id: BR-19
          severity: Minor
          title: The atlas's canonical on-disk layout block does not list usage/
          detail: |-
            atlas/define.md:216-219 shows words/, events/ and user-model.md as the layout, while the
            new section at :1064 says usage/<slug>.yaml sits "beside words/ and events/". One fact,
            two places, one stale - a reader who consults the layout map will not see the new
            directory.
          family: fact-documented-in-two-places
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-26T21:40:19-07:00"
      agent: claude
      dispose:
        - id: BR-13
          disposition: addressed
          note: 'Verified by probe: panicking in Fetch when base=="" never fires over the full suite — zero live requests.'
          round: 4
        - id: BR-14
          disposition: not-addressed
          note: news.go's branches are pinned (verified by reverting), but the quoted-word query still drops green under mutation and usage.go:107-109 is at zero coverage — both named in the finding's own text.
          round: 4
        - id: BR-15
          disposition: not-addressed
          note: bothSources.warn is nil at BOTH production sites (main.go:180, main.go:215, probed); warnTo no-ops on nil, so production degrades exactly as silently as before.
          round: 4
        - id: BR-5
          disposition: not-addressed
          note: 'Re-verified by mutation: e.Blocks[:1] leaves the entire cmd/define suite green.'
          round: 4
        - id: BR-6
          disposition: not-addressed
          note: renderSpans (usage_test.go:74) is still byte-identical to marked (highlight_test.go:110).
          round: 4
        - id: BR-7
          disposition: not-addressed
          note: store/news.go:15 still names the publisher field Source, one line from Usage.Source.
          round: 4
        - id: BR-8
          disposition: not-addressed
          note: Re-verified false for "e.g.", "9/11", "rock 'n' roll" against text containing them verbatim; still undocumented in usage.go and the atlas.
          round: 4
        - id: BR-9
          disposition: not-addressed
          note: 'Measured at the close: 0 of 26 plan checkboxes ticked, newsUsageSource still at plan line 88, and the issue''s ticked M2/M3 rows still name Store.Usages, NewsSource and /usage.'
          round: 4
        - id: BR-16
          disposition: not-addressed
          note: news.go:133-138 unchanged — both branches still return items, nil.
          round: 4
        - id: BR-17
          disposition: not-addressed
          note: rss.go:16-38 unchanged; well-formed non-RSS XML still returns (empty, nil) and is cacheable as "no news".
          round: 4
        - id: BR-18
          disposition: not-addressed
          note: rss_test.go property unchanged and still unreachable; neither the plan nor the Log records what actually pins the Done-when clause.
          round: 4
        - id: BR-19
          disposition: not-addressed
          note: atlas/define.md:210-220 layout block still lists only words/, events/ and user-model.md.
          round: 4
      findings:
        - id: BR-20
          severity: Minor
          title: newsFile was spliced under Forget's doc comment, so YAML.Forget lost its documentation
          detail: |-
            cmd/define/store/yaml.go:356-361. The four-line comment documenting Forget ("Forget removes
            one word file... Filename derivation goes through wordFileName...") is immediately followed,
            with no blank line, by the newsFile type declaration — so godoc attaches it to newsFile, which
            it does not describe, and func (y *YAML) Forget at :417 has no doc at all. Verified:
            `go doc store.YAML.Forget` prints only the signature, and `git show 361136b:cmd/define/store/yaml.go`
            has the comment attached at :353-358. A regression introduced by this window. Fix: move the
            Forget doc back above :417 and give newsFile its own.
          family: doc-comment-detached-from-declaration
          round: 4
        - id: BR-21
          severity: Minor
          title: Forget does not remove usage/<slug>.yaml, and neither the doc nor storetest says so
          detail: |-
            cmd/define/store/yaml.go:417 removes only words/<slug>.yaml; cmd/define/store/mem.go:138 only
            deletes from m.words. Store.Forget's doc deliberately enumerates what it leaves behind ("does
            NOT remove events: the deck is a working set, the log is history"), and this window added a
            per-word on-disk record that the enumeration does not mention. Unlike events, the news cache is
            derived and refetchable, so the argument for keeping it is weaker — a user who forgets a word
            still has its headlines on disk. The rule: adding a per-word record obliges the delete verb to
            declare whether it is removed, and a storetest row to pin the answer. Decide either way, but say
            it in the interface doc and pin it in the suite.
          family: delete-scope-unstated-for-new-record
          round: 4
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

## Round 2 — 2026-08-26T20:35:48-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Verified by reverting: inverting the order reddens usage_test.go:200 and :216.
- BR-2 — addressed — Subsequence property removed; its replacement has a new defect, raised separately.
- BR-3 — addressed — atlas/define.md:1035 now names M1 vs M2 in the sentence itself.
- BR-4 — not-addressed — Re-verified clean: blanking URL and At leaves the whole suite green.
- BR-5 — not-addressed — Re-verified clean: e.Blocks[:1] leaves the whole suite green.
- BR-6 — not-addressed — renderSpans is still at usage_test.go:73, still byte-identical to marked.
- BR-7 — not-addressed — store/news.go:15 still names the publisher field Source.
- BR-8 — not-addressed — Re-verified: containsWord is false for "e.g.", "9/11", "rock 'n' roll"; still undocumented.
- BR-9 — not-addressed — Plan untouched since plan-quality round 1; all five prose claims and every checkbox stand.

### Raised

- **BR-10** [Important] `property-rejects-correct-behaviour` The count-bound that replaced the subsequence property also fails on correct parsing — 3rd in family
  cmd/define/rss_test.go:157. Measured two refuting shapes against correct code.
  (1) `<rss><channel><x:item><title>t</title></x:item></channel></rss>` yields items=1,
  strings.Count(data,"<item")=0, so `len(items) > tags` fires "the parser invented items" —
  encoding/xml matches `channel>item` by LOCAL name, and the committed fixture already
  declares xmlns:media, so the prefix is in the seed corpus and a 2-byte insertion reaches it.
  (2) One byte of the seeded fixture, "Mon, 24 Aug 2026" -> "1026", parses to year 1026 and
  trips the `it.At.Year() < 1900` assertion against a date the input actually supplied. The
  commit's "Sound under any decoding" is false. THIS IS THE 3RD FINDING IN THIS FAMILY on one
  target (substring, subsequence, count-bound) — do not write a fourth property. State the
  rule: a fuzz property may assert only what the code's own contract guarantees over arbitrary
  input, never a property of the input's textual form the decoder may transform (entities,
  namespace prefixes, line endings) and never a plausibility judgment about a supplied value.
  Sound form: count items with an xml.Decoder token walk, the same authority parseRSS uses.
  Enumeration to sweep this round: the 10 fuzz targets in cmd/define plus internal/llm
  (complete_test.go:67, usermodel_test.go:198, key_test.go:98 and :117, rss_test.go:143,
  highlightwriter_test.go:228, invariant_test.go:120, highlight_test.go:58 and :160,
  store/word_test.go:88, internal/llm/task_test.go:106).
- **BR-11** [Important] `output-field-unasserted` A parsePubDate that GUESSES a date survives the whole M1 suite — 2nd in family
  cmd/define/rss_test.go:174 guards the date assertion with `!it.At.IsZero()`, which skips
  exactly the unreadable-date case the comment above it claims to pin, and the table row at
  rss_test.go:75 asserts only the item count. Measured: replacing parsePubDate's final
  `return time.Time{}` with `return time.Date(2000,1,1,...)` — contradicting rss.go:50-56's
  stated contract that an unreadable date is the zero time, never a guess — leaves the entire
  cmd/define suite green. THIS IS THE 2ND FINDING IN THIS FAMILY (BR-4 named Usage.URL/At).
  Do not fix this site alone. Rule: every output field with a documented contract needs one
  assertion that goes red when the field is blanked or invented. Enumeration: Usage{Text,
  Source, Publisher, URL, At} and store.NewsItem{Title, URL, Source, At} — the three
  currently unpinned are Usage.URL, Usage.At, and NewsItem.At-on-unreadable.
- **BR-12** [Important] `boundary-marker-strands-commits` The Review-Verdict trailer is on a fix commit, so this window is empty and b0e907e is in none
  `git log --grep 'Review-Verdict' | head -1` returns 61c9774 — the FIX commit — so this
  round's base resolved to HEAD and the diff handed to me was empty. workshop/lessons.md:914
  records this exact failure from define #16 M2, and `sdlc milestone-close --help` states the
  protocol: on FIX-THEN-SHIP the fixes bundle into the ONE milestone-close commit and it is
  not re-run. Here `- [x] M1` was ticked at 98d8ccc, the fixes landed separately at 61c9774
  carrying the trailer, and milestone-close ran again. Consequence: round 1 covered
  68e5551..98d8ccc, this round covers nothing, and M2's window starts at this boundary — so
  b0e907e (news.go 128 lines including net/http, news_test.go 203, news_fake_test.go 60,
  plus store.go/mem.go/yaml.go/storetest/suite.go) and 61c9774's own 103 test/atlas lines are
  in NO review window. Fix: close M2 with an explicit widened window,
  `sdlc judge milestone-review --base 98d8ccc7 --head <M2 close>`, and record it in the Log.

## Round 3 — 2026-08-26T21:07:45-07:00 (claude) — BLOCKED

### Disposed

- BR-4 — addressed — Verified by reverting: blanking URL and At reddens usage_test.go:270 on both fields.
- BR-5 — not-addressed — Re-verified against the FULL suite: e.Blocks[:1] leaves all of cmd/define green.
- BR-6 — not-addressed — renderSpans (usage_test.go:74) is still byte-identical to marked (highlight_test.go:110).
- BR-7 — not-addressed — store/news.go:15 still names the publisher field Source, one line from Usage.Source.
- BR-8 — not-addressed — Re-verified false for "e.g.", "9/11", "rock 'n' roll"; still undocumented in usage.go and atlas.
- BR-9 — not-addressed — Worse than reported - the Integration table names newsUsageSource, which exists nowhere, and the issue's own M2/M3 rows are ticked while naming Store.Usages, NewsSource and /usage, none of which exist.
- BR-10 — addressed — Count-bound gone; rule stated in lessons.md and the target's doc. I read all 12 fuzz targets in the tree - no other instance of the class.
- BR-11 — addressed — Verified by reverting: the guessing mutant reddens rss_test.go:87 and FuzzParsePubDate seeds 1 and 6.
- BR-12 — addressed — This window's base equals git merge-base main HEAD, so all eight #9 commits including b0e907e are covered; no M2/M3 commit carries a verdict trailer.

### Raised

- **BR-13** [Critical] `unit-test-reaches-real-dependency` TestWithStoreCarriesTheUsageSourceThrough makes a live request to news.google.com on every untagged test run
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
- **BR-14** [Important] `documented-degradation-unpinned` Every error branch in news.go is uncovered, and the store-read degradation survives inversion
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
- **BR-15** [Important] `degradation-without-signal` bothSources discards the feed error with no warning, and UsageSource.Usages can never return one
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
- **BR-16** [Minor] `dead-branch` cachingFeed's SetNewsItems error handler and its fallthrough return the same value
  news.go:122-126 is `if err := c.st.SetNewsItems(...); err != nil { return items, nil }`
  followed by `return items, nil`. Both branches are identical, so the guard is dead and
  cannot be mutated. `_ = c.st.SetNewsItems(...)` with the same comment says it honestly.
- **BR-17** [Minor] `parser-accepts-wrong-document` parseRSS never checks the root element, so non-RSS XML becomes a cached "no news"
  rss.go:16-38 unmarshals into a struct keyed only on channel>item, so any well-formed XML
  that is not an RSS feed returns (empty, nil). Round 2 noted this as M2 scope; M2 is in this
  window, and at M2 it becomes cache outcome 2 - a wrong-content 200 frozen as "no news for
  this word" for a full cacheTTL.
- **BR-18** [Minor] `assertion-cannot-fire` FuzzParseRSS's remaining property can never fire, so the target is now a no-panic smoke test
  rss_test.go:181 asserts `err != nil && items != nil`, but parseRSS's only error return is
  `return nil, err`, so the guard is unreachable by construction. This is NOT a request for a
  fourth property - BR-10 was right about that. The recommendation is to record, in the plan
  and the Log, that the Done-when's second clause ("never returns an item it did not find in
  the input") is pinned by the exact-count and exact-title assertions in
  TestParseRSSOverACapturedFeed and TestParseRSSDecisions, not by the fuzz target.
- **BR-19** [Minor] `fact-documented-in-two-places` The atlas's canonical on-disk layout block does not list usage/
  atlas/define.md:216-219 shows words/, events/ and user-model.md as the layout, while the
  new section at :1064 says usage/<slug>.yaml sits "beside words/ and events/". One fact,
  two places, one stale - a reader who consults the layout map will not see the new
  directory.

## Round 4 — 2026-08-26T21:40:19-07:00 (claude) — BLOCKED

### Disposed

- BR-13 — addressed — Verified by probe: panicking in Fetch when base=="" never fires over the full suite — zero live requests.
- BR-14 — not-addressed — news.go's branches are pinned (verified by reverting), but the quoted-word query still drops green under mutation and usage.go:107-109 is at zero coverage — both named in the finding's own text.
- BR-15 — not-addressed — bothSources.warn is nil at BOTH production sites (main.go:180, main.go:215, probed); warnTo no-ops on nil, so production degrades exactly as silently as before.
- BR-5 — not-addressed — Re-verified by mutation: e.Blocks[:1] leaves the entire cmd/define suite green.
- BR-6 — not-addressed — renderSpans (usage_test.go:74) is still byte-identical to marked (highlight_test.go:110).
- BR-7 — not-addressed — store/news.go:15 still names the publisher field Source, one line from Usage.Source.
- BR-8 — not-addressed — Re-verified false for "e.g.", "9/11", "rock 'n' roll" against text containing them verbatim; still undocumented in usage.go and the atlas.
- BR-9 — not-addressed — Measured at the close: 0 of 26 plan checkboxes ticked, newsUsageSource still at plan line 88, and the issue's ticked M2/M3 rows still name Store.Usages, NewsSource and /usage.
- BR-16 — not-addressed — news.go:133-138 unchanged — both branches still return items, nil.
- BR-17 — not-addressed — rss.go:16-38 unchanged; well-formed non-RSS XML still returns (empty, nil) and is cacheable as "no news".
- BR-18 — not-addressed — rss_test.go property unchanged and still unreachable; neither the plan nor the Log records what actually pins the Done-when clause.
- BR-19 — not-addressed — atlas/define.md:210-220 layout block still lists only words/, events/ and user-model.md.

### Raised

- **BR-20** [Minor] `doc-comment-detached-from-declaration` newsFile was spliced under Forget's doc comment, so YAML.Forget lost its documentation
  cmd/define/store/yaml.go:356-361. The four-line comment documenting Forget ("Forget removes
  one word file... Filename derivation goes through wordFileName...") is immediately followed,
  with no blank line, by the newsFile type declaration — so godoc attaches it to newsFile, which
  it does not describe, and func (y *YAML) Forget at :417 has no doc at all. Verified:
  `go doc store.YAML.Forget` prints only the signature, and `git show 361136b:cmd/define/store/yaml.go`
  has the comment attached at :353-358. A regression introduced by this window. Fix: move the
  Forget doc back above :417 and give newsFile its own.
- **BR-21** [Minor] `delete-scope-unstated-for-new-record` Forget does not remove usage/<slug>.yaml, and neither the doc nor storetest says so
  cmd/define/store/yaml.go:417 removes only words/<slug>.yaml; cmd/define/store/mem.go:138 only
  deletes from m.words. Store.Forget's doc deliberately enumerates what it leaves behind ("does
  NOT remove events: the deck is a working set, the log is history"), and this window added a
  per-word on-disk record that the enumeration does not mention. Unlike events, the news cache is
  derived and refetchable, so the argument for keeping it is weaker — a user who forgets a word
  still has its headlines on disk. The rule: adding a per-word record obliges the delete verb to
  declare whether it is removed, and a storetest row to pin the answer. Decide either way, but say
  it in the interface doc and pin it in the suite.

## Open findings

- **BR-5** [Minor] `at-least-one-hides-undercollection` entryUsages' multi-block traversal is unpinned by an at-least-one assertion
- **BR-6** [Minor] `helper-duplicated-in-package` renderSpans duplicates marked from highlight_test.go in the same package
- **BR-7** [Minor] `name-means-two-things` store.NewsItem.Source is the publisher while Usage.Source is provenance
- **BR-8** [Minor] `matcher-limit-silently-drops-input` A punctuated deck key can never yield a usage, and nothing says so
- **BR-9** [Minor] `plan-record-drift` Every plan checkbox is unticked at the boundary, and five prose claims contradict the code
- **BR-14** [Important] `documented-degradation-unpinned` Every error branch in news.go is uncovered, and the store-read degradation survives inversion
- **BR-15** [Important] `degradation-without-signal` bothSources discards the feed error with no warning, and UsageSource.Usages can never return one
- **BR-16** [Minor] `dead-branch` cachingFeed's SetNewsItems error handler and its fallthrough return the same value
- **BR-17** [Minor] `parser-accepts-wrong-document` parseRSS never checks the root element, so non-RSS XML becomes a cached "no news"
- **BR-18** [Minor] `assertion-cannot-fire` FuzzParseRSS's remaining property can never fire, so the target is now a no-panic smoke test
- **BR-19** [Minor] `fact-documented-in-two-places` The atlas's canonical on-disk layout block does not list usage/
- **BR-20** [Minor] `doc-comment-detached-from-declaration` newsFile was spliced under Forget's doc comment, so YAML.Forget lost its documentation
- **BR-21** [Minor] `delete-scope-unstated-for-new-record` Forget does not remove usage/<slug>.yaml, and neither the doc nor storetest says so
