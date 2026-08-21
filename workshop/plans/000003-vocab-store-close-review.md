# Boundary Review — tools#3 (whole-issue close)

| field | value |
|-------|-------|
| issue | 3 — vocabulary store: per-user YAML deck in a brain, behind a Store seam |
| repo | tools |
| issue file | workshop/issues/000003-vocab-store.md |
| boundary | whole-issue close |
| milestone | — |
| window | e44ac7885e15a22ce5a23bc05d404c812688296f..6b79de85b2cf229b6f84beada1bae971ef0cc876 |
| command | sdlc close --issue 3 |
| reviewer | claude |
| timestamp | 2026-08-20T21:30:16-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The store lands cleanly and the central claim of the issue holds under independent verification: `go build`, `go vet`, `gofmt -l`, `go test ./...` and `go test -race ./cmd/define/store/...` are all clean, `FuzzSlugIsSafe` survived 6.5M execs, and I confirmed by probe that the `deps.history → runEditor → disk` path really does write `words/sycophantic.yaml` + `events/<day>.yaml`, that `define <word>` and the piped path write nothing (the README's stated `#3`/`#4` boundary), and that the atlas's load-bearing "timestamps keep their offset" claim is actually true through the YAML round-trip (an event stamped 20:00 −07:00 lands in `2026-08-21.yaml` and reads back as `2026-08-20 20:00 −0700`). Degradation is real, not asserted: a read-only directory, a 300-char headword, a corrupt word file and a leftover temp file all return errors or skip rather than panic. Nothing here is Critical. What holds it back from SHIP is three cheap fixes: `storeHistory.Prefix` is a byte-identical copy of `memHistory.Prefix` (ARCH-DRY, and it splits the `History` contract's test coverage 15-to-1 in favour of the fallback); `YAML.Upsert` silently resets a word's `FirstSeen`/`Lookups` when the existing file is unreadable, while `Deck` warns on the same condition; and the project file still documents the superseded git/brain/`nous push` storage model (PQ-9, flagged at plan-quality and never addressed).

### 1. Strengths

- **`storetest.Suite` is the real deliverable and it is done right** (`cmd/define/store/storetest/suite.go`). One exported suite, run against `Mem` (`mem_test.go:11`) and `YAML` (`yaml_test.go:15`), with `merge` (`mem.go:34`) and `sortDeck` (`mem.go:58`) shared so the two implementations *cannot* disagree about upsert semantics or deck ordering. This is a textbook ARCH-MOCK pass: production flow and test flow share the `Store` boundary, and the fake is production code rather than a `_test.go` alibi.
- **`Slug` is genuinely total and safe** (`word.go:48`). I checked the adversarial cases by hand — `../etc/passwd` → `etc-passwd-<hash>`, `""` → `w-<hash>`, `.hidden` → `hidden-<hash>`, `with\x00null` → `with-null-<hash>` — and branch-A output is provably injective (reversibility forces the key to contain no `-`). The fuzz target is aimed at the right property.
- **`Add` appends an event always but upserts a word only when `found`** (`history_store.go:52-73`), with `Prefix` reading the event log rather than the deck. This is the "one record, two readers" split `#14` settled, and `TestStoreHistoryRecallsTyposButDoesNotDeckThem` pins both halves in one test.
- **The UTC-filename / offset-preserving-timestamp trap is documented at the exact line that causes it** (`yaml.go:89-93`), with the consequence for `#8` spelled out. I verified the claim empirically rather than taking it on trust; it holds.
- **The honesty correction landed in three places** — `yaml.go:20-24`, `atlas/define.md`, and `workshop/lessons.md:111` — turning an overstated conflict rationale into a rate claim. Fixing the lesson, not just the sentence, is the right move.

### 2. Critical findings

None.

### 3. Important findings

**I-1 — `storeHistory.Prefix` is a verbatim copy of `memHistory.Prefix` (ARCH-DRY).**
`cmd/define/history_store.go:76` vs `cmd/define/history.go:34`. The bodies are byte-identical apart from the mutex: same reverse loop, same `seen` map, same `HasPrefix` guard, same newest-first append. The `History` contract ("newest first, deduped") now has two definitions that can drift. Worse, the coverage is lopsided — `editor_test.go`'s `hist()` helper (line 28) builds `&memHistory{}`, so ~15 recall/dedup/prefix tests exercise the *fallback* copy, while the production copy has exactly one (`TestStoreHistoryPrefixIsNewestFirstAndDeduped`). A fix or refinement to one would silently not apply to the other.
*Fix sketch:* extract into `history.go` and call it from both —
```go
// prefixMatch is the one definition of History.Prefix's contract.
func prefixMatch(lines []string, p string) []string { /* the existing body */ }
```
then `memHistory.Prefix` → `return prefixMatch(h.lines, p)`, and `storeHistory.Prefix` → lock, `return prefixMatch(h.lines, p)`.

**I-2 — `YAML.Upsert` silently discards an unreadable existing entry; `Deck` warns on the identical condition.**
`cmd/define/store/yaml.go:44-49`. `readWord` returns an error for *any* failure — corrupt YAML, EACCES, a partially-materialised placeholder from a sync client — and the branch resets `old = Word{}` with no `y.warnf`. `merge(Word{}, w)` then writes back `FirstSeen` = the new sighting and `Lookups` = 1, so the word's entire history is destroyed with no user-visible signal. Meanwhile `Deck` (yaml.go:70-78) warns on exactly the same class of failure, and the issue's Done-when commits to "One corrupt or unreadable word file is **skipped with a warning**". This is inconsistent error handling across the diff, and it is sharper than it looks given the README actively markets running inside a synced directory — a transiently-unreadable file is a realistic input there, not just a corrupt one.
*Fix sketch:* one line inside the branch —
```go
if err != nil && !os.IsNotExist(err) {
    y.warnf("overwriting unreadable %s: %v", filepath.Base(path), err)
    old = Word{}
}
```

**I-3 — the project file still documents the storage model this issue replaced (ARCH-PURPOSE shadow-sweep; PQ-9, still open).**
`workshop/projects/define-learn.md:54-56` reads "**Storage shape is chosen for git.** … a brain syncs across machines … `nous push` supplies sync, so no sync code is written", and line 40 lists `#3` as "Store seam, **YAML in a brain**, clock injected". Both contradict the shipped design (cwd-only, no git of any kind, no brain resolution) and contradict the plan's own Non-goals. This is a hand-maintained restatement of a model the code no longer derives from — the exact class the shadow-sweep is for — and AGENTS.md §8 makes the project file a per-milestone obligation. The close gate ticks checkboxes; it will not rewrite this prose. It was raised at plan-quality as PQ-9 and disposed `not-addressed` in round 2, so it arrives at this boundary unfixed.
*Fix sketch:* replace the "chosen for git" bullet with the rate-not-impossibility claim already written in `yaml.go:20-24`, and change line 40 to "Store seam, YAML in the working directory, clock injected".

**I-4 — no test covers the wiring that is the issue's entire purpose.**
No test anywhere sets `deps.history` (`grep -rn "history:" cmd/define/*_test.go` returns nothing), so `runEditor`'s `hist := d.history` (`replraw.go:61`) is only ever exercised through the `nil` fallback to `memHistory`. Deleting the `history` field from `deps`, or reverting `replraw.go:61-64` to `var hist History = &memHistory{}`, leaves the whole suite green — persistence would silently stop working with zero test signal, and the only evidence it works today is a manual check in the `## Log`. Related and equally cheap: nothing pins that `newStoreHistory` (`history_store.go:37-48`) restores *not-found* lines across a restart — adding `&& e.Found` to that loop breaks Up-arrow recall of typos after a restart and every existing test still passes.
*Fix sketch:* this passes today and closes the first half (I ran it) —
```go
func TestEditorPersistsThroughDeps(t *testing.T) {
	dir := t.TempDir()
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	rig.deps.history = newStoreHistory(store.NewYAML(dir, nil), fixedClock(1), nil)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\r"), rig.deps, opt, cooked, finish, &out, &errb)
	if words, _ := os.ReadDir(filepath.Join(dir, "words")); len(words) != 1 {
		t.Errorf("the editor did not persist through deps.history: %v", words)
	}
}
```
For the second half, extend `TestStoreHistoryPersistsAcrossSessions` with a `first.Add("sykophantic", false)` and assert it comes back from `second.Prefix("sy")`.

### 4. Minor findings

- `cmd/define/main.go:3-16` — the import block has a stray blank line after `"context"` and puts `github.com/xianxu/tools/cmd/define/store` inside the stdlib group. `gofmt` accepts it (it is alphabetical within the group), but it reads as an accident and differs from every other file in the package.
- `cmd/define/store/mem.go:83` and `yaml.go:140` — the event sort is duplicated verbatim. `sortDeck` was extracted for exactly this reason; a matching `sortEvents(out)` would finish the job (ARCH-DRY, low stakes).
- `cmd/define/history_store.go:94` and `store/yaml.go:144` — two near-identical `warnf` helpers, both prefixing `"define: "`. Different semantics (once vs. always) justify two functions, but not two copies of the prefix literal.
- `history_store.go:98-100` — `warnf` reads and writes `h.warned` outside `h.mu`, which `Add`/`Prefix` otherwise hold. Single-goroutine in practice, but the type is otherwise mutex-guarded, so the inconsistency invites a future race.
- `history_store.go:37-41` — a failed `Events()` read at construction consumes the one-warning budget, so every later *write* failure in that session is silent. Two different failure classes sharing one budget.
- `history_store.go:71` — the `"(history is session-only)"` suffix is appended to every warning, including `"could not save word"`, where the event may have been saved fine.
- `cmd/define/store/word.go:48` — `Slug` does not bound length. A 300-char headword produces a 300-char filename and `Upsert` fails with `ENAMETOOLONG` (verified). It degrades correctly — error returned, warn-once, session continues — but truncate-plus-hash past ~200 bytes would make it work instead of merely not crash.
- `cmd/define/store/word.go:29` — `Key` does no Unicode normalisation, so NFC `café` and NFD `café` are two different words with two different files. Not reachable through the editor today (`parseREPLLine` does not normalise either), but it will matter once `#4` captures from other entry points.
- `cmd/define/store/mem.go:43` — `out.Lookups = old.Lookups + max(1, w.Lookups)` means `Upsert` always increments, so it cannot be used to *correct* a count and re-upserting an identical `Word` double-counts. Fine for the current callers; worth a doc line on the interface since `#4`/`#8` will consume it.
- `cmd/define/store/yaml_test.go:99-112` — `TestYAMLDifferentWordsTouchDisjointFiles` asserts only that the file *count* grew by one. The property it names ("disjoint files, so a sync has nothing to merge") would be pinned by capturing `alpha.yaml`'s content or mtime before and asserting it is unchanged after.
- `cmd/define/history_store_test.go:120-127` — `errFail`/`failErr` hand-roll what `errors.New("store unavailable")` already provides.
- `cmd/define/store/clock.go` — `SystemClock`/`FixedClock` have no direct test. Trivial enough that this is a note, not a gap.
- `cmd/define/store/event.go:20-21` — `Found` has no `omitempty` while `Correct` does, so day files always carry `found: false` rows for reviews. Harmless, but inconsistent within one struct.

### 5. Test coverage notes

Coverage is strong where the risk was named up front and thin exactly at the seam the issue exists to create.

Well covered: the conformance suite runs both implementations against one contract; the disk-only hazards each have a dedicated test (reopen, leftover temp file, corrupt file + warning, day grouping, disjoint files); `Slug`'s safety property has both a table test and a fuzz target that I ran to 6.5M executions clean; `storeHistory`'s three behavioural commitments (typos recalled but not decked, `Prefix` never touching the store, warn-once degradation) are each pinned by a test that would actually fail if the behaviour regressed — `TestStoreHistoryPrefixDoesNotQueryTheStore` in particular is testing the real invariant rather than restating the implementation.

Gaps, in priority order:
1. **The `deps.history → runEditor` wiring is untested** (I-4). The issue's purpose has no automated proof.
2. **`newStoreHistory`'s restore has no not-found case** (I-4). Adding `&& e.Found` to the restore loop passes the whole suite.
3. **Timestamp offset preservation is untested** despite `atlas/define.md` and `yaml.go:89-93` making it a load-bearing contract for `#8`. Every timestamp in `storetest.Suite` is constructed with `time.UTC`, so the round-trip that actually matters — a non-UTC offset surviving marshal/unmarshal — has no assertion. It works today (I verified: `2026-08-20T20:00:00-07:00` round-trips with its offset intact and lands in `2026-08-21.yaml`), but a yaml-library bump or a struct-tag change would break `#8`'s local-day grouping silently. Add one case to the suite using `time.FixedZone` and assert `got.Zone()` offset equals what went in.
4. **`YAML.Upsert`'s corrupt-file branch has no test** — the branch at yaml.go:44-49 is dead code as far as the suite is concerned, which is how I-2's missing warning survived.
5. `Mem.Deck()` returns a non-nil empty slice while `YAML.Deck()` returns `nil` for a missing directory. Both satisfy `len() == 0` so the suite passes, and it is harmless in Go — but if the suite intends to pin nil-ness either way, it should say so.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — flag.** I-1 (`Prefix` duplicated verbatim) is the substantive one; the duplicated event sort and the two `warnf` copies are minor echoes of the same habit. Everywhere the diff *did* extract — `merge`, `sortDeck`, `storetest.Suite`, `Key`/`Slug` — it is exemplary, which makes the three misses look like fatigue at the end rather than a design position.
- **ARCH-PURE — pass.** The pure core (`Word`, `ReviewEvent`, `Key`, `Slug`, `merge`, `sortDeck`, `Clock`) is genuinely pure and its tests run with no IO; `word_test.go` needs no filesystem, no mocks, no fakes. `YAML` is the thin shell, and the one IO decision that could have leaked into business logic — *where* the directory is — is a constructor parameter resolved at `main.go:50` in a 7-line function. `openHistory` is the entire IO seam for this feature. This is the right shape.
- **ARCH-PURPOSE — flag, narrow.** The shadow-sweep passes on code: `storeHistory` derives from `Store`, `runEditor` takes the seam, and the deferred consumers (one-shot, piped) are a genuinely separable extension with an explicit `#3`/`#4` split table rather than the deferred point of the issue. It fails on one artifact — I-3, the project file still restating the git/brain model.
- **ARCH-MOCK — pass, and the strongest part of the diff.** The filesystem dependency has a stateful fake (`Mem`) behind the same seam, both sides run the same conformance suite in the ordinary `go test`, and the owned component boots from portable non-production storage by construction (`NewYAML(t.TempDir(), nil)`). No scheduled live-conformance check is needed here because the "real" side already runs against a real filesystem in CI.
- **For `#5`/`#8`:** two constraints this diff creates are worth carrying forward. First, `Word` deliberately has no box/interval — keep the schedule out of `store/word.go` or `#5` inherits a migration. Second, `Events(time.Time{})` currently loads and parses *every* day file ever written, on *every* `define` invocation including one-shot lookups that never touch history (`realDeps()` at main.go:40 constructs it eagerly). That is ~0ms today and a growing tax later; the `since` parameter is already the fix for recall (load the last N days), and lazy construction would keep the one-shot path free. Related: a corrupt events file makes `define <word>` print store warnings on a path that never uses the store, since `YAML.warnf` has no once-guard.

### 7. Plan revision recommendations

Add a `## Revisions` section to `workshop/plans/000003-vocab-store-plan.md` recording these four deltas, so the plan stops describing something the code does not:

1. **Type names.** The Integration-points table names `yamlStore` and `memStore`; the shipped types are exported as `YAML` and `Mem` (`store/yaml.go:25`, `store/mem.go:13`). Paths and roles match the table — only the identifiers changed, because they are consumed from outside the package as `store.NewYAML` / `store.NewMem`.
2. **`Word` has no `Found` field.** The Pure-entities prose lists `Word` as `Text, FirstSeen, LastSeen, Lookups int, Found bool`; the shipped struct (`store/word.go:20-25`) has no `Found`. This is correct — `found` is a property of a *lookup event*, not of a word, and only found words are decked at all — but the plan should say so rather than describe a field that does not exist.
3. **`Key` collapses interior whitespace.** Slug rule 1 specifies `strings.ToLower(strings.TrimSpace(text))`; the implementation is `strings.ToLower(strings.Join(strings.Fields(text), " "))` (`store/word.go:30`), so `hot   dog` normalises to `hot dog` rather than to a `hot---dog` slug. The implementation is the better rule and matches `parseREPLLine`'s existing collapsing — record the change, keep the code.
4. **Constructor signature.** Spec and plan both say `NewStore(dir)`; the shipped constructor is `NewYAML(dir string, warn io.Writer)` (`store/yaml.go:32`). The `warn` seam is what makes "skip a corrupt file with a warning" testable without capturing stderr, so it earns its place — but it is an undeclared addition to the stated surface, and downstream issues will consume it.

Separately, `workshop/plans/000003-vocab-store-plan-gate.md` still carries **PQ-9** as an open `not-addressed` finding, which I-3 confirms is genuinely still open in the tree. It should be disposed `addressed` only once `define-learn.md:54-56` and `:40` are corrected — not waived at the close.

```findings
findings:
  - id: new
    severity: Important
    title: |
      storeHistory.Prefix duplicates memHistory.Prefix verbatim (ARCH-DRY)
    detail: |
      cmd/define/history_store.go:76 and cmd/define/history.go:34 have
      byte-identical bodies apart from the mutex, so the History contract
      "newest first, deduped" now has two definitions that can drift. Coverage
      is lopsided too: editor_test.go's hist() helper builds memHistory, so
      ~15 recall tests exercise the fallback copy while the production copy has
      one. Extract prefixMatch(lines, p) into history.go and call it from both.
  - id: new
    severity: Important
    title: |
      YAML.Upsert silently resets a word's history when the existing file is unreadable
    detail: |
      cmd/define/store/yaml.go:44-49 sets old = Word{} for any non-NotExist read
      error with no y.warnf, so merge writes back FirstSeen = now and Lookups = 1
      and the word's history is destroyed silently. Deck (yaml.go:70-78) warns on
      the identical condition, and the issue's Done-when promises "skipped with a
      warning". Sharper than it looks because the README markets running inside a
      synced directory, where a transiently-unreadable placeholder file is a
      realistic input. One-line fix: y.warnf before the reset.
  - id: new
    severity: Important
    title: |
      Project file still documents the git/brain/nous-push storage model (PQ-9)
    detail: |
      workshop/projects/define-learn.md:54-56 still says "Storage shape is chosen
      for git ... a brain syncs across machines ... nous push supplies sync", and
      line 40 lists issue 3 as "YAML in a brain". Both contradict the shipped
      cwd-only design and the plan's own Non-goals. This is the hand-maintained
      restatement ARCH-PURPOSE's shadow-sweep names; it was raised at
      plan-quality as PQ-9 and disposed not-addressed in round 2. The close gate
      ticks checkboxes but will not rewrite this prose.
  - id: new
    severity: Important
    title: |
      No test covers the deps.history to runEditor wiring — the issue's purpose
    detail: |
      No test anywhere sets deps.history, so replraw.go:61's hist := d.history is
      only exercised through the nil fallback to memHistory. Reverting that line
      to "var hist History = &memHistory{}" leaves the whole suite green while
      persistence silently stops working; the only evidence it works today is a
      manual check in the Log. Related and equally cheap: nothing pins that
      newStoreHistory (history_store.go:37-48) restores not-found lines, so
      adding "&& e.Found" to the restore loop breaks typo recall across a restart
      with zero test signal.
  - id: new
    severity: Minor
    title: |
      Timestamp offset preservation is a load-bearing contract with no test
    detail: |
      atlas/define.md and yaml.go:89-93 commit to "timestamps keep their offset"
      so issue 8 can recover a local-day view from UTC-named files. Every
      timestamp in storetest.Suite is time.UTC, so the round-trip that matters is
      unasserted. It holds today (verified: 2026-08-20T20:00:00-07:00 round-trips
      intact and lands in 2026-08-21.yaml), but a yaml-library bump would break
      issue 8 silently. Add one time.FixedZone case asserting the Zone offset.
  - id: new
    severity: Minor
    title: |
      main.go import block has a stray blank line and a third-party import in the stdlib group
    detail: |
      cmd/define/main.go:3-16. gofmt accepts it because it is alphabetical within
      the group, but it differs from every other file in the package.
  - id: new
    severity: Minor
    title: |
      Event sort and warnf helper are each duplicated across the two stores
    detail: |
      store/mem.go:83 and store/yaml.go:140 repeat the event sort verbatim;
      sortDeck was extracted for exactly this reason, so a matching sortEvents
      finishes the job. history_store.go:94 and store/yaml.go:144 are two
      near-identical warnf helpers repeating the "define: " prefix literal.
  - id: new
    severity: Minor
    title: |
      storeHistory.warned is read and written outside the mutex
    detail: |
      history_store.go:98-100 touches h.warned without h.mu, which Add and Prefix
      otherwise hold. Single-goroutine in practice, but the inconsistency in an
      otherwise mutex-guarded type invites a future race.
  - id: new
    severity: Minor
    title: |
      A construction-time read failure consumes the one-warning budget for the whole session
    detail: |
      history_store.go:37-41 warns via the same warned flag that later write
      failures use, so an unreadable events directory at startup silences every
      subsequent write failure. Two different failure classes share one budget.
      Separately, the "(history is session-only)" suffix at line 71 is appended
      even to "could not save word", where the event may have saved fine.
  - id: new
    severity: Minor
    title: |
      Slug does not bound filename length, so a very long headword fails to save
    detail: |
      store/word.go:48. A 300-char word produces a 300-char filename and Upsert
      fails with ENAMETOOLONG (verified). It degrades correctly — error returned,
      warn-once, session continues — but truncate-plus-hash past ~200 bytes would
      make it work rather than merely not crash.
  - id: new
    severity: Minor
    title: |
      Key does no Unicode normalisation, so NFC and NFD spellings are two words
    detail: |
      store/word.go:29. Not reachable through the editor today since
      parseREPLLine does not normalise either, but it will matter once issue 4
      captures from other entry points.
  - id: new
    severity: Minor
    title: |
      merge always increments Lookups, so Upsert cannot correct a count
    detail: |
      store/mem.go:43 computes old.Lookups + max(1, w.Lookups), so re-upserting an
      identical Word double-counts and a caller can never set an exact value.
      Fine for the current callers; worth a doc line on the Store interface since
      issues 4 and 8 will consume it.
  - id: new
    severity: Minor
    title: |
      TestYAMLDifferentWordsTouchDisjointFiles asserts file count, not disjointness
    detail: |
      store/yaml_test.go:99-112 checks only that the file count grew by one. The
      property it names would be pinned by capturing alpha.yaml's content or
      mtime before the second write and asserting it is unchanged after.
  - id: new
    severity: Minor
    title: |
      YAML.Upsert's corrupt-file branch has no test
    detail: |
      store/yaml.go:44-49 is dead code as far as the suite is concerned, which is
      how the missing warning survived to this boundary.
  - id: new
    severity: Minor
    title: |
      errFail hand-rolls what errors.New already provides
    detail: |
      cmd/define/history_store_test.go:120-127 defines a failErr type and a
      package-level errFail instead of errors.New("store unavailable").
  - id: new
    severity: Minor
    title: |
      Found lacks omitempty while Correct has it, within one struct
    detail: |
      store/event.go:20-21. Day files always carry "found: false" rows for review
      events. Harmless, but inconsistent.
  - id: new
    severity: Minor
    title: |
      History is constructed eagerly and loads every day file on every invocation
    detail: |
      realDeps() at main.go:40 builds the store even for one-shot "define <word>"
      and piped input, neither of which uses history, and newStoreHistory calls
      Events(time.Time{}) which parses every day file ever written. ~0ms today, a
      growing tax later; the since parameter is already the fix for recall. A
      corrupt events file also makes "define <word>" print store warnings on a
      path that never uses the store, since YAML.warnf has no once-guard.
  - id: new
    severity: Minor
    title: |
      Plan needs a Revisions entry — four documented deltas the code does not match
    detail: |
      (1) the table names yamlStore/memStore but the shipped types are YAML/Mem;
      (2) Word is specified with a Found bool that does not exist (correctly — found
      is a property of an event); (3) Key collapses interior whitespace via
      strings.Fields rather than TrimSpace as slug rule 1 states; (4) the
      constructor is NewYAML(dir, warn io.Writer), not NewStore(dir). Also, the
      gate ledger still carries PQ-9 as open, which finding I-3 confirms is
      genuinely still open in the tree.
```

---

## Re-review — 2026-08-20T21:40:22-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 3 — vocabulary store: per-user YAML deck in a brain, behind a Store seam |
| repo | tools |
| issue file | workshop/issues/000003-vocab-store.md |
| boundary | whole-issue close |
| milestone | — |
| window | e44ac7885e15a22ce5a23bc05d404c812688296f..8bd988a03bfd696cce4a034210307070ca17be55 |
| command | sdlc close --issue 3 |
| reviewer | claude |
| timestamp | 2026-08-20T21:40:22-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 2's four Important findings are genuinely fixed, and I verified each rather than trusting the commit message: `prefixMatch` is now the single definition called by both `History` implementations; `YAML.Upsert`'s reset branch now warns (probe output: `define: overwriting unreadable alpha.yaml: yaml: found character that cannot start any token`); the project file no longer restates the git/brain/`nous push` model; and both new tests are real — reverting `replraw.go:61-64` to `var hist History = &memHistory{}` fails `TestEditorPersistsThroughDeps`, and adding `&& e.Found` to the restore loop fails `TestStoreHistoryRestoresTyposAcrossSessions`. Independent verification is clean throughout: `go build`, `go vet`, `gofmt -l`, `go test ./...`, `go test -race ./cmd/define/store/...`, and `FuzzSlugIsSafe` to 8.2M execs. Done-when 3 checks out by grep — the only `time.Now()` in production code is inside `systemClock`. Nothing here is Critical. One new Important holds it back from SHIP: the atlas claims "**Writes are atomic** (temp file in the same directory, then rename)" as a blanket property, but only *word* writes are atomic — the event log is a raw `O_APPEND` write, and I confirmed by probe that one truncated record makes `Events` skip the entire day file, silently erasing that day from Up-arrow recall. That's the exact class of overstated claim this issue's own `workshop/lessons.md` entry was written about. The rest is a Minor tail, most of it carried unfixed from round 2 by design.

### 1. Strengths

- **`storetest.Suite` is the deliverable and it holds up** (`cmd/define/store/storetest/suite.go`), run against `Mem` (`mem_test.go:11`) and `YAML` (`yaml_test.go:15`), with `merge` (`mem.go:34`) and `sortDeck` (`mem.go:58`) shared so the two cannot disagree about upsert semantics or ordering. Textbook ARCH-MOCK: production and test flow share the `Store` boundary and the fake is package code, not a `_test.go` alibi.
- **The round-2 fixes were done at the right altitude.** `prefixMatch` (`history.go:44`) isn't just a de-dup — the comment records *why* the duplication was dangerous (lopsided coverage), which is the thing that would otherwise be re-learned. Same for the `Upsert` warning comment at `yaml.go:44-49`.
- **The two new tests are mutation-proof, and the commit says so honestly** ("I verified each FAILS on the mutation it exists to catch"). I re-ran both mutations independently; the claim is accurate. `TestStoreHistoryPrefixDoesNotQueryTheStore` (`history_store_test.go:82`) is likewise testing an invariant rather than restating the implementation.
- **`Slug` is total and safe under real fuzzing** (`word.go:48`). Branch-A output is provably injective — reversibility forces the key to contain no `-` — and 8.2M execs found nothing.
- **Degradation is real, not asserted.** A read-only directory returns `mkdir …: permission denied` from both `Upsert` and `AppendEvent` and the session continues; a 300-char headword, a corrupt word file and a leftover temp file all skip or error rather than panic.

### 2. Critical findings

None.

### 3. Important findings

**The event log has no torn-record recovery, and the atlas's atomicity claim doesn't scope itself to words.**
`cmd/define/store/yaml.go:96-110` (`AppendEvent`) writes with `os.OpenFile(…O_APPEND…)` + `f.Write(b)` — no temp-file-then-rename, no per-record recovery on read. `Events` (`yaml.go:126-131`) unmarshals the *whole* day file and skips it entirely on any parse error. Probe: after appending 35 bytes of a truncated record to a day file holding one good event, `s.Events(time.Time{})` returns **0 events** with `define: skipping 2026-08-21.yaml: yaml: line 7: could not find expected ':'`. Because `storeHistory` restores recall from events and nothing else, one partial record permanently erases that day's Up-arrow history — a strictly larger blast radius than the word path, where one bad file costs one word. Meanwhile `atlas/define.md` states "**Writes are atomic** (temp file in the same directory, then rename) because this process is quit with Ctrl-C by design" without saying that half the writes aren't. Ctrl-C genuinely can't tear a completed `write()`, so this isn't a crash-path panic — but the README actively markets running inside a synced directory, which is the same argument used to justify the round-2 `Upsert` warning, and a partially-materialised day file is exactly that input.
*Fix sketch, cheap version:* scope the atlas sentence — "word writes are atomic; the day log is append-only, and an interrupted or partially-synced append costs that day's log." *Durable version (~10 lines):* in `Events`, on unmarshal failure, split the file on lines beginning with `- ` and unmarshal each record independently, keeping the parseable ones and warning about the remainder — the same skip-one-not-all principle already applied to `Deck`.

### 4. Minor findings

- `cmd/define/store/yaml.go:161-193` vs `:96-110` — `words/*.yaml` lands at **0600** (inherited from `os.CreateTemp`, preserved by the rename) while `events/*.yaml` is **0644**. Verified by probe. Two files written by one store with two permission stories.
- `cmd/define/store/store.go:10` — the `Store` interface states no thread-safety contract, and the implementations differ: `Mem` guards every method with a mutex, `YAML` has none (`Upsert` is a non-atomic read-modify-write). The conformance suite cannot catch this because the fake is *stronger* than the real one — the precise fake-diverges-from-real gap the suite exists to close (ARCH-MOCK). Same shape, lower stakes: `Mem.Deck()` returns a non-nil empty slice where `YAML.Deck()` returns `nil`. Both harmless today (single-goroutine CLI); a one-line interface doc comment settles the intent before `#4`/`#5` consume it.
- `cmd/define/replraw.go:69` — stale comment: "Querying twice doubled the work the History seam will do **once `#3` backs it with a store**." `#3` now does, three lines above.
- `workshop/issues/000003-vocab-store.md:115-134` — `## Log` has no entry for the round-2 close review or the four Important fixes it produced. AGENTS.md §3 makes logging the boundary-review outcome part of crossing the boundary.

### 5. Test coverage notes

The two gaps that mattered at round 2 are closed and I confirmed both by mutation rather than by reading. Remaining, in priority order:

1. **The event-log torn-record path has no test** — the Important above. `TestYAMLSkipsCorruptFileWithWarning` covers a corrupt *word* file only; nothing writes a partial record into a day file.
2. **`YAML.Upsert`'s newly-warning branch still has no test** (BR-18). The fix is correct — I verified it by probe — but the branch that just changed is still invisible to the suite, which is how the missing warning survived to round 2 in the first place. This is the cheapest remaining test in the diff: write `\t: [unclosed` to `words/alpha.yaml`, `Upsert`, assert the warn buffer is non-empty.
3. **Timestamp offset preservation is still unasserted** (BR-9) despite being load-bearing for `#8`.
4. `cmd/define/store/yaml_test.go` uses wall-clock `time.Now()` in four tests while the package ships a `FixedClock` for exactly this. Harmless — nothing asserts on the value — but it's the one place the diff doesn't take its own clock-injection medicine.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass, with a residue.** The round-2 fix was the substantive one and it landed correctly. What remains is cosmetic and still open: the event sort duplicated verbatim at `mem.go:83` and `yaml.go:144` (BR-11), and two `warnf` helpers each hard-coding the `"define: "` prefix. `sortDeck` was extracted for exactly this reason; `sortEvents` would finish the thought.
- **ARCH-PURE — pass.** `Word`, `ReviewEvent`, `Key`, `Slug`, `merge`, `sortDeck` and `Clock` are genuinely pure, and `word_test.go` runs with no filesystem, no mocks, no fakes. `YAML` is the thin shell; the one IO decision that could have leaked into logic — *where* the directory is — is a constructor parameter resolved in the 8-line `openHistory` at `main.go:47-54`. That is the whole IO seam for this feature.
- **ARCH-PURPOSE — pass on code, one artifact still open.** Shadow-sweep: `storeHistory` derives from `Store`; `runEditor` takes the seam and was mutation-proven to depend on it; the deferred consumers (one-shot, piped) are a genuinely separable extension with an explicit `#3`/`#4` split table. Hand-maintained restatements: README ✓, atlas ✓, project file ✓ (round-2 fix). Only the **plan** still restates a model the code doesn't derive from — BR-22, below.
- **ARCH-MOCK — pass, and still the strongest part of the diff.** Stateful fake behind the same seam, both sides run the same suite in ordinary `go test`, and the owned component boots from portable non-production storage by construction (`NewYAML(t.TempDir(), nil)`). No live-conformance check is needed because the "real" side already runs against a real filesystem. The one caveat is the Minor above: properties where `Mem` is *stronger* than `YAML` are invisible to the suite by construction, so they need to be stated on the interface instead.
- **For `#5`/`#8`:** three constraints this diff creates. `Word` deliberately carries no box/interval — keep the schedule out of `store/word.go` or `#5` inherits a migration. `Events(time.Time{})` parses every day file ever written on *every* `define` invocation including one-shot lookups that never touch history (BR-21); the `since` parameter is already the fix. And `#8` must group by timestamp, never by filename — currently guaranteed only by a comment.

### 7. Plan revision recommendations

`workshop/plans/000003-vocab-store-plan.md` still has **no `## Revisions` section** (BR-22, re-confirmed by grep). All four deltas are still live and I re-verified each against the tree:

1. **Type names** — the Integration-points table names `yamlStore`/`memStore`; the shipped exported types are `YAML` (`store/yaml.go:25`) and `Mem` (`store/mem.go:13`), because they are consumed from outside the package as `store.NewYAML`/`store.NewMem`.
2. **`Word` has no `Found` field** — the Pure-entities prose lists `Found bool`; `store/word.go:20-25` has none. The code is right (`found` is a property of a lookup event) — and note the plan's `ReviewEvent` bullet correspondingly *omits* the `Found` field the shipped struct does carry (`event.go:20`). Both halves of that swap need recording.
3. **`Key` collapses interior whitespace** — slug rule 1 specifies `strings.ToLower(strings.TrimSpace(text))`; `store/word.go:30` is `strings.ToLower(strings.Join(strings.Fields(text), " "))`, so `hot   dog` normalises to `hot dog`. The implementation is the better rule and matches `parseREPLLine`; record the change, keep the code.
4. **Constructor signature** — plan and Spec both say `NewStore(dir)`; the shipped constructor is `NewYAML(dir string, warn io.Writer)` (`store/yaml.go:32`). The `warn` seam earns its place, but it's an undeclared addition that downstream issues will consume.

Separately, `workshop/plans/000003-vocab-store-plan-gate.md` still lists **PQ-9** under "Open findings" — now stale in the *other* direction, since `define-learn.md` was corrected in `8bd988a`. It should be disposed `addressed`.

```findings
dispose:
  - id: BR-1
    disposition: withdrawn
    note: |
      Overtaken by the shipped code — deps.history plus a nil default at replraw.go:62-64; no rig panics and the full suite is green.
  - id: BR-2
    disposition: withdrawn
    note: |
      A plan-authoring style nit about pre-images of code that now exists; no value left at a code boundary.
  - id: BR-3
    disposition: addressed
    note: |
      go.mod and go.sum pin go.yaml.in/yaml/v3 v3.0.5; go build and go test run offline.
  - id: BR-4
    disposition: addressed
    note: |
      Same fix as BR-7 — define-learn.md now records the cwd-only model.
  - id: BR-5
    disposition: addressed
    note: |
      prefixMatch at history.go:44 is now the single definition, called by memHistory.Prefix and storeHistory.Prefix.
  - id: BR-6
    disposition: addressed
    note: |
      Verified by probe — the branch now emits "define: overwriting unreadable alpha.yaml: ..." before the reset.
  - id: BR-7
    disposition: addressed
    note: |
      define-learn.md:37 and :54-60 rewritten to the cwd-only model plus the rate-not-impossibility claim.
  - id: BR-8
    disposition: addressed
    note: |
      Both mutations re-verified independently — reverting replraw.go:61-64 and adding "&& e.Found" each fail exactly one new test.
  - id: BR-9
    disposition: not-addressed
    note: |
      No FixedZone case anywhere in the tree; every suite timestamp is still time.UTC.
  - id: BR-10
    disposition: not-addressed
    note: |
      main.go:3-16 still has the stray blank line after "context" and the store import inside the stdlib group.
  - id: BR-11
    disposition: not-addressed
    note: |
      mem.go:83 and yaml.go:144 still carry the identical event sort; two warnf helpers still repeat the prefix literal.
  - id: BR-12
    disposition: not-addressed
    note: |
      history_store.go:84-90 still reads and writes h.warned outside h.mu.
  - id: BR-13
    disposition: not-addressed
    note: |
      One warned flag still covers both the construction-time read failure and every later write failure.
  - id: BR-14
    disposition: not-addressed
    note: |
      No length bound in Slug; word.go still has no truncate-plus-hash path.
  - id: BR-15
    disposition: not-addressed
    note: |
      Key still does no Unicode normalisation.
  - id: BR-16
    disposition: not-addressed
    note: |
      store.go:11-12 documents "Lookups accumulates" but not that Upsert can never set an exact count.
  - id: BR-17
    disposition: not-addressed
    note: |
      yaml_test.go:99-112 still asserts only the file count, not that alpha.yaml was untouched.
  - id: BR-18
    disposition: not-addressed
    note: |
      Still no test for the Upsert corrupt-file branch — the very branch round 2 changed remains invisible to the suite.
  - id: BR-19
    disposition: not-addressed
    note: |
      history_store_test.go:124-128 still hand-rolls failErr instead of errors.New.
  - id: BR-20
    disposition: not-addressed
    note: |
      event.go:20 Found still lacks omitempty while Correct at :21 has it.
  - id: BR-21
    disposition: not-addressed
    note: |
      openHistory is still eager and Events(time.Time{}) still parses every day file on every invocation.
  - id: BR-22
    disposition: not-addressed
    note: |
      No "## Revisions" section exists in the plan (grep confirms); all four deltas re-verified live. The PQ-9 note in this finding is now stale in the other direction — the plan-gate ledger should dispose PQ-9 as addressed.
findings:
  - id: new
    severity: Important
    title: |
      The event log has no torn-record recovery, and the atlas claims atomic writes without scoping it to words
    detail: |
      AppendEvent (yaml.go:96-110) is a raw O_APPEND write with no temp-file-then-rename,
      and Events (yaml.go:126-131) unmarshals the whole day file and skips it entirely on
      any parse error. Verified by probe: appending 35 bytes of a truncated record to a day
      file holding one good event makes Events return 0 events with a "skipping" warning, so
      that day vanishes from Up-arrow recall permanently — a larger blast radius than the
      word path, where one bad file costs one word. Meanwhile atlas/define.md states
      "Writes are atomic (temp file in the same directory, then rename)" as a blanket
      property. Cheap fix: scope the atlas sentence to word writes. Durable fix, about ten
      lines: on unmarshal failure, split the file on lines starting with "- " and unmarshal
      each record independently, applying the skip-one-not-all rule Deck already follows.
  - id: new
    severity: Minor
    title: |
      words/*.yaml is written 0600 while events/*.yaml is 0644
    detail: |
      yaml.go:161-193 inherits 0600 from os.CreateTemp and the rename preserves it, while
      AppendEvent at yaml.go:106 opens with 0644. Verified by probe. Two files written by
      one store with two permission stories.
  - id: new
    severity: Minor
    title: |
      Store states no thread-safety contract and the two implementations differ (ARCH-MOCK)
    detail: |
      store.go:10 — Mem guards every method with a mutex; YAML has none and Upsert is a
      non-atomic read-modify-write. The conformance suite cannot catch this because the
      fake is stronger than the real one, which is the fake-diverges-from-real gap the
      suite exists to close. Same shape, lower stakes: Mem.Deck returns a non-nil empty
      slice where YAML.Deck returns nil. Harmless today since the CLI is single-goroutine;
      a one-line interface doc comment settles the intent before issues 4 and 5 consume it.
  - id: new
    severity: Minor
    title: |
      Stale comment at replraw.go:69 still says the store is future work
    detail: |
      "Querying twice doubled the work the History seam will do once #3 backs it with a
      store" — issue 3 now does, three lines above at replraw.go:61.
  - id: new
    severity: Minor
    title: |
      The issue Log records no boundary-review outcome for round 2
    detail: |
      workshop/issues/000003-vocab-store.md:115-134 ends at the implementation notes; there
      is no entry for the round-2 close review or the four Important fixes it produced.
      AGENTS.md section 3 makes logging the review outcome part of crossing the boundary.
```

---

## Re-review — 2026-08-20T21:48:20-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 3 — vocabulary store: per-user YAML deck in a brain, behind a Store seam |
| repo | tools |
| issue file | workshop/issues/000003-vocab-store.md |
| boundary | whole-issue close |
| milestone | — |
| window | e44ac7885e15a22ce5a23bc05d404c812688296f..e54313f2fdb6fb4795050ba39d8ec4ebd58a6a4d |
| command | sdlc close --issue 3 |
| reviewer | claude |
| timestamp | 2026-08-20T21:48:20-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: REWORK
confidence: high
```

The window's first two commits are solid and I re-verified them independently (`go build`, `go vet`, `gofmt -l`, `go test ./...`, `go test -race ./cmd/define/store/...` all clean; `FuzzSlugIsSafe` 6.5M execs clean; the only `time.Now()` in production code is inside `systemClock`, satisfying Done-when 3). The third commit — `e54313f`, the BR-23 torn-record fix — is what blocks. Its atlas half is correct (the blanket atomicity claim is now scoped to word writes, and the event-log paragraph is honest), but its code half ships a data-corruption bug: when the whole-file unmarshal fails, `parseDay` does **not** reset the partially-decoded slice before running the per-record recovery loop, so every whole record in that day file is returned **twice**. I reproduced it end to end through the real store: two real events plus a 3-byte torn append (`- w`) makes `Events` return 4 events — `sycophantic, sycophantic, ephemeral, ephemeral` — while warning "recovered 4 event(s), dropped 1 torn record(s)". 27 of the 78 possible truncation offsets of a single record reproduce it. Two aggravating facts: the shipped test `TestYAMLRecoversFromATornEventRecord` never reaches `splitRecords` at all (its input, `good + "- word: thi"`, is valid YAML, so it exercises only the `complete()` filter), and `event.go` declares the event log "deliberately the ONLY record of activity: every statistic #8 lists … is a fold over these" — so silent duplicates land straight in `#8`'s arithmetic. One line fixes it; the gate should re-run after.

### 1. Strengths

- **`storetest.Suite` is the deliverable and it holds** (`cmd/define/store/storetest/suite.go`), run against `Mem` (`mem_test.go:11`) and `YAML` (`yaml_test.go:15`), with `merge` (`mem.go:34`) and `sortDeck` (`mem.go:58`) shared so the two cannot disagree about upsert semantics or ordering. ARCH-MOCK pass: production flow and test flow share the `Store` boundary, and the fake is package code rather than a `_test.go` alibi.
- **The atlas half of BR-23 was done exactly right** (`atlas/define.md:220-236`). Rather than deleting the awkward sentence, it now contrasts the two file kinds and states what each defends against — "word files are written atomically… event files are appended, deliberately not rewritten." That is the honest-claim discipline `workshop/lessons.md:111` was written to enforce.
- **`complete()` is the right *idea*, documented at the right altitude** (`event.go:26-34`, and the note at `atlas/define.md:233`). "Parsing successfully is not the test" is a genuine insight about append-torn YAML, and putting it in `workshop/lessons.md` rather than only in a comment is the durable move.
- **`Slug` is total and safe under real fuzzing** (`word.go:48`); branch-A output is provably injective because reversibility forces the key to contain no `-`.
- **Round 2's four Important fixes stayed fixed** — `prefixMatch` (`history.go:44`) is still the single definition called by both `History` implementations, `Upsert`'s reset branch still warns (`yaml.go:51`), and `TestEditorPersistsThroughDeps` still pins the wiring that is the issue's purpose.

### 2. Critical findings

**`parseDay` double-counts every whole record when the recovery path runs** — `cmd/define/store/yaml.go:211-221`.

`yaml.Unmarshal(b, &all)` populates `all` with everything it could decode *before* returning its error, and the fallback loop then **appends** the individually-parsed records to that same slice instead of replacing them. Verified by probe against the real store:

```
2 real events + a 3-byte torn append ("- w")
→ Events() = 4:  sycophantic, sycophantic, ephemeral, ephemeral
→ warn:  define: 2026-08-20.yaml: recovered 4 event(s), dropped 1 torn record(s)
```

27 of 78 single-record truncation offsets reproduce this. Recall happens to survive because `prefixMatch` dedupes, but the `Store` contract is violated and `#8`'s folds (words/day, streaks, active days, accuracy) would all be inflated by a duplicated day. It also falsifies `atlas/define.md:235` — "an interrupted write costs the event in flight and nothing else."

*Fix sketch* — one line, at `yaml.go:212`:
```go
if err := yaml.Unmarshal(b, &all); err != nil {
    all = nil // a failed decode leaves PARTIAL results; the split rebuilds from scratch
    for _, rec := range splitRecords(string(b)) {
```

### 3. Important findings

**A truncation inside the timestamp passes `complete()` and is admitted as a real event with a fabricated date** — `cmd/define/store/event.go:31-33`, consumed at `yaml.go:222-228`.

`complete()` tests for presence, not integrity, so a record cut at `at: 2026-08-2` parses as **2026-08-02** and is returned as a whole event — an 18-day-displaced record silently invented from a fragment. `at: 2026-08-20` likewise becomes midnight. This is the exact failure mode `workshop/lessons.md:120` was written about ("a recovery path that tests for a parse error therefore accepts the fragment and silently invents a record"), reappearing one truncation point over, and `atlas/define.md:233-236` states completeness *is* what distinguishes a whole record from a fragment.

*Fix sketch:* the reliable torn-tail signal is the one `AppendEvent` already guarantees — every complete record ends with `\n`. In `parseDay`, if `len(b) > 0 && b[len(b)-1] != '\n'`, the final record is torn by construction: drop it and count it, then apply the existing checks to the rest. That subsumes both this case and the ones `complete()` already catches.

**The recovery path added by this commit has no test that reaches it** — `cmd/define/store/yaml_test.go:121-148`.

`TestYAMLRecoversFromATornEventRecord` appends `- word: thi`, which makes the day file *valid* YAML — I confirmed `yaml.Unmarshal` returns `nil` on that exact input, so the test never enters the `err != nil` branch and `splitRecords` is dead code as far as the suite is concerned. Both defects above live in that unreached branch. Compounding it: `parseDay`, `splitRecords` and `complete()` are pure, in-package, and trivially unit-testable, but `word_test.go` (`package store`) tests only `Key`/`Slug`, and `yaml_test.go` is `package store_test` and cannot see them (ARCH-PURE — the pure core is there, it just isn't being tested as pure).

*Fix sketch:* add a table test in `package store` driving `parseDay` directly over a good record plus each interesting truncation (`- w`, `  kin`, `at: 2026-08-2`, `at: 2026-08-20T10`), asserting both the returned events **and** `torn`. The all-prefixes sweep is ~10 lines and would have caught both findings above:
```go
for i := 1; i < len(rec); i++ {
    ev, torn := parseDay([]byte(good + rec[:i]))
    if len(ev) != 1 || ev[0].Word != "first" || torn != 1 { t.Errorf(...) }
}
```

### 4. Minor findings

- `cmd/define/store/yaml.go:137` — the warning reports `len(day)` as "recovered", which is the post-duplication count; it will read correctly once the Critical is fixed, but the message is currently user-facing and wrong.
- `cmd/define/store/yaml.go:233` — `splitRecords` assumes every record begins at column 0 with `- `. Unreachable today (editor lines carry no newlines, and yaml indents block scalars), but it is an undocumented assumption in a function whose whole job is parsing damaged input.

### 5. Test coverage notes

The suite is strong where the risk was named in the plan and blind exactly where this window's new code lives.

Well covered: the conformance suite runs both implementations against one contract; the disk-only hazards each have a test (reopen, leftover temp file, corrupt word file + warning, day grouping); `Slug` has a table test plus a fuzz target I ran clean to 6.5M execs; `storeHistory`'s three behavioural commitments are each mutation-proof, and `TestStoreHistoryPrefixDoesNotQueryTheStore` (`history_store_test.go:82`) tests a real invariant rather than restating the implementation.

Gaps, in priority order:

1. **The `splitRecords` recovery branch is unreached** by every test in the tree (Important above). This is how both new defects shipped.
2. **`parseDay`/`complete()` have no direct unit test** despite being pure and in-package.
3. **`YAML.Upsert`'s warning branch still has no test** (BR-18) — round 2 changed that branch and it remains invisible to the suite.
4. **Timestamp offset preservation is still unasserted** (BR-9) despite being load-bearing for `#8`; every suite timestamp is `time.UTC`.
5. `yaml_test.go` uses wall-clock `time.Now()` in four tests while the package ships `FixedClock` for exactly this. Harmless — nothing asserts on the value — but it is the one place the diff does not take its own clock-injection medicine.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — flag (residue only).** The substantive round-2 fix (`prefixMatch`) held. What remains is BR-11: the event sort is still byte-identical at `mem.go:83` and `yaml.go:145`, and two `warnf` helpers each hard-code the `"define: "` prefix. `sortDeck` was extracted for precisely this reason; `sortEvents` finishes the thought.
- **ARCH-PURE — flag, narrow.** The shape is right: `Word`, `ReviewEvent`, `Key`, `Slug`, `merge`, `sortDeck`, `Clock`, and now `parseDay`/`splitRecords`/`complete()` are pure, with `YAML` as the thin shell and `openHistory` (`main.go:47-54`) as the entire IO seam. The flag is that the newest pure logic is only reachable from a `package store_test` end-to-end test, so its purity buys no test leverage. Pure recovery logic deserves a pure test.
- **ARCH-PURPOSE — pass on code, one artifact open.** Shadow sweep over the consumers of the storage model: README ✓, atlas ✓ (rewritten this window), project file ✓ (round 2). The **plan** is still a hand-maintained restatement that the code does not derive from — BR-22, four live deltas, no `## Revisions` section.
- **ARCH-MOCK — pass, still the strongest part of the diff.** Stateful fake behind the same seam, both sides run one suite under ordinary `go test`, and the owned component boots from portable non-production storage by construction (`NewYAML(t.TempDir(), nil)`). The standing caveat is BR-25: properties where `Mem` is *stronger* than `YAML` (mutex guarding; non-nil-vs-nil empty `Deck`) are invisible to a conformance suite by construction, so they have to be stated on the interface instead. Worth settling before `#4`/`#5` consume it.
- **For `#5`/`#8`:** three constraints this window creates. `Word` deliberately carries no box/interval — keep the schedule out of `store/word.go` or `#5` inherits a migration. `Events(time.Time{})` still parses every day file ever written on every `define` invocation, including one-shot lookups that never touch history (BR-21); the `since` parameter is already the fix. And `#8` must group by timestamp, never by filename — still guaranteed only by a comment (BR-9).

### 7. Plan revision recommendations

`workshop/plans/000003-vocab-store-plan.md` still has **no `## Revisions` section** (BR-22, re-confirmed by grep). All four deltas are live, and a fifth is now needed:

1. **Type names** — the Integration-points table names `yamlStore`/`memStore`; the shipped exported types are `YAML` (`store/yaml.go:25`) and `Mem` (`store/mem.go:13`), because they are consumed from outside the package as `store.NewYAML`/`store.NewMem`.
2. **`Word` has no `Found` field** — the Pure-entities prose lists `Found bool`; `store/word.go:20-25` has none. The code is right (`found` is a property of a lookup event), and correspondingly the plan's `ReviewEvent` bullet *omits* the `Found` field the shipped struct does carry (`event.go:20`). Both halves of that swap need recording.
3. **`Key` collapses interior whitespace** — slug rule 1 specifies `strings.ToLower(strings.TrimSpace(text))`; `store/word.go:30` is `strings.ToLower(strings.Join(strings.Fields(text), " "))`. The implementation is the better rule and matches `parseREPLLine`; record the change, keep the code.
4. **Constructor signature** — plan and Spec both say `NewStore(dir)`; the shipped constructor is `NewYAML(dir string, warn io.Writer)` (`store/yaml.go:32`).
5. **New:** the plan's Task 3 lists atomicity as a blanket write rule ("**Writes must be atomic** — write to a temp file in the same directory, then rename"). The shipped design deliberately splits this — atomic for words, append-plus-record-level-recovery for events — which the atlas now documents but the plan does not.

Separately, `workshop/plans/000003-vocab-store-plan-gate.md` still lists **PQ-9** under "Open findings"; it was fixed in `8bd988a` and should be disposed `addressed`.

```findings
dispose:
  - id: BR-9
    disposition: not-addressed
    note: |
      No FixedZone anywhere in the tree; every storetest.Suite timestamp is still time.UTC.
  - id: BR-10
    disposition: not-addressed
    note: |
      main.go:3-16 still has the stray blank line after "context" and the store import inside the stdlib group.
  - id: BR-11
    disposition: not-addressed
    note: |
      mem.go:83 and yaml.go:145 still carry the byte-identical event sort; history_store.go:84 and yaml.go:149 still repeat the "define: " prefix literal.
  - id: BR-12
    disposition: not-addressed
    note: |
      history_store.go:84-90 still reads and writes h.warned outside h.mu.
  - id: BR-13
    disposition: not-addressed
    note: |
      One warned flag still covers both the construction-time read failure and every later write failure; the "(history is session-only)" suffix is still appended to "could not save word".
  - id: BR-14
    disposition: not-addressed
    note: |
      No length bound in Slug; word.go has no truncate-plus-hash path.
  - id: BR-15
    disposition: not-addressed
    note: |
      Key at word.go:29-31 still does no Unicode normalisation.
  - id: BR-16
    disposition: not-addressed
    note: |
      store.go:11-12 documents "Lookups accumulates" but not that Upsert can never set an exact count.
  - id: BR-17
    disposition: not-addressed
    note: |
      TestYAMLDifferentWordsTouchDisjointFiles still asserts only the file count, not that alpha.yaml was untouched.
  - id: BR-18
    disposition: not-addressed
    note: |
      Still no test for Upsert's "overwriting unreadable" branch (yaml.go:44-53) — the branch round 2 changed remains invisible to the suite.
  - id: BR-19
    disposition: not-addressed
    note: |
      history_store_test.go:124-128 still hand-rolls failErr instead of errors.New.
  - id: BR-20
    disposition: not-addressed
    note: |
      event.go:20 Found still lacks omitempty while Correct at :21 has it.
  - id: BR-21
    disposition: not-addressed
    note: |
      realDeps still builds openHistory eagerly and newStoreHistory still calls Events(time.Time{}), parsing every day file on every invocation.
  - id: BR-22
    disposition: not-addressed
    note: |
      No "## Revisions" section exists in the plan (grep confirms); all four deltas re-verified live, plus a fifth — the plan states atomic writes as a blanket rule the shipped design deliberately splits.
  - id: BR-23
    disposition: addressed
    note: |
      Atlas is now scoped to word writes and the record-level recovery landed (parseDay/splitRecords/complete). The fix itself ships a duplication defect and an untested branch, raised separately below rather than re-raised here.
  - id: BR-24
    disposition: not-addressed
    note: |
      writeAtomic still inherits 0600 from os.CreateTemp while AppendEvent at yaml.go:102 opens 0644.
  - id: BR-25
    disposition: not-addressed
    note: |
      store.go:5-20 still states no thread-safety contract; Mem is mutex-guarded and YAML is not.
  - id: BR-26
    disposition: not-addressed
    note: |
      replraw.go:70-71 still reads "the History seam will do once #3 backs it with a store".
  - id: BR-27
    disposition: not-addressed
    note: |
      The issue Log still ends at the implementation notes; no entry records the round-2 or round-3 close-review outcomes.
findings:
  - id: new
    severity: Critical
    title: |
      parseDay double-counts every whole record whenever the torn-record recovery path runs
    detail: |
      yaml.go:211-221 — yaml.Unmarshal populates `all` with everything it could decode BEFORE
      returning its error, and the fallback loop appends the individually-parsed records to that
      same slice instead of replacing them. Verified end to end through the real store: two real
      events plus a 3-byte torn append makes Events return 4 (sycophantic, sycophantic, ephemeral,
      ephemeral) and warn "recovered 4 event(s), dropped 1 torn record(s)". 27 of 78 single-record
      truncation offsets reproduce it. Recall survives only because prefixMatch dedupes, but the
      Store contract is violated and event.go:15-17 declares the log the ONLY source every #8
      statistic folds over, so a duplicated day inflates words/day, streaks, active days and
      accuracy. It also falsifies atlas/define.md:235 ("an interrupted write costs the event in
      flight and nothing else"). One-line fix: set `all = nil` immediately inside the `err != nil`
      branch, before the splitRecords loop.
  - id: new
    severity: Important
    title: |
      A truncation inside the timestamp passes complete() and is admitted as a real event with a fabricated date
    detail: |
      event.go:31-33 tests for field presence, not integrity. A record cut at "at: 2026-08-2"
      parses as 2026-08-02 and is returned as a whole event — an 18-day-displaced record invented
      from a fragment; "at: 2026-08-20" likewise becomes midnight. This is the exact failure mode
      workshop/lessons.md:120 was written about, one truncation point over, and atlas/define.md:233
      states completeness IS what distinguishes a whole record from a fragment. Cheap fix: the
      reliable torn-tail signal is the one AppendEvent already guarantees — every complete record
      ends with a newline. In parseDay, if the file does not end in "\n", the final record is torn
      by construction: drop and count it, then apply the existing checks to the rest.
  - id: new
    severity: Important
    title: |
      The torn-record recovery branch added this window has no test that reaches it (ARCH-PURE)
    detail: |
      yaml_test.go:121-148 appends "- word: thi", which leaves the day file VALID YAML — confirmed
      yaml.Unmarshal returns nil on that exact input — so the test never enters the err != nil
      branch and splitRecords is dead code as far as the suite is concerned. Both defects above
      live in that unreached branch. Compounding it, parseDay/splitRecords/complete are pure and
      in-package but yaml_test.go is package store_test and cannot see them, while word_test.go
      (package store) tests only Key and Slug — so the pure core buys no test leverage. A table
      test in package store driving parseDay over each interesting truncation, asserting both the
      events and the torn count, is about ten lines and catches both.
```

---

## Re-review — 2026-08-20T21:58:57-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 3 — vocabulary store: per-user YAML deck in a brain, behind a Store seam |
| repo | tools |
| issue file | workshop/issues/000003-vocab-store.md |
| boundary | whole-issue close |
| milestone | — |
| window | e44ac7885e15a22ce5a23bc05d404c812688296f..af70ea73d387a6dae49200138859c233fb0aa591 |
| command | sdlc close --issue 3 |
| reviewer | claude |
| timestamp | 2026-08-20T21:58:57-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 4's Critical is genuinely gone and I verified it rather than reading the commit message: `parseDay` now has one path, `TestYAMLRecoveryDoesNotDuplicate` pins it, and removing the round-trip check fails three tests — so BR-28, BR-29 and BR-30 are all closed by real coverage. An exhaustive sweep of all 72 truncation offsets of a final record returns exactly the whole records plus `torn=1` in 70 of them; the two deviations are one benign (a record complete but for its trailing newline is admitted, correctly) and one real. Everything else is clean: `go build`, `go vet`, `gofmt -l`, `go test ./...`, `go test -race ./cmd/define/store/...`, `FuzzSlugIsSafe` to 6.2M execs, and the only `time.Now()` in production is inside `systemClock`. What blocks SHIP is two defects in the recovery code this window added, both verified end to end through the real store and both cheap. First, a torn fragment is not line-isolated: after a killed mid-append the *next session's ordinary lookup* is glued onto the fragment and permanently lost — I recorded `sycophantic`, tore an append, then recorded `quotidian`, and `Events` returns only `sycophantic` while warning "dropped 1 torn record(s)". That directly falsifies `atlas/define.md:231`. Second, validity is byte-equality with this yaml version's emitter applied to *every* record, not just the one an append can tear — so a CRLF-normalised day file drops all 3 of 3 records, and one requoted field drops that record.

### 1. Strengths

- **The round-trip rule is the right insight and it is load-bearing.** I deleted the check at `yaml.go:225-229` and three tests failed with exactly the fabricated records the comment predicts — `{Word:third … At:2026-08-20 00:00:00}` from a mid-timestamp cut. The tests are mutation-proof, not decorative.
- **One parse path, and the comment explains the corpse.** `yaml.go:204-207` records *why* the two-path version was wrong (double-counting, unreachable fallback) rather than just deleting it. Same for `workshop/lessons.md:118-131`, which generalises both failed heuristics rather than the one instance.
- **The atlas rewrite is honest engineering prose** (`atlas/define.md:220-244`): it contrasts what the two file kinds each defend against instead of asserting a blanket property, and it names the two weaker tests that looked sufficient. Apart from the one sentence at :231 this is a model entry.
- **`storetest.Suite` still carries the diff** — run against `Mem` (`mem_test.go:11`) and `YAML` (`yaml_test.go:15`), with `merge` and `sortDeck` shared so the two cannot disagree. ARCH-MOCK pass: the fake is package code, not a `_test.go` alibi.
- **Timestamps really do survive.** Probed rather than assumed: a `FixedZone(-07:00)` event round-trips through marshal/unmarshal with `off=-25200` intact, and every emitter-stressing headword I tried (`yes`, `null`, `123`, `a: b`, `#hash`, `café`, `-dash`) round-trips clean.

### 2. Critical findings

None. BR-28 is fixed.

### 3. Important findings

**A torn fragment is not line-isolated, so the next session's append is swallowed with it** — `cmd/define/store/yaml.go:236-251` (`splitRecords`), with `AppendEvent` at `:102-112`.

`splitRecords` starts a new record only at a line beginning `- ` at column 0. A fragment left by a killed append has **no trailing newline**, so the next `AppendEvent` writes its record onto the same line. Verified end to end:

```
file  = "- word: sycophantic\n  kind: …\n- word: ephe- word: quotidian\n  kind: …\n"
Events() = [sycophantic]
warn     = "define: 2026-08-20.yaml: recovered 1 event(s), dropped 1 torn record(s)"
```

`quotidian` was a complete, successfully-written record from a healthy session and it is unrecoverable — permanently, since day files are never rewritten. Across all fragment lengths the following record is lost in **69 of 72** cases. The mirror case: a fragment that does not start with `- ` (a 1-byte `-`) glues onto the record *before* it and destroys that one instead. Either way `atlas/define.md:231` — "An interrupted write costs the event in flight and nothing else" — is false, and the warning under-reports (1 torn, 2 events gone).

*Fix sketch* — make the fragment self-delimiting at write time, in `AppendEvent`, ~4 lines:
```go
if st, err := f.Stat(); err == nil && st.Size() > 0 {
    var last [1]byte
    if _, err := f.ReadAt(last[:], st.Size()-1); err == nil && last[0] != '\n' {
        f.Write([]byte("\n")) // a torn fragment must not swallow the next record
    }
}
```
(needs `O_RDWR|O_APPEND`). That also buys the invariant the next finding wants: **only the final record of a file can ever be torn.**

**Byte-identical round-trip is applied to every record, so a reformatted log is silently discarded** — `cmd/define/store/yaml.go:225-229`.

The rule as implemented is not "this is a whole record", it is "these bytes are exactly what *this build's* `yaml.Marshal` emits". Every record in the file is held to it, though an interrupted append can only ever tear the last one. Verified through the real store — each of these is semantically intact and every record is dropped:

| input | result |
|---|---|
| CRLF line endings (3 whole records) | `events=0`, "dropped 3 torn record(s)" |
| one field requoted by hand (`word: "sycophantic"`) | `events=0`, "dropped 1 torn record(s)" |
| reordered keys / a trailing comment / 4-space indent / explicit `correct: false` | dropped |

The sharpest trigger is a routine `go get -u`: any change to the yaml emitter's quoting or indentation reclassifies **the entire accumulated history** as torn, and `event.go:15-17` declares that log the only source every `#8` statistic folds over. It degrades with a warning rather than crashing, but it is total loss of the thing the log exists for.

*Fix sketch:* once the fix above guarantees torn ⇒ last record, apply the strict round-trip only to the final record and accept earlier ones on unmarshal success. Blast radius drops from "the whole file" to "one record", truncation detection is unchanged, and the `parseDay` doc comment gets to state the invariant it now relies on.

### 4. Minor findings

- `cmd/define/store/yaml_test.go:121` — `TestYAMLRecoversFromATornEventRecord` is now a strict subset of `TestYAMLDropsEveryShapeOfTornRecord`'s `cut mid-value` case (`:161`); same setup, same tail, weaker assertions. One of them is redundant (ARCH-DRY, tests).
- `cmd/define/store/yaml.go:137` — the warning counts *records* dropped, not *events* lost; under the glue case above those differ, and the number a user sees is the smaller one.

### 5. Test coverage notes

The two gaps that made round 4 REWORK are closed, and I confirmed by mutation rather than by reading: deleting the round-trip check fails `TestYAMLRecoversFromATornEventRecord` plus two `TestYAMLDropsEveryShapeOfTornRecord` subtests. Because `parseDay` is now the only path, every existing event test exercises `splitRecords` — the unreachable-branch problem is structurally gone, which is a better fix than adding a test to reach it.

Gaps, in priority order:

1. **Nothing writes a fragment and then appends again** — the sequence a real Ctrl-C-then-rerun produces, and the one that loses a valid record. The six-case table at `yaml_test.go:154` always tears *last*.
2. **Nothing feeds `parseDay` a well-formed record it did not itself emit.** Every event test writes through `AppendEvent`, so the suite can never observe that the validity rule is emitter-specific. One case with a hand-written-but-valid record would pin the intended contract either way.
3. **`parseDay`/`splitRecords` still have no direct unit test** — they are pure and in-package, but the only tests live in `package store_test` and reach them through the filesystem. The exhaustive-truncation sweep that found the glue case is ~8 lines in `package store` and needs no temp dir.
4. `YAML.Upsert`'s warning branch still has no test (BR-18); timestamp-offset preservation is still unasserted (BR-9) despite being load-bearing for `#8` — I verified it holds today by probe, but nothing in the suite would notice it breaking.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — flag (residue only).** `prefixMatch` held. Still open from round 2: the byte-identical event sort at `mem.go:83` / `yaml.go:145` (BR-11) and two `warnf` helpers each hard-coding `"define: "`. New this window, minor: the two overlapping torn-record tests.
- **ARCH-PURE — pass, with the same caveat as last round.** `Word`, `ReviewEvent`, `Key`, `Slug`, `merge`, `sortDeck`, `Clock`, `parseDay`, `splitRecords` are genuinely pure; `YAML` is the thin shell and `openHistory` (`main.go:45-56`) is the entire IO seam. The caveat is that the pure recovery core is only exercised through the disk, so its purity is buying no test leverage — which is why an off-by-one in string splitting reached a boundary review twice.
- **ARCH-PURPOSE — pass on code, two artifacts open.** Shadow-sweep over consumers of the storage model: README ✓, atlas ✓ (except the one falsified sentence at :231), project file ✓, code ✓. The **plan** is still a hand-maintained restatement the code does not derive from — BR-22, now five deltas, still no `## Revisions` section. And the issue `## Log` still records no boundary-review outcome for rounds 2–4 (BR-27), which AGENTS.md §3 makes part of crossing the boundary.
- **ARCH-MOCK — pass, still the strongest part of the diff.** Stateful fake behind the same seam, one suite over both, portable non-production storage by construction (`NewYAML(t.TempDir(), nil)`). Standing caveat BR-25: properties where `Mem` is *stronger* than `YAML` (mutex guarding, non-nil vs nil empty `Deck`) are invisible to a conformance suite by construction and have to be stated on the interface instead. Worth settling before `#4`/`#5` consume it. Note the same blind spot covers the event-log parse: `Mem` has no serialisation, so no conformance obligation constrains what `YAML` considers a valid stored record — that contract lives only in `yaml_test.go`.
- **For `#5`/`#8`:** `Word` deliberately carries no box/interval — keep the schedule out of `store/word.go` or `#5` inherits a migration. `Events(time.Time{})` still parses every day file on every `define` invocation (BR-21), and this window roughly doubles that cost by re-marshalling every record to validate it; the `since` parameter is already the fix. And because nothing ever repairs a day file, one torn record means `define` prints its warning on **every** future invocation forever, including the one-shot `define <word>` path that never touches history.

### 7. Plan revision recommendations

`workshop/plans/000003-vocab-store-plan.md` still has **no `## Revisions` section** (grep confirms). All five deltas are live and re-verified against the tree:

1. **Type names** — the Integration-points table names `yamlStore`/`memStore`; the shipped exported types are `YAML` (`store/yaml.go:25`) and `Mem` (`store/mem.go:13`).
2. **`Word` has no `Found` field** — the Pure-entities prose lists `Found bool`; `store/word.go:20-25` has none, and correspondingly the plan's `ReviewEvent` bullet omits the `Found` field the shipped struct does carry (`event.go:21`). Both halves of that swap need recording; the code is right.
3. **`Key` collapses interior whitespace** — slug rule 1 says `strings.TrimSpace`; `word.go:30` is `strings.Join(strings.Fields(text), " ")`. The implementation is the better rule; record the change.
4. **Constructor signature** — plan and Spec say `NewStore(dir)`; the shipped constructor is `NewYAML(dir string, warn io.Writer)` (`store/yaml.go:32`).
5. **Atomicity is not a blanket rule** — the plan states "**Writes must be atomic** — write to a temp file … then rename" for `yamlStore` as a whole. The shipped design deliberately splits it: atomic for words, append-plus-record-level-recovery for events. The atlas now documents the split; the plan does not.

Also worth adding to the plan's Core concepts: `parseDay`/`splitRecords` are now pure entities in `store/yaml.go` with real invariants, and the table that governs the review has no row for them.

```findings
dispose:
  - id: BR-9
    disposition: not-addressed
    note: |
      No FixedZone anywhere in the tree; every storetest.Suite timestamp is still time.UTC. Verified by probe that the offset does survive today, so this remains an untested contract rather than a bug.
  - id: BR-10
    disposition: not-addressed
    note: |
      main.go:3-16 still has the stray blank line after "context" and the store import inside the stdlib group.
  - id: BR-11
    disposition: not-addressed
    note: |
      mem.go:83 and yaml.go:145 still carry the byte-identical event sort; history_store.go:89 and yaml.go:151 still repeat the prefix literal.
  - id: BR-12
    disposition: not-addressed
    note: |
      history_store.go:85-91 still reads and writes h.warned outside h.mu.
  - id: BR-13
    disposition: not-addressed
    note: |
      One warned flag still covers both the construction-time read failure and every later write failure; the session-only suffix is still appended to "could not save word".
  - id: BR-14
    disposition: not-addressed
    note: |
      No length bound in Slug; word.go has no truncate-plus-hash path.
  - id: BR-15
    disposition: not-addressed
    note: |
      Key at word.go:29-31 still does no Unicode normalisation.
  - id: BR-16
    disposition: not-addressed
    note: |
      store.go:11-12 documents "Lookups accumulates" but not that Upsert can never set an exact count.
  - id: BR-17
    disposition: not-addressed
    note: |
      TestYAMLDifferentWordsTouchDisjointFiles still asserts only the file count, not that alpha.yaml was untouched.
  - id: BR-18
    disposition: not-addressed
    note: |
      Still no test for Upsert's "overwriting unreadable" branch at yaml.go:44-53.
  - id: BR-19
    disposition: not-addressed
    note: |
      history_store_test.go:124-128 still hand-rolls failErr instead of errors.New.
  - id: BR-20
    disposition: not-addressed
    note: |
      event.go:21 Found still lacks omitempty while Correct at :22 has it.
  - id: BR-21
    disposition: not-addressed
    note: |
      Still eager, and this window amplifies it — parseDay re-marshals every record to validate it, so startup now costs two yaml operations per stored event, and a torn record makes the warning print on every future invocation forever.
  - id: BR-22
    disposition: not-addressed
    note: |
      No "## Revisions" section exists (grep confirms); all five deltas re-verified live, and the Core concepts table now also lacks rows for parseDay/splitRecords.
  - id: BR-24
    disposition: not-addressed
    note: |
      writeAtomic still inherits 0600 from os.CreateTemp while AppendEvent at yaml.go:102 opens 0644.
  - id: BR-25
    disposition: not-addressed
    note: |
      store.go:5-20 still states no thread-safety contract. Note the same blind spot now also covers what counts as a valid stored record — Mem has no serialisation, so no conformance obligation constrains it.
  - id: BR-26
    disposition: not-addressed
    note: |
      replraw.go:68-69 still reads "the History seam will do once #3 backs it with a store".
  - id: BR-27
    disposition: not-addressed
    note: |
      The issue Log still ends at the implementation notes; no entry records the round-2, round-3 or round-4 close-review outcomes.
  - id: BR-28
    disposition: addressed
    note: |
      The whole-file parse path is gone; parseDay is record-by-record only. TestYAMLRecoveryDoesNotDuplicate pins it, and an exhaustive sweep of all 72 truncation offsets produces no duplicate.
  - id: BR-29
    disposition: addressed
    note: |
      Verified by probe — "at: 2026-08-2", "at: 2026-08-20" and a cut inside the time all now return torn=1 and admit no fabricated event. The round-trip check subsumes complete(), which is gone.
  - id: BR-30
    disposition: addressed
    note: |
      One parse path means every event test now reaches splitRecords; deleting the round-trip check fails three tests. The suggested pure in-package test was not added, which is re-raised as a coverage note rather than as this finding.
findings:
  - id: new
    severity: Important
    title: |
      A torn fragment is not line-isolated, so the next session's append is glued to it and permanently lost
    detail: |
      splitRecords (yaml.go:236-251) starts a record only at a line beginning "- " at column 0, but a
      fragment left by a killed append has no trailing newline, so the next AppendEvent writes onto the
      same line. Verified end to end through the real store: record "sycophantic", tear an append, then
      record "quotidian" from a healthy session, and Events returns only [sycophantic] while warning
      "dropped 1 torn record(s)" — quotidian is unrecoverable, permanently, since day files are never
      rewritten. The following record is lost in 69 of 72 fragment lengths. The mirror case, a fragment
      not starting with "- " such as a 1-byte "-", destroys the PRECEDING record instead. Either way
      atlas/define.md:231 ("an interrupted write costs the event in flight and nothing else") is false,
      and the warning under-reports — 1 torn record, 2 events gone. Fix at write time in AppendEvent,
      about four lines: open O_RDWR|O_APPEND, and if the file is non-empty and its last byte is not a
      newline, write one first. That also establishes the invariant the next finding needs — only the
      final record of a file can ever be torn.
  - id: new
    severity: Important
    title: |
      Byte-identical round-trip is applied to every record, so any reformatted log is silently discarded
    detail: |
      yaml.go:225-229 decides validity by exact string equality with what THIS build's yaml.Marshal
      emits, and applies it to every record even though an interrupted append can only tear the last
      one. Verified through the real store, each input semantically intact: CRLF line endings drop all
      3 of 3 records ("recovered 0 event(s), dropped 3 torn record(s)"); one hand-requoted field drops
      that record; reordered keys, a trailing comment, 4-space indent and an explicit "correct: false"
      each drop too. The sharpest trigger is a routine dependency bump — any change to the emitter's
      quoting or indentation reclassifies the entire accumulated history as torn, and event.go:15-17
      declares that log the only source every issue-8 statistic folds over. It warns rather than
      crashing, but it is total loss of the thing the log exists for. Once the fix above guarantees
      torn implies last-record, apply the strict round-trip only to the final record and accept earlier
      ones on unmarshal success — truncation detection is unchanged and the blast radius drops from the
      whole file to one record.
  - id: new
    severity: Minor
    title: |
      Two torn-record tests overlap; one is a strict subset of the other
    detail: |
      yaml_test.go:121 TestYAMLRecoversFromATornEventRecord uses the same setup and the same tail as
      TestYAMLDropsEveryShapeOfTornRecord's "cut mid-value" case at :161, with weaker assertions.
  - id: new
    severity: Minor
    title: |
      The torn warning counts dropped records, not lost events, and the two differ
    detail: |
      yaml.go:137 reports the number of unparseable chunks. Under the glue case above one chunk holds
      both the fragment and a whole record, so the user is told "dropped 1" when two events are gone.
```
