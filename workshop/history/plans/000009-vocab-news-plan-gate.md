---
gate: plan-quality
issue: 9
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-26T19:36:17-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Usage is placed in package main but Store.Usages is in package store — that cannot compile
          detail: |-
            The core-concepts table puts Usage in cmd/define/usage.go (package main) and
            Store.Usages/SetUsages in cmd/define/store/store.go. Go forbids importing package
            main, and store/storetest are self-contained today. store.Word and
            store.ReviewEvent are the precedent: state explicitly that Usage moves to the
            store package (or a third one) and that the pure functions take a store-owned type.
          family: type-ownership-across-packages
          round: 1
        - id: PQ-2
          severity: Important
          title: Plan caches post-filter []Usage while the Spec says raw feed items are cached
          detail: |-
            The 2026-08-22 revision states "What is cached is unchanged (raw feed items)".
            SetUsages writes only what survived containsWord, so a later fix to containsWord —
            the plan's own "one place judgment is encoded" — cannot reach anything already on
            disk, and 60-70% of each fetched feed is discarded permanently. It also makes both
            the HTTP source and the fake re-run parseRSS+usagesFrom (ARCH-DRY). Reconcile the
            plan with the Spec or revise the Spec.
          family: cache-record-contract
          round: 1
        - id: PQ-3
          severity: Important
          title: NOAD examples are a bare function, not the seam the Done-when promises
          detail: |-
            Done-when says NOAD examples are available "through the same seam". The plan gives
            entryUsages(Entry) alongside NewsSource.Fetch(ctx, word), so the consumer must call
            the Dictionary, call entryUsages, call Fetch and merge — the exact join the plan's
            own rationale for the Source field says a seam exists to prevent (ARCH-PURPOSE).
            Either widen to a UsageSource returning both, or amend the Done-when and say why.
          family: seam-completeness
          round: 1
        - id: PQ-4
          severity: Important
          title: '"/usage costs one row in the command table" is wrong — commandCtx has no ctx and no news dep'
          detail: |-
            commandCtx (command.go:151) carries neither a context.Context nor a news
            dependency, and run is func(commandCtx, []string) int. Adding both changes
            newCommandCtx's signature and all three call sites — main.go:387, repl.go:331,
            replraw.go:222 — none listed in Task 6's Files block. It is also the first
            REPL command that blocks on the network, and without a threaded ctx it is
            uncancellable despite main.go routing SIGINT through NotifyContext.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-5
          severity: Important
          title: Zero-usage success is a third cache outcome the plan does not model
          detail: |-
            The plan models success and failure only. A successful fetch yielding zero usages
            after filtering, written to disk, is the same "permanent empty answer" the Done-when
            forbids. fetch.go:108-134 already draws this line for audio (ErrNoAudio cached,
            ErrFetchFailed retryable) — name that taxonomy as the reuse (ARCH-DRY). Also decide
            now whether the on-disk record carries a fetched-at, since Usage.At is the item's
            pubDate and schema is expensive to widen later.
          family: cache-outcome-taxonomy
          round: 1
        - id: PQ-6
          severity: Important
          title: containsWord needs phrase matching, which highlight.go already implements — plan names only wordRuns
          detail: |-
            Task 2 Step 1 requires "hot dog" to match as a phrase. phraseGap (highlight.go:95),
            phraseRunsJoin (highlight.go:148) and the longest-match window in highlightSpans
            (highlight.go:112) already do exactly this over wordRuns, normalising through
            store.Key — which also gives the case-insensitivity containsWord wants. Reusing only
            wordRuns re-commits the prior consolidation finding one level up, and divergence is
            user-visible: a headline highlighted green that containsWord rejects.
          family: tokenizer-consolidation
          round: 1
        - id: PQ-7
          severity: Minor
          title: Torn-record test does not match the whole-file atomic write the plan chose
          detail: |-
            writeBytesAtomic (store/yaml.go:246) cannot tear; the existing discipline for
            whole-file records is warn-and-skip the whole file (store/yaml.go:110), while the
            terminator-plus-completeness rule belongs to the append-only day log
            (store/event.go:45). State which shape usage/<slug>.yaml is and test that one.
          family: store-record-format
          round: 1
        - id: PQ-8
          severity: Minor
          title: Compress the prose test-case enumerations to one strategy line per risky function
          detail: |-
            Task 1 Step 2, Task 2 Step 1, Task 3 Step 1 and Task 4 Step 2 each list individual
            cases. They will be rewritten as code within the hour, and enumeration is blind to
            the malformed-input class by construction. FuzzParseRSS is already the model to
            follow: adversarial class plus mechanical guard, one line.
          family: test-plan-enumeration
          round: 1
        - id: PQ-9
          severity: Minor
          title: Widening the Store interface breaks failingStore, which is not in Task 3's file list
          detail: |-
            failingStore (history_store_test.go:67-77) implements the method set explicitly
            rather than embedding store.Store, so it needs the two new methods. Compiler-caught,
            but add the file to Task 3's Files block.
          family: interface-widening-consumers
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-26T19:40:30-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Split into store.NewsItem (package store, cached) and Usage (package main, derived on read) — compiles, matches the store.Word precedent.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Only raw items are cached now, so improving containsWord reaches every already-cached word with no re-fetch.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: UsageSource with a bothSources composite is the seam; the news-fails-degrades-to-NOAD property is tested.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Claim retracted and /usage dropped with the reason recorded; the issue file's M3 row still names /usage and should be tidied to match.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Three outcomes, FetchedAt plus a TTL, and a stale-copy fallback on failed re-fetch, each with a test row and a mutation check.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Delegates to highlightSpans, pinned by an agreement test and a mutation check against a naive strings.Contains.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: Corrected to the whole-file warn-and-skip discipline (yaml.go:110), not the day log's terminator rule.
          round: 2
        - id: PQ-8
          disposition: addressed
          note: Compressed to one strategy line per risky function, with the fuzz target carrying the malformed class.
          round: 2
        - id: PQ-9
          disposition: addressed
          note: failingStore is now listed in Task 3's Files block with the reason.
          round: 2
      blocked: false
content_hash: ca02df09bf60ceb60089d3daf3a6bd0035bcce5c0410049c66d42cc2b0eb0a6b
---

# Gate ledger — tools#9 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-26T19:36:17-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `type-ownership-across-packages` Usage is placed in package main but Store.Usages is in package store — that cannot compile
  The core-concepts table puts Usage in cmd/define/usage.go (package main) and
  Store.Usages/SetUsages in cmd/define/store/store.go. Go forbids importing package
  main, and store/storetest are self-contained today. store.Word and
  store.ReviewEvent are the precedent: state explicitly that Usage moves to the
  store package (or a third one) and that the pure functions take a store-owned type.
- **PQ-2** [Important] `cache-record-contract` Plan caches post-filter []Usage while the Spec says raw feed items are cached
  The 2026-08-22 revision states "What is cached is unchanged (raw feed items)".
  SetUsages writes only what survived containsWord, so a later fix to containsWord —
  the plan's own "one place judgment is encoded" — cannot reach anything already on
  disk, and 60-70% of each fetched feed is discarded permanently. It also makes both
  the HTTP source and the fake re-run parseRSS+usagesFrom (ARCH-DRY). Reconcile the
  plan with the Spec or revise the Spec.
- **PQ-3** [Important] `seam-completeness` NOAD examples are a bare function, not the seam the Done-when promises
  Done-when says NOAD examples are available "through the same seam". The plan gives
  entryUsages(Entry) alongside NewsSource.Fetch(ctx, word), so the consumer must call
  the Dictionary, call entryUsages, call Fetch and merge — the exact join the plan's
  own rationale for the Source field says a seam exists to prevent (ARCH-PURPOSE).
  Either widen to a UsageSource returning both, or amend the Done-when and say why.
- **PQ-4** [Important] `unbacked-existing-behavior-claim` "/usage costs one row in the command table" is wrong — commandCtx has no ctx and no news dep
  commandCtx (command.go:151) carries neither a context.Context nor a news
  dependency, and run is func(commandCtx, []string) int. Adding both changes
  newCommandCtx's signature and all three call sites — main.go:387, repl.go:331,
  replraw.go:222 — none listed in Task 6's Files block. It is also the first
  REPL command that blocks on the network, and without a threaded ctx it is
  uncancellable despite main.go routing SIGINT through NotifyContext.
- **PQ-5** [Important] `cache-outcome-taxonomy` Zero-usage success is a third cache outcome the plan does not model
  The plan models success and failure only. A successful fetch yielding zero usages
  after filtering, written to disk, is the same "permanent empty answer" the Done-when
  forbids. fetch.go:108-134 already draws this line for audio (ErrNoAudio cached,
  ErrFetchFailed retryable) — name that taxonomy as the reuse (ARCH-DRY). Also decide
  now whether the on-disk record carries a fetched-at, since Usage.At is the item's
  pubDate and schema is expensive to widen later.
- **PQ-6** [Important] `tokenizer-consolidation` containsWord needs phrase matching, which highlight.go already implements — plan names only wordRuns
  Task 2 Step 1 requires "hot dog" to match as a phrase. phraseGap (highlight.go:95),
  phraseRunsJoin (highlight.go:148) and the longest-match window in highlightSpans
  (highlight.go:112) already do exactly this over wordRuns, normalising through
  store.Key — which also gives the case-insensitivity containsWord wants. Reusing only
  wordRuns re-commits the prior consolidation finding one level up, and divergence is
  user-visible: a headline highlighted green that containsWord rejects.
- **PQ-7** [Minor] `store-record-format` Torn-record test does not match the whole-file atomic write the plan chose
  writeBytesAtomic (store/yaml.go:246) cannot tear; the existing discipline for
  whole-file records is warn-and-skip the whole file (store/yaml.go:110), while the
  terminator-plus-completeness rule belongs to the append-only day log
  (store/event.go:45). State which shape usage/<slug>.yaml is and test that one.
- **PQ-8** [Minor] `test-plan-enumeration` Compress the prose test-case enumerations to one strategy line per risky function
  Task 1 Step 2, Task 2 Step 1, Task 3 Step 1 and Task 4 Step 2 each list individual
  cases. They will be rewritten as code within the hour, and enumeration is blind to
  the malformed-input class by construction. FuzzParseRSS is already the model to
  follow: adversarial class plus mechanical guard, one line.
- **PQ-9** [Minor] `interface-widening-consumers` Widening the Store interface breaks failingStore, which is not in Task 3's file list
  failingStore (history_store_test.go:67-77) implements the method set explicitly
  rather than embedding store.Store, so it needs the two new methods. Compiler-caught,
  but add the file to Task 3's Files block.

## Round 2 — 2026-08-26T19:40:30-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Split into store.NewsItem (package store, cached) and Usage (package main, derived on read) — compiles, matches the store.Word precedent.
- PQ-2 — addressed — Only raw items are cached now, so improving containsWord reaches every already-cached word with no re-fetch.
- PQ-3 — addressed — UsageSource with a bothSources composite is the seam; the news-fails-degrades-to-NOAD property is tested.
- PQ-4 — addressed — Claim retracted and /usage dropped with the reason recorded; the issue file's M3 row still names /usage and should be tidied to match.
- PQ-5 — addressed — Three outcomes, FetchedAt plus a TTL, and a stale-copy fallback on failed re-fetch, each with a test row and a mutation check.
- PQ-6 — addressed — Delegates to highlightSpans, pinned by an agreement test and a mutation check against a naive strings.Contains.
- PQ-7 — addressed — Corrected to the whole-file warn-and-skip discipline (yaml.go:110), not the day log's terminator rule.
- PQ-8 — addressed — Compressed to one strategy line per risky function, with the fuzz target carrying the malformed class.
- PQ-9 — addressed — failingStore is now listed in Task 3's Files block with the reason.

## Open findings

(none — every finding has been disposed)
