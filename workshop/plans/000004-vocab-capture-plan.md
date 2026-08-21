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

- **decideCapture(found bool, opt options, env lookupEnv) captureDecision** — the
  whole policy in one pure function: record the event, record the word, or do
  nothing.
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
| `noCapture` | `cmd/define/capture.go` | new | nothing — the opt-out |

- **Capturer** — `Capture(word string, found bool)`. Deliberately returns **no
  error**: capture must never change the outcome of a lookup. A store failure
  warns and the definition still prints, exactly as a missing recording does.
- **storeCapturer** — `AppendEvent` always, `Upsert` when found. This is the code
  `storeHistory` currently inlines; it moves here and `storeHistory` calls it.
- **noCapture** — what `DEFINE_NO_CAPTURE=1` and `--raw` install. A null object
  rather than a nil check at three call sites.

---

## Chunk 2: Tasks

### Task 1: Extract the policy and the seam

**Files:** create `cmd/define/capture.go`; modify `history_store.go`; test `capture_test.go`

- [ ] **Step 1: Write the failing tests.** `decideCapture` obligations: found →
      event + word; not found → event only (so `#14`'s recall keeps typos and
      `#15`'s `/history` can filter); `DEFINE_NO_CAPTURE=1` → nothing at all, not
      even an event; `--raw` → nothing, because it is the scripting form and
      scripting a dictionary should not mutate a deck.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement, and rewrite `storeHistory.Add` to call it** — the
      existing behaviour must come out unchanged, proven by `#3`'s tests staying
      green **without edits**. A test needing a change here means the extraction
      was not behaviour-preserving.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#4: extract the capture policy behind a seam`

### Task 2: Widen to the one-shot and line paths

**Files:** modify `main.go`, `repl.go`; test `main_test.go`

- [ ] **Step 1: Write the failing tests.** `define <word>` records a found word;
      an unknown word records the event but no deck entry; `echo w | define`
      records; a **store failure still prints the definition and exits 0**.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** `defineOnce` calls `d.capture.Capture(word, found)`
      after rendering, never before — a lookup that fails to render should not be
      claimed as studied.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Commit** — `#4: capture on every entry path`

### Task 3: `--forget` and the opt-out

**Files:** modify `main.go`, `store/`; test `main_test.go`, `store/`

- [ ] **Step 1: Write the failing tests.** `Store.Forget(key)` removes the word
      and is a no-op on a word that is absent (not an error); `--forget <word>`
      exits 0 and prints what it removed; `--forget` on an unknown word says so
      and exits non-zero, since silently succeeding hides a typo; **events are
      NOT deleted** — the deck is a working set, the log is history, and rewriting
      the past to remove a word would corrupt every statistic `#8` derives.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** `Forget` joins the `Store` interface, so it lands in
      the shared conformance suite and both implementations must satisfy it.
- [ ] **Step 4: Run, expect PASS**
- [ ] **Step 5: Manual check**

```sh
cd $(mktemp -d)
define sycophantic && ls words/          # one-shot now records
define rizz; ls words/                   # a failed lookup does not
DEFINE_NO_CAPTURE=1 define ephemeral && ls words/   # unchanged
define --forget sycophantic && ls words/ events/    # word gone, log intact
```

- [ ] **Step 6: Update `README.md`, `atlas/define.md`** — capture is now the
      tool's default behaviour on every path, which is a bigger user-facing claim
      than `#3` made.
- [ ] **Step 7: Commit, then `sdlc close --issue 4 --verified '<evidence>'`**

---

## Risks

- **Silent behaviour change:** `define <word>` has never written a file. It now
  does, on every use. README and atlas must lead with that, and `DEFINE_NO_CAPTURE`
  must be documented beside it rather than in a footnote.
- **The extraction in Task 1 changing behaviour** would show up as `#3`'s tests
  needing edits. That is the signal; do not edit them to fit.
- **`--forget` deleting events** would silently corrupt `#8`'s statistics. The
  deck is a working set; the log is history. Tested explicitly.
