---
id: '000025'
status: done
started: 2026-08-27T20:36:42-07:00
created: 2026-08-27
updated: 2026-08-27
estimate_hours: 3.64
actual_hours: 4.32
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
