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
