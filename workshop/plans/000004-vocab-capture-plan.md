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

**Consequence for the wiring:** with the flag set, `openHistory` installs the
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
  word, or nothing. The opt-out arrives as `opt.noCapture`, set once at flag/env
  parse, so the environment is an input to the policy rather than a second
  mechanism beside it.
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


- **Capturer** — `Capture(word string, found bool)`. Deliberately returns **no
  error**: capture must never change the outcome of a lookup. A store failure
  warns and the definition still prints, exactly as a missing recording does.
- **storeCapturer** — `AppendEvent` always, `Upsert` when the lookup found
  something. This is the code `storeHistory` currently inlines; it moves here and
  `storeHistory` delegates.
  - **The warn-once rule moves with the writes.** `#3` put `warned bool` on
    `storeHistory` because that was where writes happened. Once they move, a
    warn-once left behind would either fire per keystroke from the new site or be
    duplicated in both — so the flag lives on `storeCapturer`, and
    `storeHistory` keeps none of its own. Asserted: N failing captures produce
    exactly one line on stderr.
There is deliberately **no null-object `noCapture`**. An earlier draft had both a
`decideCapture` policy *and* a null object, so "is capture off?" had two homes and
would drift the moment one grew a case. `decideCapture` is the only place that
answers it; the environment reaches it as an input, not as a second mechanism.

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

- [ ] **Step 0: Name the adversarial classes for `Forget`**, since it is the
      first operation that *deletes* a file from a path derived from user input:
      - **traversal** — `--forget ../../../etc/passwd`. `Slug` already guarantees
        one safe path element and is fuzzed, but `Forget` must assert it rather
        than inherit it: the test removes nothing outside `words/`.
      - **empty or whitespace key** — a no-op, never a wildcard.
      - **a key whose file is unreadable** — removal must still succeed; refusing
        to delete a corrupt entry is the opposite of useful.
      - **`Forget` must not touch `events/`**, mechanically asserted by comparing
        the directory before and after.
- [ ] **Step 1: Write the failing tests.** `Store.Forget(key)` removes the word
      and is a no-op on a word that is absent (not an error); `--forget <word>`
      exits 0 and prints what it removed; `--forget` on an unknown word says so
      and exits non-zero, since silently succeeding hides a typo; **events are
      NOT deleted** — the deck is a working set, the log is history, and rewriting
      the past to remove a word would corrupt every statistic `#8` derives.
- [ ] **Step 2: Run, expect FAIL**
- [ ] **Step 3: Implement.** `Forget` joins the `Store` interface, so it lands in
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
