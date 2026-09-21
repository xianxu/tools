# Lessons

Compact rules distilled from the incident history. Keep durable prevention here;
leave incident transcripts and one-off details in their owning issue or plan.

## Proof-shaped testing

- Test the behavior at the production entry point and through the real dependency
  boundary. A helper, fake, or parser test does not prove the loop, CLI wiring,
  subprocess, network, or renderer that consumes it.
- A test must go red when the code it names is reverted. Assert the guard's own
  decision and the absence/presence of its effect; a later error or bare
  `err != nil` is not evidence.
- Mutation-check the wiring as well as the helper. A pure function can be fully
  covered while its caller still uses an inline twin or never reaches it.
- Derive enumerations from the production set. A table copied from the same
  list as the assertion cannot detect a missing site, branch, field, or artifact.
- A negative assertion needs a subject that can fail. Empty inputs, self-generated
  fixtures, and checks guarded by their own output are vacuous.
- Re-run the measurement that produced a finding after the fix. A check that
  never observed the old failure cannot distinguish a real guard from a comment.
- Fuzz both sides of an invariant and include adversarial alternatives. A fixture
  tuned to the happy path proves only the fixture.
- Verify the mutation applied and compiled. A no-op replacement, wrong flag, or
  filtered suite must fail loudly rather than report a plausible zero.

## Contracts and ownership

- Give each fact one producer and make consumers derive from it. Duplicate
  registries, prose lists, and constants standing in for measurements drift.
- A receipt, filename, display key, or path locates data; it does not prove
  identity, provenance, or persistence. Carry the canonical ID and failure state.
- A guard's scope is part of its contract. Enumerate all callers, packages,
  generated files, and build-tag variants before claiming a family is closed.
- If two error classes require opposite responses, preserve the distinction at
  the seam; do not collapse them into one fallback.
- A convenience exemption must name the exact case and retain every unrelated
  invariant. A carve-out with no exercised instance is not a rule.
- A comment, plan row, or review sentence is a claim. Cite symbols and verify it
  against the tree, not memory or the finding's example.

## Terminal, protocol, and runtime behavior

- In raw mode, render cooked output but play raw input; do not assume escape
  sequence length or event boundaries. Decode prefixes and unknown complete
  controls separately.
- Keep one decision table for related loops. If two implementations encode one
  idea, test the distinction and delete the wrong copy rather than letting the
  comment choose a winner.
- Measure widths as runes/glyphs consistently across renderers. Controls are not
  display cells; terminal rows are not frame lines.
- A marker must distinguish the state it names from the state it excludes. Test
  truncation with a round trip and test captures against real framing.
- A runtime override is a second door into the same field. Route both doors
  through one validation and ownership boundary.
- When removing a cost or behavior, mutation-test the old path and ensure the
  remaining test is not merely testing the replacement's absence.

## Fakes, seams, and integration

- A fake at a seam cannot reveal bugs below that seam. Run one live or stateful
  integration path for every external contract that matters.
- Fakes reproduce the wrapped dependency's documented returns, including
  cancellation, partial output, empty output, and shutdown races.
- A double must defend against the obvious alternative and must be constructed
  like production's object. Otherwise the test only proves the double.
- If a live check is opt-in, document and invoke it in the release evidence;
  an unrun conformance test is not coverage.
- Tests that enter a temp directory, sandbox, or package shell must hard-guard
  the path and run the same command entry point users run.
- A test that skips the only pin is green evidence of nothing. Prefer a clear
  failure or explicit unsupported result.

## Review and repository workflow

- Before ticking a plan row, delete the behavior it claims to cover and run the
  named suite. A family finding closes only when every enumerated instance closes.
- Review the boundary verdict before reporting it closed; the `Review-Verdict:`
  trailer belongs to the boundary commit, not a later fix.
- Keep mutation safety in Git (`git stash` or a committed baseline), not a
  scratch copy whose source may differ from the tested tree.
- A deletion sweeps every artifact that named the concept: code, tests, docs,
  helpers, generated output, and comments. Grep the rendered shape as well as
  the identifier.
- Run the suite named by the checklist, with `-count=1` where cache matters,
  and read the exit status. `go test` runs in the package directory; a pipeline
  reports the last command.
- Published-binary smokes must execute the installed/released artifact, not a
  convenient dev build. Record what was actually measured.
- A plan is a consumer of the tree. Keep entity tables, citations, revisions,
  and status rows synchronized with the committed symbols and evidence.

## Working rule

When a result surprises you, inspect the instrument before redesigning the
system. Ask: what exact production path ran, who owns its state, what alternative
would make this test green, and which mutation would make the claim fail?
