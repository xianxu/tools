package main

import (
	"slices"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// Every row of the plan's transition table (#54), so the legal states and moves
// are read off one table rather than inferred from the loop.
func TestStepBackgroundTransitions(t *testing.T) {
	did := bgJobResult{authored: 3}
	nothing := bgJobResult{}
	noModel := bgJobResult{noModel: true}
	for _, tc := range []struct {
		name    string
		from    bgState
		ev      bgEvent
		to      bgState
		runJob  bool
		notices int
	}{
		{"start runs a job", bgState{phase: bgIdle}, bgEvent{kind: bgSessionStart}, bgState{phase: bgRunning}, true, 0},
		{"a lookup counts", bgState{phase: bgIdle, since: 3}, bgEvent{kind: bgLookedUp}, bgState{phase: bgIdle, since: 4}, false, 0},
		{"the threshold runs a job", bgState{phase: bgIdle, since: bgThreshold - 1}, bgEvent{kind: bgLookedUp}, bgState{phase: bgRunning}, true, 0},
		{"lookups count while running", bgState{phase: bgRunning, since: 2}, bgEvent{kind: bgLookedUp}, bgState{phase: bgRunning, since: 3}, false, 0},
		{"a result goes idle and says so", bgState{phase: bgRunning}, bgEvent{kind: bgJobDone, result: did}, bgState{phase: bgIdle}, false, 1},
		{"a result with nothing new is silent", bgState{phase: bgRunning}, bgEvent{kind: bgJobDone, result: nothing}, bgState{phase: bgIdle}, false, 0},
		{"a result after the threshold runs again", bgState{phase: bgRunning, since: bgThreshold}, bgEvent{kind: bgJobDone, result: nothing}, bgState{phase: bgRunning}, true, 0},
		{"no model turns it off, once", bgState{phase: bgRunning, since: bgThreshold}, bgEvent{kind: bgJobDone, result: noModel}, bgState{phase: bgOff, since: bgThreshold}, false, 1},
		{"off ignores a lookup", bgState{phase: bgOff}, bgEvent{kind: bgLookedUp}, bgState{phase: bgOff}, false, 0},
		{"off ignores a start", bgState{phase: bgOff}, bgEvent{kind: bgSessionStart}, bgState{phase: bgOff}, false, 0},
		{"a stray result while idle is ignored", bgState{phase: bgIdle, since: 1}, bgEvent{kind: bgJobDone, result: did}, bgState{phase: bgIdle, since: 1}, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, effects := stepBackground(tc.from, tc.ev)
			var runs, notices int
			for _, e := range effects {
				if e.runJob {
					runs++
				}
				if e.notice != "" {
					notices++
				}
			}
			if got != tc.to || (runs == 1) != tc.runJob || runs > 1 || notices != tc.notices {
				t.Errorf("got %+v, %d job(s), %d notice(s); want %+v, job %v, %d notice(s)", got, runs, notices, tc.to, tc.runJob, tc.notices)
			}
		})
	}
}

// Pending is the Spec's definition: no band yet, or no practice item. Newest
// first, minus the words whose authoring already failed this session.
func TestPendingWordsFollowsTheSpec(t *testing.T) {
	st := store.NewMem()
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	for i, w := range []string{"alpha", "bravo", "charlie", "delta"} {
		if err := st.Upsert(store.Word{Text: w, LastSeen: base.Add(time.Duration(i) * time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	band, _ := store.ParseBand("B2")
	for _, w := range []string{"alpha", "bravo"} {
		if err := st.SetWordFacts(w, store.WordFacts{Band: band, Domain: store.DomainGeneral, At: base}); err != nil {
			t.Fatal(err)
		}
	}
	// alpha is done: banded, with an item. bravo is banded with no item, so it is
	// still pending: --harvest would still author it.
	if err := st.SetItems("alpha", []store.Item{{Word: "alpha", Form: store.FormCloze,
		Stem: "the alpha test", Answer: "alpha", Distractors: []string{"bravo"}, At: base}}); err != nil {
		t.Fatal(err)
	}
	if got, err := pendingWords(st, nil); err != nil || !slices.Equal(got, []string{"delta", "charlie", "bravo"}) {
		t.Fatalf("pendingWords = %v, %v; want [delta charlie bravo], newest first", got, err)
	}
	// A word whose authoring failed this session is skipped; a forgotten word is gone.
	if _, err := st.Forget("delta"); err != nil {
		t.Fatal(err)
	}
	if got, _ := pendingWords(st, map[string]bool{"bravo": true}); !slices.Equal(got, []string{"charlie"}) {
		t.Errorf("with bravo skipped and delta forgotten: %v, want [charlie]", got)
	}
}
