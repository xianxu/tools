# Boundary Review — tools#10 (milestone M1)

| field | value |
|-------|-------|
| issue | 10 — authored practice items: level-tagged words, and stems the model writes offline |
| repo | tools |
| issue file | workshop/issues/000010-vocab-harvest.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 0b8d9930762168cf52f77c5d0864599f678d3b5d..b5cb91d996272c654d158a95e01944c6c1a90f68 |
| command | sdlc milestone-close --issue 10 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-04T12:35:21-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what its four Plan rows claim, and delivers it with unusually honest pins: `--harvest` is a mode, the cache check precedes the network, the "second run makes zero calls" claim is asserted on the request *count* rather than on files existing, the sitting test panics the seam instead of nilling it, and the outage test asserts what survived is *whole* rather than merely present. The PQ-1 loop was genuinely closed rather than asserted — `--reflect` now validates through `ParseBand`, `renderUserModel` emits `level:`, `parseLearnerBand` reads it back, and all three have tests. `go build`, `go vet`, `gofmt`, and `go test ./...` are green as committed. Nothing here is blocking. What stands between this and SHIP is three cheap Important fixes: the stability measure — the one number M1 claims as *measured* — counts `"c1"` and `"C1"` as disagreement, so it can fail its own 0.8 floor on a perfectly stable model (verified: `agreement(["C1","c1","C1"]) = 0.67`); `Mem` and `YAML` disagree about a damaged record in exactly the way `storetest` exists to prevent (verified: `Mem` returns an off-scale band as harvested, `YAML` returns unharvested); and the band prompt hardcodes "English vocabulary" while the diff's headline storage decision is that facts are per-language *because Spanish decks exist*.

## 1. Strengths

- **`cmd/define/store/yaml.go:562-601`** — the read-side re-parse is the right placement, and `yaml_test.go`'s four-row `TestUnparseableWordFactsReadAsUnharvested` plus the band/domain asymmetry test (`a bad band voids the record, an unknown domain degrades`) pin a genuinely subtle contract. ARCH-SECURE at its best: persisted input is not trusted because this program wrote it.
- **`cmd/define/harvest_test.go:51`** — `TestHarvestAsksOncePerWordAndNeverAgain` asserts `len(fake.Requests())` is unchanged, not that files exist. That is the difference between pinning the claim and pinning a side effect.
- **`cmd/define/harvest.go:96-107`** — stopping on the first model error rather than continuing, with the store left whole and the count of what survived printed. `TestHarvestOutageKeepsWhatWasAlreadyBought` asserts `banded != 0 && banded != len(deck)`, so the test cannot pass vacuously in either direction.
- **The domain single-source actually swept.** `grep "Nautical"` over the tree finds the table only in `store/vocab.go`; `glosslabel.go:51` derives, and `TestBandPromptCarriesTheClosedDomainSet` means a new label cannot reach the store without reaching the prompt. ARCH-DRY: pass.
- **`cmd/define/reflect.go:262-283`** — the third switch arm drops rather than coerces, and `reflect_test.go` covers both halves (off-scale drops with a message naming `A1-C2`; `"c1 "` canonicalises to `C1`). The fix has a test that fails without it.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `agreement` keys on the raw band, so case/space variants score as disagreement (`cmd/define/harvest_band.go:113`).** `ParseBand` documents case and surrounding space as "transcription noise rather than a different answer" and `agreement` calls it — but then does `counts[b]++` on the *unparsed* value. Verified: `agreement(["C1","C1","C1"]) = 1.00`, `agreement(["C1","c1","C1"]) = 0.67`, `agreement(["C1","C1 "]) = 0.50`. A model that is perfectly stable but inconsistent in casing scores below the 0.8 conformance floor, whose message prescribes "build the hand-labelled sample the issue defers" — an expensive remedy for a formatting difference. This is the one number M1 claims as measured (ARCH-DRY: two spellings of the same fact, one canonical and one not). Fix: `if b, ok := store.ParseBand(string(x)); ok { counts[b]++ }`, and add a row `{"casing is not disagreement", []store.Band{"C1","c1","C1"}, 1}` to `TestAgreement`.

**I2 — `Mem` and `YAML` disagree on a damaged record; `storetest` has no row (`cmd/define/store/mem.go:147`, `store.go:45`).** `Store.WordFacts`' own doc says "A stored record too damaged to parse also reads as unharvested, on purpose." Verified against both: `SetWordFacts("w", {Band:"B2+", Domain:"Nonsense", At:t})` then read back gives `Mem → band="B2+" harvested=true`, `YAML → band="" harvested=false`. Neither `SetWordFacts` validates at the write, so the interface's guarantee is a YAML implementation detail. The plan put this row *in the suite* ("so BOTH implementations are held to them at once", Task 1 Step 1) and it landed in `yaml_test.go` only. Fix: canonicalise through `ParseBand`/`ParseDomain` at the write in both implementations (or refuse), and move the row into `storetest/suite.go` so the fake cannot drift (ARCH-MOCK).

**I3 — the band prompt hardcodes English while the facts it produces are stored per-language (`cmd/define/harvest_band.go:31`).** `bandSystem` opens `"You place English vocabulary on the CEFR scale."` and `renderBandPrompt` takes no language. Meanwhile PQ-2, `yaml.go:163`, `atlas/define.md` and `README.md:415` all justify `facts/<lang>/` with *`red` is a different word in English and Spanish*, and a Spanish working directory is shipped (`lang.txt`, `dictselect.go`, `user-model.es.md`). Running `define --harvest` in one today writes `facts/es/*.yaml` from a prompt asserting English — cached forever, and the plan's own words for undoing a bad forever-cache are "a migration". `d.lang` is already in scope at the call site (`main.go` passes it to `applyVoice`). Fix: thread the language into `bandTask`/`renderBandPrompt` (one golden per shape, as the known-domain split already does), or refuse `--harvest` outside `DefaultLang` until M2 and say so in the README. ARCH-PURPOSE: the language dimension is half-delivered — the storage derives, the prompt does not.

**I4 — `sanitiseFacts`/`sanitiseItem` do not exist, but the plan step naming them is ticked (`workshop/plans/000010-vocab-harvest-plan.md`, Task 1 Step 4).** `grep sanitise cmd/define/` finds only `sanitiseModel`/`sanitiseMeta`. For `WordFacts` the closed parses arguably cover it — but the plan explicitly pre-rejected that argument ("`Band` and `Domain` are additionally parse-refusing, which is a narrower guarantee than neutralisation and **not a substitute for it**"). `SetItems` ships the write path for `Stem`/`Answer`/`Distractors` with no neutralisation, and the whole point of putting it *at the write* was that M2's authoring cannot then forget. Fix: either land `sanitiseItem` at `SetItems` now, or add a `## Revisions` entry recording that the closed parse covers `WordFacts` and `sanitiseItem` lands with the authoring in M2 — and untick the step's claim accordingly.

## 4. Minor findings

- **`harvest_band.go:117-127`** — the `keys` slice and `sort.Slice` cannot affect the return value (`best` is a max over `counts`, order-independent). The comment describes a tie-break that is not observable. Delete ~10 lines and the `sort` import.
- **`harvest.go:34`** — `agreementRounds = 5` is declared and never used; `-agreement` is `flag.Int`, so the documented `--agreement[=N]` default of N=5 (issue Done-when 3, plan Task 2 Step 3) is unreachable — a bare `-agreement` is a flag error. Wire it as the default or drop the constant and correct the issue/plan text. README already documents the required-N form correctly.
- **`harvest_band_test.go:88-96`** — hand-rolled `contains` duplicating `strings.Contains`, which `harvest_test.go` already imports in the same package (ARCH-DRY).
- **`glosslabel.go:56`** — the newly *computed* longest-first ordering has no pin. Verified by mutation: flipping `>` to `<` in `sortedByLengthDesc` leaves the whole `cmd/define` suite green (109s run). The atlas advertises this as the improvement over a hand-maintained invariant; a two-line test asserting descending length would make it one.
- **`harvest.go:186-188`** — `wordSense(d, w.Text)` is loop-invariant but sits inside the per-round loop, so the measurement mode does N dictionary lookups + `ParseEntry` per word instead of one (ARCH-CONSTRAINTS: repeated work that should be hoisted). Cheap (CGO, local), but free to fix.
- **`main.go:584-592`** — `-limit` is accepted alongside `-agreement` and silently ignored, and `-agreement N` is unbounded above (K=20 × N calls). Also `define --play --harvest` silently runs only `--play` (pre-existing shape for `--play`/`--reflect`, now a third mode on it) — the file's own rule three lines up is that silently honouring one of two commands is how `-raw` came to mean two things.
- **`workshop/projects/define-learn.md:449`** — "the milestone had no remediation round at all" and `actual: 2.16h` were committed before this gate ran. If any finding above is remediated, that calibration prose and the actual are already stale.

## 5. Test coverage notes

Coverage is strong where it counts: the store contract is exercised through `storetest` against both implementations, the pure vocabulary has an explicit refuse-list including every plausible model answer (`B2+`, `intermediate`, `C1-C2`, `A0`), the prompt has two goldens for its two shapes, and the live conformance file asserts *both* the floor and the answer *shape* — the second is the one a fake-only suite structurally cannot see. Gaps that map to shipped findings: no casing row in `TestAgreement` (I1), no damaged-record row in `storetest` (I2), no non-English `--harvest` test (I3), and `sortedByLengthDesc` survives an inverted comparator. `TestEveryUntrustedFieldIsNeutralised`'s "level band" row does now cover the new frontmatter line, which is the right instinct — that suite exists because a fix once shipped with no failing site.

## 6. Architectural notes for upcoming work

- **ARCH-PURE: pass.** `Band`, `Domain`, `agreement`, `parseLearnerBand`, `renderBandPrompt` are pure and tested with no IO or mocks; `runHarvest` is the thin shell. M2's `pickDistractors`/`topicSpread` should stay on that side of the line — they are arithmetic over already-banded candidates and want no store handle.
- **ARCH-MOCK: pass with I2.** Wire-level `llmtest.Fake` plus a live conformance row per behavior is the right pattern; keep the entailment and veto judges to it. The lesson from I2 is that a *new contract sentence on the interface* is the trigger for a `storetest` row, not a new method.
- **ARCH-SECURE for M2:** `Item.Stem`/`Answer`/`Distractors` are the first free-text model fields this store will persist and later render onto a board. Land the neutralisation at `SetItems` (I4) before the authoring task, not after — the ordering is the whole argument `usermodel.go:222` makes.
- **The `quokka`-at-C2 observation in the Log is the most valuable thing here.** It is the stability/correctness gap made concrete, and M2's `pickDistractors` is where it will surface. Consider having selection log the band it selected at, so a mispitched batch is diagnosable without re-deriving the cache.

## 7. Plan revision recommendations

- **`sanitiseFacts`/`sanitiseItem` (I4):** a `## Revisions` entry recording that they did not land in M1 and why — either "the closed parse covers `WordFacts`; `sanitiseItem` moves to M2 Task 4 with the authoring", or land them. Task 1 Step 4 currently reads as delivered.
- **The `--agreement` flag shape:** the plan and issue Done-when 3 both say `--agreement[=N]` with `N=5` default; the shipped flag requires an explicit N. Correct the text (README is already right).
- **Language scope for `bandTask` (I3):** the plan's PQ-2 argues per-language storage from a live Spanish path but never says what the *prompt* does about language. Record the decision either way.

```findings
findings:
  - id: new
    severity: Important
    family: parse-result-not-canonicalised
    title: |
      agreement counts raw band strings, so "c1" and "C1" score as disagreement
    detail: |
      cmd/define/harvest_band.go:113 does counts[b]++ on the unparsed value after
      filtering with ParseBand, which documents case and space as transcription
      noise. Verified: agreement(["C1","c1","C1"]) = 0.67 and agreement(["C1","C1 "])
      = 0.50, so a perfectly stable model can fall below the 0.8 conformance floor
      whose prescribed remedy is the deferred hand-labelled sample. Key on the
      parsed band and add a casing row to TestAgreement.
  - id: new
    severity: Important
    family: store-contract-unheld-by-suite
    title: |
      Mem and YAML disagree about a damaged WordFacts record, and storetest has no row
    detail: |
      store.go:45 states "a stored record too damaged to parse also reads as
      unharvested". Verified: SetWordFacts with Band "B2+" reads back harvested=true
      from Mem and harvested=false from YAML. Neither SetWordFacts validates at the
      write, so the interface guarantee is a YAML detail. The plan placed this row in
      storetest/suite.go "so BOTH implementations are held to them at once"; it landed
      in yaml_test.go only.
  - id: new
    severity: Important
    family: language-scope-not-threaded
    title: |
      bandTask hardcodes "English vocabulary" while facts are stored per-language
    detail: |
      harvest_band.go:31 asserts English and renderBandPrompt takes no language, but
      facts/<lang>/ exists precisely because Spanish decks are live (yaml.go:163,
      atlas, README). define --harvest in a Spanish directory writes facts/es/*.yaml
      from an English-asserting prompt, cached forever. d.lang is already in scope at
      the call site; thread it, or refuse --harvest outside DefaultLang and say so.
  - id: new
    severity: Important
    family: plan-element-ticked-unbuilt
    title: |
      sanitiseFacts/sanitiseItem do not exist although the plan step naming them is ticked
    detail: |
      grep sanitise cmd/define/ finds only sanitiseModel/sanitiseMeta. The plan
      explicitly pre-rejected "the parse covers it" ("a narrower guarantee ... not a
      substitute"), and SetItems ships the write path for Stem/Answer/Distractors with
      no neutralisation. Land sanitiseItem at the write, or add a ## Revisions entry
      recording the deferral.
  - id: new
    severity: Minor
    family: inert-mechanism
    title: |
      the sort in agreement cannot affect the result, and agreementRounds is unused
    detail: |
      harvest_band.go:117-127 builds and sorts a keys slice that never influences
      `best` (a max over counts); the tie-break comment describes unobservable
      behaviour. Separately agreementRounds = 5 (harvest.go:34) is declared and never
      referenced, so the documented "-agreement default N=5" is unreachable through
      flag.Int.
  - id: new
    severity: Minor
    family: property-without-a-pin
    title: |
      the computed longest-first label ordering has no test
    detail: |
      Verified by mutation: inverting the comparator in sortedByLengthDesc
      (glosslabel.go:63) leaves the full cmd/define suite green. The atlas advertises
      the computed ordering as the improvement over a hand-maintained invariant; a
      two-line descending-length assertion would make it one.
  - id: new
    severity: Minor
    family: stdlib-reimplemented
    title: |
      hand-rolled contains in harvest_band_test.go duplicates strings.Contains
    detail: |
      harvest_band_test.go:88-96 reimplements substring search in a package whose
      harvest_test.go already imports strings for the same purpose (ARCH-DRY).
  - id: new
    severity: Minor
    family: loop-invariant-work
    title: |
      runHarvestAgreement re-runs wordSense inside the per-round loop
    detail: |
      harvest.go:186-188 recomputes the dictionary gloss and domain N times per word
      although both are invariant across rounds. Cheap (local CGO lookup) but free to
      hoist out of the loop.
  - id: new
    severity: Minor
    family: flag-silently-ignored
    title: |
      -limit is ignored in -agreement mode, N is unbounded, and --play --harvest drops a mode
    detail: |
      main.go accepts -limit with -agreement and ignores it; -agreement N has no upper
      bound (K=20 x N calls); and dispatch order means `define --play --harvest` runs
      only --play. The file's own comment three lines above says silently honouring one
      of two commands is how -raw came to mean two things.
  - id: new
    severity: Minor
    family: doc-predeclares-outcome
    title: |
      the project's M1 calibration prose was written before this gate ran
    detail: |
      workshop/projects/define-learn.md records actual 2.16h and "the milestone had no
      remediation round at all" in the commit that precedes the boundary review. Any
      remediation of the findings above makes both stale.
```
