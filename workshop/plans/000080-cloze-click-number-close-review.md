# Boundary Review — tools#80 (whole-issue close)

| field | value |
|-------|-------|
| issue | 80 — define: click a cloze option's number to answer it |
| repo | tools |
| issue file | workshop/issues/000080-cloze-click-number.md |
| boundary | whole-issue close |
| milestone | — |
| window | ffc63ddf2f6b8e722692c91dbfa92e625fcc424f..03dc4afa91f8cf1d1979c95d1d2c6d2fbff18491 |
| command | sdlc close --issue 80 |
| reviewer | claude |
| timestamp | 2026-09-23T18:30:47-07:00 |
| verdict | SHIP |

## Review

Tests are green and I've finished the review. Here's the verdict.

```verdict
verdict: SHIP
confidence: high
```

Every Done-when row is met and pinned by the test it names. The design keeps each concern in one owner. The form records where it drew its numbers (`promptBuilder.pickable` → `Presentation.Options`), so main never works out the option layout a second time. Checking that a click belongs to the current question uses the absolute buffer line, the same rule #67 uses for passages. Both hazards the issue named are handled: the pure `Apply` refuses `InputMark` on a Graded question, and a click goes through the same `graded` tail as the digit key, so the verdict, the record and the reveal can't drift apart.

At head, outside the sandbox: `go vet` is clean; `go test ./...` passes for `define`, `play`, `schedule` and `store`; and `TestPTYPlayClickingAnOptionNumberAnswers`, `TestPTYPlayBoardIsDrawnAndClickable` and `TestPTYPlayChoiceOffersOptionsAndRecordsTheAxis` pass under `-tags conformance`. The only failures inside the sandbox were pty-open "operation not permitted" errors, which aren't caused by this diff. I also wrote a throwaway test for a question the listed tests don't cover: a multiple-choice gloss long enough to wrap. Its number target survives on the first row (`layoutOutput` projects it rather than dropping it), and the next option's `[2]` still lands on the right row.

1. **Strengths**
   - `play/session.go`: pulling out `graded(s, q, verdict)` and making `Grade` call `Pick` means a key and a click can't disagree. `TestAPickedOptionIsGradedExactlyAsItsDigit` checks both forms, every option, with and without a peek, comparing the whole session and all outcomes.
   - `play_loop.go` `formCell`: the Picker shape checks the region kind and `asking.holds(hit.line)`. That ties a click to its question structurally, with no question index to keep in sync.
   - The kind registry stays derived. `regionAnswers` sits beside `regionPlaysAudio`, and the editor's actionability guard defers answering kinds to a named sitting row instead of silently skipping them.
   - The loop test drives the hazards in the order a learner meets them, with the store as witness: word column, gap column, clicks after the answer (the option list and the `you chose` line), and the previous question's stale `[2]`. Bounding the key sends turned a hang into a failure.
   - Side quest: `seedDeckN --here` and the press+release board click fix a pty harness that was quietly testing an empty deck.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `cmd/define/play_loop.go:1596`: `gofmt -l` flags this file because of a double blank line before `lineRange`.
   - `cmd/define/README.md:492`: the `click` row in the keys table still says "Anywhere else, a click plays the word". The `1`–`4` row now covers clicking a number, but this row should mention it too.
   - The keys line is 84 columns and wraps at 80. The Log already flags this to the operator; it's noted here only so the verdict records it.
   - `play_loop_test.go:1163` still builds its fixture with the old `"1  "` prefix. It's consistent with itself, but it's no longer the shape `optionLine` writes.

5. **Test coverage:** the pure layer covers pick equals digit, the Graded refusal, out-of-range picks, and spans on the prompt only. The loop layer covers staleness, clicks after the answer, the word/number boundary, the recorded review matching, and a wrapped stem with colour on and off. The pty row is a real SGR click. The listed tests have no row for a wrapped multiple-choice gloss; my throwaway check showed it works, and a permanent row would be cheap.

6. **Architecture**
   - ARCH-DRY: pass.
   - ARCH-PURE: pass. `Picker`/`graded` are pure; the loop only converts bytes to cells and checks the line range.
   - ARCH-PURPOSE: pass. Both numbered forms are in, and the registry guards derive from `numRegionKinds`.
   - ARCH-MOCK: pass. The pty conformance row runs against the real terminal path.
   - ARCH-CONSTRAINTS: pass. Clicks are interactive and cost O(options).
   - ARCH-SECURE: pass. No untrusted input; `optionRegions` bounds-checks the spans.
   - ARCH-ORDER: pass. The Graded refusal is in the pure transition, and `asking` is replaced before any key is read for a new prompt.
   - ARCH-FUNERAL: pass. Nothing durable is created, and click regions follow the existing buffer lifetime.
   - For #75: the cloze prompt will then have three gestures by target (stem word marks, option word speaks, number answers). This diff already keeps them disjoint by region kind.

7. **Plan revisions:** none needed. The existing Revisions entry already describes `Picker` and `lineRange` accurately.

```findings
findings:
  - id: new
    severity: Minor
    family: gofmt-clean
    title: |
      play_loop.go is not gofmt-clean (double blank line before lineRange)
    detail: |
      gofmt -l lists cmd/define/play_loop.go; the blank-line pair precedes the lineRange type at the file's end.
  - id: new
    severity: Minor
    family: readme-surface-complete
    title: |
      README keys table "click" row still says a click elsewhere only plays the word
    detail: |
      The 1-4 row mentions clicking [1]-[4], but the click row's "Anywhere else, a click plays the word" now omits option numbers answering.
  - id: new
    severity: Minor
    family: stale-fixture-shape
    title: |
      play_loop_test.go:1163 fixture still uses the pre-#80 "1  " option prefix
    detail: |
      Self-consistent so it still passes, but no longer the shape optionLine writes.
```
