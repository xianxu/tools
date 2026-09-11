---
gate: plan-quality
issue: 50
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-10T17:02:17-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: store.WriteLang is never gated, so `define /lang es` still creates lang.txt in an unconfirmed directory
          detail: |-
            The Spec names WriteLang as needing the same gate; the plan file never mentions it, and
            its Architecture paragraph claims a Store wrapper "catches every disk-creating operation",
            which is false for a free function. openStore wires persistLang at main.go:350 -> lang.go:98
            -> writeBytesAtomic, and newCommandCtx hands it to a ONE-SHOT /lang (command.go:232), so
            `define /lang es` writes lang.txt having asked nothing, and /lang after a decline does too.
            Task 8's e2e test only does a one-shot lookup and would not see it. Name the gated seam for
            WriteLang and enumerate the non-Store write paths the wrapper cannot reach.
          family: ungated-write-path
          round: 1
        - id: PQ-2
          severity: Important
          title: The decision is held per-gatedStore, but a session builds two or more wrappers
          detail: |-
            openStore builds two YAML stores — `flat` (main.go:323, backing newStoreHistory and
            newCachingFeed at main.go:340) and the per-language `st` (main.go:326). Task 4 wraps only
            `st` while the plan's integration table claims newStoreHistory is wrapped too. Wrap one and
            cachingFeed's SetNewsItems (news.go:133 -> yaml.go:561 MkdirAll) bypasses the gate; wrap both
            and each holds its own `state` and its own Mem, so "asked exactly once" is false at the
            wiring level even though TestGatedStoreAsksOncePerSession passes. And /lang rebuilds langDeps
            (command.go:426), producing a fresh deckUndecided wrapper that discards Task 6's
            pre-resolution and prompts from inside raw mode. The decision belongs to the directory — one
            memoised closure per openStore — not to the wrapper instance.
          family: decision-extent
          round: 1
        - id: PQ-3
          severity: Important
          title: Task 6 pre-resolves in runEditor only; replLines also runs with stdin a terminal
          detail: |-
            replraw.go:23 and :28 fall back to replLines with `false /*stdin is a tty*/`, and replLines
            reads stdin through scanLines' goroutine (repl.go:396). A prompt firing mid-loop there races
            that scanner for the user's answer exactly as it would in raw mode. Resolve above the
            replRaw/replLines choice at repl.go:229 so one place covers both loop shells, and pin it with
            a test that drives each loop shell.
          family: loop-shell-wiring
          round: 1
        - id: PQ-4
          severity: Important
          title: The gate needs stdin and a terminal predicate; openStore receives neither, and Tasks 4 and 5 assume different arities
          detail: |-
            The seam is `newStore func(options, io.Writer) storeDeps` (main.go:74), assigned to openStore
            in realDeps (main.go:103) and in ~15 tests. Task 4's test calls openStore(options{},
            io.Discard); Task 5 posits deckGate(dir, opt, in, out, tty). State whether newStore's type
            changes or the gate is built in run (stdin and d.stdinIsTerminal are in scope at main.go:737)
            and injected. Also replace the bare `tty bool`: this repo has three distinct terminal
            questions (opt.tty for stdout at main.go:403, d.stdinIsTerminal at main.go:95, colour) and
            repl.go:228 already computes terminalUI for this; conflating two is the bug it has made three
            times.
          family: undeclared-seam-change
          round: 1
        - id: PQ-5
          severity: Important
          title: IsDeck splices the untrusted working-directory path into a glob pattern
          detail: |-
            filepath.Glob(filepath.Join(dir, pat)) with dir from os.Getwd(): a cwd containing `[` makes
            the pattern malformed, Glob returns ErrBadPattern, the branch is skipped and an established
            deck reads as NOT a deck — breaking Done-when #5 and then re-MkdirAll-ing over a live deck on
            a yes. `*`/`?` in the path can match a sibling. Match entry names instead (os.ReadDir +
            filepath.Match) so the path is never part of the pattern, and give IsDeck one strategy line
            for the adversarial class: directory names carrying glob metacharacters.
          family: path-as-pattern
          round: 1
        - id: PQ-6
          severity: Important
          title: renderStats "gains one parameter" with no named source for its value
          detail: |-
            printStats(deckStore store.Store, clock store.Clock, who string, out, errOut io.Writer)
            (stats.go:62) cannot see gatedStore.state, and reading the decision must not RESOLVE it or
            --stats starts asking — the false alarm the lazy design exists to prevent. Name the seam that
            answers "is anything being saved here" without resolving, or the implementer will either
            prompt on --stats or restate the gate policy in stats.go (ARCH-DRY).
          family: unwired-display-input
          round: 1
        - id: PQ-7
          severity: Minor
          title: 'Two claims about existing code are wrong: the Go version and the eleven nil-deck refusals'
          detail: |-
            The plan says "Go 1.24" and the Risks section leaves "t.Chdir is Go 1.24 ... confirm at Task
            4" OPEN; go.mod line 3 says `go 1.26`, so the risk is already closed. The Spec says all
            eleven nil-deck sites "REFUSE with noDeckMessage and exit 1": main.go:176 is
            `if d.deck == nil { d.deck = sd.deck }` inside withStore (not a refusal at all), ask.go:267
            returns a context without the learner model, and cloze.go:190 returns a nil question — none
            of the three prints noDeckMessage or exits 1. The design conclusion survives; the count does
            not.
          family: unbacked-claim
          round: 1
        - id: PQ-8
          severity: Minor
          title: Forget is classified as a gated write but creates nothing, so --forget asks to CREATE a deck
          detail: |-
            YAML.Forget only os.Remove/os.RemoveAll (yaml.go:789, 820) — no MkdirAll. Under the gate
            d.deck is never nil, so `define --forget x` in a non-deck directory now asks permission to
            create a deck in order to delete nothing: the false alarm the Spec's lazy rule exists to
            avoid. Either say why Forget is gated anyway, or make the trigger "would create" rather than
            "is write-shaped".
          family: false-alarm-prompt
          round: 1
        - id: PQ-9
          severity: Minor
          title: The plan embeds full implementation and test bodies that will be rewritten within the hour
          detail: |-
            gatedStore's ~60-line body, four complete test functions and the IsDeck implementation are
            reproduced verbatim. This repo has a recorded lesson that plan code blocks "get pasted
            verbatim" and that stale claims survive inside them across rounds. One strategy line per
            risky function plus the signatures carries the same information and cannot rot.
          family: plan-restates-the-diff
          round: 1
        - id: PQ-10
          severity: Minor
          title: The plan states no non-goals
          detail: |-
            Worth one short section: not changing DEFINE_NO_CAPTURE's exit 1, not teaching the eleven
            nil-deck sites a new "empty but absent" meaning, not adding a persistent marker file, not
            gating reads. The Spec argues several of these; the plan should say them as boundaries so a
            later round does not reopen them.
          family: no-stated-non-goals
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-10T17:12:03-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: persistLang gated explicitly; non-Store write paths enumerated (WriteLang, MigrateToLanguages) and both pinned by tests.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: One *deckPermission per process, held by every wrapper and the persistLang closure; flat and st both wrapped.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Task 8 resolves above the replRaw/replLines branch at repl.go:229, with one test per loop shell.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: newStore/openStore arity stated once for both tasks; deckAsker reuses deps.stdinIsTerminal instead of a bare tty bool.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: os.ReadDir + filepath.Match on entry names only, with a bracket-named-directory test row.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: saving() reads (allowed, decided) without resolving; threaded through printStats and mutation-pinned.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: go.mod:3 confirms go 1.26 and the risk is closed; the 8-refuse/3-degrade split is corrected in both Spec and plan.
          round: 2
        - id: PQ-8
          disposition: addressed
          note: Buckets renamed createsOnDisk/doesNotCreate; Forget ungated with the reason stated.
          round: 2
        - id: PQ-9
          disposition: addressed
          note: Bodies replaced by signatures plus one strategy line per risky function.
          round: 2
        - id: PQ-10
          disposition: addressed
          note: Five non-goals stated, including the DEFINE_NO_CAPTURE divergence and not gating reads.
          round: 2
      blocked: false
content_hash: 15484777b32988696ca4c76696523abd0f9ad8ed7f5d19229f88aa0185596923
---

# Gate ledger — tools#50 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-10T17:02:17-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `ungated-write-path` store.WriteLang is never gated, so `define /lang es` still creates lang.txt in an unconfirmed directory
  The Spec names WriteLang as needing the same gate; the plan file never mentions it, and
  its Architecture paragraph claims a Store wrapper "catches every disk-creating operation",
  which is false for a free function. openStore wires persistLang at main.go:350 -> lang.go:98
  -> writeBytesAtomic, and newCommandCtx hands it to a ONE-SHOT /lang (command.go:232), so
  `define /lang es` writes lang.txt having asked nothing, and /lang after a decline does too.
  Task 8's e2e test only does a one-shot lookup and would not see it. Name the gated seam for
  WriteLang and enumerate the non-Store write paths the wrapper cannot reach.
- **PQ-2** [Important] `decision-extent` The decision is held per-gatedStore, but a session builds two or more wrappers
  openStore builds two YAML stores — `flat` (main.go:323, backing newStoreHistory and
  newCachingFeed at main.go:340) and the per-language `st` (main.go:326). Task 4 wraps only
  `st` while the plan's integration table claims newStoreHistory is wrapped too. Wrap one and
  cachingFeed's SetNewsItems (news.go:133 -> yaml.go:561 MkdirAll) bypasses the gate; wrap both
  and each holds its own `state` and its own Mem, so "asked exactly once" is false at the
  wiring level even though TestGatedStoreAsksOncePerSession passes. And /lang rebuilds langDeps
  (command.go:426), producing a fresh deckUndecided wrapper that discards Task 6's
  pre-resolution and prompts from inside raw mode. The decision belongs to the directory — one
  memoised closure per openStore — not to the wrapper instance.
- **PQ-3** [Important] `loop-shell-wiring` Task 6 pre-resolves in runEditor only; replLines also runs with stdin a terminal
  replraw.go:23 and :28 fall back to replLines with `false /*stdin is a tty*/`, and replLines
  reads stdin through scanLines' goroutine (repl.go:396). A prompt firing mid-loop there races
  that scanner for the user's answer exactly as it would in raw mode. Resolve above the
  replRaw/replLines choice at repl.go:229 so one place covers both loop shells, and pin it with
  a test that drives each loop shell.
- **PQ-4** [Important] `undeclared-seam-change` The gate needs stdin and a terminal predicate; openStore receives neither, and Tasks 4 and 5 assume different arities
  The seam is `newStore func(options, io.Writer) storeDeps` (main.go:74), assigned to openStore
  in realDeps (main.go:103) and in ~15 tests. Task 4's test calls openStore(options{},
  io.Discard); Task 5 posits deckGate(dir, opt, in, out, tty). State whether newStore's type
  changes or the gate is built in run (stdin and d.stdinIsTerminal are in scope at main.go:737)
  and injected. Also replace the bare `tty bool`: this repo has three distinct terminal
  questions (opt.tty for stdout at main.go:403, d.stdinIsTerminal at main.go:95, colour) and
  repl.go:228 already computes terminalUI for this; conflating two is the bug it has made three
  times.
- **PQ-5** [Important] `path-as-pattern` IsDeck splices the untrusted working-directory path into a glob pattern
  filepath.Glob(filepath.Join(dir, pat)) with dir from os.Getwd(): a cwd containing `[` makes
  the pattern malformed, Glob returns ErrBadPattern, the branch is skipped and an established
  deck reads as NOT a deck — breaking Done-when #5 and then re-MkdirAll-ing over a live deck on
  a yes. `*`/`?` in the path can match a sibling. Match entry names instead (os.ReadDir +
  filepath.Match) so the path is never part of the pattern, and give IsDeck one strategy line
  for the adversarial class: directory names carrying glob metacharacters.
- **PQ-6** [Important] `unwired-display-input` renderStats "gains one parameter" with no named source for its value
  printStats(deckStore store.Store, clock store.Clock, who string, out, errOut io.Writer)
  (stats.go:62) cannot see gatedStore.state, and reading the decision must not RESOLVE it or
  --stats starts asking — the false alarm the lazy design exists to prevent. Name the seam that
  answers "is anything being saved here" without resolving, or the implementer will either
  prompt on --stats or restate the gate policy in stats.go (ARCH-DRY).
- **PQ-7** [Minor] `unbacked-claim` Two claims about existing code are wrong: the Go version and the eleven nil-deck refusals
  The plan says "Go 1.24" and the Risks section leaves "t.Chdir is Go 1.24 ... confirm at Task
  4" OPEN; go.mod line 3 says `go 1.26`, so the risk is already closed. The Spec says all
  eleven nil-deck sites "REFUSE with noDeckMessage and exit 1": main.go:176 is
  `if d.deck == nil { d.deck = sd.deck }` inside withStore (not a refusal at all), ask.go:267
  returns a context without the learner model, and cloze.go:190 returns a nil question — none
  of the three prints noDeckMessage or exits 1. The design conclusion survives; the count does
  not.
- **PQ-8** [Minor] `false-alarm-prompt` Forget is classified as a gated write but creates nothing, so --forget asks to CREATE a deck
  YAML.Forget only os.Remove/os.RemoveAll (yaml.go:789, 820) — no MkdirAll. Under the gate
  d.deck is never nil, so `define --forget x` in a non-deck directory now asks permission to
  create a deck in order to delete nothing: the false alarm the Spec's lazy rule exists to
  avoid. Either say why Forget is gated anyway, or make the trigger "would create" rather than
  "is write-shaped".
- **PQ-9** [Minor] `plan-restates-the-diff` The plan embeds full implementation and test bodies that will be rewritten within the hour
  gatedStore's ~60-line body, four complete test functions and the IsDeck implementation are
  reproduced verbatim. This repo has a recorded lesson that plan code blocks "get pasted
  verbatim" and that stale claims survive inside them across rounds. One strategy line per
  risky function plus the signatures carries the same information and cannot rot.
- **PQ-10** [Minor] `no-stated-non-goals` The plan states no non-goals
  Worth one short section: not changing DEFINE_NO_CAPTURE's exit 1, not teaching the eleven
  nil-deck sites a new "empty but absent" meaning, not adding a persistent marker file, not
  gating reads. The Spec argues several of these; the plan should say them as boundaries so a
  later round does not reopen them.

## Round 2 — 2026-09-10T17:12:03-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — persistLang gated explicitly; non-Store write paths enumerated (WriteLang, MigrateToLanguages) and both pinned by tests.
- PQ-2 — addressed — One *deckPermission per process, held by every wrapper and the persistLang closure; flat and st both wrapped.
- PQ-3 — addressed — Task 8 resolves above the replRaw/replLines branch at repl.go:229, with one test per loop shell.
- PQ-4 — addressed — newStore/openStore arity stated once for both tasks; deckAsker reuses deps.stdinIsTerminal instead of a bare tty bool.
- PQ-5 — addressed — os.ReadDir + filepath.Match on entry names only, with a bracket-named-directory test row.
- PQ-6 — addressed — saving() reads (allowed, decided) without resolving; threaded through printStats and mutation-pinned.
- PQ-7 — addressed — go.mod:3 confirms go 1.26 and the risk is closed; the 8-refuse/3-degrade split is corrected in both Spec and plan.
- PQ-8 — addressed — Buckets renamed createsOnDisk/doesNotCreate; Forget ungated with the reason stated.
- PQ-9 — addressed — Bodies replaced by signatures plus one strategy line per risky function.
- PQ-10 — addressed — Five non-goals stated, including the DEFINE_NO_CAPTURE divergence and not gating reads.

## Open findings

(none — every finding has been disposed)
