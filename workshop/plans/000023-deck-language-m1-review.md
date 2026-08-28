# Boundary Review — tools#23 (milestone M1)

| field | value |
|-------|-------|
| issue | 23 — deck grouped by language, one language per --play session |
| repo | tools |
| issue file | workshop/issues/000023-deck-language.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 2e929fc56d1edc6b04af61100116a02abb7d9146..30ee9e29e62aa901c599291aa39620aa688fa2ed |
| command | sdlc milestone-close --issue 23 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-28T12:12:35-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 is careful, well-documented work: the deck really is per-language, the migration is genuinely non-destructive and pinned by four tests, the `RuntimeFiles` guard closes the class rather than the instance, and the `&voc` pointer in `sessionSetLang` catches the one hazard a `deps` swap cannot reach. What blocks SHIP is that the *other* thing derived from the language before the switch — `opt.voice` — is never re-derived, so a mid-session `/lang es` leaves the fetch loop asking for the four **English** URLs, including the legacy `/sounds/oxford/` pair this range deliberately gated to English. I verified it end-to-end in a scratch copy (`/lang es` then a lookup → `sycophantic_en_us_1.mp3`, `_2`, `sycophantic--_us_1`, `_2`), and it directly contradicts README and `atlas/define.md` text shipped in this same range ("Everything follows it — the deck a word files into, the words `--play` offers, and the recording that is fetched"). Separately, the Done-when row "a one-shot `define madrugar` uses the persisted language" is checked but unpinned: replacing `store.ReadLang(dir)` with `store.DefaultLang` at `main.go:257` leaves the entire suite green.

## 1. Strengths

- **`cmd/define/command.go:324-345` + `cmd/define/vocab_test.go:405-457`** — the `vocPtr` parameter is the real insight of this milestone, and it is pinned by assertions that actually redden (the `notWant` half), not by a test written to agree with the fix.
- **`cmd/define/store/migrate.go`** — language-blind by construction with the justification stated as a fact about the files, never overwrites, never deletes, prints the `mv` remedy, and `store/migrate_test.go` covers all four rules (move, collision, idempotence, silence) plus the "leaves other languages alone" case the plan didn't ask for.
- **`cmd/define/audiourl_test.go:139-201`** — `TestTheFetchLoopAsksOnlyForTheSessionsLanguage` asserts what is *requested*, not what the pure function returns. That is the right altitude, and its negative case is the one that would catch a regression.
- **`cmd/define/repo_guard_test.go:295-345`** — `legacyRuntimeFilePaths` as an exact-set ratchet rather than a comment, with the vacuity guards (`len(RuntimeFiles) == 0`, `seen == 0`) kept. The anchored/un-anchored `.gitignore` check is carried over from the directories guard rather than reinvented (ARCH-DRY).
- **`cmd/define/voice.go:36-55`** — `localeFor` *returns* the complaint instead of printing it, and `main.go:529` prints it once where both halves first meet. Textbook ARCH-PURE.

## 2. Critical findings

**C1 — `cmd/define/main.go:529` / `cmd/define/command.go:324`: `/lang` does not re-derive `opt.voice`, so the recording keeps following the *old* language for the rest of the session.**

`opt.voice` is resolved exactly once, in `run()` after `withStore`. `sessionSetLang` reassigns `d.lang`, `d.deck`, `d.capture`, `d.vocab` and `*vocPtr` — but `opt` is the loop's own value copy and nothing writes `opt.voice`. `playAnnounced → speak(…, opt.voice, …)` (`main.go:726`) therefore uses the pre-switch voice forever.

Verified in a scratch copy (`run(nil, …, stdin: "/lang es\nsycophantic\n")`, fake CDN):
```
requested: [ …/sycophantic_en_us_1.mp3  …/sycophantic_en_us_2.mp3
             …/sounds/oxford/sycophantic--_us_1.mp3  …--_us_2.mp3 ]
```
Both loops are affected. The legacy `/sounds/oxford/` pair is exactly the ~450ms-per-miss the language gate in `audiourl.go:52-58` was added to stop paying, and after `/lang es` the session pays it on every lookup.

Fix sketch: `options` already carries `locale string`; add the `flagSet` bool beside it, and have `sessionSetLang` recompute `opt.voice` the way `setTimes` writes back — i.e. take `*options` (or an `setVoice func(store.Lang)` closure supplied by each loop, on the `setTimes` precedent) and call `voiceFor(l, opt.locale, opt.localeSet)` after `d.lang = l`. Pin it with the fetch-loop-level test above, driven through `run()` with `/lang es` in stdin — a unit test on `voiceFor` will not catch this.

## 3. Important findings

**I1 — `cmd/define/main.go:255-258`: nothing pins that a one-shot run reads `lang.txt`.**
Done-when row *"A one-shot `define madrugar` uses the persisted language, with no session to inherit from"* is ticked, but every test either writes the setting (`lang_cmd_test.go:140`, `vocab_test.go:457`) or passes `-lang`/`opt.lang`. Mutation check: replacing `lang = store.ReadLang(dir)` with `lang = store.DefaultLang` leaves `go test ./cmd/define/...` fully green. Fix: a test that `store.WriteLang(dir, "es")`, then `run(ctx, []string{"madrugar"}, …)`, then asserts the word landed in `words/es/` (and that `-lang en` still overrides it — the precedence, not just the read).

**I2 — `cmd/define/store/yaml.go:87` (ARCH-DRY / ARCH-PURPOSE): `userModelFile()` hand-writes `"user-model.md"` while `RuntimeFiles[0]` is declared the single source.**
`langFile()` (`store/lang.go:51`) derives its name from `RuntimeFiles[1]` and says why ("so the guards … and the code that writes it cannot disagree about the name"); the writer of the *other* runtime file does not. Renaming `RuntimeFiles[0]` would move `.gitignore` and both guards while `SetUserModel` kept writing the old name — reopening the exact hole this milestone closed. Shadow-sweep: 4 of 5 consumers derive, 1 restates. Fix: `filepath.Join(y.dir, RuntimeFiles[0])`.

**I3 — `cmd/define/fetch_conformance_test.go:33` (ARCH-MOCK): the new Spanish CDN facts the fake models have no live conformance row.**
The legacy-path gate in `audiourl.go:52-58` rests on measured facts (`madrugar_es_es_1` = 200; `madrugar--_es_1` and `madrugar--_us_1` = 404). Those live only in comments and in a fake that was written to agree with them. `TestCDNStillServesTheExpectedPaths` already exists for precisely this class of English fact. Fix: two `head()` rows — Spanish primary is 200, Spanish legacy is not — so a CDN move surfaces loudly instead of as silent silence in Spanish sessions.

**I4 — plan Core-concepts table names `deckDeps` in `cmd/define/main.go`; no such entity exists.**
D1 and Task 4 Step 4 both specify "factor the triple out of `openStore` into `deckDeps(dir, lang, clk, warn)`". The code instead uses an inline closure `newDeck` (`main.go:260-269`) stored on `storeDeps`/`deps`. The *substance* — one builder, called by `openStore` and by `/lang` — is delivered, and `atlas/define.md` describes the delivered shape correctly, so I'm calling this Important rather than the Critical the table cross-check nominally assigns. The plan is now the only artifact claiming an entity the tree does not have; it needs a `## Revisions` entry (see §7).

## 4. Minor findings

- `cmd/define/main.go:285` — not gofmt-clean: `usage:` is misaligned in the `storeDeps` literal. `gofmt -l ./cmd/` reports `cmd/define/main.go`, and it is clean at the base commit.
- `workshop/projects/define-learn.md:434` — self-contradicting sentence: "was ignored by nothing at all (`git check-ignore` matched it before this milestone and matches it now)". Should read *matched nothing before this milestone*.
- `cmd/define/store/lang.go:51` — `RuntimeFiles[1]` couples the settings filename to slice order; a named constant indexed into the list would survive a reorder. (Consistent with the existing `RuntimeDirs[0..2]` style, so this is a note, not a change request.)
- `cmd/define/lang_cmd.go:46` — `/lang en` in a directory with no `lang.txt` reports "already defining in en" and writes nothing, so the setting stays implicit. Correct, but it means "I made it explicit" is unavailable.
- `cmd/define/store/lang_test.go:48-60` — `TestAcceptedLangIsASafePathSegment`'s inner `bad` loop re-checks `len(l) != 2` on every iteration; the `bad` comparison is subsumed by the length check. Harmless, slightly misleading as a "property" test.

## 5. Test coverage notes

- Genuinely strong: `TestWordsAreScopedByLanguage` and `TestEventsAreNotScopedByLanguage` pin both halves of the deck/event split; `TestForgetActsOnTheCurrentLanguageOnly` leads with the negative assertion; `TestLangSwitchKeepsOneHighlightSetAndItIsTheNewLanguages` covers the invariant a rebuild is most likely to break; `TestMigrateFlatDeck*` covers all four migration rules.
- Two gaps, both listed above: C1 (no test drives `/lang` and then observes what the CDN is asked for) and I1 (no test observes the persisted setting being *read*).
- `d`-in-`--play` deletion is delivered by construction — `play_loop.go:154` calls the same `d.deck.Forget` as `--forget` — but the Done-when row that names it is only tested through `--forget`. Given the shared seam, a one-line addition to `TestForgetActsOnTheCurrentLanguageOnly`'s sibling would close it; not worth blocking on.
- No test asserts that `openStore`'s second, flat store keeps `history`/`usage` unscoped after a `/lang` switch. The rationale is documented at `main.go:271-276`; a `d.history` identity assertion in the switch test would make it structural.

## 6. Architectural notes

- **ARCH-DRY — pass, with I2.** `newDeck` is one builder called twice, `MigrateFlatDeck` derives `"words"` from `RuntimeDirs[0]`, `isRuntimeFile` mirrors `isRuntimeDir`, and the `.gitignore` guard reuses the anchoring rule rather than restating it. The single exception is `userModelFile()` (I2).
- **ARCH-PURE — pass.** `ParseLang`, `defaultLocale`, `localeFor`, `voiceFor`, `parseLangArgs`, `AudioCandidates` are all pure and unit-tested with no IO; `localeFor` returning its complaint instead of printing it is the model of the principle. `ReadLang`'s error flattening is IO-shell behaviour, deliberate and documented, with `/lang` as the reporting escape hatch.
- **ARCH-PURPOSE — flag (C1).** The issue's Spec commits to "*Everything* inherits the mode: lookups file into that language's deck, `--play` reviews that language, and the audio asks for that language's recording." Two of three follow `/lang`; the third does not. The class here is "state derived from the language before the switch" — `d.deck`/`d.capture`/`d.vocab` and the editor's `voc` were enumerated and swept; `opt.voice` is the member of the same enumeration that was missed. Worth writing the enumeration down in the fix rather than patching the one site.
- **ARCH-MOCK — flag (I3).** The CDN fake and the production path share one boundary (`rebasedSource` runs the real `AudioCandidates` output), which is right. What's missing is the conformance half for the newly-added Spanish facts. For M2, the plan's `dcsDictionaries` + `-tags conformance` design is the correct shape; hold it to the same standard — the fake must model dictionary *identity* across calls, not just entry presence, or `chooseDictionary`'s curated-list branch will be untestable against reality.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000023-deck-language-plan.md`:

- **`deckDeps` was delivered as the `newDeck` closure.** D1 and the Core-concepts integration table name a package-level `deckDeps(dir, lang, clk, warn)` in `cmd/define/main.go`. The implementation instead builds the language-scoped triple as a closure over `dir`/`clk`/`warn` inside `openStore` and carries it on `storeDeps.newDeck` / `deps.newDeck`. The invariant D1 was protecting (one builder, called by `openStore` and by `/lang`) holds; only the name and shape differ. Update the table row to `newDeck` and adjust D1's bullet so the plan stops naming a function the tree does not have.
- **The `/lang` re-derive set was under-enumerated.** D1 says "only the language-scoped triple is re-derived: `deck`, `capture`, `vocab`" (plus `runEditor`'s `voc`). `opt.voice` belongs to that enumeration and was not listed, which is how C1 shipped. State the full set, and state the rule that generates it: *anything derived from the language before the switch must be re-derived by it.*
