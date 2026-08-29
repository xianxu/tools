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

---

## Re-review — 2026-08-28T22:56:00-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 27 — pronunciation locale and language as parameters, not literals |
| repo | tools |
| issue file | workshop/issues/000027-pronunciation-locale.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1e568519dd837fb5506110c3d24dfdb6f8869213..c17d1c802532b5eb1a6d240cf5fb903d4e49290c |
| command | sdlc close --issue 27 |
| reviewer | claude |
| timestamp | 2026-08-28T22:56:00-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both Important findings from round 1 are genuinely fixed, and I verified each by reverting rather than by reading the commit message: replacing the `<!-- locale-help -->` span in `atlas/define.md` with prose turns `TestDocsQuoteTheLocaleHelp` red naming the atlas, and setting `localeSet: false` at `main.go:522` turns two rows of `TestLocaleFlagReachesTheCDN` red while `TestLocaleFor`, `TestAudioCandidatesSpanishLocales` and `TestVoiceForBuildsBothFieldsTogether` all stay green — that asymmetry is exactly the property BR-2 asked for. The full package suite is green at HEAD (94.3s). What stops SHIP is one cheap Important: commit `c17d1c8` renamed `TestREADMEQuotesTheLocaleHelp` → `TestDocsQuoteTheLocaleHelp` and left `cmd/define/voice.go:84` naming the retired symbol — and this repo already mechanised that exact class in `retiredSymbolNames` / `TestNoArtifactNamesARetiredSymbol`, which the rename skipped. I proved the mechanism catches it: adding the one map row makes the guard fail on `voice.go` by name. The five Minor findings from round 1 (BR-3…BR-7) are all still present verbatim and are disposed `not-addressed` below; none blocks the gate.

**1. Strengths**

- `cmd/define/lang_scope_test.go:373` — `TestLocaleFlagReachesTheCDN` is a real failing-without-the-fix test, not a restatement. Mutation-verified: `localeSet: false` at `main.go:522` produces `the session never asked for _es_us_` / `never asked for _en_gb_` while every pure-layer test stays green. That is the `#23` C1 shape finally pinned at the layer the bug lives at.
- `cmd/define/lang_scope_test.go:428` — the vacuity guard (`asked == ""` → `Fatalf`, with stderr attached) earns its place: the fixture bug it was written after reported "never asked for `_es_us_`", which reads like a wiring bug and was a corpus mismatch. The guard makes the two distinguishable.
- `cmd/define/doc_sync_test.go:116` — the doc-sync loops **both** docs, and the failure message names which one drifted plus the exact span expected. Mutation-verified red.
- `cmd/define/voice.go:32-55` — the change is a deletion with its reasoning attached, citing the right in-repo precedent (`ParseLang` deliberately does not enumerate, so the adjacent field should not either). Withdrawing the whitelist at the plan gate was correct.
- `cmd/define/render_test.go:335` + `:356` — the Spanish half has a vacuity guard on an empty corpus, and the English half asserts against a committed `jalapeño` fixture rather than skipping, with `capture.sh:65-70` recording why the fixture exists and why the tilde matters.
- `cmd/define/fetch_conformance_test.go:74-87` — both locales plus `cazar`/`casar`, all routed through `head()` → `conformance.SkipOrFail`, satisfying the `#25` Done-when row.

**2. Critical findings**

None.

**3. Important findings**

- **`cmd/define/voice.go:84` names `TestREADMEQuotesTheLocaleHelp`, a symbol this same commit retired.** `git show 3d0b750:cmd/define/doc_sync_test.go` declares it; `c17d1c8` renamed it to `TestDocsQuoteTheLocaleHelp` and updated `atlas/define.md:1163` but not the const's own doc comment. **This is the 2nd finding in family `stale-comment-at-changed-site`** — so per the escalation rule, do not fix this instance. State and fix the rule. The rule is already written down and already mechanised: `repo_guard_test.go:643` `retiredSymbolNames` + `TestNoArtifactNamesARetiredSymbol` (`:656`), whose own comment says the symbol half "recurred nine times" and that "a rename adds a row here". The rename in this window did not add the row, which is the 10th recurrence. Fix = add `"TestREADMEQuotesTheLocaleHelp": "TestDocsQuoteTheLocaleHelp"` to the map; I verified in a scratch worktree that the guard then fails with `cmd/define/voice.go names the retired symbol …`, so the row both catches this instance and re-arms the mechanism. BR-3's three sites are the *non-rename* half of the same family, which the map cannot express — that half stays a hand sweep, and its enumeration for this window is: every comment in `voice.go`, `main.go`, `command.go` mentioning the deleted `-locale` complaint or the removed `"us"` flag default.

**4. Minor findings**

- `cmd/define/command.go:383` / `main.go:585` — a mid-session `/lang` silently reinterprets `-locale`. Measured at HEAD: `define -locale gb` then `/lang es` requests `sycophantic_es_gb_1.mp3`, 404s, and prints only the generic `no recorded pronunciation` for the rest of the session; there is no `/locale` command to recover. The old code printed a diagnostic naming the pair and fell back to a working `es_es`. The plan's Risks already names the remedy ("a warning that NAMES the pair, not a rejection") for the explicit `-lang es -locale gb` case; the mid-session case is sharper because the flag was given under a different language.

**5. Test coverage notes**

- Full `./cmd/define` suite green at HEAD (94.3s). `go build ./...` clean.
- Revert-checks performed (not inferred): BR-1 fix red without the atlas span; BR-2 fix red under `localeSet: false`, with the three pure-layer tests unaffected. Both claimed fixes are genuinely pinned.
- Gap: no `run()`-level row for an **unserved** pair. `TestLocaleFlagReachesTheCDN` has four rows, all serving pairs. The Done-when row "`-locale gb` with Spanish degrades to a warning, exit 0" is hand-verified in the `## Log` only. One more row (`-lang es -locale gb`, want `_es_gb_`, assert exit 0 and the warning on stderr) would pin the documented trade *and* the mid-session behaviour above in the same table.
- `TestLocaleFor`'s `flagSet` column still asserts nothing for any reachable input (BR-4, not-addressed): with the flag default `""`, `!flagSet || flag == ""` is decided by `flag == ""` alone.
- `TestSpanishEntriesCarryNoPronunciationNotation` runs over 5 committed NFC entries; an NFD-accented entry could trip `hasPhoneticMarker`'s combining-diacritic branch and the corpus cannot see it. Noted, not blocking.

**6. Architectural notes**

- **ARCH-DRY — pass on the policy text, flag on the symbol.** `localeHelp` is one source and the flag help, README and atlas all derive from it, proven by mutation. The flag is the `voice.go:84` retired-symbol restatement above.
- **ARCH-PURE — pass.** `localeFor` / `voiceFor` / `applyVoice` are now pure in signature, not just in comment; `opt.locale` is read at exactly one site (`voice.go:77`); `AudioCandidates` remains pure and offline. The new `run()`-level test drives the seam through injected fakes rather than mocking the pure layer.
- **ARCH-PURPOSE — pass, with a durability note.** Shadow-sweep of the single-source change: a tree-wide grep for the locale policy finds exactly three consumers (`main.go:409` flag help, `README.md:199`, `atlas/define.md:1160`), and all three derive from `localeHelp` under assertion. `workshop/projects/define-learn.md` mentions the locale narratively, not the policy text — not a consumer. The class, not the instance, was fixed. Note for later: the doc list inside `TestDocsQuoteTheLocaleHelp` is hand-enumerated, so a fourth doc restating the policy is still invisible to the guard — the same structural limit the prompt doc-sync test has.
- **ARCH-MOCK — pass.** `fakeCDN` is a stateful double with an ordered request log (`fetch_fake_test.go:18,44`), reached through the same `AudioSource` seam production uses via `rebasedSource`; the new wiring test asserts on requests rather than on internals. Live conformance covers both locales plus the phonemic pair and skips-or-fails through `conformance.SkipOrFail`. No external call escapes the seam.
- For upcoming `#29`: the locale policy now has one home and one enforced statement, which is what that issue needed. The thing it will still have to decide for itself is the third input the Spec calls out — source orthography (`jalapeño` vs `jalapeno`) — which nothing in this window models.

**7. Plan revision recommendations**

- BR-7 is still open: `workshop/plans/000027-pronunciation-locale-plan.md:141-183` has all five `## Tasks` rows as `- [ ]` while the issue's Done-when is fully ticked. Tick them, or add a `## Revisions` entry recording delivery.
- That same Revisions entry should record what the "one source" task actually landed as, since it changed under review: `localeHelp` plus `TestDocsQuoteTheLocaleHelp` asserting **README and atlas** (round 1 shipped the README alone), and that `localeFor`'s doc comment remains narrative prose by design rather than a fourth derived copy.
- Core-concepts cross-check: no contradiction. `voice` (`cmd/define/voice.go`) exists; `voices` is marked deleted and is genuinely absent; `AudioCandidates` (`cmd/define/audiourl.go`) is modified as stated; `-lang` flag, `fakeCDN` (`fetch_fake_test.go`) and `TestCDNStillServesSpanishOnTheExpectedPaths` (`fetch_conformance_test.go`) all exist at the stated paths. No plan revision needed on that table.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Verified by reverting: replacing the atlas span with prose turns TestDocsQuoteTheLocaleHelp red, naming atlas/define.md.
  - id: BR-2
    disposition: addressed
    note: |
      Verified by reverting: localeSet:false at main.go:522 reds two rows of TestLocaleFlagReachesTheCDN while all pure-layer tests stay green.
  - id: BR-3
    disposition: not-addressed
    note: |
      All three comments intact verbatim at HEAD — voice.go:23-25, main.go:384, main.go:387-389.
  - id: BR-4
    disposition: not-addressed
    note: |
      localeFor still takes flagSet; flag default still ""; TestLocaleFor's flagSet column still asserts nothing.
  - id: BR-5
    disposition: not-addressed
    note: |
      audiourl.go still interpolates v.Locale raw while PathEscaping the word.
  - id: BR-6
    disposition: not-addressed
    note: |
      README.md:37 unchanged; es corpus still 5 words without jalapeño, and no es dictionary is installed here to settle it either.
  - id: BR-7
    disposition: not-addressed
    note: |
      All five "## Tasks" rows are still "- [ ]" and no Revisions entry records delivery.
findings:
  - id: new
    severity: Important
    family: stale-comment-at-changed-site
    title: |
      the rename in c17d1c8 skipped its retiredSymbolNames row, so voice.go:84 still names a symbol the tree retired
    detail: |
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
  - id: new
    severity: Minor
    family: session-value-reinterpreted-by-a-mode-switch
    title: |
      a mid-session /lang silently reinterprets -locale, and the old code used to say so
    detail: |
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
```

---

## Re-review — 2026-08-28T23:06:48-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 27 — pronunciation locale and language as parameters, not literals |
| repo | tools |
| issue file | workshop/issues/000027-pronunciation-locale.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1e568519dd837fb5506110c3d24dfdb6f8869213..c9ccf5eb1696a2c6f1871a840f3618de03bb4b9b |
| command | sdlc close --issue 27 |
| reviewer | claude |
| timestamp | 2026-08-28T23:06:48-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The one Important finding from round 2 (BR-8) is genuinely fixed and I mutation-verified it: reintroducing `TestREADMEQuotesTheLocaleHelp` into `voice.go` makes `TestNoArtifactNamesARetiredSymbol` fail by name, so the row re-arms the guard rather than merely satisfying it. The feature itself is delivered end-to-end and pinned — I reverted `localeFor`'s deleted English-only guard in a scratch worktree and `TestLocaleFlagReachesTheCDN`'s `es_us` row went red naming the URLs actually requested, and I drifted the atlas' `locale-help` span and `TestDocsQuoteTheLocaleHelp` went red naming `atlas/define.md`. Full suite green (94s), conformance live-green against the real CDN. Nothing Critical or Important remains open, so this passes the gate. What holds it back from a clean SHIP is that the closing commit disposed exactly one of round 2's seven findings and left six Minors untouched — the implementor's own ledger records them `not-addressed` — including the enumeration that BR-8's detail explicitly named as the class half of its own fix. Those six need either a fix or an explicit withdrawal with reasons, not a third round of carrying.

## 1. Strengths

- **`cmd/define/voice.go:33-78` — the pure core got purer, not just preserved.** Removing the refusal let `localeFor`, `voiceFor` and `applyVoice` all shed their `io.Writer`; the file now imports only `store`. The `applyVoice` comment (`voice.go:72-75`) argues the parameter's removal rather than keeping it warm, which is the right instinct.
- **`cmd/define/lang_scope_test.go:383` `TestLocaleFlagReachesTheCDN` — the right test at the right altitude.** It asserts what the fetch loop *requested*, so it covers flag parse → `localeSet` → `applyVoice` → `AudioCandidates` → fetch. The `t.Fatalf` on an empty request log (`:428`) is a real non-vacuity guard, not decoration. Mutation-verified red.
- **`cmd/define/doc_sync_test.go:121` — the doc-sync reaches both consumers.** Wiring only the README would have left the atlas as the next copy to go stale; the loop over `{README.md, atlas/define.md}` closes the class. Mutation-verified red on the atlas half specifically.
- **`README.md:36` — `colour` → `schedule` is a correctness fix, not cosmetics.** The plan's own re-measurement caught that `colour_en_us` *and* `colour_en_gb` are both 404, so the old example demonstrated nothing. I independently measured `schedule_en_gb_1.mp3` → 200.
- **`cmd/define/fetch_conformance_test.go:74-85` + `workshop/lessons.md:2314`.** The conformance row now covers both Spanish locales *and* the `cazar`/`casar` pair the phonemic distinction is actually about — live-green when I ran it. And the lesson generalises BR-8 to the rule ("a mechanism with an unsupplied input is unprotected"), rather than just recording the slip.

## 2. Critical findings

None.

## 3. Important findings

None open. BR-8 disposed `addressed` — verified by revert, not by commit message.

## 4. Minor findings

All six are prior findings re-raised, not new ones. Measured at HEAD:

- **BR-3** — all three sites intact. `voice.go:23-25` still forward-refers to "#27 owns real locale policy … the interim rule it inherits"; `main.go:384` still explains a "-locale complaint" that no longer exists; `main.go:387-389` still says the flag default is `"us"` (it is `""` at `main.go:409`). BR-8's detail named exactly this enumeration as the non-rename half of its own class, and the closing commit swept only the rename half.
- **BR-4** — `localeFor` (`voice.go:51`) still branches on `!flagSet || flag == ""`, which `flag == ""` decides alone for every input. `TestLocaleFor`'s `flagSet` column still asserts nothing.
- **BR-5** — two sites, not one: `audiourl.go:50` and `audiourl.go:58` both interpolate `v.Locale` raw while `PathEscape`-ing the word. Re-measured at HEAD: `-locale '%zz'` yields `define: madrugar: parse "…madrugar_es_%zz_1.mp3": invalid URL escape "%zz"` instead of the promised "no recorded pronunciation"; `-locale ../../x` rewrites the path.
- **BR-6** — narrower than round 2 stated. I probed the CDN: `jalape%C3%B1o_es_us_1.mp3` and `…_es_es_1.mp3` are both 200, so the percent-escaping of the non-ASCII headword works. The unconfirmed half is the *dictionary* lookup that must succeed before `speak()` — `capture.sh:91` still has `es_words=(mesa bonito once real madrugar)`, and the Larousse is not installed on this machine either.
- **BR-7** — all five `- [ ]` rows unticked (`plan:142,160,176,185,188`), no `## Revisions` entry recording delivery. Two more instances of the same lag: `plan:57` still calls `localeFor` "the interim locale rule `#27` inherits", and the new `localeHelp` const has no Core-concepts row (the table guard only catches stale rows, not missing ones).
- **BR-9** — reproduced at HEAD. Driving `run()` with `{-locale gb}` and stdin `/lang es\nsycophantic\n` requests `sycophantic_es_gb_1.mp3`/`_2`, both 404, stderr carrying only the generic warning, and every later Spanish lookup stays silent with no `/locale` to recover.

## 5. Test coverage notes

Coverage of the shipped feature is good and honestly pinned — the two mutations I ran both went red at the right test with the right message. The gaps are exactly the untested boundaries the open Minors name: the mid-session `/lang` × `-locale` interaction (BR-9 — one more row in `TestLocaleFlagReachesTheCDN` would convert it from untested drift to a recorded decision), the malformed-locale path (BR-5), and `TestLocaleFor`'s inert `flagSet` column (BR-4). The `-no-audio=false` in every `TestLocaleFlagReachesTheCDN` row is redundant against the current default; harmless, but it means the rows would not notice if that default flipped.

## 6. Architectural notes

- **ARCH-DRY — pass.** `localeHelp` (`voice.go:96`) is a real single source with three derived consumers (flag help, README, atlas), and I confirmed the atlas one is not decorative. The only remaining hand-maintained restatements of the model are BR-3's two `main.go` comments.
- **ARCH-PURE — pass, improved.** The policy is now `(store.Lang, string, bool) → string` with no IO; `TestLocaleFor` and `TestAudioCandidatesSpanishLocales` run without any mock. BR-4 is the one residue: a parameter the body can no longer use to reach a different answer.
- **ARCH-PURPOSE — flag (already open as BR-3).** The issue's purpose is fully delivered — `-locale` works for every language, the help says what the choice *is*, both Spanish locales are live-verified. But the shadow-sweep on BR-8's own class shows the instance fixed and the enumeration it named left in place. That is the "instance, not the class" pattern the principle names, and it is the third round this family has appeared.
- **ARCH-MOCK — pass.** `fakeCDN` is stateful (ordered request log), injected through `deps.audio` — the same `AudioSource` seam production fills with `newHTTPAudioSource()` — and the new `run()`-level test drives the whole stack through it. Live conformance is build-tagged and routes through `conformance.SkipOrFail` (`fetch_conformance_test.go:28`), satisfying Done-when row 5.
- **For `#29`:** the no-whitelist decision is the right foundation — an origin-language voice can select `_fr_fr_` without touching `localeFor`. Keep `applyVoice` as the single derivation point when the deck word starts carrying its own `voice`; BR-9 is early evidence that a second derivation site is where this drifts.

## 7. Plan revision recommendations

The plan still claims work as pending that `main` has. It needs one `## Revisions` entry, dated 2026-08-28, saying:

- **All five `## Tasks` rows shipped** — name the delivering commits (`3d0b750`, `c17d1c8`, `c9ccf5e`) and tick them.
- **The "one source" task landed wider than written.** It specified the README as the derived consumer; the boundary review found that wiring one doc left the atlas as the next copy to go stale, so `TestDocsQuoteTheLocaleHelp` (renamed from `TestREADMEQuotesTheLocaleHelp`) now covers both.
- **`plan:57` is stale.** `localeFor` is no longer "the interim locale rule `#27` inherits" — this issue replaced it. Restate the row as the delivered policy.
- **Add a Core-concepts row for `localeHelp`** (`cmd/define/voice.go`, PURE, new) — a new pure entity shipped with no row, and the table guard only catches stale rows, not absent ones.

```findings
dispose:
  - id: BR-8
    disposition: addressed
    note: |
      Verified by revert, not by message: reintroducing TestREADMEQuotesTheLocaleHelp into
      voice.go makes TestNoArtifactNamesARetiredSymbol fail naming cmd/define/voice.go.
  - id: BR-3
    disposition: not-addressed
    note: |
      All three sites intact at HEAD (voice.go:23-25, main.go:384, main.go:387-389); this is the
      enumeration half BR-8's own detail named, and c9ccf5e swept only the rename half.
  - id: BR-4
    disposition: not-addressed
    note: |
      voice.go:51 still branches on !flagSet || flag == "", decided by flag == "" alone.
  - id: BR-5
    disposition: not-addressed
    note: |
      Two sites, audiourl.go:50 and :58; re-measured, -locale '%zz' still errors instead of warning.
  - id: BR-6
    disposition: not-addressed
    note: |
      CDN half now settled by probe (jalapeno_es_us and _es_es both 200); the dictionary half is
      still unconfirmed — capture.sh:91 es_words lacks jalapeno and the Larousse is not installed here.
  - id: BR-7
    disposition: not-addressed
    note: |
      Five rows still unticked, no Revisions entry; plus plan:57 stale and no localeHelp row.
  - id: BR-9
    disposition: not-addressed
    note: |
      Reproduced at HEAD — {-locale gb} then /lang es requests sycophantic_es_gb_*, silent thereafter.
```
