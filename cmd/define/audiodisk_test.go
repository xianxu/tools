package main

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// diskRig is a store in a real directory plus a CDN that counts requests.
//
// A REAL DIRECTORY, not a fake store: the whole claim of this file is that bytes
// survive a process, and a map cannot be wrong about that in the way a
// filesystem can (a missing MkdirAll, an unwritable path, a name that is not a
// legal file). ARCH-MOCK — the fake is the CDN, at the wire, which is the thing
// that would otherwise cost a network call.
type diskRig struct {
	dir string
	cdn *fakeCDN
}

func newDiskRig(t *testing.T, files map[string][]byte) diskRig {
	t.Helper()
	return diskRig{dir: t.TempDir(), cdn: newFakeCDN(t, files)}
}

// seam builds the production layering: memo over disk over the CDN. A NEW seam
// and a NEW store each time, which is what makes two calls to this the moral
// equivalent of two runs of the binary.
func (r diskRig) seam(t *testing.T) *audioSeam {
	t.Helper()
	st := store.NewYAML(r.dir, store.DefaultLang, io.Discard)
	return newAudioSeam(newDiskAudioCache(st, r.cdn.source()))
}

// THE POINT OF THE WHOLE MILESTONE: a second PROCESS pays nothing.
//
// The memo already made a second play within one sitting free. This is the half
// it cannot do — everything in memory is gone between two runs of `define`, and
// a deck reviewed daily would otherwise re-fetch the same recordings daily.
func TestASecondRunReusesTheRecordingOnDisk(t *testing.T) {
	r := newDiskRig(t, map[string][]byte{"/a.mp3": []byte("ID3audio")})
	urls := r.cdn.urls("/a.mp3")

	first, from, err := r.seam(t).FetchFor(t.Context(), "sycophantic", urls)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	second, from2, err := r.seam(t).FetchFor(t.Context(), "sycophantic", urls)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if string(first) != string(second) {
		t.Errorf("the two runs got different bytes: %q vs %q", first, second)
	}
	// AND THE PROVENANCE SURVIVES. `from` is what spokeSource reads to decide
	// whether a source voice answered, and reportVoice prints a record from that
	// — a cache hit that forgets it makes the record silent or false.
	if from2 != from {
		t.Errorf("the cached hit reports from=%q, the live one reported %q", from2, from)
	}
	if got := r.cdn.Requested(); len(got) != 1 {
		t.Errorf("two runs made %d requests, want 1: %v", len(got), got)
	}
}

// A word with no recording is asked ONCE, not four candidate URLs per replay per
// day. This is the half of the cache that saves the most requests, because a
// miss costs a request per candidate where a hit costs one.
func TestASecondRunDoesNotReaskAWordWithNoRecording(t *testing.T) {
	r := newDiskRig(t, nil) // every candidate 404s
	urls := r.cdn.urls("/a.mp3", "/b.mp3", "/c.mp3")

	if _, _, err := r.seam(t).FetchFor(t.Context(), "quokka", urls); !errors.Is(err, ErrNoAudio) {
		t.Fatalf("first run: %v, want ErrNoAudio", err)
	}
	before := len(r.cdn.Requested())
	if _, _, err := r.seam(t).FetchFor(t.Context(), "quokka", urls); !errors.Is(err, ErrNoAudio) {
		t.Fatalf("second run: %v, want ErrNoAudio", err)
	}
	if after := len(r.cdn.Requested()); after != before {
		t.Errorf("the second run re-asked %d candidates; a recorded verdict means none", after-before)
	}
}

// A VERDICT EXPIRES. The CDN gains recordings, and the rare words a learner most
// wants are the likeliest to gain one — so a word that missed once must not be
// unplayable forever.
func TestAStaleVerdictIsReasked(t *testing.T) {
	r := newDiskRig(t, nil)
	urls := r.cdn.urls("/a.mp3")
	if _, _, err := r.seam(t).FetchFor(t.Context(), "quokka", urls); !errors.Is(err, ErrNoAudio) {
		t.Fatalf("first run: %v", err)
	}
	before := len(r.cdn.Requested())

	st := store.NewYAML(r.dir, store.DefaultLang, io.Discard)
	disk := newDiskAudioCache(st, r.cdn.source())
	disk.now = func() time.Time { return time.Now().Add(store.AudioVerdictTTL + time.Hour) }
	if _, _, err := newAudioSeam(disk).FetchFor(t.Context(), "quokka", urls); !errors.Is(err, ErrNoAudio) {
		t.Fatalf("later run: %v", err)
	}
	if after := len(r.cdn.Requested()); after <= before {
		t.Error("a verdict past its TTL was still believed; the word can never start working")
	}
}

// A TRANSPORT FAILURE IS NOT A VERDICT. Recording "no recording" during an
// outage would poison the word for a month — the taxonomy in fetch.go is the
// single source of that distinction and this is where it has teeth.
func TestAnOutageIsNotRecordedAsAMissingRecording(t *testing.T) {
	r := newDiskRig(t, nil)
	urls := r.cdn.urls("/a.mp3")
	r.cdn.Close()

	if _, _, err := r.seam(t).FetchFor(t.Context(), "sycophantic", urls); !errors.Is(err, ErrFetchFailed) {
		t.Fatalf("got %v, want ErrFetchFailed", err)
	}
	st := store.NewYAML(r.dir, store.DefaultLang, io.Discard)
	_, rec, err := st.Audio(store.NewAudioKey("sycophantic", urls))
	if err != nil {
		t.Fatal(err)
	}
	if rec.Missing || !rec.At.IsZero() {
		t.Errorf("an outage was written to disk as a verdict: %+v", rec)
	}
}

// A CACHE THAT CAN BREAK PLAYBACK IS WORSE THAN NO CACHE. Every way the disk can
// fail must fall through to the network, so this drives the two that a learner
// can actually produce: no store at all, and a fetch whose word is unknown.
func TestPlaybackSurvivesAnUnusableCache(t *testing.T) {
	cdn := newFakeCDN(t, map[string][]byte{"/a.mp3": []byte("ID3audio")})
	urls := cdn.urls("/a.mp3")

	for _, tc := range []struct {
		name string
		seam *audioSeam
		word string
	}{
		{"no store", newAudioSeam(newDiskAudioCache(nil, cdn.source())), "sycophantic"},
		{"no word to file under", newAudioSeam(newDiskAudioCache(store.NewMem(), cdn.source())), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, _, err := tc.seam.FetchFor(t.Context(), tc.word, urls)
			if err != nil || string(data) != "ID3audio" {
				t.Errorf("playback broke when the cache was unusable: %q, %v", data, err)
			}
		})
	}
}
