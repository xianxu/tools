---
id: 000051
status: codecomplete
deps: []
github_issue:
created: 2026-09-11
updated: 2026-09-11
estimate_hours: 1.24
started: 2026-09-11T13:16:01-07:00
actual_hours: 3.14
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
  One guard is position-sensitive by design, at paragraph scale: the
  superseded-claims check looks for a claim's qualifier within 400 characters
  after it (`deckasker_test.go:364-376`). A move that separated a claim from its
  qualifying sentence would fail it — and a stale claim placed beside unrelated
  qualifying words would pass it. Neither happens here, because sections move whole: the check passes on the
  reorganised README, and reverting the corrected no-deck sentence reddens
  it (PQ-3).

### The expected diff (PQ-1)

Every line not listed here is carried over verbatim. Run from the repo root on the
branch — `<` is a removed line, `>` an added one:

```sh
{ git show "$(git merge-base main HEAD)":cmd/define/README.md | grep -v '^$' | sed 's/^/-/'
  grep -v '^$' cmd/define/README.md | sed 's/^/+/'; } |
awk '{ s = substr($0, 1, 1); l = substr($0, 2); n[l] += (s == "+") ? 1 : -1 }
     END { for (l in n) { if (n[l] < 0) for (i = 0; i < -n[l]; i++) print "< " l
                          else if (n[l] > 0) for (i = 0; i < n[l]; i++) print "> " l } }' | sort
```

**Pipes only, on purpose.** An earlier version used `diff <(…) <(…)` and a second
attempt used `mktemp`. In a sandboxed shell the first fails on `/dev/fd` and the
second on the system temp directory — and both still print a count of zero, a
false "no changes" rather than an error. This form needs neither, and was proven
in the most restricted shell available.

| # | edit | removed | added |
|---|---|---|---|
| 1 | TUI-first intro | the 3-line *"Print a word's dictionary definition…"* paragraph | a 4-line *"Run `define`, type a word…"* paragraph |
| 2 | new section | — | `## Start it: \`define\`` and its 3-line lead |
| 3 | deck question demoted | `## The directory is the deck, so it asks first` | the same text as `###` — anchor unchanged |
| 4 | session folded into Start it | `## The interactive session` | `### Keys` |
| 5 | new section | — | `## Everything is a \`/\` command`, a 1-line lead, and the 11-line command table (two markers, header, separator, seven rows) quoted from the registry's marked span |
| 6 | stale sentence corrected | the 4-line *"In a directory with no deck … exit `1`"* | a 5-line version: only `DEFINE_NO_CAPTURE` or no working directory exits `1` |
| 7 | position word → link | *"…for the two question hatches above: …"* | the same line, linking `#asking-questions` |
| 8 | position word → link | *"`/stats` is the screen above, from the prompt…"* | the same line, linking `#is-any-of-this-working` |
| 9 | `/play`-first lead | the 3-line *"**`define --play` reviews what is due today.**…"* | a 4-line *"**`/play` reviews what is due today** — `define --play` from the shell…"* |
| 10 | dictionary selection moved into Languages | — | `### Which dictionary answers` |
| 11 | new section | `## Using it` | `## From the command line` and its 3-line lead |
| 12 | review key table gets a marked span | — | `<!-- review-keys -->` and `<!-- /review-keys -->` around the existing table |

Plus the 11 command-table lines appearing in the README for the first time. Line
counts: 873 → 906 (`wc -l`); verified 15 lines removed and 41 added, every one accounted for by the table.

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
| `milestone-review` 0.0/0.60 | one close — a **deliberate 3× override of v3.1's scaled range (0.08–0.20)**, not a reading of it; an earlier version of this row compared it to v2's unscaled 0.2–0.5. Local evidence: `#50` closed at ratio 0.74 after a five-round review that raised `readme-gate` three rounds running, and this diff is README-shaped end to end. Stated as an override so a calibration pass can tell a model miss from a departure. |

**Design buffer 0.30, not 0.15:** the plan is thorough but lives in the issue —
there is no separate `workshop/plans/` document, which is v3.1's condition for the
lower buffer.

**Reconciliation.** Σdesign = 0.29, Σimpl = 0.86. 0.29 × 1.30 + 0.86 = **1.24**.

## Done when

- [x] The README's first usage section is the session (`define`), and the `/`
      commands come before any flag.
- [x] Flags and one-shot use live in one "From the command line" section.
- [x] No prose is lost: the diff command above shows exactly the twelve edits
      enumerated in "The expected diff", and nothing else.
- [x] The stale no-deck sentence is corrected and added to the superseded-claims
      guard, so it cannot come back.
- [x] The README's command table is pinned against the registry.
- [x] Every in-page link in the README resolves, pinned by a test.
- [x] The review-key guard locates its table by a marked span, so section order
      cannot change which table it checks.
- [x] The root README's `define` block leads with the session.
- [x] README doc-sync tests and the full suite are green.

## Plan

Single-pass: one boundary, plain checkboxes (AGENTS.md §3).

- [x] Reassemble `cmd/define/README.md` in the approved order by content anchors,
      each asserted to match exactly once.
- [x] `## Start it` and `## Everything is a / command` built from the existing
      sections, with the command table quoted from the registry's marked span.
- [x] Exit codes to `## From the command line`; dictionary selection into
      `## Languages`.
- [x] Correct the stale no-deck sentence; add it to the superseded-claims guard.
- [x] Extend `TestDocsQuoteTheCommandList` over `derivedDocs` (`dictselect_test.go:491`),
      which already holds both pages — not a second hand-typed list (ARCH-DRY).
- [x] The two position words that pointed across moved sections become links;
      `TestREADMEAnchorsResolve` pins every in-page link.
- [x] `keyTableIn` locates the review key table by a `<!-- review-keys -->` marked
      span (via `markedSpan`) instead of the first `| key | does |`.
- [x] Root README's `define` block leads with the session.
- [x] Verify: the diff command above, README doc-sync tests, full suite.

## Log

### 2026-09-11
- 2026-09-11: closed — README reorganised TUI-first in the operator-approved order: Install, Start it (deck question, keys, clickable words, known words, completion), Everything is a / command, Reviewing what is due (quiz detail inline, operator choice), then the rest, ending with From the command line and Checking the model connection. VERIFIED AS A MOVE, not assumed: the pipe-only multiset diff in the Spec shows 15 lines removed and 41 added, every one mapped to the twelve enumerated edits, run in the most restricted shell available. That command is pipes only because both earlier forms (diff with process substitution, then mktemp) printed a false zero in a sandbox. PRE-VALIDATION IN A THROWAWAY WORKTREE caught a guard this issue had declared position-independent: keyTableIn took the first "| key | does |" in the README, there are two, and the move put the session table first, so it silently checked the wrong table. The review key table now has a review-keys marked span found via markedSpan, and Constraints names both position-sensitive guards (PQ-3). Two position words ("the question hatches above", "the screen above") pointed across moved sections and are now links, pinned by TestREADMEAnchorsResolve. The command table is quoted from the registry span and pinned over derivedDocs; the stale no-deck sentence #50 left is corrected and added to the superseded-claims guard; the root README and the atlas account of the command table are updated. MUTATION-VERIFIED in every shape each claim has, 7 of 7 red with a control run green: three link-target renames, dropping the ? row from the review keys, deleting the review-keys marker, drifting one README command-table row, and reverting the corrected no-deck sentence. build, vet, vet -tags conformance and gofmt clean; go test ./... fully green; run-merge-checks.sh passed.; review verdict: SHIP

### 2026-09-11 — reorganised, and a guard this issue declared safe was not

The README now leads with the session. Twelve edits; every other line is carried
over verbatim, and the diff command in the Spec shows exactly those and nothing
else.

**Pre-validating in a throwaway worktree caught what reading had cleared.** The
Constraints section said every README doc-sync guard located its target by a
literal header or a marked span, so moving sections could not weaken them — and
named `keyTableIn` as checked. Applying the move and running the suite failed
`TestREADMEKeyTableNamesEveryLiveKey`: `keyTableIn` used `strings.Index` on
`| key | does |`, the README has two such tables, and the move put the session's
first. The check had confirmed the header was *literal*, never that it was
*unique*. Plan-quality's PQ-3 had flagged the same overstated claim for a second
guard — the superseded-claims check's 400-character window — and both are now
named in Constraints.

**Two position words were broken by the move, which a line diff cannot see.**
*"the two question hatches above"* and *"`/stats` is the screen above"* pointed at
sections that now sit below them. Both became links, and
`TestREADMEAnchorsResolve` pins every in-page link.

**Every new guard mutation-verified in each shape its claim has** — 7 of 7 red,
then a control run with all four green:

| mutation | guard | result |
|---|---|---|
| rename `## Asking questions` | `TestREADMEAnchorsResolve` | red |
| rename `## Is any of this working` | `TestREADMEAnchorsResolve` | red |
| rename the deck-question heading | `TestREADMEAnchorsResolve` | red |
| drop the `?` row from the review keys | `TestREADMEKeyTableNamesEveryLiveKey` | red — *does not name "bad question", which \*play.Cloze offers* |
| delete the `review-keys` opening marker | `TestREADMEKeyTableNamesEveryLiveKey` | red (fatal) |
| drift one README command-table row | `TestDocsQuoteTheCommandList` | red — names `README.md` |
| revert the corrected no-deck sentence | superseded-claims guard | red |

The last row reverts the *real* correction rather than inserting the stale
sentence somewhere new: the window-based guard passes a stale claim placed beside
unrelated qualifying words, and the correction itself contains "not a deck yet".

The atlas's account of the command table now says both pages quote it.

**The Spec's own verification command failed twice in my sandbox, silently.**
`diff <(…) <(…)` failed on `/dev/fd`, and a `mktemp` rewrite failed on the system
temp directory — and both still printed a count of **zero**, a false "no changes"
rather than an error. The command the close review is told to run is now pipes
only, and it was proven in the most restricted shell available: 15 lines removed,
41 added, each mapped to one of the twelve edits.

**Verified on the branch:** build and vet clean, gofmt clean; go test ./... fully green; the merge gate passed (run-merge-checks.sh, release-stamp check); the pipe-only diff in the Spec shows 15 lines removed and 41 added, every one mapped to the twelve listed edits; seven mutations each red, with a control run green; the stale no-deck claim is absent from the README, the root README and the atlas.

### 2026-09-11 — closed SHIP; four advisories and their dispositions

- **BR-1** (slug rule unstated in the plan): the rule is stated where it runs, in
  `TestREADMEAnchorsResolve`'s comment — lower-case, spaces to hyphens, drop
  everything but letters, digits, hyphens and underscores — and exercised by all
  three link targets, each mutation-verified.
- **BR-2** (a trailing comment in the superseded-claims list now sits on the #51
  line): real, and mine. The edit anchored on `"persists when\ngiven.",` as a line
  *prefix*, so the new entry was inserted ahead of that line's own comment.
  Cosmetic, in a test file. Deferred: a post-close change under `cmd/` forces a
  re-close, which is disproportionate for a comment. Fixed with BR-4 in a follow-up.
- **BR-3** (the anchor guard counts any `#` line as a heading, and does not model
  GitHub's `-1` suffix for duplicate headings): lenient rather than strict, and
  unreachable today — no fenced `#` lines, no duplicate slugs. Noted for when the
  README gains a shell snippet with comments.
- **BR-4** (`play_loop.go:35` still says an ordinary directory has no deck): outside
  this window — a #50 residual surviving in a code comment. With BR-2 in the
  follow-up.

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
