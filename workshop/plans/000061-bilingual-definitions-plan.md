# Bilingual Spanish Definitions Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use superpowers-executing-plans for warm-context integration and bounded subagents for the native adapter and pure record selection. Steps use checkbox syntax.

**Goal:** Spanish word lookups show Spanish definitions followed by English explanations from installed macOS dictionaries.

**Architecture:** Preserve the primary Dictionary.Lookup contract for raw output, word identity, pronunciation, and practice construction. Add an optional supplemental-definition capability to the Spanish dictionary composition. Compose independently rendered, source-labeled sections at full-definition display sites; select Oxford records by verified source direction rather than its misleading dictionary-level metadata.

**Tech Stack:** Go, macOS DictionaryServices/CoreFoundation via runtime-resolved cgo, existing terminal renderer and stateful dictionary fakes. No LLM or new network dependency.

## Core concepts

| Name | Lives in | Status |
|------|----------|--------|
| bilingualRecord | cmd/define/bilingual.go | new |
| selectSpanishRecords | cmd/define/bilingual.go | new |
| definitionSection / definitionSet | cmd/define/definitions.go | new |
| renderDefinitions | cmd/define/definitions.go | new |
| dictionaryFor | cmd/define/dictselect.go | modified |

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
| buildQuestions full-definition renderer | cmd/define/play_loop.go | modified | Practice choice reveal and region map |
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

- [ ] Capture and commit independent fixtures for red, mesa, madrugar, arbol, jalapeno, como, solo, pie, son, madrugaste and mesas; confirm exact API signatures and ownership from the completed probe.
- [ ] Write failing pure tests for wrong direction, malformed/unknown root, duplicates, exact versus accent matches, canonical inflection matching, stable homograph ordering, limits and no usable match.
- [ ] Implement the pure selector, stateful record fake and bounded runtime-resolved native adapter. Native failure stays separate from entry absence. Add a portable stub.
- [ ] Run `go test ./cmd/define -run 'TestBilingual' -count=1` and strict live `CONFORMANCE_STRICT=1 go test -tags conformance ./cmd/define -run '^TestBilingualNative' -count=1`; the latter must exercise Spanish red→net/network and reject English red→rojo.

### 2. Ordered definition sections and language composition

Files: definitions.go, definitions_test.go, dictselect.go, dictselect_test.go, dict_darwin.go, dict_stub.go, lang_scope_test.go.

- [ ] Write failing tests for Spanish-first ordering, both sources installed, either source missing, both missing, per-word misses, native errors, and malformed bilingual results. Do not suppress a good section because the other fails.
- [ ] Implement the optional dictionary capability and constructor wiring through the existing `newDict` seam; use known dictionary IDs, never localized name matching. Extend `/lang` source reporting and cover es→en→es rebuilds.
- [ ] Implement section rendering with source labels, independently parsed entries, no-data-loss assertions, and exact region offsets at narrow widths, wide Unicode and wrapped headings. Compose primary prose word regions via existing `wordRegions`/`mergeRegions`; do not invent a second deck walker.
- [ ] Verify non-Spanish output and `Dictionary.Lookup`/raw remain unchanged. Keep training predicates tied to primary Spanish data.

### 3. User-visible wiring and regression coverage

Files: main.go, play_loop.go, relevant main/editor/play/selection tests, README.md, atlas/define.md.

- [ ] In lookupAndRender, keep raw on primary-only path; for normal display resolve both sections before deciding failure/capture. Capture once per user lookup. Pronounce the Spanish canonical headword once; do not derive voice from the English translation. Preserve primary session context when available.
- [ ] Use the same renderDefinitions output for one-shot/editor and full choice-definition reveal. Reuse the primary parsed entry for options/board glosses. Resolve supplemental text only for an actual full-definition render, not every option-pool candidate.
- [ ] Audit writeWords and practice write paths: they currently add deck regions over entire strings. Preserve section provenance through those paths so English prose is not remapped to Spanish deck entries. Reuse existing region-merging and already-rendered content handling; cover this through a live frame test, not just a pure renderer test.
- [ ] Add fake-driven end-to-end tests for one-shot, editor, language switch, practice reveal, partial availability, zero LLM calls, one capture, Spanish audio, raw byte equality, and copy/region coordinates across both sections.
- [ ] Document enabling Spanish Larousse and Spanish–English Oxford in Dictionary.app settings, waiting for downloads, and `define -lang es madrugar` / `/lang es`. Explain section ordering and partial setup diagnostics.
- [ ] Run `go test ./cmd/define/...`, focused new race tests, `go vet ./...`, `GOOS=linux CGO_ENABLED=0 go build ./...`, strict native conformance and existing release-stamp check. Confirm no-data-loss and language-isolation guards have meaningful updated expectations.
- [ ] Commit, run one `sdlc close --issue 61 --verified '<actual evidence>'`, address findings, and merge via SDLC. Release is a separate requested action.

## Architectural checks

- ARCH-DRY / ARCH-PURE: one section model and renderer; one pure record selector; existing dictionary factory, parser, vocabulary and region helpers remain owners.
- ARCH-PURPOSE: both explanations, in user-specified order, in ordinary full lookup and full dictionary reveal; neither a translation toggle nor LLM fallback replaces the requested behavior.
- ARCH-MOCK: stateful fake behind native record seam plus independently captured fixtures and strict live conformance.
- ARCH-CONSTRAINTS: bounded records/bytes, at most one supplemental search per full definition construction, no new network latency. Truncation is not accepted as a complete answer.
- ARCH-SECURE: no shell interpolation or external HTML rendering; copied dictionary markup is parsed as bounded data. Unknown source direction is rejected.
- ARCH-ORDER: lookup composition is synchronous and has no persistent pending state; existing editor serializes language switches and capture. Rebuild the whole dictionary composition on switch. No additional goroutines survive a lookup.
- ARCH-FUNERAL: lookup values die with the call or existing practice question/session; native copied refs released on every path. No new durable runtime artifact or cache.
