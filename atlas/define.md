# define

`define <word>` prints the New Oxford American Dictionary entry for a word and
plays its recorded pronunciation.

NOAD is the dictionary Google licenses for its US definition panel, and macOS
bundles it — which is why `define sycophantic` shows `/ˌsikəˈfan(t)ik/`,
character-for-character what the browser shows. That notation is **not** standard
IPA: NOAD writes `i` for /ɪ/, `a` for /æ/, and `(t)` for the optional flap.

## Shape

A pure core with three thin IO seams. Everything lives in `cmd/define/` as
`package main` — `internal/` is earned on the second consumer (`AGENTS.local.md`),
and there is no second consumer yet.

| Seam | Wraps | Fake |
|---|---|---|
| `Dictionary` | CoreServices `DCSCopyTextDefinition` (cgo) | `fakeDictionary` over the captured corpus |
| `AudioSource` | Google's gstatic MP3 CDN | `fakeCDN`, an `httptest` server recording request order |
| `Player` | `afplay(1)` | `fakePlayer`, recording play count |

Pure: `ParseEntry` (flat text → `Entry`), `Render` (`Entry` → string),
`AudioCandidates` (word → ordered URLs). None of them touch IO, so their tests
need no mocks.

`dict_darwin.go` holds the only cgo. `dict_stub.go` (`//go:build !darwin`) keeps
`go build ./...` and `go vet ./...` green off macOS.

## Two parsing rules worth knowing

NOAD returns flat text with no schema. Two shapes are non-obvious and both came
out of reading real output:

1. **The head can carry a part-of-speech.** `record` returns
   `record rec·ordnoun | ˈrekərd | …` — the POS is welded onto the
   syllabification, ahead of the pronunciation. It opens the first block
   (`Entry.HeadPOS` / `Block.FromHead`) but renders in the head, because that is
   where NOAD puts it.
2. **`|` is overloaded.** It delimits pronunciations *and* separates examples.
   The discriminator is word shape, not character class — `ˈrekərd` and `baNGk`
   are mostly ASCII letters, so "contains no ASCII letters" fails. A span is a
   pronunciation iff every comma-separated part is a single space-free token
   (`isPronunciation`).

## The invariant

`TestRenderLosesNothing` asserts that every letter and digit of the raw entry
appears **in order** in the rendered output. Punctuation may be restructured;
words may not vanish or move.

This is what makes a best-effort parser safe against entries nobody sampled: an
unrecognized construct degrades to a paragraph instead of disappearing. It caught
two reorderings on `record` during M1 that no content-presence test would have.

It guarantees **fidelity, not completeness** — see Limits.

## Limits

- **Only one homograph is reachable.** `DCSCopyTextDefinition("bank")` returns
  `bank 1` (the riverbank) and stops; the financial sense is not in the response.
  `define` renders the homograph number so the truncation is visible.
- **NOAD has gaps.** Recent coinages (`rizz`, `unalive`) are absent; `define`
  exits 1. Their audio is missing too — the gaps correlate, both tracing to Oxford.
- **No second dictionary source**, by decision.

## Conformance

Live checks sit behind `//go:build conformance` and run **on demand, not in CI** —
they need a host with NOAD installed and reachable network, neither of which
belongs in `merge-check.yml`.

```sh
go test -tags conformance ./cmd/define/   # must run UNSANDBOXED
cmd/define/testdata/capture.sh            # re-capture the corpus
```

Trigger: after a macOS upgrade, or when a parse/audio bug is reported.
`DCSCopyTextDefinition` returns *silence, not an error*, without real access to
`/System/Library/AssetsV2` — which is why `capture.sh` enforces a byte floor.
