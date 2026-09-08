package store

import "testing"

// EVERY FIELD THAT REACHES A PATH IS VALIDATED BY WHAT THE PATH REQUIRES.
//
// Probe-verified before the fix: SetAudio with a Digest of "../../../../pwned"
// wrote pwned.mp3 and pwned.yaml four levels above the store root and returned
// nil. Word was laundered through Slug; Digest reached filepath.Join verbatim
// while ok()'s own doc said "usable as a filename".
//
// Not reachable from production — NewAudioKey yields hex — but AudioKey is
// EXPORTED with exported fields and SetAudio sits on the exported Store
// interface, so "the only caller is careful" describes today's callers rather
// than the type.
func TestAnAudioKeyCannotEscapeItsDirectory(t *testing.T) {
	for _, tc := range []struct {
		name string
		k    AudioKey
	}{
		{"a traversing digest", AudioKey{Word: "keel", Digest: "../../../../pwned"}},
		{"a separator in the digest", AudioKey{Word: "keel", Digest: "a/b"}},
		{"a dotfile digest", AudioKey{Word: "keel", Digest: ".hidden"}},
		{"a dot digest", AudioKey{Word: "keel", Digest: ".."}},
		{"an empty digest", AudioKey{Word: "keel", Digest: ""}},
		{"an empty word", AudioKey{Word: "", Digest: "abc"}},
	} {
		if tc.k.ok() {
			t.Errorf("%s: %+v was accepted as a filename", tc.name, tc.k)
		}
	}
	// AND A REAL KEY STILL WORKS, so this cannot pass by refusing everything.
	if k := NewAudioKey("keel", []string{"https://cdn/keel.mp3"}); !k.ok() {
		t.Errorf("a real key was refused: %+v", k)
	}

	// A DANGEROUS WORD IS LAUNDERED, NOT REFUSED, and the asymmetry with Digest
	// is the whole point of the class. Slug exists to turn any key into exactly
	// one safe path element, and Forget computes the same Slug — so the file is
	// both safe and reachable. Digest has no such function, which is why it
	// needed a predicate rather than a laundering.
	//
	// This row was a WRONG ASSERTION first: it demanded a traversing word be
	// refused, which would have broken a real deck entry for a rule that does not
	// apply to it.
	k := AudioKey{Word: "../../etc/passwd", Digest: "abc"}
	if !k.ok() {
		t.Errorf("a laundered word was refused: Slug gives %q", Slug(Key(k.Word)))
	}
	if !safeElement(Slug(Key(k.Word))) {
		t.Errorf("Slug did not launder %q into a safe element: %q", k.Word, Slug(Key(k.Word)))
	}
}
