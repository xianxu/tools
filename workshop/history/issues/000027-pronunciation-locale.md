---
id: 000027
status: done
deps: [tools#23]
github_issue:
created: 2026-08-28
updated: 2026-08-28
estimate_hours: 0.93
started: 2026-08-28T00:38:19-07:00
actual_hours: 1.89
---

# pronunciation locale and language as parameters, not literals

## Problem

Split from **#18 M1**, which said it plainly: *"M1 is independently shippable and
does not wait on #10."* #18's remaining scope (deck language dimension,
inflection/lemma identity, gender, agreement-safe distractors) is blocked on
`#10`/`#12`; this half is blocked on nothing, and two other issues are waiting on
the concept it introduces.

`AudioCandidates` (`cmd/define/audiourl.go:41`) writes the language as a
**literal**:

```go
out = append(out, audioBase+"/pronunciation/2022-03-02/audio/"+shard+"/"+esc+"_en_"+locale+"_"+n+".mp3")
```

So `madrugar` is only ever requested as an English word. Measured on 2026-08-22 —
all four candidates 404 while the recording sits two characters away:

```
madrugar_en_us_1.mp3   404      ← what define asks for
madrugar--_us_1.mp3    404
madrugar_es_es_1.mp3   200      ← what exists
madrugar_es_us_1.mp3   200
```

Coverage is real, not incidental: `sobremesa`, `empalagoso`, `chapucero`,
`desvelarse` all return 200 on `es_es`.

## Spec

Carried verbatim in substance from #18 M1.

**The locale is not cosmetic, and this is the part worth designing around.**
Spanish orthography is phonemic — spelling plus the written accent determines
pronunciation exactly — so Spanish dictionary entries carry **no phonetic
notation at all**, unlike NOAD's `lig·a·ment | ˈliɡəmənt |`. Verified on the live
entries. That means:

- `ParseEntry` finding no pronunciation on a Spanish entry is **correct**. Do not
  "fix" it.
- The recording is therefore the *only* place pronunciation information exists,
  which makes audio more load-bearing for Spanish than for English.
- The two locales encode a genuine phonemic split, not an accent flavour: `es_es`
  is Castilian (*cazar* /θ/ ≠ *casar* /s/), `es_us` is Latin American *seseo*
  (both /s/). Choosing one chooses which sound system a learner acquires.
  `es_419`, `es_mx` and `es_ar` are **not** valid — measured 404.

Design: language becomes a parameter beside locale. `AudioCandidates` already
returns an *ordered* list that `httpAudioSource` walks until the first 200, so
widening it costs no new mechanism — an unspecified language can try several, and
`-lang es` reorders rather than restricts. A conformance assertion belongs beside
`TestCDNStillServesTheExpectedPaths`, which pins the English ordering the same
way.

### `#29` consumes this policy, and a borrowing shows why (added 2026-08-28)

`#29` — origin pronunciation for borrowed words — **depends on this issue**, and
the reason is concrete. Measured:

```
jalapeño_es_es   200      ← Castilian
jalapeño_es_us   200      ← seseo
```

Knowing that `jalapeño` is a Spanish word does not determine which recording to
fetch: both exist, and they differ by exactly the phonemic split this issue is
about. So the locale policy decided here is consumed there rather than
reinvented — which is the argument for keeping it a POLICY with one home, not a
default buried in a URL builder.

**A caveat on this Spec's own claim, so `#29` does not inherit it wrongly.**
Above, this issue states that Spanish entries carry *no phonetic notation at
all* — verified, and true of **Spanish dictionary** entries. It is not true of a
Spanish word looked up in an ENGLISH dictionary: NOAD gives `jalapeño` four
pronunciations, all anglicised (`/ˌhaləˈpān(y)ō/` and three variants). Same word,
different entry, different fact. The claim needs its scope stated, or `#29`'s
design will read it as "there is no notation to compare against" when for
borrowings there is.

**And a constraint this issue's URL work should carry:** the Spanish recording is
keyed on the source ORTHOGRAPHY. `jalapeno_es_es` (unaccented) is a **404**;
`jalapeño_es_es` is a 200, while the English recording keys the unaccented
`jalapeno`. Whatever this issue does to make language and locale parameters, the
SPELLING is a third input — not a property that follows from the other two.

## Done when

- [x] `define madrugar` plays a recording; asserted against the fake, and a
      conformance test measures the live CDN the way the English ordering is.
- [x] `-lang` selects the language and `-locale` still selects the variant;
      `es_es` and `es_us` both reachable, and the help text says what the
      difference *is* (θ vs seseo) rather than naming two country codes.
- [x] An English word with no recording still degrades to a warning, exit 0 — the
      existing behaviour does not regress.
- [x] No pronunciation is rendered for a Spanish entry, and a test says that is
      expected rather than a gap.
- [x] The new conformance assertions route their dependency probe through
      `conformance.SkipOrFail` (#25) — a CDN that cannot be reached SKIPS by
      default and FAILS under `CONFORMANCE_STRICT`.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.15 impl=0.08
item: smaller-go-module        design=0.05 impl=0.10
item: smaller-go-module        design=0.05 impl=0.12
item: smaller-go-module        design=0.00 impl=0.08
item: atlas-docs               design=0.04 impl=0.06
item: milestone-review         design=0.00 impl=0.16
design-buffer: 0.15
total: 0.93
```

Derivation notes.

- **`issue-spec` impl 0.08 covers the gate rounds that follow the claim,** not
  the planning that preceded it. `#26` taught this: `sdlc actual` anchors on the
  claim commit, and pricing pre-claim triage at zero while post-claim gate
  iteration goes unpriced is how that issue's first block was wrong. The plan
  rewrite and two gate rounds are inside the window.
- **Three `smaller-go-module`s, and the first is genuinely small.** The locale
  policy change is deleting one guard — the whitelist I planned was withdrawn at
  the gate — so 0.10 is the table test around it rather than the edit. The second
  is the help-const plus its README doc-sync, which is a mechanism this repo
  already has and I am copying. The third is the Spanish-notation test at 0.08
  design-free, because the plan already scoped it (Spanish dictionary, not
  Spanish word) and the `entries/es/` corpus exists.
- **No `real-api-discovery` line, deliberately, and `#26` is why I checked.**
  That issue needed one because the classifier had to CONVERGE against a live
  population it had never run over. Here the CDN facts are already measured and
  the `es_us` conformance row is one assertion beside an existing one — a live
  run to confirm, not a loop to converge.
- **One `milestone-review` at 0.16.** Below the 0.20 ceiling: the diff is a guard
  deletion, a const, two tests and doc lines.
- Library-availability check: nothing external; no halving applies.

Σdesign 0.29 × 1.15 = 0.3335; Σimpl 0.60; total **0.93**.

## Plan

- [x] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-28
- 2026-08-28: closed — Round-2 finding fixed.; review verdict: FIX-THEN-SHIP

BR-8: #26 built retiredSymbolNames + TestNoArtifactNamesARetiredSymbol so a rename sweeps every prose restatement of the old symbol, and its own comment states why one step must stay manual — a rename cannot be detected automatically, because only the person doing it knows the old name. One issue later I renamed TestREADMEQuotesTheLocaleHelp to TestDocsQuoteTheLocaleHelp, skipped the row, and left a stale mention in voice.go — in the SAME commit that widened the test being renamed. The guard could not fire, because its input is the thing I did not supply.

Row added, stale mention corrected, and mutation-verified: reintroducing the old name in voice.go now fails by name with the replacement named.

Lesson recorded in workshop/lessons.md: the human half of a mechanical guard is the half that fails, and "I built the guard" is not "the guard is armed" — a mechanism with an unsupplied input is unprotected, not protected.

EARLIER ROUNDS (disposed): -locale honoured for every language with the whitelist withdrawn at the plan gate in favour of ParseLang open-domain stance; the dead complaint plumbing removed; the Spanish-notation claim scoped with both halves tested and jalapeño captured so the English half asserts rather than skips; conformance covering both es locales plus cazar/casar; the doc-sync reaching BOTH docs; and TestLocaleFlagReachesTheCDN pinning the flag-to-CDN wiring, proved to earn its place by a mutation that leaves the pure function green and the wiring test red.

TESTS: go build ./... && go vet ./... && GOOS=linux go vet ./cmd/define/ && go test ./... all green; gofmt -l ./cmd/ empty.

- **Split out of #18** so the locale concept can land ahead of the two issues that
  want it, neither of which needs #18's deck/agreement scope:
  - **#26** (NOAD ratchet 27→32) — its open design question is whether a dual
    locale block `| AmE brɛnt, BrE brɛnt |` should render both pronunciations or
    the one matching the deck's locale. That question is unanswerable while
    "locale" is a literal in a URL builder.
  - **#18 M2** — the deck language dimension builds directly on language being a
    parameter.
- Everything in the Spec was measured against the live CDN and dictionary on
  2026-08-22, not reasoned about; see #18's Log for the measurement session,
  including a wrong claim it corrected (a sandboxed `DCSCopyTextDefinition`
  reports a clean miss, indistinguishable from an absent word).

- 2026-08-28: **BLOCKED on #23, and the plan's central design was wrong.**
  Operator, before implementation started:
  > *we will split `define` into language specific thing, so every `define`
  > invocation will operate in 1 language only. it can switch in a TUI app, but
  > at any given time one language only. make this change first.*

  That is `#23`, already filed the day before with the same design in the
  operator's own words. Under a language MODE there is no unspecified language,
  which deletes this plan's central mechanism — a `voices()` policy that tried
  `en` then `es` and paid ~900ms of misses for a Spanish word. Designing a
  fallback for a question the mode answers is solving a problem that is about to
  stop existing.
  **Sequencing (operator's call):** `#23` end-to-end first, with the audio
  language plumbing as one task inside it. What plausibly remains here afterwards
  is the *variant* half — `es_es` vs `es_us`, the θ/seseo help text, and the live
  CDN conformance — which is reassessed at `#23`'s close rather than assumed now.
  **The measurements survive and are the reusable part** — see
  `workshop/plans/000027-pronunciation-locale-plan.md`, whose Revisions entry
  records what changed and why. They were re-run on 2026-08-28 and three of them
  are load-bearing for `#23`'s audio task regardless of sequencing: languages are
  disjoint on the CDN, the legacy `/sounds/oxford/` path is English-only, and a
  404 costs ~10x a hit.

### 2026-08-28 — implemented; the plan gate made the change smaller

`#23 M1` had already shipped five of the original plan's seven tasks, so this
issue reduced to the locale policy and its consequences.

**The gate reversed my central design, and it was right.** I planned a closed
table of valid language/locale pairs, refusing anything not in it. PQ-1 caught
that contradicting this plan's own Risks section, and the Risks argument was
better: a table restates a fact the CDN owns and goes stale when Google adds a
variant — which is exactly why `ParseLang` does not enumerate languages. A table
would also have had no answer for `fr`, which `ParseLang` admits and the CDN
serves.

So the change is a DELETION: `localeFor` loses the `l == store.DefaultLang`
guard, and `-locale` is honoured for every language. An unserved pair 404s into
the warning every missing recording already produces.

**A dead guard went with it.** With no refusal, `localeFor` could only ever
return an empty complaint, so the complaint return value and `applyVoice`'s
`io.Writer` were removed rather than kept warm for a hypothetical caller.

**Two scope corrections that came out of the work:**

The "Spanish entries carry no phonetic notation" claim in this Spec is true of a
Spanish word in a SPANISH dictionary and false of one in an English dictionary —
NOAD gives `jalapeño` four anglicised pronunciations. Both halves are now tests,
and `jalapeño` was captured into the English corpus so the second is a real
assertion rather than a skip.

The conformance row covered `es_es` only — the locale this issue makes selectable
was the one nothing checked. It covers both now, plus `cazar`/`casar`, so a
coverage drift shows up on the words where the phonemic split is actually
audible.

**Verified against the real binary:** `-lang es -locale us` and `-locale es` both
play; `-locale gb` with Spanish degrades to "no recorded pronunciation" rather
than refusing; the help text states the θ/seseo distinction.

### 2026-08-28 — close review: two findings, both repeats of lessons written this session

**BR-1: I wired the README to `localeHelp` and left the atlas restating the
policy by hand.** The finding this issue was fixing is "the policy is stated in
four places with nothing keeping them in step" — and the fix reached one of the
two docs, leaving the other as the next copy to go stale. Same half-fix shape as
`#26`'s BR-4, where I marked a total and missed a second statement of it. The
doc-sync loops both docs now.

**BR-2: no `run()`-level test drove `-locale`, so the flag→CDN wiring was
unpinned.** `workshop/lessons.md` carries this rule from earlier today — *"test
what the pure function's CALLER does"* — written after `#23`'s C1, where
`voiceFor` was correct the whole time and nothing re-derived `opt.voice`. I
shipped the same gap again.

Proved the new test earns its place with a mutation that breaks the WIRING while
leaving the pure function correct — `localeSet: false` in the flag parse:

```
-- unit test on the pure function --   ok      (green, unaffected)
-- the wiring test --                  FAIL    the session never asked for _en_gb_
```

That asymmetry is the whole argument for the test: a unit test on `localeFor`
cannot see a caller that stops consulting it.

The four rows also carry a vacuity guard. A first version asserted on the Spanish
locales while passing the ENGLISH fake corpus, so the lookup failed, no audio was
fetched, and the rows asserted nothing — they reported "never asked for _es_us_"
which reads like a wiring bug and was a fixture bug. The guard now fails loudly
when a row requests nothing at all.

### 2026-08-28 — close sanctioned; three Minors fixed before committing

**BR-5 was a real input-handling bug, not a nit.** The locale reaches the CDN
path straight from `-locale`, unescaped, while the word beside it goes through
`url.PathEscape`. A locale containing `/` would rewrite the path rather than 404
— and a clean miss is this issue's contract, since nothing whitelists which
locales exist. Escaped now, and pinned.

**BR-4: `localeSet` went inert in my own change and I kept it.** Changing the
flag's default from `"us"` to `""` made the value carry "not given" by itself;
the separate bool was only ever needed because `"us"` is a real locale, so
`fs.Visit` was the only way to tell "asked for American" from "said nothing".
Removed rather than kept warm — the same call as the complaint plumbing earlier
in this issue.

**BR-3: three comments described rules this issue deleted** — `voice.go` still
said `#27` *owns* the locale policy as future work, and `main.go` explained a
`-locale` complaint that no longer exists. Corrected, keeping the historical note
where it explains why the English exception survives (it is a fact about the
CDN's key format, not a policy).

