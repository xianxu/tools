---
id: 000082
status: open
deps: []
github_issue:
created: 2026-09-27
updated: 2026-09-27
estimate_hours:
card_mirror: '0f7de70e5dbb004cd5daa9eacd7ecfecbfc4204d' # card fields mirrored from issue-cards; edit via sdlc
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

## Spec

Fix the class at the two helpers, not per row (ARCH-PURPOSE: 11 instances, two
causes):

- `seedDeck` passes `--here`, as `seedDeckN` now does.
- The editor launch path settles the deck question before the test types. The
  choice is which settlement each row's subject wants — a row about the editor
  (menu, suggestion, mouse, transcript) should not depend on deck policy at all,
  so a launch that pre-decides it (`--here`, or a pre-created deck directory)
  keeps the prompt out of its input stream. The deck question itself keeps its
  own rows (`deckperm_e2e_test.go`, `activity_conformance_test.go` answer it
  explicitly) and must stay covered.
- Remove #81's inline workaround once the helper covers it, so the class has
  one fix.

Open: pre-deciding with `--here` makes these rows create a deck they did not
ask for; if any row's assertion depends on "not a deck" (e.g. capture being
off), it answers the prompt explicitly instead. Check per row, not by
assumption.

## Done when

- All `TestPTY*` rows pass on a real pty (run unsandboxed, `-tags conformance`),
  with the command and exit status recorded in the Log.
- The deck question's own coverage is unchanged — its dedicated rows still pass
  and still exercise the prompt.
- No row answers the deck prompt ad hoc for a reason the helper now owns
  (#81's inline decline removed or justified).

## Plan

- [ ] `seedDeck` → `--here`; rerun the 5 `--play` rows
- [ ] settle the deck question in the editor launch helper; rerun the 6 rows
- [ ] remove #81's inline workaround; full `TestPTY` run, record in Log
- [ ] `sdlc close`

## Log

### 2026-09-27

Filed after #80/#81 shipped. #80 fixed `seedDeckN` (`--here`) and the board
row's press-without-release click, which took the failing set from 13 to 11.
Per-row failure messages above are from this date's run.
