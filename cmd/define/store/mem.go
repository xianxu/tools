package store

import (
	"sort"
	"sync"
	"time"
)

// Mem is the in-memory Store.
//
// Production code, not a test double: it is the reference implementation the
// YAML one is measured against, and an ephemeral deck is a plausible mode later.
type Mem struct {
	mu     sync.Mutex
	words  map[string]Word
	events []ReviewEvent
	// userModel is what YAML reads from the learner model file. A field so the reference
	// implementation can hold the state the real one holds: a getter the fake
	// cannot back makes the conformance row asserting it unfalsifiable.
	userModel string
	news      map[string]newsCache
	facts     map[string]WordFacts
	items     map[string][]Item
}

func NewMem() *Mem {
	return &Mem{
		words: map[string]Word{},
		news:  map[string]newsCache{},
		facts: map[string]WordFacts{},
		items: map[string][]Item{},
	}
}

// newsCache is items plus WHEN, because the timestamp is what distinguishes
// "fetched and found nothing" from "never fetched".
type newsCache struct {
	items []NewsItem
	at    time.Time
}

func (m *Mem) Upsert(w Word) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := Key(w.Text)
	if k == "" {
		return nil
	}
	m.words[k] = merge(m.words[k], w)
	return nil
}

// merge is the one definition of "combine an existing entry with a new sighting",
// shared by both stores so they cannot disagree about it.
func merge(old, w Word) Word {
	out := w
	out.Text = Key(w.Text)
	if !old.FirstSeen.IsZero() && (out.FirstSeen.IsZero() || old.FirstSeen.Before(out.FirstSeen)) {
		out.FirstSeen = old.FirstSeen
	}
	if old.LastSeen.After(out.LastSeen) {
		out.LastSeen = old.LastSeen
	}
	out.Lookups = old.Lookups + max(1, w.Lookups)
	return out
}

func (m *Mem) Deck() ([]Word, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Word, 0, len(m.words))
	for _, w := range m.words {
		out = append(out, w)
	}
	sortDeck(out)
	return out, nil
}

func sortDeck(ws []Word) {
	sort.SliceStable(ws, func(i, j int) bool {
		if ws[i].LastSeen.Equal(ws[j].LastSeen) {
			return ws[i].Text < ws[j].Text
		}
		return ws[i].LastSeen.After(ws[j].LastSeen)
	})
}

func (m *Mem) AppendEvent(e ReviewEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
	return nil
}

func (m *Mem) Events(since time.Time) ([]ReviewEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []ReviewEvent
	for _, e := range m.events {
		if !e.At.Before(since) {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out, nil
}

func (m *Mem) SetUserModel(text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userModel = text
	return nil
}

func (m *Mem) UserModel() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.userModel, nil
}

func (m *Mem) NewsItems(key string) ([]NewsItem, time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.news[Key(key)]
	if !ok {
		return nil, time.Time{}, nil
	}
	return append([]NewsItem(nil), c.items...), c.at, nil
}

func (m *Mem) SetNewsItems(key string, items []NewsItem, at time.Time) error {
	k := Key(key)
	if k == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.news == nil {
		m.news = map[string]newsCache{}
	}
	// Copied, not aliased: the caller keeps its slice and a later append on
	// their side must not mutate what this store believes it holds.
	m.news[k] = newsCache{items: append([]NewsItem(nil), items...), at: at}
	return nil
}

func (m *Mem) WordFacts(key string) (WordFacts, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.facts[Key(key)], nil
}

func (m *Mem) SetWordFacts(key string, f WordFacts) error {
	k := Key(key)
	if k == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.facts == nil {
		m.facts = map[string]WordFacts{}
	}
	m.facts[k] = f
	return nil
}

func (m *Mem) Items(key string) ([]Item, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	its := m.items[Key(key)]
	if len(its) == 0 {
		return nil, nil
	}
	return copyItems(its), nil
}

func (m *Mem) SetItems(key string, items []Item) error {
	k := Key(key)
	if k == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.items == nil {
		m.items = map[string][]Item{}
	}
	// Copied, not aliased, as SetNewsItems is: the caller keeps its slice and a
	// later append on their side must not mutate what this store believes it
	// holds.
	m.items[k] = copyItems(items)
	return nil
}

// copyItems deep-copies far enough to matter: Item's only reference field is
// Distractors, and sharing that slice is the aliasing SetNewsItems' comment
// warns about, one level down.
func copyItems(in []Item) []Item {
	out := make([]Item, len(in))
	copy(out, in)
	for i := range out {
		out[i].Distractors = append([]string(nil), in[i].Distractors...)
	}
	return out
}

func (m *Mem) Forget(key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := Key(key)
	if _, ok := m.words[k]; !ok {
		return false, nil
	}
	delete(m.words, k)
	return true, nil
}
