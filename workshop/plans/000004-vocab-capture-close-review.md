# Boundary Review — tools#4 (whole-issue close)

| field | value |
|-------|-------|
| issue | 4 — capture looked-up words into the deck |
| repo | tools |
| issue file | workshop/issues/000004-vocab-capture.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4afa1c537819ad662ad149a14bd757ac149772e5..3ca9ab4752f6106b26153dbaf62bd6d4300566f0 |
| command | sdlc close --issue 4 |
| reviewer | claude |
| timestamp | 2026-08-21T11:08:20-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The extraction is structurally right and the shipped behavior is correct as far as I could verify: capture happens at `lookupAndRender`, exactly one writer exists, `Forget` joined the conformance suite so both `Mem` and `YAML` answer for it, and the whole suite is green. What blocks a clean SHIP is that two of the invariants this issue exists to protect are **not pinned by any test** — I verified both by revert. Restoring the store writes to `storeHistory.Add` (the double-count failure mode the plan names three times) leaves the *entire* suite green while the deck records `Lookups:2` and two events per lookup; deleting the `Forget` traversal guard leaves the store suite green too, though the issue checks off "asserted, not inherited from `Slug`". Neither is a live bug, so this is non-blocking, but the pins are cheap and I've written and validated one of them.

### 1. Strengths

- **The capture site is right, and for the reason stated.** `lookupAndRender` (`cmd/define/main.go:244`, `:252`) really is the one function all three entry paths share — `defineOnce` would have missed `submitLine`. The call graph in the plan and atlas matches the code; I checked every `.Capture(` site.
- **`Forget` joined `storetest.Suite`** (`cmd/define/store/storetest/suite.go:113`) rather than getting a per-implementation test, so `Mem` and `YAML` both answer for case-insensitivity, absence-is-not-an-error, and events-survive. This is ARCH-MOCK done properly.
- **`decideCapture` is genuinely pure** (`cmd/define/capture.go:26`) — table-tested with no IO, no fixtures, no clock. Branch order (`noCapture` → `raw` → `!found`) is correct.
- **Capture-after-render ordering** is honored (`main.go:252` follows the `Render` write), matching the plan's "never before — a lookup that fails to render should not be claimed as studied."
- **Dropping `storeHistory.warned`** is correct rather than a regression: `warnf` now has exactly one call site (construction), so warn-once is structural. The comment at `history_store.go:68` explains why.

### 2. Critical findings

None.

### 3. Important findings

**I-1 — `cmd/define/capture_test.go:86` — the capture-arity test cannot see the bug it names.**
`countingCapturer` is injected *at* the seam (`rig.deps.capture = c`), so it counts `Capturer.Capture` calls, not store writes. The plan's Task 2 Step 1 specified "driven through a **counting store**"; the substitution removed the test's power, because `decideCapture` and the writes both live *below* the seam that the fake replaces. Verified: I restored the `AppendEvent`/`Upsert` pair in `storeHistory.Add` and the full suite stayed green (`ok github.com/xianxu/tools/cmd/define`) while the deck double-counted. `atlas/define.md:285` claims this is "Pinned by a capture-arity test across all three paths" — it is not.
*Fix:* add a test asserting against the store with the real wiring. This one goes red on the double-write variant (`Lookups:2`, `2 events`) and green at HEAD — verified both directions:
```go
dir := t.TempDir()
st := store.NewYAML(dir, nil)
rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
rig.deps.history = newStoreHistory(st, fixedClock(1), nil)
rig.deps.capture = newStoreCapturer(st, fixedClock(1), nil)
runEditor(t.Context(), scriptKeys("sycophantic\r"), rig.deps, opt, cooked, finish, &out, &errb)
deck, _ := st.Deck()          // want len 1, Lookups == 1
ev, _ := st.Events(time.Time{}) // want len 1
```
Keep `countingCapturer` for the per-path arity subtests; it answers a different question well.

**I-2 — `cmd/define/store/yaml.go:306` — the `Forget` traversal guard is unpinned, and half of it is unreachable.**
The issue's Done-when checks "`--forget` cannot delete outside `words/` — asserted, not inherited from `Slug`", and plan Task 3 Step 0 names the traversal class explicitly. No test calls `Forget` with a traversal key. Verified: I replaced `filepath.Base(Slug(k))` + the unsafe-name check with a bare `Slug(k)` and `go test ./cmd/define/store/...` stayed green. Separately, `name` always ends in `".yaml"`, so the `name == "."` and `name == ".."` sub-conditions can never fire — an assertion no fixture enters.
*Fix:* add to the `storetest` suite: `Forget("../../../etc/passwd")` returns `(false, nil)`, and a sentinel file one level above `words/` still exists afterward. Drop the two dead sub-conditions or move the check onto `Slug(k)` before the suffix is appended.

**I-3 — `cmd/define/main.go:247` — the raw success path bypasses `decideCapture`, so "capture is off" has three homes.**
`if opt.raw { …; return 0, false }` returns before `main.go:252`, which makes `Capture(word, true, opt)` under `-raw` unreachable — `capture.go:30`'s `case opt.raw` is consulted only on the *failure* path, and the truth-table row `{"raw", true, options{raw: true}, captureNothing}` asserts a combination production never produces. `openStore` (`main.go:74`) reads `opt.noCapture` a third time to install `noopCapturer`. The plan spends a paragraph forbidding exactly this ("There is deliberately **no** null-object `noCapture` … `decideCapture` is the only place that answers it"), and ARCH-DRY says one source of truth per behavior. Behavior is currently correct; the risk is that the copies diverge the first time either grows a case.
*Fix:* move the capture call inside the raw branch so the policy is the sole arbiter:
```go
if opt.raw {
    fmt.Fprintln(stdout, text)
    d.capture.Capture(word, true, opt) // decideCapture says "nothing"; it should say it
    return 0, false
}
```
`openStore`'s branch is defensible as "there is nowhere to write" — but say so at `main.go:74` rather than leaving it reading like a second policy.

**I-4 — `atlas/define.md:300` contradicts `atlas/define.md:282` fifteen lines later.**
"History is events, the deck is successes" still opens with "`storeHistory.Add` always appends an **event**, and upserts a **word** only when the lookup found something" and goes on to attribute the warn-once rule to it. Both are now false; the new section directly above says so. This is the same rule the plan gate raised as PQ-11 (one normative statement per design fact), recurring in the atlas.
*Fix:* rewrite `:300-309` to state the split — `storeCapturer` appends the event and upserts the word; `storeHistory` reads the log once at construction and recalls from memory; `Prefix` therefore reads events, not the deck.

**I-5 — `cmd/define/main.go:67` claims `DEFINE_NO_CAPTURE` is documented in `--help`; it is not.**
The comment reads "Stated here, in --help, and in the README", and the plan requires "it is stated in `--help`, the README and the atlas beside the flag, not in a footnote." `fs.Usage` (`main.go:132`) never mentions `DEFINE_NO_CAPTURE`, nor that `define` now writes to the working directory at all. Since it is an env var, `PrintDefaults` will never surface it. Confirmed against `go run ./cmd/define -h`.
*Fix:* add a short paragraph to the usage text: define records lookups under `words/` and `events/` in the current directory; `DEFINE_NO_CAPTURE=1` disables that, and with it history is session-only.

**I-6 — `cmd/define/main.go:73` — `openStore` has zero test coverage, so half a Done-when is unverified.**
Done-when says "`DEFINE_NO_CAPTURE=1` writes nothing at all, **and history falls back to session-only rather than half-persisting**." The first clause is covered (`TestNoCaptureSuppressesEverything`, which only re-asserts `decideCapture`'s branch). The second lives entirely in `openStore`'s `if opt.noCapture` return, and nothing exercises it — nor the `os.Getenv` → `opt.noCapture` wiring at `main.go:172`, nor the `Getwd`-failure warning. `openStore` is small and takes `(options, io.Writer)`, so it is directly callable.
*Fix:* a table test on `openStore` asserting the returned `History` is `*memHistory`, the `Capturer` is `noopCapturer`, and the deck is nil under `noCapture`; plus one `t.Setenv("DEFINE_NO_CAPTURE", "1")` test through `run` that asserts nothing lands on disk.

**I-7 — `.gitignore` does not ignore `words/` or `events/`.**
This diff makes `define` write to CWD on *every* invocation including failed lookups, and this repo's developers run `define` from the checkout. I hit it during review — a failed lookup created an untracked `events/` in the repo root (removed; `git status` is clean). The existing `.gitignore` already carries a comment about a build artifact that "first got committed" this way.
*Fix:* add `/words/` and `/events/`.

### 4. Minor findings

- `cmd/define/main.go:188` — `define -forget=""` falls through to the `NArg` switch and starts the REPL; an explicitly-empty `-forget` is a usage error, not a lookup session.
- `forgetWord` under `DEFINE_NO_CAPTURE=1` prints "define: no deck in this directory" (verified). There may well be a deck — the user opted out of writes. Say that instead.
- `cmd/define/main.go:46` — `deps.forgetter()` is a four-line nil-check wrapper around one field with one caller; inline it.
- `deps.newStore func(options, io.Writer) (History, Capturer, store.Store)` plus the three `if d.X == nil` assignments at `main.go:176-186` is a lumpy seam. A small `storeDeps` struct would let `run` do one nil-merge.
- Issue `## Log` has no implementation entry at all. Task 3 Step 5 "Manual check" is ticked with no recorded evidence, and AGENTS.md §2 asks for discoveries there. (Note: I could not run the success path myself — the macOS dictionary returns "no dictionary entry" in this sandbox — so the success-side manual check is unverified by me either way.)
- `atlas/define.md` "Entry modes" table lists three invocations and omits `define -forget <word>`, which the new section introduces as a fourth mode.

### 5. Test coverage notes

The suite is green (`go test ./cmd/define/...`, 23.8s) and the new tests are well-written prose-wise, but the coverage sits on the wrong side of the seam in three places, all the same shape: **`decideCapture` and the store writes both live below `Capturer`, so every fake that replaces `Capturer` bypasses the policy *and* the writes it is supposed to pin.** That is why I-1 and I-6 both slipped. The three tests that "moved with the writes" (`TestCapturedWordsPersistAcrossSessions`, `TestCapturerRecallsTyposButDoesNotDeckThem`, `TestStoreCapturerDegradesOnWriteFailure`) were genuinely re-targeted rather than deleted — that part of the plan's discipline held, and `TestEditorPersistsThroughDeps` correctly wires both halves over one store with a comment saying why. What's missing is a single end-to-end assertion **against the store** through the real wiring; I-1's sketch is that test and it also happens to cover the `Lookups` half of Done-when #2, which nothing currently exercises through an entry path (the `Lookups = 2` assertion at `storetest/suite.go:71` is about `Upsert` merge semantics, not capture arity).

### 6. Architectural notes

- **ARCH-DRY — flag (I-3, I-4).** One fact ("does this lookup get recorded") restated at three code sites; one fact ("who writes to the store") restated in two contradicting atlas sections. Consolidation points named above.
- **ARCH-PURE — pass.** `decideCapture` is a real pure function with a real IO-free test; `storeCapturer` is a thin shell over it; `lookupAndRender` stays the injected boundary. No business logic leaked into the store package.
- **ARCH-PURPOSE — flag (I-1, I-2).** Shadow-sweep on the single source: `decideCapture` is enforced for `storeCapturer` ✓, hand-restated at `lookupAndRender`'s raw return ✗, hand-restated at `openStore` ✗. And the issue's stated purpose includes two "asserted, not assumed" obligations (arity, traversal) that shipped as documentation rather than as assertions — the pin *is* the deliverable there, not a follow-up.
- **ARCH-MOCK — mostly pass, one flag.** The store seam is exemplary: `Mem` ships as production code, `storetest.Suite` runs both implementations, `Forget` joined it in the same commit. The flag is `countingCapturer` — a *stateless* double standing in for a stateful interaction, placed above the policy it is supposed to verify. Production flow and test flow do not share the same boundary there, which is precisely the ARCH-MOCK failure mode.
- **For #5 (ordering by `Lookups`):** it will read the number this diff writes. Land I-1 before #5 starts, or the first symptom of a second writer will be a wrong review schedule, not a red test.

### 7. Plan revision recommendations

The plan has no `## Revisions` section and four gate findings are still listed open. Add one entry covering:

- **Task 2 Step 3** still reads "`defineOnce` calls `d.capture.Capture(word, found)`" — the code captures in `lookupAndRender` and the signature is `Capture(word string, found bool, opt options)`. This is PQ-11, raised at round 4 and never resolved; the checkbox was ticked over it. Rewrite the step to point at Chunk 1's statement rather than restating it.
- **Task 2 Step 1** specifies "a counting store"; a counting *capturer* shipped. Either the test changes (preferred, see I-1) or the plan records the substitution and why.
- **Chunk 1, `storeCapturer` bullet** — "it moves here and `storeHistory` **delegates**" contradicts Chunk 1's own "`storeHistory` therefore stops writing". `storeHistory` does not delegate; it does nothing. This is PQ-10, still open.
- **Chunk 1, "Why pure and separate"** — "consulted from three call sites" is wrong; `decideCapture` has exactly one caller (`storeCapturer.Capture`). This is PQ-6, still open.
- **`Capturer` signature** in Chunk 1 is `Capture(word string, found bool)`; shipped with a third `opt options` parameter.
- **"`openHistory` installs the session-only `memHistory`"** — the function is now `openStore` and returns a triple.
- **Task 3 Step 0's traversal and directory-comparison assertions** were not delivered (I-2); the suite counts events rather than comparing the directory before and after.

```findings
findings:
  - id: new
    severity: Important
    family: unpinned-invariant
    title: |
      The capture-arity test counts Capturer calls, not store writes, so the double-count it names passes
    detail: |
      cmd/define/capture_test.go:86 injects countingCapturer at the seam, above both decideCapture
      and the writes. Verified by revert: restoring the AppendEvent/Upsert pair in storeHistory.Add
      leaves the entire suite green while the deck records Lookups:2 and two events per lookup. The
      plan's Task 2 Step 1 specified a counting STORE. atlas/define.md:285 claims the invariant is
      pinned; it is not. A store-level test with the real wiring goes red on the double-write variant
      and green at HEAD - verified both directions.
  - id: new
    severity: Important
    family: unpinned-invariant
    title: |
      The Forget traversal guard is asserted by no test, and half of it is unreachable
    detail: |
      cmd/define/store/yaml.go:306. The issue's Done-when checks "cannot delete outside words/ -
      asserted, not inherited from Slug" and plan Task 3 Step 0 names the class. No test passes a
      traversal key to Forget. Verified by revert: replacing filepath.Base(Slug(k)) plus the
      unsafe-name check with a bare Slug(k) leaves go test ./cmd/define/store/... green. Separately
      name always ends in ".yaml", so the name == "." and name == ".." sub-conditions can never fire.
  - id: new
    severity: Important
    family: unpinned-invariant
    title: |
      openStore and the DEFINE_NO_CAPTURE env wiring have zero coverage, leaving half a Done-when unverified
    detail: |
      cmd/define/main.go:73. Done-when says the opt-out also drops history to session-only; that
      clause lives entirely in openStore's noCapture return and nothing exercises it, nor the
      os.Getenv to opt.noCapture wiring at main.go:172, nor the Getwd-failure warning.
      TestNoCaptureSuppressesEverything only re-asserts decideCapture's branch through a
      storeCapturer built by hand.
  - id: new
    severity: Important
    family: single-source-restated-by-hand
    title: |
      The raw success path bypasses decideCapture, giving "capture is off" three homes
    detail: |
      cmd/define/main.go:247 returns before the capture call at :252, so Capture(word, true, opt)
      under -raw is unreachable and capture.go:30's raw branch fires only on the failure path - the
      truth-table row {"raw", true, ...} asserts a combination production never produces. openStore
      (main.go:74) reads opt.noCapture a third time to install noopCapturer. The plan explicitly
      forbids exactly this second home (ARCH-DRY).
  - id: new
    severity: Important
    family: prose-contradicts-code
    title: |
      atlas/define.md:300 still describes storeHistory.Add as the writer, contradicting :282 fifteen lines above
    detail: |
      "History is events, the deck is successes" opens with "storeHistory.Add always appends an event,
      and upserts a word only when the lookup found something" and attributes the warn-once rule to it.
      Both are false since this diff, and the new "Capture: one site, one policy" section directly
      above says so. Same rule the gate raised as PQ-11, recurring in the atlas.
  - id: new
    severity: Important
    family: prose-contradicts-code
    title: |
      main.go:67 claims DEFINE_NO_CAPTURE is documented in --help; fs.Usage never mentions it
    detail: |
      Confirmed against `go run ./cmd/define -h`. The plan requires the opt-out and its cost be stated
      in --help, the README and the atlas. It is an env var, so PrintDefaults will never surface it,
      and the usage text also never says define now writes to the working directory at all.
  - id: new
    severity: Important
    family: writes-to-cwd-unignored
    title: |
      .gitignore does not ignore words/ or events/, which define now creates in the repo on every lookup
    detail: |
      This diff makes define write to CWD on every invocation including failed ones, and this repo's
      developers run define from the checkout. Reproduced during review - a failed lookup created an
      untracked events/ in the repo root (removed; tree is clean). The existing .gitignore already
      carries a comment about a build artifact that first got committed this way.
  - id: new
    severity: Minor
    family: flag-mode-dispatch
    title: |
      define -forget="" falls through to the REPL instead of erroring
    detail: |
      main.go:188 gates on *forget != "", so an explicitly-empty -forget starts an interactive session.
  - id: new
    severity: Minor
    family: misleading-error-text
    title: |
      -forget under DEFINE_NO_CAPTURE reports "no deck in this directory"
    detail: |
      Verified. There may well be a deck; the user opted out of writes. The message should say that.
  - id: new
    severity: Minor
    family: needless-indirection
    title: |
      deps.forgetter() is a four-line nil-check wrapper around one field with one caller
  - id: new
    severity: Minor
    family: needless-indirection
    title: |
      newStore's three-return seam plus three nil-merges in run is lumpy; a small struct would collapse it
  - id: new
    severity: Minor
    family: undocumented-work-log
    title: |
      The issue's Log has no implementation entry and the ticked "Manual check" step records no evidence
  - id: new
    severity: Minor
    family: prose-contradicts-code
    title: |
      atlas "Entry modes" table omits define -forget, the fourth invocation this diff adds
```
