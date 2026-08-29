---
id: 000026
status: working
deps: []
github_issue:
created: 2026-08-27
updated: 2026-08-28
estimate_hours: 1.57
started: 2026-08-28T17:57:39-07:00
---

# NOAD ratchet regressed 27 to 32: a British-English pronunciation block renders raw

## Problem

`TestRenderLosesNothingOverLiveEntries` carries a ratchet — a measured count of
live entries that still render unconverted NOAD notation, held at
`knownRawNotationEntries = 27`. Run against the live dictionary it now reports
**32**, and fails:

```
live_property_test.go:127: 32/73502 live entries rendered unconverted NOAD
    notation, up from the known 27 — a regression
```

Two of the three named entries are a shape the ratchet's own comment does not
cover:

```
Brent: unconverted NOAD notation survived, near
  "brent\n\n    | AmE brɛnt, BrE brɛnt | noun (British English) "
brent: (identical)
charge: "…“he charged me”\n    2. euros for the postcard | the restaurant
  charged $15 for dinner | [no obje…"
```

`charge` is the documented cause — *"a prose numeral that happens to continue a
sense sequence is accepted as a sense number"*. `Brent`/`brent` are not: they are
a **dual-pronunciation block**, `| AmE … , BrE … |`, which the renderer leaves in
raw pipe notation.

## Spec

**Triaged 2026-08-28, and the triage answered both questions by removing them.**

`#23 M2` made English select `com.apple.dictionary.NOAD` plus
`com.apple.dictionary.AppleDictionary` **by identifier**, where before it passed
NULL and searched every ACTIVE dictionary on the host. `Brent` was a
British-dictionary entry reachable only through that search. Verified: `define
brent` now reports **no dictionary entry** at all.

**Positive control, because a negative probe alone proves nothing** (PQ-6). In
the same session and the same build: `define charge` returns an entry with no
`no known en dictionary is installed` warning, so the curated path IS being
exercised and `brent`'s absence is a fact about the dictionaries rather than
about a broken lookup context.

So the dual-locale block is not fixed — **its input is gone**. Both Spec
questions (converter vs splitter; render both pronunciations or the deck's) are
moot, and so is `deps: [tools#27]`: they existed only to decide how to render a
shape that no longer arrives.

**It is DORMANT, not dead, and that is the thing to write down.** `curated["en"]`
is `{NOAD, AppleDictionary}`. Adding Oxford Dictionary of English — a plausible
future entry, it is strictly monolingual `en` and passes every metadata filter —
brings `| AmE … , BrE … |` straight back, along with this defect. The resurrection
condition belongs in the record so the next person adding a curated dictionary
meets it.

### What the re-measurement actually found

Row 3 is the real work, and it was larger than "re-count". The ratchet samples
only the first three survivors, so 23 of them had never been looked at. Enumerated
in full against the live dictionary, 2026-08-28 — **26 entries, four causes, not
one**:

| cause | n | entries |
|---|---|---|
| a prose numeral continuing a sense sequence — *the documented cause* | 7 | `charge`, `chargee`, `charging`, `depth`, `just`, `justness`, `shortness` |
| a pronunciation glued to the headword — `(aˈhəndrədzˈhəndrəd/)` | 7 | `hundred`, `hundredfold`, `million`, `millionfold`, `millions`, `thousand`, `thousandfold` |
| a phrase-block pronunciation run into prose — `lick someoneˌoud əv ˈSHāp/` | 8 | `shape`, `shapable`, `Shape`, `shaper`, `shaping`, `short`, `shorter`, `shortish` |
| **a literal `|` that IS the content** — *"• the symbol \|."* | 4 | `pipe`, `piped`, `pipeful`, `pipeless` |

`ratio`, `glop` and `logarithmic` — which `atlas/define.md` names as the same
shape — are **no longer in the list, and they are NOT unreachable**. Verified:
all three still return entries. They render CLEAN now. A first draft of this Plan
said "unreachable", which is a different claim and a wrong one (PQ-1).

Only 7 of 26 are the cause the ratchet's own comment claims covers all of them,
and `atlas/define.md` repeats that attribution.

**The oracle is a DISJUNCTION of two, not the pipe check a first draft of this
Spec claimed.** Read at `cmd/define/live_property_test.go:84`:

```go
if near, bar := strayStress(out), strings.IndexByte(out, '|'); near != "" || bar >= 0 {
```

`strayStress` (`cmd/define/invariant_test.go:42`) strips `/…/` spans and then
looks for a stress mark `ˈ`/`ˌ`. So the headword-glued and phrase-block groups —
`(aˈhəndrədzˈhəndrəd/)`, `lick someoneˌoud əv ˈSHāp/` — fire on STRESS MARKS and
carry no pipe at all. A pipe-keyed classifier would mis-handle most of the
population Done-when row 3 wants attributed. The per-oracle split is deliberately
NOT asserted here; it is measured during implementation, because asserting it is
how this Spec was wrong twice already.

**The `pipe` family is a false positive, and the oracle stays broad anyway.**
Those four entries define the pipe character, so their content legitimately
contains one. Narrowing the oracle to
exclude them is exactly the move `workshop/lessons.md` records as having already
failed here — the narrow oracle read 0% while 2.2% of entries showed raw pipes,
and that false 0% was published in the atlas. So the count stays 26 and the four
are ATTRIBUTED rather than excluded, which is what "attributable rather than
absorbed" asks for.

## Done when

- [x] The `| AmE … , BrE … |` shape is DISPOSITIONED rather than ticked: recorded
      as unreachable on the curated path — with its positive control — and with
      the condition that resurrects it (a British dictionary entering
      `curated["en"]`) written where the next person adding one will meet it.
- [x] The ratchet is re-measured and `knownRawNotationEntries` lowered to the new
      true count — not raised to make the test pass.
- [x] **The run itself names which group moved.** Each survivor is classified by
      cause and the per-cause counts are reported, so a future 27 does not print
      one number and three samples and send the next person back to raising the
      diagnostic cap by hand — a cost this issue already paid once.
- [x] The classifier is testable WITHOUT the live dictionary: one real captured
      exemplar per cause, so the taxonomy is not conformance-only (ARCH-MOCK).
- [x] Every measured number in the docs is either DERIVED from a producer with a
      doc-sync test, or DATED as a measurement — the rule, not the six sites.
      The count has drifted three times, which is this repo's own trigger for
      making a doc a consumer.
- [x] The live test's own comment claiming "there is no public API to select
      one" is corrected — untrue since `#23 M2`, and it sits in the file this
      issue edits.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.15 impl=0.20
item: smaller-go-module        design=0.05 impl=0.16
item: smaller-go-module        design=0.05 impl=0.12
item: smaller-go-module        design=0.05 impl=0.10
item: real-api-discovery       design=0.00 impl=0.20
item: cross-cutting-refactor   design=0.05 impl=0.12
item: atlas-docs               design=0.04 impl=0.06
item: milestone-review         design=0.00 impl=0.16
design-buffer: 0.15
total: 1.57
```

Derivation notes.

- **This was 0.65 and is 1.22, and the gate is why.** The first block priced
  "lower a constant and correct a comment". PQ-4 and PQ-5 pointed out that
  Done-when row 3 asks the next drift to be ATTRIBUTABLE, and a comment cannot
  enforce that — so the deliverable is a classifier with per-cause reporting plus
  offline exemplars, not a number. Three `smaller-go-module`s: the classifier and
  its reporting, the exemplar corpus and its unit test, and the atlas doc-sync
  test.
- **`cross-cutting-refactor` for the artifact sweep.** The sites are whatever the
  grep returns at implementation time, deliberately not counted here: a first
  draft of this note said "six sites across two files", which is a hand
  enumeration surviving inside the block that claims to have replaced hand
  enumerations. Priced as the shape `#24`'s call-site change was, not as a
  one-liner.
- **`issue-spec` impl is 0.20, not 0.05, and the correction is the interesting
  one.** The first block defended 0.05 with "the triage was spent before this
  block existed" — but `sdlc actual` anchors on the CLAIM commit, and four
  plan-gate rounds have landed since it. Pre-claim triage is genuinely excluded;
  post-claim gate iteration is not, and nothing was pricing it. The design half
  stays 0.15 because the triage really did happen before the claim.
- **`real-api-discovery` for the live convergence loop**, which nothing covered.
  The Plan commits `causeUnclassified` to ZERO across the live population, so the
  classifier must be iterated against 26 survivors it has never run over, at ~51s
  per unsandboxed cycle against a host-dependent dictionary. That is probing an
  external system from outside the test suite — this vocabulary's closest
  primitive — and it is convergence, not the writing the classifier item prices.
- **One `milestone-review`** at 0.16 rather than the 0.20 ceiling: the diff is a
  test, a fixture directory and doc corrections.
- Library-availability check: nothing external; no halving applies.

Σdesign 0.39 × 1.15 = 0.4485; Σimpl 1.12; total **1.57**.

## Plan

Plain checkboxes, one `sdlc close`: one coherent deliverable across a test file,
a fixture directory and two docs is a single review boundary, not two.

- [x] **Move the pinned count and the classifier into an UNTAGGED file**
      (`rawnotation_test.go`, package `main`). `live_property_test.go` is
      `//go:build darwin && conformance`, so nothing in the normal suite can see
      `knownRawNotationEntries` today — which is why the atlas cannot consume it
      and why a live-only taxonomy would be untestable off-conformance. Untagged
      files compile into every build, so one producer then serves three
      consumers: the live ratchet, the classifier's unit test, and the doc-sync
      test below.
- [x] **`classifyRawNotation(rendered string) rawCause`** — pure, untagged,
      returning exactly one of a closed set: `causeProseNumeral`,
      `causeHeadwordPronunciation`, `causePhrasePronunciation`,
      `causeLiteralPipe`, `causeUnclassified`.
      **Totality is the design, not a detail:** every survivor gets exactly one
      cause, and `causeUnclassified` is a real bucket the live run asserts is
      ZERO. A new shape then surfaces as unclassified rather than being absorbed
      into whichever existing cause it happens to resemble — which is the failure
      mode this issue exists to end.
      It must key on BOTH oracles (see the Spec): a pipe-keyed classifier
      mis-handles the stress-mark majority. Risky inputs a table would miss: an
      entry whose content legitimately contains a pipe, and an entry matching two
      causes at once — hence "exactly one", by this precedence:
      **stress-mark causes outrank pipe causes.** A stress mark surviving outside
      a `/…/` span is unambiguously leaked notation; a pipe can be legitimate
      content (`pipe` defines the character). So an entry firing both is a
      pronunciation leak that happens to contain a pipe, not the reverse.
- [x] **One real captured exemplar per cause** in `testdata/rawnotation/` —
      deliberately NOT under `testdata/entries/<lang>/`, which `capturedLanguages`
      walks and over which `TestNoRawPronunciationNotationSurvives` asserts a hard
      zero. Captured through the curated identifiers like every other fixture, so
      the classifier is pinned offline (ARCH-MOCK).
- [x] **Lower the pinned count to the measured value** and rewrite its comment
      with the four causes.
- [x] **Make the derivable numbers CONSUMERS, and DATE the rest.** See the rule
      below; on the `TestREADMEQuotesThePromptsTheLoopActuallyPrints` precedent.
- [x] Re-run the live ratchet unsandboxed; green in BOTH directions.

### The rule this issue is fixing, not the sites

Second finding in the `doc-sweep-incomplete` family, so the deliverable is the
rule. A measured number stated in ANY artifact — a code comment as much as the
atlas, the README, an issue or a plan — may exist in exactly two forms:

1. **DERIVED** — the doc quotes a value the code owns, and a doc-sync test
   asserts it. EVERY figure, not just a headline one: marking only the total let
   a compensating swap (one cause up, another down) keep the whole suite green
   while two atlas numbers were wrong. Applies to the pinned raw-notation count and the captured-fixture
   count, both of which a producer can supply in the normal suite.
2. **DATED** — the doc states it as *"measured YYYY-MM-DD: N"*. Applies to the
   live sweep width and the non-Latin count, which only a conformance run against
   the host's dictionaries can produce. A dated measurement is a RECORD and
   cannot go stale; an undated one reads as a standing fact and drifts silently,
   which is what happened to all four.

3. **HISTORICAL** — past-tense narration of a number that was true then and is
   being reported as a past state ("the atlas published 0% while 2.0% of entries
   were still rendering raw pipes"). A record, in the sense `#23` established.

Anything that is none of the three is the defect. That is what makes this a rule rather
than a sweep: the atlas's *"every reachable entry — 70,897"* was wrong because it
was written as a fact, not because nobody re-ran the number.

**Scoping it to "a document" was itself the second miss.** `cmd/define/parse.go:535`
carries *"27 of 70,897 live entries (0.04%)"* in a CODE comment — same number,
same staleness, outside a rule that said "document". Artifacts, not documents.

### And a second rule, because the same shape produced two wrong claims

**Every statement about existing behavior in a Spec or Plan carries a `file:line`
and is read against it before being written.** Two of the two behavioral claims
this gate checked were wrong: *"ratio/glop/logarithmic are no longer reachable"*
(falsified by running the code) and *"the check is `IndexByte(out, '|')`"*
(falsified by reading it). Both were written from memory of code I had touched
hours earlier, which is exactly the condition under which memory feels reliable.

**Non-goal, stated rather than left implicit:** the `pipe` family is not excluded
from the count. Those four entries define the pipe character, so their content
legitimately contains one — but narrowing the oracle is the move `lessons.md`
records as already having failed here (the narrow oracle read 0% while 2.2% of
entries showed raw pipes, and that false 0% was published in the atlas). They are
ATTRIBUTED instead, which is what "attributable rather than absorbed" asks for.

## Log

### 2026-08-27

- **Found by running the conformance suite UNSANDBOXED**, which is the point of
  the suite and had not happened in the sessions around #24. Sandboxed, this test
  reaches 0 live entries and skips (or, before #24, hard-failed as "sandboxed?"),
  so the ratchet had effectively never run.
- **Confirmed PRE-EXISTING, not introduced by #24.** Ran the same test in a
  worktree at `87e6307b` (the #24 branch point): identical `32/73502`, same three
  named entries. #24 touched no rendering code.
- The 5-entry gap (27 → 32) is only partly enumerated: three entries are named in
  the failure output because the test reports "a handful, not thousands". A full
  list needs the report cap raised for one run.
- Reproduce:
  ```sh
  go test -tags conformance ./cmd/define/ -run TestRenderLosesNothingOverLiveEntries -v
  ```
  Must run unsandboxed; takes ~51s over 73502 live entries.

### 2026-08-28

- **Blocked on [tools#27]** (pronunciation locale as a parameter), split out of
  #18 the same day for this reason among others. Deliberately NOT started: the
  cheap fix here is to widen the notation converter to swallow
  `| AmE …, BrE … |`, and that would bake in "show both" as a decision by
  accident, at the exact moment the codebase is about to gain a real locale.
- The ratchet stays red until then, which is correct — it is reporting a true
  regression, and the number must not be raised to 32 to silence it.

### 2026-08-28 — re-measured after `#23 M2`; the cause is gone and the pin is now too high

`#23 M2` made English select `com.apple.dictionary.NOAD` plus
`com.apple.dictionary.AppleDictionary` **by identifier**, instead of passing NULL
to `DCSCopyTextDefinition` and searching every ACTIVE dictionary on the host.

That removes this bug's cause rather than fixing its symptom. `Brent`'s
`| AmE brɛnt, BrE brɛnt |` is a British-dictionary shape — it was reachable only
because the NULL search consulted whatever the host had enabled. Re-measured
against the live dictionary, 2026-08-28:

```
checked 70886 live entries: 0 lost content, 26 kept raw notation;
0 non-Latin (other active dictionaries), 165090 absent
```

- **26, not 32** — and below the pinned 27, so the ratchet now fails in the GOOD
  direction: *"only 26 … below the pinned 27 — lower knownRawNotationEntries to
  lock the improvement in"*.
- **`0 non-Latin`**, where the previous run had a class of them. The three sampled
  survivors are now all `charge`/`chargee`/`charging` — the prose-numeral cause
  the ratchet's own comment already documents, not a new shape.
- **The sweep narrowed**, and that qualification belongs here rather than being
  discovered later: 70,886 entries checked against 73,502, with 165,090 absent,
  because two dictionaries are asked rather than all of them. Some of the drop is
  "we stopped looking at entries we never serve." It is still better
  proportionally — 0.037% against 0.044% — so the improvement is real, but it is
  not purely a parsing gain.

**What is left of this issue:** lower `knownRawNotationEntries` from 27 to 26 and
close it. The uncovered-shape half of the Problem no longer reproduces.

**Its `deps: [tools#27]` is now moot** — the dependency was on `#27` owning
locale as a parameter, and the British-English entry that motivated it is
unreachable on the curated path.

### 2026-08-28 — implemented; the classifier agrees with the hand enumeration

The live run, unsandboxed, after the work:

```
checked 70886 live entries: 0 lost content, 26 kept raw notation; 0 non-Latin, 165090 absent
  raw notation by cause: prose-numeral            7
  raw notation by cause: headword-pronunciation   7
  raw notation by cause: phrase-pronunciation     8
  raw notation by cause: literal-pipe             4
  raw notation by cause: unclassified             0
```

**`classifyRawNotation` reproduces the hand enumeration exactly** — 7/7/8/4 — and
`causeUnclassified` is 0, so the taxonomy is total over the live population. That
agreement is the evidence the classifier is right; the hand grouping was done
first and independently, from the enumerated snippets.

**Two things the implementation found that the Plan had wrong.**

`strayStress` windows the STRIPPED text (`slashSpan.ReplaceAllString`), so its
return does not exist verbatim in the rendered entry. A first classifier searched
for it with `strings.Index(rendered, …)` and silently got `-1` every time; only
the headword check was carrying `hundred`. The classifier now strips in the same
way, so its index is in the same coordinate space as the oracle's.

The headword/phrase discriminator was a guessed fraction (`idx < len/3`) and
misclassified `shape`. Measured instead: `hundred` puts the stress at byte 27 of
1828, `shape` at 2436 of 4116 — two orders of magnitude apart, so
`headwordBlockBytes = 128` sits nowhere near either population.

**The pin is per-cause, not a total.** A single number is a weak ratchet: two
causes can move opposite ways and leave it unchanged, and a misclassification
between two known causes is invisible in a sum. `knownRawNotationEntries` is now
DERIVED from `knownRawByCause`, so the two cannot disagree.

**Mutation-checked.** Moving a cause count reddens the atlas doc-sync by name;
disabling the stress-mark half of the classifier reddens the exemplars.

### 2026-08-28 — close review round 2: seven findings, and BR-1 was the real one

**BR-1: the totality assertion was DEAD, and I reported its passing as evidence.**
Both branches of the first classifier were catch-alls — a stress mark that was
not headword-shaped fell into *phrase*, a pipe that was not the literal case fell
into *prose numeral* — so every oracle survivor landed in a named bucket BY
CONSTRUCTION. `causeUnclassified` could not fire for any input the live run
passes it, so `unclassified == 0` was vacuously true. A new shape would have been
absorbed into whichever bucket it resembled, which is precisely the failure this
classifier was written to end, implemented as its opposite. Probed and confirmed
before fixing.

Every cause is a POSITIVE test now — headword position, a trailing slash in the
stress window, a sense-number-then-pipe regex, `the symbol |` — with
`causeUnclassified` as the residue. Verified reachable for three novel shapes,
including this issue's own dormant `| AmE brɛnt, BrE brɛnt |`: if a British
dictionary is ever re-added, it surfaces as unclassified rather than being
miscounted as a prose numeral.

**The live population is unchanged and the breakdown is now earned:** 7 / 7 / 8 /
4, unclassified 0, over 70,886 entries. Same numbers as before the fix — but
before, the zero could not have been anything else.

**BR-4 falsified a claim in the previous Log entry.** It said "moving a cause
count reddens the atlas doc-sync by name". The reviewer swapped two causes,
keeping the total at 26, and the whole suite stayed green while two atlas numbers
were wrong. Only the total had a marker span. Every cause has one now and the
doc-sync loops `rawCauses`; the reviewer's exact mutation reddens both.

**BR-6 was a rule I had just written in `#23` and broke here.** The exemplar
corpus had no path in `capture.sh` and no conformance check — a fake with no live
check becomes a fossil, and a macOS update fixing one of these shapes upstream
would have left the classifier green against text the tool no longer produces.
`capture.sh` now captures them through the same curated books, and
`TestRawNotationExemplarsMatchLiveDictionary` byte-compares each one and asserts
it still trips the oracle.

**BR-7: the oracle was written in three spellings** across `invariant_test.go`,
`live_property_test.go` and `render_test.go`, and its body was copy-pasted into
the classifier — three chances for the ratchet, the corpus test and the
classifier to disagree about what they count. `strayStressAt` and
`rawNotationNear` are the shared pair now.

**BR-5: the resurrection condition sat only in this issue,** which archives to
`workshop/history/`. It is a comment above `curated` now — the exact line someone
edits when adding a dictionary — and it says what will happen (new entries report
as unclassified) rather than only that a risk exists.

**BR-2, BR-3: the DERIVED-or-DATED rule reached its named sites, not its class.**
Four more undated standing measurements remained, including a third sweep width
and a claim that both oracles "measure 0" twenty lines above the count that
falsifies it. Swept, and the rule gained a third legitimate form — HISTORICAL,
for past-tense narration — so the exception is stated rather than unexplained.

