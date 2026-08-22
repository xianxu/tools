---
id: 000009
status: open
deps: ["tools#3"]
github_issue:
created: 2026-08-20
updated: 2026-08-20
estimate_hours:
---

# news seam: Google News RSS client with a stateful fake

## Problem

Generated questions are better when the sentence is real and current. That
needs a news source that can be fetched without a key and without scraping.

## Spec

Google News **RSS**, behind a seam.

- `news.google.com/rss/search?q="<word>"&hl=en-US&gl=US&ceid=US:en`.
- **Measured before planning:** 41–100 items per word; headlines containing the
  word range 12 (`defenestrate`) to 99 (`ephemeral`). Structured XML, key-free.
- **The SERP is not an option, and this was measured, not assumed:** an earlier
  probe of `google.com/search` returned a 91 KB JS shell (`enablejs`) with zero
  usable content. Scraping it needs a headless browser.
- Terms note: the feed is licensed for personal, non-commercial feed-reader use.
  A personal vocabulary tool fits; do not redistribute the content.
- Results **cached in the store**, so review works offline and a session does not
  hit the network per question.
- Stateful fake serving canned RSS; live conformance check on the feed shape,
  on-demand like the other seams.

## Done when

- [ ] Fetches, parses and caches; a second request for the same word is served
      from cache.
- [ ] The parser survives a malformed feed without taking down the session.
- [ ] Live conformance asserts the feed shape and non-trivial coverage.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

## Revisions

### 2026-08-22 — the consumer changed; NOAD's own examples join the feed

**Reason.** #10 now authors finished items rather than harvesting a word pool, and
the operator raised general web search as an alternative usage source.

**Delta.**

- **General Google search stays out, on the measurement already recorded above** —
  the SERP is a 91 KB JS shell needing a headless browser. That finding is why this
  issue is RSS-shaped, and it has not changed.
- **NOAD's own example sentences are a second usage source, and a free one.** The
  entry is already fetched, already parsed, offline, editorially curated, and
  register-correct. It complements the feed exactly where the feed is weakest: the
  measured thematic collapse (10 of 14 `sycophantic` headlines were about AI
  chatbots) is a *current-events* artifact, and the dictionary's examples are not
  current, which is the point. Authoring gets both.
- **What is cached is unchanged** (raw feed items), but the downstream consumer is
  now the authoring step, not a question at review time.

**Unchanged.** Feed shape, the personal-use terms note, the stateful fake, the
malformed-feed requirement, and live conformance on shape and coverage.
