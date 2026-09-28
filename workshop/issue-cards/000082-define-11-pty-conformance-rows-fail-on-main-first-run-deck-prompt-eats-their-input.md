---
id: 000082
status: open
created: 2026-09-27
updated: 2026-09-27
estimate_hours:
github_issue:
---

# define: 11 pty conformance rows fail on main — first-run deck prompt eats their input

## Problem

Measured 2026-09-27 on main (`71468f0`), `go test -tags conformance -run TestPTY
./cmd/define/` outside the sandbox (inside it every pty row SKIPs on "operation
not permitted" and the suite reads green). 11 rows fail; they fail identically
with no product change, so this is test-harness rot, not a product regression.

Both causes are the first-run deck question (#50, `deckPrompt`: *"… is not a
deck yet. Create one here? [y/N]"*), which appears whenever `define` starts in a
directory that is not a deck — and every pty row starts in a fresh `t.TempDir()`.

**Cause 1 — `seedDeck` never creates its deck (5 rows).** It runs
`define --no-audio sycophantic` without `--here`, so the lookup succeeds (the
helper's own `sikəˈfan(t)ik` check passes) but the answer to the deck question
is EOF → decline → nothing saved. `--play` then reads an empty deck.
`seedDeckN` had the identical defect and #80 fixed it (`--here`); `seedDeck`,
its single-word sibling, was not swept.

- `TestPTYPlayRendersEveryLineAtColumnZero` — "--play never offered the seeded word"
- `TestPTYPlayCorrectAnswerNeverRevealsIt` — "grading keys were not offered up front"
- `TestPTYPlayGradeFirst` — same
- `TestPTYPlayKeepsTheAlternateScreenAcrossAReveal` — "never took the alternate screen"
- `TestPTYPlayResizeRepaints` — "never drew its bar"

**Cause 2 — the editor rows' first typed line answers the deck question (6
rows).** `startDefine` / `startDefineWithEnv` launch the editor in a fresh
directory; the prompt reads one line from the tty, so the test's first input —
`sycophantic\r`, `/`, `?why\r` — is consumed as the answer ("not saving in this
directory") and never reaches the editor.

- `TestPTYSuggestionAndAcceptance` — no history to suggest from
- `TestPTYCommandMenuAppearsAndClears` — "typing / did not draw the menu"
- `TestPTYCtrlCMidAnswerKeepsTheSession` — "the answer never streamed"
- `TestPTYMouseTrackingIsAskedForAndGivenBack` — "never enabled mouse reporting"
- `TestPTYTranscriptIsPrintedOnExit` — "the word was never defined"
- `TestPTYWithoutMouseBehavesAsBefore` — "the lookup did not answer"

#81's `TestPTYAWideDiagnosticIsWrappedNotClipped` hit cause 2 and worked around
it inline (decline with a bare `\r` first) — one row's patch, not the class.

Why it rotted unseen: these rows run in nothing automated (#37), and in the
sandbox they skip rather than fail.
