---
gate: boundary-review
issue: 44
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-02T14:10:25-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: fitsABoard is also the draw-time boardWhole predicate, so charging chromeGap there fires a false refusal at one height
          detail: |-
            This is the 2nd finding in family `frame-budget-completeness`; PQ-3 fixed the
            charge-vs-emission mismatch inside Paint. The RULE that covers both: chromeGap's
            charge and its emission must agree at every place either is consulted, and every
            consumer of fitsABoard must be named when a term is added to it. Prevalence is 2
            consumers, both live: boardFitsIn (play_loop.go:591-601) is asked at SELECTION and
            at every DRAW, and only the selection role is reasoned about in the plan. At
            termRows == boardRows+promptRows+barRows (the existing table row
            play_loop_test.go:2306, {8,6,1,true}) Paint computes avail=0, declines the gap and
            draws the board whole, while the new fitsABoard returns false so boardPrompt swaps
            in boardRefusal and Enter is held over a whole board. Safe direction, so a note.
            (carried from plan-quality PQ-8, deferred to the boundary review)
          family: frame-budget-completeness
          round: 1
        - id: BR-2
          severity: Minor
          title: the AST-guard step points at dict_symbols_darwin_test.go, which parses no Go source
          detail: |-
            dict_symbols_darwin_test.go compares a C resolver's symbol list against a Go list
            and imports no go/ast. The package's real precedent is repo_guard_test.go:7-8
            (go/ast + go/parser), which also already carries the Fatal-never-Skip discipline
            the plan wants (repo_guard_test.go:46-48). Reuse that walker rather than a new one.
            (carried from plan-quality PQ-9, deferred to the boundary review)
          family: cite-the-code-you-claim
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-02T14:10:25-07:00"
      agent: claude
      findings:
        - id: BR-3
          severity: Important
          title: 'newPinnedScreen''s `gap: chromeGap` is pinned by no test — deleting it leaves the full suite and the conformance suite green'
          detail: |-
            Every gap test builds `screen{pinned: true, gap: chromeGap}` by hand, so the
            production wiring at cmd/define/screen.go:650 is unreachable from any
            assertion. Verified by deleting the line in a scratch checkout: `go test
            ./cmd/define` and `go test -tags conformance ./cmd/define` both pass. Have
            TestAFullBufferStillLeavesARowAboveThePrompt construct through
            newPinnedScreen, or assert newPinnedScreen(...).s.gap == chromeGap.
          family: pin-must-fail-without-the-code
          round: 2
        - id: BR-4
          severity: Important
          title: TestAFooterClickIsUnmovedByTheChromeGap is tautological — it passes for any value of footerTop
          detail: |-
            cmd/define/screen_test.go:1394 queries `sc.footerTop + want` and asserts
            FooterRowAt decodes it, which is FooterRowAt's definition (`row -
            s.footerTop`). Mutating footerTop to `bufRows+promptRows` and to
            `bufRows+gap+promptRows+3` left it green both times. The property is covered
            by the pre-existing TestFooterRowAtNamesTheEntryUnderAClick, which does go
            red — but the designated pin for "the arithmetic that must not be wrong"
            pins nothing. Assert an absolute row instead.
          family: pin-must-fail-without-the-code
          round: 2
        - id: BR-5
          severity: Important
          title: Paint's 35-line doc comment was reparented onto `const chromeGap`, leaving Paint undocumented
          detail: |-
            The chromeGap comment was inserted at cmd/define/screen.go:447 with no blank
            line after Paint's doc block, so Go attaches the whole block ("A frame is
            budgeted in DISPLAY ROWS…") to the constant. Confirmed with `go doc -all
            -u`: Paint prints with no prose. Insert a blank line, or move
            chromeGap/grantedGap above Paint's doc block.
          family: doc-attaches-to-the-wrong-decl
          round: 2
        - id: BR-6
          severity: Important
          title: the indicator guard's screenHostedFiles is an opt-in list, so a new screen-hosted file escapes the rule silently
          detail: |-
            cmd/define/indicator_guard_test.go:18 enumerates the files the rule applies
            to. The list is complete today (main.go is the one-shot path, repl.go the
            piped loop, both correctly exempt), but the guard's own comment cites this
            package's purity guard as precedent and that one is an allowlist: in scope
            unless exempted. Invert to `nonScreenFiles = {"main.go", "repl.go"}` over
            package main's non-test files so a sixth site defaults into the class.
          family: sweep-every-site-of-the-rule
          round: 2
        - id: BR-7
          severity: Minor
          title: the issue's guard Done-when overstates what the by-type predicate reaches
          detail: |-
            workshop/issues/000044-play-chrome.md:179 says the guard "also reaches the
            two sites that call playAnnounced through playRegion — which a callee-name
            walk would have missed". After playRegion's parameter was deleted it takes
            no indicator, so the guard checks exactly the four playAnnounced arguments,
            all of which a callee-name walk would also have reached. The by-type rule is
            still the better one; the sentence describes the pre-deletion tree.
          family: cite-the-code-you-claim
          round: 2
        - id: BR-8
          severity: Minor
          title: README's "the bottom two rows are chrome" is placed in the board section, where it is not true
          detail: |-
            cmd/define/README.md:148 sits under the board paragraphs, but on a board the
            dimmed rows are the prompt (drawn above the grid, as the fence at line 106
            shows) and the bar below it — not the bottom two. Accurate for an ordinary
            sitting; reword or relocate.
          family: cite-the-code-you-claim
          round: 2
        - id: BR-9
          severity: Minor
          title: the chrome-band test hardcodes the dim escape rather than reading it from the palette
          detail: |-
            cmd/define/play_loop_test.go:3648 spells "\x1b[2m" where newPalette(true).dim
            is the owner. Cosmetic — it fails loudly rather than silently — but it is a
            second speller of a sequence the palette exists to own (ARCH-DRY).
          family: one-owner-per-quantity
          round: 2
      blocked: true
---

# Gate ledger — tools#44 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-02T14:10:25-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `frame-budget-completeness` fitsABoard is also the draw-time boardWhole predicate, so charging chromeGap there fires a false refusal at one height
  This is the 2nd finding in family `frame-budget-completeness`; PQ-3 fixed the
  charge-vs-emission mismatch inside Paint. The RULE that covers both: chromeGap's
  charge and its emission must agree at every place either is consulted, and every
  consumer of fitsABoard must be named when a term is added to it. Prevalence is 2
  consumers, both live: boardFitsIn (play_loop.go:591-601) is asked at SELECTION and
  at every DRAW, and only the selection role is reasoned about in the plan. At
  termRows == boardRows+promptRows+barRows (the existing table row
  play_loop_test.go:2306, {8,6,1,true}) Paint computes avail=0, declines the gap and
  draws the board whole, while the new fitsABoard returns false so boardPrompt swaps
  in boardRefusal and Enter is held over a whole board. Safe direction, so a note.
  (carried from plan-quality PQ-8, deferred to the boundary review)
- **BR-2** [Minor] `cite-the-code-you-claim` the AST-guard step points at dict_symbols_darwin_test.go, which parses no Go source
  dict_symbols_darwin_test.go compares a C resolver's symbol list against a Go list
  and imports no go/ast. The package's real precedent is repo_guard_test.go:7-8
  (go/ast + go/parser), which also already carries the Fatal-never-Skip discipline
  the plan wants (repo_guard_test.go:46-48). Reuse that walker rather than a new one.
  (carried from plan-quality PQ-9, deferred to the boundary review)

## Round 2 — 2026-09-02T14:10:25-07:00 (claude) — BLOCKED

### Raised

- **BR-3** [Important] `pin-must-fail-without-the-code` newPinnedScreen's `gap: chromeGap` is pinned by no test — deleting it leaves the full suite and the conformance suite green
  Every gap test builds `screen{pinned: true, gap: chromeGap}` by hand, so the
  production wiring at cmd/define/screen.go:650 is unreachable from any
  assertion. Verified by deleting the line in a scratch checkout: `go test
  ./cmd/define` and `go test -tags conformance ./cmd/define` both pass. Have
  TestAFullBufferStillLeavesARowAboveThePrompt construct through
  newPinnedScreen, or assert newPinnedScreen(...).s.gap == chromeGap.
- **BR-4** [Important] `pin-must-fail-without-the-code` TestAFooterClickIsUnmovedByTheChromeGap is tautological — it passes for any value of footerTop
  cmd/define/screen_test.go:1394 queries `sc.footerTop + want` and asserts
  FooterRowAt decodes it, which is FooterRowAt's definition (`row -
  s.footerTop`). Mutating footerTop to `bufRows+promptRows` and to
  `bufRows+gap+promptRows+3` left it green both times. The property is covered
  by the pre-existing TestFooterRowAtNamesTheEntryUnderAClick, which does go
  red — but the designated pin for "the arithmetic that must not be wrong"
  pins nothing. Assert an absolute row instead.
- **BR-5** [Important] `doc-attaches-to-the-wrong-decl` Paint's 35-line doc comment was reparented onto `const chromeGap`, leaving Paint undocumented
  The chromeGap comment was inserted at cmd/define/screen.go:447 with no blank
  line after Paint's doc block, so Go attaches the whole block ("A frame is
  budgeted in DISPLAY ROWS…") to the constant. Confirmed with `go doc -all
  -u`: Paint prints with no prose. Insert a blank line, or move
  chromeGap/grantedGap above Paint's doc block.
- **BR-6** [Important] `sweep-every-site-of-the-rule` the indicator guard's screenHostedFiles is an opt-in list, so a new screen-hosted file escapes the rule silently
  cmd/define/indicator_guard_test.go:18 enumerates the files the rule applies
  to. The list is complete today (main.go is the one-shot path, repl.go the
  piped loop, both correctly exempt), but the guard's own comment cites this
  package's purity guard as precedent and that one is an allowlist: in scope
  unless exempted. Invert to `nonScreenFiles = {"main.go", "repl.go"}` over
  package main's non-test files so a sixth site defaults into the class.
- **BR-7** [Minor] `cite-the-code-you-claim` the issue's guard Done-when overstates what the by-type predicate reaches
  workshop/issues/000044-play-chrome.md:179 says the guard "also reaches the
  two sites that call playAnnounced through playRegion — which a callee-name
  walk would have missed". After playRegion's parameter was deleted it takes
  no indicator, so the guard checks exactly the four playAnnounced arguments,
  all of which a callee-name walk would also have reached. The by-type rule is
  still the better one; the sentence describes the pre-deletion tree.
- **BR-8** [Minor] `cite-the-code-you-claim` README's "the bottom two rows are chrome" is placed in the board section, where it is not true
  cmd/define/README.md:148 sits under the board paragraphs, but on a board the
  dimmed rows are the prompt (drawn above the grid, as the fence at line 106
  shows) and the bar below it — not the bottom two. Accurate for an ordinary
  sitting; reword or relocate.
- **BR-9** [Minor] `one-owner-per-quantity` the chrome-band test hardcodes the dim escape rather than reading it from the palette
  cmd/define/play_loop_test.go:3648 spells "\x1b[2m" where newPalette(true).dim
  is the owner. Cosmetic — it fails loudly rather than silently — but it is a
  second speller of a sequence the palette exists to own (ARCH-DRY).

## Open findings

- **BR-1** [Minor] `frame-budget-completeness` fitsABoard is also the draw-time boardWhole predicate, so charging chromeGap there fires a false refusal at one height
- **BR-2** [Minor] `cite-the-code-you-claim` the AST-guard step points at dict_symbols_darwin_test.go, which parses no Go source
- **BR-3** [Important] `pin-must-fail-without-the-code` newPinnedScreen's `gap: chromeGap` is pinned by no test — deleting it leaves the full suite and the conformance suite green
- **BR-4** [Important] `pin-must-fail-without-the-code` TestAFooterClickIsUnmovedByTheChromeGap is tautological — it passes for any value of footerTop
- **BR-5** [Important] `doc-attaches-to-the-wrong-decl` Paint's 35-line doc comment was reparented onto `const chromeGap`, leaving Paint undocumented
- **BR-6** [Important] `sweep-every-site-of-the-rule` the indicator guard's screenHostedFiles is an opt-in list, so a new screen-hosted file escapes the rule silently
- **BR-7** [Minor] `cite-the-code-you-claim` the issue's guard Done-when overstates what the by-type predicate reaches
- **BR-8** [Minor] `cite-the-code-you-claim` README's "the bottom two rows are chrome" is placed in the board section, where it is not true
- **BR-9** [Minor] `one-owner-per-quantity` the chrome-band test hardcodes the dim escape rather than reading it from the palette
