---
gate: plan-quality
issue: 4
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-21T10:07:33-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: The opt-out suppresses the event log, which backs persisted history, not just the deck
          detail: |-
            Task 1 routes storeHistory's AppendEvent through the Capturer and has
            decideCapture return "nothing at all, not even an event" for
            DEFINE_NO_CAPTURE=1 and --raw. But newStoreHistory rebuilds Up-arrow recall
            by folding EventLookedUp out of that log (history_store.go:38-47, :22), and
            #15's /history reads it too. So the opt-out silently kills cross-session
            recall, and `define -raw` (a live interactive path) stops persisting it —
            contradicting Task 1 Step 3's own claim that behaviour comes out unchanged.
            State separately what each switch suppresses: the word, or the event.
          family: event-log-vs-deck-policy
          round: 1
        - id: PQ-2
          severity: Important
          title: Two homes decide the same opt-out - decideCapture and the noCapture null object
          detail: |-
            decideCapture is specified to answer DEFINE_NO_CAPTURE and --raw, while
            noCapture is described as "what DEFINE_NO_CAPTURE=1 and --raw install".
            Both cannot hold without writing the check twice (ARCH-DRY), which is the
            failure the plan defines for itself. Also, `lookupEnv` in the decideCapture
            signature is defined nowhere in the plan or the tree, so the env-read
            location is unspecified.
          family: single-policy-home
          round: 1
        - id: PQ-3
          severity: Important
          title: Task 3 never says how --forget reaches the dispatch in run
          detail: |-
            run switches on fs.NArg() at main.go:134-148, where case 1 is
            unconditionally defineOnce and case 0 is the REPL. The plan does not name
            the flag shape, the new branch, or what bare `define --forget` does — as
            written it starts an interactive session. Widening Store also breaks
            failingStore at history_store_test.go:114-121, which implements the
            interface explicitly rather than embedding it.
          family: unspecified-command-dispatch
          round: 1
        - id: PQ-4
          severity: Important
          title: Moving the store writes out of storeHistory strands its warn-once rule
          detail: |-
            storeHistory.warnf reports at most once per session with a
            "(history is session-only)" suffix (history_store.go:84-90), pinned by an
            existing assertion. Capture returns no error, so the plan must say where
            storeCapturer's warn writer comes from and who owns the once-only rule,
            or the extraction duplicates it or drops it.
          family: extraction-strands-behavior
          round: 1
        - id: PQ-5
          severity: Important
          title: No adversarial input class or mechanical guard named for Forget
          detail: |-
            Forget is the first operation here that deletes a user file, yet the plan
            offers prose cases rather than a strategy: a key normalising to empty, an
            unreadable word file, a forget racing an upsert, and the events-untouched
            property, which belongs in storetest.Suite as a property rather than one
            example. Replace the per-task case lists with one strategy line per risky
            function (decideCapture: exhaustive table over its three inputs).
          family: test-strategy-not-enumeration
          round: 1
        - id: PQ-6
          severity: Minor
          title: '"three call sites" is two - defineOnce serves both the one-shot and line paths'
          detail: |-
            main.go:144 and repl.go:144 both call defineOnce, so widening adds one
            invocation site, not two. The design is still right; the count is not.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-7
          severity: Minor
          title: Nothing pins that exactly one Capture fires per lookup
          detail: |-
            After the change defineOnce and storeHistory.Add are independent
            invocation sites over the same store. They do not currently overlap
            (submitLine calls lookupAndRender directly, replraw.go:165), but the
            invariant is unstated and untested, so routing submitLine through
            defineOnce later would silently double-record.
          family: capture-arity-invariant
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-21T10:12:23-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: 'DEFINE_NO_CAPTURE half fully addressed; --raw still suppresses the event log on the live replRaw path with no stated cost, and forces edits to #3''s pinned 3-arg newStoreHistory tests.'
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: Null object removed, but openHistory needs noCapture before run() parses flags, so the env is still read in two places and neither is named.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Dispatch pinned with code and a usage error; the failingStore compile break is mechanical and left as an implementation detail.
          round: 2
        - id: PQ-4
          disposition: not-addressed
          note: Ownership named (storeCapturer), but the warn writer's source is still unstated and the "(history is session-only)" suffix has no decided home.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Task 3 Step 0 now names traversal, empty key, unreadable file, and the events-untouched directory-diff guard.
          round: 2
        - id: PQ-6
          disposition: not-addressed
          note: Plan still says decideCapture is consulted from three call sites; there are two.
          round: 2
        - id: PQ-7
          disposition: not-addressed
          note: No capture-arity invariant stated or tested.
          round: 2
      findings:
        - id: PQ-8
          severity: Minor
          title: Per-case prose survived in all three tasks; state one strategy line per risky function instead
          detail: |-
            This is the 2nd finding in family test-strategy-not-enumeration; PQ-5 fixed the Forget
            instance without applying the rule. Prevalence 3 (Task 1 Step 1, Task 2 Step 1, Task 3
            Step 1). The rule: each risky function gets one line naming its adversarial input class
            and mechanical guard, and the case lists are deleted. decideCapture takes an exhaustive
            table over its three inputs rather than four hand-picked rows; Forget's events-untouched
            obligation is a storetest.Suite property; storeCapturer currently has no strategy line.
          family: test-strategy-not-enumeration
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-21T10:17:59-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The flag's meaning is now chosen explicitly, its cost stated, and openHistory installs memHistory under it.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: Null object deleted; decideCapture is the only home and the env is read once at flag parse into opt.noCapture.
          round: 3
        - id: PQ-4
          disposition: addressed
          note: Two messages, two owners, both writers injected from run's stderr, warn-once asserted.
          round: 3
        - id: PQ-6
          disposition: not-addressed
          note: '"three call sites" became "every entry path goes through defineOnce" — still wrong; only main.go:144 and repl.go:144 call it.'
          round: 3
        - id: PQ-7
          disposition: addressed
          note: Arity is now an explicit invariant with a counting store; the per-path refinement is raised below.
          round: 3
        - id: PQ-8
          disposition: addressed
          note: 'Residual for the close review: decideCapture''s table should be exhaustive over its three inputs, not four rows.'
          round: 3
      findings:
        - id: PQ-9
          severity: Critical
          title: 'The chosen capture site misses the raw interactive path, which is where #3''s capture lives today'
          detail: |-
            This is the 2nd finding in family capture-arity-invariant (prevalence 2: PQ-7's
            double-capture risk, now zero-capture on raw). Do not patch line 72 — state the rule:
            the plan must name, per entry path, the single function where capture happens, and
            assert arity per path rather than in aggregate. Plan line 72 claims submitLine goes
            through defineOnce; it calls lookupAndRender at replraw.go:165 and hist.Add at
            replraw.go:171, and defineOnce has only two callers (main.go:144, repl.go:144). With
            line 77's "storeHistory stops writing", the interactive path records nothing and #3's
            persistence regresses. An aggregate counting store cannot tell zero-on-raw from
            twice-on-piped.
          family: capture-arity-invariant
          round: 3
        - id: PQ-10
          severity: Important
          title: storeHistory is given two incompatible fates, and Task 1 Step 3's "tests unchanged" is unsatisfiable
          detail: |-
            This is the 2nd finding in family extraction-strands-behavior (prevalence 2: PQ-4's
            warn-once, now the durability contract itself). The rule: an extraction must enumerate
            every obligation the source component is contracted for — the tests that pin it, the
            warn-once, the durability guarantee — and say where each lands. Plan line 77 says
            storeHistory becomes a pure reader; line 109 says it delegates; the issue checkbox
            agrees with line 109. Under the reader reading, history_store_test.go:17-33, :37-55,
            :86-99, :155-166 and :133-150 all fail, contradicting "green without edits … do not
            edit them to fit".
          family: extraction-strands-behavior
          round: 3
      blocked: true
---

# Gate ledger — tools#4 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-21T10:07:33-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `event-log-vs-deck-policy` The opt-out suppresses the event log, which backs persisted history, not just the deck
  Task 1 routes storeHistory's AppendEvent through the Capturer and has
  decideCapture return "nothing at all, not even an event" for
  DEFINE_NO_CAPTURE=1 and --raw. But newStoreHistory rebuilds Up-arrow recall
  by folding EventLookedUp out of that log (history_store.go:38-47, :22), and
  #15's /history reads it too. So the opt-out silently kills cross-session
  recall, and `define -raw` (a live interactive path) stops persisting it —
  contradicting Task 1 Step 3's own claim that behaviour comes out unchanged.
  State separately what each switch suppresses: the word, or the event.
- **PQ-2** [Important] `single-policy-home` Two homes decide the same opt-out - decideCapture and the noCapture null object
  decideCapture is specified to answer DEFINE_NO_CAPTURE and --raw, while
  noCapture is described as "what DEFINE_NO_CAPTURE=1 and --raw install".
  Both cannot hold without writing the check twice (ARCH-DRY), which is the
  failure the plan defines for itself. Also, `lookupEnv` in the decideCapture
  signature is defined nowhere in the plan or the tree, so the env-read
  location is unspecified.
- **PQ-3** [Important] `unspecified-command-dispatch` Task 3 never says how --forget reaches the dispatch in run
  run switches on fs.NArg() at main.go:134-148, where case 1 is
  unconditionally defineOnce and case 0 is the REPL. The plan does not name
  the flag shape, the new branch, or what bare `define --forget` does — as
  written it starts an interactive session. Widening Store also breaks
  failingStore at history_store_test.go:114-121, which implements the
  interface explicitly rather than embedding it.
- **PQ-4** [Important] `extraction-strands-behavior` Moving the store writes out of storeHistory strands its warn-once rule
  storeHistory.warnf reports at most once per session with a
  "(history is session-only)" suffix (history_store.go:84-90), pinned by an
  existing assertion. Capture returns no error, so the plan must say where
  storeCapturer's warn writer comes from and who owns the once-only rule,
  or the extraction duplicates it or drops it.
- **PQ-5** [Important] `test-strategy-not-enumeration` No adversarial input class or mechanical guard named for Forget
  Forget is the first operation here that deletes a user file, yet the plan
  offers prose cases rather than a strategy: a key normalising to empty, an
  unreadable word file, a forget racing an upsert, and the events-untouched
  property, which belongs in storetest.Suite as a property rather than one
  example. Replace the per-task case lists with one strategy line per risky
  function (decideCapture: exhaustive table over its three inputs).
- **PQ-6** [Minor] `unbacked-existing-behavior-claim` "three call sites" is two - defineOnce serves both the one-shot and line paths
  main.go:144 and repl.go:144 both call defineOnce, so widening adds one
  invocation site, not two. The design is still right; the count is not.
- **PQ-7** [Minor] `capture-arity-invariant` Nothing pins that exactly one Capture fires per lookup
  After the change defineOnce and storeHistory.Add are independent
  invocation sites over the same store. They do not currently overlap
  (submitLine calls lookupAndRender directly, replraw.go:165), but the
  invariant is unstated and untested, so routing submitLine through
  defineOnce later would silently double-record.

## Round 2 — 2026-08-21T10:12:23-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — not-addressed — DEFINE_NO_CAPTURE half fully addressed; --raw still suppresses the event log on the live replRaw path with no stated cost, and forces edits to #3's pinned 3-arg newStoreHistory tests.
- PQ-2 — not-addressed — Null object removed, but openHistory needs noCapture before run() parses flags, so the env is still read in two places and neither is named.
- PQ-3 — addressed — Dispatch pinned with code and a usage error; the failingStore compile break is mechanical and left as an implementation detail.
- PQ-4 — not-addressed — Ownership named (storeCapturer), but the warn writer's source is still unstated and the "(history is session-only)" suffix has no decided home.
- PQ-5 — addressed — Task 3 Step 0 now names traversal, empty key, unreadable file, and the events-untouched directory-diff guard.
- PQ-6 — not-addressed — Plan still says decideCapture is consulted from three call sites; there are two.
- PQ-7 — not-addressed — No capture-arity invariant stated or tested.

### Raised

- **PQ-8** [Minor] `test-strategy-not-enumeration` Per-case prose survived in all three tasks; state one strategy line per risky function instead
  This is the 2nd finding in family test-strategy-not-enumeration; PQ-5 fixed the Forget
  instance without applying the rule. Prevalence 3 (Task 1 Step 1, Task 2 Step 1, Task 3
  Step 1). The rule: each risky function gets one line naming its adversarial input class
  and mechanical guard, and the case lists are deleted. decideCapture takes an exhaustive
  table over its three inputs rather than four hand-picked rows; Forget's events-untouched
  obligation is a storetest.Suite property; storeCapturer currently has no strategy line.

## Round 3 — 2026-08-21T10:17:59-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — The flag's meaning is now chosen explicitly, its cost stated, and openHistory installs memHistory under it.
- PQ-2 — addressed — Null object deleted; decideCapture is the only home and the env is read once at flag parse into opt.noCapture.
- PQ-4 — addressed — Two messages, two owners, both writers injected from run's stderr, warn-once asserted.
- PQ-6 — not-addressed — "three call sites" became "every entry path goes through defineOnce" — still wrong; only main.go:144 and repl.go:144 call it.
- PQ-7 — addressed — Arity is now an explicit invariant with a counting store; the per-path refinement is raised below.
- PQ-8 — addressed — Residual for the close review: decideCapture's table should be exhaustive over its three inputs, not four rows.

### Raised

- **PQ-9** [Critical] `capture-arity-invariant` The chosen capture site misses the raw interactive path, which is where #3's capture lives today
  This is the 2nd finding in family capture-arity-invariant (prevalence 2: PQ-7's
  double-capture risk, now zero-capture on raw). Do not patch line 72 — state the rule:
  the plan must name, per entry path, the single function where capture happens, and
  assert arity per path rather than in aggregate. Plan line 72 claims submitLine goes
  through defineOnce; it calls lookupAndRender at replraw.go:165 and hist.Add at
  replraw.go:171, and defineOnce has only two callers (main.go:144, repl.go:144). With
  line 77's "storeHistory stops writing", the interactive path records nothing and #3's
  persistence regresses. An aggregate counting store cannot tell zero-on-raw from
  twice-on-piped.
- **PQ-10** [Important] `extraction-strands-behavior` storeHistory is given two incompatible fates, and Task 1 Step 3's "tests unchanged" is unsatisfiable
  This is the 2nd finding in family extraction-strands-behavior (prevalence 2: PQ-4's
  warn-once, now the durability contract itself). The rule: an extraction must enumerate
  every obligation the source component is contracted for — the tests that pin it, the
  warn-once, the durability guarantee — and say where each lands. Plan line 77 says
  storeHistory becomes a pure reader; line 109 says it delegates; the issue checkbox
  agrees with line 109. Under the reader reading, history_store_test.go:17-33, :37-55,
  :86-99, :155-166 and :133-150 all fail, contradicting "green without edits … do not
  edit them to fit".

## Open findings

- **PQ-6** [Minor] `unbacked-existing-behavior-claim` "three call sites" is two - defineOnce serves both the one-shot and line paths
- **PQ-9** [Critical] `capture-arity-invariant` The chosen capture site misses the raw interactive path, which is where #3's capture lives today
- **PQ-10** [Important] `extraction-strands-behavior` storeHistory is given two incompatible fates, and Task 1 Step 3's "tests unchanged" is unsatisfiable
