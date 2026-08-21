# Vocabulary store — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A `Store` seam backed by YAML files in the working directory, so the deck survives restarts — and `#14`'s history becomes persistent without the editor changing.

**Architecture:** Pure data types (`Word`, `ReviewEvent`) plus a `Store` interface with two implementations — in-memory and YAML-on-disk — held to one shared conformance suite. A `Clock` is injected everywhere, because every consumer of this store is date-driven and `time.Now()` would make "due today" untestable.

**Tech Stack:** Go 1.26, `go.yaml.in/yaml/v3` (already used by ariadne; first use here).

**Milestone:** single-pass. One boundary, one `sdlc close` — no `Mx` tags.

---

## Scope, after two operator narrowings

Both cut real work, so they are recorded before the design rather than inside it:

- **YAML files, not a database.** That was the only thing the original "use a
  git repo / nous push" framing was asking for.
- **The working directory is the location.** No config, no brain resolution, no
  home-directory search. `NewStore(dir)` takes it as a parameter.

What survives from the original framing is the **file shape**, and only because
the operator plans to run this inside a replicated directory (below).

## Non-goals

- No config file, no `--dir` flag, no environment variable. `define` runs where
  it runs. Adding one later is a single call site by construction.
- No sync, push, commit or `git` invocation of any kind.
- No migration format or schema versioning. The first schema change, if it comes,
  is cheap while the corpus is one person's word list.
- No scheduling, review logic or statistics — `#5`, `#6`, `#8` own those. This
  issue stores and returns; it decides nothing.

---

## Chunk 1: Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Word` | `cmd/define/store/word.go` | new |
| `ReviewEvent` | `cmd/define/store/event.go` | new |
| `Slug` | `cmd/define/store/word.go` | new |
| `Clock` | `cmd/define/store/clock.go` | new |

- **Word** — one entry in the deck: `Text`, `FirstSeen`, `LastSeen`, `Lookups int`,
  `Found bool`. Everything the scheduler will need is derivable from events, so
  `Word` stays what is true about the *word*, not about the studying of it.
  - **DRY rationale:** `#5` (scheduling) and `#8` (stats) both need per-word
    state. Putting box/interval here would make this file the schedule's home and
    force `#5` to migrate it.
  - **Future extensions:** a `Source` when a second dictionary lands.

- **ReviewEvent** — one thing that happened, at a time: `Word`, `Kind`
  (`looked-up`, `reviewed`), `Correct bool`, `At time.Time`.
  - **Append-only.** Every statistic `#8` lists — words per day, streak, active
    days, accuracy — is a fold over these. Storing counters instead would create a
    second source of truth that drifts (ARCH-DRY).

- **Slug(text) string** — the on-disk name for a word. Pure and **total**: it must
  produce a safe filename for any input the dictionary accepts, including
  multi-word headwords (`hot dog`), non-ASCII (`café`), and case variants that
  must not collide-by-accident yet must not become two decks either.

- **Clock** — `Now() time.Time`. Injected, never called globally.
  - **Injected into:** every `Store` method that stamps a time. `#5`'s entire
    behaviour is "what is due today", so a wall clock makes it untestable.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Store` | `cmd/define/store/store.go` | new | interface (seam) |
| `yamlStore` | `cmd/define/store/yaml.go` | new | the filesystem |
| `memStore` | `cmd/define/store/mem.go` | new | nothing — the reference implementation |
| `storeHistory` | `cmd/define/history_store.go` | new | adapts `Store` to `#14`'s `History` |

- **Store** — `Upsert(Word) error`, `Deck() ([]Word, error)`,
  `AppendEvent(ReviewEvent) error`, `Events(since time.Time) ([]ReviewEvent, error)`.
  - **Why `memStore` is production code, not a test double:** it is the reference
    implementation the YAML one is held against, and `--play` may want an
    ephemeral deck later. It lives beside `yamlStore`, not in `_test.go`.

- **yamlStore** — `words/<slug>.yaml`, one file per word; `events/YYYY-MM-DD.yaml`,
  append-only per day.
  - **Why this shape:** the operator intends to run `define` inside a replicated
    directory. The failure mode there is a **merge conflict**, not corruption. One
    file per word and an append-only day log never conflict; a single mutable
    `vocab.yaml` conflicts on the second machine. The cost is zero now and high
    after files exist.
  - **Writes must be atomic** — write to a temp file in the same directory, then
    rename. A half-written YAML file is a corrupted deck entry, and this process
    is killed with Ctrl-C by design.
  - **Injected into:** `run()`, constructed once with the working directory.

- **storeHistory** — makes a `Store` satisfy `#14`'s `History`.
  - `Add(line, found)` → `Upsert` + `AppendEvent`. `Prefix(p)` → deck entries with
    that prefix, newest-first, deduped.
  - **This is the whole persistence deliverable for the editor.** `#14` already
    talks to `History`; nothing in the editor changes.

### Test surface

The two `Store` implementations share **one conformance suite** run against both
(`storetest.Suite`), because "the fake behaves like the real thing" is otherwise
a claim rather than a test — the exact gap `#2`'s fake-CDN work exposed. Plus:
`Slug` gets a table test and a fuzz target (it converts arbitrary text into a
filename, so it is the one function here that can produce an unsafe path).

---

## Chunk 2: Tasks

### Task 1: Types and `Slug`

**Files:** create `cmd/define/store/{word,event,clock}.go`; test `word_test.go`

- [ ] **Step 1: Write the failing tests.** `Slug` obligations: lowercases so
      `Define` and `define` are one word; `hot dog` → `hot-dog`; keeps non-ASCII
      readable rather than mangling it; **never** produces `.`, `..`, an absolute
      path, or anything containing a path separator; distinct inputs do not
      collide after normalisation (`hot dog` vs `hot-dog` must not silently merge —
      pick one rule and assert it).
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement**
- [ ] **Step 4: Run, expect PASS. Add `FuzzSlug`** asserting the result is always
      a single safe path element — this function turns user input into a filename.
- [ ] **Step 5: Commit** — `#3: word, event and slug types`

### Task 2: `Store` + `memStore` + the shared conformance suite

**Files:** create `store/{store,mem}.go`, `store/storetest/suite.go`; test `mem_test.go`

- [ ] **Step 1: Write the suite first**, as an exported function taking a
      `func() Store`. Obligations: `Upsert` then `Deck` round-trips; a second
      `Upsert` of the same word updates rather than duplicating; `Events(since)`
      filters by time and returns chronological order; an empty store returns
      empty, not nil-with-error; times survive the round trip **to the second**.
- [ ] **Step 2: Run against `memStore`, expect FAIL, then implement**
- [ ] **Step 3: Run, expect PASS**
- [ ] **Step 4: Commit** — `#3: Store seam, in-memory implementation, conformance suite`

### Task 3: `yamlStore`

**Files:** create `store/yaml.go`; test `store/yaml_test.go`

- [ ] **Step 1: Run the SAME suite against `yamlStore`** on a temp dir. Expect
      FAIL, then implement. The suite passing for both is the deliverable — it is
      what makes `memStore` a reference rather than an alibi.
- [ ] **Step 2: Add the tests only the disk implementation can fail:**
  - a store re-opened on the same directory sees what the first wrote;
  - **atomicity** — a temp file left behind by a killed write is not read as a
    word, and no partially-written file ever becomes a deck entry;
  - a `words/` file that is corrupt or unreadable is **skipped with a warning**,
    not fatal: one bad file must not make the deck unopenable;
  - two events on the same day land in one file, in order;
  - a directory that does not exist yet is created on first write.
- [ ] **Step 3: Run, expect PASS**
- [ ] **Step 4: Commit** — `#3: YAML store`

### Task 4: Wire persistence into the editor

**Files:** create `cmd/define/history_store.go`; modify `main.go`, `replraw.go`

- [ ] **Step 1: Write the failing test.** `storeHistory` satisfies `History`;
      `Add` then `Prefix` returns the word; a second process (a second `Store` on
      the same dir) sees the first's history — that is the whole point.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** `realDeps()` constructs the store on the working
      directory; the loop takes `History` and no longer constructs `memHistory`.
      **A store that fails to open must not break `define`** — warn on stderr and
      fall back to session history, exactly as audio failure degrades today.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Manual check — the thing that makes this issue worth doing**

```sh
cd $(mktemp -d) && define        # look up two words, ^C
ls words/ events/                # they are on disk
define                           # Up recalls them across the restart
```

- [ ] **Step 6: Update `README.md` and `atlas/define.md`** — a tool that writes
      files in your working directory is user-facing behaviour and must be
      documented as such.
- [ ] **Step 7: Commit, then `sdlc close --issue 3 --verified '<evidence>'`**

---

## Risks

- **Writing files into the user's cwd is surprising** if undocumented. `define`
  has never written anything before. README and atlas say so plainly, and the
  files are created only when a word is actually looked up.
- **A corrupt file making the deck unopenable** would lose the whole deck to one
  bad write. Skip-and-warn per file, tested.
- **Ctrl-C during a write** is routine here — the tool is quit that way by
  design. Atomic rename is the mitigation, and it is on the critical path, not a
  refinement.
- **`Slug` collisions** silently merge two words into one file. Table test plus
  fuzz on the safety property.
