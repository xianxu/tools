package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestParseBilingualArgs(t *testing.T) {
	for _, tc := range []struct {
		args          []string
		current, want bool
		bad           bool
	}{
		{nil, true, false, false}, {nil, false, true, false}, {[]string{"on"}, false, true, false},
		{[]string{"off"}, true, false, false}, {[]string{"ON"}, false, true, false},
		{[]string{"off", "on"}, true, true, true}, {[]string{"yes"}, false, false, true},
	} {
		got, err := parseBilingualArgs(tc.args, tc.current)
		if (err != nil) != tc.bad || err == nil && got != tc.want {
			t.Errorf("%v: %v %v", tc.args, got, err)
		}
	}
}

func TestBilingualCommandEffects(t *testing.T) {
	for _, tc := range []struct {
		name      string
		persist   error
		available bool
		want      bool
		saved     bool
	}{
		{"saved", nil, true, false, true}, {"session only", nil, false, false, false},
		{"declined", errDeckDeclined, true, false, false}, {"write failure", errors.New("disk failed"), true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := deps{}
			var writes int
			var persist func(bool) error
			if tc.available {
				persist = func(bool) error { writes++; return tc.persist }
			}
			saved, err := sessionSetBilingual(&d, persist)(false)
			if d.bilingualEnabled() != tc.want || saved != tc.saved {
				t.Fatalf("enabled=%v saved=%v err=%v", d.bilingualEnabled(), saved, err)
			}
			if (err != nil) != (tc.name == "write failure") {
				t.Fatalf("error=%v", err)
			}
			if tc.available && writes != 1 {
				t.Fatal("write not attempted once")
			}
		})
	}
	var out, errout bytes.Buffer
	c := commandCtx{bilingual: true, stdout: &out, stderr: &errout}
	if runBilingual(c, nil) != 2 || out.Len() != 0 {
		t.Fatal("one-shot without setter claimed success")
	}
	c.setBilingual = func(bool) (bool, error) { return false, errDeckDeclined }
	if runBilingual(c, nil) != 2 {
		t.Fatal("one-shot decline claimed session effect")
	}
	d := deps{}
	c.setBilingual = sessionSetBilingual(&d, nil)
	if runBilingual(c, nil) != 0 || !strings.Contains(out.String(), "session only") {
		t.Fatalf("%s %s", &out, &errout)
	}
}

func FuzzParseBilingualArgs(f *testing.F) {
	for _, s := range []string{"", "on", "off", "on off", "yes"} {
		f.Add(s, true)
	}
	f.Fuzz(func(t *testing.T, s string, current bool) {
		args := strings.Fields(s)
		got, err := parseBilingualArgs(args, current)
		valid := len(args) == 0 || len(args) == 1 && (strings.EqualFold(args[0], "on") || strings.EqualFold(args[0], "off"))
		if (err == nil) != valid {
			t.Fatalf("%q: %v", s, err)
		}
		if err == nil && len(args) == 0 && got == current {
			t.Fatal("did not toggle")
		}
	})
}

func TestBilingualExplicitSetsAreIdempotent(t *testing.T) {
	for _, on := range []bool{false, true} {
		t.Run(bilingualState(on), func(t *testing.T) {
			current := on
			d := deps{bilingual: &current}
			var writes []bool
			persist := func(value bool) error { writes = append(writes, value); return nil }
			for i := 0; i < 2; i++ {
				var out, errout bytes.Buffer
				cc := newCommandCtx(d, options{}, &out, &errout)
				cc.setBilingual = sessionSetBilingual(&d, persist)
				code := dispatchCommand(replCommand{name: "bilingual", args: []string{bilingualState(on)}}, commands, cc)
				if code != 0 || d.bilingualEnabled() != on {
					t.Fatalf("set %v iteration %d: state=%v code=%d stderr=%s", on, i, d.bilingualEnabled(), code, &errout)
				}
			}
			for _, saved := range writes {
				if saved != on {
					t.Fatalf("persisted %v, want %v", saved, on)
				}
			}
		})
	}
}

func TestBilingualHelpDoesNotMutate(t *testing.T) {
	for _, command := range []replCommand{
		{name: "help", args: []string{"bilingual"}},
		{name: "bilingual", args: []string{"--help"}},
		{name: "bilingual", args: []string{"-h"}},
	} {
		for _, on := range []bool{false, true} {
			current := on
			d := deps{bilingual: &current}
			var out, errout bytes.Buffer
			cc := newCommandCtx(d, options{}, &out, &errout)
			cc.setBilingual = sessionSetBilingual(&d, func(bool) error { t.Fatal("help wrote the setting"); return nil })
			if code := dispatchCommand(command, commands, cc); code != 0 || d.bilingualEnabled() != on {
				t.Fatalf("help mutated state or failed: code=%d state=%v stderr=%s", code, d.bilingualEnabled(), &errout)
			}
			if !strings.Contains(out.String(), bilingualUsage) {
				t.Fatalf("missing bilingual usage: %s", &out)
			}
		}
	}
}
