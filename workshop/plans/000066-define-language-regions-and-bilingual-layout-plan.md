# Language regions and bilingual layout implementation plan

> **For agentic workers:** Follow AGENTS.md §3 and superpowers-executing-plans. Delegate bounded source-parser and row-painter work after the common output contract is settled. SDLC owns the close review.

**Goal:** Preserve dictionary structure and paint coherent, full-width language regions instead of text-run strips.

**Architecture:** Producers emit ordered text with explicit structural/language regions. One shared layout path carries text, click regions and language-region metadata through wrapping. The terminal painter fills physical rows at current width; plain text and selectable cells contain no synthetic padding. Oxford HTML supplies structure, emphasis and ownership in one bounded parse.

**Tech stack:** Existing Go renderers, XML parser, import-free play presentations, screen/selection core, native dictionary captures and stateful SSE fake.

**Status:** Operator approved the corrected preview and layout on 2026-09-15. Entering implementation gate.

## Visual contract

The screenshot changes #65's contract: background belongs to a whole language region, including indentation, numbers, headings, short lines, continuation rows and interior blank rows. For dictionary output, the entire returned dictionary section is the presentation region, including foreign-language quotations and translations. This is a visual grouping rule, not a claim that every token has the same source language. Unknown-source fallback dictionaries stay neutral.

Keep existing foreground colors and emphasis. Answer marks and selection retain precedence. Do not pad stored text, guess languages from spellings, infer ownership from ANSI colors, or repaint old output with the new `/lang` setting.

Oxford keeps its source order and hierarchy: headword, grammatical A/B/C groups, numbered senses, lettered sub-senses, idioms, examples and translations. **Do not change background within a dictionary definition.** With `/lang es`, the whole verified Spanish primary section is tinted and the whole English-explanation Oxford supplement is normal, including its Spanish examples. With `/lang en`, a shown English-explanation section is tinted as one block. Headings and interior blank rows share the section background. Each Spanish example and its English translation stay together on one logical row, wrapping naturally at the available width. Preserve their ordering, emphasis and source association; do not insert a language-switch line break. Headings, senses and separate example pairs retain their own structural rows.

Actual source-language spans remain available for provenance and actions. Keep a separate explicit section presentation role: primary uses verified source language, Oxford supplement uses its provider-declared English-explanation role. Do not derive that role from a display label or guess it for every-active-dictionary fallback. In full practice reveals, preserve these same dictionary section boundaries.

Outside returned dictionary definitions, keep the planned row-level policy for practice/model output: a physical row with mixed or unknown substantive prose stays neutral, and known target rows receive full fill. The dictionary section rule takes precedence within embedded definitions; do not run their contents through the per-row language classifier again.

Preview: `/tmp/define66-region-preview.html` (dark/light and es/en toggles). The Oxford examples come from installed native `rendir`. This is a design illustration, not a production screenshot. Updated after operator rejected alternating language backgrounds within dictionary definitions.

## Alternatives

- Pad current ANSI strings: small patch, but contaminates copy/history and freezes old terminal widths; reject.
- Alternate background at language changes inside dictionary output: rejected by operator as distracting zebra striping.
- Uniform dictionary-section backgrounds with explicit presentation roles: selected by operator; actual source provenance stays independent.
- Preserve structural regions and paint rows at the terminal boundary: selected. More metadata plumbing, but formatting, source ownership, copy and resize stay independently correct.

## Core concepts

### Pure entities

| Name | Lives in | Status | Responsibility |
|---|---|---|---|
| `parseBilingualDocument` → `bilingualDocument` | `cmd/define/bilingual_layout.go` | new | Ordered structural nodes and inline runs from one bounded Oxford XML walk |
| `languageText` | `cmd/define/language_text.go` | unchanged | Verified byte ownership; presentation regions live separately in renderedOutput and Presentation |
| `renderedOutput` | `cmd/define/output_layout.go` | new | Unpadded styled text, click regions and semantic region metadata in one validated value |
| `layoutOutput` | `cmd/define/output_layout.go` | new | Shared wrap/clip projection of text, click coordinates and row ownership |
| `paintLanguageRow` | `cmd/define/language_row.go` | new | Paint physical row fill and exclusions at current width without altering source text |
| `selectionRow` | `cmd/define/selection_frame.go` | modified | Paint metadata separate from selectable cells |
| `Presentation` | `cmd/define/play/presentation.go` | modified | Producer-owned row/region ownership and answer-state exclusions; no imports |

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| `renderDefinitions` | `cmd/define/definitions.go` | modified | String adapter around the structured definition core |
| `renderDefinitionOutput` | `cmd/define/definitions.go` | new | Primary and structural Oxford output shared by lookup/Choice/Cloze reveals |
| `liveScreen` | `cmd/define/screen.go` | unchanged | Existing synchronized IO shell; new methods carry structured output |
| `screen` | `cmd/define/screen.go` | modified | Stores immutable row paint beside unpadded text and actions |
| `WriteOutput` | `cmd/define/output_screen.go` | new | Atomic structured text/action/paint ingress |
| `answerWrapWriter` | `cmd/define/answerwrap.go` | unchanged | Legacy unannotated wrapping API retained for existing callers |
| `ownedAnswerWrapWriter` | `cmd/define/answerwrap.go` | new | Bounded structured physical-row emission before any sink receives paint |
| `languageAnswer` | `cmd/define/answer_language.go` | modified | Existing decoded stream and plain history, explicit row-ownership accumulation |
| `renderPracticeOutput` | `cmd/define/practice_output.go` | new | Prompt/reveal/chrome/footer regions and dictionary-section paint |

## Detailed contracts

### Oxford structure

Native `rendir` capture: `/tmp/define66-rendir-native.json`, 11,404 HTML bytes / 1,960 Text bytes. Structure digest: `/tmp/define66-rendir-native-tree.txt`. Promote the exact native capture with provenance/regeneration instructions during implementation; never hand-author a service response.

Recognize actual structural classes: `gramb`, nested `semb`, `exg`, `trg`, `idmb`/`idmsec`; retain `hw`, `ps`, `sn`, `ex`, `trans`, `ind`, `lg`, `co`, `idm` and explicit emphasis. Unrecognized wrappers retain ordered text/children. Reuse identity validation, size/depth limits, source-language classification and exact text-alignment checks. Consolidate the current ownership XML walk with the structural walk rather than maintaining separate class grammars.

Every source leaf must be emitted exactly once and in order. Do not invent sense text or infer divisions from flattened punctuation. Native Text correspondence is verified before trusting ownership; unsupported/malformed structure falls back to readable neutral source text without losing the primary section, and exposes a concise formatting diagnostic. Preserve all source content even when a class cannot be styled. No external XML entities or fetched DTDs.

### Row ownership and paint

Keep text, click regions and fill metadata together through wrapping; carry immutable resolved background per output so later `/lang` switches affect only subsequent output. Region ownership is explicit, never deduced from existing background escape bytes. Ordinary plain writes have no fill.

Dictionary rows inherit their explicit section presentation role uniformly, regardless of embedded languages. Elsewhere, a row is target-owned when its substantive prose belongs to that target region; structural decorations do not disqualify it, while unknown substantive prose or a second language makes it neutral. Producer-owned empty rows inside a region retain its fill; screen gaps, prompt spacing and unrelated empty rows stay neutral.

At paint time, clip/split using existing display-unit geometry, compose original foreground styles with fill, render remaining cells to the current terminal width, then reset background before cursor movement/newline/erase. Protect marked-answer cells through explicit exclusions. Use the same composer for ordinary paint and selection repaint. Selection cells and clipboard use original unpadded text. Exact-width rows must not cause extra wraps/scrolls.

Structured output extends the existing `WriteRegions` seam with an atomic text/regions/ownership value; plain writers get a shared serializer. One-shot terminal output and exit transcript serialize with the current width, while pipes/no-color/off preserve clean layout without paint padding. Preserve current historical clipping-on-resize behavior; do not introduce historical reflow. Footer draw uses the same metadata path, not an ANSI string escape hatch. Bounds follow existing source/selection limits, with safe unfilled fallback when metadata is invalid.

### Streaming ownership

Reuse the #65 annotation/control decoder unchanged. `languageAnswer` stores only original decoded prose. The shared wrap/layout core owns **physical row** boundaries: source newline and inserted wrap both finalize that physical row. It carries language metadata through word wrapping before aggregating ownership; never aggregate a whole source paragraph and later retroactively recolor its completed rows.

Use the pure `advanceRowOwnership` transition function with states empty, known(language), and mixed/unknown. Events: append substantive run, append structural decoration/whitespace, finalize(source-newline or wrap), resize(width), finish. Empty + known prose becomes known; known + same stays known; unknown or another substantive language becomes mixed; mixed cannot become known until finalize resets to empty. Decorations do not independently establish language; producer-declared blank region rows can carry explicit ownership. Source newline and wrap finalize identically for ownership but retain distinct text-boundary provenance so stored logical answers never gain display-only newlines.

**Both live and append-only color output buffer the unfinished physical row until finalize/finish**, then emit its final text/ownership atomically. Do not paint provisional tint and later retract it. Completed physical rows stay immutable. Latency trade-off: at most one unfinished physical row is held in addition to the existing bounded annotation segment. Preserve the existing 64 KiB unfinished-text cap; exceeding it stops display with the existing write-failure diagnostic while clean decoded history still finishes. Width-zero plain output streams clean text without fill buffering. A live resize supplies a new width to the pure layout core before its next emission: only uncommitted pending text is laid out at that width, while old rows retain the existing clip-only history policy. The screen painter independently fills each row to its current width.

Finish, cancellation, truncation and failure run the same finalization path before recording partial history. Output errors poison later paint but do not prevent clean decoder/history completion. There is no earlier-row repaint protocol because no sink receives a row whose ownership can change. Regression: Spanish spans fill several wrapped rows, then English/unknown prose arrives before the source newline; earlier all-Spanish physical rows remain filled, only the mixed physical row is neutral. Assert the same completed rows for live and ordinary sinks, at every chunk split and on finish/cancel, with resize while a row is pending.

## Implementation and verification

### Function-level test strategy

Direct pure-core tests use independent source-order and terminal-cell oracles; integration tests then prove each consumer reaches those cores.

| Production function | Adversarial input class | Independent mechanical guard |
|---|---|---|
| `parseBilingualDocument` | Exact native Oxford records with nested senses, paired examples, idioms/emphasis, repeated words, unknown wrappers, malformed/truncated XML and size/depth limits | Literal A/B/C and A2/A4 tree assertions plus ordered source-leaf conservation against the captured native record; fuzz rejects panic, duplication, loss and ownership on invalid sources; removing a group boundary must fail |
| `layoutOutput` | Narrow/wide Unicode, exact-edge wrapping, embedded multi-language dictionary sections, interior blanks, invalid metadata and narrow→wide→narrow clipping | Literal physical rows and click-cell coordinates at fixed widths, source-text conservation and bounds assertions; mutating section ownership or retaining stale width must fail |
| `paintLanguageRow` | Short/indented/blank rows, embedded SGR resets, semantic answer exclusions, selection inverse, tint off and width changes | Independent terminal-state emulator asserts every cell's background including blank cells, unchanged foreground/excluded cells, reset before movement and no extra wrap/scroll; mutations removing padding/reset must fail |
| `advanceRowOwnership` | Exhaustive empty/known/mixed states crossed with same/foreign/unknown prose, decorations, finalize/wrap/finish events | Literal transition table, mixed-state absorption until finalize, no language from decoration, explicit blank-region inheritance and dictionary-role bypass; mutation assigning mixed rows to target must fail |
| `ownedAnswerWrapWriter` / `languageAnswer` | Every-byte splits of real SSE capture, multiple pure wrapped rows followed by mixed prose, pending-row resize, cancellation/truncation/error and 64 KiB overflow | Identical finalized physical rows across chunk splits/live/append sinks, unchanged decoded logical history, immutable completed-row ownership and bounded pending text; no display-only newlines/padding in history |
| `selectionCells` / `selectedText` | Selection across synthetic padding, clipped wide glyphs and source spaces at multiple widths | Literal clipboard text derived from unpadded source; paint-padding mutation must fail |
| `renderDefinitionOutput` / `renderPracticeOutput` / `boardFooterOutput` | Lookup, both full reveals, prompt/help/answer rows, footer and unknown dictionary fallback | Literal text/action coordinates plus terminal-cell section-role assertions; mutation losing footer metadata or alternating backgrounds within a dictionary must fail |

### Task 1 — Source structure and regression oracle

- [x] Promote exact native `rendir` capture with provenance; add the direct parser and render regressions above, run against the flat path and record failures.
- [x] Implement the bounded structural core and integrate with `definitionSection`/`renderDefinitions`, preserving ordinary primary parsing and the source-conservation guards.

### Task 2 — Shared physical-row paint

- [x] Add the direct layout/painter/selection regressions above and demonstrate the current text-strip failure.
- [x] Implement validated structured output and shared geometry projection; wire screen buffer/paint, `selectionRow`, selection repaint, terminal/plain serializers and exit transcript. Keep synthetic fill out of source strings/cells.

### Task 3 — Consumer sweep

- [x] Wire ordinary dictionary lookup and both full practice reveals; preserve headword/vocabulary/origin action regions through new Oxford rows.
- [x] Wire Choice/Cloze/Board prompt/reveal/help/footer and semantic answer exclusions via producer metadata. Dictionary glosses continue using verified dictionary-source ownership.
- [x] Replace model text-run tint with row metadata using `advanceRowOwnership`; demonstrate the streaming guards above through the existing real-capture SSE fake.

### Task 4 — Demonstrate and close

- [x] Update README/atlas to supersede text-only tint. Add integration-registry rows if new conformance files are introduced.
- [x] Run full Go suite, focused race checks, bounded source/stream fuzz, vet, Linux build and diff check. Run strict native `rendir` conformance and real PTY dark/light/off tests at multiple widths.
- [x] Capture actual rendered `rendir`, inspect dark/light output visually, and show the operator concrete output. Use the terminal-state oracle above; no visual-completion claim based solely on the design mockup.
- [x] Run the function-level mutations above, commit, pass the single SDLC close review, then publish.

Commands: `go test ./... -count=1`; focused `go test -race ./cmd/define/...`; bounded new fuzz targets; `go vet ./...`; `GOOS=linux CGO_ENABLED=0 go build ./...`; strict relevant `-tags conformance` tests; `git diff --check`. Exact focused test names land with each regression; passing requires the independent behavioral assertions above, not only unchanged text snapshots.

## Architecture and operating envelope

- ARCH-DRY: one source-class walk; one geometry projection and one row painter for normal/selection/footer/plain-terminal paths.
- ARCH-PURE: source parsing, layout, ownership accumulation and painting stay pure; native access and live-screen IO remain shells.
- ARCH-PURPOSE: sweep lookup, both reveals, practice footer and model rows; formatting preservation and full-width backgrounds are both deliverables.
- ARCH-MOCK: reuse native record fake, actual captures and stateful SSE replay; extend terminal-cell oracle and live PTY/native checks.
- ARCH-CONSTRAINTS: retain 1 MiB native source, 64-level XML validation and selection limits; linear source/layout walks plus work proportional to visible painted cells. No new network call, background worker, or terminal theme probe.
- ARCH-SECURE: external records validated before structural ownership; unknown provenance remains neutral; decoder continues filtering model controls before history/display.
- ARCH-ORDER: one pure partial-row transition owner; metadata and text enter screen atomically under its existing mutex. Resize repaints current width without rewriting history; late model chunks follow existing cancellation scope.
- ARCH-FUNERAL: region metadata dies with its response/screen buffer; no new runtime persistent artifact/cache. Committed captures are bounded regression fixtures; temporary previews/captures are development artifacts.

## Revisions

- 2026-09-15: Operator screenshot supersedes #65 ink-only/no-padding visual contract. Source inspection confirms Oxford hierarchy still exists in native HTML; renderer discarded it. Screen inspection establishes paint-time fill to preserve copy and resize. Mixed Oxford pairs are drafted as separate rows pending operator clarification. Estimate deferred until plan-quality gate, per workflow.

- 2026-09-15: Fresh review identified ambiguous logical-paragraph versus physical-row streaming ownership. Resolved to completed physical rows, bounded pending-row buffering for both live and append-only color sinks, immutable finalized rows, explicit wrap/source boundary distinction, and multi-wrap mixed-language/cancellation/resize regressions. This avoids repainting already emitted history and preserves sentence layout.

- 2026-09-15: Fresh reviewer approved the revised spec/plan with no remaining important findings. Mixed Oxford pair layout and the complete plan await operator review before change-code.

- 2026-09-15: Operator approved the general visual direction but rejected different backgrounds within one dictionary definition as distracting zebra striping. Supersedes the earlier example/translation color alternation: each dictionary result uses one section presentation role and continuous background; formatting remains structural and foreground emphasis remains intact. Source provenance is distinct from this visual role, unknown fallback stays neutral, and embedded full reveals preserve section identity. Preview updated at the same path.

- 2026-09-15: Operator explicitly selected “Keep bilingual pairs on one row; use the section background.” Each example/translation pair is one logical row with natural wrapping. Supersedes all earlier proposals to place the two languages on separate rows, including the old pending clarification. Preview updated to inline pairs; uniform section background remains unchanged.

- 2026-09-15: Operator explicitly approved the final preview (“yes, looks great”): uniform dictionary-section backgrounds and inline bilingual pairs. Proceed to change-code, estimate and implementation without further layout approval.

- 2026-09-15: Plan-quality finding PQ-1 requested named pure-function test strategies. Named `parseBilingualDocument` and `advanceRowOwnership`; replaced duplicated case inventories with one function/input-class/independent-guard matrix. Integration commands and actual visual acceptance remain. Approved product behavior and scope are unchanged.

- 2026-09-15: Implementation preserves the established unpadded `Transcript()` API.
  `PaintedTranscript()` owns terminal handback and `OutputTranscript()` transfers
  text/actions/paint between nested screens. This avoids clipping or padding logical
  history when current-width painting occurs. Added a nested-transfer regression.

- 2026-09-15 — Close review BR-2: reconcile Core concepts with declaration-level
  implementation. languageText/liveScreen/legacy answerWrapWriter declarations are
  unchanged; screen storage changed, and ownedAnswerWrapWriter, WriteOutput,
  renderDefinitionOutput and renderPracticeOutput are new. The painter lives in
  language_row.go. BR-1 extends streaming verification through runAsk's terminating
  plain newline, for success and cancellation, plus the shared screen append policy.

- 2026-09-15 — Close review BR-3: word wrapping intentionally leaves overlong
  tokens intact, so the terminal serializer must additionally split them into
  display-unit physical rows before invoking the clipping painter. Shared
  outputWrappedRows also supplies definition ingress and practice ownership
  projection; historical screen rows retain clip-only resizing. Regression scope
  includes headword/body tokens, wide glyphs, action/exclusion mapping, all profiles
  and clean width-zero serialization.

### 2026-09-15 — BR4: one geometry for text and metadata

Invariant: text, actions and exclusions share physical-row geometry (ARCH-DRY,
ARCH-PURPOSE). Consumer enumeration found practice actions still using legacy
word-only mapping and the regionWriter fallback serializing text without moving
actions. Practice now supplies source actions to renderPracticeOutput's shared
layoutOutput, and the fallback layouts the entire renderedOutput before emitting.
Definitions already use layoutOutput; pinned liveScreen layouts whole outputs;
streaming owns physical rows and has no actions; nested-session transfer carries
already-physical text/actions/paint. Footer chrome has no click regions and uses
shared physical display-unit boundaries for exclusion slicing. Legacy WriteRegions
retains paired legacy wrapping/mapping outside the structured-output path.
Production-path regressions through writePracticePresentation into liveScreen and
through the regionWriter fallback fail before and pass after these corrections.

### 2026-09-15 — acceptance and publication

Close round 4 returned SHIP: all four findings addressed, none new. Final full
suite (127.818s), race and strict native/PTY checks passed in the implementation
session. Published PR #49; deterministic merge/archive is the remaining SDLC step.
