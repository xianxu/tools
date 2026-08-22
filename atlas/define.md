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
| `TestRenderLosesNothing` | the 29 captured fixtures | regressions on known shapes |
| `FuzzRenderLosesNothing` | arbitrary strings, corpus-seeded | parser crashes, boundary bugs |
| `TestRenderLosesNothingOverLiveEntries` (conformance) | **every** reachable entry — 70,897 | shapes nobody thought to sample |

The third is what earns the claim "safe against entries nobody sampled"; a corpus
test alone covers only what someone already sampled. Content loss over the live
sample went 7% → **0%**, measured over **all 70,897** reachable entries — not a
sample. The 530 non-Latin entries from other active dictionaries are counted and
excluded, not silently skipped. The sweep runs in ~38s.

Sampling is why this section had to be rewritten three times: at 2,749 entries
(3.8%) the raw-notation count read 0, and at full width it was 27.

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
published "0%" while 2.0% of entries were still rendering raw `|`. Both now
measure **0** over the live sample. The lesson generalises past the first fix —
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
- **A prose numeral that continues a sense sequence is taken as a sense number.**
  27 of 70,897 entries (0.04%). `define charge` buries its real sense 2 inside a
  quoted example, and two raw `|` reach the screen; `just`, `ratio`, `glop`,
  `logarithmic`, `depth` and `shortness` are the same shape. A sequence-opening
  `1` must be structurally placed, but a continuing number is exempt — and that
  exemption is the defect. It stands because requiring placement for every number
  regresses senses NOAD genuinely writes unplaced (`bases`: "plural form of
  base1 2 …", and likewise `absolute`, `ambrosia`, `bind`). Pinned by the live
  ratchet rather than left to drift.
- **Some block boundaries are genuinely ambiguous in the source, and this is not
  rare.** A part-of-speech opens a block when it follows a sentence end or a
  closing `. ) : ; ]`, but NOAD does not always write one. Two shapes remain,
  measured over all 71,427 reachable entries:
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
`-no-color`**) it enters raw mode and runs its own editor:

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

The editor is a pure state machine — `Apply(Editor, Key, matches) → (Editor,
Action)` plus `RenderLine` — so every behaviour above is a table test over key
sequences with no terminal. `Action` is what the *loop* must do; the editor never
acts. Candidates arrive as a plain slice rather than a `History` handle, so
**`Apply` never queries**; the loop resolves matches once per keystroke and hands
the same slice to both the state machine and the suggestion.

**Cancellation changes shape in raw mode, and this is the subtle part.** Ctrl-C
arrives as byte `0x03`, not a signal, so `signal.NotifyContext` — which the
one-shot and piped paths still rely on — never fires. The key reader owns
cancellation instead, calling `cancel()` the moment it decodes an interrupt, which
works even while the loop is blocked in playback.

That forced a second decision: **render cooked, play raw.** Printing a definition
needs cooked mode so newlines translate; playback must stay raw so the key reader
keeps seeing bytes. Doing the whole lookup cooked made Ctrl-C during playback
hang — verified, then fixed, then pinned by `TestPTYCtrlCDuringPlaybackExitsPromptly`.

`#2`'s `eraseLineAndStepBack` and `skipPrompt` are **deleted, not ported**: they
existed to step back over the terminal's echo of Enter, and raw mode does not
echo. That also removes `#2`'s documented limitation that typing during playback
stranded the indicator — the arithmetic has nothing left to correct for.

## The store

Persistence is YAML files under the **working directory** — no config, no brain
resolution, no home-directory search. `NewYAML(dir, warn)` takes the directory as
a parameter, so *who chooses it* stays one line at the boundary if a config
arrives later.

```
words/<slug>.yaml        one file per word
events/YYYY-MM-DD.yaml   append-only, one file per day, named in UTC
```

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

A whole record therefore **ends with a newline** and **carries every field**. The
writer always terminates a record, and `AppendEvent` repairs a missing terminator
before writing — without which one interrupted write costs *two* events, because
the next append lands on the fragment's line and is parsed as part of it.

Parsing is one path, always record-by-record. A fast-path-plus-fallback version
double-counted whatever the failed whole-file parse had already collected, and
left the fallback unreachable for input that stayed valid — which is exactly what
a truncation produces.

**Two Store implementations, one conformance suite.** `Mem` is the reference and
ships as production code; `storetest.Suite` runs against both, so "the fake
behaves like the real thing" is a test rather than an assumption.

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
`lookupAndRender` so the raw path could render cooked and play raw, and that is
what makes it the one function every path shares.

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

**Type-ahead needed no change to the pure editor.** `Apply(e, k, matches)` always
took its candidate list from the caller, so command mode is a different match
*source*, not a different editor. `completionsFor` is the one place that decides
which namespace a line draws from, and it replaced four `hist.Prefix(...)` call
sites. Candidates come back `/`-prefixed because `Suggestion` matches against the
whole typed line: with `/his` typed, `/history` is what completes it. Once an
argument is typed (`/history 7`) the completion is shorter than the line, so no
suggestion is offered — that falls out rather than being special-cased.

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
commands and Tab accepted `/help`. `Apply` still receives the pre-keystroke list,
which is correct — it is deciding what to do with that keystroke given the line
as it stands — but nothing can hand `draw` a stale one.

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
`commandCtx` is deliberately narrower than `deps` — a command cannot reach the
dictionary or the player.

| command | does |
|---|---|
| `/help` | lists the commands |
| `/history [N]` | words looked up in the last N local days (default 2) |
| `/sound [N]` | how many times a pronunciation plays, for the rest of the session |

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
…/sounds/oxford/sycophantic--_us_1.mp3                        (and _2)
```

The order is measured, not assumed: across a 10-word survey the 2022 generation
strictly dominates the legacy paths (`gaslighting` exists only on the newer one;
`defenestrate` needs `_2` on the older one). That is why there is a candidate
*list* rather than one URL, and `fetch_conformance_test.go` asserts both facts
still hold.

`playN` keeps the repeat loop in the shell rather than behind `Player.Play(n)`,
so `fakePlayer` can count plays — which is how "play it three times" is an
assertion rather than something checked by ear. Repeats are separated by 250 ms
so they are distinguishable; the gap is not paid after the last one.

A missing recording is **not** a failed lookup: the definition has already been
printed, so audio failures warn on stderr and leave the exit code at 0.

## Conformance

Live checks sit behind `//go:build conformance` and run **on demand, not in CI** —
they need a host with NOAD installed and reachable network, neither of which
belongs in `merge-check.yml`.

All three seams have one, and each pins the assumption that seam rests on:

| check | asserts |
|---|---|
| `dict_conformance_test.go` | live lookups still byte-match every fixture |
| `fetch_conformance_test.go` | the CDN path survey still holds (2022 generation dominates) |
| `player_conformance_test.go` | `afplay` **blocks** until playback finishes |

The third is the least obvious and the most load-bearing: if `afplay` ever
returned immediately, three *overlapping* sounds would satisfy `fakePlayer`'s
count and every other test here — "plays three times" would be true on paper and
wrong in the room.

```sh
go test -tags conformance ./cmd/define/   # must run UNSANDBOXED
cmd/define/testdata/capture.sh            # re-capture the corpus
```

Trigger: after a macOS upgrade, or when a parse/audio bug is reported.
`DCSCopyTextDefinition` returns *silence, not an error*, without real access to
`/System/Library/AssetsV2` — which is why `capture.sh` enforces a byte floor.
