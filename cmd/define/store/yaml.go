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
	path := filepath.Join(y.wordsDir(), Slug(k)+".yaml")

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
		var day []ReviewEvent
		if err := yaml.Unmarshal(b, &day); err != nil {
			y.warnf("skipping %s: %v", e.Name(), err)
			continue
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
