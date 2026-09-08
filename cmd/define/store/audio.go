package store

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// AudioKey identifies one cached recording.
//
// IT IS THE SEAM'S KEY, NOT THE WORD, and that distinction is the whole reason
// this type exists. A recording is fetched for a CANDIDATE LIST, which
// utterance.Candidates() builds from a source voice, a session voice and the
// word's spellings. Keying on the word alone would make `-locale gb` and
// `-locale us`, and `-pron en red` beside a plain `red`, all one entry — a cache
// that serves the wrong recording, which is worse than no cache at all.
//
// The word survives as the FILING prefix so Forget can still find everything a
// word owns. Identity and filing are different jobs and this carries both.
type AudioKey struct {
	// Word is what the file is filed under. Normalised the way every other
	// per-word file is.
	Word string
	// Digest is over the candidate list exactly as the fetch seam received it.
	Digest string
}

// NewAudioKey derives the key from the candidate URLs a fetch was given.
//
// DERIVED FROM THE LIST, never rebuilt from voice fields. `spokeSource` states
// the same rule one package over — "by MEMBERSHIP in the list actually built,
// never by parsing the URL" — because a key reconstructed from the parts is a
// second, driftable statement of what identifies a recording, and it goes wrong
// the day the CDN generation moves.
//
// Truncated to 16 hex characters: this is a cache key, not a security boundary,
// and a collision costs one wrong recording for one voice of one word. Sixteen
// characters is 64 bits, which for a few thousand entries is not a risk anyone
// will meet.
func NewAudioKey(word string, urls []string) AudioKey {
	sum := sha256.Sum256([]byte(strings.Join(urls, "\n")))
	return AudioKey{Word: Key(word), Digest: hex.EncodeToString(sum[:])[:16]}
}

// dir is the word's own directory; stem is the file inside it.
//
// A DIRECTORY PER WORD, not a filename prefix — and the boundary review is why.
// The first version filed recordings as `<slug>--<digest>` and claimed a slug
// could not contain "--". That is false: `re-` slugs to `re--ddf427`, so
// forgetting `re` would glob `re--` and take a DIFFERENT WORD's recordings.
//
// The fix is not a rarer separator. Any separator has to be reasoned about
// against Slug's alphabet, and that reasoning is what was wrong. A directory
// name is an EXACT match — `audio/re` and `audio/re--ddf427` are simply
// different directories — so Forget removes a tree by name and no prefix
// ambiguity exists to get wrong.
func (k AudioKey) dir() string  { return Slug(k.Word) }
func (k AudioKey) stem() string { return k.Digest }

// ok reports whether the key is usable as a filename — BOTH halves of it.
//
// EVERY FIELD IS VALIDATED BY THE PREDICATE ITS USE REQUIRES. Word is laundered
// through Slug, which is what safeElement exists for; Digest reached
// filepath.Join VERBATIM and was checked only for being non-empty, while this
// comment already claimed "usable as a filename". Probe-verified: a Digest of
// "../../../../pwned" wrote pwned.mp3 four levels above the store root and
// SetAudio returned nil.
//
// Not reachable from production — NewAudioKey yields hex — but AudioKey is
// EXPORTED with exported fields and SetAudio is on the exported Store interface,
// so "the only caller is careful" is a property of today's callers rather than of
// the type. ARCH-SECURE: a value crossing into a path is untrusted even when this
// program produced it.
func (k AudioKey) ok() bool {
	// AN EMPTY WORD IS REFUSED even though Slug("") yields the legal filename
	// "w-<hash>". The predicate a key's USE requires includes "Forget can reach
	// it", and Forget("") returns early — so a recording filed under one would
	// be material no forget could take, which is #10's BR-45 arriving by a new
	// path. Found by the guard's own row rather than by review.
	return Key(k.Word) != "" && safeElement(Slug(k.Word)) && safeElement(k.Digest)
}

// AudioRecord is what is stored beside the bytes: where they came from, when,
// and whether there were any.
//
// ONE SHAPE FOR A HIT AND A MISS. A record always exists; a hit names its From
// and has a blob beside it, a miss has neither. That is one artifact to read,
// one to write and one for Forget to remove — where a `.mp3`/`.none` pair would
// be two shapes free to drift apart.
type AudioRecord struct {
	// From is the URL that actually answered.
	//
	// LOAD-BEARING, not provenance decoration. utterance.spokeSource(from)
	// decides by membership in the source candidate list, and reportVoice prints
	// a RECORD from that decision — one which, as its own comment says, "survives
	// on a pipe and cannot be taken back". A cache hit that cannot say which URL
	// answered makes that record silent or false, so a cached recording is bytes
	// AND provenance or it is not a cache of this seam.
	From string `yaml:"from,omitempty"`
	// At is the day the fetch happened, and it is what makes a miss expire.
	At time.Time `yaml:"at"`
	// Missing records that no candidate carried a recording. A verdict, not an
	// error: the fetch succeeded in establishing there is nothing there.
	Missing bool `yaml:"missing,omitempty"`
}

// AudioVerdictTTL is how long "the CDN has no recording for this" is believed.
//
// NOT FOREVER, which is what an in-memory memo can afford and a durable one
// cannot. The CDN gains recordings over time, and the words most likely to gain
// one are the rare words a learner is most likely to want — so a permanent
// verdict is a word that can never start working. Thirty days costs four
// candidate requests a month per unrecorded word and bounds the staleness.
//
// A HIT never expires. Bytes that answered once are still the right bytes: the
// key is the candidate list, so a changed URL is a different key rather than a
// stale one.
const AudioVerdictTTL = 30 * 24 * time.Hour

// Fresh reports whether a record may still be believed, as of now.
func (r AudioRecord) Fresh(now time.Time) bool {
	if !r.Missing {
		return true
	}
	return now.Sub(r.At) < AudioVerdictTTL
}
