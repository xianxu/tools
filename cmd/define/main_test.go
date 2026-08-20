package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// testDeps is for tests about the DEFINITION half: a real fixture dictionary,
// and an audio source that reports no recording. Every dependency is injected —
// a nil one would make run() panic rather than fail a test.
func testDeps(t *testing.T) deps {
	return deps{dict: testDict(t), audio: noAudioSource{}, player: &fakePlayer{}}
}

type noAudioSource struct{}

func (noAudioSource) Fetch(context.Context, []string) ([]byte, string, error) {
	return nil, "", ErrNoAudio
}

type audioRig struct {
	deps   deps
	cdn    *fakeCDN
	player *fakePlayer
}

// newAudioRig serves the real CDN path for `word` from the fake, so the shell
// exercises AudioCandidates' actual output rather than a hand-written URL.
func newAudioRig(t *testing.T, word string, present bool) *audioRig {
	t.Helper()
	files := map[string][]byte{}
	if present {
		first := AudioCandidates(word, "us")[0]
		files[stripHost(t, first, audioBase)] = []byte("ID3fakeaudio")
	}
	cdn := newFakeCDN(t, files)
	p := &fakePlayer{}
	return &audioRig{
		deps:   deps{dict: testDict(t), audio: &rebasedSource{cdn: cdn}, player: p},
		cdn:    cdn,
		player: p,
	}
}

// rebasedSource points the real fetch logic at the fake server while keeping the
// production URL derivation intact — the path that reaches the CDN is exactly
// what AudioCandidates produced.
type rebasedSource struct{ cdn *fakeCDN }

func (r *rebasedSource) Fetch(ctx context.Context, urls []string) ([]byte, string, error) {
	rebased := make([]string, len(urls))
	for i, u := range urls {
		rebased[i] = r.cdn.URL + strings.TrimPrefix(u, audioBase)
	}
	return r.cdn.source().Fetch(ctx, rebased)
}

func TestRunPrintsDefinition(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"sycophantic"}, testDeps(t), &out, &errb); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}
	got := out.String()
	for _, want := range []string{"sycophantic", "/ˌsikəˈfan(t)ik/", "adjective", "obsequious"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestRunUnknownWordExitsOne(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"rizz"}, testDeps(t), &out, &errb)
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "rizz") {
		t.Errorf("stderr should name the word, got %q", errb.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout should stay empty, got %q", out.String())
	}
}

func TestRunNoArgsIsUsageError(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(nil, testDeps(t), &out, &errb); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

func TestRunRawPrintsUnparsed(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"-raw", "quokka"}, testDeps(t), &out, &errb); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "quok·ka | ˈkwäkə |") {
		t.Errorf("raw output should be the unparsed entry, got %q", out.String())
	}
}

// Colour must be off for a non-TTY writer so piping yields clean text.
func TestRunNoColorWhenNotATerminal(t *testing.T) {
	var out, errb bytes.Buffer
	run([]string{"quokka"}, testDeps(t), &out, &errb)
	if strings.Contains(out.String(), "\x1b[") {
		t.Error("ANSI escapes leaked into non-TTY output")
	}
}

// --- audio wiring ----------------------------------------------------------

func TestRunPlaysThreeTimesByDefault(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	if code := run([]string{"sycophantic"}, rig.deps, &out, &errb); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}
	if got := rig.player.count(); got != 3 {
		t.Errorf("played %d times, want 3", got)
	}
	if !strings.Contains(out.String(), "playing 3") {
		t.Error("output should announce playback")
	}
}

func TestRunTimesFlag(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	run([]string{"-times", "1", "sycophantic"}, rig.deps, &out, &errb)
	if got := rig.player.count(); got != 1 {
		t.Errorf("played %d times, want 1", got)
	}
}

// --no-audio must skip the FETCH too, not just the playback — a suppressed
// sound should not still cost a network round trip.
func TestRunNoAudioMakesNoRequests(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	if code := run([]string{"-no-audio", "sycophantic"}, rig.deps, &out, &errb); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if got := rig.player.count(); got != 0 {
		t.Errorf("played %d times, want 0", got)
	}
	if got := rig.cdn.Requested(); len(got) != 0 {
		t.Errorf("made %d CDN requests with -no-audio: %v", len(got), got)
	}
	if strings.Contains(out.String(), "playing") {
		t.Error("output announced playback with -no-audio")
	}
}

// A missing recording is not a failed lookup: the definition is the deliverable.
func TestRunMissingAudioStillSucceeds(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", false) // CDN 404s every candidate
	var out, errb bytes.Buffer
	if code := run([]string{"sycophantic"}, rig.deps, &out, &errb); code != 0 {
		t.Errorf("exit = %d, want 0 — the definition printed fine", code)
	}
	if !strings.Contains(out.String(), "/ˌsikəˈfan(t)ik/") {
		t.Error("definition missing from stdout")
	}
	if errb.Len() == 0 {
		t.Error("a missing recording should warn on stderr")
	}
	if got := rig.player.count(); got != 0 {
		t.Errorf("played %d times despite no audio", got)
	}
}

// The URLs that reach the CDN must be the ones AudioCandidates derived, in order.
func TestRunUsesDerivedCandidateOrder(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	run([]string{"sycophantic"}, rig.deps, &out, &errb)
	want := stripHost(t, AudioCandidates("sycophantic", "us")[0], audioBase)
	got := rig.cdn.Requested()
	if len(got) != 1 || got[0] != want {
		t.Errorf("requested %v, want exactly [%s]", got, want)
	}
}

func TestRunNegativeTimesIsUsageError(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	if code := run([]string{"-times", "-1", "sycophantic"}, rig.deps, &out, &errb); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}
