---
id: 000064
status: open
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
github_issue:
---

# define: adaptive bilingual interaction level, learned or instructed, remembered per deck

## Problem

define's model features assume one learner profile: an advanced English reader
learning English words. The console-question prompt says "You are helping someone
build their English vocabulary" and carries no language or level. The learner
model grades on A2–C2 and never names the deck's language. With a Spanish deck
that is wrong for the user, a beginner, and for their daughter at Spanish 3. They
need very different amounts of English, and the right amount changes as they
progress.

#61 adds a per-deck `/bilingual on|off` switch and English help while practising.
That is a binary version of what is needed. This issue generalizes it.
