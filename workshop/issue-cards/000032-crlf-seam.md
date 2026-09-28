---
id: 000032
status: open
created: 2026-08-29
updated: 2026-08-29
estimate_hours:
github_issue:
---

# reportVoice writes a bare newline to a raw terminal, and the seam is where it should be fixed

## Problem

Carried from `#29`'s close review (BR-12, Minor, `raw-mode-bare-newline`).

`reportVoice` (`cmd/define/main.go`) ends its line with `"\n"`, while every
sibling write in `replraw.go` spells `"\r\n"`. In raw mode a bare `\n` is a line
feed with NO carriage return, so the next line starts at the current column —
the diagonal cascade `#16` built `crlfWriter` for and `atlas/define.md` records.

**Reproduced by the reviewer, not inferred:** `runEditor` driven with
`"jalapeno\r/pron es\r"` against an English-only CDN gives

```
stderr = "define: no es recording for jalapeno; played the en one\n"
```

**Masked today**, which is why it is Minor rather than a bug report: `runEditor`
writes `"\r\n"` immediately after `replayInPlace` returns, so the cascade never
becomes visible. It is a defect waiting for its guard to move.
