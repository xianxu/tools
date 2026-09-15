package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// helpEntriesEqual compares by value, with At compared by instant: a UTC time
// round-trips through JSON as an equal instant, and that is the equality a
// cache consumer cares about.
func helpEntriesEqual(a, b []HelpEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Lang != b[i].Lang || a[i].Mode != b[i].Mode || a[i].Source != b[i].Source ||
			a[i].English != b[i].English || !a[i].At.Equal(b[i].At) {
			return false
		}
	}
	return true
}

// numberedHelpEntries builds n distinct entries whose At rises with index, so
// "newest" and "later slice position" coincide unless a test says otherwise.
func numberedHelpEntries(n int, source func(i int) string) []HelpEntry {
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	out := make([]HelpEntry, n)
	for i := range out {
		out[i] = HelpEntry{
			Lang:    "es",
			Mode:    "gloss",
			Source:  source(i),
			English: fmt.Sprintf("english %d", i),
			At:      base.Add(time.Duration(i) * time.Second),
		}
	}
	return out
}

func firstSource(entries []HelpEntry) string {
	if len(entries) == 0 {
		return ""
	}
	return entries[0].Source
}

func TestPracticeHelpCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if got := ReadPracticeHelp(dir); got != nil {
		t.Fatalf("empty dir read %v, want nil", got)
	}

	first := []HelpEntry{
		{Lang: "es", Mode: "gloss", Source: "el perro corre", English: "the dog runs", At: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)},
		{Lang: "es", Mode: "cloze", Source: "la ___ es azul", English: "the ___ is blue", At: time.Date(2026, 9, 1, 12, 0, 1, 0, time.UTC)},
	}
	if err := WritePracticeHelp(dir, first); err != nil {
		t.Fatal(err)
	}
	if got := ReadPracticeHelp(dir); !helpEntriesEqual(got, first) {
		t.Fatalf("read back %+v, want %+v", got, first)
	}

	// A second write REPLACES: the cache is a snapshot, not a log.
	second := []HelpEntry{
		{Lang: "es", Mode: "gloss", Source: "buenos días", English: "good morning", At: time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC)},
	}
	if err := WritePracticeHelp(dir, second); err != nil {
		t.Fatal(err)
	}
	if got := ReadPracticeHelp(dir); !helpEntriesEqual(got, second) {
		t.Fatalf("after replacement read %+v, want %+v", got, second)
	}

	// The on-disk shape is the documented one, and the write left no shadow.
	b, err := os.ReadFile(filepath.Join(dir, PracticeHelpFileName()))
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Version int               `json:"version"`
		Entries []json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(b, &file); err != nil {
		t.Fatalf("saved file is not JSON: %v\n%s", err, b)
	}
	if file.Version != 1 || len(file.Entries) != 1 {
		t.Fatalf("saved version=%d entries=%d, want version 1 with one entry", file.Version, len(file.Entries))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != PracticeHelpFileName() {
		t.Fatalf("writes left %v, want only %s", entries, PracticeHelpFileName())
	}
}

func TestPracticeHelpCacheBounds(t *testing.T) {
	t.Run("newest 512 by At", func(t *testing.T) {
		dir := t.TempDir()
		all := numberedHelpEntries(513, func(i int) string { return fmt.Sprintf("source %d", i) })
		// Present them out of order so the bound is by At, not by position.
		shuffled := append([]HelpEntry(nil), all[256:]...)
		shuffled = append(shuffled, all[:256]...)
		if err := WritePracticeHelp(dir, shuffled); err != nil {
			t.Fatal(err)
		}
		if got := ReadPracticeHelp(dir); !helpEntriesEqual(got, all[1:]) {
			t.Fatalf("kept %d entries starting %q, want the 512 newest starting %q", len(got), firstSource(got), all[1].Source)
		}
		// The caller's slice is its own: bounding must not reorder it.
		if shuffled[0].Source != all[256].Source {
			t.Fatalf("write reordered the caller's slice: first is now %q", shuffled[0].Source)
		}
	})

	t.Run("ties keep later slice position", func(t *testing.T) {
		dir := t.TempDir()
		all := numberedHelpEntries(513, func(i int) string { return fmt.Sprintf("source %d", i) })
		for i := range all {
			all[i].At = all[0].At
		}
		if err := WritePracticeHelp(dir, all); err != nil {
			t.Fatal(err)
		}
		if got := ReadPracticeHelp(dir); !helpEntriesEqual(got, all[1:]) {
			t.Fatalf("kept %d entries starting %q, want the later 512 starting %q", len(got), firstSource(got), all[1].Source)
		}
	})

	t.Run("unreadable entries spend no budget", func(t *testing.T) {
		dir := t.TempDir()
		valid := numberedHelpEntries(512, func(i int) string { return fmt.Sprintf("source %d", i) })
		// Newer than every valid entry, so it would claim a slot first — and a
		// read would then drop it, leaving 511.
		junk := HelpEntry{Lang: "es", Mode: "gloss", Source: "no translation", At: valid[511].At.Add(time.Hour)}
		if err := WritePracticeHelp(dir, append(append([]HelpEntry(nil), valid...), junk)); err != nil {
			t.Fatal(err)
		}
		if got := ReadPracticeHelp(dir); !helpEntriesEqual(got, valid) {
			t.Fatalf("kept %d entries starting %q, want all 512 valid ones", len(got), firstSource(got))
		}
	})

	t.Run("oversize write drops oldest until it fits", func(t *testing.T) {
		dir := t.TempDir()
		const perEntry = 100 * 1024
		all := numberedHelpEntries(20, func(i int) string { return strings.Repeat(string(rune('a'+i)), perEntry) })
		if err := WritePracticeHelp(dir, all); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(filepath.Join(dir, PracticeHelpFileName()))
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() > 1<<20 {
			t.Fatalf("saved %d bytes, want <= 1 MiB", info.Size())
		}
		got := ReadPracticeHelp(dir)
		if len(got) == 0 || len(got) >= len(all) {
			t.Fatalf("kept %d of %d entries; the write should have dropped some but not all", len(got), len(all))
		}
		// The survivors are the newest suffix, in order.
		if !helpEntriesEqual(got, all[len(all)-len(got):]) {
			t.Fatalf("kept entries are not the newest suffix: first kept %q", firstSource(got))
		}
		// And only as many were dropped as needed: one more entry would not fit.
		if info.Size()+perEntry <= 1<<20 {
			t.Fatalf("saved %d bytes with %d dropped; one more %d-byte entry would still have fit", info.Size(), len(all)-len(got), perEntry)
		}
	})

	// The byte budget is ARITHMETIC over separately encoded entries, so its
	// edge is pinned exactly: an off-by-one there writes a file one byte over
	// the cap, which the next read discards whole. The layout is spelled out
	// here rather than taken from the writer, so it is an independent statement
	// of the documented format.
	t.Run("exactly 1 MiB fits and one byte more drops the oldest", func(t *testing.T) {
		const layout = `{"version":1,"entries":[` +
			`{"lang":"es","mode":"gloss","source":"%s","english":"o","at":"2026-09-01T12:00:00Z"},` +
			`{"lang":"es","mode":"cloze","source":"n","english":"n","at":"2026-09-01T12:00:01Z"}]}`
		fill := 1<<20 - len(fmt.Sprintf(layout, ""))
		newest := HelpEntry{Lang: "es", Mode: "cloze", Source: "n", English: "n", At: time.Date(2026, 9, 1, 12, 0, 1, 0, time.UTC)}
		oldest := func(n int) HelpEntry {
			return HelpEntry{Lang: "es", Mode: "gloss", Source: strings.Repeat("x", n), English: "o", At: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)}
		}

		dir := t.TempDir()
		if err := WritePracticeHelp(dir, []HelpEntry{oldest(fill), newest}); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(dir, PracticeHelpFileName()))
		if err != nil {
			t.Fatal(err)
		}
		if want := fmt.Sprintf(layout, strings.Repeat("x", fill)); string(b) != want {
			t.Fatalf("saved %d bytes, want the documented %d-byte layout", len(b), len(want))
		}
		if got := ReadPracticeHelp(dir); !helpEntriesEqual(got, []HelpEntry{oldest(fill), newest}) {
			t.Fatalf("an exactly-1-MiB cache read back %d entries, want both", len(got))
		}

		dir = t.TempDir()
		if err := WritePracticeHelp(dir, []HelpEntry{oldest(fill + 1), newest}); err != nil {
			t.Fatal(err)
		}
		if got := ReadPracticeHelp(dir); !helpEntriesEqual(got, []HelpEntry{newest}) {
			t.Fatalf("a cache one byte over read back %d entries starting %.10q, want only the newest", len(got), firstSource(got))
		}
	})

	// ONE byte over is the row that matters: a file any larger is also cut short
	// by the bounded read and fails to parse, so only a VALID file at exactly
	// cap+1 shows the cap itself is enforced.
	t.Run("oversize file on disk reads as nil", func(t *testing.T) {
		const layout = `{"version":1,"entries":[{"lang":"es","mode":"gloss","source":"%s","english":"y","at":"2026-09-01T12:00:00Z"}]}`
		for _, over := range []int{1, 100 * 1024} {
			body := fmt.Sprintf(layout, strings.Repeat("x", 1<<20+over-len(fmt.Sprintf(layout, ""))))
			if len(body) != 1<<20+over || !json.Valid([]byte(body)) {
				t.Fatalf("fixture is %d bytes (valid=%v), want valid JSON of %d", len(body), json.Valid([]byte(body)), 1<<20+over)
			}
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, PracticeHelpFileName()), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := ReadPracticeHelp(dir); got != nil {
				t.Fatalf("a valid file %d byte(s) over the cap read %d entries, want nil", over, len(got))
			}
		}
	})

	t.Run("wrong version reads as nil", func(t *testing.T) {
		for _, body := range []string{
			`{"version":2,"entries":[{"lang":"es","mode":"gloss","source":"a","english":"b","at":"2026-09-01T12:00:00Z"}]}`,
			`{"version":0,"entries":[{"lang":"es","mode":"gloss","source":"a","english":"b","at":"2026-09-01T12:00:00Z"}]}`,
			`{"entries":[{"lang":"es","mode":"gloss","source":"a","english":"b","at":"2026-09-01T12:00:00Z"}]}`,
		} {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, PracticeHelpFileName()), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := ReadPracticeHelp(dir); got != nil {
				t.Fatalf("%s read %d entries, want nil", body, len(got))
			}
		}
	})

	// Empty fields, plus a Lang that is not a language tag: Lang is a path
	// segment elsewhere in this package, and a hand-edited "../x" decoded
	// straight into one would be exactly the traversal ParseLang exists to stop.
	t.Run("incomplete or invalid entries dropped on read", func(t *testing.T) {
		dir := t.TempDir()
		body := `{"version":1,"entries":[` +
			`{"lang":"","mode":"gloss","source":"a","english":"b","at":"2026-09-01T12:00:00Z"},` +
			`{"lang":"es","mode":"","source":"a","english":"b","at":"2026-09-01T12:00:00Z"},` +
			`{"lang":"es","mode":"gloss","source":"","english":"b","at":"2026-09-01T12:00:00Z"},` +
			`{"lang":"es","mode":"gloss","source":"a","english":"","at":"2026-09-01T12:00:00Z"},` +
			`{"lang":"../x","mode":"gloss","source":"a","english":"b","at":"2026-09-01T12:00:00Z"},` +
			`{"lang":"spa","mode":"gloss","source":"a","english":"b","at":"2026-09-01T12:00:00Z"},` +
			`null,` +
			`{"lang":"es","mode":"gloss","source":"keep","english":"kept","at":"2026-09-01T12:00:00Z"}` +
			`]}`
		if err := os.WriteFile(filepath.Join(dir, PracticeHelpFileName()), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		want := []HelpEntry{{Lang: "es", Mode: "gloss", Source: "keep", English: "kept", At: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)}}
		if got := ReadPracticeHelp(dir); !helpEntriesEqual(got, want) {
			t.Fatalf("read %+v, want only the complete entry", got)
		}
	})

	t.Run("corrupt file reads as nil", func(t *testing.T) {
		for _, body := range []string{"", "{", "[]", "null", `{"version":1,"entries":{}}`, `{"version":"1","entries":[]}`, "\x00\xff"} {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, PracticeHelpFileName()), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := ReadPracticeHelp(dir); got != nil {
				t.Fatalf("%q read %d entries, want nil", body, len(got))
			}
		}
		// Unreadable, not merely malformed: a directory where the file should be.
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, PracticeHelpFileName()), 0o700); err != nil {
			t.Fatal(err)
		}
		if got := ReadPracticeHelp(dir); got != nil {
			t.Fatalf("directory at the cache path read %d entries, want nil", len(got))
		}
	})

	t.Run("failed write returns an error and keeps the previous cache", func(t *testing.T) {
		existing := []HelpEntry{{Lang: "es", Mode: "gloss", Source: "a", English: "b", At: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)}}
		replacement := HelpEntry{Lang: "es", Mode: "cloze", Source: "c", English: "d", At: time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)}

		// A regular file where the deck directory should be: nothing can be
		// created under it, whatever the user's privileges.
		notADir := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(notADir, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := WritePracticeHelp(notADir, []HelpEntry{replacement}); err == nil {
			t.Fatal("write under a regular file succeeded")
		}

		// A directory AT the destination makes the final rename fail, again
		// regardless of privileges — bilingual_test.go's technique. A valid cache
		// saved inside it is what detects damage, and the listing detects debris.
		dir := t.TempDir()
		destination := filepath.Join(dir, PracticeHelpFileName())
		if err := WritePracticeHelp(destination, existing); err != nil {
			t.Fatal(err)
		}
		if err := WritePracticeHelp(dir, []HelpEntry{replacement}); err == nil {
			t.Fatal("replacement of a nonempty directory succeeded")
		}
		if got := ReadPracticeHelp(destination); !helpEntriesEqual(got, existing) {
			t.Fatalf("failed replacement damaged the existing destination: %+v", got)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != PracticeHelpFileName() || !entries[0].IsDir() {
			t.Fatalf("failed replacement left temporary files or changed destination: %v", entries)
		}

		// And with a real cache FILE at the destination: an entry the encoder
		// refuses (a year JSON timestamps cannot carry) fails the replacement,
		// and the saved cache is byte-for-byte what it was.
		dir = t.TempDir()
		if err := WritePracticeHelp(dir, existing); err != nil {
			t.Fatal(err)
		}
		before, err := os.ReadFile(filepath.Join(dir, PracticeHelpFileName()))
		if err != nil {
			t.Fatal(err)
		}
		unencodable := HelpEntry{Lang: "es", Mode: "gloss", Source: "e", English: "f", At: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)}
		if err := WritePracticeHelp(dir, []HelpEntry{replacement, unencodable}); err == nil {
			t.Fatal("a replacement the encoder refused reported success")
		}
		after, err := os.ReadFile(filepath.Join(dir, PracticeHelpFileName()))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, after) {
			t.Fatalf("failed replacement changed the saved cache:\nbefore %s\nafter  %s", before, after)
		}
		if got := ReadPracticeHelp(dir); !helpEntriesEqual(got, existing) {
			t.Fatalf("failed replacement left the cache unreadable: %+v", got)
		}
	})
}

func FuzzPracticeHelpCacheRead(f *testing.F) {
	valid := `{"version":1,"entries":[{"lang":"es","mode":"gloss","source":"el perro","english":"the dog","at":"2026-09-01T12:00:00Z"}]}`
	for _, s := range []string{
		valid,
		valid[:len(valid)/2],
		valid[:len(valid)-2],
		`{"version":1,"entries":[{"lang":1,"mode":true,"source":null,"english":[],"at":"x"}]}`,
		`{"version":1,"entries":[{"lang":"es","mode":"gloss","source":"a","english":"b","at":17}]}`,
		`{"version":1e999,"entries":[]}`,
		`{"version":99999999999999999999999,"entries":[]}`,
		`{"version":-1,"entries":[]}`,
		`{"version":1,"entries":[{"lang":"es","mode":"gloss","source":"a","english":"b","at":"9999999999-01-01T00:00:00Z"}]}`,
		`{"version":1,"entries":[{"lang":"es","mode":"gloss","source":"","english":"b","at":"2026-09-01T12:00:00Z"}]}`,
		`{"version":1,"entries":[{"lang":"../x","mode":"gloss","source":"a","english":"b","at":"2026-09-01T12:00:00Z"}]}`,
		`{"version":1,"entries":[null,{}]}`,
		"", "null", "[]", "{", "\x00",
	} {
		f.Add(s)
	}
	// The decoder ReadPracticeHelp hands its bounded read to, as
	// FuzzBilingualSetting fuzzes its parser: the file around it is a few
	// syscalls the unit tests above cover branch by branch.
	f.Fuzz(func(t *testing.T, s string) {
		got := parsePracticeHelp([]byte(s))
		if !json.Valid([]byte(s)) && got != nil {
			t.Fatalf("invalid JSON read %d entries", len(got))
		}
		if len(s) > 1<<20 && got != nil {
			t.Fatalf("a %d-byte input read %d entries; the cap is 1 MiB", len(s), len(got))
		}
		for _, e := range got {
			if e.Lang == "" || e.Mode == "" || e.Source == "" || e.English == "" {
				t.Fatalf("read returned an incomplete entry %+v", e)
			}
			if l, err := ParseLang(string(e.Lang)); err != nil || l != e.Lang {
				t.Fatalf("read returned Lang %q, which is not a canonical language tag", e.Lang)
			}
		}
	})
}
