---
type: project
name: "define-learn"
goal: "Make define an adaptive vocabulary trainer on three verbs — definition, pronunciation, free-form learning — where a frontier model authors practice material from current usage and adapts it to a durable model of this learner."
done_when: "A real day of use runs end to end on generated material: words captured by ordinary lookup; practice items authored from current usage and stored offline; a review session played and scored; a free-form question answered in the console with the session's own words as context; and user-model.md regenerated from the resulting events, visibly steering the next batch of items."
status: defined
created: 2026-08-20
updated: 2026-08-22
mvp_scope: ["tools#5", "tools#6", "tools#7", "tools#8", "tools#9", "tools#10", "tools#11", "tools#12", "tools#13", "tools#16", "tools#17"]
explicitly_out_note: "tools#18 (Spanish) is real and filed, but OUT of this project's MVP: define-learn is done when the English loop works end to end. #18 M1 (audio) is independently shippable at any time."
explicitly_out: ["multi-learner accounts", "sync/replication beyond whichever directory you run it in", "languages other than English", "generated (TTS) pronunciation — recorded audio only", "a GUI or mobile client"]
sources: ["workshop/parley/ — define-learn ideation", "operator conversation 2026-08-22 (adaptive scope)"]
---

# define-learn

`define` becomes a trainer that meets the learner where they are: type a word and
get a definition, type a question and get an answer, sit down with `--play` and
review material a frontier model authored from how those words are actually being
used this week. **Not in MVP: any of it working for a second person.** The learner
model, the deck, the authored items and the corrections are one person's, in one
directory, and the design leans on that — the words you look up are the level
signal a generic vocabulary app does not have.

## PRD

### Three verbs on one noun

The whole product is three things you can do with a word, and the boundary between
them is invisible to the user — input is free-form and `define` classifies it,
rather than the learner selecting a mode.

| verb | how | availability |
|---|---|---|
| **definition** | NOAD via macOS `DCSCopyTextDefinition` | offline, instant, no key |
| **pronunciation** | recorded Oxford audio via the CDN | offline after first fetch |
| **free-form learning** | frontier model: ask anything, get authored practice, be modelled as a learner | needs the seam; degrades to the first two |

The first two already ship. This project is the third, plus the retention loop
(deck, schedule, review, stats) that gives the third something to adapt to.

### The adaptive loop

This is the thesis. Each arrow is a real artifact on disk, not an abstraction:

```
ordinary lookups ─┐
                  ├─→ user-model.md ─→ authored items ─→ review session ─→ events ─┐
review events ────┘        ▲                                                       │
                           └───────────────────────────────────────────────────────┘
```

- **Lookups reveal the domain.** A learner who looks up `certiorari`, `dicta` and
  `arguendo` reads judicial opinions. Nobody asked them; the deck says so. Their
  distractors and comparables should come from that register, not from a generic
  frequency band.
- **Misses reveal the weakness.** Batch-analysed, not diagnosed per-answer: the
  question is *what kind of thing does this learner get wrong* (near-synonym
  collapse, connotation, register, domain), and that is a pattern across many
  events, not a property of one.
- **The model steers authoring.** `user-model.md` is an input to every authoring
  prompt, which is what makes the next item adapted rather than merely generated.

### How the model is leveraged — and where it is not

Deliberately generous with the model where judgment is needed; deliberately
absent where a deterministic answer exists. Cost is not a constraint (operator,
2026-08-22): access is through the local `cli-proxy-api` on a subscription plan,
and the goal is the best material achievable, not the cheapest.

| task | where | why a model |
|---|---|---|
| author a practice item | `#10`, offline batch | raw headlines are not questions — see below |
| classify a word's level, register, domain | `#10`/`#17`, cached per word forever | frequency lists cannot see register |
| veto a distractor | `#12` | a yes/no on a concrete pair is checkable; open generation is not |
| grade a written sentence | `#13` | the only judgment a local rule genuinely cannot make |
| batch-analyse errors into a learner model | `#17` | pattern-finding across many events |
| answer a free-form question | `#16` | this is the verb |

**Not the model's job:** the definition (NOAD is better and offline), the
pronunciation (a recording is ground truth), the schedule (Leitner is explainable
in one sentence; an ease factor is not), and *which real words the distractors are
drawn from* (see the decision below).

### Why authoring is a real job, and measured

Measured 2026-08-22 against the live feed for `sycophantic` (100 items):

- **The feed collapses thematically.** 10 of the first 14 matching headlines were
  about AI chatbots. Drill on that and the learner acquires the collocation
  *"sycophantic AI"*, not the word.
- **Headlines are not stems.** Blanking *"Sycophantic AI decreases prosocial
  intentions"* yields a question the sentence does not entail — unanswerable from
  its own context, so it teaches nothing.

So the model must read *across* usages and write a stem that entails its answer:

> The board meeting produced nothing but ______ agreement — every executive
> praised a plan they had privately called unworkable.

That is authoring. It is also why authoring belongs in the **async harvest step**
rather than at question time: a session then costs nothing, waits on nothing, and
works with the network off.

### Decisions

Appended, never overwritten — original intent stays visible.

**2026-08-20**

- **Storage is YAML files in the working directory.** `define` resolves no brains,
  workspaces or home directories, and invokes no git of any kind. The original
  "git repo / nous push" framing meant only *YAML rather than a database*;
  replication is whichever directory you run it in.
- **One file per word plus an append-only day log.** That does not make sync
  conflicts impossible — the same word, or the same day, on two machines still
  conflicts. It changes the *rate*: with a single `vocab.yaml` every write on a
  second machine conflicts, because every write touches the one file.
- **Distractors are selected, never invented.** The pool is news-harvested words at
  the learner's level plus the learner's own deck, filtered for substantial
  semantic difference. The model only *vetoes* a candidate that would also fit.
  This removes the "the LLM's wrong answer is also right" failure mode.
- **Google News RSS, not the SERP.** Measured: 41–100 items per word. The SERP was
  measured too — a 91 KB JS shell with zero usable content.
- **Clock injected everywhere.** Spaced repetition is date-driven; "due today" is
  untestable against a wall clock.
- **Every LLM/network feature degrades.** No key, no network → `--play` falls back
  to the local forms rather than failing.

**2026-08-22 — the adaptive scope**

- **The model authors the stem; the options are still selected.** Broadens the
  2026-08-20 rule rather than reversing it: the model reads many real usages and
  writes a clean, self-entailing stem, but the four options still come from the
  level-matched pool and the learner's deck, and the model still only vetoes.
  Two-right-answers stays impossible by construction rather than by a check.
- **Cost is not a constraint; quality is the goal.** Access via the local
  `cli-proxy-api` (running on `127.0.0.1:8317`) against a subscription plan, with a
  direct API key as fallback. Use frontier models generously.
- **A single `user-model.md`, batch-generated, human-correctable.** One durable
  markdown artifact holding level, domains read, and weaknesses. Regenerated by
  batch analysis; a `## Corrections` section is human-owned, never rewritten, and
  authoritative over anything inferred.
- **Free-form input is a first-class verb.** A question typed at the prompt routes
  to the model with the session's recent words as context; the classifier asks NOAD
  first, so "is this a headword" is a free, offline, deterministic signal. One
  decision table, not a second parser.
- **The LLM harness is repo infrastructure (`internal/llm`).** Operator override,
  2026-08-22: `AGENTS.local.md` says `internal/` is earned on the *second*
  consumer, and this creates it for the first. Recorded rather than done quietly —
  a transport with auth, retries, a stateful fake and a live conformance check is
  the kind of thing the next tool would otherwise copy.

## Estimate

Not yet costed. Per ariadne #113/#187 the estimate is derived at `sdlc change-code`
per issue — after the plan clears the plan-quality gate, when scope is knowable —
not guessed at project definition.

## Breakdown

The ordered list is execution order. It **departs from the 2026-08-20 sequencing**,
which put the offline trainer first: the goal then was "a working trainer", and the
goal now is "how good can the material get". So the harness and the authoring
pipeline come first, and there is a deliberate stop at `#10` to read generated
items before building the forms that consume them.

`#16` sits early because it is the cheapest thing that makes the tool better daily,
and it exercises the harness end to end on a real task before anything depends on it.

`#17` also lands early in a reduced form: **domain and level fall out of lookups
alone**, which already exist — only the weakness taxonomy needs review events. So
authoring is learner-aware from the first generated item, and `#17 M2` deepens it
once `#6` is producing misses.

- [x] define REPL — bare invocation reads, defines, speaks; bare return replays [tools#2]
- [x] REPL line editor — history, prefix search, inline autosuggestion [tools#14]
- [x] vocabulary store — Store seam, YAML in the working directory, clock injected [tools#3]
- [x] capture on lookup — successful lookups build the deck [tools#4]
- [x] REPL command mode — `/`-commands with type-ahead, starting `/history` [tools#15]
- [x] LLM harness — transport, wire fake, obligation suite [tools#11 M1]
- [x] LLM harness — typed tasks, goldens, conformance, `--llm-check` [tools#11 M2]
- [x] free-form Q&A — the console knows a question from a word [tools#16 M1]
- [x] free-form Q&A — the answer: context pack, streaming, scoped Ctrl-C [tools#16 M2]
- [x] learner model — `user-model.md` from lookups; batch analysis [tools#17 M1]
- [x] news seam — Google News RSS (not the SERP) [tools#9]
- [x] scheduling engine — Leitner, pure [tools#5]
- [ ] `--play` loop + form 2.1 [tools#6]
- [ ] form 2.3 — meaning multiple choice, deck distractors, no LLM [tools#7]
- [ ] item authoring + harvest — async, level-aware, learner-aware, stores finished items [tools#10]
- [ ] `--stats` — all derived from the event log [tools#8]
- [ ] form 2.2 — cloze from authored items, distractors **selected not invented** [tools#12]
- [ ] form 2.4 — free sentence, graded [tools#13]
- [ ] learner model — weakness taxonomy from review events, steers authoring [tools#17 M2]
- [ ] Spanish — pronunciation locale (independently shippable) [tools#18 M1]
- [ ] Spanish — language-aware deck + agreement-safe distractors [tools#18 M2]

<a id="tools-11-m1"></a>
### tools#11 M1 — transport, wire fake, obligation suite

**est:** 7.98 (whole issue)
**actual:** 3.45h
**closed:** 2026-08-22

`internal/llm` exists: a provider-independent `Client` over `anthropic-sdk-go`
pointed at the parley-managed proxy, a five-member error taxonomy, pure config
resolution, a wire-level stateful fake, and one obligation suite that runs against
both the fake and (under `-tags conformance`) the live service. Prompts
deliberately live with consumers, not here.

The decision worth preserving is where the test double sits: an httptest server
speaking the Anthropic protocol, **not** a stubbed `Client`. Placement decides
what a test can see, and a stubbed client sits above every bug this harness can
actually have. Content in tests comes from committed captures rather than
literals — you cannot fake judgment, but you can freeze a real answer.

Three surprises, all from measurement rather than reasoning. `claude-opus-5`
returns a **thinking block first** on any non-trivial prompt, so `content[0]` is
not the text — and one capture has a thinking block *after* the text, so no
sequence may be asserted at all. A **truncated structured answer parses**
(`{"verdict":"yes","reason":": Ā"}`, every required field present), so the stop
reason must be checked before decoding; that specimen is preserved as a fixture.
And the **proxy answers 502 for an unknown model** where the direct API answers
400, so an our-bug-class error is absorbed as `ErrUnavailable` — recorded as a
known limitation rather than papered over.

Cost note for calibration: `sdlc actual` measured 3.45h against a window whose
wall clock is 1.83h (`b5d50ea2` 17:20 → `bd94021` 19:10). The measured value was
recorded rather than a hand-typed one, but it is **1.9× the window it names**, so
this row should not be treated as clean evidence for the ledger.

<a id="tools-11-m2"></a>
### tools#11 M2 — typed tasks, goldens, conformance

**est:** 7.98 (whole issue)
**actual:** 3.75h
**closed:** 2026-08-23

A consumer now writes a prompt and a result type and gets a typed answer:
`SchemaFor[T]` reflects the schema from the struct, `Run[T]` calls and decodes,
and the stop reason is checked **before** decoding because a truncated structured
answer parses cleanly.

The decision worth preserving is that goldens and cassettes are two views of one
request, both deriving from a single `renderRequest`. Two renderers would have
drifted in the worst direction — a prompt edit visible in the golden while a stale
cassette kept matching — so it landed before either consumer.

The cassette is the answer to "how do you mock an LLM": you don't. You freeze a
real response, key it by the request hash, and a miss fails loudly rather than
falling back, because falling back is how an edited prompt comes to pass against a
recording of the question it no longer asks.

`decode` needed one real fix: "another token" is not "consumed cleanly" —
`dec.Token()` returns an *error* for trailing prose, so `{...} — hope that helps!`
was accepted with a populated value. 1.2M fuzz executions now hold the invariant.

**Calibration caveat, stronger than M1's.** `sdlc actual` reports 7.20h for the
issue, but its attribution warnings show it reaching into sessions from 2026-07-27
and 2026-08-20/21 — days before this issue was claimed — and the window is shared
with [tools#18]. Two spans are attributed to #11 by *mention fallback* rather than
by commit boundary. The increment above is derived by subtraction from a
measurement, not typed, but this row should not be treated as clean ledger
evidence.

<a id="tools-16-m1"></a>
### tools#16 M1 — the console knows a question from a word

**est:** 6.49 (whole issue)
**actual:** 1.46h
**closed:** 2026-08-23

Typing a question at the prompt is now understood as one, in all three entry
modes, and answered with an honest "no model configured" until M2 wires the seam.
The classifier is the part worth preserving: **the dictionary is the classifier.**
Word count cannot be the signal — `hot dog` is a two-word headword and
`defenestrate` is one word — so the tool asks NOAD first and classifies only what
it misses. `hot dog`, `a priori` and `use` stay lookups because the dictionary
says so, not because a predicate was careful. One decision table in two pure
halves (`parseREPLLine` syntactic, `readsAsQuestion` semantic), asserted end to
end by `TestConsoleDecisionTable` rather than half-by-half — the half-by-half
version proves each half correct and leaves the table unasserted.

A draft had a fourth arm, "≥5 words → question", to catch `difference between
sycophantic and obsequious`. Operator call: that is a word count wearing a
different hat, and the spec rejects word count. It was dropped, the cost named in
the atlas, and `?` is the recovery.

Two defects were caught **at the plan gate rather than in code**, which is the
strongest argument for the gate this project has produced. PQ-3: the ask outcome
carries exit code 0 — a question is not a failed lookup — so `if out.code == 0 {
current = word }` was exactly wrong, and a question would have become the word a
bare Enter replays. PQ-1/PQ-6: the scoped-interrupt design (M2's) rested on
`rawterm.go`'s comment that Ctrl-C is a byte in raw mode, while the pty suite's
own header records the measured opposite — and the first fix would have left every
piped run uninterruptible. Three plan rounds, zero of those found by running the
code.

Cost note: 1.46h measured against a window that is mostly design — the plan and
its three gate rounds sit inside it. The first estimate (3.72) priced only the
tasks and was revised to 6.49 for exactly that reason before any code was written.

<a id="tools-16-m2"></a>
### tools#16 M2 — the answer

**est:** 6.49 (whole issue)
**actual:** 4.88h
**closed:** 2026-08-24

The third verb works. A question goes to the model with the DIRECTORY as its
context — the word on screen and its entry, the session's lookups, the deck,
`user-model.md`, and the session's earlier exchanges — and the answer streams
back interruptibly.

The adaptation is visible in the output, which is the only place it counts:
against the live proxy with a two-line learner model ("B2, reads business news,
weak on near-synonym distinctions"), the answer came back with a *"Business-news
nuance"* paragraph and *"Related near-synonyms in your range"*, and quoted the
NOAD entry back at the learner. A follow-up resolved against it.

Two decisions worth not re-deriving. **The interrupt swallow lives in the
reader**: an interrupt a scope consumed must not also be delivered as a key, or
the loop quits the session the moment the answer ends — and the alternative, the
loop racing for keys during the stream, silently ate type-ahead. That made the
key channel's buffering load-bearing. **The prompt renderer returns an
`llm.Request`**, so the golden is what the transport actually sends rather than a
string assembled for the test.

The bug worth remembering was invisible to every M1 test: `ask()` short-circuited
the FORCED route to "no model configured", so `?why` never reached the seam. M1
had no seam to reach, so nothing could have caught it there — it surfaced the
first time a test waited for a stream that was never requested.

Filed out of this milestone rather than fixed in it: [tools#19] — the proxy
answers **200 with an error body** on overload, which the taxonomy reads as our
bug rather than an unavailable service.

<a id="tools-10"></a>
### tools#10 — item authoring + harvest

**status:** open — the material-quality checkpoint

Broadened 2026-08-22: the harvester no longer maintains a *word pool*, it produces
**finished, verified practice items** with provenance, ahead of time and offline.
Consumes `user-model.md` so items are learner-aware from the start. This is where
the project's central question gets answered — stop here and read the output before
building the forms that consume it.

<a id="tools-17-m1"></a>
### tools#17 M1 — the learner model, from lookups

**est:** 8.39 (M1)
**actual:** 3.8h
**closed:** 2026-08-26

`define --reflect` writes `user-model.md`: a working level and the domains the
learner reads in, each naming the deck words it was read off, each carrying what
authoring should DO about it. Batch and on demand — no model call moved onto the
lookup path.

The decision worth not re-deriving: **evidence is checked, not trusted.**
`checkEvidence` drops any claim citing a word the deck does not hold — the
*selected, never invented* rule arriving in its second place. It grew a second
arm from a measured failure: under a schema requiring every field the model fills
the ones it does not believe in, so a domain with no name or no directive is
dropped too. A claim authoring cannot act on is not a claim.

Three bugs that only RUNNING it found, with the unit suite green throughout: the
model put prose in the `evidence` array (renaming the JSON field to
`evidence_words` fixed it — the field name is what steers); it stubbed claims it
did not believe in; and it wrote an empty file when everything was dropped, which
reads as an answer. Two floors now, both preferring nothing to something
confident.

Measured honestly at the end: with the model present an answer leads with the
right domain and reaches into the deck's structure, but WITHOUT it the answer is
already good, because #16 already sends the deck. Depth and ordering, not a
different topic — the `directive` fields are aimed at #10, which is where the
payoff is designed to land.

<a id="tools-17-m2"></a>
### tools#17 M2 — weakness taxonomy

**status:** blocked — needs review events from [tools#6]

<a id="tools-18-m1"></a>
### tools#18 M1 — Spanish pronunciation locale

**status:** open — independently shippable, blocked on nothing

`AudioCandidates` hardcodes `_en_`, so a Spanish word is only ever requested as an
English one. Measured: `madrugar_en_us_1` 404s while `madrugar_es_es_1` and
`madrugar_es_us_1` both return 200. The locale is phonemically load-bearing here
rather than cosmetic — `es_es` is Castilian /θ/, `es_us` is *seseo* — and since
Spanish orthography is phonemic, dictionary entries carry no phonetic notation at
all, so the recording is the only place that information exists.

<a id="tools-18-m2"></a>
### tools#18 M2 — language-aware deck, agreement-safe distractors

**status:** blocked — needs [tools#10] and [tools#12]

The one that would otherwise break #12: Spanish leaks answers grammatically.
*"La actitud del jefe era claramente ______"* eliminates every masculine option
without the learner knowing a single word's meaning, so distractor selection needs
a gender/number agreement filter beside the semantic-distance one.

## Log

### 2026-08-22 — scope event: the project became adaptive

Original goal (2026-08-20): *"turn `define` from a lookup tool into a retention
tool"* — deck, schedule, forms, stats. Model use was scoped narrowly and
defensively: veto a distractor, grade a sentence, nothing else.

Operator broadened it in conversation on 2026-08-22. Three additions, none of which
the original nine issues covered:

1. **The model authors the material.** Real usage is raw input, not the question;
   producing something worth answering is a real authoring job. Measured against the
   live feed the same day — see PRD.
2. **A durable learner model steers it.** One `user-model.md`, batch-generated from
   lookups and misses, holding level, domains read and weaknesses — and used as an
   input to authoring, so distractors for a reader of Supreme Court opinions come
   from that register.
3. **Free-form input is a first-class verb.** Typing a question at the prompt is as
   ordinary as typing a word. "Meet the user where they are, rather than the user
   conforming to a certain way to use it."

Also settled: cost is not a constraint (subscription via local `cli-proxy-api`), and
the `internal/` first-consumer rule is explicitly overridden for the harness.

Consequences recorded elsewhere: `## Revisions` on [tools#10], [tools#11] and
[tools#12]; new issues [tools#16] and [tools#17]; `done_when` above rewritten (the
old one said results are *"persisted to the brain"*, contradicting this file's own
2026-08-20 decision that `define` resolves no brains — stale phrasing from the
original framing, now removed).

Task list also brought current: [tools#4] and [tools#15] were `done` and archived but
still showed open here.

<a id="tools-9-m1"></a>
### tools#9 M1 — real sentences, filtered by the matcher that highlights them

**est:** 8.22 (whole issue)
**actual:** 0.9h (M1)
**closed:** 2026-08-26

`parseRSS` turns a Google News feed into `store.NewsItem`s; `usagesFrom` and
`entryUsages` turn those — and NOAD's own examples — into one `Usage` shape
tagged by provenance. The feed is the current half and the dictionary is the
durable half, which matters because the feed's thematic collapse is measured:
ten of fourteen `sycophantic` headlines were about AI chatbots, so a word sourced
only from this week would be taught narrowly.

**The decision worth not re-deriving: raw is cached, usable is derived.** Only
the feed's own words go to disk. Everything the learner sees is computed at read
time, so improving the filter improves every word already cached with no
re-fetch — caching the filtered result would freeze today's judgment. The plan's
first draft put `Usage` in `package main` while the store returned it, which
cannot compile; the Critical that forced the split was the design telling the
truth.

**And `containsWord` delegates to `highlightSpans` rather than matching itself.**
The filter and #21's highlighter must never disagree about the same text, or the
learner sees a headline whose word renders green while the filter rejected that
headline — the same word in two states on one screen. That agreement has its own
test.

Two things capturing a REAL feed taught that reasoning about one did not: every
title carries a `" - Publisher"` suffix (stripped only when it is the publisher
the feed named, so a headline with an internal dash keeps it), and a naive first
fixture was ten copies of one article. The committed fixture is deliberately
adversarial to its own test — 10 matches across 10 publishers plus 3
non-matches, so the filter has work to do.

<a id="tools-5-m1"></a>
### tools#5 M1 — the schedule, derived from the log rather than stored beside it

**est:** 5.56 (whole issue)
**actual:** 1.2h (M1)
**closed:** 2026-08-26

Leitner boxes (1/3/7/14/30/90 days) with `Fold`, `Due`, `Answer` and `Mastered`,
in a new `schedule` package that imports `store` and pure standard-library
packages only, enforced by two guards rather than asserted.

**The decision worth not re-deriving: schedule state is DERIVED from the event
log, never stored on the word.** `store/event.go` had already written the rule for
the whole store — the log is the only record of activity, and counters kept
alongside become a second source of truth that drifts — so a `Box` field on
`store.Word` would have been exactly that, going wrong invisibly whenever a
hand-edited deck file disagreed with the events that produced it. `store` gained
no fields.

Two things surfaced that were not about scheduling at all. `#15` had needed
"which local day is this" twice for `/history` and written it inline twice in two
DIFFERENT shapes; `#5` needing it a third time is what moved it into
`store.StartOfDay` beside the `Clock`. And the plan claimed the package's purity
was self-enforcing because "a test needing a fake would not compile" — false, a Go
test may import anything — so the claim now has a guard that reads the import set.
This repo keeps being bitten by facts that live only in comments.

[tools#2]: #tools-2
[tools#3]: #tools-3
[tools#4]: #tools-4
[tools#5]: #tools-5
[tools#5 M1]: #tools-5-m1
[tools#6]: #tools-6
[tools#7]: #tools-7
[tools#8]: #tools-8
[tools#9]: #tools-9
[tools#9 M1]: #tools-9-m1
[tools#10]: #tools-10
[tools#11 M1]: #tools-11-m1
[tools#11 M2]: #tools-11-m2
[tools#12]: #tools-12
[tools#13]: #tools-13
[tools#14]: #tools-14
[tools#15]: #tools-15
[tools#16]: #tools-16-m1
[tools#16 M1]: #tools-16-m1
[tools#16 M2]: #tools-16-m2
[tools#17 M1]: #tools-17-m1
[tools#17 M2]: #tools-17-m2
[tools#18 M1]: #tools-18-m1
[tools#18 M2]: #tools-18-m2
[tools#19]: ../issues/000019-llm-overloaded.md

### 2026-08-26 — scope event: two console features shipped alongside, outside MVP

Neither is in `mvp_scope` and neither changes the done-when. Recorded because the
project is the portfolio view and both shipped inside its window, on operator
request, touching the same console the project's own verbs live in.

- **tools#20 — typeahead beyond the first word.** The grey suggestion matched
  only whole past lines, so it died at the first space — exactly where a
  free-form question gets asked, which is `#16`'s verb. Now completes a deck word
  anywhere in the line. est 5.24 / actual 1.35.
- **tools#21 — highlight the words you are learning.** Words from the deck render
  bold green wherever they appear: the line you type, definition bodies, and
  streamed answers. Reinforcement at the moment of reading rather than only at
  review time, so it sits beside the retention loop rather than inside it.
  est 9.28 / actual 6.0.

**Why they matter to this project rather than merely coinciding with it.** Both
make the *lookup* surface teach, which is the loop's entry point — the deck is
built by ordinary lookup (`#4`), and these two put the deck back on screen during
ordinary lookup. #21 also lands a `Vocabulary` predicate seam that
**tools#22** (words graduating out of highlighting) will narrow; #22 is filed,
out of MVP, and blocked on the review signal from `#5`/`#6`.

### 2026-08-26 — scope event: retention loop before the authoring loop

**Reason.** Operator, after #9 closed: *"I think we can rearrange, to have some
scheduling engine first, so that I can use this to remember words, and practice
them, before making it better through google news and other integrations?"*

**Delta.** `#5` → `#6` → `#7` move ahead of `#10`. Nothing else changes: same
issues, same MVP scope, same done-when.

**Why it works, checked rather than assumed.** The dependency graph already
allowed it — `#5` needs only `#3` (done), `#6` needs `#5`, `#7` needs `#6` — and
the two forms in that path were specified from the start to need neither. Form
2.1: *"No question generation, no network."* `#7`'s Problem: *"the whole review
loop should work with no API key and no network."* So this ordering reaches a
learner who can actually practise, using the deck ordinary lookup has already
built, with no key and offline.

**Why it is the better order, not merely a possible one.** The PRD's loop is
`lookups → user-model → authored items → review → events → user-model`. Building
the authoring half first would have produced items with **no review events to
learn from** — the arrow back into the model would have had nothing on it, and
`#17 M2`'s weakness taxonomy is blocked on exactly those events. Doing retention
first closes the small loop (lookup → schedule → review → events) and gives the
authoring half a learner to adapt to when it arrives. `#9` was not wasted: it is
the input `#10` needs, and it is done and cached.
