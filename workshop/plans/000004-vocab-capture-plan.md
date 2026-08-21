# Capture on lookup — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every successful lookup builds the deck — one-shot, piped, and interactive alike — so the deck curates itself.

**Architecture:** `#3` already writes on the interactive path via `storeHistory`. This issue moves the *decision to record* out of the history adapter and into one `capture` seam that all three entry paths call, then adds `--forget` and the opt-out. No new storage.

**Milestone:** single-pass. One boundary, one `sdlc close` — no `Mx` tags.

---

## What already exists, and why this is not "wire it up again"

`#3` shipped `storeHistory.Add(line, found)`, which appends an event and upserts
a word when found. That is *capture*, but it lives on the editor's `History` seam
and therefore only fires in the raw loop.

Two paths record nothing today: `define <word>` (one-shot) and `echo w | define`
(line loop). Making them call `History.Add` would be wrong — they have no history
and no editor; they would be borrowing an interface for a side effect.

So the move is **extract, then widen**: the recording decision becomes a
`Capturer` that `storeHistory` also uses, rather than a second implementation
beside it (ARCH-DRY). If this issue ends with two places that decide what a
lookup means, it has failed.

## What `DEFINE_NO_CAPTURE=1` suppresses — and what that costs

The draft said "nothing at all, not even an event". That collides with `#3`:
**persisted history is the event log**. Suppressing events would silently turn
off cross-session history, which the user never asked to lose.

Both readings of the flag are defensible, so the choice is made explicitly:

| reading | effect |
|---|---|
| "don't build a deck from my lookups" | suppress `Upsert` only; events keep flowing, history persists |
| **"don't write in this directory"** ← chosen | suppress **everything**; history falls back to session-only |

Chosen because that is the intent someone has when they opt a tool out of
touching their filesystem, and because a flag that still writes files after you
disabled it is the more surprising of the two failures. The cost — history stops
persisting — is real, so it is stated in `--help`, the README and the atlas
beside the flag, not in a footnote.

**Consequence for the wiring:** with the flag set, `openStore` installs the
session-only `memHistory` rather than `storeHistory`. Nothing downstream changes,
because `#14` consumes the seam.

## Non-goals

- No ordering, scoring or scheduling. `#5` owns "which word next"; `Lookups` is
  recorded here only because it is cheap now and impossible to backfill later.
- No deck browsing or editing beyond `--forget`. `#15`'s `/history` reads.
- No change to the store's file format.

---

## Chunk 1: Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `captureDecision` | `cmd/define/capture.go` | new |
| `decideCapture` | `cmd/define/capture.go` | new |

- **decideCapture(found bool, opt options) captureDecision** — the ONE place that
  answers "does this lookup get recorded, and how far": event only, event plus
  word, or nothing.
  - **One call site: `lookupAndRender`.** Verified against the call graph rather
    than assumed — an earlier draft named `defineOnce`, which is **not** on the
    raw path:

    ```
    defineOnce        ← main.go:144 (one-shot), repl.go:144 (line loop)
      └─ lookupAndRender
    submitLine        ← replraw.go:165 (raw editor) → lookupAndRender  [skips defineOnce]
    ```

    `#14` extracted `lookupAndRender` precisely so the raw path could render in
    cooked mode and play in raw mode, and that extraction is what makes it the
    one function every path shares. Capturing in `defineOnce` would have left the
    interactive path — the only one that captures *today* — recording nothing.
  - **`storeHistory` therefore stops writing, and this is a behaviour move, not a
    refactor.** It becomes recall only: load the event log once at construction,
    keep the in-memory list, append to it on `Add`. Every store write in the
    process then happens in exactly one place.

    **`#3`'s tests must change, and Task 1 Step 3's "tests unchanged" does not
    apply to them.** Naming them, because "a test needing a change is the signal"
    is only true of a behaviour-*preserving* extraction, and these three assert
    the writes that are moving:

    | test | fate |
    |---|---|
    | `TestStoreHistoryRecallsTyposButDoesNotDeckThem` | **moves** to the capturer — the assertion is about what reaches the deck |
    | `TestStoreHistoryPersistsAcrossSessions` | **moves**: it writes via `Add` today; it must write via capture |
    | `TestStoreHistoryDegradesOnWriteFailure` | **moves** with the warn-once rule |

    They are re-targeted, never deleted — `#14`'s close review caught me deleting
    five tests on the claim that a design change had superseded them, and five of
    them still passed. The recall tests stay where they are.
  - **The environment is read exactly once**, at flag parse, into
    `opt.noCapture`. `realDeps` no longer opens history — it cannot, because the
    flag is not parsed yet — so `run` constructs it after `opt` is known, and a
    test that supplies `d.history` is respected unchanged.

- **Capture arity — one lookup, one event.** Stated as an invariant because the
  refactor's failure mode is silent double-counting, and a deck that counts every
  lookup twice is wrong in a way nobody notices until `#5` orders by it. Asserted
  with a counting store across all three entry paths.
  - **Why pure and separate:** the policy has three inputs (did the lookup
    succeed, is capture disabled, is this `--raw`) and is consulted from three
    call sites. Left inline it would be re-derived at each, and they would drift —
    which is exactly how `#2`'s `-raw` came to mean two different things.
  - **DRY rationale:** one answer to "does this lookup count", asserted once.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Capturer` | `cmd/define/capture.go` | new | interface (seam) |
| `storeCapturer` | `cmd/define/capture.go` | new | a `store.Store` |


- **Capturer** — `Capture(word string, found bool, opt options)`. Deliberately returns **no
  error**: capture must never change the outcome of a lookup. A store failure
  warns and the definition still prints, exactly as a missing recording does.
- **storeCapturer** — `AppendEvent` always, `Upsert` when the lookup found
  something. This is the code `storeHistory` currently inlines; it moves here and
  `storeHistory` stops writing entirely — it does not delegate, it does nothing.
  - **The warn-once rule moves with the writes**, and there are now two distinct
    messages with two distinct homes — conflating them is what left the rule
    stranded:

    | message | who owns it | when |
    |---|---|---|
    | `could not record <word>` | `storeCapturer`, once per process | a write fails |
    | `… history is session-only` | `openStore`, once at startup | the store cannot be opened at all |

    Both take their writer from `run`'s `stderr`, passed in at construction —
    neither reaches for `os.Stderr`, so tests capture them. Asserted: N failing
    captures produce exactly one line.
There is deliberately **no null-object `noCapture`**. An earlier draft had both a
`decideCapture` policy *and* a null object, so "is capture off?" had two homes and
would drift the moment one grew a case. `decideCapture` is the only place that
answers it; the environment reaches it as an input, not as a second mechanism.

---

## Chunk 2: Tasks

### Task 1: Extract the policy and the seam

**Files:** create `cmd/define/capture.go`; modify `history_store.go`; test `capture_test.go`

- [x] **Step 1: Write the failing test.** `decideCapture` is a four-case truth
      table over (found, noCapture, raw) — table test, no IO. The two cases that
      are decisions rather than mechanics: a *failed* lookup still records an
      event, so `#14` recalls typos and `#15` can filter them out; and `--raw`
      records nothing, because scripting a dictionary must not mutate a deck.
- [x] **Step 2: Run, expect FAIL**
- [x] **Step 3: Implement.** Move the store writes out of `storeHistory` into
      `storeCapturer`, and **move the three named tests with them** rather than
      editing them in place. The recall tests must stay green untouched — those
      *are* behaviour-preserving, and a change needed there is the real signal.
- [x] **Step 4: Run, expect PASS**
- [x] **Step 5: Commit** — `#4: extract the capture policy behind a seam`

### Task 2: Widen to the one-shot and line paths

**Files:** modify `main.go`, `repl.go`; test `main_test.go`

- [x] **Step 1: Write the failing tests.** Obligation: every entry path records,
      and records **once** — driven through a counting store across one-shot,
      piped and raw. Plus the degradation rule this repo has applied since `#1`:
      a store failure still prints the definition and exits 0.
- [x] **Step 2: Run, expect FAIL**
- [x] **Step 3: Implement.** `lookupAndRender` calls
      `d.capture.Capture(word, found, opt)` after rendering, never before — a lookup that fails to render should not be
      claimed as studied.
- [x] **Step 4: Run, expect PASS**
- [x] **Step 5: Commit** — `#4: capture on every entry path`

### Task 3: `--forget` and the opt-out

**Files:** modify `main.go`, `store/`; test `main_test.go`, `store/`

- [x] **Step 0: Name the adversarial classes for `Forget`**, since it is the
      first operation that *deletes* a file from a path derived from user input:
      - **traversal** — `--forget ../../../etc/passwd`. `Slug` already guarantees
        one safe path element and is fuzzed, but `Forget` must assert it rather
        than inherit it: the test removes nothing outside `words/`.
      - **empty or whitespace key** — a no-op, never a wildcard.
      - **a key whose file is unreadable** — removal must still succeed; refusing
        to delete a corrupt entry is the opposite of useful.
      - **`Forget` must not touch `events/`**, mechanically asserted by comparing
        the directory before and after.
- [x] **Step 1: Write the failing tests.** `Forget` joins the conformance suite,
      so both implementations answer for it. The decisions, as opposed to the
      mechanics: **events are never deleted** (the deck is a working set, the log
      is history, and rewriting the past corrupts every statistic `#8` derives),
      and `--forget` on an absent word exits **non-zero** — succeeding silently
      would hide a typo in the command meant to correct one.
- [x] **Step 2: Run, expect FAIL**
- [x] **Step 3: Implement.** `Forget` joins the `Store` interface, so it lands in
      the shared conformance suite and both implementations must satisfy it.

      **Dispatch:** `-forget <word>` is a string flag checked in `run` *before*
      the `NArg` switch, since it is a mode rather than a lookup:

      ```go
      if *forget != "" {
          return forgetWord(d, *forget, stdout, stderr)   // 0 removed, 1 absent
      }
      switch fs.NArg() { … }
      ```

      `define -forget x y` is a usage error — `NArg() > 0` alongside `-forget` is
      two commands in one line, and silently ignoring one of them is how `-raw`
      came to mean two things in `#2`.
- [x] **Step 4: Run, expect PASS**
- [x] **Step 5: Manual check**

```sh
cd $(mktemp -d)
define sycophantic && ls words/          # one-shot now records
define rizz; ls words/                   # a failed lookup does not
DEFINE_NO_CAPTURE=1 define ephemeral && ls words/   # unchanged
define --forget sycophantic && ls words/ events/    # word gone, log intact
```

- [x] **Step 6: Update `README.md`, `atlas/define.md`** — capture is now the
      tool's default behaviour on every path, which is a bigger user-facing claim
      than `#3` made.
- [x] **Step 7: Commit, then `sdlc close --issue 4 --verified '<evidence>'`**

---

## Risks

- **Silent behaviour change:** `define <word>` has never written a file. It now
  does, on every use. README and atlas must lead with that, and `DEFINE_NO_CAPTURE`
  must be documented beside it rather than in a footnote.
- **The extraction in Task 1 changing behaviour** would show up as `#3`'s tests
  needing edits. That is the signal; do not edit them to fit.
- **`--forget` deleting events** would silently corrupt `#8`'s statistics. The
  deck is a working set; the log is history. Tested explicitly.

---

## Revisions

### 2026-08-21 — the design moved twice during implementation; five review rounds

AGENTS.md §1 requires this section when a plan is revised mid-stream. It was
recommended at three consecutive boundaries and not written; the plan was edited
only to tick checkboxes for the entire window. Recording the deltas now, because
`#15` and `#5` will read this file to learn what capture does.

**1. The capture site is `lookupAndRender`, not `defineOnce`.** Chunk 1 originally
counted three call sites and named `defineOnce`. Both were wrong, and the second
was load-bearing: `defineOnce` is reached by the one-shot and line paths only —
the raw editor's `submitLine` calls `lookupAndRender` directly, which is exactly
what `#14` extracted it for. Capturing in `defineOnce` would have left the
interactive path, *the only one capturing before this issue*, silent. There is
**one** call site.

**2. `storeHistory` stops writing entirely.** The draft had it delegating to the
capturer. It does not delegate — it does nothing, and reads only at construction.
Anything else double-writes on the raw path, and `Capturer.Capture` takes
`(word, found, opt)`, not `(word, found)`: the policy needs the options to see
`-raw` and the opt-out.

**3. `deps` gained three fields the plan never mentioned** — `capture`, `deck`,
and `newStore` — plus a `storeDeps` value and a `withStore` method. `openHistory`
became `openStore`, because the opt-out is a flag-parse-time input and nothing
store-backed can be built in `realDeps` before flags exist.

**4. `DEFINE_NO_CAPTURE` means "write nothing here", including events.** Recorded
in the Spec during round 1 of the gate; the consequence is that history drops to
session-only, because the event log is what persists it.

**What five rounds actually cost, and why:** every round I fixed the instance
named in a finding's title and left the instances enumerated in its body. Round 4
measured 3 of 10; round 5 measured code 10/10 and artifacts 0/7 — the same
substitution one layer out, applied to a markdown file instead of a function.
The rule that generalises, now in `lessons.md`: *an escalated family finding is
closed only when every instance it enumerates is disposed — regardless of which
artifact the instances live in — and where they live in a plan, the closing move
is this section, not a checkbox.*
