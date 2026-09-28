---
id: 000079
status: open
created: 2026-09-20
updated: 2026-09-20
estimate_hours:
github_issue:
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
