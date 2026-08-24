package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseSoundArgs(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		want    int
		set     bool
		wantErr string
	}{
		{"no argument reports", nil, 0, false, ""},
		{"a count", []string{"1"}, 1, true, ""},
		{"zero turns playback off", []string{"0"}, 0, true, ""},
		{"the cap itself is fine", []string{"20"}, 20, true, ""},
		{"negative", []string{"-2"}, 0, false, "-2"},
		{"not a number", []string{"loud"}, 0, false, "loud"},
		// A fat-fingered /sound 1000 would wedge the session behind twenty
		// minutes of playback that Ctrl-C is the only escape from.
		{"absurd is refused by name", []string{"1000"}, 0, false, "20"},
		{"too many arguments", []string{"1", "2"}, 0, false, "2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, set, err := parseSoundArgs(tc.args)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("want an error naming %q, got %d", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error %q does not name %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want || set != tc.set {
				t.Errorf("= (%d, %v), want (%d, %v)", got, set, tc.want, tc.set)
			}
		})
	}
}

func TestRunSound(t *testing.T) {
	t.Run("reports the current count", func(t *testing.T) {
		var out bytes.Buffer
		c := commandCtx{times: 3, stdout: &out, stderr: &bytes.Buffer{}}
		if code := runSound(c, nil); code != 0 {
			t.Errorf("exit = %d", code)
		}
		if !strings.Contains(out.String(), "3") {
			t.Errorf("did not report the count: %q", out.String())
		}
	})

	t.Run("sets it through the session seam", func(t *testing.T) {
		got := -1
		var out bytes.Buffer
		c := commandCtx{times: 3, setTimes: func(n int) { got = n }, stdout: &out, stderr: &bytes.Buffer{}}
		if code := runSound(c, []string{"1"}); code != 0 {
			t.Errorf("exit = %d", code)
		}
		if got != 1 {
			t.Errorf("setTimes got %d, want 1", got)
		}
	})

	// Without a session there is nothing to change, and silently accepting the
	// command would be a lie about what it did.
	t.Run("refuses outside a session", func(t *testing.T) {
		var errb bytes.Buffer
		c := commandCtx{times: 3, stdout: &bytes.Buffer{}, stderr: &errb}
		if code := runSound(c, []string{"1"}); code != 2 {
			t.Errorf("exit = %d, want 2", code)
		}
		if !strings.Contains(errb.String(), "session") {
			t.Errorf("stderr does not explain why: %q", errb.String())
		}
	})

	t.Run("a bad count names the operand", func(t *testing.T) {
		var errb bytes.Buffer
		c := commandCtx{times: 3, setTimes: func(int) {}, stdout: &bytes.Buffer{}, stderr: &errb}
		if code := runSound(c, []string{"loud"}); code != 2 {
			t.Errorf("exit = %d, want 2", code)
		}
		if !strings.Contains(errb.String(), "loud") {
			t.Errorf("stderr does not name the operand: %q", errb.String())
		}
	})
}

// The seam has to change the SESSION, not just call a closure. This drives the
// real loop: /sound 1, then a word, and the player must be asked once.
func TestSoundChangesPlaybackForTheRestOfTheSession(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	rig.deps.stdinIsTerminal = func() bool { return false }

	var out, errb bytes.Buffer
	replLines(t.Context(), nil, rig.deps, options{times: 3, locale: "us"},
		strings.NewReader("/sound 1\nsycophantic\n"), &out, &errb, true, false)

	if got := rig.player.count(); got != 1 {
		t.Errorf("played %d times after /sound 1, want 1", got)
	}
	if !strings.Contains(out.String(), "once") {
		t.Errorf("the change was not reported: %q", out.String())
	}
}
