---
id: 000016
status: working
deps: [tools#3, tools#11]
github_issue:
created: 2026-08-22
updated: 2026-08-23
estimate_hours:
started: 2026-08-23T13:35:58-07:00
---

# free-form Q&A in the console: input classification + the directory as context

## Problem

You look up `sycophantic`, and the next thing you want is *"what's the difference
to obsequious?"* — with examples. Today that means leaving the tool.

The wider point (operator, 2026-08-22): the software should meet the learner where
they are, not require them to conform to a way of using it. Free-form learning is
the third verb, alongside definition and pronunciation — not a mode you switch into.

## Spec

Typing a question at the prompt answers it. There is no mode, no prefix to
remember, and no ceremony.

### Input classification — one decision table

`parseREPLLine` already splits `/command` from lookup. This adds a third outcome,
and it must stay ONE parser: two loops re-implementing "what does this line mean"
is a mistake this repo has already paid for twice (`lessons.md`, define #14).

Word count cannot be the signal — `hot dog` is a two-word headword and
`defenestrate` is one word. The signal that works is free, offline and already
present: **ask NOAD first.**

| input | classified as |
|---|---|
| `/history 7` | command (unchanged — `/` in column 1) |
| `sycophantic`, `hot dog` | lookup — NOAD has an entry |
| `what's the difference to obsequious?` | question — no entry, reads interrogative |
| `sycophanti` | not-found (unchanged) — no entry, does not read interrogative |

The interrogative test is a pure function over the line, table-tested against a
fixture set. Both directions of failure need an escape hatch, and both must be
cheap: a way to force a question, and a way to force a lookup. Neither may be the
only way to reach its outcome.

### The directory is the context

Not a chat history. `define` was started in a directory that holds `words/`,
`events/` and (once #17 lands) `user-model.md`. The question goes to the model with:

- the last few words of this session, and the recent deck from the store;
- the current word's NOAD entry, when the question follows a lookup;
- `user-model.md`, so the answer pitches at the right level and register.

Three consequences fall out, and they are the reason for this shape: a fresh
process answers just as well as a long-running one; the context is inspectable as
files rather than trapped in memory; and the answer is adaptive for the same reason
the generated items are.

### Shape

- Streams, because a paragraph arriving all at once after four seconds reads as a
  hang. Interruptible with Ctrl-C like everything else in the loop (raw mode: the
  key reader owns cancellation — see `lessons.md`, define #14).
- Multi-turn within a session: a follow-up *"give me three more examples"* resolves
  against the previous exchange.
- Q&A exchanges are recorded as events, so #17 can see what the learner asked about
  — a question is a strong signal of what they are working on.
- Seam unavailable → the classifier still runs and says so plainly (`define: no
  model configured; \`what's the difference…\` is not a word`), rather than
  attempting a dictionary lookup of a sentence.

## Done when

- [ ] Classification is a pure function with a fixture table; every row in the
      table above is asserted, including `hot dog` as a lookup and a typo as
      not-found.
- [ ] Both escape hatches work, and each is exercised by a test that fails when the
      hatch is removed.
- [ ] A full Q&A round trip runs against the fake, with recent words and
      `user-model.md` visible in the recorded prompt.
- [ ] A follow-up question resolves against the previous exchange.
- [ ] Ctrl-C mid-stream returns to the prompt with the session intact.
- [ ] With the seam unavailable, a question produces the explanatory message and a
      lookup still works.
- [ ] Driven through the raw TUI loop, not only the piped loop — a wiring only a
      loop shell supplies must be pinned by a test that drives that loop shell
      (`lessons.md`, define #15).

## Plan

Design: [`workshop/plans/000016-console-qa-plan.md`](../plans/000016-console-qa-plan.md).

- [x] Design via `sdlc start-plan` before implementing.
- [ ] M1 — the console knows a question from a word: `readsAsQuestion`, the `?`
      and `\` hatches on `parseREPLLine`, routing from the NOAD miss branch, and
      the honest "no model configured" message in all three entry modes.
- [ ] M2 — the answer: `askContext` + pure prompt, `runAsk` streaming through
      `internal/llm`, `Store.UserModel`, the `asked` event, scoped Ctrl-C.

## Log

### 2026-08-22

Created from the operator conversation that broadened `define-learn` to an adaptive
program. See the project file's `## Log` scope event for the surrounding decisions.

### 2026-08-23

`sdlc start-plan` run; durable plan written to
`workshop/plans/000016-console-qa-plan.md`. Three decisions worth surfacing
before code: the NOAD lookup is **not** repeated — routing happens inside
`lookupAndRender` after its single `dict.Lookup`, which also keeps it the one
capture site (ARCH-DRY); a question is routed **before** capture, so it never
lands in the event log as a not-found lookup (#8/#17 fold over that log); and
Ctrl-C mid-stream has to mean something narrower than everywhere else, so the raw
key reader's cancel becomes a swappable sink rather than the session cancel
(ARCH-PURE keeps the swap testable without a terminal).

Three open questions for the operator are listed at the foot of the plan: the
choice of `\` for the force-literal hatch, whether the ≥5-word arm of the
classifier is too generous, and whether `Store.UserModel` belongs here or in #17.
