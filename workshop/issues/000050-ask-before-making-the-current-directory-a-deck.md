---
id: 000050
status: working
deps: []
github_issue:
created: 2026-09-10
updated: 2026-09-10
estimate_hours: 4.19
started: 2026-09-10T09:41:40-07:00
---

# ask before making the current directory a deck

## Problem

`define` makes the current directory a deck by writing into it, and it never
asks. Six `os.MkdirAll` calls scattered through `store/yaml.go` each fire on
their own first write, so `words/`, `events/`, `usage/`, `facts/`, `items/` and
`audio/` appear silently — along with `lang.txt` and `user-model.<lang>.md`,
which carries inferred claims about the learner.

The directory IS the deck (that is the design), which makes running `define` in
the wrong shell a real and quiet accident: you get a stray deck somewhere you
never meant, and you find out later.

## Spec

**Ask before making this directory a deck. When it cannot ask, do not create —
still work, and read as empty.**

### The three states

| state | deck reads | writes | exit |
|---|---|---|---|
| directory is already a deck | the deck | to disk | as today |
| not a deck, user says yes | the deck | to disk | as today |
| not a deck, user says no **or** cannot ask | **empty** | discarded | **0** |

The third row is the operator's design and it is the whole point: *"it should
still function, meaning looking up definition would work. all others relying on
history would return empty, as if there's empty history."*

### Why that is nearly free

`store.Mem` is already a complete `Store` — the atlas calls it "the reference" in
*"Two Store implementations, one conformance suite."* Supplying one as the deck
gives "as if empty" with **no consumer changes**: `--stats` folds an empty deck,
`--play` finds nothing due, `/history` recalls only this session, writes go to
memory and evaporate.

**This is why the design avoids the `nil` deck rather than extending it.** A nil
deck is checked in eleven places, and — measured per site, not assumed — **eight
of them REFUSE** with `noDeckMessage` and exit 1 (`play_loop.go:29`,
`play_cmd.go:44`, `stats.go:37`, `stats.go:236`, `history_cmd.go:219`,
`harvest.go:110`, `reflect.go:339`, `main.go:1223`) while **three DEGRADE**:
`ask.go:267` returns a context without the learner model, `cloze.go:190` returns a
nil question, and `main.go:176` is an assignment inside `withStore` rather than a
check at all.

So "no deck" today means *refuse* at eight sites and *quietly do less* at two more.
Threading a new "empty but absent" meaning through all of them is the expensive
version of this feature; handing over a real empty store is the cheap one — and it
makes the three degraders strictly better, since `ask.go` gets a `UserModel` call
that succeeds and `cloze.go` an empty item list instead of a nil question.

(An earlier draft of this Spec said all eleven refused. That was wrong, and the
plan-quality gate caught it — PQ-7.)

### It asks LAZILY, at the first write

Not at startup. `define --stats` in the wrong directory creates nothing, so it
must read empty and never ask — a prompt for a command that would not have
written anything is a false alarm, and false alarms train people to hit `y`.

Lazy also means the trigger is DERIVED rather than enumerated: any write, present
or future, meets the gate. An "these modes write" list is the hand-enumerated set
this repo keeps being bitten by (#49 I-2, #49 IV).

### Files as well as folders

The literal ask is folders. This covers `lang.txt` and `user-model.<lang>.md`
too, and that widening is deliberate rather than quiet: the learner model is the
*more* alarming artifact to find in a stranger's directory, since it holds
inferred claims about a person. `store.WriteLang` is a free function outside the
`Store` interface and needs the same gate.

### `--here`, which becomes load-bearing

Since a non-tty no longer gets a deck, a script that legitimately wants one needs
a way to say so. `--here` creates without asking. It is not a nicety under this
design; it is the only path for automation.

### `DEFINE_NO_CAPTURE` deliberately keeps its own answer

It produces a nil deck and exits 1 with *"DEFINE_NO_CAPTURE is set, so no deck
was opened"*. Under this design an unconfirmed directory exits **0** with empty
figures while the explicit opt-out exits **1**. That divergence is intended: an
explicit instruction deserves a direct answer, an accident deserves graceful
degradation. Recorded because they are neighbours and the next reader will ask.

### The message that would become a lie

`--stats` on an empty deck prints *"Nothing yet — look a word up and it joins
your deck."* In the third state nothing will join anything, and that sentence
lands in front of exactly the confused user this feature exists for. The empty
RENDERING is right; this empty MESSAGE needs a variant that says nothing is being
saved and how to fix it.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Calibration tagged **stale** (#127), so the table hours are
treated as provisional and one family is corrected against local actuals — said
out loud below rather than folded in silently.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.60 impl=0.08
item: smaller-go-module        design=0.05 impl=0.12
item: smaller-go-module        design=0.05 impl=0.12
item: greenfield-go-module     design=0.15 impl=0.28
item: cross-cutting-refactor   design=0.05 impl=0.18
item: smaller-go-module        design=0.05 impl=0.16
item: smaller-go-module        design=0.05 impl=0.14
item: smaller-go-module        design=0.02 impl=0.08
item: smaller-go-module        design=0.02 impl=0.20
item: atlas-docs               design=0.02 impl=0.06
item: milestone-review         design=0.0  impl=0.70
item: milestone-review         design=0.0  impl=0.85
design-buffer: 0.15
total: 4.19
```

| row | the work |
|---|---|
| `issue-spec` 0.60/0.08 | high in the 0.5–1.5 design band: the operator settled the shape ("as if there's empty history"), but the design still had to find that the `nil` deck is refused at eight sites and degrades at three, that `store.Mem` makes the third state nearly free, and then survive **two plan-quality rounds** whose nine findings changed five design decisions. |
| `smaller-go-module` 0.05/0.12 | `store.IsDeck` — derive from `RuntimeDirs`/`RuntimeFiles`, `ReadDir`+`Match` so the cwd never enters a pattern. Fully specced, so design ≈ 0. |
| `smaller-go-module` 0.05/0.12 | `deckPermission` — three states, three operations, the third of which (`saving`) must not resolve. |
| `greenfield-go-module` 0.15/0.28 | `gatedStore`: 15 interface methods, the `createsOnDisk`/`doesNotCreate` split, the AST classification guard with its **four** mutation shapes, AND Task 6 (running the wrapper through `storetest.Suite` allowed and denied — cheap to write, but the place a delegation bug surfaces). Three plan tasks in one row, near the 0.32 ceiling because of it. |
| `cross-cutting-refactor` 0.05/0.18 | the wiring: `newStore`/`openStore` arity, `withStore`, both YAML stores, inside `newLangDeps`, and `persistLang`. Mechanical but it touches the one file everything runs through. |
| `smaller-go-module` 0.05/0.16 | `deckAsker` + `--here` — a four-input policy with an eight-row table. |
| `smaller-go-module` 0.05/0.14 | pre-resolution above the loop-shell choice, pinned **per shell** (one test per shell, since a single test covers only the branch it took). |
| `smaller-go-module` 0.02/0.08 | the honest empty `--stats` screen. |
| `smaller-go-module` 0.02/0.20 | the end-to-end tests, which list the directory rather than trust the gate — the Done-when says so explicitly, and that is where the real bugs will be. |
| `atlas-docs` 0.02/0.06 | three states, `IsDeck` derives, `persistLang` is the one non-`Store` path. |
| `milestone-review` ×2 | **0.70 and 0.85, well above the table's 0.2–0.5 — the one deliberate departure.** |

**The review rows are corrected against local actuals, not the stale table.**
`#49` closed at **est 1.41 / actual 3.62 (ratio 0.4×)** and the entire overrun was
boundary review: four rounds, each finding real defects in the previous round's
fix. Two more data points agree — `#8` (3.71/2.98) and `#48` (3.60/3.40). Using
0.2–0.5 here would reproduce exactly the error `#49` just measured, so the two
boundaries are costed at roughly two rounds each. That is the largest single
component of this estimate and it is the one I would revise first if it proves
wrong in either direction.

**The repo-wide bias is acknowledged and deliberately NOT back-fitted.** The
estimate-quality gate points out that across eight closes the est/actual ratio
runs about 0.5 median with 6 of 8 under — so correcting only the review family
leaves the general low bias standing. That is a fair criticism of the claim
"corrected against local actuals". I am not scaling the total to match, because
v3.1 forbids ad-hoc back-fitting and a number moved to hit a remembered ratio
teaches the calibration nothing. The honest statement is: **this total is more
likely to come in low than high, and the review rows are where I would look
first.** If it lands near 4.19 that is evidence the review correction was the
missing piece; if it lands near 8 the bias is systemic and belongs in #127's
recalibration, not in this block.

**Reconciliation.** Σdesign = 1.06, Σimpl = 2.97.
1.06 × 1.15 + 2.97 = **4.19**.

**Design buffer 0.15, not 0.30**, per v3.1 step 4: there is a thorough plan doc
(`workshop/plans/000050-…-plan.md`, cleared plan-quality in 2 rounds).

**Why this is bigger than `#49`'s 3.62 actual.** `#49` was one flag, a tag and a
formula. This adds a new seam every write in the program passes through, a
mechanically-derived guard over an interface, and a policy with four inputs — plus
it changes what happens in a directory the user has never run `define` in, which
is the case nobody has test coverage for today.

## Done when

- [x] In a non-deck directory on a terminal, `define <word>` asks before creating
      anything, and answering no leaves the directory untouched — verified by
      listing it afterwards, not by reading the code.
- [x] Answering no still prints the definition, and `--stats` / `--play` /
      `/history` report EMPTY rather than refusing.
- [x] With stdin not a terminal, nothing is created, nothing is asked, the
      lookup still prints, and the exit code is unchanged from today.
- [x] `--here` creates without asking, on a terminal or not.
- [x] An existing deck is never asked about — no prompt appears in a directory
      that already has any runtime dir or file.
- [x] Every `store.Store` method that can CREATE on disk consults the gate,
      asserted mechanically against the interface so a 16th method cannot be
      added unclassified. **Not "write-shaped"** — see Revisions: `Forget` only
      removes, so gating it would ask permission to create a deck in order to
      delete nothing.
- [x] `--stats` in an unsaved directory does not claim a word will "join your
      deck".


## Plan

Two boundaries, genuinely closed apart (AGENTS.md §3): **M1 lands the seam with a
gate that always allows — no behaviour change at all** — so the plumbing is proven
before any policy rides on it. M2 supplies the policy.

Full design: `workshop/plans/000050-ask-before-making-the-current-directory-a-deck-plan.md`

- [x] M1 — `store.IsDeck`, derived from `RuntimeDirs` + `RuntimeFiles`, not from a
      hardcoded `words/`.
- [x] M1 — `gatedStore`: all 15 `Store` methods — **7 gated (creates on disk), 8
      ungated (reads, plus `Forget`, which only removes)**; denial swaps to
      `store.Mem`.
- [x] M1 — the write set is DERIVED from the `Store` interface by AST, so a 16th
      method cannot be added ungated; mutated in all FOUR shapes (#49
      `guard-fails-open`).
- [x] M1 — `gatedStore` runs through `storetest.Suite`, allowed and denied.
- [x] M1 — `openStore` wraps the deck; gate allows always. Whole suite still green.
- [x] M2 — `--here` and `deckGate`: already-a-deck / `--here` / no-tty / ask,
      defaulting to no.
- [x] M2 — the raw loop resolves the gate BEFORE entering raw mode, so no store
      write can ever prompt against the loop's own key reader.
- [x] M2 — the empty `--stats` screen stops promising a word will "join your deck"
      when nothing is being saved.
- [x] M2 — end-to-end on a real directory: declining leaves it byte-identical
      (verified by listing it), the lookup still prints, exit 0.
- [x] M2 — atlas: the three states, and that `IsDeck` derives.
- [x] M2 — README: the question, the decline, and `--here` (BR-5; the plan had no README row, which is why it was missed).

## Log

### 2026-09-10

### 2026-09-10 — M1 Task 1: `store.IsDeck`

Derived from `RuntimeDirs` + `RuntimeFiles`; `os.ReadDir` once, then
`filepath.Match` against entry **names**, so the working directory never enters a
pattern (PQ-5). `tmpPattern` is skipped — the atomic-write shadow is debris from a
crashed write, and counting it would skip the question in exactly the case where
something already went wrong.

**A test I wrote to catch PQ-5 did not catch PQ-5, and the mutation is what said
so.** The bracket-named-directory row created `words/` — a *directory* — but the
directory branch is a map lookup with no pattern in it, so the row never reached
the globbing it was written about. Restoring the `filepath.Glob(filepath.Join(dir,
pat))` form left it green. Corrected to place a runtime **file** in the
bracket-named directory; the same mutation now reddens with *"a real deck in a
directory named \"br[acket\" read as NOT a deck"*. This is #49's
`guard-fails-open` rule earning its keep one issue later: a guard verified against
the case that motivated it is verified against nothing.

Three mutations, each red on its own:

| mutation | result |
|---|---|
| `Match(pat, name)` → `Glob(Join(dir, pat))` | reddens the bracket rows (the PQ-5 defect) |
| `Match` → exact equality via `filepath.Clean` | reddens `user-model.en.md` and `.es.md`, the `??` family; literal names stay green |
| append `"scratch"` to `RuntimeDirs` | a 7th subtest appears and PASSES with no edit to `isdeck.go` — the derivation is real |

A fourth was attempted (`strings.HasPrefix` instead of `Match`) and **cannot
compile**: it orphans the `filepath` import. Recorded because "the compiler
refuses it" is a stronger guarantee than a red test, not a skipped check.

### 2026-09-10 — M1 Tasks 2–4, 6: the decision, the wrapper, the guard

`deckPermission` holds the decision for the PROCESS, not per wrapper (PQ-2). A
session builds two YAML stores and `/lang` rebuilds more; a per-wrapper state
would ask once per wrapper, so the learner answers, switches language, and is
asked again. `saving()` reads `(allowed, decided)` **without** resolving —
mutation-verified: making it call `allowed()` reddens with *"reading whether
anything is being saved must not ASK whether to start saving"*.

`gatedStore` classifies by **what a method does to the disk**, not by being
write-shaped (PQ-8). `Forget` is `os.Remove`/`os.RemoveAll` only, so it is
ungated: gating it would make `define --forget x` in a non-deck directory ask
permission to CREATE a deck in order to delete nothing.

`storetest.Suite` runs over the wrapper **allowed and denied** — 74 subtests, 0
skips, confirmed with `-v` rather than assumed from a green line. Denied conforms
too, which is the evidence that "a declined store is empty, not broken".

**The four guard shapes, each red on its own:**

| shape | result |
|---|---|
| a 16th `Store` method | *"store.Store.Ping is in neither createsOnDisk nor doesNotCreate"* |
| a method moved buckets | `TestNoNonCreatingMethodConsultsThePermission/SetItems` reddens |
| a creating method skips the permission | `TestEveryCreatingMethodConsultsThePermission/SetAudio` reddens |
| the parse returns a hardcoded list | *"derived 2 Store methods … at least fifteen"* |

**Shape 1 took three attempts, and the first two were my error, not the guard's.**
Attempt one edited `Forget(key string) (bool, error)` — the real signature is
`(removed bool, err error)`, so an unasserted `replace` silently did nothing and
the "passing" result was a test that never saw a 16th method. Attempt two added
the method without implementing it, so the package would not compile and the
guard never got to speak. The realistic form — add it AND implement it on all
four implementations (`Mem`, `YAML`, `gatedStore`, and the `failingStore` test
double) — is the only one that reaches the guard, and it reddens.

That is the third unasserted string replacement to silently no-op in this session.
Every one produced a green result that meant nothing. Recorded in
`workshop/lessons.md`.

### 2026-09-10 — M1 Task 5: the wiring, and two tests that proved nothing first

`deps.newStore` and `openStore` changed signature together, so the 22 sites that
ASSIGN `openStore` were untouched. The plan said that was the whole story; it was
not. Five sites needed edits the plan did not predict — four **direct calls** to
`openStore` (`vocab_test.go:152`, `capture_test.go:333/382/528`) and two inline
`func(options, io.Writer) storeDeps` **literals** (`askrun_test.go:147`,
`main_test.go:527`). Found by fixing `go vet` one error at a time until I stopped
and grepped for the shapes instead, which took one command and would have taken
one command at the start.

**Whole suite green with the seam in: M1 is a genuine no-op**, which is the
property it was split out to prove.

**Two of the five wiring tests passed while asserting nothing, and mutation is
what said so.**

- `TestOpenStoreGatesTheFlatStoreToo` called `sd.history.Add("alpha")` and then
  listed the directory. It passed under the ungated mutation because
  `storeHistory.Add` only appends to an in-memory slice (`history_store.go:77`)
  and never writes at all. Replaced with an assertion on the actual wiring — the
  store backing `storeHistory` must be a `*gatedStore` — plus
  `TestTheNewsCacheCannotCreateWhenDenied`, which drives the write path `flat`
  really owns without needing a network.
- The `IsDeck` bracket-directory row, earlier today, had the same shape.

Four wiring mutations, each red on its own: ungate `persistLang` → *"declining
still wrote [lang.txt]"*; ungate `flat` → *"backed by \*store.YAML"*; ungate the
language deck → *"deck is \*store.YAML"*; and `MigrateToLanguages` is pinned as
creating nothing in a non-deck directory, since it is UNGATED on that claim.

### 2026-09-10 — M2, and a lie smoke testing caught that no test did

The policy is in: `deckPolicy` answers the three cases that need nobody (already
a deck / `--here` / no terminal), `deckAsker` puts the question for the fourth,
`deckPermission` settles it once for the process, and both loop shells settle it
before they read a key.

**Smoke testing found a defect the whole suite was green on.** `define --stats`,
piped, in a non-deck directory printed *"Nothing yet — look a word up and it joins
your deck"* — the exact promise this feature exists to stop making. My `saving()`
refused to resolve (correct: it must not prompt) and therefore reported
"undecided", so the render fell back to the ordinary message. But the answer there
needs **nobody**: no terminal means no deck. I had conflated "resolving might
prompt" with "resolving is off limits".

Fixed by splitting `deckPolicy` out of `deckAsker`: the free answers settle
quietly, the one that would prompt does not. `TestStatsIsHonestWhereTheAnswerNeeds
Nobody` pins all three cases including that settling asks nobody. This is the
second time this issue that running the program taught me something the tests
could not — the first was two tests that passed while asserting nothing.

**M1 boundary review (5 findings) folded in.** BR-2 was a real bug the review
MEASURED: I applied PQ-2's rule to the decision but not to the store it swaps in,
so `newLangDeps` rebuilt the fallback and a declined session forgot itself the
moment the learner typed `/lang`. Fallbacks are now keyed by language in
`openStore`; the mutation reddens. BR-1: "nothing reaches a declined directory"
was pinned for one of seven creating methods against an in-memory backing store
that cannot show whether a directory was made — now iterated from `createsOnDisk`
on a real filesystem, with a companion test so the denial cannot pass by the call
doing nothing at all. BR-6: the answer was read through a `bufio.Reader` over
shared stdin, which reads ahead and would have swallowed the loop's next line.
BR-5: both READMEs document the question, the decline and `--here`.

**Verified on the real binary**, not only in tests: piped lookup creates nothing
and asks nothing; `--stats` says nothing is being saved; `--here` creates
`words/ events/ audio/` unasked; a second lookup in the now-real deck asks
nothing; and `define --forget cat` in an empty directory refuses the word without
offering to create a deck — PQ-8's false alarm, confirmed absent in the shipped
path.

## Revisions

**2026-09-10 — the gated set is "creates on disk", not "write-shaped" (7/8, not 8/7).**
*Reason:* plan-quality PQ-8 measured that `YAML.Forget` is `os.Remove`/`os.RemoveAll`
only. Gating it would make `define --forget x` in a non-deck directory ask permission
to CREATE a deck in order to delete nothing — the false alarm the lazy design exists to
prevent.
*Delta:* the buckets are `createsOnDisk` (7) and `doesNotCreate` (8); Done-when and the
M1 Plan row above are corrected. The classification is derived from the `Store`
interface, so the counts are the code's, not this file's.

**2026-09-10 — `store.IsDeck` is not a PURE entity.**
*Reason:* the plan's Core-concepts table lists it under Pure entities, but it calls
`os.ReadDir` and its tests need a real mutable filesystem (`t.TempDir`), which is the
plan skill's own definition of an integration point rather than a pure one.
*Delta:* it is an integration point that wraps the filesystem. The plan's own
"PURE-with-IO note" already conceded the substance; the table row was the part that
claimed otherwise. Its tests genuinely need no mocks, which is why the mislabel
survived review twice.

### 2026-09-10 — close review round 2: 9 disposed, and the guard family caught me a third time

The gate's verdict was the useful part: *"Not converging: fix rules, not
instances."* Two findings I believed fixed came back **not-addressed**, and the
reviewer proved both by mutation in a scratch worktree rather than by reading.

**BR-1/BR-10 — my absence claims had no controls, and I had WAIVED the check that
would have caught it.** Under `callStoreMethod`'s reflect-built zero arguments,
only `AppendEvent` and `SetUserModel` write anything at all — so five of seven
"nothing was created in a declined directory" subtests were asserting that a call
which does nothing creates nothing. Worse, my positive control was an aggregate
("some method wrote"), and I had written a comment explaining why per-method
checking was unnecessary. That comment is the defect: I noticed the asymmetry,
rationalised it, and shipped the hole. The reviewer's mutation — `SetItems`
dual-writing to `g.disk` AND routing through the gate — left the whole suite
green while a declined directory grew an `items/` tree.

Fixed as the rule the review states: **an absence or ordering claim needs a
PER-INSTANCE control.** `sampleCalls` gives every creating method arguments that
really write; each subtest asserts the positive control FIRST and fails loudly if
the call writes nothing even when allowed. `TestEveryCreatingMethodHasASample`
derives the table from `createsOnDisk` so it cannot fall behind. The reviewer's
own mutation now reddens.

**The same rule applied to the ordering claim, which was the third instance.**
`TestBothLoopShellsResolveBeforeReading` sampled `saving()` AFTER `repl` returned
— by which time the answer is settled either way — so moving `resolve()` below
both shells left it green. An `observingReader` now records whether the question
was settled at the moment the shell first READ stdin, and it fails if the shell
never read at all. Moving `resolve()` down now reddens both shells.

**BR-12 — two encodings of one precedence.** `deckPolicy` and `deckAsker` each
implemented already-a-deck / `--here` / no-terminal. They agreed, and nothing made
them: the observable consequence was that only one printed the "nothing will be
saved" line, so whether a piped learner was told depended on which encoding
settled first. `deckAsker` now switches on `deckPolicy`.

**BR-11 — the surfaces were swept one at a time, so one was always left.**
`--help` still promised unconditional recording after the READMEs and atlas were
updated. Fixed as a rule: `TestEverySurfaceDescribingCaptureMentionsTheQuestion`
enumerates the four surfaces, and it immediately found a fifth gap I had also
missed — `atlas/define.md` never mentioned `--here`. The prompt string now lives
in one `deckPrompt` const the README is pinned against.

**BR-6** now has a test that fails without `readLineUnbuffered`, asserted on the
unread REMAINDER — the observable difference — rather than on the answer, which
passes under either implementation.

**BR-4** corrected in both artifacts with `## Revisions` sections: the gated set is
7 creates / 8 non-creates (not 8 writes / 7 reads), and `store.IsDeck` is an
integration point, not a pure entity — it calls `os.ReadDir`. That mislabel
survived two rounds because its tests genuinely need no mocks.

**A self-inflicted loss worth recording:** an index-based slice while editing the
test file deleted two passing tests along with the block I meant to replace
(`TestStatsDoesNotPromiseToSaveWhenItCannot`, `TestStatsNeverAsks`). Caught by an
unused-import error, not by noticing. Both restored.
