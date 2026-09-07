package store

import (
	"sort"
	"strings"
	"time"
)

// WordFacts is what is cached about a word FOREVER: its band and its domain.
//
// One artifact rather than two caches, because they are one judgement about the
// word and they expire together (never). Assigned once and re-read on every
// later run — which is what makes run-to-run model fuzziness acceptable: a
// word's band is a fact about the word, not about the run that asked.
//
// Stored per LANGUAGE, like the deck and unlike the event log. The dividing line
// is DERIVATION (see yaml.go): `red`, `once`, `actual` and `sensible` are real
// words in both English and Spanish with different bands and unrelated
// meanings, and because these are cached forever and replaced rather than
// merged, a collision would be permanent.
type WordFacts struct {
	Band   Band      `yaml:"band"`
	Domain Domain    `yaml:"domain"`
	At     time.Time `yaml:"at"`
}

// Harvested reports whether these facts were ever assigned.
//
// The timestamp is the signal, so absence needs no second return value — unlike
// NewsItems, which times items it does not timestamp itself. A word that has
// never been harvested and a facts file too damaged to parse read the same way
// here ON PURPOSE: both are worth exactly one re-ask, and a half-trusted record
// is worth less than none.
func (f WordFacts) Harvested() bool { return !f.At.IsZero() }

// Form is which kind of question an item was authored for.
//
// It exists so ONE store holds every form's material rather than a second store
// appearing beside this one when #13's free-sentence form arrives. A word may
// hold several items of different forms, and #12 picks among the ones it can
// render.
type Form string

const (
	// FormCloze is #12's blanked sentence: the stem hides the answer.
	FormCloze Form = "cloze"
	// FormSentence is #13's free written sentence, graded by the model. The stem
	// is a prompt rather than a sentence with a hole in it.
	FormSentence Form = "sentence"
)

// forms is the closed set, and ParseForm is the only way in.
var forms = []Form{FormCloze, FormSentence}

// ParseForm reads a form, refusing anything outside the set. An empty form is
// not an error here — Item's zero value is a legitimate intermediate state — but
// an unrecognised one is, and it degrades to empty rather than being stored.
func ParseForm(s string) (Form, bool) {
	for _, f := range forms {
		if string(f) == s {
			return f, true
		}
	}
	return "", false
}

// Item is one finished practice item, authored offline and stored complete.
//
// COMPLETE is the point. A review sitting must stay instant, free and offline,
// so everything a question needs — the stem, the answer, and the distractors
// already selected and already vetoed — is decided here, ahead of time. Nothing
// downstream reaches for a model to render one.
//
// The item is also the ONLY evidence about itself. The Spec gave up provenance
// deliberately: a model-authored stem has no source URL to inspect when the pool
// goes bad, so when material reads wrong there is nothing to check but the text
// in this struct. That raises the bar on the judges that write it.
type Item struct {
	Word        string    `yaml:"word"`
	Form        Form      `yaml:"form"`
	Stem        string    `yaml:"stem"`
	Answer      string    `yaml:"answer"`
	Distractors []string  `yaml:"distractors,omitempty"`
	At          time.Time `yaml:"at"`
}

// sanitiseFacts and sanitiseItem neutralise and canonicalise on the WRITE, once,
// for BOTH implementations.
//
// Here rather than in each store because the guarantee is the INTERFACE's, not
// YAML's. Store.WordFacts promises a damaged record reads as unharvested, and
// before this existed that promise was a YAML implementation detail: Mem handed
// back whatever it was given, so `SetWordFacts(w, {Band: "B2+"})` read back
// harvested from the fake and unharvested from the real store. A fake that can
// hold a state the real one cannot is the exact gap storetest exists to close
// (#16 M2, BR-45), and this time the divergence was in the direction that
// matters: the fake was the permissive one.
//
// ONE PASS OVER THE STRUCT, which is the placement sanitiseModel argues for at
// cmd/define/usermodel.go:213 and the reason it exists — two earlier rounds
// neutralised the fields a finding happened to list and missed the ones it did
// not. Adding a field to Item now makes this the one place to add a line, and
// every RENDER site is automatically safe.
//
// The parses are a narrower guarantee than neutralisation and NOT a substitute:
// Band and Domain are closed sets, while Stem, Answer and Distractors are free
// model text that later reaches a terminal.
func sanitiseFacts(f WordFacts) WordFacts {
	// Canonicalised, not merely checked. ParseBand forgives case and space as
	// transcription noise, so storing the raw answer would keep "c1" and "C1"
	// as two spellings of one fact on disk — the second source everything else
	// here is written to avoid.
	if b, ok := ParseBand(string(f.Band)); ok {
		f.Band = b
	} else {
		// Refused at the boundary: an off-scale band must not reach Rank, which
		// answers -1 and sorts below A1. Zeroing At is what makes the record read
		// as unharvested, so the word is re-asked rather than silently mispitched.
		return WordFacts{}
	}
	f.Domain, _ = ParseDomain(string(f.Domain))
	return f
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

// sanitiseItems is the pass every WRITE and every READ of the items surface
// goes through — see the read-side rule on Items in store.go.
//
// It COPIES first, which sanitiseItem relies on: sanitiseItem writes through
// i.Distractors' backing array, so handing it a caller's slice would mutate what
// the caller still holds. The copy is here rather than there so there is one
// place to look.
func sanitiseItems(in []Item) []Item {
	out := copyItems(in)
	for i := range out {
		out[i] = sanitiseItem(out[i])
	}
	// PRUNED HERE, at the write, beside the other invariants of this surface. A
	// cap enforced by callers is a cap every future caller has to remember; a cap
	// enforced by the store is one the store guarantees, which is what "growth is
	// bounded" has to mean for it to be checkable.
	return prune(out, ItemCap)
}

// sanitiseItem neutralises one item. It writes through the Distractors backing
// array, so callers hand it a COPY — sanitiseItems is the only caller and makes
// one.
func sanitiseItem(i Item) Item {
	i.Word = Key(i.Word)
	// Form refuses like Band and Domain do. It was the one persisted vocabulary
	// in this store that did not, and #13 adds a third value — an unrecognised
	// form reaching a renderer that switches on it is a question nothing knows
	// how to draw. Empty rather than a guess, so the item is visibly unusable.
	if _, ok := ParseForm(string(i.Form)); !ok {
		i.Form = ""
	}
	i.Stem = oneLine(i.Stem)
	i.Answer = oneLine(i.Answer)
	for n := range i.Distractors {
		i.Distractors[n] = oneLine(i.Distractors[n])
	}
	return i
}

// oneLine collapses model text to a single line.
//
// Item.Stem, Answer and Distractors are the first free-text model fields this
// store persists, and #40's board renders them one per line into a grid. A
// distractor carrying a newline forges a row there the same way a band carrying
// one forged a "define: ..." diagnostic in #17 — the class is untrusted text
// reaching ANY structured output, and the store is the one place every consumer
// of these fields shares.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// ItemCap bounds how many items one word may hold.
//
// Growth is per WORD rather than per deck, because that is the axis that grows
// without limit: a word is re-authored whenever someone deletes its items, and
// #13 adds a second Form that wants its own. Four is two forms with a spare
// each — enough that #12 can pick among them, few enough that a deck of
// thousands stays a directory a person can read.
const ItemCap = 4

// prune bounds a word's items and is DETERMINISTIC.
//
// Its truncation branch is UNREACHABLE from production today: SetItems' only
// non-test caller writes exactly one item, for words that have none. It is a
// guard for #13, which adds a second Form wanting its own items — stated here
// rather than left for a reader to discover that the cap has never fired.
//
// DETERMINISTIC: the same input prunes to the same output, every time.
//
// Newest first by At, ties broken by Stem — the tie-break is what makes it
// deterministic rather than merely usually-stable, because two items authored in
// one run share a timestamp exactly. Without it the survivors would depend on
// map iteration order somewhere upstream, and "pruning twice gives the same
// result" would pass by luck on small inputs.
//
// Newest rather than best: nothing here can rank quality, and pretending to
// would be the self-oracle problem again. Recency is at least a fact.
func prune(items []Item, max int) []Item {
	if max <= 0 || len(items) <= max {
		out := append([]Item(nil), items...)
		sortItems(out)
		return out
	}
	out := append([]Item(nil), items...)
	sortItems(out)
	return out[:max]
}

func sortItems(items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].At.Equal(items[j].At) {
			return items[i].At.After(items[j].At)
		}
		if items[i].Stem != items[j].Stem {
			return items[i].Stem < items[j].Stem
		}
		return items[i].Form < items[j].Form
	})
}
