package main

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
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

// Below the threshold a job makes no call at all, not even building a client.
func TestRunBackgroundJobHarvestsOnlyPastTheThreshold(t *testing.T) {
	d, fake, st := harvestRig(t, bgThreshold-1)
	d.newLLM = func(llm.Config) llm.Client {
		t.Fatal("a job built a model client below the threshold")
		return nil
	}
	if r := runBackgroundJob(t.Context(), d, nil); r.authored != 0 || r.noModel {
		t.Fatalf("below the threshold: %+v, want nothing done", r)
	}
	if err := st.Upsert(store.Word{Text: deckWord(bgThreshold - 1), LastSeen: harvestClock, Lookups: 1}); err != nil {
		t.Fatal(err)
	}
	d.newLLM = llm.New
	scriptAll(fake, 4)
	if r := runBackgroundJob(t.Context(), d, nil); r.authored == 0 {
		t.Errorf("at the threshold: %+v, want items authored", r)
	}
}

// A model that is not answering is typed as noModel, so the session stops asking.
func TestRunBackgroundJobTypesNoModel(t *testing.T) {
	d, fake, _ := harvestRig(t, bgThreshold)
	fake.Script(markBand, llmtest.Reply{Status: 500, Text: "upstream is having a day"})
	if r := runBackgroundJob(t.Context(), d, nil); !r.noModel {
		t.Errorf("a model answering 500: %+v, want noModel", r)
	}
}

// A job never makes more than bgBudget calls. Today the batch guarantees it on its
// own: ten words cost at most six calls each (band, author, entail, a veto per
// wrong answer), which is the budget itself, so the cap cannot bind and replacing
// it changes nothing this test can see (the M1 mutation run records that). It
// becomes a pin for the budget the day an item costs more calls.
func TestRunBackgroundJobStaysInItsBudget(t *testing.T) {
	d, fake, _ := harvestRig(t, 12)
	scriptAll(fake, 8)
	runBackgroundJob(t.Context(), d, nil)
	if n := len(fake.Requests()); n == 0 || n > bgBudget {
		t.Errorf("one job made %d model call(s); the budget is %d", n, bgBudget)
	}
}

// A job bands and authors the SAME batch, the newest pending words, so a backlog
// drains on both halves. The order that bands the whole backlog first would spend
// a small budget on banding alone, and cannot pass this.
func TestABacklogDrainsOnBothHalves(t *testing.T) {
	d, fake, st := harvestRig(t, 12)
	scriptAll(fake, 8)
	pending, err := pendingWords(st, nil)
	if err != nil || len(pending) != 12 {
		t.Fatalf("pending = %v, %v; want all twelve", pending, err)
	}
	batch := map[string]bool{}
	for _, w := range pending[:bgThreshold] {
		batch[w] = true
	}
	if r := runBackgroundJob(t.Context(), d, nil); r.authored == 0 {
		t.Fatalf("the job authored nothing: %+v", r)
	}
	for _, w := range pending {
		if f, _ := st.WordFacts(w); f.Harvested() != batch[w] {
			t.Errorf("%s banded = %v; only the job's batch, the %d newest, should be", w, f.Harvested(), bgThreshold)
		}
	}
}

// A word whose authoring fails is retried at most once per session: the runner
// keeps it in tried, and the next job skips it.
func TestAWordThatFailsIsRetriedOncePerSession(t *testing.T) {
	d, fake, _ := harvestRig(t, bgThreshold)
	preBand(t, d)
	for _, w := range allDeckWords() {
		fake.Script(authorKey(w), llmtest.Reply{Text: `{"stem":"Senator Murkowski raised the question of ` + w + ` at the Commerce Committee hearing in Anchorage."}`})
	}
	fake.Script(markEntail, llmtest.Reply{Text: `{"entails":true,"glosses":false,"named":true,"reason":"names the committee"}`})
	fake.Script(markVeto, llmtest.Reply{Text: `{"fits":true,"reason":"a near-synonym"}`})
	r := newBgRunner(t.Context(), runBackgroundJob)
	defer r.stop(time.Second)
	r.start(d)
	first := awaitJobResult(t, r)
	if len(first.failed) == 0 {
		t.Fatalf("the first job reported no failures: %+v", first)
	}
	r.received(first)
	before := len(fake.Requests())
	r.start(d)
	second := awaitJobResult(t, r)
	if after := len(fake.Requests()); after != before || second.authored != 0 {
		t.Errorf("the second job made %d more call(s) and authored %d; the failed words should be skipped", after-before, second.authored)
	}
}

// A stopped runner never leaves its goroutine blocked on a send nobody will read,
// even with the result channel already full.
func TestTheRunnerNeverBlocksAfterStop(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	r := newBgRunner(t.Context(), func(ctx context.Context, _ deps, _ map[string]bool) bgJobResult {
		select {
		case <-ctx.Done():
		case <-release:
		}
		return bgJobResult{authored: 1}
	})
	r.results <- bgJobResult{} // a result the loop never read: the next send would wait
	r.start(deps{})
	began := time.Now()
	r.stop(time.Second)
	if took := time.Since(began); took >= time.Second {
		t.Errorf("stop waited the full %v: the job did not end on cancel", took)
	}
	select {
	case <-r.done:
	case <-time.After(time.Second):
		t.Fatal("the job's goroutine is still running after stop: its send blocked")
	}
}

// awaitJobResult is the loop's receive, with a deadline so a lost result fails
// the test instead of hanging it.
func awaitJobResult(t *testing.T, r *bgRunner) bgJobResult {
	t.Helper()
	select {
	case res := <-r.results:
		return res
	case <-time.After(30 * time.Second):
		t.Fatal("no result from the background job")
		return bgJobResult{}
	}
}
