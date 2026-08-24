package store_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/cmd/define/store/storetest"
)

// The deliverable: the same suite passes for both implementations. That is what
// makes the in-memory store a reference rather than an alibi.
func TestYAMLConformance(t *testing.T) {
	storetest.Suite(t, func(t *testing.T) store.Store {
		return store.NewYAML(t.TempDir(), nil)
	})
}

func TestYAMLPersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	s1 := store.NewYAML(dir, nil)
	if err := s1.Upsert(store.Word{Text: "sycophantic", LastSeen: time.Now()}); err != nil {
		t.Fatal(err)
	}
	// A second store on the same directory is what a restart looks like.
	deck, err := store.NewYAML(dir, nil).Deck()
	if err != nil {
		t.Fatal(err)
	}
	if len(deck) != 1 || deck[0].Text != "sycophantic" {
		t.Errorf("reopened deck = %+v", deck)
	}
}

// An interrupted write leaves a temp file. It must never be read as a word.
func TestYAMLIgnoresInterruptedWrites(t *testing.T) {
	dir := t.TempDir()
	s := store.NewYAML(dir, nil)
	_ = s.Upsert(store.Word{Text: "good", LastSeen: time.Now()})

	partial := filepath.Join(dir, "words", ".tmp-halfwritten")
	if err := os.WriteFile(partial, []byte("text: bad\nlast_seen: not-a-ti"), 0o644); err != nil {
		t.Fatal(err)
	}
	deck, err := s.Deck()
	if err != nil {
		t.Fatalf("a leftover temp file broke the deck: %v", err)
	}
	if len(deck) != 1 || deck[0].Text != "good" {
		t.Errorf("deck = %+v, want only the completed write", deck)
	}
}

// One corrupt file must not make the whole deck unopenable — that would lose
// every word to a single bad byte.
func TestYAMLSkipsCorruptFileWithWarning(t *testing.T) {
	dir := t.TempDir()
	var warn bytesBuffer
	s := store.NewYAML(dir, &warn)
	_ = s.Upsert(store.Word{Text: "good", LastSeen: time.Now()})

	if err := os.WriteFile(filepath.Join(dir, "words", "broken.yaml"), []byte("\t: [unclosed"), 0o644); err != nil {
		t.Fatal(err)
	}
	deck, err := s.Deck()
	if err != nil {
		t.Fatalf("Deck failed on one corrupt file: %v", err)
	}
	if len(deck) != 1 || deck[0].Text != "good" {
		t.Errorf("deck = %+v, want the readable word", deck)
	}
	if warn.String() == "" {
		t.Error("a skipped file must be reported, not silently dropped")
	}
}

// Two events on the same day share one file, in order.
func TestYAMLGroupsEventsByDay(t *testing.T) {
	dir := t.TempDir()
	s := store.NewYAML(dir, nil)
	day := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	_ = s.AppendEvent(store.ReviewEvent{Word: "a", Kind: store.EventLookedUp, At: day})
	_ = s.AppendEvent(store.ReviewEvent{Word: "b", Kind: store.EventLookedUp, At: day.Add(time.Hour)})
	_ = s.AppendEvent(store.ReviewEvent{Word: "c", Kind: store.EventLookedUp, At: day.AddDate(0, 0, 1)})

	files, _ := os.ReadDir(filepath.Join(dir, "events"))
	if len(files) != 2 {
		t.Errorf("got %d day files, want 2: %v", len(files), files)
	}
	ev, _ := s.Events(time.Time{})
	if len(ev) != 3 || ev[0].Word != "a" || ev[2].Word != "c" {
		t.Errorf("events = %+v", ev)
	}
}

// The property the layout is actually chosen for: two stores writing DIFFERENT
// words to one directory touch disjoint files, so a sync has nothing to merge.
func TestYAMLDifferentWordsTouchDisjointFiles(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	_ = store.NewYAML(dir, nil).Upsert(store.Word{Text: "alpha", LastSeen: now})
	before, _ := os.ReadDir(filepath.Join(dir, "words"))

	_ = store.NewYAML(dir, nil).Upsert(store.Word{Text: "beta", LastSeen: now})
	after, _ := os.ReadDir(filepath.Join(dir, "words"))

	if len(after) != len(before)+1 {
		t.Errorf("second word changed %d files, want exactly 1 new", len(after)-len(before))
	}
}

type bytesBuffer struct{ b []byte }

func (w *bytesBuffer) Write(p []byte) (int, error) { w.b = append(w.b, p...); return len(p), nil }
func (w *bytesBuffer) String() string              { return string(w.b) }

// Events are appended, not renamed, so an interrupted write can leave a torn
// record. Losing that record is acceptable; losing the whole day is not.
func TestYAMLRecoversFromATornEventRecord(t *testing.T) {
	dir := t.TempDir()
	var warn bytesBuffer
	s := store.NewYAML(dir, &warn)
	day := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	_ = s.AppendEvent(store.ReviewEvent{Word: "first", Kind: store.EventLookedUp, At: day})
	_ = s.AppendEvent(store.ReviewEvent{Word: "second", Kind: store.EventLookedUp, At: day.Add(time.Hour)})

	// Simulate a kill mid-append: a partial record on the end.
	path := filepath.Join(dir, "events", "2026-08-20.yaml")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("- word: thi")
	f.Close()

	ev, err := s.Events(time.Time{})
	if err != nil {
		t.Fatalf("a torn record broke the day: %v", err)
	}
	if len(ev) != 2 || ev[0].Word != "first" || ev[1].Word != "second" {
		t.Errorf("events = %+v, want the two whole records preserved", ev)
	}
	if warn.String() == "" {
		t.Error("a dropped record must be reported")
	}
}

// A whole asked record survives beside the lookups, so the torn cases below are
// asserting truncation rather than an event kind the reader cannot parse.
func TestYAMLKeepsAWholeAskedRecord(t *testing.T) {
	dir := t.TempDir()
	s := store.NewYAML(dir, nil)
	day := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	_ = s.AppendEvent(store.ReviewEvent{Word: "first", Kind: store.EventLookedUp, Found: true, At: day})
	_ = s.AppendEvent(store.ReviewEvent{
		Kind: store.EventAsked, Question: "what's the difference?", At: day.Add(time.Hour),
	})

	ev, err := s.Events(time.Time{})
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(ev) != 2 {
		t.Fatalf("got %d events, want 2: %+v", len(ev), ev)
	}
	if ev[1].Kind != store.EventAsked || ev[1].Question != "what's the difference?" {
		t.Errorf("asked event = %+v", ev[1])
	}
	// The field ORDER on disk is load-bearing: `at:` must be last, or a cut that
	// drops it would leave a fragment that still looks whole.
	b, err := os.ReadFile(filepath.Join(dir, "events", "2026-08-20.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	rec := string(b)
	if strings.LastIndex(rec, "at:") < strings.LastIndex(rec, "question:") {
		t.Errorf("`at:` is not written last; the torn-record rule depends on it:\n%s", rec)
	}
}

// Truncation cases, each cut at a different point in the record. The earlier
// completeness check admitted two of these: "- word: thi" parses into an event
// with no timestamp, and a cut inside the timestamp leaves a SHORTER DATE THAT
// STILL PARSES, fabricating an event that never happened.
func TestYAMLDropsEveryShapeOfTornRecord(t *testing.T) {
	day := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	for _, tc := range []struct{ name, tail string }{
		{"cut mid-key", "- wor"},
		{"cut mid-value", "- word: thi"},
		{"cut after word", "- word: third\n"}, // terminated but incomplete
		// A truncation never ADDS a terminator, so these carry no trailing newline.
		// An earlier version of this case appended one, which produced an input the
		// writer cannot emit — it always writes a full RFC3339 timestamp, so a
		// short date followed by a newline can only come from a human editing the
		// log, and that is an edit rather than corruption.
		{"cut inside the timestamp", "- word: third\n  kind: looked-up\n  found: true\n  at: 2026-08-20"},
		{"cut just before the terminator", "- word: third\n  kind: looked-up\n  found: true\n  at: 2026-08-20T09:00:00Z"},
		{"unterminated quote", `- word: "third`},
		{"cut mid-key of last field", "- word: third\n  kind: looked-up\n  found: true\n  a"},
		// #16's asked event, cut at each field it added. `question:` is written
		// BEFORE `at:`, so the timestamp is still the last thing a cut removes —
		// which is what the completeness half of the rule leans on.
		{"an asked record cut mid-question", "- kind: asked\n  question: what is the diff"},
		{"an asked record cut before its timestamp", "- kind: asked\n  question: what is the difference?\n"},
		{"an asked record cut inside its timestamp", "- kind: asked\n  question: what is it?\n  at: 2026-08-20"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			var warn bytesBuffer
			s := store.NewYAML(dir, &warn)
			_ = s.AppendEvent(store.ReviewEvent{Word: "first", Kind: store.EventLookedUp, Found: true, At: day})
			_ = s.AppendEvent(store.ReviewEvent{Word: "second", Kind: store.EventLookedUp, Found: true, At: day.Add(time.Hour)})

			f, err := os.OpenFile(filepath.Join(dir, "events", "2026-08-20.yaml"), os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				t.Fatal(err)
			}
			f.WriteString(tc.tail)
			f.Close()

			ev, err := s.Events(time.Time{})
			if err != nil {
				t.Fatalf("a torn record broke the day: %v", err)
			}
			if len(ev) != 2 {
				t.Fatalf("got %d events, want exactly the 2 whole ones: %+v", len(ev), ev)
			}
			if ev[0].Word != "first" || ev[1].Word != "second" {
				t.Errorf("events = %+v", ev)
			}
			if warn.String() == "" {
				t.Error("a dropped record must be reported")
			}
		})
	}
}

// Whole records must survive the recovery path unduplicated. An earlier version
// parsed the whole file first and appended the per-record results to whatever
// the failed parse had already collected, doubling every good record.
func TestYAMLRecoveryDoesNotDuplicate(t *testing.T) {
	dir := t.TempDir()
	s := store.NewYAML(dir, nil)
	day := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	for i, w := range []string{"one", "two", "three"} {
		_ = s.AppendEvent(store.ReviewEvent{Word: w, Kind: store.EventLookedUp, Found: true, At: day.Add(time.Duration(i) * time.Hour)})
	}
	f, _ := os.OpenFile(filepath.Join(dir, "events", "2026-08-20.yaml"), os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString(`- word: "torn`)
	f.Close()

	ev, err := s.Events(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 3 {
		t.Errorf("got %d events, want 3 — the recovery path duplicated: %+v", len(ev), ev)
	}
	seen := map[string]int{}
	for _, e := range ev {
		seen[e.Word]++
	}
	for w, n := range seen {
		if n != 1 {
			t.Errorf("%q appears %d times", w, n)
		}
	}
}

// A log that has been reformatted — by a person, an editor, or a sync tool
// rewriting quotes — must survive. Byte-identical round-tripping would discard
// every record in it, destroying the history it exists to protect.
func TestYAMLAcceptsAReformattedLog(t *testing.T) {
	dir := t.TempDir()
	s := store.NewYAML(dir, nil)
	day := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	_ = s.AppendEvent(store.ReviewEvent{Word: "first", Kind: store.EventLookedUp, Found: true, At: day})

	// Rewrite it the way a formatter might: quoted scalars, different spacing.
	path := filepath.Join(dir, "events", "2026-08-20.yaml")
	reformatted := "- word:   \"first\"\n  kind:   \"looked-up\"\n  found:  yes\n  at:     2026-08-20T09:00:00Z\n"
	if err := os.WriteFile(path, []byte(reformatted), 0o644); err != nil {
		t.Fatal(err)
	}

	ev, err := s.Events(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 1 || ev[0].Word != "first" || !ev[0].Found {
		t.Errorf("a reformatted log was discarded: %+v", ev)
	}
}

// One interrupted write must cost ONE event. If the fragment is not isolated on
// its own line, the next session's append lands on it and both are lost.
func TestYAMLTornFragmentDoesNotSwallowTheNextAppend(t *testing.T) {
	dir := t.TempDir()
	var warn bytesBuffer
	s := store.NewYAML(dir, &warn)
	day := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	_ = s.AppendEvent(store.ReviewEvent{Word: "first", Kind: store.EventLookedUp, Found: true, At: day})

	// A kill mid-append: no terminator.
	path := filepath.Join(dir, "events", "2026-08-20.yaml")
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString("- word: torn")
	f.Close()

	// The NEXT session appends normally.
	_ = store.NewYAML(dir, nil).AppendEvent(store.ReviewEvent{
		Word: "afterwards", Kind: store.EventLookedUp, Found: true, At: day.Add(time.Hour),
	})

	ev, err := s.Events(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 2 {
		t.Fatalf("got %d events, want 2 — the fragment swallowed the next append: %+v", len(ev), ev)
	}
	if ev[1].Word != "afterwards" {
		t.Errorf("events = %+v, want the later append preserved", ev)
	}
}
