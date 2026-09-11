---
id: 000050
status: working
deps: []
github_issue:
created: 2026-09-10
updated: 2026-09-10
estimate_hours:
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
      method cannot be added ungated; mutated in all three shapes (#49
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
