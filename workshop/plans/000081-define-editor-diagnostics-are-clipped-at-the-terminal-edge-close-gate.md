---
gate: boundary-review
issue: 81
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-23T17:12:26-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Diagnostic tokens wider than the terminal are still clipped (wrapText never breaks a word)
          detail: 'wrapText (render.go:667) puts an over-long word on its own row and Paint clips it; a real provider 400 body''s first token is about 71 cells, so below about 72 cols the provider message is cut again. Instances in the window: the JSON body token and the POST url token. Fix: hard-break tokens over cols in the diagnostics path only, and test at 40 cols with a no-space body.'
          family: wrap-leaves-overlong-token-clipped
          round: 1
        - id: BR-2
          severity: Important
          title: No in-process test for Diagnostics; clause 2 (every row fits width) is never asserted
          detail: The only test is darwin+conformance tagged and checks tail presence only. Add a pure newLiveScreen test writing a long message through Diagnostics() and asserting every buffer row is at most cols cells, including an overlong-token case.
          family: done-when-clause-untested
          round: 1
        - id: BR-3
          severity: Minor
          title: Wrapping a CRLF-terminated diagnostic drops its trailing CR while fitting lines keep it
          family: wrap-drops-carriage-return
          round: 1
      recipe: small-diff-review
      blocked: true
---

# Gate ledger — tools#81 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-23T17:12:26-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `wrap-leaves-overlong-token-clipped` Diagnostic tokens wider than the terminal are still clipped (wrapText never breaks a word)
  wrapText (render.go:667) puts an over-long word on its own row and Paint clips it; a real provider 400 body's first token is about 71 cells, so below about 72 cols the provider message is cut again. Instances in the window: the JSON body token and the POST url token. Fix: hard-break tokens over cols in the diagnostics path only, and test at 40 cols with a no-space body.
- **BR-2** [Important] `done-when-clause-untested` No in-process test for Diagnostics; clause 2 (every row fits width) is never asserted
  The only test is darwin+conformance tagged and checks tail presence only. Add a pure newLiveScreen test writing a long message through Diagnostics() and asserting every buffer row is at most cols cells, including an overlong-token case.
- **BR-3** [Minor] `wrap-drops-carriage-return` Wrapping a CRLF-terminated diagnostic drops its trailing CR while fitting lines keep it

## Open findings

- **BR-1** [Important] `wrap-leaves-overlong-token-clipped` Diagnostic tokens wider than the terminal are still clipped (wrapText never breaks a word)
- **BR-2** [Important] `done-when-clause-untested` No in-process test for Diagnostics; clause 2 (every row fits width) is never asserted
- **BR-3** [Minor] `wrap-drops-carriage-return` Wrapping a CRLF-terminated diagnostic drops its trailing CR while fitting lines keep it
