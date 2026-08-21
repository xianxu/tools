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
