---
id: 000079
status: open
deps: []
github_issue:
created: 2026-09-20
updated: 2026-09-20
estimate_hours:
---

# define: top up cloze items per word so a question stops repeating

## Problem

Operator, 2026-09-20:

> make a task to also periodically generate more cloze style questions. I find
> repeating questions easier to remember, just the question, not necessarily
> grasping the word or concept.

**The reading this issue takes (unconfirmed — see Spec, first open question):** the
learner sees the *same* cloze question for a word again and again, and what they
end up remembering is that question — the sentence, its blank, the shape of the
answer — rather than the word. The fix is more questions per word, generated
over time.

That is what the code does today. A word is authored **once and never again**:

- `runAuthoring` skips any word that already has an item (`existing` non-empty,
  `harvest.go:342`), and `pendingWords` (`background.go:160`) counts a word as
  needing work only when it has **no** band or **no** item. One item and the word
  is finished forever ("assigned once, re-read forever", `atlas/define.md`).
- `pickCloze` (`cloze.go:151`) takes the **newest usable** `FormCloze` item. The
  per-day seed (`seedFor(key, day)`) only shuffles the OPTION ORDER — so the
  stem, the blank and the answer are identical every time the word comes up; only
  the position of the right answer moves.
- `store.ItemCap = 4` (`store/item.go:217`) already reserves room for four items
  per word, but its own comment says the truncation branch is "UNREACHABLE from
  production today: SetItems' only non-test caller writes exactly one item".

So the storage, the cap and the background job that would do the work all exist.
Nothing writes a second item, and nothing would pick between two if it did.

## Spec

Not designed yet — this is the task as filed, with the shape the code suggests. It
needs a brainstorm before a plan.

### Open questions, in the order they change the design

1. **Is the reading above right?** The quote can also mean the opposite: that
   repetition *works* for this learner and they want more cloze questions
   available for the words they already cover. Both readings want the same verb
   (more cloze items) but disagree on the rotation policy — **vary the stem each
   time** vs **repeat a stem until it is known, then introduce another**. Ask
   before designing the sitting side.
2. **What triggers a top-up?** The existing periodic mechanism is #54's
   background job: at session start, then every `bgThreshold` (10) lookups
   (`stepBackground`, `background.go`). Candidates for "which words get another
   item": (a) any word below a target count, oldest item first; (b) words the
   learner has actually been asked, more than once, against a single stem — the
   direct measure of "repeating question", *if* the events log records cloze asks
   (unverified; first plan step); (c) time since the newest item's `At`.
3. **How many, and at what cost?** `ItemCap` is 4. A top-up is a model call per
   item plus entail plus up to three vetoes (the same 2–5 calls a first item
   costs), drawn on `bgBudget` (60) — which is already sized to roughly ten
   fresh words. Top-up work must not starve first-time authoring: a word with **no**
   item outranks a word wanting a second.
4. **Sitting side: rotation.** `pickCloze` returns the first usable item, so a
   second item would simply *replace* the first as "newest" rather than sit
   beside it. Choosing among items needs a rule that is still deterministic under
   a fixed seed (the existing `seedFor(key, day)` contract — same word, same
   question all day, a different one tomorrow) and still offline and free at
   sitting time (`TestAClozeSittingNeverReachesForTheModel` must stay green).

### Constraints already visible in the code

- **A new item must differ from the old ones.** The author prompt sees the word,
  gloss, facts and learner model — not the stems already written — so it would
  happily rewrite the same sentence. The prompt has to receive the existing stems,
  and `harvest.go:400` seeds distractor selection with `seedFor("harvest-options",
  c.Word)`, which is constant per word: a second item would inherit identical
  distractors unless that seed varies. A repeated *option set* is the same
  memorisation problem one layer down.
- **Prompt changes invalidate goldens and cassettes** (`llm.RequestHash`,
  `internal/llm/render.go:65` — the same constraint #68 recorded), so the prompt
  change is deliberate and re-recorded, not incidental.
- **Overlap with #68 and #73**, both open and both editing `runAuthoring` and the
  `bgBudget` arithmetic. #68 changes where a stem comes from (passage / NOAD /
  authored) and re-sizes `bgBudget`; #73 records failed attempts so they are not
  re-asked. A top-up loop should read #73's record rather than re-derive it, and
  #68's candidate list is the natural source of *different* stems. Sequence, or
  land on top of them — decide at brainstorm, not by collision at merge.
- **`FormSentence` (#13)** shares the same `ItemCap` and store; top-up counts
  `FormCloze` items only, or the cap is spent by a form this issue does not
  generate.

### Out of scope

- Changing how a question is *asked* (the board, option layout, reveal) — this
  issue is about which stored item is served and how items get written.
- Real-sentence sources — #68.
- A user-facing flag or command to author more on demand; the background job is
  the delivery mechanism, and `--harvest` picks the behaviour up through the
  shared core (`harvestDeck`) without a new surface.

## Done when

- A word with fewer than the target number of cloze items, and eligible under the
  chosen trigger, gains one more from a background job — asserted through the LLM
  fake, counting calls, with **no** change to a word that is at target.
- The new item's stem differs from every existing stem for that word, and its
  distractors are not identical to the previous item's — pinned by a test whose
  fake model tries to return the old sentence.
- A word with no item is authored before any word is topped up, under a budget
  too small for both.
- A sitting serves the stored items with a documented rule that is deterministic
  under a fixed seed, and a word with two items does not show the same stem twice
  in a row across days. The sitting still reaches for no model
  (`TestAClozeSittingNeverReachesForTheModel` and its cold/warm-cache siblings
  stay green).
- `bgBudget`'s and `harvestLimit`'s call-arithmetic comments state the new worst
  case, and a test holds that the background job still covers `bgThreshold` fresh
  words.
- The store cap holds: topping up never exceeds `ItemCap`, counted per `Form`, and
  the prune keeps the items the rotation still needs.
- `atlas/define.md` records the trigger, the rotation rule and why the repeat was
  a problem.

## Plan

- [ ] brainstorm — open question 1 first (which reading), then trigger and
      rotation; `sdlc issue sync --issue 79` when it lands
- [ ] verify whether the events log records cloze asks per word — decides trigger
      (b) vs (a)/(c)
- [ ] `sdlc start-plan`, then the durable plan in `workshop/plans/`
- [ ] the store/selection side: rotation over several items
- [ ] the authoring side: top-up trigger, prompt that sees existing stems, varied
      distractor seed, budget priority
- [ ] atlas, then `sdlc close`

## Log

### 2026-09-20

Filed from an operator request in a session on `main`. Read before writing:
`background.go`, `harvest.go`, `cloze.go`, `store/item.go`, the atlas's
*Harvested facts* and *Background preparation* sections, and #68 / #75 / #22 for
overlap. The repeat is a property of the code, not a bug in it: one item per word,
seeded only for option order, is exactly what `atlas/define.md` documents as
intended ("assigned once, re-read forever"). This issue changes that decision, so
the atlas sentence is part of the deliverable.

Not claimed and not started — the request was to file the task.
