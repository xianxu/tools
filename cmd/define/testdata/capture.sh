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
#   bank     homograph number, no syllabification, only sense 1 reachable;
#            "(banked as adjective)" -- a POS inside a paren
#   record   part-of-speech glued to syllabification; per-block IPA; pipe overload
#   run/set  huge multi-block entries; bare numerals in examples ("the 200 meters")
#   quokka   minimal entry, no numbered senses
#   present  homograph AND syllabification together, in that order
#   iPhone   no pronunciation span at all, newline-separated
#   read     a parenthesised inflected-form pronunciation before the entry's own,
#            plus a head token equal to the headword
#   man      "[with adjective or noun modifier]" -- POS words inside a bracket
#   thing    "(things) [with adjective or noun modifier]" -- same, plus a paren
#   subject  "a noun phrase functioning as" -- a POS word in plain prose; and
#            "(subject to) adjective" -- a REAL opener after a paren, so the
#            three cases together pin the block-opener rule from both sides
#   use      the head swallows sense 1's number ("use verb 1 [with object]")
#   alewife  "(plural alewives | ˈālˌwīvz |)" -- a pronunciation inside a gloss
#
# The last three, plus content/even/desert/minute/use/subject/iPad/MacBook/Amazon,
# were all rendering with dropped or reordered content at the M1 boundary review.
words=(
    sycophantic quokka ephemeral defenestrate bank record run gaslighting set
    present iPhone read content even desert minute use subject iPad MacBook Amazon
    man thing alewife bases
)
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
