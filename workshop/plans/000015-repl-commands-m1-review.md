# Boundary Review — tools#15 (milestone M1)

| field | value |
|-------|-------|
| issue | 15 — REPL command mode: /-prefixed commands with type-ahead, starting with /history |
| repo | tools |
| issue file | workshop/issues/000015-repl-commands.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | cbcd30c4ffaa849717d85a2924021b116e57f4d8^..e8fe4f082191b4903e248fe51447c24f56af77d0 |
| command | sdlc milestone-close --issue 15 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-21T16:11:01-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what it claims: `/` in the first column becomes a `cmdCommand` on `parseREPLLine` — the classifier **both** loops already route through — so `echo /help | define` and the raw editor cannot disagree about what a line means, and the plan gate's PQ-2 (dispatch buried in `submitLine`) is genuinely fixed rather than described. I verified that by mutation, not by reading the commit message: deleting the `cmdCommand` case from `replLines` reddens `TestLineLoopDispatchesCommands`, exactly as the atlas claims. `go vet`, `gofmt -l`, `go test -count=1` and `go test -race` are all clean. Nothing here is Critical. What holds it back from SHIP is that the milestone's *headline* Done-when — "type-ahead narrows" — is pinned by nothing in the production path: I reverted **all five** `completionsFor(e.WalkBase(), hist, commands)` call sites in `runEditor` to the old `hist.Prefix(e.WalkBase())` and the entire suite stayed green (verified, `-count=1`, build OK). The behaviour is correct — a probe test I wrote passes on HEAD and fails under that mutation — it is simply unasserted, which is the same shape as `#4`'s I-4 (`deps.history` reachable only through the nil fallback). Second, `TestRawEditorDispatchesCommands` cannot fail for the reason it states. Fix those two, tick the plan, and add the README lines and this ships.

## 1. Strengths

- **`parseREPLLine` as the single dispatch point** (`cmd/define/repl.go:39`) is the right call and is *pinned*, not just asserted — mutation-verified above. The comment at `repl.go:34-38` naming the failure mode it prevents is the kind of note that survives.
- **Both loop tests use a dictionary that fails the test if consulted** (`commandloop_test.go:12-17`). Asserting a *negative* interaction ("a command never reaches the dictionary") is much stronger than asserting the output looks right, and it is what makes `TestUnknownCommandSuggestsWithoutDefining` a real pin — removing dispatch from `runEditor` reddens it (verified).
- **`completionsFor` as one namespace switch** (`command.go:36-41`) genuinely required no change to `Apply`: `Apply(e, k, matches)` already took candidates from the caller, so command mode is a different match *source*. The "argument typed ⇒ completion shorter than the line ⇒ no suggestion" property really does fall out — traced by hand for `/history 7`, `/history `, `/hi 7`.
- **`commandCtx` narrower than `deps`** (`command.go:130-141`) makes "a command cannot reach the dictionary or the player" structural rather than conventional (ARCH-PURE).
- **The command table is data; `dispatchCommand` switches on outcome, never on which command** (`command.go:148-168`) — the "adding a command needs no dispatch change" Done-when is architecturally true, not just currently true.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — command type-ahead reaches the editor, but nothing pins that it does.** `cmd/define/replraw.go:79,92,124,131,153` (five `completionsFor` sites). Verified: replacing all five with `hist.Prefix(e.WalkBase())` leaves `go test ./cmd/define/ -count=1` fully green. `TestCompletionsFor` tests the pure switch; no test drives `runEditor` and asserts a *command* suggestion. Fix — this test passes on HEAD and fails under that mutation (both verified):

```go
func TestEditorLoopSuggestsFromCommands(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("/he"), rig.deps, opt, cooked, finish, &out, &errb)
	if !strings.Contains(out.String(), greyOn+"lp") {
		t.Errorf("no grey command suggestion in the final frame: %q", tailOf(out.String()))
	}
}
```

**I-2 — `TestRawEditorDispatchesCommands` cannot fail for its stated reason.** `cmd/define/commandloop_test.go:73-83`. It asserts `out` contains `"/help"`, but `runEditor` writes the committed line through `RenderLine(submitted, "", opt.color)` (`replraw.go:108`) *before* dispatching — so stdout contains `/help` whether or not the command ran. Verified: neutering the `dispatchCommand` call inside the `cooked` closure leaves this test **PASS**. Assert on `runHelp`'s output shape instead, e.g. `"  /help "` (two-space indent + `%-10s` padding), or count occurrences: `strings.Count(out.String(), "/help") >= 2`. This is an instance of the lesson this very diff adds to `workshop/lessons.md` ("Mutation-check a claimed behaviour against its OWN code path").

**I-3 — README not updated for the `/` surface shipped at M1.** `README.md` (no occurrence of `/help` or "command"); `--help` text at `cmd/define/main.go:171-186` likewise. The keybinding table still describes Tab only as "accept the grey suggestion", and the binary now ships a typed surface a reader can run. The plan defers this to M2 Task 8 Steps 2–3, but the atlas deliberately landed *in* M1 ("per AGENTS.md section 8"), so the two artifacts now disagree about when docs land. Two lines in the key table plus one sentence ("a `/` in the first column opens command mode; `/help` lists what there is") closes it.

**I-4 — the plan's M1 tasks are entirely unticked at the milestone boundary.** `workshop/plans/000015-repl-commands-plan.md:184-221` — all fifteen Task 1–3 step boxes are `- [ ]` while the issue's M1 bullet is `- [x]`. AGENTS.md §8 asks for per-milestone ticking, and the close gate's `plan-unchecked` guard reads these. Tick Tasks 1–3.

**I-5 — the `commandCtx` literal is constructed twice, and M2 will double its fields (ARCH-DRY).** `cmd/define/repl.go:150-152` and `cmd/define/replraw.go:117-119` carry the identical `commandCtx{stdout: …, stderr: …, width: terminalWidth(stdout)}`. Task 4/7 add `deck` and `clock`; wiring one site and not the other is precisely the two-loops-disagree failure this milestone's whole design exists to prevent, one level down. Extract `newCommandCtx(d deps, stdout, stderr io.Writer) commandCtx` now, while it is a three-line change.

## 4. Minor findings

- `commandCtx.width` (`command.go:139`) is written at both call sites and read by nothing at M1 — forward-looking for `renderHistory`; fine, but it is a field at zero read sites today.
- Case policy is inconsistent three ways: `dispatchCommand` uses `EqualFold`, `commandCompletions` lowercases only the *input* (an uppercase registry name would silently never complete), and `Suggestion` compares case-sensitively — so the `{"HIS", []string{"/history"}}` guarantee in `command_test.go:63` is unreachable through the editor's grey tail (Up-arrow does reach it).
- `dispatchCommand` re-derives "nothing was close" as `len(near) == len(cmds)` (`command.go:161`); `nearestCommands` knows which branch it took and discards it. With two commands both within edit distance 2 it prints the menu form instead of "did you mean".
- `sameSet` compares `args` in `repl_test.go:29` and `commandloop_test.go:43` — args are ordered, so a reversal passes. (`TestParseCommandLine` does pin order via `DeepEqual` for `--days 7`.)
- No guard that every `commands` row has a non-nil `run`; a row added without one panics at dispatch rather than failing a test. A three-line loop over `commands` in a test pins the "a row plus its run" contract.
- Orphaned comment: `command_test.go:60-62` ("an argument already typed means the command is settled") sits above the `"HIS"` case-insensitivity row and describes neither it nor any row in that table.
- Entry-mode asymmetry: `define /help` as a one-shot *argument* still goes to the dictionary (`main.go` `defineOnce`), while `echo /help | define` runs the command. Acceptable under the Spec's "line" framing; worth a doc line once `/history` exists, since `define /history` failing with "no dictionary entry" is a plausible user error.
- `cmd/define/main.go:4-12`: stray blank line after `"context"` and the `store` import interleaved into the stdlib group (inherited from the `#4` window; gofmt-clean but unusual for this repo).

## 5. Test coverage notes

- Verified green: full suite, `-race`, `go vet`, `gofmt -l` (empty).
- Verified pins by mutation: `cmdCommand` case removed from `replLines` → `TestLineLoopDispatchesCommands` reddens ✓; dispatch removed from `runEditor` → `TestUnknownCommandSuggestsWithoutDefining` reddens ✓ (so the raw path *is* covered, just not by the test named for it — I-2).
- Verified **unpinned**: the `completionsFor` wiring (I-1).
- Untested but correct (probed): a piped unknown command exits 1 via `anyFailed` — `replLines(…, "/qqqqqq\n", …, pipedInput=true)` returns 1 with `define: unknown command /qqqqqq. Commands: /help`. Worth one assertion alongside `TestLineLoopDispatchesCommands`, since it is script-visible behaviour.
- `TestDispatchCommand` correctly uses a fixture registry rather than the live one, so adding `/history` in M2 will not break unrelated tests. Good call.

## 6. Architectural notes

- **ARCH-DRY — pass with one flag.** `parseREPLLine` (one decision table, mutation-verified) and `completionsFor` (one namespace switch, four call sites collapsed) are both correct applications. Flagged: the duplicated `commandCtx` literal (I-5).
- **ARCH-PURE — pass.** `parseCommandLine`, `commandCompletions`, `nearestCommands`, `editDistance`, `completionsFor` are pure and table-tested with no store, no clock, no IO; `dispatchCommand`/`runHelp` take writers and are the thin shell. No mock is needed to run any of them.
- **ARCH-PURPOSE — pass with the docs flag.** The purpose was "both loops get command mode", not the easy raw-editor subset, and the diff delivers both. Shadow-sweep of consumers of the "what does a line mean" source: `replLines` ✓ derives, `runEditor` ✓ derives, the one-shot path does not (Minor, out of Spec scope), README/`--help` remain hand-maintained restatements that no longer describe the surface (I-3).
- **ARCH-MOCK — pass.** No new external dependency at M1. The terminal stays doubled at one seam (scripted `<-chan Key` in-process, live pty behind `//go:build conformance`), and the honest note in `pty_conformance_test.go:12-25` about what that suite does *not* pin is a model of how to record a fake's limits.
- For M2: `commandCtx` is the surface Tasks 4–7 consume. Settling I-5 and the `width`/`clock` fields now means `runHistory` lands as a row plus a `run`, which is what the Done-when promises.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000015-repl-commands-plan.md`:

- **Atlas landed at M1, not M2.** Task 8 Step 1 (`atlas/define.md` gains `## Command mode`) was delivered in `ee605ee` per AGENTS.md §8's per-milestone rule. Record it, and either move Step 1 out of Task 8 or mark it done — otherwise M2 will re-derive a section that exists.
- **README/`--help` should follow the same rule.** Task 8 Steps 2–3 currently defer all user-facing docs to M2 while the atlas did not; either move the `/`-namespace half of those steps into M1's scope retroactively (I-3) or state explicitly that README lags the atlas by one milestone and why.
- **Task 2 Step 3's signature is stale.** The plan specifies `completionsFor(line string, hist History) []string`; the shipped function is `completionsFor(base string, hist History, cmds []command)`. The extra parameter is an improvement (it is what lets the tests use a fixture registry) — record it rather than leaving the plan naming a signature that does not exist.
- **Task 2's "four `hist.Prefix` call sites"** is now five in `runEditor` (the `cmdCommand` branch added one). Cosmetic, but the atlas repeats the number.

```findings
findings:
  - id: new
    severity: Important
    family: unpinned-production-wiring
    title: |
      Command type-ahead works in the editor but no test pins the wiring
    detail: |
      Verified by mutation: replacing all five completionsFor(e.WalkBase(), hist, commands)
      calls in runEditor (replraw.go:79,92,124,131,153) with hist.Prefix(e.WalkBase()) leaves
      the whole suite green with -count=1. TestCompletionsFor covers the pure switch only, so
      M1's headline Done-when "type-ahead narrows" is asserted nowhere in the production path.
      A runEditor test scripting "/he" and asserting greyOn+"lp" passes on HEAD and fails under
      the mutation (both verified).
  - id: new
    severity: Important
    family: assertion-cannot-distinguish-paths
    title: |
      TestRawEditorDispatchesCommands passes with dispatch removed
    detail: |
      commandloop_test.go:73-83 asserts stdout contains "/help", but runEditor echoes the
      committed line via RenderLine (replraw.go:108) before dispatching, so the assertion is
      satisfied by the echo. Verified: neutering the dispatchCommand call inside the cooked
      closure leaves this test PASS. Assert on runHelp's output shape ("  /help " with its
      %-10s padding) or on an occurrence count instead.
  - id: new
    severity: Important
    family: docs-consumer-not-updated
    title: |
      README and --help never mention the / command surface shipped at M1
    detail: |
      README.md contains no occurrence of "/help" or command mode, and the usage text at
      main.go:171-186 is unchanged, while atlas/define.md gained a full "## Command mode"
      section in the same commit. The binary now ships a surface a reader types; two lines in
      the key table plus one sentence closes it. The plan defers this to M2 Task 8 Steps 2-3,
      which is what makes the two artifacts disagree.
  - id: new
    severity: Important
    family: plan-artifact-lags-code
    title: |
      All fifteen M1 step boxes in the plan are unticked at the boundary
    detail: |
      workshop/plans/000015-repl-commands-plan.md:184-221 shows Tasks 1-3 entirely as "- [ ]"
      while the issue's M1 bullet is ticked and the code is shipped. AGENTS.md section 8 asks
      for per-milestone ticking and the close gate's plan-unchecked guard reads these boxes.
  - id: new
    severity: Important
    family: parallel-construction-drift
    title: |
      commandCtx is constructed twice, and M2 adds two more fields to it
    detail: |
      repl.go:150-152 and replraw.go:117-119 carry the identical commandCtx literal. Plan
      Tasks 4 and 7 add deck and clock; wiring one site and not the other is the same
      two-loops-disagree failure this milestone's design exists to prevent (ARCH-DRY).
      Extract newCommandCtx(d, stdout, stderr) while it is a three-line change.
  - id: new
    severity: Minor
    family: dead-field-at-boundary
    title: |
      commandCtx.width is written at both call sites and read by nothing
  - id: new
    severity: Minor
    family: case-policy-inconsistent
    title: |
      Completion lowercases only the input while dispatch uses EqualFold and Suggestion is case-sensitive
    detail: |
      commandCompletions (command.go:73) lowercases the prefix but not c.name, so an uppercase
      registry name would silently never complete; and because Suggestion compares
      case-sensitively, the {"HIS", ["/history"]} guarantee at command_test.go:63 is
      unreachable through the grey tail (Up-arrow does reach it).
  - id: new
    severity: Minor
    family: caller-rederives-callee-knowledge
    title: |
      dispatchCommand infers "nothing was close" from len(near) == len(cmds)
    detail: |
      command.go:161. nearestCommands knows which branch it took and discards it; with two
      commands both within edit distance 2 the caller prints the menu form instead of
      "did you mean".
  - id: new
    severity: Minor
    family: registry-row-unvalidated
    title: |
      Nothing checks that every commands row has a non-nil run
  - id: new
    severity: Minor
    family: set-compare-for-ordered-data
    title: |
      sameSet compares ordered args in repl_test.go:29 and commandloop_test.go:43
  - id: new
    severity: Minor
    family: comment-orphaned-by-insertion
    title: |
      command_test.go:60-62 comment describes no row in the table it heads
  - id: new
    severity: Minor
    family: entry-mode-inconsistency
    title: |
      define /help as a one-shot argument still goes to the dictionary
    detail: |
      Acceptable under the Spec's "line" framing, but worth a doc line once /history exists:
      define /history will fail with "no dictionary entry" while echo /history | define runs it.
  - id: new
    severity: Minor
    family: import-grouping
    title: |
      main.go:4-12 has a stray blank line and the store import inside the stdlib group
```

---

## Re-review — 2026-08-21T16:22:33-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 15 — REPL command mode: /-prefixed commands with type-ahead, starting with /history |
| repo | tools |
| issue file | workshop/issues/000015-repl-commands.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | cbcd30c4ffaa849717d85a2924021b116e57f4d8^..d9d26cec1fe320d67fd3b852147cb98508798ef8 |
| command | sdlc milestone-close --issue 15 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-21T16:22:33-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All five Important findings from round 2 are genuinely closed, and the two that mattered — BR-2 and BR-3 — I closed by re-running the measurement that produced them rather than by reading the commit message. Reverting the fix for BR-2 (all five `completionsFor(e.WalkBase(), hist, commands)` sites in `runEditor` → `hist.Prefix(e.WalkBase())`; mutation applied, `BUILD_OK`, `-count=1`) reddens `TestEditorSuggestsFromCommands`; neutering `dispatchCommand` inside the raw editor's `cooked` closure reddens `TestRawEditorDispatchesCommands`, and its failure output literally shows the echoed `/help` that made the old assertion pass — the fix and the diagnosis are both correct. I also independently verified the atlas's own claim (deleting the `cmdCommand` case from `replLines` reddens `TestLineLoopDispatchesCommands` — it does), so both loops are pinned, not just described. `go vet`, `gofmt -l`, `go test -count=1`, `go test -race`, and `go vet -tags conformance` are all clean. Nothing here is Critical and no Important remains open, so this does not block. What keeps it off SHIP is that nine Minors have now ridden through two rounds untouched, and probing production behaviour showed one of them (BR-9) is materially worse than its original description: with M1's single-command registry, `len(near) == len(cmds)` holds for *every* near-miss, so the "did you mean" branch is unreachable in production and `/hel` prints the menu form. Three of these Minors get harder in M2, not easier.

## 1. Strengths

- **The classifier really is single-source, and it is pinned.** `parseREPLLine` (`cmd/define/repl.go:33`) decides `/` first, and both `replLines` (`repl.go:140`) and `runEditor` (`replraw.go:100`) route through it. Mutation-verified in both directions. The PQ-2 defect — dispatch in `submitLine`, which only the raw editor reaches — is structurally impossible now, not merely avoided.
- **The BR-3 fix is the right fix, for the right stated reason.** Both loop tests now assert `"list the commands"`, a string only `runHelp` emits, instead of `"/help"`, which the editor's own echo produces. That is the correct repair for a circular oracle: change the observable so the two paths stop sharing it.
- **`TestEditorSuggestsFromCommands` (`commandloop_test.go:144`) is well-constructed** — it stocks history with `hibernate` so a regression that consulted history would produce a *visibly wrong* tail, and asserts both the positive (`greyOn+"p"`) and the negative (`"ibernate"` absent). Two-sided assertions like this are what made the mutation check unambiguous.
- **`refusingDict` (`commandloop_test.go:12`) asserts a negative interaction** — "the dictionary was never reached" — which is strictly stronger than "the output looks right", and it is why `TestUnknownCommandSuggestsWithoutDefining` also reddens under the dispatch mutation.
- **`newCommandCtx` (`command.go:148`) closes BR-6 in a compiler-enforced way.** M2 adding `deck`/`clock` will force a signature change that breaks both call sites at compile time — so the "wire one loop, forget the other" failure cannot recur silently.

## 2. Critical findings

None.

## 3. Important findings

None open. BR-2, BR-3, BR-4, BR-5, BR-6 verified closed (BR-2/BR-3 by reverting the fix; BR-4 by reading `README.md:31` and `main.go` usage text; BR-5 by counting fifteen `- [x]` boxes at `000015-repl-commands-plan.md:184-221`; BR-6 by grepping both call sites).

## 4. Minor findings

- **BR-9 is worse than its original measurement.** Probed against the live registry: `/hel` + Enter prints `define: unknown command /hel. Commands: /help`, not "did you mean". With one command, `len(near) == len(cmds)` is true for every near-miss, so the "did you mean" branch has **zero production reachability at M1** — it is exercised only by `dispatchCmds`, the fixture set. The atlas's own sentence ("`/his` + Return dispatches `his` and gets a suggestion") describes the branch that never runs.
- **BR-7 now has a second edge**: `commandCtx.width` is still read by nothing, and `newCommandCtx` re-derives it via `terminalWidth(stdout)` on every dispatch — a second source of truth for terminal width alongside `opt.width`, computed once in `run()`. They can disagree across a resize, and M2's `renderHistory` is the first consumer that will care which one it got.
- **BR-12 has a second instance**, created by this round: `command_test.go:93` still says "the four call sites in replraw.go" — the `cmdCommand` branch made it five. (The *atlas* claim "replaced four `hist.Prefix(...)` call sites" is accurate; there were exactly four before `8688137`.)
- BR-8, BR-10, BR-11, BR-13, BR-14, BR-1 — unchanged from round 2; re-verified present.
- **New:** the BR-4 fix was appended rather than integrated — `README.md:31` adds a second `define` row to a shell block whose every other line is a distinct invocation, and the editor key table (`README.md:44-51`), which is where a reader looks for "what can I type", gained no `/` row while its `Enter | define what you typed` row is now incomplete.
- **New:** `echo /histry | define` exits **1**, though `dispatchCommand` computes **2** and README documents 2 as the usage-error code. `replLines` collapses any non-zero to `anyFailed`. Defensible for a multi-line loop, but it is script-visible and asserted nowhere.

## 5. Test coverage notes

- Verified green: full suite, `-race`, `go vet`, `go vet -tags conformance`, `gofmt -l` (empty).
- Verified pins by mutation (applied + compiled + `-count=1` each time): `completionsFor` wiring in `runEditor` → `TestEditorSuggestsFromCommands` reddens ✓; dispatch removed from `runEditor` → `TestRawEditorDispatchesCommands` **and** `TestUnknownCommandSuggestsWithoutDefining` redden ✓; `cmdCommand` case removed from `replLines` → `TestLineLoopDispatchesCommands` reddens ✓.
- Probed correct but unasserted: Tab accepts a command completion (`/hel`+Tab+Enter runs help); `/`+Tab and `/`+Up both reach `/help`; `define /help` as a one-shot argument exits 1 with "no dictionary entry" (BR-13); piped unknown-command exit code (above).
- `TestDispatchCommand` correctly uses a fixture registry rather than the live one, so M2 adding `/history` will not break unrelated tests.

## 6. Architectural notes

- **ARCH-DRY — pass.** `parseREPLLine` as the one classifier and `completionsFor` as the one namespace switch are both mutation-verified live. BR-6 closed the duplicated `commandCtx` literal. The one residue is the second terminal-width derivation noted above.
- **ARCH-PURE — pass.** `parseCommandLine`, `commandCompletions`, `nearestCommands`, `editDistance`, `completionsFor` are pure and table-tested with `memHistory` (production code, not a mock); no store, clock, or IO is needed to run any of them. `dispatchCommand`/`runHelp` take writers and are the thin shell.
- **ARCH-PURPOSE — pass.** Shadow-sweep of every consumer of "what does a line mean": `replLines` derives ✓, `runEditor` derives ✓, `--help` and README now describe the surface ✓, `atlas/define.md` ✓. The one non-deriving consumer is the one-shot argument path (BR-13), which the Spec's "line" framing places out of scope. No hand-maintained restatement of the command table survives — `/help` lists `c.cmds`, the table it was dispatched from.
- **ARCH-MOCK — pass.** No new external dependency at M1. The terminal stays doubled at one seam (scripted `<-chan Key` in-process, live pty behind `//go:build darwin && conformance`), and `pty_conformance_test.go:12-25` states plainly what that suite does *not* pin.
- **For M2:** settle BR-7/BR-9 before `/history` lands. `renderHistory` is `width`'s first real consumer, and a second registry row is what makes BR-9's branch reachable — fixing both after the fact means changing behaviour a test has by then frozen.

## 7. Plan revision recommendations

- **Task 2 Step 3's signature is stale.** The plan specifies `completionsFor(line string, hist History) []string`; the shipped function is `completionsFor(base string, hist History, cmds []command) []string`. The extra parameter is what lets the tests use a fixture registry — record it rather than leaving the plan naming a function that does not exist.
- **Record that `atlas/define.md` landed at M1, not M2.** Task 8 Step 1 was delivered in `ee605ee`, and Steps 2–3 (README, `--help`) in `d9d26ce` — so Task 8 is three-quarters done while sitting entirely unticked under M2. Either tick those steps or move them into M1's scope retroactively; as written, M2 will re-derive documentation that exists.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan Tasks 1/3/5/6/7 still enumerate cases in prose; no fuzz target named for either parser.
  - id: BR-2
    disposition: addressed
    note: |
      Mutation-verified: all five completionsFor sites reverted to hist.Prefix, BUILD_OK, TestEditorSuggestsFromCommands FAILS.
  - id: BR-3
    disposition: addressed
    note: |
      Mutation-verified: dispatch neutered in runEditor, BUILD_OK, TestRawEditorDispatchesCommands FAILS on "list the commands".
  - id: BR-4
    disposition: addressed
    note: |
      README.md:31 and main.go usage both name the / surface; see the new fix-appended-not-integrated finding for the residue.
  - id: BR-5
    disposition: addressed
    note: |
      All fifteen M1 step boxes ticked at plan lines 184-221.
  - id: BR-6
    disposition: addressed
    note: |
      newCommandCtx at command.go:148, used at repl.go:155 and replraw.go:117; M2 field additions are now compiler-enforced.
  - id: BR-7
    disposition: not-addressed
    note: |
      Still unread, and newCommandCtx now re-derives width per dispatch — a second source beside opt.width.
  - id: BR-8
    disposition: not-addressed
    note: |
      Probed live: /HEL + Tab does not accept, then dispatches and fails; /HELP + Enter works. Unchanged.
  - id: BR-9
    disposition: not-addressed
    note: |
      Worse than measured: with one command, len(near)==len(cmds) always, so "did you mean" is unreachable in production at M1.
  - id: BR-10
    disposition: not-addressed
    note: |
      No nil-run check anywhere; a row without run panics at dispatch.
  - id: BR-11
    disposition: not-addressed
    note: |
      sameSet still compares ordered args at repl_test.go:30 and commandloop_test.go:47.
  - id: BR-12
    disposition: not-addressed
    note: |
      Prevalence now 2 — command_test.go:93 "the four call sites in replraw.go" was made wrong by the cmdCommand branch's fifth.
  - id: BR-13
    disposition: not-addressed
    note: |
      Probed: `define /help` exits 1 with "define: /help: no dictionary entry".
  - id: BR-14
    disposition: not-addressed
    note: |
      main.go:3-16 unchanged — blank line after "context", store import inside the stdlib group.
findings:
  - id: new
    severity: Minor
    family: fix-appended-not-integrated
    title: |
      The BR-4 README fix was appended to the sh block instead of integrated into the key table
    detail: |
      README.md:31 adds a second "define" row to a shell block whose every other line is a
      distinct invocation, so the block now lists the same command twice with a continuation
      comment. The editor key table at README.md:44-51 — the doc's actual structure for "what
      can I type" — gained no row for /, and its "Enter | define what you typed" row is now
      incomplete, since Enter also dispatches a command. The finding asked for two lines in
      the key table; the fix landed elsewhere.
  - id: new
    severity: Minor
    family: loop-collapses-callee-exit-code
    title: |
      A piped unknown command exits 1 where dispatchCommand computes 2 and README documents 2
    detail: |
      Probed: `echo /qqqqqq | define` exits 1. dispatchCommand returns 2 (repl.go:155 discards
      it into anyFailed), and README documents 2 as the usage-error code, so a script cannot
      tell "no such command" from "no dictionary entry". Defensible for a multi-line loop, but
      it is script-visible behaviour and no test asserts it in either direction.
```
