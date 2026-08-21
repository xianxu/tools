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
}

func NewMem() *Mem { return &Mem{words: map[string]Word{}} }

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
