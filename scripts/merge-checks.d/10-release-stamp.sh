#!/usr/bin/env bash
# Release-stamp conformance, run at the merge gate (#49 I-3).
#
# WHY THIS EXISTS AS A GATE AND NOT JUST AS A TEST. `define --version` is what a
# bug report names, and the number reaches the binary ONLY through the linker:
# `go build -ldflags "-X main.version=..."`, which Formula/define.rb passes. That
# path has a failure mode with no error anywhere — rename or move `main.version`
# and the linker writes nothing, `go build` exits 0, and the release ships saying
# "built from source" while every unit test stays green (the unit test assigns
# the Go variable, so it cannot see the linker).
#
# TestLdflagsStampReachesTheBinary closes that by shelling the real toolchain
# with the formula's own flag. But it is behind `//go:build conformance`, and the
# boundary review measured that nothing — not the Makefile, not parallel-checks,
# not this workflow — ever runs `-tags conformance`. A check that only protects
# whoever remembers the command is not a gate, so this runs it.
#
# Scoped deliberately to one anchored test name: the rest of the conformance
# suite makes live model and dictionary calls, which a merge gate must not.
#
# THE CHECK ASSERTS THAT IT RAN. `go test -run <pattern that matches nothing>`
# exits 0 and prints "ok … [no tests to run]", so a gate that trusts the exit
# code reports a green tick having executed nothing — rename or delete the test
# and this check would still pass (#49 II, measured). So the output is required
# to contain the test's own PASS line. CONFORMANCE_STRICT=1 for the same reason
# one level down: it turns a routed skip inside that path into a failure rather
# than a silent pass.
#
# It ignores the $BASE $HEAD range the runner passes and always runs. That is
# deliberate: ~3s, and "did this PR touch the version plumbing" is exactly the
# judgment that goes wrong — the ldflags path can break from a rename in a file
# a path-scoped filter would not have matched.
#
# (The fleet-wide question of where CONFORMANCE_STRICT gets set is ariadne#37;
# this is the repo-local floor that does not wait on it.)
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

TEST='^TestLdflagsStampReachesTheBinary$'
echo "==> release stamp: go test -tags conformance -run $TEST ./cmd/define/"

out="$(CONFORMANCE_STRICT=1 go test -tags conformance -run "$TEST" -v -count=1 ./cmd/define/ 2>&1)" || {
    echo "$out"
    exit 1
}
echo "$out"

if ! printf '%s\n' "$out" | grep -q -- '--- PASS: TestLdflagsStampReachesTheBinary'; then
    echo "✖ release stamp: TestLdflagsStampReachesTheBinary did not run." >&2
    echo "  'go test' exits 0 when -run matches nothing, so this gate requires the" >&2
    echo "  test's own PASS line as proof it executed. If it was renamed, update" >&2
    echo "  TEST above; if it was deleted, the release stamp has no gate at all." >&2
    exit 1
fi
