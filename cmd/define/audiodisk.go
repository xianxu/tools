package main

import (
	"context"
	"errors"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// diskAudioCache is an AudioSource that reads and writes recordings in the
// working directory.
//
// A DECORATOR ON THE SEAM, like the memo above it, and for the reason the memo's
// own comment gives: the existing fakeCDN request recorder is then the assertion
// that a replay costs no second request, with no bespoke scaffolding. What it
// adds is the only thing the memo cannot — surviving the process.
//
// THE LAYERING, stated once because two things depend on it: the memo is
// OUTERMOST, this is beneath it, and the network is innermost. A repeat within
// one sitting must not touch the filesystem, and a repeat across sittings must
// not touch the network. Production wires `newAudioSeam(newDiskAudioCache(st,
// http))`; every test that means to exercise this wires it the same way, because
// "the line the tests exercise is the line production runs" (#2 I-1) is the
// argument this file leans on twice.
type diskAudioCache struct {
	st    store.Store
	inner AudioSource
	// word is what the recording is FILED under. The seam is handed URLs, not a
	// word, so the caller that knows the word supplies it — see fetchFor.
	word string
	// now is injectable so the verdict TTL is testable without sleeping.
	now func() time.Time
}

func newDiskAudioCache(st store.Store, inner AudioSource) *diskAudioCache {
	return &diskAudioCache{st: st, inner: inner, now: time.Now}
}

// forWord returns a view of the cache filed under a word.
//
// The AudioSource seam takes URLs alone, which is right — AudioCandidates is
// pure and the walk order is what the fake asserts against. But a cache that
// Forget must be able to reach needs the word too, and inventing it by parsing a
// URL would be the second, driftable statement of identity that AudioKey's doc
// comment refuses. So the caller that HAS the word says so.
//
// A cache with no word still works and simply stores nothing: a fetch whose word
// is unknown is a fetch nobody can forget, and storing it would leak.
func (c *diskAudioCache) forWord(word string) AudioSource {
	if c == nil {
		return nil
	}
	dup := *c
	dup.word = word
	return &dup
}

// A CAPABILITY ASKED FOR BY TYPE ASSERTION MUST BE ASSERTED AT COMPILE TIME.
//
// Without this line the first version of forWord returned *diskAudioCache rather
// than AudioSource — a signature Go accepts everywhere except as an
// implementation of wordFiler. The assertion in FetchFor simply never matched,
// so every fetch silently bypassed the disk and the cache did nothing at all. It
// compiled, it ran, and only the request-count assertions caught it.
var _ wordFiler = (*diskAudioCache)(nil)

// Fetch reads through to the disk, then to the source.
//
// DEGRADES, NEVER FAILS. A store error, an unwritable directory, a corrupt
// record — every one of them falls through to the network rather than
// propagating. A cache that can break playback is worse than no cache, and this
// is the one place that property is decided.
func (c *diskAudioCache) Fetch(ctx context.Context, urls []string) ([]byte, string, error) {
	if c == nil || c.inner == nil {
		return nil, "", ErrNoAudio
	}
	k, cacheable := c.keyFor(urls)
	if cacheable {
		if data, rec, err := c.st.Audio(k); err == nil && !rec.At.IsZero() && rec.Fresh(c.now()) {
			if rec.Missing {
				return nil, "", ErrNoAudio
			}
			return data, rec.From, nil
		}
	}

	data, from, err := c.inner.Fetch(ctx, urls)
	if err != nil {
		// ONLY ErrNoAudio IS RECORDED. A transport failure is transient — the
		// taxonomy in fetch.go is the single source of that distinction, and
		// writing "this word has no recording" during an outage would poison the
		// word for a month.
		if cacheable && errors.Is(err, ErrNoAudio) {
			c.put(k, nil, store.AudioRecord{At: c.now(), Missing: true})
		}
		return nil, "", err
	}
	if cacheable {
		c.put(k, data, store.AudioRecord{From: from, At: c.now()})
	}
	return data, from, nil
}

// keyFor builds the store key, and says when there is nothing to file under.
func (c *diskAudioCache) keyFor(urls []string) (store.AudioKey, bool) {
	if c.st == nil || c.word == "" || len(urls) == 0 {
		return store.AudioKey{}, false
	}
	k := store.NewAudioKey(c.word, urls)
	return k, k.Word != ""
}

// put writes, and swallows. A failed write means the next play refetches, which
// is exactly what no cache at all would have cost.
func (c *diskAudioCache) put(k store.AudioKey, data []byte, rec store.AudioRecord) {
	_ = c.st.SetAudio(k, data, rec)
}
