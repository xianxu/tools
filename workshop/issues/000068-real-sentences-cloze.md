---
id: 000068
status: working
deps: [tools#67]
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
(`renderAuthorPrompt`, `harvest_item.go:370`), then pays another to entailment-judge
it (`harvest.go:369`), for words that may already have real sentences sitting in
`usage/`.

#67 surfaced a third source: a passage the learner actually read and marked a word
in. It was deliberately kept out of #67 so this issue can serve all three sources
with ONE consumer rather than accumulating a second stalled producer.

## Spec

Settled by the 2026-09-17 brainstorm; see `## Revisions` for what this replaced
and why.

### What the evidence changed

The issue was filed on the premise *"three sources, one shape"* — `Usage` tagged
by provenance, and a consumer that need not know which it is. That is true of the
struct and false of the material. `usagesFrom` (`usage.go:82`) sets `Usage.Text`
to the news **headline** (`it.Title`, attribution stripped), and the committed
capture at `testdata/news/ephemeral.rss` shows what that means: of twelve items,
one ("Springtime is ephemeral") is a grammatical sentence usable as a stem. Two
use the word as a proper noun — *Ephemeral Technologies*, *The Ephemeral Players*
— which `containsWord` cannot catch, because it normalises through `store.Key` and
so always matches. The rest are title-case noun phrases and gerund titles whose
blanks constrain nothing.

So the core question as filed — selection or repair — was aimed at the wrong
half. A judge cannot repair a title into a sentence, and selecting from that pool
yields about one candidate per word. **Headlines are not stem material, and no
amount of judging makes them so.**

They are, however, good *authoring context*. See below.

### Stems: an ordered candidate list

`authoredStem.Stem` stops being one string and becomes the first survivor of an
ordered list. `runAuthoring` (`harvest.go:293`) grows a screening loop; nothing
else about the pipeline moves.

Order, and the reason for each position:

1. **A passage sentence** (#67's third source) — prose the learner actually read,
   in the register they read it in. The only source that is both authentic and
   sentence-shaped.
2. **A NOAD example** (`entryUsages`, `usage.go:133`) — a real fragment,
   grammatical, free, offline.
3. **An authored sentence** — the model, as today. Last, and therefore the
   fallback; "fall back to authoring" becomes the final element of a list rather
   than a special case.

**Screening is the EXISTING judge, not a new one.** `entailTask(lang, word, stem)`
(`harvest.go:495`) is fully determined by the stem text — it never sees the item
or the distractors — so a candidate is screened by the call that already screens
authored stems. `stemUsesTheWord` (`harvest_item.go:444`) runs first, free, as it
does today.

**Capped at `maxCandidates` = 3.** Each candidate costs one entail call through
`runWithin`, which charges the same budget the vetoes draw on — and a budget cut
mid-veto abandons the item whole (`harvest.go:405`), because an item is cached
forever and a short write permanently costs the third option. Unbounded screening
would therefore starve the part of the pipeline that finishes the item. A word
with twelve NOAD examples screens three of them.

**A known limitation, stated rather than discovered:** a NOAD example comes from
the entry the learner just looked up, so its answer is briefly recallable from the
lookup rather than from the meaning. Review is scheduled days later
(`cmd/define/schedule`), not immediately, so the window is normally gone. This is
a reason to rank NOAD below passage, not to exclude it.

### `Named` is waived for real sentences

The entail judge requires *"a real person, place or institution the reader can
picture"* (`harvest_judge.go:134`). That rule exists to discipline the MODEL — the
author prompt bans "a manager", "the company" (`harvest_item.go:407`) — and a
sentence somebody actually wrote has no such failure mode. So `Named` is enforced
when the candidate's provenance is `authored` and waived otherwise. `Entails` and
`Glosses` apply to every candidate, unchanged: the appositive ban is exactly what
wild prose most needs.

### Provenance on the item

`store.Item` gains one field:

```go
Source Source `yaml:"source,omitempty"`   // authored | news | noad | passage
```

A closed enum with `ParseSource` following `ParseForm` (`store/item.go:57`), and
`sanitiseItem` degrading an unrecognised value rather than storing it. **Empty
means legacy** — every item already on disk has no source — and the `Named` waiver
must NOT apply to empty, or every pre-existing authored item silently loses the
rule it was written under.

No URL field. The enum answers "how should this be treated", which is what drives
behaviour; the URL would answer "which artifact was it", and nothing yet reads
that. Recorded as a deliberate omission: it means a bad news-sourced item cannot
be traced to its headline, which is the cost being accepted.

This supersedes `store/item.go:73` in one direction only. That comment says a
model-authored stem has no source to inspect, so the text is all there is — still
true of `authored`, and now false of the other three.

### The news cache becomes authoring context

This is `Usages`' production caller, and the issue's stated purpose.

`renderAuthorPrompt` (`harvest_item.go:351`) gains the word's cached headlines as
context, with an instruction to draw real names from them where they fit. The
`Named` requirement is currently aspirational — the model must supply current real
entities from training data, which for anything recent it cannot, and `Named` is
one of the three booleans that then rejects the stem. Headlines make it
satisfiable.

The thematic collapse that ruins headlines as teaching material (*"ten of fourteen
`sycophantic` headlines were about AI chatbots"*, `usage.go:145`) is harmless here:
the model is mining them for names, not for a sentence.

**Explicitly NOT the `web_search` server tool.** Considered and declined: $10 per
1,000 searches on top of tokens, new transport work in `internal/llm` (no tools
field on `llm.Request`, and `pause_turn` needs a resume loop neither `Complete`
nor `Stream` has), an unverified interaction with the `Task[T]` output schema, and
nondeterminism in a suite built on captured cassettes and request-hashed goldens.
The cache we already own delivers the same grounding for nothing.

### Failures are recorded, and NOT retried by default

Today a rejected word keeps no item, so the `len(existing) > 0` skip
(`harvest.go:342`) does not fire and the next run re-asks with a byte-identical
prompt — forever, for every word that ever failed. Nothing records the attempt, so
"do not ask again" is not currently expressible.

A new per-word record, sibling to `WordFacts` and `Items`, stored per language:

```go
type Attempt struct {
    At       time.Time
    Form     Form          // cloze today; #13's sentence form later
    Source   Source        // which candidate failed
    State    AttemptState  // closed enum, ParseAttemptState like ParseForm
    Digest   string        // identity of a REAL candidate's text; empty for authored
    Provider string        // llm.ModelSelection, via clientModelSelection
    Model    string
    PoolSize int           // banded words available at the time
    Reason   string        // the judge's clause, currently printed and dropped
}
```

Not folded into `WordFacts`, whose doc says its fields "are one judgement about
the word and they expire together (never)" (`store/item.go:10`). An attempt record
is precisely the thing that does expire.

`AttemptState` is the five ways an item dies today, which do **not** share a retry
trigger:

| state | today's site | what would change the answer |
| --- | --- | --- |
| `malformed` — stem lacked the word | `harvest.go:362` | a different model |
| `notEntailed` | `harvest.go:382` | a different model |
| `glossed` | `harvest.go:382` | a different model |
| `unnamed` | `harvest.go:382` | a different model |
| `noDistractors` | `harvest.go:438` | **a larger banded pool** — not the model |

`noDistractors` is not a stem failure at all: `pickDistractors` requires
`Facts.Harvested()` on every candidate, so a small or partly-banded deck starves
it and re-authoring fails identically forever. `PoolSize` is what makes its retry
trigger measurable instead of guessed.

**Budget exhaustion is NOT a failure.** `markUnfinished` (`harvest.go:274`) already
draws this line — a rejection counts, a budget cut does not, "so the word stays
pending for the next pass". Recording a budget cut as a failure would suppress a
retry that must happen.

**Two caching semantics, because the two kinds of candidate differ.** A real
sentence is stable text, so its verdict is permanent: keyed by `Digest` (a
truncated SHA-256 of the normalised text — identity, not security), it is never
screened twice. An authored stem is a fresh sample per run, so its record is only
ever "this model failed here, at this time", keyed by word plus `(Provider,
Model)`.

**Retry policy, default off** — the point is controlling model spend:

- default — a word whose candidates are all recorded as rejected is skipped,
  silently, and simply has no cloze item.
- `--retry-failed=model-changed` — re-attempt authoring where `(Provider, Model)`
  differs from the client's current selection. Real-candidate verdicts are still
  final; their text did not change.
- `--retry-failed=all` — re-attempt everything.

"Older than N" is deliberately absent and costs nothing to add later: the record
already carries `At`.

### What does not change

- **`pickDistractors` is untouched** — it does not take the stem, in signature or
  body, so "distractor selection unchanged" holds by construction rather than by
  test.
- **The veto still runs** per surviving candidate, against the winning stem, which
  is the only stem its question is meaningful against.
- **No stem means no cloze question**, already true: `clozeFor` (`cloze.go:121`)
  picks the newest usable `FormCloze` item and a word with none is not offered in
  that form. Nothing to build.

### Non-goals

Repairing a sentence (trimming a gloss out of real prose) — it re-introduces a
model call and invents a sentence nobody wrote. Headline stems. The `web_search`
server tool. A second producer of any kind.

## Done when

- A passage sentence reaches a practice item with **no authoring model call**, and
  the stored item carries `Source: passage`.
- `Usages` has a production caller: the word's cached headlines reach
  `renderAuthorPrompt`, asserted on the REAL request through the LLM fake, not on
  the code that builds it.
- A candidate that glosses its own word is rejected, with a fixture that is an
  appositive of the shape `renderAuthorPrompt` bans.
- `Named` is waived for `passage`/`noad`/`news` and enforced for `authored` — a
  table over all four values **plus empty**, since a legacy item must not inherit
  the waiver.
- Screening is capped: a word with more candidates than `maxCandidates` makes
  exactly `maxCandidates` entail calls, counted through the fake.
- The veto still runs against the winning stem, and `pickDistractors` is
  unchanged — it takes no stem, so this is a compile-level property plus a test
  that the veto saw the stem that won.
- **A recorded failure suppresses the next attempt:** two consecutive `--harvest`
  runs over the same deck make model calls on the first and **zero** on the
  second, counted through the fake. This is the cost behaviour the issue exists to
  fix, so it is asserted by call count, not by inspecting a record.
- `--retry-failed=model-changed` re-attempts authoring when `(Provider, Model)`
  differs and not when it matches; a real candidate's verdict stays final under
  every policy, because its text did not change.
- A budget cut writes **no** `Attempt` record — asserted, since recording it would
  suppress a retry that must happen.
- `noDistractors` records `PoolSize` and is not retried by `model-changed`.
- The model-call accounting is **measured before and after, and reported in
  whichever direction it goes.** Screening candidates can cost more calls than
  authoring one stem; the issue claims a quality change, and any saving is a
  finding rather than a premise.
- `atlas/define.md` records the candidate order, the provenance enum, the retry
  policy, and where the headline context enters the prompt.

## Plan

Durable plan to be authored via `superpowers-writing-plans` into
`workshop/plans/000068-real-sentences-cloze-plan.md`. Four review boundaries —
the store format, the screening loop, the record + policy, and the authoring
context — each closing with its own `sdlc milestone-close`.

- [x] brainstorm: selection vs repair (2026-09-17 — answered; see `## Revisions`)
- [ ] `sdlc start-plan`, then the durable plan
- [ ] M1 — the store: `Source` on `Item`, `Attempt` + `AttemptState`,
      `ParseSource`/`ParseAttemptState`, sanitise-on-write, `storetest` coverage
      so the fake cannot hold a state the real store cannot
- [ ] M2 — the candidate list: ordered sources, the screening loop capped at
      `maxCandidates`, the `Named` waiver by provenance
- [ ] M3 — the attempt record wired into every rejection site, the digest for
      real candidates, and `--retry-failed`
- [ ] M4 — cached headlines into `renderAuthorPrompt`; `Usages` gets its caller
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

