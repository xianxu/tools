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
	warn io.Writer // where a skipped file is reported; nil silences it
}

// NewYAML returns a store rooted at dir. The directory is a parameter, not a
// policy — who chooses it stays a one-line question at the boundary.
func NewYAML(dir string, warn io.Writer) *YAML { return &YAML{dir: dir, warn: warn} }

func (y *YAML) wordsDir() string  { return filepath.Join(y.dir, "words") }
func (y *YAML) eventsDir() string { return filepath.Join(y.dir, "events") }

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
	if st, err := f.Stat(); err == nil && st.Size() > 0 {
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := yaml.Marshal(w)
	if err != nil {
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
