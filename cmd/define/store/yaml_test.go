package store_test

import (
	"bytes"
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
		return store.NewYAML(t.TempDir(), store.DefaultLang, nil)
	})
}

func TestYAMLPersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	s1 := store.NewYAML(dir, store.DefaultLang, nil)
	if err := s1.Upsert(store.Word{Text: "sycophantic", LastSeen: time.Now()}); err != nil {
		t.Fatal(err)
	}
	// A second store on the same directory is what a restart looks like.
	deck, err := store.NewYAML(dir, store.DefaultLang, nil).Deck()
	if err != nil {
		t.Fatal(err)
	}
	if len(deck) != 1 || deck[0].Text != "sycophantic" {
		t.Errorf("reopened deck = %+v", deck)
	}
}

// One corrupt file must not make the whole deck unopenable — that would lose
// every word to a single bad byte.
func TestYAMLSkipsCorruptFileWithWarning(t *testing.T) {
	dir := t.TempDir()
	var warn bytesBuffer
	s := store.NewYAML(dir, store.DefaultLang, &warn)
	_ = s.Upsert(store.Word{Text: "good", LastSeen: time.Now()})

	// words/en/, not words/: after #23 the deck is per-language, and a file left
	// flat in words/ is invisible to Deck() rather than corrupt-looking — which
	// is exactly what makes the migration's "leave a colliding flat file alone"
	// rule non-destructive.
	if err := os.WriteFile(filepath.Join(dir, "words", "en", "broken.yaml"), []byte("\t: [unclosed"), 0o644); err != nil {
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
	s := store.NewYAML(dir, store.DefaultLang, nil)
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
	_ = store.NewYAML(dir, store.DefaultLang, nil).Upsert(store.Word{Text: "alpha", LastSeen: now})
	before, _ := os.ReadDir(filepath.Join(dir, "words", "en"))

	_ = store.NewYAML(dir, store.DefaultLang, nil).Upsert(store.Word{Text: "beta", LastSeen: now})
	after, _ := os.ReadDir(filepath.Join(dir, "words", "en"))

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
	s := store.NewYAML(dir, store.DefaultLang, &warn)
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
	s := store.NewYAML(dir, store.DefaultLang, nil)
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
			s := store.NewYAML(dir, store.DefaultLang, &warn)
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
	s := store.NewYAML(dir, store.DefaultLang, nil)
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
	s := store.NewYAML(dir, store.DefaultLang, nil)
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
	s := store.NewYAML(dir, store.DefaultLang, &warn)
	day := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	_ = s.AppendEvent(store.ReviewEvent{Word: "first", Kind: store.EventLookedUp, Found: true, At: day})

	// A kill mid-append: no terminator.
	path := filepath.Join(dir, "events", "2026-08-20.yaml")
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString("- word: torn")
	f.Close()

	// The NEXT session appends normally.
	_ = store.NewYAML(dir, store.DefaultLang, nil).AppendEvent(store.ReviewEvent{
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

// A corrupt cache file reads as never-fetched and warns, rather than failing the
// session or returning half a record.
//
// This is a WHOLE-FILE record — writeBytesAtomic cannot tear it — so the failure
// to model is corruption from outside, and the discipline for that shape is the
// one Deck() already uses: warn and skip. The terminator-plus-completeness rule
// belongs to the append-only day log, not here; the plan's first draft named the
// wrong one.
func TestCorruptNewsCacheReadsAsNeverFetched(t *testing.T) {
	dir := t.TempDir()
	var warn bytes.Buffer
	y := store.NewYAML(dir, store.DefaultLang, &warn)

	if err := y.SetNewsItems("ephemeral", []store.NewsItem{{Title: "real"}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	// Corrupt it the way a truncated copy or a bad editor would.
	path := filepath.Join(dir, "usage", "ephemeral.yaml")
	if err := os.WriteFile(path, []byte("items:\n  - title: \"unterminated\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, at, err := y.NewsItems("ephemeral")

	if err != nil {
		t.Fatalf("a corrupt cache file failed the read: %v", err)
	}
	if len(items) != 0 || !at.IsZero() {
		t.Errorf("got %d items at %v, want never-fetched", len(items), at)
	}
	if !strings.Contains(warn.String(), "ephemeral.yaml") {
		t.Errorf("no warning naming the file: %q", warn.String())
	}
}

// Words file under their language, and — the assertion that is the point — a
// Spanish deck cannot see an English word.
//
// Isolation, not just tidiness: #5's schedule interleaves the deck by due-date,
// so a mixed directory would not merely allow a mixed sitting, it would produce
// one. The learner asked for the opposite.
func TestWordsAreScopedByLanguage(t *testing.T) {
	dir := t.TempDir()
	en := store.NewYAML(dir, store.DefaultLang, nil)
	es := store.NewYAML(dir, store.Lang("es"), nil)

	if err := en.Upsert(store.Word{Text: "sycophantic"}); err != nil {
		t.Fatal(err)
	}
	if err := es.Upsert(store.Word{Text: "madrugar"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "words", "en", "sycophantic.yaml")); err != nil {
		t.Errorf("English word not filed under words/en: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "words", "es", "madrugar.yaml")); err != nil {
		t.Errorf("Spanish word not filed under words/es: %v", err)
	}

	for _, tc := range []struct {
		name    string
		deck    *store.YAML
		want    string
		notWant string
	}{
		{name: "es", deck: es, want: "madrugar", notWant: "sycophantic"},
		{name: "en", deck: en, want: "sycophantic", notWant: "madrugar"},
	} {
		got, err := tc.deck.Deck()
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if len(got) != 1 || got[0].Text != tc.want {
			t.Errorf("%s deck = %+v, want just %q", tc.name, got, tc.want)
		}
		for _, w := range got {
			if w.Text == tc.notWant {
				t.Errorf("the %s deck can see %q, a word from the other language", tc.name, tc.notWant)
			}
		}
	}
}

// An empty language means the default, so a caller that has not been taught
// about languages yet cannot accidentally create a words// directory.
func TestEmptyLangIsTheDefault(t *testing.T) {
	dir := t.TempDir()
	if err := store.NewYAML(dir, "", nil).Upsert(store.Word{Text: "sycophantic"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "words", string(store.DefaultLang), "sycophantic.yaml")); err != nil {
		t.Errorf("an empty language did not file under words/%s: %v", store.DefaultLang, err)
	}
}

// The event log is NOT scoped, and that is a design commitment rather than an
// omission: a review event names a word and a verdict, and which deck it came
// from is the deck's business. Splitting it would make "how much did I study
// today" a join, and would migrate an append-only artifact to get there.
func TestEventsAreNotScopedByLanguage(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	if err := store.NewYAML(dir, store.Lang("es"), nil).AppendEvent(store.ReviewEvent{
		Word: "madrugar", Kind: store.EventLookedUp, Found: true, At: now,
	}); err != nil {
		t.Fatal(err)
	}
	// Read back through the OTHER language: the log is one log.
	evs, err := store.NewYAML(dir, store.DefaultLang, nil).Events(now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 1 || evs[0].Word != "madrugar" {
		t.Errorf("events = %+v, want the one event visible from either language", evs)
	}
}
