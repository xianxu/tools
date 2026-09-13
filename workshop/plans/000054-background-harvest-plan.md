# Background practice preparation Implementation Plan (#54)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** In the interactive session, practice material stays current without a command. A background job harvests what is new once at start and after every 10 lookups, the learner model is written at 12 words and refreshed when the deck's lookups double, and nothing a user does waits on either.

**Architecture:** A pure state machine, `stepBackground`, decides when a job runs. The editor loop holds its state and feeds it three events: the session started, a lookup succeeded, a job finished. A job runs on one goroutine with a copy of the session's deps, takes the newest ten words that still need work, bands and authors that batch through the same harvest core `--harvest` uses (refactored to take a batch and return a typed outcome), reflects through the `--reflect` core when a model is due, and hands one result back on a buffered channel the loop selects on, so every screen write happens on the loop, between prompts. Every dictionary call in the process goes through one lock, because the job looks words up while the loop does.

**Tech Stack:** Go; `cmd/define` package tests; the `llmtest` fake and a blocking stub `llm.Client`; `store.YAML` on a temp dir.

**Branch.** Planned on the #53 branch. Code branches after PR #38 (#53) merges, from main, because M1's docs edit the README rewrite that PR carries.

**Decisions for the operator to confirm** (the defaults this plan assumes):

1. Harvest when at least 10 deck words still need work (no band yet, or no practice item): the operator's number, checked once at session start and after every 10 successful lookups. A job takes the newest 10 of them.
2. At most 60 model calls per job, about 10 words at roughly 6 calls each (band, author, entail, one veto per wrong answer). A bigger backlog drains over several jobs.
3. The learner model is written once the deck has 12 looked-up words (reflect's own floor) and none exists, and refreshed when the deck's lookups have doubled since it was written.
4. Off switch: `DEFINE_NO_BACKGROUND=1`.
5. No cross-process lock (see ARCH-ORDER below).
6. Notices: "N new practice questions ready for /play", "learner model updated", and once per session "practice questions are not being prepared: no model answered (see define --llm-check)".
7. A word whose sentence the model cannot write is retried at most once per session, which costs up to one job's calls per session. A persistent backoff (record the failed attempt and wait 30 days, the way the store already believes a missing recording for thirty days) would stop even that, but it needs a store change; this plan leaves it for a follow-up unless you want it now.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `bgState` | `cmd/define/background.go` | new |
| `bgEvent` | `cmd/define/background.go` | new |
| `bgEffect` | `cmd/define/background.go` | new |
| `bgJobResult` | `cmd/define/background.go` | new |
| `stepBackground` | `cmd/define/background.go` | new |
| `pendingWords` | `cmd/define/background.go` | new |
| `modelLookups` | `cmd/define/background.go` | new |
| `reflectDue` | `cmd/define/background.go` | new |
| `harvestOutcome` | `cmd/define/harvest.go` | new |
| `reflectOutcome` | `cmd/define/reflect.go` | new |

- **`bgState`** — everything the loop carries between events: a phase (`bgIdle`, `bgRunning`, `bgOff`) and `since`, the successful lookups since the last job started. Two fields with a written transition table (below) instead of a set of flags (ARCH-ORDER).
- **`bgEvent`** — one of `bgSessionStart`, `bgLookedUp`, `bgJobDone`; the last carries a `bgJobResult`.
- **`bgEffect`** — what the loop must do after a step: start a job, or print a notice.
- **`bgJobResult`** — what one job did: `authored` (new practice items), `failed` (words whose authoring kept nothing), `reflected` (M2), `noModel`.
- **`stepBackground(s bgState, ev bgEvent) (bgState, []bgEffect)`** — the transition function. Pure; the loop applies its effects.
- **`pendingWords(st store.Store, skip map[string]bool) ([]string, error)`** — the deck words `--harvest` would still do work for, exactly the Spec's definition: no band yet, or no practice item. Newest first (`Deck()` is ordered by last seen), minus `skip`, the words whose authoring already failed this session. The store is the only count: words looked up from the command line, harvested by hand, or forgotten are counted the same way.
- **`modelLookups(md string) (int, bool)`** — the lookup count a learner model was written from, read off its frontmatter `window:` line (`usermodel.go` writes `# N lookups`). Frontmatter only; a hand-edited or truncated file reads as "unknown".
- **`reflectDue(md string, lookups, words int) bool`** — no model and at least `minDeckForReflection` words, or a readable model whose recorded lookups have been doubled (`bgRefreshFactor`). Unknown means not due: a model someone edited by hand is left alone.
- **`harvestOutcome`** — one harvest pass as data: `banded`, `refused`, `skipped`, `authored`, `failed` (the words authoring ran for and kept nothing), `stopped` (the error that stopped it, if any) and `code` (the CLI's exit code, unchanged).
- **`reflectOutcome`** — one reflect pass as data: `written`, `stopped`, `code`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `lockedDictionary` | `cmd/define/dict.go` | new | the cgo DictionaryServices lookup |
| `harvestDeck` | `cmd/define/harvest.go` | new | the model, the store, the dictionary |
| `runHarvest` | `cmd/define/harvest.go` | modified | the `--harvest` flag |
| `runAuthoring` | `cmd/define/harvest.go` | modified | the model, the store |
| `reflectDeck` | `cmd/define/reflect.go` | new | the model, the store |
| `runReflect` | `cmd/define/reflect.go` | modified | the `--reflect` flag |
| `runBackgroundJob` | `cmd/define/background.go` | new | model config, the store, the two cores |
| `bgRunner` | `cmd/define/background.go` | new | one goroutine and its result channel |
| `runEditor` | `cmd/define/replraw.go` | modified | the editor loop |

- **`lockedDictionary`** — a `Dictionary` whose `Lookup` holds `dictionaryMu`, one package-level mutex, so two instances (the loop's, and a job's copy after `/lang`) still serialize. DictionaryServices is reached through cgo (`dict_darwin.go`) with no lock and no documented thread-safety. Every production `Dictionary` is wrapped where it is built.
- **`harvestDeck(ctx, d deps, client llm.Client, deck []store.Word, bud *budget, limit int, batch []string, out, errOut io.Writer) harvestOutcome`** — the banding loop and the call to `runAuthoring`, moved out of `runHarvest`, recording what `runHarvest` used to only print. `batch` limits both halves to those words; nil, which is what the CLI passes, means the whole deck, as today. Authoring still draws wrong answers from every banded word, so a batch costs no distractor quality. Banding a batch and then authoring the same batch is what lets a backlog drain on both halves: today's order, band everything and then author, spends a small budget on banding alone. `runHarvest` keeps its signature: guards, config, client, the agreement mode, then `harvestDeck`, returning `o.code`. Every line `--harvest` prints today, it still prints (the harvest tests pin them).
- **`runAuthoring`** takes the batch and returns `(authored int, failed []string, stopped error, code int)` instead of an exit code alone.
- **`reflectDeck` / `runReflect`** — the same split for `--reflect` (M2).
- **`runBackgroundJob(ctx, d deps, skip map[string]bool) bgJobResult`** — resolve the model config, reflect if due (M2), list the pending words, and if there are at least `bgThreshold`, band and author the newest `bgThreshold` of them within `bgBudget` calls. Its prose goes to `io.Discard`; only its result reaches the screen. A stop whose error is `llm.ErrUnavailable` or `llm.ErrRequest` (`internal/llm/errors.go`, the two that repeat on every call) sets `noModel`. `llm.Resolve` alone cannot say whether a model exists (it succeeds on the default local proxy with nothing running), so the first call decides.
- **`bgRunner`** — starts one job with a copy of `deps`, sends its result on `results` (capacity 1) inside a `select` that also watches `ctx.Done()`, and `stop(wait)` cancels and waits up to `wait` for the job to return. It keeps the session's `tried` set, the words whose authoring failed this session: each result's `failed` joins it, and it is the next job's `skip`, so a word that fails is retried at most once per session.
- **`runEditor`** — builds the runner only when the session can use it (below), feeds `stepBackground`, and selects on `results` beside `con.resizes`. A nil `results` channel never fires, so a session without background work runs exactly as today.

The model sits behind the existing seam (`deps.newLLM`, `deps.getenv`). Tests use the `llmtest` fake, scripted through the harvest tests' own helpers (`harvestRig`, `scriptAll`), plus a stub `llm.Client` whose `Complete` blocks on a channel, because the fake cannot hold a non-streamed reply. The store is `store.YAML` on a temp dir, and the dictionary is `fakeDictionary` (`dict_fake_test.go`). The live conformance checks for the tasks the job calls already exist (`harvest_conformance_test.go`, `reflect_conformance_test.go`); the job adds no new call shape (ARCH-MOCK).

### The state machine (ARCH-ORDER)

| state | event | next | effects |
|---|---|---|---|
| idle | session start | running, since 0 | start a job |
| idle | looked up | since + 1; at `bgThreshold`: running, since 0 | start a job at the threshold |
| running | looked up | since + 1 | none |
| running | job done, `noModel` | off | one notice |
| running | job done | idle; if since ≥ `bgThreshold`: running, since 0 | a notice per thing it did; start a job if due |
| off | any | off | none |
| idle | job done | idle | none (unreachable: a job starts only from idle) |

Events the loop cannot block, and what governs each:

- **Exit mid-job.** `runEditor` defers `bgRunner.stop(2s)`: cancel the context, then wait at most two seconds. The in-flight model call returns, `harvestDeck` stops, and the send selects on `ctx.Done()`, so it never blocks. Every store write is atomic (`writeBytesAtomic` renames a temp file in the same directory), so a stop leaves each file old or new, never half; the wait keeps a killed write from leaving a stray temp file.
- **A result during `/play`.** The sitting borrows the keys and resizes, so the loop is not selecting. The result waits in the channel (capacity 1, at most one job) and prints when the loop resumes. The sitting itself never reads the channel, so `TestASittingNeverWaitsOnHarvesting` keeps holding.
- **`/lang` mid-job.** The job holds the deps copy it started with, so it finishes the language it started in; the next check counts the new language's store.
- **A second `define` on the same deck.** No lock. `SetItems` replaces a word's items file, so two processes authoring one word leave one item (last rename wins). The cost is wasted calls, bounded by each process's budget.
- **A word forgotten mid-job.** A sitting's `d` key, or `define --forget` in another process, removes a word with its facts and items while the job may be authoring it, and the job's later writes can re-create the facts and items. Governed by ignoring it: nothing reads facts or items for a word outside the deck, and if the word is looked up again, its band and sentence are still true of it. The cost is that the word is not re-harvested, which is already the outcome when `define --forget` races `--harvest` today.
- **Which context.** The runner's is a child of the session context `runEditor` receives. An unscoped `interrupter.Fire` cancels the session, which is quit; a scoped Ctrl-C during a streamed answer cancels only that question and leaves the job running.
- **The model disappears mid-job.** A stop typed `ErrUnavailable` or `ErrRequest` turns the session's background work off with one notice; any other stop (a malformed answer, a cancel) leaves it idle, and the next check retries.

Nondeterminism enters through the scheduler and IO completion. Tests fix the order with the blocking stub client (release a channel), never with sleeps, and the pure table covers every transition.

### Operating envelope (ARCH-CONSTRAINTS)

| constraint | budget | basis | when exceeded |
|---|---|---|---|
| lookup latency | unchanged; the job never runs on the loop; the lock adds at most one lookup's wait | lookups are local CoreServices calls, milliseconds | — |
| model calls | ≤ 60 per job (`bgBudget`), one job at a time, plus 1 per reflect | operator choice, decision 2 | the job stops at the budget; the next check continues |
| disk per check | `Deck` (1 + N reads) plus N `WordFacts` and N `Items` reads | `store/yaml.go` | background only; N = 5,000 is about 15,000 small reads |
| reflect input | `Events` reads every day file | `reflect.go` | background only |
| concurrency | one job goroutine; results channel capacity 1 | the machine starts a job only from idle | — |
| shutdown | cancel, then wait ≤ 2 s | atomic writes make any stop safe | after 2 s the process exits anyway |
| authoring retries | a word whose authoring fails is retried at most once per session, about 6 calls each | the session's `tried` set | a persistent backoff is decision 7 |

**ARCH-SECURE.** The job reads files a user may edit by hand: the learner model (parsed by `modelLookups`, where anything unreadable means "not due") and facts and items (already sanitised by the store on read). It touches no credential beyond the existing model config.

**ARCH-DRY.** One harvest core and one reflect core serve the flags and the job. One count, from the store. One lock, at the one unsafe resource.

**ARCH-PURPOSE.** Every Done-when bullet has a task: the ten-word trigger and the end-to-end cloze (1.5), the store count (1.3), not waiting (1.5), quitting (1.4, 1.5), the deck question (1.5), the learner model (2.3), notices on screen and never stderr (1.5), budget and off switch (1.4, 1.5), two processes (argued above), and docs (1.6, 2.4).

### Plan guards at every commit

This file is read by seven guards in `cmd/define/repo_guard_test.go` (`grep -n 'currentTruthFiles(t\|"workshop", "plans"'`), and each task ends with the whole package green:

- `TestPlanTablesNameEntitiesThatExist` — the `modified` rows exist today; the `new` rows are promises while any box is unticked.
- `TestPlanTableStatusMatchesTheChangeWindow` — a `modified` row whose file the branch touches (merge-base to HEAD) must have its declaration touched too. The suite runs after every commit, so in practice the first commit that touches a row's file touches its symbol. Task 1.2 touches both `runHarvest` and `runAuthoring`, Task 1.5 is the first commit to touch `replraw.go` and touches `runEditor`, and Task 2.1 touches `runReflect`.
- `TestPlanNamedTestsExist`, `TestPlanCitesTestsThatExist` — only existing tests appear in backticks here.
- `TestNoArtifactNamesARetiredSymbol`, `TestARemovedDeclarationIsSweptOrRetired`, `TestNoArtifactDescribesARetiredDrawnElement` — nothing is renamed or removed; moving code into `harvestDeck` and `reflectDeck` keeps every old name declared.

- **Prose has no guard.** The table guards resolve only table rows, so every backticked path and symbol in this file's prose is resolved by a grep pass before the gate. Round 2 found two that did not resolve: a Log line and a `config.go` comment that does not exist.

After each commit the check is the whole package, not a filter (lessons, #53).

---

## M1 — background harvest

### Task 1.1: one lock for every dictionary call

**Files:** Modify `cmd/define/dict.go` and every place a production `Dictionary` is built (the one `deps` starts with, and the one `newDict` builds for another language). Test `cmd/define/dict_test.go` (new).

- [x] **Step 1: Write the failing tests.**
  ```go
  type overlapDict struct{ in, max atomic.Int32 }

  func (d *overlapDict) Lookup(string) (string, error) {
  	n := d.in.Add(1)
  	for m := d.max.Load(); n > m && !d.max.CompareAndSwap(m, n); m = d.max.Load() {
  	}
  	time.Sleep(time.Millisecond)
  	d.in.Add(-1)
  	return "", nil
  }

  // Two INSTANCES share the lock: the loop's dictionary and a job's copy after
  // /lang are different values over the same DictionaryServices.
  func TestLockedDictionarySerializesAcrossInstances(t *testing.T) {
  	inner := &overlapDict{}
  	a, b := lockedDictionary{inner: inner}, lockedDictionary{inner: inner}
  	var wg sync.WaitGroup
  	for i := range 8 {
  		wg.Add(1)
  		go func() {
  			defer wg.Done()
  			if i%2 == 0 {
  				a.Lookup("x")
  			} else {
  				b.Lookup("x")
  			}
  		}()
  	}
  	wg.Wait()
  	if got := inner.max.Load(); got != 1 {
  		t.Errorf("%d lookups overlapped; every dictionary call must hold one lock", got)
  	}
  }
  ```
  plus TestProductionDictionariesAreLocked: the dictionary production `deps` starts with, and the one `newDict` returns for another language, are both `lockedDictionary`.
- [x] **Step 2: Run.** → FAIL (`lockedDictionary` undefined).
- [x] **Step 3: Implement** `dictionaryMu`, `lockedDictionary`, and the wrapping at each production construction site.
- [x] **Step 4: Run** the new tests, then the whole package → PASS.
- [x] **Step 5: Commit.** `#54: every dictionary call holds one lock`

### Task 1.2: the harvest core returns what it did

**Files:** Modify `cmd/define/harvest.go`. Test `cmd/define/harvest_test.go`.

- [ ] **Step 1: Write the failing tests.** With `harvestRig(t, 3)` and `scriptAll`:
  - TestHarvestDeckReportsWhatItAuthored: `harvestDeck` returns `banded == 3`, `authored` equal to the number of words `Items` now holds, and a nil `stopped`.
  - TestHarvestDeckTypesAMissingModel: with the fake scripted to answer 500 (the pattern the harvest tests already use), `errors.Is(o.stopped, llm.ErrUnavailable)` and `o.code == 1`.
  - TestHarvestDeckWorksOnlyOnItsBatch: `harvestRig(t, 6)` with four words pre-banded (`preBand`) and a batch of the other two → only those two are banded and authored, and their wrong answers may come from the four.
  - TestHarvestDeckReportsWhatItCouldNotAuthor: a veto that rejects every candidate → the word is in `o.failed` and has no items.
- [ ] **Step 2: Run.** → FAIL (`harvestDeck` undefined).
- [ ] **Step 3: Implement** `harvestOutcome`, `harvestDeck` (the banding loop moved out of `runHarvest`, filtered to the batch when there is one) and the new `runAuthoring` signature. `runHarvest` becomes guards, config, client, the agreement mode, `harvestDeck`, `return o.code`.
- [ ] **Step 4: Run** every existing harvest test unchanged (their output assertions are the guard that the CLI did not move), the new tests, then the whole package → PASS.
- [ ] **Step 5: Commit.** `#54: the harvest core returns what it did`

### Task 1.3: the state machine and the count

**Files:** Create `cmd/define/background.go`, `cmd/define/background_test.go`.

- [ ] **Step 1: Write the failing tests.**
  ```go
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
  	if err := st.SetItems("alpha", []store.Item{{Word: "alpha", Form: store.FormCloze, Stem: "the alpha test", Answer: "alpha"}}); err != nil {
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
  ```
  The implementer checks the `store` calls against `store.go` and adjusts the calls, never the assertions.
- [ ] **Step 2: Run.** → FAIL (undefined).
- [ ] **Step 3: Implement** `bgPhase` and its three constants, `bgState`, `bgEvent` and its three kinds, `bgEffect`, `bgJobResult`, `stepBackground` per the table, `pendingWords`, and the constants `bgThreshold = 10`, `bgBudget = 60`. Notice texts come from one function, `bgNoticeFor(r bgJobResult) []string`.
- [ ] **Step 4: Run** → PASS; whole package → PASS.
- [ ] **Step 5: Commit.** `#54: when a background job runs, as a table`

### Task 1.4: the job and its runner

**Files:** Modify `cmd/define/background.go`, `cmd/define/background_test.go`.

- [ ] **Step 1: Write the failing tests.**
  - TestRunBackgroundJobHarvestsOnlyPastTheThreshold: a `harvestRig` store with `bgThreshold - 1` pending words and a `newLLM` that fails the test if called → a zero result; one more word and the scripted fake → `authored > 0`.
  - TestRunBackgroundJobTypesNoModel: the fake's URL pointing at a closed server → `noModel`.
  - TestRunBackgroundJobStaysInItsBudget: a backlog of 30 words → the fake saw at most `bgBudget` requests.
  - TestABacklogDrainsOnBothHalves: a never-harvested deck of 25 words and the scripted fake; run jobs until one finds too little pending → every word is banded, every word the fake lets it author has items, and no job banded a word it then left unauthored for want of budget.
  - TestAWordThatFailsIsRetriedOncePerSession: a veto that rejects every candidate for one word → the first job reports it in `failed`, the runner adds it to `tried`, and the next job's pending list leaves it out.
  - TestTheRunnerNeverBlocksAfterStop: a job blocked on a stub client; `stop(time.Second)` returns within the second, and the goroutine's send does not block with nobody reading.
- [ ] **Step 2: Run.** → FAIL.
- [ ] **Step 3: Implement** `runBackgroundJob` (with its `skip` set) and `bgRunner` (with the session's `tried` set), and the `noBackgroundEnv` constant (`"DEFINE_NO_BACKGROUND"`).
- [ ] **Step 4: Run** → PASS; whole package → PASS.
- [ ] **Step 5: Commit.** `#54: a background job, bounded and cancellable`

### Task 1.5: the session runs it

**Files:** Modify `cmd/define/replraw.go`. Test `cmd/define/background_loop_test.go` (new).

The runner exists only when all hold: this is the raw editor (`replRaw`), the deck is open (`d.deck != nil`), the deck question is decided and allowed (`d.deckPermission.saving()` returns `true, true`; `repl` settles it before `replRaw` starts), and `d.getenv(noBackgroundEnv)` is empty. At start the loop applies `bgSessionStart`. After `submitLine` returns with `out.code == 0 && out.ask == ""` it applies `bgLookedUp`. The new case is `case res := <-results:` (nil when there is no runner): `view.Draw("", nil)`, apply `bgJobDone`, print each notice as `define: <notice>`, `draw()`. A job's `deps` is a copy of the loop's at the moment it starts.

- [ ] **Step 1: Write the failing tests**, on the editor rig with `fakeDictionary` holding a dozen words, a `harvestRig`-style store and the scripted fake:
  - TestTheSessionPreparesPracticeAfterTenNewWords: look up ten words → the screen shows the ready notice, `Items` holds items for them, and `todaysQuestions` builds cloze questions for them. The Done-when, end to end.
  - TestALookupNeverWaitsForTheBackgroundJob: a stub client whose `Complete` blocks until released; after the tenth word, look up two more and assert both render while the job is blocked; then release.
  - TestQuittingCancelsTheJobAndKeepsWhatItWrote: the stub answers the first band and then blocks; cancel the session → `runEditor` returns within the stop's wait, and that word's facts are on disk.
  - TestNoJobWhereTheDeckWasNotAgreedTo: an undecided, then a declined, permission and a `newLLM` that fails the test if called; ten lookups → no job.
  - TestTheOffSwitchStopsIt: `DEFINE_NO_BACKGROUND=1` → the same.
  - TestNoModelIsOneNoticeThenQuiet: a closed server → one notice after the first job; ten more lookups → no job, no second notice.
  - `TestNothingIsWrittenWhileAPromptIsShown` stays green with a result delivered mid-session.
- [ ] **Step 2: Run.** → FAIL.
- [ ] **Step 3: Implement** the wiring above.
- [ ] **Step 4: Run** → PASS; whole package → PASS.
- [ ] **Step 5: Commit.** `#54: the session prepares practice in the background`

### Task 1.6: M1 docs

**Files:** Modify `cmd/define/README.md` (practice material), `atlas/define.md` (the harvest section and a new "Background preparation" section: the table, the envelope, the lock, the notices), and every sentence that calls harvesting batch-only: the comment above `runHarvest` in `harvest.go`, the atlas's "Batch, and the only path here that may block" paragraph, and the README's "It is batch, on demand" sentence. M2 takes the learner-model ones.

- [ ] **Step 1:** The README says the session harvests in the background every 10 new words and once at start, at most 60 model calls a time, and says when questions are ready; `DEFINE_NO_BACKGROUND=1` turns it off, and `--harvest` still runs it by hand. The rule is restated as what still holds: nothing you do waits on a model.
- [ ] **Step 2:** The atlas and code comments say the same, and `TestASittingNeverWaitsOnHarvesting`'s premise is restated (a sitting never harvests; the session's job runs beside it).
- [ ] **Step 3:** Whole package → PASS (the README guards).
- [ ] **Step 4: Commit.** `#54: M1 docs: the session prepares practice itself`

### Task 1.7: verify M1, then close it

- [ ] **Step 1: Mutation-verify** in a throwaway worktree, each mutation asserted to match once and restored:

  | mutation | guard that must go red |
  |---|---|
  | `lockedDictionary.Lookup` skips the lock | TestLockedDictionarySerializesAcrossInstances |
  | a production site returns the bare dictionary | TestProductionDictionariesAreLocked |
  | `stepBackground` starts a job while running | TestStepBackgroundTransitions |
  | `pendingWords` drops the no-item half | TestPendingWordsFollowsTheSpec |
  | the job ignores `bgThreshold` | TestRunBackgroundJobHarvestsOnlyPastTheThreshold |
  | the job bands the whole backlog before authoring | TestABacklogDrainsOnBothHalves |
  | the runner drops a result's `failed` | TestAWordThatFailsIsRetriedOncePerSession |
  | the job's budget is `harvestLimit` | TestRunBackgroundJobStaysInItsBudget |
  | the send ignores `ctx.Done()` | TestTheRunnerNeverBlocksAfterStop |
  | the permission check is dropped | TestNoJobWhereTheDeckWasNotAgreedTo |
  | the off switch is ignored | TestTheOffSwitchStopsIt |
  | `noModel` is never set | TestNoModelIsOneNoticeThenQuiet |
  | the loop prints the notice from the goroutine | `TestNothingIsWrittenWhileAPromptIsShown` |

  Plus an unmutated control run.
- [ ] **Step 2:** gofmt, `go vet ./...`, `go test ./...` → PASS, run after the last commit.
- [ ] **Step 3:** The operator smoke-tests a session: ten new words, the notice, `/play` shows cloze questions.
- [ ] **Step 4:** `sdlc milestone-close --issue 54 --milestone M1 --verified '<evidence>'`.

## M2 — background reflect

### Task 2.1: the reflect core returns what it did

**Files:** Modify `cmd/define/reflect.go`. Test `cmd/define/reflect_test.go`.

- [ ] **Step 1: Write the failing tests.** TestReflectDeckReportsWhatItWrote (12 deck words and a scripted model → `written`, a nil `stopped`) and TestReflectDeckTypesAMissingModel (a closed server → `errors.Is(o.stopped, llm.ErrUnavailable)`).
- [ ] **Step 2: Run.** → FAIL.
- [ ] **Step 3: Implement** `reflectOutcome` and `reflectDeck` (the body of `runReflect` from the floor check to the write, unchanged). `runReflect` becomes the flag's guards, `reflectDeck`, `return o.code`; every existing reflect test passes unchanged.
- [ ] **Step 4: Run** → PASS; whole package → PASS.
- [ ] **Step 5: Commit.** `#54: the reflect core returns what it did`

### Task 2.2: when a learner model is due

**Files:** Modify `cmd/define/background.go`, `cmd/define/background_test.go`.

- [ ] **Step 1: Write the failing tests.**
  ```go
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
  	} {
  		if got := reflectDue(tc.md, tc.lookups, tc.words); got != tc.want {
  			t.Errorf("%s: reflectDue = %v, want %v", tc.name, got, tc.want)
  		}
  	}
  }
  ```
  and FuzzModelLookups, seeded with the table's malformed forms (the shape of `cloze_fuzz_test.go`): it never panics, and when it reports a count, that number appears in the frontmatter's `window:` line.
- [ ] **Step 2: Run.** → FAIL.
- [ ] **Step 3: Implement** `modelLookups`, `reflectDue` and `bgRefreshFactor = 2`.
- [ ] **Step 4: Run** → PASS; whole package → PASS.
- [ ] **Step 5: Commit.** `#54: when a learner model is due`

### Task 2.3: the job reflects before it harvests

**Files:** Modify `cmd/define/background.go`. Test `cmd/define/background_loop_test.go`.

`runBackgroundJob` folds the deck (`foldLookups`) and, when `reflectDue`, calls `reflectDeck` before counting unbanded words, because authoring reads the model (`readLearner`). A reflect stop typed `ErrUnavailable` or `ErrRequest` sets `noModel`. `reflected` produces the "learner model updated" notice.

- [ ] **Step 1: Write the failing tests.** TestTheSessionWritesALearnerModelAtTwelveWords (twelve lookups → `UserModel()` is written before the harvest's authoring requests, and the notice shows) and TestTheSessionRefreshesTheModelWhenLookupsDouble (a model recording 20 lookups and a deck with 40 → rewritten; with 30 → left alone).
- [ ] **Step 2: Run.** → FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** → PASS; whole package → PASS.
- [ ] **Step 5: Commit.** `#54: the session writes the learner model when it is due`

### Task 2.4: M2 docs, verify, close

- [ ] **Step 1:** The README's learner-model section, the atlas's reflect section and the comment above `runReflect` say the session writes the model at 12 words and refreshes it when lookups double; `--reflect` still runs it by hand.
- [ ] **Step 2: Mutation-verify:** `reflectDue` ignores the refresh factor (TestReflectDue); `modelLookups` reads past the frontmatter (TestModelLookupsReadsOnlyTheFrontmatter); the job harvests before it reflects (TestTheSessionWritesALearnerModelAtTwelveWords); plus a control.
- [ ] **Step 3:** gofmt, `go vet ./...`, `go test ./...` → PASS, run after the last commit.
- [ ] **Step 4:** `sdlc close --issue 54 --verified '<evidence>'`.
