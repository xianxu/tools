---
gate: plan-quality
issue: 30
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-29T16:31:08-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: M2.1 claims ORIGIN regions fall out of Render's colouring walk; render.go:200-218 says they do not
          detail: |-
            Render colours the section NAME (p.sect on sec.Name) and pushes sec.Text
            through prettyPronunciations, Split("|"), wrapText and opt.prose — four
            transforms that insert escapes and re-break lines, none of which knows
            where "French" is. OriginLanguage (origin.go:133) returns code and name
            but no offset, computed on raw s.Text before any of them. The plan must
            say how a span survives those transforms; the headword half of the claim
            is correct (render.go:110-130), the ORIGIN half is not.
          family: unbacked-existing-behavior
          round: 1
        - id: PQ-2
          severity: Important
          title: M2.4 wraps OriginLanguage, which is first-named-wins, so only one language is ever clickable
          detail: |-
            The issue's premise is that a click dissolves ambiguity because the user
            points at the language they meant (piano: French or Italian). OriginLanguage
            (origin.go:133) answers first-named and masks cognates/stages, so wrapping
            it leaves Italian inert; scanning originLanguages naively instead makes
            "German" in bring's cognate clause live, which is #29's D1 failure. Pick
            one and state it, including how cognateMarkers/historicalStages apply to
            the region set (ARCH-PURPOSE).
          family: arch-purpose-easy-subset
          round: 1
        - id: PQ-3
          severity: Important
          title: D5 replaces the crlf seam that open issue tools#32 owns, and never says whether stderr routes through the screen
          detail: |-
            workshop/issues/000032-crlf-seam.md is open and is exactly the question of
            where the crlf decision lives, naming reportVoice/playAnnounced's bare "\n"
            on stderr. #30 declares deps [tools#29, tools#35] only. D4's continuous raw
            mode unmasks those sites, and D5 lists the callers that keep writing without
            saying whether stderr is one of them. Either subsume #32 explicitly or
            sequence behind it.
          family: undeclared-cross-issue-dep
          round: 1
        - id: PQ-4
          severity: Important
          title: M1 ships a viewport with no input that moves it — the wheel is M2 and no key scrolls
          detail: |-
            screen.Scroll and Frame's offset land in M1, but decodeMouse is M2.2 and
            Apply binds Up/Down to the history walk while PageUp decodes as a 4-byte
            KeyUnknown (key_test.go:65). At the M1 boundary the alt screen has removed
            the terminal's scrollback and replaced it with nothing, so a long entry is
            unreachable. Name the M1 scroll input or state the boundary is knowingly
            incomplete.
          family: milestone-leaves-feature-unreachable
          round: 1
        - id: PQ-5
          severity: Important
          title: decodeMouse has no test strategy and no Done-when row, and it is the plan's one adversarial-input surface
          detail: |-
            It is a byte scanner over arbitrary device output. One strategy line:
            fuzz/property-test it seeded with malformed forms — truncated "ESC[<",
            overflowing coordinates, M vs m, unknown button bits, wheel codes where a
            click is expected, and a sequence split across two Reads (decodeKey's
            used==0 contract must extend to it).
          family: missing-test-strategy
          round: 1
        - id: PQ-6
          severity: Important
          title: Discoverability is deferred inside M2.5, pinned by no Done-when row, and the tracking mode it depends on is never chosen
          detail: |-
            The issue's Done-when requires a live token be visible before it is clicked;
            M2.5 says "underline on hover, or a footer line. Decide with a measurement"
            and M2's six Done-when rows pin none of it. 1006 is the encoding, not the
            mode: click-only (1000) makes hover impossible, motion (1002/1003) floods
            the loop's select and changes the wheel. Decide the mode in the plan, and
            note that an underline re-baselines TestRenderOutputUnchangedByRegions.
          family: undefined-acceptance-criteria
          round: 1
        - id: PQ-7
          severity: Minor
          title: M1.1 enumerates four table-test cases in prose; compress to one strategy line per risky function
          detail: |-
            "a partial write continues the last line; a write containing \n\n appends
            an empty line; Frame clamps the offset at both ends; a viewport taller than
            the buffer pads" will be code within the hour. The chunk-boundary property
            for Write (any split of the same byte stream yields the same lines) is
            worth more than the four cases and is what they are groping at.
          family: test-cases-enumerated-in-prose
          round: 1
        - id: PQ-8
          severity: Minor
          title: screen.lines and D3's exit transcript are unbounded
          detail: |-
            The buffer holds every line of a session and D3 replays all of it into the
            normal buffer on exit. A long session dumps thousands of lines at quit.
            State a cap, or state deliberately that there is none.
          family: unbounded-buffer
          round: 1
        - id: PQ-9
          severity: Minor
          title: M1.3's "only the destination changes" understates deleting cooked
          detail: |-
            cooked is a parameter of runEditor and submitLine; editorRig
            (editorloop_test.go:29) returns it and 47 test call sites pass it. The
            churn is mechanical, but it is a signature change across the suite, not a
            writer swap — say so, so the reviewer at the M1 boundary is not surprised
            by the diff size.
          family: blast-radius-understated
          round: 1
      blocked: true
---

# Gate ledger — tools#30 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-29T16:31:08-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `unbacked-existing-behavior` M2.1 claims ORIGIN regions fall out of Render's colouring walk; render.go:200-218 says they do not
  Render colours the section NAME (p.sect on sec.Name) and pushes sec.Text
  through prettyPronunciations, Split("|"), wrapText and opt.prose — four
  transforms that insert escapes and re-break lines, none of which knows
  where "French" is. OriginLanguage (origin.go:133) returns code and name
  but no offset, computed on raw s.Text before any of them. The plan must
  say how a span survives those transforms; the headword half of the claim
  is correct (render.go:110-130), the ORIGIN half is not.
- **PQ-2** [Important] `arch-purpose-easy-subset` M2.4 wraps OriginLanguage, which is first-named-wins, so only one language is ever clickable
  The issue's premise is that a click dissolves ambiguity because the user
  points at the language they meant (piano: French or Italian). OriginLanguage
  (origin.go:133) answers first-named and masks cognates/stages, so wrapping
  it leaves Italian inert; scanning originLanguages naively instead makes
  "German" in bring's cognate clause live, which is #29's D1 failure. Pick
  one and state it, including how cognateMarkers/historicalStages apply to
  the region set (ARCH-PURPOSE).
- **PQ-3** [Important] `undeclared-cross-issue-dep` D5 replaces the crlf seam that open issue tools#32 owns, and never says whether stderr routes through the screen
  workshop/issues/000032-crlf-seam.md is open and is exactly the question of
  where the crlf decision lives, naming reportVoice/playAnnounced's bare "\n"
  on stderr. #30 declares deps [tools#29, tools#35] only. D4's continuous raw
  mode unmasks those sites, and D5 lists the callers that keep writing without
  saying whether stderr is one of them. Either subsume #32 explicitly or
  sequence behind it.
- **PQ-4** [Important] `milestone-leaves-feature-unreachable` M1 ships a viewport with no input that moves it — the wheel is M2 and no key scrolls
  screen.Scroll and Frame's offset land in M1, but decodeMouse is M2.2 and
  Apply binds Up/Down to the history walk while PageUp decodes as a 4-byte
  KeyUnknown (key_test.go:65). At the M1 boundary the alt screen has removed
  the terminal's scrollback and replaced it with nothing, so a long entry is
  unreachable. Name the M1 scroll input or state the boundary is knowingly
  incomplete.
- **PQ-5** [Important] `missing-test-strategy` decodeMouse has no test strategy and no Done-when row, and it is the plan's one adversarial-input surface
  It is a byte scanner over arbitrary device output. One strategy line:
  fuzz/property-test it seeded with malformed forms — truncated "ESC[<",
  overflowing coordinates, M vs m, unknown button bits, wheel codes where a
  click is expected, and a sequence split across two Reads (decodeKey's
  used==0 contract must extend to it).
- **PQ-6** [Important] `undefined-acceptance-criteria` Discoverability is deferred inside M2.5, pinned by no Done-when row, and the tracking mode it depends on is never chosen
  The issue's Done-when requires a live token be visible before it is clicked;
  M2.5 says "underline on hover, or a footer line. Decide with a measurement"
  and M2's six Done-when rows pin none of it. 1006 is the encoding, not the
  mode: click-only (1000) makes hover impossible, motion (1002/1003) floods
  the loop's select and changes the wheel. Decide the mode in the plan, and
  note that an underline re-baselines TestRenderOutputUnchangedByRegions.
- **PQ-7** [Minor] `test-cases-enumerated-in-prose` M1.1 enumerates four table-test cases in prose; compress to one strategy line per risky function
  "a partial write continues the last line; a write containing \n\n appends
  an empty line; Frame clamps the offset at both ends; a viewport taller than
  the buffer pads" will be code within the hour. The chunk-boundary property
  for Write (any split of the same byte stream yields the same lines) is
  worth more than the four cases and is what they are groping at.
- **PQ-8** [Minor] `unbounded-buffer` screen.lines and D3's exit transcript are unbounded
  The buffer holds every line of a session and D3 replays all of it into the
  normal buffer on exit. A long session dumps thousands of lines at quit.
  State a cap, or state deliberately that there is none.
- **PQ-9** [Minor] `blast-radius-understated` M1.3's "only the destination changes" understates deleting cooked
  cooked is a parameter of runEditor and submitLine; editorRig
  (editorloop_test.go:29) returns it and 47 test call sites pass it. The
  churn is mechanical, but it is a signature change across the suite, not a
  writer swap — say so, so the reviewer at the M1 boundary is not surprised
  by the diff size.

## Open findings

- **PQ-1** [Important] `unbacked-existing-behavior` M2.1 claims ORIGIN regions fall out of Render's colouring walk; render.go:200-218 says they do not
- **PQ-2** [Important] `arch-purpose-easy-subset` M2.4 wraps OriginLanguage, which is first-named-wins, so only one language is ever clickable
- **PQ-3** [Important] `undeclared-cross-issue-dep` D5 replaces the crlf seam that open issue tools#32 owns, and never says whether stderr routes through the screen
- **PQ-4** [Important] `milestone-leaves-feature-unreachable` M1 ships a viewport with no input that moves it — the wheel is M2 and no key scrolls
- **PQ-5** [Important] `missing-test-strategy` decodeMouse has no test strategy and no Done-when row, and it is the plan's one adversarial-input surface
- **PQ-6** [Important] `undefined-acceptance-criteria` Discoverability is deferred inside M2.5, pinned by no Done-when row, and the tracking mode it depends on is never chosen
- **PQ-7** [Minor] `test-cases-enumerated-in-prose` M1.1 enumerates four table-test cases in prose; compress to one strategy line per risky function
- **PQ-8** [Minor] `unbounded-buffer` screen.lines and D3's exit transcript are unbounded
- **PQ-9** [Minor] `blast-radius-understated` M1.3's "only the destination changes" understates deleting cooked
