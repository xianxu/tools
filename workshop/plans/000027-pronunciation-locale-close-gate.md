---
gate: boundary-review
issue: 27
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-28T22:43:52-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: atlas/define.md restates the locale policy by hand while only the README was wired to localeHelp
          detail: |-
            The plan named four drifting sites (localeFor, flag help, README, atlas) and the commit wired
            two. The atlas is the site with a demonstrated drift record — this diff had to rewrite it because
            it still carried #23 M1's interim rule — and the repo already has the mechanism for atlas spans
            (TestAtlasQuotesTheRawNotationCount, doc_sync_test.go:66). Fix: add a marked
            <!-- locale-help --> span at atlas/define.md:1158 and assert it alongside the README
            (ARCH-DRY, ARCH-PURPOSE: the class, not the instance).
          family: doc-restates-code-owned-fact
          round: 1
        - id: BR-2
          severity: Important
          title: no run() level test drives -locale, so the flag to CDN wiring is unpinned
          detail: |-
            TestLocaleFor and TestAudioCandidatesSpanishLocales both build the voice by hand; no test passes
            -locale as an argument. Dropping `locale: *locale` at main.go:522 or breaking the applyVoice call
            at main.go:585 leaves the whole suite green while `define -lang es -locale us` silently plays
            Castilian. This is the shape of #23's C1 (voiceFor was right; nothing called it). Model the new
            case on TestLangSwitchReDerivesEverythingDownstreamOfTheLanguage (lang_scope_test.go:176) and
            assert cdn.Requested() contains _es_us_.
          family: derived-value-untested-at-its-boundary
          round: 1
        - id: BR-3
          severity: Minor
          title: three comments describe rules this commit deleted
          detail: |-
            voice.go:23-25 still says "#27 owns real locale policy ... the interim rule it inherits";
            main.go:387-391 says the flag default is "us" (it is now ""); main.go:383-385 explains a -locale
            complaint that no longer exists. All three sit in files the diff edits and contradict
            atlas/define.md:1163.
          family: stale-comment-at-changed-site
          round: 1
        - id: BR-4
          severity: Minor
          title: localeSet and localeFor's flagSet parameter are inert now that the flag default is empty
          detail: |-
            localeFor's `!flagSet || flag == ""` is decided by `flag == ""` alone for every input, so
            options.localeSet (main.go:392), isSet(fs, "locale") (main.go:522) and the parameter carry no
            information, and TestLocaleFor's flagSet column asserts nothing. The same "not kept warm for a
            hypothetical caller" argument the commit applied to applyVoice's io.Writer applies here — either
            drop them or restore the "us" default so flagSet means something again.
          family: dead-parameter-kept-warm
          round: 1
        - id: BR-5
          severity: Minor
          title: the locale is interpolated into the CDN URL unescaped, so some values error instead of missing cleanly
          detail: |-
            audiourl.go:52 PathEscapes the word but not v.Locale. Measured: -locale '%zz' or a locale with a
            control character makes http.NewRequestWithContext fail, so the user sees
            `define: madrugar: parse "...": invalid URL escape` rather than the "no recorded pronunciation"
            warning the plan promises for unserved pairs; -locale ../../x rewrites the path. Pre-existing for
            English, but this commit widened the input domain to every language. url.PathEscape(v.Locale) makes
            any locale a clean 404.
          family: unescaped-input-in-url
          round: 1
        - id: BR-6
          severity: Minor
          title: the README's jalapeño example depends on a Larousse entry nothing in the tree can confirm
          detail: |-
            README.md:37 shows `define -lang es -locale us jalapeño`. A dictionary miss returns before speak(),
            so the example only demonstrates the flag if the Larousse carries the word; the committed es corpus
            (5 words) does not include it, and the CDN 200 does not settle it. madrugar is both committed and
            conformance-checked.
          family: doc-example-not-verified
          round: 1
        - id: BR-7
          severity: Minor
          title: all five plan tasks are still unticked though the code delivers them
          detail: |-
            workshop/plans/000027-pronunciation-locale-plan.md's `## Tasks` rows are `- [ ]` while the issue's
            Done-when is fully ticked. Tick them or record delivery in `## Revisions`; also note there that the
            "one source" task wired the README only.
          family: plan-state-lags-the-code
          round: 1
      blocked: true
---

# Gate ledger — tools#27 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-28T22:43:52-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `doc-restates-code-owned-fact` atlas/define.md restates the locale policy by hand while only the README was wired to localeHelp
  The plan named four drifting sites (localeFor, flag help, README, atlas) and the commit wired
  two. The atlas is the site with a demonstrated drift record — this diff had to rewrite it because
  it still carried #23 M1's interim rule — and the repo already has the mechanism for atlas spans
  (TestAtlasQuotesTheRawNotationCount, doc_sync_test.go:66). Fix: add a marked
  <!-- locale-help --> span at atlas/define.md:1158 and assert it alongside the README
  (ARCH-DRY, ARCH-PURPOSE: the class, not the instance).
- **BR-2** [Important] `derived-value-untested-at-its-boundary` no run() level test drives -locale, so the flag to CDN wiring is unpinned
  TestLocaleFor and TestAudioCandidatesSpanishLocales both build the voice by hand; no test passes
  -locale as an argument. Dropping `locale: *locale` at main.go:522 or breaking the applyVoice call
  at main.go:585 leaves the whole suite green while `define -lang es -locale us` silently plays
  Castilian. This is the shape of #23's C1 (voiceFor was right; nothing called it). Model the new
  case on TestLangSwitchReDerivesEverythingDownstreamOfTheLanguage (lang_scope_test.go:176) and
  assert cdn.Requested() contains _es_us_.
- **BR-3** [Minor] `stale-comment-at-changed-site` three comments describe rules this commit deleted
  voice.go:23-25 still says "#27 owns real locale policy ... the interim rule it inherits";
  main.go:387-391 says the flag default is "us" (it is now ""); main.go:383-385 explains a -locale
  complaint that no longer exists. All three sit in files the diff edits and contradict
  atlas/define.md:1163.
- **BR-4** [Minor] `dead-parameter-kept-warm` localeSet and localeFor's flagSet parameter are inert now that the flag default is empty
  localeFor's `!flagSet || flag == ""` is decided by `flag == ""` alone for every input, so
  options.localeSet (main.go:392), isSet(fs, "locale") (main.go:522) and the parameter carry no
  information, and TestLocaleFor's flagSet column asserts nothing. The same "not kept warm for a
  hypothetical caller" argument the commit applied to applyVoice's io.Writer applies here — either
  drop them or restore the "us" default so flagSet means something again.
- **BR-5** [Minor] `unescaped-input-in-url` the locale is interpolated into the CDN URL unescaped, so some values error instead of missing cleanly
  audiourl.go:52 PathEscapes the word but not v.Locale. Measured: -locale '%zz' or a locale with a
  control character makes http.NewRequestWithContext fail, so the user sees
  `define: madrugar: parse "...": invalid URL escape` rather than the "no recorded pronunciation"
  warning the plan promises for unserved pairs; -locale ../../x rewrites the path. Pre-existing for
  English, but this commit widened the input domain to every language. url.PathEscape(v.Locale) makes
  any locale a clean 404.
- **BR-6** [Minor] `doc-example-not-verified` the README's jalapeño example depends on a Larousse entry nothing in the tree can confirm
  README.md:37 shows `define -lang es -locale us jalapeño`. A dictionary miss returns before speak(),
  so the example only demonstrates the flag if the Larousse carries the word; the committed es corpus
  (5 words) does not include it, and the CDN 200 does not settle it. madrugar is both committed and
  conformance-checked.
- **BR-7** [Minor] `plan-state-lags-the-code` all five plan tasks are still unticked though the code delivers them
  workshop/plans/000027-pronunciation-locale-plan.md's `## Tasks` rows are `- [ ]` while the issue's
  Done-when is fully ticked. Tick them or record delivery in `## Revisions`; also note there that the
  "one source" task wired the README only.

## Open findings

- **BR-1** [Important] `doc-restates-code-owned-fact` atlas/define.md restates the locale policy by hand while only the README was wired to localeHelp
- **BR-2** [Important] `derived-value-untested-at-its-boundary` no run() level test drives -locale, so the flag to CDN wiring is unpinned
- **BR-3** [Minor] `stale-comment-at-changed-site` three comments describe rules this commit deleted
- **BR-4** [Minor] `dead-parameter-kept-warm` localeSet and localeFor's flagSet parameter are inert now that the flag default is empty
- **BR-5** [Minor] `unescaped-input-in-url` the locale is interpolated into the CDN URL unescaped, so some values error instead of missing cleanly
- **BR-6** [Minor] `doc-example-not-verified` the README's jalapeño example depends on a Larousse entry nothing in the tree can confirm
- **BR-7** [Minor] `plan-state-lags-the-code` all five plan tasks are still unticked though the code delivers them
