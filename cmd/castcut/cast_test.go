package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseCastRejects(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"v2", `{"version": 2}` + "\n[0, \"o\", \"x\"]\n", "expected asciicast v3"},
		{"no events", `{"version": 3}` + "\n\n", "no events"},
		{"empty", "\n \n", "empty file"},
		{"two-element event", `{"version": 3}` + "\n[0, \"o\"]\n", "t.cast:2: expected an event"},
		{"trailing data", `{"version": 3}` + "\n[0, \"o\", \"x\"] 1\n", "t.cast:2: expected an event"},
		{"negative gap", `{"version": 3}` + "\n[-1, \"o\", \"x\"]\n", "not a number of seconds"},
		{"kind not string", `{"version": 3}` + "\n[0, 1, \"x\"]\n", "kind is not a string"},
		{"huge interval", `{"version": 3}` + "\n[1e308, \"o\", \"a\"]\n", "t.cast:2: event interval is not a number of seconds"},
		{"overlong take", `{"version": 3}` + "\n[400000, \"o\", \"a\"]\n[400000, \"o\", \"b\"]\n", "t.cast:3: recording runs past 168 hours"},
		{"header not object", "[1]\n[0, \"o\", \"x\"]\n", "header is not a JSON object"},
	} {
		_, err := parseCast("t.cast", []byte(tc.in))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want %q", tc.name, err, tc.want)
		}
	}
}

func TestParseCastAcceptsFloatVersionAndLineEndings(t *testing.T) {
	c, err := parseCast("t.cast", []byte("{\"version\": 3.0}\r\n[0.5, \"o\", \"a\"]\r[1, \"i\", \"b\"]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Events) != 2 || c.Events[0].Gap != 0.5 || c.Events[1].Kind != "i" {
		t.Fatalf("events = %+v", c.Events)
	}
}

func TestEncodeCastRoundTripsUnknownFieldsAndData(t *testing.T) {
	in := `{"version": 3, "term": {"cols": 40}, "env": {"A": "<&>"}, "x": true}` + "\n" +
		`[0.25, "o", "\u001b[1mé\"\\"]` + "\n" + `[1e-7, "r", "50x12"]` + "\n"
	c, err := parseCast("t.cast", []byte(in))
	if err != nil {
		t.Fatal(err)
	}
	enc, err := encodeCast(c)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(enc, []byte(`"A":"<&>"`)) || !bytes.Contains(enc, []byte(`"x":true`)) {
		t.Errorf("header fields lost or HTML-escaped: %s", enc)
	}
	if !bytes.Contains(enc, []byte(`[0.0000001, "r", "50x12"]`)) {
		t.Errorf("tiny interval not in fixed notation: %s", enc)
	}
	back, err := parseCast("enc.cast", enc)
	if err != nil {
		t.Fatal(err)
	}
	for i := range c.Events {
		a, b := c.Events[i], back.Events[i]
		if a.Gap != b.Gap || a.Kind != b.Kind || !bytes.Equal(a.Data, b.Data) {
			t.Errorf("event %d: %+v != %+v", i, a, b)
		}
	}
}

func FuzzParseCast(f *testing.F) {
	f.Add([]byte(`{"version": 3}` + "\n[0, \"o\", \"x\"]\n"))
	f.Add([]byte(`{"version": 3, "idle_time_limit": 2}` + "\n[1.5, \"o\", \"\\u001b\"]\n[0, \"m\", \"\"]\n"))
	f.Add([]byte(`{"version": 3}` + "\n[0, \"o\"\n"))
	f.Add([]byte(`{"version": 3}` + "\n[\"x\", \"o\", 1]\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		c, err := parseCast("f.cast", data)
		if err != nil {
			return
		}
		enc, err := encodeCast(c)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		back, err := parseCast("enc.cast", enc)
		if err != nil {
			t.Fatalf("re-parse of encoded cast: %v\n%s", err, enc)
		}
		if len(back.Events) != len(c.Events) {
			t.Fatalf("event count %d != %d", len(back.Events), len(c.Events))
		}
		for i := range c.Events {
			if back.Events[i].Gap != c.Events[i].Gap || back.Events[i].Kind != c.Events[i].Kind {
				t.Fatalf("event %d changed: %+v != %+v", i, back.Events[i], c.Events[i])
			}
		}
	})
}
