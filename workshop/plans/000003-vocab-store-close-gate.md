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
    - "n": 3
      timestamp: "2026-08-20T21:40:22-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: withdrawn
          note: Overtaken by the shipped code — deps.history plus a nil default at replraw.go:62-64; no rig panics and the full suite is green.
          round: 3
        - id: BR-2
          disposition: withdrawn
          note: A plan-authoring style nit about pre-images of code that now exists; no value left at a code boundary.
          round: 3
        - id: BR-3
          disposition: addressed
          note: go.mod and go.sum pin go.yaml.in/yaml/v3 v3.0.5; go build and go test run offline.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Same fix as BR-7 — define-learn.md now records the cwd-only model.
          round: 3
        - id: BR-5
          disposition: addressed
          note: prefixMatch at history.go:44 is now the single definition, called by memHistory.Prefix and storeHistory.Prefix.
          round: 3
        - id: BR-6
          disposition: addressed
          note: 'Verified by probe — the branch now emits "define: overwriting unreadable alpha.yaml: ..." before the reset.'
          round: 3
        - id: BR-7
          disposition: addressed
          note: define-learn.md:37 and :54-60 rewritten to the cwd-only model plus the rate-not-impossibility claim.
          round: 3
        - id: BR-8
          disposition: addressed
          note: Both mutations re-verified independently — reverting replraw.go:61-64 and adding "&& e.Found" each fail exactly one new test.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: No FixedZone case anywhere in the tree; every suite timestamp is still time.UTC.
          round: 3
        - id: BR-10
          disposition: not-addressed
          note: main.go:3-16 still has the stray blank line after "context" and the store import inside the stdlib group.
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: mem.go:83 and yaml.go:144 still carry the identical event sort; two warnf helpers still repeat the prefix literal.
          round: 3
        - id: BR-12
          disposition: not-addressed
          note: history_store.go:84-90 still reads and writes h.warned outside h.mu.
          round: 3
        - id: BR-13
          disposition: not-addressed
          note: One warned flag still covers both the construction-time read failure and every later write failure.
          round: 3
        - id: BR-14
          disposition: not-addressed
          note: No length bound in Slug; word.go still has no truncate-plus-hash path.
          round: 3
        - id: BR-15
          disposition: not-addressed
          note: Key still does no Unicode normalisation.
          round: 3
        - id: BR-16
          disposition: not-addressed
          note: store.go:11-12 documents "Lookups accumulates" but not that Upsert can never set an exact count.
          round: 3
        - id: BR-17
          disposition: not-addressed
          note: yaml_test.go:99-112 still asserts only the file count, not that alpha.yaml was untouched.
          round: 3
        - id: BR-18
          disposition: not-addressed
          note: Still no test for the Upsert corrupt-file branch — the very branch round 2 changed remains invisible to the suite.
          round: 3
        - id: BR-19
          disposition: not-addressed
          note: history_store_test.go:124-128 still hand-rolls failErr instead of errors.New.
          round: 3
        - id: BR-20
          disposition: not-addressed
          note: event.go:20 Found still lacks omitempty while Correct at :21 has it.
          round: 3
        - id: BR-21
          disposition: not-addressed
          note: openHistory is still eager and Events(time.Time{}) still parses every day file on every invocation.
          round: 3
        - id: BR-22
          disposition: not-addressed
          note: No "## Revisions" section exists in the plan (grep confirms); all four deltas re-verified live. The PQ-9 note in this finding is now stale in the other direction — the plan-gate ledger should dispose PQ-9 as addressed.
          round: 3
      findings:
        - id: BR-23
          severity: Important
          title: The event log has no torn-record recovery, and the atlas claims atomic writes without scoping it to words
          detail: |-
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
          round: 3
        - id: BR-24
          severity: Minor
          title: words/*.yaml is written 0600 while events/*.yaml is 0644
          detail: |-
            yaml.go:161-193 inherits 0600 from os.CreateTemp and the rename preserves it, while
            AppendEvent at yaml.go:106 opens with 0644. Verified by probe. Two files written by
            one store with two permission stories.
          round: 3
        - id: BR-25
          severity: Minor
          title: Store states no thread-safety contract and the two implementations differ (ARCH-MOCK)
          detail: |-
            store.go:10 — Mem guards every method with a mutex; YAML has none and Upsert is a
            non-atomic read-modify-write. The conformance suite cannot catch this because the
            fake is stronger than the real one, which is the fake-diverges-from-real gap the
            suite exists to close. Same shape, lower stakes: Mem.Deck returns a non-nil empty
            slice where YAML.Deck returns nil. Harmless today since the CLI is single-goroutine;
            a one-line interface doc comment settles the intent before issues 4 and 5 consume it.
          round: 3
        - id: BR-26
          severity: Minor
          title: Stale comment at replraw.go:69 still says the store is future work
          detail: |-
            "Querying twice doubled the work the History seam will do once #3 backs it with a
            store" — issue 3 now does, three lines above at replraw.go:61.
          round: 3
        - id: BR-27
          severity: Minor
          title: The issue Log records no boundary-review outcome for round 2
          detail: |-
            workshop/issues/000003-vocab-store.md:115-134 ends at the implementation notes; there
            is no entry for the round-2 close review or the four Important fixes it produced.
            AGENTS.md section 3 makes logging the review outcome part of crossing the boundary.
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-20T21:48:20-07:00"
      agent: claude
      dispose:
        - id: BR-9
          disposition: not-addressed
          note: No FixedZone anywhere in the tree; every storetest.Suite timestamp is still time.UTC.
          round: 4
        - id: BR-10
          disposition: not-addressed
          note: main.go:3-16 still has the stray blank line after "context" and the store import inside the stdlib group.
          round: 4
        - id: BR-11
          disposition: not-addressed
          note: 'mem.go:83 and yaml.go:145 still carry the byte-identical event sort; history_store.go:84 and yaml.go:149 still repeat the "define: " prefix literal.'
          round: 4
        - id: BR-12
          disposition: not-addressed
          note: history_store.go:84-90 still reads and writes h.warned outside h.mu.
          round: 4
        - id: BR-13
          disposition: not-addressed
          note: One warned flag still covers both the construction-time read failure and every later write failure; the "(history is session-only)" suffix is still appended to "could not save word".
          round: 4
        - id: BR-14
          disposition: not-addressed
          note: No length bound in Slug; word.go has no truncate-plus-hash path.
          round: 4
        - id: BR-15
          disposition: not-addressed
          note: Key at word.go:29-31 still does no Unicode normalisation.
          round: 4
        - id: BR-16
          disposition: not-addressed
          note: store.go:11-12 documents "Lookups accumulates" but not that Upsert can never set an exact count.
          round: 4
        - id: BR-17
          disposition: not-addressed
          note: TestYAMLDifferentWordsTouchDisjointFiles still asserts only the file count, not that alpha.yaml was untouched.
          round: 4
        - id: BR-18
          disposition: not-addressed
          note: Still no test for Upsert's "overwriting unreadable" branch (yaml.go:44-53) — the branch round 2 changed remains invisible to the suite.
          round: 4
        - id: BR-19
          disposition: not-addressed
          note: history_store_test.go:124-128 still hand-rolls failErr instead of errors.New.
          round: 4
        - id: BR-20
          disposition: not-addressed
          note: event.go:20 Found still lacks omitempty while Correct at :21 has it.
          round: 4
        - id: BR-21
          disposition: not-addressed
          note: realDeps still builds openHistory eagerly and newStoreHistory still calls Events(time.Time{}), parsing every day file on every invocation.
          round: 4
        - id: BR-22
          disposition: not-addressed
          note: No "## Revisions" section exists in the plan (grep confirms); all four deltas re-verified live, plus a fifth — the plan states atomic writes as a blanket rule the shipped design deliberately splits.
          round: 4
        - id: BR-23
          disposition: addressed
          note: Atlas is now scoped to word writes and the record-level recovery landed (parseDay/splitRecords/complete). The fix itself ships a duplication defect and an untested branch, raised separately below rather than re-raised here.
          round: 4
        - id: BR-24
          disposition: not-addressed
          note: writeAtomic still inherits 0600 from os.CreateTemp while AppendEvent at yaml.go:102 opens 0644.
          round: 4
        - id: BR-25
          disposition: not-addressed
          note: store.go:5-20 still states no thread-safety contract; Mem is mutex-guarded and YAML is not.
          round: 4
        - id: BR-26
          disposition: not-addressed
          note: 'replraw.go:70-71 still reads "the History seam will do once #3 backs it with a store".'
          round: 4
        - id: BR-27
          disposition: not-addressed
          note: The issue Log still ends at the implementation notes; no entry records the round-2 or round-3 close-review outcomes.
          round: 4
      findings:
        - id: BR-28
          severity: Critical
          title: parseDay double-counts every whole record whenever the torn-record recovery path runs
          detail: |-
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
          round: 4
        - id: BR-29
          severity: Important
          title: A truncation inside the timestamp passes complete() and is admitted as a real event with a fabricated date
          detail: |-
            event.go:31-33 tests for field presence, not integrity. A record cut at "at: 2026-08-2"
            parses as 2026-08-02 and is returned as a whole event — an 18-day-displaced record invented
            from a fragment; "at: 2026-08-20" likewise becomes midnight. This is the exact failure mode
            workshop/lessons.md:120 was written about, one truncation point over, and atlas/define.md:233
            states completeness IS what distinguishes a whole record from a fragment. Cheap fix: the
            reliable torn-tail signal is the one AppendEvent already guarantees — every complete record
            ends with a newline. In parseDay, if the file does not end in "\n", the final record is torn
            by construction: drop and count it, then apply the existing checks to the rest.
          round: 4
        - id: BR-30
          severity: Important
          title: The torn-record recovery branch added this window has no test that reaches it (ARCH-PURE)
          detail: |-
            yaml_test.go:121-148 appends "- word: thi", which leaves the day file VALID YAML — confirmed
            yaml.Unmarshal returns nil on that exact input — so the test never enters the err != nil
            branch and splitRecords is dead code as far as the suite is concerned. Both defects above
            live in that unreached branch. Compounding it, parseDay/splitRecords/complete are pure and
            in-package but yaml_test.go is package store_test and cannot see them, while word_test.go
            (package store) tests only Key and Slug — so the pure core buys no test leverage. A table
            test in package store driving parseDay over each interesting truncation, asserting both the
            events and the torn count, is about ten lines and catches both.
          round: 4
      blocked: true
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

## Round 3 — 2026-08-20T21:40:22-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — withdrawn — Overtaken by the shipped code — deps.history plus a nil default at replraw.go:62-64; no rig panics and the full suite is green.
- BR-2 — withdrawn — A plan-authoring style nit about pre-images of code that now exists; no value left at a code boundary.
- BR-3 — addressed — go.mod and go.sum pin go.yaml.in/yaml/v3 v3.0.5; go build and go test run offline.
- BR-4 — addressed — Same fix as BR-7 — define-learn.md now records the cwd-only model.
- BR-5 — addressed — prefixMatch at history.go:44 is now the single definition, called by memHistory.Prefix and storeHistory.Prefix.
- BR-6 — addressed — Verified by probe — the branch now emits "define: overwriting unreadable alpha.yaml: ..." before the reset.
- BR-7 — addressed — define-learn.md:37 and :54-60 rewritten to the cwd-only model plus the rate-not-impossibility claim.
- BR-8 — addressed — Both mutations re-verified independently — reverting replraw.go:61-64 and adding "&& e.Found" each fail exactly one new test.
- BR-9 — not-addressed — No FixedZone case anywhere in the tree; every suite timestamp is still time.UTC.
- BR-10 — not-addressed — main.go:3-16 still has the stray blank line after "context" and the store import inside the stdlib group.
- BR-11 — not-addressed — mem.go:83 and yaml.go:144 still carry the identical event sort; two warnf helpers still repeat the prefix literal.
- BR-12 — not-addressed — history_store.go:84-90 still reads and writes h.warned outside h.mu.
- BR-13 — not-addressed — One warned flag still covers both the construction-time read failure and every later write failure.
- BR-14 — not-addressed — No length bound in Slug; word.go still has no truncate-plus-hash path.
- BR-15 — not-addressed — Key still does no Unicode normalisation.
- BR-16 — not-addressed — store.go:11-12 documents "Lookups accumulates" but not that Upsert can never set an exact count.
- BR-17 — not-addressed — yaml_test.go:99-112 still asserts only the file count, not that alpha.yaml was untouched.
- BR-18 — not-addressed — Still no test for the Upsert corrupt-file branch — the very branch round 2 changed remains invisible to the suite.
- BR-19 — not-addressed — history_store_test.go:124-128 still hand-rolls failErr instead of errors.New.
- BR-20 — not-addressed — event.go:20 Found still lacks omitempty while Correct at :21 has it.
- BR-21 — not-addressed — openHistory is still eager and Events(time.Time{}) still parses every day file on every invocation.
- BR-22 — not-addressed — No "## Revisions" section exists in the plan (grep confirms); all four deltas re-verified live. The PQ-9 note in this finding is now stale in the other direction — the plan-gate ledger should dispose PQ-9 as addressed.

### Raised

- **BR-23** [Important] The event log has no torn-record recovery, and the atlas claims atomic writes without scoping it to words
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
- **BR-24** [Minor] words/*.yaml is written 0600 while events/*.yaml is 0644
  yaml.go:161-193 inherits 0600 from os.CreateTemp and the rename preserves it, while
  AppendEvent at yaml.go:106 opens with 0644. Verified by probe. Two files written by
  one store with two permission stories.
- **BR-25** [Minor] Store states no thread-safety contract and the two implementations differ (ARCH-MOCK)
  store.go:10 — Mem guards every method with a mutex; YAML has none and Upsert is a
  non-atomic read-modify-write. The conformance suite cannot catch this because the
  fake is stronger than the real one, which is the fake-diverges-from-real gap the
  suite exists to close. Same shape, lower stakes: Mem.Deck returns a non-nil empty
  slice where YAML.Deck returns nil. Harmless today since the CLI is single-goroutine;
  a one-line interface doc comment settles the intent before issues 4 and 5 consume it.
- **BR-26** [Minor] Stale comment at replraw.go:69 still says the store is future work
  "Querying twice doubled the work the History seam will do once #3 backs it with a
  store" — issue 3 now does, three lines above at replraw.go:61.
- **BR-27** [Minor] The issue Log records no boundary-review outcome for round 2
  workshop/issues/000003-vocab-store.md:115-134 ends at the implementation notes; there
  is no entry for the round-2 close review or the four Important fixes it produced.
  AGENTS.md section 3 makes logging the review outcome part of crossing the boundary.

## Round 4 — 2026-08-20T21:48:20-07:00 (claude) — BLOCKED

### Disposed

- BR-9 — not-addressed — No FixedZone anywhere in the tree; every storetest.Suite timestamp is still time.UTC.
- BR-10 — not-addressed — main.go:3-16 still has the stray blank line after "context" and the store import inside the stdlib group.
- BR-11 — not-addressed — mem.go:83 and yaml.go:145 still carry the byte-identical event sort; history_store.go:84 and yaml.go:149 still repeat the "define: " prefix literal.
- BR-12 — not-addressed — history_store.go:84-90 still reads and writes h.warned outside h.mu.
- BR-13 — not-addressed — One warned flag still covers both the construction-time read failure and every later write failure; the "(history is session-only)" suffix is still appended to "could not save word".
- BR-14 — not-addressed — No length bound in Slug; word.go has no truncate-plus-hash path.
- BR-15 — not-addressed — Key at word.go:29-31 still does no Unicode normalisation.
- BR-16 — not-addressed — store.go:11-12 documents "Lookups accumulates" but not that Upsert can never set an exact count.
- BR-17 — not-addressed — TestYAMLDifferentWordsTouchDisjointFiles still asserts only the file count, not that alpha.yaml was untouched.
- BR-18 — not-addressed — Still no test for Upsert's "overwriting unreadable" branch (yaml.go:44-53) — the branch round 2 changed remains invisible to the suite.
- BR-19 — not-addressed — history_store_test.go:124-128 still hand-rolls failErr instead of errors.New.
- BR-20 — not-addressed — event.go:20 Found still lacks omitempty while Correct at :21 has it.
- BR-21 — not-addressed — realDeps still builds openHistory eagerly and newStoreHistory still calls Events(time.Time{}), parsing every day file on every invocation.
- BR-22 — not-addressed — No "## Revisions" section exists in the plan (grep confirms); all four deltas re-verified live, plus a fifth — the plan states atomic writes as a blanket rule the shipped design deliberately splits.
- BR-23 — addressed — Atlas is now scoped to word writes and the record-level recovery landed (parseDay/splitRecords/complete). The fix itself ships a duplication defect and an untested branch, raised separately below rather than re-raised here.
- BR-24 — not-addressed — writeAtomic still inherits 0600 from os.CreateTemp while AppendEvent at yaml.go:102 opens 0644.
- BR-25 — not-addressed — store.go:5-20 still states no thread-safety contract; Mem is mutex-guarded and YAML is not.
- BR-26 — not-addressed — replraw.go:70-71 still reads "the History seam will do once #3 backs it with a store".
- BR-27 — not-addressed — The issue Log still ends at the implementation notes; no entry records the round-2 or round-3 close-review outcomes.

### Raised

- **BR-28** [Critical] parseDay double-counts every whole record whenever the torn-record recovery path runs
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
- **BR-29** [Important] A truncation inside the timestamp passes complete() and is admitted as a real event with a fabricated date
  event.go:31-33 tests for field presence, not integrity. A record cut at "at: 2026-08-2"
  parses as 2026-08-02 and is returned as a whole event — an 18-day-displaced record invented
  from a fragment; "at: 2026-08-20" likewise becomes midnight. This is the exact failure mode
  workshop/lessons.md:120 was written about, one truncation point over, and atlas/define.md:233
  states completeness IS what distinguishes a whole record from a fragment. Cheap fix: the
  reliable torn-tail signal is the one AppendEvent already guarantees — every complete record
  ends with a newline. In parseDay, if the file does not end in "\n", the final record is torn
  by construction: drop and count it, then apply the existing checks to the rest.
- **BR-30** [Important] The torn-record recovery branch added this window has no test that reaches it (ARCH-PURE)
  yaml_test.go:121-148 appends "- word: thi", which leaves the day file VALID YAML — confirmed
  yaml.Unmarshal returns nil on that exact input — so the test never enters the err != nil
  branch and splitRecords is dead code as far as the suite is concerned. Both defects above
  live in that unreached branch. Compounding it, parseDay/splitRecords/complete are pure and
  in-package but yaml_test.go is package store_test and cannot see them, while word_test.go
  (package store) tests only Key and Slug — so the pure core buys no test leverage. A table
  test in package store driving parseDay over each interesting truncation, asserting both the
  events and the torn count, is about ten lines and catches both.

## Open findings

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
- **BR-24** [Minor] words/*.yaml is written 0600 while events/*.yaml is 0644
- **BR-25** [Minor] Store states no thread-safety contract and the two implementations differ (ARCH-MOCK)
- **BR-26** [Minor] Stale comment at replraw.go:69 still says the store is future work
- **BR-27** [Minor] The issue Log records no boundary-review outcome for round 2
- **BR-28** [Critical] parseDay double-counts every whole record whenever the torn-record recovery path runs
- **BR-29** [Important] A truncation inside the timestamp passes complete() and is admitted as a real event with a fabricated date
- **BR-30** [Important] The torn-record recovery branch added this window has no test that reaches it (ARCH-PURE)
