# Language Mode Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `define` operates in one language at a time — the deck, the review session, the audio and (M2) the dictionary all follow a declared mode.

**Architecture:** The language is a property of the vocab DIRECTORY, read once at the boundary and passed inward as a parameter. Nothing infers it. M1 makes the mode exist and scopes the deck and audio to it; M2 makes the *dictionary* follow it too, through private DictionaryServices symbols resolved at run time with a fallback to today's behaviour.

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

## A defect found while designing (fold into M1)

`user-model.md` is a runtime artifact `--reflect` writes into the **current directory**, containing inferred claims about the learner, and it is **NOT gitignored** — verified with `git check-ignore -v user-model.md` (no match). `store.RuntimeDirs` single-sources the runtime *directories* into `.gitignore` and `TestGitignoreCoversRuntimeDirs`, so the guard structurally cannot see a runtime *file*.

Nothing has leaked (`git ls-files | grep user-model.md` returns only a golden fixture and issue docs), so this is latent. It matters here because **M1 adds a second runtime file** (the persisted language), and adding it without generalising the guard would be the instance rather than the class — the exact shape `.gitignore`'s own comment says "has now cost four review rounds across three issues".

## Scope check

Two milestones with genuinely separate boundaries:

- **M1 — the mode exists and the deck follows it.** Ships a working single-language `define`; English behaviour unchanged. `mesa` in Spanish mode still returns the English entry — filing is correct, defining is not yet.
- **M2 — the dictionary follows it too.** The private-symbol seam, curated defaults, override, honest degradation, conformance check. This is where `mesa` becomes Spanish.

M1 is useful alone (deck grouping, scoped review, correct audio). M2 carries the OS-version risk and is the half that can degrade.

`#27` (pronunciation locale variants — `es_es` vs `es_us`, the θ/seseo help text, live CDN conformance) is `blocked` on this issue; M1 folds in only the *language plumbing* it needs.

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `Lang` | `cmd/define/store/lang.go` | new |
| `store.RuntimeFiles` | `cmd/define/store/yaml.go` | new |
| `voice` | `cmd/define/voice.go` | new |
| `AudioCandidates` | `cmd/define/audiourl.go` | modified |
| `dictChoice` | `cmd/define/dictselect.go` | new (M2) |
| `chooseDictionary` | `cmd/define/dictselect.go` | new (M2) |

- **`Lang`** — a validated language tag: `"en"`, `"es"`. A named type over `string` so a bare directory name cannot be passed where a language is meant.
  - **Relationships:** 1:1 with a vocab directory's persisted setting; 1:N with the words filed under it.
  - **DRY rationale:** First occurrence. It exists so `words/<lang>/`, the audio `voice`, and (M2) the dictionary choice all read the same value rather than each parsing a flag.
  - **Future extensions:** a third language is a table row, not a code change.

- **`store.RuntimeFiles`** — the runtime *files* define writes into the working directory, beside the existing `RuntimeDirs`.
  - **DRY rationale:** Not a new pattern — the completion of an existing one. `RuntimeDirs` exists precisely so a new runtime artifact reaches `.gitignore`, the index guard and the history guard together; it covers only directories, so `user-model.md` slipped through and the persisted language would have been next.
  - **Future extensions:** any future runtime file is one entry, and the guard test fails if `.gitignore` does not follow.

- **`voice`** — language plus regional variant for a recording, e.g. `voice{Lang: "es", Locale: "es"}`.
  - **DRY rationale:** kills a mix-up hazard rather than a duplication: `"es"` is a legal value of BOTH fields, so two positional strings are transposable at 20+ call sites and a struct is not.
  - **NOTE the change from `#27`'s plan:** there is no `voices()` ordering policy and no fallback. The mode supplies exactly one language; `AudioCandidates` builds candidates for it alone.
  - **Future extensions:** `#27` adds the θ/seseo locale variants on top without touching callers.

- **`dictChoice` / `chooseDictionary`** *(M2)* — the decision "which installed dictionary serves language L", and the pure function that makes it from dictionary metadata plus a curated list.
  - **Relationships:** 1:1 with a `Lang`. Pure over a slice of metadata records, so it is unit-testable with no CoreServices at all.
  - **DRY rationale:** first occurrence, and the reason it is pure and separate is that the *policy* is the contested part (measurement proves metadata alone picks a thesaurus for English) while the *lookup* is mechanical.
  - **Future extensions:** a learner-supplied identifier is one more branch here, not a new mechanism.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `store.YAML` | `cmd/define/store/yaml.go` | modified | the vocab directory |
| `langFile` | `cmd/define/store/lang.go` | new | the persisted setting |
| `/lang` command | `cmd/define/command.go` | new | the TUI namespace |
| `-lang` flag | `cmd/define/main.go` | new | operator input |
| `dcsDictionaries` | `cmd/define/dict_darwin.go` | new (M2) | private DictionaryServices symbols |
| `fakeDictionary` | `cmd/define/dict_fake_test.go` | modified | the system dictionary |

- **`store.YAML`** — gains a language: `words/<lang>/<slug>.yaml`.
  - **Injected into:** everything that reads the deck. The language is a CONSTRUCTOR parameter, matching the existing comment on `dir`: *"a parameter, not a policy — who chooses it stays a one-line question at the boundary."* The store does not read the persisted setting itself.
- **`langFile`** — reads and writes the vocab directory's language.
  - **Injected into:** `main.go` at the boundary, once.
  - **Why persisted, not session-scoped:** a one-shot `define madrugar` has no session to inherit from. `/sound` is the precedent for a `/`-command, and language deliberately differs from it in exactly this way.
- **`dcsDictionaries`** *(M2)* — resolves the private symbols with `dlsym` and returns metadata records.
  - **State model of its fake:** `fakeDictionary` already models a set of entries; M2 widens it to a set of *dictionaries*, each with an identifier, index/description languages, and entries. That is what lets `chooseDictionary` and the "no entry in this language" path be tested with no CoreServices.
  - **Conformance cadence:** on-demand with `-tags conformance`, routed through `conformance.SkipOrFail` (`#25`).

---

## Chunk 1 (M1): the mode exists

### Task 1: `Lang`, and the runtime-file guard it needs

**Files:**
- Create: `cmd/define/store/lang.go`, `cmd/define/store/lang_test.go`
- Modify: `cmd/define/store/yaml.go` (add `RuntimeFiles`)
- Modify: `.gitignore`
- Modify: `cmd/define/repo_guard_test.go`

- [ ] **Step 1: Write the failing guard test first — it is the pre-existing defect**

In `repo_guard_test.go`, beside `TestGitignoreCoversRuntimeDirs`:

```go
// The runtime FILES define writes, covered the way the directories are.
//
// RuntimeDirs exists so a new runtime artifact reaches .gitignore, the index
// guard and the history guard together — and it covers directories only, so
// user-model.md slipped through: `git check-ignore -v user-model.md` matched
// nothing, and --reflect writes it into the CURRENT directory with inferred
// claims about the learner in it. Nothing has leaked, but the language file this
// issue adds would have been the second instance.
func TestGitignoreCoversRuntimeFiles(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	lines := map[string]bool{}
	for _, l := range strings.Split(string(b), "\n") {
		lines[strings.TrimSpace(l)] = true
	}
	for _, f := range store.RuntimeFiles {
		if !lines[f] {
			t.Errorf(".gitignore has no un-anchored %q entry — define writes it into the "+
				"working directory and a git add -A would commit it", f)
		}
		if lines["/"+f] {
			t.Errorf(".gitignore anchors %q to the repo root; go test runs in the package "+
				"directory, where an anchored pattern does not match", f)
		}
	}
}
```

- [ ] **Step 2: Run it, watch it fail**

Run: `go test ./cmd/define/ -run TestGitignoreCoversRuntimeFiles -v`
Expected: FAIL — `undefined: store.RuntimeFiles`.

- [ ] **Step 3: Add `RuntimeFiles` and the `.gitignore` entries**

In `yaml.go`, beside `RuntimeDirs`:

```go
// RuntimeFiles names every FILE define writes into the working directory.
//
// The sibling of RuntimeDirs, and it exists because that list covers directories
// only. user-model.md is a runtime file — --reflect writes it into the current
// directory, carrying inferred claims about the learner — and it reached none of
// the three places RuntimeDirs was built to reach. The language setting added
// here would have been the second instance, which is why this is a list and not
// two more lines in .gitignore.
var RuntimeFiles = []string{"user-model.md", "lang"}
```

In `.gitignore`, under the existing deck block:

```
user-model.md
lang
```

- [ ] **Step 4: Run it, watch it pass; confirm git agrees**

```sh
go test ./cmd/define/ -run TestGitignoreCoversRuntime -v
git check-ignore -v user-model.md lang     # both must now match
```

- [ ] **Step 5: Write the failing `Lang` test**

```go
package store

import "testing"

// A language tag is validated at the edge, once, so nothing downstream has to
// wonder whether "ES " or "" is a language.
func TestParseLang(t *testing.T) {
	for _, tc := range []struct {
		in      string
		want    Lang
		wantErr bool
	}{
		{in: "en", want: "en"},
		{in: "es", want: "es"},
		{in: "ES", want: "es"},   // case-folded: a directory name is lowercase
		{in: " es\n", want: "es"}, // a file read keeps its newline
		{in: "", wantErr: true},
		{in: "english", wantErr: true},
		{in: "../etc", wantErr: true}, // it becomes a PATH SEGMENT; traversal is not a language
		{in: "e/s", wantErr: true},
	} {
		got, err := ParseLang(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseLang(%q) = %q, want an error", tc.in, got)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("ParseLang(%q) = (%q, %v), want (%q, nil)", tc.in, got, err, tc.want)
		}
	}
}
```

- [ ] **Step 6: Implement `Lang`**

```go
package store

import (
	"fmt"
	"strings"
)

// Lang is a validated language tag — the name of a deck, the language of a
// recording, and (M2) the language of a dictionary.
//
// A named type rather than a bare string because it becomes a PATH SEGMENT:
// words/<lang>/. An unvalidated string there is a directory traversal, which is
// why ParseLang is the only way to make one from input.
type Lang string

// DefaultLang is what a directory with no setting means. English, because that
// is what every existing deck holds, and a migration that changed the answer
// would be a different feature.
const DefaultLang Lang = "en"

// ParseLang validates and normalises a language tag.
func ParseLang(s string) (Lang, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", fmt.Errorf("empty language")
	}
	// Two lowercase letters. Deliberately NOT a list of known languages: the CDN
	// and the installed dictionaries decide what exists, and a hardcoded list
	// here would be a restatement of a fact they own. An unknown-but-well-formed
	// tag degrades to "no recording, no dictionary", which is honest.
	if len(s) != 2 || s[0] < 'a' || s[0] > 'z' || s[1] < 'a' || s[1] > 'z' {
		return "", fmt.Errorf("not a language tag: %q (want two letters, like en or es)", s)
	}
	return Lang(s), nil
}
```

- [ ] **Step 7: Run, then commit**

```bash
go test ./cmd/define/store/ ./cmd/define/ -run 'Lang|Runtime' -v
git add cmd/define/store/lang.go cmd/define/store/lang_test.go cmd/define/store/yaml.go .gitignore cmd/define/repo_guard_test.go
git commit -m "#23 M1: Lang, and the runtime-FILE guard that user-model.md needed"
```

### Task 2: the deck is scoped by language

**Files:**
- Modify: `cmd/define/store/yaml.go` (`NewYAML`, `wordsDir`)
- Modify: `cmd/define/store/yaml_test.go`

- [ ] **Step 1: Write the failing test**

```go
// Words file under their language, and a Spanish deck cannot see an English word.
func TestWordsAreScopedByLanguage(t *testing.T) {
	dir := t.TempDir()
	en := NewYAML(dir, DefaultLang, nil)
	es := NewYAML(dir, Lang("es"), nil)

	if err := en.Upsert(Word{Text: "sycophantic"}); err != nil {
		t.Fatal(err)
	}
	if err := es.Upsert(Word{Text: "madrugar"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "words", "en", "sycophantic.yaml")); err != nil {
		t.Errorf("English word not filed under words/en: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "words", "es", "madrugar.yaml")); err != nil {
		t.Errorf("Spanish word not filed under words/es: %v", err)
	}

	// The isolation is the point: a sitting must not mix languages even by
	// accident, and #5's schedule interleaves by due-date, so mixing would not
	// even be incidental.
	all, err := es.Deck()
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range all {
		if w.Text == "sycophantic" {
			t.Error("the Spanish deck can see an English word")
		}
	}
}
```

- [ ] **Step 2: Run it, watch it fail** — `NewYAML` takes two arguments.

- [ ] **Step 3: Implement**

```go
func NewYAML(dir string, lang Lang, warn io.Writer) *YAML {
	if lang == "" {
		lang = DefaultLang
	}
	return &YAML{dir: dir, lang: lang, warn: warn}
}

// wordsDir is per-language; events and usage are NOT.
//
// A review event names a word and a verdict; which deck it came from is the
// deck's business, and splitting the log would make "how much did I study today"
// a join. The issue says this explicitly: language is a deck dimension, not an
// event one.
func (y *YAML) wordsDir() string { return filepath.Join(y.dir, RuntimeDirs[0], string(y.lang)) }
```

- [ ] **Step 4: Fix call sites — there are 31, across 8 files**

```sh
grep -rn "NewYAML(" --include='*.go' . | grep -v "func NewYAML"
```

Enumerated 2026-08-28: `main.go`, `store/yaml.go`, `store/yaml_test.go`,
`reflect_run_test.go`, `reflect_conformance_test.go`, `askrun_test.go`,
`capture_test.go`, `history_store_test.go`, plus a fixture under
`puretest/testdata/impure/`. Mechanical, but count it as a
`cross-cutting-refactor` when the estimate is derived, not as a one-liner —
`#24`'s `Apply` signature change was the same shape and its nine call sites were
priced that way.

- [ ] **Step 5: Run the package, then commit**

```bash
go test ./cmd/define/... && git commit -am "#23 M1: words/<lang>/, with events left whole"
```

### Task 3: migrate an existing deck

**Files:**
- Modify: `cmd/define/store/yaml.go`
- Test: `cmd/define/store/yaml_test.go`

- [ ] **Step 1: Write the failing test**

```go
// An existing flat deck moves under words/en/ rather than being orphaned.
//
// Idempotent and non-destructive: run twice, and a file already migrated is left
// alone rather than overwritten. A learner's deck is the one thing here that
// cannot be regenerated.
func TestMigrateFlatDeckIntoTheDefaultLanguage(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "words"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(dir, "words", "sycophantic.yaml")
	if err := os.WriteFile(old, []byte("text: sycophantic\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ { // idempotent
		if err := MigrateFlatDeck(dir, DefaultLang); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "words", "en", "sycophantic.yaml")); err != nil {
		t.Errorf("word not migrated: %v", err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("the flat file survived the migration; the deck now exists twice")
	}
}
```

- [ ] **Step 2–4: Implement, run, commit**

`MigrateFlatDeck` moves `words/*.yaml` into `words/<lang>/`, skipping anything already in a subdirectory, and is called once at the boundary in `main.go`. Expected: PASS both iterations.

```bash
git commit -am "#23 M1: migrate a flat deck under words/en, idempotently"
```

### Task 4: `-lang`, the persisted setting, and `/lang`

**Files:**
- Create: `cmd/define/store/lang.go` (extend with `ReadLang`/`WriteLang`)
- Modify: `cmd/define/main.go`, `cmd/define/command.go`
- Test: `cmd/define/store/lang_test.go`, `cmd/define/command_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// The setting survives the session, because a one-shot lookup has no session.
func TestLangRoundTrips(t *testing.T) {
	dir := t.TempDir()
	if got := ReadLang(dir); got != DefaultLang {
		t.Errorf("an unset directory = %q, want %q", got, DefaultLang)
	}
	if err := WriteLang(dir, Lang("es")); err != nil {
		t.Fatal(err)
	}
	if got := ReadLang(dir); got != "es" {
		t.Errorf("after WriteLang, ReadLang = %q, want es", got)
	}
}

// An unreadable or garbage setting degrades to the default rather than failing
// the lookup: the learner asked for a word, not for a configuration audit.
func TestAGarbageLangFileFallsBackToTheDefault(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "lang"), []byte("not-a-language"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ReadLang(dir); got != DefaultLang {
		t.Errorf("ReadLang = %q, want the default", got)
	}
}
```

And in `command_test.go`:

```go
// /lang reports, /lang es switches, and the switch persists.
func TestLangCommand(t *testing.T) { /* table over: report, switch, bad tag */ }
```

- [ ] **Step 2: Add the flag, the command row, and the wiring**

- `lang := fs.String("lang", "", "language for this invocation: en, es (default: the directory's setting)")`
- A `{name: "lang", summary: "the language this deck is in", run: runLang}` row in `command.go`'s table.
- In `main.go`, once, at the boundary:

```go
// Precedence: the flag wins for THIS invocation and does not persist; otherwise
// the directory's setting; otherwise English. -lang exists so a script can ask a
// question without mutating state.
lang := store.ReadLang(dir)
if *langFlag != "" {
    parsed, err := store.ParseLang(*langFlag)
    if err != nil { fmt.Fprintf(stderr, "define: %v\n", err); return 2 }
    lang = parsed
}
```

- [ ] **Step 3: Run, then commit**

```bash
go test ./cmd/define/... && git commit -am "#23 M1: -lang, the persisted setting, and /lang"
```

### Task 5: audio asks for the language (folds `#27`'s plumbing)

**Files:**
- Create: `cmd/define/voice.go`
- Modify: `cmd/define/audiourl.go`, `cmd/define/audiourl_test.go`, `cmd/define/main.go`
- Test: `cmd/define/fetch_fake_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// One language, and the legacy path only for English.
//
// Measured 2026-08-28: madrugar--_us_1 and madrugar--_es_1 are BOTH 404 while
// sycophantic--_us_1 is 200, so the legacy /sounds/oxford/ path is English-only
// and a Spanish candidate on it is a guaranteed miss at ~450ms.
func TestAudioCandidatesSpanish(t *testing.T) {
	got := AudioCandidates("madrugar", voice{Lang: "es", Locale: "es"})
	want := []string{
		audioBase + "/pronunciation/2022-03-02/audio/ma/madrugar_es_es_1.mp3",
		audioBase + "/pronunciation/2022-03-02/audio/ma/madrugar_es_es_2.mp3",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// English is unchanged, legacy pair included — this must not regress.
func TestAudioCandidatesEnglishUnchanged(t *testing.T) { /* the existing four URLs */ }
```

Plus the `fakeCDN` walk test from `#27`'s plan (using `stripHost`, `cdn.urls`, `cdn.source`, `cdn.Requested` — all verified present at `fetch_fake_test.go:44-64`).

- [ ] **Step 2–4: Implement `voice`, rewrite `AudioCandidates` for ONE language, run**

No `voices()` and no fallback: the mode supplies the language. `speak`'s `locale string` becomes `v voice`.

- [ ] **Step 5: Commit**

```bash
git commit -am "#23 M1: the recording follows the mode; the legacy path stays English-only"
```

### Task 6: everything else inherits the mode

**Files:**
- Modify: `cmd/define/main.go` (`--play`, `--forget`), `cmd/define/play_loop.go` (drop)

- [ ] **Step 1: Write the failing test** — a `--play` session in Spanish mode offers only Spanish words; `--forget` and `d` remove from the current language's deck.
- [ ] **Step 2–4: Wire the `Lang` through, run, commit.**

### Task 7: M1 docs and close

- [ ] **Step 1: Sweep, do not remember**

```sh
grep -rniE "words/|deck|locale|language" README.md atlas/ --include='*.md'
```

- [ ] **Step 2:** README gets `-lang` and `/lang`; `atlas/define.md` gets the deck-layout change and the *language is a deck dimension, not an event one* rule.
- [ ] **Step 3:** `sdlc milestone-close --issue 23 --milestone M1`

## Chunk 2 (M2): the dictionary follows the mode

### Task 8: `chooseDictionary`, pure over metadata

**Files:**
- Create: `cmd/define/dictselect.go`, `cmd/define/dictselect_test.go`

- [ ] **Step 1: Write the failing test — the table IS the measurement**

```go
// The selection rule, over the metadata actually installed on this machine.
//
// Measured 2026-08-28: requiring every language entry to be L->L gives exactly
// ONE candidate for es (the Larousse) and SIX for en — NOAD, ODE,
// AppleDictionary, OAWT and OTE (both THESAURUSES) and
// com.apple.accessibility.dictionary.TTY. Nothing in the metadata says
// "general-purpose dictionary", so no rule over it can prefer NOAD to a
// thesaurus. That is why a curated list decides and metadata only narrows.
func TestChooseDictionary(t *testing.T) {
	installed := []dictMeta{
		{ID: "com.apple.dictionary.NOAD", Langs: []langPair{{"en", "en"}}},
		{ID: "com.apple.dictionary.OTE", Langs: []langPair{{"en", "en"}}},           // thesaurus
		{ID: "com.apple.accessibility.dictionary.TTY", Langs: []langPair{{"en", "en"}}},
		{ID: "com.apple.dictionary.es.DGLEV", Langs: []langPair{{"es", "es"}}},
		{ID: "com.apple.dictionary.OxfordSpanish", Langs: []langPair{{"es", "es"}, {"en", "es"}}}, // bilingual
	}
	for _, tc := range []struct{ lang Lang; want string; ok bool }{
		{"en", "com.apple.dictionary.NOAD", true},
		{"es", "com.apple.dictionary.es.DGLEV", true},
		{"de", "", false}, // nothing indexes it -> no entry, NOT English
	} {
		got, ok := chooseDictionary(installed, tc.lang)
		if ok != tc.ok || got.ID != tc.want {
			t.Errorf("chooseDictionary(%q) = (%q, %v), want (%q, %v)", tc.lang, got.ID, ok, tc.want, tc.ok)
		}
	}
}

// The curated list is honest about being curated: an unknown machine gets the
// narrowed set's behaviour, not a confident wrong pick.
func TestChooseDictionaryWithNoCuratedMatch(t *testing.T) { /* -> ok=false, caller falls back to NULL */ }
```

- [ ] **Steps 2–5: implement, run, mutation-check, commit.**

Mutations that must each redden a named row: drop the curated preference (English picks a thesaurus); accept bilingual as monolingual (Spanish picks OxfordSpanish); return ok for an unindexed language.

### Task 9: the `dlsym` seam and its fake

**Files:**
- Modify: `cmd/define/dict_darwin.go`, `cmd/define/dict_fake_test.go`

- [ ] **Step 1: Write the failing test** — the fake models a SET of dictionaries; a word absent from the current language reports no entry rather than answering from another.
- [ ] **Step 2: Implement the seam**

`CFSetGetValues`, **not** `CFArrayGetValueAtIndex` — verified 2026-08-28 that treating the result as a CFArray crashes with `-[__NSCFSet objectAtIndex:]: unrecognized selector`. Every symbol is `dlsym`'d; **any** missing symbol falls back to today's `NULL` behaviour, which is a Done-when row and not a nicety.

- [ ] **Step 3: Prove the fallback** — a build/run with one symbol name deliberately misspelled must still define an English word.
- [ ] **Steps 4–5: run, commit.**

### Task 10: live conformance for the private surface

**Files:**
- Create: `cmd/define/dict_conformance_test.go` additions

- [ ] **Step 1:** Assert all nine symbols resolve and that `mesa` through the Spanish choice is a Spanish entry — routed through `conformance.SkipOrFail` so an unreachable dictionary SKIPS by default and FAILS under `CONFORMANCE_STRICT` (`#25`).
- [ ] **Step 2:** Run unsandboxed in BOTH env states; sandboxed it must skip, not fail.
- [ ] **Step 3:** Commit, then `sdlc close --issue 23`.

## Risks

**The private symbols can vanish on an OS update.** The whole of M2 rests on nine undocumented symbols. Mitigated by `dlsym` + fallback to today's behaviour (so the failure mode is "M1's tool", not a crash) and by Task 10's conformance check. Recorded as a Done-when row in the issue, not as a note.

**CFSet, not CFArray.** Measured: getting this wrong is an uncaught ObjC exception, not a wrong answer. Called out here because the natural Go/C reflex is `CFArrayGetValueAtIndex` and the crash is in a cgo frame where the cause is not obvious.

**The curated list is wrong on someone else's machine.** 87 dictionaries here; another Mac has a different set. Mitigated by the explicit override and by *saying which dictionary is in use* so a wrong pick is visible rather than puzzling.

**Deck migration touches the one irreplaceable artifact.** `MigrateFlatDeck` is idempotent and non-destructive by test, and it moves rather than rewrites. If it is ever made to rewrite, the test asserting a twice-run migration must be the thing that stops it.

**`events/` deliberately NOT scoped.** Language is a deck dimension, not an event one. If a later issue wants "how much Spanish did I study", that is a join over the deck, not a split log — and splitting the log later would be a migration of the append-only artifact, which is worse.

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

- **Every symbol this plan names was checked against the tree before saving.**
  One was wrong: the deck accessor is `Deck() ([]Word, error)`, not `Words()`.
  `#21`'s rule — API names in a plan are code nothing compiles — and the third
  time today that check has caught something.
- **Every measurement was re-run on 2026-08-28**, not carried from the issue's own probe, per `#17 BR-21`. All of the issue's claims held. Two things are STRONGER than the issue states: the CFSet mistake is a crash rather than merely unordered iteration, and the six English monolingual candidates were enumerated to confirm two are thesauruses and one is an accessibility dictionary — which is what makes the curated list necessary rather than convenient.
- **`RuntimeFiles` is scope this issue grew deliberately.** `user-model.md` being unignored is a pre-existing defect found while designing; fixing only the new `lang` file would have been the instance, not the class.
- **`#27`'s `voices()` fallback is deleted, not adapted.** A mode does not guess. See `workshop/plans/000027-pronunciation-locale-plan.md`'s Revisions entry.
