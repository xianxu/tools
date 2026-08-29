package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// testDeps is for tests about the DEFINITION half: a real fixture dictionary,
// and an audio source that reports no recording. Every dependency is injected —
// a nil one would make run() panic rather than fail a test.
func testDeps(t *testing.T) deps {
	return deps{dict: testDict(t), audio: noAudioSource{}, player: &fakePlayer{},
		langDeps: langDeps{capture: noopCapturer{}}}
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
	if !present {
		return newAudioRigServing(t)
	}
	return newAudioRigServing(t, AudioCandidates(word, voice{Lang: "en", Locale: "us"})[0])
}

// newAudioRigServing is newAudioRig's general form: it serves EXACTLY the URLs
// given and nothing else.
//
// #29 needs worlds newAudioRig cannot describe — one where the source-language
// recording is missing and only the session's exists, which is what makes the
// fallback and its report testable at all. Taking whole URLs from
// AudioCandidates rather than paths keeps the production derivation in the loop,
// which is the property newAudioRig was written for and this must not lose.
func newAudioRigServing(t *testing.T, urls ...string) *audioRig {
	t.Helper()
	files := map[string][]byte{}
	for _, u := range urls {
		files[stripHost(t, u, audioBase)] = []byte("ID3fakeaudio")
	}
	cdn := newFakeCDN(t, files)
	p := &fakePlayer{}
	return &audioRig{
		deps: deps{dict: testDict(t), audio: &rebasedSource{cdn: cdn}, player: p,
			langDeps: langDeps{capture: noopCapturer{}}},
		cdn:    cdn,
		player: p,
	}
}

// rebasedSource points the real fetch logic at the fake server while keeping the
// production URL derivation intact — the path that reaches the CDN is exactly
// what AudioCandidates produced.
//
// It TRANSLATES THE ANSWER BACK, and that is not tidiness. #29 made the returned
// URL load-bearing: reportVoice decides which voice was heard by testing that
// URL for membership in the candidate list. Handing back the rebased form leaks
// the rig's own rewriting into the value under assertion, so a source recording
// that answered perfectly well is reported as a fallback — a test failure with
// no defect behind it. In production nothing rebases, so `from` is always one of
// the URLs passed in; the double has to keep that true.
type rebasedSource struct{ cdn *fakeCDN }

func (r *rebasedSource) Fetch(ctx context.Context, urls []string) ([]byte, string, error) {
	rebased := make([]string, len(urls))
	origin := make(map[string]string, len(urls))
	for i, u := range urls {
		rebased[i] = r.cdn.URL + strings.TrimPrefix(u, audioBase)
		origin[rebased[i]] = u
	}
	data, from, err := r.cdn.source().Fetch(ctx, rebased)
	if was, ok := origin[from]; ok {
		from = was
	}
	return data, from, err
}

func TestRunPrintsDefinition(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"sycophantic"}, testDeps(t), strings.NewReader(""), &out, &errb); code != 0 {
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
	code := run(t.Context(), []string{"rizz"}, testDeps(t), strings.NewReader(""), &out, &errb)
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

// Rewritten, not deleted: no-args used to be a usage error and is now the loop.
// Keeping a test on this branch at all times is what makes the contract change
// visible in the diff rather than a test quietly disappearing.
func TestRunNoArgsEntersTheLoop(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	rig.deps.stdinIsTerminal = func() bool { return false }
	var out, errb bytes.Buffer

	code := run(t.Context(), nil, rig.deps, strings.NewReader("sycophantic\n"), &out, &errb)
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr = %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "/ˌsikəˈfan(t)ik/") {
		t.Error("no-args should have read the word from stdin and defined it")
	}
}

// `echo word | define` is NEW capability: before this issue it exited 2 with
// usage. Asserted as new, not as preserved.
func TestRunPipedStdinDefinesTheWord(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	rig.deps.stdinIsTerminal = func() bool { return false }
	var out, errb bytes.Buffer

	if code := run(t.Context(), nil, rig.deps, strings.NewReader("sycophantic\n"), &out, &errb); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if strings.Contains(out.String(), prompt) {
		t.Error("piped stdin must not print a prompt")
	}
}

// More than one positional argument is still a usage error.
func TestRunTooManyArgsIsUsageError(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"a", "b"}, testDeps(t), strings.NewReader(""), &out, &errb); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

func TestRunRawPrintsUnparsed(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-raw", "quokka"}, testDeps(t), strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "quok·ka | ˈkwäkə |") {
		t.Errorf("raw output should be the unparsed entry, got %q", out.String())
	}
}

// Colour must be off for a non-TTY writer so piping yields clean text.
func TestRunNoColorWhenNotATerminal(t *testing.T) {
	var out, errb bytes.Buffer
	run(t.Context(), []string{"quokka"}, testDeps(t), strings.NewReader(""), &out, &errb)
	if strings.Contains(out.String(), "\x1b[") {
		t.Error("ANSI escapes leaked into non-TTY output")
	}
}

// --- audio wiring ----------------------------------------------------------

func TestRunPlaysThreeTimesByDefault(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"sycophantic"}, rig.deps, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}
	if got := rig.player.count(); got != 3 {
		t.Errorf("played %d times, want 3", got)
	}
	// Pin the exact record form, not just its presence. Every other assertion on
	// this line is a Contains/Count, so dropping the trailing newline was a green
	// mutation — and `define word | cat` would then glue the record to whatever
	// followed it.
	if !strings.HasSuffix(out.String(), "\n  ♫ playing 3×\n") {
		t.Errorf("record form changed: %q", out.String())
	}
}

// -raw is the scripting form and must mean the same thing on both paths. It
// returned before playing one-shot, while the loop's replay branch ignored it
// and fetched — so the flag meant two things depending on which line you were on.
func TestRawNeverPlays(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	rig.deps.stdinIsTerminal = func() bool { return false }
	var out, errb bytes.Buffer

	// One-shot, then the loop (a word plus a bare return, which is where it played).
	run(t.Context(), []string{"-raw", "sycophantic"}, rig.deps, strings.NewReader(""), &out, &errb)
	run(t.Context(), []string{"-raw"}, rig.deps, strings.NewReader("sycophantic\n\n"), &out, &errb)

	if got := rig.player.count(); got != 0 {
		t.Errorf("-raw played %d times, want 0", got)
	}
	if got := rig.cdn.Requested(); len(got) != 0 {
		t.Errorf("-raw made %d CDN requests, want 0: %v", len(got), got)
	}
}

func TestRunTimesFlag(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	run(t.Context(), []string{"-times", "1", "sycophantic"}, rig.deps, strings.NewReader(""), &out, &errb)
	if got := rig.player.count(); got != 1 {
		t.Errorf("played %d times, want 1", got)
	}
}

// --no-audio must skip the FETCH too, not just the playback — a suppressed
// sound should not still cost a network round trip.
func TestRunNoAudioMakesNoRequests(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-no-audio", "sycophantic"}, rig.deps, strings.NewReader(""), &out, &errb); code != 0 {
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
	if code := run(t.Context(), []string{"sycophantic"}, rig.deps, strings.NewReader(""), &out, &errb); code != 0 {
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
	// A non-erasable indicator is a record, and it played zero times. Without
	// this, `define <word-with-no-recording> > out.txt` files a claim it played
	// three. TestRunNoAudioMakesNoRequests already used exactly this assertion.
	if strings.Contains(out.String(), "playing") {
		t.Errorf("announced playback that never happened: %q", out.String())
	}
}

// The URLs that reach the CDN must be the ones AudioCandidates derived, in order.
func TestRunUsesDerivedCandidateOrder(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	run(t.Context(), []string{"sycophantic"}, rig.deps, strings.NewReader(""), &out, &errb)
	want := stripHost(t, AudioCandidates("sycophantic", voice{Lang: "en", Locale: "us"})[0], audioBase)
	got := rig.cdn.Requested()
	if len(got) != 1 || got[0] != want {
		t.Errorf("requested %v, want exactly [%s]", got, want)
	}
}

func TestRunNegativeTimesIsUsageError(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-times", "-1", "sycophantic"}, rig.deps, strings.NewReader(""), &out, &errb); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

// -pron applies to ONE lookup and must not become a session mode (#29 D3): a
// pronunciation language in opt.voice would survive a /lang switch and ask for
// fr_fr recordings in a Spanish session.
func TestPronWithoutAWordIsRefusedAndPointsAtTheCommand(t *testing.T) {
	var out, errb bytes.Buffer
	code := run(t.Context(), []string{"-pron", "fr"}, testDeps(t), strings.NewReader(""), &out, &errb)
	if code != 2 {
		t.Errorf("exit = %d, want 2 (a usage error)", code)
	}
	if !strings.Contains(errb.String(), "/pron") {
		t.Errorf("the refusal must name the in-session form; stderr = %q", errb.String())
	}
}

func TestPronRejectsSomethingThatIsNotALanguageTag(t *testing.T) {
	var out, errb bytes.Buffer
	code := run(t.Context(), []string{"-pron", "french", "sycophantic"}, testDeps(t), strings.NewReader(""), &out, &errb)
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	// store.ParseLang's message, not a second one invented at the flag.
	if !strings.Contains(errb.String(), "two letters") {
		t.Errorf("stderr should carry ParseLang's complaint, got %q", errb.String())
	}
}

// The whole point of #29, end to end: the RECORDING comes from the source
// language while the deck, the dictionary and the session stay English.
//
// jalapeno is the case that needs every piece — typed unaccented, headword
// jalapeño, and jalapeno_es_es is a 404 where jalapeño_es_es is a 200.
func TestPronFetchesTheSourceRecordingWithoutMovingTheSession(t *testing.T) {
	es := voice{Lang: "es", Locale: "es"}
	rig := newAudioRigServing(t, AudioCandidates("jalapeño", es)[0])
	cap := &countingCapturer{}
	rig.deps.capture = cap
	var out, errb bytes.Buffer

	code := run(t.Context(), []string{"-pron", "es", "jalapeno"}, rig.deps, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}
	asked := rig.cdn.Requested()
	if len(asked) == 0 {
		t.Fatal("nothing was requested from the CDN")
	}
	// FIRST, and spelled the way Spanish spells it — not the way it was typed.
	want := stripHost(t, AudioCandidates("jalapeño", es)[0], audioBase)
	if asked[0] != want {
		t.Errorf("first request = %q, want %q", asked[0], want)
	}
	// And it STOPPED there. The typed spelling is a legitimate later candidate —
	// SourceSpellings keeps it as the safety net — so "never asked" would be the
	// wrong assertion; "never needed, because the accented one answered" is the
	// right one, and it is what saves jalapeno_es_es's measured 404.
	if len(asked) != 1 {
		t.Errorf("walked %d candidates, want 1 — the first answered: %q", len(asked), asked)
	}
	// The English entry was still printed, and nothing was reported: the source
	// recording answered.
	if !strings.Contains(out.String(), "chili pepper") {
		t.Errorf("the English entry should still be shown, got %q", out.String())
	}
	if errb.Len() != 0 {
		t.Errorf("the source recording answered, so nothing should be reported: %q", errb.String())
	}

	// THE SESSION DID NOT MOVE — #29's first Done-when, and the reason this test
	// is named for it. Asserting the Spanish URL alone would pass just as well
	// for a -pron that had quietly switched the whole session to Spanish, which
	// is the mode this issue exists NOT to be. The capture is where a lookup's
	// language becomes observable: a word files into words/<lang>/, so the voice
	// at capture time IS the deck it landed in.
	if len(cap.calls) != 1 || cap.calls[0] != "jalapeno" {
		t.Fatalf("captured %v, want exactly [jalapeno]", cap.calls)
	}
	if got := cap.voices[0]; got.Lang != store.DefaultLang {
		t.Errorf("the word was filed under %q, want %q — -pron moved the session, "+
			"which is the mode #29 exists not to be", got.Lang, store.DefaultLang)
	}
}

// Done-when 6 / D4: the locale for a source recording comes from #27's policy,
// which means -locale keeps qualifying whatever language is IN EFFECT.
//
// `-pron es` alone is Castilian (es_es, where cazar /θ/ and casar /s/ differ);
// `-pron es -locale us` is Latin American seseo, where both are /s/. Choosing
// one chooses which sound system a learner hears, so it is not a flavour.
//
// This existed as an ARGUMENT — "voiceFor is unchanged, so the locale is
// literally #27's" — and the close review showed the argument is not a pin:
// utteranceFor's CALL SITE is new code, and rewriting it to voiceFor(pron, "")
// left the entire suite green. A reused function cannot vouch for a new caller.
func TestPronHonoursTheLocaleFlagForTheSourceLanguage(t *testing.T) {
	seseo := voice{Lang: "es", Locale: "us"}
	rig := newAudioRigServing(t, AudioCandidates("jalapeño", seseo)[0])
	var out, errb bytes.Buffer

	code := run(t.Context(), []string{"-pron", "es", "-locale", "us", "jalapeno"},
		rig.deps, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}
	asked := rig.cdn.Requested()
	if len(asked) == 0 {
		t.Fatal("nothing was requested from the CDN")
	}
	want := stripHost(t, AudioCandidates("jalapeño", seseo)[0], audioBase)
	if asked[0] != want {
		t.Errorf("first request = %q, want %q — -locale did not reach the SOURCE voice, "+
			"so -pron es -locale us asked for Castilian when seseo was requested",
			asked[0], want)
	}
	if errb.Len() != 0 {
		t.Errorf("the es_us recording answered, so nothing should be reported: %q", errb.String())
	}
}
