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

---

## Re-review — 2026-08-21T11:22:26-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 4 — capture looked-up words into the deck |
| repo | tools |
| issue file | workshop/issues/000004-vocab-capture.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4afa1c537819ad662ad149a14bd757ac149772e5..6eb36f8f5e94711874aa26a279251f5bafd14081 |
| command | sdlc close --issue 4 |
| reviewer | claude |
| timestamp | 2026-08-21T11:22:26-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: REWORK
confidence: high
```

The shipped runtime behavior is correct and I found no correctness bugs: capture happens at `lookupAndRender` (the one function all three entry paths share), there is exactly one writer, `Forget` joined the shared conformance suite, and I drove every `-forget` surface end-to-end against a real deck on disk. Six of the seven Important fixes this round claims are real and I verified them by mutation. What blocks the boundary is the seventh: commit `6eb36f8` states *"The check moved onto the slug, and the conformance suite now asserts that traversal keys delete nothing"* — `cmd/define/store/yaml.go` was never touched by that commit, and I confirmed by revert that stripping `filepath.Base(Slug(k))` plus the unsafe-name check leaves `go test ./cmd/define/store/...` fully green, new traversal subtest included. The issue's Done-when box for exactly that clause — "`--forget` cannot delete outside `words/` — asserted, not inherited from `Slug`" — is ticked. A false tick plus a commit message describing a change that is not in the tree is the record this close would make permanent, and it is the third round on this issue where a claimed fix was not actually landed. Separately, all four plan-level findings (BR-1..BR-4) are untouched and the plan still has no `## Revisions` section, and the issue `## Log` still contains only its creation line.

### 1. Strengths

- **BR-5 was fixed properly, and the fix is mutation-proven.** `TestNoDoubleWriteThroughTheRealWiring` (`cmd/define/capture_test.go:232`) asserts against the store through the real wiring. I restored the `AppendEvent`/`Upsert` pair in `storeHistory.Add` and it was the *only* test that went red (`Lookups = 2`, `2 events`) — confirming both that the new test works and that the per-path `countingCapturer` subtests are blind, exactly as the finding said.
- **BR-7 verified in both directions.** Severing `os.Getenv("DEFINE_NO_CAPTURE")` reddens `TestNoCaptureWritesNothingToDisk`; making `openStore` ignore `opt.noCapture` reddens `TestOpenStoreUnderOptOut` on all three returns. Neither test is reasserting the implementation — they pin distinct halves of the Done-when.
- **`workshop/lessons.md:144` is the right lesson, drawn correctly.** "Where you inject the double decides what the test can see… when a plan names a specific seam for a test, substituting a different one is a design change, not an implementation detail." That is the generalizable rule, not a patch note.
- **`-forget` is correct end-to-end.** I ran the real binary against a seeded deck: present word → `removed sycophantic` / 0; absent → `not in the deck` / 1; `-forget=""` → `-forget needs a word` / 2; `-forget a b` → usage error / 2; under `DEFINE_NO_CAPTURE=1` → `DEFINE_NO_CAPTURE is set, so no deck was opened` / 1. BR-12 and BR-13 confirmed by observation, and the `openStore` → `d.deck` wiring works.
- **`Forget` joined `storetest.Suite` rather than getting per-implementation tests** (`storetest/suite.go:113`), so `Mem` and `YAML` both answer for it. `store.Store` gaining a method is a breaking interface change; I swept the tree and both implementations plus both test doubles were updated — no consumers outside `cmd/define`.
- **Store directories are still created lazily** (`yaml.go:90`, `:185`), so opening the REPL without a lookup writes nothing.

### 2. Critical findings

None.

### 3. Important findings

**N-1 — `cmd/define/history_store.go:17` — the sentence round 2 fixed in the atlas survives verbatim in the code, contradicting line 25 of the same comment block.**
**This is the 4th finding in family `prose-contradicts-code`** (BR-9 atlas, BR-10 `--help`, BR-17 entry-modes table). Per the escalation rule I am not asking for this instance to be patched. The type comment reads:

```
//   - Add always appends an EVENT, but upserts a Word only when the lookup
//     found something. …
…
// … Since #4 this type only
// READS the store — at construction — and storeCapturer owns every write.
```

Eight lines apart, in one comment. Round 2's I-4 fixed the atlas copy of this exact claim and left the code copy — the same one-copy-per-round pattern the plan gate named as PQ-11 and BR-3.

*The rule, which covers BR-9, BR-10, BR-17 and this:* **a behavioral fact gets exactly one normative home, and every other mention points at it instead of restating it.** Concretely for this codebase: the atlas section is the home; a doc comment that restates *what* a component does (rather than *why* it is shaped that way) is a copy and should be deleted or reduced to a pointer. Applying the rule here means deleting the "Two things are deliberately NOT the same here" bullet list from `history_store.go:15-22` rather than rewording it — the surviving prose ("this type only READS", plus the `Add` comment at `:60`) already says everything true, and the atlas owns the split. The measured prevalence is 4; a fifth instance means the rule was written down and not applied, not that a fifth file needs editing.

**N-2 — `unpinned-invariant`: two fixes landed this round are unreachable by any test, and BR-6's is unreachable in principle as written.**
**This is the 4th finding in family `unpinned-invariant`** (BR-5, BR-6, BR-7). Not a request to patch these instances. Residual, all verified:
- Removing `d.capture.Capture(word, true, opt)` from the `-raw` branch (`main.go:257`) — the entire BR-8 fix — leaves `go test ./cmd/define/...` green. `TestCaptureArityIsOnePerLookup` has subtests for one-shot, piped, raw editor, replay and failure, but none for `-raw`, which is the one row the truth table (`capture_test.go:24`) asserts and production can now actually produce.
- `openStore`'s non-`noCapture` deck return has no test; `forgetRig` sets `d.deck` by hand. I verified that wiring only by running the binary.
- BR-6 remains fully unpinned (see disposition).

*The rule:* **a fix ships with a test whose failure you have observed by removing the fix.** Round 2 stated this ("A fix is complete only when a test FAILS WITHOUT IT") and `lessons.md:144` now records its seam-placement corollary; BR-5 and BR-7 show the practice works when applied, and these three show it was applied selectively. The operational form worth adding to `lessons.md`: *before ticking a Done-when or closing a finding, delete the line you added and run the suite; if it stays green, you have documentation, not a pin.* For BR-6 specifically the rule forces an honest choice, because no test at the current API can distinguish the guard from `Slug` — either extract the name computation (`func (y *YAML) wordFile(k string) (string, error)`) so the guard is exercisable at its own level with a hostile name, or accept that the guarantee *is* inherited and remove "asserted, not inherited from `Slug`" from the Done-when.

**N-3 — `workshop/issues/000004-vocab-capture.md:44` — a Done-when box is ticked for a clause that verification shows is not delivered.**
**This is the 2nd finding in family `undocumented-work-log`** (BR-16). Not asking for this box alone to be unticked. "`--forget` cannot delete outside `words/` — asserted, not inherited from `Slug`" is `[x]`; the assertion does not exist (revert-verified, see BR-6). Adjacently, "`--forget` … leaves `events/` untouched, **asserted by comparing the directory before and after**" is ticked against `storetest/suite.go:129`, which counts events rather than comparing the directory — substantively equivalent for the risk, but not what the box claims. And `## Log` still ends at "Created as part of the `define-learn` project."

*The rule:* **a tick is a claim that evidence exists; record the evidence at the moment of ticking, in `## Log`, naming the test or the command output that establishes it.** Where the evidence is a manual run, paste the output. This rule also covers BR-16's ungrounded "Manual check" tick and generalizes to the commit message: `6eb36f8` describes a `yaml.go` change that the commit does not contain, which is the same failure one layer out.

### 4. Minor findings

- **3rd in family `needless-indirection`** (BR-14, BR-15 both still open): `newStoreHistory(st store.Store, _ store.Clock, warn io.Writer)` (`history_store.go:33`) keeps a dead `store.Clock` parameter that four call sites construct and pass. *The rule covering all three:* when a refactor strips a component's responsibilities, strip the surface that served them in the same commit — a retained parameter, wrapper, or return slot outlives the reason for it and reads as intentional. One pass over `deps.forgetter()`, the `newStore` triple, and this parameter closes the family.
- `workshop/plans/000004-vocab-capture-close-review.md:18` — the committed review artifact opens its `## Review` section with a harness stderr preamble ("Ignoring 6 permissions.allow entries from .claude/settings.json…"). Generated-artifact capture should take the agent's stdout only.
- `atlas/define.md:319` — the prose above the entry-modes table still says `run` "dispatches on argument count into a single shared `defineOnce`"; `-forget` now dispatches before the `NArg` switch. Same instance as BR-17.
- `README.md:83` — "Exit codes: `0` success, `1` no dictionary entry, `2` usage error" now under-describes `1`, which also means "not in the deck".

### 5. Test coverage notes

The suite is green (`go test ./cmd/define/...`, 23.9s) and this round's additions are genuinely load-bearing where they exist — I mutation-checked three of them and all three reddened correctly, which is a real improvement over round 2, where the flagship arity test was blind. The coverage shape is now: `decideCapture` table-tested with zero IO; per-path arity at the `Capturer` seam; one store-level arity test through the real wiring on the raw path; `openStore` tested directly plus one `t.Setenv` end-to-end disk assertion. What remains uncovered is narrow and named in N-2. One further gap worth noting without raising: Done-when #2 ("repeat lookups increment the count") is pinned only at `storetest/suite.go:71`, which tests `Upsert` merge semantics — no test drives two captures of the same word through a capturer and asserts `Lookups == 2`. `merge`'s `max(1, w.Lookups)` makes that robust in practice, so this is a note, not a finding. `TestNoCaptureWritesNothingToDisk` is the model to copy for the missing positive case: build the rig before `t.Chdir`, wire `newStore = openStore`, and assert on the directory.

### 6. Architectural notes

- **ARCH-DRY — pass, with the artifact caveat.** The code side is now clean: `decideCapture` has one caller, `storeCapturer` is the only writer, `lookupAndRender` is the only capture site, and the raw early-return that gave the policy a second home is gone. The remaining duplication is in prose (N-1), which is where ARCH-DRY applied to artifacts keeps failing on this issue — five rounds, four `prose-contradicts-code` instances.
- **ARCH-PURE — pass.** `decideCapture` is a real pure function with a real IO-free table test. `storeCapturer` is a thin shell over it; `openStore` is the boundary and is now injectable through `deps.newStore`, which is what made the env-wiring test possible without touching the developer's filesystem. No business logic leaked into `store/`.
- **ARCH-PURPOSE — flag.** Shadow-sweep on the single source `decideCapture`: `storeCapturer.Capture` derives ✓; `lookupAndRender`'s raw branch now derives ✓ (was the round-2 flag); `openStore`'s `if opt.noCapture` does not derive — and I confirmed it is load-bearing rather than redundant (removing it leaves history persisted, which `TestNoCaptureWritesNothingToDisk` cannot see but `TestOpenStoreUnderOptOut` can), so it is a legitimate second reader of the same input, correctly labelled at `main.go:74`. The flag is elsewhere: the issue's purpose includes two "asserted, not assumed" obligations, and one of them (traversal) still ships as documentation. The pin *is* the deliverable there — it is not a separable follow-up.
- **ARCH-MOCK — pass.** `store.Mem` ships as production code behind the same interface the YAML store implements, `storetest.Suite` runs both, and `Forget` joined the suite in the same commit that introduced it. Production flow and test flow share the boundary. The round-2 flag (`countingCapturer` standing above the writers it was meant to observe) is resolved by keeping it for the question it answers well and adding the store-level test for the one it cannot.
- **For `#5` (ordering by `Lookups`):** the arity invariant it depends on is now genuinely pinned, so `#5` can trust the number. The `-raw` gap in N-2 is the one path where a second writer could reappear unseen — close it before `#5` starts.

### 7. Plan revision recommendations

The plan has no `## Revisions` section; round 2 recommended one and the commit ticked its checkboxes instead. One entry, dated, covering:

- **Line 116** — "consulted from three call sites" is wrong twice over: `decideCapture` has exactly one caller, `storeCapturer.Capture`. (BR-1, open since PQ-6.)
- **Line 133** — "it moves here and `storeHistory` delegates" contradicts line 86's "`storeHistory` therefore stops writing". It does not delegate; it does nothing. (BR-2, open since PQ-10.)
- **Line 181** — "`defineOnce` calls `d.capture.Capture(word, found)`" contradicts Chunk 1's `lookupAndRender`. Rewrite it to point at the Chunk 1 statement rather than restating it, which is the fix PQ-11 asked for and is the same rule as N-1. (BR-3.)
- **`Capturer` signature, Chunk 1** — declared `Capture(word string, found bool)`; shipped as `Capture(word string, found bool, opt options)`.
- **`openHistory`** — renamed `openStore` and now returns `(History, Capturer, store.Store)`; the warn-table row and the Chunk 1 prose still name the old function.
- **`deps` gains `capture`, `deck` and `newStore`** — never stated; record that `run` installs a `noopCapturer` fallback and that both test rigs supply one explicitly. (BR-4.)
- **Task 2 Step 1** — "driven through a counting store" was implemented as a counting *capturer*; record the substitution, that it was wrong, and that a store-level test was added alongside. `lessons.md:144` has the lesson; the plan should carry the fact.
- **Task 3 Step 0** — the traversal assertion and the before/after directory comparison were not delivered as specified; record what shipped instead and which Done-when clauses that leaves unproven.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan line 116 unchanged and now wrong twice over - decideCapture has exactly one caller.
  - id: BR-2
    disposition: not-addressed
    note: |
      Plan line 133 "storeHistory delegates" unchanged; it neither delegates nor writes.
  - id: BR-3
    disposition: not-addressed
    note: |
      Plan line 181 unchanged; signature also drifted to Capture(word, found, opt).
  - id: BR-4
    disposition: not-addressed
    note: |
      Code resolved it (run installs noopCapturer, rigs supply one); the plan still never says so.
  - id: BR-5
    disposition: addressed
    note: |
      Verified by revert - restoring the double write reddens only TestNoDoubleWriteThroughTheRealWiring.
  - id: BR-6
    disposition: not-addressed
    note: |
      yaml.go untouched by 6eb36f8 despite the commit message; guard removal still leaves the store suite green.
  - id: BR-7
    disposition: addressed
    note: |
      Verified both directions - severing the env wiring and gutting openStore each redden a distinct test.
  - id: BR-8
    disposition: addressed
    note: |
      Raw branch now calls Capture; behaviour-identical so no test sees it - carried into the N-2 rule finding.
  - id: BR-9
    disposition: addressed
    note: |
      Atlas rewritten correctly; the same sentence survives in history_store.go - raised as N-1.
  - id: BR-10
    disposition: addressed
    note: |
      Confirmed against go run ./cmd/define -h; usage now states cwd writes and DEFINE_NO_CAPTURE.
  - id: BR-11
    disposition: addressed
    note: |
      /words/ and /events/ added with the reason recorded.
  - id: BR-12
    disposition: addressed
    note: |
      Verified with the built binary - define -forget="" exits 2 with "-forget needs a word".
  - id: BR-13
    disposition: addressed
    note: |
      Verified - now reports "DEFINE_NO_CAPTURE is set, so no deck was opened".
  - id: BR-14
    disposition: not-addressed
    note: |
      deps.forgetter() unchanged at main.go:46.
  - id: BR-15
    disposition: not-addressed
    note: |
      newStore still returns a triple with three nil-merges in run.
  - id: BR-16
    disposition: not-addressed
    note: |
      Issue Log still ends at the 2026-08-20 creation line; no implementation entry, no manual-check evidence.
  - id: BR-17
    disposition: not-addressed
    note: |
      Entry-modes table still lists three invocations; the prose above it is now wrong too.
findings:
  - id: new
    severity: Important
    family: prose-contradicts-code
    title: |
      history_store.go:17 still says Add appends events and upserts words, contradicting line 25 of the same comment
    detail: |
      4th in family (BR-9 atlas, BR-10 --help, BR-17 entry-modes; prevalence 4). Do NOT patch this
      instance. Round 2's I-4 fixed this exact sentence in the atlas and left the code copy eight lines
      above the sentence that refutes it. The rule: a behavioural fact gets exactly one normative home
      and every other mention points at it. Applied here that means DELETING the "Two things are
      deliberately NOT the same here" bullets at history_store.go:15-22 - the atlas owns the split and
      the surviving prose already says everything true - not rewording them into a fifth copy.
  - id: new
    severity: Important
    family: unpinned-invariant
    title: |
      Two fixes landed this round are revert-green, and BR-6's is unpinnable at the current API
    detail: |
      4th in family (BR-5, BR-6, BR-7; prevalence 4). Do NOT patch these instances. Removing the entire
      BR-8 fix - d.capture.Capture at main.go:257 - leaves go test ./cmd/define/... green, because the
      arity test has subtests for one-shot, piped, raw editor, replay and failure but none for -raw, the
      one truth-table row production can now produce. openStore's non-noCapture deck return is likewise
      untested; I verified it only by running the binary. The rule: a fix ships with a test whose failure
      you have OBSERVED by deleting the fix - delete the line, run the suite, and if it stays green you
      wrote documentation. BR-5 and BR-7 show the practice works; these show it was applied selectively.
      For BR-6 the rule forces an honest choice: extract the name computation so the guard is exercisable
      at its own level, or drop "asserted, not inherited from Slug" from the Done-when.
  - id: new
    severity: Important
    family: undocumented-work-log
    title: |
      A Done-when box is ticked for a clause that revert-verification shows is not delivered
    detail: |
      2nd in family (BR-16; prevalence 2). Do NOT just untick this box. Issue line 44 ticks "--forget
      cannot delete outside words/ - asserted, not inherited from Slug"; the assertion does not exist.
      Line 41 ticks "asserted by comparing the directory before and after" against a test that counts
      events instead. Log still ends at the creation line. The rule: a tick claims evidence exists, so
      record the evidence in "## Log" at the moment of ticking, naming the test or pasting the command
      output. Same rule one layer out covers commit 6eb36f8, which describes a yaml.go change the commit
      does not contain.
  - id: new
    severity: Minor
    family: needless-indirection
    title: |
      newStoreHistory keeps a dead store.Clock parameter that four call sites construct and pass
    detail: |
      3rd in family (BR-14, BR-15; prevalence 3). Do NOT patch this instance alone. The rule: when a
      refactor strips a component's responsibilities, strip the surface that served them in the same
      commit - a retained parameter, wrapper or return slot outlives its reason and reads as intentional.
      One pass over deps.forgetter(), the newStore triple and this parameter closes the family.
  - id: new
    severity: Minor
    family: generated-artifact-noise
    title: |
      The committed close-review artifact opens with a harness stderr preamble
    detail: |
      workshop/plans/000004-vocab-capture-close-review.md:18 carries "Ignoring 6 permissions.allow
      entries from .claude/settings.json..." inside the "## Review" section. Artifact capture should
      take the agent's stdout only, or this recurs on every review run in an untrusted workspace.
```

---

## Re-review — 2026-08-21T11:34:57-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 4 — capture looked-up words into the deck |
| repo | tools |
| issue file | workshop/issues/000004-vocab-capture.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4afa1c537819ad662ad149a14bd757ac149772e5..00f9b94bda84a80cdac8f3f8905768dd5713be12 |
| command | sdlc close --issue 4 |
| reviewer | claude |
| timestamp | 2026-08-21T11:34:57-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: REWORK
confidence: high
```

The shipped behavior is correct and I found no correctness bug: the full suite is green (`go test ./cmd/define/...`, 24.2s), capture happens at `lookupAndRender` (the one function every entry path shares — I traced the call graph), there is exactly one writer, and I drove every `-forget` surface end-to-end against a real deck on disk with a sentinel file one level up, which survived. Two of the three revert-green fixes BR-19 named are now genuinely pinned — I verified both by mutation and both go red. What blocks the boundary is BR-6, open since round 2 and now asserted as delivered when it is not. The fix extracted `wordFileName` — a good, properly-tested pure guard — and wired it into `Upsert`, the function BR-6 was *not* about. `YAML.Forget` (`cmd/define/store/yaml.go:326`) still carries its own inline copy of the same rule, still contains the two sub-conditions BR-6 showed can never fire, and is still revert-green: I deleted the entire guard and `go test ./cmd/define/store/...` stayed green, traversal subtest included. Meanwhile the issue's Done-when now states "The guard is now a pure `wordFileName(slug)` tested directly with hostile names" about `--forget`, the `## Log` repeats it, and commit `00f9b94`'s message repeats it again — three records asserting a change that is not at the site they name. That is the third consecutive round in which a claimed fix is not in the tree at the named location, and it is the exact defect BR-20 exists to prevent, recurring in a new form after being reported fixed. Separately, `needless-indirection` went 0-for-3 despite BR-21 specifying "one pass over `deps.forgetter()`, the `newStore` triple and this parameter closes the family," and the plan artifact remains unedited except for checkbox ticks.

### 1. Strengths

- **`wordFileName` is the right shape, and it is genuinely pinned.** `cmd/define/store/yaml.go:180` takes an already-sanitised slug rather than a key, which is what makes `TestWordFileNameRefusesUnsafeNames` (`word_test.go:113`) able to feed it names `Slug` does not produce. This is exactly the "restructure so the guard is exercisable at its own level" option BR-19 offered. The problem is only that `Forget` was not converted with it.
- **The `-raw` capture call is now pinned by an observed failure.** I deleted `d.capture.Capture(word, true, opt)` from the raw branch and `TestCaptureArityIsOnePerLookup/raw_captures_nothing_but_still_asks` went red with `captured 0 time(s) [], want 1`. Round 3's rule applied and it works.
- **`openStore`'s live deck return is pinned too.** Changing the non-`noCapture` return's third value to `nil` reddens `TestOpenStoreWithoutOptOut` at `capture_test.go:352`. Both directions verified.
- **BR-18 was fixed by deletion, not rewording.** `history_store.go:13-22` now carries only the one locally-owned fact (`Prefix` reads once at construction) and explicitly points at `capture.go` / `atlas/define.md` for the rest. That is the "one normative home" rule applied correctly.
- **`-forget` is correct across every surface I could drive.** Present word → `removed sycophantic` / 0; absent → 1; `-forget=""` → 2; `-forget a b` → 2; under `DEFINE_NO_CAPTURE=1` → `DEFINE_NO_CAPTURE is set, so no deck was opened` / 1; `../sentinel.txt`, `../../../etc/passwd` and `..` all → "not in the deck" with the sentinel intact.
- **`workshop/lessons.md:157` records the right operational rule** — "delete the line, run the suite" — plus the mutation-did-not-apply caveat, which is a real and easily-missed failure mode.

### 2. Critical findings

None. The traversal guard's absence is not exploitable: `Slug` sanitises first (`../../../etc/passwd` → `etc-passwd-<hash>`), which I confirmed against the running binary.

### 3. Important findings

**BR-6 (not-addressed) — `cmd/define/store/yaml.go:326` — the fix went to the wrong function, and now the rule has two implementations.**

`wordFileName` is called from exactly one place, `Upsert` (`yaml.go:42`). `Forget` — the only operation that deletes, and the entire subject of BR-6 — still reads:

```go
name := filepath.Base(Slug(k)) + ".yaml"
if name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
```

Three consequences, all verified:
1. **Still revert-green.** Replacing those four lines with `name := Slug(k) + ".yaml"` leaves `go test ./cmd/define/store/...` green. I confirmed the substitution landed before believing the result, per the lesson recorded this round.
2. **The dead sub-conditions BR-6 named are untouched.** `name` always ends in `".yaml"`, so `name == "."` and `name == ".."` cannot fire. Confirmed live: `define -forget ..` returns "not in the deck", never "refusing unsafe name".
3. **New, and caused by the fix: ARCH-DRY is now violated where it was not before.** "What is a safe word file name" has two implementations 280 lines apart in one file, and they already disagree — `wordFileName` rejects a leading `.`, the `Forget` copy does not. ARCH-PURPOSE's shadow-sweep on the new single source finds one consumer deriving (`Upsert`) and one hand-maintained restatement (`Forget`).

*Fix:* `name, err := wordFileName(Slug(k)); if err != nil { return false, err }`, and delete the inline check. One line of wiring closes the duplication, the dead branches, and the Done-when claim at once.

**BR-20 (not-addressed) — `workshop/issues/000004-vocab-capture.md:45-49` — the tick is still false, now in a new way.**

The box reads "`--forget` cannot delete outside `words/`. The guard is now a pure `wordFileName(slug)` tested directly with hostile names." The `--forget` path does not call `wordFileName`. `## Log:120` repeats the claim ("so it became a pure `wordFileName(slug)`"), and `00f9b94`'s commit body repeats it a third time. Round 3 raised this exact defect against `6eb36f8`, which "describe[d] a `yaml.go` change the commit does not contain"; `00f9b94` does contain a `yaml.go` change, but not at the function the record names.

BR-20's rule was right and is not yet in force. Its operational form, now testable: **before ticking a Done-when, name the symbol the evidence exercises and grep that the production path reaches it.** Here `grep -rn wordFileName cmd/define/` returns one production call site and it is not on the `--forget` path.

**NEW [Important] `family-rule-applied-selectively` — the family fixes closed the instance in each finding's title and left the instances enumerated in its body.**

Round 3 escalated three families and stated a rule for each. Measured against the tree:

| finding | instances the finding named | closed |
|---|---|---|
| BR-18 `prose-contradicts-code` | `history_store.go` + BR-9, BR-10, BR-17 | 1 of 4 — BR-17's atlas table still lists three invocations, and its prose still says `run` "dispatches on argument count into a single shared `defineOnce`" |
| BR-19 `unpinned-invariant` | raw capture, `openStore` deck, BR-6 | 2 of 3 |
| BR-21 `needless-indirection` | `deps.forgetter()`, the `newStore` triple, the dead `Clock` param — "one pass over [all three] closes the family" | 0 of 3 |

That is 3 of 10 named instances, and the three closed are the ones in the titles. A fifth instance of `prose-contradicts-code` also arrived unremarked: `README.md:85` still says exit `1` means "no dictionary entry", but `1` now also means "not in the deck" and "no deck was opened".

*The rule:* **an escalated family finding is closed only when every instance it enumerates is disposed, and the response says which were fixed and which were not.** Marking a family finding `addressed` asserts the family is closed, not that the headline site was patched — which is the same substitution (fix the named thing, leave the class) that the escalation mechanism was built to stop. Concretely for the next round: reply to BR-18/BR-19/BR-21 instance-by-instance, not finding-by-finding.

### 4. Minor findings

- **BR-14/BR-15/BR-21 (all not-addressed)** — `deps.forgetter()` (`main.go:46`), the `newStore` triple plus three nil-merges (`main.go:179-193`), and `newStoreHistory(st, _ store.Clock, warn)` (`history_store.go:29`) are all unchanged. Covered by the family rule above; do not patch one.
- **BR-22 (not-addressed)** — the harness preamble is now at `000004-vocab-capture-close-review.md:18` *and* `:251`. It recurred on the next run, exactly as the finding predicted.
- **BR-17 (not-addressed)** — atlas `Entry modes` (`atlas/define.md:317-327`): table omits `define -forget <word>`, prose above it is now wrong.
- `forgetWord` prints a bare `define: %v` on a store error (`main.go:380`) where every neighbouring message carries context (`define: %s: %v`).
- `run` calls `d.newStore` before the `-forget` dispatch, so `--forget` reads the whole event log through `newStoreHistory` for a result it never uses.
- `storeCapturer.mu` guards `warned` only, not the `AppendEvent`/`Upsert` pair. Single-goroutine today; worth a comment saying so, since the type is named "the only writer in the process."

### 5. Test coverage notes

Coverage is materially better than round 3 and the new tests are load-bearing — I mutation-checked three and all three reddened, none reasserted the implementation. The shape now: `decideCapture` table-tested with zero IO; per-path arity at the `Capturer` seam including the `-raw` row; one store-level arity test through the real wiring (`TestNoDoubleWriteThroughTheRealWiring`); `openStore` covered on both branches; one `t.Setenv` end-to-end disk assertion; `Forget` in the shared conformance suite so `Mem` and `YAML` both answer for it. The one remaining hole is the traversal guard (BR-6) — and note the suite's "forget cannot escape the words directory" subtest (`storetest/suite.go:137`) is not a fix for it: it passes identically with the guard deleted, because `Slug` does all the work at that API. `TestWordFileNameRefusesUnsafeNames` is the right test; it just has no production caller on the delete path. Round 3's un-raised note still stands: no test drives two captures of the same word through a capturer and asserts `Lookups == 2` — `storetest/suite.go:71` tests `Upsert` merge semantics, and `merge`'s `max(1, w.Lookups)` makes it robust in practice.

### 6. Architectural notes

- **ARCH-DRY — flag (BR-6).** The capture side is clean: one policy, one caller, one writer, one capture site. The regression is new and was introduced by this round's fix — the safe-filename rule now has two implementations in `yaml.go` that already differ on the leading-dot case. On the artifact side, the family measured at 5 instances (BR-9, BR-10, BR-17, BR-18, README exit codes).
- **ARCH-PURE — pass.** `decideCapture` and `wordFileName` are both real pure functions with real IO-free tests. `openStore` is the boundary and is injectable through `deps.newStore`, which is what makes the env-wiring test possible without touching the developer's filesystem. No logic leaked into `store/`.
- **ARCH-PURPOSE — flag (BR-6).** Shadow-sweep on `decideCapture`: `storeCapturer` derives ✓, the raw branch derives ✓ and is pinned ✓, `openStore`'s `noCapture` read is a legitimate second reader of the same input and is labelled as such ✓. Shadow-sweep on the *new* single source `wordFileName`: `Upsert` derives ✓, `Forget` is a hand-maintained restatement ✗. And the issue's purpose includes an "asserted, not inherited" obligation on the delete path specifically; that pin is the deliverable, not a follow-up.
- **ARCH-MOCK — pass.** `store.Mem` ships as production code behind the same interface, `storetest.Suite` runs both implementations, `Forget` joined the suite in the same commit that introduced it, and production and test flows share the boundary. `countingCapturer` is retained for the question it answers well with the store-level test covering what it cannot see.
- **For `#5` (ordering by `Lookups`):** the arity invariant it depends on is now genuinely pinned through the real wiring on every path production can produce. `#5` can trust the number.

### 7. Plan revision recommendations

`workshop/plans/000004-vocab-capture-plan.md` has no `## Revisions` section and its only edit in this window is `3ca9ab4` ticking checkboxes. Seven gate rounds (PQ-6, PQ-10, PQ-11, PQ-12, BR-1, BR-2, BR-3, BR-4) have raised plan-vs-code drift and two rounds have explicitly recommended a `## Revisions` entry. AGENTS.md §1 requires one: *"Revising a plan artifact mid-stream: append a `## Revisions` section (timestamp + reason + delta), don't overwrite."* Ticking checkboxes is not reconciliation. One dated entry covering:

- **Line 116** — "consulted from three call sites": `decideCapture` has exactly one caller, `storeCapturer.Capture`. (BR-1)
- **Line 133** — "it moves here and `storeHistory` delegates" contradicts line 86's "`storeHistory` therefore stops writing". It does not delegate; it does nothing. (BR-2)
- **Line 181** — "`defineOnce` calls `d.capture.Capture(word, found)`" contradicts Chunk 1's `lookupAndRender`. Rewrite it to *point at* the Chunk 1 statement rather than restating it — restating is what has produced this finding four rounds running. (BR-3)
- **Line 128** — `Capturer` declared as `Capture(word string, found bool)`; shipped as `Capture(word string, found bool, opt options)`.
- **Lines 47, 141** — `openHistory` is now `openStore` and returns `(History, Capturer, store.Store)`.
- **`deps` gains `capture`, `deck`, `newStore`** — never stated; record that `run` installs a `noopCapturer` fallback and both test rigs supply one. (BR-4)
- **Task 2 Step 1** — "driven through a counting store" shipped as a counting *capturer*; record the substitution, that it was wrong, and that a store-level test was added alongside. `lessons.md:144` has the lesson; the plan should carry the fact.
- **Task 3 Step 0** — the before/after directory comparison shipped as an event count (corrected honestly in the issue), and the traversal assertion is still not on the `--forget` path. Record which Done-when clauses that leaves unproven.
- **Chunk 1 core-concepts tables** — omit `noopCapturer`, `wordFileName`, `openStore`, `forgetWord`, `isSet`, and `Store.Forget`/`Mem.Forget`/`YAML.Forget`, all new in this diff.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan line 116 unchanged; decideCapture still has exactly one caller.
  - id: BR-2
    disposition: not-addressed
    note: |
      Plan line 133 "storeHistory delegates" unchanged; it neither delegates nor writes.
  - id: BR-3
    disposition: not-addressed
    note: |
      Plan line 181 unchanged; the plan file's only edit this window is checkbox ticks.
  - id: BR-4
    disposition: not-addressed
    note: |
      Plan still never states that deps gains capture, deck or newStore, nor the noopCapturer fallback.
  - id: BR-6
    disposition: not-addressed
    note: |
      wordFileName was wired into Upsert, not Forget; Forget's guard is unchanged and still revert-green.
  - id: BR-14
    disposition: not-addressed
    note: |
      deps.forgetter() unchanged at main.go:46.
  - id: BR-15
    disposition: not-addressed
    note: |
      newStore still returns a triple with three nil-merges at main.go:179-193.
  - id: BR-16
    disposition: addressed
    note: |
      Implementation Log entry landed and is substantive; the manual-check evidence residual carries under BR-20.
  - id: BR-17
    disposition: not-addressed
    note: |
      atlas/define.md:317-327 unchanged - table still lists three invocations, prose still names defineOnce as the dispatch target.
  - id: BR-18
    disposition: addressed
    note: |
      The restatement was deleted rather than reworded; history_store.go:13-22 now keeps only the locally-owned fact.
  - id: BR-19
    disposition: addressed
    note: |
      Verified by mutation - deleting the raw Capture call and nilling openStore's deck each redden a distinct test.
  - id: BR-20
    disposition: not-addressed
    note: |
      Issue line 45-49, the Log and commit 00f9b94 all state the --forget guard is wordFileName; it is not.
  - id: BR-21
    disposition: not-addressed
    note: |
      newStoreHistory's dead store.Clock parameter unchanged; the one-pass fix the finding specified did not happen.
  - id: BR-22
    disposition: not-addressed
    note: |
      Preamble still at close-review.md:18 and recurred at :251 on the round-3 run, exactly as predicted.
findings:
  - id: new
    severity: Important
    family: family-rule-applied-selectively
    title: |
      The family fixes closed each finding's titled instance and left the instances enumerated in its body
    detail: |
      Measured across round 3: BR-18 named 4 instances and 1 closed; BR-19 named 3 and 2 closed;
      BR-21 named 3, said "one pass over all three closes the family", and 0 closed. That is 3 of 10,
      and the three closed are the ones in the titles. A fifth prose-contradicts-code instance also
      arrived unremarked - README.md:85 still says exit 1 means "no dictionary entry", which now also
      means "not in the deck" and "no deck was opened". The rule: an escalated family finding is closed
      only when every instance it enumerates is disposed, and the response states which were fixed and
      which were not. Marking a family finding addressed asserts the family is closed, not that the
      headline site was patched - which is the same fix-the-named-thing substitution the escalation
      mechanism exists to stop. Next round: reply to BR-18, BR-19 and BR-21 instance-by-instance.
```

---

## Re-review — 2026-08-21T11:49:52-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 4 — capture looked-up words into the deck |
| repo | tools |
| issue file | workshop/issues/000004-vocab-capture.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4afa1c537819ad662ad149a14bd757ac149772e5..96adc2007554a9530f29ece0dbf38e06c01f4e0d |
| command | sdlc close --issue 4 |
| reviewer | claude |
| timestamp | 2026-08-21T11:49:52-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Here is the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code is right, and this round I could prove it rather than take it on trust. All three pins BR-19 disputed survive this round's `storeDeps` refactor and go RED under mutation (deleting the `-raw` `Capture` call, nilling `openStore`'s deck, restoring the double write each redden exactly one distinct test, each verified to have applied *and* compiled). `wordFileName` is genuinely pinned — gutting its guard reddens `TestWordFileNameRefusesUnsafeNames` with 8 assertions — and it now has both filename-deriving call sites, `Upsert` (`yaml.go:42`) and `Forget` (`yaml.go:326`), closing BR-6's ARCH-DRY regression. I drove `-forget` end-to-end against a real deck with a sentinel one level up: traversal keys remove nothing, the sentinel and the event log survive, exit codes are 0/1/2 as documented. BR-23's family sweep was done properly — I checked all ten enumerated instances and all ten are closed. What keeps this from a clean SHIP is the artifact side, which is now the whole residual: **the plan file has not been edited in this entire window except to tick checkboxes** (`git log 4afa1c5..96adc20 -- ...plan.md` returns only `3ca9ab4`), so BR-1/BR-2/BR-3/BR-4 stand exactly as raised in round 1 — and BR-3 is itself an escalated family finding that enumerated four plan lines and instructed a rewrite of each; zero were touched. The same commit that corrected the issue's Done-when about `Forget`'s safety left `Forget`'s own doc comment asserting the retracted claim and naming a `filepath.Base` call it deleted in the same hunk.

### 1. Strengths

- **The honest retraction on BR-6 is the right call, and it is measured, not asserted.** The Done-when now separates three things — `Slug` is the effective guarantee (fuzzed), `wordFileName` is a tested second net, both call sites derive from it by grep — and states outright that bypassing it on the `--forget` path leaves the suite GREEN, so the value is ARCH-DRY not a behavioural pin. I reproduced that exact measurement: mutation applied, `BUILD_OK`, suite green. A Done-when that records its own negative result is stronger than one that claims a pin.
- **The dead sub-conditions are actually gone.** BR-6 noted `name == "."` / `".."` could never fire because `name` already carried `.yaml`. Moving the check onto the slug *before* the suffix makes both reachable, and the test feeds them (`word_test.go:113`).
- **Three mutation-verified pins, re-verified after a refactor that reshaped `openStore`.** `capture_test.go:140`, `:354`, `:283`. A fix that survives the refactor that came after it is the strongest version of this evidence.
- **`needless-indirection` closed 3-for-3.** `deps.forgetter()` deleted, the triple collapsed into `storeDeps`, `newStoreHistory`'s dead `Clock` gone from four call sites — the one-pass fix BR-21 specified, done as specified.
- **`workshop/lessons.md:144-195` are real rules, not patch notes.** "Where you inject the double decides what the test can see"; "a mutation is a result only if it applied AND compiled". Both are transferable and both were earned here.
- **BR-11's fix works in practice.** My own stray run created `words/` and `events/` in the repo root and `git status` stayed clean.

### 2. Critical findings

None. No correctness bug, no crash path, no silent error swallowing. The traversal guard's non-pin is not exploitable: `Slug` sanitises first (`../../../etc/passwd` → `etc-passwd-<hash>`), confirmed against the running binary.

### 3. Important findings

**BR-1/BR-2/BR-3/BR-4 (not-addressed) + NEW `family-rule-applied-selectively` — the family rule was applied to code and not to artifacts.**

**This is the 2nd finding in family `family-rule-applied-selectively`** (BR-23; prevalence 2). Per the escalation rule I am not asking for these four plan lines to be patched in isolation — BR-3 already asked for exactly that and got nothing.

Measured, code side vs artifact side, this round:

| side | enumerated instances | closed |
|---|---|---|
| code (BR-18, BR-19, BR-21) | 10 | **10** |
| plan artifact (BR-1, BR-2, BR-3, BR-4) | 4 findings, ≥7 sites | **0** |

BR-3 is not an ordinary Minor — it is an escalated family finding (`capture-arity-invariant`, 3rd) whose body enumerates plan lines 7, 24, 116, 181 and says "Rewrite 7, 24, 116, 181 to point at the Chunk 1 statement". Line 115-116 still reads "consulted from three call sites" (there is one). Line 133 still reads "`storeHistory` delegates" (it does not delegate; it does nothing). Line 181 still reads "`defineOnce` calls `d.capture.Capture(word, found)`" — the single claim this entire issue exists to disprove, with a signature that also drifted. Lines 47 and 141 still name `openHistory`. Line 128 still declares the two-argument `Capturer`. The plan never states that `deps` gains `capture`, `deck` or `newStore` (BR-4). There is no `## Revisions` section, which AGENTS.md §1 requires for mid-stream plan revision and which three consecutive rounds have recommended.

*The rule, which is BR-23's with the scope clause it was missing:* **every open finding gets an instance-by-instance disposition regardless of which artifact its instances live in — and where the instances live in the plan, the closing move is an AGENTS.md §1 `## Revisions` entry, not a checkbox tick.** BR-23's rule as written ("reply instance-by-instance") was applied faithfully to the three families whose instances a code commit could close, and not at all to the one whose instances live in a markdown file. That is the same fix-what-the-diff-touches substitution one layer out.

**NEW [Important] `prose-contradicts-code` — this round's own edits left two comments describing code they no longer describe, both in files the commit touched.**

**This is the 6th finding in family `prose-contradicts-code`** (BR-9 atlas, BR-10 `--help`, BR-17 entry-modes, BR-18 `history_store.go`, README exit codes; prevalence now 6). Do not patch these two sites.

- `cmd/define/store/yaml.go:317-320` — `Forget`'s doc comment: *"the slug's single-safe-path-element guarantee is **asserted here rather than inherited**: **`filepath.Base` is applied to the slug** before joining, so no key can reach outside `words/` even if `Slug` ever regressed."* Commit `96adc20` deleted the `filepath.Base` call from `Forget` in the same hunk (`grep -rn filepath.Base cmd/define/store/` returns zero hits inside `Forget`), and "asserted rather than inherited" is precisely the claim the same commit rewrote the issue's Done-when to **retract**. A future reader auditing "is the delete path safe?" reads a false security claim on the delete path — the exact harm the whole BR-6/BR-20 arc was about.
- `cmd/define/main.go:55-64` — inserting `type storeDeps` between `openStore`'s doc comment and `openStore` orphaned it. AST-verified: `type storeDeps` now carries *"openStore builds the store-backed dependencies over the WORKING DIRECTORY… DEFINE_NO_CAPTURE means…"*, and `openStore` has an empty doc. The issue's Spec names this comment as one of three homes for the opt-out's cost ("Stated here, in `--help`, and in the README"); it now documents a struct that never reads the environment.

*The rule, which is round 3's with the clause that would have caught both:* **closing a `prose-contradicts-code` instance means sweeping the file the fix touched, not the line the finding named.** Round 3 stated "one normative home per behavioural fact" and applied it to `history_store.go`; both new sites are in files this round edited, and the first is a third restatement of a story `wordFileName`'s own doc comment (`yaml.go:171-179`) and the issue Done-when already tell. The sweep is `git diff --stat` on the commit, file by file.

### 4. Minor findings

- **BR-22 (not-addressed)** — the harness preamble is now at `close-review.md:18`, `:251` **and `:485`**. It recurred a third time on the round-4 run, exactly as the finding predicted twice.
- **NEW `unstated-seam-default` (2nd in family, BR-4; prevalence 2)** — `main.go:77`: `withStore`'s early return `if d.history != nil && d.capture != nil { return d }` silently strands `deck`. The pre-refactor block filled `deck` from `newStore` independently; now supplying history+capture also suppresses the deck. Verified with a probe: `newStore called = false, deck = <nil>`. Not reachable from production (`realDeps` supplies neither) and no test hits it, so not a live bug — but it is a new seam whose rule no comment states and no test pins, introduced by a cleanup whose stated purpose was "the merge was three chances to forget one". *The rule covering this and BR-4:* every field of an injected seam states its own default and how it is filled; a fill-in helper must not make one field's default depend on another field's presence.
- `run` builds store deps before the `-forget` dispatch, so `--forget` loads the entire event log through `newStoreHistory` for a result it never uses — now with a visible symptom: I saw `define: 2026-08-21.yaml: recovered 0 event(s), dropped 1 torn record(s)` printed by a `-forget` invocation that never touches the log.
- `deps.newStore`'s comment claims "no test ever touches the real filesystem"; `TestNoCaptureWritesNothingToDisk` and `TestOpenStoreWithoutOptOut` set it and do (safely, inside `t.TempDir` + `t.Chdir`).
- `forgetWord` prints a bare `define: %v` on a store error (`main.go:398`) where neighbours carry context (`define: %s: %v`).
- `orElse[T comparable]` is a generic helper introduced to replace two nil checks in a commit closing `needless-indirection`.

### 5. Test coverage notes

The suite is green (`go test ./cmd/define/...`, 24.2s) and — unlike round 2 — the tests that matter are load-bearing. I mutation-checked four independently, confirming each applied and compiled before reading the result, and all four reddened on exactly the assertion they should: `TestWordFileNameRefusesUnsafeNames`, `.../raw_captures_nothing_but_still_asks`, `TestOpenStoreWithoutOptOut`, `TestNoDoubleWriteThroughTheRealWiring`. The shape is now sound: `decideCapture` and `wordFileName` table-tested with zero IO; per-path arity at the `Capturer` seam; one store-level arity assertion through the real wiring; `openStore` on both branches; one `t.Setenv` end-to-end disk assertion; `Forget` in the shared conformance suite so `Mem` and `YAML` both answer for it. One gap I am noting rather than raising, for the third round running because `#5` is what makes it matter: **nothing drives two `Capture` calls of the same word and asserts `Lookups == 2`.** `storetest/suite.go:71` asserts that at the `Upsert` level, and `merge`'s `old.Lookups + max(1, w.Lookups)` (`mem.go:43`) makes it correct in practice — but a capturer that stopped calling `Upsert` on the second sighting would pass everything, and `#5` orders by that number. One line inside `TestNoDoubleWriteThroughTheRealWiring` covers it.

### 6. Architectural notes

- **ARCH-DRY — pass on code, flag on artifacts.** Code is clean: one policy (`decideCapture`, one caller), one writer (`storeCapturer`), one capture site (all three `.Capture(` call sites are inside `lookupAndRender` — grep-verified), one filename rule (`wordFileName`, both call sites grep-verified, the drifted inline copy deleted). The duplication that remains is entirely in prose — "why the filename guard is safe" has three homes and two are now false — which is where ARCH-DRY-applied-to-artifacts has failed on this issue for six gate rounds.
- **ARCH-PURE — pass.** `decideCapture` and `wordFileName` are real pure functions with real IO-free tests, both mutation-proven. `openStore` is the boundary and is injectable through `deps.newStore`, which is what makes the env-wiring test possible without touching the developer's filesystem. `storeCapturer` is a thin shell over the policy. Nothing leaked into `store/`.
- **ARCH-PURPOSE — pass on code.** Shadow-sweep on `decideCapture`: `storeCapturer` derives ✓, the raw branch derives ✓ and is pinned ✓, `openStore`'s `noCapture` read is a labelled second reader of the same *input* rather than a restatement of the *policy* ✓. Shadow-sweep on `wordFileName`: `Upsert` ✓, `Forget` ✓ — this is the consumer that was hand-maintained last round and now derives. The "asserted, not inherited" obligation was discharged by honest retraction with a measured result, which is the branch BR-19 explicitly offered as acceptable. The remaining non-deriving consumer is the plan, which still describes a different design (finding above).
- **ARCH-MOCK — pass.** `store.Mem` ships as production code behind the same interface the YAML store implements, `storetest.Suite` runs both, `Forget` joined the suite in the commit that introduced it, and the audio CDN has a fixture-backed fake at the same seam. Production flow and test flow share the boundary. `countingCapturer` is retained for the per-path question it answers well, with the store-level test covering what it structurally cannot see — the round-2 flag is properly resolved rather than papered over.
- **For `#5`:** the arity invariant it depends on is pinned through the real wiring on every path production can produce, and the pin survived a refactor. `#5` can trust `Lookups`. Add the two-lookup assertion before ordering by it.

### 7. Plan revision recommendations

`workshop/plans/000004-vocab-capture-plan.md` needs its first `## Revisions` entry — the finding above is about the *pattern*, but the concrete delta is unchanged from what rounds 2, 3 and 4 each specified, so here it is once more, consolidated. One dated entry covering:

- **Line 115-116** — "consulted from three call sites": `decideCapture` has exactly one caller, `storeCapturer.Capture`. (BR-1)
- **Line 132-133** — "it moves here and `storeHistory` delegates" contradicts line 86's "`storeHistory` therefore stops writing". It does not delegate; it does nothing. (BR-2)
- **Line 181** — "`defineOnce` calls `d.capture.Capture(word, found)`" contradicts Chunk 1's `lookupAndRender`. **Rewrite it to point at the Chunk 1 statement rather than restating it** — restating is what has produced this finding five rounds running. (BR-3)
- **Line 128** — `Capturer` declared `Capture(word string, found bool)`; shipped with a third `opt options`.
- **Lines 47, 141** — `openHistory` is now `openStore` and returns a `storeDeps` value.
- **`deps` gains `capture`, `deck`, `newStore`** — never stated; record the `noopCapturer` fallback, and (new this round) that `withStore` returns early when history and capture are both supplied. (BR-4)
- **Task 2 Step 1** — "driven through a counting store" shipped as a counting *capturer*; record the substitution, that it was wrong, and that a store-level test was added alongside. `lessons.md:144` has the lesson; the plan should carry the fact.
- **Task 3 Step 0** — the before/after directory comparison shipped as an event count, and the traversal assertion was resolved by *retracting* the claim rather than delivering it. Record both, and that the residual guarantee is `Slug`'s.
- **Chunk 1 core-concepts tables** — omit `noopCapturer`, `wordFileName`, `openStore`, `storeDeps`, `withStore`, `orElse`, `forgetWord`, `isSet`, and `Store.Forget`/`Mem.Forget`/`YAML.Forget`, all new in this diff.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan line 115-116 unchanged; the plan file's only edit in the whole window is 3ca9ab4's checkbox ticks.
  - id: BR-2
    disposition: not-addressed
    note: |
      Plan line 133 "storeHistory delegates" unchanged; it neither delegates nor writes.
  - id: BR-3
    disposition: not-addressed
    note: |
      Lines 7, 24, 116, 181 all unchanged - an escalated family finding that enumerated four sites and closed none.
  - id: BR-4
    disposition: not-addressed
    note: |
      Plan still never states deps gains capture, deck or newStore, nor the noopCapturer fallback.
  - id: BR-6
    disposition: addressed
    note: |
      wordFileName now serves Upsert AND Forget (grep-verified); guard is mutation-RED; Done-when honestly retracts the "asserted" claim with a measured GREEN I reproduced.
  - id: BR-14
    disposition: addressed
    note: |
      deps.forgetter() deleted; forgetWord tests d.deck == nil directly.
  - id: BR-15
    disposition: addressed
    note: |
      Collapsed into a storeDeps value and one withStore call, as the finding specified.
  - id: BR-17
    disposition: addressed
    note: |
      Entry-modes table now has four rows including define -forget, and the prose above it names lookupAndRender.
  - id: BR-20
    disposition: addressed
    note: |
      Both false ticks rewritten to state what is actually asserted; Log carries two substantive entries with a per-instance table.
  - id: BR-21
    disposition: addressed
    note: |
      newStoreHistory's Clock parameter removed along with all four call sites' constructions.
  - id: BR-22
    disposition: not-addressed
    note: |
      Preamble now at close-review.md:18, :251 AND :485 - a third occurrence added by the round-4 run.
  - id: BR-23
    disposition: addressed
    note: |
      Verified all ten enumerated instances across BR-18/BR-19/BR-21 are closed, and the response replies instance-by-instance.
findings:
  - id: new
    severity: Important
    family: family-rule-applied-selectively
    title: |
      The family rule was applied to every code instance and to no artifact instance
    detail: |
      2nd in family (BR-23; prevalence 2). Do NOT patch the four plan lines in isolation - BR-3 already
      asked for exactly that and got nothing. Measured this round: code-side families closed 10 of 10
      enumerated instances; artifact-side closed 0 of 4 findings covering at least 7 sites. BR-3 is
      itself an escalated family finding whose body enumerates plan lines 7, 24, 116 and 181 and says
      "rewrite [them] to point at the Chunk 1 statement"; none were touched, and the plan file's only
      edit in the entire window is 3ca9ab4's checkbox ticks. There is still no "## Revisions" section,
      which AGENTS.md section 1 requires and three rounds have recommended. The rule is BR-23's with
      the scope clause it was missing: every open finding gets an instance-by-instance disposition
      regardless of which artifact its instances live in, and where they live in the plan the closing
      move is a "## Revisions" entry, not a checkbox tick.
  - id: new
    severity: Important
    family: prose-contradicts-code
    title: |
      This round's own edits left two comments describing code they no longer describe
    detail: |
      6th in family (BR-9, BR-10, BR-17, BR-18, README exit codes; prevalence 6). Do NOT patch these
      two sites. cmd/define/store/yaml.go:317-320 - Forget's doc still says the guarantee is "asserted
      here rather than inherited: filepath.Base is applied to the slug", but 96adc20 deleted that
      filepath.Base call in the same hunk, and the same commit rewrote the issue Done-when to RETRACT
      "asserted rather than inherited". A false safety claim on the delete path is the exact harm the
      BR-6/BR-20 arc was about. cmd/define/main.go:55-64 - inserting type storeDeps between openStore's
      doc comment and openStore orphaned it; AST-verified, type storeDeps now carries "openStore builds
      the store-backed dependencies..." and openStore has no doc, while the Spec names that comment as
      one of three homes for the opt-out's cost. The rule is round 3's with the missing clause: closing
      a prose-contradicts-code instance means sweeping the file the fix touched, not the line the
      finding named. Both sites are in files this commit edited, and the first is a third restatement of
      a story wordFileName's own doc comment and the Done-when already tell.
  - id: new
    severity: Minor
    family: unstated-seam-default
    title: |
      withStore's early return silently strands deck when history and capture are both supplied
    detail: |
      2nd in family (BR-4; prevalence 2). Do NOT patch this instance alone. main.go:77 returns early
      when history and capture are non-nil, so newStore is never consulted and deck stays nil - the
      pre-refactor block filled deck independently. Verified with a probe: "newStore called = false,
      deck = <nil>". Not reachable from production (realDeps supplies neither) and no test hits it, so
      not a live bug, but it is a new seam whose rule no comment states and no test pins, introduced by
      a cleanup whose stated purpose was that "the merge was three chances to forget one". The rule
      covering this and BR-4: every field of an injected seam states its own default and how it is
      filled, and a fill-in helper must not make one field's default depend on another field's presence.
```

---

## Re-review — 2026-08-21T12:01:34-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 4 — capture looked-up words into the deck |
| repo | tools |
| issue file | workshop/issues/000004-vocab-capture.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4afa1c537819ad662ad149a14bd757ac149772e5..e940f987cc8d6e0c93fd1c44eae3184b81a791eb |
| command | sdlc close --issue 4 |
| reviewer | claude |
| timestamp | 2026-08-21T12:01:34-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I've completed the review. Cleanup verified: scratch worktree removed, tree clean.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Six rounds in, the code is right and I proved it rather than trusted it. Four independent mutations all landed, compiled, and reddened exactly one test each: gutting `wordFileName`'s guard (8 assertions), deleting the `-raw` `Capture` call, nilling `openStore`'s live deck, and adding a second writer *below* the `Capturer` seam (which reddens `TestNoDoubleWriteThroughTheRealWiring` while the per-path arity subtests stay green — exactly as the atlas documents). `go test -race` and `go vet` are green across both packages. I drove every `-forget` surface against a real deck with a sentinel one level up: traversal keys remove nothing, sentinel and event log survive, exit codes 0/1/2 as the README now documents. Both BR-25 sites are genuinely fixed — `Forget`'s doc no longer claims a `filepath.Base` that does not exist and does derive from `wordFileName` (grep-verified), and `openStore`'s doc is re-attached to `openStore` (AST-verified, `storeDeps` carries its own). The plan finally got its `## Revisions` section, closing the artifact side that went 0-for-4 last round. What keeps this off a clean SHIP is one ticked Done-when that nothing pins: **"Repeat lookups increment the count rather than duplicating the word"** — I made `storeCapturer` silently skip `Upsert` on a repeat sighting and the *entire* suite stayed green (`ok github.com/xianxu/tools/cmd/define 24.084s`). `#5` orders by that number. It is a one-line fix inside a test that already exists.

### 1. Strengths

- **The pins survived the refactor that came after them.** `storeDeps`/`withStore` reshaped `openStore` in the previous round, and all three capture-side pins still go RED. A fix that outlives the next refactor is the strongest form of this evidence.
- **`TestNoDoubleWriteThroughTheRealWiring` (`capture_test.go:283`) does the job its predecessor could not.** My below-the-seam mutation reddened it with `Lookups = 2` and `got 2 events` while `TestCaptureArityIsOnePerLookup` stayed green — the exact asymmetry `atlas/define.md:283-289` claims, verified rather than asserted.
- **`wordFileName` now has both filename-deriving call sites** (`yaml.go:42`, `yaml.go:325`) and one honest doc comment; the Done-when's retraction (`issue:48-60`) separates "Slug is the effective guarantee, fuzzed" from "wordFileName is a second net, RED on removal" from "both derive, grep-verified, and bypassing it leaves the suite GREEN." A Done-when that records its own negative result is worth more than one that claims a pin.
- **The `## Revisions` section is substantive, not ceremonial.** It names the four design deltas *and* the process failure that produced five rounds, and `workshop/lessons.md:186-195` carries the transferable form. The artifact side went from 0/4 to substantially closed in one commit.
- **`--help`, README and atlas all now carry the cwd-writes claim and `DEFINE_NO_CAPTURE`'s real cost.** I read all three; they agree with each other and with the code. Docs gate: pass.

### 2. Critical findings

None. No correctness bug, no crash path, no silent error swallowing. Traversal is not exploitable — `Slug` sanitises first, confirmed against the running binary.

### 3. Important findings

**NEW — `unpinned-invariant` (5th in family; prevalence 5) — Done-when #2 is ticked and nothing pins it through the capture path.**

Per the escalation rule I am not asking for this instance to be patched in isolation. Measured, this round:

```
mutation: storeCapturer.Capture skips Upsert on a repeat sighting of the same word
result:   MUTATION_APPLIED, BUILD_OK
          ok github.com/xianxu/tools/cmd/define        24.084s
          ok github.com/xianxu/tools/cmd/define/store  (cached)
```

`issue:42` ticks "Repeat lookups increment the count rather than duplicating the word." The only `Lookups` assertions in the tree are `storetest/suite.go:71` (`Upsert` merge semantics — a different question) and `capture_test.go:282` (`== 1`). Nothing drives two captures of one word and asserts the count moved. Round 5 noted this in prose for the third round running and declined to raise it as "robust in practice"; the mutation shows the practice is unpinned, and `#5` is the consumer.

*The rule, which is BR-19's with the clause it was missing:* **the delete-the-line discipline was applied to fixes and never to Done-when ticks.** A tick is a behavioural claim exactly as a fix is; both deserve a mutation whose failure you have observed. That single clause also unifies this family with `undocumented-work-log` (BR-20), whose rule was "a tick claims evidence exists" — the two findings have been chasing the same rule from opposite ends for four rounds. Applied to the remaining ticks: #1 and #3-#6 are each mutation-backed (I verified #4's and #6's); #2 is the one that is not.

*Cheap close:* two lines inside `TestNoDoubleWriteThroughTheRealWiring`, which already has the real wiring over a real store — a second `runEditor`/`Capture` of the same word and `Lookups == 2`.

### 4. Minor findings

- **BR-4 / BR-26 (both not-addressed)** — `main.go:67`'s `if d.history != nil && d.capture != nil { return d }` still strands `deck`, no comment states the rule, no test enters the branch (I traced every `deps` literal: the rigs that set both call `runEditor` directly and never reach `withStore`). Revisions §3 names the three new fields but not their defaults, which was the second half of BR-4's ask and is BR-26's whole rule. Covered by the existing family; do not patch one site.
- **NEW `prose-contradicts-code` (7th in family; prevalence 7)** — `main.go:35-36`: *"Tests leave it nil and get in-memory defaults, so no test ever touches the real filesystem."* `capture_test.go:316` sets `rig.deps.newStore = openStore`, and four test files use `t.TempDir()`. Both clauses false. It sits ~30 lines above the comment this very commit moved, in the file the commit's own message says it swept. *The rule, and the mechanism that defeated it:* round 5 named this defect in its **Minor prose list and never gave it a `BR-` id** — and 0 of that list's 5 items were addressed, while 100% of the id'd findings were. The sweep is not failing on attention, it is failing because the response works the machine-read `findings:` block. So the corrective is on both sides: the reviewer puts every stated defect in the block (I am doing that now), and the sweep becomes mechanical — for each file in `git diff --stat`, read every comment that makes a claim about *another* symbol and grep that symbol.
- **BR-22 (not-addressed, and not fixable here)** — the harness preamble is now at `close-review.md:18`, `:251`, `:485` **and `:674`**: a fourth occurrence, exactly as predicted twice. The generator is `sdlc`, whose source is `/Users/xianxu/workspace/ariadne/cmd/sdlc` — a peer repo. No commit in `tools` can close this. The actionable move is `sdlc issue new` in **ariadne** for "artifact capture takes agent stdout only"; re-raising it here every round cannot converge.
- Still open from round 5's un-id'd list, now recorded so they can be disposed: `run` builds store deps at `main.go:212` before the `-forget` dispatch at `:214`, so `--forget` loads the whole event log for a result it never uses; `forgetWord` prints a bare `define: %v` at `:398` where neighbours carry context; `orElse[T comparable]` is a generic helper introduced by the commit that closed `needless-indirection`.

### 5. Test coverage notes

The suite is green with `-race` (24.4s / 25.7s) and the load-bearing tests are load-bearing — four mutations, each confirmed to have applied *and* compiled before I read the result, each reddening exactly the assertion it should. Shape: `decideCapture` and `wordFileName` table-tested with zero IO; per-path arity at the `Capturer` seam including the `-raw` row; one store-level arity assertion through the real wiring that provably sees writers below the seam; `openStore` on both branches; one `t.Setenv` end-to-end disk assertion; `Forget` in the shared conformance suite so `Mem` and `YAML` both answer. The single hole is Done-when #2 above, and it is the last one — I mutated for it specifically because it is the only ticked box I could not find an assertion for. Note the conformance suite's "forget cannot escape the words directory" subtest is still not a pin for the guard (it passes with `wordFileName` bypassed, because `Slug` does the work at that API) — the issue now says so in writing, which is the honest resolution.

### 6. Architectural notes

- **ARCH-DRY — pass.** One policy (`decideCapture`, one caller), one writer (grep-verified: `c.st.AppendEvent` and `c.st.Upsert` in `capture.go` are the only store writes outside `store/`), one capture site (all three `.Capture(` calls are inside `lookupAndRender`), one filename rule with both consumers deriving. The prose duplication that drove six rounds of this family is down to one stale clause (Minor above).
- **ARCH-PURE — pass.** `decideCapture` and `wordFileName` are genuine pure functions with IO-free tests, both mutation-proven. `openStore` is the boundary and is injectable through `deps.newStore`, which is what makes the env-wiring test possible. Nothing leaked into `store/`.
- **ARCH-PURPOSE — pass.** Shadow-sweep on `decideCapture`: `storeCapturer` derives ✓, the raw branch derives and is pinned ✓, `openStore`'s `noCapture` read is a labelled second reader of the same *input*, not a restatement of the *policy* ✓. On `wordFileName`: `Upsert` ✓, `Forget` ✓. The last non-deriving consumer was the plan, and the `## Revisions` entry closes it. The "asserted, not inherited" obligation was discharged by measured retraction — the branch BR-19 explicitly offered.
- **ARCH-MOCK — pass.** `store.Mem` ships as production code behind the interface `YAML` implements, `storetest.Suite` runs both, `Forget` joined it in the commit that introduced it, and the owned backend boots from any portable folder (`store.NewYAML(dir, warn)`, tests use `t.TempDir`) — no production configuration required. `countingCapturer` is kept for the per-path question it answers well, with the store-level test covering what it structurally cannot see. Production and test flows share the boundary.
- **For `#5`:** `Lookups` is trustworthy for *arity* (one lookup, one write — pinned through the real wiring on every producible path). It is **not yet** pinned for *accumulation*. Close the finding above before ordering by it.

### 7. Plan revision recommendations

The `## Revisions` section three rounds asked for now exists and covers BR-1, BR-2, BR-3 and the field half of BR-4. Two residuals for the same entry, not a new one:

- **Append to Revisions §3** — the seam defaults: `withStore` installs `&memHistory{}` and `noopCapturer{}` when a caller supplies neither, and **returns early when `history` and `capture` are both non-nil, leaving `deck` nil.** That is the unstated rule BR-4 asked for and BR-26 escalated.
- **One consistency note.** AGENTS.md §1 says append, don't overwrite; this commit did both — five in-place corrections (lines 47, 128, 133, 141, 181) *plus* the Revisions section. The result is more accurate for a future reader, so I am not raising it, but the mixed convention means `Chunk 1`'s line 115-116 ("consulted from three call sites") and line 24 ("a `Capturer` that `storeHistory` also uses") still read wrong in place while Revisions §1/§2 correct them. Either finish the in-place pass or state at the top of Chunk 1 that Revisions supersedes it.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Revisions section 1 states "There is one call site" explicitly; the AGENTS.md section 1 append-not-overwrite move BR-24 specified.
  - id: BR-2
    disposition: addressed
    note: |
      Line 133 now reads "stops writing entirely - it does not delegate, it does nothing", plus Revisions section 2.
  - id: BR-3
    disposition: addressed
    note: |
      Line 181 rewritten to lookupAndRender with the three-arg signature; lines 7/24/116 covered by Revisions rather than in place.
  - id: BR-4
    disposition: not-addressed
    note: |
      Revisions section 3 names the three new deps fields but never states their defaults - the exact half BR-4 asked for and BR-26 escalated.
  - id: BR-22
    disposition: not-addressed
    note: |
      Fourth occurrence (lines 18, 251, 485, 674). Generator is ariadne/cmd/sdlc, a peer repo - unfixable from tools; needs an ariadne issue.
  - id: BR-24
    disposition: addressed
    note: |
      Plan substantively edited with a Revisions section; 6 of 7 enumerated artifact instances closed, the 7th carried under BR-4/BR-26.
  - id: BR-25
    disposition: addressed
    note: |
      Both sites verified - Forget's doc derives from wordFileName (grep), openStore's doc re-attached to openStore (AST).
  - id: BR-26
    disposition: not-addressed
    note: |
      main.go:67 early return unchanged; no comment states deck's default, no test enters the branch.
findings:
  - id: new
    severity: Important
    family: unpinned-invariant
    title: |
      Done-when 2 "repeat lookups increment the count" is ticked and no test pins it through the capture path
    detail: |
      5th in family (BR-5, BR-6, BR-7, BR-19; prevalence 5). Do NOT patch this instance alone. Verified by
      mutation: making storeCapturer skip Upsert on a repeat sighting of the same word leaves the ENTIRE suite
      green (MUTATION_APPLIED, BUILD_OK, ok cmd/define 24.084s). The only Lookups assertions are
      storetest/suite.go:71 (Upsert merge semantics, a different question) and capture_test.go:282 (== 1);
      nothing drives two captures of one word and asserts the count moved. Issue 5 orders by this number.
      The rule is BR-19's with the clause it was missing - the delete-the-line discipline was applied to FIXES
      and never to Done-when TICKS, though a tick is the same kind of behavioural claim. That clause also
      unifies this family with undocumented-work-log, whose rule (BR-20, "a tick claims evidence exists") has
      been chasing the same thing from the other end for four rounds. Cheap close: two lines inside
      TestNoDoubleWriteThroughTheRealWiring, which already has real wiring over a real store.
  - id: new
    severity: Minor
    family: prose-contradicts-code
    title: |
      main.go:35 claims no test touches the real filesystem, in the file this commit says it swept
    detail: |
      7th in family (BR-9, BR-10, BR-17, BR-18, README exit codes, BR-25; prevalence 7). Do NOT patch this
      site. deps.newStore's comment says "Tests leave it nil and get in-memory defaults, so no test ever
      touches the real filesystem"; capture_test.go:316 sets rig.deps.newStore = openStore, and four test
      files use t.TempDir(). Both clauses false, ~30 lines above the comment this commit moved. The mechanism
      that defeated BR-25's sweep rule, measured: round 5 stated this defect in its Minor PROSE list and never
      gave it a BR id - 0 of that list's 5 items were addressed while 100 percent of the id'd findings were.
      So the rule needs both halves: the reviewer puts every stated defect in the machine-read findings block
      (done this round for all five), and the sweep becomes mechanical rather than attentional - for each file
      in git diff --stat, read every comment making a claim about another symbol and grep that symbol.
  - id: new
    severity: Minor
    family: generated-artifact-noise
    title: |
      BR-22 cannot be closed from this repo - the generator lives in the ariadne peer
    detail: |
      Recording this so BR-22 stops recurring undisposed. The close-review artifact is written by sdlc, whose
      source is /Users/xianxu/workspace/ariadne/cmd/sdlc; no commit in tools can change what it captures. The
      preamble is now at lines 18, 251, 485 and 674 - one new occurrence per review round, exactly as predicted
      at rounds 3 and 4. The actionable move is an ariadne issue for "artifact capture takes agent stdout only",
      referenced from this issue's Log, rather than a fifth not-addressed disposition here.
```

---

## Re-review — 2026-08-21T12:15:28-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 4 — capture looked-up words into the deck |
| repo | tools |
| issue file | workshop/issues/000004-vocab-capture.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4afa1c537819ad662ad149a14bd757ac149772e5..42cc96d1f93e0521220a1b8fcc6cf161ff11241b |
| command | sdlc close --issue 4 |
| reviewer | claude |
| timestamp | 2026-08-21T12:15:28-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: REWORK
confidence: high
```

Seven rounds in, the code is right and this round I could verify the two claims the commit makes rather than trust them: both new pins go RED under mutations I confirmed applied *and* compiled, each reddening exactly one test — skipping `Upsert` on a repeat sighting reddens only `TestRepeatLookupsIncrementThroughCapture` (`Lookups = 1 after three lookups, want 3`), and giving `Capturer.Capture` an error return that `lookupAndRender` propagates reddens only `TestFailingStoreStillDefinesAndExitsZero` (`exit = 1, want 0`). `go test`, `-race`, `go vet` and a `GOOS=linux CGO_ENABLED=0` build are all green, and I drove capture end-to-end against a real directory: a failed lookup writes one `found: false` event and no `words/`, `DEFINE_NO_CAPTURE=1` writes nothing at all, `--forget` removes the word and leaves the log. BR-27 is genuinely closed. **What blocks the boundary is one thing, and it is in the HEAD commit itself: `42cc96d` committed a 9,616,546-byte Mach-O arm64 executable at `cmd/define/define`.** It is the largest object in the repo by ~97× (next largest blob: 99 KB), it roughly doubles `.git` (currently 9.3 M), and `.gitignore`'s own opening comment says this exact class already happened twice and that "Both paths are ignored so neither recurs" — a claim this commit falsifies at a third path. Unlike every other open item, it is cheap now (`git rm --cached cmd/define/define` + amend, the branch is unmerged) and permanent after merge. Everything else below is non-blocking.

### 1. Strengths

- **Both new pins survive mutation, and the mutations were verified to land.** `capture_test.go:370` and `:394`. Round 6's own lesson — "a mutation is a result only if it applied AND compiled" — applied to its own work, and it holds up under independent re-execution.
- **The Done-when audit was real, not a response to the named box.** BR-27 named #2; the commit found #6 unpinned as well, by audit. That is the family rule working the way six rounds of findings asked for, and the Log says which instance was found by audit rather than by being named.
- **Live behaviour matches every user-facing claim I checked.** Failed lookup → event only, exit 1; opt-out → empty directory; `--forget` → word gone, `events/` intact, exit 0; event timestamps keep their `-07:00` offset while the day file is named in UTC, exactly as `atlas/define.md` says `#8` will depend on.
- **`ARCH-PURE` is genuinely earned.** `decideCapture` (`capture.go:26`) and `wordFileName` (`yaml.go:180`) are pure, table-tested with zero IO, and both mutation-proven; `openStore` is the boundary and is injectable through `deps.newStore`, which is what makes the env-wiring test possible without touching the developer's filesystem.
- **`Forget` joined `storetest.Suite`** rather than getting per-implementation tests, so `Mem` and `YAML` both answer for case-insensitivity, absence-is-not-an-error and events-survive — ARCH-MOCK done properly.

### 2. Critical findings

**C-1 — `cmd/define/define` — a 9.6 MB build artifact is committed in the commit under review.**

**This is the 2nd finding in family `writes-to-cwd-unignored`** (BR-11; prevalence 2). Per the escalation rule I am not asking for this instance to be patched — but the instance is also irreversible after merge, so it needs both halves.

Measured:

```
git ls-files cmd/define/define        → tracked
git log --diff-filter=A -- …/define   → 42cc96d  (HEAD, this commit)
file …                                → Mach-O 64-bit executable arm64
largest blobs                         → 9616546  cmd/define/define
                                        99336    (next largest, a markdown file)
```

Three consequences: every clone pays for it forever; a Linux or amd64 checkout gets a 9.6 MB file that cannot run; and because it is *tracked*, `cd cmd/define && go build` now dirties `git status` on every developer build — the working-tree hazard `.gitignore` was extended twice to prevent.

*The rule, which covers BR-11 and this:* **an artifact the developer workflow drops into the working tree gets ignored by a pattern that covers every directory it can be produced in, and the ignore is added when the tool learns to produce it — not after the first one is committed.** `.gitignore` has been patched three times by this rule's absence and each patch was path-anchored to the one place it had already happened: `bin/`, then `/define` (root only), then `/words/` and `/events/`. `cmd/define/define` is not matched by any of them, because `go build` in a `main` package directory writes the binary *there*. The pattern that closes the class is un-anchored (`define` and/or `cmd/*/define`), not a fourth anchored line. The second half, since the ignore alone would not have caught this one — it was `git add`-ed before any ignore existed: **read `git show --stat` before committing**; a `Bin 0 -> 9616546 bytes` line is visible at a glance and appears in this commit's own stat output.

*Fix, before this branch merges:* `git rm --cached cmd/define/define`, add the un-anchored ignore, `git commit --amend`. The blob then never enters shared history.

### 3. Important findings

**I-1 — `cmd/define/store/storetest/suite.go:140-142` — the retracted safety claim survives in a third file, and it is false by measurement.**

**This is the 8th finding in family `prose-contradicts-code`** (BR-9, BR-10, BR-17, BR-18, README exit codes, BR-25, BR-28; prevalence 8). Do NOT patch this site.

The subtest comment reads: *"the guarantee is asserted here rather than inherited from `Slug` — a regression in `Slug` must fail HERE, loudly."* That is the exact sentence `96adc20` deleted from `YAML.Forget`'s doc (BR-25) and the exact claim the same commit rewrote the issue's Done-when to **retract**. `wordFileName`'s own doc (`yaml.go:174-179`) now says the opposite in plain words — "no test could distinguish the guard from `Slug`, and removing it left every test green."

Verified by mutation rather than argued: I regressed `Slug` completely (`return strings.ReplaceAll(key, " ", "-")`, no sanitising at all), confirmed it applied and compiled, and ran the store package. Red: `TestSlugDoesNotMergeHyphenAndSpace`, `TestSlugIsAlwaysOneSafePathElement`, `FuzzSlugIsSafe` (5 seeds). **Green: `Suite/forget cannot escape the words directory`** — the subtest whose comment claims it must fail loudly. It cannot: `_ = err` discards the only signal that would differ, so the assertions pass whether the guard is present, absent, or bypassed.

A second site the same sweep turns up: `main.go:27`, `capture.go:50` and `atlas/define.md:283` each state absolutely that `storeCapturer` is *"the only thing in the process that writes to the store"* — while `forgetWord` → `d.deck.Forget(word)` (`main.go:396`) deletes a word file, which is a store mutation performed outside the capturer. `deps.deck`'s own comment three lines below `main.go:27` acknowledges the second mutator exists. A reader of `#5` or `#8` auditing "what can change this deck?" is told three times that the answer is one thing.

*The rule, which is BR-28's with the scope clause that would have caught this one:* BR-28 made the sweep mechanical — "for each file in `git diff --stat`, read every comment making a claim about another symbol and grep that symbol." `storetest/suite.go` **is** in `git diff --stat` for this window, and grepping `Slug` from that comment lands directly on `wordFileName`'s contradicting doc. So the rule is right and was not run; the gap is that it has never been executed as written on the *full* changed-file list, only on the files a finding named. Run it once over all 14 changed `.go`/`.md` files and this family closes.

### 4. Minor findings

- **BR-28 (not-addressed)** — `main.go:36` still reads "Tests leave it nil and get in-memory defaults, so no test ever touches the real filesystem." `capture_test.go:316` sets `rig.deps.newStore = openStore`; five test files use `t.TempDir()`. Both clauses false, unchanged. Covered by I-1's rule; do not patch this site alone.
- **BR-4 / BR-26 (both not-addressed)** — `withStore`'s early return (`main.go:65-67`) still strands `deck`, and Revisions §3 still names the three new `deps` fields without stating any of their defaults. Re-verified this round with a probe: replacing the early return with `panic(...)` leaves the **entire suite green**, so no test enters the branch. Unchanged from round 6's disposition.
- **NEW `needless-indirection` (4th in family; prevalence 4)** — `History.Add(line string, found bool)` (`history.go:16`): the `found` parameter is now ignored by **both** implementations (`history.go:28` and `history_store.go:50` both take `_ bool`) while the sole caller computes `code == 0` to supply it (`replraw.go:171`). It existed only because `storeHistory.Add` decided event-vs-deck; `#4` moved that decision and left the parameter. Same rule as BR-21 verbatim — *strip the surface a refactor's removed responsibility served, in the same commit* — and it was missed because it lives in `history.go`, which the diff never touched, so BR-28's file-based sweep could not reach it. That is the extension the rule needs: sweep the changed files **and the interface + callers of every symbol whose responsibilities the diff moved.** Second instance of the same family this round, weaker and reasonably `withdrawn`: `orElse[T comparable]` (`main.go:83`) is a generic helper introduced by the commit that closed `needless-indirection`, replacing two nil checks with six lines.
- **NEW `eager-dependency-construction`** — `run` calls `d.withStore(opt, stderr)` at `main.go:212`, *before* the `-forget` dispatch at `:214`, so `--forget` constructs `storeHistory` and reads the whole event log for a result it never uses. Visible symptom, reproduced with a seeded directory: `define: 2026-08-21.yaml: recovered 0 event(s), dropped 1 torn record(s)` printed by a `-forget` run that never touches the log. Stated in round 6's prose and never given an id; recording it so it can be disposed.
- **NEW `misleading-error-text` (2nd in family; prevalence 2)** — `forgetWord` prints a bare `define: %v` on a store error (`main.go:398`) where every neighbouring message carries the operand (`define: %s: %v`). *The rule covering this and BR-13:* a user-facing error names both the condition and the operand it failed on; BR-13's message named the wrong condition, this one names no operand.
- **BR-22 (not-addressed) / BR-29 (not-addressed)** — the harness preamble is now at `close-review.md:18, 251, 485, 674, 878` — a fifth occurrence, one per round, exactly as predicted at rounds 3, 4 and 6. BR-29 asked for an ariadne issue referenced from this issue's Log; I checked `/Users/xianxu/workspace/ariadne/workshop/issues/` (no artifact-capture issue) and this issue's Log (only `ariadne#195`, a different mechanism). Neither exists yet. This cannot converge here — the generator is `ariadne/cmd/sdlc`.

### 5. Test coverage notes

The suite is green with `-race` (24.0 s / 25.5 s) and I mutation-checked four things independently this round, confirming each applied and compiled before reading the result: both new pins (RED, one test each), the `Slug`-regression claim in I-1 (GREEN where the comment says it must be RED), and the `withStore` branch probe (GREEN, i.e. unreachable). All six Done-when boxes now name a symbol, and every one I traced has an assertion behind it — that is a real change from rounds 2–5, where three did not. The shape is sound: `decideCapture` and `wordFileName` table-tested with zero IO; per-path arity at the `Capturer` seam including the `-raw` row; one store-level arity assertion through the real wiring that provably sees writers *below* the seam; `openStore` on both branches; one `t.Setenv` end-to-end disk assertion; `Forget` in the shared conformance suite. The one structural gap left is not a missing test but a mislabelled one — `Suite/forget cannot escape the words directory` (I-1) discards the error it would need to distinguish anything, so it asserts less than its name and comment claim. Making it assert the error, or renaming it to what it actually covers, is the honest close.

### 6. Architectural notes

- **ARCH-DRY — pass on code, flag on prose (I-1).** Grep-verified: `decideCapture` has one caller; `c.st.AppendEvent`/`c.st.Upsert` in `capture.go` are the only store writes outside `store/`; all three `.Capture(` calls are inside `lookupAndRender`; `wordFileName` has both filename-deriving call sites. The duplication that remains is entirely in comments — "why the delete path is safe" has three homes and one still asserts a retracted claim, and "who mutates the store" has three absolute statements that omit `Forget`.
- **ARCH-PURE — pass.** Two real pure functions with IO-free, mutation-proven tests; a thin injected shell (`storeCapturer`); the boundary (`openStore`) behind a seam. No business logic leaked into `store/`.
- **ARCH-PURPOSE — pass.** Shadow-sweep on `decideCapture`: `storeCapturer` derives ✓, the raw branch derives and is pinned ✓, `openStore`'s `noCapture` read is a labelled second reader of the same *input* rather than a restatement of the *policy* ✓. On `wordFileName`: `Upsert` ✓, `Forget` ✓. The issue's stated purpose — every entry path records, and records once — is delivered on every path, not the easy subset, and the "asserted, not inherited" obligation was discharged by measured retraction rather than left as documentation.
- **ARCH-MOCK — pass, with one note for downstream.** `store.Mem` ships as production code behind the interface `YAML` implements, `storetest.Suite` runs both, the owned backend boots from any portable folder (`store.NewYAML(dir, warn)`, tests use `t.TempDir`) with no production configuration, and the audio CDN and player each have a fake at the same seam production uses. The note, pre-existing and outside this issue's scope: the macOS CoreServices dictionary has a fixture-backed fake (`testDict`) but **no live conformance check** — nothing detects the day CoreServices changes its output shape. Worth an issue before `#15` builds more on top of `ParseEntry`.
- **For `#5` (ordering by `Lookups`):** the number is now trustworthy for both properties it needs — arity (one lookup, one write, pinned through the real wiring on every producible path) and accumulation (pinned through the capture path as of this commit, verified RED). `#5` can order by it.

### 7. Plan revision recommendations

The `## Revisions` section exists and is substantive; four of its five deltas are accurate. Two residuals, appended to the same entry rather than a new one:

- **Append to Revisions §3 — the seam defaults.** §3 names `capture`, `deck` and `newStore` but states no default for any of them. Record: `withStore` installs `&memHistory{}` and `noopCapturer{}` when a caller supplies neither, and **returns early when `history` and `capture` are both non-nil, leaving `deck` nil** — a branch no test reaches (probe-verified). That is the half BR-4 asked for and BR-26 escalated, still open after two rounds.
- **Add a forward pointer at the top of Chunk 1.** Round 6 raised this in prose and declined to file it, and I am keeping it here rather than as a finding since BR-1/BR-2/BR-3 are disposed: line 115-116 still reads "consulted from **three call sites**" and line 24 still describes "a `Capturer` that `storeHistory` also uses", while Revisions §1/§2 correct both. Under AGENTS.md §1 leaving them is defensible, but a reader hits the superseded text ~130 lines before the correction. One line — "Chunk 1 predates implementation; `## Revisions` supersedes it where they disagree" — closes it.
- **The `### Integration points` table is accurate but incomplete.** Both rows verify against the code (`Capturer` and `storeCapturer`, both new, both at `cmd/define/capture.go`), and the `Capture(word, found, opt)` signature bullet now matches. It omits `noopCapturer`, `storeDeps`, `withStore`, `openStore`, `wordFileName`, `forgetWord`, `isSet`, and `Store.Forget`/`Mem.Forget`/`YAML.Forget` — recommended at rounds 4 and 5 and not done. It also carries no PURE/INTEGRATION column, which is what the boundary-review protocol cross-checks; adding one would make the next issue's table checkable rather than descriptive.

```findings
dispose:
  - id: BR-27
    disposition: addressed
    note: |
      Both mutations re-run independently - skipping Upsert on a repeat and propagating a capture error each redden exactly one test; applied and compiled verified.
  - id: BR-4
    disposition: not-addressed
    note: |
      Revisions section 3 still names the three deps fields with no default stated for any of them.
  - id: BR-26
    disposition: not-addressed
    note: |
      Probe re-verified - a panic in withStore's early return leaves the entire suite green, so no test enters the branch; no comment states deck's default.
  - id: BR-28
    disposition: not-addressed
    note: |
      main.go:36 comment verbatim; capture_test.go:316 still sets newStore = openStore and five test files use t.TempDir().
  - id: BR-22
    disposition: not-addressed
    note: |
      Fifth occurrence (lines 18, 251, 485, 674, 878). Unfixable from tools - see BR-29.
  - id: BR-29
    disposition: not-addressed
    note: |
      No ariadne issue exists (checked ariadne/workshop/issues) and this issue's Log references only ariadne#195, a different mechanism.
findings:
  - id: new
    severity: Critical
    family: writes-to-cwd-unignored
    title: |
      A 9.6MB Mach-O arm64 binary is committed at cmd/define/define in the HEAD commit under review
    detail: |
      2nd in family (BR-11; prevalence 2). Do NOT just add a fourth anchored .gitignore line - but DO excise the
      blob before merge, because it is the one open item that becomes irreversible. Measured: git ls-files shows
      it tracked, git log --diff-filter=A dates it to 42cc96d, it is 9616546 bytes against a 99336-byte
      next-largest blob, and .git is currently 9.3M so it roughly doubles the repo for every clone forever. It
      is arm64-only, so a Linux or amd64 checkout gets an unrunnable file; and because it is tracked, cd
      cmd/define && go build now dirties git status on every developer build. The rule covering this and BR-11:
      an artifact the developer workflow drops into the working tree gets an ignore pattern covering EVERY
      directory it can be produced in, added when the tool learns to produce it. .gitignore has been patched
      three times by this rule's absence and each patch was anchored to the one place it had already happened
      (bin/, then /define root-only, then /words/ and /events/); go build in a main package writes the binary
      into that package's directory, which none of them match. Second half, since no ignore would have caught
      this one - it was git add-ed before any ignore existed: read git show --stat before committing, where
      "Bin 0 -> 9616546 bytes" is visible at a glance and does appear in this commit's own stat. Fix while the
      branch is unmerged: git rm --cached cmd/define/define, add an un-anchored pattern, git commit --amend.
  - id: new
    severity: Important
    family: prose-contradicts-code
    title: |
      The retracted "a regression in Slug must fail HERE" claim survives in storetest/suite.go and is false by measurement
    detail: |
      8th in family (BR-9, BR-10, BR-17, BR-18, README exit codes, BR-25, BR-28; prevalence 8). Do NOT patch this
      site. storetest/suite.go:140-142 says the traversal guarantee is "asserted here rather than inherited from
      Slug - a regression in Slug must fail HERE, loudly". That is the exact sentence 96adc20 deleted from
      YAML.Forget's doc for BR-25 and the exact claim the same commit rewrote the Done-when to RETRACT; wordFileName's
      own doc now says the opposite. Verified by mutation: regressing Slug entirely (no sanitising) reddens
      TestSlugDoesNotMergeHyphenAndSpace, TestSlugIsAlwaysOneSafePathElement and FuzzSlugIsSafe, and leaves
      Suite/forget cannot escape the words directory GREEN - it discards err, so its assertions pass whether the
      guard is present, absent or bypassed. Second site the same sweep finds: main.go:27, capture.go:50 and
      atlas/define.md:283 each state absolutely that storeCapturer is the only thing that writes to the store,
      while forgetWord's d.deck.Forget at main.go:396 deletes a word file - deps.deck's own comment three lines
      below main.go:27 acknowledges the second mutator. The rule is BR-28's, which is correct and was never
      executed as written: run the mechanical sweep over ALL 14 changed .go/.md files, not the files a finding
      named - storetest/suite.go is in git diff --stat and grepping Slug from that comment lands on the
      contradiction directly.
  - id: new
    severity: Minor
    family: needless-indirection
    title: |
      History.Add's found parameter is now dead in both implementations, and orElse is a generic for two nil checks
    detail: |
      4th in family (BR-14, BR-15, BR-21; prevalence 4). Do NOT patch these instances. History.Add(line string,
      found bool) at history.go:16 is ignored by BOTH implementations - history.go:28 and history_store.go:50 both
      take _ bool - while the sole caller computes code == 0 to supply it (replraw.go:171). It existed only
      because storeHistory.Add decided event-vs-deck; #4 moved that decision and left the parameter, which is
      BR-21's rule verbatim. It was missed because it lives in history.go, a file the diff never touched, so
      BR-28's file-based sweep cannot reach it - the extension the rule needs is: sweep the changed files AND the
      interface plus callers of every symbol whose responsibilities the diff moved. Weaker second instance,
      reasonably withdrawn: orElse[T comparable] at main.go:83 is a generic helper introduced by the very commit
      that closed this family, replacing two nil checks with six lines.
  - id: new
    severity: Minor
    family: eager-dependency-construction
    title: |
      run builds the store-backed deps before the -forget dispatch, so --forget reads an event log it never uses
    detail: |
      main.go:212 calls d.withStore before the -forget dispatch at :214, so --forget constructs storeHistory and
      loads the whole event log for a result it never consults. Reproduced against a seeded directory: a -forget
      run printed "define: 2026-08-21.yaml: recovered 0 event(s), dropped 1 torn record(s)" for a log it does not
      touch. Stated in round 6's prose and never given an id, which is the mechanism BR-28 named; recording it so
      it can be disposed rather than restated. The rule: a mode dispatch decides which dependencies are needed,
      so it runs before they are constructed.
  - id: new
    severity: Minor
    family: misleading-error-text
    title: |
      forgetWord prints a bare "define: %v" where every neighbouring message names the operand
    detail: |
      2nd in family (BR-13; prevalence 2). Do NOT patch this instance alone. main.go:398 formats a store error as
      "define: %v" while its neighbours use "define: %s: %v" with the word. The rule covering this and BR-13: a
      user-facing error names both the condition and the operand it failed on - BR-13's message named the wrong
      condition, this one names no operand.
```

---

## Re-review — 2026-08-21T12:27:06-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 4 — capture looked-up words into the deck |
| repo | tools |
| issue file | workshop/issues/000004-vocab-capture.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4afa1c537819ad662ad149a14bd757ac149772e5..82e3684c8a3c0185c866ea1d8949a6b8bf9a454a |
| command | sdlc close --issue 4 |
| reviewer | claude |
| timestamp | 2026-08-21T12:27:06-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Here is the review.

```verdict
verdict: REWORK
confidence: high
```

Eight rounds in, the shipped code is correct and I verified it rather than trusted it: `go vet`, the full suite, and a live end-to-end run all agree with the documented contracts (failed lookup → one `found: false` event, no `words/`, exit 1; `DEFINE_NO_CAPTURE=1` → nothing written; `--forget` absent → 1; `--forget a b` → 2). This round's headline claim also holds up under the discipline its own lesson demands — I verified `TestNoCommittedBinaries` by planting what it hunts in a **new** `cmd/` directory: clean PASS, planted FAIL (`compiled binary is tracked: cmd/newtool/newtool`), removed PASS, with `-count=1` so the result was not the cached one my first attempt read. What blocks the boundary is that BR-30's fix removed the file and not the cost. `b3ec4bd` ran `git rm` in a **follow-up commit** where the finding specified `git commit --amend`; `42cc96d` still adds the blob, so it remains reachable from HEAD. Measured by actually cloning: **`main` clones to 604 K of `.git`; this branch clones to 5.9 M, and blob `d12d8e7` (9,616,546 bytes) is present in the clone.** The repo merges with merge commits (all four of `main`'s merges are `Merge pull request`), so merging carries `42cc96d` into `main` permanently. The branch is 8 commits ahead and unpushed — this is the last cheap moment. Everything else below is non-blocking.

### 1. Strengths

- **The guard test is real, and it covers the class rather than the path.** I planted a binary at `cmd/newtool/newtool` — a directory that does not exist and that no `.gitignore` line matches — and `repo_guard_test.go:59` caught it. The magic-byte approach (`repo_guard_test.go:38-43`) is the right property; I confirmed the false-positive narrative too, since the repo really does track 28 symlinks.
- **`git rev-parse --show-toplevel` + `git -C`** (`repo_guard_test.go:27-32`) genuinely fixes the package-directory problem, and `.gitignore:28` demonstrably works — a fresh `go build` inside `cmd/define` leaves `git status` clean (`git check-ignore -v` confirms the rule that matched).
- **BR-31's titled site was swept, not patched.** `storetest/suite.go:140-145` now says what the test actually is — "an end-to-end net, NOT the assertion of the traversal guard… (measured)" — which is the honest form, consistent with `wordFileName`'s own doc and with the retracted Done-when.
- **The two new lessons are correctly drawn and transferable.** "`go test` runs in the package directory" and "test the property, not a proxy for it" are both earned here and both stated as rules rather than patch notes.
- **Every Done-when box now names a symbol**, and the ones I traced have assertions behind them. `issue:38-73` reads as an audit rather than a set of ticks — including a box that records its own negative result.

### 2. Critical findings

**BR-30 (not-addressed) — the artifact was removed from the working tree; the 9.2 MB blob is still in the branch's history.**

The commit message says "the artifact is removed and `.gitignore` covers the path" — both true of `HEAD`'s tree, neither true of the repo. Measured:

```
git rev-list --objects HEAD | grep cmd/define/define
  → d12d8e7fb992e427e87cf1651496291afc3d4b98 cmd/define/define
git log --oneline --all -- cmd/define/define
  → b3ec4bd (Bin 9616546 -> 0)      # deletion, a NEW commit
  → 42cc96d (Bin 0 -> 9616546)      # still adds it

clone --single-branch main                   → 604K  .git
clone --single-branch 000004-vocab-capture   → 5.9M  .git   ← blob present in the clone
```

BR-30 named the mechanism precisely — "git rm --cached …, add an un-anchored pattern, **git commit --amend**". The amend is the load-bearing verb; a `git rm` in a later commit changes what `HEAD` looks like and nothing about what is fetched. The finding's stated harm ("roughly doubles the repo for every clone forever") is unchanged, and this repo's merge convention preserves the branch's commits, so it becomes `main`'s cost on merge.

Two further residuals from the same finding's body, recorded so they can be disposed rather than restated:

- The pattern shipped as `/cmd/*/define` — **anchored and name-specific**, where the finding asked for un-anchored. `go build` in a future `cmd/words` writes `cmd/words/words`, which no line matches. That is defensible *given* the guard test (which I proved catches it), but only once the file is staged — the ignore is what keeps `git status` clean before that.
- BR-30's second half — "read `git show --stat` before committing, where `Bin 0 -> 9616546 bytes` is visible at a glance" — was **not recorded**. `grep` over `workshop/lessons.md` returns nothing for it, while the same commit recorded two lessons about how the guard was *built*. The lesson that would have prevented the artifact is the one missing.

*Fix, while the branch is unmerged:* rebase `42cc96d` to drop the blob (`git rebase -i` is unavailable here, but `git filter-branch`/`git-filter-repo` over `main..HEAD`, or reordering so the add never lands, all work), then `git gc --prune=now` and re-clone to confirm `.git` returns to the ~700 K range. Verify the same way I did — clone the branch and grep `git rev-list --objects HEAD`; do not read `git status`, which is green today and tells you nothing about this.

### 3. Important findings

**BR-31 (not-addressed) — the titled site was swept; the site enumerated in the body was not.**

BR-31's body named two sites. Site 1 (`storetest/suite.go`) is genuinely fixed. Site 2 is verbatim:

```
cmd/define/capture.go:51   // storeCapturer is the only thing in the process that writes to the store.
cmd/define/main.go:28      // capture is the only thing in the process that writes to the store.
atlas/define.md:281        **`storeCapturer` is the only writer in the process.**
cmd/define/main.go:396         removed, err := d.deck.Forget(word)
```

`Forget` deletes a word file — a store mutation performed outside the capturer. The contradiction is **two lines apart** in `main.go`: line 28 says "the only thing… that writes," and line 30 introduces `deck` as "the store `--forget` acts on." A reader of `#5` or `#8` auditing "what can change this deck?" is told three times the answer is one thing.

BR-31 also stated the rule and why it had never been run: *"run the mechanical sweep over ALL 14 changed `.go`/`.md` files, not the files a finding named."* The sweep was run on the file BR-31's title named and not on the list. I ran it, and it turns up two further instances beyond site 2, which I am recording under one id below rather than leaving as prose — because BR-28 measured that un-id'd prose items get 0 % addressed while id'd findings get 100 %.

**NEW [Important] `family-rule-applied-selectively` — this is the 3rd finding in this family (BR-23, BR-24; prevalence 3).** Per the escalation rule I am not asking for these instances to be patched.

Measured across this round, 10 open findings entering it:

| finding | body enumerated | closed |
|---|---|---|
| BR-30 | excise the blob (amend), un-anchored pattern, record the `--stat` rule | 0 of 3 — the *file* was untracked, which is none of the three |
| BR-31 | `storetest/suite.go`; the three "only writer" absolutes | 1 of 2 — the one in the title |
| BR-4, BR-22, BR-26, BR-28, BR-29, BR-32, BR-33, BR-34 | — | 0 of 8, untouched |

**Zero of ten closed.** BR-23 measured 3 of 10 and named the substitution; BR-24 added the scope clause for artifacts; both were disposed `addressed`. The rule they state is right and this round did not execute it — and the shape is now sharper than "fix the title": both partial fixes stopped at the *most legible* half of the finding (the tracked file you can see in `git status`; the comment whose file the title named), leaving the half that requires re-deriving the finding's measurement (clone size, the enumerated grep list). *The clause the rule was missing:* **a finding is closed only when you have re-run the measurement that produced it.** BR-30 came with a clone-size number and BR-31 with a grep list; re-running either takes one command and would have shown the work incomplete before the commit claimed it done.

### 4. Minor findings

- **BR-4 / BR-26 (both not-addressed)** — unchanged, and I re-verified BR-26 by probe: replacing `withStore`'s early return (`main.go:68`) with `panic("BR26_BRANCH_ENTERED")` — mutation confirmed applied, `BUILD_OK` — leaves the **entire suite green**, so no test enters the branch. Revisions §3 still names `capture`, `deck` and `newStore` with no default stated for any of them.
- **BR-28 (not-addressed)** — `main.go:36` verbatim: "Tests leave it nil and get in-memory defaults, so no test ever touches the real filesystem." `capture_test.go:316` sets `rig.deps.newStore = openStore`; **five** test files use `t.TempDir()`.
- **NEW `unpinned-invariant` — this is the 6th finding in this family (BR-5, BR-6, BR-7, BR-19, BR-27; prevalence 6).** Do NOT patch this instance. `.gitignore:25-28` claims "the general case is enforced by `TestNoCommittedBinaries`". It is not: the test reads `git ls-files` (`repo_guard_test.go:32`), which is the **index**, while the cost of this class lives in **history**. Verified on the clone above — the 9.6 MB blob is present and `git ls-files | grep -c cmd/define/define` returns 0, so the guard is green on a repo carrying exactly the artifact it exists to prevent. Secondary: both `git` calls fall back to `t.Skipf` (`:29`, `:34`), so the guard is a silent no-op wherever `git` is absent or the tree is exported. *The rule this family needs, and the clause it was missing:* the family's five prior instances were all "a claim is ticked and the named test does not redden"; this one is "a claim is enforced by a test whose **scope** is narrower than the claim." Before naming a test as the enforcement for a class, state what the test reads and check the class lives there.
- **NEW `prose-contradicts-code` — this is the 9th finding in this family (prevalence 9).** Do NOT patch these; BR-31's rule covers them and was not executed. Two instances the mechanical sweep over the changed-file list turns up, beyond BR-31's site 2: `workshop/issues/000004-vocab-capture.md:24` — the Spec's normative bullet "**Capture only on a successful lookup**" is contradicted by `decideCapture`'s `captureEventOnly` branch, by `capture_test.go:24`'s truth table, and by the README and atlas, all of which say a failed lookup *is* captured as history. And `main.go:173-176` — the `--help` text this window added says "define records **every** lookup under `words/` and `events/`", which is false for `-raw` (records nothing) and half-false for a failed lookup (event only, never `words/`); the README two files over states both exceptions correctly.
- **BR-32 (not-addressed)** — `History.Add(line string, found bool)` (`history.go:16`) is still ignored by both implementations (`history.go:28`, `history_store.go:50`), while `replraw.go:171` computes `code == 0` to supply it. The interface's own doc at `history.go:11` still says "Every SUBMITTED line is recorded, **with whether the lookup found anything**", which is now false in both implementations — the same finding from the prose side.
- **BR-33 (not-addressed)** — `main.go:212` still calls `d.withStore` before the `-forget` dispatch at `:214`.
- **BR-34 (not-addressed)** — `main.go:398` still formats a store error as `define: %v` where `:402` and `:405` name the word.
- **BR-22 / BR-29 (both not-addressed)** — the preamble is now at `close-review.md:18, 251, 485, 674, 878, 1043` — a sixth occurrence, one per round, exactly as predicted at rounds 3, 4, 6 and 7. I re-checked `/Users/xianxu/workspace/ariadne/workshop/issues/` (26 issues, none about artifact capture) and this issue's Log (`ariadne#195` only, a different mechanism). Neither exists yet; this cannot converge here.

### 5. Test coverage notes

The suite is green (`go vet` clean; `go test ./cmd/...` 24.3 s) and the new guard is the strongest-verified thing in this round — I ran the plant-and-remove cycle the lesson prescribes, and it behaved correctly in all three states once I re-ran with `-count=1`. Worth flagging as a caution for the next round: my first attempt at that verification printed `ok (cached)` for the planted run, which reads exactly like a blind test and is neither — the same class of false reading the round-4 Log records for `sed`-based mutations, arriving through Go's test cache instead. **`-count=1` belongs in the mutation recipe alongside "confirm it applied and compiled."** The rest of the coverage shape is unchanged and sound: `decideCapture` and `wordFileName` table-tested with zero IO; per-path arity at the `Capturer` seam including the `-raw` row; one store-level arity assertion through real wiring that provably sees writers below the seam; `openStore` on both branches; a `t.Setenv` end-to-end disk assertion; `Forget` in the shared conformance suite. The one gap that is a *scope* problem rather than a missing test is the guard's index-vs-history blindness, in the Minor above.

### 6. Architectural notes

- **ARCH-DRY — pass on code, flag on prose (BR-31, and the 9th prose instance).** Grep-verified: `decideCapture` has one caller; `c.st.AppendEvent`/`c.st.Upsert` in `capture.go` are the only append/upsert paths outside `store/`; all three `.Capture(` calls are inside `lookupAndRender`; `wordFileName` serves both filename-deriving call sites. The duplication that remains is entirely comments — "who mutates the store" has three absolute statements that omit `Forget`, and "what capture means" has a Spec bullet and a `--help` sentence that contradict `decideCapture`.
- **ARCH-PURE — pass.** `decideCapture` (`capture.go:26`) and `wordFileName` (`yaml.go:180`) are genuine pure functions with IO-free tests; `storeCapturer` is a thin shell over the policy; `openStore` is the boundary and is injectable through `deps.newStore`, which is what makes the env-wiring test possible without touching the developer's filesystem. Nothing leaked into `store/`.
- **ARCH-PURPOSE — pass on the issue, flag on this round's fix.** Shadow-sweep on `decideCapture`: `storeCapturer` derives ✓, the raw branch derives and is pinned ✓, `openStore`'s `noCapture` read is a labelled second reader of the same *input* ✓. On `wordFileName`: `Upsert` ✓, `Forget` ✓. The issue's stated purpose — every entry path records, and records once — is delivered on every path, not the easy subset. The flag is BR-30: the purpose of that finding was "the repo must not carry a 9.6 MB blob," and what shipped is the cheap, visible subset of it ("`git status` is clean"). Removing the file is not the point; removing the cost is.
- **ARCH-MOCK — pass, with one note.** `store.Mem` ships as production code behind the interface `YAML` implements, `storetest.Suite` runs both, `Forget` joined the suite in the commit that introduced it, and the owned backend boots from any portable folder (`store.NewYAML(dir, warn)`; tests use `t.TempDir`) with no production configuration. `repo_guard_test.go` shells out to `git` with no seam and no fake — correct here, since a faked `git` would defeat a test whose whole subject is the real index; the cost is the silent-skip failure mode noted above. The pre-existing gap, still outside this issue's scope: the macOS CoreServices dictionary has a fixture-backed fake (`testDict`) but **no live conformance check**, so nothing detects the day its output shape changes. Worth an issue before `#15` builds more on `ParseEntry`.
- **For `#5` (ordering by `Lookups`):** the number is trustworthy for both properties it needs — arity, pinned through the real wiring on every producible path, and accumulation, pinned through the capture path. `#5` can order by it.

### 7. Plan revision recommendations

The `## Revisions` section is substantive and four of its five deltas are accurate. Two residuals, appended to the same entry rather than a new one — both unchanged from round 7:

- **Append to Revisions §3 — the seam defaults.** §3 names `capture`, `deck` and `newStore` and states no default for any. Record: `withStore` installs `&memHistory{}` and `noopCapturer{}` when a caller supplies neither, and **returns early when `history` and `capture` are both non-nil, leaving `deck` nil** — a branch no test reaches (probe-verified green under `panic`). That is the half BR-4 asked for and BR-26 escalated, open three rounds.
- **A forward pointer at the top of Chunk 1.** Line 115-116 still reads "consulted from **three call sites**" and line 24 still describes "a `Capturer` that `storeHistory` also uses", while Revisions §1/§2 correct both ~130 lines later. One line — "Chunk 1 predates implementation; `## Revisions` supersedes it where they disagree" — closes it.
- **The `### Integration points` table** still omits `noopCapturer`, `storeDeps`, `withStore`, `openStore`, `wordFileName`, `forgetWord`, `isSet`, and `Store.Forget`/`Mem.Forget`/`YAML.Forget`, and carries no PURE/INTEGRATION column — which is what this protocol cross-checks. Recommended at rounds 4, 5 and 7.

```findings
dispose:
  - id: BR-30
    disposition: not-addressed
    note: |
      File untracked at HEAD, but b3ec4bd used a follow-up commit where the finding specified --amend; 42cc96d still adds the blob. Clone measured: main 604K, this branch 5.9M, blob d12d8e7 present.
  - id: BR-31
    disposition: not-addressed
    note: |
      Site 1 (storetest/suite.go) swept correctly; site 2 (capture.go:51, main.go:28, atlas/define.md:281 vs d.deck.Forget at main.go:396) untouched.
  - id: BR-4
    disposition: not-addressed
    note: |
      Revisions section 3 still names the three deps fields with no default stated for any of them.
  - id: BR-26
    disposition: not-addressed
    note: |
      Probe re-verified - panic in withStore's early return, mutation applied and BUILD_OK, entire suite still green.
  - id: BR-28
    disposition: not-addressed
    note: |
      main.go:36 verbatim; capture_test.go:316 still sets newStore = openStore and five test files use t.TempDir().
  - id: BR-32
    disposition: not-addressed
    note: |
      History.Add's found still ignored by both implementations; history.go:11 doc still claims it is recorded.
  - id: BR-33
    disposition: not-addressed
    note: |
      main.go:212 still calls withStore before the -forget dispatch at :214.
  - id: BR-34
    disposition: not-addressed
    note: |
      main.go:398 still prints a bare "define: %v" while :402 and :405 name the word.
  - id: BR-22
    disposition: not-addressed
    note: |
      Sixth occurrence (lines 18, 251, 485, 674, 878, 1043). Unfixable from tools - see BR-29.
  - id: BR-29
    disposition: not-addressed
    note: |
      Re-checked ariadne/workshop/issues (26 issues, none on artifact capture); this issue's Log still references only ariadne#195.
findings:
  - id: new
    severity: Important
    family: family-rule-applied-selectively
    title: |
      Both partial fixes closed the legible half of the finding and left the half that required re-running its measurement
    detail: |
      3rd in family (BR-23, BR-24; prevalence 3). Do NOT patch these instances. Measured: 10 open findings
      entered this round, 0 closed. BR-30 enumerated three moves (excise the blob via amend, an un-anchored
      pattern, record the git show --stat rule) and closed none of them - the file was untracked, which is
      none of the three. BR-31 enumerated two sites and closed the one in its title. The other eight were
      untouched. BR-23 measured 3 of 10 and BR-24 added the artifact-scope clause; both were disposed
      addressed, so the rule is written and was not executed. The clause it is still missing: a finding is
      closed only when you have RE-RUN the measurement that produced it. BR-30 shipped with a clone-size
      number and BR-31 with a grep list; re-running either is one command and would have shown the work
      incomplete before the commit claimed it done.
  - id: new
    severity: Minor
    family: unpinned-invariant
    title: |
      .gitignore names TestNoCommittedBinaries as enforcing the general case, but the test reads the index and the cost lives in history
    detail: |
      6th in family (BR-5, BR-6, BR-7, BR-19, BR-27; prevalence 6). Do NOT patch this instance. .gitignore:25-28
      claims "the general case is enforced by TestNoCommittedBinaries"; the test reads git ls-files
      (repo_guard_test.go:32), which is the index. Verified on a real clone of this branch: blob d12d8e7
      (9616546 bytes) is present, git ls-files reports it 0 times, and the guard is GREEN - certifying "no
      committed binaries" on a repo that carries one. Secondary: both git calls fall back to t.Skipf (:29, :34),
      so the guard is a silent no-op wherever git is absent or the tree is exported. The family's five prior
      instances were all "a claim is ticked and the named test does not redden"; this one is "a claim is
      enforced by a test whose SCOPE is narrower than the claim". The clause the rule needs: before naming a
      test as the enforcement for a class, state what the test reads and check the class lives there.
  - id: new
    severity: Minor
    family: prose-contradicts-code
    title: |
      The Spec's "capture only on a successful lookup" and --help's "records every lookup" both contradict decideCapture
    detail: |
      9th in family (prevalence 9). Do NOT patch these sites - BR-31's rule covers them and was not executed
      as written. Two instances the mechanical sweep over the full changed-file list turns up, beyond BR-31's
      own site 2. workshop/issues/000004-vocab-capture.md:24 - the Spec's normative bullet "Capture only on a
      successful lookup" is contradicted by decideCapture's captureEventOnly branch, by capture_test.go:24's
      truth table, and by the README and atlas, both of which state that a failed lookup IS captured as
      history. cmd/define/main.go:173-176 - the --help text this window added says define "records every
      lookup under words/ and events/", false for -raw (records nothing) and half-false for a failed lookup
      (event only, never words/); the README states both exceptions correctly two files over. Recording these
      with ids rather than as prose, because BR-28 measured that un-id'd prose items get 0 percent addressed.
```
