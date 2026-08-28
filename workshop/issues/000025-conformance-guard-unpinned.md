---
id: 000025
status: working
deps: []
github_issue:
created: 2026-08-27
updated: 2026-08-27
estimate_hours:
started: 2026-08-27T20:36:42-07:00
---

# conformance guard: the strict inversion was unpinned, and it broke a test that had pinned it

## Problem

#24 built `internal/conformance`: an absent external dependency SKIPS by default
and FAILS under `CONFORMANCE_STRICT`, so "green" cannot silently mean "did not
run". The behaviour was verified by running whole suites and reading their
output — and the package itself shipped with **no test of the inversion**.

It broke something immediately, and #24 did not see it:
`internal/llm/llmtest.SkipIfUnreachable` was routed through the guard, and its
own unit test `TestSkipsOnlyWhenNothingIsListening` asserts that an unreachable
service *skips*. Under strict it now fails instead, so that test **passed under
`go test` and failed under `CONFORMANCE_STRICT=1`, against unchanged code**.

#24 never caught it because its verification ran the conformance suite in a
sandbox, where the llm suites are absent-dependency skips either way. Running it
unsandboxed is what surfaced it.

The deeper fault is the same one in both places: **a test whose result depends on
ambient environment it does not control**, and **a rule with two modes pinned at
neither.**

## Spec

1. `TestSkipsOnlyWhenNothingIsListening` neutralizes `CONFORMANCE_STRICT` with
   `t.Setenv`, so it asserts the helper's reachability contract rather than
   whatever the environment happens to be.
2. The strict direction gets its own test at that call site — the half nothing
   pinned.
3. `internal/conformance` gets unit tests for `SkipOrFail` and `Strict` in both
   directions, including the `nil` error path and the empty-value-is-off case.

Out of scope, tracked below: the MIRROR half of the rule — an absent dependency
written as an unconditional `Fatal` — is still enforced by reading sites, not by
a test. `TestEverySkipIsRoutedOrWaived` sees `t.Skip*` only. #24's plan Risks
records this; designing the enforcement is a separate piece of thinking, not a
drive-by.

## Done when

- [ ] `go test ./...` and `CONFORMANCE_STRICT=1 go test -tags conformance ./...`
      agree about every test that is not itself an absent dependency — no test
      changes verdict on the variable alone.
- [ ] `SkipOrFail` is pinned in both directions, mutation-verified.
- [ ] `Strict()` treats an empty value as off, pinned.

## Plan

Single boundary — no `Mx` tags.

- [ ] `t.Setenv(conformance.StrictEnv, "")` in `TestSkipsOnlyWhenNothingIsListening`.
- [ ] `TestStrictTurnsAnUnreachableServiceIntoAFailure` at the same call site.
- [ ] `internal/conformance/skiporfail_test.go`: both directions × err/nil,
      `Strict()`'s empty-value case, and the message naming the variable.
- [ ] Mutation table, each row reddening a named test.
- [ ] Verify unsandboxed in both env states.

## Log

### 2026-08-27
