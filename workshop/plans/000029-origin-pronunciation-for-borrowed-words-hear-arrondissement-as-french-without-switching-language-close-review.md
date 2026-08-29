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
