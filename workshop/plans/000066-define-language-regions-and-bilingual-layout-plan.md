# Language regions and bilingual layout implementation plan

> **For agentic workers:** Follow AGENTS.md §3 and superpowers-executing-plans. Delegate bounded source-parser and row-painter work after the common output contract is settled. SDLC owns the close review.

**Goal:** Preserve dictionary structure and paint coherent, full-width language regions instead of text-run strips.

**Architecture:** Producers emit ordered text with explicit structural/language regions. One shared layout path carries text, click regions and language-region metadata through wrapping. The terminal painter fills physical rows at current width; plain text and selectable cells contain no synthetic padding. Oxford HTML supplies structure, emphasis and ownership in one bounded parse.

**Tech stack:** Existing Go renderers, XML parser, import-free play presentations, screen/selection core, native dictionary captures and stateful SSE fake.

**Status:** Operator accepts the general layout with a correction: dictionary sections use one uniform background, without alternating colors inside definitions. No runtime implementation yet.

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
| `bilingualDocument` | `cmd/define/bilingual_layout.go` | new | Ordered structural nodes and inline runs from one bounded Oxford XML walk |
| `languageText` | `cmd/define/language_text.go` | modified | Preserve verified byte ownership; add explicit region boundaries/decorations independently of source prose |
| `renderedOutput` | `cmd/define/output_layout.go` | new | Unpadded styled text, click regions and semantic region metadata in one validated value |
| `layoutOutput` | `cmd/define/output_layout.go` | new | Shared wrap/clip projection of text, click coordinates and row ownership |
| `paintLanguageRow` | `cmd/define/language_style.go` | new | Paint physical row fill and exclusions at current width without altering source text |
| `selectionRow` | `cmd/define/selection_frame.go` | modified | Paint metadata separate from selectable cells |
| `Presentation` | `cmd/define/play/presentation.go` | modified | Producer-owned row/region ownership and answer-state exclusions; no imports |

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| `renderDefinitions` | `cmd/define/definitions.go` | modified | Primary renderer and structured Oxford output, shared by lookup/Choice/Cloze reveals |
| `liveScreen` | `cmd/define/screen.go` | modified | Atomic structured output, current-width paint, selection, resize and exit transcript |
| `answerWrapWriter` | `cmd/define/answerwrap.go` | modified | Bounded structured physical-row emission before any sink receives paint |
| `languageAnswer` | `cmd/define/answer_language.go` | modified | Existing decoded stream and plain history, explicit row-ownership accumulation |
| Practice output adapters | `cmd/define/practice_language.go` | modified | Prompt/reveal/chrome/footer regions and dictionary-source roles |

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

Use a pure row-ownership accumulator with states empty, known(language), and mixed/unknown. Events: append substantive run, append structural decoration/whitespace, finalize(source-newline or wrap), resize(width), finish. Empty + known prose becomes known; known + same stays known; unknown or another substantive language becomes mixed; mixed cannot become known until finalize resets to empty. Decorations do not independently establish language; producer-declared blank region rows can carry explicit ownership. Source newline and wrap finalize identically for ownership but retain distinct text-boundary provenance so stored logical answers never gain display-only newlines.

**Both live and append-only color output buffer the unfinished physical row until finalize/finish**, then emit its final text/ownership atomically. Do not paint provisional tint and later retract it. Completed physical rows stay immutable. Latency trade-off: at most one unfinished physical row is held in addition to the existing bounded annotation segment. Preserve the existing 64 KiB unfinished-text cap; exceeding it stops display with the existing write-failure diagnostic while clean decoded history still finishes. Width-zero plain output streams clean text without fill buffering. A live resize supplies a new width to the pure layout core before its next emission: only uncommitted pending text is laid out at that width, while old rows retain the existing clip-only history policy. The screen painter independently fills each row to its current width.

Finish, cancellation, truncation and failure run the same finalization path before recording partial history. Output errors poison later paint but do not prevent clean decoder/history completion. There is no earlier-row repaint protocol because no sink receives a row whose ownership can change. Regression: Spanish spans fill several wrapped rows, then English/unknown prose arrives before the source newline; earlier all-Spanish physical rows remain filled, only the mixed physical row is neutral. Assert the same completed rows for live and ordinary sinks, at every chunk split and on finish/cancel, with resize while a row is pending.

## Implementation and verification

### Task 1 — Source structure and regression oracle

- [ ] Add exact native `rendir` capture and independent assertions for A/B/C groups, A2/A4 lettered senses, example/translation associations, idioms and emphasis from the existing corpus. Run them against the flat path and record failures.
- [ ] Implement the bounded structural core and integrate it with `definitionSection`/`renderDefinitions`; preserve ordinary primary parsing. Run ordered-leaf/no-data-loss, malformed/truncated/unknown-class and repeated-word ownership tests.

### Task 2 — Shared physical-row paint

- [ ] Add literal background-cell regressions for uniform dictionary sections containing foreign examples, short lines, indentation, interior blank rows, SGR resets, answer exclusions and width changes; demonstrate current text-strip failure.
- [ ] Implement validated structured output and shared geometry projection; wire screen buffer/paint, `selectionRow`, selection repaint, terminal/plain serializers and exit transcript. Keep synthetic fill out of source strings/cells.
- [ ] Test narrow→wide→narrow resize, clipping before wide Unicode, exact-right-edge wrapping, frozen historical language, copy through padding, and no paint leaks into chrome or subsequent rows.

### Task 3 — Consumer sweep

- [ ] Wire ordinary dictionary lookup and both full practice reveals; preserve headword/vocabulary/origin action regions through new Oxford rows.
- [ ] Wire Choice/Cloze/Board prompt/reveal/help/footer and semantic answer exclusions via producer metadata. Dictionary glosses continue using verified dictionary-source ownership.
- [ ] Replace model text-run tint with row metadata using the pure streaming accumulator; test every-byte splits, mixed inline prose, cancellation, truncation, output error and clean history using existing real-capture SSE fake.

### Task 4 — Demonstrate and close

- [ ] Update README/atlas to supersede text-only tint. Add integration-registry rows if new conformance files are introduced.
- [ ] Run full Go suite, focused race checks, bounded source/stream fuzz, vet, Linux build and diff check. Run strict native `rendir` conformance and real PTY dark/light/off tests at multiple widths.
- [ ] Capture actual rendered `rendir`, inspect dark/light output visually, and show the operator concrete output. Assert background of blank cells in a terminal-state oracle that models SGR background and erasure, not only byte substrings. No visual-completion claim based solely on the design mockup.
- [ ] Mutation-check alternating backgrounds within dictionary sections, dropped Oxford grouping, lost footer metadata, stale width fill, padding entering copied text, and mixed non-dictionary rows wrongly receiving target ownership. Commit, pass the single SDLC close review, then publish.

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
