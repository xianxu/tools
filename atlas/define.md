# define

`define <word>` prints the New Oxford American Dictionary entry for a word and
plays its recorded pronunciation.

NOAD is the dictionary Google licenses for its US definition panel, and macOS
bundles it — which is why `define sycophantic` shows `/ˌsikəˈfan(t)ik/`,
character-for-character what the browser shows. That notation is **not** standard
IPA: NOAD writes `i` for /ɪ/, `a` for /æ/, and `(t)` for the optional flap.

**The lookup is not NOAD-only.** `DCSCopyTextDefinition` takes a
`DCSDictionaryRef`, and the SDK exports no way to construct one — so the tool
passes NULL, which means *search every active dictionary*. NOAD answers for
ordinary English words; `iPhone` and `MacBook` come from Apple Dictionary (which
is why they have no pronunciation), and with the Chinese dictionaries enabled
some words return Han-script entries this parser does not model. What `define`
shows depends on the host's Dictionary.app configuration — and so does whether
the conformance tests pass.

## Shape

A pure core with three thin IO seams. Everything lives in `cmd/define/` as
`package main` — `internal/` is earned on the second consumer (`AGENTS.local.md`),
and there is no second consumer yet.

| Seam | Wraps | Fake | Status |
|---|---|---|---|
| `Dictionary` | CoreServices `DCSCopyTextDefinition` (cgo) | `fakeDictionary` over the captured corpus |
| `AudioSource` | Google's gstatic MP3 CDN | `fakeCDN`, an `httptest` server recording request order |
| `Player` | `afplay(1)` | `fakePlayer`, recording play count |
| `deps.newLLM` + `getenv` | `internal/llm` (the model) | `llmtest.Fake`, an httptest server on the wire |
| `deps.notifySignals` | `signal.Notify` | a channel a test writes to |
| `--reflect` | the model, batch | `llmtest.Fake` + a live conformance check |

Pure: `ParseEntry` (flat text → `Entry`), `Render` (`Entry` → string),
`AudioCandidates` (word → ordered URLs), `isPronunciation`, `opensBlock`,
`rewritePronunciations`. None touch IO, so their tests need no mocks.

Every fake lives in a `_test.go` file, so none links into the shipped binary.

`dict_darwin.go` holds the only cgo. `dict_stub.go` (`//go:build !darwin`) keeps
`go build ./...` and `go vet ./...` green off macOS.

## Three parsing rules worth knowing

NOAD returns flat text with no schema. Two shapes are non-obvious and both came
out of reading real output:

1. **The head can carry a part-of-speech.** `record` returns
   `record rec·ordnoun | ˈrekərd | …` — the POS is welded onto the
   syllabification, ahead of the pronunciation. It opens the first block
   (`Entry.HeadPOS` / `Block.FromHead`) but renders in the head, because that is
   where NOAD puts it.
2. **The head has no fixed field order.** `present 1 pres·ent` puts the
   homograph first; `record rec·ordnoun` welds the POS on; `read verb (past and
   past participle read | red |)` carries a whole parenthetical, including a
   pronunciation that is *not* the entry's. So the head is kept as an ordered
   `[]HeadTok` and `Render` walks it — order is carried by the data, never
   re-guessed by the renderer.
3. **`|` is overloaded.** It delimits pronunciations *and* separates examples.
   The discriminator is word shape, not character class — `ˈrekərd` and `baNGk`
   are mostly ASCII letters, so "contains no ASCII letters" fails. A span is a
   pronunciation iff every comma-separated part is **either** a single space-free
   token **or** a short multi-word run carrying a NOAD stress mark
   (`isPronunciation`). The single-token-only rule was tried and was wrong:
   multi-word headwords have spaces in their pronunciations (`hot dog | ˈhät ˌdäɡ |`),
   and rejecting them made the parser walk on and adopt a derivative's —
   `define "hot dog"` showed `/ˈhätˌdäɡər/`. Prose carries no stress marks, which
   is what keeps the second clause safe.

## The invariant

Every letter and digit of the raw entry must appear **in order** in the rendered
output. Punctuation may be restructured; words may not vanish or move.

The property is checked at three widths, and it needs all three — the boundary
reviews found bugs that each narrower check had shipped green:

| check | scope | catches |
|---|---|---|
| `TestRenderLosesNothing` | the captured fixtures under `testdata/entries/<lang>/` | regressions on known shapes |
| `FuzzRenderLosesNothing` | arbitrary strings, corpus-seeded | parser crashes, boundary bugs |
| `TestRenderLosesNothingOverLiveEntries` (conformance) | **every** reachable entry | shapes nobody thought to sample |
| `TestClassifyRawNotationOverRealEntries` | one captured exemplar per raw-notation cause | the taxonomy, without needing the live dictionary |

The third is what earns the claim "safe against entries nobody sampled"; a corpus
test alone covers only what someone already sampled. Content loss over the live
sweep is **0%**, over every reachable entry rather than a sample.

**Measured 2026-08-28: 70,886 entries reachable, 0 non-Latin, ~116s.** Those are
DATED rather than stated as standing facts, and the distinction is the point —
each is a property of one host's installed dictionaries on one day, and every
one of them was wrong in this file until `#26` re-ran them. A number that only a
live run can produce is a record; a number the code owns is consumed instead (see
the raw-notation count below). Anything written as neither drifts silently.

The non-Latin count was 530 before `#23 M2` and is 0 after it: the sweep used to
search every ACTIVE dictionary, and now asks the curated English books. The
counting branch is kept anyway, because adding a book to the curated list can
bring the shape back.

Sampling is why this section had to be rewritten three times: at 2,749 entries
(3.8%) the raw-notation count read 0, and at full width it was 27 — both
historical readings, not current ones. The current count is derived below.

### Two things the property cannot see

**Order-preserving corruption.** A phantom `adjective` block on `bank`, or a
sense numbered from "the 200 meters", keeps every letter in order. The structural
goldens in `render_test.go` cover that class.

**Its own oracle.** The companion check — "did raw NOAD notation survive
rendering?" — was originally written as *scan the output for `|…|` spans and ask
`isPronunciation` whether each is a pronunciation*. That asks the function under
test to grade its own output, so it can detect false positives only: every span
`isPronunciation` wrongly rejected was reported as "not a pronunciation" and
passed. It read **0% while 2.2% of live entries were displaying raw pipes**, and
that false 0% was published here.

The check now uses **two** oracles, neither of which consults the parser:

- `strayStress` — a NOAD stress mark (`ˈ`/`ˌ`) may appear only inside a `/…/`
  span in rendered output.
- a bare `strings.IndexByte(out, '|')` — NOAD's delimiter has no place in
  rendered output at all.

The second exists because the first was still too narrow: example-separator
pipes carry no stress mark, so `strayStress` could not see them, and the atlas
published "0%" while 2.0% of entries were still rendering raw `|`. Neither reads
zero: together they trip on the pinned handful counted below, which is why that
count exists. (An earlier version of this line said "both now measure 0", stated
undated, twenty lines above the count that falsifies it.) The lesson generalises
past the first fix —
*an honest oracle can still be a narrow one, and a claim must not outrun what was
actually measured.*

It guarantees **fidelity, not completeness** — see Limits.

## Limits

- **Only one homograph is reachable.** `DCSCopyTextDefinition("bank")` returns
  `bank 1` (the riverbank) and stops; the financial sense is not in the response.
  `define` renders the homograph number so the truncation is visible.
- **NOAD has gaps.** Recent coinages (`rizz`, `unalive`) are absent; `define`
  exits 1. Their audio is missing too — the gaps correlate, both tracing to Oxford.
- **No second dictionary source**, by decision.
- **Raw NOAD notation survives on a pinned handful of entries — FOUR causes, not
  one.** The ratchet is `knownRawByCause` (`cmd/define/rawnotation_test.go`),
  which pins the population per cause; `TestAtlasQuotesTheRawNotationCount` keeps
  the number below in step with it, because this one has drifted three times.

  <!-- raw-notation-count -->26<!-- /raw-notation-count --> entries, as
  *prose numeral* <!-- raw:prose-numeral -->7<!-- /raw:prose-numeral -->,
  *headword pronunciation* <!-- raw:headword-pronunciation -->7<!-- /raw:headword-pronunciation -->,
  *phrase pronunciation* <!-- raw:phrase-pronunciation -->8<!-- /raw:phrase-pronunciation -->,
  *literal pipe* <!-- raw:literal-pipe -->4<!-- /raw:literal-pipe -->.

  A total alone is a weak ratchet — two causes can move opposite ways and leave
  it unchanged — so the breakdown is what makes a movement attributable. Each
  number above is DERIVED: the doc-sync loops over every cause, because marking
  only the total let exactly that compensating swap pass green.

  Only the first is a parser defect in the sense below. The *literal pipe* four
  (`pipe`, `piped`, `pipeful`, `pipeless`) are a false positive the oracle keeps
  deliberately: those entries define the `|` character, so their content contains
  one. Narrowing the oracle to exclude them is the move that already failed here
  — see the sampling note above.

- **A prose numeral that continues a sense sequence is taken as a sense number.**
  <!-- raw:prose-numeral -->7<!-- /raw:prose-numeral --> of those entries. `define charge` buries its real sense 2 inside a
  quoted example, and two raw `|` reach the screen; `just`, `depth` and
  `shortness` are the same shape. (`ratio`, `glop` and `logarithmic` were named
  here too and no longer are: they render CLEAN now, which is not the same as
  being unreachable — all three still resolve.) A sequence-opening
  `1` must be structurally placed, but a continuing number is exempt — and that
  exemption is the defect. It stands because requiring placement for every number
  regresses senses NOAD genuinely writes unplaced (`bases`: "plural form of
  base1 2 …", and likewise `absolute`, `ambrosia`, `bind`). Pinned by the live
  ratchet rather than left to drift.
- **Some block boundaries are genuinely ambiguous in the source, and this is not
  rare.** A part-of-speech opens a block when it follows a sentence end or a
  closing `. ) : ; ]`, but NOAD does not always write one. Two shapes remain,
  **measured 2026-08-27 over 71,427 reachable entries** — a different width from
  the 70,886 dated above, because that sweep predates `#23 M2` narrowing the
  dictionary set. Dated rather than restated, so the two are visibly different
  measurements rather than one of them being stale:
  - **~32 entries lose a block into a quoted example** — a register label sits
    where punctuation would be (`shuttle`, `chloroform`: "mainly British English
    verb …"). `parrot` is the no-punctuation variant ("…and budgerigars verb…").
  - **up to ~600 entries render a first block with no part-of-speech heading**,
    its gloss opening "mainly British English verb [with object] …"
    (`backheel`, `Barmecide`).

  Loosening `opensBlock` further reintroduces the phantom blocks it exists to
  prevent, so this is accepted for now rather than fixed. A mirror of
  `isGrammarLabelOnly` for short leading register labels is the tractable fix if
  it becomes worth doing.

## The line editor (raw mode)

When `define` owns the terminal (**stdin and stdout both a tty, and not
`-no-color`**) it enters raw mode, takes the alternate screen (see "The screen"
below) and runs its own editor:

- Up/Down walk history newest-first; with text typed they walk only entries with
  that prefix (zsh's `history-beginning-search-backward`).
- The newest matching entry appears ahead of the cursor in grey; **Right, End or
  Tab** accepts it. **Enter submits only what was typed** — and redraws the line
  without the grey tail before advancing, since an unaccepted suggestion in
  scrollback claims the user typed something they did not.
- **Cmd+Delete clears the line.** Terminals send it as Ctrl-U (`\x15`) — Ghostty
  binds `super+backspace` to exactly that — so the byte is handled rather than a
  key code no terminal actually emits.
- Both loops route submissions through the same `parseREPLLine`, so the
  interactive and piped paths cannot disagree about what a line means.
- Paragraphs wrap at word boundaries to the terminal width, measured in **visible
  columns** because every line here is coloured. Width 0 (a pipe) disables it:
  a consumer re-wraps for itself and baked-in breaks cannot be undone.

The editor is a pure state machine — `Apply(Editor, Key, candidates) → (Editor,
Action)` plus `RenderLine` — so every behaviour above is a table test over key
sequences with no terminal. `Action` is what the *loop* must do; the editor never
acts. Candidates arrive as plain slices rather than a `History` handle, so
**`Apply` never queries**; the loop resolves them once per keystroke.

`candidates` carries TWO lists, and which one each key reads is part of the
model: Up/Down walk `recall`, Tab/Right/End accept from `complete`. They were one
slice until #20 made completions stop being past lines — see "Type-ahead" below
for why sharing one became unsafe.

**Cancellation changes shape in raw mode, and this is the subtle part.** Raw mode
clears ISIG, so Ctrl-C arrives as byte `0x03` and the key reader can act on it
even while the loop is blocked in playback. It does not *own* the meaning,
though, and the three claims this paragraph used to make were each disproved:
the pty suite measured a `\x03` arriving as a **SIGINT** anyway; the loop no
longer relies on `NotifyContext` at all (`repl` detaches with
`context.WithoutCancel` above its choice of loop, so both loops are served); and
the reader fires an **interrupt sink** rather than a cancel. See "Free-form
input" below for what the sink is and why both transports feed it.

That forced a second decision — **render cooked, play raw** — which `#30` then
DISSOLVED, and this paragraph is the history rather than the current design.
Printing a definition needed cooked mode so its newlines translated, while
playback had to stay raw so the key reader kept seeing bytes; doing the whole
lookup cooked made Ctrl-C during playback hang, verified, fixed, and pinned by
`TestPTYCtrlCDuringPlaybackExitsPromptly`. That test still stands. What is gone
is the flapping: inside a screen the app places every line itself, so no output
depends on the line discipline and raw mode is simply continuous. The rule was
satisfied by removing the thing it constrained. `workshop/lessons.md` keeps it,
because it is still true of any program that flaps modes around a blocking call.

`#2`'s `eraseLineAndStepBack` and `skipPrompt` are **deleted, not ported**: they
existed to step back over the terminal's echo of Enter, and raw mode does not
echo. That also removes `#2`'s documented limitation that typing during playback
stranded the indicator — the arithmetic has nothing left to correct for.

## The screen

`#30` made the interactive loop a full-screen program, and the reason is
COORDINATES: clicking a rendered token means knowing which line a screen row is,
and that is only knowable if nothing but this program can move the view.

**The alternate screen answers it by construction.** It has no scrollback, so
there is nothing above the viewport for the terminal to show and no offset the
app does not own. Terminals that offer to scroll it do so by SENDING KEYS, which
is still the application's own scroll.

```
screen        lines []string + offset      PURE: Write, Frame, Scroll, Page, Paint
liveScreen    screen + tty + rows,cols     the only part that does terminal IO
display       Draw(prompt, menu), Page,    what the editor loop draws on
              Scroll, Resize
console       display + resizes + finish   THE TERMINAL, as one parameter
              + stdout + stderr
handBack      Stop → restore → transcript  the exit sequence, as one function
```

`runEditor`'s doc comment always said "the editor loop with the terminal factored
out"; `console` is that concept given a type, and it arrived when five of the
function's ten parameters turned out to be the same one. Two of those five were
adjacent `io.Writer`s, so a call that swapped stdout and stderr compiled and put
diagnostics where the definition goes. In production every field is the same
`liveScreen`; a click is a method on `display`, not an eleventh parameter.

- **`screen` is an `io.Writer`, and that is what kept this from being a rewrite.**
  `Render` returns a string, the ask path streams, commands and the indicator
  print — every one of them feeds the buffer unchanged. It REPLACED the
  translating writer this path used to wrap stdout in, because two owners of line
  endings is how they drift. `#41` made `--play` the second consumer, which left
  that writer with no caller at all — `crlf.go` is DELETED, not retained. The
  piped path never needed it: `replLines` runs cooked, where the terminal
  translates.
- **A write REPAINTS.** The buffer alone is invisible, and a streamed answer
  arrives token by token while `♫ playing 3×` has to show during playback that
  blocks for seconds. So `liveScreen.Write` paints, and `screen` stays pure and
  unit-tested with no pty anywhere.
- **The prompt and the command menu are NOT buffer lines.** They are the live
  edge, rewritten per keystroke; buffering them would file a copy of the prompt
  per character typed. They are arguments to a frame — which is also what deleted
  the menu's erase arithmetic and its documented off-by-a-row limit, since a
  whole-frame redraw never counts rows it drew earlier.
- **The prompt belongs to a loop that is WAITING.** Between a submit and the next
  frame the loop is looking up, streaming or playing, and a prompt drawn then
  invites typing at a line nothing is reading — it also repainted the line just
  submitted, so the word appeared twice under its own definition.
- **`eraseLine` (`\r\x1b[K`) is honoured by the buffer**, not stripped: it is the
  ephemeral indicator's own "take that line back". A buffer that ignored it would
  carry `♫ playing 3×` into the exit transcript, which is the "ephemeral UI vs
  record" doctrine failing in the direction it exists to prevent.
- **STDERR routes through the screen too.** A diagnostic written past it would
  land wherever the cursor happens to be and corrupt the frame. It also means
  `define: … no dictionary entry` is part of the record now instead of scrolling
  past. The one-shot and piped paths keep the real stderr.
- **The session is printed back into the normal buffer on exit.** The alt buffer
  is discarded, so without it quitting throws the session away — and `define
  arrondissement` used to leave the entry where you could scroll back to it
  tomorrow. Printed from `finish` AFTER `restore`, when the terminal is cooked
  again, and `finish` is once-only for exactly that reason.

**A frame is budgeted in DISPLAY ROWS, and that is the guarantee — not a
detail.** A line wider than the terminal wraps onto a second row, so a frame that
counted it as one is a frame one row too tall; the terminal scrolls to fit it,
and every row the app believes it placed has moved. Two routine ways in: narrow
the window (buffer lines keep the wrapping they were rendered with, by decision)
or type a line longer than the terminal is wide. So the prompt and each menu row
are charged their real height, the cursor walks back by the rows the terminal
actually moved rather than by menu entries, and buffer lines are CLIPPED to the
width at paint time — the buffer keeps the whole text, so the transcript and
the click map lose nothing.

**Every component is budgeted, and the order of sacrifice is the order of
value.** Charging the live edge its height and then writing it unclipped is not a
budget: a menu taller than the space left overflows exactly as a wide buffer line
did. The prompt survives first — it is the line you are typing, and it is clipped
only when it alone is taller than the terminal, where the alternative is a frame
nobody owns. The menu gives up whole rows next (`fitMenu`, from the end, because
the list is sorted and the first matches are the likely ones). The buffer takes
what is left, because it is the part you can scroll.

**A frame is a PLACEMENT, not a set of substrings**, and the tests read it that
way: `readFrame` interprets what `Paint` emits the way a terminal would —
including the deferred wrap that lets a line clipped to exactly the width still
cost one row — and asserts two properties over shapes. That the frame fits, and
that it leaves the cursor at the end of the prompt. Both were breakable while the
suite was green, which is how the cursor came to walk back over menu ENTRIES
rather than the rows the terminal moved.

**One owner answers "where does this escape sequence end", too.** `scanEscape`
(`sgr.go`) has always known; `escapeLen` wraps it for whole strings, and every
site that walks styled text — measuring, clipping, highlighting — skips through
that rather than re-deriving "ESC, then optional `[`, then parameters, then a
final byte in `0x40`–`0x7E`". Four spellings of one grammar agree right up until
they do not, and `M2.5` splices an underline through the same text that
`clipVisible` cuts.

**One owner answers "how wide is this", and it counts CELLS.** `visibleCells`
(`render.go`) skips escape sequences and reads `cellWidth` per rune: a combining
mark is 0 columns and a CJK or fullwidth rune is 2. Both are this program's daily
traffic — NOAD writes `bänˈZHo͝or` with a combining double breve, and a Japanese
entry is full-width — so counting runes is wrong in both directions, cutting text
that fits and building frames twice as tall as measured. Every wrap, every frame
budget and every clip reads that one function, so a line cannot be measured one
way where it is written and another where it is placed. `terminalSize` is its
counterpart for the terminal: `terminalWidth` answers a POLICY question ("how
wide should text wrap", 0 meaning "do not") while the screen needs a true column
count that cannot be a sentinel.

**Writes are throttled; the trailing flush is not an optimisation.** The ask path
writes once per streamed delta, and a frame per delta is a full-screen redraw per
token. `liveScreen` paints at most every 16 ms, with a timer that flushes a held
frame whether or not another write follows — because `♫ playing 3×` is written
and then playback blocks for seconds, so a throttle that waited for the next
write would hide it for the whole recording. Draw, Page, Scroll and Stop paint
unconditionally. The buffer itself is uncapped, deliberately: a cap would
silently truncate the record the exit transcript exists to be.

## Clickable regions

A rendered entry contains tokens that MEAN something the tool can act on, and
until `#30` they were inert text — the action had to be retyped as a command.
Now: **click the headword to hear it; click the language after `ORIGIN` to hear
it in that language.**

```
Region        {Kind, Text, Word, Lang, Line, Col, Width}   what a span OFFERS
regionsIn     (Entry, rendered, key) -> []Region          PURE, reads the OUTPUT
screen        addRegions / RegionAt / LineAt              the click map
markClickable underline spliced at paint time             the mark
```

**One registry, not two special cases**, which is how the issue was filed: a
third consumer is a row rather than a new feature. `numRegionKinds` is the
registry's extent and every guard derives from it — `TestEveryRegionKindIsActionable`
fails for a kind that draws, invites a click and does nothing.

- **`RegionHeadword`** — play this word's recording. The primary target, because
  every entry has a headword in every language, whereas the IPA is English-only
  (`#31` measured it: Spanish writes none, Italian writes syllabification).
- **`RegionOriginLang`** — play the word in the language its ORIGIN names.

**`RenderOpts` is what a caller decides**, and one of its four fields is not
about how the entry looks:

| field | what it decides |
|---|---|
| `RenderOpts.Color` | whether the palette is emitted at all — `-no-color` makes the output a RECORD, and a record carries no escapes |
| `RenderOpts.Width` | where prose wraps, in display cells. `0` means "do not wrap", which a pipe wants and a terminal under 20 columns also gets |
| `RenderOpts.Vocab` | the deck words to highlight, resolved by `vocabularyFor` so no path can render against an empty set by forgetting to ask |
| `RenderOpts.Word` | the LOOKUP KEY — identity, not presentation. See "a shortcut must not re-derive its target" below; empty means "no click map wanted" |

**A region is read out of the FINISHED output.** A position recorded while
writing describes what `Render` intended; a click map has to be right about what
the terminal shows. It also leaves `Render`'s body untouched, so "the bytes are
identical" is a property of the shape rather than a promise every edit re-earns —
`TestRenderOutputMatchesTheCorpusGolden` holds it, against a golden generated
from the commit BEFORE the signature change.

**Which ORIGIN languages are clickable is the mentions producer's answer**, not a
search of the rendered text. `OriginLanguageMentions` cuts cognate clauses and
masks historical stages, so the "Dutch" in `read`'s etymology is on screen and is
not a source; positions carry across by OCCURRENCE INDEX, because rendering
preserves the text's characters in order. EVERY language named is clickable, not
just the first — `/pron` takes the first by NOAD's convention, while a click has
nothing to disambiguate: `piano` names French and Italian and the user points at
one. That is the insight the whole issue rests on.

**A SHORTCUT MUST NOT RE-DERIVE ITS TARGET**, and this is where that rule is
paid. A click on the headword is a shortcut for the bare Enter beside it, which
replays the session's current word — the LOOKUP KEY. Deriving the target from the
entry instead made the two disagree: `Entry.Headword()` is `fields[0]` alone, so
`hot dog` underlined only "hot" and played it, `a priori` reduced to the letter
"a", and `bargainer` — an inflected form finding its base entry, the common case
— played "bargain".

So the key belongs to the CALLER and travels on `RenderOpts.Word`, which is the
one field there that is identity rather than presentation. The clickable span is
that key where the head line shows it, falling back to the headword token when it
does not (`define jalapeno` finds "jalapeño"): always something on screen, always
the word Enter would play. No rule over the parsed tokens can find the phrase —
`a priori` parses as `[a, priori, a, pri·o·ri]`, the phrase and then the phrase
again syllabified.

`Region.Word` then carries that key onward, because a reader can scroll back and
click a word from earlier in the session while the session keeps only the CURRENT
entry's raw text — which is what supplies the source spellings for a foreign
replay. An older entry replays through `#29`'s fallback on the headword itself:
the degraded answer rather than a wrong one.

**A span with no visible extent is not a span.** An entry that is blank or a
single space parses to an empty headword; asking for that span answered "found,
at column zero" and indexed an empty line, so `define` PANICKED on input a
dictionary can return. `findVisible` refuses it, measured in CELLS rather than
bytes — the fuzzer found a NUL headword, whose width is zero for the same reason
a combining mark's is.

**The actions are `replayInPlace` with one parameter** — the same path a bare
Enter and `/pron` take, so a click cannot drift from the gesture it shortcuts. A
click on ordinary text is NOTHING: no beep, no message. Pointing at a word that
offers nothing is not an error.

**The mark is an attribute, spliced by the SCREEN.** `markClickable` underlines a
clickable span at paint time, turned off with `24` rather than `0` so the
palette's colour survives. Static rather than on hover, because hover needs mode
`1003` — an event per cell the pointer crosses — while `1000` reports presses
only and never says where the pointer is. Emitted by `Render` it would leak into
`define <word>`, a pipe, `-raw` and `> out.txt`: decoration claiming an
affordance a file does not have. `writeRendered` is the seam — a writer that can
hold a click map gets one, everything else gets bytes.

**Degrading is the absence of input, not a code path.** There is no reliable way
to ask a terminal whether it will honour mouse reporting, so the enable is
unconditional and a terminal that ignores it simply never sends a report;
everything else still works. `-no-color` degrades by ROUTING — it clears
`opt.tty`, so the session takes the line loop and no mark can reach output the
user asked to keep plain.

**Scrolling, and why the mouse had to be reported.** PageUp/PageDown move the
viewport by a screenful less one line of overlap; the wheel moves three lines. The
wheel took mouse reporting (`1000` + `1006`) to arrange at all: in the alternate
screen a terminal translates the wheel into ARROW KEYS — the convention that lets
`less` scroll with no mouse support — and this editor binds Up/Down to the history
walk, so scrolling recalled words. The bytes are identical, so nothing can
separate them; asking the terminal to report the mouse is the only way to be
handed the gesture the user made.

**Enabling a mode means accepting its whole grammar.** `1000` is answered in
X10 — `ESC[M` plus three RAW bytes — by any terminal that ignores `1006`, and
those bytes belong to no CSI grammar: the scan stops at `M` as a final byte and
the payload reaches the line as text, so a click typed `" !!"` into the word
being looked up. That is `#14`'s family one encoding over, shipped by the commit
that enabled the mode. `decodeX10Mouse` consumes six bytes or none, `decodeWheel`
handles the SGR form, and both read one `wheelFromButton`. The rule to carry into
`M2`: **for every mode we enable, the decoder answers every encoding that mode
can reply in.**

**The cost, decided rather than discovered:** with tracking on, drag-select
belongs to this program, so copying text needs Option (iTerm2, Terminal.app,
Ghostty) or Shift. `/help` says so, which is where a user meets it. Text
selection of our own is a NON-GOAL — a whole model of anchors, extents and
clipboard integration.

**Terminal state is one guarantee, not three.** Raw mode, the alternate screen
and mouse reporting all hang off `rawSession`, which restores from a defer AND on
the cancellation path. Unwound in reverse: mouse first (a terminal left reporting
it types escape sequences into the next program while the shell still looks
fine), then the alt screen, then the line discipline. `rawSession` writes those
mode sequences to an `io.Writer` rather than to the stdin handle — they change
the screen the frames are drawn on, and it is also what makes the protocol
assertable with no terminal, which the first version of that test was not.

**The exit sequence is one function, `handBack`.** Stop painting, restore, print
the session — in that order, because a frame drawn after the alternate screen is
gone lands on the normal one, and a transcript printed before the line discipline
is back has bare newlines. `replRaw` has no in-process caller (it demands a real
`*os.File` it can put into raw mode), so a sequence left inline there could only
ever be pinned by pty rows that skip where no pty exists.

**Resize is SIGWINCH → measure → redraw, and both halves of the shape matter.**
Rows because a frame one row too tall makes the terminal scroll, which moves every
row the app believes it placed; columns because `opt.width` was read ONCE at flag
parse, so a resized window kept wrapping new entries to the old width. Shapes
coalesce, newest wins — a drag fires dozens of signals and a queue of stale shapes
is a queue of wrong frames. Lines already in the buffer keep the wrapping they
were rendered with: re-wrapping means re-rendering from entries this program does
not keep.

**Only the interactive loop.** `define <word>`, `echo w | define`, `-raw` and
`> out.txt` are untouched — same bytes, same "ephemeral UI vs record" doctrine.
Inside the alternate screen that doctrine reads differently, because the frame is
ephemeral by construction and the transcript is the record.

## The store

Persistence is YAML files under the **working directory** — no config, no brain
resolution, no home-directory search. `NewYAML(dir, lang, warn)` takes both the
directory and the language as parameters, so *who chooses them* stays one line at
the boundary if a config arrives later.

```
words/<lang>/<slug>.yaml one file per word, under its language
events/YYYY-MM-DD.yaml   append-only, one file per day, named in UTC
                         kinds: looked-up, asked
lang.txt                 the directory's language (#23)
user-model.<lang>.md     the learner model — markdown, because a person edits it
```

**Language is a DECK dimension, not an event one (`#23`).** `wordsDir()` and
`userModelFile()` carry the language; `eventsDir()` and `usageDir()` do not.

The dividing line is *derivation*, not storage. The learner model is READ OFF a
language's deck — level, domains, the words each claim cites — so one shared file
meant a Spanish `--reflect` replaced the English model and every English answer
was then pitched at "A2 — Spanish beginner". M1's boundary review caught exactly
that. An event is different in kind: it is a fact about a moment, not a summary
of a deck, so the argument for leaving `events/` flat does not transfer to it.
A review event names a word and a verdict, and which deck it came from is the
deck's business — splitting the log would turn "how much did I study today" into
a join, and would get there by migrating an append-only artifact. If a later
issue wants per-language study totals, that is a join over the deck.

**A pre-language deck is migrated, blindly and non-destructively.**
`MigrateToLanguages(dir, warn)` runs once in `openStore` and moves `words/*.yaml`
into `words/en/` — without it, every existing word is orphaned rather than lost,
since `Deck()` now reads `words/<lang>/`. It takes NO language: the destination
is always the default, justified by a fact about the files rather than a
preference — a flat deck was written by a tool that only ever consulted the
English dictionary and asked for `_en_us_` recordings. Taking the *active*
language instead would mean one `define -lang es` on a first run filed an entire
English deck under `words/es/`.

`MigrateToLanguages` moves the learner model the same way, for the same reason
and under the same rules. It therefore cannot tell a Spanish word from an English
one, and says so instead of guessing. On a collision the subdirectory wins, the flat file **survives** and
is named — inert, because nothing reads `words/*.yaml` any more, which is what
makes "leave it" strictly non-destructive on the one artifact here that cannot be
regenerated.

**What that layout buys, stated precisely:** it does *not* make sync conflicts
impossible — the same word, or the same day, touched on two machines still
conflicts. It changes the *rate*. With a single `vocab.yaml` every write on the
second machine conflicts, because every write touches the one file.

**Slug rule.** The key is lower-cased, space-collapsed text; the filename is that
with spaces as `-`; and if replacing `-` with ` ` does not return the key, a
6-hex-digit hash is appended. So `hot dog` → `hot-dog.yaml` stays readable while
the hyphenated `hot-dog` cannot share its file. Every file stores `text:`, so the
key lives in the content and the filename is only an index — which makes a future
naming change a rename rather than a migration.

**Interruption is routine here** — this process is quit with Ctrl-C by design —
and the two file kinds defend against it differently:

- **Word files are written atomically**: temp file in the same directory, then
  rename. A leftover temp file is never read as a word, and one corrupt file is
  skipped with a warning rather than making the deck unopenable.
- **Event files are appended, deliberately not rewritten.** Two machines
  appending to the same day merge cleanly; a whole-file rewrite would not. The
  cost is a possible torn final record, so a day log is parsed record-by-record
  and incomplete ones are dropped with a warning. An interrupted write costs the
  event in flight and nothing else.

**A torn record is detected by TERMINATION plus COMPLETENESS**, and three
weaker rules were tried first — each looks sufficient and none is:

| rule | why it fails |
|---|---|
| "it parsed" | a cut leaves valid YAML — `- word: thi` becomes an event with no timestamp |
| "the fields are present" | a cut **inside the timestamp** leaves a shorter date that parses, fabricating an event |
| byte-identical round trip | catches both, but discards **every** record in a log that was ever reformatted |

That last one is the trap worth remembering: it is the strictest rule and it
destroys the history it exists to protect the moment a person, an editor, or a
sync tool rewrites the file's quoting.

A whole record therefore **ends with a newline** and **identifies its subject** —
a word for a lookup, a question for an `asked` event. It was "carries every
field" until #16 added a question that may have no word; a reader still following
that rule discards exactly the wordless events the generalisation exists to keep.
`at:` stays the LAST field for the same reason it always was: completeness leans
on a cut record losing its timestamp, so a field written after it would survive
the cut and make a fragment look whole.

The writer always terminates a record, and `AppendEvent` repairs a missing
terminator before writing — without which one interrupted write costs *two*
events, because the next append lands on the fragment's line and is parsed as
part of it.

**`question:` is the first free-form USER TEXT this log holds.** Every value
before it was a single dictionary headword, and the record boundary a reader
looks for is a literal top-level `- `. What keeps a typed question off column 0
is the writer's quoting, and `storetest` now round-trips five adversarial
questions — including one whose text is a complete forged event record — against
both implementations.

Parsing is one path, always record-by-record. A fast-path-plus-fallback version
double-counted whatever the failed whole-file parse had already collected, and
left the fallback unreachable for input that stayed valid — which is exactly what
a truncation produces.

**Two Store implementations, one conformance suite.** `Mem` is the reference and
ships as production code; `storetest.Suite` runs against both, so "the fake
behaves like the real thing" is a test rather than an assumption.

That claim is only as strong as the fake's ability to HOLD the state, which is
the trap #16 fell into: `UserModel` arrived as a getter with no writer anywhere,
so the suite row asserting "empty before anything writes one" asserted the only
value `Mem` could produce. A method added to `Store` brings its setter with it,
or the row that covers it is unfalsifiable for the reference implementation.

### Capture: one site, one policy

Every entry path records, and records **once**. The single site is
`lookupAndRender` — verified against the call graph, not assumed:

```
defineOnce   ← one-shot, line loop
  └─ lookupAndRender
submitLine   ← raw editor → lookupAndRender      [skips defineOnce]
```

An earlier design captured in `defineOnce`, which would have left the interactive
path — the only one that captured at all before `#4` — silent. `#14` extracted
`lookupAndRender` so the raw path could render cooked and play raw — a split that
outlived its reason when `#30` removed the modes, and stayed because the halves
still differ in kind: this one only writes, and the caller owns the playback that
can be interrupted. Either way it is the one function every path shares.

`decideCapture(found, opt)` is the only answer to "does this lookup count":
found → event + word; not found → event only; `--raw` or `DEFINE_NO_CAPTURE` →
nothing. The environment is read once at flag parse into `opt.noCapture`, so it
is an *input* to the policy rather than a second mechanism beside it.

**`storeCapturer` is the only thing that records a lookup.** It is not the only
writer: `--forget` deletes a word file through `store.Forget`, deliberately and
loudly, which is why it is a separate seam. `storeHistory` used to
write too; since `#4` it only reads at construction and recalls from memory.
Otherwise the raw path would record every lookup twice, and a deck that
double-counts is wrong in a way nobody notices until `#5` orders by it. Pinned two ways, because
one of them is blind on its own: per-path subtests count `Capture` calls with a
fake **at** the seam, and a separate test asserts `Lookups == 1` and one event
against the **real** wiring. The first cannot see a double write, since both
writers live *below* the seam it replaces — verified by restoring the old
behaviour: the per-path tests stayed green while the deck double-counted.

**Two warnings, two homes**, because conflating them stranded the warn-once rule
when the writes moved: a failed write is `storeCapturer`'s (once per process); a
store that cannot be opened at all is `openStore`'s (once at startup, and it says
history is session-only).

**`--forget` removes the word, never the events.** The deck is a working set; the
log is history, and `#8`'s statistics are a fold over it. An absent word exits
non-zero — succeeding silently would hide a typo in the command meant to correct
one — and `-forget` combined with a word is a usage error rather than a silently
honoured half.

### History is events, the deck is successes

**`storeCapturer` appends the event and upserts the word** — see Capture above.
`storeHistory` only *reads*: it loads the log once at construction and recalls
from memory. An earlier version had it writing too, which is why the raw path
would have recorded every lookup twice.

`Prefix` therefore reads the **event log**, not the deck: Up-arrow must recall
the typo you just made — that is when you want to edit and retry — while the deck
must not fill with misspellings. It runs on every keystroke and returns no error,
so it never touches disk.

**Timestamps keep their offset**, so a local-day view is recoverable even though
day files are named in UTC. `#8` must group by timestamp, never by filename.

## Command mode

A `/` in the **first column** switches the line from "define this word" to a
command. The character was forced by the data: `define` takes multi-word
headwords (`hot dog`), so the namespace could not be a reserved word, and no
English headword begins with a slash. A slash anywhere else is part of the
word — `and/or` is a lookup.

**Dispatch lives on `parseREPLLine`**, the classifier both loops already route
through — `replLines` for the piped path, `runEditor`'s `ActSubmit` for the raw
editor, which carries a comment explaining why it must not bypass it. A `/` test
placed in either loop alone would make them disagree about what a line means:
`echo /history | define` would go to the dictionary while the terminal ran the
command. The plan gate caught exactly that draft (PQ-2), and
`TestLineLoopDispatchesCommands` is the pin — deleting the `cmdCommand` case from
`replLines` reddens it.

**Type-ahead needed no change to the pure editor.** `Apply` always took its
candidate list from the caller, so command mode is a different match *source*,
not a different editor. `completionsFor` is the one place that decides which
namespace a line draws from, and it replaced four `hist.Prefix(...)` call sites.
Candidates come back `/`-prefixed because `Suggestion` matches against the whole
typed line: with `/his` typed, `/history` is what completes it. Once an argument
is typed (`/history 7`) the completion is shorter than the line, so no suggestion
is offered — that falls out rather than being special-cased.

**#20 extended type-ahead past the first word without touching the editor
either**, by the same move. `Suggestion` still byte-prefix-matches the WHOLE
line; what changed is where candidates come from. `trailingSegments` cuts the
line into its trailing word-boundary slices, longest first — `what is a syc`
becomes `[what is a syc, is a syc, a syc, syc]`, each carrying the head it was
cut from, with `head+text == line` as an invariant a fuzz target defends.
`historyCompletions` returns the first segment that matches anything, glued back
onto its head, so a deck word arrives as a whole line (`what is a sycophantic`)
and the editor never learns what a word boundary is. Longest-first is also the
precedence rule: segment 0 IS the whole line, so a real past line always beats a
word glued onto a head.

Three constraints keep it quiet rather than chatty. A segment starts only where a
word starts, so a trailing space produces none — load-bearing, because
`History.Prefix("")` returns every entry by contract and an empty segment would
suggest an unrelated line after every space. Inner segments need
`minInnerSegment` runes, so `to` does not suggest `torpid`; the floor skips
segment 0 so two-rune single-word completion is unchanged. And the namespace is
decided ONCE, on the whole line: `completionsFor` tests `parseCommandLine(base)`
before `historyCompletions`, and the segment loop never re-tests it — otherwise
the `7` in `/history 7` would draw from the deck, and a mid-line `/his` would
complete a command. Both are pinned
(`TestACommandArgumentDoesNotDrawFromHistory`,
`TestASlashSegmentMidLineDoesNotCompleteACommand`); an earlier version of the
first passed over the bug it was written for, because its history matched nothing
either way.

**Recall and completion are two lists.** They were one until #20, when
completions stopped being past lines: `walk` assigns `e.Line = matches[next]` and
Enter submits it, so a glued candidate in that list would let Up offer — and
Enter run — a sentence nobody typed. `Apply` takes `candidates{recall, complete}`;
`recall` is `hist.Prefix(base)`, only ever really-submitted lines. That also
retired a pre-existing wart: command mode used to feed the MENU to `walk`, so Up
inside `/his` put `/help` on the line rather than recalling.

**The completion namespace sees past submission markers.** `recallLine` stores
the canonical re-submittable form, so an asked question is `?…` and a forced
literal is `\word`. `matchesFor` unwraps both — the `?` especially, because
`readsAsQuestion` classifies a bare sentence and the system adds that marker, so
requiring the user to type one to complete a question they asked without one
would make past questions uncompletable. `/` is deliberately not unwrapped:
commands are a real separate namespace, not a marker on a word.

### `/lang` and the one thing a `deps` swap cannot reach

`/lang` is two halves with different preconditions, and separating them is what
makes a one-shot `define /lang es` work: **persisting needs a DIRECTORY,
re-deriving needs a SESSION.** `newCommandCtx` supplies the durable half
(`deps.persistLang`), and both loops override it with `sessionSetLang`, which
adds the rebuild. So the only case `/lang` refuses is having nowhere to write —
where `/sound` refuses whenever there is no session, because `/sound` is
explicitly for the rest of *this* session and a language outlives it.

The rebuild goes through **one builder called twice**: `openStore` constructs the
session with `newLangDeps(lang)`, and `/lang` calls the same closure.

**`langDeps` is the set, as a TYPE, and it is EMBEDDED in `deps`.** Both facts
were earned. The members are the deck, the capturer, the vocabulary, the usage
source and — through its own seam — the dictionary; `#23 M2` adds nothing to that
list, but it changed what belongs on it, which is the point.

The set was a list in a doc comment first, and it went stale **three times**:
`opt.voice` at M1's boundary, the learner model a round later, and `d.usage` at
the close — the last added by the very milestone whose comment still called it
"not language-scoped". Making it a struct was not enough either, because both
`openStore` and `applyLang` still copied its fields by hand, so a fifth field
stayed forgettable at two sites. Embedding is what finally made adoption one
assignment (`d.langDeps = d.newLangDeps(l)`), so a member added later is adopted
without anyone remembering to.

**`history` is the one thing deliberately outside it.** `events/` is not
language-scoped, and rebuilding it on a switch would orphan the `History` that
`runEditor` has already `Load()`ed while every other path read a fresh empty one.
That exclusion is checked by an identity assertion rather than argued in prose —
because "deliberately not in this set" is a claim with a shelf life, and the
`usage` exclusion was true when written and false three commits later.

**The `&voc` parameter is the subtlest member.** `runEditor` resolves the
highlight set into a LOCAL before its loop starts (`voc := vocabularyFor(d,
opt)`), reads it on every redraw, and a `deps` reassignment structurally cannot
reach it. Without the pointer the editor keeps painting the previous language's
words while every other path has moved on. The general shape — *a loop local
derived from `d` before the loop* — is why the switch lives in the loop that owns
those locals rather than in `commandCtx`.

Three pins, at three altitudes, each mutation-checked:
`TestLangSwitchKeepsOneHighlightSetAndItIsTheNewLanguages` for the editor local
and the usage feed (in both directions — losing the feed is the quieter
failure); `TestLangSwitchReDerivesEverythingDownstreamOfTheLanguage`, which
drives `/lang es` through `run()` and asserts what the CDN was ASKED for; and
`TestTheDictionaryIsBuiltForTheLanguageAtBothMoments`, which drives the boundary
through `run()` too. That last one matters: `d.newDict` was set at zero call
sites in any test, so deleting both derivations left the whole suite green while
production would dereference a nil `Dictionary`.

The pattern across all three: a unit test on the pure function could never have
caught any of these, because the pure functions were always right. What was wrong
was that nothing called them again.

## Highlighting the words you are learning

**One predicate, and it is the point.** Every highlight decision goes through
`Vocabulary.Has` (`vocab.go`). Today the set is the deck; #22 narrows it to the
words still being learned, so a word that has become the learner's own stops
being highlighted. That swap is one constructor, and nothing above the seam
moves — which is why the seam exists now rather than being introduced later.

**Not `History`, though the shapes rhyme.** Different source (`words/` vs the
event log), different query (set membership vs ordered prefix search), and
decisively different contents: history carries typos on purpose so they stay
recallable (#20), and highlighting a misspelling as a word you know is the
opposite of reinforcement. `storeVocabulary` embeds `memVocabulary` rather than
growing a second set, so "what is in the set" has one implementation.

**`highlightSpans` is the pure core** (`highlight.go`): text plus a `Vocabulary`
in, alternating known/unknown `span`s out. Concatenating the spans reproduces the
input exactly — the invariant a fuzz target defends, because a renderer joins
them back into what the user sees and a lossy split silently corrupts a
definition. Longest match wins at each position, so with both `hot` and `hot dog`
in the set the phrase highlights rather than its first word.

`wordRuns` is the single tokenizer, used by the prompt line today and by the
definition and answer paths from M2 — one tokenizer, so a word highlighted in a
definition is the same word highlighted at the prompt.
Apostrophes and hyphens are INSIDE a word, so `don't` and `hot-dog` are each one
token — hyphens especially, since a hyphenated deck entry has `hot-dog` as its
`store.Key` and splitting on the hyphen would make it unmatchable. Phrases are a
level up: `phraseGap` lets a multi-word key span only spaces and tabs, so a
`Render`-wrapped `hot\n  dog` cannot form the candidate `hot dog` and paint a
green run through the wrap indent.

**ANSI does not nest, and that shapes every renderer.** `RenderLine` writes
`knownOn + word + sgrOff + inputOn` for each known span: without re-opening
`inputOn`, everything after the first highlighted word goes plain. The same
constraint is why definitions and streamed answers share a mechanism rather than
each growing their own — see M2/M3.

**`highlightWriter` is why definitions and answers are one mechanism** (M2).
They share the hard part: the text already carries ANSI codes, and it may arrive
in pieces. A definition is a complete string; an answer arrives as stream deltas
where `obsequious` can land as `obseq` + `uious`. Giving them separate
implementations would mean two sets of ANSI-resume rules to keep in agreement.
`Render` calls it per admitted REGION; the answer stream wraps its writer.

Its contract, in the order the rules matter:

1. **Nothing overtakes held text.** The writer holds a tail while a phrase is
   still possible. An escape arriving during a hold RESOLVES the hold first, then
   passes through — otherwise a reset can land before the word it was closing.
   Left-to-right draining is what makes this structural rather than a check.
2. **A phrase spans only spaces and tabs.** An escape, newline or punctuation
   closes the window. So `hot\x1b[0m dog` is not `hot dog`: a phrase whose halves
   are styled differently is not a phrase, and that is what makes rule 1
   implementable.
3. **The hold point may not cut a completed phrase.** Holding the last
   `MaxPhraseWords` tokens is right for text that may still grow, but with
   `hot dog` in the deck it lands inside `one hot dog please` — emitting `hot`
   alone and losing the match forever. A known span straddling the hold point
   drags it back to that span's start. Plain text may be cut freely.
4. **Downstream errors poison the writer.** The first failure is remembered and
   nothing is emitted after it, so no byte is written twice. A short write with a
   nil error is a failure — the line-ending writer produced exactly that, which
   is how this rule was found. Both interactive loops replaced it with the screen
   (`#30`, then `#41`), so the writer that motivated the rule is gone; the rule
   outlives it because `screen.Write` and any future wrapper owe the same
   contract.
5. **Flush is part of the contract.** Held text is invisible until it happens.

`sgrState` is the pure half: it watches escapes go past and answers "what style
would a terminal be in right now", so a highlight can hand that style back. It
accumulates SGRs until a reset, because `Render` opens bold and colour
separately and a terminal composes them.

**The decision is per region, and the table has to be complete.**
`RenderOpts.Vocab` reaches `Render` rather than wrapping its output, because a
finished string has no structure left to consult — wrapping re-styled the
headword, which the Spec puts out of scope and which is the COMMON case for a
learning tool, since you revisit words. `admitsHighlight`'s doc comment is the
table — prose admitted, labels withheld, with no count written beside it because
a number in prose beside an enumeration is a second source of truth that drifts.
It is a decision procedure only because `TestHighlightsAppearOnlyInAdmittedRegions` derives the
admitted text from the parsed `Entry`, so a region that starts leaking fails
without anyone remembering to add a row. `Render` stays pure — `Vocabulary` is
injected data and `highlightSpans` is a pure function of it.

`TestHighlightingLosesNothing` is `Render`'s own no-data-loss invariant
re-asserted with highlighting on, stripping escapes first — the codes carry
digits that `alnum()` would otherwise read as content.

**`vocabularyFor` answers "what should be highlighted right now" for every
render path.** Highlighting needs the set LOADED, and `Load` used to live in
`runEditor` — so `define <word>` and piped stdin rendered against an empty set
and highlighted nothing, two of three entry paths dead while the suite was
green. One function now owns "loaded, and only with colour". The enumeration that
guards it is **entry path × render surface**, not entry path alone:
`TestEveryEntryPathHighlightsDefinitions` and
`TestEveryEntryPathHighlightsAnswers`, each row driven with an UNLOADED set,
because a pre-filled one begins after the hop that fills it. M3 added the answer
surface without widening the first table, and a mutant dropping the `Load` passed
the whole suite — the same Critical one surface over. A new surface needs its own
rows.

**The answer stream is the writer's other caller**, and it is the OUTERMOST
writer on the answer — over the screen in the raw loop, over the real stdout when
piped. So highlighting sees the answer's own logical text and the screen places
the highlighted bytes as lines afterwards. It used to wrap the raw loop's
line-ending writer instead, which `#30` D5 removed. Inverted, the highlighter
would meet `\r\n` where it expects `\n`.

The `Flush` is DEFERRED rather than written at each return, and that is
structural: `runAsk` returns on five paths and held text is invisible until a
flush, so a per-path flush is four chances to forget one silently-dropped last
word. On four of those paths the hold is already released before the return (the
answer's own newline, or the trailing `Fprintln`), and the fifth is unreachable
through `llmtest` — `askhighlight_test.go` records that scope in full rather than
implying coverage, and names what a fake would need to reach it. What DOES
observe the defer is the error report below.

**The command namespace is withheld, like every other boundary decision.**
`highlightSetFor` hands `RenderLine` a nil vocabulary on a `/command` line, so a
deck word sharing a command's name does not render green inside `/history 7`.
`parseCommandLine`'s own comment calls `/` a separate namespace rather than a
marker on a word, and #20 decides that namespace exactly once, on the whole
line — highlighting inside it read as the vocabulary feature leaking across.

**The set grows mid-session, from the one place that already knows.**
`storeCapturer.Capture` adds a word after `Upsert` succeeds — the single site that
knows a lookup both succeeded and earned a deck entry, so a failed lookup (which
is history, not vocabulary) never enters. `openStore` hands the SAME instance to
the capturer and the renderers; two instances would mean lookups landing in a set
nothing draws from, which has its own test.

**Typing `/` shows the menu.** The inline grey suggestion and the dropdown
answer different questions, which is why both exist: completion answers "what
single string extends this line" and only helps someone who already knows the
command's name; the menu answers "what are my choices". `menuLines` is pure and
filtered by the same prefix, so the list narrows as you type and vanishes when
nothing matches — a stale set left on screen would be worse than none. The name
column is measured against ALL commands so the summaries do not shuffle sideways
while the list shrinks.

The terminal half is a dropdown, not scrollback: `paintMenu` writes the rows
below the prompt and walks the cursor back up by the same count, erasing
`max(previous, new)` rows so a shrinking list leaves nothing behind. It is
cleared before a submit and before exit. **Known limit**, shared with the erase
arithmetic elsewhere in the raw path: if the menu does not fit below the cursor
the terminal scrolls and the cursor-up count lands a row off; it self-corrects on
the next keystroke, because the prompt line is fully rewritten each time. A pty
conformance test covers the placement, because an in-process test can only prove
the bytes were emitted, not that the cursor came back.

**One source for "what does this line match".** `draw` computes the candidate
list itself rather than accepting one. It used to take a parameter, and one
caller passed the list computed BEFORE the keystroke was applied — so the grey
tail was rendered against the previous line. Both lists were history before
command mode and a stale superset usually shared its first match, so nothing
showed; the namespace switch made the stale list come from a *different set*, and
typing `/` suggested `/history` out of recall while the menu under it listed
commands and Tab accepted `/help`. `Apply` still receives the pre-keystroke
pair, which is correct — it is deciding what to do with that keystroke given the
line as it stands — but nothing can hand `draw` a stale one. `draw` resolves only
the completion half, because that is all it renders; resolving the pair there
would compute a recall list per keystroke that nothing reads.

**Tab accepts, Return submits what was typed.** `/his` + Return dispatches `his`
and gets a suggestion, it does not run the unique match. That is `#14`'s contract
for words, and command mode diverging from it would make Return mean two things
on one line.

**Every entry mode reaches it.** `define /help` as a one-shot argument, `echo
/help | define`, and `/help` typed at the prompt all go through `parseREPLLine`;
a one-shot that skipped it would have sent `/help` to the dictionary (BR-13).

**Complete exactly, accept loosely.** `commandCompletions` is case-sensitive
because `Suggestion` byte-prefix-matches the typed line — a completion has to
literally extend what was typed, so `/HIS` cannot be completed by `/history`
without rewriting keystrokes. `dispatchCommand` uses `EqualFold`, so a submitted
`/HELP` still runs.

Adding a command is a row in `commands` plus its `run`; `dispatchCommand`
switches on outcome (found / not found), never on which command it is, and
`nearestCommands` REPORTS whether anything was close rather than leaving the
caller to infer it from a count — inferring it was wrong for every near-miss
while one command was registered (BR-9).
`commandCtx` is deliberately narrower than `deps` — a command still cannot reach
the dictionary or the player. `#29`'s `/pron` did not widen that: `commandCtx`
gained a `replay` CLOSURE, so the command records a language and the loop
performs the playback, in raw mode, where Ctrl-C can still reach the key reader.

**The registry is the single source, and this list derives from it.** It listed
three of five commands for two releases — `/lang` (`#23`) and `/pron` (`#29`)
were both added without it, 2 for 2 — so it is generated from `commands` and
pinned by `TestDocsQuoteTheCommandList`, the mechanism `localeHelp` and
`pronHelp` already use. Add a command and this page fails the build until it
catches up.

<!-- command-list -->
| command | does |
|---|---|
| `/help` | list the commands |
| `/history` | words looked up recently |
| `/sound` | how many times to play a pronunciation |
| `/lang` | the language this deck is in |
| `/pron` | replay this word in its source language, once |
<!-- /command-list -->

Argument forms are documented with each command rather than in the summary: the
summary is what `/help` prints, and a table that padded it with syntax would stop
matching the screen. `/history [N]` takes `N`, `--days N` or `--days=N`;
`/sound [N]` reports when bare; `/lang` reports when bare and persists when
given.

`/pron`'s own rule is code-owned rather than restated here, because this is the
third prose site to state it and the first two went stale:

<!-- pron-command-help -->`/pron` takes a language, or nothing: with no argument it reads the source language off the entry's ORIGIN and says which it chose. It declines when ORIGIN names only historical stages (Old French, Latin) or cognates ("related to Dutch …"), because neither is a language anyone speaks the word in today.<!-- /pron-command-help -->

**Opening a store does not read it.** `storeHistory` used to read the whole event
log in its constructor, so `define /help` paid for a log it never consulted and
`/history` read it *twice*. The read is now `History.Load`, called by the raw
editor — the only thing that recalls. `replLines` never touched history at all,
so it does not pay either, and `Prefix` stays IO-free because it runs on every
keystroke.

A first attempt at this exempted commands from `withStore` instead. That is worth
recording as the wrong shape: the exemption also skipped where `deps.clock` is
supplied, so it stranded an invariant it was not thinking about. Removing the
cost at its source meant nothing needed exempting.

`/sound` is the first command that CHANGES the session rather than reporting on
it, and the seam is deliberately narrow: `commandCtx.setTimes func(int)` writing
through to the loop's own copy of `opt`, not a `*options` a command could use to
reach anything else. `nil` is the honest representation of "there is no session
here" — the one-shot path refuses rather than silently accepting a command that
could not do anything. `--sound` is the same setting for one run; `-times` is its
older name and still works, and passing both is a usage error rather than a guess
at which was meant.

### `/history` and the local-day question

Two facts collide here, and every rule below comes from one of them: **the
question is a local-calendar one**, and **the event log is a set of UTC-named
files whose records carry their own offsets**.

- **The window is local midnights**, `AddDate(0,0,-(days-1))` from today's, never
  `now - N×24h`. A window starting at 00:30 drops everything before half past
  midnight on the first day, and a DST day is 23 or 25 hours, so a Duration lands
  an hour off.
- **Never filter by filename.** A lookup at 19:50 local *today* is written to
  *tomorrow's* UTC-named file. A filter over local day names never opens it, so a
  lookup from ten minutes ago vanishes and `/history` reads "nothing today".
  `store.Events` compares timestamps for exactly this reason.
- **Membership and ordering are different time facts.** A word is in the list
  because it was queried inside the window; it sits where it does because of when
  it was **first ever** seen — so a word you keep returning to holds the position
  its first sighting earned instead of churning to the top. `summariseLookups`
  therefore takes the whole log, which costs nothing because `Events` reads every
  day file whatever `since` says, and answers both facts from one source.
- **Found lookups only.** A typo stays in `#14`'s up-arrow recall and out of the
  words-queried view. One log, two readers.
- The row shows the **key**, not whichever spelling arrived first — the row *is*
  the key, so showing a spelling would make the dedupe rule invisible.
- The date shown is `FirstAt`, the field the list is **sorted** on. Showing any
  other date makes the ordering look arbitrary.

Known cost: `Events` is O(all history) per call. One read per `/history` on a
personal word list is the right trade today; when it stops being, the fix belongs
in the store — an index, or a filename pre-filter that still *decides* on
timestamps — not in this command.

## Free-form input

A line that is not a word and reads as a question is answered by the model rather
than looked up. There is no mode and no prefix to remember — which is the whole
claim, so the interesting part is how "is this a word" gets decided.

**The dictionary is the classifier.** Word count cannot be the signal: `hot dog`
is a two-word headword and `defenestrate` is one word. What works is free,
offline and already on the path — **ask NOAD first**, and classify only what it
misses. `hot dog`, `a priori` and `use` are lookups because the dictionary has
them, not because a predicate was careful.

**One decision table, in two pure halves.** `parseREPLLine` is the syntactic half
(command / blank / forced question / forced literal / word); `readsAsQuestion` is
the semantic half, and it is only ever asked about a line NOAD already missed.
Both loops and the one-shot route through the same two functions —
`TestConsoleDecisionTable` drives the whole table end to end through the real
route, because asserting each half separately proves each is correct and leaves
the *table* unasserted.

| input | classified as |
|---|---|
| `/history 7` | command — `/` in column 1 still wins |
| `sycophantic`, `hot dog` | lookup — NOAD has an entry |
| `what's the difference to obsequious?` | question — no entry, reads interrogative |
| `sycophanti` | not found — no entry, does not read interrogative |
| `?hot dog` | question, forced — the dictionary is not consulted at all |
| `\how so` | not found, forced — the question fallback is suppressed |

`readsAsQuestion` has three arms: a trailing `?`, a leading interrogative or
auxiliary (`what's` → what, `isn't` → is, and `when` is not a negation), or a
leading request verb with an object (`use it in a sentence`). A single-word line
with no question mark is never a question — that is a headword shape, and a miss
is a typo. `why?` is, because the mark is explicit and its arm is tested first.

**There is deliberately no length arm.** A draft had "≥5 words → question" to
catch `difference between sycophantic and obsequious`, which reads as neither
interrogative nor imperative. That is a word count wearing a different hat, and
word count is the signal that cannot work. The cost is real and named: that line
answers "not found", and `?` is its recovery.

**Both hatches, and why neither is exclusive.** `?` forces a question and `\`
forces a lookup; a bare question still asks and a bare word still looks up, so
each hatch is a recovery rather than syntax. They are decided in `parseREPLLine`
next to the `/` test, for the reason that test is there: a prefix checked in
either loop alone makes the loops disagree about what a line means.

**A question is not a lookup, and the log knows it.** The route decision sits
*before* `d.capture.Capture` in `lookupAndRender`'s miss branch — deliberately,
because the event log is what `#8`'s statistics and `#17`'s learner model fold
over, and a question recorded as a not-found lookup is data that was never a
lookup. Verified end to end: six lines in, four events out, neither question
among them.

**A question does not become the current word** either. The ask outcome carries
exit code 0 — it is not a failure — so the guard is `out.ask == "" && out.code ==
0`, not the code alone. Testing the code alone made a bare Enter "replay" the
question, and would have made the question the word `#16`'s context claims the
next one is about. `session` exists for this: it replaced three separate
declarations of "what is this session holding" (`replLines`, `runEditor`,
`submitLine`), because the rule would otherwise have been written three times.

**The answer.** A question goes to `internal/llm` with the DIRECTORY as its
context: the word on screen and its dictionary entry, this session's lookups, the
recent deck, the learner model, and the session's own earlier exchanges. Three
consequences fall out, and they are the reason for this shape — a fresh process
answers as well as a long-running one, the context is inspectable as files rather
than trapped in memory, and the answer is adaptive for the same reason the
generated items will be.

`askContext` is that context as DATA and `renderAskPrompt` is pure, which is what
makes "what did we send" assertable without a socket: the golden snapshots the
`llm.Request` through the same renderer the transport hashes, so a field added to
the prompt without thought shows up in its diff. An absent section is **omitted**,
never rendered empty — an empty `## The learner` says there IS a model and it is
blank, a different claim, and the one that produces a confident generic answer.

Measured end to end against the live proxy with a two-line learner model ("B2,
reads business news, weak on near-synonym distinctions"): the answer came back
with a *"Business-news nuance"* paragraph and *"Related near-synonyms in your
range"*, and quoted the NOAD entry back — *"the dictionary definition you looked
up actually contains both"*. The adaptation is visible in the output, which is
the only place it counts.

**Ctrl-C stops the answer, not the session — and the swallow lives in the
READER.** An interrupt that a scope consumed must not ALSO be delivered as a key,
or the loop applies it as "quit" the moment the answer ends. Having the loop race
for keys during a stream would have worked too, and would have eaten type-ahead;
`interrupter.Fire` reports whether a scope took the interrupt, and `readKeys`
drops it when one did.

That made the key channel's buffering load-bearing: the loop stops reading while
an answer streams, and on an unbuffered channel the reader blocks on the first
key typed during it and never decodes the Ctrl-C behind it.

`TestPTYCtrlCMidAnswerKeepsTheSession` is the row that suite's own header said it
lacked — "the answer stopped AND the next lookup rendered" is an observable only
a surviving session produces, where "exited cleanly" is produced by the byte
path, the signal path and a crash alike.

**The log records the question, not the answer.** `#17` wants to know what the
learner asked about — a strong signal of what they are working on — and every
consumer of that log is a fold, which answers would bloat for nothing.
`complete()` generalised from "has a word" to "has a SUBJECT" to allow it: a
question asked before any lookup has no word, and requiring one would drop
exactly the events `#17` reads. `at:` stays the last field in the struct, because
the torn-record rule leans on a cut record losing its timestamp.

**One ask entry, six cells.** A question arrives by two routes — forced (`?…`,
decided by the parser without a dictionary call) and unforced (a miss that reads
as one) — across three entry modes. That is the enumeration every claim about
asking quantifies over, and all six go through one `ask(ctx, d, opt, sess, out, errOut, q)`.

`question` carries **how** it arrived, because the route changes what can
honestly be said: an unforced question is one the dictionary missed, so "is not a
word" is true by construction; a forced one skipped the dictionary, and `?why`
*is* a headword. Each loop wraps that one call in its own closure — the raw
loop's is where the streaming writers hang, because a raw terminal is what makes
them differ; the interrupt SCOPE is not in either closure, since `askScoped`
owns it for both — but the
decision itself does not fork.

**`-raw` never asks, and "never" names its cells.** It is the scripting form, so
an unforced miss simply does not fall back, and an explicit `?` alongside it is a
usage error rather than a guess between contradicting flags. The predicate
(`mayAsk`) lives with the ask rather than with either dispatch: guarding only the
fallback left three of the six cells asking anyway while the README stated the
absolute. **A rule stated as an absolute has to be enforced where the thing
happens, not on one route to it** — and the test that asserts it names the
enumeration and covers every cell.

**A per-line exit code has to survive the loop.** `replLines` computes one code
for the whole run, and collapsing a dispatch's code into a boolean loses the
difference between a lookup failure (1) and a usage error (2). That is BR-16,
fixed for commands in #15 — and #16 reintroduced it for questions, because the
new branch collapsed `ask`'s code the same way. There is now one `fail(code)`
sink every branch feeds, so a third branch cannot repeat it.

**The exit-code absolutes README states are measured across `{one-shot, piped}`**
— the enumeration they quantify over; the raw editor has no exit code of its own,
because an interactive typo does not fail a session. This paragraph used to say
"the FOUR absolutes", and a later change added ask-path exit sites without
touching it: a count is a measured claim that drifts the moment anything is
added, and it drifts silently. What the ask path can exit with is enumerated by
the degradation messages themselves — `unavailable` when nothing was configured,
`unavailableAfterSending` carrying the underlying cause when something was, and
the loud `ErrRequest` arm — each of which is the subject of a row test rather
than of a number here.

**A hatch with no payload is a malformed line, in both hatches and every mode.**
`?` and `\` alone are usage errors (exit 2) rather than blank lines — a bare `\`
used to strip its prefix, fall into the empty-line test, and *replay audio* when
a word was current, so the two hatches disagreed at the prompt. `nothingSays` is
the one place that answers "this line meant nothing — why"; the line ending stays
with the caller, because only the raw loop needs `\r\n`.

**Recall stores what a line MEANT.** A forcing prefix is part of that: `\how so`
recorded as `how so` comes back from Up-arrow and re-submits as a *question* —
the opposite of what the hatch was typed to force. So `recallLine` is the one
canonical, re-submittable form, and all three recall sites use it. Whitespace is
still collapsed, because that changes no meaning.

## The learner model

`define --reflect` folds the deck and the lookup log into the learner model, the
third artifact in the working directory. Batch and on demand: no model call ever
sits on the lookup or review path, which is what keeps a lookup instant and
offline.

**Every claim names its evidence, and the evidence is CHECKED.** The typed answer
carries the deck words behind each claim and `checkEvidence` drops any claim
citing a word the deck does not hold — the same rule this project applies to
distractors, *selected and never invented*, arriving in a second place. Without
the check, "every claim names its evidence" is a formatting convention that a
plausible hallucination satisfies: a fabricated sailing domain citing `luffing`
and `clew` looks exactly as checkable as a real legal one.

It checks **usability** too, and that arm was written from a measured failure.
Under a schema requiring every field, the model fills the ones it does not
believe in — a domain literally named `x`, a rationale of `placeholder` — because
a stub satisfies the shape. A domain with no name or no directive tells authoring
nothing, which is the only reason a domain claim is generated, so it is dropped
and said out loud.

**The field name is what steers the model.** `evidence` invited whole sentences
(*"Advanced specialist items looked up: 'certiorari', 'estoppel' — near-native
legal register"*), and every level claim was then correctly dropped, so `## Level`
went silently missing from every run. Renaming it `evidence_words` fixed it. The
tell was that DOMAIN claims had been citing bare words correctly the whole time:
they have no `rationale` field competing for the explanation.

**Two sources, one rule.** The DECK decides which words are evidence; the LOG
decides how many times and when. It follows from what `--forget` already
promises — remove a word from the deck, keep its events — so a forgotten word
stops being evidence while its history still counts, without a special case.
The per-word fold is `summariseLookups`, the same one `/history` reads: a second
fold would be a second answer to "how many times has this learner looked this up".

**Two floors, both preferring nothing to something confident.** Below 12 deck
words `--reflect` writes nothing and says how far off you are; and when every
claim is dropped it writes nothing rather than a file with frontmatter and an
empty promise, which reads as an answer. `#16`'s ask path degrades cleanly on an
ABSENT file and not on an empty one.

**`## Corrections` is spliced, never regenerated.** Everything above the marker is
replaced, the marker and everything below it copied byte-for-byte. The marker is
matched only at line start and only outside fenced blocks — this file documents
its own format, and a learner arguing with the analysis may paste that block in.
A fuzz property over malformed input (tilde fences, unterminated fences, CRLF)
asserts that whatever follows the first out-of-fence marker survives
byte-identical. That is **one direction of the invariant, not the invariant** —
it says nothing about everything ABOVE the marker being replaced, which is
exactly where a forged marker lives, so a separate test asserts regeneration.
The atlas called it "the one thing that must hold" until a forged marker proved
otherwise.

**Model text is neutralised before it is rendered**, and that is the other half
of the file's integrity. The learner model is marker-delimited, so a directive — or
an evidence word — containing a line-start `## Corrections` forges a second
marker above the real one; the next run splices there and everything below,
including the learner's actual corrections, is frozen forever. `oneLine`
collapses whitespace and escapes `|` in every model-supplied field, which closes
the injection outright rather than filtering for the marker: every line this
renderer emits is prefixed by `**`, `Read off: ` or `| `, so text that cannot
start a line cannot forge structure nobody has thought of yet.

Note the asymmetry the file rests on. The corrections text is the LEARNER's and
is copied byte-for-byte precisely because they own it; the analysis text is the
MODEL's, and is rendered into a structure the file's integrity depends on. Same
file, two opposite rules — conflating them is what created the hole.

**What it is worth, measured.** With a deck of sailing, law and cooking words,
asking *"what does trim mean here?"* answers well WITHOUT the model — `#16`
already sends the deck. With it, the answer leads with the right domain and
reaches into the deck's structure (*"the clew is where the sheet attaches, so
trimming acts on the clew"*). Depth and ordering, not a different topic. The
`directive` fields are aimed at `#10`'s authoring, which is where the payoff is
designed to land.

## Entry modes

`run` dispatches modes first (`-forget`), then on argument count. The function
every path converges on is **`lookupAndRender`**, not `defineOnce` — the raw
editor bypasses `defineOnce` entirely, which is why capture lives one level down.

| invocation | behaviour | reaches |
|---|---|---|
| `define <word>` | one-shot | `defineOnce` → `lookupAndRender` |
| `define` on a terminal | raw editor | `submitLine` → `lookupAndRender` |
| `define` piped, or `echo w \| define` | line loop | `defineOnce` → `lookupAndRender` |
| `define -forget <word>` | mode; no lookup — deletes one deck entry | `forgetWord` → `store.Forget` |
| `define /help` | one-shot command | `parseREPLLine` → `dispatchCommand` |

The loop reads stdin **unconditionally** and only the prompt is TTY-conditional —
there is no interactive/batch branch to keep in sync, and the whole loop is
testable from a string. `deps.stdinIsTerminal` is injected because a test
harness's stdin is never a terminal; note it is a different question from the
stdout probe that drives colour.

**The "♫ playing N×" indicator is ephemeral.** It exists to show the program
responded to a keypress; once the sound has finished it is noise, so on a
terminal it is erased and the settled screen shows only the definition. This
holds on both paths — the define path erases in place, the replay path
additionally steps back onto the prompt. Piped output keeps the indicator as a
plain line and emits no escape sequence, asserted.

A bare return replays audio and leaves the screen **unchanged**. `speak` is
silent by construction and `defineOnce` owns the "♫ playing N×" line, so the loop
cannot accidentally reprint a definition. Interactively the loop flashes the
indicator on the line Enter's echo just opened, then erases it and steps back
onto the prompt (`eraseLineAndStepBack`) — the echo is undone rather than
accepted, so the view never scrolls and the word you are hearing stays beside the
definition you are reading. Cursor control was a `#2` non-goal, lifted by the
operator for exactly this. Gated on `interactive && tty` — **both** streams must
be the terminal, because the escapes go to stdout while the echo being undone
came from stdin; gating on stdin alone leaked escapes into `define > out.txt`.

**Cancellation prints nothing.** Killing `afplay` is *how* Ctrl-C is implemented,
so `playAnnounced` — the single report site — suppresses a diagnostic when
`ctx.Err() != nil`; without that guard a SIGINT during playback printed `define: afplay: signal: killed` and exited
0, telling the user their own keypress had failed.

**"No recording" is cached; a transport failure is not.** `ErrNoAudio` is
permanent, so replaying a word without audio must not re-issue all four candidate
requests each time; `ErrFetchFailed` stays retryable so a transient outage does
not poison the session. The cache derives that distinction from the error
taxonomy rather than re-deciding what "failed" means.

**(Cooked path only.) The erase arithmetic assumes no input arrives during
playback.** This applies to the fallback line loop, not the editor above: `eraseLine`
acts on whatever line the cursor is on *now*, and the loop is blocked inside
`speak` for seconds with the tty in cooked mode and ECHO on. A second impatient
Return during playback is echoed by the driver, moves the cursor down, and the
post-playback erase then clears the echoed line instead of the indicator —
stranding `♫ playing N×` on screen. So "the view never scrolls" holds for a user
who waits, not unconditionally. Raw mode removes the assumption entirely by not
echoing at all, which is `#14`'s job. Replay costs no network: `cachingAudioSource` decorates the `AudioSource` seam
*inside* `repl`, so the production and test wiring are the same line and
`fakeCDN.Requested()` is the assertion.

`main` wraps the context in `signal.NotifyContext`, which changed the one-shot
path too: Ctrl-C during playback now cancels `afplay` through
`exec.CommandContext` and lets deferred cleanup run, rather than killing the
process and stranding a temp file.

## Pronunciation

The speaker button on Google's dictionary panel is a plain static MP3, and the
URL is derivable from the word alone — no key, no scraping. `AudioCandidates`
returns them in preference order:

```
…/pronunciation/2022-03-02/audio/sy/sycophantic_en_us_1.mp3   (and _2)
…/sounds/oxford/sycophantic--_us_1.mp3                        (and _2)   English only
…/pronunciation/2022-03-02/audio/ma/madrugar_es_es_1.mp3      (and _2)
```

The order is measured, not assumed: across a 10-word survey the 2022 generation
strictly dominates the legacy paths (`gaslighting` exists only on the newer one;
`defenestrate` needs `_2` on the older one). That is why there is a candidate
*list* rather than one URL, and `fetch_conformance_test.go` asserts both facts
still hold.

**One language unless you name another (`#23`, then `#29`).** `AudioCandidates`
takes a `voice{Lang, Locale}` and still builds for that language alone — nothing
in it searches. What changed is one level up: `utterance.Candidates` walks a
SOURCE voice first and then the session's own, and `utterance` is the type every
play site now goes through.

The walk exists only when `-pron` or `/pron` named a language, or `/pron` read
one off the entry's `ORIGIN` (`#35`). `#23` deleted
`#27`'s planned `voices()` because a fallback would have been *guessing* which
language a word belongs to, on every lookup, at every user's expense — and `#29`
measured how badly that guess would go (see below). Being told is a different
thing from guessing, and a told fallback is announced: `reportVoice` says which
voice actually answered, read off the URL that answered rather than predicted
from the request.

`voice` is a struct rather than two strings because `"es"` is a legal value of
**both** fields — two positional arguments are transposable at every call site
and the compiler cannot tell.

**The legacy generation is gated to English**, on measurement: `madrugar--_us_1`
and `madrugar--_es_1` are both 404 while `sycophantic--_us_1` is 200. At
~300–600 ms per miss against ~40 ms per hit, asking anyway costs most of a second
per Spanish lookup for a guaranteed 404. `TestTheFetchLoopAsksOnlyForTheSessionsLanguageWhenNoneWasNamed`
asserts what is actually REQUESTED, not just what the pure function
returns — its negative case is the one that catches a regression here.

**The locale rule (`#27`).** `localeFor` defaults the locale to the language code
with one exception — English is `us`, because the CDN writes `_en_us_` and
`_en_gb_` rather than `_en_en_`. `-locale` then overrides it **for every
language**: `-lang es -locale us` builds `madrugar_es_us`.

**Nothing whitelists which pairs exist**, and that is a decision. A table of valid
language/locale combinations would restate a fact the CDN owns and go stale the
moment Google adds a variant — the same argument `ParseLang` makes for not
enumerating languages, and using a different philosophy for the adjacent field
would be the inconsistency. So `-lang es -locale gb` builds `madrugar_es_gb`,
which 404s and degrades to the warning every missing recording produces.

**For Spanish the choice is phonemic, not an accent flavour.** `es_es` is
Castilian, where *cazar* /θ/ and *casar* /s/ are distinct; `es_us` is Latin
American *seseo*, where both are /s/. Choosing one chooses which sound system the
learner acquires. It matters more for Spanish than for English because Spanish
orthography is phonemic, so the dictionary writes **no notation at all** — the
recording is the only place that information exists.
`TestNonEnglishEntriesCarryNoPronunciationNotation` pins that as expected rather than
a gap, and its sibling pins the scope: a Spanish word in an ENGLISH entry does
carry notation, four anglicised pronunciations for `jalapeño`.

**One source for the policy text, and this page consumes it too:**

<!-- locale-help -->regional variant of the pronunciation, per language: en us|gb; es es (Castilian, cazar /θ/) or us (seseo, /s/). Others exist — the CDN decides, not a list here<!-- /locale-help -->

`localeHelp` is that string, and both this page and the README derive from it
through `TestDocsQuoteTheLocaleHelp`. The policy was stated in four places with
nothing keeping them in step; wiring only the README would have left this page as
the next copy to go stale, which is the half-fix the boundary review caught.

For one milestone `-locale` was English-only: `#23 M1`'s D2 shipped that as an
explicit interim rule, named `#27` as its successor, and printed a diagnostic
rather than silently ignoring the flag. That is now history.

`playN` keeps the repeat loop in the shell rather than behind `Player.Play(n)`,
so `fakePlayer` can count plays — which is how "play it three times" is an
assertion rather than something checked by ear. Repeats are separated by 250 ms
so they are distinguishable; the gap is not paid after the last one.

A missing recording is **not** a failed lookup: the definition has already been
printed, so audio failures warn on stderr and leave the exit code at 0.

### The dictionary follows the language (`#23 M2`)

`systemDictionary(lang, warn)` returns the dictionary for a language, and the
three outcomes are each a deliberate answer:

| situation | result | why |
|---|---|---|
| curated books installed | search them, in curated order | the mode is correct end to end |
| the private surface is gone | the NULL search, loudly | degrade to the pre-`#23` tool, not to a crash |
| nothing curated for this language | the NULL search, loudly | guessing is how a tiebreak picks a thesaurus |

**Metadata NARROWS, a curated list DECIDES.** Only the first step is derivable:
"indexes L, monolingually" is a fact `DCSDictionaryGetLanguages` reports, while
"is a general dictionary rather than a thesaurus" is a judgement nothing in the
metadata supports. Measured: requiring every language pair to be L→L leaves
exactly one candidate for `es` and SIX for `en`, two of them thesauruses and one
an accessibility dictionary — and a deterministic smallest-identifier tiebreak
picks the accessibility one for English and the *bilingual* Oxford for Spanish.
Deterministic and wrong is still wrong.

**Curation is an ordered LIST, not one book,** and that is not a convenience.
Selecting NOAD alone was the first shape and it lost `iPhone`, `iPad` and
`MacBook`, which are Apple Dictionary entries — a real regression, caught by
`TestFixturesMatchLiveDictionary` going red. Selecting *nothing* is the other
failure and a worse one: with Spanish dictionaries enabled, the NULL search
answers `madrugar` in ENGLISH mode (verified on this machine), which is exactly
the Done-when row the milestone exists for. Same-language books in preference
order satisfy both, because every one of them indexes L→L and so cannot leak.

**Every pair must be L→L, not any.** Gran Diccionario Oxford has an `es→es` pair
*and* an `en→es` one; "any" would accept it as monolingual Spanish and put
English glosses back in a Spanish session.

**A CFSet, not a CFArray.** `CFArrayGetValueAtIndex` on the result does not
return garbage — it raises `-[__NSCFSet objectAtIndex:]: unrecognized selector`,
an uncaught ObjC exception in a cgo frame where the cause is not obvious. The
unspecified iteration order that follows is why selection is by identifier, never
by position or display name, and why `chooseDictionary` is pinned
order-independent.

**"No entry" and "no dictionary" are different answers** (`dcs_lookup_in` status
1 vs 3). "This word is not Spanish" is a correct result; "the Spanish dictionary
is not installed" means fall back rather than report an absence you cannot vouch
for. Collapsing them makes a missing dictionary look like a missing word.

**The selection symbols are private** — absent from the SDK header, which
declares two functions and says of the dictionary argument *"not supported for
Leopard. You should always pass NULL."* True of the header, false of the
framework. They are `dlsym`'d at run time so a disappearance degrades to the
pre-`#23` NULL search. The list is `dcsPrivateSymbols`, and it is deliberately
not restated as a count anywhere: "the nine symbols" was repeated into four
documents and was wrong in all four — nine is how many the issue's survey FOUND,
while the resolver needs three. `TestPrivateDictionarySurfaceStillResolves` walks
the list member by member, which makes the degradation loud for a maintainer
since it is deliberately silent for a user.

**The news feed is gated the same way (D6).** `httpFeed` hardcodes
`hl=en-US&gl=US&ceid=US:en` — English by construction — so it is consulted only
for English, and other languages take their examples from their own dictionary
entry. That is also why `usage/` has no language dimension: nothing writes it
outside English. If `#10` or `#18` makes the feed language-aware, scoping the
cache becomes required, and that is the moment to add it.

**The curated languages are named below, not counted here** — a count is a
restatement that goes stale silently, and this sentence said "three" from four
lines outside the guarded span, where nothing could check it. The list is a
judgement rather than a rule — nothing in the metadata separates a general dictionary from a
thesaurus, so `chooseDictionary` NARROWS by metadata and `curated` DECIDES.

<!-- curated-languages -->
| language | book | notation it writes |
|---|---|---|
| English | NOAD + Apple Dictionary | IPA, always: `\| ˈrekərd \|` |
| Spanish | Larousse *Diccionario General* | **none** — phonemic orthography, so there is nothing to write |
| Italian | *Devoto-Oli* | **not a transcription** — syllabification with stress, `(cià·o)`, `(pìz·za)`. `isPronunciation` declines it, correctly |
<!-- /curated-languages -->

`#31` measured French and German too, and did NOT curate them. The parser is
NOAD-shaped in two closed vocabularies — `posWords` is English (`noun`, not `nom
masculin` / `Substantiv`) and `sectionWords` is English (`ORIGIN`, not
`ETIMOLOGIA` / `HERKUNFT`) — so over five common words each it finds 245 senses
in Devoto-Oli against NOAD's 153, but **six** in `fr.Multi` and **five** in
`de.DDDSI`, whose longest single undifferentiated blob runs 2,811 and 4,404
runes. `Haus` puts its grammar table, synonym list and etymology in one example.
That is `#34`. Two facts worth inheriting from it: `fr.Multi` is **Québécois**,
not France French, and German's Duden field is real but LOSSY — `Wạsser` and
`ˈkatsə, Kạtze` survive, while a long vowel's underline does not, so 9 of 15
sampled entries render the bare headword.

**A standing limitation: the raw-notation ratchet is ENGLISH-ONLY.**
`TestNoRawPronunciationNotationSurvives` sweeps every captured language since
`#31`, so `es` and `it` are checked at CORPUS width — over whatever
`testdata/entries/<lang>/` holds, which `TestEveryCuratedLanguageHasACorpus`
requires to be non-empty. Not a number here: this sentence read "six fixtures
each" while `entries/es/` held five.

The live ratchet in `live_property_test.go` is the one that sweeps at DICTIONARY width,
and it walks `/usr/share/dict/words` against `systemDictionary(DefaultLang)`.
There is no Italian or Spanish word list on the host, so neither language has a
ratchet at that width. This is stated rather than fixed: `#23 M2` added Spanish
under exactly the same condition, and inventing a word list to close it would be
a bigger decision than the gap warrants.

## Source pronunciation (`#29`)

`define -pron fr arrondissement` plays the French recording while the deck, the
dictionary and the highlight set stay English. `/pron fr` is the in-session
form: it replays the current word once and leaves no mode behind, which is why
it is an ACTION where `/sound` and `/lang` are settings.

**One source for the policy text, and this page consumes it:**

<!-- pron-help -->hear THIS lookup in another language without switching the session: -pron fr arrondissement. The entry's ORIGIN says which; at the prompt /pron alone reads it for you. Falls back to the session's recording, and says so, when the source has none<!-- /pron-help -->

**The language is DECLARED, never inferred, and that is measured rather than
inherited.** NOAD writes the two cases identically —

```
arrondissement  ORIGIN French, from arrondir 'make round'.
police          ORIGIN … from French, from medieval Latin politia …
```

— and the CDN does not discriminate either: `police_fr_fr`,
`restaurant_fr_fr`, `garage_fr_fr`, `machine_fr_fr`, `unique_fr_fr`,
`genre_fr_fr`, `nuance_fr_fr`, `montage_fr_fr` and `bureau_fr_fr` are **all
200**. So "try the origin language and fall back" would silently replace the
English recording for a large class of fully naturalised words. This re-derives
`#23`'s rejection of inference from new evidence at a narrower scope.
`TestCDNStillCannotTellALoanwordFromANaturalisedOne` is the live row; if it ever
fails, the decision is worth reopening.

**The source recording is keyed on the source ORTHOGRAPHY**, so the spelling is
an input beside the language:

```
jalapeno_en_us  200   ← what define asked for before #29
jalapeño_es_es  200   ← the Spanish recording
jalapeno_es_es  404   ← the same word, Spanish locale, unaccented
```

`SourceSpellings` answers it from three places: the HEADWORD, the `(also …)`
alternatives that differ from it **only by diacritics**, and the typed word as a
last resort. Both dictionary sources are needed because NOAD files the accent on
either side of the headword — `jalapeño`, `piñata`, `Señor`, `cliché` and
`fiancé` are headwords, while `café`, `naïve` and `façade` sit under unaccented
ones. Accented spellings are tried first: headword-first is right 5 times in 8,
non-ASCII-first 8 times in 8.

**`(also …)` is not a spelling list**, which is what the diacritic filter is for.
Surveyed across 400 live entries it holds phrases (`(also good as gold)`),
compounds (`(also jalapeño pepper)`), derivatives (`(also naïveness)`) and real
English variants (`(also advisor)`, `(also caldron)`, `(also convertor)`) — each
worth two wasted requests if admitted. The filter took the three real gains and
nothing else in that sample.

**A known limitation, recorded so it is inherited rather than rediscovered:**
`role` has a French recording (`rôle_fr_fr` is a 200) that no rule here reaches.
NOAD heads the entry `role`, offers no `(also rôle)`, and spells the accented
form only inside ORIGIN — *"from French rôle, from obsolete French roule
'roll'"*. Mining ORIGIN would be a parsing problem rather than a fourth lookup:
that one sentence offers three candidate tokens.

**Coverage is partial, and a miss is REPORTED rather than silent.** `hotel` and
`debut` are 404 on `fr_fr` and 200 on `en_us`; Italian (`ciao`, `pizza`,
`espresso`, `opera`) and Japanese (`karaoke`, `tsunami`) have no recordings in
this generation at all. So `-pron it ciao` plays the English recording and says
`no it recording for ciao; played the en one`. The line is written AFTER the
fetch, from the URL that answered — it survives on a pipe, so it is a record and
a record has to be true.

**The locale comes from `#27`, unchanged.** `voiceFor`, `localeFor`,
`defaultLocale` and `applyVoice` are untouched by `#29`; the source voice is
`voiceFor(pron, opt.locale)`. So `-pron es` builds `es_es` and `-pron es -locale
us` builds `es_us`, and there is no second locale policy to keep in step.

**`-pron` never enters `options`, and that is load-bearing.** `opt.voice` is a
session value `applyLang` re-derives on every `/lang`; an override stored there
would survive the switch and ask for `fr_fr` recordings in a Spanish session.
It rides on `replCommand` instead, beside `literal`, because a per-line modifier
is what it is — which is also why `-pron` with no word is refused rather than
quietly made session-wide.

## Conformance

Live checks sit behind `//go:build conformance` and run **on demand, not in CI** —
they need a host with NOAD installed and reachable network, neither of which
belongs in `merge-check.yml`.

Every seam has one, and each pins the assumption that seam rests on:

| check | asserts |
|---|---|
| `dict_conformance_test.go` | live lookups still byte-match every fixture |
| `fetch_conformance_test.go` | the CDN path survey still holds (2022 generation dominates) |
| `player_conformance_test.go` | `afplay` **blocks** until playback finishes |
| `news_conformance_test.go` | the live RSS feed still parses, and its terms still say personal use |
| `reflect_conformance_test.go` | the live model still answers in the shape the parser expects |
| `live_property_test.go` | the no-data-loss predicate holds over the WHOLE dictionary, not a sample |
| `pty_conformance_test.go` | the raw-mode loop on a REAL terminal — `--play`'s CRLF defect (#6) was invisible to every non-pty test, and `TestPTYPlayGradeFirst` (#24) drives the grade-first flow the same way |

**A skip reads as green, so green has to be made to mean "it ran".** Every suite
above routes its dependency check through `conformance.SkipOrFail`
(`internal/conformance`): absent dependency SKIPS by default, and FAILS under
`CONFORMANCE_STRICT` — the mode for CI and for a close that has to mean
something. Checks about the dependency's *shape* — a drifted fixture, a feed that
stopped parsing — are not routed there and stay hard failures in both modes; the
package doc names all four classes.

**It is a repo-wide package, and enforced rather than swept**, because four
review rounds went to this one rule and each fixed only the sites its grep
reached: one pty site while six suites skipped; then all seven, enumerated by a
pattern that could not see the mirror defect; then a carve-out that excluded a
file by name; then a sweep over `cmd/define` under a README claiming `./...`,
where the documented strict command measurably reported `ok` with `internal/llm`
skipped. `TestEverySkipIsRoutedOrWaived` now walks the tree and fails on any
unrouted, unwaived skip — the same move that ended the doc-sweep family, applied
to a guarantee instead of a sentence.

`player_conformance_test.go` is the least obvious and the most load-bearing: if
`afplay` ever returned immediately, three *overlapping* sounds would satisfy
`fakePlayer`'s count and every other test here — "plays three times" would be
true on paper and wrong in the room. (Named, not numbered: this sentence said
"the third" while the table above it grew from three rows to seven.)

```sh
go test -tags conformance ./cmd/define/   # must run UNSANDBOXED; skips what it cannot reach
CONFORMANCE_STRICT=1 \
  go test -tags conformance ./cmd/define/ # CI / close: a skip is a failure
cmd/define/testdata/capture.sh            # re-capture the corpus
```

Trigger: after a macOS upgrade, or when a parse/audio bug is reported.
`DCSCopyTextDefinition` returns *silence, not an error*, without real access to
`/System/Library/AssetsV2` — which is why `capture.sh` enforces a byte floor.

## Usage sources: real sentences for a word

**Two sources, one shape.** `Usage` is a sentence worth showing, tagged by
provenance — `news` from Google News RSS, `noad` from the dictionary's own
examples. One shape because the consumer (#10 authoring) wants "sentences for
this word", and two parallel lists would push the join into every caller. NOAD's
examples cost no network and are not current, which is exactly why they are here:
the feed's thematic collapse is measured and real (10 of 14 `sycophantic`
headlines were about AI chatbots), and a word taught only through this week's
news cycle is taught narrowly.

**Raw is cached; usable is derived.** `store.NewsItem` is what goes to disk — the
feed's own words, knowing nothing about filtering or provenance — and `Usage` is
computed from it at read time. The split is the point: it lets
`containsWord` improve and every word already cached improve with it, with no
re-fetch, where caching the filtered result would freeze today's judgment onto
disk. The plan's first draft had `Usage` in `package main` while `store.Store`
returned it, which cannot compile — the Critical that forced the split was really
the design telling the truth.

**`containsWord` is the one place judgment lives, and it delegates.** The feed is
queried with the word quoted and still returns items that do not contain it
(measured: 12–99 matching out of 41–100), so the filter is load-bearing. It calls
`highlightSpans` with a one-word vocabulary rather than matching itself — that is
the whole matcher, `store.Key` normalisation and phrase gaps and joiner trimming
included. Reimplementing any part would put the same word in two states on one
screen: a headline rendering green while the filter rejected that headline.
`TestContainsWordAgreesWithTheHighlighter` is the pin.

**Headlines carry attribution, and stripping it is not "cut at the last dash".**
Every Google News title ends `" - Publisher"`, and `<source>` names the
publisher — so the suffix is removed only when it IS that publisher. A headline
like `Ephemeral - a study in impermanence` keeps its dash. Found by capturing a
real feed rather than reasoning about one.

**The feed is licensed for personal, non-commercial feed-reader use** — the
copyright notice is in the feed body itself. A personal vocabulary tool fits;
nothing here redistributes content.

**The cache has three outcomes, and modelling two is the easy mistake.**
`cachingFeed` writes `usage/<slug>.yaml` beside `words/` and `events/`:

1. a fetch that succeeds **with items** is cached and served;
2. a fetch that succeeds with **zero** items is *also* cached, with its
   timestamp — some words are simply not in the news, and that is a real answer.
   Not caching it re-fetches forever for exactly the words the feed is worst at;
3. a fetch that **fails** is never cached, so a network blip cannot become a
   permanent empty answer for a word.

Because (2) is cached, entries carry a fetch time and go stale after `cacheTTL` —
and **a stale entry whose refresh fails is served anyway**. That is what keeps
"works offline" true while letting a word that had no news last week pick some up
this week. The file is a WHOLE-FILE record like `words/`, written through
`writeBytesAtomic` and therefore untearable; a file corrupted from outside is
warned about and read as never-fetched, which costs a re-fetch and nothing more.

**`bothSources` is the seam, and a feed outage degrades to the dictionary**
rather than propagating. A nil news source is the same case — "no feed
configured" is not an error, the shape every seam here uses for absent. Under
`DEFINE_NO_CAPTURE` the seam still exists, cached in memory: that flag means
"write nothing into this directory", not "the feed does not exist", the same
reading that gives that path a `memHistory`.

**Nothing user-facing consumes it yet, deliberately.** #10's authoring step is
the consumer. A debug command was planned and dropped: `commandCtx` carries no
`context.Context` and no dependency, so a fetching command means widening the
contract every command shares — for a debug affordance, ahead of the real
consumer. The live conformance check prints what it fetched, which is how a
person looks at real output in the meantime.

## Scheduling: what is worth attention today

**Leitner, not SM-2, and the reason is explainability.** For a personal tool
*"why is this due?"* must be answerable in one sentence, and an ease factor
cannot be. The sentence is **"each correct recall multiplies the wait by 1.6"**:
`IntervalDays(box) = floor(1.6^box)`, giving 1, 1, 2, 4, 6, 10, 16, 26, 42, 68,
109, 175, 281 days and onward.

**The interval is COMPUTED, in exact integers.** `8^box / 5^box` in `int64`, not
`math.Pow`. The reason is the one `#7` established for its PRNG and hash:
`math.Pow` is not guaranteed bit-identical across architectures, and a schedule
that differs by platform is one this repo cannot pin in a table test and a
learner cannot reason about. A rational ratio in integers is exact everywhere and
needs no import, so `schedule`'s allowlist is untouched.

**Boxes 0 AND 1 are both one day, and that is the most valuable rung.**
`floor(1.6^0)` and `floor(1.6^1)` are both 1, so a new word is seen on day 1 and
again on day 2 — which is when the forgetting curve is steepest. It reads like a
rounding artifact, and the "fix" (indexing from `1.6^(box+1)`) removes the entire
acquisition density the ladder exists to provide. `TestTheFirstTwoRungsAreBothOneDay`
exists because it looks like a bug cold.

**There is no pedagogical ceiling, and that is what makes a large deck
affordable.** A word in box b costs `1/IntervalDays(b)` reviews per day, so with
a CAPPED top rung the stock of parked words grows linearly forever and no
admission rate is sustainable — at 7 new words a day against a 90-day cap, the
accumulated stock alone costs ~55 reviews/day after two years. With intervals
that keep growing, a word of age τ is reviewed at roughly `1/τ` per day and the
total load integrates to `a·ln(T)`: logarithmic, so a fixed daily budget supports
a nearly constant new-word rate indefinitely.

`ladderLimit = 20` is therefore an ARITHMETIC bound and not a ceiling: `8^21`
overflows `int64`, and box 20 is reached only after 20,135 days of correct
answers — 55 years, pinned by `TestTheClampIsUnreachable`. It is named
`ladderLimit` rather than `maxBox` because `Progress` carries a `MaxBox` field,
and `p.Box < maxBox` versus `p.Box < p.MaxBox` are both valid Go with opposite
meanings — the first would grant every word a permanent express lane, silently.

**What the deck costs is COMPUTED and SHOWN.** `DailyLoad` is `Σ 1/interval` over
the deck, and it takes the DECK rather than the folded progress map: `Fold`
returns an entry only for words with a review event, so summing the map would
report a tenth of the cost of a mostly-unreviewed deck — understating it exactly
when the warning matters most. `SustainableNewWords` divides the leftover budget
by `reviewsInFirstYear()`, which is derived by walking the ladder rather than
written down. The sitting summary is its first reader; `#41`'s status bar is the
second.

**Schedule state is DERIVED from the event log, never stored beside it.** This is
the load-bearing decision of `#5`, and `store/event.go` had already written the
rule for the whole store: the log is *"deliberately the ONLY record of activity
... storing counters alongside would create a second source of truth that
drifts."* A `Box` field on `store.Word` would be exactly that — and it would go
wrong invisibly, a hand-edited or partially-written deck file disagreeing with
the events that produced it. `Fold` is one pass over a log the session already
reads. If it ever costs real time the answer is a cache keyed by the log's
length, not a stored field.

`Fold` **applies** `Answer` rather than reimplementing the transition, so
"correct promotes, wrong demotes one box and resets the streak" has one encoding.
Its ORDERING CONTRACT is stated rather than fuzzed: it consumes events in the
order `store.Store.Events` returns them and is **not** order-independent — two
reviews sharing a timestamp with different outcomes fold differently by order,
and `ReviewEvent` carries no tiebreaker. What `FuzzFold` asserts instead holds
over the whole domain: box in range, non-negative streak, idempotence.

Only `EventReviewed` participates. A lookup or a question is *activity*, not
*assessment* — they say what the learner is working ON, which is `#17`'s signal,
not what they know.

**Demotion HALVES the box, and the express lane is its other half.** One
sentence, and it scales where a fixed step cannot: box 12 (281 days) falls to box
6 (16 days), a real relearning interval, while box 2 falls to box 1, barely a
nudge. Halving alone would make a single slip cost most of a year, so `Progress`
carries `MaxBox` — the high-water mark — and a correct answer below it climbs TWO
rungs instead of one. Storage strength survives when retrieval strength does not,
which is why relearning is faster than learning; Ebbinghaus called it savings.

**They are a PAIR and neither works alone**, which `TestRecoveryFromALapse` pins
by walking the whole recovery step by step: removing the lane makes it five
reviews instead of three, and removing the halving means the word never leaves
the top. A test asserting only the end state would have passed on either.

`MaxBox` erodes by one on every lapse, so a word that keeps failing gradually
loses the express lane and is eventually relearned properly rather than being
waved back up forever.

**A two-rung promotion is earned by an OBSERVATION, never a claim.** `Apply`
knows whether the learner revealed before answering, so "correct, cold, in a form
that checked the answer" is something the session saw. Form 2.1's `y` is not
that — it means *"I knew it"* with nobody checking — so `Recall` declares itself
through `play.SelfRated` and never earns it. The flag is computed in `Apply`'s
`InputRune` arm and PASSED to `advance`, because `advance` zeroes `s.Revealed`
before it builds the outcome: reading it there would mark every correct answer
unaided and run the ladder at double speed.

**A day is a LOCAL CALENDAR day.** `Due` compares through `store.StartOfDay`,
never `N × 24h`: a learner who reviews at 9am Monday and sits down at 8am Tuesday
must find a box-0 word due. `store` owns that helper because it owns `Clock` —
and `#5` is what finally moved it there, after `#15` had written the same idea
inline twice in two different shapes for `/history`.

**`Mastered` is defined once, exported, and consumed by two.** `#6`'s `--play`
decides what to stop offering; `#8`'s `--stats` reports how many words are known.
Two conditions written separately would drift, and the drift would surface as a
stats screen disagreeing with the review queue. `masteryStreak` is 7 with a
stated reason: reaching the last box takes 5 consecutive correct answers, so any
N at or below that would make "mastered" mean "arrived".

**The package is pure, and that is ENFORCED by three guards.** `schedule` imports
`store` and pure standard-library packages only; one guard reads the import set
against an allowlist, a second greps for every wall-clock reader
(`time.Now`, `time.Since`, `time.Until`, `time.After`, the timer constructors),
and a third holds `store` to its pure types and helpers — allowlisting the
package would otherwise grant the disk along with it. The second exists because
it is the hazard an import list structurally cannot see, since `time` is
legitimately
imported for its types. The allowlist itself was wrong first and fired on `sort`:
it said "exactly two imports" when the claim is "no IO and no hidden clock", and
a guard that reddens on correct code invites deleting the guard. The plan's first draft claimed a test needing a
fake "would not compile", which is false — a Go test may import anything — and
this repo has been bitten repeatedly by facts that lived only in comments. The
caller does the IO: `#6` reads the deck and the log through the store seams, gets
`now` from the injected `Clock`, and writes `EventReviewed` back through the
capture path.

**The queue has TWO TIERS, and the Done-when forces it.** Due words with review
history come first, most overdue first; words never reviewed follow, most
looked-up first. Ranking everything on one "how long since we saw it" axis looks
simpler and is wrong: a word first seen months ago and never reviewed has a
larger age than a word reviewed last week and three days overdue, so the newcomer
would go first and the word actually being learned would wait. That is the
starvation `#5`'s Done-when forbids, and it is why the tiers are separate rather
than one sort key with a clever weight.

A mastered word does NOT leave the rotation — it sits at the 90-day interval and
comes round, which is what keeps mastery from being absorbing: a word never
offered can never be answered wrong, so the count would only grow while recall
decayed. `Mastered` is a status for reporting and presentation, not a removal.
A word not yet due is not offered; and the
DECK is the roster while the log is the history — a word with progress but no
deck entry is not queued, because `--forget` deliberately keeps a word's events
after removing it and resurrecting it here would make forgetting not work.
`budget <= 0` returns nothing: "no budget" is not "unlimited", and the opposite
reading is a way to sit down to four hundred words by accident.

## Review sessions: how a word is asked

**`play` is the second pure package, and `puretest` is why there will not be a
third copy of the guards.** `#5` wrote three purity guards inline; `#6` needs two
of them and `#7`/`#12`/`#13` each add a form package, so they were extracted
into `cmd/define/puretest` — one body, many callers, the shape `storetest.Suite`
already established here. Each guard sees a hazard the others structurally
cannot: an import allowlist misses that allowlisting `store` grants the disk with
it, and no import list can see `time.Since`, because `time` is legitimately
imported for its types.

`schedule` takes all three; `play` takes two, because it names no store symbol —
and a store-symbol guard with an empty allowlist would fatal on finding nothing
to check, which is right, since it would be asserting nothing.

**The guards have their own tests, against committed known-bad fixtures**, and
that is not bookkeeping: a guard nothing has ever seen fail is indistinguishable
from a guard that cannot fail. `puretest/testdata/impure` imports `os` and calls
`store.NewYAML`; `testdata/clocky` imports only `time` and calls `time.Since`,
the case an import allowlist structurally cannot catch; `testdata/pure` is the
known-good case, a fixture rather than a real package so production changes
cannot silently change what the positive assertions mean. The guards take a
`puretest.T` interface rather than `*testing.T` precisely so a recorder can
capture the failures instead of failing.

**`Outcome.SessionDone` says the session ended, whatever the outcome's kind.**
The last answer produces `OutcomeRecord` and finishes the queue, so a caller
holding only an `Outcome` would otherwise have to consult the `Session` too —
making "did we finish" two facts in two places.

**A deck whose words all fail to look up is NOT "nothing due today".** Words were
due; the dictionary is the problem. Saying nothing is due would send the learner
away believing their deck is clear, so that path reports what happened and exits
1. It used to have a sibling — "lost the terminal after playback", the same exit
code because it was the same failure — and `#41` deleted it: playback no longer
hands the terminal back, so there is no re-entry left to fail.

**`Question` is the whole of what a session knows about a form.** `Word`,
`Prompt`, `Reveal`, `Grade`, `Keys` — and `Grade` lives on the FORM, which is
what makes "adding a second form requires no change to the loop" a property
rather than a promise. Form 2.1 grades `y`/`n`; form 2.3 grades digits; the
session never learns either.

**`Keys` joined that list when the second form shipped, and the reason is the
kind of bug an interface exists to prevent.** The line under the question —
"y = got it, n = missed it, …" — was a CONST in the loop. It named form 2.1's
keys, so the moment form 2.3 was asked the learner was told to press `y` on a
screen where only `1`–`4` did anything, and no test could see it because every
test typed the keys the const named. A form describing its own keys is the only
arrangement in which that cannot recur. The loop still owns the SESSION's
reserved half (`d`, Ctrl-C) and appends it, because those are true whatever form
is asking and a form restating them would be two owners of one fact.

### The sitting is a frame (`#41`)

`--play` draws through `console`/`display` exactly as the editor does, and the
divergence `#30` D5a predicted is closed. `newConsole` is the shared builder
both loops now call, with the screen constructor as its one parameter — the
first cut of it was a verbatim copy of `replRaw`'s
construction: alternate screen, mouse reporting, a screen, `watchResize`,
`onceHandBack`. What that bought, in the order it matters:

- **A long reveal PAGES instead of scrolling the word away.** Form 2.3's reveal
  is the whole rendered entry, which on `run` or `bank` is several screenfuls;
  before frames the word being asked about was simply gone off the top. The loop
  intercepts PageUp/PageDown and the wheel and calls `view.Page`/`view.Scroll` —
  BEFORE `toInput`, never inside it, because a viewport is not something the pure
  `play` package may learn about.
- **A status bar, pinned.** `sittingBar` formats it and `finish()` formats its
  summary through the same `costPhrase`, so the two cannot word the `-count`
  assumption differently.
- **Three surfaces are SHARED with the editor rather than copied**, and each
  was a copy first: `newConsole(ctx, d, sess, stdout, newScreen)` builds the
  terminal for both loops with the screen constructor as its one difference,
  `viewportGesture(view, k)` owns which keys move the view and which way a page
  goes, and `wrapWritten` owns the wrap. `#40`'s board is the third caller of all
  three.
- **The question is a BUFFER LINE and the keys are the LIVE EDGE.** The old
  `draw()` wrote the question, the reveal and the keys on every call, which is
  right for a scrolling terminal and would file a copy of the question per
  keystroke against a line buffer. The loop tracks the written index and writes
  on transition; `livePrompt` returns the keys, which are painted and never
  filed.

**A SITTING'S WORDS ARE CLICKABLE, through the same registry the editor uses
(`#38`).** `#30` Done-when 7 asked for *"one mechanism, so a third consumer is a
row rather than a new feature"*, and until `#38` the switch on `RegionKind` lived
inside `runEditor`'s own closure, where a second loop could only copy it.
`playRegion` is that registry lifted out: both loops call it, and the INDICATOR
is its one parameter, because the editor's is erasable and a sitting's is the
record-shaped `defaultIndicator`. A kind with no row there draws an underline
that does nothing, which `TestEveryRegionKindIsActionable` catches by deriving
its loop from `numRegionKinds`.

- **The prompt word is line 1, column 0 of the write the loop already makes.**
  Both forms put the headword on their first line, so there is nothing to search
  for and no offset to survive a wrap.
- **The reveal's regions come from `Render` and are SHIFTED into the coordinates
  of what is written.** A form's reveal is larger than the render inside it —
  `Choice.Reveal` names the right option and the learner's pick first — so the
  loop locates the render in the reveal rather than counting from a formula,
  which would be a second copy of a layout the form owns.
- **A click never answers** (D8). It stops before `toInput`, beside the viewport
  gestures, so `play.Apply` never learns a mouse exists. A click that recorded a
  review would corrupt the schedule silently, which is the worst kind of bug
  here: the damage is to data the learner cannot see.
- **The click map is DROPPED rather than misplaced when the wrap would move it.**
  `#41` put a wrap between the caller and the buffer, and a region's column is
  relative to the text it was computed from — so `writeClickable` applies the wrap
  first and passes the regions along only if it changed nothing. At a sitting's
  own width it changes nothing; after a NARROWING resize the text still arrives
  whole and the underlines stop until the next question is written. An underline
  that plays the wrong word is worse than no underline, because losing an
  affordance is visible and a wrong click is not.

**A SITTING REFUSES rather than degrades, and it settles that before doing any
work.** `--play` needs stdin to be a terminal (a review is a conversation, and
piped input would answer questions it never saw), stdout to be a terminal (a
redirected one would collect frames at a fabricated 80 columns), and `-no-color`
to be off (that flag means "emit no ANSI", which main.go's own comment extends to
cursor control, for terminals that mangle escapes). All three are checked before
the deck is read — the same rule usage errors follow — and all three refuse,
because there is no line-mode fallback for a sitting and pretending otherwise
would write the frames anyway. The editor degrades instead, by ROUTING to
`replLines`; that option does not exist here.

**Everything the loop writes is wrapped at the moment of WRITING, not of
rendering (`wrapWritten`).** This took three findings in one family to state.
`Paint` CLIPS a buffer line at the terminal's width — letting it wrap would make
the frame a row too tall and the terminal would scroll every row the sitting
placed — while `--play` renders its text when the queue is built and writes it
much later. So every pre-rendered artifact carries a width that may already be
wrong: an option gloss at the startup width (the operator found this one), the
same lines after a narrowing resize, and the rendered definition a reveal writes.
Wrapping one line-kind at a time is what produced three findings; the loop routes
the question, the reveal, the drop notice, the summary and its diagnostics
through one function, and the resize case keeps `opt.width` current. A line that
already fits is returned untouched, so `Render`'s own wrapping passes through.
The question already on screen keeps the wrapping it was written with, exactly as
the editor's scrollback does — and nothing is lost by it, because the clip
happens at paint and the whole text is still in the buffer if the window widens.

**`newPinnedScreen` versus `newLiveScreen`, and the difference is a decision.**
`Paint` writes the visible frame, then the prompt, then the footer — so a
five-line question on a forty-row terminal put the bar at row seven. A pinned
screen pads the buffer region to its full height at PAINT time, so the footer
sits on the bottom row. Blank ROWS, never lines: the transcript must not gain
rows because the terminal is tall. The editor keeps the unpadded constructor,
because a REPL prompt belongs directly under the last output.

**Adopting frames DELETED the playback dance, and that was a Critical rather than
a tidy-up.** Every reveal used to `restore()`, play the pronunciation in cooked
mode, and `enterRaw` again. `enterAlt` is opt-in on `rawSession` and `restore()`
leaves the alternate screen, while `enterRaw` returns a session with `alt` false
— so a frame-drawing sitting would have lost the alternate screen on its FIRST
reveal and painted every frame after it over the user's scrollback. Playback now
stays raw: under the frame model the `♫ playing 3×` indicator is a frame write,
and `screen.Write` already honours its `\r\x1b[K` erase by taking the open line
back. `TestPTYPlayKeepsTheAlternateScreenAcrossAReveal` is the pin.

**The bar's figures are read ONCE per sitting and updated in memory.** `Deck()`
reads a file per WORD and `Events()` a file per DAY of history, so recomputing
per answer is thousands of file reads per question with a person waiting.
`todaysQuestions` returns what it already computed as a `sittingDeck`, and the
loop applies `schedule.Answer` to it through `schedule.GradeOf` — the same rule
`Fold` reaches through `gradeOf`, so the figures a sitting SHOWS cannot drift
from the ones the next sitting DERIVES. A drop removes the word from the copy
too, so the bar cannot charge for a word the learner just curated away. `finish`
takes the figures rather than re-reading: `#39` T7's reasoning (the learner
should see the AFTER-today figure) survives, because the in-memory copy already
is that figure.

### Form 2.3: choosing a definition

**The distractors are the learner's OWN deck, which is a pedagogical choice
before it is an offline one.** A model could invent plausible wrong answers; the
words the learner is actually confusing right now are better wrong answers, and
they cost no key and no network.

**The three distractors vary along axes NOAD labels itself**, and that is what
makes a miss informative rather than binary. `Axis` is the reduced taxonomy —
`AxisDomain`, `AxisRegister`, `AxisGeneral` — and the reduction is measured, not
a shortcut: NOAD prints domain (`Law`, `Grammar`, `Nautical`) and register
(`informal`, `archaic`, `dated`) inline at the head of a sense, so both are free
to read and neither creates a second defensible answer. Near-synonym collapse and
connotation need semantics, hence a model, hence they belong to `#12`/`#13`,
which have a model veto. This form has none by design.

**Selection fills the SCARCE axis first.** Counted over the committed corpus,
register labels outnumber domain roughly five to one and only 12 of 34 entries
carry a labelled sense at all — so a set that filled register first would almost
never leave a domain candidate unused. A second pass tops up from whatever
remains, because three axes and three slots only line up when the deck has all
three; without it a deck with no `Law`-labelled word would return three-option
questions forever.

**The near-synonym guard is the dictionary's own cross-references.** "Never a
near-synonym" named no mechanism until it was noticed that NOAD defines close
words THROUGH each other — `sycophantic` is glossed *"behaving or done in an
obsequious way"*. So a candidate is excluded when either headword appears in the
other's gloss, on a word boundary, above six characters. Six is measured: three
corpus glosses contain "thing" and every one is a coincidence. It is a REDUCTION
and not a proof — two deck words can be near-synonyms NOAD never links — and the
residual is accepted because the options are definitions, which two different
words rarely share.

**`readGloss` walks the head of a gloss rather than matching a prefix**, and the
walk exists because the obvious design was measured and found wrong. Labels
usually lead, but a grammar bracket or a parenthetical can come first
(`[no object] Military (of a soldier) …`, `(the runs) informal diarrhea.`), and
NOAD stacks REGIONAL labels in front of real ones (`North American English
informal …`). A prefix match would have called every one of those unlabelled,
losing exactly the senses the axes are built from. Regional labels are recognized
so they can be scanned past, but are not an axis — the settled taxonomy has three
values, and folding "where" into "tone" would blunt the finding.

**Not every `Sense.Gloss` is a definition, which the plan had asserted it was.**
`bank`, `complete`, `concrete` and `defenestrate` each carry a sense whose gloss
is exactly `[with object]`; `man` carries `(plural men /men/)`; `alewife` and
`bases` carry cross-references (`another term for menhaden`). `readGloss` returns
`Usable`, and an unusable gloss is never an option — one reading "[with object]"
would make the form look broken.

**The seeded shuffle and the pool sample are hand-rolled, not `math/rand`.**
The Done-when claims a fixed seed reproduces a question, and `math/rand`'s
sequence for a seed is a property of the Go runtime rather than of this repo. A
xorshift64 defined in `play` is pinned by this repo's tests, which is the
guarantee actually being claimed. The pool is sampled with a partial
Fisher-Yates, capped at 40 lookups per sitting — per-question it would be one
dictionary lookup per deck word per due word, quadratic in a deck that only
grows.

**A wrong answer records WHICH option, through an optional interface.**
`Outcome.Axis` carries it, filled from `Missed` — a capability, never a form, so
`Apply` still contains no branch that knows what a `Choice` is. Form 2.1 does not
implement it, because a failed recall genuinely has no kind, and requiring every
form to answer would make the interface lie for the one that cannot. A correct
answer reports `AxisNone`, which stringifies to `""`, which `omitempty` drops —
so "no axis on a right answer" is enforced by the type rather than by a branch a
caller could forget. `ReviewEvent.Missed` sits ABOVE `At`, because the
torn-record rule leans on `at:` being the last key on disk.

**Three things send a word to form 2.1**, and the list is DECLARED
(`fallbackReasons`, `optionpool.go`) because it was being hand-maintained in the
code, the README and here, and had already drifted to two, two and one:

1. **the deck has no other word to draw on** — a learner three lookups in has a
   deck of three, which is the normal early state of the tool rather than an
   edge case;
2. **the entry is nothing but cross-references** — `bases` is "plural form of
   base1", so there is no definition to be the right answer;
3. **the entry defines a different word** — NOAD redirects derived forms to
   their base, so `bargainer` returns the `bargain` entry.

The fallback is invisible to the learner and the sitting stays the length the
schedule asked for.

**Reason 3 is the one that had to be learned.** Form 2.1 shows the whole rendered
entry, DERIVATIVES line included, so a redirect is harmless under it; form 2.3
asserts that ONE gloss IS the word's meaning, records `Correct`, and promotes the
word on that basis. The same dictionary behaviour is fine under one form and
wrong under the other, and the assumption was inherited rather than re-examined
when the form changed. `entryDefines` gates it, and both PRODUCERS of a
(word, gloss) pair call it — `optionCandidates` and `targetCandidate` — rather
than the caller, because gating the caller caught the target and left the
distractor path open for a round.

**An option set also dedups on GLOSS, not only on word.** Two deck keys can
resolve to one entry: `jalapeño` and `jalapeno` are separate keys (`store.Key`
folds case and whitespace but not diacritics) and the dictionary answers both
identically. Keyed on word alone, a set then carries byte-identical options with
one marked correct — so a learner who reads both and picks the other is recorded
as a miss, given an axis they never chose, and has the word demoted for being
right. Nothing upstream can catch it: both words are real deck entries whose
entry genuinely defines them, and neither headword appears in the shared gloss,
so `crossReferenced` is blind to it. The option set is the only place that can
see two options saying the same thing.

**The seed drives SELECTION, not just the order options appear in**, and the
first version got that wrong in a way both Done-when rows were green on. The
pool is built once per sitting, so scanning it in fixed order made every
question take the same first-matching candidate: measured at 17 of 20 questions
sharing one distractor set, after which a learner answers the rest by
elimination. "Exactly one correct option" and "deterministic under a fixed seed"
are both true of a sitting that asks the same question twenty times, which is
why neither noticed. Both passes now walk a seed-shuffled index permutation —
a permutation rather than a shuffled slice, since the pool is shared across
questions and reordering it would make each selection depend on the ones before.

**Revealing before answering shows the right option, and that is inherited
rather than chosen.** Space and Enter are reserved by the session, and on form
2.1 they show a definition the learner then self-rates against; on form 2.3 they
show which option is correct, and nothing stops the learner pressing that digit
for a `Correct`. It is a personal tool and the only person deceived is the one
doing it, so the behaviour is left alone and named in the README instead of
being special-cased per form — a form-specific reveal rule would put key
semantics back inside the session, which is exactly what `Question` exists to
prevent. `TestSessionIsFormAgnostic` drives the same table through a fake
form using entirely different keys, and asserts that 2.1's own keys mean nothing
to it — the only honest way to test that claim before a second form exists.

**`main.Key` stops at the package boundary, and the compile error was the design
telling the truth.** The plan's first draft had `Grade(k Key)`, which cannot
compile: `Key` lives in `package main`. `main` owns the terminal and knows Ctrl-C
is `0x03`; `play` must not, or it could not be tested without one. The loop
decodes once and hands over `play.Input` — a rune for graded keys, and CONTROL
intents (`InputReveal`, `InputQuit`) as their own kinds.

**Three verdicts, not two.** A learner who skips has not got it wrong, and
recording a skip as a miss would demote the word through `schedule.Answer` —
punishing honesty about a word you half-know. **The skip filter lives in exactly
one place**, `advance`: only the session knows a verdict, so a loop that inspected
verdicts would be re-deciding what the state machine already decided. `Apply`
emits `OutcomeRecord` only for `Correct` and `Wrong`, and the loop records
whenever it sees one without ever looking at the verdict.

**`runPlay` owns everything the pure packages cannot.** It folds the log, asks
`schedule.Queue` for today's words, renders each definition, drives `play.Apply`
over the key channel, and performs the outcomes: `OutcomeRecord` calls
`CaptureReview`, `OutcomeReveal` plays the pronunciation. `toInput` is where
`main.Key` stops — Ctrl-C and EOF become `InputQuit`, Enter and space become
`InputReveal`, `d` becomes `InputDrop`, and everything else is a rune for the
form to grade. Those keys are RESERVED from every form — a later form choosing
`d` for "definitely" would find it silently taken — which is why the reservation
is stated in `Question`'s doc comment rather than only living in `toInput`.

**`d` drops the current word, and it is a SESSION action rather than a verdict.**
"This word does not belong in my deck" is true whatever form is asking, so it is
an `Input` kind and every future form gets it free — the same reasoning that puts
`Grade` on the form. It records no review, because dropping is not an assessment,
and the EVENTS stay: `--forget`'s contract, since history is what happened and
cannot be untrue while the deck is the working set the learner curates.

**A sitting draws WHOLE FRAMES, through the same seam the editor uses (`#41`).**
It used to write lines through a translating writer to a scrolling terminal, because in
raw mode a bare `\n` moves down WITHOUT returning to column 0 and a multi-line
definition cascades diagonally across the screen — `#16` built that writer for
exactly this, `--play` shipped without it, and the operator's first real session
found it. The screen places every row itself, so the second writer is gone; see
"The sitting is a frame" below for what owning coordinates bought.

**Cancellation is checked BEFORE the select, not only inside it.** `select` picks
uniformly at random among ready cases, so a cancelled context with a key already
buffered would sometimes grade one more answer after Ctrl-C. It surfaced as an
intermittent test failure, which is the only way a random-choice bug ever shows.

**Reviews record through `Capturer`, never `store.AppendEvent`.** `capture.go`
already stated the rule — capture is the ONLY thing that records, and a second
appender beside it is how that stops being true unnoticed. `CaptureReview` is the
third verb after `Capture` and `CaptureAsk`, it consults `decideCapture` so
`-raw` and `DEFINE_NO_CAPTURE` mean the same thing here as everywhere, and it
never touches the deck — a review that upserted would make reviewing a word count
as looking it up, inflating the count `#5`'s queue orders fresh words by.

**Recorded as it happens**, before the next question is drawn, which is what
makes Ctrl-C mid-session lossless by construction rather than by a flush. That
property is free from the append-only log (`#3`) and any batching would lose it.

**No deck means no session**, guarded on `d.deck == nil` rather than on
`DEFINE_NO_CAPTURE`: that flag is one cause and `openStore`'s `Getwd` failure is
another, so keying on the flag would hand a nil store to the queue builder and
panic on the other path.

**Grading before reveal is THE NORMAL PATH** (#24). It used to be ignored, on the
grounds that a learner cannot rate what they have not seen — true of a
recognition test, false of a RECALL test, which is what form 2.1 is. The learner
rates their own recall, which they know before they check; the definition is
FEEDBACK, not stimulus. Reversing it removed a mandatory keystroke, and the slow
one, from in front of every correct answer.

So `y` records and advances with no reveal and no audio, while `n` records the
miss AND reveals, **staying on the word** — advancing would scroll the answer
past unread, which is the entire reason for showing it. That is the one input
owing the loop TWO effects, and it is why `Apply` returns `[]Outcome`: the
alternative, a flag on `Outcome`, would make "reveal" expressible two ways.

`Session.Graded` is what keeps the next keystroke from grading twice — `Fold`
would read a duplicate as a second review. It is independent of `Revealed`: a
learner can reveal without grading (space) and grade without revealing (`y`).
Note that **Enter and space reach the `InputReveal` arm**, not the rune arm. Both
arms therefore meant the same thing in the graded state — "move on" — and were
first written as two identical bodies; the rule now sits in ONE guard hoisted
above the switch (`if s.Graded && (in.Kind == InputRune || in.Kind == InputReveal)`).
`InputDrop` and `InputQuit` stay outside it: "this word is not mine" and "stop"
are still true after a verdict.

**The prompt lines are consts, and the README is a test-enforced consumer of
them.** `gradePrompt` and `gradedPrompt` (`cmd/define/play_loop.go`) are what
`draw` prints, and `TestREADMEQuotesThePromptsTheLoopActuallyPrints`
(`cmd/define/doc_sync_test.go`) asserts `README.md` contains both verbatim. So
**changing what the learner is told is a two-file edit** — the const and the
README — and forgetting the second fails the build rather than surviving until
someone re-reads the prose.

That convention exists because the `doc-sweep-incomplete` family reached four
findings on this one screen, each found by a human re-reading docs and each fixed
by another sweep. A grep cannot fail a build. It is deliberately narrow: it pins
the two lines the learner reads off the screen and types against, not the prose
around them, which stays free to be rewritten.

`score` splits from `advance` for the same reason: a miss must be tallied without
moving on. **Revealing is idempotent**, so a second reveal does not play the
pronunciation twice. **An empty queue is immediately done** — the message names
its cause, and "nothing due today" is reserved for the schedule genuinely having
nothing.
