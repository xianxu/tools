package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
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
	deckFail := bgJobResult{deckErr: deckIO(errors.New("permission denied"))}
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
		{"a deck that fails turns it off, once", bgState{phase: bgRunning}, bgEvent{kind: bgJobDone, result: deckFail}, bgState{phase: bgOff}, false, 1},
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
	if got, err := pendingWords(st, readDeck(t, st), nil); err != nil || !slices.Equal(got, []string{"delta", "charlie", "bravo"}) {
		t.Fatalf("pendingWords = %v, %v; want [delta charlie bravo], newest first", got, err)
	}
	// A word whose authoring failed this session is skipped; a forgotten word is gone.
	if _, err := st.Forget("delta"); err != nil {
		t.Fatal(err)
	}
	if got, _ := pendingWords(st, readDeck(t, st), map[string]bool{"bravo": true}); !slices.Equal(got, []string{"charlie"}) {
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
	if r := runBackgroundJob(t.Context(), d, bgMemory{}); r.authored != 0 || r.noModel {
		t.Fatalf("below the threshold: %+v, want nothing done", r)
	}
	if err := st.Upsert(store.Word{Text: deckWord(bgThreshold - 1), LastSeen: harvestClock, Lookups: 1}); err != nil {
		t.Fatal(err)
	}
	d.newLLM = llm.New
	scriptAll(fake, 4)
	if r := runBackgroundJob(t.Context(), d, bgMemory{}); r.authored == 0 {
		t.Errorf("at the threshold: %+v, want items authored", r)
	}
}

// A model that is not answering is typed as noModel, so the session stops asking.
func TestRunBackgroundJobTypesNoModel(t *testing.T) {
	d, fake, _ := harvestRig(t, bgThreshold)
	fake.Script(markBand, llmtest.Reply{Status: 500, Text: "upstream is having a day"})
	if r := runBackgroundJob(t.Context(), d, bgMemory{}); !r.noModel {
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
	runBackgroundJob(t.Context(), d, bgMemory{})
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
	pending, err := pendingWords(st, readDeck(t, st), nil)
	if err != nil || len(pending) != 12 {
		t.Fatalf("pending = %v, %v; want all twelve", pending, err)
	}
	batch := map[string]bool{}
	for _, w := range pending[:bgThreshold] {
		batch[w] = true
	}
	if r := runBackgroundJob(t.Context(), d, bgMemory{}); r.authored == 0 {
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
	r := newBgRunner(t.Context(), func(ctx context.Context, _ deps, _ bgMemory) bgJobResult {
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

// readDeck is the deck as a caller reads it before asking what is pending.
func readDeck(t *testing.T, st store.Store) []store.Word {
	t.Helper()
	deck, err := st.Deck()
	if err != nil {
		t.Fatal(err)
	}
	return deck
}

// Every way a job leaves a word unfinished is retried at most once a session, a
// refused band included (#54). A model that answers with no CEFR band leaves the
// word unbanded, so it stays pending, and a runner that did not remember it would
// spend a band call on it at every check.
func TestABandRefusalIsRetriedOncePerSession(t *testing.T) {
	d, fake, _ := harvestRig(t, bgThreshold)
	for range bgThreshold {
		fake.Script(markBand, llmtest.Reply{Text: `{"band":"Z9","domain":"Law"}`})
	}
	r := newBgRunner(t.Context(), runBackgroundJob)
	defer r.stop(time.Second)
	r.start(d)
	first := awaitJobResult(t, r)
	if len(first.failed) != bgThreshold {
		t.Fatalf("the first job reported %d unfinished word(s), want all %d refused: %+v", len(first.failed), bgThreshold, first)
	}
	r.received(first)
	before := len(fake.Requests())
	r.start(d)
	awaitJobResult(t, r)
	if after := len(fake.Requests()); after != before {
		t.Errorf("the second job made %d more call(s); the refused words should be skipped", after-before)
	}
}

// The job never writes to the terminal (#54). The session's stores warn to the
// process stderr and a job reads off the loop, so it reads through a view that
// drops warnings, whether the store is bare or behind the deck gate. The control
// reads the same store plainly and must warn, or the test proves nothing.
func TestTheJobWritesNothingToTheTerminal(t *testing.T) {
	for name, gated := range map[string]bool{"bare": false, "behind the gate": true} {
		t.Run(name, func(t *testing.T) {
			d, fake, _ := harvestRig(t, 0)
			scriptAll(fake, 4)
			dir := t.TempDir()
			warn := &syncBuf{}
			st := store.NewYAML(dir, store.DefaultLang, warn)
			for i := range bgThreshold {
				if err := st.Upsert(store.Word{Text: deckWord(i), FirstSeen: harvestClock.AddDate(0, 0, -20),
					LastSeen: harvestClock.AddDate(0, 0, -i), Lookups: 1}); err != nil {
					t.Fatal(err)
				}
			}
			bad := filepath.Join(dir, store.RuntimeDirs[0], string(store.DefaultLang), "unreadable.yaml")
			if err := os.WriteFile(bad, []byte("text: [unclosed\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := st.Deck(); err != nil || warn.Len() == 0 {
				t.Fatalf("a plain read of the planted file drew no warning (%v); the test would pass vacuously", err)
			}
			warn.TakeAll()
			d.deck = st
			if gated {
				d.deck = newGatedStore(st, store.NewMem(), nil)
			}
			if r := runBackgroundJob(t.Context(), d, bgMemory{}); r.authored == 0 {
				t.Fatalf("the job did no work, so its silence proves nothing: %+v", r)
			}
			if warn.Len() != 0 {
				t.Errorf("the job wrote %q to the store's warning writer, which is the terminal in a session", warn.String())
			}
		})
	}
}

// The learner model's lookup count comes from its frontmatter's window: line and
// nowhere else, and anything a person could have written reads as unknown (#54).
func TestModelLookupsReadsOnlyTheFrontmatter(t *testing.T) {
	for _, tc := range []struct {
		name string
		md   string
		want int
		ok   bool
	}{
		{"written by reflect", "---\ntype: user-model\nwindow: 2026-08-01..2026-09-01          # 40 lookups, 3 questions\n---\n", 40, true},
		{"no window line", "---\ntype: user-model\n---\n", 0, false},
		{"hand-edited", "---\nwindow: whenever\n---\n", 0, false},
		{"no model", "", 0, false},
		{"only in Corrections", "---\ntype: user-model\n---\n\n## Corrections\nwindow: a..b # 99 lookups\n", 0, false},
		{"unterminated frontmatter", "---\nwindow: a..b # 40 lookups\n", 0, false},
	} {
		if got, ok := modelLookups(tc.md); got != tc.want || ok != tc.ok {
			t.Errorf("%s: modelLookups = %d, %v; want %d, %v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

// When the session writes the learner model: at the floor when there is none, and
// again when the deck's lookups have doubled (#54).
func TestReflectDue(t *testing.T) {
	model := func(n int) string {
		return fmt.Sprintf("---\nwindow: a..b          # %d lookups, 0 questions\n---\n", n)
	}
	for _, tc := range []struct {
		name           string
		md             string
		lookups, words int
		want           bool
	}{
		{"below the floor", "", 30, minDeckForReflection - 1, false},
		{"none yet, at the floor", "", 30, minDeckForReflection, true},
		{"fresh", model(40), 60, 30, false},
		{"lookups doubled", model(40), 80, 30, true},
		{"unreadable model is left alone", "---\nwindow: ???\n---\n", 500, 30, false},
		// Doubling nothing is no growth: without this a model written from no
		// lookups would be rewritten, a paid call, at every check.
		{"a model from no lookups waits for one", model(0), 0, 30, false},
		{"a refresh still needs the floor", model(40), 80, minDeckForReflection - 1, false},
	} {
		if got := reflectDue(tc.md, tc.lookups, tc.words); got != tc.want {
			t.Errorf("%s: reflectDue = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// FuzzModelLookups: the learner model is a file a person edits, so reading its
// count must never panic, and a count it reports must be one the frontmatter's
// window: line states.
func FuzzModelLookups(f *testing.F) {
	for _, s := range []string{
		"---\ntype: user-model\nwindow: 2026-08-01..2026-09-01          # 40 lookups, 3 questions\n---\n",
		"---\ntype: user-model\n---\n",
		"---\nwindow: whenever\n---\n",
		"",
		"---\ntype: user-model\n---\n\n## Corrections\nwindow: a..b # 99 lookups\n",
		"---\nwindow: a..b # 40 lookups\n",
		"---\nwindow: a..b # -3 lookups\n---\n",
		"---\nwindow: a..b # 99999999999999999999999 lookups\n---\n",
		"---\r\nwindow: a..b # 7 lookups\r\n---\r\n",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, md string) {
		n, ok := modelLookups(md)
		if !ok {
			if n != 0 {
				t.Errorf("modelLookups(%q) = %d with no count", md, n)
			}
			return
		}
		if n < 0 {
			t.Fatalf("modelLookups(%q) = %d, a negative count", md, n)
		}
		for _, line := range strings.Split(md, "\n")[1:] {
			if strings.TrimSpace(line) == "---" {
				break
			}
			if key, val, _ := strings.Cut(line, ":"); strings.TrimSpace(key) == "window" && strings.Contains(val, strconv.Itoa(n)) {
				return
			}
		}
		t.Errorf("modelLookups(%q) = %d, which no window: line in the frontmatter states", md, n)
	})
}

// A deck the job cannot read is said once and turns the session's background work
// off, as a model that does not answer does (#54): it would fail the same way at
// every check, and a session that stayed silent would never say why no practice
// appeared.
func TestAStoreErrorStopsTheSessionOnce(t *testing.T) {
	d, _, _ := harvestRig(t, bgThreshold)
	d.deck = failingFacts{d.deck}
	r := runBackgroundJob(t.Context(), d, bgMemory{})
	if !errors.Is(r.deckErr, errDeckIO) {
		t.Fatalf("result = %+v, want a deck error", r)
	}
	s, effects := stepBackground(bgState{phase: bgRunning}, bgEvent{kind: bgJobDone, result: r})
	if s.phase != bgOff || len(effects) != 1 || !strings.Contains(effects[0].notice, "could not be read or written") {
		t.Errorf("after a deck error: %+v, %+v; want off and one notice naming the deck", s, effects)
	}
}

// The session refreshes the learner model only when the deck's lookups have
// doubled since the count the model records (#54): a level is stable.
func TestTheSessionRefreshesTheModelWhenLookupsDouble(t *testing.T) {
	const recorded = "---\ntype: user-model\nwindow: 2026-08-01..2026-08-20          # 20 lookups, 0 questions\n---\n"
	for _, tc := range []struct {
		name    string
		lookups int
		want    bool
	}{{"doubled", 40, true}, {"not yet", 30, false}} {
		t.Run(tc.name, func(t *testing.T) {
			d, fake, st, _ := reflectRig(t, minDeckForReflection)
			for i := minDeckForReflection; i < tc.lookups; i++ {
				if err := st.AppendEvent(store.ReviewEvent{Word: deckWord(i % minDeckForReflection),
					Kind: store.EventLookedUp, Found: true, At: reflectClock}); err != nil {
					t.Fatal(err)
				}
			}
			if err := st.SetUserModel(recorded); err != nil {
				t.Fatal(err)
			}
			fake.Script(markReflect, llmtest.Reply{Text: reflectReply})
			r := runBackgroundJob(t.Context(), d, bgMemory{})
			md, _ := st.UserModel()
			if r.reflected != tc.want || (md != recorded) != tc.want {
				t.Fatalf("%d lookups against a model from 20: reflected %v, rewritten %v; want %v",
					tc.lookups, r.reflected, md != recorded, tc.want)
			}
			if tc.want && !strings.Contains(md, fmt.Sprintf("# %d lookups", tc.lookups)) {
				t.Errorf("the rewritten model does not record the %d lookups it was written from:\n%s", tc.lookups, md)
			}
		})
	}
}

// A reflect that writes nothing is not asked for again this session (#54), by the
// same rule as a word a job could not finish: the model stays due at every check
// until one is written, so an answer that cannot be used would cost a call at each.
func TestAFailedReflectIsNotRetriedThisSession(t *testing.T) {
	d, fake, _, _ := reflectRig(t, minDeckForReflection)
	fake.Script(markReflect, llmtest.Reply{Text: "not json"})
	reflects := func() (n int) {
		for _, r := range fake.Requests() {
			if strings.Contains(r.Prompt(), markReflect) {
				n++
			}
		}
		return n
	}
	r := newBgRunner(t.Context(), runBackgroundJob)
	defer r.stop(time.Second)
	r.start(d)
	first := awaitJobResult(t, r)
	if !first.reflectFailed || reflects() == 0 {
		t.Fatalf("the first job: %+v after %d reflect call(s); want a reflect that wrote nothing", first, reflects())
	}
	r.received(first)
	before := reflects()
	r.start(d)
	awaitJobResult(t, r)
	if n := reflects() - before; n != 0 {
		t.Errorf("the second job asked for the learner model %d more time(s); a failed reflect is not retried this session", n)
	}
}

// What a stop means for the session, by its kind (#54): the two model stops that
// repeat on every call, and a deck whose files fail, turn background work off;
// anything else, a malformed answer or a cancel, leaves the next check to try.
func TestStopMeans(t *testing.T) {
	deck := deckIO(errors.New("permission denied"))
	for _, tc := range []struct {
		name    string
		err     error
		noModel bool
		deckErr error
	}{
		{"no stop", nil, false, nil},
		{"not answering", fmt.Errorf("band: %w", llm.ErrUnavailable), true, nil},
		{"a request it refuses", fmt.Errorf("band: %w", llm.ErrRequest), true, nil},
		{"the deck's files", deck, false, deck},
		{"a malformed answer", fmt.Errorf("band: %w", llm.ErrMalformed), false, nil},
		{"a cancel", context.Canceled, false, nil},
	} {
		if noModel, deckErr := stopMeans(tc.err); noModel != tc.noModel || deckErr != tc.deckErr {
			t.Errorf("%s: stopMeans = %v, %v; want %v, %v", tc.name, noModel, deckErr, tc.noModel, tc.deckErr)
		}
	}
}

// A store error inside the harvest reaches the session as one while counting does
// (#54): the harvest marks it where the store returned it, and the job reads the
// mark.
func TestAStoreWriteErrorInTheHarvestStopsTheSession(t *testing.T) {
	d, fake, _ := harvestRig(t, bgThreshold)
	fake.Script(markBand, llmtest.Reply{Text: bandReply})
	d.deck = failingWrites{d.deck}
	if r := runBackgroundJob(t.Context(), d, bgMemory{}); !errors.Is(r.deckErr, errDeckIO) || r.noModel {
		t.Errorf("result = %+v, want a deck error and not a model one", r)
	}
}
