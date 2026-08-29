---
id: 000032
status: open
deps: [tools#29]
github_issue:
created: 2026-08-29
updated: 2026-08-29
estimate_hours:
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

## Spec

**The fix belongs at the SEAM, not on that line.** `stderr` is wrapped in
`crlfWriter` for the ask path (`replraw.go`) and the review loop
(`play_loop.go`) — but NOT for `replayInPlace`. So the line-ending decision is
made per-write by whoever happens to be writing, which is exactly the shape
`#16` removed everywhere else by putting one writer over the whole stream.

**There is a pre-existing sibling**, and it is the reason this is a seam problem
rather than a typo: `playAnnounced`'s error line (`cmd/define/main.go`) has the
same bare `\n` and predates `#29`. A fix that touches only `reportVoice` leaves
its twin, which is the fix-the-instance-not-the-class pattern
`workshop/lessons.md` records repeatedly.

**The candidate shapes differ in size and should be chosen deliberately:**

1. Wrap `stderr` in `crlfWriter` for the raw editor's replay path too, so the
   raw loop has ONE stream policy rather than three sites that each decide.
2. Make `playAnnounced`/`reportVoice` take a writer that is already correct for
   the mode, pushing the decision to the boundary that knows the mode.

Shape 2 is closer to `#16`'s stated rule — *"EVERY byte of session output goes
through crlfWriter"* — but touches more call sites.

## Done when

- [ ] A `/pron` miss reported while the raw editor owns the terminal renders at
      column 0, asserted rather than observed.
- [ ] `playAnnounced`'s error line is fixed in the same change — the class, not
      the instance.
- [ ] The rule is stated once, at the seam, so a fourth site cannot decide for
      itself.
- [ ] A pty conformance row covers it, since this is a raw-terminal surface and
      `workshop/lessons.md` says such a surface gets one in the same milestone.

## Plan

- [ ] Claim, then design via `sdlc start-plan`.

## Log

### 2026-08-29

Filed from `#29`'s close review round 4. The reproduction above is the
reviewer's scratch test, not a hypothesis.
