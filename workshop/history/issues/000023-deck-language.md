---
id: 000023
status: done
deps: []
github_issue:
created: 2026-08-27
updated: 2026-08-28
estimate_hours: 5.23
started: 2026-08-28T10:47:27-07:00
actual_hours: 9.20
---

# deck grouped by language, one language per --play session

## Problem

The deck has one namespace and `--play` reviews all of it. A learner working in
two languages gets `madrugar` and `sycophantic` in the same sitting, which is not
how anyone studies — and `#5`'s schedule interleaves them by due-date, so the
mixing is not even incidental.

Operator, after the first real `--play` session:

> words should be grouped in language and I think each invocation of
> `define --play` should just do one language.

This is `#18 M2`'s first bullet — *"a language dimension on the deck. One
directory per language works today and may be enough; decide deliberately rather
than by accident"* — pulled out because `#18 M2` is blocked on `#10`/`#12` for its
agreement-safe-distractor half, and this half is not.

## Spec

**`define` operates in ONE language at a time, and `/lang es` switches it.**
Operator's design, and it is better than the one this issue was filed with —
see Revisions. A mode, not a per-lookup flag.

- **`words/<lang>/`, one directory per language.** A Spanish session cannot see
  an English word even by accident, and "what am I learning in Spanish" is `ls`.
- **`/lang` reports the current language; `/lang es` switches it.** In the
  command namespace `#15` built, beside `/help`, `/history` and `/sound`.
- **The setting PERSISTS in the vocab directory**, unlike `/sound`, which is
  explicitly "for the rest of this session". Language cannot be session-scoped:
  a one-shot `define madrugar` has no session to inherit from, and re-declaring
  the language every time is exactly the friction this design removes. It is a
  property of the directory, which is already the unit everything else here
  scopes to.
- **Everything inherits the mode**: lookups file into that language's deck,
  `--play` reviews that language, and `#18 M1`'s audio asks for that language's
  recording — `madrugar_es_es_1.mp3` is a 200 where the `_en_us_` form this tool
  currently requests is a 404.
- **`-lang es` as a flag** for one-shot use without switching the mode, and
  because scripts should not have to mutate state to ask a question.

### The dictionary CAN be selected — measured, and it changes what is possible

This issue was first specified around a limit that does not exist. The claim —
carried in `dict_darwin.go`'s own comment and repeated into this issue — was that
`DCSCopyTextDefinition` must be passed NULL because the SDK exports no way to
build a `DCSDictionaryRef`. The operator pushed back; measurement says the claim
was wrong.

**True of the public header.** `DictionaryServices.h` declares exactly two
functions and documents the dictionary parameter as *"not supported for Leopard.
You should always pass NULL."*

**False of the framework.** `dlsym` resolves all of these:

```
DCSCopyAvailableDictionaries   DCSDictionaryGetName      DCSCopyDefinitionMarkup
DCSGetActiveDictionaries       DCSDictionaryGetIdentifier DCSDictionaryCreate
DCSDictionaryGetLanguages      DCSDictionaryGetShortName  DCSCopyRecordsForSearchString
```

87 dictionaries are available on this machine, including *Larousse Editorial
Diccionario General de la Lengua Española* and *Oxford Spanish Dictionary*. The
refs they return are accepted by `DCSCopyTextDefinition`. Measured, against the
six words this issue previously listed as unreachable:

| word | NULL (what the tool does today) | Spanish dictionary, selected |
|---|---|---|
| `mesa` | *an isolated flat-topped hill* | *nombre femenino — Mueble formado por un tablero horizontal* |
| `bonito` | *a smaller relative of the tunas* | *adjetivo (femenino bonita) — Que tiene belleza o atractivo* |
| `once` | *on one occasion* | *numeral cardinal — está 11 veces* |
| `real` | *actually existing as a thing* | *adjetivo — Que tiene existencia verdadera* |
| `madrugar` | *to get up early* (bilingual gloss) | *verbo intransitivo — Levantarse muy temprano, especialmente al amanecer* |
| `sycophantic` | the English entry | *(no entry)* — correctly not a Spanish word |

**Three things follow, and they make this issue bigger and better.**

1. **A language mode can be fully correct, not merely correct-at-filing.** In
   Spanish mode `mesa` is filed as Spanish AND defined as Spanish. The caveat
   this issue previously accepted — "filing it correctly is not the same as
   defining it correctly" — is gone.
2. **Monolingual beats bilingual for learning.** `madrugar` through NULL gives
   "to get up early"; through Larousse it gives a Spanish definition with a usage
   example. Reading the target language is the point of the exercise, and the
   better entry was there the whole time.
3. **"Not a word in this language" becomes answerable.** `sycophantic` in Spanish
   mode returns no entry, which is correct and which the tool cannot currently
   say about anything.

### Selecting the RIGHT dictionary is principled, not a name match

A second measurement, because the first probe matched dictionaries by name
substring and that is unsound: `DCSCopyAvailableDictionaries` returns a **CFSet**,
whose iteration order is unspecified, so `"Espa"` could match Larousse on one run
and Oxford Spanish on the next. The API offers a real key and real metadata:

```
New Oxford American Dictionary        id com.apple.dictionary.NOAD
  index=en_US  description=en_US                      -> monolingual English
Larousse Diccionario General          id com.apple.dictionary.es.DGLEV
  index=es     description=es                         -> monolingual SPANISH
Gran Diccionario Oxford               id com.apple.dictionary.OxfordSpanish
  index=es     description=es
  index=en     description=es                         -> bilingual
```

`DCSDictionaryGetIdentifier` gives a stable reverse-DNS id, and
`DCSDictionaryGetLanguages` gives an array of dictionaries keyed
`DCSDictionaryIndexLanguage` (what the headwords are) and
`DCSDictionaryDescriptionLanguage` (what the definitions are).

**The metadata NARROWS the candidates; it does not choose among them.** An earlier
draft of this issue said "the selection rule writes itself" — measured, and that
was an overclaim. Requiring every language entry to be L→L (strictly monolingual)
gives:

| language | strictly-monolingual candidates |
|---|---|
| `es` | **exactly one** — `com.apple.dictionary.es.DGLEV`, the Larousse |
| `en` | **six** — `NOAD`, `ODE`, `AppleDictionary`, `OAWT` and `OTE` (both THESAURUSES), and `com.apple.accessibility.dictionary.TTY` |

Nothing in the metadata says "general-purpose dictionary", so no rule over it can
prefer NOAD to a thesaurus. Proof it matters: a deterministic
smallest-identifier tiebreak — 10 runs, 10 identical results — picks
`com.apple.accessibility.dictionary.TTY` for English and the BILINGUAL
`OxfordSpanish` for Spanish. Deterministic and wrong is still wrong.

**So the design is: metadata narrows, a curated default decides, the learner can
override.**

- Narrow to dictionaries indexing L, preferring strictly monolingual.
- Among those, prefer a known-good identifier — `com.apple.dictionary.NOAD` for
  `en`, `com.apple.dictionary.es.DGLEV` for `es` — a short list, honestly a
  curated one, and easy to extend.
- Neither matches: fall back to today's NULL behaviour rather than guessing, and
  say which dictionary is in use so a wrong pick is visible rather than puzzling.
- The learner can name a dictionary explicitly, because on a machine with a
  different set installed no curated list will be right.

**Degradation stays honest:** no dictionary indexes L at all → no entry, rather
than silently answering from English.

**The cost, recorded rather than discovered.** These symbols are private and
undocumented: they can change or disappear on an OS update, and nothing in the
SDK promises otherwise. So the seam must `dlsym` them at run time and FALL BACK
to today's NULL behaviour when any is missing — which degrades to exactly what
ships now, rather than to a crash. That fallback is a Done-when row, not a nicety,
and it wants a conformance check like `#9`'s: an on-demand test that says loudly
when the private surface has moved.

## Done when

- [x] `words/<lang>/`, with existing decks migrated rather than orphaned.
- [x] `--play -lang es` reviews Spanish only; the default reviews English only.
- [x] `/lang` reports the current language; `/lang es` switches it; the setting
      survives the session ending.
- [x] A one-shot `define madrugar` uses the persisted language, with no session
      to inherit from.
- [x] A word shared with English is filed AND DEFINED in the current language —
      `mesa` in Spanish mode returns the Spanish entry, not the flat-topped hill.
- [x] A word absent from the current language reports no entry rather than
      silently answering from another language's dictionary.
- [x] The private DictionaryServices symbols are resolved at run time and the
      seam FALLS BACK to today's NULL behaviour if any is missing — degrading to
      what ships now rather than crashing.
- [x] A live conformance check says loudly when the private surface moves, on
      demand like `#9`'s feed check rather than in merge-check.
- [x] `--forget` and `d`-in-`--play` remove from the right language's deck.
- [x] The schedule and the event log are unchanged: language is a deck dimension,
      not an event one. A review event names a word; which deck it came from is
      the deck's business.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=1.00 impl=0.06
item: smaller-go-module        design=0.05 impl=0.16
item: cross-cutting-refactor   design=0.10 impl=0.20
item: smaller-go-module        design=0.05 impl=0.16
item: tui-screen               design=0.20 impl=0.24
item: smaller-go-module        design=0.05 impl=0.12
item: smaller-go-module        design=0.05 impl=0.16
item: smaller-go-module        design=0.00 impl=0.10
item: atlas-docs               design=0.04 impl=0.06
item: milestone-review         design=0.00 impl=0.20
item: smaller-go-module        design=0.05 impl=0.16
item: api-integration          design=0.40 impl=0.60
item: real-api-discovery       design=0.00 impl=0.18
item: cross-cutting-refactor   design=0.05 impl=0.12
item: smaller-go-module        design=0.00 impl=0.10
item: atlas-docs               design=0.04 impl=0.06
item: milestone-review         design=0.00 impl=0.16
design-buffer: 0.15
total: 5.23
```

Derivation notes, in plan-task order. M1 is rows 2–10, M2 rows 11–17.

- **`issue-spec` design is NOT discounted — it IS the design**, and it is priced
  inside the table's undiscounted 0.5–1.5 band accordingly. 0.60 covers a full
  re-spec after operator pushback, the dlsym/87-dictionary measurement campaign,
  the plan doc, an independent re-measurement of nine claims on 2026-08-28, and
  three plan-quality rounds settling five blocking findings. Two drafts of this
  block put it lower — 0.20, then 0.60 — while claiming it was undiscounted; the
  first was 40% of the band's FLOOR, a harder discount than the ×0.2 rows take,
  and the second was still near it. This is the one row with real headroom rather
  than a band ceiling, and it is the row v3.1 deliberately leaves unscaled because
  design does not compress the way implementation does. Mid-band is the honest
  reading of what it actually bought. Every other design figure below IS discounted ×0.2,
  because that work is what this row bought.

- **Row → task, since seventeen rows over ten tasks is not self-evident.** M1:
  row 2 = T1, row 3 = T2, row 4 = T3, **rows 5 AND 6 = T4** (the `/lang` state
  machine and the `ReadLang`/`WriteLang` + `-lang` half are separate primitives
  in one task, and T4 is M1's heaviest), row 7 = T5, row 8 = T6, rows 9–10 = T7.
  M2: row 11 = T8, **rows 12–14 = T9**, rows 15–17 = T10.
- **Two `cross-cutting-refactor`s, and the plan enumerated both before pricing
  them.** `NewYAML(dir, lang, warn)` is 31 call sites across 8 files (T2); the
  per-language fixture corpus is 4 direct `loadFakeDictionary` sites plus ~12
  `testDict` callers (T9). `#24`'s `Apply` signature change was the same shape at
  nine sites and was priced this way — counting either as a one-liner is the
  mistake the ledger has already recorded.
- **`tui-screen` for `/lang`, not `smaller-go-module`.** The command row itself
  mirrors `/sound` and would be trivial; what it costs is the `setLang` closure
  across BOTH loops plus the highlight set the raw editor captures into a local
  before the loop (`replraw.go:79`) — state-machine work, which is what the
  primitive names. The plan's D1 is why this is 0.24 and not 0.12.
- **`api-integration` carries M2's novelty, and carries it ALONE.** Design 0.40 is
  the table's 2.0 discounted ×0.2: the nine symbols are measured, `CFSetGetValues`
  is named, the fallback is specified. Impl is picked at the TOP of the range
  (1.5 → 0.6 at v3.1's 40%) rather than mid, because cgo against undocumented
  symbols where the wrong container call is an uncaught ObjC exception is
  novel-but-bounded work. That is Step 5's ×1.5 applied where it belongs — to the
  one primitive that is novel — instead of as a blended `familiarity` over
  fifteen rows that are ordinary Go in a package this repo knows well. Hence
  `familiarity: 1.0`, honestly.
- **`real-api-discovery` (0.18) is `capture.py`, not the probe.** The private
  surface was measured before planning. What remains unbought is resolving a
  `DCSDictionaryRef` through a CFSet from ctypes and capturing through it.
- **Two `milestone-review`s, one per boundary, and M1's is the larger** (0.20 vs
  0.16): its diff spans ~15 files including a signature change, a migration of the
  one irreplaceable artifact, and three repo guards. `#3` needed four rounds on a
  single function in this same store.
- **Library-availability check (v2.1 Step 2.5):** applied and NOT triggering.
  DictionaryServices' private surface has no Go shim — `dlsym` from cgo is the
  only route — and the CDN work is `net/http` plus the fake already in the tree.
  No design halving applies.
- **Design buffer 0.15, not 0.30**, per v2.1 Step 6: the plan doc resolves the
  decisions, and D1–D5 are exactly that resolution written down.
- **Where a miss is most likely, recorded now rather than rediscovered at close.**
  Row 12's 0.60 is the band CEILING, and the band was fitted against documented
  HTTP APIs; this is an undocumented private ObjC surface where the wrong
  container call is an uncaught exception in a cgo frame. Row 3 is at its ceiling
  too, with 31 call sites across 8 files against the `#24` precedent's nine — 3.4×
  the precedent at the same maximum the primitive can express. Both
  `milestone-review` rows are also at their ceiling (0.20 is the max v3.1 can express), while `#25`
  ran BR-1..BR-12 over five fix commits and `#17` BR-17..BR-25 over four. A 2×
  overrun on those three rows is a model-expressiveness limit, not an estimating
  error — read it that way at close.
- **`#18 M1` coordination is deliberately unpriced.** The model has no primitive
  for cross-issue coordination. If `#18 M1` lands concurrently on `audiourl.go` or
  `speak`, T5 absorbs a merge that no row budgeted.

Σdesign 2.08 × 1.15 = 2.392; Σimpl 2.84 × 1.0 = 2.84; total **5.23**.

## Plan

Two review boundaries, so two `Mx` rows — `close.go`'s milestone-verdict guard
reads THIS section, and with a single un-tagged row the "was M1 reviewed" check
finds zero milestones and passes vacuously (PQ-8). Detail lives in
`workshop/plans/000023-deck-language-plan.md`; these are the boundaries, not a
second copy of the tasks.

- [x] Design via `sdlc start-plan` before implementing. Coordinate with `#18 M1`,
      which owns the audio half of the same `-lang` flag.
- [x] M1 — the mode exists and the deck follows it: `Lang` + the runtime-FILE
      guard, `words/<lang>/`, the flat-deck migration, `-lang` / `lang.txt` /
      `/lang`, the recording following the mode, `--play` and `--forget`
      inheriting it, docs. Ships a working single-language `define`.
- [x] M2 — the dictionary follows it too: `chooseDictionary` over metadata plus a
      curated default, the `dlsym` seam with a fallback to today's NULL
      behaviour, the per-language fixture corpus, live conformance.

## Log

### 2026-08-28
- 2026-08-28: closed — M2 makes the dictionary follow the mode; all 13 Done-when rows ticked. Close-review rounds 5, 6 and 7 addressed as rules.; review verdict: FIX-THEN-SHIP

HEADLINE, re-run unsandboxed after every refactor: `define mesa` = flat-topped hill; `define -lang es mesa` = "nombre femenino ... Mueble formado por un tablero horizontal"; `define -lang es sycophantic` = no entry; `define iPhone` answers via the second curated book; `define madrugar` = no entry, where the NULL search returns the Spanish entry.

ROUND 7:

BR-20/BR-29 — my sweep was three sites short, and the three were newDeck comments left by the very commit that added the guard against stale symbol names. Ninth in that family, and it names what was missing: a rename cannot be detected automatically, because only the person doing it knows the old name. retiredSymbolNames is that one human-written row; TestNoArtifactNamesARetiredSymbol makes the rest mechanical across non-test Go, README, atlas/ and active plans. Mutation-checked. It immediately found a live D1 bullet naming deckDeps to explain what the plan first called it — a backreference that belongs in ## Revisions, where it already was.

BR-28 — capture.sh hand-restated the curated identifiers, the same divergence that already bit once (English captured through NULL while production selected curated ids, agreeing only because this host active set matched). A shell script cannot import a Go map, so TestCaptureScriptUsesTheCuratedDictionaries COMPARES them — the same move as the cgo-preamble symbol guard. Mutation-checked both directions.

MINOR, A REAL DEFECT: capture.py returned a DCSDictionaryRef borrowed from the copied CFSet from inside a try whose finally released that set, then passed the ref to DCSCopyTextDefinition — use-after-release that worked only because CoreServices keeps dictionaries alive. It retains now and releases after the lookup; re-verified capturing mesa, bonito and madrugar unsandboxed.

EARLIER ROUNDS (disposed): C1, BR-2, BR-13 — the language-derived set went doc-comment -> struct -> EMBEDDED struct, because each weaker form went stale; BR-22b — the headline d.newDict wiring had zero tests and its first pin replicated the production line instead of calling run(); BR-23 — pure logic behind //go:build darwin broke GOOS=linux go vet and let a fix ship inoperative; BR-24 — dcsPrivateSymbols was two producers.

TESTS: go build ./... && go vet ./... && GOOS=linux go vet ./cmd/define/ && go test ./... all green; gofmt -l ./cmd/ empty. Conformance passes unsandboxed in both env states, skips sandboxed.

ACTUAL: omitted so close measures and adopts. The issue Log records this row should be UNTRUSTED for calibration — the window base is a pre-claim 2026-08-27 commit ~20h before started:, with mention-fallback attribution across eleven issues.
- 2026-08-28: closed M1 — M1 ships a working single-language define. Three boundary-review rounds, all addressed as rules rather than instances.; review verdict: FIX-THEN-SHIP

**Three ledger rows read "open" and are NOT outstanding work.** They were fixed
in the M1 close commit (`514c4e6`), which `#174` requires instead of a fifth
review, so no gate re-ran to dispose them:

- **PQ-8** (plan gate) — the `## Plan` above now carries real `Mx` rows.
- **BR-11** — the artifact-name rule is enforced over prose, not just `*.go`
  (`TestProseDoesNotSpellStaleRuntimeArtifactNames`), and the project file's
  false "single learner model" claim is gone.
- **BR-12** — the interrupted-write pin moved into package `store` and derives
  its paths; `store.LangFileName()` is exported so `main`'s tests stop restating.

The close review's window starts at the M1 boundary and will not see these fixes
in its diff. Recorded so it disposes them rather than re-raising them.


ROUND 3 BLOCKERS FIXED AS RULES (the gate said "4 repeat families — not converging: fix rules, not instances", and the reviewer said explicitly not to fix the instances):

BR-7 rule — a runtime artifact name has exactly ONE producing function; guards, migrations and tests derive from it. Four producers now (UserModelName, langFileName, newTempFile, userModelLegacy), and RuntimeFiles BUILDS its two pattern entries from them. Mutation-checked BOTH ways: changing the atomic-write prefix, and changing the learner-model scheme, each now fail TestGitignoreCoversRuntimeFiles BY NAME. Before this, the reviewer proved the prefix change left the entire suite green while the shadow beside the two root-level runtime files matched no .gitignore pattern.

BR-6 rule — no output line, comment or doc may spell a runtime artifact filename or a symbol the code owns. Now MECHANICAL: TestRuntimeArtifactNamesAreSpelledOnceInSource permits a learner-model filename in non-test Go only in the two consts that build every such name, and it is mutation-checked. Mechanising found FOUR sites the manual enumeration missed. The user-facing one is why it matters: --reflect printed "wrote user-model.md" while writing the per-language file, and README tells the learner to hand-edit that file ## Corrections, so following the tool own output put corrections where UserModel() never looks. store.UserModelName(lang) is exported so a consumer names the file without restating the scheme. The symbol half swept too — atlas, a main.go comment and the plan named MigrateFlatDeck, which the tree does not export; the plan records the rule, since this was the second absent entity it named.

ROUND 3 MINORS FIXED: detached PATTERNS comment merged into its doc; MigrateToLanguages doc says it moves two named artifacts; wordsDir comment corrected (it still claimed userModelFile was unscoped); README states -locale is English-only three lines from where it demonstrates it; d.history identity across a switch asserted rather than only argued; project actual corrected from 1.03h, which predated the review rounds.

EARLIER ROUNDS (disposed): C1 — applyLang owns the full enumeration with the rule that generates it, applyVoice is one derivation with two callers. BR-2 — the learner model is per-language, since it is DERIVED from the language-scoped deck; the events/ argument does not transfer to a summary of a deck. I1/I2/I3/I4 all fixed.

VERIFIED BEYOND TESTS: git check-ignore -v matches user-model.md, user-model.en.md, user-model.es.md, lang.txt and .tmp-abc, and still does NOT match the golden fixture. Running the binary in a directory holding a pre-language user-model.md prints "moved user-model.md to user-model.en.md" and leaves exactly that file. go test -tags conformance CDNStillServesSpanish passes against the LIVE CDN.

FULL SUITE: go build ./... && go vet ./... && go test ./... green; gofmt -l ./cmd/ empty.

ACTUAL: 2.15h is the M1 increment, not the issue total — session commit timestamps 10:55 to 13:05, continuous. sdlc actual measures from a PRE-CLAIM 2026-08-27 commit with mention-fallback attribution across eleven issues; the issue Log records that this row should be marked untrusted for calibration.

M2 NOT in scope: mesa in a Spanish session is filed as Spanish but still defined as the flat-topped hill. The review also flagged that usage/ becomes a scoping candidate the moment the dictionary follows the mode — carried into M2 planning rather than discovered at M2 boundary.

Plan cleared plan-quality on round 2 (verdict CLEAN); five blocking findings
PQ-1..PQ-5 answered as D1–D5 in the plan, each verified against the tree before
being answered rather than accepted on its face. One turned out to be worse than
stated: `git check-ignore -v user-model.md` matches nothing today, so the
unignored-runtime-file defect is live rather than latent. Ledger:
`workshop/plans/000023-deck-language-plan-gate.md`.

Estimate derived after the plan cleared, per `#187`: **5.23h** (v3.1, seventeen
rows). The estimate-quality judge returned INFO twice and its substantive
findings were applied both times — `issue-spec` design sat below the table's
undiscounted floor while the note claimed it was undiscounted (0.20 → 0.60 →
1.00, mid-band); the two `atlas-docs` rows used the undiscounted floor while
claiming the ×0.2; the row→task key was added; and rows at their band ceiling
(3, 12, 16, 17) are now named as model-expressiveness limits rather than
estimates. PQ-8 — the issue's `## Plan` carrying no `Mx` rows, which would let the
close-time milestone-verdict guard pass vacuously — is fixed above.

**Carry to close: this issue's actual is NOT a clean calibration row.** `sdlc
actual --issue 23` already reported 4.92h before any implementation, because the
window base is a pre-claim `#23` commit from 2026-08-27 14:15 — roughly 20h of
wall-clock before `started:` — and attribution spreads across eleven issues with
mention-fallback on nearly every span (`#15` alone draws a 284.3m dominant
segment inside the window). Mark the ledger row untrusted rather than feeding it
to the v3.1 scale fit; this is `baseline-v3.1.md`'s open question #1.

### 2026-08-27

Filed from the operator's request after the first real `--play` session. The
detection measurement above was taken before planning, in the same spirit as
`#9`'s feed measurement and `#18`'s audio measurement: the API limit it found is
what turns "infer it" from a design into "infer it, with an override that is
required rather than convenient".

## Revisions

### 2026-08-27 — a declared mode, not inference

**Reason.** Operator, on being shown the measurement above:

> ok, I guess there are same word different meaning in en/es. let `define`
> operate in a single language. add a `/lang en` in the define TUI to switch
> language.

**Delta.** Inference is dropped entirely. `define` has a language MODE, persisted
in the vocab directory, switched with `/lang`.

**Why this is better rather than merely different.** The measurement found a hard
API limit: for `mesa`, `bonito`, `pie`, `once`, `real`, `red` the dictionary
returns the English entry and never reveals a Spanish one exists. Infer-plus-
override would have handled those by being wrong and waiting for the learner to
notice. A declared mode removes the question — in Spanish mode `mesa` is Spanish
because the learner said so. The design that needs no heuristic beats the design
whose heuristic provably cannot see half its cases.

It also matches how the tool is actually used: one deck directory, one sitting,
one language. `/sound` is the precedent for a `/`-command that changes how a
session behaves — but language differs from it in one way that matters, and the
Spec now says so: `/sound` is explicitly session-scoped, while language must
persist, because a one-shot lookup has no session to inherit from.
