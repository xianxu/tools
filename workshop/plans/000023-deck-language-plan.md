# Language Mode Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `define` operates in one language at a time — the deck, the review session, the audio and (M2) the dictionary all follow a declared mode.

**Architecture:** The language is a property of the vocab DIRECTORY, read once at the boundary and passed inward as a CONSTRUCTOR parameter. Nothing infers it and nothing re-reads it mid-call. `/lang es` does not mutate a shared setting: it writes the file and REBUILDS the language-scoped store deps through the same construction path the boundary used, so there is one builder called twice rather than two ways to be in a language. M1 makes the mode exist and scopes the deck and audio to it; M2 makes the *dictionary* follow it too, through private DictionaryServices symbols resolved at run time with a fallback to today's behaviour.

**Tech Stack:** Go, cgo against CoreServices. No new module dependencies. `dlsym` for the private surface so a missing symbol degrades instead of failing to link.

---

## Measurements this plan rests on

Re-run **2026-08-28**, independently of the issue's own probe. `#17 BR-21` is why: a claim repeated from another session is not a measurement.

| claim | measured | consequence |
|---|---|---|
| the nine private DCS symbols resolve via `dlsym` | **all 9 true** | M2 is possible at all |
| dictionaries available | **87** | the curated list is a small subset of a large set |
| `DCSCopyAvailableDictionaries` returns a **CFSet** | **confirmed — and treating it as a CFArray CRASHES** (`-[__NSCFSet objectAtIndex:]: unrecognized selector`) | stronger than the issue's "order is unspecified": getting this wrong is a hard crash, so the seam must use `CFSetGetValues` |
| `mesa` through `com.apple.dictionary.es.DGLEV` | **"nombre femenino — Mueble formado por un tablero horizontal…"** | the mode can be *fully* correct, not merely correct-at-filing |
| `sycophantic` through DGLEV | **no entry** | "not a word in this language" becomes answerable |
| strictly-monolingual candidates, `es` | **exactly 1** — `com.apple.dictionary.es.DGLEV` | metadata alone suffices for Spanish |
| strictly-monolingual candidates, `en` | **6** — `NOAD`, `ODE`, `AppleDictionary`, `OAWT`, `OTE`, `com.apple.accessibility.dictionary.TTY` | **metadata alone CANNOT choose**: two are thesauruses and one is an accessibility dictionary. This is what forces a curated default rather than a rule. |
| CDN: `sycophantic_es_es_1` | **404** | languages are disjoint; a mis-set mode yields a MISS, never the wrong word's audio |
| CDN: `madrugar--_us_1`, `madrugar--_es_1` | **404, 404** | the legacy `/sounds/oxford/` path is English-only |
| CDN: 404 vs 200 latency | **~300–600ms vs ~40ms** | asking for one language instead of guessing is a ~10× saving per miss avoided |

The last three carry over from `#27`'s plan, whose fallback design this issue's mode replaces.

## Two defects found while designing (both fold into M1)

**1. A runtime FILE reaches none of the three guards.** `user-model.md` is written by `--reflect` into the **current directory**, carries inferred claims about the learner, and is **not gitignored** — `git check-ignore -v user-model.md` matches nothing (re-verified 2026-08-28, exit 1). `store.RuntimeDirs` single-sources the runtime *directories* into `.gitignore`, `TestNoTrackedRuntimeState` (the index) and `TestNoRuntimeStateInHistory` (the history), so the guard structurally cannot see a runtime *file*. Nothing has leaked, so this is latent — and M1 adds a second runtime file, which is why the fix is the class and not the instance.

**2. A tracked fixture squats on that name.** `cmd/define/testdata/golden/user-model.md` is tracked and reachable from HEAD. So the class fix is blocked until the fixture moves: pointing the index and history guards at `RuntimeFiles` by basename fails on contact, and an un-anchored `user-model.md` in `.gitignore` also matches that golden path — harmless while it is tracked, silently un-addable after a `git rm`. Task 1 renames the fixture and states the rule the rename encodes: **a runtime-artifact basename is reserved; a fixture may not squat on one**, and the index guard is what says so out loud.

## Scope check

Two milestones with genuinely separate boundaries:

- **M1 — the mode exists and the deck follows it.** Ships a working single-language `define`; English behaviour unchanged. `mesa` in Spanish mode still returns the English entry — filing is correct, defining is not yet.
- **M2 — the dictionary follows it too.** The private-symbol seam, curated defaults, override, honest degradation, conformance check. This is where `mesa` becomes Spanish.

M1 is useful alone (deck grouping, scoped review, correct audio). M2 carries the OS-version risk and is the half that can degrade.

`#27` (pronunciation locale variants — `es_es` vs `es_us`, the θ/seseo help text, live CDN conformance) is `blocked` on this issue; M1 folds in only the *language plumbing* it needs, and pins the interim locale rule below so `#27` inherits a stated rule rather than an accident.

## Decisions this plan pins

The five questions round 1 of the plan gate found unanswered. Each is settled here once; the tasks below implement these and do not re-decide them.

### D1 — `/lang` rebuilds the language-scoped deps; it does not mutate a store (PQ-1)

Verified in the tree: `openStore` derives `history`, `capture`, `deck`, `vocab` and `usage` from ONE `*store.YAML` (`main.go:202-212`), pinned by `TestOpenStoreSharesOneHighlightSet` (`vocab_test.go:148`); `repl` takes `d deps` **by value** (`repl.go:181`); both loops rebuild `commandCtx` per dispatch (`repl.go:322`, `replraw.go:222`); and `runEditor` captures the highlight set ONCE into a local before the loop (`voc := vocabularyFor(d, opt)`, `replraw.go:79`), reading it at `:135` and `:213`.

That last one is the hazard: reassigning the loop's `d` alone would leave the editor highlighting from the OLD language's set.

- **The store stays immutable.** `lang` is a `NewYAML` constructor parameter, matching the existing comment on `dir` — *"a parameter, not a policy"*. Nothing re-reads `lang.txt` below the boundary.
- **The re-derived set is `d.lang`, `opt.voice`, the deck triple (`deck`, `capture`, `vocab`) and the editor's `voc`** — see the Revisions entry below; naming only the triple here is what let `opt.voice` ship un-re-derived. `history` reads `events/` and `usage` reads `usage/`, neither of which is language-scoped — and re-deriving `history` would put a second, *unloaded* `History` beside the one `runEditor` already `Load()`ed at `replraw.go:73`. The usage cache keeps its original lang-carrying `*YAML`, which is harmless because `usageDir()` ignores the language; that is stated so a reviewer does not have to re-derive it.
- **One builder, called twice (ARCH-DRY).** Factor the set out of `openStore` into the `newLangDeps` closure, which returns a `langDeps`; `openStore` calls it and so does the `/lang` closure. (This plan named a different shape first — see Revisions, which is where that belongs.)
- **`commandCtx` gains `lang store.Lang` and `setLang func(store.Lang) error`,** on the `setTimes` precedent (`command.go:162-163`) and for the same reason: a command has no business reaching the rest of the deps.
- **`setLang` is the loop's closure**, and it assigns BOTH the loop's `d` and — in `runEditor` — the captured `voc`, because that local is the one thing a `d` reassignment cannot reach.
- **Unlike `/sound`, a nil `setLang` does not make `/lang` refuse.** `/sound` with no session has nothing to do; `/lang` still has its durable half. The split is: `store.WriteLang` always runs when there is a directory; the session re-derive runs only when there is a session. A one-shot `define /lang es` therefore sets the language for subsequent invocations and says so. With no directory at all, `/lang` reports that, the way `/history` reports a nil deck.
- **Test pin:** extend the existing invariant — after `/lang es`, `capture` and `vocab` still share ONE highlight set, and it is the SPANISH one.

### D2 — `-locale` is English-only; the language supplies the rest (PQ-2)

`main.go:272` already defines `-locale` (`"pronunciation locale: us or gb"`, default `"us"`), threaded through `speak` (`main.go:611`) into `AudioCandidates` (`audiourl.go:21`), whose 2022 path hardcodes `_en_`. So today's default silently means "English, US".

The M1 rule, stated now because `#27` — which owns locale policy — is blocked on this issue:

- **The default locale is the language code itself, except `en` → `us`.** One rule, one exception, and the exception is the measured one: the CDN's English recordings are `_en_us_` / `_en_gb_` while Spanish is `_es_es_`.
- **`-locale` is honoured for `en` only.** Its documented values are `us or gb`, which are English variants. Passed together with a non-English `-lang`, it prints one line saying locale variants for that language are `#27`'s and that the default was used — a diagnostic, not an error, and not a silently ignored flag.
- **No unmeasured URL form is ever built.** `madrugar_es_us_1.mp3` is exactly the shape this rule prevents.
- `#27` replaces the exception table with its real locale policy; nothing above `voice` moves when it does.

### D3 — the migration is language-blind, and a collision leaves the flat file alone (PQ-3)

- **`MigrateToLanguages(dir, warn)` takes no language.** The destination is `DefaultLang`, hardcoded, and the ambiguity is removed structurally rather than by a convention a caller can get wrong. The justification is not "English is the default" but a fact about the files: a flat deck was written by a pre-language `define`, which only ever consulted the English dictionary and requested `_en_us_` audio. Whatever the headwords are, the *entries* are English.
- **It is language-BLIND by design, and says so.** This tree's own deck is `words/ligament.yaml` and `words/madrugar.yaml` — the second is this issue's headline Spanish example, and it will land in `words/en/`. Nothing can tell; a heuristic here is the inference the whole design rejects. So the migration **prints what it moved** and names the remedy (`mv words/en/madrugar.yaml words/es/`). It prints nothing when it moves nothing.
- **Collision rule: the subdirectory wins, the flat file survives, and it is reported.** With `words/mesa.yaml` and `words/en/mesa.yaml` both present, the destination is never overwritten and the source is never deleted. A surviving flat file is inert — after this change `wordsDir()` is `words/<lang>/`, so nothing reads `words/*.yaml` — which makes "leave it" strictly non-destructive on the one artifact here that cannot be regenerated.
- **Two tests, because the earlier single assertion was true only of one case:** non-colliding → moved, flat gone; colliding → destination byte-identical, flat file still there, warning names it. Plus the idempotence run: twice, same result.
- **It runs in `openStore`**, once at the boundary. `/lang`'s re-derive goes through `newLangDeps`, not `openStore`, so it does not re-run; if that ever changes, a second run is a no-op scan by construction.

### D4 — the language reaches the dictionary through the constructor (PQ-4)

`Dictionary` is `Lookup(word string) (string, error)` (`dict.go:14-16`) and `systemDictionary()` takes no argument (`dict_darwin.go:86`, `dict_stub.go:18`).

- **The seam signature becomes `systemDictionary(lang store.Lang) Dictionary`.** The interface itself does not move — `Lookup(word)` stays — so nothing downstream of the seam learns about languages. Four call sites: `main.go:78`, `dict_conformance_test.go:27`, `live_property_test.go:44`, plus the stub's own definition.
- **The fixture corpus becomes per-language: `testdata/entries/<lang>/*.txt`,** with today's 30-odd fixtures moved to `entries/en/` and `loadFakeDictionary(dir, lang)` globbing one language's subdirectory. It keeps the empty-corpus hard failure, which is what stops a vacuous invariant suite. `testDict(t)` keeps its English meaning for the ~12 existing callers; `testDictFor(t, lang)` is the new one. Four direct `loadFakeDictionary` call sites move with it.
- **`capture.py` grows an optional dictionary identifier** and, when given one, resolves it through `DCSCopyAvailableDictionaries` + `DCSDictionaryGetIdentifier` — **`CFSetGetCount`/`CFSetGetValues`, not array indexing** — and passes the ref to `DCSCopyTextDefinition` instead of NULL. `capture.sh` gains a Spanish word list captured through `com.apple.dictionary.es.DGLEV`: `mesa`, `bonito`, `once`, `real`, `madrugar`. The existing byte floor applies unchanged; it is what makes a sandboxed capture fail loudly instead of writing silence.
- **"No entry" needs no fixture.** `sycophantic` is absent from `entries/es/` and that absence IS the assertion — the fake returns `ErrNoEntry` for anything it does not hold.

### D5 — `RuntimeFiles` reaches all three guards, and the fixture moves out of its way (PQ-5)

`store/yaml.go:34-41` states the contract: a runtime name must reach `.gitignore`, the index guard and the history guard. `RuntimeFiles` meets it in full, or it is a restatement of the pattern rather than the pattern.

- **Rename `cmd/define/testdata/golden/user-model.md` → `user-model.golden.md`** (one reference, `usermodel_test.go:46`). The rule it encodes: a runtime-artifact basename is reserved.
- **`isRuntimeFile(basename)` beside `isRuntimeDir(seg)`,** consumed by both `TestNoTrackedRuntimeState` and `TestNoRuntimeStateInHistory`. A future tracked file named `user-model.md` or `lang.txt` then fails loudly at the index, which is the same shape of protection `RuntimeDirs` already gets — and it turns the shadow this plan just removed into something that cannot come back silently.
- **The persisted setting is `lang.txt`, not `lang`.** `.gitignore` entries here must be un-anchored (`go test` runs in the package directory — the comment in `.gitignore` is emphatic about it), and a bare `lang` would also hide any *directory* named `lang` anywhere in the tree, which the basename guard cannot see. `lang.txt` has no plausible directory or source collision, and it matches `user-model.md`'s precedent of a runtime file with an extension.
- **Third `.gitignore` guard, not a fourth mechanism:** `TestGitignoreCoversRuntimeFiles` mirrors `TestGitignoreCoversRuntimeDirs`, including its anchored-pattern check.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `Lang` | `cmd/define/store/lang.go` | new |
| `store.RuntimeFiles` | `cmd/define/store/yaml.go` | new |
| `voice` / `localeFor` | `cmd/define/voice.go` | new |
| `AudioCandidates` | `cmd/define/audiourl.go` | modified |
| `dictMeta` / `chooseDictionary` / `dictionaryFor` | `cmd/define/dictselect.go` | new (M2) |

- **`Lang`** — a validated language tag: `"en"`, `"es"`. A named type over `string` so a bare directory name cannot be passed where a language is meant, and because it becomes a PATH SEGMENT — an unvalidated string there is a directory traversal, which is why `ParseLang` is the only constructor from input.
  - **Relationships:** 1:1 with a vocab directory's persisted setting; 1:N with the words filed under it.
  - **DRY rationale:** First occurrence. It exists so `words/<lang>/`, the audio `voice`, and (M2) the dictionary choice all read the same value rather than each parsing a flag.
  - **Future extensions:** a third language is a table row, not a code change.

- **`store.RuntimeFiles`** — the runtime *files* define writes into the working directory, beside the existing `RuntimeDirs`.
  - **DRY rationale:** Not a new pattern — the completion of an existing one, and completed to the same three consumers per D5. `RuntimeDirs` exists precisely so a new runtime artifact reaches `.gitignore`, the index guard and the history guard together; it covers only directories, so `user-model.md` slipped through and `lang.txt` would have been next.
  - **Future extensions:** any future runtime file is one entry, and three guards follow.

- **`voice` / `localeFor`** — language plus regional variant for a recording, e.g. `voice{Lang: "es", Locale: "es"}`, and the pure function that derives it from the mode plus the `-locale` flag per D2.
  - **DRY rationale:** kills a mix-up hazard rather than a duplication: `"es"` is a legal value of BOTH fields, so two positional strings are transposable at 20+ call sites and a struct is not.
  - **NOTE the change from `#27`'s plan:** there is no `voices()` ordering policy and no fallback. The mode supplies exactly one language; `AudioCandidates` builds candidates for it alone.
  - **Future extensions:** `#27` replaces `localeFor`'s exception table with the θ/seseo policy without touching callers.

- **`dictMeta` / `chooseDictionary` / `dictionaryFor`** *(M2)* — the decision "which installed dictionary serves language L", and the pure function that makes it from dictionary metadata plus a curated list.
  - **Relationships:** 1:1 with a `Lang`. Pure over a slice of metadata records, so it is unit-testable with no CoreServices at all.
  - **DRY rationale:** first occurrence, and the reason it is pure and separate is that the *policy* is the contested part (measurement proves metadata alone picks a thesaurus for English) while the *lookup* is mechanical.
  - **Future extensions:** a learner-supplied identifier is one more branch here, not a new mechanism.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `store.YAML` | `cmd/define/store/yaml.go` | modified | the vocab directory |
| `ReadLang` / `WriteLang` | `cmd/define/store/lang.go` | new | the persisted setting |
| `MigrateToLanguages` | `cmd/define/store/migrate.go` | new | an existing flat deck AND a flat learner model |
| `langDeps` / `newLangDeps` | `cmd/define/main.go` | new | every language-derived dependency |
| `/lang` command | `cmd/define/lang_cmd.go` | new | the TUI namespace |
| `-lang` flag | `cmd/define/main.go` | new | operator input |
| `installedDictionaries` | `cmd/define/dict_darwin.go` | new (M2) | private DictionaryServices symbols |
| `fakeDictionary` | `cmd/define/dict_fake_test.go` | modified | the system dictionary |

- **`store.YAML`** — gains a language: `words/<lang>/<slug>.yaml`. A constructor parameter (D1); the store never reads the persisted setting itself.
- **`ReadLang` / `WriteLang`** — read and write `lang.txt`. `ReadLang` degrades an absent, unreadable or garbage file to `DefaultLang`: the learner asked for a word, not for a configuration audit.
- **`langDeps` / `newLangDeps`** — the language-derived set as a TYPE, built by one closure that `openStore` and `/lang` both call (D1). `langDeps` is EMBEDDED in `deps`, so adopting a switch is one whole-struct assignment and a member added later is adopted automatically. The set was a list in a doc comment first and went stale three times; see the close-review Revisions entry.
- **`installedDictionaries`** *(M2)* — resolves the private symbols with `dlsym` and returns metadata records. Its fake widens from a set of entries to a set of *dictionaries*, which is what lets `chooseDictionary` and the "no entry in this language" path be tested with no CoreServices. Conformance is on-demand with `-tags conformance`, routed through `conformance.SkipOrFail` (`#25`).

---

## Chunk 1 (M1): the mode exists

Tests are named by what they pin, not transcribed — the strategy line is the part worth agreeing on before the code exists.

### Task 1: `Lang`, and the runtime-FILE guard the class needs

**Files:** create `store/lang.go`, `store/lang_test.go`; modify `store/yaml.go`, `.gitignore`, `cmd/define/repo_guard_test.go`, `cmd/define/usermodel_test.go`; rename `testdata/golden/user-model.md` → `user-model.golden.md`.

- [x] **Step 1 — rename the squatting fixture first.** `git mv` the golden and update its one reference. Doing this first is what lets the guards below be written at full strength rather than with an exception.
- [x] **Step 2 — write the three failing guard tests.** `TestGitignoreCoversRuntimeFiles` mirrors the `RuntimeDirs` one (entry present, and NOT anchored — `go test` runs in the package directory); `TestNoTrackedRuntimeState` and `TestNoRuntimeStateInHistory` each gain an `isRuntimeFile(basename)` arm. Run them: the first fails on `undefined: store.RuntimeFiles`, and the other two must be *seen* to fail before the rename would have been proved unnecessary.
- [x] **Step 3 — add `RuntimeFiles = []string{"user-model.md", "lang.txt"}`** beside `RuntimeDirs`, with the comment stating why a directory list could not see either file, and the two un-anchored `.gitignore` lines.
- [x] **Step 4 — run, and confirm git agrees**, because a passing test about `.gitignore` is not the same as `.gitignore` working: `git check-ignore -v user-model.md lang.txt` must now match both (it currently matches neither).
- [x] **Step 5 — `ParseLang`, test first.** The strategy is the input space, not the body: `"en"`, `"es"`, case-folded `"ES"`, a trailing newline (`ReadLang` hands it a file's bytes), and the rejections — empty, `"english"`, `"e/s"`, and `"../etc"`, which is the one that matters because the value becomes a path segment. Deliberately NOT a list of known languages: the CDN and the installed dictionaries own what exists, and an unknown-but-well-formed tag degrades to "no recording, no dictionary", which is honest.
- [x] **Step 6 — implement, run, commit.** `git commit -m "#23 M1: Lang, and the runtime-FILE guard user-model.md needed"`

### Task 2: the deck is scoped by language

**Files:** modify `store/yaml.go` (`NewYAML`, `wordsDir`), `store/yaml_test.go`.

- [x] **Step 1 — failing test:** `TestWordsAreScopedByLanguage`. An English store and a Spanish store over the SAME directory file into `words/en/` and `words/es/`, and — the assertion that is the point — the Spanish deck cannot see the English word. `#5`'s schedule interleaves by due-date, so mixing would not even be incidental.
- [x] **Step 2 — implement.** `NewYAML(dir, lang, warn)` with an empty lang meaning `DefaultLang`; `wordsDir()` returns `words/<lang>/`. `eventsDir()`, `usageDir()` and `userModelFile()` are deliberately untouched: language is a deck dimension, not an event one. A review event names a word; which deck it came from is the deck's business, and splitting the log would make "how much did I study today" a join.
- [x] **Step 3 — fix the call sites.** Enumerated 2026-08-28: **31 across 8 files** — `main.go`, `store/yaml.go`, `store/yaml_test.go`, `reflect_run_test.go`, `reflect_conformance_test.go`, `askrun_test.go`, `capture_test.go`, `history_store_test.go`, plus a fixture under `puretest/testdata/impure/`. Mechanical, but priced as a `cross-cutting-refactor` and not a one-liner — `#24`'s `Apply` signature change was the same shape with nine sites.
- [x] **Step 4 — run the package, commit.**

### Task 3: migrate an existing flat deck

**Files:** create `store/migrate.go`, `store/migrate_test.go`.

- [x] **Step 1 — failing tests, three of them**, one per rule in D3: moved-and-gone for the ordinary case; destination byte-identical and flat file surviving for a collision; and a twice-run migration reaching the same state. The third is the one that stops a future rewrite-in-place, on the one artifact here that cannot be regenerated.
- [x] **Step 2 — implement `MigrateToLanguages(dir string, warn io.Writer) error`.** No language parameter (D3). It moves `words/*.yaml` into `words/en/`, skips subdirectories the way `Deck()` already does (`yaml.go:103-115`), never overwrites, and reports what it moved plus the `mv` remedy — printing nothing when it moves nothing.
- [x] **Step 3 — call it once in `openStore`, run, commit.**

### Task 4: `-lang`, the persisted setting, and `/lang`

**Files:** create `cmd/define/lang_cmd.go`, `lang_cmd_test.go`; extend `store/lang.go`; modify `main.go` (flag, `options`, `openStore`, `newLangDeps`), `command.go` (`commandCtx`, table), `repl.go`, `replraw.go`.

- [x] **Step 1 — failing store tests:** `ReadLang` on an unset directory is `DefaultLang`; a written language round-trips; a garbage file degrades to the default rather than failing the lookup.
- [x] **Step 2 — failing command tests:** `/lang` reports; `/lang es` switches, persists, and re-derives; `/lang xx` is refused with the tag echoed; `/lang es` with no session still writes and says which half it did (D1).
- [x] **Step 3 — the boundary.** `options` gains `lang store.Lang`; the flag is `-lang` (`"language for this invocation: en, es (default: the directory's setting)"`). Precedence, stated once in `main.go`: the flag wins for THIS invocation and does not persist; otherwise the directory's setting; otherwise English. `-lang` exists so a script can ask a question without mutating state.
- [x] **Step 4 — `newLangDeps` and `setLang`.** Factor the language-scoped triple out of `openStore`; add `lang` and `setLang` to `commandCtx`; wire the closure in BOTH loops, and in `runEditor` assign the captured `voc` as well as `d` — the local at `replraw.go:79` is the thing a `d` reassignment cannot reach.
- [x] **Step 5 — extend the wiring pin.** `TestOpenStoreSharesOneHighlightSet` has a sibling: after `/lang es`, `capture` and `vocab` are still ONE set, and it is the Spanish one. Mutation check: reassigning `d` without reassigning `voc` must redden it.
- [x] **Step 6 — run, commit.**

### Task 5: the recording follows the mode (folds `#27`'s plumbing)

**Files:** create `voice.go`; modify `audiourl.go`, `audiourl_test.go`, `main.go` (`speak`), `fetch_fake_test.go`.

- [x] **Step 1 — failing tests.** Spanish: `AudioCandidates("madrugar", voice{Lang: "es", Locale: "es"})` yields the two 2022-path URLs and **no legacy pair** — measured 2026-08-28, `madrugar--_us_1` and `madrugar--_es_1` are both 404 while `sycophantic--_us_1` is 200, so the legacy `/sounds/oxford/` path is English-only and a Spanish candidate on it is a guaranteed miss at ~450ms. English: the existing four URLs, unchanged — this is the regression that must not happen. `localeFor`: `en`→`us`, `en`+`-locale gb`→`gb`, `es`→`es`, `es`+`-locale gb`→`es` plus the diagnostic (D2). Plus the `fakeCDN` walk from `#27`'s plan, using `stripHost`, `cdn.urls`, `cdn.source`, `cdn.Requested` — all verified present at `fetch_fake_test.go:44-64`.
- [x] **Step 2 — implement.** `speak`'s `locale string` becomes `v voice`; `AudioCandidates` takes a `voice` and builds `_<lang>_<locale>_`; no `voices()` and no fallback, because the mode supplies the language.
- [x] **Step 3 — run, commit.**

### Task 6: `--play` and `--forget` inherit the mode

**Files:** modify `main.go` (`--play`, `--forget`), `play_loop.go`.

- [x] **Step 1 — failing test:** a `--play` session in Spanish mode offers only Spanish words, and `--forget` plus `d`-in-`--play` remove from the current language's deck and leave the other language's untouched. The negative half is the assertion — a delete that reaches the wrong deck is the failure mode worth pinning.
- [x] **Step 2 — wire the `Lang` through, run, commit.** Both already act through `d.deck`, so this is mostly proof rather than change; if it turns out to be free, the test is still the deliverable.

### Task 7: M1 docs and close

- [x] **Step 1 — sweep, do not remember:** `grep -rniE "words/|deck|locale|language" README.md atlas/ --include='*.md'`.
- [x] **Step 2 —** README gets `-lang` and `/lang`; `atlas/define.md` gets the deck layout, the *language is a deck dimension, not an event one* rule, and D2's interim locale rule so `#27` inherits it in writing.
- [x] **Step 3 —** `sdlc milestone-close --issue 23 --milestone M1`

## Chunk 2 (M2): the dictionary follows the mode

### D6 — `usage/` is not scoped, because the FEED is English and will not be consulted

Decided here rather than discovered at M2's boundary, which is the review's
instruction and the lesson `user-model.md` taught at M1's.

Measured in the tree: `httpFeed.Fetch` hardcodes
`&hl=en-US&gl=US&ceid=US:en` (`news.go:60`). The feed is English **by
construction**, and the cache is keyed by word alone (`usage/<slug>.yaml`).

So a Spanish session asking it about `mesa` gets English news about a
flat-topped hill or a city in Arizona — the wrong language AND the wrong sense —
and caches it under a key an English session also uses.

**The rule is the one M2 exists to apply, so apply it consistently: no data beats
the wrong language's data.** The news feed is consulted only for the language it
actually serves. In any other language `bothSources.news` is nil, which
`Usages` already handles, and the examples come from that language's own
dictionary entry — which M2 makes correct, and which the issue's measurement
already showed carries usage examples (`madrugar` → *"Levantarse muy temprano,
especialmente al amanecer"*).

**And `usage/` therefore needs no language dimension:** nothing writes it outside
English. Adding one now would create `usage/es/` holding English sentences, which
is scoping without correctness — the worse half of the trade. When `#10` or `#18`
makes the feed language-aware, scoping becomes *required*, and that is the moment
to add it. Written down here so it is a decision with a trigger rather than a
thing someone rediscovers.

**Not a live saving today, and worth saying so rather than overclaiming:**
`UsageSource.Usages` has no production caller yet — `d.usage` is plumbing for
`#10`. So this costs nothing now and prevents something later: the moment `#10`
wires it up, a Spanish lookup would otherwise pay a live Google News request per
word, up to the 20s client timeout, for items `containsWord` will almost always
discard. A first draft of this section claimed the saving was already being
made.


### Task 8: `chooseDictionary`, pure over metadata

**Files:** create `dictselect.go`, `dictselect_test.go`.

- [x] **Step 1 — failing test, and the table IS the measurement.** The installed-metadata fixture is the real one from 2026-08-28: `NOAD`, `OTE` (a thesaurus) and `com.apple.accessibility.dictionary.TTY` all indexing `en`→`en`; `es.DGLEV` monolingual `es`; `OxfordSpanish` bilingual. Expectations: `en`→`NOAD`, `es`→`DGLEV`, `de`→not found (so the caller reports no entry rather than answering from English). A second test pins the honest-degradation case: no curated match → `ok=false`, and the caller falls back to today's NULL behaviour rather than guessing.
- [x] **Step 2 — implement:** narrow to dictionaries indexing L, prefer strictly monolingual, then prefer a curated identifier; the curated list is a short honest list, not a rule, because nothing in the metadata says "general-purpose dictionary".
- [x] **Step 3 — mutation check, three named mutations, each reddening a named row:** drop the curated preference (English picks a thesaurus); accept bilingual as monolingual (Spanish picks `OxfordSpanish`); return `ok` for an unindexed language.
- [x] **Step 4 — commit.**

### Task 9: the `dlsym` seam, its fake, and the per-language corpus

**Files:** modify `dict_darwin.go`, `dict_stub.go`, `dict_fake_test.go`, `testdata/capture.py`, `testdata/capture.sh`; move `testdata/entries/*.txt` → `testdata/entries/en/`.

- [x] **Step 1 — the corpus move and the capture path (D4), first**, because the fixtures have to exist before the test that reads them: `capture.py` takes an optional dictionary identifier and resolves it through `DCSCopyAvailableDictionaries` + `DCSDictionaryGetIdentifier`; `capture.sh` captures `mesa`, `bonito`, `once`, `real`, `madrugar` through `com.apple.dictionary.es.DGLEV` into `entries/es/`. Unsandboxed, with the existing byte floor.
- [x] **Step 2 — failing test:** `loadFakeDictionary(dir, lang)` serves one language; `mesa` in Spanish is the Spanish entry, and `sycophantic` in Spanish is `ErrNoEntry` — an absence, which needs no fixture and is the assertion the issue calls newly answerable.
- [x] **Step 3 — implement the seam.** `systemDictionary(lang store.Lang)` at four call sites; **`CFSetGetValues`, not `CFArrayGetValueAtIndex`** — verified 2026-08-28 that treating the result as a CFArray crashes with `-[__NSCFSet objectAtIndex:]: unrecognized selector`, in a cgo frame where the cause is not obvious. Every private symbol is `dlsym`'d.
- [x] **Step 4 — prove the fallback**, which is a Done-when row and not a nicety: with one symbol name deliberately misspelled, the binary must still define an English word through today's NULL path.
- [x] **Step 5 — say which dictionary is in use**, so a wrong pick on a machine with a different set installed is visible rather than puzzling. Run, commit.

### Task 10: live conformance for the private surface

**Files:** modify `dict_conformance_test.go`.

- [x] **Step 1 —** assert every symbol in `dcsPrivateSymbols` resolves (the resolver needs three; "nine" was how many the survey probed) and that `mesa` through the Spanish choice is a Spanish entry, routed through `conformance.SkipOrFail` (`#25`) so an unreachable dictionary SKIPS by default and FAILS under `CONFORMANCE_STRICT`.
- [x] **Step 2 —** run unsandboxed in BOTH env states; sandboxed it must skip, not fail.
- [x] **Step 3 —** atlas + README for the dictionary selection, then `sdlc close --issue 23`.

## Risks

**The private symbols can vanish on an OS update.** The whole of M2 rests on nine undocumented symbols. Mitigated by `dlsym` + fallback to today's behaviour (so the failure mode is "M1's tool", not a crash) and by Task 10's conformance check. Recorded as a Done-when row in the issue, not as a note.

**CFSet, not CFArray.** Measured: getting this wrong is an uncaught ObjC exception, not a wrong answer.

**The curated list is wrong on someone else's machine.** 87 dictionaries here; another Mac has a different set. Mitigated by the explicit override and by saying which dictionary is in use.

**Deck migration touches the one irreplaceable artifact.** `MigrateToLanguages` moves rather than rewrites, never overwrites, and leaves a colliding flat file in place — inert, because nothing reads `words/*.yaml` after Task 2. The twice-run test is what must stop any future version that rewrites.

**A mid-session `/lang` leaves a stale captured local.** The concrete instance is `runEditor`'s `voc` (`replraw.go:79`); Task 4 Step 5's mutation check is what keeps it honest. The class — "a loop local derived from `d` before the loop" — is why `setLang` lives in the loop that owns those locals rather than in `commandCtx`.

**`events/` deliberately NOT scoped.** Language is a deck dimension, not an event one. "How much Spanish did I study" is a join over the deck, not a split log — and splitting an append-only artifact later is worse.

## Done-when → task map

| Done-when row | Task |
|---|---|
| `words/<lang>/`, existing decks migrated rather than orphaned | 2, 3 |
| `--play -lang es` reviews Spanish only; default reviews English only | 4, 6 |
| `/lang` reports and switches; the setting survives the session | 4 |
| a one-shot `define madrugar` uses the persisted language | 4 |
| a word shared with English is filed AND DEFINED in the current language | 2 (filed), 8–9 (defined) |
| a word absent from the current language reports no entry | 9 |
| private symbols resolved at run time, seam falls back to NULL | 9 |
| a live conformance check says loudly when the private surface moves | 10 |
| `--forget` and `d`-in-`--play` remove from the right deck | 6 |
| schedule and event log unchanged | 2 (`events/` deliberately not scoped) |

## Notes for the reviewer

- **Every symbol and line number this plan names was checked against the tree on 2026-08-28.** The round-1 gate findings were each verified before being answered rather than accepted on their face: `openStore`'s single `*YAML` (`main.go:202-212`), the editor's captured `voc` (`replraw.go:79`), `-locale`'s hardcoded `_en_` (`audiourl.go:21`), `Deck()`'s directory skip (`yaml.go:103-115`), the flat corpus glob (`dict_fake_test.go:24-49`), `capture.py`'s NULL, and the tracked `testdata/golden/user-model.md`. All held. One is worse than the finding stated: `git check-ignore -v user-model.md` matches nothing *today*, so the defect is live, not hypothetical.
- **`RuntimeFiles` is scope this issue grew deliberately,** and D5 takes it to all three consumers rather than to `.gitignore` alone — the instance-versus-class distinction the gate's own `family:` slug names.
- **`#27`'s `voices()` fallback is deleted, not adapted.** A mode does not guess. See `workshop/plans/000027-pronunciation-locale-plan.md`'s Revisions entry.

## Revisions

### 2026-08-28 — round 1 of the plan-quality gate: five blocking findings answered

**Reason.** `sdlc change-code --issue 23` refused with PQ-1 (Critical) and PQ-2…PQ-5 (Important), plus PQ-6 and PQ-7 (Minor). Ledger: `workshop/plans/000023-deck-language-plan-gate.md`.

**Delta.**

- **New section "Decisions this plan pins" (D1–D5)**, one per blocking finding, each verified against the tree first.
  - **D1 (PQ-1)** — names the `/lang` mechanism: a `setLang` closure on `commandCtx`, on the `setTimes` precedent, re-deriving only the language-scoped triple through a new `deckDeps` builder shared with `openStore`, and reassigning `runEditor`'s captured `voc`. Reconciles "read once at the boundary" as *one builder called twice*, with the store still immutable. Adds the wiring pin and its mutation check.
  - **D2 (PQ-2)** — states the `-lang`/`-locale` precedence M1 ships: default locale is the language code, `en`→`us` excepted; `-locale` honoured for English only, with a diagnostic otherwise; no unmeasured URL form is ever built.
  - **D3 (PQ-3)** — `MigrateFlatDeck` loses its language parameter (destination hardcoded to `en`, justified by what the pre-language tool actually wrote, not by "English is the default"); collision rule pinned as subdirectory-wins/flat-file-survives-and-is-reported; the single "flat file must not survive" assertion becomes three tests.
  - **D4 (PQ-4)** — `systemDictionary(lang)` as the seam signature with `Lookup(word)` unchanged; corpus moves to `testdata/entries/<lang>/`; `capture.py` grows a dictionary identifier and captures through a chosen ref; five Spanish captures named; "no entry" needs no fixture.
  - **D5 (PQ-5)** — `RuntimeFiles` reaches all three guards, which requires renaming the squatting golden fixture; the persisted file becomes `lang.txt` rather than `lang`, because an un-anchored `lang` would also hide any directory of that name and the basename guard cannot see that.
- **The design-defect section now carries both halves** — the unignored runtime file AND the tracked fixture that blocks the class fix.
- **PQ-6 (Minor) — the plan no longer reproduces the diff.** Roughly 250 lines of full function bodies and transcribed test tables are replaced by one strategy line per risky decision: the traversal input for `ParseLang`, the three named mutations for `chooseDictionary`, the three rules for the migration, the negative assertion for `--forget`.
- **PQ-7 (Minor) — the false claim is corrected.** "English, because that is what every existing deck holds" is replaced by the true one (the flat deck's *entries* are English because the pre-language tool only ever consulted the English dictionary), and the migration is now explicitly language-blind, prints what it moved, and names the `mv` remedy for `madrugar` — this tree's own live deck.
- **Task 1 grows the fixture rename and two more guard arms; Task 3 splits out into `store/migrate.go`; Task 9 gains the corpus move and capture-path work as its FIRST step.** Each is scope the findings named, and each is priced when the estimate is derived.

### 2026-08-28 — M1 boundary review: REWORK, one Critical and four Important

**Reason.** `sdlc milestone-close --issue 23 --milestone M1` returned REWORK.
Sidecar: `workshop/plans/000023-deck-language-m1-review.md`.

**Delta.**

- **C1 (Critical) — the `/lang` re-derive set was UNDER-ENUMERATED, and that is
  the finding, not the one missing line.** D1 said "only the language-scoped
  triple is re-derived: `deck`, `capture`, `vocab`" plus `runEditor`'s `voc`.
  `opt.voice` belongs to the same enumeration and was not in it, so a mid-session
  `/lang es` left the fetch loop asking for the four **English** URLs — including
  the two legacy `/sounds/oxford/` ones this very milestone gated to English for
  costing ~450 ms per guaranteed miss. The deck went Spanish; the pronunciation
  did not, contradicting README and atlas text shipped in the same range.
  - The fix is the CLASS: `applyLang` is now the one function that applies a
    language, it lists every member with the reason each is one, and it states
    the rule that generates the list — *anything derived from the language before
    a switch must be re-derived by it.* It also names `d.dict` as the member M2
    will add, so the next one is written down before it is missed.
  - `applyVoice` makes the voice derivation ONE function with two callers (the
    boundary and the switch). Two expressions for the same derived value is what
    the bug was.
  - Pinned at the level the bug lived at: `TestLangSwitchReDerivesEverythingDown
    streamOfTheLanguage` drives `/lang es` through `run()` and asserts what the
    CDN was ASKED for, plus a non-vacuity check that Spanish was requested at
    all. A unit test on `voiceFor` cannot see this — `voiceFor` was always right.
- **I1 — a ticked Done-when row was unpinned.** "A one-shot `define madrugar`
  uses the persisted language" was checked while every test either wrote the
  setting or passed `-lang`; replacing `store.ReadLang(dir)` with
  `store.DefaultLang` left the whole suite green. Now covered both ways (the read
  AND `-lang` still overriding it), and the mutation was re-run to confirm it
  reddens.
- **I2 — the single-source sweep was 4 of 5.** `userModelFile()` hand-wrote
  `"user-model.md"` while `RuntimeFiles[0]` was declared its source, so renaming
  the entry would have moved `.gitignore` and both guards while `SetUserModel`
  kept writing the old name — reopening the exact hole M1 closed. It derives now.
- **I3 — the new Spanish CDN facts had no live conformance row.** The
  English-only gate rests on measurement that lived only in a comment and in a
  fake written to agree with it. `TestCDNStillServesSpanishOnTheExpectedPaths`
  checks both halves against the real CDN; it passes today, which independently
  re-verifies the measurement this plan rests on.
- **I4 — this plan named `deckDeps`, which the tree does not have.** The
  substance (one builder, two callers) shipped as the `newDeck` closure. D1 and
  the Core-concepts table are corrected above so the plan stops describing an
  entity that does not exist.
- **Minors:** `main.go` restored to gofmt-clean; the project file's
  self-contradicting sentence about `git check-ignore` fixed;
  `TestAcceptedLangIsASafePathSegment` rewritten to assert the real property (a
  `filepath.Join` cannot escape the deck directory) rather than a comparison its
  own length check subsumed; `/lang <current>` now persists, because a directory
  with no `lang.txt` is already `en` by default and skipping the write left the
  learner no way to make that explicit.

### 2026-08-28 — M1 boundary review round 2: FIX-THEN-SHIP, BR-2 blocking

**Reason.** Round 2 disposed C1 and I1–I4 and raised BR-2 (Important) plus three
Minors. Sidecar: `workshop/plans/000023-deck-language-m1-review.md`.

**Delta.**

- **BR-2 — `user-model.md` was a REGRESSION M1 introduced, and it is C1's class
  one member further out.** `--reflect` reads the language-scoped deck and wrote
  one shared file, so reflecting in Spanish replaced the English learner model
  and every English answer was then pitched at "A2 — Spanish beginner". The
  atlas text shipped in this range justified leaving it flat with the `events/`
  argument, which does not transfer: an event is a fact about a moment, while
  the model is a SUMMARY OF A DECK. The dividing line is derivation, not storage,
  and the atlas now says so.
  - `userModelFile()` is per-language (`user-model.<lang>.md`);
    `MigrateToLanguages` moves a pre-language `user-model.md` alongside the deck,
    under the same never-overwrite / never-delete / say-what-happened rules.
- **`RuntimeFiles` entries became PATTERNS**, which the per-language model forced
  and which also closes a Minor: `writeBytesAtomic` leaves `.tmp-*` shadows in
  the working-directory ROOT for these two files, where no runtime directory
  covers them. `user-model.??.md` rather than `user-model*.md` so the golden
  fixture is not shadowed again — `??` is exactly a two-letter `Lang`.
  - A pattern cannot name a file, so the writers stopped deriving from the list
    and `TestRuntimeFilePatternsCoverWhatWeWrite` keeps them in step instead. It
    is the stronger check: it asserts the real output of the writing functions,
    and it also asserts the patterns are not loose enough to shadow the fixture.
  - Verified against git itself, not only the test: `git check-ignore -v` matches
    `user-model.md`, `user-model.en.md`, `user-model.es.md`, `lang.txt` and
    `.tmp-abc`, and still does NOT match the golden fixture.
- **Minors.** `applyLang` no longer writes `opt.lang`, whose documented meaning
  is "the `-lang` FLAG, empty when it was not given" — a switch does not
  retroactively make the flag present. `ParseLang`'s comment now says why the tag
  is two letters (the CDN's `_<lang>_<locale>_` shape and the path segment),
  rather than only why there is no whitelist. The user-facing strings that named
  `user-model.md` now name the artifact instead of a filename that varies.
- **One test comment corrected rather than left overclaiming.** A first draft
  asserted the unreadable-model diagnostic still carried the file PATH; the
  store double in that test fails abstractly, so the path is a production-only
  property. The assertion is now on what is actually guaranteed — that the
  message says which artifact failed — with the distinction stated.

### 2026-08-28 — M1 boundary review round 3: two rules, not two instances

**Reason.** Round 3 disposed BR-2..BR-5 and raised BR-6 and BR-7 (both
Important) plus four Minors, with the gate reporting *4 repeat families — not
converging: fix rules, not instances.* Sidecar:
`workshop/plans/000023-deck-language-m1-review.md`.

**Delta.**

- **BR-6 — the artifact-rename sweep was 3 of ~12, and one miss was user-facing
  output on the happy path.** `--reflect` printed "wrote user-model.md" while
  writing the per-language file, and the README tells the learner to hand-edit
  that file's `## Corrections` — so following the tool's own output would have
  put their corrections in a file `UserModel()` never reads.
  - The reviewer's instruction was explicit: do NOT fix the instance. The rule —
    *no output line, comment, README line, atlas line or plan line may spell a
    runtime artifact's filename or a symbol the code owns; name the artifact or
    derive the name* — is now MECHANICAL.
    `TestRuntimeArtifactNamesAreSpelledOnceInSource` is a ratchet in this repo's
    established shape: non-test Go may spell a learner-model filename only in
    the two consts that BUILD every such name, and a second spelling fails.
  - It immediately found four sites the manual enumeration had missed, which is
    the argument for mechanising it rather than sweeping again.
  - `store.UserModelName(lang)` is exported so a consumer can name the file
    without restating the scheme.
  - The symbol half is swept too: `atlas/define.md`, `main.go`'s comment and this
    plan named `MigrateFlatDeck`, which the tree does not export. **This is the
    second time this plan has named an absent entity** (round 1's I4,
    `deckDeps`), so the rule is recorded here: a rename in code sweeps every
    prose restatement of the symbol in the SAME commit.
- **BR-7 — the runtime-file guard asserted coverage from hand-typed literals.**
  The reviewer's mutation was decisive: changing the atomic-write prefix left the
  suite green while the shadow beside the two root-level runtime files matched no
  `.gitignore` pattern, so a `git add -A` would commit a partial learner model.
  - Rule applied: *a runtime artifact's name has exactly one producing function;
    guards, migrations and tests derive from it and never restate it.* There are
    now four producers (`UserModelName`, `langFileName`, `newTempFile`,
    `userModelLegacy`), and `RuntimeFiles` BUILDS its two pattern entries from
    them — so a scheme change moves the pattern and the `.gitignore` guard fails
    loudly instead of silently ceasing to cover anything.
  - Both mutations re-run and confirmed to redden now: the prefix change and a
    learner-model scheme change each fail `TestGitignoreCoversRuntimeFiles` by
    name.
  - `migrateUserModel` builds its destination through `UserModelName`, and the
    migration tests assert through `UserModel()` rather than a literal path — so
    a scheme change cannot leave the migration writing a file nothing reads.
- **Minors.** The detached PATTERNS comment merged into `RuntimeFiles`' doc;
  `MigrateToLanguages`' doc says it moves two named artifacts rather than "a
  directory"; `wordsDir`'s comment corrected (it claimed `userModelFile` was
  unscoped, false since BR-2); README now states `-locale` is English-only, three
  lines from where it is demonstrated; and `d.history`'s identity across a switch
  is asserted rather than only argued in a comment.
- **Carried to the close review, not fixed here:** `usage/` is not
  language-scoped and is correctly excluded today because the news feed takes no
  language — but M2's plan must decide it deliberately, since `mesa`'s usage
  examples in a Spanish session are not the English ones and the cache is keyed
  by word alone. Discovering that at M2's boundary is exactly how the learner
  model arrived at M1's.

### 2026-08-28 — M1 boundary review round 4: FIX-THEN-SHIP, fixes bundled into the close

**Reason.** Round 4 disposed BR-6..BR-10 and returned FIX-THEN-SHIP. BR-11
(Important) was demoted past the round cap and would NOT block — and no later
gate picks it up, so leaving it would have meant shipping a finding nothing
would ever revisit. Fixed. Per `#174` these fixes are bundled into the
milestone-close commit rather than triggering a fifth review.

**Delta.**

- **BR-11 — the artifact-name rule binds prose, but was enforced over `*.go`
  only, and the hand-swept half left a LIVE FALSE claim.** The project file still
  said "a single `user-model.md`" after BR-2 made it one per language. Sixth
  finding in `comment-contract-drift`, so again the rule, not the site:
  `TestProseDoesNotSpellStaleRuntimeArtifactNames` now enforces it over the three
  artifact kinds a reader takes as describing the tool AS IT IS — README,
  `atlas/`, and the project portfolio view.
  - **The scope is the interesting decision.** It deliberately does NOT bind
    issues, plans, lessons or `workshop/history/`: those are dated RECORDS, and a
    Spec or Log naming what was true when written is correct — rewriting them to
    match today is the actual lie. `currentTruthOnly` strips the record-bearing
    parts by SHAPE rather than by a list of names (`## Revisions`, `## Log`, and
    any `###` block carrying `**closed:**`), so a new record section is covered
    without anyone remembering to add it.
  - Mutation-checked: reintroducing the exact false sentence reddens it.
- **The orphaned doc comment (BR-10's shape, still live).** `Forget`'s doc ran
  into `type newsFile` with no blank line, so godoc attached "Forget removes one
  word file" to the wrong declaration. Reattached.
- **Minors, each fixed as the family's own rule — a path or name a test asserts
  against comes from the function that produces it.**
  - `TestYAMLIgnoresInterruptedWrites` planted its fixture at the pre-`#23` flat
    path while `wordsDir()` had moved, so `Deck()` never listed it and the test
    exercised nothing. Moved INTERNAL as `TestDeckIgnoresInterruptedWrites`,
    deriving both the directory and the temp name from their producers. Its body
    is now valid YAML, which the reviewer's own caveat showed matters: with the
    old unparseable fixture, deleting `Deck()`'s suffix check left the suite
    green at BOTH base and HEAD. It now reddens — a pin that was decorative is
    load-bearing.
  - `store.LangFileName()` exported, so package `main`'s tests stop restating
    `lang.txt`. The absence assertion at `lang_cmd_test.go` was the dangerous
    one: a negative assertion against a hand-typed name passes VACUOUSLY the
    moment the writer renames the file.
  - `TestAOneShotLookupReadsThePersistedLanguage` gained its negative half; the
    positive one alone survives a double write, which is what a precedence bug
    looks like.
- **Carried into M2's plan, not fixed here:** `usage/` is correctly unscoped
  today because the news feed takes no language, but it becomes a scoping
  candidate the moment the dictionary follows the mode — `mesa`'s usage examples
  in a Spanish session are not the English ones, and the cache is keyed by word
  alone. Decide it in M2's plan; discovering it at M2's boundary is exactly how
  the learner model arrived at M1's.

### 2026-08-28 — close review: FIX-THEN-SHIP, four blocking findings as rules

**Reason.** `sdlc close` returned FIX-THEN-SHIP with BR-13..BR-16 open and the
gate reporting *3 repeat families — not converging: fix rules, not instances.*
Sidecar: `workshop/plans/000023-deck-language-close-review.md`. Per `#174` the
fixes are bundled into the close commit rather than triggering a sixth review.

**Delta.**

- **BR-13 — the language-derived enumeration failed a THIRD time, and this is
  what finally made it structural.** M2 made the news feed's presence a function
  of the language (D6), and `applyLang`'s doc comment still called `d.usage` "not
  language-scoped" — so the gate held at the boundary and not across a
  mid-session `/lang`. Both directions were wrong: a session started in English
  kept the English feed after `/lang es`, and one started in Spanish kept
  `news == nil` forever after `/lang en`.
  - The class fix is not another line in `applyLang`. The set is now a TYPE —
    `langDeps` — built by one function that `openStore` and `/lang` both call,
    so a member constructed at the boundary is necessarily re-derived at the
    switch. An enumeration in a doc comment did not survive one milestone; a
    struct returned from one builder cannot be half-adopted.
  - Pinned in both directions, since losing the feed is the quieter failure.
- **BR-14 — the artifact-name rule's SYMBOL half was still unenforced after four
  recurrences** (`deckDeps`, `MigrateFlatDeck`, `dictChoice`,
  `dcsDictionaries`). Both existing ratchets count filenames only.
  - `TestPlanTablesNameEntitiesThatExist` makes a plan a CONSUMER of the tree:
    every Core-concepts row naming an identifier at an EXISTING path must find it
    declared there. A row whose file does not exist yet is skipped, because a
    plan legitimately precedes its code — a row pointing at a real file that does
    not declare the name is a lie.
  - It immediately caught both entities the review named plus two in `#27`'s
    plan that M1 had invalidated (`voices`, deleted rather than adapted; and the
    Spanish CDN conformance test, which shipped here under a different name).
  - **"The nine symbols" was wrong in four documents.** Nine is what the issue's
    survey FOUND; the resolver needs three. Rather than sync a count, the count is
    gone: `dcsPrivateSymbols` is the one producer, the conformance test walks it
    member by member, and the prose says "the private symbols". A count in prose
    is a restatement with nothing keeping it true.
  - Also swept: a duplicate `warnf` byte-identical to `warnTo` (ARCH-DRY, and the
    comment claiming `warnTo` was "the one place" was false), a stranded pre-M2
    README paragraph about Dictionary.app configuration, and `atlas/index.md`
    still calling this "NOAD word lookup".
- **BR-15 — both fallback branches had no automated test on any platform.** The
  Done-when row about degrading was ticked on a manual misspell-and-run
  experiment, which proved it once on one machine and pinned nothing. The policy
  is now `dictionaryFor`, pure over metadata and extracted from the cgo shell
  (ARCH-PURE), so all three outcomes unit-test anywhere. This matters more than
  it sounds: the reviewer measured a shell context returning ONE dictionary, in
  which every run takes the second fallback.
- **BR-16 — `atlas/repo-guards.md` did not name the two ratchets this range
  added.** It is the catalogue a contributor reads before adding a doc line, so a
  guard missing from it is a guard they will trip blindly. Added, with the
  records-vs-current-truth scope rule.
- **Minors, both real bugs in the cgo:** `selectedDictionary.Lookup` overwrote a
  status-3 "dictionary vanished" with a later status-1 "word absent", collapsing
  exactly the two the C side keeps distinct — the first non-absence error now
  wins. And `installedDictionaries` indexed position `i` across N separately
  copied CFSets, whose order this file's own comment calls unspecified; it is one
  copy described in one pass now.
- **Capture path now matches production.** English fixtures were captured through
  the NULL search while production selects curated identifiers; they agreed only
  because this host's active set happened to match. `capture.sh` walks the same
  curated list, and the re-captured corpus is byte-identical to the committed one
  — including `iPhone`, which comes from the second book.

### 2026-08-28 — close review round 6: the set becomes a TYPE, embedded

**Reason.** Round 6 returned FIX-THEN-SHIP with BR-14 and BR-20..BR-24 open and
the gate reporting *8 repeat families — not converging*. Sidecar:
`workshop/plans/000023-deck-language-close-review.md`.

**Delta.**

- **BR-22 — a struct was NOT enough, and this is the third correction to the same
  rule.** `langDeps` existed, but `openStore` and `applyLang` each copied its
  four fields BY HAND, so a fifth member stayed forgettable at two sites — the
  exact failure the struct was introduced to prevent. `langDeps` is now
  EMBEDDED in `deps`, so adoption is `d.langDeps = d.newLangDeps(l)`: one
  assignment, and a member added later is adopted without anyone remembering to.
  Cost: six composite literals in tests now nest it. Worth it.
- **BR-22b — the milestone's HEADLINE wiring had no test at all.** `d.newDict`
  was set at zero call sites; deleting BOTH the boundary derivation and the
  `/lang` re-derivation left the whole suite green while production would
  dereference a nil `Dictionary`. Pinned at both moments, and the first draft of
  that pin REPLICATED the boundary derivation inline and stayed green when it was
  deleted — a test reimplementing the code it pins. It drives `run()` now, and
  both mutations redden.
- **BR-23 — pure logic inside the platform shell, and it cost real things.** The
  status fold was untestable where it lived, so BR-17's fix shipped INOPERATIVE:
  `case lookupNoEntry` overwrote unconditionally, so a vanished dictionary
  followed by an ordinary miss still reported "no entry". And `parseDictRecords`,
  `parseLangPairs` and `baseLang` carried doc comments claiming to be "testable
  without CoreServices" while sitting behind `//go:build darwin` — `GOOS=linux
  go vet ./cmd/define/` FAILED, which this range broke and which `dict_stub.go`
  simultaneously claimed to prevent. All four moved to `dictselect.go`;
  `ErrLookupFailed` moved to `dict.go` beside `ErrNoEntry` to follow them.
- **BR-24 — "one producer" was two.** `dcsPrivateSymbols` is walked by the
  conformance check while `dcs_resolve` hand-writes the same names in C, so
  adding a `dlsym` would leave the check green over a symbol nobody verifies —
  BR-7's shape applied to the list M2's entire OS-version risk rests on. cgo
  gives Go no view of the preamble, so the copies cannot be unified; they are
  COMPARED instead, by reading the source.
- **BR-20 — the new plan-table guard passed on a COMMENT.** Its fallback clause
  accepted any occurrence in the file, and a stale `newDeck` row was green solely
  because one comment still mentioned the old name. Clause removed; the row is
  corrected to `langDeps`/`newLangDeps`.
- **BR-21 — `atlas/define.md` still described the pre-BR-13 design**, including
  the exact claim BR-13 disproved ("usage… would put a second, unloaded
  History"). Rewritten, and it now records WHY the set is a type and embedded
  rather than only that it is.

### 2026-08-28 — close review round 7: the symbol rule gets its human-written half

**Reason.** Round 7 disposed ten findings and left three: BR-20 re-opened as
*not-addressed* (the `newDeck` sweep was three sites short), plus BR-28 and
BR-29. Sidecar: `workshop/plans/000023-deck-language-close-review.md`.

**Delta.**

- **BR-20/BR-29 — the sweep missed three `newDeck` comments in the very commit
  that added the guard against exactly this.** That is the ninth finding in the
  family and the argument for `retiredSymbolNames`: a rename cannot be detected
  automatically, because only the person doing it knows the old name — so
  renaming writes ONE row, and `TestNoArtifactNamesARetiredSymbol` makes the rest
  mechanical across non-test Go, README, `atlas/` and active plans.
  - It immediately found a live D1 bullet still naming `deckDeps` to explain what
    the plan first called it. That backreference belongs in `## Revisions`, where
    it already was — a live design section states the design as it IS.
- **BR-28 — `capture.sh` hand-restated the curated identifiers**, which is the
  same divergence that already bit once (English captured through the NULL search
  while production selected curated ids, agreeing only by coincidence). A shell
  script cannot import a Go map, so `TestCaptureScriptUsesTheCuratedDictionaries`
  COMPARES them — the same move as the cgo-preamble symbol guard, applied to the
  other boundary Go cannot see into.
- **Minor, and a real defect:** `capture.py` returned a `DCSDictionaryRef`
  borrowed from the copied CFSet from inside a `try` whose `finally` released
  that set, then passed the ref to `DCSCopyTextDefinition` — a use-after-release
  that worked only because CoreServices happens to keep dictionaries alive. It
  `CFRetain`s now and releases after the lookup. `dict_darwin.go` gets the same
  sequence right by ordering; two implementations of one boundary should not
  disagree about handle lifetime.
