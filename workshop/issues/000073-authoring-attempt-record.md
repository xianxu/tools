---
id: 000073
status: open
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-17
estimate_hours:
---

# define: record a failed authoring attempt and stop re-asking for it

## Problem

`--harvest` re-pays for every failure it has ever had, on every run, forever.

`runAuthoring` (`harvest.go:293`) rejects a word on four paths — the stem does not
contain the word (`harvest.go:362`), the entail judge refuses it
(`harvest.go:382`), every distractor is vetoed (`harvest.go:438`), or the write
fails. Each one does the same thing: print to `errOut` and `continue`. The word
ends the run **with no item**, so the skip at `harvest.go:342` (`len(existing) >
0`) does not fire on the next run, and authoring re-asks with a byte-identical
prompt. Nothing anywhere records that the attempt happened.

So a word that cannot be authored costs 2 model calls (author + entail) every
single run, indefinitely, and a word whose distractors all get vetoed costs up to
5. Across a deck that accumulates unauthorable words, this is the dominant cost
of `--harvest` and it is invisible: the only trace is a line on stderr, and the
background harvester passes `io.Discard` for both writers (`background.go:224`).

The operator's decision (2026-09-17): **fail closed.** A word whose authoring
failed should simply have no cloze question, and should not be asked again unless
a policy explicitly says to. `--harvest`'s spend should not be a function of how
many times it has been run.

## Spec

### The record

A per-word, per-language surface, sibling to `WordFacts` and `Items`:

```go
type Attempt struct {
    At       time.Time
    Form     Form          // cloze today; #13's sentence form later
    State    AttemptState  // closed enum, ParseAttemptState following ParseForm
    Provider string        // llm.ModelSelection.Provider
    Model    string        // llm.ModelSelection.ID
    Reason   string        // the judge's clause, today printed and dropped
}
```

**Not folded into `WordFacts`**, whose doc says its fields "are one judgement
about the word and they expire together (never)" (`store/item.go:10`). An attempt
record is exactly the thing that expires.

A bounded list per word, like `Items`: every other per-word surface here states a
cap (`ItemCap = 4`, `store/item.go:210`) and this one needs the same, or a word
retried under ten model versions grows an unbounded file. Newest kept.

`Reason` is free model text being persisted for the first time on this surface, so
it joins the `oneLine` class (`store/item.go:157`) — whitespace collapsed, control
runes dropped — and the read-side rule in `store.go:67`. It reaches a terminal.

### The states, and what each one's retry trigger actually is

```
malformed   — the stem did not contain the word      (harvest.go:362)
notEntailed — the word's meaning did no work         (harvest.go:382)
glossed     — the sentence defined the word          (harvest.go:382)
unnamed     — no real person, place or institution   (harvest.go:382)
allVetoed   — every distractor candidate was vetoed  (harvest.go:438)
```

**All five are model-dependent, including `allVetoed`.** This corrects an earlier
reading. `pickDistractors`' last tier accepts everything —
`{tierAboveBand, func(c bandedWord) bool { return true }}` (`harvest_item.go:259`)
— so in production it always returns at least one candidate, and `len(kept) == 0`
is reachable *only* because the veto (a model call, against the stem text) refused
all of them. A different model, or a different stem, is what changes it. So
`allVetoed` is admitted by the same retry policy as the other four, and there is
no pool-size field on the record.

**Pool starvation is a different event and is out of scope here.** When the banded
pool is smaller than 2, `runAuthoring` returns early (`harvest.go:306`) for the
whole run rather than rejecting a word, so it is per-run, not per-word, and it
writes no record because there is nothing per-word to write.

**A single verdict can fail several booleans at once.** `harvest.go:382` is one
branch over `!verdict.Entails || verdict.Glosses || !verdict.Named`, so the
record stores the FIRST failing condition in that written order, and the order is
part of the contract rather than an accident of evaluation.

**Budget exhaustion writes no record.** `markUnfinished` (`harvest.go:274`)
already draws this line — a rejection counts, a budget cut does not, which leaves
the word pending for the next pass. Recording a budget cut would suppress a retry
that must happen. Same for a transport error, which returns rather than
continuing.

### The policy: default never

- **default** — a word with a recorded failure for the form being authored is
  skipped before any model call. No cloze item, no spend.
- `--retry-failed=model-changed` — attempt again where the record's
  `(Provider, Model)` differs from the client's current selection.
- `--retry-failed=all` — attempt again regardless.

An **unknown selection** is not a change. `llm.SelectionOf` returns a zero
`ModelSelection` for a client that does not report one
(`internal/llm/auto.go:38`), so comparing empty to empty must read as "same
model", or `model-changed` silently becomes `all` on any client without
provenance.

"Older than N" is deliberately absent, and costs nothing to add later: the record
carries `At`.

### The skip must be observable

`harvest.go:306`'s own comment is the warning: *"A silent skip here is what made
three M2 pins unfalsifiable."* A skip that prints nothing and counts nothing is
indistinguishable from a word that was never considered — and the background
harvester discards both writers (`background.go:224`), so in the path that runs
unattended it would be invisible by construction.

So the run summary (`harvest.go:463`, which today counts `authored` / `skipped` /
`rejected`) gains a **suppressed** count, and it is a count rather than a line per
word: a deck with 400 unauthorable words must not print 400 lines every run.

### Undo

`store.Forget` (`store/store.go:100`) is the existing "remove what this word
owns". A new per-word directory must be classified in `perWordDirs` or `Forget`
silently leaves the record behind — `yaml_test.go:744` makes that a hard
requirement — and `define -forget <word>` is then the operator's undo for a
single word, with `--retry-failed=all` the bulk one.

### The homes a new runtime directory has that the compiler cannot see

A new per-word surface needs an entry in `store.RuntimeDirs` (`store/yaml.go:57`),
which is load-bearing in four places no compiler checks:

1. `.gitignore` — `TestGitignoreCoversRuntimeDirs` (`repo_guard_test.go:562`)
2. the index and history guards (`repo_guard_test.go:475`, `:520`)
3. `perWordDirs` / `Forget` (`yaml_test.go:744`)
4. the store-layout blocks in `cmd/define/README.md` **and** `atlas/define.md` —
   `TestStoreLayoutDocsNameEveryRuntimeDir` (`doc_sync_test.go:667`)

## Done when

- **Two consecutive `--harvest` runs over a deck with an unauthorable word make
  model calls on the first and zero on the second**, counted through the LLM fake.
  The fixture must hold **at least 2 banded words**, or `runAuthoring` returns at
  `harvest.go:306` having made no calls and the assertion passes vacuously — the
  precise trap that comment was written about.
- That test **fails against today's code**, where the word is re-asked. A
  cost-control test that passes before the change is testing the fake.
- Each of the five states is recorded by the site that produces it, driven from a
  fake whose verdict selects the state — an enumeration over the states, so a
  sixth rejection path added later without a state fails rather than recording
  nothing.
- A verdict failing two booleans at once records the first in the documented
  order, asserted.
- **A budget cut writes no record**, and the word is attempted again on the next
  run — asserted, because the opposite silently loses the word forever.
- A transport error writes no record.
- `--retry-failed=model-changed` attempts again when the selection differs and not
  when it matches; a zero `ModelSelection` on both sides counts as matching.
- `--retry-failed=all` attempts everything with a record.
- The run summary reports a suppressed count, and it is a count — a deck with many
  suppressed words prints O(1) lines, asserted.
- `define -forget <word>` removes the record: `Forget` names the new directory,
  asserted through `perWordDirs`, not by reading the YAML.
- `Reason` round-trips through `oneLine`: a reason containing an ANSI escape is
  stored without it.
- The record list is capped, and the newest survive.
- `storetest` covers the new surface for BOTH implementations, so `Mem` cannot
  hold a state `YAML` cannot — the divergence `store/item.go:88` records having
  been bitten by.
- `RuntimeDirs` is updated and all four non-compiler homes are green:
  `TestGitignoreCoversRuntimeDirs`, both repo guards, and
  `TestStoreLayoutDocsNameEveryRuntimeDir`.
- `atlas/define.md` records the record, the five states, and the policy.

## Plan

- [ ] `sdlc start-plan`, then the durable plan in `workshop/plans/`
- [ ] M1 — the store surface: `Attempt`, `AttemptState`, `ParseAttemptState`,
      the cap, `oneLine` on `Reason`, `RuntimeDirs` + its four homes,
      `perWordDirs`/`Forget`, `storetest` for both implementations
- [ ] M2 — the write side: every rejection site records its state; budget and
      transport errors record nothing; the first-failing-boolean order
- [ ] M3 — the read side: the pre-call skip, `--retry-failed`, the suppressed
      count in the summary
- [ ] atlas, then `sdlc close`

## Log

### 2026-09-17

Split out of #68 (real sentences as cloze material) after its spec review. #68
had absorbed this work, but the two share no code path: this is pure cost control
over today's pipeline and lands independently, while #68 turned out to need a
capture path and a sentence segmenter that do not exist yet. #68 depends on this
one, because caching a rejected *candidate* is the same machinery.

The `allVetoed` correction came from that review. An earlier draft had its retry
trigger as "a larger banded pool, not the model", which is backwards:
`tierAboveBand` accepts every word, so `pickDistractors` never returns empty in
production and `len(kept) == 0` means the veto refused everything — a model
decision. The draft's `PoolSize` field and its exclusion from `model-changed` are
both dropped; they would have withheld retry from exactly the case a model change
fixes. Two error classes needing opposite responses must not collapse
(`lessons.md`), and this was the inverse: one class wearing two triggers.
