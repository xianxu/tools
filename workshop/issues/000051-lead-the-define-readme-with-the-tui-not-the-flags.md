---
id: 000051
status: working
deps: []
github_issue:
created: 2026-09-11
updated: 2026-09-11
estimate_hours: 1.24
started: 2026-09-11T13:16:01-07:00
---

# lead the define README with the TUI, not the flags

## Problem

## Spec

**Lead the define README with the TUI.** Operator: *"I don't think typical user
will use the command line version of it."* The session already does everything —
lookups, `/play`, `/stats`, `/history`, `/lang`, `/sound`, `/pron`, questions — so
the flags are a scripting interface, and the README should read that way.

Today `cmd/define/README.md` opens usage with 13 command-line examples and does
not reach "The interactive session" until line 333 of 873.

### Approved order (operator, 2026-09-11)

Install → **Start it: `define`** (the deck question, keys, clickable words, known
words, completion) → **Everything is a `/` command** → Reviewing what is due (quiz
detail kept inline — the operator's choice over moving it to a reference section)
→ Asking questions → Is any of this working → Languages → The learner model ·
Practice material → What it writes → **From the command line** → Checking the
model connection.

### Found while mapping it

- **"Checking the model connection" is mostly misfiled.** Of its ~140 lines only
  the first ~30 are about the model connection; the rest is the exit-code table,
  the whole per-command reference (`/play`, `/stats`, `/history`, `/sound`,
  `/pron`, `/lang`) and dictionary selection. The commands section the operator
  asked for already exists — under the wrong heading.
- **A sentence #50 made stale:** *"In a directory with no deck, both forms … exit
  `1`"*. Since #50 a directory that is not a deck has an EMPTY deck and exits
  `0`; only `DEFINE_NO_CAPTURE`, or no working directory, still has no deck at
  all. #50's superseded-claims guard missed it because the phrase is not in its
  list.
- **The README's command list is unpinned.** `TestDocsQuoteTheCommandList` pins
  the registry-derived table in `atlas/define.md` only. The new `/` section quotes
  the same marked span and the test extends to both pages — `commands` owns the
  list and the pages consume it (ARCH-DRY).
- **Two position words point across sections that now move.** *"the two question
  hatches above"* and *"`/stats` is the screen above"* both referred to sections
  that sat above the commands text and now sit below it. A line-multiset diff
  cannot see this — the words survive intact while becoming false — so they become
  links, and links survive the next reorganisation. That creates a new way to
  break (dead anchors), so a test pins that every in-page link resolves.
- **A guard this Spec declared position-independent was not.** `keyTableIn` found
  its table with `strings.Index` on `| key | does |` — and the README has TWO tables
  with that header, the session's keys and the review keys. It checked the review
  table only because that section came first; moving the session to the top made
  it silently check the wrong one. Reading the code had concluded otherwise.
  Applying the reorganisation in a throwaway worktree and running the suite is
  what showed it. The review table gets a marked span and `keyTableIn` finds it
  with `markedSpan`, the helper `dictselect_test.go` already uses — a locator has
  to be unique.

### Constraints

- A move, not a rewrite: prose is carried verbatim except the listed edits, and a
  line-multiset diff of old against new proves it.
- "The directory is the deck, so it asks first" keeps its heading text, so the
  anchor the README links to (added in #50) still resolves.
- Every README doc-sync guard must locate content by something UNIQUE — a marked
  span, or a header that occurs once. An earlier draft of this Spec claimed that
  already held and named `keyTableIn` as checked; it did not hold (see "Found
  while mapping it"), and that guard is now scoped by a marker. `markedSpan` and
  the stale-artifact-name counts do not depend on position.

### The expected diff (PQ-1)

Every line not listed here is carried over verbatim. Run from the repo root on the
branch — `<` is a removed line, `>` an added one:

```sh
diff <(git show "$(git merge-base main HEAD)":cmd/define/README.md | grep -v '^$' | sort) \
     <(grep -v '^$' cmd/define/README.md | sort)
```

| # | edit | removed | added |
|---|---|---|---|
| 1 | TUI-first intro | the 3-line *"Print a word's dictionary definition…"* paragraph | a 4-line *"Run `define`, type a word…"* paragraph |
| 2 | new section | — | `## Start it: \`define\`` and its 3-line lead |
| 3 | deck question demoted | `## The directory is the deck, so it asks first` | the same text as `###` — anchor unchanged |
| 4 | session folded into Start it | `## The interactive session` | `### Keys` |
| 5 | new section | — | `## Everything is a \`/\` command`, a 1-line lead, and the 10-line command table quoted from the registry's marked span |
| 6 | stale sentence corrected | the 4-line *"In a directory with no deck … exit `1`"* | a 5-line version: only `DEFINE_NO_CAPTURE` or no working directory exits `1` |
| 7 | position word → link | *"…for the two question hatches above: …"* | the same line, linking `#asking-questions` |
| 8 | position word → link | *"`/stats` is the screen above, from the prompt…"* | the same line, linking `#is-any-of-this-working` |
| 9 | `/play`-first lead | the 3-line *"**`define --play` reviews what is due today.**…"* | a 4-line *"**`/play` reviews what is due today** — `define --play` from the shell…"* |
| 10 | dictionary selection moved into Languages | — | `### Which dictionary answers` |
| 11 | new section | `## Using it` | `## From the command line` and its 3-line lead |
| 12 | review key table gets a marked span | — | `<!-- review-keys -->` and `<!-- /review-keys -->` around the existing table |

Plus the 10 command-table lines appearing in the README for the first time. Line
counts: 874 → ~905.

### Non-goals

- **No prose rewrite.** Everything outside the twelve edits is moved verbatim;
  per-command reference content is untouched.
- **No code change** beyond four test edits: the command-table pin, the
  superseded-claims phrase, the anchor-resolution guard, and `keyTableIn`'s
  marker scoping.
- **No atlas restructuring.** The atlas already quotes the command table and is
  the map, not the manual.
- **Not re-deciding the quiz-detail placement** — the operator kept it inline.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Calibration tagged **stale** (#127).*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.20 impl=0.04
item: atlas-docs               design=0.05 impl=0.08
item: atlas-docs               design=0.02 impl=0.04
item: smaller-go-module        design=0.02 impl=0.10
item: milestone-review         design=0.0  impl=0.60
design-buffer: 0.30
total: 1.24
```

| row | the work |
|---|---|
| `issue-spec` 0.20/0.04 | mapping the README found the misfiled tail and a sentence #50 made stale; two operator questions settled the order. |
| `atlas-docs` 0.05/0.08 | the `cmd/define/README.md` reorganisation — scripted as a move by asserted content anchors and verified by a line-multiset diff, so the cost is the script, not hand-editing 873 lines. |
| `atlas-docs` 0.02/0.04 | the root README's `define` block. |
| `smaller-go-module` 0.02/0.10 | four guard edits: the command table pinned over `derivedDocs`, the stale phrase added to the superseded-claims list, a new anchor-resolution test, and `keyTableIn` scoped by a marked span — the last found by pre-validating the move in a worktree. |
| `milestone-review` 0.0/0.60 | one close, above the table's 0.2–0.5 on local evidence: `#50`'s close review raised `readme-gate` three rounds running, and this diff is README-shaped end to end. |

**Design buffer 0.30, not 0.15:** the plan is thorough but lives in the issue —
there is no separate `workshop/plans/` document, which is v3.1's condition for the
lower buffer.

**Reconciliation.** Σdesign = 0.29, Σimpl = 0.86. 0.29 × 1.30 + 0.86 = **1.24**.

## Done when

- [ ] The README's first usage section is the session (`define`), and the `/`
      commands come before any flag.
- [ ] Flags and one-shot use live in one "From the command line" section.
- [ ] No prose is lost: the diff command above shows exactly the twelve edits
      enumerated in "The expected diff", and nothing else.
- [ ] The stale no-deck sentence is corrected and added to the superseded-claims
      guard, so it cannot come back.
- [ ] The README's command table is pinned against the registry.
- [ ] Every in-page link in the README resolves, pinned by a test.
- [ ] The review-key guard locates its table by a marked span, so section order
      cannot change which table it checks.
- [ ] The root README's `define` block leads with the session.
- [ ] README doc-sync tests and the full suite are green.

## Plan

Single-pass: one boundary, plain checkboxes (AGENTS.md §3).

- [ ] Reassemble `cmd/define/README.md` in the approved order by content anchors,
      each asserted to match exactly once.
- [ ] `## Start it` and `## Everything is a / command` built from the existing
      sections, with the command table quoted from the registry's marked span.
- [ ] Exit codes to `## From the command line`; dictionary selection into
      `## Languages`.
- [ ] Correct the stale no-deck sentence; add it to the superseded-claims guard.
- [ ] Extend `TestDocsQuoteTheCommandList` over `derivedDocs` (`dictselect_test.go:491`),
      which already holds both pages — not a second hand-typed list (ARCH-DRY).
- [ ] The two position words that pointed across moved sections become links;
      `TestREADMEAnchorsResolve` pins every in-page link.
- [ ] `keyTableIn` locates the review key table by a `<!-- review-keys -->` marked
      span (via `markedSpan`) instead of the first `| key | does |`.
- [ ] Root README's `define` block leads with the session.
- [ ] Verify: the diff command above, README doc-sync tests, full suite.

## Log

### 2026-09-11

## Revisions

**2026-09-11 — a fourth test edit and a 12th README edit, after plan-quality cleared.**
*Reason:* applying the approved reorganisation in a throwaway worktree and running
the suite failed `TestREADMEKeyTableNamesEveryLiveKey`. `keyTableIn` located its
table as the first `| key | does |` in the README, and there are two; the move put
the session's table first. The Constraints section had claimed that guard was
position-independent — checked by reading, disproved by running.
*Delta:* the review key table gains a `<!-- review-keys -->` marked span (edit 12);
`keyTableIn` finds it with `markedSpan`; Constraints corrected in place with the
earlier claim named; the non-goals' test-edit count, already wrong at three, is now
four; Done-when and Plan gain a row each.
