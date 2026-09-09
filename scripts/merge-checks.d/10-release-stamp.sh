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
# Scoped deliberately to `-run Ldflags`: the rest of the conformance suite makes
# live model and dictionary calls, which a merge gate must not.
#
# (The fleet-wide question of where CONFORMANCE_STRICT gets set is ariadne#37;
# this is the repo-local floor that does not wait on it.)
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

echo "==> release stamp: go test -tags conformance -run Ldflags ./cmd/define/"
go test -tags conformance -run Ldflags -count=1 ./cmd/define/
