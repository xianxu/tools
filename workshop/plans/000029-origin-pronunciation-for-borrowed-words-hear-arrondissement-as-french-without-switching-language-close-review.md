# Boundary Review — tools#29 (whole-issue close)

| field | value |
|-------|-------|
| issue | 29 — origin pronunciation for borrowed words: hear arrondissement as French, without switching language |
| repo | tools |
| issue file | workshop/issues/000029-origin-pronunciation-for-borrowed-words-hear-arrondissement-as-french-without-switching-language.md |
| boundary | whole-issue close |
| milestone | — |
| window | a9ea60371a708172c0347261d2c110af8879200f..380a24f0a4a0170054f1ee0d94e2bf272af3da84 |
| command | sdlc close --issue 29 |
| reviewer | claude |
| timestamp | 2026-08-29T08:13:51-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The feature is real, correct, and unusually well evidenced: I ran `go test ./...` (green, 97s) and the whole unfiltered `-tags conformance` suite (green, 104s), mutation-verified all four test-double fixes the Log claims (each goes red when reverted), and confirmed by scratch test that the raw editor's `/pron` path works and plays **outside** the cooked block. No correctness bug survived verification. What holds it back from SHIP is three cheap gaps: the issue's headline Done-when (the session does not move) has no automated assertion despite living in a test named for it; the raw-loop `/pron` branch — the one with the real hazard — has no committed test at all, although the harness for it already exists; and the new atlas H2 was inserted mid-section, re-parenting ~18 lines and one H3 that belong to `## Pronunciation`.

## 1. Strengths

- **`utterance` is the right seam.** `spokeSource` answering by membership in the list actually built (`cmd/define/audiourl.go:224`) rather than by parsing a language back out of a URL is what makes the report a record instead of a prediction — and `TestAnUtteranceAsksTheSourceFirstAndFallsBackToTheSession` pins the no-source walk as *byte-identical* to `AudioCandidates`, which is the no-regression assertion every existing caller needed.
- **The three test-double corrections are the strongest part of the diff (ARCH-MOCK).** `fakeCDN`'s `EscapedPath`, `fakeDictionary`'s accent-insensitivity, and `rebasedSource` translating the answer back are each *load-bearing*, not tidying. I mutation-reverted all three: each takes a specific test red with a message that names the real defect (`first request = "…/jalapeño_es_es_1.mp3", want "…/jalape%C3%B1o_es_es_1.mp3"`, `no dictionary entry`, `stderr should be empty`). This is the "a fix is complete only when a test fails without it" bar, met.
- **The five-site invariant sweep was enumerated rather than grepped-and-hoped**, including two sites explicitly disposed as "keep, and here is why" (`cmd/define/audiourl_test.go:64`) and a `retiredSymbolNames` row that turns the rename into a build failure — which I verified fires by putting the old name back into `atlas/define.md`.
- **The plan-table guard now expresses a method, and it is pinned.** Mutating the row to `Entry.AlsoSpellingsZ` and to `Block.AlsoSpellings` both fail `TestPlanTablesNameEntitiesThatExist` with distinct messages, so the receiver pinning is real and the `new`-row exemption correctly stopped exempting once the boxes were ticked.
- **Four live conformance rows, all passing, each failing with the DECISION it invalidates** rather than a URL — `TestCDNStillCannotTellALoanwordFromANaturalisedOne` is the row that would justify reopening D1, and it says so.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — the first Done-when is delivered but unpinned; the test named for it does not check it.** `cmd/define/main_test.go:320` `TestPronFetchesTheSourceRecordingWithoutMovingTheSession` asserts the first request URL, the walk length, the printed entry and empty stderr — nothing about the session. The plan's Task 6 Step 1(c) promised "`d.lang` is still `en`, and the capturer filed the word in the ENGLISH deck", and `newAudioRigServing` wires `noopCapturer{}`, so nothing could have recorded it. I could not verify it live either: NOAD is unreachable from this process context, so the dictionary conformance rows *skip* here and the plan's manual `ls words/` check is not reproducible. Fix: swap in a `countingCapturer` (`cmd/define/capture_test.go:73`) or a `newStoreCapturer` over a memory store as `askrun_test.go:34` does, and assert the recorded language / deck path.

**I-2 — the raw editor's `/pron` branch has no test (ARCH-DRY, the two-loops rule).** `cmd/define/replraw.go:223-251` is the only place the record-in-cooked / perform-in-raw split exists, and it is the hazard `workshop/lessons.md`'s "render cooked, play raw" exists for; `cmd/define/repl_test.go` was listed in Task 7's Files and never touched. The repo already pairs `TestLineLoopDispatchesCommands` with `TestRawEditorDispatchesCommands` for exactly this reason. It is ~20 lines with the existing `editorRig`/`scriptKeys` harness — and the cooked-block property is directly assertable by counting CDN requests inside the `cooked` callback. I wrote that test as a scratch and it passes today, so this is coverage, not correctness.

**I-3 — `atlas/define.md:1170`: the new H2 was inserted mid-section and re-parented the prose after it.** `## Source pronunciation (#29)` now sits between the `<!-- locale-help -->` block and the paragraph that explains it, so "`localeHelp` is that string, and both this page and the README derive from it" (:1249), the `-locale` English-only history (:1254), the `playN` note (:1258) and "A missing recording is **not** a failed lookup" (:1263) all read as part of the source-pronunciation section — as does the H3 `### The dictionary follows the language (#23 M2)` (:1266), which is now a child of the wrong H2. Fix: move the whole new section to just before `## Conformance` (:1329).

**I-4 — the plan's Core-concepts table contradicts the tree on the test doubles.** `workshop/plans/000029-origin-pronunciation-plan.md:80` still reads `` | `fakeCDN` | `cmd/define/fetch_fake_test.go` | unchanged — REUSED | ``, but the diff changed its keying semantics; `fakeDictionary` and `rebasedSource` were also changed and appear in no table row. The issue's Log records all three honestly — the *plan* does not, and the plan is the artifact a reader trusts to describe the design. Marked Important rather than Critical because no runtime behaviour is at stake; the fix is a `## Revisions` entry (see §7), not code.

## 4. Minor findings

- `cmd/define/main.go:856` — `reportVoice` prints the *undefaulted* `Lang` fields while `AudioCandidates` (`audiourl.go:117`) defaults an empty `Lang` to `store.DefaultLang`. With a zero `opt.voice` the record reads `played the  one` — I hit exactly that in a scratch run. Production always calls `applyVoice`, so it is latent, but the record is the one thing this design insists must be true. Report the normalised value.
- `cmd/define/audiourl.go:180` vs `:233` — `sourceCandidates`' guard and `askedForSource` are the same predicate written twice, negated; the latter's own comment says they must agree (ARCH-DRY). Have `sourceCandidates` call `askedForSource()`.
- `cmd/define/dict_fake_test.go:79` — the accent-insensitive fallback iterates a Go map, so two entries differing from the query only by diacritics give a nondeterministic answer. Iterate sorted keys.
- `README.md:29-41` — the `### define` cheat-sheet lists `--sound`, `-no-audio`, `-locale`, `-lang`, `-raw`, `-no-color` but not `-pron`; Task 8 Step 3 named "README flag table". One line: `define -pron fr arrondissement   # the French recording, English everything else`.
- `cmd/define/repo_guard_test.go:593` — `inProgress` is a document-wide `- [ ] ` scan and the status match is `Contains(lower(m[3]), "new")`, so "renewed"/"newly" would also exempt a row. Low risk while gated on in-progress; worth an anchored match.
- The Spec's split-out French/Italian/German dictionary-curation win was never filed as an issue; it survives only as prose in `#30:92-97` ("nobody has taken yet"). It is correctly *deferred* (the operator is explicitly not learning those languages, so it is not this issue's purpose — ARCH-PURPOSE passes), but a promised split with no tracker item is how a cheap win evaporates.

## 5. Test coverage notes

Suites run by me, not taken on report: `go test ./...` green (97s); `go test -tags conformance ./cmd/define/` green unfiltered (104s), with the four new `TestCDN*` rows individually verified passing against the live CDN. Note that the NOAD-backed rows — including the new `TestLiveDictionaryResolvesAnUnaccentedQuery` — **skip** in this environment, so the live half of `fakeDictionary`'s model is only checked where the dictionary is reachable; that is by design (`conformance.SkipOrFail`) but means it should be run somewhere NOAD answers before close.

Mutation results, all four claimed fixes confirmed: `EscapedPath`→`Path` reddens two tests; dropping `rebasedSource`'s origin translation reddens two; dropping `fakeDictionary`'s diacritic fallback reddens two; the plan-table receiver matching reddens under both a wrong-symbol and a wrong-receiver mutation.

Gaps are I-1 and I-2. Also uncovered, and cheaper than they look: `/pron` when `opt.noAudio` (both loops route to `nothingToReplay`, newly extracted as a shared const — nothing pins that either loop still reaches it), and `/pron es` inside an already-Spanish session (`Source == Session`) end-to-end, which is unit-tested on `utterance` but not through a loop.

## 6. Architectural notes

- **ARCH-DRY — pass, one Minor flag.** `nothingToReplay`, `replayPiped`, `utteranceFor` as the single builder for all five play sites, and `replayInPlace` parameterised rather than forked are all the right calls; `fakeDictionary` folding through the production predicate is a genuinely good instance. Flagged: the duplicated `askedForSource` predicate.
- **ARCH-PURE — pass.** `differsOnlyByDiacritics`, `Entry.AlsoSpellings`, `SourceSpellings`, `utterance` and `parsePronArgs` are all deterministic and unit-tested with no IO, no mocks; the shell (`speak`/`playAnnounced`/`reportVoice`/`utteranceFor`) stayed thin and takes its dependencies injected. `speak` returning the URL that answered is what moved the decision out of the IO layer rather than adding a second one inside it.
- **ARCH-PURPOSE — pass, with I-1.** Shadow-sweep on the single-source change: `pronHelp` has three consumers and all three *derive* — the flag registration (`main.go:409`, confirmed in the built binary's `-h` output), README and atlas via `TestDocsQuoteThePronHelp`. The `#23` invariant sweep enumerated five sites and disposed of each, including two as explicit keeps. D6 disposes of the Spec's third option rather than deferring it. No hand-maintained restatement of the model remains.
- **ARCH-MOCK — pass, and the strongest axis here.** Production and test flow share the `AudioSource` seam; the fake is stateful (ordered path recording) and three separate divergences from the real dependency were found and corrected rather than worked around; every measurement the design rests on has a live row that fails with the decision it invalidates. The `-run CDN` filter removal is the right fix for the trap, not just the instance.
- For `#30` (clickable regions), `Entry.AlsoSpellings` and `SourceSpellings` are the surfaces it will consume; both are pure and table-tested, so it inherits a good seam. The `role`/ORIGIN-mining limitation is honestly recorded in three places and is the natural first thing `#30`'s click dissolves.

## 7. Plan revision recommendations

One `## Revisions` entry, dated 2026-08-29, reason "implementation-phase measurement corrected the plan's model of the test doubles":

- **Core concepts / Integration points table** — `fakeCDN` is no longer `unchanged — REUSED`: it was modified to key on `EscapedPath` because `r.URL.Path` percent-decodes and 404'd every accented URL the real CDN serves. Add rows for `fakeDictionary` (`cmd/define/dict_fake_test.go`, modified — now accent-insensitive on a miss, through the production predicate) and `rebasedSource` (`cmd/define/main_test.go`, modified — translates the answering URL back, because `#29` made `from` load-bearing).
- **Task 7 Files** — `cmd/define/repl_test.go` was named and not written; either record it as unwritten (I-2) or tick it once the raw-loop test lands.
- **Task 6 Step 1(c)** — records a `d.lang` + capturer-deck assertion that was not written (I-1); the delivered test asserts the CDN walk only.
- Note also, for the record rather than as a revision: PQ-4's entry was **edited in place** during implementation rather than superseded by a new dated entry. The rewritten text is honest about the disproof, so nothing is lost — but AGENTS.md §1 asks for append, not overwrite, and appending is what makes the correction legible as a second measurement rather than a first one.

```findings
findings:
  - id: new
    severity: Important
    family: done-when-unpinned
    title: |
      The issue's first Done-when — the session does not move — has no automated assertion, in the test named for it
    detail: |
      cmd/define/main_test.go:320 TestPronFetchesTheSourceRecordingWithoutMovingTheSession
      asserts the first CDN URL, the walk length, the printed entry and empty stderr, and
      nothing about the deck, the dictionary or d.lang. The rig wires noopCapturer, so it
      could not have recorded one. Task 6 Step 1(c) promised exactly that assertion. NOAD
      is unreachable from this process context, so the plan's manual `ls words/` check is
      also not reproducible at this gate. Use countingCapturer (capture_test.go:73) or a
      store-backed capturer as askrun_test.go:34 does.
  - id: new
    severity: Important
    family: two-loops-one-test
    title: |
      The raw editor's /pron branch is untested, though it carries the design's only real hazard
    detail: |
      cmd/define/replraw.go:223-251 is the record-in-cooked / perform-in-raw split that
      lessons.md's "render cooked, play raw" exists for. Task 7 named cmd/define/repl_test.go
      and it was not touched. The repo already pairs TestLineLoopDispatchesCommands with
      TestRawEditorDispatchesCommands for this reason, and editorRig/scriptKeys make it a
      ~20-line test; the cooked-block property is assertable by counting CDN requests inside
      the cooked callback. I verified by scratch test that the path works today, so this is
      coverage rather than correctness.
  - id: new
    severity: Important
    family: heading-reparents-prose
    title: |
      atlas/define.md:1170 — the new H2 was inserted mid-section and re-parented four paragraphs and an H3
    detail: |
      "## Source pronunciation (#29)" sits between the locale-help block and the paragraph
      explaining it, so the localeHelp derivation note (:1249), the -locale English-only
      history (:1254), the playN note (:1258), "A missing recording is not a failed lookup"
      (:1263) and the H3 "### The dictionary follows the language (#23 M2)" (:1266) now read
      as part of the source-pronunciation section. Move the new section to just before
      "## Conformance" (:1329).
  - id: new
    severity: Important
    family: plan-table-vs-tree
    title: |
      The plan's Core-concepts table calls fakeCDN "unchanged — REUSED" after the diff changed it
    detail: |
      workshop/plans/000029-origin-pronunciation-plan.md:80 claims fakeCDN was reused
      untouched; the diff changed its keying from r.URL.Path to r.URL.EscapedPath, which is
      load-bearing. fakeDictionary and rebasedSource were also modified and appear in no
      row. The issue's Log records all three honestly; the plan does not. Fix with a
      "## Revisions" entry, not code.
  - id: new
    severity: Minor
    family: report-must-use-the-normalised-value
    title: |
      reportVoice prints undefaulted Lang fields while AudioCandidates defaults them
    detail: |
      cmd/define/main.go:856 formats u.Source.Lang and u.Session.Lang raw, while
      AudioCandidates (audiourl.go:117) substitutes store.DefaultLang for an empty Lang. With
      a zero opt.voice the record reads "played the  one" — observed in a scratch run.
      Production always calls applyVoice, so it is latent, but this line is the record the
      design insists must be true.
  - id: new
    severity: Minor
    family: one-predicate-two-spellings
    title: |
      utterance.sourceCandidates and utterance.askedForSource write the same predicate twice, negated
    detail: |
      audiourl.go:180 guards on `u.Source.Lang == "" || u.Source == u.Session`; audiourl.go:233
      returns its negation. The latter's comment says the two must agree — make it so by
      construction: sourceCandidates should call askedForSource() (ARCH-DRY).
  - id: new
    severity: Minor
    family: nondeterministic-fake
    title: |
      fakeDictionary's accent-insensitive fallback iterates a Go map
    detail: |
      cmd/define/dict_fake_test.go:79 — with two entries that both differ from the query only
      by diacritics, the entry returned is nondeterministic. Iterate sorted keys.
  - id: new
    severity: Minor
    family: doc-sweep-incomplete
    title: |
      README's define cheat-sheet lists every other flag a reader types, but not -pron
    detail: |
      README.md:29-41 shows --sound, -no-audio, -locale, -lang, -raw and -no-color; Task 8
      Step 3 named "README flag table". The prose section further down does document -pron,
      so this is completeness rather than a gap in the docs gate.
  - id: new
    severity: Minor
    family: guard-heuristic-too-loose
    title: |
      The plan-guard's new-row exemption matches any status cell containing "new"
    detail: |
      cmd/define/repo_guard_test.go:593 scans the whole document for "- [ ] " and then
      exempts a row whose status cell lower-cases to contain "new" — "renewed" or "newly"
      would exempt too. Low risk while gated on in-progress, but an anchored match is one
      character more.
  - id: new
    severity: Minor
    family: split-out-not-filed
    title: |
      The Spec's split-out dictionary-curation win was never filed as an issue
    detail: |
      #29's Spec says the fr.Multi / it.Devoto-Oli / de.DDDSI curation is "split out so the
      cheap win is not blocked on this design", but no issue exists; it survives only as
      prose in #30:92-97 ("nobody has taken yet"). Correctly deferred — it is not this
      issue's purpose — but a promised split with no tracker item evaporates.
```

---

## Re-review — 2026-08-29T08:42:14-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 29 — origin pronunciation for borrowed words: hear arrondissement as French, without switching language |
| repo | tools |
| issue file | workshop/issues/000029-origin-pronunciation-for-borrowed-words-hear-arrondissement-as-french-without-switching-language.md |
| boundary | whole-issue close |
| milestone | — |
| window | a9ea60371a708172c0347261d2c110af8879200f..fd9a71c903e1f3117119355189586c7f8b625398 |
| command | sdlc close --issue 29 |
| reviewer | claude |
| timestamp | 2026-08-29T08:42:14-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All four blocking findings from the prior round are genuinely fixed, and I verified the two that a commit message alone could have faked: reverting the fix reddens the test in both cases (`applyVoice(&opt, pron)` reddens `main_test.go:368`; moving the replay inside the cooked closure reddens `commandloop_test.go:486`). BR-3's atlas move is a pure reorder — 80 lines out, 80 in, identical multiset, no prose lost. The design itself is sound: `utterance` owns the walk, `spokeSource` answers by membership rather than by parsing a URL back, and the three test doubles were corrected to model the dependency instead of the assertions being loosened. `go build ./...`, `go vet ./...`, `go test ./...` and the live `go test -tags conformance ./cmd/define/` are all green here, and I independently re-probed the CDN facts the design rests on. What holds SHIP is one real coverage hole — Done-when 6 / D4 (`-locale` qualifies the source language) is asserted nowhere: I replaced `voiceFor(pron, opt.locale)` with `voiceFor(pron, "")` and the entire package stayed green — and a plan row that still claims `AudioCandidates` is "unchanged" after the diff changed it, which is BR-4's family surviving the round that closed it.

### 1. Strengths

- **`utterance.spokeSource` by membership, and the fake was fixed to match** — `cmd/define/audiourl.go:216` refuses to read a language back out of a path, and `rebasedSource` (`cmd/define/main_test.go:69-90`) was changed to translate the answer back rather than letting the rig's own rewriting leak into the value under assertion. That is the harder and correct direction.
- **`differsOnlyByDiacritics` states its precondition and tests the class** — `cmd/define/parse.go:78-83` rejects invalid UTF-8 *because* `[]rune` turns a stray byte into U+FFFD, which reads to the loop as a diacritic. Pinned by `TestDiacriticsOnlyRefusesMalformedInput` and a fuzz target, not by more equality cases.
- **BR-2's new test asserts the property, not the outcome** — counting CDN requests made *inside* the cooked callback (`commandloop_test.go:470-487`) pins the record-in-cooked / perform-in-raw split directly, which is the design's only real hazard.
- **The retired-symbol row works despite a hostile shape** — the retired name is a strict prefix of its replacement; the `\b…\b` match still fires. I reverted the atlas mention to the old name and `TestNoArtifactNamesARetiredSymbol` went red (`repo_guard_test.go:755`).
- **`langOrDefault` fixed the class, not the line** — BR-5 could have been a one-line format change; instead the defaulting rule became one accessor (`voice.go:24`) with both readers deriving from it (ARCH-DRY).

### 2. Critical findings

None.

### 3. Important findings

**I-1 — Done-when 6 / D4 is unpinned; the whole suite is green with `-locale` dropped from the source voice** (`cmd/define/main.go:883`)

> **This is the 2nd finding in family `done-when-unpinned`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

`utteranceFor` builds the source voice as `voiceFor(pron, opt.locale)`. I changed it to `voiceFor(pron, "")` and `go test ./cmd/define` passed. `utteranceFor` has **zero direct test call sites** (grep: only `main.go`, `repl.go`, `replraw.go`, `play_loop.go` reference it), and every test that reaches it runs with `opt.locale == ""`, where the two spellings are identical. So `-pron es -locale us` → `es_us` — the issue's sixth Done-when and the plan's D4 — is asserted nowhere.

The rule, which is what should actually be fixed: **a Done-when is pinned only by a named test that goes red when the wiring is removed. A structural argument — "`voiceFor` is unchanged, so the locale is literally `#27`'s" — is not a pin, because the *call site* is new code and can be miswired without touching the reused function.** The plan's "Done-when coverage" line is where this is decidable: every cell must name a `Test…` symbol, and each named test must be shown red under removal of the wiring it claims to pin. Measured prevalence on this issue: of six Done-whens, the two whose coverage cell names a *task* or a *decision* rather than a test are exactly the two that turned out unpinned — Done-when 1 last round (BR-1, cell named "Tasks 5–7 + the `ls words/` check") and Done-when 6 this round (cell named "D4 — `voiceFor` unchanged"). 2 for 2. The sweep is the deliverable; the `-locale` test falls out of it.

**I-2 — the plan's Core-concepts table still claims `AudioCandidates` is "unchanged" after the diff changed it** (`workshop/plans/000029-origin-pronunciation-plan.md:79`)

> **This is the 2nd finding in family `plan-table-vs-tree`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

Row 8 of the Pure-entities table reads `| AudioCandidates | cmd/define/audiourl.go | unchanged — reused once per spelling |`. The diff rewrites ten lines of its doc comment — mandated by the plan's own Task 8, site 1 — and changes its body (`v.Lang = v.langOrDefault()`, from the BR-5 fix). Same shape as BR-4: a row asserting "unchanged" about a symbol the window modifies. BR-4 was disposed by hand-adding three rows; the fourth wrong row was in the table the whole time.

The rule: **"unchanged"/"reused" in a Core-concepts table is a claim about the DIFF, not about behaviour, and it is mechanically checkable.** `TestPlanTablesNameEntitiesThatExist` already parses the status cell (`repo_guard_test.go:579`) — extend it so a row claiming `unchanged`/`reused` fails when the named symbol's *declaration region* appears in the plan's change window, and a row claiming `modified` fails when it does not. Declaration-level, not file-level: `voice.go` is modified while `voiceFor`/`localeFor`/`defaultLocale`/`applyVoice` genuinely are not, and that row is correct. Measured prevalence: 4 of the table's 20 rows have been wrong across two rounds (`fakeCDN`, `fakeDictionary`, `rebasedSource`, `AudioCandidates`), and the hand-fix round caught three of four. Include Task 7's `**Files:**` line in the sweep while you are there — it still names `cmd/define/repl_test.go`, which was never touched; the test landed in `commandloop_test.go` beside its `TestRawEditorDispatchesCommands` pair, which is the right place.

### 4. Minor findings

- **M-1 (`doc-sweep-incomplete`, 2nd in family)** — `atlas/define.md:737-741`'s command table lists `/help`, `/history`, `/sound`; the registry has five. `/pron` (this issue) and `/lang` (`#23`) are both absent, so the last two commands added both missed it — 2 of 2. **Do not add the row.** The rule: `commands` (`command.go:20-30`) is the single source and every doc that *enumerates* commands must be a derived consumer pinned by a doc-sync test — the exact mechanism `TestDocsQuoteTheLocaleHelp` / `TestDocsQuoteThePronHelp` already give `localeHelp`/`pronHelp`. One test closes `/lang` and `/pron` together and stops the next one. (The adjacent sentence at `:734` — "a command cannot reach the dictionary or the player" — is worth re-reading in the same pass now that `commandCtx.replay` exists; it is still literally true, since the closure records and the loop plays, but a reader would not learn `/pron` exists from it.)
- **M-2 (`guard-heuristic-too-loose`, 2nd in family)** — `repo_guard_test.go:597` now exempts a row when `strings.Fields(status)[0] == "new"`. That fails the other way: a bolded `**new**` cell is not exempted, and bold status cells are this repo's live convention — the `#29` plan itself writes `**modified**` in three rows. **Do not tweak the match again.** The rule: the status column is a controlled vocabulary (`new`/`modified`/`unchanged`/`deleted`); normalise the cell (strip markdown emphasis and trailing prose) and match it against that vocabulary, failing loudly on an unrecognised status — rather than a substring, a first-word, or any other positional heuristic. That is also the hook I-2 needs.
- **M-3 (new family `check-expects-the-wrong-outcome`)** — Task 8 Step 4 (`plan.md:332`) expects `grep -n "and the recording that is fetched" README.md # NOTHING`, but the same task's own table dispositions that site as **AMEND**, so the phrase must survive. It does, at `README.md:195`. The step is ticked `[x]` under a heading that says "verify the deletions, do not assert them… a commit message is not evidence". A check whose expected output contradicts the change it verifies either was not run or was run and waved through; correct the expectation to `# amended, still present`.
- **M-4 (new family `probe-subset-of-the-walk`)** — the three new negative conformance rows (`fetch_conformance_test.go:135`, `:163`, `:180`) probe only `AudioCandidates(…)[0]`, while the claim they pin — "Italian is absent", "French coverage is still partial", "`jalapeno_es_es` is a 404" — is about the whole list production walks. A recording appearing only at the `_2` suffix would leave every row green while the fallback stopped firing. I re-probed both suffixes live: `hotel`, `debut`, `jalapeno`, `ciao`, `pizza`, `espresso` are 404 on `_1` and `_2` today, so nothing is currently false. Fix by asserting through `newHTTPAudioSource().Fetch(ctx, AudioCandidates(w, v))` returning `ErrNoAudio`, which is the production shape and what `TestCDNReturnsRealAudio` already does for the positive case.

### 5. Test coverage notes

- The live dictionary rows (`TestLiveDictionaryResolvesAnUnaccentedQuery`, `TestFixturesMatchLiveDictionary`) **SKIP** from this process context — NOAD is unreachable here, same as the prior round — so the plan's `## Verification before close` CLI script is not reproducible at this gate: every lookup returns `no dictionary entry`. The CDN half and the whole unit suite are reproducible and green. The refusal path is verifiable without a dictionary and I ran it: `define -pron fr` → `define: -pron applies to one lookup; at the prompt use /pron fr`, exit 2, and no `words/` directory created.
- `TestPronWithNothingLookedUpSaysSo` (`pron_cmd_test.go:59`) discards `run`'s exit code. `runPron` returns 2 and the README documents 2 for a usage error; asserting it costs one line and pins the code path `fail()` exists for.
- The three-cell truth table in `TestPlayAnnouncedReportsTheVoiceThatAnswered` is the right shape — it includes both silent cells, so the report cannot degrade into a line on every lookup.

### 6. Architectural notes

- **ARCH-DRY — pass.** BR-6's fix is the right one: `sourceCandidates` now *calls* `askedForSource()` instead of spelling its negation, so the comment promising they agree became unnecessary. `nothingToReplay` (`repl.go:185`) and `replayInPlace` taking `pron` as a parameter rather than growing a second replay are both the same move. The one duplication I-2 and M-1 name is in the *documentation* layer, not the code.
- **ARCH-PURE — pass.** `SourceSpellings`, `utterance`, `differsOnlyByDiacritics`, `Entry.AlsoSpellings` and `parsePronArgs` are all pure and tested with no IO; `utteranceFor`, `reportVoice` and `runPron` are thin glue. `reportVoice` taking an `io.Writer` rather than reaching for stderr is correct and is what made `TestTheVoiceReportNamesALanguageEvenWithAZeroVoice` a two-line test.
- **ARCH-PURPOSE — flag, at I-1.** The shadow-sweep on `pronHelp` passes: `fs.String("pron", …, pronHelp)`, README and atlas all derive, pinned by `TestDocsQuoteThePronHelp`. The sweep on the `commands` registry does not (M-1). And the issue's purpose is delivered rather than the easy subset — both `-pron` and `/pron` ship, `#31` files the split-out curation, D6 disposes of the notation-labelling option — but Done-when 6 is delivered as an *argument* rather than as something the tree defends, which is the axis this principle is about.
- **ARCH-MOCK — pass.** No new external dependency; the CDN keeps its stateful `fakeCDN` behind `AudioSource`, and the `EscapedPath` fix is a genuine correction to the fake's model of the dependency rather than a test accommodation. `fakeDictionary` now folds through the production predicate with sorted iteration, and `TestLiveDictionaryResolvesAnUnaccentedQuery` is its live conformance half. M-4 is the one place a conformance row models less than the production flow it stands in for.

### 7. Plan revision recommendations

Append one `## Revisions` entry, dated 2026-08-29, covering:

- **`AudioCandidates` was marked `unchanged` and the window changed it** — its doc comment was rewritten by this plan's own Task 8 site 1, and its body gained `v.Lang = v.langOrDefault()` from the BR-5 fix. Record the row as `modified`, and record that the status column is now checked mechanically against the change window (I-2) so the next round's table cannot claim what the diff contradicts.
- **Task 7's `**Files:**` line names `cmd/define/repl_test.go`, which was never touched** — the raw-editor `/pron` test landed in `cmd/define/commandloop_test.go` beside `TestRawEditorDispatchesCommands`, which is where it belongs. The prior Revisions entry says the test now exists but not that it lives elsewhere.
- **`## Verification before close`'s "Done-when coverage" cells 4 and 6 name decisions, not tests** — replace each cell with the test symbol that goes red without the wiring, and record that cell 6 had no such test (`voiceFor(pron, "")` left the suite green) until I-1's sweep.
- **Task 8 Step 4's third grep expects `NOTHING` for a site the task dispositions as AMEND** — correct the expectation so re-running the block verifies the change rather than contradicting it.

---

## Re-review — 2026-08-29T09:17:02-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 29 — origin pronunciation for borrowed words: hear arrondissement as French, without switching language |
| repo | tools |
| issue file | workshop/issues/000029-origin-pronunciation-for-borrowed-words-hear-arrondissement-as-french-without-switching-language.md |
| boundary | whole-issue close |
| milestone | — |
| window | a9ea60371a708172c0347261d2c110af8879200f..540b12f50fb459f432e5e4fabcb192d0df206be0 |
| command | sdlc close --issue 29 |
| reviewer | claude |
| timestamp | 2026-08-29T09:17:02-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All ten open findings (BR-1…BR-10) are genuinely addressed, and I verified the four that a commit message could have faked by mutation rather than by reading: re-adding `applyVoice(&opt, pron)` *after* `applyVoice(&opt, d.lang)` reddens `main_test.go:368`; moving the replay inside the cooked closure reddens `commandloop_test.go:486`; `voiceFor(pron, "")` reddens `main_test.go:400`; flipping the plan's `AudioCandidates` row to `unchanged` reddens the new plan-status guard; dropping the session tail from `Candidates()` reddens two tests. The feature itself is sound and I found no correctness defect in it. What blocks the gate is that **`go test ./...` is RED at `540b12f`** — `TestEverySkipIsRoutedOrWaived` fails on two unwaived `t.Skip` sites that commit `8832440` introduced. The base commit is green, so this window broke it, and the plan's own `## Verification before close` names `go test ./...` as the evidence. Beyond that, the ~200 lines of new guard machinery that answered round 2's family findings landed without being reviewed themselves: the emphasis-stripping that *is* M-2's rule is entered by no fixture (removing it leaves the suite green), and the new git calls swallow errors into a silent skip in the one file whose own `git()` helper documents why that was removed before.

## 1. Strengths

- **The three test-double corrections remain the strongest part of the diff (ARCH-MOCK).** `fakeCDN`'s `EscapedPath`, `fakeDictionary` folding through `differsOnlyByDiacritics` with `slices.Sorted(maps.Keys(...))`, and `rebasedSource` translating the answering URL back (`main_test.go:66-90`) are each load-bearing. `TestLiveDictionaryResolvesAnUnaccentedQuery` is the live half of the model.
- **`utterance` is the right seam, and `spokeSource` by membership is the right call** (`audiourl.go:216-226`). The report is written after the fetch, from what answered, and the walk-order test asserts the no-source case is *byte-identical* to `AudioCandidates`.
- **BR-5 was fixed as a class, not a line** — `voice.langOrDefault` (`voice.go:24`) became one accessor with both readers deriving from it, pinned by `TestTheVoiceReportNamesALanguageEvenWithAZeroVoice`.
- **The plan-status guard genuinely works, including its subtlest claim.** Declaration-level rather than file-level is correct (the `voiceFor/localeFor/defaultLocale/applyVoice` row survives while `voice.go` changed), and I confirmed the doc-comment expansion is load-bearing: deleting the backwards walk over `//` lines reddens on `rebasedSource`.
- **`TestDocsQuoteTheCommandList` closed `/lang` and `/pron` together** rather than adding the missing rows — the class-level answer M-1 asked for.
- **Conformance is green and non-vacuous**: `go test -tags conformance ./cmd/define/` passes unfiltered (105s), and the four new `TestCDN*` rows pass live. `missesEntirely` asserting through `newHTTPAudioSource().Fetch` is the right production shape.

## 2. Critical findings

**C-1 — `go test ./...` is RED at the review head** (`cmd/define/repo_guard_test.go:825`, `:857`)

```
--- FAIL: TestEverySkipIsRoutedOrWaived (0.07s)
    guard_test.go:90: 2 skip site(s) neither routed through conformance.SkipOrFail nor waived:
        cmd/define/repo_guard_test.go:825  t.Skip("no active plans")
        cmd/define/repo_guard_test.go:857  t.Skip("no unchanged/modified rows pointed at files this window touched")
```

I ran it on a clean tree at `540b12f` and again at the base `a9ea6037`, where `internal/conformance` is **ok**. So this window introduced it, in `8832440` — the commit that answered round 2. `TestPlanTableStatusMatchesTheChangeWindow` added three skip sites; the author waived the first one correctly (`repo_guard_test.go:816`, `conformance:inapplicable — on a merged branch there is no window`) and missed the other two in the same function.

Fix: give both skips a `conformance:inapplicable — <why>` comment on the line or in the three above it — and for `:857` decide first whether "checked nothing" should be a skip at all, since the sibling at `:617` reasons it out explicitly while this one does not.

The rule underneath: the evidence was measured before the last two commits, not after them. Re-run `go test ./... && go test -tags conformance ./cmd/define/` on the final HEAD and put *that* in `--verified`.

## 3. Important findings

**I-1 — `planStatus`'s emphasis stripping is entered by no fixture; the rule M-2 asked for shipped unpinned** (`cmd/define/repo_guard_test.go:639-646`)

> **This is the 3rd finding in family `done-when-unpinned`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

I replaced `strings.Trim(fields[0], "*_`")` with `fields[0]` and `go test ./cmd/define -run TestPlanTable` stayed **green**. No plan in the tree writes a bolded status cell, so the branch that handles `**new**` — the entire substance of round 2's M-2 — is protection that has never been exercised. The out-of-vocabulary `t.Errorf` branch is likewise unreached. This is the third spelling of this one parse (`Contains(…, "new")` → `Fields(…)[0] == "new"` → vocabulary), and the first two both shipped broken.

The rule, which is the widening the family now needs: **the "observed red when the wiring is removed" discipline the plan applies to Done-when cells applies to *every* fix delivered in answer to a finding, not only to Done-whens.** A finding-fix with no test that reddens without it is `not-addressed`, however plausible the diff. Concretely: `planStatus` is a pure function taking a string and returning `(string, bool)` — it wants a table test (`new`, `**new**`, `` `modified` ``, `*unchanged*`, `Modified — gains X`, `renewed` → not-ok, `` `` → not-ok), and the same pass applied to this round's other fixes would have caught it before the gate.

**I-2 — the new guard's git calls swallow errors into a silent skip, in the file whose own helper documents why that is wrong** (`cmd/define/repo_guard_test.go:864`, `:869`, `:881`)

`repo_guard_test.go:45-53` already has `git(t, args...)` with the comment: *"Deliberately Fatal, never Skip. The previous version skipped on any git error, so it was a silent no-op in an exported tree or without git on PATH — a guard that reports nothing when it cannot run certifies nothing."* `changeWindowBase` and `changedLines` bypass it. `changedLines` returns `nil` on `err != nil`, which is the same value it returns for "this window did not touch the file" — so a git failure downgrades every `modified`/`unchanged` row to unchecked, silently. `changeWindowBase` turns any `merge-base` failure into `t.Skip`, conflating "on main / no window" (legitimately inapplicable) with "git is unavailable" (the guard did not run). This is the `check-that-cannot-fail-reads-as-green` family `internal/conformance/guard_test.go:12-27` records four rounds of.

Fix: route `rev-parse HEAD` and `diff --unified=0` through the existing `git()` helper (Fatal), and keep the skip only for the one genuinely inapplicable case — `merge-base` failing because there is no `main` — with the waiver comment C-1 needs anyway.

## 4. Minor findings

- **`checkPlanName` and `checkPlanStatus`/`declarationRegion` write the same symbol locator twice** (`repo_guard_test.go:656-685` vs `:915-921` + `:938-941`) — the receiver-split block is duplicated verbatim and the declaration regexes are written twice, and they have *already* diverged: `checkPlanName` also accepts an `assigned` form (`^name :?=`) that `declarationRegion` returns `ok=false` for and silently skips. **This is the 2nd finding in family `one-predicate-two-spellings`.** Do not fix the site: the rule BR-6 established — one named function, the second call site calls it — applies here as `splitReceiver(name, path) (recv, bare string)` and `declRegexp(name, recv) *regexp.Regexp`, both shared. Prevalence: 2 instances on this issue, the second landing in the commit that fixed the first.
- `repo_guard_test.go:812-858` — the guard checks *every* active plan against *this branch's* window, so with two plans in flight touching a shared file, the other issue's `modified` row fails here for work that legitimately happened on another branch. Scope the check to the plan whose issue the window belongs to, or skip rows from plans the window does not otherwise touch.
- `cmd/define/replraw.go:249` — `pron = ""` after `replayInPlace` is dead: `pron` is declared inside the `cmdCommand` block and does not outlive the iteration.
- `cmd/define/pron_cmd_test.go:65` — `TestPronWithNothingLookedUpSaysSo` still discards `run`'s exit code; `runPron` returns 2 and the README documents 2 for a usage error. One line pins the path `fail()` exists for. (Raised as a coverage note last round, never as a finding.)

## 5. Test coverage notes

- Ran by me, not taken on report: `go build ./...` and `go vet ./...` clean; `go test ./...` **FAILS** (see C-1) with `cmd/define` itself green at 97.8s; `go test -tags conformance ./cmd/define/` green unfiltered at 105s, with the four new `TestCDN*` rows individually passing live.
- Mutation results — confirmed pinned: BR-1 (`applyVoice(&opt, pron)` after the session's derivation → `main_test.go:368`), BR-2 (replay inside the cooked closure → `commandloop_test.go:486`), Done-when 6 (`voiceFor(pron, "")` → `main_test.go:400`), Done-when 2 (`Candidates()` stops appending the session tail → two tests), the plan-status guard (`modified`→`unchanged` on `AudioCandidates`), and the doc-comment half of `declarationRegion` (via `rebasedSource`). Confirmed **not** pinned: `planStatus`'s emphasis stripping (I-1).
- A useful negative result for anyone repeating this: mutating `opt.voice` *above* `applyVoice(&opt, d.lang)` (`main.go:614`) reads as "the assertion is blind" when it is not — the mutation never survives to the read. `workshop/lessons.md` now records this; it is correct and worth keeping.
- The NOAD-backed rows (`TestLiveDictionaryResolvesAnUnaccentedQuery`, `TestFixturesMatchLiveDictionary`) **SKIP** from this process context, third round running. The plan's `## Verification before close` CLI script is therefore still not reproducible at this gate; it needs one run somewhere NOAD answers before close.

## 6. Architectural notes

- **ARCH-DRY — flag, at the Minor above.** The feature code is clean: `utteranceFor` as the single builder for all five play sites, `replayInPlace` parameterised rather than forked, `nothingToReplay` as one const, `sourceCandidates` calling `askedForSource()`, `fakeDictionary` folding through the production predicate. The duplication is entirely in the new guard machinery, and it is the second instance of the family BR-6 opened.
- **ARCH-PURE — pass.** `differsOnlyByDiacritics`, `Entry.AlsoSpellings`, `SourceSpellings`, `utterance` and `parsePronArgs` are deterministic and unit-tested with no IO and no mocks. `reportVoice` taking an `io.Writer` is what made `TestTheVoiceReportNamesALanguageEvenWithAZeroVoice` a two-line test. `speak` returning the URL that answered moved the decision out of the IO layer rather than adding a second one inside it.
- **ARCH-PURPOSE — pass on the feature, flag on the finding-class discipline.** Shadow-sweep: `pronHelp` has three consumers (flag registration, README, atlas) and all three derive, pinned; the `commands` registry now has the atlas as a derived consumer, and the README states each command in prose with all five present, so no hand-maintained enumeration remains; the `#23` invariant sweep's five sites are all disposed and I re-ran the plan's own greps (all five give the expected result). `#31` files the split-out curation, D6 disposes of the notation-labelling option. The flag is I-1: the class-level answers to round 2 were written into the *code* but not into the *tests*, so one of them does nothing.
- **ARCH-MOCK — pass.** No new external dependency; production and test flow share the `AudioSource` seam; the fake is stateful and ordered; every measurement the design rests on has a live row that fails with the decision it invalidates, and `missesEntirely` now walks the whole candidate list the way production does. The new `git` calls are test-only and consistent with the file's existing practice — the objection to them is I-2's error handling, not the seam.
- For `#30`, `Entry.AlsoSpellings` and `SourceSpellings` remain the surfaces it will consume; both are pure and table-tested. The `role`/ORIGIN-mining limitation is recorded in three places and is the first thing a click dissolves.

## 7. Plan revision recommendations

The plan now matches the tree on the Core-concepts tables (mechanically, per `TestPlanTableStatusMatchesTheChangeWindow`), and the three `## Revisions` entries cover rounds 1 and 2 honestly. One entry is still owed, dated 2026-08-29, reason "close review round 3":

- **`## Verification before close` was ticked on evidence measured before the last two commits.** `go test ./...` is red at HEAD; record that the suite must be re-run on the *final* HEAD, after the last fix commit, not after the round the fixes answered.
- **Widen the "Done-when coverage" rule from Done-when cells to every finding-fix.** The plan already states "a Done-when is pinned only by a NAMED TEST observed red under removal of the wiring". Round 3 found the same rule broken one level up: `planStatus`'s emphasis stripping — the class-level fix for `guard-heuristic-too-loose` — is green when removed. Record that a finding disposed `addressed` names the test that reddens without it, or is not addressed.

---

## Re-review — 2026-08-29T09:48:45-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 29 — origin pronunciation for borrowed words: hear arrondissement as French, without switching language |
| repo | tools |
| issue file | workshop/issues/000029-origin-pronunciation-for-borrowed-words-hear-arrondissement-as-french-without-switching-language.md |
| boundary | whole-issue close |
| milestone | — |
| window | a9ea60371a708172c0347261d2c110af8879200f..1cd3911b31d5bf4276f6147ec56dd4faccbe127e |
| command | sdlc close --issue 29 |
| reviewer | claude |
| timestamp | 2026-08-29T09:48:45-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All ten open findings (BR-1…BR-10) are genuinely addressed, and I re-verified the ones a commit message could have faked by mutation rather than by reading: adding `applyVoice(&opt, pron)` after the session's derivation reddens `main_test.go:368` with the deck assertion; `voiceFor(pron, "")` reddens `main_test.go:400`; replacing `strings.Trim(fields[0], "*_\`")` with `fields[0]` reddens four `TestPlanStatusNormalisesToTheVocabulary` cells. Round 3's blocker is gone: `go test ./...` is green here (97.9s, whole tree), as are `go build`, `go vet`, and the unfiltered `go test -tags conformance ./cmd/define/` (105s), with all four new `TestCDN*` rows passing live against the real CDN. I re-ran the plan's own five doc-sweep greps and each gives its stated result. No correctness defect survived verification, and nothing here blocks the boundary — the three Minors below are notes, one of which is an ask about what goes in `--verified` rather than a code change.

## 1. Strengths

- **`utterance` is the right seam, and `spokeSource` by membership is what makes the report a record.** `cmd/define/audiourl.go:216-226` refuses to read a language back out of a URL, and `TestAnUtteranceAsksTheSourceFirstAndFallsBackToTheSession` (`audiourl_test.go:330`) pins the no-source walk as *byte-identical* to `AudioCandidates` — the no-regression assertion every pre-existing caller needed. The walk-end assertion uses membership rather than `Contains("_en_us_")`, which is correct: an English walk ends on the legacy `/sounds/oxford/…--_us_2.mp3` path that carries no such marker.
- **The three test-double corrections remain the strongest part of the diff (ARCH-MOCK).** `fakeCDN`'s `EscapedPath` (`fetch_fake_test.go:29`), `fakeDictionary` folding through the *production* predicate with `slices.Sorted(maps.Keys(...))` (`dict_fake_test.go:85`), and `rebasedSource` translating the answering URL back are each load-bearing, not tidying — and the fake's model has a live half in `TestLiveDictionaryResolvesAnUnaccentedQuery`.
- **BR-5 and BR-9 were both answered as the class, not the line.** `voice.langOrDefault` (`voice.go:24`) became one accessor with both readers deriving from it; `planStatus` became a controlled vocabulary that fails loudly outside it — and this time every branch, including the emphasis-stripping and the loud-failure one, is entered by a fixture. That is the round-3 rule applied to itself.
- **The `#23` invariant sweep survives its own greps.** `never a search across languages` and `One language, no fallback` are both gone, `and the recording that is fetched` survives amended at `README.md:195`, the fetch-loop test name is narrowed with a `retiredSymbolNames` row behind it, and `pron-help` appears exactly once in each doc.
- **`TestDocsQuoteTheCommandList` closed `/lang` and `/pron` together** by generating the atlas table from the `commands` registry rather than adding the two missing rows — the class-level answer, and it is the mechanism `pronHelp`/`localeHelp` already use.
- **`TestRawEditorPronPlaysOutsideTheCookedBlock` asserts the property, not the outcome** (`commandloop_test.go:462-490`): it counts CDN requests made *inside* the cooked callback and requires zero, which pins the design's only real hazard directly.

## 2. Critical findings

None.

## 3. Important findings

None.

## 4. Minor findings

**M-1 (`plan-table-vs-tree`, 3rd in family) — the status guard checks rows against the tree, but never the tree against the rows, and skips any row whose file the window did not touch.**

> **This is the 3rd finding in family `plan-table-vs-tree`.** Earlier rounds fixed instances (BR-4 by hand, round 2 by mechanism). Do NOT fix these two instances — state the rule that covers them and fix that.

Two measured holes in `TestPlanTableStatusMatchesTheChangeWindow`:

- `cmd/define/repo_guard_test.go:855` — `touched := changedLines(...); if touched == nil { continue }`. A `modified` row pointing at a file this window never touched is silently unchecked. Verified: I added `| \`crlfWriter\` | \`cmd/define/crlf.go\` | modified |` to the `#29` plan in a scratch copy and both plan guards stayed green. That is precisely the "the row describes work that did not happen" case the guard's own error message names, and it is the *common* shape of a wrong row.
- Nothing checks the table for **completeness**. `voice.langOrDefault` is a new production entity created by this window (the BR-5 class fix) and has no row — while the `AudioCandidates` row's own status text names it (`plan.md:44`: "modified — one line (`langOrDefault`)"). `isASCIIOnly` (`audiourl.go:90`) and `nothingToReplay` (`repl.go:185`) are likewise new and rowless, and this plan does put test infra in scope (`newFakeCDN`, `fakeDictionary.Lookup`, `rebasedSource` all have rows).

The rule: **the Core-concepts table is a projection of the diff in BOTH directions.** Row→tree is now mechanical; tree→row is not, and row→tree fails open on untouched files. Concretely: move the `touched == nil` case *inside* `checkPlanStatus` so it reads as `inWindow = false` and the `modified` branch fires (if cross-milestone tolerance is wanted, say so and gate it, rather than skipping silently); and give `checkPlanStatus` a fixture table — its two `t.Errorf` branches are today entered only by mutating a real plan, which is the same "green when removed" state round 3 found in `planStatus`. Measured prevalence: 4 wrong rows across rounds 1–2, plus 2 holes and 3 missing rows now.

**M-2 (new family `raw-mode-bare-newline`) — `reportVoice` writes a bare `\n` to a stderr that is a raw terminal.**

`cmd/define/main.go:857` uses `fmt.Fprintf(w, "…\n")`, and the `/pron` path reaches it through `replayInPlace` (`replraw.go:330`) while the terminal is raw and `stderr` is *not* wrapped in `crlfWriter` — that wrapping exists only for the ask path (`replraw.go:172`) and the review loop (`play_loop.go:65`). Every sibling write in `replraw.go` spells `\r\n` explicitly (`:274`, `:325`, `:327`). Confirmed by scratch test: driving `runEditor` with `jalapeno\r/pron es\r` against a CDN serving only the English recording yields `stderr = "define: no es recording for jalapeno; played the en one\n"`. It is **masked today** because `runEditor` writes `\r\n` immediately after `replayInPlace` returns, so nothing is visibly staircased — hence Minor, not Important. Note the sibling defect pre-exists on `playAnnounced`'s error line (`main.go:824`), so the fix belongs at the seam (wrap stderr, or let the indicator carry the line ending) rather than on this one line. The `/pron` report is now the *common* case for Italian and Japanese, where the error line was rare.

**M-3 (new family `conformance-row-never-runs`) — the NOAD-backed live rows SKIP in every review environment, fourth round running.**

`TestLiveDictionaryResolvesAnUnaccentedQuery` and `TestFixturesMatchLiveDictionary` skip here with `NOAD unavailable: no dictionary entry` (all five subtests). So the chain the feature *is* — typed `jalapeno` → NOAD headword `jalapeño` → `jalapeño_es_es` — has never been executed against both real dependencies at any gate; the CDN half is verified live, the dictionary half only through the fake. The issue's Log records the implementor running `SourceSpellings` against the live dictionary during design, so this is an evidence-recording gap rather than an unmeasured claim. Ask: `sdlc close --verified` should name a run of `go test -tags conformance ./cmd/define/` **and** the plan's `## Verification before close` CLI script from somewhere NOAD answers, on this final HEAD — which is round 3's own "evidence has a timestamp" rule applied to the half that has never had one.

## 5. Test coverage notes

- Run by me, not taken on report, all on a clean tree at `1cd3911`: `go build ./...` and `go vet ./...` clean; `go test ./...` **green** (`cmd/define` 97.9s, all other packages ok); `go test -tags conformance ./cmd/define/` **green unfiltered** (105.3s), with `TestCDNStillKeysSourceRecordingsOnTheSourceSpelling`, `TestCDNStillCannotTellALoanwordFromANaturalisedOne`, `TestCDNItalianIsStillAbsentFromThisGeneration` and `TestCDNFrenchCoverageIsStillPartial` individually passing against the live CDN.
- Mutation results, mine this round: BR-1 pinned (`applyVoice(&opt, pron)` → `main_test.go:368`, "the word was filed under \"es\", want \"en\""); Done-when 6 pinned (`voiceFor(pron, "")` → `main_test.go:400`); BR-9's emphasis stripping pinned (4 red cells); `checkPlanStatus`'s `modified` branch confirmed reachable (forcing `inWindow = false` reddens 7 real rows). Confirmed **not** pinned: a fabricated `modified` row on a file outside the window (M-1).
- Deliberately uncovered and worth knowing: `/pron` with `opt.noAudio` in either loop (both route to `nothingToReplay`, nothing pins that either loop still reaches it), and the caching seam's key — `cachingAudioSource` keys on the whole joined candidate list (`fetch.go:109`), so a `/pron fr` walk correctly gets its own entry and correctly returns the URL that answered on a hit. I checked that specifically because `speak`'s new `from` return would report a false fallback if a cache hit returned `""`; it does not (`fetch.go:116`).

## 6. Architectural notes

- **ARCH-DRY — pass.** `utteranceFor` as the single builder for all five play sites; `replayInPlace` parameterised rather than forked so a bare Enter and `/pron fr` are one path; `nothingToReplay` as one const with the caller owning the ending; `sourceCandidates` calling `askedForSource()` (BR-6) rather than spelling it negated; `langOrDefault` as one accessor (BR-5); `splitReceiver`/`declRegexp` shared by both plan guards after the two copies had already diverged on the assigned form. Docs derive from `pronHelp` and from the `commands` registry. No new duplication found in this window.
- **ARCH-PURE — pass.** `differsOnlyByDiacritics`, `Entry.AlsoSpellings`, `SourceSpellings`, `utterance`, `parsePronArgs` and `planStatus` are deterministic and unit-tested with no IO and no mocks. `reportVoice` taking an `io.Writer` is what makes `TestTheVoiceReportNamesALanguageEvenWithAZeroVoice` a two-line test; `speak` returning the URL that answered moved the decision out of the IO layer rather than adding a second one inside it. The `utf8.ValidString` precondition in `differsOnlyByDiacritics` (`parse.go:78-83`) is a real one — `[]rune` turns a stray byte into U+FFFD, which reads to the loop as a diacritic — and it is pinned by a named test plus a fuzz target.
- **ARCH-PURPOSE — pass on the feature, flag at M-1.** Shadow-sweep on the single-source changes: `pronHelp` has three consumers (flag registration, README, atlas) and all three derive, pinned by `TestDocsQuoteThePronHelp`; `commands` now has the atlas as a derived consumer pinned by `TestDocsQuoteTheCommandList`; the `#23` invariant's five sites are all disposed, two as explicit keeps, and I re-ran every grep. D6 disposes of the Spec's notation-labelling option rather than deferring it, and `#31` files the split-out curation. The one remaining hand-maintained restatement of the model is the plan's own Core-concepts table, in the part the guard does not reach — that is M-1.
- **ARCH-MOCK — pass, with M-3.** No new external dependency; production and test flow share the `AudioSource` seam; the fake is stateful and records the walk in order, which is the only way the ordering assertions are possible; three divergences from the real dependency were found and corrected rather than worked around; `missesEntirely` asserts through `newHTTPAudioSource().Fetch` returning `ErrNoAudio`, which is the production shape. The gap is that the NOAD half of the conformance suite has never been observed green at a gate (M-3).
- For `#30` (clickable regions): `Entry.AlsoSpellings` and `SourceSpellings` are the surfaces it will consume; both are pure and table-tested, so it inherits a clean seam, and `commandCtx.replay` is already the "record a request, let the loop perform it" shape a click will need. The `role`/ORIGIN-mining limitation is honestly recorded in three places and is the first thing a click dissolves.

## 7. Plan revision recommendations

The plan now matches the tree mechanically on statuses, and the four `## Revisions` entries cover rounds 1–3 honestly, including the PQ-4 edited-in-place confession. One entry is owed, dated 2026-08-29, reason "close review round 4":

- **The Core-concepts table is incomplete, and the guard cannot see it.** Add rows for `voice.langOrDefault` (`cmd/define/voice.go`, new — the BR-5 class fix, and the symbol the `AudioCandidates` row already names in its status text), and record `isASCIIOnly` / `nothingToReplay` as new. Then state the rule the table is now under: it is a projection of the diff in both directions, and only one direction is mechanised.
- **Record the two holes in `TestPlanTableStatusMatchesTheChangeWindow`** — a `modified` row whose file the window never touched is skipped, and `checkPlanStatus`'s two error branches have no fixture — so the next reader knows the guard's actual reach rather than the reach its doc comment claims.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Mutation-verified: applyVoice(&opt, pron) after the session's derivation reddens main_test.go:368 on the countingCapturer deck assertion.
  - id: BR-2
    disposition: addressed
    note: |
      TestRawEditorPronPlaysOutsideTheCookedBlock counts CDN requests inside the cooked callback and requires zero.
  - id: BR-3
    disposition: addressed
    note: |
      The H2 now sits at atlas/define.md:1270, after the #23 M2 H3 and before ## Conformance; nothing is re-parented.
  - id: BR-4
    disposition: addressed
    note: |
      Revisions entry plus three added rows, and the claim is now mechanically checked at declaration level. See M-1 for the family's remaining holes.
  - id: BR-5
    disposition: addressed
    note: |
      voice.langOrDefault is one accessor with both readers deriving from it, pinned by TestTheVoiceReportNamesALanguageEvenWithAZeroVoice.
  - id: BR-6
    disposition: addressed
    note: |
      sourceCandidates now calls askedForSource() rather than spelling its negation.
  - id: BR-7
    disposition: addressed
    note: |
      dict_fake_test.go:85 iterates slices.Sorted(maps.Keys(d.entries)).
  - id: BR-8
    disposition: addressed
    note: |
      README.md:39 adds the -pron line to the define cheat-sheet.
  - id: BR-9
    disposition: addressed
    note: |
      planStatus is a controlled vocabulary failing loudly outside it; mutation-verified — removing the emphasis Trim reddens four fixture cells.
  - id: BR-10
    disposition: addressed
    note: |
      workshop/issues/000031-curate-dictionaries.md exists.
findings:
  - id: new
    severity: Minor
    family: plan-table-vs-tree
    title: |
      The status guard fails open on untouched files, and nothing checks the table is complete
    detail: |
      3rd in family — state the rule, do not fix the two instances. repo_guard_test.go:855
      skips any row whose FILE this window did not touch, so a `modified` row over an
      untouched file is unchecked: I added `| crlfWriter | cmd/define/crlf.go | modified |`
      to the #29 plan in a scratch copy and both plan guards stayed green. And nothing
      checks tree-to-row: voice.langOrDefault, isASCIIOnly and nothingToReplay are new in
      this window with no row, while the AudioCandidates row names langOrDefault in its own
      status text. The rule: the table is a projection of the diff in BOTH directions, and
      the touched==nil skip belongs inside checkPlanStatus as inWindow=false. Its two
      t.Errorf branches also have no fixture, which is the same green-when-removed state
      round 3 found in planStatus.
  - id: new
    severity: Minor
    family: raw-mode-bare-newline
    title: |
      reportVoice writes a bare \n to a stderr that is a raw terminal on the /pron path
    detail: |
      main.go:857 uses "\n" while every sibling write in replraw.go (:274, :325, :327)
      spells "\r\n", and stderr is wrapped in crlfWriter only for the ask path (replraw.go:172)
      and the review loop (play_loop.go:65) — not for replayInPlace. Confirmed by scratch
      test: runEditor driven with "jalapeno\r/pron es\r" against an English-only CDN gives
      stderr = "define: no es recording for jalapeno; played the en one\n". Masked today
      because runEditor writes "\r\n" right after replayInPlace returns, hence Minor. The
      sibling defect pre-exists on playAnnounced's error line (main.go:824), so the fix
      belongs at the seam, not on this line.
  - id: new
    severity: Minor
    family: conformance-row-never-runs
    title: |
      The NOAD-backed live rows SKIP in every review environment, fourth round running
    detail: |
      TestLiveDictionaryResolvesAnUnaccentedQuery skips all five subtests here with "NOAD
      unavailable: no dictionary entry", as does TestFixturesMatchLiveDictionary. So the
      chain the feature IS — typed jalapeno reaching jalapeño_es_es via NOAD's headword —
      has never run against both real dependencies at a gate; only the fake models the
      dictionary half. The issue's Log shows the implementor ran the chain against the live
      dictionary during design, so this is an evidence-recording gap. Ask: --verified should
      name a run of the unfiltered conformance suite AND the plan's CLI script from a
      context where NOAD answers, on this final HEAD.
```
