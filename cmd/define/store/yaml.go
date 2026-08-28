package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// YAML persists the deck as files under a directory:
//
//	words/<slug>.yaml        one file per word
//	events/YYYY-MM-DD.yaml   append-only, one file per day
//
// That layout does NOT make sync conflicts impossible — the same word, or the
// same day, touched on two machines still conflicts. It changes the rate: with a
// single vocab.yaml every write on the second machine conflicts, because every
// write touches the one file. Here a conflict needs a collision on one small,
// human-readable file.
type YAML struct {
	dir  string
	lang Lang
	warn io.Writer // where a skipped file is reported; nil silences it
}

// NewYAML returns a store rooted at dir, holding one language's deck.
//
// Both the directory and the language are parameters, not policies — who
// chooses them stays a one-line question at the boundary. In particular this
// store never reads the persisted setting itself: #23's whole design is that the
// language is decided ONCE, above here, and passed down. An empty lang means
// DefaultLang so a caller that predates languages cannot create a words//
// directory by omission.
func NewYAML(dir string, lang Lang, warn io.Writer) *YAML {
	if lang == "" {
		lang = DefaultLang
	}
	return &YAML{dir: dir, lang: lang, warn: warn}
}

// RuntimeDirs names every directory define writes into the working directory.
//
// ONE source, because a new one has to reach three places that cannot see each
// other: .gitignore, the index guard, and the history guard. #9 added usage/ and
// reached none of them — the deck-in-git class that has now cost four review
// rounds across three issues. TestGitignoreCoversRuntimeDirs closes the loop the
// compiler cannot: adding a name here and forgetting .gitignore fails a test.
var RuntimeDirs = []string{"words", "events", "usage"}

// RuntimeFiles names every FILE define writes into the working directory.
//
// The sibling of RuntimeDirs, and it exists because that list covers directories
// only. user-model.md is a runtime file — --reflect writes it into the current
// directory, carrying inferred claims about the learner — and it reached none of
// the three places RuntimeDirs was built to reach: `git check-ignore -v
// user-model.md` matched nothing. Nothing leaked, but #23's language setting
// would have been the second instance, which is why this is a list and not two
// more lines in .gitignore.
//
// A name here is RESERVED. .gitignore hides these un-anchored (go test runs in
// the package directory), so a tracked file sharing one is silently un-addable
// after any git rm — repo_guard_test.go's index guard is what says so out loud,
// and testdata/golden/user-model.md was renamed to clear the way for it.
//
// lang.txt, not lang: an un-anchored `lang` would also hide any DIRECTORY of
// that name anywhere in the tree, and a basename guard structurally cannot see
// that. The extension costs nothing and closes the hole.
var RuntimeFiles = []string{"user-model.md", "lang.txt"}

// wordsDir is per-language; eventsDir, usageDir and userModelFile are NOT.
//
// Language is a DECK dimension, not an event one. A review event names a word
// and a verdict; which deck it came from is the deck's business, and splitting
// the log would make "how much did I study today" a join — reached by migrating
// an append-only artifact, which is the worse half of the trade.
func (y *YAML) wordsDir() string  { return filepath.Join(y.dir, RuntimeDirs[0], string(y.lang)) }
func (y *YAML) eventsDir() string { return filepath.Join(y.dir, RuntimeDirs[1]) }

// userModelFile is the third artifact in the directory, beside words/ and
// events/. Markdown rather than YAML because a person edits it: #17 regenerates
// the inferred sections and never touches the human-owned ## Corrections.
//
// The name comes from RuntimeFiles, for the same reason langFile's does: that
// list is what .gitignore and both repo guards derive from, so a hand-written
// copy here would mean renaming the entry moved the guards while SetUserModel
// kept writing the old name — silently reopening the very hole RuntimeFiles was
// added to close.
func (y *YAML) userModelFile() string { return filepath.Join(y.dir, RuntimeFiles[0]) }

// usageDir holds the news cache, one file per word, beside words/ and events/.
func (y *YAML) usageDir() string { return filepath.Join(y.dir, RuntimeDirs[2]) }

// SetUserModel writes the learner model.
//
// Atomically, like a word file and unlike the append-only day log: this file is
// REPLACED wholesale, and a torn one would lose the human-owned corrections #17
// promises never to rewrite.
func (y *YAML) SetUserModel(text string) error {
	return writeBytesAtomic(y.userModelFile(), []byte(text))
}

// UserModel reads the learner model, or "" when there is none.
//
// Absent is not an error — it is the normal state until #17 first writes one —
// but an UNREADABLE file is, because silently answering "" for a model that
// exists would make every answer pitched at the wrong level with no way to tell.
func (y *YAML) UserModel() (string, error) {
	b, err := os.ReadFile(y.userModelFile())
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (y *YAML) Upsert(w Word) error {
	k := Key(w.Text)
	if k == "" {
		return nil
	}
	name, err := wordFileName(Slug(k))
	if err != nil {
		return err
	}
	path := filepath.Join(y.wordsDir(), name)

	old, err := readWord(path)
	if err != nil && !os.IsNotExist(err) {
		// A corrupt or unreadable entry must not block recording a new sighting —
		// but overwriting it destroys that word's FirstSeen and Lookups, so say
		// so. Deck warns on exactly this condition, and inside a synced directory
		// a transiently-unreadable file is a realistic input, not just a corrupt
		// one.
		y.warnf("overwriting unreadable %s: %v", filepath.Base(path), err)
		old = Word{}
	}
	return writeAtomic(path, merge(old, w))
}

func (y *YAML) Deck() ([]Word, error) {
	entries, err := os.ReadDir(y.wordsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Word
	for _, e := range entries {
		name := e.Name()
		// Skip the temp files an interrupted write leaves behind. Reading one
		// would surface a half-written entry as a real word.
		if e.IsDir() || !strings.HasSuffix(name, ".yaml") {
			continue
		}
		w, err := readWord(filepath.Join(y.wordsDir(), name))
		if err != nil {
			// One unreadable file must not make the whole deck unopenable.
			y.warnf("skipping %s: %v", name, err)
			continue
		}
		if w.Text == "" {
			y.warnf("skipping %s: no text field", name)
			continue
		}
		out = append(out, w)
	}
	sortDeck(out)
	return out, nil
}

func (y *YAML) AppendEvent(e ReviewEvent) error {
	if err := os.MkdirAll(y.eventsDir(), 0o755); err != nil {
		return err
	}
	// Day files are named in UTC so the grouping is stable across timezone
	// changes. The stored timestamp keeps its OFFSET, so a local-day view is
	// fully recoverable — #8 must group by the timestamp, never by the filename,
	// or an evening lookup west of UTC lands in "tomorrow".
	path := filepath.Join(y.eventsDir(), e.At.UTC().Format("2006-01-02")+".yaml")

	// Append rather than rewrite: the day log is the one file two machines are
	// most likely to touch on the same day, and an append conflict is resolvable
	// by keeping both sides.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	// If the last append was cut mid-record the file does not end in a newline,
	// and this write would land ON that fragment's line — so the parser would see
	// one malformed record and swallow THIS event along with the broken one. One
	// interrupted write must cost one event, not two.
	// The Stat error is ACTED ON rather than skipped past: a failure here would
	// silently bypass the torn-record guard below, which is the one thing this
	// block exists for. We are about to write to this handle, so a Stat we cannot
	// take means the write is not safe either.
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if st.Size() > 0 {
		if !endsWithNewline(path) {
			if _, err := f.WriteString("\n"); err != nil {
				return err
			}
		}
	}
	b, err := yaml.Marshal([]ReviewEvent{e})
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	return err
}

func (y *YAML) Events(since time.Time) ([]ReviewEvent, error) {
	entries, err := os.ReadDir(y.eventsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []ReviewEvent
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(y.eventsDir(), e.Name()))
		if err != nil {
			y.warnf("skipping %s: %v", e.Name(), err)
			continue
		}
		day, torn := parseDay(b)
		if torn > 0 {
			// A torn record, not a corrupt file: keep every whole record and drop
			// only the fragment.
			y.warnf("%s: recovered %d event(s), dropped %d torn record(s)", e.Name(), len(day), torn)
		}
		for _, ev := range day {
			if !ev.At.Before(since) {
				out = append(out, ev)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out, nil
}

func (y *YAML) warnf(format string, args ...any) {
	if y.warn != nil {
		fmt.Fprintf(y.warn, "define: "+format+"\n", args...)
	}
}

// wordFileName turns a slug into a filename, refusing anything that is not a
// single safe path element.
//
// Split out as a PURE function taking a slug rather than a key, so the guard is
// exercisable at its own level with hostile input. Inlined behind Slug it was
// unreachable in principle — Slug sanitises first, so no test could distinguish
// the guard from Slug, and removing it left every test green. Defence in depth
// that cannot be tested is decoration; this version is a real net under a future
// Slug regression.
func wordFileName(slug string) (string, error) {
	if slug == "" || slug == "." || slug == ".." || strings.ContainsAny(slug, `/\`) || strings.HasPrefix(slug, ".") {
		return "", fmt.Errorf("refusing unsafe word file name %q", slug)
	}
	return slug + ".yaml", nil
}

func readWord(path string) (Word, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Word{}, err
	}
	var w Word
	if err := yaml.Unmarshal(b, &w); err != nil {
		return Word{}, err
	}
	return w, nil
}

// writeAtomic writes to a temp file in the SAME directory and renames.
//
// This process is quit with Ctrl-C by design, so an interrupted write is routine
// rather than exceptional. A half-written YAML file would be a corrupted deck
// entry; a rename is atomic, so a reader sees the old file or the new one.
func writeAtomic(path string, w Word) error {
	b, err := yaml.Marshal(w)
	if err != nil {
		return err
	}
	return writeBytesAtomic(path, b)
}

// writeBytesAtomic is the atomic write itself, without the marshalling. Split
// out when user-model.md arrived: it is markdown a person edits rather than a
// serialised Word, and the alternative was a second temp-file-then-rename dance
// that could drift from this one (ARCH-DRY).
func writeBytesAtomic(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name) // no-op once renamed; cleans up on any failure below

	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// parseDay reads a day log, tolerating a torn final record.
//
// Events are APPENDED rather than written-and-renamed: two machines appending to
// the same day merge cleanly, which a whole-file rewrite would not. The cost is
// that an interrupted append can leave a partial record — and this process is
// quit with Ctrl-C by design, so that is routine rather than exceptional.
//
// ONE path, always record-by-record. An earlier version tried whole-file parsing
// first and fell back on error, which double-counted every record the failed
// parse had already collected, and left the fallback unreachable for any input
// that happened to remain valid YAML.
//
// Validity is TERMINATION plus COMPLETENESS, and neither alone is enough:
//
//   - "it parsed" is not enough — a cut leaves valid YAML ("- word: thi").
//   - "the fields are present" is not enough on its own — a cut inside the
//     timestamp can leave a shorter date that parses as a real one.
//   - byte-identical round-tripping WOULD catch both, but it also discards every
//     record in a log that was ever reformatted — a cosmetic edit or a sync tool
//     rewriting quotes would destroy the history it was meant to protect.
//
// So: a whole record ends with a newline (the writer always terminates one, and
// AppendEvent repairs a missing terminator before writing), and carries every
// field an event has. A truncation fails one or both, and reformatting fails
// neither.
func parseDay(b []byte) (events []ReviewEvent, torn int) {
	for _, rec := range splitRecords(string(b)) {
		if strings.TrimSpace(rec) == "" {
			continue
		}
		var one []ReviewEvent
		if err := yaml.Unmarshal([]byte(rec), &one); err != nil || len(one) != 1 {
			torn++
			continue
		}
		if !strings.HasSuffix(rec, "\n") || !one[0].complete() {
			torn++
			continue
		}
		events = append(events, one[0])
	}
	return events, torn
}

// splitRecords cuts a day log at the top-level "- " that begins each event.
//
// SplitAfter, not Split: the newline must stay ON the line it terminated. An
// earlier version split on "\n" and re-appended one to every line, which handed
// a terminator to the final fragment — destroying the exact signal parseDay uses
// to tell a whole record from a truncated one.
func splitRecords(s string) []string {
	var out []string
	var cur strings.Builder
	for _, line := range strings.SplitAfter(s, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "- ") && cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
		cur.WriteString(line)
	}
	if strings.TrimSpace(cur.String()) != "" {
		out = append(out, cur.String())
	}
	return out
}

func endsWithNewline(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true // unknown: do not inject a spurious newline
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.Size() == 0 {
		return true
	}
	var b [1]byte
	if _, err := f.ReadAt(b[:], st.Size()-1); err != nil {
		return true
	}
	return b[0] == '\n'
}

// Forget removes one word file. Events are untouched: the deck is a working set,
// the log is history.
//
// Filename derivation goes through wordFileName, the same function Upsert uses —
// see its doc comment for what that guard is and is not worth.
// newsFile is a WHOLE-FILE record, like words/ and unlike the append-only day
// log in events/. It is written through writeBytesAtomic and therefore cannot
// tear; the failure to handle is a file corrupted from outside, and the
// discipline for that shape here is warn and skip the whole file — never return
// half a record.
type newsFile struct {
	FetchedAt time.Time  `yaml:"fetched_at"`
	Items     []NewsItem `yaml:"items"`
}

func (y *YAML) NewsItems(key string) ([]NewsItem, time.Time, error) {
	k := Key(key)
	if k == "" {
		return nil, time.Time{}, nil
	}
	name, err := wordFileName(Slug(k))
	if err != nil {
		return nil, time.Time{}, err
	}
	b, err := os.ReadFile(filepath.Join(y.usageDir(), name))
	if os.IsNotExist(err) {
		return nil, time.Time{}, nil // never fetched
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	var f newsFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		// One corrupt cache file must not make the word unusable: reading as
		// never-fetched costs a re-fetch, which is exactly what a cache miss
		// costs anyway.
		y.warnf("skipping unreadable %s: %v", name, err)
		return nil, time.Time{}, nil
	}
	return f.Items, f.FetchedAt, nil
}

func (y *YAML) SetNewsItems(key string, items []NewsItem, at time.Time) error {
	k := Key(key)
	if k == "" {
		return nil
	}
	name, err := wordFileName(Slug(k))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(y.usageDir(), 0o755); err != nil {
		return err
	}
	b, err := yaml.Marshal(newsFile{FetchedAt: at, Items: items})
	if err != nil {
		return err
	}
	return writeBytesAtomic(filepath.Join(y.usageDir(), name), b)
}

func (y *YAML) Forget(key string) (bool, error) {
	k := Key(key)
	if k == "" {
		return false, nil
	}
	name, err := wordFileName(Slug(k))
	if err != nil {
		return false, err
	}
	err = os.Remove(filepath.Join(y.wordsDir(), name))
	switch {
	case err == nil:
		return true, nil
	case os.IsNotExist(err):
		return false, nil
	default:
		return false, err
	}
}
