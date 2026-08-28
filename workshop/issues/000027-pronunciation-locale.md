---
id: 000027
status: working
deps: []
github_issue:
created: 2026-08-28
updated: 2026-08-28
estimate_hours:
started: 2026-08-28T00:38:19-07:00
---

# pronunciation locale and language as parameters, not literals

## Problem

Split from **#18 M1**, which said it plainly: *"M1 is independently shippable and
does not wait on #10."* #18's remaining scope (deck language dimension,
inflection/lemma identity, gender, agreement-safe distractors) is blocked on
`#10`/`#12`; this half is blocked on nothing, and two other issues are waiting on
the concept it introduces.

`AudioCandidates` (`cmd/define/audiourl.go:41`) writes the language as a
**literal**:

```go
out = append(out, audioBase+"/pronunciation/2022-03-02/audio/"+shard+"/"+esc+"_en_"+locale+"_"+n+".mp3")
```

So `madrugar` is only ever requested as an English word. Measured on 2026-08-22 —
all four candidates 404 while the recording sits two characters away:

```
madrugar_en_us_1.mp3   404      ← what define asks for
madrugar--_us_1.mp3    404
madrugar_es_es_1.mp3   200      ← what exists
madrugar_es_us_1.mp3   200
```

Coverage is real, not incidental: `sobremesa`, `empalagoso`, `chapucero`,
`desvelarse` all return 200 on `es_es`.

## Spec

Carried verbatim in substance from #18 M1.

**The locale is not cosmetic, and this is the part worth designing around.**
Spanish orthography is phonemic — spelling plus the written accent determines
pronunciation exactly — so Spanish dictionary entries carry **no phonetic
notation at all**, unlike NOAD's `lig·a·ment | ˈliɡəmənt |`. Verified on the live
entries. That means:

- `ParseEntry` finding no pronunciation on a Spanish entry is **correct**. Do not
  "fix" it.
- The recording is therefore the *only* place pronunciation information exists,
  which makes audio more load-bearing for Spanish than for English.
- The two locales encode a genuine phonemic split, not an accent flavour: `es_es`
  is Castilian (*cazar* /θ/ ≠ *casar* /s/), `es_us` is Latin American *seseo*
  (both /s/). Choosing one chooses which sound system a learner acquires.
  `es_419`, `es_mx` and `es_ar` are **not** valid — measured 404.

Design: language becomes a parameter beside locale. `AudioCandidates` already
returns an *ordered* list that `httpAudioSource` walks until the first 200, so
widening it costs no new mechanism — an unspecified language can try several, and
`-lang es` reorders rather than restricts. A conformance assertion belongs beside
`TestCDNStillServesTheExpectedPaths`, which pins the English ordering the same
way.

## Done when

- [ ] `define madrugar` plays a recording; asserted against the fake, and a
      conformance test measures the live CDN the way the English ordering is.
- [ ] `-lang` selects the language and `-locale` still selects the variant;
      `es_es` and `es_us` both reachable, and the help text says what the
      difference *is* (θ vs seseo) rather than naming two country codes.
- [ ] An English word with no recording still degrades to a warning, exit 0 — the
      existing behaviour does not regress.
- [ ] No pronunciation is rendered for a Spanish entry, and a test says that is
      expected rather than a gap.
- [ ] The new conformance assertions route their dependency probe through
      `conformance.SkipOrFail` (#25) — a CDN that cannot be reached SKIPS by
      default and FAILS under `CONFORMANCE_STRICT`.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-28

- **Split out of #18** so the locale concept can land ahead of the two issues that
  want it, neither of which needs #18's deck/agreement scope:
  - **#26** (NOAD ratchet 27→32) — its open design question is whether a dual
    locale block `| AmE brɛnt, BrE brɛnt |` should render both pronunciations or
    the one matching the deck's locale. That question is unanswerable while
    "locale" is a literal in a URL builder.
  - **#18 M2** — the deck language dimension builds directly on language being a
    parameter.
- Everything in the Spec was measured against the live CDN and dictionary on
  2026-08-22, not reasoned about; see #18's Log for the measurement session,
  including a wrong claim it corrected (a sandboxed `DCSCopyTextDefinition`
  reports a clean miss, indistinguishable from an absent word).
