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
    - "n": 2
      timestamp: "2026-08-28T22:56:00-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Verified by reverting: replacing the atlas span with prose turns TestDocsQuoteTheLocaleHelp red, naming atlas/define.md.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'Verified by reverting: localeSet:false at main.go:522 reds two rows of TestLocaleFlagReachesTheCDN while all pure-layer tests stay green.'
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: All three comments intact verbatim at HEAD — voice.go:23-25, main.go:384, main.go:387-389.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: localeFor still takes flagSet; flag default still ""; TestLocaleFor's flagSet column still asserts nothing.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: audiourl.go still interpolates v.Locale raw while PathEscaping the word.
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: README.md:37 unchanged; es corpus still 5 words without jalapeño, and no es dictionary is installed here to settle it either.
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: All five "## Tasks" rows are still "- [ ]" and no Revisions entry records delivery.
          round: 2
      findings:
        - id: BR-8
          severity: Important
          title: the rename in c17d1c8 skipped its retiredSymbolNames row, so voice.go:84 still names a symbol the tree retired
          detail: |-
            c17d1c8 renamed TestREADMEQuotesTheLocaleHelp to TestDocsQuoteTheLocaleHelp, updated
            atlas/define.md:1163, and left cmd/define/voice.go:84 naming the old symbol. 2nd finding
            in this family, so do not fix the instance — fix the rule, which this repo already
            wrote down and mechanised: repo_guard_test.go:643 retiredSymbolNames plus
            TestNoArtifactNamesARetiredSymbol at :656, whose own comment records nine prior
            recurrences and states "a rename adds a row here". This window is the tenth, and the
            row was not added. Verified in a scratch worktree: adding
            "TestREADMEQuotesTheLocaleHelp": "TestDocsQuoteTheLocaleHelp" makes the guard fail with
            'cmd/define/voice.go names the retired symbol' — so the one row both clears this
            instance and re-arms the mechanism. BR-3's three sites are the non-rename half of the
            same family, which the map cannot express; its enumeration for this window is every
            comment in voice.go, main.go and command.go mentioning the deleted -locale complaint or
            the removed "us" flag default.
          family: stale-comment-at-changed-site
          round: 2
        - id: BR-9
          severity: Minor
          title: a mid-session /lang silently reinterprets -locale, and the old code used to say so
          detail: |-
            Measured at HEAD by driving run() with args {"-locale","gb"} and stdin "/lang es\nsycophantic\n":
            the session requests sycophantic_es_gb_1.mp3 and _2, both 404, and stderr carries only the
            generic "define: sycophantic: no recorded pronunciation". Every Spanish lookup stays silent
            for the rest of the session, and there is no /locale command to recover. Before this window
            localeFor refused the pair, printed a diagnostic naming it, and fell back to a working es_es.
            The plan's Risks names the intended remedy for the explicit case ("a warning that NAMES the
            pair, not a rejection"); the mid-session case is sharper because the flag was supplied under a
            different language. Cheapest disposition is one more row in TestLocaleFlagReachesTheCDN
            pinning the unserved pair (want _es_gb_, exit 0, warning present), so the trade is a recorded
            decision rather than untested drift.
          family: session-value-reinterpreted-by-a-mode-switch
          round: 2
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

## Round 2 — 2026-08-28T22:56:00-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Verified by reverting: replacing the atlas span with prose turns TestDocsQuoteTheLocaleHelp red, naming atlas/define.md.
- BR-2 — addressed — Verified by reverting: localeSet:false at main.go:522 reds two rows of TestLocaleFlagReachesTheCDN while all pure-layer tests stay green.
- BR-3 — not-addressed — All three comments intact verbatim at HEAD — voice.go:23-25, main.go:384, main.go:387-389.
- BR-4 — not-addressed — localeFor still takes flagSet; flag default still ""; TestLocaleFor's flagSet column still asserts nothing.
- BR-5 — not-addressed — audiourl.go still interpolates v.Locale raw while PathEscaping the word.
- BR-6 — not-addressed — README.md:37 unchanged; es corpus still 5 words without jalapeño, and no es dictionary is installed here to settle it either.
- BR-7 — not-addressed — All five "## Tasks" rows are still "- [ ]" and no Revisions entry records delivery.

### Raised

- **BR-8** [Important] `stale-comment-at-changed-site` the rename in c17d1c8 skipped its retiredSymbolNames row, so voice.go:84 still names a symbol the tree retired
  c17d1c8 renamed TestREADMEQuotesTheLocaleHelp to TestDocsQuoteTheLocaleHelp, updated
  atlas/define.md:1163, and left cmd/define/voice.go:84 naming the old symbol. 2nd finding
  in this family, so do not fix the instance — fix the rule, which this repo already
  wrote down and mechanised: repo_guard_test.go:643 retiredSymbolNames plus
  TestNoArtifactNamesARetiredSymbol at :656, whose own comment records nine prior
  recurrences and states "a rename adds a row here". This window is the tenth, and the
  row was not added. Verified in a scratch worktree: adding
  "TestREADMEQuotesTheLocaleHelp": "TestDocsQuoteTheLocaleHelp" makes the guard fail with
  'cmd/define/voice.go names the retired symbol' — so the one row both clears this
  instance and re-arms the mechanism. BR-3's three sites are the non-rename half of the
  same family, which the map cannot express; its enumeration for this window is every
  comment in voice.go, main.go and command.go mentioning the deleted -locale complaint or
  the removed "us" flag default.
- **BR-9** [Minor] `session-value-reinterpreted-by-a-mode-switch` a mid-session /lang silently reinterprets -locale, and the old code used to say so
  Measured at HEAD by driving run() with args {"-locale","gb"} and stdin "/lang es\nsycophantic\n":
  the session requests sycophantic_es_gb_1.mp3 and _2, both 404, and stderr carries only the
  generic "define: sycophantic: no recorded pronunciation". Every Spanish lookup stays silent
  for the rest of the session, and there is no /locale command to recover. Before this window
  localeFor refused the pair, printed a diagnostic naming it, and fell back to a working es_es.
  The plan's Risks names the intended remedy for the explicit case ("a warning that NAMES the
  pair, not a rejection"); the mid-session case is sharper because the flag was supplied under a
  different language. Cheapest disposition is one more row in TestLocaleFlagReachesTheCDN
  pinning the unserved pair (want _es_gb_, exit 0, warning present), so the trade is a recorded
  decision rather than untested drift.

## Open findings

- **BR-3** [Minor] `stale-comment-at-changed-site` three comments describe rules this commit deleted
- **BR-4** [Minor] `dead-parameter-kept-warm` localeSet and localeFor's flagSet parameter are inert now that the flag default is empty
- **BR-5** [Minor] `unescaped-input-in-url` the locale is interpolated into the CDN URL unescaped, so some values error instead of missing cleanly
- **BR-6** [Minor] `doc-example-not-verified` the README's jalapeño example depends on a Larousse entry nothing in the tree can confirm
- **BR-7** [Minor] `plan-state-lags-the-code` all five plan tasks are still unticked though the code delivers them
- **BR-8** [Important] `stale-comment-at-changed-site` the rename in c17d1c8 skipped its retiredSymbolNames row, so voice.go:84 still names a symbol the tree retired
- **BR-9** [Minor] `session-value-reinterpreted-by-a-mode-switch` a mid-session /lang silently reinterprets -locale, and the old code used to say so
