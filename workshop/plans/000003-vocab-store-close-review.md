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
