# Language response tint implementation plan

> **For agentic workers:** Follow AGENTS.md §3. Use superpowers-executing-plans; delegate bounded parser/renderer work after interfaces settle. The SDLC close gate owns the boundary review.

**Goal:** Make target-language passages easy to distinguish throughout bilingual definitions, practice and model answers.

**Status:** Implementation, verification and close review complete 2026-09-15; SHIP, ready for publication.

**Architecture:** Producers preserve explicit language ownership. One pure background composer consumes owned text and effective `/lang`. Dictionary source metadata, practice presentation roles and a bounded model-annotation decoder all feed that composer. Unknown text remains neutral.

**Tech stack:** Existing Go renderer, no-import `play` package, ANSI styles, dictionary HTML records, stateful LLM/SSE fake and native conformance harness. No language detector or adaptive-learning change.

## User-visible design

- A soft neutral background means “this passage matches `/lang`.” The rule also applies when English is selected; colors are not assigned independently to languages.
- Tint covers text and internal spaces, ending before line breaks, leading indentation and trailing padding. Wrapped continuation text resumes the tint; terminal erase/padding operations never inherit it.
- Other-language and unknown text use the normal background. Remove the current dimming of English practice help; retain labels, vocabulary foreground colors, dictionary emphasis and links.
- Selection and semantic answer markings take precedence. A marked practice word is shown with its existing answer state, suppressing language tint for that marked fragment.
- Use xterm-256 neutral backgrounds: dark profile 236 (`#303030`), light profile 254 (`#e4e4e4`). `-language-tint=dark|light|off` is an invocation preference, default dark to match the existing terminal palette. `-no-color`, non-TTY output, and `TERM=dumb` disable tint. Unsupported or unsuitable color rendering has the explicit off fallback; no terminal capability probe or theme-detection claim. Light/dark profiles affect only this new background, preserving existing foreground policy.
- Subsequent output follows successful `/lang` changes. Already emitted transcript keeps its original styling, just as #62 keeps the submitted command's language.
- Copied/stored prose has neither ANSI nor model annotations. Model annotations are decoded even with tint disabled. Streaming proceeds in short validated passages rather than displaying raw markers.

A browser preview at `/tmp/define-language-tint-preview.html` demonstrates both profiles and changing `/lang`. It illustrates color intent; actual terminal font/color behavior is verified separately.

## Alternatives considered

1. Tint each whole response section: smallest change, but wrong for mixed Oxford examples and inline model answers.
2. Guess each word's language: ambiguous across languages and cannot reliably distinguish repeated `red`, `son`, or `once`.
3. Preserve explicit ownership and compose tint once: selected approach. More metadata work, but it satisfies the inline requirement and supports all existing renderers without teaching them language detection.

## Core concepts

### Pure entities

| Name | Lives in | Status | Responsibility |
|---|---|---|---|
| `languageText` / `languageSpan` | `cmd/define/language_text.go` | new | Exact text plus nonoverlapping UTF-8 byte ranges with validated language, unknown outside ranges |
| `tintPolicy` / `styleLanguageText` | `cmd/define/language_style.go` | new | Current language and profile determine background; compose it without changing prose, foreground, or semantic state |
| `play.Presentation` / `LanguageRole` | `cmd/define/play/presentation.go` | new | Form-owned emitted text and byte ranges: Target, English, Neutral; explicit answer-style suppression |
| `promptBuilder` and form render walks | `cmd/define/play/choice.go` | modified | Prompt/reveal text and ownership derive from one layout walk, including truncation |
| Selected bilingual records and source projection | `cmd/define/bilingual.go` | modified | Preserve HTML-owned language spans beside source text; project only proven ownership into rendered fragments |
| `RenderOpts` and field rendering | `cmd/define/render.go` | modified | Preserve section/source language separately from lookup language; annotate headword and prose at known boundaries |
| `answerTextFilter` | `cmd/define/answer_text.go` | new | Incremental UTF-8 and terminal-control filtering before annotation parsing, bounded state and no retained control payload |
| `languageDecoder` | `cmd/define/language_decode.go` | new | Incremental model markers → clean owned spans, with bounded buffering and neutral recovery |
| `askContext` / `renderAskPrompt` | `cmd/define/askctx.go` | modified | Tell the model effective language and annotation grammar; retain current answer-level policy |

The main package owns actual language codes. `play` remains import-free: its roles mean target/English/neutral, resolved by the main adapter at presentation time. `Prompt()`/`Reveal()` and metadata must derive from the same builder; no independent searches for displayed words. Definition ownership and model ownership converge only at `languageText`, not through a new cross-tool package.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Invocation tint policy | `cmd/define/main.go` | modified | CLI flags and existing stdout/environment checks |
| Dictionary orchestration | `cmd/define/definitions.go` | modified | Selected dictionaries and pre-rendered reveals |
| Practice presentation adapter | `play_loop.go` | modified | Form presentations, word regions, board footer and reveal output |
| Annotated answer writer | `cmd/define/ask.go` | modified | LLM stream, clean session transcript, highlighting and wrapping |
| Native/source/model conformance | existing conformance tests plus `language_conformance_test.go` | new/modified | Installed dictionary records, real model requests, PTY lifecycle |

No persistent deck schema changes. Ownership lasts as long as its response; completed session text stores clean prose. Future #64 can choose language proportions through the same annotated response seam.

## Ownership contracts

### Dictionaries

`definitionsFor` must retain the selected bilingual record's Text and HTML-derived provenance instead of reducing it immediately to `[]string`. Reuse existing record selection and identity validation. Do not infer language from a section label.

- Primary monolingual headwords, verified prose fields and examples inherit the selected source language. IPA, numbering, generated punctuation and uncertain embedded foreign material are neutral. Origin text is conservatively neutral unless explicitly annotated.
- The Spanish–English supplement has Spanish headwords/examples/idioms and English translations. Its real HTML contains source classes such as `hw`, `ex`, `ind`, `idm`, and `trans`; validate their meaning across committed captures before authorizing each class. Unknown classes inherit only a validated language-bearing ancestor; otherwise neutral.
- Preserve ordered source spans through HTML-to-record text normalization. Verify that the normalized HTML leaf stream agrees with the record Text before trusting its ranges. Any unproven record/field alignment stays neutral without dropping content.
- The existing `ParseEntry`/`Render` continues to own formatting. Project source ranges onto the exact parsed fragments at their source occurrence; do not reimplement a second dictionary renderer, search by word spelling, or indiscriminately mark all repeated matches. Add source offsets where needed at extraction boundaries; transformed/generated text is neutral unless its mapping is proven.
- Bound annotation input to the existing source size limits, and bound projection work to linear walks over source and rendered fragments. If an existing field loses provenance, preserve content neutrally and cover the fallback. The corpus guard must demonstrate that real mixed examples such as `subir a la red` are tinted and their English translation is not; making everything neutral does not satisfy the feature.
- Keep current region ownership: English supplement must not gain target-deck word actions. Language spans do not create click regions.

### Practice

| Surface | Target-owned | English-owned | Neutral/preserved |
|---|---|---|---|
| Choice prompt | Headword, option glosses | Option.Help | Number/key prefixes and spacing |
| Cloze prompt | Blanked stem and option words | Translated help | Blank marker and key/number prefixes |
| Choice/Cloze reveal | Restored sentence and choice/gloss content | “you chose” | Embedded definition retains its pre-rendered tint |
| Board grid | Truncated word text | — | Keys/padding; answer-styled word suppresses tint |
| Board gloss panel | Word and gloss | — | Separators |
| Board help panel | — | Translated help | Padding |
| Sitting chrome | — | Natural-language instructions/status | Key glyphs, counters and separators |

`PromptPresentation`/`RevealPresentation` expose exact emitted byte ranges, and existing Prompt/Reveal delegate to them. HelpLines remains derived from the same help emissions for existing callers. Record board ranges after truncation; never rebuild its geometry outside the form. Main maps roles to codes, styles once, and reuses existing region generation/held reveal ownership. `boardFooter` is an explicit consumer because it bypasses `writeHelped` today.

### Model annotations and streaming

Request short nonnested passages using reserved markers `[lang=es]…[/lang]`; language is a normalized two-letter code or `und` for unknown. Untagged prose is permitted and neutral. Use marker boundaries at actual language changes, including inline examples. The model gets effective `/lang` as context; this corrects the hardcoded English context, without introducing new proficiency inference or automatic bilingual proportion choices.

`languageDecoder` is a pure incremental state machine. It emits clean text plus ownership only after a segment closes correctly. Its writer adapter handles IO and first-error poisoning.

| Input/termination | Result |
|---|---|
| Valid closed segment | Strip markers, emit its prose with validated ownership |
| Untagged prose or unknown code | Emit neutral prose |
| Nested/mismatched markers | Invalidate the current segment, remove recognized reserved markers, emit its readable body neutrally |
| EOF, cancellation or truncation in an open segment | Flush held prose neutrally; remove recognized marker syntax and incomplete reserved-marker tail |
| Segment exceeds 16 KiB | Release held prose neutrally and stream neutrally until its closing marker; never grow the buffer further |
| Malformed reserved header | Remove bounded recognized marker syntax, treat following prose neutrally; a header cap of 64 bytes prevents an unbounded token |
| Literal marker examples | Model must spell reserved brackets as entities; decode those only as literal prose after marker parsing, never recursively |
| Model-supplied controls | Pass through `answerTextFilter` before marker decoding: retain newline/tab and printable Unicode; discard complete control sequences and their payloads before display/storage |

### Authoritative annotation transition model

`stepLanguageDecode(state, event)` owns all changes of annotation state and emits append/flush effects; the byte lexer only produces events. Legal states are `neutral`, `segment` (bounded held body and validated/unknown language), and `recovery` (no held body, neutral streaming). No separate depth counter exists: recovery resynchronizes at the **first** closing marker.

| Event | neutral | segment | recovery |
|---|---|---|---|
| `text` | Emit neutral | Append held body; overflow raises `limit` | Emit neutral |
| `open(code)` | Enter segment with validated code or unknown | Flush held body neutral, discard new opener, enter recovery | Discard opener; remain recovery |
| `close` | Discard orphan marker | Emit held body with ownership; clear and enter neutral | Discard marker, enter neutral |
| `badHeader` | Enter recovery | Flush held body neutral; enter recovery | Stay recovery |
| `limit` | Reject/no state or output change | Flush held body neutral; enter recovery | Reject/no state or output change |
| `finish` | Remain neutral | Flush held body neutral; clear and enter neutral | Enter neutral |

A later closing marker from an invalid outer segment is an orphan and is discarded. After recovery's first close, a new opening marker starts an independent segment. EOF is the only other way out of recovery; newline does not resynchronize it. Subsequent openers during recovery never re-enable tint or allocate a body. `limit` in non-segment states is a rejected internal event, covered by generated transition sequences. Chunking cannot change emitted text or ownership. The lexer retains only a bounded candidate marker (64 bytes); a completed reserved opener/closer becomes one event. At EOF, an unfinished candidate beginning `[lang` or `[/lang` is removed as reserved syntax; shorter ordinary bracket prefixes are emitted as text before `finish`.

Define exact recovery for overlong headers in tests: after 64 bytes discard the reserved header prefix and resume neutral text; no unbounded “wait for closing bracket” state. Ordinary bracketed prose that does not start a reserved marker is preserved. Buffer at most one incomplete UTF-8 rune across deltas; normalization and terminal-control filtering must be independent of chunking. Replace invalid UTF-8 with the replacement character under one documented policy; do not let entity decoding reintroduce controls.

`answerTextFilter` must not reuse `scanEscape` as a complete sanitizer: that helper only recognizes CSI and two-byte escapes. Use a small pure incremental automaton for ESC/CSI, OSC, DCS, SOS, PM and APC, including C1 introducers/terminators. OSC ends at BEL or ST; the other string controls end at ST. Discard control bytes/payloads immediately while retaining only state (and a possible ESC before ST), never accumulating the payload. CSI consumes parameters/intermediates through its final byte; malformed CSI abandons its control state at the first non-grammar character and reprocesses that character as prose. EOF/cancellation discards incomplete control state and UTF-8 follows the documented replacement policy. Thus an unterminated control string stays a discarded payload through EOF without growing memory; readable prose after a valid terminator always resumes. Apply the same control filter to decoded literal entities before they reach spans. This filter owns untrusted answer controls only; it does not change the renderer's trusted ANSI grammar.

`runAsk` feeds deltas through the control filter and decoder before *both* transcript accumulation and display. The display adapter composes tint with vocabulary foreground and `answerWrapWriter`; it flushes the highlight tail before ownership changes so delayed words cannot acquire the next span's language. At termination finish the control filter, then flush decoder, highlight and wrap in that order, then store the resulting clean prose. Preserve cancellation, partial-answer history, truncation, and downstream write-error behavior on all existing paths. Annotation parsing is presentation-domain code under `cmd/define`, not transport code under `internal/llm`.

## Style composition

The shared composer changes only background. Reuse existing escape scanning and `sgrState` semantics; reset/reopen the language background after full SGR resets, preserve vocabulary foreground, and suspend tint around explicit semantic answer styles. Close the background with SGR 49 at boundaries and before newline/erase/indent. Selection continues to reapply inverse video; tests must verify the selected cells are distinct and copying is unchanged. All geometry sees clean text plus ANSI, never model markers.

The policy is passed as data into RenderOpts and practice/model adapters; pure renderers do not inspect environment or session deps. Audit every RenderOpts construction and all prompt/reveal/board paths. No default zero-valued policy may silently tint legacy isolated render tests.

## Implementation tasks

- [x] Implement shared ownership validation and tint composition, invocation policy and field/source provenance, with pure regressions and real dictionary corpus coverage.
- [x] Emit practice ownership during existing layout walks and route prompt, board/footer, help and reveal through the shared composer while retaining no-import purity and answer/selection behavior.
- [x] Implement bounded annotation decoding and the single answer-stream adapter, update effective-language model context, obtain a real annotated capture, and verify stateful fake plus live conformance.
- [x] Update README/atlas, demonstrate dark/light/disabled output, complete verification, commit and pass the single SDLC close review before PR and merge.

## Function-level verification strategy

| Surface | Adversarial strategy | Independent guard |
|---|---|---|
| `validateLanguageText`, `dictionaryLanguageText`, `projectSourceLanguages` | Repeated bilingual spellings, unmatched HTML/text, transformed fields, unknown classes | Literal source-span ownership on committed real records; exact plain-text preservation; ambiguous matches neutral |
| `styleLanguageText` | Nested foreground/reset sequences, whitespace, marked answers, malformed ranges | Exact expected SGR transitions; unchanged visible bytes/columns and no background outside matching text |
| `promptBuilder` emission methods; Choice/Cloze/Board `PromptPresentation` and `RevealPresentation` | Narrow truncation, help toggles, wrong/right/selected answers, absent definitions | Exact emitted range slices/roles, unchanged keys/grade/region coordinates; no imports added to play |
| `answerTextFilter` | CSI/OSC/DCS and other string controls, missing terminators, C1 and malformed UTF-8 | Chunk-independent sanitized prose, constant control-state memory, no control payload in either display or transcript |
| `stepLanguageDecode`, decoder lexer/write/finish | Generated state/event sequences (including rejected events), every byte split, nested/incomplete headers, Unicode/entities, oversize/control-bearing input | Chunk-independent clean text and ownership, bounded pending memory, no emitted metadata/control injection; fuzz progress and preservation |
| `runAsk` | Stateful SSE fake with real annotated capture, interrupted/failed/truncated streams, output failures | Clean visible/stored partial answer agreement; correct request language; no metadata in transcript or geometry |
| Real loop/screen integration | Switch language, copy tinted content, resize, prompt/board/footer/reveal interactions | Literal styles/physical rows and copied prose; #62 prompt indicator agrees with new output |
| Native dictionary/model/PTY seams | Production extraction/requests and actual raw lifecycle | Captures match class semantics; model emits validated mixed spans; no-color/pipes and explicit off remain clean |

Use existing `internal/llm/llmtest` stateful transport; semantic streaming payloads come from committed real captures, not invented SSE literals. Pure decoder malformed-input tests are synthetic protocol tests and remain separate. Follow the existing practice-help conformance pattern, use strict mode when configured, and update the atlas integration table for each new conformance file. Live annotation conformance must exercise the production prompt and decoder; mechanical syntax checks cannot prove language identification, so also inspect captured bilingual phrases and record that evidence.

Verification: `go test ./... -count=1`; focused race tests for language/ask/practice/selection; bounded decoder fuzzing; `go vet ./...`; `GOOS=linux CGO_ENABLED=0 go build ./...`; strict relevant dictionary/model/PTY conformance; `git diff --check`. Mutations must detect whole-supplement tinting, forgotten board/footer ownership, reset-erased tint, parsing after transcript storage, and lost decoder flush on cancellation. Inspect real dark/light output and record any unavailable manual checks honestly.

## Architecture decisions

- ARCH-DRY: one ownership representation/composer; form text and metadata share their existing builders, source identity remains owned by record selection.
- ARCH-PURE: ownership, projection, decoder and style composition are pure; main alone maps session context, env and IO.
- ARCH-PURPOSE: cover definitions, all current practice presentation paths and inline model text in this issue; #64's adaptive policy remains separate.
- ARCH-MOCK: retain installed-source captures and the stateful SSE fake; live conformance checks the new semantics at the real seam.
- ARCH-CONSTRAINTS: bounded decoder/source projection, linear walks, no new background task or terminal probe.
- ARCH-SECURE: validate language/ranges, remove model control sequences and reserved annotation syntax before display/storage.
- ARCH-ORDER: resolve ownership before styling and persistence; language switches affect subsequent output, flush decoder before storing a partial answer.
- ARCH-FUNERAL: response metadata is transient; no persistent preference or deck schema, new goroutines or resources to leak.

## Revisions

- 2026-09-15: Initial design after #62 merge and #65 claim/start-plan. Inspection found mixed language inside Oxford supplement glosses and a direct board-footer render path; the plan preserves their actual producer boundaries. Theme preference requested asynchronously; draft defaults to the existing dark-palette environment with explicit light/off options. Estimate intentionally deferred until the plan-quality gate passes.

- 2026-09-15: Fresh design review found existing `scanEscape` is insufficient for untrusted OSC/DCS and incomplete controls. Added an explicit incremental answer-text filter with constant control-state memory, byte-split/termination contracts and display/transcript guards; no runtime code changed.

- 2026-09-15: Fresh reviewer re-read the corrected plan and approved it with no remaining Important gaps. Runtime implementation and estimate remain pending operator approval and change-code gate.

- 2026-09-15: Operator approved the complete implementation plan. Proceed through plan-quality and estimate gates, then implement without further design approval unless scope materially changes.

- 2026-09-15: Plan-quality PQ-1 requested explicit recovery states, transition ownership and resynchronization. Added the three-state `stepLanguageDecode` model with first-close recovery and rejected-event verification. Addressed the strategy-format minor by naming production functions and removing repeated control-test prose. Renamed the plan to match the issue basename so gate discovery supplies it directly. Approved feature scope is unchanged.

- 2026-09-15: Implementation checkpoint: plan-quality and estimate passed; shared rendering, source metadata, practice presentations and model adapter are implemented. Normalized Core concepts table locations to one owning file per row for the repository guard. Integrated tests exposed tint across physical wrap newlines and a dictionary wrapper hiding supplemental capability; both receive root-cause fixes and regressions within approved scope. Documentation and final verification remain in progress.

- 2026-09-15: Implementation and full verification passed; mutation checks rejected all five planned regression classes. Both per-session profiles retained after operator confirmed use of light and dark terminals. Final row remains pending close review/publication. See issue Log for exact commands, live capture evidence and manual visual limitation.

- 2026-09-15: Close review BR-1 (`source-ownership-requires-provenance`) found that `/lang` can differ from a dictionary's actual source when native selection falls back to every active dictionary. Source ownership now comes only from verified monolingual metadata for the actual selected IDs, preserved through wrappers and carried by definition sections. Study language remains solely the tint comparison target. Enumerated consumers: ordinary lookup, Choice full reveal, Cloze full reveal, plus dictionary-derived Choice option/reveal glosses and Board panel/footer glosses. Unknown fallback remains neutral in every consumer; deck-authored words/cloze text retain Target ownership and English help retains English ownership. Added missing-source and known-other-source integration regressions for all three full-definition paths, and practice-source regressions. No change to fallback lookup behavior or #64 policy.

- 2026-09-15: Close re-review returned SHIP with no new findings; BR-1 disposed as addressed. Implementation checklist complete, with PR/merge publication following the accepted boundary. Reviewer independently reran the package suite/vet and rejected a provenance-regression mutation.
