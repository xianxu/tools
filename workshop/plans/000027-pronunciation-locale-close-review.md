# Boundary Review — tools#27 (whole-issue close)

| field | value |
|-------|-------|
| issue | 27 — pronunciation locale and language as parameters, not literals |
| repo | tools |
| issue file | workshop/issues/000027-pronunciation-locale.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1e568519dd837fb5506110c3d24dfdb6f8869213..3d0b750683fea1d8ec553404d28add1516d4713d |
| command | sdlc close --issue 27 |
| reviewer | claude |
| timestamp | 2026-08-28T22:43:52-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The diff delivers what the plan's five tasks describe: `localeFor` loses the `l == store.DefaultLang` guard so `-locale` applies to every language, the now-dead complaint plumbing (`io.Writer` through `applyVoice`/`voiceFor`) is deleted rather than kept warm, `localeHelp` becomes one source with the README wired to it by a doc-sync test, both halves of the "Spanish notation" claim are pinned by tests against a committed corpus (with `jalapeño` captured so the English half asserts instead of skipping), and the live Spanish conformance row now covers both locales plus `cazar`/`casar`. I ran the full package suite (green, 94s) and the live conformance suite (`TestCDNStillServesSpanishOnTheExpectedPaths` PASS), and independently probed the CDN: `jalapeño_es_us`, `jalapeño_es_es`, `schedule_en_gb` are 200 and `madrugar_es_gb` is 404, matching the documented trade. Nothing here is Critical. Two Important items block only cheaply: the atlas is still a hand-maintained restatement of the policy the commit just single-sourced (the very doc that had gone stale), and the headline user-visible path — `-locale` reaching the CDN request — is asserted only at the pure layer, which is exactly the shape of the bug `#23`'s C1 shipped.

**1. Strengths**

- `cmd/define/voice.go:33-55` — the change is a deletion with the reasoning attached, and it cites the right precedent (`ParseLang` doesn't enumerate languages, so the adjacent field shouldn't either). Withdrawing the whitelist at the gate was correct; a table would have had no answer for `fr`, and `arrondissement_fr_fr` really is served.
- `cmd/define/voice.go:75-77`, `command.go:383`, `main.go:585` — with the refusal gone, the `io.Writer` had nothing to say and was removed rather than kept warm for a hypothetical caller. That's the right disposal of dead plumbing (ARCH-PURE: `localeFor`/`voiceFor` are now pure in signature, not just in comment).
- `cmd/define/render_test.go:335-352` — the vacuity guard (`len(d.entries) == 0 → Fatal`) is what makes this a real assertion, and it isn't trivially true: the Spanish fixtures carry pipe characters as example separators (`madrugar … misa de siete | a quien madruga …`), so the parser genuinely has something to mis-read. It survives because `hasPhoneticMarker` needs runes Spanish prose lacks.
- `cmd/define/render_test.go:356-370` + `testdata/capture.sh:65-70` — the scope half asserts rather than skips, and the fixture is reproducible from the capture script with the reason recorded, including why the tilde matters.
- `cmd/define/fetch_conformance_test.go:67-85` — both locales plus the phonemic pair, all routed through `head()` → `conformance.SkipOrFail`, satisfying the `#25` Done-when row. Verified live: PASS.
- `README.md:36` — swapping `colour` for `schedule` quietly fixes an example that could never have worked (`colour_en_us` is also 404; the word is absent from the corpus). Good catch, consistent with the plan's own near-miss note.

**2. Critical findings**

None.

**3. Important findings**

- **`atlas/define.md:1136-1165` — the atlas is still a hand-maintained restatement of the policy `localeHelp` now owns (ARCH-DRY, ARCH-PURPOSE).** The plan named four drifting sites — `localeFor`, the flag help, the README, the atlas — and the commit wired two of them. The atlas is the one with a demonstrated drift record: this very diff had to rewrite it because it was still carrying `#23 M1`'s interim rule verbatim. The repo already has the exact mechanism for the atlas specifically (`TestAtlasQuotesTheRawNotationCount`, `doc_sync_test.go:66`), so the class is enumerable and the sweep is mechanical. Fix: add a `<!-- locale-help -->…<!-- /locale-help -->` span in the "One source for the policy text" paragraph (`atlas/define.md:1158`) and extend `TestREADMEQuotesTheLocaleHelp` (or add a sibling) to require the span in both files. Fixing the README alone is the instance, not the class.
- **`cmd/define/main.go:409,522,585` — no test drives `-locale` through `run()`, so the flag→voice→CDN wiring is unpinned.** `TestLocaleFor` (`audiourl_test.go:110`) tests the policy and `TestAudioCandidatesSpanishLocales` (`:142`) tests URL construction from a hand-built `voice{Lang:"es",Locale:"us"}`, but no test passes `-locale` as an argument. Delete `locale: *locale` from the options literal, or break `applyVoice`'s call site, and the entire suite stays green while `define -lang es -locale us` silently plays Castilian. This is the failure mode `#23`'s C1 already shipped once — `voiceFor` was right and nothing called it — and the fix template is right there: `TestLangSwitchReDerivesEverythingDownstreamOfTheLanguage` (`lang_scope_test.go:176`). Add a case using the same `fakeCDN` rig with `args = {"-lang","es","-locale","us","sycophantic"}` asserting `cdn.Requested()` contains `_es_us_` and no `_es_es_`.

**4. Minor findings**

- `cmd/define/voice.go:23-25` — `defaultLocale`'s comment still says "*`#27` owns real locale policy … This is the interim rule it inherits*". `#27` is this commit; the comment now contradicts `atlas/define.md:1163` ("that is now history").
- `cmd/define/main.go:387-391` — "*the flag's default is `us`*" is false as of `main.go:409` (default is now `""`).
- `cmd/define/main.go:383-385` — "*so the `-locale` complaint is not re-emitted on every replay*" describes a complaint this commit deleted.
- `cmd/define/voice.go:52` — with the flag default empty, `flagSet` no longer discriminates: for every `(flag, flagSet)` pair the result is decided by `flag == ""` alone, so `options.localeSet`, `isSet(fs, "locale")` and the parameter are inert, and `TestLocaleFor`'s `flagSet` column asserts nothing. Same "kept warm for a hypothetical caller" argument the commit applied to the `io.Writer`. Either drop them or restore the `"us"` default to make `flagSet` meaningful again.
- `cmd/define/audiourl.go:52` — the word is `url.PathEscape`d but `v.Locale` is interpolated raw. `-locale '%zz'` or a locale containing a control character makes `http.NewRequestWithContext` fail, so the user gets `define: madrugar: parse "…": invalid URL escape` instead of the "no recorded pronunciation" warning the plan promises for unserved pairs; `-locale ../../x` rewrites the path (same host, harmless, but not a clean miss). Pre-existing for English, but this commit widened the input domain to every language. `url.PathEscape(v.Locale)` makes every locale a clean 404.
- `README.md:37` — `define -lang es -locale us jalapeño` is unverified end-to-end: a lookup miss returns before `speak`, so the example only demonstrates the flag if the Larousse carries `jalapeño`, and the committed `es` corpus (5 words) can't say. `madrugar` is both committed and conformance-checked. Either verify on a machine with the Larousse or use `madrugar`.
- `cmd/define/main.go:409` — `-h` no longer prints a default for `-locale`. Defensible (the default is per-language now), but the help text doesn't state that `en` defaults to `us` and other languages to their own code.
- Behaviour note, declared in the plan's Risks but worth one line: `-locale` survives a mid-session `/lang` switch, so `define -locale gb` followed by `/lang es` makes every Spanish lookup a silent 404 for the rest of the session. The old code re-printed a complaint at each `applyVoice`. The plan's stated remedy if it proves confusing is a warning naming the pair, not a rejection.

**5. Test coverage notes**

- Full package suite green (94.2s); `-tags conformance -run CDN` green against the live CDN (2.0s).
- Revert-checks: `TestLocaleFor`'s "Spanish honours the flag: seseo" row and `TestAudioCandidatesSpanishLocales` both fail under the old `l == store.DefaultLang` guard (the old row asserted the opposite and was deleted, not weakened); `TestREADMEQuotesTheLocaleHelp` fails on any README drift from `localeHelp`. These are genuine failing-without-the-fix tests, not restatements.
- `TestSpanishEntriesCarryNoPronunciationNotation` is scoped to 5 committed entries, all NFC. A Spanish entry with decomposed (NFD) accents would trip `hasPhoneticMarker`'s combining-diacritic branch (`parse.go:186`) and could yield a false IPA from a pipe-delimited example; the corpus can't see that case. Worth a line in the test comment or a synthetic NFD case, not a blocker.
- Gap already noted above: nothing exercises `-locale` at the `run()` boundary.

**6. Architectural notes for upcoming work**

- **ARCH-DRY — flag.** The policy prose now appears in `localeHelp`, `localeFor`'s comment, the README (derived), the atlas (not derived), and three test comments. One derived consumer is a real improvement; the atlas is the one that has actually drifted before.
- **ARCH-PURE — pass.** Business logic (`localeFor`, `voiceFor`, `AudioCandidates`) is pure and unit-tested without IO; the `io.Writer` that had leaked into the policy layer is gone. The doc-sync tests read files, which is appropriate for doc guards.
- **ARCH-PURPOSE — flag (the atlas half above); otherwise delivered.** Every Done-when row is genuinely satisfied, including the `SkipOrFail` routing. The shadow-sweep over `localeHelp`'s consumers is what turns up the atlas.
- **ARCH-MOCK — pass.** `fakeCDN` reused rather than extended, the fetch loop asserts requested paths through it, and the live conformance check is the drift detector for the facts the fake models. Both flows share the `AudioSource` seam.
- For `#29`: the third input this issue's Spec called out — source orthography — is still implicit. `AudioCandidates` escapes the word but derives nothing about accent-vs-unaccented keying (`jalapeño_es_es` 200 / `jalapeno_es_es` 404 / English keys `jalapeno`). That's the seam `#29` will need, and it belongs beside `voice`, not in a caller.

**7. Plan revision recommendations**

- `workshop/plans/000027-pronunciation-locale-plan.md` — all five rows under `## Tasks` are still `- [ ]` while the code delivers them and the issue's Done-when is ticked. Tick them, or add a `## Revisions` entry recording delivery.
- Same file, the help-text task ("*the policy is currently stated in four places … the README derives from it*"): add a `## Revisions` entry stating that only the README was wired and the atlas remains hand-maintained — or, preferably, wire the atlas and let the task stand as written. As it reads now, the plan claims a single-source outcome the tree doesn't have.

```findings
findings:
  - id: new
    severity: Important
    family: doc-restates-code-owned-fact
    title: |
      atlas/define.md restates the locale policy by hand while only the README was wired to localeHelp
    detail: |
      The plan named four drifting sites (localeFor, flag help, README, atlas) and the commit wired
      two. The atlas is the site with a demonstrated drift record — this diff had to rewrite it because
      it still carried #23 M1's interim rule — and the repo already has the mechanism for atlas spans
      (TestAtlasQuotesTheRawNotationCount, doc_sync_test.go:66). Fix: add a marked
      <!-- locale-help --> span at atlas/define.md:1158 and assert it alongside the README
      (ARCH-DRY, ARCH-PURPOSE: the class, not the instance).
  - id: new
    severity: Important
    family: derived-value-untested-at-its-boundary
    title: |
      no run() level test drives -locale, so the flag to CDN wiring is unpinned
    detail: |
      TestLocaleFor and TestAudioCandidatesSpanishLocales both build the voice by hand; no test passes
      -locale as an argument. Dropping `locale: *locale` at main.go:522 or breaking the applyVoice call
      at main.go:585 leaves the whole suite green while `define -lang es -locale us` silently plays
      Castilian. This is the shape of #23's C1 (voiceFor was right; nothing called it). Model the new
      case on TestLangSwitchReDerivesEverythingDownstreamOfTheLanguage (lang_scope_test.go:176) and
      assert cdn.Requested() contains _es_us_.
  - id: new
    severity: Minor
    family: stale-comment-at-changed-site
    title: |
      three comments describe rules this commit deleted
    detail: |
      voice.go:23-25 still says "#27 owns real locale policy ... the interim rule it inherits";
      main.go:387-391 says the flag default is "us" (it is now ""); main.go:383-385 explains a -locale
      complaint that no longer exists. All three sit in files the diff edits and contradict
      atlas/define.md:1163.
  - id: new
    severity: Minor
    family: dead-parameter-kept-warm
    title: |
      localeSet and localeFor's flagSet parameter are inert now that the flag default is empty
    detail: |
      localeFor's `!flagSet || flag == ""` is decided by `flag == ""` alone for every input, so
      options.localeSet (main.go:392), isSet(fs, "locale") (main.go:522) and the parameter carry no
      information, and TestLocaleFor's flagSet column asserts nothing. The same "not kept warm for a
      hypothetical caller" argument the commit applied to applyVoice's io.Writer applies here — either
      drop them or restore the "us" default so flagSet means something again.
  - id: new
    severity: Minor
    family: unescaped-input-in-url
    title: |
      the locale is interpolated into the CDN URL unescaped, so some values error instead of missing cleanly
    detail: |
      audiourl.go:52 PathEscapes the word but not v.Locale. Measured: -locale '%zz' or a locale with a
      control character makes http.NewRequestWithContext fail, so the user sees
      `define: madrugar: parse "...": invalid URL escape` rather than the "no recorded pronunciation"
      warning the plan promises for unserved pairs; -locale ../../x rewrites the path. Pre-existing for
      English, but this commit widened the input domain to every language. url.PathEscape(v.Locale) makes
      any locale a clean 404.
  - id: new
    severity: Minor
    family: doc-example-not-verified
    title: |
      the README's jalapeño example depends on a Larousse entry nothing in the tree can confirm
    detail: |
      README.md:37 shows `define -lang es -locale us jalapeño`. A dictionary miss returns before speak(),
      so the example only demonstrates the flag if the Larousse carries the word; the committed es corpus
      (5 words) does not include it, and the CDN 200 does not settle it. madrugar is both committed and
      conformance-checked.
  - id: new
    severity: Minor
    family: plan-state-lags-the-code
    title: |
      all five plan tasks are still unticked though the code delivers them
    detail: |
      workshop/plans/000027-pronunciation-locale-plan.md's `## Tasks` rows are `- [ ]` while the issue's
      Done-when is fully ticked. Tick them or record delivery in `## Revisions`; also note there that the
      "one source" task wired the README only.
```
