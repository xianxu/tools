package main

import (
	"strings"
	"testing"
)

func TestTrailingSegments(t *testing.T) {
	for _, tc := range []struct {
		name string
		line string
		want []segment
	}{
		{"empty line has no segments", "", nil},
		{"only spaces has no segments", "   ", nil},
		{"single word is its own whole-line segment", "obseq", []segment{{"", "obseq"}}},
		{"each word starts a segment, longest first", "what is the diff", []segment{
			{"", "what is the diff"},
			{"what ", "is the diff"},
			{"what is ", "the diff"},
			{"what is the ", "diff"},
		}},
		// A trailing space must not produce an empty segment: hist.Prefix("")
		// returns EVERY entry, so an empty segment would suggest a random line
		// after every space typed.
		{"trailing space starts no segment", "what is ", []segment{
			{"", "what is "},
			{"what ", "is "},
		}},
		{"a run of spaces is one boundary", "what  is", []segment{
			{"", "what  is"},
			{"what  ", "is"},
		}},
		{"leading space is part of the first head", " hot dog", []segment{
			{" ", "hot dog"},
			{" hot ", "dog"},
		}},
		// Heads are byte slices of the line; a multi-byte rune before a boundary
		// is where naive index arithmetic corrupts the text.
		{"multi-byte runes slice cleanly", "¿qué obseq", []segment{
			{"", "¿qué obseq"},
			{"¿qué ", "obseq"},
		}},
		{"tabs and newlines are boundaries too", "a\tb", []segment{
			{"", "a\tb"},
			{"a\t", "b"},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := trailingSegments(tc.line)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d segments %v, want %d %v", len(got), got, len(tc.want), tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("segment %d: got %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// The whole safety story of gluing is one invariant: a candidate is glued to a
// head and rendered as the user's own line, so if head+text ever differs from
// the line, the editor shows text nobody typed. A table is blind to the
// malformed-input class by construction, so this is a fuzz target.
func FuzzTrailingSegments(f *testing.F) {
	for _, s := range []string{"", " ", "obseq", "what is the diff", "hot dog", "¿qué obseq", "a\tb\nc", "  \t "} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, line string) {
		segs := trailingSegments(line)
		for i, s := range segs {
			if s.head+s.text != line {
				t.Fatalf("segment %d of %q: head %q + text %q = %q, want the line back",
					i, line, s.head, s.text, s.head+s.text)
			}
			if s.text == "" {
				t.Fatalf("segment %d of %q has empty text; hist.Prefix(\"\") returns everything", i, line)
			}
			if i > 0 && len(segs[i-1].text) <= len(s.text) {
				t.Fatalf("segments not strictly shortening: %d=%q then %d=%q", i-1, segs[i-1].text, i, s.text)
			}
		}
	})
}

// Segment 0 is always the whole line, which is what keeps pre-#20 whole-line
// recall working unchanged: historyCompletions tries it first.
func TestTrailingSegmentsStartsWithTheWholeLine(t *testing.T) {
	for _, line := range []string{"obseq", "what is the diff", "hot d", " leading"} {
		segs := trailingSegments(line)
		if len(segs) == 0 {
			t.Fatalf("%q produced no segments", line)
		}
		if segs[0].head != "" || segs[0].text != line {
			// A leading space belongs to the head, so segment 0 can differ there.
			if !strings.HasPrefix(line, " ") {
				t.Errorf("%q: segment 0 is %+v, want the whole line", line, segs[0])
			}
		}
	}
}

// tailFor drives the real path: type the line a key at a time through Apply,
// then ask for the grey tail exactly the way draw() does.
func tailFor(h History, line string) string {
	e := NewEditor()
	for _, r := range line {
		e, _ = Apply(e, Key{Kind: KeyRune, Rune: r}, candidatesFor(e.WalkBase(), h, commands))
	}
	return Suggestion(e, completionsFor(e.WalkBase(), h, commands))
}

func TestGreyTailCompletesADeckWordMidSentence(t *testing.T) {
	h := hist("obsequious")

	if got, want := tailFor(h, "what's the difference to obseq"), "uious"; got != want {
		t.Errorf("tail = %q, want %q — the whole point of #20", got, want)
	}
}

// The pre-#20 behaviour is segment 0 and must be untouched, including at two
// runes: the inner-segment floor would regress this if it were applied globally.
func TestGreyTailStillCompletesASingleWordAtTwoRunes(t *testing.T) {
	h := hist("obsequious")

	if got, want := tailFor(h, "ob"), "sequious"; got != want {
		t.Errorf("tail = %q, want %q", got, want)
	}
}

// A real past line always beats a word glued onto a head.
func TestWholeLineBeatsAnInnerSegment(t *testing.T) {
	h := hist("island", "hot dog and fries")

	if got, want := tailFor(h, "hot dog"), " and fries"; got != want {
		t.Errorf("tail = %q, want %q — segment 0 must win", got, want)
	}
}

// Two runes mid-line is more likely a preposition than a word being reached for.
func TestShortInnerSegmentsDoNotComplete(t *testing.T) {
	h := hist("torpid")

	if got := tailFor(h, "what is to"); got != "" {
		t.Errorf("tail = %q, want none — %q is under the floor", got, "to")
	}
	// Three runes is where it starts.
	if got, want := tailFor(h, "what is tor"), "pid"; got != want {
		t.Errorf("tail = %q, want %q", got, want)
	}
}

// A multi-word headword is a legal lookup, and matching only the last token
// could never complete one.
func TestMultiWordHeadwordCompletesMidLine(t *testing.T) {
	h := hist("hot dog")

	if got, want := tailFor(h, "I ate a hot d"), "og"; got != want {
		t.Errorf("tail = %q, want %q", got, want)
	}
}

// A trailing space must suggest nothing: History.Prefix("") returns everything
// by contract, so an empty segment would suggest a random line after every word.
func TestTrailingSpaceSuggestsNothing(t *testing.T) {
	h := hist("obsequious", "torpid")

	if got := tailFor(h, "what is "); got != "" {
		t.Errorf("tail = %q, want none", got)
	}
}

// The command namespace resolves once, on the whole line, and no trailing
// segment may re-enter it.
func TestCommandCompletionIsUnchanged(t *testing.T) {
	h := hist("historic")

	if got, want := tailFor(h, "/his"), "tory"; got != want {
		t.Errorf("tail = %q, want %q", got, want)
	}
	// Settled command: the completion is shorter than the line, so nothing.
	if got := tailFor(h, "/history 7"); got != "" {
		t.Errorf("tail = %q, want none", got)
	}
}

// The namespace is decided ONCE, on the whole line. If the segment loop ran
// first — or ran at all on a command line — the argument of a settled command
// would draw from history and glue a word onto it.
//
// This needs history that an inner segment actually matches: an earlier version
// of TestCommandCompletionIsUnchanged used history that matched nothing here, so
// swapping the two branches produced identical output and the test passed over
// the bug it was written to catch.
func TestACommandArgumentDoesNotDrawFromHistory(t *testing.T) {
	h := hist("historic", "sevenfold")

	if got := tailFor(h, "/history seven"); got != "" {
		t.Errorf("tail = %q, want none — %q must not complete from the deck", got, "seven")
	}
}

// The mirror: a "/…" segment mid-line must not reach the command MENU. History
// is empty, so any tail at all could only have come from commandCompletions.
func TestASlashSegmentMidLineDoesNotCompleteACommand(t *testing.T) {
	if got := tailFor(hist(), "see /his"); got != "" {
		t.Errorf("tail = %q, want none — the command menu is not reachable mid-line", got)
	}
}

// recallLine stores a force-literal lookup as "\word". The completion namespace
// wants the WORD: the marker disambiguates submission, not meaning.
func TestForcedLiteralLookupCompletesAsThePlainWord(t *testing.T) {
	h := hist(`\obsequious`)

	if got, want := tailFor(h, "about obseq"), "uious"; got != want {
		t.Errorf("tail = %q, want %q", got, want)
	}
}

// The "?" on an asked line is added by the SYSTEM — readsAsQuestion classifies a
// bare sentence and recallLine stores it "?…". Requiring the user to type "?" to
// complete a question they asked bare would make past questions uncompletable,
// which is the opposite of what #20 is for.
func TestPastQuestionCompletesWhenRetypedBare(t *testing.T) {
	h := hist("?what is the difference between wry and ironic")

	if got, want := tailFor(h, "what is the diff"), "erence between wry and ironic"; got != want {
		t.Errorf("tail = %q, want %q", got, want)
	}
	// Typing the marker still recalls it directly, as before.
	if got, want := tailFor(h, "?what is the diff"), "erence between wry and ironic"; got != want {
		t.Errorf("tail with marker = %q, want %q", got, want)
	}
}
