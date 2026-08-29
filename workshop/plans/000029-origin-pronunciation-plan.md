# Origin Pronunciation Implementation Plan (`#29`)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hear a borrowed word in its source language — `define -pron fr arrondissement`, or `/pron fr` at the prompt — while the session's deck, dictionary and highlight set stay exactly where they were.

**Architecture:** `#27` made the *locale* half of a `voice` an independently overridable parameter. This does the same for the *language* half, but **per-line rather than per-session**: the pronunciation language rides on `replCommand` (what one line of input means) and never enters `options`, so no `/lang` switch can inherit it. One new pure type, `utterance`, owns the whole candidate walk — source spellings in the source voice, then the session's own recording as the fallback — and reports which one answered by MEMBERSHIP in the list it built, never by parsing a URL back.

**Tech Stack:** Go 1.26, stdlib only — no new module dependency (see Task 1). Tests: `go test ./cmd/define`, the existing `fakeCDN` `httptest` seam, and `-tags conformance` for live CDN rows.

**On the level of detail here:** this plan states each function's *contract* and the *class* its tests must cover, not its body. Pre-written implementations and doc comments get rewritten within the hour and a hand-picked case table is a lossy pre-image of the executable artifact; the measured tables below are kept because they are the evidence, which is the part that does not regenerate.

---

## Decisions

Argued from measurement taken 2026-08-28/29. Full evidence in `workshop/issues/000029-*.md`.

**D1 — the language is DECLARED, never inferred.** NOAD writes `arrondissement`'s etymology as `ORIGIN French, from arrondir` and `police`'s as `ORIGIN … from French, from medieval Latin` — structurally the same. The CDN does not discriminate either: `police_fr_fr`, `restaurant_fr_fr`, `garage_fr_fr`, `machine_fr_fr`, `unique_fr_fr`, `genre_fr_fr`, `nuance_fr_fr`, `montage_fr_fr`, `bureau_fr_fr` are **all 200**. Automatic origin audio would silently replace the English recording for a large class of naturalised everyday words. This re-derives `#23`'s rejection of inference from new evidence at a narrower scope.

**D2 — `/pron fr` is an ACTION, not a mode.** `/sound 5` and `/lang es` name settings; "say it in French" is a thing you do once. It replays the current word and leaves nothing switched on. Operator's choice, and it is the same call `#30`'s click on `ORIGIN French` will make.

**D3 — the pronunciation language never enters `options`.** `opt.voice` is a SESSION value that `applyLang` re-derives on every `/lang`; an override living there would survive a switch and ask for `fr_fr` recordings in a Spanish session — the drift `applyLang`'s enumeration comment exists to prevent. It rides on `replCommand` instead, beside `literal`, which is already a per-line modifier. `voiceFor`, `localeFor`, `defaultLocale` and `applyVoice` are **not modified at all** (ARCH-DRY: the source voice is `voiceFor(pronLang, opt.locale)`, the function that already exists).

**D4 — `-locale` still qualifies whatever language is in effect.** No second locale policy, which is `#29`'s sixth Done-when. `-pron es` → `es_es`; `-pron es -locale us` → `es_us`, seseo. An unserved pair (`-pron fr -locale gb`) 404s and degrades, exactly as `atlas/define.md` already promises for `-lang es -locale gb`.

**D5 — this REINTRODUCES a cross-language fallback, which `#23` removed.** `#23`'s would have been a GUESS about which language a word belongs to, run on every lookup and paid for by everyone — and D1 measures that such a guess is often wrong. This one runs only when `-pron`/`/pron` was typed, and what answered is REPORTED. Task 8 rewrites every statement of the old invariant; the enumeration is there, not here.

**D6 — NON-GOAL: labelling which notation variant is which.** The Spec called this "worth pricing": `/əˈrändəsmənt, eˌrändēsˈmäN/` does not tell the reader the first is English and the second French. It is not built here and it is not deferred as "follow-up" — it is a different product. The operator's ask was the authentic SOUND, and D2 chose an action that produces it; labelling notation answers a reading question instead, needs a per-variant language attribution NOAD does not supply, and would be wrong to guess. Not filed as an issue; recorded here so `#29` closes having disposed of it rather than left it unanswered.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `differsOnlyByDiacritics` | `cmd/define/parse.go` | new |
| `Entry.AlsoSpellings` | `cmd/define/parse.go` | new |
| `SourceSpellings` | `cmd/define/audiourl.go` | new |
| `utterance` | `cmd/define/audiourl.go` | new |
| `pronHelp` | `cmd/define/voice.go` | new |
| `parsePronArgs` | `cmd/define/pron_cmd.go` | new |
| `replCommand` | `cmd/define/repl.go` | modified — gains `pron store.Lang` |
| `AudioCandidates` | `cmd/define/audiourl.go` | unchanged — reused once per spelling |
| `voiceFor` / `localeFor` / `defaultLocale` / `applyVoice` | `cmd/define/voice.go` | unchanged — D3 |

- **`differsOnlyByDiacritics(alt, head string) bool`** — the same word wearing different accents.
  - **Relationships:** 1:1 with `AlsoSpellings` today.
  - **DRY rationale:** first occurrence; exists so the `(also …)` filter is one testable predicate rather than a condition inlined in a loop.
  - **Future extensions:** `ß`/`ss` and `œ`/`oe` expansions change the length and are refused; widening means a length-tolerant comparison, and it goes here.

- **`Entry.AlsoSpellings() []string`** — every `(also …)` alternative that is the headword up to diacritics.
  - **Relationships:** N:1 with `Entry`, derives from `Blocks`, adds no state.
  - **DRY rationale:** joins the `Headword()`/`Homograph()`/`Syllables()` family, which all derive from one representation so the fields and the render order cannot drift.
  - **Future extensions:** `#30` wants clickable head tokens — the same "what does the head offer" surface.

- **`SourceSpellings(typed string, e Entry) []string`** — spellings the source language might key the recording under, best first.
  - **Relationships:** 1:1 with a lookup; consumed only by `utteranceFor`.
  - **DRY rationale:** the ONE answer to "what might the source call this", so the one-shot path and both loops cannot each grow their own.
  - **Future extensions:** ORIGIN text literally embeds a spelling (`ORIGIN … from French café`); if that becomes a fourth source it is a row here.

- **`utterance`** — one request: the word, its source spellings, and two voices tried in order.
  - **Relationships:** 1:1 with a play; holds two `voice` values.
  - **DRY rationale:** a struct rather than positional arguments because `Source` and `Session` are BOTH `voice` and transposable at every call site — the argument that made `voice` itself a struct, one level up.
  - **Future extensions:** a third voice (a user recording) is a field, not a new walk.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `speak` | `cmd/define/main.go` | modified | `AudioSource` + `Player` |
| `playAnnounced` | `cmd/define/main.go` | modified | the terminal |
| `reportVoice` | `cmd/define/main.go` | new | an `io.Writer` |
| `utteranceFor` | `cmd/define/main.go` | new | — (glue) |
| `replayInPlace` | `cmd/define/replraw.go` | modified | raw terminal + player |
| `runPron` | `cmd/define/pron_cmd.go` | new | `commandCtx` |
| `commandCtx` | `cmd/define/command.go` | modified | the player, via a `replay` closure |
| `fakeCDN` | `cmd/define/fetch_fake_test.go` | unchanged — REUSED | Google's CDN |

- **`speak`** — takes an `utterance`; returns the URL that answered, which it currently fetches and discards. The reasoning it used to imply moves into `utterance.Candidates()`.
- **`reportVoice`** — says what actually played, only when a source was asked for and the session answered. Written after the fetch, from the URL that answered.
- **`commandCtx`** gains `replay func(store.Lang)` — the only capability a command gains over playback, and it does **not** play: it RECORDS a request the loop performs. In the raw editor, commands dispatch inside `cooked(func(){…})`; playing there hands Ctrl-C to the line discipline, which swallows the byte — `workshop/lessons.md`, *"Raw mode: render cooked, play raw"*. A closure like the existing `setLang`/`setTimes`, so "a command has no business reaching the dictionary or the player" holds in substance.

**ARCH-MOCK.** The external dependency is Google's CDN and it already has a stateful fake behind the `AudioSource` seam: `fakeCDN` records **every path requested, in order** — the only way to assert a walk order. No new fake; the deliverable REUSES this one. `fetch_conformance_test.go` is the live conformance check and Task 9 adds the rows these decisions rest on.

---

## Chunk 1: the pure core

### Task 1: `differsOnlyByDiacritics`

**Files:** modify `cmd/define/parse.go`; test `cmd/define/parse_test.go`.

**Contract.** `differsOnlyByDiacritics(alt, head string) bool` — true iff, lowercased, the two are equal length, differ somewhere, and every difference sits at a position where at least one rune left ASCII.

**Why no `golang.org/x/text`.** The natural implementation is NFD-normalise and strip combining marks, but `x/text` is not in `go.mod` (only `x/term`, `x/sys`, `x/sync`) and pulling a large module in for one predicate is not worth it. The stated rule is exactly as discriminating on every measured case, needs no table of accented characters, and says what it means. Two consequences are accepted rather than overlooked: `ß`/`ss` and `œ`/`oe` change the length and are refused, so such an entry degrades to English and says so — the same outcome as any other miss.

**Why the rule is this and not "contains an accent".** Measured over 400 live NOAD entries (2026-08-29), `(also …)` is not a spelling list. The predicate must **admit**:

| head | alt | why |
|---|---|---|
| `cafe` | `café` | accent added |
| `naive` | `naïve` | diaeresis added |
| `facade` | `façade` | cedilla added |
| `cliché` | `cliche` | accent REMOVED — the headword is the accented one |
| `senor` | `señor`, `Señor` | tilde; case is not a difference that matters |

and **refuse** `adviser`/`advisor`, `converter`/`convertor` (different letters), `cauldron`/`caldron` (length), `naive`/`naïveness` (derivative), `jalapeño`/`jalapeño pepper` (compound), every phrase, and an identical string.

- [ ] **Step 1:** write the table test over both lists above, plus the empty-string and identical-string cells.
- [ ] **Step 2:** run `go test ./cmd/define -run TestDiacriticsOnly -v` — FAIL, undefined.
- [ ] **Step 3:** implement.
- [ ] **Step 4:** add the **malformed-input class**, which the table does not reach. This predicate runs over arbitrary NOAD gloss text, so it must not panic or misbehave on invalid UTF-8, lone surrogates, combining marks arriving decomposed (`e` + U+0301, which is length-2 against a length-1 `é` and must simply return false, not crash), zero-width joiners, or a multi-KB gloss. A short `testing/quick` or fuzz target over random byte strings asserting only "returns, and is symmetric in neither argument's absence" is enough; the point is the crash class, not more equality cases.
- [ ] **Step 5:** `go test ./cmd/define -run 'TestDiacriticsOnly|FuzzDiacritics' -v` — PASS.
- [ ] **Step 6:** commit — `#29: the same word in another dress — the diacritic test`

---

### Task 2: `Entry.AlsoSpellings`

**Files:** modify `cmd/define/parse.go` (beside the `Headword()` accessors); test `cmd/define/parse_test.go`.

**Contract.** Returns every `(also X)` alternative in the entry whose `X` satisfies `differsOnlyByDiacritics(X, Headword())`, in source order.

**The shape it reads, verified against the live dictionary 2026-08-29.** `(also …)` lands as an UNNUMBERED sense in the POS-less first block:

```
cafe → Blocks[0]{POS:"", Senses:[{Number:"", Gloss:"(also café)"}]}
```

**EVERY occurrence in a gloss, not the first.** Also verified by running `ParseEntry` — one gloss can carry two parentheticals:

```
"naive … (also naïve) (also naïveness) adjective …"
  → Blocks[0].Senses[0].Gloss == "(also naïve) (also naïveness)"
```

A `CutPrefix` + `Cut(rest, ")")` reads only the first and passes only because the diacritic one happens to be written first. **The reversed order — measured to parse as `"(also naïveness) (also naïve)"` — would read `naïveness`, fail the diacritic test, and return nothing, silently dropping the source spelling.** Scan the whole gloss.

- [ ] **Step 1:** write the table test: `cafe`→`[café]`; `jalapeño`+`(also jalapeño pepper)`→`nil`; `adviser`+`(also advisor)`→`nil`; no parenthetical→`nil`; **and both orderings of the two-parenthetical `naive` gloss, each →`[naïve]`** — the second is the one that fails a first-match implementation.
- [ ] **Step 2:** run — FAIL, undefined.
- [ ] **Step 3:** implement, scanning every `(also …)` in each gloss.
- [ ] **Step 4:** `go test ./cmd/define` — PASS, package still green.
- [ ] **Step 5:** commit — `#29: every alternative in a gloss, not the first one`

---

### Task 3: `SourceSpellings`

**Files:** modify `cmd/define/audiourl.go`; test `cmd/define/audiourl_test.go`.

**Contract.** `SourceSpellings(typed string, e Entry) []string` — the headword, the `AlsoSpellings`, and the typed word; each lowercased and whitespace-collapsed; deduped preserving order; **then stably ordered so spellings carrying a non-ASCII rune come first**; empties dropped.

**Why it exists.** The CDN keys a source recording on the source ORTHOGRAPHY, not on what a person types:

```
jalapeno_en_us  200   ← what define asked for before #29
jalapeño_es_es  200   ← the Spanish recording
jalapeno_es_es  404   ← the same word, Spanish locale, unaccented
```

Swapping the language field in the existing URL builder is therefore not enough. Same split measured for `piñata`/`pinata`, `señor`/`senor`, `café`/`cafe`, `naïve`/`naive`, `façade`/`facade`, `cliché`/`cliche`, `fiancé`/`fiance`.

**Why NON-ASCII FIRST rather than headword first.** NOAD files the accented form on either side of the headword, so "the dictionary's own form is best" is right only 5 times in 9:

| typed | headword | `(also …)` | which spelling is the 200 |
|---|---|---|---|
| `jalapeno` `pinata` `senor` `cliche` `fiance` | accented | — | the headword |
| `cafe` `naive` `facade` | **unaccented** | `café` `naïve` `façade` | **the alternative** |
| `role` | **unaccented** | *none* | **unreachable** — see below |

Ordering by "carries a non-ASCII rune" is right 8 times in 8, and saves the `café` class two 404s at the ~300–600 ms per miss that `atlas/define.md` prices. Where nothing carries an accent (`arrondissement`) the order is unchanged, so this costs nothing in the common case.

`role` is the ninth borrowing and no rule here reaches it — recorded so it is inherited rather than rediscovered. `rôle_fr_fr` is a 200 and `role_fr_fr` a 404, but NOAD heads the entry `role`, offers no `(also rôle)`, and spells the accented form only inside ORIGIN: *"from French rôle, from obsolete French roule 'roll'"*. That sentence offers three candidate tokens, so mining it is a parsing problem rather than a fourth lookup; `role` degrades to English and says so.

**Why deduping matters more than it looks:** for `arrondissement` all three sources agree, and without it the walk asks the CDN the same question three times.

**Why lowercasing must reach the headword-derived spelling:** `Señor_es_es` is a 404 where `señor_es_es` is a 200.

- [ ] **Step 1:** table test over `jalapeno`→`[jalapeño jalapeno]`, `cafe`→`[café cafe]` (non-ASCII first — this cell fails a headword-first implementation), `arrondissement`→`[arrondissement]` (one spelling, no wasted request), `senor` with head `Señor`→`[señor senor]`, and an empty entry→`[ciao]` (an unparseable entry still leaves the typed word).
- [ ] **Step 2:** run — FAIL, undefined.
- [ ] **Step 3:** implement.
- [ ] **Step 4:** run — PASS.
- [ ] **Step 5:** commit — `#29: the source recording is keyed on the source orthography`

---

### Task 4: `utterance` — the whole walk, in one pure place

**Files:** modify `cmd/define/audiourl.go`; test `cmd/define/audiourl_test.go`.

**Contract.**

```go
type utterance struct {
	Word      string   // what was typed; the SESSION recording is keyed on it, unchanged from before #29
	Spellings []string // SOURCE orthographies, best first, from SourceSpellings
	Source    voice    // what -pron asked for; a zero Lang means nothing was asked for
	Session   voice    // the fallback, and the only voice when Source is zero
}

func (u utterance) sourceCandidates() []string  // one AudioCandidates run per spelling; nil when Source.Lang == "" || Source == Session
func (u utterance) Candidates() []string        // sourceCandidates, then AudioCandidates(Word, Session)
func (u utterance) spokeSource(from string) bool // membership in sourceCandidates — never a URL parse
```

**Why `Source == Session` yields nothing:** `/pron es` inside a Spanish session requests the recording already being fetched; asking twice is two requests for one answer.

**Why `spokeSource` tests membership rather than reading the language out of the path:** the report it feeds is a RECORD — it survives on a pipe and cannot be taken back — and a record has to be true (`workshop/lessons.md`, *"Ephemeral UI vs. a record"*). A URL parse would be a second, driftable statement of what a source URL looks like, wrong the day the CDN generation moves.

**Why the fallback exists at all** — source coverage is partial, measured 2026-08-28/29: `hotel_fr_fr` and `debut_fr_fr` are 404 while their `_en_us_` are 200; Italian is absent from this CDN generation entirely (`ciao`, `pizza`, `espresso`, `opera`), as is Japanese (`karaoke`, `tsunami`). Without it those words play nothing.

- [ ] **Step 1:** four tests — (a) an utterance with no source is **byte-identical** to `AudioCandidates(word, session)`, which is the no-regression assertion for every existing caller; (b) every source spelling precedes the session's, and the walk ENDS at the session's recording; (c) `Source == Session` does not double the walk; (d) `spokeSource` is true for a source URL, false for the session's, false for a URL in neither list.
- [ ] **Step 2:** run — FAIL, undefined.
- [ ] **Step 3:** implement. `Candidates` allocates a fresh slice — do not `append` onto the slice `sourceCandidates` returned.
- [ ] **Step 4:** `go test ./cmd/define` — PASS, whole package green.
- [ ] **Step 5:** commit — `#29: one utterance owns the walk — source first, session as the fallback`

---

## Chunk 2: the shell

### Task 5: `speak`, `playAnnounced`, and the honest report

**Files:** modify `cmd/define/main.go`; update every `playAnnounced` call site — `cmd/define/repl.go:317`, `cmd/define/replraw.go:306` and `:325`, `cmd/define/play_loop.go` (the `OutcomeReveal` branch), and `defineOnce` (`main.go:672`). Test `cmd/define/fetch_test.go`.

**Contract.**

- `speak(ctx, d, u utterance, n int) (from string, err error)` — returns the URL that ANSWERED, which it currently fetches and discards. That is what lets the caller report from what happened rather than from what was requested.
- `reportVoice(w io.Writer, u utterance, from string)` — writes `define: no <src> recording for <word>; played the <session> one` to **stderr**, and only when a source was asked for, differs from the session, and did not answer. On the success path of `playAnnounced`, after the erase.
- `utteranceFor(word, entry string, pron store.Lang, opt options) utterance` — ONE builder for all five call sites, so the source spellings are derived identically at each. Two paths that each decide what to ask the CDN is `#14`'s *"two loops, one decision table"* in a new costume, and that divergence is silent because each path is individually tested. `entry` may be empty (the review session has no raw text): `SourceSpellings` then falls back to the typed word. Uses `voiceFor(pron, opt.locale)` — D4, `#27`'s function unchanged.

**Why the report is stderr and late:** it is a record, not ephemeral UI. Announcing it up front would file the claim before the fact — the defect `workshop/lessons.md` records under *"Ephemeral UI vs. a record"*.

- [ ] **Step 1:** three tests covering the whole truth table — (a) source asked for and MISSING: the English recording plays AND stderr names both languages (drive it with `it`/`ciao`, the measured-absent case); (b) source asked for and PRESENT: stderr is **empty**, so the report cannot become noise on every lookup; (c) no source asked for: stderr is empty even though the session's recording is what played.
- [ ] **Step 2:** run — FAIL, `playAnnounced` does not take an `utterance`.
- [ ] **Step 3:** implement, and thread `utteranceFor` through all five call sites.
- [ ] **Step 4:** `go test ./cmd/define`. **`TestTheFetchLoopAsksOnlyForTheSessionsLanguageWhenNoneWasNamed` must pass unchanged** — an ordinary lookup's request list is byte-identical to before. Task 8 is where that name gained its qualifier.
- [ ] **Step 5:** commit — `#29: report the voice that answered, not the one that was asked for`

---

### Task 6: the `-pron` flag

**Files:** modify `cmd/define/voice.go` (`pronHelp` beside `localeHelp`), `cmd/define/repl.go` (`replCommand.pron`), `cmd/define/main.go`. Test `cmd/define/main_test.go`.

**How `pron` reaches the play site, and why not through `options`.** `defineOnce` is shared by the one-shot and the piped loop, and it already receives a `replCommand` — the type whose doc says *"what one line of input means"*, and which already carries `literal`, a per-line modifier of exactly this kind. So:

- `replCommand` gains `pron store.Lang`.
- `run()` sets `oneShot.pron = pron` after validating the flag.
- `defineOnce` calls `playAnnounced(…, utteranceFor(cmd.word, out.entry, cmd.pron, opt), …)`.
- `parseREPLLine` never sets it, so every line either loop parses carries the zero value and no source attempt.

A field on `options` is the easy path and is refused: it would recreate the session-scoped value D3 exists to prevent, and `applyLang` would carry it across a `/lang` switch.

**`pronHelp`** is the one source for what `-pron` means, with the README and atlas as consumers (Task 8) — the same mechanism `localeHelp` uses, and for the same reason: the `-locale` policy was written in four places with nothing keeping them in step.

**Validation** sits beside `-lang`'s (`main.go` ~line 488), before any directory is opened, because this value becomes a path segment. It uses `store.ParseLang`, so the message is `ParseLang`'s and not a second one invented here.

**Refusal.** A new case in the argument-count switch, beside `--play`/`--reflect`:

```
case pron != "" && oneShot.kind != cmdDefine:
    "define: -pron applies to one lookup; at the prompt use /pron fr"  → exit 2
```

- [ ] **Step 1:** tests — (a) `-pron fr` with no word exits 2 and the message NAMES `/pron`; (b) `-pron french` exits 2 via `ParseLang`; (c) end to end against the fake CDN: `-pron fr arrondissement` requests the French path FIRST, `d.lang` is still `en`, and the capturer filed the word in the ENGLISH deck — which is the issue's first Done-when.
- [ ] **Step 2:** run — FAIL, `flag provided but not defined: -pron`.
- [ ] **Step 3:** implement.
- [ ] **Step 4:** run — PASS.
- [ ] **Step 5:** commit — `#29: -pron fr — one lookup in another language, session untouched`

---

### Task 7: `/pron` — the in-session action

**Files:** create `cmd/define/pron_cmd.go`; modify `cmd/define/command.go`, `cmd/define/repl.go`, `cmd/define/replraw.go`. Test `cmd/define/pron_cmd_test.go`, `cmd/define/repl_test.go`.

**Contract.**

- `parsePronArgs(args []string) (store.Lang, error)` — a language is REQUIRED. This is where it differs from `parseLangArgs`/`parseSoundArgs`, which split "set" from "report" because they name a SETTING with a current value worth printing. `/pron` names an action and leaves nothing behind, so a bare one is a half-typed command, not a question. Accepts loosely (`FR` works), matching dispatch's `EqualFold`.
- `runPron` — does NOT play. It records the request through `c.replay` and the LOOP performs it, because commands dispatch inside the raw editor's cooked block. A nil `c.replay` means there is no current word — the one-shot path, or a loop before the first lookup — and saying so beats playing silence, the same call `/sound` makes for a nil `setTimes`.
- `commands` gains one row: `{name: "pron", summary: "replay this word in another language, once", run: runPron}`. The dispatch loop still never grows a case, which is a `#15` Done-when, and `/help` lists the registry so the row appears with no second edit.
- `commandCtx.replay func(store.Lang)` — a closure like `setLang`/`setTimes`, nil where there is no current word.
- `replayInPlace` gains the language, so **a bare Enter and `/pron fr` are ONE replay path with one parameter** rather than two implementations (ARCH-DRY; `#14`'s two-loops lesson). It needs the session's raw entry for `SourceSpellings`, which `session.entry` already holds.

Both loops dispatch, then perform:

```go
var pending store.Lang
… cc.replay = func(l store.Lang) { pending = l }   // only when sess.hasCurrent()
… dispatchCommand(cmd, commands, cc)
if pending != "" { replayInPlace(ctx, d, opt, sess, pending, stdout, stderr) }
```

- [ ] **Step 1:** tests — (a) `parsePronArgs` table: bare rejected, two languages rejected, `FR`→`fr`; (b) **the no-mode assertion**, which is D2 and the one that would catch someone "simplifying" this into a session field later: drive the piped loop with `arrondissement`, `/pron fr`, `police` and assert the fake CDN saw `arrondissement_en_us…`, then `arrondissement_fr_fr…`, then `police_en_us…`; (c) `/pron fr` with nothing looked up says so rather than playing silence.
- [ ] **Step 2:** run — FAIL, undefined.
- [ ] **Step 3:** implement.
- [ ] **Step 4:** `go test ./cmd/define` — PASS, including the `/help` fixture tests.
- [ ] **Step 5:** commit — `#29: /pron fr — say it that way once, leave no mode behind`

---

## Chunk 3: docs and live conformance

### Task 8: dispose of EVERY statement of the invariant this change falsifies

**Files:** `cmd/define/audiourl.go`, `cmd/define/audiourl_test.go`, `atlas/define.md`, `README.md`, `cmd/define/main.go` (`fs.Usage`), `cmd/define/doc_sync_test.go`.

`#23`'s "one language, no fallback" is asserted in **five** places. The enumeration is the deliverable — `workshop/lessons.md` records three rounds lost to sweeping the file a fix touched instead of the class. Each member is disposed of explicitly, including the two that need no change:

| # | site | disposition |
|---|---|---|
| 1 | `audiourl.go:24-26` — *"ONE language, never a search across languages … `#27`'s planned `voices()` fallback — a mode does not need one"* | **REWRITE.** Task 4 adds `utterance.Candidates()` — the cross-language walk — to this same file, so shipping this comment means the file asserts the opposite of what it does. It becomes a statement about `AudioCandidates` alone, pointing at `utterance` for the walk. |
| 2 | `atlas/define.md:1119-1123` — *"One language, no fallback (`#23`)"* | **REWRITE**, not append. State both halves: the session still asks for one language, and `-pron` adds an explicit second voice tried first — with D5's reason the two cases differ. |
| 3 | `README.md:189` — *"Everything follows it — the deck a word files into, the words `--play` offers, and the recording that is fetched"* | **AMEND.** The last clause is now conditional: the recording follows the language unless `-pron`/`/pron` asked otherwise for one lookup. |
| 4 | `audiourl_test.go:64` — *"One language, and the legacy path for English only"* | **KEEP.** It documents `TestAudioCandidatesSpanish`, which tests `AudioCandidates`, which really is still one language. Enumerated so the decision is recorded rather than the site merely missed. |
| 5 | `audiourl_test.go:184` — the fetch-loop test's NAME | **NARROWED**, to `…WhenNoneWasNamed`, plus a `retiredSymbolNames` row so the rename sweeps itself. The test still passes (it passes no `-pron`), but the name now over-claims: with `-pron` the loop deliberately asks for another language first. A name that asserts a false invariant is the same defect as a comment that does. |

Then the atlas gains, in the same section: the source-orthography constraint with the `jalapeño`/`jalapeno` measurement, the `(also …)` filter with the 400-entry survey behind it, and D1's `police_fr_fr` 200 — the row a future reader will want when they wonder why this is not automatic.

- [ ] **Step 1:** add `TestDocsQuoteThePronHelp`, modelled exactly on `TestDocsQuoteTheLocaleHelp`, over both `../../README.md` and `../../atlas/define.md`, pinning `<!-- pron-help -->` + `pronHelp` + `<!-- /pron-help -->`. Run it and watch it FAIL, naming both docs.
- [ ] **Step 2:** work the table above, top to bottom.
- [ ] **Step 3:** README flag table, the `/pron` row in the command list, and the `-pron` sentence in `fs.Usage`.
- [ ] **Step 4: verify the deletions, do not assert them.** `workshop/lessons.md`: a `replace()` on reflowed text is a silent no-op and a commit message is not evidence.

```bash
grep -rn "never a search across languages" cmd/define/     # NOTHING
grep -n  "One language, no fallback"       atlas/define.md # NOTHING
grep -n  "and the recording that is fetched" README.md     # NOTHING
grep -rn "AsksOnlyForTheSessionsLanguage"  cmd/define/     # narrowed, not bare
grep -c  "pron-help" README.md atlas/define.md             # 1 each
go test ./cmd/define
```

- [ ] **Step 5:** commit — `#29: the tree said one language and no fallback in five places`

---

### Task 9: live conformance rows (ARCH-MOCK)

**Files:** modify `cmd/define/fetch_conformance_test.go`.

**Every new test name MUST begin `TestCDN`.** The file's header documents its cadence as `go test -tags conformance -run CDN ./cmd/define/`, and the existing rows (`TestCDNStillServesTheExpectedPaths`, `TestCDNStillServesSpanishOnTheExpectedPaths`, `TestCDNReturnsRealAudio`) all match it — a row that does not is silently skipped and its verification step passes vacuously. `workshop/lessons.md` already records this class: *"A live conformance check that is never run is not a check"* — `TestPTYSuggestionAndAcceptance` sat RED through two merges for exactly this reason.

**Fix the trap as well as the instance:** amend that header comment so the documented cadence is the **whole-file** form (`go test -tags conformance ./cmd/define/`), because a `-run` filter is precisely the mechanism that lets a row decay unnoticed. That is lesson rule 2 — *run the on-demand suites at a close, not only the default `go test ./...`* — turned into the thing the file actually tells you to type.

Rows to add, each failing with the DECISION it invalidates rather than a URL:

- [ ] `TestCDNStillKeysSourceRecordingsOnTheSourceSpelling` — `jalapeño_es_es` 200 **and** `jalapeno_es_es` not-200. If the second starts answering, `SourceSpellings` is carrying weight it no longer needs.
- [ ] `TestCDNStillCannotTellALoanwordFromANaturalisedOne` — `police_fr_fr` 200. This is D1's evidence; if it stops answering, the argument against ORIGIN inference weakens and the decision deserves re-opening.
- [ ] `TestCDNItalianIsStillAbsentFromThisGeneration` — `ciao`, `pizza`, `espresso` all not-200. If Italian arrives, the reporting path stops firing for it and the atlas's limitation note is stale.
- [ ] `TestCDNFrenchCoverageIsStillPartial` — `hotel`/`debut` 404 on `fr_fr` and 200 on `en_us`. This is why the fallback exists.
- [ ] **Run the WHOLE suite, unfiltered:** `go test -tags conformance ./cmd/define/ -v`. Not `-run CDN` — that is the filter this task exists to stop trusting.
- [ ] commit — `#29: pin the measurements the design rests on, live`

---

## Verification before close

In a scratch directory, so no deck is touched:

```bash
cd "$(mktemp -d)"
define arrondissement            # English recording, English entry
define -pron fr arrondissement   # French recording, same English entry
define -pron es jalapeno         # reaches jalapeño_es_es, NOT jalapeno_es_es
define -pron fr hotel            # fr 404 → English plays + "no fr recording for hotel"
define -pron it ciao             # Italian absent → English plays + says so
define -pron fr                  # refused, points at /pron
ls words/                        # ONLY en/ — no fr/, no es/. The session never moved.
```

Then in the loop: `arrondissement`, `/pron fr`, `police` — `police` plays English with nothing to undo (D2).

And both opt-in suites, per `workshop/lessons.md` rule 2: `go test ./... && go test -tags conformance ./cmd/define/`.

**Done-when coverage** (`#29`): (1) Tasks 5–7 + the `ls words/` check; (2) Task 4's fallback + Task 5's report, at `hotel`/`debut` — **not `déjeuner`**, which has no NOAD entry and never reaches audio; (3) Task 5's report + Task 9's Italian row; (4) D1, with Task 9 pinning its evidence; (5) Tasks 1–3, at `jalapeno`; (6) D4 — `voiceFor` unchanged, so the locale is literally `#27`'s. D6 disposes of the Spec's third option.

**Close:** single pass, plain checkboxes, no `Mx` — one review boundary, one `sdlc close`.

---

## Revisions

### 2026-08-29 — first plan-quality round (PQ-1…PQ-7)

**Reason:** `sdlc change-code`'s stateful plan gate raised two Important and five Minor findings. All seven addressed; the two Important ones were fixed as the CLASS they name, not the site.

- **PQ-1 (Important, `doc-sweep-incomplete`)** — Task 8 named only the atlas paragraph. Grepping the class found **five** sites, including `audiourl.go:24-26` in the very file Task 4 adds the cross-language walk to. Task 8 is now an enumeration table disposing of all five, two of them explicitly as "keep, and here is why", with a grep in Step 4 for each deletion.
- **PQ-2 (Important, `conformance-row-never-runs`)** — `TestFrenchCoverageIsStillPartial` does not match the `-run CDN` filter the file documents, so it would have been skipped and its verification step would have passed vacuously. Renamed `TestCDNFrenchCoverageIsStillPartial`, every new row given the `TestCDN` prefix, **and the trap itself fixed**: the file's cadence comment now documents the unfiltered whole-file run, since a `-run` filter is the mechanism that lets a row decay.
- **PQ-3 (Minor, `first-match-not-all-matches`)** — VERIFIED by running `ParseEntry`: one gloss really does carry `"(also naïve) (also naïveness)"`, and the reversed order parses too. A first-match read would silently drop the source spelling. `AlsoSpellings` now scans every occurrence, and Task 2 tests **both orderings**.
- **PQ-4 (Minor, `candidate-order-by-measurement`)** — VERIFIED: headword-first is right 5/8, non-ASCII-first 8/8, so `SourceSpellings` now orders by non-ASCII first. `rôle_fr_fr` 200 / `role_fr_fr` 404 was measured while checking and was FIRST RECORDED HERE AS A NINTH INSTANCE OF THE ORDERING RULE, which running the chain against the live dictionary disproved: NOAD carries `rôle` only inside ORIGIN prose, so no ordering of the three sources reaches it. Corrected in Task 3's contract as a known limitation rather than left as a claim the code does not support.
- **PQ-5 (Minor, `unstated-seam-threading`)** — resolved without an `options` field, which the finding correctly flagged as recreating what D3 refuses. `-pron` rides on `replCommand`, which `defineOnce` already receives and which already carries `literal`, a per-line modifier of the same kind. Task 6 states the four-line path.
- **PQ-6 (Minor, `unstated-non-goal`)** — the Spec's third option (labelling which notation variant is which) is now **D6**, disposed of with its reason rather than left unanswered at close.
- **PQ-7 (Minor, `plan-restates-the-diff`)** — the plan was 1131 lines, mostly pre-written bodies and doc comments that get rewritten within the hour. Rewritten to state each function's CONTRACT and the CLASS its tests must cover. The measured tables are kept — they are the evidence, and evidence does not regenerate. For `differsOnlyByDiacritics`, which runs over arbitrary gloss text, Task 1 Step 4 now names the malformed-input class (invalid UTF-8, decomposed combining marks, huge input) instead of fourteen hand-picked pairs.
