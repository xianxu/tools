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

- [ ] In a non-deck directory on a terminal, `define <word>` asks before creating
      anything, and answering no leaves the directory untouched — verified by
      listing it afterwards, not by reading the code.
- [ ] Answering no still prints the definition, and `--stats` / `--play` /
      `/history` report EMPTY rather than refusing.
- [ ] With stdin not a terminal, nothing is created, nothing is asked, the
      lookup still prints, and the exit code is unchanged from today.
- [ ] `--here` creates without asking, on a terminal or not.
- [ ] An existing deck is never asked about — no prompt appears in a directory
      that already has any runtime dir or file.
- [ ] Every write-shaped method on `store.Store` consults the gate, asserted
      mechanically against the interface so a 16th method cannot be added
      ungated.
- [ ] `--stats` in an unsaved directory does not claim a word will "join your
      deck".


## Plan

Two boundaries, genuinely closed apart (AGENTS.md §3): **M1 lands the seam with a
gate that always allows — no behaviour change at all** — so the plumbing is proven
before any policy rides on it. M2 supplies the policy.

Full design: `workshop/plans/000050-ask-before-making-the-current-directory-a-deck-plan.md`

- [ ] M1 — `store.IsDeck`, derived from `RuntimeDirs` + `RuntimeFiles`, not from a
      hardcoded `words/`.
- [ ] M1 — `gatedStore`: all 15 `Store` methods, 8 gated writes, 7 ungated reads;
      denial swaps to `store.Mem`.
- [ ] M1 — the write set is DERIVED from the `Store` interface by AST, so a 16th
      method cannot be added ungated; mutated in all FOUR shapes (#49
      `guard-fails-open`).
- [ ] M1 — `gatedStore` runs through `storetest.Suite`, allowed and denied.
- [ ] M1 — `openStore` wraps the deck; gate allows always. Whole suite still green.
- [ ] M2 — `--here` and `deckGate`: already-a-deck / `--here` / no-tty / ask,
      defaulting to no.
- [ ] M2 — the raw loop resolves the gate BEFORE entering raw mode, so no store
      write can ever prompt against the loop's own key reader.
- [ ] M2 — the empty `--stats` screen stops promising a word will "join your deck"
      when nothing is being saved.
- [ ] M2 — end-to-end on a real directory: declining leaves it byte-identical
      (verified by listing it), the lookup still prints, exit 0.
- [ ] M2 — atlas: the three states, and that `IsDeck` derives.

## Log

### 2026-09-10
