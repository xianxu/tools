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

## Open findings

- **PQ-1** [Critical] `event-log-vs-deck-policy` The opt-out suppresses the event log, which backs persisted history, not just the deck
- **PQ-2** [Important] `single-policy-home` Two homes decide the same opt-out - decideCapture and the noCapture null object
- **PQ-3** [Important] `unspecified-command-dispatch` Task 3 never says how --forget reaches the dispatch in run
- **PQ-4** [Important] `extraction-strands-behavior` Moving the store writes out of storeHistory strands its warn-once rule
- **PQ-5** [Important] `test-strategy-not-enumeration` No adversarial input class or mechanical guard named for Forget
- **PQ-6** [Minor] `unbacked-existing-behavior-claim` "three call sites" is two - defineOnce serves both the one-shot and line paths
- **PQ-7** [Minor] `capture-arity-invariant` Nothing pins that exactly one Capture fires per lookup
