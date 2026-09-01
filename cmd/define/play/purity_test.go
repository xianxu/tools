package play_test

import (
	"os"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/puretest"
)

const playPkg = "github.com/xianxu/tools/cmd/define/play"

// TWO of schedule's three guards, from the same shared body.
//
// play is stricter: it needs no store symbols at all today, because the session
// works in deck KEYS the caller supplies rather than in store types. Declaring
// an empty allowlist would make the guard vacuous (it fatals when it finds
// nothing), so the store guard is deliberately absent and the import guard
// carries the weight — if play ever imports store, that line has to be added
// consciously, which is the point.
func TestPlayPurity(t *testing.T) {
	t.Run("imports", func(t *testing.T) {
		puretest.ImportsOnly(t, playPkg, []string{})
	})
	t.Run("no wall clock", func(t *testing.T) {
		puretest.NoWallClock(t, playPkg)
	})
}

// DONE-WHEN 8: THE SESSION STILL LEARNS NOTHING ABOUT WHICH FORM IS ASKING.
//
// `TestSessionIsFormAgnostic` says the machine works with a form it has never
// seen. This says the opposite thing about the same claim: that the machine does
// not NAME one. A type switch on `*Board` would be exactly what `#6`'s Done-when
// forbids, and the five capability interfaces exist so it never has to.
//
// A grep, deliberately, and it is the kind that can fail: the file is read from
// disk rather than reasoned about, and every concrete form is checked rather
// than only the one this issue added.
func TestTheSessionNamesNoForm(t *testing.T) {
	b, err := os.ReadFile("session.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	// Strip comments: `session.go`'s prose legitimately mentions the forms by
	// name to explain why the capabilities exist, and a grep that could not tell
	// prose from code would force the explanations out.
	var code strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		code.WriteString(line + "\n")
	}
	for _, form := range []string{"Board", "Choice", "Recall"} {
		for _, ref := range []string{"*" + form, "(" + form + ")", form + "{"} {
			if strings.Contains(code.String(), ref) {
				t.Errorf("session.go names %q in code — the session must ask a CAPABILITY, never a form (#6's Done-when)", ref)
			}
		}
	}
	// The premise: the capabilities it asks about ARE there, so the assertions
	// above are not passing because the file is empty.
	for _, cap := range []string{"Missed", "SelfRated", "Batch", "Moded", "Grid"} {
		if !strings.Contains(code.String(), cap) {
			t.Errorf("session.go does not mention the %s capability; this test is checking the wrong file", cap)
		}
	}
}
