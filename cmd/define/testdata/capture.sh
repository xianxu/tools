#!/usr/bin/env bash
# Capture real NOAD output for the parser fixture corpus.
#
# MUST run OUTSIDE a sandbox -- DCSCopyTextDefinition needs real access to
# /System/Library/AssetsV2 and silently returns nothing without it. That silence
# is exactly why this script fails hard on a short capture: a directory of
# zero-byte fixtures would make TestRenderLosesNothing vacuously green.
#
# Re-run after a macOS upgrade; dict_conformance_test.go detects the drift.
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p entries

# The corpus is chosen for structural variety, not vocabulary:
#   bank    homograph number, no syllabification, only sense 1 reachable
#   record  part-of-speech glued to syllabification; per-block IPA; pipe overload
#   run/set huge multi-block entries
#   quokka  minimal entry, no numbered senses
words=(sycophantic quokka ephemeral defenestrate bank record run gaslighting set)
MIN_BYTES=40

for w in "${words[@]}"; do
    out="entries/$w.txt"
    if ! python3 capture.py "$w" > "$out.tmp"; then
        rm -f "$out.tmp"
        echo "capture failed: $w" >&2
        exit 1
    fi
    n=$(wc -c < "$out.tmp" | tr -d ' ')
    if [ "$n" -lt "$MIN_BYTES" ]; then
        rm -f "$out.tmp"
        echo "capture too short for '$w' ($n bytes) -- sandboxed, or NOAD is absent." >&2
        exit 1
    fi
    mv "$out.tmp" "$out"
done

echo "captured ${#words[@]} entries:"
wc -c entries/*.txt
