package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseCaptions(t *testing.T) {
	in := "~1:02.5  late one\n\n0:05 no tilde\r\n  ~0:05  a tie sorts by text  \r~0:00.25 first\n"
	caps, err := parseCaptions("c.txt", []byte(in))
	if err != nil {
		t.Fatal(err)
	}
	want := []Caption{{0.25, "first"}, {5, "a tie sorts by text"}, {5, "no tilde"}, {62.5, "late one"}}
	if fmt.Sprint(caps) != fmt.Sprint(want) {
		t.Fatalf("caps = %v\nwant   %v", caps, want)
	}
}

func TestParseCaptionsNamesTheBadLine(t *testing.T) {
	for _, in := range []string{"~0:01  ok\n1:2\n", "~0:01  ok\nno stamp here\n", "~0:01  ok\n~0:1x  text\n"} {
		_, err := parseCaptions("c.txt", []byte(in))
		if err == nil || !strings.HasPrefix(err.Error(), "c.txt:2: expected `~m:ss.s  text`") {
			t.Errorf("%q: err = %v", in, err)
		}
	}
}

func TestParseCaptionsBoundsStamps(t *testing.T) {
	for _, line := range []string{"~153722867280912931:00  overflow", "~99999999999999999999:00  too big", "~10081:00  past a week", "~10080:00.5  just past"} {
		_, err := parseCaptions("c.txt", []byte(line+"\n"))
		if err == nil || !strings.Contains(err.Error(), "c.txt:1: stamp") {
			t.Errorf("%q: err = %v", line, err)
		}
	}
	if caps, err := parseCaptions("c.txt", []byte("~10080:00  a week exactly\n")); err != nil || caps[0].At != maxSeconds {
		t.Errorf("a week: %v %v", caps, err)
	}
}

func TestSidecarPath(t *testing.T) {
	for in, want := range map[string]string{
		"recordings/take-01.cast": "recordings/take-01.captions.txt",
		"take":                    "take.captions.txt",
		"a.b/take.json":           "a.b/take.captions.txt",
	} {
		if got := sidecarPath(in); got != want {
			t.Errorf("sidecarPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func FuzzParseCaptions(f *testing.F) {
	f.Add([]byte("~0:03.0  hello world\n0:12 x\n"))
	f.Add([]byte("~0:03.0\n"))
	f.Add([]byte("~99999999999999999999:00  overflow\n"))
	f.Add([]byte("\r\r~1:1.5\ttab separated\r"))
	f.Fuzz(func(t *testing.T, data []byte) {
		caps, err := parseCaptions("f.txt", data)
		if err != nil {
			return
		}
		for i, c := range caps {
			if c.At < 0 || c.Text == "" || c.Text != strings.TrimSpace(c.Text) {
				t.Fatalf("caption %d malformed: %+v", i, c)
			}
			if i > 0 && caps[i-1].At > c.At {
				t.Fatalf("captions not sorted at %d", i)
			}
		}
	})
}
