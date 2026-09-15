# Bilingual Spanish Definitions Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use superpowers-executing-plans for warm-context integration and bounded subagents for the native adapter and pure record selection. Steps use checkbox syntax.

**Goal:** Spanish word lookups show Spanish definitions followed by English explanations from installed macOS dictionaries.

**Architecture:** Preserve the primary Dictionary.Lookup contract for raw output, word identity, pronunciation, and practice construction. Add an optional supplemental-definition capability to the Spanish dictionary composition. Compose independently rendered, source-labeled sections at full-definition display sites; select Oxford records by verified source direction rather than its misleading dictionary-level metadata.

**Tech Stack:** Go, macOS DictionaryServices/CoreFoundation via runtime-resolved cgo, existing terminal renderer and stateful dictionary fakes. No LLM or new network dependency.

## Core concepts

| Name | Lives in | Status | Kind |
|------|----------|--------|------|
| `bilingualRecord` / `selectSpanishRecords` | `cmd/define/bilingual.go` | new | PURE |
| `definitionSection` / `definitionSet` / `renderDefinitions` | `cmd/define/definitions.go` | new | PURE |
| `definitionsFor` | `cmd/define/definitions.go` | new | INTEGRATION |
| `parseBilingualArgs` | `cmd/define/bilingual_cmd.go` | new | PURE |
| `runBilingual` / `sessionSetBilingual` | `cmd/define/bilingual_cmd.go` | new | INTEGRATION |
| `ReadBilingual` / `WriteBilingual` | `cmd/define/store/bilingual.go` | new | INTEGRATION |
| `systemDictionary` | `cmd/define/dict_darwin.go` | modified | INTEGRATION |
| `spanishDictionarySources` | `cmd/define/bilingual_sources.go` | new | PURE |
| `spanishDictionaryFromInstalled` | `cmd/define/bilingual_sources.go` | new | INTEGRATION |
| `clozeAsk` | `cmd/define/cloze.go` | modified | INTEGRATION |
| `todaysQuestions` | `cmd/define/play_loop.go` | modified | INTEGRATION |
| `lookupAndRender` / `writeWords` | `cmd/define/main.go` | modified | INTEGRATION |
| `posWords` | `cmd/define/parse.go` | modified | PURE |


`bilingualRecord` contains copied XHTML metadata and flat source text from one native record. Pure selection reads the root `d:entry` identifier and title using a bounded standard-library XML token walk. Oxford's measured Spanish→English IDs contain `s_b-es-en`; English→Spanish IDs contain `e_b-en-es`. Accept only the verified Spanish family and a usable title/text, reject unknown roots/directions, deduplicate by ID, and preserve distinct Spanish homographs. Prefer exact case-insensitive title matches, then diacritic-equivalent titles, then the primary Spanish entry's canonical headword for inflections. If no verified title matches, report a direction/entry selection miss rather than guess among unrelated search results. Tests pin this decision order using independently captured records.

`definitionSet` owns ordered Spanish and English sections, each with source, zero or more original entry texts, and an optional availability/error result. It does not concatenate two dictionaries into one ParseEntry input or align their senses. Its primary source text remains separately accessible for pronunciation and session context. Each successful record is parsed/rendered individually. A single successful section is a successful displayed lookup; no successful sections uses the existing failure path. Errors distinguish dictionary unavailable, private API unavailable, ordinary entry miss, malformed/unsupported records, and native lookup failure.

`renderDefinitions` returns one text plus aligned regions. Spanish appears first, English second, under clear labels identifying Larousse and Oxford. English and Italian monolingual outputs stay byte-compatible. Section labels consume visible rows; region offsets derive from the rendered prefix, never an independently maintained layout formula. Rendering preserves the ordered alphanumeric content of each source record. Spanish vocabulary highlighting and word actions apply to Spanish prose; English prose must not silently acquire Spanish deck-word actions. Both sections remain mouse-selectable/copyable; English section headwords still refer to the Spanish lookup for pronunciation.

## Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Spanish dictionary composition | cmd/define/definitions.go | new | Primary Dictionary plus optional supplemental records |
| Oxford record adapter | cmd/define/bilingual_darwin.go, bilingual_stub.go | new | DictionaryServices record search/copy APIs |
| systemDictionary | cmd/define/dict_darwin.go | modified | Installed dictionary construction |
| lookupAndRender | cmd/define/main.go | modified | One-shot and editor display/capture |
| todaysQuestions full-definition renderer | cmd/define/play_loop.go, cmd/define/cloze.go | modified | Practice Choice/Cloze full post-answer reveals and region maps |
| stateful record fake | cmd/define/bilingual_test.go | new | Installed set, per-word records, errors and call observations |

The optional capability is owned by the same dictionary object rebuilt through `newDict`, so `/lang` and startup cannot retain stale supplements. `Dictionary.Lookup` continues to query only the primary language. Spanish composition must not fall back to the uncontrolled all-active-dictionaries search when Larousse is missing: that could label English output as Spanish. Other languages retain their existing fallback behavior. `/lang` names both installed sources or the missing source.

The supplemental interface accepts the already-fetched primary raw text (or its failure) so ordinary lookup and practice construction do not fetch the primary twice. Existing Dictionary fakes lacking the capability remain monolingual. A stateful composite fake models presence/absence, errors, aliases and installation replacement; it records calls to verify raw/training paths never request supplemental records.

## Native evidence and constraints

A read-only probe independently used the existing Python capture helper to select `com.apple.dictionary.OxfordSpanish` and confirmed:

- `DCSCopyTextDefinition` alone returns English red→Spanish rojo for `red`.
- `DCSCopyRecordsForSearchString(dictionary, word, 0, 20)` exposes both directions.
- `DCSRecordCopyData(record, 0)` supplies XHTML with directional entry IDs; style 3 supplies corresponding flat text. Anchor/title getters were NULL; do not depend on them.
- `madrugar`, `mesa`, `árbol`/`arbol`, `jalapeño`/`jalapeno` have usable Spanish→English records.
- `como`, `solo`, `pie`, and `son` return unrelated inflections, repeated record IDs, and/or English records. Direction alone is insufficient.

The adapter resolves the two additional symbols at runtime independently from primary dictionary lookup. Keep every copied CF object alive until its children have been copied to Go values, then release all ownership on success/error paths. No borrowed pointer escapes. Bound results at 64 records and 1 MiB per copied representation; hitting a search cap is an explicit incomplete-result failure, not a false word miss. Query/representation limits are per lookup; no persistent cache or workers. A changed private contract leaves the Spanish definition available with an explicit English-section diagnostic.

Capture committed fixtures from the independent probe, not from newly written Go production code. Preserve dictionary ID, query, record ID, capture method and expected language in a fixture manifest. Keep a small discriminating corpus, including ambiguity, accents, inflection and malformed/unknown direction. Live conformance compares native record selection with these contract properties rather than requiring all dictionary prose to remain byte-identical across OS updates.

## Tasks

### 1. Record selection and native adapter

Files: bilingual.go, bilingual_darwin.go, bilingual_stub.go, bilingual_test.go, bilingual_conformance_test.go; testdata/bilingual/.

- [x] Capture and commit independent fixtures for red, mesa, madrugar, arbol, jalapeno, como, solo, pie, son, madrugaste and mesas; confirm exact API signatures and ownership from the completed probe.
- [x] Write failing pure tests for wrong direction, malformed/unknown root, duplicates, exact versus accent matches, canonical inflection matching, stable homograph ordering, limits and no usable match.
- [x] Implement the pure selector, stateful record fake and bounded runtime-resolved native adapter. Native failure stays separate from entry absence. Add a portable stub.
- [x] Run `go test ./cmd/define -run 'TestBilingual' -count=1` and strict live `CONFORMANCE_STRICT=1 go test -tags conformance ./cmd/define -run '^TestBilingualNative' -count=1`; the latter must exercise Spanish red→net/network and reject English red→rojo.

### 2. Ordered definition sections and language composition

Files: definitions.go, definitions_test.go, dictselect.go, dictselect_test.go, dict_darwin.go, dict_stub.go, lang_scope_test.go.

- [x] Write failing tests for Spanish-first ordering, both sources installed, either source missing, both missing, per-word misses, native errors, and malformed bilingual results. Do not suppress a good section because the other fails.
- [x] Implement the optional dictionary capability and constructor wiring through the existing `newDict` seam; use known dictionary IDs, never localized name matching. Extend `/lang` source reporting and cover es→en→es rebuilds.
- [x] Implement section rendering with source labels, independently parsed entries, no-data-loss assertions, and exact region offsets at narrow widths, wide Unicode and wrapped headings. Compose primary prose word regions via existing `wordRegions`/`mergeRegions`; do not invent a second deck walker.
- [x] Verify non-Spanish output and `Dictionary.Lookup`/raw remain unchanged. Keep training predicates tied to primary Spanish data.

### 3. User-visible wiring and regression coverage

Files: main.go, play_loop.go, cloze.go, relevant main/editor/play/selection tests, README.md, atlas/define.md.

- [x] In lookupAndRender, keep raw on primary-only path; for normal display resolve both sections before deciding failure/capture. Capture once per user lookup. Preserve the existing typed/deck lookup-key audio identity and utteranceFor policy; pronounce once and do not derive voice from the English translation. Use canonical-primary matching only to select supplemental dictionary records. Preserve primary session context when available.
- [x] Use the same renderDefinitions output for one-shot/editor and full post-answer Choice and Cloze definition reveals. Reuse the primary parsed entry for options/board glosses. Resolve supplemental text only for an actual full-definition render, not every option-pool candidate.
- [x] Audit writeWords and practice write paths: they currently add deck regions over entire strings. Preserve section provenance through those paths so English prose is not remapped to Spanish deck entries. Reuse existing region-merging and already-rendered content handling; cover this through a live frame test, not just a pure renderer test.
- [x] Add fake-driven end-to-end tests for one-shot, editor, language switch, practice reveal, partial availability, zero LLM calls, one capture, Spanish audio (including initial audio and headword replay for inflected madrugaste), raw byte equality, and copy/region coordinates across both sections.
- [x] Document enabling Spanish Larousse and Spanish–English Oxford in Dictionary.app settings, waiting for downloads, and `define -lang es madrugar` / `/lang es`. Explain section ordering and partial setup diagnostics.
- [x] Run `go test ./cmd/define/...`, focused new race tests, `go vet ./...`, `GOOS=linux CGO_ENABLED=0 go build ./...`, strict native conformance and existing release-stamp check. Confirm no-data-loss and language-isolation guards have meaningful updated expectations.
- [x] Commit verified implementation and prepare evidence for one `sdlc close --issue 61 --verified '<actual evidence>'`. Address gate findings and merge via SDLC afterward. Release is a separate requested action.

## Architectural checks

- ARCH-DRY / ARCH-PURE: one section model and renderer; one pure record selector; existing dictionary factory, parser, vocabulary and region helpers remain owners.
- ARCH-PURPOSE: both explanations, in user-specified order, in ordinary full lookup and full dictionary reveal; neither a translation toggle nor LLM fallback replaces the requested behavior.
- ARCH-MOCK: stateful fake behind native record seam plus independently captured fixtures and strict live conformance.
- ARCH-CONSTRAINTS: bounded records/bytes, at most one supplemental search per full definition construction, no new network latency. Truncation is not accepted as a complete answer.
- ARCH-SECURE: no shell interpolation or external HTML rendering; copied dictionary markup is parsed as bounded data. Unknown source direction is rejected.
- ARCH-ORDER: lookup composition is synchronous and has no persistent pending state; existing editor serializes language switches and capture. Rebuild the whole dictionary composition on switch. No additional goroutines survive a lookup.
- ARCH-FUNERAL: lookup values die with the call or existing practice question/session; native copied refs released on every path. No new durable runtime artifact or cache.

## Review corrections

- 2026-09-14: Fresh review corrected the original instruction “Pronounce the Spanish canonical headword once.” This would turn spoken madrugaste into madrugar and violate existing lookup-key audio identity. Preserve utteranceFor and RenderOpts.Word, using the canonical headword only for dictionary matching; add initial/replay inflection coverage. Corrected the navigation name buildQuestions to todaysQuestions. Otherwise review approved the plan.


## Revision: /bilingual toggle (2026-09-14)

**Reason:** The user requested a command to switch between selected-language-only definitions and selected-language definitions followed by English. This supersedes the unconditional two-section behavior above. Spanish remains the first bilingual source adapter; the setting is independent of the selected language.

**Behavior:**

- `/bilingual` toggles the current value and prints the resulting state. `/bilingual on` and `/bilingual off` set it explicitly; invalid arguments fail without mutation. `/help bilingual` is read-only.
- On: render the selected language first, then English where a verified source adapter is available. Off: use the current primary-only display path, with no supplement lookups, section labels or missing-English warnings. `--raw` remains primary-only in either state.
- Save the value per deck alongside the existing language preference; without an authorized writable deck, change the interactive session only and report that it was not saved, using the existing deck-permission policy. One-shot commands must report their actual durable effect and must not claim a session-only change when there is no session. A real persistence failure leaves the live value unchanged.
- `/lang` reports both language and bilingual state. Switching `/lang` preserves the toggle while rebuilding language-derived sources. English mode does not repeat English as a second section. A language with no verified English adapter retains its primary definition and explains the unavailable supplement only when bilingual is on.
- The changed setting applies to subsequent lookups and newly started practice sessions; it does not replay audio, recapture the previous lookup, or append an automatic duplicate definition.
- New-deck default is awaiting the user's preference; if unanswered after a reasonable opportunity, use off to preserve current behavior. An explicit saved choice always wins.

**Additional pure entities and integration:**

| Name | Lives in | Status | Responsibility |
|------|----------|--------|----------------|
| parseBilingualArgs | cmd/define/bilingual_cmd.go | new | Toggle/on/off argument decision with no IO |
| Bilingual setting read/write | cmd/define/store/bilingual.go | new | Per-deck boolean, absent/malformed default and atomic writes |
| runBilingual / sessionSetBilingual | cmd/define/bilingual_cmd.go | new | Persistence-before-session-update, honest effect reporting |
| command registry / commandCtx | cmd/define/command.go | modified | Shared dispatch, completion, help and setting callbacks |
| deps / storeDeps / options propagation | cmd/define/main.go, command.go, replraw.go | modified | Startup state and session setting shared by lookup and practice |

**Implementation additions to Task 2/3:**

- [x] Add table tests for no arguments, on, off, case policy, extra/invalid arguments, explicit idempotent sets, and help without mutation. Register `/bilingual [on|off]` through the existing command table; help and completion derive from it.
- [x] Persist `bilingual.txt` using the existing atomic writer. Read absence/malformed values with the agreed default, write bounded `on`/`off` content, and register the filename in RuntimeFiles and matching gitignore/runtime-artifact guards. The single file is overwritten per change and ends with deletion of the deck; no append-only log or migration is needed.
- [x] Thread the startup setting and persistence callback through every openStore/applyTo return path, including DEFINE_NO_CAPTURE and declined deck creation. Reuse the established permission gate, not a new approval mechanism. Audit raw editor and line-loop command contexts and one-shot dispatch; all must read the same effective setting.
- [x] Add a settings round-trip test across process startups plus invalid-file, failed-write, declined-deck, read-only/session-only, and one-shot effect tests. Preserve the existing language preference.
- [x] Gate supplemental resolution at the shared display-composition entry point. Test that off performs zero native supplemental searches and emits no English availability diagnostic, even with missing/broken Oxford installation; switching on enables them on the next lookup. Defer supplement availability diagnostics until this gate so startup in off mode stays primary-only.
- [x] Add es/on→en/on→es/on and on→off→on integration tests across ordinary lookup and nested practice startup. Assert no duplicate English section in English mode, no stale source after language switch, no extra capture/audio caused by toggling, and current options copied into a new sitting.
- [x] Update `/help`, command completion fixtures, README command table/usage block, setup example and atlas. Cover `/bilingual on`, Spanish-first order, `/bilingual off`, default and persistence semantics.

**Additional review focus:** Settings lifetime is one value per deck/session, not per word or per language. Persistence succeeds before live mutation; deck-decline may intentionally produce session-only state. No new asynchronous work. Off must preserve legacy primary-only rendering, not render a one-section bilingual wrapper.


## Revision: default on (2026-09-14)

The user explicitly selected **on by default**. This supersedes the pending/default-off assumption in the preceding revision. Missing or malformed settings resolve to on; an explicit saved off remains off. Existing decks without a saved setting therefore gain bilingual display. Update default/read/startup tests and documentation accordingly.


### Toggle review clarifications (2026-09-14)

Fresh review approved the toggle/default-on revisions without blockers. Implement interactive session-only mutation even when no persistence callback exists: do not copy sessionSetLang's nil-callback refusal. Wire both repl.go and replraw.go. Carry an explicitly resolved setting at startup (or an optional value before resolution); never treat false as unspecified, because saved off must survive default-on fallback. The planned restart, DEFINE_NO_CAPTURE, failed-write and declined-deck tests cover these cases.

## Revision: function-level verification strategy (2026-09-14, PQ-1)

The following named surfaces are the execution contract for the case lists above. Each row's adversarial class is exercised independently from its implementation; failing assertions precede production changes.

| Production surface | Test surface and adversarial strategy | Mechanical guard |
|---|---|---|
| selectSpanishRecords | TestBilingualSelection over independently captured full arrays; FuzzBilingualRecords seeded with both directions, malformed XML, nested impostor IDs, truncation and huge attributes | Unknown direction cannot become selected; IDs unique; selected text belongs to input records; deterministic selection under duplicate/permuted candidates; no panic under fuzz |
| native record adapter | TestBilingualNativeDirection and TestBilingualNativeLimits with installed Oxford; stateful record-source fake drives empty, absent, malformed and incomplete sets | Strict conformance fails missing required symbols/dictionary; cap failures remain failures rather than ErrNoEntry; all returned records pass the pure selector |
| definitionsFor/composition | TestDefinitionAvailability uses a stateful primary/supplement fake with independently changeable installation/results and call journal | Every successful section preserved; no false success when both fail; primary fetched once; off/raw never query supplemental seam |
| renderDefinitions | TestDefinitionRendering and FuzzDefinitionRendering, seeded from real bilingual flat text and varied narrow/wide Unicode widths | Per-record source alphanumerics retained in order; Spanish precedes English; an independent stripped terminal-cell grid validates region text at every coordinate; English prose has no Spanish deck regions |
| parseBilingualArgs | TestParseBilingualArgs and FuzzParseBilingualArgs over arbitrary tokens | Only zero args or one on/off token changes state; invalid input cannot produce a valid mutation; same explicit set is idempotent |
| store setting parser/read/write | TestBilingualSettingRoundTrip and FuzzBilingualSetting using arbitrary file bytes; real temp directory for writes and seeded existing-off files | Missing/malformed resolves to on, saved off survives reload; atomic writes leave one bounded file; interrupted/failed replacement cannot claim success |
| sessionSetBilingual/runBilingual | TestBilingualCommandEffects with stateful setting fake whose persistence can fail/decline | Real write error preserves live value; interactive no-store/decline changes session only; one-shot never reports a nonexistent session effect; help has no effect |
| startup/applyLang and both command loops | TestBilingualSessionSwitch and TestBilingualStartup, driven through real dispatch/newStore seams with fake dictionaries | Saved off wins over default on; both command ingress paths see same value; es/en/es rebuild preserves setting and does not retain stale source |
| lookupAndRender/todaysQuestions/region writers | TestBilingualLookupPaths, TestBilingualPracticeReveal, TestBilingualClozeReveal, TestBilingualInflectedAudioAndCapture and TestBilingualScreenSelection using deterministic screen and audio/capture recorders | Off and raw are primary-only; one user lookup has one capture/initial audio; inflected initial/replay identity remains typed/deck word; terminal-grid oracle validates click/copy across sections and protected English prose |

PQ-2: Run strict native conformance before every release containing dictionary changes and after a macOS upgrade; this is an explicit maintainer check policy, not a claim that CI has Dictionary.app assets. Record its result in issue/release evidence.

PQ-3: Expected practice queue is the existing default 20 words; at most one supplemental search per full definition produced, none for board glosses/option-pool entries. Target warm local lookup overhead is below 250 ms per word and below 2 s for a default 20-word queue, to be measured against the native adapter during verification. These are acceptance budgets, not promised cancellation deadlines: DictionaryServices calls are synchronous and non-cancellable. A budget miss triggers profiling/re-plan before completion rather than silently adding unbounded background workers or claiming a timeout that cannot interrupt C. Byte/record bounds remain hard constraints.


## Revision: Spanish question quality (2026-09-14)

The user asked whether /play is suitable for Spanish study and should obey /bilingual. A fresh isolated probe over the committed Larousse corpus found a concrete defect: bonito's marked-correct meaning was `adjetivo (femenino bonita)`; other targets retained Spanish grammar labels because ParseEntry's POS vocabulary is English-only. Fix this extraction defect as part of Spanish readiness, not merely bilingual display. Add named tests in parse_spanish_test.go over actual fixtures and targetCandidate/optionCandidates/senseFacts; recognize Spanish POS/morphology without losing displayed source text or regressing English. Sweep target, pool, board gloss and harvest sense consumers because they share this parser. Keep the existing task's primary-language grading identity.

Pending user preference: whether English help is visible before answering or only afterward. The current recommendation/approved baseline is post-answer explanations, with Spanish prompts/options unchanged; before-answer bilingual cues would need deliberate per-question semantics and cannot pair independent dictionary senses by position.


## Done when

| Behavior | Verification |
|---|---|
| Correct Spanish-to-English direction and bounded native records | `TestBilingualSelection`, strict native conformance |
| Both sections ordered and unavailable sections explicit | `TestDefinitionAvailability`, `TestDefinitionRendering` |
| Toggle controls one-shot/editor/practice and survives language switches | `TestBilingualLookupPaths`, `TestBilingualSessionSwitch`, `TestBilingualPracticeReveal`, `TestBilingualClozeReveal` |
| Default on and saved off survive startup; declined deck remains unwritten | `TestBilingualStartup`, `TestBilingualDeclinedDeckWritesNothing` |
| English-section text remains selectable | `TestBilingualScreenSelection` |
| Spanish question targets, distractors and harvest senses contain definitions rather than grammar | `TestSpanishPracticeUsesDefinitionsInsteadOfGrammar`, `TestSpanishGrammarLabelsDoNotBecomePracticeAnswers` |

## Implementation reconciliation (2026-09-14)

Choice and Cloze full dictionary reveals both obey `/bilingual` **after the
answer**. Questions, answer options, compact board glosses and grading remain
Spanish. The user's optional pre-answer-help clarification is unanswered; the
approved post-answer baseline remains in force. No pre-answer translation or
cross-dictionary sense alignment was added.

`TestBilingualPracticeReveal` covers Choice prompts and on/off reveal/search
counts; `TestBilingualClozeReveal` covers Cloze on/off reveal content.
`TestBilingualInflectedAudioAndCapture` pins initial and replay audio for the
requested inflection. `TestBilingualScreenSelection` exercises actual English
text copying through the screen/router. `TestSpanishPracticeUsesDefinitionsInsteadOfGrammar`
and `TestSpanishGrammarLabelsDoNotBecomePracticeAnswers` cover the repaired
Spanish parser and primary-language candidate/harvest consumers.

The focused new tests passed with `go test ./cmd/define/... -run
'TestBilingual|TestDefinition|TestParseBilingual|TestSpanish' -count=1`.
The approved toggle revision supersedes the earlier unconditional Italian-output
compatibility statement: English remains single-section; unsupported non-English
languages preserve primary content and add an unavailable-English diagnosis only
when on. Off/raw retain the original primary-only output contract.

Unchecked rows retain outstanding combined requirements; they are not waived by
nearby passing tests. Final verification, native/performance evidence, commit,
SDLC review and merge remain the integrating agent's responsibility.


### Final implementation evidence (2026-09-14)

All implementation and verification rows are evidenced. `go test ./...` passed
(define 111.545s); focused race checks, `go vet ./...`, Linux cross-build,
strict native direction/limits/assembled factory checks and release-stamp
conformance passed. Rendering fuzz ran 61,742 inputs and command fuzz 32,316;
native selector and setting fuzz evidence is recorded in the issue log.
`TestBilingualAvailabilityThroughLookup` covers the installed/missing/malformed/
native-failure matrix through actual composition and capture. `TestBilingualNestedPracticeUsesCurrentSetting`
covers editor-to-practice on/off/on state. `TestBilingualExplicitSetsAreIdempotent`
and `TestBilingualHelpDoesNotMutate` cover command effects. `TestBilingualSourceAvailabilityNames`
covers source availability (see its exact test declaration for subcases).
The screen test also rejects a Spanish action on English red. The native assembled
factory check compares off and raw with the original primary entry.


## Revision: assisted practice before answering (2026-09-14)

**Status:** design ready for approval. User clarification supersedes the prior
post-answer-only baseline: a beginner needs English assistance while answering;
turning bilingual off later removes that assistance. Keep the implemented
lookup/reveal work and the reviewed metadata fix; do not close or merge #61 until
this extension is implemented and verified.

### Behavior

- Meaning choices: Spanish first, then an English translation of each displayed
  definition. Translate every option equally; do not translate the target
  headword or identify the correct option. Preserve choice order and grading.
- Cloze: Spanish blanked sentence, then its English translation with the same
  blank. Choices remain Spanish words. Never translate the restored sentence
  before grading or send it to the assistance task.
- Board: keep the Spanish grid; show Spanish then English in the selected word's
  gloss panel. It is self-assessment, so explaining that word is intentional.
  Bilingual non-English boards reserve two panel rows from construction: one
  per language, clipped independently to width, even if help is unavailable.
  The same row budget drives Rows/boardFits and fallback selection. Grid columns
  and hit coordinates stay unchanged; no asynchronous help can resize a board.
- Full answer explanations remain Spanish then English as already implemented.
  Off means all of these surfaces are Spanish-only, with no assistance cache or
  model work. English decks remain unchanged.
- Prepare help before the first question using the configured CLIProxyAPI model
  and common Braille spinner. Reuse cached successful translations in later
  sessions. A missing model or failed translation gives one concise warning and
  leaves original Spanish material playable; never single out the right answer
  by translating only one option.

### Additional concepts and integration

| Concept | Planned location | Kind | Responsibility |
|---------|------------------|------|----------------|
| Assistance source and checked translation | cmd/define/practice_help.go | PURE | Versioned source-language/mode/exact-text identity, bounded response validation and blank preservation |
| Assistance preparation | cmd/define/practice_help_client.go | INTEGRATION | Existing llm.Client/task, foreground spinner, cancellation, one queue's deduplicated batches and warning |
| Assistance cache | cmd/define/store/practice_help.go | INTEGRATION | Atomic per-deck bounded cache; in-memory fallback without a writable deck |
| Choice/Cloze/Board help presentation | cmd/define/play/choice.go, cloze.go, board.go | PURE | Spanish-first rendering with original grading/identity and independent English spans |
| Question preparation boundary | cmd/define/play_loop.go, cloze.go | INTEGRATION | Both direct and nested practice prepare help before playback starts |
| Explicit authoring language | cmd/define/harvest_item.go | PURE | Require authored sentences in the selected language, including Spanish |

ARCH-DRY: one exact-text translation adapter serves all forms. Do not substitute
Oxford's first headword gloss for translation of a selected Larousse sense.
ARCH-PURE: model requests receive displayed prose only, not answer indexes,
Correct flags, Axis, or hidden headwords. Keep model/cache IO outside package play.
Choice receives English only after every option validates. Cloze requests receive
already blanked prose and preserve the exact count of a reserved placeholder;
reject missing/extra placeholders, empty/oversized/control-character output, or
the target answer copied into assistance. English assistance must not inherit
Spanish vocabulary/audio actions; both languages remain selectable.

ARCH-CONSTRAINTS: cap one source at 4 KiB, one response at 8 KiB, a batch at 16
sources/32 KiB, and one queue preparation at 30 seconds with cancellation. Dedup
source texts before model work; cache hits cost no model calls. At a limit, warn
once and retain Spanish-only material for incomplete questions. No retries in
foreground beyond the existing bounded client policy; failures are not cached.


Cache hits are untrusted data and pass the same structural validation as fresh
responses plus current-question checks (including the current Cloze answer).
The shared content key does not exempt cached prose from question-specific leak
checks. An unusable hit becomes a miss; never apply a partial Choice set.

ARCH-FUNERAL: practice preparation owns a single versioned practice-help.json
cache per deck, partitioned by source language and exact source/mode/version key.
Read at most 1 MiB; write at most 512 entries and 1 MiB, evicting oldest entries
before atomic replacement. No append log. Content changes naturally miss; old
policy versions are discarded on rewrite. Deleting the deck removes the cache.
No-capture or declined deck permission uses only session memory; cache write
failure retains the valid live translation. Reuse existing permission and atomic
write helpers; register the runtime filename and gitignore entry.

### Implementation and verification tasks

- [ ] Add source/response validation tables and fuzzing before implementation:
  exact option association; completeness; unchanged Spanish/order/grades;
  placeholder mismatch, target leakage, control text and size limits. Use
  independent expected strings, not the production transformation as oracle.
- [ ] Add a real temp-directory cache suite and stateful memory equivalent:
  first request fetches, restart hits, text/language/mode/version misses,
  eviction bounds, corrupt files, failed atomic writes, no-capture and off.
- [ ] Implement assistance preparation using existing structured llm tasks and
  stateful llmtest service. Assert request bodies exclude hidden answer metadata;
  simulate partial responses, malformed output, outage, delayed cancellation,
  cache hits, bounded batches and one warning. No failures enter the cache.
- [ ] Wire both standalone and nested /play queue startup. Refactor Cloze item
  selection once so the blanked text being translated is exactly what renders.
  Update prior prompt-identity tests: off equals original, on includes complete
  equal help; selection, answers, grades, capture and audio remain unchanged.
- [ ] Add pure form presentation and terminal integration tests for Choice,
  Cloze and Board at narrow widths, English action isolation, selection/copy,
  unchanged grid hit-testing and safe spinner cancellation. Add an explicit
  Spanish author prompt requirement and fake-driven authoring regression.
- [ ] Update README/atlas and existing offline-practice comments: preparation can
  call the proxy once for uncached material; answering uses prepared local data.
  Add opt-in live translation conformance for Spanish polysemy and blank safety.
- [ ] Run focused red/green/fuzz/race tests, full Go suite, native conformance,
  vet, Linux build and repository guards; then rerun close review and merge.

The extension changes latency and introduces cached model translations, so it
needs explicit plan approval under AGENTS.md §2 before new implementation.

## Revisions

- 2026-09-14: Moved the record boundary to the end of the document so later approved scope updates and verification contracts remain visible to repository guards. The dated design/implementation updates above retain the decision trail. Updated the Core concepts table to exact implemented symbol paths and added test-backed Done when rows. No user behavior changed.

- 2026-09-14: Reconciled implemented Choice/Cloze post-answer scope and actual test symbols, recorded focused verification, and checked only evidenced implementation rows. Preserved optional pre-answer help as unanswered and left final verification/close and incompletely covered composite rows open.

- 2026-09-14: Final checklist now records completed gate preparation; the close review and merge themselves remain subsequent SDLC transitions. Committed-window repository guards passed.

- 2026-09-14 (BR-2): Classified core concepts as PURE or INTEGRATION and split mixed parser/effect rows. This clarifies verification boundaries without changing behavior.

- 2026-09-14 (BR-1): Extracted the Spanish factory through `spanishDictionaryFromInstalled` so its metadata-failure diagnostics are tested at actual lookup/composition. `TestBilingualFactoryMetadataDiagnostic` distinguishes failed enumeration (nil) from confirmed empty installation, with usable and absent English. Unknown primary availability reports the metadata failure rather than download advice.

- 2026-09-14: User confirmed English assistance must be available before answering because pure Spanish exceeds their current level. Added the assisted-practice design above; this supersedes earlier post-answer-only statements. Existing implementation is retained but the issue remains open pending this extension.

- 2026-09-14: Fresh assistance-plan review identified panel row budgeting and per-question cache validation. Added a shared two-row bilingual Board budget and mandatory structural/current-answer validation on cache hits; both findings are resolved in the design.
