---
id: 000077
status: working
deps: []
github_issue:
created: 2026-09-18
updated: 2026-09-18
estimate_hours:
started: 2026-09-18T11:29:09-07:00
flow: {kind: quick, provenance: inferred, spec: "9ca556cc", done: "c819ada4"}
---

# define: confirm light detection in a real light terminal

## Problem

#70's detection (OSC 11 asked at raw-mode entry, the reply read as a
`KeyBackground`) is proven live only for a DARK terminal: on 2026-09-18 the
operator's terminal, after `/scheme auto`, reported `scheme dark (detected)`.
LIGHT detection is evidenced only by the modelled pty terminal
(`TestPTYBackgroundDetection/light`) and the in-process reply
(`TestRawEditorBackgroundReplyRepaints`) — not by a real terminal. This is the
ARCH-MOCK live conformance check #70 owes (see #70's Log and the atlas's
*Terminals checked by hand*).

## Spec

In a real terminal set to a LIGHT appearance: run `define`, `/scheme auto`, then
`/scheme`. Record the terminal app, its appearance, and the exact report line in
the atlas's *Terminals checked by hand*. `light (detected)` closes this; any other
answer (`dark (detected)` on a light background, or `(default: …)`) is a
detection bug or an unanswering terminal — record it and open the fix.

## Done when

- The atlas records a real light terminal's `/scheme` line after `/scheme auto`,
  with the terminal app named, and the "Owed" sentence there is removed.

## Plan

- [x] Record the operator's light-terminal check in the atlas's *Terminals
  checked by hand*, and drop its "Owed" sentence.
- [x] Log the check here, stating exactly what was and was not reported.

## Log

### 2026-09-18

The operator verified light detection in a real light terminal against this
issue's criterion (`/scheme auto`, then `/scheme` → `light (detected)`), reported
as "#77 verified". Not reported: the terminal app's name and the verbatim line;
the atlas records the check as verified against the criterion, app not named.
With #70's `dark (detected)`, detection is observed live in both appearances.
The atlas's "Owed" sentence is replaced by this record.

Close review round 1 (BR-1, blocking): the Done-when requires the terminal app
named, and both hand-check records — this one and #70's dark one — lacked it.
The operator named it: both checks ran in Terminal.app. The atlas now records
each with app, appearance and report line: dark `scheme dark (detected)` (quoted
by the operator), light verified against the criterion `scheme light (detected)`
(reported as verified, not quoted). Both lines carry their `scheme ` prefix.


Close review round 2 (BR-3, blocking): the atlas hand-quoted the `/scheme`
report with nothing deriving it — the missing `scheme ` prefix had already shown
the drift. Now both quotations sit in marked spans that
`TestAtlasQuotesTheSchemeReportItPrints` composes from `describeScheme`, and
`describeScheme` joins the atlas's Re-check triggers. The "/scheme auto removed
the saved file" clause is re-attached to the dark run that observed it.
