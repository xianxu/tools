# Boundary Review — tools#9 (milestone M1)

| field | value |
|-------|-------|
| issue | 9 — news seam: Google News RSS client with a stateful fake |
| repo | tools |
| issue file | workshop/issues/000009-vocab-news.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 68e555122533f0d3fbd7b2d7efab26bc48eed2b8^..98d8ccc7284e067040a686646420c1667dca6d43 |
| command | sdlc milestone-close --issue 9 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-26T20:12:36-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what the issue's Plan row claims — `parseRSS`, `store.NewsItem`, `Usage`, `containsWord`, and NOAD's examples as the second source — and the boundary snapshot is genuinely green: at a clean extraction of `98d8ccc`, `go build ./...`, `go vet ./...` and `go test ./...` all pass (the only four failures are the repo-guard tests, which fail solely because my scratch copy is not a git checkout). I verified the plan's own mutation claims by running them rather than reading them: dropping the filter, `containsWord → true`, the naive `strings.Contains` delegation, and the "cut at the last dash" rule each redden a named test. What holds this back from SHIP is three things I demonstrated by experiment. `usagesFrom` strips attribution *then* filters, and nothing pins that order — I inverted it and the entire package stayed green, while a headline whose only occurrence of the word is inside the stripped publisher (`"Best startups of 2026 - Ephemeral Media"`) becomes a usage whose `Text` is `"Best startups of 2026"` and whose `containsWord` is `false`, which is the Done-when row this milestone exists to satisfy. `FuzzParseRSS`'s subsequence property is unsound in the other direction: I seeded it with `&#65;`, `&#39;` and a lone `\r` and all three reported "the parser invented content" against correct XML parsing — and `&#39;` is what Google News actually emits for apostrophes, so a fixture refresh or the plan's own mandated 60s fuzz run will redden on correct code. Both are cheap. Note that HEAD moved past this window during the review (`b0e907e`, M2's cache); I scoped to `98d8ccc` as instructed, and I re-checked both Importants against current HEAD — still open. The diff also carries all of #17/#20/#21 because `BASE_SHA` predates the #17 merge; those milestones have their own recorded verdicts and I did not re-review them.

## 1. Strengths

- **`containsWord` delegating to `highlightSpans` is the milestone's best decision** (`cmd/define/usage.go:55`), and it is genuinely enforced, not asserted: I replaced the delegation with a case-folded `strings.Contains` and both `TestContainsWord` and `TestContainsWordAgreesWithTheHighlighter` went red. The stated payoff — a headline rendering green while the filter rejects it — is a real user-visible divergence that this closes. ARCH-DRY at its best.
- **The captured fixture earns its keep, and its composition is right.** 13 items, 10 matching, 10 distinct publishers — so `containsWord → true` fails on the count (13≠10) and the filter has actual work to do. That is exactly the discrimination the repo's own lesson at `workshop/lessons.md` demands, and it is why four of five plan-named mutations died first time.
- **`withoutAttribution` is the correct rule, correctly fixtured** (`cmd/define/usage.go:98`). `TestUsageKeepsADashThatIsNotAttribution` uses a title carrying *no* attribution, so the correct rule and the last-dash rule diverge — I ran the last-dash mutant and it reddens. The Log records that this fixture was fixed after the first version let the mutant live; the fix is real.
- **`parsePubDate` returning the zero time rather than an error** (`cmd/define/rss.go:57`) is the right call for a feed source, stated with its reason, and pinned by a table row that asserts one bad date does not lose the other item.
- **Pure over bytes throughout.** `parseRSS`, `containsWord`, `usagesFrom`, `withoutAttribution` and `entryUsages` are all deterministic functions with no clock, no net, no fs — every test in `usage_test.go` runs with no double at all. ARCH-PURE passes cleanly, and the malformed-feed requirement really is a table plus a fuzz target with no network anywhere near it.

## 2. Critical findings

None.

## 3. Important findings

**I1 — the strip-then-filter order is unpinned, and inverting it produces usages that do not contain the word** (`cmd/define/usage.go:74-77`). `usagesFrom` computes `text := withoutAttribution(...)` and filters on `text`; changing that one call to `containsWord(it.Title, word)` leaves `go test ./cmd/define/` fully green (verified; `usagesFrom`/`containsWord`/`withoutAttribution` have no callers outside `usage.go` and `usage_test.go`, so nothing else could catch it). The consequence, measured with a scratch fixture: `{Title: "Best startups of 2026 - Ephemeral Media", Source: "Ephemeral Media"}` yields one usage with `Text = "Best startups of 2026"` and `containsWord(u.Text, "ephemeral") == false` — a direct violation of the Done-when row *"Only headlines that genuinely contain the word become usages."* `TestUsagesFromTheCapturedFeed:100` already has exactly the right assertion for this (`if !containsWord(u.Text, "ephemeral")`); it cannot fire because no fixture row puts the word only in the attribution. Fix: one row in `TestUsageTextDropsThePublisherSuffix`'s neighbourhood asserting zero usages for a title whose sole occurrence of the word is the publisher suffix.

**I2 — `FuzzParseRSS`'s subsequence property reports correct parsing as invention** (`cmd/define/rss_test.go:156`). The comment correctly identifies that substring is too strong and names entity decoding as a reason, but subsequence does not survive it either. Verified by adding three seeds to a scratch copy and running the target:

```
item 0 title "A"     is not an ordered subsequence  <- &#65;
item 0 title "'q'"   is not an ordered subsequence  <- &#39;q&#39;
item 0 title "a\nb"  is not an ordered subsequence  <- a\rb  (XML line-ending normalisation)
```

Each is `encoding/xml` behaving exactly to spec. `&#39;` is not contrived — Google News emits it for apostrophes routinely, and the plan's Task 1 Step 1 makes re-capturing the fixture a first-class operation, so a refresh can turn this red deterministically. The plan's Step 6 ("Fuzz for 60s; expect clean") will also find it eventually. The danger is not the false failure itself but the natural response to one: weakening the property that defends a Done-when row. Fix: apply the parser's own normalisations before comparing, or narrow the property honestly — `if strings.Contains(data, "&#") || strings.ContainsAny(data, "\r") { continue }` costs one line and keeps the claim true.

**I3 — the atlas describes M2's cache in the present tense at an M1 boundary** (`atlas/define.md`, "Raw is cached; usable is derived"). At `98d8ccc` nothing goes to disk and nothing queries the feed: there is no `Store.NewsItems`, no `cachingFeed`, no `httpFeed`. The paragraph tells a reader that `store.NewsItem` "is what goes to disk" and that improving `containsWord` improves "every word already cached", and the section carries no scope marker anywhere. This is verbatim the class `workshop/lessons.md` records from #21 M1 — *"Doc prose at a boundary describes what THAT milestone shipped"*, whose stated fix form is to write the scope into the sentence ("…from M2"). **Honest qualification: this is already overtaken.** `b0e907e`, committed outside the review window while I was reviewing, lands the cache, so the claim is accurate at current HEAD. I report it because the rule recurred, not because the instance still needs a patch — dispose it `withdrawn` if you prefer, but the closing step ("write the milestone's scope into the sentence") is what stops the next one.

## 4. Minor findings

- **`Usage.URL` and `Usage.At` are never asserted.** Replacing `URL: it.URL, At: it.At` with zero values (`usage.go:82-83`) leaves the suite green — and those are the two fields #10 needs to attribute a sentence.
- **`entryUsages`' multi-block traversal is unpinned.** Restricting the loop to `e.Blocks[0]` survives, because `TestEntryUsagesLiftsNOADExamples:175` asserts only `len(got) != 0` plus per-item properties, so any under-collection is invisible. A count, or one example known to live under a later block, closes it.
- **`renderSpans` (`usage_test.go:73`) is a byte-for-byte duplicate of `marked` (`highlight_test.go`) in the same package** — ARCH-DRY, in the one milestone whose thesis is "don't reimplement the matcher". `marked` returns `string` and is directly usable.
- **`Source` means two different things one line apart.** `store.NewsItem.Source` is the publisher; `Usage.Source` is provenance; `usagesFrom` writes `Publisher: it.Source`. Renaming the store field `Publisher` would remove the trap before #10 reads both types.
- **A punctuated deck key silently yields zero usages, forever.** Verified: `containsWord("a sentence about e.g. here", "e.g.")`, and likewise `9/11` and `rock 'n' roll`, all return `false` — inherited from #21's `phraseRunsJoin` bound on `MaxPhraseWords`. In #21 the cost was "not highlighted"; here it is "this word can never be taught from a real sentence", which is a different stake and is documented nowhere.
- **The durable plan records none of this.** Every checkbox in `workshop/plans/000009-vocab-news-plan.md` is unticked, including all seven of Task 1's and Task 2's, while the issue marks `- [x] M1`. See §7 for the prose drift in the same file.

`UsageSource` (`usage.go:38`) has zero implementations and zero call sites at this boundary — noted rather than filed, since the plan places it here deliberately to document the seam and M2 supplies `bothSources`.

## 5. Test coverage notes

- Mutation results, all run against a clean extraction of the boundary: **dead** — drop the filter, `containsWord → true`, naive `strings.Contains` delegation, last-dash attribution rule, `entryUsages` without its filter. **Survived** — filter on the raw title (I1), blank `URL`/`At`, `entryUsages` reading only block 0.
- `TestContainsWordAgreesWithTheHighlighter` is worth understanding precisely: both sides call `highlightSpans` with an equivalent one-word vocabulary, so as a general "agreement" property it is close to tautological today. It does kill the exact mutation the plan named (replacing the delegation), which is its stated job — but it will not notice divergence introduced at the *rendering* layer (`admitsHighlight`, `highlightSetFor`), which is where the user actually sees green.
- The `if !containsWord(u.Text, …)` loops in `TestUsagesFromTheCapturedFeed` and `TestEntryUsagesLiftsNOADExamples` are correct and precise assertions — they are simply under-fixtured (I1). Do not delete them; give them the row that makes them discriminating.
- ARCH-MOCK is not yet in play: M1 makes no external call, and the captured real feed is the right first artifact. The seam, the stateful fake and the live conformance check are M2/M3 and correctly deferred.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass, with one test-side flag.** `containsWord` reusing the whole matcher is the model to keep, and `parseRSS` returning `store.NewsItem` rather than an `encoding/xml` struct is the right seam for a second feed format. The one duplication is `renderSpans`/`marked`.
- **ARCH-PURE — pass.** All five new functions are pure; the IO shell is entirely deferred to M2, and that is what makes the malformed-feed requirement a fuzz target instead of a network test.
- **ARCH-PURPOSE — pass at the milestone's scope.** M1's four claimed deliverables are all present. The "same seam" Done-when row is M2's by plan, and the interface is declared here rather than promised.
- **ARCH-MOCK — not yet applicable; the plan names the seam, the byte-serving fake, and the conformance cadence correctly.**
- For M2: `cachingFeed`'s three outcomes each need a mutation that names its own test (plan Task 4 Step 8). The class I1 belongs to — *a fixture that cannot tell the correct rule from the plausible wrong one* — is the one to watch there, because "served from cache" and "re-fetched and got the same answer" produce identical output and differ only on the counter. The plan already says to assert on the counter; hold it to that.

## 7. Plan revision recommendations

A `## Revisions` entry in `workshop/plans/000009-vocab-news-plan.md`, plus ticking M1's boxes. The prose has drifted from the code in five places (the Core-concepts *table* rows are all accurate — every entity exists at its stated path, all `new`):

- `Usage{Text, Source, Title, URL, At}` — the shipped struct has `Publisher`, not `Title` (`usage.go:25`).
- `store.NewsItem{Title, URL, At}` — the shipped struct also carries `Source`, the publisher, which the whole attribution rule depends on (`store/news.go:12`).
- `entryUsages(e Entry) []Usage` — the shipped signature is `(e Entry, word string)`, and the filtering that extra parameter enables is a documented decision.
- Test surface: *"never return an item whose title is not a **substring** of the input"* — the code correctly uses subsequence, the issue Log records the correction, the plan does not. Record it, and record I2's remaining unsoundness with it.
- Task 2 Step 5: *"assert the count in the measured 12–99 RANGE, not an exact number"* — impossible against a 13-item fixture, and the code's exact-10 assertion with its stated rationale is the better call. Correct the step rather than leaving the code looking like a deviation.

```findings
findings:
  - id: new
    severity: Important
    family: fixture-does-not-discriminate
    title: |
      usagesFrom's strip-then-filter order is unpinned; inverting it yields usages whose text lacks the word
    detail: |
      cmd/define/usage.go:74-77. Verified: changing `containsWord(text, word)` to
      `containsWord(it.Title, word)` leaves the whole cmd/define suite green, and these
      functions have no callers outside usage.go/usage_test.go so nothing else could catch
      it. Measured consequence with a scratch fixture — {Title: "Best startups of 2026 -
      Ephemeral Media", Source: "Ephemeral Media"} yields Text="Best startups of 2026" with
      containsWord=false, violating the Done-when row "only headlines that genuinely contain
      the word become usages". TestUsagesFromTheCapturedFeed:100 already holds the right
      assertion; it cannot fire because no fixture row puts the word only in the attribution.
      One row closes it.
  - id: new
    severity: Important
    family: property-rejects-correct-behaviour
    title: |
      FuzzParseRSS's subsequence property reports correct XML parsing as invented content
    detail: |
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
  - id: new
    severity: Important
    family: docs-describe-unshipped-milestone
    title: |
      The atlas describes M2's cache in the present tense at the M1 boundary
    detail: |
      atlas/define.md, "Raw is cached; usable is derived" — at 98d8ccc there is no
      Store.NewsItems, no cachingFeed and no httpFeed, yet the section states that
      store.NewsItem "is what goes to disk", that "every word already cached" improves, and
      that "the feed is queried", with no scope marker. This is verbatim the class
      workshop/lessons.md records from #21 M1, whose stated fix is to write the milestone
      into the sentence. Overtaken in fact: b0e907e (outside this window) landed the cache,
      so the claim is accurate at current HEAD — dispose withdrawn if preferred. Reported
      because the rule recurred, and the closing step is what prevents the next instance.
  - id: new
    severity: Minor
    family: output-field-unasserted
    title: |
      Usage.URL and Usage.At are never asserted, so blanking them is invisible
    detail: |
      cmd/define/usage.go:82-83. Replacing them with zero values leaves the suite green,
      and they are the two fields #10 needs to attribute a sentence to its source.
  - id: new
    severity: Minor
    family: at-least-one-hides-undercollection
    title: |
      entryUsages' multi-block traversal is unpinned by an at-least-one assertion
    detail: |
      Restricting the loop to e.Blocks[0] survives the suite, because
      TestEntryUsagesLiftsNOADExamples:175 asserts only len(got) != 0 plus per-item
      properties. A count, or an example known to live under a later block, closes it.
  - id: new
    severity: Minor
    family: helper-duplicated-in-package
    title: |
      renderSpans duplicates marked from highlight_test.go in the same package
    detail: |
      cmd/define/usage_test.go:73 is a byte-for-byte reimplementation of `marked`, which is
      already in package main and directly usable — ARCH-DRY, in the milestone whose thesis
      is that the matcher must not be reimplemented.
  - id: new
    severity: Minor
    family: name-means-two-things
    title: |
      store.NewsItem.Source is the publisher while Usage.Source is provenance
    detail: |
      usagesFrom writes `Publisher: it.Source` — the same identifier carries two meanings one
      line apart, and #10 will read both types. Renaming the store field Publisher removes
      the trap before it has a second reader.
  - id: new
    severity: Minor
    family: matcher-limit-silently-drops-input
    title: |
      A punctuated deck key can never yield a usage, and nothing says so
    detail: |
      Verified: containsWord returns false for "e.g.", "9/11" and "rock 'n' roll" even when
      the text contains them verbatim, inherited from #21's phraseRunsJoin bound on
      MaxPhraseWords. In #21 the cost was "not highlighted"; here it is "this word can never
      be taught from a real sentence", which is a different stake and is documented in
      neither the code comment nor the atlas.
  - id: new
    severity: Minor
    family: plan-record-drift
    title: |
      Every plan checkbox is unticked at the boundary, and five prose claims contradict the code
    detail: |
      workshop/plans/000009-vocab-news-plan.md: all of Task 1's and Task 2's steps are
      unticked while the issue marks M1 [x]. The prose also states Usage{...Title...}
      (shipped: Publisher), store.NewsItem{Title, URL, At} (shipped: plus Source),
      entryUsages(e Entry) (shipped: plus word), a "substring" fuzz property (shipped:
      subsequence, correction recorded in the Log but not the plan), and Task 2 Step 5's
      "12-99 RANGE" which is impossible against a 13-item fixture — the code's exact-10
      assertion is the better call and should be recorded as such.
```
