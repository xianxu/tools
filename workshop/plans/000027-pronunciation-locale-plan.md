# Pronunciation Locale Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the recording's language a parameter instead of a literal, so a Spanish word is asked for as Spanish.

**Architecture:** A pure `voice{Lang, Locale}` value replaces the bare `locale string`, and a pure `voices()` function owns the ordering policy — which languages to try, in what order. `AudioCandidates` builds URLs for that ordered list. Nothing about the fetch loop changes: it already walks candidates until the first 200, so widening the list costs no new mechanism.

**Tech Stack:** Go, no new dependencies. Existing `fakeCDN` stateful fake for integration tests; existing `-tags conformance` suite for the live CDN.

---

## Measurements this plan rests on

Every number below was re-measured on **2026-08-28** against the live CDN, not carried from the issue (which measured on 2026-08-22). Three of these are NEW and change the design; the issue did not have them.

| claim | measured | consequence |
|---|---|---|
| `madrugar_en_us_1` | **404** | today's builder can never find a Spanish recording |
| `madrugar_es_es_1` / `madrugar_es_us_1` | **200 / 200** | both variants exist |
| `es_419`, `es_mx`, `es_ar` | **404, 404, 404** | only `es`/`us` are real Spanish locales |
| `sobremesa`, `empalagoso`, `chapucero`, `desvelarse` on `es_es` | **200 ×4** | coverage is real, not incidental |
| **`sycophantic_es_es_1`** | **404** | **languages are DISJOINT — trying a second language can never return the wrong word's audio** |
| **`madrugar--_us_1`, `madrugar--_es_1`** | **404, 404** | **the legacy `/sounds/oxford/` path is ENGLISH-ONLY — Spanish candidates on it are guaranteed waste** |
| **a 404 vs a 200** | **~300–600ms vs ~40ms** | **a wasted candidate costs ~10× a hit; ordering is a performance decision, not only a correctness one** |
| `defenestrate` modern_1 | **200** (legacy_1 also 200) | the modern path now serves every sampled word; legacy is pure fallback and belongs last |

The three bolded rows are why this plan is not simply "append `es` candidates to the list".

Reproduce with (unsandboxed — the sandbox cannot reach the CDN):

```sh
B=https://ssl.gstatic.com/dictionary/static/pronunciation/2022-03-02/audio
curl -s -o /dev/null -w '%{http_code} %{time_total}\n' "$B/ma/madrugar_es_es_1.mp3"
curl -s -o /dev/null -w '%{http_code} %{time_total}\n' "$B/sy/sycophantic_es_es_1.mp3"
```

## Scope check

One subsystem: the audio URL builder and the flag that feeds it. `#18 M2` (deck language dimension, lemma identity, gender, agreement-safe distractors) is explicitly NOT here and remains blocked on `#10`/`#12`. `#26` (NOAD dual-locale rendering) depends on this landing but is not part of it.

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `voice` | `cmd/define/voice.go` | new |
| ~~voices~~ (never built) | `cmd/define/voice.go` | **superseded by `#23`** — see Revisions |
| `AudioCandidates` | `cmd/define/audiourl.go` | modified |

- **`voice`** — the language plus regional variant a recording is asked for: `voice{Lang: "es", Locale: "es"}`.
  - **Relationships:** 1:N with the candidate URLs it produces. Held by `options`, passed to `speak` and `AudioCandidates`.
  - **DRY rationale:** First occurrence, and it exists to kill a *mix-up hazard rather than a duplication*. `"es"` is a valid value of BOTH fields — `AudioCandidates(w, "es", "es")` (Castilian) and a transposed `AudioCandidates(w, "es", "en")` are indistinguishable at the call site, and there are 20+ call sites in tests. A two-field struct makes the transposition unspellable.
  - **Future extensions:** `#18 M2` gives the deck a language, so a word will carry its own `voice`; this is the type that lands on it. A third field (dialect, speaker sex) widens here without touching callers.

- **~~voices~~ — DELETED, not adapted.** `#23` made the language a declared MODE, so there is nothing to order: the mode supplies exactly one language and `AudioCandidates` builds for it alone. What shipped instead is `localeFor`, the interim locale rule `#27` inherits.
  - **Relationships:** pure `voice → []voice`. Called only by `AudioCandidates`.
  - **DRY rationale:** Splitting the policy from the URL construction is what makes the policy testable as a table without asserting on URL strings. `AudioCandidates` then has one job (spell a URL) and `voices` has one job (decide what to ask for).
  - **Why it is its own function, not an `if` inside `AudioCandidates`:** the ordering is the part with measured facts behind it and the part most likely to change (a third language, a cheaper probe, a per-deck default). Burying it in string concatenation is how the `_en_` literal happened in the first place.
  - **Future extensions:** when the deck knows a word's language, an unspecified `voice` stops meaning "guess" and starts meaning "ask the deck" — that is a change to this function alone.

- **`AudioCandidates`** — unchanged in purpose: the ordered CDN URLs to try. Now takes a `voice`, and emits legacy `/sounds/oxford/` candidates **only for English**, because they are measured English-only.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `-lang` flag | `cmd/define/main.go` | new | operator input |
| `fakeCDN` | `cmd/define/fetch_fake_test.go` | reused | Google's pronunciation CDN |
| `TestCDNStillServesSpanishOnTheExpectedPaths` | `cmd/define/fetch_conformance_test.go` | **landed in `#23 M1`** | the live CDN |

- **`-lang` flag** — selects the language; empty means unspecified.
  - **Injected into:** `options.voice`, then `speak`, then `AudioCandidates`. The pure builder never reads a flag.
  - **Future extensions:** a per-deck default from `#18 M2` supersedes the flag's default without changing its meaning.

- **`fakeCDN`** — the existing stateful fake. **Reused, not extended**: it already records every requested path in order, which is exactly what proves a Spanish word's walk skips the English-only legacy path. No new fake is needed and adding one would be the near-fit double `#6 BR-43` warns about.
  - **State model:** a set of present paths plus an ordered request log.

- **`TestCDNStillServesSpanishOnTheExpectedPaths`** — live conformance beside `TestCDNStillServesTheExpectedPaths`, pinning the facts the gate rests on. Landed in `#23 M1` rather than here, since `#23` needed the same measurement.
  - **Cadence:** on-demand with the rest of `-tags conformance`; routes its dependency probe through `conformance.SkipOrFail` (`#25`) so an unreachable CDN SKIPS by default and FAILS under `CONFORMANCE_STRICT`.

---

## What `#23` already delivered — measured, not assumed

Re-measured 2026-08-28 against the live tree and CDN. **Five of this plan's seven
original tasks are done**, because `#23 M1` needed the same plumbing and built it.
The original task bodies are superseded and were removed rather than left to
mislead; `git log` has them.

| this plan's original task | state |
|---|---|
| T1 `voice` + the `voices()` ordering policy | `voice` **shipped**; `voices()` **deleted, not adapted** — a declared mode has nothing to order |
| T2 `AudioCandidates` takes a `voice` | **shipped** |
| T3 the `-lang` flag | **shipped**; its help text is not this issue's |
| T4 the walk skips what cannot exist | **shipped** — the legacy `/sounds/oxford/` pair is gated to English |
| T5 live conformance for Spanish | **shipped** as `TestCDNStillServesSpanishOnTheExpectedPaths` — but `es_es` only |
| T6 a Spanish entry has no pronunciation, and that is correct | **not started** |
| T7 docs | partial — the deck/mode half landed, the locale half did not |

Verified by running the binary: `define -lang es -sound 1 madrugar` fetches and
plays. Done-when row 1 is satisfied except for its `es_us` half.

## What is actually left

**The core is one thing: `-locale` does not work for Spanish.** `#23 M1`'s D2
shipped an explicit INTERIM rule — the locale is the language code, except
English which is `us`, and `-locale` is honoured for English ONLY. Measured:

```
$ define -lang es -locale us madrugar
define: -locale us is an English variant; es recordings use es_es.
        Locale variants for other languages are not supported yet
```

That message was written to be replaced by this issue, and D2 said so. Replacing
it is the work.

### Measurements this replacement rests on

Re-run 2026-08-28, independently of `#18`'s 2026-08-22 session:

| claim | measured |
|---|---|
| `madrugar_es_es`, `madrugar_es_us` | **200, 200** |
| the phonemic pair: `cazar`, `casar` on both locales | **200 × 4** |
| `es_419`, `es_mx` | **404, 404** — the two-locale set is the whole set |
| `en_gb` on the 2022 path (`schedule`, `tomato`, `privacy`, `laboratory`, `water`) | **200 × 5** |
| `en_gb` on the legacy path (`schedule`, `tomato`, `privacy`, `sycophantic`) | **200 × 4** |

**One correction worth recording, because it nearly became a finding.** A first
probe read `colour_en_gb` as 404 and I almost reported that `gb` was unserved.
`colour_en_us` is *also* 404 — the word is simply absent from the corpus. One
negative probe is not a measurement; the five-word sweep is.

## Tasks

Plain checkboxes, one `sdlc close`: a locale policy, its help text, one test and
a conformance row is a single review boundary.

- [ ] **Replace `localeFor`'s interim rule: `-locale` is honoured for ANY
      language.** The whole change is deleting the `l == store.DefaultLang`
      guard that makes it English-only, so `-lang es -locale us` builds
      `madrugar_es_us` instead of warning and falling back.

      **NO whitelist of valid pairs**, and this is a decision reversed during
      planning rather than a default. A first draft proposed a closed table
      (`en` → {`us`,`gb`}, `es` → {`es`,`us`}) refusing anything else. That
      contradicts this codebase's own precedent: `ParseLang` deliberately does
      not whitelist languages, because *"the CDN and the installed dictionaries
      decide what exists, and a hardcoded list here would be a restatement of a
      fact they own"*. The same argument applies exactly. A table would also have
      no answer for `fr`, which `ParseLang` admits and which the CDN serves —
      `arrondissement_fr_fr` is a 200.

      An unserved pair therefore 404s and degrades to the existing warning at
      exit 0, which is what every missing recording already does. The default arm
      is unchanged: locale = language code, `en` → `us`.
- [ ] **The help text says what the difference IS, from ONE source.**
      `"pronunciation locale: us or gb"` names two country codes, explains
      nothing, and is now wrong twice over — it omits Spanish and implies the set
      is closed, which the bullet above says it is not.

      Done-when asks for the θ/seseo distinction because it is a phonemic choice
      about which sound system the learner acquires: `cazar` /θ/ ≠ `casar` /s/ in
      `es_es`, both /s/ in `es_us`. It must give examples without claiming to
      enumerate.

      **One source, because the policy is currently stated in four places** —
      `localeFor`, the flag help, the README and the atlas — with nothing keeping
      them in step. The help string becomes a const, and the README derives from
      it through `doc_sync_test.go`, which already implements exactly this for
      the play-loop prompts. Same family, same fix; it has recurred enough in
      this repo to be mechanical rather than swept.
- [ ] **A Spanish entry has no pronunciation, and a TEST says that is expected.**
      Spanish orthography is phonemic, so Larousse carries no notation at all —
      `ParseEntry` finding none is CORRECT and must not be "fixed" later by
      someone reading it as a gap. Pin it against the committed `entries/es/`
      corpus, which `#23 M2` captured, so it needs no live dictionary.
      **Scope it precisely:** the claim is about a Spanish word in a SPANISH
      dictionary. A Spanish word in an English one is a different case with real
      notation — NOAD gives `jalapeño` four anglicised pronunciations — and
      conflating them is how the "no notation" claim becomes wrong.
- [ ] **Extend the Spanish conformance row to `es_us`.**
      `TestCDNStillServesSpanishOnTheExpectedPaths` covers `es_es` only, so the
      locale this issue makes selectable is the one nothing checks.
- [ ] **Docs:** the README's `-locale` line and `atlas/define.md`'s pronunciation
      section carry `#23`'s interim rule verbatim. Both must state the real one,
      and the atlas entry that says `#27` owns this becomes past tense.

## Risks

**No whitelist means a wrong pair fails late and quietly.** `-lang es -locale gb`
builds `madrugar_es_gb`, which 404s, and the learner gets the standard
"no recording" warning rather than "that locale does not exist for Spanish". That
is the deliberate trade — see the locale task — and it follows `ParseLang`'s
precedent rather than inventing a second philosophy for the adjacent field. If it
proves confusing in use, the fix is a warning that NAMES the pair, not a
rejection: the CDN stays the authority on what exists.

**The two-locale Spanish set is measured, not guaranteed.** `es_419` and `es_mx`
are 404 today. Google could add them, and nothing here would need to change —
which is the point of not encoding the set. The conformance row records what was
true when measured.

**A Spanish entry legitimately has no pronunciation, and the next reader may
"fix" it.** That is why a test asserts it rather than a comment. The risk is the
test being written too broadly: *a Spanish word in an English dictionary DOES
carry notation* — NOAD gives `jalapeño` four anglicised pronunciations — so a
test asserting "Spanish words have no notation" would be false. It must assert
about the Spanish DICTIONARY's entries.

**The legacy path could grow a language segment.**
`TestCDNStillServesSpanishOnTheExpectedPaths` asserts the negative — that
`/sounds/oxford/` does not serve Spanish — so it is what would catch the change.

## Done-when → task map

| Done-when row | where |
|---|---|
| `define madrugar` plays a recording, asserted against the fake + live conformance | **done in `#23 M1`**; the live half extends to `es_us` here |
| `-lang` selects the language, `-locale` the variant; both `es` locales reachable | the locale-policy task |
| the help text says what the difference IS (θ vs seseo) | the help-text task |
| an English word with no recording still warns and exits 0 | **done** — unchanged by `#23`, re-asserted by the existing suite |
| no pronunciation for a Spanish entry, with a test saying that is expected | the Spanish-notation task |
| conformance probes route through `conformance.SkipOrFail` (`#25`) | **done** — the `#23` row already does; the `es_us` extension inherits it |

## Notes for the reviewer

- **This plan was 821 lines and is now 300.** Five of its seven tasks shipped
  inside `#23 M1`, which needed the same plumbing; their step-by-step bodies were
  removed rather than left to describe work already done. `git log` has them, and
  the Revisions entry below records what went and why.
- **Every measurement was re-run on 2026-08-28**, not carried from `#18`'s
  2026-08-22 session, and one of them corrected a near-miss of my own: `colour_en_gb`
  reads 404, but so does `colour_en_us` — the word is absent from the corpus, and
  reporting "gb is unserved" off that single probe would have been wrong.
- **The interim rule this issue replaces was written to be replaced.** `#23 M1`'s
  D2 states it explicitly and names this issue as its successor, so the
  replacement is a hand-off rather than a reversal.
- **`#29` consumes whatever this decides.** `jalapeño_es_es` and `jalapeño_es_us`
  are both 200, so a borrowed word's source language does not determine its
  recording — the locale policy does. Keeping it one policy with one home is
  what makes that possible.

## Revisions

### 2026-08-28 — the multi-language fallback is deleted; `#23`'s language mode replaces it

**Reason.** Operator, before implementation started:

> *we will split `define` into language specific thing, so every `define`
> invocation will operate in 1 language only. it can switch in a TUI app, but at
> any given time one language only. make this change first.*

That design is `#23`, filed the day before with the same words, and this plan
was written without reconciling against it. **The plan's central mechanism was
designed for a question the mode answers.**

**Delta — what is now wrong above:**

- **`voices()` is deleted, not adjusted.** Its whole job was ordering an
  unspecified language across `en` then `es`. Under a mode there is no
  unspecified language: the invocation has exactly one, either persisted in the
  vocab directory or given by `-lang`. Task 1's table, its three mutations, and
  the `languageOrder` var all go with it.
- **The ~900ms fallback cost in Risks stops existing.** It was the price of
  guessing; a mode does not guess. The Risks entry predicting it "stops being
  acceptable when `#18 M2` gives the deck a language" was directionally right and
  arrived at the wrong remedy — the answer was not a better guess, it was not
  guessing.
- **`AudioCandidates` builds candidates for ONE language.** Same signature
  (`word string, v voice`), simpler body: one modern pair for `v.Lang`, plus the
  legacy pair only when `v.Lang == "en"`.
- **The disjointness measurement changes role, not truth.** It justified the
  fallback being *safe*; with no fallback it instead justifies a mode being
  *sufficient* — asking the wrong language returns nothing rather than the wrong
  word's audio, so a mis-set mode is a visible miss, not a silent wrong answer.

**What survives unchanged, and is the reusable part.** Every measurement in
*"Measurements this plan rests on"*, re-run 2026-08-28. Three are load-bearing
for `#23`'s audio task whatever the sequencing:

1. languages are **disjoint** on the CDN,
2. the legacy `/sounds/oxford/` path is **English-only** (so Spanish candidates
   on it are guaranteed waste),
3. a 404 costs **~10×** a hit (~300–600ms vs ~40ms).

Also surviving: the `voice{Lang, Locale}` type and its mix-up argument, the
θ/seseo semantics of the Spanish locale pair, the `fakeCDN`-reuse decision, and
the "a Spanish entry has no IPA, and that is correct" test.

**Sequencing (operator's call).** `#23` end-to-end first, with the audio language
plumbing as one task inside it. `#27` is `blocked` on `#23`; what plausibly
remains here afterwards is the *variant* half — `es_es` vs `es_us`, the help text
that says what the choice is, and the live CDN conformance — reassessed at
`#23`'s close rather than assumed now.

**The process lesson, since this is the second time today.** This plan reached
727 lines before anything reconciled it against the open issue that already
specified the design. `#23` was on the board, was filed from the operator's own
words, and its Spec says *"everything inherits the mode … the audio asks for that
language's recording"*. The `sdlc state` board was read at the start of this
session and the overlap still went unnoticed, because I read `#27` and the code
and never re-read the neighbours it names. **Before planning an issue, read the
issues it touches** — `#27`'s own Log named `#18` and `#26`, and `#23` was one
`grep -l lang workshop/issues/` away.

### 2026-08-28 — two rows this plan named were superseded by `#23`

**Reason.** `#23`'s close review found the symbol half of the artifact-name rule
unenforced for the fourth time, and the fix — a guard asserting every
Core-concepts row names an entity the tree actually has — flagged two rows here.

**Delta.** `voices` was DELETED rather than adapted: `#23` made the language a
declared mode, so the ordering policy this plan designed has nothing left to
order. `TestCDNServesSpanish` shipped as
`TestCDNStillServesSpanishOnTheExpectedPaths` in `#23 M1`, because `#23` needed
the same measurement. Both rows now say so instead of naming absent entities.

### 2026-08-28 — five of seven tasks shipped inside `#23 M1`; the plan is rewritten around what is left

**Reason.** `#23` needed the same plumbing this plan designed — `voice`,
`AudioCandidates` taking one, the language flag, the English-only legacy gate —
and shipped it. Re-measured against the tree rather than assumed: only the locale
policy, its help text, the Spanish-notation test and an `es_us` conformance row
remain.

**Delta.**

- **Chunk 1 and Chunk 2 (about 400 lines of task steps) were REMOVED**, not
  ticked. Ticking them would claim this issue did the work; leaving them would
  describe work as pending that is already in `main`. A table now records each
  original task's fate, and `git log` holds the bodies.
- **`voices()` stays deleted.** Recorded in the earlier revision and restated
  here because it is the one design element that was not merely moved: a declared
  mode supplies exactly one language, so there is no ordering policy to write.
- **The remaining work is one boundary**, so the plan uses plain checkboxes
  rather than milestones — `AGENTS.md` §3: an `Mx` tag is a review boundary, and
  tagging a single-pass task forces a redundant double-close.
- **The Spanish-notation task gained a scope caveat** it did not have: the
  "Spanish entries carry no phonetic notation" claim is true of a Spanish word in
  a SPANISH dictionary and false of one in an English dictionary. `#27`'s own
  Spec stated it unscoped, and `#29` would have inherited it wrongly.
