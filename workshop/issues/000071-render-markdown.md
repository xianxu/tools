---
id: 000071
status: open
deps: [#70]
github_issue:
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
---

# define: render markdown — in pasted input and in model output

## Problem

Markdown arrives on both sides of the session and define renders neither side.

**Model output.** Everything the model streams — `/ask`, the bilingual gloss,
reflect's learner model, harvest's rationales — is markdown, because that is what
models write: `**bold**` around the word under discussion, backticks around a
term, `-` bullets, `##` headings. This repo actively primes it: the prompts in
`harvest_item.go` and `harvest_judge.go` are themselves written in markdown
(`**band** — the CEFR level …`), so the model answers in the register it was
asked in. The pipeline that carries the answer — `answerTextFilter` →
`highlightWriter` → `answerWrapWriter` — strips terminal controls, matches deck
words and wraps, and passes the markup through untouched. The reader sees the
asterisks.

**User input.** #67 lets the reader paste a passage and click the words they do
not know. Text pasted from a web page, a doc, or a chat with a model is routinely
markdown. `newPassage` wraps it and `wordRuns` tokenises it, so `**climate**`
becomes a run whose word is `**climate`, the `**` occupies four clickable cells
that answer to nothing, and `CaptureMarked` would file `**climate` into the deck.

## Spec

### Scope: a bounded inline subset, not CommonMark

Ten roles, chosen because they are what models actually emit and what people
actually paste. Explicitly OUT: tables, reference links, nested list semantics
beyond indentation, raw HTML, footnotes, setext headings. "Render markdown" is
the kind of scope that grows a parser; the boundary is written here so the review
can hold it.

`**bold**`/`__bold__` · `*italic*`/`_italic_` · `` `code` `` · fenced code blocks ·
ATX headings `#`–`###` · `-`/`*`/`+` and `1.` list markers · `[text](url)` ·
`> quote` · `~~strike~~` · `---` rule.

### The colour scheme

Two palettes, selected by #70's theme. The `48;5;236` / `48;5;254` pair is
`languageDark` / `languageLight` reused rather than a third set of greys
(ARCH-DRY).

| role | dark | light |
| --- | --- | --- |
| heading (`#`–`###`, marks dropped) | `1;38;5;75` | `1;38;5;25` |
| bold | `1` | `1` |
| italic | `3` | `3` |
| strike | `9` | `9` |
| code span | `38;5;215` on `48;5;236` | `38;5;130` on `48;5;254` |
| code block (fence dropped) | `38;5;215` on `48;5;235` | `38;5;130` on `48;5;255` |
| list marker (drawn as `•`) | `38;5;245` | `38;5;242` |
| link text | `4;38;5;75` | `4;38;5;25` |
| link URL | `2` | `2` |
| quote text / `│` gutter | `3;38;5;245` / `38;5;240` | `3;38;5;242` / `38;5;250` |
| rule (drawn as `─` to width) | `2` | `2` |

**Bold, italic and strike carry no colour, on purpose.** They are the three roles
that wrap arbitrary prose, and prose is where deck highlighting lives:
`highlight.go`'s `knownOn` is `1;32m`, and colour is the channel highlighting
owns. A bold span that also set a foreground would either lose the green on a
known word or fight it. Weight, slant and strike compose with colour; colour does
not compose with itself.

### Where it goes in the pipeline (ARCH-ORDER)

`answerTextFilter` → **markdown** → `highlightWriter` → `answerWrapWriter`.

- After `answerTextFilter`, so the model cannot reach the terminal through
  markdown: by the time this code runs, its input is already control-free
  (ARCH-SECURE — the only escapes in the output are ones this renderer emits,
  from the closed table above).
- Before `highlightWriter`, so a deck word inside `**bold**` gets its green
  injected into styled text and the bold handed back afterwards. That is exactly
  what `sgrState` exists for ("ANSI does not nest… the writer has to re-open what
  was in effect", `sgr.go:8`) and this is its second caller, not a new mechanism.
- Before `answerWrapWriter`, so wrapping measures what a reader sees. Rendering
  removes the `**`, which changes the display width of the line; the wrapper
  already counts cells rather than bytes (`render.go:497`) and needs no change,
  but only if the markup is gone before it counts.

The ordering is the design, so it gets a test that asserts the composition — a
known deck word inside a bold span inside a wrapped line — not three tests that
each check one stage.

### Streaming

`**` splits across chunks, and so does a fence. The renderer holds an unresolved
opener the way `answerWrapWriter` already holds an unfinished word, with the same
bounded pending buffer (`maxAnswerWrapPending` / `maxAnswerWrapEscape` are the
precedent — unbounded lookahead for a closer that never arrives is a hang).
`Flush` emits any unterminated opener as literal text, on every way a request can
end. A model that writes a single `*` must not swallow the rest of the answer.

### Colour off: pass it through untouched

With `-no-color` or a pipe, the markup is left exactly as written. `lessons.md`'s
"Ephemeral UI vs. a record" is the rule: piped output is a record, and
`define /ask … > notes.md` should yield the markdown the model wrote. Stripping
markup from a record destroys information the reader may want; styling it is
impossible. So the rule is *render* or *pass through*, never *strip*.

### The paste side

`visibleIndex` is the hinge. `deckSpans` already runs the matcher over plain text
and maps offsets back through a column table (`deckwords.go:46`), and
`passage.go:58` says the passage's column table is built with "the same
escape-aware walk the rest of the program uses". If markdown is rendered into
SGR-carrying text before `newPassage` sees it, the passage may already be correct
— the words it tokenises are the plain words, and the cells it maps are the
visible cells. That is a claim to VERIFY, not to assume: the acceptance test is a
pasted `**climate** change` where clicking either word marks that word and
`CaptureMarked` files `climate`, not `**climate`.

### Pure core (ARCH-PURE)

The renderer is a pure function from a text chunk plus carried state to emitted
text plus new state — no writer, no terminal, no palette lookup of its own. It
takes the palette as a value. That is what makes the ten roles unit-testable as a
table and the streaming behaviour testable by feeding the same input in every
chunk split.

## Done when

- Each of the ten roles renders under both themes, driven by a table test that
  enumerates the roles — the enumeration is the test, so a role added later
  without a row fails rather than passing silently.
- A deck word inside `**bold**` inside a line that wraps comes out green, bold,
  and wrapped at the right column, in one composition test.
- Feeding the same answer in every possible chunk split produces byte-identical
  output, including splits inside `**`, inside a fence, and inside a link.
- An unterminated `**`, an unterminated fence, and a lone `*` all flush as
  literal text with no content lost — asserted by COUNT, not containment
  (`lessons.md`: a subsequence check passes on text that inserts characters).
- With `-no-color`, `define` output is byte-identical to the model's text; no
  markdown is stripped from a record.
- Pasting `**climate** change` into a passage (#67) makes two clickable words,
  marks the one clicked, and files `climate` into the deck.
- The model cannot emit a terminal escape through markdown — a fuzz or property
  test over hostile input asserting the output contains only escapes from the
  palette.
- `atlas/define.md` records the markdown stage and its position in the pipeline.

## Plan

- [ ] brainstorm: confirm the ten-role boundary and the render-or-pass-through
      rule; decide whether headings keep a blank line
- [ ] `sdlc start-plan`, then the durable plan in `workshop/plans/`
- [ ] the pure renderer + its role table, against a palette passed in
- [ ] streaming: the bounded pending opener, `Flush`, the chunk-split test
- [ ] wire it into the answer pipeline between the filter and the highlighter;
      the composition test
- [ ] the paste side: verify or fix `newPassage`/`wordRuns`/`CaptureMarked`
- [ ] atlas, then `sdlc close`

## Log

### 2026-09-16

Filed with #70 (the light/dark theme switch), which owns the axis this scheme's
two columns are selected by; the colour table above is the proposal, and #70's
palette is where it lands.

Cross-reference: #72 (stream the answer as it arrives) sets a bound on how long
any stage may hold text back. An inline markdown span whose closer has not
arrived is exactly that kind of hold, so this renderer's streaming behaviour is
constrained by #72's rule rather than free to choose its own.
