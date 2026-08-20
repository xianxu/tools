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

## Entry modes

One word per invocation was the original shape; `run` now dispatches on argument
count into a single shared `defineOnce`, so neither mode owns a copy of the
define path.

| invocation | behaviour |
|---|---|
| `define <word>` | one-shot |
| `define` | reads stdin until EOF or Ctrl-C |
| `echo w \| define` | same loop, no prompt |

The loop reads stdin **unconditionally** and only the prompt is TTY-conditional —
there is no interactive/batch branch to keep in sync, and the whole loop is
testable from a string. `deps.stdinIsTerminal` is injected because a test
harness's stdin is never a terminal; note it is a different question from the
stdout probe that drives colour.

A bare return replays audio and writes nothing to stdout — no definition, no
announcement. `speak` is silent by construction and `defineOnce` owns the "♫
playing N×" line, so the loop cannot accidentally reprint. The only screen change
on a replay is the terminal's own echo of the Enter key and the next prompt;
suppressing those would need cursor control, which is a stated non-goal. Replay costs no network: `cachingAudioSource` decorates the `AudioSource` seam
*inside* `repl`, so the production and test wiring are the same line and
`fakeCDN.Requested()` is the assertion. Failed fetches are not cached, so a
transient outage does not poison a session.

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
