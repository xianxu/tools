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
// APPEND ONLY, at the TAIL. wordsDir, eventsDir, usageDir, factsDir and
// itemsDir index this slice POSITIONALLY, so inserting a name anywhere but the
// end silently repoints existing directories at each other — a data migration
// disguised as a one-line edit.
var RuntimeDirs = []string{"words", "events", "usage", "facts", "items", "audio"}

// The names define writes into the working directory, each with exactly ONE
// producing function. Guards, migrations and tests DERIVE from these; nothing
// restates them.
//
// That rule is not stylistic. A hand-typed copy of a name is a second source for
// it, and the second source is where the guard stops guarding: with the atomic
// shadow's prefix written out in both the writer and its test, changing the
// writer left every test green while a partial learner model became `git add
// -A`-able in the working-directory root.
const (
	// userModelLegacy is the PRE-#23 flat name. It survives only so
	// MigrateToLanguages can find a directory written before the model became
	// per-language; nothing writes it.
	userModelLegacy = "user-model.md"
	userModelPrefix = "user-model."
	userModelExt    = ".md"
	// tmpPattern is the atomic-write shadow. It reaches .gitignore because for
	// the two root-level runtime files there is no runtime DIRECTORY covering it.
	tmpPattern = ".tmp-*"
)

// UserModelName is the one producer of a learner model's filename.
//
// Exported because a CONSUMER needs to name the file: --reflect tells the
// learner what it wrote, and the README tells them to hand-edit that file's
// ## Corrections section. It printed a literal for exactly one commit, and in
// that commit it printed the legacy flat name while writing the per-language
// one, so corrections would have landed in a file UserModel() does not read.
func UserModelName(l Lang) string { return userModelPrefix + string(l) + userModelExt }

// newTempFile is the one producer of an atomic-write shadow, so a test can
// observe the real name rather than restate the pattern.
func newTempFile(dir string) (*os.File, error) { return os.CreateTemp(dir, tmpPattern) }

// RuntimeFiles names every FILE define writes into the working directory, as
// gitignore-style PATTERNS.
//
// The sibling of RuntimeDirs, and it exists because that list covers directories
// only. The learner model is a runtime file — --reflect writes it into the
// current directory, carrying inferred claims about the learner — and it reached
// none of the three places RuntimeDirs was built to reach: `git check-ignore`
// matched nothing. Nothing leaked, but #23's language setting would have been
// the second instance, which is why this is a list and not two more lines in
// .gitignore.
//
// Patterns rather than names, because two entries are FAMILIES: the learner
// model is per-language, and the atomic-write shadow is one file per write. Both
// pattern entries are BUILT BY their producers, so a change to either scheme
// moves the pattern too and TestGitignoreCoversRuntimeFiles fails loudly instead
// of the guard silently ceasing to cover anything.
//
// `??` — exactly a two-letter Lang, which ParseLang guarantees — rather than
// `*`, which would also shadow the tracked golden fixture under testdata/ and
// quietly re-break the reserved-basename rule below. TestRuntimeFilePatterns
// CoverWhatWeWrite asserts that it does not.
//
// A name here is RESERVED. .gitignore hides these un-anchored (go test runs in
// the package directory), so a tracked file matching one is silently un-addable
// after any git rm — repo_guard_test.go's index guard is what says so out loud.
//
// lang.txt, not lang: an un-anchored `lang` would also hide any DIRECTORY of
// that name anywhere in the tree, and a basename guard structurally cannot see
// that. The extension costs nothing and closes the hole.
var RuntimeFiles = []string{
	userModelLegacy,
	UserModelName("??"), // the per-language family, built by its own producer
	langFileName,
	tmpPattern,
}

// wordsDir and userModelFile are per-language; eventsDir and usageDir are NOT.
//
// The dividing line is DERIVATION, not storage. The learner model is read off a
// language's deck, so one shared file meant a Spanish --reflect replaced the
// English model. An event is a different kind of thing — a fact about a moment,
// not a summary of a deck — so splitting the log would make "how much did I
// study today" a join, reached by migrating an append-only artifact. The
// argument for leaving events/ flat does not transfer to anything derived.
func (y *YAML) wordsDir() string  { return filepath.Join(y.dir, RuntimeDirs[0], string(y.lang)) }
func (y *YAML) eventsDir() string { return filepath.Join(y.dir, RuntimeDirs[1]) }

// userModelFile is the learner model, beside words/ and events/. Markdown rather
// than YAML because a person edits it: #17 regenerates the inferred sections and
// never touches the human-owned ## Corrections.
//
// PER-LANGUAGE, because it is DERIVED from the language-scoped deck. With one
// shared file, `--reflect` in Spanish overwrote the English learner model and
// every English answer was then pitched at "A2 — Spanish beginner". That is the
// same class as #23's other language-derived state, one member further out: it
// is derived from the language AND persisted, so it must be scoped like the deck
// rather than left flat like events/. The events/ argument does not transfer —
// an event is a fact about a moment, while this is a summary of one deck.
//
// Not derived from a RuntimeFiles entry, because those are patterns and a
// pattern cannot name a file. TestRuntimeFilePatternsCoverWhatWeWrite is what
// keeps the two in step instead — a stronger check than derivation, since it
// asserts the actual output rather than a shared string.
func (y *YAML) userModelFile() string {
	return filepath.Join(y.dir, UserModelName(y.lang))
}

// usageDir holds the news cache, one file per word, beside words/ and events/.
func (y *YAML) usageDir() string { return filepath.Join(y.dir, RuntimeDirs[2]) }

// factsDir and itemsDir are PER-LANGUAGE, mirroring wordsDir.
//
// The dividing line stated above is DERIVATION, not storage, and both are
// derived from the language-scoped deck: a band and a domain are judgements
// about a word IN A LANGUAGE, and an authored item is a sentence in one. `red`,
// `once`, `actual` and `sensible` are real words in English and in Spanish with
// different bands and unrelated meanings. Flat would collide them — and because
// facts are cached forever and both are REPLACED rather than merged, the
// collision would be permanent and unwinding it a migration. #23 paid for that
// lesson once with the learner model; this is the same class, two members
// further out.
//
// The events/ argument does not transfer, for the reason it never does: an event
// is a fact about a moment, while these are derived from one deck.
func (y *YAML) factsDir() string { return filepath.Join(y.dir, RuntimeDirs[3], string(y.lang)) }
func (y *YAML) itemsDir() string { return filepath.Join(y.dir, RuntimeDirs[4], string(y.lang)) }

// audioDir holds cached recordings, FLAT — not per language.
//
// The first draft scoped it like factsDir and itemsDir, and the boundary review
// found the bug that makes: a recording fetched while the session was English
// lands under audio/en, and `/lang es` then makes it unreachable — `--forget`
// clears audio/es, reports success, and the file survives. That is #10's BR-45
// exactly, one directory further on.
//
// FLAT IS ALSO THE HONEST SHAPE, not merely the safe one. A language shelf would
// be a SECOND statement of which voice a recording is for, and the first one —
// AudioKey's digest over the candidate list — already carries the voice and the
// locale. Two statements of one fact is the drift this repo keeps paying for. The
// same reasoning usage/ records: per-word, flat, and forgetting is therefore
// cross-language, which for a cache costs a refetch rather than lost work.
func (y *YAML) audioDir() string { return filepath.Join(y.dir, RuntimeDirs[5]) }

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
// out when the learner model arrived: it is markdown a person edits rather than a
// serialised Word, and the alternative was a second temp-file-then-rename dance
// that could drift from this one (ARCH-DRY).
func writeBytesAtomic(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := newTempFile(filepath.Dir(path))
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

// itemsFile is what items/<lang>/<key>.yaml holds: a word's authored items under
// one key, so the file says what it is when a person opens it.
type itemsFile struct {
	Items []Item `yaml:"items"`
}

// WordFacts reads a word's cached band and domain.
//
// A missing file means never harvested, and so does an UNREADABLE one. That is
// the same degradation NewsItems chooses and for a stronger reason: reading a
// damaged record as absent costs one model call, while surfacing half of it puts
// an unvalidated band into the comparison every distractor rule depends on. The
// warning is how a directory quietly going bad still becomes visible.
func (y *YAML) WordFacts(key string) (WordFacts, error) {
	k := Key(key)
	if k == "" {
		return WordFacts{}, nil
	}
	name, err := wordFileName(Slug(k))
	if err != nil {
		return WordFacts{}, err
	}
	b, err := os.ReadFile(filepath.Join(y.factsDir(), name))
	if os.IsNotExist(err) {
		return WordFacts{}, nil // never harvested
	}
	if err != nil {
		return WordFacts{}, err
	}
	var f WordFacts
	if err := yaml.Unmarshal(b, &f); err != nil {
		y.warnf("skipping unreadable %s: %v", name, err)
		return WordFacts{}, nil
	}
	// Re-parsed on the way OUT, not trusted because it is on disk. A file can be
	// hand-edited, half-written or produced by an older build, and a band that
	// does not survive ParseBand must not reach Rank — which answers -1 and would
	// sort the word below A1 rather than refusing it.
	band, ok := ParseBand(string(f.Band))
	if !ok {
		if f.Band != "" {
			y.warnf("skipping %s: %q is not a CEFR band", name, f.Band)
		}
		return WordFacts{}, nil
	}
	f.Band = band
	// A domain that no longer parses degrades to general rather than voiding the
	// record: unlike a band, general is a usable answer, and the table is
	// explicitly allowed to be incomplete.
	f.Domain, _ = ParseDomain(string(f.Domain))
	return f, nil
}

func (y *YAML) SetWordFacts(key string, f WordFacts) error {
	k := Key(key)
	if k == "" {
		return nil
	}
	name, err := wordFileName(Slug(k))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(y.factsDir(), 0o755); err != nil {
		return err
	}
	b, err := yaml.Marshal(sanitiseFacts(f))
	if err != nil {
		return err
	}
	// Atomic, like a word file: this record is REPLACED wholesale and a torn one
	// would read as a word that is banded but not really.
	return writeBytesAtomic(filepath.Join(y.factsDir(), name), b)
}

func (y *YAML) Items(key string) ([]Item, error) {
	k := Key(key)
	if k == "" {
		return nil, nil
	}
	name, err := wordFileName(Slug(k))
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(y.itemsDir(), name))
	if os.IsNotExist(err) {
		return nil, nil // nothing authored yet
	}
	if err != nil {
		return nil, err
	}
	var f itemsFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		// One corrupt file must not make the word unusable: reading as
		// never-authored costs a re-author, which is what an absent file costs.
		y.warnf("skipping unreadable %s: %v", name, err)
		return nil, nil
	}
	// Re-neutralised on the way OUT, the same rule WordFacts states forty lines
	// up: a record read back out of a runtime directory is UNTRUSTED INPUT, no
	// matter that this build wrote it. The write-side pass covers what this
	// build wrote; it cannot cover a file a person edited — and the README
	// documents these as inspectable, so hand-editing is an invited workflow.
	// A distractor carrying a newline would otherwise forge a row on #40's grid,
	// arriving by the one path oneLine was not applied to.
	//
	// Mem cannot reproduce a hand-edited file, so storetest structurally cannot
	// hold this one — yaml_test.go does.
	return sanitiseItems(f.Items), nil
}

func (y *YAML) SetItems(key string, items []Item) error {
	k := Key(key)
	if k == "" {
		return nil
	}
	name, err := wordFileName(Slug(k))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(y.itemsDir(), 0o755); err != nil {
		return err
	}
	b, err := yaml.Marshal(itemsFile{Items: sanitiseItems(items)})
	if err != nil {
		return err
	}
	return writeBytesAtomic(filepath.Join(y.itemsDir(), name), b)
}

// perWordDirs is every directory holding one file per deck word: everything a
// word OWNS, derived from it and regenerable by looking it up again.
//
// events/ is deliberately absent, which is the rule Forget's comment has always
// stated — the deck is a working set, the log is history, and rewriting the past
// would corrupt every statistic derived from it.
//
// Derived from RuntimeDirs rather than hand-listed, and
// TestPerWordDirsCoverEveryRuntimeDir fails when a new runtime directory is
// added without being classified. #10 added TWO (facts/, items/) and Forget
// reached neither.
func (y *YAML) perWordDirs() []perWordDir {
	return []perWordDir{
		{path: y.wordsDir(), scoped: true},
		// usage/ is FLAT, and that makes forgetting cross-language: `--forget red`
		// in a Spanish directory removes the news cache the English deck filled.
		// The cost is a refetch rather than lost work, which is why it is
		// recorded here rather than migrated — but it is recorded, because the
		// classification is what the guard checks.
		{path: y.usageDir(), scoped: false},
		{path: y.factsDir(), scoped: true},
		{path: y.itemsDir(), scoped: true},
		// audio/ is per-word, FLAT, and MANY — the first directory to need all
		// three answers. Flat because the recording's identity is the candidate
		// list, not a shelf (see audioDir); many because a word owns a whole
		// SUBDIRECTORY here, one file per voice inside it, so Forget removes a
		// tree rather than a file.
		{path: y.audioDir(), scoped: false, many: true},
	}
}

// perWordDir is a directory Forget clears, on THREE axes.
//
// Each axis was added by a bug the previous set could not see, which is the
// argument for the shape as much as the shape itself:
//
//  1. per-word or history — whether Forget touches it at all. events/ is
//     history; rewriting the past corrupts every statistic derived from it.
//  2. language-scoped or flat — whose copy it touches. usage/ is per-word and
//     FLAT, so forgetting in one language reaches another's cache; a guard that
//     asked only the first question could not see that.
//  3. one file or many — HOW it is removed. audio/ gives a word a whole
//     subdirectory, one file per voice, so an exact-name removal would report
//     success and leave every recording on disk (#10 BR-45, one directory over).
type perWordDir struct {
	path   string
	scoped bool
	// many says a word owns a whole SUBDIRECTORY here rather than one file, so
	// Forget removes a tree instead of a name.
	//
	// A third question, added by #46 for the same reason the second was added by
	// #10: the classification is what the guard checks, and a directory answering
	// only "per-word?" and "language-scoped?" would have had audio removed by the
	// exact name `<slug>.yaml`, which no audio file is ever called. Forget would
	// have reported success and left every recording on disk.
	many bool
}

// Forget removes everything a word owns. Events are untouched: the deck is a
// working set, the log is history.
//
// EVERYTHING IT OWNS, not just the deck entry. Removing only words/ left the
// word's cached band, domain and authored items on disk — so looking it up again
// re-added it to the deck while --harvest, seeing facts already harvested and
// items already present, SKIPPED it. "Forget this word, its material is bad" was
// the one thing forgetting could not do.
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
	// REPORTS on the deck entry, removes from all of them. Whether the word was
	// in the deck is the question the caller asked; a stale facts file for a word
	// with no deck entry is debris, and removing it is not "found something".
	var removed bool
	for i, dir := range y.perWordDirs() {
		if dir.many {
			if err := removeWordTree(dir.path, Slug(k)); err != nil {
				return false, err
			}
			continue
		}
		err := os.Remove(filepath.Join(dir.path, name))
		switch {
		case err == nil:
			if i == 0 {
				removed = true
			}
		case os.IsNotExist(err):
			// Normal: most words have no news cache and no authored items.
		default:
			return false, err
		}
	}
	return removed, nil
}

// removeWordTree deletes the directory a word owns in a `many` directory.
//
// AN EXACT NAME, not a prefix — and the history is the argument. The first
// design filed recordings as flat `<slug>--<digest>` files and removed them by
// globbing `<slug>--`, on the claim that a slug cannot contain "--". It can:
// `re-` slugs to `re--ddf427`, so forgetting `re` took a different word's
// recordings. The fix is not a rarer separator, because any separator has to be
// reasoned about against Slug's alphabet and that reasoning is what was wrong. A
// directory name admits no such ambiguity.
//
// A missing directory is not an error, exactly as a missing file is not: most
// words have no cached recording.
func removeWordTree(dir, slug string) error {
	if slug == "" {
		return nil
	}
	return os.RemoveAll(filepath.Join(dir, slug))
}

// readCapped reads a file, refusing anything past max.
//
// The network path caps at maxAudioBytes because "anything far larger is not a
// pronunciation"; a file on disk deserves the same ceiling and did not have one.
// The directory is documented as inspectable and hand-editable, so its contents
// are untrusted input like any other (ARCH-SECURE).
func readCapped(path string, max int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, max))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// audioRecordFile is the YAML beside the bytes.
type audioRecordFile struct {
	Record AudioRecord `yaml:"record"`
}

// Audio reads a cached recording and its record.
//
// DEGRADES, NEVER FAILS. Every way this can go wrong — a missing file, a corrupt
// record, an unreadable blob — reads as "nothing cached", because the only cost
// of that answer is one refetch and the cost of the alternative is a learner who
// cannot hear a word. A cache that can break playback is worse than no cache,
// which is the property this method exists to guarantee rather than to document.
func (y *YAML) Audio(k AudioKey) ([]byte, AudioRecord, error) {
	if !k.ok() {
		return nil, AudioRecord{}, nil
	}
	b, err := os.ReadFile(filepath.Join(y.audioDir(), k.dir(), k.stem()+".yaml"))
	if err != nil {
		return nil, AudioRecord{}, nil
	}
	var f audioRecordFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		y.warnf("skipping unreadable audio record %s/%s: %v", k.dir(), k.stem(), err)
		return nil, AudioRecord{}, nil
	}
	if f.Record.Missing {
		return nil, f.Record, nil
	}
	data, err := readCapped(filepath.Join(y.audioDir(), k.dir(), k.stem()+audioBlobExt), maxAudioBlobBytes)
	if err != nil {
		// A record naming a blob that is gone is not a hit. Reporting the record
		// alone would serve an EMPTY recording, which plays as silence and reads
		// as "this word has no audio" — a lie the caller cannot detect.
		return nil, AudioRecord{}, nil
	}
	return data, f.Record, nil
}

// maxAudioBlobBytes caps a cached recording read back off disk, mirroring the
// ceiling fetch.go puts on the network path — "anything far larger is not a
// pronunciation". The two are the same number stated where each is enforced;
// the store cannot import main.
const maxAudioBlobBytes = 4 << 20

// audioBlobExt is what the bytes are called. The CDN serves mp3 and the
// extension is for a human browsing the directory, which the README documents as
// an invited workflow — nothing reads it back by extension.
const audioBlobExt = ".mp3"

// SetAudio stores a recording, or the verdict that there is none.
//
// THE BLOB IS WRITTEN FIRST, and the order is the durability argument. A record
// naming a blob that does not exist would serve an empty recording; a blob with
// no record is invisible and simply re-fetched. Writing the bytes first makes
// the crash window cost a wasted fetch rather than silence.
//
// Both go through writeBytesAtomic, so neither can be observed half-written —
// the `.tmp-*` shadow RuntimeFiles already covers.
func (y *YAML) SetAudio(k AudioKey, data []byte, rec AudioRecord) error {
	if !k.ok() {
		return nil
	}
	dir := filepath.Join(y.audioDir(), k.dir())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	blob := filepath.Join(dir, k.stem()+audioBlobExt)
	if rec.Missing {
		// A VERDICT REPLACES A RECORDING. Skipping the blob write would leave a
		// stale .mp3 beside a record saying there is none — and Audio reads the
		// record first, so the bytes would be unreachable debris that Forget
		// still has to carry.
		if err := os.Remove(blob); err != nil && !os.IsNotExist(err) {
			return err
		}
	} else if err := writeBytesAtomic(blob, data); err != nil {
		return err
	}
	b, err := yaml.Marshal(audioRecordFile{Record: rec})
	if err != nil {
		return err
	}
	return writeBytesAtomic(filepath.Join(dir, k.stem()+".yaml"), b)
}
