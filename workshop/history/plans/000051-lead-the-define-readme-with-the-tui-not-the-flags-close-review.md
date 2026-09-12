# Boundary Review — tools#51 (whole-issue close)

| field | value |
|-------|-------|
| issue | 51 — lead the define README with the TUI, not the flags |
| repo | tools |
| issue file | workshop/issues/000051-lead-the-define-readme-with-the-tui-not-the-flags.md |
| boundary | whole-issue close |
| milestone | — |
| window | b87bb119c9bdfb381c8edfdd3354d186a2eabb22..ee906617b38ee13d84d81f8b7dc4602246f504c2 |
| command | sdlc close --issue 51 |
| reviewer | claude |
| timestamp | 2026-09-11T17:44:27-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The boundary delivers exactly what the Spec enumerates. I reproduced the Spec's line-multiset diff against the pinned base independently: 15 lines removed, 41 added, every one mapping to a row of the twelve-edit table, with the README growing from 873 to 906 lines as stated. The heading order at head matches the operator-approved order verbatim. The corrected no-deck sentence agrees with the code paths it describes (`runStats`, `runPlay`, `forgetWord` exit 1 only on a nil deck, which `openStore` produces only under `DEFINE_NO_CAPTURE` or a failed Getwd). All four guard changes are real: in a scratch copy I ran six mutations and every one went red, with a control run green. gofmt, vet, build and the `cmd/define` suite are clean. Nothing blocks SHIP; three Minor notes follow.

**Strengths**

- The verification recipe is reproducible by a stranger. The pipes-only multiset diff in the Spec ran unchanged against the pinned base and produced exactly the claimed counts.
- `keyTableIn` now locates by a unique marker via the existing `markedSpan` helper (`cmd/define/doc_sync_test.go:605`), and the mutation that marks the wrong table fails loudly by naming the missing review keys. The comment records the discovery honestly, including that reading had cleared it and running caught it.
- `TestDocsQuoteTheCommandList` iterates `derivedDocs` (`doc_sync_test.go:369`) rather than a second hand-typed pair of paths, so a third derived page joins the pin by being added once.
- `TestREADMEAnchorsResolve` (`doc_sync_test.go:814`) has an under-reading guard: fewer than three in-page links is a fatal, so a broken regex cannot pass by matching nothing.
- The superseded-claims enumeration gained its entry in the same commit that narrowed the promise, which is the rule that array's comment states and which prior rounds had violated.

**Critical findings**

None.

**Important findings**

None.

**Minor findings**

- `cmd/define/deckasker_test.go:356-357`: the `"persists when\ngiven."` entry lost its `// the atlas's unconditional form` comment, which now dangles at the end of the new entry's line. Move it back to line 356.
- `cmd/define/doc_sync_test.go:835-838`: the heading scan treats any line starting with `#` as a heading, so a shell comment inside a fenced code block would register as an anchor, and repeated headings are not given GitHub's `-1` suffix. Both make the guard lenient rather than strict. Neither is reachable today (I checked: no fenced `#` lines, no duplicate slugs), so note for the next time the README grows a shell snippet with comments.
- `cmd/define/play_loop.go:35` (outside the window): the comment "an ordinary directory simply has none" is the pre-#50 model of a nil deck. The README this issue corrected now says a non-deck directory has an EMPTY deck. Same stale-claim class the issue swept, but in a code comment rather than a doc surface.

**Test coverage notes**

Mutations run in a scratch copy, each against the named guard:

| mutation | result |
|---|---|
| revert the corrected no-deck sentence | red |
| rename `## Asking questions` | red |
| dead anchor in a link | red |
| drift one README command row | red, names README.md |
| mark the session key table instead of the review one | red, names three missing keys |
| delete both review-keys markers | red, fatal |

I ran `go test ./cmd/define/...`, `go build ./...`, `go vet`, and gofmt myself. I did not re-run the full `go test ./...`; the diff touches only `cmd/define`, and the Log reports it green.

**Architectural notes**

- ARCH-DRY: pass. Both pages consume the registry span through one list; `markedSpan` reused. No existing markdown-slug helper exists in the repo (the store's `Slug` is a path-safe word slug, a different rule), so the inline slug closure is not a duplicate.
- ARCH-PURE: pass. The slug rule is a pure closure; the guards read only version-controlled fixtures.
- ARCH-PURPOSE: pass. Shadow-sweep: the stale sentence is absent from the root README, the define README and the atlas; both position words became links; the command table is consumed by both derived pages. The one remaining sibling is the code comment noted above.
- ARCH-MOCK: N/A. No external binary or service is touched.
- ARCH-CONSTRAINTS: N/A. Test-time whole-file reads of a 906-line document.
- ARCH-SECURE: N/A. Inputs are in-repo docs; a missing marker fails visibly via `t.Fatalf`.
- ARCH-ORDER: N/A. No state carried across events. The one ordering hazard in play, section order versus guard locators, is exactly what the marked span removed, and the wrong-table mutation proves it.

**Plan revision recommendations**

None. The Non-goals' "four test edits" matches the diff (three in `doc_sync_test.go`, one in `deckasker_test.go`), and the Revisions entry already records the fourth.

```findings
findings:
  - id: new
    severity: Minor
    family: comment-misattributed
    title: |
      deckasker_test.go:356-357 — the "persists when\ngiven." entry's comment now dangles after the new superseded-claim entry.
    detail: |
      The trailing "// the atlas's unconditional form" moved off its own line onto the new #51 line. Cosmetic; move it back.
  - id: new
    severity: Minor
    family: guard-input-not-scoped
    title: |
      TestREADMEAnchorsResolve counts any line starting with "#" as a heading and does not model GitHub's -1 suffix for repeated headings.
    detail: |
      Both make the guard lenient, not strict. Not reachable today (no fenced "#" lines, no duplicate slugs), so note for when the README gains a shell snippet with comments.
  - id: new
    severity: Minor
    family: superseded-claim-in-code-comment
    title: |
      play_loop.go:35 comment still says an ordinary directory has no deck; since #50 it has an EMPTY deck.
    detail: |
      Outside this window. Same stale-claim class the issue swept across doc surfaces, surviving in a code comment.
```
