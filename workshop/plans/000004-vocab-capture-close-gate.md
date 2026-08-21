---
gate: boundary-review
issue: 4
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-21T11:08:20-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: '"three call sites" is two - defineOnce serves both the one-shot and line paths'
          detail: |-
            main.go:144 and repl.go:144 both call defineOnce, so widening adds one
            invocation site, not two. The design is still right; the count is not.
            (carried from plan-quality PQ-6, deferred to the boundary review)
          family: unbacked-existing-behavior-claim
          round: 1
        - id: BR-2
          severity: Important
          title: storeHistory is given two incompatible fates, and Task 1 Step 3's "tests unchanged" is unsatisfiable
          detail: |-
            This is the 2nd finding in family extraction-strands-behavior (prevalence 2: PQ-4's
            warn-once, now the durability contract itself). The rule: an extraction must enumerate
            every obligation the source component is contracted for — the tests that pin it, the
            warn-once, the durability guarantee — and say where each lands. Plan line 77 says
            storeHistory becomes a pure reader; line 109 says it delegates; the issue checkbox
            agrees with line 109. Under the reader reading, history_store_test.go:17-33, :37-55,
            :86-99, :155-166 and :133-150 all fail, contradicting "green without edits … do not
            edit them to fit".
            (carried from plan-quality PQ-10, deferred to the boundary review)
          family: extraction-strands-behavior
          round: 1
        - id: BR-3
          severity: Important
          title: Task 2 Step 3 still names defineOnce as the capture site, contradicting Chunk 1's lookupAndRender
          detail: |-
            This is the 3rd finding in family capture-arity-invariant (prevalence 3: PQ-7 double-capture,
            PQ-9 zero-capture-on-raw, now the executable step disagreeing with the prose). Do not patch
            line 181. The rule, which also covers PQ-6 and PQ-10: each design fact gets exactly ONE
            normative statement in the plan and every other mention references it rather than restating
            it (ARCH-DRY applied to the artifact). "Where capture happens" is currently stated at lines
            7, 24, 72, 116 and 181; three rounds have each fixed one copy and left the rest, which is
            why the family keeps recurring. Rewrite 7, 24, 116, 181 to point at the Chunk 1 statement,
            and give storeHistory's fate the same single-home treatment across lines 7, 24, 133 and the
            issue checkbox.
            (carried from plan-quality PQ-11, deferred to the boundary review)
          family: capture-arity-invariant
          round: 1
        - id: BR-4
          severity: Minor
          title: The plan calls d.capture but never says deps gains the field, nor what a deps literal without it does
          detail: |-
            deps is the injected IO seam (main.go:20) and both test rigs build it as a literal
            (main_test.go:14, main_test.go:41), so an unguarded d.capture.Capture panics the suite.
            The codebase already has the idiom for this at replraw.go:63, where a nil history seam
            falls back to memHistory. State whether capture takes a nil-fallback null object or every
            rig must supply one.
            (carried from plan-quality PQ-12, deferred to the boundary review)
          family: unstated-seam-default
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-08-21T11:08:20-07:00"
      agent: claude
      findings:
        - id: BR-5
          severity: Important
          title: The capture-arity test counts Capturer calls, not store writes, so the double-count it names passes
          detail: |-
            cmd/define/capture_test.go:86 injects countingCapturer at the seam, above both decideCapture
            and the writes. Verified by revert: restoring the AppendEvent/Upsert pair in storeHistory.Add
            leaves the entire suite green while the deck records Lookups:2 and two events per lookup. The
            plan's Task 2 Step 1 specified a counting STORE. atlas/define.md:285 claims the invariant is
            pinned; it is not. A store-level test with the real wiring goes red on the double-write variant
            and green at HEAD - verified both directions.
          family: unpinned-invariant
          round: 2
        - id: BR-6
          severity: Important
          title: The Forget traversal guard is asserted by no test, and half of it is unreachable
          detail: |-
            cmd/define/store/yaml.go:306. The issue's Done-when checks "cannot delete outside words/ -
            asserted, not inherited from Slug" and plan Task 3 Step 0 names the class. No test passes a
            traversal key to Forget. Verified by revert: replacing filepath.Base(Slug(k)) plus the
            unsafe-name check with a bare Slug(k) leaves go test ./cmd/define/store/... green. Separately
            name always ends in ".yaml", so the name == "." and name == ".." sub-conditions can never fire.
          family: unpinned-invariant
          round: 2
        - id: BR-7
          severity: Important
          title: openStore and the DEFINE_NO_CAPTURE env wiring have zero coverage, leaving half a Done-when unverified
          detail: |-
            cmd/define/main.go:73. Done-when says the opt-out also drops history to session-only; that
            clause lives entirely in openStore's noCapture return and nothing exercises it, nor the
            os.Getenv to opt.noCapture wiring at main.go:172, nor the Getwd-failure warning.
            TestNoCaptureSuppressesEverything only re-asserts decideCapture's branch through a
            storeCapturer built by hand.
          family: unpinned-invariant
          round: 2
        - id: BR-8
          severity: Important
          title: The raw success path bypasses decideCapture, giving "capture is off" three homes
          detail: |-
            cmd/define/main.go:247 returns before the capture call at :252, so Capture(word, true, opt)
            under -raw is unreachable and capture.go:30's raw branch fires only on the failure path - the
            truth-table row {"raw", true, ...} asserts a combination production never produces. openStore
            (main.go:74) reads opt.noCapture a third time to install noopCapturer. The plan explicitly
            forbids exactly this second home (ARCH-DRY).
          family: single-source-restated-by-hand
          round: 2
        - id: BR-9
          severity: Important
          title: atlas/define.md:300 still describes storeHistory.Add as the writer, contradicting :282 fifteen lines above
          detail: |-
            "History is events, the deck is successes" opens with "storeHistory.Add always appends an event,
            and upserts a word only when the lookup found something" and attributes the warn-once rule to it.
            Both are false since this diff, and the new "Capture: one site, one policy" section directly
            above says so. Same rule the gate raised as PQ-11, recurring in the atlas.
          family: prose-contradicts-code
          round: 2
        - id: BR-10
          severity: Important
          title: main.go:67 claims DEFINE_NO_CAPTURE is documented in --help; fs.Usage never mentions it
          detail: |-
            Confirmed against `go run ./cmd/define -h`. The plan requires the opt-out and its cost be stated
            in --help, the README and the atlas. It is an env var, so PrintDefaults will never surface it,
            and the usage text also never says define now writes to the working directory at all.
          family: prose-contradicts-code
          round: 2
        - id: BR-11
          severity: Important
          title: .gitignore does not ignore words/ or events/, which define now creates in the repo on every lookup
          detail: |-
            This diff makes define write to CWD on every invocation including failed ones, and this repo's
            developers run define from the checkout. Reproduced during review - a failed lookup created an
            untracked events/ in the repo root (removed; tree is clean). The existing .gitignore already
            carries a comment about a build artifact that first got committed this way.
          family: writes-to-cwd-unignored
          round: 2
        - id: BR-12
          severity: Minor
          title: define -forget="" falls through to the REPL instead of erroring
          detail: main.go:188 gates on *forget != "", so an explicitly-empty -forget starts an interactive session.
          family: flag-mode-dispatch
          round: 2
        - id: BR-13
          severity: Minor
          title: -forget under DEFINE_NO_CAPTURE reports "no deck in this directory"
          detail: Verified. There may well be a deck; the user opted out of writes. The message should say that.
          family: misleading-error-text
          round: 2
        - id: BR-14
          severity: Minor
          title: deps.forgetter() is a four-line nil-check wrapper around one field with one caller
          family: needless-indirection
          round: 2
        - id: BR-15
          severity: Minor
          title: newStore's three-return seam plus three nil-merges in run is lumpy; a small struct would collapse it
          family: needless-indirection
          round: 2
        - id: BR-16
          severity: Minor
          title: The issue's Log has no implementation entry and the ticked "Manual check" step records no evidence
          family: undocumented-work-log
          round: 2
        - id: BR-17
          severity: Minor
          title: atlas "Entry modes" table omits define -forget, the fourth invocation this diff adds
          family: prose-contradicts-code
          round: 2
      blocked: true
---

# Gate ledger — tools#4 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-21T11:08:20-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `unbacked-existing-behavior-claim` "three call sites" is two - defineOnce serves both the one-shot and line paths
  main.go:144 and repl.go:144 both call defineOnce, so widening adds one
  invocation site, not two. The design is still right; the count is not.
  (carried from plan-quality PQ-6, deferred to the boundary review)
- **BR-2** [Important] `extraction-strands-behavior` storeHistory is given two incompatible fates, and Task 1 Step 3's "tests unchanged" is unsatisfiable
  This is the 2nd finding in family extraction-strands-behavior (prevalence 2: PQ-4's
  warn-once, now the durability contract itself). The rule: an extraction must enumerate
  every obligation the source component is contracted for — the tests that pin it, the
  warn-once, the durability guarantee — and say where each lands. Plan line 77 says
  storeHistory becomes a pure reader; line 109 says it delegates; the issue checkbox
  agrees with line 109. Under the reader reading, history_store_test.go:17-33, :37-55,
  :86-99, :155-166 and :133-150 all fail, contradicting "green without edits … do not
  edit them to fit".
  (carried from plan-quality PQ-10, deferred to the boundary review)
- **BR-3** [Important] `capture-arity-invariant` Task 2 Step 3 still names defineOnce as the capture site, contradicting Chunk 1's lookupAndRender
  This is the 3rd finding in family capture-arity-invariant (prevalence 3: PQ-7 double-capture,
  PQ-9 zero-capture-on-raw, now the executable step disagreeing with the prose). Do not patch
  line 181. The rule, which also covers PQ-6 and PQ-10: each design fact gets exactly ONE
  normative statement in the plan and every other mention references it rather than restating
  it (ARCH-DRY applied to the artifact). "Where capture happens" is currently stated at lines
  7, 24, 72, 116 and 181; three rounds have each fixed one copy and left the rest, which is
  why the family keeps recurring. Rewrite 7, 24, 116, 181 to point at the Chunk 1 statement,
  and give storeHistory's fate the same single-home treatment across lines 7, 24, 133 and the
  issue checkbox.
  (carried from plan-quality PQ-11, deferred to the boundary review)
- **BR-4** [Minor] `unstated-seam-default` The plan calls d.capture but never says deps gains the field, nor what a deps literal without it does
  deps is the injected IO seam (main.go:20) and both test rigs build it as a literal
  (main_test.go:14, main_test.go:41), so an unguarded d.capture.Capture panics the suite.
  The codebase already has the idiom for this at replraw.go:63, where a nil history seam
  falls back to memHistory. State whether capture takes a nil-fallback null object or every
  rig must supply one.
  (carried from plan-quality PQ-12, deferred to the boundary review)

## Round 2 — 2026-08-21T11:08:20-07:00 (claude) — BLOCKED

### Raised

- **BR-5** [Important] `unpinned-invariant` The capture-arity test counts Capturer calls, not store writes, so the double-count it names passes
  cmd/define/capture_test.go:86 injects countingCapturer at the seam, above both decideCapture
  and the writes. Verified by revert: restoring the AppendEvent/Upsert pair in storeHistory.Add
  leaves the entire suite green while the deck records Lookups:2 and two events per lookup. The
  plan's Task 2 Step 1 specified a counting STORE. atlas/define.md:285 claims the invariant is
  pinned; it is not. A store-level test with the real wiring goes red on the double-write variant
  and green at HEAD - verified both directions.
- **BR-6** [Important] `unpinned-invariant` The Forget traversal guard is asserted by no test, and half of it is unreachable
  cmd/define/store/yaml.go:306. The issue's Done-when checks "cannot delete outside words/ -
  asserted, not inherited from Slug" and plan Task 3 Step 0 names the class. No test passes a
  traversal key to Forget. Verified by revert: replacing filepath.Base(Slug(k)) plus the
  unsafe-name check with a bare Slug(k) leaves go test ./cmd/define/store/... green. Separately
  name always ends in ".yaml", so the name == "." and name == ".." sub-conditions can never fire.
- **BR-7** [Important] `unpinned-invariant` openStore and the DEFINE_NO_CAPTURE env wiring have zero coverage, leaving half a Done-when unverified
  cmd/define/main.go:73. Done-when says the opt-out also drops history to session-only; that
  clause lives entirely in openStore's noCapture return and nothing exercises it, nor the
  os.Getenv to opt.noCapture wiring at main.go:172, nor the Getwd-failure warning.
  TestNoCaptureSuppressesEverything only re-asserts decideCapture's branch through a
  storeCapturer built by hand.
- **BR-8** [Important] `single-source-restated-by-hand` The raw success path bypasses decideCapture, giving "capture is off" three homes
  cmd/define/main.go:247 returns before the capture call at :252, so Capture(word, true, opt)
  under -raw is unreachable and capture.go:30's raw branch fires only on the failure path - the
  truth-table row {"raw", true, ...} asserts a combination production never produces. openStore
  (main.go:74) reads opt.noCapture a third time to install noopCapturer. The plan explicitly
  forbids exactly this second home (ARCH-DRY).
- **BR-9** [Important] `prose-contradicts-code` atlas/define.md:300 still describes storeHistory.Add as the writer, contradicting :282 fifteen lines above
  "History is events, the deck is successes" opens with "storeHistory.Add always appends an event,
  and upserts a word only when the lookup found something" and attributes the warn-once rule to it.
  Both are false since this diff, and the new "Capture: one site, one policy" section directly
  above says so. Same rule the gate raised as PQ-11, recurring in the atlas.
- **BR-10** [Important] `prose-contradicts-code` main.go:67 claims DEFINE_NO_CAPTURE is documented in --help; fs.Usage never mentions it
  Confirmed against `go run ./cmd/define -h`. The plan requires the opt-out and its cost be stated
  in --help, the README and the atlas. It is an env var, so PrintDefaults will never surface it,
  and the usage text also never says define now writes to the working directory at all.
- **BR-11** [Important] `writes-to-cwd-unignored` .gitignore does not ignore words/ or events/, which define now creates in the repo on every lookup
  This diff makes define write to CWD on every invocation including failed ones, and this repo's
  developers run define from the checkout. Reproduced during review - a failed lookup created an
  untracked events/ in the repo root (removed; tree is clean). The existing .gitignore already
  carries a comment about a build artifact that first got committed this way.
- **BR-12** [Minor] `flag-mode-dispatch` define -forget="" falls through to the REPL instead of erroring
  main.go:188 gates on *forget != "", so an explicitly-empty -forget starts an interactive session.
- **BR-13** [Minor] `misleading-error-text` -forget under DEFINE_NO_CAPTURE reports "no deck in this directory"
  Verified. There may well be a deck; the user opted out of writes. The message should say that.
- **BR-14** [Minor] `needless-indirection` deps.forgetter() is a four-line nil-check wrapper around one field with one caller
- **BR-15** [Minor] `needless-indirection` newStore's three-return seam plus three nil-merges in run is lumpy; a small struct would collapse it
- **BR-16** [Minor] `undocumented-work-log` The issue's Log has no implementation entry and the ticked "Manual check" step records no evidence
- **BR-17** [Minor] `prose-contradicts-code` atlas "Entry modes" table omits define -forget, the fourth invocation this diff adds

## Open findings

- **BR-1** [Minor] `unbacked-existing-behavior-claim` "three call sites" is two - defineOnce serves both the one-shot and line paths
- **BR-2** [Important] `extraction-strands-behavior` storeHistory is given two incompatible fates, and Task 1 Step 3's "tests unchanged" is unsatisfiable
- **BR-3** [Important] `capture-arity-invariant` Task 2 Step 3 still names defineOnce as the capture site, contradicting Chunk 1's lookupAndRender
- **BR-4** [Minor] `unstated-seam-default` The plan calls d.capture but never says deps gains the field, nor what a deps literal without it does
- **BR-5** [Important] `unpinned-invariant` The capture-arity test counts Capturer calls, not store writes, so the double-count it names passes
- **BR-6** [Important] `unpinned-invariant` The Forget traversal guard is asserted by no test, and half of it is unreachable
- **BR-7** [Important] `unpinned-invariant` openStore and the DEFINE_NO_CAPTURE env wiring have zero coverage, leaving half a Done-when unverified
- **BR-8** [Important] `single-source-restated-by-hand` The raw success path bypasses decideCapture, giving "capture is off" three homes
- **BR-9** [Important] `prose-contradicts-code` atlas/define.md:300 still describes storeHistory.Add as the writer, contradicting :282 fifteen lines above
- **BR-10** [Important] `prose-contradicts-code` main.go:67 claims DEFINE_NO_CAPTURE is documented in --help; fs.Usage never mentions it
- **BR-11** [Important] `writes-to-cwd-unignored` .gitignore does not ignore words/ or events/, which define now creates in the repo on every lookup
- **BR-12** [Minor] `flag-mode-dispatch` define -forget="" falls through to the REPL instead of erroring
- **BR-13** [Minor] `misleading-error-text` -forget under DEFINE_NO_CAPTURE reports "no deck in this directory"
- **BR-14** [Minor] `needless-indirection` deps.forgetter() is a four-line nil-check wrapper around one field with one caller
- **BR-15** [Minor] `needless-indirection` newStore's three-return seam plus three nil-merges in run is lumpy; a small struct would collapse it
- **BR-16** [Minor] `undocumented-work-log` The issue's Log has no implementation entry and the ticked "Manual check" step records no evidence
- **BR-17** [Minor] `prose-contradicts-code` atlas "Entry modes" table omits define -forget, the fourth invocation this diff adds
