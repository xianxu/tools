---
gate: boundary-review
issue: 3
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-20T21:30:16-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Task 4 does not say where History is injected or what the nil default is
          detail: |-
            A deps field touches realDeps (main.go:31) and the rigs at main_test.go:14,41; a
            runEditor parameter touches twelve call sites in editorloop_test.go. replraw.go:61
            currently guarantees non-nil, so the deps route nil-panics every existing rig
            unless a default is stated.
            (carried from plan-quality PQ-6, deferred to the boundary review)
          round: 1
        - id: BR-2
          severity: Minor
          title: Task 1 Step 1 and Task 3 Step 2 enumerate test cases in prose; compress to one strategy line per risky function
          detail: |-
            Both lists are lossy pre-images of code that will exist within the hour. Task 2's
            suite obligations are the seam contract and should stay.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          round: 1
        - id: BR-3
          severity: Minor
          title: go.yaml.in/yaml/v3 is not a dependency of this module and no step adds it
          detail: |-
            go.mod declares github.com/xianxu/tools with only creack/pty and golang.org/x/term,
            so the estimate's "already an ariadne dependency" does not transfer. Pin @v3.0.4,
            which is present in the shared module cache, since an @latest query needs a proxy
            this environment may not reach.
            (carried from plan-quality PQ-8, deferred to the boundary review)
          round: 1
        - id: BR-4
          severity: Minor
          title: The define-learn project file still records the retired brain/nous-push storage decision
          detail: |-
            workshop/projects/define-learn.md lists "Storage shape is chosen for git... nous
            push supplies sync" under design decisions taken up front, which the narrowed spec
            replaced with cwd-only, no git of any kind.
            (carried from plan-quality PQ-9, deferred to the boundary review)
          round: 1
      boundary: '*'
      blocked: false
    - "n": 2
      timestamp: "2026-08-20T21:30:16-07:00"
      agent: claude
      findings:
        - id: BR-5
          severity: Important
          title: storeHistory.Prefix duplicates memHistory.Prefix verbatim (ARCH-DRY)
          detail: |-
            cmd/define/history_store.go:76 and cmd/define/history.go:34 have
            byte-identical bodies apart from the mutex, so the History contract
            "newest first, deduped" now has two definitions that can drift. Coverage
            is lopsided too: editor_test.go's hist() helper builds memHistory, so
            ~15 recall tests exercise the fallback copy while the production copy has
            one. Extract prefixMatch(lines, p) into history.go and call it from both.
          round: 2
        - id: BR-6
          severity: Important
          title: YAML.Upsert silently resets a word's history when the existing file is unreadable
          detail: |-
            cmd/define/store/yaml.go:44-49 sets old = Word{} for any non-NotExist read
            error with no y.warnf, so merge writes back FirstSeen = now and Lookups = 1
            and the word's history is destroyed silently. Deck (yaml.go:70-78) warns on
            the identical condition, and the issue's Done-when promises "skipped with a
            warning". Sharper than it looks because the README markets running inside a
            synced directory, where a transiently-unreadable placeholder file is a
            realistic input. One-line fix: y.warnf before the reset.
          round: 2
        - id: BR-7
          severity: Important
          title: Project file still documents the git/brain/nous-push storage model (PQ-9)
          detail: |-
            workshop/projects/define-learn.md:54-56 still says "Storage shape is chosen
            for git ... a brain syncs across machines ... nous push supplies sync", and
            line 40 lists issue 3 as "YAML in a brain". Both contradict the shipped
            cwd-only design and the plan's own Non-goals. This is the hand-maintained
            restatement ARCH-PURPOSE's shadow-sweep names; it was raised at
            plan-quality as PQ-9 and disposed not-addressed in round 2. The close gate
            ticks checkboxes but will not rewrite this prose.
          round: 2
        - id: BR-8
          severity: Important
          title: No test covers the deps.history to runEditor wiring — the issue's purpose
          detail: |-
            No test anywhere sets deps.history, so replraw.go:61's hist := d.history is
            only exercised through the nil fallback to memHistory. Reverting that line
            to "var hist History = &memHistory{}" leaves the whole suite green while
            persistence silently stops working; the only evidence it works today is a
            manual check in the Log. Related and equally cheap: nothing pins that
            newStoreHistory (history_store.go:37-48) restores not-found lines, so
            adding "&& e.Found" to the restore loop breaks typo recall across a restart
            with zero test signal.
          round: 2
        - id: BR-9
          severity: Minor
          title: Timestamp offset preservation is a load-bearing contract with no test
          detail: |-
            atlas/define.md and yaml.go:89-93 commit to "timestamps keep their offset"
            so issue 8 can recover a local-day view from UTC-named files. Every
            timestamp in storetest.Suite is time.UTC, so the round-trip that matters is
            unasserted. It holds today (verified: 2026-08-20T20:00:00-07:00 round-trips
            intact and lands in 2026-08-21.yaml), but a yaml-library bump would break
            issue 8 silently. Add one time.FixedZone case asserting the Zone offset.
          round: 2
        - id: BR-10
          severity: Minor
          title: main.go import block has a stray blank line and a third-party import in the stdlib group
          detail: |-
            cmd/define/main.go:3-16. gofmt accepts it because it is alphabetical within
            the group, but it differs from every other file in the package.
          round: 2
        - id: BR-11
          severity: Minor
          title: Event sort and warnf helper are each duplicated across the two stores
          detail: |-
            store/mem.go:83 and store/yaml.go:140 repeat the event sort verbatim;
            sortDeck was extracted for exactly this reason, so a matching sortEvents
            finishes the job. history_store.go:94 and store/yaml.go:144 are two
            near-identical warnf helpers repeating the "define: " prefix literal.
          round: 2
        - id: BR-12
          severity: Minor
          title: storeHistory.warned is read and written outside the mutex
          detail: |-
            history_store.go:98-100 touches h.warned without h.mu, which Add and Prefix
            otherwise hold. Single-goroutine in practice, but the inconsistency in an
            otherwise mutex-guarded type invites a future race.
          round: 2
        - id: BR-13
          severity: Minor
          title: A construction-time read failure consumes the one-warning budget for the whole session
          detail: |-
            history_store.go:37-41 warns via the same warned flag that later write
            failures use, so an unreadable events directory at startup silences every
            subsequent write failure. Two different failure classes share one budget.
            Separately, the "(history is session-only)" suffix at line 71 is appended
            even to "could not save word", where the event may have saved fine.
          round: 2
        - id: BR-14
          severity: Minor
          title: Slug does not bound filename length, so a very long headword fails to save
          detail: |-
            store/word.go:48. A 300-char word produces a 300-char filename and Upsert
            fails with ENAMETOOLONG (verified). It degrades correctly — error returned,
            warn-once, session continues — but truncate-plus-hash past ~200 bytes would
            make it work rather than merely not crash.
          round: 2
        - id: BR-15
          severity: Minor
          title: Key does no Unicode normalisation, so NFC and NFD spellings are two words
          detail: |-
            store/word.go:29. Not reachable through the editor today since
            parseREPLLine does not normalise either, but it will matter once issue 4
            captures from other entry points.
          round: 2
        - id: BR-16
          severity: Minor
          title: merge always increments Lookups, so Upsert cannot correct a count
          detail: |-
            store/mem.go:43 computes old.Lookups + max(1, w.Lookups), so re-upserting an
            identical Word double-counts and a caller can never set an exact value.
            Fine for the current callers; worth a doc line on the Store interface since
            issues 4 and 8 will consume it.
          round: 2
        - id: BR-17
          severity: Minor
          title: TestYAMLDifferentWordsTouchDisjointFiles asserts file count, not disjointness
          detail: |-
            store/yaml_test.go:99-112 checks only that the file count grew by one. The
            property it names would be pinned by capturing alpha.yaml's content or
            mtime before the second write and asserting it is unchanged after.
          round: 2
        - id: BR-18
          severity: Minor
          title: YAML.Upsert's corrupt-file branch has no test
          detail: |-
            store/yaml.go:44-49 is dead code as far as the suite is concerned, which is
            how the missing warning survived to this boundary.
          round: 2
        - id: BR-19
          severity: Minor
          title: errFail hand-rolls what errors.New already provides
          detail: |-
            cmd/define/history_store_test.go:120-127 defines a failErr type and a
            package-level errFail instead of errors.New("store unavailable").
          round: 2
        - id: BR-20
          severity: Minor
          title: Found lacks omitempty while Correct has it, within one struct
          detail: |-
            store/event.go:20-21. Day files always carry "found: false" rows for review
            events. Harmless, but inconsistent.
          round: 2
        - id: BR-21
          severity: Minor
          title: History is constructed eagerly and loads every day file on every invocation
          detail: |-
            realDeps() at main.go:40 builds the store even for one-shot "define <word>"
            and piped input, neither of which uses history, and newStoreHistory calls
            Events(time.Time{}) which parses every day file ever written. ~0ms today, a
            growing tax later; the since parameter is already the fix for recall. A
            corrupt events file also makes "define <word>" print store warnings on a
            path that never uses the store, since YAML.warnf has no once-guard.
          round: 2
        - id: BR-22
          severity: Minor
          title: Plan needs a Revisions entry — four documented deltas the code does not match
          detail: |-
            (1) the table names yamlStore/memStore but the shipped types are YAML/Mem;
            (2) Word is specified with a Found bool that does not exist (correctly — found
            is a property of an event); (3) Key collapses interior whitespace via
            strings.Fields rather than TrimSpace as slug rule 1 states; (4) the
            constructor is NewYAML(dir, warn io.Writer), not NewStore(dir). Also, the
            gate ledger still carries PQ-9 as open, which finding I-3 confirms is
            genuinely still open in the tree.
          round: 2
      blocked: false
---

# Gate ledger — tools#3 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-20T21:30:16-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] Task 4 does not say where History is injected or what the nil default is
  A deps field touches realDeps (main.go:31) and the rigs at main_test.go:14,41; a
  runEditor parameter touches twelve call sites in editorloop_test.go. replraw.go:61
  currently guarantees non-nil, so the deps route nil-panics every existing rig
  unless a default is stated.
  (carried from plan-quality PQ-6, deferred to the boundary review)
- **BR-2** [Minor] Task 1 Step 1 and Task 3 Step 2 enumerate test cases in prose; compress to one strategy line per risky function
  Both lists are lossy pre-images of code that will exist within the hour. Task 2's
  suite obligations are the seam contract and should stay.
  (carried from plan-quality PQ-7, deferred to the boundary review)
- **BR-3** [Minor] go.yaml.in/yaml/v3 is not a dependency of this module and no step adds it
  go.mod declares github.com/xianxu/tools with only creack/pty and golang.org/x/term,
  so the estimate's "already an ariadne dependency" does not transfer. Pin @v3.0.4,
  which is present in the shared module cache, since an @latest query needs a proxy
  this environment may not reach.
  (carried from plan-quality PQ-8, deferred to the boundary review)
- **BR-4** [Minor] The define-learn project file still records the retired brain/nous-push storage decision
  workshop/projects/define-learn.md lists "Storage shape is chosen for git... nous
  push supplies sync" under design decisions taken up front, which the narrowed spec
  replaced with cwd-only, no git of any kind.
  (carried from plan-quality PQ-9, deferred to the boundary review)

## Round 2 — 2026-08-20T21:30:16-07:00 (claude) — passed

### Raised

- **BR-5** [Important] storeHistory.Prefix duplicates memHistory.Prefix verbatim (ARCH-DRY)
  cmd/define/history_store.go:76 and cmd/define/history.go:34 have
  byte-identical bodies apart from the mutex, so the History contract
  "newest first, deduped" now has two definitions that can drift. Coverage
  is lopsided too: editor_test.go's hist() helper builds memHistory, so
  ~15 recall tests exercise the fallback copy while the production copy has
  one. Extract prefixMatch(lines, p) into history.go and call it from both.
- **BR-6** [Important] YAML.Upsert silently resets a word's history when the existing file is unreadable
  cmd/define/store/yaml.go:44-49 sets old = Word{} for any non-NotExist read
  error with no y.warnf, so merge writes back FirstSeen = now and Lookups = 1
  and the word's history is destroyed silently. Deck (yaml.go:70-78) warns on
  the identical condition, and the issue's Done-when promises "skipped with a
  warning". Sharper than it looks because the README markets running inside a
  synced directory, where a transiently-unreadable placeholder file is a
  realistic input. One-line fix: y.warnf before the reset.
- **BR-7** [Important] Project file still documents the git/brain/nous-push storage model (PQ-9)
  workshop/projects/define-learn.md:54-56 still says "Storage shape is chosen
  for git ... a brain syncs across machines ... nous push supplies sync", and
  line 40 lists issue 3 as "YAML in a brain". Both contradict the shipped
  cwd-only design and the plan's own Non-goals. This is the hand-maintained
  restatement ARCH-PURPOSE's shadow-sweep names; it was raised at
  plan-quality as PQ-9 and disposed not-addressed in round 2. The close gate
  ticks checkboxes but will not rewrite this prose.
- **BR-8** [Important] No test covers the deps.history to runEditor wiring — the issue's purpose
  No test anywhere sets deps.history, so replraw.go:61's hist := d.history is
  only exercised through the nil fallback to memHistory. Reverting that line
  to "var hist History = &memHistory{}" leaves the whole suite green while
  persistence silently stops working; the only evidence it works today is a
  manual check in the Log. Related and equally cheap: nothing pins that
  newStoreHistory (history_store.go:37-48) restores not-found lines, so
  adding "&& e.Found" to the restore loop breaks typo recall across a restart
  with zero test signal.
- **BR-9** [Minor] Timestamp offset preservation is a load-bearing contract with no test
  atlas/define.md and yaml.go:89-93 commit to "timestamps keep their offset"
  so issue 8 can recover a local-day view from UTC-named files. Every
  timestamp in storetest.Suite is time.UTC, so the round-trip that matters is
  unasserted. It holds today (verified: 2026-08-20T20:00:00-07:00 round-trips
  intact and lands in 2026-08-21.yaml), but a yaml-library bump would break
  issue 8 silently. Add one time.FixedZone case asserting the Zone offset.
- **BR-10** [Minor] main.go import block has a stray blank line and a third-party import in the stdlib group
  cmd/define/main.go:3-16. gofmt accepts it because it is alphabetical within
  the group, but it differs from every other file in the package.
- **BR-11** [Minor] Event sort and warnf helper are each duplicated across the two stores
  store/mem.go:83 and store/yaml.go:140 repeat the event sort verbatim;
  sortDeck was extracted for exactly this reason, so a matching sortEvents
  finishes the job. history_store.go:94 and store/yaml.go:144 are two
  near-identical warnf helpers repeating the "define: " prefix literal.
- **BR-12** [Minor] storeHistory.warned is read and written outside the mutex
  history_store.go:98-100 touches h.warned without h.mu, which Add and Prefix
  otherwise hold. Single-goroutine in practice, but the inconsistency in an
  otherwise mutex-guarded type invites a future race.
- **BR-13** [Minor] A construction-time read failure consumes the one-warning budget for the whole session
  history_store.go:37-41 warns via the same warned flag that later write
  failures use, so an unreadable events directory at startup silences every
  subsequent write failure. Two different failure classes share one budget.
  Separately, the "(history is session-only)" suffix at line 71 is appended
  even to "could not save word", where the event may have saved fine.
- **BR-14** [Minor] Slug does not bound filename length, so a very long headword fails to save
  store/word.go:48. A 300-char word produces a 300-char filename and Upsert
  fails with ENAMETOOLONG (verified). It degrades correctly — error returned,
  warn-once, session continues — but truncate-plus-hash past ~200 bytes would
  make it work rather than merely not crash.
- **BR-15** [Minor] Key does no Unicode normalisation, so NFC and NFD spellings are two words
  store/word.go:29. Not reachable through the editor today since
  parseREPLLine does not normalise either, but it will matter once issue 4
  captures from other entry points.
- **BR-16** [Minor] merge always increments Lookups, so Upsert cannot correct a count
  store/mem.go:43 computes old.Lookups + max(1, w.Lookups), so re-upserting an
  identical Word double-counts and a caller can never set an exact value.
  Fine for the current callers; worth a doc line on the Store interface since
  issues 4 and 8 will consume it.
- **BR-17** [Minor] TestYAMLDifferentWordsTouchDisjointFiles asserts file count, not disjointness
  store/yaml_test.go:99-112 checks only that the file count grew by one. The
  property it names would be pinned by capturing alpha.yaml's content or
  mtime before the second write and asserting it is unchanged after.
- **BR-18** [Minor] YAML.Upsert's corrupt-file branch has no test
  store/yaml.go:44-49 is dead code as far as the suite is concerned, which is
  how the missing warning survived to this boundary.
- **BR-19** [Minor] errFail hand-rolls what errors.New already provides
  cmd/define/history_store_test.go:120-127 defines a failErr type and a
  package-level errFail instead of errors.New("store unavailable").
- **BR-20** [Minor] Found lacks omitempty while Correct has it, within one struct
  store/event.go:20-21. Day files always carry "found: false" rows for review
  events. Harmless, but inconsistent.
- **BR-21** [Minor] History is constructed eagerly and loads every day file on every invocation
  realDeps() at main.go:40 builds the store even for one-shot "define <word>"
  and piped input, neither of which uses history, and newStoreHistory calls
  Events(time.Time{}) which parses every day file ever written. ~0ms today, a
  growing tax later; the since parameter is already the fix for recall. A
  corrupt events file also makes "define <word>" print store warnings on a
  path that never uses the store, since YAML.warnf has no once-guard.
- **BR-22** [Minor] Plan needs a Revisions entry — four documented deltas the code does not match
  (1) the table names yamlStore/memStore but the shipped types are YAML/Mem;
  (2) Word is specified with a Found bool that does not exist (correctly — found
  is a property of an event); (3) Key collapses interior whitespace via
  strings.Fields rather than TrimSpace as slug rule 1 states; (4) the
  constructor is NewYAML(dir, warn io.Writer), not NewStore(dir). Also, the
  gate ledger still carries PQ-9 as open, which finding I-3 confirms is
  genuinely still open in the tree.

## Open findings

- **BR-1** [Minor] Task 4 does not say where History is injected or what the nil default is
- **BR-2** [Minor] Task 1 Step 1 and Task 3 Step 2 enumerate test cases in prose; compress to one strategy line per risky function
- **BR-3** [Minor] go.yaml.in/yaml/v3 is not a dependency of this module and no step adds it
- **BR-4** [Minor] The define-learn project file still records the retired brain/nous-push storage decision
- **BR-5** [Important] storeHistory.Prefix duplicates memHistory.Prefix verbatim (ARCH-DRY)
- **BR-6** [Important] YAML.Upsert silently resets a word's history when the existing file is unreadable
- **BR-7** [Important] Project file still documents the git/brain/nous-push storage model (PQ-9)
- **BR-8** [Important] No test covers the deps.history to runEditor wiring — the issue's purpose
- **BR-9** [Minor] Timestamp offset preservation is a load-bearing contract with no test
- **BR-10** [Minor] main.go import block has a stray blank line and a third-party import in the stdlib group
- **BR-11** [Minor] Event sort and warnf helper are each duplicated across the two stores
- **BR-12** [Minor] storeHistory.warned is read and written outside the mutex
- **BR-13** [Minor] A construction-time read failure consumes the one-warning budget for the whole session
- **BR-14** [Minor] Slug does not bound filename length, so a very long headword fails to save
- **BR-15** [Minor] Key does no Unicode normalisation, so NFC and NFD spellings are two words
- **BR-16** [Minor] merge always increments Lookups, so Upsert cannot correct a count
- **BR-17** [Minor] TestYAMLDifferentWordsTouchDisjointFiles asserts file count, not disjointness
- **BR-18** [Minor] YAML.Upsert's corrupt-file branch has no test
- **BR-19** [Minor] errFail hand-rolls what errors.New already provides
- **BR-20** [Minor] Found lacks omitempty while Correct has it, within one struct
- **BR-21** [Minor] History is constructed eagerly and loads every day file on every invocation
- **BR-22** [Minor] Plan needs a Revisions entry — four documented deltas the code does not match
