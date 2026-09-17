---
id: 000068
status: working
deps: [tools#67, tools#73]
github_issue:
created: 2026-09-16
updated: 2026-09-17
estimate_hours:
started: 2026-09-17T10:56:31-07:00
---

# define: use real sentences as cloze material — finish the `usage/` thread

## Problem

`define` caches real sentences for a word and never reads them.

`usage/<slug>.yaml` holds `store.NewsItem` records from Google News RSS, and
`entryUsages` supplies NOAD's own examples; `bothSources` (`usage.go:153`) joins
them behind one seam and is constructed into `deps` at `main.go:280` and `:389`.
**`Usages` has no caller** (`usage.go:161` — the only references are the interface,
the implementation and its own helper). The atlas says plainly: *"Nothing
user-facing consumes it yet, deliberately. #10's authoring step is the consumer."*
That consumer was never built.

Meanwhile `--harvest` pays a model call to AUTHOR a sentence for every deck word
(`renderAuthorPrompt`, `harvest_item.go:351`), then pays another to entailment-judge
it (`harvest.go:369`), for words that may already have real sentences sitting in
`usage/`.

#67 surfaced a third source: a passage the learner actually read and marked a word
in. It was deliberately kept out of #67 so this issue can serve all three sources
with ONE consumer rather than accumulating a second stalled producer.

## Spec

Settled by the 2026-09-17 brainstorm and revised after its spec review; see
`## Revisions` for what changed and why.

### What the evidence changed

The issue was filed on the premise *"three sources, one shape"* — `Usage` tagged
by provenance and a consumer that need not know which it is. That is true of the
struct and false of the material. `usagesFrom` (`usage.go:82`) sets `Usage.Text`
to the news **headline** (`it.Title`, attribution stripped), and the committed
capture at `testdata/news/ephemeral.rss` shows what that means. It holds 13
items, of which 10 contain the word (`usage_test.go:99` asserts exactly that). Of
those 10, one is a grammatical sentence that tests the meaning — *"Springtime is
ephemeral"* — and one is marginal (*"WhatsApp working on yet another ephemeral
feature on iOS"*: headline register, but the meaning does work). Two use the word
as a proper noun — *Ephemeral Technologies*, *The Ephemeral Players* — which
`containsWord` can never reject, because it normalises through `store.Key`. The
rest are title-case noun phrases and gerund titles whose blanks constrain nothing.

So the core question as filed — selection or repair — was aimed at the wrong half.
A judge cannot repair a title into a sentence.

**This is one word's fixture, so it is an operator decision rather than a measured
property of all headlines.** The decision: news is not a stem source. If it is
ever revisited, the measurement to make first is the same screening run across
several words, which this issue's machinery makes cheap.

### Stems: an ordered candidate list

`authoredStem.Stem` stops being one string and becomes the first survivor of an
ordered list. `runAuthoring` (`harvest.go:293`) grows a screening loop.

1. **A passage sentence** — prose the learner actually read, in the register they
   read it in. **This source has no producer yet; building it is in scope here and
   owned by M3 below.**
2. **A NOAD example** — a real fragment, grammatical, free, offline.
3. **An authored sentence** — the model, as today. Last, so "fall back to
   authoring" is the final element of a list rather than a special case.

**Screening is the EXISTING judge.** `entailTask(lang, word, stem)`
(`harvest.go:495`) is determined by its three arguments alone — it never sees the
item or the distractors — so a candidate is screened by the call that already
screens authored stems. `stemUsesTheWord` (`harvest_item.go:444`) runs first, free.

**Capped at `maxCandidates` = 3**, and the cap interacts with #73's suppression,
so the rule is stated rather than left to an implementer: screening takes the
first `maxCandidates` **not-yet-rejected** candidates in order, so the window
advances across runs instead of retrying the same three. A word whose every
candidate is rejected falls through to authoring; a word whose authoring is also
recorded as failed is skipped by #73's policy.

Each candidate costs one entail call through `runWithin`, drawn on the same budget
as the vetoes — and a budget cut mid-veto abandons the item whole
(`harvest.go:405`). Worst case per word rises from 5 calls to 9 (band, 3 entail,
author, entail, 3 veto), which **stales two comments that state the arithmetic**:
`harvestLimit`'s "a word costs between two and five calls" (`harvest.go:22`) and
`bgBudget`'s sizing at "roughly six calls each" (`background.go:25`). Both are
updated, and `bgBudget` is re-sized or the background harvester silently covers
~6 of its 10 words.

**Budget exhaustion mid-screening** falls through to the next candidate only if
budget remains; with none, the word is left pending exactly as today
(`harvest.go:373`) and, per #73, records nothing.

**A known limitation, stated rather than discovered:** a NOAD example comes from
the entry the learner just looked up, so its answer is briefly recallable from the
lookup rather than the meaning. Review is scheduled days later
(`cmd/define/schedule`), so the window is normally gone. A reason to rank NOAD
below passage, not to exclude it.

**Platform:** the dictionary is macOS-only (`dict_stub.go`'s
`unsupportedDictionary`), so NOAD candidates exist only on darwin. Off darwin the
list degrades to passage-then-authoring. This is a platform-conditional feature,
not a uniform one, and the tests say so.

### Reaching the two real sources

**NOAD needs an `Entry` the loop currently throws away.** `entryUsages(e Entry,
word)` (`usage.go:125`) and `UsageSource.Usages(ctx, word, e Entry)`
(`usage.go:47`) both require a parsed entry, but `runAuthoring` holds only
`gloss, _ := wordSense(d, c.Word)` (`harvest.go:360`), and `wordSense` parses the
entry and **discards** it, returning `(gloss, domain)` (`harvest.go:586`). So
`wordSense` returns the `Entry` too, rather than a second `ParseEntry` walk per
word — one parse, two consumers.

**A passage sentence has no producer at all, and that is M3.** `CaptureMarked`
records only the word — `c.record(word, store.EventMarked, true, opt)`
(`capture.go:215`) — and `store.ReviewEvent` has no field for a sentence. #67
deliberately deferred this. Three pieces have to exist:

- **A sentence segmenter.** `passage.lines` are wrapped to terminal width
  (`wrapPassageLines`, `passage.go:75`) and a `passageSpan` is a word range inside
  a wrapped line, so "the sentence the word was marked in" is currently undefined
  in this codebase. It must be defined over the passage's `src` (the unwrapped
  text), not its wrapped lines, or the sentence breaks where the terminal did.
- **A capture path** that records that sentence with the mark.
- **A persisted surface** for it, which brings `store.RuntimeDirs`
  (`store/yaml.go:57`) and its four non-compiler homes: `.gitignore`, both repo
  guards, `perWordDirs`/`Forget`, and the store-layout blocks in
  `cmd/define/README.md` and `atlas/define.md` (`doc_sync_test.go:667`).

The sentence is untrusted pasted text reaching a model prompt and a persisted
store, so it joins the `oneLine` class (`store/item.go:157`) like every other text
field on that surface (ARCH-SECURE).

### `Named` is waived for real sentences

The entail judge requires *"a real person, place or institution the reader can
picture"* (`harvest_judge.go:134`). That rule disciplines the MODEL — the author
prompt bans "a manager", "the company" (`harvest_item.go:407`) — and a sentence
somebody wrote has no such failure mode. So `Named` is enforced when the candidate
is `authored` and waived for `noad` and `passage`. `Entails` and `Glosses` apply to
every candidate unchanged: the appositive ban is what wild prose most needs.

**The waiver is decided at SCREENING time from the candidate in hand**, never read
back off a stored item. Nothing re-judges a stored item, so a stored provenance
value never reaches the waiver.

### Provenance on the item

`store.Item` gains one field, three values:

```go
Source Source `yaml:"source,omitempty"`   // authored | noad | passage
```

`news` is deliberately **not** a value: no path can produce it, and a field value
no row sets reads as coverage while being dead.

Two collisions to resolve rather than discover:

- `usage.go:16` already defines `usageNews`/`usageNOAD` as bare strings on
  `Usage.Source string`. The new enum is the single definition and `Usage` derives
  from it (ARCH-DRY) — two vocabularies for provenance is how they drift apart.
- In package `store`, `NewsItem.Source` already means **the publisher**
  (`usage.go:89` does `Publisher: it.Source`). A second `Source` with an unrelated
  meaning in one package needs the name settled; `Origin` on `Item` is the
  recommendation.

**Empty is a parse failure, not a legacy marker.** `sanitiseItem` degrades an
unrecognised `Form` to empty as a sentinel meaning *visibly unusable*
(`store/item.go:162`), and reusing empty for "legacy" would make a corrupt value
indistinguishable from an old one. Items already on disk predate the field and are
read as `authored`, which is what every one of them is — a migration-free default
that happens to be true, stated so nobody later reads empty as "unknown".

No URL field: the enum drives behaviour, and nothing yet reads which artifact it
was. The accepted cost is that a bad item cannot be traced to its source. This
supersedes `store/item.go:75` in one direction only — still true of `authored`,
now false of the other two.

### Out of scope

- **The failure record and the retry policy are #73.** This issue depends on it:
  screening a candidate that was already rejected is the same repeat-spend bug,
  and the suppression rule above reads #73's record.
- **The news half of `usage/` gets no consumer here**, so `bothSources` and
  `cachingFeed` remain uncalled in production after this issue. Feeding headlines
  to the author prompt as context was considered and declined: the cache is empty
  for every word (its only writer is `cachingFeed.items`, reachable only through
  `Usages`, which has no production caller), so reading it **fetches** — one
  Google News request per word, serially, with a 20s timeout (`news.go:48`) — and
  a time-varying cache in the prompt changes `llm.RequestHash`
  (`internal/llm/render.go:61`), invalidating the author goldens and cassettes.
  That is the same nondeterminism that ruled out the `web_search` server tool.
  Retiring the news half (ARCH-FUNERAL) or giving it a display consumer is a
  separate decision, and this issue's title overclaims until it is made.
- **The `web_search` server tool**: $10 per 1,000 searches, no tools field on
  `llm.Request`, a `pause_turn` resume loop neither `Complete` nor `Stream` has,
  an unverified interaction with `Task[T]`'s output schema, and the
  nondeterminism above.
- Repairing a sentence — trimming a gloss out of real prose re-introduces a model
  call and invents a sentence nobody wrote.
- Headline stems.

### What does not change

- **`pickDistractors` is untouched** — it takes no stem, in signature or body
  (`harvest_item.go:187`), so "distractor selection unchanged" holds by
  construction.
- **The veto still runs** per surviving candidate against the winning stem, the
  only stem its question is meaningful against.
- **No stem means no cloze question**, already true: `clozeFor` (`cloze.go:121`)
  picks the newest usable `FormCloze` item and a word with none is not offered in
  that form.

## Done when

Each row names the test obligation that pins it, per #67's close convention.

- A **NOAD example** reaches a stored item with no authoring model call, and the
  item carries `Origin: noad` — asserted by counting calls through the LLM fake,
  on darwin.
- A **passage sentence** reaches a stored item the same way, carrying
  `Origin: passage`.
- The segmenter splits over the passage's unwrapped `src`: a sentence that spans
  two wrapped lines comes back whole — the test wraps at a width that guarantees
  the split.
- A candidate that glosses its own word is rejected, with a fixture that is an
  appositive of the shape `renderAuthorPrompt` bans.
- `Named` is waived for `noad`/`passage` and enforced for `authored` — a table
  over the three values, with the waiver read from the candidate in hand and never
  from a stored item.
- Screening is capped at `maxCandidates` entail calls per run, counted; and the
  window **advances** — a word with 6 candidates and 3 recorded rejections screens
  candidates 4–6 on the next run, not 1–3 again.
- Off darwin the list degrades to passage-then-authoring with no NOAD candidate
  and no dictionary call.
- `wordSense` returns the parsed `Entry` and `runAuthoring` does **not** call
  `ParseEntry` a second time — asserted by counting dictionary lookups per word.
- The veto saw the stem that won, not a stem that lost.
- `Origin` round-trips through `sanitiseItem`; an unrecognised value degrades and
  does not store; an item written before the field reads as `authored`.
- `RuntimeDirs` gains the passage-sentence surface and all four non-compiler homes
  are green: `TestGitignoreCoversRuntimeDirs`, both repo guards,
  `TestStoreLayoutDocsNameEveryRuntimeDir`; `Forget` removes it via `perWordDirs`.
- A pasted sentence carrying an ANSI escape is stored without it.
- `storetest` covers the new surface for both implementations.
- `harvestLimit`'s and `bgBudget`'s call-arithmetic comments are updated to the
  new worst case, and `bgBudget` is re-sized — asserted by a test that the
  background harvester still covers `bgThreshold` words.
- **The model-call accounting is measured before and after and reported in
  whichever direction it goes.** Screening can cost more calls than authoring one
  stem; the issue claims a quality change, and a saving is a finding rather than a
  premise.
- `atlas/define.md` records the candidate order, the provenance enum, the
  segmenter, and the passage-sentence store layout.

## Plan

Durable plan to be authored via `superpowers-writing-plans` into
`workshop/plans/000068-real-sentences-cloze-plan.md`. Blocked on #73, whose record
the screening suppression reads.

- [x] brainstorm: selection vs repair (2026-09-17 — answered; see `## Revisions`)
- [x] spec review (2026-09-17 — three factual errors and one missing producer;
      see `## Revisions`)
- [ ] `sdlc start-plan`, then the durable plan
- [ ] M1 — the store: `Origin` on `Item`, `ParseOrigin`, sanitise-on-write, the
      `authored` default for pre-field items, `Usage` derived from the same enum,
      `storetest` for both implementations
- [ ] M2 — the candidate list with NOAD as its first working source: `wordSense`
      returns the `Entry`, the screening loop, the cap and its advancing window,
      the `Named` waiver, the two stale call-arithmetic comments
- [ ] M3 — the passage producer: a sentence segmenter over `src`, the capture
      path, the persisted surface + `RuntimeDirs` and its four homes
- [ ] atlas, then `sdlc close`

## Log

### 2026-09-16

Split out of #67 during its planning. #67 originally carried a droppable task to
write a marked word's sentence straight into `store.Item{Form: FormCloze}` — the
`Stem` IS the example sentence there (`harvest.go:445`), so the datatype fit. Two
things killed it as a #67 task: distractors are not supplied by a sentence, and the
gloss problem above means a wild sentence needs the judge more than an authored one
does. Investigating that turned up the unused `usage/` cache, which is the same
problem one layer down — hence one issue, not two.

Corrects a note still visible in #67's history: *"an authentic sentence is better
material than an authored one."* It is more REAL, not automatically better as a
stem, and the difference is exactly the appositive ban.

## Revisions

### 2026-09-17 — brainstormed; scope settled, and the premise corrected

**Reason.** The Spec as filed asked one question — selection or repair — and it
was aimed at the wrong half of the material. Reading the committed feed capture
(`testdata/news/ephemeral.rss`) rather than reasoning about the sources showed
that the news half of `usage/` stores article TITLES: of twelve items, one is a
usable sentence and two use the word as a proper noun that `containsWord` can
never reject. Neither selection nor repair rescues that.

**Delta from the filed Spec.**

- *"Three sources, one shape"* is withdrawn as a design premise. It holds for the
  struct and not for the material: a headline, a dictionary fragment and a read
  sentence are different things, and only the third is stem material.
- **Selection, not repair** — and the selector is the judge that already exists.
  `entailTask` is determined by the stem text alone, so screening a real candidate
  needs no new judge. Repair is a stated non-goal.
- **News is not a stem source.** It becomes authoring CONTEXT instead, which is
  where it earns its keep: it makes the `Named` requirement satisfiable, which the
  model currently cannot satisfy for recent entities from training data alone.
  This is what gives `Usages` the production caller the issue was opened to
  deliver.
- **`Named` is waived by provenance** (operator decision, 2026-09-17). It
  disciplines the model, and a sentence a person wrote has no such failure mode.
- **Provenance is added to `store.Item`** as a parsed enum (operator decision).
  Empty means legacy and does not inherit the waiver. No URL field.
- **Failure recording and a retry policy are IN SCOPE** (operator decision, and
  explicitly kept in this issue rather than split out). Today every word that ever
  failed is re-asked on every subsequent run with an identical prompt, because
  nothing records the attempt — so the record is the mechanism that makes "do not
  ask again" expressible, not bookkeeping. Retry defaults to OFF: the operator's
  stated preference is to fail closed and simply not offer a cloze question.
- **The `web_search` server tool was considered and declined** on cost ($10/1,000
  searches), new transport work (`llm.Request` has no tools field; `pause_turn`
  needs a resume loop), an unverified interaction with the `Task[T]` schema, and
  nondeterminism in a cassette-and-golden suite.
- The economics claim is inverted from the filing. The issue said harvest pays to
  author and then to judge; measured, a fresh word costs 2–5 calls (band, author,
  entail, up to three vetoes), and screening replaces only the author call while
  paying entail per candidate. Any saving is now a finding to report, not a
  premise — which is why the retry policy, not the sentence source, is where the
  spend actually falls.

### 2026-09-17 — spec review: three factual errors, and a source with no producer

A fresh-context spec review checked every claim against the code. Most held;
these did not, and two of them were load-bearing.

- **The `passage` source, ranked first, has no producer.** `CaptureMarked` records
  only the word (`capture.go:215`) and `store.ReviewEvent` has no sentence field;
  #67 deferred this deliberately. The Spec's *"nothing else about the pipeline
  moves"* was false — a segmenter, a capture path and a persisted surface all
  have to move, and no milestone owned them. Now M3, with the segmenter's
  hardest part named: it must split over the passage's unwrapped `src`, because
  `passage.lines` are wrapped to terminal width.
- **`allVetoed`'s retry trigger was backwards.** The draft said `len(kept) == 0`
  means a starved pool, not a model failure. But `tierAboveBand` accepts every
  word (`harvest_item.go:259`), so `pickDistractors` never returns empty in
  production and an empty `kept` means the veto refused everything — a model
  decision. The `PoolSize` field and its exclusion from `model-changed` are
  dropped; they would have withheld retry from exactly the case a model change
  fixes. Now #73's problem, corrected there.
- **The fixture numbers were wrong.** The Spec said "twelve items, one usable".
  It is 13 items, 10 containing the word (`usage_test.go:99`) — the count came
  from grepping `<title>`, which caught the channel title. Corrected, and the
  conclusion is now stated as an operator decision from one word's fixture rather
  than a measured property of headlines.
- **"The cache we already own delivers the same grounding for nothing" was
  false.** `usage/` is empty for every word — its writer is reachable only
  through `Usages`, which has no production caller — so reading it fetches, one
  Google News request per word. And a time-varying cache in the author prompt
  invalidates the goldens and cassettes, which is the same argument used to
  decline `web_search`. Headlines-as-authoring-context is withdrawn, and the news
  half of `usage/` is left with no consumer: the issue's title overclaims until
  retiring it or giving it a display consumer is decided.

Also corrected: `renderAuthorPrompt` is at `harvest_item.go:351`, not `:370` (the
citation was wrong as filed); the provenance comment is `store/item.go:75`, not
`:73`; `news` is dropped from the enum as a value no path can produce; the enum
collides with `usage.go:16`'s strings (now derived from it) and with
`NewsItem.Source` meaning publisher, so `Origin` is the recommended name; empty is
a parse sentinel rather than a legacy marker, with pre-field items read as
`authored`; NOAD needs the `Entry` that `wordSense` discards; the dictionary is
darwin-only so NOAD candidates are platform-conditional; screening raises the
worst case per word to 9 calls, staling the arithmetic in `harvestLimit`'s and
`bgBudget`'s comments.

**Scope.** The failure record and retry policy moved to #73 (operator decision,
2026-09-17), which this issue now depends on. They share no code path, #73 is pure
cost control that lands immediately, and #68 turned out to need producer work #73
should not wait behind.

