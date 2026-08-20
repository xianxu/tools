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
