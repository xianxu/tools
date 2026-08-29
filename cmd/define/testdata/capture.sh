#!/usr/bin/env bash
# Capture real dictionary output for the parser fixture corpus.
#
# Per-language since #23: entries/<lang>/, and every capture goes through the
# SAME curated identifiers production selects. Capturing English through the NULL
# search instead would make the capture path and the production path agree only
# by coincidence -- they did, on this host, until the active set differed and
# TestFixturesMatchLiveDictionary went red (ARCH-MOCK: the fake must model what
# the seam actually does, not what a neighbouring seam does).
#
# MUST run OUTSIDE a sandbox -- DCSCopyTextDefinition needs real access to
# /System/Library/AssetsV2 and silently returns nothing without it. That silence
# is exactly why this script fails hard on a short capture: a directory of
# zero-byte fixtures would make TestRenderLosesNothing vacuously green.
#
# Re-run after a macOS upgrade; dict_conformance_test.go detects the drift.
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p entries/en entries/es

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
#   parrot   "(parrots, parroting | ˈperədiNG |)" -- pronunciations in an
#            inflection list, whose pipes collide with example separators
#   complete "…The Compleat Angler] verb [with object]" -- a part of speech
#   pulp     opening straight after an editorial note's closing bracket
#   bargainer PHRASES text carrying example-separator pipes
#
# NOTE: DCSCopyTextDefinition searches the host's ACTIVE dictionaries, not NOAD
# specifically (there is no public API to select one). iPhone, iPad and MacBook
# below are Apple Dictionary entries, not NOAD -- which is precisely why they
# have no pronunciation and exercise that branch.
#
# The last three, plus content/even/desert/minute/use/subject/iPad/MacBook/Amazon,
# were all rendering with dropped or reordered content at the M1 boundary review.
words=(
    sycophantic quokka ephemeral defenestrate bank record run gaslighting set
    present iPhone read content even desert minute use subject iPad MacBook Amazon
    man thing alewife bases parrot complete pulp bargainer
    # Multi-word headwords: their pronunciations have interior spaces
    # ("hot dog | ˈhät ˌdäɡ |"), which the single-token rule rejected outright --
    # and when an entry's OWN pronunciation is rejected the parser used to walk
    # past it and adopt a derivative's, collapsing the whole entry.
    "hot dog" "a priori"
    # concrete has NO pronunciation of its own but DERIVATIVES has one.
    concrete
    # mesa is here for #23 rather than for the parser: it is the homograph that
    # makes the language mode visible. Through the English dictionary it is "an
    # isolated flat-topped hill"; through the Larousse below it is furniture.
    # Captured in BOTH languages, and the pair is the assertion.
    mesa
)
# Spanish, captured through the Larousse Diccionario General specifically.
#
# The identifier, never a name match: DCSCopyAvailableDictionaries returns a SET
# whose iteration order is unspecified, so a substring like "Espa" matches
# Larousse on one run and the BILINGUAL Gran Diccionario Oxford on the next.
#
# Chosen for what they prove rather than for vocabulary:
#   mesa      the headline case -- an English homograph. Through NULL this is
#             "an isolated flat-topped hill"; through Larousse it is furniture.
#   bonito    same shape, and carries a feminine-form gloss
#   once      "11", not "on one occasion"
#   real      an adjective, not the English adverb sense
#   madrugar  a verb with a usage example, which is what monolingual buys
es_words=(mesa bonito once real madrugar)
ES_DICT=com.apple.dictionary.es.DGLEV
# English is two books, in the same preference order chooseDictionary uses: NOAD
# answers ordinary words, Apple Dictionary answers iPhone/iPad/MacBook.
EN_DICTS=(com.apple.dictionary.NOAD com.apple.dictionary.AppleDictionary)
# One exemplar per raw-notation cause, in knownRawByCause order: a prose numeral
# read as a sense number, a pronunciation glued to the headword, a phrase block's
# pronunciation run into prose, and an entry whose SUBJECT is the pipe character.
RAW_WORDS=(charge hundred shape pipe)

MIN_BYTES=40

capture() { # <word> <outdir> <dictionary-id>...
    local w="$1" dir="$2"; shift 2
    local out="$dir/$w.txt" id
    # The curated list in order, first hit wins -- the same walk
    # selectedDictionary.Lookup performs.
    : > "$out.tmp"
    for id in "$@"; do
        if python3 capture.py "$w" "$id" > "$out.tmp" 2>/dev/null; then
            break
        fi
    done
    if [ ! -s "$out.tmp" ]; then
        rm -f "$out.tmp"
        echo "capture failed: $w" >&2
        exit 1
    fi
    local n
    n=$(wc -c < "$out.tmp" | tr -d ' ')
    if [ "$n" -lt "$MIN_BYTES" ]; then
        rm -f "$out.tmp"
        echo "capture too short for '$w' ($n bytes) -- sandboxed, or the dictionary is absent." >&2
        exit 1
    fi
    mv "$out.tmp" "$out"
}

for w in "${words[@]}"; do
    capture "$w" entries/en "${EN_DICTS[@]}"
done

# The raw-notation exemplars (#26) — one real entry per cause the live ratchet
# classifies. A SEPARATE corpus, not entries/<lang>/, because these are the
# entries that still render unconverted notation: capturedLanguages walks
# entries/ expecting language names, and TestNoRawPronunciationNotationSurvives
# asserts a hard ZERO over it. Here the raw notation is the point.
#
# Captured through the same curated English books production selects, so the
# classifier is pinned against what the tool actually renders.
mkdir -p rawnotation
for w in "${RAW_WORDS[@]}"; do
    capture "$w" rawnotation "${EN_DICTS[@]}"
done
for w in "${es_words[@]}"; do
    capture "$w" entries/es "$ES_DICT"
done

echo "captured ${#words[@]} English, ${#es_words[@]} Spanish, ${#RAW_WORDS[@]} raw-notation entries:"
wc -c entries/en/*.txt entries/es/*.txt rawnotation/*.txt
